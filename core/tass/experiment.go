package tass

// Stage 3B (#120; design #107 §3.3, §6): one instruction change, planned as an isolated A/B experiment.
//
// This slice saves the PLAN only. What it does:
//   - starts from a run the owner already observed (a processed nudge of this owner and linked Engine entity,
//     read through DevelopmentRun: GET-only, no model call);
//   - validates the one variable: the instruction sentence (≤ 300 characters, one line, no NOTE:/STATE:
//     line, different from the control); the output contract is fixed and not editable (C11);
//   - records the snapshot choice honestly: AS_OF_NOW. AS_OF_BASELINE needs an Engine snapshot check that
//     does not exist yet, so it is refused rather than assumed;
//   - appends one immutable plan to a local, owner-scoped journal (experiments.jsonl, 0600, fsync), never
//     synced to GitHub.
//
// What it deliberately does NOT do: run an arm. Running A or B needs owner decisions D4–D7 (#107 §8.2),
// a new Identity bound action (cognition.experiment.run) and grant (cognition.experiment-run), and an
// Engine route under experiments/. None exist, so every arm is NOT RUN and not runnable, and this file has
// no path that calls the Engine with anything but GET. It never writes to the Thought, its Notes, its
// history, the advisory Design graph, settings or code.

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	experimentSchema     = "thought-processor.experiment-plan/1"
	experimentFile       = "experiments.jsonl"
	experimentVariantMax = 300
	experimentJournalMax = 4 << 20
	experimentListMax    = 50

	// The instruction sentence and the fixed output contract of the Engine's thinkPrompt at the commits in
	// nudgeInstructionEngineCommits (prompts.go). TestExperimentControlMatchesEngineSource checks both
	// against the Engine source when ENGINE_SRC is set.
	experimentControl  = "Add ONE new idea, possible explanation, or thing to watch for next. Do not repeat the evidence or earlier notes."
	experimentContract = "Reply with exactly two lines and nothing else:\nNOTE: <at most 40 words>\nSTATE: <keep, sleep or resolve>"

	experimentNotAuthorized = "NOT AUTHORIZED: running an arm needs your decisions D4–D7, Identity's cognition.experiment.run action, " +
		"the cognition.experiment-run grant and the Engine experiment route. None exist yet, so nothing can run."
)

var (
	ErrExperimentInvalid  = errors.New("experiment refused")
	ErrExperimentConflict = errors.New("experiment idempotency conflict")
	ErrExperimentWrite    = errors.New("the experiment plan could not be saved on this computer; nothing was saved")
	ErrExperimentNotFound = errors.New("no experiment with this ID for this owner")

	experimentIDRE = regexp.MustCompile(`^exp_[0-9a-f]{20}$`)
	// A line the Engine's parser (reNote/reState) would read as part of the output contract.
	experimentContractRE = regexp.MustCompile(`(?i)\b(NOTE|STATE)\s*\**\s*[:=]`)
)

// ExperimentDraft is what the owner submits.
type ExperimentDraft struct {
	BaselineNudgeID string `json:"baseline_nudge_id"`
	Variant         string `json:"variant"`
	Snapshot        string `json:"snapshot"` // AS_OF_NOW (AS_OF_BASELINE is refused in this slice)
	IdempotencyKey  string `json:"idempotency_key"`
}

// Experiment is one immutable plan record.
type Experiment struct {
	Schema            string    `json:"schema"`
	ExperimentID      string    `json:"experiment_id"`
	OwnerEntityID     string    `json:"owner_entity_id"`
	EngineEntityID    string    `json:"engine_entity_id"`
	ThoughtID         string    `json:"thought_id"`
	BaselineNudgeID   string    `json:"baseline_nudge_id"`
	BaselineRequestID string    `json:"baseline_request_id"`
	BaselineEngine    string    `json:"baseline_engine_revision"` // RECONCILED / MISMATCH / NOT REPORTED / NOT INSPECTED
	Snapshot          string    `json:"snapshot"`
	Variable          string    `json:"variable"` // always "instruction"
	Control           string    `json:"control"`
	ControlStatus     string    `json:"control_status"` // CONFIRMED (reconciled Engine) or NOT CONFIRMED
	Variant           string    `json:"variant"`
	VariantSHA        string    `json:"variant_sha256"`
	Contract          string    `json:"contract"`
	Settings          string    `json:"settings"`
	CreatedAt         time.Time `json:"created_at"`
	IdempotencyKey    string    `json:"idempotency_key"`
	BodyDigest        string    `json:"body_digest"`
}

