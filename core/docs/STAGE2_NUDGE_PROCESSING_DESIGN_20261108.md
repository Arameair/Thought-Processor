# Stage 2 design: one supervised nudge → one verified Thought development

**Revision 3 (2026-11-08).** Answers GPT-6's review of revision 2 (PR #101 comment 6726234836: P0-1, P0-2, P1-3–P1-7). Revision 2 (`ac83ead`) answered the Codex audit (PR #102, A-01–A-12, X1–X14); revision 1 was `033b797`.
**Status:** DESIGN FOR RE-REVIEW. Not approved, not implemented, nothing deployed, no grant changed, no nudge processed.
**Baseline (read, not changed):** Core `094f43ba2bc167ace8835bf286f37951fee432c9`, Thought Engine `d69d3917d1601d806d4b2ba2254693e1adb92b6b`, Identity `64fdf8e8cf926df5ea6bb6c23fbab864d0f09e3b`.

**Goal (unchanged):** the owner approves one unexpired DRAFT nudge addressed to `self` on one real persistent Thought, and gets **one** development of that Thought from the local model, with a truthful receipt. Nothing else: no routing to positions 1–5, no autonomy, no queue or scheduler, no new service or model.

---

## 0. Revision 3: resolution of GPT-6's review of revision 2

| ID | Counterexample / finding | Resolution (revision 3) | Testable guarantee | Where |
|---|---|---|---|---|
| **P0-1** | The Engine redeems at t=119 s, stalls, and claims after Core has shown NOT_DELIVERED; Copy as new then gives two developments | **Time is no longer the fence.** NOT_DELIVERED and Copy as new now require a **void tombstone**: Core asks the Engine to create the nudge's `claim.json` as `{"void":true}` with the same `O_CREAT\|O_EXCL`. Whichever create wins, the handler's claim or the void, is final, and the loser gets `EEXIST`. A handler that resumes after a successful void cannot claim (EEXIST), so it makes no HTTP request. If the void loses, the run is shown as whatever the handler recorded. The 120 s decision TTL stays only to reject stale requests early; it is not a safety argument. | **G1:** for a nudge ID, at most one of {handler claim, void} exists; NOT_DELIVERED is shown **only** after a successful void; after a successful void the model gets **0** requests for that nudge, however long any handler was paused (X9b: pause after E2, before E5). | §4.5, §6, X9 |
| **P0-2** | Restoring a pre-processing backup erases every record, and the same nudge ID can be processed again | **Restore fence written by the restore itself.** `tass.sh restore_data()` is the single function every supported restore goes through (`restore-previous`, `rollback --restore-data`, `recover`, update auto-restore). Right after unpacking, it writes `data/core/restore-fence.json` `{"restored_at", "backup", "draft_lines": N}`, where N is the line count of the restored `nudges.jsonl`. This file **is not in the backup being restored**; it is created after the restore. Core treats every draft on lines 1..N as **PRE_RESTORE**: viewable, never processable. Core refuses to obtain a bound decision for it, so the Engine can't be asked either. The owner can only **Copy as new** (a new ID, a new explicit approval, and a dialog saying "this nudge may already have been processed before the restore"). | **G2:** after any `tass.sh` restore, no nudge ID that existed in the restored data can be processed again, through Core or directly at the Engine (the Engine needs a bound decision, which Core won't request). X11b: whole-app pre-claim restore with the draft surviving → POST process → 409 PRE_RESTORE; 0 HTTP. | §4.6, X11 |
| P0-2 (scope) | Arbitrary restores | Restores **not** made with `tass.sh` (a manual untar, a VM snapshot, file copies) bypass the fence. **Stated limitation**, not covered. | stated in §9 R4 | §4.6, §9 |
| **P1-3** | Core replays a run before the fresh grant check | Core order is now: **C2 fresh `cognition.nudge.process` (unbound) → C3 existing run → C4 bound decision → C5 submit.** A POST after revocation is 403 even when a run exists. GET history stays read-authorized. | revoked Core POST → 403 (S16) | §3 |
| **P1-4** | Core SUBMITTED append errors and restarts | Core uses the **same immutable per-nudge files** as the Engine: `<core data>/nudge-runs/<id>/submitted.json` (`O_EXCL`, fsync, dir fsync). If the create fails, nothing was written and nothing is sent. Once the file exists the nudge is "submitted" forever: unparsable means **RECOVERY_REQUIRED** (persistent, survives restart). Only the process that created the file may send, so a restart never sends. "One active run" is derived from these files plus the Engine state, so it holds across restarts. The shared `nudge-runs.jsonl` is dropped. | X15: short write / fsync / dir-fsync / crash at each boundary → never two sends, never a silent re-submit | §3 C5, §4.2 |
| **P1-5** | Privacy claim too absolute | Reworded: the raw attention is not **duplicated into prompt logs** (digest only, D10). The model's **reply may echo it** and is stored in `model-requests.jsonl`, `calls.jsonl` and the Thought note: local, 0600, sensitive. X12 now checks the logging boundaries, not "the marker appears nowhere". | X12 reworded | §5 |
| **P1-6** | Deleting the lock file is not a repair | **D9 = BLOCKED.** The only supported way to clear a ledger lock in v1 is a `tass.sh` restore of a consistent backup, which also applies the restore fence (G2). No manual deletion and no unlock button. A read-only integrity scan (`tass-core -nudge-audit`, prints only) is proposed to diagnose. | ledger lock persists until a `tass.sh` restore (X11c) | §4.6, §11 |
| **P1-7** | Service identities; scope | Confirmed in pinned Identity: `service()` maps each key in the service-keys file to its name (`core`, `engine`), and handlers can already restrict by name (`/v1/github/lesson` allows only `svc == "core"`). Mint = `svc == "core"` only; redeem = `svc == "engine"` only; both also need the current entity to be UNLOCKED with the grant. A redeemed decision is used up: a **new** claim always needs a new decision, and a replay status response needs its own fresh decision (E2 runs before E3). **Scope:** the guarantees cover the nudge route only, not the plain `think` route (R1/D11). | X1, X2, S1 | §2, §3, §9 |

