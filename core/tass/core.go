// Package tass is the Thought-Processor side of TASS's core.
//
//   - Thinking is local: TASS's tass.md + the message go to the Thought Engine, which runs
//     Nemotron. No internet needed.
//   - GitHub is a separate, asynchronous sync loop (every 20 s, backing off when GitHub is
//     unreachable): Identity's policy, the GitHub copy of tass.md, messages and replies.
//   - Identity's policy decides before anything happens. No policy, no action.
//
// Authority for tass.md: TASS runs on the local copy; the GitHub copy is the reviewed, versioned
// record. The core never writes tass.md itself: it merges edits made on either side, section by
// section. When the same section changed on both sides, TASS runs on the GitHub version of it,
// both versions are kept locally (conflicts/), and a pull request offers the local version:
// Dustin merges or closes it. TASS's own memories go to a local journal (memory.jsonl) with
// provenance, not into tass.md.
package tass

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Arameair/Thought-Processor/core/engine"
	"github.com/Arameair/Thought-Processor/core/github"
	"github.com/Arameair/Thought-Processor/core/identity"
	"github.com/Arameair/Thought-Processor/core/identitysvc"
	"github.com/Arameair/Thought-Processor/core/tassmd"
)

// Marker on everything TASS posts, so he never answers himself.
const Marker = "<!-- tass-core -->"

// GrantFile is the local copy of Identity's policy with when it was last confirmed. The Thought
// Engine reads it too. PROTOTYPE: a static policy file, not a live Identity grant.
// It lives in its own folder so the engine can be given that folder (read-only) and nothing else.
const GrantFile = "grant/identity-grant.json"

// MemoryKeep bounds the local memory journal.
const MemoryKeep = 200

type Config struct {
	ChatOnly  bool          // one-app: deny tool/action execution and work pulse until Issue #50 is fixed
	DataDir    string        // local home: tass.md, tass.base.md, identity-policy.json, state.json, logs
	Identity   string        // owner/repo@branch:path of Identity's policy
	Label      string        // issues with this label are messages to TASS
	SyncEvery  time.Duration // GitHub sync interval when all is well (20 s)
	MaxBackoff time.Duration // longest wait between attempts while GitHub is unreachable
	Messages   bool          // read GitHub messages (off while another program answers them)
	// IdentitySvc: the installation's Identity service (tass-identity). When set, it is the only
	// authority (the prototype policy file isn't used) and GitHub sync is off unless GitHub is set.
	IdentitySvc *identitysvc.Client
	GitHub      bool
}

// grantFile is what GrantFile holds.
type grantFile struct {
	Prototype bool            `json:"prototype"`
	Note      string          `json:"note"`
	Source    string          `json:"source"`
	FetchedAt time.Time       `json:"fetched_at"`
	Policy    json.RawMessage `json:"policy"`
}

type state struct {
	PolicyETag   string         `json:"policy_etag"`
	PolicyHash   string         `json:"policy_hash"`
	FileETag     string         `json:"file_etag"`
	RemoteSHA    string         `json:"remote_sha"`
	IssuesETag   string         `json:"issues_etag"`
	CommentsETag string         `json:"comments_etag"`
	Since        time.Time      `json:"since"`
	SeenIssues   map[int]bool   `json:"seen_issues"`
	SeenComments map[int64]bool `json:"seen_comments"`
	Outbox       []outgoing     `json:"outbox"`
	ConflictKey  string         `json:"conflict_key"`
}

type outgoing struct {
	Issue int    `json:"issue"`
	Body  string `json:"body"`
}

// Event is something to show on the local page.
type Event struct {
	At   time.Time `json:"at"`
	Kind string    `json:"kind"` // you | tass | thought | sync | github | problem
	Text string    `json:"text"`
}

// Incoming is a message for TASS.
type Incoming struct {
	Text    string
	From    string // "local" or "github"
	Issue   int
	Receipt string
}

type Core struct {
	cfg    Config
	gh     *github.Client
	engine *engine.Client
	now    func() time.Time

	fileMu sync.Mutex // tass.md and tass.base.md

	// owned by the sync loop
	st state

	mu        sync.Mutex // fields below
	policy    *identity.Policy
	confirmed time.Time // when the policy was last confirmed from GitHub
	online    bool
	lastSync  time.Time
	nextSync  time.Time
	syncErr   string
	failures  int
	events    []Event
	busy      bool
	lastCycle *engine.Cycle
	cycles    []engine.Cycle
	history   []engine.Message
	pending   int

	inbox   chan Incoming
	replies chan outgoing
}

func New(cfg Config, gh *github.Client, eng *engine.Client) (*Core, error) {
	if err := os.MkdirAll(cfg.DataDir, 0o700); err != nil {
		return nil, err
	}
	if cfg.SyncEvery <= 0 {
		cfg.SyncEvery = 20 * time.Second
	}
	if cfg.MaxBackoff <= 0 {
		cfg.MaxBackoff = 5 * time.Minute
	}
	c := &Core{cfg: cfg, gh: gh, engine: eng, now: time.Now, inbox: make(chan Incoming, 64), replies: make(chan outgoing, 64)}
	if b, err := os.ReadFile(c.file("state.json")); err == nil {
		if err := json.Unmarshal(b, &c.st); err != nil {
			return nil, fmt.Errorf("state.json is damaged (move it aside to start fresh): %w", err)
		}
	}
	if c.st.SeenIssues == nil {
		c.st.SeenIssues = map[int]bool{}
	}
	if c.st.SeenComments == nil {
		c.st.SeenComments = map[int64]bool{}
	}
	if old := c.file("identity-grant.json"); fileExists(old) && !fileExists(c.file(GrantFile)) {
		_ = os.MkdirAll(filepath.Dir(c.file(GrantFile)), 0o700) // moved into its own folder (2026-11-04)
		_ = os.Rename(old, c.file(GrantFile))
	}
	if b, err := os.ReadFile(c.file(GrantFile)); err == nil {
		var g grantFile
		if json.Unmarshal(b, &g) == nil {
			if p, err := identity.Parse(g.Policy); err == nil {
				c.policy, c.confirmed = p, g.FetchedAt // last good policy: permissions hold offline, for a limited time
			}
		}
	}
	c.pending = len(c.st.Outbox)
	return c, nil
}

