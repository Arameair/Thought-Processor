# The First Spark — executable test designs and acceptance matrix

**Status: TEST DESIGN ONLY; NONE OF THESE TESTS HAVE BEEN RUN BY THIS DOCUMENT.**  
**Implementation brief:** [Claude build instructions](CLAUDE_BUILD_BRIEF.md). **Expectations:** [Gates and risks](EXPECTATIONS_AND_GATES.md). **Report:** [Fill-in template](TEST_RUN_REPORT_TEMPLATE.md). **Narrative:** [The First Spark](../README.md).

## 1. Test discipline

- **Run tests against actual processes and APIs.** Programmatic unit fixtures may verify parsing and failure paths, but may not certify live connectivity, authentication, model behavior, repository ownership or grant enforcement. Distinguish **UNIT FIXTURE**, **INTEGRATED REAL**, and **MANUAL OBSERVED** in reports.
- Every test has four possible outcomes: **PASS**, **FAIL**, **BLOCKED** (required genuine service or owner decision unavailable), **NOT RUN**. Never turn BLOCKED/NOT RUN into PASS.
- Record exact target branch and SHA, actual executable hash if available, OS/Go/browser versions, tested local data root (redacted where necessary), PID/instance ID, requested operation, authoritative result or error, times, sanitized logs and rollback.
- Separate test roots for every experiment. Keep user installation data, actual customer repositories and tokens untouched. One explicit test identity/entity X and test source S only **after** they are genuinely created by their authoritative services; no made-up IDs shown as verified.
- Begin with an **observed baseline**: Engine already running vs not running; Identity available vs unavailable; worker actually connected vs not. Do not infer from last run's screen.
- Do not automatically send AI requests in connectivity or health tests. Any real model call must be deliberate, with visible free-session/usage constraints and owner approval. No paid fallback.
- A test where the correct result is "Not implemented" or "Unknown" can PASS **the truthful UI behavior**, but it does **not** PASS the missing feature.

## 2. Phase A — Processor itself is real (P0)

| ID | Preconditions + action | Required outcome | Proof to capture |
| --- | --- | --- | --- |
| A01 | Build and start Processor with isolated root; open localhost UI | Real backend responds; product says **Thought Processor** and actual PID/port/startup timestamp; no identity/engine status fabricated | Exact build, health/status response, PID, page capture |
| A02 | Start without Identity/Engine/GitHub or worker | Page works for diagnostics; states **NOT CHECKED / NOT CONNECTED**; clearly **not full customer-ready** | Real API/UI result |
| A03 | Close browser tab then reopen while backend stays running | Same Processor process/run ID; tab close did not kill backend | OS PID before/after + status |
| A04 | Bad or unwritable config/data root | Startup fails safely and says why; no partial READY or writes into source checkout | Exit code, stderr, file audit |
| A05 | Inspect returned JSON/UI for identities, worker, GitHub | Every claim has an actual authoritative source + checked time or displays Unknown/Not Verified | Field/source ledger |
| A06 | Try to expose on non-loopback network address without explicit supported auth | Unsupported/insecure public bind refused in this experiment | Actual bind/connection test |
| A07 | Use frontend buttons in normal mode with no corresponding supported operation | Disabled with clear reason; no inert/cosmetic control suggests work completed | Click/response evidence |

## 3. Phase B — Real Engine lifecycle handshake (P0; FIRST MILESTONE)

