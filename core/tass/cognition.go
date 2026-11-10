package tass

// Cognition presents evidence *already recorded* by the Go Thought Engine.
// It does not run a cycle, approve a lesson, or create a skill request.

import (
    "bytes"
    "encoding/json"
    "errors"
    "fmt"
    "sort"
    "strings"
    "time"

    "github.com/Arameair/Thought-Processor/core/engine"
)

var ErrCognitionDenied=errors.New("Cognition read denied by Identity")
var ErrCognitionUnavailable=errors.New("Cognition evidence unavailable")

const cognitionCardMax=100
const cognitionTextMax=360

type CognitionField struct {
    Name string `json:"name"`
    Value string `json:"value"`
}
type CognitionCard struct {
    ID string `json:"id"`
    Kind string `json:"kind"`
    Title string `json:"title"`
    Summary string `json:"summary"`
    State string `json:"status"`
    Source string `json:"source"`
    SourceID string `json:"source_id"`
    RecordedAt time.Time `json:"recorded_at,omitempty"`
    Verified string `json:"verification"` // VERIFIED-LINK | OBSERVED | APP-REPORTED | UNVERIFIED | NOT RECORDED
    ThoughtID string `json:"thought_id,omitempty"`
    RelatedCycleIDs []string `json:"related_cycle_ids,omitempty"`
    Fields []CognitionField `json:"fields"`
}
type CognitionLibrary struct{
    IdentityEntityID string `json:"identity_entity_id"`
    EngineEntityID string `json:"engine_entity_id,omitempty"`
    IdentityName string `json:"identity_name"`
    EngineCommit string `json:"engine_commit"`
    ObservedAt time.Time `json:"observed_at"`
    BrainRevision int `json:"brain_revision"`
    Thoughts []CognitionCard `json:"thoughts"`
    Cycles []CognitionCard `json:"cycles"`
    Outcomes []CognitionCard `json:"outcomes"`
    Library []CognitionCard `json:"library"`
    Note string `json:"note"`
    Limited bool `json:"limited"`
    Link CognitionLinkState `json:"link"`
    Junctions []CognitionCard `json:"junctions"`
}
type CognitionThought struct {
    ThoughtID string `json:"thought_id"`
    Topic string `json:"topic"`
    Question string `json:"question"`
    State string `json:"state"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
    History []engine.ThoughtStep `json:"history"`
    Traces []CognitionTrace `json:"traces"`
    RelatedCycleIDs []string `json:"related_cycle_ids"`
    Source string `json:"source"`
    Verification string `json:"verification"`
    Truncated bool `json:"truncated"`
    // P0 (#106): history keeps the NEWEST cognitionHistoryMax steps; HistoryTotal is the Engine's count.
    HistoryTotal int `json:"history_total"`
    // P0 (#106, D1): what TASS wrote, newest first. Single-Thought detail only.
    Developments []CognitionDevelopment `json:"developments"`
    DevelopmentsTotal int `json:"developments_total"`
    DevelopmentsTruncated bool `json:"developments_truncated"`
}
type CognitionTrace struct {
    TraceID string `json:"trace_id"`
    SourceEventID string `json:"source_event_id"`
    At time.Time `json:"recorded_at"`
    Status string `json:"status"`
    Verified bool `json:"verified"`
}
type CognitionTimeline struct {
    CycleID string `json:"cycle_id"`
    ThoughtID string `json:"thought_id,omitempty"`
    LinkStatus string `json:"link_status"`
    Source string `json:"source"`
    EngineCommit string `json:"engine_commit"`
    IdentityEvidence string `json:"identity_evidence"`
    ObservedAt time.Time `json:"observed_at"`
    State string `json:"state"`
    Steps []engine.RecordedStep `json:"steps"`
    Missing []string `json:"missing"`
    Pending []string `json:"pending"` // stages a cycle still in progress has not reached yet (not missing)
    Truncated bool `json:"truncated"`
    Partial bool `json:"partial"`
    InvalidLines int `json:"invalid_lines"`
    Note string `json:"note"`
}
func cognitionFirst(values ...string) string {
    for _,v:=range values {if v!="" {return v}}
    return ""
}
func cognitionClip(s string,n int)string {
    s=strings.TrimSpace(s)
    if len(s)>n {return s[:n]+"…"}
    return s
}
func cognitionField(name,value string) CognitionField {
    if value=="" {value="NOT RECORDED"}
    return CognitionField{Name:name,Value:value}
}
type cognitionScope struct{
    identityID,name,engineID,engineCommit string
    link CognitionLinkState // link.Link (when set) is the owner link record the scope's discriminator names
}
// cognitionAccess authorizes one Cognition request. rq (optional, #106 lifecycle contract) narrows it
// to the scope the page expects and receives the scope it was authorized under.
func (c *Core) cognitionAccess(rq ...*CognitionReq)(cognitionScope,error){
    var out cognitionScope
    r:=cognitionReqOf(rq)
    if c.cfg.IdentitySvc==nil {return out,ErrCognitionDenied}
    state:=c.cfg.IdentitySvc.State()
    if state.State!="UNLOCKED"||state.Entity==nil||state.Entity.EntityID==""||state.Entity.Name=="" {
        return out,ErrCognitionDenied
    }
    // A different entity is refused before any Identity decision is asked for or any record is read.
    if r!=nil&&r.Want!=nil&&r.Want.Identity!=state.Entity.EntityID {return out,ErrCognitionScopeMismatch}
    for _,action:=range []string{"conversation","thought.process"}{
        d:=c.cfg.IdentitySvc.Authorize(action,"thought-processor:cognition.read")
        if !d.Allow||d.DecisionID==""||d.EntityID!=state.Entity.EntityID {
            return out,ErrCognitionDenied
        }
    }
    out.identityID,out.name=state.Entity.EntityID,state.Entity.Name
    status,err:=c.engine.CognitionStatus()
    if err!=nil {return out,ErrCognitionUnavailable}
    out.engineCommit=status.Config.Commit
    // Validate version provenance; never export an arbitrary server-controlled string.
    if len(out.engineCommit)!=40 {
        out.engineCommit="NOT RECORDED"
    } else {
        for _,r:=range out.engineCommit {
            if !(r>='0'&&r<='9'||r>='a'&&r<='f') {out.engineCommit="NOT RECORDED";break}
        }
    }
    // #82 P0-A: only an owner-reviewed link (canonical Identity entity → Engine entity) scopes
    // Engine data. A matching display name is never used as evidence.
    out.engineID,out.link=c.resolveEngineEntity(out.identityID,out.name,status)
    got:=out.attested()
    if !ValidCognitionScope(got){return out,ErrCognitionUnavailable}
    if err:=r.narrow(got);err!=nil{return out,err}
    return out,nil
}
// cognitionOwnedCycle: a cycle belongs to this identity ONLY when its recorded Engine entity is
// exactly the linked one. Cycles without an Engine entity are UNSCOPED and never shown by name.
func cognitionOwnedCycle(cy engine.CognitionCycle,scope cognitionScope)bool {
    return scope.engineID!="" && cy.EntityID!="" && cy.EntityID==scope.engineID
}
func (c *Core) CognitionLibrary(rq ...*CognitionReq)(CognitionLibrary,error){
    scope,err:=c.cognitionAccess(rq...)
    if err!=nil{return CognitionLibrary{},err}
    out:=CognitionLibrary{IdentityEntityID:scope.identityID,EngineEntityID:scope.engineID,
        IdentityName:scope.name,EngineCommit:scope.engineCommit,ObservedAt:c.now().UTC(),
        Thoughts:[]CognitionCard{},Cycles:[]CognitionCard{},Outcomes:[]CognitionCard{},Library:[]CognitionCard{},
        Junctions:[]CognitionCard{},Link:scope.link}
    g:=c.Brain() // one consistent saved snapshot for revision AND notes (#82 P0-B)
    out.BrainRevision=g.Revision
    entity:=engine.CognitionEntity{}
    if scope.engineID!="" {
        entity,err=c.engine.CognitionEntity(scope.engineID)
        if err!=nil||entity.Card.EntityID!=scope.engineID{
            return CognitionLibrary{},ErrCognitionUnavailable
        }
    } else {
        out.Note="Engine history NOT LINKED to this Identity entity: persistent Thoughts and cycles are not shown."
    }
    cycles,err:=c.engine.CognitionCycles()
    if err!=nil{return CognitionLibrary{},ErrCognitionUnavailable}
    validCycles:=[]engine.CognitionCycle{}
    for _,cy:=range cycles {
        switch {
        case cognitionOwnedCycle(cy,scope):validCycles=append(validCycles,cy)
        case cy.EntityID=="":out.Link.UnscopedCycles++
        default:out.Link.OtherCycles++
        }
    }
    sort.SliceStable(validCycles,func(i,j int)bool{
        if validCycles[i].ReceivedAt.Equal(validCycles[j].ReceivedAt){return validCycles[i].CycleID<validCycles[j].CycleID}
        return validCycles[i].ReceivedAt.After(validCycles[j].ReceivedAt)
    })
    thoughtSet:=map[string]engine.Thought{}
    for _,th:=range entity.Thoughts {if th.EntityID==scope.engineID&&engine.CognitionID.MatchString(th.ThoughtID){thoughtSet[th.ThoughtID]=th}}
    thoughts:=make([]engine.Thought,0,len(thoughtSet))
    for _,v:=range thoughtSet {thoughts=append(thoughts,v)}
    sort.Slice(thoughts,func(i,j int)bool{
        if thoughts[i].UpdatedAt.Equal(thoughts[j].UpdatedAt){return thoughts[i].ThoughtID<thoughts[j].ThoughtID}
        return thoughts[i].UpdatedAt.After(thoughts[j].UpdatedAt)
    })
    if len(thoughts)>cognitionCardMax {thoughts=thoughts[:cognitionCardMax];out.Limited=true}
    for _,th:=range thoughts{
        related:=[]string{}
        for _,cy:=range validCycles {
            if cy.EntityID==scope.engineID&&cy.Spark.ThoughtID==th.ThoughtID {
                related=append(related,cy.CycleID)
            }
        }
        out.Thoughts=append(out.Thoughts,CognitionCard{
            ID:"thought:"+th.ThoughtID,Kind:"thought",Title:cognitionClip(cognitionFirst(th.Topic,th.Question,th.ThoughtID),100),
            Summary:cognitionClip(th.Question,cognitionTextMax),State:th.State,
            Source:"Thought Engine: persistent Thought",SourceID:th.ThoughtID,
            RecordedAt:th.UpdatedAt,Verified:"OBSERVED",ThoughtID:th.ThoughtID,
            RelatedCycleIDs:related,Fields:[]CognitionField{
                cognitionField("Canonical Thought ID",th.ThoughtID),
                cognitionField("Engine entity ID",th.EntityID),
                cognitionField("State",th.State),
                cognitionField("Related cycles",strings.Join(related,", ")),
            },
        })
    }
    for _,cy:=range validCycles {
        if !engine.CognitionID.MatchString(cy.CycleID){continue}
        thoughtID:=""
        linkStatus:="UNLINKED"
        if scope.engineID!=""&&cy.EntityID==scope.engineID {
            if th,found:=thoughtSet[cy.Spark.ThoughtID];found&&th.EntityID==scope.engineID {
                thoughtID=cy.Spark.ThoughtID
                linkStatus="VERIFIED-LINK"
            }
        }
        out.Cycles=append(out.Cycles,CognitionCard{
            ID:"cycle:"+cy.CycleID,Kind:"cycle",Title:"Cycle "+cy.CycleID,
            Summary:"Engine Spark "+cognitionFirst(cy.Spark.Status,"NOT RECORDED")+" · "+linkStatus,
            State:cy.State,Source:"Thought Engine: cycle aggregate",SourceID:cy.CycleID,
            RecordedAt:cy.ReceivedAt,Verified:linkStatus,ThoughtID:thoughtID,
            Fields:[]CognitionField{
                cognitionField("Cycle ID",cy.CycleID),
                cognitionField("Recorded Spark Thought ID",cy.Spark.ThoughtID),
                cognitionField("Verified linked Thought",thoughtID),
                cognitionField("Outcome",cognitionCycleOutcome(cy)),
            },
        })
        if cy.Outcome!=nil {
            status:=cy.Outcome.Status
            out.Outcomes=append(out.Outcomes,CognitionCard{
                ID:"outcome:"+cy.CycleID,Kind:"outcome",
                Title:"Outcome: "+cognitionFirst(status,"NOT RECORDED"),
                Summary:"Executor-reported outcome; independently verified execution NOT RECORDED.",
                State:status,Source:"Thought Engine: executor-report",SourceID:cy.CycleID,
                RecordedAt:cy.ReceivedAt,Verified:"APP-REPORTED",ThoughtID:thoughtID,
                Fields:[]CognitionField{
                    cognitionField("Cycle",cy.CycleID),
                    cognitionField("Reported outcome",status),
                    cognitionField("Execution independently confirmed","NOT RECORDED"),
                },
            })
        }
    }
    // Other shelves are actual Core-owned records, never fabricated Thoughts.
    // Notes come from the same saved revision; a revision saved under another Identity entity is not shown.
    notes:=g.Notes
    notesNote:=""
    switch {
    case g.EntityID!="" && g.EntityID!=scope.identityID:
        notes,notesNote=nil,"Saved Development notes belong to another Identity entity and are not shown."
    case g.EntityID=="" && len(g.Notes)>0:
        notesNote="Development notes were saved before revisions recorded their Identity entity: IDENTITY NOT RECORDED."
    }
    for _,b:=range c.BrainBlocks(notes) {
        var kind,source,verified string
        switch b.Kind {
        case "lesson":kind,source,verified="lesson","TASS Core: learning status","APP-REPORTED"
        case "code-cell":kind,source,verified="cell","TASS Core: notebook code cell (read-only, never run here)","APP-REPORTED"
        case "text-cell":kind,source,verified="text-cell","TASS Core: notebook text cell (read-only)","APP-REPORTED"
        case "skill":kind,source,verified="skill","TASS Core: registered skill","APP-REPORTED"
        case "note":kind,source,verified="note",fmt.Sprintf("Core saved brain note (revision %d) — advisory, not a Thought",g.Revision),"OWNER-WRITTEN"
            if g.EntityID=="" {verified="OWNER-WRITTEN · IDENTITY NOT RECORDED"}
        default:continue
        }
        if len(out.Library)>=cognitionCardMax {out.Limited=true;break}
        card:=CognitionCard{
            ID:kind+":"+b.ID,Kind:kind,Title:cognitionClip(b.Title,100),
            Summary:cognitionClip(b.Purpose,240),State:b.Status,
            Source:source,SourceID:b.ID,Verified:verified,
            Fields:[]CognitionField{cognitionField("Source block",b.ID),cognitionField("Revision",b.Revision),
                cognitionField("Permission/Execution",b.Runnable),cognitionField("Last receipt",b.Last)},
        }
        if kind=="lesson" {
            f,cyc:=c.lessonEvidence(strings.TrimPrefix(b.ID,"lesson:"))
            card.Fields=append(card.Fields,f...)
            card.RelatedCycleIDs=cyc
        }
        out.Library=append(out.Library,card)
    }
    out.Junctions=c.junctionCards(scope.identityID)
    if notesNote!="" {out.Note=strings.TrimSpace(out.Note+" "+notesNote)}
    if out.Note=="" && len(out.Thoughts)==0 {out.Note="Persistent Thoughts NOT RECORDED for this engine-local entity."}
    return out,nil
}
func cognitionCycleOutcome(c engine.CognitionCycle) string{
    if c.Outcome==nil{return "NOT RECORDED"}
    return c.Outcome.Status
}
func (c *Core) CognitionThought(id string,rq ...*CognitionReq)(CognitionThought,error){return c.cognitionThought(id,true,rq...)}
// cognitionThought: withDevelopments=false for internal checks (e.g. SaveNudgeDraft, which holds
// nudgeMu; developments read the nudge journal under nudgeMu, and Note text isn't needed there).
func (c *Core) cognitionThought(id string,withDevelopments bool,rq ...*CognitionReq)(CognitionThought,error){
    scope,err:=c.cognitionAccess(rq...)
    if err!=nil{return CognitionThought{},err}
    if !engine.CognitionID.MatchString(id)||scope.engineID==""{return CognitionThought{},ErrCognitionUnavailable}
    raw,err:=c.engine.CognitionThought(scope.engineID,id)
    if err!=nil||raw.Thought.ThoughtID!=id||raw.Thought.EntityID!=scope.engineID{return CognitionThought{},ErrCognitionUnavailable}
    th:=raw.Thought
    out:=CognitionThought{ThoughtID:id,Topic:cognitionClip(th.Topic,180),Question:cognitionClip(th.Question,900),
        State:th.State,CreatedAt:th.CreatedAt,UpdatedAt:th.UpdatedAt,Source:"Thought Engine: persistent Thought",
        Verification:"OBSERVED",History:[]engine.ThoughtStep{},Traces:[]CognitionTrace{},RelatedCycleIDs:[]string{}}
    // P0 (#106): keep the NEWEST steps (the oldest are dropped first) and say how many exist.
    out.HistoryTotal=len(th.History)
    hist:=th.History
    if len(hist)>cognitionHistoryMax {hist=hist[len(hist)-cognitionHistoryMax:];out.Truncated=true}
    for _,h:=range hist{
        // Still no free-form Detail (it repeats the worker text); the Note text is shown once, in Developments.
        out.History=append(out.History,engine.ThoughtStep{
            At:h.At,Kind:cognitionClip(h.Kind,32),EventID:h.EventID,TraceID:h.TraceID,
            RequestID:cognitionRef(h.RequestID),NudgeID:cognitionNudgeRef(h.NudgeID),By:cognitionClip(h.By,40),
        })
    }
    out.Developments=[]CognitionDevelopment{}
    if withDevelopments {out.Developments,out.DevelopmentsTotal,out.DevelopmentsTruncated=c.cognitionDevelopments(scope,id,th)}
    for _,v:=range raw.Evidence {
        if len(out.Traces)>=64 {out.Truncated=true;break}
        trace:=v.Trace
        linked:=!v.Missing&&trace.ThoughtID==id&&trace.EntityID==scope.engineID
        state:="NOT RECORDED"
        if linked {state="OBSERVED"}
        if v.Event!=nil && linked {state="APP-REPORTED: "+cognitionClip(v.Event.Status,32)}
        out.Traces=append(out.Traces,CognitionTrace{
            TraceID:trace.TraceID,SourceEventID:trace.SourceEventID,
            At:trace.CreatedAt,Status:state,Verified:linked,
        })
    }
    cycles,err:=c.engine.CognitionCycles()
    if err!=nil{return CognitionThought{},ErrCognitionUnavailable}
    for _,cy:=range cycles {
        if cognitionOwnedCycle(cy,scope)&&cy.EntityID==scope.engineID&&cy.Spark.ThoughtID==id {
            out.RelatedCycleIDs=append(out.RelatedCycleIDs,cy.CycleID)
        }
    }
    sort.Strings(out.RelatedCycleIDs)
    return out,nil
}
// Defense in depth: even if an Engine response contains an unexpected raw
// field/value, Core exports ONLY known stages, statuses and reference-shaped IDs.
func cognitionRef(s string) string {
    if !engine.CognitionID.MatchString(s){return ""}
    return s
}
func cognitionOneOf(s string,allowed ...string) string {
    for _,a:=range allowed {if s==a{return s}}
    return "NOT RECORDED"
}
func cognitionSafeStep(x engine.RecordedStep) engine.RecordedStep {
    var out engine.RecordedStep
    out.Offset=x.Offset
    if out.Offset<0 {out.Offset=0}
    out.At=x.At
    out.Source="cycles.jsonl"
    out.Stage=cognitionOneOf(x.Stage,"received","spark","proposal","outcome","evaluation","problem")
    out.Status=cognitionOneOf(x.Status,"OBSERVED","APP-REPORTED","PROPOSED")
    out.EventID=cognitionRef(x.EventID)
    out.TraceID=cognitionRef(x.TraceID)
    out.ThoughtID=cognitionRef(x.ThoughtID)
    out.EntityID=cognitionRef(x.EntityID)
    out.ModelRequestID=cognitionRef(x.ModelRequestID)
    out.Action=cognitionOneOf(x.Action,"NONE","tass_sections","tass_words")
    out.PredictionState=cognitionOneOf(x.PredictionState,"LOCKED","NOT_PREDICTED")
    out.OutcomeStatus=cognitionOneOf(x.OutcomeStatus,"NO_ACTION","NOT_AUTHORIZED","EXECUTED","FAILED")
    out.Forecast=cognitionOneOf(x.Forecast,"NO_ACTION","PREDICTION_CORRECT","PREDICTION_INCORRECT","UNSCORABLE")
    out.Goal=cognitionOneOf(x.Goal,"GOAL_MET","GOAL_UNMET","GOAL_UNKNOWN")
    out.Duplicate=x.Duplicate
    out.OutOfOrder=x.OutOfOrder
    // Note intentionally not projected; it could hold raw private explanation.
    return out
}

func (c *Core) CognitionTimeline(id string,rq ...*CognitionReq)(CognitionTimeline,error){
    scope,err:=c.cognitionAccess(rq...)
    if err!=nil{return CognitionTimeline{},err}
    if !engine.CognitionID.MatchString(id){return CognitionTimeline{},ErrCognitionUnavailable}
    cy,err:=c.engine.CognitionCycle(id)
    if err!=nil||cy.CycleID!=id||!cognitionOwnedCycle(cy,scope){return CognitionTimeline{},ErrCognitionUnavailable}
    journal,err:=c.engine.CognitionJournal(id)
    if err!=nil{return CognitionTimeline{},ErrCognitionUnavailable}
    out:=CognitionTimeline{CycleID:id,Source:"Thought Engine: cycles.jsonl",
        EngineCommit:scope.engineCommit,ObservedAt:c.now().UTC(),
        State:cognitionOneOf(cy.State,"evaluated","awaiting_outcome","failed","denied","started"),Steps:[]engine.RecordedStep{},Missing:[]string{},Truncated:journal.Truncated,
        Partial:journal.Partial,InvalidLines:journal.InvalidLines,LinkStatus:"UNLINKED",
        IdentityEvidence:"NOT RECORDED: original cycle Identity decision receipt was not independently joined"}
    if cy.Spark.ThoughtID!=""&&cy.EntityID!=""&&cy.EntityID==scope.engineID {
        detail,err:=c.engine.CognitionThought(scope.engineID,cy.Spark.ThoughtID)
        if err==nil&&detail.Thought.EntityID==scope.engineID&&detail.Thought.ThoughtID==cy.Spark.ThoughtID {
            out.ThoughtID=cy.Spark.ThoughtID
            out.LinkStatus="VERIFIED-LINK"
        }
    }
    for _,step:=range journal.Steps {
        out.Steps=append(out.Steps,cognitionSafeStep(step))
    }
    out.Pending=[]string{}
    inFlight:=cy.State=="awaiting_outcome"||cy.State=="started"
    for _,stage:=range journal.Missing {
        if stage=="received"||stage=="spark"||stage=="proposal"||stage=="outcome"||stage=="evaluation" {
            // A cycle still in progress has not reached its later stages: PENDING, not missing evidence.
            if inFlight && (stage=="outcome"||stage=="evaluation"||(cy.State=="started"&&stage=="proposal")) {
                out.Pending=append(out.Pending,stage)
                continue
            }
            out.Missing=append(out.Missing,stage)
        }
    }
    if out.Truncated {out.Note="Journal bounded tail or event limit: historical entries may be missing"}
    if len(out.Steps)==0&&out.Note==""{out.Note="Historical journal steps NOT RECORDED"}
    return out,nil
}

// Export an explicit, minimal selected cycle. No prompts, freeform reasoning,
// context, trace notes, raw Identity details, local paths, or arbitrary JSONL.
type CognitionReport struct{
    Format string `json:"format"`
    CoreVersion string `json:"core_version"`
    EngineCommit string `json:"engine_commit"`
    CycleID string `json:"cycle_id"`
    ThoughtID string `json:"thought_id,omitempty"`
    LinkStatus string `json:"link_status"`
    Source string `json:"source"`
    State string `json:"cycle_state"`
    Steps []engine.RecordedStep `json:"steps"`
    Missing []string `json:"missing"`
    Pending []string `json:"pending"`
    Truncated bool `json:"truncated"`
    Partial bool `json:"partial"`
    InvalidLines int `json:"invalid_lines"`
    Verification string `json:"verification"`
    IdentityEvidence string `json:"identity_evidence"`
}
func (c *Core) CognitionExport(id,format,coreVersion string,rq ...*CognitionReq)([]byte,string,error){
    if format!="json"&&format!="md" {return nil,"",fmt.Errorf("unsupported Cognition export format")}
    v,err:=c.CognitionTimeline(id,rq...)
    if err!=nil{return nil,"",err}
    if len(coreVersion)>64 {coreVersion="NOT RECORDED"}
    report:=CognitionReport{Format:"Cognition sanitized v1",CoreVersion:coreVersion,
        EngineCommit:v.EngineCommit,CycleID:v.CycleID,ThoughtID:v.ThoughtID,
        LinkStatus:v.LinkStatus,Source:v.Source,State:v.State,Steps:v.Steps,
        Missing:v.Missing,Pending:v.Pending,Truncated:v.Truncated,Partial:v.Partial,
        InvalidLines:v.InvalidLines,Verification:"APP-REPORTED: an executor-reported outcome is not independently verified execution",
        IdentityEvidence:v.IdentityEvidence}
    if format=="json"{
        raw,err:=json.MarshalIndent(report,"","  ")
        if err!=nil{return nil,"",err}
        if len(raw)>64<<10{return nil,"",fmt.Errorf("sanitized report too large")}
        return append(raw,'\n'),"application/json",nil
    }
    var b bytes.Buffer
    fmt.Fprintf(&b,"# Cognition — sanitized recorded cycle\n\n- Core: %s\n- Engine: %s\n- Cycle: %s\n- Thought link: %s (%s)\n- Cycle state: %s\n- Source: %s\n- Verification: %s\n- Missing: %s\n- Pending (cycle still in progress): %s\n- Truncated: %v / partial: %v\n\n",
        coreVersion,v.EngineCommit,v.CycleID,cognitionFirst(v.ThoughtID,"NOT RECORDED"),
        v.LinkStatus,v.State,v.Source,report.Verification,cognitionFirst(strings.Join(v.Missing,", "),"none reported"),
        cognitionFirst(strings.Join(v.Pending,", "),"none"),
        v.Truncated,v.Partial)
    for _,s:=range report.Steps {
        at:="NOT RECORDED"
        if !s.At.IsZero(){at=s.At.UTC().Format(time.RFC3339Nano)}
        fmt.Fprintf(&b,"- %s / %s / byte %d / %s / status %s / action %s / outcome %s / thought %s / trace %s\n",
            at,s.Stage,s.Offset,s.Source,s.Status,cognitionFirst(s.Action,"NOT RECORDED"),
            cognitionFirst(s.OutcomeStatus,"NOT RECORDED"),cognitionFirst(s.ThoughtID,"NOT RECORDED"),
            cognitionFirst(s.TraceID,"NOT RECORDED"))
    }
    if b.Len()>64<<10{return nil,"",fmt.Errorf("sanitized report too large")}
    return b.Bytes(),"text/markdown",nil
}
