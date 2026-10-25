package main

// Engine-link tests. The Engine here is a TEST FIXTURE (httptest) that speaks
// the documented contract thought-engine.processor/1-experimental. It proves
// Processor's own behaviour. The real Engine + real Processor run together in
// the Thought-engine repo's demo (docs/ENGINE_PROCESSOR_LOOP_20261026.md).

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

type fixtureEngine struct {
	mu       sync.Mutex
	srv      *httptest.Server
	code     string
	token    string
	on       bool
	ids      []map[string]any
	obs      []map[string]any
	worker   string
	lastAuth string
}

func newFixtureEngine(t *testing.T) *fixtureEngine {
	f := &fixtureEngine{code: "GOOD-CODE", worker: "READY"}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		out := func(code int, v any) { w.WriteHeader(code); json.NewEncoder(w).Encode(v) }
		if r.URL.Path == "/api/health" {
			out(200, map[string]any{"app": "thought-engine", "version": "fixture", "protocol": "thought-engine.processor/1-experimental"})
			return
		}
		if r.Header.Get("Origin") != "" {
			out(403, map[string]any{"error": "web page"})
			return
		}
		if r.URL.Path == "/api/v1/attach" {
			var in map[string]string
			json.NewDecoder(r.Body).Decode(&in)
			if in["code"] != f.code || f.code == "" {
				out(403, map[string]any{"error": "DENIED: wrong attach code"})
				return
			}
			f.code, f.token = "", "tea_"+strings.Repeat("ab", 32)
			out(200, map[string]any{"attachment_id": "att-1", "token": f.token, "engine": map[string]any{"installation_id": "inst-1"}})
			return
		}
		f.lastAuth = r.Header.Get("Authorization")
		if f.token == "" || f.lastAuth != "Bearer "+f.token {
			out(401, map[string]any{"error": "not attached"})
			return
		}
		switch {
		case r.URL.Path == "/api/v1/status":
			out(200, map[string]any{"engine": map[string]any{"version": "fixture", "installation_id": "inst-1", "test_fixture_worker": true},
				"worker": map[string]any{"state": f.worker, "label": f.worker, "test_fixture": true}, "waiting": 0, "identities": f.ids})
		case r.URL.Path == "/api/v1/identities":
			f.ids = append(f.ids, map[string]any{"entity_id": "e1", "display_name": "TASS", "thought_processing": false})
			out(201, map[string]any{"entity_id": "e1"})
		case r.URL.Path == "/api/v1/identities/e1/processing":
			var in map[string]bool
			json.NewDecoder(r.Body).Decode(&in)
			f.on = in["enabled"]
			out(200, map[string]any{"thought_processing": f.on})
		case r.URL.Path == "/api/v1/identities/e1/observations":
			if !f.on {
				out(403, map[string]any{"error": "DENIED: Thought Processing is off for TASS"})
				return
			}
			var in map[string]string
			json.NewDecoder(r.Body).Decode(&in)
			o := map[string]any{"observation_id": in["observation_id"], "text": in["text"], "state": "waiting_for_ai", "state_text": "waiting for AI"}
			f.obs = append([]map[string]any{o}, f.obs...)
			out(202, map[string]any{"observation": o})
		case strings.HasPrefix(r.URL.Path, "/api/v1/identities/e1/activity"):
			out(200, map[string]any{"identity": map[string]any{"entity_id": "e1", "display_name": "TASS", "thought_processing": f.on},
				"observations": f.obs, "thoughts": []any{}, "counts": map[string]any{"traces": 0}})
		case r.URL.Path == "/api/v1/detach":
			f.token = ""
			out(200, map[string]any{"attached": false})
		default:
			out(404, map[string]any{"error": "not found"})
		}
	}))
	t.Cleanup(f.srv.Close)
	return f
}

func linkState(t *testing.T, h *harness) LinkView {
	t.Helper()
	_, body := h.do("GET", "/api/link", "", nil)
	var v LinkView
	if err := json.Unmarshal([]byte(body), &v); err != nil {
		t.Fatalf("link: %v %s", err, body)
	}
	return v
}

