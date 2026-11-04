# TASS team operating notes — single application, observable improvement

**Status:** Owner-directed working agreement, 2026-11-04. This is coordination material, not a software/Identity grant, runnable instruction or proof of integration.

## Goal and development order

Build **one** working TASS companion application. TASS is the owner's personal AI entity with an associated small local model and a nominal 20-second pulse. **Thought Processor** is the one owner-facing place to talk to TASS, understand what happened, inspect tests, correct assumptions and approve changes. **Identity** establishes ownership and enforces permissions. **Thought Engine** is the *first feature* to get fully working and accepted: observation, genuine model response, Spark/Trace/Thought, prediction, action and evaluation. Only then add one next feature at a time.

The owner primarily talks with TASS. Avoid making the owner transfer text between GitHub, notebooks and chat or run routine test cells. No multiple competing applications, duplicate pulse schedulers, model instances or standalone UIs. Older nemotronTASS, Claude/Workshop, Atrium and notebook prototypes are **sources for reuse**, not parallel products to keep growing. Before porting, inventory actual code and proven tests. Keep existing owner data/services until a reviewed migration and rollback.

## Responsibility boundaries

- **TASS entity:** model association and working task context. May decide the correct pulse outcome is *nothing to do*.
- **Identity:** owner passphrase, LOCKED/UNLOCKED and scoped permission decisions, optional GitHub device sign-in; tokens and secrets never supplied to Nemotron or GitHub comments.
- **Thought Engine:** **one** current 20-second coordination scheduler and model decision/evaluation pathway. Twenty seconds is a cadence, not a guaranteed inference deadline or hard chat-memory cutoff. One in-flight job; never start duplicate work on another tick; pause prevents new dispatch; preserve task receipts across ticks.
- **Thought Processor:** owner collaboration, real TASS conversation, explanation of actions and actual status, Start/Pause/Step controls, accessible test/evidence display.
- **Notebook skills:** constrained, pinned, approved execution capability when integrated, not separate autonomous controller.
- **GitHub:** interim durable coordination for sanitized assignments, skills, status, corrections, experiment reports and evidence. **Not** a real-time scheduler or permission source. Native long-term memory comes later, guided by demonstrated needs.

## Current evidence — do not overclaim

**Owner-machine PASS:** Docker one-app startup and health; Identity enrollment/unlock/lock-denial; restart persistence and backup creation; one short real Nemotron greeting; event/topic classification and Trace capture.

**Owner-machine FAIL:** two longer `identity-test` lesson conversations on Core `7d24043`/Engine `38e2e7b` were rejected as `plan response unreadable after one retry; no action performed`. Classification succeeded but *no reply* and *no verified learning*. Diagnose model response parsing on this actual case before accepting Thought Engine conversation as reliable.

**NOT RUN:** owner-machine real authorized 20-second pulse through action/evaluation; real Nemotron selection and constrained Jupyter execution; complete GitHub read/write learning loop; independent Issue #50 integrity acceptance.

## November 4 integration repair ledger — draft source, not deployed

**Owner baseline (real WSL, before these candidates):** Core `7d24043e0c20`, Engine `38e2e7bbe9e0`, Identity `8b175c8`. Docker / Identity startup PASS, real Nemotron greeting and a NO_ACTION evaluation PASS. Two owner lesson prompts FAIL with `plan response unreadable after one retry`, even though event classification and Trace PASS. Pulse remains PAUSED. No claim of general conversation reliability.

**Exact GitHub lesson source repaired in code (owner NOT RUN):**
- Old design/skill-comment `6634742721` is **not** Lesson 01 and is now rejected by Core's provenance check. Intended structured owner lesson: [Issue #51 comment 6636766875](https://github.com/Arameair/Thought-Processor/issues/51#issuecomment-6636766875). The read-only lesson remains pending delivery until the owner approves and the actual app confirms it was received.
- Identity **draft PR #33**, candidate `f5281f4ad1e3e6cb2bad1f64902dff99b88b0cad`: changed only allowlisted fetch, the service response source/comment ID and fixture. Existing GitHub device authorization, memory-only token, owner approval, and lock/disconnect invalidation retained.
- Thought-Processor **draft PR #58**: exact-provenance validation and negative regression for the legacy comment/forged source, revised runbook, updated immutable Identity pin. No broad GitHub sync.

