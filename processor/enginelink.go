package main

// Engine link: Processor attached to one Thought Engine through the Engine's
// documented contract (Thought-engine docs/PROCESSOR_CONTRACT.md,
// protocol thought-engine.processor/1-experimental).
//
// How permission works here (interim, owner-approved, local):
//   - The ENGINE shows a one-time attach code (its console at start, or its
//     page). Only the owner can see it.
//   - The owner types that code into Processor. Processor sends it once to
//     the Engine and receives a token. The Engine keeps only the token's hash;
//     Processor keeps the token in its profile (engine-link.json, 0600).
//   - The Engine enforces every call (token, per-identity Thought Processing
//     switch). Processor's buttons are convenience, not the lock.
//   - This is NOT an Identity grant. Start/stop stay blocked (no runner).
//
// Logged by Processor: fixed strings and IDs only — never the code, the token,
// observation text or Engine answers.
import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const engineProtocolPrefix = "thought-engine.processor/"

type linkRecord struct {
	Schema         int        `json:"schema_version"`
	ClientID       string     `json:"client_id"`
	Engine         string     `json:"engine,omitempty"`
	AttachmentID   string     `json:"attachment_id,omitempty"`
	Token          string     `json:"token,omitempty"`
	InstallationID string     `json:"engine_installation_id,omitempty"`
	AttachedAt     *time.Time `json:"attached_at,omitempty"`
	EntityID       string     `json:"entity_id,omitempty"`
}

type EngineLink struct {
	mu   sync.Mutex
	path string
	rec  linkRecord
	err  string
	http *http.Client
}

func OpenEngineLink(dir string) *EngineLink {
	l := &EngineLink{path: filepath.Join(dir, "engine-link.json"), http: &http.Client{
		Timeout:       4 * time.Second,
		Transport:     &http.Transport{Proxy: nil, DisableKeepAlives: true, DialContext: (&net.Dialer{Timeout: tcpTimeout}).DialContext},
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}}
	b, err := os.ReadFile(l.path)
	switch {
	case errors.Is(err, os.ErrNotExist):
	case err != nil:
		l.err = "engine-link.json can't be read; left unchanged"
	default:
		if json.Unmarshal(b, &l.rec) != nil {
			l.err = "engine-link.json is unreadable; left unchanged (Attach again to replace it)"
			l.rec = linkRecord{}
		}
	}
	if l.rec.ClientID == "" {
		l.rec.ClientID = newID("tp-")
	}
	l.rec.Schema = 1
	return l
}

func (l *EngineLink) snapshot() linkRecord {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.rec
}

func (l *EngineLink) save(next linkRecord) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := writeJSONAtomic(l.path, next); err != nil {
		return err
	}
	l.rec, l.err = next, ""
	return nil
}

// EngineError is a non-2xx answer from the Engine.
type EngineError struct {
	Status int
	Msg    string
}

func (e *EngineError) Error() string { return e.Msg }

var errEngineDown = errors.New("the Engine is not answering")

func (l *EngineLink) call(ctx context.Context, method, base, path, token string, in, out any) error {
	var body io.Reader
	if in != nil {
		b, _ := json.Marshal(in)
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, base+path, body)
	if err != nil {
		return err
	}
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := l.http.Do(req)
	if err != nil {
		return errEngineDown
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 512<<10))
	if res.StatusCode/100 != 2 {
		var e struct {
			Error string `json:"error"`
		}
		json.Unmarshal(raw, &e)
		msg := strings.TrimSpace(e.Error)
		if msg == "" {
			msg = fmt.Sprintf("the Engine answered HTTP %d", res.StatusCode)
		}
		return &EngineError{Status: res.StatusCode, Msg: clip(msg, 300)}
	}
	if out != nil {
		if err := json.Unmarshal(raw, out); err != nil {
			return fmt.Errorf("the Engine's answer could not be read")
		}
	}
	return nil
}

func clip(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n]) + "…"
	}
	return s
}

// ---------- what the page shows ----------

type LinkView struct {
	State        string         `json:"state"` // NO_ENGINE_ADDRESS | ENGINE_NOT_RUNNING | NOT_THOUGHT_ENGINE | NOT_ATTACHED | ATTACHMENT_REFUSED | ATTACHED_OTHER_ADDRESS | ATTACHED
	Label        string         `json:"label"`
	Detail       string         `json:"detail"`
	Engine       string         `json:"engine"`
	AttachmentID string         `json:"attachment_id,omitempty"`
	AttachedAt   *time.Time     `json:"attached_at,omitempty"`
	OpenURL      string         `json:"open_url,omitempty"`
	Status       map[string]any `json:"engine_status,omitempty"`
	EntityID     string         `json:"entity_id,omitempty"`
	Activity     map[string]any `json:"activity,omitempty"`
	Problem      string         `json:"problem,omitempty"`
	CheckedAt    time.Time      `json:"checked_at"`
}

