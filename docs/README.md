# Thought Processor documentation map

## Current coordination sources

- Issue #119: preservation-first repository workflow and stabilization decisions.
- Issue #120: accepted Stage 3 design, audits, installation, and owner acceptance.
- Issue #111: delta-focused independent testing policy.
- PR #123 and its ancestors: accepted Core source stack through Stage 3C.

## Historical documentation imported from the old default branch

This tree preserves unique design, architecture, research, decision, and project-history documents from the earlier documentation-only `main` history. Most were written in October 2026 before the accepted Go Stage 3 runtime.

Unless a document explicitly cites the accepted Stage 3 revisions and the current Issue #119 workflow, treat its branch names, release status, current-version claims, and next steps as historical context—not current operational instructions or merge policy. Git history remains the authoritative source for its original context.

Current entry points are [STATUS.md](STATUS.md) and [WORKFLOW.md](WORKFLOW.md). Security-sensitive runtime behavior must be established from accepted source, exact-SHA tests, and independent audit evidence rather than generalized historical prose.

## Current review playbook (proposed)

- [Defensive review playbook](DEFENSIVE_REVIEW_PLAYBOOK.md) — security acceptance evidence, repeated AI-audit access interruption ledger, and conservative handling of blocked reviews. This is a **draft proposal in PR #130** until accepted; it does not alter runtime or CI.
- For each current audit, verify the active PR comments and exact SHA rather than assuming this documentation map is a live status feed.

## Canonical live-work and project records (draft in PR #130)

- [Active team and work register](TEAM_ACTIVE.md) — assigned implementer/auditor, current tasks, evidence links, and next handoffs; human-maintained, not a live task feed.
- [Development workflow](WORKFLOW.md) — required task start/close document checks, approval boundaries and release process.
- [Defensive review playbook](DEFENSIVE_REVIEW_PLAYBOOK.md) — security evidence and access-interruption history.
- [Source and installation status](STATUS.md) — verified release/project state, separate from current task ownership.

Contributors read the canonical records at the start of **every work cycle** and synchronize affected Markdown records after **each BUILD → TEST → AUDIT** pass and subsequent repair/retest cycles. Auditors may update permitted review/coordination documentation without modifying the implementation under audit; explicitly read-only reviewers supply proposed updates. PR #130 is a proposal until reviewed and merged.

## Repository-wide documentation readiness (proposed)

- [Cross-repository documentation inventory](REPOSITORY_DOCS_AUDIT.md) — verified 2026-11-19 defaults, active refresh PRs, and precise missing entrypoints.
- [TASS PR #27](https://github.com/Arameair/TASS/pull/27) — draft public-safe overview, TASS-owned team-presence file and handoff log. TASS is currently private; access/publication is not implied.
- Before claiming the GitHub work loop is running, separately verify that the existing [TASS Issue #24 relay](https://github.com/Arameair/TASS/issues/24) supports the required scoped task interaction.
