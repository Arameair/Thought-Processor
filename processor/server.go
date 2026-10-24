package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

//go:embed web/index.html web/app.js web/app.css
var webFS embed.FS

type App struct {
	prof      *Profile
	port      int
	build     BuildInfo
	runID     string
	started   time.Time
	startupMS int64
	log       *Logbook
	sess      *Sessions
	auth      Authority

	mu       sync.Mutex
	cands    Candidates
	candErr  string
	idProbe  *ProbeResult
	engProbe *ProbeResult
	probing  bool
	refusals int // protected-operation requests refused this run

	shutdownOnce sync.Once
	shutdownReq  chan struct{}
}

func NewApp(prof *Profile, port int, b BuildInfo, t0 time.Time) (*App, error) {
	a := &App{prof: prof, port: port, build: b, runID: newID("run_"), started: t0.UTC(),
		log: OpenLogbook(prof.LogPath()), sess: NewSessions(), auth: noProducer{}, shutdownReq: make(chan struct{})}
	if prof.Previous != nil {
		a.log.Add("lifecycle", "previous_unclean", "", fmt.Sprintf("previous run %s (pid %d) ended without a clean shutdown; nothing was started or stopped because of it", prof.Previous.RunID, prof.Previous.PID))
	}
	c, err := prof.LoadCandidates()
	if err != nil {
		a.candErr = err.Error()
		a.log.Add("profile", "candidates_unreadable", "", "saved addresses could not be used; file left unchanged")
	}
	a.cands = c
	if err := prof.WriteCode(a.sess.Code()); err != nil {
		return nil, fmt.Errorf("can't write operator code to profile: %w", err)
	}
	if err := prof.WriteRun(RunRecord{RunID: a.runID, PID: os.Getpid(), Port: port, StartedAt: a.started, Revision: b.Revision}); err != nil {
		return nil, fmt.Errorf("can't write run record to profile: %w", err)
	}
	return a, nil
}

// ---------- request guards ----------

func (a *App) hostOK(r *http.Request) bool {
	p := fmt.Sprint(a.port)
	switch r.Host {
	case "127.0.0.1:" + p, "localhost:" + p, "[::1]:" + p:
		return true
	}
	return false
}

func secureHeaders(w http.ResponseWriter) {
	h := w.Header()
	h.Set("Content-Security-Policy", "default-src 'none'; script-src 'self'; style-src 'self'; img-src 'self'; connect-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'none'")
	h.Set("X-Frame-Options", "DENY")
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Referrer-Policy", "no-referrer")
	h.Set("Cache-Control", "no-store")
	h.Set("Cross-Origin-Resource-Policy", "same-origin")
}

type guard int

const (
	public    guard = iota // GET only, no private data
	private                // GET, requires operator session
	mutate                 // POST, requires session + Origin + JSON + CSRF
	bootstrap              // POST /api/session: Origin + JSON, no session yet
)

func (a *App) wrap(g guard, h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		secureHeaders(w)
		// No CORS headers are ever sent, so browsers block cross-origin reads.
		if !a.hostOK(r) {
			a.log.Add("guard", "bad_host", "", "request refused: untrusted Host header")
			writeJSON(w, http.StatusMisdirectedRequest, errBody("untrusted_host", "Processor only answers to its own loopback address"))
			return
		}
		if g == mutate || g == bootstrap {
			if r.Header.Get("Origin") != "http://"+r.Host {
				a.log.Add("guard", "bad_origin", "", "state-changing request refused: missing or foreign Origin")
				writeJSON(w, http.StatusForbidden, errBody("untrusted_origin", "this action must come from the Processor page itself"))
				return
			}
			if ct := r.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
				writeJSON(w, http.StatusUnsupportedMediaType, errBody("json_required", "Content-Type must be application/json"))
				return
			}
		}
		if g == private || g == mutate {
			rec, ok := a.sess.Lookup(r)
			if !ok {
				writeJSON(w, http.StatusUnauthorized, errBody("no_operator_session", "sign in with this run's operator code first"))
				return
			}
			if g == mutate && (len(r.Header.Get("X-TP-CSRF")) != 64 || r.Header.Get("X-TP-CSRF") != rec.csrf) {
				a.log.Add("guard", "bad_csrf", "", "state-changing request refused: missing or wrong CSRF token")
				writeJSON(w, http.StatusForbidden, errBody("csrf", "missing or invalid anti-forgery token"))
				return
			}
		}
		h(w, r)
	}
}

