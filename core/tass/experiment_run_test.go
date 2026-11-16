package tass

// Stage 3B runtime (owner decisions D4–D7 + the frozen snapshot): Core's admission, snapshot handling,
// authorization and receipts for one experiment arm, against a stand-in Engine that speaks the experiment
// routes. The real Engine and Identity are exercised by test/experiment_native.py.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/Arameair/Thought-Processor/core/engine"
)

// fakeExperiments is the stand-in Engine's experiment state.
type fakeExperiments struct {
	snaps      map[string]*engine.SnapshotInfo
	arms       map[string]*engine.ExperimentArm // "<exp>/<arm>"
	ins        map[string]engine.ArmIn
	captures   int
	armPosts   int
	modelCalls int
	gate       chan struct{}                              // if set, an arm waits here while RUNNING
	refuse     int                                        // HTTP status for arm POSTs (0 = accept)
	outcome    func(arm string) (state, why, http string) // nil = COMPLETED
	reply      map[string]string                          // arm → reply text
	tamper     bool                                       // the stored snapshot no longer verifies
	otherSHA   string                                     // the stored snapshot has another digest
}

func (e *runEngine) experimentHandle(w http.ResponseWriter, r *http.Request) bool {
	p := r.URL.Path
	if !strings.Contains(p, "/experiments/") {
		return false
	}
	parts := strings.Split(strings.Trim(p, "/"), "/")
	e.mu.Lock()
	fx := &e.fx
	if fx.snaps == nil {
		fx.snaps, fx.arms, fx.ins = map[string]*engine.SnapshotInfo{}, map[string]*engine.ExperimentArm{}, map[string]engine.ArmIn{}
	}
	record := func(exp string) map[string]any {
		rec := engine.ExperimentRecord{ExperimentID: exp, EntityID: "eng_1", EngineCommit: "test"}
		if s := fx.snaps[exp]; s != nil {
			rec.Snapshot = *s
			rec.Snapshot.Created = false
			if fx.tamper {
				rec.Snapshot = engine.SnapshotInfo{Present: true, Problem: "the snapshot file is not in its canonical form (it was changed after capture)"}
			} else if fx.otherSHA != "" {
				rec.Snapshot.SHA256 = fx.otherSHA
			}
		}
		for _, arm := range []string{"A", "B"} {
			if a := fx.arms[exp+"/"+arm]; a != nil {
				rec.Arms = append(rec.Arms, *a)
			} else {
				rec.Arms = append(rec.Arms, engine.ExperimentArm{Arm: arm, State: "NOT_CLAIMED"})
			}
		}
		return map[string]any{"experiment": rec}
	}
	switch {
	case r.Method == "GET" && len(parts) == 5: // /api/entities/eng_1/experiments/<exp>
		out := record(parts[4])
		e.mu.Unlock()
		json.NewEncoder(w).Encode(out)
	case r.Method == "POST" && strings.HasSuffix(p, "/snapshot"): // /api/entities/eng_1/thoughts/<tid>/experiments/<exp>/snapshot
		exp := parts[6]
		var in engine.SnapshotIn
		json.NewDecoder(r.Body).Decode(&in)
		s := fx.snaps[exp]
		created := false
		if s == nil {
			fx.captures++
			sum := sha256.Sum256([]byte(fmt.Sprintf("snapshot %s %d %s", exp, fx.captures, in.AttentionSHA256)))
			s = &engine.SnapshotInfo{Present: true, Verified: true, SHA256: hex.EncodeToString(sum[:]), CapturedAt: time.Date(2026, 11, 12, 9, 0, fx.captures, 0, time.UTC),
				ThoughtID: parts[4], ThoughtRevision: "2026-11-08T09:00:00Z", Evidence: 3, Notes: 1, EngineCommit: "test"}
			fx.snaps[exp], created = s, true
		}
		out := *s
		out.Created = created
		e.mu.Unlock()
		json.NewEncoder(w).Encode(map[string]any{"snapshot": out})
	case r.Method == "POST" && len(parts) == 9 && parts[7] == "arms":
		exp, arm := parts[6], parts[8]
		key := exp + "/" + arm
		var in engine.ArmIn
		json.NewDecoder(r.Body).Decode(&in)
		fx.armPosts++
		if fx.refuse != 0 {
			code := fx.refuse
			e.mu.Unlock()
			w.WriteHeader(code)
			json.NewEncoder(w).Encode(map[string]string{"error": "refused (test)"})
			return true
		}
		if fx.arms[key] != nil {
			out := record(exp)
			e.mu.Unlock()
			json.NewEncoder(w).Encode(out)
			return true
		}
		at := time.Date(2026, 11, 12, 9, 1, 0, 0, time.UTC)
		fx.ins[key] = in
		fx.arms[key] = &engine.ExperimentArm{Arm: arm, State: "RUNNING", RequestID: "mr_exp" + arm, SnapshotSHA: in.SnapshotSHA, Authorize: in.DecisionID, ClaimedAt: &at}
		gate := fx.gate
		e.mu.Unlock()
		if gate != nil {
			<-gate
		}
		e.mu.Lock()
		a := fx.arms[key]
		state, why, mh := "COMPLETED", "", "responded"
		if fx.outcome != nil {
			state, why, mh = fx.outcome(arm)
		}
		a.State, a.Reason, a.ModelHTTP = state, why, mh
		if mh != "none" {
			fx.modelCalls++
		}
		if state == "COMPLETED" {
			reply := fx.reply[arm]
			if reply == "" {
				reply = "NOTE: reply of arm " + arm + " for " + exp + "\nSTATE: keep"
			}
			sum := sha256.Sum256([]byte(reply))
			a.Result = &engine.ArmResult{RequestID: a.RequestID, Reply: reply, ReplySHA256: hex.EncodeToString(sum[:]), Parsed: true,
				Note: strings.TrimPrefix(strings.Split(reply, "\n")[0], "NOTE: "), State: "keep", Model: "local", ElapsedMS: 12, At: at}
		}
		out := record(exp)
		e.mu.Unlock()
		json.NewEncoder(w).Encode(out)
	default:
		e.mu.Unlock()
		w.WriteHeader(404)
	}
	return true
}