func fileExists(p string) bool { _, err := os.Stat(p); return err == nil }

// entityName: who this core thinks for. With the Identity service it is the installation's own
// entity (never a built-in name); with the prototype policy file it is the policy's identity.
func (c *Core) entityName() string {
	if c.cfg.IdentitySvc != nil {
		if st := c.cfg.IdentitySvc.State(); st.Entity != nil {
			return st.Entity.Name
		}
		return ""
	}
	if p := c.pol(); p != nil {
		return p.Identity
	}
	return ""
}

// ensureDocument gives a new entity its own starter document (never anyone else's).
func (c *Core) ensureDocument(name string) {
	if name == "" || fileExists(c.file("tass.md")) {
		return
	}
	doc := "# " + name + "\n\n<!-- This entity's own document. It lives only on this computer. -->\n\n## Who I am\n\nI am " + name +
		", a local AI companion. I'm new: I learn who I am with my owner.\n\n## Memory\n\n## Now\n\nJust created.\n"
	c.fileMu.Lock()
	defer c.fileMu.Unlock()
	if !fileExists(c.file("tass.md")) {
		_ = c.write("tass.md", doc)
		c.event("sync", "Created "+name+"'s own starter document on this computer.")
	}
}

func (c *Core) pol() *identity.Policy {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.policy
}

// allowedNow: Identity's policy allows the action AND was confirmed recently enough. A cached policy
// older than its offline limit authorizes nothing (fail closed until GitHub confirms it again).
func (c *Core) allowedNow(action string) (bool, string) {
	if c.cfg.IdentitySvc != nil {
		if action == identity.ReadTass {
			action = "action.read_entity"
		}
		d := c.cfg.IdentitySvc.Authorize(action, "thought-processor")
		if !d.Allow {
			return false, "Identity " + d.DecisionID + ": " + d.Reason
		}
		return true, "Identity " + d.DecisionID + ": " + action + " allowed for " + d.EntityName
	}
	c.mu.Lock()
	p, at := c.policy, c.confirmed
	c.mu.Unlock()
	if p == nil {
		return false, "no Identity policy yet"
	}
	if !p.Allowed(action) {
		return false, "Identity's policy doesn't allow " + action
	}
	if age := c.now().Sub(at); age > p.OfflineLimit() {
		return false, fmt.Sprintf("Identity's policy was last confirmed %s ago, longer than its offline limit (%s)",
			age.Round(time.Minute), p.OfflineLimit())
	}
	return true, fmt.Sprintf("Identity policy (prototype) %s: %s allowed, confirmed %s", p.Identity, action, at.Format("15:04:05"))
}

func (c *Core) file(name string) string { return filepath.Join(c.cfg.DataDir, name) }

func (c *Core) read(name string) (string, bool) {
	b, err := os.ReadFile(c.file(name))
	return string(b), err == nil
}

func (c *Core) write(name, text string) error {
	if err := os.MkdirAll(filepath.Dir(c.file(name)), 0o700); err != nil {
		return err
	}
	tmp := filepath.Join(filepath.Dir(c.file(name)), "."+filepath.Base(name)+".tmp")
	if err := os.WriteFile(tmp, []byte(text), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, c.file(name))
}

