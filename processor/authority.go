package main

import (
	"context"
	"errors"
)

// Protected Engine operations. Each is a SEPARATE scope: an approval for one
// never implies another. These are Processor's names for the operations it
// would ask about, not an agreed wire contract (none exists yet).
type ProtectedAction string

const (
	ActStart   ProtectedAction = "start"   // runner starts exactly one approved Engine instance
	ActAttach  ProtectedAction = "attach"  // operational attachment to an already-running instance
	ActStop    ProtectedAction = "stop"    // stop an Engine provably started by this session
	ActObserve ProtectedAction = "observe" // observation ingest — Engine-enforced grant, separate contract
)

var protectedActions = map[ProtectedAction]string{
	ActStart:   "Start the selected Thought Engine",
	ActAttach:  "Attach to a running Thought Engine",
	ActStop:    "Stop the Engine this session started",
	ActObserve: "Send an observation",
}

// Authority is the Processor-side port for asking Identity whether an
// authenticated operator/Processor may perform one protected action.
//
// IMPORTANT: even a real ALLOW returned here would not be enforcement. The
// component that executes the action (the bounded runner for start/stop; the
// Engine for attach and ingest) must independently validate the approval.
// This build contains NO runner, so start/stop are impossible regardless.
type Authority interface {
	// Available reports whether a real decision producer is configured.
	Available() (bool, string)
	Decide(ctx context.Context, action ProtectedAction) error
}

var (
	// ErrNoProducer: the Identity lifecycle-approval contract does not exist.
	ErrNoProducer = errors.New("no Identity start/attach/stop approval service exists yet")
	// ErrNoEnforcer: even with a decision, nothing on the executing side validates it.
	ErrNoEnforcer = errors.New("no runner or Engine control boundary that validates Identity approval exists yet")
	// ErrIngestSeparate: ingest is Engine-enforced under a different contract.
	ErrIngestSeparate = errors.New("observation ingest needs a separate Identity grant enforced by the Engine; no authenticated source is configured here")
)

// noProducer is the ONLY Authority in this build. There is deliberately no
// allow-all, developer-mode or local-checkbox implementation.
type noProducer struct{}

func (noProducer) Available() (bool, string) {
	return false, "Identity does not provide a start/attach/stop approval service. Inspected: Arameair/identity alpha, main and every experiment branch (see docs)."
}

func (noProducer) Decide(_ context.Context, a ProtectedAction) error {
	if a == ActObserve {
		return ErrIngestSeparate
	}
	return ErrNoProducer
}
