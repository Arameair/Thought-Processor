package tass

// P0 (#106, design PR #107 §3.1 / §7 A1–A11): what TASS wrote, shown to the unlocked, linked owner in
// the single-Thought detail only, correlated by request and nudge IDs; statuses derived, not stale.

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Arameair/Thought-Processor/core/engine"
)

const privateMarker = "PRIVATE-NOTE-MARKER-7f3a"

// A1/A2/A5/A6-part: a COMPLETED supervised nudge's Note is shown with the same request and nudge IDs
// as Core's verified receipt; reading it makes no model request and no Engine write.
func TestCognitionShowsWhatTASSWroteForAVerifiedNudge(t *testing.T) {
	c, _, re := runCore(t, t.TempDir())
	v := selfDraft(c, t, "key-p0-note-0001")
	if v.Processing.State != "READY" {
		t.Fatalf("a self draft with the grant held must be READY (derived): %+v", v.Processing)
	}
	if _, ok, err := c.ProcessNudge(v.NudgeID, v.BodyDigest); err != nil || !ok {
		t.Fatalf("process: %v", err)
	}
	run := waitRun(t, c, v.NudgeID, "COMPLETED")
	posts := re.posts
	th, err := c.CognitionThought("thought_1")
	if err != nil {
		t.Fatal(err)
	}
	if len(th.Developments) != 1 || th.DevelopmentsTotal != 1 {
		t.Fatalf("developments: %+v", th.Developments)
	}
	d := th.Developments[0]
	if d.Note == nil || d.Note.Text != "NOTE-TEXT-"+v.NudgeID || d.Note.By == "" || d.Note.SuggestedState != "resolve" || d.Note.SuggestedApplied {
		t.Fatalf("note: %+v", d.Note)
	}
	if d.RequestID != run.RequestID || d.SourceNudgeID != v.NudgeID || d.Persistence != "NOTE_AND_STEP_CORRELATED" || d.NudgeReceipt != "COMPLETED_VERIFIED" {
		t.Fatalf("correlation: %+v (receipt request %s)", d, run.RequestID)
	}
	if !strings.Contains(d.Meaning, "not that it answers") {
		t.Fatalf("meaning must separate saved from useful: %q", d.Meaning)
	}
	if len(th.History) != 1 || th.History[0].NudgeID != v.NudgeID || th.History[0].RequestID != run.RequestID || th.History[0].By != "owner-nudge" {
		t.Fatalf("history step must keep request and nudge IDs: %+v", th.History)
	}
	b, _ := json.Marshal(th)
	if strings.Contains(string(b), "DETAIL-REPEATS-NOTE") {
		t.Fatal("the step's free-form detail must stay out (the Note is shown once)")
	}
	if re.posts != posts {
		t.Fatalf("reading the Thought made a model request: %d -> %d", posts, re.posts)
	}
	// The receipt deep link target exists: the run's request ID equals the development's.
	list, _ := c.NudgeHistory("thought_1")
	if list.Nudges[0].Run == nil || list.Nudges[0].Run.RequestID != d.RequestID || list.Nudges[0].Processing.State != "SEE_RUN" {
		t.Fatalf("receipt ↔ development: %+v", list.Nudges[0])
	}
	// The Stage 2 draft check path (SaveNudgeDraft holds nudgeMu) must not deadlock on developments.
	done := make(chan struct{})
	go func() { selfDraft(c, t, "key-p0-note-0002"); close(done) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("SaveNudgeDraft deadlocked")
	}
}

// thoughtEngine: a read-only Engine stand-in serving one entity with a configurable Thought.
type thoughtEngine struct {
	mu      sync.Mutex
	thought map[string]any
	calls   []string
	srv     *httptest.Server
}

