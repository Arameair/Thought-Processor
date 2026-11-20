// Package workloop is an EXPERIMENT (branch experiment/workloop-claude-demo-20261119):
// a small, testable model of the TASS development team loop. It is not wired into
// tass-core, has no command, and is never run against real GitHub by this package's
// tests. See README.md.
//
// Design in one paragraph: every workstream has its own append-only ledger file
// (team/ws/<id>.md). A ledger is the only source of truth for that workstream; its
// events form a hash chain and are re-validated by a state machine on every read, so
// a FAIL can't become PASS without a new candidate and a new independent audit, an
// audit is bound to one exact commit, and a hand edit fails closed. team/REGISTER.md
// is only ever *derived* from the ledgers, so racing writers re-render instead of
// overwriting each other. Writes use Core's existing GitHub client (conditional GET,
// sha-checked PUT) and Core's tass.md merge; permission comes from a local Policy
// (standing in for Identity), never from Markdown.
package workloop

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Arameair/Thought-Processor/core/github"
)

// Role is what a participant does in one workstream.
type Role string

const (
	Coordinator Role = "coordinator" // GPT-6: opens workstreams, assigns roles
	Implementer Role = "implementer" // Claude: builds and repairs candidates
	Auditor     Role = "auditor"     // Codex: independently tests one exact revision
	Observer    Role = "observer"    // TASS: reads its assignment, acknowledges
	Owner       Role = "owner"       // Dustin: the only role that can approve
)

// Event kinds.
const (
	Assign    = "ASSIGN"
	Candidate = "CANDIDATE"
	Audit     = "AUDIT"
	Approve   = "OWNER_APPROVE"
)

// Workstream states (derived, never stored as truth).
const (
	StAssigned      = "ASSIGNED"
	StAwaitingAudit = "AWAITING_AUDIT"
	StAuditFailed   = "AUDIT_FAILED"
	StAuditPassed   = "AUDIT_PASSED" // still needs the owner
	StApproved      = "OWNER_APPROVED"
)

var (
	// ErrUnauthorized: the local policy refused before any request was made.
	ErrUnauthorized = errors.New("workloop: not authorized")
	// ErrRefused: the transition is not allowed by the workstream state machine.
	ErrRefused = errors.New("workloop: transition refused")
	// ErrTampered: a ledger is not the canonical rendering of a valid event chain.
	ErrTampered = errors.New("workloop: ledger failed verification")
	// ErrPrivate: the text would publish a secret or private memory.
	ErrPrivate = errors.New("workloop: refusing to publish private content")
	// ErrTarget: the target would touch the identity/lesson sync (live:tass.md, lessons/).
	ErrTarget = errors.New("workloop: target overlaps the identity/lesson sync")
)

// Event is one transition. Its JSON is the hashed, canonical form.
type Event struct {
	Seq       int       `json:"seq"`
	At        time.Time `json:"at"`
	Actor     string    `json:"actor"`
	Role      Role      `json:"role"`
	Kind      string    `json:"kind"`
	Task      string    `json:"task,omitempty"`      // ASSIGN
	Assignees []string  `json:"assignees,omitempty"` // ASSIGN: "role=actor"
	Commit    string    `json:"commit,omitempty"`    // CANDIDATE, AUDIT: exact 40-hex revision
	Verdict   string    `json:"verdict,omitempty"`   // AUDIT: PASS | FAIL
	Evidence  string    `json:"evidence,omitempty"`  // command + result summary
	Repair    []string  `json:"repair,omitempty"`    // AUDIT FAIL: actionable instructions
	ReadRules string    `json:"read_rules"`          // blob sha of AGENTS.md read this iteration
	Prev      string    `json:"prev"`                // hash of the previous event ("" for e1)
}

