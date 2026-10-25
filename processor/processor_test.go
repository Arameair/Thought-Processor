package main

// Layer L1/L2 tests. Servers created with httptest here are FIXTURES and are
// labeled as such; they prove Processor's own behavior, never integration.

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

type harness struct {
	t    *testing.T
	app  *App
	base string
	host string
	quit chan struct{}
	done chan error
	cook string
	csrf string
	once sync.Once
}

func start(t *testing.T, dir string) *harness {
	t.Helper()
	if dir == "" {
		dir = t.TempDir()
	}
	h := &harness{t: t, quit: make(chan struct{}), done: make(chan error, 1)}
	ready := make(chan *App, 1)
	go func() { h.done <- run(Config{DataDir: dir, Port: 0}, h.quit, ready, nil) }()
	select {
	case h.app = <-ready:
	case err := <-h.done:
		t.Fatalf("run failed: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("timeout starting")
	}
	h.host = fmt.Sprintf("127.0.0.1:%d", h.app.port)
	h.base = "http://" + h.host
	t.Cleanup(func() { h.stop() })
	return h
}

func (h *harness) stop() {
	h.once.Do(func() {
		close(h.quit)
		select {
		case <-h.done:
		case <-time.After(5 * time.Second):
			h.t.Error("Processor did not stop within 5 s")
		}
	})
}

func (h *harness) do(method, path, body string, hdr map[string]string) (int, string) {
	req, _ := http.NewRequest(method, h.base+path, strings.NewReader(body))
	for k, v := range hdr {
		if k == "Host" {
			req.Host = v
		} else {
			req.Header.Set(k, v)
		}
	}
	if h.cook != "" {
		req.Header.Set("Cookie", cookieName+"="+h.cook)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		h.t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	for _, c := range resp.Cookies() {
		if c.Name == cookieName {
			h.cook = c.Value
		}
	}
	return resp.StatusCode, string(b)
}

func (h *harness) signIn() {
	code, body := h.do("POST", "/api/session", `{"code":"`+h.app.sess.Code()+`"}`,
		map[string]string{"Origin": h.base, "Content-Type": "application/json"})
	if code != 200 {
		h.t.Fatalf("sign-in: %d %s", code, body)
	}
	var d struct{ CSRF string }
	json.Unmarshal([]byte(body), &d)
	h.csrf = d.CSRF
}

func (h *harness) post(path, body string) (int, string) {
	return h.do("POST", path, body, map[string]string{"Origin": h.base, "Content-Type": "application/json", "X-TP-CSRF": h.csrf})
}

func TestEmptyProfileShell(t *testing.T) {
	h := start(t, "")
	code, body := h.do("GET", "/", "", nil)
	if code != 200 || !strings.Contains(body, "Thought Processor") {
		t.Fatalf("index: %d", code)
	}
	// Private state needs a session.
	if code, _ := h.do("GET", "/api/state", "", nil); code != 401 {
		t.Fatalf("state without session: %d", code)
	}
	h.signIn()
	code, body = h.do("GET", "/api/state", "", nil)
	if code != 200 {
		t.Fatalf("state: %d", code)
	}
	var s struct {
		Identity, Engine DepView
		Next             NextAction  `json:"next_action"`
		Ops              []Operation `json:"operations"`
		Self             map[string]any
	}
	json.Unmarshal([]byte(body), &s)
	if s.Identity.State != "NOT_CONFIGURED" || s.Engine.State != "NOT_CONFIGURED" || s.Next.Code != "set_engine" {
		t.Fatalf("unexpected empty state: %+v %+v %+v", s.Identity.State, s.Engine.State, s.Next)
	}
	for _, op := range s.Ops {
		want := map[ProtectedAction]string{ActStart: "BLOCKED", ActStop: "BLOCKED", ActAttach: "AVAILABLE", ActObserve: "NOT YET"}[op.Action]
		if op.Status != want || (op.Action != ActAttach && op.Enabled) {
			t.Fatalf("operation %s: %s (enabled %v), want %s", op.Action, op.Status, op.Enabled, want)
		}
	}
	if owned := s.Self["owned_processes"].([]any); len(owned) != 0 {
		t.Fatal("claims owned processes")
	}
}

func TestProtectedOperationsRefusedServerSide(t *testing.T) {
	h := start(t, "")
	h.signIn()
	// attach/observe moved to the Engine card (attach code); the old
	// parameterless boundary answers "use the Engine card" with no effect.
	for _, a := range []string{"attach", "observe"} {
		if code, body := h.post("/api/engine/"+a, `{}`); code != 409 || !strings.Contains(body, "USE_ENGINE_CARD") {
			t.Fatalf("%s: %d %s", a, code, body)
		}
	}
	for _, a := range []string{"start", "stop"} {
		code, body := h.post("/api/engine/"+a, `{}`)
		if code != 501 || !strings.Contains(body, `"BLOCKED"`) || !strings.Contains(body, "effects") {
			t.Fatalf("%s: %d %s", a, code, body)
		}
		// Injection attempts: any parameter is refused before the authority is asked.
		code, body = h.post("/api/engine/"+a, `{"path":"/bin/sh","args":["-c","id"],"endpoint":"http://evil.example"}`)
		if code != 400 || !strings.Contains(body, "parameters_not_accepted") {
			t.Fatalf("%s injection: %d %s", a, code, body)
		}
	}
	if code, _ := h.post("/api/engine/..%2f..%2fshutdown", `{}`); code != 404 {
		t.Fatalf("path trick: %d", code)
	}
	if code, _ := h.post("/api/engine/run", `{}`); code != 404 {
		t.Fatalf("unknown action: %d", code)
	}
	// GET never acts.
	if code, _ := h.do("GET", "/api/engine/start", "", nil); code != 405 {
		t.Fatalf("GET start: %d", code)
	}
}

func TestRequestGuards(t *testing.T) {
	h := start(t, "")
	h.signIn()
	cases := []struct {
		name string
		hdr  map[string]string
		want int
	}{
		{"untrusted Host (DNS rebinding)", map[string]string{"Host": "evil.example:80", "Origin": h.base, "Content-Type": "application/json", "X-TP-CSRF": h.csrf}, 421},
		{"no Origin", map[string]string{"Content-Type": "application/json", "X-TP-CSRF": h.csrf}, 403},
		{"cross Origin", map[string]string{"Origin": "http://evil.example", "Content-Type": "application/json", "X-TP-CSRF": h.csrf}, 403},
		{"form post", map[string]string{"Origin": h.base, "Content-Type": "application/x-www-form-urlencoded", "X-TP-CSRF": h.csrf}, 415},
		{"no CSRF", map[string]string{"Origin": h.base, "Content-Type": "application/json"}, 403},
		{"wrong CSRF", map[string]string{"Origin": h.base, "Content-Type": "application/json", "X-TP-CSRF": strings.Repeat("0", 64)}, 403},
	}
	for _, c := range cases {
		code, _ := h.do("POST", "/api/candidates", `{"identity":"127.0.0.1:9"}`, c.hdr)
		if code != c.want {
			t.Errorf("%s: got %d want %d", c.name, code, c.want)
		}
	}
	saved := h.cook
	h.cook = ""
	if code, _ := h.do("POST", "/api/candidates", `{"identity":"127.0.0.1:9"}`, map[string]string{"Origin": h.base, "Content-Type": "application/json", "X-TP-CSRF": h.csrf}); code != 401 {
		t.Errorf("no session: %d", code)
	}
	h.cook = saved
	h.app.mu.Lock()
	got := h.app.cands.Identity
	h.app.mu.Unlock()
	if got != "" {
		t.Fatalf("rejected requests changed state: %q", got)
	}
	if _, err := os.Stat(filepath.Join(h.app.prof.Dir, "candidates.json")); err == nil {
		t.Fatal("rejected requests wrote a file")
	}
	// No CORS header is ever granted.
	req, _ := http.NewRequest("OPTIONS", h.base+"/api/state", nil)
	req.Header.Set("Origin", "http://evil.example")
	resp, _ := http.DefaultClient.Do(req)
	if resp.Header.Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("CORS granted")
	}
	resp.Body.Close()
}

func TestWrongCodeAndSessionIsolation(t *testing.T) {
	h := start(t, "")
	code, _ := h.do("POST", "/api/session", `{"code":"nope"}`, map[string]string{"Origin": h.base, "Content-Type": "application/json"})
	if code != 401 {
		t.Fatalf("wrong code: %d", code)
	}
	code, _ = h.do("POST", "/api/session", `{"code":"`+h.app.sess.Code()+`"}`, map[string]string{"Origin": "http://evil.example", "Content-Type": "application/json"})
	if code != 403 {
		t.Fatalf("cross-origin sign-in: %d", code)
	}
}

func TestCandidateValidation(t *testing.T) {
	bad := []string{"http://192.168.1.5:7777", "http://example.com:80", "https://127.0.0.1:7777", "http://u:p@127.0.0.1:7777",
		"http://127.0.0.1:7777/v1/authorization", "http://127.0.0.1:7777?x=1", "127.0.0.1", "127.0.0.1:0", "127.0.0.1:70000",
		"file:///etc/passwd", "http://127.0.0.1:7860", "http://[::ffff:10.0.0.1]:80", "http://0.0.0.0:7777"}
	for _, b := range bad {
		if _, err := parseCandidate(b, 7860); err == nil {
			t.Errorf("accepted %q", b)
		}
	}
	good := map[string]string{"127.0.0.1:7777": "http://127.0.0.1:7777", "localhost:7798": "http://127.0.0.1:7798", "http://[::1]:7777/": "http://[::1]:7777"}
	for in, want := range good {
		if got, err := parseCandidate(in, 7860); err != nil || got != want {
			t.Errorf("%q -> %q, %v", in, got, err)
		}
	}
}

// FIXTURE (L2): stand-in HTTP servers. They prove only Processor's
// classification and redaction, not anything about real Identity.
func TestIdentityProbeClassificationFixture(t *testing.T) {
	const canary = "CANARYsecretTOKEN123"
	mk := func(status int, body string, delay time.Duration) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(delay)
			if r.URL.Path != "/api/health" {
				w.WriteHeader(404)
				return
			}
			w.WriteHeader(status)
			io.WriteString(w, body)
		}))
	}
	ctx := context.Background()
	claims := mk(200, `{"app":"identity","version":"`+canary+`"}`, 0)
	defer claims.Close()
	other := mk(200, `{"app":"thought-engine"}`, 0)
	defer other.Close()
	none := mk(404, ``, 0)
	defer none.Close()
	slow := mk(200, `{"app":"identity"}`, 3*time.Second)
	defer slow.Close()
	redir := httptest.NewServer(http.RedirectHandler("http://example.com/", 302))
	defer redir.Close()

	r := ProbeIdentity(ctx, claims.URL)
	if r.Metadata != "RESPONDED" || !r.ClaimMatches {
		t.Fatalf("claims: %+v", r)
	}
	if v := identityView(claims.URL, &r, time.Now(), false, ""); v.State != "CLAIMS_IDENTITY_UNVERIFIED" {
		t.Fatalf("claim must stay unverified: %s", v.State)
	}
	if r := ProbeIdentity(ctx, other.URL); r.ClaimMatches || r.Metadata != "RESPONDED" {
		t.Fatalf("other: %+v", r)
	}
	if r := ProbeIdentity(ctx, none.URL); r.Metadata != "NO_PUBLIC_METADATA" {
		t.Fatalf("none: %+v", r)
	}
	t0 := time.Now()
	if r := ProbeIdentity(ctx, slow.URL); r.Metadata != "FAILED" || time.Since(t0) > 2500*time.Millisecond {
		t.Fatalf("slow: %+v in %v", r, time.Since(t0))
	}
	if r := ProbeIdentity(ctx, redir.URL); r.Metadata != "NO_PUBLIC_METADATA" || r.HTTPStatus != 302 {
		t.Fatalf("redirect must not be followed: %+v", r)
	}

	// Canary must never reach Copy Context or the log file.
	h := start(t, "")
	h.signIn()
	code, body := h.post("/api/candidates", `{"identity":"`+strings.TrimPrefix(claims.URL, "http://")+`"}`)
	if code != 200 || !strings.Contains(body, "UNVERIFIED") {
		t.Fatalf("set: %d", code)
	}
	_, ctxText := h.do("GET", "/api/context", "", nil)
	if strings.Contains(ctxText, canary) || strings.Contains(ctxText, h.app.sess.Code()) || strings.Contains(ctxText, h.csrf) || strings.Contains(ctxText, h.cook) {
		t.Fatalf("Copy Context leaked a secret:\n%s", ctxText)
	}
	h.stop()
	logb, _ := os.ReadFile(h.app.prof.LogPath())
	if strings.Contains(string(logb), canary) || strings.Contains(string(logb), h.cook) {
		t.Fatal("log leaked a secret")
	}
}

