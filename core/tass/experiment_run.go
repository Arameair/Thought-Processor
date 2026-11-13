package tass

// Stage 3B runtime (#120; owner decisions D4–D7 and the frozen-snapshot requirement, 2026-11-12): run ONE
// arm of one saved experiment plan, with the owner's explicit confirmation.
//
//   POST /api/development/experiments/{id}/arms/{A|B}
//     X1 the plan is this owner's and linked Engine entity's; the page's confirmation names this arm and
//        this plan; not PRE_RESTORE; nothing locked
//     X2 this arm has not been submitted (an existing arm is returned, never sent again)
//     X3 order: B only after A COMPLETED; a failed, uncertain or undelivered A ends the experiment
//     X4 Identity is unlocked and holds cognition.experiment.run (checked before anything is captured)
//     X5 no other active run, nudge or experiment arm (one model slot)
//     X6 the snapshot: A captures it ONCE at the Engine and records its digest here before anything is
//        sent; B must find that record AND the Engine's stored snapshot verified with the same digest.
//        B never causes a capture. Any doubt → nothing runs.
//     X7 ONE bound Identity decision for exactly (owner, engine entity, Thought, experiment, arm,
//        snapshot digest, variable, variant digest)
//     X8 submitted receipt (O_EXCL, durable) → 202; this process (only) sends it to the Engine once
//
// Core's receipts are immutable files in <data>/experiment-runs/<exp>/ (snapshot.json, A.submitted.json,
// A.outcome.json, B.…). Core never re-sends: after a restart a submitted arm is only READ from the Engine.
// A submitted arm the Engine never claimed can no longer be claimed once its decision has expired; it is
// then recorded as NOT_DELIVERED (the model was never asked) and is final.
// Nothing here writes a Thought, a Note, the advisory graph, settings or grants.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Arameair/Thought-Processor/core/engine"
)

const (
	experimentRunSchema = "thought-processor.experiment-run/1"
	experimentRunDir    = "experiment-runs"
	ActionExperimentRun = "cognition.experiment.run"
	// A bound decision lives 120 s at Identity. After this long an arm the Engine never claimed cannot be
	// claimed any more (only the process that submitted it sends it, once).
	experimentUndeliveredAfter = 150 * time.Second
)

var (
	ErrExperimentNotGranted = errors.New("Identity has not granted experiment runs: enable \"cognition.experiment-run\" on Identity's page (127.0.0.1:8796) with your passphrase")
	ErrExperimentRefused    = errors.New("this experiment arm can't be run")
)

// ArmConfirm is what the page sends when the owner confirms one arm.
type ArmConfirm struct {
	BodyDigest string `json:"body_digest"` // the saved plan's digest, as shown on the page
	Confirm    string `json:"confirm"`     // exactly "RUN <arm> <experiment_id>"
}

type expSnapshotRec struct {
	Schema         string    `json:"schema"`
	ExperimentID   string    `json:"experiment_id"`
	OwnerEntityID  string    `json:"owner_entity_id"`
	EngineEntityID string    `json:"engine_entity_id"`
	ThoughtID      string    `json:"thought_id"`
	SnapshotSHA    string    `json:"snapshot_sha256"`
	CapturedAt     time.Time `json:"captured_at"`
	Evidence       int       `json:"evidence_lines"`
	Notes          int       `json:"note_lines"`
	ThoughtRev     string    `json:"thought_revision"`
	EngineCommit   string    `json:"engine_commit"`
	VariantSHA     string    `json:"variant_sha256"`
	At             time.Time `json:"at"`
}
type expSubmittedRec struct {
	Schema         string    `json:"schema"`
	ExperimentID   string    `json:"experiment_id"`
	Arm            string    `json:"arm"`
	OwnerEntityID  string    `json:"owner_entity_id"`
	EngineEntityID string    `json:"engine_entity_id"`
	ThoughtID      string    `json:"thought_id"`
	SnapshotSHA    string    `json:"snapshot_sha256"`
	VariantSHA     string    `json:"variant_sha256"`
	Binding        string    `json:"binding"`
	DecisionID     string    `json:"identity_decision"`
	At             time.Time `json:"at"`
}
type expOutcomeRec struct {
	Schema string               `json:"schema"`
	Arm    engine.ExperimentArm `json:"engine_arm"`
	Detail string               `json:"detail,omitempty"`
	At     time.Time            `json:"at"`
}

func (c *Core) expRunPath(exp, name string) string {
	return filepath.Join(c.cfg.DataDir, experimentRunDir, exp, name)
}