func (e Event) hash() string {
	b, _ := json.Marshal(e)
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

// State is what a ledger's events add up to.
type State struct {
	ID, Task, Phase string
	Roles           map[Role]string
	Commit          string          // latest candidate
	Failed          map[string]bool // commits that failed an audit
	Repair          []string        // latest FAIL's instructions
	Last            Event
	Head            string // hash of the last event
}

var commitRE = regexp.MustCompile(`^[0-9a-f]{40}$`)

// Apply validates e against s and returns the next state. Nothing is trusted from the
// file: the whole chain is replayed through Apply on every read.
func Apply(s State, e Event) (State, error) {
	bad := func(f string, a ...any) (State, error) {
		return s, fmt.Errorf("%w: e%d %s: %s", ErrRefused, e.Seq, e.Kind, fmt.Sprintf(f, a...))
	}
	if e.Seq != s.Last.Seq+1 || e.Prev != s.Head {
		return bad("breaks the chain (seq %d, expected %d)", e.Seq, s.Last.Seq+1)
	}
	if e.ReadRules == "" {
		return bad("did not record which project instructions (AGENTS.md) were read")
	}
	if e.Seq > 1 && e.At.Before(s.Last.At) {
		return bad("time goes backwards")
	}
	if e.Seq > 1 && s.Roles[e.Role] != e.Actor {
		return bad("%s is not the assigned %s", e.Actor, e.Role)
	}
	n := s
	n.Failed = map[string]bool{}
	for k := range s.Failed {
		n.Failed[k] = true
	}
	switch e.Kind {
	case Assign:
		if e.Seq != 1 || e.Role != Coordinator || e.Task == "" {
			return bad("only the coordinator opens a workstream, once, with a task")
		}
		n.Roles = map[Role]string{Coordinator: e.Actor}
		for _, a := range e.Assignees {
			r, who, ok := strings.Cut(a, "=")
			if !ok || who == "" {
				return bad("bad assignee %q", a)
			}
			n.Roles[Role(r)] = who
		}
		if n.Roles[Implementer] == "" || n.Roles[Auditor] == "" || n.Roles[Owner] == "" {
			return bad("needs an implementer, an auditor and an owner")
		}
		if n.Roles[Implementer] == n.Roles[Auditor] {
			return bad("the auditor must be independent of the implementer")
		}
		n.Task, n.Phase = e.Task, StAssigned
	case Candidate:
		if e.Role != Implementer || (s.Phase != StAssigned && s.Phase != StAuditFailed) {
			return bad("only the implementer submits, and only when nothing is awaiting audit (phase %s)", s.Phase)
		}
		if !commitRE.MatchString(e.Commit) || e.Evidence == "" {
			return bad("needs an exact 40-hex commit and test evidence")
		}
		if s.Failed[e.Commit] {
			return bad("%s already failed an audit: resubmitting an unchanged revision", e.Commit[:12])
		}
		n.Commit, n.Phase, n.Repair = e.Commit, StAwaitingAudit, nil
	case Audit:
		if e.Role != Auditor || s.Phase != StAwaitingAudit {
			return bad("only the independent auditor audits, and only an awaiting candidate (phase %s)", s.Phase)
		}
		if e.Commit != s.Commit {
			return bad("audited %s but the candidate is %s", short(e.Commit), short(s.Commit))
		}
		switch e.Verdict {
		case "PASS":
			n.Phase = StAuditPassed
		case "FAIL":
			if len(e.Repair) == 0 {
				return bad("a FAIL must say what to repair")
			}
			n.Phase, n.Repair = StAuditFailed, e.Repair
			n.Failed[e.Commit] = true
		default:
			return bad("verdict must be PASS or FAIL")
		}
	case Approve:
		if e.Role != Owner || s.Phase != StAuditPassed || e.Commit != s.Commit {
			return bad("only the owner approves, and only the exact revision that passed")
		}
		n.Phase = StApproved
	default:
		return bad("unknown kind")
	}
	n.Last, n.Head = e, e.hash()
	return n, nil
}

func short(c string) string {
	if len(c) > 12 {
		return c[:12]
	}
	return c
}

// Replay validates a whole chain.
func Replay(id string, evs []Event) (State, error) {
	s := State{ID: id}
	for _, e := range evs {
		var err error
		if s, err = Apply(s, e); err != nil {
			return s, err
		}
	}
	return s, nil
}

// --- Markdown rendering ----------------------------------------------------------------

const ledgerNote = "> Append-only ledger. Every event is re-validated on each read (hash chain + state machine); " +
	"a hand edit anywhere in this file makes the workstream fail closed. Markdown here never grants permission."

// RenderLedger is the canonical text of a workstream ledger.
func RenderLedger(id string, evs []Event) string {
	s, err := Replay(id, evs)
	var b strings.Builder
	fmt.Fprintf(&b, "# Workstream %s\n\n%s\n\n", id, ledgerNote)
	if err == nil && len(evs) > 0 {
		fmt.Fprintf(&b, "**Task:** %s  \n**Phase:** `%s`", s.Task, s.Phase)
		if s.Commit != "" {
			fmt.Fprintf(&b, " · **candidate:** `%s`", s.Commit)
		}
		fmt.Fprintf(&b, "  \n**Roles:** %s\n", roles(s.Roles))
		if len(s.Repair) > 0 {
			b.WriteString("\n**Repair required:**\n")
			for _, r := range s.Repair {
				fmt.Fprintf(&b, "- %s\n", r)
			}
		}
	}
	b.WriteString("\n## Events\n")
	for _, e := range evs {
		head := fmt.Sprintf("\n### e%d · %s", e.Seq, e.Kind)
		if e.Verdict != "" {
			head += " " + e.Verdict
		}
		fmt.Fprintf(&b, "%s · %s (%s) · %s\n\n", head, e.Actor, e.Role, e.At.UTC().Format(time.RFC3339))
		raw, _ := json.MarshalIndent(e, "", "  ")
		fmt.Fprintf(&b, "```json\n%s\n```\n", raw)
	}
	return b.String()
}

func roles(m map[Role]string) string {
	var out []string
	for _, r := range []Role{Coordinator, Implementer, Auditor, Observer, Owner} {
		if m[r] != "" {
			out = append(out, fmt.Sprintf("%s=%s", r, m[r]))
		}
	}
	return strings.Join(out, " · ")
}

// ParseLedger extracts the events and proves the text is exactly their canonical
// rendering: any edit outside the generator (including to the summary) is detected.
func ParseLedger(id, text string) ([]Event, State, error) {
	var evs []Event
	rest := text
	for {
		i := strings.Index(rest, "```json\n")
		if i < 0 {
			break
		}
		rest = rest[i+len("```json\n"):]
		j := strings.Index(rest, "\n```")
		if j < 0 {
			return nil, State{}, fmt.Errorf("%w: %s: unterminated event block", ErrTampered, id)
		}
		var e Event
		if err := json.Unmarshal([]byte(rest[:j]), &e); err != nil {
			return nil, State{}, fmt.Errorf("%w: %s: %v", ErrTampered, id, err)
		}
		evs = append(evs, e)
		rest = rest[j:]
	}
	s, err := Replay(id, evs)
	if err != nil {
		return nil, State{}, fmt.Errorf("%w: %s: %v", ErrTampered, id, err)
	}
	if RenderLedger(id, evs) != text {
		return nil, State{}, fmt.Errorf("%w: %s: the file is not the canonical rendering of its events (edited by hand?)", ErrTampered, id)
	}
	return evs, s, nil
}

// RenderRegister derives the team register from the ledgers. Nobody edits it.
func RenderRegister(states []State) string {
	sort.Slice(states, func(i, j int) bool { return states[i].ID < states[j].ID })
	var b strings.Builder
	b.WriteString("# Team register (derived)\n\n> Generated from `team/ws/*.md`. Do not edit: the next transition re-renders it. " +
		"PASS means an independent audit passed one exact revision; it is **not** owner approval, merge, deployment or a grant.\n\n")
	b.WriteString("| Workstream | Task | Phase | Exact candidate | Implementer | Auditor | Next |\n| --- | --- | --- | --- | --- | --- | --- |\n")
	for _, s := range states {
		fmt.Fprintf(&b, "| %s | %s | `%s` | %s | %s | %s | %s |\n", s.ID, s.Task, s.Phase, code(s.Commit),
			s.Roles[Implementer], s.Roles[Auditor], next(s))
	}
	return b.String()
}

func code(c string) string {
	if c == "" {
		return "—"
	}
	return "`" + short(c) + "`"
}

func next(s State) string {
	switch s.Phase {
	case StAssigned:
		return s.Roles[Implementer] + " builds a candidate"
	case StAwaitingAudit:
		return s.Roles[Auditor] + " audits " + short(s.Commit)
	case StAuditFailed:
		return s.Roles[Implementer] + " repairs (see ledger)"
	case StAuditPassed:
		return s.Roles[Owner] + " decides (approval is separate)"
	case StApproved:
		return "owner approved " + short(s.Commit) + "; merge/deploy still separate"
	}
	return ""
}

// --- policy, privacy and targets -----------------------------------------------------------

// Policy is local configuration (the stand-in for Identity). It is the ONLY source of
// write permission; nothing read from GitHub can change it.
type Policy struct {
	Writes map[string][]string // actor -> path patterns it may write (path.Match)
}

// MayWrite reports whether actor may write p.
func (p Policy) MayWrite(actor, p2 string) bool {
	for _, pat := range p.Writes[actor] {
		if ok, _ := path.Match(pat, p2); ok {
			return true
		}
	}
	return false
}

var secretRE = regexp.MustCompile(`(ghp_[A-Za-z0-9]{20,}|github_pat_[A-Za-z0-9_]{20,}|-----BEGIN [A-Z ]*PRIVATE KEY-----|(?i)bearer\s+[A-Za-z0-9._-]{20,})`)

// Publishable refuses text containing a secret-shaped string or any private snippet.
func Publishable(text string, private []string) error {
	if secretRE.MatchString(text) {
		return fmt.Errorf("%w: secret-shaped string", ErrPrivate)
	}
	for _, p := range private {
		if p != "" && strings.Contains(text, p) {
			return fmt.Errorf("%w: private memory", ErrPrivate)
		}
	}
	return nil
}

// Target is where the team mailbox lives. It must not overlap Core's identity/lesson
// sync (by default live:tass.md and live:lessons/).
type Target struct{ Repo, Branch string }

func (t Target) check(p string) error {
	if t.Branch == "live" || p == "tass.md" || p == "tass.base.md" || strings.HasPrefix(p, "lessons/") || strings.Contains(p, "..") {
		return fmt.Errorf("%w: %s@%s:%s", ErrTarget, t.Repo, t.Branch, p)
	}
	return nil
}

// --- the team on GitHub ------------------------------------------------------------------------

// Paths in the team mailbox.
const (
	RulesPath    = "AGENTS.md"
	RegisterPath = "team/REGISTER.md"
	TASSPath     = "team/TASS.md"
)

// LedgerPath is a workstream's ledger.
func LedgerPath(id string) string { return "team/ws/" + id + ".md" }

// Team is one participant's view of the shared mailbox. Each participant has its own
// GitHub client (its own token) and its own conditional-GET cache.
type Team struct {
	GH      *github.Client
	Target  Target
	Policy  Policy
	Actor   string
	Private []string // this participant's secrets and private memory: never published
	Now     func() time.Time
	IDs     func() []string // the workstreams this participant knows about

	mu    sync.Mutex
	etags map[string]string
	cache map[string]*github.File
	seen  map[string][]string // ledger -> event hashes this participant has already verified
	Stats Stats
}

// Stats counts what this participant did (for the tests and the demo transcript).
type Stats struct{ Reads, NotModified, Writes, Conflicts int }

func (t *Team) init() {
	if t.etags == nil {
		t.etags, t.cache, t.seen = map[string]string{}, map[string]*github.File{}, map[string][]string{}
	}
}

// Read is a conditional GET: an unchanged file costs a 304 and returns changed=false.
func (t *Team) Read(p string) (f *github.File, changed bool, err error) {
	if err := t.Target.check(p); err != nil {
		return nil, false, err
	}
	t.mu.Lock()
	t.init()
	etag := t.etags[p]
	t.mu.Unlock()
	nf, got, err := t.GH.GetFile(t.Target.Repo, t.Target.Branch, p, etag)
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Stats.Reads++
	if err != nil {
		return nil, false, err
	}
	if got.NotModified {
		t.Stats.NotModified++
		return t.cache[p], false, nil
	}
	t.etags[p], t.cache[p] = got.ETag, nf
	return nf, true, nil
}

// fresh reads p without the cache (for a write that must be based on the latest blob).
func (t *Team) fresh(p string) (*github.File, error) {
	t.mu.Lock()
	t.init()
	delete(t.etags, p)
	t.mu.Unlock()
	f, _, err := t.Read(p)
	return f, err
}

// put is the single write path: target check, local authorization, privacy, then a
// sha-conditional PUT (a stale sha is a conflict, never an overwrite).
func (t *Team) put(p, text, sha, msg string) (string, error) {
	if err := t.Target.check(p); err != nil {
		return "", err
	}
	if !t.Policy.MayWrite(t.Actor, p) {
		return "", fmt.Errorf("%w: %s may not write %s", ErrUnauthorized, t.Actor, p)
	}
	if err := Publishable(text+"\n"+msg, t.Private); err != nil {
		return "", err
	}
	s, err := t.GH.PutFile(t.Target.Repo, t.Target.Branch, p, text, sha, msg)
	t.mu.Lock()
	defer t.mu.Unlock()
	if errors.Is(err, github.ErrConflict) {
		t.Stats.Conflicts++
	} else if err == nil {
		t.Stats.Writes++
	}
	return s, err
}

// Rules reads the project instructions; every transition records the revision it read.
func (t *Team) Rules() (string, error) {
	f, _, err := t.Read(RulesPath)
	if err != nil || f == nil {
		return "", fmt.Errorf("workloop: project instructions (%s) unreadable: %v", RulesPath, err)
	}
	return f.SHA, nil
}

// Ledger reads and verifies one workstream. Beyond the chain and the state machine, the
// ledger must extend what this participant already saw: a rewritten or removed event it
// had verified (even a fully re-rendered forgery of the newest event) fails closed.
func (t *Team) Ledger(id string) ([]Event, State, *github.File, error) {
	f, _, err := t.Read(LedgerPath(id))
	if err != nil {
		return nil, State{}, nil, err
	}
	evs, s, err := ParseLedger(id, f.Text)
	if err != nil {
		return nil, State{}, nil, err
	}
	return evs, s, f, t.extends(id, evs)
}

func (t *Team) extends(id string, evs []Event) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.init()
	seen := t.seen[id]
	if len(evs) < len(seen) {
		return fmt.Errorf("%w: %s: %d events, but %s already verified %d (history removed)", ErrTampered, id, len(evs), t.Actor, len(seen))
	}
	for i, h := range seen {
		if evs[i].hash() != h {
			return fmt.Errorf("%w: %s: e%d differs from the version %s already verified (history rewritten)", ErrTampered, id, i+1, t.Actor)
		}
	}
	for i := len(seen); i < len(evs); i++ {
		t.seen[id] = append(t.seen[id], evs[i].hash())
	}
	return nil
}

