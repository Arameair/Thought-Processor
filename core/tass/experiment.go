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
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	// ErrExperimentJournal: the local journal is damaged, over its size ceiling, or not a private regular
	// file. Nothing is read from it and nothing is appended to it until the owner has looked at the file.
	ErrExperimentJournal = errors.New("the experiment journal on this computer is damaged, too large or not private; nothing was read from it and nothing was changed")

	experimentIDRE = regexp.MustCompile(`^exp_[0-9a-f]{20}$`)
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
	if v == "" {
		return fmt.Errorf("%w: write the changed instruction sentence", ErrExperimentInvalid)
	}
	if !utf8.ValidString(v) {
		return fmt.Errorf("%w: the instruction is not valid text", ErrExperimentInvalid)
	}
	if utf8.RuneCountInString(v) > experimentVariantMax {
		return fmt.Errorf("%w: the instruction is longer than %d characters", ErrExperimentInvalid, experimentVariantMax)
	}
	for _, r := range v {
		switch {
		// Every line break Unicode knows: CR, LF, VT, FF, NEL (U+0085), LINE SEPARATOR (U+2028),
		// PARAGRAPH SEPARATOR (U+2029).
		case r == '\n' || r == '\r' || r == '\v' || r == '\f' || r == 0x85 || unicode.Is(unicode.Zl, r) || unicode.Is(unicode.Zp, r):
			return fmt.Errorf("%w: the instruction must be one line; the reply format lines are fixed", ErrExperimentInvalid)
		case unicode.IsControl(r):
			return fmt.Errorf("%w: the instruction contains a control character", ErrExperimentInvalid)
		// Format characters (zero-width, joiners, direction overrides, BOM), private-use and unassigned code
		// points cannot be seen on the page, so what is saved would not be what the owner read.
		case unicode.Is(unicode.Cf, r) || unicode.Is(unicode.Co, r) || unicode.Is(unicode.Cs, r) || r == utf8.RuneError || !unicode.IsPrint(r) && !unicode.IsSpace(r):
			return fmt.Errorf("%w: the instruction contains an invisible or unprintable character (U+%04X)", ErrExperimentInvalid, r)
		}
	}
	if experimentHasContractMarker(v) {
		return fmt.Errorf("%w: the instruction may not contain NOTE: or STATE: (in any script or look-alike punctuation); the reply format is fixed so the Engine's parser stays valid", ErrExperimentInvalid)
	}
	if strings.Join(strings.Fields(v), " ") == experimentControl { // Fields splits on every Unicode space
		return fmt.Errorf("%w: B must differ from A: change the instruction sentence", ErrExperimentInvalid)
	}
	return nil
}

// experimentHasContractMarker reports whether v contains the word NOTE or STATE followed by something that
// is, or could be read as, the delimiter of a reply-format line. It fails closed:
//   - letters are folded first: fullwidth forms (ＮＯＴＥ) to ASCII, then Unicode upper-casing (ſtate → STATE);
//   - after the word, any run of spaces (every Unicode space), emphasis marks (* _ ` ~ " ') and combining
//     marks is skipped;
//   - the next character is then a marker if it is ASCII ':' or '=', or ANY non-ASCII punctuation or symbol.
//     That covers every colon and equals look-alike (： ﹕ ∶ ꞉ ։ ＝ ⩵ …) without a list that could miss one.
//     Ordinary ASCII punctuation after the word ("take note, then …", "the state.") is allowed.
//
// The word must start at a word boundary, so "footnotes:" and "estate:" are not markers.
func experimentHasContractMarker(v string) bool {
	fold := make([]rune, 0, len(v))
	for _, r := range v {
		if r >= 0xFF01 && r <= 0xFF5E { // fullwidth ASCII block → ASCII
			r -= 0xFEE0
		}
		fold = append(fold, unicode.ToUpper(r))
	}
	isWord := func(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) }
	for i := range fold {
		if i > 0 && isWord(fold[i-1]) {
			continue
		}
		for _, w := range [][]rune{[]rune("NOTE"), []rune("STATE")} {
			if i+len(w) > len(fold) || string(fold[i:i+len(w)]) != string(w) {
				continue
			}
			j := i + len(w)
			for j < len(fold) && (unicode.IsSpace(fold[j]) || unicode.Is(unicode.Mn, fold[j]) || unicode.Is(unicode.Me, fold[j]) || strings.ContainsRune("*_`~\"'", fold[j])) {
				j++
			}
			if j == len(fold) {
				continue
			}
			if r := fold[j]; r == ':' || r == '=' || (r > unicode.MaxASCII && (unicode.IsPunct(r) || unicode.IsSymbol(r))) {
				return true
			}
		}
	}
	return false
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
	scope, err := c.cognitionAccess(rq...)
	if err != nil {
		return ExperimentView{}, err
	}
	// An unsafe journal refuses the request before anything else is read (and again, under the same lock
	// as the append, below).
	if err := c.experimentJournalSafe(); err != nil {
		return ExperimentView{}, err
	}
	// The owner-scoped, read-only run view decides that the baseline exists and is this owner's.
	run, err := c.DevelopmentRun(d.BaselineNudgeID, rq...)
	if err != nil {
		return ExperimentView{}, err
	}
	if run.RequestID == "" {
		return ExperimentView{}, fmt.Errorf("%w: the baseline run has no recorded request, so there is nothing to compare against", ErrExperimentInvalid)
	}
	digest := d.digest()

	c.expMu.Lock()
	defer c.expMu.Unlock()
	all, err := c.readExperiments()
	if err != nil {
		return ExperimentView{}, err
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
		if errors.Is(err, ErrExperimentInvalid) || errors.Is(err, ErrExperimentJournal) {
			return ExperimentView{}, err
		}
		return ExperimentView{}, fmt.Errorf("%w: %v", ErrExperimentWrite, err)
	}
	c.event("sync", fmt.Sprintf("Saved experiment plan %s (instruction change; not run: running needs separate owner decisions).", e.ExperimentID))
	return c.experimentView(e, run), nil
}

