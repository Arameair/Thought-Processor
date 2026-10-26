# Thought Processor P0 — reusable feature onboarding (experiment report, 2026-10-26)

**Status: experiment, not promoted.** `alpha`, `stable` and `main` are unchanged. No tags or releases were created. The owner decides the next step.

## Branches and commits

| Repo | Branch | Code commit tested | Based on |
| --- | --- | --- | --- |
| Arameair/Thought-Processor | `experiment/p0-feature-onboarding-20261026` | **`9f27a1b`** (this report and the evidence are added in later docs-only commits) | `alpha` @ `9fac9fa` (docs only; the app is new code) |
| Arameair/identity | `experiment/feature-contract-20261026` | **`c8c0a54`** (API `identity/2-alpha`) | `alpha-candidate-20261026` @ `dfbb24e` |
| Arameair/Thought-engine | none changed | — | Release workflow exists on `experiment/release-workflow-20261026` @ `2baacc7`; **no release published** |

The Windows runbook and acceptance steps are in [`AGENTS.md`](../../AGENTS.md).

## What a person can do now

1. Start **Thought Processor** on its own. It says *Start Identity* until Identity is running, then finds it automatically. Either program can start first.
2. **Connect** once. Identity shows a code in its own window; the person types it into Processor. The connection is saved: with Windows DPAPI on Windows, as a 0600 file elsewhere. Processor reconnects by itself after a restart of either program, and even if Identity comes back on a different port.
3. See and add **real identities** held by Identity (person, companion, business, project). Features are shown for one selected identity at a time.
4. **Discover features** the account is approved for. Thought Engine is private: it stays invisible until GitHub sign-in shows the account may see it. Before that, Processor only says that signing in may reveal more.
5. **Sign in to GitHub** from Processor. Identity runs the sign-in; the person approves on github.com.
6. **Install** through a reusable flow:
   - Processor shows exactly what will be installed: version, source repository, file, size, SHA-256, what was checked, and what was *not* checked.
   - Nothing downloads until the person presses **Install**.
   - The file is downloaded through Identity, checked, and recorded.
7. **See the real install state** after restarts: *Installed vX · Files verified*. The file is re-hashed on every start, so changed or missing files are flagged.
8. **Exit** with the owner's choice:
   - **Shut down managed workspace** (default) or **Keep services running**.
   - An Identity the person started separately is stopped only if they tick *Also stop Identity*, and the stop is verified.

## What it deliberately does not do

- Start or stop Thought Engine or any feature.
- Grant feature or source access. Identity answers DENIED to everything; cards say *Not turned on for <identity>* and *No sources allowed*.
- Update channels (alpha/stable), automatic update notices, or rollback.
- Admin sign-in per person.
- Publisher-signed releases.
- Memory.
- Changes to Engine cognition.

## Results

### Security finding during verification — FIXED

An independent check from fresh clones found a **high-severity flaw** in the Identity API as first written (`identity/1`). Processor sent the pairing secret on every request, and Identity signed its answers with that same secret.

**What an attacker could do:** run a program on Identity's port while Identity was stopped. That program could:
- collect the secret;
- show Processor fake data, including a fake release and SHA-256 in the approval dialog;
- reuse the secret against the real Identity.

The checker demonstrated this with a 40-line impostor.

**Fix (`identity/2`, Identity `c8c0a54` + Processor `9f27a1b`):**
- After pairing, the secret is never sent again.
- Each request carries a one-time signature over the method, path, time, a fresh nonce and the body. Identity refuses a time more than 60 s off and any reused nonce.
- Answers are signed with a separate derived key, covering the request's nonce, the status and the body.
- An impostor now gets nothing reusable and can't produce an answer Processor accepts.

**Proof:**
- Unit tests cover an impostor signing with everything it can observe, status swaps, replay, tampering, stale time and a wrong secret.
- Integration scenario 11b: an impostor on Identity's real port leaves Processor at *WRONG_SERVICE*, actions are refused, and the secret never reaches the impostor.