| ID | Preconditions + action | Required outcome | Proof to capture |
| --- | --- | --- | --- |
| B01 | Engine not running, approved executable and digest present; Ensure once | Exactly **one session-owned Engine** starts, after which readiness is checked separately; only verified fields marked verified | Launcher request ID, process tree, PID, digest, Engine status, readiness times |
| B02 | Correct independently running Engine; Ensure once | **ATTACHED, ownership=external**; no new Engine created | Before/after process list, instance ID |
| B03 | Double click Ensure or send two simultaneous identical requests | **One instance** per session/service/build; second request attaches/returns same operation/result | Parallel request IDs, process counts, state |
| B04 | Selected Engine executable not installed | **NOT INSTALLED** and next action, never silent download or fake green | Filesystem check + UI state |
| B05 | Executable present but different digest/commit | **INCOMPATIBLE / WRONG BUILD**; do not start/overwrite/replace | Expected vs actual digest; no new PID |
| B06 | Port occupied by unrelated process | **PORT CONFLICT / UNVERIFIED SERVICE**; leave unrelated process alone | Process ownership and occupied-port proof |
| B07 | Engine responds to HTTP but with wrong product/protocol | **REACHABLE, INCOMPATIBLE**; no privileged Engine actions | Response and supported-protocol comparison |
| B08 | A process claims correct product/SHA but has no trust proof | Never mark **AUTHENTICATED**; show claimed/reachable only | Origin of each verification field and trust status |
| B09 | Engine exits unexpectedly; refresh UI | **DISCONNECTED/CRASHED**, correct previous instance/exit evidence; never stale READY | PID exit check, bounded logs, health timestamps |
| B10 | Restart Processor backend while launcher/Engine survive | Reconcile same Engine rather than launching duplicate; correct session ownership | Run IDs before/after and process list |
| B11 | Stop Demo with only a session-owned Engine | Graceful stop of owned Engine, wait/reap, report exit or recorded failure; preserve data directory | PID/process descendants before/after; files still present |
| B12 | Stop Demo while attached to independently started Engine | **Do not stop** external Engine | OS PID and original Engine endpoint still responding |
| B13 | Stop Demo twice or after partial startup | Idempotent status; no unrelated process killed; no orphan session child | Request receipts + process tree |
| B14 | Ensure request contains unapproved executable path/args or traversal | **Refuse**; no shell injection/execution and no writes outside approved data root | Rejection receipt, no process created |
| B15 | Simulate Engine slow startup or startup exit | **STARTING → FAILED/TIMEOUT** with actual error; no indefinite spinner or guessed timing | Captured elapsed time, exit/status, logs |

## 4. Phase C — Experience and evidence integrity (P0)

| ID | Preconditions + action | Required outcome | Proof to capture |
| --- | --- | --- | --- |
| C01 | Engine running but no Identity grant endpoint | UI says **AUTHORIZATION INTEGRATION NOT READY**, not **AUTHORIZED**. Protected observation control disabled. | Engine mode + UI + endpoint evidence |
| C02 | Engine worker missing/unreachable | Engine may be CONNECTED while worker says **UNAVAILABLE**; no default model/response substituted | Actual worker status and request count |
| C03 | Identity, GitHub or publisher connection not verified | Each independently says NOT VERIFIED/UNKNOWN; UI does not claim full first-run completion | Source and last-check times |
| C04 | Read actual Engine metrics and diagnostics | Values match authenticated actual responses where supported; no invented counts, app uptime or status | Request/response correlated to UI value |
| C05 | Copy Context with no services connected | Contains actual Processor and test context only, clear Unknowns; no fake service IDs | Captured sanitized clipboard contents |
| C06 | Copy Context when tokens/config/observations exist | No credentials, signing keys, bearer tokens, private observations or unapproved personal paths; allowed metadata accurate | Redaction review with canary secrets, inspected clipboard |
| C07 | Lose Engine network/port after status was shown | Previous state becomes **STALE/UNREACHABLE** after actual failed check, never stays fresh green | Last-check times, induced failure and UI state |
| C08 | Start a manual test observation UI without real authenticated source | Disabled or explicitly blocked, not send with fabricated source identity | Network trace: zero unauthorized sends |
| C09 | Click all visible commands in the demo | Only genuinely supported controls are enabled; each records requested/accepted/observed or an explicit refusal | Operation IDs and actual state changes |
| C10 | Open original Engine UI; return to Processor | Only actual Engine URL is used; opening preview doesn't change grant or launch another Engine | Real URL/product and process count |

## 5. Phase D — Identity permission integration (P0 security; **BLOCKED until real Identity exists**)

**Prerequisite:** Identity has authenticated registered services and admin approval plus real current grant check, and Engine has passed its independent interop tests. A local fixture is **not enough**.

| ID | Preconditions + action | Required outcome | Proof to capture |
| --- | --- | --- | --- |
| D01 | No grant for S→E, entity X; submit from authenticated S | **DENY/403**, no event persisted as accepted and zero new model calls | Identity decision + Engine refusal and worker count |
| D02 | Explicit owner approves exactly S→E, X, ingest | Identity returns verified grant ID/version, Engine accepts one **authenticated** manual event | Identity authority record + Engine event 202 receipt |
| D03 | Change entity X→Y or sender S→S2 | Denied despite permitted X/S, even if payload claims approved source | Signed scope + Engine refused receipt |
| D04 | Revoke while Engine running, submit next observation | Next event **403** without restart; no new worker call | Revoke receipt + unchanged PID + worker count |
| D05 | Revoke after an event is queued but before model call | Engine rechecks and does not dispatch queued event | Queue decision/receipt, model request count |
| D06 | Stop Identity, submit new protected observation | **AUTHORITY UNAVAILABLE / 503**, not allow; no new model send | Real outage event, Engine 503, zero new calls |
| D07 | Authority responds malformed/unsigned/wrong nonce | Fail closed, not ALLOW | Request/response proof and Engine refusal |
| D08 | Worker had data before revocation | UI does **not** claim transmitted content was withdrawn; accurately reports in-flight limit | Timestamped event/worker/revoke ordering |

