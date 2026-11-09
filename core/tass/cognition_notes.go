package tass

// P0 of the Cognition → Development → Cognition cycle (#106, design PR #107 §3.1; owner decision D1).
//
// "Cognition shows what was produced": the single-Thought detail now carries what TASS actually wrote
// (the Engine's saved Notes), correlated with the developed history steps and, for supervised nudges,
// Core's own receipt. Rules:
//
//   - Only CognitionThought (the single-Thought detail, behind cognitionAccess and the reviewed
//     Identity → Engine link) calls this. The library list, exports, Copy Evidence, logs and GitHub
//     sync never carry Note text.
//   - Bounded: at most cognitionDevelopmentsMax developments, newest first; text clipped.
//   - Read only: Core's nudge receipts are READ (outcome.json); no Engine call, no write, no model.
//   - Honest: "saved and correlated" is persistence, not usefulness; a suggested state is never
//     applied by any code path; an evicted Note is STEP_ONLY, never invented.

import (
	"sort"
	"time"
	"unicode/utf8"

	"github.com/Arameair/Thought-Processor/core/engine"
)

const (
	cognitionHistoryMax      = 64
	cognitionDevelopmentsMax = 50
	cognitionNoteMaxRunes    = 600 // the Engine keeps at most 400; anything longer is clipped, not trusted
)

// cognitionRef (cognition.go) keeps only reference-shaped IDs; anything else is NOT RECORDED.

func cognitionNudgeRef(s string) string {
	if nudgeIDOK(s) {
		return s
	}
	return ""
}

func clipRunes(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	r := []rune(s)
	return string(r[:n]) + "…"
}

type CognitionNote struct {
	At             time.Time `json:"at"`
	Text           string    `json:"text"`
	By             string    `json:"by"`
	SuggestedState string    `json:"suggested_state"`
	// No code path in the Engine applies a suggested state; Core says so instead of implying it.
	SuggestedApplied bool `json:"suggested_state_applied"`
}

type CognitionDevStep struct {
	At   time.Time `json:"at"`
	Kind string    `json:"kind"`
	By   string    `json:"by,omitempty"`
}

type CognitionDevelopment struct {
	RequestID     string            `json:"request_id"` // "" = NOT RECORDED
	At            time.Time         `json:"at"`
	Note          *CognitionNote    `json:"note"` // nil: no saved Note matches (evicted or never saved)
	Step          *CognitionDevStep `json:"step"` // nil: no developed step matches
	SourceNudgeID string            `json:"source_nudge_id,omitempty"`
	// NOTE_AND_STEP_CORRELATED | NOTE_ONLY | STEP_ONLY | IDS_DISAGREE. Persistence only, never quality.
	Persistence string `json:"persistence"`
	// Core's own receipt for the nudge, when the nudge is in THIS owner's journal:
	// COMPLETED_VERIFIED | COMPLETED_NOT_VERIFIED | <run state> | NO_CORE_RECEIPT | NOT_IN_THIS_OWNERS_JOURNAL | "" (no nudge)
	NudgeReceipt string `json:"nudge_receipt,omitempty"`
	Meaning      string `json:"meaning"`
}