## 0b. Revision 2 resolution of Codex #102 (kept for traceability)

| ID | Finding (short) | Resolution | Where |
|---|---|---|---|
| A-01 | The Engine can't tell that the owner approved *this* nudge | **Bound, single-use Identity decision.** On the owner's click, Core asks Identity for `cognition.nudge.process` **bound to a digest of the exact tuple** (owner, Engine entity, Thought, nudge ID, attention digest, recipient, draft expiry), with a short TTL. Only Core's service key may obtain one, and only the Engine's may redeem it. The Engine redeems it before it claims. The shared Engine key alone is no longer enough for the nudge route. | §2.1, §3 |
| A-02 | A replay is answered before authorization | The Engine **redeems a fresh bound decision before any successful POST response, a replay included.** Read-only status is on the separately keyed GET. | §3 E2–E3 |
| A-03 | "fsync failed ⇒ no claim" is false | **The claim is an immutable per-nudge file created with `O_CREAT|O_EXCL`**, then file fsync, close and parent-directory fsync. If the create fails there is no claim; once the file exists, the nudge is claimed **forever**. Any later error means `RECOVERY_REQUIRED` for that nudge, never "unused". No appends, no rotation. | §4.1–4.2 |
| A-04 | The PID lock fails across PID namespaces | (a) The per-nudge `O_EXCL` claim is atomic across processes on one local filesystem, whatever the PIDs. (b) The Engine's data-folder lock becomes an **`flock(LOCK_EX|LOCK_NB)` held for the process lifetime** (Unix), replacing the PID-equality check. Two-PID-namespace test X4. | §4.1, §4.7 |
| A-05 | Revocation and state are checked at the wrong times | **Two point-of-use checks in the Engine:** **P1** after taking the model slot, immediately before the HTTP request; **P2** immediately before saving the Note. Each checks: current `cognition.nudge.process` grant, Thought Processing ON, Thought exists in the same entity and is unresolved. A save that already happened is not undone by a later revocation. | §3 E5, E7 |
| A-06 | A request ID or `model-requests.jsonl` line doesn't prove a model call | **Phase files** record each step: `claim` (intent) → `http` (written and synced *before* the HTTP POST) → `response` → `outcome`. `model_http` is reported as `none` / `attempted` / `responded`, and a crash after `http` is **UNCERTAIN**. Tests count POSTs **at the fixture HTTP server**. | §4.2–4.4, §7 |
| A-07 | False COMPLETED; restore erases the fence | **COMPLETED needs:** a `response` that says OK, plus an `outcome` (or a recovered one), plus **exactly one** Note and **exactly one** `developed` step with this nudge ID and request ID. Anything conflicting is `RECOVERY_REQUIRED`. Thought saves also fsync the parent directory, and a save error is re-checked by re-reading the file. **Restore:** the guarantee is stated for one continuous ledger history. Whole-app restores (the only kind `tass.sh` makes) rewind the claim **and** the Thought together. Divergence is detected and **locks processing**. I argue against a ledger that survives restores (§4.6). | §4.4–4.6 |
| A-08 | The attention text is in `calls.jsonl`, not `model-requests.jsonl` | Corrected data map. Proposed: the nudge path writes the attention to `calls.jsonl` **as its digest only** (owner decision D10). The model **reply** is still in `model-requests.jsonl` and `calls.jsonl`, as for any Think. All of this is local, 0600, and inside the whole-app backups. | §5 |
| A-09 | Background dispatch not specified | **One tracked dispatcher** with a Core-lifetime context, at most one SUBMITTED/RUNNING run at a time, cancelled on shutdown (the Engine keeps going; Core reconciles by reading). `202` means **"accepted locally; delivery not confirmed"**. Startup and GET never POST. | §3, §6 |
| A-10 | GET 404 isn't terminal; copy-as-new races a delayed POST | **(Superseded in revision 3 by the void tombstone, §4.5; the time window below was unsound, GPT-6 P0-1.)** The Engine enforces a delivery fence: it claims only with an unexpired bound decision (TTL 120 s). After TTL + 60 s with no claim, Core may show **NOT_DELIVERED** (terminal). Before that, a 404 is "not claimed yet", and **Copy as new is disabled** while a run is SUBMITTED/RUNNING. | §4.5 |
| A-11 | Relinking mid-run undefined | **Core refuses link/unlink while a run is SUBMITTED or RUNNING.** A finished run stays bound to the Engine entity it ran on. | §3 C1, §6 |
| A-12 | Disk-full / read-only / cap / missing ledger | **Fail closed before any HTTP.** Bounded number of run folders (refuse past 1,000), no rotation or truncation. A missing or inconsistent ledger locks processing (§4.6). Status stays readable. | §4.6, §4.7 |
| GPT-6 P0-B | fsync ambiguity | Same as A-03. | §4.2 |
| GPT-6 P0-C | Recheck before the save | Same as A-05 (P2). | §3 E7 |
| GPT-6 P1-G | Record revisions | The Thought revision is recorded at confirmation, at claim, at P1 and at P2. The confirm dialog shows a change since the draft; it is not refused for that. | §3, §6 |

**Requirements I challenge (§8):** a ledger that survives restores; a separate Engine credential for Core; a durable single-use flag inside Identity; a global "any uncertainty locks everything" rule.

---

## 1. What the pinned code gives and lacks (unchanged facts, corrected)