Pairings made with `identity/1` must connect again. **The Oct 26 Identity alpha candidate (`alpha-candidate-20261026`) still speaks `identity/1`. It carries a warning and should not be built on.**

The same check led to three smaller fixes:
- Install tag and file names can no longer contain `.` or `..`, and the final path must stay in the feature folder.
- Feature ids are validated on every route.
- Identity's status no longer names a hidden private feature.

### Automated tests (Linux developer container, Go 1.24, race detector)

| Suite | Result | Evidence |
| --- | --- | --- |
| Processor: `idclient` (proof verification incl. tampered body, wrong installation, missing proof, 401, down; sealed pairing; locator discovery, loopback-only, relocation) | **PASS** | [go-test-processor.txt](p0-evidence-20261026/go-test-processor.txt) |
| Processor: `install` (approval required; verified install; restart re-verify; changed/missing file; tampered and cut downloads install nothing and leave no partial file; untrusted uploader and wrong tag refused before download; release changed after approval; expired approval; leftover cleanup) | **PASS** | same |
| Processor: `app` (session, CSRF, origin, host checks; protected actions fail closed without Identity; Developer Mode grants nothing) | **PASS** | same |
| Identity `experiment/feature-contract-20261026`, all packages | **PASS** | [go-test-identity.txt](p0-evidence-20261026/go-test-identity.txt) |
| `go vet` with `GOOS=windows`; Windows cross-build of both programs | **BUILDS** | not run on Windows |

### Integration: real processes, headless browser, GitHub as a labelled TEST FIXTURE

The test runs the real Thought Processor and real Identity programs. A headless browser clicks through Processor's own page, and faults are injected through the fixture. **28/28 PASS** at the final commits: [report](p0-evidence-20261026/p0-integration-report.txt).

| # | Scenario | Result |
| --- | --- | --- |
| 1 | Processor started first shows *Start Identity*; Identity started later is found through its locator | PASS |
| 2 | Wrong code refused; correct code pairs; status signed; token stored sealed | PASS |
| 3 | Identity created in Identity, selected, no access; private Engine hidden before sign-in | PASS |
| 4 | GitHub sign-in through Identity makes the Engine appear, eligible to install, still not granted to run or read sources | PASS |
| 5 | Release uploaded by a person (not the release workflow) refused; nothing downloaded | PASS |
| 6 | No published release reported honestly | PASS |
| 7 | Tampered download → CHECKSUM_MISMATCH, nothing installed, no leftover file | PASS |
| 8 | Download cut halfway → DOWNLOAD_INTERRUPTED, nothing installed, no leftover file; retry works | PASS |
| 9 | Person approves in the dialog → installed; file on disk matches the approved SHA-256; the record keeps the source | PASS |
| 10 | Exit → Keep running; Processor restarts and reconnects with no code; install re-verified | PASS |
| 11 | Identity stopped → AUTHORITY UNAVAILABLE banner; install and create refused (503); installed feature still shown | PASS |
| 11b | An **impostor** program on Identity's port is not trusted (WRONG_SERVICE), actions are refused, and it never receives the pairing secret | PASS |
| 12 | A different Identity installation on the same port is not trusted | PASS |
| 13 | The paired Identity on a new port is found again by installation ID | PASS |
| 14 | After the Identity restart (GitHub signed out) the private feature is hidden while the local install is still shown; another account never sees it | PASS |
| 15 | Installed file changed on disk → FILES_CHANGED | PASS |
| 16 | Developer Mode shows details, grants nothing | PASS |
| 17 | Exit → Shut down + *Also stop Identity* → both stopped, verified; Identity's locator removed | PASS |

Screenshots of these steps are in [p0-evidence-20261026/](p0-evidence-20261026/): `p01`–`p12`, covering start-identity, found, type code, identity without features, GitHub code, Engine available, checksum mismatch, approve dialog, installed, Identity unavailable, Developer Mode and the exit dialog.

**Bugs found and fixed during testing:**
- Relocation trusted a healthy answer on the old port over the locator for the paired installation ID.
- The test harness itself let a crash look like a pass. It now records a FAIL.

