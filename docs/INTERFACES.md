# Draft service contracts — not yet implemented here

**These are possible shapes for independent components, not existing endpoints or agreed protocols.** Processor must discover compatible services through validated metadata, not merely scan open ports or GitHub repository names.

## 1. Application discovery and card state

A possible *read-only* advertisement needs:
- a stable runtime **instance ID** and service **kind** (e.g. identity, thought-engine, worker);
- protocol/schema version and supported capabilities;
- endpoint and current state plus **checked-at** timestamp;
- observed branch, full build SHA, PID and run ID **when the service can report them**;
- optional safe original UI endpoint, whether embedding is supported, diagnostics endpoint;
- explicit authorization and control capabilities, kept separate from discovery.

A selected card may display the original UI under allowed browser/session policies. **No inference of authorization** from an app responding on localhost. Unrecognized or incompatible versions should be labeled as such.

## 2. Source feature → Thought Engine

**Example event envelope (proposal only):**

```json
{
  "schema": "thought.observation.v1-DRAFT",
  "identity_id": "<immutable identity identifier>",
  "source_id": "tass-wow-addon",
  "event_id": "<deduplication identifier>",
  "observed_at": "<RFC3339 timestamp>",
  "content_type": "text/plain",
  "text": "Player entered Goldshire"
}
```

A receipt must distinguish **accepted/queued** from **actually processed/persisted as Thought**. Per-feed permission, identity isolation, bounded input/queue, dedup, observed vs received timestamps, retention and backpressure are prerequisites for general log ingestion.

## 3. Processor-assisted discovery → Engine → worker

1. Engine requests a compatible **capability**, not a hard-coded provider brand.
2. Processor's connection/discovery mechanism returns an approved, verified endpoint and capability/version information (exact handshake and authority are **OPEN**).
3. **Engine calls the worker directly** with its chosen schedule/policy. The API call is the Engine's **heartbeat** in the current design discussion.
4. Engine records request ID, selected worker/provider, mode, prompt/evidence references, time sent, actual worker reply/status, validation result and durable consequences.
5. Processor presents **Engine-reported** telemetry/trace; it does not claim to intercept traffic that bypassed it.

Status dimensions should distinguish **not configured, not running, incompatible, unauthorized, ready, busy, blocked, failed, uncertain**. A timeout after submit may mean the worker was contacted; do **not automatically duplicate the request**.

## 4. Processor → authoritative service controls

Read: service readiness, verified identity/capabilities, discovered links, logs/metrics and Engine evidence.

Write (where authorized): select a service/feed, approve/revoke a **connection** within granted scopes, request an Engine-supported processing action, or pause the Thought Processing subscription. Must return a request ID, accepted/refused result, owning component, and observable effect. Processor cannot change an Identity grant on its own or silently start/stop an independent service.

## 5. Security, privacy and performance

Local HTTP and same-origin checks are **not** authentication. Design genuine service authorization and grant revocation before presenting cross-process security as real. Logs may contain sensitive content; default Copy Context to redacted metadata. Processor should keep only bounded diagnostic indexes if approved, **not its own copy of the Engine's Thought database**. Use lightweight status streams/polling for cards, only one active original UI preview.

## 6. Version and provenance

Each experimental adapter must pin the **actual API/schema version it implements** and record cross-repo SHAs. If a component is too old or lacks a capability, report **INCOMPATIBLE**, not pretend to support it. Record whether response data was real, rule-derived, or fixture-tested.

See [Architecture](ARCHITECTURE.md), [Decisions](DECISIONS.md) and [Control point](CONTROL_POINT.md).
