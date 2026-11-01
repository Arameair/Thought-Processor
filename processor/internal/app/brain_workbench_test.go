package app

import (
	"bytes"
	"strings"
	"testing"
)

// Browser interaction and owner Windows/WSL acceptance are separate checks.
// This test verifies that the binary embeds every local Brain Workbench asset.
func TestBrainWorkbenchAssetsEmbedded(t *testing.T) {
	for _, path := range []string{
		"web/workstation.html", "web/workstation.js",
		"web/brain-workbench.js", "web/brain-workbench.css",
	} {
		t.Run(path, func(t *testing.T) {
			b, err := webFS.ReadFile(path)
			if err != nil || len(b) == 0 {
				t.Fatalf("missing embedded asset %s: %v", path, err)
			}
		})
	}
	html, _ := webFS.ReadFile("web/workstation.html")
	for _, marker := range []string{
		"id=\"bw-map\"", "id=\"bw-wires\"", "id=\"bw-log\"",
		"id=\"bw-copy-log\"", "id=\"bw-activity\"", "id=\"ws-tass-chat\"",
		"id=\"ws-runs\"", "id=\"ws-th-list\"",
	} {
		if !bytes.Contains(html, []byte(marker)) {
			t.Errorf("UI lost element %s", marker)
		}
	}
	js, _ := webFS.ReadFile("web/brain-workbench.js")
	for _, marker := range []string{
		"/api/ws/diagnostics", "/api/installations/", "PROPOSED_INACTIVE",
		"bw:workstation-state", "navigator.clipboard.writeText",
	} {
		if !bytes.Contains(js, []byte(marker)) {
			t.Errorf("Brain controller missing %s", marker)
		}
	}
}

// Diagnostics are scrubbed before storage. Never copy raw feature.log files.
func TestBrainWorkbenchProcessorLogRedaction(t *testing.T) {
	log := OpenLog(t.TempDir(), "test-run")
	defer log.Close()
	log.Info("workstation", "sample", "password=my-secret #code=abcd1234")
	last := log.Recent(1)
	if len(last) != 1 || strings.Contains(last[0].Detail, "my-secret") ||
		strings.Contains(last[0].Detail, "abcd1234") {
		t.Fatalf("unscrubbed Processor diagnostic detail: %q", last)
	}
}
