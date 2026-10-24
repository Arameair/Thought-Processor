# Prototype expectations — what “working” means

**Draft owner acceptance contract · 24 October 2026 · not a claim of implemented features**

The owner wants to experience a real, small, modular software product—not a static architecture diagram, a cosmetically complete feature catalog, or a fabricated Thought Engine.

**Core sentence:**
> Thought Processor controls the operational setup of thinking. Thought Engine controls the actual act of thinking.

## Outcome we are buying with this prototype

Within one isolated local session, the owner should be able to understand:

1. What is running and where the evidence came from.
2. Which identity/entity is selected versus which independent service is running.
3. Whether authorization is truly controlled by Identity rather than Processor or Engine local flags.
4. What the owner can do **next** to achieve an actual observation-processing result.
5. Why a deliberately broken or unavailable path is blocked.
6. What happens when Processor closes or when the selected demo is stopped.

**Experience goal:** “I understand what this system is doing, and I can tell whether it actually worked.” A successful UI is not evidence of successful inference or safe authorization.

## Entry paths — both must be considered

| Entry | Expected behavior | Dependency |
| --- | --- | --- |
| **Standalone Thought Processor** | Starts independently with own server/profile/window. Honest status even when Identity is absent; read-only discovery without false trust. | Processor alone |
| **Identity → Start Experience** | Identity starts healthy without optional modules; explicit owner action chooses the pinned Processor and demo services. Starts/attaches safely; user sees provenance. | Identity team's separate session runner integration; later gate |

**Opening Identity must never silently start every service.** “Start Experience” is an explicit demo workflow, not permission for automatic startup at login or across the installation.

## Gate definitions and expected deliverables

| Gate | “Done” observable definition | May be called | Must NOT be called |
| --- | --- | --- | --- |
| **G0** Discovery/engineering survey | Branch, dependencies, API audit, blockers recorded with evidence | Contract/design inspected | Working prototype |
| **G1** Standalone operator UI | Real Processor process, clean first-run, accurate empty/wrong-port states, restart/stop, redacted diagnostics, one next action | Usable operational shell | Verified Identity integration |
| **G2** Read-only real Engine integration | Actual engine status/worker/queue data, with timestamps/provenance and honest unverified auth label | Real engine telemetry viewed | Authorized source→Engine data flow |
| **G3a** Identity-approved launcher | Explicit Start Experience starts/attaches correct pinned build, owner sees process ownership, safe Stop | Demo session launched | Live grant enforcement |
| **G3b** Real authorization and observation | Authenticated source S, authenticated Engine E, Identity-scoped allow/revoke/offline, direct event and correlated receipts | End-to-end authorization tested (if all checks pass) | Production security certification |
| **G4** Owner trial | Repeatable owner script, evidence matrix, known failures, rollback, no orphans or branch changes | Owner-ready candidate | Promoted / accepted until owner decides |

G3a and G3b are **separate dependencies** and may be delivered in either order if interfaces dictate. They cannot be silently bypassed. A release with G1/G2 and BLOCKED G3 remains useful if labeled correctly.

## Initial experience — content hierarchy

**Initial view:** the installation/system context, selected workspace if previously chosen, verified runtime summary and exactly one suggested next action based on actual evidence.

**Primary visual structure (proposed, not a locked mockup):**
- Header: Processor product identity, profile, running instance, current status and inspection timestamp.
- Compact real application tiles: Identity, Engine and any genuinely registered/verified features. Unknown features never masquerade as installed.
- Selection area: *Entity/workspace* (for example TASS) and *Feature/instance* independently.
- Focus panel: real original application's UI, or safe **Open original UI** fallback. Only one preview expanded.
- Collapsible diagnostics: Connection, Flow, Logs, Metrics and Copy Context.
- One **Next action** appropriate to current state rather than a set of unrelated buttons.

**No companion required:** a system with only a business identity or an independently installed non-thinking feature must not be labeled globally “Brain failed.”

**No original UI replacement:** Processor provides the control/diagnosis envelope. It should not clone every feature screen or pretend that a simulated feature card is an actual app.

## Truthfulness contract for status

