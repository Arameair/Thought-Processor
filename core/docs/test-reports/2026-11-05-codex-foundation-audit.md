# Independent TASS Foundation Audit — 2026-11-05 candidate

**Audit completed:** 2026-11-06 (America/Chicago)  
**Candidate:** `Arameair/Thought-Processor@8e040b33e7a492f042a1f4449b8c7b013c227965` (PR #88)  
**Pinned Engine:** `Arameair/Thought-Engine@d69d3917d1601d806d4b2ba2254693e1adb92b6b` (PR #15)  
**Pinned Identity:** `Arameair/Identity@7da7f7c5bbecc42d3a7e9316a2037243eda10b88` (PR #40)  
**Verdict:** **FOUNDATION PARTIAL**

The integrated source/fixture foundation is verified. No product failures remained in the final isolated runs. The foundation is not yet owner-accepted because real Docker recovery, real Nemotron, real GitHub delivery, owner-machine UI/runtime acceptance, sustained resource measurement, and a real multi-identity binary run were deliberately not performed.

## Safety boundary

- The owner installation was not changed, restarted, deployed, unlocked, or granted anything.
- The owner installation's pulse and Autonomous settings were not touched.
- All real binaries ran in `/tmp/codex-tass-foundation-audit-20261105` with throwaway data.
- Fixture services used audit-only ports `18895`, `18896`, `17798`, `18081`, and `28998`.
- Autonomous was enabled only inside the disposable pulse fixture and was paused before exit.
- GitHub and the model were fixtures unless a row explicitly says otherwise.

## Candidate manifest

| Component | Revision | Evidence role |
|---|---|---|
| Thought-Processor integrated candidate | `8e040b33e7a492f042a1f4449b8c7b013c227965` | PR #88; resolves the Core/Cognition/mailbox/updater integration |
| Thought-Processor Cognition base | `1ec138de22acfe63147e24ff7587ec3aeb4cda22` | PR #87 |
| Thought-Processor mailbox source | `e34085eb0e3e20c0e59d88937a92188b28562fc8` | PR #81 |
| Thought-Engine | `d69d3917d1601d806d4b2ba2254693e1adb92b6b` | PR #15 |
| Identity | `7da7f7c5bbecc42d3a7e9316a2037243eda10b88` | PR #40 |
| Historical #71 affected Core | `3ff2a52cf1d948e3283f0d72f60052521f7f5e41` | Reproduction seed only |
| Tracked-file fixture seed | `Arameair/TASS@df5c3cca1e9ddc0b4100f12dea7b62c031d4f676` | Fake GitHub content only |

## Result matrix

| ID | Area | Environment | Status | Evidence |
|---|---|---|---|---|
| A01 | Core static analysis | Source/offline, PR #88 exact | PASS | `go vet ./...` |
| A02 | Core unit/integration suite | Source/offline, PR #88 exact | PASS | `go test ./...` |
| A03 | Core race suite | Source/offline, PR #88 exact | PASS | `go test -race ./...` |
| A04 | Engine static analysis | Source/offline, PR #15 exact | PASS | `go vet ./...` |
| A05 | Engine unit/integration suite | Source/offline, PR #15 exact | PASS | `go test ./...` |
| A06 | Engine race suite | Source/offline, PR #15 exact | PASS | `go test -race ./...` |
| A07 | Identity static analysis | Source/offline, PR #40 exact | PASS | `go vet ./...` |
| A08 | Identity unit/integration suite | Source/offline, PR #40 exact | PASS | `go test ./...` |
| A09 | Identity race suite | Source/offline, PR #40 exact | PASS | `go test -race ./...` |
| A10 | Cognition/Junction end to end | Real candidate binaries, fixture model/GitHub, Chrome | PASS | 11/11; read-only playback, proposal staging, Identity DATA, later Engine consumption |
| A11 | Historical lesson retry #71 | Real candidate binaries, fixture model/GitHub | PASS | 24/24; reproduced old duplicate ID, candidate used `_2`, restart and history preserved |
| A12 | Supervised skill execution | Real candidate binaries, fixture model/GitHub | PASS | 22/22; denial, approval, execute-once, revoke, restart, traversal refusal |
| A13 | Mailbox round trip and durability | Real candidate binaries, fixture model/GitHub | PASS | 20/20; exact source/cycle IDs, dedup, LOCKED hold, outage recovery, AMBIGUOUS no-resend |
| A14 | Learning, retest, Cognition, graph guards | Real candidate binaries, fixture model/GitHub, Chrome | PASS | 21/21; truthful failure, new retry ID, approval, fresh retest, restart, invalid-edge refusal |
| A15 | Controlled 20-second pulse | Real candidate binaries, fixture model/GitHub | PASS | 24/24; 39 ticks, 22 jobs, queue bound, outages, kill/restart, no duplicate starts, pause |
| A16 | Updater recovery | Real `tass.sh`, fake Docker | PASS | 14/14; rollback, power cuts, tampering, locking, backup/restore |
| A17 | Raw PR #87 + PR #81 merge | Source integration before PR #88 | BLOCKED | Conflicts in shared `core.go` sync/cycle/reply paths; not silently resolved. PR #88 is the tested resolution. |
| A18 | Real Docker updater/recovery | Owner-like Docker | NOT RUN | OWNER TEST REQUIRED |
| A19 | Real Nemotron behavior/quality | Real model | NOT RUN | OWNER TEST REQUIRED |
| A20 | Real GitHub GPT-6 mailbox delivery | Live `Arameair/TASS#24` | NOT RUN | OWNER AUTH REQUIRED; fixture proves plumbing only |
| A21 | Owner running installation and UI | Owner machine/live data | NOT RUN | Deliberately untouched |
| A22 | Sustained CPU/memory/disk bounds | Owner-like runtime | NOT RUN | Functional queue/timing bounds passed; resource budget was not measured |
| A23 | Real multi-identity binary isolation | Real binaries with multiple identities | NOT RUN | Unit coverage passed; owner-like integration remains |

**Matrix totals:** 16 PASS, 0 FAIL, 1 BLOCKED, 6 NOT RUN.  
**Scripted acceptance totals:** 136/136 assertions passed in the final corrected isolated runs, plus 9/9 `vet`/`test`/race commands.

## Detailed findings

### Cognition and the thought path

- Browser playback and log inspection remained read-only.
- A TASS reply entered the Junction, Identity returned DATA, and a later Engine cycle consumed it once.
- Cognition showed staged/proposed state without claiming an effect was measured.
- Same-name entity isolation, rename/recreate behavior, canonical identity IDs, revision conflicts, restore, stale validation, and read-only saved brains passed Core tests.
- The Development editor rejected `uses` to a non-skill and rejected a reverse `next` edge that would form a cycle before changing the graph.

### Learning and supervised execution

- The historical duplicate lesson-job condition was reproduced from Core `3ff2a52`; the candidate preserved history and allocated the next ID.
- A malformed model response was recorded as `MODEL_FAILED` and **NOT GRADED**.
- `PASSED` did not become `LEARNED` until explicit owner approval.
- Fresh-task retests used approved-knowledge context with lesson text withheld and survived a full service restart.
- Learning did not grant or execute a skill. The separate supervised-skill flow required approval, executed once, retained receipts, respected revoke, and refused traversal.

### Mailbox

- Default remained mailbox OFF and `CHAT_ONLY`; enabling mailbox did not enable pulse or Autonomous.
- Only the allowlisted author and issue were consumed.
- Replies carried the exact input comment ID and Engine cycle ID; audit storage was metadata-only.
- Restart/re-poll did not duplicate a reply.
- Identity `LOCKED` held work until unlock; GitHub read failure surfaced a visible hold.
- A lost POST acknowledgement became `AMBIGUOUS`, invented no outgoing ID, was never blindly resent, and required owner review.

### Pulse and recovery

- Pulse started `PAUSED`; it was started only inside the sandbox fixture.
- Idle timer ticks were approximately 20 seconds apart and made no model request.
- Long work started once and produced `CONTINUE` rather than a second start.
- The queue accepted 20 waiting jobs and refused the 21st with HTTP 429.
- `LOCKED` produced `BLOCKED` ticks and preserved queued work.
- Ticks continued through model and GitHub outages; Core named the model/sync problem.
- Core SyncOnce and Engine Pulse.Tick were separately observable 20-second clocks.
- Killing Engine mid-job produced `INTERRUPTED`; the job was not rerun, pulse sequence had no repeats or gaps, and Cognition matched Engine state.
- Pause produced no later ticks.
- The updater fixture passed candidate-health failure, data damage, failed/tampered backup, rollback interruption, restore, and exclusive-lock cases. This is not a real-Docker acceptance result.

## Harness discrepancies preserved

These were audit setup mismatches, not silently counted as product passes:

1. A first Cognition seed copied `.git` binary content; rerun used tracked files only and passed 11/11.
2. A first #71 comparison used a later historical Core that already contained the repair; rerun used the exact affected `3ff2a52` revision and passed 24/24.
3. A first skill seed contained four Markdown lessons while its fixture expected three; rerun used the intended three-file seed and passed 22/22.
4. The first learning browser run passed 20/21 because its request-path recorder split on normal port `8795` while the safety wrapper used `18895`. The displayed requests were all lesson endpoints. After correcting only that audit-port normalization, the unchanged candidate test passed 21/21.

The fixture model and Engine readiness probes printed `healthy=False` during startup because the candidate readiness endpoint reports model readiness rather than merely TCP process presence; the acceptance runners continued and every final model-dependent assertion passed.

## Reproduction commands

Run only in a disposable checkout/data directory. The audit used a wrapper that rewrites fixture ports away from the owner stack.

```text
# Each Go module
go vet ./...
go test ./...
go test -race ./...

# Core updater fixture
python3 test/test_updater_recovery.py -v

# Real-binary fixture suites (environment abbreviated)
python audit_harness/run_cognition_junction_alt_ports.py
AUDIT_MODULE=lesson_retry_native python audit_harness/run_native_alt_ports.py
AUDIT_MODULE=skill_native python audit_harness/run_native_alt_ports.py
AUDIT_MODULE=mailbox_native python audit_harness/run_native_alt_ports.py
AUDIT_BROWSER_MODULE=learning_e2e python audit_harness/run_browser_alt_ports.py
AUDIT_MODULE=pulse_multicycle_native python audit_harness/run_native_alt_ports.py
```

Required environment values were isolated `APP_HOME`, `BIN`, tracked-file `SEED`, exact candidate binaries, and `AUDIT_CORE_TEST`. The port wrapper changed only fixture process URLs and the learning fixture's request-path normalization.

## Recommendation

Keep the candidate in draft and preserve `CHAT_ONLY`/pulse-off on the owner installation. The next owner action should be one supervised owner-machine acceptance window, in this order: real Docker recovery with a verified backup, real Nemotron learning/retest, real GitHub mailbox canary with explicit authorization, then a time-boxed pulse run with CPU/memory/disk measurement and an immediate pause/rollback path. Do not merge or deploy solely from fixture evidence.


