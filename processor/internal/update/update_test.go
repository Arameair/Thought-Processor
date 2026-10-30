package update

import (
	"strings"
	"testing"
)

func TestCompareVersions(t *testing.T) {
	for _, c := range []struct {
		a, b string
		want int
		ok   bool
	}{
		{"0.2.0", "0.1.0", 1, true}, {"0.10.0", "0.9.9", 1, true}, {"1.0", "1.0.0", 0, true},
		{"1.0.0-rc1", "1.0.0", -1, true}, {"v2.0.0", "1.9", 1, true}, {"abc", "1.0", 0, false},
	} {
		got, ok := CompareVersions(c.a, c.b)
		if got != c.want || ok != c.ok {
			t.Errorf("%s vs %s = %d,%v want %d,%v", c.a, c.b, got, ok, c.want, c.ok)
		}
	}
}

func TestJudge(t *testing.T) {
	in := Installed{FeatureID: "w", Version: "0.1.0", Artifact: "a1", API: "api/1"}
	good := Candidate{FeatureID: "w", Version: "0.2.0", Artifact: "a2", API: "api/1", Reviewed: true, Declared: true, DataMigration: "none"}
	const api = "identity.installation-upgrade/1"
	if v := Judge(in, good, api); v.State != Available {
		t.Fatalf("good candidate: %+v", v)
	}
	cases := map[string]struct {
		mut  func(*Candidate) string
		want string
		why  string
	}{
		"unreviewed":      {func(c *Candidate) string { c.Reviewed = false; return api }, Blocked, "NOT_REVIEWED"},
		"no contract":     {func(c *Candidate) string { c.Declared = false; return api }, Unsupported, "NO_UPDATE_CONTRACT"},
		"irreversible":    {func(c *Candidate) string { c.DataMigration = "irreversible"; return api }, Blocked, "IRREVERSIBLE"},
		"other api":       {func(c *Candidate) string { c.API = "api/2"; return api }, Blocked, "INCOMPATIBLE_API"},
		"too old":         {func(c *Candidate) string { c.MinFromVersion = "0.1.5"; return api }, Blocked, "TOO_OLD"},
		"no identity api": {func(c *Candidate) string { return "" }, Blocked, "IDENTITY_UPGRADE_API_MISSING"},
		"same version":    {func(c *Candidate) string { c.Version = "0.1.0"; return api }, Blocked, "UNEXPECTED_REVISION"},
		"older":           {func(c *Candidate) string { c.Version = "0.0.9"; return api }, "", ""},
		"same artifact":   {func(c *Candidate) string { c.Artifact = "a1"; return api }, "", ""},
		"bad version":     {func(c *Candidate) string { c.Version = "next"; return api }, Blocked, "VERSION_NOT_COMPARABLE"},
	}
	for name, cs := range cases {
		c := good
		a := cs.mut(&c)
		v := Judge(in, c, a)
		if v.State != cs.want || !strings.Contains(v.Reason, cs.why) {
			t.Errorf("%s: got %s %q", name, v.State, v.Reason)
		}
	}
}

func TestRecoveringIsNotFinal(t *testing.T) {
	if Terminal(Recovering) || Terminal(Updating) || Terminal(ApprovalRequired) || !Terminal(Recovered) || !Terminal(Pass) {
		t.Fatal("terminal states wrong")
	}
}

func TestBestPrefersHighestAvailable(t *testing.T) {
	vs := []Verdict{
		{State: Blocked, Candidate: Candidate{Version: "0.9.0"}},
		{State: Available, Candidate: Candidate{Version: "0.2.0"}},
		{State: Available, Candidate: Candidate{Version: "0.3.0"}},
		{State: ""},
	}
	b, ok := Best(vs)
	if !ok || b.State != Available || b.Candidate.Version != "0.3.0" {
		t.Fatalf("best %+v", b)
	}
	if b, ok := Best([]Verdict{{State: ""}}); ok || b.State != Current {
		t.Fatal("no candidates should be CURRENT")
	}
}

type refuse struct{}

func (refuse) Decide(string, string, string) EntitlementDecision {
	return EntitlementDecision{State: "REFUSED", Source: "test"}
}

func TestEntitlementNeverBlocksSecurityFixes(t *testing.T) {
	if d := EntitlementFor(refuse{}, true, "w", "1", "2"); !d.Allows() {
		t.Fatal("security fix was blocked by an entitlement")
	}
	if d := EntitlementFor(refuse{}, false, "w", "1", "2"); d.Allows() {
		t.Fatal("refused entitlement allowed an update")
	}
	if d := EntitlementFor(nil, false, "w", "1", "2"); d.State != "NOT_REQUIRED" {
		t.Fatal("default must charge nothing")
	}
}