func errBody(code, msg string) map[string]string {
	return map[string]string{"error": code, "message": msg}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

// readStrict decodes a small JSON body into v, refusing unknown fields.
func readStrict(r *http.Request, v any) error {
	dec := json.NewDecoder(io.LimitReader(r.Body, 2048))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return err
	}
	if dec.More() {
		return fmt.Errorf("trailing data")
	}
	return nil
}

// ---------- routes ----------

func (a *App) Routes() *http.ServeMux {
	m := http.NewServeMux()
	static := func(name, ctype string) http.HandlerFunc {
		b, err := webFS.ReadFile("web/" + name)
		if err != nil {
			panic(err)
		}
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", ctype)
			w.Write(b)
		}
	}
	m.HandleFunc("GET /{$}", a.wrap(public, static("index.html", "text/html; charset=utf-8")))
	m.HandleFunc("GET /app.js", a.wrap(public, static("app.js", "text/javascript; charset=utf-8")))
	m.HandleFunc("GET /app.css", a.wrap(public, static("app.css", "text/css; charset=utf-8")))

	// Public, minimal, non-sensitive: lets a launcher see what answers here.
	m.HandleFunc("GET /api/health", a.wrap(public, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]any{"app": product, "protocol": protocol, "version": appVersion,
			"note": "self-reported; not authentication"})
	}))

	m.HandleFunc("GET /api/session", a.wrap(private, func(w http.ResponseWriter, r *http.Request) {
		rec, _ := a.sess.Lookup(r)
		writeJSON(w, 200, map[string]any{"signed_in": true, "csrf": rec.csrf})
	}))
	m.HandleFunc("POST /api/session", a.wrap(bootstrap, a.handleSignIn))

	m.HandleFunc("GET /api/state", a.wrap(private, func(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, a.State()) }))
	m.HandleFunc("GET /api/context", a.wrap(private, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		io.WriteString(w, a.CopyContext())
	}))

	m.HandleFunc("POST /api/candidates", a.wrap(mutate, a.handleCandidates))
	m.HandleFunc("POST /api/probe", a.wrap(mutate, func(w http.ResponseWriter, r *http.Request) {
		var in struct{}
		if err := readStrict(r, &in); err != nil {
			writeJSON(w, 400, errBody("bad_request", "this action takes no parameters"))
			return
		}
		a.Probe(r.Context())
		writeJSON(w, 200, a.State())
	}))
	m.HandleFunc("POST /api/engine/{action}", a.wrap(mutate, a.handleProtected))
	m.HandleFunc("POST /api/shutdown", a.wrap(mutate, func(w http.ResponseWriter, r *http.Request) {
		var in struct{}
		if err := readStrict(r, &in); err != nil {
			writeJSON(w, 400, errBody("bad_request", "this action takes no parameters"))
			return
		}
		a.log.Add("lifecycle", "stop_requested", "", "operator asked Processor to stop itself (only Processor)")
		writeJSON(w, 202, map[string]any{"status": "STOPPING", "affects": "this Processor process only"})
		a.shutdownOnce.Do(func() { go func() { time.Sleep(100 * time.Millisecond); close(a.shutdownReq) }() })
	}))
	return m
}

func (a *App) handleSignIn(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Code string `json:"code"`
	}
	if err := readStrict(r, &in); err != nil || len(in.Code) > 128 {
		writeJSON(w, 400, errBody("bad_request", "expected {\"code\": \"...\"}"))
		return
	}
	id, csrf, ok, locked := a.sess.Exchange(strings.TrimSpace(in.Code))
	if locked {
		writeJSON(w, 429, errBody("paused", "too many wrong codes; wait a minute"))
		return
	}
	if !ok {
		a.log.Add("session", "code_rejected", "", "operator sign-in refused: wrong code")
		writeJSON(w, 401, errBody("wrong_code", "that is not this run's operator code"))
		return
	}
	http.SetCookie(w, &http.Cookie{Name: cookieName, Value: id, Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode})
	a.log.Add("session", "signed_in", "", "local operator session created")
	writeJSON(w, 200, map[string]any{"signed_in": true, "csrf": csrf})
}

