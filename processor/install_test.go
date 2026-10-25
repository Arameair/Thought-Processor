package main

// Installer tests against a TEST FIXTURE Identity that speaks the same
// pairing/access/release/download protocol as Arameair/identity (verified
// against the real Identity binary in the end-to-end demo, not here).

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

type fakeIdentity struct {
	srv      *httptest.Server
	mu       sync.Mutex
	token    string
	state    string // access state
	file     []byte
	sha      string // published sha (may be wrong on purpose)
	badProof bool
}

func newFakeIdentity(t *testing.T) *fakeIdentity {
	f := &fakeIdentity{token: strings.Repeat("ab", 32), state: "ALLOWED", file: []byte("ENGINE BINARY (fixture)")}
	s := sha256.Sum256(f.file)
	f.sha = hex.EncodeToString(s[:])
	auth := func(w http.ResponseWriter, r *http.Request) bool {
		if r.Header.Get("Authorization") != "Bearer "+f.token {
			w.WriteHeader(401)
			return false
		}
		return true
	}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/pair", func(w http.ResponseWriter, r *http.Request) {
		var b map[string]string
		json.NewDecoder(r.Body).Decode(&b)
		if b["code"] != "ABCD-EFGH" || r.Header.Get("Origin") != "" {
			w.WriteHeader(401)
			w.Write([]byte(`{"error":"that pairing code is wrong or expired"}`))
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"token": f.token, "identity_system_id": "idn_fixture000000000000000"})
	})
	mux.HandleFunc("GET /api/v1/features/thought-engine/access", func(w http.ResponseWriter, r *http.Request) {
		if !auth(w, r) {
			return
		}
		f.mu.Lock()
		defer f.mu.Unlock()
		d := AccessDecision{Feature: "thought-engine", State: f.state, Allowed: f.state == "ALLOWED", Reason: "fixture reason", NextStep: "fixture next",
			Account: &AccessAccount{Login: "fixture-user", ID: 7}, Rule: "repo-visibility-v0", CheckedAt: time.Now().UTC(), DecisionID: "dec_fixture", IdentitySystemID: "idn_fixture000000000000000"}
		d.Proof = proof(f.token, d.canonical())
		if f.badProof {
			d.Proof = strings.Repeat("0", 64)
		}
		json.NewEncoder(w).Encode(d)
	})
	mux.HandleFunc("GET /api/v1/features/thought-engine/release", func(w http.ResponseWriter, r *http.Request) {
		if !auth(w, r) {
			return
		}
		f.mu.Lock()
		defer f.mu.Unlock()
		ri := ReleaseInfo{DecisionID: "dec_fixture", Tag: "engine-v0.1.0", Platform: r.URL.Query().Get("platform"), Asset: "thought-engine-" + Platform() + map[bool]string{true: ".exe"}[strings.HasPrefix(Platform(), "windows")],
			Size: int64(len(f.file)), SHA256: f.sha}
		ri.Proof = proof(f.token, ri.canonical())
		json.NewEncoder(w).Encode(ri)
	})
	mux.HandleFunc("GET /api/v1/features/thought-engine/release/download", func(w http.ResponseWriter, r *http.Request) {
		if !auth(w, r) {
			return
		}
		f.mu.Lock()
		defer f.mu.Unlock()
		w.Header().Set("X-Release-Tag", "engine-v0.1.0")
		w.Header().Set("Content-Length", strconv.Itoa(len(f.file)))
		w.Write(f.file)
	})
	f.srv = httptest.NewServer(mux)
	t.Cleanup(f.srv.Close)
	return f
}

