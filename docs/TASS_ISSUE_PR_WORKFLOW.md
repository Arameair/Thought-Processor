# TASS issue → PR → acceptance workflow (proposed)

**Status: proposal, 2026-11-12.** This document coordinates work across repositories. It does not alter authority to merge, deploy, change Identity grants or designate a stable release. This process must be reviewed before adoption. See [branch/release policy](TASS_REPOSITORY_RELEASE_POLICY.md) and [open PR inventory](OPEN_PR_TRIAGE_20261112.md).

## The simple loop

1. **Issue = the problem.** Describe observed behavior, why it matters, evidence/reproduction, and pass/fail acceptance criteria. Claude, Codex and other contributors may document confirmed defects without asking permission. Proposed new scope or altered acceptance expectations require owner review.
2. **One primary working branch and PR per focused solution.** Name an accountable implementer and link the source issue. Draft PRs are allowed while developing. Keep test evidence and fix iterations on this PR; don't open a new PR solely because a test failed.
3. **Review = proof, not a claim.** Record exact tested commit SHA, environment, fixture versus real-model boundaries, security checks, independent audit, PASS/FAIL/NOT RUN and remaining defects. If code changes after an audit, the new SHA is not automatically accepted.
4. **Disposition = finish the work.** Merge only after the relevant approval; or close as superseded/rejected/historical with a reason, successor link and preserved evidence. If blocked, name the blocker, responsible party and recheck condition. Don't equate closed issue, merged PR, installed build and owner-accepted behavior.
5. **Release = separately controlled.** Record compatible Core/Engine/Identity exact SHAs, approved artifact and installed verification. An owner-used Stage 3 candidate does not automatically become `stable`; external-trial readiness has an additional gate.

## Status vocabulary

| Status | Required next action |
| --- | --- |
| ACTIVE | Named implementer continues the linked task |
| REVIEW | Named auditor checks an exact SHA |
| READY | Verified; explicit decision/merge gate pending |
| BLOCKED | Dependency, owner decision and next check listed |
| SUPERSEDED | Replacement proven; close with link and reason |
| HISTORICAL | Valuable record but no ongoing changes; close with preserved outcomes |
| UNKNOWN | Needs inventory; do not assume a disposition |

These are proposed triage categories, not GitHub merge states or implemented automation.

## Single-PR handoff template

- **Problem/issue:** URL and one-sentence expected result.
- **Owner/implementer/auditor:** names/roles; do not let the same change self-certify.
- **Source:** repository, head branch, exact candidate SHA, merge base, dependent PRs, compatible component pins.
- **Change:** files, behavior, explicitly out of scope.
- **Evidence:** commands, environment and real-vs-fixture designation; links to reproducible logs, PASS/FAIL/NOT RUN.
- **Status and next action:** ACTIVE / REVIEW / READY / BLOCKED / SUPERSEDED / HISTORICAL; one accountable next action.
- **Acceptance:** outstanding owner hands-on test and authorization, merge/deploy/release decisions stated separately.

## Failure handling

- If an audit fails, comment on the **same** PR and link evidence. The implementer commits a repair on the **same** branch when possible. Recheck the new SHA. Create a new issue only for separate work or a confirmed independent defect; cross-link it.
- If the PR is an experiment that was rejected or superseded, write down what was learned, link the replacement, then close; don't erase history.
- If multiple repos are required, use one coordinating issue/plan and linked component PRs. Explicitly pin versions. Do not create multiple unrelated coordinating PRs.
- Naming convention: specify *Cognition Stage 2 (Expanded Thought)* versus *supervised nudge processing Stage 2*; a numeric stage alone is ambiguous.

## Periodic maintenance (proposed)

At agreed checkpoints, a contributor reviews the OPEN PR inventory: reconcile active chains, verify which PRs can close, note missing evidence, and publish a short **opened / closed / reviewed / unresolved** summary. Avoid repeating full CI for documentation triage. A closure requires a reason, not an arbitrary PR cap. Avoid interrupting Codex/Claude on an active SHA for housekeeping.

## Immediate adoption gates

- [ ] Owner reviews the workflow and the initial 121-PR inventory.
- [ ] Confirm current Codex Stage 2 active head and dependent branches before touching related work.
- [ ] Triage historical PR candidates in small batches with evidence and owner-visible results.
- [ ] Introduce PR/issue templates or automated labels only after this process proves useful by reducing ambiguous open work.

## Decision gate: ticket or PR?

Ask **“Is there a concrete repository change for reviewers to accept?”**

- **Issue:** behavior request, defect report, coordination assignment, design question, review verdict/finding or rollout decision. Keep subsequent repairs in the same linked implementation PR if possible.
- **PR:** an actual candidate change to source, tests **or durable documentation** which is intended for review and eventual merge. Documentation is not categorically “issue-only.”
- **Audit-only PR exception:** a distinct report document intended to be added permanently to the repo can be a legitimate PR. If instead it is a completed audit of an existing implementation/design PR, attach verdict and evidence to that PR/issue and close the extra report-only PR with preservation links.
- **Cross-repo decisions:** one coordination issue first; open per-repo PRs only after the decision is approved and actual file changes are ready. Do not open placeholders in each repo simply to advertise a future policy.
- **Never auto-close based on age, PR number, title or a passing fixture.** Closure requires source/dependency check and a reason. Don't erase branches or confuse a closed review PR with a fixed product defect.

First cleanup example: [2026-11-12 inventory/dispositions](OPEN_PR_TRIAGE_20261112.md) (121 open initially → 110 after eleven evidence-linked, nonmerged closures).
