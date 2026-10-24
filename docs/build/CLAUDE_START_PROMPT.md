> **Story revision — 24 October 2026:** Read [the revised owner journey](../story/01_OWNER_JOURNEY.md) and [blueprint handoff](../story/03_BLUEPRINT_HANDOFF.md) first. This build document predates the accessible-feature/identity-relationship direction and is **historical, not a current executable assignment** until reconciled. In particular, any earlier optional-GitHub or unapproved attach/control allowance is superseded by the [Identity-approved build package](https://github.com/Arameair/Thought-Processor/blob/experiment/identity-approved-build-package-20261024/prototype/START_HERE.md). Creating/searching features, broad dependency installation and version comparisons are not authorized implementation scope. The owner has authorized story edits only in this pass.

# Copy/paste handoff for Claude — build the Thought Processor prototype

**Instructions intended to be supplied directly to Claude. All linked source material is on the story branch, not the implementation base branch.**

---

I want you to **build the next real Thought Processor prototype**, using the story-first experience we developed. This is an **implementation assignment**, not another round of conceptual documentation.

**Repository:** https://github.com/Arameair/Thought-Processor

**Story and full build package (READ FIRST):**
- Story prototype index: https://github.com/Arameair/Thought-Processor/tree/experiment/processor-story-prototype-20261024
- Actual owner journey: https://github.com/Arameair/Thought-Processor/blob/experiment/processor-story-prototype-20261024/docs/story/01_OWNER_JOURNEY.md
- Full implementation brief: https://github.com/Arameair/Thought-Processor/blob/experiment/processor-story-prototype-20261024/docs/build/CLAUDE_IMPLEMENTATION_BRIEF.md
- Product expectations: https://github.com/Arameair/Thought-Processor/blob/experiment/processor-story-prototype-20261024/docs/build/PRODUCT_EXPECTATIONS.md
- Test design: https://github.com/Arameair/Thought-Processor/blob/experiment/processor-story-prototype-20261024/docs/build/TEST_DESIGN.md
- Owner trial card: https://github.com/Arameair/Thought-Processor/blob/experiment/processor-story-prototype-20261024/docs/build/OWNER_TRIAL_CARD.md
- Existing first-use implementation tracking: https://github.com/Arameair/Thought-Processor/issues/9 and authorization work https://github.com/Arameair/Thought-Processor/issues/8

**Our fundamental rule:**
> Thought Processor controls the operational setup of thinking. Thought Engine controls the actual act of thinking.

Identity owns identity/trust/grant authority. Thought Engine owns observation intake, queue, model calls and thought processing. Processor connects, configures via the owning service, tests and explains. It must not fake Identity authorization or become a mandatory proxy for source→Engine/Engine→worker operations.

## Branch discipline

Always start implementation from the current **\`alpha\`**. Record the full Alpha SHA and create a **new, separate \`experiment/*\` branch** for your code. **Do not write code on this story branch, Main, Stable or Alpha.** Read the story branch as specifications only. Never promote or merge automatically. Any other repo modification requires separately confirmed integration scope.

## Build in demonstrable gates

**Gate 0:** Audit the actual repo and selected Engine + Identity interfaces. Record blockers and the smallest implementable stack.

**Gate 1 (minimum deliverable):** Build a truly runnable standalone Thought Processor with clean-window launch, honest empty states, one clear next action, small dynamic service/identity workspaces, read-only discovery, original UI/open-original fallback, accurate logs/metrics, redacted Copy Context, and reliable Stop/restart. Independent of Project Controller.

**Gate 2:** Read live status from the selected *real* Thought Engine and show only what its existing endpoint actually proves (service/product/status, queue, worker, timestamps); label unauthenticated discovery **UNVERIFIED**, not CONNECTED/SECURE.

**Gate 3a (requires Identity coordination):** If Identity provides its explicit pinned **Start Experience** launcher, integrate with it. Preserve Processor standalone launch and verify exact branch/SHA and process ownership. Do not create uncontrolled service lifecycle management in Processor.

**Gate 3b (requires real security backend):** Once Identity and Engine provide actual authenticated grants, use one independently authenticated Demo Sender S, entity X and Engine E; Identity issues scoped \`thought.observation.ingest\` permission; sender sends directly to Engine. Prove default deny, allow+202 queued, revoke+403, Identity unavailable+503, blocked queued work and zero unauthorized worker calls. Show real decision/event/worker/persistence receipts. If the backend isn't available, report **BLOCKED** and disable the real-authorized buttons. Never use Engine's local ON toggle or a fake \`source:"tass"\` claim as a substitute.

**Gate 4:** Complete the test matrix with exact evidence and provide a short owner runbook plus traffic-light report. Treat fixtures as fixtures, unavailable workers as unavailable, and one queued event as **not** proof a Thought formed.

Keep the UI focused on the actual first-use journey, not an elaborate demo filled with speculative cards. The user should always understand which selected **entity**, **feature**, and **runtime instance** they are seeing and what can be done next.

## Deliverables in the experimental branch

A runnable app, independent build/run/stop instructions, honest integration adapters, tests with evidence, redacted Copy Context, safe data path, process ownership information, a concise trial script, precise missing dependencies, and full SHAs. Finish with a table marking each gate **PASS / FAIL / BLOCKED / NOT RUN** and a recommendation **KEEP / REVISE / REJECT** for owner review.

**Do not call the integration complete if you only built a UI or ran mocked grant tests. Do not modify Alpha, Stable or Main without my explicit promotion approval.**

Please begin by reading all linked files, checking the actual current Alpha and collaborator APIs, recording what is feasible, then implementing Gate 1 and Gate 2 in your own experiment. Continue to real authorization only when those independent dependencies are genuinely available and verified.