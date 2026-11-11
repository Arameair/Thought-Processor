package tass

// Stage 2 (Thought-Processor #100, design rev 4, owner-approved D1–D16): process ONE confirmed nudge
// to TASS himself, or stop it.
//
//   POST /api/nudges/{id}/process   C1 checks → C2 ONE bound Identity decision → C3 existing run? →
//                                   C4 no other active run → C5 submitted.json (O_EXCL, durable) →
//                                   202, then this process (only) sends it to the Engine once.
//   POST /api/nudges/{id}/stop      a bound cognition.nudge.stop decision → Engine void.
//
// Core's receipts are immutable per-nudge files in <data>/nudge-runs/<id>/ (submitted, observed,
// void, outcome). Core never re-sends: after a restart, a SUBMITTED run is only READ from the Engine.
// admitMu serializes C1–C5 for all nudge IDs; one Core per data folder is enforced by .core.lock.

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
	"sync"
	"time"

	"github.com/Arameair/Thought-Processor/core/engine"
)

const (
	nudgeRunSchemaCore = "thought-processor.nudge-run/2"
	nudgeRunDirCore    = "nudge-runs"
	ActionNudgeProcess = "cognition.nudge.process"
	ActionNudgeStop    = "cognition.nudge.stop"
)

var (
	ErrProcessLocked     = errors.New("nudge processing is locked")
	ErrProcessNotGranted = errors.New("Identity has not granted nudge processing: enable \"cognition.nudge-process\" on Identity's page (127.0.0.1:8796) with your passphrase")
	ErrProcessBusy       = errors.New("another nudge is being processed; wait for it to finish or stop it")
	ErrProcessRefused    = errors.New("this nudge can't be processed")
)

// RunView is what the page shows about a nudge's processing.
type RunView struct {
	State       string           `json:"state"` // SUBMITTED | RUNNING | COMPLETED | FAILED | UNCERTAIN | NOT_DELIVERED | RECOVERY_REQUIRED
	Detail      string           `json:"detail,omitempty"`
	Verified    bool             `json:"verified"`
	ModelHTTP   string           `json:"model_http,omitempty"`
	RequestID   string           `json:"request_id,omitempty"`
	StepAt      *time.Time       `json:"step_at,omitempty"`
	SubmittedAt time.Time        `json:"submitted_at"`
	Decision    string           `json:"process_decision_id,omitempty"`
	StopDec     string           `json:"stop_decision_id,omitempty"`
	Engine      *engine.NudgeRun `json:"engine,omitempty"`
	CanStop     bool             `json:"can_stop"`
	CanCopy     bool             `json:"can_copy"`
}

type submittedRec struct {
	Schema         string    `json:"schema"`
	NudgeID        string    `json:"nudge_id"`
	OwnerEntityID  string    `json:"owner_entity_id"`
	EngineEntityID string    `json:"engine_entity_id"`
	ThoughtID      string    `json:"thought_id"`
	AttentionSHA   string    `json:"attention_sha256"`
	DraftExpiresAt string    `json:"draft_expires_at"`
	Binding        string    `json:"binding"`
	DecisionID     string    `json:"process_decision_id"`
	At             time.Time `json:"at"`
}
type coreOutcome struct {
	Schema   string          `json:"schema"`
	Run      engine.NudgeRun `json:"engine_run"`
	Verified bool            `json:"verified"`
	Detail   string          `json:"detail,omitempty"`
	At       time.Time       `json:"at"`
}
type coreVoid struct {
	Schema  string    `json:"schema"`
	StopDec string    `json:"stop_decision_id"`
	At      time.Time `json:"at"`
}
type coreObserved struct {
	Schema string    `json:"schema"`
	State  string    `json:"engine_state"`
	At     time.Time `json:"at"`
}

// nudgeRunState: Core-side, in memory.
type nudgeRunState struct {
	admitMu     sync.Mutex      // C1–C5 for all nudge IDs (R3-09)
	mu          sync.Mutex      // fields below
	dispatching map[string]bool // this process is sending it now
	wg          sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
	divergence  string // the Engine forgot a claim Core observed: processing locked
}

