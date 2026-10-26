package app

// P1 — feature foundation in Thought Processor:
//   - Library: packages from folders the operator explicitly trusted
//     (-trust-feature-dir) whose manifest says TRUSTED_LOCAL_TEST_FIXTURE.
//     Everything else is shown BLOCKED.
//   - Install transaction: verify → stage → ask Identity (Admin approval in
//     Identity's own page) → register (Identity mints the installation ID and
//     credential) → commit folder → confirm with the credential. Idempotent
//     install key; a crash at any point resumes, never mints a second ID.
//   - Grants: ask → Admin approves in Identity → commit; revoke.
//   - Synthetic action: ticket from Identity FIRST (no ticket, no start) →
//     on-demand start if configured → request to the feature, which verifies
//     the ticket and asks Identity live → result. Never retried.
//   - Runtime: manual / on-demand / always-running, idle stop, controlled recovery.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Arameair/Thought-Processor/processor/internal/featpkg"
	"github.com/Arameair/Thought-Processor/processor/internal/idclient"
	"github.com/Arameair/Thought-Processor/processor/internal/store"
	"github.com/Arameair/Thought-Processor/processor/internal/supervisor"
)

var (
	insIDRE = regexp.MustCompile(`^ins_[0-9a-f]{20}$`)
	txKeyRE = regexp.MustCompile(`^[0-9a-f]{32}$`)
	grantRE = regexp.MustCompile(`^gnt_[0-9a-f]{20}$`)
	entIDRE = regexp.MustCompile(`^ent_[0-9a-f]{20}$`)
	capIDRE = regexp.MustCompile(`^[a-z][a-z0-9_.-]{2,63}$`)
)

// InstallRecord is Processor's local record of one installed copy.
type InstallRecord struct {
	InstallationID   string               `json:"installation_id"`
	FeatureID        string               `json:"feature_id"`
	Name             string               `json:"name"`
	Version          string               `json:"version"`
	Description      string               `json:"description"`
	PublisherClaim   string               `json:"publisher_claim"`
	ArtifactSHA256   string               `json:"artifact_sha256"`
	ManifestSHA256   string               `json:"manifest_sha256"`
	Files            []featpkg.File       `json:"files"`
	Command          []string             `json:"command"`
	Requires         []string             `json:"requires"`
	Capabilities     []featpkg.Capability `json:"capabilities"`
	StartupModes     []string             `json:"startup_modes"`
	SourceDir        string               `json:"source_dir"`
	SourceRepo       string               `json:"source_repo"`
	TrustClass       string               `json:"trust_class"`
	IdentitySystemID string               `json:"identity_system_id"`
	InstalledAt      time.Time            `json:"installed_at"`
	State            string               `json:"state"` // INSTALLED | UNINSTALLED
	UninstalledAt    *time.Time           `json:"uninstalled_at,omitempty"`
	DataKept         bool                 `json:"data_kept,omitempty"`
	Policy           supervisor.Policy    `json:"policy"`
}

