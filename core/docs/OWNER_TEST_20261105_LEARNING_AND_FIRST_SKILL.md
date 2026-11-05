# Owner test — November 5, 2026: lesson retry repair (#71) and first @ skill (#72)

Two checkpoints. Each one stands on its own: test Checkpoint 1 first, then update to Checkpoint 2.

| | Core (Thought-Processor) | Engine | Identity | PR |
|---|---|---|---|---|
| Installed now | `3ff2a52cf1d948e3283f0d72f60052521f7f5e41` (alpha) | `21b3d502775b3ed065ab445b88f51909500770b3` | `91e1521659a31db79044a44bd33bce48178554d9` | — |
| **Checkpoint 1**: #71 fix | `9be30a502741dcbe3ad6589794ae61776c18681f` · branch `experiment/p0-lesson-retry-history-20261105` | same `21b3d50` | same `91e1521` | Core #73 |
| **Checkpoint 2**: @ skill + workspace | head of `experiment/p0-first-skill-20261105` (full SHA in its PR) | same `21b3d50` | `87dd9243c721fd11701575033d95d3dcf286f765` · branch `experiment/skill-grant-lesson-workspace-20261105` | Core PR on top of #73 · Identity PR |

`./tass.sh update` reads the Engine and Identity pins from the candidate's `core/tass.sh` and builds them. You don't install them separately. No new model, no paid API, no Autonomous, and no merge to alpha.

## What was tested here, and what was not

