# Independent acceptance: Go TASS (do not touch Docker or Identity)

This is a test harness for the Go Thought-Processor/Thought Engine at the source revisions in INTERFACES.md. It is NOT an implementation of the future thought-pulse controller. This engineer did not run real Nemotron on the owner's GPU. Test fixtures are clearly labelled.

## CI and fixture checks

From a checkout of the independent test branch:

    cd core
    go test -race ./...
    cd ..
    python3 -m unittest discover -s core/acceptance -p 'test_verify.py' -v

The Go busy-intake test runs the actual Go Core worker/sync loop and HTTP adapter with a fake Engine and fake GitHub. The Python tests use generated JSON evidence fixtures to prove the verifier rejects missing, reordered or fabricated records. Neither is a real-model acceptance.

## Read-only owner preflight (safe with existing Python stack)

Keep running services untouched and do NOT merge Identity PR #31 merely to make a check green. Do NOT use a Docker start/update/rollback command for this test. The baseline is port 8795 Core, 7798 Engine, 18083 llama.cpp as previously reported; override CLI URLs if different.

    cd ~/tass-workspace/core
    ./start.sh check
    ./start.sh status
    python3 acceptance/verify.py baseline
    python3 acceptance/verify.py security

The security command issues GET requests only (including deliberate unauthenticated GET), never POST against legacy Engine routes. It does not disclose API keys or conversation text. An unapproved Identity prototype will show BLOCKED even if Nemotron health is good.

To collect ten ACTUAL 20-second health samples, leave Engine running for more than three minutes, then run:

    python3 acceptance/verify.py health-pulses --count 10

This proves ONLY timing of model HEALTH checks. A PASS here does NOT prove ten coordinated thought pulses. It requires no GPU inference.

## Explicitly authorized owner test (only after policy review)

Only after you choose an appropriate narrowed Identity authorization and the real local model is already running, type one harmless request in the Go UI at http://127.0.0.1:8795:

    How many sections does your tass.md have? Give the actual count.

When the UI displays a completed cycle, copy its tp_* ID and run (example placeholder, replace exactly):

    python3 acceptance/verify.py cycle tp_REPLACE_WITH_REAL_CYCLE_ID

The verifier reads ~/.thought-engine/tass/cycles.jsonl (append-only) and calls the Engine API to check the same ID. It requires the Spark/Trace, model plan, locked SHA-256 prediction, authorized EXECUTED read-only action, timestamps in order, and scored evaluation. A simple hello with NO_ACTION cannot pass the executable-action gate. A model reply with format UNREADABLE also cannot pass. If the Spark is too weak to create a Trace, try another appropriate owner-approved request and report it honestly.

To test chat intake with one new, identifiable harmless message:

    python3 acceptance/verify.py chat --send

While a different long-running local Go cycle is actively WORKING (start it manually), run:

    python3 acceptance/verify.py chat --send --require-busy

The probe issues one POST /api/say and polls status for a matching completed cycle. HTTP 202 alone is NOT PASS; the current Go API does not return a per-message ID, so correlation uses the unique message text and final cycle ID, and the verifier reports the Go Full Auto/Autonomy gap. These tests can propose the existing read-only action (subject to policy), so they are opt-in. No writes to GitHub or permissions are performed.

Inspect the local model catalogue as corroboration: baseline queries /v1/models at localhost:18083. Require the actual model name/runtime evidence and at least one cycle with Engine model-request records before declaring REAL NEMOTRON PASS. A successful scripted llama fixture is never real-GPU evidence.

## Coordinated thought pulse: NOT RUN until implemented

The future normalized pulse evidence schema proposed by the verifier is one JSON record per tick:

    {"kind":"thought_tick","pulse_id":"stable-id","at":"2026-11-04T15:00:00Z","mode":"RUNNING","state":"WORKING","active_request_id":"tp_123","model_started_ids":[],"action_executed_ids":[]}

Separate control receipts use kind=control, command=pause|step_once|resume. A manual step is a thought_tick with mode=PAUSED, reason=STEP_ONCE. Restart evidence uses kind=process_start and an at timestamp. Once the implementer provides an agreed actual journal (not pulse.jsonl), run:

    python3 acceptance/verify.py thought-pulses /path/to/ACTUAL_THOUGHT_PULSE.jsonl

The probe checks 10 automatic 20-second ticks, unique IDs, pause/step/resume evidence, a model request persisting through at least three ticks with exactly one start, non-duplicate action IDs, explicit statuses and >=2 process starts. The acceptance standard also requires a verified offline GitHub outage during continued local pulses. The current source has no such controller, so do NOT fabricate a file to claim acceptance.

## Acceptance matrix (source observations vs executable results)

| Gate | Source review | Fixture/CI | Owner real-PC |
|---|---|---|---|
| Go API token and Origin checks | Implemented, but page token is obtainable via local GET | GO tests (see CI) | NOT RUN |
| Go chat intake during WORKING | queue + serialized worker; 202 has no durable receipt | GO fixture test (see CI) | NOT RUN |
| Spark/Trace/Thought and lock-before-execute ledger | implemented one-Spark path, deterministic evaluation | Existing Engine and Core fixture tests; independent journal verifier fixtures | NOT RUN |
| Denied, expired or missing prototype grant | code checks these for configured file | Existing fixture tests | NOT RUN |
| Malformed model output, offline/unavailable model | parser marks UNREADABLE; model errors recorded | Existing Engine fixtures; verifier rejects unreadable action as successful cycle | NOT RUN |
| Real Nemotron GPU identity, response, trace and evaluation | HTTP path exists | NOT RUN (fixtures are not GPU proof) | NOT RUN |
| Ten 20-second MODEL HEALTH checks | implemented | source/fixture timing test | NOT RUN |
| Ten unique COORDINATED THOUGHT pulse IDs + controls | not implemented | future evidence-contract validator, fixture only | NOT RUN |
| Full Auto+Autonomy chat in Python Atrium | issue #41 fix proposed PR #43 | PR #43 scripted fixtures | NOT RUN |
| Go autonomy while owner chats | no Go Full Auto/Autonomy API | only ordinary WORKING intake fixture | NOT RUN |
| GitHub offline sync/backoff, restart, conflict | source and fixtures | Existing Core tests | NOT RUN |
| Docker deployment, Update/Rollback | exclusively Claude issue #44 | OUT OF SCOPE | NOT RUN |

A failed acceptance gate is FAIL, not a green service status. A missing feature is NOT IMPLEMENTED; do not turn it into PASS because a fixture matched a future schema. For denied policy on current machine, record BLOCKED rather than change permissions ad hoc.

## Required owner evidence

Submit: current branch/commit; redacted ./start.sh check/status output; /v1/models model ID or model runtime log with weights; one actual tp_* cycle ID; Core UI cycle, spark and action state; Engine cycle journal lines for that ID (without secrets); the verifier output; ten health pulse timestamps; whether Full Auto/Autonomy was actually ON in the relevant UI; and a separate report of any future controller ticks. Avoid publishing raw user messages, model reasoning, API keys, GitHub tokens, or private tass.md sections.

End state: engineering harness may be DONE while real-PC acceptance remains BLOCKED / NOT RUN.
