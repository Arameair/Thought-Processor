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
//   - Repair after Codex's audit (PR #121, comment 6832088431):
//       A  model-log text and metadata are used only when they hash to THIS run's recorded reply digest;
//       B  a verified historical receipt is reported as what it was, and "verified" now also needs the
//          current Thought to hold exactly one Note and one developed step for this nudge and request;
//       C  statements that come from the Engine's code (CODE-DEFINED, NOT PART OF THIS PATH, settings,
//          "thinking off") are made only when the revision the Engine reports now equals the one in the
//          run record AND is a revision whose code was inspected. Otherwise both are shown and those
//          stages say the revision was not reconciled.

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
	"c54390df43edddc161c5853c35db64462c39d37f", // + model-log binding repair (prompts unchanged)
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

// ProvenanceBlock is one box on the Observe map: a presentation grouping of recorded stages.
type ProvenanceBlock struct {
	ID      string   `json:"id"`
	Title   string   `json:"title"`
	Status  string   `json:"status"`
	Summary string   `json:"summary"`
	Stages  []string `json:"stages"`
}

// ProvenanceLink is one connection on the Observe map. Basis is the CONDITION that would join the two
// blocks, never a statement that it holds. Status: CONNECTED (the condition holds on recorded IDs or
// digests) | PARTIAL (part of it holds; Missing says what does not) | NOT ESTABLISHED (drawn as a gap).
type ProvenanceLink struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Status  string `json:"status"`
	Basis   string `json:"basis"`
	Missing string `json:"missing,omitempty"`
}

type DevelopmentRun struct {
	NudgeID        string `json:"nudge_id"`
	ThoughtID      string `json:"thought_id"`
	RequestID      string `json:"request_id"` // "" = NOT RECORDED
	EngineEntityID string `json:"engine_entity_id"`
	SourceRevision string `json:"source_revision"` // the Thought's revision when the draft was saved
	EngineCommit   string `json:"engine_commit"`
	RunState       string `json:"run_state"` // Core's view (RunView.State)
	// Verified: Core's receipt verified this run AND the current Thought still holds exactly one Note and
	// one developed step with this nudge and request ID. ReceiptVerified is the receipt alone.
	Verified        bool   `json:"verified"`
	ReceiptVerified bool   `json:"receipt_verified"`
	Result          string `json:"result"` // one line for the header
	// EngineRevision: RECONCILED | MISMATCH | NOT INSPECTED | NOT REPORTED (see engineRevision).
	EngineRevision     string            `json:"engine_revision"`
	RecordEngineCommit string            `json:"record_engine_commit"`
	Question           string            `json:"question"`
	Stages             []ProvenanceStage `json:"stages"`
	// Stage 3A UX repair: the Observe map. Blocks group stages for the owner; links say whether recorded
	// IDs or digests tie two blocks together. They are decided here, from the same correlations as the
	// stages, never by the page. A link is CONNECTED only when its Basis holds.
	Blocks   []ProvenanceBlock `json:"blocks"`
	Extra    []ProvenanceBlock `json:"extra"` // recorded facts outside the processing path (memory, tools, reasoning)
	Links    []ProvenanceLink  `json:"links"`
	ReadOnly string            `json:"read_only"`
	// Note is the text of the single current Note with this nudge and request ID; empty when there is not
	// exactly one, so Observe never shows one of several as "the" outcome.
	Note string `json:"note,omitempty"`
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

func hex40(s string) string {
	if len(s) != 40 {
		return ""
	}
	for _, r := range s {
		if !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f') {
			return ""
		}
	}
	return s
}

// engineRevision compares the revision the Engine reports now with the one its run record names.
func engineRevision(now, record string) string {
	now, record = hex40(now), hex40(record)
	switch {
	case now == "" || record == "":
		return "NOT REPORTED"
	case now != record:
		return "MISMATCH"
	}
	for _, c := range nudgeInstructionEngineCommits {
		if c == now {
			return "RECONCILED"
		}
	}
	return "NOT INSPECTED"
}

