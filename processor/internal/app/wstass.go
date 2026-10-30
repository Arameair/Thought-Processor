package app

// TASS inside the Development Workstation: talk to TASS about the project
// file being viewed. Read-only: TASS is shown the file's text for one
// question through a ticketed tass.chat.send request (Identity checks it
// live, as for every TASS message). Nothing here edits files.
//
// Who TASS talks to: ONE owner person identity. An existing one is reused;
// if there is none, one is created once, named after the Admin's GitHub
// sign-in. Every permission for it is still approved by the Admin.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Arameair/Thought-Processor/processor/internal/idclient"
	"github.com/Arameair/Thought-Processor/processor/internal/store"
	"github.com/Arameair/Thought-Processor/processor/internal/supervisor"
)

const (
	tassFeatureID   = "tass"
	tassFileVersion = "0.2.0" // first TASS that accepts a file context
	capChat         = "tass.chat.send"
	capModel        = "tass.model.set"
	tassGrantSecs   = 12 * 60 * 60 // one working day; shown to the Admin
	maxFileContext  = 24000        // characters of file text TASS is shown
	defaultModel    = "qwen3.5:4b"
)

func (a *App) tassInstallation() (InstallRecord, *InstallRecord) {
	var best, old *InstallRecord
	recs := a.records()
	sort.Slice(recs, func(i, j int) bool { return recs[i].InstalledAt.After(recs[j].InstalledAt) })
	for i := range recs {
		r := recs[i]
		if r.State != "INSTALLED" || r.FeatureID != tassFeatureID {
			continue
		}
		if versionAtLeast(r.Version, tassFileVersion) {
			if best == nil {
				best = &recs[i]
			}
		} else if old == nil {
			old = &recs[i]
		}
	}
	if best == nil {
		return InstallRecord{}, old
	}
	return *best, old
}

// owner returns the owner subject if it is still a live person identity.
func (a *App) owner() (idclient.Entity, bool) {
	id := a.Prof.Settings().OwnerSubject
	a.Conn.mu.Lock()
	defer a.Conn.mu.Unlock()
	for _, e := range a.Conn.identities {
		if e.ID == id && e.ArchivedAt == nil && e.Kind == "person" {
			return e, true
		}
	}
	return idclient.Entity{}, false
}

// ensureOwner reuses the single existing person identity, or creates one.
func (a *App) ensureOwner(ctx context.Context, choose string) (idclient.Entity, []idclient.Entity, error) {
	if e, ok := a.owner(); ok && choose == "" {
		return e, nil, nil
	}
	cl, err := a.Conn.Require(ctx)
	if err != nil {
		return idclient.Entity{}, nil, err
	}
	all, err := cl.Identities(ctx)
	if err != nil {
		return idclient.Entity{}, nil, err
	}
	var people []idclient.Entity
	for _, e := range all {
		if e.Kind == "person" && e.ArchivedAt == nil {
			people = append(people, e)
		}
	}
	pick := func(e idclient.Entity) (idclient.Entity, []idclient.Entity, error) {
		a.Prof.UpdateSettings(func(s *store.Settings) { s.OwnerSubject = e.ID })
		a.Conn.refreshData(ctx)
		a.note("info", "workstation.owner", "the Development Workstation acts for "+e.Name+" ("+e.ID+")")
		return e, nil, nil
	}
	if choose != "" {
		for _, e := range people {
			if e.ID == choose {
				return pick(e)
			}
		}
		return idclient.Entity{}, nil, errors.New("that is not a person identity here")
	}
	switch len(people) {
	case 1:
		return pick(people[0])
	case 0:
		name := "Owner"
		if adm := a.Conn.adminView(); adm != nil && adm.Admin != nil && adm.Admin.GitHub.Login != "" {
			name = adm.Admin.GitHub.Login + " (owner)"
		}
		e, err := cl.CreateIdentity(ctx, name, "person")
		if err != nil {
			return idclient.Entity{}, nil, err
		}
		return pick(e)
	}
	return idclient.Entity{}, people, errors.New("CHOOSE_OWNER: more than one person identity exists; choose which one is you")
}

