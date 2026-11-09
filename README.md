# Thought Processor — local-first companion workspace

Thought Processor is the human-facing workspace for an experimental AI companion. It makes actions, permissions, recorded thoughts, and evidence inspectable, so the owner can **teach, supervise, and evaluate** the system instead of trusting model output alone.

> **Project status — November 9, 2026:** Active experimental development, **not a public release**. Public `main` is a documentation/portfolio entry point, **not** the runnable TASS installation. Go-based integration lives on development branches and draft pull requests. Owner-host acceptance, security hardening and reproducible public installation remain incomplete.

## What is real, and what is not yet proven

| Capability | Evidence and limitation |
| --- | --- |
| Local conversation | Real Nemotron chat has been exercised on the owner's development machine; this is not an independently reproducible public release. |
| Cognition | Engine-backed Thought and cycle history can be inspected. Read-only playback **does not execute thinking**. |
| Manual nudge drafts | Local persistence, authorization checks and zero-model-call behavior have independent test evidence ([workbench audit #99](https://github.com/Arameair/Thought-Processor/pull/99)). Drafting alone is not processing. |
| Nudge processing | A narrowly supervised design passed review; **implementation is incomplete and runtime acceptance is NOT RUN** ([Stage 2 tracker #100](https://github.com/Arameair/Thought-Processor/issues/100)). |
| Skill execution and Brain Builder | Experimental; changing a visual graph is not yet verified to change real authorized execution ([acceptance #89](https://github.com/Arameair/Thought-Processor/issues/89)). |
| Upgrade and release | Safe owner-host backup/restore and release verification remain open ([#79](https://github.com/Arameair/Thought-Processor/issues/79)). |

**Next milestone:** Inspect one persistent Thought → write a nudge to TASS → explicitly authorize one model-assisted development → compare the real before/after Thought and correlated receipt. A fixture-model integration pass and a separately approved real-model usefulness demonstration are **different milestones**. See [#100](https://github.com/Arameair/Thought-Processor/issues/100).

## Current engineering references

- [Pinned Go integration and status index](https://github.com/Arameair/Thought-Processor/blob/af59e7d9213170669eddd4f269ba1f1b50c427e9/core/docs/CURRENT_GO_STACK_20261106.md) — exact source revisions, component responsibilities, tested/untested boundaries and historical/prototype distinction.
- [Earlier integration candidate PR #88](https://github.com/Arameair/Thought-Processor/pull/88) — reported isolated and fixture-model evidence for its pinned version; **not** the complete November 8 nudge-processing implementation.
- [Owner acceptance #89](https://github.com/Arameair/Thought-Processor/issues/89) — proposed authorized execution and measured graph A/B; [recovery #79](https://github.com/Arameair/Thought-Processor/issues/79) remains a separate release gate.
- [Portability review #90](https://github.com/Arameair/Thought-Processor/issues/90) and [C1–C10 conflict cleanup PR #91](https://github.com/Arameair/Thought-Processor/pull/91) — open issues, not completed fixes.
- [Cognition #82](https://github.com/Arameair/Thought-Processor/issues/82) and [Identity authority](https://github.com/Arameair/identity). Other companion repositories may be private and not accessible to public visitors.

## Component boundaries

| Area | Responsibility |
| --- | --- |
| **Identity** | Canonical identity, authorization, ownership and revocation, independently enforced at each service |
| **Thought Engine** | Local model-backed thought cycles, persistent Thoughts/Traces, predictions, actual outcome journal and work pulse |
| **Thought Processor / TASS Core** | One owner/admin interface, supervised learning, approved skills, safe update and orchestration |
| **Development** | Reviewed ideas/experiments and comparison of prospective capabilities; current integrated /dev is in Core |
| **Cognition** | Engine-backed, read-only thought library, timeline and sanitized evidence (part of Core, not another service) |
| **TASS** | Entity-specific descriptive mind and approved lessons; not a second security authority |

## Historical experiments and honest limits

The October 2026 P0/P1 Identity pairing, Python fixture integrations, independent Thought Controller, Chat-Jimmy, standalone Workers and older launchers were legitimate experiments for their specific branches, but **must not be presented as current one-app instructions**. Their original source, test results and audit history remain in Git. No previous approval/CI result is automatically applicable to this newer pinned trio.

The Go candidates are **local-first and currently Docker/WSL-oriented**, with configurable installation roots, host ports, model and GitHub settings. Full cross-identity isolation, authenticated mobile access and reproducible upgrades still require proof. A browser token on localhost is not sufficient authorization for a phone or Internet-facing API.

**Release policy:** Source-built experiments and documentation-only changes do not merge, install, enable autonomy, approve lessons or enlarge grants without a separate owner decision.
