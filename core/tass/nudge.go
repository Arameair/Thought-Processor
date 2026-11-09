package tass

// Manual nudge workbench (Thought-Processor #96, first deliverable).
//
// The owner reads a REAL persistent Thought in Cognition and saves a manual nudge DRAFT: who it is
// for (one of the five card positions, or TASS himself), why, and what attention is requested.
//
// What this file does:
//   - validates the draft at the Go boundary and checks that the source Thought really exists in
//     the owner-reviewed, linked Engine entity (GET-only Engine reads; no model call);
//   - asks Identity for its own owner-granted action, cognition.nudge.draft, before the one local
//     write. It is NOT a default grant and is not implied by thought.process (which every unlocked
//     entity has and which Cognition reads use): the owner enables it on Identity's page with the
//     passphrase (Arameair/identity "cognition.nudge-draft"), and can revoke it at any time;
//   - appends an immutable record to a local, owner-scoped journal (nudges.jsonl, 0600, fsync).
//
// What it deliberately does NOT do:
//   - process a nudge. No Identity action for nudge processing exists yet, so every record carries
//     processing BLOCKED. Adding it needs a separate, owner-reviewed Identity policy change (#96 step 3);
//   - send anything outside this computer (no Discord, Claude, GPT-6 or GitHub recipient exists);
//   - change the source Thought, tass.md, skills, lessons, memory or any permission.

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Arameair/Thought-Processor/core/engine"
)

const (
	nudgeSchema        = "thought-processor.manual-nudge/1"
	nudgeFile          = "nudges.jsonl"
	nudgeActionMax     = 500
	nudgeDefaultExpiry = 24 * time.Hour
	nudgeMaxExpiry     = 7 * 24 * time.Hour
	nudgeListMax       = 200
	nudgeJournalMax    = 4 << 20 // refuse new drafts rather than rotate away evidence
)

// NudgePositions are the five card positions. The Liaison is one of them, but which one (and
// whether the role rotates) is an open owner decision, so no position is labelled here.
var NudgePositions = []string{"position-1", "position-2", "position-3", "position-4", "position-5"}

// NudgeReasons are the allowed reason codes for a manual nudge.
var NudgeReasons = []string{"owner_request", "new_evidence", "follow_up", "question", "correction"}

var nudgeKeyRE = regexp.MustCompile(`^[A-Za-z0-9._:-]{8,128}$`)

// NudgeDraftAction is the Identity action a draft write needs (owner-granted, never a default).
const NudgeDraftAction = "cognition.nudge.draft"

var (
	ErrNudgeInvalid    = errors.New("nudge refused")
	ErrNudgeConflict   = errors.New("nudge idempotency conflict")
	ErrNudgeWrite      = errors.New("the draft could not be saved on this computer; nothing was saved")
	ErrNudgeNotGranted = errors.New("Identity has not granted nudge drafts: enable \"cognition.nudge-draft\" on Identity's page (127.0.0.1:8796) with your passphrase")
)

// NudgeDraft is what the owner submits.
type NudgeDraft struct {
	SourceThoughtID string `json:"source_thought_id"`
	RecipientID     string `json:"recipient_id"` // "self" or position-1..5
	ReasonCode      string `json:"reason_code"`
	RequestedAction string `json:"requested_action"`
	Priority        string `json:"priority"`         // low | normal | high (default normal)
	ExpiresInHours  int    `json:"expires_in_hours"` // 0 = 24 h; at most 168
	IdempotencyKey  string `json:"idempotency_key"`
}

// NudgeProcessing is the honest processing state of a nudge.
type NudgeProcessing struct {
	State  string `json:"state"` // BLOCKED until a reviewed processing grant and route exist
	Reason string `json:"reason"`
}

