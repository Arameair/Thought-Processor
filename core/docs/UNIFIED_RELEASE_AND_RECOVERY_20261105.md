# TASS — one integration track, automatic update recovery, owner-controlled restore
**Owner direction:** November 5, 2026  
**Status:** IMPLEMENTATION HANDOFF / DESIGN + SOURCE AUDIT. The branch and this document exist; changes to the updater described below are **not implemented or live-tested**. No deployment was changed.

## 0. Non-negotiable owner behavior

1. There is **one current running TASS stack**, not parallel competing Core installations or separate AI models.
2. The owner should **not manually make a backup before each update**. Clicking/running Update must always make, verify, and retain its own recovery snapshot before touching active data or switching services. Refuse the update if this fails.
3. Update from a **reviewed, pinned commit** of the unified integration track. Keep the original stack running during source retrieval, build, preflight and CI inspection.
4. If a candidate cannot be activated or validated, **automatically put the previous tested version and its exact pre-update data back**, then verify recovery and report genuine success/failure. Never display a success message just because containers started.
5. The owner must always be able to select **Restore Previous Update** without knowing backup paths or commit hashes. Make the choice explicit between (A) the exact pre-update snapshot and (B) code-only rollback with current data when proven compatible. Preserve the current state in a new snapshot before either choice; disclose data written since the older snapshot that may be lost from the *active* state.
6. Preserve an owner-visible chronological deployment history and backups. Never delete the last known-good snapshot or currently selected restore target automatically. Provide safe disk-space/retention reporting.
7. **Identity is the only authorization authority**. Update or restore must not silently grant skills, turn on Autonomous/Pulse/GitHub relay, alter owner policies, enable a scheduler, or start an extra model.
8. All AI collaborators work on feature branches and submit evidence. A new GitHub commit does **not** autonomously deploy to the owner's PC.

## 1. Source audit and starting point

Repositories and roles:
- **Arameair/Thought-Processor**: the deployed TASS Go Core, user interface, Compose bundle and **single update/restore entry point** (`core/tass.sh`).
- **Arameair/Thought-engine**: separately versioned, pinned thought runtime; preserve audit and pulse records.
- **Arameair/identity**: separately versioned, pinned authority. Preserve the existing owner entity, passphrase, grants and decision records across version changes.
- **Arameair/TASS**: companion source document, lessons, GitHub message issue #24; its `live` branch is content, **not** an application release branch. A local backup cannot undo already-published GitHub changes.
- **Arameair/Development**: upstream Brain Builder prototype/design reference; the integrated `/dev` page in Thought-Processor is the running implementation. Do not run a second Development service.