type grantState struct {
	Status    string     `json:"status"` // ACTIVE | PENDING_APPROVAL | MISSING
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	URL       string     `json:"approval_url,omitempty"`
}

func (a *App) tassGrant(ins, owner, capability string) grantState {
	me := "processor:" + a.Conn.Pairing().ProcessorID
	if iv := a.Conn.installation(ins); iv != nil {
		for _, g := range iv.Grants {
			if g.Status == "ACTIVE" && g.Grant.Subject == owner && g.Grant.Capability == capability && g.Grant.Requester == me {
				return grantState{Status: "ACTIVE", ExpiresAt: g.Grant.ExpiresAt}
			}
		}
	}
	a.P1.mu.Lock()
	defer a.P1.mu.Unlock()
	for _, op := range a.P1.pending {
		if op.Kind == "grant_capability" && op.InstallationID == ins && op.State == "PENDING" && time.Now().Before(op.ExpiresAt) &&
			strings.Contains(strings.Join(op.Summary, " "), capability) {
			return grantState{Status: "PENDING_APPROVAL", URL: op.ApprovalURL}
		}
	}
	return grantState{Status: "MISSING"}
}

// tassView: what the workstation's TASS panel shows, and the ONE next step.
func (a *App) tassView() map[string]any {
	out := map[string]any{"file_context_chars": maxFileContext, "default_model": defaultModel}
	rec, old := a.tassInstallation()
	if old != nil {
		out["older_tass"] = map[string]any{"installation_id": old.InstallationID, "version": old.Version,
			"note": "TASS " + old.Version + " can't read a file you show it. It stays installed with its conversation; the workstation uses TASS " + tassFileVersion + " or newer."}
	}
	for _, t := range a.txs() {
		if t.FeatureID == tassFeatureID && t.State != "FAILED" {
			out["install_pending"] = map[string]any{"approval_url": t.ApprovalURL, "version": t.Version, "state": t.State}
		}
	}
	if rec.InstallationID == "" {
		out["step"] = "INSTALL"
		return out
	}
	id := rec.InstallationID
	v := a.Sup.View(id)
	out["installation_id"], out["version"], out["runtime"] = id, rec.Version, v.State
	model := map[string]any{"state": "UNKNOWN"}
	if v.Health != nil && v.Health.Status != nil {
		if m, ok := v.Health.Status["model"].(map[string]any); ok {
			model = m
		}
	}
	out["model"] = model
	e, ok := a.owner()
	if ok {
		out["owner"] = map[string]any{"id": e.ID, "name": e.Name}
	}
	chat, mset := a.tassGrant(id, e.ID, capChat), a.tassGrant(id, e.ID, capModel)
	out["chat_grant"], out["model_grant"] = chat, mset
	ms, _ := model["state"].(string)
	switch {
	case v.State != supervisor.Running:
		out["step"] = "START"
	case !ok:
		out["step"] = "OWNER"
	case chat.Status != "ACTIVE":
		out["step"] = "ALLOW_CHAT"
	case ms != "REACHABLE" && mset.Status != "ACTIVE":
		out["step"] = "ALLOW_MODEL"
	case ms != "REACHABLE":
		out["step"] = "SET_MODEL"
	default:
		out["step"] = "READY"
	}
	return out
}

