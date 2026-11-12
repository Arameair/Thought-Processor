package engine

// Stage 3A (#120; design #107 §3.2): Core's client for the Engine's read-only run record,
// GET /api/entities/{eid}/nudges/{nid}/record (Thought Engine b594357 and later). Read only.

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type RecordFile[T any] struct {
	Present  bool `json:"present"`
	Readable bool `json:"readable"`
	Value    *T   `json:"value,omitempty"`
}

type RecordClaim struct {
	Void           bool      `json:"void,omitempty"`
	NudgeID        string    `json:"nudge_id"`
	EntityID       string    `json:"entity_id"`
	ThoughtID      string    `json:"thought_id"`
	AttentionSHA   string    `json:"attention_sha256,omitempty"`
	AttentionChars int       `json:"attention_chars,omitempty"`
	RequestID      string    `json:"request_id,omitempty"`
	Revision       string    `json:"revision,omitempty"`
	Decision       string    `json:"identity_decision,omitempty"`
	Redeemed       string    `json:"identity_redeem,omitempty"`
	At             time.Time `json:"at"`
}
type RecordHTTP struct {
	RequestID string    `json:"request_id"`
	At        time.Time `json:"at"`
}
type RecordResponse struct {
	OK          bool      `json:"ok"`
	Error       string    `json:"error,omitempty"`
	HTTPAnswer  bool      `json:"http_answered"`
	ReplySHA256 string    `json:"reply_sha256,omitempty"`
	ElapsedMS   int64     `json:"elapsed_ms"`
	At          time.Time `json:"at"`
}
type RecordOutcome struct {
	State     string `json:"state"`
	Reason    string `json:"reason,omitempty"`
	ModelHTTP string `json:"model_http"`
	Suggested string `json:"suggested_state,omitempty"`
	Applied   bool   `json:"applied"`
	RevP1     string `json:"revision_p1,omitempty"`
	RevP2     string `json:"revision_p2,omitempty"`
}
type RecordCall struct {
	At            time.Time `json:"at"`
	RequestID     string    `json:"request_id"`
	Status        string    `json:"status"`
	Prompt        string    `json:"prompt_as_logged"`
	PromptClipped bool      `json:"prompt_clipped"`
	Reply         string    `json:"reply_as_logged"`
	ReplyClipped  bool      `json:"reply_clipped"`
	Error         string    `json:"error,omitempty"`
}
type RecordModelRequest struct {
	At         time.Time `json:"at"`
	RequestID  string    `json:"request_id"`
	OK         bool      `json:"ok"`
	Text       string    `json:"text"`
	ElapsedMS  int64     `json:"elapsed_ms"`
	Error      string    `json:"error,omitempty"`
	Authorized string    `json:"authorized"`
	Model      string    `json:"model"`
}
type RecordScan struct {
	Found   bool     `json:"found"`
	Files   []string `json:"files"`
	Scanned int64    `json:"bytes_scanned"`
	Window  int64    `json:"window_bytes"`
}
type NudgeRecord struct {
	Run          NudgeRun                   `json:"run"`
	Claim        RecordFile[RecordClaim]    `json:"claim"`
	HTTP         RecordFile[RecordHTTP]     `json:"http"`
	Response     RecordFile[RecordResponse] `json:"response"`
	Outcome      RecordFile[RecordOutcome]  `json:"outcome"`
	Call         *RecordCall                `json:"call"`
	CallScan     RecordScan                 `json:"call_scan"`
	ModelRequest *RecordModelRequest        `json:"model_request"`
	ModelScan    RecordScan                 `json:"model_request_scan"`
	EngineCommit string                     `json:"engine_commit"`
}

// ErrNoRecordRoute: the Engine answered without its JSON error shape (an Engine older than b594357
// has no such route). Core reports the run record as unavailable rather than as not recorded.
var ErrNoRecordRoute = fmt.Errorf("this Thought Engine has no run-record read")

// NudgeRecord returns (record, HTTP status, error). 404 with a JSON body: the Engine never claimed it.
func (c *Client) NudgeRecord(eid, nid string) (NudgeRecord, int, error) {
	req, _ := http.NewRequest("GET", c.URL+"/api/entities/"+url.PathEscape(eid)+"/nudges/"+url.PathEscape(nid)+"/record", nil)
	req.Header.Set("Authorization", "Bearer "+c.Key)
	cl := *c.HTTP
	cl.Timeout = 15 * time.Second
	resp, err := cl.Do(req)
	if err != nil {
		return NudgeRecord{}, 0, fmt.Errorf("Thought Engine not reachable (%v)", err)
	}
	defer resp.Body.Close()
	// The record holds at most one full reply plus clipped log text; 4 MiB is far above it.
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode == 200 {
		var rec NudgeRecord
		if err := json.Unmarshal(raw, &rec); err != nil {
			return NudgeRecord{}, 200, fmt.Errorf("unreadable run record: %v", err)
		}
		return rec, 200, nil
	}
	var e struct {
		Error string `json:"error"`
	}
	if json.Unmarshal(raw, &e) != nil || e.Error == "" {
		return NudgeRecord{}, resp.StatusCode, ErrNoRecordRoute
	}
	return NudgeRecord{}, resp.StatusCode, &ErrNudgeHTTP{resp.StatusCode, strings.TrimSpace(e.Error)}
}