// Nudge is one durable record. It is never rewritten; later state changes would be new events.
type Nudge struct {
	Schema                  string          `json:"schema"`
	NudgeID                 string          `json:"nudge_id"`
	OwnerEntityID           string          `json:"owner_entity_id"`
	SourceEngineEntityID    string          `json:"source_engine_entity_id"`
	SourceThoughtID         string          `json:"source_thought_id"`
	SourceRevision          string          `json:"source_revision"` // the Thought's updated_at when the draft was saved
	SourceEvidenceRefs      []string        `json:"source_evidence_refs"`
	SenderIdentityID        string          `json:"sender_identity_id"` // the unlocked Identity entity; never typed in
	RecipientKind           string          `json:"recipient_kind"`     // local_entity | card_position
	RecipientID             string          `json:"recipient_id"`
	ReasonCode              string          `json:"reason_code"`
	RequestedAction         string          `json:"requested_action"`
	Priority                string          `json:"priority"`
	CreatedAt               time.Time       `json:"created_at"`
	ExpiresAt               time.Time       `json:"expires_at"`
	IdempotencyKey          string          `json:"idempotency_key"`
	BodyDigest              string          `json:"body_digest"`
	Status                  string          `json:"status"` // DRAFT (EXPIRED is shown on read, never written)
	DraftDecisionID         string          `json:"draft_identity_decision_id"`
	Processing              NudgeProcessing `json:"processing"`
	ProcessingCycleID       *string         `json:"processing_cycle_id"`       // null: NOT RECORDED
	ResultThoughtID         *string         `json:"result_thought_id"`         // null: NOT RECORDED
	AuthorizationDecisionID *string         `json:"authorization_decision_id"` // processing decision; null: none
	ModelCalls              int             `json:"model_calls"`               // always 0 for a draft
}

// NudgeView is what the page gets: the stored record plus the state as of now.
type NudgeView struct {
	Nudge
	Replayed   bool     `json:"replayed,omitempty"`
	PreRestore bool     `json:"pre_restore,omitempty"` // Stage 2: existed before a tass.sh restore: copy only
	Run        *RunView `json:"run,omitempty"`         // Stage 2: processing receipt, if submitted
}

// NudgeList is the owner-scoped history.
type NudgeList struct {
	OwnerEntityID string          `json:"owner_entity_id"`
	Nudges        []NudgeView     `json:"nudges"`
	InvalidLines  int             `json:"invalid_lines"`
	Positions     []string        `json:"positions"`
	Reasons       []string        `json:"reasons"`
	Processing    NudgeProcessing `json:"processing"`
	Locked        string          `json:"processing_locked,omitempty"` // Stage 2: read-only RECOVERY_REQUIRED for all processing
	Note          string          `json:"note"`
}

func nudgeBlocked() NudgeProcessing {
	return NudgeProcessing{State: "BLOCKED", Reason: "No Identity action for nudge processing exists yet. Processing needs a separate, " +
		"owner-reviewed Identity policy change and a verified local route (Thought-Processor #96). Nothing was queued."}
}

func nudgeOneOf(s string, allowed []string) bool {
	for _, a := range allowed {
		if s == a {
			return true
		}
	}
	return false
}

// normalize validates a draft without touching any service.
func (d *NudgeDraft) normalize() (recipientKind string, expiry time.Duration, err error) {
	bad := func(why string) (string, time.Duration, error) {
		return "", 0, fmt.Errorf("%w: %s", ErrNudgeInvalid, why)
	}
	d.SourceThoughtID = strings.TrimSpace(d.SourceThoughtID)
	d.RecipientID = strings.TrimSpace(d.RecipientID)
	d.ReasonCode = strings.TrimSpace(d.ReasonCode)
	d.RequestedAction = strings.TrimSpace(d.RequestedAction)
	d.Priority = strings.TrimSpace(d.Priority)
	d.IdempotencyKey = strings.TrimSpace(d.IdempotencyKey)
	if !engine.CognitionID.MatchString(d.SourceThoughtID) {
		return bad("source_thought_id must be a real persistent Thought ID")
	}
	switch {
	case d.RecipientID == "self":
		recipientKind = "local_entity"
	case nudgeOneOf(d.RecipientID, NudgePositions):
		recipientKind = "card_position"
	default:
		// Discord, Claude, GPT-6, GitHub, other entities: not recipients in v1 (#96 M12).
		return bad("recipient must be self or one of the five card positions; external recipients are not available")
	}
	if !nudgeOneOf(d.ReasonCode, NudgeReasons) {
		return bad("reason_code must be one of " + strings.Join(NudgeReasons, ", "))
	}
	if d.RequestedAction == "" || !utf8.ValidString(d.RequestedAction) || utf8.RuneCountInString(d.RequestedAction) > nudgeActionMax {
		return bad(fmt.Sprintf("requested_action must be 1-%d characters", nudgeActionMax))
	}
	for _, r := range d.RequestedAction {
		if r < 0x20 && r != '\n' && r != '\t' {
			return bad("requested_action contains control characters")
		}
	}
	if d.Priority == "" {
		d.Priority = "normal"
	}
	if !nudgeOneOf(d.Priority, []string{"low", "normal", "high"}) {
		return bad("priority must be low, normal or high")
	}
	// Compare hours before multiplying: a huge int must not overflow past the bound.
	if d.ExpiresInHours < 0 || d.ExpiresInHours > int(nudgeMaxExpiry/time.Hour) {
		return bad("expires_in_hours must be 0-168")
	}
	expiry = nudgeDefaultExpiry
	if d.ExpiresInHours > 0 {
		expiry = time.Duration(d.ExpiresInHours) * time.Hour
	}
	if !nudgeKeyRE.MatchString(d.IdempotencyKey) {
		return bad("idempotency_key must be 8-128 of A-Z a-z 0-9 . _ : -")
	}
	return recipientKind, expiry, nil
}

