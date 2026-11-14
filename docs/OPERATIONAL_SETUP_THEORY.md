# Working theory: operational setup versus thinking

**24 October 2026 · Architectural hypothesis for critique · NOT a released design or working integration**

> **Thought Processor controls the operational setup of thinking. Thought Engine controls the actual act of thinking.**

**Owner position:** The owner explicitly endorsed the sentence above as the working distinction and requested that the theory be developed in detail, then challenged. The definitions, proposed contracts and testing methods below are **AI-assisted interpretations for review**, not individually approved protocol decisions.

**Related:** [Identity-first architecture](https://github.com/Arameair/identity/blob/main/docs/ARCHITECTURE.md) · [Processor architecture](ARCHITECTURE.md) · [current Identity–Engine authorization plan](INTEGRATION_AUTHORIZATION_PLAN.md) · [design decisions](DECISIONS.md) · [first integration issue #8](https://github.com/Arameair/Thought-Processor/issues/8) · [independent critical review issue #7](https://github.com/Arameair/Thought-Processor/issues/7) · [initial Engine choice issue #2](https://github.com/Arameair/Thought-Processor/issues/2).

## 1. Problem this theory attempts to solve

We want independent, replaceable capabilities—Identity, observation sources, Thought Engine, Memory, AI workers and future features—to form an understandable operating system for an AI companion *without*:

- turning Thought Processor into a universal router for inference and logs;
- giving the Engine ownership over identity-wide permissions or other services' data;
- demanding the operator open several unrelated applications to understand one failure;
- confusing a pretty UI, listening port or HTTP 200 with an actually authorized, completed action;
- making installed applications stop working when the Processor UI closes;
- making an untested intelligence/coherence hypothesis the foundation of operational security.

The key hypothesis is that a **lightweight operational control layer** can reliably configure, coordinate and explain *independently running cognitive execution services* while the authority and actual work stay with the appropriate components.

This is an **operational architecture hypothesis**, distinct from the [Identity–Experience cognitive hypothesis](IDENTITY_EXPERIENCE_RESEARCH.md). Proving one does not prove the other.

## 2. Terms and ownership—be precise

| Concern | Authoritative owner | What Processor may do | What it must not do |
| --- | --- | --- | --- |
| Durable entity / feature identities | **Identity** | List, select and display verified records | Mint a competing canonical service identity |
| Trust, pairing, permission and revocation | **Identity**; receiving feature enforces | Request an authenticated owner-approved grant change; display Identity decision | Grant access itself or toggle a private Engine flag as a substitute |
| Live service instance / endpoint truth | **Running service**, with authenticated verification; Identity tracks registered services | Discover, compare registration to reality, show last-checked evidence | Infer trust from a claimed name, PID or port |
| Feature-owned data and source events | **Source feature** | Request approved subscriptions, inspect bounded authorized metadata | Default to duplicating all sensitive logs/observations |
| Thought queue, processing algorithm, Sparks/Traces/Thoughts and persistent records | **Thought Engine** | Read authorized status/evidence, request explicit allowed operations | Compute thoughts in the UI or maintain a second authoritative thought store |
| Worker invocation, heartbeat and scheduling | **Thought Engine** | Configure supported/approved policy; inspect actual worker outcome | Schedule or proxy every inference request; fabricate a result on timeout |
| Source subscription and active Engine processing configuration | **Receiving/operating feature**, subject to Identity permissions | Propose configuration, send a validated command, reconcile outcome | Treat desired UI settings as applied to a stopped/disconnected service |
| Operator's UI layout, diagnostic selection, saved test scenario | **Thought Processor** | Persist its own presentation preferences and non-sensitive test definitions | Store unprotected service credentials or own downstream state |
| Cross-service trace | **Contributing services** retain local receipts/logs | Correlate bounded permitted evidence by IDs | Claim invisible steps happened or capture all traffic by default |
| Long-term experience and retrieval | **Memory feature, if justified** (boundary still open) | Show feature availability / connection evidence | Move memory contents into Processor |

**Control does not mean absolute ownership.** Processor *requests* operational changes through supported, authenticated feature interfaces. The owning feature validates the action, persists its own effective configuration and reports what actually changed.

## 3. Two planes; not two competing brains

### Control / operational setup (Thought Processor)

Processor is the place to **understand and deliberately shape an authorized system configuration**:

1. Discover available compatible applications through Identity's approved catalog and verified runtime checks.
2. Select which entity/workspace is being managed without treating the entity as a running service.
3. Expose what sources, workers and processing capabilities each participating feature actually supports.
4. Help establish approved connections and per-entity subscriptions; route grant approvals to **Identity**.
5. Request specific authorized operational commands (pause/resume, process once, set supported policy, inspect, reconnect, run one deliberate test) and show their validated receipts.
6. Compare **requested / persisted / observed** states and surface disagreement.
7. Correlate failures, logs, trace IDs and metrics while limiting data exposure.
8. Continue to present meaningful status for independent features and entities even when Thought Engine/Memory have not been installed.

This is **not** a promise to install, start, stop or rebuild arbitrary applications. Lifecycle management requires a later explicit capability and owner consent; personal Project Controller is a separate developer tool.

### Execution / actual thinking (Thought Engine)

The Engine is responsible for **carrying out the cognitive process**:

1. Authenticate the sending service and verify Identity-governed permission before accepting an observation.
2. Deduplicate, validate, persist and queue eligible inputs according to bounded policies.
3. Select meaningful inputs, assemble appropriate context and decide whether/how to ask a model.
4. Manage the real worker API calls, inference heartbeat/scheduling, timeouts, budgets and uncertainty.
5. Interpret results, apply its own rules and preserve traceable Sparks/Traces/Thoughts.
6. Reject unapproved commands and unverified grants even if Processor is absent or compromised.
7. Maintain the effective processing configuration and queue/history across restarts.
8. Expose **truthful** read-only state, receipts, diagnostics and supported operator commands.

**Engine retains its own agency within bounds**: Processor may request a setting or one-shot operation, but cannot force unsafe ingestion, override a grant, bypass the Engine's validation or silently send a model request on Engine's behalf.

Whether Processor recommends a worker while Engine selects the runtime instance, or Engine owns all selection, is **OPEN**; what is settled is that **the Engine initiates and owns actual model requests**.

### Identity (the indispensable third role)

Identity governs **who exists, which service is trustworthy, which relationships are approved and which operation may be performed**. Identity can operate independently of Processor. Feature identity, installation identity, entity identity and transient process/instance identity are different.

A service may be **discovered** without being **paired**, and **paired** without being **authorized** for an entity's observations. Receiving services must enforce scoped grants; Processor visibility is not an enforcement boundary.

## 4. Logical communications (the arrows matter)

\`\`\`text
          OWNER / OPERATOR
                  |
                  | configure / test / inspect (authenticated operator)
                  v
           THOUGHT PROCESSOR  . . . . UI, test plan, commands, traces
             /          \
       grant request    status / command / receipt
           /                  \
          v                    v
       IDENTITY  <--- permission check --- THOUGHT ENGINE
     grant owner                            ^       |
          ^                                 |       | direct model API
          | registration / trust            |       v
          |                             authenticated    AI WORKER
          |                              observation     (replaceable)
    SOURCE FEATURE ------------------------>|
   its own lifecycle, logs and storage
\`\`\`

Legend: **Solid semantic interactions** are potential authenticated network calls, not an implemented wire protocol. Processor is not between the source and the Engine and does not intercept the Engine–worker API request. The source's trust/registration and Engine authorization verification are governed by Identity. Source-to-Engine requests can function after Processor closes if the involved services are otherwise running and currently authorized.

An approved operation is not equivalent to completed processing: **grant allowed → event accepted/queued → worker requested → worker replied → result validated → thought persisted** are different facts.

## 5. Configuration ownership: proposed rule

**Proposed default: distributed authoritative configuration + one central operational interface.**

1. Identity durably owns service registry, trusted identities, permission grants, revocations and permitted relationships.
2. Each feature durably owns its *effective* operational settings, retained data, queue and internal decisions.
3. Processor owns only its workspace preferences, view filters, explicitly saved test recipes and optionally a non-authoritative plan of requested connections.
4. Processor sends a command to the owning feature, receives a **command receipt**, then reads back the feature's observed state.
5. A mismatch between requested, acknowledged and effective state is shown as **PENDING / CONFLICT / UNVERIFIED**, not quietly reconciled into “green.”
6. Reopening Processor refreshes from authoritative services; previous display caches must carry freshness timestamps and never override real feature settings.

**Example:** Processor requests “pause ScreenBridge observations for TASS.” Identity must have authorized the actor and the subscription operation; the Engine (or subscription-owning feature) validates, stores the new state and returns a receipt. The ScreenBridge application continues to work independently. Processor reports the actual stopped *feed* rather than “ScreenBridge stopped.”

**Open design detail:** Who persists cross-feature *desired connection templates* beyond Processor? A stored plan is distinct from a valid grant and a feature's effective subscription. Do not declare one canonical store for complex connection recipes until workflows and outage recovery are tested.

## 6. Lifecycle and state model

A single “connected” badge is inadequate. At minimum, keep separate axes and evidence:

| Axis | Example states | Evidence source |
| --- | --- | --- |
| Discovery | unknown / discovered / not-found | registration record / observed endpoint |
| Authentication & trust | unverified / verified / incompatible / revoked | Identity or cryptographic peer verification |
| Entity-level permission | no grant / denied / allowed / expired / authority unavailable | authoritative Identity decision and recipient check |
| Runtime | offline / responsive / stale / degraded / ready / busy | verified, timestamped service status |
| Effective configuration | not requested / requested / accepted / applied / refused / unknown | feature-owned command receipt + read-back |
| Observation lifecycle | not sent / rejected / accepted / queued / processing / failed / persisted | Engine/source/worker receipts with trace ID |
| Evidence quality | observed / reported by service / cached / unverified / incomplete | original receipts, checking method and timestamp |

Avoid conflating status axes: a running Engine may be **unauthorized**, an authorized feature may be **offline**, and an accepted event may **never have reached a model**.

Use periodic, bounded health verification plus explicit change notifications where supported; show how old the last verification is.

## 7. Minimal shared service contract (proposal; not current API)

Every compatible feature could expose a **small versioned core**, with capability-specific extensions:

| Core action | Required answer |
| --- | --- |
| **Identify / verify** | Registered service ID, installation/instance identity, verified ownership/caller proof, protocol version |
| **Describe** | Capability identifiers/versions, supported read/command interfaces, optional original UI link and safe embedding policy |
| **Health / status** | Live/ready/degraded state, run ID, checked time, reasons, bounded resource/queue indicators when available |
| **Command** | Named operation, authenticated actor and scope, idempotency key, accepted/refused reason and operation receipt |
| **Inspect / correlate** | Permissioned, bounded status/event/metrics/log retrieval using correlation IDs with explicit omissions |

**Optional Thought Engine capability:** query observations and their processing receipts, queue, worker activity, Sparks/Traces/Thoughts and supported controls.

Proposed common metadata: \`service_id\`, \`instance_id\`, \`entity_id\` where applicable, \`request_id\`, \`trace_id\`, \`operation_id\`, \`build_sha\` if verifiable, \`observed_at\`, \`checked_at\`, \`status\`, \`error_code\` and protocol version. Not every field belongs in every message; avoid collecting sensitive raw prompt/observation content by default.

**No invented endpoint paths or claim of implemented authentication.** The transport, peer trust establishment, grant wire representation, health interval and command semantics must be specified and tested independently.

## 8. First-use operator journey and the role of the UI

**Primary user question: “What can my selected system/identity do now, and what is the next defensible action?”**

Proposed navigation structure:
- **System overview:** known identities and independent features, what is actually running, cross-component problems, one evidence-based next step; must work before TASS or an Engine exists.
- **Identity workspace:** e.g. TASS's approved capabilities, actual feature relationships, outstanding permission or operational issues.
- **Feature workspace:** e.g. Thought Engine's real interface or clearly labeled Open Original UI, current queues, worker readiness, effective controls and failures; not a duplicate reconstructed simulator.
- **Connection/test inspector:** selected source→destination relationship, trust, grant, effective configuration, explicit guided test and evidence timeline.
- **Logs / Flow / Metrics:** collapsible on demand; trace map, individual event timeline, sanitized context/copy, measured timings and errors.

A Netflix-style card catalog is a **navigation affordance**, not evidence that the overall companion works. Cards must lead to meaningful controls and honest observed results.

**Suggested primary action depends on current evidence:** connect Identity; select identity/service; review missing grant; associate authorized feature; run one guided test; inspect failed span; inspect activity. Do not default to “Run Test” forever or require selecting a companion before any feature can be managed.

## 9. First meaningful demonstration: one controlled observation

**Suggested test target:** the existing \`Arameair/Thought-engine\` experimental branch because it already has status, event APIs, queue and worker calls. This is an **initial candidate**, not a permanent choice of canonical engine. [Issue #2](https://github.com/Arameair/Thought-Processor/issues/2) remains open.

Test actors: one Identity authority; one independently running Engine; one real test sender with its **own authenticated service identity**; one selected entity (TASS as an example); optional Processor UI after backend proof.

1. Discover and authenticate participating services. A port label is not proof of trust.
2. Default **DENY**: with no matching Identity grant, Engine refuses ingestion and sends nothing to a worker.
3. Owner, using an authenticated interface, approves *only* source S → Engine E → entity X → \`thought.observation.ingest\`. Identity records decision/version.
4. Sender S submits one unique, authenticated event directly to E. E checks with Identity before queueing; response distinguishes queued from processed.
5. Engine rechecks authorization before worker dispatch. A model call and resulting persistent Thought are **separate receipts** and may not both happen in every test.
6. Revoke the exact grant. **The next event must be denied without restart; queued events must not reach the worker.**
7. Stop Identity. For the **first security experiment**, no cached allow: deny/pause new protected operations (authority unavailable, distinct from policy denial). A short-lived offline lease is a later, separately reviewed design.
8. Stop Processor. Source, Engine and Identity continue independently whenever currently permitted; then reopen Processor and reconstruct actual status without inventing intervening events.
9. Review the same trace ID across Identity decision, Engine intake, optional worker call, optional persistence and any missing evidence. Protect observation text/secrets.

**Pass criteria:** proven caller/service binding; actual allow/deny, revocation and outage behavior; no worker request for denied/queued-then-revoked observation; Processor status reflects authoritative receipts; zero mandatory Processor calls in direct production ingestion; exact timing and full commit SHAs recorded. A green tile is never sufficient.

## 10. Rival architectures and why we need to challenge this one

| Alternative | Potential advantage | Risk / uncertainty that merits a test |
| --- | --- | --- |
| **A. Central Processor proxy for all observations/worker calls** | Simplest place to instrument everything, possible centralized backpressure | Single bottleneck/failure domain, sensitive-content collection, hidden inference dependency |
| **B. Engine owns all UI, discovery and operational control** | Fewer moving parts, easier local-first MVP | Increasing coupling to Identity, other workers/sources and future independent features |
| **C. Read-only dashboard + manual separate feature setup** | Very small control-plane implementation | Poor operator journey, repetitive setup, unclear cause/effect across components |
| **D. Proposed split: Processor coordinates, Engine executes, Identity authorizes** | Independent modules, one operator workspace, direct hot path | Distributed state conflicts, more difficult cross-service authentication/correlation, possibly unjustified complexity |

**Do not call D the winner without comparison.** Propose two or more equal-scope tests: same source, same observation, same engine worker, same host and permissions; compare end-to-end processing latency, restart/outage behavior, task-completion steps, configuration drift detection, diagnosability, implementation/maintenance complexity and security boundary failures.

### Specific falsifiers / reasons to revise this theory

- Processor is required for normal ingestion or inference even after authorized connections have been set up.
- Processor and Engine disagree about the effective configuration and no reliable read-back/repair path exists.
- A grant can be bypassed by sending directly to Engine or claiming another \`source\` field.
- An authorized feed continues to run only because an old Processor window remains open.
- A failure shown as “success” lacks corresponding authoritative event/worker/persistence receipts.
- Keeping independent coordination services creates more operator confusion or unacceptable latency than a simpler Engine-first local architecture.
- A stopped/restarted Engine cannot restore its own operational state without Processor's private cache or an unapproved actor.
- Distributed trace/log collection leaks observation content or secrets by default.

## 11. Questions to put to Grok, designers and engineers

**Please disagree where warranted. Prefer a concrete counterexample and a discriminating test over endorsement.**

1. Does “Processor controls operational setup” accidentally give it responsibility for too much orchestration? Should it only render a declarative plan, with feature-owned reconcilers?
2. Who owns *desired* cross-feature connections, and what happens to a requested-but-not-applied configuration across restart? Is a separate registry needed, or is that needless infrastructure?
3. What is the smallest common service contract that supports a non-cognitive feature, not only Thought Engine? Can a service supply useful capability data without exposing private operations?
4. Which operations are *configuration* versus *thinking*? Who chooses worker capability, fallback and processing policy when the Engine can function without Processor?
5. How do we verify service identity, owner approval and the original observation sender on a local machine without mistaking localhost for authentication?
6. Does early adoption of distributed traces and multi-component receipts add overhead that exceeds the value of diagnosis? What is the least instrumentation that works?
7. Could a normal Engine UI plus an Identity permission interface produce the same experience with far less infrastructure? Propose a small user test.
8. Which UI should open first—system overview, selected companion/identity, or failure/task inspector—when a user has zero optional features or five independent ones?
9. Can Thought Engine truly run independently if it performs an online Identity check on every observation? Define the difference between **Processor independence** and **Identity availability**.
10. What specific empirical result would convince us to merge Processor into Engine, split Processor into narrower services, or abandon the proposed distinction?

For critical review, use **claim → evidence/assumption → consequence → alternative → smallest falsifying test**. Record whether a statement is owner-selected, reviewer opinion, proposed protocol or measured result.

## 12. Decisions recorded versus open

### Owner-endorsed high-level direction
- **“Thought Processor controls the operational setup of thinking. Thought Engine controls the actual act of thinking.”**
- Identity owns participant trust and permission decisions; Engine enforces; Processor helps connect, deliberately test and explain.
- Features operate independently and keep their own data; Processor is not a mandatory data-path relay.
- First prototype must show real status/results and owner-controlled branch promotions, not mock success.

### Explicitly OPEN—not silently chosen here
- Precise scope of Processor's connection plans and durable storage of desired state.
- What Engine commands are supported by the first shared contract.
- Core API/auth transport and bootstrap trust mechanism across Windows/WSL/VM.
- Source-level grant semantics, cached/leased authorization later, effective revocation timing.
- Which existing Thought prototype is the canonical future Engine; using model-assisted \`Thought-engine\` as test target does not decide migration.
- Subscription authority when both source and Engine store effective preferences.
- Worker-selection policy and fallback, queue overflow/drop semantics, retention and detailed diagnostics.
- Whether Processor should ship integrated with Engine for end users versus remain a separate optional management feature.
- Measurement thresholds at which distributed control costs more than it helps.

## 13. Proposed next sequence—do not implement from this note alone

**Review → settle a minimal contract → backend security proof → thin Processor test surface → compare against a simpler baseline → revisit the broader schematic.**

Begin with [Identity grant issue #15](https://github.com/Arameair/identity/issues/15) and [Engine enforcement issue #1](https://github.com/Arameair/Thought-engine/issues/1). The Processor-driven acceptance plan is [Issue #8](https://github.com/Arameair/Thought-Processor/issues/8). Continue open challenges in [Issue #7](https://github.com/Arameair/Thought-Processor/issues/7).

No code is created by documenting this theory. No experiment is proven and no branch promotion is authorized.

---

**Human–AI collaboration:** Owner-originated architectural distinction, expanded into this reviewable theory with ChatGPT assistance. Prior user-supplied Grok and other independent feedback is treated as **opinion for critical evaluation**, not acceptance or a verified integration.