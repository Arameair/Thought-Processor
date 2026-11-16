package main

// #96 / PR #98 review: the REAL GET/POST /api/nudges handlers, with a real Core, a stand-in
// Identity (LOCKED/UNLOCKED, owner grants) and a GET-only stand-in Engine.

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Arameair/Thought-Processor/core/engine"
	"github.com/Arameair/Thought-Processor/core/github"
	"github.com/Arameair/Thought-Processor/core/identitysvc"
	"github.com/Arameair/Thought-Processor/core/tass"
)

const nudgeSvcKey = "core-service-key-0123456789abcdef0123"

type stubIdentity struct {
	mu           sync.Mutex
	state        string
	nudgeGranted bool
	actions      []string
}

func (s *stubIdentity) serve(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ent := map[string]any{"entity_id": "ent_0123", "name": "Mira", "grants": []string{"conversation", "thought.process"}}
	switch r.URL.Path {
	case "/v1/state":
		json.NewEncoder(w).Encode(map[string]any{"state": s.state, "entity": ent})
	case "/v1/authorize":
		if r.Header.Get("Authorization") != "Bearer "+nudgeSvcKey {
			http.Error(w, "no", 401)
			return
		}
		var in map[string]string
		json.NewDecoder(r.Body).Decode(&in)
		s.actions = append(s.actions, in["action"])
		// Like tass-identity: thought.process is a default grant; cognition.nudge.draft is not.
		allow := s.state == "UNLOCKED" && (in["action"] != "cognition.nudge.draft" || s.nudgeGranted)
		json.NewEncoder(w).Encode(map[string]any{"allow": allow, "decision_id": fmt.Sprintf("dec_%d", len(s.actions)),
			"entity_id": "ent_0123", "entity_name": "Mira"})
	default:
		w.WriteHeader(404)
	}
}

type stubEngine struct {
	mu      sync.Mutex
	nonGET  []string
	srvCall int
}

func (e *stubEngine) serve(w http.ResponseWriter, r *http.Request) {
	e.mu.Lock()
	e.srvCall++
	if r.Method != "GET" {
		e.nonGET = append(e.nonGET, r.Method+" "+r.URL.Path)
	}
	e.mu.Unlock()
	if r.Method != "GET" {
		w.WriteHeader(405)
		return
	}
	stamp := time.Date(2026, 11, 8, 9, 0, 0, 0, time.UTC)
	w.Header().Set("Content-Type", "application/json")
	switch r.URL.Path {
	case "/api/status":
		json.NewEncoder(w).Encode(map[string]any{"entities": []any{
			map[string]any{"card": map[string]string{"entity_id": "eng_1", "display_name": "Mira", "entity_type": "ai_companion"}},
		}, "config": map[string]string{"commit": "21b3d502775b3ed065ab445b88f51909500770b3"}})
	case "/api/entities/eng_1/thoughts/thought_1":
		json.NewEncoder(w).Encode(map[string]any{"thought": map[string]any{
			"thought_id": "thought_1", "entity_id": "eng_1", "topic": "t", "question": "q", "state": "active", "updated_at": stamp}})
	case "/api/cycles":
		json.NewEncoder(w).Encode(map[string]any{"cycles": []any{}})
	default:
		w.WriteHeader(404)
	}
}

func nudgeServer(t *testing.T) (http.Handler, *stubIdentity, *stubEngine, string) {
	t.Helper()
	ids, eng := &stubIdentity{state: "LOCKED"}, &stubEngine{}
	isrv, esrv := httptest.NewServer(http.HandlerFunc(ids.serve)), httptest.NewServer(http.HandlerFunc(eng.serve))
	t.Cleanup(isrv.Close)
	t.Cleanup(esrv.Close)
	kf := filepath.Join(t.TempDir(), "core.key")
	os.WriteFile(kf, []byte(nudgeSvcKey+"\n"), 0o600)
	cl, err := identitysvc.New(isrv.URL, kf)
	if err != nil {
		t.Fatal(err)
	}
	gh := github.New("")
	gh.Base = "http://127.0.0.1:1" // never used
	ec := engine.New(esrv.URL)
	ec.Key = "testkey"
	dir := t.TempDir()
	core, err := tass.New(tass.Config{DataDir: dir, Identity: "tass-identity", Label: "tass", SyncEvery: 20 * time.Second,
		MaxBackoff: time.Minute, IdentitySvc: cl}, gh, ec)
	if err != nil {
		t.Fatal(err)
	}
	return handler(core, "page-token", "api-key-file-token"), ids, eng, dir
}

