package tass

// @ skills (Thought-Processor #72): the first real, owner-supervised capability TASS can use.
//
// A skill is a fixed operation compiled into this Core, described by a structured descriptor.
// It is never code from a lesson, notebook, GitHub or the model. One skill exists:
//
//	@lesson-workspace.list-lessons  — list the lesson files in TASS's local, read-only lesson
//	                                  workspace (the Core's copy of the lessons folder from GitHub).
//
// Flow, every time:
//  1. Request: the owner names the skill with @ in local Talk (or TASS's own reply proposes it).
//     The Core resolves it against the registry (no model text is trusted as a descriptor),
//     checks Identity skill.discover, and records a receipt AWAITING_APPROVAL. Nothing runs.
//  2. Approve: only the owner's local page. Identity checks skill.run:<id> at that moment.
//     Denied → DENIED, nothing runs. Allowed → RUNNING is persisted BEFORE the operation,
//     the fixed function runs once, and DONE/FAILED is persisted with the real output.
//  3. Repeating approve, a pulse, or a restart never runs a receipt again; a receipt left
//     RUNNING by a crash becomes INTERRUPTED and needs a new request.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/Arameair/Thought-Processor/core/lessons"
)

// BuildVersion is the Core build recorded in every receipt (set by main from -ldflags).
var BuildVersion = "dev"

type SkillDescriptor struct {
	ID           string   `json:"id"`
	Version      string   `json:"version"`
	Owner        string   `json:"owner"`
	Purpose      string   `json:"purpose"`
	Inputs       []string `json:"inputs"`
	Permissions  []string `json:"permissions"`
	Result       string   `json:"result"`
	Restrictions []string `json:"restrictions"`
	Expect       string   `json:"expect"`
	topics       []string // words in a request that make this the matching skill for a bare "@"
}

const skillListLessons = "lesson-workspace.list-lessons"

var skillRegistry = []SkillDescriptor{{
	ID: skillListLessons, Version: "1", Owner: "Thought-Processor Core (built in)",
	Purpose:     "List the lesson files in my local lesson workspace: the read-only copy of the lessons folder synced from GitHub.",
	Inputs:      []string{},
	Permissions: []string{"skill.discover", "skill.run:" + skillListLessons},
	Result:      "one line per lesson file: path, kind (markdown|notebook), title, checks, bytes, sha256; plus the count",
	Restrictions: []string{
		"fixed Go function inside the Core: no shell, subprocess, interpreter, notebook kernel or model-written code",
		"reads only <data>/lessons (opened read-only); regular .md/.ipynb files; no symlinks, writes or network",
		"at most 500 files, 1 MiB each, 5 seconds",
	},
	Expect: "the files listed are exactly the lessons the Learning card has on disk (same count)",
	topics: []string{"lesson", "lessons"},
}}

func skillByID(id string) *SkillDescriptor {
	for i := range skillRegistry {
		if skillRegistry[i].ID == id {
			return &skillRegistry[i]
		}
	}
	return nil
}

// SkillReceipt is one request and, if approved and allowed, its one execution.
type SkillReceipt struct {
	ReceiptID       string     `json:"receipt_id"`
	Skill           string     `json:"skill"`
	SkillVersion    string     `json:"skill_version"`
	CoreVersion     string     `json:"core_version"`
	Request         string     `json:"request"`
	SelectedBy      string     `json:"selected_by"` // "owner @mention" | "TASS reply (cycle …)"
	Why             string     `json:"why"`
	State           string     `json:"state"` // AWAITING_APPROVAL | DENIED | RUNNING | DONE | FAILED | INTERRUPTED | CANCELLED
	CreatedAt       time.Time  `json:"created_at"`
	DecidedAt       *time.Time `json:"decided_at,omitempty"`
	FinishedAt      *time.Time `json:"finished_at,omitempty"`
	DiscoverDecided string     `json:"discover_decision,omitempty"`
	RunDecided      string     `json:"run_decision,omitempty"`
	Executed        bool       `json:"executed"`
	Workspace       string     `json:"workspace,omitempty"`
	ExitCode        *int       `json:"exit_code,omitempty"`
	Output          string     `json:"output,omitempty"`
	OutputSHA256    string     `json:"output_sha256,omitempty"`
	Items           int        `json:"items"`
	Expected        string     `json:"expected,omitempty"`
	Observed        string     `json:"observed,omitempty"`
	ExpectationMet  *bool      `json:"expectation_met,omitempty"`
	Error           string     `json:"error,omitempty"`
	ElapsedMS       int64      `json:"elapsed_ms"`
}

