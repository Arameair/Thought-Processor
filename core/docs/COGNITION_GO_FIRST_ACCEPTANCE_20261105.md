# TASS Cognition — Go-first isolated owner acceptance

Date: 2026-11-05. Engineering task: Thought-Processor #82, evidence #80, release gate #79. **Status: code candidate, NOT deployed or owner accepted.**

## Immutable source and service pins

- Core feature branch: `experiment/cognition-go-first-gpt6-20261105`, derived from unified integration `913a4ff687054fb8e2dddf076ccaac992398394a` and owner-running Core `60661aa807cac4d93ebc461e1447f3c1e6ad9473`.
- Engine candidate branch: `Arameair/Thought-engine:experiment/cognition-journal-projection-20261105`, pinned `cab101c3d1d62953d99dd929e4d1b11ad55eb750` (draft PR #15). This is **not currently on the owner's computer**; it extends installed `21b3d502775b3ed065ab445b88f51909500770b3`.
- Identity pin: `87dd9243c721fd11701575033d95d3dcf286f765` (unchanged). No new permission, no schema migration or standalone service.
- Only reviewed Core candidate should ever be installed through the existing updater **after** #79 verified recovery approval. Until then, installed Core/Engine/Identity are unchanged.
- Images/digests on owner's machine: NOT INSPECTED. Go CI tests use isolated fixtures and a Docker build but are not a running owner image manifest.

## Why the visual prototype alone was not enough

PR #84 was intentionally a visual/fixture proof: it grouped cycles into a supposed thought shelf via browser JS. Canonical persistent `Thought` and `Trace` records already live in the Go Engine's `GET /api/entities/{eid}` / `thoughts/{tid}` endpoints, **not** in the browser. A cycle's `spark.thought_id` joins a Thought **only** when IDs and Engine entity match; otherwise the cycle remains visibly UNLINKED. Engine's `cycles.jsonl` is the only chronological journal source for playback. Predictive model claims and `NO_ACTION` are never accepted as proof of tool execution.

## Implementation

1. **Thought Engine read-only journal** (new Go `cognition_journal.go` and authenticated `GET /api/cycles/{id}/journal`): limits each read to 8 MiB tail and 96 steps, preserves actual timestamp and byte offset, flags duplicate/out-of-order/missing/truncated/partial events; only allowlisted stage IDs/statuses and executor-*reported* outcome fields. No raw `data` JSONL, prompts, model reply, model reasoning, secrets or authorization detail. Existing Engine bearer key plus local-host guard; no model or pulse POST.
2. **TASS Core Go clients/projection** (`core/engine/cognition.go`, `core/tass/cognition.go`): current canonical Identity UNLOCKED state + matching `conversation` and `thought.process` Identity decision IDs required on every sensitive read, Engine-local entity matched by unique companion name, typed real Thoughts/cycles, deterministic ordering, strict ID joins; no topic/title heuristic. Unrecorded evidence explicit. Bounded details and 20 newest Engine cycles.
3. **Owner-only same-origin API**: `GET /api/cognition/library`, `GET /api/cognition/thoughts/{id}`, `GET /api/cognition/cycles/{id}/timeline`, `GET /api/cognition/cycles/{id}/export?format=md|json`. Existing page token and local-host controls. `no-store`, reject invalid IDs or unauthorized identity. No filesystem paths in API.
4. **Go export**: typed, allowlist-only metadata (commits, exact IDs, journal offsets, recorded stages, status labels and missing stages). Markdown or JSON capped at 64 KiB. No raw logs, user text, private model prompts, reasoning or Identity token. Explicit browser click required to copy/download; never publishes to GitHub.
5. **Existing embedded Netflix-like UI**: preserves PR #84 dark shelves, navigation and detail, now reads only Go-generated cards. Persistent Thoughts have a separate shelf from cycles/outcomes; no demo fallback when connected Core is unavailable. Play/Pause/Step/View Logs only advance a cursor across immutable historical entries. No `POST` exists in Cognition JavaScript.
6. **Development bridge**: selected thought/cycle and saved Brain revision become a query-string reference to `/dev`. That page displays **UNEXECUTED DEVELOPMENT PROPOSAL** but creates no note, graph edge, run, authorization change or deployment automatically.

## Test inventory (keep PASS distinct from live acceptance)

- Engine isolated Go unit/race tests (including fake-model HTTP, secret injection, missing/partial/duplicate/out-of-order journal): see Engine PR #15 CI `42093780542` PASS.
- Core Go vet and race tests: validate LOCKED Identity, wrong Engine entity, exact persisted Thought/cycle ID joins, no invented links, `NO_ACTION` as APP-REPORTED, engine GET-only, typed export secret stripping and strict read scope.
- Core HTTP page/API tests: local Host, missing token 401, wrong export format/path rejection, no-store.
- Browser: Chromium fixture tests (not real owner) browse persistent Thought and unlinked cycle, playback, pause/step, view recorded logs, copy selected sanitized report, download, Development reference, no POST and unavailable-source no-fallback. Record run URL and any failures.
- **NOT RUN**: real owner-machine `/cognition`, real local Nemotron/Identity/Engine cycle journal, backup/restore, owner copy/download, real note/graph selection, comparison of Core and Engine service logs or Identity decisions.

## Owner read-only preflight; no automatic deployment

From the existing checkout on the owner machine, before approving any candidate:

```bash
cd ~/tass-one-app/core
./tass.sh status
git rev-parse HEAD
git status --short
git fetch origin experiment/cognition-go-first-gpt6-20261105
./tass.sh check-update origin/experiment/cognition-go-first-gpt6-20261105
```

Verify pinned engine candidate and Identity release against the PR. Do not use an older Core candidate, merge alpha/main, change relay/Pulse/Autonomous, approve skills, or restart running containers just for this review.

## Owner-machine acceptance (only after separate explicit approval and #79 recovery guarantee)

1. Use the reviewed update path, ensure a verified automatic snapshot of the prior Core/Engine/Identity/data; reject deployment if backup/restore checks fail. Identity may require local owner unlock.
2. Confirm `READY`, `UNLOCKED`, `CHAT_ONLY`, `autonomous=false`, `pulse=false`. Open `http://127.0.0.1:8797/cognition` (or current configured Core port).
3. Verify persistent Thought shelf comes from Engine entity, not generated from a recent cycle. If none exist, accept `NOT RECORDED` and verify the separate recent-cycle shelf; do **not** fabricate a Thought.
4. Select one actually linked Thought/cycle, compare exact `thought_id`, `trace_id`, `cycle_id`, Engine entity ID and journal entries with Engine's local `cycles.jsonl`; check recorded timestamps, offset, gaps and source. A `NO_ACTION` cycle is explicitly **not executed**.
5. Use Play/Pause/Step/View Logs. Inspect Engine model-request and Identity decision records locally to confirm the **viewing session created no action, skill or new model cycle**. Any new authorization *read* decisions may legitimately be recorded by Identity.
6. Click Copy Evidence and Download Markdown/JSON; check exact ID/provenance and absence of secrets, private prompts, filesystem paths or model reasoning.
7. Click Try adjustment in Development. Verify unexecuted reference text with saved Brain revision; `/dev` Brain revision, notes, skills, jobs and TASS configuration must remain unchanged unless the owner explicitly chooses to edit and save later.
8. Confirm the single stack is still healthy and settings unchanged; record full Core, Engine, Identity SHAs, user-checked IDs, and PASS/FAIL evidence. If owner requests return to previous accepted version, use **tested** Restore Previous via #79 and verify existing Identity+data are preserved.

## Defects / limitations

- Direct Engine summary `GET /api/cycles` is bounded to recent 20; historical Thoughts may not have a recent linked cycle. State remains UNLINKED/NOT RECORDED.
- Engine journal endpoint reads a bounded tail, not an unbounded historical index; older stages outside it are marked TRUNCATED.
- Engine-local entity ID is distinct from canonical Identity `entity_id`; the Core checks unique display-name match and cycle entity IDs, never claims they are identical.
- The Engine's `EXECUTED` status originates from the executor's recorded report. Independent verification against a skill receipt must be done separately; exported results are labelled APP-REPORTED.
- This slice does **not** implement full raw local audit browsing, generalized cross-process Identity log joins, experiment execution, A/B tests, notification, speech, notebook execution, new models or automatic deployment.
- Transactional updater/Restore Previous is a separate #79 release blocker and is **not** implemented by this Cognition feature PR.

**Status: PARTIAL until actual owner-machine evidence and approved release gates.**
