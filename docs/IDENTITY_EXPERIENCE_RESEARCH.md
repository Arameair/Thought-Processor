# Identity–Experience Hypothesis: design review and test plan

**Research note · 24 October 2026 · Not proven or implemented**  
**Companion architecture:** [Identity-first model](https://github.com/Arameair/identity/blob/main/docs/IDENTITY_FIRST_MODEL.md)  
**Discussion:** [Architecture review open to Grok and other critical perspectives](https://github.com/Arameair/Thought-Processor/issues/7)

## The proposed theory

> A companion supplied with stable identity/purpose context **and** selectively retrieved relevant experience will make more coherent, contextually appropriate decisions over repeated sessions than an otherwise equivalent companion supplied only with recent observations.

This is a **falsifiable working hypothesis**, not an established cognitive theory, a claim of artificial self-awareness, or an argument that more infrastructure automatically means better behavior.

The system may have architectural benefits from Identity **even if the cognitive effect is zero**. Do not confuse secure service identification and independent module replacement with improved companion intelligence.

## Responsibility model

| Participant | Authority | Must not implicitly own |
| --- | --- | --- |
| **Identity** | Durable entity **and feature identities**, trust, capabilities, relationships, grants, possible stable purpose metadata (extent open) | All live observations, thought state, AI calls, memory contents |
| **Thought Processor** | Discovery, compatible connection control, honest health/status, request tracing, developer UI | Every inference request; service credentials, grant issuance, owner of thought data |
| **Thought Engine** | Observation interpretation, request timing, Sparks/Traces/Thoughts, bounded context assembly and processing decisions | System-wide entitlement authority; all cross-feature secret stores |
| **Memory (future)** | Selected durable experiences and retrieval, if independent memory proves useful | Service identity/grant authority; unconstrained raw ingestion by default |
| **AI workers** | Replaceable inference ability | Trusted data provenance, permission decisions, unquestioned write authority |

**Thought Processor should not become the cognitive engine.** The Engine may make direct requests to authorized workers after discovery; Processor learns the request lifecycle through correlated metadata.

## Grok critique supplied by the owner

The following is a **summary of the user's pasted Grok opinion**, not code-audited findings. Grok did not directly connect to GitHub through this review.

**What Grok found promising**
- Separate Identity, Thought Engine, and independent source features.
- Lightweight Processor controls/diagnostics rather than a universal gateway.
- Local persisted state and explicit refusal to report invented model success in experiments.

**What Grok challenged**
- Duplicated 'Thought Processing enabled' state in Identity vs Engine; conflicting grant authority until a real handshake exists.
- Local service discovery without authentication and unprotected ingestion endpoints.
- Expensive identity context + memory retrieval on every observation.
- Observation queues exceeding bounds, single-threaded workers, oversized prompts, sensitive unbounded diagnostics.
- Lack of tests showing that identity context improves coherence beyond a simple persistent summary.

**Grok's alternative**
- Treat a narrow Memory service as the continuity/retrieval layer; keep Identity for durable IDs, purposes and grants, Engine for transformations, Processor for read-only visibility/control.
- Do not require full identity-relationship graph construction before the first observation is processed.
- Use narrow locally verifiable authorization at the receiving service.

**Our assessment / adjustments to investigate**
- **Accept the concern** about duplicated grant flags: the feature's local enable switch may reflect operational state, but Identity should be the eventual grant authority. The real receiving service needs verifiable allow/deny enforcement.
- **Accept the memory distinction as a promising experiment**, not an immediate forced migration of Thought Engine records.
- **Retain individual feature identities** as an architectural goal, but do not demand a complete graph before V1.
- **Correct the simplified security framing:** an asserted or signed entity ID alone is insufficient unless issuer, audience, subject, scopes, expiry and caller authenticity are verified against the requested operation.
- **Audit Grok's code-specific claims** against exact branch/SHA, concrete logs and runtime tests before treating them as facts.

**No Grok viewpoint is represented as an owner decision.** Additional and opposing critiques are welcome on [Issue #7](https://github.com/Arameair/Thought-Processor/issues/7).

## Fair experiment: separate the two variables

Run the same observation sequence through the **same model**, approximately equal context/token budgets, comparable retrieval time allowance and identical authorized input data. Keep authorization protections constant; cognitive comparisons do not justify unsecured API calls.

| Variant | Stable identity/purpose context | Relevant long-term memory retrieval |
| --- | --- | --- |
| **A — baseline** | No additional purpose profile beyond task-neutral instructions | No; recent-context window only |
| **B — purpose** | Yes | No |
| **C — memory** | No | Yes |
| **D — combined** | Yes | Yes |

Evaluation questions:
1. **Delayed recall** — Does the companion correctly recall verified facts from earlier sessions?
2. **Consistency** — How often does it contradict known, previously established facts?
3. **Purpose alignment** — Does it prioritize events relevant to the assigned role without fabricating relevance?
4. **Relevance and timing** — Does it respond usefully, avoid interruptions, and decline to speak when unnecessary?
5. **Performance** — What do assembly time, tokens, model latency, queue lag, memory reads and total cost add?
6. **Reproducibility** — Can blind evaluation reproduce the difference across multiple observation sequences and seeds?

**Interpretation limits:** If D doesn't beat C on measured outcomes, identity purpose may add no meaningful *cognitive* value at this scale; Identity may still be essential for trust, feature identity, authorization and independent module evolution. The experiment cannot prove consciousness or long-term general intelligence. A single small test only gives preliminary evidence.

## First minimal end-to-end technical demonstration (different from cognitive test)

- Start Identity, Thought Processor and one compatible test feature **independently**, without Project Controller.
- Identify durable entity and **feature service** identities separately from runtime instance IDs/ports.
- Discover a feature automatically but label it **unverified/unpaired** and deny privileged access.
- Pair once through Identity; request and verify one scoped grant.
- Send one real authorized observation to Engine, call a real worker, persist a result, and correlate sanitized distributed request IDs.
- Restart services in varying order; confirm automatic verified reconnection, stale-state honesty and no accidental privilege expansion.
- Stop Identity, observe short-lived already-approved access expiring correctly. Stop Processor and confirm other services continue where allowed.

**This integration check does not prove the cognitive theory.** The research comparison doesn't by itself prove the integration is secure. Keep them separate.

## Decisions awaiting the owner

1. Should independent feature identities be organized by **ownership hierarchy** or **explicit permissioned relationships**? No final selection.
2. Does Identity encode only stable purpose, or also changing goals/self-model? No final selection.
3. What is the minimal service-identity, installation-identity and instance-identity schema?
4. When multiple workers advertise the same capability, who applies selection policy: Identity, Processor or Engine?
5. Does Memory become a separate physical service or begin as a clearly owned retrieval component within Engine?
6. Protocol-only versus protocol plus optional helper libraries—still open.
7. Exact authorization lifetime, cryptographic grant scheme, cross-platform discovery details and diagnostic retention.

## Development policy and authorship

[Branch workflow](WORKFLOW.md): `alpha` is owner-controlled; `stable` is accepted evidence; `main` is the documentation/public-facing lane. **No automatic merges or promotions.** Product decisions and experimental observations require owner consent and direct evidence.

This note was written with ChatGPT from owner-established design directions and a **user-provided Grok response**. It is an invitation to scrutiny, not an assertion that either AI independently tested the running implementation.
