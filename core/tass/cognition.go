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
    ObservedAt time.Time `json:"observed_at"`
    State string `json:"state"`
    Steps []engine.RecordedStep `json:"steps"`
    Missing []string `json:"missing"`
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
}
func (c *Core) cognitionAccess()(cognitionScope,error){
    var out cognitionScope
    if c.cfg.IdentitySvc==nil {return out,ErrCognitionDenied}
    state:=c.cfg.IdentitySvc.State()
    if state.State!="UNLOCKED"||state.Entity==nil||state.Entity.EntityID==""||state.Entity.Name=="" {
        return out,ErrCognitionDenied
    }
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
    if out.engineCommit=="" {out.engineCommit="NOT RECORDED"}
    for _,entity:=range status.Entities {
        if strings.EqualFold(entity.Card.DisplayName,out.name)&&entity.Card.EntityType=="ai_companion" {
            if out.engineID!="" {return out,ErrCognitionUnavailable} // ambiguous engine identity: fail closed
            out.engineID=entity.Card.EntityID
        }
    }
    return out,nil
}
func cognitionOwnedCycle(cy engine.CognitionCycle,scope cognitionScope)bool {
    if !strings.EqualFold(cy.Identity,scope.name) {return false}
    if cy.EntityID!="" && scope.engineID!="" && cy.EntityID!=scope.engineID {return false}
    if cy.EntityID!="" && scope.engineID=="" {return false}
    return true
}
func (c *Core) CognitionLibrary()(CognitionLibrary,error){
    scope,err:=c.cognitionAccess()
    if err!=nil{return CognitionLibrary{},err}
    out:=CognitionLibrary{IdentityEntityID:scope.identityID,EngineEntityID:scope.engineID,
        IdentityName:scope.name,EngineCommit:scope.engineCommit,ObservedAt:c.now().UTC(),
        Thoughts:[]CognitionCard{},Cycles:[]CognitionCard{},Outcomes:[]CognitionCard{},Library:[]CognitionCard{}}
    out.BrainRevision=c.Brain().Revision
    entity:=engine.CognitionEntity{}
    if scope.engineID!="" {
        entity,err=c.engine.CognitionEntity(scope.engineID)
        if err!=nil||entity.Card.EntityID!=scope.engineID||!strings.EqualFold(entity.Card.DisplayName,scope.name){
            return CognitionLibrary{},ErrCognitionUnavailable
        }
    } else {
        out.Note="No matching engine-local AI companion entity; persistent Thoughts NOT RECORDED."
    }
    cycles,err:=c.engine.CognitionCycles()
    if err!=nil{return CognitionLibrary{},ErrCognitionUnavailable}
    validCycles:=[]engine.CognitionCycle{}
    for _,cy:=range cycles {if cognitionOwnedCycle(cy,scope){validCycles=append(validCycles,cy)}}
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
    for _,b:=range c.BrainBlocks(nil) {
        var kind string
        switch b.Kind {
        case "lesson":kind="lesson"
        case "code-cell":kind="cell"
        case "skill":kind="skill"
        case "note":kind="note"
        default:continue
        }
        if len(out.Library)>=cognitionCardMax {out.Limited=true;break}
        out.Library=append(out.Library,CognitionCard{
            ID:kind+":"+b.ID,Kind:kind,Title:cognitionClip(b.Title,100),
            Summary:cognitionClip(b.Purpose,240),State:b.Status,
            Source:"TASS Core: BrainBlocks",SourceID:b.ID,Verified:"APP-REPORTED",
            Fields:[]CognitionField{cognitionField("Source block",b.ID),cognitionField("Revision",b.Revision),
                cognitionField("Permission/Execution",b.Runnable),cognitionField("Last receipt",b.Last)},
        })
    }
    if out.Note=="" && len(out.Thoughts)==0 {out.Note="Persistent Thoughts NOT RECORDED for this engine-local entity."}
    return out,nil
}
func cognitionCycleOutcome(c engine.CognitionCycle) string{
    if c.Outcome==nil{return "NOT RECORDED"}
    return c.Outcome.Status
}
func (c *Core) CognitionThought(id string)(CognitionThought,error){
    scope,err:=c.cognitionAccess()
    if err!=nil{return CognitionThought{},err}
    if !engine.CognitionID.MatchString(id)||scope.engineID==""{return CognitionThought{},ErrCognitionUnavailable}
    raw,err:=c.engine.CognitionThought(scope.engineID,id)
    if err!=nil||raw.Thought.ThoughtID!=id||raw.Thought.EntityID!=scope.engineID{return CognitionThought{},ErrCognitionUnavailable}
    th:=raw.Thought
    out:=CognitionThought{ThoughtID:id,Topic:cognitionClip(th.Topic,180),Question:cognitionClip(th.Question,900),
        State:th.State,CreatedAt:th.CreatedAt,UpdatedAt:th.UpdatedAt,Source:"Thought Engine: persistent Thought",
        Verification:"OBSERVED",History:[]engine.ThoughtStep{},Traces:[]CognitionTrace{},RelatedCycleIDs:[]string{}}
    for _,h:=range th.History{
        if len(out.History)>=64 {out.Truncated=true;break}
        // No free-form Detail: includes private worker-generated strings.
        out.History=append(out.History,engine.ThoughtStep{
            At:h.At,Kind:h.Kind,EventID:h.EventID,TraceID:h.TraceID,RequestID:h.RequestID,
        })
    }
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
func (c *Core) CognitionTimeline(id string)(CognitionTimeline,error){
    scope,err:=c.cognitionAccess()
    if err!=nil{return CognitionTimeline{},err}
    if !engine.CognitionID.MatchString(id){return CognitionTimeline{},ErrCognitionUnavailable}
    cy,err:=c.engine.CognitionCycle(id)
    if err!=nil||cy.CycleID!=id||!cognitionOwnedCycle(cy,scope){return CognitionTimeline{},ErrCognitionUnavailable}
    journal,err:=c.engine.CognitionJournal(id)
    if err!=nil{return CognitionTimeline{},ErrCognitionUnavailable}
    out:=CognitionTimeline{CycleID:id,Source:"Thought Engine: cycles.jsonl",
        EngineCommit:scope.engineCommit,ObservedAt:c.now().UTC(),
        State:cy.State,Steps:journal.Steps,Missing:journal.Missing,Truncated:journal.Truncated,
        Partial:journal.Partial,InvalidLines:journal.InvalidLines,Note:journal.Note,LinkStatus:"UNLINKED"}
    if cy.Spark.ThoughtID!=""&&cy.EntityID!=""&&cy.EntityID==scope.engineID {
        detail,err:=c.engine.CognitionThought(scope.engineID,cy.Spark.ThoughtID)
        if err==nil&&detail.Thought.EntityID==scope.engineID&&detail.Thought.ThoughtID==cy.Spark.ThoughtID {
            out.ThoughtID=cy.Spark.ThoughtID
            out.LinkStatus="VERIFIED-LINK"
        }
    }
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
    Truncated bool `json:"truncated"`
    Partial bool `json:"partial"`
    InvalidLines int `json:"invalid_lines"`
    Verification string `json:"verification"`
}
func (c *Core) CognitionExport(id,format,coreVersion string)([]byte,string,error){
    if format!="json"&&format!="md" {return nil,"",fmt.Errorf("unsupported Cognition export format")}
    v,err:=c.CognitionTimeline(id)
    if err!=nil{return nil,"",err}
    report:=CognitionReport{Format:"Cognition sanitized v1",CoreVersion:coreVersion,
        EngineCommit:v.EngineCommit,CycleID:v.CycleID,ThoughtID:v.ThoughtID,
        LinkStatus:v.LinkStatus,Source:v.Source,State:v.State,Steps:v.Steps,
        Missing:v.Missing,Truncated:v.Truncated,Partial:v.Partial,
        InvalidLines:v.InvalidLines,Verification:"APP-REPORTED unless independently correlated"}
    if format=="json"{
        raw,err:=json.MarshalIndent(report,"","  ")
        if err!=nil{return nil,"",err}
        if len(raw)>64<<10{return nil,"",fmt.Errorf("sanitized report too large")}
        return append(raw,'\n'),"application/json",nil
    }
    var b bytes.Buffer
    fmt.Fprintf(&b,"# Cognition — sanitized recorded cycle\n\n- Core: %s\n- Engine: %s\n- Cycle: %s\n- Thought link: %s (%s)\n- Cycle state: %s\n- Source: %s\n- Verification: %s\n- Missing: %s\n- Truncated: %v / partial: %v\n\n",
        coreVersion,v.EngineCommit,v.CycleID,cognitionFirst(v.ThoughtID,"NOT RECORDED"),
        v.LinkStatus,v.State,v.Source,report.Verification,cognitionFirst(strings.Join(v.Missing,", "),"none reported"),
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
