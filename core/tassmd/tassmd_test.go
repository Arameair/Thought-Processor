package tassmd

import (
	"strings"
	"testing"
	"time"
)

const base = "# TASS\n\n## Who I am\n\nI am TASS.\n\n## Memory\n\n- a\n- b\n\n## Now\n\nStarting.\n"

func TestMergeOneSideChanged(t *testing.T) {
	l := strings.Replace(base, "Starting.", "Talking.", 1)
	if m, _, c := Merge(base, l, base); m != l || c != nil {
		t.Fatal("local-only change should win")
	}
	if m, _, c := Merge(base, base, l); m != l || c != nil {
		t.Fatal("remote-only change should win")
	}
}

func TestMergeDifferentSections(t *testing.T) {
	l := strings.Replace(base, "Starting.", "Talking.", 1)
	r := strings.Replace(base, "I am TASS.", "I am TASS, a companion.", 1)
	m, _, c := Merge(base, l, r)
	if c != nil || !strings.Contains(m, "Talking.") || !strings.Contains(m, "a companion") {
		t.Fatalf("both changes expected, conflicts %v:\n%s", c, m)
	}
}

func TestMergeMemoryBothAdded(t *testing.T) {
	l := strings.Replace(base, "- b\n", "- b\n- from TASS\n", 1)
	r := strings.Replace(base, "- a\n", "", 1)
	r = strings.Replace(r, "- b\n", "- b\n- from Dustin\n", 1)
	m, _, c := Merge(base, l, r)
	if c != nil || !strings.Contains(m, "from TASS") || !strings.Contains(m, "from Dustin") || strings.Contains(m, "- a\n") {
		t.Fatalf("memory lines should merge (and Dustin's removal kept), conflicts %v:\n%s", c, m)
	}
}

func TestMergeConflict(t *testing.T) {
	l := strings.Replace(base, "Starting.", "Local now.", 1)
	r := strings.Replace(base, "Starting.", "Remote now.", 1)
	m, mine, c := Merge(base, l, r)
	if len(c) != 1 || c[0] != "Now" || !strings.Contains(m, "Remote now.") || !strings.Contains(mine, "Local now.") {
		t.Fatalf("conflict %v\nmerged:\n%s\nmine:\n%s", c, m, mine)
	}
}

func TestAddMemoryAndNow(t *testing.T) {
	at := time.Date(2026, 11, 4, 7, 30, 0, 0, time.UTC)
	d := AddMemory(base, "likes Go", at)
	if !strings.Contains(d, "- b\n- 2026-11-04: likes Go\n") {
		t.Fatalf("memory:\n%s", d)
	}
	if AddMemory(d, "likes Go", at) != d {
		t.Fatal("duplicate memory added")
	}
	d = SetNow(d, "learning", at)
	if !strings.Contains(d, "## Now\n\nlearning\n") || strings.Contains(d, "Starting.") {
		t.Fatalf("now:\n%s", d)
	}
	if got := Render(Parse(d)); got != d {
		t.Fatal("parse/render must round-trip")
	}
}