// fileContext reads one allowlisted file for TASS (same guards as the viewer).
func (a *App) fileContext(repo, path string) (map[string]any, error) {
	rp, ok := a.WS.repo(repo)
	if !ok {
		return nil, errors.New("unknown repository")
	}
	p, err := a.WS.resolve(rp, path)
	if err != nil {
		return nil, err
	}
	st, err := os.Stat(p)
	if err != nil || !st.Mode().IsRegular() {
		return nil, errors.New("not a regular file")
	}
	if st.Size() > wsMaxFile {
		return nil, errors.New("file is larger than 512 KB")
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return nil, errors.New("could not read the file")
	}
	if strings.IndexByte(string(b), 0) >= 0 || !utf8.Valid(b) {
		return nil, errors.New("binary or non-UTF-8 file; TASS is only shown text")
	}
	sum := sha256.Sum256(b)
	text, trunc := string(b), false
	if utf8.RuneCountInString(text) > maxFileContext {
		r := []rune(text)
		text, trunc = string(r[:maxFileContext]), true
	}
	rel, _ := filepath.Rel(rp.Dir, p)
	return map[string]any{"kind": "project_file", "repo": rp.Name, "path": filepath.ToSlash(rel), "head_sha": gitOut(rp.Dir, "rev-parse", "HEAD"),
		"file_sha256": hex.EncodeToString(sum[:]), "bytes": len(b), "truncated": trunc, "text": text}, nil
}

