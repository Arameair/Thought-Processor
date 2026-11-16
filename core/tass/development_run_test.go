package tass

// Stage 3A (#120; design #107 §3.2): the read-only provenance of one processed nudge.

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func stageOf(t *testing.T, r DevelopmentRun, name string) ProvenanceStage {
	t.Helper()
	for _, s := range r.Stages {
		if s.Stage == name {
			return s
		}
	}
	t.Fatalf("no %q stage in %+v", name, r.Stages)
	return ProvenanceStage{}
}

func fieldOf(s ProvenanceStage, label string) string {
	for _, f := range s.Fields {
		if f.Label == label {
			return f.Value
		}
	}
	return ""
}

func coreFiles(t *testing.T, dir string) map[string]string {
	t.Helper()
	out := map[string]string{}
	filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			if fi, err := d.Info(); err == nil {
				out[p] = fmt.Sprintf("%d/%s", fi.Size(), fi.ModTime())
			}
		}
		return nil
	})
	return out
}

// processedRun: one completed, verified supervised nudge.
func processedRun(t *testing.T) (*Core, *fakeIdentitySvc, *runEngine, string, string) {
	t.Helper()
	dir := t.TempDir()
	c, ids, re := runCore(t, dir)
	v := selfDraft(c, t, "key-stage3a-0001")
	if _, ok, err := c.ProcessNudge(v.NudgeID, v.BodyDigest); err != nil || !ok {
		t.Fatalf("process: %v", err)
	}
	waitRun(t, c, v.NudgeID, "COMPLETED")
	return c, ids, re, v.NudgeID, dir
}

var stageOrder = []string{"thought", "attention", "authorization", "context_sent", "instruction", "model_call", "model_reply",
	"parse", "saved", "memory_retrieval", "tools", "model_reasoning"}

func TestDevelopmentRunShowsEachRecordedStage(t *testing.T) {
	c, _, re, nid, dir := processedRun(t)
	reconciled(re) // an Engine that reports, now and in the run record, one inspected revision
	re.mu.Lock()
	posts := re.posts
	re.mu.Unlock()
	before := coreFiles(t, dir)
	r, err := c.DevelopmentRun(nid)
	if err != nil {
		t.Fatal(err)
	}
	if r.NudgeID != nid || r.ThoughtID != "thought_1" || r.RequestID != "mr_"+nid[6:14] || r.EngineEntityID != "eng_1" || !r.Verified || r.RunState != "COMPLETED" {
		t.Fatalf("header: %+v", r)
	}
	var got []string
	for _, s := range r.Stages {
		got = append(got, s.Stage)
	}
	if strings.Join(got, ",") != strings.Join(stageOrder, ",") {
		t.Fatalf("stages: %v", got)
	}
	want := map[string]string{"thought": "RECORDED", "attention": "RECORDED", "authorization": "RECORDED", "context_sent": "RECORDED",
		"instruction": "CODE-DEFINED", "model_call": "RECORDED", "model_reply": "RECORDED", "parse": "CODE-DEFINED", "saved": "RECORDED",
		"memory_retrieval": "NOT PART OF THIS PATH", "tools": "NOT PART OF THIS PATH", "model_reasoning": "NOT RECORDED"}
	for _, s := range r.Stages {
		if s.Status != want[s.Stage] {
			t.Errorf("%s: %s (%s), want %s", s.Stage, s.Status, s.Summary, want[s.Stage])
		}
		if s.Summary == "" {
			t.Errorf("%s: every stage explains itself", s.Stage)
		}
	}
	if s := stageOf(t, r, "attention"); s.Text != "Please look at this again: key-stage3a-0001" {
		t.Errorf("attention text: %q", s.Text)
	}
	if s := stageOf(t, r, "context_sent"); !strings.Contains(s.Text, "[owner attention sha256:") || strings.Contains(s.Text, "key-stage3a-0001") {
		t.Errorf("the prompt is shown as logged, with the attention as its digest only: %q", s.Text)
	}
	if s := stageOf(t, r, "model_reply"); s.Text != "NOTE: NOTE-TEXT-"+nid+"\nSTATE: resolve" {
		t.Errorf("raw reply: %q", s.Text)
	}
	if s := stageOf(t, r, "parse"); s.Text != "NOTE-TEXT-"+nid || fieldOf(s, "Suggested state") != "resolve — not applied" {
		t.Errorf("parse: %+v", s)
	}
	if s := stageOf(t, r, "model_reasoning"); !strings.Contains(s.Summary, "thinking off") || s.Text != "" {
		t.Errorf("reasoning must not be invented: %+v", s)
	}
	if s := stageOf(t, r, "tools"); !strings.Contains(s.Summary, "not evidence the action happened") {
		t.Errorf("tools stage: %q", s.Summary)
	}
	// Read only: no Engine write, no new or changed Core file, and reading again gives the same answer.
	r2, err := c.DevelopmentRun(nid)
	re.mu.Lock()
	posts2 := re.posts
	re.mu.Unlock()
	if err != nil || posts2 != posts || fmt.Sprint(r2) != fmt.Sprint(r) {
		t.Fatalf("reading twice changed something: posts %d→%d, err %v", posts, posts2, err)
	}
	after := coreFiles(t, dir)
	if len(after) != len(before) {
		t.Errorf("reading the run created or removed files: %d → %d", len(before), len(after))
	}
	for p, v := range after {
		if before[p] != v {
			t.Errorf("reading the run wrote %s", p)
		}
	}
}