func TestFreshnessAndDisappearance(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	target := "http://" + ln.Addr().String()
	r := ProbeEngine(context.Background(), target)
	if r.Port != "OPEN" {
		t.Fatalf("open: %+v", r)
	}
	if v := engineView(target, &r, time.Now(), linkRecord{}); v.State != "PORT_OPEN_UNIDENTIFIED" {
		t.Fatalf("state %s", v.State)
	}
	if v := engineView(target, &r, time.Now().Add(staleAfter+time.Second), linkRecord{}); v.State != "STALE" {
		t.Fatalf("old result must be STALE, got %s", v.State)
	}
	ln.Close()
	r2 := ProbeEngine(context.Background(), target)
	if r2.Port != "CLOSED" {
		t.Fatalf("closed: %+v", r2)
	}
}

func TestSingleInstancePerProfile(t *testing.T) {
	dir := t.TempDir()
	h := start(t, dir)
	err := run(Config{DataDir: dir, Port: 0}, make(chan struct{}), nil, nil)
	if err == nil || !strings.Contains(err.Error(), "already using profile") || !strings.Contains(err.Error(), h.app.runID) {
		t.Fatalf("second instance: %v", err)
	}
}

func TestRestartRevalidatesSession(t *testing.T) {
	dir := t.TempDir()
	h := start(t, dir)
	h.signIn()
	oldCook, oldRun := h.cook, h.app.runID
	h.stop()
	if _, err := os.Stat(filepath.Join(dir, "runtime", "run.json")); err == nil {
		t.Fatal("clean stop left run.json")
	}
	h2 := start(t, dir)
	if h2.app.runID == oldRun || h2.app.prof.Previous != nil {
		t.Fatal("restart should be a new, clean run")
	}
	h2.cook = oldCook
	if code, _ := h2.do("GET", "/api/state", "", nil); code != 401 {
		t.Fatalf("old session accepted after restart: %d", code)
	}
}

