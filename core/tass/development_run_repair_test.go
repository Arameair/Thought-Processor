package tass

// Repair after Codex's Stage 3A audit (PR #121, comment 6832088431). Three invariants:
//   A  no model-log text or metadata is shown unless it is tied to this run by the run's own reply digest;
//   B  a verified historical receipt is not a claim about which Note the Thought holds NOW;
//   C  code-defined statements are made only for an Engine revision that was reconciled and inspected.
// The checks read the JSON Core sends, so they also run against the rejected head b3fcf8b.

import (
	"encoding/json"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"
)

const knownEngineCommit = "c54390df43edddc161c5853c35db64462c39d37f"

func runJSON(t *testing.T, c *Core, nid string) (map[string]any, string) {
	t.Helper()
	r, err := c.DevelopmentRun(nid)
	if err != nil {
		t.Fatalf("DevelopmentRun: %v", err)
	}
	b, _ := json.Marshal(r)
	var m map[string]any
	json.Unmarshal(b, &m)
	return m, string(b)
}

func jsonStage(t *testing.T, m map[string]any, name string) map[string]any {
	t.Helper()
	for _, s := range m["stages"].([]any) {
		if s.(map[string]any)["stage"] == name {
			return s.(map[string]any)
		}
	}
	t.Fatalf("no stage %q", name)
	return nil
}

func reconciled(re *runEngine) {
	re.mu.Lock()
	re.statusCommit, re.recordCommit = knownEngineCommit, knownEngineCommit
	re.mu.Unlock()
}

// ---- A
func TestDevelopmentRunNeverShowsUncorrelatedModelLogText(t *testing.T) {
	foreign := map[string]any{"at": time.Now(), "request_id": "x", "ok": true, "text": "FOREIGN-OWNER-PRIVATE-REPLY", "elapsed_ms": 1,
		"error": "FOREIGN-ERROR-TEXT", "authorized": "FOREIGN-AUTHORIZED-META", "model": "FOREIGN-MODEL-META"}
	val := func(rec map[string]any, k string) map[string]any {
		return rec[k].(map[string]any)["value"].(map[string]any)
	}
	cases := []struct {
		name string
		hook func(rec map[string]any) (int, any)
	}{
		{"no reply digest in the run's response", func(rec map[string]any) (int, any) {
			delete(val(rec, "response"), "reply_sha256")
			rec["model_request"] = foreign
			return 200, rec
		}},
		{"failed request: no digest, not ok", func(rec map[string]any) (int, any) {
			rec["response"] = map[string]any{"present": true, "readable": true, "value": map[string]any{"ok": false, "error": "model down", "http_answered": false, "elapsed_ms": 3}}
			rec["model_request"] = foreign
			return 200, rec
		}},
		{"response record missing", func(rec map[string]any) (int, any) {
			rec["response"] = map[string]any{"present": false, "readable": false}
			rec["model_request"] = foreign
			return 200, rec
		}},
		{"response record unreadable", func(rec map[string]any) (int, any) {
			rec["response"] = map[string]any{"present": true, "readable": false}
			rec["model_request"] = foreign
			return 200, rec
		}},
		{"digest present, text does not match it", func(rec map[string]any) (int, any) {
			rec["model_request"] = foreign
			return 200, rec
		}},
		{"void claim", func(rec map[string]any) (int, any) {
			val(rec, "claim")["void"] = true
			rec["model_request"] = foreign
			return 200, rec
		}},
		{"no digest, no call-log record either", func(rec map[string]any) (int, any) {
			delete(val(rec, "response"), "reply_sha256")
			rec["model_request"], rec["call"] = foreign, nil
			return 200, rec
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, _, re, nid, _ := processedRun(t)
			re.mu.Lock()
			re.recordHook = func(_ string, rec map[string]any) (int, any) { return tc.hook(rec) }
			re.mu.Unlock()
			m, raw := runJSON(t, c, nid)
			if strings.Contains(raw, "FOREIGN-") {
				t.Fatalf("uncorrelated model-log bytes reached Core's response: %s", raw)
			}
			if st := jsonStage(t, m, "model_reply")["status"]; st == "RECORDED" {
				t.Fatalf("model_reply must not be RECORDED: %v", st)
			}
		})
	}
	// Positive control: the same-owner reply whose bytes match the run's digest is shown in full.
	c, _, re, nid, _ := processedRun(t)
	reconciled(re)
	m, _ := runJSON(t, c, nid)
	s := jsonStage(t, m, "model_reply")
	if s["status"] != "RECORDED" || s["text"] != "NOTE: NOTE-TEXT-"+nid+"\nSTATE: resolve" {
		t.Fatalf("positive control: %v", s)
	}
}

