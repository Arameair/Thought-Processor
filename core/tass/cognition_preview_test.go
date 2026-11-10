package tass

// Stage 1 slice 1 (plan PR #109 §8.2, WP1-A): truthful, metadata-only Thought card previews.

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Arameair/Thought-Processor/core/engine"
)

const previewMarker = "PRIVATE-NOTE-TEXT-MARKER-51c9"

var previewT0 = time.Date(2026, 11, 10, 9, 0, 0, 0, time.UTC)

// previewEngine: a GET-only stand-in Engine that serves one entity's Thoughts and counts every path.
type previewEngine struct {
	mu       sync.Mutex
	thoughts []map[string]any
	hits     map[string]int
	nonGET   int
	srv      *httptest.Server
}

func newPreviewEngine(t *testing.T, thoughts ...map[string]any) *previewEngine {
	e := &previewEngine{thoughts: thoughts, hits: map[string]int{}}
	e.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		e.mu.Lock()
		defer e.mu.Unlock()
		e.hits[r.URL.Path]++
		if r.Method != "GET" {
			e.nonGET++
			w.WriteHeader(405)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch p := r.URL.Path; {
		case p == "/api/status":
			json.NewEncoder(w).Encode(map[string]any{"entities": []any{
				map[string]any{"card": map[string]string{"entity_id": "eng_1", "display_name": "Mira", "entity_type": "ai_companion"}},
			}, "config": map[string]string{"commit": "21b3d502775b3ed065ab445b88f51909500770b3"}})
		case p == "/api/entities/eng_1":
			json.NewEncoder(w).Encode(map[string]any{"card": map[string]string{"entity_id": "eng_1", "display_name": "Mira", "entity_type": "ai_companion"},
				"thoughts": e.thoughts, "traces": []any{}})
		case strings.HasPrefix(p, "/api/entities/eng_1/thoughts/"):
			for _, th := range e.thoughts {
				if th["thought_id"] == strings.TrimPrefix(p, "/api/entities/eng_1/thoughts/") {
					json.NewEncoder(w).Encode(map[string]any{"thought": th, "evidence": []any{}})
					return
				}
			}
			w.WriteHeader(404)
		case p == "/api/cycles":
			json.NewEncoder(w).Encode(map[string]any{"cycles": []any{}})
		default:
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(e.srv.Close)
	return e
}

func (e *previewEngine) paths() string {
	e.mu.Lock()
	defer e.mu.Unlock()
	var out []string
	for p, n := range e.hits {
		out = append(out, fmt.Sprintf("%s x%d", p, n))
	}
	sort.Strings(out)
	return strings.Join(out, ", ")
}

func (e *previewEngine) reset() {
	e.mu.Lock()
	e.hits = map[string]int{}
	e.mu.Unlock()
}

func pvThought(id, state string, notes, history []any) map[string]any {
	return map[string]any{"thought_id": id, "entity_id": "eng_1", "topic": "topic " + id, "question": "question " + id, "state": state,
		"updated_at": previewT0, "created_at": previewT0, "notes": notes, "history": history}
}

func pvNote(min int, by, req, nudge string) map[string]any {
	n := map[string]any{"at": previewT0.Add(time.Duration(min) * time.Minute), "text": previewMarker + " " + req, "by": by, "worker_request_id": req}
	if nudge != "" {
		n["source_nudge_id"] = nudge
	}
	return n
}

func pvStep(min int, kind, by, req string) map[string]any {
	return map[string]any{"at": previewT0.Add(time.Duration(min) * time.Minute), "kind": kind, "by": by, "worker_request_id": req, "detail": previewMarker + " detail"}
}

// previewCore: real Core, unlocked Identity ent_0123 linked to eng_1, with the nudge draft grant.
func previewCore(t *testing.T, thoughts ...map[string]any) (*Core, *fakeIdentitySvc, *previewEngine, string) {
	t.Helper()
	dir := t.TempDir()
	ids, pe := newFakeIdentitySvc(), newPreviewEngine(t, thoughts...)
	t.Cleanup(ids.srv.Close)
	legacy := newFakeEngine()
	t.Cleanup(legacy.srv.Close)
	c := newIdentityCore(t, dir, ids, legacy)
	c.engine = engine.New(pe.srv.URL)
	c.engine.Key = "testkey"
	ids.set("UNLOCKED")
	ids.mu.Lock()
	ids.nudgeGranted = true
	ids.mu.Unlock()
	if _, err := c.LinkCognition("eng_1"); err != nil {
		t.Fatal(err)
	}
	return c, ids, pe, dir
}

func pvCard(t *testing.T, c *Core, id string) CognitionCard {
	t.Helper()
	lib, err := c.CognitionLibrary()
	if err != nil {
		t.Fatal(err)
	}
	for _, x := range lib.Thoughts {
		if x.SourceID == id {
			return x
		}
	}
	t.Fatalf("no card for %s", id)
	return CognitionCard{}
}

// Lifecycle, Note count, and the latest recorded development for 0, 1 and many Notes.
func TestThoughtCardPreviewCountsAndLatestOutput(t *testing.T) {
	many := []any{}
	hist := []any{pvStep(0, "formed", "engine", "mr_form")}
	for i := 1; i <= 200; i++ {
		req := fmt.Sprintf("mr_%04d", i)
		many = append(many, pvNote(i, "Nemotron (heartbeat)", req, ""))
		hist = append(hist, pvStep(i, "developed", "heartbeat", req))
	}
	c, _, _, _ := previewCore(t,
		pvThought("thought_none", "active", nil, []any{pvStep(0, "formed", "engine", "mr_f0")}),
		pvThought("thought_one", "sleeping", []any{pvNote(5, "Nemotron (owner nudge)", "mr_one", "nudge_0123456789abcdef0123")},
			[]any{pvStep(5, "developed", "owner-nudge", "mr_one")}),
		pvThought("thought_many", "resolved", many, hist),
		pvThought("thought_owner", "active", []any{pvNote(7, "Nemotron (owner)", "mr_own", "")}, []any{pvStep(7, "developed", "owner", "mr_own")}),
		// The step for mr_old is recorded but its Note is no longer stored; it is the newest development.
		pvThought("thought_evicted", "active", []any{pvNote(3, "Nemotron (heartbeat)", "mr_kept", "")},
			[]any{pvStep(3, "developed", "heartbeat", "mr_kept"), pvStep(9, "developed", "heartbeat", "mr_old")}),
		// A Note with no matching step, by an unknown writer, but a valid nudge ID.
		pvThought("thought_noteonly", "active", []any{pvNote(4, "someone", "mr_solo", "nudge_0123456789abcdef0124")}, nil),
	)
	want := map[string]struct {
		lifecycle string
		count     int
		capped    bool
		kind, by  string
		min       int
	}{
		"thought_none":     {"active", 0, false, "", "", 0},
		"thought_one":      {"sleeping", 1, false, "DEVELOPED_NOTE", "owner-nudge", 5},
		"thought_many":     {"resolved", 200, true, "DEVELOPED_NOTE", "engine", 200},
		"thought_owner":    {"active", 1, false, "DEVELOPED_NOTE", "owner", 7},
		"thought_evicted":  {"active", 1, false, "DEVELOPED_STEP_NOTE_NOT_STORED", "engine", 9},
		"thought_noteonly": {"active", 1, false, "DEVELOPED_NOTE", "owner-nudge", 4},
	}
	for id, w := range want {
		x := pvCard(t, c, id)
		if x.Lifecycle != w.lifecycle || x.NotesCount == nil || *x.NotesCount != w.count || x.NotesCountCapped != w.capped {
			t.Errorf("%s: lifecycle %q count %v capped %v, want %q %d %v", id, x.Lifecycle, x.NotesCount, x.NotesCountCapped, w.lifecycle, w.count, w.capped)
		}
		if w.kind == "" {
			if x.LatestOutput != nil {
				t.Errorf("%s: latest output %+v, want none", id, x.LatestOutput)
			}
			continue
		}
		if x.LatestOutput == nil || x.LatestOutput.Kind != w.kind || x.LatestOutput.By != w.by || !x.LatestOutput.At.Equal(previewT0.Add(time.Duration(w.min)*time.Minute)) {
			t.Errorf("%s: latest output %+v, want %s by %s at +%dm", id, x.LatestOutput, w.kind, w.by, w.min)
		}
		if x.RunActivity != nil {
			t.Errorf("%s: run activity %+v without any Core receipt", id, x.RunActivity)
		}
	}
}

// Unknown or hostile state and source values are never passed through.
func TestThoughtCardPreviewRejectsGarbageStateAndSource(t *testing.T) {
	c, _, _, _ := previewCore(t,
		pvThought("thought_g1", "BUSY", []any{pvNote(1, "<script>x</script>", "mr_g1", "")}, []any{pvStep(1, "developed", "THINKING", "mr_g1")}),
		pvThought("thought_g2", "", nil, []any{pvStep(2, "developed", "", "not a ref!")}),
		pvThought("thought_g3", "Active", nil, nil),
	)
	for _, id := range []string{"thought_g1", "thought_g2", "thought_g3"} {
		x := pvCard(t, c, id)
		if x.Lifecycle != "NOT RECORDED" {
			t.Errorf("%s: lifecycle %q, want NOT RECORDED", id, x.Lifecycle)
		}
		if x.LatestOutput != nil && x.LatestOutput.By != "NOT RECORDED" {
			t.Errorf("%s: source %q, want NOT RECORDED", id, x.LatestOutput.By)
		}
	}
	lib, _ := c.CognitionLibrary()
	raw, _ := json.Marshal(lib.Thoughts)
	for _, bad := range []string{"WORKING", "THINKING", "<script>"} {
		if strings.Contains(string(raw), bad) {
			t.Errorf("the library cards carry %q", bad)
		}
	}
	// The raw Engine state "BUSY" is still in the card's existing status/State fields (unchanged P0
	// behavior); the new lifecycle field never repeats it.
	for _, x := range lib.Thoughts {
		if x.Lifecycle == "BUSY" {
			t.Errorf("lifecycle repeats an unknown Engine state")
		}
	}
}

// The library never carries Note text: not in the new fields and not anywhere else.
func TestThoughtCardPreviewCarriesNoNoteText(t *testing.T) {
	c, _, _, _ := previewCore(t,
		pvThought("thought_p", "active", []any{pvNote(1, "Nemotron (heartbeat)", "mr_p1", ""), pvNote(2, "Nemotron (owner nudge)", "mr_p2", "nudge_0123456789abcdef0123")},
			[]any{pvStep(1, "developed", "heartbeat", "mr_p1"), pvStep(2, "developed", "owner-nudge", "mr_p2")}))
	lib, err := c.CognitionLibrary()
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(lib)
	if strings.Contains(string(raw), previewMarker) {
		t.Fatal("the library response carries Note text or step detail")
	}
	x := pvCard(t, c, "thought_p")
	if *x.NotesCount != 2 || x.LatestOutput == nil || x.LatestOutput.By != "owner-nudge" {
		t.Fatalf("preview: %+v %+v", x.NotesCount, x.LatestOutput)
	}
	// Control: the single-Thought detail (owner decision D1) is where the text is.
	th, err := c.CognitionThought("thought_p")
	if err != nil || len(th.Developments) != 2 || !strings.Contains(th.Developments[0].Note.Text, previewMarker) {
		t.Fatalf("detail control: %v %+v", err, th.Developments)
	}
}

// One library read asks the Engine exactly what it asked before this slice: no per-card request, no
// write. Run activity comes from Core's files only.
func TestThoughtCardPreviewAddsNoEngineRequests(t *testing.T) {
	ths := []map[string]any{}
	for i := 0; i < 12; i++ {
		ths = append(ths, pvThought(fmt.Sprintf("thought_%02d", i), "active", []any{pvNote(i, "Nemotron (heartbeat)", fmt.Sprintf("mr_n%02d", i), "")}, nil))
	}
	c, ids, pe, _ := previewCore(t, ths...)
	pvSubmit(t, c, "thought_03", "key-noeng-0001", nil, nil)
	pe.reset()
	lib, err := c.CognitionLibrary()
	if err != nil || len(lib.Thoughts) != 12 {
		t.Fatalf("library: %v, %d cards", err, len(lib.Thoughts))
	}
	if got := pe.paths(); got != "/api/cycles x1, /api/entities/eng_1 x1, /api/status x1" {
		t.Errorf("Engine requests for one library read: %s", got)
	}
	if pe.nonGET != 0 {
		t.Errorf("Engine writes: %d", pe.nonGET)
	}
	// LOCKED: denied before any Engine request.
	ids.set("LOCKED")
	pe.reset()
	if _, err := c.CognitionLibrary(); err != ErrCognitionDenied {
		t.Fatalf("LOCKED: %v", err)
	}
	if got := pe.paths(); got != "" {
		t.Errorf("LOCKED library read reached the Engine: %s", got)
	}
}

// pvSubmit saves a real draft for the Thought and writes Core's own receipts for it, as Core does.
func pvSubmit(t *testing.T, c *Core, thought, key string, outcome *coreOutcome, void *coreVoid) string {
	t.Helper()
	v, err := c.SaveNudgeDraft(NudgeDraft{SourceThoughtID: thought, RecipientID: "self", ReasonCode: "owner_request",
		RequestedAction: "Look again: " + key, IdempotencyKey: key})
	if err != nil {
		t.Fatal(err)
	}
	pvReceipt(t, c, v.NudgeID, "submitted.json", submittedRec{Schema: nudgeRunSchemaCore, NudgeID: v.NudgeID, OwnerEntityID: "ent_0123",
		EngineEntityID: "eng_1", ThoughtID: thought, At: previewT0.Add(time.Hour)})
	if outcome != nil {
		outcome.Schema = nudgeRunSchemaCore
		outcome.At = previewT0.Add(2 * time.Hour)
		pvReceipt(t, c, v.NudgeID, "outcome.json", outcome)
	}
	if void != nil {
		void.Schema, void.At = nudgeRunSchemaCore, previewT0.Add(2*time.Hour)
		pvReceipt(t, c, v.NudgeID, "void.json", void)
	}
	return v.NudgeID
}

func pvReceipt(t *testing.T, c *Core, id, name string, v any) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(c.runPath(id, name)), 0o700); err != nil {
		t.Fatal(err)
	}
	b, ok := v.([]byte)
	if !ok {
		b, _ = json.Marshal(v)
	}
	if err := os.WriteFile(c.runPath(id, name), b, 0o600); err != nil {
		t.Fatal(err)
	}
}

