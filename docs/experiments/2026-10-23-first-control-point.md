# Experiment 001 — first real Thought Processor control point

**State:** PLANNING / NOT CODED / NOT TESTED  
**Branch:** `experiment/processor-first-control-point-20261023`  
**Created from:** `alpha` @ `9fac9fa44cb020bb6f8aa6bcb1e9faeda6e0b47a` (documentation-only baseline)  
**No Alpha, Stable or Main changes are proposed until owner acceptance.**

## Objective

Build the *smallest* useful **real control point** for the optional Thought Processing feature. It must work independently, report real status, and not replace Thought Engine, Identity or Chat-Jimmy.

## Read before implementation

1. [Vision](../VISION.md) — features are separate products; the Processor is a control point.
2. [Architecture](../ARCHITECTURE.md) — Engine is the only authority for thoughts; Identity for grants; source features remain independent.
3. [Control-point UX](../CONTROL_POINT.md) — operational controls, evidence pipeline, diagnostics.
4. [Draft interfaces](../INTERFACES.md) — proposals, *not* existing APIs.
5. [Workflow](../WORKFLOW.md) — experiment → owner test → manual Alpha/Stable promotions.

## Minimal candidate: three independent running programs

**For the first experiment only:** use whichever of the two Thought implementations the owner explicitly selects (do not connect both stores); one source feed; one worker adapter (Chat-Jimmy may be used, but no permanent dependency).

1. Processor Play in Project Controller works without starting/stopping dependencies.
2. Discover Thought Engine by verified identity and protocol, not HTTP port alone. Show exact endpoint, build SHA, runtime state and reason for a mismatch.
3. Observe the Engine's actual identities, queue, observations and persisted Thoughts. Never fabricate a status.
4. Show what the Engine did for one **real owner-submitted observation/log** (receipt → queue → interpretation or rule → Spark/Trace/Thought → saved state).
5. Expose only supported, **engine-confirmed** controls such as Process Once; display accepted/rejected and verify actual state changed.
6. Show worker readiness, provider and mode separately from engine availability; report Chat-Jimmy as *current selection*, not mandatory vendor.
7. Preserve user intent: no covert worker sends, no source-app stop, no migration of existing Thought data, no default admin grants.
8. Keep visible Copy Diagnostics and run timings. Use a clear neutral state for unverified integration.

## Not in first candidate

- Generic router for all app actions.
- New storage engine or competing Thought database.
- Autonomous sweeping of every available log source.
- Paid API dependencies.
- Claims of authenticated Identity grants before an actual protocol exists.
- Auto-promote to Alpha, Stable or Main.
- Mocked user-facing outputs presented as real status.

## Demonstration checklist

| Test | Required observation | Status |
|---|---|---|
| Start standalone in Project Controller | Actual port/PID/version; clean Stop | NOT RUN |
| Engine stopped | NOT CONNECTED, no cached result labeled live | NOT RUN |
| Engine present | Verified protocol/build, actual identity list | NOT RUN |
| Wrong service on port | Refuse as NOT COMPATIBLE | NOT RUN |
| One real feed event | Original source/event ID preserved in Engine | NOT RUN |
| One worker/rule decision | Worker name/mode, prompt/reply or explicit rule label, timings | NOT RUN |
| Persisted Thought | Exact evidence/Thought ID survives Engine Stop/Play | NOT RUN |
| Permission denied | Engine rejects forbidden identity processing | NOT RUN |
| Feed disabled | Source application continues independently | NOT RUN |
| Busy/blocked/uncertain | No duplicate automatic sends; clear error | NOT RUN |
| Owner Windows/WSL test | Copy Context, logs and acceptance decision | NOT RUN |

## Next required decision

The existing experiments under `Arameair/Thought` and `Arameair/Thought-engine` are different implementations. **The owner must choose a canonical engine for the first Processor test**, with its exact branch/commit. Until then, build only independent control-plane discovery or design work that does not assume the canonical store.

## Evidence and promotion

After real testing, update this document with exact SHAs and evidence, plus [STATUS](../STATUS.md) on this branch. The owner alone decides whether to promote. All failures stay visible; the experiment must not silently replace Alpha.
