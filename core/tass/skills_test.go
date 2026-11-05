package tass

// #72: the first @ skill. Fixture Identity/Engine/GitHub; not the owner's real Nemotron.

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func skillCore(t *testing.T, granted bool) (*fakeIdentitySvc, *fakeEngine, *fakeGH, string, *Core) {
	t.Helper()
	ids, eng, gh := newFakeIdentitySvc(), newFakeEngine(), newFakeGH()
	ids.set("UNLOCKED")
	ids.skillGranted = granted
	gh.set("Mira/home@live:tass.md", entityDoc)
	gh.set("Mira/home@live:lessons/claude/01-pulse.md", lessonPulse)
	gh.set("Mira/home@live:"+pulsePath, threeCheckPulseLesson)
	dir := t.TempDir()
	c := newLearningCore(t, dir, ids, gh, eng)
	c.SyncOnce()
	if err := c.DiscoverLessons(); err != nil {
		t.Fatal(err)
	}
	return ids, eng, gh, dir, c
}

func lastReceipt(c *Core) SkillReceipt {
	r := c.Skills(false).Receipts
	if len(r) == 0 {
		return SkillReceipt{}
	}
	return r[0]
}

func lines(t *testing.T, p string) int {
	b, _ := os.ReadFile(p)
	return strings.Count(string(b), "\n")
}

func TestAtSkillRequestWaitsThenRunsExactlyOnce(t *testing.T) {
	_, eng, _, dir, c := skillCore(t, true)
	c.Answer(Incoming{Text: "Find lessons using @", From: "local"})
	r := lastReceipt(c)
	if r.Skill != skillListLessons || r.State != "AWAITING_APPROVAL" || r.Executed || !strings.Contains(r.Why, "lessons") || r.SelectedBy != "owner @mention" {
		t.Fatalf("request should resolve and wait: %+v", r)
	}
	if eng.count() != 0 {
		t.Fatal("a skill request must not become a model cycle")
	}
	done, err := c.ApproveSkill(r.ReceiptID)
	if err != nil || done.State != "DONE" || !done.Executed || done.ExitCode == nil || *done.ExitCode != 0 {
		t.Fatalf("approve: %v %+v", err, done)
	}
	for _, want := range []string{"lessons/claude/01-pulse.md\tmarkdown", pulsePath + "\tmarkdown", "count=2"} {
		if !strings.Contains(done.Output, want) {
			t.Fatalf("real output missing %q:\n%s", want, done.Output)
		}
	}
	sum := sha256.Sum256([]byte(done.Output))
	if done.OutputSHA256 != hex.EncodeToString(sum[:]) || done.ExpectationMet == nil || !*done.ExpectationMet || !strings.Contains(done.RunDecided, "skill.run:"+skillListLessons) {
		t.Fatalf("evidence incomplete: %+v", done)
	}
	n := lines(t, filepath.Join(dir, "skill_receipts.jsonl"))
	again, err := c.ApproveSkill(r.ReceiptID)
	if err != nil || again.FinishedAt == nil || !again.FinishedAt.Equal(*done.FinishedAt) || lines(t, filepath.Join(dir, "skill_receipts.jsonl")) != n {
		t.Fatalf("second approve must not run again: %v %+v", err, again)
	}
	if !strings.Contains(c.context(true), "@"+skillListLessons) || !strings.Contains(c.context(true), r.ReceiptID+" @"+skillListLessons+" DONE") {
		t.Fatal("TASS's context lacks the skill or its real receipt")
	}
}

func TestSkillWithoutGrantOrAfterRevokeIsDeniedAndDoesNothing(t *testing.T) {
	ids, _, _, _, c := skillCore(t, false)
	c.Answer(Incoming{Text: "@lesson-workspace.list-lessons please", From: "local"})
	r := lastReceipt(c)
	if r.State != "DENIED" || r.Executed || r.Output != "" || !strings.Contains(r.Error, "Enable this skill") {
		t.Fatalf("no grant must deny at request: %+v", r)
	}
	if got, _ := c.ApproveSkill(r.ReceiptID); got.Executed {
		t.Fatal("approving a denied receipt ran it")
	}
	ids.mu.Lock()
	ids.skillGranted = true
	ids.mu.Unlock()
	r2, _ := c.RequestSkill(skillListLessons, "list", "test", "test")
	ids.mu.Lock()
	ids.skillGranted = false // owner revokes before approving
	ids.mu.Unlock()
	got, _ := c.ApproveSkill(r2.ReceiptID)
	if got.State != "DENIED" || got.Executed || got.Output != "" || !strings.Contains(got.RunDecided, "no grant") {
		t.Fatalf("revoked grant must deny at point of use: %+v", got)
	}
	ids.mu.Lock()
	ids.skillGranted = true
	ids.mu.Unlock()
	r3, _ := c.RequestSkill(skillListLessons, "list", "test", "test")
	ids.set("LOCKED")
	if got, _ := c.ApproveSkill(r3.ReceiptID); got.State != "DENIED" || got.Executed {
		t.Fatalf("locked Identity must deny: %+v", got)
	}
}

