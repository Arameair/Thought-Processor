package tass

// Lesson job IDs and attempt history (Thought-Processor #71).
//
// The Engine keeps every job it has ever accepted (jobs.jsonl) and refuses a
// second job with the same ID (HTTP 409). The Core used to build the ID from
// "attempts since the last Retry", which Retry reset to 0, so the next Study
// re-used an ID the Engine already held. Now:
//
//   - JobSeq is the highest job number ever used for a lesson revision. It never goes down:
//     not on Retry, restart, or re-discovery.
//   - Every finished, failed or reconciled attempt is appended to History and kept.
//   - If the Engine still answers 409 (for example after the Core's lessons.json was lost),
//     the Core looks the job up: a QUEUED/RUNNING job is attached to, a finished or unknown
//     one is recorded as history (never graded or approved), and the next number is tried.

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Arameair/Thought-Processor/core/engine"
)

// AttemptRecord is one study attempt as it actually happened. Records are only appended.
type AttemptRecord struct {
	JobID   string    `json:"job_id"`
	CycleID string    `json:"cycle_id,omitempty"`
	SHA     string    `json:"sha,omitempty"`     // lesson revision studied
	State   string    `json:"state"`             // Engine job state: DONE | FAILED | INTERRUPTED | REFUSED | QUEUED | RUNNING | UNKNOWN
	Outcome string    `json:"outcome,omitempty"` // Core's verdict: PASSED | RETRY | STUCK | MODEL_FAILED | NOT_COLLECTED
	Right   int       `json:"right"`
	Checks  int       `json:"checks"`
	Reply   string    `json:"reply,omitempty"`
	Note    string    `json:"note,omitempty"`
	Source  string    `json:"source"` // "study" | "reconciled 409" | "recorded before history existed"
	At      time.Time `json:"at"`
}

const (
	maxLessonHistory = 50 // per lesson; oldest are dropped only past this bound
	max409Probes     = 8  // job numbers tried after a 409 before stopping with an error
)

func lessonJobPrefix(p, sha string) string { return "lesson_" + hash(p + sha)[:10] + "_" }

func lessonJobID(p, sha string, seq int) string { return lessonJobPrefix(p, sha) + strconv.Itoa(seq) }

// jobSeqOf returns n for an ID "<prefix>n" of this lesson revision, else 0.
func jobSeqOf(p, sha, id string) int {
	pre := lessonJobPrefix(p, sha)
	if !strings.HasPrefix(id, pre) {
		return 0
	}
	n, err := strconv.Atoi(strings.TrimPrefix(id, pre))
	if err != nil || n < 0 {
		return 0
	}
	return n
}

// migrateLessonLocked upgrades lesson state written by an older Core: the job counter
// can never be below a job ID already used, and an answer still shown on the card is
// kept as a history record before anything else changes. Caller holds learnMu.
func migrateLessonLocked(ls *LessonState, now time.Time) bool {
	changed := false
	if ls.Attempts > ls.JobSeq {
		ls.JobSeq, changed = ls.Attempts, true
	}
	if n := jobSeqOf(ls.Path, ls.SHA, ls.JobID); n > ls.JobSeq {
		ls.JobSeq, changed = n, true
	}
	for _, h := range ls.History {
		if n := jobSeqOf(ls.Path, ls.SHA, h.JobID); n > ls.JobSeq {
			ls.JobSeq, changed = n, true
		}
	}
	if len(ls.History) == 0 && ls.JobID != "" && ls.Status != "QUEUED" && (ls.Reply != "" || ls.Status == "MODEL_FAILED") {
		at := now
		if ls.StudiedAt != nil {
			at = *ls.StudiedAt
		}
		outcome := ls.Status
		if outcome == "NEW" { // a Retry already reset the card; the evidence is what is left
			outcome = ""
		}
		ls.History = append(ls.History, AttemptRecord{JobID: ls.JobID, CycleID: ls.CycleID, SHA: ls.SHA, State: "UNKNOWN",
			Outcome: outcome, Right: ls.Right, Checks: ls.Checks, Reply: ls.Reply, Note: ls.Note,
			Source: "recorded before history existed", At: at})
		changed = true
	}
	return changed
}

func (ls *LessonState) addHistory(r AttemptRecord) {
	ls.History = append(ls.History, r)
	if len(ls.History) > maxLessonHistory {
		ls.History = append([]AttemptRecord(nil), ls.History[len(ls.History)-maxLessonHistory:]...)
	}
}

// submitLessonJob hands one lesson attempt to the Engine under a job ID it has never
// seen, or attaches to the same lesson's job that the Engine is still working on.
// It returns the ID in use and its sequence number. History records for reconciled
// IDs are returned for the caller to store (the caller holds no lock while this runs).
func (c *Core) submitLessonJob(p, sha string, lastSeq int, build func(id string) engine.Job) (id string, seq int, attached bool, found []AttemptRecord, err error) {
	for k := 1; k <= max409Probes; k++ {
		seq = lastSeq + k
		id = lessonJobID(p, sha, seq)
		err = c.engine.SubmitJob(build(id))
		if err == nil {
			return id, seq, false, found, nil
		}
		if !errors.Is(err, engine.ErrJobExists) {
			return "", lastSeq, false, found, err
		}
		// The Engine already holds this ID. Look before doing anything else.
		var job *engine.JobStatus
		if ps, perr := c.engine.PulseStatus(); perr == nil {
			for i := range ps.Jobs {
				if ps.Jobs[i].JobID == id {
					job = &ps.Jobs[i]
				}
			}
		}
		now := c.now().UTC()
		switch {
		case job != nil && (job.State == "QUEUED" || job.State == "RUNNING"):
			c.event("sync", fmt.Sprintf("Engine is already working on lesson job %s (%s); attached to it instead of submitting again.", id, job.State))
			return id, seq, true, found, nil
		case job != nil:
			found = append(found, AttemptRecord{JobID: id, CycleID: job.CycleID, SHA: sha, State: job.State, Outcome: "NOT_COLLECTED",
				Reply: job.Result, Source: "reconciled 409", At: now,
				Note: "the Engine already had this earlier job; kept as history, not graded"})
		default:
			found = append(found, AttemptRecord{JobID: id, SHA: sha, State: "UNKNOWN", Outcome: "NOT_COLLECTED", Source: "reconciled 409", At: now,
				Note: "the Engine already had this job ID but it isn't among its 20 most recent jobs, so its result can't be shown here; nothing was deleted"})
		}
		c.event("sync", fmt.Sprintf("Engine already held lesson job %s (earlier attempt); kept it as history and used the next job number.", id))
	}
	return "", lastSeq + max409Probes, false, found, fmt.Errorf("the Engine already holds %d consecutive job IDs for this lesson; stopped without submitting (see lesson history)", max409Probes)
}
