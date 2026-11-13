package tass

// Stage 3B slice-1 repair (Codex audit, PR #122 comment 6839415411). Four fail-closed boundaries:
// Unicode input validation, corrupt journal records, the journal size ceiling, journal permissions.
// Every refusal must expose no saved plan and leave the journal byte-for-byte unchanged.

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const expSecret = "PRIVATE-SAVED-INSTRUCTION please answer the question directly"

// planned: a processed run with one valid saved plan; returns the journal path and its bytes.
func planned(t *testing.T) (*Core, string, string, ExperimentView, []byte) {
	t.Helper()
	c, _, re, nid, _ := processedRun(t)
	reconciled(re)
	v, err := c.SaveExperimentDraft(expDraft(nid, "exp-repair-0001", expSecret))
	if err != nil {
		t.Fatal(err)
	}
	p := c.file(experimentFile)
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return c, nid, p, v, b
}

// refusedEverywhere: list, get and save are all refused with the journal error, return nothing, and the
// journal (and its directory listing) is unchanged.
func refusedEverywhere(t *testing.T, c *Core, nid, path, id string) {
	t.Helper()
	before, _ := os.ReadFile(path)
	fi0, _ := os.Lstat(path)
	names := func() string {
		es, _ := os.ReadDir(filepath.Dir(path))
		var s []string
		for _, e := range es {
			if strings.HasPrefix(e.Name(), "experiments") {
				s = append(s, e.Name())
			}
		}
		return strings.Join(s, ",")
	}
	n0 := names()
	l, err := c.Experiments(nid)
	if !errors.Is(err, ErrExperimentJournal) || len(l.Experiments) != 0 || l.Control != "" {
		t.Errorf("list: err=%v, %d plans returned", err, len(l.Experiments))
	}
	g, err := c.ExperimentByID(id)
	if !errors.Is(err, ErrExperimentJournal) || g.ExperimentID != "" || g.Variant != "" {
		t.Errorf("get: err=%v plan=%q", err, g.ExperimentID)
	}
	s, err := c.SaveExperimentDraft(expDraft(nid, "exp-repair-new1", "A new and different instruction sentence."))
	if !errors.Is(err, ErrExperimentJournal) || s.ExperimentID != "" {
		t.Errorf("save new: err=%v plan=%q", err, s.ExperimentID)
	}
	// A replay of the already-saved plan must not be answered from an unsafe journal either.
	r, err := c.SaveExperimentDraft(expDraft(nid, "exp-repair-0001", expSecret))
	if !errors.Is(err, ErrExperimentJournal) || r.ExperimentID != "" {
		t.Errorf("replay: err=%v plan=%q", err, r.ExperimentID)
	}
	if err != nil && strings.Contains(err.Error(), "PRIVATE-SAVED") {
		t.Errorf("the error carries plan text: %v", err)
	}
	after, _ := os.ReadFile(path)
	fi1, _ := os.Lstat(path)
	if !bytes.Equal(before, after) || fi0.Mode() != fi1.Mode() || names() != n0 {
		t.Errorf("the journal changed: %d → %d bytes, mode %v → %v, files %q → %q", len(before), len(after), fi0.Mode(), fi1.Mode(), n0, names())
	}
}