// expSetup: a processed run, a saved plan and the experiment grant enabled.
func expSetup(t *testing.T) (*Core, *fakeIdentitySvc, *runEngine, ExperimentView) {
	t.Helper()
	c, ids, re, nid, _ := processedRun(t)
	reconciled(re)
	v, err := c.SaveExperimentDraft(expDraft(nid, "exp-run-0001", expVariant))
	if err != nil {
		t.Fatal(err)
	}
	ids.mu.Lock()
	ids.expGranted = true
	ids.mu.Unlock()
	return c, ids, re, v
}

func confirmOf(v ExperimentView, arm string) ArmConfirm {
	return ArmConfirm{BodyDigest: v.BodyDigest, Confirm: "RUN " + arm + " " + v.ExperimentID}
}

func armOfView(v ExperimentView, arm string) ExperimentArm {
	for _, a := range v.Arms {
		if a.Arm == arm {
			return a
		}
	}
	return ExperimentArm{}
}

func waitArm(t *testing.T, c *Core, id, arm, want string) ExperimentView {
	t.Helper()
	var v ExperimentView
	for i := 0; i < 300; i++ {
		var err error
		if v, err = c.ExperimentByID(id); err == nil && armOfView(v, arm).State == want {
			return v
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("arm %s: want %s, got %+v", arm, want, armOfView(v, arm))
	return v
}

func (e *runEngine) fxCounts() (captures, armPosts, modelCalls int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.fx.captures, e.fx.armPosts, e.fx.modelCalls
}

func boundFor(ids *fakeIdentitySvc, action string) []string {
	ids.mu.Lock()
	defer ids.mu.Unlock()
	var out []string
	for _, b := range ids.bound {
		if strings.HasPrefix(b, action+" ") {
			out = append(out, strings.TrimPrefix(b, action+" "))
		}
	}
	return out
}

func TestExperimentRunAThenBOnOneFrozenSnapshot(t *testing.T) {
	c, ids, re, plan := expSetup(t)
	id := plan.ExperimentID
	thoughtBefore := thoughtJSON(t, c)
	// Before anything: A runnable, B not; no snapshot.
	v, err := c.ExperimentByID(id)
	if err != nil {
		t.Fatal(err)
	}
	if a, b := armOfView(v, "A"), armOfView(v, "B"); !a.Runnable || a.Why != "" || b.Runnable || !strings.Contains(b.Why, "Run A first") ||
		v.Frozen.Status != "NOT CAPTURED" || v.ModelRequests != 0 || a.Confirm != "RUN A "+id {
		t.Fatalf("before: A %+v B %+v frozen %+v", a, b, v.Frozen)
	}
	// A.
	v, created, err := c.RunExperimentArm(id, "A", confirmOf(plan, "A"))
	if err != nil || !created {
		t.Fatalf("run A: %v created=%v", err, created)
	}
	v = waitArm(t, c, id, "A", "COMPLETED")
	a := armOfView(v, "A")
	if cap, posts, calls := re.fxCounts(); cap != 1 || posts != 1 || calls != 1 {
		t.Fatalf("after A: %d captures, %d arm requests, %d model calls", cap, posts, calls)
	}
	if v.Frozen.Status != "FROZEN" || len(v.Frozen.SHA256) != 64 || v.Frozen.Reference != "snap-"+v.Frozen.SHA256[:12] || v.Frozen.CapturedAt == nil ||
		!strings.Contains(v.SnapshotLabel, "Frozen snapshot "+v.Frozen.Reference) || strings.Contains(v.SnapshotLabel, "as it is when") {
		t.Fatalf("frozen: %+v label %q", v.Frozen, v.SnapshotLabel)
	}
	if a.Reply == "" || !a.Parsed || a.Requests != 1 || a.Saved != "no" || a.RequestID == "" || a.DecisionID == "" || a.SnapshotSHA != v.Frozen.SHA256 || a.Runnable {
		t.Fatalf("A: %+v", a)
	}
	if b := armOfView(v, "B"); !b.Runnable || v.Comparison.Ready || v.ModelRequests != 1 || v.Status != "A completed. B has not run." {
		t.Fatalf("after A: B %+v comparison %+v status %q", b, v.Comparison, v.Status)
	}
	// The decision was bound to exactly this tuple.
	want := func(arm string) string {
		return engine.Binding(engine.ExperimentBinding{Action: ActionExperimentRun, OwnerEntityID: "ent_0123", EngineEntityID: "eng_1", ThoughtID: "thought_1",
			ExperimentID: id, Arm: arm, SnapshotSHA: v.Frozen.SHA256, Variable: "instruction", VariantSHA: plan.VariantSHA})
	}
	if got := boundFor(ids, ActionExperimentRun); len(got) != 1 || got[0] != want("A") {
		t.Fatalf("A's binding: %v", got)
	}
	// B: the same snapshot digest, a second and different decision, no second capture.
	if _, created, err := c.RunExperimentArm(id, "B", confirmOf(plan, "B")); err != nil || !created {
		t.Fatalf("run B: %v", err)
	}
	v = waitArm(t, c, id, "B", "COMPLETED")
	if cap, posts, calls := re.fxCounts(); cap != 1 || posts != 2 || calls != 2 {
		t.Fatalf("after B: %d captures, %d arm requests, %d model calls", cap, posts, calls)
	}
	re.mu.Lock()
	inA, inB := re.fx.ins[id+"/A"], re.fx.ins[id+"/B"]
	re.mu.Unlock()
	if inA.SnapshotSHA != v.Frozen.SHA256 || inB.SnapshotSHA != inA.SnapshotSHA || inA.Variant != expVariant || inB.Variant != expVariant ||
		inA.DecisionID == inB.DecisionID || inA.VariantSHA != plan.VariantSHA {
		t.Fatalf("arm requests: A %+v B %+v", inA, inB)
	}
	got := boundFor(ids, ActionExperimentRun)
	if len(got) != 2 || !((got[0] == want("A") && got[1] == want("B")) || (got[0] == want("B") && got[1] == want("A"))) {
		t.Fatalf("bindings: %v", got)
	}
	b := armOfView(v, "B")
	if !v.Comparison.Ready || v.Comparison.SameReply || v.ModelRequests != 2 || v.Status != "Both arms completed." || b.Reply == a.Reply || b.Saved != "no" ||
		!strings.Contains(v.Comparison.Text, "only input that differed is the instruction sentence") {
		t.Fatalf("comparison: %+v B %+v", v.Comparison, b)
	}
	// Nothing was written to the Thought.
	if thoughtJSON(t, c) != thoughtBefore {
		t.Fatal("the Thought changed")
	}
	// A confirmed replay of either arm shows the record: no new decision, no Engine write, no call.
	decs := len(boundFor(ids, ActionExperimentRun))
	nonGet := re.nonGet.Load()
	for _, arm := range []string{"A", "B"} {
		if _, created, err := c.RunExperimentArm(id, arm, confirmOf(plan, arm)); err != nil || created {
			t.Fatalf("replay %s: created=%v %v", arm, created, err)
		}
	}
	if len(boundFor(ids, ActionExperimentRun)) != decs || re.nonGet.Load() != nonGet {
		t.Fatal("a replay minted a decision or wrote to the Engine")
	}
	// Receipts are private files.
	for _, f := range []string{"snapshot.json", "A.submitted.json", "A.outcome.json", "B.submitted.json", "B.outcome.json"} {
		if fi, err := os.Stat(c.expRunPath(id, f)); err != nil || fi.Mode().Perm() != 0o600 {
			t.Errorf("receipt %s: %v", f, err)
		}
	}
}

func TestExperimentSameReplyIsReportedAsNoDifference(t *testing.T) {
	c, _, re, plan := expSetup(t)
	re.mu.Lock()
	re.fx.reply = map[string]string{"A": "NOTE: identical\nSTATE: keep", "B": "NOTE: identical\nSTATE: keep"}
	re.mu.Unlock()
	c.RunExperimentArm(plan.ExperimentID, "A", confirmOf(plan, "A"))
	waitArm(t, c, plan.ExperimentID, "A", "COMPLETED")
	c.RunExperimentArm(plan.ExperimentID, "B", confirmOf(plan, "B"))
	v := waitArm(t, c, plan.ExperimentID, "B", "COMPLETED")
	if !v.Comparison.Ready || !v.Comparison.SameReply || !strings.HasPrefix(v.Comparison.Text, "No difference") {
		t.Fatalf("comparison: %+v", v.Comparison)
	}
}

func TestExperimentArmRefusalsSendNothing(t *testing.T) {
	type env struct {
		c    *Core
		ids  *fakeIdentitySvc
		re   *runEngine
		plan ExperimentView
	}
	cases := []struct {
		name string
		arm  string
		prep func(e *env) ArmConfirm
		is   error
		say  string
	}{
		{"no confirmation", "A", func(e *env) ArmConfirm { return ArmConfirm{BodyDigest: e.plan.BodyDigest} }, ErrExperimentConflict, "confirmation"},
		{"confirmation for the other arm", "A", func(e *env) ArmConfirm { return confirmOf(e.plan, "B") }, ErrExperimentConflict, "confirmation"},
		{"confirmation for another plan", "A", func(e *env) ArmConfirm {
			return ArmConfirm{BodyDigest: e.plan.BodyDigest, Confirm: "RUN A exp_00000000000000000000"}
		}, ErrExperimentConflict, "confirmation"},
		{"another plan's digest", "A", func(e *env) ArmConfirm {
			return ArmConfirm{BodyDigest: strings.Repeat("0", 64), Confirm: "RUN A " + e.plan.ExperimentID}
		}, ErrExperimentConflict, "confirmation"},
		{"grant not enabled (the default)", "A", func(e *env) ArmConfirm {
			e.ids.mu.Lock()
			e.ids.expGranted = false
			e.ids.mu.Unlock()
			return confirmOf(e.plan, "A")
		}, ErrExperimentNotGranted, "cognition.experiment-run"},
		{"the nudge grant is not the experiment grant", "A", func(e *env) ArmConfirm {
			e.ids.mu.Lock()
			e.ids.expGranted, e.ids.processGranted = false, true
			e.ids.mu.Unlock()
			return confirmOf(e.plan, "A")
		}, ErrExperimentNotGranted, ""},
		{"Identity locked", "A", func(e *env) ArmConfirm { e.ids.set("LOCKED"); return confirmOf(e.plan, "A") }, nil, ""},
		{"B before A", "B", func(e *env) ArmConfirm { return confirmOf(e.plan, "B") }, ErrExperimentRefused, "run A first"},
		{"another owner", "A", func(e *env) ArmConfirm {
			e.ids.mu.Lock()
			e.ids.entityID = "ent_other"
			e.ids.mu.Unlock()
			return confirmOf(e.plan, "A")
		}, nil, ""},
		{"PRE_RESTORE (fence records the experiment journal)", "A", func(e *env) ArmConfirm {
			b, _ := os.ReadFile(e.c.file(experimentFile))
			n := int64(len(b))
			f, _ := json.Marshal(map[string]any{"schema": "tass.restore-fence/2", "restored_at": "2026-11-12T00:00:00Z", "backup": "b.tgz",
				"journal_bytes": fileLen(e.c.file(nudgeFile)), "journal_sha256": sha256File(e.c.file(nudgeFile)), "experiments_bytes": n, "experiments_sha256": sha256Hex(b)})
			os.WriteFile(e.c.file(restoreFenceFile), f, 0o600)
			return confirmOf(e.plan, "A")
		}, nil, "PRE_RESTORE"},
		{"PRE_RESTORE (older fence without experiment fields)", "A", func(e *env) ArmConfirm {
			f, _ := json.Marshal(map[string]any{"schema": "tass.restore-fence/2", "restored_at": time.Now().Add(time.Hour).UTC().Format(time.RFC3339), "backup": "b.tgz",
				"journal_bytes": fileLen(e.c.file(nudgeFile)), "journal_sha256": sha256File(e.c.file(nudgeFile))})
			os.WriteFile(e.c.file(restoreFenceFile), f, 0o600)
			return confirmOf(e.plan, "A")
		}, nil, "PRE_RESTORE"},
		{"the restored journal part changed", "A", func(e *env) ArmConfirm {
			b, _ := os.ReadFile(e.c.file(experimentFile))
			f, _ := json.Marshal(map[string]any{"schema": "tass.restore-fence/2", "restored_at": "2026-11-12T00:00:00Z", "backup": "b.tgz",
				"journal_bytes": fileLen(e.c.file(nudgeFile)), "journal_sha256": sha256File(e.c.file(nudgeFile)), "experiments_bytes": int64(len(b)), "experiments_sha256": strings.Repeat("0", 64)})
			os.WriteFile(e.c.file(restoreFenceFile), f, 0o600)
			return confirmOf(e.plan, "A")
		}, ErrProcessLocked, "changed since the restore"},
		{"unsafe plan journal", "A", func(e *env) ArmConfirm { os.Chmod(e.c.file(experimentFile), 0o644); return confirmOf(e.plan, "A") }, ErrExperimentJournal, ""},
		{"the Engine's snapshot does not verify", "A", func(e *env) ArmConfirm {
			e.re.mu.Lock()
			e.re.fx.tamper = true
			e.re.mu.Unlock()
			return confirmOf(e.plan, "A")
		}, ErrExperimentRefused, "nothing was sent"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, ids, re, plan := expSetup(t)
			e := &env{c, ids, re, plan}
			in := tc.prep(e)
			thought := ""
			if tc.name != "Identity locked" && tc.name != "another owner" {
				thought = thoughtJSON(t, c)
			}
			_, created, err := c.RunExperimentArm(plan.ExperimentID, tc.arm, in)
			if err == nil || created {
				t.Fatalf("accepted (created=%v)", created)
			}
			if tc.is != nil && !errors.Is(err, tc.is) {
				t.Errorf("err = %v, want %v", err, tc.is)
			}
			if tc.say != "" && !strings.Contains(err.Error(), tc.say) {
				t.Errorf("err = %v, want it to say %q", err, tc.say)
			}
			time.Sleep(30 * time.Millisecond)
			if _, posts, calls := re.fxCounts(); posts != 0 || calls != 0 {
				t.Errorf("%d arm requests, %d model calls", posts, calls)
			}
			if len(boundFor(ids, ActionExperimentRun)) != 0 {
				t.Error("a decision was minted")
			}
			if _, serr := os.Stat(c.expRunPath(plan.ExperimentID, tc.arm+".submitted.json")); serr == nil {
				t.Error("a submission receipt was written")
			}
			if cap, _, _ := re.fxCounts(); cap != 0 && tc.name != "the Engine's snapshot does not verify" {
				t.Errorf("a snapshot was captured (%d)", cap)
			}
			if thought != "" && thoughtJSON(t, c) != thought {
				t.Error("the Thought changed")
			}
		})
	}
}

func fileLen(p string) int64 {
	fi, err := os.Stat(p)
	if err != nil {
		return 0
	}
	return fi.Size()
}

func sha256File(p string) string {
	b, _ := os.ReadFile(p)
	return sha256Hex(b)
}

func TestExperimentBIsRefusedUnlessTheSameSnapshotStillVerifies(t *testing.T) {
	for _, tc := range []struct {
		name string
		prep func(c *Core, re *runEngine, id string)
	}{
		{"the Engine's stored snapshot was changed", func(c *Core, re *runEngine, id string) {
			re.mu.Lock()
			re.fx.tamper = true
			re.mu.Unlock()
		}},
		{"the Engine's stored snapshot has another digest", func(c *Core, re *runEngine, id string) {
			re.mu.Lock()
			re.fx.otherSHA = strings.Repeat("b", 64)
			re.mu.Unlock()
		}},
		{"the Engine lost the snapshot", func(c *Core, re *runEngine, id string) {
			re.mu.Lock()
			delete(re.fx.snaps, id)
			re.mu.Unlock()
		}},
		{"Core's snapshot record is gone (B never captures)", func(c *Core, re *runEngine, id string) { os.Remove(c.expRunPath(id, "snapshot.json")) }},
		{"Core's snapshot record is torn", func(c *Core, re *runEngine, id string) {
			os.WriteFile(c.expRunPath(id, "snapshot.json"), []byte("{"), 0o600)
		}},
		{"Core's snapshot record names another digest", func(c *Core, re *runEngine, id string) {
			var s expSnapshotRec
			b, _ := os.ReadFile(c.expRunPath(id, "snapshot.json"))
			json.Unmarshal(b, &s)
			s.SnapshotSHA = strings.Repeat("c", 64)
			nb, _ := json.Marshal(s)
			os.WriteFile(c.expRunPath(id, "snapshot.json"), nb, 0o600)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, ids, re, plan := expSetup(t)
			id := plan.ExperimentID
			c.RunExperimentArm(id, "A", confirmOf(plan, "A"))
			waitArm(t, c, id, "A", "COMPLETED")
			tc.prep(c, re, id)
			_, created, err := c.RunExperimentArm(id, "B", confirmOf(plan, "B"))
			if !errors.Is(err, ErrExperimentRefused) || created || !strings.Contains(strings.ToLower(err.Error()), "nothing was sent") {
				t.Fatalf("B: created=%v err=%v", created, err)
			}
			time.Sleep(30 * time.Millisecond)
			if cap, posts, calls := re.fxCounts(); cap != 1 || posts != 1 || calls != 1 {
				t.Fatalf("%d captures, %d arm requests, %d model calls (want 1, 1, 1)", cap, posts, calls)
			}
			if len(boundFor(ids, ActionExperimentRun)) != 1 {
				t.Fatal("a decision was minted for B")
			}
			if v, err := c.ExperimentByID(id); err == nil {
				if b := armOfView(v, "B"); b.Runnable || b.State != "NOT RUN" {
					t.Fatalf("the page still offers B: %+v", b)
				}
			}
		})
	}
}

func TestExperimentFailedUncertainOrUndeliveredAIsFinal(t *testing.T) {
	for _, tc := range []struct {
		name, state, http string
		refuse            int
		calls             int
	}{
		{"FAILED", "FAILED", "none", 0, 0},
		{"UNCERTAIN", "UNCERTAIN", "attempt_uncertain", 0, 1},
		{"RECOVERY_REQUIRED", "RECOVERY_REQUIRED", "unknown", 0, 1},
		{"refused by the Engine", "FAILED", "none", 403, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, ids, re, plan := expSetup(t)
			id := plan.ExperimentID
			re.mu.Lock()
			re.fx.refuse = tc.refuse
			re.fx.outcome = func(string) (string, string, string) { return tc.state, "test " + tc.state, tc.http }
			re.mu.Unlock()
			c.RunExperimentArm(id, "A", confirmOf(plan, "A"))
			v := waitArm(t, c, id, "A", tc.state)
			a, b := armOfView(v, "A"), armOfView(v, "B")
			if a.Reply != "" || a.Runnable || b.Runnable || !strings.Contains(b.Why, "final") || !strings.HasPrefix(v.Status, "Ended: A is "+tc.state) || v.Comparison.Ready {
				t.Fatalf("A %+v B %+v status %q", a, b, v.Status)
			}
			re.mu.Lock()
			re.fx.refuse, re.fx.outcome = 0, nil
			re.mu.Unlock()
			// Never retried: A again shows the record; B is refused.
			posts0 := re.nonGet.Load()
			if _, created, err := c.RunExperimentArm(id, "A", confirmOf(plan, "A")); err != nil || created {
				t.Fatalf("A again: created=%v %v", created, err)
			}
			if _, _, err := c.RunExperimentArm(id, "B", confirmOf(plan, "B")); !errors.Is(err, ErrExperimentRefused) {
				t.Fatalf("B after %s: %v", tc.state, err)
			}
			if re.nonGet.Load() != posts0 || len(boundFor(ids, ActionExperimentRun)) != 1 {
				t.Fatal("something was sent or authorized after a final A")
			}
			if _, _, calls := re.fxCounts(); calls != tc.calls {
				t.Fatalf("model calls: %d", calls)
			}
		})
	}
	t.Run("delivery not confirmed, then never claimed: NOT_DELIVERED after the approval expired", func(t *testing.T) {
		c, _, re, plan := expSetup(t)
		id := plan.ExperimentID
		re.mu.Lock()
		re.fx.refuse = 503 // not a definite refusal: Core must not guess
		re.mu.Unlock()
		c.RunExperimentArm(id, "A", confirmOf(plan, "A"))
		time.Sleep(80 * time.Millisecond)
		c.StopNudgeDispatch()
		v, _ := c.ExperimentByID(id)
		if a := armOfView(v, "A"); a.State != "SUBMITTED" || a.Runnable {
			t.Fatalf("right after: %+v", a)
		}
		re.mu.Lock()
		re.fx.refuse = 0
		re.mu.Unlock()
		// Still open: it holds the one slot (a nudge is refused), and it is not sent again.
		posts := re.nonGet.Load()
		d := selfDraft(c, t, "key-exp-busy-0001")
		if _, _, err := c.ProcessNudge(d.NudgeID, d.BodyDigest); !errors.Is(err, ErrProcessBusy) {
			t.Fatalf("a nudge was admitted beside an open arm: %v", err)
		}
		real := c.now
		c.now = func() time.Time { return real().Add(experimentUndeliveredAfter + time.Second) }
		v, _ = c.ExperimentByID(id)
		a := armOfView(v, "A")
		if a.State != "NOT_DELIVERED" || a.ModelHTTP != "none" || a.Requests != 0 || !strings.Contains(armOfView(v, "B").Why, "final") {
			t.Fatalf("after expiry: %+v", a)
		}
		if re.nonGet.Load() != posts {
			t.Fatal("the undelivered arm was sent again")
		}
		c.now = real
		if v, _ = c.ExperimentByID(id); armOfView(v, "A").State != "NOT_DELIVERED" {
			t.Fatal("NOT_DELIVERED was not recorded")
		}
	})
}