// ---- B
func TestDevelopmentRunVerifiedReceiptIsNotCurrentNoteAttribution(t *testing.T) {
	stamp := time.Date(2026, 11, 8, 9, 5, 0, 0, time.UTC)
	note := func(nid, req, text string) map[string]any {
		return map[string]any{"at": stamp, "text": text, "by": "Nemotron (owner-nudge)", "worker_request_id": req, "source_nudge_id": nid}
	}
	step := func(nid, req string) map[string]any {
		return map[string]any{"at": stamp, "kind": "developed", "by": "owner-nudge", "worker_request_id": req, "source_nudge_id": nid}
	}
	cases := []struct {
		name string
		set  func(re *runEngine, nid, req string)
		want string // what the saved stage must mention
	}{
		{"a second current Note with the same nudge and request ID", func(re *runEngine, nid, req string) {
			re.extraNotes = []map[string]any{note(nid, req, "SECOND-CURRENT-NOTE")}
		}, "2 Notes"},
		{"a second developed step with the same IDs", func(re *runEngine, nid, req string) {
			re.extraHist = []map[string]any{step(nid, req)}
		}, "2 developed steps"},
		{"the Note is no longer stored", func(re *runEngine, nid, req string) { re.hideNotes = true }, "0 Notes"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, _, re, nid, _ := processedRun(t)
			if v := c.runView(nid); v == nil || !v.Verified {
				t.Fatalf("setup: Core's receipt must be verified: %+v", v)
			}
			re.mu.Lock()
			tc.set(re, nid, "mr_"+nid[6:14])
			re.mu.Unlock()
			m, raw := runJSON(t, c, nid)
			if m["verified"] != false {
				t.Errorf("verified must be false when the current Thought cannot identify one Note: %v", m["verified"])
			}
			if m["receipt_verified"] != true {
				t.Errorf("the historical receipt is still reported as what it was: %v", m["receipt_verified"])
			}
			if strings.Contains(raw, "exactly one") || strings.Contains(raw, "verified by Core") {
				t.Errorf("an 'exactly one' / 'verified by Core' claim survived: %s", raw)
			}
			s := jsonStage(t, m, "saved")
			if s["status"] != "PARTIAL" || !strings.Contains(s["summary"].(string), tc.want) || !strings.Contains(s["summary"].(string), "When Core wrote its receipt") {
				t.Errorf("saved stage must separate the receipt from the present state: %v", s)
			}
			if p := jsonStage(t, m, "parse"); p["text"] != nil && tc.name != "a second developed step with the same IDs" {
				t.Errorf("no Note may be shown as the one kept: %v", p)
			}
			if strings.Contains(raw, "SECOND-CURRENT-NOTE") {
				t.Errorf("neither duplicate Note is shown as the kept Note: %s", raw)
			}
		})
	}
	// Positive control: one Note, one step, verified receipt.
	c, _, re, nid, _ := processedRun(t)
	reconciled(re)
	m, _ := runJSON(t, c, nid)
	s := jsonStage(t, m, "saved")
	if m["verified"] != true || m["receipt_verified"] != true || s["status"] != "RECORDED" || !strings.Contains(m["result"].(string), "verified by Core") {
		t.Fatalf("positive control: verified=%v receipt=%v result=%v saved=%v", m["verified"], m["receipt_verified"], m["result"], s)
	}
}