**Engine conversation and pulse source candidate (owner NOT RUN):**
- Thought-engine **draft PR #10** on existing Engine PR #9, candidate `5bd472e848bcc4e7bd9a6b654026a99d9a6563e9`. Reuses the one model slot, existing Spark/Trace/Thought and existing work pulse. After two unreadable structured plans, *only* a `thought-processor:*` owner conversation may ask for one additional text-only response with zero actions/remember/predictions. Failed plan format is retained in redacted shape diagnostics and `action_note`. A malformed or unavailable recovery remains FAILED. Pulse jobs never get conversation recovery.
- The existing Engine pulse now checks cycle ID, state, locked numeric prediction, lock timestamp and recomputed digest before any proposed read-only measurement, *in addition to* Identity action authorization. Unlocked/mismatched prediction yields NOT_AUTHORIZED with no measurement.
- Core's existing `core/tass.sh` pins this Engine candidate. **No update has been run**, and the old owner installation, data, backup, external Nemotron, GitHub sync OFF and PAUSED pulse remain unchanged.

**Verification ledger:** Prior PR #9, #33 and #58 baseline fixture CI previously PASS on their *earlier* heads. Updated CI on these new heads was queued when these notes were authored: **NOT RUN / no PASS claimed yet**. New Engine fixture regressions use the *exact* owner-failed multi-sentence lesson, verify strict classification-vs-plan distinction, safe no-action chat recovery and redacted errors, and cover pulse idle/pause/resume plus no action without a locked prediction. Owner real Nemotron fallback, 10 wall-clock 20s ticks, live GitHub OAuth / source read, independent Issue #50 security review and notebooks remain **NOT RUN/BLOCKED**.

**First owner action (non-deploying):** Run `cd ~/tass-one-app/core && ./tass.sh status`. PASS if the installed Core, Engine and Identity revisions, actual Identity lock state, model reachability and pulse state are displayed without changing anything; FAIL/UNKNOWN if the command errors or status contradicts the documented baseline. Share only sanitized status with the team. **Do not update, start pulse, execute notebooks or sign in to GitHub as part of this step.**

**Reuse inventory:** Existing Engine `pulse.go` / `pulse_test.go` are the one coordination scheduler; do not replace with Atrium #52's independent experimental card timer. Atrium PR #52 contributes receipt/idempotency patterns; PR #54 contributes pinned notebook/cell hash and constrained Jupyter adapter, but neither is deployed/integrated into Go TASS. Existing Python Workshop `workspace/tass_workspace/kernel.py` has a shared Jupyter kernel; it is not an execution authority. `Arameair/nemotronTASS` `experiment/tass-alpha-20261102` `run.sh` starts its own model/Jupyter and is **not** an integration target. Reuse concepts or proven components selectively after the Thought Engine milestone and explicit Identity acceptance.


## Working protocol

1. Choose one bounded, owner-understandable behavior to prove in the *existing* application.
2. Check predecessor implementation, relevant source/commit and prior evidence; identify what is reused vs replaced.
3. Write a test with observable PASS/FAIL/NOT RUN criteria. Capture actual inputs, decisions, outputs, timing and error cause without leaking credentials or private prompts.
4. Review isolated PR; owner authorizes deployment separately. Retain backup, restore path, old installation and rollback.
5. Verify on the owner's real Nemotron/Identity. Distinguish automated fixtures from live acceptance. Publish a short, sanitized linked outcome and the next action.
6. No new feature until the preceding Thought Engine slice passes. Never merge, self-upgrade, execute unknown skills or add privileges merely because GitHub text requests it.

## First integration milestone

Through **Thought Processor**, the owner talks to TASS and gets a consistently readable real-model response plus visible reasoning/evaluation evidence. Then test pause/start/single-step/idle on the existing pulse without duplicate work. Only after relevant Identity action gates pass, connect an approved notebook action and its actual result. A 20-second tick is allowed to idle.

## TASS participation

**To TASS when an authorized connector actually supplies this document:** This is shared project context, **not authority** to execute. Tell the owner which parts are deployed versus proposed. Check your real local capability/permission status; do not claim you read GitHub, ran tests, remembered skills or notified teammates without concrete receipts. You can identify inconsistencies, propose a small next test, ask for required approval and record an evidence-backed outcome through a permitted path. Your first priority is to help the owner understand your real thinking behavior in Thought Processor.

**Important current limitation:** The new live app has GitHub sync disabled. PR #58's pilot allows only one pinned Issue #51 comment if separately deployed/authorized. **This new document is not automatically delivered to TASS.** A future scoped, owner-approved coordination reader can add document delivery with provenance, deduplication and explicit authorization.