- `Engine.Think` (`engine.go` L1331–1419) is synchronous, holds `workMu` for the whole call, makes one worker call, and saves one Note plus one `developed` step in one atomic file write (`thoughts/<tid>.json`). It never applies the suggested state. After the call it checks only that the entity and Thought still exist.
- `ModelLink.run` (`model.go` L333–390) authorizes the action it is given (Think passes `thought.process`, a **default** grant) before and after taking the model slot, then `ask` makes the HTTP POST. `record` appends to `model-requests.jsonl` **for denials too**, after the fact, without fsync, rotating at 5 MB.
- The worker writes the **prompt** (trimmed to 2,000 chars) and the reply to `calls.jsonl` (`worker.go` L385–395, L456–503).
- `writeJSONAtomic` (`store.go`) fsyncs the temp file, renames it, and reads it back. It does **not** fsync the parent directory and can return an error after the rename.
- `lock.go` treats the lock as stale or own by PID equality and liveness, which is unsafe across PID namespaces.
- Identity `Authorize` checks only entity, state and grant membership; `detail` is audit text, not a constraint (`entity.go` L355–377). It knows which service is asking (`core` or `engine`, from the service key).
- Core does **not** use the Engine's plain `think` route. That route stays reachable to any holder of the Engine key (residual risk R1, §9).

## 2. Changes (smallest set that meets the audit)

