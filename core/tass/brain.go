package tass

// Development platform: the shared "brain" where the owner and TASS arrange what TASS knows
// and can do (Arameair/Development ideas/tass-development-platform, milestones M1–M3, and
// the M4 skill path from #72). Ported from the Development repo's Go Brain Builder prototype
// and connected to this Core's real data instead of a fixture folder.
//
//	A lesson cell teaches or describes. A block references it. An edge proposes a relationship.
//	A registered skill offers an operation. Identity grants execution. A receipt shows what happened.
//
// Blocks come from real state: lessons (with Core's learning status), notebook cells (read-only
// data, never run), registered skills (Identity grant + last receipt) and owner notes. Nodes pin
// the block's source revision; when the source changes the node shows STALE until the owner
// re-pins it. The graph is versioned: every save is a new immutable revision; restoring an old
// one makes a new revision. Nothing here grants a permission or runs anything.

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type BrainBlock struct {
	ID       string `json:"id"`   // stable: lesson:<path> | cell:<path>:<cell id> | skill:<id> | note:<id>
	Kind     string `json:"kind"` // lesson | code-cell | text-cell | skill | note
	Title    string `json:"title"`
	Source   string `json:"source"` // repository path, skill id or "owner note"
	CellID   string `json:"cell_id,omitempty"`
	Revision string `json:"revision"` // GitHub blob SHA, notebook content hash, skill version, note revision
	Purpose  string `json:"purpose"`  // why use it
	Expect   string `json:"expect"`
	Preview  string `json:"preview,omitempty"`
	Status   string `json:"status"`         // LEARNED | PASSED | NEW … | DISCOVERED | GRANTED | NOT GRANTED | NOTE
	Runnable string `json:"runnable"`       // what can actually run it, truthfully
	Last     string `json:"last,omitempty"` // last real receipt / attempt
}

type BrainNode struct {
	ID      string  `json:"id"`
	BlockID string  `json:"block_id"`
	Pinned  string  `json:"pinned"` // block revision when added or last re-pinned
	X       float64 `json:"x"`
	Y       float64 `json:"y"`
}

type BrainEdge struct {
	ID       string `json:"id"`
	From     string `json:"from"`
	To       string `json:"to"`
	Relation string `json:"relation"`
	Reason   string `json:"reason"`
	Creator  string `json:"creator"` // owner | TASS
}

type BrainNote struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Text  string `json:"text"`
}

type BrainGraph struct {
	Revision int         `json:"revision"`
	Parent   int         `json:"parent"`
	SavedAt  time.Time   `json:"saved_at"`
	Actor    string      `json:"actor"`
	Change   string      `json:"change"`
	Nodes    []BrainNode `json:"nodes"`
	Edges    []BrainEdge `json:"edges"`
	Notes    []BrainNote `json:"notes"`
}

type BrainCheck struct {
	Severity string `json:"severity"`
	Text     string `json:"text"`
}

type BrainReport struct {
	Status    string       `json:"status"`
	Checks    []BrainCheck `json:"checks"`
	Stale     []string     `json:"stale"`
	Summary   string       `json:"summary"`
	Execution string       `json:"execution"`
}

// Relations (Development doc 03). None of them authorizes or runs anything.
var brainRelations = map[string]string{
	"next":        "proposed order, not permission to run the next one",
	"supports":    "advisory: this helps with that",
	"alternative": "another possible approach, not both",
	"requires":    "a dependency that must exist first",
	"uses":        "a thought or lesson refers to a skill",
}

// BrainRelations: the relation vocabulary and what each one means.
func BrainRelations() map[string]string { return brainRelations }

const (
	maxBrainNodes = 150
	maxBrainEdges = 300
	maxBrainNotes = 100
)

func (c *Core) brainDir() string { return c.file("brain") }

func (c *Core) loadBrainLocked() BrainGraph {
	var g BrainGraph
	if b, err := os.ReadFile(filepath.Join(c.brainDir(), "current.json")); err == nil {
		_ = json.Unmarshal(b, &g)
	}
	return g
}

