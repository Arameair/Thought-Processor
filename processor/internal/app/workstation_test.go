package app

import (
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func wsTestRepo(t *testing.T) (*Workstation, wsRepo, string) {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "repo")
	os.MkdirAll(filepath.Join(dir, "src"), 0o755)
	if out, err := exec.Command("git", "init", "-q", dir).CombinedOutput(); err != nil {
		t.Skipf("git unavailable: %v %s", err, out)
	}
	os.WriteFile(filepath.Join(dir, "src", "main.go"), []byte("package main\n"), 0o644)
	os.WriteFile(filepath.Join(dir, ".env"), []byte("SECRET=1\n"), 0o600)
	os.WriteFile(filepath.Join(dir, "deploy.pem"), []byte("key\n"), 0o600)
	outside := filepath.Join(root, "outside.txt")
	os.WriteFile(outside, []byte("private\n"), 0o600)
	os.Symlink(outside, filepath.Join(dir, "escape"))
	os.Symlink(filepath.Join(dir, "src", "main.go"), filepath.Join(dir, "inside"))
	ws := &Workstation{}
	if err := ws.AddRepo("demo=" + dir); err != nil {
		t.Fatal(err)
	}
	r, _ := ws.repo("demo")
	return ws, r, outside
}

func TestWorkstationPathGuards(t *testing.T) {
	ws, r, _ := wsTestRepo(t)
	for _, bad := range []string{"../outside.txt", "/etc/passwd", "src/../../outside.txt", ".git/config", ".env", "deploy.pem", "escape", "a\x00b"} {
		if _, err := ws.resolve(r, bad); err == nil {
			t.Errorf("path %q was allowed", bad)
		}
	}
	for _, good := range []string{"", "src", "src/main.go", "inside"} {
		if _, err := ws.resolve(r, good); err != nil {
			t.Errorf("path %q refused: %v", good, err)
		}
	}
}

func TestWorkstationRepoSpec(t *testing.T) {
	ws := &Workstation{}
	for _, bad := range []string{"noequals", "Bad=/tmp", "x=" + t.TempDir()} {
		if err := ws.AddRepo(bad); err == nil {
			t.Errorf("repo spec %q accepted", bad)
		}
	}
}

func TestWorkstationOnlyRunsCatalogForKnownRepos(t *testing.T) {
	ws, _, _ := wsTestRepo(t)
	if len(ws.tests()) != 0 {
		t.Fatal("tests offered for a repo with no catalog entry")
	}
	for _, c := range wsCatalog() {
		if len(c.Argv) == 0 || strings.ContainsAny(strings.Join(c.Argv, " "), ";|&$`") {
			t.Fatalf("catalog entry %s is not a plain argv", c.ID)
		}
	}
}

// Without Identity, every workstation data route is locked, and nothing is
// reachable without the browser session.
func TestWorkstationLockedWithoutIdentity(t *testing.T) {
	x := start(t)
	_, r, _ := wsTestRepo(t)
	x.a.WS.AddRepo("processor=" + r.Dir)
	for _, p := range []string{"/api/ws/tree?repo=processor&path=", "/api/ws/file?repo=processor&path=src/main.go"} {
		resp, _ := http.Get(x.base + p)
		if resp.StatusCode != 401 {
			t.Fatalf("%s readable without session: %d", p, resp.StatusCode)
		}
		req, _ := http.NewRequest("GET", x.base+p, nil)
		resp, err := x.c.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != 403 {
			t.Fatalf("%s open without Identity: %d", p, resp.StatusCode)
		}
	}
	if st, b := x.post("/api/ws/test", `{"id":"processor-go-app"}`, nil); st != 403 || !strings.Contains(b, "IDENTITY_NOT_CONNECTED") {
		t.Fatalf("test ran without Identity: %d %s", st, b)
	}
	req, _ := http.NewRequest("GET", x.base+"/api/ws/state", nil)
	resp, _ := x.c.Do(req)
	if resp.StatusCode != 200 {
		t.Fatalf("state %d", resp.StatusCode)
	}
}
