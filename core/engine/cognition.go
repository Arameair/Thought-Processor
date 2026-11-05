package engine

// Sensitive Cognition reads use the Core's already configured Engine service
// key. Browser requests never call Engine directly or supply a filesystem path.

import (
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "net/url"
    "regexp"
    "strings"
    "time"
)

var CognitionID = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,99}$`)

func (c *Client) cognitionGet(path string, out any) error {
    if c.Key=="" {return fmt.Errorf("Thought Engine service key NOT RECORDED")}
    req,err:=http.NewRequest(http.MethodGet,c.URL+path,nil)
    if err!=nil {return err}
    req.Header.Set("Authorization","Bearer "+c.Key)
    cl:=*c.HTTP
    cl.Timeout=12*time.Second
    resp,err:=cl.Do(req)
    if err!=nil{return fmt.Errorf("Thought Engine read unavailable")}
    defer resp.Body.Close()
    if resp.StatusCode!=200{return fmt.Errorf("Thought Engine read refused (HTTP %d)",resp.StatusCode)}
    // Reject oversize, even when JSON happens to parse before the limit.
    raw,err:=io.ReadAll(io.LimitReader(resp.Body, 6<<20+1))
    if err!=nil||len(raw)>6<<20{return fmt.Errorf("Thought Engine read exceeded 6 MiB bound")}
    if err:=json.Unmarshal(raw,out);err!=nil{return fmt.Errorf("Thought Engine returned invalid JSON")}
    return nil
}

type CognitionCard struct {
    EntityID string `json:"entity_id"`
    DisplayName string `json:"display_name"`
    EntityType string `json:"entity_type"`
}
type CognitionEntitySummary struct {Card CognitionCard `json:"card"`}
type CognitionStatus struct {
    Entities []CognitionEntitySummary `json:"entities"`
    Config struct{Commit string `json:"commit"`} `json:"config"`
}
func (c *Client) CognitionStatus() (CognitionStatus,error) {
    var out CognitionStatus
    err:=c.cognitionGet("/api/status",&out)
    return out,err
}

type ThoughtStep struct{
    At time.Time `json:"at"`
    Kind string `json:"kind"`
    EventID string `json:"event_id"`
    TraceID string `json:"trace_id"`
    RequestID string `json:"worker_request_id"`
}
type Thought struct{
    ThoughtID string `json:"thought_id"`
    EntityID string `json:"entity_id"`
    Topic string `json:"topic"`
    Question string `json:"question"`
    State string `json:"state"`
    TraceIDs []string `json:"trace_ids"`
    OriginTraceIDs []string `json:"origin_trace_ids"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
    History []ThoughtStep `json:"history"`
}
type Trace struct{
    TraceID string `json:"trace_id"`
    EntityID string `json:"entity_id"`
    SourceEventID string `json:"source_event_id"`
    ThoughtID string `json:"thought_id"`
    CreatedAt time.Time `json:"created_at"`
}
type CognitionEntity struct{
    Card CognitionCard `json:"card"`
    Thoughts []Thought `json:"thoughts"`
    Traces []Trace `json:"traces"`
}
func (c *Client) CognitionEntity(id string) (CognitionEntity,error) {
    var out CognitionEntity
    if !CognitionID.MatchString(id) {return out,fmt.Errorf("invalid Engine entity ID")}
    err:=c.cognitionGet("/api/entities/"+url.PathEscape(id),&out)
    return out,err
}
type ThoughtEvidence struct{
    Trace Trace `json:"trace"`
    Origin bool `json:"origin"`
    Missing bool `json:"missing"`
    Event *struct{
        EventID string `json:"event_id"`
        Status string `json:"status"`
        Outcome string `json:"outcome"`
    } `json:"event"`
}
type CognitionThoughtDetail struct{
    Thought Thought `json:"thought"`
    Evidence []ThoughtEvidence `json:"evidence"`
}
func (c *Client) CognitionThought(entityID,thoughtID string)(CognitionThoughtDetail,error){
    var out CognitionThoughtDetail
    if !CognitionID.MatchString(entityID)||!CognitionID.MatchString(thoughtID){return out,fmt.Errorf("invalid Thought reference")}
    err:=c.cognitionGet("/api/entities/"+url.PathEscape(entityID)+"/thoughts/"+url.PathEscape(thoughtID),&out)
    return out,err
}

