package tass

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Arameair/Thought-Processor/core/engine"
	"github.com/Arameair/Thought-Processor/core/github"
	"github.com/Arameair/Thought-Processor/core/identitysvc"
)

const lessonPulse = `# How my pulse works
by: Claude

Know what happens every 20 seconds.

## Step 1: The beat
Teach: My work pulse ticks every 20 seconds while it is on.
Check: How many seconds between two ticks?
Answer: 20

## Step 2: Who decides
Teach: Identity decides before anything runs.
Check: Which service decides whether an action may run?
Answer: Identity
`

const entityDoc = "# Mira\n\n## Who I am\n\nI am Mira.\n\n## Memory\n\n## Now\n\nLearning.\n"

func newLearningCore(t *testing.T, dir string, ids *fakeIdentitySvc, gh *fakeGH, eng *fakeEngine) *Core {
	t.Helper()
	kf := filepath.Join(t.TempDir(), "core.key")
	os.WriteFile(kf, []byte("core-service-key-0123456789abcdef0123\n"), 0o600)
	cl, _ := identitysvc.New(ids.srv.URL, kf)
	g := github.New("test-token")
	g.Base = gh.srv.URL
	ec := engine.New(eng.srv.URL)
	ec.Key = "testkey"
	c, err := New(Config{DataDir: dir, Identity: "tass-identity", Label: "tass", SyncEvery: 20 * time.Second, MaxBackoff: time.Minute,
		IdentitySvc: cl, GitHub: true, Repo: "Mira/home", Branch: "live", DocPath: "tass.md", LessonsDir: "lessons"}, g, ec)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestLessonsLoopFromGitHubToLearned(t *testing.T) {
	ids, eng, gh := newFakeIdentitySvc(), newFakeEngine(), newFakeGH()
	ids.set("UNLOCKED")
	gh.set("Mira/home@live:tass.md", entityDoc)
	gh.set("Mira/home@live:lessons/README.md", "# Lessons\nhow to write them")
	gh.set("Mira/home@live:lessons/claude/01-pulse.md", lessonPulse)
	gh.set("Mira/home@live:lessons/claude/00-old.md", "# Make a tool\n\n## Step 1: x\n```python\nprint(1)\n```\n")
	dir := t.TempDir()
	c := newLearningCore(t, dir, ids, gh, eng)

	// Pulse setting off: the document syncs, but no lessons are looked at and nothing is studied.
	c.SyncOnce()
	if local(t, dir) != entityDoc || gh.treeCalls != 0 || len(eng.jobs) != 0 {
		t.Fatalf("setting off: doc %q, tree calls %d, jobs %d", local(t, dir), gh.treeCalls, len(eng.jobs))
	}
	// Turn the pulse on: the engine pulse starts, the next sync finds the lessons and queues ONE.
	if err := c.SetSettings(Settings{Pulse: true, Learn: true}); err != nil {
		t.Fatal(err)
	}
	if eng.control != "RUNNING" {
		t.Fatal("setting ON starts the engine's pulse")
	}
	c.SyncOnce()
	l := c.Learning()
	if len(l.Lessons) != 2 || l.Counts["NEEDS_WORKSPACE"] != 1 || l.Counts["QUEUED"] != 1 || len(eng.jobs) != 1 {
		t.Fatalf("lessons: %+v jobs %d", l.Counts, len(eng.jobs))
	}
	job := eng.jobs[0]
	ctx := job["context"].(string)
	if strings.Contains(ctx, "Answer:") || !strings.Contains(ctx, "Check 2: Which service") || !strings.Contains(ctx, "# Mira") || job["identity"] != "Mira" {
		t.Fatalf("the entity studies the lesson without its answers, as itself:\n%s", ctx)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "lessons", "lessons", "claude", "01-pulse.md")); string(b) != lessonPulse {
		t.Fatal("lessons are kept locally for offline study")
	}
	// Unchanged on GitHub: the next look costs a 304, and nothing is queued twice.
	c.SyncOnce()
	if len(eng.jobs) != 1 {
		t.Fatal("one lesson at a time; never queued twice")
	}
	// First attempt: one wrong answer → RETRY, queued again on a later pulse.
	id1 := job["job_id"].(string)
	eng.finish(id1, "DONE", "ANSWER 1: 20 seconds\nANSWER 2: the model decides", "")
	c.SyncOnce()
	if ls := lessonAt(c, "lessons/claude/01-pulse.md"); ls.Status != "RETRY" || ls.Right != 1 {
		t.Fatalf("graded by the core, not the model: %+v", ls)
	}
	c.SyncOnce()
	if len(eng.jobs) != 2 || eng.jobs[1]["job_id"] == id1 {
		t.Fatalf("retry is a new job: %v", eng.jobs)
	}
	eng.finish(eng.jobs[1]["job_id"].(string), "DONE", "I studied it.\nANSWER 1: twenty\nANSWER 2: Identity", "my pulse ticks every 20 s and Identity decides")
	c.SyncOnce()
	got := lessonAt(c, "lessons/claude/01-pulse.md")
	if got.Status != "PASSED" || got.Right != 2 {
		t.Fatalf("passed: %+v", got)
	}
	if strings.Contains(local(t, dir), "## Learned") {
		t.Fatal("nothing counts as learned before the owner approves")
	}
	// Owner approves → LEARNED, written to the document, synced to GitHub.
	if err := c.ApproveLesson("lessons/claude/00-old.md"); err == nil {
		t.Fatal("a lesson that didn't pass can't be approved")
	}
	if err := c.ApproveLesson(got.Path); err != nil {
		t.Fatal(err)
	}
	c.SyncOnce()
	remote := gh.get("Mira/home@live:tass.md")
	if !strings.Contains(remote, "## Learned") || !strings.Contains(remote, "How my pulse works: my pulse ticks every 20 s") {
		t.Fatalf("learned lessons sync to GitHub:\n%s", remote)
	}
	if !strings.Contains(c.context(true), "LESSONS YOU HAVE LEARNED") {
		t.Fatal("what was learned is used in conversation")
	}
	// Settings survive a restart.
	c2 := newLearningCore(t, dir, ids, gh, eng)
	if s := c2.GetSettings(); !s.Pulse || !s.Learn || c2.Learning().Counts["LEARNED"] != 1 {
		t.Fatalf("after restart: %+v %+v", s, c2.Learning().Counts)
	}
}

