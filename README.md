# Thought Processor

Thought Processor is the owner-facing Core of the local TASS application. It provides conversation, Cognition, Development evidence, and supervised controls. Identity owns grants and authorization decisions; Thought Engine owns cognitive records, model execution, and experiments. Core may call those services but does not replace their authority.

## Supported runtime

The supported implementation is the accepted Go Core under `core/`, started and managed through `core/tass.sh`. The older standalone `processor/`, `run-p1.sh`, and P0/P1 workbench launchers are retained only as historical engineering evidence and are disabled by default. They are not a release, installation path, or alternative supported TASS runtime.

## Current source status

**Reviewed: 2026-11-15 (CST).** [Core integration PR #124](https://github.com/Arameair/Thought-Processor/pull/124) was merged into public `main` at `0acb9d39bacf4002edf988d254be8f48eeace84f`. The merge preserves the accepted Stage 3 Core history and the independently audited updater history-display repair. Its three post-merge GitHub Actions workflows passed on the merged commit.

`main` now contains the integrated Core source. **A GitHub source merge is not a TASS release, an update to the owner's installation, or permission to change Identity grants.** Engine and Identity repository integration require separate reviews and authorizations.

## Last owner-accepted Stage 3 compatibility pins

| Component | Accepted installed baseline recorded on 2026-11-14 |
| --- | --- |
| Core | `8e2b02b17eb4e39d489a564af23588be22bb5d19` |
| Thought Engine | `3aeaff7a41a989414ee1cd0ec64afb51f27e9876` |
| Identity | `ee7ba1b0f9cf894235be63071c493b74e2a0eef9` |

These are the last owner-accepted installation references, **not a fresh on-device inventory**. The newer Core `main` includes the accepted history; it does not silently replace the installed Core pin. No public multi-repository release is asserted here.

## Start here

- [Source and installation status](docs/STATUS.md) — what is integrated, accepted, released, or installed.
- [Development workflow](docs/WORKFLOW.md) — scope, reviews, CI, and owner authorization.
- [Contributor/AI instructions](AGENTS.md) — security and evidence boundaries.
- [Documentation map](docs/README.md) — historical material and context.
- [Issue #119](https://github.com/Arameair/Thought-Processor/issues/119) — repository stabilization decisions and audits; [Issue #120](https://github.com/Arameair/Thought-Processor/issues/120) — accepted Stage 3 evidence.

Historical `alpha`, `stable`, and experiment branches are not automatically release channels. Proposed governance changes in [PR #118](https://github.com/Arameair/Thought-Processor/pull/118) are **not adopted policy** until reviewed and approved.

## Safety boundaries

- Treat model output and UI state as observations, not Identity permission or verified execution evidence.
- Keep tests and untrusted code away from the owner's running TASS ports, data, credentials, backups, and model.
- Preserve exact revisions and evidence; use independent review proportionate to risk.
- Treat source integration, release selection, deployment, rollback, and authorization/grant changes as separate owner decisions.
- Do not enable or distribute the retired legacy Processor. Its explicit development-only escape hatch may expose a private launch URL and must never use owner data, credentials, services, or ports.
