# Thought-Processor contributor and AI-agent instructions

These are standing repository instructions, not a disposable stabilization candidate. Read [README.md](README.md), [docs/STATUS.md](docs/STATUS.md), and [docs/WORKFLOW.md](docs/WORKFLOW.md) before changing source or reporting repository status. [Issue #119](https://github.com/Arameair/Thought-Processor/issues/119) records owner decisions and evidence; [Issue #120](https://github.com/Arameair/Thought-Processor/issues/120) records accepted Stage 3 evidence. PR #118 is a proposal, not adopted policy.

## Scope and authority

- Follow the owner's **current, explicit** task authorization. A PR, discussion, or previous approval does not authorize later merge, deployment, branch deletion, repo settings changes, or Identity grant changes.
- Keep Core, Thought Engine, and Identity responsibilities separate. Identity owns canonical records, permissions, and revocation; the Core UI and adapters do not acquire that authority.
- Favor a single bounded issue and PR per task; record exact source SHAs, path scope, relevant tests, and remaining blockers. Do not broaden scope or refactor unrelated services to simplify a review.
- Preserve auditable history. Do not silently rebase, force-push, delete branches, close PRs, or adopt old policy without approval. Check unique history and references before cleanup.

## Safety, evidence, and privacy

- Do not access or modify running TASS, local model, owner data, credentials, grants, ports, backups, or service state without specific permission. Use isolated fixtures for untrusted tests.
- Treat AI suggestions, model output, UI display, and unverified logs as observations, not proof of authorized execution or Identity decisions.
- Do not publish secrets, private workspace paths, personal contact information, owner-private records, or unsanitized author/committer metadata. Verify public Git commit identity and newly exposed history before a public push.
- Match test claims to execution: PASS / FAIL / NOT RUN, with exact SHA, workflow event, and tested ref (branch head versus synthetic merge) as relevant. A draft PR does not automatically suppress CI.
- The author must not claim to be an independent auditor. Record evidence and limitations; obtain proportionate separate review before protected operations.
- A source merge is not a release or installation. Never change release pins, authorization grants, autonomy or deployment state based solely on merged GitHub source.

## Historical instructions

Older P0 operator instructions remain available only as evidence at `docs/history/legacy-agent-instructions/AGENTS_P0_WINDOWS_RUNBOOK.md`. They are not active operating requirements. See the current [workflow](docs/WORKFLOW.md) instead.
