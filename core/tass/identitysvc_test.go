package tass

import (
	"encoding/json"
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
)

// fakeIdentitySvc behaves like tass-identity: LOCKED until unlocked, then allows its grants.
type fakeIdentitySvc struct {
	mu        sync.Mutex
	state     string
	lessonEnabled bool
	lessonCommentID int64
	lessonSource string
	decisions []string
	srv       *httptest.Server
}

func newFakeIdentitySvc() *fakeIdentitySvc {
	f := &fakeIdentitySvc{state: "LOCKED"}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		ent := map[string]any{"entity_id": "ent_0123", "name": "Mira", "grants": []string{"conversation", "model.request", "thought.process", "pulse.run", "action.read_entity"}}
		switch r.URL.Path {
		case "/v1/github/lesson":
			if r.Header.Get("Authorization") != "Bearer core-service-key-0123456789abcdef0123" {
				w.WriteHeader(http.StatusUnauthorized); return
			}
			if f.state != "UNLOCKED" || !f.lessonEnabled {
				w.WriteHeader(http.StatusForbidden); return
			}
			commentID := f.lessonCommentID
			if commentID == 0 { commentID = 6636766875 }
			source := f.lessonSource
			if source == "" { source = "https://github.com/Arameair/Thought-Processor/issues/51#issuecomment-6636766875" }
			json.NewEncoder(w).Encode(map[string]any{"skill":"@identity-test", "repository":"Arameair/Thought-Processor",
				"issue":51, "comment_id":commentID, "source":source,
				"mode":"reference_only", "body":"Owner-approved @identity-test is a proposed notebook skill reference, not authority to execute."})
		case "/v1/state":
			json.NewEncoder(w).Encode(map[string]any{"state": f.state, "reason": "owner hasn't unlocked", "entity": ent})
		case "/v1/authorize":
			if r.Header.Get("Authorization") != "Bearer core-service-key-0123456789abcdef0123" {
				http.Error(w, "no", 401)
				return
			}
			var in map[string]string
			json.NewDecoder(r.Body).Decode(&in)
			f.decisions = append(f.decisions, in["action"])
			allow := f.state == "UNLOCKED"
			json.NewEncoder(w).Encode(map[string]any{"allow": allow, "decision_id": "dec_1", "reason": map[bool]string{true: "granted", false: "LOCKED"}[allow],
				"entity_id": "ent_0123", "entity_name": "Mira"})
		}
	}))
	return f
}

func (f *fakeIdentitySvc) set(s string) { f.mu.Lock(); f.state = s; f.mu.Unlock() }