func TestDevelopmentRunIsOwnerAndEntityScoped(t *testing.T) {
	c, ids, re, nid, _ := processedRun(t)
	// A saved draft that was never processed has no run.
	draft := selfDraft(c, t, "key-stage3a-0002")
	for _, id := range []string{draft.NudgeID, "nudge_00000000000000000099", "nudge_../../etc", "", "thought_1"} {
		if _, err := c.DevelopmentRun(id); !errors.Is(err, ErrDevelopmentRunNotFound) {
			t.Errorf("%q: %v, want not found", id, err)
		}
	}
	// Locked Identity: denied before anything is read from the Engine.
	re.mu.Lock()
	gets := re.recordGets
	re.mu.Unlock()
	ids.set("LOCKED")
	if r, err := c.DevelopmentRun(nid); !errors.Is(err, ErrCognitionDenied) || len(r.Stages) != 0 {
		t.Fatalf("LOCKED: %v %+v", err, r)
	}
	ids.set("UNLOCKED")
	// Grant revoked (thought.process): denied.
	ids.mu.Lock()
	ids.revoked = []string{"thought.process"}
	ids.mu.Unlock()
	if _, err := c.DevelopmentRun(nid); !errors.Is(err, ErrCognitionDenied) {
		t.Fatalf("revoked grant: %v", err)
	}
	re.mu.Lock()
	if re.recordGets != gets {
		t.Fatalf("a denied read contacted the Engine's run record: %d→%d", gets, re.recordGets)
	}
	re.mu.Unlock()
}

func TestDevelopmentRunUnlinkedIdentityAndOtherOwnersNudge(t *testing.T) {
	c, _, _, nid, _ := processedRun(t)
	if err := c.UnlinkCognition(); err != nil {
		t.Fatal(err)
	}
	if r, err := c.DevelopmentRun(nid); err == nil || len(r.Stages) != 0 {
		t.Fatalf("an unlinked Identity must not read the run: %v", err)
	}
	if _, err := c.LinkCognition("eng_1"); err != nil {
		t.Fatal(err)
	}
	// Another owner's nudge in the same journal: rewrite this one's owner and ask again.
	c.nudgeMu.Lock()
	all, _, _ := c.readNudges()
	c.nudgeMu.Unlock()
	if len(all) == 0 {
		t.Fatal("no journal")
	}
	path := c.file(nudgeFile)
	b, _ := os.ReadFile(path)
	os.WriteFile(path, []byte(strings.ReplaceAll(string(b), all[0].OwnerEntityID, "ent_someoneelse0")), 0o600)
	if _, err := c.DevelopmentRun(nid); !errors.Is(err, ErrDevelopmentRunNotFound) {
		t.Fatalf("another owner's nudge: %v", err)
	}
}

