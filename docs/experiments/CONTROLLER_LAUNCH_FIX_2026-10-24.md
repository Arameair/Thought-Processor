# Project Controller — launch fix and first owner test

Date: 2026-10-24. **New Alpha-based experiment** `experiment/processor-controller-launch-fix-20261024`.

The owner's previous Processor run failed at **approval/detection** after a successful clone: `couldn't tell how to run this branch`. The nested Go module lacked a root manifest. This branch supplies `marquee.json` with Linux/WSL and Windows commands, using Controller-provided `PORT` and `PC_DATA`.

## Verify before Play

1. In Controller → Thought-Processor → Project Settings, **clear any stale saved Setup/Run overrides**. If left set, Controller uses those instead of the repo manifest. Check the effective plan it displays.
2. Choose **`experiment/processor-controller-launch-fix-20261024`**. Confirm setup builds `processor/`, run passes `-port "$PORT" -data "$PC_DATA"` (Linux; Windows uses `%PORT%` and `%PC_DATA%`).
3. Approve the shown setup/run plan manually. This approves **Project Controller's developer harness operation only**, not Identity authorization to control Engine.
4. Check build/HTTP readiness, then use **Open ↗**. Browser iframe is intentionally blocked; do not disable CSP.
5. To sign in, **read the locally stored code** from the `profile: ...` directory printed to startup logs, file `runtime/operator-code`. **Managed startup no longer emits a secret URL into Controller logs**. Do not share code or its content. On Linux `cat '<profile>/runtime/operator-code'`; paste into the local sign-in form.
6. Inspect first-use Next Step, Identity unreachable/wrong program, Engine port state, Flow/Logs/Metrics, blocked protected actions and Processor Copy Context.
7. Controller Stop and Restart. Verify old instance/process cleaned up, independently running Identity and Engine unaffected, operator code rotated.

## Before sharing results

- Controller **Copy Logs** should now have no `#code=<secret>` URL from Processor startup when launched in Controller. **Still inspect exports for any other secrets** before sharing.
- Include the running branch/SHA, process ID, assigned port, startup/stop timing, sanitized logs, and what happens after restart.
- Report `PASS/FAIL/BLOCKED/NOT RUN`; do not equate `GET /` or `GET /api/health` answering with authenticated Identity/Engine readiness.
- Windows file-lock recovery remains **not verified**. Identity/Engine grants are blocked by design, outside this fix.

## Scope / correctness

The manifest's build output is an untracked binary in Controller's **managed isolated checkout**. No user source checkout is modified. Controller's checkout is cleaned between runs. The app's stdout/no-secret safety mode only activates when `PC_MANAGED_BY_CONTROLLER=1`; independent foreground launch retains the one-time link.

The new code must be tested on the owner's machine; copying the pretested source and adding a manifest **does not** establish a passing live Controller run. Human owner decides whether to promote.