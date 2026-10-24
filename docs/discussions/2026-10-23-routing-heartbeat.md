# Discussion record — discovery, heartbeat and observability (2026-10-23)

**Status:** Architecture direction discussed with owner; documentation only; no API or runtime capability verified.  
**Based on:** `alpha` at `9fac9fa44cb020bb6f8aa6bcb1e9faeda6e0b47a`.  
**Attribution:** Human-directed discussion with AI-assisted synthesis/documentation. This is not a claim of implemented software.

## Clarified owner direction

1. **Thought Processor should be simple.** Treat it as the Thought Processing feature's **control point / router for discovering, connecting and observing components**, not as the machine that decides when or how to think.
2. **Thought Engine owns thinking behavior**: what context to gather, how to use observations, when to call a model, scheduling/rate behavior, what to retain as Sparks/Traces/Thoughts, and its own state/history. Do **not** put heartbeat frequency or thought-creation rules in Processor.
3. **A heartbeat means an API request to a fast AI worker.** It is not merely an independent timer in Processor. Chat-Jimmy is the current experimental fast worker only; no permanent coupling.
4. **Find workers through the control point.** The Engine should discover/select a compatible worker via the Processor-controlled discovery/connection mechanism. After a connection is established, **Engine ↔ worker API calls can be direct**, avoiding unnecessary forwarding through the Processor.
5. **Developers must see what is happening.** The controller view should explain what the Engine chose, where a request went, whether a worker responded, what changed, and where an error occurred, **even when the call is direct**. The Processor should expose enough evidence and diagnostics for someone watching from Project Controller or Thought Processor to debug it.
6. **Keep features independently operable.** Identity owns grants; sources provide explicitly approved feeds; Thought Engine owns thought processing/history. The Processor must not become a generic router for every application or auto-start/stop unrelated apps.

These are architectural intentions, **not observed or tested behavior**.

## Tentative communication paths (not a specified protocol)

```text
Identity (authorizes Thought Processing and allowed feeds)
                    |
                    v
Thought Processor — service discovery + connection control + diagnostics
    |                  |                         ^
    | finds compatible | publishes connection    | engine status and
    | worker endpoint  | info to Engine          | traceable activity
    v                  v                         |
 Worker registry    Thought Engine -------------+
                       |  owns heartbeat requests
 Approved source feeds |  owns processing and persistence
             --------->|
                       +--------------------> Fast worker (Chat-Jimmy now)
                       <--------------------+ Direct API result
```

**Important distinction:** Routing/discovery through the Processor does *not* require proxying every byte of model traffic. The Engine can call the chosen worker directly, while making request lifecycle and result metadata inspectable in the Processor. Whether any actual feed messages should pass through Processor is still open; do not make Processor a mandatory data-path bottleneck without a test.

## What one heartbeat should make inspectable

A single correlation/request ID should let a developer inspect an **actual** chain, with honest unavailable fields:

| Stage | Questions the operator can answer |
| --- | --- |
| Source/context | Which authorized feed(s), event IDs and bounded input context were considered? |
| Engine decision | Why did the Engine request this worker capability now? Which policy/version? |
| Discovery/connection | Which worker was selected? How was its identity/capability validated? What endpoint/build? |
| API heartbeat | When was the actual API call submitted? Was it busy, rejected, completed, blocked, timed out or uncertain? |
| Result | Response status and measured duration; sanitized input/output sample when authorized |
| Engine consequence | Did a Spark, Trace, Thought or queue item change? What durable record ID proves it? |
| Diagnostic evidence | Request/correlation ID, timestamps, branch+SHA, runtime instance/port, errors and safe log reference |

**A dashboard green light alone is never proof of completion.** Separate service running, endpoint discovered, connection authorized, worker ready, API request sent, response received, and thought persisted.

## Failure isolation to preserve

- The worker is offline: show **WORKER UNAVAILABLE**, preserve Engine/source state, and avoid fictitious responses.
- Engine calls a worker directly: Processor visibility must come from reliable engine telemetry/trace reports or a read API, not from pretending it intercepted traffic.
- Processor unavailable after an Engine–worker connection: whether the Engine may continue using a previously verified endpoint, and for how long, is **OPEN**. Do not quietly decide in code.
- Worker endpoint changes: revalidate identity/capabilities; don't silently attach to an unrelated process on the same port.
- Feed or Thought Processing disabled: do not stop the source application or delete its logs/history merely as a side effect.
- Sensitive prompts/logs: default diagnostics should expose metadata, with content displayed only after authorization and redaction.

## Explicitly not decided yet

- Whether an operator can switch the live worker during an active Engine session; which component owns the switch handshake.
- Whether discovery lives entirely inside the Processor or uses a separate registry managed by it.
- Worker caching/reconnect behavior if Processor is offline; persistent vs best-effort trace delivery.
- Whether a model request must be emitted on *every* cycle even if observations haven't changed; the Engine should answer through experiments.
- UI layout and whether its primary workflow is **overview-first** or **debug trace-first** (see [UI/DevEx discussion](2026-10-23-ui-devex.md)).
- Precise worker contract, metrics, security model, heartbeat rate and state transitions.

**Next:** Discuss UI and developer experience with owner, *before* adding implementation or claiming these design proposals are tested.
