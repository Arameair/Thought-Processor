package tass

// Focused repair (Codex audit on PR #113): an out-of-order Engine answer must not lock nudge processing.
//
// Core sends the nudge POST and, at the same time, reads the run's status. The status read can be
// answered "404 / NOT_CLAIMED" before the Engine claims the nudge and ARRIVE after the POST finished
// and Core recorded what it observed. That old answer says nothing about the Engine's data now.
// The recovery lock is for the opposite case, which must keep working: Core saw the Engine claim a
// nudge, and a LATER read finds the Engine no longer has it.

import (
	"errors"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

// opener returns a function that closes ch exactly once; it also runs at cleanup, so a failed
// assertion never leaves the stand-in Engine waiting (its server could then not shut down).
func opener(t *testing.T, ch chan struct{}) func() {
	var once sync.Once
	open := func() { once.Do(func() { close(ch) }) }
	t.Cleanup(open)
	return open
}

func waitFile(t *testing.T, path string) {
	t.Helper()
	for i := 0; i < 500; i++ {
		if _, err := os.Stat(path); err == nil {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("%s was never written", path)
}

// The exact order from the audit, forced: status GET decides NOT_CLAIMED → POST claims and completes →
// Core records observed.json and outcome.json → the old GET answer is delivered.
func TestStalePreClaimReadDoesNotLockCompletedNudge(t *testing.T) {
	dir := t.TempDir()
	c, ids, re := runCore(t, dir)
	d := selfDraft(c, t, "key-stale-read-0001")

	decided, release := make(chan bool, 1), make(chan struct{})
	var once sync.Once
	re.mu.Lock()
	re.preClaim = make(chan struct{})
	re.statusHook = func(nid string, claimed bool) {
		held := false
		once.Do(func() { held = true }) // only the FIRST status read is held
		if held {
			decided <- claimed
			<-release
		}
	}
	openClaim, deliver := opener(t, re.preClaim), opener(t, release)
	re.mu.Unlock()

	type result struct {
		v       *RunView
		created bool
		err     error
	}
	done := make(chan result, 1)
	go func() {
		v, created, err := c.ProcessNudge(d.NudgeID, d.BodyDigest)
		done <- result{v, created, err}
	}()

	select {
	case claimed := <-decided:
		if claimed {
			t.Fatal("setup: the held status read must have decided NOT_CLAIMED")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("setup: the status read never reached the Engine")
	}
	openClaim() // now the Engine claims, runs and answers the POST
	waitFile(t, c.runPath(d.NudgeID, "outcome.json"))
	if _, err := os.Stat(c.runPath(d.NudgeID, "observed.json")); err != nil {
		t.Fatalf("setup: observed.json must exist before the old answer arrives: %v", err)
	}
	deliver() // the pre-claim answer arrives last

	var r result
	select {
	case r = <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("ProcessNudge never returned")
	}
	if r.err != nil || !r.created || r.v == nil {
		t.Fatalf("process: %+v created=%v err=%v", r.v, r.created, r.err)
	}
	if r.v.State != "COMPLETED" || !r.v.Verified {
		t.Fatalf("the stale answer decided the run's state: %+v", r.v)
	}
	if why := c.processingLock(); why != "" {
		t.Fatalf("a stale pre-claim answer locked nudge processing: %s", why)
	}
	// The recorded nudge reads COMPLETED, and the lock is not set by a later read either.
	list, err := c.NudgeHistory("")
	if err != nil || len(list.Nudges) != 1 || list.Nudges[0].Run == nil || list.Nudges[0].Run.State != "COMPLETED" || list.Locked != "" {
		t.Fatalf("history after the stale answer: %+v %v", list, err)
	}

	// Revoked replay is the authorization denial, not a recovery lock.
	ids.mu.Lock()
	ids.processGranted = false
	ids.mu.Unlock()
	if _, _, err := c.ProcessNudge(d.NudgeID, d.BodyDigest); !errors.Is(err, ErrProcessNotGranted) {
		t.Fatalf("revoked replay: %v", err)
	}
	ids.mu.Lock()
	ids.processGranted = true
	ids.mu.Unlock()

	// Processing is still available: a second nudge is accepted and completes. One POST per nudge.
	d2 := selfDraft(c, t, "key-stale-read-0002")
	if _, created, err := c.ProcessNudge(d2.NudgeID, d2.BodyDigest); err != nil || !created {
		t.Fatalf("second nudge after the stale answer: created=%v err=%v", created, err)
	}
	waitRun(t, c, d2.NudgeID, "COMPLETED")
	// The same Process again returns the same run and sends nothing.
	if v, created, err := c.ProcessNudge(d.NudgeID, d.BodyDigest); err != nil || created || v.State != "COMPLETED" {
		t.Fatalf("replay: %+v created=%v err=%v", v, created, err)
	}
	re.mu.Lock()
	posts, completed := re.posts, len(re.completed)
	re.mu.Unlock()
	if posts != 2 || completed != 2 {
		t.Fatalf("exactly one Engine POST and one model run per nudge: posts=%d completed=%d", posts, completed)
	}

	// Durable receipts and restart: a new Core on the same data reads both runs and sends nothing.
	for _, name := range []string{"submitted.json", "observed.json", "outcome.json"} {
		if _, err := os.Stat(c.runPath(d.NudgeID, name)); err != nil {
			t.Fatalf("receipt %s: %v", name, err)
		}
	}
	c.StopNudgeDispatch()
	c2 := nudgeCoreReopen(t, dir, c)
	list, err = c2.NudgeHistory("")
	if err != nil || len(list.Nudges) != 2 || list.Locked != "" {
		t.Fatalf("after restart: %+v %v", list, err)
	}
	for _, n := range list.Nudges {
		if n.Run == nil || n.Run.State != "COMPLETED" || !n.Run.Verified {
			t.Fatalf("after restart: %s %+v", n.NudgeID, n.Run)
		}
	}
	re.mu.Lock()
	posts = re.posts
	re.mu.Unlock()
	if posts != 2 {
		t.Fatalf("restart re-sent a nudge: posts=%d", posts)
	}
}

// The same held answer when the POST has only been CLAIMED (observed by a newer read), not finished:
// the stale answer must not lock, and the run must not be reported as unclaimed.
func TestStalePreClaimReadWhileRunningDoesNotLock(t *testing.T) {
	c, _, re := runCore(t, t.TempDir())
	d := selfDraft(c, t, "key-stale-read-0003")
	decided, release := make(chan bool, 1), make(chan struct{})
	var once sync.Once
	re.mu.Lock()
	re.preClaim, re.gate = make(chan struct{}), make(chan struct{})
	re.statusHook = func(nid string, claimed bool) {
		held := false
		once.Do(func() { held = true })
		if held {
			decided <- claimed
			<-release
		}
	}
	openClaim, finish, deliver := opener(t, re.preClaim), opener(t, re.gate), opener(t, release)
	re.mu.Unlock()

	done := make(chan *RunView, 1)
	go func() {
		v, _, err := c.ProcessNudge(d.NudgeID, d.BodyDigest)
		if err != nil {
			t.Errorf("process: %v", err)
		}
		done <- v
	}()
	select {
	case claimed := <-decided:
		if claimed {
			t.Fatal("setup: the held status read must have decided NOT_CLAIMED")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("setup: the status read never reached the Engine")
	}
	openClaim() // the Engine claims; the POST stays open (RUNNING)
	// A newer read (not held) sees RUNNING and records observed.json.
	var seen *RunView
	for i := 0; i < 500; i++ {
		if seen = c.runView(d.NudgeID); seen != nil && seen.State == "RUNNING" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if seen == nil || seen.State != "RUNNING" {
		t.Fatalf("setup: a newer read must see RUNNING: %+v", seen)
	}
	waitFile(t, c.runPath(d.NudgeID, "observed.json"))
	deliver()
	var v *RunView
	select {
	case v = <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("ProcessNudge never returned")
	}
	if v == nil || v.State != "RUNNING" {
		t.Fatalf("the stale answer decided the run's state: %+v", v)
	}
	if why := c.processingLock(); why != "" {
		t.Fatalf("a stale pre-claim answer locked nudge processing: %s", why)
	}
	finish()
	waitRun(t, c, d.NudgeID, "COMPLETED")
	re.mu.Lock()
	posts := re.posts
	re.mu.Unlock()
	if posts != 1 || c.processingLock() != "" {
		t.Fatalf("posts=%d lock=%q", posts, c.processingLock())
	}
}

// The protection itself: Core observed the Engine's claim, and a LATER read finds the Engine has no
// such run (for example Engine data restored without Core's). Processing locks for every nudge.
func TestGenuineEngineDivergenceStillLocksProcessing(t *testing.T) {
	c, _, re := runCore(t, t.TempDir())
	re.mu.Lock()
	re.gate = make(chan struct{})
	finish := opener(t, re.gate)
	re.mu.Unlock()
	d := selfDraft(c, t, "key-divergence-0001")
	if _, created, err := c.ProcessNudge(d.NudgeID, d.BodyDigest); err != nil || !created {
		t.Fatalf("process: created=%v err=%v", created, err)
	}
	waitRun(t, c, d.NudgeID, "RUNNING") // Core read the claim and recorded observed.json
	waitFile(t, c.runPath(d.NudgeID, "observed.json"))
	if why := c.processingLock(); why != "" {
		t.Fatalf("locked before any divergence: %s", why)
	}

	re.mu.Lock()
	delete(re.runs, d.NudgeID) // the Engine no longer has the run Core saw it claim
	re.mu.Unlock()

	v := c.runView(d.NudgeID)
	if v == nil || v.State != "RECOVERY_REQUIRED" || v.CanStop {
		t.Fatalf("a run the Engine lost must need recovery: %+v", v)
	}
	if why := c.processingLock(); !strings.Contains(why, d.NudgeID) {
		t.Fatalf("genuine divergence must lock processing: %q", why)
	}
	list, err := c.NudgeHistory("")
	if err != nil || list.Locked == "" {
		t.Fatalf("history must report the lock: %+v %v", list, err)
	}
	// Every nudge is refused while locked, before any Engine request.
	d2 := selfDraft(c, t, "key-divergence-0002")
	if _, _, err := c.ProcessNudge(d2.NudgeID, d2.BodyDigest); !errors.Is(err, ErrProcessLocked) {
		t.Fatalf("processing while diverged: %v", err)
	}
	if _, _, err := c.ProcessNudge(d.NudgeID, d.BodyDigest); !errors.Is(err, ErrProcessLocked) {
		t.Fatalf("replay while diverged: %v", err)
	}
	re.mu.Lock()
	posts := re.posts
	re.mu.Unlock()
	if posts != 1 {
		t.Fatalf("a locked Core sent a nudge: posts=%d", posts)
	}
	finish()
	c.StopNudgeDispatch()
	if c.processingLock() == "" {
		t.Fatal("the lock must stay until the owner recovers")
	}
}
