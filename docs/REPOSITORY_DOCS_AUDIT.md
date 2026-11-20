# Repository documentation readiness — TASS development loop

**Observed:** 2026-11-19 through the linked GitHub connection. **Type:** source documentation inventory, not a runtime/installation audit or live-agent state. The checks below looked at repository visibility, default branches, root README and available documentation directories. Open drafts are not treated as adopted policies.

| Repository | Inspected default branch and documentation | Assessment | Recommended follow-up |
| --- | --- | --- | --- |
| [Thought-Processor](https://github.com/Arameair/Thought-Processor) | `main` `53e0857`: `AGENTS.md`, `docs/WORKFLOW.md`, `docs/STATUS.md`, `docs/README.md` | Shared coordination draft in [PR #130](https://github.com/Arameair/Thought-Processor/pull/130) includes team register and security review playbook, with Stage 1 reconciliation; **not adopted**. Plan [#133](https://github.com/Arameair/Thought-Processor/pull/133) and demos [#131](https://github.com/Arameair/Thought-Processor/pull/131)/[#132](https://github.com/Arameair/Thought-Processor/pull/132) are proposals | Focused independent doc audit; owner merge decision; verify links after merge |
| [TASS](https://github.com/Arameair/TASS) | `main` still has old October Discord landing; `live` contains owner-approved `tass.md` and lessons. Repository currently private | Updated landing, public-safe overview and prospective `team/TASS.md` + log are **drafts in [TASS PR #27](https://github.com/Arameair/TASS/pull/27)**. [TASS PR #28](https://github.com/Arameair/TASS/pull/28) (draft, `83f5cc8`) is a separate README refresh for the **`live`** branch (#27 targets `main`), not a competing change; [#31](https://github.com/Arameair/TASS/pull/31) is a separate `live/tass.md` privacy cleanup | Review private-to-public privacy boundaries; #27 (`main`) and #28 (`live`) are reviewed and merged independently, each after its own audit; never overwrite `live/tass.md` |
| [Thought-engine](https://github.com/Arameair/Thought-engine) | Default `experiment/thought-engine-v1-20261023` README still describes original Chat-Jimmy experiment; independent new branches exist | **STALE LANDING**; [PR #18](https://github.com/Arameair/Thought-engine/pull/18) is a draft, based on older experimental revision | Determine correct accepted source baseline before refreshing canonical docs; do not casually rebase or merge old draft |
| [Identity](https://github.com/Arameair/identity) | `main` is public documentation, README describes Oct 27 P0/P1 state; `alpha` contains newer owner-promoted code | **STALE LANDING**; [PR #41](https://github.com/Arameair/identity/pull/41) is a draft docs refresh | Review against actual accepted `alpha` grants/source and current PR #47 status before promoting status text |
| [Communication-Bridge](https://github.com/Arameair/Communication-Bridge) | Private `main` has a brief README; larger docs/tests on experimental branch; [PR #3](https://github.com/Arameair/Communication-Bridge/pull/3) is security-sensitive and under review | **SPLIT DOCUMENTATION** | Do not amend audit candidate solely for doc housekeeping; after acceptance, make a scoped landing/index follow-up |
| [updater](https://github.com/Arameair/updater) | Private `main` `ffef310`: README still calls project design-only, although [PR #2](https://github.com/Arameair/updater/pull/2) (read-only eligibility, Milestone 1) merged 2026-11-18 | **STALE STATUS TEXT**; code presence does not prove installability | Update README to accurately describe implemented *read-only eligibility* and leave installation explicitly unapproved |
| [Development](https://github.com/Arameair/Development) | Private `main` README only names the project; prototype work on experimental branches | **MISSING ENTRYPOINT** | Minimal purpose, active branch links, current scope and common workflow pointer when next worked on |
| [Avatar](https://github.com/Arameair/Avatar) | Private `main` README explains experimental visual presence; several prototype branches | **PARTIAL ENTRYPOINT** | Add evidence-linked prototype index and cross-team workflow pointer when next worked on |

## Stage 1 retain / repair / supersede decisions (2026-11-19)

Decisions applied in PR #130 and TASS PR #27 by the Stage 1 commits. "Proposed" items need their own separately approved, scoped change; Stage 1 does not edit other repositories' landings or any security candidate.

| Path | Decision | Reason |
| --- | --- | --- |
| `AGENTS.md` | RETAIN (reviewed, no change) | Read-first rules agree with WORKFLOW; procedure lives in WORKFLOW |
| `docs/WORKFLOW.md` | REPAIR | Stale "current source milestone" (`0acb9d3`) now points to STATUS. Duplicate task-start rule merged into one. Adds that sessions are owner-started and that demos are not operational, plus the reviewer-provenance rule |
| `docs/TEAM_ACTIVE.md` | REPAIR | Two-workstream handoff view with exact candidates, verdict state, blockers and next actions; provenance caveat |
| `docs/DEFENSIVE_REVIEW_PLAYBOOK.md` | REPAIR (one row) | Independent review: reviewer identity is owner-attested, not proven by a comment author |
| `docs/STATUS.md` | REPAIR | `main` moved from `0acb9d3` to `53e0857` (#125, #126, #128); records actual post-merge CI. Installed baseline unchanged, still not re-inspected |
| `docs/README.md` | REPAIR | Stale "PR #123 accepted source stack"; adds plan/demos as non-operational and lists superseded docs |
| `core/docs/TEAM_TASS_WORKFLOW.md` | SUPERSEDE (banner, content kept) | Nov 4 team notes with dated candidates and sync/pulse states |
| `docs/AI_COLLABORATION.md` | RETAIN + label | Disclosure/verification policy still valid; Oct 27 status paragraph labelled historical |
| `docs/INTEGRATION_AUTHORIZATION_PLAN.md` | RETAIN (no change) | Identity-to-Engine authorization design; not this workloop |
| `docs/history/legacy-agent-instructions/` | RETAIN (no change) | Already labelled historical by AGENTS.md |
| `teams` branches; `experiment/tass-core-team-claude-20261031`; `docs/tass-team-charters-20261031` | SUPERSEDE (listed in docs/README.md; branches untouched) | Earlier role-spec and team-cycle attempts |
| TASS `README.md`, `AGENTS.md`, `docs/README.md`, `docs/TEAM_LOOP.md`, `team/TASS.md` (PR #27) | REPAIR (TASS #27) | Align with plan #133: TASS participation is Stage 3 only; demos are not operational |
| Thought-engine README, Identity `main` README, updater README, Development README, Avatar README | PROPOSED, not changed | Stale or missing landings. Use existing drafts (Engine #18, Identity #41) where their bases still fit, under separate approval |
| Bridge #3, door #129 | NO CHANGE | Active security candidates under exact-SHA audit |

## Canonical Markdown responsibilities

- **Who and what:** [TEAM_ACTIVE.md](TEAM_ACTIVE.md) is the coordination index for named workstreams; no assumption that a desktop agent is currently running.
- **How:** [WORKFLOW.md](WORKFLOW.md), with continuous BUILD → TEST → INDEPENDENT AUDIT → REPAIR → RETEST → RE-AUDIT across parallel workstreams.
- **Security:** [DEFENSIVE_REVIEW_PLAYBOOK.md](DEFENSIVE_REVIEW_PLAYBOOK.md), including independent acceptance evidence and recurring AI access interruptions.
- **Project fact:** [STATUS.md](STATUS.md), with source/release/installed distinctions.
- **TASS-facing publication proposal:** [TASS team loop design](https://github.com/Arameair/TASS/blob/docs/tass-main-landing-refresh-20261106/docs/TEAM_LOOP.md), [public-safe overview](https://github.com/Arameair/TASS/blob/docs/tass-main-landing-refresh-20261106/docs/PUBLIC_OVERVIEW.md). These links require repository access; TASS repository visibility is currently private.

## Next acceptance milestones

1. Independently review Core PR #130 and TASS PR #27 for scope, link validity, no unapproved private data and no claims of completed runtime behavior; owner decides any merge independently.
2. Inventory Go Core's current Issue #24 relay, `github.sync` authorization, pulse scheduler, durable receipts and conflict handling. Avoid a new transport if the existing relay can be extended safely.
3. Prove a **read-only** TASS observer reading one approved team assignment at an exact revision, then owner-approved draft/acknowledgment. Require tests for idempotency, stale revisions, authorization, privacy and backoff before GitHub-writing mode.
4. Tackle the separate component landing-refresh gaps in their *own approved workstreams*, without resetting active security audits or making a burst of new PRs.

This snapshot should be refreshed when a source status materially changes, not automatically every local pulse. Agents do not gain access to other agents' desktop conversations simply by reading Markdown.

## Immediate cleanup order — unblock the work cycle without PR sprawl

**Gate 1: Make the coordination contract authoritative.** Independently audit and obtain owner approval for [Core PR #130](https://github.com/Arameair/Thought-Processor/pull/130). Confirm the canonical Team/Workflow/Security/Status references are accurate and minimal. A draft document is not an enforced policy.

**Gate 2: Update TASS's project landing safely.** Independently audit and obtain owner approval for [TASS PR #27](https://github.com/Arameair/TASS/pull/27), whose public-safe content remains private while the repository is private. Preserve `live/tass.md`, lessons, memory boundaries and the current issue relay. Do not combine this merge with a repository visibility change.

**Gate 3: Repair stale component entrypoints with exact accepted source.** Engine's default branch still documents a Chat-Jimmy prototype even though Core's supported Compose explicitly chooses `-worker nemotron`. Identity's `main` landing under-reports promoted `alpha`. Updater's README still describes a design-only status. Prefer updating their existing documentation proposals where compatible; for mismatched bases, make narrowly scoped later proposals without rebasing or merging unrelated old experiments. Development, Avatar and Bridge need minimal truthful links, not duplicate project handbooks.

**Gate 4: Demonstrate one real bounded GitHub collaboration exchange.** Inspect the accepted Go Core relay and `github.sync` contract before coding. Current Compose defaults to `GITHUB_BRANCH=live`, `GITHUB_DOC=tass.md`; the proposed TASS mailbox is **separate** (`main:team/TASS.md`). Never replace the identity-sync target to force team coordination. Test a supervised read and acknowledgment, permission gating, revisions, idempotency, backoff and no private-memory leakage. Audit independently and obtain separate owner activation approval.

**Gate 5: Extend from one supervised handoff to parallel workstreams.** Once one exchange is real and evidenced, let TASS track multiple approved tasks while Claude builds, Codex audits an immutable candidate, and GPT-6 coordinates. Each participant posts bounded status to its authorized record; no agent self-approves its own privileges or merge.

**What is blocked:** A complete project-development loop is **NOT VERIFIED** until Gate 4 passes. The documentation-only gates do not require Discord PR #3 or door PR #129 to be weakened, merged or deployed.
