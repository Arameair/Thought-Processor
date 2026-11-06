# TASS Go Core — current integration entry point

**Status checked November 6, 2026 — integration candidate, not yet owner-accepted.** This directory contains the Go Core of the local-first TASS application, not the old `experiment/go-core-20261104` standalone launcher. The earlier README contained **unsafe historical cleanup/rollback advice that could delete persistent TASS data**. That advice has been withdrawn from current instructions. Do not delete data directories as a rollback strategy.

## Read first

1. [Current pinned Go stack, component responsibilities, installed vs candidate limits](docs/CURRENT_GO_STACK_20261106.md).
2. [Owner session / acceptance checklist for Core PR #88](docs/OWNER_SESSION_FOUNDATION_20261105.md) — **review only until explicitly approved**, including the actual recovery procedure.
3. [Update and rollback safety issue #79](https://github.com/Arameair/Thought-Processor/issues/79) — fake-Docker updater tests reported, **real owner-host backup/restore acceptance pending**.
4. [First authorized action and evaluation gate #89](https://github.com/Arameair/Thought-Processor/issues/89) and [Cognition status](docs/COGNITION_STATUS_20261106.md).
5. [Complete cross-repository issue triage](docs/ISSUE_TRIAGE_20261106.md).

## What the current Go stack does

- **Identity** controls canonical entity IDs, lock state, grants and revocation. Each receiving service must enforce permissions; the full current candidate has not passed broad adversarial isolation acceptance. [Engine entity-ID defect #17](https://github.com/Arameair/Thought-engine/issues/17).
- **Thought Engine** handles model-backed cycles and durable operational evidence; its 20-second **Pulse** is not the same as Core's 20-second **SyncOnce**. A tick with NO_ACTION is not an executed skill or prediction evaluation.
- **Core** hosts the local owner UI, scoped learning and skills, Development, Cognition read projections, GitHub synchronization and supervised controls. Its restricted mailbox prototype is **conversation only** until independently authorized.
- **TASS `live`** holds versioned, optionally synchronized descriptive mind and approved lesson sources, not Identity authority or automatically public private-memory logs.
- **Cognition** is the Core-hosted read-only view of actual Engine journal records; its fixtures are not evidence of a live accepted result.

## Safe operational boundary

**Read-only status first:** on a specifically known and checked-out current candidate, `./tass.sh status` is the intended inventory step. Confirm the actual installed Core/Engine/Identity full SHAs, model endpoint, data roots, permissions and Compose project. **Do not assume the currently checked-out branch is running.**

**Before any update, reinstall or rollback:** follow the exact candidate owner runbook, verify a recoverable backup and explicit owner approval. Do **not** run old `start.sh` instructions, delete `~/.tass`, `~/.thought-engine/tass`, or `~/.tass-app`, or start a second stack against the existing data. The safe updater/recovery workflow has simulated tests, but owner real-Docker acceptance is pending.

**Network:** localhost/page tokens must not be treated as mobile or remote-owner authentication. Do not expose internal API ports to the LAN or Internet; [security issue #47](https://github.com/Arameair/Thought-Processor/issues/47) remains open.

## Historical implementation record

The former README described the **November 4 initial Go core**, fixture model tests, a static `policies/tass.json` policy, direct `start.sh` launcher, old ports, and a destructive rollback. Those were dated prototype details, **not valid instructions for the present Docker/Identity service candidate**. Its original content and test evidence remain recoverable from Git history on earlier commits and branches; historical fixture PASS remains labeled fixture-only.

**Evidence and change policy:** exact source and running SHAs, real vs fixture, PASS/FAIL/NOT RUN and authorization/receipt IDs are required before anyone claims acceptance. These documentation changes do not modify the running owner installation.