func TestExperimentArmsAndNudgesShareOneSlot(t *testing.T) {
	c, _, re, plan := expSetup(t)
	id := plan.ExperimentID
	nid := plan.BaselineNudgeID
	other, err := c.SaveExperimentDraft(expDraft(nid, "exp-run-0002", "Say exactly what evidence is missing."))
	if err != nil {
		t.Fatal(err)
	}
	// An arm is running: a nudge, the other plan's A, and relinking are refused.
	gate := make(chan struct{})
	re.mu.Lock()
	re.fx.gate = gate
	re.mu.Unlock()
	c.RunExperimentArm(id, "A", confirmOf(plan, "A"))
	waitArm(t, c, id, "A", "RUNNING")
	d := selfDraft(c, t, "key-exp-slot-0001")
	if _, _, err := c.ProcessNudge(d.NudgeID, d.BodyDigest); !errors.Is(err, ErrProcessBusy) {
		t.Fatalf("nudge beside a running arm: %v", err)
	}
	if _, _, err := c.RunExperimentArm(other.ExperimentID, "A", confirmOf(other, "A")); !errors.Is(err, ErrProcessBusy) {
		t.Fatalf("a second arm beside a running arm: %v", err)
	}
	if _, _, err := c.RunExperimentArm(id, "B", confirmOf(plan, "B")); !errors.Is(err, ErrProcessBusy) {
		t.Fatalf("B while A runs: %v", err)
	}
	if err := c.UnlinkCognition(); err == nil {
		t.Fatal("unlinked under a running arm")
	}
	if v, _ := c.ExperimentByID(other.ExperimentID); armOfView(v, "A").Runnable || !strings.Contains(armOfView(v, "A").Why, "Busy") {
		t.Fatalf("the other plan's page offers A while the slot is taken: %+v", armOfView(v, "A"))
	}
	close(gate)
	re.mu.Lock()
	re.fx.gate = nil
	re.mu.Unlock()
	waitArm(t, c, id, "A", "COMPLETED")
	// A nudge is running: an arm is refused.
	re.mu.Lock()
	re.gate = make(chan struct{})
	ngate := re.gate
	re.mu.Unlock()
	if _, _, err := c.ProcessNudge(d.NudgeID, d.BodyDigest); err != nil {
		t.Fatal(err)
	}
	waitRun(t, c, d.NudgeID, "RUNNING")
	if _, _, err := c.RunExperimentArm(id, "B", confirmOf(plan, "B")); !errors.Is(err, ErrProcessBusy) {
		t.Fatalf("an arm beside a running nudge: %v", err)
	}
	close(ngate)
	waitRun(t, c, d.NudgeID, "COMPLETED")
	if _, created, err := c.RunExperimentArm(id, "B", confirmOf(plan, "B")); err != nil || !created {
		t.Fatalf("B after the nudge finished: %v", err)
	}
	waitArm(t, c, id, "B", "COMPLETED")
	if cap, posts, calls := re.fxCounts(); cap != 1 || posts != 2 || calls != 2 {
		t.Fatalf("%d captures, %d arm requests, %d model calls", cap, posts, calls)
	}
}

