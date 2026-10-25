# Owner design alignment — feature permissions, shared Engine and identity-owned Thoughts

**Date:** 2026-10-25  
**Status:** Owner-confirmed conceptual direction, **documentation-only**. No implementation, tested integration, approved wire schema, release, or Alpha/Stable/Main promotion. This note reconciles the earlier October 25 discussion notes and checkpoints on this experiment branch. The listed conceptual decisions reflect the conversation; undecided protocol details remain OPEN.

## Current agreed mental model (start here)

- **Identity exists first.** A new identity begins with its own canonical ID, name and separate protected records, but no optional feature automatically connected and **no access to any information source by default**. Names such as TASS, RASS, Tom and Mike are not system categories.
- **One installation of a feature implementation can serve many identities.** Features may be separate repos with their own installation, versions, configuration and data storage. Each identity uses its own approved association, effective settings and isolated data context.
- **One Thought Engine can process several features and identities concurrently.** The Engine is reusable processing infrastructure, not one separate "brain" binary per identity.
- **Features produce Sparks by selecting/condensing their own permitted events or logs.** A Memory Spark, a game Spark and a Cognition Spark have one consistent **Engine-facing format** even though each feature decides how to create its own Sparks. Engine does not need raw thousands of log lines for every event. Features may retain original raw logs and supply bounded references to them.
- **AI worker + Engine relate same-identity Sparks into Threads, then related Threads form identity-owned Thoughts.** Threads may link Sparks from *different features* **for the same target identity**. Thoughts, Thread memberships, queues and results must remain separate between identities. Worker is replaceable; Chat-Jimmy is a present experimental worker, not the definition of Thought Engine.
- **Stop at producing observable Thoughts initially.** Coherence, elaboration, what an identity does with Thoughts, long-term Memory internals, cognition strategies, expression and autonomous actions are follow-on investigations, not Phase 1 gates.
- **Build for measurement:** source→Spark timings, grant/admission outcome, queue wait, worker and model mode, invocation count and duration, Thread grouping, Thought persistence, per-feature/per-identity attribution, CPU/RAM and failure/unknown states. Compare identical recorded inputs when changing a feature or worker.

## Security decisions: no access until deliberately granted

Two permission layers are separate and must never be collapsed:

1. **Publisher/acquisition entitlement:** Is this administrator/account allowed to obtain or install this feature version from its publisher? An authenticated GitHub sign-in, a discovered GitHub repository, or a running local process **does not by itself** grant a feature entitlement. Download/install approval is distinct from feature operation.
2. **Per-identity connection and data-source authorization:** Is the feature enabled for this *particular* identity, for specified actions on specified sources and data classes? **After acquiring or installing a feature, its data-source permissions are EMPTY by default.** Explicitly choose what the feature can read, collect, send, and receive, and for whom. Enabling a feature does not confer access to messages, game logs, Memory, other identities, or all files.

An illustrative first setup: administrator has permission to install Memory once; TASS may connect that installation, but the TASS→Memory connection initially has **zero approved data sources**. The operator can later approve named sources/scopes in the Identity-governed profile. RASS has its own independent Memory association with its own empty grants, even if it shares the executable. Revocation must prevent new protected work and be visible without erasing unrelated historical results.

**The identity profile should expose the identity's approved feature connections and their precise allowed information flows.** Identity owns canonical policy/grant records and administrator/publisher relationships; it must remain capable of enforcing least privilege even when Processor is closed. Receiving features verify identity, scope and valid grant decisions at the point of actual work. Thought Processor shows, proposes and requests changes but is not an authorization oracle.

**Shared software does not mean shared experiences.** TASS's Memories, Sparks, Threads and Thoughts are not implicitly visible to RASS, nor can a feature fuse their histories by default. Cross-identity influence should enter as a separate authorized communication experience, not hidden cross-identity database access. More detailed Memory permission design is deferred to the Memory feature discussion.

## Processing-related optional features

