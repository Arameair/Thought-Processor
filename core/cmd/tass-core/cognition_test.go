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
    for _, required := range []string{"Inside TASS's mind.", "LIVE / READ ONLY", "test-page-token", "/api/cognition/library", "/api/cognition/cycles/", "PREVIEW ONLY"} {
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

func TestCognitionAPIRequiresOwnerTokenAndRejectsBadExports(t *testing.T) {
    h:=handler(nil,"test-page-token","test-api-token")
    paths:=[]string{
        "/api/cognition/library","/api/cognition/thoughts/thought_1",
        "/api/cognition/cycles/tp_1/timeline","/api/cognition/cycles/tp_1/export?format=json",
    }
    for _,path:=range paths{
        req:=httptest.NewRequest(http.MethodGet,"http://127.0.0.1:8795"+path,nil)
        r:=httptest.NewRecorder();h.ServeHTTP(r,req)
        if r.Code!=http.StatusUnauthorized {t.Fatalf("unauthenticated %s: got %d",path,r.Code)}
        if r.Header().Get("Cache-Control")!="no-store" {t.Fatalf("%s was cacheable",path)}
    }
    for _,path:=range []string{"/api/cognition/cycles/tp_1/export?format=raw",
        "/api/cognition/cycles/..%2Fsettings/export?format=json"} {
        req:=httptest.NewRequest(http.MethodGet,"http://127.0.0.1:8795"+path,nil)
        req.Header.Set("X-TASS-Token","test-page-token")
        rec:=httptest.NewRecorder();h.ServeHTTP(rec,req)
        if rec.Code!=http.StatusBadRequest&&rec.Code!=http.StatusNotFound {
            t.Fatalf("unsafe export %s: got %d",path,rec.Code)
        }
    }
}

func TestCognitionHTTPDeniedWhileIdentityLocked(t *testing.T) {
    // Server route auth is not enough: Identity must authorize each read too.
    // nil Core cannot serve a valid token. This only checks the HTTP token gate;
    // Core package tests exercise the actual LOCKED Identity decision.
    req:=httptest.NewRequest(http.MethodGet,"http://127.0.0.1:8795/api/cognition/library",nil)
    req.Header.Set("X-TASS-Token","wrong")
    rec:=httptest.NewRecorder()
    handler(nil,"test-page-token","test-api-token").ServeHTTP(rec,req)
    if rec.Code!=http.StatusUnauthorized {t.Fatalf("wrong key got HTTP %d",rec.Code)}
    if rec.Header().Get("Cache-Control")!="no-store" {t.Fatal("Cognition read must be no-store")}
}
