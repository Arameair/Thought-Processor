> **Current design register — 2026-10-27:** The owner confirmed **Thought Processor is the normal human/admin interface for the entire system**, while **Identity owns security/permission decisions** and **the receiving feature enforces them**. New identities/features have **no information-source access without approval**. Older Engine-only control-point and Identity-first launch statements below describe the historical design stage; do not treat them as newer approved requirements. **Begin with [Design readiness](DESIGN_READINESS_2026-10-26.md), [the current workshop](https://github.com/Arameair/Thought-Processor/blob/experiment/processor-experience-design-20261026/docs/PROCESSOR_FOUNDATION_DESIGN_2026-10-26.md) and [Issue #15](https://github.com/Arameair/Thought-Processor/issues/15).** This banner is a documentation reconciliation, not authorization to implement or promote.

---

# Decisions and open questions

## Owner-confirmed additions — October 27, 2026

- **Canonical IDs:** Identity owns the one durable identity record. Processor requests creation; Thought Engine references that Identity ID for thought/history and cannot create a competing canonical identity.
- **Autonomy with no privilege escalation:** Processor may create a disconnected identity with zero grants. An identity may recommend features, but a suggestion cannot install, run, or access data. Routine work may use previously approved valid grants for the specific subject, destination and capability without re-approval on each call.
- **No protected bypass:** Identity remains the authorization authority. The receiving Engine and Worker enforce independently; local Engine toggles or attach tokens cannot grant access. New protected work and queued dispatch fail closed on missing/revoked/expired grants or Identity outage.
- **Replaceable worker:** Fast AI Worker is intended as a Thought Engine backend utility, not the only or permanent backend and not an ecosystem-wide worker for every feature. Dependency bundling/installation remains open.

Owner design thread: [Issue #20](https://github.com/Arameair/Thought-Processor/issues/20). These decisions do **not** claim Engine/Worker code complies yet. Current [P0/P1 implementation status](STATUS.md).

## Owner-confirmed conflict resolutions — October 26, 2026

[**Cross-repository decisions: authoritative October 26 register**](CROSS_REPO_DECISIONS_2026-10-26.md) · [Design discussion #15](https://github.com/Arameair/Thought-Processor/issues/15)

1. **Interface vs first process:** Thought Processor is the normal human/admin interface. **Identity or Processor may start first**, provided both arrive at the same verified ready state for full operation. Startup sequence is flexible; unauthenticated discovery is not a ready state.
2. **Install vs use:** GitHub repository visibility is only a **temporary prototype acquisition/install eligibility** rule. Identity separately authorizes specific identities and named source/action scopes; receiving features enforce. **Zero information-source grants by default.**
3. **Lifecycle:** **Explicit Processor application exit** prompts **Shut down managed workspace (default)** or **Keep services running**. System-owned processes and descendants must be tracked, shut down cleanly when chosen, checked for orphans, and reported. No killing unowned/external services without separate authorization. **Closing a browser tab is not an explicit app exit.**
4. **Pairing:** One operator-approved pairing; secure remembered credentials; automatic later reconnect to the **same verified peer**; explicit unpair/revoke. Pairing alone never grants data/feature access. Cryptographic mechanics remain open.
5. **Offline security:** **Fail closed in the first prototype** if Identity is unreachable or a required grant cannot be verified; block new protected work and queued protected dispatch. **No offline grace/cached allows.** Keep process state distinct from authorization.
6. **Memory:** Separate optional **future** feature, not implemented, designed or required in the current Processor prototype.
7. **Sequence:** Identity concept established → **Processor UX/design now** → **Thought Engine next**. Canonical `Thought` vs `Thought-engine` remains open; no automatic replacement/migration.
8. **Branch policy:** Design approval is not code, test evidence or owner Alpha promotion.

The October 23–25 concepts preserved below may disagree with the above. Where they do, the October 26 confirmed choices govern **design**, while historic text remains for provenance.

This log separates **owner direction**, **proposals**, and **confirmed implementation**. Avoid marking design assumptions as owner approval.

## Endorsed working distinction — 24 October 2026

> **Thought Processor controls the operational setup of thinking. Thought Engine controls the actual act of thinking.**

**Owner status: endorsed as a working architectural theory to challenge; not a final protocol or code acceptance.** The [detailed, falsifiable theory](OPERATIONAL_SETUP_THEORY.md) outlines possible operational controls, configuration ownership, feature interfaces, failure modes, first integration checks and competing architectures. Specific API shapes, persistence of desired cross-feature configuration, worker routing policy, identity proof and exact UI are **proposals or OPEN decisions**. Challenge them through [architecture review issue #7](https://github.com/Arameair/Thought-Processor/issues/7). No build, runtime test or branch promotion is implied.

## Owner-selected integration and UI direction — updated 24 October 2026

**New scope:** Identity is the system's durable authority for participants **including independently developed features**, their relationships and permission policies. Thought Processor combines that knowledge with live discovery to help establish permitted connections and provide developer-visible evidence; Thought Engine keeps cognition and actual worker heartbeat/API calls.

**Selected behavior** (all still unimplemented here): a small shared connection core plus optional capabilities; automatic same-OS discovery with configured/manual remote fallback; one-time pairing through Identity; entity/scoped feature approvals; **no offline grant allowance in v1 (fail-closed)**; push-on-change plus occasional health checks; distributed logs and shared request traces; application cards with a single original UI preview; on-demand Flow map/timeline; verified version/test comparisons. Do not interpret connection discovery as authentication or feature approval.

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
| D-02 | Mechanism for per-identity and per-feed selection | **Default-deny and explicitly selected sources are confirmed.** Exact UI/API and enforcement mechanics remain open | OPEN implementation |
| D-03 | Grant verification protocol and auth | Prevent one local service spoofing another | OPEN |
| D-04 | On/off behavior for existing waiting observations | Durability, owner expectation, retention | OPEN |
| D-05 | Filter-first vs model-first for high-volume logs | Throughput, costs, usefulness | OPEN |
| D-06 | Worker adapter contract, routing and fallback | Model independence without fake reasoning | OPEN |
| D-07 | Whether Processor UI and Engine ship together as one optional feature | Packaging/sales while keeping their runtime and ownership separate | OPEN |

## How to close a decision

Record the date, issue/PR, alternatives, tradeoffs, **explicit owner choice**, exact build/experiment evidence (if technical) and rollback. Do not replace old entries; append a new one if the decision changes.
