package app

// Universal update transaction for installed features
// (thought.feature-update/1 + Identity's identity.installation-upgrade/1).
//
// One engine for every feature: nothing here knows any particular feature.
// A component is updatable only if its NEW version declares the contract,
// its exact artifact is pinned (reviewed), and the paired Identity can
// authorize in-place upgrades. Identity's Admin approves every update.
//
// Journal: installations/<id>/update/tx.json. Phases are written BEFORE the
// step they describe, so an interrupted Processor recovers on its next
// start: anything that touched the installation is rolled back to the exact
// previous version (files, local record, Identity record).
//
//   staged/package      the verified new package (never run before commit)
//   previous/package    the old package, moved aside (never deleted mid-tx)
//   previous/install.json

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/Arameair/Thought-Processor/processor/internal/featpkg"
	"github.com/Arameair/Thought-Processor/processor/internal/idclient"
	"github.com/Arameair/Thought-Processor/processor/internal/store"
	"github.com/Arameair/Thought-Processor/processor/internal/supervisor"
	"github.com/Arameair/Thought-Processor/processor/internal/update"
)

// Internal phases (what the journal says was about to happen).
const (
	phStaged   = "STAGED"
	phWaiting  = "WAITING_FOR_APPROVAL"
	phStopping = "STOPPING"
	phSwapped  = "FILES_SWAPPED"
	phCommit   = "IDENTITY_COMMITTED"
	phRecord   = "RECORD_SAVED"
	phVerify   = "VERIFYING"
	phRecover  = "RECOVERING"
	phDone     = "DONE"
)

type UpdateStep struct {
	At     time.Time `json:"at"`
	Step   string    `json:"step"`
	Result string    `json:"result"` // OK | FAIL | INFO
	Detail string    `json:"detail,omitempty"`
}

type UpdateTx struct {
	TxID           string                     `json:"tx_id"`
	InstallationID string                     `json:"installation_id"`
	FeatureID      string                     `json:"feature_id"`
	Name           string                     `json:"name"`
	FromVersion    string                     `json:"from_version"`
	ToVersion      string                     `json:"to_version"`
	FromArtifact   string                     `json:"from_artifact"`
	ToArtifact     string                     `json:"to_artifact"`
	Source         string                     `json:"source"`
	DataMigration  string                     `json:"data_migration"`
	SecurityFix    bool                       `json:"security_fix,omitempty"`
	Entitlement    update.EntitlementDecision `json:"entitlement"`
	State          string                     `json:"state"`
	Phase          string                     `json:"phase"`
	ApprovalID     string                     `json:"approval_id,omitempty"`
	ApprovalURL    string                     `json:"approval_url,omitempty"`
	WasRunning     bool                       `json:"was_running"`
	Result         string                     `json:"result,omitempty"`
	Steps          []UpdateStep               `json:"steps"`
	CreatedAt      time.Time                  `json:"created_at"`
	UpdatedAt      time.Time                  `json:"updated_at"`
}

func (a *App) updDir(id string) string     { return filepath.Join(a.recDir(id), "update") }
func (a *App) updTxPath(id string) string  { return filepath.Join(a.updDir(id), "tx.json") }
func (a *App) updStaged(id string) string  { return filepath.Join(a.updDir(id), "staged", "package") }
func (a *App) updPrevDir(id string) string { return filepath.Join(a.updDir(id), "previous") }

func (a *App) loadUpdTx(id string) (UpdateTx, bool) {
	var t UpdateTx
	b, err := os.ReadFile(a.updTxPath(id))
	if err != nil || json.Unmarshal(b, &t) != nil || t.InstallationID != id {
		return UpdateTx{}, false
	}
	return t, true
}

func (a *App) saveUpdTx(t *UpdateTx) error {
	t.UpdatedAt = time.Now().UTC()
	if err := os.MkdirAll(a.updDir(t.InstallationID), 0o700); err != nil {
		return err
	}
	return store.WriteJSON(a.updTxPath(t.InstallationID), t)
}

func (a *App) step(t *UpdateTx, name, result, detail string) {
	t.Steps = append(t.Steps, UpdateStep{At: time.Now().UTC(), Step: name, Result: result, Detail: detail})
	a.saveUpdTx(t)
	lvl := map[string]string{"FAIL": "warn"}[result]
	if lvl == "" {
		lvl = "info"
	}
	a.note(lvl, "update."+name, t.InstallationID+" "+t.FromVersion+"→"+t.ToVersion+": "+result+" "+detail)
}

