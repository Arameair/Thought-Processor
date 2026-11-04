package tass

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestTeachingRequestParsing(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"TASS, ask ChatGPT to teach you how to use Developer", "teach you how to use Developer"},
		{"ask chatgpt to explain the current thought cycle", "explain the current thought cycle"},
		{"@ask-chatgpt: explain one verified repository link", "explain one verified repository link"},
		{"hello can you see the repo now?", ""},
		{"TASS, explain @identity-test", ""},
		{"ask chatgpt about this", ""},
	}
	for _, tc := range cases {
		if got := requestedTeachingTopic(tc.input); got != tc.want {
			t.Errorf("request %q: got %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestTeachingDraftRemainsUnsent(t *testing.T) {
	dir := t.TempDir()
	c := &Core{cfg: Config{DataDir: dir}, now: func() time.Time {
		return time.Date(2026, 11, 4, 19, 0, 0, 0, time.UTC)
	}}
	id, err := c.createTeachingDraft("learn how to inspect one verified Developer connection")
	if err != nil { t.Fatal(err) }
	if !strings.HasPrefix(id, "teach_") { t.Fatalf("bad draft id %q", id) }
	f, err := os.Open(filepath.Join(dir, "teaching-requests.jsonl"))
	if err != nil { t.Fatal(err) }
	defer f.Close()
	scan := bufio.NewScanner(f)
	if !scan.Scan() { t.Fatal("no durable teaching draft") }
	var rec teachingDraft
	if err := json.Unmarshal(scan.Bytes(), &rec); err != nil { t.Fatal(err) }
	if rec.ID != id || rec.Status != "DRAFT_NOT_SENT" || rec.To != "ChatGPT" {
		t.Fatalf("draft unexpectedly claimed delivery: %+v", rec)
	}
	if scan.Scan() { t.Fatal("unexpected extra record") }
	if err := scan.Err(); err != nil { t.Fatal(err) }
}

func TestTeachingDraftRejectsOversizedMessage(t *testing.T) {
	c := &Core{cfg: Config{DataDir: t.TempDir()}, now: time.Now}
	if _, err := c.createTeachingDraft(strings.Repeat("x", 501)); err == nil {
		t.Fatal("oversized request accepted")
	}
	if _, err := c.createTeachingDraft("short"); err == nil {
		t.Fatal("undersized request accepted")
	}
}