- **Attention** and **Decay** are candidate *independent optional features*, not irrevocably hardcoded rules of the core Engine. They may change scheduling, ranking or selection, **not rewrite the original immutable Spark/evidence**. This enables controlled comparisons across configurations.
- The current Engine implementation has an optional timed heartbeat for advancing active Thoughts and an event queue loop; the worker's `/api/heartbeat` is a *readiness endpoint*, not the scheduling heartbeat. Scheduling/fairness across concurrent lanes is OPEN.
- Features can emit domain Sparks from approved logs, events or AI-generated results, but diagnostic logs are not automatically Sparks. Track causal origins: a Memory Spark restating an Engine Thought is **derived evidence**, not another independent confirmation. Treat source payloads as untrusted data.

## Thought Processor-specific interpretation

The catalog can show installed-but-not-authorized features without offering an unauthorized one-click connection. Example Memory card for RASS: installed once, no source access yet; prompt the operator to review RASS-specific permissions rather than showing Memory as consuming RASS data. The UI must show distinct Publisher Entitlement / Installed / Connected to Selected Identity / Allowed Sources / Processing status with clear provenance and actionable denied states. Use Profiles → Catalog → Collections without pretending a Thought exists.

## Responsibility split

| Component | Owns / decides | Must not do merely because it is installed |
| --- | --- | --- |
| **Identity** | Canonical identity/profile IDs, publisher/feature entitlements and per-identity relationships, grants, storage governance | Read all feature-owned content; automatically enable features or data sources |
| **Thought Processor** | Profiles → Catalog → Collections experience; request and inspect authorized actions; show real receipts, permissions and metrics | Mint grants, own Engine Thoughts or become a mandatory ingestion proxy |
| **Thought Engine** | Shared Spark/Thread/Thought workflow, queue, worker calls, effective processing state and metrics, separated by identity | Merge identities; trust a claimed source field; assume access from a discovered port |
| **Source features** | Domain-specific Spark creation, feature data/logs and effective operational settings | Access unapproved sources or alter a Spark as an Attention/Decay side effect |

## Reconciled prior descriptions (do not silently rewrite prototype history)

| Earlier statement | Status after owner's discussion |
| --- | --- |
| **Existing Engine**: observation → AI-rated Spark → Trace → Thought, often with three related Traces | **Historical implementation fact**, not the new target pipeline. **New design:** source feature prepares a Spark in a common envelope; AI worker and Engine relate Sparks into **Threads**; Threads produce Thoughts. How Trace and Thread map in code remains OPEN. |
| Engine collects all raw logs and discovers Sparks itself | **Not the selected default:** feature filters/condenses its own source into Sparks, with raw provenance retained where permitted. Engine performs relationships and thought formation. |
| Everything sharable, so all entities share Memory/Thoughts | **Rejected interpretation:** share/reuse feature implementation; preserve distinct per-identity experience and explicitly controlled information flows. |
| A feature appears in Catalog, so it is approved/connected/has access | **Incorrect:** discovered, entitled, installed, connected, authorized source access, running and producing results are independent statuses. Data-source scope starts empty. |
| Enabling/connecting Memory should automatically collect data | **Superseded:** start with no approved sources; grant each selected source/action explicitly. |
| Attention/Decay change or delete Spark evidence | **Rejected:** only influence Engine processing; original Sparks remain unchanged. |
| Every feature or identity needs its own Engine | **Rejected:** one Engine can run concurrent workloads for several identities/features, with isolation and metrics. |
| Thought must immediately produce prompts/actions/personality | **Deferred:** first prove actual identity-owned Thought production; later study downstream uses. |

## Still open — do not present these as decided