// ExperimentArm is the state of one arm. In this slice every arm is NOT RUN and not runnable.
type ExperimentArm struct {
	Arm      string `json:"arm"` // A or B
	Label    string `json:"label"`
	State    string `json:"state"`
	Runnable bool   `json:"runnable"`
	Why      string `json:"why"`
	Requests int    `json:"model_requests"`
}

// ExperimentView is a plan plus what the page needs to present it honestly.
type ExperimentView struct {
	Experiment
	Replayed      bool            `json:"replayed,omitempty"`
	SnapshotLabel string          `json:"snapshot_label"`
	Unchanged     []string        `json:"unchanged"`
	HeldConstant  []string        `json:"held_constant"`
	Cost          string          `json:"cost"`
	ModelRequests int             `json:"model_requests"` // made for this experiment so far
	Arms          []ExperimentArm `json:"arms"`
	Original      ExperimentNote  `json:"original"`
	Note          string          `json:"note"`
}

// ExperimentNote is the original run's outcome, for reference beside A and B.
type ExperimentNote struct {
	Text      string `json:"text,omitempty"`
	RequestID string `json:"request_id"`
	State     string `json:"state"`
	Saved     string `json:"saved_into_thought"`
}

// ExperimentList is what GET /api/development/experiments returns.
type ExperimentList struct {
	Experiments  []ExperimentView `json:"experiments"`
	Control      string           `json:"control"`
	Contract     string           `json:"contract"`
	VariantMax   int              `json:"variant_max"`
	Snapshots    []SnapshotOption `json:"snapshots"`
	Unchanged    []string         `json:"unchanged"`
	HeldConstant []string         `json:"held_constant"`
	Cost         string           `json:"cost"`
	Note         string           `json:"note"`
}

// SnapshotOption is one snapshot the form offers, with whether it can be chosen and why.
type SnapshotOption struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	Available bool   `json:"available"`
	Why       string `json:"why"`
}

var (
	experimentUnchanged    = []string{"the Thought", "its Notes", "its history and state", "the advisory Design graph", "Engine settings and code"}
	experimentHeldConstant = []string{"the Thought's question, evidence and newest Notes (one snapshot for A and B)", "your attention text from the original nudge",
		"the reply format (NOTE / STATE lines; not editable)", "the model settings: " + nudgeModelSettings}
)

var experimentSnapshots = []SnapshotOption{
	{"AS_OF_NOW", "As the Thought is now", true,
		"A and B both use the Thought as it is when A runs. Not the original run's conditions: the original Note is shown for reference only."},
	{"AS_OF_BASELINE", "As it was at the original run", false,
		"NOT AVAILABLE: rebuilding the original inputs and checking them against the logged prompt (VERIFIED / VERIFIED-PREFIX / NOT REPRODUCIBLE) needs an Engine snapshot check that does not exist yet."},
}

func (d *ExperimentDraft) normalize() error {
	d.Variant = strings.TrimSpace(d.Variant)
	if !nudgeKeyRE.MatchString(d.IdempotencyKey) {
		return fmt.Errorf("%w: idempotency_key must be 8-128 letters, digits or ._:-", ErrExperimentInvalid)
	}
	if !nudgeIDOK(d.BaselineNudgeID) {
		return fmt.Errorf("%w: baseline_nudge_id must be a processed nudge ID", ErrExperimentInvalid)
	}
	switch d.Snapshot {
	case "", "AS_OF_NOW":
		d.Snapshot = "AS_OF_NOW"
	case "AS_OF_BASELINE":
		return fmt.Errorf("%w: %s", ErrExperimentInvalid, experimentSnapshots[1].Why)
	default:
		return fmt.Errorf("%w: snapshot must be AS_OF_NOW", ErrExperimentInvalid)
	}
	v := d.Variant
	switch {
	case v == "":
		return fmt.Errorf("%w: write the changed instruction sentence", ErrExperimentInvalid)
	case !utf8.ValidString(v):
		return fmt.Errorf("%w: the instruction is not valid text", ErrExperimentInvalid)
	case utf8.RuneCountInString(v) > experimentVariantMax:
		return fmt.Errorf("%w: the instruction is longer than %d characters", ErrExperimentInvalid, experimentVariantMax)
	case strings.ContainsAny(v, "\r\n"):
		return fmt.Errorf("%w: the instruction must be one line; the reply format lines are fixed", ErrExperimentInvalid)
	case experimentContractRE.MatchString(v):
		return fmt.Errorf("%w: the instruction may not contain NOTE: or STATE:; the reply format is fixed so the Engine's parser stays valid", ErrExperimentInvalid)
	case strings.Join(strings.Fields(v), " ") == experimentControl:
		return fmt.Errorf("%w: B must differ from A: change the instruction sentence", ErrExperimentInvalid)
	}
	for _, r := range v {
		if unicode.IsControl(r) {
			return fmt.Errorf("%w: the instruction contains a control character", ErrExperimentInvalid)
		}
	}
	return nil
}

