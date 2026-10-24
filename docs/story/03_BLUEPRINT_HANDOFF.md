# From the story to the blueprints

**A design handoff, not a screen specification, task order or implementation authorization.**

Read [the story](01_OWNER_JOURNEY.md) first. This page translates the moments that matter into **questions and observable acceptance criteria**. A designer or engineer should not infer that a feature already exists because it appears in a story.

## What the first blueprint must actually explain

Before drawing detailed interfaces, we should be able to follow one person through:

> **Open Thought Processor → recognize Identity → select an identity or feature → inspect a genuine connection → ask Identity for a narrow permission → have the receiving service apply it → run one authenticated test → see a reliable result or the first failed step → leave and return without breaking independent services.**

That is the core journey. It is a **product narrative**, not a requirement to create ten separate screens.

## Blueprint map: decisions to draw, not just boxes to color

| Story moment | User question the interface must answer | Must get evidence from | Blueprint questions still open |
| --- | --- | --- | --- |
| First open, no Identity | “What is this, and how do I get started without pretending it's already connected?” | Local app state, verified discovery when enabled | How much may be explored unauthenticated? What is the one honest first action? |
| Candidate Identity located | “Did we find the right authority, and is it trusted?” | Identity + secure peer verification | What genuine pairing flow exists? How is owner consent performed? |
| System overview | “What participants and independent features exist?” | Identity registry + live service checks | How do we distinguish entity, feature and process? Is system or selected-workspace view primary? |
| Select TASS | “What can TASS actually do now?” | Identity grants + Engine/source effective state | How to summarize partial availability without a fake overall READY state? |
| Open Thought Engine | “How do I inspect the actual feature?” | Its original UI and verifiable capability/instance metadata | Embedded single preview, new window or explicit Open Original UI fallback? |
| Propose a connection | “What information will flow, and what permission would be needed?” | Source/Engine capability description + Identity policy | What must be approved separately? Where do saved plans live? |
| Request permission | “Who made the authorization decision?” | Identity decision ID and grant version | Does Processor deep-link or request on the owner's behalf? Which confirmations are explicit? |
| Apply configuration | “Did the destination actually accept the requested setting?” | Service command receipt + status read-back | Who stores source subscription preferences? Can setup be rolled back safely? |
| Run guided test | “Did the observation reach the right place and actually get processed?” | Test sender, Identity, Engine, worker, storage | What is the minimum useful test? What if one event doesn't form a Thought? |
| Diagnose failure | “Exactly which step prevented progress?” | Correlated receipts, timestamps and redacted logs | How do we distinguish DENIED, OFFLINE, QUEUED and UNKNOWN? |
| Close and reopen | “Are the services still independent and is this state current?” | Direct service operation + timestamped re-read | What cached information can we show, and for how long? |
| Add a feature / no companion | “Can I manage something that isn't TASS or a thinking service?” | Newly registered feature and actual availability | Can a new compatible feature appear without a custom UI rewrite? |

## Three candidate navigation models (not decided)

### A — Feature catalog first

Open to all discovered compatible applications. Strong for an advanced operator who already knows what each service does. **Weakness:** people may see ten cards and still not know what to accomplish.

### B — Selected identity's operational overview first

Open on TASS, business or another saved entity and display next verified action. Strong for users whose goal is a functioning companion. **Weakness:** forces a companion/entity concept on a first-time installation or on an engineer who wants to inspect Engine itself.

### C — System overview plus contextual workspace (proposed)

Open on a small installation summary with a **last-used or deliberately selected context**, one evidence-based next action and an optional discovery catalog. Enter an **Identity workspace**, **Feature workspace** or **Connection test** without leaving the coherent system context.

This appears to satisfy both classes of user, but we must test it against A/B. It is **not yet an approved navigation blueprint**. The UI designer should identify what C would add in complexity, then propose a cheaper alternative if possible.

## Interaction principles carried into the blueprints

**One credible next action:** On a failed first-run, the primary action should be “Verify Identity,” not “Run Thought Test.” When a test fails authorization, the action should be “Review authorization,” not “Restart everything.” When the path is healthy, the action might be “Inspect activity.”

**Progressive disclosure:** Summary first, single selected genuine app preview, and inspectable Logs / Flow / Metrics on demand. Do not render every log, permission and setting on the landing page.

**One source of truth for each operation:** A Processor button invokes the owner service's supported action. The UI displays *requested*, *accepted/refused* and *observed effect* separately. A stale status must look stale.

**Accessible to non-experts:** Say “Permission not verified” before displaying a grant identifier. Keep the identifier and full trace one level deeper for developers.

