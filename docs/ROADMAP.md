# Roadmap — prove one real feature end-to-end

Each gate is independent, should leave existing applications usable, and has an explicit **pass condition**. The dates below are not release promises.

| Gate | Aim | Pass criteria | Current state |
|---|---|---|---|
| 0 | Documentation and branch lanes | README, architecture, STATUS, workflow, decision log; Alpha/Stable docs baseline | In progress / docs only |
| 1 | Real Processor status/control view | Processor starts alone in Project Controller; discovers compatible Engine; shows actual PID, SHA, port, state; no simulated success | Not implemented |
| 2 | One log source into one thought engine | Owner selects TASS identity + one authorized source; observation id persists; queue, Spark/Trace/Thought visible; feed pause leaves source running | Not implemented |
| 3 | Feature authority | Identity grant OFF blocks, ON allows, revocation stops future processing and keeps history; authorized feed selection; no cross-identity data | Not implemented |
| 4 | Replaceable intelligence | Engine asks for *capability*, not "Chat-Jimmy"; adapter health, one call, metrics, error classification, no duplicate uncertain sends; provider swap without data migration | Not implemented |
| 5 | Operations / fault recovery | Thought Processor can pause/resume, process one, inspect/retry eligible jobs, show evidence pipeline and diagnostics; survives restart | Not implemented |
| 6 | Owner-approved Alpha, Stable and release | Owner Project Controller tests with exact SHA and logs, then manual promotion only | Not implemented |

## Owner-selected integration plan — Thought Processor as test hub (2026-10-24)

**[Read the focused authorization integration plan](INTEGRATION_AUTHORIZATION_PLAN.md)** · [Tracking issue #8](https://github.com/Arameair/Thought-Processor/issues/8).

**Identity decides; Thought Engine enforces; Thought Processor connects, tests, and explains.** Processor is the owner-facing location for the first Identity↔Engine integration test, but is **not** a mandatory observation proxy, grant issuer or provider of fabricated state. Backend checks must work and remain secure even when Processor is closed.

Before building an extensive Processor dashboard, prove **one Engine, one entity, one authenticated source, one Identity grant**. The current Gate 2 vs Gate 3 order above is a long-term roadmap: **the first security test should prove real authorization at intake BEFORE calling any end-to-end event path "authorized."** No fixture grant is a release authorization.

| Mini-gate | Required proof | Status |
| --- | --- | --- |
| A — Design and choose test Engine | Authenticated caller contract, one Identity-owned grant and failure behavior; [canonical Engine issue #2](https://github.com/Arameair/Thought-Processor/issues/2) scoped to the experiment | **OPEN** |
| B — Identity/Engine backend proof | Real source allow/deny, revoked next POST, queued event recheck, Identity outage fail-closed | **NOT RUN** |
| C — Processor-driven test and diagnostics | Real grant requests via Identity; a test actor sends one event; correlated Identity and Engine receipts, no spoofed source | **NOT IMPLEMENTED** |
| D — Independent operation and review | Shut Processor; eligible direct connections continue; compare measured time and call counts; owner verdict | **NOT RUN** |

This sequence intentionally excludes model-wave changes, OAuth, Memory, auto-feeds, broad catalog and an offline authorization cache. Existing aspirational broad gates still apply afterward.

## Thin first vertical slice

Before integrating many source features or cloud models, implement **only**:

- One running compatible Thought Engine discovered and verified.
- One explicitly selected identity (TASS), with grant clearly labeled as real or provisional.
- One permitted source feed delivering real logs.
- One model worker, currently Chat-Jimmy, through an adapter boundary (no permanent hardcoding).
- One controlled processing request; show original log, queue receipt, model decision, persisted Trace/Thought and latency.
- Stop/Play and failure handling with no data mix-ups.

If Identity-grant integration is not ready, show **AUTHORITY NOT CONNECTED** and do **not claim production permission enforcement**. Tests can use isolated fixtures, but they cannot be the live customer-facing state.

## Questions to settle before implementation

1. Which Thought Engine is **canonical**? `Thought` and `Thought-engine` are separate prototypes. Do not migrate data or declare one an approved successor without a decision and evidence.
2. How does Identity issue/verifiably revoke a grant across processes, and how do local applications authenticate?
3. Are source subscriptions configured separately for each identity, or inherited from allowed source features? Do not automatically select all sources.
4. How much raw log input should be stored/retained, batched or prefiltered, and where does consent apply?
5. What happens to queued events when the feature/worker is disabled? Set explicit retention/budget rules.
6. What is a strong enough real-model benefit to justify model calls compared with deterministic filtering?
7. When does the operator need to approve a worker-generated action versus merely view an interpretation?

**Current recommendation:** resolve canonical Thought ownership and permissions before building a broad manager, but a read-only local discovery/status view can be tested earlier without assuming authorization.