type hreq struct {
	method, host, path, token, origin, ctype, body string
}

func do(h http.Handler, q hreq) (int, map[string]any) {
	host := q.host
	if host == "" {
		host = "127.0.0.1:8795"
	}
	r := httptest.NewRequest(q.method, "http://"+host+q.path, strings.NewReader(q.body))
	if q.token != "" {
		r.Header.Set("X-TASS-Token", q.token)
	}
	if q.origin != "" {
		r.Header.Set("Origin", q.origin)
	}
	if q.ctype != "" {
		r.Header.Set("Content-Type", q.ctype)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	out := map[string]any{}
	json.Unmarshal(rec.Body.Bytes(), &out)
	return rec.Code, out
}

const goodBody = `{"source_thought_id":"thought_1","recipient_id":"position-2","reason_code":"follow_up","requested_action":"Look again.","idempotency_key":"http-key-0001"}`

func post(body string) hreq {
	return hreq{method: "POST", path: "/api/nudges", token: "page-token", origin: "http://127.0.0.1:8795", ctype: "application/json", body: body}
}

func TestNudgeHTTPRoutes(t *testing.T) {
	h, ids, eng, dir := nudgeServer(t)
	journal := filepath.Join(dir, "nudges.jsonl")
	noJournal := func(step string) {
		t.Helper()
		if _, err := os.Stat(journal); !os.IsNotExist(err) {
			t.Fatalf("%s: a refused request wrote the journal", step)
		}
	}

	// Token, host, Origin, content type, JSON: all checked by the real router before Core runs.
	for name, q := range map[string]hreq{
		"no token":      {method: "POST", path: "/api/nudges", ctype: "application/json", body: goodBody},
		"wrong token":   {method: "POST", path: "/api/nudges", token: "guess", ctype: "application/json", body: goodBody},
		"foreign host":  {method: "POST", host: "evil.example", path: "/api/nudges", token: "page-token", ctype: "application/json", body: goodBody},
		"GET no token":  {method: "GET", path: "/api/nudges"},
		"GET bad token": {method: "GET", path: "/api/nudges", token: "guess"},
	} {
		if code, _ := do(h, q); code != http.StatusUnauthorized {
			t.Fatalf("%s: want 401, got %d", name, code)
		}
	}
	bad := post(goodBody)
	bad.origin = "https://evil.example"
	if code, _ := do(h, bad); code != http.StatusForbidden {
		t.Fatalf("foreign Origin: want 403, got %d", code)
	}
	bad = post(goodBody)
	bad.ctype = "text/plain"
	if code, _ := do(h, bad); code != http.StatusUnsupportedMediaType {
		t.Fatalf("not JSON: want 415, got %d", code)
	}
	if code, _ := do(h, post(`{"source_thought_id":`)); code != http.StatusBadRequest {
		t.Fatalf("malformed JSON: want 400, got %d", code)
	}
	noJournal("transport checks")

	// LOCKED Identity: read and write refused.
	if code, _ := do(h, post(goodBody)); code != http.StatusForbidden {
		t.Fatalf("LOCKED POST: want 403, got %d", code)
	}
	if code, _ := do(h, hreq{method: "GET", path: "/api/nudges", token: "page-token"}); code != http.StatusForbidden {
		t.Fatalf("LOCKED GET: want 403, got %d", code)
	}
	ids.mu.Lock()
	ids.state = "UNLOCKED"
	ids.mu.Unlock()
	// Unlocked, not linked to Engine history yet.
	if code, _ := do(h, post(goodBody)); code != http.StatusBadRequest {
		t.Fatalf("unlinked POST: want 400, got %d", code)
	}
	if code, _ := do(h, hreq{method: "POST", path: "/api/cognition/link", token: "page-token", origin: "http://127.0.0.1:8795",
		ctype: "application/json", body: `{"engine_entity_id":"eng_1"}`}); code != http.StatusOK {
		t.Fatalf("link: %d", code)
	}
	// Unlocked + linked, but the owner hasn't granted cognition.nudge.draft: thought.process is not enough.
	code, out := do(h, post(goodBody))
	if code != http.StatusForbidden || !strings.Contains(fmt.Sprint(out["error"]), "cognition.nudge-draft") {
		t.Fatalf("no grant: want 403 naming the grant, got %d %v", code, out)
	}
	noJournal("no grant")

	ids.mu.Lock()
	ids.nudgeGranted = true
	ids.mu.Unlock()
	code, out = do(h, post(goodBody))
	if code != http.StatusCreated || out["status"] != "DRAFT" || out["model_calls"] != float64(0) {
		t.Fatalf("save: %d %v", code, out)
	}
	if p, _ := out["processing"].(map[string]any); p["state"] != "NOT_FOR_PROCESSING" {
		t.Fatalf("a card-position draft must be NOT_FOR_PROCESSING (P0: derived, never a stale BLOCKED): %v", out["processing"])
	}
	id := out["nudge_id"]
	digest, _ := out["body_digest"].(string)
	code, out = do(h, post(goodBody))
	if code != http.StatusOK || out["replayed"] != true || out["nudge_id"] != id {
		t.Fatalf("replay: %d %v", code, out)
	}
	if code, _ := do(h, post(strings.Replace(goodBody, "position-2", "self", 1))); code != http.StatusConflict {
		t.Fatalf("same key, different body: want 409, got %d", code)
	}
	if code, _ := do(h, post(strings.Replace(strings.Replace(goodBody, "position-2", "discord", 1), "0001", "0002", 1))); code != http.StatusBadRequest {
		t.Fatalf("external recipient: want 400, got %d", code)
	}
	if code, _ := do(h, post(strings.Replace(goodBody, `"idempotency_key"`, `"expires_in_hours":9223372036854775807,"idempotency_key"`, 1))); code != http.StatusBadRequest {
		t.Fatalf("max-int expiry: want 400, got %d", code)
	}
	code, out = do(h, hreq{method: "GET", path: "/api/nudges?thought_id=thought_1", token: "page-token"})
	if list, _ := out["nudges"].([]any); code != http.StatusOK || len(list) != 1 {
		t.Fatalf("history: %d %v", code, out)
	}
	// Revoke the owner grant: the SAME request (a replay) is now 403, a new one too; nothing is added.
	ids.mu.Lock()
	ids.nudgeGranted = false
	ids.mu.Unlock()
	for _, body := range []string{goodBody, strings.Replace(goodBody, "0001", "0003", 1)} {
		if code, out := do(h, post(body)); code != http.StatusForbidden || out["nudge_id"] != nil {
			t.Fatalf("after revoke: want 403 with no draft, got %d %v", code, out)
		}
	}
	// Reading history is independently authorized and unchanged.
	code, out = do(h, hreq{method: "GET", path: "/api/nudges", token: "page-token"})
	if list, _ := out["nudges"].([]any); code != http.StatusOK || len(list) != 1 {
		t.Fatalf("history after revoke: %d %v", code, out)
	}
	ids.mu.Lock()
	ids.nudgeGranted = true
	ids.mu.Unlock()
	// A write failure on disk is a 500 that says nothing was saved.
	// (Works the same as root or not: the journal path becomes a directory, so it can't be read or appended.)
	if err := os.Rename(journal, journal+".moved"); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(journal, 0o700); err != nil {
		t.Fatal(err)
	}
	if code, _ := do(h, post(strings.Replace(goodBody, "0001", "0004", 1))); code == http.StatusCreated || code == http.StatusOK {
		t.Fatalf("a broken journal must not report success, got %d", code)
	}
	os.Remove(journal)
	if err := os.Rename(journal+".moved", journal); err != nil {
		t.Fatal(err)
	}
	// Stage 2: the process route exists, but a nudge to a card position is draft-only, a stale page
	// digest is refused, and none of it reaches the Engine.
	pp := func(body string) int {
		code, _ := do(h, hreq{method: "POST", path: "/api/nudges/" + fmt.Sprint(id) + "/process", token: "page-token",
			origin: "http://127.0.0.1:8795", ctype: "application/json", body: body})
		return code
	}
	if code := pp(`{"body_digest":"stale"}`); code != http.StatusConflict {
		t.Fatalf("stale digest: want 409, got %d", code)
	}
	if code := pp(fmt.Sprintf(`{"body_digest":%q}`, digest)); code != http.StatusBadRequest {
		t.Fatalf("card-position nudge: want 400, got %d", code)
	}
	if code, _ := do(h, hreq{method: "POST", path: "/api/nudges/" + fmt.Sprint(id) + "/process", body: "{}", ctype: "application/json"}); code != http.StatusUnauthorized {
		t.Fatalf("process without the page token: want 401, got %d", code)
	}
	eng.mu.Lock()
	defer eng.mu.Unlock()
	if len(eng.nonGET) != 0 {
		t.Fatalf("the Engine received operations: %v", eng.nonGET)
	}
}