func expReadOnce(path string, v any) (exists, ok bool) {
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, false
	}
	if err != nil || len(strings.TrimSpace(string(b))) == 0 || json.Unmarshal(b, v) != nil {
		return true, false
	}
	var s struct {
		Schema string `json:"schema"`
	}
	json.Unmarshal(b, &s)
	return true, s.Schema == experimentRunSchema
}

func expTerminal(state string) bool {
	switch state {
	case "COMPLETED", "FAILED", "UNCERTAIN", "NOT_DELIVERED", "RECOVERY_REQUIRED":
		return true
	}
	return false
}

// armState is one arm as Core knows it.
type armState struct {
	State     string // NOT RUN | SUBMITTED | RUNNING | COMPLETED | FAILED | UNCERTAIN | NOT_DELIVERED | RECOVERY_REQUIRED
	Detail    string
	Submitted *expSubmittedRec
	Engine    *engine.ExperimentArm
}

// armView derives an arm from Core's files and (only while it is still open) ONE Engine read. It never
// sends anything. It may record what it observed (the outcome receipt).
func (c *Core) armView(exp, arm string) armState {
	var sub expSubmittedRec
	sex, sok := expReadOnce(c.expRunPath(exp, arm+".submitted.json"), &sub)
	if !sex {
		return armState{State: "NOT RUN"}
	}
	if !sok || sub.ExperimentID != exp || sub.Arm != arm {
		return armState{State: "RECOVERY_REQUIRED", Detail: "the submission record is unreadable; this arm will never be sent"}
	}
	st := armState{Submitted: &sub}
	var out expOutcomeRec
	if oex, ook := expReadOnce(c.expRunPath(exp, arm+".outcome.json"), &out); oex {
		if !ook {
			st.State, st.Detail = "RECOVERY_REQUIRED", "the outcome record is unreadable"
			return st
		}
		a := out.Arm
		st.Engine, st.State, st.Detail = &a, a.State, nonEmpty(a.Reason, out.Detail)
		return st
	}
	rs := c.nrs()
	rs.mu.Lock()
	sending := rs.dispatching["exp:"+exp+"/"+arm]
	rs.mu.Unlock()
	rec, _, err := c.engine.ExperimentRecord(sub.EngineEntityID, exp)
	if err != nil {
		st.State, st.Detail = "SUBMITTED", "accepted locally; the Engine can't be read right now ("+err.Error()+")"
		return st
	}
	a := rec.Arm(arm)
	switch {
	case a.State == "NOT_CLAIMED" && sending:
		st.State, st.Detail = "SUBMITTED", "accepted locally; being delivered to the Engine"
	case a.State == "NOT_CLAIMED" && c.now().Sub(sub.At) > experimentUndeliveredAfter:
		// The approval expired unused and nothing will send this arm again: final, the model was never asked.
		nd := engine.ExperimentArm{Arm: arm, State: "NOT_DELIVERED", ModelHTTP: "none",
			Reason: "the Engine never claimed this arm and its approval has expired; the model was never asked"}
		c.recordArmOutcome(exp, arm, nd, "")
		st.Engine, st.State, st.Detail = &nd, nd.State, nd.Reason
	case a.State == "NOT_CLAIMED":
		st.State, st.Detail = "SUBMITTED", "accepted locally; not claimed by the Engine yet"
	case a.State == "RUNNING":
		st.State, st.Detail, st.Engine = "RUNNING", "the Engine is asking the model once", &a
	default:
		c.recordArmOutcome(exp, arm, a, "")
		st.Engine, st.State, st.Detail = &a, a.State, a.Reason
	}
	return st
}

func (c *Core) recordArmOutcome(exp, arm string, a engine.ExperimentArm, detail string) {
	coreCreateOnce(c.expRunPath(exp, arm+".outcome.json"), expOutcomeRec{Schema: experimentRunSchema, Arm: a, Detail: detail, At: c.now().UTC()})
}

// activeExperiment: "<exp>/<arm>" of an arm that is SUBMITTED or RUNNING, if any.
func (c *Core) activeExperiment() string {
	ents, _ := os.ReadDir(filepath.Join(c.cfg.DataDir, experimentRunDir))
	for _, d := range ents {
		if !experimentIDRE.MatchString(d.Name()) {
			continue
		}
		for _, arm := range []string{"A", "B"} {
			if _, err := os.Stat(c.expRunPath(d.Name(), arm+".submitted.json")); err != nil {
				continue
			}
			if s := c.armView(d.Name(), arm); !expTerminal(s.State) {
				return d.Name() + "/" + arm
			}
		}
	}
	return ""
}