func TestExperimentGrantRevokedBetweenAAndB(t *testing.T) {
	c, ids, re, plan := expSetup(t)
	id := plan.ExperimentID
	c.RunExperimentArm(id, "A", confirmOf(plan, "A"))
	waitArm(t, c, id, "A", "COMPLETED")
	ids.mu.Lock()
	ids.expGranted = false
	ids.mu.Unlock()
	if _, _, err := c.RunExperimentArm(id, "B", confirmOf(plan, "B")); !errors.Is(err, ErrExperimentNotGranted) {
		t.Fatalf("B after revoke: %v", err)
	}
	v, _ := c.ExperimentByID(id)
	if b := armOfView(v, "B"); b.Runnable || !strings.Contains(b.Why, "NOT AUTHORIZED") || armOfView(v, "A").Reply == "" {
		t.Fatalf("B %+v", b)
	}
	if _, posts, _ := re.fxCounts(); posts != 1 {
		t.Fatalf("arm requests: %d", posts)
	}
	// Re-enabled: B runs on the same recorded snapshot.
	ids.mu.Lock()
	ids.expGranted = true
	ids.mu.Unlock()
	if _, created, err := c.RunExperimentArm(id, "B", confirmOf(plan, "B")); err != nil || !created {
		t.Fatal(err)
	}
	waitArm(t, c, id, "B", "COMPLETED")
	if cap, _, _ := re.fxCounts(); cap != 1 {
		t.Fatalf("captures: %d", cap)
	}
}