type skillState struct {
	Receipts []*SkillReceipt `json:"receipts"`
}

const maxSkillReceipts = 200

func (c *Core) loadSkills() {
	c.skillMu.Lock()
	defer c.skillMu.Unlock()
	c.skills = skillState{}
	if b, err := os.ReadFile(c.file("skills.json")); err == nil {
		_ = json.Unmarshal(b, &c.skills)
	}
	changed := false
	for _, r := range c.skills.Receipts {
		if r.State == "RUNNING" { // crashed mid-run: never re-run on its own
			now := c.now().UTC()
			r.State, r.FinishedAt, r.Error = "INTERRUPTED", &now, "the Core stopped while this ran; it was not run again. Make a new request."
			changed = true
		}
	}
	if changed {
		c.saveSkillsLocked()
	}
}

func (c *Core) saveSkillsLocked() error {
	if len(c.skills.Receipts) > maxSkillReceipts {
		c.skills.Receipts = append([]*SkillReceipt(nil), c.skills.Receipts[len(c.skills.Receipts)-maxSkillReceipts:]...)
	}
	b, _ := json.MarshalIndent(c.skills, "", "  ")
	tmp := c.file("skills.json.tmp")
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, c.file("skills.json"))
}

// logSkillLocked appends the receipt's current state to an append-only audit log.
func (c *Core) logSkillLocked(r *SkillReceipt) {
	c.appendLog("skill_receipts.jsonl", r)
}

func (c *Core) receiptLocked(id string) *SkillReceipt {
	for _, r := range c.skills.Receipts {
		if r.ReceiptID == id {
			return r
		}
	}
	return nil
}

var ErrNoReceipt = errors.New("no such skill receipt")

// RequestSkill records one supervised request. It never runs anything.
func (c *Core) RequestSkill(id, request, selectedBy, why string) (SkillReceipt, error) {
	d := skillByID(id)
	if d == nil {
		return SkillReceipt{}, fmt.Errorf("no registered skill @%s", id)
	}
	if c.cfg.IdentitySvc == nil || c.cfg.ChatOnly {
		return SkillReceipt{}, errors.New("skills are unavailable in forced safety mode")
	}
	r := &SkillReceipt{ReceiptID: "skr_" + c.now().UTC().Format("20060102T150405") + "_" + hash(fmt.Sprint(c.now().UnixNano(), id, request))[:8],
		Skill: d.ID, SkillVersion: d.Version, CoreVersion: BuildVersion, Request: oneLine(request, 300),
		SelectedBy: selectedBy, Why: why, CreatedAt: c.now().UTC(), Expected: d.Expect}
	ok, decision := c.allowedNow("skill.discover")
	r.DiscoverDecided = decision
	if ok {
		r.State = "AWAITING_APPROVAL"
	} else {
		now := c.now().UTC()
		r.State, r.DecidedAt, r.Error = "DENIED", &now, "Identity did not allow skill.discover; nothing ran. "+identitySkillFix(decision)
	}
	c.skillMu.Lock()
	c.skills.Receipts = append(c.skills.Receipts, r)
	err := c.saveSkillsLocked()
	c.logSkillLocked(r)
	out := *r
	c.skillMu.Unlock()
	if err != nil {
		return out, fmt.Errorf("skill request not saved: %w", err)
	}
	return out, nil
}

