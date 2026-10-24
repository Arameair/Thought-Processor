# Decisions and open questions

This log separates **owner direction**, **proposals**, and **confirmed implementation**. Avoid marking design assumptions as owner approval.

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
