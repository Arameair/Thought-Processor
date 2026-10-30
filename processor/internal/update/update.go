// Package update holds the component-independent rules of Thought
// Processor's update system: which candidate is a reviewed, compatible
// update for an installed component; whether the component can be updated
// in place at all; the user-visible states; and the entitlement hook.
//
// It decides nothing about permissions. Identity authorizes every update
// (an Admin approval of kind upgrade_feature); an entitlement — the future
// home of a separately authorized payment decision — can only ever REFUSE an
// update that Identity would allow, never allow one Identity refuses, and is
// never consulted for a security fix.
package update

import (
	"strconv"
	"strings"
)

// User-visible states.
const (
	Current          = "CURRENT"
	Available        = "UPDATE_AVAILABLE"
	ApprovalRequired = "APPROVAL_REQUIRED"
	Updating         = "UPDATING"
	Restarting       = "RESTARTING"
	Pass             = "PASS"
	Fail             = "FAIL"
	Recovering       = "RECOVERING" // not final: the previous version is being restored
	Recovered        = "RECOVERED_PREVIOUS_VERSION"
	Blocked          = "BLOCKED"
	Unsupported      = "UNSUPPORTED"
	NeedsAttention   = "NEEDS_ATTENTION"
	Denied           = "DENIED_BY_ADMIN"
	Expired          = "APPROVAL_EXPIRED"
	Cancelled        = "CANCELLED"
)

// Terminal reports whether a transaction state is final.
func Terminal(s string) bool {
	switch s {
	case Pass, Fail, Recovered, NeedsAttention, Denied, Expired, Cancelled, Blocked:
		return true
	}
	return false
}

// CompareVersions compares dotted numeric versions ("0.10.2" > "0.9").
// A pre-release suffix ("-rc1") sorts before the plain version. ok=false if
// either side isn't of that shape — then no ordering is assumed.
func CompareVersions(a, b string) (int, bool) {
	pa, preA, okA := parse(a)
	pb, preB, okB := parse(b)
	if !okA || !okB {
		return 0, false
	}
	for i := 0; i < len(pa) || i < len(pb); i++ {
		x, y := 0, 0
		if i < len(pa) {
			x = pa[i]
		}
		if i < len(pb) {
			y = pb[i]
		}
		if x != y {
			if x < y {
				return -1, true
			}
			return 1, true
		}
	}
	switch {
	case preA == preB:
		return 0, true
	case preA == "":
		return 1, true
	case preB == "":
		return -1, true
	case preA < preB:
		return -1, true
	}
	return 1, true
}

func parse(v string) ([]int, string, bool) {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	pre := ""
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		v, pre = v[:i], v[i+1:]
	}
	parts := strings.Split(v, ".")
	if len(parts) == 0 || len(parts) > 4 {
		return nil, "", false
	}
	out := make([]int, len(parts))
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return nil, "", false
		}
		out[i] = n
	}
	return out, pre, true
}

// Installed is what the update rules need about the installed component.
type Installed struct {
	FeatureID string
	Version   string
	Artifact  string
	API       string
}

// Candidate is one package that might update it. Reviewed = its exact
// artifact is pinned in this Processor (the existing trust rule).
type Candidate struct {
	Dir            string
	FeatureID      string
	Version        string
	Artifact       string
	API            string
	Reviewed       bool
	ReviewNote     string
	Declared       bool // declares thought.feature-update/1
	MinFromVersion string
	DataMigration  string
	SecurityFix    bool
}

// Verdict is the decision about one candidate.
type Verdict struct {
	Candidate Candidate `json:"-"`
	State     string    `json:"state"` // Available | Blocked | Unsupported | "" (not an update: same or older)
	Reason    string    `json:"reason,omitempty"`
}