| Displayed claim | Minimum evidence to show as confirmed |
| --- | --- |
| **Found** | Real endpoint probe or catalog reference (still unverified) |
| **Authenticated** | Valid peer identity verification; self-reported service name and loopback address are insufficient |
| **Authorized** | Current Identity decision for the exact authenticated source, Engine, entity and action |
| **Running** | Timestamped response from the selected real instance |
| **Worker ready** | Real owning Engine/worker readiness, not a placeholder badge |
| **Queued** | Real Engine event receipt; 202 does not imply model work |
| **Processed** | Real Engine processing/worker evidence tied to event ID |
| **Thought persisted** | Real storage/result receipt, not an optimistic spinner or one model reply |
| **Stopped** | Process exit confirmed where Processor or session runner owns the child; no inference about outside processes |

A stale status must show last-checked timestamp and cannot silently remain green. **UNKNOWN** is an acceptable, honest outcome.

## Required failure behavior

- **Wrong port/wrong app:** do not claim Identity/Engine or attach by guessed product string.
- **Identity missing:** Processor UI still runs, shows missing authority; protected integration controls unavailable.
- **Identity present but no auth API:** show **AUTHORIZATION NOT IMPLEMENTED**, not fake grant ON.
- **Engine stopped:** source not treated as processed; last logs retained but marked historical.
- **Permission denied/revoked:** next new observation rejected with an Identity+Engine receipt; no model call. Queued event must recheck before worker invocation.
- **Identity authority unreachable:** initial experiment fails closed; distinguish **503 authority unavailable** from **403 explicit denial**.
- **Worker unavailable/mock/uncertain:** never invent inference; event can remain genuinely queued if previously authorized. Do not automatically resend uncertain model calls.
- **Original UI cannot embed:** offer Open Original UI after validating origin and access; do not inject browser credentials into a URL.
- **Repeated Start/Stop:** no duplicate Processor instances, no external process takeover, no unowned process killed.
- **Copy Context:** redact raw prompts, observations, credentials and sensitive logs by default; report omitted fields.
- **Conflicting requested vs applied config:** display pending/conflict, not a false success.

## Performance, reliability and usability expectations

**Correctness gates (zero tolerance for an accepted demo):**
- **Zero** invented green/authenticated/authorized/processed/persisted states.
- **Zero** unauthorized worker calls in intentional denial tests.
- **Zero** secret values in sanitized diagnostics, URLs or repo commits.
- **Zero** pre-existing or unrelated services killed by Stop Demo.
- **Zero** automatic Alpha/Stable/Main branch promotion.
- **Zero** hidden new runtime dependencies on Project Controller.

**Observe before hard SLA**: capture cold start, status probe, Identity decision, Engine intake, worker first reply/completion, result persistence, Copy Context and Stop timings. Record n and distribution; do not invent a “fast” score without trials.

**Usability trial goals (proposed owner checks, not measured):**
- First-use explanation: the owner can state what Identity, Processor and Engine each do after viewing the first experience.
- Find a next action: visible without reading the entire README.
- Diagnose missing Identity, wrong Engine, or denied grant without guessing which app failed.
- No more than one main live preview expanded at once; detailed logs behind deliberate action.
- UI remains coherent on a narrow viewport and with long service names; keyboard navigation, visible focus, and sensible contrast.
- Responsive UI should not block or freeze when a remote endpoint times out.

## Explicitly deferred from the first real demo

Real game feeds, TASS Discord messages, arbitrary user-feature installers, customer monetization, Memory retrieval, autonomous dynamic coding, always-on model heartbeat, multi-user production identity security, remote/VM trust, full OAuth/GitHub account linking, a complete marketplace, distributed queues, and general lifecycle ownership of all software.

Don't accept “we built the full vision” as a successful report. **The first prototype succeeds by making one small path understandable, testable, independent and honest.**

## Evidence and owner decision

A demo is **Owner Accepted** only after owner trial, linked exact SHA and evidence. Reports must explicitly differentiate **PASS / FAIL / BLOCKED / NOT RUN**. Code inspections and fixture green tests do not establish real cross-service authentication.

Claude must close by requesting owner verdict **KEEP / REVISE / REJECT**, and by listing known limitations. No branch promotion.

Related: [Claude build brief](CLAUDE_IMPLEMENTATION_BRIEF.md) · [test design](TEST_DESIGN.md) · [owner trial card](OWNER_TRIAL_CARD.md) · [original story](../story/01_OWNER_JOURNEY.md).