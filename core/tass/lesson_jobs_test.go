package tass

// Regression tests for Thought-Processor #71: a manual Study after Retry, restart, lost
// Core metadata or a changed lesson must never re-submit a job ID the Engine already holds,
// and earlier attempts must stay on record. Fixture Engine/Identity/GitHub, not the owner's
// real Nemotron.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const pulsePath = "lessons/claude/01-my-work-pulse.md"

func setupPulseLesson(t *testing.T) (*fakeIdentitySvc, *fakeEngine, *fakeGH, string, *Core) {
	t.Helper()
	ids, eng, gh := newFakeIdentitySvc(), newFakeEngine(), newFakeGH()
	ids.set("UNLOCKED")
	gh.set("Mira/home@live:tass.md", entityDoc)
	gh.set("Mira/home@live:"+pulsePath, threeCheckPulseLesson)
	dir := t.TempDir()
	c := newLearningCore(t, dir, ids, gh, eng)
	c.SyncOnce()
	if err := c.DiscoverLessons(); err != nil {
		t.Fatal(err)
	}
	return ids, eng, gh, dir, c
}

func acceptedJobs(e *fakeEngine) []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	var out []string
	for _, j := range e.jobs {
		out = append(out, j["job_id"].(string))
	}
	return out
}

// The exact owner state from #71: the first attempt FAILED in the Engine (contract bug),
// the lesson card was reset to NEW / attempt 0 by Retry while still showing the old
// answer, and the data was written by a Core without job_seq/history. Study must use a
// new job ID, keep the old failure, then grade and approve normally.
func TestStudyAfterRetryOfFailedAttemptUsesNewJobAndKeepsHistory(t *testing.T) {
	ids, eng, gh, dir, c := setupPulseLesson(t)
	first, err := c.StudyLesson(pulsePath)
	if err != nil {
		t.Fatal(err)
	}
	eng.finish(first, "FAILED", "plan response unreadable after one retry; first bytes=114 labels=REMEMBER", "")
	c.SyncOnce()
	if ls := lessonAt(c, pulsePath); ls.Status != "MODEL_FAILED" {
		t.Fatalf("want MODEL_FAILED, got %+v", ls)
	}

	// Recreate what the installed alpha Core (3ff2a52) leaves on disk after Retry: NEW,
	// attempts 0, the old job ID and answer still on the card, no job_seq, no history.
	var st map[string]any
	raw, _ := os.ReadFile(filepath.Join(dir, "lessons.json"))
	json.Unmarshal(raw, &st)
	l := st["lessons"].(map[string]any)[pulsePath].(map[string]any)
	l["status"], l["attempts"] = "NEW", 0
	delete(l, "job_seq")
	delete(l, "history")
	raw, _ = json.Marshal(st)
	os.WriteFile(filepath.Join(dir, "lessons.json"), raw, 0o600)

	c = newLearningCore(t, dir, ids, gh, eng) // restart / update
	before := lessonAt(c, pulsePath)
	if before.JobSeq != 1 || len(before.History) != 1 || before.History[0].JobID != first || !strings.Contains(before.History[0].Reply, "REMEMBER") {
		t.Fatalf("migration lost the earlier failed attempt: %+v", before)
	}

	second, err := c.StudyLesson(pulsePath)
	if err != nil {
		t.Fatalf("Study after Retry was refused (the #71 bug): %v", err)
	}
	if second == first || !strings.HasSuffix(second, "_2") || eng.conflicts != 0 {
		t.Fatalf("re-used or skipped job ID: first=%s second=%s conflicts=%d", first, second, eng.conflicts)
	}
	if got := acceptedJobs(eng); len(got) != 2 {
		t.Fatalf("want exactly 2 Engine jobs, got %v", got)
	}
	eng.finish(second, "DONE", "ANSWER 1: 20\nANSWER 2: continue\nANSWER 3: blocked", "")
	c.SyncOnce()
	ls := lessonAt(c, pulsePath)
	if ls.Status != "PASSED" || ls.Right != 3 || ls.LearnedAt != nil {
		t.Fatalf("want PASSED awaiting owner, got %+v", ls)
	}
	if len(ls.History) != 2 || ls.History[0].JobID != first || ls.History[1].JobID != second || ls.History[1].Outcome != "PASSED" {
		t.Fatalf("history not append-only: %+v", ls.History)
	}
	if err := c.ApproveLesson(pulsePath); err != nil {
		t.Fatal(err)
	}
	c = newLearningCore(t, dir, ids, gh, eng) // restart again: learned state and evidence persist
	ls = lessonAt(c, pulsePath)
	if ls.Status != "LEARNED" || len(ls.History) != 2 || !strings.Contains(c.context(true), "My work pulse") {
		t.Fatalf("learned lesson or history not durable across restart: %+v", ls)
	}
}

