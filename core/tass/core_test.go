package tass

// Evidence tests for the core (TEST FIXTURES: a fake GitHub API and a fake Thought Engine;
// no real model, no real GitHub). They cover the four acceptance points: local-first thinking,
// tass.md authority and conflicts, the separate GitHub sync with backoff, and start/stop/outage
// recovery without losing local state or skipping Identity.

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
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
)

const policyJSON = `{"identity":"TASS","owner":"Arameair","obey":["Arameair"],"tass_md":"Arameair/TASS@live:tass.md",
"allow":{"model.request":true,"thought.process":true,"action.read_tass":true,"github.read_messages":true,"github.reply":true,"tass_md.sync":true,"tass_md.conflict_pr":true}}`

const startDoc = "# TASS\n\n## Who I am\n\nI am TASS.\n\n## Memory\n\n- 2026-11-03: Dustin likes short answers\n\n## Now\n\nStarting.\n"

// ---- fake GitHub ---------------------------------------------------------------

type ghFile struct {
	text string
	ver  int
}

type fakeGH struct {
	mu        sync.Mutex
	files     map[string]*ghFile // "repo@branch:path"
	branches  map[string]bool    // "repo@branch"
	issues    []github.Issue
	comments  []github.Comment
	posted    []string // "issue#n: body"
	prs       []string
	treeCalls int
	nextID    int64
	srv       *httptest.Server
}

func newFakeGH() *fakeGH {
	f := &fakeGH{files: map[string]*ghFile{}, branches: map[string]bool{"Arameair/TASS@live": true, "Arameair/TASS@alpha": true}, nextID: 100}
	f.srv = httptest.NewServer(http.HandlerFunc(f.serve))
	return f
}

func (f *fakeGH) set(key, text string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if old, ok := f.files[key]; ok {
		old.text, old.ver = text, old.ver+1
		return
	}
	f.files[key] = &ghFile{text: text, ver: 1}
}

func (f *fakeGH) get(key string) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	if x, ok := f.files[key]; ok {
		return x.text
	}
	return ""
}

func (f *fakeGH) json(w http.ResponseWriter, code int, v any) {
	b, _ := json.Marshal(v)
	etag := fmt.Sprintf(`"%x"`, len(b)*31+int(b[len(b)/2]))
	w.Header().Set("ETag", etag)
	w.WriteHeader(code)
	w.Write(b)
}

