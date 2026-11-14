> **Earlier wireframe — not the current first-screen specification (historical October 26 design).** P0 is now runnable Alpha and P1 has an unpromoted deterministic-fixture implementation; this remains a conceptual wireframe, not live functionality. See [current status](STATUS.md). This page illustrates the diagnostic/Engine-specific controls considered before the owner's system-wide Processor direction. The starting UX is being reviewed at [Issue #15](https://github.com/Arameair/Thought-Processor/issues/15); see [Design readiness](DESIGN_READINESS_2026-10-26.md). All numbers in the old wireframe are placeholders.

---

# Thought Processor — operator control surface

**Wireframe and acceptance requirements — not a implemented UI.**

```text
┌───────────────────────────────────────────────────────────────┐
│ Thought Processor   TASS [identity]   FEATURE: enabled        │
│ Engine: connected · real protocol · pinned identity/build     │
├───────────────────────────────────────────────────────────────┤
│ FEEDS            │ ENGINE / WORKERS         │ OPERATIONS       │
│ TASS logs   ON   │ Queue 12 waiting         │ Process once     │
│ Screen      OFF  │ 2 accepted / 1 rejected  │ Pause / Resume   │
│ Discord     OFF  │ Fast model: Chat-Jimmy   │ Retry uncertain? │
│                 │ Deep model: NOT SET      │ (manual only)    │
├───────────────────────────────────────────────────────────────┤
│ EVIDENCE PIPELINE: received -> queued -> rated -> kept -> saved│
│ OBSERVATION INSPECTOR · TRACE/THOUGHT LINK · REQUEST/REPLY     │
├───────────────────────────────────────────────────────────────┤
│ LOGS / TIMINGS / ERRORS / BUILD SHA / SOURCE STATUS            │
└───────────────────────────────────────────────────────────────┘
```

The above values are illustrative **design placeholders**, not metrics or screenshots of real software. Production UI must read actual values.

## First real operator test (owner-selected next integration)

Before expanding this illustrative wireframe into a full application, implement the smallest **Identity + Engine authorization test through Processor**, as planned in [INTEGRATION_AUTHORIZATION_PLAN.md](INTEGRATION_AUTHORIZATION_PLAN.md) and [issue #8](https://github.com/Arameair/Thought-Processor/issues/8).

The operator should see: authenticated Identity service; authenticated Engine service; selected entity and explicitly identified test source; an administrator-authenticated request to Identity to grant/revoke one operation; a deliberately submitted test observation; and actual Identity decision + Engine receipt side by side, with a shared correlation ID. **A GUI checkbox must never set an Engine-local "authorized" flag and imply Identity approval.** Controls stay unavailable until both authoritative services implement the needed endpoints. Do not impersonate TASS using a JSON source field.

For this first integration, show **AUTHORITY UNAVAILABLE** on Identity outage and verify the Engine fails closed without a new model call. Recheck queued events at worker dispatch. Already-sent model data cannot be retrospectively removed after a grant is revoked.

This is a **historical proposed UI slice**. The owner has since promoted runnable Processor P0 Alpha, and an unpromoted P1 fixture demonstrates limited Identity grants. Actual Identity↔Thought Engine enforcement is still unproven; do not treat this drawing as a live UI or its placeholder values as evidence.

## Required panels

### 1. Identity and feature grant
Identity name and immutable ID; permission authority and last verification time; feature installed/enabled/disabled and reason; engine installation/build and capability match. Show a distinct **UNVERIFIED**, **PERMISSION DENIED**, **NOT CONNECTED**, **DEGRADED** or **READY**, not a single misleading "online" badge.

### 2. Source subscriptions
List available sources, explicitly authorized for this identity, with connection/readiness, source version, last received observation, rate, lag, dropped count, failure reason. Start with a single real feed; added integrations must use the same observability standard. Toggling a feed affects only the Thought Processing subscription.

### 3. Processing operations
Engine status, waiting queue and backlog, current job, last success/error, processing budget, pause/resume, **Process once**, and explicit manual retry where safe. Calls to a worker that might already have been sent must be classified **UNCERTAIN** and never automatically resent.

### 4. Workers
Show capability selection (e.g. relevance scoring / topic matching / develop thought), selected provider, readiness, actual/mock mode, active request and request ID, response source, first text and completion timings, rate limits, timeout, model/worker version. A model is a replaceable provider; don't name a universal setting "Chat-Jimmy required".

### 5. Evidence inspector
One observation's full path:
- source ID and event ID with receipt timestamp and any redaction;
- accepted/rejected/queued/processed status and rule;
- request ID and input sent to the model (owner-controlled disclosure);
- model raw reply (separately labeled), parsing and validation outcome;
- resulting Spark, Trace, Thought and saved receipt/commit outcome;
- errors and retry safety.

### 6. Diagnostics
**Copy Context / Copy Logs / Metrics** as first-class actions. Include branch + exact SHA, port, process PID, uptime, source address, installation ID, identity ID (where safe), checked timestamps, queue length, ingress/processing latency, errors, memory/CPU when available. Copy Context should default to omitting log body, chat prompts and secrets, with explicit opt-in to include content.

## Control point is NOT a simulation screen

Buttons only appear enabled when the authoritative engine reports the capability and permission. Every action must show **requested → accepted/refused → observed effect**. No fake samples in a live state. Experimental fixtures belong in tests and clearly segregated developer tools.

## Useful operational acceptance tests

- Stop Engine: Processor switches to **NOT CONNECTED** and never displays cached thoughts as live.
- Start Engine on another port: discovery reports supported service, correct protocol and owner-selectable binding.
- Grant disabled in Identity: processing and ingestion blocked, with reason; old thoughts remain.
- Source disconnected: feature stops consuming that feed but original source stays running.
- Worker busy/blocked/uncertain: queue and status reflect it, no silent duplicate messages.
- Stop/Play across branches: records persist only where explicitly configured; do not confuse test data with Alpha data.
