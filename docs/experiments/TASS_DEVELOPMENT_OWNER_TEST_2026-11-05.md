# TASS Development Platform — Owner-Machine Test Notes

**Test date:** 2026-11-05  
**Tester:** Codex, acting under the owner's bounded acceptance instructions  
**Overall result:** **PARTIAL**  
**GitHub report:** https://github.com/Arameair/Thought-Processor/issues/78

## 1. Scope and safety boundaries

The goal was to test the already-running TASS Development Platform on the owner's machine without changing the deployed build or expanding TASS's authority.

The following boundaries were maintained:

- No update, deployment, container restart, or service replacement.
- No Identity grant or revocation.
- No approval of a skill execution.
- Autonomous remained OFF.
- Work Pulse remained OFF.
- Learning remained ON.
- Action gate remained `CHAT_ONLY`.
- Thought-Processor issue #76, PR #69, and TASS issue #24 were not modified.
- No GitHub mailbox round trip or hourly GPT-6 schedule was attempted.
- Invalid graph connections created during negative testing were removed before saving.

## 2. Exact tested deployment

| Component | Exact revision |
| --- | --- |
| Core | `60661aa807cac4d93ebc461e1447f3c1e6ad9473` |
| Thought Engine | `21b3d502775b3ed065ab445b88f51909500770b3` |
| Identity | `87dd9243c721fd11701575033d95d3dcf286f765` |

Deployment record:

- Deployed at `2026-11-05T17:53:08Z`.
- Core source snapshot: `/home/dustin/.tass-app/deploy/src/core-60661aa807ca/core`.
- The snapshot was a clean detached checkout at the exact Core SHA.
- Owner backup observed: `20261105T175240Z-manual.tgz`.
- Automatic pre-deployment backup observed: `20261105T175309Z-before-60661aa807ca.tgz`.

Live URLs:

- Core: `http://127.0.0.1:8797/`
- Development: `http://127.0.0.1:8797/dev`

The runbook mentioned port 8795, but the owner's current live configuration was actually reachable on 8797. Port 8795 refused the connection.

## 3. Preflight and final-state evidence

The command used for both preflight and final verification was:

```bash
cd /home/dustin/tass-one-app/core
timeout 20s ./tass.sh status
```

Observed at the end of testing:

```text
Deployed: core 60661aa807ca · engine 21b3d502775b · identity 87dd9243c721
Identity: UNLOCKED TASS
State: READY
engine reachable True · model ok True
action gate: CHAT_ONLY
last pulse tick: pulse_000065_174324
```

The last pulse remained `pulse_000065_174324`, demonstrating that this test did not start Pulse or produce a new tick.

## 4. Development library

**Result: PASS**

The live Development library displayed real project material:

- Lessons:
  - `My work pulse` — LEARNED
  - `How I learn` — LEARNED
  - `Building together` — LEARNED
- Notebook text and code cells — DISCOVERED
- Skill:
  - `@lesson-workspace.list-lessons` — GRANTED

## 5. Brain Builder graph test

### 5.1 Blocks added

Four blocks were added to the live canvas:

1. `List every lesson` — notebook code cell, DISCOVERED
2. `@lesson-workspace.list-lessons` — skill, GRANTED
3. `My work pulse` — lesson, LEARNED
4. `Owner acceptance note` — owner note

The note body was:

> This note documents the bounded Development test. It does not authorize execution.

The note was dragged to another canvas position. The canvas registered the movement as an unsaved change.

### 5.2 Clean saved connections

Five intended connections were created:

| From | Relation | To | Reason |
| --- | --- | --- | --- |
| List every lesson | `uses` | @lesson-workspace.list-lessons | This notebook step needs the supervised read-only list-lessons skill. |
| My work pulse | `supports` | List every lesson | The pulse lesson explains how work is observed without duplicate execution. |
| Owner acceptance note | `alternative` | My work pulse | The owner note is an alternative explanation, not executable authority. |
| Owner acceptance note | `requires` | @lesson-workspace.list-lessons | Any future listing claim requires the registered read-only skill and approval workflow. |
| List every lesson | `next` | Owner acceptance note | After reviewing the discovered cells, record the bounded owner-test conclusion. |

### 5.3 Structural check

**Result: PASS**

The clean graph reported:

