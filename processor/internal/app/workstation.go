package app

// Development Workstation (#31, first slice): a read-only view of the
// operator-allowlisted source repositories, a file viewer, and a fixed set
// of allowlisted test commands.
//
// Every request needs BOTH this run's browser session AND a fresh signed
// status from the paired Identity that says GitHub is VERIFIED (real GitHub
// unless a test explicitly allows the fixture). Nothing here writes to a
// repository, runs a shell, or accepts a command line from the browser.

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Arameair/Thought-Processor/processor/internal/store"
)

type wsRepo struct {
	Name string
	Dir  string // absolute, symlinks resolved
}

type wsTest struct {
	ID      string   `json:"id"`
	Repo    string   `json:"repo"`
	Label   string   `json:"label"`
	Sub     string   `json:"dir"`
	Argv    []string `json:"argv"`
	Timeout time.Duration
}

type wsRun struct {
	ID       string    `json:"id"`
	Test     string    `json:"test"`
	Label    string    `json:"label"`
	Repo     string    `json:"repo"`
	HeadSHA  string    `json:"head_sha"`
	Argv     []string  `json:"argv"`
	State    string    `json:"state"` // RUNNING | PASS | FAIL | TIMEOUT | ERROR
	ExitCode int       `json:"exit_code"`
	Started  time.Time `json:"started"`
	Ended    time.Time `json:"ended,omitempty"`
	Output   string    `json:"output"`
	Truncate bool      `json:"output_truncated"`
	By       string    `json:"requested_by"`
}

type Workstation struct {
	mu              sync.Mutex
	repos           []wsRepo
	runs            []*wsRun
	running         bool
	AllowTestGitHub bool // TEST ONLY: accept Identity's TEST FIXTURE GitHub mode
}

const (
	wsMaxFile   = 512 << 10
	wsMaxOutput = 96 << 10
	wsMaxRuns   = 10
)

var wsRepoName = regexp.MustCompile(`^[a-z][a-z0-9-]{0,31}$`)

// Never shown even inside an allowlisted repository.
var wsSecretName = regexp.MustCompile(`(?i)^(\.env.*|.*\.(pem|key|p12|pfx|kdbx)|id_(rsa|dsa|ecdsa|ed25519).*|identity-pairing\.json|\.netrc|\.git-credentials|operator-link\.txt|open-link\.txt)$`)

// catalog: the ONLY commands the workstation can run. Argv is fixed here;
// the browser chooses an id, never a command.
func wsCatalog() []wsTest {
	return []wsTest{
		{ID: "processor-go-app", Repo: "processor", Label: "Thought Processor: Go unit tests (app, idclient)", Sub: "processor",
			Argv: []string{"go", "test", "-count=1", "./internal/app/", "./internal/idclient/"}, Timeout: 3 * time.Minute},
		{ID: "identity-go-github", Repo: "identity", Label: "Identity: GitHub sign-in unit tests", Sub: ".",
			Argv: []string{"go", "test", "-count=1", "./internal/github/"}, Timeout: 3 * time.Minute},
		{ID: "tass-unit", Repo: "tass", Label: "TASS: feature unit tests (no network)", Sub: ".",
			Argv: []string{"python3", "-I", "-m", "unittest", "discover", "-s", "tests"}, Timeout: 3 * time.Minute},
	}
}

// AddRepo registers one operator-chosen repository: "name=/abs/path".
func (ws *Workstation) AddRepo(spec string) error {
	name, dir, ok := strings.Cut(spec, "=")
	if !ok || !wsRepoName.MatchString(name) {
		return fmt.Errorf("-workstation-repo wants name=path with a lowercase name, got %q", spec)
	}
	abs, err := filepath.Abs(dir)
	if err == nil {
		abs, err = filepath.EvalSymlinks(abs)
	}
	if err != nil {
		return fmt.Errorf("workstation repo %s: %v", name, err)
	}
	if st, err := os.Stat(filepath.Join(abs, ".git")); err != nil || st == nil {
		return fmt.Errorf("workstation repo %s: %s is not a Git checkout", name, abs)
	}
	ws.mu.Lock()
	defer ws.mu.Unlock()
	for _, r := range ws.repos {
		if r.Name == name {
			return fmt.Errorf("workstation repo %s given twice", name)
		}
	}
	ws.repos = append(ws.repos, wsRepo{Name: name, Dir: abs})
	return nil
}

func (ws *Workstation) repo(name string) (wsRepo, bool) {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	for _, r := range ws.repos {
		if r.Name == name {
			return r, true
		}
	}
	return wsRepo{}, false
}

