# P0 supervised Markdown lesson repair — November 5, 2026

**Scope:** One existing TASS Go application. Owner machine is untouched. No new model, scheduler, GitHub issue relay, notebook executor or Identity grants.

## Observed owner evidence

- Core PR #66 source at `6ee10620e0f6d532b53fb20f73e0dcbdb8cf148e`, Engine `d8208409457fa6d0bdddfc80eeec728f5a5d6f21`, Identity `91e1521659a31db79044a44bd33bce48178554d9` are **source candidates identified in reports**, not a fresh direct read of the owner's installed deployment.
- Existing GitHub lesson discovery: **PASS**. `Arameair/TASS@live` had three Markdown lessons and one notebook marked NEEDS_WORKSPACE.
- Manual Study of `lessons/claude/01-my-work-pulse.md`: **FAIL**. Job `lesson_ef303a18a_1` emitted `plan response unreadable after one retry; first bytes=114 labels=REMEMBER; retry bytes=114 labels=REMEMBER`. **NOT GRADED** (the UI's 0/3 was misleading). No owner approval.
- Ordinary Nemotron no-action chat: **PASS** on prior owner build, not yet retested on this candidate.
- UI previously collapsed open answer during 1s redraw: **FAIL**, fixed as separate PR #67 source candidate, owner retest outstanding.

## Source-rooted failure explanation

Core `manual_learning.go` asked for `ANSWER n:` inside `REPLY:` plus `REMEMBER` and `ACTION: NONE`. The Engine `cycles.go` general action-plan parser rejected any response without nonempty `REPLY:` even though a numbered-answer quiz does not need an action proposal. Real model output included only a recognized `REMEMBER` label in each 114-byte attempt. **We cannot infer from that evidence alone why the model selected REMEMBER**; no request-ID-correlated raw completion or exact context-window trace was provided. The validated source contract mismatch and Core's unconditional 0/3 display are fixable without changing the executable-action parser.

## Code repair

- Thought-engine **draft PR #13** `experiment/p0-supervised-lesson-contract-20261105`, pinned at Engine `21b3d502775b3ed065ab445b88f51909500770b3`. Existing `Job.LessonChecks` forwards only numbered check IDs to `CycleRequest.LessonChecks`. Only `Source=pulse:lesson` and **zero actions** may enter a dedicated instruction/parser requiring exactly `ANSWER n:` for every checked step. The system prompt precedes untrusted reference Markdown, thinking is OFF, temperature zero, token cap 768, a single retry is bounded and logged by request ID. No answer keys are sent to Engine. All malformed responses remain FAILED/not graded; no actions or memory markers are extracted.
- Core **draft PR #68**, based on the existing UI fix PR #67, forwards the check numbers from Markdown, uses the small lesson-only context (not the entire entity document), retains the existing manual pulse Step, and never executes Python notebooks.
- Core `collect`: Engine FAILED is `MODEL_FAILED` with `NOT GRADED`, not an incorrect-answer result. DONE replies use the existing `lessons.Grade` evaluator. Only fully correct answers yield PASSED, always pending an explicit owner approval; **no automatic LEARNED** even if an older Autonomous setting is ON.
- Core `ApproveLesson`: checks Identity `lesson.learn` at point of use; rejects any non-PASSED/incomplete result. Writes `## Learned` to the existing local entity document before setting local lesson status LEARNED; duplicate approval is rejected. New learning becomes available through existing `learnedForContext` and local state is read on restart.
- UI reuses PR #67's opened `<details>` retention, skips replacing Learning DOM entirely when the lesson data is unchanged, shows Model/Engine failure as NOT GRADED, and supports copying full response/error text. Find and Study still use existing `/api/lessons/discover` and `/api/lessons/study`.

## Evidence and security checks

Engine fixture regression: actual 114-byte REMEMBER-only response is malformed/ungraded; realistic numbered 3-answer output traverses Engine and remains NO_ACTION; wrong or duplicate labels reject; model retry bounded; action-bearing lesson mode rejected; manual pulse Step leaves repeating timer PAUSED; duplicate job rejected. Core fixture regression: a failed job is MODEL_FAILED (not 0/3 wrong), genuinely incorrect 3-answer results cannot be approved, correct results reach PASSED but not LEARNED, locking Identity prevents approval, explicit approval writes and persists the local learned record and effective context after restart. Existing fixture CI is **not** owner real Nemotron acceptance.

**CI receipts:** See Engine PR #13 and Core PR #68 checks for exact, final commit results. Do not describe a previous commit's green CI as proof of a newer commit. Do not merge PRs or deploy automatically. Identity source is unchanged.

## Owner procedure — ONE step at a time

**A — read-only preflight (safe now):**

```bash
cd ~/tass-one-app/core
./tass.sh status
git fetch origin experiment/p0-supervised-lesson-acceptance-20261105:refs/remotes/origin/experiment/p0-supervised-lesson-acceptance-20261105
./tass.sh check-update origin/experiment/p0-supervised-lesson-acceptance-20261105
```

Record the **actual installed** Core/Engine/Identity revisions, the full **candidate** Core SHA, and confirm exact Engine pin `21b3d502775b3ed065ab445b88f51909500770b3` and Identity pin `91e1521659a31db79044a44bd33bce48178554d9`. Check same external model, ports, health and Autonomous OFF; do not assume installed source equals PR #66. Stop on any mismatch.

**B — only after explicit owner authorization of exact final Core SHA and verified CI:**

```bash
cd ~/tass-one-app/core
./tass.sh backup
sudo -v
./tass.sh update REVIEWED_FULL_CORE_SHA
./tass.sh status
```

Preserve current data and external model; the updater should snapshot and retain rollback. After restart, unlock existing Identity locally (do not recreate TASS). If update validation fails, stop and preserve logs; use `./tass.sh rollback` only after inspecting the current deployment state.

**C — one owner acceptance:** Autonomous OFF; open existing Thought Processor; Find lessons; select real `My work pulse`; Study (manual) once. Observe actual three ANSWER lines, grade, job/cycle IDs, Identity decisions and PASSED awaiting approval. Only if all correct, explicitly click Approve and verify `## Learned`, restart persistence and a subsequent model reply using learned context. Do not execute notebooks or enable repeating pulse.

**Owner acceptance statuses at this handoff:** existing discovery PASS; existing broken supervised run FAIL; new live-model lesson NOT RUN; full owner approval/persistence NOT RUN. No claim that TASS learned anything yet.
