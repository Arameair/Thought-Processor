package tass

// One-issue, opt-in GPT-6 mailbox pilot. GitHub text is untrusted conversational
// input, never a permission grant. This is independent of Autonomous and Pulse.
// The older Identity Liaison uses a separate issue and is never consulted here.

import (
    "encoding/json"
    "fmt"
    "strings"
    "time"
)

const pilotMailboxIssue = 24
const pilotMailboxRepo = "Arameair/TASS"
const maxRelaySeen = 5000

// RelayReceipt is a durable, metadata-only record. No prompts or private
// model output belong in state.json or relay-audit.jsonl.
type RelayReceipt struct {
    Issue int `json:"issue"`
    SourceCommentID int64 `json:"source_comment_id"`
    Author string `json:"author,omitempty"`
    Stage string `json:"stage"` // RESERVED | ACCEPTED | POSTING | POSTED | AMBIGUOUS | CANCELLED
    CycleID string `json:"cycle_id,omitempty"`
    OutgoingCommentID int64 `json:"outgoing_comment_id,omitempty"`
    IdentityChecks []string `json:"identity_checks,omitempty"`
}

// This write must succeed before reserving, queueing or posting a remote ID.
// Core's general saveState logs errors; the mailbox instead fails closed.
func (c *Core) persistRelayState() error {
    raw, err := json.MarshalIndent(c.st, "", " ")
    if err != nil { return err }
    return c.write("state.json", string(raw))
}

func (c *Core) relayOn() bool {
    return c.cfg.IdentitySvc != nil && c.cfg.GitHub && c.cfg.Repo != "" && c.GetSettings().Relay
}

func (c *Core) relayLabel() string {
    if c.cfg.RelayLabel != "" { return c.cfg.RelayLabel }
    return "tass-core"
}

func (c *Core) relayAuthors() []string {
    if len(c.cfg.RelayAuthors) > 0 { return c.cfg.RelayAuthors }
    if i := strings.Index(c.cfg.Repo, "/"); i > 0 { return []string{c.cfg.Repo[:i]} }
    return nil
}

func (c *Core) relayAllowed(login string) bool {
    if login == "" { return false }
    for _, allowed := range c.relayAuthors() {
        if strings.EqualFold(login, allowed) { return true }
    }
    return false
}

func (c *Core) setRelayHold(reason string) {
    c.mu.Lock()
    c.relayHold = reason
    c.mu.Unlock()
}

func (c *Core) relayOnce() error {
    if !c.relayOn() {
        // A later re-enable must not replay messages written while relay was OFF.
        c.st.RelaySince = time.Time{}
        var keep []outgoing
        for _, o := range c.st.Outbox {
            if o.SourceCommentID == 0 { keep = append(keep, o); continue }
            r := c.st.RelaySeen[o.SourceCommentID]
            r.Stage = "CANCELLED"
            c.st.RelaySeen[o.SourceCommentID] = r
            c.event("github", fmt.Sprintf("Mailbox reply for comment %d cancelled because relay is OFF", o.SourceCommentID))
        }
        c.st.Outbox = keep
        c.setRelayHold("")
        return nil
    }
    if c.cfg.Repo != pilotMailboxRepo || c.relayLabel() != "tass-core" {
        c.setRelayHold("pilot restricted to Arameair/TASS issue #24 and tass-core")
        return nil
    }
    for _, action := range []string{"conversation", "github.sync"} {
        if ok, why := c.allowedNow(action); !ok {
            c.setRelayHold("Identity denied " + action + ": " + why)
            return nil
        }
    }
    issue, err := c.gh.MailboxIssue(c.cfg.Repo, pilotMailboxIssue)
    if err != nil { c.setRelayHold("mailbox issue verification failed"); return err }
    labelOK := false
    for _, label := range issue.Labels {
        if label.Name == c.relayLabel() { labelOK = true }
    }
    if issue.Number != pilotMailboxIssue || issue.State != "open" || issue.PullRequest != nil || !labelOK {
        c.setRelayHold("issue #24 is not an open tass-core issue")
        return nil
    }
    c.setRelayHold("")
    if c.st.RelaySince.IsZero() {
        // Capture EXISTING IDs before enabling intake. A timestamp alone is
        // insufficient: GitHub created_at is second-resolution. A comment
        // just before activation must never be replayed as a new command.
        for page := 1; page <= 10; page++ {
            comments, err := c.gh.MailboxComments(c.cfg.Repo, pilotMailboxIssue, time.Time{}, page)
            if err != nil { return fmt.Errorf("priming mailbox: %w", err) }
            for _, cm := range comments {
                if cm.ID <= 0 || cm.IssueNumber() != pilotMailboxIssue { continue }
                if len(c.st.RelaySeen) >= maxRelaySeen { return fmt.Errorf("mailbox baseline exceeds ledger limit") }
                if _, known := c.st.RelaySeen[cm.ID]; !known {
                    c.st.RelaySeen[cm.ID] = RelayReceipt{Issue:pilotMailboxIssue,SourceCommentID:cm.ID,
                        Author:cm.User.Login,Stage:"BASELINE"}
                }
            }
            if len(comments) < 100 {
                c.st.RelaySince = c.now().UTC().Add(-2*time.Second)
                if err := c.persistRelayState(); err != nil { return err }
                c.event("github", "Mailbox #24 primed from existing comment IDs; waiting for new ChatGPT: comments")
                return nil
            }
        }
        return fmt.Errorf("mailbox baseline exceeds 1000 comments; review required")
    }
    if err := c.postRelayOutbox(); err != nil { c.setRelayHold(err.Error()); return err }
    if err := c.readRelayMailbox(); err != nil { c.setRelayHold(err.Error()); return err }
    return nil
}

