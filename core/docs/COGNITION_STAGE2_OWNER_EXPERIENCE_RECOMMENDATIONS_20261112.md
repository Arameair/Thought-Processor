# Cognition Stage 2 review — owner experience recommendations (2026-11-12)

**Status:** Advisory / documentation only. **Not** implementation authorization, approved acceptance, merge, deployment, new permissions, or autonomous model execution.

**Context:** Owner conversation after Stage 1 WP1-E load measurements. The owner considers direct experience of TASS's behavior a more useful indicator of progress than another isolated technical feature. This document is a set of review criteria, not a replacement for the approved Stage 2 plan.

## Product distinction

- **Cognition = observe:** Follow the actual persistent Thought lifecycle, recorded activity, Notes, timestamps, sources, and recorded results. When a new Note is saved, make it understandable and accessible. Display the recorded processing events, not an invented real-time chain of model reasoning.
- **Development = experiment:** For a selected Thought, view an evidence-grounded graph of actual configurable processing relationships, propose a bounded change, execute it only under the proper Identity grant and explicit authorization, and compare the recorded outcome.
- The two views should refer to the **same Thought ID, source record, evidence provenance, and authorized owner/entity scope**, with a preserved path back to Cognition. Avoid fabricating connections or suggesting that a visual prototype graph changes live cognition until a verified execution path exists.

## Stage 2 review recommendations

1. **Make observable behavior the acceptance goal.** A user should be able to select a Thought and understand what was recorded, what changed, and what TASS actually wrote. Prefer a straightforward Note/activity chronology for passive observation; don't require an advanced graph to read a Note.
2. **Show truthfully what is recorded.** Separate Thought lifecycle (active/sleeping/resolved), Engine-wide pulse status, Core nudge/run receipt, model-generated suggestion, and independently verified result. Do not display BUSY or THINKING on an individual Thought without an event that supports that statement.
3. **Handle latency and staleness explicitly.** New records should appear on refresh or authorized updates; show as-of timestamps and unavailable/paused conditions honestly. Maintain existing Identity scope attestation, response-boundary checks, privacy purge, and no Note text in library previews.
4. **Preserve simple navigation.** From a Thought, the owner should see its full Note without hunting through logs, then be able to follow the source/evidence and, eventually, try a change in Development with the selected Thought intact. A link alone is not a useful handoff if the destination offers no clear next action.
5. **Don't pre-design the experimental UI.** Owner tentatively favors side-by-side baseline vs experiment, but explicitly wants to try it before treating that layout as a requirement. Build only the smallest usable comparison once the approved experimental stage allows it.
6. **Protect the observation/experimentation boundary.** Stage 2 read-only inspection must not accidentally execute a model, modify a Thought, create a grant, deploy code, or enable autonomy. Any future A/B run needs its own approved scope, Identity decision, isolated execution and durable provenance.

## Acceptance evidence to request (do not presume these pass)

- Automated: real/pinned Core–Engine–Identity integration with synthetic data; permission denial and revocation; no leaked private Note text; no unexpected model calls/writes; lifecycle and delayed-response tests; half-window and mobile/desktop browser presentation.
- Manual **batched** owner acceptance, only after automated checks: can the owner locate one Thought, see its actual newest Note and earlier development, understand what is recorded versus suggested, and return to the same context? If any item remains manual, record its reason, precise candidate SHA, expected outcome and rollback notes.
- For Development later: after a separately authorized isolated experiment, compare baseline and variant with source revisions, one changed variable, observed outputs, and a clear statement of what did or did not improve. Do not infer genuine self-improvement from one generated Note.

## Operating rules

Codex should attempt accessible automated local/WSL/browser tests and attach raw results before owner involvement. Claude implements only focused authorized deltas; Codex audits the exact resulting SHA. Avoid repetitive full-gate runs for unchanged revisions. Group non-urgent owner checks into one acceptance session; explicit owner approvals for permission, model use and deployment remain separate.

## Scope and dependencies

- The 2026-11-12 WP1-E native test results show a 100-Thought Engine response close to the 6 MiB cap. Engine issue #21 is a **proposal**, not permission to add a new endpoint. Keep performance/capacity work separate from the observation UX.
- Stage 1 final sign-off, if any, must be determined from its latest acceptance records, not inferred from passing load samples.
- Existing references: Thought-Processor PRs #107 (Cognition/Development review), #109 (Stage 1 plan), #112–#115 (Stage 1 evidence), issue #111 (workflow), and Thought-engine issue #21.
- **No code, schema, test, privilege, grant, or deployment change is requested by this recommendation note.**
