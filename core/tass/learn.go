package tass

// The lessons loop (app mode). While the owner has the pulse turned on, every 20 seconds:
//
//  1. GitHub: one conditional request lists the lessons folder of the owner's repository; new or
//     changed lessons are downloaded and kept locally (so learning goes on offline).
//  2. One lesson at a time is handed to the Thought Engine's pulse as a job. The entity studies it
//     (it never sees the expected answers) and answers each check.
//  3. The core, not the model, grades the answers against the lesson.
//  4. A lesson that passes waits for the owner's approval. Approved, it becomes LEARNED: one line in
//     the "## Learned" section of the entity's document, which syncs back to GitHub.
//
// Identity decides each step (github.sync, lesson.learn, pulse.run). Nothing here runs code.

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Arameair/Thought-Processor/core/engine"
	"github.com/Arameair/Thought-Processor/core/lessons"
)

// Settings the owner turns on and off on the page. Kept in settings.json, applied every 20 s.
type Settings struct {
	// Autonomous is the master switch. Off (the default on a new install) is chat-only: no model action,
	// no pulse task, no work pulse. On, the entity works on its own: the 20-s pulse, lessons from GitHub,
	// read-only actions that pass the locked-prediction and Identity checks, and lessons that pass every
	// check are approved automatically (marked "auto-approved").
	Autonomous bool `json:"autonomous"`
	Pulse      bool `json:"pulse"` // the 20-s work pulse runs (and comes back on after every restart)
	Learn      bool `json:"learn"` // each pulse looks for new lessons on GitHub and studies them
}

// chatOnly: no actions, tasks or pulse. Forced by TASS_CHAT_ONLY=on, or the owner's Autonomous switch is off.
func (c *Core) chatOnly() bool {
	if c.cfg.ChatOnly {
		return true
	}
	return c.cfg.IdentitySvc != nil && !c.GetSettings().Autonomous
}

// LessonState is what the core knows about one lesson.
type LessonState struct {
	Path       string           `json:"path"`
	SHA        string           `json:"sha"`
	Title      string           `json:"title"`
	By         string           `json:"by,omitempty"`
	Goal       string           `json:"goal,omitempty"`
	Status     string           `json:"status"` // NEW | QUEUED | MODEL_FAILED | PASSED | RETRY | STUCK | LEARNED | NEEDS_WORKSPACE | NO_CHECKS
	Checks     int              `json:"checks"`
	Attempts   int              `json:"attempts"` // attempts since the owner last pressed Retry (limits STUCK)
	JobSeq     int              `json:"job_seq"`  // highest Engine job number ever used for this revision; never decreases (#71)
	History    []AttemptRecord  `json:"history,omitempty"`
	JobID      string           `json:"job_id,omitempty"`
	CycleID    string           `json:"cycle_id,omitempty"`
	Right      int              `json:"right"`
	Results    []lessons.Result `json:"results,omitempty"`
	Reply      string           `json:"reply,omitempty"`
	Remember   string           `json:"remember,omitempty"`
	FoundAt    time.Time        `json:"found_at"`
	StudiedAt  *time.Time       `json:"studied_at,omitempty"`
	LearnedAt  *time.Time       `json:"learned_at,omitempty"`
	Note       string           `json:"note,omitempty"`
	Superseded string           `json:"superseded,omitempty"` // sha of the version learned before it changed
}

const maxAttempts = 3

type learnState struct {
	TreeETag string                  `json:"tree_etag"`
	Lessons  map[string]*LessonState `json:"lessons"`
	LastLook time.Time               `json:"last_look"`
	LastNote string                  `json:"last_note"`
}

func (c *Core) loadLearning() {
	c.learnMu.Lock()
	defer c.learnMu.Unlock()
	c.learn = learnState{Lessons: map[string]*LessonState{}}
	if b, err := os.ReadFile(c.file("lessons.json")); err == nil {
		_ = json.Unmarshal(b, &c.learn)
		if c.learn.Lessons == nil {
			c.learn.Lessons = map[string]*LessonState{}
		}
	}
	migrated := false
	for _, ls := range c.learn.Lessons {
		if migrateLessonLocked(ls, c.now().UTC()) {
			migrated = true
		}
	}
	if migrated {
		c.saveLearningLocked()
	}
	c.settings = Settings{Learn: true}
	if b, err := os.ReadFile(c.file("settings.json")); err == nil {
		_ = json.Unmarshal(b, &c.settings)
	}
}