func (c *Core) readRelayMailbox() error {
    if len(c.st.RelaySeen) >= maxRelaySeen {
        return fmt.Errorf("mailbox ledger is full; require owner review before processing more")
    }
    for page := 1; page <= 10; page++ {
        comments, err := c.gh.MailboxComments(c.cfg.Repo, pilotMailboxIssue, c.st.RelaySince.Add(-time.Second), page)
        if err != nil { return fmt.Errorf("reading mailbox page %d: %w", page, err) }
        for _, cm := range comments {
            if cm.ID <= 0 || cm.IssueNumber() != pilotMailboxIssue || !cm.CreatedAt.After(c.st.RelaySince) {
                continue
            }
            if _, seen := c.st.RelaySeen[cm.ID]; seen { continue }
            body := strings.TrimSpace(cm.Body)
            if !c.relayAllowed(cm.User.Login) || !strings.HasPrefix(body, "ChatGPT:") ||
                strings.Contains(body, Marker) || strings.Contains(body, "<!-- identity-liaison") {
                continue
            }
            if len(body) < len("ChatGPT:")+2 || len(body) > 2048 { continue }
            if len(c.st.RelaySeen) >= maxRelaySeen {
                return fmt.Errorf("mailbox ledger limit reached; no silent history reset")
            }
            // Reserve the remote ID on disk BEFORE enqueuing. A crash may leave a
            // RESERVED/ACCEPTED item requiring owner review, but cannot replay it.
            r := RelayReceipt{Issue:pilotMailboxIssue,SourceCommentID:cm.ID,Author:cm.User.Login,Stage:"RESERVED"}
            c.st.RelaySeen[cm.ID] = r
            if err := c.persistRelayState(); err != nil { return fmt.Errorf("source reservation not durable: %w", err) }
            if err := c.queue(Incoming{Text:body,From:"github",Issue:pilotMailboxIssue,
                CommentID:cm.ID,Author:cm.User.Login}); err != nil {
                delete(c.st.RelaySeen, cm.ID) // refused delivery remains eligible for retry
                if saveErr := c.persistRelayState(); saveErr != nil { return saveErr }
                return fmt.Errorf("mailbox comment %d not queued: %w", cm.ID, err)
            }
            r.Stage = "ACCEPTED"
            c.st.RelaySeen[cm.ID] = r
            if err := c.persistRelayState(); err != nil { return err }
            c.appendLog("relay-audit.jsonl", map[string]any{"at":c.now().UTC(),"stage":"accepted",
                "issue":pilotMailboxIssue,"source_comment_id":cm.ID,"author":cm.User.Login})
            c.event("github", fmt.Sprintf("Accepted GitHub issue #24 comment %d from @%s", cm.ID, cm.User.Login))
        }
        if len(comments) < 100 { return nil }
    }
    return fmt.Errorf("mailbox pagination exceeded 1000 comments; paused rather than silently skipping")
}

func (c *Core) postRelayOutbox() error {
    for len(c.st.Outbox) > 0 {
        o := c.st.Outbox[0]
        if o.SourceCommentID <= 0 || o.Issue != pilotMailboxIssue || o.CycleID == "" {
            return fmt.Errorf("uncorrelated outbox entry refused by mailbox relay")
        }
        r, ok := c.st.RelaySeen[o.SourceCommentID]
        if !ok || r.Issue != pilotMailboxIssue {
            return fmt.Errorf("no reserved source for outgoing comment %d", o.SourceCommentID)
        }
        if r.Stage == "POSTED" {
            // Crash between persisted receipt and removal from outbox: no resend.
            c.st.Outbox = c.st.Outbox[1:]
            if err := c.persistRelayState(); err != nil { return err }
            continue
        }
        if r.Stage != "ACCEPTED" {
            return fmt.Errorf("mailbox comment %d has %s state; delivery needs owner review", o.SourceCommentID, r.Stage)
        }
        // An Identity denial occurs before POST and is not ambiguous.
        if ok, why := c.allowedNow("github.sync"); !ok {
            return fmt.Errorf("Identity denied GitHub reply for comment %d: %s", o.SourceCommentID, why)
        }
        // Persist POSTING before HTTP. Network timeout or restart is ambiguous:
        // no blind retry that could manufacture two publicly visible replies.
        r.Stage, r.CycleID, r.IdentityChecks = "POSTING", o.CycleID, o.IdentityChecks
        c.st.RelaySeen[o.SourceCommentID] = r
        if err := c.persistRelayState(); err != nil { return fmt.Errorf("posting reservation not durable: %w", err) }
        outputID, err := c.gh.PostComment(c.cfg.Repo, pilotMailboxIssue, o.Body)
        if err != nil {
            r.Stage = "AMBIGUOUS"
            c.st.RelaySeen[o.SourceCommentID] = r
            if saveErr := c.persistRelayState(); saveErr != nil { return saveErr }
            return fmt.Errorf("mailbox post for comment %d unverified: %w", o.SourceCommentID, err)
        }
        r.Stage, r.OutgoingCommentID = "POSTED", outputID
        c.st.RelaySeen[o.SourceCommentID] = r
        if err := c.persistRelayState(); err != nil { return err }
        c.appendLog("relay-audit.jsonl", map[string]any{"at":c.now().UTC(),"stage":"posted",
            "issue":pilotMailboxIssue,"source_comment_id":o.SourceCommentID,
            "outgoing_comment_id":outputID,"cycle_id":o.CycleID,"identity_checks":o.IdentityChecks})
        c.st.Outbox = c.st.Outbox[1:]
        if err := c.persistRelayState(); err != nil { return err }
        c.event("github", fmt.Sprintf("Mailbox receipt: input %d -> cycle %s -> output %d",
            o.SourceCommentID, o.CycleID, outputID))
    }
    return nil
}