// activeExperimentLocal: display only (Core's files, no Engine read).
func (c *Core) activeExperimentLocal() string {
	ents, _ := os.ReadDir(filepath.Join(c.cfg.DataDir, experimentRunDir))
	for _, d := range ents {
		for _, arm := range []string{"A", "B"} {
			if _, err := os.Stat(c.expRunPath(d.Name(), arm+".submitted.json")); err != nil {
				continue
			}
			if _, err := os.Stat(c.expRunPath(d.Name(), arm+".outcome.json")); err != nil {
				return d.Name() + "/" + arm
			}
		}
	}
	return ""
}

// ---------- restore fence, extended to experiment plans (design #107 §3.3) ----------

type experimentFenceRec struct {
	Schema     string `json:"schema"`
	RestoredAt string `json:"restored_at"`
	ExpBytes   *int64 `json:"experiments_bytes"`
	ExpSHA     string `json:"experiments_sha256"`
}

// experimentFence returns the plans that existed before a restore (they can never run: an arm may already
// have been sent in the history that was restored away) and a non-empty reason if every run must be refused.
//   - no fence file: nothing is PRE_RESTORE;
//   - a fence that records the experiment journal (bytes + sha256): plans that start before that boundary
//     are PRE_RESTORE; a journal shorter than the boundary, or whose restored part changed, locks all runs;
//   - a fence from before experiments existed (no experiment fields): the boundary is unknown, so every
//     plan saved at or before the restore time is PRE_RESTORE (all of them if the time can't be read).
func (c *Core) experimentFence(plans []Experiment, offsets []int64, prefix []byte) (map[string]bool, string) {
	pre := map[string]bool{}
	b, err := os.ReadFile(c.file(restoreFenceFile))
	if errors.Is(err, os.ErrNotExist) {
		return pre, ""
	}
	var f experimentFenceRec
	if err != nil || json.Unmarshal(b, &f) != nil || f.Schema != "tass.restore-fence/2" {
		return pre, "the restore fence is unreadable"
	}
	if f.ExpBytes == nil {
		at, terr := time.Parse(time.RFC3339, f.RestoredAt)
		for _, p := range plans {
			if terr != nil || !p.CreatedAt.After(at.Add(time.Second)) {
				pre[p.ExperimentID] = true
			}
		}
		return pre, ""
	}
	if *f.ExpBytes < 0 || len(f.ExpSHA) != 64 {
		return pre, "the restore fence is unreadable"
	}
	if int64(len(prefix)) < *f.ExpBytes {
		return pre, "the experiment journal is shorter than when it was restored"
	}
	sum := sha256.Sum256(prefix[:*f.ExpBytes])
	if hex.EncodeToString(sum[:]) != f.ExpSHA {
		return pre, "the restored part of the experiment journal changed since the restore"
	}
	for i, p := range plans {
		if offsets[i] < *f.ExpBytes {
			pre[p.ExperimentID] = true
		}
	}
	return pre, ""
}

// experimentGuards: PRE_RESTORE plans and the reason (if any) that no arm may run now.
func (c *Core) experimentGuards() (map[string]bool, string) {
	c.expMu.Lock()
	plans, offsets, prefix, _, _, err := c.experimentJournalLoadRaw()
	c.expMu.Unlock()
	if err != nil {
		return map[string]bool{}, "the experiment journal is unsafe"
	}
	pre, why := c.experimentFence(plans, offsets, prefix)
	if why == "" {
		why = c.processingLock()
	}
	return pre, why
}

// ---------- readiness (display and the pre-capture check) ----------

type experimentReadiness struct {
	Identity string `json:"identity"` // UNLOCKED | LOCKED | UNKNOWN
	Grant    string `json:"grant"`    // ENABLED | NOT_ENABLED | UNKNOWN
	Locked   string `json:"locked,omitempty"`
	Busy     string `json:"busy,omitempty"`
}

