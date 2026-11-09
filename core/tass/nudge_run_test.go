package tass

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Arameair/Thought-Processor/core/engine"
)

// runEngine: a stand-in Thought Engine with the Stage 2 nudge routes (one claim per nudge ID, void
// with the same exclusivity) and a Thought whose notes/history gain the nudge's step on COMPLETED.
type runEngine struct {
	mu        sync.Mutex
	runs      map[string]string // nid → RUNNING | COMPLETED | VOID | ...
	reqID     map[string]string
	posts     int // POST .../nudge received
	completed []string
	gate      chan struct{} // if set, a nudge POST waits on it before completing
	refuse    int           // if set, POST nudge answers this status (no claim)
	srv       *httptest.Server
}

func newRunEngine(t *testing.T) *runEngine {
	e := &runEngine{runs: map[string]string{}, reqID: map[string]string{}}
	stamp := time.Date(2026, 11, 8, 9, 0, 0, 0, time.UTC)
	e.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer testkey" {
			w.WriteHeader(401)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		p := r.URL.Path
		runJSON := func(nid string) map[string]any {
			e.mu.Lock()
			defer e.mu.Unlock()
			st := e.runs[nid]
			out := map[string]any{"nudge_id": nid, "entity_id": "eng_1", "thought_id": "thought_1", "state": st, "request_id": e.reqID[nid], "model_http": "responded"}
			if st == "VOID" {
				out["model_http"] = "none"
			}
			return out
		}
		switch {
		case r.Method == "GET" && p == "/api/status":
			json.NewEncoder(w).Encode(map[string]any{"entities": []any{
				map[string]any{"card": map[string]string{"entity_id": "eng_1", "display_name": "Mira", "entity_type": "ai_companion"}},
			}, "config": map[string]string{"commit": "21b3d502775b3ed065ab445b88f51909500770b3"}})
		case r.Method == "GET" && p == "/api/entities/eng_1/thoughts/thought_1":
			e.mu.Lock()
			var notes, hist []any
			for _, nid := range e.completed {
				notes = append(notes, map[string]any{"at": stamp, "text": "NOTE-TEXT-" + nid, "by": "Nemotron (owner-nudge)", "suggested_state": "resolve",
					"worker_request_id": e.reqID[nid], "source_nudge_id": nid})
				hist = append(hist, map[string]any{"at": stamp, "kind": "developed", "detail": "DETAIL-REPEATS-NOTE", "by": "owner-nudge", "worker_request_id": e.reqID[nid], "source_nudge_id": nid})
			}
			e.mu.Unlock()
			json.NewEncoder(w).Encode(map[string]any{"thought": map[string]any{"thought_id": "thought_1", "entity_id": "eng_1", "topic": "t",
				"question": "q", "state": "active", "updated_at": stamp, "notes": notes, "history": hist}})
		case r.Method == "GET" && p == "/api/cycles":
			json.NewEncoder(w).Encode(map[string]any{"cycles": []any{}})
		case r.Method == "POST" && strings.HasSuffix(p, "/nudge"):
			var in engine.NudgeIn
			json.NewDecoder(r.Body).Decode(&in)
			e.mu.Lock()
			e.posts++
			if e.refuse != 0 {
				code := e.refuse
				e.mu.Unlock()
				w.WriteHeader(code)
				json.NewEncoder(w).Encode(map[string]string{"error": "refused (test)"})
				return
			}
			if st, ok := e.runs[in.NudgeID]; ok {
				e.mu.Unlock()
				_ = st
				json.NewEncoder(w).Encode(map[string]any{"run": runJSON(in.NudgeID)})
				return
			}
			e.runs[in.NudgeID], e.reqID[in.NudgeID] = "RUNNING", "mr_"+in.NudgeID[6:14]
			gate := e.gate
			e.mu.Unlock()
			if gate != nil {
				<-gate
			}
			e.mu.Lock()
			e.runs[in.NudgeID] = "COMPLETED"
			e.completed = append(e.completed, in.NudgeID)
			e.mu.Unlock()
			json.NewEncoder(w).Encode(map[string]any{"run": runJSON(in.NudgeID)})
		case r.Method == "POST" && strings.HasSuffix(p, "/void"):
			nid := strings.Split(p, "/")[5]
			e.mu.Lock()
			if _, ok := e.runs[nid]; !ok {
				e.runs[nid] = "VOID"
			}
			e.mu.Unlock()
			json.NewEncoder(w).Encode(map[string]any{"run": runJSON(nid)})
		case r.Method == "GET" && strings.HasPrefix(p, "/api/entities/eng_1/nudges/"):
			nid := strings.TrimPrefix(p, "/api/entities/eng_1/nudges/")
			e.mu.Lock()
			_, ok := e.runs[nid]
			e.mu.Unlock()
			if !ok {
				w.WriteHeader(404)
				json.NewEncoder(w).Encode(map[string]any{"run": map[string]any{"nudge_id": nid, "state": "NOT_CLAIMED"}, "state": "NOT_CLAIMED"})
				return
			}
			json.NewEncoder(w).Encode(map[string]any{"run": runJSON(nid)})
		default:
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(e.srv.Close)
	return e
}

func runCore(t *testing.T, dir string) (*Core, *fakeIdentitySvc, *runEngine) {
	t.Helper()
	ids, re := newFakeIdentitySvc(), newRunEngine(t)
	t.Cleanup(ids.srv.Close)
	legacy := newFakeEngine()
	t.Cleanup(legacy.srv.Close)
	c := newIdentityCore(t, dir, ids, legacy)
	c.engine = engine.New(re.srv.URL)
	c.engine.Key = "testkey"
	ids.set("UNLOCKED")
	ids.mu.Lock()
	ids.nudgeGranted, ids.processGranted = true, true
	ids.mu.Unlock()
	if _, err := c.LinkCognition("eng_1"); err != nil {
		t.Fatal(err)
	}
	return c, ids, re
}

func selfDraft(c *Core, t *testing.T, key string) NudgeView {
	t.Helper()
	v, err := c.SaveNudgeDraft(NudgeDraft{SourceThoughtID: "thought_1", RecipientID: "self", ReasonCode: "owner_request",
		RequestedAction: "Please look at this again: " + key, IdempotencyKey: key})
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func waitRun(t *testing.T, c *Core, id string, want string) *RunView {
	t.Helper()
	for i := 0; i < 200; i++ {
		if v := c.runView(id); v != nil && v.State == want {
			return v
		}
		time.Sleep(10 * time.Millisecond)
	}
	v := c.runView(id)
	t.Fatalf("run %s: want %s, got %+v", id, want, v)
	return nil
}

func TestProcessNudgeHappyPathVerifiedAndNeverResent(t *testing.T) {
	dir := t.TempDir()
	c, _, re := runCore(t, dir)
	d := selfDraft(c, t, "key-process-0001")
	v, accepted, err := c.ProcessNudge(d.NudgeID, d.BodyDigest)
	if err != nil || !accepted || v.State != "SUBMITTED" && v.State != "RUNNING" && v.State != "COMPLETED" {
		t.Fatalf("process: %+v %v %v", v, accepted, err)
	}
	done := waitRun(t, c, d.NudgeID, "COMPLETED")
	if !done.Verified || re.posts != 1 {
		t.Fatalf("verified=%v posts=%d", done.Verified, re.posts)
	}
	// Again: the existing run, no second POST. A restarted Core reads it and never sends.
	if v, accepted, err := c.ProcessNudge(d.NudgeID, d.BodyDigest); err != nil || accepted || v.State != "COMPLETED" {
		t.Fatalf("replay: %+v %v %v", v, accepted, err)
	}
	c.StopNudgeDispatch()
	c2 := nudgeCoreReopen(t, dir, c)
	list, _ := c2.NudgeHistory("")
	if len(list.Nudges) != 1 || list.Nudges[0].Run == nil || list.Nudges[0].Run.State != "COMPLETED" || re.posts != 1 {
		t.Fatalf("after restart: %+v posts=%d", list, re.posts)
	}
}

// nudgeCoreReopen: a new Core on the same data folder with the same services (= restart).
func nudgeCoreReopen(t *testing.T, dir string, old *Core) *Core {
	t.Helper()
	c, err := New(old.cfg, old.gh, old.engine)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// S16/X18: one bound decision before replay; revoked → 403 even with an existing run.
func TestProcessNudgeRevokedReplayIs403(t *testing.T) {
	c, ids, re := runCore(t, t.TempDir())
	d := selfDraft(c, t, "key-revoke-00001")
	c.ProcessNudge(d.NudgeID, d.BodyDigest)
	waitRun(t, c, d.NudgeID, "COMPLETED")
	ids.mu.Lock()
	ids.processGranted = false
	ids.mu.Unlock()
	if _, _, err := c.ProcessNudge(d.NudgeID, d.BodyDigest); !errors.Is(err, ErrProcessNotGranted) {
		t.Fatalf("revoked replay: %v", err)
	}
	if list, err := c.NudgeHistory(""); err != nil || list.Nudges[0].Run == nil || list.Nudges[0].Run.State != "COMPLETED" {
		t.Fatalf("GET after revoke: %+v %v", list, err)
	}
	if re.posts != 1 {
		t.Fatalf("posts: %d", re.posts)
	}
	// Only bound decisions were ever requested for processing.
	ids.mu.Lock()
	defer ids.mu.Unlock()
	for id, b := range ids.bound {
		if !strings.HasPrefix(b, "cognition.nudge.") {
			t.Fatalf("%s: %s", id, b)
		}
	}
}

func TestProcessNudgeRefusals(t *testing.T) {
	c, ids, re := runCore(t, t.TempDir())
	d := selfDraft(c, t, "key-refuse-00001")
	pos, _ := c.SaveNudgeDraft(NudgeDraft{SourceThoughtID: "thought_1", RecipientID: "position-3", ReasonCode: "question",
		RequestedAction: "x", IdempotencyKey: "key-refuse-00002"})
	cases := []struct {
		name, id, digest string
		want             error
	}{
		{"stale digest", d.NudgeID, "stale", ErrNudgeConflict},
		{"card position", pos.NudgeID, pos.BodyDigest, ErrProcessRefused},
		{"unknown nudge", "nudge_00000000000000000000", "x", ErrNudgeInvalid},
		{"bad id", "../x", "x", ErrNudgeInvalid},
	}
	for _, tc := range cases {
		if _, _, err := c.ProcessNudge(tc.id, tc.digest); !errors.Is(err, tc.want) {
			t.Fatalf("%s: %v", tc.name, err)
		}
	}
	ids.mu.Lock()
	ids.processGranted = false
	ids.mu.Unlock()
	if _, _, err := c.ProcessNudge(d.NudgeID, d.BodyDigest); !errors.Is(err, ErrProcessNotGranted) {
		t.Fatalf("no grant: %v", err)
	}
	ids.mu.Lock()
	ids.processGranted = true
	ids.mu.Unlock()
	ids.set("LOCKED")
	if _, _, err := c.ProcessNudge(d.NudgeID, d.BodyDigest); err == nil {
		t.Fatal("LOCKED processed")
	}
	ids.set("UNLOCKED")
	// Expired draft.
	c.now = func() time.Time { return time.Now().Add(48 * time.Hour) }
	if _, _, err := c.ProcessNudge(d.NudgeID, d.BodyDigest); !errors.Is(err, ErrProcessRefused) {
		t.Fatalf("expired: %v", err)
	}
	c.now = time.Now
	// Forced chat-only.
	c.cfg.ChatOnly = true
	if _, _, err := c.ProcessNudge(d.NudgeID, d.BodyDigest); err == nil {
		t.Fatal("forced chat-only processed")
	}
	c.cfg.ChatOnly = false
	if re.posts != 0 {
		t.Fatalf("posts: %d", re.posts)
	}
	if ents, _ := os.ReadDir(filepath.Join(c.cfg.DataDir, nudgeRunDirCore)); len(ents) != 0 {
		t.Fatal("a refused nudge left a submission")
	}
}

// S18: two different nudges at the same instant: one submission, the other 409; one model run.
func TestProcessNudgeAdmissionRace(t *testing.T) {
	for round := 0; round < 20; round++ {
		c, _, re := runCore(t, t.TempDir())
		re.gate = make(chan struct{})
		a, b := selfDraft(c, t, fmt.Sprintf("key-race-a-%04d", round)), selfDraft(c, t, fmt.Sprintf("key-race-b-%04d", round))
		var wg sync.WaitGroup
		var okCount, busy int
		var mu sync.Mutex
		for _, d := range []NudgeView{a, b} {
			wg.Add(1)
			go func(d NudgeView) {
				defer wg.Done()
				_, accepted, err := c.ProcessNudge(d.NudgeID, d.BodyDigest)
				mu.Lock()
				defer mu.Unlock()
				if err == nil && accepted {
					okCount++
				} else if errors.Is(err, ErrProcessBusy) {
					busy++
				}
			}(d)
		}
		wg.Wait()
		close(re.gate)
		if okCount != 1 || busy != 1 {
			t.Fatalf("round %d: accepted=%d busy=%d", round, okCount, busy)
		}
		c.StopNudgeDispatch()
		re.mu.Lock()
		posts := re.posts
		re.mu.Unlock()
		if posts > 1 {
			t.Fatalf("round %d: posts=%d", round, posts)
		}
	}
}

// Stop before the Engine claims → NOT_DELIVERED; the late delivery then gets VOID (no run).
func TestStopNudgeBeforeClaimAndAfter(t *testing.T) {
	c, ids, re := runCore(t, t.TempDir())
	d := selfDraft(c, t, "key-stop-000001")
	if _, err := c.StopNudge(d.NudgeID); !errors.Is(err, ErrNudgeConflict) {
		t.Fatalf("stop without a submission: %v", err)
	}
	// A submission that the Engine refuses to claim at first (it is down): SUBMITTED, never terminal by itself.
	re.refuse = 503
	c.ProcessNudge(d.NudgeID, d.BodyDigest)
	time.Sleep(50 * time.Millisecond)
	if v := c.runView(d.NudgeID); v.State != "SUBMITTED" || !v.CanStop {
		t.Fatalf("not-claimed: %+v", v)
	}
	re.refuse = 0
	// Stop works even after the process grant is revoked (D15).
	ids.mu.Lock()
	ids.processGranted = false
	ids.mu.Unlock()
	v, err := c.StopNudge(d.NudgeID)
	if err != nil || v.State != "NOT_DELIVERED" {
		t.Fatalf("stop: %+v %v", v, err)
	}
	ids.mu.Lock()
	ids.processGranted = true
	ids.mu.Unlock()
	// After a successful stop, the nudge can't run.
	if v, _, err := c.ProcessNudge(d.NudgeID, d.BodyDigest); err != nil || v.State != "NOT_DELIVERED" {
		t.Fatalf("process after stop: %+v %v", v, err)
	}
	// Stop after the claim: reports the run, never NOT_DELIVERED.
	e := selfDraft(c, t, "key-stop-000002")
	re.gate = make(chan struct{})
	c.ProcessNudge(e.NudgeID, e.BodyDigest)
	waitRun(t, c, e.NudgeID, "RUNNING")
	if v, err := c.StopNudge(e.NudgeID); err != nil || v.State != "RUNNING" {
		t.Fatalf("stop after claim: %+v %v", v, err)
	}
	close(re.gate)
	waitRun(t, c, e.NudgeID, "COMPLETED")
}

// X15: submission write failures never send and never re-submit silently.
func TestProcessNudgeSubmitFailures(t *testing.T) {
	c, _, re := runCore(t, t.TempDir())
	orig := nudgeOpenFile
	t.Cleanup(func() { nudgeOpenFile = orig })
	d := selfDraft(c, t, "key-subfail-0001")
	nudgeOpenFile = func(name string, flag int, perm os.FileMode) (nudgeWriter, error) {
		if strings.HasSuffix(name, "submitted.json") {
			return nil, errors.New("read-only disk (test)")
		}
		return orig(name, flag, perm)
	}
	if _, _, err := c.ProcessNudge(d.NudgeID, d.BodyDigest); !errors.Is(err, ErrNudgeWrite) {
		t.Fatalf("create failure: %v", err)
	}
	// Torn: half the record reaches the disk, then an error.
	nudgeOpenFile = func(name string, flag int, perm os.FileMode) (nudgeWriter, error) {
		f, err := os.OpenFile(name, flag, perm)
		if err != nil || !strings.HasSuffix(name, "submitted.json") {
			return f, err
		}
		return partialFile{f}, nil
	}
	v, _, err := c.ProcessNudge(d.NudgeID, d.BodyDigest)
	nudgeOpenFile = orig
	if err != nil || v.State != "RECOVERY_REQUIRED" {
		t.Fatalf("torn submission: %+v %v", v, err)
	}
	time.Sleep(50 * time.Millisecond)
	if v, _, _ := c.ProcessNudge(d.NudgeID, d.BodyDigest); v.State != "RECOVERY_REQUIRED" || re.posts != 0 {
		t.Fatalf("after torn submission: %+v posts=%d", v, re.posts)
	}
}

// X17: the restore fence by byte offset; any change to the restored prefix locks processing.
func TestRestoreFencePreRestoreAndLocks(t *testing.T) {
	dir := t.TempDir()
	c, _, re := runCore(t, dir)
	old := selfDraft(c, t, "key-fence-old-01")
	journal := c.file(nudgeFile)
	raw, _ := os.ReadFile(journal)
	// Simulate tass.sh: the restored journal has no final newline (a valid last record).
	trimmed := []byte(strings.TrimRight(string(raw), "\n"))
	os.WriteFile(journal, trimmed, 0o600)
	writeFence := func(b []byte) {
		f := restoreFence{Schema: "tass.restore-fence/2", RestoredAt: "now", Backup: "b.tgz", JournalBytes: int64(len(b)), JournalSHA: sha256Hex(b)}
		j, _ := json.Marshal(f)
		os.WriteFile(c.file(restoreFenceFile), j, 0o600)
	}
	writeFence(trimmed)
	if _, _, err := c.ProcessNudge(old.NudgeID, old.BodyDigest); !errors.Is(err, ErrNudgeConflict) || !strings.Contains(err.Error(), "PRE_RESTORE") {
		t.Fatalf("pre-restore: %v", err)
	}
	// A new draft after the restore is processable (appended after the boundary, newline repaired).
	fresh := selfDraft(c, t, "key-fence-new-01")
	if _, accepted, err := c.ProcessNudge(fresh.NudgeID, fresh.BodyDigest); err != nil || !accepted {
		t.Fatalf("new draft after restore: %v", err)
	}
	waitRun(t, c, fresh.NudgeID, "COMPLETED")
	list, _ := c.NudgeHistory("")
	for _, n := range list.Nudges {
		if n.NudgeID == old.NudgeID && (!n.PreRestore || n.Processing.State != "PRE_RESTORE") {
			t.Fatalf("history: %+v", n)
		}
	}
	// Mutating the restored prefix locks all processing.
	b, _ := os.ReadFile(journal)
	b[10] ^= 1
	os.WriteFile(journal, b, 0o600)
	another := selfDraftRaw(c, "key-fence-lock-1")
	if _, _, err := c.ProcessNudge(another, "x"); !errors.Is(err, ErrProcessLocked) {
		t.Fatalf("mutated prefix: %v", err)
	}
	// Truncated below the boundary: locked.
	os.WriteFile(journal, trimmed[:len(trimmed)/2], 0o600)
	if why := c.processingLock(); why == "" {
		t.Fatal("truncated journal not locked")
	}
	// An invalid record inside the restored part: locked.
	bad := append([]byte("{not json\n"), trimmed...)
	os.WriteFile(journal, bad, 0o600)
	writeFence(bad)
	if why := c.processingLock(); !strings.Contains(why, "invalid record") {
		t.Fatalf("invalid restored record: %q", why)
	}
	// An unreadable fence: locked.
	os.WriteFile(c.file(restoreFenceFile), []byte("{"), 0o600)
	if why := c.processingLock(); why == "" {
		t.Fatal("unreadable fence not locked")
	}
	if re.posts != 1 {
		t.Fatalf("posts: %d", re.posts)
	}
}

// selfDraftRaw saves a draft, ignoring errors (the journal may be locked/mutated); returns its ID.
func selfDraftRaw(c *Core, key string) string {
	v, _ := c.SaveNudgeDraft(NudgeDraft{SourceThoughtID: "thought_1", RecipientID: "self", ReasonCode: "owner_request",
		RequestedAction: "x " + key, IdempotencyKey: key})
	return v.NudgeID
}

func TestCognitionRelinkRefusedWhileActive(t *testing.T) {
	c, _, re := runCore(t, t.TempDir())
	re.gate = make(chan struct{})
	d := selfDraft(c, t, "key-relink-0001")
	c.ProcessNudge(d.NudgeID, d.BodyDigest)
	waitRun(t, c, d.NudgeID, "RUNNING")
	if err := c.UnlinkCognition(); err == nil {
		t.Fatal("unlink while running")
	}
	close(re.gate)
	waitRun(t, c, d.NudgeID, "COMPLETED")
	if err := c.UnlinkCognition(); err != nil {
		t.Fatalf("unlink after: %v", err)
	}
}

func TestCoreDataLockOneCorePerFolder(t *testing.T) {
	dir := t.TempDir()
	a, err := AcquireDataLock(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := AcquireDataLock(dir); err == nil {
		t.Fatal("second Core got the lock")
	}
	a.Close()
	if b, err := AcquireDataLock(dir); err != nil {
		t.Fatal("lock not released on close")
	} else {
		b.Close()
	}
}

func sha256Hex(b []byte) string {
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

// TestRestoreFenceWrittenByTassSh: the fence that tass.sh restore_data writes (restore_fence_write,
// the same shell function) is read by Core exactly as designed: every draft already in the restored
// journal is PRE_RESTORE, and a draft saved afterwards can be processed.
func TestRestoreFenceWrittenByTassSh(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("no bash")
	}
	root := t.TempDir()
	dir := filepath.Join(root, "core")
	os.MkdirAll(dir, 0o700)
	c, _, re := runCore(t, dir)
	old := selfDraft(c, t, "key-shfence-old1")
	script, _ := filepath.Abs("../tass.sh")
	cmd := exec.Command("bash", "-c", `TASS_LIB_ONLY=1 source "$1" && restore_fence_write "$2" /x/backups/b.tgz`, "_", script, root)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("restore_fence_write: %v\n%s", err, out)
	}
	if fi, err := os.Stat(c.file(restoreFenceFile)); err != nil || fi.Mode().Perm() != 0o600 {
		t.Fatalf("fence file: %v %v", fi, err)
	}
	if why := c.processingLock(); why != "" {
		t.Fatalf("a correct shell fence must not lock processing: %q", why)
	}
	if _, _, err := c.ProcessNudge(old.NudgeID, old.BodyDigest); !errors.Is(err, ErrNudgeConflict) || !strings.Contains(err.Error(), "PRE_RESTORE") {
		t.Fatalf("pre-restore draft: %v", err)
	}
	fresh := selfDraft(c, t, "key-shfence-new1")
	if _, accepted, err := c.ProcessNudge(fresh.NudgeID, fresh.BodyDigest); err != nil || !accepted {
		t.Fatalf("draft after the restore: %v", err)
	}
	waitRun(t, c, fresh.NudgeID, "COMPLETED")
	if re.posts != 1 {
		t.Fatalf("posts: %d", re.posts)
	}
}