type CognitionCycle struct {
    CycleID string `json:"cycle_id"`
    Identity string `json:"identity"`
    EntityID string `json:"entity_id"`
    Source string `json:"source"`
    State string `json:"state"`
    ReceivedAt time.Time `json:"received_at"`
    Spark struct{
        Status string `json:"status"`
        EventID string `json:"event_id"`
        TraceID string `json:"trace_id"`
        ThoughtID string `json:"thought_id"`
    } `json:"spark"`
    Proposal *struct{
        ModelRequestID string `json:"model_request_id"`
        Action string `json:"action"`
        Prediction struct{State string `json:"state"`} `json:"prediction"`
    } `json:"proposal"`
    Outcome *struct{ Status string `json:"status"` } `json:"outcome"`
    Evaluation *struct{Forecast string `json:"forecast"`;Goal string `json:"goal"`} `json:"evaluation"`
}
func (c *Client) CognitionCycles()([]CognitionCycle,error){
    var out struct{Cycles []CognitionCycle `json:"cycles"`}
    err:=c.cognitionGet("/api/cycles",&out)
    if len(out.Cycles)>20 {out.Cycles=out.Cycles[:20]}
    return out.Cycles,err
}
func (c *Client) CognitionCycle(id string)(CognitionCycle,error){
    var out CognitionCycle
    if !CognitionID.MatchString(id){return out,fmt.Errorf("invalid cycle ID")}
    err:=c.cognitionGet("/api/cycles/"+url.PathEscape(id),&out)
    return out,err
}

type RecordedStep struct {
    Offset int64 `json:"offset"`
    At time.Time `json:"at"`
    Stage string `json:"stage"`
    Source string `json:"source"`
    EventID string `json:"event_id,omitempty"`
    TraceID string `json:"trace_id,omitempty"`
    ThoughtID string `json:"thought_id,omitempty"`
    ModelRequestID string `json:"model_request_id,omitempty"`
    EntityID string `json:"entity_id,omitempty"`
    Action string `json:"action,omitempty"`
    PredictionState string `json:"prediction_state,omitempty"`
    OutcomeStatus string `json:"outcome_status,omitempty"`
    Forecast string `json:"forecast,omitempty"`
    Goal string `json:"goal,omitempty"`
    Status string `json:"status"`
    Note string `json:"note,omitempty"`
    Duplicate bool `json:"duplicate,omitempty"`
    OutOfOrder bool `json:"out_of_order,omitempty"`
}
type CognitionJournal struct {
    CycleID string `json:"cycle_id"`
    Source string `json:"source"`
    Steps []RecordedStep `json:"steps"`
    Truncated bool `json:"truncated"`
    Partial bool `json:"partial"`
    InvalidLines int `json:"invalid_lines"`
    Missing []string `json:"missing"`
    Note string `json:"note"`
}
func (c *Client) CognitionJournal(id string)(CognitionJournal,error){
    var out CognitionJournal
    if !CognitionID.MatchString(id){return out,fmt.Errorf("invalid cycle ID")}
    err:=c.cognitionGet("/api/cycles/"+url.PathEscape(id)+"/journal",&out)
    if len(out.Steps)>96{return CognitionJournal{},fmt.Errorf("Engine journal exceeds step limit")}
    if out.CycleID!=id{return CognitionJournal{},fmt.Errorf("Engine journal cycle ID mismatch")}
    for _,s:=range out.Steps {if s.Stage==""||!strings.HasSuffix(s.Source,"cycles.jsonl"){return CognitionJournal{},fmt.Errorf("Engine journal source invalid")}}
    return out,err
}
