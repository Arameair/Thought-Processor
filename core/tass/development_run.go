package tass

// Stage 3A (#120; design PR #107 §3.2): "How this Note was produced" for ONE supervised nudge.
//
//   - Read only. One Thought read and one run-record read from the Engine, Core's own receipts and nudge
//     journal. No model call, no Identity decision beyond cognitionAccess, no write, no Thought change.
//   - Scoped like NudgeHistory: cognitionAccess (Identity UNLOCKED, conversation + thought.process, the
//     reviewed Identity → Engine link) and the nudge must be in THIS owner's journal for THIS linked
//     Engine entity. Nothing from the request chooses an entity or a Thought; both come from the journal.
//   - Honest. Each stage carries one status:
//       RECORDED             read from a record of this run, and consistent with the other records
//       PARTIAL              recorded, but clipped, incomplete or not fully confirmed (the reason says which)
//       CODE-DEFINED         not stored per run; fixed by the Engine code at the pinned commit
//       NOT RECORDED         no record of it exists (or none could be found, which the reason says)
//       NOT PART OF THIS PATH  this kind of processing does not happen for a nudge
//     The model's private reasoning is never claimed: nudges run with thinking off, so none exists.

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Arameair/Thought-Processor/core/engine"
)

// ErrDevelopmentRunNotFound: not a processed nudge of this owner and linked Engine entity. One error for
// "not yours" and "never processed", so the answer does not reveal other owners' nudges.
var ErrDevelopmentRunNotFound = errors.New("no processed nudge with this ID for this owner")

// nudgeInstructionEngineCommits: the Engine commits whose thinkPrompt / parseThink / askNudge this summary
// describes. TestNudgeInstructionSummaryMatchesThePinnedEngine fails if tass.sh pins another Engine.
var nudgeInstructionEngineCommits = []string{
	"40d4e91e3b04196fa4e4f55a92b25789b82d8675", // installed Stage 2 Engine
	"b594357c021db3a5164f321ec2b66b21198a8455", // + the read-only run record (prompts unchanged)
}

const (
	nudgeInstructionSummary = "Add ONE new idea, possible explanation, or thing to watch for next. Do not repeat the evidence or earlier notes. " +
		"Reply with exactly two lines: NOTE (at most 40 words) and STATE (keep, sleep or resolve)."
	nudgeModelSettings = "one request; at most 256 tokens; thinking off; temperature 0"
	nudgeParseRule     = "The Engine keeps the first NOTE line (at most 400 characters) and the STATE word; the rest of the reply is not stored in the Thought. " +
		"A NOTE that is too short or only repeats the inputs is refused. The STATE word is only a suggestion: no code path applies it."
	devRunTextMax = 6000 // runes of any one long text (prompt, reply, Note) sent to the page
)

type ProvenanceField struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type ProvenanceStage struct {
	Stage     string            `json:"stage"`
	Title     string            `json:"title"`
	Status    string            `json:"status"`
	Summary   string            `json:"summary"`
	Fields    []ProvenanceField `json:"fields"`
	TextLabel string            `json:"text_label,omitempty"`
	Text      string            `json:"text,omitempty"` // shown literally; private: this single-run view only
}

type DevelopmentRun struct {
	NudgeID        string            `json:"nudge_id"`
	ThoughtID      string            `json:"thought_id"`
	RequestID      string            `json:"request_id"` // "" = NOT RECORDED
	EngineEntityID string            `json:"engine_entity_id"`
	SourceRevision string            `json:"source_revision"` // the Thought's revision when the draft was saved
	EngineCommit   string            `json:"engine_commit"`
	RunState       string            `json:"run_state"` // Core's view (RunView.State)
	Verified       bool              `json:"verified"`
	Question       string            `json:"question"`
	Stages         []ProvenanceStage `json:"stages"`
	ReadOnly       string            `json:"read_only"`
}

func devSHA256(s string) string { h := sha256.Sum256([]byte(s)); return hex.EncodeToString(h[:]) }

// engineQuote mirrors Thought Engine prompts.go quote(): how the question appears in the prompt.
func engineQuote(s string) string {
	s = strings.ReplaceAll(strings.TrimSpace(s), "\n", " ")
	return `"` + strings.ReplaceAll(s, `"`, "'") + `"`
}

