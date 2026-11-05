# Owner session: TASS foundation candidate (2026-11-05)

One integrated candidate for the owner to test, in order. Nothing in this document has been deployed,
merged, started on the owner's computer, or run against real Nemotron. Every result says how it was
obtained:

| Label | Meaning |
|---|---|
| **CI PASS** | GitHub Actions run on the pushed commit |
| **Fixture PASS** | isolated test with fakes (fake Docker, fake GitHub, fake Identity/Engine in Go tests) |
| **Real Go Services PASS** | the real Identity, Thought Engine and Core *binaries* as local processes, with the TEST FIXTURE model and fake GitHub (no Docker, not the owner's machine) |
| **Real Nemotron PASS** | the owner's local model. **NOT RUN for anything here** |
| **Owner Accepted** | the owner ran it on their machine. **NOT RUN for anything here** |

## 1. Exact versions

| Component | Branch | Full SHA | PR |
|---|---|---|---|
| Core (candidate) | `experiment/claude-foundation-integration-20261105` | `fafea55ac1bb91b00df3497bb1ac324c4cc67ffa` (code; this doc is the next commit) | draft PR into `experiment/claude-cognition-repair-junction-20261105` (see §7) |
| Core base: Cognition + junction | `experiment/claude-cognition-repair-junction-20261105` | `1ec138de22acfe63147e24ff7587ec3aeb4cda22` | [#87](https://github.com/Arameair/Thought-Processor/pull/87) (draft) |
| Core merged in: GPT-6 mailbox | `experiment/tass-unified-mailbox-gpt6-20261105` | `e34085eb0e3e20c0e59d88937a92188b28562fc8` | [#81](https://github.com/Arameair/Thought-Processor/pull/81) (draft) |
| Thought Engine (pinned) | `experiment/cognition-journal-projection-20261105` | `d69d3917d1601d806d4b2ba2254693e1adb92b6b` | [Engine #15](https://github.com/Arameair/Thought-engine/pull/15) |
| Identity (pinned) | `experiment/junction-grant-lesson-metadata-20261105` | `7da7f7c5bbecc42d3a7e9316a2037243eda10b88` | [Identity #40](https://github.com/Arameair/identity/pull/40) |

`core/tass.sh` pins: `ENGINE_COMMIT_PIN=d69d3917…`, `IDENTITY_COMMIT_PIN=7da7f7c5…` (unchanged from #87).
The candidate commits on top of #87: `231c380` (merge #81), `2574a0a` (#79 updater), `bc46ad9` (mailbox tests, #78), `fafea55` (learning).

## 2. What is implemented and what is not

| Area | Status | Evidence |
|---|---|---|
| **P0 #79 update manager**: verified backups; failed backup aborts and restarts the old version; operation journal + `recover`; bad rollback undoes itself; `restore-previous` (exact previous version + its verified pre-update data, current data saved first); `history`; one operation at a time | Implemented | **Fixture PASS** 14/14 (`test_updater_recovery.py`, real `tass.sh` code, fake Docker); **CI PASS**. **Real Docker NOT RUN** (sandbox has no image registry) |
| **P1 mailbox #81 on #87**: GPT-6 → GitHub #24 → Core → Identity → Engine cycle → reply comment with input ID + cycle ID → durable receipt | Implemented (merged) | **Real Go Services PASS** 20/20 (`mailbox_native.py`); Go **Fixture PASS** (3 tests); **CI PASS**. Real GitHub and Real Nemotron **NOT RUN** |
| **P2 one Markdown lesson**: Git revision → Engine study → graded → failure recorded, new job ID (#71) → owner approve → restart → **fresh-task retest** | Implemented (retest is new) | **Real Go Services PASS** 21/21 in Chrome (`learning_e2e.py`); Go **Fixture PASS**. The fixture model answers from a lookup table: this proves the plumbing, **not learning**. Real Nemotron **NOT RUN** |
| **P3 Development "Teach & test"** (Choose → Study → Inspect → Grade → Approve → Test again → Evidence) and **Cognition lesson evidence** (source SHA, attempts with job + cycle IDs, grade, approval, retests, NOT RECORDED when missing) | Implemented | **Real Go Services PASS** (same run, screenshots) |
| **#78 editor**: `uses` → non-skill and a `next` circle refused before the graph changes (add and relation change); Save refuses both | Implemented | Browser **Real Go Services PASS** (4 checks); Go **Fixture PASS** `TestBrainSaveRefusesUsesToNonSkillAndNextCycle` |
| **P4 real Engine 20 s pulse, many cycles** | Tested, no code change needed | **Real Go Services PASS**, see §5 |
| Skills: Identity grant + owner approval + receipt | Unchanged from #87 (already tested there) | Learning never grants or runs a skill: shown in Development and Cognition |
| Five-card thought design | **NOT implemented** | — |
| Hourly external-AI scheduler | **Not added** (by instruction) | — |
| Compose: `TASS_RELAY_AUTHORS` is not passed into the container | Not changed | Core then allows the repository owner (`Arameair` for `Arameair/TASS`), which is the intended author |

## 3. Test commands (all isolated; none touch Docker or the owner's data)

```bash
cd core
go vet ./... && go test -race -count=1 ./...                    # CI PASS
python3 test/test_updater_recovery.py -v                        # #79, 14 tests, fake Docker
python3 test/test_launcher_regression.py                        # older launcher checks
# real binaries (build them as in the PR description), fixture model, fake GitHub:
APP_HOME=/tmp/mb BIN=/tmp/e2e/bin SEED=/tmp/e2e/seed CORE_BIN=tass-core-int python3 test/mailbox_native.py
APP_HOME=/tmp/e2e/learn BIN=/tmp/e2e/bin SEED=/tmp/e2e/seed OUT=/tmp/shots CORE_BIN=tass-core-int python3 test/browser/learning_e2e.py
APP_HOME=/tmp/e2e/pulse BIN=/tmp/e2e/bin SEED=/tmp/e2e/seed CORE_BIN=tass-core-int python3 test/pulse_multicycle_native.py
```

Actions (pushed commits): [core-tests fafea55](https://github.com/Arameair/Thought-Processor/actions/runs/42108803165) ·
[core-tests 2574a0a](https://github.com/Arameair/Thought-Processor/actions/runs/42105270994) ·
[one-app-launcher-regression 2574a0a](https://github.com/Arameair/Thought-Processor/actions/runs/42105271035) (includes the #79 tests).

## 4. Recovery procedure (#79)

Verified with fault injection (fake Docker, real `tass.sh`); **not yet on real Docker**.

| Situation | What `tass.sh` does now | Owner command |
|---|---|---|
| Image build fails | Nothing stopped, nothing changed | none |
| Backup can't be written or doesn't match the data | Update refused; old version started again; no partial backup kept | none (free disk space, retry) |
| New version unhealthy, or records shrank | Old version + data from the verified backup put back; damaged data moved to `deploy/failed/` (kept) | none |
| Power cut / terminal closed during update, rollback or restore | `deploy/operation.env` remains; `start`, `update`, `rollback`, `restore-previous` refuse; `status` says INTERRUPTED | `./tass.sh recover` |
| Rollback to a version that won't validate | Returns to the version from before the rollback | none |
| "Put everything back the way it was before the last update" | | `./tass.sh restore-previous` (saves current data first in a new verified backup; undo with `./tass.sh rollback --restore-data <that backup>`) |
| See versions, backups (verified / CHANGED / older unverified), history | read-only | `./tass.sh history` |

Backups are never pruned. A backup counts only after it was unpacked to a scratch folder and compared
file by file with the stopped data; its SHA-256 is stored next to it and re-checked before any restore.

The owner's *installed* `tass.sh` is the old one. To use the new safety without switching the owner's
branch, run the candidate script from its own worktree; it manages the same `~/.tass-app`:

```bash
cd ~/path/to/Thought-Processor            # the owner's existing clone
git fetch origin
git worktree add ../tp-candidate fafea55ac1bb91b00df3497bb1ac324c4cc67ffa   # (use the doc commit's SHA if newer)
cd ../tp-candidate/core
```

## 5. P4: real Engine 20-second pulse, many cycles (Real Go Services, fixture model)

Engine `Pulse.Tick` (every 20 s, `ticks.jsonl`) and Core `SyncOnce` (every 20 s, GitHub + lessons +
mailbox) are **separate clocks**. Core sync is not thinking. The pulse was started by the sandbox owner
through Core with Autonomous ON *in the throwaway sandbox only*; nothing does this on the owner's machine.

| Check | Result |
|---|---|
| Engine starts PAUSED; nothing ticks by itself | PASS |
| Timer ticks 20.0 s apart; sequence +1, no gaps or repeats, including across an Engine kill/restart | PASS (35 ticks) |
| Idle ticks: READY "nothing pending", **zero** model requests | PASS |
| Long job: START once, CONTINUE ("not started again"), finished job OBSERVED by a later tick, one model run | PASS |
| Bounded queue: 20 waiting accepted, 21st refused (HTTP 429) | PASS |
| Identity LOCKED: ticks BLOCKED, job WAITs queued; Cognition answers 403 instead of guessing; unlock → runs in order | PASS |
| Model down 80 s: ticks keep coming every 20 s; Core health names the model; the job in hand is shown CONTINUE (it waits for the model, it is **not** failed) and ends DONE once the model is back, never started twice | PASS (run 2) |
| GitHub down: Core reports a sync problem; Engine still ticks every 20 s | PASS |
| Engine killed mid-job: that job INTERRUPTED, never started again; Core turns the pulse back on | PASS (36 ticks, sequence 1..36 unbroken) |
| No job STARTed twice in the whole run | PASS |
| Cognition pulse = Engine (control, 20 s, latest tick ID) | PASS |
| Pause: no ticks afterwards | PASS |

**Run 2: 24/24 PASS** (`pulse_multicycle_native.py`, ~12 min, 37 ticks, 23 jobs).
Run 1 was 23/24: the failing check assumed the Engine would fail a job while the model is down. It
doesn't: the job keeps the model slot and waits (CONTINUE every tick) until the model answers. The check
was rewritten to measure what must hold (ticks continue, the job ends once, never re-run) and passed.
Owner note: if Nemotron is down, one pulse job waits rather than failing, and later jobs queue behind it.

## 6. Owner checklist (in order, one step at a time)

Stop at the first step that does not match and report it. Do not enable Autonomous, start the pulse,
or turn on Talk on GitHub unless a step says so.

1. **Read-only status** (changes nothing): `cd ~/path/to/Thought-Processor/core && ./tass.sh status`.
   Write down Core/Engine/Identity versions, Identity state, action mode, pulse.
2. **Read-only history with the new script**: create the worktree (§4), then
   `cd ../tp-candidate/core && ./tass.sh history`. Expect your current and previous versions, your
   existing backups listed as "unverified (older)", and no INTERRUPTED operation.
3. **Update** (owner decision): `sudo -v` (if Docker needs sudo), then
   `./tass.sh update fafea55ac1bb91b00df3497bb1ac324c4cc67ffa`. Expect "Backup (verified, SHA-256 recorded)"
   then UPDATED, or an automatic ROLLED BACK. Unlock Identity on 127.0.0.1:8796.
4. **Recovery drill** (optional, recommended): `./tass.sh history` shows "Restore previous update
   available". Only if you want to rehearse: `./tass.sh restore-previous`, check status, then
   `./tass.sh rollback --restore-data <the before-restore backup it printed>` to come back.
5. **Real Nemotron lesson** (Development → Teach & test): pick *My work pulse* → Study → wait for the
   grade → read TASS's raw answers → Approve only if you agree → **Test again** with a question of your
   own, e.g. `If Identity is locked when my pulse ticks, what do I report? => blocked`. Restart
   (`./tass.sh stop && ./tass.sh start`, unlock), open Teach & test again: LEARNED, attempts, approval
   and retest are still there; Test again with a second new question. Check the lesson card in
   Cognition (attempts with job/cycle IDs, retests, "no skill from learning").
6. **First manual GPT-6 mailbox exchange**: Core page → turn **Talk on GitHub** on (Autonomous stays OFF).
   Wait one 20 s sync (it primes on existing comments; none are answered). As `Arameair`, post on
   [Arameair/TASS#24](https://github.com/Arameair/TASS/issues/24):
   `ChatGPT: MAILBOX-01 — please acknowledge in one sentence; do not run tools`.
   Within ~40 s TASS replies on #24 with `Source: … comment <id>` and `Cycle: tp_…`. Turn Talk on GitHub off.
7. **20 s validation** (only when you choose): with Autonomous OFF, open Cognition: pulse PAUSED and
   "every 20 s". If you decide to watch real ticks, start the pulse for a few minutes, confirm READY
   ticks 20 s apart in Cognition, then Pause. Leave Autonomous OFF.

## 7. Result table

| Item | CI | Fixture | Real Go Services | Real Nemotron | Owner |
|---|---|---|---|---|---|
| #79 update manager | PASS | PASS 14/14 | n/a (needs Docker) | NOT RUN | NOT RUN |
| Mailbox round trip + failure modes | PASS | PASS | PASS 20/20 | NOT RUN | NOT RUN |
| Lesson study → grade → approve → restart → fresh-task retest | PASS | PASS | PASS 21/21 | NOT RUN | NOT RUN |
| Development Teach & test + Cognition evidence | — | — | PASS | NOT RUN | NOT RUN |
| #78 editor refusals + Save refusal | PASS | PASS | PASS | n/a | NOT RUN |
| Real Engine 20 s multi-cycle | — | — | PASS 24/24 | NOT RUN | NOT RUN |
| Five-card thought design | — | — | NOT BUILT | — | — |
