package tass

import (
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Arameair/Thought-Processor/core/engine"
)

// nudgeEngine is a GET-only Engine fixture with one real persistent Thought (thought_1 of eng_1)
// and one Thought of another Engine entity. It records every call so tests can prove that saving
// a draft never reaches a model or any write endpoint.
type nudgeEngine struct {
	mu    sync.Mutex
	calls []string
	srv   *httptest.Server
}

func newNudgeEngine(t *testing.T) *nudgeEngine {
	n := &nudgeEngine{}
	stamp := time.Date(2026, 11, 8, 9, 0, 0, 0, time.UTC)
	n.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n.mu.Lock()
		n.calls = append(n.calls, r.Method+" "+r.URL.Path)
		n.mu.Unlock()
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
		case "/api/entities/eng_1/thoughts/thought_1":
			json.NewEncoder(w).Encode(map[string]any{"thought": map[string]any{
				"thought_id": "thought_1", "entity_id": "eng_1", "topic": "real topic", "question": "real question",
				"state": "active", "updated_at": stamp},
				"evidence": []any{map[string]any{"trace": map[string]any{
					"trace_id": "trace_1", "entity_id": "eng_1", "thought_id": "thought_1", "source_event_id": "evt_1", "created_at": stamp}}}})
		case "/api/entities/eng_1/thoughts/other_thought":
			// The Engine answers, but the Thought belongs to another entity: must be refused.
			json.NewEncoder(w).Encode(map[string]any{"thought": map[string]any{"thought_id": "other_thought", "entity_id": "eng_other"}})
		case "/api/cycles":
			json.NewEncoder(w).Encode(map[string]any{"cycles": []any{
				map[string]any{"cycle_id": "tp_1", "identity": "Mira", "entity_id": "eng_1", "received_at": stamp,
					"state": "evaluated", "spark": map[string]any{"thought_id": "thought_1", "status": "done"}},
			}})
		default:
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(n.srv.Close)
	return n
}

func (n *nudgeEngine) snapshot() []string {
	n.mu.Lock()
	defer n.mu.Unlock()
	return append([]string(nil), n.calls...)
}

func nudgeCore(t *testing.T, dir string, ids *fakeIdentitySvc, ne *nudgeEngine) *Core {
	t.Helper()
	legacy := newFakeEngine()
	t.Cleanup(legacy.srv.Close)
	c := newIdentityCore(t, dir, ids, legacy)
	c.engine = engine.New(ne.srv.URL)
	c.engine.Key = "testkey"
	return c
}

func goodDraft(key string) NudgeDraft {
	return NudgeDraft{SourceThoughtID: "thought_1", RecipientID: "position-2", ReasonCode: "follow_up",
		RequestedAction: "Look at this again with the newest evidence.", IdempotencyKey: key}
}

