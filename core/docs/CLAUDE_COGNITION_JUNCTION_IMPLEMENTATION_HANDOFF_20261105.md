# Claude implementation handoff — Go-first Cognition repair and first visible information junction

**Prepared:** 2026-11-05 (owner testing intended next session; no background work promised).  
**Status:** IMPLEMENTATION INSTRUCTIONS / OWNER REVIEW REQUIRED. This text is not a claim of execution, CI success on future commits, an accepted release, or deployment.  
**Primary owner design:** [cross-repo loop alignment](./CROSS_LOOP_ALIGNMENT_20261105.md) · [Cognition #82](https://github.com/Arameair/Thought-Processor/issues/82) · [release/recovery #79](https://github.com/Arameair/Thought-Processor/issues/79) · [evidence #80](https://github.com/Arameair/Thought-Processor/issues/80) · [TASS 20-second design #25](https://github.com/Arameair/TASS/issues/25).  
**Candidate review:** [Core PR #86](https://github.com/Arameair/Thought-Processor/pull/86), [Engine PR #15](https://github.com/Arameair/Thought-engine/pull/15), [earlier visual-only PR #84](https://github.com/Arameair/Thought-Processor/pull/84).  
**Cross-repo design branches:** [Thought-Processor PR #85](https://github.com/Arameair/Thought-Processor/pull/85), [Engine #16](https://github.com/Arameair/Thought-engine/pull/16), [Identity #39](https://github.com/Arameair/identity/pull/39), [Development #3](https://github.com/Arameair/Development/pull/3), [TASS #26](https://github.com/Arameair/TASS/pull/26).

## 0. Priority / scope lock

**The owner's goal is to BUILD observable circles incrementally, not to polish a dashboard that depicts nonexistent completed loops.** Each new junction or thought path must become observable at the time it is being wired, in Cognition, and testable in Development.

- There is **one Go-owned 20-SECOND controlling work/thought cadence**, not a 20-minute pulse. The old Team Liaison "20-minute pulse" design is not an owner requirement. Do not add any independent periodic Liaison service or scheduling loop.
- **No completed end-to-end thought circle has yet been owner-verified.** A stored Thought, a rendered graph edge, and a playable historical cycle are useful building blocks, not a verified living circle.
- Go owns the Engine pulse/coordination, state machine, typed information requests/results, validation, authorization enforcement, receipts, and safe evidence. Markdown holds **small described information cells** at loop junctions; it is *not* arbitrary executable code, a scheduler or security authority.
- Cognition is **read-only historical/near-live observation**. Playing a recorded timeline must never perform a model call, execute a cell/skill, approve a lesson, resume a pulse, alter a graph, or send a message.
- Development is the supervised experiment workspace. It may propose a modified cell and compare outcomes; it cannot automatically activate a proposed graph, grant permissions or deploy code.
- Identity remains the canonical identity and authorization authority. Shared feature definitions do not share entity histories, model contexts, credentials, Discord channels or grants.
- **Not this assignment:** automatic Talk/Discord posting, new TTS/voice, Team Liaison nudges, production autonomous behavior, generic Markdown/Jupyter/shell interpreter, five physically parallel models, new microservices, moving the canonical Thought repository or changing the owner's running stack without separate approval.
- Preserve the existing user-facing Netflix-like Cognition look, but fix anything it misrepresents. Keep the new UI light: Go backend + embedded HTML/vanilla JS.
- **Return real outcomes:** PASS / FAIL / NOT RUN, exact SHA, run ID, source ID, test evidence. Never turn a model claim of "DONE" into "VERIFIED."

## 1. Exact source snapshot and prerequisites

**Inspect latest branch heads again before writing**, because other contributors may be working.

| Component | Candidate currently reviewed | What it actually implements |
| --- | --- | --- |
| Core | [PR #86](https://github.com/Arameair/Thought-Processor/pull/86) branch \`experiment/cognition-go-first-gpt6-20261105\`; reviewed head \`fb53772be6f18c7a7ad2f504e239487278e25cef\` | Go Cognition library/details/timeline/export, read-only browser, /dev unexecuted reference |
| Engine | [PR #15](https://github.com/Arameair/Thought-engine/pull/15) branch \`experiment/cognition-journal-projection-20261105\`; reviewed head \`d69d3917d1601d806d4b2ba2254693e1adb92b6b\` | Go \`GET /api/cycles/{id}/journal\`, bounded sanitized \`cycles.jsonl\` projection |
| Unified Core baseline | \`experiment/tass-unified-integration-20261105\`, reviewed base \`913a4ff687054fb8e2dddf076ccaac992398394a\` | One running installation/update direction; do not overwrite ongoing work |
| Previously reported owner-running stack | Core \`60661aa807cac4d93ebc461e1447f3c1e6ad9473\`; Engine \`21b3d502775b3ed065ab445b88f51909500770b3\`; Identity \`87dd9243c721fd11701575033d95d3dcf286f765\` | Historical deployment report **only**. Re-query owner machine before claiming current |

CI evidence at review: Core PR #86 head \`fb53772...\` [Core tests run 42096638015](https://github.com/Arameair/Thought-Processor/actions/runs/42096638015) and [launcher run 42096637864](https://github.com/Arameair/Thought-Processor/actions/runs/42096637864) **success**; Engine PR #15 head [run 42096016070](https://github.com/Arameair/Thought-engine/actions/runs/42096016070) **success**. These are GitHub CI/fake fixtures, **not owner-machine acceptance**. Subsequent changes require their own new CI.

**Do not modify or deploy any of these accepted-owner runtime versions as a side effect of documentation changes.** Work on new, isolated Claude experimental branches from the **current candidate heads**; submit focused draft PRs to the appropriate candidate branch/PR owners. Preserve GPT-6's commits, older PR #84, and docs PR #85. Avoid resets, forced pushes, merges, changes to \`alpha\` or \`main\`, new ports, credentials, releases or background automation.

## 2. P0 — Correct actual Core Cognition defects (first, focused patch)

### P0-A: Fail-closed cycle ownership / multi-identity isolation

**Current source:** \`Thought-Processor/core/tass/cognition.go\`, \`cognitionAccess\` (~109), \`cognitionOwnedCycle\` (~142), \`CognitionLibrary\` (~148), \`CognitionThought\` (~265), \`CognitionTimeline\` (~338). \`core/engine/cognition.go\` defines Engine DTOs.

**Observed problem:** \`cognitionOwnedCycle\` allows a cycle where \`Cycle.EntityID == ""\` if the \`Cycle.Identity\` display-name matches the current Identity name. A name is not canonical identity evidence. A later second identity, rename, migration or collision could expose someone else's unscoped cycle in the library/timeline. Engine-local ID and canonical Identity ID are intentionally different; don't simply compare them as though identical.

**Required approach:**
1. Keep existing Identity \`UNLOCKED\` + per-read permission check and Engine bearer key. Inspect Identity's actual \`EntityID\` and the Engine entity mapping; do not weaken the current checks or invent a new grant.
2. For cycles with nonempty Engine \`EntityID\`, require exact match to the **uniquely resolved** authorized Engine entity. If mapping is ambiguous/unavailable, deny scoped Thought/cycle data.
3. For cycles without Engine entity ID, do **not** count a matching display name as sufficient. Default **UNSCOPED/NOT LINKED** and exclude from the authorized per-identity library/timeline until an explicitly owner-reviewed historical mapping exists. If compatibility demands surfacing historical unscoped items, do it in a distinct metadata-only restricted view after a tested authorization policy; never expose their journal by name alone.
4. Add regression cases for two Engine entities with the same display name, null/missing Engine ID, name change, foreign Engine entity ID, and a missing/invalid Thought. Verify fail-closed access on library, details, timeline and export, and that a locked/revoked Identity never calls Engine.
5. **No silent migration** and no rewriting old journal files. Record excluded historical cycles as \`UNSCOPED/NOT RECORDED\` where safe.

**Acceptance:** A cycle can be viewed under an identity only when the ownership mapping is verified; no cross-identity leakage by same-name heuristic.

### P0-B: Restore saved owner notes in Cognition

**Current source:** \`CognitionLibrary\` sets \`out.BrainRevision=c.Brain().Revision\`, later iterates \`c.BrainBlocks(nil)\`. But \`BrainBlocks(notes []BrainNote)\` includes notes only when passed.

**Required approach:**
1. Take one consistent saved graph snapshot \`g := c.Brain()\`, use \`g.Revision\` and \`c.BrainBlocks(g.Notes)\`. Do not read twice and accidentally combine different revisions.
2. Keep source/provenance explicitly \`Core saved brain note\`; notes are advisory, **not a persistent Thought or executed cell**.
3. Add test: saved note appears, correct revision, no note leak into another identity, absent notes correctly empty. Keep bounds and sanitization; private note contents must not accidentally appear in public unauthenticated routes or generalized exports.
4. Check whether only \`code-cell\` but not \`text-cell\` is copied from \`BrainBlocks\`; deliberate omission should be explicit in UI. If owner expects Markdown notebook cells visible, include appropriate read-only \`text-cell\` cards (no execution claim).

**Acceptance:** Owner's real saved notes and appropriate discovered cells appear under distinct card types with correct revision and status.

### P0-C: Correct evidence semantics and safety in the existing playback

Keep the strong work already in PRs #86/#15:
- \`Thought\` vs \`Trace\` vs \`Cycle\` are distinct. Only join when actual \`cy.Spark.ThoughtID\` equals an existing persistent Thought for the correct Engine entity, never by topic/title or guessed chronology.
- Engine \`cognition_journal.go\` reads a bounded 8 MiB tail, up to 96 journal entries per cycle, preserves actual \`cycles.jsonl\` byte offsets and UTC timestamps, reports truncation, partial/corrupt, duplicate and out-of-order records.
- Go allowlist projects **only** safe IDs/states/events; no raw prompt, model reasoning, auth details, private trace content or arbitrary JSONL. Core sanitizes a second time; exports are Markdown/JSON up to 64 KiB with no auto GitHub upload.
- A journal \`outcome.status=EXECUTED\` is still *executor-reported*, not independently verified without a correlated skill receipt. \`NO_ACTION\` is not execution. Original historical Identity decision remains **NOT RECORDED** until a verified read-only join exists.
- Play/Pause/Step/View Logs must **not** call write endpoints. Unavailable Engine/API should yield an honest visible failure, not fixture fallback.
- Be precise about recorded timestamps: a summary Cycle is not the stage timeline. A staged replay of records is **history**, not a live internal-thought transcript.
- Match expected stage gaps to actual state where possible. An in-flight \`awaiting_outcome\` cycle legitimately lacks later stages: label \`PENDING\` rather than suggesting journal corruption, while preserving missing-stage evidence.
- Tests should include a corrupted/truncated JSONL tail, absent cycle, large journal, duplicate timestamp/stage, no leaked fake secrets, no raw prompt in Copy or Download, and browser no-POST checks.
- The UI currently shows buttons to open related cycles and a **unexecuted** \`/dev?cognition_ref=...\` proposal; preserve these truthful semantics.

**Acceptance:** Source-based replay with real offsets, exact lineage when possible, clear UNKNOWN/PENDING/UNLINKED, and no side effects beyond authorized reads.

## 3. P1 — Show the active 20-second system truthfully (observation alongside work)

The owner intentionally built visibility **before** working full circles; do not declare completion of a loop. The current Cognition shelves focus on historical cycles; add a **small Go-backed observable cycle/pulse status strip**, not another dashboard or scheduler, only if it can be correctly derived from present Engine data.

**Relevant source:** Engine \`pulse.go\` \`Pulse.Tick\`, \`GET /api/pulse\`, \`ticks.jsonl\`; Core already has authenticated \`engine.Client.Pulse()\` / \`PulseStatus()\` and \`core/tass/core.go\` pulse controls. Distinguish *work-coordination Tick* from independent model-health checks and GitHub sync.

**Minimum read-only view:**
- Actual \`RUNNING | PAUSED | UNKNOWN\`, actual configured \`every_s\` (target 20 seconds, don't hardcode 20 for a different observed installation), last actual \`pulse_id\` and timestamp, last \`READY/WORKING/BLOCKED/BUSY\`, active \`job_id\`, and real pending count if exposed.
- Show \`NOT RECORDED\` for unavailable receipt IDs, and \`PAUSED\` if the owner's stack hasn't started the pulse. A page refresh does **not** start a pulse.
- If wiring into Cognition via new read-only Core endpoint, enforce same token, same Identity checks, no-store, fixed bounds and typed projection. Do not expose full raw job descriptions, secrets or unrestricted logs.
- Test paused/idle, working, blocked, restart, missing Engine, and refresh creates **no** new tick, cycle, authorization for an action, skill request or model generation.
- Keep this small and separable; if it threatens P0 repair and safe evidence acceptance, write an honest follow-up issue instead of faking it.

**Acceptance:** We can open Cognition and see the *real* current/past coordination state while building the loops. No fake “20-second cycle happened” claim when stopped/paused.

## 4. P2 — First *single* Markdown information junction (independent controlled slice)

This is **the next proof of the architecture**, NOT a mandate to build arbitrary Markdown execution or entire autonomous cognition. Implement after P0 and evidence test surfaces exist, or deliver a separate PR with clear NOT RUN status if insufficient readiness.

**Goal:** During an existing Go-controlled thought/task path, an authorized information need is recorded; a short version-pinned Markdown-described cell supplies lesson metadata through a **registered read-only Go handler**; a durable receipt is correlated; an eligible later 20-second cycle can consume the result or record why it could not. Cognition displays it. Development proposes an alternative cell without activating it.

**Important distinction:** the existing \`@lesson-workspace.list-lessons\` compiled skill in \`core/tass/skills.go\` already requires **Identity grant and explicit owner Approve for every execution**. Do not silently reuse that handler through an automatic tick or reinterpret read-only as permission. Either (a) use its existing supervised request→approval→receipt sequence for the first demonstrator, or (b) define a distinctly authorized \`junction.read.lesson-metadata\` Go capability/permission path and seek owner approval for it **before any automatic execution**. If no grant/interface exists, implement a **dry-run only** and record \`AWAITING_APPROVAL\`; do not fake data.

**Conservative proposed contract (schema may be adapted after inspecting actual Go structs):**

\`\`\`text
JunctionRequest
  junction_id / idempotency_key
  entity_id (canonical Identity ID) and engine_entity_id if resolved
  cycle_id / pulse_id / job_id / thought_id ONLY when real and recorded
  cell_id + pinned source SHA / revision
  requested capability: READ_LESSON_METADATA
  typed input: lesson ID/path (validated, no traversal)
  deadline (bounded), reason, originating event ID

JunctionReceipt
  junction_id + request/idempotency key
  status: DATA | AWAITING_APPROVAL | DENIED | UNAVAILABLE | TIMEOUT | FAILED
  source revision + recorded UTC time
  safe typed summary / data, or explicit reason
  Identity decision ID if authorized and safe to expose
  associated cycle/pulse/job IDs, provenance, consumed_at if actually used
\`\`\`

**Go-first mechanics:**
1. Define exactly one small, explicit Markdown cell document (e.g. a versioned lesson-lookup cell) with purpose, expected inputs/output, and ID. No dynamic Go/shell/Python inside Markdown. Avoid inventing a sprawling DSL or “tool orchestration engine.”
2. Use an **allowlisted compiled Go handler** in existing Core or Engine, aligned with current ownership. Validate resource scope, path, content length, timeout, version and authorization. Caller does not decide its own Identity permission.
3. Capture an information request **without blocking \`Pulse.Tick\`** on long I/O or human approval. Persist/queue using existing durable job/receipt patterns where appropriate; no additional periodic scheduler. If work needs human approval, record pending state and let Tick continue.
4. On a later 20-second cycle, check the durable receipt and consume only if the originating task/intent and cell version are still current. Prevent duplicate requests or repeated effects after restart, retry and unknown-delivery responses. Use no side effects for a read-only pilot.
5. Persist the correlation and safe evidence in an Engine/Core-owned Go store, with a defined single owner of each record (no two authoritative journals). Add *read-only* Cognition projection and a Development link so the owner can see requested → authorized? → returned? → consumed? → subsequent actual outcome.
6. **No invented “effect on decisions.”** A difference is VERIFIED only when you can correlate an actual later cycle/prediction/outcome, with same intended task and snapshot; if not, label \`EFFECT NOT MEASURED\`.
7. Distinguish \`PASSED STRUCTURE\`, \`EXECUTED HANDLER\`, \`OBSERVED RESULT\`, \`OWNER ACCEPTED\`. A candidate Markdown file sitting in a graph is \`VISIBLE\`, not \`EXECUTABLE\`.

**Minimal tests:** DATA happy path; missing file; stale SHA; invalid/traversal ID; Identity DENIED/LOCKED/revoked/unavailable; owner approval pending; timeout; same ID retry; interrupted/persisted job; duplicate-safe consumption; pulse remains responsive; no new model call on a Cognition read; no cross-entity leak; actual exact identifiers under Cognition.

**Completion gate:** One owner-reviewed junction appears with real timestamps/status in Cognition, and a real later cycle shows consumption or “NOT CONSUMED” truthfully. If real pulse cannot be run safely yet, return the Go tests and a separately marked \`OWNER TEST NOT RUN\` candidate. Do not secretly unpause live pulse.

## 5. P3 — Development testing and Liaison future boundary

**Development now** (Core \`/dev\`, \`core/tass/brain.go\`):
- Keep \`BrainGraph\` revisions and \`BrainEdge\` \`next/supports/requires/uses\` as **advisory** until a registered authorized Go operation actually runs. Validate stale cells and rejected graph changes server-side.
- Cognition's \`Try adjustment\` currently passes only \`cognition_ref\` and \`brain_revision\` and presents an **unexecuted proposal**. Preserve it; do not quietly mutate the saved graph.
- Add a *candidate experiment note*, a pinned cell revision and a comparison record only after owner review. Record baseline and candidate receipts, model and build pins, observed outcomes and failure paths. No fabricated A/B claim from unpaired data.
- Publish an owner-approved, versioned **feature definition** only after independent tests and rollback path; each later identity separately adopts it under Identity grants. Do not globally activate anything.

**Team Liaison later**: A feature at an information/communication junction, **not another boss/scheduler**. A 20-second Tick can consult *durable communication state* without sending a message each Tick. Separate DISPATCH → WORKING acknowledgment → DONE/BLOCKED claim → VERIFIED evidence, idempotent handoff ID and no blind resends. A status check alone must not cause nudging. Discord will be a future **text** destination, owner-authorized per identity/channel, not voice. **Do not implement this in tonight's P0/P1/P2 PRs.**

## 6. Release/integration protection — #79 is not waived by green CI

1. Keep work on experimental branches and produce draft PRs targeted to candidate branches. Only an explicit owner-approved integration process may pin both Core and Engine candidates together.
2. PR #86 requires Engine PR #15's new journal endpoint; a Core-only update against old Engine is not a complete integration. Check the actual SHA/image manifest in the release candidate. Identity pin and credential custody remain unchanged unless independently approved.
3. The existing \`core/tass.sh\` exposes \`status\`, \`check-update\`, \`update\`, \`rollback\`, but **presence of these commands is not proof that automatic backup/Restore Previous fully satisfies #79**. Review #79's interrupted-update/backup-integrity/data-schema tests before proposing an owner update.
4. Preserve exactly one live owner stack and all existing lessons/memory/Identity state. No resetting data, force pushes, merging \`alpha/main/live\`, changing pulse from PAUSED, enabling autonomous, inventing permission grants, service/port conflicts, or installing new models.
5. CI for final candidate should include \`go vet ./...\`, \`go test -race ./...\`, Core browser fixture, Engine journal fixture, auth negative tests, Go/Compose build and relevant script syntax checks. Document each exact run URL/result and mark untested cases \`NOT RUN\`.

## 7. Owner session checklist — next morning

**Safe read-only inventory first** from the existing owner checkout:

\`\`\`bash
cd ~/tass-one-app/core
./tass.sh status
git rev-parse HEAD
git status --short
\`\`\`

Compare installed Core/Engine/Identity versions to the **candidate's full SHAs**, not truncated names; retrieve the current manifest via existing status if available. Verify the new draft PRs and their CI. Only then, with explicit owner approval and a verified safe update/restore path, plan a staged installation of the reviewed Core+Engine pair. **Do not use \`./tass.sh update\` merely because the CLI command exists**.

After authorized update/restore readiness:
1. Confirm \`READY\`, Identity \`UNLOCKED\`, owner-selected chat-only/autonomous/pulse state unchanged. Do **not** start the pulse automatically.
2. Open the real \`/cognition\`. Confirm persisted Thoughts vs recent unlinked Cycles, actual Engine SHA and true empty state where no Thoughts exist.
3. Select a real cycle; Play/Pause/Step and View Logs; compare cycle ID, thought/trace linkage, stage timestamps and offsets with local Engine \`cycles.jsonl\` through the authorized host, not through an exposed web API.
4. Test Copy Evidence / download sanitized Markdown/JSON; verify no private text, tokens or local paths; check actual copied contents and no new model/skill/journal cycle caused by viewing.
5. Open \`Try adjustment\` → Development. Confirm “UNEXECUTED PROPOSAL,” no graph save, no lesson approval, no new model action.
6. If the supervised first-junction pilot exists and owner explicitly approves its designated permissions/test, try DATA, DENIED/UNAVAILABLE, and next-cycle continuation. No 20-minute scheduler and no circular wait. Record exact receipt IDs and action statuses; if missing, mark \`NOT RUN\`.
7. Ensure current Core+Engine+Identity still healthy and user data preserved. Where rollback is requested or failure occurs, use the separately tested #79 process, and recheck exact SHAs and data.

**Test report format:**

\`\`\`text
Owner acceptance — date/time and WSL host
Core SHA: ...
Engine SHA: ...
Identity SHA: ...
PRs and CI runs: ...
Pulse status/configured seconds: ... (20 seconds is intended; running status is observed)
Thought ID / cycle ID / source offset: ...
Cognition library: PASS | FAIL | NOT RUN
Recorded timeline + Play/Pause/Step: PASS | FAIL | NOT RUN
Identity fail-closed/cross-identity: PASS | FAIL | NOT RUN
Copy/Download redaction: PASS | FAIL | NOT RUN
Development reference non-execution: PASS | FAIL | NOT RUN
First junction request/receipt/consumption: PASS | FAIL | NOT RUN
Backup/update/restore (#79): PASS | FAIL | NOT RUN
Blockers + reproductions: ...
Owner approval for promotion: YES/NO (default NO)
\`\`\`

## 8. Deliverables Claude must leave for handoff

- **Focused draft PRs** for P0 fixes (Core; Engine only if a real Engine defect), with exact base/head SHAs, test evidence and code review.
- **If feasible and separately reviewable:** P1 real Go pulse-status projection and P2 one read-only information junction, each with actual receipts/tests. Never say "all circles working" when only one segment works. If blocked by missing permissions/current acceptance, leave a detailed design+tests and mark runtime \`NOT RUN\`, not a fake pass.
- Update PR #86, Engine PR #15, Cognition issue #82, release #79 and Development issue/PR with links, blockers, and what the owner must click/run. Keep collaborators on one shared handoff.
- Final summary limited to: CHANGED / ACTUALLY TESTED / NOT TESTED / OWNER NEXT ACTION, with explicit DONE, BLOCKED or HANDOFF.

**Success for the next owner session:** Cognition honestly displays real Thought/Trace/Cycle history; playback and sanitized evidence work; no cross-identity leak; the interface remains read-only; and the first small junction, if approved and implemented, can be observed as an actual Go exchange. **Future Talk, Discord, Liaison automation and globally shared feature activation remain separate work.**