// ApproveSkill is the owner's explicit approval of one receipt. It runs the fixed operation at
// most once per receipt: any later call returns the existing receipt unchanged.
func (c *Core) ApproveSkill(receiptID string) (SkillReceipt, error) {
	if c.cfg.IdentitySvc == nil || c.cfg.ChatOnly {
		return SkillReceipt{}, errors.New("skills are unavailable in forced safety mode")
	}
	c.skillRun.Lock() // one execution at a time
	defer c.skillRun.Unlock()
	c.skillMu.Lock()
	r := c.receiptLocked(receiptID)
	if r == nil {
		c.skillMu.Unlock()
		return SkillReceipt{}, ErrNoReceipt
	}
	if r.State != "AWAITING_APPROVAL" {
		out := *r
		c.skillMu.Unlock()
		return out, nil // idempotent: already decided, never run twice
	}
	d := skillByID(r.Skill)
	c.skillMu.Unlock()

	ok, decision := c.allowedNow("skill.run:" + r.Skill) // point of use
	now := c.now().UTC()
	c.skillMu.Lock()
	r.RunDecided, r.DecidedAt = decision, &now
	if !ok || d == nil {
		r.State, r.FinishedAt = "DENIED", &now
		r.Error = "Identity did not allow skill.run:" + r.Skill + "; nothing ran. " + identitySkillFix(decision)
		c.saveSkillsLocked()
		c.logSkillLocked(r)
		out := *r
		c.skillMu.Unlock()
		c.event("problem", fmt.Sprintf("Skill @%s (receipt %s) DENIED by Identity: nothing ran.", r.Skill, r.ReceiptID))
		return out, nil
	}
	r.State = "RUNNING"
	if err := c.saveSkillsLocked(); err != nil { // must be durable before anything runs
		r.State, r.Error = "FAILED", "could not record RUNNING before the run, so it did not run: "+err.Error()
		out := *r
		c.skillMu.Unlock()
		return out, errors.New(r.Error)
	}
	c.logSkillLocked(r)
	c.skillMu.Unlock()

	start := c.now()
	res := c.runListLessons()
	fin := c.now().UTC()

	c.skillMu.Lock()
	r.Executed, r.FinishedAt, r.ElapsedMS = true, &fin, fin.Sub(start).Milliseconds()
	r.Workspace, r.ExitCode, r.Output, r.Items = res.workspace, &res.exit, res.output, res.items
	sum := sha256.Sum256([]byte(res.output))
	r.OutputSHA256 = hex.EncodeToString(sum[:])
	if res.err != "" {
		r.State, r.Error = "FAILED", res.err
	} else {
		r.State, r.Observed, r.ExpectationMet = "DONE", res.observed, &res.met
	}
	c.saveSkillsLocked()
	c.logSkillLocked(r)
	out := *r
	c.skillMu.Unlock()
	c.event("tass", skillReport(out))
	return out, nil
}

// CancelSkill: the owner declines a waiting request. Nothing runs.
func (c *Core) CancelSkill(receiptID string) (SkillReceipt, error) {
	c.skillMu.Lock()
	defer c.skillMu.Unlock()
	r := c.receiptLocked(receiptID)
	if r == nil {
		return SkillReceipt{}, ErrNoReceipt
	}
	if r.State == "AWAITING_APPROVAL" {
		now := c.now().UTC()
		r.State, r.DecidedAt, r.FinishedAt = "CANCELLED", &now, &now
		c.saveSkillsLocked()
		c.logSkillLocked(r)
	}
	return *r, nil
}

type listResult struct {
	workspace, output, observed, err string
	exit, items                      int
	met                              bool
}

// runListLessons is the whole skill: read-only listing of the local lesson workspace.
func (c *Core) runListLessons() listResult {
	root := c.file("lessons")
	res := listResult{workspace: "<data>/lessons (local copy of " + c.cfg.Repo + "@" + c.cfg.Branch + ":" + c.lessonsDir() + "/)"}
	deadline := c.now().Add(5 * time.Second)
	type row struct{ path, kind, title string; checks int; size int64; sum string }
	var rows []row
	info, err := os.Lstat(root)
	if err != nil || !info.IsDir() {
		res.exit, res.err = 1, "lesson workspace not present yet: press Find lessons first"
		res.output = "ERROR: " + res.err
		return res
	}
	walkErr := filepath.WalkDir(root, func(p string, de fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if c.now().After(deadline) {
			return errors.New("time limit (5 s) reached")
		}
		if de.Type()&fs.ModeSymlink != 0 {
			return nil // never follow links out of the workspace
		}
		if de.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(root, p)
		rel = filepath.ToSlash(rel)
		low := strings.ToLower(rel)
		if !de.Type().IsRegular() || strings.Contains(rel, ".ipynb_checkpoints") || !(strings.HasSuffix(low, ".md") || strings.HasSuffix(low, ".ipynb")) {
			return nil
		}
		if len(rows) >= 500 {
			return errors.New("more than 500 lesson files")
		}
		fi, err := de.Info()
		if err != nil || fi.Size() > 1<<20 {
			return fmt.Errorf("%s: unreadable or larger than 1 MiB", rel)
		}
		f, err := os.OpenFile(p, os.O_RDONLY, 0)
		if err != nil {
			return err
		}
		b := make([]byte, fi.Size())
		_, err = f.Read(b)
		f.Close()
		if err != nil && fi.Size() > 0 {
			return err
		}
		s := sha256.Sum256(b)
		r := row{path: rel, kind: "markdown", size: fi.Size(), sum: hex.EncodeToString(s[:])[:16]}
		l := lessons.Parse(string(b))
		if strings.HasSuffix(low, ".ipynb") {
			r.kind, l = "notebook", lessons.Notebook(string(b))
		}
		r.title, r.checks = firstNonEmpty(l.Title, filepath.Base(rel)), len(l.Checks())
		rows = append(rows, r)
		return nil
	})
	if walkErr != nil {
		res.exit, res.err = 1, walkErr.Error()
		res.output = "ERROR: " + walkErr.Error()
		return res
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].path < rows[j].path })
	var b strings.Builder
	for _, r := range rows {
		fmt.Fprintf(&b, "%s\t%s\t%q\tchecks=%d\tbytes=%d\tsha256=%s…\n", r.path, r.kind, r.title, r.checks, r.size, r.sum)
	}
	fmt.Fprintf(&b, "count=%d\n", len(rows))
	res.output, res.items = b.String(), len(rows)

	// Compare with what the Learning card says is on disk (not GONE).
	onCard := map[string]bool{}
	for _, l := range c.Learning().Lessons {
		if l.Status != "GONE" {
			onCard[l.Path] = true
		}
	}
	missing, extra := 0, 0
	listed := map[string]bool{}
	for _, r := range rows {
		listed[r.path] = true
		if !onCard[r.path] {
			extra++
		}
	}
	for p := range onCard {
		if !listed[p] {
			missing++
		}
	}
	res.met = missing == 0 && extra == 0
	res.observed = fmt.Sprintf("listed %d file(s); Learning card has %d; not on the card: %d; on the card but not on disk: %d", len(rows), len(onCard), extra, missing)
	return res
}

