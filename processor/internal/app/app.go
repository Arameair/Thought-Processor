// Package app is Thought Processor's P0 application: the person's interface
// for connecting to Identity, choosing an identity, discovering features and
// installing them with the reusable installer.
package app

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Arameair/Thought-Processor/processor/internal/idclient"
	"github.com/Arameair/Thought-Processor/processor/internal/install"
	"github.com/Arameair/Thought-Processor/processor/internal/store"
	"github.com/Arameair/Thought-Processor/processor/internal/supervisor"
)

const (
	Product  = "thought-processor"
	Protocol = "thought-processor/p0"
)

//go:embed web/*
var webFS embed.FS

type Build struct {
	Version  string `json:"version"`
	Branch   string `json:"branch"`
	Revision string `json:"revision"`
	Modified string `json:"modified"`
}

type App struct {
	Prof    *store.Profile
	Log     *Log
	Conn    *Conn
	Inst    *install.Installer
	Sess    *Sessions
	P1      *p1
	Sup     *supervisor.Supervisor
	Build   Build
	RunID   string
	Started time.Time
	Port    int

	stop     chan struct{}
	stopOnce sync.Once
	stopWhy  string
	exitNote string
}

func New(prof *store.Profile, b Build) *App {
	run := "run_" + store.RandHex(6)
	lg := OpenLog(prof.Dir+string(os.PathSeparator)+"logs", run)
	a := &App{Prof: prof, Log: lg, Sess: NewSessions(), Build: b, RunID: run, Started: time.Now(), stop: make(chan struct{})}
	a.Inst = install.New(prof.Dir, func(level, event, feature, detail string) { lg.Add(level, "install", event, feature+": "+detail) })
	a.Conn = NewConn(prof, lg)
	a.initP1()
	return a
}

// Run serves on 127.0.0.1 until ctx ends or the person exits.
func (a *App) Run(ctx context.Context, ln net.Listener, ready func(link string)) error {
	if ln == nil {
		var err error
		ln, err = net.Listen("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(a.Port)))
		if err != nil {
			return fmt.Errorf("can't use 127.0.0.1:%d — is another Thought Processor running? (%v)", a.Port, err)
		}
	}
	a.Port = ln.Addr().(*net.TCPAddr).Port
	link := fmt.Sprintf("http://127.0.0.1:%d/#code=%s", a.Port, a.Sess.Code())
	store.WriteFile(a.Prof.CodePath(), []byte(link+"\n"))
	srv := &http.Server{Handler: a.Handler(), ReadHeaderTimeout: 5 * time.Second}
	cctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go a.Conn.Run(cctx)
	go a.reconcileLoop(cctx)
	a.Log.Info("lifecycle", "processor.started", fmt.Sprintf("pid %d port %d profile %s version %s commit %s", os.Getpid(), a.Port, store.DisplayPath(a.Prof.Dir), a.Build.Version, short(a.Build.Revision)))
	errc := make(chan error, 1)
	go func() { errc <- srv.Serve(ln) }()
	if ready != nil {
		ready(link)
	}
	why := "signal from the console (Ctrl+C / window closed)"
	select {
	case <-ctx.Done():
	case <-a.stop:
		why = a.stopWhy
	case err := <-errc:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	}
	a.Log.Info("lifecycle", "processor.stopping", why)
	if n := a.Sup.StopAll("Thought Processor is exiting"); n > 0 {
		a.Log.Info("lifecycle", "features.stopped", fmt.Sprintf("stopped %d feature process(es) this Processor started; their installation identities and data are kept", n))
	}
	sctx, scancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer scancel()
	srv.Shutdown(sctx)
	a.Log.Close()
	return nil
}

func (a *App) requestStop(why string) { a.stopOnce.Do(func() { a.stopWhy = why; close(a.stop) }) }

// ExitNote is what main prints after the app stops (e.g. Identity left running).
func (a *App) ExitNote() string { return a.exitNote }

func short(s string) string {
	if len(s) > 12 {
		return s[:12]
	}
	return s
}

// ---- HTTP ----

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

type level int

const (
	public level = iota
	bootstrap
	private
	mutate
)

func (a *App) wrap(lv level, h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 16 KB for every request, except a feature request that may carry one
		// image attachment (validated again in the action handler and in the feature).
		limit := int64(16 << 10)
		if r.Method == "POST" && strings.HasPrefix(r.URL.Path, "/api/installations/") && strings.HasSuffix(r.URL.Path, "/action") {
			limit = 8 << 20
		}
		r.Body = http.MaxBytesReader(w, r.Body, limit)
		if lv == public {
			h(w, r)
			return
		}
		if lv == bootstrap || lv == mutate {
			if r.Header.Get("Origin") != "http://"+r.Host || !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
				writeJSON(w, 403, map[string]string{"error": "request did not come from this Processor window"})
				return
			}
		}
		if lv == bootstrap {
			h(w, r)
			return
		}
		csrf, ok := a.Sess.Lookup(r)
		if !ok {
			writeJSON(w, 401, map[string]string{"error": "This window isn't signed in. Open Processor with the link printed in its console window."})
			return
		}
		if lv == mutate && r.Header.Get("X-TP-CSRF") != csrf {
			writeJSON(w, 403, map[string]string{"error": "missing or wrong anti-forgery token"})
			return
		}
		h(w, r)
	}
}

