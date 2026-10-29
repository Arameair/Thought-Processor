# P1 secure worker runtime — what changed and how to test it (2026-10-29)

**Branch:** `experiment/p1-secure-worker-runtime-claude-20261029` (from `experiment/p1-launcher-gpt6-20261028` @ `45a6af3`).
**Identity:** unchanged — `experiment/p1-cleanup-claude-20261026` @ `64940b6`. No Identity code was modified.
**Experimental only.** Nothing merged; `main`, `alpha`, `stable` and the earlier P1 branches are untouched.

## What the owner hit, and the cause

| Seen | Cause in the code |
| --- | --- |
| Processor allowed two installations of the same feature | `requestInstall` minted a new random install key on every click and never looked for an existing installation of the same artifact. (The earlier P1 suite even expected a second click to create a second copy.) |
| Couldn't tell how to start or use the worker | Start/Stop and the self-test were only on the third tab (**Runtime**). The self-test result was written into a block the page redraws every 2 seconds, so it disappeared almost at once. |

## What changed (Thought Processor only)

1. **Idempotent install (default).** Installing the same pinned artifact into the same workspace (this Processor profile, paired with this Identity) returns the existing installation. Processor checks Identity's live list, not a cache. A second click while an install is still waiting for the Admin returns *that* install. Install requests are serialized, so simultaneous clicks can't race. A deliberate second copy needs `{"new_copy": true}` on the API (not offered in the page). Nothing existing is deleted; a workspace that already has extra copies shows a warning on each one.
2. **Start fails closed.** A start now needs Identity to have confirmed, on a verified connection, that this exact installation is `INSTALLED` with the same artifact digest. If Identity is unreachable: `AUTHORITY_UNAVAILABLE`, nothing starts. Always-running features also wait for that confirmation. The existing checks still run before every start: pinned artifact, files re-hashed, the interpreter recorded at install.
3. **One process per installation.** Already true in the supervisor; now explicit. A repeated Start reports `already_running` with the same PID and launch and starts nothing. The browser request no longer owns the start, so closing the tab mid-start can't leave a half-started process.
4. **Clear states on the first tab.** The Overview tab opens with a **Run** card: *Stopped / Starting… / Running / Stopping… / Error*, the one button that applies (**Start**, **Stop**, **Start again**), and **Run self-test**.
5. **Self-test result kept.** Processor stores the last self-test per installation (PASS / FAIL / ERROR, time, launch ID, output) from the worker's launch-signed answer, and the page shows it from there. It survives refreshes. A non-PASS answer is shown as FAIL, never as success.
6. **Failed starts handled safely.** Fixed a race where a start that failed and was killed could be recorded as a clean **STOPPED** instead of **FAILED_TO_START**. The launch-key file is now removed on every failed-start path.
7. **Redacted logs.** The log scrubber now also masks approval-link tokens (`…&t=`), request tickets (`tk1.…`) and `secret` / `launch_key` / passphrase / auth fields. It is also applied to the worker's own log before the page shows it. IDs and SHA-256 digests stay visible.
8. **`run-p1.sh` on this branch** uses its own work folder `~/p1c/p1-worker-runtime` and ports **7794 / 7864**. Your existing trial in `~/p1c/p1-launcher` (7792 / 7862) is never reused or contacted.

Unchanged on purpose: the worker (`worker.py` is a pinned artifact; any edit would need re-pinning in Identity), Identity's authorization model (Processor still can't approve anything; Admin approvals happen only in Identity's own page), tickets, live checks, and revocation.

## Automated tests

| Command (from the repo root, Identity clone at `../identity`) | What it covers |
| --- | --- |
| `cd processor && go vet ./... && go test -race -count=1 ./...` | unit tests, incl. new redaction test |
| `python3 -I processor/acceptance/p1_worker_runtime.py --identity-src ../identity --work ~/p1c/run-worker-runtime` | **new**: the 12 acceptance items, 33 checks, real processes, OS-level process count |
| `python3 -I processor/acceptance/p1_integration.py --identity-src ../identity --work ~/p1c/run-integration` | existing 81 checks (second copy now asked for explicitly) |
| `python3 -I processor/acceptance/p1_findings.py --identity-src ../identity --processor-src . --work ~/p1c/run-findings` | existing Grok findings, 15 attacks |

These run on Linux. They use OS-chosen ports and private folders and never touch 7790/7792/7794/7860/7862/7864.

## Owner test on Windows + WSL (about 15 minutes)

You need the same WSL setup as before: Git, Go 1.22+, Python 3. This uses a **separate folder and fresh profiles**, so you will set up a new Admin passphrase for this trial. Your current P1 trial is not touched.

**Never share:** the pairing code, the GitHub device code, anything after `#code=`, `#key=` or `/approve#`, the Admin passphrase, or any `credential.json`.

### 1. Get the branch next to your existing clones (WSL terminal)

Go to the folder that holds your `identity` and `Thought-Processor` clones, then:

```bash
cd Thought-Processor
git fetch origin experiment/p1-secure-worker-runtime-claude-20261029:experiment/p1-secure-worker-runtime-claude-20261029
git worktree add ../tp-worker-runtime experiment/p1-secure-worker-runtime-claude-20261029
git -C ../tp-worker-runtime log -1 --format='%h %s'
```