func TestLockedIdentityHoldsGitHubAndLessons(t *testing.T) {
	ids, eng, gh := newFakeIdentitySvc(), newFakeEngine(), newFakeGH()
	gh.set("Mira/home@live:tass.md", entityDoc)
	gh.set("Mira/home@live:lessons/a.md", lessonPulse)
	dir := t.TempDir()
	c := newLearningCore(t, dir, ids, gh, eng)
	c.SetSettings(Settings{Pulse: true, Learn: true})
	c.SyncOnce()
	if fileExists(filepath.Join(dir, "tass.md")) || gh.treeCalls != 0 || len(eng.jobs) != 0 || eng.control == "RUNNING" {
		t.Fatal("LOCKED: no GitHub sync, no lessons, no pulse")
	}
	if !strings.Contains(c.Status().SyncHold, "LOCKED") {
		t.Fatalf("status says why: %q", c.Status().SyncHold)
	}
}

func TestAddToSection(t *testing.T) {
	d := addToSection("# A\n\n## Learned\n\n- one\n\n## Now\n\nx\n", "Learned", "- two")
	if d != "# A\n\n## Learned\n\n- one\n- two\n\n## Now\n\nx\n" {
		t.Fatalf("%q", d)
	}
	d = addToSection("# A\n\n## Now\n\nx\n", "Learned", "- one")
	if !strings.HasSuffix(d, "## Learned\n\n- one\n") {
		t.Fatalf("%q", d)
	}
}

func lessonAt(c *Core, p string) LessonState {
	for _, ls := range c.Learning().Lessons {
		if ls.Path == p {
			return ls
		}
	}
	return LessonState{}
}
