package tass

import (
	"strings"
	"sync"
	"testing"
	"time"
)

// Nov 4 P0 owner acceptance: the engine may return a plausible action even
// when its prediction is not LOCKED. The Core must never execute that action.
func TestConversationRefusesUnlockedProposal(t *testing.T) {
	gh, eng := newFakeGH(), newFakeEngine()
	defer gh.srv.Close()
	defer eng.srv.Close()
	gh.set("Arameair/identity@main:policies/tass.json", policyJSON)
	gh.set("Arameair/TASS@live:tass.md", startDoc)
	eng.unlockedPrediction = true
	c := newCore(t, t.TempDir(), gh, eng)
	c.SyncOnce()
	c.Answer(Incoming{Text: "count your sections", From: "local"})
	if len(eng.outcomes) != 1 || eng.outcomes[0]["status"] != "NOT_AUTHORIZED" || eng.outcomes[0]["value"] != nil {
		t.Fatalf("unlocked action must not execute: %+v", eng.outcomes)
	}
}

// Identity must be consulted again at the protected action boundary. An allow
// observed before slow inference is not permission to execute after revocation.
func TestConversationRechecksRevokedGrantAfterInference(t *testing.T) {
	ids, eng := newFakeIdentitySvc(), newFakeEngine()
	defer ids.srv.Close()
	defer eng.srv.Close()
	ids.set("UNLOCKED")

	entered, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	eng.beforeCycleReply = func() {
		once.Do(func() {
			close(entered)
			<-release
		})
	}
	c := newIdentityCore(t, t.TempDir(), ids, eng)
	if err := c.SetSettings(Settings{Autonomous: true}); err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		c.Answer(Incoming{Text: "count your sections", From: "local"})
	}()

	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		close(release)
		t.Fatal("fixture Engine did not receive the cycle")
	}
	ids.mu.Lock()
	ids.revoked = append(ids.revoked, "action.read_entity")
	ids.mu.Unlock()
	close(release)
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Core did not finish after the Engine response was released")
	}

	eng.mu.Lock()
	outcomes := append([]map[string]any(nil), eng.outcomes...)
	eng.mu.Unlock()
	if len(outcomes) != 1 || outcomes[0]["status"] != "NOT_AUTHORIZED" || outcomes[0]["value"] != nil {
		t.Fatalf("revoked action must not execute: %+v", outcomes)
	}
	ids.mu.Lock()
	decisions := append([]string(nil), ids.decisions...)
	ids.mu.Unlock()
	if len(decisions) == 0 || decisions[len(decisions)-1] != "action.read_entity" {
		t.Fatalf("last boundary check must reauthorize the protected action: %v", decisions)
	}
}

