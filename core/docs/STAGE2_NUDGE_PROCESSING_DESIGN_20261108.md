# Stage 2 design: one supervised nudge → one verified Thought development

**Status:** DESIGN FOR REVIEW. Not approved, not implemented, nothing deployed. Answers [Issue #100](https://github.com/Arameair/Thought-Processor/issues/100).
**Author:** Claude, 2026-11-08. **Reviewer next:** Codex (independent audit, §11), then GPT-6 and the owner.
**Baseline (read, not changed):** Core `094f43ba2bc167ace8835bf286f37951fee432c9` (PR #98), Thought Engine `d69d3917d1601d806d4b2ba2254693e1adb92b6b`, Identity `64fdf8e8cf926df5ea6bb6c23fbab864d0f09e3b` (PR #42). Codex acceptance of the draft-only stage: PR #99.

**Goal (unchanged from #100):** the owner picks one unexpired DRAFT nudge addressed to `self` on one real persistent Thought, approves it once, and gets **one** Thought development from the local model (Nemotron) with a truthful receipt. Nothing else.

---

## 1. Verdict on compatibility

**Compatible.** The existing Engine `Think` is the right primitive: it runs one model request, adds one `developed` step and one Note carrying the model `request_id`, and never applies the suggested state. It needs a **narrow nudge variant**, not a new queue or job system, because four facts in the pinned code make plain `think` unsafe for run-once:

| Fact in pinned Engine `d69d391` | Consequence |
|---|---|
| `Think` is **synchronous** inside the HTTP request and holds `workMu` for the whole model call (`engine.go` L1331–1419). Core's Engine client and the Engine's model client both allow **10 minutes**. | The browser must not wait on it. Core must return quickly and run the Engine call in the background. |
| `think` has **no idempotency**: two calls = two model requests. `ModelLink.ids` is in-memory only (`model.go` L341–347). | A durable claim is needed **in the Engine** before the model call. |
| The model `request_id` is generated **inside** `ModelLink.run` and written to `model-requests.jsonl` **only after the model answers** (`record` after `ask`; no fsync; rotates at 5 MB). | After a crash mid-call there is no request record. The Engine must pre-generate the request ID and store it in the claim, or nothing ties a crash to a request. |
| Model access is authorized with `thought.process` (twice: before and after waiting for the model slot), which is a **default** Identity grant. | That check is real but is not owner consent for nudges. The nudge-specific grant must be checked separately. |

Also relevant: the Engine's per-entity **Thought Processing** switch (`s.grant.Enabled`) must be ON, and `Think` refuses **resolved** Thoughts (sleeping is allowed). The Engine heartbeat (off by default) can also develop Thoughts; `workMu` serializes it with nudges.

## 2. Smallest change set

| Repo | Change | Size |
|---|---|---|
| **Identity** | One allowlisted entry `cognition.nudge-process` → grant `cognition.nudge.process`, same pattern as `cognition.nudge-draft` (#42): not default, not implied by anything, enable needs passphrase, revoke immediate, no `skill.discover`. | ~15 lines + 1 test |
| **Engine** | (a) `POST /api/entities/{eid}/thoughts/{tid}/nudge` and `GET /api/entities/{eid}/nudges/{nid}`. (b) `nudge-runs.jsonl` claim journal per entity. (c) `Think` gains an internal variant that takes the owner's attention text, a pre-generated request ID and the nudge ID. (d) `ThoughtStep` and `Note` gain `source_nudge_id` (omitempty, backward compatible). (e) Worker/ModelLink accept a caller-supplied request ID. Plain `think` behavior is unchanged. | ~200 lines + tests |
| **Core** | (a) `POST /api/nudges/{id}/process` and processing state in `GET /api/nudges`. (b) `nudge-runs.jsonl` receipt journal (the draft journal is never rewritten). (c) A background dispatcher plus reconciliation on start and on read. (d) UI: "Process once…" confirm on a `self` DRAFT; receipt view. (e) `tass.sh` pin bump. | ~300 lines + tests |

**Not added:** a queue, job scheduler, new service, new model, five-position routing, Discord, autonomy, GitHub sync of nudge content, applying suggested states, or attempt numbering (§4.5).

## 3. Authorization flow

Every step fails closed. "No call" means the model is not contacted.

```
Owner clicks "Process once…" on a DRAFT (recipient self) → confirm dialog shows the immutable fields
  │ POST /api/nudges/{nudge_id}/process  {body_digest}        page token + local Origin + JSON (existing pagePost)
  ▼
Core C1  draft exists, owner = current Identity entity, Engine link unchanged, recipient = self,
         not expired, body_digest == stored digest, forced TASS_CHAT_ONLY is off      → else 400/409/403, no call
Core C2  Identity: cognition.nudge.process (fresh decision)                           → else 403, no call, nothing written
Core C3  existing process record for this nudge? → return it (200). Never sends twice.
Core C4  append SUBMITTED{decision_id,…} to nudge-runs.jsonl, fsync                   → write fails: 500, no call
         reply 202 to the page; background: POST Engine …/thoughts/{tid}/nudge
  ▼
Engine E1 key + local + JSON (existing write()); entity exists; Thought belongs to it; not resolved;
          Thought Processing ON; attention ≤ 500 chars and sha256 matches               → 4xx, NO claim, no call
Engine E2 existing claim for (entity, nudge_id)? same digest → return it, never call; other digest → 409
Engine E3 Identity: cognition.nudge.process (Engine's own service key, defense in depth) → denied: no claim, no call
Engine E4 append CLAIMED{nudge_id, thought, digest, request_id(pre-generated)} to nudge-runs.jsonl, fsync
          → fsync fails: 503, no claim, no call
Engine E5 ModelLink.run(thought.process): authorize, wait for slot, authorize again, ask        (existing)
Engine E6 parse NOTE/STATE (existing); re-check cognition.nudge.process  [owner decision D1]
Engine E7 save Thought: Note + developed step with source_nudge_id and request_id (existing atomic tx)
Engine E8 append OUTCOME{COMPLETED|FAILED, …} to nudge-runs.jsonl, fsync; return the run
  ▼
Core C5  append OUTCOME to Core's nudge-runs.jsonl; then VERIFY (§5) before showing COMPLETED
```

**CHAT_ONLY:** follow the existing skills precedent (`skills.go` L168, L197). Supervised processing is allowed while the owner's Autonomous switch is OFF (the page shows `CHAT_ONLY`), and refused when `TASS_CHAT_ONLY=on` forces chat-only. This is owner decision **D2**.

**Revocation semantics (stated, testable):**
- revoked before C2 or E3 → refused, no claim, no call;
- revoked while the request waits for the model slot → `ModelLink` re-check denies, no call, outcome `FAILED` (`model_called: no`);
- revoked while the model is answering → the call cannot be cancelled. With **D1 = discard** (recommended), E6 sees the revocation, the note is **not** saved, and the outcome is `FAILED` (`revoked_after_call`, `model_called: yes`, request ID recorded).

## 4. Run-once design

### 4.1 The safety property

**At most one model request per nudge ID, across concurrent requests and restarts of either service.** Exactly-once completion across two services is **not** promised: a crash can leave a nudge `INTERRUPTED`, and it is never retried automatically.

### 4.2 Who is authoritative

- **Engine claim journal = truth about calls.** Only the Engine knows whether it claimed and called. Duplicate protection lives here, so even a Core bug or a lost Core journal cannot cause a second call.
- **Core receipt journal = the owner's record.** Core never sends a second process request for a nudge that has a `SUBMITTED` record, and it reconciles by **reading** the Engine, never by re-sending.

### 4.3 Engine journal `<entity dir>/nudge-runs.jsonl` (0600, append-only, fsync per line, newline repair like Core's, never rotated; refuse new claims past 4 MB)

```json
{"schema":"thought-engine.nudge-run/1","event":"CLAIMED","nudge_id":"nudge_…","entity_id":"…","thought_id":"…",
 "attention_sha256":"…","attention_chars":123,"request_id":"mr_…","at":"…","identity_decision":"dec_…"}
{"schema":"thought-engine.nudge-run/1","event":"OUTCOME","nudge_id":"nudge_…","state":"COMPLETED",
 "model_called":"yes","request_id":"mr_…","step_at":"…","suggested_state":"keep","applied":false,"at":"…"}
```
`state` ∈ `COMPLETED | FAILED`; `FAILED` carries `reason` ∈ `model_denied | model_error | unreadable_reply | revoked_after_call | save_failed` and `model_called` ∈ `yes | no`. **The attention text is not stored here**: it is in the prompt, which `model-requests.jsonl` already records (trimmed), as today.

**Concurrency:** E2 and E4 run under `e.mu` with an in-memory `running` set, so the second of two concurrent requests sees the claim and returns it (`RUNNING`). `workMu` then serializes the model call as today.

### 4.4 Reading a run after a restart (`GET /api/entities/{eid}/nudges/{nid}`)

| Journal shows | Thought file shows | Engine reports |
|---|---|---|
| nothing | — | **404 NOT_CLAIMED** |
| CLAIMED, OUTCOME | — | that OUTCOME |
| CLAIMED, in `running` set (same process) | — | **RUNNING** |
| CLAIMED, not running (restart), no OUTCOME | step with this `source_nudge_id` | **COMPLETED (recovered)**: the Thought save landed, the OUTCOME append didn't. The Engine appends the OUTCOME now. |
| same | no such step | **INTERRUPTED**, `model_called: unknown`, request ID shown. Never retried. If `model-requests.jsonl` has that request ID, show `model_called: yes` and its result. |

### 4.5 One attempt per nudge (simplification)

A nudge is processed **once**, whatever happens. To try again the owner uses **"Copy as new draft"** (new nudge ID, same fields). This removes attempt keys, retry counters and "which attempt is current" from both journals. All refusals before E4 (C1–C2, E1–E3) leave **no** claim, so after fixing the cause the same nudge can still be processed. Core records only `SUBMITTED` once it is about to send.

### 4.6 Core journal `<data>/nudge-runs.jsonl` (0600, append-only, fsync, same writer as `nudges.jsonl`)

```json
{"schema":"thought-processor.nudge-run/1","event":"SUBMITTED","nudge_id":"…","owner_entity_id":"…","engine_entity_id":"…",
 "thought_id":"…","body_digest":"…","process_decision_id":"dec_…","at":"…"}
{"schema":"thought-processor.nudge-run/1","event":"OUTCOME","nudge_id":"…","engine_state":"COMPLETED","model_called":"yes",
 "request_id":"mr_…","verified":true,"verify_detail":"1 developed step with source_nudge_id and request_id","at":"…"}
```

### 4.7 Crash and failure matrix

| # | When it breaks | Engine claim | Model calls | Owner sees | Automatic retry |
|---|---|---|---|---|---|
| F1 | Core denies (C1/C2) | none | 0 | 4xx reason | n/a (nothing recorded; fix and click again) |
| F2 | Core SUBMITTED write fails | none | 0 | 500 "nothing was sent" | no |
| F3 | Core crashes after SUBMITTED, before sending | none | 0 | `SUBMITTED` → reconcile reads **NOT_CLAIMED** → shown as **NOT CLAIMED (as of time)**, re-read on each view | **no**: never re-sent; "Copy as new draft" |
| F4 | Engine refuses before claim (E1–E3) | none | 0 | `FAILED` with Engine's reason, `model_called: no` | no |
| F5 | Claim fsync fails (E4) | none | 0 | `FAILED` `claim_not_saved`, `model_called: no` | no |
| F6 | Engine crashes after claim, before or during the model call | CLAIMED | 0 or 1 (unknown) | **INTERRUPTED**, `model_called: unknown` (or `yes` if the request ID is in `model-requests.jsonl`) | **no** |
| F7 | Model error / timeout / unreadable reply | CLAIMED + FAILED | 1 | `FAILED`, request ID, reason | no |
| F8 | Thought save fails | CLAIMED + FAILED `save_failed` | 1 | `FAILED`, request ID | no |
| F9 | Crash after Thought save, before OUTCOME | CLAIMED | 1 | **COMPLETED (recovered)** from the Thought file | n/a |
| F10 | Core times out / crashes while the Engine runs | CLAIMED (→ OUTCOME) | 1 | `SUBMITTED` → reconcile reads RUNNING, then the OUTCOME | no |
| F11 | Duplicate / concurrent POST to Core | — | ≤1 | the same receipt | n/a |
| F12 | Duplicate POST to Engine (Core bug or lost journal) | existing | ≤1 | the existing run | n/a |
| F13 | Grant revoked mid-call (D1 = discard) | CLAIMED + FAILED `revoked_after_call` | 1 | `FAILED`, note not saved | no |
| F14 | Owner relinks Core to another Engine entity | unchanged | — | the old nudge disappears from views (owner + link scoped); processing it is refused (C1) | n/a |

F3 has one narrow race: Core sent the request and crashed, the TCP connection dropped, but the Engine handler keeps running and claims a moment after Core's first reconcile read. That is safe because Core never re-sends: the next read shows the claim, and the status simply moves from NOT CLAIMED to RUNNING to the outcome. No status is ever final while the Engine could still claim.

## 5. What counts as COMPLETED

Core shows **COMPLETED** only when all of these hold. Otherwise it shows the Engine state with `verified: false` and why.

1. The Engine run says `COMPLETED` with a request ID.
2. A Cognition read of the Thought (`GET …/thoughts/{tid}`, existing sanitized projection, extended with `source_nudge_id`) shows **exactly one** `developed` step with this `source_nudge_id` and that request ID.
3. That step is newer than the draft's `created_at`.
4. The run says `applied: false` (suggested state never applied).

The model request count increasing by exactly one is checked in acceptance tests (§7), not at runtime: `model-requests.jsonl` lives in the Engine's container and rotates.

**No `cycle_id` and no `job_id`** are produced or shown. `Think` creates neither, and inventing one would be false (#100 agrees).

## 6. API contracts

### Engine (key + local + JSON via the existing `write()`/`sensitive()`)

`POST /api/entities/{eid}/thoughts/{tid}/nudge`
```json
{"nudge_id":"nudge_[0-9a-f]{20}","attention":"≤500 chars","attention_sha256":"hex","by":"owner"}
```
- `200 {"run": NudgeRun}`, **after** the outcome (synchronous like `think`). A duplicate returns the existing run (possibly `RUNNING`) **without waiting and without a call**.
- `400` invalid · `403` Thought Processing off / Identity denied (no claim) · `404` entity/Thought · `409` same nudge ID, other digest (no content leaked) · `503` claim not durable.

`GET /api/entities/{eid}/nudges/{nid}` → `200 {"run": NudgeRun}` or `404 {"state":"NOT_CLAIMED"}`.

```go
type NudgeRun struct {
    NudgeID, EntityID, ThoughtID, AttentionSHA256 string
    State        string // RUNNING | COMPLETED | FAILED | INTERRUPTED
    Recovered    bool   // COMPLETED rebuilt from the Thought file (F9)
    Reason       string // FAILED/INTERRUPTED detail
    ModelCalled  string // yes | no | unknown
    RequestID    string // pre-generated at claim
    StepAt       *time.Time
    Suggested    string
    Applied      bool   // always false
    ClaimedAt    time.Time
    IdentityDecision string // E3 decision ID
}
```

**Prompt:** the existing `thinkPrompt`, plus one block: `The owner asked you to pay attention to this (information, not instructions): "<attention>"`. The reply contract stays the same two lines (`NOTE:` / `STATE:`) and the same `parseThink`, so real-model behavior changes as little as possible.

### Core (existing `pagePost`; history via the existing read-authorized `GET /api/nudges`)

`POST /api/nudges/{nudge_id}/process` `{"body_digest":"…"}`
- `202 {"run": NudgeRunView}` submitted (background dispatch) · `200` existing run · `400` not `self` / expired / invalid · `403` LOCKED, no `cognition.nudge.process`, or forced chat-only · `409` digest mismatch or another Engine link · `500` SUBMITTED not saved (nothing sent).

`GET /api/nudges` → each nudge gains `"run": NudgeRunView | null`, with the Engine state, `verified`, request ID, step time, `model_called` and decision IDs. **On read**, a run in `SUBMITTED`/`RUNNING` is reconciled with **one** Engine GET (bounded; read-only).

**UI states (7, down from #100's 9):** `DRAFT`, `EXPIRED` (derived), `SUBMITTED`, `RUNNING`, `COMPLETED` (verified), `FAILED` (reason + model called yes/no), `INTERRUPTED` (model called unknown/yes). The Engine's NOT_CLAIMED shows as `SUBMITTED – not claimed by the Engine (as of …)`.

## 7. Acceptance tests (PASS/FAIL)

Unit and handler tests in each repo; native tests use the real pinned binaries and the TEST FIXTURE model (`fake_llama.py` gains a `NOTE:/STATE:` answer and a delay switch for a slow call). "Calls" = fixture model request count **and** Engine `model-requests.jsonl` lines.

| ID | Given | PASS only if |
|---|---|---|
| S1 | Identity entry `cognition.nudge-process` | not default; not implied by `thought.process`, `cognition.nudge.draft`, skill or junction grants; enable needs passphrase; revoke immediate |
| S2 | each denial: LOCKED, no process grant, only draft grant, recipient position-1..5, expired, digest mismatch, relinked, other entity's Thought, resolved Thought, Thought Processing OFF, forced `TASS_CHAT_ONLY=on`, bad token/Origin/JSON | right 4xx; **0 calls**; no Engine claim; no Core SUBMITTED (Core-side denials) |
| S3 | happy path (fixture) | 202 → COMPLETED verified; **1 call**; exactly 1 new Note + 1 `developed` step with the same `source_nudge_id` and request ID; `applied:false`; no new cycle or job; GET after both restart shows the same |
| S4 | same POST again (Core) after COMPLETED; and 10 concurrent POSTs before | same run; **total 1 call** |
| S5 | direct duplicate POST to Engine (simulated Core bug), same and different digest | same run, 0 extra calls / 409 without content |
| S6 | Core killed after SUBMITTED, before send (seam) | after restart: SUBMITTED, NOT_CLAIMED; **0 calls**; never sent |
| S7 | Core killed during the Engine call (slow fixture) | after restart: RUNNING → COMPLETED by reconciliation reads; **1 call** |
| S8 | Engine killed during the model call (slow fixture) | after restart: INTERRUPTED; Core shows it; **no new call on restart or on any read** |
| S9 | Engine crash after Thought save, before OUTCOME (putHook seam) | COMPLETED (recovered); 1 call |
| S10 | claim fsync failure (seam) | 503; no claim; 0 calls; the nudge can be processed later |
| S11 | model returns an unreadable reply / repeated text | FAILED `unreadable_reply`; 1 call; no note; no retry |
| S12 | grant revoked while the request waits for the slot / while the model answers | `model_called:no` / FAILED `revoked_after_call` with the note not saved (D1) |
| S13 | heartbeat ON during processing | the nudge's step is identified by `source_nudge_id`; the heartbeat's own steps don't affect verification |
| S14 | regressions | draft-only tests, plain `think`, mailbox native, pulse native, lesson flow, Cognition reads, browser fixtures, updater 14/14, Go vet/test/race in all three repos |
| S15 | privacy | attention text not in the Engine claim journal, Core logs, GitHub or `tass.md`; journals 0600 |
| **O1** | **owner, real Nemotron, after review + CI + owner approval** | preflight (Thought not resolved, Thought Processing ON, counts noted) → enable `cognition.nudge-process` → Process once → COMPLETED verified; `model-requests.jsonl` +1; Thought shows the new note; survives a restart. Then revoke the grant. |

O1 is **not** run during design review.

## 8. Where #100 is wrong, over-built or missing something

1. **"ModelRequest ID" as evidence.** The request ID is generated inside `ModelLink.run` and logged only after the answer, without fsync, in a 5 MB rotating file. After a crash it can't be found. **Fix:** pre-generate it and store it in the Engine claim (§4.3).
2. **Nine UI states.** `PREPARED` adds nothing (a draft is already immutable). `FAILED_NOT_SENT` can't be determined honestly, because a connection error does not prove the request wasn't sent. **Seven states** (§6), with an explicit `model_called: yes|no|unknown`.
3. **Attempt keys.** Retries need attempt numbering, "current attempt" and duplicate-attempt rules in two journals. **One attempt per nudge** plus "Copy as new draft" gives the same owner power with far less to get wrong (§4.5).
4. **"Expected source revision".** The Thought can change between draft and processing (a heartbeat, or another nudge). Refusing then is fragile and gains nothing. **Record both revisions; refuse only resolved Thoughts** (owner decision D4).
5. **"Existing Engine worker also checks thought.process".** True, but `thought.process` is a **default** grant, so it does not express nudge consent. Hence the Engine-side `cognition.nudge.process` check (E3).
6. **Missing: the synchronous 10-minute `Think`.** #100 doesn't say who waits. The page must not: Core returns 202 and reconciles (§3).
7. **Missing: Thought Processing switch and heartbeat.** The Engine's per-entity switch must be ON (it is a separate gate from Identity). The heartbeat can also develop the same Thought; verification by `source_nudge_id` keeps them apart (S13).
8. **Missing: definite vs uncertain outcomes.** #100 asks to distinguish `claimed-but-not-called` from `model-failed` from `succeeded-but-save-failed`. While the Engine is alive all three are definite (F5, F7, F8). Only a **crash** after the claim is uncertain (F6), and F9 recovers the success case from the Thought file.
9. **Real-model risk not mentioned.** `parseThink` rejects notes that repeat existing text, and Nemotron already drifted from line contracts once (#71). O1 can fail as `unreadable_reply` without any bug. That is a valid, truthful FAILED, and the reason to keep the prompt contract unchanged.

## 9. Owner decisions (not invented here)

| ID | Question | Recommendation |
|---|---|---|
| D1 | The grant is revoked while the model is answering: save the note or discard it? | **Discard**: FAILED `revoked_after_call`; the request is recorded. |
| D2 | Allow supervised processing while the Autonomous switch is OFF (`CHAT_ONLY`)? | **Yes**, like supervised skills; refuse when `TASS_CHAT_ONLY=on` is forced. |
| D3 | One attempt per nudge; retry = "Copy as new draft"? | **Yes.** |
| D4 | Refuse if the Thought changed since the draft? | **No**: record both revisions; refuse only resolved Thoughts. |
| D5 | Should the Engine also check `cognition.nudge.process`? | **Yes** (E3), using its existing Identity client. |
| D6 | Is a confirm dialog per run enough, or should each run ask for the passphrase? | **Dialog.** Enabling the grant already needs the passphrase. |
| D7 | How long to keep run journals? | Keep all for now (bounded at 4 MB, refuse new runs past that, like drafts). |
| D8 | For O1: is the Engine's Thought Processing switch ON for TASS's Engine entity on your machine, and is `identity-test` unresolved? | Check in the read-only preflight. |

## 10. Implementation order once approved (separate draft PRs, CI, pinned SHAs)

1. Identity: `cognition.nudge-process` (+ S1).
2. Engine: claim journal, nudge route + run read, request-ID pass-through, `source_nudge_id`, reconciliation on load (+ S3–S13 at Engine level).
3. Core: process route, run journal, dispatcher, reconciliation, verification, UI, pins (+ S2–S15 native).
4. Codex audit of the implementation, then owner O1.

## 11. Handoff for Codex (independent audit of this design)

Please check against the pinned source, not this document's summary:

1. **Is the at-most-once claim real?** Try to get two model calls for one nudge ID: concurrent Core POSTs, Core restart between C4 and send, Engine restart between E4 and E5, a duplicate Engine POST while RUNNING, journal damage, and two Engine processes on one data folder (the Engine already has `lock.go`; confirm it covers this).
2. **Can anything be called COMPLETED falsely?** Heartbeat steps, a recovered F9 run whose Thought file was rolled back by `tx.rollback`, a step from an older nudge with the same Thought.
3. **Authorization gaps.** Does any path call the model without a fresh `cognition.nudge.process` decision (Core C2 and Engine E3)? Is the revoke-mid-call behavior (D1) implementable given `ModelLink.run`'s structure?
4. **The F3 race.** Is "Core never re-sends; status is read-derived" enough, or can a UI state become final while the Engine could still claim?
5. **Simplifications (§8).** Challenge them: are seven states, one attempt per nudge, and no source-revision check safe, or do they hide a failure the owner needs to see?
6. **Test plan.** Mark each of S1–S15 PASS-able as written, or say what is missing.

Return PASS / FAIL / OPEN per item with code references. **No implementation, deployment, grant change or real nudge processing in the audit.**
