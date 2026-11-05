package tass

// Fresh-task retest (foundation gate F, step 5): after a lesson is owner-approved (LEARNED), TASS
// answers DIFFERENT questions about it, given only what it has learned (the same approved-knowledge
// context a conversation gets), never the lesson text. Core grades the answers with the same grader.
//
// PASSED (graded) ≠ LEARNED (owner approved) ≠ RETESTED (applied on a fresh task) ≠ a skill. A
// retest never changes the lesson's status, approves anything, grants a skill or runs code.

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Arameair/Thought-Processor/core/engine"
	"github.com/Arameair/Thought-Processor/core/lessons"
)

// RetestQuestion is one fresh question and its expected answer ("|" separates accepted answers).
// The expected answer is never sent to the model or shown before grading.
type RetestQuestion struct {
	N      int    `json:"n"`
	Ask    string `json:"ask"`
	Answer string `json:"answer"`
	From   string `json:"from"` // "owner" (typed in Development) | "lesson" (its ## Retest section)
}

// RetestRecord is one retest attempt and its evidence.
type RetestRecord struct {
	JobID     string           `json:"job_id"`
	CycleID   string           `json:"cycle_id,omitempty"` // NOT RECORDED until the Engine reports it
	SHA       string           `json:"sha"`                // lesson revision that was approved
	State     string           `json:"state"`              // QUEUED | DONE | FAILED | INTERRUPTED | REFUSED
	Outcome   string           `json:"outcome,omitempty"`  // RETEST_PASSED | RETEST_FAILED | MODEL_FAILED
	Questions []RetestQuestion `json:"questions"`
	Results   []lessons.Result `json:"results,omitempty"`
	Right     int              `json:"right"`
	Checks    int              `json:"checks"`
	Reply     string           `json:"reply,omitempty"`
	Context   string           `json:"context"` // what TASS was given, described (never the lesson text)
	At        time.Time        `json:"at"`
	GradedAt  *time.Time       `json:"graded_at,omitempty"`
}

const maxRetests = 20

func retestJobID(p, sha string, seq int) string { return "retest_" + hash(p + sha)[:10] + "_" + fmt.Sprint(seq) }

// lessonRetests: the "## Retest N" sections of a lesson (hidden from study, so they stay fresh).
func lessonRetests(raw string) []RetestQuestion {
	var out []RetestQuestion
	for _, s := range lessons.Parse(raw).Retests {
		if s.Check != "" && s.Answer != "" {
			out = append(out, RetestQuestion{N: s.N, Ask: s.Check, Answer: s.Answer, From: "lesson"})
		}
	}
	return out
}

func validRetest(qs []RetestQuestion, studied lessons.Lesson) error {
	if len(qs) < 1 || len(qs) > 5 {
		return errors.New("a retest needs 1 to 5 questions")
	}
	seen := map[int]bool{}
	for _, q := range qs {
		if q.N < 1 || q.N > 99 || seen[q.N] {
			return errors.New("retest questions need distinct numbers 1-99")
		}
		seen[q.N] = true
		if strings.TrimSpace(q.Ask) == "" || strings.TrimSpace(q.Answer) == "" || len(q.Ask) > 500 || len(q.Answer) > 200 {
			return errors.New("each retest question needs a question (≤500 chars) and an expected answer (≤200 chars)")
		}
		for _, s := range studied.Checks() { // a fresh task, not the study quiz again
			if strings.EqualFold(strings.TrimSpace(s.Check), strings.TrimSpace(q.Ask)) {
				return fmt.Errorf("question %d is the lesson's own Check %d; a retest must ask something new", q.N, s.N)
			}
		}
	}
	return nil
}

