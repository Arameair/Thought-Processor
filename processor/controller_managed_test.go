package main

import (
    "bytes"
    "strings"
    "testing"
    "time"
)

func TestControllerManagedStartupDoesNotPrintOperatorCode(t *testing.T) {
    t.Setenv("PC_MANAGED_BY_CONTROLLER", "1")
    var out bytes.Buffer
    quit := make(chan struct{})
    ready := make(chan *App, 1)
    result := make(chan error, 1)
    go func() { result <- run(Config{DataDir: t.TempDir(), Port: 0}, quit, ready, &out) }()

    var app *App
    select {
    case app = <-ready:
    case err := <-result:
        t.Fatalf("server exited before ready: %v", err)
    case <-time.After(5 * time.Second):
        t.Fatal("startup timeout")
    }
    close(quit)
    select {
    case err := <-result:
        if err != nil { t.Fatalf("shutdown: %v", err) }
    case <-time.After(5 * time.Second):
        t.Fatal("shutdown timeout")
    }
    printed := out.String()
    if strings.Contains(printed, "#code=") || strings.Contains(printed, app.sess.Code()) {
        t.Fatal("Controller stdout contained the operator access credential")
    }
    if !strings.Contains(printed, "operator code: stored locally in") || !strings.Contains(printed, "local sign-in required") {
        t.Fatalf("Controller-safe sign-in instructions missing from output: %s", printed)
    }
}

func TestStandaloneStartupStillShowsOneTimeSignInLink(t *testing.T) {
    t.Setenv("PC_MANAGED_BY_CONTROLLER", "")
    var out bytes.Buffer
    quit := make(chan struct{})
    ready := make(chan *App, 1)
    result := make(chan error, 1)
    go func() { result <- run(Config{DataDir: t.TempDir(), Port: 0}, quit, ready, &out) }()

    var app *App
    select {
    case app = <-ready:
    case err := <-result:
        t.Fatalf("server exited before ready: %v", err)
    case <-time.After(5 * time.Second):
        t.Fatal("startup timeout")
    }
    close(quit)
    select {
    case err := <-result:
        if err != nil { t.Fatalf("shutdown: %v", err) }
    case <-time.After(5 * time.Second):
        t.Fatal("shutdown timeout")
    }
    printed := out.String()
    if !strings.Contains(printed, "#code="+app.sess.Code()) {
        t.Fatal("standalone foreground launch must still display operator sign-in URL")
    }
}