func (a *App) handleCandidates(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Identity *string `json:"identity"`
		Engine   *string `json:"engine"`
	}
	if err := readStrict(r, &in); err != nil {
		writeJSON(w, 400, errBody("bad_request", "expected {\"identity\": \"127.0.0.1:PORT\"} and/or {\"engine\": ...}; empty string clears"))
		return
	}
	a.mu.Lock()
	next := a.cands
	a.mu.Unlock()
	if in.Identity != nil {
		v, err := parseCandidate(*in.Identity, a.port)
		if err != nil {
			a.log.Add("config", "address_refused", "", "Identity address refused: "+err.Error())
			writeJSON(w, 400, errBody("address_refused", "Identity address: "+err.Error()))
			return
		}
		next.Identity = v
	}
	if in.Engine != nil {
		v, err := parseCandidate(*in.Engine, a.port)
		if err != nil {
			a.log.Add("config", "address_refused", "", "Engine address refused: "+err.Error())
			writeJSON(w, 400, errBody("address_refused", "Engine address: "+err.Error()))
			return
		}
		next.Engine = v
	}
	if err := a.prof.SaveCandidates(next); err != nil {
		writeJSON(w, 500, errBody("save_failed", "could not save to the profile; nothing changed"))
		return
	}
	a.mu.Lock()
	if next.Identity != a.cands.Identity {
		a.idProbe = nil
	}
	if next.Engine != a.cands.Engine {
		a.engProbe = nil
	}
	a.cands, a.candErr = next, ""
	a.mu.Unlock()
	a.log.Add("config", "addresses_saved", "", fmt.Sprintf("candidate addresses saved (identity=%q engine=%q); saving an address grants nothing", next.Identity, next.Engine))
	a.Probe(r.Context())
	writeJSON(w, 200, a.State())
}