func (ws *Workstation) tests() []wsTest {
	var out []wsTest
	for _, t := range wsCatalog() {
		if _, ok := ws.repo(t.Repo); ok {
			out = append(out, t)
		}
	}
	return out
}

// gate: fresh, signed Identity status with GitHub VERIFIED.
type wsGate struct {
	OK         bool   `json:"ok"`
	Reason     string `json:"reason,omitempty"` // IDENTITY_NOT_CONNECTED | GITHUB_NOT_VERIFIED | GITHUB_TEST_FIXTURE
	Detail     string `json:"detail,omitempty"`
	Identity   string `json:"identity,omitempty"`
	GitHub     string `json:"github_state"`
	GitHubMode string `json:"github_mode,omitempty"`
	Login      string `json:"github_login,omitempty"`
}

func (a *App) wsGate(ctx context.Context) wsGate {
	g := wsGate{GitHub: "UNKNOWN"}
	if _, err := a.Conn.Require(ctx); err != nil {
		g.Reason, g.Detail = "IDENTITY_NOT_CONNECTED", "Pair Thought Processor with Identity first (Thought Processor page)."
		return g
	}
	a.Conn.mu.Lock()
	st := a.Conn.status
	a.Conn.mu.Unlock()
	if st == nil {
		g.Reason = "IDENTITY_NOT_CONNECTED"
		return g
	}
	g.Identity, g.GitHub, g.GitHubMode = st.IdentitySystemID, st.GitHub.State, st.GitHub.Mode
	if st.GitHub.State != "VERIFIED" || st.GitHub.Account == nil || st.GitHub.Account.Login == "" {
		g.Reason, g.Detail = "GITHUB_NOT_VERIFIED", "Identity reports GitHub "+st.GitHub.State+". Sign in to GitHub from the Thought Processor page."
		return g
	}
	g.Login = st.GitHub.Account.Login
	if st.GitHub.Mode != "real" && !a.WS.AllowTestGitHub {
		g.Reason, g.Detail = "GITHUB_TEST_FIXTURE", "Identity is using a GitHub TEST FIXTURE; the workstation needs real GitHub."
		return g
	}
	g.OK = true
	return g
}

var credInURL = regexp.MustCompile(`://[^/\s@]+@`)

