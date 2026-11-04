# Independent Go TASS integration baseline and security review

Source snapshots inspected 2026-11-04:
- Thought-Processor experiment/go-core-20261104 at 6f4f37fb96dc581799389427828d7bb9b32c991b.
- Thought-engine experiment/nemotron-link-20261104 at 9b55d1d107cc668e93c4a9313de802a50f98d3ff.
- Identity PR #31, 94393be115e5e97cb411d82b92b8f48d4b0adda3; OPEN / UNMERGED at review.
- Owner observation in TP issues #42 and #44 is not a repeat test by this engineer.

## Route and component ownership

Browser -> Go Thought-Processor core -> Go Thought Engine -> local llama.cpp/Nemotron
                |                          |
                |                          +-- existing Spark/Trace/Thought, append-only cycles ledger
                +-- policy-gated read-only tass.md counts -> Engine outcome -> evaluation
                +-- independent GitHub policy/TASS live/message sync with offline backoff
Identity static prototype policy -> core ~/.tass/identity-grant.json -> Engine authorizer.

Identity is not connected via live authorize API; GitHub is not required for local inference while the cached grant is valid. Go core uses HTTP to call Engine; Engine serializes Nemotron work with a model slot mutex.

### Actual endpoints

| Caller | Endpoint | Authorization | Response |
|---|---|---|---|
| Browser -> Core, 127.0.0.1:8795 | GET / | loopback Host only | HTML embeds fresh page token |
| Browser/CLI -> Core | GET /api/status | X-TASS-Token or Authorization Bearer, Host | 200 JSON policy_valid_now, busy, engine, cycles, github_online, events; 401 unauth |
| Browser/CLI -> Core | POST /api/say JSON {"text":"hello"} | same token, local Host, JSON Content-Type, local Origin if present | 202 EMPTY response (no request ID); 400 invalid, 401 bad token, 403 Origin, 415 media type |
| Core -> Engine, 127.0.0.1:7798 | GET /api/model | loopback Host; NO bearer | 200 last_pulse, busy, requests, failures, identity_policy, recent model IDs |
| Core -> Engine | POST /api/cycles | Engine bearer key, local Host, JSON; prototype policy checks thought.process and model.request | 200 Cycle awaiting_outcome with event/Spark/Trace and locked prediction; 403 denied, 503 worker failure |
| Core -> Engine | POST /api/cycles/{id}/outcome | Engine bearer key | 200 evaluated cycle; 404 missing, 409 already evaluated |
| Core -> Engine | GET /api/cycles or GET /api/cycles/{id} | Engine bearer key | 200 cycles or record; 401 unauth |
| Core -> Engine | POST /api/model/requests | Engine bearer key; model.request policy checked before and after slot wait | 200 model result, 403 denied, 409 duplicate, 503 model error |
| Local caller -> Engine | GET /api/model/requests/{id} | ONLY loopback Host | 200 FULL model result, including reply and thought, without Engine key |
| Engine -> llama.cpp, owner default 127.0.0.1:18083 | GET /health | local | 200 health, NOT evidence of model inference |
| Engine -> llama.cpp | POST /v1/chat/completions | local model client | 200 choices[0].message.content and optional reasoning_content |
| Legacy Engine state mutation | POST /api/identities, /api/entities/{eid}/grant, /api/entities/{eid}/events, /api/entities/{eid}/events/{evid}/retry, /api/entities/{eid}/thoughts/{tid}, /api/settings, /api/worker/check, /api/worker/scan | NO bearer, only Host, Origin and JSON checks | state changes possible from local clients |

Engine GET /api/status and GET /api/entities/{eid} are also unauthenticated. Model and cycle endpoints have distinct bearer requirements. Current primary Go Engine/core model ports are sourced from owner issue #44, not measured by this test branch.

### Policy / sync / data

Prototype policy selector: Arameair/identity@main:policies/tass.json, missing on main while PR #31 remains open. Proposed document: Arameair/TASS@live:tass.md. Grants: model.request, thought.process, action.read_tass, github.read_messages, github.reply, tass_md.sync, tass_md.conflict_pr; offline_grant_hours default 24. The Engine grant file records fetched_at and source. File permission 0600 on grant/key creation; ~/.tass and Engine data directory normally 0700. A missing configured grant denies model requests; an expired cached grant denies after expiry. GitHub writes require freshly confirmed policy through sync; GitHub token is GITHUB_TOKEN or TASS_TOKEN_FILE. The actual policy is a static GitHub file, NOT live Identity authentication or revocation.

Data: Engine ~/.thought-engine/tass/{cycles.jsonl,model-requests.jsonl,pulse.jsonl,api-key}; Core ~/.tass/{tass.md,identity-grant.json,conversation.jsonl,memory.jsonl,sync.jsonl,state.json,api-key,conflicts/}. Engine rebuilds cycles from durable ledger at restart. Core /api/status cycles are in-memory only and are not loaded from Engine on restart.

