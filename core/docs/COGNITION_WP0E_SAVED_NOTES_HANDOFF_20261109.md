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