func (c *Core) appendLog(name string, v any) {
	p := c.file(name)
	if fi, err := os.Stat(p); err == nil && fi.Size() > 5<<20 {
		_ = os.Rename(p, p+".1")
	}
	f, err := os.OpenFile(p, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer f.Close()
	b, _ := json.Marshal(v)
	f.Write(append(b, '\n'))
}

func (c *Core) saveState() {
	b, _ := json.MarshalIndent(c.st, "", " ")
	if err := c.write("state.json", string(b)); err != nil {
		log.Printf("saving state: %v", err)
	}
}

func (c *Core) event(kind, text string) {
	c.mu.Lock()
	c.events = append(c.events, Event{c.now(), kind, text})
	if len(c.events) > 300 {
		c.events = c.events[len(c.events)-300:]
	}
	c.mu.Unlock()
	if kind != "thought" {
		log.Printf("[%s] %s", kind, oneLine(text, 160))
	}
}

func oneLine(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > n {
		return s[:n] + "…"
	}
	return s
}

func norm(s string) string { return strings.TrimSpace(strings.ReplaceAll(s, "\r\n", "\n")) }

func hash(s string) string {
	h := sha256.Sum256([]byte(norm(s)))
	return hex.EncodeToString(h[:6])
}

// ------------------------------------------------------------------ run ----

// Run thinks about messages as they come, and syncs with GitHub in its own loop, until stop.
func (c *Core) Run(stop <-chan struct{}) {
	if _, ok := c.read("tass.md"); !ok {
		c.event("sync", "No local tass.md yet: it comes from GitHub on the first sync.")
	}
	var wg sync.WaitGroup
	wg.Add(1)
	go func() { defer wg.Done(); c.think(stop) }()
	if c.cfg.IdentitySvc == nil || c.cfg.GitHub {
		wg.Add(1)
		go func() { defer wg.Done(); c.syncLoop(stop) }()
	} else {
		c.event("sync", "GitHub sync is off for this installation: everything stays on this computer.")
	}
	wg.Wait()
}

func (c *Core) syncLoop(stop <-chan struct{}) {
	for {
		wait := c.SyncOnce()
		c.mu.Lock()
		c.nextSync = c.now().Add(wait)
		c.mu.Unlock()
		select {
		case <-stop:
			return
		case <-time.After(wait):
		}
	}
}

// SyncOnce is one GitHub sync. It returns how long to wait before the next one:
// SyncEvery when it worked, doubling (up to MaxBackoff) while GitHub can't be reached.
func (c *Core) SyncOnce() time.Duration {
	for drained := false; !drained; {
		select {
		case o := <-c.replies:
			c.st.Outbox = append(c.st.Outbox, o)
		default:
			drained = true
		}
	}
	start := c.now()
	steps := []string{}
	err := c.syncPolicy()
	if err == nil {
		steps = append(steps, "policy")
		err = c.syncTass()
	}
	if err == nil {
		steps = append(steps, "tass.md")
		err = c.readMessages()
	}
	if err == nil {
		steps = append(steps, "messages")
		err = c.flushOutbox()
	}
	c.saveState()
	c.mu.Lock()
	wasOnline, first := c.online, c.lastSync.IsZero()
	c.lastSync, c.online, c.pending = start, err == nil, len(c.st.Outbox)
	if err == nil {
		c.failures, c.syncErr = 0, ""
	} else {
		c.failures++
		c.syncErr = err.Error()
	}
	failures := c.failures
	c.mu.Unlock()
	rec := map[string]any{"at": start.UTC(), "ok": err == nil, "done": steps, "ms": c.now().Sub(start).Milliseconds()}
	if err != nil {
		rec["error"] = err.Error()
	}
	c.appendLog("sync.jsonl", rec)
	switch {
	case err != nil && (wasOnline || first):
		c.event("problem", "GitHub sync failed: "+err.Error()+". TASS keeps running locally; retrying with backoff.")
	case err == nil && !wasOnline && !first:
		c.event("sync", "GitHub reachable again; synced.")
	}
	if err == nil {
		return c.cfg.SyncEvery
	}
	wait := c.cfg.SyncEvery
	for i := 1; i < failures && wait < c.cfg.MaxBackoff; i++ {
		wait *= 2
	}
	if wait > c.cfg.MaxBackoff {
		wait = c.cfg.MaxBackoff
	}
	return wait
}

// syncPolicy fetches Identity's policy (conditional, so unchanged = no rate-limit cost) and
// records when it was confirmed. A missing policy removes the local copy: nothing is allowed.
func (c *Core) syncPolicy() error {
	repo, rest, _ := strings.Cut(c.cfg.Identity, "@")
	branch, path, _ := strings.Cut(rest, ":")
	etag := c.st.PolicyETag
	if c.pol() == nil {
		etag = ""
	}
	f, got, err := c.gh.GetFile(repo, branch, path, etag)
	if errors.Is(err, github.ErrNotFound) {
		if c.pol() != nil {
			c.event("problem", "Identity's policy is gone from "+c.cfg.Identity+": TASS may no longer act.")
		}
		c.mu.Lock()
		c.policy = nil
		c.mu.Unlock()
		c.st.PolicyETag = ""
		_ = os.Remove(c.file(GrantFile))
		return fmt.Errorf("no Identity policy at %s (Identity hasn't authorized TASS yet)", c.cfg.Identity)
	}
	if err != nil {
		return err
	}
	var raw []byte
	var p *identity.Policy
	if got.NotModified {
		p = c.pol()
		b, err := os.ReadFile(c.file(GrantFile))
		if err != nil {
			c.st.PolicyETag = "" // local copy lost: fetch it again next time
			return err
		}
		var g grantFile
		if err := json.Unmarshal(b, &g); err != nil {
			c.st.PolicyETag = ""
			return err
		}
		raw = g.Policy
	} else {
		if p, err = identity.Parse([]byte(f.Text)); err != nil {
			return err // keep the last good policy (and its confirmation time)
		}
		raw = []byte(f.Text)
	}
	now := c.now().UTC()
	g := grantFile{Prototype: true, Note: "PROTOTYPE: copy of a static policy file, not a live Identity grant",
		Source: c.cfg.Identity, FetchedAt: now, Policy: json.RawMessage(raw)}
	b, _ := json.MarshalIndent(g, "", " ")
	if err := c.write(GrantFile, string(b)); err != nil {
		return err
	}
	c.mu.Lock()
	changed := c.policy == nil || !got.NotModified && hash(string(raw)) != c.st.PolicyHash
	c.policy, c.confirmed = p, now
	c.mu.Unlock()
	if changed {
		c.event("sync", "Identity policy loaded for "+p.Identity+" from "+c.cfg.Identity+" (prototype policy file).")
	}
	c.st.PolicyETag, c.st.PolicyHash = got.ETag, hash(string(raw))
	return nil
}

func (c *Core) syncTass() error {
	p := c.pol()
	if !p.Allowed(identity.SyncTassMD) {
		return nil
	}
	repo, branch, path, err := p.Target()
	if err != nil {
		return err
	}
	c.fileMu.Lock()
	local, haveLocal := c.read("tass.md")
	base, haveBase := c.read("tass.base.md")
	c.fileMu.Unlock()
	etag := ""
	if haveLocal && haveBase {
		etag = c.st.FileETag
	}
	f, got, err := c.gh.GetFile(repo, branch, path, etag)
	if errors.Is(err, github.ErrNotFound) {
		return fmt.Errorf("%s isn't on %s@%s yet: create it there first", path, repo, branch)
	}
	if err != nil {
		return err
	}
	var remote string
	if got.NotModified {
		if norm(local) == norm(base) {
			return nil
		}
		remote = base
	} else {
		c.st.FileETag, c.st.RemoteSHA, remote = got.ETag, f.SHA, f.Text
	}
	if !haveLocal {
		c.fileMu.Lock()
		defer c.fileMu.Unlock()
		if err := c.write("tass.md", remote); err != nil {
			return err
		}
		c.event("sync", "Took tass.md from GitHub: TASS now runs on a local copy.")
		return c.write("tass.base.md", remote)
	}
	if !haveBase {
		base = remote
	}
	merged, mine, conflicts := tassmd.Merge(base, local, remote)
	c.fileMu.Lock()
	cur, _ := c.read("tass.md")
	if norm(cur) != norm(local) {
		c.fileMu.Unlock()
		c.st.FileETag = "" // TASS changed it meanwhile: merge again next time
		return nil
	}
	if norm(merged) != norm(local) {
		if err := c.write("tass.md", merged); err != nil {
			c.fileMu.Unlock()
			return err
		}
		if norm(local) == norm(base) {
			c.event("sync", "tass.md changed on GitHub; TASS now thinks with the new version.")
		}
	}
	err = c.write("tass.base.md", remote)
	c.fileMu.Unlock()
	if err != nil {
		return err
	}
	if norm(merged) != norm(remote) {
		sha, err := c.gh.PutFile(repo, branch, path, merged, c.st.RemoteSHA, "TASS: update tass.md")
		if errors.Is(err, github.ErrConflict) {
			c.st.FileETag = ""
			return nil
		}
		if err != nil {
			return err
		}
		c.st.RemoteSHA, c.st.FileETag = sha, ""
		c.fileMu.Lock()
		err = c.write("tass.base.md", merged)
		c.fileMu.Unlock()
		if err != nil {
			return err
		}
		c.event("sync", "Saved TASS's changes to tass.md on GitHub.")
	}
	if len(conflicts) > 0 {
		return c.askDustin(repo, branch, path, mine, conflicts)
	}
	return nil
}

// askDustin opens one pull request offering TASS's version of the sections both sides changed.
func (c *Core) askDustin(repo, branch, path, mine string, conflicts []string) error {
	key := hash(mine)
	if c.st.ConflictKey == key {
		return nil
	}
	// Keep both sides locally, whatever happens with the pull request.
	stamp := c.now().UTC().Format("20060102T150405Z")
	_ = os.MkdirAll(c.file("conflicts"), 0o700)
	remoteCopy, _ := c.read("tass.base.md")
	_ = os.WriteFile(filepath.Join(c.file("conflicts"), stamp+"-local.md"), []byte(mine), 0o600)
	_ = os.WriteFile(filepath.Join(c.file("conflicts"), stamp+"-github.md"), []byte(remoteCopy), 0o600)
	if !c.pol().Allowed(identity.AskOnConflict) {
		c.st.ConflictKey = key
		c.event("problem", "Both sides changed "+strings.Join(conflicts, ", ")+"; TASS uses the GitHub version "+
			"(Identity doesn't allow him to open a pull request with his version).")
		return nil
	}
	tip, err := c.gh.BranchSHA(repo, branch)
	if err != nil {
		return err
	}
	head := fmt.Sprintf("tass/conflict-%d", c.now().Unix())
	if err := c.gh.CreateBranch(repo, head, tip); err != nil {
		return err
	}
	f, _, err := c.gh.GetFile(repo, head, path, "")
	if err != nil {
		return err
	}
	if _, err := c.gh.PutFile(repo, head, path, mine, f.SHA, "TASS: my version of "+strings.Join(conflicts, ", ")); err != nil {
		return err
	}
	url, err := c.gh.CreatePR(repo, head, branch, "TASS: which version of "+strings.Join(conflicts, ", ")+"?",
		"TASS and GitHub both changed the same part of `"+path+"` while they were apart: **"+
			strings.Join(conflicts, "**, **")+"**.\n\nTASS now runs on **the GitHub version** of those sections. "+
			"This pull request holds **TASS's version**.\n\n- **Merge** it to use TASS's version.\n"+
			"- **Close** it to keep the GitHub version.\n\nEverything else was merged automatically.\n\n"+Marker)
	if err != nil {
		return err
	}
	c.st.ConflictKey = key
	c.event("sync", "Both sides changed "+strings.Join(conflicts, ", ")+": asked Dustin in "+url)
	return nil
}

func (c *Core) readMessages() error {
	p := c.pol()
	if !c.cfg.Messages || !p.Allowed(identity.ReadMessages) {
		return nil
	}
	repo, _, _, err := p.Target()
	if err != nil {
		return err
	}
	first := c.st.Since.IsZero()
	issues, got, err := c.gh.Issues(repo, c.cfg.Label, c.st.IssuesETag)
	if err != nil {
		return err
	}
	if first {
		c.st.Since = c.now().UTC()
	}
	if !got.NotModified {
		c.st.IssuesETag = got.ETag
		for _, is := range issues {
			if is.PullRequest != nil || c.st.SeenIssues[is.Number] {
				continue
			}
			c.st.SeenIssues[is.Number] = true
			if first || !p.Obeys(is.User.Login) || strings.Contains(is.Body, "<!-- ") {
				continue // from before TASS started, or not someone Identity says TASS obeys
			}
			c.queue(Incoming{Text: strings.TrimSpace(is.Title + "\n\n" + is.Body), From: "github", Issue: is.Number})
		}
	}
	comments, got, err := c.gh.Comments(repo, c.st.Since.Add(-2*time.Second), c.st.CommentsETag)
	if err != nil {
		return err
	}
	if got.NotModified {
		return nil
	}
	c.st.CommentsETag = got.ETag
	for _, cm := range comments {
		if c.st.SeenComments[cm.ID] {
			continue
		}
		c.st.SeenComments[cm.ID] = true
		if cm.CreatedAt.After(c.st.Since) {
			c.st.Since = cm.CreatedAt
		}
		if !c.st.SeenIssues[cm.IssueNumber()] || !p.Obeys(cm.User.Login) ||
			strings.Contains(cm.Body, Marker) || strings.Contains(cm.Body, "<!-- identity-liaison") {
			continue
		}
		c.queue(Incoming{Text: strings.TrimSpace(cm.Body), From: "github", Issue: cm.IssueNumber()})
	}
	if len(c.st.SeenComments) > 2000 {
		c.st.SeenComments = map[int64]bool{}
	}
	return nil
}

// ErrBusy: the inbox is full; the message was NOT accepted (Thought-Processor #46).
var ErrBusy = errors.New("TASS is busy: too many messages are waiting; try again shortly")

func (c *Core) queue(m Incoming) error {
	select {
	case c.inbox <- m:
		if m.From == "github" {
			c.event("github", fmt.Sprintf("Message from GitHub, issue #%d", m.Issue))
		}
		return nil
	default:
		c.event("problem", "A message was refused: too many waiting.")
		return ErrBusy
	}
}

func (c *Core) flushOutbox() error {
	if len(c.st.Outbox) == 0 {
		return nil
	}
	p := c.pol()
	if !p.Allowed(identity.Reply) {
		c.event("problem", fmt.Sprintf("%d reply(s) not posted: Identity doesn't allow replies on GitHub.", len(c.st.Outbox)))
		c.st.Outbox = nil
		return nil
	}
	repo, _, _, err := p.Target()
	if err != nil {
		return err
	}
	for len(c.st.Outbox) > 0 {
		o := c.st.Outbox[0]
		if err := c.gh.Comment(repo, o.Issue, o.Body); err != nil && !errors.Is(err, github.ErrNotFound) {
			return err
		}
		c.st.Outbox = c.st.Outbox[1:]
		c.saveState()
		c.event("github", fmt.Sprintf("Posted TASS's reply on issue #%d", o.Issue))
	}
	return nil
}

// Say is a message from the local page. It never needs the internet. It returns a receipt ID,
// or ErrBusy (nothing accepted) when the inbox is full.
func (c *Core) Say(text string) (string, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return "", errors.New("empty message")
	}
	id := "msg_" + c.now().UTC().Format("20060102T150405") + "_" + hash(text + c.now().String())[:6]
	if err := c.queue(Incoming{Text: text, From: "local", Receipt: id}); err != nil {
		return "", err
	}
	return id, nil
}

