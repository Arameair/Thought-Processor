package tass

// Stage 3B runtime repair (Codex audit, PR #122 comment 6850390555): Core's experiment receipts
// (<data>/experiment-runs/<exp>/snapshot.json, A|B.submitted.json, A|B.outcome.json) are a trust boundary.
// A receipt that is not a private, regular, bounded file in private, real directories, or whose identity
// does not match the plan and the submission it belongs to, is never trusted, shown or built on:
// no B is sent, no decision is minted, no reply is exposed, the Thought is untouched.

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func symlinkToCopy(t *testing.T, path string) {
	t.Helper()
	real := filepath.Join(t.TempDir(), "copy")
	if err := os.Rename(path, real); err != nil {
		t.Skip(err)
	}
	if err := os.Symlink(real, path); err != nil {
		t.Skip(err)
	}
}

func rewriteJSON(t *testing.T, path string, edit func(m map[string]any)) {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if json.Unmarshal(b, &m) != nil {
		t.Fatal("not JSON: " + path)
	}
	edit(m)
	nb, _ := json.Marshal(m)
	if err := os.WriteFile(path, append(nb, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestExperimentCoreReceiptBoundaryFailsClosed(t *testing.T) {
	type kind int
	const (
		snapshotBad kind = iota // the page must say UNVERIFIED and never offer B
		armABad                 // A must be RECOVERY_REQUIRED with no reply
	)
	cases := []struct {
		name string
		kind kind
		edit func(t *testing.T, c *Core, id string)
	}{
		{"snapshot receipt chmod 0644", snapshotBad, func(t *testing.T, c *Core, id string) { os.Chmod(c.expRunPath(id, "snapshot.json"), 0o644) }},
		{"snapshot receipt replaced by a symlink to identical bytes", snapshotBad, func(t *testing.T, c *Core, id string) { symlinkToCopy(t, c.expRunPath(id, "snapshot.json")) }},
		{"snapshot receipt without its final newline", snapshotBad, func(t *testing.T, c *Core, id string) {
			p := c.expRunPath(id, "snapshot.json")
			b, _ := os.ReadFile(p)
			os.WriteFile(p, bytes.TrimSuffix(b, []byte("\n")), 0o600)
		}},
		{"snapshot receipt over the size ceiling", snapshotBad, func(t *testing.T, c *Core, id string) {
			p := c.expRunPath(id, "snapshot.json")
			b, _ := os.ReadFile(p)
			os.WriteFile(p, append(bytes.Repeat([]byte(" "), expReceiptMax), b...), 0o600)
		}},
		{"snapshot receipt for another owner", snapshotBad, func(t *testing.T, c *Core, id string) {
			rewriteJSON(t, c.expRunPath(id, "snapshot.json"), func(m map[string]any) { m["owner_entity_id"] = "ent_other" })
		}},
		{"snapshot receipt for another Thought", snapshotBad, func(t *testing.T, c *Core, id string) {
			rewriteJSON(t, c.expRunPath(id, "snapshot.json"), func(m map[string]any) { m["thought_id"] = "thought_9" })
		}},
		{"snapshot receipt for another Engine entity", snapshotBad, func(t *testing.T, c *Core, id string) {
			rewriteJSON(t, c.expRunPath(id, "snapshot.json"), func(m map[string]any) { m["engine_entity_id"] = "eng_9" })
		}},
		{"experiment receipt directory chmod 0755", snapshotBad, func(t *testing.T, c *Core, id string) { os.Chmod(filepath.Dir(c.expRunPath(id, "x")), 0o755) }},
		{"experiment receipt directory replaced by a symlink to a copy", snapshotBad, func(t *testing.T, c *Core, id string) { symlinkToCopy(t, filepath.Dir(c.expRunPath(id, "x"))) }},
		{"experiment-runs directory replaced by a symlink to a copy", snapshotBad, func(t *testing.T, c *Core, id string) {
			symlinkToCopy(t, filepath.Dir(filepath.Dir(c.expRunPath(id, "x"))))
		}},
		{"experiment-runs directory chmod 0777", snapshotBad, func(t *testing.T, c *Core, id string) {
			os.Chmod(filepath.Dir(filepath.Dir(c.expRunPath(id, "x"))), 0o777)
		}},
		{"A outcome chmod 0644", armABad, func(t *testing.T, c *Core, id string) { os.Chmod(c.expRunPath(id, "A.outcome.json"), 0o644) }},
		{"A outcome replaced by a symlink to identical bytes", armABad, func(t *testing.T, c *Core, id string) { symlinkToCopy(t, c.expRunPath(id, "A.outcome.json")) }},
		{"A outcome over the size ceiling", armABad, func(t *testing.T, c *Core, id string) {
			rewriteJSON(t, c.expRunPath(id, "A.outcome.json"), func(m map[string]any) { m["detail"] = strings.Repeat("x", expReceiptMax) })
		}},
		{"A outcome with trailing data", armABad, func(t *testing.T, c *Core, id string) {
			p := c.expRunPath(id, "A.outcome.json")
			b, _ := os.ReadFile(p)
			os.WriteFile(p, append(b, []byte("{}\n")...), 0o600)
		}},
		{"A outcome is arm B's", armABad, func(t *testing.T, c *Core, id string) {
			rewriteJSON(t, c.expRunPath(id, "A.outcome.json"), func(m map[string]any) { m["engine_arm"].(map[string]any)["arm"] = "B" })
		}},
		{"A outcome reply edited (digest no longer matches)", armABad, func(t *testing.T, c *Core, id string) {
			rewriteJSON(t, c.expRunPath(id, "A.outcome.json"), func(m map[string]any) {
				m["engine_arm"].(map[string]any)["result"].(map[string]any)["reply"] = "NOTE: PLANTED-REPLY\nSTATE: keep"
			})
		}},
		{"A outcome names another snapshot", armABad, func(t *testing.T, c *Core, id string) {
			rewriteJSON(t, c.expRunPath(id, "A.outcome.json"), func(m map[string]any) { m["engine_arm"].(map[string]any)["snapshot_sha256"] = strings.Repeat("d", 64) })
		}},
		{"A outcome names another approval", armABad, func(t *testing.T, c *Core, id string) {
			rewriteJSON(t, c.expRunPath(id, "A.outcome.json"), func(m map[string]any) { m["engine_arm"].(map[string]any)["identity_decision"] = "dec_other" })
		}},
		{"A submission chmod 0660", armABad, func(t *testing.T, c *Core, id string) { os.Chmod(c.expRunPath(id, "A.submitted.json"), 0o660) }},
		{"A submission replaced by a symlink to identical bytes", armABad, func(t *testing.T, c *Core, id string) { symlinkToCopy(t, c.expRunPath(id, "A.submitted.json")) }},
		{"A submission for another owner", armABad, func(t *testing.T, c *Core, id string) {
			rewriteJSON(t, c.expRunPath(id, "A.submitted.json"), func(m map[string]any) { m["owner_entity_id"] = "ent_other" })
		}},
		{"A submission for another Thought", armABad, func(t *testing.T, c *Core, id string) {
			rewriteJSON(t, c.expRunPath(id, "A.submitted.json"), func(m map[string]any) { m["thought_id"] = "thought_9" })
		}},
		{"A submission for another variant", armABad, func(t *testing.T, c *Core, id string) {
			rewriteJSON(t, c.expRunPath(id, "A.submitted.json"), func(m map[string]any) { m["variant_sha256"] = strings.Repeat("e", 64) })
		}},
		{"A submission on another snapshot", armABad, func(t *testing.T, c *Core, id string) {
			rewriteJSON(t, c.expRunPath(id, "A.submitted.json"), func(m map[string]any) { m["snapshot_sha256"] = strings.Repeat("f", 64) })
		}},
		{"A submission is arm B's", armABad, func(t *testing.T, c *Core, id string) {
			rewriteJSON(t, c.expRunPath(id, "A.submitted.json"), func(m map[string]any) { m["arm"] = "B" })
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, ids, re, plan := expSetup(t)
			id := plan.ExperimentID
			thought := thoughtJSON(t, c)
			c.RunExperimentArm(id, "A", confirmOf(plan, "A"))
			ok := waitArm(t, c, id, "A", "COMPLETED")
			replyA := armOfView(ok, "A").Reply
			runs := filepath.Dir(filepath.Dir(c.expRunPath(id, "x")))
			t.Cleanup(func() { os.Chmod(runs, 0o700); os.Chmod(filepath.Join(runs, id), 0o700) })
			tc.edit(t, c, id)

			_, created, err := c.RunExperimentArm(id, "B", confirmOf(plan, "B"))
			if err == nil && created {
				t.Fatal("B was accepted")
			}
			time.Sleep(30 * time.Millisecond)
			if cap, posts, calls := re.fxCounts(); cap != 1 || posts != 1 || calls != 1 {
				t.Fatalf("%d captures, %d arm requests, %d model calls (want 1, 1, 1)", cap, posts, calls)
			}
			if n := len(boundFor(ids, ActionExperimentRun)); n != 1 {
				t.Fatalf("%d decisions minted (want only A's)", n)
			}
			// What the owner sees.
			v, verr := c.ExperimentByID(id)
			if verr != nil {
				t.Fatalf("view: %v", verr)
			}
			a, b := armOfView(v, "A"), armOfView(v, "B")
			raw, _ := json.Marshal(v)
			if a.Runnable || b.Runnable {
				t.Errorf("the page offers an arm: A %v B %v", a.Runnable, b.Runnable)
			}
			if b.State == "COMPLETED" || b.Reply != "" || v.Comparison.Ready {
				t.Errorf("B: %+v", b)
			}
			if strings.Contains(string(raw), "PLANTED-REPLY") {
				t.Error("a planted reply is shown")
			}
			switch tc.kind {
			case snapshotBad:
				if v.Frozen.Status != "UNVERIFIED" || v.Frozen.SHA256 != "" {
					t.Errorf("snapshot shown as %s %q; want UNVERIFIED with no digest vouched for", v.Frozen.Status, v.Frozen.SHA256)
				}
			case armABad:
				if a.State != "RECOVERY_REQUIRED" || a.Reply != "" || strings.Contains(string(raw), replyA) {
					t.Errorf("A is %s, reply shown: %v; want RECOVERY_REQUIRED and no reply", a.State, a.Reply != "" || strings.Contains(string(raw), replyA))
				}
			}
			// A is never sent again either, and nothing new is written beside the receipts.
			c.RunExperimentArm(id, "A", confirmOf(plan, "A"))
			time.Sleep(20 * time.Millisecond)
			if _, posts, _ := re.fxCounts(); posts != 1 {
				t.Fatalf("A was sent again (%d arm requests)", posts)
			}
			if _, serr := os.Lstat(c.expRunPath(id, "B.submitted.json")); serr == nil && tc.name != "experiment receipt directory replaced by a symlink to a copy" && tc.name != "experiment-runs directory replaced by a symlink to a copy" {
				t.Error("a B submission was written")
			}
			if thoughtJSON(t, c) != thought {
				t.Error("the Thought changed")
			}
		})
	}
}

func TestExperimentCoreReceiptsAreNeverCreatedThroughUnsafeParents(t *testing.T) {
	for _, tc := range []struct {
		name string
		prep func(t *testing.T, c *Core, id string) string // returns a directory that must stay empty
	}{
		{"experiment-runs is a symlink before A", func(t *testing.T, c *Core, id string) string {
			target := t.TempDir()
			runs := filepath.Dir(filepath.Dir(c.expRunPath(id, "x")))
			os.RemoveAll(runs)
			if err := os.Symlink(target, runs); err != nil {
				t.Skip(err)
			}
			return target
		}},
		{"the experiment's receipt directory is a symlink before A", func(t *testing.T, c *Core, id string) string {
			target := t.TempDir()
			dir := filepath.Dir(c.expRunPath(id, "x"))
			os.MkdirAll(filepath.Dir(dir), 0o700)
			if err := os.Symlink(target, dir); err != nil {
				t.Skip(err)
			}
			return target
		}},
		{"the experiment's receipt directory is group-writable before A", func(t *testing.T, c *Core, id string) string {
			dir := filepath.Dir(c.expRunPath(id, "x"))
			os.MkdirAll(dir, 0o700)
			os.Chmod(dir, 0o770)
			return dir
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, ids, re, plan := expSetup(t)
			id := plan.ExperimentID
			empty := tc.prep(t, c, id)
			_, created, err := c.RunExperimentArm(id, "A", confirmOf(plan, "A"))
			if err == nil || created {
				t.Fatalf("A was accepted (created=%v)", created)
			}
			time.Sleep(30 * time.Millisecond)
			if cap, posts, calls := re.fxCounts(); cap != 0 || posts != 0 || calls != 0 {
				t.Fatalf("%d captures, %d arm requests, %d model calls (want 0: nothing may be captured before Core can record it)", cap, posts, calls)
			}
			if len(boundFor(ids, ActionExperimentRun)) != 0 {
				t.Fatal("a decision was minted")
			}
			if ents, _ := os.ReadDir(empty); len(ents) != 0 {
				t.Fatalf("receipts were written through an unsafe directory: %v", ents)
			}
		})
	}
}

// Positive control: what Core writes passes its own boundary (0700 directories, 0600 files; 0400 is
// still private), and the full A/B run is unaffected (TestExperimentRunAThenBOnOneFrozenSnapshot).
func TestExperimentCoreReceiptsItWritesPassTheBoundary(t *testing.T) {
	c, _, re, plan := expSetup(t)
	id := plan.ExperimentID
	c.RunExperimentArm(id, "A", confirmOf(plan, "A"))
	waitArm(t, c, id, "A", "COMPLETED")
	dir := filepath.Dir(c.expRunPath(id, "x"))
	for _, d := range []string{filepath.Dir(dir), dir} {
		if fi, err := os.Lstat(d); err != nil || !fi.IsDir() || fi.Mode().Perm() != 0o700 {
			t.Fatalf("%s: %v", d, fi)
		}
	}
	for _, f := range []string{"snapshot.json", "A.submitted.json", "A.outcome.json"} {
		if fi, err := os.Lstat(filepath.Join(dir, f)); err != nil || !fi.Mode().IsRegular() || fi.Mode().Perm() != 0o600 {
			t.Fatalf("%s: %v", f, fi)
		}
		os.Chmod(filepath.Join(dir, f), 0o400)
	}
	if _, created, err := c.RunExperimentArm(id, "B", confirmOf(plan, "B")); err != nil || !created {
		t.Fatalf("B on untouched (0400) receipts: %v", err)
	}
	v := waitArm(t, c, id, "B", "COMPLETED")
	if cap, posts, _ := re.fxCounts(); cap != 1 || posts != 2 || !v.Comparison.Ready || v.Frozen.Status != "FROZEN" {
		t.Fatalf("%d captures, %d arm requests, comparison %v, snapshot %s", cap, posts, v.Comparison.Ready, v.Frozen.Status)
	}
}