**Expected:** a new folder `tp-worker-runtime` next to `Thought-Processor`, and the commit line from the delivery report. Your `Thought-Processor` checkout stays on its branch, with its files unchanged.

### 2. Start the trial

```bash
cd ../tp-worker-runtime
bash ./run-p1.sh
```

**Expected:**
- The terminal prints `Identity P1 source: 64940b6…`, then the Processor commit.
- Then `P1 Processor ready on 127.0.0.1:7864. Identity trial port 7794.`
- A private pairing challenge appears in this terminal.
- Your Windows browser opens Thought Processor. If it doesn't, copy the private link from **this terminal only** into your browser.

Leave this terminal open.

### 3. Connect, Admin, identity (same as your last trial)

1. **Connect**, then type the code from the terminal. **Expected:** the chip turns green, *Identity: connected*.
2. Admin: **Sign in to GitHub**, then **Establish Admin Identity…**. In Identity's page, choose a passphrase, tick the confirmation, and establish.
3. **+ Identity:** add `Tass` (companion).

### 4. Install once — then try to install again

1. Go to **Add a feature…**, click **Install…** on the *P1 Deterministic Worker*, then **Install**. Open the Identity approval page, enter your passphrase, and **Approve**.
   **Expected:** *P1 Deterministic Worker* appears under **Installed features** on the left.
2. Go back to **Add a feature…**.
   **Expected:** the card's button now says **Open installed feature** (no *Install…*) and shows the badge **installed ×1**.
   Clicking it opens the same installation.

### 5. Start, check the process count, self-test

1. Click the feature. The **Overview** tab opens with a **Run** card that says **Stopped**, with a **Start** button.
2. Press **Start**.
   **Expected:** **Starting…**, then **Running · pid … · launch lch_… · health READY**. Start is replaced by **Stop**.
3. Confirm exactly one worker process. In a **second** WSL terminal:
   ```bash
   pgrep -af 'p1-worker-runtime/processor-profile/installations/.*/worker.py'
   ```
   **Expected:** exactly **one** line.
4. Press **Run self-test**.
   **Expected:** a green **Self-test PASS** with the time and the same launch ID, plus *external calls: 0* and output like `"upper": "P1 SELF-TEST"`. It is **still there 10 seconds later.**

### 6. Authorization

On the **Connections** tab:

1. **Send request** (Input `hello from P1`).
   **Expected:** red **DENIED BY IDENTITY** — *NO_GRANT — nothing was started or sent*.
2. **Ask the Admin…** (1 hour), then approve in Identity's page. Send again.
   **Expected:** green **COMPLETED**, with output `"upper": "HELLO FROM P1"`.
   Re-run the `pgrep` from step 5: still **one** line.
3. **Revoke**, then send again.
   **Expected:** **DENIED BY IDENTITY** — *GRANT_REVOKED*.

### 7. Stop

On **Overview**, press **Stop**.

**Expected:** the card says **Stopped**, and the `pgrep` command prints **nothing**.

### 8. Restart keeps the same installation

1. In the first terminal, press **Ctrl+C**. **Expected:** both programs stop.
2. Run `bash ./run-p1.sh` again and open the page.
   **Expected:**
   - It reconnects **without a new code**.
   - The same feature is listed, with the same `ins_…` start on the left.
   - **Add a feature…** still shows **Open installed feature**.
3. **Start**, then **Run self-test**.
   **Expected:** **Running** and **Self-test PASS**, and `pgrep` shows one line.

### Report

For each step, report **PASS / FAIL / NOT RUN** with what you saw, plus the commit line from step 1.

Identity-outage and tampered-file behaviour are covered by the automated test. They aren't part of this walkthrough, because `run-p1.sh` runs both programs from one terminal.

### Clean up (optional)

1. Press **Ctrl+C** in the trial terminal.
2. Remove the trial folders:
   ```bash
   rm -rf ~/p1c/p1-worker-runtime
   git -C ../Thought-Processor worktree remove ../tp-worker-runtime
   ```
   Run the second command from inside `tp-worker-runtime`'s parent, or adjust the paths.

## Known limits (unchanged from P1 unless noted)

- **HIGH — no OS isolation.** A pinned worker runs with your user's rights. Any program running as you can read installation credentials or call loopback ports. Pins stop *unreviewed packages*, not hostile local code.
- **HIGH — passphrase authenticator.** It can be guessed offline or key-logged by same-user malware. Recovery is still deferred.
- **MEDIUM — new:** the start gate uses Identity's last signed installation list on a verified connection. That list refreshes every few seconds and on every change Processor makes. A grant change never waits for it, because every privileged request is still checked live by the worker.
- **MEDIUM — new:** "same workspace" means this Processor profile paired with this Identity. A second Processor profile can still install its own copy, after its own Admin approval.
- **LOW — new:** the last self-test result is kept in memory for this Processor run, not on disk. After a restart it shows *not run yet* until you run it again.
- **LOW:** loopback HTTP is HMAC-authenticated but not encrypted.
- Extra copies made before this change are kept and flagged on each copy's page; uninstall them yourself (Admin-approved) if you don't want them.