// cognitionDevelopments correlates Notes and developed steps by worker_request_id.
func (c *Core) cognitionDevelopments(scope cognitionScope, thoughtID string, th engine.Thought) ([]CognitionDevelopment, int, bool) {
	byReq := map[string]*CognitionDevelopment{}
	var all []*CognitionDevelopment
	get := func(req string, at time.Time) *CognitionDevelopment {
		if req != "" {
			if d, ok := byReq[req]; ok {
				return d
			}
		}
		d := &CognitionDevelopment{RequestID: req, At: at}
		if req != "" {
			byReq[req] = d
		}
		all = append(all, d)
		return d
	}
	nudgeOf := func(d *CognitionDevelopment, nid string) {
		switch {
		case nid == "":
		case d.SourceNudgeID == "":
			d.SourceNudgeID = nid
		case d.SourceNudgeID != nid:
			d.Persistence = "IDS_DISAGREE"
		}
	}
	for _, n := range th.Notes {
		d := get(cognitionRef(n.RequestID), n.At)
		if d.Note != nil { // two Notes for one request ID: keep both visible, never merge
			d = &CognitionDevelopment{RequestID: d.RequestID, At: n.At, Persistence: "IDS_DISAGREE"}
			all = append(all, d)
		}
		d.Note = &CognitionNote{At: n.At, Text: clipRunes(n.Text, cognitionNoteMaxRunes), By: clipRunes(n.By, 60),
			SuggestedState: clipRunes(n.SuggestedState, 16)}
		nudgeOf(d, cognitionNudgeRef(n.NudgeID))
		if n.At.After(d.At) {
			d.At = n.At
		}
	}
	for _, h := range th.History {
		if h.Kind != "developed" {
			continue
		}
		d := get(cognitionRef(h.RequestID), h.At)
		if d.Step != nil {
			d = &CognitionDevelopment{RequestID: d.RequestID, At: h.At, Persistence: "IDS_DISAGREE"}
			all = append(all, d)
		}
		d.Step = &CognitionDevStep{At: h.At, Kind: h.Kind, By: clipRunes(h.By, 40)}
		nudgeOf(d, cognitionNudgeRef(h.NudgeID))
	}
	// This owner's nudges for this Thought (owner AND linked Engine entity must match, as in NudgeHistory).
	mine := map[string]bool{}
	c.nudgeMu.Lock()
	journal, _, err := c.readNudges()
	c.nudgeMu.Unlock()
	if err == nil {
		for _, n := range journal {
			if n.OwnerEntityID == scope.identityID && n.SourceEngineEntityID == scope.engineID && n.SourceThoughtID == thoughtID {
				mine[n.NudgeID] = true
			}
		}
	}
	for _, d := range all {
		if d.Persistence == "" {
			switch {
			case d.Note != nil && d.Step != nil && d.RequestID != "":
				d.Persistence = "NOTE_AND_STEP_CORRELATED"
			case d.Note != nil:
				d.Persistence = "NOTE_ONLY"
			default:
				d.Persistence = "STEP_ONLY"
			}
		}
		if d.SourceNudgeID != "" {
			d.NudgeReceipt = c.nudgeReceipt(d.SourceNudgeID, d.RequestID, mine)
		}
		d.Meaning = developmentMeaning(d)
	}
	sort.SliceStable(all, func(i, j int) bool { return all[i].At.After(all[j].At) })
	total := len(all)
	if total > cognitionDevelopmentsMax {
		all = all[:cognitionDevelopmentsMax]
	}
	out := make([]CognitionDevelopment, 0, len(all))
	for _, d := range all {
		out = append(out, *d)
	}
	return out, total, total > cognitionDevelopmentsMax
}

// nudgeReceipt READS Core's outcome receipt for a nudge in this owner's journal. No Engine call, no write.
func (c *Core) nudgeReceipt(nid, requestID string, mine map[string]bool) string {
	if !mine[nid] {
		return "NOT_IN_THIS_OWNERS_JOURNAL"
	}
	var out coreOutcome
	ex, ok, _ := coreReadOnce(c.runPath(nid, "outcome.json"), &out)
	switch {
	case !ex:
		return "NO_CORE_RECEIPT"
	case !ok:
		return "RECOVERY_REQUIRED"
	case out.Run.State == "COMPLETED" && out.Verified && out.Run.RequestID == requestID && requestID != "":
		return "COMPLETED_VERIFIED"
	case out.Run.State == "COMPLETED":
		return "COMPLETED_NOT_VERIFIED"
	}
	return clipRunes(out.Run.State, 24)
}

func developmentMeaning(d *CognitionDevelopment) string {
	switch d.Persistence {
	case "NOTE_AND_STEP_CORRELATED":
		s := "Saved: one Note and one developed step share this request ID. That shows it was saved, not that it answers the Thought's question."
		if d.NudgeReceipt == "COMPLETED_VERIFIED" {
			s += " Core's receipt for the supervised nudge verified the same IDs."
		}
		return s
	case "NOTE_ONLY":
		return "A Note is saved, but no developed step with the same request ID was found."
	case "STEP_ONLY":
		return "A developed step is recorded, but its Note is no longer stored (the Engine keeps the newest 200 Notes) or was never saved. Nothing is shown in its place."
	}
	return "The recorded IDs disagree; both records are shown as they are."
}
