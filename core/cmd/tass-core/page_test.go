package main

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// The owner's "Find lessons (manual)" button sits in a .ctl row. A page-wide
// ".ctl button" binding for the pulse controls once replaced its click handler,
// so Find sent POST /api/pulse (refused in chat-only mode) and never asked for
// discovery. Pulse bindings must only reach buttons that carry a pulse action.
func TestPulseBindingDoesNotCaptureLessonButtons(t *testing.T) {
	b, err := os.ReadFile("page.html")
	if err != nil {
		t.Fatal(err)
	}
	page := string(b)
	if strings.Contains(page, `querySelectorAll(".ctl button")`) {
		t.Fatal(`pulse handler is bound to every ".ctl button", which includes Find lessons`)
	}
	if !strings.Contains(page, `querySelectorAll(".ctl button[data-a]")`) {
		t.Fatal("pulse controls must be bound only to buttons with a data-a pulse action")
	}
	if !regexp.MustCompile(`id="findLessons"[^>]*>`).MatchString(page) || strings.Contains(page, `id="findLessons" data-a`) {
		t.Fatal("Find lessons must exist and must not carry a pulse action")
	}
	if !strings.Contains(page, `post("/api/lessons/discover"`) {
		t.Fatal("Find lessons must request discovery")
	}
}

// The periodic refresh must not overwrite the result of the owner's last click
// (it used to replace it every second while a "no grant" note was present).
func TestRefreshDoesNotOverwriteOwnerResultMessage(t *testing.T) {
	b, err := os.ReadFile("page.html")
	if err != nil {
		t.Fatal(err)
	}
	i := strings.Index(string(b), "async function refresh()")
	j := strings.Index(string(b), "const post = ")
	if i < 0 || j < i {
		t.Fatal("page layout changed; update this test")
	}
	if strings.Contains(string(b)[i:j], `$("lessonMessage")`) {
		t.Fatal("refresh() must not write lessonMessage (that line belongs to the owner's last action)")
	}
}

// Show the distinction between this Go Core and the older GitHub Identity Liaison.
// The two new Talk controls must use the real supervised endpoints, never the pulse
// controls or an invented local model result.
func TestTalkLessonControlsAndRelayStatusAreHonest(t *testing.T) {
    b, err := os.ReadFile("page.html")
    if err != nil { t.Fatal(err) }
    s := string(b)
    for _, want := range []string{
        `id="talkFind"`, `id="talkTest"`, `id="relayState"`,
        `st.github_messages`, `GitHub issue relay OFF in this Go Core`,
        `$("talkFind").onclick`, `$("talkTest").onclick`,
        `post("/api/lessons/discover", {})`, `post("/api/lessons/study", {path: chosen.path})`,
    } {
        if !strings.Contains(s, want) { t.Errorf("missing real Talk/relay behavior %q", want) }
    }
    if strings.Contains(s, `querySelectorAll(".ctl button")`) {
        t.Fatal("pulse must not overwrite the Talk lesson buttons")
    }
}

// Regression for the owner's live run: expanding "His full answer" must
// survive the periodic status refresh that rebuilds the Learning card.
func TestRefreshPreservesExpandedLessonAnswer(t *testing.T) {
    b, err := os.ReadFile("page.html")
    if err != nil { t.Fatal(err) }
    page := string(b)
    for _, part := range []string{
        `querySelectorAll("details[data-lesson-path][open]")`,
        `d => d.dataset.lessonPath`,
        `data-lesson-path="${esc(l.path)}"`,
        `openAnswers.has(l.path) ? "open" : ""`,
    } {
        if !strings.Contains(page, part) { t.Errorf("Learning refresh no longer preserves opened answer state: %s", part) }
    }
}
