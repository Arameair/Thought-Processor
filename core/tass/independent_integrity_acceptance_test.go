//go:build acceptance
// +build acceptance

package tass

// Independent contract gates against actual Go Core code. The Engine, Nemotron
// and GitHub are SCRIPTED FIXTURES; these tests do NOT prove real GPU inference.
// Deliberate RED tests expose boundaries that require engineering fixes.
// Run: cd core && go test -tags acceptance ./tass -run TestAcceptance -v

import (
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "strings"
    "sync"
    "testing"
    "time"

    "github.com/Arameair/Thought-Processor/core/engine"
    "github.com/Arameair/Thought-Processor/core/identity"
)

type boundaryFixture struct {
    srv *httptest.Server
    mu sync.Mutex
    outcomes []engine.Outcome
    cycleRequests []engine.CycleRequest
    replyCycleID string
    prediction engine.Prediction
    entered chan struct{}
    release chan struct{}
}

func newBoundaryFixture(t *testing.T, pred engine.Prediction, replyID string, block bool) *boundaryFixture {
    t.Helper()
    f := &boundaryFixture{
        prediction: pred, replyCycleID: replyID,
        entered: make(chan struct{}, 1),
    }
    if block {
        f.release = make(chan struct{})
    }
    f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        if r.URL.Path == "/api/model" && r.Method == http.MethodGet {
            _ = json.NewEncoder(w).Encode(map[string]any{
                "last_pulse": map[string]any{"model_ok": true, "at": time.Now().UTC()},
                "busy": false,
            })
            return
        }
        if r.Header.Get("Authorization") != "Bearer testkey" {
            http.Error(w, "fixture: missing engine bearer", http.StatusUnauthorized)
            return
        }
        if r.URL.Path == "/api/cycles" && r.Method == http.MethodPost {
            var in engine.CycleRequest
            if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
                http.Error(w, err.Error(), http.StatusBadRequest)
                return
            }
            f.mu.Lock()
            f.cycleRequests = append(f.cycleRequests, in)
            f.mu.Unlock()
            select { case f.entered <- struct{}{}: default: }
            if f.release != nil {
                select {
                case <-f.release:
                case <-r.Context().Done(): return
                }
            }
            id := in.CycleID
            if f.replyCycleID != "" { id = f.replyCycleID }
            _ = json.NewEncoder(w).Encode(engine.Cycle{
                CycleID: id, State: "awaiting_outcome",
                Proposal: &engine.Proposal{
                    Format: "READ", Reply: "SCRIPTED FIXTURE: proposed count",
                    Action: "tass_sections", Prediction: f.prediction,
                },
            })
            return
        }
        if strings.HasSuffix(r.URL.Path, "/outcome") && r.Method == http.MethodPost {
            var out engine.Outcome
            if err := json.NewDecoder(r.Body).Decode(&out); err != nil {
                http.Error(w, err.Error(), http.StatusBadRequest)
                return
            }
            f.mu.Lock()
            f.outcomes = append(f.outcomes, out)
            f.mu.Unlock()
            _ = json.NewEncoder(w).Encode(engine.Cycle{
                CycleID: strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/cycles/"), "/outcome"),
                State: "evaluated",
                Evaluation: &engine.Evaluation{Forecast: "UNSCORABLE", Goal: "GOAL_UNKNOWN", Evaluator: "fixture"},
            })
            return
        }
        http.NotFound(w, r)
    }))
    t.Cleanup(f.srv.Close)
    return f
}

func (f *boundaryFixture) result() []engine.Outcome {
    f.mu.Lock()
    defer f.mu.Unlock()
    return append([]engine.Outcome(nil), f.outcomes...)
}

func setupBoundaryCore(t *testing.T, f *boundaryFixture) *Core {
    t.Helper()
    gh := newFakeGH()
    t.Cleanup(gh.srv.Close)
    gh.set("Arameair/identity@main:policies/tass.json", policyJSON)
    gh.set("Arameair/TASS@live:tass.md", startDoc)
    c := newCore(t, t.TempDir(), gh, &fakeEngine{srv: f.srv})
    if wait := c.SyncOnce(); wait != 20*time.Second {
        t.Fatalf("fixture policy synchronization failed, retry: %s", wait)
    }
    if ok, why := c.allowedNow(identity.ReadTass); !ok {
        t.Fatalf("fixture action grant missing: %s", why)
    }
    return c
}

func lockedFixturePrediction() engine.Prediction {
    return engine.Prediction{
        State: "LOCKED", Text: "expected 3 sections",
        SHA256: strings.Repeat("a", 64),
        LockedAt: time.Now().UTC().Add(-time.Second),
    }
}

// FAIL at inspected Core 5188c26: an Engine 200 with Action != NONE and an
// UNLOCKED/absent prediction still causes an EXECUTED outcome.
// This is a red acceptance gate, not a test that should be weakened to pass.
func TestAcceptanceRejectsActionWithoutLockedPrediction(t *testing.T) {
    f := newBoundaryFixture(t, engine.Prediction{State: "NOT_PREDICTED"}, "", false)
    c := setupBoundaryCore(t, f)
    c.Answer(Incoming{Text: "count sections", From: "local"})
    outs := f.result()
    if len(outs) != 1 {
        t.Fatalf("expected a recorded non-execution outcome, got %d", len(outs))
    }
    if outs[0].Status == "EXECUTED" {
        t.Fatalf("FAIL P0: core executed %q with prediction state NOT_PREDICTED and no durable lock", "tass_sections")
    }
    if outs[0].Value != nil {
        t.Fatalf("unauthorized/unlocked action returned measured value %v", *outs[0].Value)
    }
}

// FAIL at inspected Core 5188c26: the Engine can return a cycle_id different
// from the ID the Core sent; the Core does not reject before action execution.
func TestAcceptanceRejectsEngineCycleIDMismatch(t *testing.T) {
    f := newBoundaryFixture(t, lockedFixturePrediction(), "different-cycle-id", false)
    c := setupBoundaryCore(t, f)
    c.Answer(Incoming{Text: "count sections", From: "local"})
    for _, out := range f.result() {
        if out.Status == "EXECUTED" {
            t.Fatal("FAIL P0: core executed an action for a cycle ID not matching its request")
        }
    }
}

// PASS regression: permission checked at point of action, after slow inference.
// This does not prove a live Identity authorize service (prototype policy only).
func TestAcceptanceRechecksRevokedActionGrantAfterModelWait(t *testing.T) {
    f := newBoundaryFixture(t, lockedFixturePrediction(), "", true)
    c := setupBoundaryCore(t, f)
    done := make(chan struct{})
    go func() {
        defer close(done)
        c.Answer(Incoming{Text: "count sections", From: "local"})
    }()
    select {
    case <-f.entered:
    case <-time.After(5 * time.Second):
        close(f.release)
        t.Fatal("fixture Engine did not receive the cycle")
    }
    c.mu.Lock()
    c.policy.Allow[identity.ReadTass] = false
    c.mu.Unlock()
    close(f.release)
    select {
    case <-done:
    case <-time.After(5 * time.Second):
        t.Fatal("Core did not finish after Engine released its response")
    }
    outs := f.result()
    if len(outs) != 1 || outs[0].Status != "NOT_AUTHORIZED" || outs[0].Value != nil {
        t.Fatalf("point-of-use revocation failed; outcomes=%+v", outs)
    }
}
