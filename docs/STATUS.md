# Thought Processor — current status and branch ledger

**Last reviewed:** 2026-10-23  
**Overall:** **DOCUMENTATION ONLY — NO RUNNABLE IMPLEMENTATION**  
**Source of truth:** This file *plus* the referenced branch heads, GitHub issues, and recorded test results. A status label must be backed by evidence; branch names alone do not prove anything.

## Release lanes

| Lane | What belongs there | Current evidence | Operator verdict |
|---|---|---|---|
| [Main](https://github.com/Arameair/Thought-Processor/tree/main) | Release-facing documentation and only **owner-approved public** releases | Documentation initialization; no binary/runtime | ⚪ Docs only |
| [Stable](https://github.com/Arameair/Thought-Processor/tree/stable) | Last **owner-accepted, verified** snapshot, with exact commit + rollback plan | Initial documentation baseline only | ⚪ **Not** a functional stable release |
| [Alpha](https://github.com/Arameair/Thought-Processor/tree/alpha) | Active owner-selected working integration candidate | Initial documentation baseline only | ⚪ **No** owner-tested program yet |
| [Experiments](https://github.com/Arameair/Thought-Processor/branches) | Isolated tests and competing solutions, each started from Alpha | See experiment ledger below | ⚪ None functionally validated |

**Important:** The stable/alpha branch links are created as part of the documentation bootstrap. If a link is temporarily missing during setup, no program or promotion has taken place.

## What is real, and what is still planned?

| Capability | This repository | Evidence needed to mark working |
|---|---|---|
| Understand role of Processor | **Documented** | Owner review |
| Stable/Alpha/experiment promotion rules | **Documented** | Owner review |
| Executable dashboard | **Not implemented** | Build and Play in Project Controller |
| Thought Engine discovery / health | **Not implemented** | Compatible real service, exact process ID and build |
| Identity authorization and feature enable/disable | **Not implemented** | Actual Identity grant checked and enforced by engine |
| Per-source thought feed controls | **Not implemented** | Real log ingress, pause/disconnect without stopping source |
| Queue/policy operations | **Not implemented** | Command accepted by engine and observable state changed |
| Model-independent worker selection | **Not implemented** | Same experiment run through two provider adapters |
| Sparks/Traces/Thoughts visualization | **Not implemented here** | Live engine read, source and persistence verified |
| Logs, diagnostics and per-request metrics | **Specified, not implemented** | Reproducible request IDs and measured durations |

## Experiment ledger

| Experiment | Based on | Goal | State | Evidence / decision |
|---|---|---|---|---|
| `experiment/processor-first-control-point-20261023` | `alpha` documentation baseline | Define first real control-point MVP before implementation | **Planning only** | See its branch notes and linked issue; no build tests |

Add rows **newest first**. Each row needs **branch link**, origin Alpha SHA, head SHA, owner-relevant difference, exact tests performed, whether run in Project Controller, failures, and explicit decision: keep / revise / reject / promoted. Do not mark **PASS** for code that has never been run.

## External prototypes — related, not built into this repo

- [Thought-engine experiment](https://github.com/Arameair/Thought-engine/tree/experiment/thought-engine-v1-20261023): independently runnable candidate that currently uses Chat-Jimmy; model quality on owner's live setup not established by this repo.
- [Thought experiment](https://github.com/Arameair/Thought/tree/experiment/spark-trace-thought-v1-20261023): separate deterministic prototype and separate storage.
- [Thought-Controller experiment](https://github.com/Arameair/Thought-Controller/tree/experiment/real-thought-dashboard-20261023): existing live-data viewer. No assumed rewrite or rename.
- [Identity](https://github.com/Arameair/identity): independent identity authority. No feature grant handshake wired here.
- [Chat-Jimmy](https://github.com/Arameair/chat-jimmy/tree/experiment/fast-clean-worker-20261023): independent candidate worker, **not a dependency promised for all future versions**.

Do not infer the health, interoperability or permission model of these running programs from repository links alone.

## Required update when anything changes

1. Exact SHA for Alpha, Stable, Main and experiment (including base SHA).
2. What was changed and *why*.
3. Build/test evidence, machine/OS and real-vs-fixture distinctions.
4. Runtime status, connected dependencies, real data observed, persistence check.
5. Known failure modes, rollback, blockers and owner decision.
6. Whether any branch moved or was promoted (normally **no** until owner explicitly approves).

## Status vocabulary

- **DESIGN:** discussed, no executable.
- **IMPLEMENTED / UNTESTED:** code exists, no verification.
- **TESTED WITH FIXTURES:** tests passed using stand-in services, *not* a live integration.
- **LIVE INTEGRATION TESTED:** actual independent programs exchanged real data.
- **OWNER VERIFIED:** owner tested in Project Controller and approved the result.
- **STABLE ACCEPTED:** owner explicitly promoted a reproducible tested SHA.
- **BLOCKED / FAILED:** show exact issue and preserve the failed candidate.

For the first release, **no state beyond DESIGN has been established in this repository**.
