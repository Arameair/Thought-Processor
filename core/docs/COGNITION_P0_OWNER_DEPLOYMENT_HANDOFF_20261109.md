# Cognition P0 — Owner-controlled deployment and acceptance handoff
**Prepared:** 2026-11-09  
**Status:** PREPARED RELEASE CANDIDATE. **NOT INSTALLED**, **NOT OWNER ACCEPTED**. Commands below are **operator runbook**, not evidence of execution.

## Exact candidate
- **Core (this release candidate):** `1cd08b683c674426dd06f8ff7a5fd4d00412580c` on `release/cognition-p0-owner-acceptance-20261109`. The only source difference against independently audited Core `42d4d7d8e4cf83a89594d8fdb64d2d9526906ef8` is one line in `core/tass.sh`: repaired Engine commit pin.
- **Engine pin:** `40d4e91e3b04196fa4e4f55a92b25789b82d8675` (ST2-I02 repair, draft PR [#20](https://github.com/Arameair/Thought-engine/pull/20)).
- **Identity pin:** `cfc4079b690e2f7dc57afb9f14c6832ae38fdf54` (unchanged).
- **Independent evidence:** [Core P0 navigation delta audit](https://github.com/Arameair/Thought-Processor/issues/106#issuecomment-6752771158), Core `42d4d7d` + Engine `40d4e91` + Identity `cfc4079` isolated-native fixture **584/584 in 8 completed runs**, browser Back/Forward 9/9 viewport runs, no model calls from reads. An occupied port produced a safe preflight refusal. This is **not** a Docker or real Nemotron proof.
- **Source of runbook semantics:** `core/tass.sh` in this exact candidate: `check`, `check-update REF`, `update REF`, `status`, `history`, `restore-previous`, `recover`. `update` builds candidate, obtains a verified pre-update backup, swaps, validates, and automatically restores prior version/data if validation fails. `restore-previous` explicitly reverts the last update plus its pre-update verified data; it also saves a verified snapshot of current data first.

## Preflight on the owner's Windows/WSL computer
**Operate from existing TASS checkout and app environment; do not clone over current installation, check out unrelated branches, reset data, or change `TASS_TRACK`.** The owner previously used `~/tass-one-app/core`; verify path still applies. The command `git fetch` below only retrieves an experimental release branch.

```bash
cd ~/tass-one-app/core
./tass.sh status
./tass.sh history
./tass.sh check

# Fetch exactly the prepared branch, without changing working checkout.
git -C .. fetch origin release/cognition-p0-owner-acceptance-20261109
git -C .. cat-file -e 1cd08b683c674426dd06f8ff7a5fd4d00412580c^{commit}

# Read-only candidate comparison: MUST print repaired Engine pin and unchanged Identity.
./tass.sh check-update 1cd08b683c674426dd06f8ff7a5fd4d00412580c
```

**STOP** if the displayed candidate Engine pin is not `40d4e91`, Identity pin is not `cfc4079`, current deployment identity/version differs from expectations, preflight reports `[stop]`, `history` reports an interrupted operation, Docker/model/ports not usable, the checkout is not the expected TASS repo, or backup storage is inadequate. Ask the owner before proceeding. Never invoke `recover` blindly; first inspect why the operation is interrupted.

## Owner-approved application step (run on owner machine, not in GitHub)
Before applying, capture the command output and obtain the owner's confirmation of displayed `Current` / `Candidate`. The script itself asks `Apply this update? [y/N]`.

```bash
cd ~/tass-one-app/core
./tass.sh update 1cd08b683c674426dd06f8ff7a5fd4d00412580c
./tass.sh status
./tass.sh history
```

The update performs its own verified pre-update backup and rollback-on-validation-failure. Optionally perform a separately verified manual `./tass.sh backup` first if desired, noting this temporarily stops/restarts the TASS containers. Never manually modify live journal/Thought files during validation. After an update/restart, Identity is expected to be LOCKED until the owner enters their passphrase.

## Owner's first acceptance test
1. Unlock TASS in the Identity interface through normal owner workflow. Verify current entity link and permissions are correct; do **not** grant new privileges merely to make the page look ready.
2. Open `http://127.0.0.1:8797/cognition` through the existing local app. Observe honest Pulse, Engine, and Identity status. No autonomous Pulse activation required.
3. Open the existing `identity-test` Thought, if present for the authorized entity. Read the original question and the **complete actual persisted Note**. Confirm it is the correct Note, not placeholder/summary. Review provenance (time, author, request/nudge IDs, correlated step, verified receipt where available) and labels saying persistence is not semantic correctness.
4. Close the detail, use browser **Back** and **Forward**, refresh, and follow an existing completed nudge receipt's **Read what TASS wrote** link. Confirm focus returns to the **same selected request/Note**, without duplicate history entries or navigation leaving Cognition unexpectedly.
5. Use a half-width window (approximately 960 px) and phone/narrow view if possible; question and Note must be readable without horizontal scrolling.
6. Verify no model execution or new Note is triggered by these *read-only* interactions, using real records/status where possible. Do not initiate an unsupervised model call merely to test observation.
7. Report PASS/FAIL/NOT RUN with the exact deployed revisions, screenshots of non-sensitive UI, and diagnostic logs if failing. Never paste private Notes/passphrases/tokens into public GitHub issues.

**Stop/review if** Note content incorrect/missing, wrong entity/authorization, misleading nudge status, broken Back/Forward, unverified backup, or any unexpected write/run. This should be documented as a real owner-test finding, not worked around by inventing success labels.

## If update succeeds but owner acceptance fails
Use the **script's verified restore path**, after inspecting its scope with the owner:

```bash
cd ~/tass-one-app/core
./tass.sh history
./tass.sh restore-previous
./tass.sh status
```

**Caution:** `restore-previous` returns to the prior version **and prior data**; it first makes a new verified backup of current data and moves newer data aside. Any post-update changes will no longer be in active data until deliberately reconciled. Do **not** conflate it with `rollback`, which normally changes code version without restoring historical data. If an update was interrupted, use the documented `recover` procedure only after inspecting `./tass.sh status` and operation record.

## Post-acceptance gate
Do **not** mark Gate 0 owner-accepted until the actual installed-stack test passes. Then keep Stage 1 Cognition previews on a separate scoped PR; no P1 Development link, P2 experimental run, new grants, or TASS Presence implementation is authorized by this handoff.
