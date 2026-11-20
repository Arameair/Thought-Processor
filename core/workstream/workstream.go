// Package workstream is the Stage 2 / S2-A contract for one development workstream
// (Plan PR #133 §4): a typed event format carried in the owning PR's comments, a
// strict parser, and a pure evaluator that separates what GitHub verified from what
// a comment merely claims.
//
// Trust model in one paragraph: everything inside a comment body is an UNTRUSTED
// CLAIM, including the actor name, the verdict and any text that looks like an
// approval. GitHub-verified facts are the comment ID, author login, timestamps and
// the PR head SHA as returned by the API. Because Claude, Codex and GPT-6 all post
// through the owner's GitHub account, a login does not identify an assistant. A
// claim changes workstream state only when an owner-held Trust input that does NOT
// come from GitHub either (a) maps a dedicated, unshared login to that actor, or
// (b) attests that exact comment (ID + body digest) to that actor and transition.
// Role assignments also come only from Trust. Nothing here writes to GitHub,
// launches agents, or grants permissions; S2-B transport is separately gated.
package workstream

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Fence opens an event block inside a PR comment. A comment that mentions the
// contract name must contain exactly one well-formed block, or it fails closed.
const (
	Fence        = "```tass-workstream-v1"
	contractName = "tass-workstream"
	maxBlock     = 16 << 10
	maxEvidence  = 4 << 10
)

// Kind is a transition type.
type Kind string

const (
	Candidate   Kind = "CANDIDATE"    // implementer submits an exact revision with test evidence
	Audit       Kind = "AUDIT"        // independent auditor's verdict on that exact revision
	OwnerAccept Kind = "OWNER_ACCEPT" // owner accepts the exact revision that passed audit
)

// Verdict of an AUDIT event (the repository's PASS / NEEDS REPAIR / BLOCKED vocabulary).
type Verdict string

const (
	Pass        Verdict = "PASS"
	NeedsRepair Verdict = "NEEDS_REPAIR"
	Blocked     Verdict = "BLOCKED"
)

// Phase is derived on every evaluation; it is never stored as truth.
type Phase string

const (
	PhaseAssigned      Phase = "ASSIGNED"
	PhaseAwaitingAudit Phase = "AWAITING_AUDIT"
	PhaseNeedsRepair   Phase = "NEEDS_REPAIR"
	PhaseAuditBlocked  Phase = "AUDIT_BLOCKED"
	PhaseAuditPassed   Phase = "AUDIT_PASSED" // still needs the owner
	PhaseOwnerAccepted Phase = "OWNER_ACCEPTED"
	// PhaseStale: the PR head is not the latest candidate, so no verdict applies to it.
	PhaseStale Phase = "STALE_CANDIDATE"
	// PhaseBlocked: an integrity failure needs owner reconciliation before any state is trusted.
	PhaseBlocked Phase = "BLOCKED"
)

// Provenance says how an event's actor was established.
type Provenance string

const (
	Unverified    Provenance = "UNVERIFIED"     // only the comment text names the actor
	Principal     Provenance = "PRINCIPAL"      // dedicated, unshared GitHub login mapped in Trust
	OwnerAttested Provenance = "OWNER_ATTESTED" // owner-held attestation of this exact comment
)

// Reason codes. Integrity reasons block the workstream; the others reject one claim.
type Reason string

const (
	// Integrity (workstream BLOCKED).
	RMalformed        Reason = "MALFORMED"
	RModified         Reason = "MODIFIED"
	RReplayed         Reason = "REPLAYED"
	RForeign          Reason = "FOREIGN_EVENT"
	RAttestedMissing  Reason = "ATTESTED_EVIDENCE_MISSING"
	RIncomplete       Reason = "INCOMPLETE_OBSERVATION"
	RObservationScope Reason = "OBSERVATION_SCOPE"
	RTrustConfig      Reason = "TRUST_CONFIG"
	// Claim rejected (state does not advance).
	RUnverified      Reason = "UNVERIFIED_ACTOR"
	RUnknownActor    Reason = "UNKNOWN_ACTOR"
	RForgedActor     Reason = "FORGED_ACTOR"
	RSelfReview      Reason = "SELF_REVIEW"
	RWrongRole       Reason = "WRONG_ROLE"
	RStaleScope      Reason = "STALE_SCOPE"
	RStaleCandidate  Reason = "STALE_CANDIDATE"
	ROutOfOrder      Reason = "OUT_OF_ORDER"
	RResubmitted     Reason = "RESUBMITTED_FAILED_REVISION"
	RNotOwnerVerfied Reason = "OWNER_NOT_VERIFIED"
)

