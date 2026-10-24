# Interfaces — proposed boundaries (not yet implemented)

**All examples in this file are design proposals. No API, authentication scheme or status contract is claimed to exist in Thought-Processor yet.**

## Identity → feature authority

Questions Processor and Engine must be able to answer:
- Which immutable identity does this action apply to?
- Is `thought.processing` installed, granted and enabled for that identity?
- Which sources are individually authorized for this feature?
- Who changed a grant, and when? Can the engine verify it **at action time**?
- What is the effective revocation behavior when Identity is unreachable?

Proposed default: deny new privileged processing when a grant cannot be verified, preserve existing history, make outage/revocation visible. Define offline grace rules explicitly before deployment.

## Feed → Thought Engine

Possible normalized event shape (draft):

```json
{
  "schema": "thought.observation.v1-DRAFT",
  "identity_id": "<immutable identity UUID>",
  "source_id": "tass-wow-addon",
  "event_id": "<stable deduplication ID>",
  "observed_at": "<RFC3339 timestamp>",
  "content_type": "text/plain",
  "text": "Player entered Goldshire",
  "context": { "session_id": "<optional>" }
}
```

Before an event is admitted: authorization, source/identity binding, length limits, deduplication, queue capacity, provenance and retention policy. Acknowledgement must mean **persisted/queued** vs **processed**, not conflate them. Avoid copying high-volume source logs indiscriminately; support bounded batching, drop/lag counters and policy-based filtering.

## Thought Engine → worker

This is a **capability interface**, not a Chat-Jimmy hardcoded interface. Initial capabilities could include:
- `score_relevance`: bounded observation → score + reason + topic suggestions
- `match_topic`: observation plus known topic IDs → existing or proposed topic
- `form_question`: linked evidence → candidate open question
- `develop_thought`: thought plus evidence and notes → proposed next note and state
- `summarize_context`: bounded text/logs → useful extraction

Each call needs a request ID, identity-specific context boundary, operation/capability, approved provider/adapter, maximum input/output, timeout, expected output schema, and status: success / busy / blocked / unavailable / timeout / uncertain / unreadable. Responses include raw text, structured validated result (if any), actual model/mode, elapsed timings and cost/usage when available.

**Routing policy (proposal):** Thought Engine requests the capability. A replaceable worker registry selects an approved provider. Workers cannot directly grant features, create identities, delete repository data or rewrite Thought history. Preserve evidence and provenance so models can be swapped without loss.

Chat-Jimmy already has its own separate heartbeat and one-request API. The adapter may translate that API into this contract but must not leak Chat-Jimmy-specific assumptions into all future engines.

## Processor → Engine

Distinct operations are needed for:
1. **Read:** actual health, protocol/build/identity, queue and worker state, feed subscriptions, current Thoughts, traces, events and provenance, metrics, diagnostics.
2. **Commands:** select authorized feed, pause/resume feature processing, process one, safely retry an eligible event, update allowed policy/worker choices, change a Thought state via Engine validation.
3. **Receipts:** each control action returns operation ID, authoritative result, error if refused, changed state and timestamp. UI verifies observable impact. Do not trust an HTTP 200 from the dashboard as evidence that an engine-side operation succeeded.

## Security

Local-only HTTP and Origin checks reduce exposure but are not authentication. No source is authorized just because it responds on a port. A real cross-process identity and grant model needs authenticated calls, replay protection, scoped permissions and secure token handling. Never put secrets in Copy Context.

## Versioning and fallback

Protocols must advertise schema/capabilities. Unknown versions result in **INCOMPATIBLE** with a clear reason. Missing worker becomes a bounded pending queue or an explicit refusal, never invented model output. A changed source/build needs explicit visibility, not silent reassociation.