func TestNudgeDraftSavesLocallyWithZeroModelCalls(t *testing.T) {
	ids, ne := newFakeIdentitySvc(), newNudgeEngine(t)
	t.Cleanup(ids.srv.Close)
	dir := t.TempDir()
	c := nudgeCore(t, dir, ids, ne)

	// M5: LOCKED is denied before any Engine contact and nothing is written.
	if _, err := c.SaveNudgeDraft(goodDraft("key-locked-0001")); !errors.Is(err, ErrCognitionDenied) {
		t.Fatalf("LOCKED draft: %v", err)
	}
	if len(ne.snapshot()) != 0 {
		t.Fatal("a LOCKED draft contacted the Engine")
	}
	ids.set("UNLOCKED")
	// M5: unlocked but no owner-reviewed Engine link: refused.
	if _, err := c.SaveNudgeDraft(goodDraft("key-unlinked-01")); !errors.Is(err, ErrNudgeInvalid) {
		t.Fatalf("unlinked draft: %v", err)
	}
	if _, err := c.LinkCognition("eng_1"); err != nil {
		t.Fatal(err)
	}
	// GPT-6 review P0: thought.process (a default grant every unlocked entity has) is NOT enough.
	if _, err := c.SaveNudgeDraft(goodDraft("key-no-grant-01")); !errors.Is(err, ErrNudgeNotGranted) {
		t.Fatalf("draft without the owner grant: %v", err)
	}
	if _, err := os.Stat(c.file(nudgeFile)); !os.IsNotExist(err) {
		t.Fatal("a refused draft wrote the journal")
	}
	ids.mu.Lock()
	ids.nudgeGranted = true
	ids.mu.Unlock()

	// M3: a draft is saved, scoped and carries real source evidence.
	v, err := c.SaveNudgeDraft(goodDraft("key-first-00001"))
	if err != nil {
		t.Fatal(err)
	}
	if v.Status != "DRAFT" || v.OwnerEntityID != "ent_0123" || v.SenderIdentityID != "ent_0123" ||
		v.SourceEngineEntityID != "eng_1" || v.SourceThoughtID != "thought_1" || v.RecipientKind != "card_position" ||
		v.DraftDecisionID == "" || v.ModelCalls != 0 || v.Replayed {
		t.Fatalf("draft record: %+v", v)
	}
	if strings.Join(v.SourceEvidenceRefs, ",") != "trace:trace_1,cycle:tp_1" {
		t.Fatalf("evidence refs: %v", v.SourceEvidenceRefs)
	}
	// M7/M8: processing is BLOCKED and results are NOT RECORDED (null), never invented.
	if v.Processing.State != "BLOCKED" || v.ProcessingCycleID != nil || v.ResultThoughtID != nil || v.AuthorizationDecisionID != nil {
		t.Fatalf("processing must be blocked with no result: %+v", v)
	}
	// Identity was asked for a fresh write decision, not just the read grants.
	ids.mu.Lock()
	decisions := strings.Join(ids.decisions, " ")
	ids.mu.Unlock()
	if !strings.Contains(decisions, "cognition.nudge.draft") {
		t.Fatalf("no fresh Identity decision for the write: %s", decisions)
	}
	// Zero model calls: every Engine call was a GET to a read endpoint.
	for _, call := range ne.snapshot() {
		if !strings.HasPrefix(call, "GET ") || strings.Contains(call, "think") || strings.Contains(call, "job") {
			t.Fatalf("draft sent an Engine operation: %s", call)
		}
	}
	if fi, err := os.Stat(c.file(nudgeFile)); err != nil || fi.Mode().Perm() != 0o600 {
		t.Fatalf("journal must exist with 0600: %v %v", fi, err)
	}

	// M4: same key + same body is a replay of the SAME record; a different body is a conflict.
	again, err := c.SaveNudgeDraft(goodDraft("key-first-00001"))
	if err != nil || !again.Replayed || again.NudgeID != v.NudgeID {
		t.Fatalf("replay: %+v %v", again, err)
	}
	changed := goodDraft("key-first-00001")
	changed.RecipientID = "self"
	if _, err := c.SaveNudgeDraft(changed); !errors.Is(err, ErrNudgeConflict) {
		t.Fatalf("same key, different body: %v", err)
	}

	// Restart: a new Core on the same folder sees exactly one record.
	c2 := nudgeCore(t, dir, ids, ne)
	list, err := c2.NudgeHistory("")
	if err != nil || len(list.Nudges) != 1 || list.Nudges[0].NudgeID != v.NudgeID || list.Processing.State != "BLOCKED" {
		t.Fatalf("after restart: %+v %v", list, err)
	}
	if l2, _ := c2.NudgeHistory("thought_9"); len(l2.Nudges) != 0 {
		t.Fatal("filter by Thought leaked other drafts")
	}

	// Expiry is shown on read, never written.
	c2.now = func() time.Time { return time.Now().Add(25 * time.Hour) }
	if l3, _ := c2.NudgeHistory(""); len(l3.Nudges) != 1 || l3.Nudges[0].Status != "EXPIRED" {
		t.Fatalf("expired draft: %+v", l3)
	}

	// Another Identity entity never sees these drafts.
	ids.mu.Lock()
	ids.entityID = "ent_other"
	ids.mu.Unlock()
	if l4, err := c2.NudgeHistory(""); err == nil && len(l4.Nudges) != 0 {
		t.Fatalf("cross-identity read: %+v", l4)
	}
}

func TestNudgeDraftRefusesBadInputs(t *testing.T) {
	ids, ne := newFakeIdentitySvc(), newNudgeEngine(t)
	t.Cleanup(ids.srv.Close)
	c := nudgeCore(t, t.TempDir(), ids, ne)
	ids.set("UNLOCKED")
	ids.nudgeGranted = true
	if _, err := c.LinkCognition("eng_1"); err != nil {
		t.Fatal(err)
	}
	before := len(ne.snapshot())
	cases := map[string]func(*NudgeDraft){
		// M12: external recipients are not available in v1, and nothing is contacted.
		"discord":       func(d *NudgeDraft) { d.RecipientID = "discord" },
		"claude":        func(d *NudgeDraft) { d.RecipientID = "claude" },
		"gpt-6":         func(d *NudgeDraft) { d.RecipientID = "gpt-6" },
		"position-6":    func(d *NudgeDraft) { d.RecipientID = "position-6" },
		"bad reason":    func(d *NudgeDraft) { d.ReasonCode = "because" },
		"empty action":  func(d *NudgeDraft) { d.RequestedAction = "  " },
		"long action":   func(d *NudgeDraft) { d.RequestedAction = strings.Repeat("a", 501) },
		"control chars": func(d *NudgeDraft) { d.RequestedAction = "hi\x00there" },
		"bad priority":  func(d *NudgeDraft) { d.Priority = "urgent" },
		"long expiry":   func(d *NudgeDraft) { d.ExpiresInHours = 169 },
		"max int":       func(d *NudgeDraft) { d.ExpiresInHours = math.MaxInt },
		"overflow wrap": func(d *NudgeDraft) { d.ExpiresInHours = int(math.MaxInt64/int64(time.Hour)) + 1 },
		"short key":     func(d *NudgeDraft) { d.IdempotencyKey = "abc" },
		"traversal":     func(d *NudgeDraft) { d.SourceThoughtID = "../secrets" },
	}
	for name, mutate := range cases {
		d := goodDraft("key-bad-input-01")
		mutate(&d)
		if _, err := c.SaveNudgeDraft(d); !errors.Is(err, ErrNudgeInvalid) {
			t.Fatalf("%s: want ErrNudgeInvalid, got %v", name, err)
		}
	}
	if len(ne.snapshot()) != before {
		t.Fatal("invalid drafts must be refused before contacting the Engine")
	}
	// M6: a Thought of another Engine entity, or one that doesn't exist, is refused.
	for _, id := range []string{"other_thought", "missing_thought"} {
		d := goodDraft("key-wrong-" + id)
		d.SourceThoughtID = id
		if _, err := c.SaveNudgeDraft(d); !errors.Is(err, ErrCognitionUnavailable) {
			t.Fatalf("%s: %v", id, err)
		}
	}
	if list, _ := c.NudgeHistory(""); len(list.Nudges) != 0 {
		t.Fatalf("refused drafts were saved: %+v", list)
	}
}

