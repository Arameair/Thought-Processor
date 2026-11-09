# Cognition → Development → Cognition: independent design review and proposed design (2026-11-09)

> **"Cognition shows what was produced, Development lets you experiment in how it's produced."**

**What this is.** This document does three things:
- Claude's independent review of GPT-6's engineering brief ([#106 comment 6748905265](https://github.com/Arameair/Thought-Processor/issues/106#issuecomment-6748905265));
- an independent check of the source code it describes;
- a revised design for one Cognition → Development → Cognition cycle.

**What this is not.** It is design only. Nothing was implemented, merged or deployed. The running installation was not touched, no permissions were changed, no autonomy was enabled, and no model requests were made. No private Thought text, Note text, nudge text or credentials appear in this document.

**Disposition (details in §9): DESIGN NEEDS REVISION.** The brief points in the right direction. However, its weakest assumption is wrong in a way that would make the first experiment uninformative (§1). It also mis-locates three of the defects it cites (§2.2). This document carries those revisions:
- **P0** (see the result) and **P1** (see how it was produced) are ready for the owner to approve for implementation;
- **P2** (one isolated experiment) is ready once the owner answers decisions **D4–D7**.

## 0. Source revisions reviewed

| Repository | Revision | What was read |
| --- | --- | --- |
| Arameair/Thought-Processor (Core) | `99f95be3315f7cfc97ab9064c3974d89c9b70f5e` (PR #104, the installed candidate) | `core/cmd/tass-core/cognition.html`, `dev.html`, `main.go`; `core/tass/cognition.go`, `nudge.go`, `nudge_run.go`, `junction.go`, `brain.go`, `core.go`; `core/engine/cognition.go`, `nudge.go`; `core/identitysvc/identitysvc.go` |
| Arameair/Thought-engine | `8bed0fc522905b3c2e110a0b426f4e258812a9b9` (PR #19) | `engine.go`, `nudge.go`, `prompts.go`, `model.go`, `worker.go`, `server.go` |
| Arameair/identity | `cfc4079b690e2f7dc57afb9f14c6832ae38fdf54` (PR #43) | `internal/entity/bound.go`, `skills.go`, `cmd/tass-identity/main.go` |
| Arameair/Development | `af0a2ed0a0fb2f39c6f6a93f74184d6c44ebecbb` (`experiment/go-brain-builder-prototype-20261105`) | `brain-builder/README.md` |
| Stage 2 spec | `3d4080876f0bb63d731a26fe804045c812a3ad07` (PR #101) | D1–D16 |
| Codex audit | PR #105 head `0c0f921` | ST2-I01 retest PASS |
| Issues | #106 (all 10 entries to 19:09Z), #100 (to 18:38Z) | owner observations, GPT-6 brief, acceptance results |

All line numbers below are at these revisions.

---

## 1. The weakest assumption, and why it changes the experiment

The brief treats the short, inadequate Note as a question of **answer quality** and proposes the nudge's attention text as the first variable to change. The source shows that the workflow which produced the Note was **never asked to answer the Thought's question**. What the Engine sends for a nudge:

- **The instruction** (`thought-engine/prompts.go:50-68`, `thinkPrompt`, reused by the nudge path at `nudge.go:631`):
  - *"Add ONE new idea, possible explanation, or thing to watch for next. Do not repeat the evidence or earlier notes."*
  - The reply format is fixed: `NOTE: <at most 40 words>` and `STATE: <keep, sleep or resolve>`.
- **The model call** (`nudge.go:777-790`, `model.go:427-430`): `max_tokens` 256, `thinking` off, and **`temperature 0`** (every `engine:*` purpose is sampled deterministically).
- **The context** (`nudge.go:610-630`):
  - the Thought's question;
  - the summaries of its **last ≤ 8 traces** (observation summaries, not documents);
  - its **last ≤ 5 notes**;
  - the owner's attention text, which the prompt labels *"information, not instructions"* (`nudge.go:632`).
- **What is not in the context.** No skill runs, no notebook or lesson content is retrieved, and no memory search happens. If a notebook's purpose was not already written into one of the eight trace summaries, the model never saw it.
- **Clipping** (`prompts.go:116-121`): the parser keeps the `NOTE:` line, clipped to 400 characters.

**Consequence.** For a question like "what is this notebook skill's purpose?", a short, non-answering Note is the *expected* output of this workflow, whatever the model's quality. Changing the attention text cannot remove a 40-word limit, an instruction to add one idea instead of answering, or the absence of the notebook content. An attention-text experiment would most likely produce another short Note and teach the owner nothing about why.

This is exactly what Development should show: the observable stages of **how** it was produced. The first experiment should vary **the one stage the owner can see is responsible**. See §6 for the recommended variable and the alternatives.

**Persistence vs reasoning, using the `identity-test` example.** These are separate claims and must stay separate:
- The Stage 2 run proved **persistence and correlation**: one request, one Note and one `developed` step sharing `worker_request_id` and `source_nudge_id`, 42 → 43 steps, and the suggested state not applied.
- It proved **nothing about useful reasoning**: the workflow wasn't asking for an answer.
- It proved nothing about learning either: nothing in this path changes future behaviour except that the Note now appears in "earlier notes" of later prompts.

---

## 2. Verified current implementation (evidence map)

`PRESENT` means present and working as described. `PRESENT-WRONG` means present but misleading or defective. `MISSING` means it doesn't exist. `HYPOTHESIS` means not verifiable from source.

### 2.1 Data path for the developed Note

| # | Item | Status | Evidence |
| --- | --- | --- | --- |
| 1 | The Engine stores Note text with `at`, `text`, `by`, `worker_request_id`, `suggested_state`, `source_nudge_id` | PRESENT | `engine.go:164-171`; nudge save in `nudge.go` (P2 block) |
| 2 | The Engine stores the `developed` step with `detail` (= worker name + Note text) and `worker_request_id` / `source_nudge_id` | PRESENT | `engine.go:151-162` |
| 3 | `GET /api/entities/{eid}/thoughts/{tid}` returns the full Thought, Notes included, behind the Engine key (`sensitive`) | PRESENT | `server.go:247`, `server.go:489-` |
| 4 | **The Note text already reaches Core**: `GET /api/entities/{eid}` (the library) and the Thought detail both carry `notes[].text` | PRESENT | Engine `Thought.Notes []Note`; Core reads both, `tass/cognition.go:148-`, `:296` |
| 5 | Core **discards** the Note text when decoding: `ThoughtNote` keeps only the IDs | PRESENT (deliberate) | `core/engine/cognition.go:61-65` |
| 6 | Core's `CognitionThought` projection has no `notes` at all | MISSING | `tass/cognition.go:58-71` |
| 7 | Core's history projection **drops `source_nudge_id`** and `detail`. It copies only `At, Kind, EventID, TraceID, RequestID`, with the comment *"No free-form Detail: includes private worker-generated strings"* | PRESENT-WRONG (nudge ID lost) / deliberate (detail) | `tass/cognition.go:302-308` |
| 8 | History is truncated to the **first** 64 steps, so the **newest** developments are the ones dropped. `identity-test` already has 43 steps. | PRESENT-WRONG | `tass/cognition.go:303` |
| 9 | The UI labels each step `event_id \|\| trace_id \|\| "NOT RECORDED"` and ignores `worker_request_id`, so a correlated development reads "NOT RECORDED" | PRESENT-WRONG | `cognition.html:216` |
| 10 | Rendering uses `textContent` only (`mk`, `field`); there is no `innerHTML` in the page | PRESENT | `cognition.html:45`, `:97-99` |
| 11 | The Engine caps Notes at the last 200, so an old Note **can be evicted** | PRESENT | `engine.go:1405-1407` |

### 2.2 Status and navigation (corrections to the brief)

| # | Item | Status | Evidence | Correction to the brief |
| --- | --- | --- | --- | --- |
| 12 | The nudge-history list shows `Processing: BLOCKED — No Identity action for nudge processing exists yet…` | PRESENT-WRONG | `NudgeList.Processing = nudgeBlocked()` (`tass/nudge.go:135-138`, `:497`); shown at `cognition.html:349` | It is the **nudge-history list field**, not the page's top banner. It is a hard-coded string, not a derived state. |
| 13 | Each self-addressed nudge shows `AVAILABLE` **without checking** the grant, the lock state or the Engine | PRESENT-WRONG | `tass/nudge.go:385-389` | The brief calls AVAILABLE correct. It is optimistic text: the real check happens only at admission (C1/C2). |
| 14 | The draft form always shows a disabled "Process — BLOCKED: no Identity processing grant exists yet" | PRESENT-WRONG | `cognition.html:327-328`; stale comments at `:39-40`, `:301` | Confirmed. (This is my own Stage 2 omission.) |
| 15 | "Process once…" / Stop / Copy as new on saved drafts | PRESENT | `cognition.html:389-439` | — |
| 16 | Copy as new reports the new draft ID only in a toast and doesn't focus it | PRESENT-WRONG (UX) | `cognition.html:429-438` | — |
| 17 | The COMPLETED receipt shows the request ID and "Suggested state (never applied)", but has **no link to the Note** | MISSING | `cognition.html:378-383` (`runFields`) | — |
| 18 | "Try adjustment in Development" opens `/dev?cognition_ref=thought:<id>&brain_revision=<n>` | PRESENT | `cognition.html:228-234` | — |
| 19 | `/dev` shows the reference only as the "UNEXECUTED DEVELOPMENT PROPOSAL" text; no run is loaded | PRESENT (honest, but empty) | `dev.html:298-310` | — |
| 20 | The Brain graph (`tass/brain.go`) is Core-local and advisory: **the Engine never reads it** | PRESENT | `brain.go:1-16`; no Engine reference | Confirmed: changing the graph changes no inference. |
| 21 | **Junction lab**: store an alternative cell, run active and proposal on the same input under one Identity decision, record both outputs, and label *"EFFECT ON TASS'S THINKING NOT MEASURED"* | PRESENT | `tass/junction.go:584-652`, `dev.html:312-350` | **Not mentioned in the brief.** It is an existing, owner-tested propose → compare → record pattern that P2 should reuse rather than reinvent. |
| 22 | Development repo brain builder: separate localhost prototype, "not connected to TASS, Identity, model execution, or a notebook executor" | PRESENT (separate) | `brain-builder/README.md` @ `af0a2ed` | Confirmed; already ported into Core as `brain.go` / `dev.html`. Not needed for this cycle. |

### 2.3 Evidence of how the result was produced (what P1 can honestly show)

| # | Stage | What is recorded per run | Where | Status |
| --- | --- | --- | --- | --- |
| 23 | Thought at the moment of processing | the revision (`UpdatedAt`) at P1 and P2 | Engine `nudge-runs/<id>/outcome.json` (`revision_p1/p2`) | PRESENT |
| 24 | Owner attention | the text (Core draft journal) and its SHA-256 (Engine `claim.json`) | `data/core/nudges.jsonl`, `claim.json` | PRESENT |
| 25 | Authorization | the draft decision, the process decision and Identity's redeem record | nudge record, `claim.json` (`identity_decision`, `identity_redeem`), Identity `decisions.jsonl` | PRESENT |
| 26 | Context actually sent (evidence + notes + question) | the **logged prompt**, with the attention replaced by its digest, **clipped to 2,000 characters** with a trailing `…` | Engine `calls.jsonl` (`worker.go:75-87`, `nudge.go:788-789`) | PRESENT, possibly PARTIAL |
| 27 | Instruction template and output format | not stored per run; defined by the code at the Engine commit | `prompts.go:50-68` @ Engine commit | CODE-DEFINED |
| 28 | Model parameters (256 tokens, thinking off, temperature 0) | not stored per run; defined by the code | `nudge.go:787`, `model.go:427-430` | CODE-DEFINED |
| 29 | Model HTTP | `http.json` (attempt), `response.json` (OK, elapsed, `reply_sha256`) | `nudge-runs/<id>/` | PRESENT |
| 30 | Raw model reply | `calls.jsonl` `reply` (clipped to 1,000 chars); `model-requests.jsonl` `text` (full) | `nudge.go:788-789`, `model.go:413` | PRESENT |
| 31 | Parse | `NOTE` line clipped to 400; `STATE` becomes the suggestion | `prompts.go:116-` | CODE-DEFINED |
| 32 | Save + Core verification | the Note and step; Core's `verified` flag | Thought file; Core receipts | PRESENT |
| 33 | Memory retrieval, skill or notebook execution, branching | **not part of this path** | — | NOT PART OF THIS PATH |
| 34 | Model's internal reasoning | thinking is off; nothing exists to show | — | NOT RECORDED (does not exist) |
| 35 | `GET /api/worker/calls` | last 100 calls, **in memory, all entities**, lost on restart | `worker.go:385-406`, `server.go:408` | PRESENT, **unsafe for owner views** (not entity-scoped, not durable) |
| 36 | Whether the `identity-test` evidence summaries mention the notebook's purpose | — | owner's local data | HYPOTHESIS (not inspected; private) |

### 2.4 Authorization today

- **Core Cognition reads.** `cognitionAccess()` (`tass/cognition.go:113-142`) needs Identity UNLOCKED and the grants `conversation` and `thought.process`, decided with detail `thought-processor:cognition.read`. It also needs the reviewed Identity → Engine link. Cross-entity results are refused: the Thought's `EntityID` must equal the linked entity (`:297`). The nudge journal is filtered by owner **and** Engine entity.
- **Identity grants are readable without minting anything.** Identity's public `/v1/state` lists the entity's grants (`identitysvc.go:31-44`). Core can therefore **display** "processing grant: enabled / not enabled" without minting a decision. Authorization still happens only at C1/C2.
- **Bound actions.** These exist for `cognition.nudge.process` and `cognition.nudge.stop` (`identity/internal/entity/bound.go:23-`). No action exists for experiments.

---

## 3. Proposed contracts and security boundaries

### 3.1 P0: the Thought detail projection (Core only; no Engine or Identity change)

`GET /api/cognition/thoughts/{id}` (existing route, same `cognitionAccess()` and entity-link checks) gains:

```jsonc
{
  "thought_id": "…", "question": "…", "state": "active",
  "history_total": 43,                       // new: total steps in the Engine record
  "history": [ /* newest 64 steps, oldest first; truncation drops the OLDEST (fixes #8) */
    { "at": "…", "kind": "developed", "event_id": "", "trace_id": "",
      "worker_request_id": "mr_…", "source_nudge_id": "nudge_…", "by": "owner-nudge" } ],   // nudge ID no longer dropped (#7)
  "developments": [                          // new: one entry per worker_request_id, newest first, max 50
    { "request_id": "mr_…",
      "at": "…",
      "note": { "text": "<≤400 chars, as stored>", "by": "…", "suggested_state": "keep",
                "suggested_state_applied": false },          // never applied: no code path applies it
      "step": { "at": "…", "kind": "developed" },            // or null when no matching step
      "source_nudge_id": "nudge_…",                          // or ""
      "persistence": "NOTE_AND_STEP_CORRELATED",             // | NOTE_ONLY | STEP_ONLY (evicted note) — never "useful"
      "nudge_receipt": "COMPLETED_VERIFIED" } ],              // Core's own receipt, when a nudge exists; else ""
  "developments_total": 1, "developments_truncated": false
}
```

The rules:

- **The library list (`/api/cognition/library`) never carries Note text.** Only the single-Thought detail does. This keeps Note text out of any list-shaped response.
- **Copy Evidence, both sanitized downloads and GitHub sync never include Note text** (they are GitHub-safe by design). Nor does Identity's audit, or any Core log.
- **Request IDs are matched to `^mr_[0-9a-f]{16}$` before display.** A Note whose request ID doesn't match is shown with "request ID NOT RECORDED", never invented.
- `step.detail` stays out. The Note is the single source of the model's text, so it is shown once.
- **This deliberately reverses one earlier rule (D1).** The comment at `tass/cognition.go:304` withheld worker-generated text. The reversal is scoped: owner-only, loopback, page token, `Cache-Control: no-store`, rendered as text.

**Nudge status, made truthful.** `NudgeList.Processing` stops being a constant. It becomes a derived, display-only summary:

```jsonc
"processing": {
  "identity": "UNLOCKED" | "LOCKED" | "UNKNOWN",
  "process_grant": "ENABLED" | "NOT_ENABLED" | "UNKNOWN",   // read from /v1/state grants; no decision minted
  "engine": "REACHABLE" | "UNREACHABLE",
  "locked": "<processing_locked reason or empty>",
  "active_run": "nudge_…" | "",
  "summary": "Ready: a saved self-addressed draft can be processed once." | "<the first blocking reason, in plain words>"
}
```

Per nudge, `processing.state` shows only what can be known:
- `DRAFT_READY`, `NOT_FOR_PROCESSING` (non-self recipient), `PRE_RESTORE` or `EXPIRED`;
- or the run state from the receipt.

Nothing reads "AVAILABLE" unless the summary is Ready.

### 3.2 P1: the run record (one small read endpoint in the Engine, one in Core)

- **Engine (new, read-only)**: `GET /api/entities/{eid}/nudges/{nid}/record`. It is `sensitive` (Engine key). It returns:
  - the four run files (`claim`, `http`, `response`, `outcome`);
  - the `calls.jsonl` record and the `model-requests.jsonl` record **whose request ID equals the claim's `request_id`**.

  The lookup reads the files from the end, is bounded (at most 8 MiB scanned per file, and stops at the first match), and only returns records whose `entity_id` / `ref` equal `{eid}` / the claim's Thought. It writes nothing. It never uses the in-memory `/api/worker/calls` (#35).
- **Core (new)**: `GET /api/development/runs/{nudge_id}`. It applies the same checks as `NudgeHistory`:
  - `cognitionAccess()`;
  - the nudge belongs to this owner **and** the linked Engine entity;
  - the ID matches `^nudge_[0-9a-f]{20}$`.

  It resolves everything server-side; no query parameter can choose another entity. It returns a list of **stages**:

```jsonc
{ "nudge_id": "nudge_…", "thought_id": "…", "engine_commit": "8bed0fc…",
  "stages": [
    { "stage": "thought",       "status": "RECORDED",      "detail": { "question": "…", "revision_p1": "…" } },
    { "stage": "attention",     "status": "RECORDED",      "detail": { "text": "<owner's own words>", "sha256_matches_claim": true } },
    { "stage": "authorization", "status": "RECORDED",      "detail": { "draft_decision": "dec_…", "process_decision": "dec_…", "redeem": "…" } },
    { "stage": "context_sent",  "status": "RECORDED" | "PARTIAL", "detail": { "prompt": "<exact prompt as sent; attention re-inserted only if its SHA matches>", "clipped_at": 2000 } },
    { "stage": "instruction",   "status": "CODE-DEFINED",  "detail": { "at_engine_commit": "8bed0fc…", "summary": "add ONE new idea … NOTE ≤ 40 words; STATE keep|sleep|resolve" } },
    { "stage": "model_call",    "status": "RECORDED",      "detail": { "request_id": "mr_…", "max_tokens": 256, "thinking": false, "temperature": 0, "params_source": "CODE-DEFINED", "elapsed_ms": 0, "reply_sha256": "…" } },
    { "stage": "model_reply",   "status": "RECORDED",      "detail": { "raw": "<full reply from model-requests.jsonl>" } },
    { "stage": "parse",         "status": "CODE-DEFINED",  "detail": { "note_kept": "…", "suggested_state": "keep", "applied": false } },
    { "stage": "saved",         "status": "RECORDED",      "detail": { "note_at": "…", "step_at": "…", "verified": true } },
    { "stage": "memory_retrieval", "status": "NOT PART OF THIS PATH" },
    { "stage": "skills",           "status": "NOT PART OF THIS PATH" },
    { "stage": "model_reasoning",  "status": "NOT RECORDED", "why": "thinking is off for this call; no reasoning text exists" } ] }
```

Status vocabulary: `RECORDED` / `PARTIAL` / `CODE-DEFINED` / `NOT RECORDED` / `NOT PART OF THIS PATH`. Each is defined on screen in one sentence. Nothing in this endpoint calls a model, mints a bound decision or writes a file.

### 3.3 P2: one isolated experiment (new Identity action, new Engine route, new Core journal)

These follow the Stage 2 patterns exactly, so nothing new has to be invented for safety.

- **Identity.** New bound action `cognition.experiment.run`, with owner grant `cognition.experiment-run` (allowlisted, not a default, revocable).
  - It is single-use, TTL 120 s, held in memory, minted by `core` and redeemed by `engine` (D7).
  - Binding fields, in order: `action, owner_entity_id, engine_entity_id, thought_id, experiment_id, arm, snapshot_sha256, variable, variant_sha256`.
- **Core journal** (`data/core/experiments.jsonl`, append-only, local, never synced) stores the plan:

```jsonc
{ "experiment_id": "exp_<20 hex>", "thought_id": "…", "baseline_nudge_id": "nudge_…",
  "baseline_request_id": "mr_…", "snapshot": "AS_OF_BASELINE" | "AS_OF_NOW",
  "variable": "instruction",           // exactly one variable per experiment (§6)
  "control": "<current instruction line, from the Engine commit>",
  "variant": "<owner text, ≤ 300 chars>", "variant_sha256": "…",
  "created_at": "…", "idempotency_key": "…" }
```

- **Engine route** `POST /api/entities/{eid}/thoughts/{tid}/experiments/{exp}/arms/{A|B}` (Engine key + bound decision). It:
  1. redeems the decision;
  2. builds the snapshot prompt (§6.2) and checks that its SHA equals the bound `snapshot_sha256`;
  3. writes `experiments/<exp>/<arm>/claim.json` with O_EXCL;
  4. runs P1 (Identity re-check, after the model slot is taken);
  5. writes `http.json`, sends **one** request (`purpose: engine:experiment`, same parameters as the baseline: 256 tokens, thinking off, temperature 0), then writes `response.json`;
  6. writes `result.json` with the reply, the parsed NOTE/STATE and the reply SHA, then `outcome.json`.

  What it never does:
  - **write to the Thought, its Notes or its history**;
  - retry;
  - apply a state.

  How interruptions are handled:
  - **Restart:** UNCERTAIN / RECOVERY_REQUIRED are derived from the files exactly as for nudges, and the startup scan covers `experiments/`.
  - **Run-time Thought gate:** none, because nothing is saved into the Thought. The Thought must still exist and belong to `{eid}` (D6).
- **Admission (Core).** The same `admitMu` and **one active run across nudges and experiments**: the model slot is shared, and C4 counts both. Arms run in order (A, then B). Each arm needs its own confirmation click and its own bound decision. B is never sent automatically after A (D5).
- **Restore.** The fence logic (`restore-fence.json`) is extended to `experiments.jsonl`: plans from before a restore are `PRE_RESTORE` and can only be copied, never run.

### 3.4 Boundaries kept

| Responsibility | Owner | This design |
| --- | --- | --- |
| Authorization | Identity | Reads: existing `cognitionAccess`. Experiments: a new bound action. Display of grant state is display only. |
| Execution and Thought records | Engine | P0: no change. P1: one read endpoint. P2: experiment records live in the Engine under `experiments/`, never in the Thought. |
| Owner interface and projections | Core (Thought Processor) | All projections, journals, admission and the UI. |
| Controlled experiments | Development (in Core's `/dev`) | Reuses the Junction-lab pattern; the brain graph stays advisory and is labelled so. |

---

## 4. Owner workflow (click by click)

`[NOW]` = works at `99f95be`. `[P0]`/`[P1]`/`[P2]` = needs that phase.

**Cognition: see what was produced**
1. Open Cognition → the `identity-test` Thought card → the side sheet. `[NOW]`
2. The sheet opens on **What TASS has written** (not "Inspect recorded details"). `[P0]` It shows:
   - the question, then the newest development first: the full Note, *when*, *by* (Nemotron, owner nudge), the request ID and the nudge ID;
   - a persistence line: "Saved and correlated: one note and one step share request mr_…".
   - Second line: "This shows that the Note was saved, not that it answers the question."
   - "Suggested: keep — not applied."
3. **Nudges for this Thought.** The COMPLETED receipt now has **Read what TASS wrote**, which scrolls to and highlights the same development entry. `[P0]` (Today the receipt shows IDs only.)
   - The list header reads, for example, "Ready: a saved self-addressed draft can be processed once", or the true blocking reason. The stale "BLOCKED — no Identity action" text is gone. `[P0]`
   - The draft form has no disabled Process button. It says: "Saving makes no model call. Process it from the saved draft below." `[P0]`
   - **Copy as new** focuses the new draft and names it. `[P0]`
4. **Advanced** (collapsed): the full step history (with "trace NOT RECORDED · request mr_…" instead of "NOT RECORDED"), traces and cycles. `[P0]` (Today these show at the top level.)

**Development: see how it was produced**

5. On the development entry, press **Investigate in Development**. `[P1]` The URL is `/dev?run=nudge_…`, and Core resolves it server-side.
   - Today the button is the generic "Try adjustment in Development", which shows only the "UNEXECUTED DEVELOPMENT PROPOSAL" text. `[NOW]`
6. Development opens on **How this Note was produced**: the vertical stage list from §3.2, with the status of each stage. `[P1]`
   - The owner can read the exact prompt that was sent and the raw reply, and sees the instruction "add ONE new idea … at most 40 words".
   - Stages that didn't happen are listed as such ("Memory retrieval — not part of this path").
   - The brain graph remains below, headed "Brain graph (advisory; the Engine does not read it)".
   - One next action is shown: **Try one change…** `[P2]`

**Experiment: change one thing and compare**

7. **Try one change…** opens a small form `[P2]`:
   - **What to change:** the instruction line (pre-filled with the current line).
   - **Snapshot:** "As it was at the original run (verified)", or "As it is now" if the original can't be reproduced (§6.2).
   - **Cost:** "2 model requests (A: unchanged, B: your change), one at a time, each with your confirmation."
   - **What won't change:** the Thought, its Notes and its history.
   - **Save experiment** writes the plan to `experiments.jsonl`. No model call is made.
8. **Run A (unchanged) once…** → **Confirm**. `[P2]` Afterwards the A receipt shows: "Reproduced the original reply: yes/no (SHA match)".
9. **Run B (your change) once…** → **Confirm**. `[P2]`

**Back to Cognition: compare**

10. **Compare** (Development, also linked from Cognition). `[P2]` Three columns, original Note | A | B, each with:
    - the full text;
    - the request ID;
    - the technical status (COMPLETED / FAILED / UNCERTAIN);
    - the cost (one request each);
    - "Saved into the Thought: original yes · A no · B no".
11. The owner rates B against A on three criteria: answers the question / relevant / useful. Each rating is better, same, worse or can't tell, plus an optional note. The ratings are saved as **owner assessment** and labelled as such. `[P2]`
12. Back in Cognition, the Thought shows an **Experiments (isolated)** section listing `exp_…` with the owner's verdict and a link back to the comparison. `[P2]`
    - "Promote" does not exist in this cycle. A better variant is evidence for a reviewed Engine change through GitHub, not a switch (§8, D8).

---

## 5. Minimal changes, by phase

**P0 — Cognition: see what was produced (Core only)**
- `core/engine/cognition.go`: decode `ThoughtNote{At, Text, By, RequestID, SuggestedState, NudgeID}`; keep the `ThoughtStep` fields.
- `core/tass/cognition.go`:
  - add `developments`, `history_total`, `developments_total`;
  - copy `NudgeID` into history;
  - keep the **newest** 64 steps;
  - keep Note text **out of** `CognitionLibrary` and the exports.
- `core/tass/nudge.go`: replace `nudgeBlocked()` and the per-nudge `AVAILABLE` with derived states (§3.1). Grant status comes from `IdentitySvc.State().Entity.Grants`, display only.
- `core/cmd/tass-core/cognition.html`:
  - "What TASS has written" first;
  - the receipt → Note deep link (anchor by request ID);
  - Advanced collapsed;
  - remove the disabled Process button and the stale comments (`:39-40`, `:301`);
  - focus on Copy as new;
  - step label `trace NOT RECORDED · request mr_…`.
- **Stops when** the §7 P0 rows PASS, including the owner's half-screen walkthrough.

**P1 — Development: see how it was produced**
- Engine: `GET …/nudges/{nid}/record` (§3.2), with a bounded reverse scan of `calls.jsonl` and `model-requests.jsonl`. Tests: entity and Thought mismatch, missing records, clipped prompt, oversize file.
- Core:
  - `GET /api/development/runs/{nudge_id}`;
  - attention re-inserted only on a SHA match;
  - the `instruction` stage summarised from a Core-side constant that names the pinned Engine commit;
  - a Core test fails if the pin changes without that summary being updated.
- `dev.html`: when `run=` is present, a "How this Note was produced" section above the graph replaces the "UNEXECUTED DEVELOPMENT PROPOSAL" banner (which stays for `thought:` / `cycle:` / `junction:` refs). Cognition gains "Investigate in Development" on each development entry that has a nudge ID.
- **Stops when** the §7 P1 rows PASS.

**P2 — one isolated experiment** (after D4–D7)
- Identity: the `cognition.experiment.run` bound action, allowlist and tests (the same three lifecycle tests as #43).
- Engine:
  - the experiments route;
  - the snapshot builder (§6.2);
  - run files under `experiments/`;
  - the startup scan;
  - fixture tests for the Stage 2 crash points; it never touches the Thought.
- Core:
  - `experiments.jsonl`, admission shared with nudges, receipts, the fence extension;
  - `/dev` experiment form, run buttons and comparison view;
  - the Cognition "Experiments (isolated)" section;
  - owner assessment records.
- **Stops when** the §7 P2 rows PASS on the fixture model, and then **one** owner-approved real run of A and B on `identity-test` is recorded (a separate approval, D9).

No new service, scheduler, repository, graph editor or notebook executor. The Development repo's standalone builder is not used.

---

## 6. The experiment protocol

### 6.1 The one variable

**Recommended: the instruction line**, i.e. the sentence that tells the model what to produce. Today it reads *"Add ONE new idea, possible explanation, or thing to watch for next. Do not repeat the evidence or earlier notes."*

- The owner edits this one sentence (≤ 300 chars).
- Everything else is held constant:
  - evidence, notes and question (the snapshot);
  - the attention text;
  - the output contract (`NOTE: <at most 40 words>` / `STATE: …`, not editable, so the parser and the 400-character clip stay valid);
  - 256 tokens, thinking off, temperature 0.
- Example B for `identity-test`: *"Answer the Thought's question directly from the evidence; if the evidence doesn't contain the answer, say exactly what is missing."*

**Alternatives** (the owner chooses, D4):

| Variable | For | Against |
| --- | --- | --- |
| Instruction line (recommended) | Targets the stage the evidence shows is responsible (§1); one sentence; contract unchanged | Needs a template slot in the Engine (small) |
| Attention text (GPT-6's choice) | No template change: the existing nudge input | The prompt calls attention "information, not instructions", so it most likely can't override "add one idea / 40 words"; a likely null result |
| Context: add the notebook's description as one more evidence line (via the existing lesson-metadata junction handler) | Tests "the model lacked the facts" | Introduces retrieval: a second subsystem, and the composition is harder to keep to one variable |
| Word limit 40 → 120 | Trivial | Doesn't address "add an idea instead of answering" |

Run the context variant as a **second** experiment only if B answers poorly because facts are missing. Each experiment changes one thing.

### 6.2 The snapshot (comparable conditions)

1. **AS_OF_BASELINE (preferred).** The Engine rebuilds the inputs of the original run:
   - Notes with `at` < the baseline step time; the last ≤ 5 are used;
   - traces in `trace_ids` created at or before `revision_p1`; the last ≤ 8 are used;
   - the question; the owner's attention (from Core; its SHA must equal `claim.json`).

   It then renders the **logged** form and compares it to the baseline's `calls.jsonl` prompt.
   - Identical, with no `…` clip → `VERIFIED`.
   - Clipped, with the first 2,000 characters identical → `VERIFIED-PREFIX` (shown as such).
   - Otherwise → `NOT REPRODUCIBLE`, and the owner is offered AS_OF_NOW.
2. **AS_OF_NOW.** The current Thought state, labelled "not the original conditions; the original Note is shown for reference only". A and B are still exactly comparable with each other.

### 6.3 Arms, controls, evidence

- **A (control): the snapshot with the current instruction.**
  - With AS_OF_BASELINE, temperature 0, the same model build and the same `llama.cpp` server, A **should** reproduce the original reply. `reply_sha256(A) == response.json.reply_sha256(baseline)` is shown as "Reproduced: yes".
  - "No" means the comparison carries a warning: model or server drift, non-determinism, or a different build. B is then compared with **A**, not with the original.
- **B (treatment):** identical except the one instruction sentence.
- **Recorded for each arm:** the request ID, the full reply, the parsed NOTE/STATE, the reply SHA, elapsed time, and the run state. The plan records the Engine commit, the Core commit and the model identity (`ModelResult.Model`).
- **Cost:** exactly 2 model requests per experiment, shown before saving. There is no retry. FAILED / UNCERTAIN arms are shown and never re-run; only "Copy as new experiment" can try again, with a new ID and a new approval.

### 6.4 Evaluation (what the owner decides, and what is claimed)

- **Technical PASS:** both arms COMPLETED, parsed, Thought unchanged (same `updated_at`, notes count and history count before and after), exactly 2 model requests.
- **Owner assessment:** answers the question / relevant / useful. Each is rated better, same, worse or can't tell, B against A, with an optional note. It is stored as `owner_assessment` and displayed as such.
- **What may be claimed:** "With this snapshot and these settings, the changed instruction produced a reply the owner rated as better / same / worse."
- **What may not be claimed:** that TASS learned, that the change generalises to other Thoughts, or that production behaviour changed. With temperature 0, one run per arm is the deterministic output for that input, but it is still a single input. Generalising would need the same experiment on more Thoughts (a later cycle).
- **Fixture-model runs prove the plumbing only.** The comparison view says so whenever the model identity isn't the real local model.

---

## 7. Acceptance criteria and negative/security tests

Result definitions:
- **PASS**: observed as specified, with evidence attached.
- **FAIL**: observed otherwise.
- **BLOCKED**: couldn't run because of a named external condition.
- **NOT RUN**: not attempted.

Docker, owner-machine and real-Nemotron rows are never PASS from a workspace without them.

| ID | Phase | Check | How it is verified |
| --- | --- | --- | --- |
| A1 | P0 | Opening the Thought shows the full Note text, its time, author, request ID and nudge ID, with no raw JSON | Browser fixture + owner walkthrough |
| A2 | P0 | The COMPLETED receipt's "Read what TASS wrote" lands on the development entry with the **same** request ID | Browser fixture: compare IDs |
| A3 | P0 | No "BLOCKED — no Identity action" text anywhere; the header state follows Identity (unlocked/locked), the grant (on/off) and the Engine (up/down) across 6 combinations | Unit test + browser fixture |
| A4 | P0 | No disabled Process button in the draft form; Copy as new focuses and names the new draft, and makes no POST beyond `/api/nudges` | Browser fixture + request log |
| A5 | P0 | **Reads make no model request** (Cognition open, Thought detail, nudge history, refresh, polling) | Fixture-model POST counter = 0 |
| A6 | P0 | The library response and both sanitized exports contain **no Note text** (search for a unique marker planted in a fixture Note) | Unit test |
| A7 | P0 | Note text is rendered as text: a fixture Note with `<img src=x onerror=…>` and `</script>` shows literally and runs nothing | Browser fixture |
| A8 | P0 | Cross-entity: a Thought ID from another Engine entity, or a relinked Identity → 403/404, no text | Unit test with two entities |
| A9 | P0 | 70-step fixture Thought: the newest development is visible; `history_total` = 70; truncation drops the oldest | Unit test |
| A10 | P0 | A Note evicted by the 200 cap → the development shows `STEP_ONLY` ("note no longer stored"), never invented | Unit test |
| A11 | P0 | Half-screen (≈ 960×1000) and 390 px wide: the question, newest Note and next action are visible without horizontal scrolling | Playwright screenshots + owner walkthrough |
| B1 | P1 | `/dev?run=nudge_…` shows the stages of **that** run; the Thought ID, request ID and nudge ID match the Cognition entry | Browser fixture |
| B2 | P1 | Every stage carries one status from the vocabulary; memory/skills say "not part of this path"; model reasoning says "not recorded" with the reason | Unit test on the projection |
| B3 | P1 | The prompt shown equals the bytes sent to the fixture model (fixture records the last body); attention re-inserted only on a SHA match; a tampered journal shows "attention digest mismatch" | Native e2e |
| B4 | P1 | A prompt longer than 2,000 chars → `PARTIAL`, with the clip point shown | Native e2e with a long fixture |
| B5 | P1 | A forged or other-entity `run=` value, a nudge of another owner, or a non-matching ID → refused, nothing shown | Unit + native |
| B6 | P1 | The Engine record endpoint refuses mismatched entity/Thought/request; never reads the in-memory call list; bounded scan on a 50 MiB `calls.jsonl` | Engine unit tests |
| B7 | P1 | Opening Development makes no model request and no write (data folder checksum unchanged) | Native e2e |
| C1 | P2 | Experiment saved → 0 model requests; A → exactly 1; B → exactly 1; the total for one experiment is 2 | Fixture-model counter |
| C2 | P2 | **Original immutability:** the Thought's `updated_at`, notes count, history count and file SHA are identical before and after A and B | Native e2e |
| C3 | P2 | No experiment grant → 403, nothing written; grant revoked between A and B → B refused; LOCKED → refused | Native e2e |
| C4 | P2 | A bound decision for arm A can't run B, another experiment, another Thought, or a changed variant (any binding field changed → 403, no folder) | Identity + Engine tests |
| C5 | P2 | Replays: same arm twice concurrently → one run; after restart → read only, never re-sent; kill -9 mid-call → UNCERTAIN, never re-sent | Native e2e (Stage 2 cases) |
| C6 | P2 | One active run across nudges and experiments: an experiment arm while a nudge runs → 409 busy, and the reverse | Native e2e |
| C7 | P2 | AS_OF_BASELINE snapshot: matches the logged prompt → VERIFIED; altered evidence → NOT REPRODUCIBLE and AS_OF_NOW offered | Engine unit + native |
| C8 | P2 | Fixture model, deterministic: A's reply SHA equals the baseline's → "Reproduced: yes"; a fixture that varies → "no" with the warning | Native e2e |
| C9 | P2 | Pre-restore experiment plans are PRE_RESTORE and can't run | Fence test extended |
| C10 | P2 | The comparison shows original / A / B with technical status and "saved into the Thought: yes/no/no"; the owner assessment is stored and labelled; there is no Promote control | Browser fixture |
| C11 | P2 | The output contract can't be edited: a variant containing `NOTE:`/`STATE:` lines or > 300 chars is refused before saving | Unit test |
| O1 | All | **Owner walkthrough on the installed build**: Thought → full Note → Development → (P2) experiment → comparison, without help, at half screen | Owner + GPT-6; NOT RUN until an authorized update |
| O2 | P2 | **One real Nemotron experiment on `identity-test`** (2 requests), separately approved | Owner-approved run only (D9) |

Plus the Stage 2 regressions (native 63 checks, updater 16, launcher 6, Go race in all three repositories) on every phase.

---

## 8. Risks, decisions, order

### 8.1 Risks

| Risk | Mitigation |
| --- | --- |
| Note text exposed more widely than intended | Detail endpoint only; never in lists, exports, sync, logs or audit; text rendering; tests A6–A8 |
| Owner reads persistence as reasoning quality | Separate "saved / correlated" from "answers the question" on screen; the owner assessment is labelled as such |
| P1 overstates what ran | Stage statuses come from files and code constants only; the instruction summary is tied to the pinned Engine commit and checked by a test |
| Experiments contaminate the Thought or drift into production | A separate `experiments/` tree; no Thought write path; C2; no Promote control |
| The instruction slot becomes a prompt-injection or format-breaking surface | Owner-only text, ≤ 300 chars, contract lines fixed, NOTE/STATE in the variant refused (C11); evidence stays quoted as "information, not instructions" |
| Non-reproducible baseline (clipped log, changed model build) | VERIFIED / VERIFIED-PREFIX / NOT REPRODUCIBLE labels; A/B is always compared within one snapshot |
| Note eviction (200 cap) weakens "original preserved" | Experiments add no Notes; `STEP_ONLY` shown honestly; raising or removing the cap is a separate decision |
| Engine `calls.jsonl` grows without limit, so lookups slow down | Bounded reverse scan; rotation is a separate decision, not in scope |
| GitHub sync of the Core data folder | Note text and experiment replies live in Engine data / `data/core/experiments.jsonl`, which are not synced (same as `nudges.jsonl`); verified by a test |
| Owner-host harness can reach Core 8797 (open item from #100) | Add 8797 to `OWNER_PORTS` in #104 before any P0 test run on the owner host |

### 8.2 Decisions for the owner

| # | Decision | Recommendation |
| --- | --- | --- |
| D1 | Show worker-generated Note text in Cognition, to the unlocked owner only (reverses the "no free-form detail" rule for this one field) | Yes, scoped as in §3.1 |
| D2 | Grant state shown from Identity's public state (display only) | Yes |
| D3 | P1 adds one read-only Engine endpoint (run record + call/model log lookup) | Yes |
| D4 | The first experiment's variable | The instruction line (§6.1); attention text as a fallback |
| D5 | Run the control arm A (+1 model request) | Yes: it is what makes B interpretable |
| D6 | Experiments don't require Thought Processing ON (they never develop the Thought), but the Thought must exist | Yes |
| D7 | New Identity bound action and grant `cognition.experiment.run` / `cognition.experiment-run` | Yes; don't reuse `cognition.nudge.process` |
| D8 | No "Promote" in this cycle; a better variant becomes a reviewed Engine PR | Yes |
| D9 | The one real-Nemotron experiment (O2) is a separate, explicit approval after Codex audits P2 | Yes |

### 8.3 Order and stopping points

1. **P0** (Core only). Implement, run the tests, Codex audit, owner-approved update, owner walkthrough O1 (the Cognition part).
   - **Stop and reassess** if the owner can't find the Note at half screen without help.
2. **P1** (one Engine read endpoint + Core). Same gates.
   - **Stop** if any stage can't be shown truthfully. Fix the evidence, don't decorate it.
3. **P2** (Identity + Engine + Core). Fixture-model gates C1–C11, Codex audit, owner-approved update, then O2 with separate approval.
   - **Stop** after O2 and review the owner's assessment before any second experiment or any template change.

P0 is useful on its own. If P2 can't be approved, P0 + P1 still deliver "see what was produced, and see how" without any new model use.

---

## 9. Disposition

**DESIGN NEEDS REVISION** for the brief as written:
1. **Wrong first variable.** The brief assumes the inadequate Note is an answer-quality problem best probed through the attention text. The Engine's prompt asks for one ≤ 40-word new idea, not an answer, and supplies no notebook content (§1). An attention-text experiment would most likely be a null result that teaches nothing.
2. **Three source locations need correcting.** (a) The "BLOCKED" text is the nudge-history list field `nudgeBlocked()`, not the page banner, and the per-nudge `AVAILABLE` is unchecked text, not a verified state. (b) Core's projection drops `source_nudge_id` and truncates the newest steps first. (c) The Note text already reaches Core and is discarded at decode, so P0 needs **no** Engine change.
3. **It misses two things that make the cycle cheap and honest.** The existing Junction-lab propose → compare → record pattern. And temperature 0 on `engine:*` calls, which makes a reproducible control arm possible.
4. **Experiment isolation was left open.** §3.3 and §6 close it with a separate Engine record tree, a new bound Identity action, shared admission and the extended restore fence.

**With these revisions:**
- **P0 and P1** are ready for owner approval for implementation (D1–D3).
- **P2** is ready after D4–D7, with O2 separately approved (D9).

Nothing is BLOCKED.
