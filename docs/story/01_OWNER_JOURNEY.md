# Thought Processor — a workspace for relationships between identities
Revised 24 October 2026. Human-directed, developed in collaboration with AI.
**Design story only. No software, service state, account, grant or test result is claimed by this narrative.**

> Thought Processor controls the operational setup of thinking. Thought Engine controls the actual act of thinking.

Processor's wider purpose is to help the operator select accessible features, establish approved relationships between identities, and understand whether those relationships work. Thinking is one use of that model, not the required purpose of every workspace.

## Scene 1 — Identity hands you a working context
You have opened Identity, verified your GitHub account and installation/repository through the approved setup, and selected the identity you want to work with. Identity opens the approved Thought Processor experience.

Processor verifies that handoff rather than trusting a URL parameter. You see the actual operator/account, repository, Identity authority and selected working identity. It does not ask you to repeat setup or select the same identity again.
If anything cannot be verified, it explains the missing evidence and provides setup/reconnect/repair. Directly opening Processor takes this same verification route; it cannot bypass Identity.

The human operator, working entity, feature identity, installation and runtime instance are different records. TASS is an optional example of a working entity; it is not a compulsory home screen or seeded user record.

## Scene 2 — What features can this identity use?
You arrive in the selected identity's workspace. Processor shows every feature that identity has access to, with supported actions and current evidence. It does not lead only with Thought Engine, require a companion, or hide other accessible capabilities because they are unrelated to thinking.

You can distinguish:
- Available to add: access established, relationship not yet configured.
- Connected: relationship configured; actual runtime/permission status shown separately.
- Needs attention: a specific missing requirement, failure or stale check needs action.

Those summaries do not collapse installed, running, authenticated, authorized and ready into a single state. An accessible feature may not be installed or compatible with this host. Its details explain that limitation instead of silently removing it from view.

An empty successful availability check offers Find features and Create a feature. A failed or unavailable check says Availability unknown and offers repair/retry; it does not pretend no features exist.
Find features searches supported sources and checks actual access. Publisher-controlled delivery access remains separate from local Identity operation approval.
Create a feature begins a definition/registration proposal. Whether it starts from a plain-language idea or an existing application remains open. It does not instantly generate, run or authorize software. A new identity record alone is not a working feature.

## Scene 3 — Every feature has an identity
You select a feature to learn what it provides, what it needs, who it works with and what is currently verified.

Thought Engine itself has a feature identity. Its executable version, installation and currently running instance are separately identified. Restarting a process must not manufacture a new permanent feature identity.
An AI worker and observation source each have their own identities too.

Relationships describe roles, not ownership transfers:
- An entity uses Thought Engine for thought processing.
- Thought Engine uses an AI worker for inference.
- An observation source supplies authorized observations to an Engine for an entity.

Each relationship answers Who? For what purpose? What is required? What did Identity approve? What is actually happening?
Neither a dependency declaration nor publisher access grants permission to read the entity's data.

## Scene 4 — Add a feature without guessing its requirements
You choose Add on an accessible feature.
Processor reads the selected feature/version's declared capabilities and requirements through a supported, verifiable description. It does not hardcode every Engine as requiring the same worker or invent a dependency from its name.

The guided setup shows which requirements are satisfied, which are missing, and which are unknown.
For each missing requirement it offers all accessible candidates that declare the relevant capability, with compatibility and limitations. It must not silently choose, install, start or authorize another feature.
If no candidate fits, explain the missing capability and offer Find features or a proposal to create one. Preserve unfinished setup where supported; it remains a non-authoritative plan.

Requirements may have requirements of their own. Do not expand forever or call a circular chain ready. Unsupported declarations, contradictory requirements or cycles are explicit blockers. The minimal declaration/compatibility contract still needs design.

If the selected Thought Engine declares an AI worker requirement, you choose an accessible compatible worker. If it does not, Processor must not force that requirement merely to follow this example.

## Scene 5 — Review the complete proposed relationships
Before applying anything, Processor presents the selected feature, its required collaborators, the purpose of each connection, affected identities, data access, intended start/attach actions and anything unresolved.
You can revise the choices or cancel.

The useful stages are:
1. Proposed: the combination you want.
2. Authorized: Identity permits the exact operation.
3. Applied: the responsible feature accepted its settings.
4. Observed: actual evidence shows what worked.

These can disagree. Identity approval with an offline destination is not a completed connection. Partial setup must identify completed and unfinished steps; no global success toast.
Processor keeps only non-authoritative proposals/UI state. Identity retains grants; features retain effective configuration and data. Durable plan storage remains an explicit design question.