// Brain returns the saved graph.
func (c *Core) Brain() BrainGraph {
	c.brainMu.Lock()
	defer c.brainMu.Unlock()
	return c.loadBrainLocked()
}

// BrainBlocks is the live catalog built from the Core's real data.
func (c *Core) BrainBlocks(notes []BrainNote) []BrainBlock {
	var out []BrainBlock
	for _, l := range c.Learning().Lessons {
		if l.Status == "GONE" || strings.HasSuffix(strings.ToLower(l.Path), ".ipynb") {
			continue
		}
		b := BrainBlock{ID: "lesson:" + l.Path, Kind: "lesson", Title: l.Title, Source: l.Path, Revision: l.SHA,
			Purpose: l.Goal, Status: l.Status, Runnable: "studied through Learning (Find → Study → your Approve); never executed"}
		if raw, err := os.ReadFile(filepath.Join(c.file("lessons"), filepath.FromSlash(l.Path))); err == nil {
			b.Preview = clip(string(raw), 1500)
		}
		var checks []string
		for _, r := range l.Results {
			checks = append(checks, fmt.Sprintf("%d. %s", r.N, r.Check))
		}
		b.Expect = fmt.Sprintf("%d checks graded by the Core", l.Checks)
		if len(checks) > 0 {
			b.Expect += ": " + strings.Join(checks, " · ")
		}
		if n := len(l.History); n > 0 {
			h := l.History[n-1]
			b.Last = fmt.Sprintf("job %s · Engine %s · %s", h.JobID, h.State, firstNonEmpty(h.Outcome, "—"))
		}
		out = append(out, b)
	}
	out = append(out, c.notebookBlocks()...)
	sk := c.Skills(true)
	for _, d := range skillRegistry {
		b := BrainBlock{ID: "skill:" + d.ID, Kind: "skill", Title: "@" + d.ID, Source: "built into the Core", Revision: "v" + d.Version,
			Purpose: d.Purpose, Expect: d.Expect, Status: firstNonEmpty(sk.Grants[d.ID], "UNKNOWN"),
			Runnable: "only after Identity grants skill.run:" + d.ID + " and you press Approve run",
			Preview:  "result: " + d.Result + "\nrestrictions:\n- " + strings.Join(d.Restrictions, "\n- ")}
		for _, r := range sk.Receipts {
			if r.Skill == d.ID {
				b.Last = fmt.Sprintf("receipt %s · %s · %s", r.ReceiptID, r.State, firstNonEmpty(r.Observed, r.Error))
				break
			}
		}
		out = append(out, b)
	}
	for _, n := range notes {
		out = append(out, BrainBlock{ID: "note:" + n.ID, Kind: "note", Title: n.Title, Source: "owner note", Revision: "note", // notes live in the graph itself; its revisions are their history
			Purpose: n.Text, Status: "NOTE", Runnable: "a note: nothing to run"})
	}
	return out
}

// notebookBlocks: each notebook cell is a read-only block (Development doc 03).
func (c *Core) notebookBlocks() []BrainBlock {
	var out []BrainBlock
	for _, l := range c.Learning().Lessons {
		if l.Status == "GONE" || !strings.HasSuffix(strings.ToLower(l.Path), ".ipynb") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(c.file("lessons"), filepath.FromSlash(l.Path)))
		if err != nil || len(raw) > 2<<20 {
			continue
		}
		var nb struct {
			Cells []struct {
				CellType string          `json:"cell_type"`
				ID       string          `json:"id"`
				Source   json.RawMessage `json:"source"`
				Metadata struct {
					TASS struct{ Title, Thought, Expect string } `json:"tass"`
				} `json:"metadata"`
			} `json:"cells"`
		}
		if json.Unmarshal(raw, &nb) != nil {
			continue
		}
		for i, cell := range nb.Cells {
			var src string
			var parts []string
			if json.Unmarshal(cell.Source, &parts) == nil {
				src = strings.Join(parts, "")
			} else {
				_ = json.Unmarshal(cell.Source, &src)
			}
			if strings.TrimSpace(src) == "" || (cell.CellType != "code" && cell.CellType != "markdown") {
				continue
			}
			id := cell.ID
			if id == "" {
				id = fmt.Sprintf("index-%d", i)
			}
			kind, title := "text-cell", cell.Metadata.TASS.Title
			if cell.CellType == "code" {
				kind = "code-cell"
			}
			if title == "" {
				title = firstLine(src, fmt.Sprintf("%s cell %d", cell.CellType, i+1))
			}
			out = append(out, BrainBlock{ID: "cell:" + l.Path + ":" + id, Kind: kind, Title: title, Source: l.Path, CellID: id,
				Revision: hash(src)[:12], Purpose: firstNonEmpty(cell.Metadata.TASS.Thought, clip(src, 250)), Expect: cell.Metadata.TASS.Expect,
				Preview: clip(src, 1500), Status: "DISCOVERED",
				Runnable: "NOT EXECUTABLE: notebook code is data here; no registered skill runs it"})
		}
	}
	return out
}

