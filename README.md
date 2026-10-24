# Thought Processor

**Discovery, connection and operational visibility for an ecosystem of independently running AI capabilities.**

Thought Processor is the proposed lightweight **control point** that helps compatible applications find one another, establish permitted connections and expose what is actually happening. It is designed to work with **Identity**, the system's authority for durable participants, relationships and access—not to replace Identity or run the cognitive Thought Engine.

> **Project status — 24 October 2026:** **Documentation-only foundation. There is no runnable Thought Processor in this repository's main/Alpha/Stable release lanes.** Related Thought Engine, worker and controller prototypes live in separate repositories and are **not** a verified end-to-end integration. See [current state](docs/STATUS.md).

## Why this exists

Our features are independently developed and may run on different machines, with different workers or different lifecycles. A developer needs to know:

- **Which compatible applications exist and which instances are running?**
- **Which identities and permissions authorize their connections?**
- **Where did an observation or worker request go, and where did it fail?**
- **Which exact build was tested, and what evidence supports its status?**

Thought Processor should answer these questions without proxying every worker API request, storing another feature's entire history or taking over another application's process.

## Proposed architecture

```text
                         IDENTITY
             durable entities + feature IDs
                trust, grants, relationships
                            |
                            v
                 THOUGHT PROCESSOR
              discovery + connections
             real app cards + diagnostics
                            |
           verified/authorized service links
                    /                \
           SOURCE FEATURES       THOUGHT ENGINE
         screen · Discord        observations, thoughts,
           game · logs           processing decisions
                                       |
                                       v
                               REPLACEABLE WORKER
                             (Chat-Jimmy candidate)
```

**Control plane is not the data plane.** After authorized discovery, a Thought Engine may call a worker directly. The Engine owns processing and the actual AI-worker **heartbeat/API request**. Thought Processor observes through honest status and correlated traces; it should not inject itself into the hot path without measured justification.

[Read the responsibility model](docs/ARCHITECTURE.md) · [Identity-first model](https://github.com/Arameair/identity/blob/main/docs/IDENTITY_FIRST_MODEL.md)

## Selected design direction — not yet implemented

| Area | Owner-selected approach |
| --- | --- |
| **Discovery** | Independent features request Identity registration; Processor discovers the approved catalog, reconciles live endpoints and supports hybrid same-OS/configured/manual fallback |
| **Trust** | Identity-approved one-time pairing, credential-verified reconnection |
| **Permissions** | Identity grants scoped access; recipient service enforces; no privilege from discovery alone |
| **Outages** | Short-lived preapproved authorization can survive brief Identity downtime; expires safely |
| **Feature adoption** | New capability appears automatically; owner approves identity-specific access |
| **Status** | Important changes pushed; occasional lightweight health checks |
| **Diagnostics** | Logs stay with services; shared request IDs and on-demand safe collection |
| **Versions** | Report real branch/commit/test evidence, never automatically promote |

### Operator interface

The intended UI is a **dynamic catalog of real running applications**, not a fixed set of static feature mocks. Collapsed cards show verified lightweight status. Only the selected card opens that application's actual UI (or a clearly labeled **Open Original UI** fallback if embedding is blocked). A collapsible side inspector offers **Logs, Flow and Metrics**, including a connection map, event timeline and copyable sanitized context.

Feature installations must operate independently even when Processor is offline. **The owner's personal Project Controller is optional development tooling, never an end-user dependency.**

## First planned integration: real Identity grants, observed in Processor

**Owner-selected workflow:** Thought Processor should be the **single operator/testing hub** for the first Identity↔Thought Engine grant integration, while leaving the services independent. **Identity decides; Thought Engine enforces; Thought Processor connects, tests, and explains.**

Start by specifying and testing an actual Identity-owned allow/deny grant, authenticated source and Engine identities, immediate denial after revocation, and fail-closed behavior when Identity is unreachable. Then add a narrow Processor test view that shows real Identity decision IDs and Engine intake receipts. Do **not** treat a local on/off flag or open port as verified permission.

[Detailed integration plan](docs/INTEGRATION_AUTHORIZATION_PLAN.md) · [Tracking issue #8](https://github.com/Arameair/Thought-Processor/issues/8) · [Canonical Engine question #2](https://github.com/Arameair/Thought-Processor/issues/2).

**Status:** Plan only; no working Processor, registered grant protocol or secure end-to-end demonstration yet.

## Research direction and independent critique

Identity can be a durable coordination authority without necessarily improving AI cognition. We are investigating an **Identity–Experience hypothesis**: do stable identity/purpose information and relevant memory improve a companion beyond recent context alone?

**[Read the hypothesis, controlled comparison and Grok's user-supplied critique](docs/IDENTITY_EXPERIENCE_RESEARCH.md).** These are proposals to test, not conclusions. A critique from Grok or another AI is explicitly welcome; no AI output counts as owner approval or verified runtime evidence.

**[Participate in the architecture review — GitHub Issue #7](https://github.com/Arameair/Thought-Processor/issues/7).**

## Start here

| Document | Purpose |
| --- | --- |
| **[Identity–Experience research](docs/IDENTITY_EXPERIENCE_RESEARCH.md)** | Working theory, competing interpretation, Grok critique, falsification test |
| [Architecture and responsibilities](docs/ARCHITECTURE.md) | Identity, Processor, Engine, Memory and worker ownership |
| [Status and evidence](docs/STATUS.md) | Current verification, releases and experiments |
| [Control surface](docs/CONTROL_POINT.md) | Operator actions and diagnostic proof |
| [Interface proposals](docs/INTERFACES.md) | Observation, worker and service contracts still to define |
| [Decision log](docs/DECISIONS.md) | Explicit owner choices versus unresolved questions |
| [Roadmap](docs/ROADMAP.md) | First independently testable integration gates |
| [Branch workflow](docs/WORKFLOW.md) | Alpha-first experiments and manual promotion |
| [AI collaboration](docs/AI_COLLABORATION.md) | Authorship and independent verification policy |

More detailed owner-selected discussion lives on the [six-decision experiment branch](https://github.com/Arameair/Thought-Processor/tree/experiment/six-design-decisions-20261023); that branch's notes do not constitute merged code or a tested integration.

## Development policy

| Branch | Meaning |
| --- | --- |
| `main` | Professional documentation and approved public-facing releases |
| `alpha` | Owner-selected integration baseline |
| `stable` | Owner-accepted verified snapshot (**currently documentation-only**) |
| `experiment/*` | Independent candidates **based on Alpha**; no automatic merge |

All claims of success require **real evidence**. Distinguish discovered, paired, authorized, enabled, reachable, request sent, worker replied and thought persisted. Report real commit SHA, build/test results, branch, runtimes, failures and rollback. Existing [Thought-engine](https://github.com/Arameair/Thought-engine), [Thought](https://github.com/Arameair/Thought), [Thought-Controller](https://github.com/Arameair/Thought-Controller) and [Chat-Jimmy](https://github.com/Arameair/chat-jimmy) remain independent experiments.

---

**AI collaboration disclosure:** The project is human-directed and developed with AI assistants. The owner makes the final decisions. The architecture critique attributed to Grok was supplied by the owner and has not been independently code-verified. [Read the disclosure](docs/AI_COLLABORATION.md).
