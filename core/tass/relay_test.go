package tass

import (
    "encoding/json"
    "os"
    "path/filepath"
    "strings"
    "strconv"
    "testing"
    "time"

    "github.com/Arameair/Thought-Processor/core/engine"
    "github.com/Arameair/Thought-Processor/core/github"
    "github.com/Arameair/Thought-Processor/core/identitysvc"
)

func pilotFixture(t *testing.T, dir string, ids *fakeIdentitySvc, eng *fakeEngine, gh *fakeGH) *Core {
    t.Helper()
    key := filepath.Join(t.TempDir(), "core.key")
    if err := os.WriteFile(key, []byte("core-service-key-0123456789abcdef0123\n"), 0600); err != nil { t.Fatal(err) }
    auth, err := identitysvc.New(ids.srv.URL, key)
    if err != nil { t.Fatal(err) }
    client := github.New("fixture-only")
    client.Base = gh.srv.URL
    e := engine.New(eng.srv.URL)
    e.Key = "testkey"
    c, err := New(Config{DataDir:dir, Identity:"tass-identity", IdentitySvc:auth,
        GitHub:true, Repo:"Arameair/TASS", Branch:"live", DocPath:"tass.md",
        LessonsDir:"lessons", RelayLabel:"tass-core", RelayAuthors:[]string{"Arameair"}},
        client, e)
    if err != nil { t.Fatal(err) }
    return c
}

func mailboxFixture(t *testing.T) (*fakeIdentitySvc, *fakeEngine, *fakeGH) {
    t.Helper()
    ids, eng, gh := newFakeIdentitySvc(), newFakeEngine(), newFakeGH()
    ids.set("UNLOCKED")
    gh.set("Arameair/TASS@live:tass.md", entityDoc)
    var issue github.Issue
    if err := json.Unmarshal([]byte(`{"number":24,"title":"Talk with TASS","state":"open","labels":[{"name":"tass-core"}]}`), &issue); err != nil { t.Fatal(err) }
    gh.issues = []github.Issue{issue}
    t.Cleanup(func(){ ids.srv.Close(); eng.srv.Close(); gh.srv.Close() })
    return ids, eng, gh
}

func TestPilotMailboxRealIDsAuthorizedChatOnlyAndRestartDedup(t *testing.T) {
    ids, eng, gh := mailboxFixture(t)
    dir := t.TempDir()
    c := pilotFixture(t, dir, ids, eng, gh)
    c.SyncOnce() // OFF: nothing primed or read
    gh.say(24, "Arameair", "ChatGPT: old before relay enable")
    if err := c.SetSettings(Settings{Learn:true,Relay:true}); err != nil { t.Fatal(err) }
    c.SyncOnce() // prime: old comment is excluded
    if len(c.inbox) != 0 { t.Fatal("must not replay comments that predate opt-in") }
    gh.say(24, "stranger", "ChatGPT: impersonation")
    gh.say(5, "Arameair", "ChatGPT: other issue")
    gh.say(24, "Arameair", "Claude: another assistant")
    gh.say(24, "Arameair", "TASS: own echo "+Marker)
    gh.say(24, "Arameair", "ChatGPT: MAILBOX-01 — acknowledge; do not execute tools")
    gh.mu.Lock(); sourceID := gh.nextID; gh.mu.Unlock()
    c.SyncOnce()
    if len(c.inbox) != 1 { t.Fatalf("expected one allowed comment; inbox=%d", len(c.inbox)) }
    incoming := <-c.inbox
    if incoming.CommentID != sourceID || incoming.Issue != 24 || incoming.Author != "Arameair" {
        t.Fatalf("input lost GitHub provenance: %+v", incoming)
    }
    c.Answer(incoming)
    if eng.count() != 1 { t.Fatalf("expected real Engine API fixture call; got %d", eng.count()) }
    if req := eng.reqs[0]; req["actions"] != nil {
        if a, ok := req["actions"].([]any); !ok || len(a) > 0 { t.Fatalf("GitHub conversation offered actions: %v", req["actions"]) }
    }
    c.SyncOnce()
    gh.mu.Lock()
    posts := append([]string(nil), gh.posted...)
    gh.mu.Unlock()
    if len(posts) != 1 || !strings.Contains(posts[0], "issue#24: TASS:") ||
        !strings.Contains(posts[0], "comment "+fmtID(sourceID)) ||
        !strings.Contains(posts[0], "Cycle: tp_") {
        t.Fatalf("outgoing issue/cycle/input correlation absent: %v", posts)
    }
    got := c.st.RelaySeen[sourceID]
    if got.Stage != "POSTED" || got.OutgoingCommentID <= 0 || got.CycleID == "" || len(got.IdentityChecks) != 4 {
        t.Fatalf("receipt missing decision or real GitHub comment ID: %+v", got)
    }
    if h := c.Health("test"); h.ActionMode != "CHAT_ONLY" { t.Fatalf("relay changed action gate: %+v", h) }
    if audit, err := os.ReadFile(filepath.Join(dir,"relay-audit.jsonl")); err != nil ||
        !strings.Contains(string(audit), `"source_comment_id":`) ||
        !strings.Contains(string(audit), `"outgoing_comment_id":`) {
        t.Fatalf("metadata-only receipt not persisted: %v %s", err, audit)
    }
    c.SyncOnce()
    restart := pilotFixture(t, dir, ids, eng, gh)
    restart.SyncOnce()
    gh.mu.Lock(); count := len(gh.posted); gh.mu.Unlock()
    if count != 1 || len(restart.inbox) != 0 { t.Fatalf("re-poll/restart duplicated %d posts, inbox %d",count,len(restart.inbox)) }
}

func fmtID(v int64) string { return strconv.FormatInt(v, 10) }

func TestPilotMailboxOffLockedAndNoOtherIssue(t *testing.T) {
    ids, eng, gh := mailboxFixture(t)
    c := pilotFixture(t, t.TempDir(), ids, eng, gh)
    if err := c.SetSettings(Settings{Relay:true}); err != nil { t.Fatal(err) }
    c.SyncOnce() // prime
    ids.set("LOCKED")
    gh.say(24,"Arameair","ChatGPT: locked message")
    c.SyncOnce()
    if len(c.inbox) != 0 || eng.count() != 0 || !strings.Contains(c.Status().RelayHold,"LOCKED") {
        t.Fatalf("LOCKED must hold, status=%+v", c.Status())
    }
    if err := c.SetSettings(Settings{Relay:false,Learn:true}); err != nil { t.Fatal(err) }
    c.SyncOnce()
    ids.set("UNLOCKED")
    c.SyncOnce()
    if len(c.inbox) != 0 { t.Fatal("OFF must not accept even after Identity unlock") }
    if err := c.SetSettings(Settings{Relay:true,Learn:true}); err != nil { t.Fatal(err) }
    c.SyncOnce() // re-prime; old LOCKED comment must not be replayed
    if len(c.inbox) != 0 { t.Fatal("re-enable replayed a message from OFF/LOCKED interval") }
}

// A simulated fixture is not evidence that owner's actual local Nemotron ran.