func (c *Core) saveLearningLocked() {
	b, _ := json.MarshalIndent(c.learn, "", "  ")
	tmp := c.file("lessons.json.tmp")
	if os.WriteFile(tmp, b, 0o600) == nil {
		_ = os.Rename(tmp, c.file("lessons.json"))
	}
}

// GetSettings / SetSettings: the page's switches.
func (c *Core) GetSettings() Settings {
	c.learnMu.Lock()
	defer c.learnMu.Unlock()
	return c.settings
}

func (c *Core) SetSettings(s Settings) error {
	c.learnMu.Lock()
	c.settings = s
	b, _ := json.Marshal(s)
	err := os.WriteFile(c.file("settings.json"), b, 0o600)
	c.learnMu.Unlock()
	c.event("sync", fmt.Sprintf("Settings: autonomous %s · work pulse %s · learn from GitHub %s.", onOff(s.Autonomous), onOff(s.Pulse), onOff(s.Learn)))
	c.applySettings()
	return err
}

func onOff(b bool) string {
	if b {
		return "ON"
	}
	return "off"
}

// applySettings makes the engine's pulse match the setting (it is PAUSED after every engine start).
// Only while Identity is UNLOCKED: a locked entity's pulse would only report BLOCKED.
func (c *Core) applySettings() {
	if c.cfg.IdentitySvc == nil {
		return
	}
	want := c.GetSettings().Pulse && !c.chatOnly()
	ps, err := c.engine.PulseStatus()
	if err != nil {
		return
	}
	switch {
	case want && ps.Control == "PAUSED":
		if st := c.cfg.IdentitySvc.State(); st.State != "UNLOCKED" {
			return
		}
		if _, err := c.engine.PulseControl("start"); err == nil {
			c.event("sync", "Work pulse started (the setting is ON): a tick every 20 seconds.")
		}
	case !want && ps.Control == "RUNNING":
		if _, err := c.engine.PulseControl("pause"); err == nil {
			c.event("sync", "Work pulse paused (the setting is off).")
		}
	}
}

func (c *Core) learningOn() bool {
	s := c.GetSettings()
	return s.Pulse && s.Learn && c.cfg.Repo != "" && !c.chatOnly()
}

func (c *Core) lessonsDir() string {
	return strings.Trim(firstNonEmpty(c.cfg.LessonsDir, "lessons"), "/")
}

