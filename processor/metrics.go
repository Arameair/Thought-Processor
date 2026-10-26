package main

import (
	"runtime"
	"sort"
	"sync"
	"time"
)

// Metrics counts only things Processor itself did or observed. There are no
// Engine, worker or Identity metrics here: Processor has no approved access
// to them, so it reports none rather than estimating.
type Metrics struct {
	mu        sync.Mutex
	probes    map[string]*probeStats // "identity" | "engine"
	guards    map[string]int         // rejected requests by reason
	refusals  map[string]int         // refused protected operations by action
	signIns   int
	signFails int
}

type probeStats struct {
	Checks     int            `json:"checks"`
	Outcomes   map[string]int `json:"outcomes"` // port result (+ metadata for Identity)
	recent     []int64        // last 50 elapsed ms
	LastChange time.Time      `json:"last_change_at"`
	lastKey    string
}

func NewMetrics() *Metrics {
	return &Metrics{probes: map[string]*probeStats{}, guards: map[string]int{}, refusals: map[string]int{}}
}

func (m *Metrics) Probe(target string, r *ProbeResult) {
	m.mu.Lock()
	defer m.mu.Unlock()
	ps := m.probes[target]
	if ps == nil {
		ps = &probeStats{Outcomes: map[string]int{}}
		m.probes[target] = ps
	}
	key := r.Port
	if r.Metadata != "NOT_REQUESTED" {
		key += "/" + r.Metadata
	}
	ps.Checks++
	ps.Outcomes[key]++
	ps.recent = append(ps.recent, r.ElapsedMS)
	if len(ps.recent) > 50 {
		ps.recent = ps.recent[len(ps.recent)-50:]
	}
	if key != ps.lastKey {
		ps.lastKey, ps.LastChange = key, r.CheckedAt
	}
}

func (m *Metrics) Guard(reason string) { m.mu.Lock(); m.guards[reason]++; m.mu.Unlock() }
func (m *Metrics) Refused(action string) {
	m.mu.Lock()
	m.refusals[action]++
	m.mu.Unlock()
}
func (m *Metrics) SignIn(ok bool) {
	m.mu.Lock()
	if ok {
		m.signIns++
	} else {
		m.signFails++
	}
	m.mu.Unlock()
}

func (m *Metrics) RefusedTotal() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := 0
	for _, v := range m.refusals {
		n += v
	}
	return n
}

func copyMap(in map[string]int) map[string]int {
	out := make(map[string]int, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func latency(samples []int64) map[string]any {
	if len(samples) == 0 {
		return map[string]any{"samples": 0}
	}
	s := append([]int64(nil), samples...)
	sort.Slice(s, func(i, j int) bool { return s[i] < s[j] })
	return map[string]any{"samples": len(s), "min_ms": s[0], "median_ms": s[len(s)/2], "max_ms": s[len(s)-1]}
}

func (m *Metrics) Snapshot(started time.Time, sessions int) map[string]any {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	m.mu.Lock()
	defer m.mu.Unlock()
	probes := map[string]any{}
	for k, ps := range m.probes {
		probes[k] = map[string]any{"checks": ps.Checks, "outcomes": copyMap(ps.Outcomes),
			"latency_last_50": latency(ps.recent), "last_change_at": ps.LastChange}
	}
	return map[string]any{
		"source":   "Processor's own counters since this run started; nothing here comes from Identity or the Engine",
		"since":    started,
		"uptime_s": int64(time.Since(started).Seconds()),
		"process": map[string]any{"goroutines": runtime.NumGoroutine(), "heap_alloc_bytes": ms.HeapAlloc,
			"sys_bytes": ms.Sys, "gc_cycles": ms.NumGC},
		"probes":               probes,
		"refused_operations":   copyMap(m.refusals),
		"rejected_requests":    copyMap(m.guards),
		"operator_sign_ins":    m.signIns,
		"operator_code_errors": m.signFails,
		"active_sessions":      sessions,
		"not_available": []string{"Engine queue/worker/throughput (needs approved attachment)",
			"Identity decisions (no decision service exists)", "worker/model calls (Processor never makes any)"},
	}
}
