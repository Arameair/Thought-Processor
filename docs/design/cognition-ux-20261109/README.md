# Cognition UI study — live previews and opened Thought card (2026-11-09)

**Owner-requested visual handoff; proposed interface only.** These two repository-native SVG mockups reconstruct the corrected Cognition design direction discussed in the GPT-6 voice/UX session. They are **not** screenshots of running TASS and are **not** an implementation specification or permission to code. Original image-generation PNGs were shown in the conversation but are **not committed here**; the SVGs are repo-friendly visual restatements.

**Owner-approved distinction:**
> “Cognition shows what was produced, Development lets you experiment in how it's produced.”

## Open the designs

- **[Cognition — shelf and live previews](./cognition-shelf-live-preview.svg):** Netflix-style browsing of persistent Thoughts, recent outputs, and a visible highlighted current/featured Thought. The preview offers source question, status, latest recorded output and a reason to open the card. Live indicators should appear **only when actually backed by recorded runtime events**; the design shows PAUSED because the owner's photographed installation had Pulse paused.
- **[Cognition — clicked Thought card](./cognition-clicked-thought-card.svg):** A full readable Note, a selection list of exact outputs/history, evidence and source nudge correlation, truthful verification state, and one obvious **Investigate in Development** action preserving the selected record. The Note body is intentionally placeholder text, not copied private model output.

## Two scales, one job
1. **Shelf/card preview (glance):** What Thought is this? Is it live, waiting, recorded or finished? What was the latest real output? Is it worth examining? Small enough to browse many Thoughts.
2. **Opened card (investigate):** What did TASS **actually** write? What prompted it? Which Note and request/nudge produced it? What is recorded, unverified, missing, or only model-proposed? Where is the next action?
3. **Development (future separate workspace):** Contextual open of the producing processing workflow, eventually safe changes and separately authorized experiments. **No graph editing in Cognition.** Return to the same saved output and compare after a permitted experiment.

## Intentional limitations and engineering guardrails
- Cards must show **authentic persisted outputs**. Never fabricate model internals, team deliberation, decisions, or a stream of events because a mockup depicts them.
- The existing installed Cognition shelf UI is the **baseline to improve**, not something to discard in favor of this graphic.
- **P0** (already separately approved for Claude) covers truthful, owner-authorized saved Note visibility and associated nudge/step evidence **only**. The graphic does not expand that authorization.
- **P1** contextual Development navigation, **P2** isolated comparison experiments, five-role collaborative artifacts, and actual live activity presentation remain future design/engineering steps requiring appropriate evidence and approvals.
- Identity remains the authorization authority; do not trigger a model run by reading a card. The Thought Engine owns execution and Thought data.
- About the concept illustrations: `identity-test` is an existing named acceptance example, but the thumbnail art, statuses, layout and other entries marked **DEMO/Sample** are illustrative and not proof those items exist. Any `Trace NOT RECORDED` applies when the actual event/trace ID is missing.
- Keep the experience readable in a half-screen window, with graceful responsive stacking or inspector collapse.

## Designer's next critique
Use these two *specific* screens (not a general workflow diagram) to ask the owner: **What belongs in a small live preview? What appears only after opening the Thought?** Describe every state indicator's real backend evidence source. Do not start implementing P1/P2 as a side effect of this design review.

See [Thought-Processor Issue #106](https://github.com/Arameair/Thought-Processor/issues/106) and the accepted P0 scope for background.
