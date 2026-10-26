# GPT-6 frictionless onboarding — evidence, attribution and time ledger

**Status:** auditable implementation evidence as of 2026-10-26. All times **UTC**. This branch is an **independent GPT-6 UX and reliability candidate**, not an independent greenfield backend rewrite. Honest reuse is part of the development-efficiency comparison.

## Starting point, provenance and comparison fairness

- **Primary repository:** `Arameair/Thought-Processor`.
- **Owner-controlled start:** `alpha`, full SHA `9fac9fa44cb020bb6f8aa6bcb1e9faeda6e0b47a`; it contains docs, not the runnable Go application.
- **New experiment:** `experiment/gpt6-frictionless-onboarding-20261026`. Initial branch commit `a86efcd98c01387cca7cfa9de6be3c42b249fa69` is a **direct child of alpha** and explicitly imports the prior GPT-6 P0 implementation source `260c90556d3eeea5c4a19733cc0d1813c803c863`.
- **Independent from Claude:** No Claude-specific experimental branch or new Claude UI was imported. The revised three-stage experience, automatic locator scan, error mapping, recovery, Windows open-browser behavior and acceptance expansion were authored on this new GPT-6 experiment.
- **Related prior GPT-6 reference:** `experiment/p0-reusable-feature-onboarding-20261026`. That branch contributed the Go diagnostic shell, authenticated API v1 client, generic installer, fixtures and initial real-process smoke. Its prior development and CI time **must not be counted as new greenfield work in this experiment**.
- **Identity dependency:** `Arameair/identity` `alpha-candidate-20261026`, API `identity/1-alpha`, exact test checkout SHA `dfbb24e9999b7f9bbb252cff82702b18f9b3e4c1`. No Identity code or branches modified by this experiment.
- **Thought Engine:** `Arameair/Thought-engine`, independent private repository; no actual release published during checked review. No Engine code modified or release/tag created.

## Verified sequence of events

