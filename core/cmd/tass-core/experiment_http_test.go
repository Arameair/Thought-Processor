package main

// Stage 3B: the experiment-plan routes use the Cognition boundary: page token, same-origin POST, expected
// scope precondition. A refused POST writes nothing and the Engine receives no write.

import (
	"bytes"
	"fmt"
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
	// The arm route (Stage 3B runtime) keeps the same boundary: stale scope 412, wrong token refused, an
	// unknown plan or arm 404, GET not allowed; none of them reaches the Engine with a write.
	armPath := "/api/development/experiments/exp_00000000000000000000/arms/A"
	confirm := `{"body_digest":"x","confirm":"RUN A exp_00000000000000000000"}`
	if rec := env.post(armPath, confirm, env.expectOf("B")); rec.Code != http.StatusPreconditionFailed {
		t.Errorf("arm with owner B's scope: HTTP %d", rec.Code)
	}
	if r := env.do("POST", armPath, confirm, map[string]string{"X-TASS-Token": "wrong"}); r.Code < 400 {
		t.Errorf("arm with a wrong token: HTTP %d", r.Code)
	}
	for _, p := range []string{armPath, "/api/development/experiments/exp_00000000000000000000/arms/C", "/api/development/experiments/exp_../arms/A"} {
		if rec := env.post(p, confirm, env.expectOf("A")); rec.Code != http.StatusNotFound {
			t.Errorf("%s: HTTP %d %s", p, rec.Code, rec.Body.String())
		}
	}
	if rec := env.get(armPath, env.expectOf("A")); rec.Code != http.StatusMethodNotAllowed && rec.Code != http.StatusNotFound {
		t.Errorf("GET arm: HTTP %d", rec.Code)
	}
	if rec := env.post("/api/development/experiments/exp_00000000000000000000/run", `{}`, env.expectOf("A")); rec.Code != http.StatusNotFound && rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("run-all route: HTTP %d (there must be no route that runs both arms)", rec.Code)
	}
	if len(env.eng.nonGET) != 0 {
		t.Errorf("Engine writes: %v", env.eng.nonGET)
	}
}

