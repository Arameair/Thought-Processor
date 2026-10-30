package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVersionAtLeast(t *testing.T) {
	for _, c := range []struct {
		v, min string
		want   bool
	}{{"0.2.0", "0.2.0", true}, {"0.10.0", "0.2.0", true}, {"0.1.9", "0.2.0", false}, {"1.0", "0.2.0", true}, {"x", "0.2.0", false}} {
		if got := versionAtLeast(c.v, c.min); got != c.want {
			t.Errorf("%s >= %s: %v", c.v, c.min, got)
		}
	}
}

func TestFileContextForTASSUsesWorkstationGuards(t *testing.T) {
	ws, r, _ := wsTestRepo(t)
	a := &App{WS: ws}
	ctx, err := a.fileContext("demo", "src/main.go")
	if err != nil || ctx["text"] != "package main\n" || ctx["truncated"] != false || len(ctx["file_sha256"].(string)) != 64 || ctx["path"] != "src/main.go" {
		t.Fatalf("file context %v %v", ctx, err)
	}
	for _, bad := range []string{".env", "deploy.pem", "escape", "../outside.txt", ".git/config"} {
		if _, err := a.fileContext("demo", bad); err == nil {
			t.Errorf("%s was shown to TASS", bad)
		}
	}
	if _, err := a.fileContext("nope", "src/main.go"); err == nil {
		t.Error("unknown repository accepted")
	}
	os.WriteFile(filepath.Join(r.Dir, "bin.dat"), []byte{0x7f, 'E', 'L', 'F', 0, 1}, 0o644)
	if _, err := a.fileContext("demo", "bin.dat"); err == nil {
		t.Error("binary file shown to TASS")
	}
	big := strings.Repeat("é", maxFileContext+10) // multi-byte: cut on characters, not bytes
	os.WriteFile(filepath.Join(r.Dir, "big.txt"), []byte(big), 0o644)
	ctx, err = a.fileContext("demo", "big.txt")
	if err != nil || ctx["truncated"] != true || len([]rune(ctx["text"].(string))) != maxFileContext || ctx["bytes"] != len(big) {
		t.Fatalf("truncation wrong: %v", err)
	}
}

func TestWorkstationTASSLockedWithoutIdentity(t *testing.T) {
	x := start(t)
	for _, p := range []string{"/api/ws/tass/ask", "/api/ws/tass/allow", "/api/ws/tass/owner", "/api/ws/tass/install", "/api/ws/tass/model"} {
		if st, b := x.post(p, `{"question":"hi","what":"chat"}`, nil); st != 403 || !strings.Contains(b, "IDENTITY_NOT_CONNECTED") {
			t.Fatalf("%s without Identity: %d %s", p, st, b)
		}
		if st, _ := x.post(p, `{}`, map[string]string{"X-TP-CSRF": ""}); st != 403 {
			t.Fatalf("%s without anti-forgery token: %d", p, st)
		}
	}
}
