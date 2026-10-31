package app

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/cookiejar"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Arameair/Thought-Processor/processor/internal/store"
)

// Launch tickets: the browser gets a one-time, short-lived ticket instead of
// the run's reusable private code.
func TestLaunchTicketIsSingleUseAndNeedsTheCode(t *testing.T) {
	x := start(t)
	code := x.a.Sess.Code()
	if st, _ := x.post("/api/launch-ticket", `{"code":"nope"}`, nil); st != 401 {
		t.Fatalf("ticket minted without the private code: %d", st)
	}
	if st, _ := x.post("/api/launch-ticket", `{"code":"`+code+`"}`, map[string]string{"Origin": "http://evil.example"}); st != 403 {
		t.Fatalf("ticket minted from a foreign origin: %d", st)
	}
	st, b := x.post("/api/launch-ticket", `{"code":"`+code+`"}`, nil)
	if st != 200 {
		t.Fatalf("ticket %d %s", st, b)
	}
	var m struct {
		Ticket string `json:"ticket"`
	}
	json.Unmarshal([]byte(b), &m)
	if len(m.Ticket) != 48 || m.Ticket == code {
		t.Fatalf("bad ticket %q", m.Ticket)
	}
	// A fresh browser (no cookies) exchanges the ticket once.
	jar, _ := cookiejar.New(nil)
	browser := &h{t: t, a: x.a, base: x.base, c: &http.Client{Jar: jar}}
	if st, b := browser.post("/api/session", `{"code":"`+m.Ticket+`"}`, nil); st != 200 {
		t.Fatalf("ticket refused first time: %d %s", st, b)
	}
	jar2, _ := cookiejar.New(nil)
	replay := &h{t: t, a: x.a, base: x.base, c: &http.Client{Jar: jar2}}
	if st, _ := replay.post("/api/session", `{"code":"`+m.Ticket+`"}`, nil); st != 401 {
		t.Fatalf("ticket accepted twice: %d", st)
	}
	// The private code itself still works for the launcher.
	if st, _ := replay.post("/api/session", `{"code":"`+code+`"}`, nil); st != 200 {
		t.Fatalf("private code stopped working: %d", st)
	}
}

func TestLaunchTicketExpires(t *testing.T) {
	s := NewSessions()
	tk, ok, _ := s.MintTicket(s.Code())
	if !ok {
		t.Fatal("mint failed")
	}
	s.tickets[tk] = time.Now().Add(-time.Second)
	if _, _, ok, _ := s.Exchange(tk); ok {
		t.Fatal("expired ticket accepted")
	}
	if _, _, ok, _ := s.Exchange(tk); ok {
		t.Fatal("expired ticket accepted on retry")
	}
}

func TestLaunchTicketsAreBounded(t *testing.T) {
	s := NewSessions()
	for i := 0; i < MaxTickets; i++ {
		if _, ok, _ := s.MintTicket(s.Code()); !ok {
			t.Fatalf("mint %d refused", i)
		}
	}
	if _, ok, _ := s.MintTicket(s.Code()); ok {
		t.Fatal("unbounded outstanding tickets")
	}
}

