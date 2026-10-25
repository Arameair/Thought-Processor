> **DESIGN RECONCILIATION — later owner decisions (2026-10-25):** Read [Owner Design Alignment](OWNER_DESIGN_ALIGNMENT_2026-10-25.md) first. It supersedes conflicting **proposals** below: newly created identities begin without optional features or approved data sources; an installed feature can be reused by multiple identities but access to each source starts empty and requires explicit Identity-profile authorization; source features prepare common-format Sparks from selected information, Engine/worker relate Sparks into **Threads** then identity-owned Thoughts; Attention/Decay are optional processing features and never rewrite Spark evidence. Existing Engine `Spark → Trace → Thought` describes the old prototype, not adopted Thread semantics. This document remains useful historical research; unresolved first-launch UI, protocols and Memory details remain OPEN. No code/build/promotion authorized.

# Design discussion — Profiles, Catalog, Collections, and a dynamic feature ecosystem

**Date:** 2026-10-25  
**Status:** COLLABORATIVE DESIGN NOTES ONLY — no implementation assignment, API approval, feature grant, branch promotion, or claim of live integration. The owner explicitly paused building to work out the product design first.  
**Provenance:** Owner-led discussion with ChatGPT (GPT-6), informed by the existing Identity / Thought Processor / Thought Engine notes, Claude's experimental implementation reports, and a Grok-attributed architectural critique posted to GitHub by the owner.

## 1. Product direction the owner liked

The main interface should follow a calm **Netflix / Prime-style** browsing model:

- **Profiles:** TASS, a business, another companion, or another entity the user manages.
- **Catalog:** Thinking, Memory, Communication, Perception, and other **actually accessible** features.
- **Collections:** Thoughts, activity, and content belonging to the **selected identity**, supplied by the owning features.
- Return to the last-selected identity when the authorized installation can verify it; profile switching remains obvious.
- Technical service identities exist and may be inspected under system/advanced views, but they are not automatically ordinary companion profiles.

This is not a decorative gallery. A card must lead to verified data, actual feature capabilities, and meaningful interactions. Empty collections remain empty; unavailable features stay visibly unavailable. Diagnostics (flows/logs/metrics/Copy Context) are first-class but secondary to the experience.

**Long-term proposition:** A stable identity can acquire, replace, disable, reconnect, or remove modular capabilities without becoming a different identity. Relationships and authority survive component churn; feature-owned data is not automatically migrated or erased.

## 2. Example: TASS as a profile

TASS can have separate, optional capabilities: a Communication feature for Discord, Perception/log-reading for game events, Thought Engine for thinking, future Memory for approved retention/retrieval, and future Cognition for planning/evaluation.

The same design should fit a non-companion business identity and identities without Thought or Memory. **An identity does not automatically think** because the software supports a thinking feature.

The Thought Engine is a **feature**, not TASS itself, not the system's authority, and not the default for every identity. An inference worker is another independent participant, replaceable subject to capability and authorization.

## 3. Features can also produce observations

Owner emphasis: as Memory, Cognition, Communication, Perception, and other features are added, their activity and logs may provide **candidate inputs** to Spark → Trace → Thought. This is a major reason to preserve dynamic modularity.

Examples (hypotheses, not implemented integrations):

| Source feature | Example source material | Potential observation candidate |
| --- | --- | --- |
| WoW game log reader / Perception | New events in an approved game log | Repeated visits to one location |
| Communication | A chat transcript or structured interaction receipt | A recurring unresolved user question |
| Memory | An authorized retrieval/contradiction result | Recalled evidence conflicts with an existing interpretation |
| Cognition | A planning outcome or evaluation record | An action failed despite a previously assumed condition |
| System monitoring | A service log or changed health state | One feature repeatedly loses its approved connection |

**Do not equate all logs with Sparks.** Operational logs are evidence for debugging; domain observations are bounded, validated events considered by Thought Engine. An approved Log Reader or source adapter may transform selected log entries into observation candidates; an authorized feature may also publish structured domain events directly. The Engine decides what to examine, rate, link, retain, or ignore. A Thought is not the same as a Memory.

A source may support **live observation** and/or **bounded historical retrieval**, allowing an existing Thought to investigate older permitted evidence. Research whether retrieval-first and ongoing-live models should coexist. Neither arbitrary disk read nor universal feature-wide access should be implied.

