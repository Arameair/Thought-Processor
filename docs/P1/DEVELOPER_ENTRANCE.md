# Developer entrance: runnable prototype (2026-10-29)

**Branch:** `Arameair/Thought-Processor` · `experiment/developer-entrance-claude-20261029`. The exact commit is in the delivery report and at the top of its draft PR.

- **Based on:** PR #28 (`a1acd8f`: secure worker runtime, worktree-safe launcher, Thought), plus the team's Developer Utility adapter (`79ecd77`, merged unchanged).
- **Identity:** unchanged at `64940b6`. No Identity code or policy changed.
- **Not merged anywhere.** `main`, `alpha`, `stable`, PR #27, PR #28 and the team's branch are untouched.

## The vertical slice

**Developer → Thought Processor → Admin Identity authorization → approved Developer operation → verifiable status → revocation → secure shutdown.**

| Part | Role in this slice |
|---|---|
| **Identity / Admin** | Approves the exact pinned Developer installation. Grants `developer.connection.check` to one identity at a time, checked live on every request. Revokes. Holds every credential. |
| **Thought Processor** | Lists the reviewed package, verifies its digest (in the launcher, at install, and before every start), asks Identity, installs, starts and stops it, routes the one operation with an Identity ticket, and records activity. |
| **Developer Utility** | An independently installed service with **one** read-only operation. No shell, no file access outside its data folder, no route to other services. |
| **Thought, the worker** | Unaffected. Each has its own grants; a Developer grant gives nothing on them. |

**What "connected" means here.** Processor talks to the Developer service only with a per-launch signing key. The service itself checks every 5 s that Identity signs an answer about **its** installation. Those two facts drive the screen:

- **RUNNING:** Processor verified the service's launch-signed status.
- **CONNECTED:** Identity's signed answer about this installation is less than 15 s old.
- **READY:** both of the above.

If Identity goes away, the screen turns **NOT CONNECTED / NOT READY** on its own. There is no green state without evidence.

**The safe operation:** `developer.connection.check`. Only if Identity allows it live, the service fetches a fresh signed answer from Identity and reports:
- its own state;
- both links;
- the grants Identity says it holds;
- who asked;
- what it **cannot** do.

It takes no input; any input is refused.

## What changed on this branch

- `processor/developer-utility/`:
  - the service (team adapter plus the reviewed ticket, ledger and live-check protocol, the Identity-link monitor, one capability, and a self-test);
  - its manifest (exact digest `9d33ed8bfccff317aeec1914718e32c3691845a5a2dac33b57d344487001b39d`);
  - 8 protocol tests;
  - README.
- **`run-p1.sh`:**
  - its own folder `~/p1c/p1-developer-entrance` and ports **7796 / 7866**;
  - the Developer package must hash to the pinned digest before anything builds;
  - the digest is pinned in both programs.

  Clone/worktree support, preflight and the Thought pin carry over from PR #28.
- **Thought Processor UI**, on the existing feature page:
  - **Overview:** RUNNING / CONNECTED / READY chips with the Identity-link evidence, **Access** (who is allowed), **one Next action**, a one-click **Run connection check for …** button, plain-language reasons for denials, the answer's headline and facts (raw answer under *Full answer*), and **Copy activity**.
  - **Copy activity:** Processor's redacted record of this service, produced by the server.
- **Wording:** the success text no longer says "SYNTHETIC TEST DATA" for every service; each service's own data class is shown.
- **Tests:** `acceptance/p1_developer_entrance.py` (39 checks, including a real-browser check of the screen).

## Trust decision for Dustin (no Identity change)

The team's blocker was the package label. Identity accepts only `trust_class: TRUSTED_LOCAL_TEST_FIXTURE` **plus** an exact pinned digest. Since the P1 cleanup the label grants nothing. Trust comes from:
1. the exact digest you pin with the existing `-trust-fixture-artifact` flag (`run-p1.sh` does this for you);
2. the Admin's approval of that exact install;
3. a live grant for every request.

This is the same mechanism Thought already uses. The test shows Identity still refuses an unknown digest, and still refuses the old `OWNER_REVIEW_REQUIRED` label even with the right digest. **Nothing was loosened.**