```text
PASS — Checked 4 blocks and 5 connections: PASS (0 stale). Nothing was run.
NOT RUN — structural review only
```

### 5.4 Persistence

**Result: PASS**

Revision 1 was saved with:

```text
Owner acceptance graph: lesson, notebook cell, skill, note, and five relation types.
```

After reloading the page, revision 1 still contained all four blocks and five connections.

## 6. Brain Builder negative tests and defects

### Defect 1 — `uses` accepts a non-skill target

**Result: FAIL**

Reproduction:

1. Select `My work pulse`, which is a lesson.
2. Choose relation `uses`.
3. Choose `List every lesson`, a notebook cell, as the target.
4. Enter `Intentional invalid connection test.` as the reason.
5. Click Connect.

Actual result:

- The UI accepted the edge and displayed `Connection added (uses). Save to keep it. Nothing ran.`

Expected result:

- The connection should be rejected because a `uses` relation must target a skill.

Cleanup:

- The invalid connection was immediately unlinked.
- It was never saved.

### Defect 2 — `next` accepts a directed cycle

**Result: FAIL**

Starting state:

```text
List every lesson --next--> Owner acceptance note
```

Reproduction:

1. Select `Owner acceptance note`.
2. Choose relation `next`.
3. Target `List every lesson`.
4. Enter `Intentional next-cycle rejection test.` as the reason.
5. Click Connect.

Actual result:

- The UI accepted the reverse edge, creating a directed cycle.

Expected result:

- The cycle should be rejected.

Cleanup:

- The cycle-producing edge was immediately unlinked.
- It was never saved.

## 7. Edit, Undo, unlink, and history

**Result: PASS**

The following sequence was verified:

1. Changed the owner note's `alternative` relation to `supports`.
2. Changed its reason to:
   `The owner note supports the lesson context without granting execution authority.`
3. Used Undo and verified that both the original `alternative` relation and original reason returned.
4. Reapplied the edit.
5. Unlinked the `requires` edge.
6. Saved revision 2 with four blocks and four connections.
7. Restored revision 1.

Restoring revision 1 created a new revision 3 rather than overwriting history. Revisions 1 and 2 remained visible.

## 8. Brain focus view

**Result: PASS**

With `My work pulse` selected and Brain view enabled:

- Selected block opacity: 1
- Direct neighbors remained at opacity 1
- The unconnected skill was dimmed to opacity 0.18

This demonstrated that focus view distinguished the selected block, direct neighbors, and non-neighbors.

## 9. Two-tab stale-revision protection

**Result: PASS**

Test sequence:

1. Opened two Development tabs at revision 3.
2. In tab 1, moved a block by one step and saved revision 4.
3. In tab 2, which was still based on revision 3, moved a block and attempted to save.

The stale tab was refused with:

```text
Not saved: the brain was saved from another tab or by TASS; reload before saving
```

Revision 3 was then restored as new revision 5 so that the final saved graph returned to the clean four-block, five-connection state.

## 10. Real saved-brain conversation

**Result: PARTIAL**

Prompt:

> Examine the saved brain revision. Which lesson, notebook cell, and skill are connected? Explain what each can do and what cannot actually execute yet.

The live Nemotron path created thought cycle:

```text
tp_20261105T180529_564a38bd6ce0
```

Verified cycle evidence:

- Brain context: revision 5, read-only.
- Identity authorized `model.request`.
- Identity authorized `thought.process`.
- Proposed action: `NONE`.
- Action result: `NO_ACTION`.
- The cycle rationale stated that the graph was advisory and execution would require an Identity grant plus owner approval.

Visible reply:

> The saved brain revision connects the lesson "My work pulse", the notebook cell "List every lesson", and the skill "@lesson-workspace.list-lessons".

The reply correctly named the connected items but did not complete the requested explanation of what each item could do. This is why the result is PARTIAL.

## 11. Workspace and read-only project context

### 11.1 Project-file view

**Result: PASS**

The Workspace opened `tass.md` with:

- Repository and branch provenance.
- Local blob identifier.
- Explicit `read-only` labeling.
- The current local TASS document contents.

### 11.2 File-grounded question

**Result: PASS**

Question:

> What does the selected tass.md say under Who I am?

TASS answered:

