# Engineering handoff — manual Cognition cards, Liaison nudges and evidence-driven loops

**Date:** 2026-11-08  
**Type:** Detailed team instructions and acceptance contract, **not** executable authorization.  
**Owner decision source:** [TASS five-card/nudge discussion #25](https://github.com/Arameair/TASS/issues/25), [manual prototype issue #96](https://github.com/Arameair/Thought-Processor/issues/96), [Cognition #82](https://github.com/Arameair/Thought-Processor/issues/82), [supervised Talk #95](https://github.com/Arameair/Thought-Processor/issues/95).

**Canonical owner decision sheet:** [TASS Owner Nudge Cognition Handoff](https://github.com/Arameair/TASS/blob/docs/owner-nudge-cognition-handoff-20261108/docs/OWNER_NUDGE_COGNITION_HANDOFF_20261108.md).

## 1. Immediate objective and failure mode

Build a minimal **manual nudge workbench inside the existing Cognition UI**. The owner can inspect real Engine persistent Thoughts as cards, choose whom to nudge and for what, save a local record, and see its evidence-linked result. This lays the foundation for an efficient dynamic loop but is deliberately not the entire five-card autonomous system.

**Weakest point:** `/cognition` is currently read-only. Attaching a "Nudge" button to an existing GET/read-only projection must **not** accidentally turn viewing into a model request, Skill execution, authorization bypass, GitHub upload or Discord send.

**Owner-confirmed architecture:**
- Liaison is **one of five card positions**, not a separate sixth model or process.
- Liaison reads **who needs attention / what / from whom**, routing **only changed/useful information**, avoiding wasteful model requests.
- Internal and (later) explicitly authorized external recipients must be possible using the same nudge concept.
- Use **real Cognition persistent Thoughts** as initial manually readable work items. Nudge output should be inspectable and chainable into further manual nudges.

**Not authorized now:** runtime installation, enabling autonomy, posting to Discord, automatically running a task, editing `live:tass.md`, Identity grant changes, or full dynamic five-card orchestration.

## 2. Inspected baseline — what exists versus what does not

Repository candidate at the time of review: **Core PR #88 HEAD `8e040b33e7a492f042a1f4449b8c7b013c227965`**, paired experimental Engine `d69d3917d1601d806d4b2ba2254693e1adb92b6b` and Identity `7da7f7c5bbecc42d3a7e9316a2037243eda10b88`. **Pin from actual PR/owner host before editing/running tests; these may change.**

| Location | Verified from code | Consequence |
|---|---|---|
| `core/tass/cognition.go` | Scoped Thought and cycle cards, exact IDs, verified/observed/unknown provenance, owner + Engine link checks | Reuse Thought projection; do not create fake Thought records from cycles |
| `core/tass/cognition_pulse.go` | Read-only, bounded pulse observation | No model job should be triggered merely by reading pulse |
| `core/tass/cognition_learning.go` | Lesson learning and test evidence distinguished from skill grants | No "learned means runnable" shortcuts |
| `core/cmd/tass-core/cognition.html` | Read-only shelves, thought details and immutable playback | Add a separate nudge workflow without changing semantics of historical playback |
| `core/tass/skills.go` | Existing supervised compiled `@lesson-workspace.list-lessons` skill | Does not authorize a generic nudge executor; inspect current dispatch/receipt code |
| `core/engine/cognition.go` | Read-only actual Engine Thoughts/cycles/journal API | Treat Engine as authoritative for Thought IDs and processing outcomes |

The owner reported the Go stack running after a later upgrade, but **we have not independently verified current local deployed commit pins, saved Thoughts, local Identity state or owner acceptance for manual nudges**.

**Do not claim**: five physical simultaneous GPU workers, active nudge execution, a completed autonomous thought loop, a working Discord Talk skill, or real nudge token savings.

## 3. First deliverable: card reader and draft workflow (no model call)

### UI (inside existing `/cognition` page)

- Preserve **Thoughts**, **Cycles**, **Lessons**, **Skills**, **Outcomes** shelves and existing read-only selection/playback.
- On a scoped real persistent Thought, add **Read Card**, **Manual Nudge**, **View Nudge History**. Source data remains immutable.
- Read detail: exact persistent `thought_id`, owner-linked Engine identity, topic/question, state, history, related cycle IDs, record time and verification label. No guessed joins or private prompt dump.
- Manual Nudge draft: display source Thought; authenticated current sender is fixed, never a free-form arbitrary impersonation field; target choice initially **TASS local / approved card-position reference** (not a made-up persistent Thought); reason and desired next action; optional priority/expiry with bounded defaults. Distinguish **role/position** from **Thought ID**.
- **Save Draft** performs one authorized local write and returns durable record; never queues model work.
- **Process** is visibly **BLOCKED / NOT AVAILABLE** until there is an authorized local submission endpoint and checked Identity capability. Do **not** implement an unguarded fake completion button.
- **View Result** shows real queue/Engine receipt (only if actual processing happened), exact event and Thought/cycle IDs, or **NOT RECORDED**. Manual chaining is possible by selecting a real result Thought.
- Explicit statuses for denied/expired/blocked/inactive/missing Engine link; no fake 'thinking' UI. Never auto-collapse owner-inspected details while polling.

### Local record contract (PROPOSED schema, not an existing endpoint)

```json
{
  "nudge_id": "generated-locally",
  "owner_entity_id": "verified-canonical-identity",
  "source_thought_id": "canonical-engine-thought-id",
  "source_engine_entity_id": "owner-reviewed-engine-link",
  "sender_identity_id": "authenticated-current-sender",
  "recipient_kind": "card-position-or-local-entity",
  "recipient_id": "authorized-target",
  "reason_code": "new-evidence-or-owner-request",
  "requested_action": "bounded-owner-authored-text",
  "source_evidence_refs": ["validated-id"],
  "created_at": "UTC timestamp",
  "expires_at": "UTC timestamp",
  "idempotency_key": "generated-stable-key",
  "status": "DRAFT",
  "processing_cycle_id": null,
  "result_thought_id": null,
  "authorization_decision_id": null,
  "source_revision": "pinned-reference"
}
```

This is *illustrative field-level design*, not approved schema or real content. Avoid leaking actual personal or security identifiers into public docs/logs. Validate all string lengths, allowable ID patterns, recipient ownership and numeric bounds at the Go boundary.

**Persistence:** local, owner-scoped, append-only state-transition receipt or equivalent durable transactional journal; atomic writes/backup and restart recovery. Verify no cross-entity list/read. Preserve original Nudge and immutable source Thought, use new events for corrections; never rewrite history to pretend a work item succeeded.

## 4. Optional second deliverable: one explicitly authorized local nudge

Only after draft CRUD and status UX pass:

1. Inspect existing Core submission/Engine job APIs; identify **existing Identity action type** and its scope. Do not use `conversation`/`thought.process` read authorization as proof of nudge write/processing authorization.
2. If no suitable grant exists, stop at **draft-only** and open a separate Identity-policy review issue. Do **not** invent capability names as if approved.
3. Owner explicitly submits a selected, immutable nudge. On submit re-check owner state, entity scope, source revision/Thought ID, expiry, selected destination, rate/quota/cooldown, and current Identity permission.
4. Create durable queue receipt **before** dispatch; use idempotency across double-click, retries and restart; bound active jobs. Reject stale/wrong-entity/fabricated cycle references.
5. Route to exactly **one** chosen local recipient on first test; no external model/service calls outside already authorized local Nemotron path. Never automatically broadcast to all five or duplicate the same job.
6. Record state transitions **only as events actually occur**: `DRAFT` → `QUEUED` → `PROCESSING` → `COMPLETED | NO_ACTION | BLOCKED | FAILED`. No false `COMPLETED` just because an HTTP request was accepted.
7. Attach actual Engine `cycle_id`, `thought_id`, model request ID, Identity decision ID and resource usage when truly recorded. If Engine did not produce a new persistent Thought, **show NOT RECORDED** and preserve the nudge/response anyway.
8. Result projection remains **read-only**; repeated GETs do not enqueue, and the original historical playback still cannot execute anything.

## 5. Liaison nudge routing — later guarded experiment, not P1 requirement

**Owner's efficiency goal:** A lightweight Go status read should determine whom to nudge, why and on whose behalf. **No model inference just to decide nothing changed.**

For later design:
- Use local change detection, pending dependencies, dedupe keys, cooldown and per-recipient quotas to suppress unnecessary work.
- Keep five **logical positions** in a common frozen snapshot; only targets with work should invoke model inference. Physical simultaneous inference is not guaranteed by the single local GPU.
- The Liaison is **within five**. The **whether role rotates among cards** remains open.
- The old design phrase “all five fire each batch” might mean five participating slots rather than five model calls. This requires an owner-facing experiment to resolve; never silently pick the expensive interpretation.
- Cap hop count, cycle depth, recipient queue, TTL and retention; prevent feedback loops, outdated revisits and message storms.
- Observe `NO_NUDGE` decisions through bounded diagnostics rather than generating a full event and full model summary every idle tick.
- Card/graph variants from Development must show **measured** differential results, not model predictions of what would have happened.

## 6. Security and privacy requirements

1. Use current Identity unlock + entity-scoped authorization for each sensitive read and **fresh write/processing action**; if unavailable, fail closed.
2. A source Thought is owned by its Engine entity. Only read it through the owner-reviewed Identity→Engine link. No name-only linking, guessed IDs or cross-identity searches.
3. Never treat imported Markdown, `tass.md`, a Brain Builder graph, a nudge, a GitHub Issue or a Discord message as a grant.
4. No private messages, raw prompts, model chain-of-thought, credential URLs, session tokens, private memory or personalized notes in GitHub issues/tests/screenshots.
5. Reply/transport to Discord is **out of scope**; Bridge `queued` ≠ delivered and uncertain sends must never be blindly retried when Talk #95 is added later.
6. Never modify `live:tass.md` or local memory as an incidental side-effect of drafting a nudge. TASS may propose identity diffs but cannot canonically apply them without owner approval under current policy.
7. Ensure **read-only Cognition requests** remain read-only. Add distinct POST endpoints only after reviewing exact authorization, CSRF/origin, page-token and loopback checks; do not weaken existing controls.
8. Save/restart must preserve original evidence, permit clean recovery, and not replay previously processed nudge without idempotent reconciliation.

## 7. Required deterministic tests (new code not yet written)

| ID | Given | Expected |
|---|---|---|
| M1 | Authorized linked persistent Thought, clicked Read | Real fields shown; **zero model/skill/write calls** |
| M2 | Cycles exist but no persistent Thoughts | Honest `NOT RECORDED`; no invented `thought:` cards |
| M3 | Read + draft nudge + refresh/restart | Exact source/target/owner persisted; **zero inference** |
| M4 | Duplicate draft/save/process with same key | At most one operation/queued job; no duplicate receipt/outbound |
| M5 | LOCKED or missing owner-reviewed Engine link | Denied and no cross-entity data |
| M6 | Wrong entity or invalid/forged source Thought ID | Denied and no target data |
| M7 | No processing Identity grant or no valid route | BLOCKED; no model job, no fake success |
| M8 | Queued/processing/result receipts in fixture | Real state labels and exact correlation; missing ID says `NOT RECORDED` |
| M9 | Nudge result `NO_ACTION` or failed Engine request | Visible truthful outcome; never marked verified execution |
| M10 | 2 unchanged snapshots | No repeated useful nudge; no unnecessary model calls in routing implementation |
| M11 | Bounded hop/expiry/cooldown and full queue | No infinite bounce, safe failure and observable backpressure |
| M12 | Attempt external Discord/Claude/GPT-6 recipient in v1 | Rejected: zero external calls |
| M13 | Same Thought + 2 manual nudge targets in controlled lab | Compare actual route/receipt and measured tokens/latency; no claimed savings without measurements |
| M14 | Copy/export sanitized nudge evidence | No credentials, private model prompts or cross-identity data |
| M15 | Existing skills, lessons and Cognition GET playback | All prior read/security/owner-review behavior preserved |

**Test evidence categories are mandatory:** UNIT/FIXTURE, CONTAINER/INTEGRATION, REAL NEMOTRON, OWNER-HOST, REAL DISCORD (out of scope); each marked `PASS/FAIL/BLOCKED/NOT RUN` independently. No false aggregation.

**Proposed validation, run from the appropriate Go module checkout if changed:**
```sh
go vet ./...
go test ./...
go test -race ./...
```
These commands are **proposed**, **not executed in writing this document**; record exact working directory, commit SHA, environment, result, errors and elapsed time in PR evidence. Browser/UI tests and actual owner-machine acceptance are separate.

## 8. Exact owner acceptance walkthrough — once implemented

1. Owner verifies installed Core/Engine/Identity SHAs, working backup and current Identity settings, no autonomous or Discord action enabled.
2. Open actual Cognition page; read one **real** persistent Thought with exact `thought_id` and linked evidence. If none, accept empty state; do not create a fake.
3. Create **one manual nudge draft**, save, refresh, restart if safely approved, verify same record and no Engine request.
4. When approved processing exists, explicitly choose a single local target and process. Inspect Identity permit decision and recorded queue/Engine/result IDs.
5. Read the **actual resulting card or receipt**; verify status against journal. Repeat with a different manual target and compare.
6. Confirm no unrequested messages, skill calls, new permissions, identity edits, model job on GET, or duplicate delivery.
7. Capture sanitized commands, complete pins, tests, result correlation, token/latency measurements or `NOT RECORDED`. Return `PASS/FAIL/BLOCKED/NOT RUN`.
8. Any backend or UI defect remains a blocker with issue and reproduction steps, not claimed PASS.

## 9. Responsibilities and handoff structure

- **Core engineer:** minimal Go types, HTTP authorization, persisted local drafts/status, existing Cognition UI enhancement; no second application.
- **Engine engineer:** only if processing scope approved, one bounded local job interface and exact cycle/Thought provenance; preserve historical read-only API.
- **Identity engineer:** review *real* write/processing grant semantics separately; never infer grant from capability documentation.
- **UX architect:** Read → Nudge → Draft → gated Process → View Result, high information density, source and result differentiation, visible no-work/error states.
- **Test Architect / Codex:** independent negative auth/isolation, replay/dedupe, restart, token-cost assertions and regression proof. Record results to PR.
- **Claude / external reviewers:** inspect outdated assumptions, nudge economy, identity data privacy and contradictory five-batch semantics; propose small changes in isolated PRs.
- **Owner:** approves new write capability, real processing/test, rollout and future autonomy separately. Inspect actual receipts, not model prose.

**Sequence:** (A) inspect, document current actual runtime → (B) manually read/write DRAFT only → (C) verify owner UI/durable receipt → (D) separately review local processing grant → (E) one local supervised nudge → (F) compare two manual thought-loop paths → (G) only later Liaison automatic routing and Talk.

## 10. Explicitly unresolved and deferred

- Exact five Markdown files per thought/team and their relation to five positions or persistent Thoughts.
- Whether Liaison's **role** rotates between positions; confirmed only it is one of five.
- Dynamic schedule: full five-position participation versus selectively invoked model requests; real cost/latency evidence needed.
- Scorable 20-second predictions and what happens with GPU overruns/late feedback.
- Trigger and approval for changes to working card text versus protected identity/instruction content; rollback rules.
- GitHub two-way mailbox relationship and future outside-service delivery policy.
- Notebook cell execution as a skill; three supervised lessons approved do **not** establish runnable Jupyter.
- Autonomous `tass.md` revisions and approved skill selection are **future owner-approved phases**, not active state.

**Do not implement a sixth coordinator, do not submit a hidden external message, do not change Identity grants or deployment, and do not claim owner acceptance from fixture test results.**

**Handoff status:** DESIGN READY FOR EXPERIMENTAL IMPLEMENTATION; CODE/TESTS/OWNER ACCEPTANCE FOR NUDGES **NOT DONE**.
