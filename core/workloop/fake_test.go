package workloop

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Arameair/Thought-Processor/core/github"
)

// fakeGH is an in-memory GitHub Contents API: conditional GETs (ETag → 304), sha-checked
// PUTs (stale sha → 409), and per-token write scopes (→ 403), like a fine-grained token
// or branch rules would enforce. It is a fixture, not a claim about live GitHub.
type fakeGH struct {
	mu      sync.Mutex
	files   map[string]map[string]blob // branch -> path -> blob
	commits []commit
	scopes  map[string]scope // token -> scope
	gets    map[string]int   // actor -> GETs
	puts    map[string]int   // actor -> PUT attempts
	refs    map[string]int   // branch -> requests that named it
	before  func(actor, p string)
}

type blob struct{ text, sha string }
type commit struct{ actor, branch, path, msg, text string }
type scope struct {
	actor  string
	writes []string
}

func newFakeGH() *fakeGH {
	return &fakeGH{files: map[string]map[string]blob{}, scopes: map[string]scope{},
		gets: map[string]int{}, puts: map[string]int{}, refs: map[string]int{}}
}

func blobSHA(text string) string {
	s := sha256.Sum256([]byte("blob\x00" + text))
	return hex.EncodeToString(s[:20])
}

func (g *fakeGH) seed(branch, p, text string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.files[branch] == nil {
		g.files[branch] = map[string]blob{}
	}
	g.files[branch][p] = blob{text, blobSHA(text)}
}

func (g *fakeGH) file(branch, p string) (blob, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	b, ok := g.files[branch][p]
	return b, ok
}

func (g *fakeGH) handler() http.Handler {
	mux := http.NewServeMux()
	who := func(r *http.Request) scope {
		return g.scopes[strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")]
	}
	mux.HandleFunc("GET /repos/{o}/{r}/contents/{p...}", func(w http.ResponseWriter, r *http.Request) {
		s, branch, p := who(r), r.URL.Query().Get("ref"), r.PathValue("p")
		g.mu.Lock()
		g.gets[s.actor]++
		g.refs[branch]++
		b, ok := g.files[branch][p]
		g.mu.Unlock()
		if s.actor == "" {
			http.Error(w, `{"message":"Bad credentials"}`, 401)
			return
		}
		if !ok {
			http.Error(w, `{"message":"Not Found"}`, 404)
			return
		}
		etag := `"` + b.sha + `"`
		if r.Header.Get("If-None-Match") == etag {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Header().Set("ETag", etag)
		json.NewEncoder(w).Encode(map[string]string{"content": base64.StdEncoding.EncodeToString([]byte(b.text)),
			"encoding": "base64", "sha": b.sha})
	})
	mux.HandleFunc("PUT /repos/{o}/{r}/contents/{p...}", func(w http.ResponseWriter, r *http.Request) {
		s, p := who(r), r.PathValue("p")
		var body struct{ Message, Content, SHA, Branch string }
		json.NewDecoder(r.Body).Decode(&body)
		if g.before != nil {
			g.before(s.actor, p)
		}
		g.mu.Lock()
		defer g.mu.Unlock()
		g.puts[s.actor]++
		g.refs[body.Branch]++
		allowed := false
		for _, pat := range s.writes {
			if ok, _ := path.Match(pat, p); ok {
				allowed = true
			}
		}
		if s.actor == "" || !allowed || body.Branch != "main" {
			http.Error(w, `{"message":"Resource not accessible by integration"}`, 403)
			return
		}
		raw, _ := base64.StdEncoding.DecodeString(body.Content)
		cur, exists := g.files[body.Branch][p]
		if (exists && body.SHA != cur.sha) || (!exists && body.SHA != "") {
			http.Error(w, `{"message":"is at `+cur.sha+` but expected `+body.SHA+`"}`, 409)
			return
		}
		if exists && body.SHA == "" {
			http.Error(w, `{"message":"\"sha\" wasn't supplied."}`, 422)
			return
		}
		nb := blob{string(raw), blobSHA(string(raw))}
		g.files[body.Branch][p] = nb
		g.commits = append(g.commits, commit{s.actor, body.Branch, p, body.Message, nb.text})
		json.NewEncoder(w).Encode(map[string]any{"content": map[string]string{"sha": nb.sha}})
	})
	return mux
}

func (g *fakeGH) writes(actor string) (n int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	for _, c := range g.commits {
		if actor == "" || c.actor == actor {
			n++
		}
	}
	return n
}

// --- the world: one fake GitHub, five participants, a fake 20-second clock ----------------

type clock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *clock) Now() time.Time       { c.mu.Lock(); defer c.mu.Unlock(); return c.t }
func (c *clock) tick(d time.Duration) { c.mu.Lock(); c.t = c.t.Add(d); c.mu.Unlock() }

type world struct {
	t       *testing.T
	gh      *fakeGH
	srv     *httptest.Server
	clk     *clock
	policy  Policy
	ids     []string
	idsMu   sync.Mutex
	private []string
}

const pulse = 20 * time.Second

// The rules every iteration must read (a fixture copy of the project instructions).
const fixtureAgents = "# Fixture project instructions\n\n" +
	"- Read this file, the register and your workstream ledger before every transition.\n" +
	"- Audit only the exact candidate revision. An auditor never edits the candidate.\n" +
	"- PASS is not owner approval, merge, deployment or a grant.\n"

const fixtureTASS = "# TASS — team presence (fixture)\n\n" +
	"## Instructions for collaborators\n\nAssign work through the ledgers. Do not overwrite TASS's sections.\n\n" +
	"## Work status\n\n- none yet\n\n## Outbox\n\n- nothing published\n"

func newWorld(t *testing.T) *world {
	g := newFakeGH()
	w := &world{t: t, gh: g, srv: httptest.NewServer(g.handler()),
		clk: &clock{t: time.Date(2026, 11, 19, 12, 0, 0, 0, time.UTC)}}
	t.Cleanup(w.srv.Close)
	team := []string{"team/ws/*.md", "team/REGISTER.md"}
	w.policy = Policy{Writes: map[string][]string{
		"gpt6":   team,
		"claude": team,
		"codex":  team,
		"dustin": append(append([]string{}, team...), "team/TASS.md"),
		"tass":   {"team/TASS.md"}, // and only once Identity grants it (Observer.Granted)
	}}
	for actor, pats := range w.policy.Writes {
		g.scopes["token-"+actor] = scope{actor, pats}
	}
	g.seed("main", RulesPath, fixtureAgents)
	g.seed("main", TASSPath, fixtureTASS)
	// Core's identity/lesson sync target: must never be touched by the team loop.
	g.seed("live", "tass.md", "# TASS\n\n## Memory\n\n- private-memory-canary-7f3a\n")
	g.seed("live", "lessons/01-hello.md", "# Lesson 1\n")
	w.private = []string{"token-tass", "private-memory-canary-7f3a", "PRIVATE: Dustin's route starts at 3am"}
	return w
}

func (w *world) member(actor string) *Team {
	c := github.New("token-" + actor)
	c.Base = w.srv.URL
	tm := &Team{GH: c, Target: Target{"Arameair/team-fixture", "main"}, Policy: w.policy, Actor: actor,
		Now: w.clk.Now, IDs: func() []string { w.idsMu.Lock(); defer w.idsMu.Unlock(); return append([]string{}, w.ids...) }}
	if actor == "tass" {
		tm.Private = w.private
	}
	return tm
}

func (w *world) open(id string, coordinator *Team, task string, assignees ...string) State {
	w.idsMu.Lock()
	w.ids = append(w.ids, id)
	w.idsMu.Unlock()
	s, err := coordinator.Append(id, Event{Role: Coordinator, Kind: Assign, Task: task, Assignees: assignees})
	if err != nil {
		w.t.Fatal(err)
	}
	return s
}

// --- real git candidates -------------------------------------------------------------------------

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=fixture", "GIT_AUTHOR_EMAIL=fixture@example.invalid",
		"GIT_COMMITTER_NAME=fixture", "GIT_COMMITTER_EMAIL=fixture@example.invalid",
		"GIT_AUTHOR_DATE=2026-11-19T12:00:00Z", "GIT_COMMITTER_DATE=2026-11-19T12:00:00Z", "GIT_CONFIG_GLOBAL=/dev/null")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// codeRepo is a tiny Go module with one package, in its own git repository.