**Your decision:** accept that this reviewed Developer package is admitted under Identity's existing "trusted local package by exact pin" class. That class is named *test fixture* today.

*Optional, later, and not done:* rename that class in Identity, e.g. to `TRUSTED_LOCAL_PINNED`, so the label reads honestly. That would be a small Identity change needing your explicit approval. It is not needed for this prototype.

## Automated results

Exact numbers for the delivered commit are in the delivery report and the PR. The suites:

| Test | What it proves |
|---|---|
| `processor/developer-utility/test_utility_agent.py` | 8 protocol checks (signatures, replay, unknown capability, ticket required, honest not-ready) |
| `acceptance/p1_developer_entrance.py --shots DIR` | the whole slice with real processes and a real browser (39 checks) |
| `p1_thought_feature.py`, `p1_worker_runtime.py`, `p1_integration.py`, `p1_findings.py`, `launcher_checkouts.py`, `p1_profile_upgrade.py`, `p1_ui_tour.py`, Go unit tests | nothing that worked before is broken |

## Owner walkthrough (Windows + WSL)

**What this keeps safe:**
- A **new** clone `~/p1c/tp-developer-clone`, folder `~/p1c/p1-developer-entrance` and ports 7796/7866.
- `~/p1c/p1-launcher`, `~/p1c/p1-worker-runtime`, `~/p1c/tp-worker-clone`, `~/p1c/tp-thought-clone` and their profiles are **never touched**. The launcher refuses the ports if they're busy, and stops nothing it didn't start.
- **Fresh profiles:** you'll pair once, sign in to GitHub and establish an Admin for this trial. Your other Admins are untouched.

**Never share:** pairing codes, the GitHub device code, anything after `#code=`, `#key=` or `/approve#`, your passphrase, `credential.json`, or `*-private.log`. **To report a problem,** copy only the `ERROR:` line, red result text, or the **Copy activity** text. None of these contain secrets.

One command at a time, in a WSL terminal.

### 1. Check the layout (reads only)

```bash
git -C ~/p1c/identity rev-parse HEAD; ls -d ~/p1c/p1-developer-entrance ~/p1c/tp-developer-clone 2>&1 | tail -2
```

**Expected:**
- `64940b629a5ef52eaa143b2402726ec81d382002`;
- two lines ending in `No such file or directory`, which means this trial is new.

### 2. Get the code (new folder)

```bash
git clone --branch experiment/developer-entrance-claude-20261029 https://github.com/Arameair/Thought-Processor.git ~/p1c/tp-developer-clone && git -C ~/p1c/tp-developer-clone rev-parse HEAD
```

**Expected:** the commit SHA from the delivery report.

### 3. Preflight (checks only; builds and starts nothing)

```bash
cd ~/p1c/tp-developer-clone && P1_PREFLIGHT_ONLY=1 bash ./run-p1.sh
```

**Expected:**
- `Identity P1 source: 64940b6…`.
- `Processor P1 launcher: <the SHA> …`.
- `Developer Utility package: matches pinned artifact 9d33ed8bfccf…`.
- A Thought line: `…package matches pinned artifact 8f92ee685c99…` if `~/p1c/Thought` is on its branch, or `Thought feature: not offered…`. Either is fine.
- `Trial folder: …/p1c/p1-developer-entrance (no profiles yet (fresh trial)). Ports 7796/7866 are free.`
- `PREFLIGHT OK — nothing was built or started.`

**FAIL:** any `ERROR:` line. Report it as is.

### 4. Start the trial

```bash
bash ./run-p1.sh
```

**Expected:**
- `P1 Processor ready on 127.0.0.1:7866. Identity trial port 7796.`
- The browser opens. If it doesn't, copy the private link from **this terminal only**.
- When you press Connect, a **PRIVATE Identity pairing challenge** appears in this terminal.

Leave it open.

### 5. In the browser

