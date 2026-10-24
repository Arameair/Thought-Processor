# First Experience — small runnable demonstration of the entire idea

**Design / implementation handoff — 2026-10-24**  
**Owner request:** Experience what the product could feel like by **starting Identity first**, which helps bring up the **correct Thought Processor** and a minimal set of connected features. Show real behavior using a few simple observations that stand in for the larger ecosystem.  
**Status:** PLANNED ONLY. Thought Processor is still documentation-only in its current release lanes; this document is **not** a running application, an owner acceptance or a validated grant protocol.  
**Implementation tracker:** [Processor Issue #9](https://github.com/Arameair/Thought-Processor/issues/9).  
**Identity companion:** [Identity Issue #16](https://github.com/Arameair/identity/issues/16) and [launch brief](https://github.com/Arameair/identity/blob/experiment/first-run-experience-brief-20261024/docs/FIRST_EXPERIENCE_LAUNCH_2026-10-24.md).  
**Build policy:** implement on an **independent new experiment from owner-approved `alpha`**, show exact SHA and measurements; do not touch Alpha/Stable/Main or merge automatically.  
**Authorship:** owner-directed design with AI assistance.

## The feeling we want to test

A new user opens **Identity**, chooses **Start Experience**, and sees an **actual** Thought Processor session that has found the right Identity service, one Engine and a small test source. They can submit a simple observation, follow the real permission and processing stages, and shut down or revoke a connection to see the system respond.

It should feel like **several independent products cooperating**, not three unrelated configuration dashboards, a static mockup, or a web server pretending to be a complete companion.

> **Identity owns the organization and authority of the system. Each feature owns its actual work.**
>
> **Identity decides. Thought Engine enforces. Thought Processor connects, tests, and explains.**

### Deliberate bootstrap refinement

Previous plans said Identity should not automatically launch independent services simply because its dashboard opens. **Keep that default.** The owner's new request is an **explicit, consented, bounded demo launcher** initiated from Identity, with a specifically selected Processor build and supporting demo profile. It is not a permanent change to the responsibility for normal feature work, nor a requirement to keep Processor open while Engine runs.

## One small visible journey

```text
Identity launches by itself
   -> IDENTITY READY (healthy with zero optional features)
   -> [Start Experience] (user approval and exact selected builds)
   -> Start/attach actual Thought Processor candidate
   -> Processor workspace opens:
         Identity      real identity authority + grant
         Engine        real observation intake / queue / trace
         Demo source   one real authenticated source service S
         Chat-Jimmy    optional real worker; unavailable stays unavailable

   -> [Approve source S for entity X]       Identity issues decision
   -> [Try an observation]                  S submits directly to Engine
   -> Processor timeline: request -> decision -> 202 queued
                        -> worker call if available -> real output
   -> [Revoke] and [Try again]              Engine 403, no worker call
   -> [Stop Experience]                     stop only session-owned processes
```

**The key distinction:** Processor starts a test **through a separately identified and authenticated Demo Source**. Processor is not allowed to claim that a `source: "TASS"` JSON field proves that TASS sent it.

## Intended minimal UI layout — a guide, not a static UI to ship

```text
 THOUGHT PROCESSOR     Experience: Local demo #<run_id>
 Identity VERIFIED | Engine CONNECTED | Demo Source AUTHENTICATED | Worker ...

 [Selected entity: Test Companion X]   [Test profile / exact branch + SHA]

 1  CONNECT                         2  PERMISSION
    Identity: verified                 Grant: NONE / GRANTED / REVOKED
    Thought Engine: ready              Source S -> Engine E for entity X
    Demo Source S: ready               [Request approval] [Revoke]
    Chat-Jimmy: ready/blocked

 3  TRY AN OBSERVATION              4  WHAT ACTUALLY HAPPENED
    [Player visited Goldshire.]        Source S -> Engine -> Identity
    [Send one]                         decision / event receipt / queue
                                       -> worker -> Spark/Trace/Thought
                                       [View original result] [Copy Context]

 [Flow] [Logs] [Metrics] [Open original application UI] [Stop Demo]
```

Collapsed service cards should show **actual** process/port/PID/run ID/branch/SHA (or `UNAVAILABLE/UNKNOWN`). Only open one real app UI/preview at a time. Logs and Flow inspectors reveal *why* an operation failed, not just a color. An Engine's local `grant.Enabled` is **NOT** the Identity-authorized state.

## The intentionally tiny feature set

### One test identity, four runtime participants

| Participant | Role and source of truth | Minimal capability |
| --- | --- | --- |
| **Identity I** | Durable entity, service IDs, grant and signed decision | Issue/revoke scope, return authenticated decision/trace |
| **Thought Processor P** | Owner session display and *requested* commands | Connect, trigger a source test, correlate status/receipts |
| **Thought Engine E** | Observation enforcement/queue/worker output | Current [grant experiment](https://github.com/Arameair/Thought-engine/tree/experiment/identity-grant-enforcement-20261024) |
| **Demo Observation Source S** | Dedicated *test-only source service*, not TASS | Send one authenticated event directly to Engine |
| **Chat-Jimmy W (optional)** | Actual external independently running worker | Respond to Engine worker request if genuinely READY |

An **entity** X (example “Test Companion”) is *not* itself Source S, Process E or an API credential. Make the distinction visible at least in diagnostics.

### Example event sequence

Start with a single example (e.g. **“The player visited Goldshire.”**). When that is a verified real event, offer up to two related follow-ups:

- “The player returned to Goldshire later.”
- “The player visited Goldshire again and paused near the inn.”

The user presses each button deliberately. The Engine's actual model decides importance/topic; existing rules determine when a Trace or Thought forms. A Thought **might not form**; the UI must show the actual rule, rating, missing evidence and how many model calls occurred. No random or prewritten model output mislabeled as Chat-Jimmy. If no real worker is ready, the event can wait **after genuine Identity authorization**; show `QUEUED — WORKER UNAVAILABLE`. In a separate automated test fixture, label `TEST FIXTURE` prominently.

### Real evidence view

Every test operation should show a coherent, timestamped chain:

```text
Session ID + request ID
S: sender authenticated [verified source service ID]
E: received event_id, entity_id, build SHA and port
I: decision = ALLOW / DENY / UNAVAILABLE
I: decision_id, grant_id, grant_version, requested action
E: receipt = 202 queued / 403 refused / 503 unavailable
W: worker ready? invoked? request_id? real vs fixture?
E: event processed? Spark / Trace / Thought ID? persisted?
End: elapsed ms + unverified/missing spans
```

The **authorization decision should be visible without leaking signing keys, passwords, raw private logs or copied prompts**. Copy Context defaults to redacted metadata; Copy Logs is bounded and explicitly labeled by origin.

## Startup and version correctness

**Identity is the user-facing entry point**, but may delegate an *approved* small launch profile to a **session runner** rather than knowing the command lines of every future feature.

- User explicitly chooses `Start Experience`; it is not “start everything on boot.”
- A selected local manifest must pin the *actual candidate* Thought Processor executable/source **commit SHA**, build provenance, supported protocol and data path. Do **not** use “latest,” silently checkout unknown HEAD or switch to legacy Thought-Controller.
- Session runner must be able to `attach` to an already-running authentic service or `start` a selected demo candidate, showing **which action happened**. Distinguish running instance ID from stable service ID and executable version.
- For the earliest build, start a **single-OS / same-host / loopback-only** profile with fixed explicit test services and isolated demo data. WSL-to-Windows and VM remote trust are future work.
- Explicitly record which processes were launched by this session. Stop only those. Do not terminate independent pre-existing services, Identity's own authority process or Project Controller.
- A separate **real worker may not exist**; report that instead of inventing a worker response or blocking Identity setup.
- If the selected Processor build is missing/incompatible, show `NOT INSTALLED / WRONG BUILD` with a deliberate way to select/build it. Do not pretend an HTTP 200 from another service is a valid Processor.
- If the `Identity authorization endpoint` is missing, show `AUTHORIZATION NOT IMPLEMENTED`; do **not** fall back to Engine's local ON grant in this test.

### Important launch ownership choice

For **the demo only**, a small session runner launched by Identity can also start the selected Engine and a **clearly approved** optional worker if a start command is available. Processor remains their operational observer/controller and is free to stay closed after connection. The session runner is responsible for ownership and clean shutdown; Identity remains the authority for *whether they are authorized to exchange data*, not for their processing.

This is deliberately **not** a permanent deployment specification. Future feature installers and lifecycles can remain independently owned.

## Implementation order for the next team

1. **Prove a real Processor process can start** with an honest `/health` or `/api/status` response (product, supported protocol, exact build SHA, process ID, instance ID, actual port) and render a minimal page without dependency on Project Controller.
2. **Connect to a running Identity** and actual selected Engine, distinguish `seen` from `authenticated` and `authorized`; present the external service URLs/provenance safely.
3. **Coordinate one session from Identity's Start Experience button** and verify that the intended Processor build opens with no substitution. If launch is unavailable, provide an explicit manual path and a truthful error.
4. **Implement the grant test** with Identity and Thought Engine teams before enabling `Try observation` as a real-authorized operation. Use [Engine protocol notes](https://github.com/Arameair/Thought-engine/blob/experiment/identity-grant-enforcement-20261024/docs/IDENTITY_AUTHORIZATION_EXPERIMENT_2026-10-24.md). The existing Engine tests are written but **not confirmed run**, and Identity has no production-compatible endpoint today.
5. **Add a tiny authenticated Demo Source S** and one observation button. Cross-link real Identity decision and Engine event receipt.
6. **Add failure trials:** revoke then resend; stop Identity and attempt again; stop Processor while valid services stay independent; reconnect/restart without duplicate service identity or unrelated-process killing.
7. **Only then** try optional 2–3 related toy events and show actual real worker results; more features/feeds are later.

An **intermediate milestone that only launches the real Processor and shows truthful status is useful**, but may not be labeled `END-TO-END GRANT DEMO PASS`.

## Acceptance checklist to show the owner

- [ ] New isolated Identity profile starts without Processor installed; healthy with zero optional features.
- [ ] User's explicit `Start Experience` launches/attaches the **pinned Thought Processor**, not Thought-Controller; confirms exact SHA and process.
- [ ] Opening Processor never silently authorizes a source or turns on the Engine's local toggle.
- [ ] One entity X and separate Demo Source S have distinguishable IDs; no impersonation by setting JSON source text.
- [ ] One real Identity-owned approval allows exactly the matching observation.
- [ ] Processor shows **actual** Identity decision and Engine receipt, with correlation to any real worker result.
- [ ] Revoking denies next event and prevents a queued event from reaching a model.
- [ ] Identity down means fail-closed, with a distinct reason and no model call.
- [ ] Worker unavailable gives QUEUED/UNAVAILABLE, never fabricated Thought.
- [ ] Clean stop only ends the session's owned child processes; recovery, restart and data persist safely.
- [ ] Test includes exact branch/head SHA, OS, process tree, ports, start/build durations, timings, log provenance, failures and rollback.
- [ ] No Alpha/Stable/Main promotion without explicit owner selection.

## Out of scope

No OAuth/GitHub account login, installer marketplace, arbitrary repositories, every feature auto-installing, Memory, TASS/Discord, real game/screen feeds, cross-VM networking, continuous AI model heartbeat/autonomous growth, dynamic coding, full permission hierarchy, broad UI gallery, multi-user, production credential management or an OS-level startup daemon.

**A tiny real flow is the product test.** If the experience works, we can decide which parts felt intuitive and which boundaries need redesign before writing a full architectural schematic.