func newThoughtEngine(t *testing.T, thought map[string]any) *thoughtEngine {
	e := &thoughtEngine{thought: thought}
	e.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		e.mu.Lock()
		e.calls = append(e.calls, r.Method+" "+r.URL.Path)
		th := e.thought
		e.mu.Unlock()
		if r.Header.Get("Authorization") != "Bearer testkey" {
			w.WriteHeader(401)
			return
		}
		if r.Method != "GET" {
			w.WriteHeader(405)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/status":
			json.NewEncoder(w).Encode(map[string]any{"entities": []any{
				map[string]any{"card": map[string]string{"entity_id": "eng_1", "display_name": "Mira", "entity_type": "ai_companion"}},
				map[string]any{"card": map[string]string{"entity_id": "eng_other", "display_name": "Other", "entity_type": "ai_companion"}},
			}, "config": map[string]string{"commit": "21b3d502775b3ed065ab445b88f51909500770b3"}})
		case "/api/entities/eng_1":
			json.NewEncoder(w).Encode(map[string]any{"card": map[string]string{"entity_id": "eng_1", "display_name": "Mira"},
				"thoughts": []any{th}, "traces": []any{}})
		case "/api/entities/eng_1/thoughts/thought_1":
			json.NewEncoder(w).Encode(map[string]any{"thought": th})
		case "/api/entities/eng_1/thoughts/foreign_1":
			// A misbehaving Engine answering with another entity's Thought: must be refused.
			json.NewEncoder(w).Encode(map[string]any{"thought": map[string]any{"thought_id": "foreign_1", "entity_id": "eng_other",
				"question": "foreign", "notes": []any{map[string]any{"text": "FOREIGN-" + privateMarker, "worker_request_id": "mr_x"}}}})
		case "/api/cycles":
			json.NewEncoder(w).Encode(map[string]any{"cycles": []any{}})
		default:
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(e.srv.Close)
	return e
}

func (e *thoughtEngine) writes() []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	var out []string
	for _, c := range e.calls {
		if !strings.HasPrefix(c, "GET ") {
			out = append(out, c)
		}
	}
	return out
}

func thoughtCore(t *testing.T, th map[string]any) (*Core, *fakeIdentitySvc, *thoughtEngine) {
	ids, legacy := newFakeIdentitySvc(), newFakeEngine()
	t.Cleanup(func() { ids.srv.Close(); legacy.srv.Close() })
	c := newIdentityCore(t, t.TempDir(), ids, legacy)
	te := newThoughtEngine(t, th)
	c.engine = engine.New(te.srv.URL)
	c.engine.Key = "testkey"
	ids.set("UNLOCKED")
	if _, err := c.LinkCognition("eng_1"); err != nil {
		t.Fatal(err)
	}
	return c, ids, te
}

