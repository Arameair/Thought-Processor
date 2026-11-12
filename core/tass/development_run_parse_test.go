package tass

// Stage 3A repair (Codex audit, PR #121 comment 6836598132): Model reply → Saved Note is CONNECTED only when
// the unique saved Note equals the NOTE value the inspected Engine parser (parseThink) keeps from the
// digest-bound reply: its FIRST NOTE line, cleaned, cut at 400 characters, at least 8 characters. A saved
// Note's text appearing anywhere else in the reply proves nothing.

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

// replyNoteRun builds a processed run whose digest-bound reply is reply and whose single current Note for
// this nudge and request is saved. commit: "" = reconciled; otherwise what the Engine reports now.
func replyNoteRun(t *testing.T, reply, saved, commit string) DevelopmentRun {
	t.Helper()
	c, _, re, nid, _ := processedRun(t)
	if commit == "" {
		reconciled(re)
	} else {
		re.mu.Lock()
		re.statusCommit = commit
		re.mu.Unlock()
	}
	re.mu.Lock()
	req := re.reqID[nid]
	re.hideNotes = false
	re.recordHook = func(_ string, rec map[string]any) (int, any) {
		h := sha256.Sum256([]byte(reply))
		rec["model_request"].(map[string]any)["text"] = reply
		rec["response"].(map[string]any)["value"].(map[string]any)["reply_sha256"] = hex.EncodeToString(h[:])
		return 200, rec
	}
	re.mu.Unlock()
	// Replace the stand-in's own Note with the one under test: same nudge and request ID, chosen text.
	re.mu.Lock()
	re.hideNotesExceptExtra = true
	re.extraNotes = []map[string]any{{"at": "2026-11-08T09:00:00Z", "text": saved, "by": "Nemotron (owner-nudge)",
		"worker_request_id": req, "source_nudge_id": nid}}
	re.mu.Unlock()
	r, err := c.DevelopmentRun(nid)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestDevelopmentRunReplyToNoteFollowsEngineParser(t *testing.T) {
	long := strings.Repeat("é", 395) + " tail-beyond-400"
	cut := string([]rune(long)[:400])
	cases := []struct {
		name, reply, saved, commit, want, missing string
	}{
		// Codex's reproduction, verbatim.
		{"saved text only outside the first NOTE line (Codex)", "NOTE: an entirely different saved idea\nSTATE: keep\nLater: keep this thought", "keep this thought", "", "NOT ESTABLISHED", "differs"},
		{"conflicting NOTE line", "NOTE: watch the queue depth\nSTATE: keep", "watch the disk latency", "", "NOT ESTABLISHED", "differs"},
		{"no NOTE line; saved text appears as prose", "I think we should keep this thought for later.\nSTATE: keep", "keep this thought for later", "", "NOT ESTABLISHED", "no NOTE line"},
		{"first of two NOTE lines is the one kept", "NOTE: the first idea here\nNOTE: the second idea here\nSTATE: keep", "the second idea here", "", "NOT ESTABLISHED", "differs"},
		{"first of two NOTE lines matches", "NOTE: the first idea here\nNOTE: the second idea here\nSTATE: keep", "the first idea here", "", "CONNECTED", ""},
		{"markdown, quotes, case and spacing are cleaned as the parser does", "  **note**  =  \"watch   the\tqueue  depth\"  \nSTATE: keep", "watch the queue depth", "", "CONNECTED", ""},
		{"cleaning is exact, not fuzzy", "NOTE: Watch the queue depth.\nSTATE: keep", "watch the queue depth", "", "NOT ESTABLISHED", "differs"},
		{"cut at 400 characters (runes)", "NOTE: " + long + "\nSTATE: keep", cut, "", "CONNECTED", ""},
		{"a saved Note longer than the parser keeps", "NOTE: " + long + "\nSTATE: keep", long, "", "NOT ESTABLISHED", "differs"},
		{"NOTE too short for the parser", "NOTE: tiny\nSTATE: keep", "tiny", "", "NOT ESTABLISHED", "too short"},
		{"empty saved Note", "NOTE: watch the queue depth\nSTATE: keep", "", "", "NOT ESTABLISHED", "differs"},
		{"genuine match", "NOTE: watch the queue depth\nSTATE: resolve", "watch the queue depth", "", "CONNECTED", ""},
		{"genuine match, Engine revision not reported (Docker 'unknown')", "NOTE: watch the queue depth\nSTATE: keep", "watch the queue depth", "unknown", "PARTIAL", "not inspected"},
		{"genuine match, Engine revision mismatch", "NOTE: watch the queue depth\nSTATE: keep", "watch the queue depth", "e" + strings.Repeat("0", 39), "PARTIAL", "not inspected"},
		{"text coincidence, Engine revision not reported", "NOTE: something else entirely\nLater: watch the queue depth", "watch the queue depth", "unknown", "PARTIAL", "not inspected"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := replyNoteRun(t, tc.reply, tc.saved, tc.commit)
			checkMapShape(t, r)
			l := mapLink(t, r, "model_reply", "saved_note")
			if l.Status != tc.want {
				t.Fatalf("model_reply→saved_note = %s (%s), want %s", l.Status, l.Missing, tc.want)
			}
			if tc.missing != "" && !strings.Contains(l.Missing, tc.missing) {
				t.Errorf("missing = %q, want it to say %q", l.Missing, tc.missing)
			}
			if mapLink(t, r, "model_request", "model_reply").Status != "CONNECTED" {
				t.Errorf("the reply itself is digest-bound in every case")
			}
		})
	}
}
