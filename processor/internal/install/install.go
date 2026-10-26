// Package install is Thought Processor's reusable feature installer.
//
// It knows nothing about any particular feature. Everything comes from the
// feature contract (thought.feature/1) and the release that the paired
// Identity reports — both signed answers — and from the person's explicit
// approval of a concrete plan:
//
//  1. Plan: read the contract + latest release, check that the release
//     matches the contract's approved source (repository, tag prefix,
//     trusted uploader, file name, size limit, SHA-256 present).
//  2. Approve: the person sees exactly what will be installed and approves.
//  3. Install: download through Identity into a .partial file while hashing;
//     refuse a short download or a wrong SHA-256; only then move it into
//     place and write the install record.
//  4. On every start: re-hash installed files (detects changes/removal) and
//     delete leftover .partial files from interrupted downloads.
//
// Layout: <profile>/features/<feature>/<tag>/<asset> + install.json,
// <profile>/features/<feature>/current.json. Nothing outside the profile is
// touched, no admin rights are needed, and installing never starts anything.
//
// Honest limit: releases are not signed by the publisher yet. The checks
// prove the file came from the approved repository's own release workflow on
// GitHub and arrived undamaged — not that the publisher personally approved it.
package install

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Arameair/Thought-Processor/processor/internal/idclient"
	"github.com/Arameair/Thought-Processor/processor/internal/store"
)

const (
	ContractSchema = "thought.feature/1"
	maxAllowed     = 1 << 30
	planTTL        = 10 * time.Minute
)