// View reads the Engine live (status and, if an identity is chosen, its
// activity) through the attachment. Nothing is cached or assumed.
func (a *App) LinkView(ctx context.Context) LinkView {
	v := a.linkViewLive(ctx)
	a.mu.Lock()
	a.lastLink = &v
	a.mu.Unlock()
	return v
}

func (a *App) lastLinkView() *LinkView {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.lastLink == nil {
		return nil
	}
	x := *a.lastLink
	return &x
}

func (a *App) linkViewLive(ctx context.Context) LinkView {
	a.mu.Lock()
	addr := a.cands.Engine
	var enp *ProbeResult
	if a.engProbe != nil {
		x := *a.engProbe
		enp = &x
	}
	a.mu.Unlock()
	rec := a.link.snapshot()
	v := LinkView{Engine: addr, CheckedAt: time.Now().UTC(), Problem: a.link.err}
	switch {
	case addr == "":
		v.State, v.Label, v.Detail = "NO_ENGINE_ADDRESS", "No Engine address set", "Enter the Thought Engine's address (default 127.0.0.1:7798) in the Thought Engine card."
		return v
	case rec.Token != "" && rec.Engine != addr:
		v.State, v.Label = "ATTACHED_OTHER_ADDRESS", "Attached to a different address"
		v.Detail = "This Processor is attached to the Engine at " + strings.TrimPrefix(rec.Engine, "http://") + ", not the address set now. Attach again with a code from the Engine at " + strings.TrimPrefix(addr, "http://") + "."
		return v
	case rec.Token == "":
		switch {
		case enp == nil:
			v.State, v.Label, v.Detail = "NOT_ATTACHED", "Not checked yet", "Press Re-check."
		case enp.Port != "OPEN":
			v.State, v.Label = "ENGINE_NOT_RUNNING", "Thought Engine is not running"
			v.Detail = "Nothing answers at " + strings.TrimPrefix(addr, "http://") + ". Start it yourself (see the runbook), then press Re-check. Processor does not start programs."
		case !enp.ClaimMatches:
			v.State, v.Label = "NOT_THOUGHT_ENGINE", "This doesn't look like a Thought Engine with the Processor contract"
			v.Detail = enp.Note
		default:
			v.State, v.Label = "NOT_ATTACHED", "Running · not attached"
			v.Detail = "Type the Engine's attach code below. The Engine prints it in its window when it starts (line starting “attach :”), or press “Make an attach code” on the Engine's own page."
		}
		return v
	}
	v.AttachmentID, v.AttachedAt, v.EntityID = rec.AttachmentID, rec.AttachedAt, rec.EntityID
	cctx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()
	var st map[string]any
	err := a.link.call(cctx, "GET", rec.Engine, "/api/v1/status", rec.Token, nil, &st)
	var ee *EngineError
	switch {
	case errors.Is(err, errEngineDown):
		v.State, v.Label = "ENGINE_NOT_RUNNING", "Attached · Thought Engine is not running"
		v.Detail = "The attachment is kept. Start the Engine again (same data folder) and this card comes back by itself."
		return v
	case errors.As(err, &ee) && ee.Status == 401:
		v.State, v.Label = "ATTACHMENT_REFUSED", "The Engine no longer accepts this attachment"
		v.Detail = "It was detached on the Engine's page, or the Engine uses a different data folder now. Attach again with a new code."
		return v
	case err != nil:
		v.State, v.Label, v.Detail = "ENGINE_NOT_RUNNING", "The Engine answered with an error", err.Error()
		return v
	}
	v.State, v.Label, v.Status = "ATTACHED", "Attached", st
	v.OpenURL = rec.Engine + "/"
	v.Detail = "Owner-approved attachment " + rec.AttachmentID + ". The Engine checks this Processor's key on every request."
	if rec.EntityID != "" {
		var act map[string]any
		if err := a.link.call(cctx, "GET", rec.Engine, "/api/v1/identities/"+rec.EntityID+"/activity?limit=25", rec.Token, nil, &act); err == nil {
			v.Activity = act
		} else if errors.As(err, &ee) && ee.Status == 404 {
			v.EntityID = ""
			v.Problem = "The identity chosen earlier no longer exists on this Engine; choose or create one."
		} else {
			v.Problem = "Activity could not be read: " + err.Error()
		}
	}
	return v
}

