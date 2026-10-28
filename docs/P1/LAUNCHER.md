# P1 cleanup: one-command WSL/Linux launch (experimental)

**Not Alpha, not an installer.** This wrapper uses the existing P1 cleanup source and built-in trusted fixture; it does not grant permissions or sign in for you.

## First-time setup

Prerequisites: Git, Go 1.22+, Python 3.8+, Bash, and network access for cloning. On WSL/Linux:

```bash
mkdir -p ~/p1c && cd ~/p1c
git clone --branch experiment/p1-cleanup-claude-20261026 https://github.com/Arameair/identity.git
git clone --branch experiment/p1-launcher-gpt6-20261028 https://github.com/Arameair/Thought-Processor.git
bash ~/p1c/Thought-Processor/run-p1.sh
```

**One command on subsequent starts:** `bash ~/p1c/Thought-Processor/run-p1.sh`

The script requires Identity to be on the exact cleanup branch and known input SHA, and Processor on its launcher branch descended from cleanup. It prints complete source SHAs, verifies clean worktrees and preflight port availability, builds from source, starts trial profiles in `~/p1c/p1-launcher`, and waits for Processor to return HTTP 200 at its private session URL. Default dedicated trial ports are **7792** (Identity) and **7862** (Processor). It does not interrupt existing processes or reuse Alpha profiles. It does not silently switch branches or advance Identity commits.

Browser opening is best-effort: in WSL it tries PowerShell/Windows; on Linux, `xdg-open`. If unavailable, **locally** inspect the `open :` line in `~/p1c/p1-launcher/processor-private.log` and paste the entire link into your own browser. The URL contains a private fragment: **never post it on GitHub, in a CI log, or in screenshots.** Identity's pairing code is visible only in its local `identity-private.log`; enter it in Processor to pair, then complete human GitHub authorization and Identity approvals as described in [P1 START_HERE](docs/P1/START_HERE.md).

**Stop:** Ctrl+C in the launcher terminal. Only its own Identity and Processor child PIDs receive SIGTERM. The profiles persist for a restart. If a service fails, inspect the *local* private logs and do not paste them into issues. A stale `.launcher-running` folder is intentionally not removed automatically; verify no previous run remains before removing it. A persisted Identity `runtime/service.json` is not by itself a reason to refuse a clean restart; the launcher lock, port preflight, and Identity service profile checks arbitrate the restart. Restart on the same ports and work path; a fresh private Processor session URL may be generated. Verify reuse in an actual runtime test.

**Choose unused ports:** `P1_IDENTITY_PORT=7793 P1_PROCESSOR_PORT=7863 bash ~/p1c/Thought-Processor/run-p1.sh`. Other settings: `P1_IDENTITY_SRC=/path/to/identity` and `P1_LAUNCH_WORK=/private/local/path`. Keep the same ports after pairing to avoid unexpected reconnection behavior. Use a new `P1_LAUNCH_WORK` to isolate a second trial; never point at active profiles. Do not share profile data, logs, pairing/device codes, `#code=`, `#key=`, or `/approve#` fragments.

**Scope and evidence:** This implements a launch wrapper, not proof of pairing, auth, feature run, or owner acceptance. It does not touch `alpha`, `stable`, or `main`; does not install third-party workers; and uses the pinned test fixture from the P1 cleanup baseline. Grok Linux and Dustin WSL checks remain independent.

## Controlled negative checks (Test Architect rev 2)

Only use **disposable** work directories under `~/p1c/` and free test ports. These hooks simulate failure edges; they do not assert real operating-system failures or real Go compiler failures. They never automate pairing or print private URLs. Capture only the exit status and sanitized one-line error; do not publish raw service logs.

```bash
# Prerequisite refusal simulations (no source or work profile touched):
P1_TEST_INJECT=missing-go bash ~/p1c/Thought-Processor/run-p1.sh
P1_TEST_INJECT=missing-python3 bash ~/p1c/Thought-Processor/run-p1.sh

# Build refusal simulation, after source checks:
P1_TEST_INJECT=go-build P1_LAUNCH_WORK="$HOME/p1c/inject-build" bash ~/p1c/Thought-Processor/run-p1.sh

# Spawn failure, after successful real builds; distinct disposable profiles:
P1_TEST_INJECT=identity-start P1_LAUNCH_WORK="$HOME/p1c/inject-id" bash ~/p1c/Thought-Processor/run-p1.sh
P1_TEST_INJECT=processor-start P1_LAUNCH_WORK="$HOME/p1c/inject-tp" bash ~/p1c/Thought-Processor/run-p1.sh
```

All injected runs must refuse nonzero; neither spawned-service case may leave launcher-owned children running. The `processor-start` case starts a **real disposable Identity child** before simulating a Processor exit, so use an otherwise idle host. Do not manually terminate unrelated processes.

For actual missing-tool failures, execute in a separate disposable test environment where the executable is absent; for actual Go build errors, inject a failing `go` command through a disposable isolated toolchain or runner without changing source. The hooks above test launcher error-handling but are **not** substitutes for those real-world cases.

**Active distinct-process lock test:** Terminal A launches the normal command with `P1_LAUNCH_WORK="$HOME/p1c/active-lock"`, and remains running. Terminal B attempts the exact same command with the same work path, and should fail at port preflight or launcher lock; terminal A must stay alive. Stop terminal A by Ctrl+C only after observing refusal. This checks both ownership and non-interference, not just a manufactured lock directory.

**Restart test:** Start and Ctrl+C the normal launcher with `P1_LAUNCH_WORK="$HOME/p1c/restart"`; run it again with the same work path and ports. Record readiness and whether persistent profiles were reused. If Identity refuses a genuinely stale runtime marker, classify as FAIL and report its error type without publishing private logs. Do not delete profiles to manufacture a PASS.

**Known limits:** The injections are deliberately test-only and must never be used with a personal workspace. The project branch SHA changes when fixes are committed: Test Architect must update the pinned revision before Grok or Dustin reruns. Neither independent Linux nor WSL runtime checks have been executed by Engineering in this connector environment.