// Retry no longer resets the job counter, so the next ID is new even with no migration.
func TestRetryKeepsJobCounterAndClearsOnlyTheCurrentCard(t *testing.T) {
	_, eng, _, _, c := setupPulseLesson(t)
	first, _ := c.StudyLesson(pulsePath)
	eng.finish(first, "DONE", "ANSWER 1: 100\nANSWER 2: continue\nANSWER 3: blocked", "")
	c.SyncOnce()
	if err := c.RetryLesson(pulsePath); err != nil {
		t.Fatal(err)
	}
	ls := lessonAt(c, pulsePath)
	if ls.Status != "NEW" || ls.Attempts != 0 || ls.JobSeq != 1 || ls.Reply != "" || len(ls.Results) != 0 || len(ls.History) != 1 || ls.History[0].Outcome != "RETRY" {
		t.Fatalf("Retry should clear the card but keep counter and history: %+v", ls)
	}
	second, err := c.StudyLesson(pulsePath)
	if err != nil || second == first || eng.conflicts != 0 {
		t.Fatalf("second=%s err=%v conflicts=%d", second, err, eng.conflicts)
	}
}

// Core metadata lost entirely (fresh lessons.json) while the Engine still has the old job:
// the 409 is reconciled, the old job is recorded (not graded) and the next ID is used.
func TestEngineConflictAfterLostCoreStateIsReconciledNotRepeated(t *testing.T) {
	ids, eng, gh, dir, c := setupPulseLesson(t)
	first, _ := c.StudyLesson(pulsePath)
	eng.finish(first, "FAILED", "plan response unreadable", "")
	os.Remove(filepath.Join(dir, "lessons.json"))
	c = newLearningCore(t, dir, ids, gh, eng)
	c.SyncOnce()
	if err := c.DiscoverLessons(); err != nil {
		t.Fatal(err)
	}
	if ls := lessonAt(c, pulsePath); ls.Status != "NEW" || ls.JobSeq != 0 {
		t.Fatalf("expected a fresh NEW lesson: %+v", ls)
	}
	second, err := c.StudyLesson(pulsePath)
	if err != nil {
		t.Fatal(err)
	}
	if second == first || eng.conflicts != 1 || len(acceptedJobs(eng)) != 2 {
		t.Fatalf("want one 409, then a new job: second=%s conflicts=%d jobs=%v", second, eng.conflicts, acceptedJobs(eng))
	}
	ls := lessonAt(c, pulsePath)
	if ls.Status != "QUEUED" || ls.JobID != second || len(ls.History) != 1 || ls.History[0].JobID != first ||
		ls.History[0].State != "FAILED" || ls.History[0].Outcome != "NOT_COLLECTED" || ls.Right != 0 {
		t.Fatalf("old job must be kept as ungraded history: %+v", ls)
	}
	// Repeated clicks while queued: no further submissions or 409s.
	for i := 0; i < 3; i++ {
		if id, err := c.StudyLesson(pulsePath); err != nil || id != second {
			t.Fatalf("repeat click %d: %s %v", i, id, err)
		}
	}
	if eng.conflicts != 1 || len(acceptedJobs(eng)) != 2 {
		t.Fatalf("repeat clicks submitted again: conflicts=%d jobs=%v", eng.conflicts, acceptedJobs(eng))
	}
}

// The Engine is still working on the same lesson job the Core forgot: attach, don't resubmit.
func TestEngineConflictWithRunningJobAttaches(t *testing.T) {
	ids, eng, gh, dir, c := setupPulseLesson(t)
	first, _ := c.StudyLesson(pulsePath) // stays QUEUED in the Engine
	os.Remove(filepath.Join(dir, "lessons.json"))
	c = newLearningCore(t, dir, ids, gh, eng)
	c.SyncOnce()
	c.DiscoverLessons()
	id, err := c.StudyLesson(pulsePath)
	if err != nil || id != first || len(acceptedJobs(eng)) != 1 {
		t.Fatalf("should attach to %s: got %s %v jobs=%v", first, id, err, acceptedJobs(eng))
	}
	eng.finish(first, "DONE", "ANSWER 1: 20\nANSWER 2: continue\nANSWER 3: blocked", "")
	c.SyncOnce()
	if ls := lessonAt(c, pulsePath); ls.Status != "PASSED" {
		t.Fatalf("attached job not collected: %+v", ls)
	}
}