func (c *Core) experimentReadiness(live bool) experimentReadiness {
	r := experimentReadiness{Identity: "UNKNOWN", Grant: "UNKNOWN"}
	if c.cfg.IdentitySvc != nil {
		st := c.cfg.IdentitySvc.State()
		if st.State == "UNLOCKED" || st.State == "LOCKED" {
			r.Identity = st.State
		}
		if st.Entity != nil && st.State == "UNLOCKED" {
			r.Grant = "NOT_ENABLED"
			for _, g := range st.Entity.Grants {
				if g == ActionExperimentRun {
					r.Grant = "ENABLED"
				}
			}
		}
	}
	if live {
		if n := c.activeRun(""); n != "" {
			r.Busy = "nudge " + n
		} else if x := c.activeExperiment(); x != "" {
			r.Busy = "experiment arm " + x
		}
	} else if n := c.activeRunLocal(); n != "" {
		r.Busy = "nudge " + n
	} else if x := c.activeExperimentLocal(); x != "" {
		r.Busy = "experiment arm " + x
	}
	return r
}

// ---------- run one arm ----------

func experimentConfirmPhrase(arm, id string) string { return "RUN " + arm + " " + id }

// RunExperimentArm is the owner's confirmed "Run A once" / "Run B once". created=true means 202.
func (c *Core) RunExperimentArm(id, arm string, in ArmConfirm, rq ...*CognitionReq) (ExperimentView, bool, error) {
	if !experimentIDRE.MatchString(id) || (arm != "A" && arm != "B") {
		return ExperimentView{}, false, ErrExperimentNotFound
	}
	st := c.nrs()
	st.admitMu.Lock() // the same admission lock as nudges: one decision about the model slot at a time
	defer st.admitMu.Unlock()
	// X1
	scope, err := c.cognitionAccess(rq...)
	if err != nil {
		return ExperimentView{}, false, err
	}
	if scope.engineID == "" {
		return ExperimentView{}, false, ErrExperimentNotFound
	}
	c.expMu.Lock()
	plans, offsets, prefix, _, _, err := c.experimentJournalLoadRaw()
	c.expMu.Unlock()
	if err != nil {
		return ExperimentView{}, false, err
	}
	var plan *Experiment
	for i := range plans {
		if plans[i].ExperimentID == id && plans[i].OwnerEntityID == scope.identityID && plans[i].EngineEntityID == scope.engineID {
			plan = &plans[i]
		}
	}
	if plan == nil {
		return ExperimentView{}, false, ErrExperimentNotFound
	}
	view := func(created bool) (ExperimentView, bool, error) {
		run, err := c.DevelopmentRun(plan.BaselineNudgeID, rq...)
		if err != nil {
			return ExperimentView{}, false, err
		}
		return c.experimentViewLive(*plan, run), created, nil
	}
	if c.cfg.ChatOnly {
		return ExperimentView{}, false, fmt.Errorf("%w: forced chat-only mode (TASS_CHAT_ONLY=on)", ErrCognitionDenied)
	}
	if in.BodyDigest != plan.BodyDigest || in.Confirm != experimentConfirmPhrase(arm, id) {
		return ExperimentView{}, false, fmt.Errorf("%w: the confirmation does not name this arm of the saved plan; nothing was sent", ErrExperimentConflict)
	}
	pre, lock := c.experimentFence(plans, offsets, prefix)
	if lock == "" {
		lock = c.processingLock()
	}
	if lock != "" {
		return ExperimentView{}, false, fmt.Errorf("%w: %s", ErrProcessLocked, lock)
	}
	if pre[id] {
		return ExperimentView{}, false, fmt.Errorf("%w: PRE_RESTORE: this plan existed before a restore and an arm may already have been sent; save a new plan", ErrExperimentConflict)
	}
	// X2: an arm that was ever submitted is only shown.
	if s := c.armView(id, arm); s.State != "NOT RUN" {
		return view(false)
	}
	// X3
	if arm == "B" {
		switch a := c.armView(id, "A"); {
		case a.State == "NOT RUN":
			return ExperimentView{}, false, fmt.Errorf("%w: run A first; B is compared with A", ErrExperimentRefused)
		case !expTerminal(a.State):
			return ExperimentView{}, false, fmt.Errorf("%w (experiment arm %s/A)", ErrProcessBusy, id)
		case a.State != "COMPLETED":
			return ExperimentView{}, false, fmt.Errorf("%w: A is %s, which is final; save a new plan to try again", ErrExperimentRefused, a.State)
		}
	}
	// X4: before anything is captured or sent.
	if r := c.experimentReadiness(false); r.Identity != "UNLOCKED" || r.Grant != "ENABLED" {
		return ExperimentView{}, false, ErrExperimentNotGranted
	}
	// X5
	if other := c.activeRun(""); other != "" {
		return ExperimentView{}, false, fmt.Errorf("%w (nudge %s)", ErrProcessBusy, other)
	}
	if other := c.activeExperiment(); other != "" {
		return ExperimentView{}, false, fmt.Errorf("%w (experiment arm %s)", ErrProcessBusy, other)
	}
	// X6: the one frozen snapshot.
	var snap expSnapshotRec
	sex, sok := expReadOnce(c.expRunPath(id, "snapshot.json"), &snap)
	switch {
	case sex && (!sok || snap.ExperimentID != id || snap.OwnerEntityID != scope.identityID || snap.EngineEntityID != scope.engineID ||
		snap.ThoughtID != plan.ThoughtID || snap.VariantSHA != plan.VariantSHA || len(snap.SnapshotSHA) != 64):
		return ExperimentView{}, false, fmt.Errorf("%w: the snapshot record is unreadable or is not this plan's; nothing was sent", ErrExperimentRefused)
	case !sex && arm == "B":
		return ExperimentView{}, false, fmt.Errorf("%w: there is no snapshot record for this experiment; B never captures one. Nothing was sent", ErrExperimentRefused)
	case !sex:
		attention, err := c.experimentAttention(*plan)
		if err != nil {
			return ExperimentView{}, false, err
		}
		sum := sha256.Sum256([]byte(attention))
		info, _, err := c.engine.CaptureExperimentSnapshot(scope.engineID, plan.ThoughtID, id,
			engine.SnapshotIn{OwnerEntityID: scope.identityID, Attention: attention, AttentionSHA256: hex.EncodeToString(sum[:])})
		if err != nil {
			return ExperimentView{}, false, fmt.Errorf("%w: the snapshot could not be captured (%v); nothing was sent", ErrExperimentRefused, err)
		}
		if !info.Present || !info.Verified || len(info.SHA256) != 64 || info.ThoughtID != plan.ThoughtID {
			return ExperimentView{}, false, fmt.Errorf("%w: the Engine's snapshot is not verified (%s); nothing was sent", ErrExperimentRefused, info.Problem)
		}
		snap = expSnapshotRec{Schema: experimentRunSchema, ExperimentID: id, OwnerEntityID: scope.identityID, EngineEntityID: scope.engineID,
			ThoughtID: plan.ThoughtID, SnapshotSHA: info.SHA256, CapturedAt: info.CapturedAt, Evidence: info.Evidence, Notes: info.Notes,
			ThoughtRev: info.ThoughtRevision, EngineCommit: info.EngineCommit, VariantSHA: plan.VariantSHA, At: c.now().UTC()}
		if created, err := coreCreateOnce(c.expRunPath(id, "snapshot.json"), snap); err != nil || !created {
			return ExperimentView{}, false, fmt.Errorf("%w: the snapshot's digest could not be recorded on this computer; nothing was sent", ErrExperimentWrite)
		}
	}
	// Both arms: the Engine's stored snapshot must verify NOW and be exactly the recorded digest.
	rec, _, err := c.engine.ExperimentRecord(scope.engineID, id)
	if err != nil {
		return ExperimentView{}, false, fmt.Errorf("%w: the Engine's snapshot can't be read (%v); nothing was sent", ErrExperimentRefused, err)
	}
	if !rec.Snapshot.Present || !rec.Snapshot.Verified || rec.Snapshot.SHA256 != snap.SnapshotSHA {
		return ExperimentView{}, false, fmt.Errorf("%w: the Engine's stored snapshot is missing, changed or not the recorded one; nothing was sent", ErrExperimentRefused)
	}
	if arm == "B" && rec.Arm("A").SnapshotSHA != snap.SnapshotSHA {
		return ExperimentView{}, false, fmt.Errorf("%w: A was not run on the recorded snapshot; nothing was sent", ErrExperimentRefused)
	}
	// X7: the only authorization, bound to the exact tuple.
	binding := engine.Binding(engine.ExperimentBinding{Action: ActionExperimentRun, OwnerEntityID: scope.identityID, EngineEntityID: scope.engineID,
		ThoughtID: plan.ThoughtID, ExperimentID: id, Arm: arm, SnapshotSHA: snap.SnapshotSHA, Variable: "instruction", VariantSHA: plan.VariantSHA})
	dec := c.cfg.IdentitySvc.AuthorizeBound(ActionExperimentRun, binding, "thought-processor:experiment.run "+id+" "+arm+" "+plan.ThoughtID)
	if !dec.Allow || dec.DecisionID == "" || dec.EntityID != scope.identityID {
		return ExperimentView{}, false, fmt.Errorf("%w (%s)", ErrExperimentNotGranted, dec.Reason)
	}
	// X8
	sub := expSubmittedRec{Schema: experimentRunSchema, ExperimentID: id, Arm: arm, OwnerEntityID: scope.identityID, EngineEntityID: scope.engineID,
		ThoughtID: plan.ThoughtID, SnapshotSHA: snap.SnapshotSHA, VariantSHA: plan.VariantSHA, Binding: binding, DecisionID: dec.DecisionID, At: c.now().UTC()}
	created, err := coreCreateOnce(c.expRunPath(id, arm+".submitted.json"), sub)
	switch {
	case !created && err == nil:
		return view(false)
	case !created:
		return ExperimentView{}, false, fmt.Errorf("%w: %v", ErrExperimentWrite, err)
	case err != nil:
		return view(false) // exists but not confirmed durable: RECOVERY_REQUIRED, never sent
	}
	st.mu.Lock()
	st.dispatching["exp:"+id+"/"+arm] = true
	ctx := st.ctx
	st.mu.Unlock()
	st.wg.Add(1)
	go c.dispatchArm(ctx, sub, plan.Variant)
	c.event("sync", fmt.Sprintf("Experiment %s arm %s accepted (Identity %s, snapshot %s…); sending it to the Engine once.", id, arm, dec.DecisionID, snap.SnapshotSHA[:12]))
	return view(true)
}

