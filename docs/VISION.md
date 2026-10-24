# Project direction

**Status: owner direction synthesized into documentation; no executable in this repository.**  
**Purpose:** a clear product narrative, not a specification of already-running software.

## Why this exists

The larger companion vision is a system whose usefulness grows by adding and improving **independent features**. Thought Processing is one optional feature: an identity may have access to it, and permitted sources may supply logs or observations that help develop sustained, evidence-backed thoughts. A customer should not need every feature to use any one feature.

**Thought Processor is the control point**, not another thinker. It makes the relationship between applications understandable and operable: which real programs are running, which connections exist, what an engine requested, what a worker returned, and what actually changed.

This fits the longer-term [companion/dynamic-coding vision](https://github.com/Arameair/Comp-idea/blob/main/docs/DYNAMIC_CODING_VISION.md), but **Thought Processor is not the dynamic-coding/update system** and is not authorized to rewrite or deploy other features.

## What the experience should feel like

An owner opens a calm workspace of lightweight **live application cards**. The Identity application, Thought Engine, and available workers may appear as separately verified running instances. The owner can inspect a connection or expand one real application without needing to rebuild it in Processor.

For Thought Processing: the owner enables the feature through Identity, approves a source feed, and sees the Engine receive events. The Engine handles processing and persistence. The Processor explains the chain and the failure point: **source → observation → Engine decision → worker API call → response → saved evidence**. If a stage is not measured or not authorized, show that instead of inventing a green state.

## Non-negotiable boundaries

| Principle | Implication |
|---|---|
| Features stay independent | Disconnecting Thought Processing does not stop Discord, TASS, ScreenBridge or their independent logging |
| Identity owns authority | Processor cannot create effective grants merely by changing a toggle |
| Engine owns cognitive state | No shadow Thought store or duplicate Spark/Trace histories in Processor |
| Models are interchangeable | Chat-Jimmy is today's fast-worker prototype; future replacements shouldn't require migrating Thoughts |
| Operator controls are real | Every Connect, Pause or Process action needs a real service acknowledgment and observable outcome |
| No simulated live state | Fixtures belong in explicitly labeled tests, never hidden in customer-facing status |
| Owner controls promotion | Experiments start from Alpha; Stable and Main do not move without explicit approval |

## Intended scope, not every app's router

The Processor assists **service discovery, connections and observability** across compatible companion components. The Engine receives and routes **thought-related observations** and owns its reasoning/worker-call decisions. Ordinary operations such as GitHub login or running Discord do not have to pass through the Thought Engine.

Discovery does not necessarily proxy traffic. After the Processor helps select a compatible worker, the Engine may call that worker directly; Processor follows honest telemetry. That distinction keeps the system fast and independently testable.

## Deliberately outside the first prototype

A complete feature marketplace or billing service; remote/unauthenticated administration; universal process lifecycle management; automatic ingestion of every log; a second Thought Engine; automated code changes; replacing all existing UIs; and high-volume live thumbnails of every app.

The first value to prove is **an accurate, useful control point for two or three real independently running services**, not a large decorative dashboard.
