# TEST_P1 — hands-on acceptance walkthrough (≈15 minutes)

Do [START_HERE.md](START_HERE.md) sections 1, 2 and 4 first (two terminals running, Processor open in a browser). Record each line as **PASS / FAIL / BLOCKED / NOT RUN** with what you actually saw. A step passes only if you saw the **Expected** result.

Rules: a **human** approves GitHub sign-in on github.com and types the Admin passphrase in **Identity's** page. Don't paste codes, links after `#`, passphrases or credential files anywhere.

## Gate 1 — Admin Identity bootstrap

| # | Do | Expected |
|---|---|---|
| 1.1 | Processor page shows *Connect to Identity* → **Connect** → type the code from terminal 1. | Chip **Identity: connected**. The code only appears in terminal 1. |
| 1.2 | Sidebar → **Administrator → Establish Admin…** | *Not established yet*; step 1 *Sign in to GitHub*. Chip **Admin: not established**. |
| 1.3 | **+ Feature** → *Install…* → *Ask the Admin…* (before any Admin exists). | Refused: *no Admin Identity is established…* Nothing installed. |
| 1.4 | Back to Administrator → **Sign in** → approve on github.com. | GitHub chip shows your login. Step 2 offers *Establish Admin Identity…* |
| 1.5 | *Establish Admin Identity…* → *Open Identity approval page*. | A **separate Identity page** (not Processor) explains the Admin Identity, shows your GitHub account + numeric id, asks for a passphrase twice and a confirmation tick. Terminal 1 shows the same request. |
| 1.6 | Try *Establish* without the tick, then with a short passphrase. | Both refused with a clear message. |
| 1.7 | Passphrase (12+ chars, twice), tick, **Establish Admin Identity**. | *Approved and carried out — adm_…*. Processor: chip **Admin: established**; Administrator page shows Admin ID, bound GitHub numeric id, authenticator `local-passphrase-pbkdf2-sha256`, *Recovery: DEFERRED*. |
| 1.8 | *(optional, needs a 2nd GitHub account or the fixture)* Sign GitHub out/in as another account (e.g. in the fixture: `curl -X POST 'http://127.0.0.1:8790/fixture/login?user=intruder'`, then reconnect GitHub). | Administrator page: same Admin ID; *Signed-in GitHub now … NOT the Admin's account — no Admin authority*. *Establish Admin…* is no longer offered. |

## Gate 2 — install one trusted test feature

| # | Do | Expected |
|---|---|---|
| 2.1 | **+ Feature** (Library). | *P1 Deterministic Worker (TEST FIXTURE)* with badges `TRUSTED_LOCAL_TEST_FIXTURE`, `thought.feature-api/1`; *Package details* shows files with SHA-256 and *Publisher … claim only — not verified*. |
| 2.2 | **Install…** | Review lists what was checked **and what was not** (publisher identity, OS sandboxing, release provenance). |
| 2.3 | **Ask the Admin to approve** → *Open Identity approval page* → type a **wrong** passphrase → Approve. | Refused (*not correct*). Nothing installed. *(The link is now spent: Close, and Install… again for a fresh one.)* |
| 2.4 | Install again → open the approval page → correct passphrase → **Approve**. | Identity page: *Approved*. Within ~2 s the sidebar shows the feature under **Installed features**. |
| 2.5 | Open it → **Overview**. | Exactly **one** `ins_…` *durable · minted by Identity*; **Identity: INSTALLED**, **Process: STOPPED**, **0 active grants**, **0 information sources**; files *verified against the approved SHA-256*; *Administered by adm_…*. |

## Gate 3 — standalone run, health, self-test, startup modes

