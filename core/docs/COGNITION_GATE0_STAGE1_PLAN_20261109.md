# Cognition Gate 0 + Stage 1: anchored implementation plan (2026-11-09)

> **"Cognition shows what was produced, Development lets you experiment in how it's produced."**

**What this is.** Planning, review and preparation only, as the owner approved on 2026-11-09 ("proceed with planning, review, and preparation for Gate 0 and Stage 1, but not implementing the entire three-stage plan at once").

**What this is not.** Nothing in this document is implemented, and no code changed. No Identity permission changed, no autonomy was started, and nothing was deployed.

It turns *TASS Cognition — Revised Engineering Plan, Revision 2* into work packages, each anchored to source. Every work package (WP) lists:
- the exact change;
- the files it touches;
- the data source each field comes from;
- the tests required;
- what it depends on;
- the authorization it needs.

Stages 2 and 3 appear only as anchors (§5), so later work starts from the same baseline.

**Revisions inspected:**

| Component | Revision |
| --- | --- |
| Core P0 | PR #108 `b79346b`, base installed `99f95be` |
| Engine ST2-I02 repair | PR #20 `40d4e91`, base installed `8bed0fc` |
| Identity | `cfc4079` |
| Design | #107, #106 (incl. GPT-6's `BUILD_INSTRUCTIONS_DRAFT.md` on `design/cognition-live-card-mockups-20261109` @ `7b08a07`) |