func (f *fakeGH) serve(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	p := r.URL.Path
	switch {
	case r.Method == "GET" && strings.Contains(p, "/contents/"):
		parts := strings.SplitN(strings.TrimPrefix(p, "/repos/"), "/contents/", 2)
		key := parts[0] + "@" + r.URL.Query().Get("ref") + ":" + parts[1]
		x, ok := f.files[key]
		if !ok {
			http.Error(w, `{"message":"Not Found"}`, 404)
			return
		}
		etag := fmt.Sprintf(`"v%d-%d"`, x.ver, len(x.text))
		if r.Header.Get("If-None-Match") == etag {
			w.WriteHeader(304)
			return
		}
		w.Header().Set("ETag", etag)
		json.NewEncoder(w).Encode(map[string]string{"content": base64.StdEncoding.EncodeToString([]byte(x.text)),
			"encoding": "base64", "sha": fmt.Sprintf("sha%d", x.ver)})
	case r.Method == "GET" && strings.Contains(p, "/git/trees/"):
		parts := strings.SplitN(strings.TrimPrefix(p, "/repos/"), "/git/trees/", 2)
		prefix := parts[0] + "@" + parts[1] + ":"
		tree := []map[string]string{}
		sig := ""
		for k, v := range f.files {
			if strings.HasPrefix(k, prefix) {
				tree = append(tree, map[string]string{"path": strings.TrimPrefix(k, prefix), "type": "blob", "sha": fmt.Sprintf("sha%d", v.ver)})
				sig += k + fmt.Sprint(v.ver)
			}
		}
		etag := fmt.Sprintf(`"t%x"`, hash(sig)[:8])
		f.treeCalls++
		if r.Header.Get("If-None-Match") == etag {
			w.WriteHeader(304)
			return
		}
		w.Header().Set("ETag", etag)
		json.NewEncoder(w).Encode(map[string]any{"tree": tree})
	case r.Method == "PUT" && strings.Contains(p, "/contents/"):
		var b struct{ Message, Content, SHA, Branch string }
		json.NewDecoder(r.Body).Decode(&b)
		parts := strings.SplitN(strings.TrimPrefix(p, "/repos/"), "/contents/", 2)
		key := parts[0] + "@" + b.Branch + ":" + parts[1]
		x, ok := f.files[key]
		if ok && b.SHA != fmt.Sprintf("sha%d", x.ver) {
			http.Error(w, `{"message":"sha mismatch"}`, 409)
			return
		}
		raw, _ := base64.StdEncoding.DecodeString(b.Content)
		if !ok {
			x = &ghFile{}
			f.files[key] = x
		}
		x.text, x.ver = string(raw), x.ver+1
		f.json(w, 200, map[string]any{"content": map[string]string{"sha": fmt.Sprintf("sha%d", x.ver)}})
	case r.Method == "GET" && strings.Contains(p, "/git/ref/heads/"):
		parts := strings.SplitN(strings.TrimPrefix(p, "/repos/"), "/git/ref/heads/", 2)
		if !f.branches[parts[0]+"@"+parts[1]] {
			http.Error(w, `{}`, 404)
			return
		}
		f.json(w, 200, map[string]any{"object": map[string]string{"sha": "tip-" + parts[1]}})
	case r.Method == "POST" && strings.HasSuffix(p, "/git/refs"):
		var b struct{ Ref, SHA string }
		json.NewDecoder(r.Body).Decode(&b)
		repo := strings.TrimSuffix(strings.TrimPrefix(p, "/repos/"), "/git/refs")
		br := strings.TrimPrefix(b.Ref, "refs/heads/")
		f.branches[repo+"@"+br] = true
		from := strings.TrimPrefix(b.SHA, "tip-")
		for k, v := range f.files { // copy the files of the source branch
			if strings.HasPrefix(k, repo+"@"+from+":") {
				f.files[repo+"@"+br+":"+strings.SplitN(k, ":", 2)[1]] = &ghFile{text: v.text, ver: v.ver}
			}
		}
		f.json(w, 201, map[string]string{"ref": b.Ref})
	case r.Method == "POST" && strings.HasSuffix(p, "/pulls"):
		var b struct{ Head, Base, Title, Body string }
		json.NewDecoder(r.Body).Decode(&b)
		f.prs = append(f.prs, b.Head+" -> "+b.Base+": "+b.Title)
		f.json(w, 201, map[string]string{"html_url": "https://github.test/pr/" + fmt.Sprint(len(f.prs))})
	case r.Method == "GET" && strings.HasSuffix(p, "/issues"):
		f.json(w, 200, f.issues)
	case r.Method == "GET" && strings.HasSuffix(p, "/issues/comments"):
		f.json(w, 200, f.comments)
	case r.Method == "POST" && strings.HasSuffix(p, "/comments"):
		var b struct{ Body string }
		json.NewDecoder(r.Body).Decode(&b)
		n := strings.Split(p, "/")[5]
		f.posted = append(f.posted, "issue#"+n+": "+b.Body)
		f.nextID++
		f.comments = append(f.comments, github.Comment{ID: f.nextID, Body: b.Body, User: github.User{Login: "Arameair"},
			IssueURL: "https://api.github.test/repos/Arameair/TASS/issues/" + n, CreatedAt: time.Now().UTC()})
		f.json(w, 201, map[string]string{})
	default:
		http.Error(w, `{"message":"not in fake"}`, 404)
	}
}

func (f *fakeGH) say(issue int, who, body string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.nextID++
	f.comments = append(f.comments, github.Comment{ID: f.nextID, Body: body, User: github.User{Login: who},
		IssueURL: fmt.Sprintf("https://api.github.test/repos/Arameair/TASS/issues/%d", issue), CreatedAt: time.Now().UTC().Add(time.Second)})
}

