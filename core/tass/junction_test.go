package tass

// #82 P2 first information junction. Fixture Identity/Engine/GitHub, not the owner's machine.

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func junctionCore(t *testing.T, granted bool) (*fakeIdentitySvc, *fakeEngine, *fakeGH, string, *Core) {
	ids, eng, gh, dir, c := skillCore(t, false)
	ids.mu.Lock()
	ids.junctionGranted = granted
	ids.mu.Unlock()
	return ids, eng, gh, dir, c
}

func receipt(c *Core, id string) JunctionReceipt {
	for _, r := range c.Junctions().Receipts {
		if r.JunctionID == id {
			return r
		}
	}
	return JunctionReceipt{}
}

func TestJunctionCellIsStrictData(t *testing.T) {
	cell := activeJunctionCell()
	if cell.Handler != "lesson-metadata.v1" || cell.Capability != "READ_LESSON_METADATA" || len(cell.Fields) != 4 || cell.Version == "" {
		t.Fatalf("active cell: %+v", cell)
	}
	for name, bad := range map[string]string{
		"other handler": strings.Replace(cell.Text, "lesson-metadata.v1", "shell.v1", 1),
		"unknown key":   cell.Text + "\nrun: rm -rf /\n",
		"bad field":     strings.Replace(cell.Text, "fields: title, goal, checks, status", "fields: title, secrets", 1),
		"duplicate key": cell.Text + "\nid: junction.lesson-metadata\n",
		"too long":      cell.Text + strings.Repeat("x", 5000),
	} {
		if _, err := ParseJunctionCell(bad); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestJunctionDataThenConsumedOnceByALaterCycle(t *testing.T) {
	_, eng, _, dir, c := junctionCore(t, true)
	r, err := c.RequestJunction(pulsePath, "TASS reply", "cyc_origin", "test")
	if err != nil || r.Status != "PENDING" || r.CellVersion != activeJunctionCell().Version {
		t.Fatalf("request: %+v %v", r, err)
	}
	if again, _ := c.RequestJunction(pulsePath, "TASS reply", "cyc_origin", "test"); again.JunctionID != r.JunctionID || len(c.Junctions().Receipts) != 1 {
		t.Fatal("duplicate request created a second receipt")
	}
	c.ProcessJunctions()
	got := receipt(c, r.JunctionID)
	if got.Status != "DATA" || got.DecisionID == "" || got.SourceSHA == "" || got.ResolvedBy != "Core 20 s sync tick" ||
		!strings.Contains(got.Data, "title: My work pulse") || !strings.Contains(got.Data, "checks: ") {
		t.Fatalf("handler result: %+v", got)
	}
	c.Answer(Incoming{Text: "what did you find out?", From: "local"})
	if eng.count() != 1 {
		t.Fatal("no cycle")
	}
	eng.mu.Lock()
	ctx := eng.reqs[0]["context"].(string)
	cycleID := eng.reqs[0]["cycle_id"].(string)
	eng.mu.Unlock()
	if !strings.Contains(ctx, "INFORMATION YOU ASKED FOR") || !strings.Contains(ctx, r.JunctionID) || !strings.Contains(ctx, "title: My work pulse") {
		t.Fatal("eligible cycle did not receive the data")
	}
	got = receipt(c, r.JunctionID)
	if got.ConsumedAt == nil || got.ConsumedByCycle != cycleID || got.Effect != "EFFECT NOT MEASURED" {
		t.Fatalf("consumption not recorded: %+v", got)
	}
	c.Answer(Incoming{Text: "and again?", From: "local"})
	eng.mu.Lock()
	ctx2 := eng.reqs[1]["context"].(string)
	eng.mu.Unlock()
	if strings.Contains(ctx2, r.JunctionID) {
		t.Fatal("consumed data given twice")
	}
	b, _ := os.ReadFile(filepath.Join(dir, "conversation.jsonl"))
	if !strings.Contains(string(b), r.JunctionID) {
		t.Fatal("conversation log doesn't record which junction data the cycle had")
	}
}

func TestJunctionAwaitsGrantThenReadsAndNeverAuditsWhileWaiting(t *testing.T) {
	ids, _, _, _, c := junctionCore(t, false)
	r, _ := c.RequestJunction(pulsePath, "owner (Development)", "", "test")
	ids.mu.Lock()
	before := len(ids.decisions)
	ids.mu.Unlock()
	c.ProcessJunctions()
	c.ProcessJunctions()
	got := receipt(c, r.JunctionID)
	ids.mu.Lock()
	after := len(ids.decisions)
	ids.mu.Unlock()
	if got.Status != "AWAITING_APPROVAL" || got.Data != "" || got.DecisionID != "" || after != before {
		t.Fatalf("waiting: %+v decisions %d→%d", got, before, after)
	}
	ids.mu.Lock()
	ids.junctionGranted = true
	ids.mu.Unlock()
	c.ProcessJunctions()
	if got := receipt(c, r.JunctionID); got.Status != "DATA" {
		t.Fatalf("after grant: %+v", got)
	}
	ids.set("LOCKED")
	r2, _ := c.RequestJunction("lessons/claude/01-pulse.md", "owner (Development)", "", "test")
	c.ProcessJunctions()
	if got := receipt(c, r2.JunctionID); got.Status != "AWAITING_APPROVAL" || got.Data != "" {
		t.Fatalf("LOCKED must not read: %+v", got)
	}
}

func TestJunctionMissingInvalidDeniedTimeoutExpired(t *testing.T) {
	ids, _, _, _, c := junctionCore(t, true)
	missing, _ := c.RequestJunction("lessons/claude/99-nope.md", "TASS reply", "c1", "")
	bad, _ := c.RequestJunction("../secrets/github_token.md", "TASS reply", "c1", "")
	c.ProcessJunctions()
	if m := receipt(c, missing.JunctionID); m.Status != "UNAVAILABLE" || m.Data != "" {
		t.Fatalf("missing: %+v", m)
	}
	if b := receipt(c, bad.JunctionID); b.Status != "UNAVAILABLE" || b.DecisionID != "" {
		t.Fatalf("traversal must be refused before any read or decision: %+v", b)
	}
	// Identity entity changed between request and read → DENIED, nothing read.
	r, _ := c.RequestJunction(pulsePath, "TASS reply", "c2", "")
	ids.mu.Lock()
	ids.entityID = "ent_OTHER"
	ids.mu.Unlock()
	c.ProcessJunctions()
	if d := receipt(c, r.JunctionID); d.Status != "DENIED" || d.Data != "" {
		t.Fatalf("entity change: %+v", d)
	}
	ids.mu.Lock()
	ids.entityID = ""
	ids.mu.Unlock()
	// Timeout: a handler that never finishes in time.
	orig := lessonMetadataHandler
	lessonMetadataHandler = func(ctx context.Context, c *Core, cell JunctionCell, p string) (string, string, string, string) {
		<-ctx.Done()
		time.Sleep(10 * time.Millisecond)
		return "late", "x", "DATA", ""
	}
	defer func() { lessonMetadataHandler = orig }()
	tr, _ := c.RequestJunction(pulsePath, "TASS reply", "c3", "")
	start := time.Now()
	c.ProcessJunctions()
	if g := receipt(c, tr.JunctionID); g.Status != "TIMEOUT" || g.Data != "" || time.Since(start) > junctionTimeout+time.Second {
		t.Fatalf("timeout: %+v after %s", g, time.Since(start))
	}
	lessonMetadataHandler = orig
	// Deadline passed while waiting → EXPIRED.
	ex, _ := c.RequestJunction(pulsePath, "TASS reply", "c4", "")
	c.now = func() time.Time { return time.Now().Add(junctionDeadline + time.Minute) }
	c.ProcessJunctions()
	if e := receipt(c, ex.JunctionID); e.Status != "EXPIRED" {
		t.Fatalf("expiry: %+v", e)
	}
}

func TestJunctionRestartRetriesReadOnlyAndTickNeverBlocks(t *testing.T) {
	ids, eng, gh, dir, c := junctionCore(t, true)
	r, _ := c.RequestJunction(pulsePath, "TASS reply", "c1", "")
	c.updateJunction(r.JunctionID, func(x *JunctionReceipt) { x.Status = "RUNNING" }) // crash mid-read
	c = newLearningCore(t, dir, ids, gh, eng)
	if got := receipt(c, r.JunctionID); got.Status != "PENDING" || !strings.Contains(got.Why, "restarted") {
		t.Fatalf("restart: %+v", got)
	}
	orig := lessonMetadataHandler
	release := make(chan struct{})
	lessonMetadataHandler = func(ctx context.Context, c *Core, cell JunctionCell, p string) (string, string, string, string) {
		select {
		case <-release:
		case <-ctx.Done():
		}
		return orig(context.Background(), c, cell, p)
	}
	defer func() { lessonMetadataHandler = orig }()
	start := time.Now()
	c.junctionTick() // what SyncOnce calls every 20 s
	c.junctionTick() // a second tick while the first worker runs: must not start another
	if time.Since(start) > 200*time.Millisecond {
		t.Fatal("the tick waited for the junction")
	}
	close(release)
	deadline := time.Now().Add(5 * time.Second)
	for receipt(c, r.JunctionID).Status != "DATA" && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if got := receipt(c, r.JunctionID); got.Status != "DATA" || got.Attempts != 1 {
		t.Fatalf("after restart: %+v", got)
	}
}

func TestJunctionNotConsumedWhenLessonChangedAndNoCrossIdentity(t *testing.T) {
	ids, eng, gh, _, c := junctionCore(t, true)
	r, _ := c.RequestJunction(pulsePath, "TASS reply", "c1", "")
	c.ProcessJunctions()
	gh.set("Mira/home@live:"+pulsePath, threeCheckPulseLesson+"\n<!-- changed -->\n")
	c.DiscoverLessons()
	c.Answer(Incoming{Text: "hi", From: "local"})
	eng.mu.Lock()
	ctx := eng.reqs[0]["context"].(string)
	eng.mu.Unlock()
	got := receipt(c, r.JunctionID)
	if strings.Contains(ctx, r.JunctionID) || got.ConsumedAt != nil || !strings.Contains(got.NotConsumed, "lesson changed") {
		t.Fatalf("stale data must not be consumed: %+v", got)
	}
	// Another Identity entity sees neither the receipts in Cognition nor the data in context.
	r2, _ := c.RequestJunction("lessons/claude/01-pulse.md", "TASS reply", "c2", "")
	c.ProcessJunctions()
	ids.mu.Lock()
	ids.entityID = "ent_OTHER"
	ids.mu.Unlock()
	if cards := c.junctionCards("ent_OTHER"); len(cards) != 0 {
		t.Fatalf("cross-identity junction cards: %d", len(cards))
	}
	if txt, used := c.junctionContext(); txt != "" || len(used) != 0 {
		t.Fatalf("cross-identity junction data: %q", txt)
	}
	if receipt(c, r2.JunctionID).ConsumedAt != nil {
		t.Fatal("consumed under another identity")
	}
}

func TestJunctionRequestedFromTASSReplyAndShownInCognition(t *testing.T) {
	_, eng, _, _, c := junctionCore(t, true)
	eng.mu.Lock()
	eng.replyExtra = "\nNEED lesson-metadata: " + pulsePath
	eng.mu.Unlock()
	c.Answer(Incoming{Text: "what's in your pulse lesson?", From: "local"})
	v := c.Junctions()
	if len(v.Receipts) != 1 || v.Receipts[0].Origin != "TASS reply" || v.Receipts[0].OriginCycleID == "" || v.Receipts[0].Status != "PENDING" {
		t.Fatalf("reply need: %+v", v.Receipts)
	}
	if !strings.Contains(eng.reqs[0]["context"].(string), "NEED lesson-metadata:") {
		t.Fatal("TASS isn't told how to ask")
	}
	c.ProcessJunctions()
	cards := c.junctionCards("ent_0123")
	if len(cards) != 1 || cards[0].Kind != "junction" || cards[0].State != "DATA" || len(cards[0].RelatedCycleIDs) != 1 {
		t.Fatalf("cognition card: %+v", cards)
	}
	var fields []string
	for _, f := range cards[0].Fields {
		fields = append(fields, f.Name+"="+f.Value)
	}
	all := strings.Join(fields, "\n")
	for _, want := range []string{"1 · Requested=", "2 · Authorized (Identity decision)=Identity dec_", "3 · Returned=DATA at", "4 · Consumed=NOT CONSUMED YET", "5 · Effect on thinking=EFFECT NOT MEASURED"} {
		if !strings.Contains(all, want) {
			t.Fatalf("card missing %q:\n%s", want, all)
		}
	}
}

func TestJunctionProposalComparesButNeverActivates(t *testing.T) {
	ids, _, _, _, c := junctionCore(t, true)
	active := activeJunctionCell()
	alt := strings.Replace(active.Text, "fields: title, goal, checks, status", "fields: title, steps, sha", 1)
	p, err := c.ProposeJunctionCell(alt, "try steps instead of checks")
	if err != nil || p.State != "PROPOSED" || p.Version == active.Version {
		t.Fatalf("proposal: %+v %v", p, err)
	}
	cmp, err := c.CompareJunctionCells(p.ID, pulsePath)
	if err != nil || cmp.Active.Version != active.Version || cmp.Proposal.Version != p.Version || cmp.Same ||
		!strings.Contains(cmp.Proposal.Output, "steps:") || strings.Contains(cmp.Active.Output, "steps:") || !strings.Contains(cmp.Effect, "NOT MEASURED") {
		t.Fatalf("comparison: %+v %v", cmp, err)
	}
	if activeJunctionCell().Version != active.Version {
		t.Fatal("a proposal changed the active cell")
	}
	r, _ := c.RequestJunction(pulsePath, "TASS reply", "c9", "")
	if r.CellVersion != active.Version {
		t.Fatal("new requests must use the active cell, not the proposal")
	}
	ids.mu.Lock()
	ids.junctionGranted = false
	ids.mu.Unlock()
	if _, err := c.CompareJunctionCells(p.ID, pulsePath); err == nil {
		t.Fatal("comparison without the Identity grant")
	}
	if _, err := ParseJunctionCell(strings.Replace(alt, "lesson-metadata.v1", "python.v1", 1)); err == nil {
		t.Fatal("a proposal may not name another handler")
	}
}

func TestCognitionReadCreatesNoJunctionOrCycle(t *testing.T) {
	_, eng, _, _, c := junctionCore(t, true)
	c.Junctions()
	c.junctionCards("ent_0123")
	if len(c.Junctions().Receipts) != 0 || eng.count() != 0 {
		t.Fatal("a read created work")
	}
}
