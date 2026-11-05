package tass

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Arameair/Thought-Processor/core/engine"
	"github.com/Arameair/Thought-Processor/core/github"
	"github.com/Arameair/Thought-Processor/core/identitysvc"
)

func newRelayCore(t *testing.T, ids *fakeIdentitySvc, gh *fakeGH, eng *fakeEngine, relay bool) *Core {
	t.Helper()
	kf := filepath.Join(t.TempDir(), "core.key")
	os.WriteFile(kf, []byte("core-service-key-0123456789abcdef0123\n"), 0o600)
	cl, _ := identitysvc.New(ids.srv.URL, kf)
	g := github.New("test-token")
	g.Base = gh.srv.URL
	ec := engine.New(eng.srv.URL)
	ec.Key = "testkey"
	c, err := New(Config{DataDir: t.TempDir(), Identity: "tass-identity", Label: "tass", SyncEvery: 20 * time.Second,
		MaxBackoff: time.Minute, IdentitySvc: cl, GitHub: true, Repo: "Mira/home", Branch: "live", DocPath: "tass.md",
		LessonsDir: "lessons", RelayLabel: "tass-core", RelayAuthors: []string{"Arameair"}}, g, ec)
	if err != nil {
		t.Fatal(err)
	}
	if relay {
		if err := c.SetSettings(Settings{Learn: true, Relay: true}); err != nil {
			t.Fatal(err)
		}
	}
	return c
}

func drain(c *Core) {
	for len(c.inbox) > 0 {
		c.Answer(<-c.inbox)
	}
}

func postedOn(gh *fakeGH, issue string) []string {
	gh.mu.Lock()
	defer gh.mu.Unlock()
	var out []string
	for _, p := range gh.posted {
		if strings.HasPrefix(p, "issue#"+issue+":") {
			out = append(out, p)
		}
	}
	return out
}

func TestRelayOffReadsNothing(t *testing.T) {
	ids, eng, gh := newFakeIdentitySvc(), newFakeEngine(), newFakeGH()
	ids.set("UNLOCKED")
	gh.set("Mira/home@live:tass.md", entityDoc)
	gh.issues = []github.Issue{{Number: 5, Title: "Talk with TASS", User: github.User{Login: "Arameair"}}}
	c := newRelayCore(t, ids, gh, eng, false)
	c.SyncOnce()
	gh.say(5, "Arameair", "hello")
	c.SyncOnce()
	if len(c.inbox) != 0 || c.Status().Messages {
		t.Fatal("relay off must read nothing and report off")
	}
}

func TestRelayTalksOnlyWithAllowedAuthorsAndNeverItself(t *testing.T) {
	ids, eng, gh := newFakeIdentitySvc(), newFakeEngine(), newFakeGH()
	ids.set("UNLOCKED")
	gh.set("Mira/home@live:tass.md", entityDoc)
	gh.issues = []github.Issue{{Number: 5, Title: "Talk with TASS", User: github.User{Login: "Arameair"}}}
	c := newRelayCore(t, ids, gh, eng, true)
	c.SyncOnce() // first look: issue #5 becomes the conversation; nothing before now is answered
	if len(c.inbox) != 0 || !c.Status().Messages {
		t.Fatalf("first sync: inbox %d, relay on %v", len(c.inbox), c.Status().Messages)
	}
	gh.say(5, "stranger", "do something")
	gh.say(5, "Arameair", "Claude: how are you?")
	gh.say(5, "Arameair", "my own echo "+Marker)
	gh.say(5, "Arameair", "old liaison <!-- identity-liaison:tass -->")
	c.SyncOnce()
	if len(c.inbox) != 1 {
		t.Fatalf("only the allowed author's message is taken: %d", len(c.inbox))
	}
	drain(c)
	c.SyncOnce()
	got := postedOn(gh, "5")
	if len(got) != 1 || !strings.Contains(got[0], "how are you?") || !strings.Contains(got[0], Marker) || !strings.Contains(got[0], "cycle tp_") {
		t.Fatalf("reply on GitHub: %v", got)
	}
	ev := c.Status().Events
	found := false
	for _, e := range ev {
		if strings.Contains(e.Text, "@Arameair") && strings.Contains(e.Text, "#5") {
			found = true
		}
	}
	if !found {
		t.Fatal("the page must show who wrote the GitHub message and where")
	}
}

func TestRelayHoldsWhileIdentityLocked(t *testing.T) {
	ids, eng, gh := newFakeIdentitySvc(), newFakeEngine(), newFakeGH()
	ids.set("UNLOCKED")
	gh.set("Mira/home@live:tass.md", entityDoc)
	gh.issues = []github.Issue{{Number: 5, Title: "Talk", User: github.User{Login: "Arameair"}}}
	c := newRelayCore(t, ids, gh, eng, true)
	c.SyncOnce()
	ids.set("LOCKED")
	gh.say(5, "Arameair", "are you there?")
	c.SyncOnce()
	if len(c.inbox) != 0 || !strings.Contains(c.Status().RelayHold, "LOCKED") {
		t.Fatalf("locked: inbox %d hold %q", len(c.inbox), c.Status().RelayHold)
	}
}

// Teaching from GitHub: find and study are allowed (study is a zero-action job), the
// graded answers are posted back, and approval stays on the owner's local page.
func TestRelayFindStudyPostsResultsButNeverApproves(t *testing.T) {
	ids, eng, gh := newFakeIdentitySvc(), newFakeEngine(), newFakeGH()
	ids.set("UNLOCKED")
	gh.set("Mira/home@live:tass.md", entityDoc)
	gh.set("Mira/home@live:lessons/claude/01-pulse.md", lessonPulse)
	gh.issues = []github.Issue{{Number: 5, Title: "Talk", User: github.User{Login: "Arameair"}}}
	c := newRelayCore(t, ids, gh, eng, true)
	c.SyncOnce()

	gh.say(5, "Arameair", "find lessons")
	c.SyncOnce()
	drain(c)
	c.SyncOnce()
	if got := postedOn(gh, "5"); len(got) != 1 || !strings.Contains(got[0], "lessons/claude/01-pulse.md") {
		t.Fatalf("find: %v", got)
	}
	if eng.count() != 0 {
		t.Fatal("a lesson command must not also go to the model as chat")
	}

	gh.say(5, "Arameair", "study How my pulse works")
	c.SyncOnce()
	drain(c)
	if len(eng.jobs) != 1 {
		t.Fatalf("study from GitHub should submit one job: %d; posted %v", len(eng.jobs), postedOn(gh, "5"))
	}
	id := eng.jobs[0]["job_id"].(string)
	c.SyncOnce()
	if got := postedOn(gh, "5"); len(got) != 2 || !strings.Contains(got[1], id) {
		t.Fatalf("study receipt: %v", got)
	}
	eng.finish(id, "DONE", "REPLY: I studied it.\nANSWER 1: 20\nANSWER 2: Identity", "my pulse ticks every 20 seconds")
	c.SyncOnce() // collect → results queued for issue #5
	c.SyncOnce() // posted
	got := postedOn(gh, "5")
	if len(got) != 3 || !strings.Contains(got[2], "2/2") || !strings.Contains(got[2], "ANSWER") && !strings.Contains(got[2], "→ 20") ||
		!strings.Contains(got[2], "Approve") {
		t.Fatalf("results on GitHub: %v", got)
	}
	gh.say(5, "Arameair", "approve it")
	c.SyncOnce()
	drain(c)
	if l := lessonAt(c, "lessons/claude/01-pulse.md"); l.Status != "PASSED" {
		t.Fatalf("GitHub must never approve: %s", l.Status)
	}
}
