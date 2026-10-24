# Thought Processor

**The control point for the optional Thought Processing feature.**

Thought Processor is the operator's place to **configure, observe, test, and troubleshoot** Thought Processing for an identity. It is not the thought database, a particular AI model, an identity authority, or a global router for unrelated applications.

## AI collaboration and authorship

**This project is being developed through collaboration between its human owner and AI assistants.** The owner sets the goals, requirements, priorities and acceptance decisions. AI assistants have helped develop design ideas, architecture proposals and repository documentation, and may assist with future implementation, testing and review.

**AI-assisted content is not independently verified solely because it has been committed.** Designs and generated outputs can contain errors; functionality must be evaluated through actual builds, tests and observed behavior. No AI-assisted proposal is automatically an approved product decision or working feature.

See [AI collaboration and verification](docs/AI_COLLABORATION.md) for how this involvement is disclosed and reviewed.

> **Current status — FOUNDATION DOCUMENTATION ONLY (2026-10-23).** This repository does not contain a runnable Thought Processor yet. No real feature integration, source subscriptions, identity authorization, or model worker has been verified here. The appearance of a branch is not evidence of a working version.

## The product boundary

- **Identity** owns identity records, feature entitlements, grants and whether Thought Processing is enabled for a particular identity.
- **Source features** (TASS, ScreenBridge, logs, Discord, etc.) keep working independently. When permitted and individually connected, their logs/observations can be *fed into* Thought Processing. Disabling Thought Processing must not shut down the source application.
- **Thought Engine** owns the actual observation queue, Sparks, Traces, Thoughts, processing rules and their persistent history. Model answers are inputs, not an unchecked authority to rewrite history.
- **Thought Processor (this project)** is the **real control point**: connect/disconnect allowed feeds, inspect service readiness, manage processing policy and queue operations, observe decisions/evidence, select a suitable worker, see logs and metrics. Every control must perform a real operation and report its actual outcome.
- **AI workers** are interchangeable. Chat-Jimmy is the current candidate fast worker, *not* a permanent dependency, product identity, or place where thought history is stored.

**Feature-based product principle:** Thought Processing is a separable, optional capability. Installing a source feature does not imply Thought Processing access. Enabling Thought Processing does not automatically authorize every possible log source. A customer may buy/use one feature without the others.

## First mental model

```text
Identity: TASS
   └── Thought Processing grant: enabled / disabled
             |
Authorized, selected feeds (TASS logs / ScreenBridge / Discord / ...)
             |
             v
      Thought Engine  <---->  Replaceable worker interface
      observations              (Chat-Jimmy today)
      Sparks / Traces
      persistent Thoughts
             ^
             | operational status, controls, read-only views
             v
      Thought Processor
      CONTROL POINT: feeds, readiness, queues, policy, workers, diagnostics
```

Thought Engine is the destination for **thought-related incoming information**, not a mandatory router for GitHub authentication, Discord operation or every other system action.

## Start here

| Document | What it answers |
|---|---|
| [Current state and branch ledger](docs/STATUS.md) | What exists on Stable, Alpha and experiments, and what is *actually verified*? |
| [Vision and boundaries](docs/VISION.md) | What is the feature we intend to sell, and what doesn't belong here? |
| [Architecture and data ownership](docs/ARCHITECTURE.md) | What talks to what; who owns which records and permissions? |
| [Control surface and evidence](docs/CONTROL_POINT.md) | What must the operator see and control? |
| [Interfaces and contracts](docs/INTERFACES.md) | What a feed, engine and replaceable worker need to exchange |
| [Branch and promotion workflow](docs/WORKFLOW.md) | How experiments become Alpha, Stable and eventually Main |
| [Roadmap and gates](docs/ROADMAP.md) | What to prove next and in what order |
| [Decision log](docs/DECISIONS.md) | Decisions made, and questions still open |
| [AI collaboration and verification](docs/AI_COLLABORATION.md) | How humans and AI assistants contribute, and how changes are checked |

For every iteration: update the **status ledger**, use an issue for the goal, and link the experimental branch, exact commit, tests, limitations and owner decision. Do not quietly overwrite previous evidence.

## Branch policy

| Branch | Meaning | Promotion |
|---|---|---|
| `main` | Public-facing release lane *when approved*; the repository may still be private | Owner approval only |
| `stable` | Latest owner-accepted **reproducible snapshot** | Owner approval only; never automatic |
| `alpha` | Owner-tested integration candidate; may change as approved | Owner approval only |
| `experiment/<topic>-YYYYMMDD` | Isolated hypothesis/prototype; **always branch from current Alpha** | Validate, compare and ask owner; no auto-merge |

**Empty-repo bootstrap exception:** The first documentation commit must initialize `main`; `stable` and `alpha` can then be created at that *same documentation-only commit*. This does **not** count as a runnable release. From that point, **every experiment starts from `alpha`**. Experiments may cherry-pick or explicitly port code from other repos/branches, with provenance documented, but never silently replace Alpha.

## Related active experiments (not yet integrated into this repository)

- [Thought-engine — model-assisted Spark/Trace/Thought experiment](https://github.com/Arameair/Thought-engine/tree/experiment/thought-engine-v1-20261023) — separate experimental service; currently Chat-Jimmy-specific. Do not treat live website reasoning as verified on the owner's machine without tests.
- [Thought — deterministic Spark/Trace/Thought experiment](https://github.com/Arameair/Thought/tree/experiment/spark-trace-thought-v1-20261023) — different implementation; avoid two concurrent authoritative thought stores.
- [Thought-Controller — live engine dashboard experiment](https://github.com/Arameair/Thought-Controller/tree/experiment/real-thought-dashboard-20261023) — existing prototype; source for control UI requirements, **not** an automatic migration.
- [Identity](https://github.com/Arameair/identity) — future owner of real feature grants.
- [chat-jimmy](https://github.com/Arameair/chat-jimmy/tree/experiment/fast-clean-worker-20261023) — current independent candidate worker.

**Nothing has been merged, renamed, deployed or promoted by writing these notes.**

## Owner acceptance rule

Do not call a behavior working based on source code, screenshots, HTTP 200 on the UI, fake fixtures, or a green unit test alone. Demonstrate it with the actual running independent components; distinguish **process running**, **connected**, **authorized**, **data received**, **worker ready**, **processed**, and **persisted**. If any stage fails, say where and expose enough diagnostics to reproduce it.
