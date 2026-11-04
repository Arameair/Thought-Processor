package tass

import (
    "net/http"
    "net/http/httptest"
    "net/http/httputil"
    "net/url"
    "strings"
    "sync/atomic"
    "testing"
    "time"
)

// Uses the REAL core worker loop, policy fetch, Engine HTTP adapter and
// executor. The Engine and GitHub are TEST FIXTURES; no GPU inference occurs.
// Regression target: conversation intake must remain available while one model
// operation owns the slot. This does NOT prove Go Full Auto, which has no API yet.
func TestAcceptanceConversationIntakeDuringWorkingCycle(t *testing.T) {
    gh, fixture := newFakeGH(), newFakeEngine()
    defer gh.srv.Close()
    defer fixture.srv.Close()
    gh.set("Arameair/identity@main:policies/tass.json", policyJSON)
    gh.set("Arameair/TASS@live:tass.md", startDoc)

    c := newCore(t, t.TempDir(), gh, fixture)
    c.SyncOnce()

    target, err := url.Parse(fixture.srv.URL)
    if err != nil { t.Fatal(err) }
    proxy := httputil.NewSingleHostReverseProxy(target)
    started := make(chan struct{})
    release := make(chan struct{})
    var entered atomic.Int32
    slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.Method == "POST" && r.URL.Path == "/api/cycles" && entered.Add(1) == 1 {
            close(started)
            <-release
        }
        proxy.ServeHTTP(w, r)
    }))
    defer slow.Close()
    c.engine.URL = slow.URL

    stop := make(chan struct{})
    done := make(chan struct{})
    go func() { c.Run(stop); close(done) }()
    defer func() {
        select { case <-release: default: close(release) }
        close(stop)
        select { case <-done: case <-time.After(5*time.Second): t.Error("worker did not stop") }
    }()

    if err := c.Say("first harmless message"); err != nil { t.Fatal(err) }
    select {
    case <-started:
    case <-time.After(5*time.Second):
        t.Fatal("first cycle never reached Engine fixture")
    }
    c.mu.Lock()
    working := c.busy
    c.mu.Unlock()
    if !working { t.Fatal("core did not report WORKING during the in-flight cycle") }

    if err := c.Say("second harmless message"); err != nil {
        t.Fatalf("owner chat must admit a second message while working: %v", err)
    }
    if n := len(c.inbox); n != 1 { t.Fatalf("expected one queued message, got %d", n) }
    close(release)

    deadline := time.Now().Add(5*time.Second)
    for time.Now().Before(deadline) {
        if fixture.count() == 2 {
            c.mu.Lock()
            n := len(c.cycles)
            c.mu.Unlock()
            if n == 2 { break }
        }
        time.Sleep(20*time.Millisecond)
    }
    c.mu.Lock()
    cycles := append([]interface{}(nil), func() []interface{} {
        out := make([]interface{}, len(c.cycles))
        for i, cycle := range c.cycles { out[i] = cycle }
        return out
    }()...)
    c.mu.Unlock()
    if len(cycles) != 2 || fixture.count() != 2 {
        t.Fatalf("wanted exactly two completed sequential cycles, got %d cycles, %d engine calls", len(cycles), fixture.count())
    }
    fixture.mu.Lock()
    first, second := fixture.reqs[0], fixture.reqs[1]
    fixture.mu.Unlock()
    if first["request"] != "first harmless message" || second["request"] != "second harmless message" {
        t.Fatalf("messages lost or reordered: %v then %v", first["request"], second["request"])
    }
    if first["cycle_id"] == second["cycle_id"] || !strings.HasPrefix(first["cycle_id"].(string), "tp_") {
        t.Fatal("cycle IDs not distinct/correlated")
    }
}