func TestExperimentVariantRefusesUnicodeSeparatorsAndConfusableMarkers(t *testing.T) {
	c, _, re, nid, _ := processedRun(t)
	reconciled(re)
	cases := []struct{ name, variant, want string }{
		// Codex's four reproductions.
		{"U+2028 line separator", "Answer the question.\u2028Then stop.", "one line"},
		{"U+2029 paragraph separator", "Answer the question.\u2029Then stop.", "one line"},
		{"fullwidth colon NOTE：", "Answer directly. NOTE： say yes", "NOTE: or STATE:"},
		{"fullwidth equals STATE＝", "Answer directly. STATE＝ resolve", "NOTE: or STATE:"},
		// The same bypass by other routes.
		{"U+0085 next line", "Answer the question.\u0085Then stop.", "one line"},
		{"vertical tab", "Answer the question.\vThen stop.", "one line"},
		{"fullwidth letters ＮＯＴＥ:", "Answer. ＮＯＴＥ: yes", "NOTE: or STATE:"},
		{"small colon NOTE﹕", "Answer. NOTE﹕ yes", "NOTE: or STATE:"},
		{"ratio sign NOTE∶", "Answer. note∶ yes", "NOTE: or STATE:"},
		{"modifier colon NOTE꞉", "Answer. Note꞉ yes", "NOTE: or STATE:"},
		{"Armenian full stop STATE։", "Answer. STATE։ keep", "NOTE: or STATE:"},
		{"long s ſtate:", "Answer. ſtate: keep", "NOTE: or STATE:"},
		{"zero-width space inside NOTE", "Answer. NO\u200bTE: yes", "invisible"},
		{"zero-width joiner before the colon", "Answer. NOTE\u200d: yes", "invisible"},
		{"right-to-left override", "Answer \u202eETON: yes", "invisible"},
		{"byte order mark", "Answer.\ufeff NOTE : yes", "invisible"},
		{"no-break space before the colon", "Answer. NOTE\u00a0: yes", "NOTE: or STATE:"},
		{"ideographic space before equals", "Answer. STATE\u3000= keep", "NOTE: or STATE:"},
		{"combining mark before the colon", "Answer. NOTE\u0338: yes", "NOTE: or STATE:"},
		{"emphasis then fullwidth colon", "Answer. **NOTE**： yes", "NOTE: or STATE:"},
		{"double-struck equals", "Answer. STATE ⩵ keep", "NOTE: or STATE:"},
		{"same as A with a no-break space", strings.Replace(experimentControl, " ", "\u00a0", 1), "B must differ"},
		{"invalid UTF-8", "Answer \xff directly", "not valid text"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			v, err := c.SaveExperimentDraft(expDraft(nid, "exp-uni-"+strings.Map(func(r rune) rune {
				if r < 128 && (r >= 'a' && r <= 'z' || r >= '0' && r <= '9') {
					return r
				}
				return '-'
			}, strings.ToLower(tc.name)), tc.variant))
			if !errors.Is(err, ErrExperimentInvalid) || !strings.Contains(err.Error(), tc.want) || v.ExperimentID != "" {
				t.Fatalf("err = %v, want refusal saying %q", err, tc.want)
			}
		})
	}
	if _, err := os.Stat(c.file(experimentFile)); !os.IsNotExist(err) {
		t.Fatalf("a refused variant created or changed the journal: %v", err)
	}
	// Ordinary sentences are still accepted, including other scripts and the words note/state in prose.
	for i, ok := range []string{
		"Take note of the evidence, state what is missing, then answer.",
		"Answer directly; note: is not allowed but a note, like this, is.",
		"Réponds directement à la question — sans répéter les notes.",
		"質問に直接答えてください。",
		"Estate planning footnotes: ignore them.",
	} {
		_, err := c.SaveExperimentDraft(expDraft(nid, "exp-uni-ok-000"+string(rune('0'+i)), ok))
		if i == 1 {
			if !errors.Is(err, ErrExperimentInvalid) {
				t.Errorf("%q must still be refused (ASCII note:)", ok)
			}
			continue
		}
		if err != nil {
			t.Errorf("%q refused: %v", ok, err)
		}
	}
}

func TestExperimentJournalCorruptRecordFailsClosed(t *testing.T) {
	good := func(b []byte) string { return string(bytes.TrimRight(b, "\n")) }
	for _, tc := range []struct {
		name string
		make func(valid []byte) []byte
	}{
		{"complete malformed record after a valid one (Codex)", func(v []byte) []byte { return append(append([]byte{}, v...), []byte("{\"schema\": broken}\n")...) }},
		{"complete malformed record before a valid one", func(v []byte) []byte { return append([]byte("not json at all\n"), v...) }},
		{"interior malformed record", func(v []byte) []byte {
			return []byte(good(v) + "\n{\"experiment_id\":\n" + good(v) + "\n")
		}},
		{"valid JSON, wrong schema", func(v []byte) []byte {
			return append(append([]byte{}, v...), []byte("{\"schema\":\"other/1\",\"experiment_id\":\"exp_00000000000000000000\"}\n")...)
		}},
		{"valid JSON, bad experiment ID", func(v []byte) []byte {
			return append(append([]byte{}, v...), []byte("{\"schema\":\""+experimentSchema+"\",\"experiment_id\":\"exp_../../x\"}\n")...)
		}},
		{"blank interior line", func(v []byte) []byte { return []byte(good(v) + "\n\n" + good(v) + "\n") }},
		{"NUL bytes", func(v []byte) []byte { return append(append([]byte{}, v...), 0, 0, 0, '\n') }},
		{"the same experiment ID twice", func(v []byte) []byte { return append(append([]byte{}, v...), v...) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, nid, path, v, valid := planned(t)
			if err := os.WriteFile(path, tc.make(valid), 0o600); err != nil {
				t.Fatal(err)
			}
			refusedEverywhere(t, c, nid, path, v.ExperimentID)
		})
	}
}