var runStates sync.Map // *Core → *nudgeRunState (keeps core.go's struct untouched)

func (c *Core) nrs() *nudgeRunState {
	v, _ := runStates.LoadOrStore(c, &nudgeRunState{dispatching: map[string]bool{}})
	st := v.(*nudgeRunState)
	st.mu.Lock()
	if st.ctx == nil {
		st.ctx, st.cancel = context.WithCancel(context.Background())
	}
	st.mu.Unlock()
	return st
}

// StopNudgeDispatch cancels in-flight sends (the Engine keeps going; Core reconciles by reading).
func (c *Core) StopNudgeDispatch() {
	st := c.nrs()
	st.cancel()
	done := make(chan struct{})
	go func() { st.wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
	}
}

func (c *Core) runPath(id, name string) string {
	return filepath.Join(c.cfg.DataDir, nudgeRunDirCore, id, name)
}

// coreCreateOnce: O_EXCL create + file fsync + close + folder fsync. created && err != nil means the
// file EXISTS but may be torn (never treat the nudge as unsubmitted).
func coreCreateOnce(path string, v any) (bool, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return false, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return false, err
	}
	f, err := nudgeOpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if errors.Is(err, os.ErrExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if _, err := f.Write(append(b, '\n')); err != nil {
		f.Close()
		return true, err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return true, err
	}
	if err := f.Close(); err != nil {
		return true, err
	}
	d, err := os.Open(filepath.Dir(path))
	if err != nil {
		return true, err
	}
	defer d.Close()
	return true, d.Sync()
}

func coreReadOnce(path string, v any) (exists, ok bool, raw []byte) {
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, false, nil
	}
	if err != nil || len(strings.TrimSpace(string(b))) == 0 || json.Unmarshal(b, v) != nil {
		return true, false, b
	}
	var s struct {
		Schema string `json:"schema"`
	}
	json.Unmarshal(b, &s)
	return true, s.Schema == nudgeRunSchemaCore, b
}

func terminal(state string) bool {
	switch state {
	case "COMPLETED", "FAILED", "UNCERTAIN", "NOT_DELIVERED", "RECOVERY_REQUIRED":
		return true
	}
	return false
}

// verifyRun: COMPLETED only with exactly one matching Note and one matching developed step.
func (c *Core) verifyRun(eid string, r engine.NudgeRun) (bool, string) {
	if r.State != "COMPLETED" || r.RequestID == "" || r.Applied {
		return false, "the Engine did not report a clean COMPLETED"
	}
	d, err := c.engine.CognitionThought(eid, r.ThoughtID)
	if err != nil {
		return false, "the Thought could not be read to verify: " + err.Error()
	}
	notes, steps := 0, 0
	for _, n := range d.Thought.Notes {
		if n.NudgeID == r.NudgeID {
			if n.RequestID == r.RequestID {
				notes++
			} else {
				notes += 100
			}
		}
	}
	for _, h := range d.Thought.History {
		if h.NudgeID == r.NudgeID {
			if h.Kind == "developed" && h.RequestID == r.RequestID {
				steps++
			} else {
				steps += 100
			}
		}
	}
	if notes != 1 || steps != 1 {
		return false, fmt.Sprintf("the Thought has %d matching note(s) and %d matching step(s); exactly 1 and 1 are required", notes, steps)
	}
	return true, "exactly one note and one developed step with this nudge and request ID"
}

