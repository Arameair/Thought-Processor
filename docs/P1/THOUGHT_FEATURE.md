# Thought plugged into Thought Processor (P1, 2026-10-29)

| Repo | Branch | Commit |
| --- | --- | --- |
| Arameair/Thought-Processor | `experiment/p1-thought-feature-claude-20261029` (on top of the secure worker runtime, `8af218d`) | the Thought-Processor commit SHA given in the delivery report and at the top of PR #28 |
| Arameair/Thought | `experiment/thought-feature-p1-claude-20261029` | `772200b` |
| Arameair/identity | `experiment/p1-cleanup-claude-20261026` | `64940b6` (unchanged) |

**What it is:** Thought (Spark → Trace → Thought) runs as a Thought Processor feature, like the P1 Deterministic Worker. It is installed through Identity's Admin approval and run by Processor. Every use is checked live with Identity, for one identity and one capability at a time. The rules, capabilities and inputs are described in Thought's [`docs/THOUGHT_FEATURE.md`](https://github.com/Arameair/Thought/blob/experiment/thought-feature-p1-claude-20261029/docs/THOUGHT_FEATURE.md).

## What changed in Thought Processor

- **`run-p1.sh` works from ordinary clones and from Git worktrees.** It used to require `.git` to be a folder; in a worktree it is a file, so the launcher refused it. Now Git itself answers the checks, the same way for both checkout types:
  - the folder is the top of a working checkout (not a subfolder, not a broken worktree link, not bare);
  - its `origin` is the expected Arameair repository;
  - the branch is right;
  - the commit is exact (Identity) or descended from the reviewed commit (Processor, Thought);
  - the tree is clean.

  Nothing was loosened.
- **Thought checks before anything starts.** When a Thought folder is next to the other repos, it must pass the same checks. It is never skipped silently. Its package must also hash to the pinned artifact `8f92ee685c99851128f322c9063ff151e28d7ccf48709315d8857fc56e8079ec` before anything is built. Processor (`-trust-feature-artifact`) and Identity (`-trust-fixture-artifact`) then both pin it. No Identity code was changed.
- **`P1_PREFLIGHT_ONLY=1 bash ./run-p1.sh`** runs only those checks plus the port check. It builds, starts and creates nothing.
- **Feature page.** The request form shows what input the chosen capability expects. A feature's refusal reason is shown in the result.

## Automated tests (Linux)

| Test | What it proves |
| --- | --- |
| `acceptance/launcher_checkouts.py` | launcher from clones **and** worktrees, 7 refusal cases, one full launch from worktrees |
| `acceptance/p1_profile_upgrade.py` | the accepted secure-worker profile opens unchanged in this build; Thought installs beside it; rolling back works |
| `acceptance/p1_thought_feature.py` | the 14 Thought service checks, with real Identity + Processor + Thought + worker |
| `p1_worker_runtime.py`, `p1_integration.py`, `p1_findings.py`, `p1_ui_tour.py`, unit tests | everything that was already there still passes |

Results are in the delivery report and PR #28.

## Owner test on Windows + WSL

**What this keeps safe:**
- `~/p1c/tp-worker-clone` is only read, never changed.
- Your secure-worker trial profile `~/p1c/p1-worker-runtime` (ports 7794/7864) is **reused, after a backup**. Your Admin, Tass, the worker installation and its grants stay as they are; this exact upgrade is tested by `p1_profile_upgrade.py`.
- The original P1 environment (`~/p1c/p1-launcher`) is never touched.
- Nothing asks you to delete an installation or reset the Admin.

**Never share:** the pairing code, the GitHub device code, anything after `#code=`, `#key=` or `/approve#`, your Admin passphrase, `credential.json`, or the `*-private.log` files. **To report a problem,** copy only the `ERROR:` line from the terminal, or the red result text from the page. Neither contains secrets.

Run one command at a time in a WSL terminal and compare against **Expected** before moving on.

### Step 1: check the layout (reads only)

```bash
git -C ~/p1c/identity rev-parse HEAD; ls -d ~/p1c/tp-worker-clone ~/p1c/p1-worker-runtime; ls ~/p1c/p1-worker-runtime/.launcher-running 2>&1 | tail -1
```

**Expected:**
- `64940b629a5ef52eaa143b2402726ec81d382002`
- the two folder names
- a line ending in `No such file or directory`, which means no launcher is running.

**FAIL / fix:**
- A different SHA, or `identity` not found: stop and report it.
- `.launcher-running` exists: a trial is still running. Press **Ctrl+C** in its terminal, then repeat.

### Step 2: get Thought (new folder; private repo, so Git may ask you to sign in)

```bash
git clone --branch experiment/thought-feature-p1-claude-20261029 https://github.com/Arameair/Thought.git ~/p1c/Thought && git -C ~/p1c/Thought rev-parse HEAD
```

**Expected:** `772200b5bc29efce6da3b0e064fb6714570da1d4`

### Step 3: get this Processor build (new folder, a normal clone)

```bash
git clone --branch experiment/p1-thought-feature-claude-20261029 https://github.com/Arameair/Thought-Processor.git ~/p1c/tp-thought-clone && git -C ~/p1c/tp-thought-clone rev-parse HEAD
```

**Expected:** the Thought-Processor commit SHA given in the delivery report and at the top of PR #28

A worktree of `tp-worker-clone` would also work now, but a normal clone keeps `tp-worker-clone` completely untouched.

### Step 4: back up the worker trial profile (copy only)

```bash
cp -a ~/p1c/p1-worker-runtime ~/p1c/p1-worker-runtime.backup-20261029 && du -sh ~/p1c/p1-worker-runtime.backup-20261029
```

**Expected:** a size line, no error. The backup holds private keys, so keep it on this computer and don't share it.

### Step 5: preflight (checks only; builds and starts nothing)

```bash
cd ~/p1c/tp-thought-clone && P1_PREFLIGHT_ONLY=1 bash ./run-p1.sh
```

**Expected** (paths show your home folder):

```
Identity P1 source: 64940b629a5ef52eaa143b2402726ec81d382002 (…/p1c/identity)
Processor P1 launcher: <the commit SHA from the delivery report> (base db31e4785d292925249a2f9f06e3c06bd78ba5b0) (…/p1c/tp-thought-clone)
Thought feature source: 772200b5bc29efce6da3b0e064fb6714570da1d4 (…/p1c/Thought) — package matches pinned artifact 8f92ee685c99…
Trial folder: …/p1c/p1-worker-runtime (existing profiles will be reused). Ports 7794/7864 are free.
PREFLIGHT OK — nothing was built or started.
```

**FAIL:** any line starting `ERROR:`. Report that line; it names the folder and the check that failed.

### Step 6: start the trial

```bash
bash ./run-p1.sh
```

**Expected:**
- The same first three lines as step 5, then `P1 Processor ready on 127.0.0.1:7864. Identity trial port 7794.`
- Your browser opens Thought Processor. If it doesn't, copy the private link from **this terminal only**.
- **No pairing code is needed.** The chip turns green, *Identity: connected*, on its own.

Leave this terminal open.

### Step 7: in the browser (PASS only if you see the Expected result)

| # | Do | Expected |
| --- | --- | --- |
| 7.1 | Look at the left list | **Admin Identity**, **Tass**, and *P1 Deterministic Worker* under Installed features, with the **same `ins_…` start** as in your worker test |
| 7.2 | **Add a feature…** | Worker card: **Open installed feature**. **Thought (Spark → Trace → Thought)** card: badge **pinned artifact**, button **Install…** |
| 7.3 | Thought → **Install…** → **Install** → open the Identity approval page → passphrase → **Approve** | *Thought* appears under Installed features with its **own** `ins_…` (different from the worker's) *(items 1, 2)* |
| 7.4 | **Add a feature…** again | Thought now says **Open installed feature**, **installed ×1** *(item 3)* |
| 7.5 | Thought → Overview → **Start** | **Running · pid … · health READY** *(item 4)* |
| 7.6 | In a second WSL terminal: `pgrep -af 'p1-worker-runtime/processor-profile/installations/.*/thought_feature.py'` | exactly **one** line *(item 4)* |
| 7.7 | **Run self-test** | green **Self-test PASS**, 8 checks all `"ok": true` *(item 5)* |
| 7.8 | Connections: Tass, `thought.event.submit`, input `goldshire: the player went back to Goldshire`, **Send request** | **DENIED BY IDENTITY**, `NO_GRANT` *(item 6)* |
| 7.9 | *Request a connection*: Tass, `thought.event.submit`, **Ask the Admin…**, approve in Identity | the grants table shows it **ACTIVE** |
| 7.10 | Send the 7.8 input **three times** | **COMPLETED** each time; outcome `trace_kept`, `trace_kept`, then **`thought_formed`**, with headline *A Thought formed: Why does "goldshire" keep coming up?* and `evidence_count: 3` *(items 7, 8)* |
| 7.11 | Capability `thought.thoughts.read`, empty input, **Send request** | **DENIED BY IDENTITY** *(item 9)* |
| 7.12 | Ask the Admin for `thought.thoughts.read` for Tass, approve, send again | **COMPLETED**: one Thought, `why_it_formed`, 3 events marked `ORIGIN` *(item 9)* |
| 7.13 | **+ Identity** `Other` (person). Ask the Admin for `thought.thoughts.read` for **Other**, approve, then read as Other | **COMPLETED** with `"thoughts": []`. Other sees none of Tass's Thoughts *(item 10)* |
| 7.14 | Revoke Tass's `thought.event.submit` in the grants table; send an event for Tass | **DENIED BY IDENTITY**, `GRANT_REVOKED`. A read for Tass still works *(item 11)* |
| 7.15 | Overview → **Stop**; rerun the 7.6 `pgrep` | card says **Stopped**; `pgrep` prints nothing *(item 12)* |
| 7.16 | The worker is unaffected: worker → **Start**, Connections → Tass → **Send request** | **COMPLETED** if your worker grant is still active. If it ran out (grants last 1 hour by default) you'll see **DENIED**, `GRANT_EXPIRED`. That is correct; ask the Admin again to re-test |

### Step 8: restart keeps everything *(item 13)*

1. Press **Ctrl+C** in the trial terminal.
   **Expected:** the launcher prints nothing alarming and the prompt returns. The 7.6 `pgrep` prints nothing.
2. Run:
   ```bash
   bash ./run-p1.sh
   ```
3. Open Thought, press **Start**, then read for Tass.
   **Expected:** the **same** Thought (same `th_…`) and the same `ins_…` IDs.

Item 14 (Identity unavailable, crash, tampered file) is covered by the automated tests and is not part of this walkthrough.

### Report

Paste back:
- the SHAs from steps 1–3;
- the step 5 output;
- **PASS / FAIL / NOT RUN** for 7.1–7.16 and step 8;
- any `ERROR:` lines or red result text.

### Shut down and clean up

- **Stop:** press **Ctrl+C** in the trial terminal. It stops only what this launcher started.
- **Go back to the worker-only build:**
  ```bash
  cd ~/p1c/tp-worker-clone && bash ./run-p1.sh
  ```
  Same profile; tested by `p1_profile_upgrade.py`. Thought stays listed there but won't start without its pin.
- **Undo this trial completely**, with the launcher stopped:
  ```bash
  rm -rf ~/p1c/p1-worker-runtime && mv ~/p1c/p1-worker-runtime.backup-20261029 ~/p1c/p1-worker-runtime
  ```
  This restores the profile exactly as it was before step 6.
- **Optional:** remove the code folders with `rm -rf ~/p1c/Thought ~/p1c/tp-thought-clone`. That deletes code only, no profiles.

## Known limits

- **No dedicated Thought screen yet.** Results show as the raw observation in the request box (headline, steps, Thought, evidence).
- **Nothing calls Thought automatically yet.** TASS, Thought Controller and Thought Engine aren't connected. Any program that pairs with Identity and gets a grant can use the same three capabilities.
- **Same P1 limits as the worker:** no OS isolation, a passphrase authenticator, and loopback HTTP that is authenticated but not encrypted.
- **A Thought belongs to one identity on one Thought installation.** A second Thought installation in another Processor profile has its own, separate Thoughts.
