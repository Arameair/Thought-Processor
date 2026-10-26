# Cross-repository decisions — October 26, 2026

**Status:** Owner-confirmed **design** direction, coordinated across [Identity](https://github.com/Arameair/identity) and [Thought Processor](https://github.com/Arameair/Thought-Processor). These are **not** claims that secure protocols, lifecycle supervision, feature grants or release installations have been implemented or tested. Related discussion: [Processor design issue #15](https://github.com/Arameair/Thought-Processor/issues/15); [Identity architecture review #14](https://github.com/Arameair/identity/issues/14) and [authorization issue #15](https://github.com/Arameair/identity/issues/15).

## Agreed decisions

| Key | Owner-selected rule | Consequence for first prototype |
| --- | --- | --- |
| **C01 — Human interface and startup** | Thought Processor is the **normal human/administrator system-wide interface**; Identity is the independently running authority. **Either program may start first**. The requirement is a clean end state with both running and **genuinely verified** connection for full administrative operation, not a particular startup order. | Both startup orders should converge to the same identities, trust and displayed states. A missing service is honestly unavailable; discovery/HTTP health alone is **not** a trusted connection. |
| **C02 — Install versus use** | In the **prototype only**, a verified GitHub account's access to the private Thought-engine repository may be used as a **temporary eligibility check to obtain/install the Engine**. **That is not permission to run it for an identity, access sources, read Memory, or communicate data.** Identity separately authorizes identity, feature, source and operation; receiving feature enforces. | Catalog must distinguish eligible-to-download, installed, service-paired, enabled, source-authorized and working. No automatic feed access. Real publisher release authorization, distribution provenance and revocation remain future design. SHA-256 alone does not establish trusted publisher provenance. |
| **C03 — Independent data and least privilege** | An identity begins with **no optional source access**. Independently running features own their execution/data; one installed feature may serve many identities, **without merging their experience**. Connections across identities are separately authorized communications, never implicit shared Memory/Thoughts. | No blanket read of conversations, game events, other features or files. Identity owns grants and metadata, not a universal data pipe. |
| **C04 — Exit and processes** | On an **explicit Thought Processor application exit**, present **Shut down managed workspace** or **Keep services running**, with **shutdown selected by default**. Processes the system starts must be visible, trackable and **cleanly stopped when shutdown is chosen**, with exit confirmation and orphan detection. | Keep **browser tab closure separate from application exit**. Do not terminate external/unowned services without valid authority and consent. Record supervised child processes, ownership and failures; graceful bounded stop, verified exit and truthful “still running” status. Unexpected crashes and OS logoff need their own recovery design. This is a **user-facing lifecycle policy**, not a claim that Engine cognition runs inside Processor; Engine remains independent. |
| **C05 — Pairing and reconnection** | Use a **consistent initial operator-approved pairing flow** across services. Store scoped credentials securely so the same verified peers can **reconnect automatically** after restart; provide explicit **unpair/revoke**. | Discovery is not authentication; pairing is not a feature/source grant. Reject changed/wrong services and revoked/expired credentials; redact secrets. Exact protocol, storage and key rotation are **not yet selected**. |
| **C06 — Identity outage** | **Fail closed for the first prototype.** When Identity cannot verify a required scoped permission, block **new protected actions** and **protected queued work before dispatch**. **No offline grant grace period, cached allow or optimistic permission**. | The program may remain alive for truthful status and safe unprotected work; show AUTHORITY UNAVAILABLE rather than an explicit DENIED. Revalidate after recovery before allowing work. Existing completed evidence need not be deleted. In-flight cancellation details remain open. |
| **C07 — Memory** | **Memory is an entirely separate optional future feature**. It has not been designed or built as part of the current scope. Not required by a valid Identity, Processor, or first prototype. | Do not store or claim working memory in Processor or merge Engine processing state with Memory. No Memory implementation assignment now. |
| **C08 — Project order / Engine selection** | **Identity concept first; Thought Processor product experience now; Thought Engine is the next focused project.** The separate `Thought` and `Thought-engine` implementations remain experiments; the owner has **not selected a permanent canonical implementation or migration**. | An existing Engine branch may serve as an explicitly labelled integration dependency, but do not launch a broad cognition redesign, pick the permanent winner or build Memory during Processor's first-experience work. |
| **C09 — Release process** | Functional experiments start from the owner's `alpha`; **only the owner approves promotion** into Alpha/Stable/Main. Technical tests, fixture demonstrations and design approvals are not interchangeable. | No automatic merge, new release, permission grant or code build is implied by these notes. |

## Distinctions not to lose

1. **System UI vs execution:** Processor orchestrates the user experience and approved controls; Engine performs thinking; Identity decides security; a receiving feature enforces permission.
2. **Lifecycle vs independence:** A coordinated, **consented** shutdown can stop *managed* services, even though their implementations remain separate. Choosing Keep Running is also valid. Processor crashing or a person closing one browser tab is **not** consent to kill unrelated services.
3. **Pairing vs authorization:** Remembering an authenticated peer does **not** authorize new source feeds or data access.
4. **Offline vs running:** A process can still be running while its protected work is blocked because the authority is unavailable.
5. **An identity vs Memory:** Every entity can have an identity without installing Thinking or Memory.

## Historical designs superseded by these decisions

- **“Identity must open first”** and **“Processor must open first”** were both too rigid. Only Processor's role as the **normal interface** is fixed; startup order is flexible.
- **“Processor close always leaves all programs running”** and **“Processor must stop all processes automatically”** are both superseded by an explicit application-exit choice, **default managed shutdown**, clear process ownership, and verification.
- **“Temporary offline allowance / cached grant on outage”** is explicitly **not** part of v1. Existing descriptions of bounded offline leases are research options, not selected policy.
- **“GitHub repo access = feature data permission”** is rejected. Prototype repo visibility is only an acquisition/install eligibility shortcut.
- **“Memory is Processor/Engine built-in state”** is rejected as a product boundary. Memory is future and independent.
- **“Engine candidate is final/canonical”** is not owner-approved. The Engine is the next project phase.

## Still open — don't invent answers

Exact first-run bootstrap and administrative authentication; permission-grant and service-identity wire contracts; precise installer lifecycle, real publisher entitlements and release signature verification; owned-process supervisor/recovery design; managed vs externally started service behavior; OS logoff/crash handling; queued/in-flight revocation; pairing credential format and key protection; which `Thought` implementation becomes canonical; Memory interfaces (deferred); and the first Processor product UX layout/meaningful action ([Issue #15](https://github.com/Arameair/Thought-Processor/issues/15)).

## Next verification gates (not tests performed today)

- Start Identity then Processor **and** Processor then Identity; verify same authenticated system result, real IDs, no duplicate services.
- Pair once, restart and reconnect; revoke; attempt wrong-service pairing; show no protected authorization without an actual grant.
- Install-eligible account must **not** automatically gain source/identity data access; a denied identity/source remains denied.
- Exit Processor → prompt; default shutdown → verify owned process exit, no orphans; Keep Running → independent processes remain visible/manageable; tab close must not silently trigger shutdown.
- Identity unavailable → new protected requests and queued protected dispatch blocked; recovery requires fresh validation.
- Separate identities remain isolated; Memory absent is supported.
- Record branch/full SHA, platform, real-vs-fixture, errors, accepted scopes and an explicit owner verdict before any promotion.

**Decision provenance:** Owner-led live architectural review on October 26, 2026, logged in [Processor design issue #15](https://github.com/Arameair/Thought-Processor/issues/15). Documentation alignment is not a functional implementation.