func firstLine(s, fallback string) string {
	for _, l := range strings.Split(s, "\n") {
		l = strings.TrimSpace(strings.TrimLeft(l, "#"))
		if l != "" {
			return clip(l, 80)
		}
	}
	return fallback
}

func clip(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) > n {
		return s[:n] + "…"
	}
	return s
}

// ValidateBrain checks structure and source freshness. It never runs anything.
func ValidateBrain(g BrainGraph, blocks []BrainBlock) BrainReport {
	r := BrainReport{Status: "PASS", Execution: "NOT RUN — structural review only"}
	add := func(level, msg string) {
		r.Checks = append(r.Checks, BrainCheck{level, msg})
		if level == "ERROR" {
			r.Status = "FAIL"
		}
	}
	byID := map[string]BrainBlock{}
	for _, b := range blocks {
		byID[b.ID] = b
	}
	if len(g.Nodes) > maxBrainNodes || len(g.Edges) > maxBrainEdges || len(g.Notes) > maxBrainNotes {
		add("ERROR", fmt.Sprintf("too large: at most %d blocks, %d connections, %d notes", maxBrainNodes, maxBrainEdges, maxBrainNotes))
	}
	nodes := map[string]BrainNode{}
	for _, n := range g.Nodes {
		if n.ID == "" || len(n.ID) > 40 {
			add("ERROR", "block IDs must be nonempty and short")
		} else if _, dup := nodes[n.ID]; dup {
			add("ERROR", "duplicate block "+n.ID)
		}
		nodes[n.ID] = n
		b, ok := byID[n.BlockID]
		switch {
		case !ok:
			add("ERROR", "source no longer exists: "+clip(n.BlockID, 90))
		case n.Pinned != b.Revision:
			r.Stale = append(r.Stale, n.ID)
			add("WARN", fmt.Sprintf("STALE: %q changed since it was placed (was %s, now %s); re-pin it after checking", b.Title, clip(n.Pinned, 12), clip(b.Revision, 12)))
		}
		if n.X < 0 || n.X > 4000 || n.Y < 0 || n.Y > 4000 {
			add("ERROR", "block outside the canvas")
		}
	}
	edgeIDs := map[string]bool{}
	pairs := map[string]bool{}
	next := map[string][]string{}
	for _, e := range g.Edges {
		if e.ID == "" || edgeIDs[e.ID] {
			add("ERROR", "connection IDs must be unique and nonempty")
		}
		edgeIDs[e.ID] = true
		a, okA := nodes[e.From]
		b, okB := nodes[e.To]
		if !okA || !okB {
			add("ERROR", "a connection points to a missing block")
			continue
		}
		if e.From == e.To {
			add("ERROR", "a block can't connect to itself")
		}
		if pairs[e.From+">"+e.To+">"+e.Relation] {
			add("ERROR", "the same connection exists twice")
		}
		pairs[e.From+">"+e.To+">"+e.Relation] = true
		if _, ok := brainRelations[e.Relation]; !ok {
			add("ERROR", "unknown relation: "+e.Relation)
		}
		if e.Creator != "owner" && e.Creator != "TASS" {
			add("ERROR", "a connection must say who made it (owner or TASS)")
		}
		if len(e.Reason) > 500 {
			add("ERROR", "a reason is longer than 500 characters")
		}
		if e.Relation == "uses" && !strings.HasPrefix(b.BlockID, "skill:") {
			add("ERROR", "\"uses\" must point to a skill")
		}
		if e.Relation == "next" {
			next[e.From] = append(next[e.From], e.To)
		}
		_ = a
	}
	state := map[string]int{}
	var cyc func(string) bool
	cyc = func(s string) bool {
		if state[s] == 1 {
			return true
		}
		if state[s] == 2 {
			return false
		}
		state[s] = 1
		for _, t := range next[s] {
			if cyc(t) {
				return true
			}
		}
		state[s] = 2
		return false
	}
	for id := range nodes {
		if cyc(id) {
			add("ERROR", "the NEXT steps go in a circle")
			break
		}
	}
	for _, n := range g.Notes {
		if n.ID == "" || strings.TrimSpace(n.Title) == "" || len(n.Title) > 120 || len(n.Text) > 4000 {
			add("ERROR", "a note needs a title (≤120 chars) and text ≤4000 chars")
		}
	}
	if len(g.Nodes) == 0 {
		add("INFO", "the canvas is empty")
	}
	if r.Status == "PASS" {
		add("INFO", "every block exists and the connections are valid")
	}
	r.Summary = fmt.Sprintf("Checked %d blocks and %d connections: %s (%d stale). Nothing was run.", len(g.Nodes), len(g.Edges), r.Status, len(r.Stale))
	return r
}

