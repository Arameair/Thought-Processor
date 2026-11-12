# TASS repository and release policy (proposed v1)

**Status: proposed for gradual adoption, 2026-11-12.** This is a documentation proposal, not a declaration that any existing branch meets the rules below. It does not rename, reset, merge, protect or create a `stable` branch; it does not authorize deployment, grants, model calls, or changes to the ongoing Cognition/Development stages.

## Four definitions

| Location | Intended meaning |
| --- | --- |
| `main` | The understandable entry point: purpose, boundaries, current documentation and **eventually** the latest reviewed/accepted source baseline for this repository. |
| `experiment/*` (or a short-lived feature/fix branch) | Work in progress. A passing fixture or CI run does not imply installed or owner-accepted behavior. |
| `stable` | Last release the owner has actually used and accepted as suitable for **other people to try**. Not simply the newest tested code. May not exist yet. |
| **Versioned release record** | Exact component SHAs and artifacts that were tested together, plus owner acceptance, known limitations and deployment/recovery evidence. This is required for a cross-repo TASS release. |

**Important distinctions:** `main` is not necessarily installed. `stable` is not necessarily installed. The installed runtime must report its own exact source/build identifiers; a Git branch name alone cannot prove what is running. Stage 3 owner-use acceptance is a prerequisite to pursuing a shared `stable` release, not an automatic promotion.

## Promotion and evidence

1. Develop on an isolated branch; preserve narrowly scoped reviewable changes and source links.
2. Run targeted automated tests and one appropriate final regression/security gate on the exact candidate SHA. Record PASS, FAIL and NOT RUN honestly, with fixture versus real-model distinctions.
3. Independently audit consequential changes. Identity grants, writes, cross-entity access, model execution and deployment require their existing separate authorization.
4. For **Stage 3 owner-usable** status, show the installed real Core + Engine + Identity + local model performing the defined workflow, with durable records and owner acceptance. This is not yet public `stable`.
5. For **stable/public-trial** promotion, additionally require repeatable real use, installation/update/recovery verification, documented compatibility, security checks, known limitations and explicit owner sign-off. Tag/pin the verified release; never silently promote experiments.
6. For each release, publish or link a single manifest naming exact full SHAs (Core, Engine, Identity and any required feature), build artifact digests where available, acceptance evidence, date, and tested upgrade/rollback procedure. Keep deployment records separate from release manifests.

## Repository hygiene — incremental, not a big-bang rewrite

- Each README should answer: what is this repo, what does it own, is it current/experimental/historical, where is the code and current evidence, and how to test or contribute.
- Keep one canonical, short cross-project policy (this file); other repos link rather than clone it. Local security/release constraints remain in force.
- Label historical designs and old experiments as such. Don't turn an old README's instructions into commands for the current install; particularly never delete owner data as an assumed rollback method.
- Close obsolete issues only after a reason and replacement reference are recorded. Don't delete valuable historical branches until their status and preservation are verified.
- Configure PR checks and branch protection only through a separately reviewed change. Documentation-only updates should not need the same expensive full release gate as execution/security changes.
- Review repo boundaries before any monorepo or permanent branch proliferation. No `stable` branches should be created just to appear mature.

## Current migration caveat

At this writing, some `main` branches are documentation/portfolio entry points; the Thought Engine default branch is an older experiment; existing `alpha` or `stable` branch names may carry **older meanings**. The table above is the **target convention**, not a claim that those refs contain working current code. First inventory actual refs, installed SHAs and owner-accepted baselines. Then propose branch/default changes and a release manifest with explicit owner review. No branch cleanup, force-push, default-branch change or stable promotion is implied by linking this file.

## Rollout checklist

- [ ] Review this policy, including how current `alpha` and historical `stable` branches will be handled.
- [ ] Add a short pointer in active repo READMEs; preserve existing honest status notes.
- [ ] Audit branch heads, docs accuracy and deployment records one repo at a time.
- [ ] Establish a cross-repo release manifest for a real accepted candidate (not fabricated placeholders).
- [ ] Agree on any branch protection or default-branch changes separately.
- [ ] After real Stage 3 owner-use acceptance, evaluate first owner-approved `stable` release for external trial.

**Principle:** Make the current truth obvious without presenting plans or mocks as working software.

## Required policy reconciliation before adoption (found 2026-11-12)

The existing [Thought-Processor workflow](WORKFLOW.md) **currently conflicts** with this proposed branch model. It documents `alpha` as experiment base, `stable` as approved snapshot and `main` as release lane; the proposed model would use `main` as accepted project baseline and `stable` as owner-used release ready for others to try. **Do not merge this proposal and tell teams it is operative until the owner explicitly chooses a migration.** Inventory active branches, dependent PRs and installed full SHAs, then reconcile existing `WORKFLOW.md`, `STATUS.md`, READMEs and protection rules in reviewable changes. Do not change Codex's active Stage 2 base automatically. Until reconciliation, this file is advisory and existing restrictions on merges/deployment stay in force.

Future maintainer instructions: [issue → PR → acceptance workflow and hygiene checklist](TASS_ISSUE_PR_WORKFLOW.md); [dated open PR inventory](OPEN_PR_TRIAGE_20261112.md). The index must be refreshed before treating its counts or statuses as current.