var (
	FeatureRE = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,39}$`)
	tagRE     = regexp.MustCompile(`^[A-Za-z0-9._-]{1,64}$`)
	assetRE   = regexp.MustCompile(`^[A-Za-z0-9._-]{1,100}$`)
	shaRE     = regexp.MustCompile(`^[0-9a-f]{64}$`)
	repoRE    = regexp.MustCompile(`^[A-Za-z0-9-]+/[A-Za-z0-9._-]+$`)
)

func Platform() string { return runtime.GOOS + "-" + runtime.GOARCH }

// Source is what the installer needs from Identity (idclient.Client).
type Source interface {
	Contract(ctx context.Context, feature string) (idclient.Contract, error)
	Release(ctx context.Context, feature, platform string) (idclient.Release, error)
	Download(ctx context.Context, path string) (io.ReadCloser, int64, error)
	DownloadPath(feature, platform, tag string) string
	Pairing() idclient.Pairing
}

// Phases.
const (
	NotInstalled = "NOT_INSTALLED"
	Planned      = "AWAITING_APPROVAL"
	Downloading  = "DOWNLOADING"
	Verifying    = "VERIFYING"
	Installed    = "INSTALLED"
	Failed       = "FAILED"
)

// Problem codes.
const (
	PContract     = "CONTRACT_INVALID"
	PUnapproved   = "UNAPPROVED_SOURCE"
	PNoRelease    = "NO_RELEASE"
	PUnavailable  = "AUTHORITY_UNAVAILABLE"
	PRefused      = "IDENTITY_REFUSED"
	PInterrupted  = "DOWNLOAD_INTERRUPTED"
	PChecksum     = "CHECKSUM_MISMATCH"
	PDisk         = "DISK_ERROR"
	PPlanExpired  = "PLAN_EXPIRED"
	PFilesChanged = "FILES_CHANGED"
	PMissing      = "FILES_MISSING"
)

type Problem struct {
	Code     string    `json:"code"`
	Message  string    `json:"message"`
	NextStep string    `json:"next_step"`
	At       time.Time `json:"at"`
}

// Plan is the exact thing the person approves.
type Plan struct {
	ID          string            `json:"plan_id"`
	Feature     string            `json:"feature"`
	Name        string            `json:"name"`
	Publisher   string            `json:"publisher"`
	Tag         string            `json:"tag"`
	Asset       string            `json:"asset"`
	Size        int64             `json:"size"`
	SHA256      string            `json:"sha256"`
	Platform    string            `json:"platform"`
	Repo        string            `json:"repo"`
	ReleaseURL  string            `json:"release_url,omitempty"`
	Uploader    string            `json:"uploader"`
	Checks      []string          `json:"checks"`
	NotChecked  string            `json:"not_checked"`
	WouldAskFor []string          `json:"would_ask_for,omitempty"`
	DecisionID  string            `json:"decision_id"`
	CreatedAt   time.Time         `json:"created_at"`
	Upgrade     bool              `json:"upgrade"`
	Contract    idclient.Contract `json:"-"`
}

// Record is written next to the installed file.
type Record struct {
	Feature          string    `json:"feature"`
	Name             string    `json:"name"`
	Tag              string    `json:"tag"`
	Asset            string    `json:"asset"`
	Platform         string    `json:"platform"`
	Size             int64     `json:"size"`
	SHA256           string    `json:"sha256"`
	Repo             string    `json:"repo"`
	ReleaseURL       string    `json:"release_url,omitempty"`
	Uploader         string    `json:"uploader"`
	ContractSchema   string    `json:"contract_schema"`
	Checks           []string  `json:"checks"`
	NotChecked       string    `json:"not_checked"`
	DecisionID       string    `json:"decision_id"`
	IdentitySystemID string    `json:"identity_system_id"`
	ApprovedAt       time.Time `json:"approved_at"`
	InstalledAt      time.Time `json:"installed_at"`
	Path             string    `json:"path"`
}

type featureState struct {
	phase     string
	got, size int64
	plan      *Plan
	record    *Record
	verify    string // OK | FILES_CHANGED | FILES_MISSING | ""
	verified  time.Time
	problem   *Problem
	busy      bool
}

// View is the per-feature state Processor shows.
type View struct {
	Feature    string    `json:"feature"`
	Phase      string    `json:"phase"`
	Got        int64     `json:"got,omitempty"`
	Size       int64     `json:"size,omitempty"`
	Plan       *Plan     `json:"plan,omitempty"`
	Installed  *Record   `json:"installed,omitempty"`
	Verify     string    `json:"verify,omitempty"`
	VerifiedAt time.Time `json:"verified_at,omitempty"`
	Problem    *Problem  `json:"problem,omitempty"`
	Busy       bool      `json:"busy"`
}

type Event func(level, event, feature, detail string)

type Installer struct {
	root string
	log  Event
	mu   sync.Mutex
	fs   map[string]*featureState
}

func New(profileDir string, log Event) *Installer {
	if log == nil {
		log = func(string, string, string, string) {}
	}
	in := &Installer{root: filepath.Join(profileDir, "features"), log: log, fs: map[string]*featureState{}}
	in.loadAll()
	return in
}

func (in *Installer) st(f string) *featureState {
	s := in.fs[f]
	if s == nil {
		s = &featureState{phase: NotInstalled}
		in.fs[f] = s
	}
	return s
}

// loadAll reads install records, re-verifies files and removes leftovers.
func (in *Installer) loadAll() {
	dirs, _ := os.ReadDir(in.root)
	for _, d := range dirs {
		if !d.IsDir() || !FeatureRE.MatchString(d.Name()) {
			continue
		}
		f := d.Name()
		if parts, _ := filepath.Glob(filepath.Join(in.root, f, "*.partial")); len(parts) > 0 {
			for _, p := range parts {
				os.Remove(p)
			}
			in.log("warn", "install.partial_removed", f, fmt.Sprintf("removed %d unfinished download(s) left by an interrupted install", len(parts)))
		}
		rec, err := in.readRecord(f)
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				in.log("error", "install.record_unreadable", f, err.Error())
				s := in.st(f)
				s.phase, s.problem = Failed, &Problem{Code: PDisk, Message: "The install record is damaged: " + err.Error(), NextStep: "Remove the feature and install it again.", At: time.Now()}
			}
			continue
		}
		s := in.st(f)
		s.record, s.phase = rec, Installed
		in.verifyLocked(f, s)
	}
}

func (in *Installer) readRecord(f string) (*Record, error) {
	b, err := os.ReadFile(filepath.Join(in.root, f, "current.json"))
	if err != nil {
		return nil, err
	}
	var cur struct {
		Tag string `json:"tag"`
	}
	if json.Unmarshal(b, &cur) != nil || !tagRE.MatchString(cur.Tag) {
		return nil, errors.New("current.json damaged")
	}
	b, err = os.ReadFile(filepath.Join(in.root, f, cur.Tag, "install.json"))
	if err != nil {
		return nil, fmt.Errorf("install.json for %s missing: %w", cur.Tag, err)
	}
	var r Record
	if json.Unmarshal(b, &r) != nil || r.Feature != f || !assetRE.MatchString(r.Asset) || !shaRE.MatchString(r.SHA256) {
		return nil, errors.New("install.json damaged")
	}
	r.Path = filepath.Join(in.root, f, cur.Tag, r.Asset)
	return &r, nil
}

func hashFile(p string) (string, int64, error) {
	f, err := os.Open(p)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, f)
	return hex.EncodeToString(h.Sum(nil)), n, err
}

func (in *Installer) verifyLocked(f string, s *featureState) {
	sum, n, err := hashFile(s.record.Path)
	s.verified = time.Now()
	switch {
	case errors.Is(err, os.ErrNotExist):
		s.verify = PMissing
		s.problem = &Problem{Code: PMissing, Message: "The installed file is gone.", NextStep: "Install it again.", At: time.Now()}
		in.log("warn", "install.files_missing", f, "installed file not found at start")
	case err != nil || sum != s.record.SHA256 || n != s.record.Size:
		s.verify = PFilesChanged
		s.problem = &Problem{Code: PFilesChanged, Message: "The installed file no longer matches what was approved (it was changed or damaged).", NextStep: "Remove it and install it again.", At: time.Now()}
		in.log("warn", "install.files_changed", f, "installed file hash differs from the install record")
	default:
		s.verify = "OK"
	}
}

// Verify re-hashes one installed feature now.
func (in *Installer) Verify(f string) View {
	in.mu.Lock()
	defer in.mu.Unlock()
	s := in.st(f)
	if s.record != nil && !s.busy {
		s.problem = nil
		in.verifyLocked(f, s)
	}
	return in.viewLocked(f, s)
}

func (in *Installer) viewLocked(f string, s *featureState) View {
	v := View{Feature: f, Phase: s.phase, Got: s.got, Size: s.size, Plan: s.plan, Verify: s.verify, VerifiedAt: s.verified, Problem: s.problem, Busy: s.busy}
	if s.plan != nil && time.Since(s.plan.CreatedAt) > planTTL {
		v.Plan, v.Phase = nil, phaseWithout(s)
	}
	if s.record != nil {
		r := *s.record
		r.Path = store.DisplayPath(r.Path)
		v.Installed = &r
	}
	return v
}

func phaseWithout(s *featureState) string {
	if s.record != nil {
		return Installed
	}
	return NotInstalled
}

func (in *Installer) View(f string) View {
	in.mu.Lock()
	defer in.mu.Unlock()
	return in.viewLocked(f, in.st(f))
}

// Views returns all known features (installed or in progress).
func (in *Installer) Views() []View {
	in.mu.Lock()
	defer in.mu.Unlock()
	keys := make([]string, 0, len(in.fs))
	for k := range in.fs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := []View{}
	for _, k := range keys {
		out = append(out, in.viewLocked(k, in.fs[k]))
	}
	return out
}

func (in *Installer) fail(f string, p Problem) error {
	in.mu.Lock()
	s := in.st(f)
	p.At = time.Now()
	s.problem, s.busy, s.plan, s.got, s.size = &p, false, nil, 0, 0
	if s.record != nil {
		s.phase = Installed
	} else {
		s.phase = Failed
	}
	in.mu.Unlock()
	in.log("warn", "install.failed", f, p.Code+": "+p.Message)
	return errors.New(p.Code + ": " + p.Message)
}

func problemFromIdentity(err error) Problem {
	var e *idclient.Error
	if errors.As(err, &e) {
		switch e.Kind {
		case idclient.Unavailable, idclient.NotPaired:
			return Problem{Code: PUnavailable, Message: "Identity can't be reached, so nothing can be checked or installed.", NextStep: "Make sure Identity is running; Processor reconnects by itself."}
		case idclient.TokenRefused, idclient.WrongService:
			return Problem{Code: PUnavailable, Message: e.Msg, NextStep: "Connect to Identity again."}
		}
		switch e.Code {
		case "NO_RELEASE":
			return Problem{Code: PNoRelease, Message: e.Msg, NextStep: "The publisher needs to publish a release. Nothing was installed."}
		case "UNAPPROVED_SOURCE":
			return Problem{Code: PUnapproved, Message: e.Msg, NextStep: "Nothing was installed. Tell the publisher; only releases built by the feature's own release workflow are accepted."}
		}
		return Problem{Code: PRefused, Message: e.Msg, NextStep: "Nothing was installed. Try again later; if it repeats, turn on Developer Mode and copy the details."}
	}
	return Problem{Code: PRefused, Message: err.Error(), NextStep: "Try again."}
}

// expandAsset fills the contract's asset template for a platform.
func expandAsset(tmpl, platform string) string {
	parts := strings.SplitN(platform, "-", 2)
	if len(parts) != 2 {
		return ""
	}
	exe := ""
	if parts[0] == "windows" {
		exe = ".exe"
	}
	return strings.NewReplacer("{os}", parts[0], "{arch}", parts[1], "{exe}", exe).Replace(tmpl)
}

func trusted(list []string, login string) bool {
	for _, u := range list {
		if login != "" && strings.EqualFold(u, login) {
			return true
		}
	}
	return false
}

// CheckRelease enforces the contract on a release. Exported for tests.
func CheckRelease(feature, platform string, k idclient.Contract, r idclient.Release) (code string, err error) {
	switch {
	case k.Schema != ContractSchema:
		return PContract, fmt.Errorf("feature contract version %q isn't one this Processor understands (needs %s)", k.Schema, ContractSchema)
	case k.FeatureID != feature:
		return PContract, errors.New("the contract is for a different feature")
	case k.Source == nil || k.Install == nil:
		return PContract, errors.New("this feature has no install rule")
	case k.Source.Kind != "github-release" || !repoRE.MatchString(k.Source.Repo) || k.Source.TagPrefix == "" || len(k.Source.TrustedUploaders) == 0:
		return PContract, errors.New("the contract's approved source is incomplete")
	case k.Install.Kind != "single-executable" || k.Install.MaxBytes <= 0 || k.Install.MaxBytes > maxAllowed:
		return PContract, errors.New("the contract's install rule isn't supported")
	}
	want := expandAsset(k.Source.AssetTemplate, platform)
	switch {
	case r.Feature != "" && r.Feature != feature:
		return PUnapproved, errors.New("the release is for a different feature")
	case r.Platform != platform || r.Asset != want || !assetRE.MatchString(r.Asset):
		return PUnapproved, fmt.Errorf("the release file %q isn't the approved file name %q for this computer", r.Asset, want)
	case !tagRE.MatchString(r.Tag) || !strings.HasPrefix(r.Tag, k.Source.TagPrefix):
		return PUnapproved, fmt.Errorf("release %q doesn't match the approved tag pattern %s*", r.Tag, k.Source.TagPrefix)
	case r.Source == nil || !strings.EqualFold(r.Source.Repo, k.Source.Repo):
		return PUnapproved, errors.New("the release doesn't come from the approved repository " + k.Source.Repo)
	case !trusted(k.Source.TrustedUploaders, r.Source.AssetUploader) || !trusted(k.Source.TrustedUploaders, r.Source.ChecksumUploader):
		return PUnapproved, fmt.Errorf("the file was uploaded by %q, not by %s", r.Source.AssetUploader, strings.Join(k.Source.TrustedUploaders, ", "))
	case r.Source.Prerelease && !k.Source.AllowPrerelease:
		return PUnapproved, errors.New("the release is a pre-release, which this feature doesn't allow")
	case r.Size <= 0 || r.Size > k.Install.MaxBytes:
		return PUnapproved, fmt.Errorf("the file size %d is outside the allowed range", r.Size)
	case !shaRE.MatchString(r.SHA256):
		return PUnapproved, errors.New("the release has no valid SHA-256 to check the file against")
	}
	return "", nil
}

// Prepare reads the contract and latest release and builds a plan for the
// person to approve. Nothing is downloaded.
func (in *Installer) Prepare(ctx context.Context, src Source, feature string) (*Plan, error) {
	if !FeatureRE.MatchString(feature) {
		return nil, errors.New("bad feature id")
	}
	in.mu.Lock()
	s := in.st(feature)
	if s.busy {
		in.mu.Unlock()
		return nil, errors.New("an install is already running for this feature")
	}
	s.problem = nil
	in.mu.Unlock()

	k, err := src.Contract(ctx, feature)
	if err != nil {
		return nil, in.fail(feature, problemFromIdentity(err))
	}
	platform := Platform()
	r, err := src.Release(ctx, feature, platform)
	if err != nil {
		return nil, in.fail(feature, problemFromIdentity(err))
	}
	if code, err := CheckRelease(feature, platform, k, r); err != nil {
		return nil, in.fail(feature, Problem{Code: code, Message: err.Error(), NextStep: "Nothing was installed."})
	}
	p := &Plan{ID: "plan_" + store.RandHex(8), Feature: feature, Name: k.Name, Publisher: k.Publisher, Tag: r.Tag, Asset: r.Asset,
		Size: r.Size, SHA256: r.SHA256, Platform: platform, Repo: r.Source.Repo, ReleaseURL: r.Source.URL,
		Uploader: r.Source.AssetUploader, Checks: r.Source.Checks, NotChecked: r.Source.NotChecked, WouldAskFor: k.WouldAskFor,
		DecisionID: r.DecisionID, CreatedAt: time.Now(), Contract: k}
	in.mu.Lock()
	if s.record != nil {
		if s.record.Tag == r.Tag && s.record.SHA256 == r.SHA256 && s.verify == "OK" {
			in.mu.Unlock()
			in.log("info", "install.up_to_date", feature, r.Tag+" is already installed and verified")
			return nil, nil
		}
		p.Upgrade = true
	}
	s.plan, s.phase = p, Planned
	in.mu.Unlock()
	in.log("info", "install.planned", feature, fmt.Sprintf("%s %s (%d bytes) from %s — waiting for approval", r.Tag, r.Asset, r.Size, r.Source.Repo))
	return p, nil
}

// CancelPlan drops a plan the person didn't approve.
func (in *Installer) CancelPlan(feature string) {
	in.mu.Lock()
	defer in.mu.Unlock()
	s := in.st(feature)
	if s.plan != nil && !s.busy {
		s.plan, s.phase = nil, phaseWithout(s)
		in.log("info", "install.plan_cancelled", feature, "the person did not approve the install")
	}
}

// Approve installs exactly the approved plan. It runs synchronously; callers
// run it in the background and poll View for progress.
func (in *Installer) Approve(ctx context.Context, src Source, feature, planID string) error {
	in.mu.Lock()
	s := in.st(feature)
	if s.busy {
		in.mu.Unlock()
		return errors.New("an install is already running for this feature")
	}
	p := s.plan
	if p == nil || p.ID != planID {
		in.mu.Unlock()
		return errors.New("that install plan is no longer current; check again")
	}
	if time.Since(p.CreatedAt) > planTTL {
		s.plan, s.phase = nil, phaseWithout(s)
		in.mu.Unlock()
		return in.fail(feature, Problem{Code: PPlanExpired, Message: "The approval is more than 10 minutes old.", NextStep: "Check again and approve the fresh plan."})
	}
	s.busy, s.phase, s.got, s.size = true, Downloading, 0, p.Size
	in.mu.Unlock()
	approvedAt := time.Now().UTC()
	in.log("info", "install.approved", feature, fmt.Sprintf("the person approved %s %s (sha256 %s…)", p.Tag, p.Asset, p.SHA256[:12]))

	// The release must still be exactly what was approved.
	r, err := src.Release(ctx, feature, p.Platform)
	if err != nil {
		return in.fail(feature, problemFromIdentity(err))
	}
	if code, err := CheckRelease(feature, p.Platform, p.Contract, r); err != nil {
		return in.fail(feature, Problem{Code: code, Message: err.Error(), NextStep: "Nothing was installed."})
	}
	if r.Tag != p.Tag || r.SHA256 != p.SHA256 || r.Size != p.Size || r.Asset != p.Asset {
		return in.fail(feature, Problem{Code: PPlanExpired, Message: "The release changed after you approved it.", NextStep: "Check again and approve the new plan."})
	}

	dir := filepath.Join(in.root, feature)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return in.fail(feature, Problem{Code: PDisk, Message: err.Error(), NextStep: "Check free disk space and folder permissions."})
	}
	tmp := filepath.Join(dir, "download-"+store.RandHex(6)+".partial")
	err = in.download(ctx, src, feature, p, tmp)
	if err != nil {
		os.Remove(tmp)
		var pr *Problem
		if errors.As(err, &pr) {
			return in.fail(feature, *pr)
		}
		return in.fail(feature, Problem{Code: PDisk, Message: err.Error(), NextStep: "Try again."})
	}

	in.mu.Lock()
	s.phase = Verifying
	in.mu.Unlock()
	final := filepath.Join(dir, p.Tag, p.Asset)
	if err := os.MkdirAll(filepath.Dir(final), 0o700); err != nil {
		os.Remove(tmp)
		return in.fail(feature, Problem{Code: PDisk, Message: err.Error(), NextStep: "Check folder permissions."})
	}
	os.Remove(final)
	if err := os.Rename(tmp, final); err != nil {
		os.Remove(tmp)
		return in.fail(feature, Problem{Code: PDisk, Message: err.Error(), NextStep: "Check folder permissions."})
	}
	if runtime.GOOS != "windows" {
		os.Chmod(final, 0o700)
	}
	// Hash again from disk: what is installed is what was approved.
	if sum, n, err := hashFile(final); err != nil || sum != p.SHA256 || n != p.Size {
		os.Remove(final)
		return in.fail(feature, Problem{Code: PChecksum, Message: "The file changed while it was being installed.", NextStep: "Try again."})
	}
	rec := Record{Feature: feature, Name: p.Name, Tag: p.Tag, Asset: p.Asset, Platform: p.Platform, Size: p.Size, SHA256: p.SHA256,
		Repo: p.Repo, ReleaseURL: p.ReleaseURL, Uploader: p.Uploader, ContractSchema: p.Contract.Schema, Checks: p.Checks,
		NotChecked: p.NotChecked, DecisionID: p.DecisionID, IdentitySystemID: src.Pairing().IdentitySystemID,
		ApprovedAt: approvedAt, InstalledAt: time.Now().UTC()}
	if err := store.WriteJSON(filepath.Join(dir, p.Tag, "install.json"), rec); err != nil {
		return in.fail(feature, Problem{Code: PDisk, Message: err.Error(), NextStep: "Try again."})
	}
	if err := store.WriteJSON(filepath.Join(dir, "current.json"), map[string]string{"tag": p.Tag}); err != nil {
		return in.fail(feature, Problem{Code: PDisk, Message: err.Error(), NextStep: "Try again."})
	}
	rec.Path = final
	in.mu.Lock()
	s.record, s.plan, s.busy, s.phase, s.problem, s.verify, s.verified = &rec, nil, false, Installed, nil, "OK", time.Now()
	in.mu.Unlock()
	in.log("info", "install.installed", feature, fmt.Sprintf("%s %s installed and verified (sha256 %s)", p.Tag, p.Asset, p.SHA256))
	return nil
}

type progressWriter struct {
	in *Installer
	s  *featureState
}

func (w progressWriter) Write(b []byte) (int, error) {
	w.in.mu.Lock()
	w.s.got += int64(len(b))
	w.in.mu.Unlock()
	return len(b), nil
}

func (in *Installer) download(ctx context.Context, src Source, feature string, p *Plan, tmp string) error {
	body, _, err := src.Download(ctx, src.DownloadPath(feature, p.Platform, p.Tag))
	if err != nil {
		pr := problemFromIdentity(err)
		return &pr
	}
	defer body.Close()
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	h := sha256.New()
	in.mu.Lock()
	s := in.st(feature)
	in.mu.Unlock()
	n, cerr := io.Copy(io.MultiWriter(f, h, progressWriter{in, s}), io.LimitReader(body, p.Size+1))
	if err := f.Close(); err != nil && cerr == nil {
		cerr = err
	}
	switch {
	case n < p.Size || (cerr != nil && n <= p.Size):
		return &Problem{Code: PInterrupted, Message: fmt.Sprintf("The download stopped after %d of %d bytes. The unfinished file was deleted.", n, p.Size), NextStep: "Press Install again to retry."}
	case n > p.Size:
		return &Problem{Code: PChecksum, Message: "The download was larger than the approved file. It was deleted.", NextStep: "Nothing was installed. Tell the publisher."}
	}
	if got := hex.EncodeToString(h.Sum(nil)); got != p.SHA256 {
		in.log("warn", "install.checksum_mismatch", feature, "downloaded sha256 "+got+" != approved "+p.SHA256)
		return &Problem{Code: PChecksum, Message: "The downloaded file doesn't match the approved SHA-256. It was deleted and nothing was installed.", NextStep: "Try again later. If it repeats, the release or the connection is damaged — tell the publisher."}
	}
	return nil
}

func (p *Problem) Error() string { return p.Code + ": " + p.Message }

// Remove deletes an installed feature (only files Processor installed).
func (in *Installer) Remove(feature string) error {
	if !FeatureRE.MatchString(feature) {
		return errors.New("bad feature id")
	}
	in.mu.Lock()
	s := in.st(feature)
	if s.busy {
		in.mu.Unlock()
		return errors.New("an install is running")
	}
	in.mu.Unlock()
	if err := os.RemoveAll(filepath.Join(in.root, feature)); err != nil {
		return err
	}
	in.mu.Lock()
	delete(in.fs, feature)
	in.mu.Unlock()
	in.log("info", "install.removed", feature, "installed files removed by the person")
	return nil
}