func (d ExperimentDraft) digest() string {
	h := sha256.Sum256([]byte(d.BaselineNudgeID + "\x00" + d.Snapshot + "\x00" + d.Variant))
	return hex.EncodeToString(h[:])
}

// SaveExperimentDraft validates, checks the baseline run, then appends one plan record. It makes zero
// model calls and sends the Engine only GET requests (through DevelopmentRun).
func (c *Core) SaveExperimentDraft(d ExperimentDraft, rq ...*CognitionReq) (ExperimentView, error) {
	if err := d.normalize(); err != nil {
		return ExperimentView{}, err
	}
	// The owner-scoped, read-only run view decides that the baseline exists and is this owner's.
	run, err := c.DevelopmentRun(d.BaselineNudgeID, rq...)
	if err != nil {
		return ExperimentView{}, err
	}
	scope, err := c.cognitionAccess(rq...)
	if err != nil {
		return ExperimentView{}, err
	}
	if run.RequestID == "" {
		return ExperimentView{}, fmt.Errorf("%w: the baseline run has no recorded request, so there is nothing to compare against", ErrExperimentInvalid)
	}
	digest := d.digest()

	c.expMu.Lock()
	defer c.expMu.Unlock()
	all, _, err := c.readExperiments()
	if err != nil {
		return ExperimentView{}, ErrCognitionUnavailable
	}
	for _, e := range all {
		if e.OwnerEntityID == scope.identityID && e.IdempotencyKey == d.IdempotencyKey {
			if e.EngineEntityID != scope.engineID {
				return ExperimentView{}, fmt.Errorf("%w: that key belongs to a different Engine link; use a new form", ErrExperimentConflict)
			}
			if e.BodyDigest != digest {
				return ExperimentView{}, fmt.Errorf("%w: that key was already used for a different experiment", ErrExperimentConflict)
			}
			v := c.experimentView(e, run)
			v.Replayed = true
			return v, nil
		}
	}
	rnd := make([]byte, 10)
	if _, err := rand.Read(rnd); err != nil {
		return ExperimentView{}, err
	}
	vs := sha256.Sum256([]byte(d.Variant))
	ctl := "NOT CONFIRMED"
	if run.EngineRevision == "RECONCILED" {
		ctl = "CONFIRMED"
	}
	e := Experiment{
		Schema: experimentSchema, ExperimentID: "exp_" + hex.EncodeToString(rnd),
		OwnerEntityID: scope.identityID, EngineEntityID: scope.engineID, ThoughtID: run.ThoughtID,
		BaselineNudgeID: run.NudgeID, BaselineRequestID: run.RequestID, BaselineEngine: run.EngineRevision,
		Snapshot: d.Snapshot, Variable: "instruction", Control: experimentControl, ControlStatus: ctl,
		Variant: d.Variant, VariantSHA: hex.EncodeToString(vs[:]), Contract: experimentContract, Settings: nudgeModelSettings,
		CreatedAt: c.now().UTC(), IdempotencyKey: d.IdempotencyKey, BodyDigest: digest,
	}
	if err := c.appendExperiment(e); err != nil {
		if errors.Is(err, ErrExperimentInvalid) {
			return ExperimentView{}, err
		}
		return ExperimentView{}, fmt.Errorf("%w: %v", ErrExperimentWrite, err)
	}
	c.event("sync", fmt.Sprintf("Saved experiment plan %s (instruction change; not run: running needs separate owner decisions).", e.ExperimentID))
	return c.experimentView(e, run), nil
}

