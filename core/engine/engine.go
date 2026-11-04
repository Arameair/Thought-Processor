// Package engine is Thought-Processor's connection to the Thought Engine, which owns the
// Nemotron model: Processor → Engine → Nemotron → Engine → Processor.
package engine

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
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
