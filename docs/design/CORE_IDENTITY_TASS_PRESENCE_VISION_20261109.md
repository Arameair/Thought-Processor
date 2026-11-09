# FUTURE DESIGN NOTES — Core, Identity, and TASS Presence
**Owner concept discussion | 2026-11-09 | NOT AN IMPLEMENTATION ASSIGNMENT**

## Status and ordering
The owner is happy with this **design direction** and wants detailed notes for later, while Claude/Codex continue existing **Cognition and Development** work. **Do not interrupt, expand, or re-scope those ongoing implementations.** No authorization here to implement, modify grants, merge, deploy, activate Pulse, call a model, or create new services. The owner plans to test Cognition and Development first, then return to design and implement Identity and Core.

## Product identity
Build **one coherent TASS application**, not four disconnected admin dashboards. Maintain shared dark blue/purple cyber visual language, readable typography, controlled motion, recognizable character, clean panels and navigation between Core / Identity / Cognition / Development.

Guiding principle for the two current specialized workspaces remains:
> “Cognition shows what was produced, Development lets you experiment in how it's produced.”

- **Core / Workstation**: personalized everyday home and presentation surface for TASS. Owner interacts with work, information, progress and context, not a cluttered developer control panel. Voice and text may be offered, but do not make a permanently dominating Core chat window; conversation will already be available in Development.
- **Identity**: separate dedicated security/ownership/login/permission management screen. This is the authoritative security service, *not* just an appearance theme.
- **Cognition**: truthful recorded Thought/output browsing and inspection.
- **Development**: interactive and conversational investigation and, later, owner-authorized experimentation.

## Identity — confirmed owner preferences
- **Dedicated Identity screen** so there is a deliberate home for security, ownership, login/unlock and grants. Core shows basic locked/unlocked/permission status and a route to Identity, rather than hosting arbitrary permission switches.
- Permission controls should be grouped **by recognizable feature/function**, so users can find related permissions easily: Cognition, Development, GitHub, Communication Bridge, Core, with future groups as needed.
- A feature group summarizes what is permitted and allows drilling down to individual specific grants with short plain-language explanations, exact scope, and effect. Favor sensible grouping, search/filter, and simple mental models over long flat technical lists.
- **Security invariant:** grouping must not collapse authorization boundaries. A visual feature-level “on” switch cannot silently mint multiple independent grants or bypass per-action policy. The UI must correctly distinguish display status, requested changes, and effective decisions.
- Group examples shown in initial mockups are **illustrative**, not verified actual grant names or grant status.
- Later topics include security history, devices, and multi-entity isolation; these are **future design candidates**, not verified current capabilities.

## Core — living workspace visual concept
The owner **liked** the cinematic TASS Core design generated in the 2026-11-09 discussion: central stylized TASS digital figure, blue cyber/hologram motif with a fox/wolf-like companion, restrained navigation, contextual panels, and a compact real-system state strip. **The original generated PNG and user-uploaded avatar examples were in the chat, not committed to the repo**; request originals from owner for pixel-level fidelity.

The owner wants:
1. A **customizable workspace**: reposition/resize content and presence, control visual intensity and informational density; preferences must never hide critical security requirements.
2. A prominent **TASS character** on the home state, with subtle ambient/breathing/idle animation.
3. **Contextual information panels**: when asked about a lesson, a real lesson card can appear; when discussing an existing output, show its authorized record; when discussing permissions, surface status and link to Identity. Avoid a permanent stack of every system control.
4. Persistent minimal status: Identity locked/unlocked, authorized run/activity state, Engine/model connection, Pulse/autonomy if actually available.
5. **Focus mode**: TASS minimizes to a small indicator, and large documents take priority.
6. Cohesive navigation to Cognition, Development, Identity; Core does not become a fake verification dashboard.

## TASS Presence — persistent floating avatar
**Owner preference:** TASS should be able to **control his avatar throughout the different screens of Thought Processor**, not merely be shown separately inside Core. The floating avatar is a future **presentation interface** for the same TASS entity, **not** a new AI/model/agent/service.

Three proposed presentation modes:
- **Full presence**: hero-sized animated figure on the Core home screen.
- **Floating companion**: small, draggable, repositionable avatar during work in Core, Cognition and Development. The avatar can move aside, indicate relevant elements, draw attention to a real recorded output, and present a read-only card.
- **Focus/minimized**: unobtrusive small icon, hidden/quiet animations on owner command; usable on narrow windows.

