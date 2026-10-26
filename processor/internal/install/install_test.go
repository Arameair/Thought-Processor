package install

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Arameair/Thought-Processor/processor/internal/idclient"
)

// fakeSource stands in for a paired Identity (TEST FIXTURE).
type fakeSource struct {
	data      []byte
	sum       string
	uploader  string
	tag       string
	serve     string // good | tamper | cut | refuse
	downloads int
}

func newFake() *fakeSource {
	d := bytes.Repeat([]byte("FAKE FEATURE BINARY — test fixture only\n"), 50)
	s := sha256.Sum256(d)
	return &fakeSource{data: d, sum: hex.EncodeToString(s[:]), uploader: "github-actions[bot]", tag: "engine-v0.1.0", serve: "good"}
}

func (f *fakeSource) Contract(ctx context.Context, feature string) (idclient.Contract, error) {
	var k idclient.Contract
	k.Schema, k.FeatureID, k.Name, k.Publisher = ContractSchema, feature, "Test Feature", "Arameair"
	k.Source = &struct {
		Kind             string   `json:"kind"`
		Repo             string   `json:"repo"`
		TagPrefix        string   `json:"tag_prefix"`
		AssetTemplate    string   `json:"asset_template"`
		Checksums        string   `json:"checksums"`
		TrustedUploaders []string `json:"trusted_uploaders"`
		AllowPrerelease  bool     `json:"allow_prerelease"`
	}{"github-release", "Arameair/Thought-engine", "engine-v", "thought-engine-{os}-{arch}{exe}", "SHA256SUMS", []string{"github-actions[bot]"}, false}
	k.Install = &struct {
		Kind     string `json:"kind"`
		MaxBytes int64  `json:"max_bytes"`
	}{"single-executable", 1 << 20}
	return k, nil
}

func (f *fakeSource) Release(ctx context.Context, feature, platform string) (idclient.Release, error) {
	if f.serve == "refuse" {
		return idclient.Release{}, &idclient.Error{Kind: idclient.Refused, Status: 404, Code: "NO_RELEASE", Msg: "no release yet"}
	}
	var r idclient.Release
	r.Feature, r.ContractSchema, r.Tag, r.Platform, r.Size, r.SHA256 = feature, ContractSchema, f.tag, platform, int64(len(f.data)), f.sum
	r.Asset = expandAsset("thought-engine-{os}-{arch}{exe}", platform)
	r.Source = &struct {
		Repo             string   `json:"repo"`
		Tag              string   `json:"tag"`
		URL              string   `json:"url"`
		AssetUploader    string   `json:"asset_uploader"`
		ChecksumUploader string   `json:"checksum_uploader"`
		Prerelease       bool     `json:"prerelease"`
		Checks           []string `json:"checks"`
		NotChecked       string   `json:"not_checked"`
	}{Repo: "Arameair/Thought-engine", Tag: f.tag, AssetUploader: f.uploader, ChecksumUploader: "github-actions[bot]", Checks: []string{"test"}}
	return r, nil
}

func (f *fakeSource) Download(ctx context.Context, path string) (io.ReadCloser, int64, error) {
	f.downloads++
	switch f.serve {
	case "tamper":
		d := append([]byte{f.data[0] ^ 0xff}, f.data[1:]...)
		return io.NopCloser(bytes.NewReader(d)), int64(len(d)), nil
	case "cut":
		return io.NopCloser(io.MultiReader(bytes.NewReader(f.data[:len(f.data)/2]), errReader{})), int64(len(f.data)), nil
	}
	return io.NopCloser(bytes.NewReader(f.data)), int64(len(f.data)), nil
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errors.New("connection reset") }

func (f *fakeSource) DownloadPath(feature, platform, tag string) string { return "/x" }
func (f *fakeSource) Pairing() idclient.Pairing {
	return idclient.Pairing{IdentitySystemID: "idn_test"}
}

func noPartials(t *testing.T, dir string) {
	t.Helper()
	parts, _ := filepath.Glob(filepath.Join(dir, "features", "*", "*.partial"))
	if len(parts) != 0 {
		t.Fatalf("unfinished download left behind: %v", parts)
	}
}