func (d NudgeDraft) digest() string {
	b, _ := json.Marshal([]any{d.SourceThoughtID, d.RecipientID, d.ReasonCode, d.RequestedAction, d.Priority, d.ExpiresInHours})
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

// nudgeLine is one physical line of the journal with its starting byte offset.
type nudgeLine struct {
	Offset int64
	Valid  bool
	N      Nudge
}

// readNudgeLines splits the journal into lines at '\n', keeping each line's starting byte offset
// (Stage 2 restore fence, R3-04). A final line without '\n' is a line too. A line longer than 64 KiB
// or that isn't a valid draft record is kept as invalid (never repaired or overwritten).
func (c *Core) readNudgeLines() ([]nudgeLine, []byte, error) {
	raw, err := os.ReadFile(c.file(nudgeFile))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	var out []nudgeLine
	off := int64(0)
	for len(raw[off:]) > 0 {
		rest := raw[off:]
		n := bytes.IndexByte(rest, '\n')
		line := rest
		next := off + int64(len(rest))
		if n >= 0 {
			line, next = rest[:n], off+int64(n)+1
		}
		l := nudgeLine{Offset: off}
		if len(bytes.TrimSpace(line)) > 0 {
			var rec Nudge
			if len(line) <= 64<<10 && json.Unmarshal(line, &rec) == nil && rec.Schema == nudgeSchema && rec.NudgeID != "" && rec.OwnerEntityID != "" {
				l.Valid, l.N = true, rec
			}
			out = append(out, l)
		}
		off = next
	}
	return out, raw, nil
}

// readNudges returns every valid record in the journal (all identities); the caller filters.
// Damaged lines are counted, never repaired or overwritten.
func (c *Core) readNudges() ([]Nudge, int, error) {
	lines, _, err := c.readNudgeLines()
	if err != nil {
		return nil, 0, err
	}
	var out []Nudge
	invalid := 0
	for _, l := range lines {
		if !l.Valid {
			invalid++
			continue
		}
		out = append(out, l.N)
	}
	return out, invalid, nil
}

// ---------- restore fence (Stage 2, design rev 4 §4.6) ----------

const restoreFenceFile = "restore-fence.json"

type restoreFence struct {
	Schema       string `json:"schema"`
	RestoredAt   string `json:"restored_at"`
	Backup       string `json:"backup"`
	JournalBytes int64  `json:"journal_bytes"`
	JournalSHA   string `json:"journal_sha256"`
}

// nudgeFence reads the restore fence written by tass.sh restore_data() and checks it against the
// journal. It returns the boundary L (drafts whose record starts before L are PRE_RESTORE) and a
// non-empty lock reason if processing must be refused (read-only RECOVERY_REQUIRED).
func (c *Core) nudgeFence(lines []nudgeLine, raw []byte) (int64, string) {
	b, err := os.ReadFile(c.file(restoreFenceFile))
	if errors.Is(err, os.ErrNotExist) {
		return 0, dupLock(lines)
	}
	var f restoreFence
	if err != nil || json.Unmarshal(b, &f) != nil || f.Schema != "tass.restore-fence/2" || f.JournalBytes < 0 || len(f.JournalSHA) != 64 {
		return 0, "the restore fence is unreadable"
	}
	if int64(len(raw)) < f.JournalBytes {
		return f.JournalBytes, "the draft journal is shorter than when it was restored"
	}
	sum := sha256.Sum256(raw[:f.JournalBytes])
	if hex.EncodeToString(sum[:]) != f.JournalSHA {
		return f.JournalBytes, "the restored part of the draft journal changed since the restore"
	}
	for _, l := range lines {
		if l.Offset < f.JournalBytes && !l.Valid {
			return f.JournalBytes, "the restored draft journal has an invalid record"
		}
	}
	return f.JournalBytes, dupLock(lines)
}

// dupLock: the same nudge ID twice anywhere is ambiguous: lock.
func dupLock(lines []nudgeLine) string {
	seen := map[string]bool{}
	for _, l := range lines {
		if l.Valid {
			if seen[l.N.NudgeID] {
				return "the draft journal has a duplicate nudge ID"
			}
			seen[l.N.NudgeID] = true
		}
	}
	return ""
}

// preRestore: nudge IDs with a record starting before the fence boundary.
func preRestore(lines []nudgeLine, boundary int64) map[string]bool {
	m := map[string]bool{}
	for _, l := range lines {
		if l.Valid && l.Offset < boundary {
			m[l.N.NudgeID] = true
		}
	}
	return m
}

// nudgeOpenFile is os.OpenFile; tests replace it to inject write failures.
var nudgeOpenFile = func(name string, flag int, perm os.FileMode) (nudgeWriter, error) {
	return os.OpenFile(name, flag, perm)
}

type nudgeWriter interface {
	Write([]byte) (int, error)
	Sync() error
	Close() error
}

func (c *Core) appendNudge(n Nudge) error {
	p := c.file(nudgeFile)
	if fi, err := os.Stat(p); err == nil && fi.Size() > nudgeJournalMax {
		return fmt.Errorf("%w: the local nudge journal is full; export and archive it first", ErrNudgeInvalid)
	}
	b, err := json.Marshal(n)
	if err != nil {
		return err
	}
	// If an earlier write was cut off mid-line, start on a fresh line so this record stays
	// readable (the damaged partial line is kept and counted, never rewritten).
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

func (c *Core) nudgeView(n Nudge) NudgeView {
	v := NudgeView{Nudge: n}
	if v.Status == "DRAFT" && !v.ExpiresAt.IsZero() && c.now().After(v.ExpiresAt) {
		v.Status = "EXPIRED"
	}
	v.Processing = nudgeBlocked()
	if n.RecipientID == "self" {
		v.Processing = NudgeProcessing{State: "AVAILABLE", Reason: "You can process this nudge once: it makes one model request that develops this Thought by one note. " +
			"It needs the owner grant cognition.nudge-process and your confirmation each time."}
	}
	return v
}

// SaveNudgeDraft validates, checks the source Thought, asks Identity, then appends one record.
// It makes zero model calls: the Engine is only read (GET) through CognitionThought.
func (c *Core) SaveNudgeDraft(d NudgeDraft) (NudgeView, error) {
	recipientKind, expiry, err := d.normalize()
	if err != nil {
		return NudgeView{}, err
	}
	scope, err := c.cognitionAccess()
	if err != nil {
		return NudgeView{}, err
	}
	if scope.engineID == "" {
		return NudgeView{}, fmt.Errorf("%w: link this Identity to its Engine history in Cognition first", ErrNudgeInvalid)
	}
	digest := d.digest()
	// Every POST success, a replay included, needs a FRESH owner grant: revoking
	// cognition.nudge.draft takes effect immediately, even for a key that was saved before.
	// (Reading saved drafts is GET /api/nudges, which needs only the Cognition read grants.)
	dec := c.cfg.IdentitySvc.Authorize(NudgeDraftAction, "thought-processor:cognition.nudge.draft "+d.SourceThoughtID)
	if !dec.Allow || dec.DecisionID == "" || dec.EntityID != scope.identityID {
		return NudgeView{}, ErrNudgeNotGranted
	}

	c.nudgeMu.Lock()
	defer c.nudgeMu.Unlock()
	all, _, err := c.readNudges()
	if err != nil {
		return NudgeView{}, ErrCognitionUnavailable
	}
	for _, n := range all {
		if n.OwnerEntityID == scope.identityID && n.IdempotencyKey == d.IdempotencyKey {
			// A key from before a relink belongs to other Engine history: never replay it (it would
			// return a draft that GET hides). Refuse without returning any of its content.
			if n.SourceEngineEntityID != scope.engineID {
				return NudgeView{}, fmt.Errorf("%w: that key belongs to a different Engine link; use a new form", ErrNudgeConflict)
			}
			if n.BodyDigest != digest {
				return NudgeView{}, fmt.Errorf("%w: that key was already used for a different nudge", ErrNudgeConflict)
			}
			v := c.nudgeView(n)
			v.Replayed = true
			return v, nil
		}
	}
	// The source must be a real persistent Thought of THIS linked Engine entity.
	th, err := c.CognitionThought(d.SourceThoughtID)
	if err != nil {
		return NudgeView{}, err
	}
	refs := []string{}
	for _, t := range th.Traces {
		if t.Verified && engine.CognitionID.MatchString(t.TraceID) && len(refs) < 16 {
			refs = append(refs, "trace:"+t.TraceID)
		}
	}
	for _, id := range th.RelatedCycleIDs {
		if engine.CognitionID.MatchString(id) && len(refs) < 32 {
			refs = append(refs, "cycle:"+id)
		}
	}
	rnd := make([]byte, 10)
	if _, err := rand.Read(rnd); err != nil {
		return NudgeView{}, err
	}
	now := c.now().UTC()
	rev := "NOT RECORDED"
	if !th.UpdatedAt.IsZero() {
		rev = th.UpdatedAt.UTC().Format(time.RFC3339Nano)
	}
	n := Nudge{
		Schema: nudgeSchema, NudgeID: "nudge_" + hex.EncodeToString(rnd),
		OwnerEntityID: scope.identityID, SourceEngineEntityID: scope.engineID,
		SourceThoughtID: th.ThoughtID, SourceRevision: rev, SourceEvidenceRefs: refs,
		SenderIdentityID: scope.identityID, RecipientKind: recipientKind, RecipientID: d.RecipientID,
		ReasonCode: d.ReasonCode, RequestedAction: d.RequestedAction, Priority: d.Priority,
		CreatedAt: now, ExpiresAt: now.Add(expiry), IdempotencyKey: d.IdempotencyKey, BodyDigest: digest,
		Status: "DRAFT", DraftDecisionID: dec.DecisionID, Processing: nudgeBlocked(),
	}
	if err := c.appendNudge(n); err != nil {
		if errors.Is(err, ErrNudgeInvalid) {
			return NudgeView{}, err
		}
		return NudgeView{}, fmt.Errorf("%w: %v", ErrNudgeWrite, err)
	}
	c.event("sync", fmt.Sprintf("Saved manual nudge draft %s for %s (not processed: no processing grant exists).", n.NudgeID, n.RecipientID))
	return c.nudgeView(n), nil
}

// NudgeHistory lists this Identity entity's nudges (newest first), optionally for one source Thought.
func (c *Core) NudgeHistory(thoughtID string) (NudgeList, error) {
	if thoughtID != "" && !engine.CognitionID.MatchString(thoughtID) {
		return NudgeList{}, ErrCognitionUnavailable
	}
	scope, err := c.cognitionAccess()
	if err != nil {
		return NudgeList{}, err
	}
	c.nudgeMu.Lock()
	all, invalid, err := c.readNudges()
	c.nudgeMu.Unlock()
	if err != nil {
		return NudgeList{}, ErrCognitionUnavailable
	}
	out := NudgeList{OwnerEntityID: scope.identityID, Nudges: []NudgeView{}, InvalidLines: invalid,
		Positions: NudgePositions, Reasons: NudgeReasons, Processing: nudgeBlocked(),
		Note: "Manual nudge drafts are local to this computer and never synced to GitHub. Saving a draft makes no model call."}
	out.Locked = c.processingLock()
	lines, raw, _ := c.readNudgeLines()
	boundary, _ := c.nudgeFence(lines, raw)
	pre := preRestore(lines, boundary)
	for i := len(all) - 1; i >= 0; i-- {
		n := all[i]
		// Owner AND linked Engine entity must match: a relinked Identity does not inherit old drafts.
		if n.OwnerEntityID != scope.identityID || n.SourceEngineEntityID != scope.engineID {
			continue
		}
		if thoughtID != "" && n.SourceThoughtID != thoughtID {
			continue
		}
		nv := c.nudgeView(n)
		nv.PreRestore = pre[n.NudgeID]
		nv.Run = c.runView(n.NudgeID)
		if nv.PreRestore && nv.Run == nil {
			nv.Processing = NudgeProcessing{State: "PRE_RESTORE", Reason: "This draft existed before a restore and may already have been processed; it can't be processed again. Use Copy as new."}
		}
		out.Nudges = append(out.Nudges, nv)
		if len(out.Nudges) >= nudgeListMax {
			break
		}
	}
	sort.SliceStable(out.Nudges, func(i, j int) bool { return out.Nudges[i].CreatedAt.After(out.Nudges[j].CreatedAt) })
	return out, nil
}