### What has NOT been shown

| Item | Status | Why |
| --- | --- | --- |
| Real Thought Engine install | **BLOCKED** | `Arameair/Thought-engine` has **no releases**. The publisher GitHub App also has only *Metadata* permission, and reading a private repository's releases needs **Contents: Read-only**. Both are owner actions (see AGENTS.md). Fixture installs prove the mechanics only. |
| Run on Windows | **NOT RUN** | Cross-built and vetted only. DPAPI sealing, the Windows file lock, `%LOCALAPPDATA%` locators and browser launch have not been observed on Windows. |
| Real GitHub sign-in through Processor | **NOT RUN** | Needs the owner on github.com. The Identity side uses the same device flow that the owner tested on Oct 25. |
| Publisher authenticity | **LIMITED** | Enforced: the approved repository, tag prefix, file name, size limit, not a draft or pre-release, and that both the file and `SHA256SUMS` were uploaded by `github-actions[bot]` (the repository's own release workflow). The SHA-256 is checked before and after the file is moved into place. **Not** enforced: a publisher signature. Someone with write access to the repository could still publish through its workflow. The approval dialog says this. |

## Security notes

- **The pairing secret never leaves Processor after pairing** (`identity/2`).
  - Every request is signed: method, path, time, a fresh nonce and the body hash.
  - Every answer is verified with a separate response key, over the nonce, status and exact body, plus the installation ID.
  - Unsigned or mismatched answers become *WRONG_SERVICE*; 401 becomes *CONNECTION_REMOVED*.
  - Protected actions ask Identity again at that moment. There are no cached allows.
- The page needs this run's private link (session cookie plus CSRF token, Origin and Host checks). The link is stripped from the address bar after use.
- The pairing token is never logged or shown. On Windows it is sealed with DPAPI for the current user.
- Installs go only into the Processor profile and need no admin rights. Nothing is executed, and leftover partial downloads are removed on start.
- Developer Mode only changes what is displayed.

## Owner acceptance checklist

- [ ] Both programs build on Windows (`scripts\build-windows.ps1` in each repo), and every test prints `ok`.
- [ ] Processor started first shows *Start Identity*; once Identity starts, it finds it.
- [ ] Connecting needs the code from the Identity window; a wrong code is refused.
- [ ] Adding *Tass* works; the feature list is shown for Tass only.
- [ ] Before GitHub sign-in, Thought Engine is not listed. After sign-in it appears as *Approved for your account* (or stays hidden for an account without access).
- [ ] **Install…** shows the approval dialog with source, size, SHA-256 and *Not checked: publisher signature*. Nothing downloads before **Install**. *(BLOCKED until a release exists and the GitHub App has Contents: Read-only.)*
- [ ] After install: *Installed … · Files verified*, and the file exists under `.thought-processor\p0\features\thought-engine\`.
- [ ] Exit → Keep running, then start again: reconnects without a code; Tass and the install are still there.
- [ ] Stopping Identity shows the amber banner and blocks actions; starting it again reconnects by itself.
- [ ] Developer Mode shows details and doesn't change what is allowed.
- [ ] Exit → Shut down + *Also stop Identity* stops both.

## Recommendation: **KEEP**, as the base for the next iteration

**Why KEEP:**
- This is the first build where a person can do the whole owner-requested flow in Processor: connect, choose an identity, discover features, approve an install.
- The installer is driven only by the feature contract, so a second feature needs a contract entry in Identity, not Processor code.
- Every protected step fails closed.
- Everything that can be tested without a real release and without Windows passes, including the failure cases.
- The independent security check found one serious design flaw. It is fixed at the API level and covered by tests, including an impostor run against the real programs.

**Before calling it an Alpha candidate:**
1. Run AGENTS.md on Windows.
2. Unblock a real install: add *Contents: Read-only* to the GitHub App and publish `engine-v0.1.0` from the release workflow.
3. Decide whether a publisher signature (for example minisign or Ed25519, with the key built into Identity) is required before testers outside the owner's machine install anything.
