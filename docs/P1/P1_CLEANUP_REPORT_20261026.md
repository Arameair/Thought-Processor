# P1 cleanup — report (2026-10-26)

Assignment: [Issue #23](https://github.com/Arameair/Thought-Processor/issues/23) · Grok's report: [Issue #22 comment](https://github.com/Arameair/Thought-Processor/issues/22#issuecomment-6422812740).
**Experimental only.** Nothing promoted; `alpha`, `stable`, `main` and the original P1 branches are untouched.

| Repo | New branch | Created from (P1 baseline) | Code commit |
| --- | --- | --- | --- |
| Arameair/identity | `experiment/p1-cleanup-claude-20261026` | `7cc5d9d717e5` | `64940b629a5e` |
| Arameair/Thought-Processor | `experiment/p1-cleanup-claude-20261026` | `e78d638e48a9` | `49bed222edede` (+ evidence / docs commits) |

## Method

1. Wrote `processor/acceptance/p1_findings.py`, which turns Grok's M1–M3, L1, L3 and L5 into concrete attacks against real processes, and ran it on the **unchanged baseline**: *secure behaviour observed: 3 of 15* ([before](evidence/cleanup/before/findings-report-BEFORE.txt)).
2. Started decoy listeners on 7790 / 7860 / 8790 (`acceptance/decoy_ports.py`, which answers like an unrelated Identity) and ran the baseline tests: Processor `go test` **FAIL** (`TestProtectedActionsFailClosedWithoutIdentity` found the decoy as `FOUND_NOT_CONNECTED`), P0 harness **crashed 0/4** — Grok's M4 reproduced ([before](evidence/cleanup/before/)).
3. Fixed, then reran everything **with the decoys still running** (and left them running untouched): *15 of 15* ([after](evidence/cleanup/after/findings-report-AFTER.txt)), all unit tests with `-race`, P1 integration 81/81, UI tour PASS, P0 27/27.
4. Pushed, then repeated from a **fresh clone of the pushed branches with a fresh HOME** following START_HERE: build 23 s, unit 39 s, integration 81/81 (101 s), findings 15/15 (32 s), P0 27/27 (46 s), UI tour PASS (40 s).

## Findings: before → after

| Grok | Before (baseline, observed) | Fix | After (observed) |
| --- | --- | --- | --- |
| **M1** self-declared trust | New code in a trusted folder labelled `TRUSTED_LOCAL_TEST_FIXTURE` was offered, Processor install `200`, Identity install request `202` | Trust = **exact pinned artifact digest** (built-in reviewed fixture, or operator `-trust-feature-artifact` / `-trust-fixture-artifact`). Processor checks before listing, staging, registering and every start; **Identity enforces independently** at request, commit, every ticket and every live check (`ARTIFACT_NOT_PINNED`). Interpreter resolved and recorded at install (L4). | Fake: **BLOCKED** in Library, Processor `409`, Identity `403 ARTIFACT_NOT_PINNED`. Real fixture still installs and runs. |
| **M2** revocation by anyone paired | Program D revoked program C's grant: `200` | Direct revoke only by the grant's requester, the program that asked for it, or the destination feature. Others: `403 REVOKE_NOT_PERMITTED`. **Admin emergency path** kept: `revoke_grant` approval → elevated `admin-revoke`. `revoked_by` / `revoked_at` recorded and shown. | D → `403`, C's grant still ACTIVE. Processor can't revoke actor B's grant directly (`403`), the Admin can (`revoked_by admin:adm_… (approval apr_…)`). |
| **M3** approval clarity / expiry | No requester label, no warning, grant without expiry = forever, "until revoked" approvable silently | Identity's page shows a `display` built only from Identity's records: requester label + paired time (from the pairing store; the label printed beside the pairing code), **warning** when it isn't the primary Thought Processor (the one that requested Admin enrollment), feature + trust basis, the one action + description, identity, *sources: none*, expiry, effect; raw IDs/digests under *Advanced details*. **Default expiry 1 hour**; "until revoked" needs an extra tick (else `400`, and can't be committed). Passphrase still only in Identity's page. | Label shown, warning shown for program D, default expiry set, until-revoked refused without the tick (`decide 400`, `commit 403`). [Screenshot](evidence/cleanup/screenshots/ui-13-identity-grant-approval.png). |
| **M4** port 7790 conflicts | See Method 2 | Tests never probe the default port (`TP_IDENTITY_DEFAULT_URL="-"` / `-identity-default-url -`), use OS-chosen ports, private locator dirs; regression test with a decoy on 7790; WRONG_SERVICE now names expected vs. actual installation and says nothing was trusted or switched. | All suites pass with decoys on 7790/7860/8790; decoys still answering afterwards (never stopped or used). |
| **L1** outage-spent ticket looked like a replay | `503` then `409 TICKET_REPLAYED` | Feature ledger records the state; resend → `409 TICKET_SPENT_DURING_OUTAGE` (**still refused**; single use unchanged). The 503 says `ticket_spent: true`. | `503 AUTHORITY_UNAVAILABLE` / `409 TICKET_SPENT_DURING_OUTAGE`; 12 simultaneous sends → exactly one allowed. |
| **L2** global pairing lockout | — (reviewed) | Pairing code bound to the `processor_id` that requested it; other programs get `403` and neither burn it nor count toward lockout. | Unit/HTTP test: 6 wrong-program attempts, then the right program pairs. |
| **L3** file permissions | `observations.jsonl` 0644 | Feature `umask 077`, files 0600; harness work dirs 0700, console files 0600. | 0600 (checked under umask 022). |
| **L4** interpreter PATH/TOCTOU | — (reviewed) | Interpreter resolved (LookPath + symlinks) at install, absolute path recorded and used; missing → `REQUIREMENT_MISSING`. | Documented residual: a same-user process can still replace the interpreter binary itself (no OS isolation). |
| **L5** fault injection always on | Dropped fault file crashed the feature | Honoured only when Processor runs with `-test-feature-faults` (env `TF_FIXTURE_TEST_MODE=1`), with a TEST MODE banner. | Without the flag: file ignored, request `COMPLETED`. |
| **R3** tracked `.pyc` | tracked | removed; `__pycache__/`, `*.pyc` ignored in both repos | — |
| **R4** Playwright/PEP 668, ports, cleanup | missing | START_HERE: venv steps, port-conflict guidance (`-port`, decoys), cleanup | venv + `pip install playwright` verified; browser download **NOT RUN** here (sandbox network) — the tour ran with a pre-installed Playwright. |

