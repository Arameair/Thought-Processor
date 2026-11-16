package tass

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The Core copy of the parser's NOTE extraction must be the Engine's. With ENGINE_SRC set to a Thought
// Engine checkout at an inspected revision, compare the source of reNote, cleanLine and the length floor.
func TestEngineNoteExtractionMatchesInspectedEngineSource(t *testing.T) {
	dir := os.Getenv("ENGINE_SRC")
	if dir == "" {
		t.Skip("ENGINE_SRC not set: compare against a Thought Engine checkout in the release gate")
	}
	src, err := os.ReadFile(filepath.Join(dir, "prompts.go"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(src)
	if !regexp.MustCompile(`reNote\s*=\s*regexp\.MustCompile\(` + "`" + regexp.QuoteMeta(engineReNote.String()) + "`" + `\)`).MatchString(s) {
		t.Errorf("Engine reNote differs from Core's %q", engineReNote.String())
	}
	for _, want := range []string{
		"m := reNote.FindStringSubmatch(reply)",
		"note := cleanLine(m[1], 400)",
		"if len([]rune(note)) < 8 {",
		"s = strings.Trim(strings.TrimSpace(s), `\"*`)",
		"s = strings.Join(strings.Fields(s), \" \")",
		"if r := []rune(s); len(r) > n {",
		"s = string(r[:n])",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("Engine parser source no longer contains %q", want)
		}
	}
}
