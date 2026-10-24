# Behind the story — responsibility and evidence guide

**Companion reference to [the owner journey](01_OWNER_JOURNEY.md). Design documentation, not an implementation or an API specification.**

The walkthrough describes *what an owner experiences*. This document captures what each part of the system must actually own for the story to be truthful.

## One sentence to protect throughout the design

> **Thought Processor controls the operational setup of thinking. Thought Engine controls the actual act of thinking.**

The phrase is owner-endorsed. The detailed interpretations below are **proposed until challenged and explicitly accepted**.

A second sentence anchors Identity:

> **Identity decides who is trusted and what is permitted. Independent features enforce authorization and own their own work.**

## The participants aren't interchangeable

| Participant | Lasting ownership | In the story, what it supplies | Should NOT do |
| --- | --- | --- | --- |
| **Human operator** | Intent, approval and acceptance decisions | Chooses TASS, requests a connection, authorizes one test, evaluates results | Depend on a UI that silently grants access |
| **Identity** | Durable entities and feature identities, service registration, trusted relationships, grant records, revocation | Authentic service catalog, selected entity, permission decisions and evidence | Process observations or pretend an open port proves trust |
| **Thought Processor** | Its own workspace state, optional saved *proposals* and test recipes | One navigable picture of the system, connection controls, guided tests, failure explanation | Be a required data-path proxy, canonical permission issuer or another thought store |
| **Observation source** | Domain observations, service instance and data it collects | Sends authenticated source events directly to Engine under appropriate permission | Assume it is authorized merely because discovered or installed |
| **Thought Engine** | Its configuration, inputs, queue, processing logic, worker calls, Sparks/Traces/Thoughts, processing receipts | Says what work it accepted, why it waited, what it processed, and what it persisted | Mint an Identity grant for itself or submit unverified claims as genuine source identity |
| **AI worker** | Its own model execution, status and provider constraints | Returns real inference output or a real error/uncertain result to Engine | Set permissions or rewrite Engine history |
| **Optional Memory** | Independently approved experience storage and retrieval | Later may provide recall if installed and granted | Be a prerequisite to registering any identity |

**Important:** The selected entity **TASS** is not the same thing as a source application named TASS, an Engine instance, an Identity login or a worker. The blueprint must let us clearly represent those distinctions.

## What “control” should mean operationally

Processor controls setup through **authenticated requests, explicit receipts and observed state**. It does not need to own every destination setting to control the user workflow.

Think of four moments, each of which may differ:

| Moment | Plain-language question | Authority |
| --- | --- | --- |
| **Proposed** | “What does the owner want connected?” | Owner action and Processor's temporary connection plan |
| **Authorized** | “Is this exact source-to-destination operation permitted for this entity?” | Identity's authenticated decision |
| **Applied** | “Has the owning feature accepted and persisted its configuration?” | Source and/or Engine's effective configuration |
| **Observed** | “Did one real event pass through the intended path and produce an actual outcome?” | Source, Engine, worker and storage receipts |

For the first version, do not design a magically atomic “Connect” action that hides four separate outcomes. The interface may be simple, but errors must reveal **which moment failed**.

Examples:
- If Identity allowed an operation but the Engine is offline: **Authorized; not connected to a running Engine**.
- If Engine accepted an event but no worker is ready: **Queued; not yet processed**.
- If source has permission but subscription is paused: **Authorized; intentionally paused**.
- If a test receiver claims a success but cannot show its own event receipt: **Unverified result**.

### Where settings remain after Processor closes

The proposal is **distributed effective configuration**:

- Identity retains permission records, service registration and trust.
- Source owns the source-side feed/data collection settings.
- Engine owns the active subscription, worker preferences, pause/resume, budget and queue state (as supported).
- Processor retains only UI preferences and **non-authoritative** connection recipes/test history, if the owner later approves saving them.

The exact owner of **cross-service desired configuration plans** is OPEN. This must be resolved before building a declarative reconnection feature. Do not conflate **a saved plan** with **a permission**.

## The status vocabulary must tell the truth

### Avoid one generic “Connected” indicator

