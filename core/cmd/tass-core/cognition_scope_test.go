package main

// P0 Cognition lifecycle (#106, contract v2 §2): every entity-scoped Cognition response is ATTESTED
// with the scope Core authorized it under, a page may NARROW a request to the scope it believes is
// current, and Core re-checks the scope at the response boundary. These tests use only the real
// HTTP handler, a stand-in Identity whose unlocked entity can switch between Owner A and Owner B,
// and a GET-only stand-in Engine that can HOLD one read so the switch happens mid-request.
// They contain no browser code: they prove the server side of the contract.

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

// Owner A and Owner B. Their Engine entities hold a Thought with the SAME ID and different text.
var scopeOwners = map[string]struct{ id, name, engine, marker string }{
	"A": {"ent_aaaaaaaaaaaa", "Mira", "eng_1", "OWNER-A-PRIVATE"},
	"B": {"ent_bbbbbbbbbbbb", "Nova", "eng_2", "OWNER-B-PRIVATE"},
}

type scopeIdentity struct {
	mu        sync.Mutex
	state     string // UNLOCKED | LOCKED
	who       string // A | B
	authorize int    // POST /v1/authorize calls (each one is an audited Identity decision)
	stateReqs int    // GET /v1/state calls (read-only, never audited by Identity cfc4079)
	decisions int
}

func (s *scopeIdentity) set(state, who string) {
	s.mu.Lock()
	s.state, s.who = state, who
	s.mu.Unlock()
}

func (s *scopeIdentity) counts() (int, int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.authorize, s.stateReqs
}

func (s *scopeIdentity) serve(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	o := scopeOwners[s.who]
	ent := map[string]any{"entity_id": o.id, "name": o.name, "grants": []string{"conversation", "thought.process", "cognition.nudge.draft"}}
	switch r.URL.Path {
	case "/v1/state":
		s.stateReqs++
		json.NewEncoder(w).Encode(map[string]any{"state": s.state, "entity": ent})
	case "/v1/authorize":
		if r.Header.Get("Authorization") != "Bearer "+nudgeSvcKey {
			http.Error(w, "no", 401)
			return
		}
		s.authorize++
		s.decisions++
		json.NewEncoder(w).Encode(map[string]any{"allow": s.state == "UNLOCKED", "decision_id": fmt.Sprintf("dec_%s%07d", strings.ToLower(s.who), s.decisions),
			"entity_id": o.id, "entity_name": o.name})
	default:
		w.WriteHeader(404)
	}
}

type scopeEngine struct {
	mu       sync.Mutex
	hits     map[string]int
	nonGET   []string
	holdPath string
	holdSkip int
	entered  chan struct{}
	release  chan struct{}
}

// hold makes the (skip+1)-th next GET of path wait until release is closed.
func (e *scopeEngine) hold(path string, skip int) (entered, release chan struct{}) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.holdPath, e.holdSkip = path, skip
	e.entered, e.release = make(chan struct{}), make(chan struct{})
	return e.entered, e.release
}

func (e *scopeEngine) hit(path string) int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.hits[path]
}

