# Experiment report — Thought Processor diagnostic shell (2026-10-24)

Human-directed, AI-assisted (Claude). Specification: [Identity-approved build package @ `be690461`](https://github.com/Arameair/Thought-Processor/tree/be6904618a9533513746f8e9aaf353ece37498f8/prototype); earlier research: [story prototype @ `1c8ebc4c`](https://github.com/Arameair/Thought-Processor/tree/1c8ebc4c1d4ed0deded8e866968b5df6a7e02230). Where they conflict, the Identity-approved package was followed (conflicts listed in the [dependency request §4](../contracts/IDENTITY_ENGINE_LIFECYCLE_APPROVAL_REQUEST.md#4-conflicts-found-between-documents)).

## Status (separate claims)

| Claim | Status |
| --- | --- |
| BUILT | **Yes** — Gate 1 diagnostic shell |
| LOCALLY TESTED | **Yes** — on a cloud Linux VM (below), **not** on the owner's Windows/WSL/Debian machines |
| Gate 2/3 (Identity-approved start/attach/stop) | **BLOCKED** — no Identity producer, no runner, no Engine-side validator |
| OWNER ACCEPTED | **PENDING** |
| Promotion | **NOT AUTHORIZED** — Alpha, Stable and Main untouched |

## Revisions and environment

| Item | Value |
| --- | --- |
| Branch | `experiment/processor-diagnostic-shell-20261024` |
| Alpha base | `9fac9fa44cb020bb6f8aa6bcb1e9faeda6e0b47a` |
| **Tested code SHA** (embedded in the binary, `vcs.modified=false`) | `37255d91f7c7318f3210e34542faaeaa3092422b` |
| Later commits | Documentation and evidence only (no `processor/` changes) |
| Identity used (external program, started by the test script) | `Arameair/identity` `experiment/first-admin-wizard-20261023` @ `4686c94f7bdb84ebb8abf70e7409c87580dc461e` (it self-reported this commit in its log) |
| Engine used (external program, started by the test script) | `Arameair/Thought-engine` `experiment/identity-grant-enforcement-20261024` @ `9ecd863645c24d3d7ad113426fefeeedbcbcd75e` (self-reported), no worker configured |
| OS / toolchain | Ubuntu 24.04.5, Linux 6.18.44 x86-64 (glibc 2.39); Go 1.24.7; headless Chromium 153 via Playwright (Python 3.13) |
| Ports | Processor 7860, Identity 7796, Engine 7798 |
| Data roots | Fresh temporary folders per run (paths in evidence shown as `<tmp>/…`) |

## What the Processor actually does

- Runs on its own (`go build`, one ~9.7 MB stdlib-only binary), own port and own profile folder. No Project Controller, Identity, Engine or GitHub needed to render or diagnose.
- **Self status from real evidence:** pid, port, run id, start time and startup duration, commit embedded by the Go toolchain (`unknown` if built without git), profile folder, and "owns no processes".
- **One next action** derived from evidence: set Identity address → start Identity yourself → check the address → blocked: read the dependency request.
- **Read-only discovery** of addresses the owner types in (loopback only; remote, credentials, paths and `file://` refused). Identity: TCP check plus its public `GET /api/health`; returned text is shown as an **UNVERIFIED CLAIM**. Engine: TCP check only — its `/api/status` contains private entity/log data, so it is not read before an approved attachment. Results carry a timestamp and become **STALE** after 20 s without a re-check.
- **Protected operations** (start, attach, stop, observe): disabled in the page with a reason, and refused **server-side** with HTTP 501 before any effect. They accept no parameters. The binary contains no process runner (`os/exec` is not in its dependency graph).
- **Local operator session:** per-run code printed in the terminal and saved in the profile (0600). Gates Processor's own private pages. Clearly labeled as *not* an Identity approval.
- **Request guards:** exact Host check (DNS-rebinding), same-Origin, JSON-only and CSRF token on every action; no CORS; GET never changes anything; strict CSP.
- **Single instance per profile** (kernel file lock), clear error on busy port, unclean-previous-run notice, clean shutdown that removes its run record and code.
- **Copy Context:** allowlisted summary; omits operator code, cookies, CSRF, request bodies, environment and any text returned by other programs.

## Capability ledger

| Capability | Status | Evidence |
| --- | --- | --- |
| Self status, single instance, operator session, request guards, address probes, Copy Context, clean stop/restart | **WORKING (observed)** | A01, A11a, A13, A18–A21, OPS-04, SAFE-07 |
| Authenticate the Identity candidate | UNAVAILABLE | Identity has no peer-authentication mechanism |
| Identity approval for start / attach / stop | **BLOCKED** | [Dependency request](../contracts/IDENTITY_ENGINE_LIFECYCLE_APPROVAL_REQUEST.md) |
| Bounded runner | **BLOCKED** | None exists; intentionally not built here |
| Read Engine status / open Engine UI | **BLOCKED** | Requires approved attachment |
| Observation ingest | **BLOCKED** | Separate Engine-enforced grant; Identity side not implemented |
| GitHub bootstrap, licensing, updates, publisher entitlements | UNAVAILABLE | Owner decisions open; not implemented |

## Acceptance results (`prototype/ACCEPTANCE.md`)

All live results come from one run (2026-10-24 08:47:57–08:48:33 UTC) of [`processor/acceptance/live_acceptance.py`](../../processor/acceptance/live_acceptance.py) on the tested SHA. Raw generated evidence: [`evidence-2026-10-24/evidence.md`](evidence-2026-10-24/evidence.md).
Layers: **L1** unit, **L2** labeled fixture, **L3** real processes on this host. No result below claims integration with a real Identity decision, because none exists.

| ID | Verdict | Layer | Evidence (abridged) | Limitation |
| --- | --- | --- | --- | --- |
| A01 | **PASS** | L3 | Empty profile: ready in 24 ms (pid 4895, :7860); state needs session (401); Identity/Engine NOT_CONFIGURED; next = set Identity; 4/4 controls disabled; Processor children `[]`; no Identity/Engine processes | — |
| A02 | **PASS** | L3 | Identity absent (:7796 CLOSED). API start/attach/stop → 501 `blocked_no_identity_approval_service`, observe → 501 `blocked_no_ingest_grant_contract`, each with op id. UI buttons disabled; UI "Ask server: start" → HTTP 501 BLOCKED. Zero Engine processes | — |
| A03 | **BLOCKED** | — | No Identity decision endpoint exists to deny | — |
| A04 | **BLOCKED** | — | No decision path to time out / be malformed | Probe-level timeout, redirect and malformed handling tested only with **L2 fixtures** |
| A05 | **BLOCKED** | — | No decision or validator to mis-scope | — |
| A06 | **BLOCKED** | — | No decision to expire/revoke/replay | — |
| A07 | **BLOCKED** | L3 | No approval type exists; attach and start refused 501; no runner | — |
| A08 | **BLOCKED** | L3 | start → 501; zero Engine processes started by Processor | — |
| A09 | **BLOCKED** | L3 | Real external Engine pid 4941 running: Processor saw only "port OPEN", did not read `/api/status`; attach → 501; Engine untouched | — |
| A10 | **BLOCKED** | L3 | No ensure operation; repeated refusals each get own op id, no effect | — |
| A11a | **PASS** | L3 | Second Processor on busy port → exit 1 with clear message; second on same profile → exit 1 naming the running run/pid/URL; Identity address pointed at the real Engine → "Port open; no recognizable Identity metadata" (404), next = check address; remote address → 400. Original Processor kept running | Processor-side only |
| A11b | **BLOCKED** | — | Missing executable / digest / protocol checks belong to a runner | — |
| A12 | **BLOCKED** | — | Nothing is started | — |
| A13 | **PASS** | L3 | New browser → sign-in screen; reopened tab → still signed in. Stop Processor → 202, exit 0 in 115 ms, run record removed, no leftover processes; Identity and Engine alive. Restart in 25 ms (pid 5008, new run id); old cookie → 401; new code; saved Identity address kept and re-checked | Processor owns nothing, so no survival is claimed |
| A14 | **BLOCKED** | — | No ownership ledger (owns no processes); never kills by PID/port | — |
| A15 | **BLOCKED** | L3 | No session-owned Engine can exist | — |
| A16 | **PASS** | L3 | Stop twice → 501 BLOCKED (two op ids); external Engine pid 4976, Identity and Processor all still alive | — |
| A17 | **BLOCKED** | L3 | Precondition (valid start/attach) cannot exist. Observed: observe refused 501, control disabled, Processor has no sending code path | — |
| A18 | **PASS** | L3 | Cross-origin form/JSON, no Origin → 403; rebinding Host → 421; missing/wrong CSRF → 403; no session → 401; GET start/shutdown → 405; CORS preflight 405 with no `Access-Control-Allow-Origin`; real Chromium cross-origin `fetch` blocked | — |
| A19 | **PASS** | L3 + L2-live fixture | start with path/args/endpoint → 400 `parameters_not_accepted`; `..` path and `file://` addresses → 400; no file created; raw TCP listener used as "Engine" received **0 bytes** (connect only) | Listener is a labeled fixture |
| A20 | **PASS** | L3 + L2 | Live Copy Context (1912 B) and profile log contain no operator code, cookie or CSRF; no remote claim text. Canary planted in a fixture Identity's reply never reached Copy Context or the log (Go test) | No private observations exist to leak |
| A21 | **PASS** | L3 | Engine OPEN, then stopped by the test script → next re-check UNREACHABLE (CLOSED). After 21 s without a re-check, Identity shows **STALE** | Detection follows the page's 10 s re-check |
| A22 | **PASS** | L3 | Engine original UI marked BLOCKED with reason; only link on page is the dependency request; no embedded or rebuilt Engine UI | — |
| A23 | **NOT RUN** | — | GitHub bootstrap undecided; out of scope | — |
| A24 | **NOT RUN** | — | Admin enrollment / GitHub out of scope | — |
| A25 | **BLOCKED** | L3 | Engine ran without a worker; Processor cannot read worker state without attachment and invents none | — |
| A26 | **BLOCKED** | — | No attach to revoke | — |
| A27 | **PASS** | L3 | Integration BLOCKED; diagnostic shell only; every protected call refused before effect; no bypass mode exists. With real Identity running: shown "Found · unverified", next = blocked on Identity, start still 501 | — |

Additional checks: **UX-01/06 PASS** (Next button focuses the address field; 0 px horizontal overflow at 390 px width) · **OPS-04 PASS** (after `kill -9`, restart in 25 ms reports "previous run ended without a clean stop" and takes no action) · **SAFE-07 PASS** (3 clean cycles).
Go tests (L1/L2): 12 tests — 11 pass, 1 skipped because the VM runs as root (unwritable-profile check needs a normal user).

**Not tested:** Windows, WSL, Debian bare metal, macOS; screen readers; owner usability.

## Timings (actual)

| Measurement | Samples |
| --- | --- |
| `go build` (warm cache) | 946 ms |
| Exec → `/api/health` answering | 24, 25, 25, 22, 22, 22 ms |
| Self-reported internal startup | 5, 2, 2, 3 ms |
| Graceful stop (SIGTERM) | 3, 3, 3 ms; via Stop button 115 ms (includes a 100 ms delay so the reply reaches the page) |
| Identity probe (TCP + health) | 0–1 ms on loopback |
| Detecting a vanished Engine | next re-check (≤ 10 s cadence); the check itself 2 ms |

No authority-decision or runner-readiness timings exist (BLOCKED).

## Process ownership and cleanup

Processor started **zero** processes in every test (children list empty each time). It stopped only itself. Identity and Engine were started and stopped by the test script, playing the owner. Final orphan audit after the run: no Processor, Identity or Engine processes left that were not present before.

## Sanitized Copy Context sample

See [`evidence-2026-10-24/copy-context-sample.txt`](evidence-2026-10-24/copy-context-sample.txt).

## Dependency handoffs

See [IDENTITY_ENGINE_LIFECYCLE_APPROVAL_REQUEST.md](../contracts/IDENTITY_ENGINE_LIFECYCLE_APPROVAL_REQUEST.md): Identity producer (three scopes), Engine authenticated attach boundary + public metadata, runner with manifest validation and ownership records, and owner decisions still open.

## Bugs found and fixed during testing

1. Opening the sign-in link in a tab already showing the sign-in page did nothing (fragment change, no reload). Fixed: page reacts to `hashchange`.
2. Next Step had no button, and its text suggested a guessed Identity port. Fixed.
3. Copy Context printed `http=0` for checks that made no HTTP request. Now `http=none`.

## Owner walkthrough (≈ 10 minutes)

1. Build and start (see [processor/README.md](../../processor/README.md)): `./thought-processor -data ~/tp-trial`. Open the printed link.
2. **Identity absent:** you should see *Tell Processor where Identity runs*. Click **Enter Identity address**, type `127.0.0.1:7796`, **Save** → *Unreachable*, next step *Start Identity yourself*. Open **Prove it: ask the server anyway** → **Ask server: start** → HTTP 501, `BLOCKED`, "no process started".
3. Start your Identity yourself (any build with `/api/health`, e.g. `experiment/first-admin-wizard-20261023`), click **Re-check now** → *Found · unverified* with the claim marked **UNVERIFIED CLAIM**; next step *Engine control is blocked — waiting on Identity*. **This is the real blocker; per ACCEPTANCE, the walkthrough stops here** — steps 2–6 of the official walkthrough (approve, ensure, revoke, stop owned Engine) cannot be performed.
4. Optional broken path: put the Engine's port in the Identity box → *Port open; no recognizable Identity metadata*.
5. Open **Diagnostics** → **Copy context**, paste it somewhere and check it holds no codes or secrets.
6. Click **Stop Processor**. Confirm Identity (and any Engine) are still running. Start Processor again: it prints a new link; the old tab asks you to sign in again; your saved address is still there.
7. Record **KEEP / REVISE / REJECT**.

## Rollback

Nothing outside this branch changed. To discard: delete the profile folder you used and `git push origin --delete experiment/processor-diagnostic-shell-20261024`. Alpha (`9fac9fa4`) is unchanged.

## Owner verdict

**PENDING.** Promotion: **NOT AUTHORIZED.**