// ---------- actions ----------

func (a *App) linkRoutes(m *http.ServeMux) {
	m.HandleFunc("GET /api/link", a.wrap(private, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, a.LinkView(r.Context()))
	}))
	m.HandleFunc("POST /api/link/attach", a.wrap(mutate, func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Code string `json:"code"`
		}
		if err := readStrict(r, &in); err != nil || len(in.Code) > 32 {
			writeJSON(w, 400, errBody("bad_request", "expected {\"code\": \"ABCD-EFGH\"}"))
			return
		}
		a.mu.Lock()
		addr := a.cands.Engine
		a.mu.Unlock()
		if addr == "" {
			writeJSON(w, 409, errBody("no_engine_address", "Enter the Engine's address first."))
			return
		}
		rec := a.link.snapshot()
		var out struct {
			AttachmentID string `json:"attachment_id"`
			Token        string `json:"token"`
			Engine       struct {
				InstallationID string `json:"installation_id"`
			} `json:"engine"`
		}
		ctx, cancel := context.WithTimeout(r.Context(), 4*time.Second)
		defer cancel()
		err := a.link.call(ctx, "POST", addr, "/api/v1/attach", "", map[string]string{"code": strings.TrimSpace(in.Code), "client_id": rec.ClientID, "label": "Thought Processor"}, &out)
		if err != nil {
			a.log.Add("engine", "attach_refused", "", "attach to the Engine at "+addr+" refused or failed (the code is not logged)")
			writeJSON(w, 400, errBody("attach_failed", engineMsg(err)))
			return
		}
		if !strings.HasPrefix(out.Token, "tea_") || len(out.Token) != 68 {
			writeJSON(w, 502, errBody("attach_failed", "the Engine's answer did not contain a usable key"))
			return
		}
		now := time.Now().UTC()
		next := rec
		next.Engine, next.AttachmentID, next.Token, next.InstallationID, next.AttachedAt = addr, out.AttachmentID, out.Token, out.Engine.InstallationID, &now
		if rec.InstallationID != out.Engine.InstallationID {
			next.EntityID = "" // a different Engine: its identities are different
		}
		if err := a.link.save(next); err != nil {
			writeJSON(w, 500, errBody("save_failed", "attached, but the key could not be saved in the profile; attach again"))
			return
		}
		a.log.Add("engine", "attached", out.AttachmentID, "attached to the Thought Engine at "+addr+" with the owner's one-time code")
		writeJSON(w, 200, a.LinkView(r.Context()))
	}))
	m.HandleFunc("POST /api/link/detach", a.wrap(mutate, func(w http.ResponseWriter, r *http.Request) {
		var in struct{}
		if err := readStrict(r, &in); err != nil {
			writeJSON(w, 400, errBody("bad_request", "this action takes no parameters"))
			return
		}
		rec := a.link.snapshot()
		note := "detached"
		if rec.Token != "" {
			ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
			if err := a.link.call(ctx, "POST", rec.Engine, "/api/v1/detach", rec.Token, struct{}{}, nil); err != nil {
				note = "forgot the key here; the Engine could not be told (" + engineMsg(err) + ") — detach it on the Engine's page too"
			}
			cancel()
		}
		next := linkRecord{Schema: 1, ClientID: rec.ClientID}
		if err := a.link.save(next); err != nil {
			writeJSON(w, 500, errBody("save_failed", "could not update the profile"))
			return
		}
		a.log.Add("engine", "detached", rec.AttachmentID, note)
		writeJSON(w, 200, a.LinkView(r.Context()))
	}))
	m.HandleFunc("POST /api/link/identity", a.wrap(mutate, func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			DisplayName string `json:"display_name"`
		}
		if err := readStrict(r, &in); err != nil {
			writeJSON(w, 400, errBody("bad_request", "expected {\"display_name\": \"TASS\"}"))
			return
		}
		rec := a.link.snapshot()
		var out struct {
			EntityID string `json:"entity_id"`
		}
		if err := a.linkCall(r, rec, "POST", "/api/v1/identities", map[string]string{"display_name": in.DisplayName, "entity_type": "ai_companion"}, &out); err != nil {
			writeJSON(w, 400, errBody("engine_refused", engineMsg(err)))
			return
		}
		rec.EntityID = out.EntityID
		a.link.save(rec)
		a.log.Add("engine", "identity_created", out.EntityID, "identity created on the Engine (Thought Processing starts OFF)")
		writeJSON(w, 200, a.LinkView(r.Context()))
	}))
	m.HandleFunc("POST /api/link/select", a.wrap(mutate, func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			EntityID string `json:"entity_id"`
		}
		if err := readStrict(r, &in); err != nil || len(in.EntityID) > 64 || strings.ContainsAny(in.EntityID, "/?#") {
			writeJSON(w, 400, errBody("bad_request", "expected {\"entity_id\": \"…\"}"))
			return
		}
		rec := a.link.snapshot()
		rec.EntityID = in.EntityID
		if err := a.link.save(rec); err != nil {
			writeJSON(w, 500, errBody("save_failed", "could not update the profile"))
			return
		}
		writeJSON(w, 200, a.LinkView(r.Context()))
	}))
	m.HandleFunc("POST /api/link/processing", a.wrap(mutate, func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Enabled bool `json:"enabled"`
		}
		if err := readStrict(r, &in); err != nil {
			writeJSON(w, 400, errBody("bad_request", "expected {\"enabled\": true|false}"))
			return
		}
		rec := a.link.snapshot()
		if rec.EntityID == "" {
			writeJSON(w, 409, errBody("no_identity", "Choose or create an identity first."))
			return
		}
		if err := a.linkCall(r, rec, "POST", "/api/v1/identities/"+rec.EntityID+"/processing", map[string]bool{"enabled": in.Enabled}, nil); err != nil {
			writeJSON(w, 400, errBody("engine_refused", engineMsg(err)))
			return
		}
		a.log.Add("engine", map[bool]string{true: "processing_on", false: "processing_off"}[in.Enabled], rec.EntityID, "Thought Processing switched by the owner through Processor; the Engine records it")
		writeJSON(w, 200, a.LinkView(r.Context()))
	}))
	m.HandleFunc("POST /api/link/observe", a.wrap(mutate, func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Text string `json:"text"`
		}
		dec := json.NewDecoder(io.LimitReader(r.Body, 8<<10))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&in); err != nil || strings.TrimSpace(in.Text) == "" {
			writeJSON(w, 400, errBody("bad_request", "type an observation first"))
			return
		}
		rec := a.link.snapshot()
		if rec.EntityID == "" {
			writeJSON(w, 409, errBody("no_identity", "Choose or create an identity first."))
			return
		}
		obsID := newID("tp-obs-")
		var out struct {
			Observation map[string]any `json:"observation"`
		}
		if err := a.linkCall(r, rec, "POST", "/api/v1/identities/"+rec.EntityID+"/observations", map[string]string{"observation_id": obsID, "text": in.Text}, &out); err != nil {
			a.log.Add("engine", "observation_refused", obsID, "the Engine did not accept an observation ("+engineCode(err)+")")
			writeJSON(w, 400, errBody("engine_refused", engineMsg(err)))
			return
		}
		a.metrics.Observation()
		a.log.Add("engine", "observation_sent", obsID, "observation accepted by the Engine (text not logged here)")
		writeJSON(w, 202, map[string]any{"observation": out.Observation, "link": a.LinkView(r.Context())})
	}))
}

func (a *App) linkCall(r *http.Request, rec linkRecord, method, path string, in, out any) error {
	if rec.Token == "" {
		return &EngineError{Status: 409, Msg: "not attached to an Engine yet"}
	}
	ctx, cancel := context.WithTimeout(r.Context(), 4*time.Second)
	defer cancel()
	return a.link.call(ctx, method, rec.Engine, path, rec.Token, in, out)
}

func engineMsg(err error) string {
	var ee *EngineError
	switch {
	case errors.Is(err, errEngineDown):
		return "The Engine is not answering. Is it running?"
	case errors.As(err, &ee) && ee.Status == 401:
		return "The Engine no longer accepts this Processor's key. Attach again with a new code."
	case errors.As(err, &ee):
		return "The Engine said: " + ee.Msg
	}
	return err.Error()
}

func engineCode(err error) string {
	var ee *EngineError
	if errors.As(err, &ee) {
		return fmt.Sprintf("HTTP %d", ee.Status)
	}
	if errors.Is(err, errEngineDown) {
		return "not answering"
	}
	return "error"
}
