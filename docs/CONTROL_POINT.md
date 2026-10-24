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
