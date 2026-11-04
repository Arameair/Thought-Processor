package main

import (
    "net/http"
    "net/http/httptest"
    "strings"
    "testing"

    "github.com/Arameair/Thought-Processor/core/engine"
    "github.com/Arameair/Thought-Processor/core/github"
    "github.com/Arameair/Thought-Processor/core/tass"
)

// Acceptance guard: a browser Origin, a loopback Host and an API credential are
// distinct checks. These tests never start a model or write to the real GitHub.
func TestAcceptanceLocalChatAPIAuthentication(t *testing.T) {
    core, err := tass.New(tass.Config{DataDir: t.TempDir()}, github.New(""), engine.New("http://127.0.0.1:1"))
    if err != nil { t.Fatal(err) }
    h := handler(core, "page-token-for-test", "api-key-for-test")

    cases := []struct {
        name, method, path, host, token, origin, contentType, body string
        want int
    }{
        {"status without credential", "GET", "/api/status", "127.0.0.1:8795", "", "", "", "", 401},
        {"say without credential", "POST", "/api/say", "127.0.0.1:8795", "", "", "application/json", "{\"text\":\"hello\"}", 401},
        {"say wrong credential", "POST", "/api/say", "127.0.0.1:8795", "incorrect", "", "application/json", "{\"text\":\"hello\"}", 401},
        {"say nonlocal host", "POST", "/api/say", "example.org:8795", "page-token-for-test", "", "application/json", "{\"text\":\"hello\"}", 401},
        {"say foreign browser origin", "POST", "/api/say", "127.0.0.1:8795", "page-token-for-test", "https://example.org", "application/json", "{\"text\":\"hello\"}", 403},
        {"say unsupported content type", "POST", "/api/say", "127.0.0.1:8795", "page-token-for-test", "", "text/plain", "{\"text\":\"hello\"}", 415},
        {"say empty text", "POST", "/api/say", "127.0.0.1:8795", "page-token-for-test", "", "application/json", "{\"text\":\" \"}", 400},
        {"say valid local request", "POST", "/api/say", "127.0.0.1:8795", "page-token-for-test", "http://127.0.0.1:8795", "application/json", "{\"text\":\"harmless acceptance probe\"}", 202},
    }
    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            req := httptest.NewRequest(tc.method, "http://"+tc.host+tc.path, strings.NewReader(tc.body))
            req.Host = tc.host
            if tc.token != "" { req.Header.Set("X-TASS-Token", tc.token) }
            if tc.origin != "" { req.Header.Set("Origin", tc.origin) }
            if tc.contentType != "" { req.Header.Set("Content-Type", tc.contentType) }
            w := httptest.NewRecorder()
            h.ServeHTTP(w, req)
            if w.Code != tc.want {
                t.Fatalf("HTTP status: got %d, want %d: %s", w.Code, tc.want, w.Body.String())
            }
        })
    }
    // 202 signifies admission to the local inbound channel, NOT completion or
    // a model reply. A separate receipt/correlation contract is still needed.
}

// The public landing page is deliberately readable, but never contains the
// reusable program credential. The ephemeral browser credential is expected.
func TestAcceptanceLandingPageDoesNotExposeProgramKey(t *testing.T) {
    core, err := tass.New(tass.Config{DataDir: t.TempDir()}, github.New(""), engine.New("http://127.0.0.1:1"))
    if err != nil { t.Fatal(err) }
    req := httptest.NewRequest("GET", "http://127.0.0.1:8795/", nil)
    w := httptest.NewRecorder()
    handler(core, "ephemeral-page-key", "private-program-key").ServeHTTP(w, req)
    if w.Code != http.StatusOK { t.Fatalf("landing page: HTTP %d", w.Code) }
    if strings.Contains(w.Body.String(), "private-program-key") { t.Fatal("program credential disclosed in page") }
    if !strings.Contains(w.Body.String(), "ephemeral-page-key") { t.Fatal("expected per-start page token is missing") }
    if w.Header().Get("Cache-Control") != "no-store" { t.Fatal("page with token must not be cached") }
}
