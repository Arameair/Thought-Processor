# Responsibilities, relationships and evidence
Design only; revised 24 October 2026. Human-directed, AI-assisted.

## Participants
| Participant | Owns | Does not imply |
| --- | --- | --- |
| Human operator | Intent, authorized choices, acceptance | GitHub sign-in alone grants every local operation |
| Identity | Durable identities, trusted registration, scoped approval and revocation | A detected endpoint is trusted |
| Thought Processor | Workspace, guided setup, non-authoritative proposals, permitted evidence | Authority to grant itself access or run inference |
| Independent feature, including Thought Engine | Own implementation, declared capabilities/requirements, effective configuration and records | Installation permits access to another identity's data |
| Runtime instance | Current execution and instance evidence associated with a durable feature identity | New permanent feature identity on every restart |
| Publisher access authority | Controlled feature delivery/hosted access under actual policy | Authority over local user data or reliable erasure of delivered source |

## Relationship model
The selected identity may be an entity or a feature identity. Each accessible feature is selectable; supported actions depend on actual access, compatibility, configuration and approval. Unsupported or unknown does not become permission.
A relationship records the participating identities and roles, intended purpose/capability, applicable entity, requirements, current Identity approval, effective owning-feature configuration and observed evidence.
This is a conceptual model, not a selected storage schema or API.

Shared feature use must retain separate applicable grants and data boundaries. No first-prototype multi-tenant security claim is implied.
Source → Engine → worker describes functional relationships; exact deployment/instance associations remain inspectable.

## Requirements
Features declare what they provide and need for the selected version/configuration. Processor guides choices among accessible candidates.
A declaration is a claim to validate, not a new authority. Missing/unsupported descriptions mean requirements unknown, not ready.
Required versus optional dependencies, compatibility, cycles, version changes and stale evidence need an explicit small contract before implementation.
Selecting a collaborator does not silently authorize its dependencies.

## Four stages
Proposed → Authorized → Applied → Observed.
Identity owns authorization. Each operating feature owns applied configuration and its records. Processor correlates actual results.
No assumption of an atomic cross-service transaction. A partial result preserves completed-step evidence and a safe next action; no blanket rollback or permission revocation affecting unrelated uses.
Where durable non-authoritative proposals live remains open.

## Multiple status dimensions
Access eligibility, installed state, compatibility, process state, authenticated peer identity, scoped authorization, dependency satisfaction, observed outcome and freshness are independent.
Available to add / Connected / Needs attention are summaries with drilldown, not replacements for those dimensions.
Unknown availability differs from a verified empty catalog.
Authenticated differs from a name/hash claimed by a candidate response.
Accepted, queued, processed and persisted each require their own receipt.
No recorded worker invocation evidence means Unknown, not zero.

## Critical boundaries
- Verified Identity handoff preserves operator/account/repository/working identity; direct launch offers repair/reconnect.
- Start/operational attach requires real Identity approval enforced at the runner/control boundary.
- Discovery metadata is narrowly read-only; no unapproved diagnostic access or control.
- Ingestion needs separate authenticated source and scoped grant; Engine enforces it.
- Processor does not become the required observation/inference relay.
- Browser close is not shutdown. Explicit stop touches only verified session-owned processes under the agreed control policy.
- Publisher delivery entitlement and Identity runtime permission stay separate.
- No fake user-facing telemetry or feature cards. Fixtures remain isolated automated-test evidence.
- Creating an identity is not creating software. Create a feature remains a guided proposal workflow until separately defined.

## Future comparison
Record exact versions, selected relationships, test conditions, permission scope and observed outcomes when comparing arrangements. Available to try is not proven useful. Do not assume recency, popularity or more features means better.
Comparison tooling and update/rollback workflows remain future design.

See [story](01_OWNER_JOURNEY.md) and [blueprint handoff](03_BLUEPRINT_HANDOFF.md).
