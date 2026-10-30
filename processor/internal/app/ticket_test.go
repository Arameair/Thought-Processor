package app

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/cookiejar"
	"os"
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
