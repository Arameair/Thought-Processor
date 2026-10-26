# Thought Processor — foundation experience design workshop

**Date:** October 26, 2026  
**Status:** DISCUSSION DRAFT. No new app code, build, schema, approved UI, grant, release or branch promotion. This is the starting point for owner review, **not** an implementation assignment.

## 1. The objective has changed — don't confuse the prototypes

The October 24–26 **diagnostic shell** was built to get Processor running and test visibility alongside the experimental Identity app. That is useful technical evidence, but **it is not the intended Thought Processor product**.

The next goal is a **small but genuinely interactive version of Thought Processor as the user's main system workspace**. We want to experience the basic layout, navigate a real identity and feature, see what works or is blocked, and then decide what feels good or wrong before building a large platform.

**The minimum useful question:** “Can another person open Thought Processor and understand *which identity they're managing*, *which independent features exist for that identity*, *what permissions are effective*, and *what meaningful action is available next* — without having to operate Identity's developer UI?”

This design exercise is **not** primarily about making the Engine think better. Thought Engine's intelligence and the Spark → Thread → Thought design are separate workstreams.

## 2. Established boundaries — not up for silent reinterpretation

These reflect owner direction, not a claim that code enforces them:

1. **Thought Processor is the normal human/administrator interface for the whole system**, not only the Thought Engine setup page. It is a control/experience plane, not a universal data proxy or database.
2. **Identity is the authoritative identity, relationship, security and permission service**. Processor shows and requests access changes; it cannot mint or silently grant one.
3. **The receiving feature enforces** each applicable Identity decision where work happens. Processor going offline does not erase permission checks or stop independent features.
4. **A new identity has no optional feature or information-source access by default.** Publisher entitlement, installation, connection, enabled state, source permissions, and actual running/processing are separate facts.
5. **One installed feature implementation may serve multiple identities**, but each keeps isolated settings, permissions and experience. No shared Memory/Thoughts between identities without an explicit authorized communication channel.
6. **Independent features own their data and execution** (Engine thinks, Memory remembers if installed, communication tools communicate, source features condense approved observations). Identity governs locations/permissions but not their entire data.
7. **Project Controller is a development tool**, never required for ordinary users. Experiments begin from approved `alpha`; only the owner manually promotes.
8. **Diagnostics report facts**, never a manufactured “connected,” “granted,” or successful thought.

