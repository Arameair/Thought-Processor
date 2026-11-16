package tass

// #82 P0-A/B/C regression tests. Fixture Identity and Engine; not the owner's machine.

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Arameair/Thought-Processor/core/engine"
)

type cogFixture struct {
	mu       sync.Mutex
	entities []map[string]string // entity_id, display_name, entity_type
	cycles   []map[string]any    // cycle aggregates
	journals map[string]map[string]any
	pulse    map[string]any // nil → 503
	calls    []string
	srv      *httptest.Server
}

func newCogEngine(t *testing.T) *cogFixture {
	f := &cogFixture{journals: map[string]map[string]any{}}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		f.calls = append(f.calls, r.Method+" "+r.URL.Path)
		if r.Header.Get("Authorization") != "Bearer testkey" || r.Method != "GET" {
			w.WriteHeader(403)
			return
		}
		p := r.URL.Path
		switch {
		case p == "/api/status":
			ents := []any{}
			for _, e := range f.entities {
				ents = append(ents, map[string]any{"card": e})
			}
			json.NewEncoder(w).Encode(map[string]any{"entities": ents, "config": map[string]string{"commit": strings.Repeat("a", 40)}})
		case strings.HasPrefix(p, "/api/entities/") && strings.Count(p, "/") == 3:
			id := strings.TrimPrefix(p, "/api/entities/")
			for _, e := range f.entities {
				if e["entity_id"] == id {
					json.NewEncoder(w).Encode(map[string]any{"card": e, "thoughts": []any{
						map[string]any{"thought_id": "th_" + id, "entity_id": id, "topic": "topic of " + id, "state": "active"}}})
					return
				}
			}
			w.WriteHeader(404)
		case strings.HasPrefix(p, "/api/entities/"):
			parts := strings.Split(p, "/") // /api/entities/{e}/thoughts/{t}
			json.NewEncoder(w).Encode(map[string]any{"thought": map[string]any{"thought_id": parts[5], "entity_id": parts[3], "state": "active"}})
		case p == "/api/pulse":
			if f.pulse == nil {
				w.WriteHeader(503)
				return
			}
			json.NewEncoder(w).Encode(f.pulse)
		case p == "/api/cycles":
			json.NewEncoder(w).Encode(map[string]any{"cycles": f.cycles})
		case strings.HasSuffix(p, "/journal"):
			id := strings.TrimSuffix(strings.TrimPrefix(p, "/api/cycles/"), "/journal")
			if j, ok := f.journals[id]; ok {
				json.NewEncoder(w).Encode(j)
				return
			}
			json.NewEncoder(w).Encode(map[string]any{"cycle_id": id, "steps": []any{}, "missing": []string{"received", "spark", "proposal", "outcome", "evaluation"}})
		case strings.HasPrefix(p, "/api/cycles/"):
			id := strings.TrimPrefix(p, "/api/cycles/")
			for _, cy := range f.cycles {
				if cy["cycle_id"] == id {
					json.NewEncoder(w).Encode(cy)
					return
				}
			}
			w.WriteHeader(404)
		default:
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *cogFixture) engineCalls() int { f.mu.Lock(); defer f.mu.Unlock(); return len(f.calls) }

func cogCore(t *testing.T, f *cogFixture) (*fakeIdentitySvc, *Core) {
	ids, legacy := newFakeIdentitySvc(), newFakeEngine()
	t.Cleanup(func() { ids.srv.Close(); legacy.srv.Close() })
	c := newIdentityCore(t, t.TempDir(), ids, legacy)
	c.engine = engine.New(f.srv.URL)
	c.engine.Key = "testkey"
	ids.set("UNLOCKED")
	return ids, c
}

func cycleIDs(cards []CognitionCard) []string {
	var out []string
	for _, x := range cards {
		out = append(out, x.SourceID)
	}
	return out
}

func TestCognitionTwoEngineEntitiesWithSameNameNeverMix(t *testing.T) {
	f := newCogEngine(t)
	f.entities = []map[string]string{
		{"entity_id": "eng_a", "display_name": "Mira", "entity_type": "ai_companion"},
		{"entity_id": "eng_b", "display_name": "Mira", "entity_type": "ai_companion"},
	}
	f.cycles = []map[string]any{
		{"cycle_id": "cy_a", "identity": "Mira", "entity_id": "eng_a", "state": "evaluated"},
		{"cycle_id": "cy_b", "identity": "Mira", "entity_id": "eng_b", "state": "evaluated"},
		{"cycle_id": "cy_none", "identity": "Mira", "entity_id": "", "state": "failed"},
	}
	_, c := cogCore(t, f)
	lib, err := c.CognitionLibrary()
	if err != nil || len(lib.Cycles) != 0 || len(lib.Thoughts) != 0 || lib.Link.State != "NOT LINKED" || len(lib.Link.Candidates) != 2 ||
		!lib.Link.Candidates[0].SameName || !lib.Link.Candidates[1].SameName {
		t.Fatalf("ambiguous same-name entities must stay unlinked: %+v %v", lib.Link, err)
	}
	if _, err := c.LinkCognition("eng_b"); err != nil {
		t.Fatal(err)
	}
	lib, _ = c.CognitionLibrary()
	if got := cycleIDs(lib.Cycles); len(got) != 1 || got[0] != "cy_b" || lib.EngineEntityID != "eng_b" {
		t.Fatalf("linked view must show only eng_b's cycles: %v", got)
	}
	if lib.Link.UnscopedCycles != 1 || lib.Link.OtherCycles != 1 || len(lib.Thoughts) != 1 || lib.Thoughts[0].SourceID != "th_eng_b" {
		t.Fatalf("counts/thoughts: %+v thoughts=%v", lib.Link, cycleIDs(lib.Thoughts))
	}
	for _, id := range []string{"cy_a", "cy_none"} {
		if _, err := c.CognitionTimeline(id); !errors.Is(err, ErrCognitionUnavailable) {
			t.Fatalf("timeline %s must be refused: %v", id, err)
		}
		if _, _, err := c.CognitionExport(id, "md", "x"); err == nil {
			t.Fatalf("export %s must be refused", id)
		}
	}
	if _, err := c.CognitionThought("th_eng_a"); !errors.Is(err, ErrCognitionUnavailable) {
		// asked under eng_b: the Engine fixture would answer for eng_b only if asked; th_eng_a must not resolve as eng_b's
		th, _ := c.CognitionThought("th_eng_a")
		if th.ThoughtID != "" && len(th.RelatedCycleIDs) != 0 {
			t.Fatalf("foreign thought leaked related cycles: %+v", th)
		}
	}
}

func TestCognitionRenameKeepsLinkButNewEntityWithSameNameIsNotLinked(t *testing.T) {
	f := newCogEngine(t)
	f.entities = []map[string]string{{"entity_id": "eng_a", "display_name": "Mira", "entity_type": "ai_companion"}}
	f.cycles = []map[string]any{{"cycle_id": "cy_a", "identity": "Mira", "entity_id": "eng_a", "state": "evaluated"}}
	ids, c := cogCore(t, f)
	if _, err := c.LinkCognition("eng_a"); err != nil {
		t.Fatal(err)
	}
	ids.mu.Lock()
	ids.name = "Mira Renamed"
	ids.mu.Unlock()
	if lib, _ := c.CognitionLibrary(); lib.Link.State != "LINKED" || len(lib.Cycles) != 1 {
		t.Fatalf("a rename must keep the ID-based link: %+v", lib.Link)
	}
	ids.mu.Lock()
	ids.entityID, ids.name = "ent_NEW", "Mira" // recreated entity with the old name
	ids.mu.Unlock()
	lib, _ := c.CognitionLibrary()
	if lib.Link.State != "NOT LINKED" || len(lib.Cycles) != 0 || len(lib.Thoughts) != 0 {
		t.Fatalf("a new Identity entity must not inherit history by name: %+v", lib.Link)
	}
	if _, err := c.LinkCognition("eng_a"); err == nil || !strings.Contains(err.Error(), "another Identity entity") {
		t.Fatalf("an Engine entity linked to another Identity entity must be refused: %v", err)
	}
}

func TestCognitionLinkRefusalsAndLockedNeverCallsEngine(t *testing.T) {
	f := newCogEngine(t)
	f.entities = []map[string]string{{"entity_id": "eng_a", "display_name": "Mira", "entity_type": "ai_companion"},
		{"entity_id": "eng_human", "display_name": "Mira", "entity_type": "human"}}
	ids, c := cogCore(t, f)
	for _, bad := range []string{"eng_missing", "eng_human", "../x", ""} {
		if _, err := c.LinkCognition(bad); err == nil {
			t.Fatalf("link %q accepted", bad)
		}
	}
	if _, err := c.LinkCognition("eng_a"); err != nil {
		t.Fatal(err)
	}
	f.mu.Lock()
	f.entities = nil // the linked Engine entity disappears
	f.mu.Unlock()
	if lib, _ := c.CognitionLibrary(); lib.Link.State != "LINK BROKEN" || len(lib.Cycles) != 0 {
		t.Fatalf("missing linked entity must fail closed: %+v", lib.Link)
	}
	ids.set("LOCKED")
	n := f.engineCalls()
	for _, call := range []func() error{
		func() error { _, err := c.CognitionLibrary(); return err },
		func() error { _, err := c.CognitionTimeline("cy_a"); return err },
		func() error { _, err := c.CognitionThought("th_eng_a"); return err },
		func() error { _, err := c.LinkCognition("eng_a"); return err },
	} {
		if err := call(); err == nil {
			t.Fatal("LOCKED read succeeded")
		}
	}
	if f.engineCalls() != n {
		t.Fatal("LOCKED Identity still contacted the Engine")
	}
	if err := c.UnlinkCognition(); err == nil {
		t.Fatal("unlink while LOCKED")
	}
}

func TestCognitionNotesUseOneSavedRevisionAndTextCellsAreShown(t *testing.T) {
	f := newCogEngine(t)
	ids, c := cogCore(t, f)
	ids.mu.Lock()
	ids.skillGranted = true
	ids.mu.Unlock()
	g := BrainGraph{Notes: []BrainNote{{ID: "n1", Title: "Ask before running", Text: "always ask the owner first"}},
		Nodes: []BrainNode{{ID: "a", BlockID: "note:n1", Pinned: "note"}}}
	saved, rep, err := c.SaveBrain(g, "owner", "a note")
	if err != nil || saved.EntityID != "ent_0123" {
		t.Fatalf("save: %v %+v %+v", err, rep, saved)
	}
	lib, err := c.CognitionLibrary()
	if err != nil {
		t.Fatal(err)
	}
	var note *CognitionCard
	for i := range lib.Library {
		if lib.Library[i].Kind == "note" {
			note = &lib.Library[i]
		}
	}
	if note == nil || note.Title != "Ask before running" || lib.BrainRevision != saved.Revision ||
		!strings.Contains(note.Source, "revision 1") || !strings.Contains(note.Source, "not a Thought") || note.Verified != "OWNER-WRITTEN" {
		t.Fatalf("saved note missing or mislabeled: %+v rev=%d", note, lib.BrainRevision)
	}
	// Another Identity entity: the same Core's saved notes are not shown under it.
	ids.mu.Lock()
	ids.entityID = "ent_OTHER"
	ids.mu.Unlock()
	lib, _ = c.CognitionLibrary()
	for _, x := range lib.Library {
		if x.Kind == "note" {
			t.Fatal("note leaked to another Identity entity")
		}
	}
	if !strings.Contains(lib.Note, "another Identity entity") {
		t.Fatalf("missing explanation: %q", lib.Note)
	}
}

func TestCognitionInFlightCycleShowsPendingNotMissing(t *testing.T) {
	f := newCogEngine(t)
	f.entities = []map[string]string{{"entity_id": "eng_a", "display_name": "Mira", "entity_type": "ai_companion"}}
	stamp := time.Date(2026, 11, 5, 19, 0, 0, 0, time.UTC)
	f.cycles = []map[string]any{{"cycle_id": "cy_live", "identity": "Mira", "entity_id": "eng_a", "state": "awaiting_outcome"},
		{"cycle_id": "cy_done", "identity": "Mira", "entity_id": "eng_a", "state": "evaluated"}}
	steps := []any{map[string]any{"at": stamp, "offset": 0, "stage": "received", "source": "cycles.jsonl", "status": "OBSERVED"},
		map[string]any{"at": stamp, "offset": 90, "stage": "proposal", "source": "cycles.jsonl", "status": "PROPOSED", "action": "NONE"}}
	f.journals["cy_live"] = map[string]any{"cycle_id": "cy_live", "steps": steps, "missing": []string{"spark", "outcome", "evaluation"}}
	f.journals["cy_done"] = map[string]any{"cycle_id": "cy_done", "steps": steps, "missing": []string{"spark", "outcome", "evaluation"}}
	_, c := cogCore(t, f)
	if _, err := c.LinkCognition("eng_a"); err != nil {
		t.Fatal(err)
	}
	live, err := c.CognitionTimeline("cy_live")
	if err != nil || strings.Join(live.Pending, ",") != "outcome,evaluation" || strings.Join(live.Missing, ",") != "spark" {
		t.Fatalf("in-flight: pending=%v missing=%v %v", live.Pending, live.Missing, err)
	}
	done, _ := c.CognitionTimeline("cy_done")
	if len(done.Pending) != 0 || strings.Join(done.Missing, ",") != "spark,outcome,evaluation" {
		t.Fatalf("finished cycle: pending=%v missing=%v", done.Pending, done.Missing)
	}
	md, _, _ := c.CognitionExport("cy_live", "md", "x")
	if !strings.Contains(string(md), "Pending (cycle still in progress): outcome, evaluation") {
		t.Fatalf("export lacks pending: %s", md)
	}
}

func TestCognitionPulseIsReadOnlyAndTruthful(t *testing.T) {
	f := newCogEngine(t)
	ids, c := cogCore(t, f)
	stamp := time.Date(2026, 11, 5, 19, 0, 0, 0, time.UTC)
	f.pulse = map[string]any{"control": "PAUSED", "state": "READY", "every_s": 20.0, "active_job": "",
		"ticks": []any{map[string]any{"pulse_id": "pulse_000061_151519", "at": stamp, "trigger": "step", "state": "WORKING", "job_id": "lesson_91936d78_1",
			"observed": "1 queued", "decision": "START: lesson_91936d78 SECRET-DETAIL"}},
		"jobs": []any{map[string]any{"state": "QUEUED", "context": "PRIVATE CONTEXT"}, map[string]any{"state": "DONE"}}}
	p, err := c.CognitionPulse()
	if err != nil || p.Control != "PAUSED" || p.EverySeconds != 20 || p.Queued != 1 || p.Running != 0 || p.Last == nil ||
		p.Last.PulseID != "pulse_000061_151519" || p.Last.JobID != "lesson_91936d78_1" || !strings.Contains(p.Note, "PAUSED") {
		t.Fatalf("paused pulse: %+v %v", p, err)
	}
	b, _ := json.Marshal(p)
	if strings.Contains(string(b), "SECRET-DETAIL") || strings.Contains(string(b), "PRIVATE CONTEXT") {
		t.Fatal("pulse projection leaked free text")
	}
	f.mu.Lock()
	f.pulse = map[string]any{"control": "RUNNING", "state": "WORKING", "every_s": 20.0, "active_job": "job_1", "ticks": []any{}, "jobs": []any{map[string]any{"state": "RUNNING"}}}
	f.mu.Unlock()
	if p, _ := c.CognitionPulse(); p.Control != "RUNNING" || p.State != "WORKING" || p.ActiveJob != "job_1" || p.Running != 1 || p.Last != nil {
		t.Fatalf("working pulse: %+v", p)
	}
	f.mu.Lock()
	f.pulse = nil
	f.mu.Unlock()
	if _, err := c.CognitionPulse(); !errors.Is(err, ErrCognitionUnavailable) {
		t.Fatalf("missing Engine pulse must be UNAVAILABLE, not invented: %v", err)
	}
	f.mu.Lock()
	for _, call := range f.calls {
		if !strings.HasPrefix(call, "GET ") {
			t.Fatalf("pulse read sent %s", call)
		}
	}
	f.mu.Unlock()
	ids.set("LOCKED")
	n := f.engineCalls()
	if _, err := c.CognitionPulse(); !errors.Is(err, ErrCognitionDenied) || f.engineCalls() != n {
		t.Fatal("LOCKED pulse read must be denied before any Engine call")
	}
}
