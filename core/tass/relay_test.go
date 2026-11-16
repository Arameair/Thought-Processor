package tass

import (
    "encoding/json"
    "os"
    "path/filepath"
    "strings"
    "strconv"
    "testing"

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
    if got, ok := eng.reqs[0]["source"].(string); !ok || !strings.Contains(got, "Arameair/TASS#24:comment-"+fmtID(sourceID)) {
        t.Fatalf("Engine did not retain exact GitHub comment provenance: %v", eng.reqs[0]["source"])
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

// #76/#81 failure modes: GitHub unavailable, Identity revoked between answer and post, a POST whose
// response is lost (uncertain delivery), and restart in each state. No blind resend, no replay.
func TestPilotMailboxOutagesRevokeAndUncertainDelivery(t *testing.T) {
    ids, eng, gh := mailboxFixture(t)
    dir := t.TempDir()
    c := pilotFixture(t, dir, ids, eng, gh)
    if err := c.SetSettings(Settings{Learn:true,Relay:true}); err != nil { t.Fatal(err) }
    c.SyncOnce() // prime

    // 1. GitHub unavailable: nothing accepted, a hold is shown, nothing lost once it is back.
    gh.say(24, "Arameair", "ChatGPT: MAILBOX-02 while GitHub is down")
    srcA := gh.nextID
    gh.mu.Lock(); gh.failRead = true; gh.mu.Unlock()
    c.SyncOnce()
    if len(c.inbox) != 0 || c.Status().RelayHold == "" { t.Fatalf("outage must hold visibly: inbox=%d hold=%q", len(c.inbox), c.Status().RelayHold) }
    gh.mu.Lock(); gh.failRead = false; gh.mu.Unlock()
    c.SyncOnce()
    if len(c.inbox) != 1 { t.Fatalf("comment written during outage must be read once afterwards; inbox=%d", len(c.inbox)) }
    in := <-c.inbox
    if in.CommentID != srcA { t.Fatalf("wrong source %d != %d", in.CommentID, srcA) }
    c.Answer(in)

    // 2. Identity revoked (LOCKED) after the answer: the post waits, it is not dropped or sent.
    ids.set("LOCKED")
    c.SyncOnce()
    gh.mu.Lock(); n := len(gh.posted); gh.mu.Unlock()
    if n != 0 || c.st.RelaySeen[srcA].Stage != "ACCEPTED" || len(c.st.Outbox) != 1 {
        t.Fatalf("revoked Identity must stop the post and keep it: posts=%d stage=%s outbox=%d", n, c.st.RelaySeen[srcA].Stage, len(c.st.Outbox))
    }
    // restart while waiting: still exactly one pending reply, no replayed input
    c = pilotFixture(t, dir, ids, eng, gh)
    ids.set("UNLOCKED")
    c.SyncOnce()
    gh.mu.Lock(); n = len(gh.posted); gh.mu.Unlock()
    if n != 1 || c.st.RelaySeen[srcA].Stage != "POSTED" || len(c.inbox) != 0 || eng.count() != 1 {
        t.Fatalf("after unlock+restart: posts=%d stage=%s inbox=%d engine=%d", n, c.st.RelaySeen[srcA].Stage, len(c.inbox), eng.count())
    }

    // 3. GitHub down at POST time (nothing created): uncertain from TASS's side → AMBIGUOUS, no retry.
    gh.say(24, "Arameair", "ChatGPT: MAILBOX-03 reply will hit an outage")
    srcB := gh.nextID
    c.SyncOnce(); c.Answer(<-c.inbox)
    gh.mu.Lock(); gh.failPost = "down"; gh.mu.Unlock()
    c.SyncOnce()
    if c.st.RelaySeen[srcB].Stage != "AMBIGUOUS" { t.Fatalf("failed POST must be AMBIGUOUS, got %s", c.st.RelaySeen[srcB].Stage) }

    // 4. POST created the comment but the response was lost: also AMBIGUOUS; never posted twice.
    gh.mu.Lock(); gh.failPost = ""; gh.mu.Unlock()
    c2 := pilotFixture(t, t.TempDir(), ids, eng, gh)
    if err := c2.SetSettings(Settings{Learn:true,Relay:true}); err != nil { t.Fatal(err) }
    c2.SyncOnce()
    gh.say(24, "Arameair", "ChatGPT: MAILBOX-04 response will be lost")
    srcC := gh.nextID
    c2.SyncOnce(); c2.Answer(<-c2.inbox)
    gh.mu.Lock(); gh.failPost = "lost"; before := len(gh.posted); gh.mu.Unlock()
    c2.SyncOnce()
    gh.mu.Lock(); gh.failPost = ""; gh.mu.Unlock()
    for i := 0; i < 3; i++ { c2.SyncOnce() }
    c2 = pilotFixture(t, c2.cfg.DataDir, ids, eng, gh) // restart
    for i := 0; i < 3; i++ { c2.SyncOnce() }
    gh.mu.Lock(); after := len(gh.posted); gh.mu.Unlock()
    if c2.st.RelaySeen[srcC].Stage != "AMBIGUOUS" || after-before != 1 {
        t.Fatalf("lost response: stage=%s posts=%d (must be exactly 1, never resent)", c2.st.RelaySeen[srcC].Stage, after-before)
    }
    if !strings.Contains(c2.Status().RelayHold, "owner review") { t.Fatalf("ambiguous delivery must be shown for owner review: %q", c2.Status().RelayHold) }
    // the original Core still has its own AMBIGUOUS item and has not resent either
    for i := 0; i < 2; i++ { c.SyncOnce() }
    gh.mu.Lock(); final := len(gh.posted); gh.mu.Unlock()
    if final != after { t.Fatalf("AMBIGUOUS item was resent: %d -> %d", after, final) }
    if h := c.Health("test"); h.ActionMode != "CHAT_ONLY" { t.Fatalf("mailbox changed the action gate: %+v", h) }
}
