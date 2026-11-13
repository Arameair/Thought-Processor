package tass

// Stage 3B slice-1 re-audit repair (Codex, PR #122 comment 6839734314): the instruction is validated RAW,
// before any trimming, so a prohibited separator at the start or end cannot disappear before the check.

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
)

var trimBoundaryRunes = []struct {
	name string
	r    string
	want string
}{
	{"CR", "\r", "one line"}, {"LF", "\n", "one line"}, {"VT", "\v", "one line"}, {"FF", "\f", "one line"},
	{"U+0085", "\u0085", "one line"}, {"U+2028", "\u2028", "one line"}, {"U+2029", "\u2029", "one line"},
	{"CRLF", "\r\n", "one line"}, {"tab", "\t", "control character"}, {"NUL", "\x00", "control character"},
	{"zero-width space", "\u200b", "invisible"}, {"BOM", "\ufeff", "invisible"},
}

func TestExperimentVariantIsValidatedBeforeTrimming(t *testing.T) {
	const sentence = "Answer the question directly from the evidence."
	run := func(t *testing.T, c *Core, nid string, existing []byte) {
		n := 0
		for _, b := range trimBoundaryRunes {
			for _, shape := range []struct{ where, v string }{
				{"leading", b.r + sentence}, {"trailing", sentence + b.r},
				{"leading, after spaces", "  " + b.r + sentence}, {"trailing, before spaces", sentence + b.r + "  "},
				{"leading, before spaces", b.r + "  " + sentence}, {"trailing, after spaces", sentence + "  " + b.r},
				{"both ends", b.r + sentence + b.r}, {"alone", b.r}, {"alone among spaces", " " + b.r + " "},
			} {
				n++
				v, err := c.SaveExperimentDraft(expDraft(nid, fmt.Sprintf("exp-trim-%04d", n), shape.v))
				if !errors.Is(err, ErrExperimentInvalid) || !strings.Contains(err.Error(), b.want) || v.ExperimentID != "" {
					t.Errorf("%s %s: err = %v, want refusal saying %q", shape.where, b.name, err, b.want)
				}
			}
		}
		now, err := os.ReadFile(c.file(experimentFile))
		if existing == nil {
			if !os.IsNotExist(err) {
				t.Errorf("a refused instruction created the journal (%d bytes)", len(now))
			}
		} else if !bytes.Equal(now, existing) {
			t.Errorf("a refused instruction changed the existing journal: %d → %d bytes", len(existing), len(now))
		}
	}
	t.Run("no journal is created", func(t *testing.T) {
		c, _, re, nid, _ := processedRun(t)
		reconciled(re)
		run(t, c, nid, nil)
	})
	t.Run("an existing journal is unchanged", func(t *testing.T) {
		c, nid, _, _, valid := planned(t)
		run(t, c, nid, valid)
	})
}

func TestExperimentVariantPermittedSurroundingSpacesAreTrimmed(t *testing.T) {
	c, _, re, nid, _ := processedRun(t)
	reconciled(re)
	const sentence = "Answer the question directly from the evidence."
	a, err := c.SaveExperimentDraft(expDraft(nid, "exp-trim-ok-0001", sentence))
	if err != nil || a.Variant != sentence {
		t.Fatalf("plain sentence: %v %q", err, a.Variant)
	}
	// Ordinary, no-break and ideographic spaces around the sentence are permitted and trimmed; the stored
	// text and its digest are the trimmed sentence, so the same form submitted again is the same plan.
	for i, v := range []string{"   " + sentence + "   ", "\u00a0" + sentence + "\u3000", " \u2003" + sentence} {
		r, err := c.SaveExperimentDraft(expDraft(nid, "exp-trim-ok-0001", v))
		if err != nil || !r.Replayed || r.ExperimentID != a.ExperimentID || r.Variant != sentence {
			t.Errorf("spaces case %d: err=%v replayed=%v variant=%q", i, err, r.Replayed, r.Variant)
		}
	}
	b, err := c.SaveExperimentDraft(expDraft(nid, "exp-trim-ok-0002", "  Say exactly what is missing.  "))
	if err != nil || b.Variant != "Say exactly what is missing." || b.Replayed {
		t.Fatalf("new plan with surrounding spaces: %v %q", err, b.Variant)
	}
	if all, _ := c.readExperiments(); len(all) != 2 {
		t.Fatalf("journal holds %d plans, want 2", len(all))
	}
	// Length is counted after trimming: 300 characters plus surrounding spaces is allowed, 301 is not.
	if _, err := c.SaveExperimentDraft(expDraft(nid, "exp-trim-ok-0300", "  "+strings.Repeat("é", 300)+"  ")); err != nil {
		t.Errorf("300 characters with surrounding spaces: %v", err)
	}
	if _, err := c.SaveExperimentDraft(expDraft(nid, "exp-trim-ok-0301", " "+strings.Repeat("é", 301)+" ")); !errors.Is(err, ErrExperimentInvalid) {
		t.Errorf("301 characters: %v", err)
	}
	if _, err := c.SaveExperimentDraft(expDraft(nid, "exp-trim-ok-empty", "     ")); !errors.Is(err, ErrExperimentInvalid) || !strings.Contains(err.Error(), "write the changed") {
		t.Errorf("only spaces: %v", err)
	}
}
