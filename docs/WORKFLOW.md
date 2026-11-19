# TASS source, review, release, and installation workflow

**Reviewed: 2026-11-19 (CST), Stage 1 consolidation in PR #130.** [Issue #119](https://github.com/Arameair/Thought-Processor/issues/119) records current owner decisions and stabilization evidence. The proposed changes in [PR #118](https://github.com/Arameair/Thought-Processor/pull/118) have not been adopted. This document describes the working agreement; it does not claim GitHub enforces protections that have not been configured.

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

This section is a pointer, not a second status record. Accepted source, release and installation facts live only in [STATUS.md](STATUS.md). Core's audited Stage 3 history was integrated by [PR #124](https://github.com/Arameair/Thought-Processor/pull/124), and later merges are listed there. Engine and Identity repository integration remain separate work.

## Defensive audit handoffs and interrupted model access

Apply [DEFENSIVE_REVIEW_PLAYBOOK.md](DEFENSIVE_REVIEW_PLAYBOOK.md) to authorized code-security reviews, particularly cross-service persistence, replay handling, authorization and recovery.

1. **Bound the work:** identify repository ownership/authorization, exact head and base, local or fake test environment, defensive invariant and prohibited operations. Do not include production secrets or unrelated exploit scenarios.
2. **Retain rigorous verification:** execute and record required unit, integration, race, crash/restart, negative-regression, and compatible-container tests in permitted isolated test environments. A safety warning is not a reason to weaken assertions, skip tests silently, or promote code with missing evidence.
3. **Write an evidence-first review:** include filenames/test names, assertions and observed PASS / FAIL / NOT RUN results, exact command/environment provenance, current residual risks, and a verdict. Separate independently observed evidence from author claims. Keep sensitive diagnostic artifacts in approved private access scopes.
4. **Handle blocked requests transparently:** stop the blocked AI-assisted step, record its warning and the missing evidence as BLOCKED / NOT VERIFIED, provide feedback or seek authorized access via published support channels. Continue nonblocked engineering through normal CI, local tests and independent reviewers; do not attempt to bypass platform safeguards by splitting/rewording a prohibited request.
5. **Preserve gates:** an access interruption never changes the owner approval, independent audit, source merge, release, Identity grant or live-deployment requirements. Never promote a component whose required acceptance evidence is missing.

Do not modify GitHub Actions test definitions to accommodate a blocked assistant response. Any test scope change requires a separate technical rationale, review and owner authorization.

## Documentation currency and recurring blockers

- **At each audit completion or interruption:** update the existing PR discussion with verified result, exact tested revision, unresolved acceptance rows, and access blockers. If the incident affects future reviews, add a sanitized dated entry to the playbook ledger as a documentation PR; do not claim the event was automatically detected.
- **Escalation:** at the second reported occurrence of the same access interruption, mark it **RECURRING** in the next handoff, compare the current review requirements with existing documentation, and assign a documentation owner. Preserve the original technical audit verdict independently of the access status.
- **Freshness check at handoff:** check that `AGENTS.md`, `WORKFLOW.md`, `STATUS.md`, documentation map, and active PR evidence do not contradict each other. Update only affected current-state claims, with dated sources. Avoid broad document rewrites or reopening settled governance.
- **Scope:** this is a reviewer responsibility, not a claim of automatic cross-chat or desktop visibility. Any future automated reminder/check requires separate implementation and permission; do not weaken CI or tests.

## Required task entry, coordination, and exit

**Every task prompt must direct the assignee to read the current canonical Markdown records** and the active issue/PR before acting:

| Record | Single source of truth for |
| --- | --- |
| [TEAM_ACTIVE.md](TEAM_ACTIVE.md) | Who owns each current workstream, what is being done, blockers and next handoff |
| [WORKFLOW.md](WORKFLOW.md) | Required development, review, owner-approval, and release procedures |
| [DEFENSIVE_REVIEW_PLAYBOOK.md](DEFENSIVE_REVIEW_PLAYBOOK.md) | Security assurance requirements, evidence matrix, recurring access incidents |
| [STATUS.md](STATUS.md) | Last verified accepted product/release/installation status (not live agent tasks) |
| [README.md](README.md) | Documentation navigation; links to historical and topic-specific files |