func TestSkillReceiptsSurviveRestartAndInterruptedRunIsNeverRepeated(t *testing.T) {
	ids, eng, gh, dir, c := skillCore(t, true)
	r, _ := c.RequestSkill(skillListLessons, "list", "test", "test")
	done, _ := c.ApproveSkill(r.ReceiptID)
	pending, _ := c.RequestSkill(skillListLessons, "list", "test", "test")
	// Simulate a crash in the middle of a run.
	c.skillMu.Lock()
	c.receiptLocked(pending.ReceiptID).State = "RUNNING"
	c.saveSkillsLocked()
	c.skillMu.Unlock()

	c = newLearningCore(t, dir, ids, gh, eng)
	var gotDone, gotInt SkillReceipt
	for _, x := range c.Skills(false).Receipts {
		switch x.ReceiptID {
		case done.ReceiptID:
			gotDone = x
		case pending.ReceiptID:
			gotInt = x
		}
	}
	if gotDone.State != "DONE" || gotDone.OutputSHA256 != done.OutputSHA256 {
		t.Fatalf("done receipt not durable: %+v", gotDone)
	}
	if gotInt.State != "INTERRUPTED" {
		t.Fatalf("crashed run should be INTERRUPTED: %+v", gotInt)
	}
	if again, _ := c.ApproveSkill(pending.ReceiptID); again.State != "INTERRUPTED" || again.Output != "" {
		t.Fatalf("interrupted receipt ran again: %+v", again)
	}
}

func TestUnknownAtNameIsOrdinaryConversationAndModelProposalOnlyRequests(t *testing.T) {
	_, eng, _, _, c := skillCore(t, true)
	c.Answer(Incoming{Text: "hello @somebody", From: "local"})
	if eng.count() != 1 || len(c.Skills(false).Receipts) != 0 {
		t.Fatal("unknown @name should be normal chat with no receipt")
	}
	// A fresh conversation: TASS's own reply proposes the skill. That creates a request only.
	c.NewConversation()
	eng.mu.Lock()
	eng.replyExtra = "I could use @lesson-workspace.list-lessons to check."
	eng.mu.Unlock()
	c.Answer(Incoming{Text: "what lessons are in your workspace?", From: "local"})
	r := lastReceipt(c)
	if r.State != "AWAITING_APPROVAL" || r.Executed || !strings.HasPrefix(r.SelectedBy, "TASS reply (cycle ") {
		t.Fatalf("model proposal should only create a supervised request: %+v", r)
	}
	if h, _ := eng.reqs[1]["history"].([]any); len(eng.reqs) < 2 || len(h) != 0 {
		t.Fatal("NewConversation must start with empty chat history")
	}
	if !strings.Contains(eng.reqs[1]["context"].(string), "SKILLS YOU CAN ASK TO USE") {
		t.Fatal("skills not in TASS's context")
	}
	// GitHub-origin messages never create skill requests.
	before := len(c.Skills(false).Receipts)
	if c.handleSkillChat(Incoming{Text: "Find lessons using @", From: "github"}) || len(c.Skills(false).Receipts) != before {
		t.Fatal("GitHub message created a skill request")
	}
}

func TestListLessonsNeverFollowsSymlinksOutOfTheWorkspace(t *testing.T) {
	_, _, _, dir, c := skillCore(t, true)
	secret := filepath.Join(t.TempDir(), "secret.md")
	os.WriteFile(secret, []byte("# Secret\n"), 0o600)
	os.Symlink(secret, filepath.Join(dir, "lessons", "lessons", "claude", "zz-link.md"))
	r, _ := c.RequestSkill(skillListLessons, "list", "test", "test")
	got, _ := c.ApproveSkill(r.ReceiptID)
	if strings.Contains(got.Output, "zz-link") || strings.Contains(got.Output, "Secret") || got.State != "DONE" {
		t.Fatalf("followed a symlink out of the workspace: %+v", got)
	}
}

func TestResolveSkill(t *testing.T) {
	for _, tc := range []struct{ in, id string }{
		{"Find lessons using @", skillListLessons},
		{"@lesson-workspace.list-lessons", skillListLessons},
		{"please list the lessons with @lesson-workspace.list-lessons.", skillListLessons},
		{"email me@example.com", ""},
		{"@somebody what's up", ""},
		{"Explain @identity-test", ""},
		{"what lessons do you have", ""}, // no @: not a skill request
	} {
		if id, _, _ := resolveSkill(tc.in); id != tc.id {
			t.Errorf("%q → %q, want %q", tc.in, id, tc.id)
		}
	}
}