func gitOut(dir string, args ...string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(wsEnv(), "GIT_TERMINAL_PROMPT=0", "GIT_OPTIONAL_LOCKS=0")
	b, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func (ws *Workstation) repoInfo(r wsRepo) map[string]any {
	dirty := gitOut(r.Dir, "status", "--porcelain")
	n := 0
	if dirty != "" {
		n = len(strings.Split(dirty, "\n"))
	}
	last := strings.SplitN(gitOut(r.Dir, "log", "-1", "--format=%s%x00%cI"), "\x00", 2)
	info := map[string]any{"name": r.Name, "path": displayHome(r.Dir), "head_sha": gitOut(r.Dir, "rev-parse", "HEAD"),
		"branch": gitOut(r.Dir, "branch", "--show-current"), "origin": credInURL.ReplaceAllString(gitOut(r.Dir, "remote", "get-url", "origin"), "://***@"),
		"changed_files": n}
	if len(last) == 2 {
		info["last_commit"], info["last_commit_at"] = last[0], last[1]
	}
	return info
}

func displayHome(p string) string {
	if h, err := os.UserHomeDir(); err == nil && h != "" && h != "/" && strings.HasPrefix(p, h) {
		return "~" + strings.TrimPrefix(p, h)
	}
	return p
}

var errWSPath = errors.New("that path is outside the repository or not allowed")

// resolve maps a repo-relative path to a real path inside the repo.
func (ws *Workstation) resolve(r wsRepo, rel string) (string, error) {
	rel = filepath.ToSlash(strings.TrimSpace(rel))
	if strings.ContainsRune(rel, 0) || filepath.IsAbs(rel) || strings.HasPrefix(rel, "/") {
		return "", errWSPath
	}
	for _, part := range strings.Split(rel, "/") {
		if part == ".." || part == ".git" || wsSecretName.MatchString(part) {
			return "", errWSPath
		}
	}
	p := filepath.Join(r.Dir, filepath.FromSlash(rel))
	real, err := filepath.EvalSymlinks(p)
	if err != nil {
		return "", errWSPath
	}
	if real != r.Dir && !strings.HasPrefix(real, r.Dir+string(os.PathSeparator)) {
		return "", errWSPath
	}
	return real, nil
}

func wsEnv() []string {
	var env []string
	for _, k := range []string{"PATH", "HOME", "LANG", "LC_ALL", "TMPDIR", "USER", "GOPATH", "GOCACHE", "GOMODCACHE", "SYSTEMROOT", "LOCALAPPDATA", "APPDATA", "USERPROFILE"} {
		if v, ok := os.LookupEnv(k); ok {
			env = append(env, k+"="+v)
		}
	}
	return env
}

func (a *App) wsRun(t wsTest, r wsRepo, by string) (*wsRun, error) {
	ws := a.WS
	ws.mu.Lock()
	if ws.running {
		ws.mu.Unlock()
		return nil, errors.New("a test is already running; wait for it to finish")
	}
	ws.running = true
	run := &wsRun{ID: "run_" + store.RandHex(6), Test: t.ID, Label: t.Label, Repo: r.Name, Argv: t.Argv, State: "RUNNING",
		Started: time.Now().UTC(), By: by, ExitCode: -1}
	ws.runs = append([]*wsRun{run}, ws.runs...)
	if len(ws.runs) > wsMaxRuns {
		ws.runs = ws.runs[:wsMaxRuns]
	}
	ws.mu.Unlock()
	run.HeadSHA = gitOut(r.Dir, "rev-parse", "HEAD")
	dir := filepath.Join(r.Dir, filepath.FromSlash(t.Sub))
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), t.Timeout)
		defer cancel()
		cmd := exec.Command(t.Argv[0], t.Argv[1:]...)
		cmd.Dir = dir
		// Offline and pinned: no module downloads, no toolchain switch.
		cmd.Env = append(wsEnv(), "GOFLAGS=-mod=readonly", "GOPROXY=off", "GOTOOLCHAIN=local", "PYTHONDONTWRITEBYTECODE=1")
		var buf bytes.Buffer
		cmd.Stdout, cmd.Stderr = &buf, &buf
		cmd.Stdin = nil
		wsProcGroup(cmd)
		state, code := "ERROR", -1
		if err := cmd.Start(); err == nil {
			done := make(chan error, 1)
			go func() { done <- cmd.Wait() }()
			select {
			case err = <-done:
				code = cmd.ProcessState.ExitCode()
				if err == nil {
					state = "PASS"
				} else {
					state = "FAIL"
				}
			case <-ctx.Done():
				wsKillGroup(cmd)
				<-done
				state = "TIMEOUT"
			}
		} else {
			buf.WriteString("could not start: " + err.Error())
		}
		out := buf.String()
		trunc := false
		if len(out) > wsMaxOutput {
			out, trunc = "…(earlier output cut)…\n"+out[len(out)-wsMaxOutput:], true
		}
		ws.mu.Lock()
		run.State, run.ExitCode, run.Output, run.Truncate, run.Ended = state, code, out, trunc, time.Now().UTC()
		ws.running = false
		ws.mu.Unlock()
		a.Log.Info("workstation", "workstation.test_"+strings.ToLower(state), fmt.Sprintf("%s on %s @ %s by %s (exit %d)", t.ID, r.Name, short(run.HeadSHA), by, code))
	}()
	return run, nil
}

func (ws *Workstation) snapshotRuns() []wsRun {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	out := make([]wsRun, len(ws.runs))
	for i, r := range ws.runs {
		out[i] = *r
	}
	return out
}