func identitySkillFix(decision string) string {
	switch {
	case strings.Contains(decision, "LOCKED"):
		return "Unlock TASS on Identity's page with your passphrase."
	case strings.Contains(decision, "no grant"):
		return `On Identity's page, under "Skills TASS may run", press "Enable this skill".`
	default:
		return "Check Identity's page."
	}
}

func skillReport(r SkillReceipt) string {
	var b strings.Builder
	switch r.State {
	case "DONE":
		met := "NOT MET"
		if r.ExpectationMet != nil && *r.ExpectationMet {
			met = "MET"
		}
		fmt.Fprintf(&b, "I ran @%s v%s once (receipt %s, %d ms). %s\n", r.Skill, r.SkillVersion, r.ReceiptID, r.ElapsedMS, r.RunDecided)
		fmt.Fprintf(&b, "Output (sha256 %s…):\n%s", r.OutputSHA256[:16], r.Output)
		fmt.Fprintf(&b, "Expected: %s\nObserved: %s → %s", r.Expected, r.Observed, met)
	case "FAILED":
		fmt.Fprintf(&b, "@%s FAILED (receipt %s): %s", r.Skill, r.ReceiptID, r.Error)
	default:
		fmt.Fprintf(&b, "@%s is %s (receipt %s). %s", r.Skill, r.State, r.ReceiptID, r.Error)
	}
	return b.String()
}

// Skills is what the page shows: the registry, Identity's current answer per skill, receipts.
type SkillsStatus struct {
	Registry []SkillDescriptor `json:"registry"`
	Grants   map[string]string `json:"grants"` // skill id → "GRANTED" | "NOT GRANTED: why"
	Receipts []SkillReceipt    `json:"receipts"`
}

func (c *Core) Skills(checkGrants bool) SkillsStatus {
	out := SkillsStatus{Registry: skillRegistry, Grants: map[string]string{}}
	if checkGrants && c.cfg.IdentitySvc != nil {
		for _, d := range skillRegistry {
			if c.grantedNoAudit("skill.run:" + d.ID) {
				out.Grants[d.ID] = "GRANTED"
			} else {
				out.Grants[d.ID] = "NOT GRANTED"
			}
		}
	}
	c.skillMu.Lock()
	for i := len(c.skills.Receipts) - 1; i >= 0 && len(out.Receipts) < 30; i-- {
		out.Receipts = append(out.Receipts, *c.skills.Receipts[i])
	}
	c.skillMu.Unlock()
	return out
}

// grantedNoAudit reads Identity's public state (no decision is recorded): display only.
// Every real run still goes through allowedNow at the point of use.
func (c *Core) grantedNoAudit(action string) bool {
	st := c.cfg.IdentitySvc.State()
	if st.State != "UNLOCKED" || st.Entity == nil {
		return false
	}
	for _, g := range st.Entity.Grants {
		if g == action {
			return true
		}
	}
	return false
}