## Scene 6 — Identity approves before Engine start or attachment
For our first bounded integration, you choose Thought Engine as one accessible feature. This is a test target, not a required default for every identity.

Processor requests Identity approval for starting or operationally attaching the selected Engine. The real runner/control boundary validates the exact actor, action and target/build under the agreed contract.
If Identity denies, is unavailable or cannot provide verifiable approval, start/attach stays blocked. There is no engineering-mode bypass. Existing independently running services are left unchanged.

With actual approval, the runner starts exactly one selected instance or attaches to the matching existing instance. Attach permission does not imply start permission; neither implies observation ingestion or arbitrary control.
The workspace shows real started-versus-external ownership, process state, peer trust, compatibility, readiness and check freshness separately.

You can inspect the genuine application's interface when supported and permitted, with Open original app when embedding is unavailable. Processor does not redraw a pretend Engine UI.

## Scene 7 — Connect an observation source and test a real path
To feed observations, select an accessible source with its own authenticated identity, the target Engine and the working entity.
Request a separate scoped ingestion permission from Identity. The owning source/Engine apply their supported settings and return evidence.
A start approval, publisher entitlement or local Engine switch cannot stand in for this permission.

A guided test identifies its actual sender and purpose. A manual test input is not live screen/game telemetry. A dedicated test source must not impersonate a production source by changing a text field.
The source sends directly to Engine. Engine enforces authorization at intake and before queued dispatch under the agreed initial integration policy. Processor observes permitted receipts, not the production data path.

Results distinguish authenticated sender, Identity decision, rejected/accepted, queued, worker invocation, processing outcome and persistence. Missing evidence stays unknown.
One accepted observation need not form a Thought. No real grant/source/worker means the corresponding test is blocked; no fixture output substitutes for a live outcome.

## Scene 8 — Find the broken relationship
When something fails, Processor identifies the affected relationship and first failed or unknown step.
- Revoked ingestion permission: Engine refuses new protected work under its actual enforced policy.
- Authority unavailable: verification failed; that is different from an explicit denial.
- Worker unavailable: runtime dependency failed; do not relabel it as permission denial.
- Uncertain response: inspect the original operation ID; do not blindly resend.
- Feature requirements changed: show which approved arrangement no longer satisfies the selected version.

Details reveal Flow, Logs and Metrics with provenance and timestamps. Copy Context is bounded and allowlisted, excluding secrets and raw private content by default.
The operator should understand what stopped, why, what still works and the next safe action without opening every log.

## Scene 9 — Explore combinations without declaring winners
You may later try a different accessible feature/version or collaborator. Identity approval and compatibility checks still apply.
Availability to try is distinct from evidence of usefulness. A new version is not automatically an improvement.

A future comparison should preserve the exact selected versions, requirements, permissions, inputs/test conditions and observed outcomes. Compare usefulness for the chosen task, failures, latency and resource needs only where actually measured. Do not invent a universal score or compare a completed real run with a fixture as equivalent evidence.
Keep the prior working arrangement recoverable; changing dependencies must not silently replace it. The comparison/update mechanism is future design, not a first-build mandate.

## Scene 10 — Close and return
Independent approved services communicate directly and keep their own data when Processor closes.
On return, Processor verifies the current context, re-reads service state and marks stale evidence. It does not recreate identities or automatically replace the operator's arrangement.
Explicit Stop session-owned Engine affects only verified owned Engine processes under the defined approval/cleanup policy; attached external services remain independent. Permission loss is not an assumed kill command.

The same experience must work for another identity with a utility feature and no Engine. No Brain offline warning appears unless that identity's chosen setup actually requires an Engine.

## What this story now commits to
The owner-supported direction is accessible features, feature identities, explicit relationships, guided requirements and Identity-approved operations.
The final layout, feature declaration schema, publisher/search sources, Create a feature workflow, durable proposal storage and comparison design remain open.
This revision is documentation, not build authorization or acceptance.

Continue: [responsibilities](02_RESPONSIBILITIES_AND_STATES.md), [blueprint handoff](03_BLUEPRINT_HANDOFF.md), [critical review](04_REVIEW_AND_ALTERNATIVES.md).
The [Identity-approved build boundary](https://github.com/Arameair/Thought-Processor/blob/experiment/identity-approved-build-package-20261024/prototype/START_HERE.md) remains mandatory.
