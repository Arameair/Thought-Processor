# Thought Processor — design discussion agenda for the first Alpha candidate

**Prepared:** October 26, 2026  
**Status:** **DISCUSSION CHECKLIST — NOT A BUILD ORDER OR APPROVED SPECIFICATION**. This is the list of questions to settle *before* approving the first system-wide Thought Processor prototype design. No source code, API, experiment, release, or Alpha promotion is authorized by this document. Decisions and their rationale belong in [design issue #15](https://github.com/Arameair/Thought-Processor/issues/15).

## Outcome we are aiming for

Not another read-only technical dashboard. The first **Alpha candidate design** should describe **one small, real interaction** that a person can complete in Thought Processor to understand and manage an Identity-backed system, observe correct results and tell us what feels right or wrong. The original diagnostic shell may supply reusable engineering components, but **it is not the product definition**.

**Important distinction:** “Alpha candidate” means a proposed, unpromoted experiment. It is **not** an Alpha release and does not have to include all future features.

## Already decided — do not reopen without a reason

- Thought Processor is the **normal human/administrator interface for the whole system**; Identity is the **independent identity, registry and permission authority**; each receiving feature verifies and enforces applicable grants.
- **Identity and Processor may start in either order**, as long as they reach a **clean, authenticated operational end state**. Full readiness may not be claimed based on an open port.
- An identity and newly connected features have **no data-source access by default**. Publisher eligibility to download/install (temporary GitHub repository-visibility rule during prototype) is **not** runtime permission to use a feature or access information.
- Independent features own execution and data, and may serve multiple identities with **isolated** memory/thought/permission contexts. Processor is not a required event relay or alternative data store.
- **Explicit Processor app exit** prompts **Shut down managed workspace (default)** or **Keep services running**, with ownership-aware clean shutdown and orphan detection. Closing one browser tab does not equal confirmed app exit.
- Operator approves the **first service pairing**, approved peers can reconnect securely, and owner can revoke/unpair. Pairing does **not** grant feature/source access.
- **Fail closed when Identity is unavailable**: no new protected actions or protected queued dispatch based on a cached or offline allow. A process can remain running for safe diagnostics.
- Memory is an **entirely separate future feature**, not being designed/built now. Focus sequence: **Identity baseline → Thought Processor now → Thought Engine next**. No permanent selection between `Thought` and `Thought-engine` yet.
- New functional experiments start from owner-controlled `alpha`; candidate acceptance and all promotions remain **manual owner choices**.

[Full owner-decision register](CROSS_REPO_DECISIONS_2026-10-26.md).

## Phase 1 — Decisions needed to define the **experience**

Work through **one question at a time**. An item counts as resolved only when the owner chooses, we record why and alternatives, and we can describe a specific expected test result.

### 1. What is the one job this first prototype must actually accomplish? **[START HERE]**
**Decision:** What does the user successfully *do*, not merely look at? Possibilities: choose an Identity and inspect its actual features; complete secure Identity connection; request one real approved connection; or operate one genuinely supported feature action.

**Evidence of answer:** One sentence in the form **“When I open Processor, I can ___, and I know it worked because ___.”** Decide the smallest useful behavior **before** choosing pages, components or colors.

### 2. Who is the first person using it, and what do they already have installed?
**Decision:** Is our first case **the owner on an existing system**, **a second person on a clean Windows computer**, or both with separate gates? Does the person already have Identity running and approved, and must GitHub be connected? Where do we deliberately stop setup?

**Evidence of answer:** Two short test personas/journeys (experienced owner vs first-time operator) with clearly identified prerequisites. No Project Controller dependency for an ordinary user.

### 3. What is the normal first screen and navigation?
**Decision:** Is **Profiles → Catalog → Collections** the right layout, and which page should open first on a fresh installation and on return? Should service/admin identities be visible on the normal profile chooser or only in advanced settings?

**Evidence of answer:** Rough wireframe of welcome/empty, one profile's home, and return visit. Establish a single calm screen, not stacked diagnostic panels.

### 4. What does a profile represent, and how does switching work?
**Decision:** Which identity types appear in the first prototype (person, companion, business, system/service), and what does selecting one actually change? Can the UI create/edit a profile already, or only show identities Identity really provides?

**Evidence of answer:** One identity with one feature and a second separate identity; no data or permission bleed on switch. Processor must not mint fake canonical identities.

### 5. What goes on the first feature card?
**Decision:** Which **real available features** must appear initially: Identity status itself, Thought Processor, optional Thought Engine, or a verified independent test feature? Which facts are essential on a card: entitled, installed, paired, enabled for selected identity, authorized sources, running, last activity? What action is genuinely possible?

**Evidence of answer:** A normal card, an uninstalled/denied card, and a stopped-but-installed card with **one clear next action**. No fake services or green “connected” label based on discovery alone.

### 6. What is the first meaningful action and its exact acceptance receipt?
**Decision:** Which action must be real for P0? Options include read-only but useful Identity-backed status/profile selection (a **navigation prototype**) versus an owner-approved **feature/grant request** with Identity-issued decision and receiver-enforced result (a **management prototype**). Don't promise permission management if the backend cannot enforce it.

**Evidence of answer:** Describe **click → authoritative request → accepted/refused → verified observed effect**, or explicitly mark the operation unavailable and reduce P0 scope. This is the most important test of whether the application is more than a shell.

## Phase 2 — Decisions needed to make that experience **credible**

### 7. How does Identity connect, authenticate and recover?
**Decision:** How can Processor locate the independently running Identity service, confirm the right installation/admin, and recover if it starts first or Identity is down? Which **minimum safe read API** needs to exist and which operations require separate administrative authorization?

**Evidence of answer:** The two startup orders reach the same verified Identity/profile. Unverified, denied, offline and mismatch are visibly different. Do not let a health endpoint or local port confer trust.

### 8. How are pairing, permissions and access explained to a person?
**Decision:** What does the one-time pairing confirmation say, how is the approved peer remembered, and where can the person disconnect it? How does the interface distinguish install eligibility from data-source authorization and revocation without burying someone in security jargon?

**Evidence of answer:** First-pair, restart-and-reconnect, wrong-service, revoke, no-sources-approved and GitHub-access-without-runtime-grant example screens. Actual crypto and key storage are technical follow-up design, not something to improvise visually.

### 9. Who starts and stops services, and what happens when Processor exits?
**Decision:** For first Alpha candidate, do people start Identity/other services themselves, use a small approved managed runner, or both? Which services are **managed vs external**? What is the explicit app-exit surface, and can someone stop running services after choosing “Keep running”?

**Evidence of answer:** One visible process inventory, startup error handling, shutdown/keep-running prompt, verified graceful stop, no orphans, no termination of unowned services. Browser tab close is not a shutdown signal.

### 10. What does “not ready” look and feel like?
**Decision:** What should a user see if Identity is stopped, wrong-version, unpaired, offline, permission denied, feature not installed, service busy, stale, or crashed? How do we avoid “localhost refused to connect” looking like a Processor crash?

**Evidence of answer:** Clear plain-language next step and honest diagnostics; **fail-closed** protected actions while Identity is unavailable and no cached “green” state.

### 11. Where do feature results and technical details live?
**Decision:** Should the chosen card open a single embedded preview when supported, or the original feature UI? What belongs on the home screen versus in a collapsible **Flow / Logs / Metrics / Copy Context** inspector? Will Collections show only genuinely source-owned records (or be omitted at first)?

**Evidence of answer:** One chosen app preview, truthful empty Collections state, no second Thought database, sanitized logs and exact run/correlation IDs where they exist.

## Phase 3 — Decisions needed to call it a **testable candidate**

### 12. What exactly is included, and what is explicitly postponed?
**Decision:** Set the **minimum test scope** and write down what does *not* need to exist yet. Suggested narrow P0: real authenticated Identity/profile discovery + profile selection + honest feature catalog/status + one genuinely usable interaction; P1 later: real Identity grant/deny/revoke enforced by a receiving feature. If the available Identity APIs do not support P0, record the missing contract as a blocker rather than introducing mock data.

**Evidence of answer:** A two-column **IN / OUT** list, exact required API contracts and which repository owns each missing interface. No unapproved Engine redesign or Memory implementation.

### 13. What first-run and repeat-run tests prove that we can learn from it?
**Decision:** Who tests (owner and optionally a second user), on which OS/environment, with which sample identities and known authorized/denied conditions? What does **PASS**, **BLOCKED** or **FAIL** mean? What feedback about usability will we collect?

**Evidence of answer:** A short script on the target Windows computer: clean start and both service orders, verified profile and feature state, permission denial, independent restarts, explicit exit prompt, owned-process cleanup, errors/redaction, and user reactions (“confusing”, “expected something else”, “useful”, “unnecessary clicks”). Every run identifies **exact branch/SHA**, time/ports/PIDs, fixture vs real evidence, and rollback.

### 14. How will we know the design is **ready for a build assignment**?
**Decision:** Agree on entry criteria: a single plain-language outcome, approved first-run journey/wireframes, agreed P0 cut line, real service/API ownership, a small negative-test matrix, known risks, and the owner's authorization for an **experiment from Alpha**.

**Evidence of answer:** A short approved design brief and trial checklist. The actual Alpha candidate would be developed/tested **later** in its own experiment; **no Alpha promotion** without a separate explicit owner decision and reproducible evidence.

## Priority and deferrals

**Discuss first:** **#1 (first real outcome)** → #2 (tester) → #3 (first screen) → #4/#5 (profile and card) → #6 (first real action). These establish what kind of app we are building before we spend time on infrastructure.

**Do not spend the first session re-arguing:** login provider implementations, release signatures, model inference quality, canonical `Thought` versus `Thought-engine`, Trace-vs-Thread migration, long-term Memory, Attention/Decay, automatic ingestion of logs, system-wide process supervisor, paid providers, multi-host distribution or autonomous agent features. These become their own workstreams **when a chosen P0 experience needs them**.

### Decision record template (fill one at a time)

> **Question number:**  
> **Owner's decision and date:**  
> **Alternatives considered / deferred:**  
> **Why this choice:**  
> **What the user will actually see or do:**  
> **Dependent Identity / Processor / feature contract:**  
> **Smallest test with observed PASS / FAIL / BLOCKED:**  
> **Remaining uncertainty:**

## Practical milestone

**Stop designing broadly once we can describe and draw one complete flow:**

**Open Processor → verified Identity/admin state → choose a real identity → inspect the real feature catalog → perform one meaningful supported action (or clearly scoped initial navigation test) → see a truthful result → exit safely.**

Then seek **owner approval of the design** before writing another runnable candidate. Only afterward should we build, test, compare, and decide what deserves Alpha promotion.

[Current issue #15](https://github.com/Arameair/Thought-Processor/issues/15) · [Current design readiness](DESIGN_READINESS_2026-10-26.md) · [Owner decisions](CROSS_REPO_DECISIONS_2026-10-26.md) · [Status/evidence](STATUS.md) · [Branch workflow](WORKFLOW.md)
