# Thought Processor source and installation status

**Reviewed: 2026-11-19 (CST)** against GitHub refs (Stage 1 consolidation, PR #130). This document distinguishes GitHub source from released and installed software. It holds accepted facts only: open drafts, demos and plans are not project status (see [TEAM_ACTIVE.md](TEAM_ACTIVE.md)). See [Issue #119](https://github.com/Arameair/Thought-Processor/issues/119) for the evidence trail.

## Integrated public Core source

| Item | Verified GitHub status |
| --- | --- |
| Default branch | `main` |
| Current `main` | `53e0857d11efe2a5eb2e7903c43cd226faa9d9c6` (tree `42372a12ff2106b683bd758e89edefd2acfff775`) — [PR #128](https://github.com/Arameair/Thought-Processor/pull/128) merge, 2026-11-17 |
| Merged since the Stage 3 integration | [PR #124](https://github.com/Arameair/Thought-Processor/pull/124) `0acb9d39bacf4002edf988d254be8f48eeace84f` (Stage 3 Core ancestry + audited updater history-display repair, tree `c9b1716051ae43af584b09e26b77085459101c1e`) → [PR #125](https://github.com/Arameair/Thought-Processor/pull/125) `c047a4417293861a81ad6833432ad5e3752962ab` (docs) → [PR #126](https://github.com/Arameair/Thought-Processor/pull/126) `f7e3fbbff4ab8d1cebbaac379e171c8ff137f713` (stabilization regressions, legacy Processor retired) → [PR #128](https://github.com/Arameair/Thought-Processor/pull/128) `53e0857` (updater fails closed: explicit full Core SHA, no dependency-pin changes) |
| Post-merge CI on `53e0857` | `core-tests` ([44061708703](https://github.com/Arameair/Thought-Processor/actions/runs/44061708703)) and `one-app-launcher-regression` ([44061708710](https://github.com/Arameair/Thought-Processor/actions/runs/44061708710)) success, first attempts; `workspace-tests` was not triggered by that merge (path-filtered), so it is NOT RUN on this SHA, not PASS |
| Release | No GitHub release was present at review; a source merge does not declare a multi-repository release |

Current Core source includes [PR #124's](https://github.com/Arameair/Thought-Processor/pull/124) complete audited historical integration and the later merges above. It is **not** equivalent to a deployment or owner installation update. Consult the [final merge receipt](https://github.com/Arameair/Thought-Processor/issues/119#issuecomment-6904392486).

## Last owner-accepted installed Stage 3 baseline

| Component | Accepted pin recorded on 2026-11-14 | Meaning |
| --- | --- | --- |
| Core | `8e2b02b17eb4e39d489a564af23588be22bb5d19` | Accepted local installation baseline; older than integrated `main` |
| Thought Engine | `3aeaff7a41a989414ee1cd0ec64afb51f27e9876` | Compatible Engine source pin; separate repository integration pending |
| Identity | `ee7ba1b0f9cf894235be63071c493b74e2a0eef9` | Compatible Identity source pin; separate repository integration pending |

These pins come from the recorded owner-accepted installation. **No direct examination of the owner's machine was performed for this documentation change.** Do not infer current container state, release status, or permission grants from GitHub `main`.

## Legacy Processor retirement

The owner retired the separate `processor/` executable on 2026-11-16 (merged in [PR #126](https://github.com/Arameair/Thought-Processor/pull/126)). The accepted Go Core under `core/` is the sole supported runtime foundation. Legacy source, tests, and P0/P1 documents remain in Git for evidence, but the executable and its former launchers refuse normal startup. An explicit development-only escape hatch is not a supported installation path and carries a private-launch-URL warning.

## Historical branches and pull requests

- The Stage 3 Core work represented by PRs #110, #112, #113, #114, #117, #121, #122, and #123 was integrated through #124; verify individual ancestry before closing any still-open PR.
- PR #115's separate load-test evidence files were preserved in the merged source; its branch and PR are not automatically eligible for deletion.
- [PR #118](https://github.com/Arameair/Thought-Processor/pull/118) remains an unmerged governance proposal, not current policy.
- `alpha` and `stable` are historical branch names, not evidence of a current release.
- Open historical PRs and branches require a separately approved preservation-first disposition review.

## Separate decisions still pending

1. Repository cleanup and governance, including any branch protection or workflow changes.
2. Independent Thought Engine source integration review.
3. Independent Identity source/security integration review.
4. An owner-approved immutable release manifest and separately authorized installation, if and when proposed.

See [WORKFLOW.md](WORKFLOW.md) for the current review process. For Stage 3 design and acceptance history, see [Issue #120](https://github.com/Arameair/Thought-Processor/issues/120).