## Tests run (all with decoys on 7790 / 7860 / 8790)

| Suite | Result |
| --- | --- |
| Identity `go test -race -count=1 ./...` | all ok (new: pinned-artifact rule, revoke authority, display/primary warning, default expiry, until-revoked confirmation, pairing-code binding, admin-revoke in route inventory) |
| Processor `go test -race -count=1 ./...` | all ok (new: built-in pin == fixture, self-labelled package blocked, operator pin, default-port isolation with a 7790 decoy) |
| `acceptance/p1_findings.py` | 15/15 secure (baseline 3/15) |
| `acceptance/p1_integration.py` | **81/81** (was 74; new: requester display, 1-hour default, until-revoked confirmation, 12× concurrent replay, cross-program revoke refused both ways, Admin emergency revocation) |
| `acceptance/p1_ui_tour.py` | PASS (13 checks incl. 1-hour default and the new approval page) |
| `acceptance/p0_integration.py` | 27/27 (crashed 0/4 at baseline with the decoys) |
| Fresh clone + fresh HOME | all of the above PASS |

Adversarial coverage (integration + findings): absent / wrong grant, wrong identity / capability / requester / recipient, forged, tampered, replayed, concurrent replay, expired ticket, expired grant, revocation (incl. with an outstanding ticket), Identity outage, account-switch takeover refused, restart persistence with stable installation IDs, crash mid-install resumes with the same ID, orphan cleanup, bounded recovery. Reports, logs and screenshots were scanned: no pairing codes, link fragments, passphrase or credential contents. Only the reviewed test fixture was ever installed; ChatJimmy is never called.

## Existing data

- Grants stored before the cleanup keep their stored expiry (or none, shown as *never (granted before expiry defaults)*); they can be revoked by their requester, the destination feature or the Admin.
- Installations of the **old** fixture digest are now refused (Processor won't start them; Identity denies their tickets). Reinstall. START_HERE says to use fresh profiles.
- An `admin.json` from before the cleanup has no `primary_requester`: every approval then shows *Identity does not know which paired program is your primary Thought Processor* (safe default).

## Remaining risks / limits

- **HIGH — no OS isolation** (unchanged, not claimed): any same-user process can read credential files, the Admin verifier, call loopback ports, or replace the interpreter. Pins protect against *unreviewed packages*, not against hostile local code.
- **HIGH — passphrase authenticator** (offline-guessable/key-loggable by same-user malware); Windows Hello/passkey BLOCKED; recovery DEFERRED.
- MEDIUM — the "primary Thought Processor" is the program that asked for Admin enrollment; a malicious paired program that copies its label still triggers the warning (bound to principal ID), but the person must read it.
- MEDIUM — any paired program can still *request* a new pairing code (replacing the pending one) and *request* approvals; requests grant nothing.
- MEDIUM — pins are a hard-coded list + operator flags; there is no signed release provenance.
- LOW — loopback HTTP is unencrypted; features don't follow Identity to a new port (fail closed).

## Grok comparison

M1, M2, M3, M4, L1, L3, L5, R3, R4: **fixed** (verified above). L2: **fixed** (bounded change). L4: **fixed for PATH substitution; binary replacement documented** (needs OS isolation). Anything else in Grok's report about isolation, authenticator strength and recovery remains as documented limits, not claimed fixed.
