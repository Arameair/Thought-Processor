// Package tass is the Thought-Processor side of TASS's core.
//
//   - Thinking is local: TASS's tass.md + the message go to the Thought Engine, which runs
//     Nemotron. No internet needed.
//   - GitHub is a separate, asynchronous sync loop (every 20 s, backing off when GitHub is
//     unreachable): Identity's policy, the GitHub copy of tass.md, messages and replies.
//   - Identity's policy decides before anything happens. No policy, no action.
//
// Authority for tass.md: the local copy is what TASS runs on; the GitHub copy is the versioned
// record. Each sync merges them section by section; when the same section changed on both sides,
// TASS keeps running on the GitHub version of it and a pull request offers his version: Dustin
// merges or closes it.
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
	"strings"
	"sync"
	"time"

	"github.com/Arameair/Thought-Processor/core/engine"
	"github.com/Arameair/Thought-Processor/core/github"
	"github.com/Arameair/Thought-Processor/core/identity"
	"github.com/Arameair/Thought-Processor/core/tassmd"
)

// Marker on everything TASS posts, so he never answers himself.
const Marker = "<!-- tass-core -->"

// PolicyFile is the local copy of Identity's policy (the Thought Engine reads it too).
const PolicyFile = "identity-policy.json"

type Config struct {
	DataDir    string        // local home: tass.md, tass.base.md, identity-policy.json, state.json, logs
	Identity   string        // owner/repo@branch:path of Identity's policy
	Label      string        // issues with this label are messages to TASS
	SyncEvery  time.Duration // GitHub sync interval when all is well (20 s)
	MaxBackoff time.Duration // longest wait between attempts while GitHub is unreachable
}

type state struct {
	PolicyETag   string         `json:"policy_etag"`
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
	Text  string
	From  string // "local" or "github"
	Issue int
}

type Core struct {
	cfg    Config
	gh     *github.Client
	engine *engine.Client
	now    func() time.Time

	fileMu sync.Mutex // tass.md and tass.base.md

	// owned by the sync loop
	st state

	mu         sync.Mutex // fields below
	policy     *identity.Policy
	online     bool
	lastSync   time.Time
	nextSync   time.Time
	syncErr    string
	failures   int
	events     []Event
	busy       bool
	lastResult *engine.Result
	history    []engine.Message
	pending    int

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
	if b, err := os.ReadFile(c.file(PolicyFile)); err == nil {
		if p, err := identity.Parse(b); err == nil {
			c.policy = p // last good policy: TASS knows his permissions offline
		}
	}
	c.pending = len(c.st.Outbox)
	return c, nil
}

func (c *Core) pol() *identity.Policy {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.policy
}

func (c *Core) file(name string) string { return filepath.Join(c.cfg.DataDir, name) }

func (c *Core) read(name string) (string, bool) {
	b, err := os.ReadFile(c.file(name))
	return string(b), err == nil
}

