# Expectations, risks and go/no-go gates — The First Spark

**Planning only · 2026-10-24.** Nothing here is a verified test result, project estimate, production commitment or owner approval of an unimplemented dependency.

**Read first:** [Claude build brief](CLAUDE_BUILD_BRIEF.md), [acceptance matrix](ACCEPTANCE_TEST_MATRIX.md), [revised story](../README.md) and [Trust & Evidence](../story/TRUST-AND-EVIDENCE.md).

## The central expectation

We are testing **whether controlling an independently running Thought Engine through a newly built Thought Processor makes the whole system coherent, understandable and trustworthy.** We are **not** testing whether a polished mock interface looks plausible.

The first successful prototype should answer:

> "I can open Processor, see the Engine's actual condition, ask for the approved Engine, know whether it started or attached, inspect what happened, and stop only the parts belonging to my session—without being misled about Identity, GitHub, publisher access or thoughts."

This is the first usable fragment of **[The First Spark](../README.md)**, not the entire customer journey.

## A product-like experience, with sharply limited claims

| Area | What we expect in the FIRST Claude build | What we must NOT claim yet |
| --- | --- | --- |
| User workspace | One independently running, calm, real Processor page and working controls | Full first-time consumer application completed |
| Processor process | Own status, build, PID/port/uptime and diagnostics from actual OS/process evidence | Persistent always-on platform or Windows service |
| Thought Engine | Start/attach one *selected* real Engine; show truthful state, exact build, origin trust level and safe stop | All possible Engines/versions auto-compatible |
| Readiness | Distinguish installed/running/reachable/compatible/authenticated/authorized/worker ready | Port open means secure or Engine can process |
| Logs/metrics | Real and traceable, bounded, redacted, copyable | Invented metrics or universal log aggregation |
| Identity permission | Clearly identify missing real authorization integration | Any true verified grant until Identity check API exists |
| AI worker | Detect real availability if supported, otherwise unavailable/unknown | Chat response, Spark/Trace/Thought, rate or speed not measured |
| GitHub/customer repository | Show setup required or actual verified existing account/repo evidence when available | New user source bootstrap, license or updates solved |
| Publisher feature access | Separate status and blockers, no self-issued entitlement | Remote kill switch, actual commercial distribution completed |
| Restart/Stop | Safe owned-process bookkeeping and reattachment | Perfect crash recovery on all OSes / across WSL or VM |

## Development milestones and what the owner should see

### Milestone 0 — A grounded technical inventory
Deliver a real current-endpoint list, branch/sha baseline, blocked dependencies and safe test run instructions. No UI necessary.

**Owner reaction to seek:** "That is what exists, and those are real blockers."

### Milestone 1 — Processor opens
A single real local page opens with its own authentic process status and Engine unknown/not installed. The page says it is an **integration trial**, not a verified product, if mandatory GitHub/repo or publisher prerequisites remain incomplete.

**Owner reaction:** "I can open a real Thought Processor, not just read a design."

### Milestone 2 — The Engine handoff
An explicit Ensure Engine Available action attaches to an approved running Engine or starts the correct approved one **once**, shows truthful readiness, and offers bounded logs/Copy Context/stop.

**Owner reaction:** "I understand what Processor does and I can trust which Engine it is working with."

### Milestone 3 — The first *owner-test candidate*
Milestone 2 survives negative testing: duplicate clicks, wrong build, port occupied, Engine exit, Processor/browser close, safe external-process handling, repeated restarts. All visible controls work or say why unavailable. Performance measured, not guessed.

**Owner reaction:** "Even when something breaks, it explains it rather than pretending to be fine."

**STOP HERE for owner review.** A working UI and local process handshake are a meaningful first prototype. Do not continue building a complete Identity/Engine/feature marketplace in the same unchecked pass.

### Milestone 4 — Real authorization (separate, dependency-blocked)
Real Identity service grants, authenticated source and Engine, signed/current decision, queue recheck, denial/revocation/unavailable tests. The owner can press **Try observation** only when its real authorization prerequisites are met.

**Owner reaction:** "I can revoke and the Engine actually obeys, without losing the independent service."

### Milestone 5 — Genuine user-owned GitHub onboarding and publisher delivery (separate open design)
Real account verification; customer-owned repository containing approved software and origin credit; separately authoritative publisher feature delivery decisions; honest recovery/update behavior. **Starter files, licensing, mechanism and access policy have not been chosen.**

**Owner reaction:** "The source and rights are actually mine, not a fictional account card."

## Risks that should shape the build, not surprise us afterward