func TestExperimentSurvivesACoreRestartBetweenArmsWithoutRecapture(t *testing.T) {
	c, ids, re, plan := expSetup(t)
	id := plan.ExperimentID
	c.RunExperimentArm(id, "A", confirmOf(plan, "A"))
	v := waitArm(t, c, id, "A", "COMPLETED")
	sha := v.Frozen.SHA256
	c.StopNudgeDispatch()
	// A new Core process on the same data folder.
	legacy := newFakeEngine()
	t.Cleanup(legacy.srv.Close)
	c2 := newIdentityCore(t, c.cfg.DataDir, ids, legacy)
	c2.engine = engine.New(re.srv.URL)
	c2.engine.Key = "testkey"
	t.Cleanup(c2.StopNudgeDispatch)
	v2, err := c2.ExperimentByID(id)
	if err != nil || armOfView(v2, "A").State != "COMPLETED" || armOfView(v2, "A").Reply == "" || v2.Frozen.SHA256 != sha || !armOfView(v2, "B").Runnable {
		t.Fatalf("after restart: %v %+v", err, v2.Frozen)
	}
	if _, created, err := c2.RunExperimentArm(id, "B", confirmOf(plan, "B")); err != nil || !created {
		t.Fatal(err)
	}
	waitArm(t, c2, id, "B", "COMPLETED")
	re.mu.Lock()
	inB := re.fx.ins[id+"/B"]
	re.mu.Unlock()
	if cap, posts, _ := re.fxCounts(); cap != 1 || posts != 2 || inB.SnapshotSHA != sha {
		t.Fatalf("%d captures, %d arm requests, B on %s", cap, posts, inB.SnapshotSHA)
	}
}

