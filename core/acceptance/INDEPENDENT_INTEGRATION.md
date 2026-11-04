# Independent integration and acceptance: Go TASS (2026-11-04)

**Test-only branch:** `experiment/independent-integrity-gates-20261104`, based on Docker PR #48 at `5188c26a71ff0012bea9cfa1f6d9e9ed2741440e`. No Docker, deployment, Identity or production Go code changes. Engine source inspected at `4924de6f1b8504abcab4fed98c0e191057f4e277`. Identity PR #31 remains OPEN; this engineer did not grant permissions.

## Interface and dependency inventory (actual code, not intended future features)

| Caller → component | Interface / data | Effective gate | Response and persistence |
| --- | --- | --- | --- |
| Browser → Go Core, `127.0.0.1:8795` | `GET /` | local Host check only | 200 HTML embeds page token (usable for API); **not actual owner authentication** |
| Browser/CLI → Core | `GET /api/health` | local Host check | 200 JSON `state` READY/WORKING/BUSY/BLOCKED, `policy_valid_now`, `model_ok`, `version`; no privileged data |
| Browser/CLI → Core | `GET /api/status` | `X-TASS-Token` or `Authorization: Bearer` = page token or local API key; local Host | 200 full status, recent cycles, events; 401 invalid key |
| Browser/CLI → Core | `POST /api/say` JSON `{"text":"..."}` | same token, local Host, JSON Content-Type; Origin checked when supplied | 202 with no receipt ID; **65th queued message can be silently dropped** (TP #46). HTTP 202 is not proof of inference/reply |
| Core → Engine (shares network namespace in Docker) | `POST http://127.0.0.1:7798/api/cycles` `CycleRequest` | Engine bearer key; prototype `thought.process` and `model.request` grants | 200 `Cycle` with `cycle_id`, Spark, proposal, locked prediction; 403 denied / 503 model failure; `cycles.jsonl` |
| Core → Engine | `POST /api/cycles/{id}/outcome` `CycleOutcome` | Engine bearer key | 200 evaluated cycle, 409 duplicate outcome; `cycles.jsonl` append-only + file sync |
| Core → Engine | `GET /api/cycles`, `GET /api/cycles/{id}` | Engine bearer key | cycle records, replayed by Engine at restart |
| Core → Engine | `GET /api/model` | **no bearer**, local Host | model health pulse/busy/recent request IDs; 200 |
| Engine → Nemotron | `GET http://model:8081/health` in Compose; host mirror `127.0.0.1:18083/health` | network isolation, not Identity | 200 health only; NO inference proof |
| Engine → Nemotron | `POST http://model:8081/v1/chat/completions` | prototype grant checked before and again after model-slot wait | OpenAI-compatible `choices[0].message.content` / `reasoning_content`; result in `model-requests.jsonl`. Model weights identity must be verified on owner GPU |
| GitHub → Core policy | `Arameair/identity@main:policies/tass.json` | checked during sync; local cache max 24 hours by default | `~/.tass/grant/identity-grant.json`; missing/expired policy blocks governed work; **static prototype, NOT live Identity authorize** |
| GitHub ↔ Core TASS | `Arameair/TASS@live:tass.md`; `tass` issues/comments | policy allow flags, obey list; `TASS_GITHUB_MESSAGES=off` while Python liaison is active | local `~/.tass` journals and conflict snapshots; GitHub sync/backoff independent of inference |
| Engine legacy mutation/read | `POST /api/identities`, `/api/entities/.../grant`, `/api/settings`, worker controls; `GET /api/model/requests/{id}` | legacy Host / Origin / content-type **only**, no Engine bearer for some routes | sensitive state and model text at risk if network isolation fails (Thought-engine #6) |

### Docker trust boundary

Core and Engine are two containers but **one deployed stack**. Core has `network_mode: service:engine`; Engine listens on localhost **inside their shared namespace**; only Core's `8795` is published to the host localhost. Both reuse the existing `model:8081` on external `tass-workspace_default` network. Docker image and config checks do not prove that live process remains isolated or that no second GPU model is running. Never publish the Engine's legacy mutation ports externally or treat Compose DNS/loopback headers as authorization. See TP #44 and Claude's Docker PR #48.

### Real cycle evidence contract

`/api/say` submission → Core incoming channel → `tp_...` cycle request → Engine observation Event → Spark (Trace/Thought **conditional**, not guaranteed for one message) → local Nemotron plan → **Engine fsyncs proposal containing immutable prediction before returning** → Core re-checks `action.read_tass` at point of use → local read-only `tass.md` measurement or explicit nonexecution → Engine outcome → deterministic evaluation → Core conversation journal and owner-visible reply.

The Engine implementation appends `{at,cycle_id,step,data}` to `~/.thought-engine/tass/cycles.jsonl` and `f.Sync()` occurs before `StartCycle` returns. An EXECUTED action must require a matching `cycle_id`, `Prediction.State=LOCKED`, valid SHA-256/locked timestamp, an authorization receipt and a measured result. **Go Core does not currently validate all of these before execution.** A model refusal or malformed proposal must not be reported as action success. A real model health ping proves only reachability.

### Pulse distinction and states

**Present:** Engine ModelLink.Pulse immediately and at `20s` intervals queries `GET /health`, writes `pulse.jsonl` with `at, model_ok, requests_total, failures_total, busy`; GitHub sync separately polls/backoffs. This pulse has no `pulse_id`, Start/Pause/Resume/Step Once or thought-dispatch semantics.

**NOT IMPLEMENTED:** coordinated thought pulse with ten 20-second ticks, unique IDs, Pause/Resume/Step Once, no duplicate slow inference/action, and durable restart recovery. Do not relabel ten health samples as a passing thought pulse. READY = valid grant + idle + healthy; WORKING = Core cycle in progress; BUSY = Engine slot in use; BLOCKED = invalid grant/down dependency; UNKNOWN = Core not answering. `/api/health` reports these as current observations, not a complete thought-controller state machine.

## Automated verification and known red gates

```bash
# In the test branch checkout; no service stop/start or GitHub write:
cd core
go vet ./...
go test -race ./...
# One expected green contract: grant revoked while model reply is held:
go test -race -tags acceptance ./tass -run TestAcceptanceRechecksRevokedActionGrantAfterModelWait -count=1 -v
# STRICT red acceptance gates; nonzero exit means the baseline violates the contract:
go test -race -tags acceptance ./tass -run 'TestAcceptanceRejects(ActionWithoutLockedPrediction|EngineCycleIDMismatch)' -count=1 -v
```

The acceptance fixture uses a real Go Core instance and local **scripted Engine/GitHub HTTP servers**, not real Thought Engine or Nemotron. The Engine #7 PR independently tests real Go Engine code with a scripted llama.cpp fixture; Core PR #45 has local admission/status tests and owner read-only evidence probes. These should be reviewed together; they are independent **fixture** evidence, not a cross-process GPU pass. This branch deliberately leaves failing behavioral gates red rather than adjusting expectations to current unsafe behavior.

| Requirement | Result as of source inspection | Evidence / next gate |
| --- | --- | --- |
| Core builds, existing unit/fixture tests | PASS if CI green | `go test -race ./...` |
| Owner conversation while Python Full Auto/Autonomy active | NOT RUN on owner stack | TP #41 / PR #43; Go Core does not yet implement those switches |
| Owner chat admitted while Go Core inference busy | PARTIAL / FAIL at capacity | Core buffer permits up to 64; `Say` silently drops thereafter, TP #46 |
| Core point-of-use re-authorization after model delay | fixture PASS when test green | `TestAcceptanceRechecksRevokedActionGrantAfterModelWait` |
| Prediction locked before any action | **FAIL: Core accepts unlocked Engine proposal** | `TestAcceptanceRejectsActionWithoutLockedPrediction` |
| Cross-service cycle correlation before action | **FAIL: Core accepts mismatched Engine cycle ID** | `TestAcceptanceRejectsEngineCycleIDMismatch` |
| Engine malformed model, expired policy, model unavailable | fixture PASS in Engine PR #7 (subject to its CI) | Scripted fake llama.cpp, not GPU |
| Ten actual coordinated 20s thought ticks, pause/resume/step/restart | NOT RUN — NOT IMPLEMENTED | Do not credit health pulse |
| Prototype Identity grants, revocation latency | PARTIAL | Identity PR #31 OPEN; live signed authorize NOT IMPLEMENTED |
| Real Nemotron inference → Spark/Trace/Thought → outcome → evaluation | NOT RUN | Owner local test below |
| Docker update/failure rollback and data persistence | prior sandbox fixture evidence only | TP #44 / PR #48; owner deployment NOT RUN |
| Local API and Engine sensitivity boundaries | FAIL / review required | TP #47, Thought-engine #6 and this branch |

## Security findings / defect references

- **P0 integrity defect:** Core `Answer` checks `prop.Action != "NONE"` and `action.read_tass` but not `prop.Prediction.State == "LOCKED"` and not `cy.CycleID == requestedID` before `measure()`. A 200 Engine response can be uncorrelated or unlocked yet produce an EXECUTED receipt. This branch adds **failing executable acceptance gates**; no production code changed.
- **P1 local owner-auth bypass:** local GET / yields browser token accepted by /api/say and /api/status (TP #47).
- **P1 message loss:** at 65th unsent message, Core acknowledges an enqueue that never occurred (TP #46).
- **P1 Engine legacy unauthenticated mutation and sensitive model-result GET; optional policy path fail-open:** Thought-engine #6. Compose isolation mitigates reachability, not authentication.
- **P1 policy:** Identity PR #31 would enable GitHub write flags alongside local thought grants. Do not merge as a test workaround. Cached prototype policy has bounded offline lifetime and sync-based revocation delay.
- **P2 visibility:** Engine replays cycles after restart; Core's recent cycles/status are in-memory, so persistent record does not prove visible recovery (TP #45 inventory).

## Minimal owner acceptance (NO destructive commands)

1. Review TP PR #48 and this test branch; don't merge or enable new permissions for CI. On your WSL machine, use `cd ~/tass-workspace/core && ./tass.sh check && ./tass.sh status`. These inspect without changing the existing Python stack. Confirm *one* model container, Engine/Core SHAs, grant state, ports and volume paths.
2. **Only after the owner approves an appropriately restricted Identity grant**, start the new stack using the procedure in `core/DOCKER.md`. Refrain from activating GitHub-writing permissions merely to make chat green.
3. At `http://127.0.0.1:8795`, send **How many sections does your tass.md have?** Wait for an actual TASS reply and record the `tp_...` cycle ID. Verify that `~/.thought-engine/tass/cycles.jsonl` contains, in order, received → Spark → proposal with LOCKED prediction and SHA → EXECUTED/NOT_AUTHORIZED → evaluation. Verify `model-requests.jsonl` has the matching plan request ID and `~/.tass/conversation.jsonl` has the matching reply. Do not paste raw conversation or API keys into GitHub.
4. Test offline GitHub and a restart **only after backing up state** and with owner approval. Recheck that the same cycle is in the Engine journal after restart; Core UI recovery remains an open gap. Run the coordinated thought-pulse gate only when the controller actually exists; today it is NOT RUN.
5. Paste sanitized cycle IDs, step names/timestamps, state, grant decision, model identity evidence, and red/green test output into the PR/issue. Mark **real-Nemotron end-to-end as NOT RUN** until you personally execute it.

**Source links:** TP #41, #42, #44, #46, #47; Thought-engine #6, #7; Identity PR #31; independent Core PR #45; Docker PR #48.

**End state:** TEST SUITE ADDED, but integration acceptance BLOCKED by prediction/correlation defects, unapproved policy and unrun owner GPU verification.
