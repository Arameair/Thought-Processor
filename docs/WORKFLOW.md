# Branching, validation and owner promotion

**No automatic promotion.** The owner picks when to keep or discard an experiment. Documented states are not software approval.

## Branch meanings

```text
experiment/new-idea-YYYYMMDD  ─┐
experiment/alternate-YYYYMMDD ─┼── compare evidence → owner selects → alpha
                                │                           |
                                └── never auto-merge        v
                                                stable (approved snapshot)
                                                         |
                                                         v
                                                 main (release approval)
```

- `alpha`: base for **all new experiments**, and active owner-selected integration.
- `experiment/<topic>-YYYYMMDD`: single, explicit hypothesis; latest Alpha SHA pinned in its note. Do not fork randomly from another experiment. If old work is useful, **port it explicitly onto a new branch from Alpha**, preserving attribution.
- `stable`: **snapshot** of what the owner has accepted, never a moving default and never automatically updated.
- `main`: public-facing lane when the owner explicitly approves a release. The repository can remain private regardless of branch name.
- Document archive and rollback refs before advancing an already-tested candidate; avoid unsafe force-updates.

### Historical bootstrap exception for the originally empty repository

Initially `main`, `stable` and `alpha` were initialized at a documentation-only commit. **That is no longer the current Alpha state**: the owner subsequently promoted runnable P0 at [`ae9e96b`](https://github.com/Arameair/Thought-Processor/commit/ae9e96bb6004f64fe11edf40ff32dc1bd6ade47e). `stable` remains the old documentation snapshot and `main` is public-facing documentation. Branch new work from **current Alpha**, not the historic bootstrap commit. Promotion remains manual. See [status](STATUS.md).

## Working on a feature

1. Read `README.md`, `docs/STATUS.md`, `docs/ARCHITECTURE.md` and current issue.
2. Note exact `alpha` SHA; create a fresh experimental branch **from Alpha**.
3. Describe one hypothesis, anticipated behavior, affected service contracts, tests and rollback.
4. Keep worker, Thought Engine, Identity and producer applications independent. No hidden auto-start/stop of other programs.
5. Build, run test suites and a real-system integration scenario; distinguish stand-ins from live services.
6. Use Project Controller Play/Stop and **Copy Context / Copy Logs** where appropriate; verify OS, port, build SHA, source identities, per-request timings, actual data and clean shutdown.
7. Update experiment report and `docs/STATUS.md` in **the experiment branch**. Do not edit Alpha, Stable or Main as a reporting shortcut.
8. Present exact branch/commit, comparisons, failures, user steps and **ask the owner before promoting**. The owner decides keep/revert/critical.
9. After approval, update the authoritative `docs/STATUS.md` as part of the approved branch change; record previous SHA and rollback. No automatic Alpha→Stable→Main merges.

## Required test matrix

| Check | Evidence |
|---|---|
| Build | Go/platform versions; exact branch and SHA; build command; elapsed time |
| Independent lifecycle | Play/Stop only own process; no orphaned processes; no stopping other apps |
| Readiness | Real service identity and protocol, not merely HTML HTTP 200 |
| Feature authorization | Denied identities cannot process; revocation enforced; history retained |
| Feeds | One real authorized log source; pause affects feature subscription, not source |
| Worker | Ready/busy/blocked/uncertain; no duplicate uncertain sends; real-vs-stand-in |
| End-to-end | log → queue → processing → evidence → persistent Thought → live Processor |
| Persistence | Stop/restart, identity isolation, source/worker changes |
| Diagnostics | Real status, request IDs, elapsed timings, errors, truthful zeroes |
| Failure | Unavailable, incompatible or changed dependency → clear fail/no invented state |

For owner acceptance, record a manual test on the owner's actual Project Controller configuration; passing tests in an isolated sandbox alone is not the same.

## Experiment report template

Copy [EXPERIMENT_TEMPLATE.md](EXPERIMENT_TEMPLATE.md) into `docs/experiments/<date>-<topic>.md` in the experiment branch. Newest-first ledger row and issue link are required. Record negatives and unverified cases, not only successes.

## Safety

No promotion to Alpha/Stable/Main without owner approval. No deleting repos or thought data as a side effect of removing an identity or feature. No live code replacement hidden behind a "Connect" action. Keep data directories outside throwaway checkouts and show provenance/retention in logs.