## 4. Logical system boundaries

**Identity** is authoritative for stable identity/service/feature identities, owner/account relationships, publisher feature-access decisions, approved capability relationships and scoped runtime grants. Distinguish feature visibility, entitlement, installed, connected, authenticated, authorized, enabled, ready, processing and persisted. GitHub repo visibility is only a **temporary experimental install rule** in a current Claude branch, not an approved universal entitlement system.

**Thought Processor** is the human/admin experience. It browses the selected identity's catalog and collections, proposes/reviews relationships, requests changes, inspects real status, and correlates evidence. It does **not** own Spark/Trace/Thought content, proxy all hot-path data, mint grants, or execute arbitrary installation/start commands.

**Thought Engine** owns observation admission/queue, Spark/Trace/Thought records, processing/worker calls, thought development, and effective domain settings, subject to verified Identity authorization. Future Memory owns its own approved retention/retrieval records. Observation sources and other features own their domain data and application-specific logs.

Canonical entity IDs must come from Identity; no separate Engine-local identities as competing long-term authorities. Installing, attaching, starting, ingesting, reading history, and stopping are different scopes and should not inherit approval from one another.

### Proposed data path (logical, not an API contract)

```text
     Identity (entity IDs, service registry, grants, relationships)
          ^             ^                   ^
          | authorization and registration   |
          |             |                   |
Processor (profiles/catalog/collections, honest operations and receipts)
          | user commands, authenticated inspection
          v
  Thought Engine <---- permitted observations ---- Source features
          |                                   (game logs, Memory, Cognition, etc.)
          +---- scoped model work ----> approved AI workers
          +---- bounded evidence retrieval ----> authorized source / Memory feature
          +---- Thought records / receipts ----> Processor's display
```

Processor is NOT required to stay running for independently authorized source → Engine or Engine → worker exchanges.

## 5. Dynamic add / connect / disable / remove — proposed experience

For a selected identity, Processor should eventually be able to:

1. **Find / inspect** available features and their declared capabilities/dependencies with trustworthy publisher and version information.
2. **Propose** a relationship: e.g. “Allow TASS's Memory retrieval events to be considered by Thought Engine,” showing exact source, destination, action, data class, consequences and retention.
3. **Request approval** through Identity; receiver verifies Identity-issued scope. A feature's self-declared name or port is not proof of identity.
4. **Connect and confirm** that participating services applied the approved configuration; show failures and the next valid action.
5. **Pause / revoke / detach / uninstall** distinctly, with explicit ownership of processes and data. Revoking future event access should not silently erase historic Thoughts or source files; uninstalling is not authorization to delete another feature's records.
6. **Replace or compare** compatible feature versions without silently changing the selected identity, overwriting the last accepted build, or treating an upgrade as automatically better.

User-facing controls should distinguish `proposed → Identity-approved → receiving-feature-applied → observed outcome`. Retain rollback evidence. Dynamic **code generation or installation** is a separate, higher-risk owner-approved workflow, never implicit when a model proposes a feature.

## 6. Prevent feature-to-feature feedback loops

Memory may persist a Thought; the persistence event may reach Thought Engine, which could produce another Thought, then Memory could persist it again. Cognition and Communication can cause similar loops.

Design candidates for review:

- Stable event ID, authenticated source-service ID, target entity ID, origin time, ingestion time, source record reference, schema/version and causal parent IDs.
- Event type and `generated_by` / derivation lineage to distinguish independent observations from re-statements of prior AI output.
- Deduplication and bounded recursion/cause depth; no implicit “new evidence” credit for a derived echo of the same fact.
- Per-identity/topic limits on model calls, event volume, queue size, context, concurrent investigations and elapsed time.
- Preserve contradiction/negative evidence and freshness; do not let one early model hypothesis recursively validate itself.
- Disconnection, revocation or source outage must have explicit treatment for already queued work, historical records and in-flight calls.
- Avoid logging sensitive raw source payloads in global diagnostics; audit metadata can exist without copying full content to GitHub.

## 7. Thought lifecycle research — not a locked algorithm

