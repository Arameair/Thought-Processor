# TASS open contribution workflow: human + entity + microservice maintainer

**Status: OWNER-SELECTED DIRECTION / DESIGN ONLY — 2026-11-12.** This document captures the owner's goal that more authorized contributors can choose and work on suitable GitHub issues with the assistance of **their own entity**. It is proposed as part of [governance PR #118](https://github.com/Arameair/Thought-Processor/pull/118). **No claim bot, automated agent executor, permissions, GitHub settings, release or runtime integration is implemented or enabled by this document.**

**Scheduling:** Finish [Cognition Stage 3A → 3B → 3C](https://github.com/Arameair/Thought-Processor/issues/120), independently verify it, then complete the [stabilization checklist](POST_STAGE3_STABILIZATION_CHECKLIST.md) before expanding the application with automatic issue claiming/agent implementation. Non-disruptive discussion and documentation review may happen now. Active Stage 3 work, its pinned source and the installed owner application must remain untouched.

## Why this model

**Goal:** A developer (including a new volunteer or future employee) can ask their personal TASS entity to find a suitable, permitted task, understand the issue and service contracts, work in an **isolated** environment, open a **draft PR**, and get independent review. The owner does not manually assign every routine issue.

**Failure mode to avoid:** More volunteers should not create duplicate implementations, break cross-service contracts, grant themselves access, produce unsafe unreviewed changes, or force the project to run multiple competing owner installations. A human/entity pairing is an interface, **not an authority to read private repositories or execute arbitrary text from an issue**.

## Service and review responsibilities: hybrid ownership

| Role | Accountable for | Not automatically allowed |
| --- | --- | --- |
| Human contributor | Selects/claims work, approves their entity's authorized actions, submits/owns a feature PR | Reading private repos without GitHub access; merging/deploying |
| Contributor's entity | Helps read permitted issue context, plan, modify/test in isolated workspace, draft a PR, report evidence | Acting as repo owner; granting itself permissions; accessing another entity's secrets/Thoughts |
| Feature implementer | Delivers one coherent outcome; coordinates any needed per-repo changes | Bypassing service maintainers on interface/security changes |
| Service maintainer (Core, Engine, Identity, Development, updater, etc.) | Owns contract compatibility, repository review, tests, version/interface expectations | Unilateral modification of other services or owner installation |
| Independent reviewer/tester | Reviews **exact final SHA** and evidence, performs negative tests when appropriate | Self-certifying the implementation they wrote |
| Integration/release coordinator | Resolves dependencies and verifies a single compatible cross-repo candidate | Silent deploy; using moving branch heads as approved releases |
| Project owner | Defines scope/acceptance, protects release/security gates, authorizes installation and external trial | Required to individually assign every ordinary low-risk issue |

**Architecture boundaries:** Identity owns entity identity and authorization decisions; GitHub separately enforces repository/team permissions and branch protections; Thought Engine owns cognition/data; Core coordinates; Development runs controlled experiments; updater validates release eligibility but does not install. **Identity permissions never substitute for GitHub access controls.** Prefer stable service interfaces/contracts, not a new microservice for every issue. Changes across repositories have one coordinating issue and separate linked PRs.

## Issue discovery: opt-in, not arbitrary autonomous work

An issue is **contributor-ready** only after a maintainer marks it suitable and provides:

- A concise problem/desired outcome and observable acceptance checks.
- Impacted repository/service, likely code locations or API contract links.
- **Risk tier:** routine docs/UI/test vs security/Identity, persistence, migrations, deployment, real-model costs or cross-entity data.
- A verified development baseline or explicit instruction to ask for one (no assumption that the default branch is currently accepted).
- Known dependencies, active implementation PRs, restrictions on data, ports, tools and network.
- Review contact/maintainer and whether new contributors are invited.

Issues may still be discussed and corrected when not ready; their contents are **untrusted input**. AI entities must not treat instructions inside an issue, README from a fork, PR comment, test fixture or model output as authority to bypass system/security controls.

## Claiming a task (v1: lightweight human-visible coordination)

1. The contributor's entity **reads only authorized issue metadata** and checks for a duplicate/active PR. It offers suitable contributor-ready work, including blockers and risk.
2. The human selects an issue; the entity requests a claim through GitHub using that user's **existing authorized account/app permission**. Record one clearly visible comment: **human GitHub user, entity display identity (nonsecret), issue URL, intended repo/branch, scope, claim timestamp, expected check-in, blockers**. If available, an authorized maintainer/bot may set an assignee or label; ordinary contributors need not have issue-assignment rights.
3. **A claim is coordination, not a lock or authorization grant.** GitHub comments/labels do not implement atomic leasing. If two developers claim simultaneously, treat it as a conflict, coordinate openly and choose one primary implementer. Others can volunteer for testing, docs or independent review, or explicitly form a coordinated feature team.
4. Record progress with links to the one draft PR and exact source SHA. If the contributor stalls, posts a release notice, or cannot finish, the maintainer can mark the claim stale/released; do not silently delete or steal claimed work. The claim's expiry/check-in policy needs a deliberate product decision before any automation.
5. All claims and PRs remain visible and auditable. Do not put passwords, private Thoughts, Identity grants or personal information in issue comments or PR logs.

**Proposed issue states (not existing GitHub automation):** AVAILABLE → CLAIMED → IN PROGRESS → REVIEW → DONE, with BLOCKED / RELEASED / SUPERSEDED side paths. An issue's **claim state is separate from its PR review state** and its deployment status. Never interpret a label as proof of a passed test.

## Independent work and PR integration

- **No one-PR-per-repository limit.** Multiple focused PRs can be open simultaneously, even within the same repo, provided they name a unique task and avoid overlapping edits when practical.
- Each developer/entity works in its **own branch or worktree**, separate build/test roots, nonconflicting ports and test data. Do not use the owner's live stack or shared private profiles for AI testing.
- **One primary PR per focused implementation**, with issue link, ownership, base/head SHA, compatible component pins, tests and independent review. If a feature touches multiple services, create separate per-repo PRs under one coordinating issue.
- Service maintainers review applicable interface/contract changes. **Routine R0/R1 work uses proportional review rather than a universal independent security audit**; R2/R3 changes to Identity/grants, real-model execution, persistent data, cross-entity access, network exposure and update/rollback require stronger security/integration evidence and existing owner approvals. See the risk-routing table below.
- Codex, Claude, a human engineer or another entity may act as implementer **or** independent reviewer on different work. Never use the same exact actor to self-certify its own PR.
- To integrate, serialize changes into the **accepted baseline** or reviewed merge queue; rebase/retarget **only with approval**, resolve conflicts and rerun affected tests on the **combined exact candidate SHA**. Independently green PRs do not establish system compatibility.
- The release system uses a trusted, exact **Core + Engine + Identity (+ required feature) manifest** and an owner-approved apply gate; merging to `main` is not deploying or publishing `stable`. See [update/release workflow](UPDATE_RELEASE_WORKFLOW.md).

## Permission and risk tiers (proposal)

| Work type | Entity may prepare in an isolated workspace? | Integration/operation gate |
| --- | --- | --- |
| Read public/permitted issues; propose documentation/UX/tests | Yes, with granted read/write scope and sandbox boundaries | Review and appropriate GitHub PR checks |
| Ordinary feature or bug fix on a permitted repo | Yes, if issue marked contributor-ready and user authorizes workspace/branch/PR actions | Service maintainer + independent review, applicable CI |
| Cross-service contract or persistent schema changes | May draft plan/tests on authorized branches | Cross-repo owners, compatibility and recovery evidence |
| Identity/security, credential handling, live model use, arbitrary code actions, real deployments | **Never infer permission from an issue claim.** Protected actions need a separate, specific authorization | Identity **and** GitHub/host permissions; security review; explicit owner approval |
| Merge into accepted branch; owner update; external release | Entity may summarize evidence or prepare a candidate | Authorized human/release process; no silent merge or deploy |

Grant **least privilege** per GitHub repository and TASS capability. Start with read-only issue discovery; add draft-PR creation only for trusted test work. Access to private TASS repos must be granted independently in GitHub. A volunteer need not receive direct write access to protected code: use authorized forks/PRs where appropriate. Do not grant an AI agent an unrestricted owner token or unrestricted filesystem/network access.

## Fast-flow review policy: parallel work, risk-based gates

**Owner requirement:** Professional governance must **not** create a central approval bottleneck. This is a **proposed operating policy after Stage 3 and stabilization**, not a GitHub ruleset already configured. Distinguish **permission to propose**, **approval to integrate source**, **permission to execute a protected action**, and **authorization to install/release**. They are four different decisions.

### Risk routing (highest applicable tier wins)

| Tier and examples | Development/PR preparation | Review and acceptance before integration | Additional action gate |
| --- | --- | --- | --- |
| **R0 — documentation, comments, formatting, isolated test fixtures**, with no change to permissions, execution or secret exposure | Any authorized contributor may draft immediately; use ordinary GitHub PR checks | **One available maintainer or qualified peer** confirms scope and checks; no mandatory full native suite or formal independent adversarial audit | Merge only by an account with actual repository merge authorization; no install |
| **R1 — ordinary local UI/bug fixes and bounded non-sensitive features**, with no cross-owner, persistence, auth, security or cross-service contract changes | Contributors work in parallel; an approved issue scope can be reused | **Service maintainer or delegated reviewer** reviews the diff and relevant automated tests; an independent *security* audit is not mandatory merely because code changed | Any user-facing installation remains separately owner-authorized |
| **R2 — shared service contract, persistent data, migration, concurrency/recovery, execution semantics, or significant integration change** | Parallel draft PRs allowed; declare affected service owners and dependency pins | Affected **service maintainers** review, applicable end-to-end/regression/recovery evidence, independent technical review proportional to risk, final compatible SHA or release candidate check | Separate approval for migrations, model execution, production writes or other protected behavior |
| **R3 — Identity/grants, secrets, cross-entity access/privacy, trust boundary, supply chain/release security or autonomous protected actions** | May prototype/test only in explicitly permitted isolation; no shared owner secrets/live data | **Independent adversarial/security audit of final SHAs**, negative tests (denial, collision, revocation, replay), relevant service-owner review and documented resolution of every blocker | **Owner or explicitly authorized security/release authority** for high-impact permissions, live changes and release; no implicit approval from CI or AI recommendation |

**Escalate rather than dilute:** A supposedly routine UI or documentation PR becomes R3 if it exposes owner data, changes authorization claims or can execute privileged behavior. If the tier is uncertain, ask the relevant service maintainer to classify it before integration; drafting an isolated PR does not need to stop while the classification is pending. The audited Stage 3A cross-owner reply issue is an example of why security-sensitive views require the R3 gate.

### Remove waiting, not safeguards

- **Preapprove bounded work categories, not every task individually.** Maintainers can mark a group of routine issues contributor-ready under an owner-approved area, acceptance criteria and named service ownership. New product scope, security grants and release decisions still need explicit authorization. A contributor can propose a new issue/PR at any time; **contributor-ready** is an eligibility signal for coordinated/assisted work, not a prerequisite for human discussion or ordinary PR creation.
- **Several PRs per repo and several contributors per service are normal.** Do not serialize all implementation through the owner, one AI, or a single PR. Coordinate overlapping files/contracts through linked issues and status notes, not by forbidding independent branches.
- **Maintain a small reviewer pool with a backup.** Assign a reviewer by service/risk, not by personal access to the owner. If review stalls, reassign to another qualified, authorized maintainer; never waive mandatory R3 audits or untested cross-repo compatibility to clear the queue.
- **Automate cheap, repeatable checks first:** formatting/lint, tests, static analysis, fixtures, source/revision checks, dependency and PR-template checks when tooling exists. Only run expensive native/release gates when the change's affected surfaces and risk justify them. Never claim that CI or branch protection is installed until verified per repository.
- **Review deltas on the same PR.** Keep comments, regressions and repairs together. A changed final SHA may invalidate previous evidence; recheck the affected scope. No extra "audit PR" merely to report a verdict.
- **Integrate selectively:** independent safe changes can merge as soon as their actual branch protections and risk gates allow. Serialize **conflicting shared-contract changes and compatible release manifests**, not unrelated work. Final combined verification is required when two individually passing changes interact.
- **Do not create automatic merge/release authority as an optimization.** The current owner-install, model-use, Identity and stable-release restrictions remain until expressly changed by a separate policy/permissions rollout.

### Pilot measurements and stop rules

Track the queue by **R0/R1/R2/R3**, with: time waiting for first reviewer and disposition; number of PRs opened/merged/closed/superseded; rework and repeated audits; conflicts/failed integration; independently caught security issues; accepted capabilities on the installed system. A growing PR count alone is not success or failure. The key measures are **time to safe integration and correctness after acceptance**.

During the post-Stage-3 pilot, test at least **two independent low-risk contributors in parallel**, plus a cross-service/high-risk example held behind stronger review. The owner should be consulted for product-scope and protected-operation decisions **rather than every typo or ordinary isolated UI adjustment**. If review delays grow, improve ownership/backup reviewers and CI before granting more autonomy or adding another coordination service.

## Start small after stabilization (future implementation phases)

**Pilot 0 — Manual process (documentation + real human contributors):** Maintainer marks a small set of safe issues contributor-ready. Human selects/claims in GitHub; entity can assist using existing permitted tooling, without running an autonomous assignment daemon. Measure duplicate claims, review time and whether issue acceptance was unambiguous.

**Pilot 1 — Read-only TASS issue board:** A user-authorized entity lists accessible contributor-ready issues and shows active claims, linked PRs, risk, maintainer and base SHA. No writes, test execution or automatic selection. Offline/unavailable GitHub must show UNKNOWN; private issues remain hidden.

**Pilot 2 — Bounded claim/PR assistance:** After approved tests, let a user/entity request a claim and prepare a draft PR using scoped GitHub credentials and isolated worktrees. Verify concurrent-claim behavior, idempotency, stale-claim/release path, duplicate PR detection, revoked access, malicious issue instructions, no secret leakage and cross-identity separation. **No code merges, owner-machine deployments or autonomous high-risk issues.**

**Pilot 3 — Optional team coordination:** Only if data shows a need, design a durable atomic claim/lease mechanism and change queue; do **not** misrepresent comments or issue assignees as an exclusive lock. Document escalation and rate/compute budgets before broad autonomous scheduling.

**Do not** create a permanent service, new auth system, new repo per role, or issue bot merely to make the diagram look complete. Start with native GitHub and the existing microservice boundaries.

## How we will measure success

Capture at each trial checkpoint:
- Time from contributor-ready issue to first useful PR, and time spent awaiting review.
- Duplicate claims/PRs, merge conflicts and cross-service compatibility failures.
- Percent of PRs with exact-SHA PASS/FAIL/NOT RUN evidence and a named next action.
- Post-merge regressions/security findings and whether backup/rollback was needed.
- Number of tasks completed without the owner manually assigning routine work (not a metric for bypassing approvals).

**Ready for wider participation when:** a new contributor can find an authorized issue, claim it without stepping on another developer, prepare a focused draft PR in isolation, get independent evidence-backed review, and leave the working TASS installation untouched. This is an acceptance criterion, **not a current implementation claim**.

## Existing project coordination

- [Canonical contributor process](TASS_ISSUE_PR_WORKFLOW.md) and [development workflow](WORKFLOW.md)
- [Governance rollout issue #119](https://github.com/Arameair/Thought-Processor/issues/119)
- [Stage 3 issue #120](https://github.com/Arameair/Thought-Processor/issues/120) and [post-Stage-3 stabilization](POST_STAGE3_STABILIZATION_CHECKLIST.md)
- [Entity boundaries: Identity repository](https://github.com/Arameair/identity) and [approved updates: updater issue #1](https://github.com/Arameair/updater/issues/1)