**No hidden process control:** Processor cannot silently launch, reinstall, kill or replace an independently running application. If future lifecycle capabilities exist they must be separately authorized, explained and evidenced.

**No permanent central relay:** A guided test may coordinate command requests, but ordinary source→Engine and Engine→worker traffic goes directly through authorized service paths.

**No duplicate authority:** Processor never creates the authoritative Identity grant or maintains a second thought database.

## Questions to resolve BEFORE detailed UI blueprints

1. **What is the primary object of the workspace?** Is it a whole installation, an entity like TASS, or a running feature? How do we keep transitions obvious?
2. **Who owns a desired cross-feature connection recipe?** What can Processor remember without becoming a fragile source of truth?
3. **What does “connected” mean to the owner?** Which distinct badges/states survive when identity, permission, runtime and worker status disagree?
4. **Which application is allowed to apply subscription configuration?** Engine, source or both? How do we prevent simultaneous contradictory settings?
5. **What must be proven before a feature has a live preview?** Verified UI origin, embedding restrictions, trust and safe fallback.
6. **How does an operator authenticate in Processor?** A convenient UI must not become a bypass around Identity's administrative consent.
7. **Does a first-time user need to know about Spark/Trace/Thought?** Could “an observation was received and processed” be the top-level experience with internals in Flow?
8. **What are the minimum useful outcomes?** “Queued” is different from “Worker answered” and “Thought saved.” What is a fair test result?
9. **How will someone troubleshoot with another AI?** Copy Context with redacted evidence, request IDs, version/branch SHAs and timing, avoiding secrets by default.
10. **How do independent feature identities appear without special-case designs?** What must a future Memory or business utility declare to create a useful card?

## Evidence required for the FIRST *implementation* blueprint

The narrative is adequate for early wireframes and questions. A buildable integration schematic needs concrete verified answers not yet supplied by this repo:

| Requirement | Current state |
| --- | --- |
| Service identity, peer authentication and admin approval method | **OPEN** |
| Minimum current Identity grant check accepted by Engine | **OPEN / not implemented** |
| Source→Engine authentic event envelope and status receipts | Existing experimental event API, but **not authenticated against Identity** |
| Selected first Engine target and its exact branch/SHA | `Thought-engine` sensible candidate; owner must choose for this test |
| Engine command/report capabilities and ownership | Existing experimental pieces; versioned shared contract **OPEN** |
| Correlated evidence across source/Identity/Engine/worker | **Not proven end-to-end** |
| Processor working control UI | **Not implemented** |

Do not mark a prerequisite as DONE because a document proposes it.

## Smallest proposed end-to-end proof

Use *one* entity (TASS as example), *one* Engine candidate, *one* independently authenticated Test Sender and *one* operation (`thought.observation.ingest`).

Show **default deny → approved allow → one directly submitted event → unique receipt → revoke → denied next event → Identity-unavailable fail closed**.

If a real worker is connected, separately show **worker called / completed / data saved** with independent receipts. A denied event must not generate a worker call. Closing Processor must not interrupt already approved direct operations while Identity and the Engine remain available.

This is the **technical research acceptance story**, not a claim that any present feature meets the test.

## Review rubric for the owner

For each future screen or interaction, ask:

- Does the user know what they are looking at and what action is available **right now**?
- Does the UI show the true source and freshness of its evidence?
- Does this interaction belong to Processor, or are we stealing work from Identity/Engine?
- If Processor crashes here, what independent service behavior survives?
- Can a newly built compatible feature reuse this design without new one-off UI code?
- Would doing the same task using only Identity + Engine be **simpler** without losing essential function?

A blueprint should record unanswered questions beside the corresponding scene instead of silently assuming a preferred design.

## Next output after story approval

When the owner is satisfied with the narrative, produce three artifacts **in this order**:

1. **Information architecture + main user journeys** (installation, selected identity, independent feature, connection test).
2. **Low-fidelity workflow schematics** (authorization request and denial, test lifecycle, outage/recovery); clearly mark unimplemented interfaces.
3. **Clickable UI prototype** tied to evidence models. Use mocked fixtures only if conspicuously identified; never confuse a prototype with a live operational product.

No coding, automatic promotion or feature acceptance is requested by this story document.

Related: [responsibility model](02_RESPONSIBILITIES_AND_STATES.md) · [critical review](04_REVIEW_AND_ALTERNATIVES.md) · [current integration plan on main](https://github.com/Arameair/Thought-Processor/blob/main/docs/INTEGRATION_AUTHORIZATION_PLAN.md).