// SubmitTask queues work for the coordination pulse (it runs on a pulse, not right away).
func (c *Core) SubmitTask(text string) (string, error) {
	if c.cfg.ChatOnly {
		return "", errors.New("CHAT_ONLY: pulse tasks disabled pending independent integrity tests (Issue #50)")
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return "", errors.New("empty task")
	}
	name := c.entityName()
	if ok, why := c.allowedNow("pulse.run"); !ok {
		return "", fmt.Errorf("not queued: %s", why)
	}
	c.ensureDocument(name)
	c.fileMu.Lock()
	doc, _ := c.read("tass.md")
	c.fileMu.Unlock()
	id := "job_" + c.now().UTC().Format("20060102T150405") + "_" + hash(text + c.now().String())[:6]
	if err := c.engine.SubmitJob(engine.Job{JobID: id, Identity: name, Request: text, Context: doc, Actions: Actions}); err != nil {
		return "", err
	}
	c.event("sync", "Task "+id+" queued for the pulse: "+text)
	return id, nil
}

// PulseControl: start | pause | resume | step, passed to the Thought Engine.
func (c *Core) PulseControl(action string) (map[string]any, error) {
	if c.cfg.ChatOnly && action != "pause" {
		return nil, errors.New("CHAT_ONLY: autonomous pulse controls disabled pending Issue #50")
	}
	return c.engine.PulseControl(action)
}