func TestUncleanPreviousRunDetected(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "runtime"), 0o700)
	writeJSONAtomic(filepath.Join(dir, "runtime", "run.json"), RunRecord{RunID: "run_dead", PID: 999999, Port: 1})
	h := start(t, dir)
	if h.app.prof.Previous == nil || h.app.prof.Previous.RunID != "run_dead" {
		t.Fatal("unclean previous run not reported")
	}
}

func TestCorruptCandidatesLeftUnchanged(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "candidates.json"), []byte("{not json"), 0o600)
	h := start(t, dir)
	h.signIn()
	_, body := h.do("GET", "/api/state", "", nil)
	if !strings.Contains(body, `"fix_profile"`) {
		t.Fatalf("expected fix_profile next action: %s", body)
	}
	b, _ := os.ReadFile(filepath.Join(dir, "candidates.json"))
	if string(b) != "{not json" {
		t.Fatal("corrupt file was modified")
	}
}

func TestUnwritableProfile(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores permissions")
	}
	dir := t.TempDir()
	os.Chmod(dir, 0o500)
	defer os.Chmod(dir, 0o700)
	if err := run(Config{DataDir: filepath.Join(dir, "p"), Port: 0}, make(chan struct{}), nil, nil); err == nil {
		t.Fatal("expected failure")
	}
}

