# Authorization integration: Thought Processor as the test hub

**Owner-approved planning direction · 2026-10-24 · DESIGN ONLY / NOT BUILT.**
**Tracking:** [Thought Processor #8](https://github.com/Arameair/Thought-Processor/issues/8) · [Identity #15](https://github.com/Arameair/identity/issues/15) · [Thought-engine #1](https://github.com/Arameair/Thought-engine/issues/1) · [Engine choice #2](https://github.com/Arameair/Thought-Processor/issues/2).

> **Identity decides. Thought Engine enforces. Thought Processor connects, tests, and explains.**

This is the selected place to **coordinate and observe** the first authorization integration. It is **not** a change of which service issues or enforces permissions. Thought Processor does not sit in the production observation path or become a duplicate grant authority.

## What the operator should accomplish in one workspace

1. View authentic **Identity and Thought Engine** service identities, health, protocol compatibility, instance IDs and exact test build.
2. Select **one entity and one verified source service**. Do not confuse an entity (e.g. TASS) with a process, account or human-readable unverified source name.
3. Request one scoped grant/disable action **through Identity's authenticated administrative endpoint**. The actual user approval and policy are Identity's, not Processor's local UI toggle.
4. Submit one intentional test observation using a **separately authenticated test sender** with proper source identity; do not forge the source in a payload or masquerade as TASS.
5. Observe the resulting **Identity decision** and **Engine intake and processing receipts**, linked by correlation/decision/event IDs. No fabricated green state.
6. Revoke the Identity grant and test **the next incoming event without restarting**. Also check queued work before the model sees it.
7. Stop Identity to test **fail-closed** rejection and authentic outage status. Close Processor and prove service independence.

## Roles at the integration boundary

| Service | Authoritative for | Cannot do |
| --- | --- | --- |
| **Identity** | identity records and authenticated service registration; grant record, issuance, revocation and decision receipts | silently route/process all observations or take over another service lifecycle |
| **Thought Engine** | input validation, caller authentication, permission enforcement at intake and before worker call, queue/processing receipts | approve its own grant via a local on/off flag |
| **Thought Processor** | operator selection, triggering appropriately authenticated tests, asking Identity for approved changes, observing both services, reporting evidence | invent permissions, bypass receiving service, store master tokens or proxy all high-rate event traffic |
| **Source feature / test actor** | generating the genuine observation with authenticated service identity | impersonate a different source simply by setting `source` text |

## Proposed smallest real grant

An **Identity-owned protected durable record**, not an Engine-owned copy of permission truth:
- Grant ID and version, entity ID;
- authenticated source service ID and destination Engine service ID;
- action `thought.observation.ingest`, allow/deny or enabled/revoked, optional expiry;
- owner approval evidence and traceable decision.

A callable **authenticated authorization check** evaluates this grant for the concrete source/entity/Engine/action. The Engine must authenticate the HTTP sender, and Identity must authenticate the Engine. Exact transport and credentials **remain undecided**: do not mistake a loopback port, service string or JSON field for secure actor authentication.

## Failure matrix for the *first* experiment

| Input or event | Engine behavior | Processor view |
| --- | --- | --- |
| Known actor with matching current Identity grant | `202 queued` and decision/event receipt; recheck before model send | **AUTHORIZED / QUEUED**, actual IDs and timestamps |
| No grant / disabled / revoked | `403`; do not queue or call worker | **DENIED**, Identity decision and Engine refusal |
| Wrong entity/source/action | `403` | **SCOPE DENIED**, exact mismatch as safely reportable |
| Unknown/unauthenticated sending service | `401` or equivalent; refuse before grant check | **UNVERIFIED SOURCE**, not a grant success |
| Identity timed out or unavailable | `503`; **fail closed**, do not accept or newly process | **AUTHORITY UNAVAILABLE**, distinct from explicit DENY |
| Grant revoked while an event is queued | Recheck immediately before any worker request; pause/deny prior to send | **QUEUED / BLOCKED BY REVOCATION** |
| Grant revoked after worker send | Attempt cancellation and prevent further result commit when possible; previously transmitted text cannot be recalled | **IN FLIGHT / REVOCATION LIMITATION**, with truthful evidence |
| Processor is closed | Direct, properly authorized source→Engine communication continues | No Processor involvement required |

**Initial gate:** no offline allow cache and no lease grace period. Earlier general design allowed testing short-lived offline leases; that remains **a later, separately approved security/performance experiment**, not the default for the first proof.

## Build order: minimal backend first, Processor integration next

**A. Decide contract.** Choose **one Engine for the experiment** (without declaring permanent repository winner or migrating data); agree on service authentication, grant record fields, API check and failure codes. Record exact branch/SHA.

**B. Prove the backend.** With real Identity and Engine running independently, test allow/deny/revoke/unavailable against an authenticated source, including a queued observation. A small CLI/test harness is permitted to collect evidence while Processor remains documentation-only. Fixture auth is labeled as such and never called secure.

**C. Build the narrow Processor control point.** Add only true status cards, one entity/source selection, request permission via Identity, one authenticated test send, event/decision timeline, Copy Context and rejection reasons. Controls reflect actual supported endpoints and verified effects. No gallery or full dashboard prerequisite.

**D. Verify independence.** Run same authorization tests with Processor open/closed. Capture performance overhead and provider/worker-call counts. Only then expand to other features and a detailed implementation schematic.

## Decisions **still open** before detailed schematic

1. **Which Engine is the initial target?** `Thought` and `Thought-engine` are different experimental stores. Resolve [issue #2](https://github.com/Arameair/Thought-Processor/issues/2) *for the first test*, leaving data migrations separate.
2. **How is service identity authenticated?** Engine↔Identity, Source↔Engine, administrative actor↔Identity and Processor→each service. Port claims and unauthenticated CLI requests aren't enough.
3. **What is the precise grant/decision interface?** Path/method, typed inputs and outputs, permissions, timeout, correlation ID, version/expiry and safe errors.
4. **What happens to already persisted queued work after revocation or outage?** No model send; specify pause vs deny, retention and eventual retry policy.
5. **How can Processor trigger a test safely?** Test actor has its own identity; owner controls and audit show that it is a test, not organic TASS behavior.
6. **Where will authority receipts be exposed?** Identity-owned decision log and Engine-owned intake/processing receipt need matching IDs; API cannot leak private observation text to unrelated identities.
7. **What counts as an end-to-end pass?** Real service caller authentication and Identity+Engine enforcement, not just mocks or green UI; exact measurable latency and failures.
8. **When does the optional Processor UI come in?** Backend gates cannot wait for a comprehensive UI, but the owner-selected integrated workflow must eventually be demonstrated there.

## Deliberately out of scope

No OAuth/GitHub token system, Memory, thought-directed wave algorithm changes, automatic feeds, background service installer, broad feature catalog, permanent remote/VM multi-host identity, multiuser control or offline cached grants in this first test.

The original architectural principle remains:
> **Identity owns the organization and authority of the system. Each feature owns its actual work.**

**Collaboration:** Owner-directed design, documented with AI assistance and subject to independent critical feedback. No tests or code changes were made by this plan.