A Thought could contain: an emerging question or hypothesis, source references, supporting and contradictory Traces, processing status, notes, next-evidence direction, and an explicit reason to continue / wait / sleep / resolve. The current Engine's three-related-Trace threshold is a **prototype heuristic**, not accepted product semantics.

Explore: what sparks deserve attention; whether retrieval of older logs helps a developing thought; which worker capability performs extraction vs reasoning; how to prevent unproductive rumination; when a Thought should be discussed with the person; how Thought and future Memory differ in storage and intent.

## 8. Open design questions — discuss before code

- Is a feature's observation stream opt-in per event class, source, entity, or all three?
- Which source-generated events become candidate observations; who chooses filtering and backpressure policies?
- Can a Thought request historical evidence, and how are its requests checked and bounded?
- What is the minimum versioned capability declaration for a feature (provide/require/permissions/schema/version)?
- How should Processor show thought state, feature state and relationship state without collapsing them into one colored badge?
- Who persists requested connection plans versus applied feature configurations?
- On feature removal, what happens to its identity registration, stored records, Thought citations, permissions and restore path?
- What objective evidence demonstrates that log-driven thinking is more useful than simple event summarization?
- Which identity attributes (purpose, goals, values, relationships) legitimately influence ranking/processing, without importing private Memory by default?
- How is an ongoing thought expressed to the user, rather than interrupting for every event?

## 9. Existing experiments and outside review to preserve

- Grok-attributed critical review warns against attractive UI masking missing secure grants and real Engine receipts: [Processor issue #7 comment](https://github.com/Arameair/Thought-Processor/issues/7#issuecomment-6380855246). Attribution is from the GitHub comment supplied by the owner, not a direct Grok connection.
- Claude's Identity ↔ Processor pairing / Engine installation prototype: [installation report](https://github.com/Arameair/identity/blob/experiment/engine-access-pairing-20261026/docs/ENGINE_INSTALL_DEMO_20261026.md). Fixture-tested GitHub access, **not** a production entitlement system or fully launched Engine.
- Claude's Processor ↔ Engine observe/thought loop: [Engine report](https://github.com/Arameair/Thought-engine/blob/experiment/engine-processor-loop-20261026/docs/ENGINE_PROCESSOR_LOOP_20261026.md). Real local processes + labelled TEST FIXTURE worker, **not** live Chat-Jimmy proof.
- Engine-side Identity-grant experiment: [authorization doc](https://github.com/Arameair/Thought-engine/blob/experiment/identity-grant-enforcement-20261024/docs/IDENTITY_AUTHORIZATION_EXPERIMENT_2026-10-24.md); the separate Identity app has not supplied the matching verified grant service.
- Older [Netflix-style identity workspace](https://github.com/Arameair/Thought-Controller/blob/experiment/multi-identity-controller-design-20261023/docs/IDENTITY_WORKSPACE_START_HERE_2026-10-23.md) and [owner journey](https://github.com/Arameair/Thought-Processor/blob/experiment/processor-story-prototype-20261024/docs/story/01_OWNER_JOURNEY.md) remain relevant references.

These experimental paths are **diverged**. Do not merge them indiscriminately or mark them end-to-end authorized; they are research inputs for one future shared design.

## 10. For other AI reviewers

Please respond with **claim → evidence/assumption → risk → alternative → smallest falsifying test**. Challenge modularity where it adds needless complexity; challenge privacy or security where untrusted log text reaches workers; distinguish observed current code from hypothetical architecture. Include a contrary example (e.g. Memory produces many derivative events and no useful new Thoughts).

**No builds until the owner explicitly decides the necessary design contracts.** Start future experiments from owner-approved Alpha when available, keep stable/main/alpha unchanged without explicit promotion, and retain contributor/AI provenance in review notes.

Related sibling design notes:
- [Identity — dynamic relationships](https://github.com/Arameair/identity/blob/experiment/dynamic-feature-observation-design-20261025/docs/DYNAMIC_FEATURE_RELATIONSHIPS_DISCUSSION_2026-10-25.md)
- [Thought Engine — observation and staged thoughts](https://github.com/Arameair/Thought-engine/blob/experiment/dynamic-feature-observation-design-20261025/docs/FEATURE_OBSERVATIONS_AND_THOUGHT_STAGING_DISCUSSION_2026-10-25.md)
