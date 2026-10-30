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
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
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
	"github.com/Arameair/Thought-Processor/processor/internal/update"
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
	TrustClass       string               `json:"trust_class"` // what the manifest says about itself (descriptive only)
	TrustBasis       string               `json:"trust_basis"` // why Processor trusts it: the pinned artifact
	Interpreter      string               `json:"interpreter,omitempty"`
	IdentitySystemID string               `json:"identity_system_id"`
	InstalledAt      time.Time            `json:"installed_at"`
	State            string               `json:"state"` // INSTALLED | UNINSTALLED
	UninstalledAt    *time.Time           `json:"uninstalled_at,omitempty"`
	DataKept         bool                 `json:"data_kept,omitempty"`
	Policy           supervisor.Policy    `json:"policy"`
	UpdatedAt        *time.Time           `json:"updated_at,omitempty"` // last in-place update (thought.feature-update/1)
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
	Kind           string    `json:"kind"` // enroll_admin | grant_capability | uninstall_feature | revoke_grant
	InstallationID string    `json:"installation_id,omitempty"`
	KeepData       bool      `json:"keep_data,omitempty"`
	GrantID        string    `json:"grant_id,omitempty"` // revoke_grant
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
	pins      featpkg.Pins
	testFault bool
	pending   []PendingOp
	requests  []ActionResult
	userStop  map[string]bool
	autoDone  bool
	notes     []string
	busy      map[string]bool
	reconcile sync.Mutex
	// installMu serializes install requests, so two clicks (or two tabs)
	// can't both decide "nothing installed yet" and start two installs.
	installMu sync.Mutex
	selfTests map[string]SelfTestResult
	// updates: one transaction per installation; updStart lets ONLY the
	// update engine start an installation while it is being updated.
	updMu    sync.Mutex
	applying map[string]bool
	updStart map[string]bool
}

// SelfTestResult is the last standalone self-test Processor ran for one
// installation, exactly as the running feature answered it (launcher-signed).
type SelfTestResult struct {
	At            time.Time      `json:"at"`
	Result        string         `json:"result"` // PASS | FAIL | ERROR
	LaunchID      string         `json:"launch_id,omitempty"`
	DataClass     string         `json:"data_class,omitempty"`
	Input         string         `json:"input,omitempty"`
	Output        map[string]any `json:"output,omitempty"`
	ExternalCalls any            `json:"external_calls,omitempty"`
	GrantsUsed    string         `json:"grants_used,omitempty"`
	Error         string         `json:"error,omitempty"`
}

