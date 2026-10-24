> **Story revision — 24 October 2026:** Read [the revised owner journey](../story/01_OWNER_JOURNEY.md) and [blueprint handoff](../story/03_BLUEPRINT_HANDOFF.md) first. This build document predates the accessible-feature/identity-relationship direction and is **historical, not a current executable assignment** until reconciled. In particular, any earlier optional-GitHub or unapproved attach/control allowance is superseded by the [Identity-approved build package](https://github.com/Arameair/Thought-Processor/blob/experiment/identity-approved-build-package-20261024/prototype/START_HERE.md). Creating/searching features, broad dependency installation and version comparisons are not authorized implementation scope. The owner has authorized story edits only in this pass.

# Owner trial card — Thought Processor first prototype

**Use when Claude has a runnable candidate. This is an unexecuted, manual acceptance script, not evidence of a pass.**

Before the trial: open [product expectations](PRODUCT_EXPECTATIONS.md) and [test cases](TEST_DESIGN.md). Have Claude provide the exact experiment branch + full SHA, Alpha base, correct OS-specific Build/Start/Stop commands and an isolated data profile. **Do not run against real production TASS/Identity data by default.**

## The single user goal

> **Can I open Thought Processor, understand what is running, connect to what is permitted, try one real observation when supported, find out exactly what happened, and close it without breaking independent services?**

### Phase 1 — the minimum useful demo (no authorization required)

- [ ] **Start from clean profile.** Processor opens in its own window/browser without Project Controller. Nothing shows fake Connected, Authorized, LIVE, or Thought Created.
- [ ] **Explain the screen.** I can tell what Thought Processor does within the first few moments. The primary next action is understandable (usually connect/verify Identity).
- [ ] **Try wrong port.** Show a real NOT FOUND/WRONG SERVICE/UNVERIFIED result; not a green status.
- [ ] **Point to selected live Engine.** If a real Engine is available, I can inspect Engine-reported queue/worker status and see the last-checked time; otherwise the absence is honest.
- [ ] **Inspect a feature.** Open the selected original application UI or a safe Open Original UI fallback; don't show a fake cloned UI.
- [ ] **Open diagnostics.** Flow/Logs/Metrics and Copy Context provide real bounded evidence, no secrets/observation body copied by default.
- [ ] **Close/reopen Processor.** Check that independently running Engine/Identity still run. Cached statuses are refreshed or correctly labeled stale.

**Phase 1 acceptance:** A usable, honest operational workspace. It is **not** yet the promised secured Identity→Engine demonstration.

### Phase 2 — Identity's explicit demo launcher (only when built)

- [ ] Start Identity alone and confirm zero optional features is a legitimate healthy state.
- [ ] Press **Start Experience**; see exactly which approved Processor branch/SHA/data root will run.
- [ ] Launch or attach the selected candidate; **do not open old Thought-Controller instead**.
- [ ] Press Start twice; no second uncontrolled Processor or wrong build appears.
- [ ] Stop Demo; only session-owned children stop. Identity and preexisting independent services persist.

**Phase 2 acceptance:** Correct explicitly approved launch and safe session ownership. Not authorization proof.

### Phase 3 — real authorization and observation (only when Identity + Engine support it)

- [ ] Select one real entity X, one authenticated Demo Source S, one authenticated Engine E and the intended operation.
- [ ] Before approval, submit attempt is **DENIED**; Engine queues **nothing** and contacts **no worker**.
- [ ] Approve the exact grant **in Identity**; a fresh authenticated event sent directly from S to E is **202 queued** (not yet a completed thought).
- [ ] Show **Identity's real decision ID and Engine's event ID** in one evidence timeline.
- [ ] Revoke grant and retry with new event ID; Engine now **403 denies** without restarting; no model call.
- [ ] Stop Identity and retry; separate **503 authority unavailable** or specified equivalent, failing closed.
- [ ] Repeat the authorized direct event with Processor **closed**, if Identity is available, and confirm Processor was not the data relay.
- [ ] If genuine non-mock worker available, show only actual model/Thought results, including no-Thought outcome where relevant. If not, show **QUEUED / WORKER UNAVAILABLE**.

**Phase 3 acceptance:** All security-critical allow/deny/outage, actor proof and receipts are from real independent services. **If backend isn't ready, mark BLOCKED, not PASS.**

## Observations to capture during trial

**First-impression questions:** What should I click first? Am I looking at an entity, a feature, or a running instance? Did this operation actually run, or is it a proposal? What is missing?

**Reliability questions:** What happens after restart, after wrong port, after Identity timeout, and after closing Processor? Did any process get duplicated or killed? Can I recover without losing data?

**Honesty questions:** Can I tell authenticated from detected, permission from runtime, queued from processed, and model answered from thought persisted?

**Diagnostics questions:** Can I copy one concise failure report for Claude without exposing secrets? Does it include exact branch SHA, relevant process/port, error, request ID and timestamp?

## Simple traffic-light scorecard

| Area | 🟢 Pass | 🟡 Partial / Blocked | 🔴 Fail |
| --- | --- | --- | --- |
| **First-run clarity** | Clear honest next action | Usable but confusing or missing polish | Fake or misleading first-run state |
| **True runtime evidence** | Actual service info + freshness | Partial read-only integration | Invented/mismatched status |
| **Feature independence** | Closing Processor doesn't break others | Not tested | Processor required for hot path / kills unrelated |
| **Identity authority** | Genuine scoped allow/revoke/outage proof | Missing backend documented BLOCKED | Local toggle or spoofed sender represented as secure |
| **Guided test** | Real correlated receipts with correct stage labels | Awaiting identity/source/worker | Fake result or duplicate uncertain send |
| **Logs and metrics** | Reproducible and redacted | Some evidence missing | Secrets exposed / invented measures |
| **Clean Stop / Restart** | Exact owned processes stop, no orphans | Limited evidence | Kills others, corrupts data, duplicates process |

For tests not in scope or lacking real dependencies, select **NOT RUN** or **BLOCKED** instead of assigning a colored success.

## Owner feedback form

```text
Candidate branch:
Full commit SHA:
OS / profile:
Trial date:
Tested gates: G1 / G2 / G3a / G3b / G4

First impression: 🟢 / 🟡 / 🔴
Live evidence: 🟢 / 🟡 / 🔴
Independent operation: 🟢 / 🟡 / 🔴
Authorization: 🟢 / 🟡 / 🔴 / BLOCKED
Guided test: 🟢 / 🟡 / 🔴 / BLOCKED
Logs / metrics: 🟢 / 🟡 / 🔴
Stop / restart: 🟢 / 🟡 / 🔴

What worked:
What was confusing:
What failed:
Copy Context / sanitized error:
Unexpected processes/ports:
Most important change:
Decision: KEEP FOR NEXT EXPERIMENT / REVISE / REJECT / NOT YET REVIEWED
Promote to Alpha? NO unless separately and explicitly approved.
```

## Expected handoff back to Claude after owner trial

Give Claude the scorecard and Copy Context. Claude should:
1. Reproduce the failing test in its own isolated profile.
2. Identify which **authority** actually owns the bug (Processor, Identity, Engine, worker or session launcher).
3. Patch **only its own experiment branch** unless a separate repo's owner authorizes an integration.
4. Report actual before/after receipts and measured differences.
5. Leave changes unpromoted until owner explicitly accepts the candidate.

**Critical reminder:** a good-looking UI cannot compensate for failed authorization boundaries. Conversely, a real backend integration doesn't mean the experience is usable until the owner can understand it.