| Evidence | Result |
|---|---|
| Go `vet` + `test -race` (fixtures; Engine fixture refuses duplicate job IDs like the real one) | PASS |
| Bug reproduced on unmodified alpha: Fail → Retry → Study → `409 … already exists` | REPRODUCED |
| `core/test/lesson_retry_native.py`: real Engine 21b3d50 + real Identity 91e1521 binaries, fixture model (fails lesson 01 REMEMBER-only, like your Nemotron), fake GitHub. Old Core reproduces the 409; the new Core on the same data runs `_2`, keeps `_1` in history, PASSED → Approve → LEARNED, survives a full restart | **24/24 PASS** (also 24/24 with the Checkpoint 2 Core) |
| `core/test/skill_native.py`: entity made on Identity 91e1521, Identity upgraded in place to 87dd924, real Engine, Checkpoint 2 Core. Denied before grant, wrong passphrase / unknown skill refused, enable → request → approve → DONE (real output, MET), repeat approve doesn't re-run, revoke → DENIED with nothing run, restart keeps receipts, fresh conversation reuses the skill, workspace file allowlist | **22/22 PASS** |
| `core/test/browser/workspace_skill.py`: real Chrome drives the Identity page and the Core page (enable skill, Talk "Find lessons using @", Approve run, view file, ask about file) | **6/6 PASS** |
| `./tass.sh update` with Docker images | **NOT RUN** (this sandbox can't pull base images) |
| Your real Nemotron | **NOT RUN**: that's the test below |

## Before anything: backup and read-only check (safe, changes nothing)

```bash
cd ~/tass-one-app/core
./tass.sh status                      # note the Core/Engine/Identity SHAs and the ports it prints
./tass.sh backup
git fetch origin experiment/p0-lesson-retry-history-20261105:refs/remotes/origin/experiment/p0-lesson-retry-history-20261105
./tass.sh check-update origin/experiment/p0-lesson-retry-history-20261105
```

Check that `check-update` prints Candidate core `9be30a502741dcbe3ad6589794ae61776c18681f`, engine `21b3d502…` and identity `91e1521…`. Stop if anything differs. Keep **Autonomous OFF** and the work pulse **paused**.

## Checkpoint 1 — lesson retry repair (#71)

```bash
./tass.sh update 9be30a502741dcbe3ad6589794ae61776c18681f
./tass.sh status
```

Then unlock TASS on Identity's page (same passphrase; do not recreate anything).

1. **A: study My work pulse.** On the Learning card, *My work pulse* should still show your earlier attempt under **Attempt history**. Press **Study (manual)**. Expect a job ending in `_2` or higher, with no 409. Wait for the result. You'll see the three real answers and a grade, or `MODEL_FAILED — NOT GRADED` with the Engine's error. Either way, the attempt is added to the history.
2. **B: approve only on PASSED.** If it says **PASSED 3/3**, press **Approve**. If it says RETRY or MODEL_FAILED, press **Study again**. It must take a new job number, never a 409. Send me the job ID and the error text.
3. **C: restart.** Run `./tass.sh stop`, then `./tass.sh start`, and unlock Identity. *How I learn*, *Building together* and (if approved) *My work pulse* are still LEARNED, the history is still there, and `tass.md` lists each lesson once.
4. **D: does he use what he learned?** Press **New conversation**, then ask TASS: *"How many seconds are there between two of your pulses, and what do you say if a job is still running?"* and *"Who has to approve a lesson before it counts?"* A correct answer uses the approved lessons (20 / CONTINUE; you). An answer that doesn't is a real result too: send it to me.

## Checkpoint 2 — first @ skill (#72) and the workspace (#31)

```bash
./tass.sh backup
git fetch origin experiment/p0-first-skill-20261105:refs/remotes/origin/experiment/p0-first-skill-20261105
./tass.sh check-update origin/experiment/p0-first-skill-20261105   # engine 21b3d502…, identity 87dd9243…
./tass.sh update <the full Candidate core SHA it printed>
./tass.sh status
```

Unlock Identity again with the same passphrase. Same entity.

5. **Denied first.** In Talk type `Find lessons using @`. Because the skill isn't granted yet, the **Workspace** card shows a receipt **DENIED** that says what to do. Nothing ran.
6. **Grant.** On Identity's page, under **Skills TASS may run**, type your passphrase and press **Enable this skill**. It now shows **GRANTED**.
7. **Request, approve, evidence.** Type `Find lessons using @` again. TASS answers which skill he picked and why, and the receipt waits as **AWAITING_APPROVAL**. Press **Approve run**. Expect **DONE · executed: yes, once**, Identity's decision ID, the real list of your lessons, the output hash, and *Expected / Observed → MET*.
8. **No double run.** Press Approve on the same receipt again (or restart). Nothing runs a second time, and the receipt is unchanged.
9. **Revoke.** On Identity's page press **Revoke**. Make a new request and approve it. Expect **DENIED**, executed: no, no output.
10. **Fresh conversation.** Enable the skill again. Press **New conversation** and ask *"What lessons are in your workspace? Use your skill."* If Nemotron writes `@lesson-workspace.list-lessons` in his reply, a request appears for you to approve; that would be TASS choosing the skill himself. If he doesn't, type `@` yourself. Note which happened.
11. **Project file.** In Workspace → Project file, choose `lessons/claude/01-my-work-pulse.md` and press **View**. You'll see the read-only content, `Arameair/TASS@live`, and the local and GitHub blob SHAs (they should say MET). Ask TASS a question about it with **Ask TASS**.

Send for each step: PASS / FAIL / NOT RUN, receipt or job ID, and any error text. Never send passphrases or tokens.

## Rollback

```bash
cd ~/tass-one-app/core
./tass.sh rollback            # back to the version you were on before the last update
./tass.sh status
```

- From Checkpoint 2 to Checkpoint 1: skill receipts stay on disk but aren't shown. Identity 91e1521 ignores the `skill.*` grants.
- From Checkpoint 1 to alpha `3ff2a52`: lesson statuses and `tass.md` are kept. The old Core drops `job_seq` and `history` the next time it saves, and the #71 bug returns.
- To put the data back exactly as it was before an update: `./tass.sh rollback --restore-data <backup file printed by ./tass.sh backup>`. Anything learned after that backup is lost.

## Not implemented / not verified

- **Not implemented:**
  - Editing files, proposed changes, running tests or notebook cells from the workspace.
  - Any skill besides `@lesson-workspace.list-lessons`.
  - TASS choosing a skill without a visible `@` (he can only propose one by writing it in his reply).
  - The GitHub issue relay from #69.
- **Not verified:**
  - Your real Nemotron on any of this.
  - The Docker `./tass.sh update` build for these SHAs.
  - Whether Nemotron writes `@skill` on his own (step 10).
