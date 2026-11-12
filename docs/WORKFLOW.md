# TASS development workflow — issue, PR, verification, acceptance

**Status:** Owner-selected **target workflow**, documented 2026-11-12. Proposed in [governance draft PR #118](https://github.com/Arameair/Thought-Processor/pull/118); **not yet a branch migration or an approved deployment.** Existing protected operations and the exact baseline assigned to any active task remain in force. Read [repository/release policy](TASS_REPOSITORY_RELEASE_POLICY.md), [contributor guide](../CONTRIBUTING.md), and [work-item hygiene](TASS_ISSUE_PR_WORKFLOW.md).

## At a glance

```text
Issue (problem, evidence, acceptance criteria)
  -> experiment/* or fix/* branch + one focused draft PR
  -> tests, independent review, repairs on that PR
  -> explicit acceptance/merge decision -> main (accepted source)
  -> owner-authorized installed trial of a pinned build, when needed
  -> separate public-trial release gate -> release tag + manifest + stable
```

An owner trial may happen **before** the merge if the owner explicitly authorizes installation of the audited candidate. The installed revision must be recorded regardless of branch. The flow is not permission to deploy or run the real model. Documentation-only PRs need an appropriate content review, not a full runtime release suite.

## Branch definitions: destination policy, not current repository state

| Name | Meaning once migrated |
| --- | --- |
| `main` | Latest reviewed and accepted source code, documentation and tests; normal base for **new** work once the repository's migration is complete. Not necessarily deployed or externally released. |
| `experiment/*` or `fix/*` | Temporary, isolated branch for one coherent change, with a linked issue and PR. A passing fixture does not mean owner acceptance. |
| `stable` | Moving convenience pointer to the last **owner-used and explicitly approved release suitable for other people to try**. Not automatically current installation, not a substitute for version tags. No `stable` release is implied by this policy. |
| Versioned **release tag** | Treat an annotated tag as immutable by policy and protect released tags. It pins a specific component commit. |
| Cross-repo **release manifest** | Identifies the exact compatible Core, Engine, Identity and required feature commits, build artifact hashes where available, test/acceptance evidence and recovery instructions. |
| `alpha` | **Legacy branch category.** Existing `alpha` branches are preserved for pending work and reference; no blanket deletion, force update or automatic promotion. Retire the category only after dependent work is inventoried and safely migrated. |

### Transition warning: what actually exists today

This repository's current `main` has historically been a **documentation/portfolio lane**, the existing `stable` is an **older documentation-only snapshot**, and `alpha` was an earlier owner-promoted baseline. The previous workflow said `experiment/* → alpha → stable → main`. That is **historical**, not the destination convention. Thought Engine currently uses an experimental default branch; other services have their own release histories.

**Until a separately reviewed migration is complete, NEVER assume `main`, `alpha`, `stable` or a repository default contains the running code.** Before any new implementation, inspect the work issue, active implementation PR and assigned exact Core/Engine/Identity SHAs. Existing Stage 2/3 work stays on its reviewed, pinned branches until explicitly transitioned. Do not silently rebase it onto `main`.

For the historical October 27 branch snapshot, see [STATUS.md](STATUS.md); it is **not** an up-to-date installed-version report. For current stage scope see [workflow trial issue #111](https://github.com/Arameair/Thought-Processor/issues/111), [Cognition stage planning PR #109](https://github.com/Arameair/Thought-Processor/pull/109) and the relevant active PR. Validate exact head SHA and current decision comments before using them.

## One work item, start to finish

1. **Find or open an issue.** Search for duplicates. State observed problem or proposal, reproduction or source reference, expected outcome, risk, and objective acceptance criteria. Contributor AIs may report confirmed defects; new feature scope and changed security requirements need an owner decision. An unverified hypothesis is labeled as such.
2. **Choose the correct base.** Use the currently accepted `main` **after migration**; until then use the explicit source pin in the owner-approved work assignment. Record upstream and cross-repo dependencies. Do not fork an old experiment merely because it appears to contain more code.
3. **Open a focused draft PR.** Link the issue, list changed files and non-goals, assign an implementer, record head SHA and component pins. Keep the PR as the primary evolving record: `DONE / OPEN / NEXT / BLOCKED`.
4. **Test and challenge.** Implementer runs focused tests and the appropriate final candidate gate. Independent auditor verifies the exact commit, especially Identity/authorization/privacy, durable writes, concurrency and recovery where applicable. Record commands, environment and PASS/FAIL/NOT RUN; distinguish fixtures, real binaries, local model and owner machine. A comment or a mock is never proof of installed behavior.
5. **Repair on the same PR when practical.** Attach failure evidence to that PR, commit the smallest fix, and audit the **new** SHA. Open a new issue only for a genuinely independent defect. If a change affects the release gate, re-run applicable checks on the revised candidate.
6. **Decide.** The authorized owner/reviewer accepts/merges, requests revisions, or closes as superseded/rejected/historical with a linked explanation. Approval of a PR is **not** automatic approval of a deployment, Identity grant, model action or `stable` promotion.
7. **Record installed behavior separately.** Owner-authorized installation trials include the exact component SHAs, local model identity, outcome, backup/recovery evidence, limitations and rollback notes. A staged/accepted `main` commit may differ from the installed build.
8. **Release intentionally.** After repeated owner use, integration/security/recovery checks, clear limitations and explicit approval for external users, publish the pinned release manifest and protected tag. Point `stable` at that exact released commit only as part of an approved, atomic release procedure; if multiple repositories have `stable` refs, keep their compatibility recorded in one release manifest.

## Required evidence and hard boundaries

| Gate | What to record |
| --- | --- |
| Build and regression | Exact full SHA, environment, commands, test totals, failures and NOT RUN |
| Service compatibility | Matched Core + Engine + Identity revisions, interface/route checks |
| Identity/security | Denials, revocation, owner/entity isolation, no privilege granted by a UI or model |
| Persistence/recovery | Append/save and restart behavior, backup/recover/rollback **when authorized** |
| Model evidence | Real Nemotron versus fixture, call receipts and honest uncertain outcomes |
| Owner usability | What was tested on the actual install; owner sign-off explicitly recorded |
| Release | Immutable tag, hashes if available, manifest, limitations, installation and recovery |

**Safety:** No merges to protected release lanes, deployments, owner-machine restarts, model requests, autonomous changes, permission changes, branch-default changes, repo deletion or private data access without the existing explicit authorization. Never replace a real security check with a fixture PASS; never delete user data as a casual rollback step. Identity owns authorization; Engine owns its cognition execution/data; Core coordinates; Development experiments; Cognition presents recorded evidence.

## PR closing and keeping the project clean

A PR that contains an actual candidate change **to code, tests or durable documentation** is legitimate. An issue describes a defect, idea or assignment. A review verdict usually belongs as a comment on the PR it reviewed; only open a separate report PR when that report itself is intended to be merged.

At each accepted stage/checkpoint:
- Reconcile open/closed/merged PR counts and unique active work with the [dated inventory](OPEN_PR_TRIAGE_20261112.md).
- Verify whether old PRs are still depended on before closing. Preserve unique design, evidence, unresolved defects and successor references. No automatic closures by age or count.
- Every open PR needs an owner/next action or explicit BLOCKED reason; every closure needs a reason and correct linked issue/replacement.
- Update the one canonical issue and PR rather than spawning handoff/audit PRs for the same change.

See the detailed [future-team maintenance checklist](TASS_ISSUE_PR_WORKFLOW.md).

## Migration sequence — separate decisions, one repository at a time

- [ ] Confirm this documentation and release meanings with the owner and reviewers; **do not merge as a declaration that the migration is finished**.
- [ ] Inventory current branch heads, dependent PR bases, active release candidates, installed SHAs and whether `main` contains the source tree.
- [ ] Select a verified accepted code baseline for each repo; propose how to bring it into `main` without losing unique changes or overwriting history. Preserve existing `alpha` refs and log prior SHAs.
- [ ] Verify CI, security tests, repository links, and applicable branch/tag protections **before** changing defaults. Protect active Stage 2/3 work from surprise rebases.
- [ ] Update README/STATUS to truthfully describe accepted source and installed/release manifests. Only then use `main` as the normal development base.
- [ ] Prepare the first `stable` reference **only** when the external-trial acceptance gate is actually met.
- [ ] Remove obsolete instructions/branches only through reviewed and documented housekeeping; keep useful history.

## Historical record

The prior Thought-Processor workflow used `alpha` as the experiment base and `main` as an owner-approved release/documentation lane. It remains traceable in Git history before this documentation revision. Past experiments and their original acceptance criteria are not retroactively reclassified by this new guide.

## Parallel work and update automation

Multiple human or AI contributors can work simultaneously using **separate isolated branches and PRs**. A PR is a proposed change, not a shared workspace. Every branch must name its actual baseline, linked issue and dependent component/PR pins; overlapping changes require an explicit integration order and retest of the **combined** candidate. The owner has one running TASS deployment, which independent contributors must not modify as part of normal PR testing.

A reviewed change on `main` is **not** an installable update. The existing `core/tass.sh` update/rollback machinery should only be extended through a separately verified release manifest and owner-controlled approval path. Start with a **read-only** release eligibility pilot, never treat a changing Git branch as a release. See [coordinated update/release workflow](UPDATE_RELEASE_WORKFLOW.md) and existing [update/recovery issue #79](https://github.com/Arameair/Thought-Processor/issues/79).
