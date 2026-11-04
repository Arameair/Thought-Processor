package tass

import (
	"strings"
	"testing"
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