// runView derives a nudge's run from Core's files and (only if still open) ONE Engine read.
// It never sends anything. It may record what it observed (observed/outcome receipts).
func (c *Core) runView(id string) *RunView {
	var sub submittedRec
	sex, sok, _ := coreReadOnce(c.runPath(id, "submitted.json"), &sub)
	if !sex {
		return nil
	}
	v := &RunView{SubmittedAt: sub.At, Decision: sub.DecisionID}
	if !sok || sub.NudgeID != id {
		v.State, v.Detail, v.CanCopy = "RECOVERY_REQUIRED", "the submission record is unreadable; this nudge will never be sent", true
		return v
	}
	var out coreOutcome
	if oex, ook, _ := coreReadOnce(c.runPath(id, "outcome.json"), &out); oex {
		if !ook {
			v.State, v.Detail, v.CanCopy = "RECOVERY_REQUIRED", "the outcome record is unreadable", true
			return v
		}
		run := out.Run
		v.Engine, v.ModelHTTP, v.RequestID, v.StepAt, v.Verified = &run, run.ModelHTTP, run.RequestID, run.StepAt, out.Verified
		v.State, v.Detail = run.State, nonEmpty(run.Reason, out.Detail)
		if run.State == "COMPLETED" && !out.Verified {
			v.State, v.Detail = "RECOVERY_REQUIRED", out.Detail
		}
		if run.State == "VOID" {
			v.State, v.Detail = "NOT_DELIVERED", "stopped before the Engine claimed it; the model was never asked"
		}
		v.CanCopy = true
		return v
	}
	var vd coreVoid
	if vex, vok, _ := coreReadOnce(c.runPath(id, "void.json"), &vd); vex {
		v.StopDec = vd.StopDec
		if vok {
			v.State, v.Detail, v.ModelHTTP, v.CanCopy = "NOT_DELIVERED", "stopped before the Engine claimed it; the model was never asked", "none", true
		} else {
			v.State, v.Detail, v.CanCopy = "RECOVERY_REQUIRED", "the stop record is unreadable", true
		}
		return v
	}
	st := c.nrs()
	st.mu.Lock()
	sending := st.dispatching[id]
	st.mu.Unlock()
	// Whether Core had ALREADY recorded the Engine's claim is decided before the Engine is asked.
	// The delivery (dispatchNudge) runs at the same time as this read, so "not claimed" can be an
	// answer from before the claim that arrives after observed.json was written. Only a read that
	// STARTED after the recorded claim can show that the Engine lost the run.
	var ob coreObserved
	claimSeen, _, _ := coreReadOnce(c.runPath(id, "observed.json"), &ob)
	run, code, err := c.engine.NudgeRun(sub.EngineEntityID, id)
	switch {
	case err != nil:
		v.State, v.Detail, v.CanStop = "SUBMITTED", "accepted locally; the Engine can't be read right now ("+err.Error()+")", true
		return v
	case code == 404 || run.State == "NOT_CLAIMED":
		if !claimSeen {
			if oex, _, _ := coreReadOnce(c.runPath(id, "observed.json"), &ob); oex {
				// The claim was recorded while this read was out: the answer is older than the
				// record and proves nothing. Derive again; that pass starts with the claim recorded,
				// so its own read is decisive (at most one extra pass).
				return c.runView(id)
			}
		}
		if claimSeen {
			st.mu.Lock()
			st.divergence = "the Engine no longer has nudge " + id + " that Core saw it claim (data restored partially?)"
			st.mu.Unlock()
			v.State, v.Detail = "RECOVERY_REQUIRED", "the Engine no longer has this run; processing is locked"
			return v
		}
		v.State, v.CanStop = "SUBMITTED", true
		v.Detail = "accepted locally; not claimed by the Engine yet"
		if sending {
			v.Detail = "accepted locally; being delivered to the Engine"
		}
		return v
	}
	coreCreateOnce(c.runPath(id, "observed.json"), coreObserved{Schema: nudgeRunSchemaCore, State: run.State, At: c.now().UTC()})
	v.Engine, v.ModelHTTP, v.RequestID, v.StepAt = &run, run.ModelHTTP, run.RequestID, run.StepAt
	switch run.State {
	case "RUNNING":
		v.State, v.Detail, v.CanStop = "RUNNING", "the Engine is developing the Thought", true
		return v
	case "VOID":
		v.State, v.Detail, v.ModelHTTP, v.CanCopy = "NOT_DELIVERED", "stopped before the Engine claimed it; the model was never asked", "none", true
		return v
	}
	c.recordOutcome(id, sub.EngineEntityID, run)
	return c.runView(id)
}

