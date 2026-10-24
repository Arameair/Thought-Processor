> **Story revision — 24 October 2026:** Read [the revised owner journey](../story/01_OWNER_JOURNEY.md) and [blueprint handoff](../story/03_BLUEPRINT_HANDOFF.md) first. This build document predates the accessible-feature/identity-relationship direction and is **historical, not a current executable assignment** until reconciled. In particular, any earlier optional-GitHub or unapproved attach/control allowance is superseded by the [Identity-approved build package](https://github.com/Arameair/Thought-Processor/blob/experiment/identity-approved-build-package-20261024/prototype/START_HERE.md). Creating/searching features, broad dependency installation and version comparisons are not authorized implementation scope. The owner has authorized story edits only in this pass.

# Test design — Thought Processor story-derived prototype

**Prepared 24 October 2026 | pre-implementation test specification, NOT test results.**

Purpose: make it difficult to mistake a beautiful UI, a green unit test, a responding port, or a mock worker for a **real secure integration**. These scenarios are drafted from [the owner journey](../story/01_OWNER_JOURNEY.md), [product expectations](PRODUCT_EXPECTATIONS.md), the [Claude brief](CLAUDE_IMPLEMENTATION_BRIEF.md), [Processor Issue #9](https://github.com/Arameair/Thought-Processor/issues/9), and [authorization Issue #8](https://github.com/Arameair/Thought-Processor/issues/8).

## Test layers; report them independently

| Layer | Allowed evidence | What it can demonstrate | What it CANNOT demonstrate |
| --- | --- | --- | --- |
| **L1 Unit** | Go/JS tests, isolated functions, crafted input | Parser, state mapping, redaction, error handling | Live Identity/Engine behavior |
| **L2 UI fixture** | Explicitly marked fake test server, synthetic receipts | UI renders states correctly and avoids fake certainty | Genuine authorization, real AI output or persistent Engine data |
| **L3 Running single service** | Processor + real Engine read endpoints or Identity status | Actual process, current status, restart, port/PID | Trust just because a health endpoint answers |
| **L4 Authenticated integration** | Actual Identity + real Engine + independently authenticated Demo Source | Scope-bound allow/deny/revoke/identity-outage behavior | Production-grade security or real model work |
| **L5 Real worker** | Non-mock, permitted Engine→worker response and Engine storage proof | Actual model invoked, real results/latency when available | Thought guaranteed from a single observation |
| **L6 Owner trial** | Human owner follows runbook and reports verdict | Product understandability and repeatable workflow | All deployments, long-term reliability or universal model quality |

**Every result must name its layer.** “PASS L2” cannot be rewritten as “PASS L4”.

## Preflight—what Claude must capture

- Full commit SHA for \`alpha\` base and experiment candidate; exact Engine and Identity branch/SHA used, plus any distinct test source.
- Machine OS, architecture and runtime versions; which local/user profile; service start commands where safe to record.
- Session/run ID, isolated data roots, actual PID/port for each process; **actual or unknown** build provenance.
- Authentication design and source of trust proof; whether any fixture/mock is present.
- Real worker available? configured? mock? blocked? offline? which source reported it?
- Which tests ran; elapsed durations and per-test log/receipt IDs.
- Clean-up ownership and list of intentionally preexisting services that must remain running.

**Never copy credentials, authorization signing keys, observation bodies or raw model prompts to an issue.**

## Group 1 — First use and independent process

| ID | Setup / action | Expected result | Evidence | Gate |
| --- | --- | --- | --- | --- |
| **UX-01** | Start Processor on a clean isolated profile without Identity/Engine | Workspace opens; truthful empty state, one next action **Connect/Find Identity**; no fabricated feature cards or ready marks | Screenshot or owner recording + status response | G1 |
| **UX-02** | Select a remembered last-used identity that is no longer reachable | Show cached record with **stale/unknown** time; do not say LIVE | State snapshot with checked timestamp | G1 |
| **UX-03** | User has only an independent non-thinking feature, or no companion | System overview remains valid; no mandatory “Brain Offline” error | UI review using appropriately labeled fixture or real service | G1/L2 or G2/L3 |
| **UX-04** | Open selected Engine original UI if valid | Actual application UI appears once, or safe **Open Original UI** fallback; no simulated rebuilt UI | Browser interaction, URL/origin validation | G1/G2 |
| **UX-05** | Select TASS identity and Engine separately | UI clearly distinguishes entity ID from service/instance ID, preserves navigation context | UI capture and model fields | G1/L2 |
| **UX-06** | Small screen, keyboard navigation, long names | No unusable overflow, controls reachable, readable statuses | Owner/dev interaction notes | G1 |
| **OPS-01** | Start Processor as standalone app, no Project Controller | Process starts, reports actual self identity/port/PID where available; opens clean window/browser | Command, pid, port, startup timing | G1 |
| **OPS-02** | Close/restart Processor, retained service remains alive | Processor recovers/refreshes, does not kill Engine/Identity; no unsafe stale success | Process list, self status, old/new checked timestamp | G1 |
| **OPS-03** | Open Processor twice / use same profile | No corrupt data, unintended duplicate service takeover or inexplicable port collision | Processes/ports, clean error/attach semantics | G1 |
| **OPS-04** | Inspect Processor after a crash/forced close | Clear recovery/restart instructions; no orphan runner-owned process unless identified as such | Process tree/cleanup report | G1 |

## Group 2 — Real discovery and observational truth

| ID | Setup / action | Expected result | Evidence | Gate |
| --- | --- | --- | --- | --- |
| **DISC-01** | Probe wrong/unused local port | **NOT FOUND / UNREACHABLE**, not CONNECTED | Port probe response and UI | G1/G2 |
| **DISC-02** | Probe unrelated HTTP server returning 200 or friendly product name | At most **DISCOVERED/UNVERIFIED**; never AUTHENTICATED, AUTHORIZED or ENGINE READY | Server identity proof absent; status/UI capture | G2 |
| **DISC-03** | Read selected real Thought-engine \`/api/status\` | Current Engine-reported queue, worker and runtime facts mapped without invention; source/freshness visible | Exact endpoint response, timestamp, UI | G2 |
| **DISC-04** | Stop Engine while Processor open | Status becomes offline/stale within measured bounded detection; historic traces marked historical | Timestamps and UI/status diff | G2 |
| **DISC-05** | Engine restarts at changed port/instance ID | Durable service identity not confused with run instance; no auto-trust of new unverified process | Old/new instance/endpoint evidence | G2/L3; L4 for auth |
| **DISC-06** | Engine busy/worker not configured or mock/unavailable | Distinct worker state; no fictitious replies, queue not relabeled processed | Actual worker state + Processor view | G2 |
| **DISC-07** | Supported/unsupported Engine protocol version | Clear compatible or incompatible state; unsupported commands disabled | Protocol descriptor/adapter test | G2 |
| **DISC-08** | UI state says “Applied” after a command | Owning feature receipt **and read-back** agree; otherwise PENDING/UNKNOWN | Command ID + post-command status | G2 for read-only; G3b for protected commands |

## Group 3 — Identity launch experience (depends on Identity team)

| ID | Setup / action | Expected result | Evidence | Gate |
| --- | --- | --- | --- | --- |
| **LAUNCH-01** | Start Identity only, clean demo profile | Healthy Identity with zero features; nothing silently started | Process list before/after | G3a |
| **LAUNCH-02** | Click Start Experience and approve pinned profile | Starts/attaches **Thought Processor** exact selected repo+SHA, **not Thought-Controller** | Profile manifest, exec SHA, product/protocol and PID | G3a |
| **LAUNCH-03** | Selected build missing, wrong, unknown SHA or incompatible | Explicit NOT INSTALLED/INCOMPATIBLE; no substitution, no silent latest download | Error/provenance log | G3a |
| **LAUNCH-04** | Start Experience twice | Second action attaches or clearly refuses; does not create duplicate conflicting sessions | Two run IDs or same instance with ownership policy | G3a |
| **LAUNCH-05** | Stop Demo with a separately pre-running Engine/worker | Only owned children stop; outside processes survive; Identity and durable data remain | Before/after PID list and files | G3a |
| **LAUNCH-06** | Launcher fails halfway through start | Clear partial-start failure and cleanup, no abandoned owned children, rollback preserved | Session log, process ownership audit | G3a |

**Owner of Start Experience:** Identity team and approved bounded session runner. Claude should build only the Processor-side endpoint/session consumption unless authorized to edit other projects.

## Group 4 — Trust and authorization (BLOCKED until real Identity + Engine contract)

| ID | Setup / action | Expected result | Evidence | Gate |
| --- | --- | --- | --- | --- |
| **AUTH-01** | Send observation with no grant | Engine returns **403 denied** before storing/queueing; worker calls **0** | Real Identity decision + Engine receipt + worker-call count | G3b/L4 |
| **AUTH-02** | Approve exact S→E, entity X, \`thought.observation.ingest\` | Signed/authenticated allow accepted by Engine; unique decision and grant version; **202 queued** if valid input | Identity grant decision + Engine queue receipt | G3b/L4 |
| **AUTH-03** | Change source S→S2 without grant | Denied; spoofed source text does not authenticate actor | Real peer identity/auth proof + 401/403 receipt | G3b/L4 |
| **AUTH-04** | Same S, but different entity Y or action | Denied despite valid other-scope grant; worker calls **0** | Decision/receipt with matching IDs and scope | G3b/L4 |
| **AUTH-05** | Unknown sender, missing credential or malformed signature | **401** unauthenticated (or documented equivalent), not allow | Request auth failure receipt | G3b/L4 |
| **AUTH-06** | Revoke allow without restarting Engine, resend one unique event | Next intake **403** and no worker call; previous persisted history not erased | Grant version transition, request/decision/event IDs | G3b/L4 |
| **AUTH-07** | Queue event, then revoke before worker dispatch | Engine rechecks and blocks dispatch; no model request for event | Queued receipt, revoke time and zero worker calls | G3b/L4 |
| **AUTH-08** | Stop Identity; send new protected event | **503 AUTHORITY UNAVAILABLE**, no queue or worker call; distinguish unavailable from deny | Timeout/error receipts with timestamps | G3b/L4 |
| **AUTH-09** | Restart Identity with same authorized registry | Grant state accurately restored; recheck respects current version/revocation | State before/after restart + decision IDs | G3b/L4 |
| **AUTH-10** | Replay duplicate event ID under same/changed permission | Deduplication is scoped to authenticated sender/entity; no old duplicate receipt leaks across revoked scope; current permission evaluated first | Duplicate receipts, matching current grant, call count | G3b/L4 |
| **AUTH-11** | Deliberately submit while Processor is closed | Authenticated source→Engine direct intake works if Identity available/authorized; Processor not on hot path | Service logs with Processor PID absent | G3b/L4 |
| **AUTH-12** | Revoke while worker request is already sent | No false promise to erase transmitted content; attempt cancellation/block further commit as designed; flag in-flight limit | Worker call timestamp, revoke decision, Engine result | G3b/L4 |

**Absolute gating:** If Identity lacks the real grant endpoint, these are **BLOCKED / NOT RUN** for L4. Fixture equivalents may run under L2 and must be reported separately. **Never flip Engine's local ON/OFF flag and call AUTH-02 a pass.**

## Group 5 — Observation / thinking / evidence lifecycle

| ID | Setup / action | Expected result | Evidence | Gate |
| --- | --- | --- | --- | --- |
| **FLOW-01** | Submit one valid authenticated test observation | Dedicated Demo Sender S calls Engine **directly**; Processor receives/verifies Identity and Engine receipts only | Source event ID, Engine receipt, network/path evidence | G3b |
| **FLOW-02** | Worker unavailable after legitimate 202 queued | **QUEUED — WORKER UNAVAILABLE**; no Spark/Thought invented, no fixture-as-live label | Real Engine queue/worker status | G3b/L4 |
| **FLOW-03** | Real worker available and call occurs | Show measured call ID, elapsed timing, actual reply validation; separate score/Trace/Thought/persisted statuses | Engine worker log and per-event receipt | G3b/L5 |
| **FLOW-04** | A single observation is scored low or unrelated | Test may be successful for receipt/processing but forms **no Thought**; explain actual Engine rule | Score/evidence/traces count | G3b/L5 |
| **FLOW-05** | Worker busy/timeout/uncertain model send | Honest failure/uncertain, no automatic duplicate send | Worker call IDs, retry safety metadata | G3b/L5 |
| **FLOW-06** | Observation accepted but processing/persistence fails | Report the exact boundary—not end-to-end PASS—with preserved data/error | Engine receipt, storage error and trace | G3b/L4 or L5 |
| **FLOW-07** | Missing trace span from Identity or Engine | “Unverified/missing step” instead of reconstructing invented success | Missing correlation link and UI | G1/L2, G3b/L4 |
| **FLOW-08** | Try 2–3 related toy events | Each explicitly sent; resulting Spark/Trace/Thought depends on actual model/rules; counts not invented | 2–3 request/event IDs, raw Engine receipts | Optional G3b/L5 |

## Group 6 — Privacy, diagnostics, resilience and usability

| ID | Setup / action | Expected result | Evidence | Gate |
| --- | --- | --- | --- | --- |
| **SAFE-01** | Copy Context/Logs with strings containing token/password/observation text | Default export excludes or redacts content, includes provenance and omissions count | Sanitized sample + test assertion | G1 |
| **SAFE-02** | Attempt arbitrary external UI URL or malicious iframe origin | Refused/not embedded; safe approved original-app link only | Validation/error capture | G1/G2 |
| **SAFE-03** | Running status endpoint stalls/timeouts | UI stays responsive; unknown/degraded status with bounded timeout | Timeouts, UI interaction and durations | G1/G2 |
| **SAFE-04** | Non-owned process shares port or looks like a known app | No forced kill or attachment based only on PID/port; show conflict/unverified | Process ownership and run receipt | G1/G3a |
| **SAFE-05** | Profile/data directory unwritable or corrupted | Explicit failure, no misleading saved success and no mutation outside isolated root | FS before/after, error evidence | G1 |
| **SAFE-06** | Restart after partially applied configuration | Processor rereads owning service; surfaces mismatch rather than applying stale desired state | Pre/post status/readback | G2/G3b |
| **SAFE-07** | Start/stop 3–5 times on same isolated profile | No duplicate server/process, orphan, wrong build or unintended data deletion | Each run ID/PID and process audit | G1/G3a |
| **UX-07** | Owner unfamiliar with architectural terms attempts the demo | Can identify next action, know source of failure, and distinguish authorization vs processing | Owner commentary and task result | Owner trial |

## Proposed test ordering and severity

**P0 / security or boundary failure:** any unauthorized worker call, permission bypass, source spoofing, misleading authorized/processed badge, killed unrelated process, leaked secret, repository promotion, insecure blind attachment. **FAIL candidate; no promotion recommendation.**

**P1 / core prototype blocker:** cannot start standalone, no meaningful truthful empty screen, wrong product opened, broken real status read, missing safe Stop, unusable Copy Context, silent uncertain resend. Fix before owner signoff on relevant gate.

**P2 / polish:** noncritical visual layout, advanced filtering, optional UI transitions or additional metrics. These can be recorded and deferred as long as clarity remains.

All tests MUST be represented in a final table with:
\`Test ID | Layer | PASS/FAIL/BLOCKED/NOT RUN | exact result | evidence file/ID | known limitation\`.

## Minimum smoke run for owner

1. Start standalone Processor; check UI states and real identity/version metadata (**UX-01, OPS-01**).
2. Intentionally point to the wrong Engine port, then the selected real Engine; verify clear difference (**DISC-01/03**).
3. Inspect a missing grant/authorization interface; see **AUTHORIZATION NOT IMPLEMENTED** if appropriate (**AUTH group explicitly BLOCKED**).
4. Copy a sanitized context report (**SAFE-01**).
5. Stop/restart Processor; verify Engine wasn't stopped (**OPS-02/SAFE-07**).
6. If real Identity, Engine and Demo Sender grant integration exists: allow one event, revoke then deny, fail closed on Identity outage, repeat with Processor closed (**AUTH-02/06/08/11, FLOW-01**).
7. If a real worker is genuinely connected, record actual processing output, or report **WORKER UNAVAILABLE** (**FLOW-02/03/04**).

Do not require a Chat-Jimmy website login or external paid API to complete the truthful stand-alone/read-only smoke run.

## Reporting form (complete for each gate)

\`\`\`text
Gate / test:
Result: PASS | FAIL | BLOCKED | NOT RUN
Evidence layer: L1 | L2 | L3 | L4 | L5 | L6
Processor repo / branch / full SHA:
Alpha base full SHA:
Identity repo / branch / full SHA or unavailable:
Engine repo / branch / full SHA or unavailable:
OS + profile root:
Service IDs + run IDs (nonsecret):
PIDs + local ports:
Input/action (sanitized):
Expected:
Actual:
Operation/request/decision/event IDs:
Elapsed timings + source:
Worker called? Actual count / unknown:
Security/side effects:
Reproduction steps:
Rollback / next step:
Owner verdict: PENDING
\`\`\`

**No executed tests are claimed here.** These are instructions to Claude and the owner for producing independently inspectable test evidence.