func devStamp(t time.Time) string {
	if t.IsZero() {
		return "NOT RECORDED"
	}
	return t.UTC().Format(time.RFC3339)
}

func orNR(s string) string {
	if s == "" {
		return "NOT RECORDED"
	}
	return s
}

// DevelopmentRun builds the read-only provenance of one processed nudge.
func (c *Core) DevelopmentRun(nid string, rq ...*CognitionReq) (DevelopmentRun, error) {
	if !nudgeIDOK(nid) {
		return DevelopmentRun{}, ErrDevelopmentRunNotFound
	}
	scope, err := c.cognitionAccess(rq...)
	if err != nil {
		return DevelopmentRun{}, err
	}
	c.nudgeMu.Lock()
	all, _, jerr := c.readNudges()
	c.nudgeMu.Unlock()
	if jerr != nil {
		return DevelopmentRun{}, ErrCognitionUnavailable
	}
	var n *Nudge
	for i := range all {
		if all[i].NudgeID == nid && all[i].OwnerEntityID == scope.identityID && all[i].SourceEngineEntityID == scope.engineID {
			n = &all[i]
		}
	}
	var sub submittedRec
	sex, sok, _ := coreReadOnce(c.runPath(nid, "submitted.json"), &sub)
	// Only a nudge Core submitted for processing has a run; its record must name this owner and entity.
	if n == nil || !sex || !sok || sub.NudgeID != nid || sub.OwnerEntityID != scope.identityID || sub.EngineEntityID != scope.engineID ||
		sub.ThoughtID != n.SourceThoughtID {
		return DevelopmentRun{}, ErrDevelopmentRunNotFound
	}
	tid := n.SourceThoughtID
	raw, err := c.engine.CognitionThought(scope.engineID, tid)
	if err != nil || raw.Thought.ThoughtID != tid || raw.Thought.EntityID != scope.engineID {
		return DevelopmentRun{}, ErrCognitionUnavailable
	}
	rec, code, rerr := c.engine.NudgeRecord(scope.engineID, nid)
	claimed := rerr == nil
	if rerr != nil && code != 404 {
		return DevelopmentRun{}, ErrCognitionUnavailable // unreachable or an Engine without the record read
	}
	if errors.Is(rerr, engine.ErrNoRecordRoute) {
		return DevelopmentRun{}, ErrCognitionUnavailable
	}
	if claimed && (rec.Run.NudgeID != nid || rec.Run.EntityID != scope.engineID) {
		return DevelopmentRun{}, ErrCognitionUnavailable // the Engine answered for another run: show nothing
	}
	v := c.runView(nid)
	out := DevelopmentRun{NudgeID: nid, ThoughtID: tid, EngineEntityID: scope.engineID, SourceRevision: orNR(n.SourceRevision),
		EngineCommit: orNR(scope.engineCommit), Question: clipRunes(raw.Thought.Question, 900),
		ReadOnly: "Read only: this view called no model, made no Identity decision beyond reading, and changed no Thought, Note or file."}
	if v != nil {
		out.RunState, out.Verified = v.State, v.State == "COMPLETED" && v.Verified
	}
	var claim *engine.RecordClaim
	if claimed && rec.Claim.Readable && rec.Claim.Value != nil && !rec.Claim.Value.Void && rec.Claim.Value.NudgeID == nid &&
		rec.Claim.Value.EntityID == scope.engineID && rec.Claim.Value.ThoughtID == tid {
		claim = rec.Claim.Value
		out.RequestID = cognitionRef(claim.RequestID)
	}
	noRun := "The Engine has no claim for this nudge, so there is no run record to read."
	if claimed && claim == nil {
		noRun = "The Engine's claim for this nudge is void, unreadable or names another run, so its records are not used."
	}
	add := func(s ProvenanceStage) {
		if s.Fields == nil {
			s.Fields = []ProvenanceField{}
		}
		s.Text = clipRunes(s.Text, devRunTextMax)
		out.Stages = append(out.Stages, s)
	}

	// 1. The Thought.
	{
		s := ProvenanceStage{Stage: "thought", Title: "The Thought", TextLabel: "Question (as the Thought holds it now)", Text: out.Question}
		s.Fields = []ProvenanceField{{"Thought", tid}, {"Revision when the draft was saved", out.SourceRevision},
			{"Revision when the Engine claimed it", "NOT RECORDED"}, {"Revision just before the model call", "NOT RECORDED"}}
		if claim != nil {
			s.Fields[2].Value = orNR(claim.Revision)
		}
		if claimed && rec.Outcome.Value != nil {
			s.Fields[3].Value = orNR(rec.Outcome.Value.RevP1)
		}
		switch {
		case rec.Call != nil && claim != nil && strings.Contains(rec.Call.Prompt, "Thought: "+engineQuote(raw.Thought.Question)):
			s.Status, s.Summary = "RECORDED", "This question appears word for word in the prompt the Engine logged for this run."
		default:
			s.Status, s.Summary = "PARTIAL", "This is the Thought's current question. The Engine sets a question once, when the Thought is formed, "+
				"but this run's logged prompt was not available to confirm it."
		}
		add(s)
	}
	// 2. The owner's attention.
	{
		s := ProvenanceStage{Stage: "attention", Title: "Your attention text", TextLabel: "Your words (from your saved draft)", Text: n.RequestedAction}
		got := devSHA256(n.RequestedAction)
		engineSHA := "NOT RECORDED"
		if claim != nil {
			engineSHA = orNR(claim.AttentionSHA)
		}
		s.Fields = []ProvenanceField{{"Draft", nid}, {"Saved", devStamp(n.CreatedAt)}, {"Digest Core submitted", orNR(sub.AttentionSHA)}, {"Digest the Engine recorded", engineSHA}}
		switch {
		case claim != nil && got == sub.AttentionSHA && got == claim.AttentionSHA:
			s.Status, s.Summary = "RECORDED", "Your saved text matches the digest Core submitted and the digest the Engine recorded for this run."
		case claim == nil:
			s.Status, s.Summary = "PARTIAL", "This is your saved draft. "+noRun
		default:
			s.Status, s.Summary = "PARTIAL", "Your saved text does NOT match the digest recorded for this run, so it is not shown as what the model received."
			s.Text, s.TextLabel = "", ""
		}
		add(s)
	}
	// 3. Authorization.
	{
		s := ProvenanceStage{Stage: "authorization", Title: "Authorization"}
		redeemed, redeem := "NOT RECORDED", "NOT RECORDED"
		if claim != nil {
			redeemed, redeem = orNR(claim.Decision), orNR(claim.Redeemed)
		}
		s.Fields = []ProvenanceField{{"Draft saved under Identity decision", orNR(n.DraftDecisionID)},
			{"Processing decision Core obtained", orNR(sub.DecisionID)}, {"Decision the Engine redeemed", redeemed}, {"Identity's redeem record", redeem}}
		switch {
		case claim != nil && sub.DecisionID != "" && sub.DecisionID == claim.Decision && claim.Redeemed != "":
			s.Status, s.Summary = "RECORDED", "Identity issued one single-use processing decision for this exact nudge; the Engine redeemed that same decision before running it."
		case claim == nil:
			s.Status, s.Summary = "PARTIAL", "Core recorded the processing decision. "+noRun
		default:
			s.Status, s.Summary = "PARTIAL", "The decision Core obtained and the decision the Engine redeemed are not the same or not both recorded."
		}
		add(s)
	}
	// 4. The prompt sent.
	{
		s := ProvenanceStage{Stage: "context_sent", Title: "What was sent to the model"}
		switch {
		case claim == nil:
			s.Status, s.Summary = "NOT RECORDED", noRun
		case rec.Call == nil:
			s.Status, s.Summary = "NOT RECORDED", fmt.Sprintf("No log record for request %s was found in the newest %d MiB of the Engine's call log; it may be older than that window.",
				out.RequestID, rec.CallScan.Window>>20)
		default:
			s.TextLabel, s.Text = "Prompt, as the Engine logged it", rec.Call.Prompt
			s.Fields = []ProvenanceField{{"Logged", devStamp(rec.Call.At)}, {"Request", out.RequestID}}
			if rec.Call.PromptClipped {
				s.Status, s.Summary = "PARTIAL", "The Engine logs at most 2,000 bytes of a prompt; the end of this one was cut off in the log. Your attention text appears only as its sha256 (the Engine never logs it)."
			} else {
				s.Status, s.Summary = "RECORDED", "The prompt as the Engine logged it for this run. Your attention text appears only as its sha256 (the Engine never logs it); the model received your words in its place."
			}
		}
		add(s)
	}
	// 5. The instruction.
	add(ProvenanceStage{Stage: "instruction", Title: "The instruction", Status: "CODE-DEFINED",
		Summary:   "Not stored per run: fixed by the Thought Engine's code (thinkPrompt). The same lines appear at the end of the logged prompt above when it was recorded.",
		TextLabel: "Instruction", Text: nudgeInstructionSummary,
		Fields: []ProvenanceField{{"Defined by Engine commit", out.EngineCommit}}})
	// 6. The model call.
	{
		s := ProvenanceStage{Stage: "model_call", Title: "The model request"}
		s.Fields = []ProvenanceField{{"Request", orNR(out.RequestID)}, {"Settings (CODE-DEFINED)", nudgeModelSettings}}
		switch {
		case claim == nil:
			s.Status, s.Summary = "NOT RECORDED", noRun
		case !rec.HTTP.Present:
			s.Status, s.Summary = "NOT RECORDED", "The Engine has no record of an HTTP attempt for this run; by its design that record is written before any request, so none was sent."
		case !rec.HTTP.Readable:
			s.Status, s.Summary = "PARTIAL", "The Engine's HTTP attempt record for this run is unreadable."
		case !rec.Response.Readable || rec.Response.Value == nil:
			s.Fields = append(s.Fields, ProvenanceField{"Sent", devStamp(rec.HTTP.Value.At)})
			s.Status, s.Summary = "PARTIAL", "The request was attempted, but no response was recorded. The model may have received it."
		default:
			r := rec.Response.Value
			s.Fields = append(s.Fields, ProvenanceField{"Sent", devStamp(rec.HTTP.Value.At)}, ProvenanceField{"Answered", fmt.Sprint(r.HTTPAnswer)},
				ProvenanceField{"Reply usable", fmt.Sprint(r.OK)}, ProvenanceField{"Time", fmt.Sprintf("%d ms", r.ElapsedMS)},
				ProvenanceField{"Reply sha256", orNR(r.ReplySHA256)})
			if rec.ModelRequest != nil {
				s.Fields = append(s.Fields, ProvenanceField{"Model", orNR(rec.ModelRequest.Model)}, ProvenanceField{"Identity check at the model", orNR(rec.ModelRequest.Authorized)})
			}
			if r.Error != "" {
				s.Fields = append(s.Fields, ProvenanceField{"Error", clipRunes(r.Error, 300)})
			}
			s.Status, s.Summary = "RECORDED", "One request to the local model, recorded by the Engine before it was sent and after it was answered."
		}
		add(s)
	}
	// 7. The raw reply.
	{
		s := ProvenanceStage{Stage: "model_reply", Title: "The model's reply"}
		var want string
		if claim != nil && rec.Response.Value != nil {
			want = rec.Response.Value.ReplySHA256
		}
		switch {
		case claim == nil:
			s.Status, s.Summary = "NOT RECORDED", noRun
		case rec.ModelRequest != nil && want != "" && devSHA256(rec.ModelRequest.Text) == want:
			s.Status, s.Summary = "RECORDED", "The full reply. Its sha256 matches the reply digest the Engine recorded for this run."
			s.TextLabel, s.Text = "Reply, exactly as returned", rec.ModelRequest.Text
		case rec.ModelRequest != nil && want == "":
			s.Status, s.Summary = "PARTIAL", "The Engine logged this reply, but the run recorded no reply digest to confirm it (the request did not succeed)."
			s.TextLabel, s.Text = "Reply, as logged", rec.ModelRequest.Text
			if rec.ModelRequest.Error != "" {
				s.Fields = []ProvenanceField{{"Error", clipRunes(rec.ModelRequest.Error, 300)}}
			}
		case rec.ModelRequest != nil:
			s.Status, s.Summary = "PARTIAL", "A log record has this request ID, but its text does not match the run's reply digest, so it is not shown."
		case rec.Call != nil && rec.Call.Reply != "":
			s.Status = "PARTIAL"
			s.Summary = "The full reply record was not found (the model log keeps about 10 MB). This is the reply as the call log kept it"
			if rec.Call.ReplyClipped {
				s.Summary += ", cut off at 1,000 bytes."
			} else {
				s.Summary += "."
			}
			s.TextLabel, s.Text = "Reply, as the call log kept it", rec.Call.Reply
		default:
			s.Status, s.Summary = "NOT RECORDED", "No reply was found for this request in the Engine's logs."
		}
		add(s)
	}
	// 8. Parsing.
	{
		s := ProvenanceStage{Stage: "parse", Title: "What the Engine kept from the reply", Status: "CODE-DEFINED", Summary: nudgeParseRule}
		var notes []engine.ThoughtNote
		for _, nt := range raw.Thought.Notes {
			if nt.NudgeID == nid && out.RequestID != "" && nt.RequestID == out.RequestID {
				notes = append(notes, nt)
			}
		}
		suggested := "NOT RECORDED"
		if claimed && rec.Outcome.Value != nil && rec.Outcome.Value.Suggested != "" {
			suggested = rec.Outcome.Value.Suggested + " — not applied"
		}
		s.Fields = []ProvenanceField{{"Suggested state", suggested}}
		switch len(notes) {
		case 1:
			s.TextLabel, s.Text = "The saved Note (the kept NOTE line)", notes[0].Text
			s.Fields = append(s.Fields, ProvenanceField{"Note saved", devStamp(notes[0].At)})
		case 0:
			s.Fields = append(s.Fields, ProvenanceField{"Saved Note", "none with this nudge and request ID (never saved, or no longer stored: the Engine keeps the newest 200)"})
		default:
			s.Fields = append(s.Fields, ProvenanceField{"Saved Note", fmt.Sprintf("%d Notes share this nudge and request ID, so none is shown as the one kept", len(notes))})
		}
		add(s)
	}
	// 9. Saved and verified.
	{
		s := ProvenanceStage{Stage: "saved", Title: "Saved and checked"}
		eState := "NOT RECORDED"
		if claimed {
			eState = orNR(rec.Run.State)
		}
		s.Fields = []ProvenanceField{{"Engine's state", eState}, {"Core's state", orNR(out.RunState)}, {"Core verified", fmt.Sprint(out.Verified)}}
		if claimed {
			s.Fields = append(s.Fields, ProvenanceField{"Matching Notes / steps", fmt.Sprintf("%d / %d", rec.Run.Evidence.Notes, rec.Run.Evidence.Steps)})
		}
		switch {
		case out.Verified:
			s.Status, s.Summary = "RECORDED", "Core's receipt verified exactly one saved Note and one developed step with this request ID. That shows it was saved, not that it answers the question."
		case v == nil:
			s.Status, s.Summary = "NOT RECORDED", "Core has no receipt for this run."
		default:
			s.Status, s.Summary = "PARTIAL", "Core's receipt says "+orNR(out.RunState)+"; it is not a verified save."
		}
		add(s)
	}
	// 10–12. What this path does not do, and what does not exist.
	add(ProvenanceStage{Stage: "memory_retrieval", Title: "Memory search", Status: "NOT PART OF THIS PATH", Fields: []ProvenanceField{},
		Summary: "A nudge does not search memory. The Engine builds the prompt from the Thought's question, its evidence summaries and its newest 5 Notes."})
	add(ProvenanceStage{Stage: "tools", Title: "Tools, skills or notebooks", Status: "NOT PART OF THIS PATH", Fields: []ProvenanceField{},
		Summary: "No tool, skill or notebook runs in this path. The reply is only read for a NOTE and a STATE line; a Note that describes an action is not evidence the action happened."})
	add(ProvenanceStage{Stage: "model_reasoning", Title: "The model's reasoning", Status: "NOT RECORDED", Fields: []ProvenanceField{},
		Summary: "This request is sent with thinking off (CODE-DEFINED), so no reasoning text exists to show. Nothing is inferred in its place."})
	return out, nil
}
