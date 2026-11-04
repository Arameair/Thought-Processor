# Atrium pulse-driven test cards (Issue #51)

**Status:** experimental, separately reviewable. First source branch:
\`experiment/atrium-pulse-test-cards-20261104\` from \`experiment/atrium-daily-mission-20261104\`.
It changes **only Atrium presentation, its local test-card module/packaging, tests and a separate CI workflow**. No Thought Engine, Go Core, Docker Compose, Identity policies, GitHub permissions, or live owner processes were altered. Existing \`desk.json\` task notes and Workshop chat routes are unchanged.

## Owner interaction on the existing Atrium

Open **Today's desk** (\`/today\`) and scroll to **06 / Verified Test Cards** or select **Test cards** in navigation. This is a new, separate ledger, not a reinterpretation of owner-reported PASS in the existing task board.

1. Inspect the objective, expected result, **exact repository and 40-character SHA**, fixed operation and arguments.
2. Click **Approve this exact test**, review the explicit confirmation. This records a *local UI attestation*, **not an Identity permission grant**.
3. **Queue once** is unavailable until a separately isolated test adapter is explicitly enabled by the operator. Once enabled, click Queue once.
4. Use **Step once** to dispatch one approved job while paused, or **Resume** for a configurable 20-second interval. **Pause** stops new dispatch, but lets work already running complete. **Stop test** cancels its process group and blocks queued work.
5. Observe QUEUED → WORKING on first pulse; on a later pulse see the real, bounded **receipt** with job/receipt ID, exact SHA, operation, started/finished timestamps, exit code, output digest and sanitized excerpt. A red CI regression assertion remains **FAIL**, not an artificial green.

**Default:** paused, execution disabled, no source checkouts mounted and no Go compiler in the existing Atrium Docker image. The UI is usable for review/approval and status without granting execution. The 20-second scheduler uses no model inference and never calls Workshop's action routes.

### The only two production operations

| Card | Exact checkout | Fixed subprocess argv (no shell) | Expected evidence |
| --- | --- | --- | --- |
| IDENTITY-014 | \`Arameair/identity\` @ \`b0c7267b47f6111c5024c9bba8a4135e2357b432\` (\`alpha\`) | \`go test ./internal/authority/... ./internal/pairing/... ./internal/entities/...\` | Exit 0 / PASS from real package tests; no Identity policy change |
| INTEGRITY-050 | \`Arameair/Thought-Processor\` @ \`aa800af3eee5d026521395bde3236737285e6287\` (PR #49) | \`go test -race -tags=acceptance ./tass -run=^TestAcceptanceRejects -count=1 -v\` from \`core/\` | **FAIL** while Issue #50 is unfixed; does not execute the Core's proposed actions |

The card request body can only select card ID and one of \`approve/queue/pause/resume/step/stop\`; it cannot supply a command, repo, SHA, filesystem path, model request, env value or arbitrary code. The backend revalidates the fixed spec, exact Git SHA, expected origin remote, separate clean worktree and configured source root immediately before dispatch.

## Interim trust boundary and local execution opt-in

**Atrium is not Identity.** Its browser CSRF token and the localhost address do not authenticate the human owner. The actual Identity authorize API for test execution is not connected. This pilot is therefore **not authorized to run Go tests in the owner’s normal Atrium Docker service**: it lacks Go, and it shares state/network access with the Workshop ecosystem. Do not add a Docker socket, privileged mount, secrets, network capability, or whole home directory to make Queue work.

A **separate disposable OS sandbox** may opt into the restricted adapter, after a human establishes OS isolation and approved checkouts. The adapter checks:

- process is a non-root POSIX user; fixed \`go test\` argv, \`shell=False\` and preapproved repo+SHA+operation only;
- \`TASS_ATRIUM_TEST_EXECUTION=1\` **and** \`TASS_ATRIUM_TEST_SANDBOX_CONFIRMED=1\` set by the sandbox operator; both absent by default; these env flags are **not** proof of owner identity or isolation;
- absolute checkouts under \`TASS_ATRIUM_TEST_ROOT\`, set separately by \`TASS_ATRIUM_TEST_IDENTITY_CHECKOUT\` and \`TASS_ATRIUM_TEST_INTEGRITY_CHECKOUT\`, with clean Git HEAD and exact known origin;
- minimal subprocess environment (no GitHub tokens/API keys passed), network-free module resolution (\`GOPROXY=off\`), bounded elapsed time and captured log size, sanitization and digest of actual stdout.

**Important:** Running \`go test\` necessarily executes code compiled from that checkout. Fixed command strings alone are not a complete OS sandbox. For owner execution the *outer host/container security boundary* must independently restrict network, mounts, process privileges, CPU/memory/disk, Git credentials and secrets. This Python adapter does not prove those constraints and should eventually be replaced by an Identity-authorized Go executor.

Fixture tests use a short, fixed Python child and a temporary side-effect file to demonstrate real process execution without touching the owner computer. The CI acceptance job separately checks out the two specific published Git revisions in disposable GitHub Actions workspace paths, starts actual Go test processes, checks exit/result/receipt, and tests restart idempotency. **Neither is real Nemotron GPU inference.** Neither changes branch pointers or merges anything.

## Durable state and pulse semantics

- Test state: \`TASS_ATRIUM_PULSE_STATE\`, Docker default \`/state/test-cards.json\`; standalone \`start_today.sh\` stores it under \`~/.local/state/tass-atrium/test-cards.json\`. Separate from \`desk.json\`.
- JSON is fsync + atomic replace + parent directory sync, mode 0600; a Linux advisory lock prevents a second Atrium process from scheduling the same ledger. Corrupt/changed specs block all work without overwriting the file.
- Job \`job-<uuid>\` is persisted as WORKING **before** launching its one subprocess. Every pulse has a unique \`pulse-<uuid>\`; interval \`TASS_ATRIUM_PULSE_INTERVAL\` defaults to **20 seconds**.
- Long jobs stay WORKING across ticks; one global active job maximum, no duplicate starts, repeated Queue never resets a completed job.
- Worker persists a bounded pending receipt; **next pulse** transitions to PASS/FAIL/ERROR. On restart, a persisted pending receipt is recovered. If restart finds WORKING without a known receipt, status becomes **BLOCKED: UNKNOWN_INFLIGHT** and never automatically retries.
- Manual Stop tries to kill a live process group, records an ERROR receipt at the next pulse, and stops further scheduling. Restart recovery intentionally does not assume an old child process is dead; external supervisor/operator must reconcile unknown work.

## Endpoints and protections

| Path | Method | Result |
| --- | --- | --- |
| \`/tests/state\` | GET | safe UI projection: pulse state, immutable test-card metadata, real receipts, no checkout secrets |
| \`/tests/action\` | POST JSON | one of \`approve, queue, pause, resume, step, stop\`; reuses Atrium's exact localhost Host, Origin, \`X-Playground-CSRF\` and JSON checks; 403/409 when not allowed |
| \`/bridge/message\` | POST | Existing Workshop chat bridge; completely separate from test-card approval/execution |
| \`/desk/state\` | GET/POST | Existing task notes; no migration and no dependency on this new work |

Data from a model/GitHub comment cannot be pushed to \`/tests/action\` and interpreted as execution authorization. The gate defaults to disabled even when the owner has approved a card.

## Automated verification (fixture vs real)

From a clean checkout of this experimental branch:

\`\`\`bash
python3 -m unittest discover -s workspace/atrium/tests -v
node --check workspace/atrium/assets/pulse.js
python3 -m py_compile workspace/atrium/server.py workspace/atrium/pulse_cards.py
\`\`\`

The test modules validate explicit approval, no execution by default, unknown commands and paths denied, local CSRF/Origin checks, simulated four-tick long-running work without duplication, pause/resume/step/stop, actual fixture child output and exit code, known FAIL not reclassified as PASS, sanitized receipts, corrupt state fail-closed, one-writer lock and crash recovery. Four ticks use a **manual simulated clock progression**, not 80 elapsed seconds.

CI workflow \`.github/workflows/atrium-pulse-evidence.yml\` performs actual pinned Go tests in a disposable runner with \`persist-credentials: false\`; uploads a JSON receipt artifact. The expected P0 FAIL of INTEGRITY-050 **does not make the workflow fail** if it is the actual known regression assertion, not a compilation error. Evidence includes exact commit, exit, digest and one-shot job count. IDENTITY-014 must actually exit 0; otherwise the CI workflow fails. Verified CI run: https://github.com/Arameair/Thought-Processor/actions/runs/41856885738 (success). The artifact `atrium-two-pinned-go-card-receipts` contains both redacted execution receipts. Its logs show IDENTITY-014 PASS and INTEGRITY-050 FAIL, each with one attempt; after controller restart, both receipts remain and no job reexecutes.

## PASS / FAIL / NOT RUN

| Gate | Status | Evidence |
| --- | --- | --- |
| Two fixed cards, approval UX, no default execution | PASS (fixture CI) | Atrium unit/HTTP tests |
| Duplicate queue, four simulated pulses while WORKING, pause/step/stop | PASS (fixture CI) | \`test_pulse_cards.py\` |
| Durable receipt and unknown-inflight restart without reexec | PASS (fixture CI) | \`test_pulse_cards.py\` |
| Unknown repo/args/path, missing CSRF/Origin, redaction | PASS (fixture CI) | \`test_pulse_cards.py\`, \`test_pulse_http.py\` |
| Actual pinned Go Identity packages | **PASS (CI, exit 0, one attempt)** | GitHub Actions 41856885738 |
| Actual pinned Go integrity regression | **FAIL (known P0 assertions reproduced; exit nonzero, one attempt)** | GitHub Actions 41856885738; PR #49 / Issue #50 |
| Owner’s local Atrium Docker running approved Go tests | NOT RUN; disabled until isolation and Identity authorization are reviewed | owner acceptance needed |
| Ten real wall-clock 20s coordination ticks under load | NOT RUN | fixture uses manual ticks; no owner run |
| Go Thought Engine pulse, live Identity authorization, Nemotron GPU inference | NOT IMPLEMENTED here | separate engineering tracks |

## Minimal, safe owner acceptance

**Single first step:** review the test-only PR and open the normal Atrium **Today's desk → Verified Test Cards** *after you choose to deploy the reviewed code*. Confirm both cards show their fixed SHA and PROPOSED status, the execution warning is **DISABLED**, Queue is unavailable, and your previous task notes/chat still work. Do **not** activate the execution env flags or mount source checkouts in the existing Atrium container. The CI artifacts establish execution behavior without your changing the current system.

To inspect the server without any new process launch (once deployed), \`curl --noproxy '*' -sS http://127.0.0.1:8791/tests/state\` should show \`execution_enabled:false\` and two PROPOSED cards. No merge, policy change or test invocation is implied by this GET.

**Replacement boundary:** Later, replace \`LocalTestAdapter.execute\` and the local coordination thread with one Go Thought Engine/Identity-authorized executor, retaining immutable card IDs, SHA, request receipt and owner-facing Atrium presentation. Do not mistake this pilot pulse for a new independent autonomous thinking system.
