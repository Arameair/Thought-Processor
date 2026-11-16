// Package engine is Thought-Processor's connection to the Thought Engine, which owns the
// Nemotron model: Processor → Engine → Nemotron → Engine → Processor.
package engine

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Result is the engine's answer to one model request.
type Result struct {
	RequestID  string `json:"request_id"`
	OK         bool   `json:"ok"`
	Text       string `json:"text"`
	Thought    string `json:"thought"`
	ElapsedMS  int64  `json:"elapsed_ms"`
	Error      string `json:"error"`
	Authorized string `json:"authorized"`
}

type Client struct {
	URL  string // http://127.0.0.1:7798
	Key  string // the engine's api-key (only Thought-Processor is given it)
	HTTP *http.Client
}

func New(url string) *Client {
	return &Client{URL: strings.TrimRight(url, "/"), HTTP: &http.Client{Timeout: 10 * time.Minute}}
}

func NewRequestID() string {
	b := make([]byte, 6)
	rand.Read(b)
	return "tp_" + time.Now().UTC().Format("20060102T150405") + "_" + hex.EncodeToString(b)
}

// Think sends one model request through the engine.
func (c *Client) Think(requestID, identity, purpose string, msgs []Message) (*Result, error) {
	body, _ := json.Marshal(map[string]any{"request_id": requestID, "identity": identity, "purpose": purpose, "messages": msgs})
	req, _ := http.NewRequest("POST", c.URL+"/api/model/requests", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.Key)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Thought Engine not reachable at %s (%v)", c.URL, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	var r Result
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, fmt.Errorf("Thought Engine answered %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	if r.RequestID == "" { // validation error shape {"error": ...}
		var e struct {
			Error string `json:"error"`
		}
		json.Unmarshal(raw, &e)
		return nil, fmt.Errorf("Thought Engine refused the request (%d): %s", resp.StatusCode, e.Error)
	}
	if !r.OK {
		return &r, fmt.Errorf("%s", r.Error)
	}
	return &r, nil
}

// Status is the engine's model link and its last local pulse.
type Status struct {
	Reachable bool `json:"reachable"`
	LastPulse *struct {
		At      time.Time `json:"at"`
		ModelOK bool      `json:"model_ok"`
		Error   string    `json:"error"`
	} `json:"last_pulse"`
	Busy           bool `json:"busy"`
	Requests       int  `json:"requests"`
	Failures       int  `json:"failures"`
	IdentityPolicy bool `json:"identity_policy"`
}

func (c *Client) Status() Status {
	cl := &http.Client{Timeout: 3 * time.Second}
	resp, err := cl.Get(c.URL + "/api/model")
	if err != nil {
		return Status{}
	}
	defer resp.Body.Close()
	var s Status
	if resp.StatusCode != 200 || json.NewDecoder(resp.Body).Decode(&s) != nil {
		return Status{}
	}
	s.Reachable = true
	return s
}

// ---- thought cycles ----------------------------------------------------------------

type Action struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type CycleRequest struct {
	CycleID  string    `json:"cycle_id"`
	Identity string    `json:"identity"`
	Source   string    `json:"source"`
	Request  string    `json:"request"`
	Context  string    `json:"context"`
	History  []Message `json:"history,omitempty"`
	Actions  []Action  `json:"actions"`
	LessonChecks []int `json:"lesson_checks,omitempty"`
}

type Spark struct {
	Status     string   `json:"status"`
	EventID    string   `json:"event_id"`
	SparkID    string   `json:"spark_id"`
	Rating     int      `json:"rating"`
	Topic      string   `json:"topic"`
	Outcome    string   `json:"outcome"`
	TraceID    string   `json:"trace_id"`
	ThoughtID  string   `json:"thought_id"`
	Reason     string   `json:"reason"`
	RequestIDs []string `json:"model_request_ids"`
	ElapsedMS  int64    `json:"elapsed_ms"`
}

type Prediction struct {
	Text     string    `json:"text"`
	Value    *float64  `json:"value"`
	State    string    `json:"state"`
	LockedAt time.Time `json:"locked_at"`
	SHA256   string    `json:"sha256"`
}

type Proposal struct {
	ModelRequestID string     `json:"model_request_id"`
	Format         string     `json:"format"`
	Reply          string     `json:"reply"`
	Action         string     `json:"action"`
	ActionNote     string     `json:"action_note"`
	Why            string     `json:"why"`
	Remember       string     `json:"remember"`
	Prediction     Prediction `json:"prediction"`
	Reasoning      string     `json:"model_reasoning"`
	ElapsedMS      int64      `json:"elapsed_ms"`
}

type Outcome struct {
	Status     string    `json:"status"` // EXECUTED | NOT_AUTHORIZED | NO_ACTION | FAILED
	Value      *float64  `json:"value"`
	Detail     string    `json:"detail"`
	Executor   string    `json:"executor"`
	ExecutedAt time.Time `json:"executed_at"`
	Authorized string    `json:"authorized"`
	ReceivedAt time.Time `json:"received_at,omitempty"`
}

type Evaluation struct {
	Forecast    string    `json:"forecast"`
	Goal        string    `json:"goal"`
	Explanation string    `json:"explanation"`
	Evaluator   string    `json:"evaluator"`
	At          time.Time `json:"at"`
}