func plural(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return fmt.Sprintf("%d %s", n, many)
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
		out.RunState, out.ReceiptVerified = v.State, v.State == "COMPLETED" && v.Verified
	}
	var claim *engine.RecordClaim
	if claimed && rec.Claim.Readable && rec.Claim.Value != nil && !rec.Claim.Value.Void && rec.Claim.Value.NudgeID == nid &&
		rec.Claim.Value.EntityID == scope.engineID && rec.Claim.Value.ThoughtID == tid {
		claim = rec.Claim.Value
		out.RequestID = cognitionRef(claim.RequestID)
	}
	if claimed {
		out.RecordEngineCommit = orNR(hex40(rec.EngineCommit))
	} else {
		out.RecordEngineCommit = "NOT RECORDED"
	}
	out.EngineRevision = engineRevision(scope.engineCommit, rec.EngineCommit)
	codeOK := out.EngineRevision == "RECONCILED" // C: code-derived statements only for a reconciled revision
	revFields := []ProvenanceField{{"Engine revision reported now", out.EngineCommit}, {"Engine revision in this run's record", out.RecordEngineCommit},
		{"Reconciliation", out.EngineRevision}}
	revWhy := map[string]string{
		"MISMATCH":      "The Engine reports a different revision now than the one in this run's record.",
		"NOT INSPECTED": "The Engine's revision is not one whose code was inspected for this view.",
		"NOT REPORTED":  "The Engine did not report its revision (or the run record has none).",
	}[out.EngineRevision]
	// B: what the current Thought holds for this nudge and request ID.
	var curNotes []engine.ThoughtNote
	curSteps := 0
	if out.RequestID != "" {
		for _, nt := range raw.Thought.Notes {
			if nt.NudgeID == nid && nt.RequestID == out.RequestID {
				curNotes = append(curNotes, nt)
			}
		}
		for _, st := range raw.Thought.History {
			if st.Kind == "developed" && st.NudgeID == nid && st.RequestID == out.RequestID {
				curSteps++
			}
		}
	}
	out.Verified = out.ReceiptVerified && len(curNotes) == 1 && curSteps == 1
	if len(curNotes) == 1 {
		out.Note = curNotes[0].Text
	}
	switch {
	case out.Verified:
		out.Result = out.RunState + " · verified by Core"
	case out.ReceiptVerified:
		out.Result = out.RunState + " · receipt verified when written; the Thought now holds " + plural(len(curNotes), "Note", "Notes") +
			" and " + plural(curSteps, "developed step", "developed steps") + " for it"
	default:
		out.Result = orNR(out.RunState)
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
		case codeOK:
			s.Status, s.Summary = "PARTIAL", "This is the Thought's current question. The Engine sets a question once, when the Thought is formed, "+
				"but this run's logged prompt was not available to confirm it."
		default:
			s.Status, s.Summary = "PARTIAL", "This is the Thought's current question; this run's logged prompt was not available to confirm it was the one sent."
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
	if codeOK {
		add(ProvenanceStage{Stage: "instruction", Title: "The instruction", Status: "CODE-DEFINED",
			Summary:   "Not stored per run: fixed by the Thought Engine's code (thinkPrompt) at the revision below, which the Engine reports now and recorded for this run. The same lines appear at the end of the logged prompt above when it was recorded.",
			TextLabel: "Instruction", Text: nudgeInstructionSummary, Fields: revFields})
	} else {
		add(ProvenanceStage{Stage: "instruction", Title: "The instruction", Status: "PARTIAL", Fields: revFields,
			Summary: revWhy + " The instruction is not stored per run, so it is not described here; if the prompt above was recorded, its last lines are the instruction as logged."})
	}
	// A: the model-log record (no entity or Thought field of its own) is used, text or metadata, only when it
	// hashes to the reply digest recorded in THIS run's response record.
	replyDigest := ""
	if claim != nil && rec.Response.Readable && rec.Response.Value != nil {
		replyDigest = rec.Response.Value.ReplySHA256
	}
	modelBound := claim != nil && rec.ModelRequest != nil && len(replyDigest) == 64 && devSHA256(rec.ModelRequest.Text) == replyDigest
	// 6. The model call.
	{
		s := ProvenanceStage{Stage: "model_call", Title: "The model request"}
		s.Fields = []ProvenanceField{{"Request", orNR(out.RequestID)}}
		if codeOK {
			s.Fields = append(s.Fields, ProvenanceField{"Settings (CODE-DEFINED)", nudgeModelSettings})
		}
		switch {
		case claim == nil:
			s.Status, s.Summary = "NOT RECORDED", noRun
		case !rec.HTTP.Present && codeOK:
			s.Status, s.Summary = "NOT RECORDED", "The Engine has no record of an HTTP attempt for this run; by its design that record is written before any request, so none was sent."
		case !rec.HTTP.Present:
			s.Status, s.Summary = "NOT RECORDED", "The Engine has no record of an HTTP attempt for this run."
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
			if modelBound {
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
		switch {
		case claim == nil:
			s.Status, s.Summary = "NOT RECORDED", noRun
		case modelBound:
			s.Status, s.Summary = "RECORDED", "The full reply. Its sha256 matches the reply digest the Engine recorded for this run."
			s.TextLabel, s.Text = "Reply, exactly as returned", rec.ModelRequest.Text
		case replyDigest == "":
			s.Status, s.Summary = "NOT RECORDED", "This run recorded no reply digest (the request did not succeed, or its response record is missing), so no logged reply can be tied to it and none is shown."
			if r := rec.Response.Value; rec.Response.Readable && r != nil && r.Error != "" {
				s.Fields = []ProvenanceField{{"Error recorded for this run", clipRunes(r.Error, 300)}}
			}
		case rec.ModelRequest != nil:
			s.Status, s.Summary = "PARTIAL", "A log record has this request ID, but its text does not match the run's reply digest, so it is not shown."
		case rec.Call != nil && rec.Call.Reply != "":
			// calls.jsonl records are matched by the Engine on entity, Thought, request and purpose.
			s.Status = "PARTIAL"
			s.Summary = "The full reply record was not found (the model log keeps about 10 MB). This is the reply as this run's call-log record kept it"
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
		if !codeOK {
			s.Status, s.Summary = "PARTIAL", revWhy+" How the Engine reads a reply is not stored per run, so it is not described here."
		}
		suggested := "NOT RECORDED"
		if claimed && rec.Outcome.Value != nil && rec.Outcome.Value.Suggested != "" {
			suggested = rec.Outcome.Value.Suggested + " — not applied"
		}
		s.Fields = []ProvenanceField{{"Suggested state", suggested}}
		switch len(curNotes) {
		case 1:
			s.TextLabel, s.Text = "The saved Note (the kept NOTE line)", curNotes[0].Text
			s.Fields = append(s.Fields, ProvenanceField{"Note saved", devStamp(curNotes[0].At)})
		case 0:
			s.Fields = append(s.Fields, ProvenanceField{"Saved Note", "none with this nudge and request ID (never saved, or no longer stored)"})
		default:
			s.Fields = append(s.Fields, ProvenanceField{"Saved Note", fmt.Sprintf("%d Notes share this nudge and request ID, so none is shown as the one kept", len(curNotes))})
		}
		add(s)
	}
	// 9. Saved and verified. B: the historical receipt and the present Thought are reported separately.
	{
		s := ProvenanceStage{Stage: "saved", Title: "Saved and checked"}
		eState := "NOT RECORDED"
		if claimed {
			eState = orNR(rec.Run.State)
		}
		s.Fields = []ProvenanceField{{"Engine's state", eState}, {"Core's state", orNR(out.RunState)},
			{"Core's receipt verified", fmt.Sprint(out.ReceiptVerified)},
			{"Now in the Thought (Notes / developed steps)", fmt.Sprintf("%d / %d", len(curNotes), curSteps)}}
		now := plural(len(curNotes), "Note", "Notes") + " and " + plural(curSteps, "developed step", "developed steps")
		switch {
		case out.Verified:
			s.Status, s.Summary = "RECORDED", "Core's receipt verified one saved Note and one developed step with this request ID, and the Thought still holds exactly one of each. That shows it was saved, not that it answers the question."
		case out.ReceiptVerified:
			s.Status, s.Summary = "PARTIAL", "When Core wrote its receipt it verified one saved Note and one developed step for this request. "+
				"The Thought now holds "+now+" with this nudge and request ID, so none of them can be identified as that Note."
		case v == nil:
			s.Status, s.Summary = "NOT RECORDED", "Core has no receipt for this run."
		default:
			s.Status, s.Summary = "PARTIAL", "Core's receipt says "+orNR(out.RunState)+"; it is not a verified save. The Thought now holds "+now+" with this nudge and request ID."
		}
		add(s)
	}
	// 10–12. What this path does not do, and what does not exist. These come from the Engine's code (C).
	if codeOK {
		add(ProvenanceStage{Stage: "memory_retrieval", Title: "Memory search", Status: "NOT PART OF THIS PATH", Fields: []ProvenanceField{},
			Summary: "A nudge does not search memory. The Engine builds the prompt from the Thought's question, its evidence summaries and its newest 5 Notes."})
		add(ProvenanceStage{Stage: "tools", Title: "Tools, skills or notebooks", Status: "NOT PART OF THIS PATH", Fields: []ProvenanceField{},
			Summary: "No tool, skill or notebook runs in this path. The reply is only read for a NOTE and a STATE line; a Note that describes an action is not evidence the action happened."})
		add(ProvenanceStage{Stage: "model_reasoning", Title: "The model's reasoning", Status: "NOT RECORDED", Fields: []ProvenanceField{},
			Summary: "This request is sent with thinking off (CODE-DEFINED), so no reasoning text exists to show. Nothing is inferred in its place."})
	} else {
		add(ProvenanceStage{Stage: "memory_retrieval", Title: "Memory search", Status: "NOT RECORDED", Fields: []ProvenanceField{},
			Summary: "No record of a memory search exists for this run. " + revWhy + " Whether this revision ever searches memory is not described here."})
		add(ProvenanceStage{Stage: "tools", Title: "Tools, skills or notebooks", Status: "NOT RECORDED", Fields: []ProvenanceField{},
			Summary: "No record shows a tool, skill or notebook for this run. " + revWhy + " A Note that describes an action is not evidence the action happened."})
		add(ProvenanceStage{Stage: "model_reasoning", Title: "The model's reasoning", Status: "NOT RECORDED", Fields: []ProvenanceField{},
			Summary: "No reasoning text was recorded for this run. " + revWhy + " Nothing is inferred in its place."})
	}
	out.Blocks, out.Extra, out.Links = runMap(out, mapFacts{
		nid: nid, tid: tid, claim: claim, httpReq: httpRequestID(rec), httpPresent: claim != nil && rec.HTTP.Present,
		httpReadable: claim != nil && rec.HTTP.Readable, decision: sub.DecisionID, attentionOK: claim != nil && devSHA256(n.RequestedAction) == sub.AttentionSHA &&
			sub.AttentionSHA == claim.AttentionSHA, modelBound: modelBound, replyDigest: replyDigest, callReply: claim != nil && rec.Call != nil && rec.Call.Reply != "",
		replyText: modelText(rec, modelBound), notes: curNotes, steps: curSteps})
	return out, nil
}

func httpRequestID(rec engine.NudgeRecord) string {
	if rec.HTTP.Readable && rec.HTTP.Value != nil {
		return rec.HTTP.Value.RequestID
	}
	return ""
}

func modelText(rec engine.NudgeRecord, bound bool) string {
	if bound {
		return rec.ModelRequest.Text
	}
	return ""
}

// mapFacts: the correlations runMap may use. Everything here was already checked to build the stages.
type mapFacts struct {
	nid, tid                  string
	claim                     *engine.RecordClaim
	httpReq                   string
	httpPresent, httpReadable bool
	decision                  string
	attentionOK               bool
	modelBound                bool
	replyDigest               string
	callReply                 bool
	replyText                 string // only when modelBound
	notes                     []engine.ThoughtNote
	steps                     int
}

// runMap groups the stages into the Observe map and decides each connection from recorded evidence.
// Two RECORDED blocks are never joined unless an ID or digest ties them.
func runMap(out DevelopmentRun, f mapFacts) ([]ProvenanceBlock, []ProvenanceBlock, []ProvenanceLink) {
	st := map[string]ProvenanceStage{}
	for _, s := range out.Stages {
		st[s.Stage] = s
	}
	blk := func(id, title string, stages ...string) ProvenanceBlock {
		b := ProvenanceBlock{ID: id, Title: title, Status: st[stages[len(stages)-1]].Status, Summary: st[stages[len(stages)-1]].Summary, Stages: stages}
		if len(stages) == 1 {
			b.Status, b.Summary = st[stages[0]].Status, st[stages[0]].Summary
		}
		// A block that groups stages is never stronger than its weakest part: a recorded call whose prompt
		// was not recorded is PARTIAL, and the summary says which part is missing.
		if b.Status == "RECORDED" {
			for _, sid := range stages[:len(stages)-1] {
				if s := st[sid].Status; s != "RECORDED" && s != "CODE-DEFINED" {
					b.Status = "PARTIAL"
					b.Summary += " " + st[sid].Title + ": " + orNR(s) + "."
				}
			}
		}
		return b
	}
	req := orNR(out.RequestID)
	note := ProvenanceBlock{ID: "saved_note", Title: "Saved Note", Stages: []string{"parse"}}
	switch len(f.notes) {
	case 1:
		note.Status, note.Summary = "RECORDED", "The Thought holds one saved Note with nudge "+f.nid+" and request "+req+"."
	case 0:
		note.Status, note.Summary = "NOT RECORDED", "The Thought holds no Note with nudge "+f.nid+" and request "+req+" (never saved, or no longer stored)."
	default:
		note.Status, note.Summary = "PARTIAL", fmt.Sprintf("The Thought holds %d Notes with nudge %s and request %s, so none can be shown as the one this run saved.", len(f.notes), f.nid, req)
	}
	blocks := []ProvenanceBlock{
		blk("thought", "Thought", "thought"),
		blk("attention", "Your attention", "attention"),
		blk("authorization", "Authorization", "authorization"),
		blk("model_request", "Model request", "context_sent", "instruction", "model_call"),
		blk("model_reply", "Model reply", "model_reply"),
		note,
		blk("verification", "Verification", "saved"),
	}
	extra := []ProvenanceBlock{blk("memory_retrieval", "Memory search", "memory_retrieval"), blk("tools", "Tools and skills", "tools"),
		blk("model_reasoning", "Model reasoning", "model_reasoning")}

	link := func(from, to string) ProvenanceLink { return ProvenanceLink{From: from, To: to} }
	c := f.claim
	var links []ProvenanceLink
	// 1. Thought → attention: Core's draft and the Engine's claim name the same Thought.
	l := link("thought", "attention")
	l.Basis = "Core's draft for this nudge and the Engine's claim for the same nudge name the same Thought."
	if c != nil {
		l.Status = "CONNECTED"
	} else {
		l.Status, l.Missing = "PARTIAL", "Core's draft names this Thought, but the Engine has no usable claim for this nudge to confirm it."
	}
	links = append(links, l)
	// 2. Attention → authorization: the decision Core obtained for this nudge is the one the Engine redeemed,
	// and the attention digest is the same at Core, at the Engine and for the saved text.
	l = link("attention", "authorization")
	l.Basis = "The Identity decision Core obtained for this nudge is the one the Engine redeemed, and your saved text matches the attention digest both recorded."
	redeemed := c != nil && f.decision != "" && f.decision == c.Decision && c.Redeemed != ""
	switch {
	case redeemed && f.attentionOK:
		l.Status = "CONNECTED"
	case redeemed || f.attentionOK:
		l.Status = "PARTIAL"
		if !redeemed {
			l.Missing = "The decision Core obtained and the decision the Engine redeemed are not the same, or not both recorded."
		} else {
			l.Missing = "Your saved text does not match the attention digest recorded for this run."
		}
	case c == nil:
		l.Status, l.Missing = "PARTIAL", "Core obtained decision "+orNR(f.decision)+", but no Engine claim shows it was redeemed."
	default:
		l.Status, l.Missing = "NOT ESTABLISHED", "Neither the decision nor the attention digest matches between Core and the Engine."
	}
	links = append(links, l)
	// 3. Authorization → model request: the claim that redeemed the decision and the HTTP attempt name one request.
	l = link("authorization", "model_request")
	l.Basis = "The Engine's claim, which redeemed the decision, and its HTTP attempt record name the same request ID."
	switch {
	case c == nil:
		l.Status, l.Missing = "NOT ESTABLISHED", "The Engine has no usable claim for this nudge."
	case !f.httpPresent:
		l.Status, l.Missing = "NOT ESTABLISHED", "No HTTP attempt is recorded for this run."
	case !f.httpReadable:
		l.Status, l.Missing = "PARTIAL", "An HTTP attempt record exists but cannot be read."
	case c.RequestID != "" && f.httpReq == c.RequestID && c.Redeemed != "":
		l.Status = "CONNECTED"
	default:
		l.Status, l.Missing = "NOT ESTABLISHED", "The claim and the HTTP attempt do not name the same request, or the decision was not redeemed."
	}
	links = append(links, l)
	// 4. Model request → model reply: the reply's bytes hash to the digest recorded for this run.
	l = link("model_request", "model_reply")
	l.Basis = "The reply's sha256 equals the reply digest the Engine recorded for this run."
	switch {
	case f.modelBound:
		l.Status = "CONNECTED"
	case f.replyDigest != "" && f.callReply:
		l.Status, l.Missing = "PARTIAL", "Only this run's call-log copy of the reply is available; it is not checked against the digest."
	case f.replyDigest == "":
		l.Status, l.Missing = "NOT ESTABLISHED", "This run recorded no reply digest, so no reply can be tied to it."
	default:
		l.Status, l.Missing = "NOT ESTABLISHED", "No logged reply matches the digest recorded for this run."
	}
	links = append(links, l)
	// 5. Model reply → saved Note: exactly one Note carries this nudge and request, and its text is in the verified reply.
	l = link("model_reply", "saved_note")
	l.Basis = "A single saved Note carries this nudge and request ID, and its text appears in the digest-checked reply."
	switch {
	case out.RequestID == "":
		l.Status, l.Missing = "NOT ESTABLISHED", "No request ID is recorded for this run, so no saved Note can be tied to it."
	case len(f.notes) != 1:
		l.Status, l.Missing = "NOT ESTABLISHED", fmt.Sprintf("The Thought holds %d Notes with this nudge and request, so no single Note can be tied to this reply.", len(f.notes))
	case f.modelBound && strings.TrimSpace(f.notes[0].Text) != "" && strings.Contains(f.replyText, strings.TrimSpace(f.notes[0].Text)):
		l.Status = "CONNECTED"
	default:
		l.Status, l.Missing = "PARTIAL", "The saved Note carries this nudge and request ID, but its text could not be found in a verified reply."
	}
	links = append(links, l)
	// 6. Saved Note → verification: Core's receipt verified the request and the Thought is still unique for it.
	l = link("saved_note", "verification")
	l.Basis = "Core's receipt verified this request, and the Thought now holds a single Note and a single developed step for it."
	switch {
	case out.Verified:
		l.Status = "CONNECTED"
	case out.ReceiptVerified:
		l.Status, l.Missing = "PARTIAL", "Core's receipt verified this request when it was written, but the Thought now holds "+
			plural(len(f.notes), "Note", "Notes")+" and "+plural(f.steps, "developed step", "developed steps")+" for it."
	default:
		l.Status, l.Missing = "NOT ESTABLISHED", "Core's receipt does not verify this save ("+orNR(out.RunState)+")."
	}
	links = append(links, l)
	return blocks, extra, links
}