// A message accepted while the first cycle is in flight must remain queued,
// then run second with its own correlated cycle ID. Nothing may be dropped,
// reordered, or processed concurrently.
func TestConversationQueuesMessagesInReceiptOrder(t *testing.T) {
	ids, eng := newFakeIdentitySvc(), newFakeEngine()
	defer ids.srv.Close()
	defer eng.srv.Close()
	ids.set("UNLOCKED")

	entered, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	eng.beforeCycleReply = func() {
		once.Do(func() {
			close(entered)
			<-release
		})
	}
	c := newIdentityCore(t, t.TempDir(), ids, eng)
	stop := make(chan struct{})
	defer close(stop)
	go c.think(stop)

	firstReceipt, err := c.Say("first owner message")
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		close(release)
		t.Fatal("first message did not reach the fixture Engine")
	}
	secondReceipt, err := c.Say("second owner message")
	if err != nil {
		close(release)
		t.Fatal(err)
	}
	if firstReceipt == secondReceipt || firstReceipt == "" || secondReceipt == "" {
		close(release)
		t.Fatalf("accepted messages need distinct receipts: %q %q", firstReceipt, secondReceipt)
	}
	if got := len(c.inbox); got != 1 {
		close(release)
		t.Fatalf("second message must wait behind the in-flight cycle, inbox=%d", got)
	}
	close(release)

	deadline := time.Now().Add(5 * time.Second)
	for {
		eng.mu.Lock()
		complete := len(eng.reqs) == 2 && len(eng.outcomes) == 2
		eng.mu.Unlock()
		if complete || time.Now().After(deadline) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	eng.mu.Lock()
	reqs := append([]map[string]any(nil), eng.reqs...)
	outcomes := append([]map[string]any(nil), eng.outcomes...)
	eng.mu.Unlock()
	if len(reqs) != 2 || len(outcomes) != 2 {
		t.Fatalf("wanted two sequential completed cycles: requests=%d outcomes=%d", len(reqs), len(outcomes))
	}
	if reqs[0]["request"] != "first owner message" || reqs[1]["request"] != "second owner message" {
		t.Fatalf("messages lost or reordered: %v then %v", reqs[0]["request"], reqs[1]["request"])
	}
	firstCycle, _ := reqs[0]["cycle_id"].(string)
	secondCycle, _ := reqs[1]["cycle_id"].(string)
	if firstCycle == "" || secondCycle == "" || firstCycle == secondCycle {
		t.Fatalf("cycle IDs must be present and distinct: %q %q", firstCycle, secondCycle)
	}
}

func TestConversationRefusesMismatchedEngineCycle(t *testing.T) {
	gh, eng := newFakeGH(), newFakeEngine()
	defer gh.srv.Close()
	defer eng.srv.Close()
	gh.set("Arameair/identity@main:policies/tass.json", policyJSON)
	gh.set("Arameair/TASS@live:tass.md", startDoc)
	eng.mismatchCycleID = "not-the-requested-cycle"
	c := newCore(t, t.TempDir(), gh, eng)
	c.SyncOnce()
	c.Answer(Incoming{Text: "count your sections", From: "local"})
	if len(eng.outcomes) != 0 || len(c.Status().Cycles) != 0 {
		t.Fatalf("wrong-cycle proposal must neither execute nor enter history: %+v", eng.outcomes)
	}
}

func TestConversationRejectsUnreadableProposalBeforeExecution(t *testing.T) {
	gh, eng := newFakeGH(), newFakeEngine()
	defer gh.srv.Close()
	defer eng.srv.Close()
	gh.set("Arameair/identity@main:policies/tass.json", policyJSON)
	gh.set("Arameair/TASS@live:tass.md", startDoc)
	eng.unreadableProposal = true
	c := newCore(t, t.TempDir(), gh, eng)
	c.SyncOnce()
	c.Answer(Incoming{Text: "count your sections", From: "local"})
	if len(eng.outcomes) != 0 || len(c.Status().Cycles) != 0 {
		t.Fatalf("unreadable proposal must not execute: %+v", eng.outcomes)
	}
}

func TestConversationReadOnlyNoActionStillReplies(t *testing.T) {
	gh, eng := newFakeGH(), newFakeEngine()
	defer gh.srv.Close()
	defer eng.srv.Close()
	gh.set("Arameair/identity@main:policies/tass.json", policyJSON)
	gh.set("Arameair/TASS@live:tass.md", startDoc)
	c := newCore(t, t.TempDir(), gh, eng)
	c.SyncOnce()
	c.Answer(Incoming{Text: "hello", From: "local"})
	if len(eng.outcomes) != 1 || eng.outcomes[0]["status"] != "NO_ACTION" {
		t.Fatalf("normal conversation must not execute an action: %+v", eng.outcomes)
	}
	if len(c.Status().Cycles) != 1 || len(c.history) != 2 ||
		!strings.Contains(c.history[1].Content, "heard hello") {
		t.Fatalf("normal conversation did not display and preserve reply: cycles=%+v history=%+v", c.Status().Cycles, c.history)
	}
}
