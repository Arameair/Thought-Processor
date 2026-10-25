> **DESIGN RECONCILIATION — later owner decisions (2026-10-25):** Read [Owner Design Alignment](OWNER_DESIGN_ALIGNMENT_2026-10-25.md) first. It supersedes conflicting **proposals** below: newly created identities begin without optional features or approved data sources; an installed feature can be reused by multiple identities but access to each source starts empty and requires explicit Identity-profile authorization; source features prepare common-format Sparks from selected information, Engine/worker relate Sparks into **Threads** then identity-owned Thoughts; Attention/Decay are optional processing features and never rewrite Spark evidence. Existing Engine `Spark → Trace → Thought` describes the old prototype, not adopted Thread semantics. This document remains useful historical research; unresolved first-launch UI, protocols and Memory details remain OPEN. No code/build/promotion authorized.

# Design checkpoint — shared features, Sparks, Threads and identity-owned Thoughts

**2026-10-25 · Owner + ChatGPT discussion · DESIGN ONLY**

This checkpoint updates the conceptual direction in [Dynamic Feature Ecosystem Design](DYNAMIC_FEATURE_ECOSYSTEM_DESIGN_2026-10-25.md). It is a **discussion record**, not a new implementation assignment, settled protocol, tested behavior, or permission to promote a branch.

## What we have agreed to work toward

1. **Profiles → Catalog → Collections** remains the user-facing Thought Processor model:
   - **Profiles**: TASS, RASS, Tom, Mike, a business, or any other identity the administrator works with. Names are labels, not special engine categories.
   - **Catalog**: discoverable, independently developed, reusable features such as Memory, Cognition, Game Awareness/Perception and Communication. Each can have its own repository, setup, configurations and versioned implementation.
   - **Collections**: the selected identity's actual Thoughts and other feature-owned records, never fabricated UI data.
2. **One Thought Engine can serve multiple identities and many features concurrently.** We do not require one Engine process per companion, person or feature. A feature implementation can be shared and configured differently in each identity's context. Identity is the authority for canonical IDs and relationships.
3. **Features produce Sparks**, which may be derived from selected activity, logs, domain observations or other outputs. A Memory Spark, game Spark and Cognition Spark are all candidate inputs to the same Thought Engine.
4. **An AI worker helps the Engine relate/process Sparks into Threads.** Threads may combine **related Sparks from different source features**. The Engine accumulates/relates Threads into **Thoughts owned by the target identity**.
5. **What happens after producing a Thought is deliberately OUT OF SCOPE for now.** First prove a repeatable Spark → Thread → Thought pipeline; only then study Thought coherence, purpose, memory use, action selection or expression.
6. **Instrumentation from the first proof**: observable and attributable timing, throughput, queue wait, worker identity and call result, resource use, formation success/failure, and per-identity/per-feature breakdowns. Improve speed and quality step by step using controlled comparisons, not assumed gains.
7. **We are designing, not building.** Work one concept/feature at a time, update repository notes when owner and collaborator reach a meaningful checkpoint, do not start code work by implication.

## Proposed mental model (not an API)

```text
                           ONE SHARED THOUGHT ENGINE
Feature implementations             |               For several identities
(each may be a separate repo)        |
   Memory --- Spark (TASS) ----------+
   Game ----- Spark (TASS) ----------+--> worker-assisted Threads --> TASS Thoughts
   Cognition  Spark (TASS) ----------+
                                     |
   Memory --- Spark (RASS) ----------+
   Game ----- Spark (RASS) ----------+--> worker-assisted Threads --> RASS Thoughts
                                     |
   Business feature Spark (Business)-+--> worker-assisted Threads --> Business Thoughts
```

The same feature can serve several identities. Identity-specific configuration and the information processed—not the name—help differentiate those identities' Thought collections.

**Terminology:** The existing Thought Engine prototype calls its middle stage **Trace**, and uses Spark → Trace → Thought. The owner is now describing a **Thread** made by AI-worker-assisted relationships among Sparks, potentially cross-feature. Whether to rename Trace, define Thread as distinct from Trace, or adapt the implementation is an **open design decision**. Do not quietly equate them or rewrite historical code claims.

## Heartbeat and worker distinction

Thought Engine already has an optional timed heartbeat for advancing active Thoughts, while its event processing loop handles queued observations. **Chat-Jimmy is presently the worker**, which the Engine checks using Chat-Jimmy's separate `/api/heartbeat` readiness endpoint; the AI worker is not itself the engine's scheduling heartbeat. Future designs can schedule independent feature workloads and use different worker capabilities, while keeping the Engine shared. Specific scheduling priorities/frequencies are NOT decided yet.

## Important boundaries preserved without prematurely solving them

- A Thought belongs to the **target identity**; sharing an Engine or feature code does **not** automatically combine unrelated identities' observations. Cross-identity source-data sharing and Memory access control deserve their own focused feature design later.
- Identity remains authoritative for identities and scoped authorization; Processor is the human control/view; Engine owns the thought-processing state and its receipts; source features own their own raw logs/data.
- “Everything can be shared” means **reusability and configurable, authorized relationships**, not default unrestricted copying of all personal or business data.
- Cross-feature feedback loops need provenance/dedup: a Memory Spark that merely quotes a prior Engine Thought must not be counted as new independent evidence.

## First future measurable proof (discussion target, NOT a build request)

With one Engine, create two distinct identity contexts using a shared feature implementation. For one identity, submit a bounded handful of Sparks from two different named features; demonstrate that the worker forms a Thread from genuinely related evidence and the Engine produces a Thought owned only by that identity. Run a separate stream for the second identity and verify no accidental mixing. Show complete processing receipts, elapsed times, and worker-call counts. Then repeat exactly to compare improvements.

**Open next discussion**: What properties define a Spark at intake, and what constitutes a genuine relationship strong enough to form a Thread? Do not decide downstream Thought behavior yet.

Related Engine checkpoint: [Shared Engine Spark/Thread/Thought processing](https://github.com/Arameair/Thought-engine/blob/experiment/dynamic-feature-observation-design-20261025/docs/SHARED_ENGINE_SPARK_THREAD_THOUGHT_CHECKPOINT_2026-10-25.md).

### External review references (not approval)

- [Grok-attributed Processor critique](https://github.com/Arameair/Thought-Processor/issues/7#issuecomment-6380855246) stresses real grants and real receipts over a persuasive mock UI.
- [Claude Engine ↔ Processor local fixture report](https://github.com/Arameair/Thought-engine/blob/experiment/engine-processor-loop-20261026/docs/ENGINE_PROCESSOR_LOOP_20261026.md) tested process connections with a labelled non-AI fixture worker, not verified live cross-feature Threads.

**No Alpha, Stable or Main change. No application code or runtime experiment performed by this checkpoint.**
