# Cognition shelves — runnable read-only prototype (2026-11-05)

**Go-first NEXT ITERATION:** [COGNITION_GO_FIRST_IMPLEMENTATION_20261105.md](COGNITION_GO_FIRST_IMPLEMENTATION_20261105.md) is the authoritative implementation contract. This file documents the existing UI-only draft. That draft currently constructs cards/associations in browser JavaScript, not Go; it treats each recent cycle as a thought card, rather than consuming canonical persistent Engine Thoughts. Future work must keep the Netflix-style visual design but move classification, evidence correlation, stage playback projection and sanitized exports into Go. **Do not send the older GPT-6 handoff below without this correction. Speaking is out of scope.**

**Scope:** A Netflix-style browsing experience for TASS's recorded thought cycles, observed outcomes, learned lessons, notebook cells, skills/receipts, and owner notes. This is *not* a working autonomous cognition system, a notebook executor, or a replacement for Development.

**Branch:** `experiment/cognition-shelf-prototype-20261105` (branched from `experiment/tass-unified-integration-20261105`).  
**Target PR:** the unified integration branch. Never fast-forward deployed Core or merge to alpha/main without owner approval.  
**Deployed owner Core before this work:** `60661aa807cac4d93ebc461e1447f3c1e6ad9473`, Engine `21b3d502775b3ed065ab445b88f51909500770b3`, Identity `87dd9243c721fd11701575033d95d3dcf286f765`.

## Design and behavior

- `core/cmd/tass-core/cognition.html`: responsive dark horizontal shelves and inspectable cards.
- `core/cmd/tass-core/main.go`: embeds `cognition.html` and serves `GET /cognition` with the same page token, local-host check, no-store cache, and CSP as existing `/dev`.
- `page.html` and `dev.html`: linked navigation. This is **inside existing Core**; no new container/model/service.
- **Read-only.** Live cards use **existing** `GET /api/status` (cycle data and Identity/Engine/supervised skill receipt state) and `GET /api/brain` (lessons/cells/skills/notes and saved graph revision) with the same `X-TASS-Token` as Development. No new API or database writes.
- Shows Thought Engine details **after a cycle card is selected and “Inspect Thought Engine” is clicked**. Nemotron model information is nested there; no flat model tile on the home view.
- Displays actual IDs and states, with explicit **NOT RECORDED** rather than fabricated output. `NO_ACTION` is never treated as tool execution; a notebook cell is not executable through Cognition.
- Search and category filters; card inspector; **Copy card summary** and **Copy evidence IDs**; explicit owner click. There is no automatic GitHub upload.
- Standalone `cognition.html` without the Core route uses **clearly marked DEMO FIXTURES**, never presented as real local TASS data. When served at `/cognition`, an API failure shows **LIVE UNAVAILABLE**, and does **not** silently switch to fixtures.

## Test now without touching the running TASS

An isolated **preview** can be tested without Docker, Identity, or a deployment:

```bash
cd ~/tass-one-app/core
git fetch origin experiment/cognition-shelf-prototype-20261105:refs/remotes/origin/experiment/cognition-shelf-prototype-20261105
git worktree add --detach ~/tass-cognition-preview origin/experiment/cognition-shelf-prototype-20261105
cd ~/tass-cognition-preview/core/cmd/tass-core
python3 -m http.server 8877 --bind 127.0.0.1
```

In the owner's browser, open **http://127.0.0.1:8877/cognition.html**. It must say **PREVIEW ONLY / DEMO FIXTURES**. Click and search cards, change category filters, inspect a thought, click **Inspect Thought Engine** to see the nested model, and try **Copy evidence IDs**. This preview is **not connected** to the owner's Core and cannot execute or approve anything. Stop it with Ctrl+C. The detached worktree may be removed later with `git worktree remove ~/tass-cognition-preview` (only after the preview server is stopped and no changes are needed).

If the worktree path already exists, **do not force-overwrite it**; use another safe directory or inspect the existing checkout. If port 8877 is occupied, choose an unused localhost port.

## Integrated route and live-data test (reviewed candidate only)

