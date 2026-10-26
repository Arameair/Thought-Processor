// Package featpkg reads and verifies a local feature package
// (manifest schema thought.feature-manifest/1) and copies it into an
// installation folder. It never executes anything.
//
// P1 trust rule (cleanup, Grok M1): a package is installable and runnable
// only when BOTH
//   - it sits in a folder the operator explicitly trusted when starting
//     Processor (-trust-feature-dir), AND
//   - its exact content — the artifact digest over the manifest and every
//     file — is PINNED: either built into this build for the reviewed test
//     fixture (BuiltinFixturePins) or pinned by the operator with
//     -trust-feature-artifact <sha256>.
//
// The manifest's own "trust_class" is a description only; it can never
// grant trust. Identity applies the same pin list independently.
// This is "trusted local fixture" — NOT a verified third-party publisher
// and NOT OS isolation (both remain BLOCKED in P1).
package featpkg

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const (
	ManifestSchema = "thought.feature-manifest/1"
	FeatureAPI     = "thought.feature-api/1"
	TrustFixture   = "TRUSTED_LOCAL_TEST_FIXTURE"
	ManifestFile   = "feature.json"
	maxManifest    = 64 << 10
	maxFile        = 10 << 20
	maxFiles       = 32
)

var (
	FeatureIDRE = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,47}$`)
	capRE       = regexp.MustCompile(`^[a-z][a-z0-9_.-]{2,63}$`)
	versionRE   = regexp.MustCompile(`^[0-9A-Za-z][0-9A-Za-z.+-]{0,39}$`)
	sha256RE    = regexp.MustCompile(`^[0-9a-f]{64}$`)
	pathRE      = regexp.MustCompile(`^[A-Za-z0-9_.-]+(/[A-Za-z0-9_.-]+)*$`)
	// Interpreters a trusted fixture may name by bare name (resolved on PATH).
	interpreters = map[string]bool{"python3": true, "node": true}
)

type Capability struct {
	ID            string `json:"id"`
	Description   string `json:"description"`
	DataClass     string `json:"data_class"`
	RequiresGrant bool   `json:"requires_grant"`
}

type File struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

type Runtime struct {
	Command        []string `json:"command"`
	Requires       []string `json:"requires"`
	StartupModes   []string `json:"startup_modes"`
	DefaultStartup string   `json:"default_startup"`
	DefaultIdle    int      `json:"default_idle_seconds"`
	Recovery       string   `json:"recovery"`
}

type Manifest struct {
	Schema      string `json:"schema"`
	FeatureID   string `json:"feature_id"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Publisher   struct {
		Name  string `json:"name"`
		Proof string `json:"proof"`
	} `json:"publisher"`
	Source struct {
		Repo string `json:"repo"`
		Path string `json:"path"`
	} `json:"source"`
	TrustClass   string       `json:"trust_class"`
	API          string       `json:"api"`
	Runtime      Runtime      `json:"runtime"`
	Capabilities []Capability `json:"capabilities"`
	Standalone   struct {
		Health   bool `json:"health"`
		SelfTest bool `json:"self_test"`
	} `json:"standalone"`
	Sources []string `json:"information_sources"`
	Files   []File   `json:"files"`
}

// BuiltinFixturePins are the exact artifact digests of reviewed trusted
// local test fixtures shipped with this source. Identity carries the same
// list. TestBuiltinPinMatchesFixture fails if the fixture changes without
// its pin being updated (scripts/refresh_fixture_manifest.py prints it).
var BuiltinFixturePins = map[string]string{
	"3cd5e7a60f193f21bd15e14ee1532259fe1088434164ef6af4e3a66a36d493df": "built-in pin: p1-deterministic-worker 0.1.0 (reviewed P1 test fixture)",
}

// Pins is the set of artifact digests this Processor trusts.
type Pins struct{ m map[string]string }

// NewPins combines the built-in pins with operator pins (64 hex characters each).
func NewPins(operator []string) (Pins, error) {
	m := map[string]string{}
	for k, v := range BuiltinFixturePins {
		m[k] = v
	}
	for _, d := range operator {
		d = strings.ToLower(strings.TrimSpace(d))
		if !sha256RE.MatchString(d) {
			return Pins{}, fmt.Errorf("-trust-feature-artifact needs a 64-character SHA-256, got %q", d)
		}
		if _, ok := m[d]; !ok {
			m[d] = "operator pin (-trust-feature-artifact)"
		}
	}
	return Pins{m: m}, nil
}

// Basis says why digest is trusted ("" = not pinned).
func (p Pins) Basis(digest string) string { return p.m[digest] }

// Package is a verified package on disk.
type Package struct {
	Dir            string   `json:"dir"`
	Manifest       Manifest `json:"manifest"`
	ManifestSHA256 string   `json:"manifest_sha256"`
	ArtifactSHA256 string   `json:"artifact_sha256"`
	Trusted        bool     `json:"trusted"`
	TrustBasis     string   `json:"trust_basis,omitempty"`
	Blocked        string   `json:"blocked,omitempty"`
}

