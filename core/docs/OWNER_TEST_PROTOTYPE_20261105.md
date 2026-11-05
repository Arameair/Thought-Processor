# Owner-run prototype — November 5, 2026
**Branch:** `experiment/owner-ready-chat-learning-20261105`  
**PR:** https://github.com/Arameair/Thought-Processor/pull/66 (DRAFT; stacked on #65)  
**Source of requirements:** [November 5 owner-route handoff](https://github.com/Arameair/Thought-Processor/blob/experiment/tass-github-lesson-20261104/core/docs/OWNER_HANDOFF_20261105_GITHUB_RELAY_AND_LEARNING.md).  
**Scope:** one existing Go TASS application; real chat-driven or button-driven supervised Markdown learning, plus truthful GitHub relay status.  
**Do not confuse** source/CI tests with the owner's live Nemotron acceptance.

## What this prototype actually does
1. Keeps the existing #65 repaired Find lessons button / Identity denial messages / clear graded result and approval controls.
2. Adds #64's exact bounded **local owner chat** requests: `find lessons`, `can we try to learn any of them?`, `study My work pulse`. These dispatch the existing real `DiscoverLessons()` and `StudyLesson()` methods and display repository lesson paths or actual Engine job receipts. Unsupported chat remains normal local Nemotron conversation. Messages originating from GitHub are not allowed to invoke the owner-only lesson dispatcher.
3. Adds **Find lessons** and **Run one lesson test** beside Talk. They call `POST /api/lessons/discover` and `POST /api/lessons/study`; a job must really be created. They do not approve a result. Autonomous ON refuses the supervised test.
4. Adds a truthful readout under Talk: if `status.github_messages` is false it says **GitHub issue relay OFF in this Go Core**. A reply from the older Identity Liaison is not evidence of a Go Core/Nemotron reply.
5. Retains the existing Identity grants, Engine jobs, candidate pinning, updater, rollback, owner entity and external model configuration.

**Not implemented here:** GitHub-Issues-to-new-Go-Core message reader/publisher, automatic three-way voice loop, arbitrary Jupyter notebook Python execution, new GitHub permissions, model fine-tuning or autonomous learning with owner OFF. COMM-01 (runtime provenance and one authorized issue relay) remains pending.

## Step A — read-only, safe to do first; do NOT update yet

On the **owner's existing WSL checkout**, not a new workspace:

```bash
cd ~/tass-one-app/core
./tass.sh status
git fetch origin experiment/owner-ready-chat-learning-20261105:refs/remotes/origin/experiment/owner-ready-chat-learning-20261105
./tass.sh check-update origin/experiment/owner-ready-chat-learning-20261105
```

Record the *full* installed Core/Engine/Identity SHAs and the *full* candidate Core SHA printed by `check-update`. Check the Engine/Identity candidate pins, changed-file inventory and current Identity state. This command applies **nothing**; fetch only updates the remote-tracking Git ref, not the deployed code. If branch fetch fails, stop; don't replace it with another branch or an unreviewed update.

**Private GitHub authentication check:** `Arameair/TASS` is a **private repository**. This Go Core's lesson-sync client uses `~/.tass-app/secrets/github_token` (the startup script can populate it from the owner's `gh auth token` or an explicitly configured `GITHUB_TOKEN`). Identity's optional GitHub device sign-in is a **different, narrowly scoped session**, and does *not* automatically provide private-file access to the Core lesson client. Read-only checks on the owner PC:

```bash
gh auth status                    # account/auth health only; NEVER paste 'gh auth token'
grep '^GITHUB_REPO=' ~/.tass-app/app.env
grep '^GITHUB_BRANCH=' ~/.tass-app/app.env
```

Expect `GITHUB_REPO=Arameair/TASS` and `GITHUB_BRANCH=live`. Do **not** paste access tokens, invent a new login, or grant broader GitHub permissions just to make the prototype show lessons; stop with the exact authentication error if required access is unavailable.

Confirm the candidate is the reviewed PR #66 **head SHA** and GitHub CI tests are green for that exact revision. Verify `./tass.sh status` says the existing one-app still runs, the model is the same external Nemotron, and the user-facing Core/Identity ports are the expected ones (usually 8797/8796 in this installation). **Never assume an old SHA or another machine's port.**

Check your saved `Autonomous` setting **in the currently running Thought Processor**. Turn it OFF before considering the candidate. If it cannot be confirmed OFF, stop and debug the state first. Turning Autonomous OFF is an owner action, not a permission to modify anything else.

**Stop if:** current runtime SHAs contradict the review; GitHub branch/CI is missing; model/Identity unhealthy; a second model is running unexpectedly; a branch update would regress accepted fixes; an unreviewed permission is requested; or previous data/backup cannot be verified.

## Step B — only when owner explicitly approves exact reviewed SHA

Do **not** paste a moving branch ref into an updater. Copy the *full immutable candidate commit SHA* from Step A/PR #66 into `REVIEWED_SHA` after checking it matches the reviewed source/CI. Do not proceed on an unresolved mismatch.