> The tass.md says "I am TASS, a local AI companion. I'm new: I learn who I am with my owner."

The resulting cycle reported:

- Proposed action: `NONE`
- Action result: `NO_ACTION`
- Reason: the document was read-only and no action was required.

## 12. Skill request and approval boundary

**Result: PASS for request gating; execution NOT TESTED**

The owner-facing `Request a run` control was exercised for:

```text
@lesson-workspace.list-lessons
```

Result:

```text
AWAITING_APPROVAL
executed: no
```

Receipt observed:

```text
skr_20261105T180613_8f006e4d
```

The `Approve run` control was deliberately not used. The request may remain pending in the live UI and must not be approved unless the owner intentionally wants the read-only skill to run.

## 13. HTTP and security checks

**Result: PASS**

| Probe | Result |
| --- | --- |
| `GET /` | 200 |
| `GET /dev` | 200 |
| Unauthenticated `GET /api/status` | 401 |
| Unauthenticated `GET /api/brain` | 401 |
| `Host: evil.test` against `/` | 403 |

Secret storage permissions were checked without reading or printing secret contents:

- `/home/dustin/.tass-app/secrets` — mode 700
- Files beneath it — mode 600

## 14. Automated source verification

All automated commands ran against the exact pinned deployed source snapshots. Go tests used their own temporary data and did not point at the live owner profile.

### Core

Source:

```text
/home/dustin/.tass-app/deploy/src/core-60661aa807ca/core
```

Commands and results:

```text
go vet ./...        PASS
go test ./...       PASS
go test -race ./... PASS
```

Packages passing tests included:

- `core/cmd/tass-core`
- `core/lessons`
- `core/tass`
- `core/tassmd`

### Identity

Source:

```text
/home/dustin/.tass-app/deploy/src/identity-87dd9243c721
```

Commands and results:

```text
go vet ./...        PASS
go test ./...       PASS
go test -race ./... PASS
```

### Thought Engine

Source:

```text
/home/dustin/.tass-app/deploy/src/engine-21b3d502775b
```

Commands and results:

```text
go vet ./...        PASS
go test ./...       PASS
go test -race ./... PASS
```

## 15. Tests deliberately not run

### Bundled Playwright fixture scripts

**Result: NOT TESTED**

The fixture browser scripts hard-code Identity port 8796. That port was already owned by the live TASS Identity service. The live service was not stopped or displaced merely to run the fixture.

Manual live browser coverage was completed for the Development, Talk, Workspace, read-only project-file, and skill-request surfaces.

### Skill execution

**Result: NOT TESTED**

The request reached the approval boundary, but no owner-authorized approval was provided and no skill executed.

### Pulse and Autonomous

**Result: NOT TESTED in this run**

Neither setting was enabled. The final action gate remained `CHAT_ONLY` and no new pulse tick appeared.

### GPT-6 ↔ GitHub ↔ TASS hourly loop

**Result: NOT TESTED / NOT CREATED**

The mailbox and hourly scheduler effort remains separately governed by Thought-Processor issue #76. This run did not modify PR #69, issue #76, TASS issue #24, relay configuration, or any scheduler.

## 16. Final assessment

**Overall: PARTIAL**

The live Development Platform successfully demonstrated:

- Real persisted brain revisions.
- Safe revision restoration with preserved history.
- Concurrent-tab overwrite protection.
- Read-only saved-brain context reaching the real model path.
- Identity-authorized thought processing with no proposed or executed action.
- Read-only project context.
- Skill request gating that stopped before execution.
- Healthy Core, Identity, and Thought Engine source tests, including race detection.
- Correct unauthenticated API refusal and local-host enforcement.

Acceptance is not fully PASS because:

1. `uses` accepts a non-skill target.
2. `next` accepts a directed cycle.
3. The saved-brain model reply did not fully answer the requested explanation.

## 17. Recommendation

1. Add server-side validation enforcing relation target types, especially `uses -> skill`.
2. Add directed-cycle detection for `next` edges.
3. Add regression tests for both invalid graph cases.
4. Strengthen the saved-brain prompt or response validation so every requested explanation is returned.
5. Rerun this acceptance against the corrected experimental build.
6. Keep Autonomous and Pulse OFF until the validation defects are resolved and owner acceptance passes.