// Event is the untrusted body of one transition. Unknown fields are rejected.
type Event struct {
	V          int     `json:"v"`
	ID         string  `json:"event_id"`
	Workstream string  `json:"workstream"`
	Repo       string  `json:"repo"`
	PR         int     `json:"pr"`
	Scope      string  `json:"scope"`
	Kind       Kind    `json:"kind"`
	Actor      string  `json:"actor"`
	Commit     string  `json:"commit"`
	Verdict    Verdict `json:"verdict,omitempty"`
	Evidence   string  `json:"evidence,omitempty"`
}

var (
	commitRE = regexp.MustCompile(`^[0-9a-f]{40}$`)
	idRE     = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)
	digestRE = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

// ErrMalformed wraps every parse failure.
var ErrMalformed = errors.New("workstream: malformed event")

// Parse extracts the event from a comment body. ok=false with a nil error means the
// comment is ordinary discussion (it never mentions the contract). Any mention of the
// contract that is not exactly one well-formed block is an error, never ignored.
func Parse(body string) (ev Event, ok bool, err error) {
	if !strings.Contains(body, contractName) {
		return Event{}, false, nil
	}
	bad := func(f string, a ...any) (Event, bool, error) {
		return Event{}, true, fmt.Errorf("%w: %s", ErrMalformed, fmt.Sprintf(f, a...))
	}
	if strings.Count(body, contractName) != 1 {
		return bad("the contract is named more than once; exactly one event block per comment")
	}
	lines := strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n")
	start, end := -1, -1
	for i, l := range lines {
		if l == Fence {
			start = i
			break
		}
	}
	if start < 0 {
		return bad("the contract is mentioned without an exact %q opening line", Fence)
	}
	for i := start + 1; i < len(lines); i++ {
		if lines[i] == "```" {
			end = i
			break
		}
	}
	if end < 0 {
		return bad("unterminated event block")
	}
	raw := strings.Join(lines[start+1:end], "\n")
	if len(raw) > maxBlock {
		return bad("event block exceeds %d bytes", maxBlock)
	}
	dec := json.NewDecoder(bytes.NewReader([]byte(raw)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&ev); err != nil {
		return bad("%v", err)
	}
	if _, err := dec.Token(); err != io.EOF {
		return bad("trailing data after the event object")
	}
	if err := ev.validate(); err != nil {
		return bad("%v", err)
	}
	return ev, true, nil
}

func (e Event) validate() error {
	switch {
	case e.V != 1:
		return fmt.Errorf("unsupported contract version %d", e.V)
	case !idRE.MatchString(e.ID):
		return fmt.Errorf("bad event_id")
	case !idRE.MatchString(e.Workstream):
		return fmt.Errorf("bad workstream")
	case e.Repo == "" || strings.Count(e.Repo, "/") != 1 || e.PR <= 0:
		return fmt.Errorf("repo and pr are required")
	case !idRE.MatchString(e.Scope):
		return fmt.Errorf("bad scope")
	case !idRE.MatchString(e.Actor):
		return fmt.Errorf("bad actor")
	case !commitRE.MatchString(e.Commit):
		return fmt.Errorf("commit must be an exact lowercase 40-hex revision")
	case len(e.Evidence) > maxEvidence:
		return fmt.Errorf("evidence exceeds %d bytes", maxEvidence)
	}
	switch e.Kind {
	case Candidate, Audit:
		if strings.TrimSpace(e.Evidence) == "" {
			return fmt.Errorf("%s needs test evidence", e.Kind)
		}
	case OwnerAccept:
	default:
		return fmt.Errorf("unknown kind %q", e.Kind)
	}
	if e.Kind == Audit {
		if e.Verdict != Pass && e.Verdict != NeedsRepair && e.Verdict != Blocked {
			return fmt.Errorf("verdict must be PASS, NEEDS_REPAIR or BLOCKED")
		}
	} else if e.Verdict != "" {
		return fmt.Errorf("only AUDIT carries a verdict")
	}
	return nil
}

// BodyDigest is the SHA-256 of a comment body exactly as GitHub returned it. An owner
// attestation pins this value, so any later edit is detected.
func BodyDigest(body string) string {
	s := sha256.Sum256([]byte(body))
	return hex.EncodeToString(s[:])
}

// ---- trusted, owner-held inputs (never read from GitHub) ------------------------------

// Binding is the owner's assignment of one workstream to its owning PR and roles.
type Binding struct {
	Workstream  string
	Repo        string // owner/name of the owning PR
	PR          int
	Scope       string // current owner-authorized task scope; events citing another scope are stale
	Implementer string
	Auditor     string
	Owner       string
}

// Attestation is the owner's record that one exact comment was written by Actor as
// that transition. It is supplied locally; a comment claiming to be an attestation
// is just another untrusted claim.
type Attestation struct {
	CommentID  int64
	BodySHA256 string
	Actor      string
	Kind       Kind
	Commit     string
}

// Trust is everything the evaluator is allowed to believe that GitHub cannot verify.
// How the owner produces and protects it is an S2-B decision (see README).
type Trust struct {
	Binding Binding
	// Principals maps a dedicated GitHub login to the one actor that alone uses it.
	Principals map[string]string
	// SharedLogins are logins used by more than one participant (e.g. the owner's
	// account, through which all assistants post). They never authenticate an actor.
	SharedLogins []string
	Attestations []Attestation
}

// loginRE is a GitHub login: alphanumerics and single inner hyphens, at most 39
// characters, optionally a GitHub App bot suffix. Anything else in Trust is invalid
// configuration, never something to trim or guess at.
var loginRE = regexp.MustCompile(`^[A-Za-z0-9](?:[A-Za-z0-9-]{0,37}[A-Za-z0-9])?(?:\[bot\])?$`)

func validLogin(l string) bool { return loginRE.MatchString(l) && !strings.Contains(l, "--") }

// trusted is the validated, normalized form of Trust that Evaluate uses. Logins are
// keyed in lower case exactly once, here; GitHub logins are case-insensitive.
type trusted struct {
	principals map[string]string // normalized login -> actor
	shared     map[string]bool   // normalized login
}

// normalize validates t and returns the only login mappings Evaluate may consult.
// Every malformed login, every case-insensitive duplicate principal (even one naming
// the same actor) and every shared/principal overlap is a TRUST_CONFIG error, so a
// conflicting binding can never be resolved by map iteration order. Keys are visited
// in sorted order so the reported error is deterministic too.
func (t Trust) normalize() (trusted, error) {
	b := t.Binding
	if !idRE.MatchString(b.Workstream) || b.Repo == "" || b.PR <= 0 || !idRE.MatchString(b.Scope) {
		return trusted{}, fmt.Errorf("binding needs workstream, repo, pr and scope")
	}
	if b.Implementer == "" || b.Auditor == "" || b.Owner == "" {
		return trusted{}, fmt.Errorf("binding needs an implementer, an auditor and an owner")
	}
	if b.Implementer == b.Auditor || b.Implementer == b.Owner || b.Auditor == b.Owner {
		return trusted{}, fmt.Errorf("implementer, auditor and owner must be distinct actors")
	}
	tr := trusted{principals: map[string]string{}, shared: map[string]bool{}}
	for _, l := range t.SharedLogins {
		if !validLogin(l) {
			return trusted{}, fmt.Errorf("shared login %q is not a valid GitHub login", l)
		}
		tr.shared[strings.ToLower(l)] = true // a set: repeating a shared login is harmless
	}
	keys := make([]string, 0, len(t.Principals))
	for l := range t.Principals {
		keys = append(keys, l)
	}
	sort.Strings(keys)
	original := map[string]string{} // normalized -> first literal key seen
	for _, l := range keys {
		a := t.Principals[l]
		if !validLogin(l) {
			return trusted{}, fmt.Errorf("principal login %q is not a valid GitHub login", l)
		}
		n := strings.ToLower(l)
		if prev, dup := original[n]; dup {
			return trusted{}, fmt.Errorf("principal logins %q and %q are the same GitHub login", prev, l)
		}
		original[n] = l
		if tr.shared[n] {
			return trusted{}, fmt.Errorf("login %q is shared and cannot be a principal", l)
		}
		if a == "" {
			return trusted{}, fmt.Errorf("principal %q maps to no actor", l)
		}
		tr.principals[n] = a
	}
	seen := map[int64]bool{}
	for _, a := range t.Attestations {
		if a.CommentID <= 0 || !digestRE.MatchString(a.BodySHA256) || a.Actor == "" || a.Kind == "" || !commitRE.MatchString(a.Commit) {
			return trusted{}, fmt.Errorf("attestation for comment %d is incomplete", a.CommentID)
		}
		if seen[a.CommentID] {
			return trusted{}, fmt.Errorf("comment %d is attested twice", a.CommentID)
		}
		seen[a.CommentID] = true
	}
	return tr, nil
}

// ---- GitHub-verified observation ---------------------------------------------------------

// Comment is one PR comment as returned by the GitHub API.
type Comment struct {
	ID        int64
	Login     string
	CreatedAt time.Time
	UpdatedAt time.Time
	Body      string
}

// Observation is one read of the owning PR. Complete must be true only when every
// page of comments was read and the head did not move during the read.
type Observation struct {
	Repo     string
	PR       int
	HeadSHA  string
	Comments []Comment
	Complete bool
}

// Verified is what GitHub, not the comment text, established about an event.
type Verified struct {
	Repo       string
	PR         int
	CommentID  int64
	Login      string
	CreatedAt  time.Time
	BodySHA256 string
}

// Record is one evaluated comment: the verified facts, the untrusted claim, how the
// actor was established and, for a rejection, why.
type Record struct {
	Verified   Verified
	Claim      *Event
	Provenance Provenance
	Transition string // BUILD, REPAIR, AUDIT, RE-AUDIT, OWNER_ACCEPT (accepted records)
	Reason     Reason
	Detail     string
}

// Report is the derived workstream state.
type Report struct {
	Workstream string
	Phase      Phase
	Candidate  string // latest accepted candidate
	HeadSHA    string // PR head at observation
	Current    bool   // Candidate == HeadSHA and nothing blocked
	Accepted   []Record
	Rejected   []Record // claims that did not change state
	Blocked    []Record // integrity failures; Phase is BLOCKED when non-empty
}

type state struct {
	phase       Phase
	candidate   string
	failed      map[string]bool
	repairing   bool // latest candidate follows a NEEDS_REPAIR
	audited     bool // latest candidate has been audited at least once
	afterRepair bool
}

// Evaluate derives the workstream state from one observation and the owner-held
// trust input. It is pure: same inputs, same report.
func Evaluate(t Trust, obs Observation) Report {
	b := t.Binding
	r := Report{Workstream: b.Workstream, HeadSHA: obs.HeadSHA, Phase: PhaseAssigned}
	block := func(v Verified, claim *Event, reason Reason, f string, a ...any) {
		r.Blocked = append(r.Blocked, Record{Verified: v, Claim: claim, Reason: reason, Detail: fmt.Sprintf(f, a...)})
	}
	tc, err := t.normalize()
	if err != nil {
		block(Verified{}, nil, RTrustConfig, "%v", err)
		r.Phase = PhaseBlocked
		return r
	}
	if !strings.EqualFold(obs.Repo, b.Repo) || obs.PR != b.PR {
		block(Verified{Repo: obs.Repo, PR: obs.PR}, nil, RObservationScope, "observed %s#%d but the workstream is bound to %s#%d", obs.Repo, obs.PR, b.Repo, b.PR)
	}
	if !obs.Complete {
		block(Verified{Repo: obs.Repo, PR: obs.PR}, nil, RIncomplete, "comment enumeration or head read was incomplete")
	}
	if !commitRE.MatchString(obs.HeadSHA) {
		block(Verified{Repo: obs.Repo, PR: obs.PR}, nil, RIncomplete, "PR head is not an exact 40-hex revision")
	}

	att := map[int64]Attestation{}
	for _, a := range t.Attestations {
		att[a.CommentID] = a
	}
	shared, principals := tc.shared, tc.principals // validated and normalized once
	roles := map[string]bool{b.Implementer: true, b.Auditor: true, b.Owner: true}

	cs := append([]Comment(nil), obs.Comments...)
	sort.Slice(cs, func(i, j int) bool { return cs[i].ID < cs[j].ID })
	seenComment, seenEventID, attSeen := map[int64]bool{}, map[string]int64{}, map[int64]bool{}
	s := state{phase: PhaseAssigned, failed: map[string]bool{}}

	for _, c := range cs {
		v := Verified{Repo: obs.Repo, PR: obs.PR, CommentID: c.ID, Login: c.Login, CreatedAt: c.CreatedAt, BodySHA256: BodyDigest(c.Body)}
		if seenComment[c.ID] {
			block(v, nil, RReplayed, "comment %d observed twice", c.ID)
			continue
		}
		seenComment[c.ID] = true
		a, attested := att[c.ID]
		if attested {
			attSeen[c.ID] = true
			if a.BodySHA256 != v.BodySHA256 {
				block(v, nil, RModified, "comment %d no longer matches the body digest the owner attested", c.ID)
				continue
			}
		}
		ev, isEvent, err := Parse(c.Body)
		if !isEvent {
			if attested {
				block(v, nil, RModified, "attested comment %d no longer carries an event", c.ID)
			}
			continue // ordinary discussion, including text such as "Codex: PASS"
		}
		if err != nil {
			block(v, nil, RMalformed, "%v", err)
			continue
		}
		claim := ev
		if c.UpdatedAt.After(c.CreatedAt) {
			block(v, &claim, RModified, "event comment %d was edited after posting", c.ID)
			continue
		}
		if ev.Workstream != b.Workstream || !strings.EqualFold(ev.Repo, b.Repo) || ev.PR != b.PR {
			block(v, &claim, RForeign, "event names %s %s#%d; this PR is %s %s#%d", ev.Workstream, ev.Repo, ev.PR, b.Workstream, b.Repo, b.PR)
			continue
		}
		// An identical repost necessarily repeats event_id, so this also catches copies.
		if prev, dup := seenEventID[ev.ID]; dup {
			block(v, &claim, RReplayed, "event_id %q already used by comment %d", ev.ID, prev)
			continue
		}
		seenEventID[ev.ID] = c.ID

		reject := func(p Provenance, reason Reason, f string, a ...any) {
			r.Rejected = append(r.Rejected, Record{Verified: v, Claim: &claim, Provenance: p, Reason: reason, Detail: fmt.Sprintf(f, a...)})
		}
		if ev.Scope != b.Scope {
			reject(Unverified, RStaleScope, "event cites scope %q; the current authorized scope is %q", ev.Scope, b.Scope)
			continue
		}
		if !roles[ev.Actor] {
			reject(Unverified, RUnknownActor, "%q holds no role in this workstream", ev.Actor)
			continue
		}

		// Provenance: owner attestation first, then a dedicated principal; never the text.
		prov := Unverified
		switch who, isPrincipal := principals[strings.ToLower(c.Login)]; {
		case attested:
			if a.Actor != ev.Actor || a.Kind != ev.Kind || a.Commit != ev.Commit {
				reject(OwnerAttested, RForgedActor, "owner attested %s %s %s; the comment claims %s %s %s",
					a.Actor, a.Kind, short(a.Commit), ev.Actor, ev.Kind, short(ev.Commit))
				continue
			}
			prov = OwnerAttested
		case isPrincipal && !shared[strings.ToLower(c.Login)]:
			if who != ev.Actor {
				reject(Principal, RForgedActor, "login %s is %s's dedicated principal; the comment claims %s", c.Login, who, ev.Actor)
				continue
			}
			prov = Principal
		}
		if prov == Unverified {
			reason := RUnverified
			if ev.Kind == OwnerAccept {
				reason = RNotOwnerVerfied
			}
			reject(Unverified, reason, "login %s does not identify %s and no owner attestation covers comment %d", c.Login, ev.Actor, c.ID)
			continue
		}

		tr, reason, detail := s.apply(b, ev)
		if reason != "" {
			reject(prov, reason, "%s", detail)
			continue
		}
		r.Accepted = append(r.Accepted, Record{Verified: v, Claim: &claim, Provenance: prov, Transition: tr})
	}

	for id := range att {
		if !attSeen[id] {
			block(Verified{Repo: obs.Repo, PR: obs.PR, CommentID: id}, nil, RAttestedMissing, "owner-attested comment %d is missing from the PR (deleted or not enumerated)", id)
		}
	}

	r.Phase, r.Candidate = s.phase, s.candidate
	switch {
	case len(r.Blocked) > 0:
		r.Phase = PhaseBlocked
	case s.candidate != "" && s.candidate != obs.HeadSHA:
		r.Phase = PhaseStale
	default:
		r.Current = s.candidate != "" && s.candidate == obs.HeadSHA
	}
	return r
}

// apply is the transition function. It returns the transition label or a rejection.
func (s *state) apply(b Binding, e Event) (string, Reason, string) {
	switch e.Kind {
	case Candidate:
		if e.Actor != b.Implementer {
			return "", RWrongRole, fmt.Sprintf("only the implementer %s submits candidates", b.Implementer)
		}
		switch s.phase {
		case PhaseAssigned, PhaseNeedsRepair, PhaseAuditBlocked:
		default:
			return "", ROutOfOrder, fmt.Sprintf("a candidate cannot be submitted in phase %s", s.phase)
		}
		if s.failed[e.Commit] {
			return "", RResubmitted, fmt.Sprintf("%s already failed an audit; a repair needs a new revision", short(e.Commit))
		}
		if e.Commit == s.candidate {
			return "", ROutOfOrder, "this revision is already the candidate"
		}
		label := "BUILD"
		s.afterRepair = s.phase == PhaseNeedsRepair || (s.afterRepair && s.phase == PhaseAuditBlocked)
		if s.afterRepair {
			label = "REPAIR"
		}
		s.phase, s.candidate, s.audited = PhaseAwaitingAudit, e.Commit, false
		return label, "", ""
	case Audit:
		if e.Actor == b.Implementer {
			return "", RSelfReview, "the implementer cannot audit its own candidate"
		}
		if e.Actor != b.Auditor {
			return "", RWrongRole, fmt.Sprintf("only the assigned auditor %s audits", b.Auditor)
		}
		if s.phase != PhaseAwaitingAudit && s.phase != PhaseAuditBlocked {
			return "", ROutOfOrder, fmt.Sprintf("nothing is awaiting audit (phase %s)", s.phase)
		}
		if e.Commit != s.candidate {
			return "", RStaleCandidate, fmt.Sprintf("audited %s but the candidate is %s", short(e.Commit), short(s.candidate))
		}
		label := "AUDIT"
		if s.afterRepair || s.audited {
			label = "RE-AUDIT"
		}
		s.audited = true
		switch e.Verdict {
		case Pass:
			s.phase = PhaseAuditPassed
		case NeedsRepair:
			s.phase = PhaseNeedsRepair
			s.failed[e.Commit] = true
		case Blocked:
			s.phase = PhaseAuditBlocked
		}
		return label, "", ""
	case OwnerAccept:
		if e.Actor != b.Owner {
			return "", RWrongRole, "only the owner accepts"
		}
		if s.phase != PhaseAuditPassed {
			return "", ROutOfOrder, fmt.Sprintf("owner acceptance needs an independent PASS first (phase %s)", s.phase)
		}
		if e.Commit != s.candidate {
			return "", RStaleCandidate, fmt.Sprintf("accepted %s but the passed candidate is %s", short(e.Commit), short(s.candidate))
		}
		s.phase = PhaseOwnerAccepted
		return "OWNER_ACCEPT", "", ""
	}
	return "", RWrongRole, "unknown kind"
}

func short(c string) string {
	if len(c) > 12 {
		return c[:12]
	}
	return c
}
