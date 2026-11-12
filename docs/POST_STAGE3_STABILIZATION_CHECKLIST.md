# TASS post-Stage-3 stabilization and cleanup checklist

**Status: PREPARED / NOT STARTED (2026-11-12).** This is an owner-requested runbook for immediately **after** full Stage 3 acceptance; it is **not** evidence that Stage 3 is complete or approval to modify the installed TASS stack. Added to the existing [governance draft PR #118](https://github.com/Arameair/Thought-Processor/pull/118), pending review. Coordinate governance rollout with [issue #119](https://github.com/Arameair/Thought-Processor/issues/119); track Stage 3 only in its [canonical issue #120](https://github.com/Arameair/Thought-Processor/issues/120).

## Operating rule: stabilize before Stage 4

**No new product features or autonomous capability expansion until checkpoints 0–5 pass and the owner approves reopening normal development.** Essential defect/security repairs and explicitly authorized Stage 3 completion are exceptions. Continue read-only documentation/PR inventory while Stage 3 proceeds, but **do not interrupt Claude's 3A work, Codex's audits, or their pinned branches**. Do not treat an issue closure, doc merge, or CI PASS as permission to update live software.

**How to use this document:** mark an item [x] only after recording a supporting **link, exact SHA and date** in the [evidence/decision ledger](#evidence-and-decision-ledger). If an item does not apply, mark **N/A with reason** in that ledger rather than claiming PASS. Assign one accountable person for each checkpoint when the cleanup begins. A [ ] task is *not* complete just because a PR or script exists.

## Checkpoint 0 — Stage 3 owner acceptance (hard prerequisite)

- [ ] **3A accepted:** Claude's read-only recorded Thought provenance → intelligible Development handoff independently audited by Codex at the exact final SHA; owner checks it on the installed version without raw JSON or a terminal.
- [ ] **3B separately authorized and accepted:** one functional instruction change, constant other inputs, two separately approved isolated A/B runs with single-use Identity decisions; preserve original Thoughts/Notes/graph; inspect real receipts and negative tests.
- [ ] **3C accepted:** save and reopen owner verdict (**better / same / worse / inconclusive**) plus correction; real local Nemotron A/B on the approved installed stack, truthful audit and owner **PASS**.
- [ ] Record complete Stage 3 Core, Engine, Identity (and other required feature) **full commit SHAs**, final test evidence, known limitations, manual owner sign-off and the result in [issue #120](https://github.com/Arameair/Thought-Processor/issues/120).
- [ ] Declare a short **stabilization window** with named coordinator, scope and defect-only exception rule. Do not call this a public `stable` release.

**Exit gate:** Stage 3C **owner PASS with exact tested/installed pins**. If 3A passes but 3B/3C is outstanding, **STOP: Stage 3 is not accepted**.

## Checkpoint 1 — Preserve a reproducible, recoverable baseline

- [ ] Read current installed Core/Engine/Identity and any required Development/Cognition/model IDs **from runtime/deployment records**; compare to the Stage 3 accepted manifest. Record UNKNOWN rather than inferring from a Git branch.
- [ ] Record source commits, artifact/image digests where available, app/model versions, configuration assumptions, schema compatibility, environment and local-only ports in **one release-candidate manifest** (no secrets, user conversations or personal data).
- [ ] Verify **one** installed application is running; no duplicate GPU model or second Core/Identity stack; capture redacted health, current action mode and persistence/read-only evidence.
- [ ] Verify a protected, complete backup and recovery procedure **against an isolated test stack** before considering any live rollback; explicitly distinguish code-only rollback from restoring compatible data and secrets.
- [ ] Retain the exact previous known-good build/backup. Do not force-push, move defaults/branches, rotate credentials, reset Identity, run an update or alter owner data during read-only inventory.
- [ ] Obtain owner approval for any separate live update/restore test; record actual observed result, including FAIL or NOT RUN if it is not demonstrated.

**Exit gate:** exact accepted component combination, build inputs, installed state, preservation/recovery evidence and unresolved recovery gaps are traceable. If reproducibility or recoverability is **unknown**, stop promotion; create/maintain a named blocker.

## Checkpoint 2 — Inventory before closing or promoting anything

- [ ] Refresh open **issues and PRs in each active repository**, with date, exact repository, current defaults and branch heads. Include Thought-Processor, Thought-engine, identity, TASS, Development, Cognition, Avatar, updater and any confirmed runtime dependencies; distinguish active code vs historical/prototype repos.
- [ ] Reconcile against the [dated November 12 PR triage](OPEN_PR_TRIAGE_20261112.md). Its **121 initial / 105 last-checked** totals are historical reference points, **not today's live counts**. Track new, closed-unmerged, merged, draft, reviewed, blocked separately.
- [ ] Build a small **dependency table** for the active work chain: coordinating issue, implementation PR, exact Core+Engine+Identity pins, unique unmerged files, owner/auditor and required next action.
- [ ] Inventory `main`, `alpha`, `stable`, default branch and relevant experimental refs **per repo**. Do not assume the proposed new meanings are already in place.
- [ ] Categorize each open item as **ACTIVE / REVIEW / READY / BLOCKED / SUPERSEDED / HISTORICAL / UNKNOWN**. Don't close UNKNOWN items merely to reduce counts.
- [ ] Flag conflicts (e.g. stale policy, duplicate issue, branch-base mismatch, grant/security requirements, schema/contract drift) in existing coordinating issues.

**Exit gate:** complete dated inventory, all current Stage 3 and integration PRs protected, actionable list of true blockers and no silent branch changes.

## Checkpoint 3 — Small, evidence-led cleanup batches

- [ ] Begin with **documentation-only/report-only/historical duplicates**, after checking their full diffs, approvals and consuming dependencies. Preserve unique learnings and link to surviving issue or accepted document.
- [ ] Reassess known retained design records [Core #85](https://github.com/Arameair/Thought-Processor/pull/85), [#91](https://github.com/Arameair/Thought-Processor/pull/91), [#101](https://github.com/Arameair/Thought-Processor/pull/101), and [#109](https://github.com/Arameair/Thought-Processor/pull/109) against the *accepted* Stage 3 code. Do not close security/contract decisions just because an old experiment ended.
- [ ] For each closure, leave **reason, what was preserved, replacement link and outstanding defect** before closing **unmerged**. For merged PRs, record the merge SHA; merging does not establish deployment or owner acceptance.
- [ ] Close obsolete issues only when resolution evidence, successor or historical label is clear. Keep active bugs/security work open with owner and next action.
- [ ] Review in **small batches**, publishing opened / closed-unmerged / merged / reviewed / blocked / owner-decisions-required deltas after each batch. No arbitrary PR-count target.
- [ ] Avoid branch deletion, repository archival, data deletion and indiscriminate force-pushes. Any branch cleanup needs explicit separate approval after preserving unique work.

**Exit gate:** no unexplained duplicate coordination chains; consequential PRs have explicit owner, dependency and next step; closure trail is auditable.

## Checkpoint 4 — Make the codebase understandable to a new engineer

- [ ] Review and accept/revise the **documentation-only governance [PR #118](https://github.com/Arameair/Thought-Processor/pull/118)**. Until accepted, it remains proposed. Do not confuse merging docs with promoting runnable code.
- [ ] Reconcile each repo README with its **actual** ownership, current source/ref, runnable/prototype/historical status, setup, relevant tests and known limitations. Roll out accepted links via [issue #119](https://github.com/Arameair/Thought-Processor/issues/119); no premature duplicate README PRs.
- [ ] Explicitly define future branch convention (`main` reviewed source, `experiment/*`/fix PRs, `stable` externally trial-ready owner release) versus **legacy** `alpha` and docs-only `main`; plan per-repo migration without rebasing unrelated work.
- [ ] Migrate/confirm one **verified accepted baseline** per runtime repo via separately reviewed PRs; only then update default-branch references, CI and branch/tag protections with the required approval. Preserve legacy refs and their original SHAs.
- [ ] Verify a new engineer can find: service ownership and Identity permission boundaries; the exact installed version; one command/runbook for startup/status; how to submit an issue/PR; tests/audits and real-vs-fixture claims; safe update/rollback; and where to get next work.
- [ ] Ask an independent contributor to perform a **cold-start documentation walkthrough** on a non-owner copy without changing the live install; capture confusing steps as focused issues.

**Exit gate:** a new contributor can reproduce the intended dev/test setup and locate accepted implementation, known risks and active work **without reading months of historical PRs or being told to assume a branch is current**.

## Checkpoint 5 — Release and update readiness (reuse existing systems)

- [ ] Keep [Arameair/updater](https://github.com/Arameair/updater) focused on approved-release discovery/manifest validation (its [issue #1](https://github.com/Arameair/updater/issues/1)); keep [Core `tass.sh`](https://github.com/Arameair/Thought-Processor/issues/79) responsible for installation, backups, rollback and interrupted-update recovery. **No second installer.**
- [ ] Independently verify a **read-only** candidate status against trusted manifest and installed pins; distinguish **NO UPDATE / AVAILABLE FOR OWNER REVIEW / BLOCKED / UNKNOWN**. An arbitrary `origin/*` head is not approved.
- [ ] Test wrong component pins, altered digest/approval, offline/missing catalog, outdated candidate, incompatible data schema and interrupted operation in isolated environments. Record evidence at the exact candidate SHA.
- [ ] Only after a separate owner decision, test an exact approved manifest via the existing update path, verified pre-update backup, health checks, deliberately failed candidate and recoverability; demonstrate clear failure reporting and previous-version restoration.
- [ ] Document release tagging, authorized publication, upgrade/recovery runbook and external-user trust/distribution questions. **External `stable` release is a separate later approval**, not an automatic reward for finishing Stage 3.

**Exit gate:** read-only approval/eligibility is credible and integrated with verified existing recovery, with failed-update test evidence and owner acceptance where actual installation is involved. Do not mark a planned one-click button as already working.

## Checkpoint 6 — Declare the foundation ready, then consider Stage 4

- [ ] Review unresolved **P0/P1 correctness, security, data-integrity and usability blockers**. Either fix and retest them or document why the owner accepts their limits; no silent exceptions for Identity/privacy or unrecoverable updates.
- [ ] Perform a short period of **real daily TASS use**, recording concrete failures, time-to-fix and useful workflows. Do not invent a daily-use metric or pretend one successful demo is sustained reliability.
- [ ] Publish a one-page **foundation handoff**: accepted release manifest/link, branch policy adoption status, known blockers, owner-tested workflows, required CI/audit commands, rollback owner and next prioritized issue.
- [ ] Owner explicitly authorizes the end of stabilization and the scope of the **first** post-freeze feature. Otherwise remain on bug fixes/maintenance.

**Exit gate:** an owner-used, documented, reproducible, reviewable and recoverable foundation, with explicit approval to resume new feature development.

## Evidence and decision ledger

Populate as work actually happens. Do **not** prefill dates, SHAs, results or approver names. Use the existing coordinating issue for progress and link here from the release record, rather than opening one PR per checklist item.

| Checkpoint | Accountable owner | Evidence links / exact revision(s) | PASS / FAIL / BLOCKED / NOT RUN | Owner decision / date |
| --- | --- | --- | --- | --- |
| 0 — Stage 3 acceptance | TBD | [Issue #120](https://github.com/Arameair/Thought-Processor/issues/120) | NOT RUN | Pending |
| 1 — Reproducible baseline | TBD | Pending | NOT RUN | Pending |
| 2 — Repository inventory | TBD | [Historical PR snapshot](OPEN_PR_TRIAGE_20261112.md) | NOT RUN | Pending |
| 3 — Cleanup batches | TBD | Pending | NOT RUN | Pending |
| 4 — Engineer-ready documentation | TBD | [Governance draft #118](https://github.com/Arameair/Thought-Processor/pull/118) | NOT RUN | Pending |
| 5 — Updater and recovery | TBD | [updater #1](https://github.com/Arameair/updater/issues/1) / [Core #79](https://github.com/Arameair/Thought-Processor/issues/79) | NOT RUN | Pending |
| 6 — Resume development gate | Owner | Pending | NOT RUN | Pending |

### Ready-to-start handoff after 3C PASS

Once [issue #120](https://github.com/Arameair/Thought-Processor/issues/120) has **exact SHA owner PASS**, the first non-destructive maintenance assignment is:

> Read-only inventory of deployed Core/Engine/Identity and required features; branches/defaults/active PR dependency chains across participating repositories; current CI/audit evidence; existing backups/restore documentation. Publish discrepancies and blockers under [issue #119](https://github.com/Arameair/Thought-Processor/issues/119). **Do not merge, deploy, run a model, change Identity grants, delete branches or touch owner data.** Return the minimum safe next cleanup batch for owner review.

**Reference rules:** [main workflow](WORKFLOW.md) · [issue/PR maintenance](TASS_ISSUE_PR_WORKFLOW.md) · [release policy](TASS_REPOSITORY_RELEASE_POLICY.md) · [updater coordination](UPDATE_RELEASE_WORKFLOW.md).
