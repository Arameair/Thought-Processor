# Owner acceptance — GPT-6 ↔ TASS issue #24 (manual pilot)

Date: 2026-11-05. Engineering issue: [Thought-Processor #76](https://github.com/Arameair/Thought-Processor/issues/76). Draft PR: [#77](https://github.com/Arameair/Thought-Processor/pull/77).

**Status: NOT ACCEPTED.** Fixture/CI passing does not prove the owner machine ran the local Nemotron. Do not claim real success until the input, Identity decisions, Engine cycle and output are independently correlated.

## Scope

- The only pilot mailbox is [Arameair/TASS#24](https://github.com/Arameair/TASS/issues/24), open with `tass-core`.
- Allowed GitHub author: account configured in `TASS_RELAY_AUTHORS`, or the `Arameair` repository owner by default. A `ChatGPT:` prefix identifies intended origin but **does not cryptographically prove** that ChatGPT authored the text, because comments use the owner's GitHub login.
- Read-only conversation, local Nemotron, no model-offered actions, no lesson commands, no `@skills`, no arbitrary execution, and no learning approvals. `autonomous=false`, `pulse=false`, `learn=true` remain unchanged.
- `relay` is opt-in, defaults OFF, and is not tied to Pulse or Autonomous. Existing Identity permissions are NOT changed.
- Do not enable or send a pilot comment until the owner explicitly approves the reviewed experimental deployment. **Never publish passwords, keys, bearer tokens, private URLs, model reasoning or local logs.**

## 1. Preflight (read-only / safe to do now)

From the owner's **existing checkout** `~/tass-one-app/core`:

```bash
cd ~/tass-one-app/core
./tass.sh status
git rev-parse HEAD
git status --short
git fetch origin experiment/tass-mailbox-receipts-20261105
./tass.sh check-update origin/experiment/tass-mailbox-receipts-20261105
```

Expect currently deployed Core `9be30a502741dcbe3ad6589794ae61776c18681f`, Engine `21b3d502775b`, Identity `91e1521659a3`. Confirm locally before any change. Fetching is not a deployment. Verify PR #77 CI and review the diff. In the TASS settings, confirm `autonomous=false`, `pulse=false`, `learn=true` and `relay=false` before proceeding.

The Go Core repository may have further upstream changes by the time of the trial; **stop** if the actual code SHAs do not match the reviewed candidate.

## 2. Approved deployment only

Only **after** owner approval, back up and apply the exact reviewed candidate SHA. Commands intentionally omit `--yes` so the script requires a second human confirmation.

```bash
cd ~/tass-one-app/core
./tass.sh backup
./tass.sh update <REVIEWED_PR_77_HEAD_SHA>
./tass.sh status
```

Record the exact candidate SHA before starting. The update may restart containers and Identity may require unlocking locally. Confirm Core reports `READY`, Identity `UNLOCKED TASS`, `CHAT_ONLY`, local Nemotron available, `autonomous=false`, `pulse=false`. If a check fails, **stop**; use the script's documented rollback path and do not post a test comment.

## 3. Prime, then one actual GitHub input

Switch **Talk on GitHub** ON in the updated TASS local page (other settings stay unchanged). Wait until the **Core's own local log** confirms:

`Mailbox #24 primed from existing comment IDs; waiting for new ChatGPT: comments`

Do not post before that receipt. The pilot deliberately excludes comments that existed when the switch was enabled.

Only after owner approval and confirmed priming, GPT-6 posts **one** message via the GitHub connector to Issue #24, with exactly this harmless text:

```text
ChatGPT: MAILBOX-01 — Please acknowledge this GitHub comment and state your cycle ID. Do not execute tools, study lessons, modify files, approve learning, or change settings.
```

Record the **actual incoming GitHub comment ID and URL**. A successful GitHub API POST alone proves only that GitHub received the comment, not that TASS read or processed it.

## 4. Acceptance evidence (collect locally; share sanitized IDs only)

Inspect the Core's local event/status and `~/.tass-app/data/core/relay-audit.jsonl`, the Core's `state.json`, the Engine's `cycles.jsonl` / `model-requests.jsonl`, and Identity's authorization decision records. Match all of:

1. `Arameair/TASS#24` **incoming** comment ID + authenticated GitHub author + exact `ChatGPT:` text
2. Core reserved and accepted **that exact comment ID**
3. Identity decision identifiers allowing `conversation`, `model.request`, `thought.process` and `github.sync` for the live attempt
4. Thought Engine **same cycle ID** and real configured local model request; verified `NO_ACTION` or `NOT_AUTHORIZED`, never `EXECUTED`
5. `Arameair/TASS#24` **outgoing** TASS comment ID and URL, reply quoting the original input comment ID and cycle ID
6. Local `relay-audit.jsonl` record `stage=posted` with source ID, output ID, cycle ID, Identity decision references

Run an additional poll, then restart/recheck **only if owner approves**; there must remain **exactly one** TASS output comment for the input. Confirm unapproved authors/issues and comments with a `TASS:`/Core marker are not handled, and LOCKED Identity holds the relay without model execution.

A fixture-only pass, a Core UI claim, a missing outgoing ID, a different Engine cycle or an `AMBIGUOUS` post is **NOT PASS**. A network timeout after POST is deliberately held, not automatically retried; manually inspect Issue #24 and reconcile before any further attempt.

## 5. Restore safe state

Turn **Talk on GitHub** OFF and confirm the saved setting. Verify `CHAT_ONLY`, `autonomous=false`, `pulse=false`, `learn=true`, no pending work and no learned-lesson approval. Record final `./tass.sh status` and installed SHAs. If the owner wants the previous Core, use `./tass.sh rollback` (do not restore/delete data without first reviewing the exact backup and rollback choice).

## 6. Reporting

Report exact branch/commit/PR and CI links; original/final Core, Engine and Identity SHAs; input and output comment URLs and numeric IDs; Engine cycle; selected Identity decision IDs; observed action outcome; duplicate-poll/restart count; any failure/hold, and final settings. **Redact private data.**

Only after this manual acceptance succeeds should anyone evaluate whether scheduled ChatGPT tasks can make **unattended, authorized GitHub connector writes and reads**; do not create a recurring task or claim this is supported beforehand.
