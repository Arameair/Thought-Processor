package tass

// GitHub conversation relay (app mode). With the owner's page switch "Talk on GitHub" ON:
//
//   - every sync, TASS reads new comments on open issues labelled tass-core in the owner's
//     repository, written by the allowed authors (default: the repository owner), and only
//     comments made after he first looked. Never his own posts (Marker) or the older
//     Identity Liaison's (it uses the "tass" label and its own marker).
//   - each message goes through the same path as the local Talk box: Identity checks,
//     one Thought Engine cycle with the real model, and his reply is posted on the issue.
//   - "find lessons" and "study <lesson>" work from GitHub too (a study is a zero-action job);
//     the graded answers are posted back on the issue. Approving a lesson is NOT possible
//     from GitHub: only the owner's Approve click on the local page.
//
// Identity must allow conversation and github.sync at that moment, or nothing is read or posted.

import (
	"fmt"
	"strings"
)

func (c *Core) relayOn() bool {
	return c.cfg.IdentitySvc != nil && c.cfg.Repo != "" && c.GetSettings().Relay
}

func (c *Core) relayLabel() string { return firstNonEmpty(c.cfg.RelayLabel, "tass-core") }

func (c *Core) relayAuthors() []string {
	if len(c.cfg.RelayAuthors) > 0 {
		return c.cfg.RelayAuthors
	}
	if i := strings.Index(c.cfg.Repo, "/"); i > 0 {
		return []string{c.cfg.Repo[:i]}
	}
	return nil
}

func (c *Core) relayAllowed(login string) bool {
	for _, a := range c.relayAuthors() {
		if strings.EqualFold(a, login) {
			return true
		}
	}
	return false
}

func (c *Core) setRelayHold(why string) {
	c.mu.Lock()
	c.relayHold = why
	c.mu.Unlock()
}

// relayOnce: post waiting replies, then read new messages. Errors are GitHub problems
// (reported like any sync problem); Identity refusals only put the relay on hold.
func (c *Core) relayOnce() error {
	if !c.relayOn() {
		return nil
	}
	for _, action := range []string{"conversation", "github.sync"} {
		if ok, why := c.allowedNow(action); !ok {
			c.setRelayHold("Identity denied " + action + ": " + why)
			return nil // replies stay in the outbox until Identity allows them
		}
	}
	c.setRelayHold("")
	if err := c.postOutbox(c.cfg.Repo); err != nil {
		return fmt.Errorf("relay post: %w", err)
	}
	if err := c.readIssues(c.cfg.Repo, c.relayLabel(), c.relayAllowed); err != nil {
		return fmt.Errorf("relay read: %w", err)
	}
	return nil
}

// toIssue queues a post for a GitHub issue (sent on the next sync). Never blocks.
func (c *Core) toIssue(issue int, body string) {
	select {
	case c.replies <- outgoing{issue, body + "\n" + Marker}:
		c.mu.Lock()
		c.pending++
		c.mu.Unlock()
	default:
		c.event("problem", fmt.Sprintf("Reply for GitHub issue #%d dropped: too many replies waiting.", issue))
	}
}

// lessonSay: a lesson-command answer, shown on the page and, for GitHub, posted on the issue.
func (c *Core) lessonSay(m Incoming, text string) {
	c.event("tass", text)
	if m.From == "github" && m.Issue > 0 {
		c.toIssue(m.Issue, text)
	}
}

// postLessonResult posts a graded study back to the issue it was asked on (learnMu held).
func (c *Core) postLessonResult(ls *LessonState, jobState string) {
	if ls.AskedOn <= 0 {
		return
	}
	var b strings.Builder
	fmt.Fprintf(&b, "**Lesson %q** (`%s`) · job `%s` · attempt %d\n\n", ls.Title, ls.Path, ls.JobID, ls.Attempts)
	if jobState != "DONE" {
		fmt.Fprintf(&b, "The study job ended **%s**, so nothing was graded: %s\n", jobState, oneLine(ls.Note, 300))
	} else {
		fmt.Fprintf(&b, "**%d/%d checks right** → %s\n\n", ls.Right, ls.Checks, ls.Status)
		for _, r := range ls.Results {
			mark := "✗"
			if r.Right {
				mark = "✓"
			}
			fmt.Fprintf(&b, "- %s %s → %s\n", mark, r.Check, firstNonEmpty(r.Given, "(no answer)"))
		}
		if ls.Remember != "" {
			fmt.Fprintf(&b, "\nWhat I claim I learned: %s\n", oneLine(ls.Remember, 300))
		}
	}
	switch ls.Status {
	case "PASSED":
		if c.settings.Autonomous {
			b.WriteString("\nAutonomous is ON, so it is approved automatically.")
		} else {
			b.WriteString("\nWaiting for my owner to **Approve** (or Retry) on the local TASS page. It can't be approved from GitHub.")
		}
	case "RETRY":
		b.WriteString("\nNot passed yet. Say `study " + ls.Title + "` to try again.")
	case "STUCK":
		b.WriteString("\nStuck after the maximum attempts: my owner can press Retry on the local page.")
	}
	c.toIssue(ls.AskedOn, b.String())
	ls.AskedOn = 0
}