// Experiments lists this owner's plans for one baseline run (newest first), with the form's options.
func (c *Core) Experiments(baselineNudgeID string, rq ...*CognitionReq) (ExperimentList, error) {
	run, err := c.DevelopmentRun(baselineNudgeID, rq...)
	if err != nil {
		return ExperimentList{}, err
	}
	scope, err := c.cognitionAccess(rq...)
	if err != nil {
		return ExperimentList{}, err
	}
	c.expMu.Lock()
	all, _, err := c.readExperiments()
	c.expMu.Unlock()
	if err != nil {
		return ExperimentList{}, ErrCognitionUnavailable
	}
	out := ExperimentList{Experiments: []ExperimentView{}, Control: experimentControl, Contract: experimentContract,
		VariantMax: experimentVariantMax, Snapshots: experimentSnapshots,
		Note: "Experiment plans are local to this computer and never synced to GitHub. Saving a plan makes no model call."}
	out.Unchanged, out.HeldConstant = experimentUnchanged, experimentHeldConstant
	out.Cost = "Saving this plan: 0 model requests. Running it later: 2 (A, then B), one at a time, each with its own confirmation."
	for i := len(all) - 1; i >= 0 && len(out.Experiments) < experimentListMax; i-- {
		e := all[i]
		if e.OwnerEntityID == scope.identityID && e.EngineEntityID == scope.engineID && e.BaselineNudgeID == run.NudgeID {
			out.Experiments = append(out.Experiments, c.experimentView(e, run))
		}
	}
	return out, nil
}

// ExperimentByID returns one of this owner's plans, with its baseline run's outcome for reference.
func (c *Core) ExperimentByID(id string, rq ...*CognitionReq) (ExperimentView, error) {
	if !experimentIDRE.MatchString(id) {
		return ExperimentView{}, ErrExperimentNotFound
	}
	scope, err := c.cognitionAccess(rq...)
	if err != nil {
		return ExperimentView{}, err
	}
	c.expMu.Lock()
	all, _, err := c.readExperiments()
	c.expMu.Unlock()
	if err != nil {
		return ExperimentView{}, ErrCognitionUnavailable
	}
	for _, e := range all {
		if e.ExperimentID == id && e.OwnerEntityID == scope.identityID && e.EngineEntityID == scope.engineID {
			run, err := c.DevelopmentRun(e.BaselineNudgeID, rq...)
			if err != nil {
				return ExperimentView{}, err
			}
			return c.experimentView(e, run), nil
		}
	}
	return ExperimentView{}, ErrExperimentNotFound
}

func (c *Core) experimentView(e Experiment, run DevelopmentRun) ExperimentView {
	v := ExperimentView{Experiment: e, Cost: "2 model requests when run: A (unchanged) then B (your change), one at a time, each with its own confirmation. Saving this plan made none.",
		Unchanged: experimentUnchanged, HeldConstant: experimentHeldConstant,
		Note: "Only the instruction sentence differs between A and B."}
	for _, s := range experimentSnapshots {
		if s.ID == e.Snapshot {
			v.SnapshotLabel = s.Label + " — " + s.Why
		}
	}
	for _, a := range []struct{ arm, label string }{{"A", "A · unchanged instruction"}, {"B", "B · your instruction"}} {
		v.Arms = append(v.Arms, ExperimentArm{Arm: a.arm, Label: a.label, State: "NOT RUN", Runnable: false, Why: experimentNotAuthorized})
	}
	v.Original = ExperimentNote{Text: run.Note, RequestID: run.RequestID, State: orNR(run.RunState), Saved: "yes (the original run)"}
	if run.Note == "" {
		v.Original.Text = ""
		v.Original.Saved = "the original Note cannot be shown as this run's (see Observe)"
	}
	return v
}

// ---------- journal ----------

func (c *Core) readExperiments() ([]Experiment, int, error) {
	b, err := os.ReadFile(c.file(experimentFile))
	if errors.Is(err, os.ErrNotExist) {
		return nil, 0, nil
	}
	if err != nil {
		return nil, 0, err
	}
	var out []Experiment
	invalid := 0
	for _, line := range strings.Split(string(b), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var e Experiment
		if json.Unmarshal([]byte(line), &e) != nil || e.Schema != experimentSchema || !experimentIDRE.MatchString(e.ExperimentID) {
			invalid++
			continue
		}
		out = append(out, e)
	}
	return out, invalid, nil
}

func (c *Core) appendExperiment(e Experiment) error {
	p := c.file(experimentFile)
	if fi, err := os.Stat(p); err == nil && fi.Size() > experimentJournalMax {
		return fmt.Errorf("%w: the local experiment journal is full; export and archive it first", ErrExperimentInvalid)
	}
	b, err := json.Marshal(e)
	if err != nil {
		return err
	}
	line := append(b, '\n')
	if r, err := os.Open(p); err == nil {
		if fi, err := r.Stat(); err == nil && fi.Size() > 0 {
			last := make([]byte, 1)
			if _, err := r.ReadAt(last, fi.Size()-1); err == nil && last[0] != '\n' {
				line = append([]byte{'\n'}, line...)
			}
		}
		r.Close()
	}
	f, err := nudgeOpenFile(p, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	if _, err := f.Write(line); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