// recordOutcome writes Core's outcome receipt for a terminal Engine run (verifying COMPLETED).
func (c *Core) recordOutcome(id, eid string, run engine.NudgeRun) {
	o := coreOutcome{Schema: nudgeRunSchemaCore, Run: run, At: c.now().UTC()}
	if run.State == "COMPLETED" {
		o.Verified, o.Detail = c.verifyRun(eid, run)
	}
	coreCreateOnce(c.runPath(id, "outcome.json"), o)
}

// activeRun: the ID of a run that is SUBMITTED or RUNNING, if any.
func (c *Core) activeRun(except string) string {
	ents, _ := os.ReadDir(filepath.Join(c.cfg.DataDir, nudgeRunDirCore))
	for _, d := range ents {
		if d.Name() == except {
			continue
		}
		if v := c.runView(d.Name()); v != nil && !terminal(v.State) {
			return d.Name()
		}
	}
	return ""
}

// NudgeActive is true while any nudge is SUBMITTED or RUNNING (Cognition link/unlink refuse it, A-11).
func (c *Core) NudgeActive() bool { return c.activeRun("") != "" }

// processScope: C1 for one nudge. Returns the draft, the scope and whether it is pre-restore.
func (c *Core) nudgeForAction(id string, rq ...*CognitionReq) (Nudge, cognitionScope, error) {
	if !nudgeIDOK(id) {
		return Nudge{}, cognitionScope{}, fmt.Errorf("%w: invalid nudge ID", ErrNudgeInvalid)
	}
	scope, err := c.cognitionAccess(rq...)
	if err != nil {
		return Nudge{}, scope, err
	}
	if scope.engineID == "" {
		return Nudge{}, scope, fmt.Errorf("%w: link this Identity to its Engine history first", ErrNudgeInvalid)
	}
	lines, _, err := c.readNudgeLines()
	if err != nil {
		return Nudge{}, scope, ErrCognitionUnavailable
	}
	for _, l := range lines {
		if l.Valid && l.N.NudgeID == id && l.N.OwnerEntityID == scope.identityID && l.N.SourceEngineEntityID == scope.engineID {
			return l.N, scope, nil
		}
	}
	return Nudge{}, scope, fmt.Errorf("%w: no such nudge for this Identity and Engine link", ErrNudgeInvalid)
}

func nudgeIDOK(id string) bool {
	if len(id) != 26 || !strings.HasPrefix(id, "nudge_") {
		return false
	}
	_, err := hex.DecodeString(id[6:])
	return err == nil
}

// processingLock: any reason processing must be refused for all nudges (read-only RECOVERY_REQUIRED).
func (c *Core) processingLock() string {
	lines, raw, err := c.readNudgeLines()
	if err != nil {
		return "the draft journal can't be read"
	}
	if _, why := c.nudgeFence(lines, raw); why != "" {
		return why
	}
	st := c.nrs()
	st.mu.Lock()
	defer st.mu.Unlock()
	return st.divergence
}

func (c *Core) isPreRestore(id string) bool {
	lines, raw, err := c.readNudgeLines()
	if err != nil {
		return true
	}
	boundary, _ := c.nudgeFence(lines, raw)
	return preRestore(lines, boundary)[id]
}

