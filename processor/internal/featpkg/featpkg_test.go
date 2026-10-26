package featpkg

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func fixtureDir(t *testing.T) string {
	_, f, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(f), "..", "..", "testfeatures", "deterministic-worker")
}

// copyPkg copies the real test feature so a test can modify it.
func copyPkg(t *testing.T) string {
	src, dst := fixtureDir(t), filepath.Join(t.TempDir(), "pkg")
	os.MkdirAll(dst, 0o700)
	for _, n := range []string{"feature.json", "worker.py"} {
		b, err := os.ReadFile(filepath.Join(src, n))
		if err != nil {
			t.Fatal(err)
		}
		os.WriteFile(filepath.Join(dst, n), b, 0o600)
	}
	return dst
}

func editManifest(t *testing.T, dir string, fn func(m map[string]any)) {
	b, _ := os.ReadFile(filepath.Join(dir, ManifestFile))
	var m map[string]any
	json.Unmarshal(b, &m)
	fn(m)
	b, _ = json.Marshal(m)
	os.WriteFile(filepath.Join(dir, ManifestFile), b, 0o600)
}

func TestRealFixtureLoadsAndIsTrusted(t *testing.T) {
	p, err := Load(fixtureDir(t), true)
	if err != nil {
		t.Fatal(err)
	}
	if p.Blocked != "" || p.Manifest.TrustClass != TrustFixture || len(p.ArtifactSHA256) != 64 {
		t.Fatalf("fixture not installable: %+v", p)
	}
	p2, _ := Load(fixtureDir(t), true)
	if p2.ArtifactSHA256 != p.ArtifactSHA256 {
		t.Fatal("artifact digest not deterministic")
	}
	if u, _ := Load(fixtureDir(t), false); !strings.HasPrefix(u.Blocked, "UNTRUSTED_EXECUTION_BLOCKED") {
		t.Fatal("untrusted folder not blocked")
	}
}

func TestRefusals(t *testing.T) {
	cases := map[string]func(t *testing.T, d string){
		"changed file": func(t *testing.T, d string) {
			os.WriteFile(filepath.Join(d, "worker.py"), []byte("print('evil')\n"), 0o600)
		},
		"symlink": func(t *testing.T, d string) {
			os.Remove(filepath.Join(d, "worker.py"))
			os.Symlink("/etc/hostname", filepath.Join(d, "worker.py"))
		},
		"symlinked parent folder": func(t *testing.T, d string) {
			os.Symlink("/etc", filepath.Join(d, "sub"))
			editManifest(t, d, func(m map[string]any) {
				// the right SHA-256, so the ONLY reason to refuse is the linked folder
				b, _ := os.ReadFile("/etc/hostname")
				m["files"] = []map[string]string{{"path": "sub/hostname", "sha256": sha(b)}}
				m["runtime"].(map[string]any)["command"] = []string{"python3", "{package}/sub/hostname"}
			})
		},
		"traversal": func(t *testing.T, d string) {
			editManifest(t, d, func(m map[string]any) {
				m["files"] = []map[string]string{{"path": "../x", "sha256": strings.Repeat("0", 64)}}
			})
		},
		"absolute command": func(t *testing.T, d string) {
			editManifest(t, d, func(m map[string]any) { m["runtime"].(map[string]any)["command"] = []string{"/bin/sh", "-c", "id"} })
		},
		"unlisted command file": func(t *testing.T, d string) {
			editManifest(t, d, func(m map[string]any) {
				m["runtime"].(map[string]any)["command"] = []string{"python3", "{package}/other.py"}
			})
		},
		"information sources": func(t *testing.T, d string) {
			editManifest(t, d, func(m map[string]any) { m["information_sources"] = []string{"discord"} })
		},
		"unknown field": func(t *testing.T, d string) {
			editManifest(t, d, func(m map[string]any) { m["auto_grant"] = true })
		},
		"wrong api": func(t *testing.T, d string) {
			editManifest(t, d, func(m map[string]any) { m["api"] = "thought.feature-api/9" })
		},
	}
	for name, mut := range cases {
		t.Run(name, func(t *testing.T) {
			d := copyPkg(t)
			mut(t, d)
			if _, err := Load(d, true); err == nil {
				t.Fatalf("%s accepted", name)
			}
		})
	}
	d := copyPkg(t)
	editManifest(t, d, func(m map[string]any) { m["trust_class"] = "THIRD_PARTY_RELEASE" })
	if p, err := Load(d, true); err != nil || !strings.Contains(p.Blocked, "UNTRUSTED_EXECUTION_BLOCKED") {
		t.Fatalf("third-party trust class not blocked: %v %q", err, p.Blocked)
	}
}

func TestStageVerifyAndDetectChanges(t *testing.T) {
	src := copyPkg(t)
	p, err := Load(src, true)
	if err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(t.TempDir(), "staged")
	if err := Stage(p, dst); err != nil {
		t.Fatal(err)
	}
	if err := Stage(p, dst); err == nil {
		t.Fatal("staged over an existing folder")
	}
	if err := VerifyInstalled(dst, p.ManifestSHA256, p.Manifest.Files); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(dst, "worker.py"), []byte("changed"), 0o600)
	if err := VerifyInstalled(dst, p.ManifestSHA256, p.Manifest.Files); err == nil || !strings.Contains(err.Error(), "FILES_CHANGED") {
		t.Fatalf("changed installed file not detected: %v", err)
	}
	// Source changed after review: staging refuses and leaves nothing behind.
	os.WriteFile(filepath.Join(src, "worker.py"), []byte("swapped after review"), 0o600)
	dst2 := filepath.Join(t.TempDir(), "staged2")
	if err := Stage(p, dst2); err == nil {
		t.Fatal("staged a file that changed after review")
	}
	if _, err := os.Stat(dst2); err == nil {
		t.Fatal("failed staging left a folder")
	}
}