The Go cycle is: local message -> Core incoming queue -> policy check -> tp_* cycle ID -> Engine Event -> Spark -> optional Trace and Thought -> Nemotron model plan -> append and fsync prediction lock + SHA-256 -> return -> Core re-check action.read_tass -> optional read-only count -> POST outcome -> deterministic evaluation -> local reply. A malformed plan is UNREADABLE, action NONE, NOT_PREDICTED; do NOT count that as successful predicted execution. No evaluator model is configured. Trace creation requires sufficient Spark rating; a single low-rated cycle does not establish a full Trace/Thought.

Core incoming buffer is 64 messages. It serializes Answer operations; busy inference does not stop local submission, BUT saturation drops with only a log event and /api/say still returns 202. No local message receipt or correlation ID is returned at admission.

### Pulse separation and state

PRESENT: Engine ModelLink.Pulse immediately at startup and every 20 seconds queries GET /health and writes pulse.jsonl: at, model_ok, elapsed_ms, error, requests_total, failures_total, busy. This has NO pulse ID, no Start/Pause/Resume/Step Once, and no coordinated thoughts. Core GitHub sync separately retries at 20 seconds with exponential backoff. Neither constitutes the future coordinated thought pulse.

FUTURE: a 20-second thought controller with Start/Pause/Step Once/Resume, ten distinct durable pulse IDs, explicit READY/WORKING/BUSY/BLOCKED status per capability or request, slow inference observed across multiple ticks without duplicate model/action issuance, and restart continuation. NOT IMPLEMENTED on inspected branches. The verify.py thought-pulses command accepts a proposed normalized JSONL contract (kind, pulse_id, at, mode, state, active_request_id, model_started_ids, action_executed_ids, control command, process_start). It must not consume or infer this from current health records. Implementer and owner must settle stable production schema before accepting this gate.

Go does NOT currently have the Python Workshop Full Auto/Autonomy flags. A Go fixture test verifies chat can enter while a cycle is WORKING. Issue #41 Atrium chat-only fix is separate PR #43, with scripted fixture evidence; it is not a live Go Autonomy feature. The owner must not mistake a busy bool or 202 acknowledgment for a delivered answer.

### Security findings from actual source (no permissions changed)

1. P1 local owner authentication bypass: Core GET / serves its per-start browser token to any loopback HTTP client, and /api/say accepts that token. A local process may GET / and POST /api/say even without the protected program key. Origin/Host are not process identity. Treat token as CSRF mitigation, not an Identity-backed owner session.
2. P1 unauthenticated Engine legacy mutation: /api/identities, grant, events, retries, thoughts, settings, worker checks/scan all use write() without an Engine key. Host/Origin checks alone do not authorize a local process.
3. P1 Engine fail-open policy configuration: ModelLink.authorize returns a permissive engine default if -identity-policy and THOUGHT_ENGINE_IDENTITY_POLICY are absent. The bearer key still guards model/cycle endpoints, but intended Identity checks do not occur. Requiring a configured grant should be a production gate.
4. P1 silent chat drop: Core.queue buffer capacity 64, logs a problem when full but Say returns nil; /api/say answers 202 regardless and has no request ID. A 202 cannot be treated as accepted/durable work.
5. P2 Engine GET /api/model/requests/{id} has no bearer check and returns the full text and model thought. GET /api/model exposes recent request IDs. Local processes can access conversation/model evidence.
6. P2 missing state/control semantics: only core busy bool and Engine model health pulse; no distinct thought pulse, per-service state machine, pulse IDs, or Go @mention routing.
7. P2 Core old-cycle visibility after restart: Engine cycles replay from disk, Core recent cycles/status do not. Persisted back-end data does not equal owner-visible recovery.

Avoid binding unauthenticated Engine routes or local model to 0.0.0.0 / untrusted Docker bridge; do not weaken Host rules merely to get containers speaking. Keep keys, identity grants, model inputs, GitHub tokens and operational logs out of images and git. Identity PR #31 activates all seven flags including GitHub writes; owner approval remains necessary. No tests here modify Identity, Docker or any running services.

Source: core/cmd/tass-core/main.go, core/tass/core.go, core/engine/engine.go, core/identity/policy.go at the TP SHA above; Thought-engine model.go, cycles.go, server.go, main.go at pinned Engine SHA. Issues: https://github.com/Arameair/Thought-Processor/issues/41, https://github.com/Arameair/Thought-Processor/issues/42, https://github.com/Arameair/Thought-Processor/issues/44; policy: https://github.com/Arameair/identity/pull/31.