func TestFlowMetricsSelection(t *testing.T) {
	h := start(t, "")
	h.signIn()
	h.post("/api/engine/stop", `{}`)
	h.post("/api/engine/start", `{"path":"/bin/sh"}`)
	h.do("POST", "/api/engine/stop", `{}`, map[string]string{"Origin": "http://evil.example", "Content-Type": "application/json"})
	code, body := h.do("GET", "/api/flow", "", nil)
	var f struct{ Traces []Trace }
	json.Unmarshal([]byte(body), &f)
	if code != 200 || len(f.Traces) != 2 {
		t.Fatalf("flow: %d %d traces", code, len(f.Traces))
	}
	// newest first: the parameter refusal, then the blocked stop
	if f.Traces[0].Outcome != "REFUSED" || f.Traces[1].Outcome != "BLOCKED" || f.Traces[1].Stages[1].Status != "NOT AVAILABLE" || f.Traces[1].Stages[3].Status != "NONE" {
		t.Fatalf("flow stages wrong: %+v", f.Traces)
	}
	_, body = h.do("GET", "/api/metrics", "", nil)
	var m struct {
		Refused  map[string]int `json:"refused_operations"`
		Rejected map[string]int `json:"rejected_requests"`
	}
	json.Unmarshal([]byte(body), &m)
	if m.Refused["stop"] != 1 || m.Refused["start"] != 1 || m.Rejected["untrusted_origin"] != 1 {
		t.Fatalf("metrics: %+v", m)
	}
	_, body = h.do("GET", "/api/state", "", nil)
	var s struct{ Selection map[string]map[string]string }
	json.Unmarshal([]byte(body), &s)
	if s.Selection["entity"]["status"] != "NONE" || s.Selection["instance"]["status"] != "UNKNOWN" {
		t.Fatalf("selection: %+v", s.Selection)
	}
	_, logs := h.do("GET", "/api/logs?kind=control", "", nil)
	if !strings.Contains(logs, "blocked_no_identity_approval_service") || strings.Contains(logs, "guard") && strings.Contains(logs, "bad_origin") {
		t.Fatalf("log filter: %s", logs)
	}
	_, ctx := h.do("GET", "/api/context", "", nil)
	if !strings.Contains(ctx, "authorized=NOT AVAILABLE") {
		t.Fatalf("context lacks flow: %s", ctx)
	}
}
