# Foundation remaining — leave TASS operating, teach via GitHub and Development, verify learning

**Date:** 2026-11-05
**Status:** Owner direction and remaining engineering/acceptance gates. Design/coordination ONLY; not deployment approval, not proof that a complete thought circle or learning loop works, and not permission to turn on autonomous behavior.
**Target:** First owner-supervised testing beginning 2026-11-06, after owner explicitly approves any changes.

Read first: [Cross-loop alignment](./CROSS_LOOP_ALIGNMENT_20261105.md), [Claude implementation handoff](./CLAUDE_COGNITION_JUNCTION_IMPLEMENTATION_HANDOFF_20261105.md), [Cognition #82](https://github.com/Arameair/Thought-Processor/issues/82), [release/recovery #79](https://github.com/Arameair/Thought-Processor/issues/79), [TASS 20-second vision #25](https://github.com/Arameair/TASS/issues/25).

## 1. What the owner wants

Once the local Go foundation is safe, the owner wants to leave TASS's **local** thought process running (with bounded autonomy and real observability). Teaching enters through two channels:

1. **AI collaborator → GitHub → TASS.** GPT-6 or Claude, when invoked in an authorized session, posts a bounded message, lesson proposal or correction through the existing supervised Go GitHub mailbox. The TASS Core processes it and produces a real correlated reply or owner-reviewable proposal. A comment alone never authorizes learning, skill use, approval or code execution.
2. **Owner → existing Development platform → TASS.** The owner selects a pinned lesson/cell, asks for study/test, sees actual Nemotron answers and grades, approves or rejects, and can test a learned capability. Cognition displays actual receipts and source IDs.

**Both teaching entrances must converge on ONE TASS identity, one catalog of authorized lesson revisions, one durable study/approval record, one skill registry and one truthful Cognition evidence view.** There must not be separate GitHub and Development versions of TASS's mind or conflicting learn/permission states.

**External AI caveat:** An ongoing local TASS service does not keep GPT-6/Claude continuously active in the background. I can interact with GitHub when invoked through an active authorized chat and connector; GitHub comments do not, by themselves, reawaken an external AI conversation. Future scheduled external teaching requires its own explicit, supported and owner-approved trigger. Do not promise an always-on ChatGPT merely because the Go relay polls.

**20 seconds is the controlling local design horizon, never 20 minutes.** No second Liaison scheduler. A pulse may record NO_ACTION; it does not automatically need a model invocation or a GitHub message each time. Five dynamic Markdown cards forming a logical batch remain [TASS #25](https://github.com/Arameair/TASS/issues/25) **owner design intent**, not yet a proven always-running complete circle.

**Important correction on actual current code:** Claude PR #87 connects junction processing to the existing Go Core 20-second **SyncOnce** synchronization tick, while Thought Engine has its own 20-second **Pulse.Tick** job/work coordinator. Those are distinct existing schedules, not one proven continuous thought circle. A successful SyncOnce junction lookup and a later conversation do not, by themselves, prove that the Engine thought pulse autonomously discovered, routed and learned from it. Preserve the distinction in Cognition labels and testing; deliberately integrate/correlate them without inventing a third scheduler.

## 2. Current implementation evidence (as checked Nov 5)

- [Core draft PR #87](https://github.com/Arameair/Thought-Processor/pull/87) stacks on [Core #86](https://github.com/Arameair/Thought-Processor/pull/86). Reviewed PR head: 1ec138de22acfe63147e24ff7587ec3aeb4cda22. Its tested code commit: d59e84e9986897ecbeec6f89da4e20b452936c63. **P0** explicit owner-linked canonical Identity→Engine entity, same-revision owner notes, pending-stage evidence. **P1** read-only Engine pulse state strip. **P2** one Markdown-described lesson-metadata request handled by fixed Go, gated by Identity, durable idempotent receipt and next-eligible-conversation consumption. **P3** a Development Junction lab for proposing and comparing alternatives without activating them.
- [Engine draft PR #15](https://github.com/Arameair/Thought-engine/pull/15), candidate d69d3917d1601d806d4b2ba2254693e1adb92b6b, provides bounded authenticated read-only recorded cycle journal. Engine's own entity selection by name remains an unresolved deeper boundary.
- [Identity draft PR #40](https://github.com/Arameair/identity/pull/40), candidate 7da7f7c5bbecc42d3a7e9316a2037243eda10b88, adds separate owner-granted permission junction.read:lesson-metadata. It **does not** authorize any skill.run or arbitrary Markdown/Jupyter execution.
- Claude reports [Core CI 42101714521](https://github.com/Arameair/Thought-Processor/actions/runs/42101714521) and [launcher CI 42101714159](https://github.com/Arameair/Thought-Processor/actions/runs/42101714159) successful at the code commit, and 11/11 Chrome tests with **real Go Core, Engine and Identity executables but a fixture model/fake GitHub**. Real Nemotron on owner's machine, Docker paired update, full restore and long-running owner test: **NOT RUN**.
- Owner previously verified the deployed /dev graph in [Core issue #78](https://github.com/Arameair/Thought-Processor/issues/78), but observed validation issues: editor accepted an unsaved invalid 'uses' target and a directed 'next' cycle. Do not equate PR #87's Junction lab with completed Development validation.

## 3. Outstanding foundation gates, in priority order

| Gate | Required proof | Main work items / status |
| --- | --- | --- |
| **A. Safe integrated update and restore** | One source-pinned Core+Engine+Identity build, verified automatic backup and integrity, rollback without corrupting existing identity/lessons, failure and interruption testing | [Core #79](https://github.com/Arameair/Thought-Processor/issues/79) open; BLOCKS owner deployment |
| **B. Real Cognition visibility** | Owner links the exact Engine entity, opens actual persistent Thought and unlinked cycle history, steps through recorded journal, copies safe evidence, sees real pulse state and notes | [Core #87](https://github.com/Arameair/Thought-Processor/pull/87), [Engine #15](https://github.com/Arameair/Thought-engine/pull/15); fixture PASS, owner NOT RUN |
| **C. Local 20-second continuity** | Supervised sustained real Engine pulses/NO_ACTION with pending work surviving restart, no queue runaway, no unauthorized actions, no confusion with Core SyncOnce; measure model resource use | [TASS #25](https://github.com/Arameair/TASS/issues/25), [Core #70](https://github.com/Arameair/Thought-Processor/issues/70); dynamic five-card batch unproven |
| **D. First information junction with Nemotron** | Real local model produces or owner requests a need, Identity grants a fixed handler, receipt records result, later real cycle consumes it; Cognition shows SOURCE and EFFECT NOT MEASURED | Core #87; fixture PASS, owner Nemotron NOT RUN |
| **E. AI ↔ GitHub ↔ TASS communication** | One bounded manual comment traverses authenticated intake, Identity, real local Engine cycle, outgoing reply comment ID; restart/poll does not duplicate, GitHub outage does not stall thought | [mailbox #76](https://github.com/Arameair/Thought-Processor/issues/76), [candidate Core PR #81](https://github.com/Arameair/Thought-Processor/pull/81), [mailbox TASS #24](https://github.com/Arameair/TASS/issues/24); not owner-verified |
| **F. Owner teaching and verified learning** | One Markdown lesson receives real Nemotron study, valid grade, explicit owner approval and a **fresh-task retest**, with state preserved across restart. If a skill: separate approved registered Go execution receipt | [duplicate job #71](https://github.com/Arameair/Thought-Processor/issues/71), [@skill #72](https://github.com/Arameair/Thought-Processor/issues/72), [restricted test #60](https://github.com/Arameair/Thought-Processor/issues/60); end-to-end owner acceptance pending |
| **G. Multi-identity security and reusable features** | No cross-identity data/permissions from names; explicit feature grants, safe revocation, no reuse of TASS secrets. Address Engine internal name matching, not just Cognition overlay | [Identity #30](https://github.com/Arameair/identity/issues/30), Identity PR #40; separate future gate |

**A successful P2 information lookup is not equivalent to Gate F.** It demonstrates information transfer to a later conversation, not teaching retention, independent use, or autonomous thought-loop correctness.

## 4. GitHub teaching entrance — keep one real mailbox

Use [Arameair/TASS issue #24](https://github.com/Arameair/TASS/issues/24), labelled 'tass-core', for the pilot specified in [Core #76](https://github.com/Arameair/Thought-Processor/issues/76). The separate historical Identity Liaison channel (#20, label 'tass') must not be used as the same Go mailbox.

[Core PR #81](https://github.com/Arameair/Thought-Processor/pull/81) is an OFF-by-default Go Core relay candidate, **not integrated into PR #87**. It allowlists a GitHub account, validates the 'ChatGPT:' prefix as a routing hint (not authorization), stores exact inbound comment IDs, refuses self-replies, reports actual posted-comment IDs and avoids blind resend on uncertain delivery. Preserve this stricter conversational-only scope; older PR #69 envisioned wider GitHub study commands, which must not automatically bypass the owner-review gate.

**First manual acceptance:** one harmless authorized comment such as "ChatGPT: MAILBOX-01 — Please acknowledge receipt" → exact GitHub issue/comment ID → Core intake ID → current Identity decision → Engine cycle ID → TASS's actual posted reply ID. Then repeat poll and restart to check no duplicates. A LOCKED Identity, foreign author, GitHub outage, wrong issue/label and uncertain outbound POST must produce clear safe hold/error, not fake success.

Only after that proof should a GitHub lesson **proposal** be accepted as candidate learning material in Development. No GitHub comment may approve, silently execute a lesson, call an @skill, grant permission or alter live mind files.

## 5. Local owner teaching entrance — complete one learning proof

Pick **one real approved Markdown lesson** already within the TASS lesson catalog, at a pinned Git source SHA. Keep it small and testable. Proposed stages (these are distinct evidence fields, not necessarily new top-level status strings in existing code):

1. **DELIVERED:** source, teacher, exact blob revision and expected checks exist; the source is visible to owner in Development. GitHub comment vs lesson file must not be conflated.
2. **STUDIED:** one supervised real Nemotron job runs with a globally unique durable attempt/job ID; display the actual answer and cycle. Resolve historical duplicate Engine job HTTP 409 [#71](https://github.com/Arameair/Thought-Processor/issues/71) by reconciling the prior job and allocating a new safe attempt if appropriate. Never clear prior failures.
3. **PASSED or RETRY/STUCK:** Core grades genuine answers against actual checks. An absent/format-invalid answer cannot be graded as passed. Preserve original attempt and limited retries; owner handles stuck lessons.
4. **OWNER APPROVED:** the owner explicitly approves one successful specific lesson revision in the existing local learning UI. Approval of **knowledge** is not authorization for runnable skills.
5. **RETESTED on fresh task:** after another conversation/restart, TASS responds to a *different* relevant prompt using the approved knowledge, with a new observation/grade. Showing the lesson text immediately after approval is NOT independent transfer.
6. **SKILL USABLE (only if relevant):** a **separate** registered fixed Go operation such as @lesson-workspace.list-lessons is invoked through Identity grant and owner approval with a real executor receipt; repeated invocation after restart is tested. Do not run arbitrary notebook Python. No skill grant is inferred from junction.read or learned text.
7. **CORRELATED:** Cognition safely links source SHA, study job, actual Cycle/Thought/Trace IDs when recorded, grade, owner approval, later retest and skill receipt. Missing correlations are labelled NOT RECORDED, not guessed.

**Learning success:** owner can teach by Development, TASS passes a real graded attempt, owner approves, and TASS passes a **new** test demonstrating application. Separately, a manually supplied GPT-6 GitHub message must produce a real TASS reply. Only then consider merging the two teaching entrances into a single repeatable supervised curriculum.

## 6. Unattended-readiness requirements

Before the owner leaves TASS on, supervise a bounded multi-cycle run and record:

- Engine configured pulse interval and RUNNING/PAUSED state **separate** from Core SyncOnce tick and any GitHub backoff; no 20-minute scheduler, no mandatory outward chatter/model calls every tick.
- NO_ACTION and pending states are legitimate; model busy and GitHub unavailable do not spawn another Nemotron process or freeze local Go control.
- Queue size, VRAM/CPU use, model request time, duplicate work and memory over actual owner hardware; five logical Markdown cards do not prove five GPU-parallel inferences.
- Restart preserves authorized Identity link, pending/consumed junctions, lesson attempt and approval history, brain revisions and GitHub message cursor. Unknown GitHub POST result never blindly resends.
- Identity LOCKED/revoked blocks access, and a GitHub commenter is never granted permission to execute skills. No tokens, passwords, notebook code or raw model prompts enter exported logs.
- Automatic backup + Restore Previous Update is proven on the current integrated candidate, not assumed from the presence of a rollback command.
- Control remains owner-readable: STOP/PAUSE/RESUME and last real timestamps/receipt. No automatic activation of Autonomous or outbound Team Liaison/Discord.

## 7. Next owner test session order

1. **Read-only preflight:** from the owner's existing ~/tass-one-app/core checkout run the existing ./tass.sh status, git rev-parse HEAD, git status --short; note actual three component SHAs and Identity/Pulse state. Do not update just because Claude's fixture passed.
2. **Gate #79:** owner reviews an exact integrated Core+Engine+Identity candidate and independently verified recovery path. New branches/PRs must not auto-deploy or alter current lessons/settings.
3. **Cognition and junction:** follow [Claude owner acceptance](https://github.com/Arameair/Thought-Processor/blob/experiment/claude-cognition-repair-junction-20261105/core/docs/OWNER_ACCEPTANCE_COGNITION_JUNCTION_20261106.md). Link Identity to Engine only after verifying the correct entity. Check real timeline and zero playback-side effects. Only grant junction.read:lesson-metadata with explicit owner choice. Test a real lesson metadata receipt and later conversation consumption. Do not start Engine pulse automatically for the display.
4. **Mailbox, separate integration:** reconcile Core #81 on top of Core #87, after safe review, and prove one manual ChatGPT→TASS→GitHub comment round trip before any recurring automation.
5. **Lesson and skill:** fix/verify #71 collision, supervise one actual study/grade/approve, perform a fresh-task retest; then one separately authorized fixed @skill if relevant. Both teaching paths must show the same canonical lesson state.
6. **Leave-on trial:** only after above, owner explicitly starts a supervised bounded series of true Engine 20-second pulses with real Nemotron, watches Cognition, verifies offline/idle/restart behavior and safe STOP/PAUSE before choosing unattended settings.

## 8. Required contributor handoff format

FOUNDATION STATUS — UTC date / repository branch / full SHA
Gate being tested:
Existing owner-running versions (if actually read from owner host):
Changed code, exact file names:
CI/fake-model receipts and PASS/FAIL:
Live local Nemotron/Identity/Engine receipts and PASS/FAIL/NOT RUN:
Engine Pulse.Tick timing/state versus Core SyncOnce timing/state:
GitHub inbound comment ID / Core intake ID / cycle ID / outbound comment ID:
Lesson Git SHA / attempt ID / actual grade / owner approval / fresh-task retest:
Registered @skill ID / Identity decision / execution receipt:
Cognition correlation and sanitized evidence:
Data integrity/backup/recovery results:
Current blocker and owner next one action:

**Never call this a complete learning foundation merely because the first junction has a DATA receipt. The agreed win is one real supervised lesson that works on a fresh task, plus one verified manual GPT-6 GitHub round trip, all observed in the same TASS Core/Cognition identity.**