// ---- discovery ----

type pkgFound struct {
	Pkg featpkg.Package
	Dir string
	Err error
}

// trustedPackages loads every package in the trusted folders (any version).
func (a *App) trustedPackages() []pkgFound {
	out := []pkgFound{}
	add := func(dir string) {
		p, err := featpkg.Load(dir, true, a.P1.pins)
		out = append(out, pkgFound{Pkg: p, Dir: dir, Err: err})
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

func candidateOf(f pkgFound) update.Candidate {
	m := f.Pkg.Manifest
	c := update.Candidate{Dir: f.Dir, FeatureID: m.FeatureID, Version: m.Version, Artifact: f.Pkg.ArtifactSHA256, API: m.API,
		Reviewed: f.Pkg.Blocked == "" && f.Pkg.TrustBasis != "", ReviewNote: f.Pkg.Blocked}
	if u := m.Update; u != nil && u.Contract == featpkg.UpdateContract {
		c.Declared, c.MinFromVersion, c.DataMigration, c.SecurityFix = true, u.MinFromVersion, u.DataMigration, u.SecurityFix
	}
	return c
}

// verdictFor judges every package for one installed record.
func (a *App) verdictFor(r InstallRecord, identityAPI string) (update.Verdict, *pkgFound) {
	in := update.Installed{FeatureID: r.FeatureID, Version: r.Version, Artifact: r.ArtifactSHA256, API: featpkg.FeatureAPI}
	var vs []update.Verdict
	found := map[string]pkgFound{}
	for _, f := range a.trustedPackages() {
		if f.Err != nil {
			// A package for this feature that fails verification (changed
			// after review, damaged) is shown, never offered.
			var hdr struct {
				FeatureID string `json:"feature_id"`
				Version   string `json:"version"`
			}
			if b, err := os.ReadFile(filepath.Join(f.Dir, featpkg.ManifestFile)); err == nil && json.Unmarshal(b, &hdr) == nil && hdr.FeatureID == r.FeatureID {
				if c, ok := update.CompareVersions(hdr.Version, r.Version); ok && c > 0 {
					vs = append(vs, update.Verdict{State: update.Blocked, Reason: "INVALID_PACKAGE: " + hdr.Version + " failed verification (" + f.Err.Error() + ")",
						Candidate: update.Candidate{Dir: f.Dir, FeatureID: hdr.FeatureID, Version: hdr.Version}})
				}
			}
			continue
		}
		if f.Pkg.Manifest.FeatureID != r.FeatureID {
			continue
		}
		v := update.Judge(in, candidateOf(f), identityAPI)
		vs = append(vs, v)
		found[f.Pkg.ArtifactSHA256] = f
	}
	best, ok := update.Best(vs)
	if !ok {
		return best, nil
	}
	f, ok := found[best.Candidate.Artifact]
	if !ok || best.Candidate.Artifact == "" {
		return best, nil
	}
	return best, &f
}

func (a *App) identityUpgradeAPI() string {
	a.Conn.mu.Lock()
	defer a.Conn.mu.Unlock()
	if a.Conn.status == nil || a.Conn.state != CVerified {
		return ""
	}
	return a.Conn.status.Authority.UpgradeAPI
}

// ComponentView is one row of the Updates screen.
type ComponentView struct {
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	Kind          string         `json:"kind"`
	Version       string         `json:"version"`
	Revision      string         `json:"revision"`
	Status        string         `json:"status"`
	UpdateSupport string         `json:"update_support"` // SUPPORTED | UNSUPPORTED | BLOCKED
	UpdateState   string         `json:"update_state"`
	Reason        string         `json:"reason,omitempty"`
	Candidate     map[string]any `json:"candidate,omitempty"`
	Requires      []string       `json:"requires,omitempty"`
	Tx            *UpdateTx      `json:"tx,omitempty"`
	CanUpdate     bool           `json:"can_update"`
}

func (a *App) components() []ComponentView {
	out := []ComponentView{}
	api := a.identityUpgradeAPI()
	// Identity: the authority itself.
	a.Conn.mu.Lock()
	st := a.Conn.status
	a.Conn.mu.Unlock()
	if st != nil {
		reason := "Identity has no transactional self-upgrade API. Replacing Identity's code, keys, records or trust pins by a file swap is never done. An owner-approved Identity build is installed through the launcher, not from here."
		out = append(out, ComponentView{ID: "identity", Name: "Identity (" + st.IdentitySystemID + ")", Kind: "identity-service",
			Version: st.Build.Version + " " + st.Build.Branch, Revision: st.Build.Revision, Status: st.State,
			UpdateSupport: update.Blocked, UpdateState: update.Blocked, Reason: reason,
			Requires: []string{"owner approval of a reviewed Identity build", "new IDENTITY_SHA pin in the launcher (owner action)"}})
	}
	out = append(out, ComponentView{ID: "processor", Name: "Thought Processor (this program)", Kind: "processor",
		Version: a.Build.Version + " " + a.Build.Branch, Revision: a.Build.Revision, Status: "RUNNING",
		UpdateSupport: update.Unsupported, UpdateState: update.Unsupported,
		Reason: "Processor cannot replace itself while running. Use the Workbench launcher's Check Updates (read-only); applying is disabled until a reviewed release exists."})
	recs := a.records()
	sort.Slice(recs, func(i, j int) bool { return recs[i].InstalledAt.Before(recs[j].InstalledAt) })
	for _, r := range recs {
		if r.State != "INSTALLED" {
			continue
		}
		v := a.Sup.View(r.InstallationID)
		cv := ComponentView{ID: r.InstallationID, Name: r.Name, Kind: "feature:" + r.FeatureID, Version: r.Version,
			Revision: r.ArtifactSHA256, Status: v.State}
		verdict, f := a.verdictFor(r, api)
		if prev, bad := a.failedBefore(r.InstallationID, verdict.Candidate.Artifact); bad && verdict.State == update.Available {
			verdict.State = update.Blocked
			verdict.Reason = "FAILED_BEFORE: this exact version failed its checks on " + prev.UpdatedAt.Format("2006-01-02 15:04") + " UTC and was rolled back; it is not offered again (a different reviewed version is needed)"
		}
		cv.UpdateState, cv.Reason = verdict.State, verdict.Reason
		switch verdict.State {
		case update.Available:
			cv.UpdateSupport, cv.CanUpdate = "SUPPORTED", true
		case update.Current:
			cv.UpdateSupport = "SUPPORTED_WHEN_DECLARED"
			cv.Reason = "No newer reviewed version in the trusted folders."
		default:
			cv.UpdateSupport = verdict.State
		}
		if f != nil {
			c := verdict.Candidate
			cv.Candidate = map[string]any{"version": c.Version, "artifact": c.Artifact, "source": store.DisplayPath(f.Dir),
				"data_migration": c.DataMigration, "security_fix": c.SecurityFix, "reviewed": c.Reviewed}
			if f.Pkg.Manifest.Update != nil {
				cv.Candidate["notes"] = f.Pkg.Manifest.Update.Notes
			}
		}
		if cv.CanUpdate {
			cv.Requires = []string{"Admin approval in Identity's own page (upgrade_feature)", "new version's self-test must PASS after restart",
				"entitlement: " + update.EntitlementFor(a.Entitlements, verdict.Candidate.SecurityFix, r.FeatureID, r.Version, verdict.Candidate.Version).State}
		}
		if t, ok := a.loadUpdTx(r.InstallationID); ok {
			tt := t
			cv.Tx = &tt
			if !update.Terminal(t.State) {
				cv.UpdateState, cv.CanUpdate = t.State, false
			}
		}
		out = append(out, cv)
	}
	// Release-installed features (P0 installer): no in-place contract.
	for _, iv := range a.Inst.Views() {
		if iv.Installed == nil {
			continue
		}
		out = append(out, ComponentView{ID: "release:" + iv.Feature, Name: iv.Installed.Name + " (release install)", Kind: "release-feature",
			Version: iv.Installed.Tag, Revision: iv.Installed.SHA256, Status: iv.Phase, UpdateSupport: update.Unsupported, UpdateState: update.Unsupported,
			Reason: "Installed by the release installer; it has no in-place update contract. Its own Install… flow plans a reviewed release."})
	}
	return out
}

// failedBefore: the exact artifact already failed an update of this
// installation and was rolled back. It is not offered again; a different
// reviewed artifact is needed.
func (a *App) failedBefore(id, artifact string) (UpdateTx, bool) {
	t, ok := a.loadUpdTx(id)
	if ok && t.ToArtifact == artifact && (t.State == update.Recovered || t.State == update.NeedsAttention) {
		return t, true
	}
	return UpdateTx{}, false
}

// ---- start ----

var errUpdateBusy = errors.New("UPDATE_CONFLICT: an update of this installation is already in progress")

func (a *App) startUpdate(ctx context.Context, id string) (UpdateTx, error) {
	a.P1.updMu.Lock()
	defer a.P1.updMu.Unlock()
	r, err := a.loadRecord(id)
	if err != nil || r.State != "INSTALLED" {
		return UpdateTx{}, errors.New("not an installed feature")
	}
	if t, ok := a.loadUpdTx(id); ok && !update.Terminal(t.State) {
		return t, errUpdateBusy
	}
	cl, err := a.Conn.Require(ctx)
	if err != nil {
		return UpdateTx{}, err
	}
	verdict, f := a.verdictFor(r, a.identityUpgradeAPI())
	if _, bad := a.failedBefore(id, verdict.Candidate.Artifact); bad && verdict.State == update.Available {
		verdict.State, verdict.Reason = update.Blocked, "FAILED_BEFORE: this exact version already failed its checks and was rolled back"
	}
	if verdict.State != update.Available || f == nil {
		why := verdict.Reason
		if why == "" {
			why = "no newer reviewed version"
		}
		return UpdateTx{}, errors.New(verdict.State + ": " + why)
	}
	c := verdict.Candidate
	t := UpdateTx{TxID: "upd_" + store.RandHex(8), InstallationID: id, FeatureID: r.FeatureID, Name: r.Name,
		FromVersion: r.Version, ToVersion: c.Version, FromArtifact: r.ArtifactSHA256, ToArtifact: c.Artifact,
		Source: store.DisplayPath(f.Dir), DataMigration: c.DataMigration, SecurityFix: c.SecurityFix,
		State: update.Updating, Phase: phStaged, CreatedAt: time.Now().UTC()}
	// A finished earlier transaction's leftovers (incl. its previous/) are
	// replaced only now, by an explicitly started new update.
	os.RemoveAll(a.updDir(id))
	t.Entitlement = update.EntitlementFor(a.Entitlements, c.SecurityFix, r.FeatureID, r.Version, c.Version)
	if !t.Entitlement.Allows() {
		t.State, t.Result = update.Blocked, "entitlement "+t.Entitlement.State+" ("+t.Entitlement.Source+")"
		a.step(&t, "entitlement", "FAIL", t.Result)
		return t, errors.New("BLOCKED: " + t.Result)
	}
	a.step(&t, "entitlement", "OK", t.Entitlement.State+": "+t.Entitlement.Source)
	// Stage: copy, then re-load the staged copy and require the same pinned digest.
	if err := os.MkdirAll(filepath.Dir(a.updStaged(id)), 0o700); err != nil {
		return t, err
	}
	if err := featpkg.Stage(f.Pkg, a.updStaged(id)); err != nil {
		t.State, t.Result = update.Fail, "staging failed: "+err.Error()+" — nothing changed"
		a.step(&t, "stage", "FAIL", err.Error())
		return t, errors.New(t.Result)
	}
	sp, err := featpkg.Load(a.updStaged(id), true, a.P1.pins)
	if err != nil || sp.ArtifactSHA256 != c.Artifact || sp.Blocked != "" {
		t.State, t.Result = update.Fail, "staged copy does not match the reviewed artifact — nothing changed"
		a.step(&t, "stage", "FAIL", t.Result)
		os.RemoveAll(filepath.Dir(a.updStaged(id)))
		return t, errors.New(t.Result)
	}
	if _, err := pinInterpreter(sp.Manifest.Runtime.Command); err != nil {
		t.State, t.Result = update.Fail, "requirement check failed: "+err.Error()+" — nothing changed"
		a.step(&t, "requirements", "FAIL", err.Error())
		return t, errors.New(t.Result)
	}
	a.step(&t, "stage", "OK", "staged and verified artifact "+c.Artifact[:12]+"…")
	caps := []idclient.Capability{}
	for _, x := range sp.Manifest.Capabilities {
		caps = append(caps, idclient.Capability{ID: x.ID, Description: x.Description, DataClass: x.DataClass})
	}
	m := sp.Manifest
	ar, err := cl.RequestUpgrade(ctx, id, idclient.UpgradeRequest{FromArtifactSHA256: r.ArtifactSHA256, DataMigration: c.DataMigration,
		SecurityFix: c.SecurityFix, Entitlement: t.Entitlement.State,
		To: idclient.InstallRequest{FeatureID: m.FeatureID, Name: m.Name, Version: m.Version, API: m.API, PublisherClaim: m.Publisher.Name,
			SourceKind: "local-trusted-package", SourceRef: m.Source.Repo + "/" + m.Source.Path, TrustClass: m.TrustClass,
			ArtifactSHA256: sp.ArtifactSHA256, ManifestSHA256: sp.ManifestSHA256, Capabilities: caps}})
	if err != nil {
		t.State, t.Result = update.Fail, "Identity refused the upgrade request: "+err.Error()+" — nothing changed"
		a.step(&t, "authorize", "FAIL", err.Error())
		os.RemoveAll(filepath.Dir(a.updStaged(id)))
		return t, errors.New(t.Result)
	}
	t.ApprovalID, t.ApprovalURL, t.Phase, t.State = ar.Approval.ID, ar.ApprovalURL, phWaiting, update.ApprovalRequired
	a.step(&t, "authorize", "OK", "waiting for the Admin in Identity ("+ar.Approval.ID+")")
	return t, nil
}

func (a *App) cancelUpdate(id string) error {
	a.P1.updMu.Lock()
	defer a.P1.updMu.Unlock()
	t, ok := a.loadUpdTx(id)
	if !ok || t.Phase != phWaiting {
		return errors.New("only an update that is still waiting for approval can be cancelled")
	}
	os.RemoveAll(filepath.Dir(a.updStaged(id)))
	t.State, t.Phase, t.Result = update.Cancelled, phDone, "cancelled before approval — nothing changed"
	a.step(&t, "cancel", "OK", "")
	return nil
}

// ---- advance (called from the reconcile loop) ----

func (a *App) advanceUpdates(ctx context.Context, cl *idclient.Client) {
	for _, r := range a.records() {
		t, ok := a.loadUpdTx(r.InstallationID)
		if !ok || update.Terminal(t.State) {
			continue
		}
		a.P1.mu.Lock()
		running := a.P1.applying[t.InstallationID]
		a.P1.mu.Unlock()
		if running {
			continue
		}
		switch t.Phase {
		case phWaiting:
			ap, err := cl.Approval(ctx, t.ApprovalID)
			if err != nil {
				continue
			}
			switch ap.State {
			case "DENIED":
				os.RemoveAll(filepath.Dir(a.updStaged(t.InstallationID)))
				t.State, t.Phase, t.Result = update.Denied, phDone, "the Admin denied the update — nothing changed"
				a.step(&t, "authorize", "FAIL", "denied in Identity")
			case "EXPIRED":
				os.RemoveAll(filepath.Dir(a.updStaged(t.InstallationID)))
				t.State, t.Phase, t.Result = update.Expired, phDone, "the approval expired — nothing changed"
				a.step(&t, "authorize", "FAIL", "approval expired")
			case "APPROVED", "CONSUMED":
				a.goApply(t, false)
			}
		case phStaged:
			// Interrupted before Identity was asked: nothing changed.
			os.RemoveAll(filepath.Dir(a.updStaged(t.InstallationID)))
			t.State, t.Phase, t.Result = update.Fail, phDone, "interrupted before authorization — nothing changed"
			a.step(&t, "interrupted", "INFO", t.Result)
		default:
			// Interrupted mid-apply (Processor stopped): recover the previous version.
			a.goApply(t, true)
		}
	}
}

func (a *App) goApply(t UpdateTx, recoverOnly bool) {
	a.P1.mu.Lock()
	if a.P1.applying[t.InstallationID] {
		a.P1.mu.Unlock()
		return
	}
	a.P1.applying[t.InstallationID] = true
	a.P1.mu.Unlock()
	go func() {
		defer func() {
			a.P1.mu.Lock()
			delete(a.P1.applying, t.InstallationID)
			delete(a.P1.updStart, t.InstallationID)
			a.P1.mu.Unlock()
		}()
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()
		if recoverOnly {
			a.step(&t, "interrupted", "INFO", "Processor stopped during phase "+t.Phase+"; recovering the previous version")
			a.recoverUpdate(ctx, &t, "interrupted during "+t.Phase)
			return
		}
		a.applyUpdate(ctx, &t)
	}()
}

// ---- apply ----

func (a *App) applyUpdate(ctx context.Context, t *UpdateTx) {
	id := t.InstallationID
	old, err := a.loadRecord(id)
	if err != nil || old.ArtifactSHA256 != t.FromArtifact {
		t.State, t.Phase, t.Result = update.Fail, phDone, "the installation changed since the update was requested — nothing changed"
		a.step(t, "apply", "FAIL", t.Result)
		return
	}
	staged, err := featpkg.Load(a.updStaged(id), true, a.P1.pins)
	if err != nil || staged.ArtifactSHA256 != t.ToArtifact || staged.Blocked != "" {
		t.State, t.Phase, t.Result = update.Fail, phDone, "the staged package changed or is no longer pinned — nothing changed"
		a.step(t, "apply", "FAIL", t.Result)
		return
	}
	interp, err := pinInterpreter(staged.Manifest.Runtime.Command)
	if err != nil {
		t.State, t.Phase, t.Result = update.Fail, phDone, "requirement missing: "+err.Error()+" — nothing changed"
		a.step(t, "apply", "FAIL", t.Result)
		return
	}
	t.State = update.Updating
	// 1. stop (record whether it was running so it is restored afterwards)
	t.WasRunning = a.Sup.View(id).State == supervisor.Running
	t.Phase = phStopping
	a.saveUpdTx(t)
	if t.WasRunning {
		if err := a.Sup.Stop(id, "update to "+t.ToVersion); err != nil {
			t.State, t.Phase, t.Result = update.Fail, phDone, "could not stop the running feature: "+err.Error()+" — nothing changed"
			a.step(t, "stop", "FAIL", err.Error())
			return
		}
	}
	a.step(t, "stop", "OK", map[bool]string{true: "stopped the running feature", false: "it was not running"}[t.WasRunning])
	// 2. swap files (old package + record kept aside)
	prev := a.updPrevDir(id)
	if err := os.MkdirAll(prev, 0o700); err != nil {
		a.recoverUpdate(ctx, t, "could not prepare: "+err.Error())
		return
	}
	if b, err := os.ReadFile(a.recPath(id)); err == nil {
		store.WriteFile(filepath.Join(prev, "install.json"), b)
	}
	t.Phase = phSwapped
	a.saveUpdTx(t)
	if err := os.Rename(filepath.Join(a.recDir(id), "package"), filepath.Join(prev, "package")); err != nil {
		a.recoverUpdate(ctx, t, "could not move the old package aside: "+err.Error())
		return
	}
	if err := os.Rename(a.updStaged(id), filepath.Join(a.recDir(id), "package")); err != nil {
		a.recoverUpdate(ctx, t, "could not put the new package in place: "+err.Error())
		return
	}
	a.step(t, "swap", "OK", "new package in place; old package kept in update/previous")
	// 3. Identity commits the approved upgrade (same installation ID + credential)
	cl, err := a.Conn.Require(ctx)
	if err != nil {
		a.recoverUpdate(ctx, t, "Identity unavailable at commit: "+err.Error())
		return
	}
	t.Phase = phCommit
	a.saveUpdTx(t)
	res, err := cl.CommitUpgrade(ctx, id, t.ApprovalID)
	if err != nil {
		a.recoverUpdate(ctx, t, "Identity refused the commit: "+err.Error())
		return
	}
	if res.Installation.ID != id || res.Installation.ArtifactSHA256 != t.ToArtifact || res.Installation.Version != t.ToVersion {
		a.recoverUpdate(ctx, t, "Identity answered for a different installation or artifact")
		return
	}
	detail := "Identity now records " + t.ToVersion + " for " + id + " (same installation ID)"
	if len(res.GrantsRevoked) > 0 {
		detail += fmt.Sprintf("; grants revoked for dropped capabilities: %v", res.GrantsRevoked)
	}
	a.step(t, "identity-commit", "OK", detail)
	if os.Getenv("TP_TEST_CRASH_POINT") == "update-after-commit" {
		a.Log.Warn("test", "test.crash", "TEST-ONLY crash point update-after-commit reached; exiting")
		os.Exit(42)
	}
	// 4. local record
	m := staged.Manifest
	rec := old
	rec.Name, rec.Version, rec.Description, rec.PublisherClaim = m.Name, m.Version, m.Description, m.Publisher.Name
	rec.ArtifactSHA256, rec.ManifestSHA256, rec.Files, rec.Command, rec.Requires = staged.ArtifactSHA256, staged.ManifestSHA256, m.Files, m.Runtime.Command, m.Runtime.Requires
	rec.Capabilities, rec.StartupModes, rec.SourceDir, rec.SourceRepo = m.Capabilities, m.Runtime.StartupModes, t.Source, m.Source.Repo+"/"+m.Source.Path
	rec.TrustClass, rec.TrustBasis, rec.Interpreter = m.TrustClass, staged.TrustBasis, interp
	now := time.Now().UTC()
	rec.UpdatedAt = &now
	t.Phase = phRecord
	a.saveUpdTx(t)
	if err := a.saveRecord(rec); err != nil {
		a.recoverUpdate(ctx, t, "could not save the installation record: "+err.Error())
		return
	}
	a.step(t, "record", "OK", "")
	// 5. verify: restart, health/self-test, Identity + files agree
	t.Phase, t.State = phVerify, update.Restarting
	a.saveUpdTx(t)
	if err := a.verifyRunning(ctx, id, t.ToArtifact, t.ToVersion, staged.Manifest.Standalone.SelfTest, t.WasRunning); err != nil {
		a.step(t, "verify", "FAIL", err.Error())
		a.recoverUpdate(ctx, t, "new version failed its checks: "+err.Error())
		return
	}
	t.Phase, t.State = phDone, update.Pass
	t.Result = "updated " + t.FromVersion + " → " + t.ToVersion + "; Identity, the installed files and a fresh launch's self-test all confirm " + t.ToVersion
	a.step(t, "verify", "OK", t.Result)
	a.Conn.Kick()
}

// verifyRunning proves an installation is healthy at an exact artifact:
// Identity's live record, the local record, the installed files, a real
// launch, and (if the feature has one) its standalone self-test.
func (a *App) verifyRunning(ctx context.Context, id, artifact, version string, selfTest, keepRunning bool) error {
	cl, err := a.Conn.Require(ctx)
	if err != nil {
		return err
	}
	a.Conn.refreshData(ctx)
	iv, err := cl.Installation(ctx, id)
	if err != nil {
		return fmt.Errorf("Identity can't confirm the installation: %v", err)
	}
	if iv.Installation.ArtifactSHA256 != artifact || iv.Installation.Version != version || iv.Installation.State != "INSTALLED" {
		return fmt.Errorf("Identity reports %s %s (%s), expected %s", iv.Installation.Version, short(iv.Installation.ArtifactSHA256), iv.Installation.State, version)
	}
	r, err := a.loadRecord(id)
	if err != nil || r.ArtifactSHA256 != artifact {
		return errors.New("local record does not match")
	}
	if err := featpkg.VerifyInstalled(filepath.Join(a.recDir(id), "package"), r.ManifestSHA256, r.Files); err != nil {
		return err
	}
	a.P1.mu.Lock()
	a.P1.updStart[id] = true
	a.P1.mu.Unlock()
	sctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err := a.Sup.Start(sctx, id, "update verification"); err != nil {
		return fmt.Errorf("restart failed: %v", err)
	}
	if v := a.Sup.View(id); v.State != supervisor.Running {
		return fmt.Errorf("restart did not reach RUNNING (%s)", v.State)
	}
	// The running process itself (launcher-signed answer) must report the version.
	if st, raw, err := a.Sup.Call(id, "GET", "/v1/status", nil); err == nil && st == 200 {
		var s struct {
			Version        string `json:"version"`
			InstallationID string `json:"installation_id"`
		}
		json.Unmarshal(raw, &s)
		if s.InstallationID != id || (s.Version != "" && s.Version != version) {
			a.Sup.Stop(id, "update verification failed")
			return fmt.Errorf("the running process reports %s %s, expected %s %s", s.InstallationID, s.Version, id, version)
		}
	} else {
		a.Sup.Stop(id, "update verification failed")
		return fmt.Errorf("the running process did not answer its signed status (HTTP %d, %v)", st, err)
	}
	if selfTest {
		st, raw, err := a.Sup.Call(id, "POST", "/v1/self-test", []byte("{}"))
		var m map[string]any
		if err == nil {
			err = json.Unmarshal(raw, &m)
		}
		if err != nil || st != 200 || m["result"] != "PASS" {
			a.Sup.Stop(id, "update verification failed")
			return fmt.Errorf("self-test did not PASS (HTTP %d, result %v, %v)", st, m["result"], err)
		}
	}
	if !keepRunning {
		a.Sup.Stop(id, "update verification finished; it was not running before")
	} else {
		a.Sup.ArmIdle(id)
	}
	return nil
}

// recoverUpdate puts the exact previous version back (files, record,
// Identity record) and proves it. Idempotent: safe after a crash at any point.
func (a *App) recoverUpdate(ctx context.Context, t *UpdateTx, why string) {
	id := t.InstallationID
	t.State, t.Result = update.Recovering, why
	t.Phase = phRecover
	a.step(t, "recover", "INFO", why)
	a.Sup.Stop(id, "update recovery")
	problems := []string{}
	// Identity first: if it records the new artifact, roll back the record.
	if cl, err := a.Conn.Require(ctx); err != nil {
		problems = append(problems, "Identity unavailable: "+err.Error())
	} else if iv, err := cl.Installation(ctx, id); err != nil {
		problems = append(problems, "Identity can't confirm: "+err.Error())
	} else if iv.Installation.ArtifactSHA256 == t.ToArtifact {
		if _, err := cl.RollbackUpgrade(ctx, id, t.ApprovalID, why); err != nil {
			problems = append(problems, "Identity refused the rollback: "+err.Error())
		} else {
			a.step(t, "identity-rollback", "OK", "Identity restored "+t.FromVersion)
		}
	}
	// Files and record.
	prev := a.updPrevDir(id)
	pkg := filepath.Join(a.recDir(id), "package")
	if _, err := os.Stat(filepath.Join(prev, "package")); err == nil {
		os.RemoveAll(pkg)
		if err := os.Rename(filepath.Join(prev, "package"), pkg); err != nil {
			problems = append(problems, "could not restore the old package: "+err.Error())
		}
	}
	if b, err := os.ReadFile(filepath.Join(prev, "install.json")); err == nil {
		var r InstallRecord
		if json.Unmarshal(b, &r) == nil && r.ArtifactSHA256 == t.FromArtifact {
			if err := a.saveRecord(r); err != nil {
				problems = append(problems, "could not restore the record: "+err.Error())
			}
		}
	}
	os.RemoveAll(filepath.Dir(a.updStaged(id)))
	if len(problems) == 0 {
		// Prove the old version: it must be what Identity, the record and the files say, and it must run if it ran before.
		if err := a.verifyRunning(ctx, id, t.FromArtifact, t.FromVersion, a.recordSelfTest(id), t.WasRunning); err != nil {
			problems = append(problems, "previous version did not verify: "+err.Error())
		}
	}
	if len(problems) > 0 {
		t.State, t.Phase = update.NeedsAttention, phDone
		t.Result = why + " — RECOVERY INCOMPLETE: " + fmt.Sprint(problems)
		a.step(t, "recover", "FAIL", fmt.Sprint(problems))
		return
	}
	t.State, t.Phase = update.Recovered, phDone
	t.Result = why + " — recovered " + t.FromVersion + " (Identity, files and a fresh launch verified)"
	a.step(t, "recover", "OK", "previous version "+t.FromVersion+" verified")
	a.Conn.Kick()
}

func (a *App) recordSelfTest(id string) bool {
	p, err := featpkg.Load(filepath.Join(a.recDir(id), "package"), true, a.P1.pins)
	return err == nil && p.Manifest.Standalone.SelfTest
}

// ---- routes (inside the Development Workstation; gated like the rest) ----

func (a *App) updateRoutes(m *http.ServeMux, gated func(level, func(http.ResponseWriter, *http.Request, wsGate)) http.HandlerFunc) {
	m.HandleFunc("GET /api/ws/updates", gated(private, func(w http.ResponseWriter, r *http.Request, g wsGate) {
		writeJSON(w, 200, map[string]any{"components": a.components(), "identity_upgrade_api": a.identityUpgradeAPI(),
			"contract": featpkg.UpdateContract, "checked_at": time.Now().UTC()})
	}))
	m.HandleFunc("POST /api/ws/updates/{id}/start", gated(mutate, func(w http.ResponseWriter, r *http.Request, g wsGate) {
		id := r.PathValue("id")
		if !insIDRE.MatchString(id) {
			a.fail(w, 404, "only installed features can be updated here")
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		t, err := a.startUpdate(ctx, id)
		if err != nil {
			code := 409
			writeJSON(w, code, map[string]any{"error": err.Error(), "tx": t})
			return
		}
		writeJSON(w, 202, map[string]any{"tx": t})
	}))
	m.HandleFunc("POST /api/ws/updates/{id}/cancel", gated(mutate, func(w http.ResponseWriter, r *http.Request, g wsGate) {
		if err := a.cancelUpdate(r.PathValue("id")); err != nil {
			a.fail(w, 409, err.Error())
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true})
	}))
}