const maxTries = 8

// Append adds one transition to a workstream. On a concurrent write it re-reads the
// latest ledger and re-validates the transition against it: an append that is no
// longer valid (e.g. auditing a candidate that was just replaced) is refused, never
// written over the newer work. Then the register is re-derived.
func (t *Team) Append(id string, e Event) (State, error) {
	p := LedgerPath(id)
	if !t.Policy.MayWrite(t.Actor, p) {
		return State{}, fmt.Errorf("%w: %s may not write %s", ErrUnauthorized, t.Actor, p)
	}
	for try := 0; try < maxTries; try++ {
		rules, err := t.Rules()
		if err != nil {
			return State{}, err
		}
		var evs []Event
		var s State
		sha := ""
		if f, err := t.fresh(p); errors.Is(err, github.ErrNotFound) {
			s = State{ID: id}
		} else if err != nil {
			return State{}, err
		} else if evs, s, err = ParseLedger(id, f.Text); err != nil {
			return State{}, err
		} else if err = t.extends(id, evs); err != nil {
			return State{}, err
		} else {
			sha = f.SHA
		}
		e.Seq, e.Prev, e.Actor, e.ReadRules = s.Last.Seq+1, s.Head, t.Actor, rules
		e.At = t.Now().UTC().Truncate(time.Second)
		ns, err := Apply(s, e)
		if err != nil {
			return s, err
		}
		_, err = t.put(p, RenderLedger(id, append(evs, e)), sha, fmt.Sprintf("workloop: %s e%d %s %s", id, e.Seq, e.Kind, e.Verdict))
		if errors.Is(err, github.ErrConflict) {
			continue
		}
		if err != nil {
			return s, err
		}
		if err := t.extends(id, append(evs, e)); err != nil {
			return ns, err
		}
		return ns, t.SyncRegister()
	}
	return State{}, fmt.Errorf("workloop: %s: gave up after %d conflicting writes", id, maxTries)
}

