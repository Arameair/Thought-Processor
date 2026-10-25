> **Branch `alpha-candidate-20261025`** — the working Processor diagnostic shell (code unchanged from `experiment/processor-controller-launch-fix-20261024` @ `0ba8bd5`, which ran for real on 2026-10-25), plus the 2026-10-25 discussion notes. The owner decides whether it becomes `alpha`.
>
> **Run it with Identity:** Processor on port **7860**, Identity (`Arameair/identity`, branch `alpha-candidate-20261025`) on port **7790**. Step-by-step for people and AI operators: [Identity's AGENTS.md](https://github.com/Arameair/identity/blob/alpha-candidate-20261025/AGENTS.md).
>
> **Discussion notes (nothing decided, no code):** [Authority to install the Thought Engine](docs/INSTALL_AUTHORITY_DISCUSSION_2026-10-25.md) (start here) · [Design ideas and approaches to look at](docs/DESIGN_IDEAS_TO_LOOK_AT_2026-10-25.md) · [The actual thought process](docs/THOUGHT_PROCESS_DISCUSSION_2026-10-25.md)

# Thought Processor — Project Controller launch trial

> **Thought Processor controls the operational setup of thinking. Thought Engine controls the actual act of thinking.**

**Status:** Experimental diagnostic shell — operational setup/metrics/flow only. Identity-authorized start/attach/stop, private Engine status and observation ingestion are **BLOCKED**, not simulated. No branch promotion authorized.

## Run this experiment in Project Controller

**Branch:** `experiment/processor-controller-launch-fix-20261024`

Open this branch in your existing Project Controller. The **repository-root `marquee.json`** provides its setup/run plan for Linux/WSL and Windows. Check and approve the actual commands displayed by Controller. If this project has old manually configured Setup/Run commands, clear the overrides first: they take precedence over `marquee.json`.

Controller builds Go from `processor/` and uses its assigned `PORT` and per-branch `PC_DATA`. **Use Open ↗ / New Tab**, not embedded iframe: this application intentionally sets `frame-ancestors 'none'` and `X-Frame-Options: DENY`.

### Sign in securely while running under Controller

The **one-time local Processor sign-in code is not printed to Controller logs** in managed mode. This avoids accidental exposure through Copy Logs/Copy Context. Opening the new tab shows a sign-in form. Retrieve the code from the local file path shown in Processor's startup log: `<profile>/runtime/operator-code` (read it **locally**; never paste it into GitHub/ChatGPT or share the raw file). On Linux/WSL, from your own terminal:

```sh
cat '<exact-profile-path-shown-in-Processor-log>/runtime/operator-code'
```

Paste the code **only into the local Processor sign-in page**. It changes every time Processor restarts. This explicit file handoff is a temporary usability trade-off until an approved Controller↔Processor secure session handoff exists.

### What to test

1. Controller **Play** → verify approved plan uses `processor/`, assigned unique port/profile, and reports the actual Git SHA.
2. **Open ↗** → login using the protected local code file (not captured stdout).
3. Empty first-use state: no fake connected Identity/Engine; one next action.
4. Enter an unavailable Identity port; it must say **Unreachable**, not connected.
5. Optional real Identity endpoint → **Found, unverified**; Engine endpoint → **Port open, unidentified**.
6. **Flow / Logs / Metrics / Copy Context** show only real Processor activity. Protected start/attach/stop/observe must remain **BLOCKED** (HTTP 501).
7. Stop Processor in Controller; preexisting Identity/Engine keep running. Restart; verify no duplicate processes, isolated profile and changed operator code. **Do not share raw secret codes.**

### What is *not* proven

This branch **does not** add Identity authority or trust, authenticated Engine status, real thinking, model calls, secure Engine lifecycle management, dynamic feature catalog or completed guided observation test. It is for the **first owner operational-shell trial** only.

### Source provenance and branch policy

This implementation is **copied as Git objects** from the reported cloud-tested `experiment/processor-first-prototype-20261024` tested application SHA `db2a46c5bf930239f286bf412872c67953946d8b` (later candidate head `378cdf10947c6ddbd7bf3ac3b613f728fd4fd733` contains evidence/doc/test-script changes). This is a **new experiment with parent `alpha` at `9fac9fa44cb020bb6f8aa6bcb1e9faeda6e0b47a`**, not an unapproved Alpha merge. Changes relative to reported tested program: the Controller manifest, secure managed-start output, and additional startup tests. **This new branch has not been run in the owner's Controller yet.**

Owner-approved release lanes remain separate. No automatic promotion.
