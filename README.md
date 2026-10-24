# Thought Processor

**A control point for independently running AI-companion features.**

Thought Processor is intended to help an owner discover, connect, inspect, and troubleshoot the applications that make up a modular companion. It focuses first on the **Thought Processing** feature: observing how approved logs and events become thoughts, and how the Thought Engine uses a replaceable AI worker.

> **Project status: design/documentation only.** This repository does not yet contain a runnable Thought Processor. The application cards, discovery registry, live previews, permission links, and trace view described below are **planned**, not implemented or verified here.

## The idea in one minute

- **Features are the product.** Identity, Thought Processing, screen observation, communication, and other features should remain independently usable. Enabling one does not silently turn on the others.
- **Thought Processing receives opted-in observations.** Separate programs can supply selected logs/events to the Thought Engine, where evidence may become Sparks, Traces, and Thoughts.
- **The Engine does the thinking.** It decides how and when to process information, makes model calls, and owns thought records. Chat-Jimmy is the current fast-worker experiment, **not a permanent dependency**.
- **Processor is the control point.** It discovers verified running services, coordinates their approved connections, exposes state and diagnostics, and lets an operator follow what actually happened. It does **not** create a competing Thought database or dictate the Engine's thinking schedule.

```text
Independent source features ── approved logs/events ──> Thought Engine
          Identity ───────── feature permissions ────────┘  │
                                                             ├──> replaceable AI worker
Thought Processor ── discovery, approved connections, ──────┘
                    status, controls, trace inspection
```

The Engine may call a discovered worker **directly**; Processor must be able to display reliable Engine-reported activity without pretending it intercepted those calls. Project Controller remains the separate tool for building, playing, and stopping branches.

## Intended operator experience

The selected initial direction is **Option C: lightweight live status cards**. Each card represents a *verified running application* (for example, the actual Identity app or Thought Engine), not an Identity profile or a fake sample tile. Cards show concise current status. Selecting one opens **one original application UI** when safely supported, with a collapsible inspector for logs, metrics, connection flow, and Copy Context. No background browser for every card. If embedding is unavailable, offer an honest **Open Original UI** action.

This is an **owner-selected design direction**, documented in an experiment; it is **not an implemented feature or an approved Alpha release**.

## Read the project

| Start here | Why |
|---|---|
| [Project direction](docs/VISION.md) | The product goal, feature boundaries and user journey |
| [Architecture](docs/ARCHITECTURE.md) | Identity vs Processor vs Engine vs sources vs workers |
| [Experience / control point](docs/CONTROL_POINT.md) | Option C cards, one live preview and debug flow |
| [Current status](docs/STATUS.md) | Exact Stable / Alpha / Main and experimental state |
| [Next milestones](docs/ROADMAP.md) | Small, independently testable integration steps |
| [Decisions and open questions](docs/DECISIONS.md) | What the owner chose vs what remains hypothetical |
| [Development workflow](docs/WORKFLOW.md) | Alpha-first experiments, evidence and manual promotion |
| [Design history](docs/DESIGN_HISTORY.md) | Earlier discussion branches and related prototypes |

Technical detail: [draft interfaces](docs/INTERFACES.md) · [experiment template](docs/EXPERIMENT_TEMPLATE.md).

## Branches and trust

- **`alpha`** — manually selected active candidate; **currently documentation only**.
- **`stable`** — owner-approved snapshot; **currently the initial documentation baseline, not a tested executable**.
- **`experiment/*`** — separate ideas, all starting from Alpha; nothing automatically promoted.
- **`main`** — release/documentation lane, not a substitute for Alpha. Its AI notice was added independently after the initial baseline.

New proposals should be made on an experimental branch. See [status and exact SHAs](docs/STATUS.md); **the owner alone promotes branches**.

## AI collaboration and verification

**This project is being developed through collaboration between its human owner and AI assistants.** The owner directs the vision and decides which changes are accepted. AI has assisted with ideas, architecture, and documentation and may assist with future code and tests. AI-generated work can be wrong; a claim of success requires independent evidence, and a documentation commit is not a working product.

Read the full [AI collaboration and verification policy](docs/AI_COLLABORATION.md).
