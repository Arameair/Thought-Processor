package tass

// Stage 3B plan slice: saving an experiment plan makes zero model calls, sends the Engine nothing but GETs,
// never touches the Thought, is owner-scoped, idempotent, never synced, and no arm can run.

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

const expVariant = "Answer the Thought's question directly from the evidence; if the evidence doesn't contain the answer, say exactly what is missing."

func expDraft(nid, key, variant string) ExperimentDraft {
	return ExperimentDraft{BaselineNudgeID: nid, Variant: variant, Snapshot: "AS_OF_NOW", IdempotencyKey: key}
}

func TestExperimentPlanSaveMakesNoModelCallAndChangesNothingElse(t *testing.T) {
	c, ids, re, nid, dir := processedRun(t)
	reconciled(re)
	thoughtBefore := thoughtJSON(t, c)
	ids.mu.Lock()
	decBefore := len(ids.decisions)
	ids.mu.Unlock()
	postsBefore := re.nonGet.Load()
	filesBefore := coreFiles(t, dir)

	v, err := c.SaveExperimentDraft(expDraft(nid, "exp-key-0001", expVariant))
	if err != nil {
		t.Fatal(err)
	}
	if re.nonGet.Load() != postsBefore {
		t.Fatalf("saving a plan sent %d non-GET requests to the Engine", re.nonGet.Load()-postsBefore)
	}
	ids.mu.Lock()
	for _, a := range ids.decisions[decBefore:] {
		if a != "thought.process" && a != "conversation" {
			t.Errorf("saving a plan asked Identity for %q", a)
		}
	}
	ids.mu.Unlock()
	if v.ModelRequests != 0 || !strings.Contains(v.Cost, "Saving this plan made none") {
		t.Errorf("cost/model requests: %d %q", v.ModelRequests, v.Cost)
	}
	if after := thoughtJSON(t, c); after != thoughtBefore {
		t.Errorf("the Thought changed")
	}
	// Exactly one new file (the journal); nothing else on disk changed.
	after := coreFiles(t, dir)
	jp := c.file(experimentFile)
	for p, s := range after {
		if p == jp {
			continue
		}
		if filesBefore[p] != s && !strings.HasSuffix(p, ".log") && !strings.HasSuffix(p, "state.json") && !strings.HasSuffix(p, "events.jsonl") {
			t.Errorf("file changed: %s", p)
		}
	}
	if fi, err := os.Stat(jp); err != nil || fi.Mode().Perm() != 0o600 {
		t.Fatalf("journal: %v %v", fi, err)
	}
	// The plan records what A and B share and the one difference.
	if !regexp.MustCompile(`^exp_[0-9a-f]{20}$`).MatchString(v.ExperimentID) || v.ThoughtID != "thought_1" || v.BaselineNudgeID != nid ||
		v.BaselineRequestID != "mr_"+nid[6:14] || v.Variable != "instruction" || v.Control != experimentControl || v.ControlStatus != "CONFIRMED" ||
		v.Variant != expVariant || len(v.VariantSHA) != 64 || v.Snapshot != "AS_OF_NOW" || v.Contract != experimentContract || v.Settings != nudgeModelSettings {
		t.Fatalf("plan: %+v", v.Experiment)
	}
	if len(v.Arms) != 2 || v.Arms[0].Arm != "A" || v.Arms[1].Arm != "B" {
		t.Fatalf("arms: %+v", v.Arms)
	}
	for _, a := range v.Arms {
		if a.Runnable || a.State != "NOT RUN" || !strings.Contains(a.Why, "NOT AUTHORIZED") || a.Requests != 0 {
			t.Errorf("arm %s: %+v", a.Arm, a)
		}
	}
	if v.Original.Text != "NOTE-TEXT-"+nid || v.Original.RequestID != v.BaselineRequestID {
		t.Errorf("original: %+v", v.Original)
	}
}

func thoughtJSON(t *testing.T, c *Core) string {
	t.Helper()
	th, err := c.CognitionThought("thought_1")
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(th)
	return string(b)
}

