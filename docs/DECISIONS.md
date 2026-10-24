# Decisions and open questions

This log separates **owner direction**, **proposals**, and **confirmed implementation**. Avoid marking design assumptions as owner approval.

## Owner-selected integration and UI direction — updated 24 October 2026

**New scope:** Identity is the system's durable authority for participants **including independently developed features**, their relationships and permission policies. Thought Processor combines that knowledge with live discovery to help establish permitted connections and provide developer-visible evidence; Thought Engine keeps cognition and actual worker heartbeat/API calls.

**Selected behavior** (all still unimplemented here): a small shared connection core plus optional capabilities; automatic same-OS discovery with configured/manual remote fallback; one-time pairing through Identity; entity/scoped feature approvals; short bounded offline authorization; push-on-change plus occasional health checks; distributed logs and shared request traces; application cards with a single original UI preview; on-demand Flow map/timeline; verified version/test comparisons. Do not interpret connection discovery as authentication or feature approval.

**Open design questions:** hierarchical vs independent feature relationships, stable vs evolving Identity purpose, precise registry and pairing protocols, grant expiry and revocation scheme, canonical Engine and Memory boundaries, worker selection authority. [Research note and owner-supplied Grok critique](IDENTITY_EXPERIENCE_RESEARCH.md) · [issue for independent review](https://github.com/Arameair/Thought-Processor/issues/7).

**No evidence of code, tests or promotions** follows from recording these owner decisions. AI feedback is reviewed critically, not treated as implemented fact.

## Owner direction established in discussion — 2026-10-23

| Area | Direction | Consequence |
|---|---|---|
| Product strategy | **Features are the product**, independently usable | Thought Processing must not require that every other feature be installed |
| Control point | Rename/reframe the old controller concept as **Thought Processor**, a system control point | Do real operations, diagnostics and orchestration within the feature; don't become just another dashboard |
| Inputs | When the Thought Processing feature is enabled, allow feeds/logs from various independent sources to develop thoughts | Connect to actual log sources; don't assume fixed fabricated scenarios are enough |
| Identity | Thought Processing is enabled for an identity and features are managed separately | Feature entitlements and identity authority live outside the cognitive store |
| Worker choice | Chat-Jimmy is only our **current fast model** | Worker adapters and capabilities should be replaceable |
| Status | Make Alpha, Stable and experiments easy to track | Show exact commits, tests, owner verdict and rollback |

**Not yet confirmed:** automatic selection of every connected log source vs owner picking feeds; default retention; security mechanism; canonical Thought vs Thought-engine migration; full autonomous processing policy.

## Owner-selected authorization integration workflow — 2026-10-24

**Selected:** Thought Processor should be the **integration and testing hub** for connecting Identity and a Thought Engine, with a single operator-facing view of the real grant decision and Engine's actual event outcome. The independent apps can still communicate directly without Processor.

**Boundary:** **Identity decides; Thought Engine enforces; Thought Processor connects, tests, and explains.** Processor is a control/observability surface, **not** a second permission authority or the required data plane. Read [integration plan](INTEGRATION_AUTHORIZATION_PLAN.md) and [issue #8](https://github.com/Arameair/Thought-Processor/issues/8).

**Not selected / still open:** canonical Engine target, real local actor-authentication method, grant API fields, precise in-flight revocation behavior, first test sender identity and exact request correlation contract. The current first security experiment should fail closed if Identity is unreachable, **without offline cached allows**. Short-lived offline continuity from prior discussions is a later test choice, not an implemented promise.

**No functionality implemented by documenting this agreement.** Every cross-service operation must remain enforced by the receiving Engine when Processor is closed.

## Architectural proposals awaiting actual testing

1. Thought Processor is the **control plane**, Thought Engine the **data/processing plane**.
2. The engine is the sole authoritative store for its identity's Sparks, Traces and Thoughts.
3. The engine requests capability-specific intelligence (importance, matching, reasoning) from a worker adapter/registry.
4. Grant-off and feed-off are separate operations. Turning Thought Processing off must not stop source features.
5. Processing changes from the model are validated by the Engine and auditable in Processor.
6. One process may eventually handle many identities, but history, permissions, queue and model context stay isolated.
7. Avoid using normal run-time sample or mock records to make the operator UI appear active. Fixtures belong to tests.

## Open decisions

| ID | Question | Why it matters | Status |
|---|---|---|---|
| D-01 | Canonical Thought Engine repository and data migration | Prevent competing stores of truth | OPEN |
| D-02 | Per-feed permission selection vs all enabled sources | Privacy, noise, operational cost | OPEN |
| D-03 | Grant verification protocol and auth | Prevent one local service spoofing another | OPEN |
| D-04 | On/off behavior for existing waiting observations | Durability, owner expectation, retention | OPEN |
| D-05 | Filter-first vs model-first for high-volume logs | Throughput, costs, usefulness | OPEN |
| D-06 | Worker adapter contract, routing and fallback | Model independence without fake reasoning | OPEN |
| D-07 | Whether Processor UI and Engine ship together as one optional feature | Packaging/sales while keeping internal components separate | OPEN |

## How to close a decision

Record the date, issue/PR, alternatives, tradeoffs, **explicit owner choice**, exact build/experiment evidence (if technical) and rollback. Do not replace old entries; append a new one if the decision changes.
