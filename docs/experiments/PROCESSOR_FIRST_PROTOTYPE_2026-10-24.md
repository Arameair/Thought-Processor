# Thought Processor — first prototype: report (2026-10-24)

Human-directed, AI-assisted (Claude). Built from the [story prototype](https://github.com/Arameair/Thought-Processor/tree/1c8ebc4c1d4ed0deded8e866968b5df6a7e02230) and its [start prompt](https://github.com/Arameair/Thought-Processor/blob/1c8ebc4c1d4ed0deded8e866968b5df6a7e02230/docs/build/CLAUDE_START_PROMPT.md).

## The rule that governed this build

The start prompt carries a banner: it is superseded by the [Identity-approved build package](https://github.com/Arameair/Thought-Processor/blob/be6904618a9533513746f8e9aaf353ece37498f8/prototype/START_HERE.md) wherever attach/control is concerned. Its **Gate 2** asks Processor to read the Engine's `/api/status`; the package forbids reading anything but public metadata before an Identity-approved attachment, and that endpoint returns entity cards, grants and the activity log. **The owner chose the strict rule** (2026-10-24, in chat). Gate 2 is therefore BLOCKED, with a precise request to the Engine team.

## Status

| Claim | Status |
| --- | --- |
| BUILT | Yes — Gate 1 complete |
| LOCALLY TESTED | Yes — Ubuntu 24.04 cloud VM, **not** your Windows/WSL/Debian machines |
| Integration with Identity/Engine authority | BLOCKED (no producer, no runner, no Engine attach boundary) |
| OWNER ACCEPTED | PENDING |
| Promotion | NOT AUTHORIZED — Alpha, Stable, Main untouched |

## Revisions

| Item | Value |
| --- | --- |
| Branch | `experiment/processor-first-prototype-20261024` |
| Alpha base | `9fac9fa44cb020bb6f8aa6bcb1e9faeda6e0b47a` |
| **Tested code SHA** (embedded in binary, `vcs.modified=false`) | `db2a46c5bf930239f286bf412872c67953946d8b` |
| Carried forward | Code from `experiment/processor-diagnostic-shell-20261024`, cherry-picked with `-x` provenance (`b5d27042`, `5153a158`, `a023e684`, `37255d91`) |
| Identity used (external, started by the test script) | `identity` `experiment/first-admin-wizard-20261023` @ `4686c94f7bdb84ebb8abf70e7409c87580dc461e` |
| Engine used (external, started by the test script) | `Thought-engine` `experiment/identity-grant-enforcement-20261024` @ `9ecd863645c24d3d7ad113426fefeeedbcbcd75e`, no worker |
| Toolchain | Go 1.24.7 linux/amd64; Chromium 153 (Playwright, Python 3.13) |

Collaborator check at build time: Identity [#16 Start Experience](https://github.com/Arameair/identity/issues/16) and [#15 grant check](https://github.com/Arameair/identity/issues/15) are design-only; Engine [#1](https://github.com/Arameair/Thought-engine/issues/1) is a design gate. No new code on either side since the survey in the [dependency request](../contracts/IDENTITY_ENGINE_LIFECYCLE_APPROVAL_REQUEST.md).

## Gates

| Gate | Verdict | Why |
| --- | --- | --- |
| G0 Survey | **PASS** | Exact SHAs, endpoints and blockers recorded in the [dependency request](../contracts/IDENTITY_ENGINE_LIFECYCLE_APPROVAL_REQUEST.md) |
| G1 Standalone workspace | **PASS** | Live tests below; owner usability not yet tested |
| G2 Read-only live Engine status | **BLOCKED** | Owner chose strict rule; Engine has no public status endpoint and no approved attach boundary |
| G3a Identity Start Experience | **BLOCKED** | Identity #16 not implemented |
| G3b Real authorization + observation | **BLOCKED** | Identity grant producer not implemented; no authenticated demo source |
| G4 Owner trial | **NOT RUN** | Needs you — procedure below |

## What you can do in it

- Start it on its own; it prints a one-time sign-in link. The first screen states what it is, what's missing and **one next step with a button**.
- **Entity / Feature / Running instance** strip: each filled only from evidence. Today: no entity (Identity's list is private), the Engine address you entered as a *candidate*, and an *unidentified* instance.
- **Identity card**: enter its address; Processor checks it and shows *Unreachable*, *Wrong program* or *Found · unverified* with the claim labeled UNVERIFIED, plus freshness that turns **Stale** after 20 s.
- **Engine card**: port check only, disabled **Open the Engine's own page** with the reason (no embedded or rebuilt Engine screen).
- **Engine controls**: start / attach / stop / send observation, disabled with reasons. **Prove it** sends the real request so you can see the server refuse it with no effect.
- **Diagnostics**: *Flow* (each request as requested → authorized → executed → observed, missing stages labeled), *Logs* (filter, Copy logs), *Metrics* (Processor's own counters and check timings; what it cannot measure is listed), *What works*, *Copy context*.
- **Stop Processor** stops only Processor.

## Test results

One live run, 2026-10-24 09:06:50–09:07:28 UTC, on the tested SHA, real processes ([raw evidence](evidence-2026-10-24-prototype/evidence.md)). The same harness also passed twice on the preceding code commit `fcfc5164`, which differs only by one page link and README text.
Layers: **L1** unit, **L2** labeled fixture, **L3** real processes. Totals: **23 PASS · 15 BLOCKED · 2 NOT RUN · 0 FAIL**. Go tests: 13 — 12 pass, 1 skipped as root.

### Story test design (`docs/build/TEST_DESIGN.md`)

| ID | Verdict | Layer | Evidence (abridged) |
| --- | --- | --- | --- |
| UX-01 | PASS | L3 | Clean profile: Not set / Not set, next = *Tell Processor where Identity runs*, button focuses the field; 4/4 controls disabled; no fake states |
| UX-02 | PASS | L3 | Remembered Identity gone after restart: *Not checked yet*, then *Unreachable* — never shown live from the saved record |
| UX-03 | NOT RUN | — | No real non-thinking feature exists to register |
| UX-04 | PASS | L3 | Engine's own page: disabled with reason; no embed or clone (= A22) |
| UX-05 | PASS | L3 | Entity / feature / instance shown separately with distinct labels |
| UX-06 | PASS (partial) | L3 | 390 px phone width: 0 px horizontal overflow; long addresses wrap. Keyboard-only use and screen readers NOT tested |
| OPS-01 | PASS | L3 | Ready in 24 ms, pid 1911, port 7860, own profile, no Project Controller |
| OPS-02 | PASS | L3 | Stop Processor → exit 0 in 116 ms; Identity and Engine alive; restart 22 ms; old session refused; address kept and re-checked |
| OPS-03 | PASS | L3 | Second instance on same profile: exit 1 naming the running run/pid/URL; busy port: exit 1, other program untouched |
| OPS-04 | PASS | L3 | After `kill -9`: restart 22 ms, reports unclean previous run, takes no action against anything |
| DISC-01 | PASS | L3 | Unused port → *Unreachable* (CLOSED) |
| DISC-02 | PASS | L3 | Real `python http.server` → *Port open; no Identity metadata* (404); a second real Processor → *Wrong program* (claims `thought-processor`). Neither authenticated |
| DISC-03 | BLOCKED | — | Reading Engine `/api/status` forbidden before approved attach (owner decision) |
| DISC-04 | PASS | L3 | Engine stopped → next check *Unreachable*; untouched results go *Stale* after 20 s (= A21) |
| DISC-05, 06, 07, 08 | BLOCKED | — | Need Engine status/identity/protocol reads or commands |
| SAFE-01 | PASS | L3 + L2 | Copy Context and log hold no code, cookie, CSRF or remote text; canary in a fixture reply never leaks |
| SAFE-02 | PASS | L3 | Remote, `file://`, credential and path addresses refused; no proxying |
| SAFE-03 | PASS | L3 + L2-live fixture | Identity address that never answers: check ends in 1504 ms as FAILED; page read during it answered in 2 ms |
| SAFE-04 | PASS | L3 | Non-owned process on Processor's port: refused to start, nothing killed |
| SAFE-05 | PASS | L3 | Profile path is a file: exit 1 with clear message, file unchanged, no port opened |
| SAFE-06 | BLOCKED | — | No configuration is applied to any service |
| SAFE-07 | PASS | L3 | 3 clean cycles: ready 22/22/23 ms, stop 3/3/3 ms, exit 0, no children |
| LAUNCH-01…06 | BLOCKED | — | Identity Start Experience not built |
| AUTH-01…12 | BLOCKED | — | No Identity grant producer; no authenticated demo source |
| FLOW-01…06, 08 | BLOCKED | — | No authorized observation path; no worker |
| FLOW-07 | PASS | L3 | Flow shows *Authorized: NOT AVAILABLE, Executed: NOT ATTEMPTED, Observed: NONE* — nothing reconstructed |
| UX-07 | NOT RUN | — | Owner trial |

### Identity-approved acceptance (`prototype/ACCEPTANCE.md`)

PASS: A01, A02, A11a, A13, A16, A18, A19, A20, A21, A22, A27 (+ live-Identity check).
BLOCKED: A03–A10, A11b, A12, A14, A15, A17, A25, A26.
NOT RUN: A23, A24 (GitHub onboarding out of scope).
Highlights: every protected call → HTTP 501 with an operation id and no effect; cross-origin, rebinding Host, missing/wrong CSRF and GET-mutation attempts all refused (403/421/401/405), and a real Chromium cross-origin `fetch` was blocked; a raw listener posing as the Engine received **0 bytes**; the binary has no `os/exec` dependency.

## Timings (actual)

| What | Samples |
| --- | --- |
| Build (warm) | 1033 ms; binary 9.8 MB |
| Exec → answering | 24, 22, 22, 22, 22, 23 ms |
| Stop | SIGTERM 3/3/3 ms; Stop button 116 ms |
| Identity check (loopback) | median 0 ms, max 1500 ms (the deliberate stall) over 15 checks |

No decision, runner or Engine-processing timings exist (BLOCKED).

## Process ownership

Processor started zero processes in every test and stopped only itself. Identity, Engine and the unrelated HTTP server were started and stopped by the test script. Orphan audit after the run: none.

## Owner trial (Phase 1 of the trial card) — about 15 minutes

Linux/WSL/macOS with Go 1.22+:

```bash
git clone -b experiment/processor-first-prototype-20261024 https://github.com/Arameair/Thought-Processor
cd Thought-Processor/processor && go build -o thought-processor .
./thought-processor -data ~/tp-trial          # Windows: go build -o thought-processor.exe . ; .\thought-processor.exe -data %USERPROFILE%\tp-trial (UNTESTED)
```

1. **Clean start.** Open the printed link. Check: nothing says Connected, Authorized or LIVE; the next step tells you what to do.
2. **Wrong port.** Click **Enter Identity address**, type `127.0.0.1:9`, **Save** → *Unreachable*.
3. **Real Identity.** Start your Identity yourself (a build with `/api/health`, e.g. first-admin-wizard, default port 7796). Enter its address → *Found · unverified*; next step *waiting on Identity*.
4. **Engine.** Start your Thought Engine yourself (default 7798), enter its address → *Port open — not identified, not attached*. Note: queue/worker are **not** shown (strict rule).
5. **Prove the lock.** Engine controls → *Prove it* → **Ask server: start** → HTTP 501, BLOCKED, no effect. Open **Diagnostics → Flow** to see the four stages.
6. **Diagnostics.** Check Logs, Metrics, then **Copy context** and paste it somewhere: no codes or secrets.
7. **Close / reopen.** Close the tab and reopen the link from the terminal → still signed in. **Stop Processor**; confirm Identity and Engine still run. Start again → new link; old tab asks you to sign in; your addresses remain.
8. Fill in the trial card scorecard. Phases 2 and 3 are **BLOCKED** — stop there.

Expected traffic lights for this candidate: First-run 🟢?, Runtime evidence 🟡 (Gate 2 blocked), Independence 🟢?, Identity authority BLOCKED, Guided test BLOCKED, Logs/metrics 🟢?, Stop/restart 🟢? — the "?" marks are for you to confirm.

## Known problems

1. **Not tested on your OS.** Windows has an untested lock fallback: after a crash you must delete `runtime/processor.lock` by hand.
2. **No live Engine information** (queue, worker, build) — by decision; the page can only say "port open".
3. **Identity shown is only a claim.** No Identity build can prove itself or approve anything.
4. **Flow list grows long** (up to 30 entries, no collapsing) — cosmetic.
5. **Change detection waits for the 10 s re-check** while the page is open; there is no background check when no page is open.
6. **Sign-in each run.** Sessions die on restart by design; you need the new link.
7. **The `/api/health` Identity check works only on Identity branches that have it**; others show "no recognizable metadata".
8. Copy Context and logs include the loopback addresses you entered (deliberately; they are not secrets).

## Recommendation for the next iteration

**KEEP for the next experiment, owner verdict pending.** The Processor is useful today only as an honest setup/diagnosis shell; nearly all of its remaining value is blocked by two small, well-defined pieces on the other teams' side. In priority order:

1. **Engine team:** add a public, minimal status endpoint (product, protocol version, instance id, build commit, ready yes/no) that contains no entities or logs. That unblocks DISC-03/05/07 and much of Gate 2 *without* any authority work.
2. **Identity team:** the Identity side of the existing ingest check (`/v1/authorization/check` that the Engine experiment already calls). That unblocks Gate 3b, the story's core proof.
3. **Owner decision:** who owns the runner (Identity's Start Experience or a separate launcher). Until decided, start/attach/stop stays blocked by design.
4. **Processor (this repo), after 1–2 exist:** consume the public Engine status, then the guided test with one authenticated demo source — still never on the data path.

## Rollback

Delete the profile folder and the branch (`git push origin --delete experiment/processor-first-prototype-20261024`). Nothing else changed.

**Owner verdict: PENDING. Promotion: NOT AUTHORIZED.**