| Meaning | What would prove it | Counterexample |
| --- | --- | --- |
| **Discovered** | A candidate endpoint or Identity catalog record exists | A hostile or unrelated service could reuse the address |
| **Authenticated** | Valid peer/service identity verification | A service's self-reported display name is not proof |
| **Authorized** | Current grant decision for this entity/source/destination/action | Generic app pairing doesn't grant TASS data access |
| **Running** | Recent authenticated instance health receipt | An approved service may be offline |
| **Ready** | The requested operation's current preconditions pass | A running Engine may have no worker |
| **Accepted** | Owning feature has issued an event/command receipt | “Accepted” doesn't mean “processed” |
| **Processed** | Engine can show real processing/worker evidence | A queued event is not a thought |
| **Persisted** | Owning feature confirms the specific result is durably saved | UI text is not a storage receipt |
| **Stale / unknown** | Most recent evidence expired or a step couldn't be checked | Last night's green status is not current health |

These are **multiple independent axes**, not a compulsory linear state machine. A source can be authorized while stopped; a worker can be ready while permission is denied.

### Failure and next-action ownership

| What went wrong | What user should see | Where owner is sent |
| --- | --- | --- |
| Identity wasn't found | No verified authority | Connect or start Identity independently |
| An endpoint answered but wasn't authenticated | Discovered, unverified | Verify service identity / owner approval |
| Permission was denied or revoked | Exact denied operation and Identity receipt | Review access in Identity |
| Engine offline or incompatible | Separate runtime or protocol failure | Inspect Engine or open original app |
| Worker missing/mock/refusing | Valid connection, worker unavailable | Check Engine's worker configuration |
| Event queued under permission later revoked | Blocked before dispatch | Inspect permission transition and queued-event result |
| Network timed out mid-send | Outcome uncertain; don't resend automatically | Inspect event/request ID in owning service |
| Processor stopped | No inspection UI; independent services carry on if otherwise authorized | Reopen for current state / trace review |

For the first security experiment, if Identity is unreachable: **no new protected intake and no model dispatch under an unverified decision**. Later bounded offline authorization is intentionally not included in this first story proof.

## Which action happens where?

| User says | Who receives the action | How Processor contributes |
| --- | --- | --- |
| “Show me what's installed and working” | Identity's registered catalog + each service's verified status | Reconcile and present results |
| “May ScreenBridge send observations for TASS?” | Identity | Submit an approved request and display the grant decision |
| “Enable this feed for TASS” | Receiving/owning feature **after** authorization | Show allowed capability, send command, verify state |
| “Process this waiting event once” | Thought Engine | Expose supported control, get receipt, verify outcome |
| “Use another AI worker” | Thought Engine / its chosen worker adapter **subject to grants** | Show actual choices, offer bounded configuration where supported |
| “Why didn't TASS react?” | Source, Identity, Engine, worker and logs—each knows part | Correlate evidence, show first failed or unknown step |
| “Open the actual application's controls” | Application itself | Display genuine UI or open it separately |
| “Delete or recover an Identity” | Identity | Link to the authorized Identity workflow; Processor is not the authority |

## Minimal capability pattern to consider later

Any compatible independent feature should eventually be able to *prove what it is*, *describe supported capabilities*, *report real health/state*, *accept authorized commands with receipts* and *expose permitted bounded evidence*. A thought-specific feature may add queue, thought, event and worker capabilities.

These five ideas are **behavioral expectations**, not a chosen HTTP schema or requirement for every product to implement a complex service SDK. First we should test whether an extremely small interface already provides enough value.

## A scene that can invalidate the boundary

Suppose you leave Processor closed for a full work session.

If an authorized observation cannot reach Thought Engine—or the Engine cannot call its worker—*because Processor wasn't running*, we haven't achieved the independence promised by our architectural principle.

But be precise: under the **first online Identity-check design**, stopping **Identity** can correctly prevent a new protected observation. That is a limitation of authorization policy, not evidence that Processor must be in the inference path.

## Reference material for subsequent blueprints

- [Narrative walkthrough](01_OWNER_JOURNEY.md)
- [Blueprint questions and required evidence](03_BLUEPRINT_HANDOFF.md)
- [Alternative structures to challenge](04_REVIEW_AND_ALTERNATIVES.md)
- [Existing conceptual contracts on main](https://github.com/Arameair/Thought-Processor/blob/main/docs/INTERFACES.md)
- [Identity's current architectural record](https://github.com/Arameair/identity/blob/main/docs/ARCHITECTURE.md)

**Documentation disclosure:** This is AI-assisted proposed design based on owner direction. It contains no working protocol, authentication verification or accepted product blueprint.