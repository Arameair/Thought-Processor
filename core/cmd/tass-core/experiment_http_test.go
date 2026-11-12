package main

// Stage 3B: the experiment-plan routes use the Cognition boundary: page token, same-origin POST, expected
// scope precondition. A refused POST writes nothing and the Engine receives no write.

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExperimentPlanRoutesKeepTheCognitionBoundary(t *testing.T) {
	env := scopeServer(t)
	journal := filepath.Join(env.dir, "experiments.jsonl")
	body := `{"baseline_nudge_id":"nudge_00000000000000000001","variant":"Answer the question directly.","snapshot":"AS_OF_NOW","idempotency_key":"exp-http-0001"}`
	// Stale or other-owner scope → 412, nothing written.
	for who, hdr := range map[string]map[string]string{"owner B's scope": env.expectOf("B"), "a stale link": func() map[string]string {
		h := env.expectOf("A")
		h["X-Cognition-Expect-Link"] = "dec_z0000000"
		return h
	}()} {
		if rec := env.post("/api/development/experiments", body, hdr); rec.Code != http.StatusPreconditionFailed || scopeCode(rec) != "SCOPE_MISMATCH" {
			t.Errorf("POST expecting %s: HTTP %d %q", who, rec.Code, scopeCode(rec))
		}
	}
	// No page token → refused.
	r := env.do("POST", "/api/development/experiments", body, map[string]string{"X-TASS-Token": "wrong"})
	if r.Code < 400 {
		t.Errorf("wrong token: HTTP %d", r.Code)
	}
	// Current scope, but no processed nudge of this owner → 404, nothing written.
	if rec := env.post("/api/development/experiments", body, env.expectOf("A")); rec.Code != http.StatusNotFound {
		t.Errorf("unknown baseline: HTTP %d %s", rec.Code, rec.Body.String())
	}
	// Invalid input is refused before anything is read or written.
	bad := strings.Replace(body, "Answer the question directly.", "NOTE: yes", 1)
	if rec := env.post("/api/development/experiments", bad, env.expectOf("A")); rec.Code != http.StatusBadRequest {
		t.Errorf("NOTE in variant: HTTP %d", rec.Code)
	}
	if _, err := os.Stat(journal); !os.IsNotExist(err) {
		t.Error("a refused POST wrote the experiment journal")
	}
	if len(env.eng.nonGET) != 0 {
		t.Errorf("Engine writes: %v", env.eng.nonGET)
	}
	// Reads: an unknown plan is 404; the list for an unknown run is 404.
	if rec := env.get("/api/development/experiments/exp_00000000000000000000", env.expectOf("A")); rec.Code != http.StatusNotFound {
		t.Errorf("unknown plan: HTTP %d", rec.Code)
	}
	if rec := env.get("/api/development/experiments?run=nudge_00000000000000000001", env.expectOf("A")); rec.Code != http.StatusNotFound {
		t.Errorf("unknown run list: HTTP %d", rec.Code)
	}
	// No route can run an arm.
	for _, p := range []string{"/api/development/experiments/exp_00000000000000000000/arms/A", "/api/development/experiments/exp_00000000000000000000/run"} {
		if rec := env.post(p, `{}`, env.expectOf("A")); rec.Code != http.StatusNotFound && rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s: HTTP %d (an arm route must not exist)", p, rec.Code)
		}
	}
}
