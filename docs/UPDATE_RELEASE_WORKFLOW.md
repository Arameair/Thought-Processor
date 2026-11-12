# TASS coordinated releases and owner-approved updates — incremental pilot

**Status:** Proposed integration contract, 2026-11-12, in [governance PR #118](https://github.com/Arameair/Thought-Processor/pull/118). **Design and engineering handoff only; no updater code has been changed or deployed by this document.** Implementation/recovery remains tracked in [existing issue #79](https://github.com/Arameair/Thought-Processor/issues/79), with earlier requirements in [#32](https://github.com/Arameair/Thought-Processor/issues/32) and [#44](https://github.com/Arameair/Thought-Processor/issues/44).

## Why we need this

Multiple people can work at once on **separate branches and PRs**, but the owner installs **one compatible TASS application**. A latest Git ref, a green isolated test or a merged PR does **not** mean a compatible, authorized, recoverable update exists. One verified release candidate must pin the exact Core, Engine, Identity and required feature versions that were tested together.

**Already in the source:** at the inspected `core/tass.sh` branch `experiment/nudge-stale-preclaim-read-repair-20261111` (historical inspected snapshot, not proof of owner deployment), CLI functions include `check-update`, `update`, `rollback`, `restore-previous`, `recover`, `history`, and a verified-backup attempt. `check-update` defaults to `origin/$TASS_TRACK` and prints candidate SHA/pinned Engine/Identity commits and an update command. It **does not by itself establish that that branch head is audited, owner-approved or a shareable release**. Don't replace or bypass this existing updater: add eligibility checking in small, separately tested steps.

## Parallel contributor workflow

1. One issue defines **one problem** and one accountable owner. Each contributor has their own isolated working directory/branch and PR. A PR is a review request, **not** the working copy itself.
2. Each PR names its actual base/head SHA, related issues, dependent PRs and compatible component pins. Separate repo changes can have separate linked PRs; one coordinating issue/release record owns integration.
3. Independent reviewers audit exact SHAs. Fix failing tests on the same PR when possible. Avoid duplicate audit PRs or changes to an unapproved running install.
4. When two PRs overlap or one depends on the other, explicitly order or rebase/retest **with reviewer agreement**. A passed test on a pre-merge SHA does not automatically cover the merged composition.
5. Integration verifies the combined pinned Core + Engine + Identity set **before** it can be proposed as a release candidate. The coordinator updates one authoritative release record. No contributor self-promotes their PR to installed/stable status.
6. Use separate local containers/test roots/ports. AI collaborators must not share or overwrite the owner's live Docker project, data or Identity secrets. Avoid parallel competing real-model loads on the owner's GPU.

## Release record — fields required before real candidate eligibility

The first version can be a human-readable manifest, later validated in automation. No sample values may be presented as an actual approved release.

| Field | Requirement |
| --- | --- |
| Release identifier, channel and timestamp | Explicit `owner-trial` or `external-trial` designation; do not infer from branch name |
| Component references | Full immutable Git commit SHAs for Core, Thought Engine, Identity and required services/features |
| Build provenance | Content digests of built/distributed artifacts and how they correspond to the pinned sources |
| Compatibility | Schema/data migration expectations, matching integration test set, supported local model/runtime assumptions |
| Evidence | Final candidate SHAs, independent review and actual PASS/FAIL/NOT RUN including real-vs-fixture distinction |
| Approval | Named accountable owner decision and release intent, linked to immutable audit/acceptance record |
| Recovery | Verified backup requirement, tested restore/rollback compatibility, prior known-good candidate and stop conditions |
| Known limitations | Unresolved issues, affected surfaces and whether external sharing is blocked |

Manifest generation may be automated; **approval and trust of the manifest cannot be inferred solely because CI or an AI-generated PR says PASS**. Protect publication/tag rights. Immutable release tags/records must not be rewritten silently.

## Minimal pilot: safer read-only update availability

**First engineering slice (separate focused PR, no deployment):** inspect the installed/current deployment record and an explicitly named **approved candidate manifest**; show candidate and installed full pins, integrity/provenance, approval status and changed components.

- **NO UPDATE:** same verified release record/pins.
- **AVAILABLE FOR OWNER REVIEW:** trusted manifest and required evidence present; source/artifact/compatibility checks pass; authorization for **applying** still not granted.
- **BLOCKED:** no approved manifest, incompatible pins/schema, invalid signatures/digests, untrusted provenance, missing required tests, stale approval or active unfinished operation.
- **UNKNOWN / UNAVAILABLE:** cannot verify GitHub/release metadata, missing installed state or incomplete evidence. Do not silently fall back to an arbitrary branch head.
- Never claim "no update" merely because the check couldn't reach the release source.

Read-only pilot must **not** alter running containers, owner data, permissions, current deployment or the selected release. Remote fetching/caching, if used, must be documented separately from the no-deployment guarantee.

**Required tests:** approved manifest with compatible pins; a newer unreviewed branch head; missing/altered manifest; mismatched artifact digest; wrong Core/Engine/Identity combo; older/replayed release; offline/unavailable GitHub; no installed release; rejected/expired approval; interrupted operation. Verify output is truthful, repeatable and does not expose private tokens or Thoughts. Run in isolated test roots; test no running-service mutation. Audit the exact resulting SHA and report NOT RUN honestly.

## Only after the pilot: owner-controlled apply and recovery

1. Owner selects the displayed, validated exact candidate, not a moving `origin/*` ref; revalidate approval/pins immediately before any change.
2. Preflight and verify free space, source integrity, schema compatibility, backup completeness and current Identity/authentication conditions.
3. Create and verify an automatic protected backup of data and secrets, recording which deployed SHA and data snapshot belong together.
4. Stage/build the exact approved artifacts in isolation. Preserve existing known-good images/manifest and owners' private data.
5. Owner explicitly approves the update; apply, recheck runtime health, Identity boundaries and durable data. Never silently enable autonomy or grant a new capability.
6. Failed or interrupted update enters BLOCKED/RECOVERY with truthful evidence. Restore previous code **and compatible data** only through the already-reviewed recovery workflow; preserve the most recent verified backup and report failure if recovery is uncertain.
7. Record outcome and installed version in a **separate append-only installed/update history**, not by moving `main` or `stable` pointers.
8. For the first release to other users, require a separate external-trial acceptance gate and protected release tag. A locally approved owner trial is not automatically a public `stable` release.

## Pilot order and owners

- **Now (documentation):** maintain this contract inside [PR #118](https://github.com/Arameair/Thought-Processor/pull/118) and the implementation scope on **existing issue #79**. No new duplicate roadmap issue.
- **Next (read-only engineering PR):** one contributor adds release-eligibility inspection to the **current** source-pinned `tass.sh`/manifest path (only after checking it against Codex's active Stage 2 source). Another contributor independently tests it; neither calls live update.
- **Later (owner-gated change):** integrate manifest-qualified updates into existing `update`, verify backup/rollback/restore with an intentionally failed **isolated** candidate, then owner authorizes one live trial.
- **Only after owner acceptance:** expose the same truthful results via a desktop Update control; a button must not pretend to perform an operation the backend cannot perform.

## Stop conditions

Do not implement or deploy a parallel updater, retarget Codex's active PR, switch `main`/default branches, auto-merge, install model changes, bypass Identity grants, discard journal/secrets, or claim a release is approved from a branch name. Stop on missing audit/provenance, incompatible component pins, unresolved migration or inability to restore the previous accepted version.

**Success of this pilot:** an engineer, Claude, Codex and the owner can all answer **"Which exact approved candidate is available, why is it eligible, and what would update do?"** with the same reproducible evidence. Later one-click apply is a distinct authorization/testing milestone.