// Experiments lists this owner's plans for one baseline run (newest first), with the form's options.
func (c *Core) Experiments(baselineNudgeID string, rq ...*CognitionReq) (ExperimentList, error) {
	scope, err := c.cognitionAccess(rq...)
	if err != nil {
		return ExperimentList{}, err
	}
	if err := c.experimentJournalSafe(); err != nil {
		return ExperimentList{}, err
	}
	run, err := c.DevelopmentRun(baselineNudgeID, rq...)
	if err != nil {
		return ExperimentList{}, err
	}
	c.expMu.Lock()
	all, err := c.readExperiments()
	c.expMu.Unlock()
	if err != nil {
		return ExperimentList{}, err
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
	all, err := c.readExperiments()
	c.expMu.Unlock()
	if err != nil {
		return ExperimentView{}, err
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
//
// experiments.jsonl is one JSON plan per line, each line ended by '\n', appended with one write and an
// fsync. Every access first checks the file WITHOUT reading it (experimentJournalCheck) and fails closed
// with ErrExperimentJournal when it:
//   - is not a regular file (a symlink, directory or device is refused; Lstat, never followed);
//   - has any permission bit beyond owner read/write (anything more permissive than 0600). It is created 0600;
//   - is larger than experimentJournalMax. The size is checked before the file is opened, the read is
//     bounded by the same ceiling, and an append that would cross it is refused, so the journal can never
//     grow into a state that locks itself.
// Then every COMPLETE record (a line ended by '\n') must be a valid plan: valid JSON, this schema, a
// well-formed and unique experiment ID. One bad complete record, a blank line or a duplicate ID anywhere
// refuses the whole journal: no plan is returned and nothing is appended.
//
// The one recovery rule is for bytes after the last '\n'. A record is acknowledged only after its full
// line, newline included, was written and fsynced, so trailing bytes without a newline are an append that
// was cut off and never acknowledged. Reads never return them, whatever they contain, and never change the
// file. The next save first copies exactly those bytes to experiments.jsonl.partial-<time> (0600, fsync),
// cuts the journal back to its last complete record (fsync), and only then appends.

// experimentOpenRead opens the journal for reading; tests count calls to prove the size check comes first.
var experimentOpenRead = func(name string) (*os.File, error) { return os.Open(name) }

// experimentJournalCheck inspects the journal without opening it. exists=false means there is no journal.
func (c *Core) experimentJournalCheck() (size int64, exists bool, err error) {
	fi, err := os.Lstat(c.file(experimentFile))
	if errors.Is(err, os.ErrNotExist) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("%w (it cannot be inspected)", ErrExperimentJournal)
	}
	switch {
	case !fi.Mode().IsRegular():
		return 0, true, fmt.Errorf("%w (it is not a regular file)", ErrExperimentJournal)
	case fi.Mode().Perm()&^0o600 != 0 || fi.Mode()&(os.ModeSetuid|os.ModeSetgid|os.ModeSticky) != 0:
		return 0, true, fmt.Errorf("%w (its permissions are %04o; nothing beyond owner read/write, 0600, is accepted)", ErrExperimentJournal, fi.Mode().Perm())
	case fi.Size() > experimentJournalMax:
		return 0, true, fmt.Errorf("%w (it is %d bytes; the limit is %d)", ErrExperimentJournal, fi.Size(), experimentJournalMax)
	}
	return fi.Size(), true, nil
}

// experimentJournalLoad returns the valid plans and the length of the complete-record prefix (the offset
// just after the last '\n'). Caller holds expMu.
func (c *Core) experimentJournalLoad() (plans []Experiment, complete int64, size int64, err error) {
	size, exists, err := c.experimentJournalCheck()
	if err != nil || !exists {
		return nil, 0, 0, err
	}
	f, err := experimentOpenRead(c.file(experimentFile))
	if err != nil {
		return nil, 0, 0, fmt.Errorf("%w (it cannot be opened)", ErrExperimentJournal)
	}
	defer f.Close()
	// Bounded: never more than the ceiling plus one byte, whatever the file became since the check.
	b, err := io.ReadAll(io.LimitReader(f, experimentJournalMax+1))
	if err != nil {
		return nil, 0, 0, fmt.Errorf("%w (it cannot be read)", ErrExperimentJournal)
	}
	if len(b) > experimentJournalMax {
		return nil, 0, 0, fmt.Errorf("%w (it grew past the limit of %d bytes)", ErrExperimentJournal, experimentJournalMax)
	}
	end := bytes.LastIndexByte(b, '\n') + 1 // 0 when there is no complete record
	seen := map[string]bool{}
	n := 0
	for rest := b[:end]; len(rest) > 0; {
		i := bytes.IndexByte(rest, '\n')
		line := rest[:i]
		rest = rest[i+1:]
		n++
		var e Experiment
		dec := json.NewDecoder(bytes.NewReader(line))
		if dec.Decode(&e) != nil || dec.More() || e.Schema != experimentSchema || !experimentIDRE.MatchString(e.ExperimentID) || seen[e.ExperimentID] {
			return nil, 0, 0, fmt.Errorf("%w (record %d is not a valid, unique plan)", ErrExperimentJournal, n)
		}
		seen[e.ExperimentID] = true
		plans = append(plans, e)
	}
	return plans, int64(end), int64(len(b)), nil
}

// experimentJournalSafe validates the whole journal and returns nothing from it.
func (c *Core) experimentJournalSafe() error {
	c.expMu.Lock()
	defer c.expMu.Unlock()
	_, err := c.readExperiments()
	return err
}

func (c *Core) readExperiments() ([]Experiment, error) {
	plans, _, _, err := c.experimentJournalLoad()
	return plans, err
}

// appendExperiment re-validates the whole journal, quarantines a cut-off trailing append if there is one,
// then appends one record. Caller holds expMu.
func (c *Core) appendExperiment(e Experiment) error {
	p := c.file(experimentFile)
	_, complete, size, err := c.experimentJournalLoad()
	if err != nil {
		return err
	}
	b, err := json.Marshal(e)
	if err != nil {
		return err
	}
	line := append(b, '\n')
	if complete+int64(len(line)) > experimentJournalMax {
		return fmt.Errorf("%w: the local experiment journal is full; export and archive it first", ErrExperimentInvalid)
	}
	if size > complete {
		if err := c.quarantineExperimentTail(p, complete, size); err != nil {
			return err
		}
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
	if err := f.Close(); err != nil {
		return err
	}
	// The journal must end up private whatever the process umask or a concurrent chmod did.
	if _, _, err := c.experimentJournalCheck(); err != nil {
		return err
	}
	return nil
}

// quarantineExperimentTail moves the bytes [complete, size) of the journal (a cut-off, unacknowledged
// append) into a private side file and cuts the journal back to its last complete record.
func (c *Core) quarantineExperimentTail(p string, complete, size int64) error {
	f, err := os.OpenFile(p, os.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("%w (its cut-off last write cannot be set aside)", ErrExperimentJournal)
	}
	defer f.Close()
	tail := make([]byte, size-complete)
	if _, err := io.ReadFull(io.NewSectionReader(f, complete, size-complete), tail); err != nil {
		return fmt.Errorf("%w (its cut-off last write cannot be set aside)", ErrExperimentJournal)
	}
	q, err := os.OpenFile(fmt.Sprintf("%s.partial-%s", p, c.now().UTC().Format("20060102T150405.000000000Z")), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("%w (its cut-off last write cannot be set aside)", ErrExperimentJournal)
	}
	if _, err := q.Write(tail); err == nil {
		err = q.Sync()
	}
	if cerr := q.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return fmt.Errorf("%w (its cut-off last write cannot be set aside)", ErrExperimentJournal)
	}
	if err := f.Truncate(complete); err != nil {
		return fmt.Errorf("%w (it cannot be cut back to its last complete record)", ErrExperimentJournal)
	}
	if err := f.Sync(); err != nil {
		return fmt.Errorf("%w (it cannot be cut back to its last complete record)", ErrExperimentJournal)
	}
	c.event("problem", "The experiment journal ended in a cut-off write. It was set aside in a .partial file and the journal was cut back to its last complete record.")
	return nil
}
