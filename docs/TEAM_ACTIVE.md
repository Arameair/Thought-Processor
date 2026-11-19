# Active team and work register

**Record date:** 2026-11-19. **Status:** proposed in PR #130, not yet adopted in `main`.

This file answers **who is working on what, why, and what is blocked**. It is a manual, evidence-linked coordination summary, **not** a live agent tracker or an assertion that any assistant is currently running. Confirm recent changes with the referenced PRs and comments before acting.

| Workstream | Role / responsible party | Current activity and state | Next handoff | Evidence |
| --- | --- | --- | --- | --- |
| Discord Bridge | Claude — implementing candidate; Codex — independent reviewer | Owner-reported Claude repair at `2b608cbde2582f0a80399103cf5522b3782d04b6`; independent re-audit not confirmed here | Codex reviews exact latest head and records PASS / NEEDS REPAIR / BLOCKED | [Bridge PR #3](https://github.com/Arameair/Communication-Bridge/pull/3) |
| Discord door | Claude — implementing candidate; Codex — independent reviewer | Owner-reported Claude repair at `f02c80de89b2f978c9814943d079353972cbdff4`; independent re-audit not confirmed here | Codex reviews exact latest head and records PASS / NEEDS REPAIR / BLOCKED | [Door PR #129](https://github.com/Arameair/Thought-Processor/pull/129) |
| Identity Discord authorization | Claude — author; Codex — auditor | Independent PASS previously reported for `a67064380cb55d1b55f9e8454cf2f60d12c6b306`; PR remains a separate promotion/deployment decision | Preserve accepted contract; do not treat as live Discord acceptance | [Identity PR #47](https://github.com/Arameair/identity/pull/47) |
| Security-review workflow documentation | GPT-6 — documentation author; independent reviewer unassigned | Draft PR #130; two owner-reported desktop access interruptions recorded in the playbook | Independent focused documentation review and separate owner merge decision | [Documentation PR #130](https://github.com/Arameair/Thought-Processor/pull/130) |
| Discord Core adapter / Compose | No active implementer assigned | Deferred until Bridge and door independently pass | Owner authorizes separate implementation slice | [Milestone A issue #127](https://github.com/Arameair/Thought-Processor/issues/127) |

## How to maintain this register

1. **Start:** every contributor reads this file with `AGENTS.md`, `docs/WORKFLOW.md`, relevant `docs/STATUS.md`, `docs/DEFENSIVE_REVIEW_PLAYBOOK.md`, and active issue/PR evidence. Confirm whether the listed head and assignment are still true.
2. **Ownership:** assign a named role to each workstream (implementer, independent auditor, owner decision). Do not infer that an agent is actively executing from an old comment or chat.
3. **Update:** when a task starts, changes owner, reaches a gate, becomes blocked or ends, the authorized writer updates only the affected row and supporting PR discussion. Record the date, actual status, exact link/ref, and next action. No meaningless per-prompt commits.
4. **Read-only mode:** an auditor who lacks write authorization produces a proposed row change in the audit handoff. A separately authorized contributor applies it. Read-only means no repository mutation.
5. **Conflict resolution:** component PRs own their exact-SHA code and test evidence; this table is only a coordination index. `docs/STATUS.md` owns accepted project/release/installation state, not live task allocation. Never overwrite a newer review with an older summary.
6. **Privacy:** no tokens, user records, private paths, sensitive diagnostic transcripts or claims of access to another agent's local environment.

**Owner gate:** A task marked DONE here is not automatically authorization to merge, enable grants, release, deploy, or install.