// syncLessons: one conditional request for the branch's file list; download what's new or changed.
func (c *Core) syncLessons(manual bool) error {
	if !manual && !c.learningOn() {
		return nil
	}
	if ok, why := c.allowedNow("lesson.learn"); !ok {
		c.noteLearning("Not looking for lessons: " + why)
		return nil
	}
	c.learnMu.Lock()
	etag := c.learn.TreeETag
	c.learnMu.Unlock()
	tree, got, err := c.gh.Tree(c.cfg.Repo, c.cfg.Branch, etag)
	if err != nil {
		return fmt.Errorf("lessons: %w", err)
	}
	c.learnMu.Lock()
	c.learn.LastLook = c.now().UTC()
	c.learnMu.Unlock()
	if got.NotModified {
		return nil
	}
	dir := c.lessonsDir() + "/"
	found := 0
	seen := map[string]bool{}
	for _, e := range tree {
		low := strings.ToLower(e.Path)
		if e.Type != "blob" || !strings.HasPrefix(e.Path, dir) || !(strings.HasSuffix(low, ".md") || strings.HasSuffix(low, ".ipynb")) ||
			strings.Contains(e.Path, ".ipynb_checkpoints") ||
			strings.EqualFold(path.Base(e.Path), "README.md") || strings.Contains(e.Path, "..") {
			continue
		}
		seen[e.Path] = true
		c.learnMu.Lock()
		old := c.learn.Lessons[e.Path]
		c.learnMu.Unlock()
		if old != nil && old.SHA == e.SHA {
			continue
		}
		f, _, err := c.gh.GetFile(c.cfg.Repo, c.cfg.Branch, e.Path, "")
		if err != nil {
			return fmt.Errorf("lesson %s: %w", e.Path, err)
		}
		local := filepath.Join(c.file("lessons"), filepath.FromSlash(e.Path))
		_ = os.MkdirAll(filepath.Dir(local), 0o700)
		if err := os.WriteFile(local, []byte(f.Text), 0o600); err != nil {
			return err
		}
		l := lessons.Parse(f.Text)
		if strings.HasSuffix(low, ".ipynb") {
			l = lessons.Notebook(f.Text)
		}
		ls := &LessonState{Path: e.Path, SHA: e.SHA, Title: firstNonEmpty(l.Title, path.Base(e.Path)), By: l.By, Goal: l.Goal,
			Status: "NEW", Checks: len(l.Checks()), FoundAt: c.now().UTC()}
		switch {
		case strings.HasSuffix(low, ".ipynb"):
			ls.Status, ls.Note = "NEEDS_WORKSPACE", fmt.Sprintf("notebook lesson (%d cells, each with its thought): run it cell by cell in JupyterLab; this app doesn't run code", len(l.Steps))
		case !l.Checkable && l.HasCode:
			ls.Status, ls.Note = "NEEDS_WORKSPACE", "made of code cells for the workspace: this app doesn't run code, so it can't teach it"
		case !l.Checkable:
			ls.Status, ls.Note = "NO_CHECKS", "no step has a Check and an Answer, so learning it can't be verified"
		}
		if old != nil && old.Status == "LEARNED" {
			ls.Superseded, ls.Note = old.SHA, strings.TrimSpace("changed on GitHub after it was learned; studying the new version. "+ls.Note)
		}
		if old != nil { // a new revision starts its own attempts, but earlier evidence stays with the lesson
			c.learnMu.Lock()
			migrateLessonLocked(old, c.now().UTC())
			ls.History = append([]AttemptRecord(nil), old.History...)
			c.learnMu.Unlock()
		}
		c.learnMu.Lock()
		c.learn.Lessons[e.Path] = ls
		c.learnMu.Unlock()
		found++
		c.event("sync", fmt.Sprintf("New lesson from GitHub: %q (%s) · %s", ls.Title, e.Path, ls.Status))
	}
	c.learnMu.Lock()
	for p, ls := range c.learn.Lessons {
		if !seen[p] && ls.Status != "LEARNED" && ls.Status != "GONE" {
			ls.Status, ls.Note = "GONE", "no longer on GitHub"
		}
	}
	c.learn.TreeETag = got.ETag
	c.saveLearningLocked()
	c.learnMu.Unlock()
	if found == 0 {
		c.noteLearning("Looked at GitHub: no new lessons.")
	}
	return nil
}

func (c *Core) noteLearning(s string) {
	c.learnMu.Lock()
	c.learn.LastNote = s
	c.learnMu.Unlock()
}

