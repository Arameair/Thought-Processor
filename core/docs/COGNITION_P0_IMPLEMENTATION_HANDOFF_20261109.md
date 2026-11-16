# P0 "Cognition shows what was produced": implementation handoff for Codex's audit (2026-11-09)

> **"Cognition shows what was produced, Development lets you experiment in how it's produced."**

- **Authorization:** owner approval of P0 only, with D1 (show Note text to the unlocked, linked owner in the single-Thought detail) and D2 (show grant state for display only). See [#106 comment 6749442449](https://github.com/Arameair/Thought-Processor/issues/106#issuecomment-6749442449).
- **Design:** draft PR #107, §3.1 / §4 / §5 P0 / §7 A1–A11.
- **Scope:** Core only. No Engine, Identity, P1 or P2 change. No merge or deployment. No grants changed. Pulse not started. No real Nemotron.
- **Branch:** `experiment/cognition-p0-20261109`, based on installed Core `99f95be`. Engine `8bed0fc` and Identity `cfc4079` are unchanged and pinned as before.

## 1. What changed

| Area | Change | Files |
| --- | --- | --- |
| Note decode | `engine.ThoughtNote` now decodes the Engine's existing `at`, `text`, `by` and `suggested_state` (it used to keep only the IDs). `ThoughtStep` gains `by`. | `core/engine/cognition.go` |
| Developments | The single-Thought detail returns `developments`: newest first, at most 50, text clipped to 600 runes (the Engine keeps at most 400). They are correlated by `worker_request_id` with developed steps. For supervised nudges, Core's own receipt is **read** (`outcome.json` only; no Engine call, no write). `COMPLETED_VERIFIED` is shown only when that receipt verified the **same** request ID. A nudge ID that isn't in this owner's journal shows `NOT_IN_THIS_OWNERS_JOURNAL`. | `core/tass/cognition_notes.go` (new) |
| Persistence vs usefulness | Each development carries `persistence`, one of `NOTE_AND_STEP_CORRELATED`, `NOTE_ONLY`, `STEP_ONLY` (an evicted Note; nothing is invented in its place) or `IDS_DISAGREE`. It also carries a `meaning` sentence that says "saved, not that it answers the question". Suggestions show `suggested_state_applied:false`, because no code path applies them. | same |
| History fidelity | Keeps the **newest** 64 steps, plus `history_total`. Copies `source_nudge_id` (Stage 2 dropped it) and `by`. Step `detail` is still excluded, so the Note text appears once. | `core/tass/cognition.go` |
| Privacy | Note text appears **only** in `GET /api/cognition/thoughts/{id}`, behind the existing `cognitionAccess()` (UNLOCKED + `conversation` + `thought.process` + reviewed Engine link + entity match) with `no-store` and `nosniff`. It never appears in the library list, cycle exports, Copy Evidence or logs. Foreign or invalid IDs get the same 503 as "not found". `SaveNudgeDraft` (which holds `nudgeMu`) uses an internal check without developments, so there is no deadlock and no text. | `cognition.go`, `cognition_notes.go`, `nudge.go` |
| Truthful statuses | The hard-coded `nudgeBlocked()` and the unchecked per-nudge `AVAILABLE` are removed.<br>**List level:** `processing` (READY / NOT_READY with a plain reason) and `readiness` (Identity state, process grant read from Identity's public grant list with **no decision minted**, Engine reachability, processing lock, active run). The active run is read from Core's own receipts only, never from the Engine.<br>**Per nudge:** READY, WAITING, NOT_FOR_PROCESSING, EXPIRED, PRE_RESTORE or SEE_RUN.<br>New drafts store `DRAFT_SAVED`, which is never displayed. **Admission (C1–C5), bound decisions and the Engine's P1/P2 checks are unchanged.** | `core/tass/nudge.go` |
| Page | A Thought opens on **What TASS has written**: the question, then each development's full Note set via `textContent`, with author, time, request ID, nudge ID, persistence, "suggested … — not applied" and the meaning line. Record fields, history ("trace NOT RECORDED · request … · nudge …") and cycles move into a collapsed **Recorded details**.<br>A COMPLETED receipt gets **Read what TASS wrote**, which focuses the development with the same request ID. `#thought=…&dev=…` survives a refresh.<br>**Process once…** appears only when the derived state is READY. The disabled "Process — BLOCKED" button and the stale comments are gone.<br>Save, Copy as new and Stop-and-copy highlight and name the new draft and say nothing was processed.<br>The sheet is full width at ≤ 1100 px; the hero is compacted.<br>"Try adjustment in Development" is relabelled "Open Development (general workspace, not this run)", because P1 isn't authorized. | `core/cmd/tass-core/cognition.html` |
| Harness safety | Added **8797** to the isolated harness's `OWNER_PORTS`, which also extends the sentinel and refusal logic to it. The older fixed-port harnesses (`nudge_native`, `mailbox_native`, `skill_native`, `pulse_multicycle_native`, `lesson_retry_native`) now **refuse to start** when 8795, 8796, 8797, 7798, 8081 or 18083 already has a listener, because they would talk to a running TASS. The isolated harness opts out with `TASS_TEST_ISOLATED_PORTS=1`. | `core/test/nudge_process_native.py`, `core/test/lesson_retry_native.py` |

## 2. Tests and results (this workspace: no Docker daemon; TEST FIXTURE model, not Nemotron)

| Check | Result |
| --- | --- |
| `go vet ./...`, `go test ./...`, `go test -race ./...` (Core) | **PASS** |
| New `TestCognitionShowsWhatTASSWroteForAVerifiedNudge` (A1, A2, A5, no deadlock in SaveNudgeDraft) | **PASS** |
| New `TestCognitionHistoryKeepsNewestAndCorrelationIsHonest` (A9 newest 64 of 73, A10 STEP_ONLY, NOTE_ONLY, IDS_DISAGREE, foreign nudge ID, unsafe request ID NOT RECORDED, A6 library without text, GET-only Engine use) | **PASS** |
| New `TestCognitionNoteTextNeedsUnlockedLinkedOwner` (A8: a foreign Thought, an invalid ID, LOCKED with no Engine contact, and unlinked all get no text) | **PASS** |
| New `TestNudgeReadinessIsDerived` (A3: 6 Identity/grant/Engine combinations; never READY from the draft type alone) | **PASS** |
| Mutation checks: oldest-64 truncation, nudge ID dropped, READY from draft type each make the matching test fail | **PASS** |
| Browser `cognition_p0.py` at 1280×1000, **960×1000 (half window)** and 390×844 (A1, A2, A3, A4, A5, A6 Copy Evidence, A7 HTML-looking text shown literally with no element and no script, A11, refresh deep link) | **PASS 3/3** |
| Browser `cognition_go.py`, `cognition_nudge.py`, `cognition_process.py` (updated for the new labels and states) | **PASS 2/2, 1/1, 1/1** |
| Native isolated e2e `nudge_process_native.py`, real binaries (Core this branch, Engine `8bed0fc`, Identity `cfc4079`), 73 checks including 10 new P0 checks: full Note text, same request ID as the verified receipt, COMPLETED_VERIFIED, step keeps IDs, library without text, 401 without the token, derived readiness before/after revoke, **0 model requests from reads**, Engine Thought file **byte-identical** after reads | **FAIL (intermittent): 73/73 in 12 of 16 runs; 4 runs failed 2–4 Stage 2 checks.** Cause: ST2-I02 (§3), a pre-existing Engine defect. One P0 check (P0-A2) failed once, because it correctly reported the falsely RECOVERY_REQUIRED receipt; the other P0 checks passed in every run. |
| The unchanged Stage 2 harness at `99f95be`, 10 runs in this session | PASS 10/10 (the defect is timing-dependent, see §3) |
| Harness: a base overlapping 8797 → ABORT; a listener on 8797 → the legacy harness ABORTs before starting anything; 8797 is in the sentinel list | **PASS** |
| `test_updater_recovery.py` | **PASS 16/16** |
| `test_launcher_regression.py` | **PASS 6/6** |
| `mailbox_native` with the P0 Core | **PASS 20/20** |
| `nudge_native` with the P0 Core (its #96 assertion updated from `BLOCKED` to `NOT_FOR_PROCESSING`) | **PASS 25/25** |
| `skill_native` with the P0 Core | **FAIL 21/22**, the known pre-existing "3 seeded lessons" check (the seed now has 4; it fails identically on the base binaries) |
| `pulse_multicycle_native` with the P0 Core | **PASS 24/24** |
| Docker, the owner machine, real Nemotron, owner half-screen walkthrough (O1) | **NOT RUN** |

## 3. New finding ST2-I02 (Engine, pre-existing): a GET during a run can permanently mark it RECOVERY_REQUIRED

**Symptom.** In 4 of 16 P0 native runs, a nudge that the Engine completed and saved (one Note, one developed step, Engine `outcome.json` = COMPLETED) is shown by Core as **RECOVERY_REQUIRED** ("a run file exists but is unreadable"), and it stays that way.

**Cause** (Engine `8bed0fc`, `nudge.go`).
- `createOnce` opens each run file with O_EXCL and only then writes and fsyncs it.
- `readRun` (`GET …/nudges/{id}`) returns RECOVERY_REQUIRED for any file that "exists but is unreadable" **before** it checks whether this Engine is still running the nudge (`b.running`).
- A GET that lands between the open and the write of `http.json`, `response.json` or `outcome.json` therefore sees a terminal state. Core's `runView` then records that state as its permanent outcome.

**Deterministic proof against unchanged Engine code (not committed).** A probe test creates an empty `response.json` from the `after_http` hook (exactly what `createOnce` leaves mid-write) and calls `readRun` while the nudge is still running. The result is `state=RECOVERY_REQUIRED reason="a run file exists but is unreadable"`, where it should be `RUNNING`.

**Why P0 makes it show up more often.** P0 adds **no** new Engine run reads: readiness uses Core's own receipts, and the developments read Core files and the Engine Thought only. It does make each `GET /api/nudges` slower, because it adds one Identity state read and one Engine status read. With the instant fixture model, that slowness puts the first poll inside the ~15 ms run window more often. With real Nemotron the window is milliseconds within seconds, and the page polls every 3 s: rarer, but still possible.

**Safety.** It fails closed: the request is never re-sent, and the Note itself is saved correctly and visible in P0. The harm is a false "Needs recovery" receipt, which hides a real, verified success. That is exactly the kind of misleading status P0 is meant to remove.

**Proposed fix** (Engine PR #19; **not done, needs authorization**). In `readRun`, when `b.running` holds this nudge, return `RUNNING` before deriving any state from files. Add the probe as a regression test. Optionally, Core should not persist an Engine RECOVERY_REQUIRED observed while its own dispatcher is still sending.

**Recommendation.** Fix ST2-I02 before deploying P0. With the fix, the P0 native run should be 73/73 repeatedly.

## 4. For Codex: how to reproduce

```sh
# Core (this branch)
cd core && go vet ./... && go test -race -count=1 ./...
python3 test/browser/cognition_p0.py           # Playwright + Chromium
python3 test/browser/cognition_go.py; python3 test/browser/cognition_nudge.py; python3 test/browser/cognition_process.py
# native, isolated ports (aborts on any collision before starting anything)
go build -o /tmp/e2e/bin/tass-core-p0 ./cmd/tass-core
APP_HOME=/tmp/p0-new BIN=/tmp/e2e/bin SEED=… CORE_BIN=tass-core-p0 ENGINE_BIN=<8bed0fc build> IDENTITY_BIN=<cfc4079 build> \
  S2_PORT_BASE=39100 python3 test/nudge_process_native.py
python3 test/test_updater_recovery.py; python3 test/test_launcher_regression.py
```

**Audit focus:**
1. Is the Note text reachable only through the single-Thought detail and its authorization? Check lists, exports, Copy Evidence, logs and the GitHub sync paths.
2. Correlation: can a development borrow another nudge's receipt, or a receipt verify a different request ID?
3. Is any state derived from the draft type alone? Does anything weaken admission?
4. Rendering: `textContent` only. Are there any new `innerHTML` uses? The CSP is unchanged.
5. Harness: can any test reach 8795, 8796, 8797, 7798 or 8081 on a host with TASS running?
6. ST2-I02: confirm or refute it.

## 5. Not done (by design)

These are not part of P0:
- P1's run-record endpoint and the `/dev?run=` view;
- P2 experiments and `cognition.experiment.run`;
- any Engine or Identity change, including the ST2-I02 fix, which needs authorization;
- deployment;
- grant changes;
- autonomy;
- real model calls.

---

## 6. Navigation repair (after Codex's PASS on `b79346b`): the delta for Codex's final review

**Authorization:** "P0 Final Navigation Repair Before Deployment" (owner, 2026-11-09). This is the only change; Stage 1, P1 and P2 are untouched.

**Root cause, at `b79346b`.** Cognition's only history call was `history.replaceState`, in `setHash()`, and there was no `popstate` handler. Opening a Thought, the nudge history or a Note therefore created **no** browser history entry. A refresh worked, because `openFromHash()` read the hash on load, but Back went to whatever page came before Cognition. The nudge-history view wasn't in the URL at all.

**The change.** One file, `core/cmd/tass-core/cognition.html`, +51 −20 lines.
- **The URL describes every view.** It is empty (the shelves), `thought=<id>`, `thought=<id>&dev=<request id>`, `thought=<id>&view=nudges|draft`, or `card=<card id>` (a cycle, outcome or other card).
- **`navTo(state, "push"|"replace")`.** A user action **pushes** an entry: opening a card, "What TASS has written", "Nudge history", "Manual nudge (draft)", "Read what TASS wrote", and Close. Follow-on updates **replace** the entry: focusing the same Note, or a saved draft becoming the history view.
- **`route()` renders whatever the URL says.** It runs on page load, refresh and every Back/Forward (`popstate`). It only **selects** a card that is already in the authorized library response.
  - Unknown or hostile references show the shelves with "not in this identity's authorized library", and the URL is cleaned.
  - A Thought whose detail read is refused shows the existing honest error (for example "Thought detail HTTP 503"), with nothing invented.
- **Unchanged:** the APIs, projections, Note authorization, nudge processing, the shelf design, and the Note rendering (`textContent` only). No new `innerHTML`, `eval` or POST was added; I checked the diff.

**New regression test:** `core/test/browser/cognition_nav.py`, run at 1280×1000, 960×1000 and 390×844.

| Check | What it verifies |
| --- | --- |
| N1 | Shelf → Thought → Back leaves you on `/cognition` with the sheet closed → Forward shows the same Thought and Note |
| N2 | Refresh with a Thought selected reopens it |
| N3 | Completed receipt → "Read what TASS wrote" → Back to the nudge history → Back to the Thought → Back to the shelf → only then does Back leave Cognition → Forward ×4 returns to the same focused Note |
| N4 | Close → shelf; Back reopens what was closed |
| N5 | Direct links: `#thought=`, `&dev=`, `&view=nudges`, `#card=cycle:` |
| N6 | Unknown Thought, unknown card, path-traversal and HTML-shaped values: shelves, no script runs. A Thought whose detail returns 503 shows an honest error, nothing invented |
| N7 | Cycle card → Back → shelf |
| N8 | **Zero** non-GET requests across all navigation (no POST, no `/process`, no `/stop`) |

**Red/green:**
- The same test against `b79346b`'s `cognition.html` **fails at N1**: Back navigated out of Cognition to the previous page.
- On the repair, it passes 3/3.

### Results for the new commit (reported separately from Codex's historical evidence)
Codex's earlier evidence for `b79346b` + Engine `40d4e91`: 16/16 native runs and 1,168/1,168 checks. That is **historical** and does not test this commit.

| Check (new commit; Engine #20 `40d4e91`, Identity `cfc4079`, fixture model, no Docker) | Result |
| --- | --- |
| Core `go vet`, `go test ./...`, `go test -race ./...` | **PASS** |
| `cognition_nav.py` (N1–N8) at 1280 / 960 / 390 | **PASS 3/3** (and FAIL on `b79346b`, as expected) |
| `cognition_p0.py`, `cognition_go.py`, `cognition_nudge.py`, `cognition_process.py` | **PASS 3/3, 2/2, 1/1, 1/1** |
| Isolated native harness (73 checks) with the new Core binary, ports 39700–39704 | **PASS 73/73 in 8 of 8 runs**; isolation sentinel PASS in all 8 |
| `test_updater_recovery.py` | **PASS 16/16** |
| `test_launcher_regression.py` | **PASS 6/6** |
| Docker, the installed owner stack, real Nemotron, the owner's half-screen walkthrough | **NOT RUN** |
| Codex delta review of this commit | **NOT RUN** (requested) |

**Remaining defects:** none known in P0's scope. `skill_native`'s pre-existing seed-count check is unchanged and unrelated.

**Known behaviour, by design:**
- Close adds a history entry, so Back after Close reopens what was closed.
- Back leaves Cognition only after you have stepped back past the shelf, which is normal browser behaviour.
- History entries made before a refresh survive the refresh (the browser keeps them).
