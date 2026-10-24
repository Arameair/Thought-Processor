# Decisions, proposals, and unresolved questions

**This file separates owner direction from unproven implementation.** Notes from separate experimental branches are cited as evidence of the design conversation. Consolidation here does **not** mean code approval, Alpha promotion, or independent verification.

## Owner direction / selected design

| Decision | Current direction | Provenance |
|---|---|---|
| **Features are the product** | Independently usable, enabled and permitted; Thought Processing can consume **approved** logs from independently operating sources | Initial project discussions; [product direction](VISION.md) |
| **Thought Processor is a control point** | Real service discovery/connection and developer visibility, **not just a Thought viewer** and not a global router for unrelated operations | Initial discussions and [routing notes](DESIGN_HISTORY.md) |
| **Engine owns thinking** | Engine chooses when/how to process, when to call workers, heartbeat timing, Sparks/Traces/Thoughts and durable state | [Routing and heartbeat discussion](https://github.com/Arameair/Thought-Processor/blob/experiment/discovery-routing-devex-notes-20261023/docs/discussions/2026-10-23-routing-heartbeat.md) |
| **Direct Engine–worker calls** | Processor can help discover/establish a verified connection; after that Engine may call worker directly while exposing truthful telemetry | Same routing experiment; design only |
| **Models are replaceable** | Chat-Jimmy is the experimental fast worker, not a hard-coded product dependency | Owner discussion and current Engine experiment |
| **Dynamic application cards** | Cards represent actual independently running apps, including Identity as an *app*, not separate cards for its personas | [Dynamic feature cards](https://github.com/Arameair/Thought-Processor/blob/experiment/dynamic-feature-cards-notes-20261023/docs/discussions/2026-10-23-dynamic-project-cards.md) |
| **Option C selected for first UI** | **Lightweight live status cards + one expanded original app UI + collapsible Logs/Flow/Metrics inspector**; no browser per tile | [Option C owner-decision note](https://github.com/Arameair/Thought-Processor/blob/experiment/lightweight-live-cards-option-c-20261023/docs/discussions/2026-10-23-option-c-lightweight-status-cards.md) |
| **Human-directed changes** | Alpha starts experiments; Stable is an owner-accepted snapshot; no auto-promotion; AI involvement disclosed | [WORKFLOW](WORKFLOW.md) and [AI_COLLABORATION](AI_COLLABORATION.md) |

**Meaning of "selected":** the Option C note records an owner design preference. **No working implementation or formal Alpha promotion has followed from that choice.**

## Technical proposals — not yet approved contracts

- Compatible services may publish versioned discovery metadata: kind, instance ID, build SHA, readiness, capabilities, safe UI endpoint, diagnostics and allowed operations.
- Processor may maintain a lightweight registry / approved connection graph without forwarding all log events or model traffic.
- Engine and worker may expose correlation IDs for direct API calls; Processor displays observed telemetry and honestly marks missing stages.
- Each identity's Thought history should have **one authoritative Engine store**; data movement requires an explicit migration.
- Logs and prompts should be redacted in copied diagnostics by default; content is revealed only when authorized.
- High-volume inputs may need bounded batching/filtering before model calls. Neither frequency nor automatic filtering behavior is decided.

## Open decisions

| ID | Question | Why it matters |
|---|---|---|
| D-01 | Which Thought implementation is canonical, and what happens to the other store? | Avoid competing authoritative history |
| D-02 | How are feature grants, feed permissions and revocations authenticated across processes? | Preserve authority and privacy |
| D-03 | How do services register with Processor, and how are instances verified? | Enable dynamic cards without spoofed identity |
| D-04 | May an Engine use a previously verified worker if Processor is offline? | Reliability without removing the control point |
| D-05 | How are feeds chosen, filtered, retained and queued? | Control noise, storage and consent |
| D-06 | What is the versioned, model-neutral worker contract and safe retry policy? | Future model replacement |
| D-07 | When may the operator change a worker connection during a live Engine job? | Prevent in-flight corruption |
| D-08 | How do embedded original UIs handle authentication, preview lifecycle and framing limitations? | Secure Option C implementation |
| D-09 | How are diagnostic metadata and failed runs retained and protected? | Debuggability without overcollection |
| D-10 | Which Engine actions require owner approval, and how are they surfaced? | Truthful and safe control plane |

Close a decision with **date, explicit owner statement, alternatives, impact, issue/branch/commit, test evidence if technical, and rollback**. Never silently turn a proposal into a confirmed direction.
