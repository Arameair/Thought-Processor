package app

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/Arameair/Thought-Processor/processor/internal/store"
)

type h struct {
	t    *testing.T
	a    *App
	base string
	c    *http.Client
	csrf string
}

func start(t *testing.T) *h {
	t.Setenv("IDENTITY_LOCATOR_DIR", t.TempDir()) // no real Identity locators
	// A deliberately unavailable fake Identity owns a random loopback port.
	// Neither discovery nor the protected-action test probes real port 7790.
	fakeIdentity := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "test Identity unavailable", http.StatusServiceUnavailable)
	}))
	t.Cleanup(fakeIdentity.Close)
	t.Setenv("THOUGHT_PROCESSOR_IDENTITY_DISCOVERY_DEFAULT", fakeIdentity.URL)
	prof, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	a := New(prof, Build{Version: "test"})
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	ctx, cancel := context.WithCancel(context.Background())
	ready := make(chan string, 1)
	done := make(chan struct{})
	go func() { a.Run(ctx, ln, func(l string) { ready <- l }); close(done) }()
	link := <-ready
	t.Cleanup(func() { cancel(); <-done; prof.Close() })
	jar, _ := cookiejar.New(nil)
	x := &h{t: t, a: a, base: "http://127.0.0.1:" + strconv.Itoa(a.Port), c: &http.Client{Jar: jar}}
	code := link[strings.Index(link, "#code=")+6:]
	st, b := x.post("/api/session", `{"code":"`+code+`"}`, nil)
	if st != 200 {
		t.Fatalf("session %d %s", st, b)
	}
	var m map[string]string
	json.Unmarshal([]byte(b), &m)
	x.csrf = m["csrf"]
	return x
}

func (x *h) post(path, body string, hdr map[string]string) (int, string) {
	req, _ := http.NewRequest("POST", x.base+path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", x.base)
	if x.csrf != "" {
		req.Header.Set("X-TP-CSRF", x.csrf)
	}
	for k, v := range hdr {
		if k == "Host" {
			req.Host = v
		} else if v == "" {
			req.Header.Del(k)
		} else {
			req.Header.Set(k, v)
		}
	}
	resp, err := x.c.Do(req)
	if err != nil {
		x.t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b)
}

func TestBrowserBoundaries(t *testing.T) {
	x := start(t)
	if st, _ := x.post("/api/devmode", `{"on":true}`, map[string]string{"X-TP-CSRF": ""}); st != 403 {
		t.Fatalf("missing CSRF accepted: %d", st)
	}
	if st, _ := x.post("/api/devmode", `{"on":true}`, map[string]string{"Origin": "http://evil.example"}); st != 403 {
		t.Fatalf("foreign origin accepted: %d", st)
	}
	if st, _ := x.post("/api/devmode", `{"on":true}`, map[string]string{"Host": "evil.example"}); st != 421 {
		t.Fatalf("wrong host accepted: %d", st)
	}
	resp, _ := http.Get(x.base + "/api/state")
	if resp.StatusCode != 401 {
		t.Fatalf("state readable without session: %d", resp.StatusCode)
	}
	if st, _ := x.post("/api/session", `{"code":"nope"}`, nil); st != 401 {
		t.Fatal("wrong link accepted")
	}
}

func TestPrivateOperatorLinkIsNeverPersisted(t *testing.T) {
	x := start(t)
	if _, err := os.Stat(x.a.Prof.CodePath()); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("private browser link persisted to profile runtime: %v", err)
	}
}

func TestProtectedActionsFailClosedWithoutIdentity(t *testing.T) {
	x := start(t)
	for _, p := range []string{"/api/features/thought-engine/plan", "/api/identities", "/api/github/connect"} {
		st, b := x.post(p, `{"name":"x","kind":"person"}`, nil)
		if st != 503 || !strings.Contains(b, "can't be reached") {
			t.Fatalf("%s without Identity: %d %s", p, st, b)
		}
	}
	// Developer Mode is display-only: turning it on doesn't unblock anything.
	x.post("/api/devmode", `{"on":true}`, nil)
	if st, _ := x.post("/api/features/thought-engine/plan", `{}`, nil); st != 503 {
		t.Fatal("Developer Mode changed what is allowed")
	}
	st, b := x.post("/api/connect/retry", `{}`, nil)
	var s map[string]any
	json.Unmarshal([]byte(b), &s)
	if st != 200 || s["verified"] != false || s["connection"].(map[string]any)["state"] != CSearching {
		t.Fatalf("state %d %v", st, s["connection"])
	}
	if _, ok := s["dev"]; !ok {
		t.Fatal("developer details missing in Developer Mode")
	}
}
