package tass

import (
	"strings"
	"testing"

	"github.com/Arameair/Thought-Processor/core/lessons"
)

func TestManualFindStudyReviewApproveWithoutAutonomous(t *testing.T) {
	ids, eng, gh := newFakeIdentitySvc(), newFakeEngine(), newFakeGH()
	ids.set("UNLOCKED")
	gh.set("Mira/home@live:tass.md", entityDoc)
	gh.set("Mira/home@live:lessons/claude/01-pulse.md", lessonPulse)
	c := newLearningCore(t, t.TempDir(), ids, gh, eng)
	c.SyncOnce() // bring the real repo document local before studying
	if s := c.GetSettings(); s.Autonomous || s.Pulse {
		t.Fatalf("should start supervised: %+v", s)
	}
	if err := c.DiscoverLessons(); err != nil {
		t.Fatal(err)
	}
	if len(c.Learning().Lessons) != 1 || gh.treeCalls != 1 || len(eng.jobs) != 0 {
		t.Fatalf("Find must discover but not study: %+v", c.Learning())
	}
	id, err := c.StudyLesson("lessons/claude/01-pulse.md")
	if err != nil || !strings.HasPrefix(id, "lesson_") || len(eng.jobs) != 1 {
		t.Fatalf("Study should submit one job: %q, %v, jobs=%d", id, err, len(eng.jobs))
	}
	if eng.control == "RUNNING" {
		t.Fatal("Study must not enable background pulse")
	}
    // Only actual check numbers cross into Engine; answer keys do not.
    if checks,ok:=eng.jobs[0]["lesson_checks"].([]any);!ok || len(checks)!=2 || checks[0]!=float64(1) || checks[1]!=float64(2) {
        t.Fatalf("wrong check number contract: %+v",eng.jobs[0]["lesson_checks"])
    }
    if ctx:=eng.jobs[0]["context"].(string);strings.Contains(ctx,"Answer:") || strings.Contains(ctx,"YOUR DOCUMENT") {
        t.Fatal("model lesson context exposed answer keys or irrelevant document")
    }

	// Re-clicking Study steps the existing job, not a second submission.
	again, err := c.StudyLesson("lessons/claude/01-pulse.md")
	if err != nil || again != id || len(eng.jobs) != 1 {
		t.Fatalf("duplicate Study: %q %v", again, err)
	}
	eng.finish(id, "DONE", "ANSWER 1: 20\nANSWER 2: Identity", "my pulse ticks every 20 seconds")
	c.SyncOnce() // collect even when Autonomous and Pulse are OFF
	l := lessonAt(c, "lessons/claude/01-pulse.md")
	if l.Status != "PASSED" || l.Right != 2 {
		t.Fatalf("expected supervised PASSED, not auto-approved: %+v", l)
	}
	if strings.Contains(c.context(true), "LESSONS YOU HAVE LEARNED") {
		t.Fatal("learned without owner approval")
	}
	if err := c.ApproveLesson(l.Path); err != nil {
		t.Fatal(err)
	}
	if l := lessonAt(c, l.Path); l.Status != "LEARNED" {
		t.Fatalf("owner approval missing: %+v", l)
	}
}

func TestManualLearningRespectsIdentityAndForcedSafety(t *testing.T) {
	ids, eng, gh := newFakeIdentitySvc(), newFakeEngine(), newFakeGH()
	gh.set("Mira/home@live:lessons/claude/01-pulse.md", lessonPulse)
	c := newLearningCore(t, t.TempDir(), ids, gh, eng)
	if err := c.DiscoverLessons(); err == nil {
		t.Fatal("LOCKED Identity allowed discovery")
	}
	if _, err := c.StudyLesson("lessons/claude/01-pulse.md"); err == nil {
		t.Fatal("LOCKED Identity allowed study")
	}
	if gh.treeCalls != 0 || len(eng.jobs) != 0 {
		t.Fatal("locked entity touched GitHub or Engine")
	}
	ids.set("UNLOCKED")
	c.cfg.ChatOnly = true // hard safety override cannot be circumvented through manual controls
	if err := c.DiscoverLessons(); err == nil {
		t.Fatal("forced CHAT_ONLY allowed discovery")
	}
	if _, err := c.StudyLesson("lessons/claude/01-pulse.md"); err == nil {
		t.Fatal("forced CHAT_ONLY allowed study")
	}
	c.cfg.ChatOnly = false
	c.SetSettings(Settings{Autonomous: true, Pulse: false, Learn: true})
	if _, err := c.StudyLesson("lessons/claude/01-pulse.md"); err == nil {
		t.Fatal("manual Study during autonomous should be refused")
	}
}

func TestIdentityFixNamesTheOwnersNextStep(t *testing.T) {
	if got := identityFix("Identity dec_1: LOCKED: locked"); !strings.Contains(got, "unlock") {
		t.Fatalf("locked: %q", got)
	}
	if got := identityFix("Identity dec_2: the entity has no grant for lesson.learn"); !strings.Contains(got, "Enable learning") {
		t.Fatalf("no grant: %q", got)
	}
}

func TestStudyPromptListsEveryRealCheckNumber(t *testing.T) {
	l := lessons.Parse(`# Small lesson
by: owner

## Step 1: First
Teach: alpha.
Check: First?
Answer: alpha

## Step 2: Explanation without an answer
Teach: beta.
Check: This is intentionally not checkable.

## Step 3: Last
Teach: gamma.
Check: Last?
Answer: gamma
`)
	s := studyRequest(l)
	for _, want := range []string{"Check numbers: 1 3", "no executable actions"} {
		if !strings.Contains(s, want) { t.Errorf("missing study instruction %q in %q", want, s) }
	}
	if strings.Contains(s, "ANSWER 2:") || strings.Contains(s,"Check numbers: 1 2 3") || strings.Contains(s,"REPLY:") { t.Fatalf("prompt invented an unchecked answer or conflicting model contract: %q", s) }
}