// SyncRegister re-derives team/REGISTER.md. Unchanged → no write (no commit per pulse);
// a conflicting writer → re-derive from the newer ledgers and try again.
func (t *Team) SyncRegister() error {
	if !t.Policy.MayWrite(t.Actor, RegisterPath) {
		return nil // e.g. TASS: reads the register, never writes it
	}
	for try := 0; try < maxTries; try++ {
		var states []State
		for _, id := range t.IDs() {
			_, s, _, err := t.Ledger(id)
			if errors.Is(err, github.ErrNotFound) {
				continue
			}
			if err != nil {
				return err
			}
			states = append(states, s)
		}
		want := RenderRegister(states)
		sha := ""
		cur, err := t.fresh(RegisterPath)
		if err == nil {
			if cur.Text == want {
				return nil
			}
			sha = cur.SHA
		} else if !errors.Is(err, github.ErrNotFound) {
			return err
		}
		_, err = t.put(RegisterPath, want, sha, "workloop: re-derive register")
		if errors.Is(err, github.ErrConflict) {
			continue
		}
		return err
	}
	return fmt.Errorf("workloop: register: gave up after %d conflicting writes", maxTries)
}

// Pulse is one 20-second tick for a participant: conditional reads of the rules and of
// the given paths. changed=false means every read was a 304; the caller does nothing.
func (t *Team) Pulse(paths ...string) (changed bool, err error) {
	for _, p := range append([]string{RulesPath}, paths...) {
		_, c, err := t.Read(p)
		if errors.Is(err, github.ErrNotFound) {
			continue
		}
		if err != nil {
			return changed, err
		}
		changed = changed || c
	}
	return changed, nil
}