```bash
cd ~/tass-one-app/core
./tass.sh backup
sudo -v
./tass.sh update REVIEWED_SHA
./tass.sh status
```

The `update` command asks for explicit confirmation, then builds and validates with automatic restore behavior on failure. Do **not** run `./tass.sh` itself under `sudo`; it handles Docker access. Keep existing running model, keys, data and TASS entity. Review backup contents before applying if you have not verified backup integrity. After a restart, Identity may be LOCKED intentionally; unlock only on the owner's local Identity page, not in chat. Stop on any failed validation, wrong pin, missing entity, forced safety mode, unexpected autonomous start or Docker error.

If the update fails, preserve outputs and use the existing rollback workflow only after verifying the current state and data safety; don't switch branch/reset or erase the checkout trying to repair it.

## Step C — owner's acceptance (REAL Nemotron, with Autonomous OFF)

**C0 — status:** open the existing **Thought Processor** browser address reported by `./tass.sh status`; verify expected Core/Engine/Identity/model state. Confirm the **GitHub issue relay OFF in this Go Core** message appears in Talk. This is deliberate and does not mean the older Liaison is offline.

**C1 — Identity grants:** unlock TASS on the existing Identity page. The *existing entity* may still have only the original five permissions. If the approved new card is present, read the warning that `github.sync` can write `tass.md` to the configured repository, then explicitly enter your owner passphrase locally to enable `github.sync` and `lesson.learn`. Confirm the UI acknowledges the exact grants. Do not confuse this with optional `@identity-test` Issue #51 approval. **If the card is missing or gives an error, do not recreate the entity. Stop and capture the message.**

**C2 — real Find:** press **Find lessons** beside Talk (or type `find lessons`). It should list the real `Arameair/TASS@live` lessons under Learning and show their paths/states. Before claiming PASS, verify at least one real lesson path and the actual `/api/lessons/discover` response. A notebook may say NEEDS_WORKSPACE; do not execute it. A 409 should display Identity's reason; do not auto-enable anything.

**C3 — real Study:** with **Autonomous OFF**, press **Run one lesson test**, or type `study My work pulse` / `can we try to learn any of them?`. The UI should show a real job ID, then actual answers to its checks and a graded status. One Engine pulse **step** is permitted for the explicit study, without switching the repeating work pulse on. Inspect the real Nemotron answer and source, not a canned fixture. If it says QUEUED for a while, wait/check pulse state; do not repeatedly trigger until status is understood.

**C4 — approval gate:** when and only when status becomes PASSED, visually confirm it is **waiting for owner approval** rather than already LEARNED. The owner decides whether to click **Approve**. After clicking, check local `## Learned`, revision/receipt and effective context through a fresh model question. A reported GitHub document write is *not* proof without the actual remote commit/file read.

**C5 — failure cases:** lock Identity and verify new Find/Study denies and shows the reason. For manual tests keep Autonomous off. Repeated clicks on a QUEUED lesson must not start an additional job. Wrong/non-checkable notebook material must never run.

**C6 — GitHub relay diagnosis (no changes):** inspect [TASS Issue #20](https://github.com/Arameair/TASS/issues/20) request [6652605601](https://github.com/Arameair/TASS/issues/20#issuecomment-6652605601) and response [6652617269](https://github.com/Arameair/TASS/issues/20#issuecomment-6652617269), then examine sanitized local old-Liaison/new-Core logs and service inventory to identify the actual poster. The reply carries `<!-- identity-liaison:tass -->`. The new Go app's source has `cfg.Messages=false`, so its current run cannot be credited for that message without independent evidence. **Do not turn on a second simultaneous GitHub issue reader/publisher.**

## Minimal evidence for the team
For each test, send: **TEST ID**, owner WSL / real Nemotron (not fixture), exact Core/Engine/Identity SHAs, repository source SHA/lesson path, visible user request, real job/cycle ID, Identity decision, actual graded answers, PASS/FAIL/NOT RUN and a concise nonsecret error. For COMM-01, also note inbound/outbound GitHub comment IDs, reader process/version and publisher identity. Never paste passphrases, tokens, private model prompts or raw personal journal data.

### Expected pass/fail summary
| Check | Must be observed | Current status |
|---|---|---|
| CI Go/race/page regression | exact candidate SHA has verified green run | PENDING |
| Owner preflight/backup | current install/pins/model/data preserved | NOT RUN |
| Identity grants | opt-in works on existing entity; no unwanted grants | NOT RUN |
| Find lessons | real GitHub source and status list, not fake result | NOT RUN |
| Study one lesson | real Nemotron job receipt, answer, grade; Autonomous OFF | NOT RUN |
| Owner approval | PASSED awaits owner, LEARNED only after click | NOT RUN |
| GitHub issue relay | identify old/new sender and correlate with model receipt | NOT RUN |
| Notebook execution | intentionally excluded | NOT RUN (not authorized) |

**Owner decision gate:** Do not merge/deploy/enable learning permissions until the read-only comparison, safety review and owner approval. A fixture-green build is not real acceptance.