| Risk | Severity | Early warning | Mitigation / owner checkpoint |
| --- | --- | --- | --- |
| **Fake operational success** | **Critical** | Hardcoded connected/authorized badge or sample Thought in live UI | Forbid seeded live data, require field provenance and explicit Unknown; reject candidate |
| **Untrusted localhost response** | **Critical** | Any process on same port is called the Engine | Verify executable origin/trusted pairing; report Reachable vs Authenticated; reject privileged actions |
| **Process killing or duplication** | **High** | Stop Demo kills independent service; Ensure spawns 2 Engines | One owned/attached ledger, idempotency, concurrency tests, isolated process group |
| **Grant authority confusion** | **Critical** | Processor changes Engine-local ON and calls it Identity permission | Disable such behavior, mark real Identity unavailable, fail closed |
| **Leaking credentials/observations** | **Critical** | Copy Context includes bearer token or private user content | Whitelisted redaction, canary tests, no secrets in repo/logs |
| **Unsanctioned GitHub starter copy** | **Critical** | Builder guesses license or copies restricted source | Keep repo creation blocked; owner approves source contents/license/process separately |
| **False publisher entitlement** | **High** | Client-side boolean treated as license | Require independent trusted publisher decision for delivery, no remote-erasure claims |
| **Scope explosion** | **High** | Several features/models/providers before first Engine handshake works | Enforce Milestone 3 stop/review gate |
| **Dependency chain break** | **High** | Identity signer absent; Engine tests written but not run | Distinguish Gate 2 lifecycle success from Gate 4 authorization; do not invent fixtures as integration |
| **Hidden cloud cost / consent** | **High** | Model call on startup or background retries | No automatic inference; owner-initiated one send with actual provider and usage evidence |
| **State destroyed on restart** | **High** | Data under git checkout, duplicate service/entity IDs | Isolated roots, stable durable IDs, idempotence, test reopen/stop |
| **Premature OS coverage claim** | **Medium** | Windows/WSL/Debian announced after only single-host demo | Declare tested OS and exact path; remote/VM separate gate |

## Three categories of "done" to avoid confusion

- **Built:** source exists, compiles and launches. Does **not** imply behavior is correct.
- **Locally tested:** actual tested behavior against real processes with recorded receipts and negative scenarios; only those scoped checks pass.
- **Owner accepted:** owner used the prototype and chose keep/revise/discard with specific notes. Only this can authorize a future Alpha promotion; it doesn't automatically do so.

No estimated completion date, success rate, request-per-hour figure or speed goal is set here. Establish a baseline with real measurements first, then decide whether an SLO is useful.

## Minimum owner test session

- Start from a **fresh disposable profile**; launch Processor directly and see truthfully disconnected statuses.
- Start the selected real Engine through Processor; inspect process/build/port and distinguish **started by demo**.
- Click Ensure again: no duplicate. Stop and restart; observe actual identities and timings.
- Start Engine independently, then attach: verify Stop Demo leaves that Engine running.
- Cause one safe error (wrong protocol or missing executable) and confirm the UI names the exact reason and offers a safe next step.
- Inspect Copy Context/Logs. A new developer should be able to diagnose the event without reading private content.
- Confirm the absence of a real Identity grant yields **AUTHORIZATION NOT READY**, *not* a success toast.
- Give one concrete verdict: **KEEP / REVISE / REJECT**. This determines which blueprint assumptions we retain.

## Stop-work / rejection triggers

Stop the candidate and surface it to the owner instead of polishing:

1. Processor launches unknown executables or cannot prove which processes it owns.
2. Privileged command path is unauthenticated, remotely reachable or bypasses local approval.
3. UI shows an authorization or AI result that no actual authority/worker returned.
4. Clipboard/diagnostics expose tokens, credentials or private input by default.
5. Prototype writes into existing production data or customer's source checkout unexpectedly.
6. It implements an unapproved repository redistribution method or unsupported license assumptions.
7. It automatically spends paid API calls or starts scanning sources without explicit approval.
8. Any suggested fix requires editing/promoting Alpha/Stable/Main without owner approval.

## Decision checkpoint after the owner trial

Record:

- **What worked?** Supported by test run IDs and receipts.
- **What felt intuitive?** Owner's direct feedback, not an AI's guess.
- **What failed or was awkward?** Include actual error and whether reproducible.
- **What remains genuinely blocked?** Specific missing Identity/GitHub/publisher/worker contracts.
- **What would we remove before adding anything?**
- **What will we test next?** Prefer one concrete experiment with an objective pass condition.
- **Do we keep the design split?** Processor control + independent Engine + shared bounded launcher, or simplify based on real test evidence?

**Success is a dependable, understandable small slice—not a complete imaginary ecosystem.**
