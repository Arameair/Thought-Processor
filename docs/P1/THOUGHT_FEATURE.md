# Thought plugged into Thought Processor (P1, 2026-10-29)

| Repo | Branch | Commit |
| --- | --- | --- |
| Arameair/Thought-Processor | `experiment/p1-thought-feature-claude-20261029` (on top of the secure worker runtime, `8af218d`) | see the delivery report |
| Arameair/Thought | `experiment/thought-feature-p1-claude-20261029` | `772200b` |
| Arameair/identity | `experiment/p1-cleanup-claude-20261026` | `64940b6` (unchanged) |

**What it is:** Thought (Spark → Trace → Thought) runs as a Thought Processor feature, like the P1 Deterministic Worker. It is installed through Identity's Admin approval and run by Processor. Every use is checked live with Identity, for one identity and one capability at a time. The rules, capabilities and inputs are described in Thought's [`docs/THOUGHT_FEATURE.md`](https://github.com/Arameair/Thought/blob/experiment/thought-feature-p1-claude-20261029/docs/THOUGHT_FEATURE.md).

## What changed in Thought Processor

- **`run-p1.sh`** finds a `Thought` clone next to `identity` and `Thought-Processor`. It requires that clone to be on its branch with no local changes. It then trusts `Thought/feature` and pins Thought's **exact artifact digest** in **both** programs, using the flags that already existed: Processor `-trust-feature-artifact` and Identity `-trust-fixture-artifact`.
  - Pinned digest: `8f92ee685c99851128f322c9063ff151e28d7ccf48709315d8857fc56e8079ec`.
  - Any other content stays **BLOCKED** in both programs.
  - Without a Thought clone, or with `P1_THOUGHT=off`, the launcher works as before.
  - No Identity code was changed.
- **Feature page → Connections:** the request form shows what input the chosen capability expects, and its button is now just **Send request**.
- **Clearer errors:** when a feature refuses a request it was allowed to make (for example unusable input), the result shows the feature's reason.

## Automated test

`python3 -I processor/acceptance/p1_thought_feature.py --identity-src ../identity --thought-src ../Thought --work <dir>` runs real Identity, Processor, Thought and the worker together. **36 checks.** See the delivery report for exact results.

## Owner test on Windows + WSL (about 15 minutes)

This continues the secure-worker-runtime trial: same folder `~/p1c/p1-worker-runtime`, same ports 7794/7864. Your Admin, identities and the installed worker carry over. Your original trial in `~/p1c/p1-launcher` is not touched.

**Never share:** the pairing code, the GitHub device code, anything after `#code=`, `#key=` or `/approve#`, the Admin passphrase, or any `credential.json`.

### 1. Get the code (WSL terminal, in the folder that holds `identity` and `Thought-Processor`)

```bash
git clone --branch experiment/thought-feature-p1-claude-20261029 https://github.com/Arameair/Thought.git
cd Thought-Processor
git fetch origin experiment/p1-thought-feature-claude-20261029:experiment/p1-thought-feature-claude-20261029
git worktree add ../tp-thought experiment/p1-thought-feature-claude-20261029
cd ..
```

Already have a `Thought` clone? Then instead run:

```bash
git -C Thought fetch origin experiment/thought-feature-p1-claude-20261029
git -C Thought switch experiment/thought-feature-p1-claude-20261029
```

Thought is a private repo, so Git may ask you to sign in.

**Expected:**
- `Thought` is next to `identity`.
- A new `tp-thought` folder is there.
- Your other checkouts are unchanged.

### 2. Start (stop the earlier runtime trial first, if it's still running: Ctrl+C in its terminal)

```bash
cd tp-thought
bash ./run-p1.sh
```

**Expected:**
- `Thought feature source: 772200b… (pinned artifact 8f92ee685c99…)`.
- Then `P1 Processor ready on 127.0.0.1:7864`.
- The page opens, or you copy the private link from **this terminal only**.

### 3. Install Thought

1. Go to **Add a feature…**. Two cards: *P1 Deterministic Worker* (**Open installed feature**, if you did the runtime test) and **Thought (Spark → Trace → Thought)** with a **pinned artifact** badge and an **Install…** button.
2. Click **Install…** on Thought, then **Install**, then approve in Identity's page with your passphrase.

**Expected:** *Thought* appears under **Installed features**, with its own `ins_…` ID.

### 4. Start it, self-test

On Thought's **Overview** tab, press **Start**, then **Run self-test**.

**Expected:**
- **Running**, then a green **Self-test PASS**.
- The output lists **8 rule checks**, all `"ok": true`.

### 5. Spark → Trace → Thought for Tass

Go to the **Connections** tab.

1. Choose **For identity: Tass**, **Capability `thought.event.submit`**, Input `goldshire: the player went back to Goldshire`, and press **Send request**.
   **Expected:** **DENIED BY IDENTITY** (`NO_GRANT`). Nothing has permission yet.
2. Under *Request a connection*: Tass, `thought.event.submit`, **Ask the Admin…**, then approve in Identity.
3. Send the same input again, three times in a row.
   **Expected:** **COMPLETED** each time. The output's `outcome` goes `trace_kept`, `trace_kept`, then **`thought_formed`**, with the headline *A Thought formed: Why does "goldshire" keep coming up?* and `evidence_count: 3`.
4. Send `goldshire: just a passing mention !weak`.
   **Expected:** `faded`. The headline says nothing was kept.

### 6. Reading needs its own permission

1. Capability **`thought.thoughts.read`**, Input empty, **Send request**.
   **Expected:** **DENIED BY IDENTITY**. Permission to send events doesn't let anyone read.
2. Ask the Admin for `thought.thoughts.read` for Tass, approve it, and send again.
   **Expected:** **COMPLETED**, listing **one** Thought with `why_it_formed` and its 3 events marked `ORIGIN`.
3. Add a second identity if you don't have one (**+ Identity**, e.g. `Other`). Try `thought.thoughts.read` for **Other**.
   **Expected:** **DENIED**. If you also grant read for Other, the answer is an **empty** list: Other never sees Tass's Thoughts.

### 7. Revoke, restart

1. In the grants table, **Revoke** Tass's `thought.event.submit`. Send an event again.
   **Expected:** **DENIED** (`GRANT_REVOKED`). Reading still works, because that's a separate grant.
2. Press **Ctrl+C** in the terminal, run `bash ./run-p1.sh` again, open Thought, **Start**, and read for Tass.
   **Expected:** the **same** Thought (same `th_…` ID) is still there.

### Report

For each step, report **PASS / FAIL / NOT RUN** with what you saw, plus the two commit lines from step 2.

### Clean up (optional)

1. Press **Ctrl+C** in the terminal.
2. Remove the worktree:
   ```bash
   git -C Thought-Processor worktree remove ../tp-thought
   ```
3. Run `rm -rf ~/p1c/p1-worker-runtime` to reset the trial.

## Known limits

- **No dedicated Thought screen yet.** Results show as the raw observation in the request box (headline, steps, Thought, evidence).
- **Nothing calls Thought automatically yet.** TASS, Thought Controller and Thought Engine aren't connected. Any program that pairs with Identity and gets a grant can use the same three capabilities.
- **Same P1 limits as the worker:** no OS isolation, a passphrase authenticator, and loopback HTTP that is authenticated but not encrypted.
- **A Thought belongs to one identity on one Thought installation.** A second Thought installation in another Processor profile has its own, separate Thoughts.
