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

Contributors read the canonical sources at the start of every task and update affected records at meaningful status transitions **only when authorized**. Read-only auditors propose updates, not commits. PR #130 is a proposal until reviewed and merged.
