# Architecture — control plane separate from evidence

**Design target, not an implemented protocol.** Existing repositories implement different experiments. These notes do not assume they already interoperate.

```text
   Identity ──(feature grant, identity ID, revocation)─────┐
                                                         v
Log sources (independent features) ──approved feed──> Thought Engine
  TASS / game / screen / Discord / app logs             | queue
                                                         | Spark/Trace/Thought
                                                         | persistent evidence
                                                         | worker requests
                                                         v
                                                   Worker interface
                                                (Chat-Jimmy for now)
                                                         ^
                                                         |
Thought Processor (control point)  <── verified status / evidence / commands ──> Engine
    discovery | grants | feeds | queue | worker | history | logs | metrics
```

## Current Identity-first architecture refinement (24 October 2026)

[Identity-first model](https://github.com/Arameair/identity/blob/main/docs/IDENTITY_FIRST_MODEL.md) · [Identity–Experience research and Grok critique](IDENTITY_EXPERIENCE_RESEARCH.md) · [multi-model review issue](https://github.com/Arameair/Thought-Processor/issues/7)

The owner clarified that **every independent feature has an identity** and that **Identity is the trusted system-level knowledge/policy authority**. Processor's job is to reconcile durable Identity-owned participant/relationship records with **which authorized service instance is actually running now**. It performs dynamic discovery, authorized connection assistance, status/trace correlation and real application previews. It **must not own Identity relationships, issue grants, require Project Controller or proxy every worker API request**.

The Thought Engine owns actual observation processing, Sparks/Traces/Thoughts and decisions to call replaceable AI workers directly when authorized. Memory's future role as a separate continuity/retrieval layer remains an experimental question. This refinement supplements the previous conceptual diagram below, which should be read as a **proposed flow**, not an implemented protocol.

**Selected behavioral design:** small core + optional capability interfaces; automatic local discovery plus configured/manual remote fallback; one-time Identity-approved pairing; scoped per-identity grants enforced by receiving services; short-lived bounded continuation during Identity outages; hybrid status reporting; distributed logs with shared correlation IDs; on-demand connection map and event timeline. **Wire format, complete identity graph, routing ownership, secure grant scheme and performance remain unproven/open.**

## Control plane

Thought Processor reads real state and sends explicit commands to the **authoritative component**. It must not invent local copies of remote grants or thought records as if they are the originals.

- **Identity** authenticates the actor and decides whether that identity may use the feature. A UI toggle in Processor cannot grant an unauthorized identity access.
- **Engine** verifies identity and feed permission on every ingestion or processing operation (not only on start). Persistence and per-identity isolation live with the engine.
- **Sources** expose or push information only through an explicitly approved feed. Thought Processor can disconnect its own subscription; it does not power off the source.
- **Worker** answers bounded prompts. The engine validates structure and evidence before acting; the model's suggested state transitions do not become authority.

## Data plane

An observation should retain a stable event ID, source ID, owner identity ID, observed timestamp, ingest timestamp, content/reference and optionally sequence/watermark. **Exact schema is a proposal**, see [INTERFACES](INTERFACES.md). Preserve original provenance with every derived Trace and Thought so the owner can inspect why it exists.

A log is not necessarily a Thought. The engine may buffer/deduplicate/filter first, and ask a worker only when interpretation is useful. Unavailable workers should not cause fake conclusions or automatic silent retries on uncertain sends.

## Connection discovery

A local port scan is **discovery only**. Confirm service identity, protocol version, installation identity and required capabilities; show whether the instance is external and who owns its lifecycle. Identity and Thought Processor must not kill, rebuild or take over an independently running Engine or worker merely because a port is open.

No hard-coded worker address assumptions. Discovery can start with explicitly configured loopback ports or local registration files. Network authentication and secure identity-to-engine grants must be designed before multi-user or remote exposure. Loopback-only does not authenticate local callers.

## Isolation and lifecycle

- Many identities can eventually use the **same engine process**, but their subscriptions, queues, histories, grants, budgets, and model context remain logically separated.
- Grant OFF: new processing refused; already saved history kept; decide explicit policy for retained vs discarded inbound events.
- Feed OFF: stop consuming **that feed only** for Thought Processing.
- Worker OFF: buffer bounded observations or use explicitly approved deterministic policies; do not mislabel a rule as model reasoning.
- Engine unavailable: Processor can report, retry *readiness checks*, and offer a user action; no covert process startup or duplicate process takeover.
- A change in build SHA, installation ID or engine instance is a potentially consequential *source change*; surface it, don't quietly trust it.

## Migration question

`Arameair/Thought` and `Arameair/Thought-engine` currently represent distinct approaches and storage models. **Do not point one identity at both as independent authorities.** Decide canonical implementation and a traceable migration process after live evidence. Nothing in this documentation declares one already selected.