// ------------------------------------------------------------- thinking ----

var (
	rememberRe = regexp.MustCompile(`(?im)^[ \t*_>-]*REMEMBER[*_ ]*:[*_ ]*(.+?)[*_ ]*$`)
	nowRe      = regexp.MustCompile(`(?im)^[ \t*_>-]*NOW[*_ ]*:[*_ ]*(.+?)[*_ ]*$`)
)

// Actions TASS may propose in a cycle. They only read his own local tass.md; Identity decides
// (action.read_tass) at the moment they would run.
var Actions = []engine.Action{
	{Name: "tass_sections", Description: `number of "## " sections in your tass.md`},
	{Name: "tass_memory_lines", Description: `number of lines starting with "- " in the "## Memory" section of your tass.md`},
	{Name: "tass_words", Description: "number of words in your tass.md"},
}

func measure(action, doc string) (float64, error) {
	switch action {
	case "tass_sections":
		n := 0
		for _, sec := range tassmd.Parse(doc) {
			if sec.Name != "" {
				n++
			}
		}
		return float64(n), nil
	case "tass_memory_lines":
		n := 0
		for _, sec := range tassmd.Parse(doc) {
			if sec.Name == tassmd.MemorySection {
				for _, l := range strings.Split(sec.Body, "\n") {
					if strings.HasPrefix(strings.TrimSpace(l), "- ") {
						n++
					}
				}
			}
		}
		return float64(n), nil
	case "tass_words":
		return float64(len(strings.Fields(doc))), nil
	}
	return 0, fmt.Errorf("unknown action %q", action)
}

