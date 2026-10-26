# AGENTS.md — Thought Processor P0 Windows runbook (owner + AI operators such as a Grok bot)

**Goal:** on the owner's **Windows** computer, without Project Controller:
1. Run **Thought Processor** (P0) and **Identity** (feature-contract experiment).
2. Connect them.
3. Add an identity and sign in to GitHub.
4. Try to install **Thought Engine** through the reusable installer.
5. Check that restarts and exit behave correctly.
6. Report each step as **PASS / FAIL / BLOCKED / NOT RUN**.

| Program | Repo · branch | Port | Profile folder |
| --- | --- | --- | --- |
| Thought Processor | `Arameair/Thought-Processor` · `experiment/p0-feature-onboarding-20261026` | 7860 | `%USERPROFILE%\.thought-processor\p0` |
| Identity | `Arameair/identity` · `experiment/feature-contract-20261026` | 7790 | `%USERPROFILE%\.identity\p0` (fresh for this trial) |

Both repositories are public, so you don't need a token to clone or build.

## Rules

1. **Never paste, log or send** any of these into chat, issues or reports. Note that Identity's console prints its private troubleshooting link (`…/#key=…`) at start, so crop or blur that line in any screenshot of the Identity window.
   - the 8-character pairing code shown in the Identity window;
   - the GitHub device code;
   - anything after `#code=` or `#key=`;
   - the contents of `identity-pairing.json`.
2. **A human approves** GitHub sign-in (on github.com) and **approves the install** (the *Install* button in the approval dialog). The AI may click everything else.
3. Identity and Processor each run in **their own** window. Don't kill processes you didn't start.
4. Don't change `alpha`, `stable` or `main`, and don't push tags or publish releases.
5. A step passes only if you saw the **Expected** result.

## Step 0 — prerequisites (any window)

```powershell
git --version
go version        # go1.22 or newer (https://go.dev/dl/); open a NEW PowerShell after installing
```
Close any older Identity or Thought Processor windows (they use the same ports).

## Step 1 — get and build both programs (window 3)

```powershell
cd $HOME
git clone --branch experiment/feature-contract-20261026 https://github.com/Arameair/identity.git identity-p0
git clone --branch experiment/p0-feature-onboarding-20261026 https://github.com/Arameair/Thought-Processor.git tp-p0
cd $HOME\identity-p0 ; .\scripts\build-windows.ps1
cd $HOME\tp-p0\processor ; .\scripts\build-windows.ps1
```
If PowerShell blocks the scripts, run these by hand instead. The `-ldflags` part makes the console show the branch name.
```powershell
cd $HOME\identity-p0
go vet ./... ; go test ./...
go build -ldflags "-X main.buildBranch=experiment/feature-contract-20261026" -o identity.exe .
cd $HOME\tp-p0\processor
go vet ./... ; go test ./...
go build -ldflags "-X main.buildBranch=experiment/p0-feature-onboarding-20261026" -o thought-processor.exe .
```

**Expected:**
- every `go test` line says `ok`;
- you end up with `identity.exe` and `thought-processor.exe`.

Record both `git log -1 --format=%H` values.

## Step 2 — start Processor FIRST (window 1)

```powershell
cd $HOME\tp-p0\processor
.\thought-processor.exe
```
**Expected:**
- The console shows `Thought Processor 0.2.0-p0  branch experiment/p0-feature-onboarding-20261026  commit <first 12 characters of the Step 1 SHA>`.
- A browser opens Thought Processor.
- The page says **Start Identity**, and the top chip says *Identity: not running*.

## Step 3 — start Identity (window 2)

```powershell
cd $HOME\identity-p0
.\identity.exe -data "$env:USERPROFILE\.identity\p0"
```
**Expected:**
- Identity's console says *running in the background*.
- Within a few seconds, Processor's page changes to **Connect to Identity** and shows one Identity with a **Connect** button.

## Step 4 — connect (pairing)

1. In Processor press **Connect**.
2. Window 2 shows a **PAIRING REQUEST** box with a code.
3. Type that code into Processor's box and press **Connect**.

**Expected:**
- The chip turns green: *Identity: connected*.
- The page shows **Identities** on the left.
- A deliberately wrong code is refused with *wrong or expired* (optional check).

## Step 5 — add an identity

Type `Tass`, choose **Companion**, then press **Add identity**.

**Expected:**
- *Features for Tass* appears.
- Before GitHub sign-in, **no Thought Engine card** is shown (private features stay hidden).
- Instead the page shows *Some features are only shown to approved accounts. Sign in to GitHub…*.

## Step 6 — sign in to GitHub (human)

Press **Sign in to GitHub**. A dialog shows a link and a code. **Owner:** open the link, enter the code and press **Authorize**.

**Expected:**
- The chip shows *GitHub: <your login>*.
- If your account can see `Arameair/Thought-engine`, a **Thought Engine** card appears with these badges:
  - *Approved for your account*
  - *Not installed*
  - *Not turned on for Tass*
  - *No sources allowed*