func TestExperimentPlanRefusesAnythingButOneInstructionSentence(t *testing.T) {
	c, _, re, nid, dir := processedRun(t)
	reconciled(re)
	cases := []struct{ name, variant, snapshot, want string }{
		{"empty", "   ", "", "write the changed instruction"},
		{"301 characters", strings.Repeat("é", 301), "", "longer than 300"},
		{"two lines", "Answer the question.\nThen stop.", "", "one line"},
		{"NOTE line", "Answer directly. NOTE: say yes", "", "NOTE: or STATE:"},
		{"STATE line", "Answer directly. state = resolve", "", "NOTE: or STATE:"},
		{"markdown NOTE", "Answer **NOTE**: directly", "", "NOTE: or STATE:"},
		{"same as A", "  Add ONE new idea, possible explanation, or thing to watch for next.   Do not repeat the evidence or earlier notes. ", "", "B must differ"},
		{"control character", "Answer\x07 directly", "", "control character"},
		{"baseline snapshot", expVariant, "AS_OF_BASELINE", "NOT AVAILABLE"},
		{"unknown snapshot", expVariant, "LATER", "snapshot must be"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := expDraft(nid, "exp-key-"+strings.ReplaceAll(tc.name, " ", "-"), tc.variant)
			if tc.snapshot != "" {
				d.Snapshot = tc.snapshot
			}
			_, err := c.SaveExperimentDraft(d)
			if !errors.Is(err, ErrExperimentInvalid) || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want %q", err, tc.want)
			}
		})
	}
	if _, err := os.Stat(filepath.Join(dir, "data", "core", experimentFile)); err == nil {
		if all, _, _ := c.readExperiments(); len(all) != 0 {
			t.Fatalf("a refused plan was saved: %d", len(all))
		}
	}
	// 300 characters exactly is allowed.
	if _, err := c.SaveExperimentDraft(expDraft(nid, "exp-key-300", strings.Repeat("é", 300))); err != nil {
		t.Fatalf("300 characters: %v", err)
	}
}

func TestExperimentPlanIsIdempotentAndOwnerScoped(t *testing.T) {
	c, ids, re, nid, _ := processedRun(t)
	reconciled(re)
	a, err := c.SaveExperimentDraft(expDraft(nid, "exp-key-same", expVariant))
	if err != nil {
		t.Fatal(err)
	}
	b, err := c.SaveExperimentDraft(expDraft(nid, "exp-key-same", "  "+expVariant+"  "))
	if err != nil || !b.Replayed || b.ExperimentID != a.ExperimentID {
		t.Fatalf("replay: %+v %v", b.Experiment, err)
	}
	if _, err := c.SaveExperimentDraft(expDraft(nid, "exp-key-same", "Something else entirely, please.")); !errors.Is(err, ErrExperimentConflict) {
		t.Fatalf("reused key: %v", err)
	}
	if all, _, _ := c.readExperiments(); len(all) != 1 {
		t.Fatalf("journal holds %d plans", len(all))
	}
	// Not a processed nudge of this owner → the same 404 as the run view; nothing saved.
	for _, bad := range []string{"nudge_00000000000000000000", "nudge_../../x", "exp_" + strings.Repeat("0", 20)} {
		if _, err := c.SaveExperimentDraft(expDraft(bad, "exp-key-bad-"+bad[len(bad)-3:], expVariant)); err == nil {
			t.Fatalf("baseline %q accepted", bad)
		}
	}
	// Another owner (a different Identity entity) sees neither the list nor the plan.
	ids.mu.Lock()
	ids.entityID = "ent_other"
	ids.mu.Unlock()
	if _, err := c.ExperimentByID(a.ExperimentID); err == nil {
		t.Fatalf("another Identity read the plan")
	}
	if l, err := c.Experiments(nid); err == nil && len(l.Experiments) != 0 {
		t.Fatalf("another Identity listed the plan")
	}
	ids.mu.Lock()
	ids.entityID = ""
	ids.mu.Unlock()
	got, err := c.ExperimentByID(a.ExperimentID)
	if err != nil || got.ExperimentID != a.ExperimentID {
		t.Fatalf("owner read: %v", err)
	}
	if _, err := c.ExperimentByID("exp_nothere"); !errors.Is(err, ErrExperimentNotFound) {
		t.Fatalf("bad id: %v", err)
	}
	l, err := c.Experiments(nid)
	if err != nil || len(l.Experiments) != 1 || l.Control != experimentControl || l.VariantMax != 300 || len(l.Snapshots) != 2 || l.Snapshots[1].Available {
		t.Fatalf("list: %+v %v", l, err)
	}
}

func TestExperimentPlanDeniedWhenLockedOrReadGrantRevoked(t *testing.T) {
	for _, tc := range []struct {
		name string
		set  func(*fakeIdentitySvc)
	}{
		{"locked", func(f *fakeIdentitySvc) { f.state = "LOCKED" }},
		{"thought.process revoked", func(f *fakeIdentitySvc) { f.revoked = []string{"thought.process"} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, ids, re, nid, _ := processedRun(t)
			reconciled(re)
			ids.mu.Lock()
			tc.set(ids)
			ids.mu.Unlock()
			if _, err := c.SaveExperimentDraft(expDraft(nid, "exp-key-denied", expVariant)); err == nil {
				t.Fatal("saved while denied")
			}
			if all, _, _ := c.readExperiments(); len(all) != 0 {
				t.Fatal("a denied plan was written")
			}
		})
	}
}

