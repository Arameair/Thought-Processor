package tass

// In this experimental one-app build, the local Identity-backed user may
// converse with the existing model but may not grant the Core an action or
// autonomous work pulse while Issue #50's integrity gates remain unresolved.
import (
	"os"
	"strings"
	"testing"
)

func TestChatOnlyRejectsEvenMaliciousEngineActionAndPulse(t *testing.T) {
	gh, eng := newFakeGH(), newFakeEngine()
	defer gh.srv.Close()
	defer eng.srv.Close()
	gh.set("Arameair/identity@main:policies/tass.json", policyJSON)
	gh.set("Arameair/TASS@live:tass.md", startDoc)
	c := newCore(t, t.TempDir(), gh, eng)
	c.cfg.ChatOnly = true
	c.SyncOnce()

	// Fake Engine deliberately proposes tass_sections despite no allowed
	// actions. Even a valid prototype grant cannot bypass this safety gate.
	c.Answer(Incoming{Text: "How many sections are there?", From: "local"})
	eng.mu.Lock()
	if len(eng.reqs) != 1 || len(eng.outcomes) != 1 {
		eng.mu.Unlock()
		t.Fatalf("expected one conversational cycle and result")
	}
	req, outcome := eng.reqs[0], eng.outcomes[0]
	if xs := req["actions"]; xs != nil {
		if list, ok := xs.([]any); !ok || len(list) != 0 {
			eng.mu.Unlock()
			t.Fatalf("chat-only offered executable actions: %v", xs)
		}
	}
	if outcome["status"] != "NOT_AUTHORIZED" || outcome["value"] != nil ||
		!strings.Contains(outcome["detail"].(string), "CHAT_ONLY") {
		eng.mu.Unlock()
		t.Fatalf("chat-only allowed an Engine action: %v", outcome)
	}
	before := len(eng.pulse)
	eng.mu.Unlock()
	if _, err := c.SubmitTask("count your sections"); err == nil ||
		!strings.Contains(err.Error(), "CHAT_ONLY") {
		t.Fatalf("task was accepted while chat-only: %v", err)
	}
	for _, act := range []string{"start", "resume", "step"} {
		if _, err := c.PulseControl(act); err == nil ||
			!strings.Contains(err.Error(), "CHAT_ONLY") {
			t.Fatalf("pulse %s was accepted: %v", act, err)
		}
	}
	eng.mu.Lock()
	if len(eng.jobs) != 0 || len(eng.pulse) != before {
		eng.mu.Unlock()
		t.Fatal("a blocked task or pulse reached the Engine")
	}
	eng.mu.Unlock()
	if _, err := c.PulseControl("pause"); err != nil {
		t.Fatalf("emergency pause should remain available: %v", err)
	}
	// The fake model can also supply NOW:/REMEMBER: directives. In chat-only
	// mode these are treated as untrusted text, not authorization to write.
	c.Answer(Incoming{Text: "remember how the model acted", From: "local"})
	if got := memories(t, c.cfg.DataDir); got != "" {
		t.Fatalf("model-directed memory changed in chat-only: %s", got)
	}
	if _, err := os.Stat(c.file("now.txt")); !os.IsNotExist(err) {
		t.Fatalf("model-directed now.txt write occurred in chat-only: %v", err)
	}
	events := c.Status().Events
	seenReply := false
	for _, e := range events {
		if e.Kind == "tass" && strings.Contains(e.Text, "TEST FIXTURE: heard") {
			seenReply = true
		}
	}
	if !seenReply {
		t.Fatalf("chat-only blocked normal model reply along with actions: %v", events)
	}
}
