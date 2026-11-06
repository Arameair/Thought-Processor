# Cross-repository conflict remediation — versioned engineering handoff

**Review date:** 2026-11-06 (America/Chicago)  
**Status:** SOURCE-AUDITED / PROPOSED REPAIRS / NOT OWNER ACCEPTED  
**Baseline:** Thought-Processor PR #88 Core head **8e040b33e7a492f042a1f4449b8c7b013c227965**, pinned Engine **d69d3917d1601d806d4b2ba2254693e1adb92b6b**, pinned Identity **7da7f7c5bbecc42d3a7e9316a2037243eda10b88**.  
**Evidence:** GitHub source/PR/issue review only. **No real owner's PC test, deployment, overnight Codex-result verification, GitHub credential change, autonomous activation or schema migration.** Source statements below refer to this exact candidate unless stated otherwise.

**Mission:** Preserve one reusable Go-based agentic harness while eliminating structural ambiguity **before** it becomes a hard-to-diagnose cross-identity, action-authorization, evidence-correlation or data-loss bug. This is a resolution register. Each conflict has a stable identifier, owner, linked issue, proposed repair and a testable exit. Do **not** treat every historical prototype as a live defect.

## 0. Rules for contributors

1. **One live owner installation.** Only work in independent experimental branches and isolated test roots. Do not merge, deploy, run an extra competing TASS, change the model, enable Autonomous/Pulse/GitHub relay, change Identity grants, or edit private TASS/KYSSA records without a separate owner decision.
2. **Don't fix security conflicts by weakening a guard.** Keep restrictive mailboxes, action allowlists, server-side access controls, locked predictions and Identity point-of-use checks until replacement security mechanisms are independently proven.
3. **One authoritative record per concern:** Identity for canonical identity/grants; Thought Engine for persisted thoughts, cycles, predictions, outcomes and event journal in the Go candidate; Core for its own application configuration/approved lesson and skill execution receipts; Development as controlled experiment surface; Cognition as read-only evidence projection. Proposed ownership changes require a documented owner decision.
4. **Evidence vocabulary:** DESIGNED, VISIBLE, WIRED, EXECUTABLE, OBSERVED, VERIFIED; and PASS, FAIL, PARTIAL, NOT RUN, NOT RECORDED, UNSCORABLE. Never turn a fixture PASS, UI graphic, source code inspection, timer tick or model claim into a live-operation PASS.
5. **Separate design decisions from defects:** Confirmed source fact versus unverified failure possibility versus future design. Keep historical data and source revisions; never reassign ownership by matching display names.
6. **Every change requires:** exact base SHA, changed public contracts, tests (positive **and negative**), migration and rollback notes, privacy/secret review, sample redacted receipt, CI link, actual owner-machine outcome where requested, and an explicit remaining-blockers section.
7. **No automatic mobile/network exposure.** A future mobile companion is an authenticated client of a deliberately designed API, not permission to bind today's localhost Core and Identity endpoints to the LAN or internet.

## 1. Conflict index and canonical work-item map

