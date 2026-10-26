# Thought Processor

**The main workspace for managing an ecosystem of independent, identity-aware features.**

Thought Processor is the application a person or administrator opens to select an identity, discover available features, review permissions, interact with feature-owned results and understand what is happening. **Identity remains the security and permission authority.** A feature, such as Thought Engine or a future Memory service, runs independently and owns its work and records.

> **Project state — October 26, 2026:** Preparing a **first usable product-experience prototype**. The `main`, `alpha`, and `stable` branches remain documentation-only; runnable diagnostic and Engine-attachment **experiments** exist but have not been promoted or accepted as the end-user product.

## First, understand the boundaries

| Component | Owns | Does not do |
| --- | --- | --- |
| **Thought Processor** | Human/admin interface; Profiles, feature Catalog, Collections, approved actions, status and diagnostics | Issue its own permission grants; hold a second Thought/Memory database; proxy every feature call |
| **[Identity](https://github.com/Arameair/identity)** | Durable identities, approved relationships, entitlements, grants and security decisions | The actual thinking or the normal human dashboard |
| **[Thought Engine](https://github.com/Arameair/Thought-engine)** | Observations, queue, processing, identity-scoped Thoughts and worker interactions | Grant itself data access or manage the whole ecosystem |
| **Independent features** | Own functionality, data and logs; enforce relevant Identity decisions | Gain access to other identities or sources merely by being installed |

**No access by default.** A discovered feature is not necessarily trusted or authorized. Publisher permission to **obtain/install** software is different from an identity's permission to **use** it or share specific conversations, game activity or logs. Installation of reusable software does not merge the separate experiences of different identities.

## The design conversation we are having now

The first diagnostic Processor was built to help test Identity; the next goal is **something an owner can actually use and critique**. The current *proposal* is a small experience organized around **Profiles → Catalog → Collections**:

- **Profiles:** Choose which person, companion, business or other approved identity is being managed.
- **Catalog:** See real available features, their permission/connection states and the next genuine action.
- **Collections:** Browse authorized results owned by those features; show honest empty states when none exist.

This **layout is not yet finalized** and no new UI code has been authorized.

**Start the review here:**

[**Current design discussion — Issue #15**](https://github.com/Arameair/Thought-Processor/issues/15) · [**Detailed workshop and testable prototype proposal**](https://github.com/Arameair/Thought-Processor/blob/experiment/processor-experience-design-20261026/docs/PROCESSOR_FOUNDATION_DESIGN_2026-10-26.md) · [**Open issues and priority map**](docs/BACKLOG.md)

## What works, and what does not

| Status | Evidence |
| --- | --- |
| **Experimental Processor diagnostic shell** | [October 25 candidate](https://github.com/Arameair/Thought-Processor/tree/alpha-candidate-20261025) starts and reports local Identity as **found, unverified**; earlier owner trial recorded in [#12](https://github.com/Arameair/Thought-Processor/issues/12). Not the intended product UX. |
| **Experimental Engine attachment and observations** | [October 26 candidate](https://github.com/Arameair/Thought-Processor/tree/alpha-candidate-20261026) adds an attach code, observation status and Thoughts view. Reported tests included a **fixture worker**, not proof of useful live thinking or Identity-issued ingest grants. |
| **Experimental Identity-approved Engine installer** | [Separate October 26 experiment](https://github.com/Arameair/Thought-Processor/tree/experiment/install-thought-engine-20261026) used a **simulated GitHub provider** and a temporary repository-visibility rule; it does not establish real publisher entitlement, a release, or permission to start and ingest. |
| **System-wide Processor experience** | **DESIGN ONLY**. An authenticated Processor-led Identity setup, real profile/catalog API and a meaningful enforced feature action are not yet an accepted end-to-end prototype. |
| **Alpha / Stable / public release** | **No runnable promotion or release.** Keep candidate commits and owner verdicts separate from branch labels. |

[Full evidence ledger](docs/STATUS.md) · [Design decision boundaries](docs/DECISIONS.md)

## Navigation

[Documentation index](docs/README.md) · [Architecture](docs/ARCHITECTURE.md) · [Current design readiness](docs/DESIGN_READINESS_2026-10-26.md) · [Issue backlog](docs/BACKLOG.md) · [Experiment workflow](docs/WORKFLOW.md)

## Collaboration and release policy

New functional experiments **start from the owner-controlled `alpha` branch**, which is presently a documentation baseline. Any older candidate source must be deliberately ported, with its original SHA documented. The owner chooses KEEP / REVISE / REJECT and authorizes each promotion; there is **no automatic merge** into `alpha`, `stable` or `main`. The Project Controller is optional development tooling, not an end-user dependency.

Design notes, source code, fixture tests, real-service tests and owner acceptance are different evidence. Never present unverified access or fake Thoughts as live results. [AI collaboration and verification](docs/AI_COLLABORATION.md).
