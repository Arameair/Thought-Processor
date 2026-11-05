# Owner handoff — identify TASS's GitHub relay and prove one real supervised lesson
**Date:** 2026-11-05 (prepared while owner is on delivery route)
**Audience:** owner, Claude, GPT-6/ChatGPT, independent test reviewer, and TASS only if a proven authorized reader later retrieves it
**Status:** documentation and test plan only. No owner-machine test or update was run while preparing this note.
**Primary repository:** Arameair/Thought-Processor (one owner-facing application); related: Arameair/Thought-engine, Arameair/identity, Arameair/TASS.
**Priority:** P0 communication provenance + supervised learning. No new applications, model servers, schedulers, arbitrary notebook execution, or broad autonomous features.

## 1. Owner's actual objective

The owner wants to talk with ChatGPT by voice during delivery routes, and—when the locally running TASS application and its authorized bridge are available—have ChatGPT exchange short messages with TASS through GitHub. The owner also wants to teach TASS something through an understandable text/notepad or Jupyter-notebook learning format, watch it study and be tested using the real local Nemotron, inspect the result in Thought Processor, and approve a learned skill only after evidence. The end state is a useful shared improvement cycle, not a GitHub comment that merely *claims* success.

Today/tonight's **minimum two proof milestones**, in this order:
1. **COMM-01** — a ChatGPT-originated, owner-authorized comment is seen by exactly the intended local TASS runtime, processed using its real Thought Engine/Nemotron, and posted back through an identified, authorized publisher with matched receipts. The owner and ChatGPT can inspect the reply. This may operate with autonomous work OFF and must not grant action authority.
2. **LEARN-01** — in the same one-app Thought Processor, an existing Markdown lesson is actually found on TASS@live, studied by real Nemotron, graded from observed answers, held for owner review with Autonomous OFF, approved by the owner, and present in the entity's effective learned context. Show the immutable source revision, job/cycle and Identity decisions; distinguish local state from a confirmed GitHub write. **Notebook code execution is not a requirement for LEARN-01.**

A clean GitHub relay and one supervised Markdown lesson are sufficient for the first milestone. A later, separately scoped notebook-cell execution experiment requires its own sandbox/security acceptance; do not silently redefine “learning” as arbitrary Python execution.

## 2. Responsibility and authority map (one application)

| Component | Owns | Must not do |
|---|---|---|
| **TASS repo** | Reviewed companion identity document (tass.md on live), versioned lessons, bounded coordination issues | Treat a GitHub comment or lesson as authorization; silently grant itself capabilities |
| **Identity repo / deployed Identity service** | Entity enrollment, owner unlock, explicit grants, optional GitHub account connection, authorized credentials/publishing boundary | Let model text, pulse ticks or an issue label substitute for policy approval |
| **Thought-Processor repo / Go Core** | One owner conversation screen, lesson discovery, supervised Study/Review/Approve controls, coordination, GitHub transport adapters, honest status | Fabricate job success, operate a second autonomous runtime, or assume the old Liaison is the new Core |
| **Thought-engine repo / Go Engine** | One nominal 20-second tool/work scheduler, model request IDs, observable Spark/Trace/Thought, prediction, state, outcome and evaluation | Treat a health ping as a completed thought/lesson, replay the same job each tick, invent hidden reasoning |
| **Local Nemotron** | Inference for bounded conversation and lesson answers | Hold GitHub credentials, directly authorize permissions or execute lesson code |
| **Older Identity Liaison / Workshop** | Previous GitHub issue reader/writer; possible existing responding process to be identified | Be silently accepted as proof the new Docker Go application replies |

One entity, one visible owner-facing Thought Processor, one existing local model. A pulse is a scheduling cadence and is distinct from message transport and from the duration of a model inference. Conversation must not require turning Autonomous ON.

## 3. Current evidence ledger — timestamped, with environment labels