func (e *scopeEngine) serve(w http.ResponseWriter, r *http.Request) {
	e.mu.Lock()
	e.hits[r.URL.Path]++
	if r.Method != "GET" {
		e.nonGET = append(e.nonGET, r.Method+" "+r.URL.Path)
	}
	var wait chan struct{}
	if r.URL.Path == e.holdPath {
		if e.holdSkip > 0 {
			e.holdSkip--
		} else {
			e.holdPath = ""
			wait = e.release
			close(e.entered)
		}
	}
	e.mu.Unlock()
	if wait != nil {
		<-wait
	}
	if r.Method != "GET" {
		w.WriteHeader(405)
		return
	}
	stamp := time.Date(2026, 11, 10, 9, 0, 0, 0, time.UTC)
	thought := func(o struct{ id, name, engine, marker string }) map[string]any {
		return map[string]any{"thought_id": "thought_1", "entity_id": o.engine, "topic": "t", "question": "QUESTION " + o.marker,
			"state": "active", "updated_at": stamp, "created_at": stamp,
			"notes":   []any{map[string]any{"at": stamp, "text": "NOTE " + o.marker, "by": "engine", "worker_request_id": "mr_1"}},
			"history": []any{map[string]any{"at": stamp, "kind": "developed", "worker_request_id": "mr_1"}}}
	}
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	switch p := r.URL.Path; {
	case p == "/api/status":
		enc.Encode(map[string]any{"entities": []any{
			map[string]any{"card": map[string]string{"entity_id": "eng_1", "display_name": "Mira", "entity_type": "ai_companion"}},
			map[string]any{"card": map[string]string{"entity_id": "eng_2", "display_name": "Nova", "entity_type": "ai_companion"}},
		}, "config": map[string]string{"commit": "21b3d502775b3ed065ab445b88f51909500770b3"}})
	case p == "/api/entities/eng_1" || p == "/api/entities/eng_2":
		o := scopeOwners["A"]
		if p == "/api/entities/eng_2" {
			o = scopeOwners["B"]
		}
		enc.Encode(map[string]any{"card": map[string]string{"entity_id": o.engine, "display_name": o.name, "entity_type": "ai_companion"},
			"thoughts": []any{thought(o)}, "traces": []any{}})
	case p == "/api/entities/eng_1/thoughts/thought_1":
		enc.Encode(map[string]any{"thought": thought(scopeOwners["A"]), "evidence": []any{}})
	case p == "/api/entities/eng_2/thoughts/thought_1":
		enc.Encode(map[string]any{"thought": thought(scopeOwners["B"]), "evidence": []any{}})
	case p == "/api/cycles":
		enc.Encode(map[string]any{"cycles": []any{map[string]any{"cycle_id": "tp_a", "entity_id": "eng_1", "state": "evaluated",
			"received_at": stamp, "spark": map[string]string{"thought_id": "thought_1"}}}})
	case p == "/api/cycles/tp_a":
		enc.Encode(map[string]any{"cycle_id": "tp_a", "entity_id": "eng_1", "state": "evaluated", "received_at": stamp,
			"spark": map[string]string{"thought_id": "thought_1"}})
	case p == "/api/cycles/tp_a/journal":
		enc.Encode(map[string]any{"cycle_id": "tp_a", "source": "data/cycles.jsonl", "steps": []any{map[string]any{"offset": 0, "at": stamp,
			"stage": "STAGE-" + scopeOwners["A"].marker, "source": "data/cycles.jsonl", "status": "ok"}}, "missing": []string{}})
	case p == "/api/pulse":
		enc.Encode(map[string]any{"control": "RUNNING", "state": "READY", "every_s": 20,
			"ticks": []any{map[string]any{"pulse_id": "pulse_1", "at": stamp, "trigger": "timer", "state": "READY"}}, "jobs": []any{}})
	default:
		w.WriteHeader(404)
	}
}

type scopeEnv struct {
	h    http.Handler
	ids  *scopeIdentity
	eng  *scopeEngine
	dir  string
	link map[string]string // owner -> link discriminator, as the library body reports it
}

