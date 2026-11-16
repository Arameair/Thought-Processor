# Owner acceptance — Cognition repair + first information junction (#82)

**Prepared:** 2026-11-05 (America/Chicago) by Claude. **Status: CANDIDATE — OWNER TEST NOT RUN.** Nothing here was installed on the owner's machine. Nothing was merged to `main`, `alpha` or `live`. No Identity grant, pulse, Autonomous setting, model or container of the owner's stack was changed.

## 1. Exact candidate

| Component | Repository · branch | Full SHA | PR |
|---|---|---|---|
| Core | `Arameair/Thought-Processor` · `experiment/claude-cognition-repair-junction-20261105` | see the PR head; the code commit is `d59e84e9986897ecbeec6f89da4e20b452936c63` | draft PR into #86's branch |
| Engine | `Arameair/Thought-engine` · `experiment/cognition-journal-projection-20261105` (PR #15, **unchanged**) | `d69d3917d1601d806d4b2ba2254693e1adb92b6b` | #15 |
| Identity | `Arameair/identity` · `experiment/junction-grant-lesson-metadata-20261105` | `7da7f7c5bbecc42d3a7e9316a2037243eda10b88` | draft PR into `experiment/skill-grant-lesson-workspace-20261105` |

These pins are in the candidate's `core/tass.sh` (`ENGINE_COMMIT_PIN`, `IDENTITY_COMMIT_PIN`). The Core candidate needs Engine #15: it reads the journal endpoint, and an older Engine is not a complete integration. Last reported owner stack (historical, re-check it): Core `60661aa…`, Engine `21b3d502…`, Identity `87dd9243…`.

## 2. What changed

| Area | What it does now |
|---|---|
| **P0-A** Identity correlation | Cognition never matches by display name. Engine data is shown only after the owner links the Identity entity (canonical ID) to one Engine entity, recorded with Identity's decision ID. Unlinked: no Thoughts or cycles; Engine entities are listed for review ("same name is a hint, not proof"). UNSCOPED cycles (no Engine entity) and other-entity cycles are counted, never shown. A rename keeps the link. A recreated Identity entity with the same name inherits nothing. One Engine entity can belong to one Identity entity only. LOCKED never contacts the Engine. |
| **P0-B** Notes | One saved brain snapshot supplies both the revision and the notes. New revisions record their Identity entity. Notes saved under another entity are hidden. Older notes are labeled `IDENTITY NOT RECORDED`. Notebook text cells get their own shelf. |
| **P0-C** Evidence | In-flight cycles show `PENDING` stages instead of "missing". Export includes Pending. Outcome wording is "executor-reported, not independently verified". Playback, View Logs, Step, Copy and Download remain GET-only. |
| **P1** Pulse strip | A read-only strip on Cognition: Engine pulse RUNNING/PAUSED/UNKNOWN, configured seconds, last real tick (ID, time, trigger, state, job), active job, queued/running counts, the Core's own 20 s tick, and waiting junctions. It is one authorized GET with no free text, and it never starts, steps or pauses anything. |
| **P2** Junction | Cell `core/tass/cells/lesson-metadata.md` is strict key/value data, never executed. Handler `lesson-metadata.v1` is compiled and read-only. Identity grant `junction.read:lesson-metadata` is checked at the point of use. TASS asks with `NEED lesson-metadata: <path>`, or the owner asks from Development. The Core's existing 20 s sync tick processes requests in its own goroutine, so the tick never waits. The next eligible conversation cycle receives the data once; if the cell or the lesson changed, the receipt says `NOT CONSUMED`. Receipts are durable (`junctions.json` + `junctions.jsonl`), idempotent and restart-safe. Statuses: DATA / AWAITING_APPROVAL / DENIED / UNAVAILABLE / TIMEOUT / EXPIRED / FAILED. |
| **P3** Cognition → Development | Junction card shows Requested → Authorized → Returned → Consumed → **EFFECT NOT MEASURED**. "Try adjustment in Development" opens the **Junction lab**: request, propose an alternative cell (`PROPOSED`, never activated), and compare both cells' output on the same lesson under one Identity decision. The comparison says what each cell returns, not how TASS used it. |

New data files: `cognition_links.json`, `junctions.json` and logs, plus a `identity_entity_id` field in brain revisions. Older Cores ignore all of these, so a rollback keeps working; after rollback the old Cognition goes back to name matching.

## 3. What was actually tested (sandbox, not the owner's machine)

| Test | Result |
|---|---|
| `go vet ./...` + `go test -race ./...` (Core) | PASS |
| New Core tests: same-name entities, rename, recreated entity, link refusals, LINK BROKEN, LOCKED → zero Engine calls, notes/one revision/cross-entity, PENDING, pulse paused/working/unavailable/no free text/GET-only, junction cell strictness, DATA→consumed once, awaiting→granted with no audit noise, missing/traversal/entity-changed/TIMEOUT/EXPIRED, restart retry, tick doesn't block, NOT CONSUMED on lesson change, no cross-identity data, TASS-reply request, proposal never activates | PASS |
| Identity `go test -race ./...` incl. new junction-grant test | PASS (local; no CI workflow ran on that branch) |
| GPT-6's `core/test/browser/cognition_go.py` (mocked Chromium) | 2/2 PASS |
| **`core/test/browser/cognition_junction_e2e.py`**: real Identity `7da7f7c` + real Engine `d69d391` + this Core, fixture model + fake GitHub, real Chrome | **11/11 PASS**: NOT LINKED by default; pulse PAUSED / every 20 s; link is the only POST; playback/logs/step zero POSTs; TASS's reply → junction; Core tick → Identity decision → DATA; a later real Engine cycle received it and the fixture reply used it; Cognition card shows all five stages; Development compare, active cell unchanged |
| GitHub CI on the Core branch | see §5 |
| Docker `./tass.sh update` build of this trio | **NOT RUN** (sandbox can't pull base images) |
| Owner machine / real Nemotron | **NOT RUN** |

## 4. Known defects and blockers

1. **#79 is not satisfied.** The updater's verified backup/restore hardening is separate, unfinished work. Do not install this candidate until #79 is accepted, or until you explicitly accept that risk.
2. **First-time link is required.** After installing, your Cognition shows no Engine history until you press **Link** on the Engine entity once. That is intended (fail closed).
3. The Engine still creates and finds its own entity by display name (`entityFor`); it never sees the Identity entity ID. The Core link makes Cognition safe, but a proper Engine-side fix is a follow-up for Engine (suggested on PR #15).
4. Whether real Nemotron will write `NEED lesson-metadata:` by itself is untested. Development's **Request through the junction** works without it.
5. The original cycle-era Identity decision is still `NOT RECORDED` in timelines; that join doesn't exist yet.

## 5. CI

- Core `core-tests` and `one-app-launcher-regression` run on each push to the branch: links are in the draft PR and in the issue #82 update.
- Engine PR #15 head `d69d391`: run 42096016070 success (unchanged by this work).

## 6. Morning: safe read-only inventory first (changes nothing)

```bash
cd ~/tass-one-app/core
./tass.sh status
git rev-parse HEAD
git status --short
```

Compare with §1 using full SHAs. Look at the draft PRs and their CI.

## 7. Proposed owner-approved update sequence (only after #79, or your explicit decision)

```bash
cd ~/tass-one-app/core
./tass.sh backup
git fetch origin experiment/claude-cognition-repair-junction-20261105:refs/remotes/origin/experiment/claude-cognition-repair-junction-20261105
./tass.sh check-update origin/experiment/claude-cognition-repair-junction-20261105
#   expect: engine d69d3917…  identity 7da7f7c5…  — stop if different
sudo -v
./tass.sh update <the full Candidate core SHA it printed>
./tass.sh status
```

Then unlock Identity with your usual passphrase. Do **not** start the pulse or turn on Autonomous for these tests.

## 8. Test checklist (PASS / FAIL / NOT RUN each)

1. **Cognition opens read-only:** `/cognition` shows the pulse strip with your real state (expect PAUSED unless you started it) and "Engine history link: NOT LINKED". No cycles are shown yet.
2. **Link:** press **Link to this Identity** on the Engine entity you recognize. Your recorded cycles and Thoughts appear. Note the linked Engine entity ID.
3. **Playback:** open a cycle, then use Play / Pause / Step / View Logs. Compare the cycle ID, stage times and byte offsets with the Engine's `cycles.jsonl` on your machine. An in-progress cycle shows PENDING, not missing.
4. **Evidence:** Copy Evidence and Download Markdown/JSON. Check them for private text, tokens and paths. Confirm that viewing created no new cycle.
5. **Notes:** a note you saved in Development shows under Owner notes with the right revision.
6. **Development reference:** "Try adjustment" opens `/dev` with the UNEXECUTED PROPOSAL banner. No graph save happens.
7. **Junction:** on Identity's page, under *Skills and junctions TASS may use*, enable **junction.lesson-metadata**. In Development → **Junction lab**, choose *My work pulse* and press **Request through the junction**. Within about 20 s the receipt shows **DATA** with an Identity decision. Then say something to TASS in Talk; the receipt shows **consumed by cycle …**. In Cognition → Information junctions, all five stages are shown.
8. **Denied path:** revoke the junction on Identity's page, then request again. The receipt shows **AWAITING_APPROVAL** with nothing read, and later **EXPIRED** after 15 min.
9. **Alternative cell:** in the Junction lab, save a proposal (for example `fields: title, steps, sha`) and press Compare. Both outputs are shown, "EFFECT … NOT MEASURED" is stated, and the ACTIVE cell is unchanged.
10. **Health:** `./tass.sh status` READY; lessons, notes, skill receipts and Identity are unchanged.

Report format: the one in `CLAUDE_COGNITION_JUNCTION_IMPLEMENTATION_HANDOFF_20261105.md` §7.

## 9. Recovery

Use the updater's own path: `./tass.sh rollback` returns to the previous deployed version. `./tass.sh rollback --restore-data <backup file>` also restores the data from the backup made before the update; anything after that backup is lost. These commands exist, but #79 says they are not yet *verified* to its standard. Recheck all three SHAs and your data afterward.