| # | Do | Expected |
|---|---|---|
| 3.1 | **Connections** → choose Tass → **Send synthetic request**. | **DENIED BY IDENTITY** — *NO_GRANT — nothing was started or sent*. Process still STOPPED. |
| 3.2 | **Runtime** → **Start**. | **RUNNING**, a `lch_…` launch, a `127.0.0.1:<port>` local API, Heartbeat **READY** with a sequence number that grows on refresh. |
| 3.3 | **Run standalone self-test**. | *Self-test PASS*, *SYNTHETIC TEST DATA*, *grants used: none*. |
| 3.4 | In a terminal: `curl -s http://127.0.0.1:<port>/v1/status` | `LAUNCHER_AUTH_REQUIRED` (only Processor's launch key opens it). `…/v1/health` says only `RUNNING`. |
| 3.5 | **Stop**. | STOPPED; *Last exit … stopped by Processor*. |
| 3.6 | Startup → **On demand**, idle **10** s, Save. (Do it again after Gate 4.2 to see it start by itself.) | Saved; text says this changes *when* it runs, never *what* it may access. |

## Gate 4 — Admin-approved narrow grant, enforced by the feature

| # | Do | Expected |
|---|---|---|
| 4.1 | **Connections → Request a connection**: For identity *Tass*, capability `worker.test.synthetic`, Expires *never* → **Ask the Admin…** → open the Identity page. | The Identity page names the requester (this Processor), the installation, the one capability, Tass and *Information sources: none*. |
| 4.2 | *(Offline check, optional: disconnect the internet / disable Wi-Fi now.)* Approve with the passphrase. | Approved — works **without GitHub/internet**. Grants table: *this Processor · Tass · worker.test.synthetic · until revoked · ACTIVE*. |
| 4.3 | Make sure the feature is **stopped**, startup **On demand** → **Send synthetic request**. | **COMPLETED** + *started on demand*, a `dec_…` decision ID and a SYNTHETIC TEST DATA block (`"upper": "HELLO FROM P1"`…). After the idle window the process goes back to STOPPED by itself. |
| 4.4 | Send for a different identity (add one, e.g. *Other*), or any capability other than the granted one. | **DENIED BY IDENTITY** (*NO_GRANT*). |
| 4.5 | **Revoke** the grant → send again. | **DENIED BY IDENTITY** — *GRANT_REVOKED*. No restart of anything was needed. |
| 4.6 | Approve a new grant, then **Ctrl+C Identity** (terminal 1) → send. | Yellow banner **AUTHORITY UNAVAILABLE**; result **AUTHORITY UNAVAILABLE** (not DENIED, not COMPLETED). A running feature still answers its standalone self-test. |
| 4.7 | **Logs** tab (with Identity running again). | *Identity decisions for this installation* (`ticket.issued`, `authz.checked` ALLOW/DENIED with reasons), what the feature hears from Identity, and the feature's own log. |

Wrong-identity, wrong-capability, forged / tampered / replayed / expired tickets, a different requester and Identity-down at the **feature** are exercised automatically by `acceptance/p1_integration.py` (Gate 4 lines), because Processor's UI only ever sends legitimate requests.

## Gate 5 — restarts, persistence, recovery

| # | Do | Expected |
|---|---|---|
| 5.1 | Start Identity again (same command). | Processor reconnects by itself; same `idn_…`, same `ins_…`; grants exactly as before (revoked stays revoked). |
| 5.2 | **Exit → Keep Identity running**, then start Processor again (same command). | Feature processes it started were stopped on exit. After restart: same installation, no duplicate, no code needed. |
| 5.3 | *(Recovery, optional)* Start the feature, then create the TEST-ONLY file `~/p1/profiles/processor/installations/<ins_…>/data/TEST-ONLY-fault-injection.json` containing `{"crash": true}` and send the input `__crash__`. Remove the file afterwards. | **REQUEST UNCERTAIN** (*NOT resent*); Runtime **NEEDS_ATTENTION** — *recovery is MANUAL*. No automatic restart. With *Crash recovery: Bounded*, the process is restarted (new launch each time) at most 3 times in 10 minutes, then NEEDS_ATTENTION. |
| 5.4 | **+ Feature → Install…** a second time (approve). | A **second, different** `ins_…` for the same package, its own data folder, 0 grants. |
| 5.5 | Second copy → Overview → **Uninstall…** (approve in Identity). | Marked *uninstalled*; tombstone keeps its ID; Identity lists it as UNINSTALLED; its grants are revoked. |

## What to send back

The PASS/FAIL table above, `git log -1` of both repos, and (if you ran them) `~/p1/run-integration/p1-integration-report.txt` and the screenshots. Redact the Identity terminal's `troubleshooting page` line and any `#…` link in screenshots.

**Not part of this test (BLOCKED / NOT BUILT):** real Fast AI Worker, ChatJimmy, Thought Engine, invitations, third-party release execution, OS sandboxing, Windows Hello/passkeys, Admin recovery, native Windows run.