// scopeServer: real Core + handler. Owner A links eng_1 and Owner B links eng_2, both as owner actions;
// then Owner A is unlocked again.
func scopeServer(t *testing.T) *scopeEnv {
	t.Helper()
	ids, eng := &scopeIdentity{state: "UNLOCKED", who: "A"}, &scopeEngine{hits: map[string]int{}}
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
	gh.Base = "http://127.0.0.1:1"
	ec := engine.New(esrv.URL)
	ec.Key = "testkey"
	dir := t.TempDir()
	core, err := tass.New(tass.Config{DataDir: dir, Identity: "tass-identity", Label: "tass", SyncEvery: 20 * time.Second,
		MaxBackoff: time.Minute, IdentitySvc: cl}, gh, ec)
	if err != nil {
		t.Fatal(err)
	}
	env := &scopeEnv{h: handler(core, "page-token", "api-key-file-token"), ids: ids, eng: eng, dir: dir, link: map[string]string{}}
	for _, who := range []string{"A", "B"} {
		ids.set("UNLOCKED", who)
		rec := env.post("/api/cognition/link", `{"engine_entity_id":"`+scopeOwners[who].engine+`"}`, nil)
		if rec.Code != 200 {
			t.Fatalf("setup: link %s: HTTP %d %s", who, rec.Code, rec.Body.String())
		}
		lib := env.get("/api/cognition/library", nil)
		var body struct {
			Link struct {
				Link *struct {
					DecisionID string `json:"identity_decision_id"`
				} `json:"link"`
			} `json:"link"`
		}
		json.Unmarshal(lib.Body.Bytes(), &body)
		if lib.Code != 200 || body.Link.Link == nil || body.Link.Link.DecisionID == "" {
			t.Fatalf("setup: library for %s: HTTP %d %s", who, lib.Code, lib.Body.String())
		}
		env.link[who] = body.Link.Link.DecisionID
	}
	ids.set("UNLOCKED", "A")
	return env
}

// expectOf is the narrowing precondition a page that shows owner `who` sends.
func (e *scopeEnv) expectOf(who string) map[string]string {
	o := scopeOwners[who]
	return map[string]string{"X-Cognition-Expect-Identity": o.id, "X-Cognition-Expect-Engine": o.engine, "X-Cognition-Expect-Link": e.link[who]}
}

func (e *scopeEnv) do(method, path, body string, hdr map[string]string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, "http://127.0.0.1:8795"+path, strings.NewReader(body))
	r.Header.Set("X-TASS-Token", "page-token")
	if method == "POST" {
		r.Header.Set("Origin", "http://127.0.0.1:8795")
		r.Header.Set("Content-Type", "application/json")
	}
	for k, v := range hdr {
		r.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	e.h.ServeHTTP(rec, r)
	return rec
}

func (e *scopeEnv) get(path string, hdr map[string]string) *httptest.ResponseRecorder {
	return e.do("GET", path, "", hdr)
}

func (e *scopeEnv) post(path, body string, hdr map[string]string) *httptest.ResponseRecorder {
	return e.do("POST", path, body, hdr)
}

// The six entity-scoped reads the Cognition page makes, and the Engine path each one can be held at.
var scopedReads = []struct {
	name, path, holdAt string
	holdSkip           int
}{
	{"library", "/api/cognition/library", "/api/cycles", 0},
	{"pulse", "/api/cognition/pulse", "/api/pulse", 0},
	{"thought", "/api/cognition/thoughts/thought_1", "/api/entities/eng_1/thoughts/thought_1", 0},
	{"nudges", "/api/nudges?thought_id=thought_1", "/api/status", 1},
	{"timeline", "/api/cognition/cycles/tp_a/timeline", "/api/cycles/tp_a/journal", 0},
	{"export-md", "/api/cognition/cycles/tp_a/export?format=md", "/api/cycles/tp_a/journal", 0},
	{"export-json", "/api/cognition/cycles/tp_a/export?format=json", "/api/cycles/tp_a/journal", 0},
}

func scopeCode(rec *httptest.ResponseRecorder) string {
	var b struct {
		Code string `json:"code"`
	}
	json.Unmarshal(rec.Body.Bytes(), &b)
	return b.Code
}

func noPrivate(t *testing.T, step string, rec *httptest.ResponseRecorder) {
	t.Helper()
	for _, who := range []string{"A", "B"} {
		if strings.Contains(rec.Body.String(), scopeOwners[who].marker) {
			t.Errorf("%s: response body carries %s's private record text", step, who)
		}
	}
	for _, h := range []string{"X-Cognition-Identity", "X-Cognition-Engine", "X-Cognition-Link"} {
		if rec.Header().Get(h) != "" {
			t.Errorf("%s: a refused response attests a scope (%s)", step, h)
		}
	}
}