// The one recovery rule: bytes after the last newline are an append that was cut off before it was
// acknowledged. Reads never return them. The next save moves exactly those bytes into a private
// quarantine file, cuts the journal back to its last complete record, and then appends.
func TestExperimentJournalTrailingPartialWriteIsQuarantinedNotRead(t *testing.T) {
	c, nid, path, v, valid := planned(t)
	frag := []byte(`{"schema":"` + experimentSchema + `","experiment_id":"exp_11111111111111111111","variant":"HALF-WRITTEN-PRIVATE`)
	if err := os.WriteFile(path, append(append([]byte{}, valid...), frag...), 0o600); err != nil {
		t.Fatal(err)
	}
	l, err := c.Experiments(nid)
	if err != nil || len(l.Experiments) != 1 || l.Experiments[0].ExperimentID != v.ExperimentID {
		t.Fatalf("read with a trailing fragment: %v, %d plans", err, len(l.Experiments))
	}
	if _, err := c.ExperimentByID("exp_11111111111111111111"); !errors.Is(err, ErrExperimentNotFound) {
		t.Fatalf("the half-written record was readable: %v", err)
	}
	if now, _ := os.ReadFile(path); !bytes.HasSuffix(now, frag) {
		t.Fatal("a read changed the journal")
	}
	// A fragment that happens to be a whole JSON object without its newline is still unacknowledged.
	whole := bytes.TrimRight(bytes.Replace(valid, []byte(v.ExperimentID), []byte("exp_22222222222222222222"), 1), "\n")
	os.WriteFile(path, append(append([]byte{}, valid...), whole...), 0o600)
	if _, err := c.ExperimentByID("exp_22222222222222222222"); !errors.Is(err, ErrExperimentNotFound) {
		t.Fatalf("an unterminated record was readable: %v", err)
	}
	os.WriteFile(path, append(append([]byte{}, valid...), frag...), 0o600)

	n, err := c.SaveExperimentDraft(expDraft(nid, "exp-repair-after", "Answer the question and say what is missing."))
	if err != nil {
		t.Fatalf("save after a trailing fragment: %v", err)
	}
	now, _ := os.ReadFile(path)
	if !bytes.HasPrefix(now, valid) || bytes.Contains(now, []byte("HALF-WRITTEN")) || !bytes.HasSuffix(now, []byte("\n")) || bytes.Count(now, []byte("\n")) != 2 {
		t.Fatalf("journal after recovery is not [valid record][new record]: %q", now)
	}
	qs, _ := filepath.Glob(path + ".partial-*")
	if len(qs) != 1 {
		t.Fatalf("quarantine files: %v", qs)
	}
	q, _ := os.ReadFile(qs[0])
	fi, _ := os.Stat(qs[0])
	if !bytes.Equal(q, frag) || fi.Mode().Perm() != 0o600 {
		t.Fatalf("quarantine holds %q mode %v", q, fi.Mode().Perm())
	}
	l, err = c.Experiments(nid)
	if err != nil || len(l.Experiments) != 2 || l.Experiments[0].ExperimentID != n.ExperimentID {
		t.Fatalf("after recovery: %v, %d plans", err, len(l.Experiments))
	}
	// Idempotency still holds across the recovery.
	if r, err := c.SaveExperimentDraft(expDraft(nid, "exp-repair-0001", expSecret)); err != nil || !r.Replayed || r.ExperimentID != v.ExperimentID {
		t.Fatalf("replay after recovery: %v", err)
	}
}

func TestExperimentJournalSizeCeilingIsEnforcedBeforeReading(t *testing.T) {
	c, nid, path, v, valid := planned(t)
	// Codex's case: a journal above the declared ceiling. It is all valid records, so only the size refuses it.
	big := bytes.Repeat(bytes.Replace(valid, []byte(v.ExperimentID), []byte("exp_33333333333333333333"), 1), experimentJournalMax/len(valid)+2)
	big = append(append([]byte{}, valid...), big...)
	if len(big) <= experimentJournalMax {
		t.Fatal("test journal is not over the ceiling")
	}
	if err := os.WriteFile(path, big, 0o600); err != nil {
		t.Fatal(err)
	}
	opened := 0
	old := experimentOpenRead
	experimentOpenRead = func(name string) (*os.File, error) { opened++; return old(name) }
	defer func() { experimentOpenRead = old }()
	refusedEverywhere(t, c, nid, path, v.ExperimentID)
	if opened != 0 {
		t.Errorf("an oversized journal was opened for reading %d times; the size must refuse it first", opened)
	}
}