Expected **future** presentation actions are conceptual, not working APIs:
- `move` (within bounded viewport / no inaccessible positions)
- `point` and `highlight` (only valid visible targets; never impersonate a clickable control)
- `show_card` (only currently owner-authorized information from existing app projections)
- `speak` / indicate response (without claiming lip-sync if no signal exists)
- `minimize`, `dismiss`, `restore` (owner override)

**Owner direction on initiative:** TASS may **automatically show authorized read-only information** as he works, without asking on each such display. He can suggest what to do next. Actual mutation or protected execution remains behind current Identity authorization and any necessary owner confirmation.

**Three-part interaction boundary:**
- **Show**: bring forward an already authorized read-only item / highlight existing app content.
- **Suggest**: present a proposal and its basis; owner can inspect/ignore/dismiss.
- **Act**: only via the normal explicit runtime action, Identity grant and approval, with separate evidence. `show_card` or `highlight` must never convert into `click`, `POST`, grant, model run, or workflow promotion.

## State-driven animations (proposed only)
Tie meaningful behavior to **actual observable source status**, never invent invisible model thinking:
- **Idle**: subtle breathing and ambient cyan glow (this is aesthetic, not proof of a live thought).
- **Processing**: gentle energy motion only with an authenticated recorded running job; label source/time.
- **Waiting for owner approval**: amber cue and relevant approval/context card; no implied permission.
- **Reply/output available**: character turns toward the real output panel, with a one-time cue.
- **Completed**: short visual acknowledgement only when a terminal receipt exists.
- **Locked/disconnected/unknown**: muted avatar and legible state, no false running cues.

Thought lifecycle (active/sleeping/resolved), Core nudge/run activity (queued/running/recovery/completed), and Engine Pulse control (paused/idle/busy) are **separate state domains**. Do not derive avatar “working” from a Thought merely marked active. Twenty-second Pulse is a periodic coordination cycle, not a task deadline.

## Suggested implementation architecture to investigate later
An **app-wide presentation layer** may be shared across Thought Processor pages, maintaining a single avatar identity and owner-configured location/mode. TASS can **request validated presentation intents** through an allowlisted contract. The UI renders and locally validates those intents; the currently active page resolves target IDs from its actual authorized view. Core serves existing authorized projections/coordination where appropriate; Thought Engine owns Thought records/execution; Identity remains the sole permission authority.

Open engineering questions, **not design decisions yet**:
- How to persist one avatar's position/state across full page navigations and browser sessions without a separate app or breaking existing view routing.
- Which concrete components expose current, owner-authorized UI target IDs. How to avoid stale target references and the avatar covering a permission/approval control.
- Whether same-origin navigation permits a shared shell, minimal page-included renderer, or another low-risk reuse approach; inspect existing code and CSP rather than assume new framework.
- Accessibility: reduced-motion support, keyboard focus, screen-reader alternatives, unobstructed content, mobile viewport support.
- Privacy: avoid putting sensitive Note text into broadcast/global state, logs, avatar events or unauthenticated overlays. Show cards through already-authorized data reads.
- Rate limits and UX: avoid unsolicited panel spam, interruption during active editing, or movement that hides work. Dismissal and quiet-hours/focus override remain owner controlled.
- How owner preferences are saved per identity, per workspace and per device; support future multi-entity isolation.
- Which avatar animation assets are actually available/licensed and how to decompose them into reusable animations. Current image/GIF examples **do not** prove a rigged 3D asset or working motion API.

## Owner-level success examples (future acceptance, not test results)
- From Core, TASS is visible as an animated presence; opening Development changes him to a small floating companion without losing workspace focus.
- While speaking with TASS in Development, he can point to a **real visible block** and show a corresponding read-only explanation; neither makes a graph change.
- When Cognition receives a real completed output, TASS may surface an authorized preview; clicking opens exact persisted details. A fake/offline/unknown event never appears as success.
- Identity remains visibly authoritative. An unauthorized record cannot be shown simply by an avatar request; automatic presentation cannot trigger grants or execution.
- Owner can move, minimize, dismiss or silence the avatar, and it stays out of the way while reading or editing.

## Explicitly deferred
No code or Issue requests to implement this yet. No new frontend framework, separate agent, model invocation, Pulse action, live telemetry, permissions redesign or execution authority is implied by the concept. Revisit **after** current Cognition and Development stages and owner testing, then inspect the actual Core/Identity source and derive a narrow staged engineering plan with independent review.

## Discussion references
- [Thought-Processor Issue #106](https://github.com/Arameair/Thought-Processor/issues/106) — owner UX walkthrough and product direction.
- [Cognition visual design notes](https://github.com/Arameair/Thought-Processor/tree/design/cognition-live-card-mockups-20261109/docs/design/cognition-ux-20261109) — related but separately scoped.
