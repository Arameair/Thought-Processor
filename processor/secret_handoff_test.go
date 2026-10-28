package main

import (
    "bufio"
    "bytes"
    "context"
    "errors"
    "io/fs"
    "os"
    "os/exec"
    "path/filepath"
    "strings"
    "testing"
    "time"
)

// Runs the REAL Processor binary with a private inherited pipe and inspects
// normal stdout, stderr and profile files for session-link leakage.
func TestRealProcessorNeverPrintsOrPersistsPrivateLink(t *testing.T) {
    if testing.Short() { t.Skip("builds a real Processor executable") }
    dir := t.TempDir()
    binary := filepath.Join(dir, "processor-real")
    build := exec.Command("go", "build", "-o", binary, ".")
    if output, err := build.CombinedOutput(); err != nil {
        t.Fatalf("build failed: %v %s", err, output)
    }
    readPipe, writePipe, err := os.Pipe()
    if err != nil { t.Fatal(err) }
    defer readPipe.Close()
    profile := filepath.Join(dir, "profile")
    ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
    defer cancel()
    cmd := exec.CommandContext(ctx, binary, "-port", "0",
        "-data", profile, "-no-browser", "-private-link-fd", "3")
    cmd.ExtraFiles = []*os.File{writePipe}
    var stdout, stderr bytes.Buffer
    cmd.Stdout, cmd.Stderr = &stdout, &stderr
    if err := cmd.Start(); err != nil { t.Fatal(err) }
    _ = writePipe.Close()
    defer func() {
        if cmd.Process != nil { _ = cmd.Process.Kill() }
        _ = cmd.Wait()
    }()
    type handoff struct { link string; err error }
    got := make(chan handoff, 1)
    go func() {
        line, e := bufio.NewReader(readPipe).ReadString('\n')
        got <- handoff{line, e}
    }()
    var link string
    select {
    case outcome := <-got:
        if outcome.err != nil { t.Fatalf("pipe handoff failed: %v", outcome.err) }
        link = strings.TrimSpace(outcome.link)
        if !strings.HasPrefix(link, "http://127.0.0.1:") || !strings.Contains(link, "/#code=") {
            t.Fatal("private pipe did not deliver the expected local URL")
        }
    case <-ctx.Done():
        t.Fatal("timed out waiting for anonymous private pipe")
    }
    // Allow the asynchronous log writer to flush after first startup.
    time.Sleep(150*time.Millisecond)
    if strings.Contains(stdout.String(), "#code=") || strings.Contains(stderr.String(), "#code=") ||
        strings.Contains(stdout.String(), link) || strings.Contains(stderr.String(), link) {
        t.Fatal("private session link leaked to console streams")
    }
    err = filepath.WalkDir(profile, func(path string, entry fs.DirEntry, walkErr error) error {
        if walkErr != nil { return walkErr }
        if entry.IsDir() { return nil }
        b, e := os.ReadFile(path)
        if e != nil { return e }
        if bytes.Contains(b, []byte(link)) || bytes.Contains(b, []byte("#code=")) {
            return errors.New("private browser session URL was persisted")
        }
        return nil
    })
    if err != nil { t.Fatal(err) }
}
