package tass

import (
    "fmt"
    "strings"
)

// handleLessonChat is a narrowly scoped, owner-origin request to START the
// existing Markdown lesson loop. Notebook code is never executed by this path.
// This runs only after Answer has passed the conversation/model/Thought Engine
// Identity checks; pulse and learning require additional live grants below.
func (c *Core) handleLessonChat(m Incoming) bool {
    if m.From != "local" {
        return false
    }
    spoken := strings.TrimRight(strings.TrimSpace(strings.ToLower(m.Text)), " .!?")
    if spoken != "tass, learn how to find and run a lesson" {
        return false
    }
    if c.cfg.IdentitySvc == nil {
        c.event("problem", "Lesson request not started: this operation requires the installed Identity service.")
        return true
    }
    if c.cfg.Repo == "" || c.cfg.Branch == "" {
        c.event("problem", "Lesson request not started: no owner-approved TASS lesson repository and branch are configured for this installation.")
        return true
    }
    for _, action := range []string{"lesson.learn", "pulse.run"} {
        if ok, why := c.allowedNow(action); !ok {
            c.event("problem", fmt.Sprintf("Lesson request blocked: %s (grant %s).", why, action))
            return true
        }
    }
    if err := c.SetSettings(Settings{Pulse: true, Learn: true}); err != nil {
        c.event("problem", "Lesson request could not persist settings: "+err.Error())
        return true
    }
    c.event("tass", fmt.Sprintf(
        "I requested the existing 20-second lesson pulse for approved Markdown lessons at %s@%s. "+
        "Watch the lesson status for a real QUEUED/PASSED/RETRY receipt. "+
        "I cannot execute notebook Python yet: .ipynb lessons remain NEEDS_WORKSPACE and require a restricted runner.",
        c.cfg.Repo, c.cfg.Branch))
    return true
}