// A9/A10 + correlation edge cases: newest 64 of 70 steps, an evicted Note (STEP_ONLY), a Note with no
// step (NOTE_ONLY), a nudge ID not in this owner's journal, disagreeing IDs, an unsafe request ID.
func TestCognitionHistoryKeepsNewestAndCorrelationIsHonest(t *testing.T) {
	base := time.Date(2026, 11, 9, 10, 0, 0, 0, time.UTC)
	var hist, notes []any
	for i := 0; i < 70; i++ {
		hist = append(hist, map[string]any{"at": base.Add(time.Duration(i) * time.Minute), "kind": "reinforced", "trace_id": fmt.Sprintf("trace_%02d", i)})
	}
	// Development 1: note + step, nudge ID not in this Core's journal.
	hist = append(hist, map[string]any{"at": base.Add(80 * time.Minute), "kind": "developed", "worker_request_id": "mr_aaaa", "source_nudge_id": "nudge_00000000000000000001", "by": "owner-nudge"})
	notes = append(notes, map[string]any{"at": base.Add(80 * time.Minute), "text": "one " + privateMarker, "by": "Nemotron (owner-nudge)", "worker_request_id": "mr_aaaa", "source_nudge_id": "nudge_00000000000000000001", "suggested_state": "keep"})
	// Development 2: step only (its Note was evicted by the Engine's 200-note cap).
	hist = append(hist, map[string]any{"at": base.Add(81 * time.Minute), "kind": "developed", "worker_request_id": "mr_bbbb", "by": "engine"})
	// Development 3: note only (no step with that request ID).
	notes = append(notes, map[string]any{"at": base.Add(82 * time.Minute), "text": "three", "by": "Nemotron (engine)", "worker_request_id": "mr_cccc", "suggested_state": "keep"})
	// Development 4: note and step agree on request but disagree on nudge ID.
	hist = append(hist, map[string]any{"at": base.Add(83 * time.Minute), "kind": "developed", "worker_request_id": "mr_dddd", "source_nudge_id": "nudge_00000000000000000004"})
	notes = append(notes, map[string]any{"at": base.Add(83 * time.Minute), "text": "four", "worker_request_id": "mr_dddd", "source_nudge_id": "nudge_00000000000000000005"})
	// Development 5: an unsafe request ID is NOT RECORDED, never shown or used to correlate.
	notes = append(notes, map[string]any{"at": base.Add(84 * time.Minute), "text": "five <img src=x onerror=alert(1)>", "worker_request_id": "../etc/passwd x"})
	c, _, te := thoughtCore(t, map[string]any{"thought_id": "thought_1", "entity_id": "eng_1", "question": "q", "state": "active",
		"updated_at": base, "history": hist, "notes": notes})
	th, err := c.CognitionThought("thought_1")
	if err != nil {
		t.Fatal(err)
	}
	if th.HistoryTotal != 73 || len(th.History) != 64 || !th.Truncated {
		t.Fatalf("history: total %d shown %d truncated %v", th.HistoryTotal, len(th.History), th.Truncated)
	}
	if last := th.History[len(th.History)-1]; last.RequestID != "mr_dddd" || last.NudgeID != "nudge_00000000000000000004" {
		t.Fatalf("the NEWEST step must be kept: %+v", last)
	}
	if first := th.History[0]; first.TraceID != "trace_09" {
		t.Fatalf("the oldest steps must be the ones dropped: first kept %+v", first)
	}
	by := map[string]CognitionDevelopment{}
	for _, d := range th.Developments {
		by[d.RequestID] = d
	}
	if len(th.Developments) != 5 || th.Developments[0].Note == nil || !strings.HasPrefix(th.Developments[0].Note.Text, "five") {
		t.Fatalf("newest first, 5 developments: %+v", th.Developments)
	}
	if d := by["mr_aaaa"]; d.Persistence != "NOTE_AND_STEP_CORRELATED" || d.NudgeReceipt != "NOT_IN_THIS_OWNERS_JOURNAL" {
		t.Fatalf("foreign nudge ID must not borrow a receipt: %+v", d)
	}
	if d := by["mr_bbbb"]; d.Persistence != "STEP_ONLY" || d.Note != nil || !strings.Contains(d.Meaning, "no longer stored") {
		t.Fatalf("evicted note must be STEP_ONLY, nothing invented: %+v", d)
	}
	if d := by["mr_cccc"]; d.Persistence != "NOTE_ONLY" || d.Step != nil {
		t.Fatalf("note without step: %+v", d)
	}
	if d := by["mr_dddd"]; d.Persistence != "IDS_DISAGREE" {
		t.Fatalf("disagreeing nudge IDs must be flagged: %+v", d)
	}
	if d := by[""]; d.Note == nil || d.Persistence != "NOTE_ONLY" {
		t.Fatalf("unsafe request ID must be NOT RECORDED: %+v", by)
	}
	// A6: the library list carries no Note text, even though the Engine sent it.
	lib, err := c.CognitionLibrary()
	if err != nil {
		t.Fatal(err)
	}
	if b, _ := json.Marshal(lib); strings.Contains(string(b), privateMarker) || strings.Contains(string(b), "onerror") {
		t.Fatal("Note text leaked into the library list")
	}
	if w := te.writes(); len(w) != 0 {
		t.Fatalf("Cognition reads sent non-GET Engine requests: %v", w)
	}
}