// ---- fake Thought Engine -------------------------------------------------------
// (The real engine's cycle is tested in the Thought-engine repo; this stands in for its HTTP contract.)

type fakeEngine struct {
	control            string
	jobState           map[string]map[string]any
	mu                 sync.Mutex
	reqs               []map[string]any
	outcomes           []map[string]any
	pulse              []string
	jobs               []map[string]any
	srv                *httptest.Server
	mismatchCycleID    string
	unlockedPrediction bool
	unreadableProposal bool
	conflicts          int // 409 answers to duplicate job IDs
	replyExtra         string
}

func newFakeEngine() *fakeEngine {
	e := &fakeEngine{}
	pred := map[string]float64{}
	e.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/model" {
			json.NewEncoder(w).Encode(map[string]any{"last_pulse": map[string]any{"at": time.Now(), "model_ok": true}})
			return
		}
		if r.Header.Get("Authorization") != "Bearer testkey" {
			http.Error(w, `{"error":"missing or wrong engine key"}`, 401)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/pulse") {
			e.mu.Lock()
			e.pulse = append(e.pulse, r.URL.Path)
			e.mu.Unlock()
			e.mu.Lock()
			defer e.mu.Unlock()
			if e.control == "" {
				e.control = "PAUSED"
			}
			if r.Method == "GET" {
				jobs := []map[string]any{}
				for _, j := range e.jobs {
					if st, ok := e.jobState[j["job_id"].(string)]; ok {
						jobs = append(jobs, st)
					} else {
						jobs = append(jobs, map[string]any{"job_id": j["job_id"], "state": "QUEUED"})
					}
				}
				json.NewEncoder(w).Encode(map[string]any{"control": e.control, "state": "READY", "jobs": jobs})
				return
			}
			var in map[string]any
			json.NewDecoder(r.Body).Decode(&in)
			switch r.URL.Path {
			case "/api/pulse/jobs":
				// Like the real Engine (pulse.go Submit): a durable job ID is never accepted twice.
				for _, j := range e.jobs {
					if j["job_id"] == in["job_id"] {
						e.conflicts++
						w.WriteHeader(409)
						json.NewEncoder(w).Encode(map[string]any{"error": fmt.Sprintf("conflict: job %v already exists", in["job_id"]), "denied": false})
						return
					}
				}
				e.jobs = append(e.jobs, in)
				w.WriteHeader(202)
			case "/api/pulse/control":
				switch in["action"] {
				case "start", "resume":
					e.control = "RUNNING"
				case "pause":
					e.control = "PAUSED"
				}
			}
			json.NewEncoder(w).Encode(in)
			return
		}
		var in map[string]any
		b, _ := io.ReadAll(r.Body)
		json.Unmarshal(b, &in)
		e.mu.Lock()
		defer e.mu.Unlock()
		if strings.HasSuffix(r.URL.Path, "/outcome") {
			id := strings.Split(r.URL.Path, "/")[3]
			e.outcomes = append(e.outcomes, in)
			ev := map[string]any{"forecast": "NO_ACTION", "goal": "GOAL_UNKNOWN", "evaluator": "fixture"}
			if v, ok := in["value"].(float64); ok {
				ev["forecast"], ev["goal"] = "PREDICTION_INCORRECT", "GOAL_MET"
				if v == pred[id] {
					ev["forecast"] = "PREDICTION_CORRECT"
				}
			} else if in["status"] != "NO_ACTION" {
				ev["forecast"], ev["goal"] = "UNSCORABLE", "GOAL_UNMET"
			}
			json.NewEncoder(w).Encode(map[string]any{"cycle_id": id, "state": "evaluated", "outcome": in, "evaluation": ev,
				"proposal": map[string]any{"action": "x"}})
			return
		}
		e.reqs = append(e.reqs, in)
		id, last := in["cycle_id"].(string), in["request"].(string)
		reply := "TEST FIXTURE: heard " + last
		if e.replyExtra != "" {
			reply += " " + e.replyExtra
		}
		prop := map[string]any{"reply": reply, "action": "NONE", "why": "fixture", "format": "READ",
			"prediction": map[string]any{"state": "NOT_PREDICTED"}}
		if strings.Contains(last, "remember") {
			prop["remember"] = strings.TrimSpace(strings.SplitN(last, "remember", 2)[1])
			prop["reply"] = reply + "\nNOW: talking with Dustin"
		}
		if strings.Contains(last, "sections") {
			pred[id] = 6
			prop["action"], prop["prediction"] = "tass_sections", map[string]any{
				"state": "LOCKED", "value": 6, "locked_at": time.Now(), "sha256": strings.Repeat("a", 64)}
		}
		if e.unlockedPrediction {
			prop["prediction"] = map[string]any{"state": "NOT_PREDICTED"}
		}
		if e.unreadableProposal {
			prop["format"] = "UNREADABLE"
		}
		responseID := id
		if e.mismatchCycleID != "" {
			responseID = e.mismatchCycleID
		}
		json.NewEncoder(w).Encode(map[string]any{"cycle_id": responseID, "state": "awaiting_outcome", "request": last,
			"spark": map[string]any{"status": "done", "rating": 7, "topic": "fixture"}, "proposal": prop})
	}))
	return e
}