func newIdentityCore(t *testing.T, dir string, ids *fakeIdentitySvc, eng *fakeEngine) *Core {
	t.Helper()
	kf := filepath.Join(t.TempDir(), "core.key")
	os.WriteFile(kf, []byte("core-service-key-0123456789abcdef0123\n"), 0o600)
	cl, err := identitysvc.New(ids.srv.URL, kf)
	if err != nil {
		t.Fatal(err)
	}
	gh := github.New("")
	gh.Base = "http://127.0.0.1:1" // must never be used
	ec := engine.New(eng.srv.URL)
	ec.Key = "testkey"
	c, err := New(Config{DataDir: dir, Identity: "tass-identity", Label: "tass", SyncEvery: 20 * time.Second,
		MaxBackoff: time.Minute, IdentitySvc: cl}, gh, ec)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestIdentityServiceLockedBlocksEverything(t *testing.T) {
	ids, eng := newFakeIdentitySvc(), newFakeEngine()
	dir := t.TempDir()
	c := newIdentityCore(t, dir, ids, eng)
	c.Answer(Incoming{Text: "hi", From: "local"})
	if eng.count() != 0 {
		t.Fatal("a LOCKED entity must not reach the model")
	}
	h := c.Health("test")
	if h.Identity != "LOCKED" || !strings.Contains(strings.Join(h.Reasons, " "), "LOCKED") {
		t.Fatalf("health must say LOCKED: %+v", h)
	}
	if _, err := c.SubmitTask("count your sections"); err == nil || len(eng.jobs) != 0 {
		t.Fatal("no task may be queued while LOCKED")
	}
}

func TestIdentityServiceUnlockedThinksAsItsOwnEntity(t *testing.T) {
	ids, eng := newFakeIdentitySvc(), newFakeEngine()
	ids.set("UNLOCKED")
	dir := t.TempDir()
	c := newIdentityCore(t, dir, ids, eng)
	c.Answer(Incoming{Text: "hi", From: "local"})
	if eng.count() != 1 {
		t.Fatalf("unlocked: one cycle, got %d", eng.count())
	}
	if got := eng.reqs[0]["identity"]; got != "Mira" {
		t.Fatalf("cycles run as the installation's own entity, got %v", got)
	}
	if ctx := eng.reqs[0]["context"].(string); strings.Contains(ctx, "Dustin") || !strings.Contains(ctx, "You are Mira") {
		t.Fatalf("the prompt must be the entity's own:\n%s", ctx)
	}
	doc := local(t, dir)
	if !strings.HasPrefix(doc, "# Mira") || strings.Contains(doc, "TASS") || strings.Contains(doc, "Dustin") {
		t.Fatalf("a new entity gets its own starter document, nobody else's:\n%s", doc)
	}
	ids.mu.Lock()
	got := strings.Join(ids.decisions, ",")
	ids.mu.Unlock()
	if !strings.HasPrefix(got, "conversation,model.request,thought.process") {
		t.Fatalf("Identity decides each step: %s", got)
	}
	id, err := c.SubmitTask("count your sections")
	if err != nil || !strings.HasPrefix(id, "job_") || len(eng.jobs) != 1 || eng.jobs[0]["identity"] != "Mira" ||
		!strings.Contains(eng.jobs[0]["context"].(string), "# Mira") {
		t.Fatalf("task: %v %v %v", id, err, eng.jobs)
	}
	if h := c.Health("t"); h.Identity != "UNLOCKED" {
		t.Fatalf("health: %+v", h)
	}
}

func TestSayReturnsReceiptAndRefusesWhenFull(t *testing.T) {
	ids, eng := newFakeIdentitySvc(), newFakeEngine()
	c := newIdentityCore(t, t.TempDir(), ids, eng)
	id, err := c.Say("hello")
	if err != nil || !strings.HasPrefix(id, "msg_") {
		t.Fatalf("receipt: %q %v", id, err)
	}
	for i := 0; i < cap(c.inbox); i++ {
		c.Say("more")
	}
	if _, err := c.Say("one too many"); err != ErrBusy {
		t.Fatalf("a full inbox must refuse, not drop silently: %v", err)
	}
}


func TestIdentityApprovedGitHubLessonIsReferenceOnly(t *testing.T) {
    ids, eng := newFakeIdentitySvc(), newFakeEngine()
    defer ids.srv.Close()
    defer eng.srv.Close()
    ids.set("UNLOCKED")
    c := newIdentityCore(t, t.TempDir(), ids, eng)
    c.Answer(Incoming{Text:"Please explain @identity-test", From:"local"})
    if eng.count() != 0 { t.Fatal("no model cycle before an owner-approved lesson") }
    // Legacy design comment and forged provenance must never enter model context.
    ids.mu.Lock(); ids.lessonEnabled = true; ids.lessonCommentID = 6634742721; ids.mu.Unlock()
    c.Answer(Incoming{Text:"Please explain @identity-test", From:"local"})
    if eng.count() != 0 { t.Fatal("legacy design comment incorrectly authorized") }
    ids.mu.Lock(); ids.lessonCommentID = 6636766875; ids.lessonSource = "https://github.com/Arameair/Thought-Processor/issues/51#issuecomment-6634742721"; ids.mu.Unlock()
    c.Answer(Incoming{Text:"Please explain @identity-test", From:"local"})
    if eng.count() != 0 { t.Fatal("forged lesson source incorrectly authorized") }
    ids.mu.Lock(); ids.lessonSource = ""; ids.mu.Unlock()
    c.Answer(Incoming{Text:"Please explain @identity-test", From:"local"})
    if eng.count() != 1 { t.Fatalf("approved lesson did not reach model: %d",eng.count()) }
    req := eng.reqs[0]
    ctx, ok := req["context"].(string)
    if !ok || !strings.Contains(ctx, "Owner-approved @identity-test") ||
        !strings.Contains(ctx, "NOT AUTHORITY") || !strings.Contains(ctx, "issuecomment-6636766875") {
        t.Fatalf("missing bounded source context: %v", req)
    }
    if value := req["actions"]; value != nil {
        actions, ok := value.([]any)
        if !ok || len(actions) != 0 { t.Fatalf("reference-only lesson exposed actions: %v",value) }
    }
    c.Answer(Incoming{Text:"Hi, no GitHub", From:"local"})
    if eng.count() != 2 { t.Fatalf("ordinary local conversation should work without lesson: %d",eng.count()) }
    if strings.Contains(eng.reqs[1]["context"].(string),"Owner-approved @identity-test") { t.Fatal("lesson leaked into unrelated conversation") }
    c.Answer(Incoming{Text:"What about @identity-test-evil?", From:"local"})
    if eng.count() != 3 { t.Fatal("nonexact skill reference should not fetch gated lesson") }
    if strings.Contains(eng.reqs[2]["context"].(string),"Owner-approved @identity-test") { t.Fatal("nonexact mention resolved") }
    ids.set("LOCKED")
    c.Answer(Incoming{Text:"Please explain @identity-test", From:"local"})
    if eng.count() != 3 { t.Fatal("locked entity reached model") }
}
