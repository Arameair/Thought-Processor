# First opening, ownership, and an approved Engine
This is a proposed user journey, not a report of working software. No example account, repository, result or service record should be seeded into the live product.

## 1. Identity opens for setup
You open Identity. It inspects its selected local profile without overwriting existing records.
If setup is incomplete, it explains what is missing. Setup and recovery remain accessible; normal operational readiness is not claimed.

The next action is Connect GitHub, or Recover existing installation.
Connecting an account does not yet create a repository or assign administrative ownership.

## 2. You establish who owns this installation
Identity obtains the actual account identity through the approved GitHub authentication mechanism. It displays the returned account and actual access, then asks whether to associate that account with administration of this installation.

**Proposal for review:** explicit confirmation establishes the first administrator only through a protected first-run enrollment flow. A first HTTP visitor or arbitrary localhost caller must not be able to claim administration.
The exact enrollment/authentication protection and recovery design are still required; this narrative is not a security protocol.
An existing installation never silently replaces its administrator because a different account signs in. Use stable account identifiers, not only a mutable display name, in the association.

## 3. You establish your own starting repository
Review the exact approved source contents, destination account/repository, visibility, origin credit and filesystem/network effects before creating anything.
Creation proceeds only once the source/distribution decisions are approved. If they are unresolved, say Repository setup blocked and why; do not invent starter contents.
This repository contains the actual approved starting software, not only configuration.
Runtime records, tokens and signing secrets stay outside source checkouts.

If creation succeeds remotely and local setup is interrupted, resume by reconciling the recorded repository identity and contents; do not blindly create a second repository.
Copying starter code creates a new installation identity; restoring an existing installation preserves its authorized durable identity. Never copy the original user's credentials.

## 4. You select what you are operating
Identity distinguishes the human administrator/operator, installation, selected entity, feature identity and running instance.
The working entity may be a companion or another supported entity. It is not inferred from a GitHub username.
If no entity is selected, show that fact and the next action.

Publisher access determines which protected features may be delivered or used at a publisher-controlled boundary. Local Identity grants determine what the installed feature may do with this user's system. Neither decision replaces the other.

## 5. Identity offers the Processor experience
Identity shows the exact approved Processor build and the services requested for the selected session.
You deliberately choose Start Experience. Launch consent and Identity authorization are recorded separately, even if a single understandable confirmation gathers both.

The approved runner starts or attaches the selected Processor under its actual contract. Processor revalidates the context: authority, operator, installation, working entity, account/repository evidence and freshness. A URL parameter or display name is not trusted proof.
Directly opening Processor offers diagnostics/reconnect, not an authority bypass.

## 6. Processor prepares the selected Engine — only with approval
Processor first asks Identity whether this authenticated operator/Processor may start or operationally attach the specific Engine in this installation/session.
The intent identifies the approved build/manifest, target service and action. Approval for attach does not silently grant start, stop, observation ingestion or other commands.

If Identity is absent, denies, returns invalid evidence or cannot establish the requested scope, the operation stays blocked. No Engine starts, no operational attachment is established, and existing independent processes remain unchanged.
A developer-trial label cannot relax this rule.

When approval is valid, the bounded launcher independently validates it and checks the approved target/build.
If the correct instance already runs, Processor attaches without spawning another or taking ownership of that process.
If the Engine is stopped, a start approval permits the runner to start exactly one selected instance. Process start, readiness, compatibility and authenticated peer identity each need their own evidence.
The page shows only actual results and a next safe action. It can open the original Engine interface only when that access is supported and permitted.

## 7. Running is not permission to observe
An Engine can be running with no permission to ingest data.
To test ingestion later, a separate authenticated source requests a scoped grant for the working entity and Engine. Engine validates that grant at intake and again before dispatching queued work under the agreed first integration policy.
No real source/grant contract means no enabled Send observation control.
Only real receipts distinguish accepted, queued, processed and persisted. No Thought is promised.

## 8. You close, return, or stop
Closing the browser does not itself stop Engine or revoke a grant.
Returning Processor verifies the current context again and does not automatically recreate the installation or Engine. Expired/missing attachment approval leaves control blocked; existing processes are not killed as a side effect.
The explicit Stop session-owned Engine action affects only the Engine and descendants provably started by this session, under the control policy. It does not stop Identity, an external Engine, the launcher or Processor itself. Broader Stop Experience remains a separate launcher responsibility, not this button.
Loss of start/attach permission is not automatically a kill instruction. Ingestion revocation is separately enforced by Engine. Already-transmitted data cannot be retroactively withdrawn.

## Review limits
This story does not select colors, a final layout, a legal license, a repository bootstrap mechanism or an authority protocol. Unknown states remain unknown. A diagnostic shell alone is not the completed first-opening experience.
