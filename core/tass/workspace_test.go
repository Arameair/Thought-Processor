package tass

import (
	"strings"
	"testing"
	"time"
)

func TestWorkspaceFileIsAllowlistedReadOnlyAndMatchesGitHub(t *testing.T) {
	_, _, _, _, c := skillCore(t, true)
	w := c.Workspace()
	if w.Repo != "Mira/home" || w.Branch != "live" || w.DocBlobSHA == "" || len(w.Files) != 3 {
		t.Fatalf("workspace info: %+v", w)
	}
	f, err := c.ReadWorkspaceFile(pulsePath)
	// The fixture GitHub reports placeholder SHAs, so a match can't be shown here; real GitHub
	// reports Git blob SHAs, which gitBlobSHA reproduces exactly (checked below).
	if err != nil || !f.ReadOnly || f.Matches || !strings.Contains(f.SourceNote, "differs") || !strings.Contains(f.Text, "# My work pulse") {
		t.Fatalf("lesson file: %v %+v", err, f)
	}
	if got := gitBlobSHA([]byte("hello")); got != "b6fc4c620b67d95f953a5c1c1230aaab5db5a1b0" { // git hash-object
		t.Fatalf("gitBlobSHA = %s", got)
	}
	for _, bad := range []string{"../secrets/github_token", "lessons/../../state.json", "/etc/passwd", "skills.json", "lessons/claude/nope.md"} {
		if _, err := c.ReadWorkspaceFile(bad); err == nil {
			t.Fatalf("read a non-workspace path %q", bad)
		}
	}
	if d, err := c.ReadWorkspaceFile("tass.md"); err != nil || !strings.HasPrefix(d.Text, "# Mira") {
		t.Fatalf("document: %v", err)
	}
}

func TestAskAboutFileSendsItReadOnlyWithNoActions(t *testing.T) {
	_, eng, _, _, c := skillCore(t, true)
	if _, err := c.SayAboutFile("what does step 2 say?", "../etc/passwd"); err == nil {
		t.Fatal("asked about a non-workspace file")
	}
	stop := make(chan struct{})
	defer close(stop)
	go c.think(stop) // only the message loop; no GitHub sync goroutine in this test
	if _, err := c.SayAboutFile("what does step 2 say?", pulsePath); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for eng.count() == 0 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if eng.count() != 1 {
		t.Fatal("question not sent")
	}
	eng.mu.Lock()
	req := eng.reqs[0]
	eng.mu.Unlock()
	ctx := req["context"].(string)
	if !strings.Contains(ctx, "READ-ONLY PROJECT FILE") || !strings.Contains(ctx, "Mira/home@live:"+pulsePath) || !strings.Contains(ctx, "CONTINUE") {
		t.Fatalf("file context missing: %s", ctx[len(ctx)-400:])
	}
	if a, ok := req["actions"].([]any); ok && len(a) != 0 {
		t.Fatalf("file question offered actions: %v", a)
	}
}
