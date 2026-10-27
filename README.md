# Thought Processor

**A local-first coordination workspace for identities, independent features, permissions, and observable operations.**

Thought Processor is the application people and administrators use to understand and manage a modular system. It **coordinates** work and presents the next meaningful action; it does not itself grant privileges or perform the AI reasoning. [Identity](https://github.com/Arameair/identity) is the authority for durable identities and permissions. Optional features, including [Thought Engine](https://github.com/Arameair/Thought-engine), own their execution and data.

> **Status · October 27, 2026:** An **owner-promoted runnable P0 Alpha** exists. A more advanced **P1 feature/identity foundation is an unpromoted experiment**, not a production release. Claude reports successful deterministic-fixture integration tests, but the owner has **not yet performed the P1 hands-on acceptance test**. Public `main` is a documentation/portfolio entry point, while `stable` remains an earlier documentation-only snapshot.

## What problem does it solve?

Independent services often have separate identities, connection states, permission rules and logs. Without a common interface, a user cannot easily tell what is installed, whether services are genuinely authenticated, who may access what, or why an action failed. Thought Processor makes those boundaries understandable **without becoming a monolith or a second security authority**.

| Component | Responsibility | Boundary |
| --- | --- | --- |
| **Thought Processor** | Human-facing onboarding, feature library, identity selection, connection/approval workflows, status and safe diagnostics. | Coordinates approved operations, **does not mint grants** or store every feature's data. |
| **[Identity](https://github.com/Arameair/identity)** | Canonical Admin, entity and installed-feature identities; authorization decisions and grant records. | Separate process; a detected service is not automatically trusted. |
| **[Thought Engine](https://github.com/Arameair/Thought-engine)** | General reasoning, observations, thoughts and per-identity cognitive history. | Must use Identity's canonical IDs and enforce grants; Engine-local switches cannot grant access. |
| **[Fast AI Worker](https://github.com/Arameair/fast-ai-worker)** | Intended replaceable processing utility behind Thought Engine. | Has its own installed identity and must validate authorized requests; real integration is pending. |
| **Other independent features** | Specialized functionality, logs and feature-owned data. | Can have their own workers; installation confers **zero** source/cross-feature grants. |

### How access is meant to work

**Discovered ≠ paired ≠ installed ≠ authorized ≠ running.** Processor can help create a disconnected identity with no grants, and that identity can recommend features. Recommendations are **not** permission to install or use them. Reusable grants support autonomous operations for specifically authorized identities and capabilities; Identity and the receiving feature enforce the decision. In the selected v1 policy, protected work fails closed if Identity cannot validate authorization.

## Current software and evidence

| Lane | What exists | What still requires proof |
| --- | --- | --- |
| [`alpha` — P0](https://github.com/Arameair/Thought-Processor/tree/alpha) | **Runnable owner-promoted Go baseline** at [`ae9e96b`](https://github.com/Arameair/Thought-Processor/commit/ae9e96bb6004f64fe11edf40ff32dc1bd6ade47e), paired with Identity P0. | Not the final full-featured end-user product or a secure published release. |
| [P1 cleanup experiment](https://github.com/Arameair/Thought-Processor/tree/experiment/p1-cleanup-claude-20261026) | Identity-first P1 UI, Admin/installation/permission workflow and a **pinned deterministic Python test feature**. [Claude's cleanup report](https://github.com/Arameair/Thought-Processor/blob/experiment/p1-cleanup-claude-20261026/docs/P1/P1_CLEANUP_REPORT_20261026.md) records **81/81 integration** and **15/15 adversarial** checks. | **Owner P1 trial still pending**; only the trusted fixture was installed, **not** Fast AI Worker or Thought Engine. |
| [Earlier Engine connection prototype](https://github.com/Arameair/Thought-Processor/tree/alpha-candidate-20261026) | Engine attach/observation round trip tested with an explicitly labeled **TEST FIXTURE worker**. | Engine's attach key and legacy permissions are **not** Identity grants. |
| `main` (this page) | Public documentation, engineering case study, traceable test evidence and attribution. | No runnable source in this branch. |
| [`stable`](https://github.com/Arameair/Thought-Processor/tree/stable) | Historical documentation baseline. | No accepted runnable Stable snapshot. |

**Known limits:** No hostile-code OS isolation, no production-ready Admin recovery, no verified real publisher provenance, and no real Engine/Worker integration against Identity's receiving-service grants. A test fixture is not real AI. See the [status ledger](docs/STATUS.md) and [security/integration boundaries](docs/ARCHITECTURE.md).

## How to inspect and test

- **Understand the architecture:** [Architecture](docs/ARCHITECTURE.md) · [Owner decisions](docs/DECISIONS.md) · [Current status](docs/STATUS.md).
- **Try the P1 experiment:** [P1 START HERE](https://github.com/Arameair/Thought-Processor/blob/experiment/p1-cleanup-claude-20261026/docs/P1/START_HERE.md) uses **matching Identity and Processor branches**, disposable profiles, real local processes and a clearly labeled deterministic fixture. No Project Controller is required.
- **See the engineering journey:** [Case study](docs/PROJECT_JOURNEY.md) · [Design history](docs/DESIGN_READINESS_2026-10-26.md) · [Current discussion](https://github.com/Arameair/Thought-Processor/issues/20).

## Teamwork, credit and release policy

Project direction and owner approval: [Arameair](https://github.com/Arameair). This is human-directed, AI-assisted engineering. Claude's experimental implementation, Grok's independent QA and other contributors' work are linked to their actual reports and commits; AI assistance does not imply that any AI vendor endorses this project. [Credits and external technology provenance](docs/CREDITS.md) · [AI collaboration policy](docs/AI_COLLABORATION.md).

`alpha` is owner-controlled. All functional changes require an experimental branch, reproducible tests and an explicit owner **KEEP / REVISE / REJECT / promote** decision. No automated promotion to `alpha`, `stable`, or `main`; documentation-only improvements to `main` do not approve code or release binaries. [Workflow](docs/WORKFLOW.md).