var ErrBrainConflict = errors.New("the brain was saved from another tab or by TASS; reload before saving")

// SaveBrain stores a new immutable revision if g is based on the current one.
func (c *Core) SaveBrain(g BrainGraph, actor, change string) (BrainGraph, BrainReport, error) {
	c.brainMu.Lock()
	defer c.brainMu.Unlock()
	cur := c.loadBrainLocked()
	if g.Revision != cur.Revision {
		return cur, BrainReport{}, ErrBrainConflict
	}
	rep := ValidateBrain(g, c.BrainBlocks(g.Notes))
	if rep.Status != "PASS" {
		return cur, rep, errors.New("the brain didn't pass its check; nothing was saved")
	}
	g.Parent, g.Revision, g.SavedAt, g.Actor, g.Change = cur.Revision, cur.Revision+1, c.now().UTC(), actor, clip(change, 200)
	if err := c.writeBrainLocked(g); err != nil {
		return cur, rep, err
	}
	return g, rep, nil
}

func (c *Core) writeBrainLocked(g BrainGraph) error {
	dir := filepath.Join(c.brainDir(), "revisions")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	b, _ := json.MarshalIndent(g, "", "  ")
	rev := filepath.Join(dir, fmt.Sprintf("%06d.json", g.Revision))
	if _, err := os.Stat(rev); err == nil {
		return fmt.Errorf("revision %d already exists; history is never overwritten", g.Revision)
	}
	if err := os.WriteFile(rev, b, 0o400); err != nil {
		return err
	}
	tmp := filepath.Join(c.brainDir(), "current.json.tmp")
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(c.brainDir(), "current.json"))
}

type BrainRevisionInfo struct {
	Revision int       `json:"revision"`
	Parent   int       `json:"parent"`
	SavedAt  time.Time `json:"saved_at"`
	Actor    string    `json:"actor"`
	Change   string    `json:"change"`
	Nodes    int       `json:"nodes"`
	Edges    int       `json:"edges"`
}

