package tass

// Stage 2 (Expanded Thought): the page's action line maps Core's receipt codes to words. This pins the
// codes nudgeReceipt() actually emits, from real outcome receipts on disk, so the page and Core agree.
// Test only: nudgeReceipt is unchanged.

import (
	"os"
	"testing"
	"time"

	"github.com/Arameair/Thought-Processor/core/engine"
)

func TestCognitionNudgeReceiptVocabulary(t *testing.T) {
	c, _, _ := runCore(t, t.TempDir())
	id := func(i int) string { return "nudge_" + string(rune('a'+i)) + "0000000000000000000" }
	mine := map[string]bool{}
	write := func(i int, run engine.NudgeRun, verified bool) string {
		n := id(i)
		mine[n] = true
		if ok, err := coreCreateOnce(c.runPath(n, "outcome.json"), coreOutcome{Schema: nudgeRunSchemaCore, Run: run, Verified: verified, At: time.Now().UTC()}); !ok || err != nil {
			t.Fatalf("write receipt %s: %v %v", n, ok, err)
		}
		return n
	}
	cases := []struct {
		name, nid, req, want string
	}{
		{"verified, same request", write(0, engine.NudgeRun{State: "COMPLETED", RequestID: "mr_0001"}, true), "mr_0001", "COMPLETED_VERIFIED"},
		{"verified, other request", write(1, engine.NudgeRun{State: "COMPLETED", RequestID: "mr_0001"}, true), "mr_9999", "COMPLETED_NOT_VERIFIED"},
		{"verified, no request ID on the Note", write(2, engine.NudgeRun{State: "COMPLETED", RequestID: ""}, true), "", "COMPLETED_NOT_VERIFIED"},
		{"completed, not verified", write(3, engine.NudgeRun{State: "COMPLETED", RequestID: "mr_0003"}, false), "mr_0003", "COMPLETED_NOT_VERIFIED"},
		{"failed", write(4, engine.NudgeRun{State: "FAILED"}, false), "mr_0004", "FAILED"},
		{"uncertain", write(5, engine.NudgeRun{State: "UNCERTAIN"}, false), "mr_0005", "UNCERTAIN"},
		{"not delivered", write(6, engine.NudgeRun{State: "NOT_DELIVERED"}, false), "mr_0006", "NOT_DELIVERED"},
		{"recovery", write(7, engine.NudgeRun{State: "RECOVERY_REQUIRED"}, false), "mr_0007", "RECOVERY_REQUIRED"},
	}
	// No receipt yet (submitted, running, or never recorded).
	mine[id(8)] = true
	cases = append(cases, struct{ name, nid, req, want string }{"no receipt", id(8), "mr_0008", "NO_CORE_RECEIPT"})
	// An unreadable receipt.
	mine[id(9)] = true
	if err := os.MkdirAll(c.runPath(id(9), ""), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(c.runPath(id(9), "outcome.json"), []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	cases = append(cases, struct{ name, nid, req, want string }{"unreadable receipt", id(9), "mr_0009", "RECOVERY_REQUIRED"})
	// A nudge that is not in this owner's journal: its receipt is never read, even if one exists.
	foreign := write(10, engine.NudgeRun{State: "COMPLETED", RequestID: "mr_0010"}, true)
	delete(mine, foreign)
	cases = append(cases, struct{ name, nid, req, want string }{"foreign", foreign, "mr_0010", "NOT_IN_THIS_OWNERS_JOURNAL"})

	// The page treats exactly these as known; anything else is shown as uncertain.
	known := map[string]bool{"COMPLETED_VERIFIED": true, "COMPLETED_NOT_VERIFIED": true, "NO_CORE_RECEIPT": true, "NOT_IN_THIS_OWNERS_JOURNAL": true,
		"RECOVERY_REQUIRED": true, "UNCERTAIN": true, "FAILED": true, "NOT_DELIVERED": true}
	for _, tc := range cases {
		got := c.nudgeReceipt(tc.nid, tc.req, mine)
		if got != tc.want {
			t.Errorf("%s: nudgeReceipt = %q, want %q", tc.name, got, tc.want)
		}
		if !known[got] {
			t.Errorf("%s: %q is not in the vocabulary the page maps", tc.name, got)
		}
	}
	// Only a COMPLETED, Core-verified run with the SAME non-empty request ID is COMPLETED_VERIFIED.
	for _, tc := range cases {
		if (tc.want == "COMPLETED_VERIFIED") != (tc.name == "verified, same request") {
			t.Errorf("%s: only one case may be COMPLETED_VERIFIED", tc.name)
		}
	}
}
