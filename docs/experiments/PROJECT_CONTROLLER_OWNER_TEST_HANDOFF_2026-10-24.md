# Project Controller owner test handoff — Processor diagnostic shell
**2026-10-24 · Owner-directed follow-up notes · DOCUMENTATION ONLY · NOT TEST RESULTS**

Purpose: The owner uses **Project Controller to run and compare all experimental project branches**. Do NOT redirect them to manual cloning/WSL as the default workflow. Claude is unavailable for additional implementation now, so preserve actionable findings and enable a **limited first test** with truthful scope. This note neither changes application code nor asserts this branch has been tested inside the owner's actual Controller.

## Canonical tested candidate

- Processor repo: `Arameair/Thought-Processor`
- Branch: `experiment/processor-diagnostic-shell-20261024`
- **Claude cloud-tested code SHA**: `37255d91f7c7318f3210e34542faaeaa3092422b`
- **Claude's report commit before this documentation follow-up**: `7ed662ec508f804f0d472b71d8395b568e435c67`.
- Built in: `processor/` (nested Go module). `GET /api/health` responds for the running Processor, and `GET /` can return the sign-in shell without an authenticated operator.
- [Claude's report](PROCESSOR_DIAGNOSTIC_SHELL_2026-10-24.md); [Processor run instructions](../../processor/README.md); [Identity/Engine missing approval contract](../contracts/IDENTITY_ENGINE_LIFECYCLE_APPROVAL_REQUEST.md).
- Existing Identity test target: `Arameair/identity` → `experiment/first-admin-wizard-20261023` @ `4686c94f7bdb84ebb8abf70e7409c87580dc461e` (read-only public health, **no** lifecycle approval).
- Existing Engine test target: `Arameair/Thought-engine` → `experiment/identity-grant-enforcement-20261024` @ `9ecd863645c24d3d7ad113426fefeeedbcbcd75e` (Engine-side grant experiment, NOT authenticated Processor attach and NOT actual Identity grant integration).
- All are distinct projects. Claude updated **Processor**, not Identity or Engine. Neither Alpha nor stable/main is to be promoted or rewritten for this trial.

## Findings from inspecting Project-controller Alpha and Processor branch

1. **Controller auto-build detection does not work for this Processor branch as-is:** Controller's detector checks the repository **root** for `go.mod`, `marquee.json`, `package.json`, or `index.html`. This Processor has its `go.mod` under `processor/` and no root `marquee.json`. With no per-project Setup/Run overrides, Controller cannot choose the correct build/run command. This is a **launch packaging gap**, not evidence that Processor's application code failed.
2. Controller passes **`PORT`** and branch-specific **`PC_DATA`**. Processor currently reads **`-port`**, **`-data`**, or `THOUGHT_PROCESSOR_DATA`; it does not read `PORT`. An auto-detected generic Go start would use Processor's default `7860` instead of Controller's assigned port.
3. Processor sends **`Content-Security-Policy: frame-ancestors 'none'`** and `X-Frame-Options: DENY`. An iframe preview inside Controller **may be blank/refused by the browser as designed**, even when Processor itself has started successfully. Use Controller's **Open ↗ / New Tab** to test the real application; do not loosen Processor's frame/security policy merely to obtain an embedded preview.
4. **Security concern: one-time local operator code in logs.** Processor prints `http://127.0.0.1:<port>/#code=<192-bit-hex>` at startup, and writes the per-run code into its protected profile. Controller captures the stdout line. Reviewing Controller Alpha's redactor, its patterns cover credential names like *token/password/session*, but **do not clearly cover a URL fragment named `#code=`**. Treat `#code=` as a secret. **Do not paste/share Controller Copy Logs or Copy Context containing startup stdout** before verifying that the full access code is redacted. Prefer Processor's own allowlisted **Copy context** after unlocking it; inspect for secrets before sharing. This remains a P0 test of Controller/Processor integration and may need code changes on one or both sides. These strings are not GitHub identity credentials and do NOT confer Engine authorization, but they unlock this Processor's local diagnostics for that run.
5. Controller `GET /` HTTP readiness only proves a sign-in page responds, **not that Processor operator session or Identity/Engine integrations are ready**. UI must distinguish branch process RUNNING from product setup or Engine attachment.
6. Linux cloud test coverage does **not** establish native Windows correctness. `processor/lock_other.go` uses exclusive-create lock outside Unix and can leave a stale lock after a crash, requiring verification before cleanup. Windows Controller commands, process termination and reused data root need genuine owner-machine testing.
7. Controller's **project-level Setup/Run settings override the repo manifest** and apply to that project's branches, not just a single selected version. Owner must review effective plan and clear overrides before testing a different branch. The Controller's SHA/command review and Copy Context should show exact source of effective build/run commands.
8. Controller's own **Play/Stop** is an independent, owner-invoked *development harness* operation, **not** Processor being authorized by Identity to Start/Attach/Stop Engine. Running an Engine branch as a separate Controller pane is a manual developer operation. Nothing about that authorizes Processor→Engine private API access or observation ingest.

## The one next Claude task (packaging, no architecture bypass)

On a **new Processor experiment branched from current Processor Alpha** (owner's branch policy), or an owner-authorized new revision of this candidate with explicit before/after SHAs, add a **repo-root `marquee.json`** (or equivalent Controller-supported manifest). It must:

- Build Go module `processor/` (not root) on Windows and Linux; place compiled executable in a managed/isolated build location, never overwrite source or a running binary.
- Pass Controller's unique `PORT` to Processor's `-port` and unique branch `PC_DATA` to `-data`. Preserve exact VCS metadata and isolated profile. Do not assume the developer's permanent `~/.thought-processor/default` profile.
- Set Controller readiness path to Processor's **actual** `/api/health` where supported. The current Controller preview path convention supports `path` in `marquee.json`, but if `/api/health` becomes the iframe URL, keep **New Tab** available to open `/` for the owner. Verify exact preview behavior; do not use the readiness path as a substitute for real UI.
- Specify and verify **Windows** and **Unix** setup/run variants; avoid platform-specific shell assumptions. Controller's `marquee.json` supports `windows.setup`, `windows.run`; `portEnv` only adds an environment variable and is **not** command-argument mapping.
- Preserve Processor's loopback binding, CSP, operator-session guards, no-runner policy and server-side 501 refusals. **Do not** install a fake Identity decision producer or downgrade private Engine attach requirements.
- Coordinate P0 **operator-code redaction** in the Controller's Copy Logs/Context (or adjust Processor's login handoff/console output) before recommending log exports to AI or GitHub.
- Test in the user's Project Controller (not just Claude's cloud VM): Play → real GET/health → Open ↗ actual sign-in UI → local unlock → inspect Identity/Engine candidate states → Copy Processor context (redacted) → Stop → Restart → orphan audit.
- Detect parent Controller OS and show a helpful **Open in new tab** affordance or accurately report embedding blocked; do not silently remove security headers.