func (c *Core) BrainHistory() []BrainRevisionInfo {
	c.brainMu.Lock()
	defer c.brainMu.Unlock()
	files, _ := filepath.Glob(filepath.Join(c.brainDir(), "revisions", "*.json"))
	sort.Sort(sort.Reverse(sort.StringSlice(files)))
	var out []BrainRevisionInfo
	for _, f := range files {
		if len(out) >= 50 {
			break
		}
		var g BrainGraph
		if b, err := os.ReadFile(f); err == nil && json.Unmarshal(b, &g) == nil {
			out = append(out, BrainRevisionInfo{g.Revision, g.Parent, g.SavedAt, g.Actor, g.Change, len(g.Nodes), len(g.Edges)})
		}
	}
	return out
}

// RestoreBrain makes revision `rev` current again as a NEW revision (history stays intact).
func (c *Core) RestoreBrain(rev, basedOn int) (BrainGraph, error) {
	c.brainMu.Lock()
	cur := c.loadBrainLocked()
	b, err := os.ReadFile(filepath.Join(c.brainDir(), "revisions", fmt.Sprintf("%06d.json", rev)))
	c.brainMu.Unlock()
	if err != nil {
		return cur, fmt.Errorf("no revision %d", rev)
	}
	if basedOn != cur.Revision {
		return cur, ErrBrainConflict
	}
	var old BrainGraph
	if err := json.Unmarshal(b, &old); err != nil {
		return cur, err
	}
	old.Revision = cur.Revision
	g, rep, err := c.SaveBrain(old, "owner", fmt.Sprintf("restored revision %d", rev))
	if err != nil && rep.Status == "FAIL" {
		return g, fmt.Errorf("revision %d no longer passes its check (a source is gone): %s", rev, rep.Summary)
	}
	return g, err
}

// brainContext is a bounded, read-only text form of one saved revision for TASS.
func (c *Core) brainContext(g BrainGraph) string {
	blocks := map[string]BrainBlock{}
	for _, b := range c.BrainBlocks(g.Notes) {
		blocks[b.ID] = b
	}
	title := map[string]string{}
	var b strings.Builder
	fmt.Fprintf(&b, "\n\nYOUR DEVELOPMENT BRAIN MAP, revision %d (saved by %s %s). Your owner and you arrange it together. It is ADVISORY and READ-ONLY: it grants no permission and runs nothing; you may disagree and say why. STALE means the source changed after it was placed.\nBLOCKS:\n",
		g.Revision, firstNonEmpty(g.Actor, "?"), g.SavedAt.Format("2006-01-02 15:04"))
	for _, n := range g.Nodes {
		bl, ok := blocks[n.BlockID]
		state := bl.Status
		if !ok {
			state = "MISSING"
		} else if n.Pinned != bl.Revision {
			state += ", STALE"
		}
		title[n.ID] = firstNonEmpty(bl.Title, n.BlockID)
		fmt.Fprintf(&b, "- [%s] %s (%s; %s): %s\n", n.ID, title[n.ID], bl.Kind, state, clip(bl.Purpose, 160))
	}
	b.WriteString("CONNECTIONS:\n")
	for _, e := range g.Edges {
		fmt.Fprintf(&b, "- %s --%s--> %s (by %s): %s\n", title[e.From], strings.ToUpper(e.Relation), title[e.To], e.Creator, clip(e.Reason, 160))
	}
	s := b.String()
	if len(s) > 24<<10 {
		s = s[:24<<10] + "\n…(truncated)"
	}
	return s
}

// AskWithBrain: one question to TASS with the saved brain revision as read-only context (M3).
func (c *Core) AskWithBrain(text string) (string, int, error) {
	g := c.Brain()
	if g.Revision == 0 {
		return "", 0, errors.New("save the brain first: TASS only sees saved revisions")
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return "", 0, errors.New("empty message")
	}
	id := "msg_" + c.now().UTC().Format("20060102T150405") + "_" + hash(text + c.now().String())[:6]
	if err := c.queue(Incoming{Text: text, From: "local", Receipt: id, FileContext: c.brainContext(g),
		FilePath: fmt.Sprintf("brain revision %d", g.Revision)}); err != nil {
		return "", 0, err
	}
	return id, g.Revision, nil
}

func newBrainID(prefix string) string {
	b := make([]byte, 6)
	rand.Read(b)
	return prefix + hex.EncodeToString(b)
}
