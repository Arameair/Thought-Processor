package tass

import (
	"strings"
	"testing"
)

const retestLesson = lessonPulse + `
## Retest 1: Applying the timing
Check: If the pulse has ticked three times, about how many seconds have passed?
Answer: 60
`

// Gate F step 5: a LEARNED lesson is retested on NEW questions with only approved knowledge in
// context (never the lesson text, never the hidden retest answers); graded by Core; the lesson's
// status, approval and skills are unchanged; evidence persists across restart.
func TestFreshTaskRetestAfterApproval(t *testing.T) {
	ids, eng, gh := newFakeIdentitySvc(), newFakeEngine(), newFakeGH()
	ids.set("UNLOCKED")
	p := "lessons/claude/01-pulse.md"
	gh.set("Mira/home@live:tass.md", entityDoc)
	gh.set("Mira/home@live:"+p, retestLesson)
	dir := t.TempDir()
	c := newLearningCore(t, dir, ids, gh, eng)
	c.SyncOnce()
	if err := c.DiscoverLessons(); err != nil {
		t.Fatal(err)
	}
	if _, err := c.RetestLesson(p, nil); err == nil || !strings.Contains(err.Error(), "LEARNED") {
		t.Fatalf("retest before approval must be refused: %v", err)
	}
	id, err := c.StudyLesson(p)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(eng.jobs[0]["context"].(string), "three times") || strings.Contains(eng.jobs[0]["request"].(string), "three times") {
		t.Fatal("the hidden retest question leaked into study")
	}
	eng.finish(id, "DONE", "ANSWER 1: 20\nANSWER 2: Identity", "my pulse ticks every 20 seconds")
	c.SyncOnce()
	if _, err := c.RetestLesson(p, nil); err == nil {
		t.Fatal("PASSED is not LEARNED: retest must wait for owner approval")
	}
	if err := c.ApproveLesson(p); err != nil {
		t.Fatal(err)
	}
	// the study quiz itself is not a fresh task
	if _, err := c.RetestLesson(p, []RetestQuestion{{N: 1, Ask: "How many seconds between two ticks?", Answer: "20"}}); err == nil || !strings.Contains(err.Error(), "ask something new") {
		t.Fatalf("re-asking the studied Check is not a fresh task: %v", err)
	}
	rid, err := c.RetestLesson(p, nil) // the lesson's own ## Retest section
	if err != nil || !strings.HasPrefix(rid, "retest_") {
		t.Fatalf("retest: %q %v", rid, err)
	}
	job := eng.jobs[len(eng.jobs)-1]
	ctx, req := job["context"].(string), job["request"].(string)
	if strings.Contains(ctx, "Teach:") || strings.Contains(ctx, "One tick every 20 seconds") || strings.Contains(ctx+req, "Answer: 60") || strings.Contains(ctx+req, "60") {
		t.Fatalf("retest must not see the lesson text or the expected answer:\n%s\n%s", ctx, req)
	}
	if !strings.Contains(ctx, "WHAT I HAVE LEARNED") || !strings.Contains(req, "Check 1: If the pulse has ticked three times") {
		t.Fatalf("retest context/request: %s / %s", ctx, req)
	}
	if a, ok := job["actions"].([]any); ok && len(a) > 0 {
		t.Fatal("a retest offers no actions")
	}
	if again, _ := c.RetestLesson(p, nil); again != rid {
		t.Fatalf("repeat click must reuse the queued retest: %s != %s", again, rid)
	}
	eng.finish(rid, "DONE", "ANSWER 1: about 60 seconds", "")
	c.SyncOnce()
	l := lessonAt(c, p)
	if l.Status != "LEARNED" || l.RetestPending != nil || len(l.Retests) != 1 {
		t.Fatalf("retest must be recorded without changing status: %+v", l)
	}
	r := l.Retests[0]
	if r.Outcome != "RETEST_PASSED" || r.Right != 1 || r.Checks != 1 || r.JobID != rid || r.SHA != l.SHA || r.GradedAt == nil {
		t.Fatalf("retest record: %+v", r)
	}
	// owner-typed fresh question, answered wrongly: RETEST_FAILED, still LEARNED
	rid2, err := c.RetestLesson(p, []RetestQuestion{{N: 1, Ask: "Who decides whether my work is allowed?", Answer: "Identity"}})
	if err != nil || rid2 == rid {
		t.Fatalf("second retest needs a new job id: %q %v", rid2, err)
	}
	eng.finish(rid2, "DONE", "ANSWER 1: the model", "")
	c.SyncOnce()
	// restart: evidence persists
	c2 := newLearningCore(t, dir, ids, gh, eng)
	l = lessonAt(c2, p)
	if l.Status != "LEARNED" || len(l.Retests) != 2 || l.Retests[1].Outcome != "RETEST_FAILED" || l.Retests[1].Questions[0].From != "owner" {
		t.Fatalf("after restart: %+v", l.Retests)
	}
	// Identity LOCKED: no retest
	ids.set("LOCKED")
	if _, err := c2.RetestLesson(p, nil); err == nil {
		t.Fatal("LOCKED Identity must refuse a retest")
	}
}
