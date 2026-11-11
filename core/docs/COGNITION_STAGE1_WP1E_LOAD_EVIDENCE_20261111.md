# Stage 1 WP1-E — Bounded-load evidence and execution blocker (2026-11-11)

**Verdict: BLOCKED / NOT RUN for the three required load scenarios.** This report is an evidence inventory and reproducibility handoff, **not** a PASS, not a characterization of 100 Thoughts, and not owner acceptance. Do not infer measured response sizes or threshold status from its presence. No production behavior, install, vault, grant, model or source code was modified.

**Authority:** [Stage 1 plan #109, WP1-E](https://github.com/Arameair/Thought-Processor/pull/109), [Slice 2 #113](https://github.com/Arameair/Thought-Processor/pull/113), [audited repair #114](https://github.com/Arameair/Thought-Processor/pull/114), [workflow #111](https://github.com/Arameair/Thought-Processor/issues/111). Delta-only: previous release tests and the Slice 2 audit-log measurement are **referenced**, not claimed as new runs.

## Immutable baseline and isolation

| Component | Exact required SHA | Verified in GitHub |
| --- | --- | --- |
| Core | `2ed99269a1ef8da96319d2c776ba0651b71633ca` | Yes; this report branch starts from it |
| Engine | `40d4e91e3b04196fa4e4f55a92b25789b82d8675` | Yes; no checkout/binary here |
| Identity | `cfc4079b690e2f7dc57afb9f14c6832ae38fdf54` | Yes; no checkout/binary here |

**Environment actually available to GPT-6:** isolated Linux x86_64 test container, Go 1.23.2, Python 3.13.5, Git 2.47.3; GitHub connector can read the three pinned repositories and write this documentation. No TASS source tree, engine/core/identity binary, test fixture or owner data is mounted in the container. Shell `git ls-remote https://github.com/Arameair/Thought-Processor.git HEAD` failed with `Could not resolve host: github.com`. There is no container network path to clone/build the three pinned revisions; no downloadable workflow artifacts were returned for their checked commits. **No native service was started, so there are no new HTTP/latency/audit-log measurements.**

Inspection was through pinned GitHub files:
- [Existing measurement script](https://github.com/Arameair/Thought-Processor/blob/2ed99269a1ef8da96319d2c776ba0651b71633ca/core/test/cognition_refresh_load_native.py): creates one Engine Thought through a fixture conversation/events; measures Identity decisions for library, pulse, detail and locked reads. It **does not** seed 1/50/100 Thoughts with 50 Notes each or capture raw Engine entity bytes/p95.
- [Isolated native harness](https://github.com/Arameair/Thought-Processor/blob/2ed99269a1ef8da96319d2c776ba0651b71633ca/core/test/nudge_process_native.py): isolated, preflighted port base; clean environment, disposable `APP_HOME`, fixture model, owner-port sentinels.
- [Core Engine client](https://github.com/Arameair/Thought-Processor/blob/2ed99269a1ef8da96319d2c776ba0651b71633ca/core/engine/cognition.go): reads the *full* Engine entity JSON through a hard **6 MiB** byte cap before projecting the library.
- [Engine `Entity` response](https://github.com/Arameair/Thought-engine/blob/40d4e91e3b04196fa4e4f55a92b25789b82d8675/server.go) contains full Thoughts, Notes, history, plus other entity fields. Engine default `MaxOpenThoughts=50` ([engine.go](https://github.com/Arameair/Thought-engine/blob/40d4e91e3b04196fa4e4f55a92b25789b82d8675/engine.go)); this restricts **open** Thoughts, not necessarily total retained Thoughts. A 100-Thought dataset must declare its state mix and not pretend to represent 100 concurrently open Thoughts.

## WP1-E required measurements — actual results

**Dataset target:** Notes per Thought = 50; tests must capture true raw HTTP byte counts from Engine and separately the Core Cognition library projection. None was run here.

| Scenario | Thoughts | Notes | Raw Engine entity bytes | Library bytes | Library latency median/p95 | HTTP statuses/failures | Measured % of 6 MiB |
| --- | ---: | ---: | ---: | ---: | --- | --- | --- |
| Small | 1 | 50 | NOT RUN | NOT RUN | NOT RUN | NOT RUN | NOT RUN |
| Medium | 50 | 2,500 | NOT RUN | NOT RUN | NOT RUN | NOT RUN | NOT RUN |
| Large | 100 | 5,000 | NOT RUN | NOT RUN | NOT RUN | NOT RUN | NOT RUN |

**Known bound (source checked; arithmetic checked):** `6 << 20` = **6,291,456 bytes**, and **50% = 3,145,728 bytes**. The proposal condition is **strictly greater than 3,145,728 bytes for the measured 100 × 50 raw Engine entity response**. Its value is unknown. **No Engine summary-endpoint issue was opened**, because the trigger has not been observed; do not raise the limit or implement an endpoint in WP1-E. The smaller library projection cannot establish whether the *upstream* raw response exceeds this limit.

Memory/CPU measurements: **NOT RUN**. No reliable service process or PID was available to sample.

## Retained evidence: Identity audit growth (NOT newly measured)

[PR #113's native measurement](https://github.com/Arameair/Thought-Processor/pull/113) used real Core/Engine/Identity, isolated ports and fixture model at its pre-repair Core revision `8d7be6c9f8cfc2c5e97aa96a4ead2a03b21e4b8e` (Engine and Identity pins unchanged). PR #114 explicitly did **not** rerun Identity load, because the changed code was nudge read ordering. The existing measurement script defaults to `LOAD_READS=20` for each successful read class and five locked requests; its prior published aggregates are:

| Request/class | Prior result at HTTP 200, or stated condition | WP1-E rerun on exact baseline |
| --- | --- | --- |
| Library GET | **4** new decision lines, **1,079 B** per request | NOT RUN |
| Pulse GET | **4** new decision lines, **1,080 B** per request | NOT RUN |
| Thought detail GET | **4** new decision lines, **1,080 B** per request | NOT RUN |
| Library/pulse GET when LOCKED, HTTP 403 | **0** new lines | NOT RUN |
| 60 s with no Cognition read | **0** background decision lines in the previous test | NOT RUN |
| Hidden tab, actual `decisions.jsonl` delta | Not measured by the previous Identity script | NOT RUN |

**Cadence extrapolation, not a new soak test:** visible tab: library GET every 60 s = 60/h, pulse GET every 10 s = 360/h. Prior rate therefore implies `(60 + 360) × 4 = 1,680` decision lines/h and `60 × 1,079 + 360 × 1,080 = 453,540` bytes/h (442.9 KiB/h). Pulse-only would be 1,440 lines/h; new library refresh adds 240 lines/h, **+16.7% relative to pulse-only** (14.3% of combined lines). Continuously visible for 24 h would imply 40,320 lines / ~10.38 MiB. Real byte volume varies with record lengths. PR #113's browser fixture verified hidden-tab polling pauses, but **that is not a measured hidden-tab log-growth observation**.

## Safety and functional contract

| Requirement / evidence | Status |
| --- | --- |
| All three pinned GitHub source revisions inspected | **PASS** (source availability, not executable verification) |
| Existing native harness and WP1-D measurement script examined | **PASS** |
| Scenario generation: 1 / 50 / 100 Thoughts × 50 Notes | **NOT RUN** |
| Raw Engine size; library projection; latency median and p95; HTTP status; 6 MiB and 50% threshold | **NOT RUN** |
| Current-baseline Identity decisions/read, pulse/read, hidden-tab growth | **NOT RUN**; prior PR #113 results retained above |
| Identity authorization and synthetic owner scoping under load | **NOT RUN**; unchanged contracts inspected, no new load evidence |
| No Note text in library cards at the three loads | **NOT RUN**; previously tested by Slice 1/2 fixtures, not this dataset |
| GET-only polling and no action/nudge/lesson/model execution at load | **NOT RUN**; prior Slice 2 fixture evidence only |
| Memory and CPU sampling | **NOT RUN** |
| No contact with or modification of installed TASS / real vault / saved Thoughts / Nemotron | **PASS** (no service started; GitHub documentation only) |
| No production code, policy, merge, deployment or next-stage work | **PASS** |

**Observed regressions in this WP1-E run:** none can be assessed, because the load run did not execute. This is **not** evidence of regression-free behavior or an acceptance pass.

## Exact next measurement-only work (proposed, NOT implemented)

1. On a machine or disposable runner that can check out all **three exact SHAs** (including the private Engine repository), build `tass-core` from `core/cmd/tass-core`, `thought-engine` from the Engine repository root, and `tass-identity` from `cmd/tass-identity`; verify `git rev-parse HEAD` in each. Never use a live owner's data directory or ports.
2. Start from [`nudge_process_native.py`](https://github.com/Arameair/Thought-Processor/blob/2ed99269a1ef8da96319d2c776ba0651b71633ca/core/test/nudge_process_native.py)'s port preflight, clean-service environment, throwaway `APP_HOME`, fixture model, and sentinel checks. The existing [`cognition_refresh_load_native.py`](https://github.com/Arameair/Thought-Processor/blob/2ed99269a1ef8da96319d2c776ba0651b71633ca/core/test/cognition_refresh_load_native.py) provides the GET, token, Identity log and request-timing helpers; it currently runs only a one-Thought baseline.
3. **Smallest necessary test-only extension:** provision one authorized **synthetic** linked entity with the existing fixture; populate disposable Engine Thought JSON **offline while the Engine is stopped** (matching Engine `Thought` schema and strict loader), then restart it. Seed 1, 50, 100 retained Thoughts, each with exactly 50 saved Notes of a documented, fixed length (use 400-character synthetic content for a bounded heavy case), and documented history entries. For 100 Thoughts, respect the default 50-open limit by retaining at least 50 in sleeping/resolved state. Use a unique private-Note canary and verify it is absent from library responses. Verify loaded counts through the raw Engine GET before timing anything.
4. For each scenario, use the existing isolated test service keys: record byte length of `GET /api/entities/{eid}` **at the Engine** and `GET /api/cognition/library` **at Core**; record exact status/failure counts, at least 30 sequential warmed library samples (plus explicit warmup count), median and nearest-rank p95 in milliseconds, and exact sample count. Measure before/after `decisions.jsonl` lines/bytes for library and pulse separately; do not count setup calls. Sample process RSS/CPU only if a reliable PID/process mechanism is available, with units and sampling method. Keep fixture-model request counts unchanged through polling.
5. For hidden-tab evidence, exercise the already-tested visibility logic against an isolated real-service browser fixture and compare actual audit-log deltas during a bounded hidden period against visible polling, explicitly accounting for in-flight requests. Assert only GETs, no nudge/lesson/model execution, correct Identity binding, and no card canary. Stop on first isolation/security failure.
6. Save exact command lines, script SHA, fixture generation seed, scenario JSONL or summarized raw samples, sample sizes, machine details, HTTP status histogram, bytes, `size / 6,291,456 × 100`, median/p95, audit deltas and any failures in a **follow-up revision of this one report**, then request independent Codex evidence review. If and only if the large raw response is **> 3,145,728 bytes**, file a separate Engine summary-endpoint proposal for owner review (no implementation).

**Available baseline invocation (NOT the three-scenario runner):** once the binaries are built under an independent `BIN` directory, run with a new empty `APP_HOME`:

```bash
APP_HOME=/tmp/wp1e-disposable/app BIN=/tmp/wp1e-disposable/bin SEED=/tmp/wp1e-disposable/seed \
CORE_BIN=tass-core-wp1e ENGINE_BIN=thought-engine-wp1e IDENTITY_BIN=tass-identity-wp1e \
S2_PORT_BASE=39100 LOAD_READS=20 \
python3 core/test/cognition_refresh_load_native.py
```

**Important:** that existing command **does not** create 50 Notes per Thought or the 50/100-Thought scenarios. It must not be represented as WP1-E completion.

### Decision / handoff

- **DONE:** exact revisions and existing harness reviewed; previous Slice 2 Identity figures and exact 6 MiB threshold preserved; this documentation-only evidence record created.
- **OPEN:** three native load scenarios, new-baseline Identity/hidden-tab audit observations, actual latency/p95, raw/projection sizes, security checks under load, memory/CPU.
- **NEXT:** provision the smallest measurement-only scenario seeder on an isolated runner; execute, amend this report with real evidence; independent Codex review; owner Stage 1 acceptance **only after** WP1-E satisfies its measurements.
- **BLOCKED:** no pinned native executable source/binaries in this test container, no GitHub DNS for checkout, and existing fixture script has no bulk-Thought/Note provisioner. **Stage 1 load acceptance remains blocked.**