References: [Identity decisions](https://github.com/Arameair/identity/blob/main/docs/DECISIONS.md) · [Oct 25 cross-repo owner alignment](https://github.com/Arameair/Thought-Processor/blob/experiment/dynamic-feature-observation-design-20261025/docs/OWNER_DESIGN_ALIGNMENT_2026-10-25.md).

## 3. The proposed first experience: Profiles → Catalog → Collections

This is a **working design hypothesis to test**, not a final approved layout.

### A. Profiles: “Whose system am I looking at?”

- Show real durable identities provided by Identity, e.g., an existing TASS profile or a business profile. Do not hard-code companion names or create an identity independently of Identity.
- Switching profiles changes only the **authorized view for that identity**: linked features, source access, status and collections. It must not mix the previous profile's information.
- Provide an always-visible selected-profile context and a system/admin settings path, but distinguish **feature/service identities** from ordinary person/business/companion profiles.
- Profile create/edit/archive requests, when implemented, go **through Identity**, not a second Processor-owned registry.

### B. Catalog: “What can this identity actually use?”

- Display **real features** from a verified Identity/feature source: installable, installed, connected, unavailable, disabled. Never imply that a repo being visible grants access.
- Each card has one clear next action appropriate to its *verified* state: inspect; view supported detail; request approval through Identity if implemented; open owning feature's UI; or a truthful “Not available yet” reason.
- Distinguish **publisher download entitlement** from **runtime access for the selected identity** and **approved information sources**. Initially those source scopes are empty.
- Do not auto-connect all installed features or claim Thought Engine is mandatory.

### C. Collections: “What did its features produce?”

- When a feature can provide authorized real output, show it in the selected identity's collection (for example, actual Thoughts from an attached Engine).
- If none exists, show an honest empty state with a next step. **Never seed fake thoughts or logs into the user's live view.**
- Read records from the owning feature; do not duplicate ownership or silently migrate history into Processor.
- An original feature UI may open inside one expanded card if permitted and supported; otherwise show **Open Original UI**. Avoid keeping three competing thought dashboards as normal entry points.

### D. A calm default screen

**Candidate wire journey** (illustrative labels only, not simulated activity):

```text
Thought Processor
Selected profile: [real name / verified Identity ID]        [System / Diagnostics]

[Profiles: Choose or create via Identity] 
[Catalog: real Feature Cards for selected profile]
  [Communication]      [Thought Engine]       [Memory]
   observed state        observed state         observed state
   next real action      next real action       next real action

[Collections: records for this profile, from owning services]
[Selected card inspector: Details | Permissions | Activity | Flow | Logs | Metrics]
```

Developer metrics and per-service ports belong one click away in an inspector, not as the entire homepage. Keep single expanded live preview; use meaningful error and empty states.

## 4. Start small: the first **real** prototype, not a slideshow

**Prototype P0 — Connected workspace and navigation (recommended first build target; owner must approve):**

1. Launch Processor independently on Windows in a fresh data folder; it must not rely on Project Controller.
2. Discover/connect to an **actual** independently running Identity service, with evidence of version and trust state. Until secure pairing exists, label it **UNVERIFIED / READ-ONLY** and prohibit privileged changes. Do not quietly treat a loopback probe as authentication.
3. List **only real** identities and features exposed by a supported, authorized Identity interface. This **may require a small cross-repo Identity interface that does not yet exist**; design and approve the contract *before* implementation. If it is absent, show the accurate blocked state, not a fixture.
4. Select one real identity. Switch to another, and verify the catalog and collections show **separate states**. Return to original and prove persisted selection where safe.
5. Inspect one feature card with **real status and permissions** (including **no allowed sources**) and a clear *next real action*. This is the first owner-facing useful experience; the prototype cannot pass with a screen of only “connected/unverified” badges.
6. Expose truthful request → accepted/refused → verified effect for any permitted action the backend truly implements. **No fake grant request** or local on/off flag described as authorization.
7. Open and close Processor while Identity/other features stay alive. Show sanitized Flow/Logs/Metrics with exact build SHA, failure cause and observed time.

**Prototype P1 — One actual approved action (follow-on gate):** After Identity supports authenticated operator administration, test requesting **one narrowly scoped feature/source permission for the selected identity**, display Identity's actual response, and verify the feature **accepts/denies accordingly**. Test revocation and offline behavior. Do not make P0 falsely claim this exists. This is required before calling the interface a working *permissions manager*.

**Prototype P2 — Optional Thought Engine experience (separate gate):** When an independently runnable Engine and one approved source/worker are ready, view one actual identity-owned Thought and its provenance. This does **not** require solving Attention, Decay, Memory, full cross-feature reasoning, or automatic process orchestration first.

The sequence is proposed to create something the owner can **use and evaluate early**, while keeping security and cognition claims honest.

## 5. Minimum owner acceptance / critique session

For P0, collect these results in the same run, with separate PASS / FAIL / BLOCKED / NOT RUN and exact build evidence:

| Test | Observable result |
| --- | --- |
| First launch, empty profile | Clear first step; no fake entities/features and no secrets exposed |
| Identity unavailable | Processor remains open, says what is unavailable, offers an accurate recovery step |
| Identity found but not authenticated | Read-only/unverified state; **no privileged action** |
| Authenticated Identity (when interface exists) | Correct durable profile IDs, actual feature catalog and version/provenance |
| Two identities | Switching does not leak source permissions, thoughts or feature-specific content |
| Unapproved source | Shows **NO ACCESS**; no implicit collection or Engine intake |
| Action evidence | A real request has the authoritative acceptance/refusal and observed effect; otherwise disabled |
| Processor stop/restart | Persistent safe UI settings, other programs continue |
| Status/diagnostics | Correct errors, source, latency, SHA, redaction, no invented pass |
| Real human test | Owner records what felt clear/confusing, unnecessary clicks, missing next action, and what should change |

**Owner feedback prompts:** “Where did you expect to click?” “Did you know which identity you were editing?” “Was the permission boundary understandable?” “Did any card seem to promise functionality it didn't have?” “Would you want to open this each day?” Do not optimize styling while the first workflow is misleading.

## 6. Known current state and useful prototypes (October 26)

| Branch/issue | Evidence and limitation | How to use it |
| --- | --- | --- |
| [Processor `alpha`](https://github.com/Arameair/Thought-Processor/tree/alpha) | **Documentation-only baseline**, not a runnable Alpha | Baseline for **new experiment branches** |
| [Oct 25 Processor candidate](https://github.com/Arameair/Thought-Processor/tree/alpha-candidate-20261025), [trial #12](https://github.com/Arameair/Thought-Processor/issues/12) | Diagnostic shell ran, detected Identity **unverified**, reports real metrics. Owner run evidence scoped to this branch. | Reuse tested shell mechanisms after review, **not** as the intended product layout |
| [Oct 26 Processor–Engine candidate](https://github.com/Arameair/Thought-Processor/tree/alpha-candidate-20261026) | Adds Engine attachment and observation/Thought paths; runbook and experiment report describe real processes with **test-fixture worker**. Identity runtime grants still absent. | Optional P2 evidence, not proof of end-user UI or real-model cognition |
| [Engine-install pairing experiment](https://github.com/Arameair/Thought-Processor/tree/experiment/install-thought-engine-20261026), [Identity demo](https://github.com/Arameair/identity/blob/experiment/engine-access-pairing-20261026/docs/ENGINE_INSTALL_DEMO_20261026.md) | Temporary repository-visibility + HMAC pairing, tested using simulated GitHub; install is distinct from start and ingest | Future distribution/security discussion only; do not treat as final publisher authorization |
| [Oct 25 owner-alignment design](https://github.com/Arameair/Thought-Processor/blob/experiment/dynamic-feature-observation-design-20261025/docs/OWNER_DESIGN_ALIGNMENT_2026-10-25.md) | No-code owner direction: Profiles/Catalog/Collections; default-deny source access, identity isolation, source-owned Sparks and shared Engine | Design baseline to compare against P0/P1/P2 |

**Documentation collision to address later:** [Main README](https://github.com/Arameair/Thought-Processor/blob/main/README.md), [VISION](https://github.com/Arameair/Thought-Processor/blob/main/docs/VISION.md), [END_USER_DESIGN_NOTES](https://github.com/Arameair/Thought-Processor/blob/main/docs/END_USER_DESIGN_NOTES.md) and [STATUS](https://github.com/Arameair/Thought-Processor/blob/main/docs/STATUS.md) describe older “Engine-only control point / no runnable implementation anywhere” assumptions. Main/Alpha/Stable are *indeed* docs-only, **but runnable experiments now exist**. These documents need a deliberate, consistent public-facing reconciliation after review, not unqualified overwrite of historical plans.

## 7. Design conflicts / decisions to work through, one at a time

**C1 — Settled boundary, documentation must catch up:** Processor is the **main system-wide UI**; Identity issues permission decisions, receiving features enforce. Earlier “Processor is only the Thought UI” and “open Identity first” proposals are historical.

**C2 — First visible workspace:** Is **Profiles → Catalog → Collections** the correct user mental model and page order? What should appear on a new empty installation? What should a business profile show that a companion profile does not?

**C3 — First useful interaction:** For P0, is selecting an existing identity and inspecting real feature availability enough, or should first acceptance require a real narrowly scoped request to Identity? A working permissions toggle **must** await authenticated enforcement.

**C4 — Identity bootstrap dependency:** How does a user open Processor first and securely find/connect/create a durable Identity administrator without circular trust? Which minimum Identity API is required for P0? How will recovery work? Existing Identity's local wizard does not prove this interface.

**C5 — Feature catalog and connections:** Identity-approved registry versus live feature discovery; difference between publisher entitlement, downloaded/installed, linked to this identity, source scopes and actual running state. Which one controls card visibility and “Request” actions?

**C6 — Lifecycle ownership:** Who installs, starts or restarts Identity/Engine independently? Processor should not kill an unrelated program or launch one without separately authorized lifecycle permission. No hidden auto-launch needed for P0.

**C7 — Service trust:** Identity↔Processor pairing and Engine↔Processor attachment currently use differing mechanisms. Define service authentication, key protection, expiry and revocation rather than baking the demo into an unsafe universal protocol.

**C8 — Collections / Thought boundaries:** One trusted source of identity-owned Thought records; no second thought store in Processor. Earlier Spark→Trace→Thought implementation is not the newer source Spark→Thread→Thought direction; don't rewrite old experimental evidence.

**C9 — UI conflict and clutter:** Existing diagnostic shell and Engine card overlap; decide whether details live in a single inspector or original feature UI. Keep the home experience usable for a non-developer.

**C10 — Validation and scope:** One small visible milestone at a time, exact OS and full commit, true authentication/permissions, isolated test profiles, no fake sample activity, repeatable owner feedback. When do we ask the owner to promote a candidate to Alpha?

Record each owner's **chosen answer, alternatives rejected/deferred, reason, needed interfaces, and which acceptance test changes**. Do not mark undecided questions as approved just because a draft recommends an option.

## 8. Deliberately outside this design-build gate

No redesign of the Thought Engine's scoring, waves, Thread algorithm, Attention/Decay or worker provider; no mandatory Memory, Discord, automatic log collection, dynamic coding, hosted publisher entitlement server, arbitrary process supervision, broad multi-machine auth, or production security certification. Any such feature requires a separate, explicit decision and its own evidence gate.

**Handoff principle:** Produce a **small usable Processor**, try it with the owner, and iterate from what felt wrong. Finish that milestone before expanding the architecture into another dozen unfinished features.

**Sources / related issues:** [Processor issue #3](https://github.com/Arameair/Thought-Processor/issues/3) · [feature registration #6](https://github.com/Arameair/Thought-Processor/issues/6) · [authorization plan #8](https://github.com/Arameair/Thought-Processor/issues/8) · [install plan #13](https://github.com/Arameair/Thought-Processor/issues/13) · [recent design critique #14](https://github.com/Arameair/Thought-Processor/issues/14) · [Identity Alpha gate #20](https://github.com/Arameair/identity/issues/20).