// InstallOutcome says what an install request actually did.
type InstallOutcome struct {
	Tx                Tx     // set when a new or in-progress install needs the Admin
	ExistingID        string // set when this exact artifact is already installed here
	AlreadyInProgress bool   // an install of this exact artifact was already waiting
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
// TrustArtifacts sets the pinned artifact digests (built-in + operator).
func (a *App) TrustArtifacts(operator []string) error {
	p, err := featpkg.NewPins(operator)
	if err != nil {
		return err
	}
	a.P1.pins = p
	return nil
}

// EnableTestFaults lets the trusted test fixture honour its TEST-ONLY
// fault-injection file (TF_FIXTURE_TEST_MODE=1). Off unless the operator
// starts Processor with -test-feature-faults; never grants anything.
func (a *App) EnableTestFaults() { a.P1.testFault = true }

func (a *App) TrustFeatureDirs(dirs []string) {
	for _, d := range dirs {
		if abs, err := filepath.Abs(d); err == nil {
			a.P1.trusted = append(a.P1.trusted, filepath.Clean(abs))
		}
	}
}

func (a *App) initP1() {
	a.P1 = &p1{userStop: map[string]bool{}, busy: map[string]bool{}, selfTests: map[string]SelfTestResult{}, applying: map[string]bool{}, updStart: map[string]bool{}}
	a.P1.pins, _ = featpkg.NewPins(nil)
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
	TrustBasis     string               `json:"trust_basis,omitempty"`
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
		p, err := featpkg.Load(dir, true, a.P1.pins)
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
		it.TrustBasis = p.TrustBasis
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
				if p, err := featpkg.Load(cand, true, a.P1.pins); err == nil && p.Manifest.FeatureID == featureID {
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

// requestInstall is idempotent by default: the same pinned artifact installed
// into the same workspace (this Processor profile, paired with this Identity)
// returns the existing installation, and an install of it that is already
// waiting for the Admin is returned instead of starting a second one. Only an
// explicit newCopy asks Identity for another installation identity. Nothing
// existing is ever deleted here.
func (a *App) requestInstall(ctx context.Context, featureID string, newCopy bool) (InstallOutcome, error) {
	if !featpkg.FeatureIDRE.MatchString(featureID) {
		return InstallOutcome{}, errors.New("unknown feature")
	}
	cl, err := a.Conn.Require(ctx)
	if err != nil {
		return InstallOutcome{}, err
	}
	a.P1.installMu.Lock()
	defer a.P1.installMu.Unlock()
	p, err := a.findPackage(featureID)
	if err != nil {
		return InstallOutcome{}, err
	}
	if !newCopy {
		// Ask Identity live, not a cached list: only an installation Identity
		// still holds for this exact artifact counts as "already installed".
		live, err := cl.Installations(ctx)
		if err != nil {
			return InstallOutcome{}, err
		}
		sys := a.Conn.Pairing().IdentitySystemID
		for _, r := range a.records() {
			if r.State != "INSTALLED" || r.FeatureID != featureID || r.ArtifactSHA256 != p.ArtifactSHA256 || r.IdentitySystemID != sys {
				continue
			}
			for _, iv := range live {
				in := iv.Installation
				if in.ID == r.InstallationID && in.ArtifactSHA256 == p.ArtifactSHA256 && (in.State == "INSTALLED" || in.State == "PENDING_CONFIRM") {
					a.note("info", "install.already_installed", featureID+" "+p.Manifest.Version+" (artifact "+p.ArtifactSHA256[:12]+"…) is already installed here as "+r.InstallationID+"; returned it, nothing new was registered")
					return InstallOutcome{ExistingID: r.InstallationID}, nil
				}
			}
		}
		for _, t := range a.txs() {
			if t.FeatureID == featureID && t.ArtifactSHA256 == p.ArtifactSHA256 && t.State != "FAILED" {
				a.note("info", "install.already_in_progress", featureID+": an install of this exact artifact is already waiting ("+t.State+"); returned it instead of starting another")
				return InstallOutcome{Tx: t, AlreadyInProgress: true}, nil
			}
		}
	}
	t, err := a.newInstallTx(ctx, cl, p, featureID)
	return InstallOutcome{Tx: t}, err
}

func (a *App) newInstallTx(ctx context.Context, cl *idclient.Client, p featpkg.Package, featureID string) (Tx, error) {
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
	p, err := featpkg.Load(filepath.Join(dir, "package"), true, a.P1.pins)
	if err != nil || p.ArtifactSHA256 != t.ArtifactSHA256 {
		fail("staged files changed; refused")
		return
	}
	if p.Blocked != "" {
		fail(p.Blocked)
		return
	}
	interp, ierr := pinInterpreter(p.Manifest.Runtime.Command)
	if ierr != nil {
		fail(ierr.Error())
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
		TrustBasis: p.TrustBasis, Interpreter: interp,
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
	// While an update transaction owns this installation, only the update
	// engine may start it (for its own verification).
	if t, ok := a.loadUpdTx(id); ok && !update.Terminal(t.State) && t.Phase != phWaiting {
		a.P1.mu.Lock()
		allowed := a.P1.updStart[id]
		a.P1.mu.Unlock()
		if !allowed {
			return supervisor.Spec{}, errors.New("UPDATING: this installation is being updated (" + t.State + "); it can't be started until the update finishes")
		}
	}
	// Verified again before every start: still pinned, files still identical.
	if a.P1.pins.Basis(r.ArtifactSHA256) == "" {
		return supervisor.Spec{}, errors.New("UNTRUSTED_EXECUTION_BLOCKED: this installation's artifact " + r.ArtifactSHA256[:12] + "… is not pinned in this Processor; it will not be started")
	}
	// Fail closed: Identity (not Processor) must confirm, from its last signed
	// answer on a verified connection, that this exact installation and
	// artifact are INSTALLED. Unreachable or unconfirmed Identity => no start.
	iv := a.Conn.installation(id)
	switch {
	case iv == nil:
		return supervisor.Spec{}, errors.New("AUTHORITY_UNAVAILABLE: Identity can't confirm this installation right now, so it was not started; start it again once Identity is connected")
	case iv.Installation.State != "INSTALLED":
		return supervisor.Spec{}, errors.New("Identity lists this installation as " + iv.Installation.State + "; it can't be started")
	case iv.Installation.ArtifactSHA256 != r.ArtifactSHA256:
		return supervisor.Spec{}, errors.New("UNTRUSTED_EXECUTION_BLOCKED: Identity approved a different artifact for this installation than the one installed here; it will not be started")
	}
	pkg := filepath.Join(a.recDir(id), "package")
	if err := featpkg.VerifyInstalled(pkg, r.ManifestSHA256, r.Files); err != nil {
		return supervisor.Spec{}, err
	}
	pr := a.Conn.Pairing()
	if pr.IdentitySystemID != r.IdentitySystemID {
		return supervisor.Spec{}, errors.New("this installation belongs to a different Identity installation than the one Processor is connected to")
	}
	cmd := append([]string(nil), r.Command...)
	if r.Interpreter != "" {
		// Use the interpreter resolved and recorded at install time, never a
		// fresh PATH lookup (a later PATH shim can't slip in).
		if fi, err := os.Stat(r.Interpreter); err != nil || fi.IsDir() {
			return supervisor.Spec{}, errors.New("REQUIREMENT_MISSING: the interpreter recorded at install (" + r.Interpreter + ") is gone; reinstall the feature")
		}
		cmd[0] = r.Interpreter
	}
	return supervisor.Spec{InstallationID: id, FeatureID: r.FeatureID, PackageDir: pkg, DataDir: filepath.Join(a.recDir(id), "data"),
		RuntimeDir: filepath.Join(a.recDir(id), "runtime"), LogPath: filepath.Join(a.recDir(id), "logs", "feature.log"),
		CredentialPath: a.credPath(id), Command: cmd, IdentityURL: pr.BaseURL, IdentitySystemID: pr.IdentitySystemID, TestMode: a.P1.testFault}, nil
}

// ---- synthetic protected action ----

// Attachment is one image a person sends with a request (e.g. "what is this
// picture of?"). Processor checks it, passes it through to the feature inside
// the ticketed request body (Identity sees only the request's SHA-256), and
// keeps no copy of it.
type Attachment struct {
	Name string `json:"name"`
	Mime string `json:"mime"`
	Data string `json:"base64"`
}

var attNameRE = regexp.MustCompile(`^[A-Za-z0-9_.() -]{1,128}$`)

const maxAttachment = 4 << 20

func checkAttachment(at *Attachment) error {
	if at == nil {
		return nil
	}
	if !attNameRE.MatchString(at.Name) || strings.Contains(at.Name, "..") {
		return fmt.Errorf("the file name isn't allowed")
	}
	magic := map[string][]byte{"image/png": []byte("\x89PNG\r\n\x1a\n"), "image/jpeg": {0xff, 0xd8, 0xff}, "image/webp": []byte("RIFF")}
	m, ok := magic[at.Mime]
	if !ok {
		return fmt.Errorf("only PNG, JPEG or WebP images can be sent")
	}
	if len(at.Data) > maxAttachment*4/3+16 {
		return fmt.Errorf("the image is larger than 4 MB")
	}
	raw, err := base64.StdEncoding.DecodeString(at.Data)
	if err != nil || len(raw) == 0 || len(raw) > maxAttachment {
		return fmt.Errorf("the image couldn't be read, or is larger than 4 MB")
	}
	if !bytes.HasPrefix(raw, m) || (at.Mime == "image/webp" && (len(raw) < 12 || string(raw[8:12]) != "WEBP")) {
		return fmt.Errorf("the file isn't really a %s image", at.Mime)
	}
	return nil
}

func (a *App) runAction(ctx context.Context, id, subject, capability, input string, att ...*Attachment) (res ActionResult) {
	res = ActionResult{RequestID: "req_" + store.RandHex(8), InstallationID: id, Capability: capability, Subject: subject, At: time.Now().UTC()}
	t0 := time.Now()
	defer func() {
		res.LatencyMS = time.Since(t0).Milliseconds()
		// Processor keeps the outcome of a request, never what the feature
		// answered (for TASS that is a private conversation): the answer goes
		// back to the caller only, and neither this list nor the log holds it.
		lr := res
		lr.Observation = nil
		a.P1.mu.Lock()
		a.P1.requests = append(a.P1.requests, lr)
		if len(a.P1.requests) > 200 {
			a.P1.requests = a.P1.requests[len(a.P1.requests)-200:]
		}
		a.P1.mu.Unlock()
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
	reqBody := map[string]any{"request_id": res.RequestID, "input": input}
	if len(att) > 0 && att[0] != nil {
		reqBody["attachment"] = att[0]
	}
	body, _ := json.Marshal(reqBody)
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
		res.Outcome, res.Detail = "COMPLETED", "the service verified the ticket, Identity allowed it live, and it answered (data class shown with the result)"
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
		if d := str("detail"); d != "" {
			// e.g. a feature that was allowed by Identity but couldn't use the input
			res.Detail += " — " + d
		}
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
	a.advanceUpdates(ctx, cl)
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
		case "revoke_grant":
			if g, err := cl.CommitAdminRevoke(ctx, op.GrantID, op.ApprovalID); err != nil {
				a.note("warn", "grant.admin_revoke_failed", op.GrantID+": "+err.Error())
			} else {
				a.note("info", "grant.revoked", "grant "+g.Grant.ID+" revoked by "+g.Grant.RevokedBy)
			}
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
		if a.Conn.installation(r.InstallationID) == nil {
			continue // Identity hasn't confirmed it yet on this connection; try next round
		}
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
	SelfTest *SelfTestResult            `json:"self_test,omitempty"` // last standalone self-test (this Processor run)
	// SameArtifactCopies counts other INSTALLED installations of the same
	// artifact in this workspace (e.g. made before installs became idempotent).
	SameArtifactCopies int `json:"same_artifact_copies"`
}

func (a *App) p1State() map[string]any {
	out := map[string]any{"test_feature_faults": a.P1.testFault}
	a.P1.mu.Lock()
	pending := append([]PendingOp(nil), a.P1.pending...)
	reqs := append([]ActionResult(nil), a.P1.requests...)
	notes := append([]string(nil), a.P1.notes...)
	a.P1.mu.Unlock()
	a.P1.mu.Lock()
	selfTests := make(map[string]SelfTestResult, len(a.P1.selfTests))
	for k, v := range a.P1.selfTests {
		selfTests[k] = v
	}
	a.P1.mu.Unlock()
	views := []InstView{}
	recs := a.records()
	for _, r := range recs {
		v := InstView{Record: r, Runtime: a.Sup.View(r.InstallationID), Identity: a.Conn.installation(r.InstallationID),
			Requests: []ActionResult{}, Pending: []PendingOp{}, DataDir: store.DisplayPath(filepath.Join(a.recDir(r.InstallationID), "data"))}
		if st, ok := selfTests[r.InstallationID]; ok {
			v.SelfTest = &st
		}
		if r.State == "INSTALLED" {
			for _, o := range recs {
				if o.InstallationID != r.InstallationID && o.State == "INSTALLED" && o.ArtifactSHA256 == r.ArtifactSHA256 {
					v.SameArtifactCopies++
				}
			}
		}
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
		var b struct {
			NewCopy bool `json:"new_copy"` // explicit: register ANOTHER installation of the same artifact
		}
		decode(r, &b)
		out, err := a.requestInstall(r.Context(), r.PathValue("feature"), b.NewCopy)
		if err != nil {
			if idclient.KindOf(err) != "" {
				a.failIdentity(w, err)
				return
			}
			a.fail(w, 409, err.Error())
			return
		}
		if out.ExistingID != "" {
			writeJSON(w, 200, map[string]any{"already_installed": true, "installation_id": out.ExistingID, "state": a.State()})
			return
		}
		t := out.Tx
		writeJSON(w, 200, map[string]any{"approval_url": t.ApprovalURL, "approval_id": t.ApprovalID, "install_key": t.Key,
			"already_in_progress": out.AlreadyInProgress, "state": a.State()})
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
		if v := a.Sup.View(id); v.State == supervisor.Running {
			// Repeated Start: the supervisor owns at most one process per
			// installation; report the one that is running, start nothing.
			a.note("info", "feature.start_ignored", fmt.Sprintf("%s: Start pressed while already RUNNING (pid %d, launch %s); no second process", id, v.PID, v.LaunchID))
			writeJSON(w, 200, map[string]any{"already_running": true, "pid": v.PID, "launch_id": v.LaunchID, "state": a.State()})
			return
		}
		// Detached from the browser request: closing the tab mid-start must not
		// leave a half-started process; the supervisor finishes or cleans up.
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := a.Sup.Start(ctx, id, "manual start by the person"); err != nil {
			a.fail(w, 409, err.Error())
			return
		}
		a.Sup.ArmIdle(id)
		v := a.Sup.View(id)
		writeJSON(w, 200, map[string]any{"already_running": false, "pid": v.PID, "launch_id": v.LaunchID, "state": a.State()})
	}))
	m.HandleFunc("POST /api/installations/{id}/stop", ins(func(w http.ResponseWriter, r *http.Request, id string) {
		a.P1.mu.Lock()
		a.P1.userStop[id] = true
		a.P1.mu.Unlock()
		a.Sup.Stop(id, "stopped by the person")
		writeJSON(w, 200, a.State())
	}))
	m.HandleFunc("POST /api/installations/{id}/self-test", ins(func(w http.ResponseWriter, r *http.Request, id string) {
		v := a.Sup.View(id)
		if v.State != supervisor.Running {
			a.fail(w, 409, "NOT_RUNNING: the self-test runs inside the worker process — press Start first (it is "+v.State+")")
			return
		}
		res := SelfTestResult{At: time.Now().UTC(), LaunchID: v.LaunchID}
		// Launcher-signed call: the answer is verified as coming from THIS launch.
		st, raw, err := a.Sup.Call(id, "POST", "/v1/self-test", []byte("{}"))
		var m map[string]any
		if err == nil {
			err = json.Unmarshal(raw, &m)
		}
		switch {
		case err != nil:
			res.Result, res.Error = "ERROR", err.Error()
		case st != 200:
			res.Result, res.Error = "ERROR", fmt.Sprintf("the worker answered HTTP %d", st)
		default:
			res.Result, _ = m["result"].(string)
			if res.Result != "PASS" {
				res.Result = "FAIL"
			}
			res.DataClass, _ = m["data_class"].(string)
			res.Input, _ = m["input"].(string)
			res.Output, _ = m["output"].(map[string]any)
			res.ExternalCalls = m["external_calls"]
			res.GrantsUsed, _ = m["grants_used"].(string)
		}
		a.P1.mu.Lock()
		a.P1.selfTests[id] = res
		a.P1.mu.Unlock()
		lvl := map[bool]string{true: "info", false: "warn"}[res.Result == "PASS"]
		a.note(lvl, "feature.self_test", id+": standalone self-test "+res.Result+" (launch "+res.LaunchID+"; TEST DATA, no grant used)"+map[bool]string{true: "", false: " — " + res.Error}[res.Error == ""])
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
			ExpiresIn  int64  `json:"expires_in_seconds"` // 0 = Identity's 1-hour default
			Forever    bool   `json:"until_revoked"`      // the Admin must also confirm it in Identity
		}
		decode(r, &b)
		if !entIDRE.MatchString(b.Subject) || !capIDRE.MatchString(b.Capability) || b.ExpiresIn < 0 {
			a.fail(w, 400, "choose an identity and a capability")
			return
		}
		cl, err := a.Conn.Require(r.Context())
		if err != nil {
			a.failIdentity(w, err)
			return
		}
		ar, err := cl.RequestGrant(r.Context(), id, b.Subject, b.Capability, b.ExpiresIn, b.Forever)
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
			var ie *idclient.Error
			if errors.As(err, &ie) && ie.Code == "REVOKE_NOT_PERMITTED" {
				a.fail(w, 403, "REVOKE_NOT_PERMITTED: this grant was made for another program, so this Thought Processor can't end it by itself. Use \"Ask the Admin to revoke…\" (approved in Identity).")
				return
			}
			a.failIdentity(w, err)
			return
		}
		a.note("info", "grant.revoked", id+": grant "+gid+" revoked; the next request is denied")
		a.Conn.Kick()
		a.Conn.refreshData(r.Context())
		writeJSON(w, 200, a.State())
	}))
	// Admin emergency revocation: for any grant, approved by the Admin in Identity.
	m.HandleFunc("POST /api/installations/{id}/grants/{gid}/admin-revoke", ins(func(w http.ResponseWriter, r *http.Request, id string) {
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
		ar, err := cl.RequestAdminRevoke(r.Context(), gid, "Admin revocation asked for in Thought Processor")
		if err != nil {
			a.failIdentity(w, err)
			return
		}
		a.addPending(PendingOp{ApprovalID: ar.Approval.ID, Kind: "revoke_grant", InstallationID: id, GrantID: gid, ApprovalURL: ar.ApprovalURL,
			Summary: ar.Approval.Summary, CreatedAt: ar.Approval.CreatedAt, ExpiresAt: ar.Approval.ExpiresAt, State: "PENDING"})
		a.note("info", "grant.admin_revoke_requested", id+": Admin revocation of "+gid+" waiting for the Admin in Identity")
		writeJSON(w, 200, map[string]any{"approval_url": ar.ApprovalURL, "approval_id": ar.Approval.ID, "state": a.State()})
	}))
	m.HandleFunc("POST /api/installations/{id}/action", ins(func(w http.ResponseWriter, r *http.Request, id string) {
		var b struct {
			Subject    string      `json:"subject"`
			Capability string      `json:"capability"`
			Input      string      `json:"input"`
			Attachment *Attachment `json:"attachment"`
		}
		if err := decode(r, &b); err != nil {
			a.fail(w, 400, "the request couldn't be read (an image must be 4 MB or smaller)")
			return
		}
		if !entIDRE.MatchString(b.Subject) || !capIDRE.MatchString(b.Capability) || len(b.Input) > 1000 {
			a.fail(w, 400, "choose an identity, a capability and up to 1000 characters of test input")
			return
		}
		if err := checkAttachment(b.Attachment); err != nil {
			a.fail(w, 400, "image refused: "+err.Error()+"; nothing was sent")
			return
		}
		res := a.runAction(r.Context(), id, b.Subject, b.Capability, b.Input, b.Attachment)
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
	// Copyable activity for one installation: Processor's own records about it
	// (lifecycle, decisions, request outcomes), scrubbed, as plain text.
	m.HandleFunc("GET /api/installations/{id}/activity", a.wrap(private, func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if !insIDRE.MatchString(id) {
			a.fail(w, 400, "unknown installation")
			return
		}
		rec, err := a.loadRecord(id)
		if err != nil {
			a.fail(w, 404, "unknown installation")
			return
		}
		writeJSON(w, 200, map[string]any{"installation_id": id, "text": a.activityText(rec)})
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
	for i := range lines {
		lines[i] = scrub(lines[i]) // a feature's own log is not trusted to be clean
	}
	return lines
}

// pinInterpreter resolves a bare interpreter name (python3, node) to its real
// absolute path once, at install time. Grok L4: the interpreter's CONTENT is
// not pinned (OS updates must keep working); a same-user attacker who can
// replace it is outside P1's non-isolated threat model.
func pinInterpreter(command []string) (string, error) {
	if len(command) == 0 || strings.HasPrefix(command[0], "{package}/") {
		return "", nil
	}
	lp, err := exec.LookPath(command[0])
	if err != nil {
		return "", errors.New("REQUIREMENT_MISSING: " + command[0] + " is not installed or not on PATH")
	}
	real, err := filepath.EvalSymlinks(lp)
	if err != nil {
		return "", err
	}
	return filepath.Abs(real)
}

// activityText is a shareable, credential-free account of one installation:
// what Processor did and what Identity decided, newest last.
func (a *App) activityText(r InstallRecord) string {
	id := r.InstallationID
	var b strings.Builder
	v := a.Sup.View(id)
	fmt.Fprintf(&b, "Thought Processor activity — %s (%s %s), installation %s\n", r.Name, r.FeatureID, r.Version, id)
	fmt.Fprintf(&b, "artifact %s · process %s", r.ArtifactSHA256, v.State)
	if v.PID != 0 {
		fmt.Fprintf(&b, " · pid %d · launch %s", v.PID, v.LaunchID)
	}
	idState := "AUTHORITY UNAVAILABLE (Identity can't confirm right now)"
	if iv := a.Conn.installation(id); iv != nil {
		idState = fmt.Sprintf("%s, %d active grant(s)", iv.Installation.State, iv.ActiveGrants)
	}
	fmt.Fprintf(&b, " · Identity: %s\n", idState)
	fmt.Fprintf(&b, "copied %s — contains no keys, tickets, pairing codes or private links\n\n", time.Now().Format(time.RFC3339))
	for _, e := range a.Log.Recent(300) {
		if strings.Contains(e.Detail, id) {
			fmt.Fprintf(&b, "%s %-5s %s — %s\n", e.Time.Local().Format("15:04:05"), e.Level, e.Event, strings.ReplaceAll(e.Detail, id+": ", ""))
		}
	}
	return scrub(b.String())
}
