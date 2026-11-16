package tass

// Stage 1 slice 1 (plan PR #109 §8.2, WP1-A): a truthful, METADATA-ONLY preview on each Thought card.
//
//   lifecycle      the Engine's own Thought state: active | sleeping | resolved. Nothing else exists
//                  per Thought, and nothing is invented: no BUSY, WORKING or THINKING.
//   notes_count    how many Notes the Engine sent for the Thought (it keeps its newest 200).
//   latest_output  the newest recorded development: its kind, time and recorded source.
//   run_activity   the newest supervised nudge for the Thought that Core holds a receipt for.
//
// Rules:
//   - No Note text. thoughtPreview never reads ThoughtNote.Text; Note text stays in the single-Thought
//     detail (owner decision D1).
//   - No new Engine request: the preview is computed from the entity payload the library already
//     reads. Run activity comes from Core's own journal and receipt files only.
//   - A receipt is history, not live work. "Submitted, no outcome recorded" is exactly that; Core
//     does not ask the Engine whether the run is still going when it draws a card.

import (
	"strings"
	"time"

	"github.com/Arameair/Thought-Processor/core/engine"
)

// engineNotesKept: Thought Engine 40d4e91 keeps the newest 200 Notes per Thought.
const engineNotesKept = 200

type CognitionOutput struct {
	Kind string    `json:"kind"` // DEVELOPED_NOTE | DEVELOPED_STEP_NOTE_NOT_STORED
	At   time.Time `json:"at"`
	By   string    `json:"by"` // engine | owner-nudge | owner | NOT RECORDED
}

type CognitionRunActivity struct {
	// SUBMITTED_NO_OUTCOME | COMPLETED_VERIFIED | STOPPED | FAILED | UNCERTAIN | NEEDS_RECOVERY.
	// As in the nudge history (runView): COMPLETED without Core's verification is NEEDS_RECOVERY.
	State   string    `json:"state"`
	NudgeID string    `json:"nudge_id"`
	At      time.Time `json:"at"`
}

func cognitionLifecycle(state string) string {
	switch state {
	case "active", "sleeping", "resolved":
		return state
	}
	return "NOT RECORDED"
}

// cognitionSource normalizes who a development is recorded as coming from. The pinned Engine writes
// "heartbeat" (its own schedule), "owner" (the owner asked the Engine directly) and "owner-nudge"
// (a supervised nudge through Core) on developed steps, and "<worker> (<source>)" on Notes.
func cognitionSource(by string) string {
	if i := strings.LastIndex(by, "("); i >= 0 && strings.HasSuffix(by, ")") {
		by = by[i+1 : len(by)-1]
	}
	switch by {
	case "engine", "heartbeat":
		return "engine"
	case "owner-nudge", "owner nudge":
		return "owner-nudge"
	case "owner":
		return "owner"
	}
	return "NOT RECORDED"
}

// thoughtPreview derives the card preview from one Engine Thought. It reads no Note text.
func thoughtPreview(th engine.Thought) (lifecycle string, notes int, capped bool, latest *CognitionOutput) {
	lifecycle, notes, capped = cognitionLifecycle(th.State), len(th.Notes), len(th.Notes) >= engineNotesKept
	stepBy := map[string]string{} // developed step source by request ID
	noted := map[string]bool{}    // request IDs that have a saved Note
	for _, h := range th.History {
		if h.Kind == "developed" {
			if req := cognitionRef(h.RequestID); req != "" {
				stepBy[req] = cognitionSource(h.By)
			}
		}
	}
	consider := func(kind string, at time.Time, by string) {
		if at.IsZero() {
			return
		}
		if latest == nil || at.After(latest.At) || (at.Equal(latest.At) && kind == "DEVELOPED_NOTE" && latest.Kind != "DEVELOPED_NOTE") {
			latest = &CognitionOutput{Kind: kind, At: at.UTC(), By: by}
		}
	}
	for _, n := range th.Notes {
		req := cognitionRef(n.RequestID)
		if req != "" {
			noted[req] = true
		}
		by := cognitionSource(n.By)
		switch {
		case req != "" && stepBy[req] != "" && stepBy[req] != "NOT RECORDED":
			by = stepBy[req] // the correlated step's recorded source wins
		case by == "NOT RECORDED" && cognitionNudgeRef(n.NudgeID) != "":
			by = "owner-nudge"
		}
		consider("DEVELOPED_NOTE", n.At, by)
	}
	for _, h := range th.History {
		if h.Kind != "developed" {
			continue
		}
		if req := cognitionRef(h.RequestID); req != "" && noted[req] {
			continue // its Note is stored; counted above
		}
		consider("DEVELOPED_STEP_NOTE_NOT_STORED", h.At, cognitionSource(h.By))
	}
	return
}