func TestNudgeJournalDamagedLineIsKept(t *testing.T) {
	ids, ne := newFakeIdentitySvc(), newNudgeEngine(t)
	t.Cleanup(ids.srv.Close)
	c := nudgeCore(t, t.TempDir(), ids, ne)
	ids.set("UNLOCKED")
	ids.nudgeGranted = true
	if _, err := c.LinkCognition("eng_1"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(c.file(nudgeFile), []byte("{not json\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := c.SaveNudgeDraft(goodDraft("key-after-damage")); err != nil {
		t.Fatal(err)
	}
	list, err := c.NudgeHistory("")
	if err != nil || len(list.Nudges) != 1 || list.InvalidLines != 1 {
		t.Fatalf("damaged line must be counted, not lost: %+v %v", list, err)
	}
	b, _ := os.ReadFile(c.file(nudgeFile))
	if !strings.HasPrefix(string(b), "{not json\n") {
		t.Fatal("the damaged line was overwritten")
	}
}

func TestStarterDocKeepsPrivateStateOut(t *testing.T) {
	doc := starterDoc("Kyssa")
	for _, gone := range []string{"## Memory", "## Now", "lives only on this computer"} {
		if strings.Contains(doc, gone) {
			t.Fatalf("starter document still contains %q", gone)
		}
	}
	if !strings.Contains(doc, "# Kyssa") || !strings.Contains(doc, "## Learned") || strings.Contains(doc, "TASS") {
		t.Fatalf("starter document must be entity-neutral with a Learned section:\n%s", doc)
	}
}

// GPT-6 review P1: after a relink, an old key must not replay the old Engine entity's draft.
func TestNudgeReplayDoesNotCrossEngineLinks(t *testing.T) {
	ids, ne := newFakeIdentitySvc(), newNudgeEngine(t)
	t.Cleanup(ids.srv.Close)
	c := nudgeCore(t, t.TempDir(), ids, ne)
	ids.set("UNLOCKED")
	ids.nudgeGranted = true
	if _, err := c.LinkCognition("eng_1"); err != nil {
		t.Fatal(err)
	}
	old, err := c.SaveNudgeDraft(goodDraft("key-before-relink"))
	if err != nil {
		t.Fatal(err)
	}
	if err := c.UnlinkCognition(); err != nil {
		t.Fatal(err)
	}
	if _, err := c.LinkCognition("eng_other"); err != nil {
		t.Fatal(err)
	}
	v, err := c.SaveNudgeDraft(goodDraft("key-before-relink"))
	if !errors.Is(err, ErrNudgeConflict) || v.NudgeID != "" || v.RequestedAction != "" || v.DraftDecisionID != "" {
		t.Fatalf("relinked replay returned the old draft: %+v %v", v, err)
	}
	if strings.Contains(err.Error(), old.NudgeID) || strings.Contains(err.Error(), old.RequestedAction) {
		t.Fatal("the refusal leaked the old draft")
	}
	if list, _ := c.NudgeHistory(""); len(list.Nudges) != 0 {
		t.Fatalf("relinked history shows the old Engine's drafts: %+v", list)
	}
	// Linking back shows it again (history is kept, never rewritten).
	c.UnlinkCognition()
	if _, err := c.LinkCognition("eng_1"); err != nil {
		t.Fatal(err)
	}
	if again, err := c.SaveNudgeDraft(goodDraft("key-before-relink")); err != nil || !again.Replayed || again.NudgeID != old.NudgeID {
		t.Fatalf("same-link replay: %+v %v", again, err)
	}
}