// handleProtected is Processor's control boundary for Engine operations.
// It accepts NO parameters (no paths, arguments, endpoints or targets), asks
// the Authority, and — in this build — always refuses before any effect.
func (a *App) handleProtected(w http.ResponseWriter, r *http.Request) {
	act := ProtectedAction(r.PathValue("action"))
	opID := newID("op_")
	if _, known := protectedActions[act]; !known {
		a.log.Add("control", "unknown_action", opID, "refused: unknown protected action")
		writeJSON(w, 404, map[string]any{"op_id": opID, "status": "REFUSED", "error": "unknown_action", "effects": "none"})
		return
	}
	var in struct{}
	if err := readStrict(r, &in); err != nil {
		a.mu.Lock()
		a.refusals++
		a.mu.Unlock()
		a.log.Add("control", "parameters_refused", opID, string(act)+": refused — this boundary accepts no parameters")
		writeJSON(w, 400, map[string]any{"op_id": opID, "action": act, "status": "REFUSED", "error": "parameters_not_accepted",
			"message": "Protected operations accept no paths, arguments, endpoints or targets.", "effects": "none"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	err := a.auth.Decide(ctx, act)
	a.mu.Lock()
	a.refusals++
	a.mu.Unlock()
	code, status := "blocked_no_identity_approval_service", http.StatusNotImplemented
	if err == ErrIngestSeparate {
		code = "blocked_no_ingest_grant_contract"
	}
	msg := "unknown"
	if err != nil {
		msg = err.Error()
	}
	a.log.Add("control", code, opID, string(act)+": BLOCKED before any effect — "+msg)
	writeJSON(w, status, map[string]any{
		"op_id": opID, "action": act, "status": "BLOCKED", "error": code, "message": msg,
		"enforcement_note": "Even with an Identity approval, the runner/Engine that executes this must validate it. This Processor build contains no process runner.",
		"effects":          "none — no process started, attached, stopped or contacted",
		"at":               time.Now().UTC(),
	})
}

// ---------- probes ----------

func (a *App) Probe(ctx context.Context) {
	a.mu.Lock()
	if a.probing {
		a.mu.Unlock()
		return
	}
	a.probing = true
	c := a.cands
	a.mu.Unlock()
	defer func() { a.mu.Lock(); a.probing = false; a.mu.Unlock() }()

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	var wg sync.WaitGroup
	var idr, enr *ProbeResult
	if c.Identity != "" {
		wg.Add(1)
		go func() { defer wg.Done(); r := ProbeIdentity(ctx, c.Identity); idr = &r }()
	}
	if c.Engine != "" {
		wg.Add(1)
		go func() { defer wg.Done(); r := ProbeEngine(ctx, c.Engine); enr = &r }()
	}
	wg.Wait()
	a.mu.Lock()
	defer a.mu.Unlock()
	// Only store results for the address still configured.
	if idr != nil && idr.Target == a.cands.Identity {
		if a.idProbe == nil || a.idProbe.Port != idr.Port || a.idProbe.Metadata != idr.Metadata {
			a.log.Add("probe", "identity_"+strings.ToLower(idr.Port+"_"+idr.Metadata), "", "Identity candidate "+idr.Target+": port "+idr.Port+", metadata "+idr.Metadata)
		}
		a.idProbe = idr
	}
	if enr != nil && enr.Target == a.cands.Engine {
		if a.engProbe == nil || a.engProbe.Port != enr.Port {
			a.log.Add("probe", "engine_"+strings.ToLower(enr.Port), "", "Engine candidate "+enr.Target+": port "+enr.Port+" (contents not read)")
		}
		a.engProbe = enr
	}
}

// ---------- state ----------

type Capability struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Status string `json:"status"` // WORKING | UNAVAILABLE | BLOCKED
	Why    string `json:"why"`
}

type Operation struct {
	Action  ProtectedAction `json:"action"`
	Label   string          `json:"label"`
	Enabled bool            `json:"enabled"`
	Status  string          `json:"status"`
	Reason  string          `json:"reason"`
}

type DepView struct {
	Name       string       `json:"name"`
	Configured string       `json:"configured"`
	State      string       `json:"state"` // NOT_CONFIGURED | UNREACHABLE | PORT_OPEN_UNIDENTIFIED | CLAIMS_IDENTITY_UNVERIFIED | ANSWERED_NOT_IDENTITY | NO_PUBLIC_METADATA | NOT_CHECKED | STALE
	Label      string       `json:"label"`
	Freshness  string       `json:"freshness"`
	Probe      *ProbeResult `json:"probe,omitempty"`
	Approval   string       `json:"approval"`
	Explain    string       `json:"explain"`
}

type NextAction struct {
	Code   string `json:"code"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
	Focus  string `json:"focus"` // UI element to focus
}

func (a *App) State() map[string]any {
	now := time.Now()
	a.mu.Lock()
	c, candErr := a.cands, a.candErr
	var idp, enp *ProbeResult
	if a.idProbe != nil {
		x := *a.idProbe
		idp = &x
	}
	if a.engProbe != nil {
		x := *a.engProbe
		enp = &x
	}
	refusals := a.refusals
	a.mu.Unlock()

	avail, availWhy := a.auth.Available()
	id := identityView(c.Identity, idp, now, avail, availWhy)
	en := engineView(c.Engine, enp, now)

	ops := []Operation{}
	for _, act := range []ProtectedAction{ActStart, ActAttach, ActStop, ActObserve} {
		reason := "Blocked: Identity has no start/attach/stop approval service, and no runner validates one. Nothing can be started or attached."
		switch act {
		case ActStop:
			reason = "Blocked: this session started no Engine (it cannot start one), and no stop approval/control policy exists."
		case ActObserve:
			reason = "Blocked: sending observations is a separate Identity grant enforced by the Engine. No authenticated source or grant is set up here."
		}
		ops = append(ops, Operation{Action: act, Label: protectedActions[act], Enabled: false, Status: "BLOCKED", Reason: reason})
	}

	var prev any
	if a.prof.Previous != nil {
		prev = map[string]any{"run_id": a.prof.Previous.RunID, "pid": a.prof.Previous.PID, "started_at": a.prof.Previous.StartedAt,
			"meaning": "that run did not shut down cleanly; Processor started fresh and took no action because of it"}
	}
	return map[string]any{
		"product": product, "protocol": protocol,
		"self": map[string]any{
			"state": "RUNNING", "run_id": a.runID, "pid": os.Getpid(), "port": a.port,
			"address": fmt.Sprintf("http://127.0.0.1:%d", a.port), "started_at": a.started,
			"uptime_s": int64(now.Sub(a.started).Seconds()), "startup_ms": a.startupMS,
			"build": a.build, "profile": displayPath(a.prof.Dir), "server_time": now.UTC(),
			"previous_unclean_run": prev, "last_clean_stop": a.prof.LastStop,
			"log_error": a.log.Err(), "profile_problem": candErr,
			"owned_processes": []string{}, "owned_processes_note": "This build contains no process runner, so it owns no processes.",
			"refused_operations_this_run": refusals,
		},
		"identity":     id,
		"engine":       en,
		"operations":   ops,
		"capabilities": a.capabilities(),
		"next_action":  nextAction(c, idp, enp, now, candErr),
		"log":          a.log.Recent(40),
	}
}

func identityView(addr string, p *ProbeResult, now time.Time, avail bool, availWhy string) DepView {
	v := DepView{Name: "Identity", Configured: addr, Probe: p, Freshness: freshness(p, now),
		Approval: "NOT AVAILABLE — " + availWhy,
		Explain:  "Identity decides who is trusted and what is permitted. Processor asks; it never decides."}
	_ = avail
	switch {
	case addr == "":
		v.State, v.Label = "NOT_CONFIGURED", "No Identity address set"
	case p == nil:
		v.State, v.Label = "NOT_CHECKED", "Not checked yet"
	case v.Freshness == "STALE":
		v.State, v.Label = "STALE", "Last check is too old to trust — re-check"
	case p.Port != "OPEN":
		v.State, v.Label = "UNREACHABLE", "Not reachable ("+strings.ToLower(p.Port)+")"
	case p.Metadata == "RESPONDED" && p.ClaimMatches:
		v.State, v.Label = "CLAIMS_IDENTITY_UNVERIFIED", "Answers and claims to be Identity (unverified)"
	case p.Metadata == "RESPONDED":
		v.State, v.Label = "ANSWERED_NOT_IDENTITY", "Something answered, but it is not Identity"
	default:
		v.State, v.Label = "PORT_OPEN_UNIDENTIFIED", "Port open; no recognizable Identity metadata"
	}
	return v
}

func engineView(addr string, p *ProbeResult, now time.Time) DepView {
	v := DepView{Name: "Thought Engine", Configured: addr, Probe: p, Freshness: freshness(p, now),
		Approval: "NOT ATTACHED — attaching requires an Identity approval that cannot be obtained yet",
		Explain:  "Thought Engine does the actual thinking. Processor may only look at it after an approved attachment."}
	switch {
	case addr == "":
		v.State, v.Label = "NOT_CONFIGURED", "No Engine address set (optional for now)"
	case p == nil:
		v.State, v.Label = "NOT_CHECKED", "Not checked yet"
	case v.Freshness == "STALE":
		v.State, v.Label = "STALE", "Last check is too old to trust — re-check"
	case p.Port != "OPEN":
		v.State, v.Label = "UNREACHABLE", "Not reachable ("+strings.ToLower(p.Port)+")"
	default:
		v.State, v.Label = "PORT_OPEN_UNIDENTIFIED", "Port open — not identified, not attached"
	}
	return v
}

func nextAction(c Candidates, idp, enp *ProbeResult, now time.Time, candErr string) NextAction {
	switch {
	case candErr != "":
		return NextAction{"fix_profile", "Fix the saved addresses", "The profile's candidates.json could not be used and was left unchanged. Re-enter the Identity address below to replace it.", "identity-addr"}
	case c.Identity == "":
		return NextAction{"set_identity", "Tell Processor where Identity runs", "Identity is the authority that would approve starting or attaching a Thought Engine. Enter the address of your local Identity app (for example 127.0.0.1:7777). Processor only looks; it never starts Identity.", "identity-addr"}
	case idp == nil || freshness(idp, now) == "STALE":
		return NextAction{"recheck", "Check Identity now", "The last check is missing or too old to trust.", "recheck"}
	case idp.Port != "OPEN":
		return NextAction{"start_identity", "Start Identity yourself, then re-check", "Nothing answers at " + c.Identity + ". Processor will not start Identity for you. Start it the way you normally do, or correct the address.", "recheck"}
	case !(idp.Metadata == "RESPONDED" && idp.ClaimMatches):
		return NextAction{"wrong_identity", "Check the Identity address", "Something is listening at " + c.Identity + " but it does not identify itself as Identity. It may be a different program or an Identity build without public health metadata.", "identity-addr"}
	default:
		return NextAction{"blocked_contract", "Engine control is blocked — waiting on Identity", "Identity answers (unverified), but it has no service that can approve starting, attaching or stopping a Thought Engine. Read the dependency request to see exactly what Identity and the Engine must add. Until then Processor can only diagnose.", "deps"}
	}
}

func (a *App) capabilities() []Capability {
	_, why := a.auth.Available()
	return []Capability{
		{"self_status", "Report its own process, port, build and profile", "WORKING", "values read from the running process and Go build metadata"},
		{"single_instance", "Refuse a second Processor on the same profile", "WORKING", "kernel file lock on runtime/processor.lock (Linux tested)"},
		{"operator_session", "Local operator sign-in (per-run code)", "WORKING", "not an Identity approval; gates Processor's own pages only"},
		{"request_guards", "Host, Origin, CSRF and JSON checks on actions", "WORKING", "no CORS; GET never changes anything"},
		{"identity_probe", "Look at an explicitly entered Identity address", "WORKING", "TCP check + public GET /api/health; result labeled unverified"},
		{"engine_probe", "Look at an explicitly entered Engine address", "WORKING", "TCP check only; Engine status is private until attached"},
		{"copy_context", "Copy a sanitized diagnostic summary", "WORKING", "allowlisted fields; no codes, cookies, bodies or remote claims"},
		{"identity_authentication", "Prove the Identity candidate is genuine", "UNAVAILABLE", "Identity exposes no peer-authentication mechanism"},
		{"lifecycle_approval", "Ask Identity to approve start / attach / stop", "BLOCKED", why},
		{"runner", "Start or stop an approved Engine executable", "BLOCKED", "no shared bounded runner exists; this build deliberately contains none"},
		{"engine_attach", "Read Engine status/queue/worker after approval", "BLOCKED", "requires lifecycle approval + Engine-side validation (neither exists)"},
		{"engine_original_ui", "Open the Engine's own page", "BLOCKED", "only after an approved attachment"},
		{"observation_ingest", "Send a test observation", "BLOCKED", "separate Identity grant contract enforced by the Engine; no authenticated source here"},
		{"github_licensing", "GitHub bootstrap, licensing, updates, publisher entitlements", "UNAVAILABLE", "undecided by the owner; intentionally not implemented"},
	}
}

func displayPath(p string) string {
	if home, err := os.UserHomeDir(); err == nil && home != "" && strings.HasPrefix(p, home+string(filepath.Separator)) {
		return "~" + p[len(home):]
	}
	return p
}

// CopyContext is an allowlisted, bounded summary for pasting into an issue
// or another AI. It deliberately omits everything not listed here.
func (a *App) CopyContext() string {
	s := a.State()
	self := s["self"].(map[string]any)
	b := self["build"].(BuildInfo)
	var w strings.Builder
	fmt.Fprintf(&w, "Thought Processor diagnostic context (sanitized)\n")
	fmt.Fprintf(&w, "generated: %s\n", time.Now().UTC().Format(time.RFC3339))
	fmt.Fprintf(&w, "product: %s %s  protocol: %s\n", product, appVersion, protocol)
	fmt.Fprintf(&w, "build: revision=%s modified=%s vcs_time=%s go=%s %s/%s\n", b.Revision, b.Modified, b.BuiltAt, b.GoVersion, b.OS, b.Arch)
	fmt.Fprintf(&w, "run: %s pid=%v port=%v started=%v startup_ms=%v\n", self["run_id"], self["pid"], self["port"], a.started.Format(time.RFC3339), self["startup_ms"])
	fmt.Fprintf(&w, "profile: %s\n", self["profile"])
	if self["previous_unclean_run"] != nil {
		fmt.Fprintf(&w, "previous run ended uncleanly: yes\n")
	}
	for _, key := range []string{"identity", "engine"} {
		d := s[key].(DepView)
		fmt.Fprintf(&w, "%s: address=%q state=%s freshness=%s\n", strings.ToLower(d.Name), d.Configured, d.State, d.Freshness)
		if d.Probe != nil {
			fmt.Fprintf(&w, "  last check %s  %d ms  port=%s metadata=%s http=%d claims_identity=%v\n",
				d.Probe.CheckedAt.Format(time.RFC3339), d.Probe.ElapsedMS, d.Probe.Port, d.Probe.Metadata, d.Probe.HTTPStatus, d.Probe.ClaimMatches)
		}
	}
	for _, op := range s["operations"].([]Operation) {
		fmt.Fprintf(&w, "operation %s: %s\n", op.Action, op.Status)
	}
	na := s["next_action"].(NextAction)
	fmt.Fprintf(&w, "next action: %s\n", na.Code)
	fmt.Fprintf(&w, "recent events:\n")
	for _, e := range a.log.Recent(20) {
		fmt.Fprintf(&w, "  %s %s/%s %s\n", e.At.Format("15:04:05"), e.Kind, e.Code, e.OpID)
	}
	fmt.Fprintf(&w, "omitted by design: operator code, session cookies, CSRF tokens, request bodies, environment variables, text returned by other programs (claims), log detail text\n")
	out := w.String()
	if len(out) > 8192 {
		out = out[:8192] + "\n[truncated at 8 KiB]\n"
	}
	return out
}