func (c *Core) write(name, text string) error {
	tmp := c.file("." + name + ".tmp")
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
	wg.Add(2)
	go func() { defer wg.Done(); c.think(stop) }()
	go func() { defer wg.Done(); c.syncLoop(stop) }()
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

// syncPolicy fetches Identity's policy (conditional, so unchanged = no rate-limit cost).
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
		_ = os.Remove(c.file(PolicyFile))
		return fmt.Errorf("no Identity policy at %s (Identity hasn't authorized TASS yet)", c.cfg.Identity)
	}
	if err != nil {
		return err
	}
	if got.NotModified {
		return nil
	}
	p, err := identity.Parse([]byte(f.Text))
	if err != nil {
		return err // keep the last good policy
	}
	old, _ := c.read(PolicyFile)
	if err := c.write(PolicyFile, f.Text); err != nil {
		return err
	}
	if norm(old) != norm(f.Text) {
		c.event("sync", "Identity policy loaded for "+p.Identity+" from "+c.cfg.Identity+".")
	}
	c.mu.Lock()
	c.policy = p
	c.mu.Unlock()
	c.st.PolicyETag = got.ETag
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
	if !p.Allowed(identity.ReadMessages) {
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

func (c *Core) queue(m Incoming) {
	select {
	case c.inbox <- m:
		if m.From == "github" {
			c.event("github", fmt.Sprintf("Message from GitHub, issue #%d", m.Issue))
		}
	default:
		c.event("problem", "Too many messages waiting; one was dropped.")
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

// Say is a message from the local page. It never needs the internet.
func (c *Core) Say(text string) error {
	text = strings.TrimSpace(text)
	if text == "" {
		return errors.New("empty message")
	}
	c.queue(Incoming{Text: text, From: "local"})
	return nil
}

// ------------------------------------------------------------- thinking ----

var (
	rememberRe = regexp.MustCompile(`(?im)^[ \t*_>-]*REMEMBER[*_ ]*:[*_ ]*(.+?)[*_ ]*$`)
	nowRe      = regexp.MustCompile(`(?im)^[ \t*_>-]*NOW[*_ ]*:[*_ ]*(.+?)[*_ ]*$`)
	blankRe    = regexp.MustCompile(`\n{3,}`)
)

const howToAct = `HOW TO ACT
Reply to Dustin as TASS: short, honest, friendly. Think step by step before you answer.
You may add these lines anywhere in your reply; they are taken out of what Dustin sees:
  REMEMBER: <one short line>   saved under "## Memory" in your tass.md (what's worth keeping)
  NOW: <one short line>        replaces "## Now" in your tass.md (what you're doing or thinking about)
Your tass.md lives on this computer, so you keep thinking and remembering without internet. It syncs
with its GitHub copy whenever GitHub can be reached.`

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

// Answer: Identity check → Thought Engine (Nemotron) → tass.md updates → reply.
func (c *Core) Answer(m Incoming) {
	c.event("you", m.Text)
	p := c.pol()
	if !p.Allowed(identity.ModelRequest) {
		c.event("problem", "Identity hasn't authorized TASS to think (model.request) yet, so this wasn't sent to the model.")
		return
	}
	c.fileMu.Lock()
	doc, _ := c.read("tass.md")
	c.fileMu.Unlock()
	if norm(doc) == "" {
		doc = "# TASS\n\n(tass.md hasn't come from GitHub yet.)\n"
	}
	where := "on this computer (the local page)"
	if m.From == "github" {
		where = fmt.Sprintf("on GitHub, issue #%d", m.Issue)
	}
	c.mu.Lock()
	online := c.online
	hist := append([]engine.Message(nil), c.history...)
	c.busy = true
	c.mu.Unlock()
	status := "reachable"
	if !online {
		status = "not reachable right now (you're running locally)"
	}
	sys := doc + "\n\n---\n" + howToAct + "\n\nRIGHT NOW: GitHub is " + status + ". This message came " + where +
		". Time: " + c.now().Format("Mon 2006-01-02 15:04") + "."
	msgs := append([]engine.Message{{Role: "system", Content: sys}}, hist...)
	msgs = append(msgs, engine.Message{Role: "user", Content: m.Text})

	id := engine.NewRequestID()
	r, err := c.engine.Think(id, p.Identity, "reply", msgs)
	c.mu.Lock()
	c.busy = false
	c.lastResult = r
	c.mu.Unlock()
	rec := map[string]any{"at": c.now().UTC(), "request_id": id, "from": m.From, "issue": m.Issue, "message": m.Text}
	if r != nil {
		rec["elapsed_ms"], rec["authorized"] = r.ElapsedMS, r.Authorized
	}
	if err != nil {
		rec["error"] = err.Error()
		c.appendLog("conversation.jsonl", rec)
		c.event("problem", "Thinking failed ("+id+"): "+err.Error())
		return
	}
	if r.Thought != "" {
		c.event("thought", r.Thought)
	}
	text := r.Text
	c.fileMu.Lock()
	cur, _ := c.read("tass.md")
	changed := cur
	for _, mm := range rememberRe.FindAllStringSubmatch(text, 5) {
		changed = tassmd.AddMemory(changed, mm[1], c.now())
	}
	if mm := nowRe.FindStringSubmatch(text); mm != nil {
		changed = tassmd.SetNow(changed, mm[1], c.now())
	}
	if norm(cur) != "" && changed != cur {
		if err := c.write("tass.md", changed); err != nil {
			c.event("problem", "Couldn't save tass.md: "+err.Error())
		}
	}
	c.fileMu.Unlock()
	for _, mm := range rememberRe.FindAllStringSubmatch(text, 5) {
		c.event("sync", "TASS remembered: "+mm[1])
	}
	text = strings.TrimSpace(nowRe.ReplaceAllString(rememberRe.ReplaceAllString(text, ""), ""))
	text = blankRe.ReplaceAllString(text, "\n\n")
	if text == "" {
		text = "(TASS only updated his tass.md.)"
	}
	c.event("tass", text)
	rec["thought"], rec["reply"] = r.Thought, text
	c.appendLog("conversation.jsonl", rec)
	c.mu.Lock()
	c.history = append(c.history, engine.Message{Role: "user", Content: m.Text}, engine.Message{Role: "assistant", Content: text})
	if len(c.history) > 16 {
		c.history = c.history[len(c.history)-16:]
	}
	c.mu.Unlock()
	if m.From == "github" {
		c.replies <- outgoing{m.Issue, text + "\n\n" + Marker}
		c.mu.Lock()
		c.pending++
		c.mu.Unlock()
	}
}

// ---------------------------------------------------------------- status ---

type Status struct {
	GitHubOnline bool           `json:"github_online"`
	LastSync     time.Time      `json:"last_sync"`
	NextSync     time.Time      `json:"next_sync"`
	SyncProblem  string         `json:"sync_problem"`
	Policy       string         `json:"policy"`
	Busy         bool           `json:"busy"`
	Engine       engine.Status  `json:"engine"`
	LastRequest  *engine.Result `json:"last_request,omitempty"`
	Pending      int            `json:"replies_waiting"`
	TassMD       string         `json:"tass_md"`
	Events       []Event        `json:"events"`
}

func (c *Core) Status() Status {
	es := c.engine.Status()
	c.fileMu.Lock()
	doc, _ := c.read("tass.md")
	c.fileMu.Unlock()
	pol := "none yet: Identity hasn't authorized TASS (" + c.cfg.Identity + ")"
	if p := c.pol(); p != nil {
		allowed := []string{}
		for k, v := range p.Allow {
			if v {
				allowed = append(allowed, k)
			}
		}
		pol = fmt.Sprintf("%s · obeys %s · allows %s", p.Identity, strings.Join(p.Obey, ", "), strings.Join(allowed, ", "))
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	ev := c.events
	if len(ev) > 150 {
		ev = ev[len(ev)-150:]
	}
	var lr *engine.Result
	if c.lastResult != nil {
		x := *c.lastResult
		x.Thought, x.Text = "", ""
		lr = &x
	}
	return Status{GitHubOnline: c.online, LastSync: c.lastSync, NextSync: c.nextSync, SyncProblem: c.syncErr,
		Policy: pol, Busy: c.busy, Engine: es, LastRequest: lr, Pending: c.pending, TassMD: doc,
		Events: append([]Event(nil), ev...)}
}
