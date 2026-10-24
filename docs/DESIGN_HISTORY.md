# Design history and provenance

This page is a **navigation map**, not a claim that any experiment has been approved or merged. Design snapshots live on their original experimental branches and remain available for comparison.

## Evolution of the idea

| Stage | What changed | Original record |
|---|---|---|
| 1. Feature foundation | Thought Processing treated as one separately enabled capability; Identity owns grants; logs from opt-in sources become evidence | [Foundation / control-point plan](https://github.com/Arameair/Thought-Processor/tree/experiment/processor-first-control-point-20261023) |
| 2. Real control point | Discovery and diagnostics, with Engine (not Processor) owning the timing and logic of AI-worker heartbeats; Engine can make direct model calls | [Routing and heartbeat](https://github.com/Arameair/Thought-Processor/blob/experiment/discovery-routing-devex-notes-20261023/docs/discussions/2026-10-23-routing-heartbeat.md) |
| 3. Operator/DevEx | Compare overview-first, trace-first and split-view for detecting real broken stages | [UI/DevEx discussion](https://github.com/Arameair/Thought-Processor/blob/experiment/discovery-routing-devex-notes-20261023/docs/discussions/2026-10-23-ui-devex.md) |
| 4. Dynamic real app cards | Netflix-style independently discovered **application** cards; original Identity app UI, not an Identity avatar per tile | [Dynamic cards](https://github.com/Arameair/Thought-Processor/blob/experiment/dynamic-feature-cards-notes-20261023/docs/discussions/2026-10-23-dynamic-project-cards.md) |
| 5. UI direction selected | **Option C**: lightweight status tiles, just one active original UI preview and collapsible inspector; Option D reserved for later research | [Option C decision](https://github.com/Arameair/Thought-Processor/blob/experiment/lightweight-live-cards-option-c-20261023/docs/discussions/2026-10-23-option-c-lightweight-status-cards.md) |
| 6. Transparent authorship | Human owner directs the project; AI contributions and verification limits are disclosed | [AI collaboration document](AI_COLLABORATION.md) |

**All six stages are design documentation.** The current consolidated docs describe the latest direction **for review**; existing Alpha/Stable/Main software statuses are unchanged.

## Separate repos are not interchangeable

| Repository | Relationship |
|---|---|
| [Comp-idea](https://github.com/Arameair/Comp-idea) | Broader companion and dynamic-coding vision, beyond Processor's initial scope |
| [Project-controller](https://github.com/Arameair/Project-controller) | Development launch/stop, branch comparison, context/log capture |
| [Identity](https://github.com/Arameair/identity) | Real identity application and future permissions authority; its UI is one potential Processor card |
| [Thought](https://github.com/Arameair/Thought) | Separate simple deterministic thought-processing prototype |
| [Thought-engine](https://github.com/Arameair/Thought-engine) | Separate model-assisted prototype; still needs canonical owner choice |
| [Thought-Controller](https://github.com/Arameair/Thought-Controller) | Earlier dashboard experiment and potential design/UX source; not automatically renamed |
| [chat-jimmy](https://github.com/Arameair/chat-jimmy) | Current independent fast-worker prototype, not mandatory forever |

## Avoid historical confusion

- An *experimental document* is not a merged requirement, tested binary or live connection.
- A real worker may have been tested in its own repository; that does not prove the Processor integrated it.
- Alpha and Stable in **this** repository are still documentation-only.
- "Heartbeat" in later discussions means **Engine → fast worker API request**, not a Processor status ping or a timer owned by Processor.
- "Identity" in the cards discussion means the **running Identity app**, not a specific companion persona.

When a new experiment changes an owner decision, **append the rationale** and update [DECISIONS](DECISIONS.md) and [STATUS](STATUS.md) in the candidate branch. Don't erase the original discussion.