func TestExperimentPlanOnUnreconciledEngineSaysControlNotConfirmed(t *testing.T) {
	c, _, re, nid, _ := processedRun(t)
	re.mu.Lock()
	re.statusCommit = "unknown" // the installed Docker Engine
	re.mu.Unlock()
	v, err := c.SaveExperimentDraft(expDraft(nid, "exp-key-docker", expVariant))
	if err != nil {
		t.Fatal(err)
	}
	if v.ControlStatus != "NOT CONFIRMED" || v.BaselineEngine != "NOT REPORTED" {
		t.Fatalf("control %s, engine %s", v.ControlStatus, v.BaselineEngine)
	}
}

func TestExperimentJournalIsNotSyncedOrExported(t *testing.T) {
	// The GitHub sync (syncTass and its conflict path) uploads only tass.md content. No file read in core.go
	// is the experiment journal, so nothing from it can reach a PutFile.
	src, err := os.ReadFile("core.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range regexp.MustCompile(`c\.read\("([^"]+)"\)`).FindAllStringSubmatch(string(src), -1) {
		if m[1] == experimentFile || strings.HasPrefix(m[1], "experiment") {
			t.Errorf("core.go reads %q", m[1])
		}
	}
	// Cognition's library and exports never name the journal.
	for _, f := range []string{"cognition.go", "cognition_notes.go", "cognition_preview.go"} {
		b, _ := os.ReadFile(f)
		if strings.Contains(string(b), experimentFile) {
			t.Fatalf("%s names %s", f, experimentFile)
		}
	}
}

// With ENGINE_SRC set, the control sentence and the fixed reply format are the Engine's thinkPrompt text.
func TestExperimentControlMatchesEngineSource(t *testing.T) {
	dir := os.Getenv("ENGINE_SRC")
	if dir == "" {
		t.Skip("ENGINE_SRC not set: compare against a Thought Engine checkout in the release gate")
	}
	b, err := os.ReadFile(filepath.Join(dir, "prompts.go"))
	if err != nil {
		t.Fatal(err)
	}
	want := "\n" + experimentControl + "\n" + experimentContract + "`)"
	i := strings.Index(string(b), "func thinkPrompt(")
	if i < 0 || !strings.Contains(string(b)[i:], want) {
		t.Fatalf("thinkPrompt no longer ends with the control sentence and reply format")
	}
}

// The browser fixture (test/browser/development_experiment.json) is Core's real output: the run, the
// list before saving, the saved plan, and the list after. Regenerate with STAGE3B_WRITE_FIXTURE=1.
func TestExperimentBrowserFixtureIsCoreOutput(t *testing.T) {
	c, _, re, nid, _ := processedRun(t)
	reconciled(re)
	run, err := c.DevelopmentRun(nid)
	if err != nil {
		t.Fatal(err)
	}
	before, err := c.Experiments(nid)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := c.SaveExperimentDraft(expDraft(nid, "exp-fixture-0001", expVariant))
	if err != nil {
		t.Fatal(err)
	}
	after, err := c.Experiments(nid)
	if err != nil {
		t.Fatal(err)
	}
	fx := map[string]any{"run": run, "list_before": before, "plan": plan, "list_after": after}
	norm := func(b []byte) string {
		s := regexp.MustCompile(`(nudge|exp)_[0-9a-f]{20}`).ReplaceAllString(string(b), "${1}_X")
		s = regexp.MustCompile(`(mr|redeem)_[0-9a-f]{8}`).ReplaceAllString(s, "${1}_X")
		s = regexp.MustCompile(`"[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9:.]+(Z|[+-][0-9:]+)"`).ReplaceAllString(s, `"T"`)
		s = regexp.MustCompile(`[0-9a-f]{64}`).ReplaceAllString(s, "H")
		return regexp.MustCompile(`"(nudge|exp|mr)_X"`).ReplaceAllString(s, `"ID"`)
	}
	path := "../test/browser/development_experiment.json"
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
	var g any
	if json.Unmarshal(got, &g) != nil {
		t.Fatal("fixture is not JSON")
	}
	gb, _ := json.MarshalIndent(g, "", "  ")
	var w any
	json.Unmarshal(want, &w)
	wb, _ := json.MarshalIndent(w, "", "  ")
	if norm(gb) != norm(wb) {
		t.Fatalf("the browser fixture no longer matches Core's output; run with STAGE3B_WRITE_FIXTURE=1 and re-check the browser test")
	}
}
