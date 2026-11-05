package tass

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func brainCore(t *testing.T) (*fakeIdentitySvc, *fakeEngine, *fakeGH, string, *Core) {
	t.Helper()
	ids, eng, gh, dir, c := skillCore(t, true)
	return ids, eng, gh, dir, c
}

func blockOf(c *Core, id string) BrainBlock {
	for _, b := range c.BrainBlocks(nil) {
		if b.ID == id {
			return b
		}
	}
	return BrainBlock{}
}

func TestBrainBlocksComeFromRealCoreState(t *testing.T) {
	_, eng, _, _, c := brainCore(t)
	id, _ := c.StudyLesson(pulsePath)
	eng.finish(id, "DONE", "ANSWER 1: 20\nANSWER 2: continue\nANSWER 3: blocked", "")
	c.SyncOnce()
	c.ApproveLesson(pulsePath)
	b := blockOf(c, "lesson:"+pulsePath)
	if b.Kind != "lesson" || b.Status != "LEARNED" || b.Revision == "" || !strings.Contains(b.Last, id) || !strings.Contains(b.Runnable, "never executed") {
		t.Fatalf("lesson block not from real learning state: %+v", b)
	}
	s := blockOf(c, "skill:"+skillListLessons)
	if s.Kind != "skill" || s.Status != "GRANTED" || !strings.Contains(s.Runnable, "Approve run") {
		t.Fatalf("skill block: %+v", s)
	}
}

func TestBrainNotebookCellsAreReadOnlyBlocks(t *testing.T) {
	ids, eng, gh, dir, _ := brainCore(t)
	nb := `{"cells":[{"cell_type":"markdown","id":"intro","metadata":{},"source":["# Find and run a lesson\n"]},
	 {"cell_type":"code","id":"list","metadata":{"tass":{"title":"List every lesson","thought":"I need to know what exists","expect":"a list of paths"}},"source":["import os\n","print(os.listdir('lessons'))"]}],"metadata":{},"nbformat":4,"nbformat_minor":5}`
	gh.set("Mira/home@live:lessons/claude/00-find.ipynb", nb)
	c := newLearningCore(t, dir, ids, gh, eng)
	c.DiscoverLessons()
	b := blockOf(c, "cell:lessons/claude/00-find.ipynb:list")
	if b.Kind != "code-cell" || b.Title != "List every lesson" || b.Purpose != "I need to know what exists" || b.Expect != "a list of paths" ||
		b.Status != "DISCOVERED" || !strings.HasPrefix(b.Runnable, "NOT EXECUTABLE") {
		t.Fatalf("notebook cell block: %+v", b)
	}
}

func TestBrainSaveIsVersionedConflictSafeAndRestorable(t *testing.T) {
	_, _, _, dir, c := brainCore(t)
	lesson := blockOf(c, "lesson:"+pulsePath)
	skill := blockOf(c, "skill:"+skillListLessons)
	g := BrainGraph{Revision: 0, Nodes: []BrainNode{
		{ID: "n1", BlockID: lesson.ID, Pinned: lesson.Revision, X: 10, Y: 10},
		{ID: "n2", BlockID: skill.ID, Pinned: skill.Revision, X: 300, Y: 10}},
		Edges: []BrainEdge{{ID: "e1", From: "n1", To: "n2", Relation: "uses", Reason: "listing finds the lesson first", Creator: "owner"}}}
	s1, rep, err := c.SaveBrain(g, "owner", "first map")
	if err != nil || s1.Revision != 1 || rep.Status != "PASS" {
		t.Fatalf("save: %v %+v %+v", err, s1, rep)
	}
	if _, _, err := c.SaveBrain(g, "owner", "stale tab"); err != ErrBrainConflict {
		t.Fatalf("stale revision must conflict, got %v", err)
	}
	s1.Edges[0].Relation = "supports"
	s2, _, err := c.SaveBrain(s1, "owner", "changed relation")
	if err != nil || s2.Revision != 2 || s2.Parent != 1 {
		t.Fatalf("second save: %v %+v", err, s2)
	}
	r3, err := c.RestoreBrain(1, 2)
	if err != nil || r3.Revision != 3 || r3.Edges[0].Relation != "uses" || !strings.Contains(r3.Change, "restored revision 1") {
		t.Fatalf("restore: %v %+v", err, r3)
	}
	if h := c.BrainHistory(); len(h) != 3 || h[0].Revision != 3 {
		t.Fatalf("history: %+v", h)
	}
	if fi, err := os.Stat(filepath.Join(dir, "brain", "revisions", "000001.json")); err != nil || fi.Mode().Perm()&0o200 != 0 {
		t.Fatal("revision files must be kept read-only")
	}
}

