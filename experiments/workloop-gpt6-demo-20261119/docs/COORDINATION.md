# Independent approach: one task, two single-writer receipts, one derived view

**Proposal only; not adopted project policy.** Avoid a central Markdown file that Claude, Codex, GPT-6, and TASS all overwrite. Keep authority in owner decisions and Identity, code in its owning repository, and the candidate/audit evidence in the owning PR. The coordination files only point to evidence.

## Reuse before building

- Existing Thought-Processor PR #130 proposes `docs/TEAM_ACTIVE.md`, `docs/WORKFLOW.md`, `docs/DEFENSIVE_REVIEW_PLAYBOOK.md`, `docs/STATUS.md`. These are *draft* canonical roles, not adopted policy.
- Existing TASS PR #27 proposes `team/TASS.md` plus `team/TASS_LOG.md`, both inactive. The private TASS `live:tass.md` identity/memory-facing doc is a **different target**.
- Existing Go Core has a GitHub client with ETag reads, SHA-conditional file writes and issue comments, and an Issue #24 relay with Identity checks and durable local receipts. Existing Engine pulse and Core synchronization are separate; the code does not implement this team workflow.
- Thought-Processor `core/compose.yaml` currently defaults to `GITHUB_BRANCH=live`, `GITHUB_DOC=tass.md`, Engine `-pulse 20s` and worker `nemotron`. Nothing points to TASS `main:team/TASS.md`.

## Proposed GitHub coordination record ownership

| Place | Writer | Purpose | Prohibition |
| --- | --- | --- | --- |
| Existing issue/PR | Owner, builder, auditor with appropriate access | Approved task and immutable candidate SHA, test output and review verdict | No new grants/merge authority from comments |
| Core `docs/TEAM_ACTIVE.md` | Explicitly designated coordinator | Derived, small latest-state index: owner, assignment, candidate SHA, next handoff | Not a task queue or accepted release status |
| Core `docs/WORKFLOW.md` | Reviewed documentation change | Stable BUILD → TEST → AUDIT → REPAIR → RETEST → RE-AUDIT procedure | No per-cycle log churn |
| Core `docs/DEFENSIVE_REVIEW_PLAYBOOK.md` | Authorized security-doc reviewer | Security gates and blockers | No secrets or unchecked incident assertions |
| Core `docs/STATUS.md` | Accepted-project-state maintainer | Verified accepted/integrated state | No draft task status or fake runtime status |
| Candidate workstream `<task>/Claude.md` | Claude only | Builder revisions and actual tests | Cannot publish independent audit verdict |
| Candidate workstream `<task>/Codex.md` | Codex only | Exact-SHA audit, failures, re-audit | Cannot edit candidate under review |
| Candidate workstream `<task>/GPT6.md` | GPT-6 only | Coordination decision, evidence links | No executable authority |
| Proposed TASS `team/TASS.md` | Future TASS adapter only | Sanitized task acknowledgment and work status | No local private memory; not connected today |

The last three per-workstream paths are *illustrative* of how independent receipts avoid writer collisions; their real repository placement should reuse existing issue/PR review records rather than create mandatory files for every small task. In this fixture, actor records are in `records/run/handoffs/WS-A` and `/WS-B`.

## Per-task transition checks

1. Start only from an explicitly approved, bounded task with a designated builder and a *different* reviewer. Read current GitHub refs and relevant Markdown/PRs; never equate a Markdown claim of approval with owner authentication.
2. Builder publishes a candidate revision and real test evidence. The auditor reads and pins the candidate SHA before the audit. Concurrent independent workstreams need distinct files and branches/PRs.
3. Auditor records PASS or NEEDS_REPAIR against **that exact revision**. A failure returns to the same builder. Repair changes the digest and invalidates any earlier PASS.
4. Repeat retest/re-audit until independently accepted or blocked. Coordinator updates only derived current-state index; no merge/deploy follows automatically.
5. A future TASS adapter reads approved work using a distinct scoped allowlist and verified task ID/input revision, checks Identity grants at the receiver, generates a redacted proposal, and conditionally writes **only** its own file if owner-authorized. Dedupe by `(task_id, source_revision, action)`; no commit on no change.

## Required real integration checks, not covered by this fixture

- Existing `live:tass.md` sync and Issue #24 relay remain unchanged; validate baseline before any new adapter.
- Identity grant: missing, expired, revoked, offline, or wrong-scope must prevent outbound effects at the point of writing.
- Remote file conflicts (GitHub blob SHA mismatch) preserve both versions, never silent overwrite; define reviewer ownership and task priority.
- Durable local queue and receipt persistence across crash, retry and ambiguous HTTP outcomes; prevent duplicate notifications.
- Sanitization must use strict approved-field serialization, not just keyword filtering. Protect author metadata and links as well as document body.
- Backoff and rate-limit handling independent from 20-second Engine pulse; no unnecessary writes.
- Owner verification, real independent audit, branch controls and separate approval before merge/deployment.

## Principal design weaknesses

**[1] GitHub is not a transactional event bus.** Multi-file updates and cross-repo read order can be stale. Derived summary is advisory; per-task exact SHA and PR evidence govern.

**[2] Apparent single writer is social until enforced.** Branch protections, least-privilege credentials and remote CAS are needed. The fixture's locks and role labels do not authenticate humans or agents.

**[3] Identical GitHub login for multiple assistants cannot prove which model wrote a message.** Require explicit attributed receipts, provenance, and separate human confirmation of sensitive decisions.

**[4] An auditor can miss a bug.** Fixture tests cover only listed inputs, not broad security or repository integration. Independent means different actor and candidate immutability, not an absolute guarantee of correctness.

**[5] TASS 20-second pulses can amplify spam, races and ambiguous retries.** Read and publish on actual changes with bounded queues, backoff, and persistent deduplication. Never stream raw thoughts or private memory.
