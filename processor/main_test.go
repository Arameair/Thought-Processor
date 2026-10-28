package main

import (
    "io"
    "os"
    "strings"
    "testing"
)

func TestPrivateLinkOnlyToAnonymousPipe(t *testing.T) {
    r, w, err := os.Pipe()
    if err != nil { t.Fatal(err) }
    defer r.Close()
    if w.Fd() < 3 { t.Fatal("unsafe descriptor") }
    secret := "http://127.0.0.1:7860/#code=" + strings.Repeat("a", 48)
    if err := handoffPrivateLink(int(w.Fd()), secret); err != nil { t.Fatal(err) }
    b, err := io.ReadAll(r)
    if err != nil { t.Fatal(err) }
    if string(b) != secret+"\n" { t.Fatal("anonymous pipe did not receive expected one-time link") }
}

func TestPrivateLinkRefusesRegularFileAndStandardStreams(t *testing.T) {
    if err := handoffPrivateLink(1,"PRIVATE"); err == nil { t.Fatal("accepted stdout") }
    if err := handoffPrivateLink(2,"PRIVATE"); err == nil { t.Fatal("accepted stderr") }
    f, err := os.CreateTemp(t.TempDir(), "must-remain-empty")
    if err != nil { t.Fatal(err) }
    if err := handoffPrivateLink(int(f.Fd()), "PRIVATE"); err == nil {
        t.Fatal("accepted ordinary file")
    }
    b, err := os.ReadFile(f.Name())
    if err != nil { t.Fatal(err) }
    if len(b) != 0 { t.Fatal("wrote secret into a file") }
}