func TestInstallNeedsApprovalVerifiesAndPersists(t *testing.T) {
	dir := t.TempDir()
	in, src, ctx := New(dir, nil), newFake(), context.Background()
	p, err := in.Prepare(ctx, src, "thought-engine")
	if err != nil || p == nil || p.SHA256 != src.sum || in.View("thought-engine").Phase != Planned {
		t.Fatalf("plan: %v %+v", err, p)
	}
	if src.downloads != 0 {
		t.Fatal("downloaded before approval")
	}
	if err := in.Approve(ctx, src, "thought-engine", "plan_wrong"); err == nil {
		t.Fatal("approved a plan that wasn't shown")
	}
	if err := in.Approve(ctx, src, "thought-engine", p.ID); err != nil {
		t.Fatal(err)
	}
	v := in.View("thought-engine")
	if v.Phase != Installed || v.Verify != "OK" || v.Installed.SHA256 != src.sum || v.Installed.IdentitySystemID != "idn_test" {
		t.Fatalf("after install %+v", v)
	}
	noPartials(t, dir)
	// Restart: recognised and re-verified.
	in2 := New(dir, nil)
	if v := in2.View("thought-engine"); v.Phase != Installed || v.Verify != "OK" {
		t.Fatalf("after restart %+v", v)
	}
	// Same release again: nothing to do.
	if p, err := in2.Prepare(ctx, src, "thought-engine"); p != nil || err != nil {
		t.Fatalf("up-to-date install planned again: %+v %v", p, err)
	}
	// Someone changes the installed file: detected on next start.
	f := filepath.Join(dir, "features", "thought-engine", "engine-v0.1.0", v.Installed.Asset)
	os.WriteFile(f, []byte("changed"), 0o700)
	if v := New(dir, nil).View("thought-engine"); v.Verify != PFilesChanged || v.Problem == nil {
		t.Fatalf("changed file not detected: %+v", v)
	}
	os.Remove(f)
	if v := New(dir, nil).View("thought-engine"); v.Verify != PMissing {
		t.Fatalf("missing file not detected: %+v", v)
	}
	if err := in2.Remove("thought-engine"); err != nil {
		t.Fatal(err)
	}
	if v := New(dir, nil).View("thought-engine"); v.Phase != NotInstalled {
		t.Fatalf("after remove %+v", v)
	}
}

func TestBadDownloadsInstallNothing(t *testing.T) {
	ctx := context.Background()
	for _, mode := range []string{"tamper", "cut"} {
		dir := t.TempDir()
		in, src := New(dir, nil), newFake()
		p, _ := in.Prepare(ctx, src, "thought-engine")
		src.serve = mode
		err := in.Approve(ctx, src, "thought-engine", p.ID)
		v := in.View("thought-engine")
		want := map[string]string{"tamper": PChecksum, "cut": PInterrupted}[mode]
		if err == nil || v.Phase != Failed || v.Problem == nil || v.Problem.Code != want || v.Installed != nil {
			t.Fatalf("%s: %v %+v", mode, err, v)
		}
		noPartials(t, dir)
		// Recovery: a good download works afterwards.
		src.serve = "good"
		p, _ = in.Prepare(ctx, src, "thought-engine")
		if err := in.Approve(ctx, src, "thought-engine", p.ID); err != nil {
			t.Fatalf("%s: retry failed: %v", mode, err)
		}
	}
}

func TestUnapprovedSourcesRefusedBeforeDownload(t *testing.T) {
	ctx := context.Background()
	cases := map[string]func(*fakeSource){
		"uploaded by a person": func(f *fakeSource) { f.uploader = "someone" },
		"wrong tag pattern":    func(f *fakeSource) { f.tag = "v9-evil" },
	}
	for name, mod := range cases {
		in, src := New(t.TempDir(), nil), newFake()
		mod(src)
		if p, err := in.Prepare(ctx, src, "thought-engine"); err == nil || p != nil || !strings.Contains(err.Error(), PUnapproved) || src.downloads != 0 {
			t.Fatalf("%s: %v %+v", name, err, p)
		}
	}
	in, src := New(t.TempDir(), nil), newFake()
	src.serve = "refuse"
	if _, err := in.Prepare(ctx, src, "thought-engine"); err == nil || in.View("thought-engine").Problem.Code != PNoRelease {
		t.Fatalf("no release: %v", err)
	}
}

func TestReleaseChangedAfterApprovalAndLeftoverPartials(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	in, src := New(dir, nil), newFake()
	p, _ := in.Prepare(ctx, src, "thought-engine")
	src.tag = "engine-v0.2.0"
	if err := in.Approve(ctx, src, "thought-engine", p.ID); err == nil || !strings.Contains(err.Error(), PPlanExpired) {
		t.Fatalf("changed release installed: %v", err)
	}
	// A crash mid-download leaves a .partial; the next start removes it.
	os.MkdirAll(filepath.Join(dir, "features", "thought-engine"), 0o700)
	os.WriteFile(filepath.Join(dir, "features", "thought-engine", "download-x.partial"), []byte("half"), 0o600)
	New(dir, nil)
	noPartials(t, dir)
	// Expired plan.
	src.tag = "engine-v0.1.0"
	p, _ = in.Prepare(ctx, src, "thought-engine")
	in.mu.Lock()
	in.fs["thought-engine"].plan.CreatedAt = time.Now().Add(-time.Hour)
	in.mu.Unlock()
	if err := in.Approve(ctx, src, "thought-engine", p.ID); err == nil {
		t.Fatal("expired plan installed")
	}
}
