# Expectations, tests and report
Prepared test design only. All tests start NOT RUN. This document contains no successful test results.

## Verdicts
PASS = actual required behavior observed with evidence.
FAIL = observed behavior violates the requirement.
BLOCKED = dependency/decision prevents the check.
NOT RUN = not attempted.
A passing disabled-control test is not a passing start/attach feature. Fixture tests are isolated and labeled; no fixture result appears as live product evidence.

## Required cases
| ID | Exercise | Required result |
| --- | --- | --- |
| A01 | Start Processor with empty isolated profile | Diagnostic/setup shell works; no invented account/entity/Engine/grant; no other process started |
| A02 | Identity absent; try UI and direct backend start/attach/control | All refused before effects; zero Engine starts/operational attachments |
| A03 | Real Identity explicitly denies | Denial with reason; existing processes unchanged |
| A04 | Identity times out or response is malformed/unverifiable | Unavailable/invalid evidence; fail closed; no implicit approval |
| A05 | Decision targets different actor, install, service, action, manifest or applicable entity/session | Refused by actual runner/control boundary, not just UI |
| A06 | Expired/revoked decision or replay outside approved operation | Refused before new operation; no unrelated kill |
| A07 | Real attach-only approval and Engine stopped | No spawn; explicit start approval required |
| A08 | Real start approval; correct Engine stopped | Exactly one approved owned instance; readiness checked separately; no inference |
| A09 | Real attach approval; correct external Engine running | Same instance; ownership external; no duplicate or takeover |
| A10 | Parallel/repeated ensure, including Processor restart | Same intended operation/instance; no duplicate; approval scope still enforced |
| A11 | Missing executable, wrong digest/protocol, occupied port or spoofed service | Specific refusal; unrelated process unchanged; no authenticated badge from claimed metadata |
| A12 | Slow startup, exit or partial failure | Bounded result and actual error; verified child cleanup only under defined runner policy |
| A13 | Close/reopen browser and restart Processor backend | Correct actual IDs and freshness; no duplicate; approval revalidated; no unproved survival claim |
| A14 | Stale ownership ledger, reused PID or uncertain child identity | No kill or false ownership; blocked reconciliation |
| A15 | Stop session-owned Engine, repeat stop; then try attached external Engine | Only proved owned processes stopped; external Engine, Identity, launcher, Processor and durable data preserved |
| A16 | No stop approval/control policy | Stop refused; denial visible; no unauthorized cleanup disguised as stop |
| A17 | Valid start/attach but no ingest grant | Engine may run; observation control disabled; zero observations/model calls |
| A18 | Cross-origin form/script, untrusted Host, missing/invalid session or CSRF proof | Control requests rejected; no effects; GET never mutates |
| A19 | Path/argument injection or unapproved endpoint/remote target | Refused; no shell execution, arbitrary proxying or unapproved network access |
| A20 | Copy Context/logs with canary secrets and private observations | Allowlisted bounded evidence only; secrets/raw user content excluded |
| A21 | Service disappears after successful status check | Freshness changes to stale/unreachable; history not mislabeled current |
| A22 | Original UI inaccessible or not authorized | Clear unavailable reason; no spoofed replacement UI or access bypass |
| A23 | Restart after repository creation succeeded but local setup failed (later onboarding) | Reconcile exact existing repo; no duplicate or overwrite |
| A24 | GitHub account differs from administrator; fresh install versus restore (later onboarding) | Explicit mismatch/recovery boundary; no administrator takeover or copied credentials |
| A25 | Worker unavailable | Report actual state; no fallback, synthetic result or implicit model call |
| A26 | Authority revoked after attach | No newly unauthorized control; report changed state; no assumed kill of independent Engine |
| A27 | Missing Identity producer/enforcement interface | Gate 2/3 integration BLOCKED; diagnostic shell only; no engineering-mode bypass |

A02–A07 and A16/A18 require both direct API and UI coverage where applicable. Actual Identity and actual selected Engine/runner are required for integration PASS. Negative isolated tests may use fixtures only with separate evidence labels.

## Evidence and measurements
For every test record: tested repo/branch/full SHA and Alpha base, OS/toolchain, isolated data root, action/command, timestamp, result, decision/operation/instance IDs where actually present, safe response/log excerpt, before/after process evidence and remaining uncertainty.
Measure actual startup, authority decision, runner readiness and stop durations; report samples and failures without invented speed targets.
For a functioning Gate 2/3 candidate, perform three clean start/inspect/stop runs and three attach/reconnect runs, each with actual approval and ownership evidence. Do not spend repeated trials on a known dependency block.
Test only the declared OS. Never mark the user's environment verified solely from a different host.

## Owner walkthrough
1. Open diagnostic shell with Identity absent: see the block and verify Ensure cannot start/attach.
2. With real protected Identity configured, select actual working context and approve a specific start/attach intent.
3. Ensure Engine once, repeat, inspect exact instance and process ownership.
4. Revoke approval or cause an authority outage, attempt a new control and see refusal.
5. Check original interface/diagnostics only within permission; inspect sanitized Copy Context.
6. Stop only a session-owned Engine; repeat with an external Engine and confirm it survives.
7. Record owner KEEP / REVISE / REJECT. No promotion implied.

If step 2 is impossible, stop the walkthrough at the real blocker; do not imitate the remaining steps.

## Report template
- Status: BUILT / LOCALLY TESTED / OWNER ACCEPTED (separate claims).
- Revisions/environment: actual values or NOT RUN.
- Capabilities: implemented and observed versus unavailable.
- Test table: ID | PASS/FAIL/BLOCKED/NOT RUN | evidence | limitation.
- Timings and process ownership/cleanup: actual measurements.
- Dependency requests: exact missing producer/consumer contract and owning repo.
- Owner trial and verdict: PENDING until actually performed.
- Promotion: NOT AUTHORIZED.
- Traffic lights: green actual verified behavior; yellow blocked/partial/unknown; red observed failure.

## Release expectations
Normal first-opening GitHub/ownership and publisher policies are not completed by this diagnostic slice.
Deny/outage and no-ingest protections are prerequisites, not future polish.
No application code or passed tests exist merely because this package is published.