// Run activity is exactly what Core's receipts record, per Thought, for this owner only.
func TestThoughtCardRunActivityFromCoreReceipts(t *testing.T) {
	ids := []string{"draft", "submitted", "verified", "unverified", "failed", "uncertain", "stopout", "stopvoid", "badoutcome", "badsubmit", "other"}
	ths := []map[string]any{}
	for _, id := range ids {
		ths = append(ths, pvThought("thought_"+id, "active", nil, nil))
	}
	c, _, pe, _ := previewCore(t, ths...)
	run := func(state string) *coreOutcome { return &coreOutcome{Run: engine.NudgeRun{State: state}} }
	// A draft that was never submitted.
	if _, err := c.SaveNudgeDraft(NudgeDraft{SourceThoughtID: "thought_draft", RecipientID: "self", ReasonCode: "owner_request",
		RequestedAction: "draft only", IdempotencyKey: "key-ra-draft-01"}); err != nil {
		t.Fatal(err)
	}
	nid := map[string]string{}
	nid["submitted"] = pvSubmit(t, c, "thought_submitted", "key-ra-0001", nil, nil)
	ok := run("COMPLETED")
	ok.Verified = true
	nid["verified"] = pvSubmit(t, c, "thought_verified", "key-ra-0002", ok, nil)
	nid["unverified"] = pvSubmit(t, c, "thought_unverified", "key-ra-0003", run("COMPLETED"), nil)
	nid["failed"] = pvSubmit(t, c, "thought_failed", "key-ra-0004", run("FAILED"), nil)
	nid["uncertain"] = pvSubmit(t, c, "thought_uncertain", "key-ra-0005", run("UNCERTAIN"), nil)
	nid["stopout"] = pvSubmit(t, c, "thought_stopout", "key-ra-0006", run("VOID"), nil)
	nid["stopvoid"] = pvSubmit(t, c, "thought_stopvoid", "key-ra-0007", nil, &coreVoid{})
	nid["badoutcome"] = pvSubmit(t, c, "thought_badoutcome", "key-ra-0008", nil, nil)
	pvReceipt(t, c, nid["badoutcome"], "outcome.json", []byte("{not json"))
	nid["badsubmit"] = pvSubmit(t, c, "thought_badsubmit", "key-ra-0009", nil, nil)
	pvReceipt(t, c, nid["badsubmit"], "submitted.json", []byte(""))
	// Another owner's nudge for the same Engine entity and Thought ID, with a full verified receipt.
	other := Nudge{Schema: "tass.nudge/1", NudgeID: "nudge_ffffffffffffffffffff", OwnerEntityID: "ent_other", SourceEngineEntityID: "eng_1",
		SourceThoughtID: "thought_other", CreatedAt: previewT0, Status: "DRAFT"}
	if err := c.appendNudge(other); err != nil {
		t.Fatal(err)
	}
	pvReceipt(t, c, other.NudgeID, "submitted.json", submittedRec{Schema: nudgeRunSchemaCore, NudgeID: other.NudgeID, OwnerEntityID: "ent_other",
		EngineEntityID: "eng_1", ThoughtID: "thought_other", At: previewT0})
	pvReceipt(t, c, other.NudgeID, "outcome.json", coreOutcome{Schema: nudgeRunSchemaCore, Run: engine.NudgeRun{State: "COMPLETED"}, Verified: true, At: previewT0})

	want := map[string]string{"draft": "", "submitted": "SUBMITTED_NO_OUTCOME", "verified": "COMPLETED_VERIFIED", "unverified": "NEEDS_RECOVERY",
		"failed": "FAILED", "uncertain": "UNCERTAIN", "stopout": "STOPPED", "stopvoid": "STOPPED", "badoutcome": "NEEDS_RECOVERY",
		"badsubmit": "NEEDS_RECOVERY", "other": ""}
	pe.reset()
	for _, id := range ids {
		x := pvCard(t, c, "thought_"+id)
		got := ""
		if x.RunActivity != nil {
			got = x.RunActivity.State
			if x.RunActivity.NudgeID != nid[id] || x.RunActivity.At.IsZero() {
				t.Errorf("%s: activity %+v, want nudge %s with a time", id, x.RunActivity, nid[id])
			}
		}
		if got != want[id] {
			t.Errorf("%s: run activity %q, want %q", id, got, want[id])
		}
	}
	if strings.Contains(pe.paths(), "nudges") || pe.nonGET != 0 {
		t.Errorf("run activity asked the Engine: %s", pe.paths())
	}
	// The newest submitted nudge for a Thought wins.
	later := pvSubmit(t, c, "thought_verified", "key-ra-0010", nil, nil)
	pvReceipt(t, c, later, "submitted.json", submittedRec{Schema: nudgeRunSchemaCore, NudgeID: later, OwnerEntityID: "ent_0123",
		EngineEntityID: "eng_1", ThoughtID: "thought_verified", At: previewT0.Add(5 * time.Hour)})
	if x := pvCard(t, c, "thought_verified"); x.RunActivity == nil || x.RunActivity.NudgeID != later || x.RunActivity.State != "SUBMITTED_NO_OUTCOME" {
		t.Errorf("newest nudge: %+v", x.RunActivity)
	}
	// A receipt that names another Thought than the journal does is not shown.
	pvReceipt(t, c, nid["failed"], "submitted.json", submittedRec{Schema: nudgeRunSchemaCore, NudgeID: nid["failed"], OwnerEntityID: "ent_0123",
		EngineEntityID: "eng_1", ThoughtID: "thought_submitted", At: previewT0.Add(9 * time.Hour)})
	if x := pvCard(t, c, "thought_failed"); x.RunActivity != nil {
		t.Errorf("mismatched receipt shown: %+v", x.RunActivity)
	}
	if x := pvCard(t, c, "thought_submitted"); x.RunActivity == nil || x.RunActivity.NudgeID != nid["submitted"] {
		t.Errorf("a mismatched receipt moved to another Thought's card: %+v", x.RunActivity)
	}
}

// An Identity entity with no Engine link gets no Thought cards and no preview.
func TestThoughtCardPreviewNeedsTheOwnerLink(t *testing.T) {
	c, _, pe, _ := previewCore(t, pvThought("thought_u", "active", []any{pvNote(1, "Nemotron (heartbeat)", "mr_u", "")}, nil))
	if err := c.UnlinkCognition(); err != nil {
		t.Fatal(err)
	}
	pe.reset()
	lib, err := c.CognitionLibrary()
	if err != nil || len(lib.Thoughts) != 0 {
		t.Fatalf("unlinked library: %v, %d Thought cards", err, len(lib.Thoughts))
	}
	if strings.Contains(pe.paths(), "/api/entities/eng_1") {
		t.Errorf("an unlinked identity read the Engine entity: %s", pe.paths())
	}
}