// Tx is an install transaction in progress (<profile>/installations/.tx/<key>/tx.json).
type Tx struct {
	Key            string    `json:"install_key"`
	FeatureID      string    `json:"feature_id"`
	Name           string    `json:"name"`
	Version        string    `json:"version"`
	SourceDir      string    `json:"source_dir"`
	ArtifactSHA256 string    `json:"artifact_sha256"`
	ApprovalID     string    `json:"approval_id,omitempty"`
	ApprovalURL    string    `json:"approval_url,omitempty"`
	InstallationID string    `json:"installation_id,omitempty"`
	State          string    `json:"state"` // STAGED | WAITING_FOR_ADMIN | REGISTERED | FAILED
	Note           string    `json:"note,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

// PendingOp is an Admin approval Processor is waiting for.
type PendingOp struct {
	ApprovalID     string    `json:"approval_id"`
	Kind           string    `json:"kind"` // enroll_admin | grant_capability | uninstall_feature
	InstallationID string    `json:"installation_id,omitempty"`
	KeepData       bool      `json:"keep_data,omitempty"`
	ApprovalURL    string    `json:"approval_url"`
	Summary        []string  `json:"summary"`
	CreatedAt      time.Time `json:"created_at"`
	ExpiresAt      time.Time `json:"expires_at"`
	State          string    `json:"state"`
	Note           string    `json:"note,omitempty"`
}

// ActionResult is one Processor-sent request and its precise outcome.
type ActionResult struct {
	RequestID      string         `json:"request_id"`
	InstallationID string         `json:"installation_id"`
	Capability     string         `json:"capability"`
	Subject        string         `json:"subject_identity_id"`
	At             time.Time      `json:"at"`
	Outcome        string         `json:"outcome"` // COMPLETED | DENIED_BY_IDENTITY | DENIED_BY_FEATURE | AUTHORITY_UNAVAILABLE | NOT_RUNNING | REQUEST_UNCERTAIN | TICKET_REJECTED | ERROR
	Detail         string         `json:"detail"`
	DecisionID     string         `json:"decision_id,omitempty"`
	TicketID       string         `json:"ticket_id,omitempty"`
	StartedFeature bool           `json:"started_feature"`
	LatencyMS      int64          `json:"latency_ms"`
	Observation    map[string]any `json:"observation,omitempty"`
}

type p1 struct {
	mu        sync.Mutex
	trusted   []string
	pending   []PendingOp
	requests  []ActionResult
	userStop  map[string]bool
	autoDone  bool
	notes     []string
	busy      map[string]bool
	reconcile sync.Mutex
}

func (a *App) insDir() string            { return filepath.Join(a.Prof.Dir, "installations") }
func (a *App) txDir() string             { return filepath.Join(a.insDir(), ".tx") }
func (a *App) recDir(id string) string   { return filepath.Join(a.insDir(), id) }
func (a *App) recPath(id string) string  { return filepath.Join(a.recDir(id), "install.json") }
func (a *App) pendingPath() string       { return filepath.Join(a.Prof.Dir, "pending-approvals.json") }
func (a *App) credPath(id string) string { return filepath.Join(a.recDir(id), "credential.json") }
func (a *App) requestLogPath(id string) string {
	return filepath.Join(a.recDir(id), "logs", "requests.jsonl")
}

// TrustFeatureDirs records the folders the operator explicitly trusted.
func (a *App) TrustFeatureDirs(dirs []string) {
	for _, d := range dirs {
		if abs, err := filepath.Abs(d); err == nil {
			a.P1.trusted = append(a.P1.trusted, filepath.Clean(abs))
		}
	}
}

func (a *App) initP1() {
	a.P1 = &p1{userStop: map[string]bool{}, busy: map[string]bool{}}
	os.MkdirAll(a.txDir(), 0o700)
	if b, err := os.ReadFile(a.pendingPath()); err == nil {
		json.Unmarshal(b, &a.P1.pending)
	}
	a.Sup = supervisor.New()
	a.Sup.Log = func(level, event, ins, detail string) { a.Log.Add(level, "feature", event, ins+": "+detail) }
	a.Sup.Policy = func(id string) supervisor.Policy {
		r, err := a.loadRecord(id)
		if err != nil {
			return supervisor.Policy{Mode: supervisor.ModeManual, Recovery: supervisor.RecoveryManual}
		}
		return r.Policy
	}
	a.Sup.SpecFor = a.specFor
	for _, r := range a.records() {
		if r.State != "INSTALLED" {
			continue
		}
		if msg := a.Sup.CleanupOrphan(r.InstallationID, filepath.Join(a.recDir(r.InstallationID), "runtime")); msg != "" {
			a.Log.Warn("feature", "feature.orphan", r.InstallationID+": "+msg)
		}
	}
}

// ---- local records ----

func (a *App) loadRecord(id string) (InstallRecord, error) {
	var r InstallRecord
	if !insIDRE.MatchString(id) {
		return r, errors.New("not an installation ID")
	}
	b, err := os.ReadFile(a.recPath(id))
	if err != nil {
		return r, err
	}
	if err := json.Unmarshal(b, &r); err != nil || r.InstallationID != id {
		return r, errors.New("install.json is damaged")
	}
	return r, nil
}

func (a *App) saveRecord(r InstallRecord) error {
	return store.WriteJSON(a.recPath(r.InstallationID), r)
}

func (a *App) records() []InstallRecord {
	out := []InstallRecord{}
	ents, _ := os.ReadDir(a.insDir())
	for _, e := range ents {
		if !e.IsDir() || !insIDRE.MatchString(e.Name()) {
			continue
		}
		if r, err := a.loadRecord(e.Name()); err == nil {
			out = append(out, r)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].InstalledAt.Before(out[j].InstalledAt) })
	return out
}

func (a *App) txs() []Tx {
	out := []Tx{}
	ents, _ := os.ReadDir(a.txDir())
	for _, e := range ents {
		if !e.IsDir() || !txKeyRE.MatchString(e.Name()) {
			continue
		}
		var t Tx
		if b, err := os.ReadFile(filepath.Join(a.txDir(), e.Name(), "tx.json")); err == nil && json.Unmarshal(b, &t) == nil && t.Key == e.Name() {
			out = append(out, t)
		}
	}
	return out
}

func (a *App) saveTx(t Tx) error {
	return store.WriteJSON(filepath.Join(a.txDir(), t.Key, "tx.json"), t)
}

func (a *App) savePending() {
	a.P1.mu.Lock()
	list := append([]PendingOp(nil), a.P1.pending...)
	a.P1.mu.Unlock()
	store.WriteJSON(a.pendingPath(), list)
}

func (a *App) addPending(op PendingOp) {
	a.P1.mu.Lock()
	a.P1.pending = append(a.P1.pending, op)
	a.P1.mu.Unlock()
	a.savePending()
}

func (a *App) note(level, event, msg string) {
	a.Log.Add(level, "feature", event, msg)
	a.P1.mu.Lock()
	a.P1.notes = append(a.P1.notes, time.Now().Format("15:04:05")+" "+msg)
	if len(a.P1.notes) > 30 {
		a.P1.notes = a.P1.notes[len(a.P1.notes)-30:]
	}
	a.P1.mu.Unlock()
}

// ---- library ----

type LibraryItem struct {
	Dir            string               `json:"dir"`
	FeatureID      string               `json:"feature_id"`
	Name           string               `json:"name"`
	Version        string               `json:"version"`
	Description    string               `json:"description"`
	Publisher      string               `json:"publisher_claim"`
	PublisherProof string               `json:"publisher_proof"`
	SourceRepo     string               `json:"source_repo"`
	TrustClass     string               `json:"trust_class"`
	API            string               `json:"api"`
	ArtifactSHA256 string               `json:"artifact_sha256"`
	Files          []featpkg.File       `json:"files"`
	Capabilities   []featpkg.Capability `json:"capabilities"`
	Command        []string             `json:"command"`
	Requires       []string             `json:"requires"`
	Blocked        string               `json:"blocked,omitempty"`
	Error          string               `json:"error,omitempty"`
	Installed      []string             `json:"installed_as"`
	NotVerified    []string             `json:"not_verified"`
}

func (a *App) library() []LibraryItem {
	out := []LibraryItem{}
	insts := a.records()
	add := func(dir string) {
		it := LibraryItem{Dir: store.DisplayPath(dir), Installed: []string{},
			NotVerified: []string{"publisher identity (the publisher name is only a claim; nothing is signed)",
				"OS sandboxing (a feature runs with your user account's rights)",
				"release provenance (this is a local folder you trusted, not a verified release)"}}
		p, err := featpkg.Load(dir, true)
		if err != nil {
			it.Error = err.Error()
			it.Blocked = "INVALID_PACKAGE"
			out = append(out, it)
			return
		}
		m := p.Manifest
		it.FeatureID, it.Name, it.Version, it.Description = m.FeatureID, m.Name, m.Version, m.Description
		it.Publisher, it.PublisherProof, it.SourceRepo, it.TrustClass, it.API = m.Publisher.Name, "none (claim only)", m.Source.Repo+"/"+m.Source.Path, m.TrustClass, m.API
		it.ArtifactSHA256, it.Files, it.Capabilities, it.Command, it.Requires, it.Blocked = p.ArtifactSHA256, m.Files, m.Capabilities, m.Runtime.Command, m.Runtime.Requires, p.Blocked
		for _, r := range insts {
			if r.FeatureID == m.FeatureID && r.State == "INSTALLED" {
				it.Installed = append(it.Installed, r.InstallationID)
			}
		}
		out = append(out, it)
	}
	for _, d := range a.P1.trusted {
		if _, err := os.Stat(filepath.Join(d, featpkg.ManifestFile)); err == nil {
			add(d)
			continue
		}
		ents, _ := os.ReadDir(d)
		for _, e := range ents {
			if e.IsDir() {
				if _, err := os.Lstat(filepath.Join(d, e.Name(), featpkg.ManifestFile)); err == nil {
					add(filepath.Join(d, e.Name()))
				}
			}
		}
	}
	return out
}

func (a *App) findPackage(featureID string) (featpkg.Package, error) {
	for _, it := range a.library() {
		if it.FeatureID != featureID {
			continue
		}
		for _, d := range a.P1.trusted {
			for _, cand := range []string{d, filepath.Join(d, filepath.Base(it.Dir))} {
				if p, err := featpkg.Load(cand, true); err == nil && p.Manifest.FeatureID == featureID {
					if p.Blocked != "" {
						return p, errors.New(p.Blocked)
					}
					return p, nil
				}
			}
		}
	}
	return featpkg.Package{}, errors.New("UNTRUSTED_EXECUTION_BLOCKED: that feature is not in a folder you trusted (-trust-feature-dir)")
}

// ---- install transaction ----

func (a *App) requestInstall(ctx context.Context, featureID string) (Tx, error) {
	if !featpkg.FeatureIDRE.MatchString(featureID) {
		return Tx{}, errors.New("unknown feature")
	}
	cl, err := a.Conn.Require(ctx)
	if err != nil {
		return Tx{}, err
	}
	p, err := a.findPackage(featureID)
	if err != nil {
		return Tx{}, err
	}
	key := store.RandHex(16)
	t := Tx{Key: key, FeatureID: featureID, Name: p.Manifest.Name, Version: p.Manifest.Version, SourceDir: p.Dir,
		ArtifactSHA256: p.ArtifactSHA256, State: "STAGED", CreatedAt: time.Now().UTC()}
	stage := filepath.Join(a.txDir(), key, "package")
	if err := os.MkdirAll(filepath.Join(a.txDir(), key), 0o700); err != nil {
		return Tx{}, err
	}
	if err := featpkg.Stage(p, stage); err != nil {
		os.RemoveAll(filepath.Join(a.txDir(), key))
		return Tx{}, err
	}
	if err := a.saveTx(t); err != nil {
		os.RemoveAll(filepath.Join(a.txDir(), key))
		return Tx{}, err
	}
	caps := []idclient.Capability{}
	for _, c := range p.Manifest.Capabilities {
		caps = append(caps, idclient.Capability{ID: c.ID, Description: c.Description, DataClass: c.DataClass})
	}
	ar, err := cl.RequestInstall(ctx, idclient.InstallRequest{FeatureID: featureID, Name: p.Manifest.Name, Version: p.Manifest.Version,
		API: p.Manifest.API, PublisherClaim: p.Manifest.Publisher.Name, SourceKind: "local-trusted-package",
		SourceRef: p.Manifest.Source.Repo + "/" + p.Manifest.Source.Path, TrustClass: p.Manifest.TrustClass,
		ArtifactSHA256: p.ArtifactSHA256, ManifestSHA256: p.ManifestSHA256, Capabilities: caps, InstallKey: key})
	if err != nil {
		os.RemoveAll(filepath.Join(a.txDir(), key))
		return Tx{}, err
	}
	t.State, t.ApprovalID, t.ApprovalURL = "WAITING_FOR_ADMIN", ar.Approval.ID, ar.ApprovalURL
	if err := a.saveTx(t); err != nil {
		return Tx{}, err
	}
	a.note("info", "install.requested", "install of "+featureID+" "+p.Manifest.Version+" is waiting for the Admin's approval in Identity ("+ar.Approval.ID+")")
	return t, nil
}

// advanceTx moves one transaction forward. Safe to call repeatedly.
func (a *App) advanceTx(ctx context.Context, cl *idclient.Client, t Tx) {
	dir := filepath.Join(a.txDir(), t.Key)
	fail := func(why string) {
		a.note("warn", "install.failed", t.FeatureID+": "+why+" — nothing was installed")
		os.RemoveAll(dir)
	}
	if t.State == "WAITING_FOR_ADMIN" {
		ap, err := cl.Approval(ctx, t.ApprovalID)
		if err != nil {
			return
		}
		switch {
		case ap.State == "DENIED":
			fail("the Admin denied the install")
			return
		case ap.State == "EXPIRED":
			fail("the approval expired before it was approved")
			return
		case ap.State != "APPROVED" && ap.State != "CONSUMED":
			return
		}
	}
	if t.State != "WAITING_FOR_ADMIN" && t.State != "REGISTERED" {
		return
	}
	res, err := cl.CommitInstall(ctx, t.ApprovalID)
	if err != nil {
		if idclient.KindOf(err) == idclient.Refused {
			fail("Identity refused the registration: " + err.Error())
		}
		return
	}
	in := res.Installation
	if in.ArtifactSHA256 != t.ArtifactSHA256 || in.InstallKey != t.Key {
		fail("Identity answered for a different install; refused")
		return
	}
	if in.State != "PENDING_CONFIRM" || res.Credential == nil {
		if in.State == "INSTALLED" {
			if _, err := a.loadRecord(in.ID); err == nil {
				os.RemoveAll(dir) // already finished earlier
				return
			}
			fail("Identity says " + in.ID + " is INSTALLED but this Processor has no copy of its credential; uninstall it in Identity and install again")
		}
		return
	}
	cred := *res.Credential
	if cred.InstallationID != in.ID || cred.IdentitySystemID != a.Conn.Pairing().IdentitySystemID {
		fail("credential does not match the installation")
		return
	}
	if err := store.WriteJSON(filepath.Join(dir, "credential.json"), cred); err != nil {
		return
	}
	t.State, t.InstallationID = "REGISTERED", in.ID
	a.saveTx(t)
	a.note("info", "install.registered", "Identity registered installation "+in.ID+" for "+t.FeatureID+" (credential generation "+fmt.Sprint(cred.Generation)+")")
	if os.Getenv("TP_TEST_CRASH_POINT") == "after-register" {
		a.Log.Warn("test", "test.crash", "TEST-ONLY crash point after-register reached; exiting")
		os.Exit(42)
	}
	// Commit the folder: installations/<id>/{package,credential.json,data,runtime,logs,install.json}
	p, err := featpkg.Load(filepath.Join(dir, "package"), true)
	if err != nil || p.ArtifactSHA256 != t.ArtifactSHA256 {
		fail("staged files changed; refused")
		return
	}
	final := a.recDir(in.ID)
	if _, err := os.Lstat(final); err == nil {
		os.RemoveAll(final) // a half-committed folder from an interrupted earlier attempt; the tx is authoritative
	}
	if err := os.MkdirAll(final, 0o700); err != nil {
		return
	}
	if err := os.Rename(filepath.Join(dir, "package"), filepath.Join(final, "package")); err != nil {
		return
	}
	if err := os.Rename(filepath.Join(dir, "credential.json"), a.credPath(in.ID)); err != nil {
		return
	}
	for _, d := range []string{"data", "runtime", "logs"} {
		os.MkdirAll(filepath.Join(final, d), 0o700)
	}
	m := p.Manifest
	mode := m.Runtime.DefaultStartup
	if mode == "" {
		mode = supervisor.ModeManual
	}
	idle := m.Runtime.DefaultIdle
	if idle <= 0 {
		idle = 30
	}
	rec := InstallRecord{InstallationID: in.ID, FeatureID: m.FeatureID, Name: m.Name, Version: m.Version, Description: m.Description,
		PublisherClaim: m.Publisher.Name, ArtifactSHA256: p.ArtifactSHA256, ManifestSHA256: p.ManifestSHA256, Files: m.Files,
		Command: m.Runtime.Command, Requires: m.Runtime.Requires, Capabilities: m.Capabilities, StartupModes: m.Runtime.StartupModes,
		SourceDir: store.DisplayPath(t.SourceDir), SourceRepo: m.Source.Repo + "/" + m.Source.Path, TrustClass: m.TrustClass,
		IdentitySystemID: cred.IdentitySystemID, InstalledAt: time.Now().UTC(), State: "INSTALLED",
		Policy: supervisor.Policy{Mode: mode, Idle: idle, Recovery: supervisor.RecoveryManual}}
	if err := a.saveRecord(rec); err != nil {
		return
	}
	if _, err := idclient.ConfirmInstallation(ctx, a.Conn.Pairing().BaseURL, cred); err != nil {
		a.note("warn", "install.confirm_pending", in.ID+": files are in place but Identity has not confirmed yet ("+err.Error()+"); will retry")
		return
	}
	os.RemoveAll(dir)
	a.note("info", "install.completed", "installed "+m.Name+" "+m.Version+" as "+in.ID+" — ZERO grants, ZERO information sources; startup "+mode)
	a.Conn.Kick()
}

func (a *App) cancelTx(ctx context.Context, key string) error {
	if !txKeyRE.MatchString(key) {
		return errors.New("unknown install")
	}
	var t Tx
	b, err := os.ReadFile(filepath.Join(a.txDir(), key, "tx.json"))
	if err != nil || json.Unmarshal(b, &t) != nil {
		return errors.New("unknown install")
	}
	if t.State == "REGISTERED" && t.InstallationID != "" {
		cl, err := a.Conn.Require(ctx)
		if err != nil {
			return err
		}
		if err := cl.AbandonInstall(ctx, t.InstallationID); err != nil {
			return err
		}
	}
	os.RemoveAll(filepath.Join(a.txDir(), key))
	a.note("info", "install.cancelled", "install of "+t.FeatureID+" cancelled; nothing installed")
	return nil
}

// ---- runtime spec ----

func (a *App) specFor(id string) (supervisor.Spec, error) {
	r, err := a.loadRecord(id)
	if err != nil {
		return supervisor.Spec{}, err
	}
	if r.State != "INSTALLED" {
		return supervisor.Spec{}, errors.New("not installed")
	}
	if r.TrustClass != featpkg.TrustFixture {
		return supervisor.Spec{}, errors.New("UNTRUSTED_EXECUTION_BLOCKED")
	}
	if iv := a.Conn.installation(id); iv != nil && iv.Installation.State != "INSTALLED" {
		return supervisor.Spec{}, errors.New("Identity lists this installation as " + iv.Installation.State + "; it can't be started")
	}
	pkg := filepath.Join(a.recDir(id), "package")
	if err := featpkg.VerifyInstalled(pkg, r.ManifestSHA256, r.Files); err != nil {
		return supervisor.Spec{}, err
	}
	pr := a.Conn.Pairing()
	if pr.IdentitySystemID != r.IdentitySystemID {
		return supervisor.Spec{}, errors.New("this installation belongs to a different Identity installation than the one Processor is connected to")
	}
	return supervisor.Spec{InstallationID: id, FeatureID: r.FeatureID, PackageDir: pkg, DataDir: filepath.Join(a.recDir(id), "data"),
		RuntimeDir: filepath.Join(a.recDir(id), "runtime"), LogPath: filepath.Join(a.recDir(id), "logs", "feature.log"),
		CredentialPath: a.credPath(id), Command: r.Command, IdentityURL: pr.BaseURL, IdentitySystemID: pr.IdentitySystemID}, nil
}

// ---- synthetic protected action ----

func (a *App) runAction(ctx context.Context, id, subject, capability, input string) (res ActionResult) {
	res = ActionResult{RequestID: "req_" + store.RandHex(8), InstallationID: id, Capability: capability, Subject: subject, At: time.Now().UTC()}
	t0 := time.Now()
	defer func() {
		res.LatencyMS = time.Since(t0).Milliseconds()
		a.P1.mu.Lock()
		a.P1.requests = append(a.P1.requests, res)
		if len(a.P1.requests) > 200 {
			a.P1.requests = a.P1.requests[len(a.P1.requests)-200:]
		}
		a.P1.mu.Unlock()
		lr := res
		lr.Observation = nil // the log keeps the outcome, not the data
		if b, err := json.Marshal(lr); err == nil {
			if f, err := os.OpenFile(a.requestLogPath(id), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o600); err == nil {
				f.Write(append(b, '\n'))
				f.Close()
			}
		}
		a.Log.Add(map[bool]string{true: "info", false: "warn"}[res.Outcome == "COMPLETED"], "feature", "action."+strings.ToLower(res.Outcome), id+" "+res.RequestID+": "+res.Detail)
	}()
	r, err := a.loadRecord(id)
	if err != nil || r.State != "INSTALLED" {
		res.Outcome, res.Detail = "ERROR", "not an installed feature"
		return res
	}
	body, _ := json.Marshal(map[string]string{"request_id": res.RequestID, "input": input})
	cl, err := a.Conn.Require(ctx)
	if err != nil {
		res.Outcome, res.Detail = "AUTHORITY_UNAVAILABLE", "Identity can't be reached, so nothing was started or sent"
		return res
	}
	tk, err := cl.RequestTicket(ctx, id, subject, capability, body)
	res.DecisionID, res.TicketID = tk.DecisionID, tk.TicketID
	if err != nil || tk.Decision != "ALLOW_TICKET" || tk.Ticket == "" {
		if tk.Decision == "DENIED" {
			res.Outcome, res.Detail = "DENIED_BY_IDENTITY", "Identity: "+tk.Reason+" — nothing was started or sent"
		} else {
			res.Outcome, res.Detail = "AUTHORITY_UNAVAILABLE", "no decision from Identity ("+errString(err)+") — nothing was started or sent"
		}
		return res
	}
	if !a.Sup.Running(id) {
		if r.Policy.Mode != supervisor.ModeOnDemand {
			res.Outcome, res.Detail = "NOT_RUNNING", "the feature is stopped and its startup mode is "+r.Policy.Mode+"; start it first (the unused ticket expires by itself)"
			return res
		}
		if err := a.Sup.Start(ctx, id, "on_demand for "+res.RequestID); err != nil {
			res.Outcome, res.Detail = "NOT_RUNNING", "on-demand start failed: "+err.Error()
			return res
		}
		res.StartedFeature = true
	}
	u := a.Sup.URL(id)
	a.Sup.Begin(id)
	defer a.Sup.End(id)
	req, _ := http.NewRequestWithContext(ctx, "POST", u+"/v1/actions/"+capability, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-TF-Ticket", tk.Ticket)
	nonce := supervisor.ActionNonce()
	req.Header.Set("X-TF-Nonce", nonce)
	resp, err := (&http.Client{Timeout: 30 * time.Second, Transport: &http.Transport{Proxy: nil}}).Do(req)
	if err != nil {
		res.Outcome, res.Detail = "REQUEST_UNCERTAIN", "the connection to the feature broke before an answer ("+err.Error()+"). It is NOT resent automatically; it may or may not have been processed."
		return res
	}
	defer resp.Body.Close()
	raw, rerr := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if rerr != nil {
		res.Outcome, res.Detail = "REQUEST_UNCERTAIN", "the answer was cut off ("+rerr.Error()+"); NOT resent"
		return res
	}
	if err := a.Sup.VerifyAnswer(id, nonce, resp.StatusCode, raw, resp.Header); err != nil {
		res.Outcome, res.Detail = "REQUEST_UNCERTAIN", err.Error()+" — the result is not trusted and NOT resent"
		return res
	}
	var m map[string]any
	json.Unmarshal(raw, &m)
	str := func(k string) string { s, _ := m[k].(string); return s }
	switch resp.StatusCode {
	case 200:
		res.Outcome, res.Detail = "COMPLETED", "the feature verified the ticket, Identity allowed it live, and it returned SYNTHETIC TEST DATA"
		res.Observation, _ = m["observation"].(map[string]any)
		if res.Observation != nil {
			res.DecisionID, _ = res.Observation["decision_id"].(string)
		}
	case 403:
		res.Outcome, res.Detail = "DENIED_BY_FEATURE", "the feature asked Identity live and was told: "+str("reason")+" — nothing was done"
		if d := str("decision_id"); d != "" {
			res.DecisionID = d
		}
	case 503:
		res.Outcome, res.Detail = "AUTHORITY_UNAVAILABLE", "the feature could not get a decision from Identity: "+str("detail")
	case 401, 409:
		res.Outcome, res.Detail = "TICKET_REJECTED", "the feature refused the ticket: "+str("error")+" "+str("reason")
	default:
		res.Outcome, res.Detail = "ERROR", fmt.Sprintf("HTTP %d %s", resp.StatusCode, str("error"))
	}
	return res
}

func errString(err error) string {
	if err == nil {
		return "no ticket"
	}
	return err.Error()
}

// ---- reconcile loop: advance installs, commit approved operations ----

func (a *App) reconcileLoop(ctx context.Context) {
	t := time.NewTicker(1500 * time.Millisecond)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
		a.reconcileOnce(ctx)
	}
}

func (a *App) reconcileOnce(ctx context.Context) {
	a.P1.reconcile.Lock()
	defer a.P1.reconcile.Unlock()
	if a.Conn.State() != CVerified {
		return
	}
	cl, err := a.Conn.Require(ctx)
	if err != nil {
		return
	}
	for _, t := range a.txs() {
		a.advanceTx(ctx, cl, t)
	}
	// Resume a confirm that failed after the folder was committed.
	for _, r := range a.records() {
		if iv := a.Conn.installation(r.InstallationID); r.State == "INSTALLED" && iv != nil && iv.Installation.State == "PENDING_CONFIRM" {
			var cred idclient.Credential
			if b, err := os.ReadFile(a.credPath(r.InstallationID)); err == nil && json.Unmarshal(b, &cred) == nil {
				if _, err := idclient.ConfirmInstallation(ctx, a.Conn.Pairing().BaseURL, cred); err == nil {
					a.note("info", "install.confirmed", r.InstallationID+" confirmed with Identity")
					a.Conn.Kick()
				}
			}
		}
	}
	a.P1.mu.Lock()
	ops := append([]PendingOp(nil), a.P1.pending...)
	a.P1.mu.Unlock()
	changed := false
	keep := []PendingOp{}
	for _, op := range ops {
		ap, err := cl.Approval(ctx, op.ApprovalID)
		if err != nil {
			if idclient.KindOf(err) == idclient.Refused {
				changed = true // Identity forgot it (e.g. profile replaced)
				continue
			}
			keep = append(keep, op)
			continue
		}
		switch ap.State {
		case "PENDING":
			keep = append(keep, op)
			continue
		case "DENIED", "EXPIRED":
			a.note("warn", "approval."+strings.ToLower(ap.State), op.Kind+" "+op.ApprovalID+": "+strings.ToLower(ap.State)+" — nothing changed")
			changed = true
			continue
		}
		changed = true
		switch op.Kind {
		case "enroll_admin":
			a.note("info", "admin.enrolled", "the Admin Identity was established in Identity ("+ap.Result+")")
		case "grant_capability":
			g, err := cl.CommitGrant(ctx, op.ApprovalID)
			if err != nil {
				a.note("warn", "grant.commit_failed", op.InstallationID+": "+err.Error())
			} else {
				a.note("info", "grant.created", fmt.Sprintf("grant %s: %s may ask %s for %s (subject %s)", g.Grant.ID, g.Grant.Requester, g.Grant.Destination, g.Grant.Capability, g.Grant.Subject))
			}
		case "uninstall_feature":
			a.finishUninstall(ctx, cl, op)
		}
	}
	if changed {
		a.P1.mu.Lock()
		a.P1.pending = keep
		a.P1.mu.Unlock()
		a.savePending()
		a.Conn.Kick()
	}
	// Always-running features start once Identity is confirmed (not after the person stopped them).
	for _, r := range a.records() {
		if r.State != "INSTALLED" || r.Policy.Mode != supervisor.ModeAlways {
			continue
		}
		a.P1.mu.Lock()
		stopped := a.P1.userStop[r.InstallationID]
		a.P1.mu.Unlock()
		if v := a.Sup.View(r.InstallationID); !stopped && (v.State == supervisor.Stopped) {
			go a.Sup.Start(context.Background(), r.InstallationID, "always_running")
		}
	}
}

func (a *App) finishUninstall(ctx context.Context, cl *idclient.Client, op PendingOp) {
	id := op.InstallationID
	a.P1.mu.Lock()
	a.P1.userStop[id] = true
	a.P1.mu.Unlock()
	a.Sup.Stop(id, "uninstall approved")
	if err := cl.CommitUninstall(ctx, id, op.ApprovalID); err != nil {
		a.note("warn", "uninstall.failed", id+": "+err.Error()+" — files kept")
		return
	}
	r, err := a.loadRecord(id)
	if err == nil {
		os.RemoveAll(filepath.Join(a.recDir(id), "package"))
		os.Remove(a.credPath(id))
		os.RemoveAll(filepath.Join(a.recDir(id), "runtime"))
		if !op.KeepData {
			os.RemoveAll(filepath.Join(a.recDir(id), "data"))
		}
		t := time.Now().UTC()
		r.State, r.UninstalledAt, r.DataKept = "UNINSTALLED", &t, op.KeepData
		a.saveRecord(r)
	}
	a.note("info", "uninstall.done", id+" uninstalled: grants revoked, credential removed, tombstone kept"+map[bool]string{true: ", data kept", false: ", data deleted"}[op.KeepData])
}

// ---- state for the UI ----

type InstView struct {
	Record   InstallRecord              `json:"record"`
	Identity *idclient.InstallationView `json:"identity,omitempty"`
	Runtime  supervisor.View            `json:"runtime"`
	Files    string                     `json:"files"`
	Requests []ActionResult             `json:"requests"`
	Pending  []PendingOp                `json:"pending"`
	DataDir  string                     `json:"data_dir"`
}

func (a *App) p1State() map[string]any {
	out := map[string]any{}
	a.P1.mu.Lock()
	pending := append([]PendingOp(nil), a.P1.pending...)
	reqs := append([]ActionResult(nil), a.P1.requests...)
	notes := append([]string(nil), a.P1.notes...)
	a.P1.mu.Unlock()
	views := []InstView{}
	for _, r := range a.records() {
		v := InstView{Record: r, Runtime: a.Sup.View(r.InstallationID), Identity: a.Conn.installation(r.InstallationID),
			Requests: []ActionResult{}, Pending: []PendingOp{}, DataDir: store.DisplayPath(filepath.Join(a.recDir(r.InstallationID), "data"))}
		if r.State == "INSTALLED" {
			if err := featpkg.VerifyInstalled(filepath.Join(a.recDir(r.InstallationID), "package"), r.ManifestSHA256, r.Files); err != nil {
				v.Files = err.Error()
			} else {
				v.Files = "OK"
			}
		}
		for i := len(reqs) - 1; i >= 0 && len(v.Requests) < 20; i-- {
			if reqs[i].InstallationID == r.InstallationID {
				v.Requests = append(v.Requests, reqs[i])
			}
		}
		for _, op := range pending {
			if op.InstallationID == r.InstallationID {
				v.Pending = append(v.Pending, op)
			}
		}
		views = append(views, v)
	}
	enroll := []PendingOp{}
	for _, op := range pending {
		if op.Kind == "enroll_admin" {
			enroll = append(enroll, op)
		}
	}
	out["installations"] = views
	out["library"] = a.library()
	out["install_txs"] = a.txs()
	out["admin_pending"] = enroll
	out["admin"] = a.Conn.adminView()
	out["identity_installations"] = a.Conn.installations()
	out["trusted_dirs"] = func() []string {
		s := []string{}
		for _, d := range a.P1.trusted {
			s = append(s, store.DisplayPath(d))
		}
		return s
	}()
	out["p1_notes"] = notes
	return out
}

// ---- HTTP routes ----

func (a *App) p1Routes(m *http.ServeMux) {
	ins := func(h func(w http.ResponseWriter, r *http.Request, id string)) http.HandlerFunc {
		return a.wrap(mutate, func(w http.ResponseWriter, r *http.Request) {
			id := r.PathValue("id")
			if !insIDRE.MatchString(id) {
				a.fail(w, 400, "unknown installation")
				return
			}
			if _, err := a.loadRecord(id); err != nil {
				a.fail(w, 404, "unknown installation")
				return
			}
			h(w, r, id)
		})
	}
	m.HandleFunc("POST /api/theme", a.wrap(mutate, func(w http.ResponseWriter, r *http.Request) {
		var b struct{ Theme string }
		decode(r, &b)
		if b.Theme != "system" && b.Theme != "light" && b.Theme != "dark" {
			a.fail(w, 400, "theme must be system, light or dark")
			return
		}
		a.Prof.UpdateSettings(func(s *store.Settings) { s.Theme = b.Theme })
		writeJSON(w, 200, a.State())
	}))
	m.HandleFunc("POST /api/admin/enroll", a.wrap(mutate, func(w http.ResponseWriter, r *http.Request) {
		cl, err := a.Conn.Require(r.Context())
		if err != nil {
			a.failIdentity(w, err)
			return
		}
		ar, err := cl.RequestAdminEnrollment(r.Context())
		if err != nil {
			a.failIdentity(w, err)
			return
		}
		a.addPending(PendingOp{ApprovalID: ar.Approval.ID, Kind: "enroll_admin", ApprovalURL: ar.ApprovalURL, Summary: ar.Approval.Summary,
			CreatedAt: ar.Approval.CreatedAt, ExpiresAt: ar.Approval.ExpiresAt, State: "PENDING"})
		a.note("info", "admin.enroll_requested", "asked Identity to establish the Admin Identity; finish it in Identity's own page ("+ar.Approval.ID+")")
		writeJSON(w, 200, map[string]any{"approval_url": ar.ApprovalURL, "approval_id": ar.Approval.ID, "state": a.State()})
	}))
	m.HandleFunc("POST /api/library/{feature}/install", a.wrap(mutate, func(w http.ResponseWriter, r *http.Request) {
		t, err := a.requestInstall(r.Context(), r.PathValue("feature"))
		if err != nil {
			if idclient.KindOf(err) != "" {
				a.failIdentity(w, err)
				return
			}
			a.fail(w, 409, err.Error())
			return
		}
		writeJSON(w, 200, map[string]any{"approval_url": t.ApprovalURL, "approval_id": t.ApprovalID, "install_key": t.Key, "state": a.State()})
	}))
	m.HandleFunc("POST /api/install-tx/{key}/cancel", a.wrap(mutate, func(w http.ResponseWriter, r *http.Request) {
		if err := a.cancelTx(r.Context(), r.PathValue("key")); err != nil {
			a.failIdentity(w, err)
			return
		}
		writeJSON(w, 200, a.State())
	}))
	m.HandleFunc("POST /api/installations/{id}/start", ins(func(w http.ResponseWriter, r *http.Request, id string) {
		a.P1.mu.Lock()
		a.P1.userStop[id] = false
		a.P1.mu.Unlock()
		if err := a.Sup.Start(r.Context(), id, "manual start by the person"); err != nil {
			a.fail(w, 409, err.Error())
			return
		}
		a.Sup.ArmIdle(id)
		writeJSON(w, 200, a.State())
	}))
	m.HandleFunc("POST /api/installations/{id}/stop", ins(func(w http.ResponseWriter, r *http.Request, id string) {
		a.P1.mu.Lock()
		a.P1.userStop[id] = true
		a.P1.mu.Unlock()
		a.Sup.Stop(id, "stopped by the person")
		writeJSON(w, 200, a.State())
	}))
	m.HandleFunc("POST /api/installations/{id}/self-test", ins(func(w http.ResponseWriter, r *http.Request, id string) {
		st, raw, err := a.Sup.Call(id, "POST", "/v1/self-test", []byte("{}"))
		if err != nil || st != 200 {
			a.fail(w, 409, "self-test needs the feature RUNNING ("+errString(err)+")")
			return
		}
		var res map[string]any
		json.Unmarshal(raw, &res)
		a.note("info", "feature.self_test", id+": standalone self-test "+fmt.Sprint(res["result"])+" (TEST DATA, no grant used)")
		writeJSON(w, 200, map[string]any{"self_test": res, "state": a.State()})
	}))
	m.HandleFunc("POST /api/installations/{id}/policy", ins(func(w http.ResponseWriter, r *http.Request, id string) {
		var b supervisor.Policy
		decode(r, &b)
		rec, _ := a.loadRecord(id)
		okMode := false
		for _, m := range append(rec.StartupModes, supervisor.ModeManual) {
			if m == b.Mode {
				okMode = true
			}
		}
		if !okMode || (b.Recovery != supervisor.RecoveryManual && b.Recovery != supervisor.RecoveryBounded) || b.Idle < 5 || b.Idle > 3600 {
			a.fail(w, 400, "startup mode must be one the feature supports; recovery manual or bounded; idle 5–3600 s")
			return
		}
		rec.Policy = b
		if err := a.saveRecord(rec); err != nil {
			a.fail(w, 500, err.Error())
			return
		}
		a.P1.mu.Lock()
		a.P1.userStop[id] = false
		a.P1.mu.Unlock()
		a.note("info", "feature.policy", fmt.Sprintf("%s: startup %s, idle %ds, recovery %s — this changes when it runs, never what it may access", id, b.Mode, b.Idle, b.Recovery))
		a.Sup.ArmIdle(id)
		writeJSON(w, 200, a.State())
	}))
	m.HandleFunc("POST /api/installations/{id}/grants/request", ins(func(w http.ResponseWriter, r *http.Request, id string) {
		var b struct {
			Subject    string `json:"subject"`
			Capability string `json:"capability"`
			ExpiresIn  int64  `json:"expires_in_seconds"`
		}
		decode(r, &b)
		if !entIDRE.MatchString(b.Subject) || !capIDRE.MatchString(b.Capability) {
			a.fail(w, 400, "choose an identity and a capability")
			return
		}
		cl, err := a.Conn.Require(r.Context())
		if err != nil {
			a.failIdentity(w, err)
			return
		}
		ar, err := cl.RequestGrant(r.Context(), id, b.Subject, b.Capability, b.ExpiresIn)
		if err != nil {
			a.failIdentity(w, err)
			return
		}
		a.addPending(PendingOp{ApprovalID: ar.Approval.ID, Kind: "grant_capability", InstallationID: id, ApprovalURL: ar.ApprovalURL,
			Summary: ar.Approval.Summary, CreatedAt: ar.Approval.CreatedAt, ExpiresAt: ar.Approval.ExpiresAt, State: "PENDING"})
		a.note("info", "grant.requested", id+": connection request "+ar.Approval.ID+" waiting for the Admin in Identity")
		writeJSON(w, 200, map[string]any{"approval_url": ar.ApprovalURL, "approval_id": ar.Approval.ID, "state": a.State()})
	}))
	m.HandleFunc("POST /api/installations/{id}/grants/{gid}/revoke", ins(func(w http.ResponseWriter, r *http.Request, id string) {
		gid := r.PathValue("gid")
		if !grantRE.MatchString(gid) {
			a.fail(w, 400, "unknown grant")
			return
		}
		cl, err := a.Conn.Require(r.Context())
		if err != nil {
			a.failIdentity(w, err)
			return
		}
		if err := cl.RevokeGrant(r.Context(), gid, "revoked by the person in Thought Processor"); err != nil {
			a.failIdentity(w, err)
			return
		}
		a.note("info", "grant.revoked", id+": grant "+gid+" revoked; the next request is denied")
		a.Conn.Kick()
		a.Conn.refreshData(r.Context())
		writeJSON(w, 200, a.State())
	}))
	m.HandleFunc("POST /api/installations/{id}/action", ins(func(w http.ResponseWriter, r *http.Request, id string) {
		var b struct {
			Subject    string `json:"subject"`
			Capability string `json:"capability"`
			Input      string `json:"input"`
		}
		decode(r, &b)
		if !entIDRE.MatchString(b.Subject) || !capIDRE.MatchString(b.Capability) || len(b.Input) > 1000 {
			a.fail(w, 400, "choose an identity, a capability and up to 1000 characters of test input")
			return
		}
		res := a.runAction(r.Context(), id, b.Subject, b.Capability, b.Input)
		writeJSON(w, 200, map[string]any{"result": res, "state": a.State()})
	}))
	m.HandleFunc("POST /api/installations/{id}/uninstall", ins(func(w http.ResponseWriter, r *http.Request, id string) {
		var b struct {
			KeepData bool `json:"keep_data"`
		}
		decode(r, &b)
		cl, err := a.Conn.Require(r.Context())
		if err != nil {
			a.failIdentity(w, err)
			return
		}
		ar, err := cl.RequestUninstall(r.Context(), id)
		if err != nil {
			a.failIdentity(w, err)
			return
		}
		a.addPending(PendingOp{ApprovalID: ar.Approval.ID, Kind: "uninstall_feature", InstallationID: id, KeepData: b.KeepData,
			ApprovalURL: ar.ApprovalURL, Summary: ar.Approval.Summary, CreatedAt: ar.Approval.CreatedAt, ExpiresAt: ar.Approval.ExpiresAt, State: "PENDING"})
		writeJSON(w, 200, map[string]any{"approval_url": ar.ApprovalURL, "approval_id": ar.Approval.ID, "state": a.State()})
	}))
	m.HandleFunc("GET /api/installations/{id}/details", a.wrap(private, func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if !insIDRE.MatchString(id) {
			a.fail(w, 400, "unknown installation")
			return
		}
		out := map[string]any{"installation_id": id}
		out["log_tail"] = tailFile(filepath.Join(a.recDir(id), "logs", "feature.log"), 120)
		if st, raw, err := a.Sup.Call(id, "GET", "/v1/connections", nil); err == nil && st == 200 {
			var c map[string]any
			json.Unmarshal(raw, &c)
			out["feature_view_of_connections"] = c
		} else {
			out["feature_view_of_connections"] = map[string]string{"state": "NOT_RUNNING", "detail": "start the feature to ask it what Identity tells it"}
		}
		if cl, err := a.Conn.Require(r.Context()); err == nil {
			if d, err := cl.Decisions(r.Context(), id); err == nil {
				out["identity_decisions"] = d
			}
		} else {
			out["identity_decisions_error"] = "AUTHORITY_UNAVAILABLE"
		}
		writeJSON(w, 200, out)
	}))
}

func tailFile(path string, n int) []string {
	b, err := os.ReadFile(path)
	if err != nil {
		return []string{}
	}
	if len(b) > 256<<10 {
		b = b[len(b)-256<<10:]
	}
	lines := strings.Split(strings.TrimRight(string(b), "\n"), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return lines
}
