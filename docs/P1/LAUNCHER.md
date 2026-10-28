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

**Stop:** Ctrl+C in the launcher terminal. Only its own Identity and Processor child PIDs receive SIGTERM. The profiles persist for a restart. If a service fails, inspect the *local* private logs and do not paste them into issues. A stale `.launcher-running` folder is intentionally not removed automatically; verify no previous run remains before removing it.

**Choose unused ports:** `P1_IDENTITY_PORT=7793 P1_PROCESSOR_PORT=7863 bash ~/p1c/Thought-Processor/run-p1.sh`. Other settings: `P1_IDENTITY_SRC=/path/to/identity` and `P1_LAUNCH_WORK=/private/local/path`. Keep the same ports after pairing to avoid unexpected reconnection behavior. Use a new `P1_LAUNCH_WORK` to isolate a second trial; never point at active profiles. Do not share profile data, logs, pairing/device codes, `#code=`, `#key=`, or `/approve#` fragments.

**Scope and evidence:** This implements a launch wrapper, not proof of pairing, auth, feature run, or owner acceptance. It does not touch `alpha`, `stable`, or `main`; does not install third-party workers; and uses the pinned test fixture from the P1 cleanup baseline. Grok Linux and Dustin WSL checks remain independent.
