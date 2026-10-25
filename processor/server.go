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
	link      *EngineLink

	mu       sync.Mutex
	cands    Candidates
	candErr  string
	idProbe  *ProbeResult
	engProbe *ProbeResult
	probing  bool
	lastLink *LinkView

	metrics *Metrics
	flow    *Flow

	shutdownOnce sync.Once
	shutdownReq  chan struct{}
}

func NewApp(prof *Profile, port int, b BuildInfo, t0 time.Time) (*App, error) {
	a := &App{prof: prof, port: port, build: b, runID: newID("run_"), started: t0.UTC(),
		log: OpenLogbook(prof.LogPath()), sess: NewSessions(), auth: noProducer{}, shutdownReq: make(chan struct{}), metrics: NewMetrics(), flow: &Flow{}, link: OpenEngineLink(prof.Dir)}
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
			a.metrics.Guard("untrusted_host")
			writeJSON(w, http.StatusMisdirectedRequest, errBody("untrusted_host", "Processor only answers to its own loopback address"))
			return
		}
		if g == mutate || g == bootstrap {
			if r.Header.Get("Origin") != "http://"+r.Host {
				a.log.Add("guard", "bad_origin", "", "state-changing request refused: missing or foreign Origin")
				a.metrics.Guard("untrusted_origin")
				writeJSON(w, http.StatusForbidden, errBody("untrusted_origin", "this action must come from the Processor page itself"))
				return
			}
			if ct := r.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
				a.metrics.Guard("not_json")
				writeJSON(w, http.StatusUnsupportedMediaType, errBody("json_required", "Content-Type must be application/json"))
				return
			}
		}
		if g == private || g == mutate {
			rec, ok := a.sess.Lookup(r)
			if !ok {
				a.metrics.Guard("no_session")
				writeJSON(w, http.StatusUnauthorized, errBody("no_operator_session", "sign in with this run's operator code first"))
				return
			}
			if g == mutate && (len(r.Header.Get("X-TP-CSRF")) != 64 || r.Header.Get("X-TP-CSRF") != rec.csrf) {
				a.log.Add("guard", "bad_csrf", "", "state-changing request refused: missing or wrong CSRF token")
				a.metrics.Guard("csrf")
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

	m.HandleFunc("GET /api/metrics", a.wrap(private, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, a.metrics.Snapshot(a.started, a.sess.Count()))
	}))
	m.HandleFunc("GET /api/flow", a.wrap(private, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]any{"traces": a.flow.Recent(),
			"note": "Each protected request, split into requested / authorized / executed / observed. Stages that never happened are shown as such."})
	}))
	m.HandleFunc("GET /api/logs", a.wrap(private, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		kind := r.URL.Query().Get("kind")
		var b strings.Builder
		fmt.Fprintf(&b, "Thought Processor events (run %s, build %s) — Processor's own events only\n", a.runID, shortRev(a.build.Revision))
		for _, e := range a.log.Recent(200) {
			if kind != "" && e.Kind != kind {
				continue
			}
			fmt.Fprintf(&b, "%s %-9s %-40s %s %s\n", e.At.Format(time.RFC3339), e.Kind, e.Code, e.OpID, e.Detail)
		}
		io.WriteString(w, b.String())
	}))

	m.HandleFunc("POST /api/candidates", a.wrap(mutate, a.handleCandidates))
	m.HandleFunc("POST /api/probe", a.wrap(mutate, func(w http.ResponseWriter, r *http.Request) {
		var in struct{}
		if err := readStrict(r, &in); err != nil {
			writeJSON(w, 400, errBody("bad_request", "this action takes no parameters"))
			return
		}
		a.Probe(r.Context())
		a.LinkView(r.Context())
		writeJSON(w, 200, a.State())
	}))
	m.HandleFunc("POST /api/engine/{action}", a.wrap(mutate, a.handleProtected))
	a.linkRoutes(m)
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
		a.metrics.SignIn(false)
		writeJSON(w, 401, errBody("wrong_code", "that is not this run's operator code"))
		return
	}
	http.SetCookie(w, &http.Cookie{Name: cookieName, Value: id, Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode})
	a.log.Add("session", "signed_in", "", "local operator session created")
	a.metrics.SignIn(true)
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
	if act == ActAttach || act == ActObserve {
		a.log.Add("control", "moved_to_engine_card", opID, string(act)+": this parameterless boundary no longer handles it; use the Engine card (attach code)")
		writeJSON(w, http.StatusConflict, map[string]any{"op_id": opID, "action": act, "status": "USE_ENGINE_CARD", "error": "moved",
			"message": "Attach and Send observation work through the Thought Engine card: the owner types the Engine's one-time attach code, and the Engine checks Processor's key on every request.",
			"effects": "none"})
		return
	}
	var in struct{}
	if err := readStrict(r, &in); err != nil {
		a.metrics.Refused(string(act))
		a.flow.Add(refusedTrace(opID, act, time.Now().UTC(), true, "not asked: the request carried parameters, which this boundary never accepts"))
		a.log.Add("control", "parameters_refused", opID, string(act)+": refused — this boundary accepts no parameters")
		writeJSON(w, 400, map[string]any{"op_id": opID, "action": act, "status": "REFUSED", "error": "parameters_not_accepted",
			"message": "Protected operations accept no paths, arguments, endpoints or targets.", "effects": "none"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	err := a.auth.Decide(ctx, act)
	a.metrics.Refused(string(act))
	code, status := "blocked_no_identity_approval_service", http.StatusNotImplemented
	if err == ErrIngestSeparate {
		code = "blocked_no_ingest_grant_contract"
	}
	msg := "unknown"
	if err != nil {
		msg = err.Error()
	}
	a.log.Add("control", code, opID, string(act)+": BLOCKED before any effect — "+msg)
	a.flow.Add(refusedTrace(opID, act, time.Now().UTC(), false, msg))
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
	if idr != nil {
		a.metrics.Probe("identity", idr)
	}
	if enr != nil {
		a.metrics.Probe("engine", enr)
	}
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
		if a.engProbe == nil || a.engProbe.Port != enr.Port || a.engProbe.Metadata != enr.Metadata {
			a.log.Add("probe", "engine_"+strings.ToLower(enr.Port+"_"+enr.Metadata), "", "Engine candidate "+enr.Target+": port "+enr.Port+", public health "+enr.Metadata)
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
	a.mu.Unlock()
	refusals := a.metrics.RefusedTotal()

	avail, availWhy := a.auth.Available()
	id := identityView(c.Identity, idp, now, avail, availWhy)
	en := engineView(c.Engine, enp, now, a.link.snapshot())

	rec := a.link.snapshot()
	lv := a.lastLinkView()
	ops := []Operation{
		{Action: ActStart, Label: protectedActions[ActStart], Status: "BLOCKED",
			Reason: "Blocked: this build has no process runner and Identity has no start approval service. Start the Engine yourself (one command, see the runbook)."},
		attachOp(rec, lv),
		{Action: ActStop, Label: protectedActions[ActStop], Status: "BLOCKED",
			Reason: "Blocked: this Processor started no Engine (it cannot start one). Stop the Engine in its own window with Ctrl+C."},
		observeOp(rec, lv),
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
		"selection":    selection(c, enp, a.lastLinkView()),
		"next_action":  nextAction(c, idp, enp, now, candErr, a.link.snapshot(), a.lastLinkView()),
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

func engineView(addr string, p *ProbeResult, now time.Time, rec linkRecord) DepView {
	v := DepView{Name: "Thought Engine", Configured: addr, Probe: p, Freshness: freshness(p, now),
		Approval: "NOT ATTACHED — type the Engine's one-time attach code in the card below",
		Explain:  "Thought Engine does the actual thinking. Processor reads its public health; everything else only through an owner-approved attachment."}
	if rec.Token != "" && rec.Engine == addr {
		v.Approval = "ATTACHED (" + rec.AttachmentID + ") — owner-approved with the Engine's one-time code; the Engine checks the key on every request. Not an Identity grant."
	}
	switch {
	case addr == "":
		v.State, v.Label = "NOT_CONFIGURED", "No Engine address set"
	case p == nil:
		v.State, v.Label = "NOT_CHECKED", "Not checked yet"
	case v.Freshness == "STALE":
		v.State, v.Label = "STALE", "Last check is too old to trust — re-check"
	case p.Port != "OPEN":
		v.State, v.Label = "UNREACHABLE", "Not running ("+strings.ToLower(p.Port)+")"
	case p.Metadata == "RESPONDED" && p.ClaimMatches:
		v.State, v.Label = "CLAIMS_ENGINE", "Running — says it is Thought Engine"
	case p.Metadata == "RESPONDED":
		v.State, v.Label = "ANSWERED_NOT_ENGINE", "Something answered, but not a Thought Engine with the Processor contract"
	case p.Metadata == "NO_PUBLIC_METADATA":
		v.State, v.Label = "NO_PUBLIC_METADATA", "Port open; no Processor contract (older Engine build?)"
	default:
		v.State, v.Label = "PORT_OPEN_UNIDENTIFIED", "Port open — not identified"
	}
	return v
}

func nextAction(c Candidates, idp, enp *ProbeResult, now time.Time, candErr string, rec linkRecord, lv *LinkView) NextAction {
	attached := rec.Token != "" && rec.Engine == c.Engine && c.Engine != ""
	switch {
	case candErr != "":
		return NextAction{"fix_profile", "Fix the saved addresses", "The profile's candidates.json could not be used and was left unchanged. Re-enter the Engine address below to replace it.", "engine-addr"}
	case c.Engine == "":
		return NextAction{"set_engine", "Tell Processor where the Thought Engine runs", "Start the Thought Engine (one command, see the runbook), then enter its address — normally 127.0.0.1:7798. Processor only looks; it never starts programs.", "engine-addr"}
	case enp == nil || freshness(enp, now) == "STALE":
		return NextAction{"recheck", "Check the Engine now", "The last check is missing or too old to trust.", "recheck"}
	case enp.Port != "OPEN":
		return NextAction{"start_engine", "Start the Thought Engine, then re-check", "Nothing answers at " + c.Engine + ". Start it in its own window (see the runbook); Processor will not start it for you.", "recheck"}
	case !enp.ClaimMatches && !attached:
		return NextAction{"wrong_engine", "Check the Engine address or build", enp.Note, "engine-addr"}
	case !attached || (lv != nil && lv.State == "ATTACHMENT_REFUSED"):
		return NextAction{"attach_engine", "Attach to the Thought Engine", "Type the attach code the Engine shows (its window prints a line starting “attach :”, or press “Make an attach code” on the Engine's page).", "attach-code"}
	case rec.EntityID == "":
		return NextAction{"create_identity", "Create or choose an identity", "Observations belong to an identity (for example TASS). Create one on the Engine from here; Thought Processing starts off.", "id-name"}
	case lv != nil && lv.Activity != nil && !activityProcessing(lv.Activity):
		return NextAction{"turn_on", "Switch Thought Processing on", "The Engine refuses observations for an identity until Thought Processing is on.", "proc-toggle"}
	case lv != nil && lv.Status != nil && workerState(lv.Status) != "READY":
		return NextAction{"start_worker", "Start Chat-Jimmy (observations will wait)", "The Engine's AI worker is not ready, so observations are saved and wait. Start Chat-Jimmy live (see the runbook); nothing is made up meanwhile.", "obs-text"}
	default:
		return NextAction{"observe", "Send an observation", "Type something that happened (for example “Player entered Goldshire”) and press Send. Watch it go Spark → Trace → Thought below.", "obs-text"}
	}
}

func activityProcessing(act map[string]any) bool {
	id, _ := act["identity"].(map[string]any)
	on, _ := id["thought_processing"].(bool)
	return on
}

func workerState(st map[string]any) string {
	w, _ := st["worker"].(map[string]any)
	s, _ := w["state"].(string)
	return s
}

func attachOp(rec linkRecord, lv *LinkView) Operation {
	op := Operation{Action: ActAttach, Label: protectedActions[ActAttach]}
	switch {
	case rec.Token != "" && lv != nil && lv.State == "ATTACHED":
		op.Enabled, op.Status, op.Reason = true, "DONE", "Attached ("+rec.AttachmentID+") with the Engine's one-time code. Detach any time here or on the Engine's page."
	case rec.Token != "":
		op.Enabled, op.Status, op.Reason = true, "ATTACHED · CHECK", "A key is saved; the Engine's answer decides whether it still works (see the Thought Engine card)."
	default:
		op.Enabled, op.Status, op.Reason = true, "AVAILABLE", "Needs the owner's one-time attach code from the Engine. Use the Thought Engine card."
	}
	return op
}

func observeOp(rec linkRecord, lv *LinkView) Operation {
	op := Operation{Action: ActObserve, Label: protectedActions[ActObserve], Status: "NOT YET",
		Reason: "Needs: attached → identity chosen → Thought Processing on. The Engine enforces each of these, not this page."}
	if rec.Token != "" && rec.EntityID != "" && lv != nil && lv.Activity != nil && activityProcessing(lv.Activity) {
		op.Enabled, op.Status, op.Reason = true, "AVAILABLE", "Attached, identity chosen, Thought Processing on. If the AI worker isn't ready, observations wait (saved) and the card says so."
	}
	return op
}

func (a *App) capabilities() []Capability {
	_, why := a.auth.Available()
	return []Capability{
		{"self_status", "Report its own process, port, build and profile", "WORKING", "values read from the running process and Go build metadata"},
		{"single_instance", "Refuse a second Processor on the same profile", "WORKING", "kernel file lock on runtime/processor.lock (Linux tested)"},
		{"operator_session", "Local operator sign-in (per-run code)", "WORKING", "not an Identity approval; gates Processor's own pages only"},
		{"request_guards", "Host, Origin, CSRF and JSON checks on actions", "WORKING", "no CORS; GET never changes anything"},
		{"identity_probe", "Look at an explicitly entered Identity address", "WORKING", "TCP check + public GET /api/health; result labeled unverified"},
		{"engine_probe", "Look at an explicitly entered Engine address", "WORKING", "TCP check + the Engine's public GET /api/health; nothing private"},
		{"copy_context", "Copy a sanitized diagnostic summary", "WORKING", "allowlisted fields; no codes, cookies, bodies or remote claims"},
		{"identity_authentication", "Prove the Identity candidate is genuine", "UNAVAILABLE", "Identity exposes no peer-authentication mechanism"},
		{"lifecycle_approval", "Ask Identity to approve start / stop", "BLOCKED", why},
		{"runner", "Start or stop an Engine executable", "BLOCKED", "no runner exists; this build deliberately contains none — start the Engine with one command (runbook)"},
		{"engine_attach", "Attach to a running Engine and read its status, worker and activity", "WORKING", "owner types the Engine's one-time attach code; Engine stores only a hash of Processor's key and checks it on every request (interim; not an Identity grant)"},
		{"engine_original_ui", "Open the Engine's own page", "WORKING", "a plain link to the Engine's own page, offered once attached"},
		{"observation_ingest", "Send an observation and follow it Spark → Trace → Thought", "WORKING", "Engine-enforced: attachment key + the identity's Thought Processing switch; waits honestly when the AI worker isn't ready"},
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
			httpS := "none"
			if d.Probe.HTTPStatus != 0 {
				httpS = fmt.Sprint(d.Probe.HTTPStatus)
			}
			fmt.Fprintf(&w, "  last check %s  %d ms  port=%s metadata=%s http=%s claims_identity=%v\n",
				d.Probe.CheckedAt.Format(time.RFC3339), d.Probe.ElapsedMS, d.Probe.Port, d.Probe.Metadata, httpS, d.Probe.ClaimMatches)
		}
	}
	if lv := a.lastLinkView(); lv != nil {
		fmt.Fprintf(&w, "engine link: state=%s attachment=%s identity_chosen=%v\n", lv.State, lv.AttachmentID, lv.EntityID != "")
		if lv.Status != nil {
			eng, _ := lv.Status["engine"].(map[string]any)
			wk, _ := lv.Status["worker"].(map[string]any)
			fmt.Fprintf(&w, "  engine version=%v commit=%v test_fixture_worker=%v waiting=%v\n", eng["version"], eng["commit"], eng["test_fixture_worker"], lv.Status["waiting"])
			fmt.Fprintf(&w, "  worker state=%v test_fixture=%v\n", wk["state"], wk["test_fixture"])
		}
		if lv.Activity != nil {
			counts, _ := lv.Activity["counts"].(map[string]any)
			states := map[string]int{}
			obs, _ := lv.Activity["observations"].([]any)
			for _, o := range obs {
				if m, ok := o.(map[string]any); ok {
					st, _ := m["state"].(string)
					states[sanitizeClaim(st)]++
				}
			}
			ths, _ := lv.Activity["thoughts"].([]any)
			fmt.Fprintf(&w, "  identity: thought_processing=%v observations(recent)=%v traces=%v thoughts=%d\n", activityProcessing(lv.Activity), states, counts["traces"], len(ths))
		}
	}
	for _, op := range s["operations"].([]Operation) {
		fmt.Fprintf(&w, "operation %s: %s\n", op.Action, op.Status)
	}
	na := s["next_action"].(NextAction)
	fmt.Fprintf(&w, "next action: %s\n", na.Code)
	m := a.metrics.Snapshot(a.started, a.sess.Count())
	fmt.Fprintf(&w, "metrics: refused=%v rejected=%v\n", m["refused_operations"], m["rejected_requests"])
	for k, v := range m["probes"].(map[string]any) {
		pm := v.(map[string]any)
		fmt.Fprintf(&w, "  %s checks=%v outcomes=%v latency=%v\n", k, pm["checks"], pm["outcomes"], pm["latency_last_50"])
	}
	for _, tr := range a.flow.Recent() {
		if len(tr.Stages) == 4 {
			fmt.Fprintf(&w, "flow %s %s %s: requested=%s authorized=%s executed=%s observed=%s\n", tr.At.Format("15:04:05"), tr.OpID, tr.Action,
				tr.Stages[0].Status, tr.Stages[1].Status, tr.Stages[2].Status, tr.Stages[3].Status)
		}
	}
	fmt.Fprintf(&w, "recent events:\n")
	for _, e := range a.log.Recent(20) {
		fmt.Fprintf(&w, "  %s %s/%s %s\n", e.At.Format("15:04:05"), e.Kind, e.Code, e.OpID)
	}
	fmt.Fprintf(&w, "omitted by design: operator code, session cookies, CSRF tokens, the Engine attach code and key, observation text, Thought questions, identity names, request bodies, environment variables, text returned by other programs (claims), log detail text\n")
	out := w.String()
	if len(out) > 8192 {
		out = out[:8192] + "\n[truncated at 8 KiB]\n"
	}
	return out
}

// selection keeps the three things the story says must never be confused:
// the entity being served, the feature (independent app) and the running
// instance. Each is filled only from evidence Processor actually has.
func selection(c Candidates, enp *ProbeResult, lv *LinkView) map[string]any {
	feature := map[string]any{"status": "NONE", "label": "No Engine address set",
		"why": "Enter the Thought Engine's address. Processor only looks at addresses you give it."}
	instance := map[string]any{"status": "UNKNOWN", "label": "No running instance identified",
		"why": "An instance is identified by an owner-approved attachment. None yet."}
	entity := map[string]any{"status": "NONE", "label": "No identity chosen",
		"why": "Identities (for example TASS) live on the attached Engine. Attach first, then create or choose one."}
	if c.Engine != "" {
		feature = map[string]any{"status": "CANDIDATE", "label": "Thought Engine at " + c.Engine,
			"why": "Address you entered. Not registered or verified by Identity."}
		if enp != nil && enp.Port == "OPEN" {
			instance["label"] = "Something is listening at " + c.Engine + " — not attached"
			instance["why"] = "Its public health is only a self-description."
		}
	}
	if lv != nil && lv.State == "ATTACHED" {
		eng, _ := lv.Status["engine"].(map[string]any)
		inst, _ := eng["installation_id"].(string)
		instance = map[string]any{"status": "ATTACHED", "label": "Engine installation " + shortID(inst) + " (attachment " + lv.AttachmentID + ")",
			"why": "The Engine accepted this Processor's key just now."}
		if lv.Activity != nil {
			id, _ := lv.Activity["identity"].(map[string]any)
			name, _ := id["display_name"].(string)
			entity = map[string]any{"status": "CHOSEN", "label": name, "why": "An identity on the attached Engine. Its name comes from the Engine."}
		}
	}
	return map[string]any{"entity": entity, "feature": feature, "instance": instance}
}

func shortID(s string) string {
	if len(s) > 8 {
		return s[:8]
	}
	return s
}