// ProcessNudge is the owner's "Process once". created=true means 202 (accepted locally).
func (c *Core) ProcessNudge(id, bodyDigest string, rq ...*CognitionReq) (*RunView, bool, error) {
	st := c.nrs()
	st.admitMu.Lock()
	defer st.admitMu.Unlock()
	// C1
	n, scope, err := c.nudgeForAction(id, rq...)
	if err != nil {
		return nil, false, err
	}
	if c.cfg.ChatOnly {
		return nil, false, fmt.Errorf("%w: forced chat-only mode (TASS_CHAT_ONLY=on)", ErrCognitionDenied)
	}
	if why := c.processingLock(); why != "" {
		return nil, false, fmt.Errorf("%w: %s", ErrProcessLocked, why)
	}
	if n.BodyDigest != bodyDigest {
		return nil, false, fmt.Errorf("%w: the nudge on the page is not the saved one (digest mismatch)", ErrNudgeConflict)
	}
	if n.RecipientID != "self" {
		return nil, false, fmt.Errorf("%w: only nudges to TASS himself can be processed (card positions are draft-only)", ErrProcessRefused)
	}
	if c.now().After(n.ExpiresAt) {
		return nil, false, fmt.Errorf("%w: the draft expired", ErrProcessRefused)
	}
	if c.isPreRestore(id) {
		return nil, false, fmt.Errorf("%w: PRE_RESTORE: this draft existed before a restore and may already have been processed; use Copy as new", ErrNudgeConflict)
	}
	// C2: the only authorization, bound to the exact tuple, before anything else is returned.
	sum := sha256.Sum256([]byte(n.RequestedAction))
	attSHA := hex.EncodeToString(sum[:])
	expires := n.ExpiresAt.UTC().Format(time.RFC3339Nano)
	binding := engine.Binding(engine.ProcessBinding{Action: ActionNudgeProcess, OwnerEntityID: scope.identityID, EngineEntityID: scope.engineID,
		ThoughtID: n.SourceThoughtID, NudgeID: id, AttentionSHA: attSHA, Recipient: "self", DraftExpiresAt: expires})
	dec := c.cfg.IdentitySvc.AuthorizeBound(ActionNudgeProcess, binding, "thought-processor:nudge.process "+id+" "+n.SourceThoughtID)
	if !dec.Allow || dec.DecisionID == "" || dec.EntityID != scope.identityID {
		return nil, false, fmt.Errorf("%w (%s)", ErrProcessNotGranted, dec.Reason)
	}
	// C3: an existing run is returned (the decision just expires unused).
	if v := c.runView(id); v != nil {
		return v, false, nil
	}
	// C4: one active run at a time.
	if other := c.activeRun(id); other != "" {
		return nil, false, fmt.Errorf("%w (%s)", ErrProcessBusy, other)
	}
	// C5: the durable submission.
	sub := submittedRec{Schema: nudgeRunSchemaCore, NudgeID: id, OwnerEntityID: scope.identityID, EngineEntityID: scope.engineID,
		ThoughtID: n.SourceThoughtID, AttentionSHA: attSHA, DraftExpiresAt: expires, Binding: binding, DecisionID: dec.DecisionID, At: c.now().UTC()}
	created, err := coreCreateOnce(c.runPath(id, "submitted.json"), sub)
	switch {
	case !created && err == nil:
		return c.runView(id), false, nil
	case !created:
		return nil, false, fmt.Errorf("%w: %v", ErrNudgeWrite, err)
	case err != nil:
		return c.runView(id), false, nil // exists but not confirmed durable: RECOVERY_REQUIRED, never sent
	}
	st.mu.Lock()
	st.dispatching[id] = true
	ctx := st.ctx
	st.mu.Unlock()
	st.wg.Add(1)
	go c.dispatchNudge(ctx, sub, n.RequestedAction)
	c.event("sync", fmt.Sprintf("Nudge %s accepted for processing (Identity %s); delivering it to the Engine once.", id, dec.DecisionID))
	v := c.runView(id)
	return v, true, nil
}