func TestDevelopmentRunMismatchesAreNeverShownAsRecorded(t *testing.T) {
	type mut func(rec map[string]any) (int, any)
	val := func(rec map[string]any, k string) map[string]any {
		return rec[k].(map[string]any)["value"].(map[string]any)
	}
	cases := []struct {
		name  string
		hook  mut
		check func(t *testing.T, r DevelopmentRun, err error)
	}{
		{"reply does not match the run's digest", func(rec map[string]any) (int, any) {
			rec["model_request"].(map[string]any)["text"] = "NOTE: something else entirely\nSTATE: keep"
			return 200, rec
		}, func(t *testing.T, r DevelopmentRun, err error) {
			s := stageOf(t, r, "model_reply")
			if err != nil || s.Status != "PARTIAL" || s.Text != "" {
				t.Fatalf("%v %+v", err, s)
			}
		}},
		{"full reply missing: call-log reply, clipped", func(rec map[string]any) (int, any) {
			rec["model_request"] = nil
			rec["call"].(map[string]any)["reply_clipped"] = true
			return 200, rec
		}, func(t *testing.T, r DevelopmentRun, err error) {
			s := stageOf(t, r, "model_reply")
			if err != nil || s.Status != "PARTIAL" || !strings.Contains(s.Summary, "cut off") || s.Text == "" {
				t.Fatalf("%v %+v", err, s)
			}
		}},
		{"decision redeemed differs", func(rec map[string]any) (int, any) {
			val(rec, "claim")["identity_decision"] = "dec_other"
			return 200, rec
		}, func(t *testing.T, r DevelopmentRun, err error) {
			if s := stageOf(t, r, "authorization"); err != nil || s.Status != "PARTIAL" {
				t.Fatalf("%v %+v", err, s)
			}
		}},
		{"attention digest differs", func(rec map[string]any) (int, any) {
			val(rec, "claim")["attention_sha256"] = strings.Repeat("0", 64)
			return 200, rec
		}, func(t *testing.T, r DevelopmentRun, err error) {
			if s := stageOf(t, r, "attention"); err != nil || s.Status != "PARTIAL" || s.Text != "" {
				t.Fatalf("a mismatched attention must not be shown as sent: %v %+v", err, s)
			}
		}},
		{"call log record not found", func(rec map[string]any) (int, any) {
			rec["call"] = nil
			return 200, rec
		}, func(t *testing.T, r DevelopmentRun, err error) {
			if s := stageOf(t, r, "context_sent"); err != nil || s.Status != "NOT RECORDED" || !strings.Contains(s.Summary, "window") {
				t.Fatalf("%v %+v", err, s)
			}
			if s := stageOf(t, r, "thought"); s.Status != "PARTIAL" {
				t.Fatalf("the question cannot be confirmed without the logged prompt: %+v", s)
			}
		}},
		{"prompt clipped in the log", func(rec map[string]any) (int, any) {
			rec["call"].(map[string]any)["prompt_clipped"] = true
			return 200, rec
		}, func(t *testing.T, r DevelopmentRun, err error) {
			if s := stageOf(t, r, "context_sent"); err != nil || s.Status != "PARTIAL" {
				t.Fatalf("%v %+v", err, s)
			}
		}},
		{"no HTTP attempt record", func(rec map[string]any) (int, any) {
			rec["http"] = map[string]any{"present": false, "readable": false}
			rec["response"] = map[string]any{"present": false, "readable": false}
			return 200, rec
		}, func(t *testing.T, r DevelopmentRun, err error) {
			if s := stageOf(t, r, "model_call"); err != nil || s.Status != "NOT RECORDED" {
				t.Fatalf("%v %+v", err, s)
			}
		}},
		{"attempted, no response", func(rec map[string]any) (int, any) {
			rec["response"] = map[string]any{"present": false, "readable": false}
			return 200, rec
		}, func(t *testing.T, r DevelopmentRun, err error) {
			s := stageOf(t, r, "model_call")
			if err != nil || s.Status != "PARTIAL" || !strings.Contains(s.Summary, "may have received") {
				t.Fatalf("%v %+v", err, s)
			}
		}},
		{"the Engine never claimed it", func(rec map[string]any) (int, any) {
			return 404, map[string]string{"error": "not found: never claimed"}
		}, func(t *testing.T, r DevelopmentRun, err error) {
			if err != nil {
				t.Fatal(err)
			}
			for _, name := range []string{"context_sent", "model_call", "model_reply"} {
				if s := stageOf(t, r, name); s.Status != "NOT RECORDED" || !strings.Contains(s.Summary, "no claim") || s.Text != "" {
					t.Fatalf("%s: %+v", name, s)
				}
			}
		}},
		{"a void claim", func(rec map[string]any) (int, any) {
			val(rec, "claim")["void"] = true
			return 200, rec
		}, func(t *testing.T, r DevelopmentRun, err error) {
			if s := stageOf(t, r, "model_reply"); err != nil || s.Status != "NOT RECORDED" || s.Text != "" {
				t.Fatalf("a void claim's logs must not be used: %v %+v", err, s)
			}
		}},
		{"a claim for another Thought", func(rec map[string]any) (int, any) {
			val(rec, "claim")["thought_id"] = "thought_other"
			return 200, rec
		}, func(t *testing.T, r DevelopmentRun, err error) {
			if s := stageOf(t, r, "context_sent"); err != nil || s.Status != "NOT RECORDED" {
				t.Fatalf("%v %+v", err, s)
			}
		}},
		{"the Engine answers for another run", func(rec map[string]any) (int, any) {
			rec["run"].(map[string]any)["nudge_id"] = "nudge_ffffffffffffffffffff"
			return 200, rec
		}, func(t *testing.T, r DevelopmentRun, err error) {
			if !errors.Is(err, ErrCognitionUnavailable) || len(r.Stages) != 0 {
				t.Fatalf("%v %+v", err, r)
			}
		}},
		{"an Engine without the record route", func(rec map[string]any) (int, any) {
			return 404, "404 page not found"
		}, func(t *testing.T, r DevelopmentRun, err error) {
			if !errors.Is(err, ErrCognitionUnavailable) || len(r.Stages) != 0 {
				t.Fatalf("an older Engine must be unavailable, never 'not recorded': %v %+v", err, r)
			}
		}},
		{"an Engine error", func(rec map[string]any) (int, any) {
			return 500, map[string]string{"error": "disk"}
		}, func(t *testing.T, r DevelopmentRun, err error) {
			if !errors.Is(err, ErrCognitionUnavailable) {
				t.Fatalf("%v", err)
			}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, _, re, nid, _ := processedRun(t)
			re.mu.Lock()
			re.recordHook = func(_ string, rec map[string]any) (int, any) { return tc.hook(rec) }
			re.mu.Unlock()
			r, err := c.DevelopmentRun(nid)
			tc.check(t, r, err)
			for _, s := range r.Stages {
				if s.Stage == "model_reasoning" && s.Status != "NOT RECORDED" {
					t.Fatalf("reasoning: %+v", s)
				}
			}
		})
	}
}