Confirmed owner-machine snapshot on Nov 5:
- Core `60661aa807cac4d93ebc461e1447f3c1e6ad9473`, Engine `21b3d502775b3ed065ab445b88f51909500770b3`, Identity `87dd9243c721fd11701575033d95d3dcf286f765`.
- Containers healthy, real Nemotron working, owner test shows `READY`, `UNLOCKED TASS`, `CHAT_ONLY` after unlock. See [Issue #78](https://github.com/Arameair/Thought-Processor/issues/78).
- Branch `experiment/dev-platform-brain-20261105` contains the integrated Development/Workspace/supervised-skill implementation. Deployment pinned to the SHA above (not floating branch HEAD).
- [Draft PR #77](https://github.com/Arameair/Thought-Processor/pull/77) adds dedicated issue #24 GitHub mailbox + message/authorization/cycle receipts, but head `4fbc26874accd155c0f7f00c8cf3d0b206219c76` forks from *older* Core `9be30a5`. Comparing to `60661aa` shows **DIVERGED**. **Never deploy #77's head over Development.**
- [Issue #78](https://github.com/Arameair/Thought-Processor/issues/78): Development editor permits temporary malformed relations (bad `uses` target / cyclic `next`); server-side `ValidateBrain` already rejects both when Check/Save is called. Fix client-side editing and test server rejection.

### Updater already present; avoid rebuilding it from zero

In deployed `core/tass.sh`:
- `cmd_update` calls `build_version`, stops the current Compose services, runs `backup "before-<sha>"`, records `previous.env`, starts a candidate, and checks health, Core SHA, Identity ID and selected record line counts. On **validation** failure, it attempts to restore the previous environment and pre-update data.
- `./tass.sh rollback` swaps `previous.env` and `current.env` and retains **current data** unless `--restore-data <file>` is supplied.
- `backup()` tarballs `data` and `secrets` but **not** `app.env`, exact source/image manifest, previous/current environment, backup metadata or compatibility information.
- `restore_data` moves the current data/keys aside before extracting. A missing/corrupt archive or interruption after the move is not a transactionally verified restore.
- `cmd_rollback` switches versions and validates, but if that validation fails, it **does not automatically return** to the version the owner just left.
- A failed backup happens **after the old services have been stopped**. A backup error therefore is not guaranteed to leave the old service running.
- `current.env` and `previous.env` are mutable deployment records; automatic rollback only covers paths that reach its explicit failure handler. Power loss / termination in the middle of an update is not comprehensively recovered.
- Existing `TRACK` default is `experiment/integration-20261104`, which is many commits behind the live Development deployment. Do not use this stale default for future check-update operations.

Treat these as auditable gaps, **not** as proof the existing updater is unusable: the owner's recent `9be30a→60661aa` update did succeed and already wrote an automatic backup.

## 2. ONE official working integration lane

**Created branch:** `experiment/tass-unified-integration-20261105`, rooted at owner-running Core `60661aa807cac4d93ebc461e1447f3c1e6ad9473`.

Policy:
- This is the **sole proposed integrated candidate lane** until owner acceptance; it is not promoted `stable` or `alpha` by creating the branch.
- Preserve current deployed version as **last known operational / partial acceptance**. Do not change current containers while merging code.
- Port the *narrow mailbox commit set* from PR #77 onto this branch using reviewable changes/cherry-picks, **not** a whole-branch merge that discards Development, skill receipts or Identity `87dd924` pin.
- Fix #78 editor-side relation creation/editing, add negative backend validation tests and regression tests. Keep the rest of Brain features.
- For changes in Thought-engine or identity, develop on their own feature branches, test independently, and update **full immutable pin SHAs** in the unified Core's `core/tass.sh`. Never float pinned dependencies to `main` or to mutable `alpha`.
- Agent workflow: separate feature branches → PR targeting **unified integration** → CI + review + owner approval for sensitive deployment/policy changes → land tested change → record exact new Core/Engine/Identity manifest → operator sees a candidate update. Feature work can be parallel; **deployment activation is serialized**.
- The owner's live `~/.tass-app` is **one stack**. Agents may test in isolated data/ports/projects; no tests that restart/toggle it without approval.
- Older `alpha`, `stable` and `main` remain unchanged until the owner explicitly decides to promote a validated candidate. An issue/PR marked DONE is not equivalent to accepting a release.
- No blanket auto-merge, direct push to `main`, or silent auto-update on every upstream commit.

Release manifest per candidate: full Core SHA, Engine SHA, Identity SHA, expected schema versions, pinned image digests (where available), source branch/PR IDs, CI evidence, migrations, data-compatibility/restore support, artifact provenance, test environment, required owner acceptance. Label a release `CANDIDATE`, `LIVE_TESTED`, `ACCEPTED`, or `REJECTED` based on evidence.

## 3. Transactional update and backups — implementation requirements

**Implementation in `core/tass.sh` (CLI first; buttons later).** Maintain backward compatibility with today's `check-update`, `update <sha>`, `status` and `rollback` until a migration path is tested.

1. **Exclusive lock**: prevent concurrent updates/restores and concurrent test runners mutating the same stack (e.g. `flock` over deployment lock). Record transaction ID, owner PID, old/candidate SHAs, phase, timestamp. Concurrent attempts return a clear busy status, never interleave.
2. **Preparation (no disruption)**: resolve exact reviewed SHA; ensure source ancestry/compatibility and CI eligibility, sufficient disk space, correct model binding and ports; download/build all images/worktrees **without stopping** the current services; verify candidate image digests and pin chain. No secrets in logs.
3. **Freeze or quiesce**: deny new work and pause the relay/pulse safely while preserving the *previous* settings; account for in-flight jobs; state how pending work is handled. For an interrupted transaction, do not resume action authority automatically before owner/Identity checks.
4. **Automatic verified snapshot**: stop or quiesce writes consistently, then save a complete 0600, owner-owned, versioned recovery unit (under 0700 storage) containing application `data/`, `secrets/`, `app.env`, deployment manifests for current/previous, approved settings and migration metadata. Include checksums, known Identity entity ID, file manifest/inventory, timestamps and status. Validate archive integrity and *ability to read/restore* it, not just tar exit code. If snapshot fails, restart unchanged previous services and return ERROR; **never attempt activation**. Do not copy credentials to GitHub.
5. **Atomic activation**: write candidate and rollback records using temporary files, `fsync` as appropriate and atomic rename; journal phases `PREPARED → SNAPSHOT_OK → SWITCHING → VALIDATING → COMMITTED`. On crash/reboot, determine whether candidate or previous should be active from a durable transaction record; expose `recover` if safe self-recovery cannot complete.
6. **Validation**: require healthy exact Core/Engine/Identity pins, correct entity ID, valid action gate and authorization boundaries, existing data and receipts, no unexpected toggles, no second model, and working local Core API. A locked Identity after restart can be an expected `NEEDS_OWNER_UNLOCK` state; **not** evidence real conversation passed. Record needed post-unlock real model acceptance separately. Verify functionality changes with relevant smoke tests, not merely `docker ps`.
7. **Failed candidate**: preserve candidate logs + failed data in protected diagnostic storage; atomically restore previous manifests and exact pre-update snapshot; bring old services up; revalidate old state; record `FAILED_RECOVERED`. If recovery fails, fail closed, do not claim success, retain all snapshots and give one recovery command. External GitHub posts/actions cannot be undone by local rollback.
8. **Success**: commit one complete release record (all 3 SHAs, immutable image refs, snapshot ID, source, tests, data schema), mark previous as available, preserve at least the previous accepted deployment and its snapshot, show status and `Restore Previous`. Do not automatically erase older backups to save disk.
9. **Manual restore**: at minimum `./tass.sh restore-previous` (PROPOSED; not a current command) and an actual owner-facing **Restore Previous Update** button wired to that action. Show candidate destination, latest active changes since the snapshot, data compatibility, Identity pin and snapshot. Require owner confirmation and create a fresh backup of the current state first. Default to **exact consistent previous version + data** for incompatible schema; offer **code-only rollback** only with an explicit compatibility check and owner selection. Preserve the just-left version as an available return target, then validate. If restore itself fails, restore *the state immediately before the attempted restore* or fail closed with recoverable snapshots.
10. **Retention/visibility**: maintain permanent protected records for the current and last known-good versions, and configurable older snapshot retention based on size/count with explicit owner choices. Never silently delete the last known-good backup or all recovery history; show current storage usage and backup verification status. Consider encrypted backups at rest because the archive includes GitHub tokens and private Identity keys.

## 4. Expected owner workflow after implementation

The operator path is **Check for updates → Review → Update → Observe PASS or automatic recovery**, with **Restore Previous** always available. CLI equivalents:
- `./tass.sh status`
- `./tass.sh check-update origin/experiment/tass-unified-integration-20261105` (read only)
- `./tass.sh update <reviewed-full-Core-SHA>` (automatically backs up; no preceding manual `backup` required)
- **Planned, not implemented**: `./tass.sh history`, `./tass.sh backups`, `./tass.sh restore-previous`, `./tass.sh recover`.

`./tass.sh backup` remains available for a deliberate extra checkpoint, **not** mandatory every update. The current `./tass.sh rollback` exists and currently swaps code versions while keeping newer data unless passed `--restore-data BACKUP`. Until the new transactional restore is implemented, do not describe the existing command as guaranteed full recovery.

UI, once implemented, must show actual currently running SHAs, candidate & differences, automatic snapshot outcome, live health + action mode, automatic restore outcome, **Restore Previous Update** as a real functioning action, and retained history. No dummy buttons, false health or hidden background deployment.

## 5. Required tests before claiming owner-ready

**Isolated CI/unit/native tests**:
- Update a healthy previous deployment: correct pins, verified backup, exact data+secret inventory preserved, candidate accepted.
- Simulate failed build/download: current stack never stops.
- Simulate full disk, unwritable backup folder, or corrupt archive: candidate never starts; previous state still reachable.
- Simulate failed service start, wrong version, failed health, mismatched entity ID, unauthorized action mode: old version and data return automatically, and verification passes.
- Kill update process / simulate crash at every transaction phase; next run reconciles to a consistent current or previous version, not split components.
- Rollback after successful update, including incompatible schema and an attempted failed rollback. Preserve the just-left data in a fresh snapshot.
- Two sequential successful updates, then choose previous version; confirm correct manifest/snapshot history and that the restore choice still exists.
- Concurrent update attempts: one wins lock and others fail safely.
- Browser/CLI paths enforce local owner authorization; no remote GitHub message can initiate update/restore; no passphrase/token echo or secrets in GitHub reports.
- Model and chat still work after owner unlock; learned lessons, brain revisions, skills receipts, history, audit cycles remain consistent. Verify source/target schema contract.
- A deliberately invalid Development edge is rejected immediately in UI and by backend Save; test #78.
- Mailbox #24 exact comment IDs and cycle receipts survive restart; no unapproved action, lesson, Pulse or Autonomous changes; test #76.
- No second model is started or other Compose projects affected.

**Owner-machine acceptance** (separate from fixtures):
A. Preserve current `60661aa` baseline and note existing backup IDs.  
B. Verify exact unified candidate manifest and tests; update once with no manual backup.  
C. Observe automatic snapshot and correct 3-service pins; unlock existing Identity; one real Nemotron conversation.  
D. Execute a *controlled failed candidate* against isolated data (not the live machine) to demonstrate automatic recovery.  
E. Execute owner-approved **Restore Previous** on live stack only after protected current snapshot; verify TASS entity, data, safety settings and rollback-forward possibility.  
F. Complete single mailbox issue #24 live round trip and first supervised skill separately; don't treat these as prerequisites for the first transactional updater smoke unless explicitly scoped.

## 6. Known limitations / blocked work

- Unified branch initially contains **deployed Development Core only**; it does **not yet** contain mailbox PR #77.
- The current updater already auto-backs up and attempts recovery; the above is **hardening**, not a feature present by merely creating this document.
- The branch tracks implementation decisions, not an auto-updating runtime. Do not change `TASS_TRACK` default on owner's computer or update containers until owner approves a reviewed candidate.
- GitHub remains the authoritative public record of code and user-authored source documents; local backup cannot unsend an issue comment or roll back remote GitHub writes.
- Keep `autonomous=false`, `pulse=false`, `relay=false` during integration unless the owner authorizes a bounded test. Preserve learning choices.
- After any upgrade/restart, Identity likely requires owner passphrase unlock. No automation should store or replay it.

## 7. Assignment and DONE criteria

**Team tasks in order**:
1. Integrate PR #77 mailbox **onto** `experiment/tass-unified-integration-20261105`, with current Core/Engine/Identity pins and tests, without replacing Development.  
2. Repair #78 editor relationship validation and add native+browser regressions.  
3. Make the single updater transaction-safe, verify backup before activation, automated rollback for *all* failures, previous-version restore with snapshot/history and recovery after interrupted operations.  
4. Add real local controls once CLI proven.  
5. Publish one exact candidate SHA + CI/sandbox results; owner decides deployment; then conduct local acceptance and document any defects.

Close this handoff as DONE only with a PR, precise integrated SHA/pins, verified updater state-transition tests, restore/recovery evidence and one real owner-run acceptance. If merely code-complete or fixture-only, label PARTIAL and do not promote `alpha` or `stable` automatically.
