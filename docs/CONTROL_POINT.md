# Operator experience — Option C

**Owner-selected initial UI direction (from an experimental design note), not yet implemented or owner-tested in Processor.**

## The interaction model

A **Netflix / Prime-style catalog of lightweight live status cards**. Each card describes one **verified real application instance**, such as Identity, Thought Engine, or the current worker service. New compatible apps can appear through discovery rather than a hard-coded change to Processor's home page.

Cards are lightweight **status views**, not continuously running embedded browsers. One selected card expands into the **actual original app UI** (if safe and supported), with a collapsible inspector for **Logs | Flow | Metrics**. If embedding isn't possible, offer **Open Original UI** and truthful status/diagnostics.

```text
Thought Processor                [Identity / Engine / Worker...]
┌──────────────────────────────────────────────────────────┐
│ Lightweight verified app cards                           │
│  Identity  ·  Thought Engine  ·  Current worker  ·  ...   │
├────────────────────────────────────┬─────────────────────┤
│ ONE selected application's         │ Inspector           │
│ actual running UI                  │ Logs / Flow /       │
│ or honest fallback                 │ Metrics             │
│                                    │ Copy Context        │
└────────────────────────────────────┴─────────────────────┘
```

*This layout is a concept, not a screenshot or statement of existing functionality.*

## What a collapsed card must tell the operator

- App and **verified service type**; connection and authorization are separate indicators.
- Endpoint, instance ID, protocol and reported build SHA/branch when available.
- Last checked time; ready/busy/degraded/disconnected/incompatible as separate meaningful states.
- One short recent-activity summary and last failure, or **not measured**.
- An explicit opening action for the original UI, only where the application advertises/supports it.

An **Identity card is the running Identity application**, not a tile for each Identity record (e.g. TASS).

## The selected app and inspector

1. **App:** show the original UI, not a reconstructed fake dashboard; preserve original sessions and framing/security restrictions.
2. **Logs:** filtered by app, run, severity, time, correlation ID; safe Copy Logs.
3. **Flow:** follow one Engine request from input sources, through worker call, to persisted Thought. Mark unseen steps **unknown** rather than filling blanks.
4. **Metrics:** real CPU/RAM/uptime/latency if available, requests and failures, queue length and trace timings. Missing values read **not measured**.
5. **Copy Context:** compact, redacted, reproducible evidence: process/instance, branch/SHA, port, selected app, checked time, request ID, error and owning component.

A **Connect** control should never automatically install, build or stop another application. Supported operations require authoritative acknowledgment and postcondition checks. The Processor does not set the Engine's heartbeat frequency or decide what a Thought should be.

## Performance and trust constraints

- One full preview **maximum** at a time; inactive cards use light status queries/events.
- Don't keep hidden browser sessions alive for collapsed tiles.
- Honor frame/session/auth restrictions. If a service can't embed, never imitate its UI or copy credentials.
- Distinguish **running**, **compatible**, **authorized**, **worker ready**, **request completed**, and **Thought persisted**.
- Preserve a bounded index of diagnostic metadata only if retention/security rules are explicitly decided; don't indiscriminately mirror raw chats or logs.
- Stop/Play independent services must not cause duplicated sessions, orphaned processes, or fake status.

## First verification scenario

1. Run **Identity** on a known branch through Project Controller.
2. Launch Processor independently and discover the exact running Identity instance. Confirm correct SHA/port/ID.
3. Select its card; display original UI or honest fallback and real diagnostics.
4. Stop Identity via Project Controller; card becomes STOPPED while Processor stays running.
5. Run a **Thought Engine + worker**, observe one actual Engine-owned request and show exact request ID/result; if telemetry is missing, show **unavailable**.
6. Measure CPU/RAM and switching responsiveness with 1, 2, 4 and 8 cards before considering more elaborate previews.

**All steps NOT RUN in this repository.** An adaptive hybrid preview is a *possible later* experiment, not part of the chosen initial design.
