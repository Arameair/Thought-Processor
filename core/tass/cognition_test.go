package tass

import (
    "bytes"
    "encoding/json"
    "errors"
    "net/http"
    "net/http/httptest"
    "strings"
    "sync"
    "testing"
    "time"

    "github.com/Arameair/Thought-Processor/core/engine"
)

func TestCognitionUsesCanonicalIdentityAndActualIDs(t *testing.T) {
    ids,legacy:=newFakeIdentitySvc(),newFakeEngine()
    t.Cleanup(func(){ids.srv.Close();legacy.srv.Close()})
    c:=newIdentityCore(t,t.TempDir(),ids,legacy)
    // LOCKED must refuse before contacting any Engine data endpoint.
    if _,err:=c.CognitionLibrary();!errors.Is(err,ErrCognitionDenied){t.Fatalf("LOCKED read: %v",err)}
    ids.set("UNLOCKED")
    stamp:=time.Date(2026,11,5,13,1,2,0,time.UTC)
    var mu sync.Mutex
    calls:=[]string{}
    mock:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
        mu.Lock();calls=append(calls,r.Method+" "+r.URL.Path);mu.Unlock()
        if r.Header.Get("Authorization")!="Bearer testkey" {w.WriteHeader(401);return}
        if r.Method!="GET" {w.WriteHeader(405);return}
        w.Header().Set("Content-Type","application/json")
        switch r.URL.Path {
        case "/api/status":
            json.NewEncoder(w).Encode(map[string]any{"entities":[]any{
                map[string]any{"card":map[string]string{"entity_id":"eng_1","display_name":"Mira","entity_type":"ai_companion"}},
                map[string]any{"card":map[string]string{"entity_id":"eng_other","display_name":"Other","entity_type":"ai_companion"}},
            },"config":map[string]string{"commit":"21b3d502775b3ed065ab445b88f51909500770b3"}})
        case "/api/entities/eng_1":
            json.NewEncoder(w).Encode(map[string]any{"card":map[string]string{"entity_id":"eng_1","display_name":"Mira"},
                "thoughts":[]any{
                    map[string]any{"thought_id":"thought_1","entity_id":"eng_1","topic":"real topic","question":"real question","state":"active","updated_at":stamp},
                    map[string]any{"thought_id":"other_wrong","entity_id":"eng_other","topic":"must not leak"},
                },"traces":[]any{}})
        case "/api/entities/eng_1/thoughts/thought_1":
            json.NewEncoder(w).Encode(map[string]any{"thought":map[string]any{
                "thought_id":"thought_1","entity_id":"eng_1","topic":"real topic","question":"a private question",
                "state":"active","history":[]any{map[string]any{"at":stamp,"kind":"formed","trace_id":"trace_1","detail":"SECRET INTERNAL NOTES"}}},
                "evidence":[]any{map[string]any{"trace":map[string]any{
                    "trace_id":"trace_1","entity_id":"eng_1","thought_id":"thought_1","source_event_id":"evt_1","created_at":stamp},
                    "event":map[string]any{"event_id":"evt_1","status":"done"},"origin":true}}})
        case "/api/cycles":
            json.NewEncoder(w).Encode(map[string]any{"cycles":[]any{
                map[string]any{"cycle_id":"tp_1","identity":"Mira","entity_id":"eng_1","received_at":stamp,
                    "state":"evaluated","spark":map[string]any{"thought_id":"thought_1","status":"done"},
                    "outcome":map[string]any{"status":"NO_ACTION"}},
                map[string]any{"cycle_id":"tp_unlinked","identity":"Mira","entity_id":"eng_1","received_at":stamp,
                    "state":"evaluated","spark":map[string]any{"thought_id":"missing_thought","status":"done"},
                    "outcome":map[string]any{"status":"NOT_AUTHORIZED"}},
                map[string]any{"cycle_id":"foreign","identity":"Other","entity_id":"eng_other","spark":map[string]any{"thought_id":"thought_1"}},
                map[string]any{"cycle_id":"cross_entity","identity":"Mira","entity_id":"eng_other","spark":map[string]any{"thought_id":"thought_1"}},
            }})
        case "/api/cycles/tp_1":
            json.NewEncoder(w).Encode(map[string]any{"cycle_id":"tp_1","identity":"Mira","entity_id":"eng_1","state":"evaluated",
                "spark":map[string]any{"thought_id":"thought_1","status":"done"},"outcome":map[string]any{"status":"NO_ACTION"}})
        case "/api/cycles/cross_entity":
            json.NewEncoder(w).Encode(map[string]any{"cycle_id":"cross_entity","identity":"Mira","entity_id":"eng_other"})
        case "/api/cycles/tp_1/journal":
            json.NewEncoder(w).Encode(map[string]any{"cycle_id":"tp_1","source":"cycles.jsonl","steps":[]any{
                map[string]any{"at":stamp,"offset":0,"stage":"received","source":"cycles.jsonl","status":"OBSERVED",
                    "note":"Authorization: Bearer FAKE-SECRET-TOKEN"},
                map[string]any{"at":stamp.Add(time.Second),"offset":123,"stage":"proposal","source":"cycles.jsonl",
                    "status":"PROPOSED","action":"NONE","prediction_state":"LOCKED","model_request_id":"req_1"},
                map[string]any{"at":stamp.Add(2*time.Second),"offset":254,"stage":"outcome","source":"cycles.jsonl",
                    "status":"APP-REPORTED","outcome_status":"NO_ACTION"},
            },"missing":[]string{"spark","evaluation"},"truncated":false})
        default:
            w.WriteHeader(404)
        }
    }))
    defer mock.Close()
    c.engine=engine.New(mock.URL);c.engine.Key="testkey"
    lib,err:=c.CognitionLibrary()
    if err!=nil{t.Fatal(err)}
    if lib.IdentityEntityID!="ent_0123"||lib.EngineEntityID!="eng_1"||len(lib.Thoughts)!=1||
       len(lib.Cycles)!=2||len(lib.Outcomes)!=2||lib.Thoughts[0].SourceID!="thought_1"||
       len(lib.Thoughts[0].RelatedCycleIDs)!=1||lib.Thoughts[0].RelatedCycleIDs[0]!="tp_1" {
        t.Fatalf("IDs/records not correlated: %+v",lib)
    }
    if lib.Cycles[0].Verification!="VERIFIED-LINK"||lib.Cycles[1].Verification!="UNLINKED" {
        t.Fatalf("cycle links were invented: %+v",lib.Cycles)
    }
    if lib.Outcomes[0].Verified=="EXECUTED"||lib.Outcomes[0].Verified!="APP-REPORTED" {
        t.Fatalf("NO_ACTION is not verified execution: %+v",lib.Outcomes[0])
    }
    th,err:=c.CognitionThought("thought_1")
    if err!=nil||len(th.Traces)!=1||!th.Traces[0].Verified||len(th.RelatedCycleIDs)!=1 {
        t.Fatalf("thought receipt: %+v, %v",th,err)
    }
    if len(th.History)!=1||strings.Contains(strings.Join([]string{th.History[0].EventID,th.History[0].Kind}," "),"SECRET"){
        t.Fatal("Thought history leaks internal text")
    }
    view,err:=c.CognitionTimeline("tp_1")
    if err!=nil||view.ThoughtID!="thought_1"||view.LinkStatus!="VERIFIED-LINK"||len(view.Steps)!=3 {
        t.Fatalf("journal projection %+v: %v",view,err)
    }
    for _,format:=range []string{"md","json"} {
        out,media,err:=c.CognitionExport("tp_1",format,"core-fixture-sha")
        if err!=nil||len(out)==0||media==""{t.Fatalf("export %s: %v",format,err)}
        for _,private:=range []string{"FAKE-SECRET-TOKEN","SECRET INTERNAL NOTES","a private question","model_reasoning","Bearer"} {
            if bytes.Contains(out,[]byte(private)){t.Fatalf("%s export leaked %q",format,private)}
        }
        if !bytes.Contains(out,[]byte("tp_1"))||bytes.Contains(out,[]byte("EXECUTED")) {
            t.Fatalf("%s export missing provenance or invented execution",format)
        }
    }
    if _,err:=c.CognitionTimeline("cross_entity");!errors.Is(err,ErrCognitionUnavailable) {
        t.Fatalf("cross-entity cycle must be refused: %v",err)
    }
    if _,err:=c.CognitionTimeline("../secrets");!errors.Is(err,ErrCognitionUnavailable) {
        t.Fatalf("path traversal must be refused: %v",err)
    }
    ids.set("LOCKED")
    mu.Lock();before:=len(calls);mu.Unlock()
    if _,err:=c.CognitionTimeline("tp_1");!errors.Is(err,ErrCognitionDenied){t.Fatalf("LOCKED timeline: %v",err)}
    mu.Lock();after:=len(calls);for _,call:=range calls{if !strings.HasPrefix(call,"GET "){t.Fatalf("cognition sent an Engine operation: %s",call)}};mu.Unlock()
    if before!=after {t.Fatal("LOCKED read contacted Engine")}
}