// advanceLessons: collect a finished study job and grade it; otherwise hand the next lesson to the pulse.
func (c *Core) advanceLessons() {
	if c.cfg.IdentitySvc == nil {
		return
	}
	c.learnMu.Lock()
	var active *LessonState
	for _, ls := range c.learn.Lessons {
		if ls.Status == "QUEUED" {
			active = ls
		}
	}
	c.learnMu.Unlock()
	if active != nil {
		c.collect(active)
		return
	}
	if !c.learningOn() {
		return
	}
	next := c.nextLesson()
	if next == nil {
		return
	}
	if ok, why := c.allowedNow("lesson.learn"); !ok {
		c.noteLearning("Not studying: " + why)
		return
	}
	text, err := os.ReadFile(filepath.Join(c.file("lessons"), filepath.FromSlash(next.Path)))
	if err != nil {
		return
	}
	l := lessons.Parse(string(text))
	name := c.entityName()
	req := studyRequest(l)
	ctx := "LESSON (from " + c.cfg.Repo + "/" + next.Path + "):\n" + l.ForStudy()
	if len(ctx) > 60<<10 {
		ctx = ctx[:60<<10]
	}
	c.learnMu.Lock()
	path, sha, lastSeq := next.Path, next.SHA, next.JobSeq
	c.learnMu.Unlock()
	jobID, seq, _, found, err := c.submitLessonJob(path, sha, lastSeq, func(id string) engine.Job {
		return engine.Job{JobID: id, Identity: name, Request: req, Context: ctx, LessonChecks: studyCheckNumbers(l)}
	})
	c.learnMu.Lock()
	for _, r := range found {
		next.addHistory(r)
	}
	if seq > next.JobSeq {
		next.JobSeq = seq
	}
	if err != nil {
		c.saveLearningLocked()
		c.learnMu.Unlock()
		c.noteLearning("Couldn't hand the lesson to the pulse: " + err.Error())
		return
	}
	next.Status, next.JobID, next.Attempts = "QUEUED", jobID, next.Attempts+1
	c.saveLearningLocked()
	c.learnMu.Unlock()
	c.event("sync", fmt.Sprintf("Studying %q on the next pulse (attempt %d, job %s).", next.Title, next.Attempts, jobID))
}

func (c *Core) nextLesson() *LessonState {
	c.learnMu.Lock()
	defer c.learnMu.Unlock()
	var list []*LessonState
	for _, ls := range c.learn.Lessons {
		if ls.Status == "NEW" || ls.Status == "RETRY" {
			list = append(list, ls)
		}
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Path < list[j].Path })
	if len(list) == 0 {
		return nil
	}
	return list[0]
}

func (c *Core) collect(ls *LessonState) {
	ps, err := c.engine.PulseStatus()
	if err != nil {
		return
	}
	var job *engine.JobStatus
	for i := range ps.Jobs {
		if ps.Jobs[i].JobID == ls.JobID {
			job = &ps.Jobs[i]
		}
	}
	if job == nil || job.State == "QUEUED" || job.State == "RUNNING" {
		return
	}
	text, err := os.ReadFile(filepath.Join(c.file("lessons"), filepath.FromSlash(ls.Path)))
	if err != nil {
		return
	}
	l := lessons.Parse(string(text))
	now := c.now().UTC()
	c.learnMu.Lock()
	defer c.learnMu.Unlock()
	ls.CycleID, ls.StudiedAt, ls.Reply, ls.Remember = job.CycleID, &now, job.Result, job.Remember
	if job.State != "DONE" {
		ls.Results, ls.Right, ls.Status = nil, 0, "MODEL_FAILED"
		ls.Note = "Model/Engine failure — NOT GRADED (" + job.State + "): " + oneLine(job.Result, 160)
		ls.addHistory(AttemptRecord{JobID: ls.JobID, CycleID: job.CycleID, SHA: ls.SHA, State: job.State, Outcome: "MODEL_FAILED",
			Checks: ls.Checks, Reply: job.Result, Note: ls.Note, Source: "study", At: now})
		c.eventLocked("problem", fmt.Sprintf("Lesson %q: Engine job %s; not graded and not learned.",ls.Title,job.State))
		c.saveLearningLocked()
		return
	}
	defer func() {
		ls.addHistory(AttemptRecord{JobID: ls.JobID, CycleID: job.CycleID, SHA: ls.SHA, State: job.State, Outcome: ls.Status,
			Right: ls.Right, Checks: ls.Checks, Reply: job.Result, Source: "study", At: now})
		c.saveLearningLocked()
	}()
	ls.Results, ls.Right, _ = lessons.Grade(l, job.Result)
	ls.Note = ""
	if ls.Checks>0 && ls.Right==ls.Checks {
		ls.Status="PASSED"
		c.eventLocked("sync",fmt.Sprintf("Lesson %q: %d/%d checks right. Waiting for explicit owner approval.",ls.Title,ls.Right,ls.Checks))
	} else if ls.Attempts>=maxAttempts {
		ls.Status="STUCK"
		c.eventLocked("problem",fmt.Sprintf("Lesson %q: %d/%d graded correct after %d attempts; owner review required.",ls.Title,ls.Right,ls.Checks,ls.Attempts))
	} else {
		ls.Status="RETRY"
		c.eventLocked("sync",fmt.Sprintf("Lesson %q: %d/%d graded correct; not approved.",ls.Title,ls.Right,ls.Checks))
	}
	c.saveLearningLocked()
}