// RetestLesson queues one fresh-task retest of a LEARNED lesson. qs are the owner's new questions;
// with none, the lesson's own "## Retest" sections are used.
func (c *Core) RetestLesson(p string, qs []RetestQuestion) (string, error) {
	if c.cfg.IdentitySvc == nil || c.cfg.ChatOnly {
		return "", errors.New("retest unavailable in forced safety mode")
	}
	if c.GetSettings().Autonomous {
		return "", errors.New("turn Autonomous OFF for the supervised retest")
	}
	if ps, err := c.engine.PulseStatus(); err != nil || ps.Control != "PAUSED" {
		return "", errors.New("a supervised retest requires the repeating work pulse to be PAUSED")
	}
	for _, action := range []string{"lesson.learn", "pulse.run"} {
		if ok, why := c.allowedNow(action); !ok {
			return "", fmt.Errorf("Identity denied %s: %s", action, why)
		}
	}
	c.manualMu.Lock()
	defer c.manualMu.Unlock()
	c.learnMu.Lock()
	ls := c.learn.Lessons[p]
	if ls == nil {
		c.learnMu.Unlock()
		return "", ErrNoLesson
	}
	if ls.Status != "LEARNED" || ls.LearnedAt == nil {
		st := ls.Status
		c.learnMu.Unlock()
		return "", fmt.Errorf("only an owner-approved (LEARNED) lesson can be retested; this one is %s", st)
	}
	if ls.RetestPending != nil {
		id := ls.RetestPending.JobID
		c.learnMu.Unlock()
		if _, err := c.engine.PulseControl("step"); err != nil {
			return id, fmt.Errorf("retest %s remains queued; pulse step failed: %w", id, err)
		}
		return id, nil
	}
	for _, other := range c.learn.Lessons {
		if other.Status == "QUEUED" || (other != ls && other.RetestPending != nil) {
			c.learnMu.Unlock()
			return "", errors.New("another lesson job is queued; finish it before starting a retest")
		}
	}
	sha, seq := ls.SHA, ls.RetestSeq
	c.learnMu.Unlock()

	raw, err := os.ReadFile(filepath.Join(c.file("lessons"), filepath.FromSlash(p)))
	if err != nil {
		return "", err
	}
	if len(qs) == 0 {
		qs = lessonRetests(string(raw))
		if len(qs) == 0 {
			return "", errors.New("give TASS a new question and its expected answer (the lesson has no ## Retest section)")
		}
	}
	for i := range qs {
		if qs[i].From == "" {
			qs[i].From = "owner"
		}
	}
	if err := validRetest(qs, lessons.Parse(string(raw))); err != nil {
		return "", err
	}
	// Only approved knowledge: exactly what a conversation is given. Never the lesson text.
	known := c.learnedForContext()
	if strings.TrimSpace(known) == "" {
		return "", errors.New("nothing approved to retest from")
	}
	ctx := "WHAT I HAVE LEARNED (owner-approved lessons; the original lesson text is NOT shown):\n" + known
	var b strings.Builder
	b.WriteString("Use what you have learned to answer these new questions. They are not the questions you studied.\n")
	nums := make([]int, 0, len(qs))
	for _, q := range qs {
		fmt.Fprintf(&b, "Check %d: %s\n", q.N, q.Ask)
		nums = append(nums, q.N)
	}
	b.WriteString("You have no executable actions. Answering does not grant a skill or approve anything.")
	name := c.entityName()
	var id string
	for k := 1; k <= max409Probes; k++ {
		id = retestJobID(p, sha, seq+k)
		err = c.engine.SubmitJob(engine.Job{JobID: id, Identity: name, Request: b.String(), Context: ctx, LessonChecks: nums})
		if err == nil {
			seq += k
			break
		}
		if !errors.Is(err, engine.ErrJobExists) {
			return "", err
		}
	}
	if err != nil {
		return "", fmt.Errorf("no free retest job ID after %d tries: %w", max409Probes, err)
	}
	now := c.now().UTC()
	c.learnMu.Lock()
	cur := c.learn.Lessons[p]
	if cur == nil || cur.SHA != sha || cur.Status != "LEARNED" {
		c.learnMu.Unlock()
		return "", errors.New("lesson changed while the retest was being queued")
	}
	cur.RetestSeq = seq
	cur.RetestPending = &RetestRecord{JobID: id, SHA: sha, State: "QUEUED", Questions: qs, Checks: len(qs), At: now,
		Context: fmt.Sprintf("approved-knowledge context only (%d chars from %d LEARNED lessons); lesson text withheld", len(known), strings.Count(known, "\n- ")+1)}
	c.saveLearningLocked()
	c.learnMu.Unlock()
	c.event("sync", fmt.Sprintf("Fresh-task retest of %q queued: %s (%d new questions). Status stays LEARNED.", cur.Title, id, len(qs)))
	if _, err := c.engine.PulseControl("step"); err != nil {
		return id, fmt.Errorf("retest queued as %s but pulse step failed: %w", id, err)
	}
	return id, nil
}

// collectRetests grades any retest whose Engine job has finished. Called from advanceLessons.
func (c *Core) collectRetests() {
	c.learnMu.Lock()
	var pending []*LessonState
	for _, ls := range c.learn.Lessons {
		if ls.RetestPending != nil {
			pending = append(pending, ls)
		}
	}
	c.learnMu.Unlock()
	if len(pending) == 0 {
		return
	}
	ps, err := c.engine.PulseStatus()
	if err != nil {
		return
	}
	for _, ls := range pending {
		c.learnMu.Lock()
		r := ls.RetestPending
		var job *engine.JobStatus
		for i := range ps.Jobs {
			if r != nil && ps.Jobs[i].JobID == r.JobID {
				job = &ps.Jobs[i]
			}
		}
		if r == nil || job == nil || job.State == "QUEUED" || job.State == "RUNNING" {
			c.learnMu.Unlock()
			continue
		}
		now := c.now().UTC()
		done := *r
		done.State, done.CycleID, done.Reply, done.GradedAt = job.State, job.CycleID, job.Result, &now
		if job.State != "DONE" {
			done.Outcome = "MODEL_FAILED"
		} else {
			steps := make([]lessons.Step, 0, len(r.Questions))
			for _, q := range r.Questions {
				steps = append(steps, lessons.Step{N: q.N, Check: q.Ask, Answer: q.Answer})
			}
			done.Results, done.Right, done.Checks = lessons.GradeSteps(steps, job.Result)
			done.Outcome = "RETEST_FAILED"
			if done.Checks > 0 && done.Right == done.Checks {
				done.Outcome = "RETEST_PASSED"
			}
		}
		ls.Retests = append(ls.Retests, done)
		if len(ls.Retests) > maxRetests {
			ls.Retests = append([]RetestRecord(nil), ls.Retests[len(ls.Retests)-maxRetests:]...)
		}
		ls.RetestPending = nil
		title := ls.Title
		c.saveLearningLocked()
		c.learnMu.Unlock()
		c.event("sync", fmt.Sprintf("Lesson %q fresh-task retest %s: %s %d/%d (cycle %s). Status unchanged (LEARNED); no skill granted.",
			title, done.JobID, done.Outcome, done.Right, done.Checks, firstNonEmpty(done.CycleID, "NOT RECORDED")))
	}
}