func (a *App) tassRoutes(m *http.ServeMux, gated func(level, func(http.ResponseWriter, *http.Request, wsGate)) http.HandlerFunc) {
	m.HandleFunc("GET /api/ws/tass", gated(private, func(w http.ResponseWriter, r *http.Request, g wsGate) {
		writeJSON(w, 200, a.tassView())
	}))
	m.HandleFunc("POST /api/ws/tass/install", gated(mutate, func(w http.ResponseWriter, r *http.Request, g wsGate) {
		out, err := a.requestInstall(r.Context(), tassFeatureID, false)
		if err != nil {
			a.fail(w, 409, err.Error())
			return
		}
		if out.ExistingID != "" {
			writeJSON(w, 200, map[string]any{"installation_id": out.ExistingID})
			return
		}
		writeJSON(w, 200, map[string]any{"approval_url": out.Tx.ApprovalURL, "version": out.Tx.Version})
	}))
	m.HandleFunc("POST /api/ws/tass/start", gated(mutate, func(w http.ResponseWriter, r *http.Request, g wsGate) {
		rec, _ := a.tassInstallation()
		if rec.InstallationID == "" {
			a.fail(w, 409, "TASS "+tassFileVersion+" is not installed yet")
			return
		}
		a.P1.mu.Lock()
		a.P1.userStop[rec.InstallationID] = false
		a.P1.mu.Unlock()
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := a.Sup.Start(ctx, rec.InstallationID, "started from the Development Workstation"); err != nil && a.Sup.View(rec.InstallationID).State != supervisor.Running {
			a.fail(w, 409, err.Error())
			return
		}
		writeJSON(w, 200, a.tassView())
	}))
	m.HandleFunc("POST /api/ws/tass/owner", gated(mutate, func(w http.ResponseWriter, r *http.Request, g wsGate) {
		var b struct {
			Choose string `json:"choose"`
		}
		decode(r, &b)
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()
		e, choices, err := a.ensureOwner(ctx, b.Choose)
		if err != nil {
			writeJSON(w, 409, map[string]any{"error": err.Error(), "choices": choices})
			return
		}
		writeJSON(w, 200, map[string]any{"owner": e})
	}))
	m.HandleFunc("POST /api/ws/tass/allow", gated(mutate, func(w http.ResponseWriter, r *http.Request, g wsGate) {
		var b struct {
			What string `json:"what"` // chat | model
		}
		decode(r, &b)
		capability := map[string]string{"chat": capChat, "model": capModel}[b.What]
		rec, _ := a.tassInstallation()
		e, ok := a.owner()
		if capability == "" || rec.InstallationID == "" || !ok {
			a.fail(w, 409, "install and start TASS and set the owner first")
			return
		}
		if gs := a.tassGrant(rec.InstallationID, e.ID, capability); gs.Status != "MISSING" {
			writeJSON(w, 200, map[string]any{"grant": gs})
			return
		}
		cl, err := a.Conn.Require(r.Context())
		if err != nil {
			a.failIdentity(w, err)
			return
		}
		ar, err := cl.RequestGrant(r.Context(), rec.InstallationID, e.ID, capability, tassGrantSecs, false)
		if err != nil {
			a.failIdentity(w, err)
			return
		}
		a.addPending(PendingOp{ApprovalID: ar.Approval.ID, Kind: "grant_capability", InstallationID: rec.InstallationID, ApprovalURL: ar.ApprovalURL,
			Summary: append([]string{capability}, ar.Approval.Summary...), CreatedAt: ar.Approval.CreatedAt, ExpiresAt: ar.Approval.ExpiresAt, State: "PENDING"})
		a.note("info", "grant.requested", rec.InstallationID+": workstation asked for "+capability+" for "+e.Name+" ("+ar.Approval.ID+"), waiting for the Admin")
		writeJSON(w, 200, map[string]any{"grant": grantState{Status: "PENDING_APPROVAL", URL: ar.ApprovalURL}})
	}))
	m.HandleFunc("POST /api/ws/tass/model", gated(mutate, func(w http.ResponseWriter, r *http.Request, g wsGate) {
		var b struct {
			Model string `json:"model"`
		}
		decode(r, &b)
		if b.Model == "" {
			b.Model = defaultModel
		}
		rec, _ := a.tassInstallation()
		e, ok := a.owner()
		if rec.InstallationID == "" || !ok || len(b.Model) > 100 || strings.ContainsAny(b.Model, "{} \n") {
			a.fail(w, 400, "choose a model name like qwen3.5:4b")
			return
		}
		res := a.runAction(r.Context(), rec.InstallationID, e.ID, capModel, b.Model)
		writeJSON(w, 200, map[string]any{"result": res})
	}))
	m.HandleFunc("POST /api/ws/tass/ask", gated(mutate, func(w http.ResponseWriter, r *http.Request, g wsGate) {
		var b struct {
			Question string `json:"question"`
			Repo     string `json:"repo"`
			Path     string `json:"path"`
		}
		decode(r, &b)
		b.Question = strings.TrimSpace(b.Question)
		if b.Question == "" || utf8.RuneCountInString(b.Question) > 1000 {
			a.fail(w, 400, "ask something (1–1000 characters)")
			return
		}
		rec, _ := a.tassInstallation()
		e, ok := a.owner()
		if rec.InstallationID == "" || !ok {
			a.fail(w, 409, "TASS isn't ready yet; follow the steps in the TASS panel")
			return
		}
		var extra map[string]any
		var about map[string]any
		if b.Path != "" {
			ctxFile, err := a.fileContext(b.Repo, b.Path)
			if err != nil {
				a.fail(w, 403, "this file can't be shown to TASS: "+err.Error())
				return
			}
			extra = map[string]any{"context": ctxFile}
			about = map[string]any{}
			for k, v := range ctxFile {
				if k != "text" {
					about[k] = v
				}
			}
		}
		// Detached from the browser tab: a closed tab must not cut a model answer in half.
		ctx, cancel := context.WithTimeout(context.Background(), 130*time.Second)
		defer cancel()
		res := a.runActionWith(ctx, rec.InstallationID, e.ID, capChat, b.Question, extra, 120*time.Second)
		out := map[string]any{"outcome": res.Outcome, "detail": res.Detail, "decision_id": res.DecisionID, "about": about}
		if o := res.Observation; o != nil {
			out["tass"] = o["output"]
		}
		writeJSON(w, 200, out)
	}))
}

// versionAtLeast compares dotted numeric versions ("0.10.0" >= "0.2.0").
func versionAtLeast(v, min string) bool {
	parse := func(s string) []int {
		s = strings.SplitN(strings.SplitN(s, "-", 2)[0], "+", 2)[0]
		var out []int
		for _, p := range strings.Split(s, ".") {
			n := 0
			for _, c := range p {
				if c < '0' || c > '9' {
					return nil
				}
				n = n*10 + int(c-'0')
			}
			out = append(out, n)
		}
		return out
	}
	a, b := parse(v), parse(min)
	if a == nil || b == nil {
		return false
	}
	for i := 0; i < len(a) || i < len(b); i++ {
		x, y := 0, 0
		if i < len(a) {
			x = a[i]
		}
		if i < len(b) {
			y = b[i]
		}
		if x != y {
			return x > y
		}
	}
	return true
}
