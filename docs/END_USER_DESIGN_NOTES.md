# End-user design notes for Thought Processor

*(captured while the design is still open — October 2026)*

These notes come from an outside-user perspective. They are not requirements or decisions; they are observations about what would make Thought Processor feel clear, honest, and usable once it exists.

## Role reminder

Thought Processor is the control point for the optional Thought Processing feature. It is not the place thoughts are stored, not the engine, and not the identity authority. It should let the user see and operate the real pipeline without inventing state.

## What the user needs to understand immediately

On open, the user should be able to answer these questions without hunting:

- Which identity is this for, and is the feature actually granted?
- Is a compatible engine running and reachable?
- Is a real worker ready (not mock, not missing)?
- Which feeds are authorized and currently being consumed?
- What is the current pipeline status (waiting / processing / blocked / failed) and why?

If any of those cannot yet be answered because the corresponding piece does not exist or is not connected, say so plainly. Do not collapse them into a single “ready” or “online” indicator.

## Honesty rules (non-negotiable)

- Never show a green or “connected” state unless the authoritative service just confirmed it.
- Distinguish clearly:
  - Engine not running
  - Engine running but worker not ready / mock refused
  - Worker ready and processing
  - Feature grant missing or held only locally by the engine
  - Feed paused vs. source stopped
- If an observation is waiting in the queue, show the reason (“Chat-Jimmy not ready,” “processing paused,” etc.).
- Buttons that trigger actions must be disabled or clearly marked unavailable until the engine reports the action is possible.
- Every action must show requested → accepted/refused → observed effect. An HTTP 200 from the UI is not proof the engine did the work.

## Frictionless path from Identity

The user should be able to arrive from Identity and immediately see:

1. The selected identity and its permanent ID.
2. The real grant status (or the honest label “permission currently held by engine — Identity grant not yet connected”).
3. Live status of the engine and worker.
4. A short, factual next step if something is missing (“Start Chat-Jimmy without -mock,” “Start Thought-engine,” etc.).

Processor should not require the user to re-enter the identity or hunt for ports if discovery can surface the real state.

## Feeds and pipeline

- Start with the ability to select and pause individual permitted feeds. Pausing a feed must not stop the source application.
- Show the evidence path with the actual reason at each hop:
  - received → queued → rated → kept or discarded → Thought created/updated → saved
- Make it possible to inspect one observation end-to-end (event ID, source, rating, topic, resulting Trace/Thought, worker reply, any failure).
- Do not invent sample data in the live view. If nothing has been processed yet, say so.

## Workers and operations

- Treat the worker as replaceable. Show capability readiness, not a hard-coded product name as a permanent requirement.
- Surface mock mode explicitly and refuse to process with it (matching the engine’s current behavior).
- Provide Process once, Pause/Resume, and clear diagnostics (queue length, last error, worker status, timings).
- Uncertain or already-sent requests must not be automatically resent.

## Cleanliness and diagnostics

- Show the engine’s data folder path and the Processor’s own data location (if any) so the user knows exactly what is being written.
- Copy Diagnostics / Copy Context should default to omitting observation text, prompts, and secrets, with an explicit opt-in to include content.
- Stopping Processor must not stop the engine, the worker, or any source. Stopping the engine must not delete saved thoughts.

## What to defer until it can be real

- Automatic process startup or takeover of the engine or worker.
- Claims that a grant is enforced by Identity before the handshake exists and is verified.
- Multiple concurrent engines or workers presented as a single seamless system.
- Any UI that implies end-to-end authorization or feed wiring that has not been demonstrated on the user’s machine.

The user should always be able to see what is actually happening, why an observation is waiting or discarded, and what still needs to be started or connected. No hidden state, no simulated success.