type Cycle struct {
	CycleID       string            `json:"cycle_id"`
	Identity      string            `json:"identity"`
	Request       string            `json:"request"`
	ReceivedAt    time.Time         `json:"received_at"`
	Authorization map[string]string `json:"authorization"`
	State         string            `json:"state"`
	Problem       string            `json:"problem"`
	Spark         Spark             `json:"spark"`
	Proposal      *Proposal         `json:"proposal"`
	Outcome       *Outcome          `json:"outcome"`
	Evaluation    *Evaluation       `json:"evaluation"`
}

func (c *Client) post(path string, in any, out any) (int, error) {
	body, _ := json.Marshal(in)
	req, _ := http.NewRequest("POST", c.URL+path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.Key)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return 0, fmt.Errorf("Thought Engine not reachable at %s (%v)", c.URL, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err := json.Unmarshal(raw, out); err != nil {
		return resp.StatusCode, fmt.Errorf("Thought Engine answered %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	return resp.StatusCode, nil
}

func errorOf(raw any) string {
	b, _ := json.Marshal(raw)
	var e struct {
		Error string `json:"error"`
	}
	json.Unmarshal(b, &e)
	return e.Error
}

// StartCycle runs the engine's cycle up to the locked prediction.
func (c *Client) StartCycle(in CycleRequest) (*Cycle, error) {
	var cy Cycle
	var raw map[string]any
	code, err := c.post("/api/cycles", in, &raw)
	if err != nil {
		return nil, err
	}
	b, _ := json.Marshal(raw)
	json.Unmarshal(b, &cy)
	if cy.CycleID == "" {
		return nil, fmt.Errorf("Thought Engine refused the cycle (%d): %s", code, errorOf(raw))
	}
	if code != 200 {
		return &cy, fmt.Errorf("%s", firstNonEmpty(cy.Problem, fmt.Sprintf("Thought Engine answered %d", code)))
	}
	return &cy, nil
}

// FinishCycle reports the executor's outcome; the engine evaluates the locked prediction.
func (c *Client) FinishCycle(id string, out Outcome) (*Cycle, error) {
	var raw map[string]any
	code, err := c.post("/api/cycles/"+id+"/outcome", out, &raw)
	if err != nil {
		return nil, err
	}
	var cy Cycle
	b, _ := json.Marshal(raw)
	json.Unmarshal(b, &cy)
	if code != 200 {
		return nil, fmt.Errorf("Thought Engine refused the outcome (%d): %s", code, errorOf(raw))
	}
	return &cy, nil
}

func firstNonEmpty(s ...string) string {
	for _, x := range s {
		if x != "" {
			return x
		}
	}
	return ""
}

// Job is work queued for the engine's coordination pulse.
type Job struct {
	JobID    string   `json:"job_id"`
	Identity string   `json:"identity"`
	Request  string   `json:"request"`
	Context  string   `json:"context,omitempty"`
	Actions  []Action `json:"actions,omitempty"`
	LessonChecks []int `json:"lesson_checks,omitempty"`
}

// SubmitJob queues a job; the pulse starts it on a later tick (never right away).
func (c *Client) SubmitJob(j Job) error {
	var raw map[string]any
	code, err := c.post("/api/pulse/jobs", j, &raw)
	if err != nil {
		return err
	}
	if code == 409 && strings.Contains(errorOf(raw), "already exists") {
		return fmt.Errorf("%w (409): %s", ErrJobExists, errorOf(raw))
	}
	if code != 202 {
		return fmt.Errorf("Thought Engine refused the job (%d): %s", code, errorOf(raw))
	}
	return nil
}

// ErrJobExists: the Engine already holds a job with this ID (its job store is
// durable). The caller must reconcile with that job, never resubmit the same ID.
var ErrJobExists = errors.New("Thought Engine already has a job with this ID")

// PulseControl sends start | pause | resume | step.
func (c *Client) PulseControl(action string) (map[string]any, error) {
	var raw map[string]any
	code, err := c.post("/api/pulse/control", map[string]string{"action": action}, &raw)
	if err != nil {
		return nil, err
	}
	if code != 200 {
		return raw, fmt.Errorf("Thought Engine refused %q (%d): %s", action, code, errorOf(raw))
	}
	return raw, nil
}

// Pulse returns the coordination pulse's status (control, state, ticks, jobs).
func (c *Client) Pulse() (map[string]any, error) {
	req, _ := http.NewRequest("GET", c.URL+"/api/pulse", nil)
	req.Header.Set("Authorization", "Bearer "+c.Key)
	hc := &http.Client{Timeout: 3 * time.Second}
	resp, err := hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var out map[string]any
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&out); err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("pulse status %d", resp.StatusCode)
	}
	return out, nil
}

// JobStatus is one pulse job as the engine reports it.
type JobStatus struct {
	JobID    string `json:"job_id"`
	State    string `json:"state"` // QUEUED | RUNNING | DONE | FAILED | INTERRUPTED | REFUSED
	CycleID  string `json:"cycle_id"`
	Result   string `json:"result"`
	Remember string `json:"remember"`
}

// PulseState is the pulse's control (RUNNING | PAUSED), state and jobs.
type PulseState struct {
	Control string      `json:"control"`
	State   string      `json:"state"`
	Jobs    []JobStatus `json:"jobs"`
}

// PulseStatus is Pulse() decoded.
func (c *Client) PulseStatus() (*PulseState, error) {
	raw, err := c.Pulse()
	if err != nil {
		return nil, err
	}
	b, _ := json.Marshal(raw)
	var ps PulseState
	if err := json.Unmarshal(b, &ps); err != nil {
		return nil, err
	}
	return &ps, nil
}
