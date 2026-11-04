// Package identitysvc is the core's client for tass-identity, the installation's Identity service.
// Identity is the only authority: every "may I?" is asked at the point of use, and anything other
// than a clear "allow" is a denial.
package identitysvc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type Client struct {
	URL  string
	Key  string
	HTTP *http.Client
}

func New(url, keyFile string) (*Client, error) {
	b, err := os.ReadFile(keyFile)
	if err != nil {
		return nil, fmt.Errorf("Identity service key: %w", err)
	}
	return &Client{URL: strings.TrimRight(url, "/"), Key: strings.TrimSpace(string(b)), HTTP: &http.Client{Timeout: 5 * time.Second}}, nil
}

type Entity struct {
	EntityID  string    `json:"entity_id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	Grants    []string  `json:"grants"`
}

// State is Identity's public state: NO_ENTITY, LOCKED or UNLOCKED (UNKNOWN when it can't be reached).
type State struct {
	State         string     `json:"state"`
	Reason        string     `json:"reason"`
	Entity        *Entity    `json:"entity"`
	UnlockedUntil *time.Time `json:"unlocked_until"`
}

func (c *Client) State() State {
	resp, err := c.HTTP.Get(c.URL + "/v1/state")
	if err != nil {
		return State{State: "UNKNOWN", Reason: "Identity not reachable: " + err.Error()}
	}
	defer resp.Body.Close()
	var s State
	if resp.StatusCode != 200 || json.NewDecoder(resp.Body).Decode(&s) != nil {
		return State{State: "UNKNOWN", Reason: fmt.Sprintf("Identity answered %d", resp.StatusCode)}
	}
	return s
}

type Decision struct {
	Allow      bool   `json:"allow"`
	DecisionID string `json:"decision_id"`
	Reason     string `json:"reason"`
	EntityID   string `json:"entity_id"`
	EntityName string `json:"entity_name"`
}

// Authorize asks Identity whether this service may do action now.
func (c *Client) Authorize(action, detail string) Decision {
	body, _ := json.Marshal(map[string]string{"action": action, "detail": detail})
	req, _ := http.NewRequest("POST", c.URL+"/v1/authorize", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.Key)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return Decision{Reason: "Identity not reachable: " + err.Error()}
	}
	defer resp.Body.Close()
	var d Decision
	if resp.StatusCode != 200 || json.NewDecoder(resp.Body).Decode(&d) != nil {
		return Decision{Reason: fmt.Sprintf("Identity answered %d", resp.StatusCode)}
	}
	return d
}

// ApprovedLesson is a read-only, bounded reference supplied by Identity.
// The GitHub access token never crosses into Core; this request uses only the
// existing Identity Core service key. All failures deny reference retrieval.
type ApprovedLesson struct {
	Skill      string `json:"skill"`
	Repository string `json:"repository"`
	Issue      int    `json:"issue"`
	CommentID  int64  `json:"comment_id"`
	Source     string `json:"source"`
	Mode       string `json:"mode"`
	Body       string `json:"body"`
}

func (c *Client) GetApprovedLesson() (ApprovedLesson, error) {
	var lesson ApprovedLesson
	req, err := http.NewRequest(http.MethodGet, c.URL+"/v1/github/lesson", nil)
	if err != nil {
		return lesson, err
	}
	req.Header.Set("Authorization", "Bearer "+c.Key)
	client := *c.HTTP
	if client.Timeout < 20*time.Second {
		client.Timeout = 20 * time.Second
	}
	resp, err := client.Do(req)
	if err != nil {
		return lesson, fmt.Errorf("Identity lesson endpoint unavailable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return lesson, fmt.Errorf("Identity has not approved a readable lesson (HTTP %d)", resp.StatusCode)
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 12<<10)).Decode(&lesson); err != nil {
		return ApprovedLesson{}, fmt.Errorf("invalid Identity lesson response")
	}
	if lesson.Skill != "@identity-test" || lesson.Repository != "Arameair/Thought-Processor" ||
		lesson.Issue != 51 || lesson.CommentID != 6636766875 ||
		lesson.Source != "https://github.com/Arameair/Thought-Processor/issues/51#issuecomment-6636766875" ||
		lesson.Mode != "reference_only" || len(lesson.Body) == 0 || len(lesson.Body) > 8192 {
		return ApprovedLesson{}, fmt.Errorf("Identity lesson provenance did not match")
	}
	return lesson, nil
}
