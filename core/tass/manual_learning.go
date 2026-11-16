package tass

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Arameair/Thought-Processor/core/engine"
	"github.com/Arameair/Thought-Processor/core/lessons"
)

// studyRequest identifies real checks; Engine alone specifies the ANSWER n
// output contract. Do not instruct the model to produce both REPLY and ANSWER.
func studyRequest(l lessons.Lesson) string {
    var b strings.Builder
    fmt.Fprintf(&b,"Read the Markdown lesson %q, then answer each Check in the lesson.\n",l.Title)
    b.WriteString("Check numbers:")
    for _,check:=range l.Checks() { fmt.Fprintf(&b," %d",check.N) }
    b.WriteString(". You have no executable actions and are not yet approved as having learned this lesson.")
    return b.String()
}

func studyCheckNumbers(l lessons.Lesson) []int {
    checks:=l.Checks()
    nums:=make([]int,0,len(checks))
    for _,step:=range checks { nums=append(nums,step.N) }
    return nums
}

// DiscoverLessons is one explicit owner request to read lessons. It requires
// live Identity permissions, never starts the pulse, changes Autonomous, or
// executes notebook cells. The scheduled sync loop stays disabled when
// Autonomous is off.
func (c *Core) DiscoverLessons() error {
	if c.cfg.IdentitySvc == nil || c.cfg.ChatOnly {
		return errors.New("owner-supervised lesson discovery is unavailable in forced safety mode")
	}
	if c.cfg.Repo == "" || c.cfg.Branch == "" {
		return errors.New("configure the owner's GitHub lesson repository first")
	}
	for _, action := range []string{"github.sync", "lesson.learn"} {
		if ok, why := c.allowedNow(action); !ok {
			return fmt.Errorf("Identity denied %s: %s; %s", action, why, identityFix(why))
		}
	}
	c.manualMu.Lock()
	defer c.manualMu.Unlock()
	if err := c.syncLessons(true); err != nil {
		c.event("problem", "Manual lesson discovery failed: "+err.Error())
		return err
	}
	c.event("sync", "Owner requested lesson discovery from GitHub. Review the Learning card; nothing was executed.")
	return nil
}

// StudyLesson queues exactly one owner-selected, parsed Markdown lesson.
// It is intentionally available while Autonomous is OFF. The Engine receives
// no executable actions; it only answers checks. One explicit pulse step runs
// the queued study job, after which the existing sync loop collects/grading.
// PASSED stays unapproved until the owner clicks Approve.
func (c *Core) StudyLesson(p string) (string, error) {
	if c.cfg.IdentitySvc == nil || c.cfg.ChatOnly {
		return "", errors.New("lesson study unavailable in forced safety mode")
	}
	if c.GetSettings().Autonomous {
		return "", errors.New("turn Autonomous OFF for the supervised Study action")
	}
	pulse,err:=c.engine.PulseStatus()
	if err!=nil || pulse.Control!="PAUSED" {
		return "",errors.New("manual Study requires the repeating work pulse to be PAUSED")
	}
	for _, action := range []string{"lesson.learn", "pulse.run"} {
		if ok, why := c.allowedNow(action); !ok {
			return "", fmt.Errorf("Identity denied %s: %s", action, why)
		}
	}
	c.manualMu.Lock()
	defer c.manualMu.Unlock()

	c.learnMu.Lock()
	var active bool
	for _, item := range c.learn.Lessons {
		if item.Status == "QUEUED" || item.RetestPending != nil {
			active = true
			break
		}
	}
	ls := c.learn.Lessons[p]
	if ls == nil {
		c.learnMu.Unlock()
		return "", ErrNoLesson
	}
	// A second Study click on the same still-queued job is an explicit
	// repeat pulse step, not a second job or duplicate lesson attempt.
	if ls.Status == "QUEUED" {
		id := ls.JobID
		c.learnMu.Unlock()
		if _, err := c.engine.PulseControl("step"); err != nil {
			return id, fmt.Errorf("existing job %s remains queued; pulse step failed: %w", id, err)
		}
		return id, nil
	}
	if active {
		c.learnMu.Unlock()
		return "", errors.New("a different lesson is already queued; finish it before starting another")
	}
	if ls.Status != "NEW" && ls.Status != "RETRY" && ls.Status != "MODEL_FAILED" {
		status := ls.Status
		c.learnMu.Unlock()
		return "", fmt.Errorf("lesson is %s; only NEW, RETRY or MODEL_FAILED lessons may be studied", status)
	}
	// Snapshot only immutable lesson provenance; do not keep an unlocked
	// pointer across model, disk or network calls.
	snapshot := *ls
	c.learnMu.Unlock()
	raw, err := os.ReadFile(filepath.Join(c.file("lessons"), filepath.FromSlash(p)))
	if err != nil {
		return "", err
	}
	l := lessons.Parse(string(raw))
	if !l.Checkable || l.HasCode {
		return "", errors.New("manual study requires a checkable Markdown lesson without code execution")
	}
	c.fileMu.Lock()
	doc, _ := c.read("tass.md")
	c.fileMu.Unlock()
	if strings.TrimSpace(doc) == "" {
		return "", errors.New("TASS document is not synced yet; wait for the next GitHub sync before studying")
	}
	request := studyRequest(l)
	// Keep model input small and unambiguous: the lesson, not the entire entity document.
	ctx := "LESSON (from " + c.cfg.Repo + "/" + p + "):\n" + l.ForStudy()
	if len(ctx) > 60<<10 {
		ctx = ctx[:60<<10]
	}
	name := c.entityName()
	id, seq, attached, found, err := c.submitLessonJob(snapshot.Path, snapshot.SHA, snapshot.JobSeq, func(id string) engine.Job {
		return engine.Job{JobID: id, Identity: name, Request: request, Context: ctx, Actions: nil, LessonChecks: studyCheckNumbers(l)}
	})
	c.learnMu.Lock()
	cur := c.learn.Lessons[p]
	if cur != nil && cur.SHA == snapshot.SHA {
		for _, r := range found {
			cur.addHistory(r)
		}
		if seq > cur.JobSeq {
			cur.JobSeq = seq
		}
		if err == nil {
			cur.Status, cur.JobID, cur.Attempts = "QUEUED", id, snapshot.Attempts+1
			cur.Results, cur.Right, cur.Reply, cur.Remember, cur.CycleID = nil, 0, "", "", ""
		}
		c.saveLearningLocked()
	}
	c.learnMu.Unlock()
	if err != nil {
		return "", err
	}
	verb := "queued"
	if attached {
		verb = "attached to the Engine's existing job"
	}
	c.event("sync", fmt.Sprintf("Supervised lesson %q %s: %s. Approval remains manual.", snapshot.Title, verb, id))
	if _, err := c.engine.PulseControl("step"); err != nil {
		c.event("problem", "Study job is QUEUED, but pulse step failed: "+err.Error())
		return id, fmt.Errorf("lesson queued as %s but pulse step failed: %w", id, err)
	}
	c.event("sync", "Owner triggered one pulse step for the selected lesson. Check the Learning card for PASSED, RETRY or MODEL_FAILED.")
	return id, nil
}

// identityFix tells the owner the one thing to do about an Identity denial.
func identityFix(why string) string {
	switch {
	case strings.Contains(why, "LOCKED"):
		return "unlock TASS on Identity's page (127.0.0.1:8796) with your passphrase"
	case strings.Contains(why, "no grant"):
		return `on Identity's page use "Enable learning" (adds github.sync and lesson.learn)`
	default:
		return "check Identity's page (127.0.0.1:8796)"
	}
}