| Repo | Change |
|---|---|
| **Identity** | (1) Allowlisted owner grant `cognition.nudge-process` → `cognition.nudge.process` (same pattern as #42). (2) **Bound decisions** for that action only: `POST /v1/authorize` accepts `binding` (sha256 hex). For `cognition.nudge.process` a binding is **required**, only the `core` service may ask, and an allowed decision is kept **in memory** with `expires_at = now + 120 s`. (3) `POST /v1/decisions/redeem` (only the `engine` service): `{decision_id, action, binding}` → allowed only if the decision exists, is unexpired and unused, has the same action and binding, the entity is unchanged, the entity is UNLOCKED, **and the grant is still held now**. It is then marked used. Both steps are audited in `decisions.jsonl`. An Identity restart forgets every bound decision, which fails closed. |
| **Engine** | (1) `POST /api/entities/{eid}/thoughts/{tid}/nudge` + `POST /api/entities/{eid}/nudges/{nid}/void` + `GET /api/entities/{eid}/nudges/{nid}`. (2) Per-nudge run folder `<entity dir>/nudge-runs/<nudge_id>/` holding immutable phase files (§4). (3) A Think variant: attention block in the prompt, pre-generated request ID, P1 hook after the model slot, P2 check before the save, `source_nudge_id` on Note and step. (4) `writeJSONAtomic` also fsyncs the parent directory. (5) `flock` data-folder lock on Unix. (6) A startup ledger scan (§4.6). Plain `think` is unchanged. |
| **Core** | `POST /api/nudges/{id}/process` and `/void`; per-nudge receipt folder `<core data>/nudge-runs/<id>/` (`submitted.json`, `outcome.json`, `O_EXCL`); one tracked dispatcher; read-only reconciliation; verification; link/unlink refused while a run is active; ledger lock; restore fence (PRE_RESTORE); UI (confirm dialog, receipt, Copy as new); pin bump. |
| **tass.sh** | In `restore_data()`, after a successful unpack: write `data/core/restore-fence.json` (`draft_lines` = lines of the restored `nudges.jsonl`, 0600). About 5 lines; nothing else changes. |

### 2.1 The binding digest

```
binding = sha256( canonical JSON of
  {"action":"cognition.nudge.process","owner_entity_id":…,"engine_entity_id":…,"thought_id":…,
   "nudge_id":…,"attention_sha256":…,"recipient":"self","draft_expires_at":…} )
```
Core computes it from the immutable draft. The Engine recomputes it from the request body and its own path values. Identity compares the two. Any substitution (another nudge, Thought, entity, text or expiry) changes the digest, and the redeem is refused (X1).

## 3. Authorization and dispatch flow

```
Owner: "Process once…" on a DRAFT (self) → dialog: nudge ID, Thought, attention, expiry,
       "Thought changed since the draft: yes/no", "This makes one model request. Not undoable after it is saved."
  │ POST /api/nudges/{id}/process {body_digest}   (page token, local Origin, JSON)
  ▼
C1  draft exists; owner = current Identity entity; Engine link = draft's; recipient self; unexpired;
    body_digest = stored; NOT PRE_RESTORE (§4.6); forced TASS_CHAT_ONLY off; ledger not locked  → 4xx, nothing written
C2  Identity /v1/authorize {cognition.nudge.process} (unbound; current grant)  → denied: 403, even if a run exists
C3  run folder for this nudge exists → 200 with its state (read-derived; never sent again)
C4  no other run SUBMITTED/RUNNING (derived from files + Engine reads)          → 409
    Identity /v1/authorize {cognition.nudge.process, binding}  (bound, TTL 120 s) → denied: 403
C5  create nudge-runs/<id>/submitted.json (O_EXCL) {decision_id, binding, revision, engine entity, thought};
    fsync file; close; fsync dir
      create fails → 500 "nothing was sent"
      later error  → the file exists → RECOVERY_REQUIRED for this nudge; nothing sent
    reply 202 "accepted locally; delivery not confirmed"; dispatcher (this process only) sends one POST
  ▼
E1  key, local, JSON; entity exists; Thought belongs to it; attention ≤500 and sha256 matches → 4xx, no claim
E2  Identity /v1/decisions/redeem {decision_id, action, binding (recomputed)}  → refused: 403, no claim, no call
E3  run folder exists? → return its state (an authorized replay, or a VOID; no call)
E4  Thought Processing ON; Thought unresolved; run-folder count < 1000; ledger not locked → 4xx, no claim
E5  CLAIM: create claim.json (O_EXCL) {request_id pre-generated, revision, decision ids}; fsync file; close; fsync dir
      EEXIST           → someone else (another handler, or a void) won: return that state, no call
      create fails     → 503, no claim, no call
      any later error  → the nudge stays claimed; RECOVERY_REQUIRED; no call
E6  ModelLink.run(thought.process) [existing: authorize, take slot, authorize again]
    P1 hook (after slot, before HTTP): current cognition.nudge.process + Thought Processing ON + Thought unresolved
      fails → outcome FAILED(p1_<reason>), model_http none
    create http.json (O_EXCL, fsync, dir fsync) → fails: outcome FAILED(not_sent), model_http none
    HTTP POST to the model (existing ask)
    write response.json {ok|error, reply sha256, elapsed} (best effort; its absence after a crash = UNCERTAIN)
E7  parse NOTE/STATE (existing parseThink) → unreadable: outcome FAILED(unreadable_reply), responded
    P2 (before save): current grant + Thought Processing ON + Thought still in entity + unresolved
      fails → outcome FAILED(p2_<reason>), responded, note NOT saved
    save Thought (Note + developed step, source_nudge_id + request_id), with parent-dir fsync
      error → re-read the Thought file: contains the step? → RECOVERY_REQUIRED (inconsistent)
                                                       no  → outcome FAILED(save_failed)
E8  create outcome.json (O_EXCL, fsync) → return the run
  ▼
C6  dispatcher creates Core outcome.json (O_EXCL, fsync) and verifies (§4.4) before showing COMPLETED
```

**CHAT_ONLY:** allowed with the owner's Autonomous switch OFF (like supervised skills); refused when `TASS_CHAT_ONLY=on` is forced (owner decision D2).
**Revocation:** before C3/E2 → refused. Before P1 → no HTTP. Between P1 and P2 (the model is answering) → the call happened, the note is **discarded** (D1). After the save → the result stands and is not retroactively undone. That is the point-of-use boundary.

## 4. Run-once: storage, states, recovery

### 4.1 The claim primitive and the filesystem assumption

`open(<run>/claim.json, O_CREAT|O_EXCL|O_WRONLY, 0600)` is atomic across processes and PID namespaces on one **local POSIX filesystem** (ext4/xfs, including a Docker bind mount or volume on one host). **Not supported** for the data folder: NFS/SMB, or Windows/WSL `drvfs` (documented; `tass.sh check` should warn). Each phase file is created once with `O_EXCL` and never modified, so there are no appends, torn tails or rotation.

### 4.2 Phase files in `<entity dir>/nudge-runs/<nudge_id>/` (0700 folder, 0600 files)

| File | Written | Contains (no attention text) |
|---|---|---|
| `claim.json` | E5, before anything else | nudge ID, Thought, attention sha256 + length, **request ID (pre-generated)**, revision at claim, Identity decision IDs (authorize + redeem), time |
| `http.json` | E6, **before** the HTTP POST, synced | time, request ID |
| `response.json` | E6, after the HTTP response | ok/error, reply sha256, elapsed ms |
| `outcome.json` | E7/E8 | `COMPLETED` / `FAILED(reason)` / `RECOVERY_REQUIRED(reason)`, `model_http`, step time, suggested state, `applied:false`, revisions at P1 and P2 |

A **void** is `claim.json` = `{"void":true, "at", "by":"core"}` created by the void route with the same `O_EXCL` (§4.5). A voided nudge has no other files.

**Core mirror** in `<core data>/nudge-runs/<id>/`: `submitted.json` (C5), `void.json` (after a successful Engine void), `outcome.json` (C6). The same rules apply: immutable, `O_EXCL`, file + dir fsync; a file that exists but doesn't parse = `RECOVERY_REQUIRED` for that nudge, which persists across restarts. No Core process ever sends for a `submitted.json` it did not create in its own lifetime.

### 4.3 Engine state, derived only by reading (never by acting)

| Files present | Thought evidence | State | `model_http` |
|---|---|---|---|
| none | — | `NOT_CLAIMED` (404) | — |
| `claim.json` with `void:true` | — | **`VOID`** (terminal; never runs) | none |
| claim, still running in this process | — | `RUNNING` | per files |
| `outcome` | consistent with outcome (§4.4) | that outcome | per files |
| `outcome` | conflicting | `RECOVERY_REQUIRED (inconsistent)` | per files |
| claim only (after a restart) | none | `FAILED(interrupted_before_http)` | **none** |
| claim + `http`, no `response` (after a restart) | none | **`UNCERTAIN`** | **attempted** |
| claim + `http` + `response ok`, no `outcome` | exactly 1 Note + 1 step | `COMPLETED (recovered)`; `outcome.json` written by the startup scan | responded |
| same | none | `FAILED(interrupted_after_response)` | responded |
| any unparsable or empty phase file, or other mixes | — | `RECOVERY_REQUIRED` | unknown |

The startup scan writes only `outcome.json` for the recovered-COMPLETED case (and only if the Thought evidence is exact). It never calls the model.

### 4.4 COMPLETED rule (falsification-resistant, X14)

**COMPLETED** iff `response.ok` **and** (`outcome = COMPLETED`, or the recovered case) **and** the Thought has **exactly one** Note **and exactly one** `developed` step whose `source_nudge_id` = this nudge and whose `worker_request_id` = the claim's request ID **and** `applied:false`. Core re-verifies through the Cognition Thought read before showing it.

| outcome ↓ / matching (Notes, steps) → | (0,0) | (1,1) | anything else |
|---|---|---|---|
| COMPLETED | RECOVERY_REQUIRED | **COMPLETED** | RECOVERY_REQUIRED |
| FAILED | FAILED | RECOVERY_REQUIRED | RECOVERY_REQUIRED |
| missing | per §4.3 | COMPLETED (recovered) only with `response.ok` | RECOVERY_REQUIRED |

### 4.5 Void tombstone and Copy as new (A-10, P0-1)

Revision 2 used a time window (TTL + 60 s). That is unsound: a handler can redeem in time, pause, and claim later. **Revision 3 uses the claim primitive itself as the fence.**

- `POST …/nudges/{nid}/void` (Engine key; local; JSON `{reason}`) does exactly one thing: `mkdir` the run folder (ok if it exists) and **create `claim.json` = `{"void":true}` with `O_CREAT|O_EXCL`**, then fsync file and dir. It needs no Identity decision, because it can only **prevent** processing.
  - create succeeds → `VOID`. From now on every handler gets `EEXIST` at E5 and returns `VOID` with no call, including one that redeemed long ago and paused.
  - `EEXIST` → a handler (or an earlier void) won. Return the existing state; the run is **not** NOT_DELIVERED.
  - create fails for another reason → `503`; nothing changed; Core keeps showing SUBMITTED.
- **Core:** a 404 from the Engine is shown only as `SUBMITTED – not claimed yet`, never as terminal. To get a terminal answer the owner presses **"Stop this nudge"** (or **Copy as new**, which voids first). Core calls void and then creates its `void.json`:
  - void succeeded → **`NOT_DELIVERED`** (terminal, and true: the model can never be asked for this nudge);
  - void returned an existing claim → show that run; Copy as new waits until the run is terminal (COMPLETED / FAILED / UNCERTAIN / RECOVERY_REQUIRED). For UNCERTAIN the dialog warns that the first request may have reached the model.
- **No timers decide safety.** The 120 s bound-decision TTL remains only so stale requests fail early at E2.

### 4.6 Restores (P0-2) and ledger integrity (A-07, A-12)

**Counterexample accepted:** backup B holds draft `nudge_A`; `nudge_A` is processed; B is restored. Core, Engine and the Thought rewind together, and without a fence `nudge_A` looks unprocessed. The first HTTP request to the model really happened, so processing it again would be a second call for the same ID.

**Fence (G2):** `tass.sh restore_data()` is the one function every supported restore uses (`restore-previous`, `rollback --restore-data`, `recover`, and auto-restore after a failed update; `tass.sh` L332–345 and its callers). After a successful unpack it writes:

```json
data/core/restore-fence.json   (0600, created after the restore, so never part of the restored backup)
{"schema":"tass.restore-fence/1","restored_at":"…","backup":"<file name>","draft_lines":N}
```
where N = number of lines in the restored `data/core/nudges.jsonl` (0 if absent). If a fence already exists (an earlier restore), N becomes the new line count, so the earlier drafts stay covered.

Core, on every read and process request: a draft on **line ≤ N** is **`PRE_RESTORE`**. It is viewable and copyable, and **never processable**: C1 refuses it (409). Because Core never requests a bound decision for it, the Engine route can't process it either (E2). The owner can **Copy as new**: a new nudge ID, with a dialog stating that "the original may already have been processed before the restore" and requiring a new approval.

**Why not lock all processing after a restore (GPT-6's alternative)?** It adds an unlock step and stops new, unrelated nudges. PRE_RESTORE blocks exactly the IDs whose history may have been erased; new drafts are safe because their IDs never existed in the erased future.

**Restores that bypass `tass.sh`** (manual untar, VM or disk snapshot, copying folders) don't write the fence, so they are **not covered** (R4).

**Ledger integrity (unchanged from revision 2, repair tightened):**
- Engine startup scan: a Thought step or Note with `source_nudge_id` but no run folder, or an unparsable phase file → `ledger_locked`.
- Core: a run with `claim_observed` that the Engine now reports as NOT_CLAIMED → `processing_locked`.
- **Repair (D9 = BLOCKED for anything else):** only a `tass.sh` restore of a consistent backup, which also writes the fence. Deleting lock markers by hand is unsupported. A read-only `tass-core -nudge-audit` prints the scan for diagnosis and changes nothing.

### 4.7 Locks and limits

- Engine data-folder lock: `flock` on `<data>/.engine.lock`, held for the process lifetime and released by the kernel when the process dies. No PID logic on Unix (Windows keeps its current lock). A second Engine on the same folder exits at start. Even if it didn't, `O_EXCL` keeps every claim single.
- Limits: at most 1,000 run folders (then new claims are refused), attention ≤500 chars, one SUBMITTED/RUNNING run in Core. Disk full or read-only: create or fsync fails before `http.json`, so there is no HTTP; status stays readable.

## 5. Data map and privacy (A-08)

| Where | What of the nudge | Notes |
|---|---|---|
| Core `nudges.jsonl` | full draft incl. attention text | as today; 0600 |
| Core `nudge-runs.jsonl` | IDs, digests, decision IDs, states | no attention text |
| Engine run folder | IDs, digests, request ID, states | no attention text |
| Engine `calls.jsonl` | prompt (2,000 chars) + reply | **D10 proposal:** for nudges, the attention is replaced by `[owner attention sha256:…]` in the stored prompt. The model still receives the real text. |
| Engine `model-requests.jsonl` | reply text + reasoning, request ID | as for any Think |
| Engine Thought file | the model's NOTE (≤400 chars), IDs | the visible result |
| Identity `decisions.jsonl` | action, `detail` (nudge ID + Thought ID), binding digest | no attention text |
| whole-app backups | all of the above | local, 0600 (`umask 077`) |
| GitHub, `tass.md`, Core event log | **nothing** of the nudge content | checked by X12 |

**What this does and doesn't promise (P1-5):** the raw attention text is **not duplicated into prompt logs** (D10) and never leaves the computer. The model's **reply may quote or paraphrase it**, and the reply is stored in `model-requests.jsonl`, `calls.jsonl` and the Thought note. Treat all three as sensitive local data (0600, in backups). Redacting model output is out of scope.

## 6. API and UI contracts

**Engine**
- `POST …/thoughts/{tid}/nudge` `{nudge_id, owner_entity_id, attention, attention_sha256, draft_expires_at, decision_id}` → `200 {run}` after the outcome, or immediately for an authorized replay. `400` invalid · `403` redeem refused / processing off / ledger locked · `404` · `503` claim not created (no claim exists).
- `POST …/nudges/{nid}/void` (Engine key; local; JSON) → `200 {run: VOID}` | `200 {run: <existing>}` (`EEXIST`) | `503`. Prevents only; needs no Identity decision.
- `GET …/nudges/{nid}` (existing `sensitive()` key check) → `{run}` | `404 NOT_CLAIMED`. Read-only, no side effects.

`run = {nudge_id, thought_id, state, reason, model_http: none|attempted|responded, request_id, revisions {claim,p1,p2}, step_at, suggested, applied:false, decisions {authorize, redeem}}`

**Core**
- `POST /api/nudges/{id}/process` `{body_digest}` → `202 "accepted locally; delivery not confirmed"` · `200` existing (only after a fresh grant check) · `400` · `403` · `409` (PRE_RESTORE / another run active / digest) · `423` processing locked · `500` SUBMITTED not created (nothing sent).
- `POST /api/nudges/{id}/void` (owner "Stop this nudge"; page token) → Engine void → `NOT_DELIVERED` or the existing run. No grant needed: it can only prevent.
- `GET /api/nudges` → each nudge + `run` (read-reconciled with at most one Engine GET per active run).
- `POST /api/cognition/link|unlink` → `409` while a run is SUBMITTED/RUNNING (A-11).

**Dispatcher (A-09):** one goroutine per accepted run, at most one active, tracked by Core's shutdown waitgroup; context = Core lifetime (not the HTTP request); Engine client timeout 10 min. On shutdown it cancels and does not wait for the Engine. After a restart, runs left SUBMITTED are **only read**, never re-sent.

**UI states (10; each is backed by evidence):** `DRAFT`, `PRE_RESTORE` (viewable, copy only), `EXPIRED`, `SUBMITTED` (accepted locally / not claimed yet: **never terminal**), `RUNNING`, `COMPLETED` (verified), `FAILED` (+ reason + model request none/attempted/responded), `UNCERTAIN` (a model request may have been sent), `NOT_DELIVERED` (**only** after a successful void), `RECOVERY_REQUIRED`.

## 7. Failure matrix (revised F1–F14 + new)

"HTTP" = POSTs received by the model server (the oracle).

| # | Break | Claim | HTTP | Shown | Retry |
|---|---|---|---|---|---|
| F1 | Core refuses (C1/C3) | — | 0 | 4xx | owner fixes and clicks again |
| F2 | `submitted.json` create fails / error after create | — | 0 | 500 nothing sent / RECOVERY_REQUIRED (persistent: the file exists) | owner clicks again / never; Stop or Copy only after a void |
| F3 | Core crashes after SUBMITTED, before/while sending | none or yes | 0/≤1 | SUBMITTED (not terminal) → RUNNING/outcome; NOT_DELIVERED only after the owner's void succeeds | never automatic |
| F4 | Engine refuses (E1–E4) | — | 0 | FAILED(reason) | owner fixes, new click = new bound decision |
| F5 | claim create fails / later claim write error | — / claimed | 0 | 503 / RECOVERY_REQUIRED | — / never |
| F6 | Engine crash after claim, before `http.json` | yes | 0 | FAILED(interrupted_before_http) | never; copy allowed |
| F6b | Engine crash after `http.json` | yes | 0 or 1 | **UNCERTAIN** | never; copy allowed with warning |
| F7 | model error / timeout / unreadable reply | yes | 1 | FAILED, responded or attempted | never |
| F8 | save error | yes | 1 | FAILED(save_failed), or RECOVERY_REQUIRED if the re-read finds the step | never |
| F9 | crash after save, before outcome | yes | 1 | COMPLETED (recovered) only with exact evidence | — |
| F10 | Core shutdown/crash during the Engine call | yes | 1 | read-reconciled outcome | never re-sent |
| F11 | concurrent Core POSTs | — | ≤1 | one run (C1 single-active + C2) | — |
| F12 | direct duplicate Engine POST | yes | ≤1 | replay only with a fresh redeemed decision; else 403 | — |
| F13 | grant revoked: before P1 / between P1 and P2 / after the save | yes | 0 / 1 / 1 | FAILED(p1) / FAILED(p2), note discarded / COMPLETED | — |
| F14 | relink during a run | — | — | relink refused (409) | — |
| F15 | resolve / Thought Processing off mid-call | yes | 1 | FAILED(p2_…), note not saved | — |
| F16 | Identity restart between C3 and E2 | — | 0 | FAILED(redeem refused) → owner clicks again | — |
| F17 | two Engines on one data folder | — | ≤1 | second exits at the lock; claims stay single anyway | — |
| F18 | divergent restore (Core-only / Engine-only) | — | 0 | processing locked | `tass.sh` restore only (D9) |
| F19 | whole-app `tass.sh` restore to before the claim | erased | 0 for the old ID | the old draft shows PRE_RESTORE; process → 409 | only Copy as new (new ID, new approval) |
| F20 | handler redeemed, paused, owner voids, handler resumes | void wins | 0 | NOT_DELIVERED; the handler returns VOID at E5 | — |
| F21 | handler claims first, owner voids | claim wins | ≤1 | void returns the run; it is shown, not NOT_DELIVERED | — |
| F22 | restore outside `tass.sh` | erased | **not covered** (R4) | — | — |

## 8. Requirements I challenge

1. **A claim ledger that survives restores** (Codex A-07 option). Still challenged, but revision 3 keeps its benefit at much lower cost: the restore itself writes a fence (§4.6). Old IDs are never processed again, and new IDs work. No storage outside the data folder, no new service, no unlock step.
2. **A separate Engine credential for Core** (A-01 option). The bound decision can only be obtained with Core's Identity service key, which already makes the nudge route Core-only. A second key adds key management without adding a boundary.
3. **A durable single-use flag in Identity.** The Engine claim is the durable fence. An in-memory bound decision that dies with an Identity restart fails closed, and replay safety comes from the claim. Making Identity durable adds a store and buys nothing more.
4. **"Any ambiguity locks all processing."** Per-nudge `RECOVERY_REQUIRED`/`UNCERTAIN` already blocks that nudge forever. Locking everything is kept only for **ledger** divergence (§4.6), where the fence itself is in doubt.
5. **Distinguishing `FAILED_NOT_SENT` after a crash.** That is now possible without guessing (claim present, `http.json` absent ⇒ no HTTP), so it is included (F6). Only after `http.json` is it UNCERTAIN.
6. **A time-based delivery fence** (revision 2's own). Withdrawn: replaced by the void tombstone (§4.5), because no timer can close the redeem→claim gap.
7. **Locking all processing after every restore** (GPT-6 P0-2 alternative). PRE_RESTORE blocks exactly the IDs whose history may have been erased and leaves new nudges working, with no unlock step (D13).

## 9. Residual risks (accepted, documented)

- **R1:** the Engine's plain `think` route still lets any holder of the Engine key make a model call and a note without nudge consent. Core doesn't use it. Gating it is a separate change (owner decision D11).
- **R2:** owner consent is asserted by Core (page token + dialog). Anyone holding Core's Identity service key could mint bound decisions. Same trust as every other Core action.
- **R3:** an UNCERTAIN run can't be resolved without server-side receipts from llama.cpp, which it doesn't provide.
- **R4:** restores **not** made with `tass.sh` (manual untar, VM/disk snapshots, copied folders) bypass the restore fence, so an old nudge ID could be processed again after one of those. Supported restores are covered (G2).
- **R5 (scope):** G1 and G2 apply to the **nudge route only**. The plain `think` route is a separate, unchanged path (R1, D11).

## 10. Acceptance tests

### 10.1 Revised S1–S15 (the oracle: fixture model POST count + files + Thought)

| ID | Test | PASS only if |
|---|---|---|
| S1 | Identity entry + bound decisions | not default; implied by no other grant; enable needs passphrase; revoke immediate; binding required; only `core` may obtain one; only `engine` may redeem; expired, used, mismatched or other-entity decisions refused; restart forgets all; absent from skill discovery |
| S2 | denials (LOCKED, no grant, draft-grant only, positions 1–5, expired draft, digest mismatch, relinked, other entity's Thought, resolved, Thought Processing off, forced chat-only, ledger locked, bad token/Origin/JSON) | right 4xx; **0 HTTP**; no claim; no Core SUBMITTED for Core-side denials |
| S3 | happy path | 202 → COMPLETED verified; **1 HTTP**; 1 Note + 1 step with the same IDs; `applied:false`; no cycle or job; all 4 phase files; same after restarting both |
| S4 | 10 concurrent Core POSTs; POST after COMPLETED | **1 HTTP total**; one run |
| S5 | direct Engine replay: with a fresh bound decision / without one / after revoke / with a substituted tuple | existing run / 403 / 403 / 403; 0 extra HTTP |
| S6 | Core killed before send; a delayed send | stays SUBMITTED (not terminal); owner Stop → void → NOT_DELIVERED; a late POST then gets VOID at E5; 0 HTTP |
| S7 | Core killed during the Engine call | read-reconciled outcome; 1 HTTP; no re-send on restart or GET |
| S8 | Engine killed before `http.json` / after `http.json` (slow fixture) | FAILED(interrupted_before_http), 0 HTTP / UNCERTAIN, ≤1 HTTP; nothing re-sent |
| S9 | crash after save before outcome; save error with the step present | COMPLETED (recovered) / RECOVERY_REQUIRED |
| S10 | claim create fails / write, fsync or dir-fsync error after create | 503, no folder / RECOVERY_REQUIRED; **0 HTTP** in both; nudge never claimable again in the second case |
| S11 | unreadable or repeated reply | FAILED(unreadable_reply), 1 HTTP, no note |
| S12 | revoke before P1 / between P1 and P2 | 0 HTTP / 1 HTTP, note discarded |
| S13 | heartbeat ON; owner resolves / switches off / revokes before P1 and before P2 | nudge steps found by `source_nudge_id`; each mid-call change gives its FAILED(p1/p2) with no note |
| S14 | regressions (draft-only, plain `think`, mailbox, pulse, lessons, Cognition, browser, updater 14/14, vet/test/race ×3) + X4 | unchanged |
| S15 | privacy (see X12) | the attention text appears only where §5 says |
| S16 | revoke the grant, then POST process for a nudge that already has a run (P1-3) | 403; the existing run is still shown by GET; 0 HTTP |
| S17 | Stop/void: before any claim / after a claim / twice | NOT_DELIVERED / the existing run / the same answer; 0 extra HTTP |
| O1 | owner, real Nemotron (after re-review, CI, owner approval) | preflight (exact Thought unresolved, Thought Processing ON, heartbeat/pulse state, model ready, counts) → grant → one run → COMPLETED verified; `model-requests.jsonl` +1 for this request ID; survives restart; revoke after |

### 10.2 Design-verification matrix X1–X14 (from Codex #102; all **OPEN** until implemented and run)

| ID | Covered by | PASS only if |
|---|---|---|
| X1 substitution | §2.1, E2 | every substitution refused before the claim; 0 HTTP |
| X2 revoked replay | E2 before E3 | POST 403; GET still returns status; 0 HTTP |
| X3 torn claim | §4.1–4.2, E5 | after each injected error + restart: RECOVERY_REQUIRED; never re-claimed; ≤1 HTTP (0 for claim errors) |
| X4 two PID namespaces | §4.1, §4.7 | two Engines, each PID 1, one data folder (`unshare -p`), the same nudge raced 100×: one claim folder, ≤1 HTTP; with `flock` the second Engine doesn't start |
| X5 slot revocation | P1 | 0 HTTP; FAILED(p1_grant) |
| X6 pre-save matrix | P2, A-11 | every disallowed change gives no save + its precise FAILED(p2_…); relink refused |
| X7 HTTP ambiguity | `http.json`, §4.3 | crash after send → UNCERTAIN; never inferred from the request ID |
| X8 save false-negative | E7, §4.4 | conflicting evidence → RECOVERY_REQUIRED; never plain COMPLETED or retry |
| X9a delayed POST vs copy | §4.5 | Copy disabled until terminal; 404 never shown as terminal |
| X9b **pause after E2, before E5** (GPT-6 P0-1) | §4.5 void | handler redeems, is paused, owner voids (succeeds), handler resumes → `EEXIST` → VOID; **0 HTTP**; the copy runs with ≤1 HTTP. Reverse order (handler claims first): void returns the run; NOT_DELIVERED is never shown. 100 randomized interleavings |
| X10 Core crash lifecycle | §6 dispatcher | startup and GET make no POST; ≤1 HTTP |
| X11a partial restores | §4.6 | Core-only / Engine-only / Thought-without-ledger → processing locked until a `tass.sh` restore |
| X11b **whole-app pre-claim restore, draft survives** (GPT-6 P0-2) | §4.6 fence | after `tass.sh restore-previous` the old draft is PRE_RESTORE; process → 409; direct Engine POST without a bound decision → 403; **0 HTTP for the old ID**; Copy as new works with a new ID |
| X11c repeated restores; restore of the guard snapshot | §4.6 | the fence's N always covers every draft that existed at the latest restore; no old ID becomes processable |
| X12 privacy boundaries | §5 | a synthetic marker in the attention is **not** in the stored prompt in `calls.jsonl` (digest only), Engine run files, Core run files, Identity, Core event log, GitHub or `tass.md`. The reply files may contain it only if the model echoed it; that is reported, not asserted absent |
| X13 disk/read-only/cap | §4.7 | refused before HTTP; status readable; nothing rotated or truncated |
| X15 Core submit ambiguity (P1-4) | §3 C5, §4.2 | short write / fsync error / dir-fsync error / crash after create, each followed by a restart: never a second send; unparsable → RECOVERY_REQUIRED, persistent; a new click on that nudge → 200 with that state |
| X14 completion cross-product | §4.4 | outcome {COMPLETED, FAILED, missing} × Notes {0,1,2} × steps {0,1,2}: only (COMPLETED or recovered, 1, 1) is COMPLETED; all conflicts RECOVERY_REQUIRED |

## 11. Owner decisions (proposed defaults; **not** approved by reviewers or by this document)

| ID | Question | Proposed default | Open tradeoff |
|---|---|---|---|
| D1 | Revocation while the model answers | Discard the note (P2); after the save it stands | — |
| D2 | Allowed with the Autonomous switch OFF? | Yes; refused when `TASS_CHAT_ONLY=on` is forced | — |
| D3 | One attempt per nudge | Yes; Copy as new only when terminal | — |
| D4 | Thought changed since the draft | Shown in the dialog, not refused; resolved = refused | — |
| D5 | Engine-side check | Yes: bound decision redeemed + grant at P1/P2 | — |
| D6 | Per-run consent | Dialog + bound decision (no passphrase per run) | stronger: passphrase per run (one more Identity step) |
| D7 | Retention | Keep run folders; refuse past 1,000 | owner archive/export later |
| D8 | O1 preflight | Exact Thought ID, Thought Processing switch, heartbeat/pulse state, model ready, grants, counts; read-only | — |
| D9 | Unlocking after a detected ledger divergence | **BLOCKED** except a `tass.sh` restore of a consistent backup; a read-only audit command for diagnosis | a reviewed repair operation later, if ever needed |
| D13 | After a `tass.sh` restore, old drafts are PRE_RESTORE (copy only) instead of locking all processing | Yes | locking everything is stricter but blocks unrelated new nudges |
| D14 | Owner "Stop this nudge" (void) | Yes; also run automatically before Copy as new | — |
| D10 | Attention text in `calls.jsonl` | Store only its digest for nudges | keeping it helps debugging |
| D11 | Gate the plain `think` route too | Separate later change | leave as R1 for now |
| D12 | Supported data-folder filesystems | Local POSIX only; `tass.sh check` warns otherwise | — |

## 12. If approved: order of work (separate draft PRs, CI, pinned SHAs)

1. **Identity:** grant + bound decisions + redeem (+ S1, X1, X2 at Identity level).
2. **Engine:** `flock`, dir fsync, run folder + phase files, nudge + void routes + read, P1/P2, startup scan (+ S3–S13, X3–X8, X11, X13, X14).
3. **Core + tass.sh:** process and void routes, per-nudge receipt files, dispatcher, verification, locks, restore fence and PRE_RESTORE, UI, pins (+ S2, S4, S6, S7, S16, S17, X9, X10, X11b/c, X12, X15, native).
4. Codex audit of the implementation → owner O1.

If any piece turns out to need more than this one slice (for example, the filesystem assumption fails on the owner's machine), it is marked **BLOCKED** rather than generalized.

## 13. Handoff for Codex (re-review)

Revision 3 first: please try to break **G1** (the void tombstone vs. a paused handler: X9b) and **G2** (the restore fence: X11b/c, including restores via every `tass.sh` path that calls `restore_data`), the Core C2→C3 order (S16), and per-nudge Core files (X15). Then, as before:
1. A-01/A-02: can any POST reach a claim or a successful replay without redeeming a fresh decision bound to the exact tuple? Is the `core`-only obtain / `engine`-only redeem split enforceable with Identity's existing service-key mapping?
2. A-03/A-04: is `O_EXCL` + file fsync + dir fsync, with "the file exists ⇒ claimed forever", a sufficient fence on the owner's filesystem? Does `flock` behave across the two PID namespaces in X4?
3. A-05/A-06: are P1 and P2 at the right lines (`ModelLink.run` after the slot; before `tx.put` in `Think`)? Is `http.json`-before-POST enough to make "none" truthful?
4. A-07/A-12: do you accept the per-continuous-history guarantee and divergence lock (§4.6, §8.1) instead of a restore-surviving ledger? If not, give the failure case.
5. A-09–A-11: the void tombstone (which replaces the time window), the single active run, relink refusal.
6. Mark S1–S15, X1–X14 and F1–F18 PASS-able / FAIL / OPEN as written.

No implementation, deployment, grant change or real nudge processing.