func (a *App) guard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Cache-Control", "no-store")
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Content-Security-Policy", "default-src 'none'; script-src 'self'; style-src 'self'; connect-src 'self'; img-src 'self' data:; frame-ancestors 'none'; base-uri 'none'; form-action 'none'")
		p := strconv.Itoa(a.Port)
		if r.Host != "127.0.0.1:"+p && r.Host != "localhost:"+p {
			http.Error(w, "wrong host", http.StatusMisdirectedRequest)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func decode(r *http.Request, v any) error { return json.NewDecoder(r.Body).Decode(v) }

func (a *App) fail(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

// failIdentity turns an Identity problem into plain words.
func (a *App) failIdentity(w http.ResponseWriter, err error) {
	var e *idclient.Error
	if errors.As(err, &e) {
		switch e.Kind {
		case idclient.Unavailable, idclient.NotPaired:
			a.fail(w, 503, "Identity can't be reached right now, so this can't be done. Processor will reconnect by itself.")
			return
		case idclient.TokenRefused, idclient.WrongService, idclient.Incompatible:
			a.fail(w, 409, e.Msg)
			return
		case idclient.Refused:
			// A signed refusal from the right Identity: keep its status.
			if e.Status >= 400 && e.Status < 600 {
				a.fail(w, e.Status, e.Msg)
				return
			}
		}
		a.fail(w, 400, e.Msg)
		return
	}
	a.fail(w, 500, err.Error())
}

func (a *App) Handler() http.Handler {
	m := http.NewServeMux()
	// Feature routes accept only well-formed feature ids.
	feat := func(h http.HandlerFunc) http.HandlerFunc {
		return a.wrap(mutate, func(w http.ResponseWriter, r *http.Request) {
			if !install.FeatureRE.MatchString(r.PathValue("id")) {
				a.fail(w, 400, "unknown feature")
				return
			}
			h(w, r)
		})
	}
	static := func(name, ctype string) http.HandlerFunc {
		b, err := webFS.ReadFile("web/" + name)
		return func(w http.ResponseWriter, r *http.Request) {
			if err != nil {
				http.Error(w, "missing", 500)
				return
			}
			w.Header().Set("Content-Type", ctype)
			w.Write(b)
		}
	}
	m.HandleFunc("GET /{$}", a.wrap(public, static("index.html", "text/html; charset=utf-8")))
	m.HandleFunc("GET /app.js", a.wrap(public, static("app.js", "text/javascript; charset=utf-8")))
	m.HandleFunc("GET /app.css", a.wrap(public, static("app.css", "text/css; charset=utf-8")))
	m.HandleFunc("GET /api/health", a.wrap(public, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]string{"app": Product, "version": a.Build.Version, "protocol": Protocol, "note": "self-reported; not authentication"})
	}))
	m.HandleFunc("POST /api/session", a.wrap(bootstrap, func(w http.ResponseWriter, r *http.Request) {
		var b struct{ Code string }
		decode(r, &b)
		id, csrf, ok, locked := a.Sess.Exchange(b.Code)
		if locked {
			a.fail(w, 429, "too many wrong links; wait a minute")
			return
		}
		if !ok {
			a.fail(w, 401, "That link isn't valid for this run of Processor (it changes every start).")
			return
		}
		http.SetCookie(w, &http.Cookie{Name: cookieName, Value: id, Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode})
		writeJSON(w, 200, map[string]string{"csrf": csrf})
	}))
	m.HandleFunc("GET /api/state", a.wrap(private, func(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, a.State()) }))

	// Connection
	m.HandleFunc("POST /api/connect/choose", a.wrap(mutate, func(w http.ResponseWriter, r *http.Request) {
		var b struct{ URL string }
		decode(r, &b)
		if err := a.Conn.Choose(b.URL); err != nil {
			a.fail(w, 400, err.Error())
			return
		}
		writeJSON(w, 200, a.State())
	}))
	m.HandleFunc("POST /api/connect/request", a.wrap(mutate, func(w http.ResponseWriter, r *http.Request) {
		var b struct{ URL string }
		decode(r, &b)
		if err := a.Conn.RequestCode(r.Context(), b.URL); err != nil {
			a.failIdentity(w, err)
			return
		}
		writeJSON(w, 200, a.State())
	}))
	m.HandleFunc("POST /api/connect/code", a.wrap(mutate, func(w http.ResponseWriter, r *http.Request) {
		var b struct{ Code string }
		decode(r, &b)
		if err := a.Conn.CompleteCode(r.Context(), b.Code); err != nil {
			a.failIdentity(w, err)
			return
		}
		a.Conn.Tick(r.Context())
		writeJSON(w, 200, a.State())
	}))
	m.HandleFunc("POST /api/connect/disconnect", a.wrap(mutate, func(w http.ResponseWriter, r *http.Request) {
		a.Conn.Disconnect(r.Context())
		writeJSON(w, 200, a.State())
	}))
	m.HandleFunc("POST /api/connect/retry", a.wrap(mutate, func(w http.ResponseWriter, r *http.Request) {
		a.Conn.Tick(r.Context())
		writeJSON(w, 200, a.State())
	}))

	// Identities (Identity is the authority; Processor only asks)
	m.HandleFunc("POST /api/identities", a.wrap(mutate, func(w http.ResponseWriter, r *http.Request) {
		var b struct{ Name, Kind string }
		decode(r, &b)
		cl, err := a.Conn.Require(r.Context())
		if err != nil {
			a.failIdentity(w, err)
			return
		}
		e, err := cl.CreateIdentity(r.Context(), b.Name, b.Kind)
		if err != nil {
			a.failIdentity(w, err)
			return
		}
		a.Prof.UpdateSettings(func(s *store.Settings) { s.SelectedIdentity = e.ID })
		a.Log.Info("identity", "identity.created", "Identity created "+e.ID+" ("+e.Kind+") — it starts with no access")
		a.Conn.refreshData(r.Context())
		writeJSON(w, 200, a.State())
	}))
	m.HandleFunc("POST /api/identities/select", a.wrap(mutate, func(w http.ResponseWriter, r *http.Request) {
		var b struct{ ID string }
		decode(r, &b)
		a.Prof.UpdateSettings(func(s *store.Settings) { s.SelectedIdentity = b.ID })
		a.Conn.refreshData(r.Context()) // drops it again if Identity doesn't have it
		writeJSON(w, 200, a.State())
	}))

	// GitHub sign-in (runs in Identity; the person approves on github.com)
	m.HandleFunc("POST /api/github/connect", a.wrap(mutate, func(w http.ResponseWriter, r *http.Request) {
		cl, err := a.Conn.Require(r.Context())
		if err != nil {
			a.failIdentity(w, err)
			return
		}
		if _, err := cl.GitHubConnect(r.Context()); err != nil {
			a.failIdentity(w, err)
			return
		}
		a.Log.Info("github", "github.connect", "asked Identity to start GitHub sign-in; the person approves it on GitHub")
		a.Conn.Tick(r.Context())
		writeJSON(w, 200, a.State())
	}))
	m.HandleFunc("POST /api/github/cancel", a.wrap(mutate, func(w http.ResponseWriter, r *http.Request) {
		if cl, err := a.Conn.Require(r.Context()); err == nil {
			cl.GitHubCancel(r.Context())
		}
		a.Conn.Tick(r.Context())
		writeJSON(w, 200, a.State())
	}))

	// Features: plan → approve → install (reusable; nothing feature-specific)
	m.HandleFunc("POST /api/features/{id}/plan", feat(func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		cl, err := a.Conn.Require(r.Context())
		if err != nil {
			a.failIdentity(w, err)
			return
		}
		p, err := a.Inst.Prepare(r.Context(), cl, id)
		if err != nil {
			writeJSON(w, 200, a.State()) // the problem is in the feature's state
			return
		}
		if p == nil {
			a.Inst.Verify(id)
		}
		writeJSON(w, 200, a.State())
	}))
	m.HandleFunc("POST /api/features/{id}/approve", feat(func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		var b struct {
			PlanID string `json:"plan_id"`
		}
		decode(r, &b)
		cl, err := a.Conn.Require(r.Context())
		if err != nil {
			a.failIdentity(w, err)
			return
		}
		go a.Inst.Approve(context.Background(), cl, id, b.PlanID)
		time.Sleep(150 * time.Millisecond)
		writeJSON(w, 200, a.State())
	}))
	m.HandleFunc("POST /api/features/{id}/cancel", feat(func(w http.ResponseWriter, r *http.Request) {
		a.Inst.CancelPlan(r.PathValue("id"))
		writeJSON(w, 200, a.State())
	}))
	m.HandleFunc("POST /api/features/{id}/verify", feat(func(w http.ResponseWriter, r *http.Request) {
		a.Inst.Verify(r.PathValue("id"))
		writeJSON(w, 200, a.State())
	}))
	m.HandleFunc("POST /api/features/{id}/remove", feat(func(w http.ResponseWriter, r *http.Request) {
		if err := a.Inst.Remove(r.PathValue("id")); err != nil {
			a.fail(w, 409, err.Error())
			return
		}
		writeJSON(w, 200, a.State())
	}))

	a.p1Routes(m)
	m.HandleFunc("POST /api/devmode", a.wrap(mutate, func(w http.ResponseWriter, r *http.Request) {
		var b struct{ On bool }
		decode(r, &b)
		a.Prof.UpdateSettings(func(s *store.Settings) { s.DeveloperMode = b.On })
		writeJSON(w, 200, a.State())
	}))

	// Exit (C04): the person chose how to leave.
	m.HandleFunc("POST /api/exit", a.wrap(mutate, func(w http.ResponseWriter, r *http.Request) {
		var b struct {
			Choice       string `json:"choice"` // "shutdown" | "keep"
			StopIdentity bool   `json:"stop_identity"`
		}
		decode(r, &b)
		res := a.exit(r.Context(), b.Choice, b.StopIdentity)
		writeJSON(w, 200, res)
		go func() { time.Sleep(300 * time.Millisecond); a.requestStop("the person chose Exit (" + b.Choice + ")") }()
	}))
	return a.guard(m)
}