func sha(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }

func readRegular(path string, max int64) ([]byte, error) {
	fi, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("%s is a symbolic link; refused", filepath.Base(path))
	}
	if !fi.Mode().IsRegular() {
		return nil, fmt.Errorf("%s is not a regular file", filepath.Base(path))
	}
	if fi.Size() > max {
		return nil, fmt.Errorf("%s is too large", filepath.Base(path))
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(io.LimitReader(f, max+1))
}

// insideNoLinks verifies that rel, inside dir, crosses no symbolic link at
// any level (a linked parent folder could point anywhere on the computer).
func insideNoLinks(dir, rel string) error {
	cur := dir
	for _, part := range strings.Split(rel, "/") {
		cur = filepath.Join(cur, part)
		fi, err := os.Lstat(cur)
		if err != nil {
			return err
		}
		if fi.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("%s crosses a symbolic link; refused", rel)
		}
	}
	return nil
}

// safeRel checks a manifest path stays inside the package.
func safeRel(p string) error {
	if !pathRE.MatchString(p) || strings.Contains(p, "..") || strings.HasPrefix(p, "/") || p == ManifestFile {
		return fmt.Errorf("file path %q is not allowed (relative, no .., no special characters)", p)
	}
	return nil
}

// Validate checks the manifest's shape.
func (m *Manifest) Validate() error {
	switch {
	case m.Schema != ManifestSchema:
		return fmt.Errorf("schema must be %s", ManifestSchema)
	case !FeatureIDRE.MatchString(m.FeatureID):
		return errors.New("feature_id must be 2–48 of a-z 0-9 -")
	case m.Name == "" || len(m.Name) > 80:
		return errors.New("name must be 1–80 characters")
	case !versionRE.MatchString(m.Version):
		return errors.New("version is not valid")
	case m.API != FeatureAPI:
		return fmt.Errorf("api must be %s", FeatureAPI)
	case len(m.Files) == 0 || len(m.Files) > maxFiles:
		return errors.New("files must list 1–32 files")
	case len(m.Runtime.Command) == 0 || len(m.Runtime.Command) > 16:
		return errors.New("runtime.command must have 1–16 parts")
	case len(m.Capabilities) > 16:
		return errors.New("too many capabilities")
	case len(m.Sources) != 0:
		return errors.New("P1 features may not declare information sources")
	}
	listed := map[string]bool{}
	for _, f := range m.Files {
		if err := safeRel(f.Path); err != nil {
			return err
		}
		if !sha256RE.MatchString(f.SHA256) || listed[f.Path] {
			return fmt.Errorf("file %s: sha256 missing or listed twice", f.Path)
		}
		listed[f.Path] = true
	}
	for i, part := range m.Runtime.Command {
		if strings.ContainsAny(part, "\x00\n") || len(part) > 200 {
			return errors.New("runtime.command contains an invalid part")
		}
		if strings.HasPrefix(part, "{package}/") {
			if !listed[strings.TrimPrefix(part, "{package}/")] {
				return fmt.Errorf("runtime.command refers to %s, which is not a listed, checked file", part)
			}
		} else if i == 0 && !interpreters[part] {
			return fmt.Errorf("runtime.command must start with a listed package file or one of: python3, node (got %q)", part)
		} else if strings.Contains(part, "{") || strings.HasPrefix(part, "/") || strings.Contains(part, "..") {
			return fmt.Errorf("runtime.command part %q is not allowed", part)
		}
	}
	seen := map[string]bool{}
	for _, c := range m.Capabilities {
		if !capRE.MatchString(c.ID) || seen[c.ID] {
			return errors.New("capability ids must be unique, 3–64 of a-z 0-9 _ . -")
		}
		seen[c.ID] = true
	}
	switch m.Runtime.DefaultStartup {
	case "", "manual", "on_demand", "always_running":
	default:
		return errors.New("runtime.default_startup must be manual, on_demand or always_running")
	}
	return nil
}

// ArtifactDigest is the identity of the exact package content.
func ArtifactDigest(manifestSHA string, files []File) string {
	fs := append([]File(nil), files...)
	sort.Slice(fs, func(i, j int) bool { return fs[i].Path < fs[j].Path })
	var b bytes.Buffer
	b.WriteString("thought.feature-package/1\nmanifest " + manifestSHA + "\n")
	for _, f := range fs {
		b.WriteString(f.SHA256 + "  " + f.Path + "\n")
	}
	return sha(b.Bytes())
}

