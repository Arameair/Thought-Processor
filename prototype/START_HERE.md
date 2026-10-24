# Identity-approved first experience — review package
Prepared 2026-10-24. Documentation only. Human-directed, developed with AI assistance.

**Approval status: build authorization has NOT been given by this package.** The owner requested story and instructions preparation. Do not begin implementation until the owner explicitly assigns the build.

Read in order:
1. [Revised story](STORY.md)
2. [Build instructions](BUILD_INSTRUCTIONS.md)
3. [Acceptance tests and reporting](ACCEPTANCE.md)

## Decision this package preserves
Identity approves → Processor starts or attaches the selected Engine → Engine reports actual readiness.
Authorization to ingest observations is a separate decision.

This package replaces the conflicting standalone start/attach allowance in the earlier [First Spark build assignment](https://github.com/Arameair/Thought-Processor/blob/3a0a9039a6b5c7a87c9c560d66e0816c496aa2cb/prototype/START_CLAUDE_HERE.md) and tightens the [Processor journey](https://github.com/Arameair/Thought-Processor/blob/a91ea4e674302358a60fdd69274ce2b6dd4362c6/docs/story/01_OWNER_JOURNEY.md). These links are provenance, not alternate instructions permitting a bypass.

If an older repository document conflicts with this authority boundary, report the conflict; do not silently choose the more permissive text.

## Prepared handoff (use only after owner authorizes)
Read this package. Inspect current Alpha and actual Identity/Engine interfaces; create a new implementation experiment from current Alpha. First deliver the dependency inventory and a truthful diagnostic shell. Start/attach/control must remain blocked until real Identity approval and the receiving control boundary enforce it. Do not implement another repository without a separate assignment. Return actual evidence and blockers; no fake authority, automatic inference, promotion or expanded scope.

## Unsettled decisions
Exact authority wire contract, safe enrollment/admin recovery, starter contents/license/copy/update method, publisher entitlement service, GitHub outage policy and launcher packaging are not selected here.