// The instruction summary describes the Engine's prompt code at specific commits; tass.sh must pin one of them.
func TestNudgeInstructionSummaryMatchesThePinnedEngine(t *testing.T) {
	b, err := os.ReadFile("../tass.sh")
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`(?m)^ENGINE_COMMIT_PIN="([0-9a-f]{40})"`).FindSubmatch(b)
	if m == nil {
		t.Fatal("no ENGINE_COMMIT_PIN in tass.sh")
	}
	for _, c := range nudgeInstructionEngineCommits {
		if c == string(m[1]) {
			return
		}
	}
	t.Fatalf("tass.sh pins Engine %s, but the instruction summary was written for %v: re-read the Engine's thinkPrompt/parseThink/askNudge and update it",
		m[1], nudgeInstructionEngineCommits)
}

// The browser fixture (test/browser/cognition_stage3a_run.json) is Core's real output for a completed,
// verified run: same stages, titles, statuses, summaries and text labels. IDs and texts are fixture values.
// STAGE3A_WRITE_FIXTURE=1 rewrites it from Core.
func TestDevelopmentRunBrowserFixtureIsCoreOutput(t *testing.T) {
	c, _, re, nid, _ := processedRun(t)
	reconciled(re)
	r, err := c.DevelopmentRun(nid)
	if err != nil {
		t.Fatal(err)
	}
	type shape struct{ Stage, Title, Status, Summary, TextLabel string }
	var want []shape
	for _, s := range r.Stages {
		want = append(want, shape{s.Stage, s.Title, s.Status, s.Summary, s.TextLabel})
	}
	path := "../test/browser/cognition_stage3a_run.json"
	if os.Getenv("STAGE3A_WRITE_FIXTURE") == "1" {
		b, _ := json.MarshalIndent(r, "", "  ")
		if err := os.WriteFile(path, append(b, '\n'), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var fx DevelopmentRun
	if err := json.Unmarshal(b, &fx); err != nil {
		t.Fatal(err)
	}
	var got []shape
	for _, s := range fx.Stages {
		got = append(got, shape{s.Stage, s.Title, s.Status, s.Summary, s.TextLabel})
	}
	if fmt.Sprint(got) != fmt.Sprint(want) || fx.ReadOnly != r.ReadOnly {
		t.Fatalf("the browser fixture no longer matches Core's output; run with STAGE3A_WRITE_FIXTURE=1 and re-check the browser test\nfixture %v\ncore    %v", got, want)
	}
}
