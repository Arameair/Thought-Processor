# Design discussion — dynamically discovered live application cards

**Date:** 2026-10-23  
**Status:** OWNER DIRECTION + DESIGN DISCUSSION; **not coded, integrated or verified**  
**Branch:** `experiment/dynamic-feature-cards-notes-20261023`, created from `alpha` @ `9fac9fa44cb020bb6f8aa6bcb1e9faeda6e0b47a`.  
**Attribution:** AI-assisted documentation of a human-directed design. This is not a claim that an AI or developer independently verified the system.

## What the owner wants

The Thought Processor's workspace should feel like a **Netflix / Amazon Prime Video-style catalog of independently developed capabilities**, not a rigid dashboard.

**Each tile/card represents an actual GitHub-backed application or feature component**, not just an abstract capability icon. As new features are independently built, tested and made available, their compatible **running** instances should be discoverable and their cards should appear **dynamically**, without adding another hard-coded tile to Processor each time.

Examples to explore:
- **Identity:** the **real running** [`Arameair/identity`](https://github.com/Arameair/identity) application, **not** the TASS profile/avatar. Its own UI appears as the live preview when possible.
- **Thinking:** a real Thought Engine instance, including heartbeat and processing activity.
- **Worker:** the currently selected model service, e.g. Chat-Jimmy *for now*; the model/worker can be replaced.
- **Memory / hearing (Discord) / vision (screen):** separately developed and enabled feature services when they actually exist and advertise compatible endpoints.
- New features can be introduced one at a time without breaking running features.

**Identity and Thought Processor are both foundational.** Identity owns entity identity records and feature grants. Processor is the discovery/connection/visibility point. Project Controller is a separate development tool that builds/runs branches and owns its Play/Stop actions.

## Card behavior (design target, not implemented)

| Actual service state | Card presentation | On expand |
| --- | --- | --- |
| Running + verified UI preview supported | Show real live UI thumbnail/viewport and branch/SHA/status | Expand **that same actual application UI** into the main workspace; show diagnostics alongside it |
| Running + healthy but embedding blocked | Honest live state/telemetry preview; say **UI embed unavailable** | Offer opening the actual existing UI in its own view/window, plus diagnostic pane |
| Running but not authorized/compatible | Visible status and reason; no false ready indicator | Read-only safe diagnostics, compatibility/permission details |
| Not running / disconnected | **STOPPED / DISCONNECTED**, with last-check timestamp; never a frozen image labeled live | Explain where to start it in Project Controller; do not silently start it |
| New compatible application discovered | Add a new card from verified discovery metadata | Use generic safe preview/inspector contract; no processor code change required |

The **runtime application instance**, not merely the GitHub repository URL, must be detected. A repository on GitHub is not proof of a running local service. Git branch/build/port/instance should be shown as verifiable metadata where the runtime reports it, not guessed from a name.

### High-level home layout

- A calm, tile-based catalog with **Identity** plainly represented as its own running application card.
- Groupings such as **Foundation / Thinking / Inputs / Communication / Experiments** are suggestions, not fixed product categories.
- Cards with real previews when possible; stopped or unsupported previews clearly marked.
- Select a card to maximize one actual live application while exposing relevant logs, metrics, connection details and **Copy Context**.
- Multiple independent applications can be running, but avoid many active embedded previews overwhelming memory/CPU; preview refresh and focus policy remain to be measured.
- A dedicated **control-flow view** should let the developer inspect connections **between** cards and follow what happened across them, rather than only seeing individual application UIs.

## Roles and ownership — do not blur them

| Component | Responsible for | Not responsible for |
| --- | --- | --- |
| Project Controller | Clone/build/Play/Stop and side-by-side branch testing | Identity grants or Thought Engine's thinking decisions |
| Identity (running app) | Entity IDs, archive/restore, linked repos, future grants, its actual UI | The Processor's card catalog or model heartbeat |
| Thought Processor | Discover compatible instances; connect/route approved integrations; surface verified state, real previews and diagnostics | Duplicate Identity UI; start/stop independent programs; own thought history; decide heartbeat timing |
| Thought Engine | Choose fast-worker heartbeat API calls, processing policies, thoughts and persistence | Service presentation catalog |
| AI worker(s) | Return bounded model results for Engine requests | Entitlements, thought-store authority or ownership of the UI |

**The heartbeat is the API request to the selected fast worker.** Engine controls when/how it happens. Engine discovers its worker through Processor-controlled service discovery and can send subsequent API requests **directly** to the worker. Processor must show credible activity traces from Engine telemetry; it is not automatically in the request data path.

This extends the earlier [discovery/heartbeat direction](https://github.com/Arameair/Thought-Processor/blob/experiment/discovery-routing-devex-notes-20261023/docs/discussions/2026-10-23-routing-heartbeat.md). That discussion is on a separate experiment, **not yet merged to Alpha**.

## Draft extensibility contract (not an implemented API)

Candidate information each compatible runtime could advertise or register:

- **Identity:** service kind, stable instance ID, source repository, branch and exact commit/build, protocol version.
- **Connectivity:** verified endpoint(s), last check, process ownership, readiness, capabilities and authorization state.
- **Presentation:** declared UI preview URL or safe status/activity stream, supported embedding/view modes, optional diagnostics endpoint.
- **Operations:** only explicit supported actions, with permissions and request IDs; no silent launch/stop or remote administration.
- **Tracing:** correlation IDs and timestamps for requests/worker calls, processing stages, error owner, real-vs-simulated evidence and persistence status where relevant.

Discovery alone must not confer permission. A TCP port or self-reported service name is insufficient for trust. Preview origins, sessions and framing restrictions must be honored; do not bypass security policies or copy session cookies between apps. When a service cannot embed, show status with **Open UI**, not a fake live view. Developer logs default to redacted metadata rather than raw private content.

## Developer experience test to aim for

1. Start Identity from its own experimental branch in Project Controller and verify the UI is actually reachable.
2. Start Processor separately, and discover the running Identity instance. Verify the card corresponds to the correct branch/SHA and instance.
3. Watch a **real** preview or a clearly labeled fallback; expand to the *original app UI* and inspect diagnostics.
4. Start a second feature independently. Verify a new matching card appears without editing Processor's hard-coded home screen.
5. Stop/restart the feature independently. Card changes to disconnected and then returns without falsifying liveness or deleting external state.
6. Trigger one genuine Engine → worker heartbeat: locate its request ID and status from a live activity/connection view; follow processing to thought persistence if implemented.
7. Capture **Copy Context** including verified source, selected app, branch/SHA, endpoint, request IDs, failure reason and sanitized logs.
8. Confirm a denied identity cannot enable an unauthorized feature; disconnection does not terminate the running source app.

**All steps NOT RUN as part of these notes.** The proper first prototype is one running Identity application + one real connection/preview, not eight simulated green cards.

## Open UX decisions for next discussion

1. Home layout: one highlighted Identity card first, or equal-sized cards grouped by use?
2. Does selecting a card expand a live preview in place, or fill one shared main viewport with a diagnostic side panel?
3. What should *every* card show in collapsed form: status, branch, last activity, instance ID, a preview still, or an actual refreshed viewport?
4. How should we show **control flow across services**: clickable connection graph, activity timeline, or a combined screen?
5. Where do features appear if installed but disabled, or tested but not authorized for the selected identity?
6. How much UI control should Processor expose vs defer to the original app's own controls?
7. Preview performance and security: when should inactive cards stop live rendering; how is session separation preserved?

Discuss these before starting implementation. Preserve one-feature-at-a-time experiments and owner-approved Alpha/Stable/Main promotion.
