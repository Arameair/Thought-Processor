# WP0-E: the saved-Note button that did nothing. Handoff for Codex's delta audit (2026-11-09)

> **"Cognition shows what was produced, Development lets you experiment in how it's produced."**

**Owner report** (installed Core `1cd08b6`, Engine `40d4e91`, Identity `cfc4079`): *"i dont see the read save notes i hit the button but nothing happened"*.

**Authorization.** [#106 comment 6753622199](https://github.com/Arameair/Thought-Processor/issues/106#issuecomment-6753622199) covers a bounded P0 usability correction only. There is no deployment, merge, permission, Engine or Identity change, no P1/P2 and no Stage 1 code.

**Base.** The deployed candidate `1cd08b683c674426dd06f8ff7a5fd4d00412580c`. It carries the audited navigation fix `42d4d7d` and the Engine pin `40d4e91`, and both are unchanged here: `tass.sh` is untouched.

## Root cause
- A Thought already opens on its saved Notes: `select()` calls `showThought()`.
- The bottom button labelled **"What TASS has written"** (`#expandEngine`) called `showThought()` again. That repeated the same authorized GET and redrew identical content, often out of view above the action row.
- Nothing visible changed, so the click looked broken.
- The Notes were saved and shown all along. This is an affordance defect, not a persistence defect.
- The same out-of-view redraw also affected **Manual nudge (draft)** and **Nudge history** when the owner had scrolled down to the buttons.
- For cycle cards, the same button was mislabelled "What TASS has written".

## Change: `core/cmd/tass-core/cognition.html` only (+31 / −11 lines across 4 files)
- **The Notes are unmistakable.** The view's heading is now **"Saved Notes — what TASS has written"** (`#savedNotes`). It sits first, in the main text colour, with a count line such as "1 saved development, newest first". When the Thought opens, focus moves to it.
- **No inert button.**
  - While the saved Notes are on screen, the bottom button is **hidden**.
  - In the Nudge history and Manual-nudge views, it reappears as **"Back to saved Notes"**. That adds a real history entry, shows the Notes again, and hides the button again.
  - For cycle and outcome cards, the button keeps its original label, **"Inspect recorded details"** (it opens the recorded log).
- **Every action shows a visible result.** A new `reveal()` scrolls the new view's heading into view only if needed (`block:"nearest"`, so Close stays on screen) and moves keyboard focus to it. It is used for Saved Notes, Nudge history, the Manual-nudge form and View logs. Copy Evidence keeps its status line, and Open Development navigates.
- **Unchanged:**
  - the APIs, projections, Note authorization (single-Thought detail only) and `textContent` rendering;
  - the receipt's **"Read what TASS wrote"**, which still focuses the Note with the same request ID;
  - Back/Forward/refresh from `42d4d7d`;
  - nudge processing.

  There is no new POST, `innerHTML` or model call.

## Tests

**New test, `core/test/browser/cognition_affordance.py`,** run at 1280×1000, **960×1000** and **390×844**. A 41-step history makes the sheet scroll, so the original "nothing happened" situation is reproduced.

| Check | What it verifies |
| --- | --- |
| E1 | On open, the Saved Notes heading and the newest Note are on screen and the heading has focus. No "What TASS has written" button exists, and no saved-Notes button shows while the Notes are displayed. |
| E2 | After scrolling to the action row, Nudge history and Manual nudge each change the view, put its heading on screen and focus it. Copy Evidence shows a visible confirmation (still no Note text). Open Development navigates. |
| E3 | "Back to saved Notes" really returns to the Notes and hides itself. Receipt → "Read what TASS wrote" focuses the same `mr_…`. Back/Forward work. |
| E4 | Refresh keeps the focused Note. |
| E5 | A denied (LOCKED) detail read shows "Thought detail HTTP 403; Identity denied", with no Saved Notes heading and nothing invented. |
| E6 | Zero POST, `/process` or `/stop` requests. |

**Red/green:** against `1cd08b6`'s page, the test fails at E1 (there is no Saved Notes heading, and the inert button is present). It passes 3/3 on this change.

**Updated tests** (label changes only): `cognition_go.py` asserts the redundant button is gone; `cognition_p0.py` and `cognition_nav.py` use the new heading text.

## Results (this workspace: no Docker; TEST FIXTURE model, not Nemotron; Engine `40d4e91`, Identity `cfc4079`)

| Check | Result |
| --- | --- |
| Core `go vet`, `go test ./...`, `go test -race ./...` | **PASS** |
| `cognition_affordance.py` (E1–E6) at 1280 / 960 / 390 | **PASS 3/3** (FAIL on `1cd08b6`, as expected) |
| `cognition_p0.py` / `cognition_nav.py` / `cognition_go.py` / `cognition_nudge.py` / `cognition_process.py` | **PASS** 3/3, 3/3, 2/2, 1/1, 1/1 |
| Isolated native harness (73 checks), WP0-E Core binary | **PASS 73/73 in 10 of 10 runs**: 6 runs with the first build, 4 with the final build (the later edits were CSS and scroll-mode only); isolation sentinel PASS in all 10 |
| `test_updater_recovery.py` / `test_launcher_regression.py` | **PASS 16/16 / 6/6** |
| Docker, the installed owner stack, real Nemotron, the owner's own click-through | **NOT RUN** (owner acceptance stays open until the owner tries it) |
| Codex delta audit `1cd08b6..<this head>` | **NOT RUN** (requested) |

## For Codex
- **Diff:** `1cd08b6..HEAD` of `experiment/p0-saved-notes-affordance-20261109`.
- **Focus:**
  1. Is any sheet action still without a visible result at 960 / 390, especially after scrolling?
  2. Is the hide/show logic correct across Thought → Nudge history → Back to saved Notes → Back/Forward?
  3. Is there any new data path, write or Note-text exposure?
  4. Is the `1cd08b6` pin unchanged?

---

## Repair 1 (2026-11-10): async view ordering, the delta for Codex's focused re-audit

**Source.** Codex audit [PR #110 comment 6766665902](https://github.com/Arameair/Thought-Processor/pull/110#issuecomment-6766665902) (NEEDS REPAIR) and the assignment in [#106 comment 6766771303](https://github.com/Arameair/Thought-Processor/issues/106#issuecomment-6766771303). Scope is P0 only: no Stage 1, Engine, Identity, permission, merge or deploy.

**Root cause** (pre-existing on `1cd08b6`; still present at `4b5fea6`):
- `showThought` checked only that the selected card *object* was unchanged after its GET. Switching views on the **same** Thought didn't change that object, so a slow detail response replaced the nudge history.
- `nudgeHistory` didn't check anything after its GET, so a slow history response replaced the Saved Notes after Back.
- The same exposure applied to:
  - **error toasts** from abandoned reads;
  - the **3-second receipt poll** (it re-armed itself, and its "still on the history?" test read the DOM text);
  - **owner-action results** (Save draft, Process, Stop, Copy) that re-rendered the history after the owner had moved on.

**Fix** (`core/cmd/tass-core/cognition.html` only, +36 / −18):
- **A view generation token.** Every change of navigation **intent** takes a new generation through `newView()`, which also cancels the receipt poll. The intents are:
  - `select()` (including Back/Forward/direct link through `route()`);
  - `hideSheet()` (Close, or the shelf route);
  - `refresh()`;
  - `showThought()`;
  - `nudgeHistory()`;
  - `nudgeForm()`;
  - `loadTimeline()`.
- **Stale reads are ignored.** Each async read remembers its generation. When it settles, it may touch the DOM, focus, toast, URL, view state or poll timer **only if** `!stale(g)`. Stale successes and stale errors alike are dropped.
- **The poll is tied to the view.** It re-arms only if its generation is still current (no more DOM-text check), so leaving the history cancels it.
- **Owner-action results** (Process / Stop / Stop-and-copy / Copy / Save draft) update the screen only while the owner is still on the view they started from (`live()`). The write itself is unaffected, and its durable receipt appears the next time the history is opened. This is the one intentional behaviour change: if the owner navigates away during a POST, its confirmation toast is not shown.
- **Unchanged:**
  - authorization (every read is still authorized server-side; the token is presentation only);
  - Note privacy and `textContent` rendering;
  - Back/Forward/refresh (`42d4d7d`);
  - the Saved Notes usability fix;
  - nudge processing;
  - `tass.sh` pins (Engine `40d4e91`, Identity `cfc4079`).

  No history entries are added, and no new POST, model call or `innerHTML` is introduced.

**New deterministic test `core/test/browser/cognition_async.py`.** The fixture **holds** chosen GET responses with no timers, changes navigation, then releases them in a chosen order. It accepts an alternative HTML path, so any revision can be re-tested. It runs at 1280, 960 and 390, with 11 checks per width:

| Check | Scenario |
| --- | --- |
| R1 | Delayed history → Back to Notes (Codex finding 1) |
| R2 | Delayed detail → Nudge history (Codex finding 2) |
| R3 | History → Back → Forward with three held reads, released newest-first and oldest-first |
| R4 | Cross-Thought: the abandoned Thought's **private Note never appears** |
| R5 | Delayed detail → Close |
| R6 | A delayed **403** for an abandoned view gives no error toast |
| R7 | A held poll response is ignored, and **no further polls** happen after leaving |
| R8 | Direct link with a delayed detail, then Back |
| R9 | Zero non-GET requests, no page errors |

**Red/green:**

| Page | Result |
| --- | --- |
| `4b5fea6` (Codex-audited candidate) | **FAIL, 21/33**: R1, R2, R6 and R7 fail at every width, in 3 of 3 repetitions |
| Repaired head | **PASS, 33/33**, in 3 of 3 repetitions |
| Installed `1cd08b6` | Can't run this test (it predates the Saved Notes heading). Codex reproduced the race there. |

**Other results (repaired head):**

| Check | Result |
| --- | --- |
| Go vet / test / race | **PASS** |
| `cognition_affordance` 3/3, `cognition_p0` 3/3, `cognition_nav` 3/3, `cognition_go` 2/2, `cognition_nudge` 1/1, `cognition_process` 1/1 | **PASS** |
| Updater 16/16, launcher 6/6 | **PASS** |
| Isolated native harness, final binary + Engine `40d4e91` + Identity `cfc4079`, fresh marked data, protected ports (39900–39904) | **PASS 73/73 in 8 of 8 runs** (584/584 checks); isolation sentinel PASS in all 8 |
| Docker, installed stack, real Nemotron, owner click-through | **NOT RUN** |
| Codex focused delta re-audit | **NOT RUN** (requested) |

**Residual notes for Codex:**
1. The generation token is page-local. A second browser tab is independent, as before.
2. Cycle playback (`advance()`) is a timer, and `select()` already stops it.
3. The confirmation toast for a POST is suppressed if the owner navigates away mid-request (described above).