| UTC (2026-10-26) | Evidence | Classification |
| --- | --- | --- |
| **12:25:35** | [First branch commit](https://github.com/Arameair/Thought-Processor/commit/a86efcd98c01387cca7cfa9de6be3c42b249fa69); direct descendant of alpha | **Earliest auditable implementation start**. Discussion/tool inspections preceded it but have no verified individual timestamp logged here |
| **12:30:54** | [Independent guided HTML layout commit](https://github.com/Arameair/Thought-Processor/commit/459ba096f0c7bae0770df7366844189669e1cdc2) | First committed new UX |
| **12:37:13** | [First workflow-enabled candidate commit](https://github.com/Arameair/Thought-Processor/commit/0cc134fda3488fe707a7ac25e955c18806dff99c) | CI checkpoint |
| **12:37:42.370** | [First CI run `40323934333`](https://github.com/Arameair/Thought-Processor/actions/runs/40323934333), real-process job log | **First successful real-process Processor–Identity pairing/restart/outage smoke**. Earlier version still required manually setting custom Identity port |
| **12:37:44.769** | Same CI Linux job log, `ok github.com/Arameair/Thought-Processor/processor` | **First observed automated test PASS** (Linux) |
| **12:38:09.093** | Same CI Windows job log, `ok …/processor` | **First observed Windows automated test PASS** |
| **12:38:12.617** | Same CI Windows job log, `GPT6-Thought-Processor-Windows` artifact successfully finalized (artifact 12650631162) | **First independently verifiable successful Windows executable build artifact** |
| **12:39:12–12:40:22** | [CI run `40324184247`](https://github.com/Arameair/Thought-Processor/actions/runs/40324184247) | **FAIL** — expanded smoke exposed a bad pairing-code HTTP error mapping |
| **12:41:00** | [Fix commit](https://github.com/Arameair/Thought-Processor/commit/d088569676371812d4eb6070fe077691151efc0d) | Pairing rejection correctly mapped as an input-specific failure, not an Identity service outage |
| **12:41:02–12:42:07** | [CI run `40324416560`](https://github.com/Arameair/Thought-Processor/actions/runs/40324416560) | **PASS** — Linux/Windows Go tests and expanded real-process integration |
| **12:42:57** | [Server-side status classification fix](https://github.com/Arameair/Thought-Processor/commit/2ae2d8d2236c1f31b5edcaec3c55959354497b5e) | Preserves wrong-service/removed status after signed endpoint failures |
| **12:42:59–12:44:09** | [CI run `40324662626`](https://github.com/Arameair/Thought-Processor/actions/runs/40324662626) | **PASS** — Windows and Linux build/test, expanded real-process smoke |

**Last completed code-and-test milestone before this document:** all 3 jobs of `40324662626` green. The final docs commit will trigger an additional run; use its exact SHA/run to judge the delivered state, not assume historic CI applies to later code.

### Run time versus active time

- **Wall-clock interval** from first auditable commit 12:25:35 to the 12:44:09 passing CI endpoint: 18 minutes 34 seconds. This **includes implementation and external CI wait** and **excludes prior GPT-6 P0 development**.
- **Active typing/implementation time:** **NOT MEASURED**. Commit-to-commit wall time is not the same as active engineering time.
- **External waiting time:** **NOT MEASURED AS AN ISOLATED COMPONENT**. GitHub Actions run timestamps are known, but those jobs overlap with development activity.
- **Failed/successful CI:** first five completed runs by 12:44:09 were **4 PASS / 1 FAIL**. Later run count, if any, must be updated from the final branch's GitHub Actions history.
- **First real feature installation time:** **NOT AVAILABLE / BLOCKED**. No approved published Engine release was found, so no genuine download-and-install run has been claimed.

## What's new vs imported

| Component | Origin | Evidence |
| --- | --- | --- |
| Local Go server, session/cookie/CSRF/Host protections, diagnostic logs/flow/metrics | Preexisting GPT-6 P0 backend, transplanted from its specific source commit | Acknowledged reuse, not original to this timed race |
| Generic per-feature signed Identity client and transactional hash-verified installer | Preexisting GPT-6 P0 backend | Current API contract preserved, no Engine-specific HTTP routes in new onboarding path |
| `welcome.html / welcome.css / welcome.js` guided journey | **New GPT-6 proposal and implementation** | Connect → Identity → Features; step-specific pages and contextual feature cards |
| Automatic Identity candidate discovery | **New** | Scan per-user locators, default port; verify authenticated peer after pairing |
| Pairing errors/recovery and signed failure classification | **New** | Invalid code mapped correctly; explicit local forget, no remote revocation claimed |
| Windows auto-open local browser | **New** | Only Processor's own UI; no Identity/Engine auto-launch |
| Developer Mode tabs and safe structural Copy report | **New UX** using preexisting read-only APIs | No tokens or code in copied overview; logs visible but not copied |
| GitHub Actions downloadable Windows binary | **New** | Workflow upload-artifact job; artifact is **not** a signed release |
| Expanded Linux real-process smoke | **New cases** atop reused first smoke | Both startup orders, auto discovery, invalid pairing input, second independent pairing + unpair, process ownership |

## Validation methodology

- **Go tests:** fixture-backed signature/wrong-installation rejection, install receipt durability, changed tag, digest mismatch, truncated downloads and protection of previously installed bytes. Fixtures do **not** prove GitHub access or published release authenticity.
- **Real process smoke:** builds actual Processor + Identity Go programs at pinned SHAs; runs both on localhost with isolated profile directories; exercises pair-request code shown in *real Identity*, signed status, canonical `ent_…` identity CRUD, restart, and outage. Uses no simulated Identity process, no GitHub provider and no actual Engine release.
- **Frontend:** `node --check web/welcome.js` and static selector cross-check. **No automated browser screenshot / real Windows interaction test** has been performed.
- **OS:** Windows CI builds and Go tests; Linux CI builds/tests and real-process smoke. **Windows owner runtime: NOT RUN.**
- **Security:** all protected operations remain blocked when signed verification fails. Live private-feature API confidentiality and publisher signing remain **BLOCKED** by upstream contract.

## Source links

- [Workflow](https://github.com/Arameair/Thought-Processor/actions/workflows/p0.yml)
- [Owner acceptance tracker](https://github.com/Arameair/Thought-Processor/issues/17) for earlier P0 baseline
- [Design/owner constraints](https://github.com/Arameair/Thought-Processor/issues/15)
- [Integration contract](https://github.com/Arameair/Thought-Processor/issues/16)
- [Identity API](https://github.com/Arameair/identity/blob/alpha-candidate-20261026/docs/API_V1.md)
- [Windows guide](GPT6_WINDOWS_FIRST_RUN_2026-10-26.md)

**Assessment:** experiment is suitable for the owner's first trial and comparative UX review; **REVISE before Alpha promotion** until Windows owner test, private feature discovery, publisher artifact and provenance blockers are closed.