// experimentAttention: the owner's attention text of the baseline nudge, from Core's own journal.
func (c *Core) experimentAttention(plan Experiment) (string, error) {
	c.nudgeMu.Lock()
	all, _, err := c.readNudges()
	c.nudgeMu.Unlock()
	if err != nil {
		return "", ErrCognitionUnavailable
	}
	for _, n := range all {
		if n.NudgeID == plan.BaselineNudgeID && n.OwnerEntityID == plan.OwnerEntityID && n.SourceEngineEntityID == plan.EngineEntityID && n.SourceThoughtID == plan.ThoughtID {
			if a := strings.TrimSpace(n.RequestedAction); a != "" {
				return a, nil
			}
		}
	}
	return "", fmt.Errorf("%w: the original nudge's attention text is not in the journal; nothing was sent", ErrExperimentRefused)
}

// dispatchArm sends the one request. Only the process that created the submitted receipt calls this.
func (c *Core) dispatchArm(ctx context.Context, sub expSubmittedRec, variant string) {
	st := c.nrs()
	defer st.wg.Done()
	key := "exp:" + sub.ExperimentID + "/" + sub.Arm
	defer func() { st.mu.Lock(); delete(st.dispatching, key); st.mu.Unlock() }()
	rec, code, err := c.engine.RunExperimentArmCtx(ctx, sub.EngineEntityID, sub.ThoughtID, sub.ExperimentID, sub.Arm,
		engine.ArmIn{OwnerEntityID: sub.OwnerEntityID, SnapshotSHA: sub.SnapshotSHA, Variant: variant, VariantSHA: sub.VariantSHA, DecisionID: sub.DecisionID})
	if err != nil {
		var he *engine.ErrNudgeHTTP
		if errors.As(err, &he) && he.Code >= 400 && he.Code < 500 {
			// A definite refusal before any claim: final, the model was never asked.
			c.recordArmOutcome(sub.ExperimentID, sub.Arm, engine.ExperimentArm{Arm: sub.Arm, State: "FAILED", ModelHTTP: "none",
				Reason: fmt.Sprintf("refused by the Engine (%d): %s", he.Code, he.Msg)}, "")
			c.event("problem", fmt.Sprintf("Experiment %s arm %s refused by the Engine (%d): nothing was sent to the model.", sub.ExperimentID, sub.Arm, code))
			return
		}
		c.event("problem", "Experiment "+sub.ExperimentID+" arm "+sub.Arm+": delivery not confirmed ("+err.Error()+"); it will NOT be re-sent. Its state is read from the Engine.")
		return
	}
	a := rec.Arm(sub.Arm)
	if expTerminal(a.State) {
		c.recordArmOutcome(sub.ExperimentID, sub.Arm, a, "")
	}
	c.event("tass", fmt.Sprintf("Experiment %s arm %s: %s %s (model %s). Nothing was saved into the Thought.", sub.ExperimentID, sub.Arm, a.State, a.Reason, a.ModelHTTP))
}
