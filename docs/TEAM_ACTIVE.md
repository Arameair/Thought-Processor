# Active team and work register

**Record date:** 2026-11-19 (refs observed via GitHub at 13:30 UTC). **Status:** proposed in PR #130, not yet adopted in `main`.

This file answers **who is working on what, why, and what is blocked**. It is a manual, evidence-linked coordination index, **not** a live agent tracker. Nobody here runs continuously: every Claude, Codex and GPT-6 session is started by the owner, does one bounded task, records its result in the owning PR, and stops. Confirm current heads and comments on GitHub before acting; this table can be behind them.

**Provenance caveat:** Claude, Codex and GPT-6 comments are all posted through the owner's GitHub account. The account authenticates `Arameair`, not which assistant wrote a comment. A verdict labelled "independent" is owner-attested, not cryptographically verified. See [Plan PR #133 §0](https://github.com/Arameair/Thought-Processor/pull/133).

## Active handoffs — two independent workstreams

| | **WS-1 · Discord security repairs** | **WS-2 · Stage 1 documentation consolidation** |
| --- | --- | --- |
| Scope | Durability/deduplication repairs: Bridge source index and door source claims | Reconcile TEAM / WORKFLOW / SECURITY / PROJECT records; no runtime code ([Plan #133](https://github.com/Arameair/Thought-Processor/pull/133) Stage 1) |
| Implementer | Claude | Claude (owner-authorized 2026-11-19 to commit on PR #130 and TASS #27) |
| Independent auditor | Codex | Codex |
| Exact candidates | [Bridge PR #3](https://github.com/Arameair/Communication-Bridge/pull/3) `2b608cbde2582f0a80399103cf5522b3782d04b6` · [Door PR #129](https://github.com/Arameair/Thought-Processor/pull/129) `f02c80de89b2f978c9814943d079353972cbdff4` | Head of this PR (#130) and of [TASS PR #27](https://github.com/Arameair/TASS/pull/27) after the Stage 1 commit; exact SHAs are in each PR description (a commit can't name its own SHA) |
| Latest verdict | NEEDS REPAIR on the **previous** heads (Bridge `39985b2`, [comment 6994737957](https://github.com/Arameair/Communication-Bridge/pull/3#issuecomment-6994737957); door `b01da88`, [comment 6994739148](https://github.com/Arameair/Thought-Processor/pull/129#issuecomment-6994739148)). Current heads: **no verdict yet** | None yet |
| Author-reported evidence | Bridge: vet/build/test/race PASS (no CI in that repo). Door: 76/76 on Python 3.12 + 3.13; `discord-door-tests` + `core-tests` CI success at `f02c80d` | Docs only: link, privacy and scope checks in the PR description; runtime tests NOT RUN (none apply) |
| Blocker | Independent re-audit of the exact current heads | Independent documentation review; owner disposition of #130 and #27 |
| Next action | Codex re-audits both heads and posts PASS / NEEDS REPAIR / BLOCKED. On NEEDS REPAIR, Claude repairs the same branches | Codex reviews accuracy, privacy, links and scope. On findings, Claude repairs the same branches and requests re-review |
| Must not | Change either candidate during its audit; touch the docs workstream | Edit either Discord candidate; claim any workloop or TASS sync is running |

The two workstreams touch disjoint repositories and paths. One audit can proceed while the other repairs.

## Other open items

| Item | Owner / role | State at observed refs | Next |
| --- | --- | --- | --- |
| [Identity PR #47](https://github.com/Arameair/identity/pull/47) — Discord `discord.reply` decision | Claude author; Codex auditor | Independent **PASS** at `a67064380cb55d1b55f9e8454cf2f60d12c6b306` ([comment 6975055740](https://github.com/Arameair/identity/pull/47#issuecomment-6975055740)); draft, unmerged | Owner decision; not a team-publishing grant |
| [Plan PR #133](https://github.com/Arameair/Thought-Processor/pull/133) — three-stage loop plan | GPT-6 author; reviewer unassigned | Draft at `7f37ce3224fb51e3d745a8486837c3bce6843189`; Stage 1 authorized, Stages 2–3 not | Independent plan review; Stage 2 only after Stage 1 is accepted |
| Workloop demos [PR #131](https://github.com/Arameair/Thought-Processor/pull/131) (GPT-6, `b47ead0`) and [PR #132](https://github.com/Arameair/Thought-Processor/pull/132) (Claude, `dcea9a9`) | Authors as named; no independent reviewer yet | **EXPERIMENTAL** fixtures; author-reported tests only; not wired into Core | Reference for Stage 2; neither is operational or adopted |
| TASS GitHub participation | Not assigned | Not implemented. [Issue #24](https://github.com/Arameair/TASS/issues/24) relay exists for conversation only; `team/TASS.md` is an inactive placeholder in TASS #27 | Plan Stage 3, separately authorized |
| Discord Core adapter / Compose | Not assigned | Deferred | After WS-1 passes and the owner authorizes ([#127](https://github.com/Arameair/Thought-Processor/issues/127)) |

## How to maintain this register

1. **Start:** every contributor reads this file with `AGENTS.md`, `docs/WORKFLOW.md`, relevant `docs/STATUS.md`, `docs/DEFENSIVE_REVIEW_PLAYBOOK.md`, and active issue/PR evidence. Confirm whether the listed head and assignment are still true.
2. **Ownership:** assign a named role to each workstream (implementer, independent auditor, owner decision). Do not infer that an agent is actively executing from an old comment or chat.
3. **Update every cycle:** after BUILD/TEST/AUDIT or REPAIR/RETEST/RE-AUDIT, the responsible authorized contributor synchronizes the affected team row and existing PR discussion: dated revision, exact result, responsible next role, blocker and next action. Do not claim people or agents are actively executing without current evidence.
4. **Audit write boundary:** an authorized independent auditor may publish their review and update this coordination register or audit docs; they may not modify the code candidate they are independently approving. Explicit read-only audits provide proposed row changes to a writer instead.
5. **Conflict resolution:** component PRs own their exact-SHA code and test evidence; this table is only a coordination index. `docs/STATUS.md` owns accepted project/release/installation state, not live task allocation. Never overwrite a newer review with an older summary.
6. **Privacy:** no tokens, user records, private paths, sensitive diagnostic transcripts or claims of access to another agent's local environment.

**Owner gate:** A task marked DONE here is not automatically authorization to merge, enable grants, release, deploy, or install.
