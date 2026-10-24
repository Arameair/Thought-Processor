# Six owner-selected design decisions — Thought Processor

**Discussion date:** 2026-10-23  
**Status:** Owner-selected architecture/UI direction. **Documentation only; no implemented or verified integrations.**  
**Baseline:** `alpha` @ `9fac9fa44cb020bb6f8aa6bcb1e9faeda6e0b47a`.  
**Attribution:** AI-assisted notes from discussions led and choices made by the human owner.

## What we're building

A **small, independently usable Thought Processor** that connects and makes independently running features understandable. Identity and Thought Processor are both foundational components. Other projects (Thought Engine, memory, hearing/Discord, vision/screens, AI workers) can be built, tested and enabled **one feature at a time** without rebuilding the catalog.

**This is a reusable product**, not an extension requiring the owner's personal Project Controller. Developers/users may never have access to Project Controller. Each service must be runnable and inspectable independently.

Responsibility boundaries:

- **Identity**: identity/entity records, authentication, feature grants and the authority to approve access; the real running `Arameair/identity` application gets its own live-application card, *not* one decorative card per entity.
- **Thought Processor**: dynamic service discovery, authorized connection assistance, lightweight status cards, cross-service control-flow/diagnostic inspection; never silently takes over external service lifecycle or thought history.
- **Thought Engine**: observation ingestion, thought history, processing rules and **when/how to send heartbeats**. A heartbeat is an **actual API request to a fast AI worker**, not a Processor-owned timer.
- **Workers**: replaceable; Chat-Jimmy is the current speed experiment, not a permanent dependency. The Engine may call its selected worker directly after discovery/authorization rather than proxying every request through Processor.
- **Sources**: keep operating independently even if their Thought Processing feed is disabled or Processor is stopped.

## Six decisions explicitly selected by the owner

| # | Topic | Selected direction | Distinction / limit |
| --- | --- | --- | --- |
| **1** | **Discovery** | **Self-registration plus manual fallback.** Compatible apps announce themselves to Processor when available; an address can be supplied manually when they do not. Aim for seamless automatic discovery, reconnection and card creation. | No reliance on Project Controller. A discovered service must still be identified/verified; discovery is **not authorization**. Broader scanning, if any, is a separate experiment. |
| **2** | **Connection security** | **Automatically discover; Identity authorizes sensitive access.** Push authentication/grant decisions into Identity; each receiving service must enforce the valid grant. Once authorized, services can exchange data directly to preserve speed. | **Identity outage:** existing trusted connections may continue only for a **short, bounded authorization lifetime**; no newly privileged access. **Short duration chosen**, with ~5–15 minutes suggested for testing, **not a fixed implemented TTL**. Renew on recovery; pause protected operations on expiry. |
| **3** | **Logs and diagnostics** | **Distributed logs with shared traces.** Original services retain their own logs; use shared request/correlation IDs, bounded status/event metadata and on-demand collection in Processor. | Do not centralize all raw logs by default. Easy **Copy Context**, **Copy Logs** and measured metrics; redact secrets and sensitive source content. Preserve run IDs/failure evidence across a preview closing. |
| **4** | **Feature enablement** | **Automatic discovery followed by one-time, identity-scoped approval** for requested feature permissions. After approval, normal reconnects may be automatic while valid authorization exists. | A new permission, identity or expanded access scope requires fresh approval. **Installed**, **running**, **compatible**, **authorized** and **enabled** are different states. |
| **5** | **Control-flow visualization** | **Both** a clickable connection map and per-request event timeline **on demand**, in the collapsible inspector's **Flow** view. | Lightweight by default; show incomplete/unobserved spans honestly. Link failures to component, request ID, timestamps, worker result and persistence receipt when available. |
| **6** | **Version tracking** | **Display and compare version metadata plus recorded test evidence** for Alpha, Stable and experiments. Keep actual Git build/branch/version facts distinguishable from manual labels. | Processor is **not** an integrated build/deploy/promotion platform. Never auto-promote; no claim of “stable working” without owner-accepted reproducible evidence. |