// Stage 3B slice-1 repair (Codex audit, PR #122): an unsafe journal refuses every experiment route with one
// fixed sentence: no saved plan in the answer, no attestation header, and the journal left byte-identical.
func TestExperimentRoutesFailClosedOnAnUnsafeJournal(t *testing.T) {
	const secret = "PRIVATE-HTTP-PLAN-TEXT"
	a := scopeOwners["A"]
	rec := func(id string) []byte {
		return []byte(fmt.Sprintf(`{"schema":"thought-processor.experiment-plan/1","experiment_id":"%s","owner_entity_id":"%s","engine_entity_id":"%s","thought_id":"thought_1","baseline_nudge_id":"nudge_00000000000000000001","baseline_request_id":"mr_00000001","snapshot":"AS_OF_NOW","variable":"instruction","control":"c","variant":"%s","idempotency_key":"exp-http-seed","body_digest":"x"}`+"\n", id, a.id, a.engine, secret))
	}
	valid := rec("exp_aaaaaaaaaaaaaaaaaaaa")
	post := `{"baseline_nudge_id":"nudge_00000000000000000001","variant":"Answer the question directly.","snapshot":"AS_OF_NOW","idempotency_key":"exp-http-0002"}`
	big := append([]byte{}, valid...)
	for i := 0; len(big) <= 4<<20; i++ {
		big = append(big, rec(fmt.Sprintf("exp_%020x", i))...)
	}
	for _, tc := range []struct {
		name string
		data []byte
		mode os.FileMode
	}{
		{"complete corrupt record", append(append([]byte{}, valid...), []byte("{broken}\n")...), 0o600},
		{"over the 4 MiB ceiling", big, 0o600},
		{"mode 0644", valid, 0o644},
	} {
		t.Run(tc.name, func(t *testing.T) {
			env := scopeServer(t)
			journal := filepath.Join(env.dir, "experiments.jsonl")
			if err := os.WriteFile(journal, tc.data, 0o600); err != nil {
				t.Fatal(err)
			}
			os.Chmod(journal, tc.mode)
			for name, call := range map[string]func() (int, string, http.Header){
				"GET plan": func() (int, string, http.Header) {
					r := env.get("/api/development/experiments/exp_aaaaaaaaaaaaaaaaaaaa", env.expectOf("A"))
					return r.Code, r.Body.String(), r.Header()
				},
				"GET list": func() (int, string, http.Header) {
					r := env.get("/api/development/experiments?run=nudge_00000000000000000001", env.expectOf("A"))
					return r.Code, r.Body.String(), r.Header()
				},
				"POST plan": func() (int, string, http.Header) {
					r := env.post("/api/development/experiments", post, env.expectOf("A"))
					return r.Code, r.Body.String(), r.Header()
				},
			} {
				code, body, hdr := call()
				if code != http.StatusServiceUnavailable || scopeCode2(body) != "EXPERIMENT_JOURNAL_UNSAFE" {
					t.Errorf("%s: HTTP %d %s", name, code, body)
				}
				if strings.Contains(body, secret) || strings.Contains(body, "exp_aaaa") || strings.Contains(body, "record") || hdr.Get("X-Cognition-Identity") != "" {
					t.Errorf("%s: the refusal exposes journal content or an attestation: %s", name, body)
				}
			}
			now, _ := os.ReadFile(journal)
			fi, _ := os.Stat(journal)
			if !bytes.Equal(now, tc.data) || fi.Mode().Perm() != tc.mode {
				t.Errorf("the journal changed (%d → %d bytes, mode %v)", len(tc.data), len(now), fi.Mode().Perm())
			}
			if len(env.eng.nonGET) != 0 {
				t.Errorf("Engine writes: %v", env.eng.nonGET)
			}
		})
	}
	// Positive control: with the same valid record in a safe journal the routes get past the journal
	// (the fixture has no processed nudge, so they end at the ordinary 404, not the journal refusal).
	env := scopeServer(t)
	os.WriteFile(filepath.Join(env.dir, "experiments.jsonl"), valid, 0o600)
	if r := env.get("/api/development/experiments/exp_aaaaaaaaaaaaaaaaaaaa", env.expectOf("A")); r.Code != http.StatusNotFound || strings.Contains(r.Body.String(), secret) {
		t.Errorf("safe journal: HTTP %d %s", r.Code, r.Body.String())
	}
	// Unicode bypasses are refused by the server (400) before anything is written.
	for _, v := range []string{"Answer.\u2028Then stop.", "Answer.\u2029Then stop.", "Answer. NOTE： yes", "Answer. STATE＝ keep"} {
		body := strings.Replace(post, "Answer the question directly.", v, 1)
		if r := env.post("/api/development/experiments", body, env.expectOf("A")); r.Code != http.StatusBadRequest {
			t.Errorf("variant %q: HTTP %d %s", v, r.Code, r.Body.String())
		}
	}
	// Re-audit repair: a separator at the start or end of the raw instruction is refused too (JSON escapes,
	// so the bytes reach the server exactly as written here).
	for _, sep := range []string{`\r`, `\n`, `\u000b`, `\f`, `\u0085`, `\u2028`, `\u2029`} {
		for _, v := range []string{sep + "Answer the question directly.", "Answer the question directly." + sep, "  " + sep + "Answer the question directly.", "Answer the question directly." + sep + "  "} {
			body := strings.Replace(post, "Answer the question directly.", v, 1)
			if r := env.post("/api/development/experiments", body, env.expectOf("A")); r.Code != http.StatusBadRequest || !strings.Contains(r.Body.String(), "one line") {
				t.Errorf("variant %q: HTTP %d %s", v, r.Code, r.Body.String())
			}
		}
	}
	if now, _ := os.ReadFile(filepath.Join(env.dir, "experiments.jsonl")); !bytes.Equal(now, valid) {
		t.Error("a refused variant changed the journal")
	}
}

func scopeCode2(body string) string {
	i := strings.Index(body, `"code":"`)
	if i < 0 {
		return ""
	}
	rest := body[i+8:]
	return rest[:strings.Index(rest, `"`)]
}

func TestOwnerFeedbackHTTPBoundaryRefusesUnauthorizedWrites(t *testing.T) {
	env := scopeServer(t)
	path := "/api/development/experiments/exp_00000000000000000000/feedback"
	body := `{"verdict":"SAME","correction":"No observed improvement."}`
	if rec := env.post(path, body, env.expectOf("B")); rec.Code != http.StatusPreconditionFailed {
		t.Fatalf("wrong scope: HTTP %d %s", rec.Code, rec.Body.String())
	}
	if rec := env.do("POST", path, body, map[string]string{"X-TASS-Token":"wrong"}); rec.Code < 400 {
		t.Fatalf("wrong page token: HTTP %d", rec.Code)
	}
	if rec := env.post(path, body, env.expectOf("A")); rec.Code != http.StatusNotFound {
		t.Fatalf("unknown owner plan: HTTP %d %s", rec.Code, rec.Body.String())
	}
	if rec := env.post(path, `{"verdict":"AUTOMATIC","correction":""}`, env.expectOf("A")); rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid verdict: HTTP %d %s", rec.Code, rec.Body.String())
	}
	if rec := env.get(path, env.expectOf("A")); rec.Code != http.StatusMethodNotAllowed && rec.Code != http.StatusNotFound {
		t.Fatalf("feedback POST route readable with GET: HTTP %d", rec.Code)
	}
	if len(env.eng.nonGET) != 0 { t.Fatalf("feedback refusal wrote to Engine: %v", env.eng.nonGET) }
	if _,err:=os.Stat(filepath.Join(env.dir,"experiment-runs")); !os.IsNotExist(err) {
		t.Fatalf("refused feedback created a storage tree: %v",err)
	}
}