| # | Do | Expected (PASS only if you see it) |
|---|---|---|
| 5.1 | **Connect**, then type the code from the terminal | green *Identity: connected* |
| 5.2 | Admin: **Sign in to GitHub** → **Establish Admin Identity…** → in Identity's page set a passphrase, tick, establish | *Admin: established* |
| 5.3 | **+ Identity**: `Dustin`, kind *person* | Dustin appears on the left |
| 5.4 | **Add a feature…** | card **Developer Utility (developer entrance)**, badge **pinned artifact**, button **Install…** |
| 5.5 | **Install…** → **Install** → open the Identity approval page → **Deny** | nothing appears under Installed features (**unapproved install denied**) |
| 5.6 | **Install…** again → **Install** → approval page → passphrase → **Approve** | *Developer Utility* under Installed features. Library now says **Open installed feature** |
| 5.7 | Open it. On **Overview**, press **Start** | **Running**, chips **RUNNING · CONNECTED · READY**, a line *Identity link: Identity idn_… signed an answer about this installation…*, and **Access:** *not allowed for anyone yet*. **Next:** *Ask the Admin for a permission →* |
| 5.8 | Second WSL terminal: `pgrep -af 'python[0-9.]* -I .*/p1-developer-entrance/processor-profile/installations/ins_[^ ]*/package/[u]tility_agent\.py$'` | exactly **one** line |
| 5.9 | Same terminal: `ss -ltnp \| grep "pid=$(pgrep -f 'python[0-9.]* -I .*/p1-developer-entrance/processor-profile/installations/ins_[^ ]*/package/[u]tility_agent\.py$'),"` | exactly **one** `LISTEN` line on `127.0.0.1:<port>`. Loopback only. *(If `ss` is missing: `sudo apt install iproute2`.)* |
| 5.10 | **Connections**: Dustin, `developer.connection.check`, **empty** input, **Send request** | **DENIED BY IDENTITY**, with *No permission yet. Ask the Admin to allow it…* (**unauthorized connection denied**) |
| 5.11 | *Request a connection*: Dustin, `developer.connection.check`, **Ask the Admin…** → approve in Identity | grants table: **ACTIVE** |
| 5.12 | **Overview**: **Next: Run connection check for Dustin** | **COMPLETED**: *Developer entrance is CONNECTED*, with *Processor link: AUTHENTICATED*, *Identity link: CONNECTED · 1 active grant(s)*, and *Cannot: run commands; …* |
| 5.13 | **Run self-test** | **Self-test PASS** with 3 ✓ lines |
| 5.14 | **Copy activity** | the activity appears under the card and is copied. It shows the start, the denied and the completed request, and contains no keys or codes |
| 5.15 | **Connections**: **Revoke** Dustin's grant, then **Send request** again | **DENIED BY IDENTITY**: *The permission was revoked…*. Overview's one-click button is gone |
| 5.16 | **Overview**: **Stop** | **Stopped** |
| 5.17 | Rerun 5.8 and 5.9 | 5.8 prints nothing; 5.9 prints nothing (**process and listening port gone**) |

### 6. Finish

Press **Ctrl+C** in the trial terminal.

**Expected:** the prompt returns, and 5.8 still prints nothing.

### Report

Paste back:
- the SHAs from steps 1–2;
- the step 3 output;
- **PASS / FAIL / NOT RUN** for 5.1–5.17;
- any `ERROR:` or red text;
- optionally, the **Copy activity** text.

## Rollback (keeps every other installation)

This trial only created two folders. With the trial stopped (Ctrl+C):

```bash
rm -rf ~/p1c/p1-developer-entrance ~/p1c/tp-developer-clone
```

Nothing else was created or changed: no Identity or Processor profile of your other trials, no Git branch of yours, no system setting.

## Known limits

- **No OS isolation** (same as all P1 services): a pinned service runs with your user's rights.
- **Passphrase-only Admin approval** (same as before).
- **Loopback HTTP** is signed but not encrypted.
- **One read-only operation for now.** Anything more (reading other services' status, files, actions) is a new capability that needs its own review, its own Identity grant, and your approval.
- **The connection is Processor → Developer.** A Developer *client* calling into Processor on its own initiative would need Identity to authorize requests addressed to Processor itself. That's a new Identity ticket audience, so it would require your explicit approval. It is not built.
- **The "test fixture" label** is the name of Identity's only P1 trust class; see the trust decision above.
