package main

import (
	"sync"
	"time"
)

// Flow records each protected request as the four moments the story asks to
// keep separate: requested → authorized (Identity) → executed (runner/Engine)
// → observed effect. A stage that never happened is shown as such; nothing is
// reconstructed or assumed atomic.
type Stage struct {
	Name   string     `json:"name"`
	Status string     `json:"status"` // DONE | REFUSED | NOT AVAILABLE | NOT ATTEMPTED | NONE
	At     *time.Time `json:"at,omitempty"`
	Owner  string     `json:"owner"`
	Detail string     `json:"detail"`
}

type Trace struct {
	OpID    string          `json:"op_id"`
	Action  ProtectedAction `json:"action"`
	At      time.Time       `json:"at"`
	Outcome string          `json:"outcome"`
	Stages  []Stage         `json:"stages"`
}

type Flow struct {
	mu     sync.Mutex
	traces []Trace
}

func (f *Flow) Add(t Trace) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.traces = append(f.traces, t)
	if len(f.traces) > 30 {
		f.traces = f.traces[len(f.traces)-30:]
	}
}

func (f *Flow) Recent() []Trace {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]Trace, len(f.traces))
	for i := range f.traces {
		out[len(f.traces)-1-i] = f.traces[i] // newest first
	}
	return out
}

func refusedTrace(opID string, act ProtectedAction, at time.Time, refusedAtBoundary bool, why string) Trace {
	t := at
	req := Stage{Name: "Requested", Status: "DONE", At: &t, Owner: "Operator → Processor", Detail: "signed-in operator asked Processor"}
	if refusedAtBoundary {
		return Trace{OpID: opID, Action: act, At: at, Outcome: "REFUSED", Stages: []Stage{req,
			{Name: "Authorized", Status: "NOT ATTEMPTED", Owner: "Identity", Detail: why},
			{Name: "Executed", Status: "NOT ATTEMPTED", Owner: "Runner / Engine", Detail: "request refused before any effect"},
			{Name: "Observed", Status: "NONE", Owner: "Engine", Detail: "nothing was sent to any service"}}}
	}
	owner := "Identity"
	exec := "no runner exists in this build"
	if act == ActObserve {
		exec = "Processor has no observation-sending path; sources send directly to the Engine"
	}
	if act == ActAttach {
		exec = "the Engine has no attach boundary that could validate an approval"
	}
	return Trace{OpID: opID, Action: act, At: at, Outcome: "BLOCKED", Stages: []Stage{req,
		{Name: "Authorized", Status: "NOT AVAILABLE", Owner: owner, Detail: why},
		{Name: "Executed", Status: "NOT ATTEMPTED", Owner: "Runner / Engine", Detail: exec},
		{Name: "Observed", Status: "NONE", Owner: "Engine", Detail: "no service was contacted, so there is no effect to observe"}}}
}