// ---- C
func TestDevelopmentRunEngineRevisionIsReconciledBeforeCodeDefinedClaims(t *testing.T) {
	const audited, recorded = "21b3d502775b3ed065ab445b88f51909500770b3", "b594357c021db3a5164f321ec2b66b21198a8455"
	cases := []struct {
		name, status, record, want string
		codeDefined                bool
	}{
		{"the audited fixture: status and record disagree", audited, recorded, "MISMATCH", false},
		{"both agree on a revision nobody inspected", audited, audited, "NOT INSPECTED", false},
		{"the Engine reports no revision", "unknown", "unknown", "NOT REPORTED", false},
		{"status reports none, the record names a known one", "unknown", knownEngineCommit, "NOT REPORTED", false},
		{"both agree on an inspected revision", knownEngineCommit, knownEngineCommit, "RECONCILED", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, _, re, nid, _ := processedRun(t)
			re.mu.Lock()
			re.statusCommit, re.recordCommit = tc.status, tc.record
			re.mu.Unlock()
			m, raw := runJSON(t, c, nid)
			if m["engine_revision"] != tc.want {
				t.Errorf("engine_revision = %v, want %s", m["engine_revision"], tc.want)
			}
			ins := jsonStage(t, m, "instruction")
			var cd []string
			for _, s := range m["stages"].([]any) {
				if s.(map[string]any)["status"] == "CODE-DEFINED" {
					cd = append(cd, s.(map[string]any)["stage"].(string))
				}
			}
			if tc.codeDefined {
				if ins["status"] != "CODE-DEFINED" || ins["text"] == nil || len(cd) != 2 {
					t.Errorf("a reconciled, inspected revision keeps CODE-DEFINED: %v %v", ins, cd)
				}
				return
			}
			if len(cd) != 0 || ins["status"] != "PARTIAL" || ins["text"] != nil {
				t.Errorf("no CODE-DEFINED stage and no instruction text for an unreconciled revision: %v %v", cd, ins)
			}
			fields, _ := json.Marshal(ins["fields"])
			for _, sha := range []string{tc.status, tc.record} {
				if len(sha) == 40 && !strings.Contains(string(fields), sha) {
					t.Errorf("both reported revisions must be shown: missing %s in %s", sha, fields)
				}
			}
			for _, claim := range []string{"thinking off", "at most 256 tokens", "does not search memory", "No tool, skill or notebook runs"} {
				if strings.Contains(raw, claim) {
					t.Errorf("code-derived claim %q made for an unreconciled Engine revision", claim)
				}
			}
		})
	}
}

// The second browser fixture is Core's real output for the three repaired states at once (uncorrelated model
// record, duplicate current Notes, revision mismatch). STAGE3A_WRITE_FIXTURE=1 rewrites it from Core.
func TestDevelopmentRunUnreconciledBrowserFixtureIsCoreOutput(t *testing.T) {
	c, _, re, nid, _ := processedRun(t)
	re.mu.Lock()
	re.statusCommit, re.recordCommit = "", ""
	re.extraNotes = []map[string]any{{"at": time.Date(2026, 11, 8, 9, 5, 0, 0, time.UTC), "text": "SECOND-CURRENT-NOTE", "by": "Nemotron (owner-nudge)",
		"worker_request_id": "mr_" + nid[6:14], "source_nudge_id": nid}}
	re.recordHook = func(_ string, rec map[string]any) (int, any) {
		delete(rec["response"].(map[string]any)["value"].(map[string]any), "reply_sha256")
		rec["model_request"] = map[string]any{"request_id": "x", "ok": true, "text": "FOREIGN-OWNER-PRIVATE-REPLY", "authorized": "FOREIGN-AUTHORIZED-META", "model": "FOREIGN-MODEL-META"}
		rec["call"] = nil
		return 200, rec
	}
	re.mu.Unlock()
	_, raw := runJSON(t, c, nid)
	if strings.Contains(raw, "FOREIGN-") || strings.Contains(raw, "exactly one") || strings.Contains(raw, "CODE-DEFINED\"") {
		t.Fatalf("the unreconciled state leaks or over-claims: %s", raw)
	}
	path := "../test/browser/cognition_stage3a_unreconciled.json"
	if os.Getenv("STAGE3A_WRITE_FIXTURE") == "1" {
		var pretty map[string]any
		json.Unmarshal([]byte(raw), &pretty)
		b, _ := json.MarshalIndent(pretty, "", "  ")
		if err := os.WriteFile(path, append(b, '\n'), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var fx, now map[string]any
	json.Unmarshal(b, &fx)
	json.Unmarshal([]byte(raw), &now)
	shape := func(m map[string]any) string {
		var out []string
		for _, s := range m["stages"].([]any) {
			x := s.(map[string]any)
			out = append(out, x["stage"].(string)+"|"+x["status"].(string)+"|"+x["summary"].(string))
		}
		// Request IDs are random per run: compare everything else.
		return regexp.MustCompile(`mr_[0-9a-f]{8}`).ReplaceAllString(strings.Join(out, "\n"), "mr_ID") + "\n" + m["result"].(string) + "|" + m["engine_revision"].(string)
	}
	if shape(fx) != shape(now) {
		t.Fatalf("the unreconciled browser fixture no longer matches Core; run with STAGE3A_WRITE_FIXTURE=1\nfixture:\n%s\ncore:\n%s", shape(fx), shape(now))
	}
}
