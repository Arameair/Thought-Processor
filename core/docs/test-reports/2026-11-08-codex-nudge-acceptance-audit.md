# Independent TASS Cognition manual-nudge acceptance audit

**Date:** 2026-11-08 (America/Chicago)

**Auditor:** Codex, independent rerun and source review

**Scope:** Thought-Processor PR [#98](https://github.com/Arameair/Thought-Processor/pull/98), Identity PR [#42](https://github.com/Arameair/identity/pull/42), Issues [#96](https://github.com/Arameair/Thought-Processor/issues/96) and [#79](https://github.com/Arameair/Thought-Processor/issues/79)

**Safety:** No merge, deployment, update, rollback, restore, live restart, live grant change, live nudge POST, external message, autonomy change, or pulse change was performed.

## Disposition

**PASS for the installed, draft-only manual nudge workbench; BLOCKED for Process and broader production readiness.**

No PR #98 nudge defect reproduced. The owner-host draft survived the completed restart, remained one valid append-only record scoped to the current owner and Engine link, and caused no Engine model request, cycle, or job. Authorization, revocation, replay, cross-entity protection, malformed input, journal failure, browser, and native integration checks passed in isolated environments.

This is not a recommendation to merge or call the whole application production-ready. Direct `docker ps` health inspection was unavailable to the auditor without an owner sudo credential, and real-Docker `restore-previous`/interrupted recovery remains unexercised. Process correctly remains unavailable.

## Exact revisions

| Component | Installed | Tested |
|---|---|---|
| Core | `094f43ba2bc167ace8835bf286f37951fee432c9` | same |
| Thought Engine | `d69d3917d1601d806d4b2ba2254693e1adb92b6b` | same pinned source/binary |
| Identity | `64fdf8e8cf926df5ea6bb6c23fbab864d0f09e3b` | same pinned source/binary |
| Regression base | n/a | Core `8e040b33e7a492f042a1f4449b8c7b013c227965` |

The installed record, Core health version, candidate Git commit, and pinned source directories agreed on these values.

## Result matrix

| ID | Provenance | Result | Evidence |
|---|---|---|---|
| LIVE-01 installed pins | Owner host, read-only | **PASS** | `current.env`, `tass.sh status`, and `/api/health` report the three expected pins; Core reports full `094f43b...`. |
| LIVE-02 service and external Nemotron path | Owner host, read-only | **PASS** | Core health: `engine_reachable:true`, `model_ok:true`; app config: `MODEL_MODE=external`, `MODEL_URL=http://model:8081`; live process args name worker `nemotron`. Identity answered `LOCKED` with the existing entity present. |
| LIVE-03 Docker container health state | Owner host | **NOT RUN** | Docker socket is `root:docker` mode `0660`; the owner account is not in that group and `sudo -n docker` requires a password. Service endpoints and all three Docker-hosted processes were healthy, but Docker's own per-container health field was not independently read. |
| LIVE-04 safety gate | Owner host, read-only | **PASS** | Before and after all isolated tests: `action gate: CHAT_ONLY`; Identity remained `LOCKED`; last live pulse stayed unchanged. |
| LIVE-05 post-restart draft durability | Owner host, read-only | **PASS** | Draft journal mtime `11:42:24`; Identity/Engine/Core process starts `11:45:14`/`11:45:19`/`11:45:25`; post-restart parse found one valid record. |
| LIVE-06 journal scope and truthfulness | Owner host, sanitized parse | **PASS** | `nudges.jsonl` mode `0600`; exactly one record; owner and sender match the current Identity link; Engine scope matches the current Engine link; status `DRAFT`; draft decision present; `model_calls:0`; processing `BLOCKED`; processing cycle, processing authorization, and result Thought are null. Private action text and identifiers were not printed. |
| LIVE-07 zero Engine work on reads | Owner host, read-only | **PASS** | Before/after loading `/cognition` and parsing the saved draft: model requests `152 -> 152`, cycles `288 -> 288`, jobs `12 -> 12`. Final counts after the full audit remained `152 / 288 / 12`. |
| LIVE-08 authorized UI history after restart | Owner host | **BLOCKED** | Identity was intentionally left `LOCKED`; the auditor did not unlock it. Durable record readability and scope passed directly, but a fresh authorized GET/UI history view was not run. |
| LIVE-09 Process absent | Owner host + source + isolated router | **PASS / BLOCKED by design** | Live `POST /api/nudges/process` returned `404`; router tests also rejected both candidate process paths. No processing grant or handler exists. |
| AUTH-01 denial and distinct grant | Isolated router/native services | **PASS** | LOCKED, unlinked, and default `thought.process` cases denied; no journal was created. Identity test confirms `cognition.nudge.draft` is separate, non-default, and does not add `skill.discover`. |
| AUTH-02 grant/revoke/replay | Isolated real Go services | **PASS** | Wrong passphrase refused; explicit grant allowed save; same request replayed the same draft; revoke made the same replay return `403`; GET history stayed readable; regrant behavior passed unit tests. |
| AUTH-03 cross-entity/relink | Isolated unit/router | **PASS** | Other-entity Thoughts refused; old idempotency key after Engine relink returned conflict without old draft content; old history stayed hidden. |
| INPUT-01 malformed/adversarial requests | Isolated real router | **PASS** | Wrong/missing token, foreign Host/Origin, wrong content type, malformed JSON, external recipients, invalid values, traversal-shaped IDs, `169` hours, and max-int expiry rejected. |
| JOURNAL-01 failure handling | Isolated unit/router | **PASS** | Failed open and directory-at-journal path did not report success; partial write produced one counted damaged line, preserved it, and allowed the next record on a new line. |
| GO-01 Core vet/test/race | Exact pinned Core source | **PASS** | `go vet ./...`; `go test ./... -count=1`; `go test -race ./... -count=1`. |
| GO-02 Engine vet/test/race | Exact pinned Engine source | **PASS** | Same three commands passed. |
| GO-03 Identity vet/test/race | Exact pinned Identity source | **PASS** | Same three commands passed, including `TestNudgeDraftGrantIsSeparateAndNotDefault`. |
| BROWSER-01 Cognition | Isolated Chromium fixture | **PASS** | `cognition_go.py`: 2/2. |
| BROWSER-02 manual nudge | Isolated Chromium fixture | **PASS** | `cognition_nudge.py`: 1/1; double-click produced one record, Process disabled, and the only POST was `/api/nudges`. |
| NATIVE-01 pinned real services | Isolated network namespace, fixture model | **PASS** | `nudge_native.py`: 25/25; real pinned Identity/Engine/Core binaries, disposable data, no GitHub, fixture model (not Nemotron). Model requests `6 -> 6`; cycles `1 -> 1`; jobs `0 -> 0` across all nudge calls. |
| UPDATE-01 recovery fixtures | Isolated fake Docker | **PASS** | `test_updater_recovery.py -v`: 14/14. |
| LAUNCH-01 maintained launcher fixtures | Isolated fake Docker | **PASS** | `test_launcher_regression.py -v`: 6/6. |
| LAUNCH-02 legacy shell fixture | Candidate and untouched base | **FAIL (pre-existing test defect)** | Both candidate and base fail identically: expected Compose exit `37`, but the stale fake deployment is first rejected by the Issue #50 guard (`saved Core 111... predates the Issue #50 safety fix`). Not a PR #98 regression. |
| PULSE-01 candidate suite | Isolated network/data, fixture model | **PASS** | `pulse_multicycle_native.py`: 24/24 at `094f43b`. |
| PULSE-02 base comparison | Separate isolated network/data, fixture model | **PASS** | Same suite: 24/24 at untouched base `8e040b3`. |
| RESTORE-01 live restore/recover | Owner host | **NOT RUN** | Prohibited by this audit's no-restart/no-rollback safety boundary. |

## Independent source findings

The implementation order is correct for strict revocation semantics: `SaveNudgeDraft` validates, establishes current Cognition scope, and requests a fresh `cognition.nudge.draft` Identity decision **before** returning an idempotent replay. Replay additionally requires the same owner, Engine link, key, and body digest. A relink returns a content-free conflict.

The durable writer uses append-only open flags, mode `0600`, newline repair after a cut-off line, `fsync`, and no rewrite path. Stored drafts hard-code `DRAFT`, zero model calls, and the shared `BLOCKED` processing state. There is no process route in `main.go`.

Identity maps the owner-facing `cognition.nudge-draft` entry only to `cognition.nudge.draft`. It is not a default grant, does not imply `skill.discover`, does not imply a skill run or `cognition.nudge.process`, requires the passphrase to enable, and can be revoked immediately.

## Pulse regression investigation

Claude reported two failures in an earlier 22/24 run on both candidate and base:

1. `Core health names the model problem`
2. `GitHub down: Core reports sync problem; Engine still ticks every ~20 s`

In this audit, both assertions passed on candidate and base under simultaneous clean runs, separate network namespaces, separate data, the same fixture seed, pinned Engine/Identity binaries, and the same host load. Candidate and base each finished 24/24 with 37 ticks, 22 jobs, and 55 fixture-model requests.

**Classification:** the failures do not currently reproduce and are not introduced by PR #98. The earlier paired failure on both revisions remains evidence of an environment-, state-, or timing-sensitive observation; this audit did not identify a code fix between those paired revisions. Keep the earlier 22/24 result in history rather than rewriting it as false, and prefer another clean CI/host rerun before declaring the tests non-flaky.

## Backup and rollback readiness

- Current pre-update backup: `20261108T173009Z-before-094f43ba2bc1.tgz`.
- Independent SHA-256 comparison against its sidecar: **PASS**.
- Archive opens and contains Core data, Engine data, Identity data, and secrets: **PASS** (151 entries; names/content not published).
- `previous.env` records Core `8e040b3`, Engine `d69d391`, Identity `7da7f7c`.
- The owner-host update history records the update to `094f43b` as `OK` with that backup.
- Recovery fixtures cover unhealthy/data-damaging candidates, bad backups, interrupted phases, restore failure, rollback failure, operation locking, and manual backup: **14/14 PASS**.

Known limitations:

- Real-Docker `restore-previous`, interrupted `recover`, and validation-failure auto-restore were **NOT RUN** in this audit.
- Many older backup archives are listed as `unverified (older)` because they predate checksum sidecars.
- The checksum sidecar contains a bare digest rather than `sha256sum -c` format; direct digest comparison passed.

## Commands executed

Sensitive output was filtered; live journal contents, Identity secrets, service keys, passphrases, tokens, and private conversation text were never printed or published.

```bash
# Live read-only inventory
./tass.sh status
curl -sS http://127.0.0.1:8797/api/health
curl -sS http://127.0.0.1:8796/v1/state
wc -l ~/.tass-app/data/engine/{model-requests,cycles,jobs}.jsonl
curl -sS -o /dev/null -w '%{http_code}' http://127.0.0.1:8797/cognition
curl -sS -o /dev/null -w '%{http_code}' -X POST http://127.0.0.1:8797/api/nudges/process

# Go verification, repeated in each exact pinned source tree
go vet ./...
go test ./... -count=1
go test -race ./... -count=1

# Browser fixtures (disposable venv/runtime)
python core/test/browser/cognition_go.py
python core/test/browser/cognition_nudge.py

# Native integration in an unprivileged network namespace
unshare -Urn ... python3 core/test/nudge_native.py

# Launcher and recovery
python3 core/test/test_updater_recovery.py -v
python3 core/test/test_launcher_regression.py -v
bash core/test/launcher_regression.sh       # candidate and base

# Pulse comparison, two separate unprivileged network namespaces
unshare -Urn ... CORE_BIN=tass-core-candidate python3 core/test/pulse_multicycle_native.py
unshare -Urn ... CORE_BIN=tass-core-base      python3 core/test/pulse_multicycle_native.py
```

## Remaining blockers and recommendation

1. Keep **Process BLOCKED**. No processing capability was accepted or tested, and no processing grant/route exists.
2. Do not merge solely from green CI or this report. Obtain a direct owner-authorized Docker container-health snapshot and complete a separately approved real-Docker restore/recovery exercise for Issue #79.
3. A fresh authorized UI read of the saved draft after restart remains blocked only because Identity was intentionally left locked; the underlying post-restart journal record is independently readable and valid.
4. Repair or remove the stale legacy `launcher_regression.sh` fixture so it tests Compose exit propagation without being preempted by the Issue #50 guard.
5. Preserve both the earlier 22/24 pulse observation and this clean 24/24 candidate/base rerun; one more clean environment rerun would help distinguish flakiness from stale-state contamination.

**Recommendation:** continue the installed candidate only as a supervised, local, draft-only workbench under `CHAT_ONLY`. The nudge draft slice itself meets acceptance; broader production readiness and merge remain blocked on the explicit items above.
