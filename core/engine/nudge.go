package engine

// Stage 2 client for the Thought Engine's supervised nudge routes (Thought-engine nudge.go).

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ProcessBinding / StopBinding: the exact tuples, field order identical to the Engine's.
type ProcessBinding struct {
	Action         string `json:"action"`
	OwnerEntityID  string `json:"owner_entity_id"`
	EngineEntityID string `json:"engine_entity_id"`
	ThoughtID      string `json:"thought_id"`
	NudgeID        string `json:"nudge_id"`
	AttentionSHA   string `json:"attention_sha256"`
	Recipient      string `json:"recipient"`
	DraftExpiresAt string `json:"draft_expires_at"`
}
type StopBinding struct {
	Action         string `json:"action"`
	OwnerEntityID  string `json:"owner_entity_id"`
	EngineEntityID string `json:"engine_entity_id"`
	ThoughtID      string `json:"thought_id"`
	NudgeID        string `json:"nudge_id"`
	SubmittedSHA   string `json:"submitted_sha256"`
}

// Binding is the sha256 hex of the tuple's JSON.
func Binding(v any) string {
	b, _ := json.Marshal(v)
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

type NudgeIn struct {
	NudgeID         string `json:"nudge_id"`
	OwnerEntityID   string `json:"owner_entity_id"`
	Attention       string `json:"attention"`
	AttentionSHA256 string `json:"attention_sha256"`
	DraftExpiresAt  string `json:"draft_expires_at"`
	DecisionID      string `json:"decision_id"`
}
type VoidIn struct {
	OwnerEntityID   string `json:"owner_entity_id"`
	ThoughtID       string `json:"thought_id"`
	SubmittedSHA256 string `json:"submitted_sha256"`
	DecisionID      string `json:"decision_id"`
}

// NudgeRun mirrors the Engine's run view.
type NudgeRun struct {
	NudgeID   string     `json:"nudge_id"`
	EntityID  string     `json:"entity_id"`
	ThoughtID string     `json:"thought_id,omitempty"`
	State     string     `json:"state"`
	Reason    string     `json:"reason,omitempty"`
	ModelHTTP string     `json:"model_http,omitempty"`
	RequestID string     `json:"request_id,omitempty"`
	StepAt    *time.Time `json:"step_at,omitempty"`
	Suggested string     `json:"suggested_state,omitempty"`
	Applied   bool       `json:"applied"`
	Recovered bool       `json:"recovered,omitempty"`
	Revisions struct {
		Claim string `json:"claim,omitempty"`
		P1    string `json:"p1,omitempty"`
		P2    string `json:"p2,omitempty"`
	} `json:"revisions"`
	Decisions struct {
		Authorize string `json:"authorize,omitempty"`
		Redeem    string `json:"redeem,omitempty"`
	} `json:"decisions"`
	Evidence struct {
		Notes int `json:"matching_notes"`
		Steps int `json:"matching_steps"`
	} `json:"evidence"`
	LedgerLocked string `json:"ledger_locked,omitempty"`
}

// ErrNudgeHTTP carries the Engine's status code for a refused request.
type ErrNudgeHTTP struct {
	Code int
	Msg  string
}

func (e *ErrNudgeHTTP) Error() string {
	return fmt.Sprintf("Thought Engine answered %d: %s", e.Code, e.Msg)
}

func (c *Client) nudgeCall(ctx context.Context, method, path string, in any, timeout time.Duration) (NudgeRun, int, error) {
	var body io.Reader
	if in != nil {
		b, _ := json.Marshal(in)
		body = bytes.NewReader(b)
	}
	req, _ := http.NewRequestWithContext(ctx, method, c.URL+path, body)
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Authorization", "Bearer "+c.Key)
	cl := *c.HTTP
	cl.Timeout = timeout
	resp, err := cl.Do(req)
	if err != nil {
		return NudgeRun{}, 0, fmt.Errorf("Thought Engine not reachable (%v)", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var out struct {
		Run   NudgeRun `json:"run"`
		Error string   `json:"error"`
	}
	if json.Unmarshal(raw, &out) != nil {
		return NudgeRun{}, resp.StatusCode, &ErrNudgeHTTP{resp.StatusCode, strings.TrimSpace(string(raw))}
	}
	if resp.StatusCode == 200 || (resp.StatusCode == 404 && out.Run.State == "NOT_CLAIMED") {
		return out.Run, resp.StatusCode, nil
	}
	return NudgeRun{}, resp.StatusCode, &ErrNudgeHTTP{resp.StatusCode, out.Error}
}

// ProcessNudgeCtx blocks until the Engine's outcome (at most 10 minutes, or until ctx ends).
func (c *Client) ProcessNudgeCtx(ctx context.Context, eid, tid string, in NudgeIn) (NudgeRun, int, error) {
	return c.nudgeCall(ctx, "POST", "/api/entities/"+url.PathEscape(eid)+"/thoughts/"+url.PathEscape(tid)+"/nudge", in, 10*time.Minute)
}
func (c *Client) VoidNudge(eid, nid string, in VoidIn) (NudgeRun, int, error) {
	return c.nudgeCall(context.Background(), "POST", "/api/entities/"+url.PathEscape(eid)+"/nudges/"+url.PathEscape(nid)+"/void", in, 15*time.Second)
}
func (c *Client) NudgeRun(eid, nid string) (NudgeRun, int, error) {
	return c.nudgeCall(context.Background(), "GET", "/api/entities/"+url.PathEscape(eid)+"/nudges/"+url.PathEscape(nid), nil, 12*time.Second)
}