// exit stops what the person chose. Processor started no other programs in
// P0, so "Shut down managed workspace" stops Processor itself; Identity is
// stopped only when it is MANAGED by this Processor, or when the person
// explicitly ticked "also stop Identity" for an EXTERNAL one.
func (a *App) exit(ctx context.Context, choice string, stopIdentity bool) map[string]any {
	n := a.Sup.StopAll("the person chose Exit")
	res := map[string]any{"processor": "stopping", "managed_processes": n, "features": fmt.Sprintf("%d feature process(es) stopped (installation identities and data kept)", n)}
	a.Conn.mu.Lock()
	st := a.Conn.status
	cl := a.Conn.client
	a.Conn.mu.Unlock()
	identityNote := "left running"
	if choice == "shutdown" && cl != nil && st != nil {
		managed := st.Lifecycle.Ownership == "MANAGED" && strings.Contains(st.Lifecycle.StartedBy, a.Prof.Self.ProcessorID)
		if managed || stopIdentity {
			if _, err := a.Conn.Require(ctx); err == nil && cl.Shutdown(ctx, "the person chose Shut down when exiting Thought Processor") == nil {
				identityNote = "stopping (pid " + strconv.Itoa(st.Lifecycle.PID) + ")"
				deadline := time.Now().Add(10 * time.Second)
				for time.Now().Before(deadline) {
					if _, err := idclient.ProbeHealth(ctx, cl.Pairing().BaseURL); err != nil {
						identityNote = "stopped (verified: no longer answering)"
						break
					}
					time.Sleep(300 * time.Millisecond)
				}
			} else {
				identityNote = "could not be asked to stop (not reachable); it may still be running"
			}
		}
	}
	res["identity"] = identityNote
	a.exitNote = "Identity: " + identityNote
	a.Log.Info("lifecycle", "exit.chosen", fmt.Sprintf("choice=%s stop_identity=%v identity=%s", choice, stopIdentity, identityNote))
	return res
}