func (e *fakeEngine) count() int { e.mu.Lock(); defer e.mu.Unlock(); return len(e.reqs) }

// ---- helpers ---------------------------------------------------------------------

func newCore(t *testing.T, dir string, gh *fakeGH, eng *fakeEngine) *Core {
	t.Helper()
	cl := github.New("test-token")
	cl.Base = gh.srv.URL
	ec := engine.New(eng.srv.URL)
	ec.Key = "testkey"
	c, err := New(Config{DataDir: dir, Identity: "Arameair/identity@main:policies/tass.json", Label: "tass",
		SyncEvery: 20 * time.Second, MaxBackoff: 5 * time.Minute, Messages: true}, cl, ec)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func local(t *testing.T, dir string) string {
	b, err := os.ReadFile(filepath.Join(dir, "tass.md"))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func offline(c *Core)            { c.gh.Base = "http://127.0.0.1:1" } // nothing listens there
func online(c *Core, gh *fakeGH) { c.gh.Base = gh.srv.URL }

// ---- tests ------------------------------------------------------------------------

func TestNoIdentityPolicyMeansNoAction(t *testing.T) {
	gh, eng := newFakeGH(), newFakeEngine()
	gh.set("Arameair/TASS@live:tass.md", startDoc)
	dir := t.TempDir()
	c := newCore(t, dir, gh, eng)
	if wait := c.SyncOnce(); wait != 20*time.Second {
		// first failure: retry after the normal interval
		t.Fatalf("first retry after %v", wait)
	}
	if _, err := os.Stat(filepath.Join(dir, "tass.md")); err == nil {
		t.Fatal("tass.md was synced without Identity's policy")
	}
	c.Answer(Incoming{Text: "hello", From: "local"})
	if eng.count() != 0 {
		t.Fatal("the model was asked without Identity's model.request permission")
	}
	if st := c.Status(); !strings.Contains(st.SyncProblem, "Identity hasn't authorized") || !strings.Contains(st.Policy, "none yet") {
		t.Fatalf("status: %+v", st)
	}
}

func memories(t *testing.T, dir string) string {
	b, _ := os.ReadFile(filepath.Join(dir, "memory.jsonl"))
	return string(b)
}

func TestThoughtCycleLoopAndGitHubMessages(t *testing.T) {
	gh, eng := newFakeGH(), newFakeEngine()
	gh.set("Arameair/identity@main:policies/tass.json", policyJSON)
	gh.set("Arameair/TASS@live:tass.md", startDoc)
	gh.issues = []github.Issue{{Number: 1, Title: "old", Body: "from before", User: github.User{Login: "Arameair"}}}
	dir := t.TempDir()
	c := newCore(t, dir, gh, eng)
	c.SyncOnce()
	if local(t, dir) != startDoc {
		t.Fatal("first sync should take tass.md from GitHub")
	}
	g, err := os.ReadFile(filepath.Join(dir, GrantFile))
	if err != nil || !strings.Contains(string(g), `"prototype": true`) || !strings.Contains(string(g), "fetched_at") {
		t.Fatalf("the grant file must be kept locally, labelled PROTOTYPE, with when it was confirmed: %s", g)
	}
	// A request becomes one cycle through the engine; the proposed action runs because Identity allows it now.
	c.Answer(Incoming{Text: "how many sections does your tass.md have?", From: "local"})
	if eng.count() != 1 || eng.reqs[0]["identity"] != "TASS" || !strings.HasPrefix(eng.reqs[0]["cycle_id"].(string), "tp_") {
		t.Fatalf("engine cycle: %v", eng.reqs)
	}
	if !strings.Contains(eng.reqs[0]["context"].(string), "I am TASS.") || len(eng.reqs[0]["actions"].([]any)) != 3 {
		t.Fatal("the cycle must carry tass.md and the offered actions")
	}
	o := eng.outcomes[0]
	if o["status"] != "EXECUTED" || o["value"] != 3.0 || !strings.Contains(o["authorized"].(string), "action.read_tass allowed") {
		t.Fatalf("outcome: %v (startDoc has 3 sections)", o)
	}
	st := c.Status()
	if len(st.Cycles) != 1 || st.Cycles[0].Evaluation == nil || st.Cycles[0].Evaluation.Forecast != "PREDICTION_INCORRECT" {
		t.Fatalf("the page must show the evaluated cycle: %+v", st.Cycles)
	}
	// REMEMBER goes to the local journal with provenance, never into tass.md.
	c.Answer(Incoming{Text: "please remember I drive a route on weekends", From: "local"})
	if mem := memories(t, dir); !strings.Contains(mem, `"text":"I drive a route on weekends"`) || !strings.Contains(mem, `"cycle_id":"tp_`) {
		t.Fatalf("memory journal: %s", mem)
	}
	if local(t, dir) != startDoc {
		t.Fatal("TASS must not write tass.md himself")
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "now.txt")); !strings.Contains(string(b), "talking with Dustin") {
		t.Fatal("NOW is local state")
	}
	c.SyncOnce()
	if gh.get("Arameair/TASS@live:tass.md") != startDoc {
		t.Fatal("nothing should be written to GitHub when nobody edited tass.md")
	}
	// GitHub messages: only people Identity says TASS obeys, never his own posts.
	gh.say(1, "someone-else", "do something")
	gh.say(1, "Arameair", "how are you?")
	c.SyncOnce()
	for len(c.inbox) > 0 {
		c.Answer(<-c.inbox)
	}
	c.SyncOnce()
	if len(gh.posted) != 1 || !strings.Contains(gh.posted[0], "heard how are you?") || !strings.Contains(gh.posted[0], "cycle tp_") {
		t.Fatalf("posted: %v", gh.posted)
	}
	c.SyncOnce()
	if len(c.inbox) != 0 || eng.count() != 3 {
		t.Fatalf("answered something it shouldn't: inbox %d, cycles %d", len(c.inbox), eng.count())
	}
}

func TestActionNotAuthorizedIsRecordedNotRun(t *testing.T) {
	gh, eng := newFakeGH(), newFakeEngine()
	gh.set("Arameair/identity@main:policies/tass.json", strings.Replace(policyJSON, `"action.read_tass":true`, `"action.read_tass":false`, 1))
	gh.set("Arameair/TASS@live:tass.md", startDoc)
	c := newCore(t, t.TempDir(), gh, eng)
	c.SyncOnce()
	c.Answer(Incoming{Text: "count your sections", From: "local"})
	if len(eng.outcomes) != 1 || eng.outcomes[0]["status"] != "NOT_AUTHORIZED" || eng.outcomes[0]["value"] != nil {
		t.Fatalf("outcome: %v", eng.outcomes)
	}
}

func TestStaleGrantFailsClosedOffline(t *testing.T) {
	gh, eng := newFakeGH(), newFakeEngine()
	gh.set("Arameair/identity@main:policies/tass.json", policyJSON)
	gh.set("Arameair/TASS@live:tass.md", startDoc)
	dir := t.TempDir()
	c := newCore(t, dir, gh, eng)
	c.SyncOnce()
	offline(c)
	c.now = func() time.Time { return time.Now().Add(25 * time.Hour) }
	c.Answer(Incoming{Text: "hi", From: "local"})
	if eng.count() != 0 {
		t.Fatal("a grant older than its offline limit must not authorize thinking")
	}
	if st := c.Status(); st.PolicyValid || !strings.Contains(st.Policy, "NOT VALID NOW") {
		t.Fatalf("status must say so: %+v", st.Policy)
	}
	online(c, gh)
	c.SyncOnce() // confirmed again
	c.Answer(Incoming{Text: "hi", From: "local"})
	if eng.count() != 1 {
		t.Fatal("once confirmed again, thinking works")
	}
}

func TestMessagesCanBePaused(t *testing.T) {
	gh, eng := newFakeGH(), newFakeEngine()
	gh.set("Arameair/identity@main:policies/tass.json", policyJSON)
	gh.set("Arameair/TASS@live:tass.md", startDoc)
	c := newCore(t, t.TempDir(), gh, eng)
	c.cfg.Messages = false
	c.SyncOnce()
	gh.say(1, "Arameair", "hello?")
	c.SyncOnce()
	if len(c.inbox) != 0 {
		t.Fatal("with messages paused, GitHub messages must not be read")
	}
}

func TestOutageBackoffAndRecoveryKeepLocalState(t *testing.T) {
	gh, eng := newFakeGH(), newFakeEngine()
	gh.set("Arameair/identity@main:policies/tass.json", policyJSON)
	gh.set("Arameair/TASS@live:tass.md", startDoc)
	dir := t.TempDir()
	c := newCore(t, dir, gh, eng)
	c.SyncOnce()

	offline(c)
	waits := []time.Duration{}
	for i := 0; i < 6; i++ {
		waits = append(waits, c.SyncOnce())
	}
	want := []time.Duration{20 * time.Second, 40 * time.Second, 80 * time.Second, 160 * time.Second, 5 * time.Minute, 5 * time.Minute}
	if fmt.Sprint(waits) != fmt.Sprint(want) {
		t.Fatalf("backoff %v, want %v", waits, want)
	}
	// Still thinks and remembers with no GitHub (the model is local).
	c.Answer(Incoming{Text: "remember the internet is down", From: "local"})
	if !strings.Contains(memories(t, dir), "the internet is down") {
		t.Fatal("TASS must keep remembering offline")
	}
	if !strings.Contains(eng.reqs[0]["context"].(string), "not reachable right now") {
		t.Fatal("TASS should know he's running locally")
	}
	// Dustin edits tass.md on this computer (Now) and on GitHub (Who I am) while apart.
	os.WriteFile(filepath.Join(dir, "tass.md"), []byte(strings.Replace(startDoc, "Starting.", "Working offline.", 1)), 0o600)
	gh.set("Arameair/TASS@live:tass.md", strings.Replace(startDoc, "I am TASS.", "I am TASS, Dustin's companion.", 1))

	// Stop and start again while still offline: nothing local is lost, permissions are still known.
	c2 := newCore(t, dir, gh, eng)
	offline(c2)
	if ok, _ := c2.allowedNow("model.request"); !ok || !strings.Contains(memories(t, dir), "the internet is down") {
		t.Fatal("restart lost the policy or the memory journal")
	}
	c2.SyncOnce()
	online(c2, gh)
	if wait := c2.SyncOnce(); wait != 20*time.Second {
		t.Fatalf("after recovery the interval should be normal again, got %v", wait)
	}
	doc := gh.get("Arameair/TASS@live:tass.md")
	if !strings.Contains(doc, "Dustin's companion") || !strings.Contains(doc, "Working offline.") {
		t.Fatalf("both sides' edits should be kept:\n%s", doc)
	}
	if local(t, dir) != doc || len(gh.prs) != 0 {
		t.Fatalf("local and GitHub should agree with no conflict (prs %v)", gh.prs)
	}
	b, _ := os.ReadFile(filepath.Join(dir, "sync.jsonl"))
	if n := strings.Count(string(b), `"ok":false`); n < 7 {
		t.Fatalf("every failed sync should be recorded locally, got %d", n)
	}
}

func TestSameSectionChangedOnBothSidesAsksDustin(t *testing.T) {
	gh, eng := newFakeGH(), newFakeEngine()
	gh.set("Arameair/identity@main:policies/tass.json", policyJSON)
	gh.set("Arameair/TASS@live:tass.md", startDoc)
	dir := t.TempDir()
	c := newCore(t, dir, gh, eng)
	c.SyncOnce()
	offline(c)
	os.WriteFile(filepath.Join(dir, "tass.md"), []byte(strings.Replace(startDoc, "I am TASS.", "I am TASS (local edit).", 1)), 0o600)
	gh.set("Arameair/TASS@live:tass.md", strings.Replace(startDoc, "I am TASS.", "I am TASS (GitHub edit).", 1))
	online(c, gh)
	c.SyncOnce()
	if len(gh.prs) != 1 || !strings.Contains(gh.prs[0], "-> live") || !strings.Contains(gh.prs[0], "Who I am") {
		t.Fatalf("expected one pull request into live: %v", gh.prs)
	}
	if !strings.Contains(local(t, dir), "GitHub edit") {
		t.Fatal("until Dustin decides, TASS runs on the GitHub version")
	}
	var mine string
	for k, v := range gh.files {
		if strings.Contains(k, "@tass/conflict-") {
			mine = v.text
		}
	}
	if !strings.Contains(mine, "local edit") {
		t.Fatal("the pull request must carry the local version")
	}
	snaps, _ := filepath.Glob(filepath.Join(dir, "conflicts", "*"))
	both := ""
	for _, p := range snaps {
		b, _ := os.ReadFile(p)
		both += string(b)
	}
	if len(snaps) != 2 || !strings.Contains(both, "local edit") || !strings.Contains(both, "GitHub edit") {
		t.Fatalf("both sides must also be kept locally: %v", snaps)
	}
	c.SyncOnce()
	if len(gh.prs) != 1 {
		t.Fatal("no duplicate pull requests")
	}
}

func TestIdentityCanWithdrawPermission(t *testing.T) {
	gh, eng := newFakeGH(), newFakeEngine()
	gh.set("Arameair/identity@main:policies/tass.json", policyJSON)
	gh.set("Arameair/TASS@live:tass.md", startDoc)
	dir := t.TempDir()
	c := newCore(t, dir, gh, eng)
	c.SyncOnce()
	gh.set("Arameair/identity@main:policies/tass.json", strings.Replace(policyJSON, `"model.request":true`, `"model.request":false`, 1))
	c.SyncOnce()
	c.Answer(Incoming{Text: "hi", From: "local"})
	if eng.count() != 0 {
		t.Fatal("withdrawn permission must stop thinking")
	}
	gh.mu.Lock()
	delete(gh.files, "Arameair/identity@main:policies/tass.json")
	gh.mu.Unlock()
	c.SyncOnce()
	if _, err := os.Stat(filepath.Join(dir, GrantFile)); err == nil {
		t.Fatal("a removed policy must be removed locally too (the engine then refuses)")
	}
}

// finish marks a pulse job done with the entity's reply.
func (e *fakeEngine) finish(jobID, state, result, remember string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.jobState == nil {
		e.jobState = map[string]map[string]any{}
	}
	e.jobState[jobID] = map[string]any{"job_id": jobID, "state": state, "result": result, "remember": remember, "cycle_id": jobID + ":cycle"}
}
