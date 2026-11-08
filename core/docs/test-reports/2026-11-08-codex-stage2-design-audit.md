# Independent TASS Stage 2 Design Audit

**Date:** 2026-11-08

**Disposition:** **FAIL — REVISE BEFORE IMPLEMENTATION**

**Scope:** Source and specification review only. No implementation, live processing, permission changes, service restart, deployment, or owner-data mutation was performed.

## Reviewed baseline

| Component | Exact source reviewed |
|---|---|
| Core | `094f43ba2bc167ace8835bf286f37951fee432c9` |
| Thought Engine | `d69d3917d1601d806d4b2ba2254693e1adb92b6b` |
| Identity | `64fdf8e8cf926df5ea6bb6c23fbab864d0f09e3b` |
| Proposed design | PR #101 head `033b79782375831cb5b161f13ad2e304d3787f8f` |

Read in the requested order: [Issue #100](https://github.com/Arameair/Thought-Processor/issues/100), [PR #101](https://github.com/Arameair/Thought-Processor/pull/101), [latest GPT-6 review](https://github.com/Arameair/Thought-Processor/pull/101#issuecomment-6725506555), then the exact pinned sources above.

## Executive finding

PR #101 has the right product boundary—manual, supervised, one nudge, no pulse or autonomy—but its durable run-once and authorization proofs are not yet implementable safely. The proposed global `cognition.nudge.process` grant plus the shared Engine bearer key does not prove that a particular request came from a particular owner-approved nudge. Its claim journal also cannot distinguish “no claim” from a torn or partly durable claim after an `fsync` error. Current cross-process locking can be defeated by container PID namespaces.

The smallest safe design should use a bound, single-nudge authorization artifact; an atomic cross-process claim primitive; explicit uncertain/recovery states; point-of-use checks immediately before the HTTP model attempt and immediately before saving the Thought; and completion verification from both the durable run record and the exact Thought mutation.

## Findings

### P0 findings

| ID | Result | Finding and evidence | Required correction |
|---|---|---|---|
| A-01 | **FAIL** | Per-run consent is not proven at the Engine. Identity authorizes only an action and optional entity; `detail` is audit text, not a policy constraint ([Identity `Authorize`](https://github.com/Arameair/Identity/blob/64fdf8e8cf926df5ea6bb6c23fbab864d0f09e3b/internal/entity/entity.go#L355-L377)). Engine authorization sends only action/detail ([Engine `askIdentity`](https://github.com/Arameair/Thought-Processor/blob/d69d3917d1601d806d4b2ba2254693e1adb92b6b/core/model.go#L181-L206)). The one shared write key admits all sensitive routes ([server wrapper](https://github.com/Arameair/Thought-Processor/blob/d69d3917d1601d806d4b2ba2254693e1adb92b6b/core/server.go#L426-L475)). | Require an Engine-verifiable, single-use capability bound to owner, entity, Engine, Thought, nudge ID, body digest, action, and expiry; or explicitly trust Core and give only Core a dedicated route credential plus a cryptographically bound receipt. A generic Engine key must not be sufficient. Validate authorization before returning a POST replay. |
| A-02 | **FAIL** | The proposed E2 duplicate lookup precedes E3 authorization. A caller can replay a known nudge after revocation and receive a successful run response without a fresh process authorization. | Check the bound capability/current grant before any successful POST response, including idempotent replay. Keep read-only status on the separately authorized GET route. |
| A-03 | **FAIL** | “`fsync` failed, therefore no claim” is false. Append bytes may be visible, partially durable, or torn. The existing append helpers do not provide a safe claim boundary: Engine logging ignores write durability ([`appendLog`](https://github.com/Arameair/Thought-Processor/blob/d69d3917d1601d806d4b2ba2254693e1adb92b6b/core/model.go#L256-L268)); Core's generic logger has the same class of weakness. | Use an immutable per-nudge claim file created with `O_CREATE|O_EXCL`, then file `fsync`, close, and parent-directory `fsync`, or an equivalent transactional store with a uniqueness constraint. Any ambiguous write becomes `RECOVERY_REQUIRED`; reserve the ID in-process and refuse new claims until recovery. Never rotate/delete the claim ledger automatically. |
| A-04 | **FAIL** | The current process lock is unsafe across PID namespaces. It removes the lock when the stored PID equals its own PID ([`acquireFileLock`](https://github.com/Arameair/Thought-Processor/blob/d69d3917d1601d806d4b2ba2254693e1adb92b6b/core/lock.go#L18-L42)); two containers can both be PID 1. Unix liveness checks only the current PID namespace ([`processAlive`](https://github.com/Arameair/Thought-Processor/blob/d69d3917d1601d806d4b2ba2254693e1adb92b6b/core/lock_unix.go#L7-L10)). Two Engines could therefore both pass the in-memory `e.mu`/running-set guard. | Use an OS advisory lock on the shared filesystem and make the claim itself cross-process atomic. Add a two-PID-namespace acceptance test. Do not rely on PID equality/liveness for exclusivity. |
| A-05 | **FAIL** | Revocation/state checks are at the wrong times. `ModelLink.run` rechecks only the action passed to it—currently `thought.process`—after acquiring the model slot ([`run`](https://github.com/Arameair/Thought-Processor/blob/d69d3917d1601d806d4b2ba2254693e1adb92b6b/core/model.go#L333-L390)). `Think` checks Thought grant/resolution before the call, but after the call checks only that entity and Thought still exist before saving ([`Think`](https://github.com/Arameair/Thought-Processor/blob/d69d3917d1601d806d4b2ba2254693e1adb92b6b/core/engine.go#L1331-L1421)). Owner actions can change the Thought while the model call is running. | Revalidate the per-run process capability after acquiring the model slot and immediately before the HTTP POST. Before saving, recheck the process grant, Thought Processing switch, entity/Thought binding, unresolved state, and allowed Thought state. Record which revision was checked. Revocation after a committed save cannot retroactively undo it; state this explicitly. |
| A-06 | **FAIL** | A generated request ID or `model-requests.jsonl` entry is not proof of a Nemotron call. The Engine records denied/error results too ([`record`](https://github.com/Arameair/Thought-Processor/blob/d69d3917d1601d806d4b2ba2254693e1adb92b6b/core/model.go#L392-L404)); the actual HTTP attempt occurs only in `ask` ([HTTP POST](https://github.com/Arameair/Thought-Processor/blob/d69d3917d1601d806d4b2ba2254693e1adb92b6b/core/model.go#L406-L430)). A crash mid-request cannot prove whether the server executed it. | Persist explicit phases: `NOT_STARTED`, `HTTP_ATTEMPTED_UNCERTAIN`, `RESPONSE_RECEIVED`, and terminal outcome. Tests must count requests at a fixture HTTP server. UI/reporting must say “unknown” after an ambiguous mid-call crash, not zero or one. |
| A-07 | **FAIL** | The proposed recovery path can show false `COMPLETED`. Existing atomic JSON writes `fsync` the file but not the parent directory and may return an error after rename/readback ([`writeJSONAtomic`](https://github.com/Arameair/Thought-Processor/blob/d69d3917d1601d806d4b2ba2254693e1adb92b6b/core/store.go#L11-L52)). A run outcome and Thought can therefore conflict. Restoring a pre-claim snapshot can erase deduplication and permit a second call. | Add `INCONSISTENT/RECOVERY_REQUIRED`. `COMPLETED` requires one matching Note and one matching developed step with the same nudge/request IDs, a compatible terminal run outcome, and no conflicting evidence. After journal loss/restore, disable processing until an integrity audit; preserve the claim ledger outside ordinary data restore or bind it to a monotonic restore epoch. |

### P1 and open findings

| ID | Result | Finding | Required correction |
|---|---|---|---|
| A-08 | **FAIL** | The design says attention is in `model-requests.jsonl`, but the actual prompt is written to `calls.jsonl`: the worker places the attention in model messages ([worker call](https://github.com/Arameair/Thought-Processor/blob/d69d3917d1601d806d4b2ba2254693e1adb92b6b/core/worker.go#L456-L474)) and persists the trimmed `Call` prompt ([call journal](https://github.com/Arameair/Thought-Processor/blob/d69d3917d1601d806d4b2ba2254693e1adb92b6b/core/worker.go#L385-L395)). | Correct the data map and S15. Specify local prompt retention/redaction, permissions, backup exposure, and journal inspection for `calls.jsonl`. |
| A-09 | **OPEN** | Core background dispatch is not fully specified. Existing detached goroutine patterns are not all tracked by Core's shutdown waitgroup. A 10-minute Engine client timeout makes shutdown/cancellation observable. | Track the dispatch worker explicitly, define context ownership and graceful shutdown, and make `202` mean only “accepted locally; delivery unconfirmed.” Startup and GET must remain read-only. |
| A-10 | **FAIL** | F3 treats Engine GET 404 as evidence that retry/copy is safe. A delayed original POST can still claim after that observation. Copy-as-new then creates a second nudge ID and can produce a second model call/development. | Treat 404 as nonterminal “not observed as claimed.” Forbid retry/copy while delivery may still start, or add a durable delivery fence with expiry/cancellation that the Engine enforces. |
| A-11 | **OPEN** | Relinking during a run is underspecified. Engine cannot observe Core's local relink. | Choose and test one rule: an already-dispatched run remains bound to the original entity and may finish, or Core prohibits relink while a run is nonterminal. Do not imply relink cancels an in-flight model call. |
| A-12 | **OPEN** | Claim storage is capped but restoration/backup limits are not operationally defined. Disk-full, read-only, torn-tail, and missing-ledger cases need fail-closed behavior. | Refuse new processing when claim durability is unavailable. Document backup consistency, restore ordering, integrity scan, and the limitation that run-once cannot survive loss of every durable claim copy. |

## S1–S15 review

| Test | Result | Audit note |
|---|---|---|
| S1 | PASS | Sound as an isolated Identity action test; also assert it is absent from unrelated discovery surfaces. |
| S2 | FAIL | Global grant/shared key does not establish per-nudge owner consent; POST replay is checked before authorization. |
| S3 | OPEN | Completion must verify Note + developed step + compatible durable outcome, not merely a step/request ID. |
| S4 | OPEN | Add Core-side synchronization and cross-process Engine claim tests. |
| S5 | FAIL | Add revoked duplicate replay and capability substitution tests. |
| S6 | OPEN | Add a real delayed-handler/copy-as-new race, not only an injected 404. |
| S7 | OPEN | Specify and test worker lifetime, cancellation, shutdown, and truthful `202`. |
| S8 | FAIL | Request journal/ID is not actual-call proof; current process lock is not namespace-safe. |
| S9 | FAIL | Conflicting outcome/Thought evidence needs `INCONSISTENT`, not recovered `COMPLETED`. |
| S10 | FAIL | The proposed fsync-failure assumption permits duplicate execution. |
| S11 | PASS | Valid if the fixture server's received POST count—not journal count—is the oracle. |
| S12 | FAIL | Existing point-of-use authorization rechecks `thought.process`, not the new per-run process grant/capability; post-model Thought state is not rechecked. |
| S13 | OPEN | Add owner resolve/switch/grant races before HTTP and before save. |
| S14 | OPEN | Add two Engine processes in distinct PID namespaces and restore-epoch tests. |
| S15 | FAIL | It omits `calls.jsonl`, where the prompt is actually persisted. |

Owner test O1 remains **OPEN** and must not run until all P0 corrections are accepted and isolated tests pass.

## F1–F14 review

| Failure | Result | Audit note |
|---|---|---|
| F1 | PASS | Safe if failure precedes all durable run writes. |
| F2 | OPEN | Ambiguous Core SUBMITTED durability must block retry and enter recovery. |
| F3 | FAIL | GET 404 is a time-bounded observation, not proof the delayed POST cannot later claim. |
| F4 | PASS | Same durable claim should return the same result after an authorized replay. |
| F5 | FAIL | Torn/ambiguous claim writes cannot be treated as no claim. |
| F6 | FAIL | Pre-generated ID/journal line does not prove an HTTP model attempt. |
| F7 | OPEN | Connection loss can mean unknown delivery; only the fixture/server can count receipt. |
| F8 | OPEN | Save can succeed at rename yet report failure later; conflicting evidence needs recovery. |
| F9 | FAIL | Current completion rule does not require matching Note plus non-conflicting outcome. |
| F10 | OPEN | Background worker shutdown and cancellation semantics are unspecified. |
| F11 | OPEN | Requires Core synchronization plus filesystem-enforced Engine uniqueness. |
| F12 | FAIL | Authorization ordering and per-run binding are insufficient. |
| F13 | FAIL | Required state/grant rechecks are absent at both point-of-use boundaries. |
| F14 | OPEN | Mid-flight relink policy is undefined and cannot cancel the Engine by itself. |

## D1–D8 recommendations

| Decision | Recommendation |
|---|---|
| D1 | Accept “discard if still possible”: check immediately before HTTP and before save. State that revocation after committed save cannot undo the result. |
| D2 | Permit supervised processing with owner Autonomous off, but never when forced `TASS_CHAT_ONLY=on`. |
| D3 | Keep one attempt per nudge; prohibit Copy-as-new while the original is uncertain or nonterminal. |
| D4 | Do not require a frozen Thought revision, but record confirmation, dispatch, pre-call, and pre-save revisions; fail if resolved or disallowed. |
| D5 | Require both Core and Engine authorization, with a bound per-run artifact. A global grant alone is insufficient. |
| D6 | The confirmation dialog is sufficient only if Core has a dedicated authenticated path and presents an Engine-verifiable bound receipt; otherwise require a one-shot Identity capability. |
| D7 | Use bounded, non-rotating claim storage that refuses new runs when full; provide owner-controlled archive/export and integrity procedures. |
| D8 | Preflight only the exact Thought/entity, Processing switch, heartbeat/work-pulse state, model readiness, and required permissions. Do not silently assume an `identity-test` Thought name. |

## Additional required tests

| ID | Setup and action | Required oracle / failure condition |
|---|---|---|
| X1 Bound-capability substitution | Mint/construct approval for nudge A; submit nudge B, altered digest, other Thought, other entity, expired capability, and replay after terminal result. | Every substitution fails before claim/model work; terminal replay is allowed only for the exact authorized tuple. |
| X2 Revoked duplicate replay | Complete/claim a run, revoke process permission, POST the same nudge again. | POST is denied; authorized GET may still return status. No new call. |
| X3 Torn claim append | Inject short write, `fsync` error, close error, directory-fsync error, and restart after each boundary. | Never retry automatically; state is `RECOVERY_REQUIRED`; fixture Nemotron count is at most one. |
| X4 Two PID namespaces | Start two Engine processes sharing the data directory, each PID 1; race the same nudge 100+ times. | Exactly one atomic claim and at most one fixture-server POST. Any second claim is a release blocker. |
| X5 Model-slot revocation | Hold model slot, submit nudge, revoke process capability before releasing slot. | Zero fixture-server POSTs; terminal discarded/denied state. |
| X6 Pre-save state matrix | During a blocked model response, independently revoke process permission, switch Thought Processing off, resolve, change state, remove entity, and relink Core. | No Thought save for every disallowed Engine-visible change; precise terminal state; relink follows the selected D/F14 rule. |
| X7 HTTP ambiguity | Fixture accepts POST then drops connection; separately crash Engine after write, after response headers, and after response body. | State is `HTTP_ATTEMPTED_UNCERTAIN` unless response+outcome is durably known; UI never infers call count from request ID. |
| X8 Save false-negative | Inject failure after rename/readback and after file fsync but before directory fsync. | Conflicting run/Thought evidence becomes `INCONSISTENT`; never a plain retry or unverified `COMPLETED`. |
| X9 Delayed POST versus copy | Delay original POST before Engine handler, observe GET 404, attempt Copy-as-new, then release original. | UI/API prevents the second process attempt or Engine fence ensures only one semantic approval can execute. |
| X10 Core crash lifecycle | Crash Core after durable SUBMITTED, before send, during send, after Engine claim, and after Engine completion. Restart read-only. | Startup/GET causes no POST. Recovery queries existing Engine run before any resend; fixture count remains at most one. |
| X11 Restore/lost-ledger matrix | Restore Core only, Engine only, both to pre-claim, and Thought without run ledger. | Processing locks in `RECOVERY_REQUIRED` whenever uniqueness cannot be proven; no automatic model call. |
| X12 Privacy inventory | Use a unique synthetic attention marker and inspect every Engine/Core journal, backup artifact, API response, and log. | Marker appears only in explicitly approved local locations; `calls.jsonl` behavior matches documented retention/redaction. |
| X13 Disk/read-only limits | Fill claim volume, make it read-only, exceed cap, and inject parent-directory sync failure. | New runs fail closed before HTTP; existing status remains readable; no rotation/truncation. |
| X14 Completion cross-product | Generate every combination of outcome `{COMPLETED, FAILED, missing}` × matching Note `{0,1,2}` × matching step `{0,1,2}`. | Only exactly one Note + one step + compatible durable outcome yields `COMPLETED`; all conflicts are `INCONSISTENT`. |

## Smallest safe corrected design

1. Core obtains or creates a single-use authorization artifact bound to the exact immutable nudge tuple and submits it over a Core-only Engine route.
2. Engine validates that artifact and current permission before all successful POST responses, including replay.
3. Engine atomically creates an immutable claim for the nudge across processes; uncertainty locks processing into recovery rather than permitting another attempt.
4. After the model slot is acquired, Engine revalidates immediately before the actual HTTP POST and durably records call phase without claiming that the client log proves server execution.
5. Immediately before saving, Engine revalidates permission and Thought state, then atomically stores one Note and one developed step carrying the same nudge/request IDs.
6. Core is read-only on startup/status paths and derives `COMPLETED` only from consistent Engine outcome plus exact Thought evidence. Ambiguity is visible and blocks retry/copy.
7. Restore procedures preserve the claim ledger or lock processing pending an integrity audit. No ordinary backup restore may silently rewind the run-once fence.

## Readiness recommendation

**Do not implement Stage 2 from PR #101 yet.** Revise the specification for A-01 through A-07, then update S/F cases and obtain owner approval. After that, implement only the narrow vertical slice above and run the isolated tests before any owner-host trial. CI green alone is not acceptance.

## Audit limitations

- This was a design/source audit; no proposed Stage 2 code exists to execute.
- No live owner installation, journals, grants, model, or UI was accessed or changed.
- Actual filesystem guarantees depend on the deployment filesystem; the implementation must test the real persistence primitive in an isolated environment matching production.
- No client can prove remote model execution after a crash at an ambiguous network boundary without a server-side idempotency/receipt mechanism.