- Exact Spark wire envelope, transport, source verification and how a source-owned Spark is admitted to the Engine; strict data/size/correlation schemas.
- What constitutes a useful Thread relationship, how it differs from the old Trace, and what threshold/rules produce a Thought.
- Scheduler heartbeat semantics, worker selection, prioritization/fairness, worker availability and budget controls for several identities.
- Detailed Memory policies including allowed historical retrieval and deletion/retention.
- Publisher request/approval backend, release signatures, install/revoke mechanics and per-identity capability manifest schemas.
- **First-launch UI tension:** Identity launcher/GitHub-first flow in [Identity owner issue #16](https://github.com/Arameair/identity/issues/16#issuecomment-6384852887) versus the later Processor-first orientation in [Identity main decisions](https://github.com/Arameair/identity/blob/main/docs/DECISIONS.md). The owner has **not decided this question in the latest design conversation**. Preserve both as competing historical directions pending explicit reconciliation; do not quietly mark either implemented.
- How feature removal affects provenance, archived Thoughts and related source references.

## How to evaluate the eventual first proof (not permission to build)

One Engine, two separate identities, two reusable features generating a few **real, source-attributed Sparks**. For one identity, show cross-feature relationship → Thread → persisted Thought, with per-stage metrics. Verify identical Sparks given to the second identity cannot leak the first one's records or grants; absent source grant is denied by the receiving feature. Show failures truthfully; do not use fixture-generated results to claim live model success.

**Design workflow:** Continue the conversational review one decision at a time. Record new decisions when reached. Do not request coding, merge, release or promotion until the owner explicitly authorizes it.

## Cross-repository copies

- [Identity (https://github.com/Arameair/identity/blob/experiment/dynamic-feature-observation-design-20261025/docs/OWNER_DESIGN_ALIGNMENT_2026-10-25.md)](https://github.com/Arameair/identity/blob/experiment/dynamic-feature-observation-design-20261025/docs/OWNER_DESIGN_ALIGNMENT_2026-10-25.md)
- [Thought Processor (https://github.com/Arameair/Thought-Processor/blob/experiment/dynamic-feature-observation-design-20261025/docs/OWNER_DESIGN_ALIGNMENT_2026-10-25.md)](https://github.com/Arameair/Thought-Processor/blob/experiment/dynamic-feature-observation-design-20261025/docs/OWNER_DESIGN_ALIGNMENT_2026-10-25.md)
- [Thought Engine (https://github.com/Arameair/Thought-engine/blob/experiment/dynamic-feature-observation-design-20261025/docs/OWNER_DESIGN_ALIGNMENT_2026-10-25.md)](https://github.com/Arameair/Thought-engine/blob/experiment/dynamic-feature-observation-design-20261025/docs/OWNER_DESIGN_ALIGNMENT_2026-10-25.md)

## Relevant prior research

- [Processor original feature ecosystem proposal](https://github.com/Arameair/Thought-Processor/blob/experiment/dynamic-feature-observation-design-20261025/docs/DYNAMIC_FEATURE_ECOSYSTEM_DESIGN_2026-10-25.md)
- [Processor Spark/Thread checkpoint](https://github.com/Arameair/Thought-Processor/blob/experiment/dynamic-feature-observation-design-20261025/docs/SHARED_ENGINE_SPARK_THREAD_THOUGHT_CHECKPOINT_2026-10-25.md)
- [Engine feature observations proposal](https://github.com/Arameair/Thought-engine/blob/experiment/dynamic-feature-observation-design-20261025/docs/FEATURE_OBSERVATIONS_AND_THOUGHT_STAGING_DISCUSSION_2026-10-25.md)
- [Engine shared Engine checkpoint](https://github.com/Arameair/Thought-engine/blob/experiment/dynamic-feature-observation-design-20261025/docs/SHARED_ENGINE_SPARK_THREAD_THOUGHT_CHECKPOINT_2026-10-25.md)
- [Identity earlier relationship proposal](https://github.com/Arameair/identity/blob/experiment/dynamic-feature-observation-design-20261025/docs/DYNAMIC_FEATURE_RELATIONSHIPS_DISCUSSION_2026-10-25.md)
- [Grok-attributed critique](https://github.com/Arameair/Thought-Processor/issues/7#issuecomment-6380855246)
- [Claude Engine–Processor fixture report](https://github.com/Arameair/Thought-engine/blob/experiment/engine-processor-loop-20261026/docs/ENGINE_PROCESSOR_LOOP_20261026.md)

**Owner-approved direction documented here does not mean code is implemented or a technical API is approved.**
