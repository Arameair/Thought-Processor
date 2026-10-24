# Blueprint handoff — accessible features and guided relationships
Design only; not implementation authorization. Revised 24 October 2026.

## Main journey
Identity supplies verified working context → Processor shows all accessible features and current relationships → choose a feature → resolve declared requirements using accessible candidates → review the complete proposal → request scoped Identity approvals → owning services apply → observe results → diagnose or retain the arrangement.

A verified Identity-selected context opens directly. System overview remains a supporting route, not a compulsory second selection screen. Direct launch requires verification/reconnect. Exact layout and navigation still need review.

## What each step must explain
| Step | User question | Evidence or blocker |
| --- | --- | --- |
| Handoff | Who am I operating as and for? | Verified Identity context and freshness; no URL-parameter trust |
| Feature list | What can this identity use? | Access-controlled catalog and availability evidence; unknown differs from empty |
| Feature details | What does it provide and need? | Selected version's supported capability/requirement description |
| Resolve requirements | Which accessible features can satisfy this? | Candidates, compatibility, required/optional state; missing/cyclic/unknown blockers |
| Review | Who will connect to whom, for what purpose and data? | Full proposal including dependency and lifecycle effects |
| Approve | What did Identity permit? | Real scoped decisions, separately for start/attach/control/ingestion |
| Apply | What actually changed? | Owner-feature receipts/readback; explicit partial outcomes |
| Observe | Does this relationship work? | Actual service/test evidence, not configuration alone |
| Diagnose | Which relationship failed? | Correlated permitted receipts, stale/unknown gaps and next safe action |
| Compare later | Is this combination better for this task? | Comparable actual conditions/outcomes, not a fabricated winner |

## Empty states
Verified no accessible features: Find features and Create a feature.
Availability check failed: explain Unknown and recovery; do not claim empty.
Create a feature starts a proposal; plain-language creation versus existing-app registration is undecided. No automatic code generation or execution.

## Decisions required before a buildable feature-setup contract
- Minimal capability/requirement description, versioning, trusted origin, required/optional semantics and compatibility rules.
- Which actual sources Find features searches and who proves delivery access.
- How incomplete plans persist without becoming a competing permission/configuration authority.
- Safe handling of cycles, version changes, multiple candidates and partial application.
- Revalidation when context/feature selection changes; no grants carried to a different entity or target.
- Real Identity startup/attachment contract and runner enforcement.
- Lifecycle ownership, explicit stop and original UI access permissions.
- Create a feature entry paths and review process.
- Future comparison, preservation of known working arrangements and controlled updates.

## Review scenarios (design acceptance, all NOT RUN)
1. No features after a successful access check: user can find or propose one.
2. Access service unavailable: user sees unknown rather than empty.
3. Several accessible features including a non-thinking utility: all selectable, no compulsory Engine.
4. Choose a feature with missing requirements: user selects accessible compatible collaborators; no hidden additions.
5. No matching dependency, unsupported declaration or cycle: truthful blocker, no fake ready or endless expansion.
6. Change the selected identity midway: revalidate access/proposal; no accidental cross-identity grant.
7. Identity absent/denies/wrong-scope approval: no start or operational attach; direct backend requests also refused.
8. Grant allowed but configuration fails: show Authorized / Not applied with exact failed step.
9. Engine running but no ingest grant: no observation processing implied.
10. New compatible feature: useful description/requirements without a custom card rewrite.
11. Two candidate versions: available-to-try separated from real comparable evidence.
12. Close/reopen: durable identities preserved; effective config read from owning features; stale states marked.

These scenarios guide future implementation instructions; they are not added executable scope merely by editing this story. Use the [Identity-approved build package](https://github.com/Arameair/Thought-Processor/blob/experiment/identity-approved-build-package-20261024/prototype/START_HERE.md) for the authority boundary and explicit build-review gates. Existing branch build briefs predate this revision and require reconciliation.

No fake live data, automatic feature generation, marketplace build, broad dependency installer, comparison platform or Alpha promotion is authorized here.

Read [story](01_OWNER_JOURNEY.md), [responsibilities](02_RESPONSIBILITIES_AND_STATES.md), [review](04_REVIEW_AND_ALTERNATIVES.md).