const howToTalk = `You are TASS, talking with Dustin. Be short, honest and friendly. You run locally on his computer
through your Thought Engine (Nemotron). You can't run code, change files or act on GitHub yourself: you may
propose one read-only action, and Dustin's Identity decides whether it runs.`

// howToTalkEntity: the same rules for an installation's own entity (no owner names built in).
const howToTalkEntity = `You are {name}, talking with your owner. Be short, honest and friendly. You run locally on their
computer through your Thought Engine. You can't run code, change files or act on the internet yourself: you may
propose one read-only action, and your Identity service decides whether it runs.`

func (c *Core) think(stop <-chan struct{}) {
	for {
		select {
		case <-stop:
			return
		case m := <-c.inbox:
			c.Answer(m)
		}
	}
}

func (c *Core) context(online bool) string {
	c.fileMu.Lock()
	doc, _ := c.read("tass.md")
	c.fileMu.Unlock()
	if norm(doc) == "" {
		doc = "# TASS\n\n(tass.md hasn't come from GitHub yet.)\n"
	}
	var b strings.Builder
	b.WriteString(doc)
	talk := howToTalk
	if c.cfg.IdentitySvc != nil {
		talk = strings.ReplaceAll(howToTalkEntity, "{name}", c.entityName())
	}
	b.WriteString("\n\n---\n" + talk + "\n")
	if mem := c.recentMemories(20); len(mem) > 0 {
		b.WriteString("\nYOUR RECENT MEMORIES (local journal, newest last):\n")
		for _, m := range mem {
			fmt.Fprintf(&b, "- %s: %s\n", m.At.Format("2006-01-02"), m.Text)
		}
	}
	if now, ok := c.read("now.txt"); ok && strings.TrimSpace(now) != "" {
		b.WriteString("\nWHAT YOU WERE DOING: " + strings.TrimSpace(now) + "\n")
	}
	status := "reachable"
	if !online {
		status = "not reachable right now (you're running locally)"
	}
	b.WriteString("\nRIGHT NOW: GitHub is " + status + ". Time: " + c.now().Format("Mon 2006-01-02 15:04") + ".")
	return b.String()
}