| ID | Priority | Type | Record owner | Closest existing tracker | Closure evidence |
| --- | --- | --- | --- | --- | --- |
| C1 | P0 decision | Competing historical thought authorities | Thought Processor + owner | [TP #2](https://github.com/Arameair/Thought-Processor/issues/2) | One declared canonical Go Engine and no second writer; migration policy |
| C2 | P0 security/correctness | Identity ID versus Engine display-name resolution | Thought Engine + Identity + Core | [TP #82](https://github.com/Arameair/Thought-Processor/issues/82), [Identity #30](https://github.com/Arameair/identity/issues/30) | ID-bound routing, hostile same-name tests, safe legacy mapping |
| C3 | P0 integration | Engine Pulse.Tick versus Core SyncOnce; no proved closed thought loop | Thought Engine + Core | [TP #70](https://github.com/Arameair/Thought-Processor/issues/70), [#89](https://github.com/Arameair/Thought-Processor/issues/89) | One correlated end-to-end task; distinct clocks observable |
| C4 | P1 ownership | Developer / Development / integrated Core /dev overlap | Development + Developer + Core | [Development #1](https://github.com/Arameair/Development/issues/1), [Developer #7](https://github.com/Arameair/developer/issues/7) | Roles, versioned interface, no dual authoritative workspace |
| C5 | P1 behavior | Advisory Brain Builder versus actual selection/execution | Core + Engine + Development | [TP #89](https://github.com/Arameair/Thought-Processor/issues/89) | Controlled graph A/B changes actual authorized execution |
| C6 | P1 authority | Legacy teaching/GitHub commands versus newer restricted mailbox | Core + Identity + TASS | [TP #72](https://github.com/Arameair/Thought-Processor/issues/72), [#76](https://github.com/Arameair/Thought-Processor/issues/76) | No external comment creates grant, approval or execution |
| C7 | P1 evidence | Cognition thought cards versus canonical Thought/journal truth | Engine + Core/Cognition | [TP #82](https://github.com/Arameair/Thought-Processor/issues/82) | Read-only, ID-correlated playback and honest missing stages |
| C8 | P1 provenance/privacy | TASS mind source and GitHub mirror documentation conflict | TASS + Core + Identity | [TASS #21](https://github.com/Arameair/TASS/issues/21) | Written source-of-truth policy; verified offline/conflict behavior |
| C9 | P1/P2 portability | TASS-only strings/schemas and install assumptions | Core + Identity + Developer | [TP #90](https://github.com/Arameair/Thought-Processor/issues/90) | Two independent synthetic installations, same code |
| C10 | P1 release truth | Old READMEs/prototypes versus current pinned one-app candidate | Release owner + all maintainers | [TP #79](https://github.com/Arameair/Thought-Processor/issues/79), [PR #85](https://github.com/Arameair/Thought-Processor/pull/85) | Exact release manifest, clear historical labels, reproducible tests |

**Related baseline:** [TP PR #88](https://github.com/Arameair/Thought-Processor/pull/88) integrates many pieces but explicitly reports **not owner-deployed and not run with real Nemotron**. [TP #89](https://github.com/Arameair/Thought-Processor/issues/89) contains the narrow first supervised execution test. [Identity #15](https://github.com/Arameair/identity/issues/15) requires authenticated sender and fail-closed observation grants. [Core issue #47](https://github.com/Arameair/Thought-Processor/issues/47) documents the localhost browser-token threat and must not disappear in any new network design.

## 2. C1 — One canonical Thought Engine and one operational evidence authority

**Source evidence:** [TP #2](https://github.com/Arameair/Thought-Processor/issues/2) explicitly compares Arameair/Thought deterministic prototype and Arameair/Thought-engine model-assisted prototype. The newer Go Core Compose pins Thought-engine while historical Thought Controller and Workshop also display or construct similarly named Thought/Trace records. The existence of multiple prototypes is **not** proof of duplicate active writers on the current installation.

**Failure if ignored:** The same event can have two unrelated thought IDs, inconsistent decision histories or retention, different grant states, or a Cognition shelf that accidentally merges unrelated records by topic. Restore/migration may resurrect an older store as though it were authoritative.

**Proposed decision:** Declare the **exact version-pinned Thought-engine used by the integrated Go Core** the canonical runtime owner for durable observations, Spark/Trace/Thought objects, cycles, predictions, outcomes and ordered journals. Existing Thought and Thought Controller are reference/read-only unless explicitly migrated. Core may keep its application receipts and conversation transcript but must reference Engine record IDs instead of creating competing authoritative thought objects.

**Implementation instructions:**
- Inventory running services and every storage writer to Thought/Trace/Cycle data, including Workshop, Atrium, legacy notebooks and Engine. Produce a table of process, disk root, schema version, read/write authority and whether installed or merely experimental.
- Pin the Engine API and authoritative record types in a versioned contract. Give each event one canonical installation ID, entity ID, source event ID, cycle ID, and optional persistent thought ID. Never infer identity by title or semantic similarity.
- Make alternate runtimes explicit **isolated experiments**; their own data must not appear as canonical TASS evidence. Do not delete historical stores. For selected migration, specify source revision, deterministic mapping, duplicate rules, backups, hash verification and revert.
- Publish an owner-reviewed decision to #2: which Engine is authoritative, where its source lives, what remains experimental, and how Cognition locates it.

**Tests/closure:** A single synthetic observation yields exactly one canonical Engine record; re-poll, restart and replay do not generate another thought; a legacy record appears only under a clearly labeled historical/unlinked view; a mismatched installation ID cannot be joined. Record exact file hashes before/after any accepted migration. **BLOCK** migration if record ownership is ambiguous.

## 3. C2 — Canonical Identity-to-Engine routing must not depend on display name

**Source evidence:** [Cognition PR #87 owner runbook](https://github.com/Arameair/Thought-Processor/blob/experiment/claude-cognition-repair-junction-20261105/core/docs/OWNER_ACCEPTANCE_COGNITION_JUNCTION_20261106.md) documents an explicit owner-reviewed Identity→Engine link in the Cognition viewer, **but Engine still selects/creates internal entities by display name**. [Engine engine.go around entity resolution](https://github.com/Arameair/Thought-engine/blob/experiment/cognition-journal-projection-20261105/engine.go) compares DisplayName. The real cross-identity *exploitability* has **not** been established; the routing ambiguity is real.

**Failure if ignored:** Two people can choose the same companion name; renaming or recreating a companion may bind a new request to an old Engine record. Cognition's display filter does **not** correct internal action/queue routing. A receipt might be attributed to the wrong user.

**Required safe design:** Each authenticated request carries the **canonical durable Identity entity_id** bound to the authenticated installation/principal. A display name is presentation-only. Engine stores that verified entity ID on records, queues and receipts; Engine-local UUID, if retained, is a separate mapping key. The mapping is created/approved through Identity-controlled enrollment, not inferred by string equality.

**Implementation instructions:**
1. Inventory current Engine entityFor behavior and every call site for StartCycle, observation ingest, pulse job, journal GET and Cognition entity list. Identify who supplies each source/entity string and how the server authenticates that caller.
2. Specify a versioned envelope: installation_id, canonical_entity_id, authenticated_source_service_id, action, request/cycle_id, authorization decision ID, optional engine-local entity ID. Bind authorization **at ingress and again at dispatch/use**; fail closed on mismatch, revoked grant, LOCKED or unavailable Identity, retaining an honest status.
3. Do **not** automatically rewrite existing Engine entity IDs. Record a one-time audited, owner-approved mapping for historical data; ambiguous/unscoped records remain UNLINKED. Preserve old records and support a read-only compatibility view.
4. Update Cognition to consume the verified mapping instead of manufacturing one; preserve the existing explicit link as a historical reconciliation step until migration acceptance. A rename changes a label, never ownership.
5. Use per-entity storage/queue partitioning and pass canonical IDs to the Engine logger/receipt builder. Never expose another entity's journal merely because a route was guessed.

**Negative acceptance matrix:** same name/different canonical IDs; renamed entity; deleted/recreated entity reusing name; forged body/header entity ID; caller authorized for A attempting B; direct Engine route bypassing Core; revoked while queued; identity unavailable; restart after mapping; Cognition export from A referencing B. Expect **DENIED or UNLINKED**, zero cross-record disclosure and zero unauthorized model/tool execution. Actual same-name collision requires real Engine test, not only a mocked Core filter.

**Stop condition:** Any unintended cross-entity read or action is a **P0 blocker**, independent of other green UI tests. Existing user data migration requires explicit owner approval and validated backups.

## 4. C3 — Two 20-second clocks are not one completed thought loop

**Source evidence:** Core SyncOnce independently handles GitHub/lesson/junction/mailbox polling, while Engine Pulse.Tick owns queued/running jobs. [Cross-loop design PR #85](https://github.com/Arameair/Thought-Processor/pull/85) explicitly warns they are separate schedules. [TASS #25](https://github.com/Arameair/TASS/issues/25) describes a **future** dynamic five-card logical batch. Engine Pulse ticks passing in CI do not prove the complete future thought design.

**Failure if ignored:** The UI may equate a sync tick, model heartbeat, partial junction or NO_ACTION with a real prediction/evaluation loop. Slow network calls may block useful local processing. Restart may duplicate a pending request, while two independent ticks could race to consume the same receipt.

**Target contract:** Engine is the owner of **thought-cycle progression and immutable predictions**. Core SyncOnce remains a separately labeled **transport/synchronization loop**, not a second agentic planner. A junction/lesson/external reply becomes an authenticated, idempotent input to a specific later Engine cycle. Never create a third scheduler to paper over the mismatch.

**Implementation instructions:**
- Instrument both clocks with separate names, tick IDs, duration, queue length, active work IDs and pause state; include a correlation edge only when a real item actually crossed from Core Sync to Engine.
- For one pilot event, record an ordered sequence: observed input → context source/revision → Engine cycle ID → decision/proposed operation → **prediction locked before action** → Identity grant/decision → actual executor receipt or NO_ACTION/DENIED → observed outcome → PASS/FAIL/UNSCORABLE evaluation. Keep actual timestamps and append-only source journals.
- Define the meaning of missed tick, busy model, offline GitHub, late response and reboot. Core must not await a slow external call inside every Engine Tick; persisted pending jobs are inspected later without blind retries.
- Bound queued work, model concurrency and memory on the owner's existing hardware; a five-card **logical** batch is not a promise of five GPU-parallel inferences.
- Show **Engine PULSE**, **Core SYNC** and **actual thought/work cycles** separately in Cognition. Mark incomplete stages PENDING or NOT RECORDED, not completed.

**Tests:** 20-second idle/no work; one task spanning two or more ticks; fake-GitHub timeout; lost acknowledgment; duplicate junction request; Core restart while Engine stays up; Engine restart while Core stays up; pause/stop, revoked permission, 60-second+ model delay. Assert no unauthorized action, no unbounded queue, no duplicate execution, truthful delayed outcomes and actual correlation. Record true timing/resources; no fabricated continuous thinking.

**Owner decision deferred:** The five-card batch reader's relationship to its five cards, editable state storage and later autonomous learning remain designs, not prerequisites for this one-cycle integration gate.

## 5. C4 — Give Development, Developer and integrated /dev noncompeting responsibilities

**Source evidence:** [Development #1](https://github.com/Arameair/Development/issues/1) is a consolidation hub; [Development Brain Builder PR #2](https://github.com/Arameair/Development/pull/2) is a standalone nonexecuting visual Go prototype. [Developer #7](https://github.com/Arameair/developer/issues/7) proposes a future TASS-taught workstation; [Developer PR #5](https://github.com/Arameair/developer/pull/5) has independent supervised Team Member calls. [Core PR #75](https://github.com/Arameair/Thought-Processor/pull/75) already implements the current Go Core /dev UI. These overlap in product language but are **not proven simultaneous live writers**.

**Risk:** Same graph or lesson exists in multiple data roots, and a UI presents a proposed update as if another service performed it. Developer's local Git authority might later bypass Identity when invoked by an agent. A wholesale merge destroys useful isolated test history.

**Proposed role split (requires owner acceptance):**
- **Arameair/Development:** design/specification, test fixtures, prototype comparisons and proposal history; **not** the production authority for companion data, Identity grants or executor results.
- **Arameair/developer:** optional future separately installed, sandboxed **workspace capability** with its own registered service ID. No permission from mere process discovery. Can inspect/work in a user-selected workspace only via explicit resource-scoped grants and supervised accept/apply.
- **Thought-Processor /dev:** current TASS-integrated owner test/review interface; one current owner of saved integrated brain revision, learning status and skill receipts. Its model/context data remain per entity.
- **Cognition:** read-only cross-links to experiments and actual Engine receipts, never another independent graph writer.

**Instructions:** Inventory schema/data roots and state in all three; choose which features to port rather than copying entire apps. Define a **versioned capability API** for future Developer invocation with entity_id, caller service ID, workspace root allowlist, proposal ID, idempotency key, expected file SHA, approval ID, exact diff and observed result. The receiving Developer must authenticate and enforce authorization; model-proposed paths are not privileged. Avoid introducing a second global workspace graph and do not reuse another app's saved token.

**Tests:** Same proposed workspace read from /dev and standalone Developer returns identical source SHA or explicit version mismatch; two independent entities cannot access each other's workspace; denied, replayed or stale apply produces zero filesystem mutations; crash/restart doesn't replay a write; no Developer result appears as verified until an actual receipt exists. OWNER-TESTED status stays separate for the standalone prototype.

## 6. C5 — A graph that renders is not a graph that governs execution

**Source evidence:** [Brain Builder README](https://github.com/Arameair/Development/blob/experiment/go-brain-builder-prototype-20261105/brain-builder/README.md) states **structural review only** and no Engine/Identity/executor integration. Core's brain revisions are editable and structurally validated, but advisory for actual tool routing/execution; [Cognition runbook](https://github.com/Arameair/Thought-Processor/blob/experiment/claude-cognition-repair-junction-20261105/core/docs/OWNER_ACCEPTANCE_COGNITION_JUNCTION_20261106.md) says proposed alternative cells are unactivated and **EFFECT NOT MEASURED**. [TP #89](https://github.com/Arameair/Thought-Processor/issues/89) already defines first authorized skill and controlled A/B milestone.

**Risk:** A connection is drawn, the prompt says a different action was selected, and the UI reports improvement despite the actual executor doing the same operation or none at all.

**Implementation sequence:**
1. First owner-test the **existing, fixed read-only** supervised @lesson-workspace.list-lessons skill with explicit Identity permission and a persisted actual output/receipt. No arbitrary notebook or shell execution.
2. Freeze the existing task, model/worker version, context source revisions, identity, grants, evaluator criteria and graph revision. Save the independent expected output/prediction **before** action.
3. Specify a **small typed, deterministic router** that maps a validated, approved graph revision to one of a registered allowlisted set of operations; reject unknown IDs and stale refs. Graph text/cells are data, never executable code or permission.
4. Add a second narrowly read-only operation such as inspect-lesson **only through separate review and Identity grant**. Graph A should actually select list-lessons; graph B should actually select inspect-lesson for a defined input while other conditions remain fixed.
5. Trace graph_revision → decision policy/reason summary → selected tool ID → Identity decision → actual tool invocation/receipt → observed output/hash → scored comparison in Engine and Cognition.
6. Roll back graph B to A without modifying historical cycle records or grants. Any mismatch becomes FAIL or UNSCORABLE, never pass by model self-report.

**Tests:** invalid edge or cycle is server-rejected **before** save; stale SHA; graph with unregistered skill; graph granted to wrong entity; duplicate execution retry; Identity revoked between selection and dispatch; A/B comparisons with negative control where both graphs choose same operation. Required proof: two distinct **real** execution receipts and distinguishable measured outcomes on the intended scenario. If only selection text differs, **NOT IMPLEMENTED**.

## 7. C6 — Teaching, owner approval, GitHub communication and ability grants must stay separate

**Source evidence:** Earlier [Core PR #69](https://github.com/Arameair/Thought-Processor/pull/69) includes broader GitHub-triggered study concepts; [Core #81](https://github.com/Arameair/Thought-Processor/pull/81) and [PR #88](https://github.com/Arameair/Thought-Processor/pull/88) deliberately restrict the pilot mailbox to **conversation only** at Arameair/TASS#24. TASS lesson PR #23 proposes acceptance of lesson **source**, while Core's Learning requires explicit owner approval to mark learned. Identity skill grants are separate; [Identity PR #40](https://github.com/Arameair/identity/pull/40) explicitly separates junction grant from skill-run grant.

**Risk:** A GitHub commenter or teacher could accidentally confer learned status, execution authority or a persistent mind update just by posting persuasive instructions. An AI may claim it learned after only reading a document or passing a fixture.

**Unifying contract:**
- **Source acquisition:** candidate lesson submitted with publisher, repo/path, exact blob SHA, timestamp, trust/sync status.
- **Study:** actual supervised Engine job with unique durable attempt/job ID and model used. Preserve source answers/gradeable checks; never rewrite past attempts.
- **Grade and learn:** independent scoring of real answers; absent/invalid responses are NOT GRADED. Owner approves **specific lesson revision** before LEARNED state. A fresh-task retest is a new distinct evidence stage.
- **Skill operation:** compiled/allowlisted operation registered with Identity; separate skill.discover and scoped skill.run grants, approval/revocation, point-of-use check and actual executor receipt.
- **Mailbox:** externally authored text is conversation or **proposal only**. GitHub comment, model reply, graph instruction and Markdown do not grant, approve, run, merge or deploy. The pilot's hardcoded repo/issue restrictions remain until a reviewed, per-identity alternative is available.

**Instructions:** Publish a single versioned lesson lifecycle schema and explicit transition function, reused by GitHub and Development teaching entrances, and a single verified state/revision. Ensure both interfaces show the same authoritative lesson/skill status. Keep source authenticity (GitHub account claims) separate from Identity owner approval. Sanitize private lesson prompts/answers from mailbox logs. Disable wider legacy routes in production app mode unless separately authorized.

**Tests:** foreign author and wrong repo/issue; fake ChatGPT prefix; comment that says APPROVE/EXECUTE; duplicate incoming comment; crash after POST with unknown delivery; revised lesson SHA; malformed answer; exhausted attempts; denied approval; fresh-task retest without answer text; learned knowledge with no skill.run grant. Assert **zero unintended grant, skill execution or learned-status promotion**.

## 8. C7 — Cognition is an evidence projection, not a second thinking system

**Source evidence:** [Cognition #82](https://github.com/Arameair/Thought-Processor/issues/82) distinguishes persistent Engine Thought and Trace records from separate Cycle records and append-only cycles.jsonl. Visual-only [PR #84](https://github.com/Arameair/Thought-Processor/pull/84) could synthesize cards from recent cycles; Go-first [PR #86](https://github.com/Arameair/Thought-Processor/pull/86) and [repair #87](https://github.com/Arameair/Thought-Processor/pull/87) provide grounded replay and explicit Identity link. The owner runbook says result is **executor-reported, not independently verified** and original cycle authorization decision may be **NOT RECORDED**.

**Risk:** A pleasant UI makes nonexistent thought history, a model's claim, or a timed playback animation appear to be an actual verified execution. A wrong entity's records could leak through a broad endpoint or export.

**Implementation instructions:**
- Engine owns immutable journal/event ordering, offsets, persistent thought IDs, model/job requests and outcomes; Core Go builds **bounded, allowlisted, entity-authorized read models** with source IDs, correlation and explicit missing state. Keep a field recording whether the result is **app reported**, externally verified, or merely inferred.
- Do not group cycles into canonical Thoughts by title/topic. Use a recorded thought_id link or display as **unlinked recent cycle**. Preserve PENDING, NOT RECORDED, UNAVAILABLE and DENIED.
- Playback, Step, Copy/Download and historical filters are read-only actions against an immutable snapshot. No model call, approval, graph mutation or pulse dispatch when viewing.
- Attach a prediction only if the Engine proved it locked **before** action. Show actual work receipts separately from model-produced narrative/summary. Generate sanitized exports in Go using explicit field allowlists; exclude secrets, unapproved private context and raw service keys.

**Tests:** read twice leaves journal/content hashes unchanged; read while cycle is pending shows PENDING; missing thought_id remains unlinked; same-name foreign entity hidden and zero Engine contact while Identity LOCKED; malicious journal string cannot inject HTML or execute a request; export never includes service keys, bearer tokens or private documents; historical playback uses original event timestamps, not browser-created time. Compare UI IDs to raw local journal **only on authorized owner machine**.

## 9. C8 — TASS's descriptive mind, GitHub mirror and local private journals need one written truth

**Source evidence:** [TASS live tass.md](https://github.com/Arameair/TASS/blob/live/tass.md) contains a comment saying the entity's document 'lives only on this computer', yet that text is visibly stored in a private GitHub repository. Historical TASS Alpha and Developmnt sources were different; [TASS #21](https://github.com/Arameair/TASS/issues/21) inventories the split. The Core explicitly uses local tass.md plus a reviewed GitHub mirror with conflict handling. The comment is a **documentation contradiction**, not proof that private memory journals were uploaded.

**Risk:** The owner believes all content is local when a deliberate sync is enabled; two repos both claim authority; a conflict overwrites the wrong mind source or a new install silently points at the original owner's GitHub.

**Proposed source policy:** Per installation, Identity maps canonical entity_id to (a) protected local runtime document, (b) **optional owner-configured and consented** versioned GitHub mirror for descriptive mind/lessons, and (c) private local memories/conversations/secrets/journals that **never auto-sync**. The GitHub repository does not grant rights. Define source precedence/conflict behavior, including when offline and how owner-reviewed conflict PRs are accepted.

**Instructions:**
- Inventory the exact files and both inbound/outbound directions of GitHub sync (including history, backup and outbox). Mark each file class LOCAL_ONLY / OPT_IN_SYNC / PUBLIC_SHAREABLE / SECRET; do not assume 'private repo' implies safe for raw personal content.
- Make first-run without GitHub the default and ensure a new installation cannot inherit Arameair/TASS or its credentials unless owner explicitly selects it.
- Rewrite ambiguous TASS mind comments and README claims through a **separately approved mind/document PR** after confirming owner meaning; do not change live identity content as an incidental engineering fix.
- Prove conflict resolution preserves local journal, exact remote SHA, local edits and owner review. Document how to disconnect GitHub while continuing to think locally; changes to long-lived identity guidance must remain owner reviewable.

**Tests:** no repo/token → local-only working; wrong repo/revoked grant → deny sync, preserve local content; diverged local/remote section → conflict PR or pending-review state, not silent loss; round-trip checksum; rebuild and rollback keep identity and approved learned lesson state; GitHub issue/raw private chat not accidentally copied to external log. Use synthetic documents, not actual sensitive companion data.

## 10. C9 — Remove TASS-only application assumptions without breaking the safe pilot

**Source evidence:** [TP portability audit #90](https://github.com/Arameair/Thought-Processor/issues/90) identifies legacy owner-specific conversation prompt in Core core.go, explicit Arameair/TASS relay source/receipt, creator literal 'owner'/'TASS' in brain.go, notebook metadata.tass and internal tass.md. These are genuine source assumptions, but current app mode already uses c.entityName and configurable APP_HOME, APP_PROJECT, host ports, MODEL_MODE, GitHub repo and docs. Internal Go module paths and a safe default file name do **not** by themselves make other computers impossible.

**Failure if ignored:** Generic installations address the wrong owner, incorrectly label provenance, cannot register their own repo/skill, or import another entity's graph history. A careless 'fix' could widen GitHub writes to arbitrary repositories.

**Instructions:**
- Replace current **generic product** assumptions with authenticated installation/entity/principal identifiers from Identity. Keep legacy reading/migration of existing TASS graph and notebook metadata; introduce versioned neutral creator/source fields before any write. Do not auto-upgrade old records when ownership is ambiguous.
- For GitHub evidence, build attribution from **approved configured repository and verified incoming IDs**; leave the **Arameair/TASS#24 pilot allowlist in place** until a separate reviewed scoped mailbox contract with tests exists. Never construct privileged source identifiers from untrusted free-form model/chat text.
- Audit packaged defaults and dynamically resolved paths instead of hardcoding $HOME, WSL paths, personal GitHub names or live ports. Every install may choose its own root/project/host-port pair; internal container ports are not automatically a portability defect.
- Retain one entity per independent installation as the **first product target**; don't force same-process multi-tenancy now. Build two synthetic installations with unique IDs, storage, grants and logs, and prove one cannot read the other.
- Model serving should be an independently selected/versioned worker contract: local llama.cpp/Nemotron first, model URL and GPU/CPU already configurable; later adapters should not change identity, authorization or execution record schema.
- For eventual mobile access, design a secure authenticated companion API/session, user-approved device pairing and revoke/transport security first. **No direct exposure of localhost browser token APIs** (see [TP #47](https://github.com/Arameair/Thought-Processor/issues/47)).

**Tests:** synthetic Entity Alpha vs Entity Beta; no default TASS/Dustin/Arameair in **generic output**, per-identity graph source and receipt; configurable repo missing/denied; isolated APP_HOME/project/ports; startup port conflict fails safely; local-only CPU model health honest; backup and restore one without touching other. Do not bring KYSSA's real account or private data into fixtures without separate consent.

## 11. C10 — Make the release/candidate record the source of engineering truth

**Source evidence:** Engine candidate [README](https://github.com/Arameair/Thought-engine/blob/experiment/cognition-journal-projection-20261105/README.md) still narrates original Chat-Jimmy/Thought Controller workflow; current [Core Compose](https://github.com/Arameair/Thought-Processor/blob/experiment/claude-foundation-integration-20261105/core/compose.yaml) launches the Engine with -worker nemotron and a llama.cpp endpoint. Other default branches and old drafts describe standalone components. This is **documentation drift**, not proof today's Core is using the wrong worker.

**Risk:** A contributor bases a fix on an obsolete architecture, ports, API or model, or interprets green fixture CI for a different pinned trio as proof a release is safe.

**Instructions:**
- Issue one dated **candidate manifest** with exact full Core, Engine, Identity SHAs; source branches; image digests; state/data schema versions; configuration defaults versus installation overrides; current owner-installed trio (only when actually read on owner host); model worker and connection; live/fixture test results; open blockers; known migration/recovery commands.
- At repository entry points, label old experimental READMEs **HISTORICAL / NOT CURRENT GO ONE-APP INTEGRATION** with a pointer to the manifest; do not edit/delete preserved experiment history. README status must say what was tested and when.
- Reconcile cross-repo acceptance issues: Identity grants are not the same as a TASS mind file; Thought Engine journal is not a Core front-end animation; Development structural graph validation is not an executable graph; external GitHub sync is not local-only data.
- Require the manifest as a release gate. The updater must refuse incompatible Engine/Core/Identity versions and must have verified backup + restore, interruption handling and rollback [TP #79](https://github.com/Arameair/Thought-Processor/issues/79).
- Keep a **conflict status ledger** in #89 and link work-item PRs. Each entry states OPEN, IN DESIGN, IN PR, FIXTURE VERIFIED, OWNER VERIFIED or BLOCKED with full SHA and evidence. Don't silently close with a passing test on unrelated code.

**Tests:** clean workspace checkout; run documented build/vet/race/Compose syntax and regression commands; startup with exact pinned trio and fixture (clearly labeled fixture); recovery fault injection; authorized owner-machine tests only after approval. An old branch's README must not be interpreted as current live runtime. Documentation-only updates never count as behavior acceptance.

## 12. Mandatory cross-cutting regression matrix

Every proposed fix must state **what it changed and what it must not change**:

| Scenario | Required result |
| --- | --- |
| Identity LOCKED or unreachable for protected operation | Fail closed; no new model/tool execution; explicit unavailable/denied state |
| Wrong entity ID, reused display name, forged sender | No cross-entity data, receipts, grant access or context; mapping remains explicit |
| Untrusted GitHub Markdown/comment requests action | Proposal/conversation at most; no grant, autonomous command or execution |
| Duplicate cycle/job/comment/approval after restart | Idempotent or safely ambiguous; never silently run or post twice |
| Engine read-only journal playback | No new model call, permission grant, state mutation or changed journal hash |
| Network outage while local thinking | Local work remains responsive; external sync pending with backoff; no fake success |
| Two synthetic installs | Different keys/IDs/roots, no shared private files or logs, one updater cannot damage other |
| Graph changes while Identity permission revoked | No tool execution even if graph/proposal suggests it |
| Update fails at backup/build/restart/restore step | Exact version/data receipts; verified restoration or honest BLOCKED |
| Model formats unexpected reply / needs longer than 20 s | Clear failure/pending, no unbounded concurrency, no forced fabricated prediction |

### Full acceptance evidence packet

Each linked issue/PR must include:
- **Baseline:** repo, branch, full source SHA, installed/candidate distinction; exact model worker/version and environment.
- **What changed:** file paths, schema/API migration, compatibility, config, threat model and permissions.
- **Positive and adversarial test cases:** commands and GitHub Actions URLs plus exact test results, fixtures vs real model clearly separated.
- **Correlations:** entity ID (synthetic/redacted), authenticated service, request/cycle/job, grant decision, source revision, prediction digest/lock, executor receipt, outcome/evaluation and timestamps.
- **Durability:** restart, repeated delivery, backup/rollback, crash/race, redaction scan and counterexample tests.
- **Verdict:** PASS / FAIL / PARTIAL / NOT RUN, unresolved items and next required owner action. Do not claim DONE based only on a PR description.

## 13. Recommended order of work and hard stop gates

**P0A — Read-only inventory and acceptance first.** Collect actual installed Core/Engine/Identity full SHAs and current operational state. Review Codex/Claude overnight audits independently. Confirm PR #88 is a **candidate**, not presumed installed. Keep owner-private data unchanged.

**P0B — Resolve C1 and C2 design authority.** Record one Engine of record and authenticated canonical entity routing. Implement/test Engine-side source identity mapping and same-name negative cases on isolated fixtures before supporting a second real entity. No irreversible history rewrite.

**P0C — Prove C3 first real loop under [#89](https://github.com/Arameair/Thought-Processor/issues/89).** Only after safe recovery [#79](https://github.com/Arameair/Thought-Processor/issues/79), run one owner-approved read-only @skill execution with actual Identity decision, Engine prediction/outcome and Cognition read-only trace. Confirm Core SYNC and Engine PULSE labels. Stop at first mismatch.

**P1 — Resolve C4/C5/C6/C7 together at interface boundaries, not as a new app.** Pick /dev as proposed active TASS UI and Developer as optional future skill, then prove one graph A/B effect, one lesson state machine and one truthful Cognition projection. Keep GitHub relay restricted until its scoped successor is independently reviewed.

**P1/P2 — Resolve C8/C9/C10 for product foundation.** Document mind/mirror consent, implement compatibility-preserving neutral schema, release manifest and synthetic two-install test. Mobile remains a later authenticated-client project, not a Core replacement milestone.

**Stop the entire integration release for:** cross-entity data exposure, unverified entity mapping on an execution route, Identity bypass, lost/replayed side effect, corrupted state after failed update, a model action reported EXECUTED without a real receipt, or inability to restore the prior owner installation. These are not cosmetic.

### Reuse versus rewrite recommendation

**[INFER] Targeted boundary repair, not a whole-core rewrite.** The already configurable per-install data roots, GitHub settings, host ports and model URLs are good foundations. The highest-risk debt is **identity correctness, authoritative state ownership and observable action correlation**, not the Go language or visual styling. Do not expand learning, team automation, new entities or mobile clients until these boundaries have evidence.

## 14. Unresolved owner decisions — do not silently decide

1. Explicitly choose the pinned Thought-engine as canonical for the integrated TASS runtime, while preserving historic Thought and Thought Controller as separate reference projects?
2. Confirm Processor /dev as the **one** active integrated Development workspace, with Developer standalone a future authorized skill/service and Development repository a design/prototype hub?
3. Whether sharing descriptive mind/lesson revisions to private GitHub is opt-in per installation; what data classes remain never synced?
4. Whether newly tested/approved skills require an additional explicit autonomous-use grant/approval even when a general tool capability is allowed? **Default: manual approval until decided.**
5. Prioritize a secure, isolated second **synthetic** installation after C2 is closed, before any actual KYSSA connection. No real person's account or data without consent.

**Document owner:** TASS Core / Thought-Processor integration review. **Resolution mechanism:** link each actual repair PR and acceptance report back to this document and issue #89. No one should mark a conflict closed unless its own evidence, negative tests and necessary owner decision are visible.