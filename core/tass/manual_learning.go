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
			return fmt.Errorf("Identity denied %s: %s; enable learning on Identity's owner page", action, why)
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
		if item.Status == "QUEUED" {
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
	if ls.Status != "NEW" && ls.Status != "RETRY" {
		status := ls.Status
		c.learnMu.Unlock()
		return "", fmt.Errorf("lesson is %s; only NEW or RETRY lessons may be studied", status)
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
	id := fmt.Sprintf("lesson_%s_%d", hash(snapshot.Path+snapshot.SHA)[:10], snapshot.Attempts+1)
	request := "Study this lesson: \"" + l.Title + "\". Read every step, then answer each check in your REPLY, " +
		"one line per check, exactly like:\nANSWER 1: <your short answer>\nANSWER 2: <your short answer>\n" +
		"Then add REMEMBER: one line saying what you can do now. Use ACTION: NONE."
	ctx := "LESSON (from " + c.cfg.Repo + "/" + p + "):\n" + l.ForStudy() + "\n\nYOUR DOCUMENT:\n" + doc
	if len(ctx) > 60<<10 {
		ctx = ctx[:60<<10]
	}
	if err := c.engine.SubmitJob(engine.Job{
		JobID: id, Identity: c.entityName(), Request: request, Context: ctx, Actions: nil,
	}); err != nil {
		return "", err
	}
	c.learnMu.Lock()
	cur := c.learn.Lessons[p]
	if cur != nil {
		cur.Status, cur.JobID, cur.Attempts = "QUEUED", id, snapshot.Attempts+1
		c.saveLearningLocked()
	}
	c.learnMu.Unlock()
	c.event("sync", fmt.Sprintf("Owner queued supervised lesson %q: %s. Approval remains manual.", snapshot.Title, id))
	if _, err := c.engine.PulseControl("step"); err != nil {
		c.event("problem", "Study job is QUEUED, but pulse step failed: "+err.Error())
		return id, fmt.Errorf("lesson queued as %s but pulse step failed: %w", id, err)
	}
	c.event("sync", "Owner triggered one pulse step for the selected lesson. Check the Learning card for PASSED, RETRY or STUCK.")
	return id, nil
}

