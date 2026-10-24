# Live acceptance evidence (generated)

```json
{
  "processor": {
    "repo": "Arameair/Thought-Processor",
    "branch": "experiment/processor-first-prototype-20261024",
    "sha": "db2a46c5bf930239f286bf412872c67953946d8b",
    "alpha_base": "9fac9fa44cb020bb6f8aa6bcb1e9faeda6e0b47a"
  },
  "identity": {
    "repo": "Arameair/identity",
    "branch": "experiment/first-admin-wizard-20261023",
    "sha": "4686c94f7bdb84ebb8abf70e7409c87580dc461e",
    "role": "external program started by the test script"
  },
  "engine": {
    "repo": "Arameair/Thought-engine",
    "branch": "experiment/identity-grant-enforcement-20261024",
    "sha": "9ecd863645c24d3d7ad113426fefeeedbcbcd75e",
    "role": "external program started by the test script"
  },
  "go": "go1.24.7 linux/amd64",
  "os_info": "Linux-6.18.44-fc-v153-x86_64-with-glibc2.39",
  "python": "3.13.17",
  "started": "2026-10-24T09:06:50Z",
  "processor_bin": "thought-processor",
  "work_root": "<tmp>/tp-live-kptgbfpp",
  "first_startup_ms": 24,
  "clean_cycles": [
    {
      "pid": 2062,
      "ready_ms": 22,
      "self_reported_startup_ms": 1,
      "stop_ms": 3,
      "exit": 0,
      "leftover_children": []
    },
    {
      "pid": 2070,
      "ready_ms": 22,
      "self_reported_startup_ms": 2,
      "stop_ms": 3,
      "exit": 0,
      "leftover_children": []
    },
    {
      "pid": 2077,
      "ready_ms": 23,
      "self_reported_startup_ms": 2,
      "stop_ms": 3,
      "exit": 0,
      "leftover_children": []
    }
  ],
  "orphan_audit": {
    "thought-process": [],
    "identity": [],
    "thought-engine": []
  },
  "finished": "2026-10-24T09:07:28Z",
  "screenshots": [
    "01-signed-out.png",
    "02-first-screen-empty-profile.png",
    "02b-first-screen-phone-width.png",
    "03-identity-absent-start-refused.png",
    "04-identity-found-unverified.png",
    "05-identity-and-engine-candidates.png",
    "07-diagnostics-flow.png",
    "08-diagnostics-metrics.png",
    "06-processor-stopped.png"
  ]
}
```