// dispatchNudge sends the one request. Only the process that created submitted.json calls this.
func (c *Core) dispatchNudge(ctx context.Context, sub submittedRec, attention string) {
	st := c.nrs()
	defer st.wg.Done()
	defer func() { st.mu.Lock(); delete(st.dispatching, sub.NudgeID); st.mu.Unlock() }()
	run, code, err := c.engine.ProcessNudgeCtx(ctx, sub.EngineEntityID, sub.ThoughtID, engine.NudgeIn{NudgeID: sub.NudgeID, OwnerEntityID: sub.OwnerEntityID,
		Attention: attention, AttentionSHA256: sub.AttentionSHA, DraftExpiresAt: sub.DraftExpiresAt, DecisionID: sub.DecisionID})
	if err != nil {
		var he *engine.ErrNudgeHTTP
		if errors.As(err, &he) && he.Code >= 400 && he.Code < 500 {
			// A definite refusal before any claim (no run folder at the Engine): record it as FAILED.
			c.recordOutcome(sub.NudgeID, sub.EngineEntityID, engine.NudgeRun{NudgeID: sub.NudgeID, EntityID: sub.EngineEntityID, ThoughtID: sub.ThoughtID,
				State: "FAILED", Reason: fmt.Sprintf("refused by the Engine (%d): %s", he.Code, he.Msg), ModelHTTP: "none"})
			c.event("problem", fmt.Sprintf("Nudge %s refused by the Engine (%d): nothing was sent to the model.", sub.NudgeID, code))
			return
		}
		c.event("problem", "Nudge "+sub.NudgeID+": delivery not confirmed ("+err.Error()+"); it will NOT be re-sent. Its state is read from the Engine.")
		return
	}
	coreCreateOnce(c.runPath(sub.NudgeID, "observed.json"), coreObserved{Schema: nudgeRunSchemaCore, State: run.State, At: c.now().UTC()})
	if terminal(run.State) || run.State == "VOID" {
		c.recordOutcome(sub.NudgeID, sub.EngineEntityID, run)
	}
	c.event("tass", fmt.Sprintf("Nudge %s: %s %s (model %s).", sub.NudgeID, run.State, run.Reason, run.ModelHTTP))
}

// StopNudge is the owner's "Stop this nudge": a bound cognition.nudge.stop decision → Engine void.
func (c *Core) StopNudge(id string, rq ...*CognitionReq) (*RunView, error) {
	st := c.nrs()
	st.admitMu.Lock()
	defer st.admitMu.Unlock()
	n, scope, err := c.nudgeForAction(id, rq...)
	if err != nil {
		return nil, err
	}
	var sub submittedRec
	sex, sok, raw := coreReadOnce(c.runPath(id, "submitted.json"), &sub)
	if !sex {
		return nil, fmt.Errorf("%w: this nudge was never submitted; there is nothing to stop", ErrNudgeConflict)
	}
	if !sok {
		return c.runView(id), nil
	}
	if v := c.runView(id); v != nil && terminal(v.State) {
		return v, nil
	}
	h := sha256.Sum256(raw)
	subSHA := hex.EncodeToString(h[:])
	binding := engine.Binding(engine.StopBinding{Action: ActionNudgeStop, OwnerEntityID: scope.identityID, EngineEntityID: sub.EngineEntityID,
		ThoughtID: n.SourceThoughtID, NudgeID: id, SubmittedSHA: subSHA})
	dec := c.cfg.IdentitySvc.AuthorizeBound(ActionNudgeStop, binding, "thought-processor:nudge.stop "+id)
	if !dec.Allow || dec.DecisionID == "" || dec.EntityID != scope.identityID {
		return nil, fmt.Errorf("%w: Identity refused the stop (%s)", ErrCognitionDenied, dec.Reason)
	}
	run, _, err := c.engine.VoidNudge(sub.EngineEntityID, id, engine.VoidIn{OwnerEntityID: scope.identityID, ThoughtID: n.SourceThoughtID,
		SubmittedSHA256: subSHA, DecisionID: dec.DecisionID})
	if err != nil {
		return c.runView(id), fmt.Errorf("the stop could not be confirmed: %v", err)
	}
	if run.State == "VOID" {
		coreCreateOnce(c.runPath(id, "void.json"), coreVoid{Schema: nudgeRunSchemaCore, StopDec: dec.DecisionID, At: c.now().UTC()})
		c.event("sync", "Nudge "+id+" stopped: the Engine will never run it.")
	}
	return c.runView(id), nil
}

func nonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
