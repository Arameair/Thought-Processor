# TASS source, review, release, and installation workflow

**Reviewed: 2026-11-15 (CST).** [Issue #119](https://github.com/Arameair/Thought-Processor/issues/119) records current owner decisions and stabilization evidence. The proposed changes in [PR #118](https://github.com/Arameair/Thought-Processor/pull/118) have not been adopted. This document describes the working agreement; it does not claim GitHub enforces protections that have not been configured.

## Lightweight development path

1. **Define the slice.** Use one primary issue to state the problem, acceptance criteria, repository owner, and permitted changes. Reuse the same PR for a repair rather than multiplying PRs.
2. **Branch from an approved revision.** Prefer short-lived `feature/*`, `fix/*`, `docs/*`, or clearly bounded `experiment/*` branches. Verify actual base/head SHAs and dependencies. Preserve distinct experimental work until disposition is approved.
3. **Keep a single review record.** Open a draft PR for the slice. Record changed paths, exact revisions, test commands, and PASS / FAIL / NOT RUN results. Do not label a path-filtered or unexecuted CI job as PASS.
4. **Review proportional to risk.** Documentation-only changes need accurate source checks and an independent focused review. Executable behavior needs applicable tests and independent review. Cross-service contracts, persistence, credentials, Identity authorization, updater execution, or deployment require stronger exact-SHA evidence and explicit owner consent.
5. **Verify the final candidate.** Audit the current head and proposed merge result, including any CI run on a synthetic PR merge ref. Do not self-certify independence. Stop on an unexplained failure, privacy concern, or unexpected automation.
6. **Merge only when authorized.** Confirm the permitted merge method, destination, state, exact source and CI evidence. Preserve audited ancestry; never force-push or rewrite accepted histories simply for tidiness. Observe post-merge CI and report its actual results.
7. **Keep release and installation separate.** A merged source tree does not select a multi-repository release or deploy TASS. Release pins, manifests, installation, rollback, Identity grants, and autonomy changes require their own review and owner authorization.
8. **Clean up with evidence.** Before closing PRs or deleting branches, verify unique commits, retained documentation/tests, cross-repository pins, and historical links. Only execute approved dispositions; preserve the audit trail.

## Practical security and collaboration rules

- Identity owns canonical authorization; Core and Engine must not bypass its grants or revocation decisions.
- Work in disposable fixtures for untrusted tests; do not modify the owner's installed services, model, data, secrets, or backups without a distinct scope authorization.
- For human and AI contributors, separate the author from an independent auditor when the change's risk requires it. The owner is the final authority for protected actions.
- Keep public commit author and committer metadata privacy-safe; do not publish credentials, private locations, or owner records.
- Current GitHub branch restrictions and required checks must be verified from settings, not inferred from this document. They were not enforced at the Core integration review.
- Archive historical documents as evidence, but do not treat their old branch, release, or installation claims as current instructions.

## Current source milestone

Core's audited Stage 3 history was integrated by [PR #124](https://github.com/Arameair/Thought-Processor/pull/124) into public `main` at `0acb9d39bacf4002edf988d254be8f48eeace84f`, with three successful post-merge CI runs. Engine and Identity repository integration remain separate work. See [STATUS.md](STATUS.md) and [Issue #119](https://github.com/Arameair/Thought-Processor/issues/119).

## Defensive audit handoffs and interrupted model access

Apply [DEFENSIVE_REVIEW_PLAYBOOK.md](DEFENSIVE_REVIEW_PLAYBOOK.md) to authorized code-security reviews, particularly cross-service persistence, replay handling, authorization and recovery.

1. **Bound the work:** identify repository ownership/authorization, exact head and base, local or fake test environment, defensive invariant and prohibited operations. Do not include production secrets or unrelated exploit scenarios.
2. **Retain rigorous verification:** execute and record required unit, integration, race, crash/restart, negative-regression, and compatible-container tests in permitted isolated test environments. A safety warning is not a reason to weaken assertions, skip tests silently, or promote code with missing evidence.
3. **Write an evidence-first review:** include filenames/test names, assertions and observed PASS / FAIL / NOT RUN results, exact command/environment provenance, current residual risks, and a verdict. Separate independently observed evidence from author claims. Keep sensitive diagnostic artifacts in approved private access scopes.
4. **Handle blocked requests transparently:** stop the blocked AI-assisted step, record its warning and the missing evidence as BLOCKED / NOT VERIFIED, provide feedback or seek authorized access via published support channels. Continue nonblocked engineering through normal CI, local tests and independent reviewers; do not attempt to bypass platform safeguards by splitting/rewording a prohibited request.
5. **Preserve gates:** an access interruption never changes the owner approval, independent audit, source merge, release, Identity grant or live-deployment requirements. Never promote a component whose required acceptance evidence is missing.

Do not modify GitHub Actions test definitions to accommodate a blocked assistant response. Any test scope change requires a separate technical rationale, review and owner authorization.

## Documentation currency and recurring blockers

- **At the beginning of each work slice:** read the active issue and PR comments, `AGENTS.md`, current `STATUS.md`, this workflow, and the [defensive review playbook](DEFENSIVE_REVIEW_PLAYBOOK.md) where relevant. Confirm the current head SHA. Do not treat an old handoff as current.
- **At each audit completion or interruption:** update the existing PR discussion with verified result, exact tested revision, unresolved acceptance rows, and access blockers. If the incident affects future reviews, add a sanitized dated entry to the playbook ledger as a documentation PR; do not claim the event was automatically detected.
- **Escalation:** at the second reported occurrence of the same access interruption, mark it **RECURRING** in the next handoff, compare the current review requirements with existing documentation, and assign a documentation owner. Preserve the original technical audit verdict independently of the access status.
- **Freshness check at handoff:** check that `AGENTS.md`, `WORKFLOW.md`, `STATUS.md`, documentation map, and active PR evidence do not contradict each other. Update only affected current-state claims, with dated sources. Avoid broad document rewrites or reopening settled governance.
- **Scope:** this is a reviewer responsibility, not a claim of automatic cross-chat or desktop visibility. Any future automated reminder/check requires separate implementation and permission; do not weaken CI or tests.
