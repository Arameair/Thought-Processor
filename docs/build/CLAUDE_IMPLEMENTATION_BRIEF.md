> **Story revision — 24 October 2026:** Read [the revised owner journey](../story/01_OWNER_JOURNEY.md) and [blueprint handoff](../story/03_BLUEPRINT_HANDOFF.md) first. This build document predates the accessible-feature/identity-relationship direction and is **historical, not a current executable assignment** until reconciled. In particular, any earlier optional-GitHub or unapproved attach/control allowance is superseded by the [Identity-approved build package](https://github.com/Arameair/Thought-Processor/blob/experiment/identity-approved-build-package-20261024/prototype/START_HERE.md). Creating/searching features, broad dependency installation and version comparisons are not authorized implementation scope. The owner has authorized story edits only in this pass.

# Claude implementation brief — Thought Processor first experience

**Prepared 24 October 2026 | status: proposed implementation handoff, NOT a built or approved prototype.**

## Mission

**Build the smallest independently runnable Thought Processor that allows the owner to *experience* the narrative**, using genuine service data wherever supported and reporting unavailable security/integration features honestly.

> **Thought Processor controls the operational setup of thinking. Thought Engine controls the actual act of thinking.**

**Other boundaries:** Identity decides trust/permissions; Engine enforces grants and runs observation processing; source sends directly to Engine; Processor coordinates setup/test/visibility, not high-volume observation traffic.

Read the owner story before choosing architecture or screens:
1. [01 Owner Journey](../story/01_OWNER_JOURNEY.md)
2. [02 Responsibilities and States](../story/02_RESPONSIBILITIES_AND_STATES.md)
3. [03 Blueprint Handoff](../story/03_BLUEPRINT_HANDOFF.md)
4. [04 Review and Alternatives](../story/04_REVIEW_AND_ALTERNATIVES.md)
5. [Product expectations](PRODUCT_EXPECTATIONS.md) and [test design](TEST_DESIGN.md)
6. Related, potentially concurrent plan: [Processor first-experience brief](https://github.com/Arameair/Thought-Processor/blob/experiment/first-run-experience-brief-20261024/docs/FIRST_EXPERIENCE_DEMO_2026-10-24.md) and [Issue #9](https://github.com/Arameair/Thought-Processor/issues/9)
7. [Identity team launch brief](https://github.com/Arameair/identity/blob/experiment/first-run-experience-brief-20261024/docs/FIRST_EXPERIENCE_LAUNCH_2026-10-24.md), [Engine experiment](https://github.com/Arameair/Thought-engine/tree/experiment/identity-grant-enforcement-20261024).

**Discrepancy to resolve intentionally:** the story can begin by opening Processor directly; the newer owner-requested full experience begins at Identity's **Start Experience** button. Preserve **both** entry paths. Identity's launcher is explicitly user-initiated and pins the actual Processor build. Neither entry path requires automatic installation, hidden startup or turning Processor into the observation bus. Coordinate launcher changes with the Identity team; do not independently rewrite their work.

## Mandatory branch and scope policy

- Repository: **https://github.com/Arameair/Thought-Processor**
- Start a **new** \`experiment/<short-descriptive-name>-20261024\` branch from the latest **\`alpha\`**; record full base SHA first. **Never branch implementation off \`main\`, \`stable\`, or this story/documentation branch.**
- This story branch is the specification source; read/link it, and port only necessary documentation into your new branch with provenance.
- **Do not update, merge or promote Alpha, Stable or Main. Do not touch another repository without a separately agreed integration scope and coordination.**
- Commit incremental work to the experiment only. Leave failed candidate intact for review.
- Do not put data/secrets in Git checkouts. Use an isolated demo data root; safe start/stop must not kill pre-existing processes.

## The first playable experience: success looks like this

**A. Open directly (standalone):** run Processor with a small local server and its own clean browser/app window, without Project Controller. First launch shows an honest system summary: Identity NOT CONNECTED/UNVERIFIED, no live features verified, one useful next step (Find/Connect Identity), and an optional explanation of the system. Do not generate synthetic feature cards pretending to be installations.

**B. Attach to real services:** user can enter/configure a local Identity endpoint and select a local Engine endpoint. Display which checks were done—candidate found, product/protocol recognizable, authenticated or *not authenticated*, ready/blocked—and actual checked time. Show available real service metadata, port/PID/instance and exact commit **only when verified**; otherwise UNKNOWN. Never treat \`localhost\` or a product label as security proof.

**C. Explore:** minimal system overview → selected entity (TASS or demo entity if legitimately discovered) → selected independent feature workspace. Feature cards must be created from actual discovery/registered records; show an empty state when none exists. One selected service's original UI may be opened only with a trusted origin/known endpoint; if iframe embedding is blocked, **Open Original UI** in a separate view. Never draw fake Engine controls or rebuild the Engine as a second UI.

**D. Operate honestly:** show Readiness, Connection, Permissions, Activity/Flow, Logs, Metrics and Copy Context. Buttons have truthful enabled states and a reason when unavailable. Read-only status works if existing endpoints support it; privileged commands are **disabled until real actor auth, Identity grant and owning-service support are proven**. A request being accepted by Processor is not evidence the Engine actually applied it.

**E. One guided observation demo (integration-gated):** only after the source, Engine and Identity teams provide **actual authenticated** service identities and verified grant APIs. Select entity X, dedicated source S, destination Engine E and \`thought.observation.ingest\`; request owner approval through Identity; use S's verified credentials to send one event **directly to E**; show Identity decision/ID, Engine accept/deny/queue receipt, worker-call evidence and persistence status if any. A single event need not create a Thought. Revoke and retry to demonstrate next-event refusal; fail closed if Identity goes unavailable in first security gate. Do **not** fall back to an Engine-local grant flag or a \`source:"tass"\` body claim.

**F. Optional Identity launch flow (coordination-gated):** the Identity team supplies or approves a bounded **Start Experience** session runner: selected, owner-approved and pinned Processor build/protocol/data root; explicit consent; attach-versus-start shown; only session-owned child processes stopped. Processor must also remain independently launchable. If Identity's launcher is not implemented yet, Processor still demonstrates A–D and explicitly reports this integration **BLOCKED**, rather than inventing launch success.

## Proposed incremental engineering slices

### Gate 0 — survey and architecture note, before feature code

- Inspect \`alpha\` and exact SHA; note it is currently documentation-first, with no runnable Processor there.
- Read story and newer Issue #9/Identity #16. Inspect **actual** Engine APIs and authentication experiment rather than assuming docs equal running endpoints.
- Pick a minimal local stack consistent with repo/project expectations (Go-based lightweight local HTTP UI is a reasonable default, not a forced framework). Avoid installing a massive stack just for a card grid.
- Record a one-page design explaining modules: independent Processor server; service registry/discovery adapter; status adapter(s); UI/view model; optional test coordinator; bounded diagnostics; and no required live data proxy.
- Write a dependency/blocker ledger. Do not claim secure capabilities are implemented before building/verifying them.

### Gate 1 — independently runnable, truthful workspace (**minimum useful deliverable**)

- Build / start / stop as its own process, own port and profile directory; open in a clean browser/window (do not depend on Project Controller).
- Read-only local status of self and selected existing service endpoints. Clear empty/failure states with one next action. No fake uptime, metrics, worker replies, grant, instance ID or SHAs.
- Calm layout: small system overview, real app cards, one selected detail area, original UI/open-original fallback, collapsible diagnostics.
- Copy Context / Copy Logs with redacted, bounded data and exact provenance.
- Real stop and restart; profile isolation and no orphans.

**Gate 1 acceptance:** one non-technical owner can start Processor and identify what is real, missing and next in under a few interactions; developer can reproduce a WRONG PORT / ENGINE STOPPED failure and find the exact cause. **This is NOT end-to-end authorization success.**

### Gate 2 — selected Engine live read-only integration

- Connect to the exact selected \`Thought-engine\` experiment (or another **explicit owner-selected** first target). Do not silently migrate \`Thought\` data or substitute \`Thought-Controller\`.
- Read actual Engine \`/api/status\` and supported event/queue/worker endpoints. Mark **endpoint reachable but service unauthenticated** accurately if peer identity cannot be proved; do not label secure VERIFIED because a product string matches.
- Show Engine-reported worker availability, queue and real event state as **Engine-reported**, not invented. Read-only functions can be useful while privilege mechanisms are unfinished. Bound polling, timeouts and diagnostic capture.
- If the Engine API cannot meet the status contract, document exact missing field/behavior and escalate; **do not secretly rewrite another repo**.

### Gate 3 — Identity launch + grant integration (**BLOCKED until interfaces exist and teams coordinate**)

- Consume the pinned Identity **Start Experience** launch session, if Identity team exposes it; display process ownership, provenance, launch logs and selected candidate.
- Integrate authenticated service discovery and the Identity-owned grant check, using exact tested interface. Never emulate a production grant from a fixture.
- Use one dedicated test source with its own identity (not impersonated TASS) and direct source→Engine ingest. Identity permission **before queueing and before model send**. Show allow/deny/revoke/unavailable and effective read-back.
- Use the existing Engine grant experiment only after verifying its real compatibility with the Identity implementation. Its fixture tests are **not** proof of live integration.
- Processor must still work when Identity or Engine is absent and must not become required for source→Engine hot-path requests.

**Gate 3 acceptance requires real secured cross-process tests**. If blocked, ship Gates 1–2 with a precise BLOCKED report and leave all grant/test controls disabled or clearly marked not implemented.

### Gate 4 — owner trial and verification

- Reproduce the [test matrix](TEST_DESIGN.md), including startup with zero optional features, wrong service, missing Identity, worker unavailable, revoke/deny/offline, repeated session start, closure, orphan audit, and recovery.
- Provide an exact independent Run/Stop recipe and [owner trial card](OWNER_TRIAL_CARD.md).
- Include logs, full SHAs, PID/ports/run IDs, elapsed startup/test times, worker call counts if available, failure receipts and rollback plan.
- No auto-promotion. Request explicit owner verdict **KEEP / REVISE / REJECT**.

## UX and functional boundaries

- **A contextual next action, not just feature tiles.** Empty state -> Connect Identity; unverified -> Verify; missing permission -> Review in Identity; healthy but untested -> Run guided test; failure -> Inspect failed step.
- **Selected entity ≠ service ≠ process.** Keep meaningful source, target and entity IDs visible in detail/diagnostics.
- **Four levels of setup:** proposed, authorized, applied, observed. A grant doesn't mean Engine ready; accepted queue doesn't mean thought formed.
- **Only one live original app preview expanded.** Browser embedding policy may prohibit it; use a safe open-original fallback. No captive simulated application UI.
- **Truthful state:** NOT INSTALLED, FOUND/UNVERIFIED, AUTHENTICATED, GRANT UNKNOWN, DENIED, AUTHORITY UNAVAILABLE, READY, WORKER UNAVAILABLE, QUEUED, IN FLIGHT, UNKNOWN, FAILED; each has a supported evidence source and freshness timestamp.
- **Owner-click controls:** no hidden service starts, automatic retries of uncertain sends, destructive resets, grant bypasses or automatic GitHub writes.
- **Observability-first:** logs/flow/metrics available on demand; Copy Context defaults to redacted metadata; user text, prompts and secrets excluded unless an explicit scoped opt-in is safe and permitted.
- **Independent apps:** a user can close Processor without stopping a pre-existing Engine/worker/Identity. Session Stop stops only explicitly session-owned children if the separate approved launcher is present.
- **No requirement** for account GitHub sign-in, real TASS, Discord, ScreenBridge, Memory, production multiuser security, remote VMs, autonomous model heartbeat, arbitrary app installers, or self-modifying code.

## Interface integration discipline

Engine's current \`experiment/thought-engine-v1-20261023\` describes \`GET /api/status\`, \`POST /api/entities/{id}/events\`, per-event receipt endpoints and worker checks. It also has an engine-local grant toggle and **no service authentication** in that original experiment. The newer \`experiment/identity-grant-enforcement-20261024\` provides Engine-side experimental checks with fixtures, **not a confirmed real Identity service**. Check actual branch and current code before issuing commands.

Identity's current \`experiment/first-run-experience-brief-20261024\` describes a launcher but **no proven real authorization check**. Never couple to a guessed grant endpoint or tell the owner the bridge is ready.

To propose any new shared API, first record the owner, caller authentication, method, fields, response status/receipts, protocol version, timeouts, idempotency and sensitive-field policy. Keep interface adapters replaceable.

## Output Claude must provide to owner

1. **Experiment branch and exact full SHA** + Alpha base SHA, exact changed files, build/test commands.
2. **Launch instructions** independent of Project Controller, OS used, port/PID/run ID and profile/data root.
3. **Working demo summary** with specific Gates achieved—not a vague “complete”.
4. **Acceptance matrix** with every test marked PASS/FAIL/BLOCKED/NOT RUN, evidence source, known limitations and rollback.
5. **Owner trial procedure** short enough to follow manually, including at least one deliberately broken path.
6. **Copy Context / Logs sample**, redacted.
7. A precise list of changes **needed in Identity and Engine** as separate follow-up requests, not unapproved edits.
8. Final recommendation and owner decision **PENDING**.

**Do not claim:** “Integrated Identity authorization”, “verified trusted feature”, “model answered”, “thought persisted”, “grants enforced” or “ready” unless the named real service/actor actually supplied the necessary evidence.

---
**Authority:** Human owner reviews and accepts. This document was drafted with AI assistance from the owner-approved story concept and current related repository notes; details not individually owner-approved remain proposals.