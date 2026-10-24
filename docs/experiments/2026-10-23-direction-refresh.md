# Experiment report — clarify project direction and documentation

- **Date:** 2026-10-23
- **Type:** Documentation / information-architecture refresh only
- **Branch:** `experiment/project-direction-refresh-20261023`
- **Base:** `alpha` @ `9fac9fa44cb020bb6f8aa6bcb1e9faeda6e0b47a`
- **Implementation:** No executable, runtime integration, UI implementation, or migration
- **Owner promotion:** None requested or performed

## Problem

The main README and Alpha docs describe an earlier Thought Processing control dashboard, but later owner-directed experiments introduced important new direction: a discovery/control point, Engine-owned heartbeat timing and direct worker calls, dynamic verified **application** cards, and a selected Option C lightweight UI. These decisions are spread across unmerged branches, leaving the starting point confusing.

## Scope of documentation changes

- Replace the landing page with a truthful one-minute product explanation, navigation and verified status.
- Document **Option C** without claiming that its live previews/status exist.
- Clarify roles of Project Controller, Identity, source apps, Thought Engine, Processor and replaceable workers.
- Separate Processor-assisted discovery from direct Engine-to-worker calls.
- Preserve AI collaboration attribution from `main` in this Alpha-based experiment (Alpha itself does not yet include it).
- Rebuild Status, Roadmap and Decisions around actual existing branches and the next smallest proof.
- Preserve original experimental notes as **source links**, not erase or promote them.
- Leave standalone Thought, Thought-engine, Thought-Controller, Identity and Chat-Jimmy unchanged.

## Checks and evidence

| Check | Result |
|---|---|
| Review original related design branches | Done (documentary review; see DESIGN_HISTORY) |
| Verify existing branch SHAs | Done before writing, recorded in STATUS |
| README navigation and relative links | Review after commit |
| Status truthfulness and prototype boundaries | Reviewed; all runtime claims marked not built/not verified |
| Program build/test | **NOT RUN — no program code in this change** |
| Project Controller Play/Stop | **NOT RUN — repository remains documentation only** |
| Actual service integrations | **NOT RUN** |

## Design provenance

- [Routing / Engine heartbeat](https://github.com/Arameair/Thought-Processor/blob/experiment/discovery-routing-devex-notes-20261023/docs/discussions/2026-10-23-routing-heartbeat.md)
- [Dynamic feature application cards](https://github.com/Arameair/Thought-Processor/blob/experiment/dynamic-feature-cards-notes-20261023/docs/discussions/2026-10-23-dynamic-project-cards.md)
- [Owner-selected Option C](https://github.com/Arameair/Thought-Processor/blob/experiment/lightweight-live-cards-option-c-20261023/docs/discussions/2026-10-23-option-c-lightweight-status-cards.md)
- [Current AI collaboration policy](https://github.com/Arameair/Thought-Processor/blob/main/docs/AI_COLLABORATION.md)

## Risks and follow-up

- Clarify whether a future Processor is only a Thought Processing feature control plane or a more general *compatible feature-app catalog*. Current writing reflects the latest dynamic-card discussion while limiting the first tests to Thought Processing connections.
- **Canonical Engine remains undecided.** No code or data migration may proceed on documentation assumptions.
- Cross-process identity authentication, worker registry, app preview embedding, and source subscriptions are proposed, not specified.
- Before any Alpha promotion, owner reviews the consolidated wording, especially which component may trigger worker calls and which status-card controls are supported.

## Rollback and decision

The unchanged `alpha` branch @ `9fac9fa44cb020bb6f8aa6bcb1e9faeda6e0b47a` is the rollback point; no data touched. **Decision pending.** The current experimental branch can be revised or discarded without changing Alpha, Stable, Main or any running program.