// Load reads and verifies a package directory. trusted says whether the
// operator listed this directory with -trust-feature-dir; pins decide
// whether this exact content may be installed and run.
func Load(dir string, trusted bool, pins Pins) (Package, error) {
	dir = filepath.Clean(dir)
	fi, err := os.Lstat(dir)
	if err != nil {
		return Package{}, err
	}
	if fi.Mode()&os.ModeSymlink != 0 || !fi.IsDir() {
		return Package{}, errors.New("package folder must be a real folder, not a link")
	}
	raw, err := readRegular(filepath.Join(dir, ManifestFile), maxManifest)
	if err != nil {
		return Package{}, fmt.Errorf("manifest: %w", err)
	}
	var m Manifest
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&m); err != nil {
		return Package{}, fmt.Errorf("manifest is not valid %s: %v", ManifestSchema, err)
	}
	if err := m.Validate(); err != nil {
		return Package{}, err
	}
	for _, f := range m.Files {
		if err := insideNoLinks(dir, f.Path); err != nil {
			return Package{}, fmt.Errorf("file %s: %w", f.Path, err)
		}
		b, err := readRegular(filepath.Join(dir, filepath.FromSlash(f.Path)), maxFile)
		if err != nil {
			return Package{}, fmt.Errorf("file %s: %w", f.Path, err)
		}
		if sha(b) != f.SHA256 {
			return Package{}, fmt.Errorf("FILES_CHANGED: %s does not match its SHA-256 in the manifest", f.Path)
		}
	}
	p := Package{Dir: dir, Manifest: m, ManifestSHA256: sha(raw), Trusted: trusted}
	p.ArtifactSHA256 = ArtifactDigest(p.ManifestSHA256, m.Files)
	p.TrustBasis = pins.Basis(p.ArtifactSHA256)
	switch {
	case !trusted:
		p.Blocked = "UNTRUSTED_EXECUTION_BLOCKED: this folder was not trusted by the operator (-trust-feature-dir)"
	case p.TrustBasis == "":
		p.Blocked = "UNTRUSTED_EXECUTION_BLOCKED: this exact package content (artifact " + p.ArtifactSHA256[:12] + "…) is not a pinned trusted fixture. " +
			"What a manifest says about itself (trust_class " + strings.TrimSpace(m.TrustClass) + ") cannot grant trust. Third-party code stays blocked until verified provenance and OS sandboxing exist."
	}
	return p, nil
}

// Stage copies the verified package into dst (which must not exist), then
// re-verifies every copied file. dst is removed again on any failure.
func Stage(p Package, dst string) error {
	if _, err := os.Lstat(dst); err == nil {
		return errors.New("staging folder already exists")
	}
	if err := os.MkdirAll(dst, 0o700); err != nil {
		return err
	}
	ok := false
	defer func() {
		if !ok {
			os.RemoveAll(dst)
		}
	}()
	raw, err := readRegular(filepath.Join(p.Dir, ManifestFile), maxManifest)
	if err != nil || sha(raw) != p.ManifestSHA256 {
		return errors.New("FILES_CHANGED: the manifest changed after review; nothing installed")
	}
	if err := os.WriteFile(filepath.Join(dst, ManifestFile), raw, 0o600); err != nil {
		return err
	}
	for _, f := range p.Manifest.Files {
		if err := insideNoLinks(p.Dir, f.Path); err != nil {
			return err
		}
		b, err := readRegular(filepath.Join(p.Dir, filepath.FromSlash(f.Path)), maxFile)
		if err != nil {
			return err
		}
		if sha(b) != f.SHA256 {
			return fmt.Errorf("FILES_CHANGED: %s changed after review; nothing installed", f.Path)
		}
		out := filepath.Join(dst, filepath.FromSlash(f.Path))
		if err := os.MkdirAll(filepath.Dir(out), 0o700); err != nil {
			return err
		}
		if err := os.WriteFile(out, b, 0o600); err != nil {
			return err
		}
	}
	if err := VerifyInstalled(dst, p.ManifestSHA256, p.Manifest.Files); err != nil {
		return err
	}
	ok = true
	return nil
}

// VerifyInstalled re-hashes an installed package (run before every start).
func VerifyInstalled(dir, manifestSHA string, files []File) error {
	raw, err := readRegular(filepath.Join(dir, ManifestFile), maxManifest)
	if err != nil {
		return fmt.Errorf("FILES_MISSING: %v", err)
	}
	if sha(raw) != manifestSHA {
		return errors.New("FILES_CHANGED: installed manifest differs from the approved one")
	}
	for _, f := range files {
		if err := insideNoLinks(dir, f.Path); err != nil {
			return fmt.Errorf("FILES_CHANGED: %v", err)
		}
		b, err := readRegular(filepath.Join(dir, filepath.FromSlash(f.Path)), maxFile)
		if err != nil {
			return fmt.Errorf("FILES_MISSING: %s: %v", f.Path, err)
		}
		if sha(b) != f.SHA256 {
			return fmt.Errorf("FILES_CHANGED: %s differs from the approved SHA-256", f.Path)
		}
	}
	return nil
}