func TestExperimentConcurrentConfirmationsSendOneArmOnce(t *testing.T) {
	c, ids, re, plan := expSetup(t)
	id := plan.ExperimentID
	done := make(chan bool, 8)
	for i := 0; i < 8; i++ {
		go func() {
			_, created, _ := c.RunExperimentArm(id, "A", confirmOf(plan, "A"))
			done <- created
		}()
	}
	n := 0
	for i := 0; i < 8; i++ {
		if <-done {
			n++
		}
	}
	waitArm(t, c, id, "A", "COMPLETED")
	if cap, posts, calls := re.fxCounts(); n != 1 || cap != 1 || posts != 1 || calls != 1 || len(boundFor(ids, ActionExperimentRun)) != 1 {
		t.Fatalf("%d accepted, %d captures, %d arm requests, %d model calls", n, cap, posts, calls)
	}
}

// The runtime browser fixture (test/browser/development_experiment_run.json) is Core's real output at each
// step of one experiment. Regenerate with STAGE3B_WRITE_FIXTURE=1.
func TestExperimentRunBrowserFixtureIsCoreOutput(t *testing.T) {
	c, _, re, plan := expSetup(t)
	id := plan.ExperimentID
	run, err := c.DevelopmentRun(plan.BaselineNudgeID)
	if err != nil {
		t.Fatal(err)
	}
	fx := map[string]any{"run": run}
	get := func(name string) ExperimentView {
		v, err := c.ExperimentByID(id)
		if err != nil {
			t.Fatal(err)
		}
		fx[name] = v
		return v
	}
	get("ready")
	gate := make(chan struct{})
	re.mu.Lock()
	re.fx.gate = gate
	re.mu.Unlock()
	accepted, _, err := c.RunExperimentArm(id, "A", confirmOf(plan, "A"))
	if err != nil {
		t.Fatal(err)
	}
	_ = accepted
	waitArm(t, c, id, "A", "RUNNING")
	get("a_running")
	close(gate)
	re.mu.Lock()
	re.fx.gate = nil
	re.mu.Unlock()
	waitArm(t, c, id, "A", "COMPLETED")
	get("after_a")
	c.RunExperimentArm(id, "B", confirmOf(plan, "B"))
	waitArm(t, c, id, "B", "COMPLETED")
	get("after_b")
	list, err := c.Experiments(plan.BaselineNudgeID)
	if err != nil {
		t.Fatal(err)
	}
	fx["list_after_b"] = list
	// A second plan whose A ends UNCERTAIN.
	p2, err := c.SaveExperimentDraft(expDraft(plan.BaselineNudgeID, "exp-run-fixture-2", "Say exactly what evidence is missing."))
	if err != nil {
		t.Fatal(err)
	}
	re.mu.Lock()
	re.fx.outcome = func(string) (string, string, string) {
		return "UNCERTAIN", "interrupted after the HTTP attempt record; the model may have received it", "attempt_uncertain"
	}
	re.mu.Unlock()
	c.RunExperimentArm(p2.ExperimentID, "A", confirmOf(p2, "A"))
	fx["a_uncertain"] = waitArm(t, c, p2.ExperimentID, "A", "UNCERTAIN")

	norm := func(b []byte) string {
		s := regexp.MustCompile(`(nudge|exp)_[0-9a-f]{20}`).ReplaceAllString(string(b), "${1}_X")
		s = regexp.MustCompile(`(mr|redeem)_[0-9a-f]{8}`).ReplaceAllString(s, "${1}_X")
		s = regexp.MustCompile(`dec_b[0-9]+`).ReplaceAllString(s, "dec_X")
		s = regexp.MustCompile(`"[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9:.]+(Z|[+-][0-9:]+)"`).ReplaceAllString(s, `"T"`)
		s = regexp.MustCompile(`[0-9a-f]{64}`).ReplaceAllString(s, "H")
		return regexp.MustCompile(`snap-[0-9a-f]{12}`).ReplaceAllString(s, "snap-X")
	}
	path := "../test/browser/development_experiment_run.json"
	want, _ := json.MarshalIndent(fx, "", "  ")
	if os.Getenv("STAGE3B_WRITE_FIXTURE") == "1" {
		if err := os.WriteFile(path, append(want, '\n'), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var g, w any
	if json.Unmarshal(got, &g) != nil {
		t.Fatal("fixture is not JSON")
	}
	json.Unmarshal(want, &w)
	gb, _ := json.MarshalIndent(g, "", "  ")
	wb, _ := json.MarshalIndent(w, "", "  ")
	if norm(gb) != norm(wb) {
		t.Fatalf("the runtime browser fixture no longer matches Core's output; run with STAGE3B_WRITE_FIXTURE=1 and re-check the browser test")
	}
}
