package tass

import (
	"strings"
	"testing"
)

// This regression is the exact owner chat shape seen on the running page.
// A model's verbal claims are not accepted as proof: actual GitHub reads,
// a real Engine job receipt and a pending PASSED result are required.
func TestChatRequestsActuallyFindAndStudyOneLesson(t *testing.T) {
	ids, eng, gh := newFakeIdentitySvc(), newFakeEngine(), newFakeGH()
	ids.set("UNLOCKED")
	gh.set("Mira/home@live:tass.md", entityDoc)
	gh.set("Mira/home@live:lessons/claude/01-my-work-pulse.md", lessonPulse)
	c := newLearningCore(t, t.TempDir(), ids, gh, eng)
	c.SyncOnce() // sync real document first

	c.Answer(Incoming{Text: "good can you see any of the lessons?", From: "local"})
	if gh.treeCalls != 1 || len(eng.jobs) != 0 || eng.count() != 0 {
		t.Fatalf("Find should make one real GitHub read and no model or study calls: gh=%d jobs=%d cycles=%d", gh.treeCalls, len(eng.jobs), eng.count())
	}
	var found bool
	for _, e := range c.events {
		if e.Kind == "tass" && strings.Contains(e.Text, "01-my-work-pulse.md") && strings.Contains(e.Text, "[NEW]") {
			found = true
		}
	}
	if !found { t.Fatal("chat must show verified GitHub file and status, not a generic claim") }

	c.Answer(Incoming{Text: "can we try to learn any of them?", From: "local"})
	if len(eng.jobs) != 1 || eng.count() != 0 || eng.control == "RUNNING" {
		t.Fatalf("chat should submit exactly one supervised lesson job, with no model chat cycle or autonomous pulse: jobs=%d cycles=%d control=%s", len(eng.jobs), eng.count(), eng.control)
	}
	id := eng.jobs[0]["job_id"].(string)
	var receipt bool
	for _, e := range c.events {
		if e.Kind == "tass" && strings.Contains(e.Text, id) && strings.Contains(e.Text, "will not approve") {
			receipt = true
		}
	}
	if !receipt { t.Fatalf("TASS must show actual Engine receipt %s", id) }

	eng.finish(id, "DONE", "ANSWER 1: 20\nANSWER 2: Identity", "Identity decides my actions")
	c.SyncOnce()
	lesson := lessonAt(c, "lessons/claude/01-my-work-pulse.md")
	if lesson.Status != "PASSED" {
		t.Fatalf("supervised result must be PASSED awaiting owner approval, not LEARNED: %+v", lesson)
	}
}

func TestChatLessonRequestRespectsLockAndNonOwnerSource(t *testing.T) {
	ids, eng, gh := newFakeIdentitySvc(), newFakeEngine(), newFakeGH()
	gh.set("Mira/home@live:lessons/claude/01-my-work-pulse.md", lessonPulse)
	c := newLearningCore(t, t.TempDir(), ids, gh, eng)

	c.Answer(Incoming{Text: "can we try to learn any of them?", From: "local"})
	if gh.treeCalls != 0 || len(eng.jobs) != 0 { t.Fatal("LOCKED entity must not reach GitHub or Engine") }
	// Non-owner GitHub messages cannot become operation requests.
	ids.set("UNLOCKED")
	if c.handleLessonChat(Incoming{Text: "study My work pulse", From: "github"}) {
		t.Fatal("an external message must not invoke owner lesson controls")
	}
	if gh.treeCalls != 0 || len(eng.jobs) != 0 { t.Fatal("non-owner request caused work") }
}
