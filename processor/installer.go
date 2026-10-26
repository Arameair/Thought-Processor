package main

// Installing Thought Engine — only when the paired Identity says ALLOWED.
//
//	<profile>/features/thought-engine/<tag>/<asset>          the program
//	<profile>/features/thought-engine/<tag>/install.json     what/when/on whose authority
//	<profile>/features/thought-engine/current.json           which tag is current
//
// The file is written as <asset>.partial while it is hashed, compared with the
// SHA-256 Identity reported (from the release's SHA256SUMS, HMAC-verified),
// and only then renamed into place. No admin rights; nothing outside the
// profile folder is touched. Installing does NOT start the Engine.

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
	"strings"
	"sync"
	"time"
)

const maxEngineBytes = 200 << 20

var (
	tagRE   = regexp.MustCompile(`^[A-Za-z0-9._-]{1,64}$`)
	assetRE = regexp.MustCompile(`^thought-engine-(windows|linux|darwin)-(amd64|arm64)(\.exe)?$`)
	shaRE   = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

func Platform() string { return runtime.GOOS + "-" + runtime.GOARCH }

type InstallRecord struct {
	Feature          string    `json:"feature"`
	Tag              string    `json:"tag"`
	Asset            string    `json:"asset"`
	SHA256           string    `json:"sha256"`
	Size             int64     `json:"size"`
	Path             string    `json:"path"`
	InstalledAt      time.Time `json:"installed_at"`
	DecisionID       string    `json:"decision_id"`
	Rule             string    `json:"rule"`
	Account          string    `json:"account"`
	IdentitySystemID string    `json:"identity_system_id"`
}

// Installer phases.
const (
	PhIdle        = "IDLE"
	PhChecking    = "CHECKING"
	PhDownloading = "DOWNLOADING"
	PhVerifying   = "VERIFYING"
	PhInstalled   = "INSTALLED"
	PhFailed      = "FAILED"
)

type InstallProblem struct {
	Code     string `json:"code"`
	Message  string `json:"message"`
	NextStep string `json:"next_step"`
}

type Installer struct {
	dir string // <profile>/features/thought-engine

	mu        sync.Mutex
	phase     string
	got, size int64
	access    *AccessDecision
	release   *ReleaseInfo
	installed *InstallRecord
	problem   *InstallProblem
	verify    string // OK | FILES_CHANGED | ""
	busy      bool
}

func NewInstaller(profileDir string) *Installer {
	in := &Installer{dir: filepath.Join(profileDir, "features", "thought-engine"), phase: PhIdle}
	if rec, err := in.loadCurrent(); err == nil && rec != nil {
		in.installed, in.phase = rec, PhInstalled
	}
	return in
}

func (in *Installer) loadCurrent() (*InstallRecord, error) {
	b, err := os.ReadFile(filepath.Join(in.dir, "current.json"))
	if err != nil {
		return nil, err
	}
	var cur struct {
		Tag string `json:"tag"`
	}
	if json.Unmarshal(b, &cur) != nil || !tagRE.MatchString(cur.Tag) {
		return nil, errors.New("current.json damaged")
	}
	b, err = os.ReadFile(filepath.Join(in.dir, cur.Tag, "install.json"))
	if err != nil {
		return nil, err
	}
	var rec InstallRecord
	if json.Unmarshal(b, &rec) != nil || !assetRE.MatchString(rec.Asset) || !shaRE.MatchString(rec.SHA256) {
		return nil, errors.New("install.json damaged")
	}
	rec.Path = filepath.Join(in.dir, cur.Tag, rec.Asset)
	return &rec, nil
}

type InstallView struct {
	Phase     string          `json:"phase"`
	Got       int64           `json:"got,omitempty"`
	Size      int64           `json:"size,omitempty"`
	Access    *AccessDecision `json:"access,omitempty"`
	Release   *ReleaseInfo    `json:"release,omitempty"`
	Installed *InstallRecord  `json:"installed,omitempty"`
	Problem   *InstallProblem `json:"problem,omitempty"`
	Verify    string          `json:"verify,omitempty"`
	Busy      bool            `json:"busy"`
	Platform  string          `json:"platform"`
}

func (in *Installer) View() InstallView {
	in.mu.Lock()
	defer in.mu.Unlock()
	v := InstallView{Phase: in.phase, Got: in.got, Size: in.size, Problem: in.problem, Verify: in.verify, Busy: in.busy, Platform: Platform()}
	if in.access != nil {
		a := *in.access
		a.Proof = "" // never shown or copied
		v.Access = &a
	}
	if in.release != nil {
		r := *in.release
		r.Proof = ""
		v.Release = &r
	}
	if in.installed != nil {
		r := *in.installed
		r.Path = displayPath(r.Path)
		v.Installed = &r
	}
	return v
}

func (in *Installer) setProblem(code, msg, next string) {
	in.mu.Lock()
	in.problem = &InstallProblem{Code: code, Message: msg, NextStep: next}
	if in.installed != nil {
		in.phase = PhInstalled
	} else {
		in.phase = PhFailed
	}
	in.mu.Unlock()
}

// problemFor turns an Identity error into user wording.
func problemFor(err error) (string, string, string) {
	var ie *IdentityError
	if errors.As(err, &ie) {
		switch ie.Code {
		case "IDENTITY_UNREACHABLE":
			return ie.Code, ie.Msg, "Start Identity, then press Check access again."
		case "NOT_PAIRED":
			return ie.Code, ie.Msg, "In Identity open Manage → Thought Processor → Pair Thought Processor, then enter the code here."
		case "PROOF_FAILED":
			return ie.Code, ie.Msg, "Unpair here, then pair again from the Identity you trust."
		case "NO_RELEASE":
			return ie.Code, ie.Msg, "Arameair needs to publish a Thought Engine release. Nothing was installed."
		}
		return ie.Code, ie.Msg, "Press Check access again; if it repeats, use Copy context."
	}
	return "ERROR", err.Error(), "Press Check access again; if it repeats, use Copy context."
}

// Check asks Identity for a fresh decision (and the latest release if allowed).
func (in *Installer) Check(ctx context.Context, pr Pairing) (AccessDecision, error) {
	in.mu.Lock()
	in.problem = nil
	in.mu.Unlock()
	d, err := IdentityAccess(ctx, pr, true)
	if err != nil {
		code, msg, next := problemFor(err)
		in.mu.Lock()
		in.access = nil
		in.mu.Unlock()
		in.setProblem(code, msg, next)
		return d, err
	}
	in.mu.Lock()
	in.access = &d
	in.mu.Unlock()
	if d.Allowed {
		if ri, err := IdentityRelease(ctx, pr, Platform()); err == nil {
			in.mu.Lock()
			in.release = &ri
			in.mu.Unlock()
		} else {
			code, msg, next := problemFor(err)
			in.setProblem(code, msg, next)
		}
	}
	return d, nil
}

// Install runs the whole install; call it in a goroutine. log receives fixed
// event codes only.
func (in *Installer) Install(ctx context.Context, pr Pairing, log func(code, detail string)) {
	in.mu.Lock()
	if in.busy {
		in.mu.Unlock()
		return
	}
	in.busy, in.phase, in.problem, in.got, in.size = true, PhChecking, nil, 0, 0
	in.mu.Unlock()
	defer func() { in.mu.Lock(); in.busy = false; in.mu.Unlock() }()

	d, err := IdentityAccess(ctx, pr, true)
	if err != nil {
		code, msg, next := problemFor(err)
		in.setProblem(code, msg, next)
		log("install_refused_"+strings.ToLower(code), "install stopped before download: Identity check failed")
		return
	}
	in.mu.Lock()
	in.access = &d
	in.mu.Unlock()
	if !d.Allowed {
		in.setProblem("NOT_ALLOWED", d.Reason, d.NextStep)
		log("install_refused_not_allowed", "install refused before download: Identity decision "+d.DecisionID+" is "+d.State)
		return
	}
	ri, err := IdentityRelease(ctx, pr, Platform())
	if err != nil {
		code, msg, next := problemFor(err)
		in.setProblem(code, msg, next)
		log("install_no_release", "install stopped: release details unavailable")
		return
	}
	if !tagRE.MatchString(ri.Tag) || !assetRE.MatchString(ri.Asset) || !shaRE.MatchString(ri.SHA256) || ri.Size <= 0 || ri.Size > maxEngineBytes {
		in.setProblem("BAD_RELEASE", "The release details are not acceptable (name, checksum or size). Nothing was installed.", "Report it with Copy context.")
		log("install_bad_release", "install refused: release details failed validation")
		return
	}
	in.mu.Lock()
	in.release = &ri
	if in.installed != nil && in.installed.Tag == ri.Tag {
		in.mu.Unlock()
		if ok, _ := in.Verify(); ok {
			in.setProblem("ALREADY_INSTALLED", fmt.Sprintf("Thought Engine %s is already installed and its files match.", ri.Tag), "Nothing to do. Use Reinstall only if something seems wrong.")
			log("install_already", "already installed: "+ri.Tag)
			return
		}
		in.mu.Lock()
	}
	in.phase, in.size = PhDownloading, ri.Size
	in.mu.Unlock()
	log("install_download_started", "downloading "+ri.Tag+" "+ri.Asset+" through Identity (decision "+d.DecisionID+")")

	rec, err := in.download(ctx, pr, ri)
	if err != nil {
		var pe *InstallProblem
		if errors.As(err, &pe) {
			in.setProblem(pe.Code, pe.Message, pe.NextStep)
		} else {
			code, msg, next := problemFor(err)
			in.setProblem(code, msg, next)
		}
		log("install_failed", "install failed; partial file removed, nothing installed")
		return
	}
	rec.DecisionID, rec.Rule, rec.IdentitySystemID = d.DecisionID, d.Rule, d.IdentitySystemID
	if d.Account != nil {
		rec.Account = d.Account.Login
	}
	if err := writeJSONAtomic(filepath.Join(in.dir, ri.Tag, "install.json"), rec); err == nil {
		err = writeJSONAtomic(filepath.Join(in.dir, "current.json"), map[string]string{"tag": ri.Tag})
	}
	if err != nil {
		in.setProblem("DISK", "Processor couldn't save the install record: "+err.Error(), "Free disk space or check folder permissions, then install again.")
		return
	}
	in.mu.Lock()
	in.installed, in.phase, in.verify = &rec, PhInstalled, "OK"
	in.mu.Unlock()
	log("install_done", fmt.Sprintf("installed %s %s (%d bytes, sha256 %s…) under decision %s", ri.Tag, ri.Asset, ri.Size, ri.SHA256[:12], d.DecisionID))
}

func (p *InstallProblem) Error() string { return p.Message }

func (in *Installer) download(ctx context.Context, pr Pairing, ri ReleaseInfo) (InstallRecord, error) {
	dir := filepath.Join(in.dir, ri.Tag)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return InstallRecord{}, &InstallProblem{"DISK", "Processor couldn't create its install folder: " + err.Error(), "Check folder permissions."}
	}
	final := filepath.Join(dir, ri.Asset)
	partial := final + ".partial"
	body, err := IdentityDownload(ctx, pr, ri.Platform, ri.Tag)
	if err != nil {
		return InstallRecord{}, err
	}
	defer body.Close()
	f, err := os.OpenFile(partial, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o700)
	if err != nil {
		return InstallRecord{}, &InstallProblem{"DISK", "Processor couldn't write to its folder: " + err.Error(), "Free disk space or check folder permissions."}
	}
	h := sha256.New()
	n, cerr := io.Copy(io.MultiWriter(f, h, progressWriter{in}), io.LimitReader(body, ri.Size+1))
	ferr := f.Close()
	fail := func(code, msg, next string) (InstallRecord, error) {
		os.Remove(partial)
		return InstallRecord{}, &InstallProblem{code, msg, next}
	}
	if cerr != nil || ferr != nil {
		return fail("DOWNLOAD_FAILED", "The download stopped before it finished. Nothing was installed.", "Press Install again.")
	}
	in.mu.Lock()
	in.phase = PhVerifying
	in.mu.Unlock()
	sum := hex.EncodeToString(h.Sum(nil))
	if n != ri.Size || sum != ri.SHA256 {
		return fail("CHECKSUM_MISMATCH", fmt.Sprintf("The downloaded file doesn't match the published checksum (got %d bytes, sha256 %s…), so it was deleted and NOT installed.", n, sum[:12]),
			"Try again. If it repeats, report it with Copy context.")
	}
	if err := os.Rename(partial, final); err != nil {
		return fail("DISK", "Processor couldn't put the file in place: "+err.Error(), "Check folder permissions.")
	}
	return InstallRecord{Feature: "thought-engine", Tag: ri.Tag, Asset: ri.Asset, SHA256: sum, Size: n, Path: final, InstalledAt: time.Now().UTC()}, nil
}

