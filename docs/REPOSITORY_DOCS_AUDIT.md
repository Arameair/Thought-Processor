# Repository documentation readiness — TASS development loop

**Observed:** 2026-11-19 through the linked GitHub connection. **Type:** source documentation inventory, not a runtime/installation audit or live-agent state. The checks below looked at repository visibility, default branches, root README and available documentation directories. Open drafts are not treated as adopted policies.

| Repository | Inspected default branch and documentation | Assessment | Recommended follow-up |
| --- | --- | --- | --- |
| [Thought-Processor](https://github.com/Arameair/Thought-Processor) | `main`: `AGENTS.md`, `docs/WORKFLOW.md`, `docs/STATUS.md`, `docs/README.md` | Shared coordination draft in [PR #130](https://github.com/Arameair/Thought-Processor/pull/130) includes team register and security review playbook; **not adopted** | Focused independent doc audit; owner merge decision; verify links after merge |
| [TASS](https://github.com/Arameair/TASS) | `main` still has old October Discord landing; `live` contains owner-approved `tass.md` and lessons. Repository currently private | Updated landing, public-safe overview and prospective `team/TASS.md` + log are **drafts in [TASS PR #27](https://github.com/Arameair/TASS/pull/27)** | Review private-to-public privacy boundaries; owner chooses doc promotion; never overwrite `live/tass.md` |
| [Thought-engine](https://github.com/Arameair/Thought-engine) | Default `experiment/thought-engine-v1-20261023` README still describes original Chat-Jimmy experiment; independent new branches exist | **STALE LANDING**; [PR #18](https://github.com/Arameair/Thought-engine/pull/18) is a draft, based on older experimental revision | Determine correct accepted source baseline before refreshing canonical docs; do not casually rebase or merge old draft |
| [Identity](https://github.com/Arameair/identity) | `main` is public documentation, README describes Oct 27 P0/P1 state; `alpha` contains newer owner-promoted code | **STALE LANDING**; [PR #41](https://github.com/Arameair/identity/pull/41) is a draft docs refresh | Review against actual accepted `alpha` grants/source and current PR #47 status before promoting status text |
| [Communication-Bridge](https://github.com/Arameair/Communication-Bridge) | Private `main` has a brief README; larger docs/tests on experimental branch; [PR #3](https://github.com/Arameair/Communication-Bridge/pull/3) is security-sensitive and under review | **SPLIT DOCUMENTATION** | Do not amend audit candidate solely for doc housekeeping; after acceptance, make a scoped landing/index follow-up |
| [updater](https://github.com/Arameair/updater) | Private `main`: README still calls project design-only, while implementation/evidence exists in repository | **STALE STATUS TEXT**; code presence does not prove installability | Update README to accurately describe implemented *read-only eligibility* and leave installation explicitly unapproved |
| [Development](https://github.com/Arameair/Development) | Private `main` README only names the project; prototype work on experimental branches | **MISSING ENTRYPOINT** | Minimal purpose, active branch links, current scope and common workflow pointer when next worked on |
| [Avatar](https://github.com/Arameair/Avatar) | Private `main` README explains experimental visual presence; several prototype branches | **PARTIAL ENTRYPOINT** | Add evidence-linked prototype index and cross-team workflow pointer when next worked on |

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
