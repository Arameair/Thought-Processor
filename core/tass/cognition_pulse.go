package tass

// Cognition pulse strip (#82 P1): what the Engine's 20-second coordination pulse and the Core's
// 20-second sync tick are actually doing, read-only. Reading this never starts, steps or pauses
// anything: it is one authorized GET to the Engine plus the Core's own in-memory times.

import (
	"encoding/json"
	"time"

	"github.com/Arameair/Thought-Processor/core/engine"
)

type CognitionTick struct {
	PulseID string    `json:"pulse_id"`
	At      time.Time `json:"at"`
	Trigger string    `json:"trigger"` // timer | step
	State   string    `json:"state"`   // READY | WORKING | BUSY | BLOCKED
	JobID   string    `json:"job_id,omitempty"`
}

type CognitionPulse struct {
	ObservedAt    time.Time       `json:"observed_at"`
	Control       string          `json:"control"` // RUNNING | PAUSED | UNKNOWN (as the Engine reports it)
	EverySeconds  float64         `json:"every_s"` // configured; 0 = NOT RECORDED
	State         string          `json:"state"`
	ActiveJob     string          `json:"active_job,omitempty"`
	Queued        int             `json:"queued"`
	Running       int             `json:"running"`
	Last          *CognitionTick  `json:"last,omitempty"`
	Recent        []CognitionTick `json:"recent"`
	CoreTickEvery float64         `json:"core_tick_every_s"`
	CoreNextTick  time.Time       `json:"core_next_tick,omitempty"`
	Junctions     map[string]int  `json:"junctions"` // waiting work by status, for this identity
	Note          string          `json:"note"`
}

func pulseEnum(s string, allowed ...string) string {
	for _, a := range allowed {
		if s == a {
			return s
		}
	}
	return "NOT RECORDED"
}

// CognitionPulse: same Identity checks as every Cognition read, then a GET of /api/pulse.
func (c *Core) CognitionPulse(rq ...*CognitionReq) (CognitionPulse, error) {
	scope, err := c.cognitionAccess(rq...)
	if err != nil {
		return CognitionPulse{}, err
	}
	raw, err := c.engine.Pulse()
	if err != nil {
		return CognitionPulse{}, ErrCognitionUnavailable
	}
	var p struct {
		Control   string  `json:"control"`
		State     string  `json:"state"`
		Every     float64 `json:"every_s"`
		ActiveJob string  `json:"active_job"`
		Ticks     []struct {
			PulseID string    `json:"pulse_id"`
			At      time.Time `json:"at"`
			Trigger string    `json:"trigger"`
			State   string    `json:"state"`
			JobID   string    `json:"job_id"`
		} `json:"ticks"`
		Jobs []struct {
			State string `json:"state"`
		} `json:"jobs"`
	}
	b, _ := json.Marshal(raw)
	if json.Unmarshal(b, &p) != nil {
		return CognitionPulse{}, ErrCognitionUnavailable
	}
	out := CognitionPulse{ObservedAt: c.now().UTC(), Control: pulseEnum(p.Control, "RUNNING", "PAUSED"),
		State: pulseEnum(p.State, "READY", "WORKING", "BUSY", "BLOCKED", "UNKNOWN"), Recent: []CognitionTick{}, Junctions: map[string]int{},
		CoreTickEvery: c.cfg.SyncEvery.Seconds()}
	if out.Control == "NOT RECORDED" {
		out.Control = "UNKNOWN"
	}
	if p.Every > 0 && p.Every <= 3600 {
		out.EverySeconds = p.Every
	}
	if engine.CognitionID.MatchString(p.ActiveJob) {
		out.ActiveJob = p.ActiveJob
	}
	for _, j := range p.Jobs {
		switch j.State {
		case "QUEUED":
			out.Queued++
		case "RUNNING":
			out.Running++
		}
	}
	for _, t := range p.Ticks {
		if len(out.Recent) >= 10 {
			break
		}
		tk := CognitionTick{PulseID: cognitionRef(t.PulseID), At: t.At, Trigger: pulseEnum(t.Trigger, "timer", "step"),
			State: pulseEnum(t.State, "READY", "WORKING", "BUSY", "BLOCKED"), JobID: cognitionRef(t.JobID)}
		out.Recent = append(out.Recent, tk)
	}
	if len(out.Recent) > 0 {
		l := out.Recent[0]
		out.Last = &l
	}
	c.mu.Lock()
	out.CoreNextTick = c.nextSync
	c.mu.Unlock()
	c.junMu.Lock()
	for _, r := range c.loadJunctionsLocked().Receipts {
		if r.IdentityEntityID == scope.identityID && (r.Status == "PENDING" || r.Status == "AWAITING_APPROVAL" || r.Status == "RUNNING") {
			out.Junctions[r.Status]++
		}
	}
	c.junMu.Unlock()
	switch {
	case out.Control == "PAUSED":
		out.Note = "The Engine pulse is PAUSED: no 20-second work ticks are happening. Opening Cognition does not start it."
	case out.Last == nil:
		out.Note = "No pulse tick has been recorded yet."
	default:
		out.Note = "Recorded by the Engine; this page only reads it."
	}
	return out, nil
}