// skillsForContext: what TASS knows about his skills and their latest real receipts.
func (c *Core) skillsForContext() string {
	var b strings.Builder
	b.WriteString("\nSKILLS YOU CAN ASK TO USE (built into your Core; write the @name in your reply to ask; each run needs Identity's grant and your owner's approval, and you only know the result after the receipt):\n")
	for _, d := range skillRegistry {
		fmt.Fprintf(&b, "- @%s — %s\n", d.ID, d.Purpose)
	}
	c.skillMu.Lock()
	n := 0
	for i := len(c.skills.Receipts) - 1; i >= 0 && n < 3; i-- {
		r := c.skills.Receipts[i]
		if r.State == "DONE" || r.State == "DENIED" || r.State == "FAILED" {
			if n == 0 {
				b.WriteString("YOUR LATEST REAL SKILL RECEIPTS (only these happened):\n")
			}
			fmt.Fprintf(&b, "- %s @%s %s at %s: %s\n", r.ReceiptID, r.Skill, r.State, r.CreatedAt.Format("2006-01-02 15:04"), oneLine(firstNonEmpty(r.Observed, r.Error), 200))
			n++
		}
	}
	c.skillMu.Unlock()
	return b.String()
}

var skillMention = regexp.MustCompile(`(?i)(^|[\s(])@([a-z0-9][a-z0-9.\-]*[a-z0-9])?`)

// resolveSkill finds the registered skill a message asks for. A named @id must match the
// registry exactly; a bare "@" (or an unknown name) matches only through the skill's topics.
func resolveSkill(text string) (id, why string, mentioned bool) {
	low := strings.ToLower(text)
	for _, m := range skillMention.FindAllStringSubmatch(text, -1) {
		mentioned = true
		name := strings.ToLower(m[2])
		if name == "identity-test" {
			continue // the read-only reference lookup, handled elsewhere
		}
		if d := skillByID(name); d != nil {
			return d.ID, "you named @" + d.ID, true
		}
	}
	if !mentioned || identityTestMention.MatchString(text) && len(skillMention.FindAllString(text, -1)) == 1 {
		return "", "", mentioned
	}
	for _, d := range skillRegistry {
		for _, t := range d.topics {
			if regexp.MustCompile(`\b` + t + `\b`).MatchString(low) {
				return d.ID, fmt.Sprintf("you asked about %q with @, and @%s is the only registered skill for that: %s", t, d.ID, d.Purpose), true
			}
		}
	}
	return "", "", true
}

// handleSkillChat: an owner's local message that uses @ for a skill creates a supervised
// request. It never runs the skill; approval is a separate owner action on the page.
func (c *Core) handleSkillChat(m Incoming) bool {
	if m.From != "local" || c.cfg.IdentitySvc == nil {
		return false
	}
	id, why, mentioned := resolveSkill(m.Text)
	if !mentioned {
		return false
	}
	if id == "" { // no registered skill matches: ordinary conversation (unchanged behavior)
		return false
	}
	r, err := c.RequestSkill(id, m.Text, "owner @mention", why)
	if err != nil {
		c.event("problem", "Skill request: "+err.Error())
		return true
	}
	c.event("tass", skillProposal(r))
	return true
}

func skillProposal(r SkillReceipt) string {
	d := skillByID(r.Skill)
	if r.State == "DENIED" {
		return fmt.Sprintf("I would use @%s for that, but %s (receipt %s)", r.Skill, r.Error, r.ReceiptID)
	}
	return fmt.Sprintf("I can do that with @%s v%s: %s\nWhy this skill: %s.\nIt needs Identity grant skill.run:%s and your approval. Nothing has run yet: receipt %s is waiting in Skills → Approve run.",
		d.ID, d.Version, d.Purpose, r.Why, d.ID, r.ReceiptID)
}

// proposeSkillFromReply: if TASS's own (model) reply names a registered @skill, record a
// supervised request so the owner can approve it. The model's text is never a run.
func (c *Core) proposeSkillFromReply(reply, cycleID string) {
	for _, m := range skillMention.FindAllStringSubmatch(reply, -1) {
		if d := skillByID(strings.ToLower(m[2])); d != nil {
			r, err := c.RequestSkill(d.ID, "TASS proposed it in his reply", "TASS reply (cycle "+cycleID+")",
				"TASS named @"+d.ID+" in his own reply")
			if err == nil {
				c.event("sync", skillProposal(r))
			}
			return
		}
	}
}