// S5: every successful entity-scoped read says which Identity entity, Engine entity and link Core
// authorized it under. The library body and its headers agree.
func TestCognitionScopeAttestedOnEveryRead(t *testing.T) {
	env := scopeServer(t)
	a := scopeOwners["A"]
	for _, rd := range scopedReads {
		rec := env.get(rd.path, nil)
		if rec.Code != 200 {
			t.Fatalf("%s: HTTP %d %s", rd.name, rec.Code, rec.Body.String())
		}
		got := [3]string{rec.Header().Get("X-Cognition-Identity"), rec.Header().Get("X-Cognition-Engine"), rec.Header().Get("X-Cognition-Link")}
		if want := [3]string{a.id, a.engine, env.link["A"]}; got != want {
			t.Errorf("%s: attested scope %v, want %v", rd.name, got, want)
		}
	}
	var lib struct {
		Identity string `json:"identity_entity_id"`
		Engine   string `json:"engine_entity_id"`
	}
	rec := env.get("/api/cognition/library", nil)
	json.Unmarshal(rec.Body.Bytes(), &lib)
	if lib.Identity != rec.Header().Get("X-Cognition-Identity") || lib.Engine != rec.Header().Get("X-Cognition-Engine") {
		t.Errorf("library body (%s, %s) disagrees with its attestation headers", lib.Identity, lib.Engine)
	}
	// An Identity entity with no Engine link: explicit sentinels, never an empty or guessed value.
	env.post("/api/cognition/unlink", `{}`, nil)
	rec = env.get("/api/cognition/library", nil)
	if rec.Code != 200 || rec.Header().Get("X-Cognition-Engine") != "UNLINKED" || rec.Header().Get("X-Cognition-Link") != "NONE" {
		t.Errorf("unlinked library: HTTP %d, engine %q, link %q; want UNLINKED / NONE", rec.Code,
			rec.Header().Get("X-Cognition-Engine"), rec.Header().Get("X-Cognition-Link"))
	}
}

// S4: the same Thought ID under two owners. Each answer is the current owner's record, attested as such.
func TestCognitionScopeSameIDAcrossOwners(t *testing.T) {
	env := scopeServer(t)
	for _, who := range []string{"A", "B", "A"} {
		env.ids.set("UNLOCKED", who)
		rec := env.get("/api/cognition/thoughts/thought_1", nil)
		other := map[string]string{"A": "B", "B": "A"}[who]
		if rec.Code != 200 || !strings.Contains(rec.Body.String(), scopeOwners[who].marker) || strings.Contains(rec.Body.String(), scopeOwners[other].marker) {
			t.Fatalf("owner %s: HTTP %d, body %s", who, rec.Code, rec.Body.String())
		}
		if rec.Header().Get("X-Cognition-Identity") != scopeOwners[who].id || rec.Header().Get("X-Cognition-Engine") != scopeOwners[who].engine {
			t.Errorf("owner %s: attested %q / %q", who, rec.Header().Get("X-Cognition-Identity"), rec.Header().Get("X-Cognition-Engine"))
		}
	}
}