// eventLocked: event() while learnMu is held (event takes c.mu only).
func (c *Core) eventLocked(kind, text string) { c.event(kind, text) }

var ErrNoLesson = errors.New("no such lesson")

// ApproveLesson: the owner says a PASSED lesson counts. It becomes LEARNED and is written into the
// "## Learned" section of the entity's document (which syncs to GitHub).
func (c *Core) ApproveLesson(p string) error {
	if c.cfg.IdentitySvc==nil || c.cfg.ChatOnly {
		return errors.New("owner approval unavailable in forced safety mode")
	}
	if ok,why:=c.allowedNow("lesson.learn");!ok {
		return fmt.Errorf("Identity denied lesson approval: %s",why)
	}
	// Prevent two simultaneous approvals from creating duplicate learned entries.
	c.manualMu.Lock()
	defer c.manualMu.Unlock()
	c.learnMu.Lock()
	ls:=c.learn.Lessons[p]
	if ls==nil { c.learnMu.Unlock(); return ErrNoLesson }
	if ls.Status!="PASSED" || ls.Right!=ls.Checks || ls.Checks==0 || len(ls.Results)!=ls.Checks {
		state:=ls.Status
		c.learnMu.Unlock()
		return fmt.Errorf("only a fully graded PASSED lesson can be approved (this one is %s)",state)
	}
	snapshot:=*ls
	c.learnMu.Unlock()
	// Point-of-use authorization, including a lock/revocation after grading.
	if ok,why:=c.allowedNow("lesson.learn");!ok {
		return fmt.Errorf("Identity denied lesson approval: %s",why)
	}
	now:=c.now().UTC()
	line:=fmt.Sprintf("- %s: %s (lesson `%s`, %d/%d, approved %s)",
		snapshot.Title,firstNonEmpty(oneLine(snapshot.Remember,200),oneLine(snapshot.Goal,200),"verified lesson"),
		p,snapshot.Right,snapshot.Checks,now.Format("2006-01-02"))
	c.fileMu.Lock()
	doc,exists:=c.read("tass.md")
	var err error
	if !exists {
		err=errors.New("entity document unavailable")
	} else if !strings.Contains(doc,"(lesson `"+p+"`,") {
		err=c.write("tass.md",addToSection(doc,"Learned",line))
	}
	c.fileMu.Unlock()
	if err!=nil { return fmt.Errorf("could not persist approved lesson: %w",err) }
	c.learnMu.Lock()
	current:=c.learn.Lessons[p]
	if current==nil || current.Status!="PASSED" || current.SHA!=snapshot.SHA {
		c.learnMu.Unlock()
		return errors.New("lesson changed during approval; reconcile source and local evidence")
	}
	current.Status,current.LearnedAt="LEARNED",&now
	c.saveLearningLocked()
	c.learnMu.Unlock()
	c.addMemory("learned "+snapshot.Title,snapshot.CycleID,"owner approval")
	c.event("sync",fmt.Sprintf("Lesson %q LEARNED after explicit owner approval. Recorded in local Learned section.",snapshot.Title))
	return nil
}