func TestNoLinkFileKeepsLinkOffDisk(t *testing.T) {
	for _, noFile := range []bool{false, true} {
		t.Setenv("IDENTITY_LOCATOR_DIR", t.TempDir())
		prof, err := store.Open(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		a := New(prof, Build{Version: "test"})
		a.NoLinkFile = noFile
		if noFile { // a stale link left by a killed earlier run must be removed
			os.MkdirAll(filepath.Dir(prof.CodePath()), 0o700)
			os.WriteFile(prof.CodePath(), []byte("stale\n"), 0o600)
		}
		ln, _ := net.Listen("tcp", "127.0.0.1:0")
		ctx, cancel := context.WithCancel(context.Background())
		ready := make(chan string, 1)
		done := make(chan struct{})
		go func() { a.Run(ctx, ln, func(l string) { ready <- l }); close(done) }()
		link := <-ready
		_, err = os.Stat(prof.CodePath())
		if noFile && err == nil {
			t.Fatal("-no-link-file still wrote operator-link.txt")
		}
		if !noFile && err != nil {
			t.Fatal("default behaviour changed: operator-link.txt missing")
		}
		if !strings.Contains(link, "#code=") || !strings.HasPrefix(link, "http://127.0.0.1:"+strconv.Itoa(a.Port)) {
			t.Fatalf("link not announced: %q", link)
		}
		cancel()
		<-done
		prof.Close()
	}
}

// #33: a signed-in window whose tab has no token, or an older session's
// token, gets the token of the session its cookie names — only same-origin,
// only with the custom header — and every change still needs that token.
func TestSessionCSRFRecovery(t *testing.T) {
	x := start(t)
	get := func(hdr map[string]string, jar http.CookieJar) (int, string) {
		req, _ := http.NewRequest("GET", x.base+"/api/session/csrf", nil)
		for k, v := range hdr {
			req.Header.Set(k, v)
		}
		c := &http.Client{Jar: jar}
		resp, err := c.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		var m map[string]string
		json.NewDecoder(resp.Body).Decode(&m)
		return resp.StatusCode, m["csrf"]
	}
	ok := map[string]string{"X-TP-Want-CSRF": "1", "Sec-Fetch-Site": "same-origin"}
	if st, tok := get(ok, x.c.Jar); st != 200 || tok != x.csrf {
		t.Fatalf("same-origin recovery: %d %q", st, tok)
	}
	for name, hdr := range map[string]map[string]string{
		"no custom header":  {"Sec-Fetch-Site": "same-origin"},
		"cross-site":        {"X-TP-Want-CSRF": "1", "Sec-Fetch-Site": "cross-site"},
		"same-site":         {"X-TP-Want-CSRF": "1", "Sec-Fetch-Site": "same-site"},
		"no fetch metadata": {"X-TP-Want-CSRF": "1"},
	} {
		if st, tok := get(hdr, x.c.Jar); st != 403 || tok != "" {
			t.Fatalf("%s: got %d %q, want 403 and no token", name, st, tok)
		}
	}
	empty, _ := cookiejar.New(nil)
	if st, tok := get(ok, empty); st != 401 || tok != "" {
		t.Fatalf("no session cookie: %d %q", st, tok)
	}
	// Changes still need the token: none → refused; wrong → refused.
	saved := x.csrf
	x.csrf = ""
	if st, _ := x.post("/api/theme", `{"theme":"dark"}`, nil); st != 403 {
		t.Fatalf("change without token accepted: %d", st)
	}
	x.csrf = "0000000000000000000000000000000000000000000000000000000000000000"
	if st, _ := x.post("/api/theme", `{"theme":"dark"}`, nil); st != 403 {
		t.Fatalf("change with wrong token accepted: %d", st)
	}
	x.csrf = saved
	if st, b := x.post("/api/theme", `{"theme":"dark"}`, nil); st != 200 {
		t.Fatalf("change with recovered token refused: %d %s", st, b)
	}
	// After a second Open replaces the cookie, recovery returns the NEW session's
	// token and the old token no longer works.
	tk, _, _ := x.a.Sess.MintTicket(x.a.Sess.Code())
	if st, b := x.post("/api/session", `{"code":"`+tk+`"}`, nil); st != 200 {
		t.Fatalf("second open: %d %s", st, b)
	}
	st, fresh := get(ok, x.c.Jar)
	if st != 200 || fresh == "" || fresh == saved {
		t.Fatalf("recovery after reopen: %d same=%v", st, fresh == saved)
	}
	x.csrf = saved
	if st, _ := x.post("/api/theme", `{"theme":"light"}`, nil); st != 403 {
		t.Fatalf("older session token accepted after reopen: %d", st)
	}
	x.csrf = fresh
	if st, _ := x.post("/api/theme", `{"theme":"light"}`, nil); st != 200 {
		t.Fatalf("fresh token refused: %d", st)
	}
}

// Pipeline: the requests endpoint returns outcomes for one capability only,
// needs a signed-in window, and never carries what a feature answered.
func TestPipelineRequestsEndpoint(t *testing.T) {
	x := start(t)
	id := "ins_0123456789abcdef0123"
	x.a.P1.mu.Lock()
	x.a.P1.requests = append(x.a.P1.requests,
		ActionResult{RequestID: "req_a", InstallationID: id, Capability: "tass.chat.send", Outcome: "COMPLETED", Observation: map[string]any{"output": "SECRET WORDS"}},
		ActionResult{RequestID: "req_b", InstallationID: id, Capability: "tass.chat.history", Outcome: "COMPLETED"},
		ActionResult{RequestID: "req_c", InstallationID: id, Capability: "tass.chat.send", Outcome: "DENIED_BY_IDENTITY"})
	x.a.P1.mu.Unlock()
	get := func(c *http.Client, q string) (int, string) {
		resp, err := c.Get(x.base + "/api/installations/" + id + "/requests?" + q)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		b := new(strings.Builder)
		buf := make([]byte, 4096)
		for {
			n, err := resp.Body.Read(buf)
			b.Write(buf[:n])
			if err != nil {
				break
			}
		}
		return resp.StatusCode, b.String()
	}
	st, body := get(x.c, "capability=tass.chat.send")
	if st != 200 || !strings.Contains(body, "req_c") || !strings.Contains(body, "req_a") || strings.Contains(body, "req_b") {
		t.Fatalf("wrong requests: %d %s", st, body)
	}
	if strings.Index(body, "req_c") > strings.Index(body, "req_a") {
		t.Fatalf("not newest first: %s", body)
	}
	if strings.Contains(body, "SECRET WORDS") {
		t.Fatal("a feature's answer leaked into the requests list")
	}
	if st, _ := get(x.c, "capability=Bad Cap"); st != 400 {
		t.Fatalf("bad capability accepted: %d", st)
	}
	if st, _ := get(&http.Client{}, "capability=tass.chat.send"); st != 401 {
		t.Fatalf("no session accepted: %d", st)
	}
}