// S6: a page's expected scope only NARROWS a request. A mismatch is refused with a distinct code,
// before any Engine read, with no record payload; a matching expectation never grants anything.
func TestCognitionExpectedScopeNarrowsReads(t *testing.T) {
	env := scopeServer(t)
	for _, rd := range scopedReads {
		if rec := env.get(rd.path, env.expectOf("A")); rec.Code != 200 {
			t.Fatalf("%s with the current scope expected: HTTP %d %s", rd.name, rec.Code, rec.Body.String())
		}
		stale := map[string]map[string]string{"other owner": env.expectOf("B"), "other link": env.expectOf("A"), "unlinked": env.expectOf("A")}
		stale["other link"]["X-Cognition-Expect-Link"] = "dec_z0000000"
		stale["unlinked"]["X-Cognition-Expect-Engine"] = "UNLINKED"
		stale["unlinked"]["X-Cognition-Expect-Link"] = "NONE"
		for name, hdr := range stale {
			before := env.eng.hit("/api/entities/eng_1/thoughts/thought_1") + env.eng.hit("/api/cycles/tp_a/journal") + env.eng.hit("/api/pulse")
			rec := env.get(rd.path, hdr)
			if rec.Code != http.StatusPreconditionFailed || scopeCode(rec) != "SCOPE_MISMATCH" {
				t.Errorf("%s expecting %s: HTTP %d code %q, want 412 SCOPE_MISMATCH", rd.name, name, rec.Code, scopeCode(rec))
			}
			noPrivate(t, rd.name+" expecting "+name, rec)
			if after := env.eng.hit("/api/entities/eng_1/thoughts/thought_1") + env.eng.hit("/api/cycles/tp_a/journal") + env.eng.hit("/api/pulse"); after != before {
				t.Errorf("%s expecting %s: the record was read from the Engine before the scope was refused", rd.name, name)
			}
		}
	}
	// Malformed precondition: refused, never ignored.
	bad := env.expectOf("A")
	bad["X-Cognition-Expect-Identity"] = "ent_a\"><script>"
	if rec := env.get("/api/cognition/thoughts/thought_1", bad); rec.Code != http.StatusBadRequest && rec.Code != http.StatusPreconditionFailed {
		t.Errorf("malformed expected scope: HTTP %d, want 400 or 412", rec.Code)
	}
	// A matching expectation is not a grant: a LOCKED Identity still answers 403 with no attestation.
	env.ids.set("LOCKED", "A")
	for _, rd := range scopedReads {
		rec := env.get(rd.path, env.expectOf("A"))
		if rec.Code != http.StatusForbidden {
			t.Errorf("LOCKED %s with the right expected scope: HTTP %d, want 403", rd.name, rec.Code)
		}
		noPrivate(t, "LOCKED "+rd.name, rec)
	}
}

// Contract v2 §2 response boundary: Identity changes WHILE Core is waiting for the Engine. Core must
// not answer with the old owner's data (or the old owner's attestation) after it can see the switch.
func TestCognitionScopeChangedDuringHeldEngineRead(t *testing.T) {
	for _, rd := range scopedReads {
		for _, variant := range []string{"switch to B, no precondition", "switch to B, precondition A", "lock", "unlink"} {
			if rd.name != "thought" && (variant == "lock" || variant == "unlink") {
				continue
			}
			t.Run(rd.name+"/"+variant, func(t *testing.T) {
				env := scopeServer(t)
				var hdr map[string]string
				if variant == "switch to B, precondition A" {
					hdr = env.expectOf("A")
				}
				entered, release := env.eng.hold(rd.holdAt, rd.holdSkip)
				done := make(chan *httptest.ResponseRecorder, 1)
				go func() { done <- env.get(rd.path, hdr) }()
				select {
				case <-entered:
				case <-time.After(5 * time.Second):
					t.Fatalf("the read never reached the held Engine path %s", rd.holdAt)
				}
				switch variant {
				case "lock":
					env.ids.set("LOCKED", "A")
				case "unlink":
					if rec := env.post("/api/cognition/unlink", `{}`, nil); rec.Code != 200 {
						t.Fatalf("unlink during the held read: HTTP %d", rec.Code)
					}
				default:
					env.ids.set("UNLOCKED", "B")
				}
				close(release)
				rec := <-done
				want := http.StatusPreconditionFailed
				if variant == "lock" {
					want = http.StatusForbidden
				}
				if rec.Code != want || (want == 412 && scopeCode(rec) != "SCOPE_CHANGED") {
					t.Errorf("HTTP %d code %q, want %d%s; body %.120s", rec.Code, scopeCode(rec), want,
						map[bool]string{true: " SCOPE_CHANGED", false: ""}[want == 412], rec.Body.String())
				}
				noPrivate(t, variant, rec)
			})
		}
	}
}