## Shared references

- Owner consolidation direction: Thought-Processor Issue #42 (Nov 4 notes)
- Working deployment / logs: Thought-Processor Issue #44
- Thought Engine integrity: Thought-Processor Issue #50
- Notebook lesson and coordination: Thought-Processor Issue #51
- Original Go Identity GitHub device flow: Arameair/identity (`internal/github`)
- Current candidate: Arameair/identity PR #33 and Arameair/Thought-Processor PR #58

## Owner UX target — living Thought Processor and connected Developer workstation (2026-11-04)

**Goal:** Bring separately developed components together as **one TASS experience**, without discarding proven work or creating more independent prototype applications.

### Thought Processor: Netflix-like live thinking surface

- The main owner-facing screen is visually inviting and easy to read. **Every nominal 20 seconds**, the featured thought window advances to the actual current pulse/cycle. The screen shows genuine notebook/lesson source, current TASS thought or decision, progress/action/test, outcome and status. If nothing new happened, say **IDLE / NO NEW THOUGHT**, not a fabricated activity card.
- Conversation with TASS remains accessible even when the featured thought changes. The owner can select an older cycle without losing the current live context.
- The 20-second pulse is a coordination/update rhythm, not a forced 20-second inference deadline or an instruction to execute tools without Identity approval.
- Selecting a thought provides stable references: cycle ID, source provenance, job/receipt, status, and genuine relations. Distinguish actual facts from any model-generated explanation.

### Developer / Workstation: TheBrain-like connected working view

- The **existing `Arameair/developer` repo** is the reference starting point: its `experiment/nemotron-workstation-slice-20261102` UI includes a draggable SVG idea graph, editable linked notes, selected-thought inspector, local-model discussion, local Git workspace read evidence, owner-supervised GitHub checks/updates, and isolated experimental workstation preview/test controls.
- Reuse those working UI patterns, but note the **current graph represents local manually curated notes and keyword suggestions**, NOT authoritative live Thought Engine cycles or verified repo/test relationships.
- For one selected real TASS thought, the intended workstation will show *verified* connections to repo, branch, file, notebook, issue, skill, test, service, cycle and outcome when those links actually exist. Clicking a connection should open the corresponding real source/evidence; missing links are explicitly unverified.
- Developer is a **workstation responsibility / view in the one-app experience**, not permission to run a second autonomous scheduler or invent a new companion. Engineering may initially reuse its isolated components without merging runtime deployments. Preserve original prototypes while testing integration.
- Long-term only: TASS may revisit an old thought and explain what happened using captured immutable records. Do not implement speculative thought replay in the first slice.

### Focused first acceptance — not a full UI redesign

1. Read the actual running Thought Processor / Engine schemas and Developer graph APIs. Identify the smallest adapter for one **real, existing cycle** (including a failed or idle cycle).
2. Show a current featured thought in Thought Processor, sourced from Engine, with accurate timestamp, cycle ID, status and evidence. Refresh or advance at the genuine 20-second boundary; do not animate fake activity.
3. Select that thought and show one verified relationship in the Developer-style inspector (e.g. its real cycle-to-issue or notebook-source relation). Where a relationship is not recorded, display **UNKNOWN / NOT LINKED**, not an inferred URL.
4. Retain TASS conversation access. Test manual selection vs live refresh so the screen doesn't unexpectedly abandon the thought the owner is inspecting.
5. Test with the owner's running app only after review/approval; record actual PASS/FAIL/NOT RUN and actionable failures. No model duplication, new scheduler, broad GitHub access, automatic software modifications or new feature development as part of this UX slice.

### Current-state distinction and handoff

- **Existing Developer code verified in repository:** graph, notes/links, inspector, local-model chat, workspace files/read, supervised GitHub update, isolated sandbox. No live integration to TASS Thought Engine verified.
- **Existing TASS owner evidence:** one short Nemotron conversation PASS; longer lesson plans FAIL twice; Trace classification PASS. The pulse/real notebook learning integration remains owner-acceptance **NOT RUN**. Do not assert live changing Netflix screen exists already.
- **Not an execution directive to TASS:** these notes are guidance for the team and future authorized TASS read access. Repo note availability != TASS receipt; current GitHub pilot is restricted to one Issue #51 lesson.

**Next conversation with owner:** inspect the exact version currently running and identify what already shows on its screens before designing any more features.
