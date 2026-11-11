package tass

// Stage 1 slice 2 (WP1-C): the activity strip may say WHY the Engine pulse is BLOCKED, but only what
// the Engine recorded, and only as a fixed code. The Engine's own text names identities and stays
// in the Engine.

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestCognitionPulseBlockedReasonIsRecordedAndCoded(t *testing.T) {
	f := newCogEngine(t)
	_, c := cogCore(t, f)
	stamp := time.Date(2026, 11, 11, 15, 0, 0, 0, time.UTC)
	tick := func(state, authorization string) map[string]any {
		m := map[string]any{"pulse_id": "pulse_000070_150000", "at": stamp, "trigger": "timer", "state": state, "job_id": "job_7",
			"decision": "WAIT: job_7 stays queued"}
		if authorization != "" {
			m["authorization"] = authorization
		}
		return m
	}
	set := func(state string, ticks ...any) {
		f.mu.Lock()
		f.pulse = map[string]any{"control": "RUNNING", "state": state, "every_s": 20.0, "active_job": "", "ticks": ticks,
			"jobs": []any{map[string]any{"state": "QUEUED"}}}
		f.mu.Unlock()
	}
	read := func() (CognitionPulse, string) {
		p, err := c.CognitionPulse()
		if err != nil {
			t.Fatalf("pulse read: %v", err)
		}
		b, _ := json.Marshal(p)
		return p, string(b)
	}

	// The Engine recorded a denial: the code says so; the Engine's sentence (which names an identity) does not cross.
	set("BLOCKED", tick("BLOCKED", "DENIED: Identity doesn't allow pulse.run for OTHER-ENTITY-NAME"))
	p, raw := read()
	if p.State != "BLOCKED" || p.BlockedReason != "AUTHORIZATION_DENIED" {
		t.Fatalf("blocked with a recorded denial: state=%q reason=%q", p.State, p.BlockedReason)
	}
	if strings.Contains(raw, "OTHER-ENTITY-NAME") || strings.Contains(raw, "stays queued") {
		t.Fatalf("pulse projection leaked the Engine's free text: %s", raw)
	}

	// BLOCKED with nothing recorded about why: the strip must not invent a reason.
	set("BLOCKED", tick("BLOCKED", ""))
	if p, _ := read(); p.BlockedReason != "NOT RECORDED" {
		t.Fatalf("blocked without a recorded reason: %q", p.BlockedReason)
	}
	// BLOCKED with an authorization text this Core does not recognize: also not invented, not passed on.
	set("BLOCKED", tick("BLOCKED", "something else PRIVATE-TEXT"))
	if p, raw := read(); p.BlockedReason != "NOT RECORDED" || strings.Contains(raw, "PRIVATE-TEXT") {
		t.Fatalf("blocked with unrecognized text: %q %s", p.BlockedReason, raw)
	}
	// The newest tick is not the blocked one (state says BLOCKED, the record does not): not invented.
	set("BLOCKED", tick("READY", "DENIED: old"))
	if p, _ := read(); p.BlockedReason != "NOT RECORDED" {
		t.Fatalf("blocked state without a blocked newest tick: %q", p.BlockedReason)
	}
	// Any other state carries no reason at all, even when a tick has a denial text.
	for _, state := range []string{"READY", "WORKING", "BUSY"} {
		set(state, tick(state, "DENIED: not relevant"))
		if p, raw := read(); p.BlockedReason != "" || strings.Contains(raw, "blocked_reason") {
			t.Fatalf("%s must carry no blocked reason: %q %s", state, p.BlockedReason, raw)
		}
	}
	// PAUSED never reports a blocked reason: nothing is trying to run.
	f.mu.Lock()
	f.pulse = map[string]any{"control": "PAUSED", "state": "BLOCKED", "every_s": 20.0, "active_job": "",
		"ticks": []any{tick("BLOCKED", "DENIED: x")}, "jobs": []any{}}
	f.mu.Unlock()
	if p, _ := read(); p.Control != "PAUSED" || p.BlockedReason != "" {
		t.Fatalf("paused pulse must carry no blocked reason: %+v", p)
	}
	f.mu.Lock()
	for _, call := range f.calls {
		if !strings.HasPrefix(call, "GET ") {
			t.Fatalf("pulse read sent %s", call)
		}
	}
	f.mu.Unlock()
}
