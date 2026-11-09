# Stage 2 implementation: handoff for Codex's independent audit (2026-11-09)

**Scope.** Audit the implementation of the Stage 2 design, revision 4: `core/docs/STAGE2_NUDGE_PROCESSING_DESIGN_20261108.md` at `3d40808`, PR #101, decisions D1–D16. Codex gave that design a DESIGN PASS. The goal is ONE owner-approved nudge, addressed to TASS himself, that produces ONE verifiable Thought development.

**Rules.** This is an audit only. Do not merge, deploy, change Identity grants, enable autonomy, touch the owner's installation, or process real nudges. Report PASS, FAIL, BLOCKED and NOT RUN separately. A test that needs Docker is never PASS in a workspace without a Docker daemon.

## What to audit
| Repo | Draft PR | Branch | Commit | Base |
| --- | --- | --- | --- | --- |
| Arameair/identity | #43 | `experiment/nudge-process-stop-20261108` | `cfc4079` | `64fdf8e` |
| Arameair/Thought-engine | #19 | `experiment/stage2-nudge-run-20261108` | `02df481` | `d69d391` |
| Arameair/Thought-Processor (core) | #104 | `experiment/stage2-nudge-processing-20261108` | head of the PR | `094f43b` |

## Focus questions (implementation vs design)
1. **Single use (G1).** Can any path send more than one model request for one nudge ID? Paths to check:
   - concurrent Process calls;
   - Core restart;
   - Engine restart or `kill -9` at each crash point;
   - a replay after a revoke;
   - Identity restart between mint and redeem (F16).

   Relevant code: Engine `nudge.go` (`ProcessNudge`, `runNudge`, `createOnce`, `ScanNudges`); Core `tass/nudge_run.go` (`ProcessNudge`, `dispatchNudge`).
2. **Authorization.**
   - Is every decision bound (binding struct order and fields), minted only by `core`, redeemed only by `engine`, single use, with a 120 s TTL, held in memory, and audited?
   - Does C2 come before C3, so that a replay needs a fresh decision?
   - Does Stop need UNLOCKED but no grant (D15)?
   - Do the Engine key alone, a forged decision, another entity, another Thought, a wrong digest, or a never-submitted nudge leave **no folder and no file**?
3. **P1/P2.** Are the Identity check and the Thought gate re-run immediately before HTTP (after the model slot is taken) and immediately before the save? Does a revoke mid-call produce FAILED with no development?
4. **Verified development.** Is COMPLETED shown only for exactly one note and one `developed` step that carry `source_nudge_id` and the request ID? Is the attention stored only as a digest in `calls.jsonl` (D10)? Is the plain `think` route unchanged (D11)?
5. **Restore (G2).**
   - In `tass.sh restore_data`, is there any crash point where a published `data/` holds restored drafts without a durable fence?
   - Does any caller bypass `restore_data`?
   - Do Core's fence checks lock processing on a prefix change, truncation, an invalid record, a duplicate ID or an unreadable fence?
   - Do the shell (`restore_fence_write`) and Core (`nudgeFence`) agree byte for byte?
6. **Locks.** Are the `.core.lock` and `thought-engine.lock` flocks held for the life of the process? Is a second instance refused across PID namespaces (D16)?
7. **UI.** Does the confirm step send the saved `body_digest`? Do PRE_RESTORE, UNCERTAIN and RECOVERY_REQUIRED offer only Copy as new (with a warning), never Process?

## How to reproduce (no Docker needed)
```sh
# build the three binaries
(cd identity && go build -o /tmp/e2e/bin/tass-identity-s2 ./cmd/tass-identity)
(cd Thought-engine && go build -o /tmp/e2e/bin/thought-engine-s2 .)
(cd Thought-Processor/core && go build -o /tmp/e2e/bin/tass-core-s2 ./cmd/tass-core)
# unit + race
go test -race ./...            # in each repo
# native end to end (TEST FIXTURE model, fault proxy; ~40 s)
cd Thought-Processor/core && APP_HOME=/tmp/e2e/s2 BIN=/tmp/e2e/bin SEED=/tmp/e2e/seed python3 test/nudge_process_native.py
# real tass.sh restore paths (fake docker)
python3 test/test_updater_recovery.py
```
`SEED` must hold `tass.md` and `lessons/` (any small copy works for this test).

## Claude's own results (2026-11-09, this workspace, no Docker daemon)
| Check | Result |
| --- | --- |
| Go vet + test + race in all three repos | PASS |
| `nudge_process_native.py` (real binaries, fixture model) | PASS 61/61, run 3 times |
| `test_updater_recovery.py` | PASS 16/16 |
| `test_launcher_regression.py` | PASS 6/6 |
| Browser fixtures: process, nudge, go | PASS |
| `nudge_native` | PASS 25/25 |
| `mailbox_native` | PASS 20/20 |
| `pulse_multicycle_native` | PASS 24/24 |
| `skill_native` | FAIL 21/22: the seed has 4 lessons, the test expects 3. Same failure on the base binaries (pre-existing). |
| Two instances, two PID namespaces, one data folder (Core and Engine) | PASS (second refused) |
| Docker images, `tass.sh start`/`update` for real, owner machine with Nemotron | NOT RUN |

## Known limits (not defects of the design)
- G1 and G2 are per-ID guarantees. **Copy as new** after UNCERTAIN or PRE_RESTORE can repeat the task under a new ID. The page warns about this and asks for a new approval (D13/D14).
- A crashed restore leaves `deploy/restore-stage.*` behind. The next `restore_data` removes it; operations hold the op lock.
- After `recover`, which re-runs the restore from the guard backup, drafts are fenced again. This is conservative and intended.