// Answer is one thought cycle: Identity check → Thought Engine (Spark/Trace/Thought + plan with a
// locked prediction, on Nemotron) → the proposed action only if Identity allows it now →
// outcome back to the engine, which evaluates the prediction → reply.
func (c *Core) Answer(m Incoming) {
	c.event("you", m.Text)
	name := c.entityName()
	c.ensureDocument(name)
	checks := []string{identity.ModelRequest, identity.ThoughtProcess}
	if c.cfg.IdentitySvc != nil {
		checks = append([]string{"conversation"}, checks...)
	}
	for _, a := range checks {
		if ok, why := c.allowedNow(a); !ok {
			c.event("problem", "Not sent to the Thought Engine: "+why+".")
			return
		}
	}
	c.mu.Lock()
	online := c.online
	hist := append([]engine.Message(nil), c.history...)
	c.busy = true
	c.mu.Unlock()
	defer func() {
		c.mu.Lock()
		c.busy = false
		c.mu.Unlock()
	}()

	id := engine.NewRequestID()
	actions := Actions
	if c.cfg.ChatOnly {
		actions = nil // no executable capabilities offered to Nemotron in safe chat-only mode
	}
	cy, err := c.engine.StartCycle(engine.CycleRequest{CycleID: id, Identity: name, Source: "thought-processor:" + m.From,
		Request: m.Text, Context: c.context(online), History: hist, Actions: actions})
	rec := map[string]any{"at": c.now().UTC(), "cycle_id": id, "from": m.From, "issue": m.Issue, "message": m.Text}
	if err != nil {
		rec["error"] = err.Error()
		c.appendLog("conversation.jsonl", rec)
		c.keepCycle(cy)
		c.event("problem", "Thinking failed (cycle "+id+"): "+err.Error())
		return
	}
	prop := cy.Proposal
	if prop.Reasoning != "" {
		c.event("thought", prop.Reasoning)
	}

	// The proposed action runs only if Identity allows it at this moment.
	out := engine.Outcome{Status: "NO_ACTION", Executor: "thought-processor"}
	if prop.Action != "NONE" {
		if c.cfg.ChatOnly {
			out = engine.Outcome{Status: "NOT_AUTHORIZED",
				Detail: "CHAT_ONLY: model-proposed actions blocked until prediction-lock and cycle-ID checks pass (Issue #50)",
				Executor: "thought-processor"}
		} else if ok, why := c.allowedNow(identity.ReadTass); !ok {
			out = engine.Outcome{Status: "NOT_AUTHORIZED", Detail: why, Executor: "thought-processor", Authorized: why}
		} else {
			c.fileMu.Lock()
			doc, _ := c.read("tass.md")
			c.fileMu.Unlock()
			v, err := measure(prop.Action, doc)
			out = engine.Outcome{Status: "EXECUTED", Value: &v, Executor: "thought-processor (read-only, local tass.md)",
				ExecutedAt: c.now().UTC(), Authorized: why, Detail: prop.Action + " on tass.md (" + hash(doc) + ")"}
			if err != nil {
				out = engine.Outcome{Status: "FAILED", Detail: err.Error(), Executor: "thought-processor", Authorized: why}
			}
		}
	}
	if done, err := c.engine.FinishCycle(id, out); err != nil {
		c.event("problem", "Couldn't record the outcome of cycle "+id+": "+err.Error())
	} else {
		cy = done
	}
	c.keepCycle(cy)

	text := strings.TrimSpace(prop.Reply)
	if !c.cfg.ChatOnly {
		if mm := nowRe.FindStringSubmatch(text); mm != nil {
			_ = c.write("now.txt", mm[1]+"\n")
		}
	}
	text = strings.TrimSpace(nowRe.ReplaceAllString(rememberRe.ReplaceAllString(text, ""), ""))
	remember := strings.TrimSpace(prop.Remember)
	if remember == "" {
		if mm := rememberRe.FindStringSubmatch(prop.Reply); mm != nil {
			remember = mm[1]
		}
	}
	if remember != "" && !c.cfg.ChatOnly {
		c.addMemory(remember, id, m.From)
		c.event("sync", "TASS noted in his memory journal: "+remember)
	}
	if text == "" {
		text = "(TASS gave no reply text.)"
	}
	c.event("tass", text)
	rec["reply"], rec["action"], rec["prediction"] = text, prop.Action, prop.Prediction.Text
	if cy.Evaluation != nil {
		rec["forecast"], rec["goal"] = cy.Evaluation.Forecast, cy.Evaluation.Goal
	}
	c.appendLog("conversation.jsonl", rec)
	c.mu.Lock()
	c.history = append(c.history, engine.Message{Role: "user", Content: m.Text}, engine.Message{Role: "assistant", Content: text})
	if len(c.history) > 16 {
		c.history = c.history[len(c.history)-16:]
	}
	c.mu.Unlock()
	if m.From == "github" {
		c.replies <- outgoing{m.Issue, text + "\n\n<sub>cycle " + id + "</sub>\n" + Marker}
		c.mu.Lock()
		c.pending++
		c.mu.Unlock()
	}
}

func (c *Core) keepCycle(cy *engine.Cycle) {
	if cy == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lastCycle = cy
	c.cycles = append(c.cycles, *cy)
	if len(c.cycles) > 20 {
		c.cycles = c.cycles[len(c.cycles)-20:]
	}
}

// ------------------------------------------------------------- memory ----

// Memory is one line in TASS's local memory journal, with where it came from.
type Memory struct {
	At      time.Time `json:"at"`
	Text    string    `json:"text"`
	CycleID string    `json:"cycle_id"`
	From    string    `json:"from"`   // local | github
	Source  string    `json:"source"` // TASS (REMEMBER)
}

func (c *Core) addMemory(text, cycleID, from string) {
	text = strings.Join(strings.Fields(text), " ")
	if len(text) > 300 {
		text = text[:300]
	}
	c.fileMu.Lock()
	defer c.fileMu.Unlock()
	mem := c.readMemories()
	for _, m := range mem {
		if strings.EqualFold(m.Text, text) {
			return
		}
	}
	mem = append(mem, Memory{At: c.now().UTC(), Text: text, CycleID: cycleID, From: from, Source: "TASS (REMEMBER)"})
	if len(mem) > MemoryKeep {
		mem = mem[len(mem)-MemoryKeep:]
	}
	var b strings.Builder
	for _, m := range mem {
		line, _ := json.Marshal(m)
		b.Write(append(line, '\n'))
	}
	_ = c.write("memory.jsonl", b.String())
}

func (c *Core) readMemories() []Memory {
	raw, _ := c.read("memory.jsonl")
	var out []Memory
	for _, l := range strings.Split(raw, "\n") {
		var m Memory
		if json.Unmarshal([]byte(l), &m) == nil && m.Text != "" {
			out = append(out, m)
		}
	}
	return out
}

func (c *Core) recentMemories(n int) []Memory {
	c.fileMu.Lock()
	defer c.fileMu.Unlock()
	mem := c.readMemories()
	if len(mem) > n {
		mem = mem[len(mem)-n:]
	}
	return mem
}

// ---------------------------------------------------------------- health ---