// W4/W6 server side: an owner POST that carries a stale expected scope writes nothing.
func TestCognitionScopeMismatchedPOSTWritesNothing(t *testing.T) {
	env := scopeServer(t)
	journal := filepath.Join(env.dir, "nudges.jsonl")
	links := filepath.Join(env.dir, "cognition_links.json")
	linksBefore, _ := os.ReadFile(links)
	draft := `{"source_thought_id":"thought_1","recipient_id":"self","reason_code":"follow_up","requested_action":"Look again.","idempotency_key":"scope-key-0001"}`
	for name, call := range map[string]func(map[string]string) *httptest.ResponseRecorder{
		"save draft": func(h map[string]string) *httptest.ResponseRecorder { return env.post("/api/nudges", draft, h) },
		"process": func(h map[string]string) *httptest.ResponseRecorder {
			return env.post("/api/nudges/nudge_00000000000000000001/process", `{"body_digest":"x"}`, h)
		},
		"stop": func(h map[string]string) *httptest.ResponseRecorder {
			return env.post("/api/nudges/nudge_00000000000000000001/stop", `{}`, h)
		},
		"link": func(h map[string]string) *httptest.ResponseRecorder {
			return env.post("/api/cognition/link", `{"engine_entity_id":"eng_1"}`, h)
		},
		"unlink": func(h map[string]string) *httptest.ResponseRecorder {
			return env.post("/api/cognition/unlink", `{}`, h)
		},
	} {
		for who, hdr := range map[string]map[string]string{"owner B's scope": env.expectOf("B"), "a stale link": func() map[string]string {
			h := env.expectOf("A")
			h["X-Cognition-Expect-Link"] = "dec_z0000000"
			return h
		}()} {
			rec := call(hdr)
			if rec.Code != http.StatusPreconditionFailed || scopeCode(rec) != "SCOPE_MISMATCH" {
				t.Errorf("%s expecting %s while A is current: HTTP %d code %q, want 412 SCOPE_MISMATCH", name, who, rec.Code, scopeCode(rec))
			}
		}
	}
	if _, err := os.Stat(journal); !os.IsNotExist(err) {
		t.Error("a refused POST wrote the nudge journal")
	}
	if now, _ := os.ReadFile(links); string(now) != string(linksBefore) {
		t.Error("a refused Link/Unlink changed the owner's links")
	}
	if len(env.eng.nonGET) != 0 {
		t.Errorf("Engine writes: %v", env.eng.nonGET)
	}
	// The same draft with the CURRENT scope is saved once (precondition met; the grant still decides).
	if rec := env.post("/api/nudges", draft, env.expectOf("A")); rec.Code != http.StatusCreated {
		t.Fatalf("save with the current scope: HTTP %d %s", rec.Code, rec.Body.String())
	}
}

// v2 §2: the response-boundary check must not add audited Identity decisions to every read (the
// pulse polls every 10 s). Measured: Identity decisions per read, and read-only state requests.
func TestCognitionScopeAddsNoIdentityDecisions(t *testing.T) {
	env := scopeServer(t)
	for _, rd := range scopedReads {
		a0, s0 := env.ids.counts()
		if rec := env.get(rd.path, env.expectOf("A")); rec.Code != 200 {
			t.Fatalf("%s: HTTP %d", rd.name, rec.Code)
		}
		a1, s1 := env.ids.counts()
		t.Logf("%-11s Identity decisions %d, read-only state requests %d", rd.name, a1-a0, s1-s0)
		if a1-a0 != 2 {
			t.Errorf("%s: %d Identity decisions per read, want 2 (unchanged)", rd.name, a1-a0)
		}
		if s1-s0 > 3 {
			t.Errorf("%s: %d Identity state requests per read", rd.name, s1-s0)
		}
	}
}