func codeRepo(t *testing.T, pkg string) string {
	dir := t.TempDir()
	git(t, dir, "init", "-q", "-b", "main")
	writeFile(t, dir, "go.mod", "module fixture\n\ngo 1.22\n")
	writeFile(t, dir, pkg+"/doc.go", "package "+pkg+"\n")
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-q", "-m", "start "+pkg)
	return dir
}

func writeFile(t *testing.T, dir, p, text string) {
	t.Helper()
	full := filepath.Join(dir, p)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

// commitFiles commits files and returns the exact revision.
func commitFiles(t *testing.T, dir, msg string, files map[string]string) string {
	for p, text := range files {
		writeFile(t, dir, p, text)
	}
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-q", "-m", msg)
	return git(t, dir, "rev-parse", "HEAD")
}

// goTest runs go test in dir and returns (ok, output).
func goTest(dir, pkg string) (bool, string) {
	cmd := exec.Command("go", "test", "-count=1", "./"+pkg+"/")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOWORK=off", "GOFLAGS=")
	out, err := cmd.CombinedOutput()
	return err == nil, string(out)
}

// auditExact is the independent audit: a fresh clone, detached at exactly the candidate,
// plus the auditor's own acceptance test that the implementer never saw.
func auditExact(t *testing.T, repo, commit, pkg, acceptance string) (verdict, evidence string, repair []string) {
	dir := filepath.Join(t.TempDir(), "audit")
	git(t, "", "clone", "-q", repo, dir)
	git(t, dir, "checkout", "-q", "--detach", commit)
	if head := git(t, dir, "rev-parse", "HEAD"); head != commit {
		t.Fatalf("auditor is at %s, not the candidate %s", head, commit)
	}
	writeFile(t, dir, pkg+"/audit_acceptance_test.go", acceptance)
	ok, out := goTest(dir, pkg)
	evidence = fmt.Sprintf("fresh clone at %s; go test -count=1 ./%s/ (+ auditor acceptance test): ", commit[:12], pkg)
	if ok {
		return "PASS", evidence + "ok", nil
	}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "--- FAIL:") || strings.Contains(line, "_test.go:") {
			repair = append(repair, "fix: "+line)
		}
	}
	repair = append(repair, fmt.Sprintf("retest: go test ./%s/ must pass the auditor's acceptance test; submit a NEW commit", pkg))
	return "FAIL", evidence + "FAIL", repair
}
