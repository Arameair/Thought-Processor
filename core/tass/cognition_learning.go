package tass

import (
	"fmt"
	"strings"
)

// lessonEvidence adds the learning record of one lesson to its Cognition card (P3): source revision,
// every attempt with its Engine job and cycle IDs, Core's grade, the owner's approval, fresh-task
// retests and the reminder that no skill comes from learning. Anything not recorded says so.
func (c *Core) lessonEvidence(path string) ([]CognitionField, []string) {
	c.learnMu.Lock()
	defer c.learnMu.Unlock()
	ls := c.learn.Lessons[path]
	if ls == nil {
		return []CognitionField{cognitionField("Learning record", "NOT RECORDED")}, nil
	}
	nr := func(s string) string { return cognitionFirst(s, "NOT RECORDED") }
	var cycles []string
	seen := map[string]bool{}
	addCycle := func(id string) {
		if id != "" && !seen[id] && len(cycles) < 40 {
			seen[id] = true
			cycles = append(cycles, id)
		}
	}
	var att []string
	for i, h := range ls.History {
		if i >= 12 {
			att = append(att, fmt.Sprintf("… %d more", len(ls.History)-i))
			break
		}
		addCycle(h.CycleID)
		att = append(att, fmt.Sprintf("job %s · cycle %s · %s → %s %d/%d (%s)", h.JobID, nr(h.CycleID), h.State, nr(h.Outcome), h.Right, h.Checks, h.Source))
	}
	grade := "NOT RECORDED (not graded yet)"
	if len(ls.Results) > 0 {
		grade = fmt.Sprintf("%d/%d graded by Core · %s", ls.Right, ls.Checks, ls.Status)
	} else if ls.Status == "MODEL_FAILED" {
		grade = "NOT GRADED: model/Engine failure"
	}
	approval := "not approved"
	if ls.LearnedAt != nil {
		approval = "owner approved " + ls.LearnedAt.UTC().Format("2006-01-02 15:04Z") + " (knowledge only; not a skill grant)"
	}
	var rt []string
	for _, r := range ls.Retests {
		addCycle(r.CycleID)
		rt = append(rt, fmt.Sprintf("job %s · cycle %s · %s %d/%d · %s", r.JobID, nr(r.CycleID), r.Outcome, r.Right, r.Checks, r.At.UTC().Format("2006-01-02 15:04Z")))
	}
	if ls.RetestPending != nil {
		rt = append(rt, "job "+ls.RetestPending.JobID+" · QUEUED · cycle NOT RECORDED yet")
	}
	addCycle(ls.CycleID)
	return []CognitionField{
		cognitionField("Source", path+" @ "+nr(ls.SHA)),
		cognitionField("Attempts", nr(strings.Join(att, "\n"))),
		cognitionField("Grade", grade),
		cognitionField("Owner approval", approval),
		cognitionField("Fresh-task retests", cognitionFirst(strings.Join(rt, "\n"), "none yet")),
		cognitionField("Skill from this lesson", "none: learning never grants or runs a skill (skill runs have their own receipts)"),
	}, cycles
}
