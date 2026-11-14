> **Historical focus note (2026-10-26):** This page describes an earlier **Thought Engine-focused** Processor UI. The owner's later goal is a **system-wide operator workspace** (profiles, feature catalog, results/collections and diagnostics). Thought Processing remains an optional feature *within* that workspace. For current scope, see [README](../README.md) and [Design readiness](DESIGN_READINESS_2026-10-26.md). Earlier text is preserved to explain design evolution; it is not a competing product decision.

---

# Vision — a real feature control point

**Status:** Design notes, not working software. Read [STATUS](STATUS.md) before treating a capability as implemented.

## Customer promise

Thought Processing is an **optional, independently enabled feature** that helps an identity develop evidence-backed thoughts from logs and observations **the owner explicitly chooses to share with it**.

Thought Processor is the place the owner operates that feature. Not just a pretty viewer:
- discover and connect a compatible Thought Engine;
- see exactly which identities have the feature, and why;
- select and pause **only this feature's consumption** of individual permitted feeds;
- configure processing policies, budgets, queue behavior and available worker capabilities;
- request a single processing step, pause/resume processing, and inspect why something happened;
- see real health, provenance, evidence, worker response, error, timing and persistence results.

## The feature is the product

Capabilities should have explicit contracts, independently understandable status, and separate entitlement/lifecycle. An installation can run ScreenBridge, Discord or other source features without Thought Processing. Turning Thought Processing off must not stop those features, erase their information, delete a repository, or silently remove saved thoughts.

**Feature states must not be collapsed:** Not installed; installed but disabled; enabled but no feeds; feeds connected but no worker; running and ready; degraded; paused; failed. Show what is actually happening.

## The separation

| Owner | Responsible for | Explicitly NOT responsible for |
|---|---|---|
| Identity | Real identities; ownership, entitlements, feature permission grants and revocations | Doing the thinking |
| Log-producing features | Producing their own logs/observations and authorizing use of a specific feed | Creating or owning Thought records |
| Thought Engine | Ingesting permitted observations, queueing, Spark/Trace/Thought decisions, persistence, model routing within its feature | GitHub login, running unrelated services, managing all feature entitlements |
| Thought Processor | **Control plane** and observation UI for Thought Processing | Maintaining a second private thought database; claiming simulated actions are real |
| Worker adapter | Specific model/provider readiness, one request/result, metrics and failure classification | Deciding product entitlements or permanently owning thoughts |

The boundary is intentionally small at first: **one local engine, one local worker, one identity, one log feed**. Prove the path end-to-end before multiplying providers and identities.

## Owner experience

An owner enables Thought Processing in Identity for TASS, visits Thought Processor, sees that grant and a verified engine connection, chooses one or more permitted log feeds, and observes the pipeline:
`source line → accepted observation → queued → model/rule decision → Spark → Trace → Thought → saved`.

For every hop show **why**, not just a green LED. The owner can disable processing without deleting the source application's logs or Thought history.

## Truthfulness

Do not represent a port answering HTTP 200 as proof that the engine works; do not label fixtures as real-model results; do not mark a grant as active solely because a toggle's visual state changed. A control is successful only when the authoritative service acknowledges and persistence or runtime change is verified. 