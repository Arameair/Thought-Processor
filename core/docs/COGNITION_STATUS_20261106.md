# Cognition — current status and authority

**Reviewed:** November 6, 2026. Cognition is **part of Thought-Processor's Go Core**, not a separately deployed or independently authorizing service. Do not use the visual-only Cognition prototype as the operational data source.

## Current implementation

- **Engine** owns persistent Thoughts/Traces, cycles, ordered append-only journal entries and actual pulse state. [Engine PR #15](https://github.com/Arameair/Thought-engine/pull/15) provides an authenticated, bounded read-only cycle journal projection.
- **Core** builds authorized, bounded Cognition view models and sanitized export. [Core #82](https://github.com/Arameair/Thought-Processor/issues/82), [PR #86](https://github.com/Arameair/Thought-Processor/pull/86), [repair PR #87](https://github.com/Arameair/Thought-Processor/pull/87), integrated as candidate [PR #88](https://github.com/Arameair/Thought-Processor/pull/88). Browser replay must not call the model, start jobs, approve lessons or modify graph records.
- **Identity/Core integration candidate** requires an explicit owner-reviewed canonical Identity-to-Engine link; the owner-machine link has not yet been accepted. **Engine's internal name-based routing is still open**, [Thought-engine #17](https://github.com/Arameair/Thought-engine/issues/17); a Cognition link does not fix it.
- **Development** receives only unexecuted adjustment proposals. A displayed graph difference is not evidence of changed real agent behavior.

## What the tests demonstrate — and what they do not

[Owner Cognition runbook](https://github.com/Arameair/Thought-Processor/blob/experiment/claude-cognition-repair-junction-20261105/core/docs/OWNER_ACCEPTANCE_COGNITION_JUNCTION_20261106.md) records passing Go, browser, and 11/11 real-binary UI cases **with a fixture model and fake GitHub**, not real owner hardware or live Nemotron. PR #88 adds further fixture/integrated-candidate tests; it remains draft, unmerged and not owner-deployed.

**Known gaps:** historical cycle-era Identity decision is sometimes NOT RECORDED; the Engine may have name-resolved entities; `NEED lesson-metadata` spontaneous real Nemotron output is untested; a junction data receipt does not prove it changed a later decision or skill execution.

## Acceptance criteria

Owner reviews safe backup/update first [#79](https://github.com/Arameair/Thought-Processor/issues/79), reads installed full SHAs, then tests on the pinned real stack: explicit link, one real Thought or explicitly unlinked cycle, actual event timestamps and IDs, PENDING/missing evidence, immutable step/playback, safe log and export, separate Core Sync vs Engine Pulse status, no side effects from browsing, no cross-entity disclosure. Finally correlate one Identity-authorized read-only skill's **actual** executor receipt to the same cycle and evaluate its outcome [#89](https://github.com/Arameair/Thought-Processor/issues/89).

**Evidence is not completion:** use APP-REPORTED versus INDEPENDENTLY VERIFIED and NOT RECORDED/UNSCORABLE honestly. No change to the owner installation is authorized by this document.