func TestEngineLinkAttachObserveAndSecrets(t *testing.T) {
	f := newFixtureEngine(t)
	dir := t.TempDir()
	h := start(t, dir)
	h.signIn()
	addr := strings.TrimPrefix(f.srv.URL, "http://")
	if code, body := h.post("/api/candidates", `{"engine":"`+addr+`"}`); code != 200 || !strings.Contains(body, `"CLAIMS_ENGINE"`) {
		t.Fatalf("engine candidate: %d %s", code, body)
	}
	if v := linkState(t, h); v.State != "NOT_ATTACHED" {
		t.Fatalf("before attach: %s", v.State)
	}
	// Wrong code: refused, nothing saved.
	if code, body := h.post("/api/link/attach", `{"code":"BAD-CODE"}`); code != 400 || !strings.Contains(body, "wrong attach code") {
		t.Fatalf("wrong code: %d %s", code, body)
	}
	// A web page from elsewhere can't attach through Processor either.
	if code, _ := h.do("POST", "/api/link/attach", `{"code":"GOOD-CODE"}`, map[string]string{"Origin": "http://evil.example", "Content-Type": "application/json", "X-TP-CSRF": h.csrf}); code != 403 {
		t.Fatalf("foreign origin attach: %d", code)
	}
	if code, body := h.post("/api/link/attach", `{"code":"GOOD-CODE"}`); code != 200 || !strings.Contains(body, `"ATTACHED"`) {
		t.Fatalf("attach: %d %s", code, body)
	}
	// Observation refused until an identity exists and Processing is on.
	if code, _ := h.post("/api/link/observe", `{"text":"Player entered Goldshire"}`); code != 409 {
		t.Fatalf("observe without identity: %d", code)
	}
	h.post("/api/link/identity", `{"display_name":"TASS"}`)
	if code, body := h.post("/api/link/observe", `{"text":"Player entered Goldshire"}`); code != 400 || !strings.Contains(body, "Thought Processing is off") {
		t.Fatalf("observe while off: %d %s", code, body)
	}
	h.post("/api/link/processing", `{"enabled":true}`)
	canary := "Goldshire-CANARY-7731"
	if code, body := h.post("/api/link/observe", `{"text":"Player entered `+canary+`"}`); code != 202 || !strings.Contains(body, "waiting_for_ai") {
		t.Fatalf("observe: %d %s", code, body)
	}
	v := linkState(t, h)
	if v.State != "ATTACHED" || v.Activity == nil || v.EntityID != "e1" {
		t.Fatalf("attached view: %+v", v)
	}
	_, st := h.do("GET", "/api/state", "", nil)
	if !strings.Contains(st, `"set_engine"`) && !strings.Contains(st, `"start_worker"`) && !strings.Contains(st, `"observe"`) {
		t.Fatalf("next action should follow the link: %s", st)
	}
	// The key is in the profile, 0600, and nowhere else.
	b, _ := os.ReadFile(filepath.Join(dir, "engine-link.json"))
	if !strings.Contains(string(b), f.token) {
		t.Fatal("key not saved")
	}
	if runtime.GOOS != "windows" {
		if fi, _ := os.Stat(filepath.Join(dir, "engine-link.json")); fi.Mode().Perm() != 0o600 {
			t.Fatalf("engine-link.json mode %v", fi.Mode().Perm())
		}
	}
	_, ctxText := h.do("GET", "/api/context", "", nil)
	_, logs := h.do("GET", "/api/logs", "", nil)
	for name, s := range map[string]string{"context": ctxText, "logs": logs, "state": st} {
		if strings.Contains(s, f.token) || strings.Contains(s, "GOOD-CODE") || strings.Contains(s, canary) {
			t.Fatalf("%s leaked the key, the code or observation text", name)
		}
	}
	if !strings.Contains(ctxText, "engine link: state=ATTACHED") {
		t.Fatalf("context lacks link state:\n%s", ctxText)
	}
	// Restart Processor: still attached (same profile).
	h.stop()
	logb, _ := os.ReadFile(filepath.Join(dir, "logs", "processor.jsonl"))
	if strings.Contains(string(logb), f.token) || strings.Contains(string(logb), canary) {
		t.Fatal("log file leaked")
	}
	h2 := start(t, dir)
	h2.signIn()
	h2.post("/api/probe", `{}`)
	if v := linkState(t, h2); v.State != "ATTACHED" || v.EntityID != "e1" {
		t.Fatalf("after restart: %s", v.State)
	}
	// Engine forgets the key → Processor says so honestly.
	f.mu.Lock()
	f.token = "tea_" + strings.Repeat("cd", 32)
	f.mu.Unlock()
	if v := linkState(t, h2); v.State != "ATTACHMENT_REFUSED" {
		t.Fatalf("revoked: %s", v.State)
	}
	// Engine stops → not running, attachment kept.
	f.srv.Close()
	if v := linkState(t, h2); v.State != "ENGINE_NOT_RUNNING" {
		t.Fatalf("engine down: %s", v.State)
	}
	// Detach forgets the key even if the Engine can't be told.
	if code, _ := h2.post("/api/link/detach", `{}`); code != 200 {
		t.Fatal("detach")
	}
	b, _ = os.ReadFile(filepath.Join(dir, "engine-link.json"))
	if strings.Contains(string(b), "tea_") {
		t.Fatal("key kept after detach")
	}
}