// RetryLesson: start a STUCK (or any not-learned) lesson over.
func (c *Core) RetryLesson(p string) error {
	c.learnMu.Lock()
	defer c.learnMu.Unlock()
	ls := c.learn.Lessons[p]
	if ls == nil {
		return ErrNoLesson
	}
	if ls.Status == "QUEUED" || ls.Status == "LEARNED" || ls.Status == "NEEDS_WORKSPACE" || ls.Status == "NO_CHECKS" || ls.Status == "GONE" {
		return fmt.Errorf("can't retry a lesson that is %s", ls.Status)
	}
	// Attempts restarts (it limits STUCK); JobSeq and History are kept, so the next Study
	// gets a job ID the Engine has never seen and earlier answers stay on record (#71).
	migrateLessonLocked(ls, c.now().UTC())
	ls.Status, ls.Attempts, ls.Note = "NEW", 0, "you asked for another try; earlier attempts are kept in its history"
	ls.Results, ls.Right, ls.Reply, ls.Remember, ls.JobID, ls.CycleID = nil, 0, "", "", "", ""
	c.saveLearningLocked()
	return nil
}

// addToSection appends line to "## name" (creating the section at the end if needed).
func addToSection(doc, name, line string) string {
	lines := strings.Split(strings.TrimRight(doc, "\n"), "\n")
	start := -1
	for i, l := range lines {
		if strings.TrimSpace(strings.TrimPrefix(l, "## ")) == name && strings.HasPrefix(l, "## ") {
			start = i
		}
	}
	if start < 0 {
		return strings.TrimRight(doc, "\n") + "\n\n## " + name + "\n\n" + line + "\n"
	}
	end := len(lines)
	for i := start + 1; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "## ") || strings.HasPrefix(lines[i], "# ") {
			end = i
			break
		}
	}
	at := end
	for at > start+1 && strings.TrimSpace(lines[at-1]) == "" {
		at--
	}
	out := append([]string{}, lines[:at]...)
	if at == start+1 {
		out = append(out, "")
	}
	out = append(out, line)
	if end < len(lines) {
		out = append(out, "")
	}
	out = append(out, lines[end:]...)
	return strings.Join(out, "\n") + "\n"
}

// LearningStatus is what the page shows.
type LearningStatus struct {
	Settings Settings       `json:"settings"`
	Repo     string         `json:"repo,omitempty"`
	Branch   string         `json:"branch,omitempty"`
	Dir      string         `json:"dir,omitempty"`
	LastLook time.Time      `json:"last_look"`
	LastNote string         `json:"last_note,omitempty"`
	Lessons  []LessonState  `json:"lessons"`
	Counts   map[string]int `json:"counts"`
}

func (c *Core) Learning() LearningStatus {
	c.learnMu.Lock()
	defer c.learnMu.Unlock()
	out := LearningStatus{Settings: c.settings, Repo: c.cfg.Repo, Branch: c.cfg.Branch, Dir: c.lessonsDir(),
		LastLook: c.learn.LastLook, LastNote: c.learn.LastNote, Counts: map[string]int{}}
	for _, ls := range c.learn.Lessons {
		out.Lessons = append(out.Lessons, *ls)
		out.Counts[ls.Status]++
	}
	sort.Slice(out.Lessons, func(i, j int) bool { return out.Lessons[i].Path < out.Lessons[j].Path })
	return out
}

// learnedForContext: short lines about approved lessons, so the entity can use what it learned in chat.
func (c *Core) learnedForContext() string {
	c.learnMu.Lock()
	defer c.learnMu.Unlock()
	var b strings.Builder
	for _, ls := range c.learn.Lessons {
		if ls.Status == "LEARNED" {
			fmt.Fprintf(&b, "- %s: %s\n", ls.Title, firstNonEmpty(ls.Remember, ls.Goal))
			// Only owner-approved and correctly graded answers enter effective
			// context. This is retrieval of verified content, not model training.
			for i,r := range ls.Results {
				if i>=10 { break }
				if r.Right {
					fmt.Fprintf(&b, "  Verified check %d: %s => %s\n",r.N,oneLine(r.Check,150),oneLine(r.Given,200))
				}
			}
		}
	}
	return b.String()
}

func firstNonEmpty(s ...string) string {
	for _, x := range s {
		if strings.TrimSpace(x) != "" {
			return x
		}
	}
	return ""
}
