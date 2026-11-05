package main

import (
    "net/http"
    "net/http/httptest"
    "strings"
    "testing"
)

func TestCognitionPageServedLocally(t *testing.T) {
    // GET /cognition serves only the read-only UI. Core API requests happen
    // after the page has loaded, and carry the same page token as /dev.
    req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:8795/cognition", nil)
    rec := httptest.NewRecorder()
    handler(nil, "test-page-token", "test-api-token").ServeHTTP(rec, req)
    if rec.Code != http.StatusOK {
        t.Fatalf("GET /cognition: got %d, want 200", rec.Code)
    }
    body := rec.Body.String()
    for _, required := range []string{"Inside TASS's mind.", "LIVE READ ONLY", "test-page-token", "/api/status", "/api/brain", "PREVIEW ONLY"} {
        if !strings.Contains(body, required) {
            t.Errorf("Cognition HTML missing %q", required)
        }
    }
    if strings.Contains(body, "__PAGE_TOKEN__") {
        t.Fatal("unreplaced page-token placeholder")
    }
    if got := rec.Header().Get("Cache-Control"); got != "no-store" {
        t.Errorf("Cache-Control = %q, want no-store", got)
    }
}

func TestCognitionRejectsNonLocalHost(t *testing.T) {
    req := httptest.NewRequest(http.MethodGet, "http://example.com/cognition", nil)
    rec := httptest.NewRecorder()
    handler(nil, "test-page-token", "test-api-token").ServeHTTP(rec, req)
    if rec.Code != http.StatusForbidden {
        t.Fatalf("nonlocal host: got %d, want 403", rec.Code)
    }
}