func TestExperimentJournalNeverGrowsPastItsCeiling(t *testing.T) {
	c, nid, path, _, valid := planned(t)
	// Fill to just under the ceiling with valid, unique records.
	at := bytes.Index(valid, []byte("exp_"))
	var buf bytes.Buffer
	buf.Write(valid)
	for i := 1; buf.Len()+2*len(valid) < experimentJournalMax; i++ {
		buf.Write(bytes.Replace(valid, valid[at:at+24], []byte(fmt.Sprintf("exp_%020x", i)), 1))
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Experiments(nid); err != nil {
		t.Fatalf("a journal under the ceiling must read: %v", err)
	}
	before, _ := os.ReadFile(path)
	// Appends are refused once the next record would cross the ceiling, so the journal can never lock itself.
	var last error
	for k := 0; k < 50; k++ {
		_, last = c.SaveExperimentDraft(expDraft(nid, fmt.Sprintf("exp-fill-%04d", k), fmt.Sprintf("Fill sentence number %d.", k)))
		if last != nil {
			break
		}
	}
	if !errors.Is(last, ErrExperimentInvalid) || !strings.Contains(last.Error(), "full") {
		t.Fatalf("filling the journal ended with %v", last)
	}
	fi, _ := os.Stat(path)
	if fi.Size() > experimentJournalMax || fi.Size() < int64(len(before)) {
		t.Fatalf("journal is %d bytes (ceiling %d)", fi.Size(), experimentJournalMax)
	}
	if _, err := c.Experiments(nid); err != nil {
		t.Fatalf("a full journal must still read: %v", err)
	}
}

func TestExperimentJournalRefusesPermissiveOrNonRegularFile(t *testing.T) {
	for _, mode := range []os.FileMode{0o644, 0o640, 0o604, 0o660, 0o666, 0o700, 0o601} {
		t.Run(mode.String(), func(t *testing.T) {
			c, nid, path, v, _ := planned(t)
			if err := os.Chmod(path, mode); err != nil {
				t.Fatal(err)
			}
			refusedEverywhere(t, c, nid, path, v.ExperimentID)
		})
	}
	t.Run("owner-only modes are accepted", func(t *testing.T) {
		c, nid, path, v, _ := planned(t)
		os.Chmod(path, 0o400)
		if g, err := c.ExperimentByID(v.ExperimentID); err != nil || g.ExperimentID != v.ExperimentID {
			t.Fatalf("0400 read: %v", err)
		}
		os.Chmod(path, 0o600)
		if _, err := c.SaveExperimentDraft(expDraft(nid, "exp-repair-0600", "Answer the question, briefly.")); err != nil {
			t.Fatalf("0600 save: %v", err)
		}
		if fi, _ := os.Stat(path); fi.Mode().Perm() != 0o600 {
			t.Fatalf("mode after save: %v", fi.Mode().Perm())
		}
	})
	t.Run("symlink", func(t *testing.T) {
		c, nid, path, v, valid := planned(t)
		real := path + ".elsewhere"
		os.WriteFile(real, valid, 0o600)
		os.Remove(path)
		if err := os.Symlink(real, path); err != nil {
			t.Skip(err)
		}
		refusedEverywhere(t, c, nid, path, v.ExperimentID)
		if b, _ := os.ReadFile(real); !bytes.Equal(b, valid) {
			t.Error("the symlink target changed")
		}
	})
	t.Run("directory", func(t *testing.T) {
		c, nid, path, v, _ := planned(t)
		os.Remove(path)
		os.Mkdir(path, 0o700)
		if _, err := c.Experiments(nid); !errors.Is(err, ErrExperimentJournal) {
			t.Errorf("list: %v", err)
		}
		if _, err := c.SaveExperimentDraft(expDraft(nid, "exp-repair-dir0", "Answer the question, briefly.")); !errors.Is(err, ErrExperimentJournal) {
			t.Errorf("save: %v", err)
		}
		_ = v
	})
}