func TestBrainValidationCatchesBadGraphsAndStaleSources(t *testing.T) {
	_, _, gh, _, c := brainCore(t)
	lesson := blockOf(c, "lesson:"+pulsePath)
	other := blockOf(c, "lesson:lessons/claude/01-pulse.md")
	base := func() BrainGraph {
		return BrainGraph{Nodes: []BrainNode{{ID: "a", BlockID: lesson.ID, Pinned: lesson.Revision}, {ID: "b", BlockID: other.ID, Pinned: other.Revision, X: 300}}}
	}
	bad := map[string]func(*BrainGraph){
		"cycle":        func(g *BrainGraph) { g.Edges = []BrainEdge{{"e1", "a", "b", "next", "", "owner"}, {"e2", "b", "a", "next", "", "owner"}} },
		"unknown rel":  func(g *BrainGraph) { g.Edges = []BrainEdge{{"e1", "a", "b", "executes", "", "owner"}} },
		"uses→lesson":  func(g *BrainGraph) { g.Edges = []BrainEdge{{"e1", "a", "b", "uses", "", "owner"}} },
		"no creator":   func(g *BrainGraph) { g.Edges = []BrainEdge{{"e1", "a", "b", "next", "", ""}} },
		"self":         func(g *BrainGraph) { g.Edges = []BrainEdge{{"e1", "a", "a", "supports", "", "owner"}} },
		"missing src":  func(g *BrainGraph) { g.Nodes[0].BlockID = "lesson:lessons/nope.md" },
		"dup node":     func(g *BrainGraph) { g.Nodes[1].ID = "a" },
		"missing node": func(g *BrainGraph) { g.Edges = []BrainEdge{{"e1", "a", "zzz", "next", "", "owner"}} },
	}
	for name, mut := range bad {
		g := base()
		mut(&g)
		if r := ValidateBrain(g, c.BrainBlocks(nil)); r.Status != "FAIL" {
			t.Errorf("%s: want FAIL, got %+v", name, r)
		}
	}
	// A source that changed on GitHub makes the block STALE (a warning), never silently re-bound.
	g := base()
	gh.set("Mira/home@live:"+pulsePath, threeCheckPulseLesson+"\n<!-- changed -->\n")
	c.DiscoverLessons()
	r := ValidateBrain(g, c.BrainBlocks(nil))
	if r.Status != "PASS" || len(r.Stale) != 1 || r.Stale[0] != "a" {
		t.Fatalf("stale source: %+v", r)
	}
}

func TestAskWithBrainSendsSavedRevisionReadOnly(t *testing.T) {
	_, eng, _, dir, c := brainCore(t)
	if _, _, err := c.AskWithBrain("which first?"); err == nil {
		t.Fatal("asking before any saved revision must be refused")
	}
	lesson := blockOf(c, "lesson:"+pulsePath)
	skill := blockOf(c, "skill:"+skillListLessons)
	g := BrainGraph{Nodes: []BrainNode{{ID: "n1", BlockID: lesson.ID, Pinned: lesson.Revision}, {ID: "n2", BlockID: skill.ID, Pinned: skill.Revision, X: 300}},
		Edges: []BrainEdge{{ID: "e1", From: "n1", To: "n2", Relation: "uses", Reason: "list first, then study", Creator: "owner"}},
		Notes: []BrainNote{{ID: "x1", Title: "Ask before running", Text: "always ask the owner"}}}
	g.Nodes = append(g.Nodes, BrainNode{ID: "n3", BlockID: "note:x1", Pinned: "note", X: 600})
	if _, rep, err := c.SaveBrain(g, "owner", "map"); err != nil {
		t.Fatalf("%v %+v", err, rep)
	}
	stop := make(chan struct{})
	defer close(stop)
	go c.think(stop)
	if _, rev, err := c.AskWithBrain("Which block should you use first to find a lesson?"); err != nil || rev != 1 {
		t.Fatalf("ask: %v rev=%d", err, rev)
	}
	deadline := time.Now().Add(5 * time.Second)
	for eng.count() == 0 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	eng.mu.Lock()
	req := eng.reqs[0]
	eng.mu.Unlock()
	ctx := req["context"].(string)
	for _, want := range []string{"DEVELOPMENT BRAIN MAP, revision 1", "ADVISORY and READ-ONLY", "My work pulse", "--USES--> @lesson-workspace.list-lessons", "list first, then study", "Ask before running"} {
		if !strings.Contains(ctx, want) {
			t.Fatalf("brain context missing %q", want)
		}
	}
	if a, ok := req["actions"].([]any); ok && len(a) != 0 {
		t.Fatal("brain question offered actions")
	}
	b, _ := os.ReadFile(filepath.Join(dir, "conversation.jsonl"))
	if !strings.Contains(string(b), `"context":"brain revision 1"`) {
		t.Fatalf("conversation log doesn't record which brain revision TASS saw:\n%s", b)
	}
}