## 6. Phase E — Product onboarding and entitlements (**BLOCKED by owner decisions**)

| ID | Preconditions + action | Required outcome | Proof to capture |
| --- | --- | --- | --- |
| E01 | Fresh user not connected to GitHub | **SETUP/REPAIR ONLY**, never full operational READY | First-open screen and non-creation audit |
| E02 | User connects GitHub account | Only account and repository permissions actually verified may be marked VERIFIED; no invented username | GitHub evidence with no token exposure |
| E03 | New starter repo requested before approved contents/license method | Creation unavailable/blocked with precise explanation, not made-up starter or forced copy | Product decision state and GitHub non-mutation |
| E04 | Approved create/select mechanism becomes available later | Only with explicit user confirmation may own source repo be created/selected; verified origin and actual files | Real GitHub response and source file audit |
| E05 | Publisher feature permission unknown/denied | No protected new download; local grants kept separate | Publisher boundary decision and zero download attempt |
| E06 | Returning user/repository already linked | Reuse verified install/repo; no duplicate user repo/identity | Actual GitHub IDs, before/after object count |
| E07 | GitHub unreachable on return | Display stale/repair without inventing fresh account/repo proof; do not delete local user data or kill unrelated services | Outage and file/process state |

**No E-series test may be marked as end-to-end product PASS until the relevant owner decisions, GitHub permissions and actual services exist.**

## 7. Repetition, resilience and measurements

**Minimum repeated real trials for Gate 1–3 candidate:** 3 independent clean start→Ensure→inspect→Stop runs in disposable directories **plus** 3 attach/reconnect passes, each with process evidence. Repetition is to expose flakiness, not to assert a statistical reliability guarantee.

**Performance measurements (measure first, do not invent an SLO):**

- Build duration; Processor startup-to-health; launcher request-to-process-start; Engine process-to-verified-health; initial workspace-to-status; authentication/authorization latency when implemented; event submit-to-202; model submit-to-first/completion only if actually observed; stop-to-exit; observed CPU/RAM only if sampled from actual OS.
- Report **per-run** durations and failed/missing measurements; provide median and maximum **only after** collecting enough real samples, and explicitly say the sample count.
- No assumed message rate, model cost, API quota, “<1 second” promise or permanent uptime percentage from a single demo.

**Test isolation:** current OS only; explicit branch SHA; local loopback; disposable data root; explicit test entity; no remote/VM claims. Avoid user-owned production GitHub repository mutation. Clean rollback: stop only children created by the test; leave external Engine untouched; archive/retain safe test logs as owner directs; don't remove unrelated data.

## 8. Gate outcomes

| Gate | Accept if | Reject / block if |
| --- | --- | --- |
| **Gate 0** | Actual existing APIs and blockers inventoried | Spec guesses treated as implementation |
| **Gate 1** | A01–A07 real evidence; no fake-ready | Crashes or misrepresents optional services |
| **Gate 2** | All B-series P0 tests PASS in real processes, including negative/process-ownership tests | Duplicate/orphan/killed external process, wrong-build execution, false authenticated state |
| **Gate 3 — first owner trial** | C-series truthful results; full setup marked incomplete until verified; owner can understand and troubleshoot one real Engine | Fake green status, exposed secret, cosmetic/inert controls, inferred Identity authorization |
| **Gate 4 — real permission** | D-series pass against **real Identity and Engine**, not fixture | Any hidden local grant bypass or missing authority |
| **Gate 5 — normal customer experience** | E-series pass after owner defines licensing/bootstrap/entitlement policy | Invented starter, repository ownership, feature access or legal terms |

Claude must report **a partial Gate 1/2/3 as a partial prototype**, not say all eight story chapters are implemented. The owner may accept the candidate for further testing without promoting it.

## 9. Owner usability tests (ask after real Gate 3 demo)

Record the owner's own words, not only the developer's opinion:
1. Without explanation, can they tell which component is **running** versus **authorized**?
2. Can they identify the next safe action when Engine is stopped, incompatible or worker unavailable?
3. Do they know whether Stop Demo will stop a session-owned or externally attached Engine?
4. Does Copy Context contain sufficient grounded details for another AI to diagnose a failure?
5. Does the page feel like opening a real product, rather than inspecting a code repository?
6. Which one part should be simplified or removed before making the blueprint permanent?

**The highest-priority failure is a false claim of success or an unauthorized operation**, even if every visual element looks polished. Keep no unverified result labeled PASS.
