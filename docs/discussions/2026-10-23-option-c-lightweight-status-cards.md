# Decision — Option C: lightweight live status cards

**Date:** 2026-10-23  
**Decision status:** Owner selected **Option C as the initial UI approach**; Option D (adaptive hybrid) is a possible **later experiment**, not approved implementation now.  
**Branch:** `experiment/lightweight-live-cards-option-c-20261023`, created from `alpha` @ `9fac9fa44cb020bb6f8aa6bcb1e9faeda6e0b47a`.  
**Implementation status:** Documentation only. No live services, previews, status protocols, measurements, or integrations were implemented or tested in this change.  
**Provenance:** Human-directed decision, documented with AI assistance.

## Core choice

**Netflix/Prime-style catalog made of lightweight live status cards.** Each card represents an independently running compatible application (not just an abstract feature), such as the actual [Identity project](https://github.com/Arameair/identity), Thought Engine, or a replaceable fast worker.

A card displays **verified lightweight state** (health/readiness, recent activity, last check, errors, branch/build and perhaps key metrics). **Do not run a separate embedded browser preview in every collapsed tile.**

**When selected**, one card expands to show **the actual running application's original web UI** as the primary viewport, accompanied by a **collapsible side panel** for logs, control flow and metrics. If the UI cannot be embedded safely, clearly offer **Open Original UI** and a truthful diagnostic/status pane rather than a fake preview. Closing the viewport must not terminate the application.

### Distinguish these independently

- Project Controller: starts/stops/builds independent branches; provides Play/Stop and owner comparison.
- Identity: running Identity app and owner of entity records, permission/grants, repository associations and original Identity UI.
- Thought Processor: discovers compatible real processes, routes/displays authorized connection details and diagnostics, renders the dynamic card catalog; does not duplicate apps or their core state.
- Thought Engine: owns heartbeat API requests, their schedule/policy, model input and Thought history. Worker API calls may proceed directly after Processor-assisted discovery.
- Source/worker applications: remain independent even when their Processor card, inspector or preview is closed.

One actual Identity application card is **not** a separate card for each entity record (TASS, etc.); those records remain within the Identity app's UI.

## Initial home card: proposed read-only status contract (not an API yet)

- Application/service name and validated kind, stable runtime instance ID and capability/protocol version.
- Exact known source repository, branch, SHA/build if truthfully reported (else **unknown**).
- Connection state and *time last checked*, not stale status presented as live.
- Running/ready/busy/degraded/disconnected/unauthorized/incompatible distinct where applicable.
- Latest activity summary (bounded and safe), error count/last error and lightweight CPU/RAM/timing *only if supported*.
- Original UI/preview address if the app supports a compatible viewer and authorization.

**Do not infer a running process from a GitHub URL or from an open TCP port alone.** Discovery identity must be validated. Access to a status endpoint does not grant permission to view secrets or enable a feature. Never bypass browser framing/session restrictions for an embed.

## Expanded experience

```text
+---------------------------------------------------------+
| Thought Processor          [ selected application ]     |
+------------------------------------+--------------------+
|                                    | Inspector          |
| ACTUAL SELECTED APPLICATION UI     | [Logs|Flow|Metrics]|
|                                    |                    |
| No other hidden full UI previews   | Filter by service  |
| should be continuously rendered.   | Filter by request  |
|                                    | Copy Context       |
|                                    | Copy Logs          |
|                                    | Export metrics     |
+------------------------------------+--------------------+
```

Side inspector is collapsible; preview gets the space when closed. Render logs from authentic app output, tagged by component, run ID, timestamp, severity, correlation/request ID. Follow one real Engine → worker heartbeat in the flow view without requiring all model traffic to proxy through Processor.

## Diagnostic collection: keep evidence easy to grab

- **Copy Context:** compact, redacted snapshot: app/instance, branch+SHA, endpoint, run ID, checked-at time, discovered dependencies, connection stages, request IDs, last failure and relevant measured timings.
- **Copy Logs:** filter by app, run, severity, time window and request; copy a bounded excerpt with truncation status. Let user explicitly request a wider export.
- **Metrics:** real CPU/RAM, uptime, startup latency, worker call duration, heartbeat throughput/failures, queue depth if available; mark missing metrics **not measured**.
- **Cross-component trace:** tie Identity permission check → source observation → Engine heartbeat → direct worker response → Engine persistence with a correlation ID when available. Do not fabricate missing stages.
- Preserve source-owned logs; Processor may maintain a **small bounded index** of recent diagnostic metadata for navigation. Storage/retention/access rules need a separate decision. Do not silently copy entire prompts, private chat contents, tokens or secrets into the index.
- Each run gets a distinct ID; failed-run diagnostics should remain accessible as permitted after Stop/Play. Retention and log custody remain open, not implicit guarantees.

## Performance intentions — subject to measurement

**Initial Option C**: lightweight health/status polling or event subscription and **one active full preview at a time**. Each collapsed card has minimal UI overhead. Stopping/restarting an application updates the card state; opening a card does not launch its process. Independently running apps keep their own logs regardless of preview state.

Start by testing **Identity + one Thought Engine or worker**, then grow to 2 / 4 / 8 cards. Measure idle CPU/RAM, update rate, number of requests, time to expand/switch, browser memory, log loss and disconnected-state accuracy. Avoid presuming a polling frequency or caching threshold before test evidence.

**Possible later Option D:** retain one recently used preview or intelligently refresh visible cards if benchmarks justify it. This is **not part of Option C** and should live on its own experimental branch with comparison evidence. Prevent dormant preview frames and duplicate connections from accumulating.

## Discussion questions for the next session

1. How should compatible projects announce themselves — Project Controller registration, direct service discovery, or both? How do we verify identity?
2. What does a collapsed card's *recent activity* look like: one line, mini chart, or last event preview?
3. How should the right-side inspector handle quick log copying vs full historical exports?
4. Should Control Flow be a dedicated tab in the inspector or a separate expanded application/network view?
5. Where will a bounded cross-component diagnostic index live, and how long may data persist?
6. Should installed-but-disabled or stopped features remain visible, and where are experiments grouped?
7. What exact measurements would justify moving from C to D, rather than adding complexity prematurely?

## Acceptance gates (all currently NOT RUN)

- Live Identity process dynamically becomes a status card showing verified instance and accurate state.
- One selected card opens original Identity UI or a truthful fallback; no collapsed-card full preview remains running.
- Stopping Identity via Project Controller changes its card to STOPPED without stopping Engine or worker.
- A second independently started compatible feature appears without modifying hardcoded Processor tiles.
- Side inspector filters, copies and exports **actual** diagnostic data with reproducible request/run IDs.
- Measured overhead remains acceptable across card counts; no invented success labels.
- No silent Alpha, Stable or Main promotions.

## Related design discussions, not merged

- [Dynamic project cards](https://github.com/Arameair/Thought-Processor/blob/experiment/dynamic-feature-cards-notes-20261023/docs/discussions/2026-10-23-dynamic-project-cards.md)
- [Discovery, routing and Engine heartbeat](https://github.com/Arameair/Thought-Processor/blob/experiment/discovery-routing-devex-notes-20261023/docs/discussions/2026-10-23-routing-heartbeat.md)
- [Identity-side design notes](https://github.com/Arameair/identity/blob/experiment/dynamic-feature-cards-notes-20261023/docs/DYNAMIC_FEATURE_CARDS.md)

These documents live on separate experiments and are **references**, not changes already included in Alpha.