### Verified source or GitHub facts (KNOW)
- On 2026-11-05 ChatGPT posted a **communication-only** question to [TASS Issue #20](https://github.com/Arameair/TASS/issues/20#issuecomment-6652605601), requesting the exact marker TASS-RELAY-1105. A later [GitHub comment](https://github.com/Arameair/TASS/issues/20#issuecomment-6652617269) appeared with the marker “TASS-RELAY-1105: I received this message through GitHub.” Its body includes the marker used by the older Identity Liaison (identity-liaison:tass). This is concrete GitHub write/reply evidence; **not** proof of which live process/model generated the text.
- Earlier [TASS Issue #20](https://github.com/Arameair/TASS/issues/20) had an older matching relay test/reply (“GitHub channel received: message”). [TASS Issue #19](https://github.com/Arameair/TASS/issues/19) was designed as an owner-enabled *sanitized* collaboration/evidence feed; it had no actual event feed in the inspection. Do not confuse GitHub post-back with local execution receipts.
- The new Go app source on Thought-Processor **PR #65 branch** explicitly sets **cfg.Messages = false** in core/cmd/tass-core/main.go when IdentitySvc is configured, with the comment “GitHub issue messages are the prototype's path; app mode talks on the page.” In that same code's core/tass/core.go, SyncOnce **app mode** runs applySettings / syncTass / syncLessons / advanceLessons; only the **prototype** branch runs readMessages / flushOutbox. Therefore the existing Go app's document/lesson sync is **not** the same as a verified two-way issue conversation. No code change has been made by this handoff.
- [TASS PR #23](https://github.com/Arameair/TASS/pull/23) is merged into live; three structured Markdown lessons are available, with Teach / Check / Answer fields, and an additional notebook lesson may be listed as NEEDS_WORKSPACE. See [lessons/README.md](https://github.com/Arameair/TASS/blob/live/lessons/README.md). The deployed model has not yet passed a real owner-observed lesson acceptance test.
- [Thought-Processor PR #63](https://github.com/Arameair/Thought-Processor/pull/63) adds owner-supervised Find → Study → graded result → Approve with Autonomous OFF; test/CI evidence is reported in PR comments, not the same as live Nemotron acceptance.
- [Thought-Processor PR #65](https://github.com/Arameair/Thought-Processor/pull/65), stacked on #63, fixes a **real page bug**: Find lessons previously called /api/pulse and returned 409 because a generic .ctl button handler replaced the Find handler. The new handler targets the pulse's data-a buttons only; it also preserves click feedback and shows job, answers and authorization errors. [Fixture/browser evidence](https://github.com/Arameair/Thought-Processor/blob/experiment/claude-supervised-lesson-fix-20261104/core/docs/SUPERVISED_LESSON_EVIDENCE_20261104.md) shows Find (4 items), one Markdown Study, PASSED 3/3, manual Approve, and a fixture GitHub document update, **using fake GitHub and a stand-in model**, not actual owner's Nemotron.
- [Thought-Processor PR #64](https://github.com/Arameair/Thought-Processor/pull/64), also stacked on #63, routes certain explicit **local owner chat** phrases to real discovery/study APIs and issues receipts. It does not handle GitHub-origin messages or execute notebooks. **#64 and #65 do not include each other's complete fix; integration is not established.**
- [Identity PR #35](https://github.com/Arameair/identity/pull/35) enables existing entities, with the owner's passphrase, to opt into specifically scoped github.sync and lesson.learn. [Identity PR #36](https://github.com/Arameair/identity/pull/36) clarifies consent/result UX. A prior screenshot saying “Confirm your Identity passphrase first” was reproduced as **client-side empty-passphrase validation** (no network request). This does not prove the server rejected a valid passphrase or that learning is enabled on the owner's machine.
- [Thought-engine PR #11](https://github.com/Arameair/Thought-engine/pull/11) integrates relevant owner-lesson dialogue, pulse safety and model-answer retention. Tests from PRs must not be represented as real-model performance.
- Existing owner reports in [Thought-Processor Issue #42](https://github.com/Arameair/Thought-Processor/issues/42) and [Issue #60](https://github.com/Arameair/Thought-Processor/issues/60) show that an earlier installed Go TASS had working Docker/Identity/real Nemotron basic conversation, but lesson commands and notebook execution remained unaccepted. Later owner updates and separate fixes changed candidate revision pins; **tonight's installed SHAs are currently UNKNOWN until status is run**.

### Inferences to verify (INFER, not acceptance)
- The newly received Issue #20 marker most likely came from a still-running **older identity-liaison route**, because its comment marker matches that path, while the new Go app explicitly disables issue messages. The precise emitter/container/commit is still UNKNOWN. A GitHub comment can also be posted by other processes acting as the same GitHub account; author name alone cannot identify the runtime.
- A short local-model greeting and a working 20-second pulse do not establish GitHub transport or actual skill acquisition.
- Merging only PR #64 or only PR #65 will leave a complementary UI/chat behavior to reconcile; a reviewed integration candidate is needed, not a blind newest-branch update.

### NOT RUN / UNKNOWN right now
- Present owner machine runtime health, current installed Core/Engine/Identity SHAs, connection state, whether old Liaison is still running, the exact local process that wrote marker TASS-RELAY-1105.
- New Go Core incoming issue comments, new Go Core outgoing issue comments, present Identity GitHub write permissions, message-to-cycle and cycle-to-publisher correlation.
- The real Nemotron supervised Markdown lesson on owner's current installation and owner approval persistence.
- Real authorized notebook Python cell execution in a restricted sandbox, broader autonomy, remote tool use or continuous autonomous multi-agent conversation.

## 4. COMM-01 — trace *who* answered; do not create competing responders

### Required end-to-end path
Owner voice → ChatGPT (explicitly relayed via connected GitHub account) → a single designated TASS GitHub issue → authorized inbound GitHub reader → verified local Core/Identity router → Thought Engine cycle → actual local Nemotron request → structured reply → authorized outbound publisher → GitHub issue comment → ChatGPT reads that *specific* comment.

The older Liaison currently appears to be capable of publishing. First **identify and instrument**, then decide whether to reuse its transport as a contained adapter or replace it with a **single** Identity-owned bridge to the existing Go Core. Do not turn on multiple readers/publishers for the same label/thread simultaneously.

### Read-only audit tonight
1. On the owner PC, obtain sanitized ./tass.sh status, Compose container names/images, process/service inventory, the exact deployed pinned versions, and the active model address. Do not stop the working app or reveal keys/tokens/passphrase.
2. Search sanitized logs around the new Issue #20 comment IDs **6652605601** (ChatGPT request) and **6652617269** (reply). Find which process polled/accepted it, which process generated the reply, and which authenticated component posted it. Compare receipt times and unique runtime identifiers.
3. Inspect old Identity Liaison/Workshop service status and its delivery/outbox logs **without restarting it**. Establish whether it still owns Issue #20. Inspect new Go Core status separately; in app mode its source disables GitHub issue messages. Record observed versus inferred behavior.
4. Verify that the responding runtime actually used Nemotron (model request ID and Engine cycle), rather than echoing a canned marker or a different worker. No raw model prompt/hidden reasoning/log dump on GitHub.
5. If no correlated local receipt exists, report **“GitHub reply observed; live runtime provenance UNKNOWN”**, not COMM-01 PASS. The GitHub reply is a useful partial test, not yet a complete proof.

### Minimal implementation target (only if audit confirms a gap)
- Route owner-authorized GitHub conversation through the **one** chosen Identity-mediated reader/writer to current Thought Processor, not through a second companion.
- Separate scoped inbound conversation permission from github.sync document changes, lesson.learn, execution permission, and outbound issue-comment publishing. A read-only lesson grant is **not** a general issue-writer grant.
- Only accept permitted repository/issue/owner, with immutable incoming comment ID, source URL, body hash, received timestamp, and attribution. Do not trust “ChatGPT” written in a comment as authentication: multiple actors may post via the same Arameair account.
- Reject self-replies via exact publisher markers, maintain durable seen-message and outbox IDs, and ensure restart/retry cannot create feedback loops or duplicate replies. Explicitly handle 429/rate limit/offline, timeouts, absent grant, locked Identity and backlog.
- Show delivery status and receipts in Thought Processor. An accepted GitHub comment must not be marked PROCESSED before a model reply; a successful local response must not be marked POSTED before GitHub confirms its comment ID.
- For a successful controlled probe, link inbound comment ID → local delivery receipt → Core/Engine/model IDs → Identity authorization decision ID → outgoing GitHub comment ID. Record process/container identity and build SHAs **locally**; publish only a safe correlation summary to GitHub.
- Do not give the local model GitHub credentials or unrestricted network/shell access. Tokens remain under Identity/approved publisher ownership. The model's generated text is data, never a permission request with implied authorization.

### COMM-01 acceptance matrix
- C1: Exactly one trusted reader handles a new controlled message once. **NOT RUN**.
- C2: Correct local Core/Engine/model identity and matched incoming-to-cycle receipt. **NOT RUN**.
- C3: One new GitHub reply with exact reply ID and local publisher receipt. **NOT RUN**.
- C4: Repeated poll / restart does not duplicate the model job or comment. **NOT RUN**.
- C5: Identity LOCKED, revoked grant, wrong account/issue, forged sender label, and offline cases fail closed with truthful visible status. **NOT RUN**.
- C6: Normal owner conversation remains possible with Autonomous OFF; no lesson executed as a side effect of the relay test. **NOT RUN**.

## 5. LEARN-01 — one Markdown/notepad lesson first, with real Nemotron

### Lesson representation now available
The TASS@live lessons README defines simple text lessons with a title, author, one-line goal, and 2–5 small steps, each containing **Teach**, **Check**, **Answer**. The model studies without seeing the expected Answer line, and the **application** grades the actual output. This is the first notepad-like format to prove. The presence of lesson files on GitHub does not establish that the model read them.

A notebook lesson (.ipynb) is a different class of artifact. At present the new one-app can list such a lesson as NEEDS_WORKSPACE; it does **not** run notebook cells in a secured kernel. Do not promise it can. A “PASSED 3/3” Markdown check demonstrates task-bound recall/response, not general model-weight fine-tuning, persistent capability to operate tools, or notebook execution.

### Exact safe manual flow
1. Confirm owner has actual currently installed version and backup/rollback available. Work pulse and Autonomous both **OFF** for the first owner acceptance.
2. Unlock the existing Identity; do **not** create a replacement entity. Inspect current grants. Existing entities may have just five older grants. If the updated consent card is installed, the owner enters the passphrase **locally** to enable only the explicit grants required (github.sync and lesson.learn). Confirm the actual authorization record / decision IDs and UI success. Do not print passphrase.
3. Confirm expected repo **Arameair/TASS**, branch **live**, and lessons directory. Confirm source commit SHA/hash of the selected lesson; observe whether the app truly read the live repo or a cached/fixture copy. Do not run source as code.
4. In **Thought Processor**, click **Find lessons**. Expected: real /api/lessons/discover request and truthful lesson list including known Markdown lessons; the .ipynb may say NEEDS_WORKSPACE. If the button instead calls /api/pulse or returns HTTP 409, stop and record browser/action error; PR #65 specifically targets this fault.
5. Choose **My work pulse** or another smallest known Markdown lesson; click **Study (manual)**. Record an actual job receipt, Engine pulse/job ID, Identity decisions, model request ID, source SHA, and transitions NEW → QUEUED → RUNNING → PASSED / RETRY / STUCK. A label change without a job is not PASS.
6. Inspect each real Nemotron answer alongside **Check** and actual grading; do not claim mastery because the model says “I learned.” A passing answer must be visible before owner approval. Do not auto-approve in supervised mode.
7. **Only after a real PASSED**: owner decides whether to press **Approve**. Verify durable LEARNED entry and loaded effective context after refresh/restart, then separately verify an actual authorized GitHub versioned write if the design promises it. A failed remote sync must not be reported as successful publishing.
8. Ask a new simple conversation question that depends on the approved lesson. Distinguish retained context retrieval from a new inference, and capture a cycle/model request. Check that conversation is still usable with Autonomous OFF.

### Known integration caveats
- #63 provides supervised behavior. #65 fixes the browser click/feedback. #64 connects specific local chat intents to real APIs. These must be reviewed **together** with the compatible Identity #35/#36 and Engine #11 versions, preserving owner's previously accepted real-model/OAuth fixes.
- #65 browser acceptance was on a **fake GitHub** and **fake model**, even though it ran real Go server/UI code. Its PASSED 3/3 and fake GitHub update are not owner Nemotron/GitHub acceptance.
- The Identity consent card is distinct from the older one-off approved Issue #51 @identity-test reference; do not conflate permission to retrieve a fixed text reference with permission to sync/write the entity document or learn.
- github.sync can **write** tass.md, not just read lessons. Explain its scope to owner before enabling; an explicit consent and a safe branch are required.

### LEARN-01 acceptance matrix
- L1: Real GitHub lesson source pinned and discovered. **NOT RUN**.
- L2: Owner/Identity granted only required scopes, logged decision, refusal when locked. **NOT RUN**.
- L3: Real Nemotron studied/answered actual lesson, model request logged. **NOT RUN**.
- L4: Application graded checks using real outputs and displayed job/cycle trace. **NOT RUN**.
- L5: Manual PASSED remains awaiting owner approval, Autonomous OFF. **NOT RUN**.
- L6: Owner approves exact passed lesson, durable LEARNED state and effective context survive restart; independent GitHub write verified only if part of this slice. **NOT RUN**.
- L7: Failed check/timeout/incorrect answer is visibly not LEARNED; retry is bounded and idempotent. **NOT RUN**.
- L8: No notebook Python cell ran and no model-issued tools were implicitly authorized. **NOT RUN** (verify guard).

## 6. Notebook/Jupyter follow-on — deliberately separate from the first lesson

Owner's longer goal is a teachable note/notebook with an explanation, a real proposed action, an immutable expected observation, an authorized single test, actual output, comparison, and eventual reusable skill. Reuse existing notebooks/prototype lessons as references. For the *first executable* notebook experiment, accept only an approved and pinned **fixed operation** in a restricted runner; do not execute arbitrary notebook cells merely because their metadata says THOUGHT/EXPECT. A folder path is not a security sandbox.

The future runner must be non-root and constrained: isolated scratch, no Docker socket, owner home, identity secrets or unrestricted subprocess/network, resource/time caps, source SHA/version checks, independent Identity authorization at point of use, durable idempotent receipts, comparison of actual stdout/stderr/exit/expected, and stop-on-mismatch. Never create a second scheduler or independent app. See [Thought-Processor Issue #60](https://github.com/Arameair/Thought-Processor/issues/60). **Owner acceptance for real code execution is NOT RUN.**

## 7. Tonight's practical order (owner at home; no route commands)

**Gate 0 — Record exact starting state; no changes.**
- In the existing checkout only, collect the sanitized output of:
    cd ~/tass-one-app/core
    ./tass.sh status
- Inventory actual containers/old relay and preserve running model. Read status and logs; do not stop the old runtime or enable the 20-second pulse. Record exact SHA pins and whether Identity is LOCKED or UNLOCKED. **Do not assume yesterday's notes describe today's deployed revision.**

**Gate 1 — Determine relay origin (COMM-01).**
- Correlate today's Issue #20 IDs to the **one** old or new local process. If only the older Liaison responds, document that partial success and close the diagnosis with a precise, small integration task. Do **not** claim a direct ChatGPT ↔ new Go TASS connection.
- If a reviewed bridge fix is needed, implement on an isolated branch, run fixture/race/security tests, and require owner approval before updating the one-app. Keep only one active listener on the final thread. Repeat a harmless marker test and capture model/outbox receipts. No action permissions.

**Gate 2 — Review rather than blindly update.**
- Review PRs #63/#64/#65 with Identity #35/#36 and Engine #11 for exact compatible commit pins and omitted changes. Use the existing check-update/backup/rollback workflow for an **immutable reviewed candidate**. Old and new repos may be divergent, so a newer-looking branch is not automatically a safe upgrade.
- Keep Autonomous OFF and prevent saved auto-on settings from starting work on restart. Never update/merge/deploy without owner's explicit approval. Keep the existing entity, passphrase, keys, GitHub session and external Nemotron. No rollback that silently discards newer data.

**Gate 3 — Real supervised lesson (LEARN-01).**
- Test Identity unlock/explicit learning grants, Find, manual Study, real model answers, app grading, waiting for owner approval, and actual durable learning outcome. One lesson, no notebook execution.
- STOP at first failure and attach sanitized diagnostics to its existing PR/issue. Do not turn on broader autonomy “to make it work.”

**Gate 4 — One shared loop demonstration (only after gates 1 and 3).**
- Owner tells ChatGPT a bounded question about what the lesson taught; ChatGPT sends **one** owner-authorized comment; verified TASS receives it, answers with real Nemotron using approved context; authenticated publisher posts **one** evidence-linked reply. ChatGPT reads and reports response. Owner sees matching Core/Engine receipts.
- Do not automatically re-post ChatGPT's answer back to itself. A human-approved or otherwise explicitly scoped next task starts the next cycle. This is a supervised collaborative loop, not an unbounded self-conversation.

### Stop conditions
Stop on wrong current SHA, missing backup, unreviewed mixed source pin, missing Identity grant, model unavailable, source mismatch, two GitHub responders, uncorrelated model cycle, incorrect/duplicate receipt, unexpected write, proposed arbitrary code execution, or any UI claiming PASS without an observed job. Preserve evidence, mark BLOCKED/UNKNOWN, and ask owner before remediation.

## 8. Evidence format for every tonight test
Capture a compact record:
- Test: COMM-01 or LEARN-01 / case ID
- Timestamp + environment: OWNER WSL / real Nemotron, or FIXTURE / sandbox
- Source: repository/branch/commit/file hash, inbound GitHub issue/comment ID if applicable
- Runtime: Core / Engine / Identity exact SHAs, worker name/build, one model endpoint/model name
- Permissions: Identity state, grant name, allow/deny and decision ID (not passphrase)
- Trace: inbound receipt, job, cycle ID, model request ID, pulse tick, outgoing comment ID
- Observed outcome: minimal sanitized facts (answer/check score, post ID, retry/error)
- Status: PASS / FAIL / BLOCKED / NOT RUN / UNKNOWN; keep “reported by model” separate from verified execution
- Next owner action: one concrete step. No raw private memories, credentials, model hidden reasoning, unredacted prompts or huge logs in GitHub.

## 9. Task/coordination ownership — avoid branch collisions
- **Owner:** approves scope, exact deployment, Identity grants, lesson promotion, and any later restricted notebook execution; runs real PC acceptance when home.
- **Claude:** inspect/merge the candidate UI and supervised-learning behavior (#63/#65 and its Identity consent dependency) on a controlled integration branch; report exact pins/fixture+CI and owner instructions, do not silently deploy.
- **GPT-6/ChatGPT:** trace the GitHub liaison/new Go Core gap; propose minimal transport/authorization/idempotence repair; independently verify real issue-post evidence vs local runtime evidence. Handle GitHub coordination notes.
- **Thought Engine maintainer/reviewer:** verify actual engine model/job/cycle/locked prediction and 20-second scheduler behavior; no duplicate scheduler, no fabricated thought stream.
- **Independent auditor:** challenge test evidence and authorization claims; verify publisher identity, no secret logging, and that a fixture PASS isn't relabeled OWNER PASS.
All work stays in experimental branches or reviewed PRs. Do not merge into main without a separate owner review; keep one authoritative integration target.

## 10. References / read-first index
- Shared design notes: [TEAM_TASS_WORKFLOW.md](https://github.com/Arameair/Thought-Processor/blob/experiment/tass-github-lesson-20261104/core/docs/TEAM_TASS_WORKFLOW.md).
- Owner specification and evolving UX: [Thought-Processor #42](https://github.com/Arameair/Thought-Processor/issues/42).
- Owner machine/deployment evidence: [Thought-Processor #44](https://github.com/Arameair/Thought-Processor/issues/44).
- Notebook safety and real execution gate: [Thought-Processor #60](https://github.com/Arameair/Thought-Processor/issues/60).
- GitHub conversation marker and earlier relay: [TASS #20](https://github.com/Arameair/TASS/issues/20); optional sanitized feed [TASS #19](https://github.com/Arameair/TASS/issues/19).
- Current candidate PRs: [Core #63](https://github.com/Arameair/Thought-Processor/pull/63), [Core #64](https://github.com/Arameair/Thought-Processor/pull/64), [Core #65](https://github.com/Arameair/Thought-Processor/pull/65), [Identity #35](https://github.com/Arameair/identity/pull/35), [Identity #36](https://github.com/Arameair/identity/pull/36), [Engine #11](https://github.com/Arameair/Thought-engine/pull/11), [TASS #23](https://github.com/Arameair/TASS/pull/23).
- Source-critical gap: core/cmd/tass-core/main.go (app-mode cfg.Messages false); core/tass/core.go (app-mode SyncOnce does not read issue messages/flush outgoing comments); both on Core #65 candidate branch.
- [Fixture/browser lesson evidence](https://github.com/Arameair/Thought-Processor/blob/experiment/claude-supervised-lesson-fix-20261104/core/docs/SUPERVISED_LESSON_EVIDENCE_20261104.md).

**Handoff summary:** The latest GitHub response is real as a **posted comment**. The live authoring runtime is **unverified**. The manual Markdown learning path exists in candidate code and passed **fixture tests** but has **not passed real Nemotron owner acceptance**. First verify which service replies; then prove one supervised lesson without autonomous side effects. The whole project remains ONE TASS application, not several competing prototypes.

## 11. November 5 prototype implementation update (added after the initial route handoff)

A runnable **draft integration candidate** now exists in [Thought-Processor PR #66](https://github.com/Arameair/Thought-Processor/pull/66), branch `experiment/owner-ready-chat-learning-20261105`, **based on** Claude's repaired #65 branch. The source combines #64's local-owner chat command dispatcher with #65's working Find/Study/Approve button wiring, adds a truthful `github_messages` relay status display, a regression test, and generates numbered answer instructions for every check instead of just two example answer lines. The candidate does not turn on Go-app issue messages or add a GitHub publisher.

Owner instructions and precise acceptance gates: [OWNER_TEST_PROTOTYPE_20261105.md](https://github.com/Arameair/Thought-Processor/blob/experiment/owner-ready-chat-learning-20261105/core/docs/OWNER_TEST_PROTOTYPE_20261105.md).

**Critical credential distinction confirmed from source:** the TASS repository is private. Existing `core/tass.sh` installs a separate Go Core GitHub token in a restricted secret file from `gh auth token` or explicit `GITHUB_TOKEN` when configured. Identity's optional memory-only GitHub device login grants a read-only pinned Issue #51 pilot and does **not** automatically provide the Core API client a token for private TASS lesson files. Before claiming Find PASS, inspect sanitized `gh auth status`, `GITHUB_REPO=Arameair/TASS`, `GITHUB_BRANCH=live`, and actual API discovery behavior. Never paste token values.

A first PR #66 Go CI run (code revision `c9a7fd7`) passed vet, race tests, build, real-image build, Compose configuration and launcher syntax. Later source improvement and documentation commits have their own CI runs; confirm **the final exact PR head** before owner approval. Fixture CI does **not** prove the owner's Nemotron or the GitHub issue bridge.

**Status: IMPLEMENTATION CANDIDATE / OWNER TEST NOT RUN.** No current owner deployment changed, no extra service created, no permissions altered and no guaranteed live ChatGPT↔new Go TASS relay. COMM-01 and LEARN-01 are still evidence gates, not success claims.
