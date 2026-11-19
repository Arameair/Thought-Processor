# Active team and work register

**Record date:** 2026-11-19. **Status:** proposed in PR #130, not yet adopted in `main`.

This file answers **who is working on what, why, and what is blocked**. It is a manual, evidence-linked coordination summary, **not** a live agent tracker or an assertion that any assistant is currently running. Confirm recent changes with the referenced PRs and comments before acting.

| Workstream | Role / responsible party | Current activity and state | Next handoff | Evidence |
| --- | --- | --- | --- | --- |
| Discord Bridge | Claude — implementing candidate; Codex — independent reviewer | Owner-reported Claude repair at `2b608cbde2582f0a80399103cf5522b3782d04b6`; independent re-audit not confirmed here | Codex reviews exact latest head and records PASS / NEEDS REPAIR / BLOCKED | [Bridge PR #3](https://github.com/Arameair/Communication-Bridge/pull/3) |
| Discord door | Claude — implementing candidate; Codex — independent reviewer | Owner-reported Claude repair at `f02c80de89b2f978c9814943d079353972cbdff4`; independent re-audit not confirmed here | Codex reviews exact latest head and records PASS / NEEDS REPAIR / BLOCKED | [Door PR #129](https://github.com/Arameair/Thought-Processor/pull/129) |
| Identity Discord authorization | Claude — author; Codex — auditor | Independent PASS previously reported for `a67064380cb55d1b55f9e8454cf2f60d12c6b306`; PR remains a separate promotion/deployment decision | Preserve accepted contract; do not treat as live Discord acceptance | [Identity PR #47](https://github.com/Arameair/identity/pull/47) |
| Shared workflow and repository documentation | GPT-6 — documentation author; independent reviewer unassigned | Draft Core PR #130 now includes a dated [repo documentation inventory](REPOSITORY_DOCS_AUDIT.md), team register, security playbook and doc sync rules | Independent focused doc review and separate owner merge decision | [Core PR #130](https://github.com/Arameair/Thought-Processor/pull/130) |
| TASS GitHub team participation | GPT-6 — documentation author; runtime implementer not assigned | [TASS PR #27](https://github.com/Arameair/TASS/pull/27) proposes public-safe docs, a dedicated unconnected mailbox and log; `live/tass.md` unchanged | Owner-reviewed docs, inspect Issue #24 relay, then authorize a bounded implementation/test/audit slice | [TASS Issue #24](https://github.com/Arameair/TASS/issues/24) |
| Discord Core adapter / Compose | No active implementer assigned | Deferred until Bridge and door independently pass | Owner authorizes separate implementation slice | [Milestone A issue #127](https://github.com/Arameair/Thought-Processor/issues/127) |

## How to maintain this register

1. **Start:** every contributor reads this file with `AGENTS.md`, `docs/WORKFLOW.md`, relevant `docs/STATUS.md`, `docs/DEFENSIVE_REVIEW_PLAYBOOK.md`, and active issue/PR evidence. Confirm whether the listed head and assignment are still true.
2. **Ownership:** assign a named role to each workstream (implementer, independent auditor, owner decision). Do not infer that an agent is actively executing from an old comment or chat.
3. **Update every cycle:** after BUILD/TEST/AUDIT or REPAIR/RETEST/RE-AUDIT, the responsible authorized contributor synchronizes the affected team row and existing PR discussion: dated revision, exact result, responsible next role, blocker and next action. Do not claim people or agents are actively executing without current evidence.
4. **Audit write boundary:** an authorized independent auditor may publish their review and update this coordination register or audit docs; they may not modify the code candidate they are independently approving. Explicit read-only audits provide proposed row changes to a writer instead.
5. **Conflict resolution:** component PRs own their exact-SHA code and test evidence; this table is only a coordination index. `docs/STATUS.md` owns accepted project/release/installation state, not live task allocation. Never overwrite a newer review with an older summary.
6. **Privacy:** no tokens, user records, private paths, sensitive diagnostic transcripts or claims of access to another agent's local environment.

**Owner gate:** A task marked DONE here is not automatically authorization to merge, enable grants, release, deploy, or install.
