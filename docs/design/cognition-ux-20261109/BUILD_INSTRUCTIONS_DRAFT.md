# DRAFT BUILD INSTRUCTIONS — Cognition live-preview shelf and opened Thought card

**Date:** 2026-11-09  
**Repository:** `Arameair/Thought-Processor`  
**Design sources:** [shelf preview](./cognition-shelf-live-preview.svg), [opened Thought card](./cognition-clicked-thought-card.svg), [concept README](./README.md), [owner discussion #106](https://github.com/Arameair/Thought-Processor/issues/106).  
**Status:** **REVIEW DRAFT — NOT IMPLEMENTATION, MERGE OR DEPLOYMENT AUTHORIZATION.** Written to support owner/Claude/GPT-6 design review. Does not enlarge already-approved Claude P0.

## The weakest assumption: the mockup implies a continuous live feed
The current installed Cognition page has a **read-only Pulse/status GET** and static authorized recorded-card projections. A *20-second engine Pulse* is **not** a 20-second model generation stream, and an active Thought is not proof that a model is currently thinking. A live preview cannot honestly display new text, five-role activity, team decisions or colorful progress indicators unless the runtime records and exposes those events for this identity.

**Build against actual evidence before appearance.** Use three unmistakable distinctions:
- **Current activity:** explicitly recorded running/busy/waiting or current active job, if present in the authorized API.
- **Latest recorded output:** persisted Note, nudge receipt, cycle or other record; timestamp and provenance. Does not imply live execution.
- **Not recorded / unavailable / paused:** honest states; never animate or fabricate thinking to fill blank space.

**Product contract, preserve verbatim:**  
> “Cognition shows what was produced, Development lets you experiment in how it's produced.”

## 1. Baseline: don't throw away working code

**Observed source at deployed baseline Core `99f95be` and draft Claude P0 branch `experiment/cognition-p0-20261109` (PR [#108](https://github.com/Arameair/Thought-Processor/pull/108)).** Verify the actual version to work from before changing files; do not assume `main` represents deployed release.

| Current surface | Present implementation / relevant file | Implication |
| --- | --- | --- |
| Shelf + filter/search cards | `core/cmd/tass-core/cognition.html`: `shelves`, `draw()`, `select()`; `GET /api/cognition/library` | **Reuse** cards, filters, scroll rails, stable IDs, source labels. Improve hierarchy without rebuilding. |
| Thought list model | `core/tass/cognition.go`: `CognitionCard`, `CognitionLibrary()` (bounded 100 Thoughts), `CognitionThought()` | Library has title/summary/state/time/verification but **does not include full private Note text**. This is intentional. |
| Full saved Thought response | PR #108: `GET /api/cognition/thoughts/{id}`; `showThought()`, `devCard()` | P0 draft supports authenticated Note text, source nudge/request correlation, collapsed details. **Build on it rather than duplicate.** |
| Pulse status | `cognition.html`: `loadPulse()` performs `GET /api/cognition/pulse`; shows `control`, `state`, last tick, active job, queue/running and limited recent ticks | **Status telemetry, not an event stream.** Do not call it an observed live Thought feed unless new verified events exist. |
| Development link | `cognition.html`: `devProposal()` currently sends `cognition_ref` and `brain_revision` to `/dev` | Generic reference **does not** load exact Note/run workflow. Contextual link is a separately designed **P1** task. |
| Live output from five collaborating roles | Earlier owner intent; **not verified as available in current runtime** | Design future adapter with real artifact IDs and timestamps; don't render fake role reports. |

**P0 release prerequisite discovered during review:** [#108](https://github.com/Arameair/Thought-Processor/pull/108) reports *intermittent* Engine status race **ST2-I02**. Status read during write may cause false permanent `RECOVERY_REQUIRED`. The Engine fix needs its own authorization, regression tests, Codex independent audit, and owner-controlled deploy decision. **Do not treat P0 as deployed or stable based on mockups.** Preserve known `skill_native` seed-count failure evidence separately rather than silently treating all checks as green.

## 2. Screen contract A — Cognition home / shelf preview

**Job:** owner can glance at real activity and select the output worth investigating, as if browsing a Netflix shelf. **Not** the Scratch graph editor; that belongs in Development.

**Structure to implement incrementally:**
1. A compact, readable **status strip**, backed by `/api/cognition/pulse`: PAUSED / IDLE / BUSY / WAITING only where actually supported and evidenced; last recorded pulse timestamp, queued/running counts, and current job where recorded. Hide fake heartbeat waveforms and unverified "TASS thinking" labels. An explicit `PAUSED` state is normal and not an error.
2. A **featured/current Thought** area only when a stable authorized Thought is selected or grounded as active; show its question/topic, state and most recent recorded activity timestamp. If there is no active Thought, show “No active Thought recorded” and offer the latest recorded Thought instead.
3. **Persistent Thoughts** shelf using the existing cards, with accessible title, one-line purpose/question, recorded status, latest real output *type* (e.g. Developed Note), a grounded timestamp, and an obvious **Open Thought** affordance.
4. **Recent recorded cycles**, **reported outcomes**, then optional lessons/skills/notes shelves. Preserve separation between linked and unlinked cycles and between `NO_ACTION` and actual execution.
5. A **recent output activity** strip showing record type, source, saved time, and a link to the appropriate authorized detail. **Do not infer a Note snippet from a library list** or add private Note bodies to broad collection responses just to fill a preview.

**Proposed preview-card fields (verify availability before binding):** `kind`, `title`, `summary`, `state`, `verification`, `recorded_at`, `source_id`, optional authorized `latest_output_kind` / `latest_output_at` *only if justified by an approved safe projection*. If a field is not available, show `NOT RECORDED` or omit it, rather than guess.

**Interaction:** click a Thought preview → use the existing authorized single-Thought detail read. No POST or model call on navigation. Search/filter and keyboard access work on all cards.

**Potential data-contract decision requiring separate authorization:** If the owner asks for actual Note **text excerpts** on every mini-card, the currently reviewed privacy boundary only permits Note body in single-Thought detail. Explore a safe, explicitly authorized bounded excerpt strategy (and consequences for list leakage), or keep only Note type/title/time on previews. Do **not** silently broaden `/api/cognition/library`.

## 3. Screen contract B — opened Thought card

**Job:** owner reads exactly what TASS produced, why it was saved, where it came from, and what can be investigated. One clear selected result, not a raw information dump.

**Minimum content, in reading order:**
1. Thought title/topic + **original question** + real Thought state; distinguish **active Thought** from **actively executing run**.
2. **Full persisted Note** text under a visible label (e.g. “What TASS wrote”), note author, time, request/nudge correlation and persistence verification; distinguish “saved and correlated” from “correct answer.”
3. A short **what happened** record assembled from *actual* nudge receipt, developed step and authorized stored evidence. If absent, say `NOT RECORDED`. Never claim direct access to hidden model reasoning.
4. A small **outputs/history navigator** for multiple developments, steps and receipts, with selection/focus and newest first. The selected record is stable across refresh/deep links. Keep advanced trace IDs in a collapsible evidence pane.
5. Optional **playback** only for records actually supporting a chronological timeline; it is event replay, **not** playback of private internal chain of thought or a simulated moving model mind.
6. Future Team/Role artifacts, when genuinely persisted and authorized, with per-role contribution content and provenance. Explicit `NOT AVAILABLE` placeholder or hide until implemented.
7. **Investigate in Development** action, initially clearly labeled **general workspace / not this run** (current state). P1 will make it contextual to exact selected Note + Thought ID + request/nudge IDs; requires server-side validation and owner approval before implementation.

**UX details:** provide focused readable text; clear not-found/denied/unavailable states; collapsed engineering metadata; explicit return to shelf; preserve selection through refresh; no unexpected model execution or write.

## 4. Technical delivery sequence and stop gates

### Gate 0: finish the existing P0 baseline **before integrating new visuals**
- Codex reviews PR #108 and the cited Engine race. Obtain explicit authorization for any Engine fix.
- Resolve ST2-I02 with proper regression tests and preserve fail-closed behavior. Rerun deterministic isolated e2e tests and investigate existing failures before calling the release healthy.
- Owner explicitly decides whether to deploy after tests; install and manually verify P0's full Note in a half-screen window.
- **STOP if** wrong Note, permission leak, false nudge status, or test harness contacts running owner services.

### Gate 1: shelf/preview layout (separate proposed implementation slice)
- Adapt existing `cognition.html` card rails and compact header, keeping APIs read-only and existing auth.
- Display truthful state/last-update/time data and prominent **Open Thought** navigation.
- No new public Note text in library, no fake status waveforms, no new services, no Development graph inside Cognition.
- **STOP if** a preview claims unsupported activity or cross-entity records become visible.

### Gate 2: expanded Thought card refinement (separate proposed implementation slice)
- Reuse the PR #108 `showThought()/devCard()` and existing authenticated detail; do **not** write a second data store or Notes API.
- Add a clear output/history selector, evidence labels and responsive legibility. Preserve selected Thought/Note across reload.
- **STOP if** anything triggers a model POST on read, alters original Thought data, or displays model text on an unauthorized page.

### Gate 3: genuine activity preview / incremental updates (requires evidence and design decisions)
- Inventory actual available event data (Pulse, cycle and persisted activity), cadence, and correlation keys before selecting polling, SSE, or other transport.
- Lowest-risk first option is periodically refreshing **only authorized status/list projections** while keeping the selected item stable. That does not by itself create a live thinking feed.
- Introduce any new stream or event projection only if a recorded backend event exists, authorization is scoped per entity, resource use is bounded, reconnect/backoff is safe and no fabricated stages appear.
- Respect the existing ~20-second Pulse as a **coordination/check interval**, not a run deadline. Busy/Waiting work spans pulses.
- **STOP if** UI refresh creates duplicate records, suggests work finished prematurely, adds model calls, or burdens the local model just to update the screen.

### Gate 4: contextual Development jump (**P1, requires separate owner approval**)
- Include exact authorized Thought and selected development/run/nudge reference, resolve server-side under Identity, and show only actual grounded processing stages.
- `/dev` opens corresponding context; not a generic unrelated graph. Provide return to same Cognition card.
- Graph editing, skill substitution and experiment execution remain **separate future stages**, not part of this build scope.
- **STOP if** untrusted URL parameters can select another entity's record or if a click has side effects.

## 5. First acceptance matrix (definitions, **NOT results**)

| Scenario | PASS condition |
| --- | --- |
| Owner opens `identity-test` | Original question and **actual** persisted Note readable without Python/JSON or another model call |
| Privacy / linked entity | Locked, wrong entity and unlinked owner cannot see Note; broad shelf payload never contains private Note bodies |
| Normal state | Last output timestamp + provenance match records; **saved** is not labeled **correct** |
| Pulse paused | UI plainly shows **PAUSED** and latest history, with no artificial live animation |
| Busy / Waiting | Work persists across 20-second checks; UI neither restarts nor claims timeout |
| Card preview | Title/status/latest *recorded* output is discoverable, card opens correct Thought |
| Opened card | Multiple developments selectable, nudge/request/step correlations grounded; missing evidence truthful |
| Navigation | Half-screen ~960 px, mobile ~390 px; keyboard selection and back-to-shelf preserve context |
| Safety | GET/read/refresh never runs Nemotron, posts an action, changes Identity grants or mutates Thought data |
| Development jump | **Not part of current P0**; mark P1 BLOCKED until exact-context endpoint/navigation is approved and proven |
| Event/role feed | **NOT IMPLEMENTED / NOT DEMONSTRATED** until real recorded events and role artifacts exist |

Use PASS / FAIL / BLOCKED / NOT RUN, with evidence and exact version references. No PASS claims from illustrative SVGs or isolated browser-only fixtures.

## 6. Decisions needed from the owner before converting this draft into a coding prompt

1. **Preview text:** For the small shelf card, is **title + last real output type + status + updated time** enough, or does the owner require a *preview of the Note's actual words*? The latter has a significant privacy/data-contract implication.
2. **Live presentation:** Should the first version show an **activity strip of real status/timestamp events** while recording remains limited, rather than imply streamed model thinking? Suggested: yes, label correctly.
3. **Opened view:** Side panel/drawer within the shelves versus full-page detail at half-width? Recommendation: preserve the current overlay, make it readable with focus and optional full-width behavior.

**Design-only approval needed for this document first.** Once the owner chooses, have an independent reviewer check real contracts, permissions and the P0 baseline, and then produce separate narrowly authorized implementation assignments. No silent expansion of Claude's P0.