**Task start:** read the records above plus `AGENTS.md`, the active issue and PR comments, and the [playbook](DEFENSIVE_REVIEW_PLAYBOOK.md) where relevant. Then confirm the current GitHub refs, the assignment authority and the team row. State the implementer, auditor, specific scope and stop condition. Do not treat an old handoff as current. If a needed record is missing or stale, report the discrepancy rather than guessing.

**How work actually starts:** no contributor runs in the background. The owner starts each Claude, Codex or GPT-6 session, and each session performs one bounded task, records it in the owning PR and stops. TASS does not participate yet. The staged plan for a GitHub-backed loop and TASS participation is [PR #133](https://github.com/Arameair/Thought-Processor/pull/133). The workloop demos in [PR #131](https://github.com/Arameair/Thought-Processor/pull/131) and [PR #132](https://github.com/Arameair/Thought-Processor/pull/132) are **experimental fixtures, not operational tooling**, and none of them changes this procedure until a stage is separately accepted.

**Each cycle close:** report exact refs, test evidence, disposition, next owner action and **DOC SYNC** for all four canonical records (updated or reviewed/no change). The authorized writer synchronizes changed facts with the existing PR/branch and discussion **on every build/test/audit iteration**. This is continuous project documentation, not an optional end-of-project cleanup. Keep edits minimal and evidence-linked; do not edit unrelated Markdown or create parallel workflow files. Draft PR content remains proposed policy until merged.

**Independent audit boundary:** the auditor is not inherently read-only for all project documentation. When authorized, the auditor can publish findings to the existing PR and update an audit record or team status **without changing candidate implementation, tests or release artifacts under review**. If the assignment explicitly requires read-only or connector permissions prevent writing, the auditor supplies `PROPOSED DOC UPDATE: <path + concise change>`, and an authorized writer applies it. Independence is preserved by separating **who authors code from who signs off on it**, not by prohibiting all documentation writes. Assistants post through the owner's GitHub account, so a comment's author is not evidence of which assistant wrote it. The owner's assignment of reviewer and implementer is what makes a review independent. Never accept a PASS from the same session that built the candidate.

**Owner authority:** no auto-merge, installation, permissions, grants, deployment, policy adoption, branch rewrites or change to test gates arises from a Markdown update. Conflicts are resolved against exact source, audited evidence and newer explicit owner decisions.

## Continuous engineering loop (default)

`BUILD → TEST → AUDIT → REPAIR → RETEST → RE-AUDIT → ... → OWNER ACCEPTANCE`

- **Builder:** implement only the approved slice; run relevant tests; update `TEAM_ACTIVE.md` with the candidate SHA, owner, blocker and next handoff; update `STATUS.md` only when a verified project-state fact changes; provide review evidence in the existing PR. Note whether `WORKFLOW.md` or the security playbook needs a scoped revision.
- **Tester:** run the appropriate offline suites and preserve exact HEAD/test-ref/results. A failed case returns to the same builder/PR. Update the shared handoff at every iteration, not just at the end.
- **Independent auditor:** verify the exact candidate and acceptance invariants; record PASS / NEEDS REPAIR / BLOCKED with evidence. Update audit documentation and current team status if explicitly authorized; do not modify and approve the same candidate implementation. Otherwise supply the update for a writer.
- **Repeat:** after NEEDS REPAIR, builder fixes the same bounded PR, tests again, updates the Markdown handoff, and returns the new head for independent re-audit. Never claim old PASS applies to a changed executable revision.
- **Owner gate:** passing the cycle permits the owner to consider the next specifically authorized step. Merging, deployment, credential/grant changes and live integrations remain separately authorized.

Every work prompt should ask for a `DOC SYNC` report covering **TEAM / WORKFLOW / SECURITY / PROJECT**: each value is `UPDATED`, `REVIEWED—NO CHANGE`, or `PROPOSED UPDATE—WRITE NOT AUTHORIZED`, with links/SHAs. Do not invent new facts simply to satisfy an update requirement.