> **Revision 2 (2026-11-09, evening): reconciled with what is now installed.** The owner installed Core `1cd08b6`, which carries P0, the audited navigation fix `42d4d7d` and the audited Engine pin `40d4e91`. The owner then reported the inert "What TASS has written" button, which is fixed in **WP0-E** (draft PR #110, `4b5fea6`). This plan is now stacked on WP0-E. Where §1 and §3 below disagree with §8, **§8 is current**. §8.2 is the Slice 1 implementation brief, **ready but not started**: Stage 1 code waits until the owner accepts WP0-E.

---

## 1. Outstanding blockers (confirmed)

| # | Blocker | Status at time of writing | Evidence |
| --- | --- | --- | --- |
| B1 | **ST2-I02**: Engine `readRun` reported a still-running nudge as RECOVERY_REQUIRED on a torn run file | **Fix exists, not yet audited.** Engine PR #20 (`40d4e91`) returns RUNNING while `b.running` holds the nudge, plus a regression test. Owner authorized the repair (#106, 2026-11-09 20:36Z). | My checks on `40d4e91`: `go vet` PASS; `go test -race ./...` PASS; `TestNudgeReadRunPartialFilesDuringActiveWriter`, `TestNudgeVoidVersusPausedHandler` and `TestNudgeVoidNeedsBoundStopDecision` PASS. Native (Core #108 + Engine `40d4e91` + Identity `cfc4079`) **73/73 in 12/12 runs** (§1.1). |
| B2 | Codex independent audit of #108 + #20 | **NOT RUN** (handoff posted on #106) | — |
| B3 | Core's `tass.sh` still pins Engine `8bed0fc` | **Open.** Changing the pin to the audited Engine commit is a separate, owner-approved step (it changes what an update installs). | `core/tass.sh:33-34` |
| B4 | Browser **Back/Forward** don't return to the shelf | **Open, inside P0's scope.** #108 uses `history.replaceState`, so refresh works but Back leaves Cognition. | `cognition.html` `setHash()` |
| B5 | Owner reads the real Note on the installed build | **NOT RUN.** Needs an owner deploy decision after B1–B4. | — |
| — | `skill_native` "3 seeded lessons" | Pre-existing and unrelated (the seed has 4); fails identically on base. **Not a blocker**; keep it reported. | #104, #108 |

### 1.1 Integrated native run with the fix
Core #108 (`b79346b`) + Engine #20 (`40d4e91`) + Identity `cfc4079`, isolated ports, fixture model: **73/73 PASS in 12 of 12 consecutive runs**, with no connection to any owner port. Before the fix (Engine `8bed0fc`), the same harness passed 12 of 16 runs, and every failure was ST2-I02.

This is Claude's evidence, not Codex's audit. Docker, the owner machine and Nemotron: NOT RUN.

### 1.2 Review note on PR #20, for Codex
`readRun` now returns RUNNING whenever this process owns the nudge. That is correct while a writer is mid-file, with one effect to check:
- If **Stop** wins the claim race against a `ProcessNudge` handler that is still unwinding, `VoidNudge`'s response can say RUNNING once.
- Core then shows "Too late to stop / Not copied". On the next read the Engine reports VOID and Core records NOT_DELIVERED.
- The failure is safe: nothing is sent twice. It is a transient false "too late".
- `TestNudgeVoidVersusPausedHandler` passes. Codex should confirm the result converges, rather than treat it as a defect.

---

## 2. Can existing APIs support the Stage 1 data contract? Yes: Core-only, with no new Engine or Identity endpoint

| Stage 1 field | Source today | Supported? | Notes |
| --- | --- | --- | --- |
| Thought lifecycle (`active` / `sleeping` / `resolved`) | Engine `GET /api/entities/{eid}` → `thoughts[].state`; Core already copies it to `CognitionCard.State` | **YES** | The Engine's `thoughts[].state` is the only per-Thought state that exists. There is no per-Thought BUSY/WAITING/IDLE, and none is invented. |
| Developed Note **count** | Same payload: `thoughts[].notes[]` (the Engine sends full Thoughts; Core's P0 decode already reads `notes[].at/by/worker_request_id`) | **YES** | Count only. Text is never copied into the card. |
| Latest saved output **type + time** | Same payload: newest `notes[].at`, newest `history[]` with `kind=developed`; the Engine also has `last_developed_at` (not yet decoded by Core) | **YES** | Type vocabulary: `Developed Note` (Note saved), `Developed step (Note not stored)`, `none recorded`. |
| Who produced the latest output (`engine` / `owner-nudge`) | `notes[].by`, `history[].by` | **YES** | Shown as "Engine development" vs "Supervised nudge". |
| Per-Thought **run activity** | Core's own receipts: `data/core/nudge-runs/<id>/{submitted,outcome,void}.json` + `nudges.jsonl` (owner + Engine entity + Thought match) | **YES** | Read from Core files only (the same rule as P0's `activeRunLocal`). **No Engine call per card.** States: `Nudge in progress` (submitted, no outcome/void), or the last outcome (`completed · verified`, `needs recovery`, `stopped`, …). |
| Pulse / system activity | `GET /api/cognition/pulse` → `control` (RUNNING/PAUSED/UNKNOWN), `state` (READY/WORKING/BUSY/BLOCKED), `active_job`, `queued`, `running`, `last` tick, `every_s` | **YES** | Shown once, in the activity strip. The Engine's words are used; WAITING isn't an Engine pulse state, so it isn't shown there. |
| Staleness | Pulse `last.at` + `every_s`; library `observed_at`; fetch failure | **YES** | Rules in WP1-D. |
| Cycles / outcomes shelves | Existing library projection | **YES** | Unchanged. |
| A Note **excerpt** on the card | Would require putting Note text in the library list | **NO, by design** | Owner decision D1 limits Note text to the single-Thought detail. An excerpt needs its own owner decision (§6, Q1). |

**Two load facts the plan must respect (verified):**
- **Every Cognition read writes two Identity audit lines.** `cognitionAccess()` calls `Authorize` twice (`tass/cognition.go:113-125`), and Identity appends each decision to `decisions.jsonl` (`identity/internal/entity/entity.go:357-378`).
  - The existing pulse poll (10 s while visible) already causes about 17,000 lines a day with the page open.
  - Library auto-refresh would add 2 lines per refresh.
  - Stage 1 must therefore poll **slowly and only while visible**. Audit rotation or a read-session decision is an **Identity** change (separate authorization, §6, Q4).
- **The library payload grows with history.** `GET /api/entities/{eid}` returns every Thought with all its Notes (up to 200 × 400 chars each) and its full history. Core rejects any Engine response over **6 MiB** (`core/engine/cognition.go:31-32`). This is fine today, but a large Thought history could eventually make the library fail closed. Stage 1 adds no Engine data. WP1-E measures the payload. A trimmed Engine summary endpoint would be a later, separately authorized Engine change.

---

## 3. Gate 0 work packages (finish P0; no new stage work)

### WP0-A — Accept the ST2-I02 fix
- **Change:** none from Claude. Engine PR #20 as authored.
- **Steps:**
  1. Codex audits #20 (red on `8bed0fc`, green on `40d4e91`).
  2. Run Core #108 + Engine #20 native ×10 or more.
- **Depends on:** nothing else.
- **Authorization:** already given (repair + audit). Merging or pinning needs a separate decision.

### WP0-B — Browser Back/Forward in P0 (inside #108; small)
- **Change:** in `core/cmd/tass-core/cognition.html`:
  - `setHash()` uses `history.pushState` when opening a Thought or focusing a development, and `replaceState` for the in-place focus updates that follow.
  - **Close** calls `history.back()` when the page pushed the entry, and otherwise uses `replaceState` to an empty hash.
  - A `popstate` handler reopens or closes the sheet from `location.hash`, using `openFromHash()`.
  - On close, focus returns to the originating tile, and the shelf scroll position is restored.
- **Files:** `cognition.html`, plus `test/browser/cognition_p0.py` extended.
- **Tests:**
  - Open Thought → Back closes the sheet and stays on `/cognition`, focus on the same tile.
  - Forward reopens the same Thought and Note.
  - "Read what TASS wrote" → Back returns to the nudge history.
  - Refresh still reopens the Note.
  - Zero POSTs in all of the above.
  - All checks at 1280, 960 and 390 widths.
- **Depends on:** nothing.
- **Authorization:** it is P0 scope ("preserve normal Back/Forward", Revision 2 Gate 0). It adds a commit to #108 **before** Codex audits, or Codex re-audits the delta.

### WP0-C — Pin and integrated evidence (preparation only)
- **Change, proposed:** in `core/tass.sh`, `ENGINE_COMMIT_PIN` moves to the audited #20 commit, keeping the comment line format that `pin_of` requires.
- **Tests:**
  - updater 16/16, launcher 6/6;
  - native isolated harness ×10 with the pinned binaries;
  - `nudge_native`, `mailbox_native`, `pulse_multicycle_native`; `skill_native` stays a known FAIL.
- **Authorization:** **owner decision**. The pin decides what `./tass.sh update` installs.

### WP0-D — Owner acceptance (O1)
On the installed build, the owner:
1. opens `identity-test`;
2. reads the Note at half screen;
3. opens the receipt, then "Read what TASS wrote";
4. uses Back, then Refresh.

Recorded as PASS/FAIL by the owner. **Authorization:** an owner deploy decision.

**Gate 0 exit:** B1–B4 closed, Codex PASS on #108 + #20, the owner's decision recorded. Stage 1 implementation starts only after this.

---

## 4. Stage 1 work packages (truthful previews + auto-refresh): Core only

**The order is chosen so that each package can ship and be checked on its own.**

### WP1-A — Card data contract (Go)
In `core/tass/cognition.go`, add to `CognitionCard` for `kind=thought` only:

```jsonc
"lifecycle": "active" | "sleeping" | "resolved" | "NOT RECORDED",   // Engine thoughts[].state, enum-checked
"notes_count": 3,                                                    // len(notes) as received; capped display "200+" if Engine cap hit
"latest_output": {                                                   // null when none recorded
  "kind": "DEVELOPED_NOTE" | "DEVELOPED_STEP_NOTE_NOT_STORED",
  "at": "2026-11-09T18:20:00Z",
  "by": "engine" | "owner-nudge" | "NOT RECORDED"                     // from notes[].by / history[].by, normalized
},
"run_activity": {                                                    // Core receipts only; null when no nudge for this Thought
  "state": "IN_PROGRESS" | "COMPLETED_VERIFIED" | "NEEDS_RECOVERY" | "STOPPED" | "FAILED" | "UNCERTAIN",
  "nudge_id": "nudge_…", "at": "…"
}
```

Rules:
- `Summary` stays the question.
- **No field may carry Note text.**
- `CognitionCard.State` keeps its current meaning, for compatibility with existing fixtures.

Helpers:
- `thoughtPreview(th engine.Thought)` goes in **`core/tass/cognition_notes.go`**, which is P0's file. It shares the request-ID and `by` normalization.
- `runActivityForThought(scope, tid)` goes in **`core/tass/nudge.go`** and reads the journal and receipts once per library call. It builds a map by Thought: **one pass, not one per card**.
- `engine.Thought` gains `LastDeveloped *time.Time` (`json:"last_developed_at"`), used only as a cross-check.

**Tests** (`tass/cognition_preview_test.go`, new):
- the library has a correct count, kind, time and `by` for 0, 1 and many Notes;
- an evicted Note gives `DEVELOPED_STEP_NOTE_NOT_STORED`;
- unknown or garbage `state` / `by` gives `NOT RECORDED`;
- **a marker planted in Note text never appears in the library JSON**;
- run activity: in progress, verified, needs recovery, stopped, and another owner's nudge for the same Thought ID → none;
- **the Engine request count for one library read is unchanged** (`/api/status`, `/api/entities/{eid}`, `/api/cycles`), with no per-card calls;
- LOCKED makes no Engine call; an unlinked identity gets no Thought cards (the existing test, extended).

### WP1-B — Card rendering (HTML)
In `core/cmd/tass-core/cognition.html`, `draw()` for Thought tiles:
- **Line 1:** the title.
- **Line 2:** the question (as today).
- **Badges:**
  - a lifecycle badge: Active / Sleeping / Resolved;
  - the latest output: "Developed Note · 2:14 PM · supervised nudge" or "No output recorded";
  - a run badge, only when `run_activity` is set: "Nudge in progress" / "Completed · verified" / "Needs recovery".
- **Accessible label:** "Open Thought: <title>".

Rules:
- **No card ever says BUSY, WORKING or THINKING.** Those words appear only in the activity strip, and only when the pulse reports them.
- Tile clicks keep using `select()`, which opens P0's single-Thought detail.
- Optional, decided by the owner (§6, Q2): a "Most recently developed" featured tile. It is the Thought with the newest `latest_output.at`, labelled exactly that, never "current".

**Tests** (`test/browser/cognition_p0.py` extended, or a new `cognition_stage1.py`):
- the badges render from the fixture;
- no "BUSY", "WORKING" or "THINKING" text on any Thought tile;
- the run badge appears only with `run_activity`;
- the tile opens the same Thought;
- 1280, 960 and 390 widths have no horizontal overflow inside a tile;
- keyboard (Tab + Enter) opens a tile.

### WP1-C — Activity strip (pulse) wording
- **Change:** `loadPulse()` in `cognition.html`. Presentation only; the endpoint is unchanged.
- **Status, in plain words:**

| `control` | `state` | Shown as |
| --- | --- | --- |
| PAUSED | any | "Paused (the owner's setting). Nothing runs on its own." |
| RUNNING | READY | "Ready — no job running" |
| RUNNING | WORKING / BUSY | "Working: job \<active_job\>" |
| RUNNING | BLOCKED | "Blocked: \<note\>" |
| UNKNOWN or fetch failed | — | "Unavailable since \<time\>" |

- **Also shown:** queued / running counts and the last tick time.
- **Removed:** the raw tick dump moves into a collapsed "Recent ticks".
- **Tests:** a fixture for each row of the table above. PAUSED is not shown as an error. A fetch failure shows "Unavailable", never the previous state.

### WP1-D — Non-destructive auto-refresh + staleness
In `cognition.html`, add a new `refreshQuiet()`, separate from `refresh()`, which clears the selection and must stay as it is for the Refresh button.

**What it does:**
1. GET `/api/cognition/library`.
2. Merge by `card.id` into `S.cards`. Update tiles in place and add new ones at the shelf start. Never duplicate. Remove cards that vanished only after **two** consecutive reads, so a single partial read doesn't drop them.
3. Keep `S.selected`, the open sheet, the focused development and each rail's `scrollLeft`.
4. If the open Thought's `recorded_at` changed, re-GET its detail **only if the owner hasn't scrolled inside the sheet in the last 10 s**. Otherwise show "Updated — show latest".

**Cadence:**
- library every **60 s** while the tab is visible;
- pulse stays at 10 s while visible;
- nothing while hidden;
- exponential backoff to 5 min on errors;
- no request overlap: a new request is skipped if the previous one is still pending.

**Staleness:**
- The strip and shelves show "as of HH:MM:SS" (from `observed_at`).
- On a failed read, cards stay but are dimmed under "Not current: Cognition couldn't be read at HH:MM (reason). Showing what was recorded before." No state is shown as current.
- Pulse is RUNNING but no tick for more than 3 × `every_s` → "No pulse tick recorded since HH:MM".

**Tests** (browser fixture with a scripted server):
- a new Note appears without a click;
- the selection, open sheet and scroll position survive 3 refreshes;
- there are no duplicate tiles after 5 refreshes;
- a card that vanished once is still shown, and is removed after the second miss;
- an Engine 503 shows the dimmed "Not current" state;
- recovery restores it;
- a hidden tab makes zero requests;
- **zero POSTs across 10 minutes of simulated time**;
- request count per minute ≤ 1 library + 6 pulse.

### WP1-E — Bounded-load evidence (no code)
Measure:
- library response size and latency, and Engine `/api/entities/{eid}` size, on the native harness with 1, 50 and 100 Thoughts × 50 Notes;
- Identity `decisions.jsonl` growth per hour with the page open.

Report this against the 6 MiB bound. If more than 50% of the bound is used at 100 × 50, file an Engine summary-endpoint proposal (separate authorization). No mitigation goes in Stage 1 silently.

### Stage 1 regression set (all packages)
Run all of these unchanged and passing:
- Core vet / test / race;
- the P0 unit and browser tests;
- the Stage 2 native harness (isolated, 8797 protected) ×10;
- updater 16, launcher 6, `nudge_native` 25, `mailbox_native` 20, `pulse_multicycle_native` 24;
- `skill_native` (known FAIL).

Docker, the owner machine and Nemotron: NOT RUN unless the owner runs them.

### Stage 1 exit (Revision 2 acceptance)
The owner sees the latest recorded outputs and correct system state without pressing Refresh, and never mistakes history for running work. Verified by the browser fixtures, then by an owner walkthrough on the installed build.

---

## 5. Anchors for later stages (not planned in detail here)

- **Stage 2 (expanded Thought)** is P0 (#108) plus WP0-B. The remaining Revision 2 item is the wording for executed actions: "No action was executed from this Note" for any development without a verified nudge receipt. That is one line in `devCard()` plus a test. It could join WP1-B, or be its own slice.
- **Stage 3 (first usable Development workflow — owner milestone, revised 2026-11-10):** the read-only P1 handoff in #107 §3.2 is **an entry condition, not the Stage 3 exit**. Stage 3 must also prove one small, owner-driven **Cognition → Development → supervised experiment → comparison → owner decision** loop. See the acceptance contract immediately below. It requires separate design/implementation approvals for P1 (D3) and the isolated experiment contract in #107 §3.3 (D4–D7); one real local-model run requires its own authorization (D9). No work is authorized by this planning amendment.

### Stage 3 milestone: owner can actually develop *through* Development (2026-11-10 clarification)

**Owner outcome.** In the installed TASS application, the owner can take one real recorded TASS decision or Thought, open an intelligible Development work item, propose and test one bounded change through the Scratch-like workflow, inspect what TASS actually did and why (with evidence), and accept/reject the proposal. **Stage 3 is not complete because the graph draws lines, a page opens, a GitHub message arrives, or a fixture passes.** No separate terminal, raw JSON hunt, or direct GitHub chat is required to complete the owner-facing test flow.

**Minimum vertical slice, in order:**

1. **Open a real work item.** From a recorded Thought/decision in Cognition, open Development with the originating Thought, run/receipt IDs, project/work-item goal, observed result, and source revision intact. The work item persists across refresh/restart and has one explicit owner-visible next action. A linked GitHub issue/PR may remain the code source of truth; Development is the owner's interaction surface, not a replacement for Git.
2. **Understand the decision.** Show recorded observation → instructions/context → selected action or no-action → authorization → model/tool output → evidence and outcome, with source/version and honest RECORDED / CODE-DEFINED / NOT RECORDED / NOT EXECUTED labels. Do not pretend the model's private reasoning or an unexecuted skill exists. The P1 read-only trace is the foundation.
3. **Make a meaningful edit.** The owner can change one constrained, *executable experimental input* in Development (initially the instruction block from #107 §6; a small block/card on the canvas is acceptable). The graph must clearly distinguish **functional inputs wired into the supervised experiment** from advisory edges; a purely decorative node/edge change cannot satisfy this requirement. Show exactly what will change and what remains fixed.
4. **Run a protected test and compare.** Save a draft (zero model calls), then explicitly authorize the isolated A and B runs (exactly one call per arm, no retries), keep the original Thought and live graph unchanged, and show both actual outputs and recorded decisions side by side. If the edit doesn't change the output, show that result honestly. Use the #107 P2 isolation, per-arm Identity-bound grant, durability/replay, no-double-run, revocation, restore-fence, and owner/entity separation tests; reuse the existing Junction compare pattern where applicable.
5. **Decide and retain evidence.** The owner records **better / same / worse / inconclusive**, adds a correction or next instruction, and can reopen the saved comparison from Development or Cognition. Provide a sanitized, user-initiated handoff/link suitable for the existing GitHub collaboration workflow, without exporting private Note/prompt text or automatically changing repositories. This is **feedback/evidence**, not proof TASS learned or an automatic production change.

**Stage 3 acceptance / definition of DONE:**
- **Owner-host demo:** the owner, at approximately half-screen width, completes the five steps **inside the installed Development interface**, using one real local Nemotron experiment only after explicit approval. The owner can identify what changed, what TASS actually produced, and whether it improved. No coaching through terminal commands is needed for the normal path.
- **Causal proof:** a controlled A/B run changes an input actually consumed by the experiment runtime; provenance and outputs are verified from receipts. Changing a static advisory graph alone is **not** proof. A no-difference outcome is a valid experiment result if labelled accurately.
- **Safety proof:** Identity gates every execution; the original Thought, live graph, code and project files are not modified by the experiment; failures/denials/revocations/restarts show truthful states; private content stays out of library lists, logs and automatic GitHub sync. Unit, browser (including half-screen), race, native/negative and independent security gates pass at one pinned revision.
- **Owner verdict:** the owner explicitly records PASS/FAIL after the walkthrough. Fixture-only, mock-only, navigation-only and unaudited work remain **NOT ACCEPTED**. If P2 authorization or a real-model run is withheld, Stage 3 remains pending rather than being described as complete.

**Boundary with Stage 4.** Stage 3 proves *one supervised project/work-item loop*. Stage 4 may broaden to multiple projects, richer block libraries, AI collaborator orchestration, reviewed code-change proposals, and more complete project-building workflows. It must not be used to defer the first useful Development interaction. Do not add autonomy, arbitrary code/notebook execution, unattended GitHub commits, promotion/deployment, new services or new grants without separate explicit review.

**Delivery order:** 3A read-only provenance (#107 P1) → 3B a single functional editable experimental input and protected A/B (#107 P2) → 3C persisted work item, owner feedback/handoff and installed-workflow acceptance. Each slice is independently scoped and audited under #111; later-stage planning cannot reopen already accepted P0/Stage 1 work.
- **Live observation.** Inventory the event sources before any design:
  - Engine `activity.jsonl` (`kind`, `entity`, `detail`) and `calls.jsonl`;
  - pulse ticks;
  - the cycle journal;
  - Core's in-memory event log (300 entries);
  - nudge receipts.

  WP1-D's polling is the first step. No stream, animation or role feed until real events with provenance exist.

---

## 6. Owner decisions needed before Stage 1 coding

| # | Question | Recommendation |
| --- | --- | --- |
| Q1 | Do cards show only type / count / time, or an **excerpt** of the Note's words? | Type / count / time only. An excerpt would put private text into the list response (it changes D1). |
| Q2 | A "Most recently developed" featured tile? | Yes, with exactly that label (never "current" or "thinking"). |
| Q3 | Auto-refresh cadence: library 60 s, pulse 10 s, visible tab only | Yes |
| Q4 | Identity audit growth from Cognition reads (≈ 2 lines per read, already ≈ 17k lines/day from the pulse with the page open) | Accept for now. A later Identity change (rotation, or one read-session decision) needs its own authorization. |
| Q5 | Engine pin update (WP0-C) after the Codex audit of #20 | Owner decision at Gate 0 exit |

## 7. Smallest safe sequence

1. **Gate 0:** WP0-B (Back/Forward) on #108 → Codex audits #108 + #20 → WP0-C pin, after the owner decides → WP0-D owner check on the installed build.
2. **Stage 1, slice 1:** WP1-A + WP1-B (truthful cards, Core only).
3. **Stage 1, slice 2:** WP1-C + WP1-D (strip wording, auto-refresh, staleness).
4. **Stage 1, slice 3:** WP1-E (load evidence) → owner walkthrough.

Each slice gets its own draft PR, Codex audit and owner decision. Stage 2 wording, Stage 3/P1 and any live feed are separate assignments.

**Authorizations at a glance:**

| Work | Needs |
| --- | --- |
| WP0-B | P0 scope (already approved). Codex re-audits the delta. |
| WP0-C (pin) | Owner decision |
| WP0-D (deploy + walkthrough) | Owner decision |
| Stage 1 slices 1–3 | A new implementation assignment after Gate 0. Core only. No Engine, Identity or permission changes. No model calls. |
| Engine summary endpoint, Identity audit change, Note excerpts, P1, P2 | Each separately |


---

## 8. Revision 2: reconciliation and the Slice 1 brief

### 8.1 Gate 0 status now

| Item | Status | Evidence |
| --- | --- | --- |
| B1: ST2-I02 Engine fix | **CLOSED.** Codex PASS (isolated fixture scope); pinned in installed Core `1cd08b6` | #106 comment 6751479291; `core/tass.sh:34` = `40d4e91…` |
| B2: Codex audit of P0 + navigation | **CLOSED** for `b79346b` and for the `42d4d7d` delta; both are in `1cd08b6` | #106 |
| B3: Engine pin | **CLOSED** (`1cd08b6`) | — |
| B4: Back/Forward | **CLOSED** (`42d4d7d`, audited) | — |
| B5: the owner reads the real Note on the installed build | **PARTIAL.** The Note is visible on the owner's machine, but the owner reported an inert "What TASS has written" button. | #106 comments 6753571562, 6753622199 |
| **WP0-E**: saved-Note affordance | **Implemented, NOT audited, NOT installed.** Draft PR #110 (`4b5fea6`, base `1cd08b6`): the button is gone or turned into "Back to saved Notes", every action shows a visible result, and there is a regression test at 1280/960/390. | #110 |
| **Gate 0 exit** | **OPEN**: Codex delta audit of #110 → owner install decision → the owner's own click-through accepts it | — |

### 8.2 Slice 1 brief: truthful, metadata-only preview cards (ready, NOT started)

- **Base:** the accepted WP0-E head (`4b5fea6` or its audited successor). **Never** build on a head older than `1cd08b6`, because that would drop the Engine `40d4e91` pin.
- **Scope:** WP1-A + WP1-B only. Featured card (Q2), auto-refresh (WP1-D) and strip wording (WP1-C) stay **proposals**. The Identity audit-log growth is measured (WP1-E) before any auto-refresh.

**Exact changes** (anchors at `4b5fea6`):

1. **`core/engine/cognition.go:73-85`** (`type Thought`): add `` LastDeveloped *time.Time `json:"last_developed_at,omitempty"` ``. It is decoded, and used only to cross-check `latest_output.at`; it is never shown on its own.
2. **`core/tass/cognition.go:28-41`** (`type CognitionCard`): add
   - `` Lifecycle string `json:"lifecycle,omitempty"` ``
   - `` NotesCount *int `json:"notes_count,omitempty"` ``
   - `` LatestOutput *CognitionOutput `json:"latest_output,omitempty"` ``
   - `` RunActivity *CognitionRunActivity `json:"run_activity,omitempty"` ``

   These are set **only** for `kind=thought`.
3. **`core/tass/cognition.go:202-212`** (thought card construction in `CognitionLibrary`): fill the new fields from `thoughtPreview(th)` and the per-Thought map from `runActivityByThought(scope)`. That map is built **once**, before the loop.
4. **`core/tass/cognition_notes.go`** (P0's file): new `thoughtPreview(th engine.Thought) (lifecycle string, count int, latest *CognitionOutput)`.
   - `lifecycle` is `active`, `sleeping` or `resolved`; anything else is `NOT RECORDED`.
   - `latest` is the newest of: a Note (`DEVELOPED_NOTE`), or a developed step with no Note for its request ID (`DEVELOPED_STEP_NOTE_NOT_STORED`).
   - `by` is normalized to `engine`, `owner-nudge` or `NOT RECORDED`.
   - **The function reads no Note text.** A unit test plants a marker in the text to prove it.
5. **`core/tass/nudge.go`** (next to `activeRunLocal`, `:144`): new `runActivityByThought(scope) map[string]CognitionRunActivity`.
   - It reads `nudges.jsonl` once (`readNudges`, `:346`), filtered by owner **and** Engine entity.
   - It then reads Core's own receipts per nudge: `submitted`, `outcome`, `void`.
   - **It makes no Engine call.**
   - Per Thought it keeps the newest nudge: `IN_PROGRESS` (submitted, no outcome or void), `COMPLETED_VERIFIED` (outcome COMPLETED and `verified`), `COMPLETED_NOT_VERIFIED`, `NEEDS_RECOVERY`, `FAILED`, `UNCERTAIN` or `STOPPED`.
6. **`core/cmd/tass-core/cognition.html:92-115`** (`draw()`, Thought tiles only):
   - a lifecycle badge: "Active" / "Sleeping" / "Resolved";
   - an output line: "Developed Note · <time> · supervised nudge | Engine", or "No output recorded";
   - a run badge, only when `run_activity` is set: "Nudge in progress" / "Completed · verified" / "Needs recovery" / …;
   - the tile's `aria-label` becomes "Open Thought: <title>".

   No card text may contain BUSY, WORKING or THINKING. Text is set with `textContent` only. Tile clicks keep using `select()`, which is P0's view plus navigation.

**Tests:**

| Test | Checks |
| --- | --- |
| New `core/tass/cognition_preview_test.go` | 0, 1 and 200 Notes; an evicted Note; garbage `state` / `by`; **a Note-text marker absent from the library JSON**; run activity for each receipt state; another owner's nudge on the same Thought ID → no badge; **the Engine request set for one library read is unchanged** (`/api/status`, `/api/entities/{eid}`, `/api/cycles`); LOCKED → no Engine call |
| New `core/test/browser/cognition_stage1.py` | Badges from fixtures; **no BUSY/WORKING/THINKING on Thought tiles**; the run badge only with `run_activity`; tile → the same Thought's saved Notes; keyboard opening; no overflow at 1280/960/390; zero writes |

**Regressions:** Go vet/test/race; all P0 browser fixtures, including `cognition_affordance.py` and `cognition_nav.py`; the isolated native harness ×10; updater 16; launcher 6.

**Dependencies:** WP0-E accepted. No Engine, Identity or permission change. No model call. No new route.

**Authorization needed:** an owner go-ahead for Slice 1 coding after WP0-E acceptance. Deployment is separate, as always.

**Stop point:** a draft PR with evidence → Codex delta audit → owner decision. WP1-C/D/E come after, as separate slices.