func waitIdle(t *testing.T, in *Installer) InstallView {
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if v := in.View(); !v.Busy {
			return v
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("install never finished")
	return InstallView{}
}

func TestPairCheckInstallVerifyUninstall(t *testing.T) {
	f := newFakeIdentity(t)
	ctx := context.Background()
	if _, err := IdentityPair(ctx, f.srv.URL, "WRON-GCOD", "tp_0123456789abcdef", "x"); err == nil {
		t.Fatal("wrong code paired")
	}
	pr, err := IdentityPair(ctx, f.srv.URL, "ABCD-EFGH", "tp_0123456789abcdef", "x")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	in := NewInstaller(dir)
	d, err := in.Check(ctx, pr)
	if err != nil || !d.Allowed || in.View().Release == nil {
		t.Fatalf("check: %+v %v", d, err)
	}
	var logs []string
	in.Install(ctx, pr, func(c, d string) { logs = append(logs, c) })
	v := waitIdle(t, in)
	if v.Phase != PhInstalled || v.Installed == nil || v.Installed.SHA256 != f.sha || v.Installed.DecisionID != "dec_fixture" {
		t.Fatalf("install: %+v", v)
	}
	exe := filepath.Join(dir, "features", "thought-engine", "engine-v0.1.0", v.Installed.Asset)
	if b, _ := os.ReadFile(exe); string(b) != string(f.file) {
		t.Fatal("installed bytes differ")
	}
	if matches, _ := filepath.Glob(filepath.Join(dir, "features", "thought-engine", "*", "*.partial")); len(matches) != 0 {
		t.Fatal("partial left behind")
	}
	// Survives restart.
	if NewInstaller(dir).View().Installed == nil {
		t.Fatal("install record not reloaded")
	}
	// Already installed.
	in.Install(ctx, pr, func(string, string) {})
	if v := waitIdle(t, in); v.Problem == nil || v.Problem.Code != "ALREADY_INSTALLED" {
		t.Fatalf("already installed: %+v", v.Problem)
	}
	// Tamper → files changed.
	os.WriteFile(exe, []byte("tampered"), 0o700)
	if ok, _ := in.Verify(); ok || in.View().Verify != "FILES_CHANGED" {
		t.Fatal("tamper not detected")
	}
	if err := in.Uninstall(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(exe); !os.IsNotExist(err) {
		t.Fatal("uninstall left the program")
	}
}

func TestChecksumMismatchInstallsNothing(t *testing.T) {
	f := newFakeIdentity(t)
	f.sha = strings.Repeat("1", 64)
	pr, _ := IdentityPair(context.Background(), f.srv.URL, "ABCD-EFGH", "tp_0123456789abcdef", "x")
	dir := t.TempDir()
	in := NewInstaller(dir)
	in.Install(context.Background(), pr, func(string, string) {})
	v := waitIdle(t, in)
	if v.Problem == nil || v.Problem.Code != "CHECKSUM_MISMATCH" || v.Installed != nil {
		t.Fatalf("mismatch: %+v", v)
	}
	files, _ := filepath.Glob(filepath.Join(dir, "features", "thought-engine", "*", "*"))
	if len(files) != 0 {
		t.Fatalf("files left after mismatch: %v", files)
	}
}

func TestNotAllowedNeverDownloads(t *testing.T) {
	f := newFakeIdentity(t)
	f.state = "NOT_ALLOWED"
	pr, _ := IdentityPair(context.Background(), f.srv.URL, "ABCD-EFGH", "tp_0123456789abcdef", "x")
	in := NewInstaller(t.TempDir())
	var logs []string
	in.Install(context.Background(), pr, func(c, d string) { logs = append(logs, c) })
	v := waitIdle(t, in)
	if v.Problem == nil || v.Problem.Code != "NOT_ALLOWED" || v.Release != nil {
		t.Fatalf("not allowed: %+v", v)
	}
	for _, l := range logs {
		if strings.Contains(l, "download") {
			t.Fatal("download attempted for a denied account")
		}
	}
}

func TestForgedAnswerRejected(t *testing.T) {
	f := newFakeIdentity(t)
	pr, _ := IdentityPair(context.Background(), f.srv.URL, "ABCD-EFGH", "tp_0123456789abcdef", "x")
	f.badProof = true
	if _, err := IdentityAccess(context.Background(), pr, true); err == nil || err.(*IdentityError).Code != "PROOF_FAILED" {
		t.Fatalf("forged decision accepted: %v", err)
	}
	pr.Token = strings.Repeat("cd", 32)
	f.badProof = false
	if _, err := IdentityAccess(context.Background(), pr, true); err == nil || err.(*IdentityError).Code != "NOT_PAIRED" {
		t.Fatalf("unknown token: %v", err)
	}
	pr.IdentityAddress = "http://127.0.0.1:1"
	if _, err := IdentityAccess(context.Background(), pr, true); err == nil || err.(*IdentityError).Code != "IDENTITY_UNREACHABLE" {
		t.Fatalf("unreachable: %v", err)
	}
}

func TestPairingFileAndInstanceID(t *testing.T) {
	p, err := OpenProfile(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	id1, _ := p.ProcessorID()
	id2, _ := p.ProcessorID()
	if id1 != id2 || !processorIDRE.MatchString(id1) {
		t.Fatal("instance id not stable")
	}
	tok := strings.Repeat("ef", 32)
	p.SavePairing(Pairing{IdentityAddress: "http://127.0.0.1:7790", Token: tok})
	fi, _ := os.Stat(p.pairingPath())
	if runtime.GOOS != "windows" && fi.Mode().Perm() != 0o600 {
		t.Fatalf("pairing file mode %v", fi.Mode())
	}
	if pr, _ := p.LoadPairing(); pr == nil || pr.Token != tok {
		t.Fatal("pairing not reloaded")
	}
	os.WriteFile(p.pairingPath(), []byte("{bad"), 0o600)
	if _, err := p.LoadPairing(); err == nil {
		t.Fatal("damaged pairing accepted")
	}
}