**Note:** A manifest will make it launchable under Controller, but **it will not enable Engine authorization**. That requires the separately scoped Identity decision producer and Runner/Engine enforcers.

## What can be tested RIGHT NOW without further Claude changes?

### Option A — safe first pass, no manual commands
Run/open the **already launchable** Identity and Engine candidate branches as separate **Project Controller panes** (only if they are not already running; don't duplicate services). Check Controller stage timings, actual ports, Copy Context (after redaction inspection), logs, Stop/Restart of pane-owned processes. **These are individual project tests**, not an integrated Processor–Engine authorization test.

For the Processor branch, **open the workspace and inspect the proposed effective build/run plan**. Without a configured override/manifest, its automatic launch is expected to refuse with **“couldn't tell how to run this branch”**; that is a known packaging blocker, not a failed cognition result.

### Option B — early Processor trial through Controller **with manual project Setup/Run override**
For an *explicitly trusted* Processor branch, Controller's Project Settings can override the missing root manifest. Review exact effective commands before Play. These are **suggestions to validate on the actual Controller host**, not commands already tested in Controller:

**Windows-native Controller (cmd.exe for Setup/Run):**

```text
Setup: cd processor && go build -o ..\thought-processor.exe .
Run:   thought-processor.exe -data "%PC_DATA%" -port %PORT%
```

**Linux/WSL Controller (sh):**

```text
Setup: cd processor && go build -o ../thought-processor .
Run:   ./thought-processor -data "$PC_DATA" -port "$PORT"
```

**Important:** Those commands put a temporary executable in the Controller-owned checkout for the manual trial. Do not perform manual edits in that checkout. A later manifest should avoid source-tree build output. If the effective plan differs, the build fails, port isn't bound, `PC_DATA` isn't populated, or the command is not supported on your Controller variant, **STOP and capture only sanitized error/status** instead of guessing alternative commands.

**The preview may show a blank iframe because of Processor CSP.** Use **Open ↗ New Tab** on the Processor pane and the one-time operator code from the local Controller startup console/log (keep it private; do not copy it into a shared issue).

**Test checkpoints:**
1. Play Processor selected branch and confirm Controller reports the correct branch/SHA and app PID/port; 2xx readiness alone means only HTTP is running.
2. New Tab reveals real Processor sign-in. Unlock only this local Processor, not GitHub/Identity.
3. Empty state reports Identity/Engine not configured or unverified; does not show fabricated services.
4. If Identity already running in Controller, enter its **actual** displayed port; Processor should report **Found · unverified**, never Identity-authorized.
5. Enter the running Engine's actual displayed port; Processor should show **Port open · unidentified**, never authenticated or attached. No private `/api/status` read.
6. “Ask server” Start/Attach/Stop/Observe should yield **501 BLOCKED**, zero subprocess effects.
7. Stop Processor pane; Identity/Engine panes remain. Restart Processor; operator code changes, settings persist in branch profile, no duplicate child Engine created.
8. Only after inspecting and REDACTING captured text, share **Processor's own Copy context** for analysis, plus Controller metadata without live operator-code URLs.

### Expected status for first owner trial
- **Possible now (not yet verified on owner machine):** Controller launch with manual settings, Processor local UI, real self-status, candidate Identity/Engine *port* checks, truthful blocked control endpoints, clean Stop/Restart.
- **Not possible with current code:** Identity-authorized Processor Engine start/attach/stop; private Engine status/worker data; source→Engine genuinely authorized observation; real Thought; complete GitHub-owned first opening; producer/launcher security acceptance.
- **What success means today:** A dependable **operational shell**, not connected cognition. It should be allowed to report **BLOCKED** for missing authority.

## What to capture at the end of the first user trial

- Controller branch/SHA, host OS, selected Processor branch/SHA, effective Setup and Run plan source (without secrets), Controller build/start/HTTP-ready timings, real port and PID.
- Whether preview embedding was blocked and **New Tab** worked; whether sign-in code could be used **without publishing it**.
- Exact Identity and Engine source branches/ports, Probe results (including UNVERIFIED wording) and checked times; server 501 evidence.
- Whether Stop Processor left independently running Identity and Engine untouched, and restart worked; Windows stale-lock behavior if applicable.
- Sanitized Processor Copy Context + your UI feedback: **KEEP / REVISE / REJECT**.
- Zero claims of authorization, publisher entitlement, a model response or Thought. Never collect/share raw tokens.

**Owner decision pending.** No code implementation, test execution, branch promotion or cross-repository runtime change is implied by these notes.

*Prepared with AI assistance for human review.*