- If no card appears, record **FAIL** with the page text. Possible causes: Identity's GitHub App isn't installed on Thought-engine, or your account can't see it.

## Step 7 — install Thought Engine (human approves)

Press **Install…**.

- **Expected today (no Engine release exists yet): BLOCKED.**
  - The card shows **No version published yet**, or a message starting *Identity couldn't read the release from GitHub* that mentions **HTTP 403**. The 403 appears when Identity's GitHub App lacks *Contents: Read-only*.
  - For a private repository GitHub may also answer "not found" when that permission is missing, so the first message alone doesn't prove there's no release.
  - Nothing is installed. Record **BLOCKED** with the message.
- **Once a release exists** (see "Owner actions" below):
  - An approval dialog lists version, source, file, size and SHA-256. It also lists what was checked and what was *not* checked.
  - **Owner:** press **Install**.
  - Expected: a progress bar, then badges **Installed engine-v…** and **Files verified**.
  - The file is in `%USERPROFILE%\.thought-processor\p0\features\thought-engine\<tag>\`. Nothing is started.

## Step 8 — restart Processor

1. Press **Exit**, choose **Keep services running**, then press **Exit**. Window 1 prints `Thought Processor stopped.` and Identity keeps running.
2. Start `.\thought-processor.exe` again.

**Expected:**
- It reconnects **without a new code**.
- *Tass* is still there.
- If you installed in Step 7, the card still says **Installed … · Files verified**.

## Step 9 — Identity goes away and comes back

1. In window 2, press **Ctrl+C**.
   - **Expected:** an amber banner says *Can't reach Identity…*.
   - Adding an identity or installing is refused.
   - Installed features stay listed.
2. Start Identity again with the Step 3 command.
   - **Expected:** Processor reconnects on its own within about 10 seconds.
   - GitHub shows signed out (Identity forgets the sign-in on restart, by design).

## Step 10 — Developer Mode

Tick **Developer Mode**.

**Expected:**
- A *Developer details* section appears below, with build commits, connection state and recent activity.
- Nothing becomes allowed that wasn't before.
- Untick it again.

## Step 11 — Exit → Shut down

1. Press **Exit**.
2. Choose **Shut down managed workspace**.
3. Tick **Also stop Identity**. It is unticked by default, because you started Identity yourself.
4. Press **Exit**.

**Expected:**
- The page says Processor stopped and *Identity: stopped (verified…)*.
- Both console windows end with `… stopped.`

## Report

Include:
- the two commit SHAs;
- **PASS / FAIL / BLOCKED / NOT RUN** for each step, with what you saw;
- a screenshot of anything unexpected.

Turn on Developer Mode to copy the technical details, and check them for codes before sharing.

## Owner actions that unblock a REAL install (not for the AI)

1. **GitHub App permission:** in the publisher GitHub App settings, add **Repository permissions → Contents: Read-only**, and make sure the app is installed on `Arameair/Thought-engine`. GitHub asks each person to approve the new permission at their next sign-in.
2. **Publish a release.** The release workflow is on branch `experiment/release-workflow-20261026` (commit `2baacc7`). Pushing a tag there makes GitHub Actions build the release and upload the files as `github-actions[bot]`, which is the *only* uploader the contract accepts:
   ```powershell
   git clone https://github.com/Arameair/Thought-engine.git ; cd Thought-engine
   git push origin 2baacc7f237a78ba22e441cbb6050821611f2959:refs/tags/engine-v0.1.0
   ```
   This publishes software, so it's your decision; nothing in this work did it.

## Troubleshooting

| You see | Meaning / fix |
| --- | --- |
| `can't use 127.0.0.1:7860` | Another Processor is running. Close it, or start with `-port 7861`. |
| `another Thought Processor is already using …` | Same profile is in use. Close the other one. |
| Page says *This window isn't signed in* | Processor restarted; use the new browser tab it opened, or the `open :` link from window 1. |
| *More than one Identity is running* | Choose the one whose profile is `…\.identity\p0`. |
| *A different Identity is answering* / *connection removed* | A different Identity profile is running on that port. Start the right one, or choose **Connect again**. |
| Processor never finds Identity, though Identity is running | Processor finds Identity through `%LOCALAPPDATA%\Arameair\Identity\instances\` (written by Identity at start) or port 7790. Both programs must run as the same Windows user. Don't set `IDENTITY_LOCATOR_DIR` unless you set it the same way for both. |
| *This connection was made with an older Identity* | Identity API is now `identity/2` (security fix: the pairing secret is never sent). Choose **Connect again**. |
| *The saved connection can't be read* | It was made by another Windows user or computer (DPAPI). Choose **Connect again**. |
| *Download stopped* / *Download didn't match* | Nothing was installed. Press **Install…** to retry. |