func (a *App) workstationRoutes(m *http.ServeMux) {
	// The page itself is public (like Processor's own page); it signs in
	// with a one-time ticket and every API call below needs the session.
	m.HandleFunc("GET /workstation", a.wrap(public, func(w http.ResponseWriter, r *http.Request) {
		b, err := webFS.ReadFile("web/workstation.html")
		if err != nil {
			http.Error(w, "missing", 500)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(b)
	}))
	m.HandleFunc("GET /workstation.js", a.wrap(public, func(w http.ResponseWriter, r *http.Request) {
		b, err := webFS.ReadFile("web/workstation.js")
		if err != nil {
			http.Error(w, "missing", 500)
			return
		}
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		w.Write(b)
	}))
	gated := func(lv level, h func(w http.ResponseWriter, r *http.Request, g wsGate)) http.HandlerFunc {
		return a.wrap(lv, func(w http.ResponseWriter, r *http.Request) {
			g := a.wsGate(r.Context())
			if !g.OK {
				writeJSON(w, 403, map[string]any{"error": "Development Workstation is locked: " + g.Reason, "gate": g})
				return
			}
			h(w, r, g)
		})
	}
	a.tassRoutes(m, gated)
	m.HandleFunc("GET /api/ws/state", a.wrap(private, func(w http.ResponseWriter, r *http.Request) {
		g := a.wsGate(r.Context())
		out := map[string]any{"gate": g, "processor_version": a.Build.Version}
		if g.OK {
			var repos []map[string]any
			a.WS.mu.Lock()
			list := append([]wsRepo(nil), a.WS.repos...)
			a.WS.mu.Unlock()
			for _, rp := range list {
				repos = append(repos, a.WS.repoInfo(rp))
			}
			out["repos"], out["tests"], out["runs"] = repos, a.WS.tests(), a.WS.snapshotRuns()
		}
		writeJSON(w, 200, out)
	}))
	m.HandleFunc("GET /api/ws/tree", gated(private, func(w http.ResponseWriter, r *http.Request, g wsGate) {
		rp, ok := a.WS.repo(r.URL.Query().Get("repo"))
		if !ok {
			a.fail(w, 404, "unknown repository")
			return
		}
		rel := r.URL.Query().Get("path")
		p, err := a.WS.resolve(rp, rel)
		if err != nil {
			a.fail(w, 403, err.Error())
			return
		}
		ents, err := os.ReadDir(p)
		if err != nil {
			a.fail(w, 400, "not a folder")
			return
		}
		type ent struct {
			Name string `json:"name"`
			Dir  bool   `json:"dir"`
			Size int64  `json:"size"`
		}
		var list []ent
		for _, e := range ents {
			if e.Name() == ".git" || wsSecretName.MatchString(e.Name()) {
				continue
			}
			if e.Type()&os.ModeSymlink != 0 {
				if _, err := a.WS.resolve(rp, filepath.ToSlash(filepath.Join(rel, e.Name()))); err != nil {
					continue // links that lead outside the repo are hidden
				}
			}
			info, err := e.Info()
			if err != nil {
				continue
			}
			isDir := e.IsDir()
			if e.Type()&os.ModeSymlink != 0 {
				if st, err := os.Stat(filepath.Join(p, e.Name())); err == nil {
					isDir = st.IsDir()
				}
			}
			list = append(list, ent{Name: e.Name(), Dir: isDir, Size: info.Size()})
		}
		sort.Slice(list, func(i, j int) bool {
			if list[i].Dir != list[j].Dir {
				return list[i].Dir
			}
			return list[i].Name < list[j].Name
		})
		writeJSON(w, 200, map[string]any{"repo": rp.Name, "path": filepath.ToSlash(filepath.Clean("/" + rel))[1:], "entries": list})
	}))
	m.HandleFunc("GET /api/ws/file", gated(private, func(w http.ResponseWriter, r *http.Request, g wsGate) {
		rp, ok := a.WS.repo(r.URL.Query().Get("repo"))
		if !ok {
			a.fail(w, 404, "unknown repository")
			return
		}
		p, err := a.WS.resolve(rp, r.URL.Query().Get("path"))
		if err != nil {
			a.fail(w, 403, err.Error())
			return
		}
		st, err := os.Stat(p)
		if err != nil || !st.Mode().IsRegular() {
			a.fail(w, 400, "not a regular file")
			return
		}
		if st.Size() > wsMaxFile {
			writeJSON(w, 200, map[string]any{"size": st.Size(), "shown": false, "note": "file is larger than 512 KB; not shown"})
			return
		}
		b, err := os.ReadFile(p)
		if err != nil {
			a.fail(w, 400, "could not read file")
			return
		}
		if bytes.IndexByte(b[:min(len(b), 8192)], 0) >= 0 {
			writeJSON(w, 200, map[string]any{"size": st.Size(), "shown": false, "note": "binary file; not shown"})
			return
		}
		writeJSON(w, 200, map[string]any{"size": st.Size(), "shown": true, "text": string(b)})
	}))
	m.HandleFunc("POST /api/ws/test", gated(mutate, func(w http.ResponseWriter, r *http.Request, g wsGate) {
		var b struct{ ID string }
		decode(r, &b)
		for _, t := range a.WS.tests() {
			if t.ID == b.ID {
				rp, _ := a.WS.repo(t.Repo)
				run, err := a.wsRun(t, rp, g.Login)
				if err != nil {
					a.fail(w, 409, err.Error())
					return
				}
				writeJSON(w, 202, map[string]any{"run": run.ID})
				return
			}
		}
		a.fail(w, 404, "not an allowlisted test")
	}))
}