// A8 + privacy: LOCKED, an Engine answer for another entity, an invalid ID and an unlinked Identity
// all get no Note text; denial looks the same as "not found" (503) for foreign IDs.
func TestCognitionNoteTextNeedsUnlockedLinkedOwner(t *testing.T) {
	th := map[string]any{"thought_id": "thought_1", "entity_id": "eng_1", "question": "q", "state": "active",
		"notes": []any{map[string]any{"text": privateMarker, "worker_request_id": "mr_1"}}}
	c, ids, te := thoughtCore(t, th)
	if v, err := c.CognitionThought("foreign_1"); !errors.Is(err, ErrCognitionUnavailable) || len(v.Developments) != 0 {
		t.Fatalf("foreign Thought: %v %+v", err, v)
	}
	if _, err := c.CognitionThought("../thought_1"); !errors.Is(err, ErrCognitionUnavailable) {
		t.Fatalf("invalid ID: %v", err)
	}
	ids.set("LOCKED")
	te.mu.Lock()
	before := len(te.calls)
	te.mu.Unlock()
	if v, err := c.CognitionThought("thought_1"); !errors.Is(err, ErrCognitionDenied) || len(v.Developments) != 0 {
		t.Fatalf("LOCKED: %v", err)
	}
	te.mu.Lock()
	after := len(te.calls)
	te.mu.Unlock()
	if after != before {
		t.Fatal("a LOCKED read contacted the Engine")
	}
	ids.set("UNLOCKED")
	if err := c.UnlinkCognition(); err != nil {
		t.Fatal(err)
	}
	if v, err := c.CognitionThought("thought_1"); err == nil || len(v.Developments) != 0 {
		t.Fatalf("unlinked Identity must not read Note text: %v", err)
	}
	// The SaveNudgeDraft path never needs Note text.
	if v, err := c.cognitionThought("thought_1", false); err == nil && len(v.Developments) != 0 {
		t.Fatal("internal check carried developments")
	}
}

// A3 (D2): readiness is derived from Identity's public state, the Engine and Core's own locks —
// never from the draft type alone. Display only: admission still decides (Stage 2 tests).
func TestNudgeReadinessIsDerived(t *testing.T) {
	cases := []struct {
		name               string
		identity           string
		grant, engineUp    bool
		want, wantIdentity string
		wantGrant          string
		wantInReason       string
	}{
		{"unlocked, grant, engine up", "UNLOCKED", true, true, "READY", "UNLOCKED", "ENABLED", "Ready"},
		{"unlocked, no grant", "UNLOCKED", false, true, "NOT_READY", "UNLOCKED", "NOT_ENABLED", "cognition.nudge-process"},
		{"unlocked, grant, engine down", "UNLOCKED", true, false, "NOT_READY", "UNLOCKED", "ENABLED", "Thought Engine"},
		{"unlocked, no grant, engine down", "UNLOCKED", false, false, "NOT_READY", "UNLOCKED", "NOT_ENABLED", "cognition.nudge-process"},
		{"locked, grant", "LOCKED", true, true, "NOT_READY", "LOCKED", "UNKNOWN", "locked"},
		{"locked, no grant, engine down", "LOCKED", false, false, "NOT_READY", "LOCKED", "UNKNOWN", "locked"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, ids, re := runCore(t, t.TempDir())
			ids.mu.Lock()
			ids.processGranted = tc.grant
			ids.mu.Unlock()
			ids.set(tc.identity)
			if !tc.engineUp {
				c.engine = engine.New("http://127.0.0.1:1")
			}
			r, sum := c.nudgeReadiness()
			if sum.State != tc.want || r.Identity != tc.wantIdentity || r.ProcessGrant != tc.wantGrant || !strings.Contains(sum.Reason, tc.wantInReason) {
				t.Fatalf("got %+v %+v", r, sum)
			}
			if (r.Engine == "REACHABLE") != tc.engineUp {
				t.Fatalf("engine: %+v", r)
			}
			v := nudgeProcessingFor(NudgeView{Nudge: Nudge{RecipientID: "self", Status: "DRAFT"}}, sum)
			if (v.State == "READY") != (tc.want == "READY") || (tc.want != "READY" && v.State != "WAITING") {
				t.Fatalf("per-nudge state must follow the derived summary: %+v", v)
			}
			if re.posts != 0 {
				t.Fatal("readiness made a model request")
			}
		})
	}
	// Never READY from the draft type alone.
	sum := NudgeProcessing{State: "READY"}
	for _, n := range []NudgeView{
		{Nudge: Nudge{RecipientID: "position-1", Status: "DRAFT"}},
		{Nudge: Nudge{RecipientID: "self", Status: "EXPIRED"}},
		{Nudge: Nudge{RecipientID: "self", Status: "DRAFT"}, PreRestore: true},
		{Nudge: Nudge{RecipientID: "self", Status: "DRAFT"}, Run: &RunView{State: "COMPLETED"}},
	} {
		if p := nudgeProcessingFor(n, sum); p.State == "READY" {
			t.Fatalf("not processable but READY: %+v", n)
		}
	}
}