After a combined branch has passed CI and the owner deliberately approves a version update, the same UI loads at **http://127.0.0.1:<CORE_PORT>/cognition**, accessible through Talk → Cognition or Development → Cognition. **Do not install this isolated branch over another agent's newer unified candidate.** Use the single update/restore workflow under Issue #79; automatic pre-update backup in `./tass.sh update` already exists, but full crash recovery hardening is not yet verified.

Read-only acceptance:
1. /cognition appears, and banner explicitly says LIVE READ ONLY (or LIVE UNAVAILABLE if unauthenticated); fixture data never appears on a live failure.
2. The shelves match **actual** /api/status cycle and skill receipts and /api/brain blocks. Verify card counts and saved brain revision against the authenticated API response.
3. Open one real cycle. Check cycle ID, request, prediction, `outcome.status`, and evaluation. Action `NONE` with `NO_ACTION` is **not** a successful skill or notebook run.
4. Verify `@lesson-workspace.list-lessons` still shows it requires **Identity grant and owner approval**. Do not approve any pending receipt to test this read-only browser.
5. Inspect Thought Engine. The underlying model's status is shown there **only**, not as a top-level Cognition card.
6. Copy an ID summary and compare it to the corresponding Core/Engine receipt. Don't copy private full prompts or secrets into GitHub.
7. Reload, search and filter. Verify the browser never changes Identity grants, Pulse, Autonomous, brain revisions or current container processes.

## Automated verification and limitations

**Completed in the authoring sandbox**:
- Extracted inline script and ran `node --check`: PASS.
- Playwright Chromium `page.set_content` against explicitly labeled demo fixtures: PASS for shelves, filters, search, inspector, nested model and no JS exceptions.
- Playwright Chromium with **mocked** authenticated Core API JSON: PASS for joins/statuses and no silent demo fallback; this is a **fixture**, not a real Nemotron round trip.
- In this environment the browser disallowed even local HTTP navigation (`ERR_BLOCKED_BY_ADMINISTRATOR`), so end-to-end HTTP integration was **not run**.

**To run on a real Go checkout / CI**:
```bash
cd core
go vet ./...
go test ./...
go test -race ./...
```

`core/cmd/tass-core/cognition_test.go` adds GET route/token substitution/local-host tests. GitHub Actions core-tests passed for prior prototype branch head `8a6a4eef7473171a9094091e95f0de023a9e3ec4` (run 42089696870), including Go route coverage available in that workflow. This does not establish owner-machine live integration or cover new Go-first implementation work. The authoring sandbox itself did not run the complete repo Go build.

## GPT-6 handoff / next iterations

- Review branch against current unified integration HEAD before merging; handle concurrent PRs (#76 mailbox, #79 transactional updater, #78 graph validation, #80 export/UX).
- Retain a small accessible homepage; this UI should feel like a **shelf of thoughts**, not an infrastructure dashboard.
- The pinned Engine already defines canonical `Thought`, `Trace`, `ThoughtStep` and correlated `Cycle.Spark.ThoughtID` in `engine.go` and `cycles.go`; inspect those actual records and the owner-sensitive `/api/entities/{eid}` and `/api/entities/{eid}/thoughts/{tid}` routes. A cycle without a verified `thought_id` is an unlinked cycle, not a fabricated persistent Thought. Implement bounded Go-backed canonical thought projections. See the Go-first spec.
- Tie lessons/skills to cycles only with verified correlations. Graph connections can be advisory, do not claim causation without evidence.
- Add a browser test under existing isolated harness for the real `/cognition` route; test both authentication failures and empty data states, and test no unsafe POSTs.
- Only then consider a cross-link from Cognition card to a Development experiment; Cognition itself remains for viewing/understanding.
- Keep progressive drill-down: Nemotron only inside Thought Engine, Core under TASS, no repository tiles across the top.

**Acceptance state:** UI prototype source committed, sandbox JS/browser-fixture checks PASS, Go route and real owner-machine end-to-end NOT YET VERIFIED. This branch is not production accepted.