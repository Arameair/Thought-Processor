# Unified Go Core: GPT-6 ↔ TASS mailbox integration — acceptance handoff

Date: 2026-11-05. Scope: Thought-Processor #76, #79 Phase A. **NOT LIVE ACCEPTED.**

## Provenance and immutable pins

- Base: unified branch `experiment/tass-unified-integration-20261105` at `913a4ff687054fb8e2dddf076ccaac992398394a`.
- Owner-installed Core **remains** `60661aa807cac4d93ebc461e1447f3c1e6ad9473`.
- Engine pin **unchanged**: `21b3d502775b3ed065ab445b88f51909500770b3`.
- Identity pin **unchanged**: `87dd9243c721fd11701575033d95d3dcf286f765`.
- Ported only the opt-in mailbox from old PR #77. Existing /dev Brain Builder, supervised skills, Workspace and lesson history retained.
- New code candidate is **not** an accepted release. Do not change installed containers until the owner approves and #79 updater recovery gates pass.

## Mechanism

1. Go Core's local Talk-on-GitHub switch is OFF by default, independent of Autonomous and Pulse.
2. The pilot accepts only new `ChatGPT:` comments on **open Arameair/TASS#24** with label `tass-core` and an allowed *GitHub account* (owner login `Arameair` by default); comment prefix is NOT cryptographic proof of assistant origin.
3. First enable persists the existing comment-ID baseline, excluding old comments despite second-resolution GitHub timestamps.
4. New incoming comment ID is recorded durably before queueing. Core obtains Identity decisions for `conversation`, `model.request`, `thought.process` and `github.sync`.
5. Engine cycle source includes original issue number and numeric comment ID. A GitHub-origin message cannot call lesson or skill controllers; Engine receives zero executable actions. It cannot change NOW, memory or settings.
6. Core requires a correlated Engine FinishCycle result before enqueueing one reply with source ID, cycle ID and TASS marker. Outgoing posting records GitHub's **real returned comment ID** and persists a metadata-only receipt.
7. Ambiguous outgoing post is held (not retried blindly); duplicate comment IDs are refused; restarted Core reads the ledger. Public model output is in the public reply only; no raw GitHub content or reasoning is added to the relay audit.

## Test gates

- GitHub Actions `core-tests` for the **new integrated head**, not PR #77's older head. Must pass Go vet, Go race tests, page JS syntax, Core build, Docker build, Compose config and script syntax.
- `relay_test.go` fake Identity/GitHub/Engine tests: exact IDs; old-comment baseline; label/author/prefix and issue filtering; no executable actions; Locked Identity; switch OFF; repeated poll/restart idempotence.
- Preserve `/dev`, skills, workspace, lesson tests from running Core. Run owner's real browser acceptance separately (#78).
- **Unrun:** isolated model/Identity/Engine integration with actual binaries; real owner's Nemotron; actual GitHub POST; recovered update; owner rollback. Keep these NOT RUN until evidence exists.

## Owner's eventual bounded trial — only after permission

1. Read PR, exact Core/Engine/Identity manifest and CI, compare against live `./tass.sh status` **without updating**. Confirm `autonomous=false`, `pulse=false`, `relay=false`.
2. Do NOT use older PR #77's `./tass.sh update 4fbc268...` instructions. First complete #79's verified automatic update/restore gate and obtain an explicit owner decision on the reviewed unified candidate.
3. After approved update, unlock Identity locally and verify `READY/CHAT_ONLY`. Enable **only** Talk on GitHub and observe the Core's priming event.
4. After explicit approval of a single POST, GPT-6 writes one harmless `ChatGPT: MAILBOX-01` comment to #24, recording its returned comment ID and URL. No other messages.
5. Compare Identity decisions, Engine `cycles.jsonl` and actual model-request record, Core `relay-audit.jsonl`, outgoing comment ID and public reply cycle. Confirm action outcome is not `EXECUTED`.
6. Poll again and restart only if approved; one input must still have exactly one output. Turn relay OFF and confirm original safe settings. A Green UI badge or fixture pass alone is not acceptance.

## Unresolved integration work (#79)

- #78 immediate editor-side bad-`uses` and cyclic-`next` rejection (server currently validates at Check/Save).
- Transactional `tass.sh` update and verified snapshots, crash-safe recovery, owner-visible history, Restore Previous Update, negative fault-injection tests.
- Approval for owner-machine update, rollback and real mailbox round trip.

**Status: PARTIAL (code candidate only).**