// Judge decides whether c can update in. identityAPI is the upgrade API
// Identity advertises ("" when the paired Identity has none).
func Judge(in Installed, c Candidate, identityAPI string) Verdict {
	v := Verdict{Candidate: c}
	if c.FeatureID != in.FeatureID || c.Artifact == in.Artifact {
		return v
	}
	cmp, ok := CompareVersions(c.Version, in.Version)
	switch {
	case !ok:
		v.State, v.Reason = Blocked, "VERSION_NOT_COMPARABLE: "+c.Version+" vs installed "+in.Version
		return v
	case cmp < 0:
		return v // older: never offered as an update
	case cmp == 0:
		v.State, v.Reason = Blocked, "UNEXPECTED_REVISION: same version "+c.Version+" with different content than the installed artifact"
		return v
	}
	switch {
	case !c.Reviewed:
		v.State, v.Reason = Blocked, "NOT_REVIEWED: artifact "+short(c.Artifact)+" is not pinned in this Processor"+note(c.ReviewNote)
	case c.API != in.API:
		v.State, v.Reason = Blocked, "INCOMPATIBLE_API: "+c.API+" (installed speaks "+in.API+")"
	case !c.Declared:
		v.State, v.Reason = Unsupported, "NO_UPDATE_CONTRACT: this version does not declare thought.feature-update/1; it can only be installed as a new copy"
	case c.DataMigration != "none" && c.DataMigration != "reversible":
		v.State, v.Reason = Blocked, "IRREVERSIBLE_MIGRATION: recovery requirements for this update are not satisfied"
	case c.MinFromVersion != "" && func() bool { x, ok := CompareVersions(in.Version, c.MinFromVersion); return !ok || x < 0 }():
		v.State, v.Reason = Blocked, "TOO_OLD: this update needs installed version "+c.MinFromVersion+" or newer"
	case identityAPI != "identity.installation-upgrade/1":
		v.State, v.Reason = Blocked, "IDENTITY_UPGRADE_API_MISSING: the paired Identity cannot authorize in-place upgrades (needs identity.installation-upgrade/1, owner-approved)"
	default:
		v.State = Available
	}
	return v
}

// Best picks the highest-version Available verdict, else the most relevant
// blocked/unsupported one (highest version), else Current.
func Best(vs []Verdict) (Verdict, bool) {
	var best *Verdict
	rank := map[string]int{Available: 3, Blocked: 2, Unsupported: 1}
	for i := range vs {
		v := &vs[i]
		if rank[v.State] == 0 {
			continue
		}
		if best == nil || rank[v.State] > rank[best.State] {
			best = v
			continue
		}
		if rank[v.State] == rank[best.State] {
			if c, ok := CompareVersions(v.Candidate.Version, best.Candidate.Version); ok && c > 0 {
				best = v
			}
		}
	}
	if best == nil {
		return Verdict{State: Current}, false
	}
	return *best, true
}

func short(s string) string {
	if len(s) > 12 {
		return s[:12] + "…"
	}
	return s
}

func note(s string) string {
	if s == "" {
		return ""
	}
	return " (" + s + ")"
}

// Entitlement is the hook for a future, separately authorized entitlement
// (for example a TASS Coin decision). It is consulted only AFTER the update
// is otherwise eligible and BEFORE asking Identity; it can refuse, never
// authorize. Security fixes skip it entirely.
type Entitlement interface {
	Decide(featureID, fromVersion, toVersion string) EntitlementDecision
}

type EntitlementDecision struct {
	State  string `json:"state"` // NOT_REQUIRED | GRANTED | REFUSED | UNAVAILABLE
	Source string `json:"source"`
	Detail string `json:"detail,omitempty"`
}

// Allows reports whether the decision lets the update continue to Identity.
func (d EntitlementDecision) Allows() bool { return d.State == "NOT_REQUIRED" || d.State == "GRANTED" }

// NoEntitlements is the only implementation today: nothing is charged.
type NoEntitlements struct{}

func (NoEntitlements) Decide(string, string, string) EntitlementDecision {
	return EntitlementDecision{State: "NOT_REQUIRED", Source: "no entitlement system configured (no payment is ever taken)"}
}

// EntitlementFor applies the security-fix rule around any Entitlement.
func EntitlementFor(e Entitlement, securityFix bool, featureID, from, to string) EntitlementDecision {
	if securityFix {
		return EntitlementDecision{State: "NOT_REQUIRED", Source: "security fix: never subject to an entitlement"}
	}
	if e == nil {
		e = NoEntitlements{}
	}
	return e.Decide(featureID, from, to)
}