type progressWriter struct{ in *Installer }

func (p progressWriter) Write(b []byte) (int, error) {
	p.in.mu.Lock()
	p.in.got += int64(len(b))
	p.in.mu.Unlock()
	return len(b), nil
}

// Verify re-hashes the installed program and compares it with the record.
func (in *Installer) Verify() (bool, error) {
	in.mu.Lock()
	rec := in.installed
	in.mu.Unlock()
	if rec == nil {
		return false, errors.New("nothing installed")
	}
	f, err := os.Open(rec.Path)
	if err != nil {
		in.mu.Lock()
		in.verify = "FILES_CHANGED"
		in.mu.Unlock()
		return false, nil
	}
	defer f.Close()
	h := sha256.New()
	io.Copy(h, io.LimitReader(f, maxEngineBytes+1))
	ok := hex.EncodeToString(h.Sum(nil)) == rec.SHA256
	in.mu.Lock()
	in.verify = map[bool]string{true: "OK", false: "FILES_CHANGED"}[ok]
	in.mu.Unlock()
	return ok, nil
}

// Uninstall removes the installed program folder (only inside this profile).
func (in *Installer) Uninstall() error {
	in.mu.Lock()
	rec := in.installed
	busy := in.busy
	in.mu.Unlock()
	if busy {
		return errors.New("an install is running")
	}
	if rec == nil {
		return errors.New("nothing installed")
	}
	dir := filepath.Join(in.dir, rec.Tag)
	if filepath.Dir(dir) != in.dir {
		return errors.New("refusing to remove an unexpected path")
	}
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	os.Remove(filepath.Join(in.dir, "current.json"))
	in.mu.Lock()
	in.installed, in.phase, in.verify, in.problem = nil, PhIdle, "", nil
	in.mu.Unlock()
	return nil
}
