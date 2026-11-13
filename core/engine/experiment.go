package engine

// Stage 3B (#120, D4–D7): client for the Engine's isolated A/B experiment routes. The binding tuple's
// field order is identical to the Engine's (Thought-engine experiment.go).

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type ExperimentBinding struct {
	Action         string `json:"action"`
	OwnerEntityID  string `json:"owner_entity_id"`
	EngineEntityID string `json:"engine_entity_id"`
	ThoughtID      string `json:"thought_id"`
	ExperimentID   string `json:"experiment_id"`
	Arm            string `json:"arm"`
	SnapshotSHA    string `json:"snapshot_sha256"`
	Variable       string `json:"variable"`
	VariantSHA     string `json:"variant_sha256"`
}

type SnapshotIn struct {
	OwnerEntityID   string `json:"owner_entity_id"`
	Attention       string `json:"attention"`
	AttentionSHA256 string `json:"attention_sha256"`
}

// SnapshotInfo is the frozen snapshot's identity (never its text).
type SnapshotInfo struct {
	Present         bool      `json:"present"`
	Verified        bool      `json:"verified"`
	Problem         string    `json:"problem,omitempty"`
	SHA256          string    `json:"sha256,omitempty"`
	CapturedAt      time.Time `json:"captured_at,omitempty"`
	ThoughtID       string    `json:"thought_id,omitempty"`
	ThoughtRevision string    `json:"thought_revision,omitempty"`
	Evidence        int       `json:"evidence_lines"`
	Notes           int       `json:"note_lines"`
	EngineCommit    string    `json:"engine_commit,omitempty"`
	Created         bool      `json:"created,omitempty"`
}

type ArmIn struct {
	OwnerEntityID string `json:"owner_entity_id"`
	SnapshotSHA   string `json:"snapshot_sha256"`
	Variant       string `json:"variant"`
	VariantSHA    string `json:"variant_sha256"`
	DecisionID    string `json:"decision_id"`
}

type ArmResult struct {
	RequestID   string    `json:"request_id"`
	Reply       string    `json:"reply"`
	ReplySHA256 string    `json:"reply_sha256"`
	Parsed      bool      `json:"parsed"`
	ParseError  string    `json:"parse_error,omitempty"`
	Note        string    `json:"note,omitempty"`
	State       string    `json:"suggested_state,omitempty"`
	Model       string    `json:"model,omitempty"`
	ElapsedMS   int64     `json:"elapsed_ms"`
	At          time.Time `json:"at"`
}

type ExperimentArm struct {
	Arm            string     `json:"arm"`
	State          string     `json:"state"`
	Reason         string     `json:"reason,omitempty"`
	ModelHTTP      string     `json:"model_http,omitempty"`
	RequestID      string     `json:"request_id,omitempty"`
	SnapshotSHA    string     `json:"snapshot_sha256,omitempty"`
	PromptSHA      string     `json:"prompt_sha256,omitempty"`
	Instruction    string     `json:"instruction_sha256,omitempty"`
	Authorize      string     `json:"identity_decision,omitempty"`
	Redeem         string     `json:"identity_redeem,omitempty"`
	ClaimedAt      *time.Time `json:"claimed_at,omitempty"`
	Recovered      bool       `json:"recovered,omitempty"`
	Result         *ArmResult `json:"result,omitempty"`
	SavedToThought bool       `json:"saved_into_thought"`
}

type ExperimentRecord struct {
	ExperimentID string          `json:"experiment_id"`
	EntityID     string          `json:"entity_id"`
	Snapshot     SnapshotInfo    `json:"snapshot"`
	Arms         []ExperimentArm `json:"arms"`
	EngineCommit string          `json:"engine_commit"`
}

// Arm returns the named arm ("NOT_CLAIMED" when the record has none).
func (r ExperimentRecord) Arm(arm string) ExperimentArm {
	for _, a := range r.Arms {
		if a.Arm == arm {
			return a
		}
	}
	return ExperimentArm{Arm: arm, State: "NOT_CLAIMED"}
}

func (c *Client) experimentCall(ctx context.Context, method, path string, in, out any, timeout time.Duration) (int, error) {
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
		return 0, fmt.Errorf("Thought Engine not reachable (%v)", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != 200 {
		var e struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(raw, &e) != nil || e.Error == "" {
			e.Error = strings.TrimSpace(string(raw))
		}
		return resp.StatusCode, &ErrNudgeHTTP{resp.StatusCode, e.Error}
	}
	if json.Unmarshal(raw, out) != nil {
		return resp.StatusCode, fmt.Errorf("Thought Engine's answer could not be read")
	}
	return 200, nil
}

func experimentPath(eid, tid, exp string) string {
	return "/api/entities/" + url.PathEscape(eid) + "/thoughts/" + url.PathEscape(tid) + "/experiments/" + url.PathEscape(exp)
}

// CaptureExperimentSnapshot freezes the experiment's common inputs once (no model call).
func (c *Client) CaptureExperimentSnapshot(eid, tid, exp string, in SnapshotIn) (SnapshotInfo, int, error) {
	var out struct {
		Snapshot SnapshotInfo `json:"snapshot"`
	}
	code, err := c.experimentCall(context.Background(), "POST", experimentPath(eid, tid, exp)+"/snapshot", in, &out, 15*time.Second)
	return out.Snapshot, code, err
}

// RunExperimentArmCtx blocks until the arm's outcome (at most 10 minutes, or until ctx ends).
func (c *Client) RunExperimentArmCtx(ctx context.Context, eid, tid, exp, arm string, in ArmIn) (ExperimentRecord, int, error) {
	var out struct {
		Experiment ExperimentRecord `json:"experiment"`
	}
	code, err := c.experimentCall(ctx, "POST", experimentPath(eid, tid, exp)+"/arms/"+url.PathEscape(arm), in, &out, 10*time.Minute)
	return out.Experiment, code, err
}

// ExperimentRecord reads the snapshot's identity and both arms' derived states.
func (c *Client) ExperimentRecord(eid, exp string) (ExperimentRecord, int, error) {
	var out struct {
		Experiment ExperimentRecord `json:"experiment"`
	}
	code, err := c.experimentCall(context.Background(), "GET", "/api/entities/"+url.PathEscape(eid)+"/experiments/"+url.PathEscape(exp), nil, &out, 12*time.Second)
	if err == nil && (out.Experiment.ExperimentID != exp || out.Experiment.EntityID != eid) {
		return ExperimentRecord{}, code, fmt.Errorf("Thought Engine answered for another experiment")
	}
	return out.Experiment, code, err
}
