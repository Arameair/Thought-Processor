# Architecture and ownership

**Design target, not an implemented integration.** Existing repositories are separate prototypes. Their interfaces and data formats are not automatically compatible.

## Component map

```text
                                         Identity
                                identity ID + feature grant
                                           |
Independent features                      v
TASS logs / screen / Discord ── approved feed ──> Thought Engine
                                                  queue, Spark/Trace/Thought,
                                                  policy, timing and persistence
                                                      |           ^
             discovery + approved connection           |           |
Thought Processor ─────────────────────────────────────┘           |
  dynamic verified cards / diagnostics / flow                       |
  finds and validates worker services                              |
             |                                                      |
             └── worker endpoint selection ───> replaceable worker ┘
                                                 (Chat-Jimmy today)
```

**Model calls can be direct from Engine to worker.** The Processor is not required to forward every model request. For observability, Engine or worker telemetry must expose request IDs, statuses and measured timings. A disconnected Processor must not be described as if it captured traffic.

### Who owns what?

| Component | Authority | Not its job |
|---|---|---|
| **Project Controller** | Repository branch build, Play/Stop, run logs and side-by-side developer testing | Managing Thought grants, cognitive records or the Processor's catalog |
| **Identity** | Identity records, entitlements, permission grants, revocation and linked feature authorization | Thinking, model heartbeat timing or Processor cards |
| **Source features** | Their own activity, source logs and independent operation | Maintaining Thought records or controlling other applications |
| **Thought Engine** | Opted-in observation queue; importance/topic decisions; heartbeat/model requests; Sparks, Traces, Thoughts and durable history | Discovering every unrelated application or deciding customer entitlements |
| **Thought Processor** | Control plane for discovery, connection setup, verified status, interactive *supported* operations, cross-service diagnostics and app cards | Duplicating records, generating thoughts, automatically starting/stopping other processes or choosing the Engine's think cycle |
| **Worker adapter/provider** | Ready/capable state, bounded request/result, failures, timing | Identity grants, thought persistence or irreversible actions |

### Two directions of flow

**Information flow:** approved log source → Engine ingestion and queue → Spark / Trace / Thought. Identity gates feature usage and source authorization. Sources need not know how the Engine thinks.

**Control/visibility flow:** Processor discovers and validates compatible running apps, helps configure approved connections and displays authentic status. Engine owns the **actual decision to make a worker call**. Processor can request only operations that Engine supports and authorizes.

A heartbeat in this project means an **actual worker API request by the Engine**. It does not mean a cosmetic UI pulse or a timer owned by Processor.

## Discovery is not permission

Port availability and GitHub branch name are clues, **not proof** of service identity or authorization. A compatible service should expose protocol version, service kind, instance ID, capabilities and verified address; build/branch/SHA only when observed. Authentication, access scopes and revocation are still unresolved.

A card represents an **application runtime instance**, not a per-identity avatar. A stopped service may remain visible as clearly stopped if previously registered; no cached image labeled live. If preview embedding is prohibited, offer the original application UI separately without bypassing browser restrictions.

## Multiple prototypes — one authority at a time

- [Thought](https://github.com/Arameair/Thought/tree/experiment/spark-trace-thought-v1-20261023) is a deterministic prototype with synthetic events and TEST permissions.
- [Thought-engine](https://github.com/Arameair/Thought-engine/tree/experiment/thought-engine-v1-20261023) is a separate prototype using model-assisted observations and a Chat-Jimmy-specific worker interface; its live model behavior on the owner's system has not been verified here.
- [Thought-Controller](https://github.com/Arameair/Thought-Controller/tree/experiment/real-thought-dashboard-20261023) is an earlier dashboard prototype, **not automatically renamed, ported or merged** into Processor.

**Do not direct one identity's writes at two competing stores.** Choose and document a canonical Engine and migration plan with owner approval before integrating cognitive state.

## Failure and lifecycle rules

- No Engine: Processor reports **NOT CONNECTED**, not stale data as live.
- No authorized feed: the Engine must not ingest that source's data.
- No worker: show unavailable/pending or explicit rule-based behavior; never fabricated model answers.
- Busy, blocked, timeout or uncertain: show actual result, queue state and safe retry policy; don't resubmit uncertain calls automatically.
- Processor Stop: independently running sources, Engine and worker remain running.
- Identity grant OFF: stop *feature* processing according to an explicit policy, preserving existing history; do not stop source applications.

Interfaces and trust contracts remain **drafts** in [INTERFACES](INTERFACES.md).
