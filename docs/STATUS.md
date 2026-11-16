# Thought Processor source and installation status

**Reviewed: 2026-11-15 (CST).** This document distinguishes GitHub source from released and installed software. See [Issue #119](https://github.com/Arameair/Thought-Processor/issues/119) for the evidence trail.

## Integrated public Core source

| Item | Verified GitHub status |
| --- | --- |
| Default branch | `main` |
| Integrated merge | `0acb9d39bacf4002edf988d254be8f48eeace84f` — [PR #124](https://github.com/Arameair/Thought-Processor/pull/124), merged |
| Merge tree | `c9b1716051ae43af584b09e26b77085459101c1e` |
| Included source | Owner-accepted Stage 3 Core ancestry plus the independently audited updater history-display repair |
| Post-merge CI | `core-tests`, `one-app-launcher-regression`, and `workspace-tests` all PASS on the merged SHA (first attempts) |
| Release | No GitHub release was present at review; a source merge does not declare a multi-repository release |

The updated Core source includes [PR #124's](https://github.com/Arameair/Thought-Processor/pull/124) complete audited historical integration. It is **not** equivalent to a deployment or owner installation update. Consult the [final merge receipt](https://github.com/Arameair/Thought-Processor/issues/119#issuecomment-6904392486).

## Last owner-accepted installed Stage 3 baseline

| Component | Accepted pin recorded on 2026-11-14 | Meaning |
| --- | --- | --- |
| Core | `8e2b02b17eb4e39d489a564af23588be22bb5d19` | Accepted local installation baseline; older than integrated `main` |
| Thought Engine | `3aeaff7a41a989414ee1cd0ec64afb51f27e9876` | Compatible Engine source pin; separate repository integration pending |
| Identity | `ee7ba1b0f9cf894235be63071c493b74e2a0eef9` | Compatible Identity source pin; separate repository integration pending |

These pins come from the recorded owner-accepted installation. **No direct examination of the owner's machine was performed for this documentation change.** Do not infer current container state, release status, or permission grants from GitHub `main`.

## Legacy Processor retirement

The owner retired the separate `processor/` executable on 2026-11-16. The accepted Go Core under `core/` is the sole supported runtime foundation. Legacy source, tests, and P0/P1 documents remain in Git for evidence, but the executable and its former launchers refuse normal startup. An explicit development-only escape hatch is not a supported installation path and carries a private-launch-URL warning.

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
