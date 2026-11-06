# Thought Processor — modular, local-first agentic harness

**Status updated November 6, 2026:** Public `main` is a documentation/portfolio branch, **not** the current runnable TASS deployment. The newer **experimental Go integration** joins Thought-Processor Core, Thought Engine, Identity, Development and a Core-hosted Cognition read-only view. It is **not** released, merged to a stable lane or fully accepted on owner hardware.

## Current engineering references

- [Pinned Go integration and status index](https://github.com/Arameair/Thought-Processor/blob/af59e7d9213170669eddd4f269ba1f1b50c427e9/core/docs/CURRENT_GO_STACK_20261106.md) — exact source revisions, component responsibilities, tested/untested boundaries and historical/prototype distinction.
- [Integrated draft PR #88](https://github.com/Arameair/Thought-Processor/pull/88) — current candidate; reported isolated and fixture model tests, **not** real owner Nemotron/deployment acceptance.
- [Owner acceptance checklist #89](https://github.com/Arameair/Thought-Processor/issues/89) — one real authorized execution → Cognition evidence → measured graph A/B; [recovery #79](https://github.com/Arameair/Thought-Processor/issues/79) must be independently verified before install.
- [Portability review #90](https://github.com/Arameair/Thought-Processor/issues/90) and [C1–C10 conflict cleanup PR #91](https://github.com/Arameair/Thought-Processor/pull/91) — open issues, not completed fixes.
- [Cognition #82](https://github.com/Arameair/Thought-Processor/issues/82), [TASS live knowledge](https://github.com/Arameair/TASS/tree/live), [Development prototype](https://github.com/Arameair/Development/pull/2), and [Identity authority](https://github.com/Arameair/identity).

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

The Go candidate is **local-first and currently Docker/WSL-oriented**, with configurable installation roots, host ports, model and GitHub settings. Full cross-identity isolation, authenticated mobile access and reproducible upgrades still require proof. A browser token on localhost is not sufficient authorization for a phone or Internet-facing API.

**Release policy:** Source-built experiments and documentation-only changes do not merge, install, enable autonomy, approve lessons or enlarge grants without a separate owner decision.