// The 409 loop is bounded: no endless resubmission, nothing graded, evidence kept.
func TestEngineConflictLoopIsBounded(t *testing.T) {
	ids, eng, gh, dir, c := setupPulseLesson(t)
	for n := 1; n <= max409Probes; n++ {
		id := lessonJobID(pulsePath, lessonAt(c, pulsePath).SHA, n)
		eng.mu.Lock()
		eng.jobs = append(eng.jobs, map[string]any{"job_id": id})
		eng.mu.Unlock()
		eng.finish(id, "FAILED", "old", "")
	}
	_ = dir
	_ = ids
	_ = gh
	if _, err := c.StudyLesson(pulsePath); err == nil {
		t.Fatal("expected a bounded error")
	}
	ls := lessonAt(c, pulsePath)
	if eng.conflicts != max409Probes || len(acceptedJobs(eng)) != max409Probes || ls.Status != "NEW" || len(ls.History) != max409Probes || ls.JobSeq != max409Probes {
		t.Fatalf("conflicts=%d jobs=%d lesson=%+v", eng.conflicts, len(acceptedJobs(eng)), ls)
	}
	// The next click starts after the probed numbers, not from 1 again.
	if id, err := c.StudyLesson(pulsePath); err != nil || !strings.HasSuffix(id, "_9") {
		t.Fatalf("next study: %s %v", id, err)
	}
}

// A lesson changed on GitHub gets a new revision (new job prefix) but keeps its history,
// and an already LEARNED lesson (the owner's 02/03) is never touched by another lesson's work.
func TestChangedLessonKeepsHistoryAndOtherLearnedLessonUntouched(t *testing.T) {
	ids, eng, gh, dir, c := setupPulseLesson(t)
	other := "lessons/claude/02-how-i-learn.md"
	gh.set("Mira/home@live:"+other, threeCheckPulseLesson)
	c.DiscoverLessons()
	o, _ := c.StudyLesson(other)
	eng.finish(o, "DONE", "ANSWER 1: 20\nANSWER 2: continue\nANSWER 3: blocked", "")
	c.SyncOnce()
	if err := c.ApproveLesson(other); err != nil {
		t.Fatal(err)
	}
	learned := lessonAt(c, other)

	first, _ := c.StudyLesson(pulsePath)
	eng.finish(first, "FAILED", "bad", "")
	c.SyncOnce()
	gh.set("Mira/home@live:"+pulsePath, threeCheckPulseLesson+"\n<!-- edited -->\n")
	c.DiscoverLessons()
	ls := lessonAt(c, pulsePath)
	if ls.Status != "NEW" || len(ls.History) != 1 || ls.History[0].JobID != first {
		t.Fatalf("changed lesson lost history: %+v", ls)
	}
	second, err := c.StudyLesson(pulsePath)
	if err != nil || strings.HasPrefix(second, lessonJobPrefix(pulsePath, ls.History[0].SHA)) || eng.conflicts != 0 {
		t.Fatalf("new revision should use its own job prefix: %s %v", second, err)
	}
	c = newLearningCore(t, dir, ids, gh, eng)
	if got := lessonAt(c, other); got.Status != "LEARNED" || got.LearnedAt == nil || !got.LearnedAt.Equal(*learned.LearnedAt) {
		t.Fatalf("approved lesson changed: %+v", got)
	}
	if doc, _ := c.read("tass.md"); strings.Count(doc, "(lesson `"+other+"`,") != 1 {
		t.Fatalf("Learned line duplicated or lost:\n%s", doc)
	}
}

// Identity locked: Study does nothing at all, even with history and a counter on disk.
func TestStudyDeniedByIdentityTouchesNothing(t *testing.T) {
	ids, eng, _, _, c := setupPulseLesson(t)
	first, _ := c.StudyLesson(pulsePath)
	eng.finish(first, "FAILED", "bad", "")
	c.SyncOnce()
	ids.set("LOCKED")
	n := len(acceptedJobs(eng))
	if _, err := c.StudyLesson(pulsePath); err == nil {
		t.Fatal("LOCKED Identity allowed Study")
	}
	if len(acceptedJobs(eng)) != n || eng.conflicts != 0 || lessonAt(c, pulsePath).Status != "MODEL_FAILED" {
		t.Fatal("denied Study changed Engine or lesson state")
	}
}
