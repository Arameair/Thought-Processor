# Independent Stage 2 implementation audit — 2026-11-09

**Overall: FAIL — one implementation blocker.** The durable run-once, bound-decision, atomic admission and staged-restore mechanisms are feasible and their isolated tests pass. The Engine does not, however, bind its P2 authorization and Thought-state check to the durable Thought write. A grant revocation or Thought resolution in the implementation's own `before_save` window still produces a note, a developed step and `COMPLETED`.

This is a source/offline audit. It did not deploy, merge, restart TASS, change a grant, enable autonomy, or process a real nudge.

## Exact scope

| Component | Base | Tested head |
| --- | --- | --- |
| Core | `094f43ba2bc167ace8835bf286f37951fee432c9` | `097a74ba11134ec095be9a44173efe90172d9e6e` (PR #104) |
| Thought Engine | `d69d3917d1601d806d4b2ba2254693e1adb92b6b` | `02df4811dca9a21e1bdc211f40913ade26ba0053` (PR #19) |
| Identity | `64fdf8e8cf926df5ea6bb6c23fbab864d0f09e3b` | `cfc4079b690e2f7dc57afb9f14c6832ae38fdf54` (PR #43) |

Core pins the two tested dependency heads in `core/tass.sh:34-36`.

## Findings

### ST2-I01 — FAIL — P2 is not bound to the save

Engine `nudge.go:582-591` defines P2 as an Identity check followed by `thoughtGate`. The run calls it at `nudge.go:690-696`, then executes the `before_save` hook at `nudge.go:697-699`. Only afterwards does it reacquire `e.mu` and write the Thought at `nudge.go:700-751`. That write checks only that the Thought still exists; it does not re-check the processing grant or reject a now-resolved Thought.

Two independent fixture regressions against the exact Engine head both fail:

```text
go test -run 'TestAuditNudge(GrantRevoked|Resolved)AfterP2BeforeSave' -count=1 -v

grant revoked in before_save:
  got State=COMPLETED, Evidence Notes=1 Steps=1
resolved Thought in before_save:
  got State=COMPLETED, Evidence Notes=1 Steps=1
FAIL
```

The tests use the existing Engine `before_save` hook. One revokes `cognition.nudge.process`; the other calls `ThoughtAction(..., "resolve")`. Both changes happen after P2 and before the durable write.

**Smallest required correction:** place the fault/crash hook before P2, perform the Identity check after that hook, then acquire the Engine mutex and re-check the local Thought gate while holding that mutex through the atomic Thought write. Add both regressions. The Identity check is the external authorization linearization point; the local mutex binds the Thought state to the save. No new architecture is required.

### ST2-I02 — PASS — durable run-once and Stop/void admission

Source inspection and existing tests support the per-nudge run-once guarantee:

- Engine `createOnce` uses `O_EXCL`, file sync and parent-directory sync (`nudge.go:211-244`).
- `ProcessNudge` consumes the exact bound decision before replay and creates the immutable claim before running (`nudge.go:480-567`).
- `VoidNudge` redeems a bound Stop decision and races on the same immutable claim (`nudge.go:799-834`).
- Core serializes C1-C5 for all IDs with `admitMu`, writes `submitted.json` durably, and only the creator dispatches (`core/tass/nudge_run.go:386-478`).
- Core's lifetime `.core.lock` is a non-blocking `flock` (`core/tass/datalock_unix.go:12-26`).
- Existing concurrency, Stop/void, torn-claim and crash-state tests pass under the race detector.

This is fixture/source evidence, not a Docker or namespace execution in this audit.

### ST2-I03 — PASS — bound authorization and replay ordering

Identity restricts mint to Core, redeem/check to Engine, validates a 64-hex binding, consumes once and uses a 120-second TTL (`internal/entity/bound.go:23-35,64-70,89-170`). Stop requires UNLOCKED but no process grant. Core performs C2 before C3, so an existing-run replay still needs a fresh decision (`core/tass/nudge_run.go:414-430`). Cross-entity, wrong Thought, wrong digest, forged decision and revoked replay cases are covered by passing unit tests.

**Evidence limitation (OPEN, not the blocker):** Identity audit appends are best-effort and ignore open/write/close errors, with no file sync (`internal/entity/bound.go:173-181`). Enforcement still fails closed, and Engine receipts retain decision IDs, but a storage failure can remove the durable Identity-side audit trail for an otherwise authorized call.

### ST2-I04 — PASS — staged restore and byte fence

Every located data-restoration caller uses `restore_data`. It extracts to staging, writes and syncs `restore-fence.json`, syncs staging, then publishes with same-filesystem renames (`core/tass.sh:336-384`). Any published restored `data/` therefore already contains its fence. An interruption before publication leaves current live data; interruptions after the data rename publish a fenced directory and leave an operation marker that blocks other mutating launcher commands until recovery.

Core verifies the SHA-256 of exactly `raw[:journal_bytes]`, rejects truncation or a changed prefix, rejects invalid restored records and duplicate IDs, and marks records by starting byte offset (`core/tass/nudge.go:216-247,281-330`). Appending after a restored final line without a newline inserts a newline after the protected byte boundary (`core/tass/nudge.go:344-377`).

The real-shell/fake-service recovery suite passed 16/16, including fence publication and a power cut after staging. Launcher regressions passed 6/6.

### ST2-I05 — PASS — verified completion and privacy in source/tests

Engine recovery derives `COMPLETED` only from one matching note and one matching developed step; Core independently re-reads the Thought and requires exactly 1+1 (`core/tass/nudge_run.go:190-221,304-311`). The Engine logs the attention digest while sending the plain attention only to the model request. These checks pass in unit tests. Actual Nemotron execution was not performed.

## Executed evidence

| Test ID | Command / evidence | Result |
| --- | --- | --- |
| T01 | `go vet ./... && go test ./... && go test -race ./...` in Core | PASS |
| T02 | same in Thought Engine | PASS |
| T03 | same in Identity | PASS |
| T04 | `python3 test/test_updater_recovery.py` | PASS 16/16 |
| T05 | `python3 test/test_launcher_regression.py` | PASS 6/6 |
| T06 | two test-only regressions at Engine's `before_save` hook | FAIL 2/2; both falsely COMPLETED with 1 note + 1 step |
| T07 | browser fixtures | NOT RUN — Playwright is not installed |
| T08 | real-binary native fixture suite | BLOCKED — script hard-codes the owner's live Identity port 8796 |
| T09 | Docker images, container namespace lock test, real update/start | NOT RUN |
| T10 | live owner draft, live Nemotron, real nudge | NOT RUN by safety requirement |

## Native-suite safety note

The native script starts its fixture Identity at hard-coded `127.0.0.1:8796`. On this owner host that address was already occupied. The fixture process logged `bind: address already in use`; the first create request reached the existing Identity and returned 409, and a later authorization returned 401 because the isolated key did not match. The run was immediately aborted and its other fixture processes stopped. No grant was changed, no service was restarted, and no nudge/model request was made. Production journals were not opened to verify further because the audit rules prohibit exposing them. Future audit runs need configurable, dynamically allocated ports.

## Recommendation

Do not merge or deploy Stage 2 yet. Correct ST2-I01 and add the two save-boundary regressions. Then rerun Go/race plus a port-isolated native suite. The restore design and per-ID run-once design do not require expansion based on this audit. Docker/live-Nemotron acceptance remains a separate NOT RUN gate.
