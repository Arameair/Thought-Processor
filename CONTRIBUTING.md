# Contributing to TASS / Thought Processor

**Start here for human engineers and AI contributors.** This file is a review proposal in [governance PR #118](https://github.com/Arameair/Thought-Processor/pull/118), **not a claim that the new branch convention has been fully migrated**.

## Before you start

1. Read [the workflow](docs/WORKFLOW.md), [cross-repo roles/release policy](docs/TASS_REPOSITORY_RELEASE_POLICY.md), and the current task's issue and implementation PR.
2. Identify **which source is accepted, which exact SHAs are installed, and which exact branch/commit you were assigned**. Historical defaults, old README status, and branch names are not sufficient evidence.
3. Check existing issues/PRs before creating another ticket. Fix one coherent problem at a time.
4. When working across Core, Engine, Identity, Cognition and Development, link one coordinating issue and pin compatible component SHAs. Identity is the permission authority; no other service or AI contributor grants itself capabilities.

## Which GitHub object do I create?

| Need | Use |
| --- | --- |
| Report a defect, suggest a feature, ask a design question, track cleanup | **Issue** |
| Propose a real change to source, tests, or durable documentation | **One focused draft PR** |
| Audit another PR and find a defect | **Comment on that PR** with repro and exact SHA |
| Fix a failed PR audit | **Add a commit to the same PR**, when possible |
| Preserve a historical decision or retired experiment | **Existing issue/documentation**, linked to the original closed PR |

Do not create a second PR just to announce that the first PR was audited. A report artifact may have its own PR **only when that file itself is meant to be merged**.

## Every PR should answer six questions

- **Why?** Linked issue, observed problem and expected outcome.
- **What?** Scope, files changed and explicit non-goals.
- **Where?** Repo, head/base full SHA, dependency PRs and compatible service pins.
- **How verified?** Exact test commands and PASS/FAIL/NOT RUN; fixture vs real services/model vs owner trial.
- **What's next?** One accountable person/action; `DONE / OPEN / NEXT / BLOCKED`.
- **Who decides?** Review/merge versus owner-authorized install/grants versus external release are **separate approvals**.

Do not call a visual prototype real cognition, a passing mock an installed success, or an audit of an old SHA approval of a new SHA.

## When the work finishes

Merge only when authorized and the gate is met. Otherwise close with an explicit reason (superseded, rejected, historical), a successor/reference and preserved relevant evidence. Never delete a branch or user data as routine cleanup.

At a stage checkpoint, consult the [PR maintenance checklist](docs/TASS_ISSUE_PR_WORKFLOW.md) and [dated inventory](docs/OPEN_PR_TRIAGE_20261112.md). Count opened, closed and merged PRs separately. Prefer one coordinated cleanup issue to many repetitive PRs.

## Release, safety and legacy migration

The **target** convention is reviewed source on `main`, short-lived `experiment/*` branches, and a separate owner-approved external-trial `stable` release pinned by tags and a cross-repository manifest. Existing `alpha`, `main`, `stable` and default-branch states are **not** automatically migrated. Do not rebase active work, change branch protections, merge, deploy, alter Identity grants, run the real model, enable autonomy or handle private owner data without the applicable approval.

For detailed rules and transition steps: [WORKFLOW.md](docs/WORKFLOW.md). For historical context, [STATUS.md](docs/STATUS.md) is an October 27 snapshot and **not a current installed-build report**.

## Working in parallel and preparing updates

Several engineers or AIs can develop simultaneously **on different local worktrees/branches and PRs**. Each person owns their branch, links their issue, and identifies cross-repo dependencies. No contributor may silently overwrite another's candidate, share the owner's test ports/data, or assume independent green tests prove the combined application works. The release coordinator verifies exact Core/Engine/Identity pins before a candidate is eligible for an update.

For the first safe updater trial, see [coordinated release/update pilot](docs/UPDATE_RELEASE_WORKFLOW.md) and [existing issue #79](https://github.com/Arameair/Thought-Processor/issues/79). The current script already has update/recovery mechanisms; do not build a second updater. A PR merge is not owner approval to update.