// runActivityByThought: for each Thought, the newest supervised nudge that Core holds a submission
// receipt for, with the state Core's own receipts record. One pass per library read: the nudge
// journal once, then Core's receipt files. It never calls the Engine and never writes.
// Owner, linked Engine entity and Thought must match in BOTH the journal and the receipt.
func (c *Core) runActivityByThought(scope cognitionScope) map[string]CognitionRunActivity {
	out := map[string]CognitionRunActivity{}
	if scope.engineID == "" {
		return out
	}
	c.nudgeMu.Lock()
	journal, _, err := c.readNudges()
	c.nudgeMu.Unlock()
	if err != nil {
		return out
	}
	newest := map[string]time.Time{}
	seen := map[string]bool{}
	for _, n := range journal {
		if n.OwnerEntityID != scope.identityID || n.SourceEngineEntityID != scope.engineID || !nudgeIDOK(n.NudgeID) ||
			!engine.CognitionID.MatchString(n.SourceThoughtID) || seen[n.NudgeID] {
			continue
		}
		seen[n.NudgeID] = true
		var sub submittedRec
		sex, sok, _ := coreReadOnce(c.runPath(n.NudgeID, "submitted.json"), &sub)
		if !sex {
			continue // a draft that was never submitted is not run activity
		}
		ra := CognitionRunActivity{NudgeID: n.NudgeID, At: n.CreatedAt.UTC()}
		switch {
		case !sok || sub.NudgeID != n.NudgeID:
			ra.State = "NEEDS_RECOVERY"
		case sub.OwnerEntityID != scope.identityID || sub.EngineEntityID != scope.engineID || sub.ThoughtID != n.SourceThoughtID:
			continue // the receipt is for another owner, Engine entity or Thought: not shown here
		default:
			ra.State, ra.At = "SUBMITTED_NO_OUTCOME", sub.At.UTC()
			var oc coreOutcome
			var vd coreVoid
			if oex, ook, _ := coreReadOnce(c.runPath(n.NudgeID, "outcome.json"), &oc); oex {
				ra.State = "NEEDS_RECOVERY"
				if ook {
					ra.At = oc.At.UTC()
					switch {
					case oc.Run.State == "COMPLETED" && oc.Verified:
						ra.State = "COMPLETED_VERIFIED"
					case oc.Run.State == "VOID":
						ra.State = "STOPPED"
					case oc.Run.State == "FAILED" || oc.Run.State == "UNCERTAIN":
						ra.State = oc.Run.State
					}
				}
			} else if vex, vok, _ := coreReadOnce(c.runPath(n.NudgeID, "void.json"), &vd); vex {
				ra.State = "NEEDS_RECOVERY"
				if vok {
					ra.State, ra.At = "STOPPED", vd.At.UTC()
				}
			}
		}
		order := sub.At
		if order.IsZero() {
			order = n.CreatedAt
		}
		if prev, ok := newest[n.SourceThoughtID]; !ok || order.After(prev) {
			newest[n.SourceThoughtID], out[n.SourceThoughtID] = order, ra
		}
	}
	return out
}