## UI decisions already selected in prior discussion

1. **Netflix / Prime Video-style dynamic project cards.** New compatible running apps get cards without a Processor UI rewrite. The **Identity card represents the original running GitHub project**, not an avatar.
2. **Rendering Option C — lightweight live status cards.** Collapsed cards show truthful health/last-activity/branch where known, not many embedded browser sessions. Only the **selected** app opens its actual full UI in the main viewport (or an honest Open Original UI fallback).
3. **Collapsible right-hand developer inspector:** **Logs / Flow / Metrics** plus Copy Context and Copy Logs. One selected preview at a time.
4. **Rendering Option D — adaptive cached/recent previews** is a potential later experiment **only after benchmarks**; not selected for the first build.
5. Show **exact source, build SHA, instance/run ID, connection and trace details** where they exist; mark unsupported/unverified measurements explicitly rather than inventing them.
6. Avoid shutting down an app because its card is hidden or because Processor restarts; individual apps keep their original logs.

## Operational flow to test (proposed, not implemented)

```text
Developer starts Identity, Thought Processor and another feature independently
                         |
Compatible running feature self-registers (or operator adds verified address)
                         |
Processor identifies service + capabilities -> lightweight card appears
                         |
Owner chooses identity/feature -> Identity grants approved scoped access
                         |
Receiving service validates grant, runs independently
                         |
Thought Engine requests a fast worker (direct API heartbeat)
                         |
Services record local logs + shared trace IDs
                         |
Processor displays honest status; on expand, actual app UI + inspector
                         |
Developer copies bounded correlated diagnostics or compares version evidence
```

**No external Project Controller is required.** It may still be used by its owner to run experimental branches, but it is not the discovery authority or a product dependency.

## Important constraints and risks

- **Trust**: a port responding with a service name is not proof of identity; registration must eventually have authenticated instance identity and a safe discovery/preview protocol.
- **Identity outage/revocation**: short-lived access is a trade-off; access revoked just before an outage may last until its approved authorization expires. Do not issue new grants offline or extend expired ones.
- **Low overhead**: no constant hidden webviews for every tile, unbounded polling, unbounded log duplication, or expensive whole-history syncing.
- **Trace truthfulness**: no fake end-to-end evidence from a mere HTTP 200; separate running, discovered, connected, authorized, sent, completed, processed and persisted.
- **Isolation**: permissions and data are per identity and feature. Service discovery does not create an authorization bypass.
- **Lifecycle independence**: a card does not grant permission to terminate the application.
- **Existing experiments are not promoted** merely by adding these notes.

## Next design discussion: shared feature connection contract

Before implementing the card catalog, define a **small service-facing connection contract**. See [Connection contract — options to settle](2026-10-23-feature-connection-contract.md). Questions include service advertisement, instance verification, optional UI preview, status/metrics, diagnostic correlation, reconnect behavior and grant validation.

## Acceptance checks for the first real prototype — ALL NOT RUN

- Independent Identity app registers or is manually added; verified running instance appears as a status card.
- New compatible feature registers and appears **without a hard-coded tile**, on a machine without Project Controller.
- Disconnected/stopped/mismatched instances are shown accurately; no stale preview labeled live.
- Selected Identity card opens **the real Identity UI** (or an explicitly labeled safe fallback).
- One identity-scoped permission approval is actually enforced by the receiving service; new scope is not silently enabled.
- Engine sends a real worker heartbeat. Distributed logs share a correlation ID and Processor can fetch a bounded diagnostic context without raw private data leakage.
- Changing app branch/build or stopping/restarting one app preserves other apps and exposes exact version/test provenance where reported.
- Resource measurements for 2/4/8 cards establish whether rendering Option C is sufficient before exploring Option D.

**Owner decisions are about the desired design, not evidence that any test has passed.**