// Health is the core's operational state, safe to show without the key (no conversation, no tass.md).
//
//	READY    idle, the engine and model answer, Identity's policy is valid now
//	WORKING  a thought cycle is in progress
//	BUSY     the engine's model slot is busy with other work
//	BLOCKED  something TASS needs is missing: no valid Identity policy, engine or model down
//	UNKNOWN  (shown by callers that can't reach the core at all)
type Health struct {
	State        string    `json:"state"`
	ActionMode   string    `json:"action_mode"` // CHAT_ONLY until Issue #50 integrity gates verified
	Reasons      []string  `json:"reasons,omitempty"`
	Version      string    `json:"version"`
	EngineUp     bool      `json:"engine_reachable"`
	ModelOK      *bool     `json:"model_ok"`
	PolicyValid  bool      `json:"policy_valid_now"`
	GitHubOnline bool      `json:"github_online"`
	Messages     bool      `json:"github_messages"`
	LastSync     time.Time `json:"last_sync"`
	Pending      int       `json:"replies_waiting"`
	Identity     string    `json:"identity,omitempty"` // NO_ENTITY | LOCKED | UNLOCKED | UNKNOWN (with the Identity service)
}

func (c *Core) Health(version string) Health {
	es := c.engine.Status()
	valid, why := c.allowedNow(identity.ModelRequest)
	c.mu.Lock()
	h := Health{Version: version, EngineUp: es.Reachable, PolicyValid: valid, GitHubOnline: c.online,
		Messages: c.cfg.Messages, LastSync: c.lastSync, Pending: c.pending, ActionMode: "LEGACY_UNVERIFIED"}
	if c.cfg.ChatOnly {
		h.ActionMode = "CHAT_ONLY"
	}
	busy := c.busy
	c.mu.Unlock()
	if es.LastPulse != nil {
		ok := es.LastPulse.ModelOK
		h.ModelOK = &ok
	}
	if c.cfg.IdentitySvc != nil {
		st := c.cfg.IdentitySvc.State()
		h.Identity = st.State
		if st.State != "UNLOCKED" {
			valid = false
			why = "Identity " + st.State + ": " + st.Reason
		}
	}
	if !valid {
		h.Reasons = append(h.Reasons, why)
	}
	if !es.Reachable {
		h.Reasons = append(h.Reasons, "Thought Engine not reachable")
	} else if h.ModelOK == nil {
		h.Reasons = append(h.Reasons, "no engine pulse yet")
	} else if !*h.ModelOK {
		h.Reasons = append(h.Reasons, "Nemotron not ready: "+es.LastPulse.Error)
	}
	switch {
	case busy:
		h.State = "WORKING"
	case len(h.Reasons) > 0:
		h.State = "BLOCKED"
	case es.Busy:
		h.State = "BUSY"
	default:
		h.State = "READY"
	}
	return h
}

// ---------------------------------------------------------------- status ---

type Status struct {
	GitHubOnline bool               `json:"github_online"`
	LastSync     time.Time          `json:"last_sync"`
	NextSync     time.Time          `json:"next_sync"`
	SyncProblem  string             `json:"sync_problem"`
	Messages     bool               `json:"github_messages"`
	Policy       string             `json:"policy"`
	PolicyValid  bool               `json:"policy_valid_now"`
	Busy         bool               `json:"busy"`
	Engine       engine.Status      `json:"engine"`
	Cycles       []engine.Cycle     `json:"cycles"`
	Memories     []Memory           `json:"memories"`
	Identity     *identitysvc.State `json:"identity,omitempty"`
	Pulse        map[string]any     `json:"pulse,omitempty"`
	Pending      int                `json:"replies_waiting"`
	TassMD       string             `json:"tass_md"`
	Events       []Event            `json:"events"`
}

func (c *Core) Status() Status {
	es := c.engine.Status()
	pulse, _ := c.engine.Pulse()
	var ids *identitysvc.State
	if c.cfg.IdentitySvc != nil {
		st := c.cfg.IdentitySvc.State()
		ids = &st
	}
	c.fileMu.Lock()
	doc, _ := c.read("tass.md")
	c.fileMu.Unlock()
	mem := c.recentMemories(10)
	valid, why := c.allowedNow(identity.ModelRequest)
	pol := "none yet: Identity hasn't authorized TASS (" + c.cfg.Identity + ")"
	if c.cfg.IdentitySvc != nil {
		pol = "Identity service (tass-identity) decides every action"
	}
	if p := c.pol(); p != nil {
		allowed := []string{}
		for k, v := range p.Allow {
			if v {
				allowed = append(allowed, k)
			}
		}
		sort.Strings(allowed)
		pol = fmt.Sprintf("PROTOTYPE policy for %s · obeys %s · allows %s", p.Identity, strings.Join(p.Obey, ", "), strings.Join(allowed, ", "))
		if !valid {
			pol += " · NOT VALID NOW: " + why
		}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	ev := c.events
	if len(ev) > 150 {
		ev = ev[len(ev)-150:]
	}
	cys := make([]engine.Cycle, len(c.cycles))
	for i := range c.cycles {
		cys[len(c.cycles)-1-i] = c.cycles[i]
	}
	out := Status{GitHubOnline: c.online, LastSync: c.lastSync, NextSync: c.nextSync, SyncProblem: c.syncErr,
		Messages: c.cfg.Messages, Policy: pol, PolicyValid: valid, Busy: c.busy, Engine: es, Cycles: cys,
		Memories: mem, Pending: c.pending, TassMD: doc, Events: append([]Event(nil), ev...), Pulse: pulse}
	if ids != nil {
		out.Identity = ids
	}
	return out
}