| ID | Verdict | Layer | Exercise | Observed | Limitation |
|---|---|---|---|---|---|
| A01 | PASS | L3 | Start Processor with empty isolated profile; open in Chromium; sign in | pid 1911 port 7860 ready in 24 ms; run run_80529921bd4754b8; build db2a46c5bf930239f286bf412872c67953946d8b modified=false; /api/state without session -> 401; identity=NOT_CONFIGURED engine=NOT_CONFIGURED next=set_identity; ops enabled=[False, False, False, False]; children of Processor=[]; identity/engine processes={'identity': [], 'thought-engine': []}; code removed from URL=True; screenshots 01,02 |  |
| UX-01/06 | PASS | L3 | One useful next action on first screen; 390 px wide phone viewport | Next button 'Enter Identity address' moves focus to #identity-addr; horizontal overflow at 390 px = 0px; screenshots 02, 02b | Owner usability (can a non-technical person follow it) is NOT tested here; that is the owner trial |
| A02 | PASS | L3 | Identity absent; try start/attach/stop/observe from UI and direct API | identity candidate 127.0.0.1:7796 -> UNREACHABLE (CLOSED); next=start_identity; API: {start: 501 BLOCKED blocked_no_identity_approval_service op_5d68d7f061a6e4cf, attach: 501 BLOCKED blocked_no_identity_approval_service op_e844fa0c4308306b, stop: 501 BLOCKED blocked_no_identity_approval_service op_a3f77ff10edaba13, observe: 501 BLOCKED blocked_no_ingest_grant_contract op_2bb54704306d8dae}; UI buttons disabled=[True, True, True, True]; UI 'Ask server: start' -> HTTP 501 BLOCKED; Processor children=[]; thought-engine processes=[] |  |
| A27 | PASS | L3 | Missing Identity producer/enforcement interface | Gate 2/3 integration BLOCKED; diagnostic shell only; every protected call refused 501 before effect; no bypass flag exists in the binary (see A19 for symbol/dependency check) |  |
| A27-live-identity | PASS | L3 | Real Identity (first-admin-wizard build) running; Processor inspects it and retries start | Identity pid 1951; probe port=OPEN metadata=RESPONDED http=200 claim app='identity' (UNVERIFIED) in 1 ms; state=CLAIMS_IDENTITY_UNVERIFIED; next=blocked_contract; start -> 501 blocked_no_identity_approval_service; Identity still alive=True | Identity has no decision endpoint, so Processor cannot ask it anything; only public health metadata was read |
| A03 | BLOCKED | — | Real Identity explicitly denies | No Identity build exposes a start/attach/stop decision; no runner/Engine control boundary validates one. Nothing to deny, time out, mis-scope or replay. Processor refuses all four actions with 501 regardless (A02). | Processor-side probe timeout/malformed/redirect classification was tested only with labeled L2 fixtures (TestIdentityProbeClassificationFixture); that is NOT decision-path evidence |
| A04 | BLOCKED | — | Identity times out / malformed / unverifiable decision | No Identity build exposes a start/attach/stop decision; no runner/Engine control boundary validates one. Nothing to deny, time out, mis-scope or replay. Processor refuses all four actions with 501 regardless (A02). | Processor-side probe timeout/malformed/redirect classification was tested only with labeled L2 fixtures (TestIdentityProbeClassificationFixture); that is NOT decision-path evidence |
| A05 | BLOCKED | — | Decision targets wrong actor/install/service/action/manifest | No Identity build exposes a start/attach/stop decision; no runner/Engine control boundary validates one. Nothing to deny, time out, mis-scope or replay. Processor refuses all four actions with 501 regardless (A02). | Processor-side probe timeout/malformed/redirect classification was tested only with labeled L2 fixtures (TestIdentityProbeClassificationFixture); that is NOT decision-path evidence |
| A06 | BLOCKED | — | Expired/revoked/replayed decision | No Identity build exposes a start/attach/stop decision; no runner/Engine control boundary validates one. Nothing to deny, time out, mis-scope or replay. Processor refuses all four actions with 501 regardless (A02). | Processor-side probe timeout/malformed/redirect classification was tested only with labeled L2 fixtures (TestIdentityProbeClassificationFixture); that is NOT decision-path evidence |
| A09 | BLOCKED | L3 | Attach to correct external Engine that is running | Engine pid 1963 on 7798: Processor saw port=OPEN metadata=NOT_REQUESTED (did not read /api/status); attach -> (501, 'blocked_no_identity_approval_service'); Engine untouched and alive=True; thought-engine processes=[1963] | Real attach approval does not exist; the refusal and non-interference were observed (ok) |
| A07 | BLOCKED | L3 | Attach-only approval with Engine stopped | No approval type exists; attach and start are both refused 501 before effect; this build has no runner so it cannot spawn. |  |
| A08 | BLOCKED | L3 | Start approval, correct Engine stopped | start -> (501, 'blocked_no_identity_approval_service'); no runner exists; zero Engine processes started by Processor (children of Processor=[]) |  |
| A10 | BLOCKED | L3 | Parallel/repeated ensure incl. Processor restart | No ensure operation exists to repeat. Repeated refusals are idempotent (each gets its own op_id, no effect); restart covered in A13. |  |
| A17 | BLOCKED | L3 | Valid start/attach but no ingest grant — observation control disabled | observe -> 501 blocked_no_ingest_grant_contract; UI control disabled; Processor has no observation-sending code path; Engine (pid 1963) ran with no observation from Processor | BLOCKED because the precondition (a valid start/attach) cannot exist. Observed parts: Engine may run, observation control disabled and refused, zero observations sent by Processor |
| A25 | BLOCKED | L3 | Worker unavailable | Engine runs with no Chat-Jimmy worker configured; Processor cannot read Engine worker state without an approved attachment, so it reports nothing about the worker rather than inventing a state. |  |
| A11a | PASS | L3 | Processor side: occupied port, second instance on same profile, wrong service at Identity address, remote target | second Processor on busy port exit=1: "error: can't listen on 127.0.0.1:7860: listen tcp 127.0.0.1:7860: bind: address already in use — another program already uses this port. Choose a free one with -port (Processor will not stop the other program)."; second Processor on same profile exit=1: 'error: another Thought Processor is already using profile <tmp>/tp-live-kptgbfpp/profile-a (run run_80529921bd4754b8, pid 1911, http://127.0.0.1:7860). Open that one, or stop it first; this one will not take it over'; Identity address pointed at real Engine port -> ('PORT_OPEN_UNIDENTIFIED', 'NO_PUBLIC_METADATA', 404, 'wrong_identity'); remote address -> 400 Identity address: only this computer (127.0.0.1, ::1 or localhost) is allowed; remote targets are refused; original Processor alive=True | Covers only Processor's own checks; see A11b |
| A11b | BLOCKED | — | Runner side: missing executable, wrong digest/protocol, spoofed Engine at an approved endpoint | No runner and no approved manifest exist; executable/digest/protocol checks belong there. Processor never shows an authenticated badge from claimed metadata (A27-live-identity shows 'unverified'). |  |
| A18 | PASS | L3 | Cross-origin form/script, untrusted Host, missing/invalid session or CSRF, GET mutation | [('cross-origin form', 403), ('cross-origin JSON', 403), ('no Origin', 403), ('untrusted Host (rebinding)', 421), ('missing CSRF', 403), ('wrong CSRF', 403), ('no session', 401), ('GET state no session', 401), ('GET start', 405), ('GET shutdown', 405), ('CORS preflight', '405 ACAO=None')]; cross-origin browser fetch from about:blank -> blocked: TypeError; refused-operation counter (only counts requests that passed guards) = 9 |  |
| A19 | PASS | L3 + L2-live fixture | Path/argument injection, path traversal, non-http scheme; bytes sent to an Engine candidate | start with path/args/endpoint -> 400 parameters_not_accepted; engine '..' path -> 400; file:// -> 400; /tmp/pwned exists=False; raw listener as Engine candidate saw 1 connection(s) carrying [0] bytes (TCP connect only, no HTTP request) | No shell exists to inject into: os/exec is not in the binary's dependency graph (checked separately with go list -deps) |
| A20 | PASS | L3 (+ L2 canary fixture in Go tests) | Copy Context and log file with live secrets present | Copy Context 3453 bytes; leaked secrets: none; remote claim strings included: False; profile log leaks: none; sample saved as copy-context-sample.txt | Canary-in-remote-claim test ran against a labeled fixture server (TestIdentityProbeClassificationFixture); no private observations exist to leak because Processor never handles any |
| A22 | PASS | L3 | Original Engine UI inaccessible / not authorized | capability engine_original_ui=BLOCKED (only after an approved attachment); links on page=['https://github.com/Arameair/Thought-Processor/blob/experiment/processor-first-prototype-20261024/docs/contracts/IDENTITY_ENGINE_LIFECYCLE_APPROVAL_REQUEST.md']; no embedded or rebuilt Engine UI |  |
| DISC-01 | PASS | L3 | Probe an unused local port | 127.0.0.1:54919 -> UNREACHABLE (CLOSED); label 'Not reachable (closed)' |  |
| DISC-02 | PASS | L3 | Real unrelated HTTP programs placed at the Identity address | [('python http.server', 34329, 'PORT_OPEN_UNIDENTIFIED', 'NO_PUBLIC_METADATA', 404, None, 'wrong_identity'), ('another Thought Processor', 7861, 'ANSWERED_NOT_IDENTITY', 'RESPONDED', 200, 'thought-processor', 'wrong_identity')] — neither shown as Identity, authenticated or authorized |  |
| SAFE-03 | PASS | L3 + L2-live fixture | Identity address that accepts connections but never answers | save+check returned in 1504 ms with state=PORT_OPEN_UNIDENTIFIED metadata=FAILED; a concurrent page read answered 200 in 2 ms | The stalling listener is a test fixture |
| SAFE-05 | PASS | L3 | Profile path is a file, not a folder | exit 1: "error: can't create profile folder <tmp>/tp-live-kptgbfpp/not-a-folder: mkdir <tmp>/tp-live-kptgbfpp/not-a-folder: not a directory"; file unchanged=True; no port opened=True |  |
| UX-05 | PASS | L3 | Entity, feature and instance shown separately | entity='No entity selected'; feature='Thought Engine at http://127.0.0.1:7798'; instance='Something is listening at http://127.0.0.1:7798 — not identified' |  |
| FLOW-07 | PASS | L3 | Missing Identity/Engine steps shown as missing, not reconstructed | 10 traces; newest op_72e7edf6171a11d3 start: Requested=DONE, Authorized=NOT ATTEMPTED, Executed=NOT ATTEMPTED, Observed=NONE |  |
| METRICS | PASS | L3 | Processor's own metrics are real counters | refused={'attach': 2, 'observe': 1, 'start': 5, 'stop': 2}; rejected={'csrf': 2, 'no_session': 6, 'untrusted_host': 1, 'untrusted_origin': 3}; identity checks=15 outcomes={'CLOSED': 3, 'OPEN/FAILED': 1, 'OPEN/NO_PUBLIC_METADATA': 2, 'OPEN/RESPONDED': 9} latency={'max_ms': 1500, 'median_ms': 0, 'min_ms': 0, 'samples': 15}; not measured: ['Engine queue/worker/throughput (needs approved attachment)', 'Identity decisions (no decision service exists)', 'worker/model calls (Processor never makes any)'] |  |
| A21 | PASS | L3 | Service disappears after a successful check; no re-check for 21 s | Engine before=PORT_OPEN_UNIDENTIFIED checked 2026-10-24T09:06:56.057855837Z; test script stopped Engine pid 1963 (SIGTERM, 3 ms, exit 0); next re-check -> UNREACHABLE (CLOSED) within 4 ms of the re-check request; after 21 s without a check Identity shows state=STALE freshness=STALE | Detection is on the page's 10 s re-check cadence, not instant |
| A16 | PASS | L3 | No stop approval/control policy: request Stop twice | stop -> [(501, 'BLOCKED', 'op_8de419068723a2ff'), (501, 'BLOCKED', 'op_09cfd79a1f00b4d4')]; external Engine pid 2018 alive=True; Identity alive=True; Processor alive=True |  |
| A15 | BLOCKED | L3 | Stop session-owned Engine, repeat, then external Engine | No session-owned Engine can exist (no runner). Stop requests were refused and the external Engine, Identity and Processor survived (A16 evidence). |  |
| A12 | BLOCKED | — | Slow startup, exit or partial failure of an Engine start | No runner; nothing is started, so there is no startup to time out or children to clean up. |  |
| A14 | BLOCKED | — | Stale ownership ledger / reused PID / uncertain child | Processor keeps no ownership ledger because it owns no processes; it never kills by PID or port. Its own run record is used only to report an unclean previous run (see OPS crash test). |  |
| A26 | BLOCKED | — | Authority revoked after attach | No attach can be granted, so there is nothing to revoke. |  |
| A23 | NOT RUN | — | Repo creation succeeded / local setup failed (later onboarding) | Out of scope: GitHub bootstrap is undecided and not implemented. |  |
| A24 | NOT RUN | — | GitHub account differs from administrator (later onboarding) | Out of scope: GitHub/admin enrollment is undecided and not implemented. |  |
| UX-02 | PASS | L3 | Remembered Identity address is gone after a restart | before first check: NOT_CHECKED ('Not checked yet'); after check: UNREACHABLE — never shown as live from the saved record |  |
| A13 | PASS | L3 | Close/reopen browser; stop and restart Processor backend on the same profile | new browser context -> sign-in screen; reopened tab in same browser -> still signed in; Stop Processor -> 202, exit 0 in 116 ms; leftover Processor processes=[]; run.json removed=True; Identity alive=True Engine alive=True; restart ready in 22 ms as pid 2048, run run_2c0dccbed46df63f (was run_80529921bd4754b8); old session cookie -> 401; new operator code differs=True; saved Identity address kept=http://127.0.0.1:7796 and re-checked at 2026-10-24T09:07:28.080588093Z | No survival is claimed for anything Processor would own — it owns nothing |
| OPS-04 | PASS | L3 | Processor killed with SIGKILL, then restarted on same profile | restart ready in 22 ms (pid 2055); previous_unclean_run={'meaning': 'that run did not shut down cleanly; Processor started fresh and took no action because of it', 'pid': 2048, 'run_id': 'run_2c0dccbed46df63f', 'started_at': '2026-10-24T09:07:27.972674154Z'}; lock reacquired (kernel released flock); no action taken against any other process |  |
| SAFE-07 | PASS | L3 | 3 clean start → inspect → stop cycles (Processor only) | [{"pid": 2062, "ready_ms": 22, "self_reported_startup_ms": 1, "stop_ms": 3, "exit": 0, "leftover_children": []}, {"pid": 2070, "ready_ms": 22, "self_reported_startup_ms": 2, "stop_ms": 3, "exit": 0, "leftover_children": []}, {"pid": 2077, "ready_ms": 23, "self_reported_startup_ms": 2, "stop_ms": 3, "exit": 0, "leftover_children": []}] |  |
