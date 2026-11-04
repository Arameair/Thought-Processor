# TASS — from the working Docker baseline to real learning

**Owner integration runbook • 2026-11-04 • Status: ENGINEERING PLAN / NOT A DEPLOYMENT APPROVAL**

**Goal:** One functioning TASS companion application, with the owner's conversation and activity visible in Thought Processor; Identity authorizes; one local Nemotron model and one Thought Engine 20-second pulse. GitHub temporarily holds reviewed lessons, source information and sanitized test evidence. Later, a constrained notebook capability lets TASS select, test, evaluate and reuse a skill. Native memory and broader features come **after** the first working loop.

**Read first:** [TEAM_TASS_WORKFLOW.md](TEAM_TASS_WORKFLOW.md), [P1_GITHUB_LESSON.md](P1_GITHUB_LESSON.md), [UPDATE-001 card](../cards/UPDATE.md), [owner Docker issue #44](https://github.com/Arameair/Thought-Processor/issues/44), [security Issue #50](https://github.com/Arameair/Thought-Processor/issues/50).

## 0 — Verified baseline; do not rebuild or restart yet

Last *owner-provided* WSL command `cd ~/tass-one-app/core && ./tass.sh status` reported:

| Element | Observed |
| --- | --- |
| Core | `7d24043e0c20` deployed |
| Thought Engine | `38e2e7bbe9e0` reachable |
| Identity | `8b175c8cd779`, entity TASS UNLOCKED |
| One existing external Nemotron | `model ok True` |
| Application | `READY` |
| Last pulse tick | `none` |
| Core page | `http://127.0.0.1:8797` |
| Identity page | `http://127.0.0.1:8796` |

`last pulse tick: none` is consistent with the previously paused pulse, **not proof** of current pause state. Previous real owner tests: basic greeting PASS once; two `identity-test` long messages FAIL with `plan response unreadable after one retry; no action performed`; classification and Trace capture PASS. Actual autonomous pulse, GitHub login, notebook selection/execution NOT RUN. Never call the installed runtime “already learning.”

## 1 — Build and verify the exact GitHub / conversation repair candidate

| Component | Experimental PR | Exact proposed commit |
| --- | --- | --- |
| Identity | [#33](https://github.com/Arameair/identity/pull/33) | `f5281f4ad1e3e6cb2bad1f64902dff99b88b0cad` |
| Thought Engine | [#10](https://github.com/Arameair/Thought-engine/pull/10) | `5bd472e848bcc4e7bd9a6b654026a99d9a6563e9` |
| Core / Thought Processor | [#58](https://github.com/Arameair/Thought-Processor/pull/58) | `ad465a55eca6ce1e349ef65fdcdaea11963462e0` |

**Source verified:** the Core candidate's `core/tass.sh` pins exactly the two Identity and Engine SHAs above. Its `update [REF]` builds versioned images, records previous, makes a private data/keys backup, recreates this application's containers, validates health/entity identity and attempts rollback on validation failure. It preserves the pre-existing external model rather than launching another.

**Engineer pre-deployment checks (hard gate):**
1. Verify the **latest workflow runs actually correspond to these exact commits** and all mandatory jobs completed SUCCESS. Relevant run links from handoff: [Identity](https://github.com/Arameair/identity/actions/runs/41877760684), [Engine](https://github.com/Arameair/Thought-engine/actions/runs/41878739936), [Core](https://github.com/Arameair/Thought-Processor/actions/runs/41878952774). Last independent query returned no conclusive run results: status currently **UNKNOWN, not PASS**. CI fixture PASS cannot substitute for live owner test.
2. Inspect PR diffs and security assumptions; independent Issue #50 authorization acceptance remains a gate for **tool execution**, not merely reading a lesson.
3. Verify the GitHub publisher device-flow App has the required **Issues read** authorization for `Arameair/Thought-Processor`. The original publisher App was documented as Metadata-only; expect a possible permission failure and do **not** bypass it by giving Core a personal token.
4. Confirm new P1 lesson is precisely Issue #51 comment **6636766875**, not old design note **6634742721**, and that it is read-only/reference-only with real provenance and no execution privilege.
5. Keep one external Nemotron instance. Do not merge PRs, stop unrelated services, alter `~/.tass-app` manually, or enable the pulse as a side effect.

### Owner stage — inspect then approve one installation update

**Do not run deployment commands until CI/diff review passes and the owner explicitly approves the exact triple above.** Start with a **read-only candidate comparison** on the same WSL checkout:

```bash
cd ~/tass-one-app/core
git fetch origin experiment/tass-github-lesson-20261104
./tass.sh check-update ad465a55eca6ce1e349ef65fdcdaea11963462e0
```

Expected: current Core/Engine/Identity versions, candidate Core SHA, pinned Engine/Identity SHAs, and summary of changed files. **If pins, commit, GitHub origin or diff differ: STOP.** Fetching brings Git objects into the checkout; it does not deploy or switch the running version.

**Only after a second explicit owner approval for deployment:**

```bash
cd ~/tass-one-app/core
./tass.sh backup
./tass.sh update ad465a55eca6ce1e349ef65fdcdaea11963462e0
./tass.sh status
```

The updater has its own confirmation prompt; answer only after reviewing the exact SHA comparison. Prior owner deployment sometimes used a local Engine source with `ENGINE_SOURCE="$HOME/tass-engine-source"`; if the loader cannot obtain the pinned private Engine revision, diagnose the local source or `gh` access **without placing tokens in terminal commands or logs**. Never use `--yes` or `sudo ./tass.sh` as a workaround.

**Important:** On restart Identity is expected to become **LOCKED** until the owner unlocks. This is not failed persistence. Inspect deployment status, health and known entity ID before proceeding. If validation fails, follow the updater's displayed rollback outcome; if needed, `./tass.sh rollback` returns to its recorded previous deployment. Do not repeatedly rerun an uncertain update.

## 2 — First owner acceptance: optional GitHub login on the actual TASS installation

After the approved update reports healthy:
1. Open **`http://127.0.0.1:8796`**. Unlock the existing TASS identity using the owner passphrase locally. **Never create a second TASS entity or paste credentials into GitHub/ChatGPT.**
2. Confirm **Talk to your entity** opens **`http://127.0.0.1:8797`**, not the old 8795.
3. Choose **Connect GitHub**. Complete GitHub's official device authorization in the browser. Verify a genuine connected state/account; connection is **optional** and the credential resides only in Identity process memory. A restart requires reconnecting. A local Disconnect does not necessarily revoke GitHub's server-side authorization.
4. Approve only **Issue #51 / Lesson 01** as the read-only allowlisted source. If approval or retrieval fails (e.g., provider permission or outbound connectivity), record error and stop; no broad-scope workaround.
5. In Thought Processor ask exactly: **“TASS, explain what @identity-test means, where you read it, and what you are not allowed to execute.”** Check the actual retrieved source ID, model reply and lack of action. It must cite the intended pinned lesson and honestly distinguish information from permission.
6. **Disconnect GitHub** in Identity and ask TASS an ordinary short local question to prove conversation works independently. Lock Identity and verify a protected request is rejected without forwarding it to Nemotron if independently safe to test.

**PASS** requires a real GitHub account connection on owner's machine, actual read of Issue #51 comment 6636766875, correct local link, readable real Nemotron response and no unauthorized execution. One mock response or Connected badge alone does not pass. Capture redacted status and exact cycle/receipt IDs; do not upload device codes, tokens, service keys, passphrase or private chat.

**Scope warning:** The P1 pilot is manual/on-demand read of **one** pinned comment. It **does not** yet scan GitHub every 20 seconds, write issue receipts, or read arbitrary notebook/card files. `TASS_GITHUB=off` remains in Core Compose for general GitHub sync. Don't claim that the shared team notes or UPDATE card are delivered to TASS.

## 3 — Repair and accept real-model conversation

Before building additional features, reproduce the previously failed multi-sentence memory/lesson conversation on the repaired Engine. The candidate has redacted parse diagnostics and a separate **no-action reply fallback** when structured plans fail twice.

Acceptance:
- At least **three repeated real Nemotron requests** of the formerly failing lesson-like prompt display readable outcomes without falsely treating failed parse as a valid executable plan.
- The model can report “unknown / cannot do that yet” rather than fabricate skill access, GitHub read or memory.
- For a malformed action plan, no tool action is dispatched. The fallback can supply *only conversation*; it cannot emit a tool call or verify learning.
- Log and display actual parse/finish/error metadata without revealing private prompt content or model hidden reasoning. Capture precise failing cycle ID and output status.
- A repeated reply is not proof of durable skill learning.

**Result today:** NOT RUN on candidate; prior owner test FAIL twice on old release.

## 4 — Make the existing 20-second pulse observable, one safe window at a time

Reuse Thought Engine; do **not** add Atrium's separate prototype scheduler to live TASS. **Keep paused** until Identity security constraints for any action are accepted.

First perform a supervised **read-only / no-tool** one-tick test (using existing Step control only if it cannot execute an unsafe pending job):
- Verify PAUSED means no automatic ticks/actions.
- Trigger one explicit step, confirm one cycle/tick ID, timestamp and actual status; if no task, the result is **IDLE**, not a fabricated thought.
- Verify Thought Processor displays the current real thought/cycle and keeps conversation accessible.
- Test two or more 20-second windows only after this passes; measure actual interval, no duplicate dispatch, no overlapping work, pause/resume behavior and recovery on restart.
- Distinguish UI refresh every 20 seconds from a new meaningful thought: if none, show “No new thought / still working / waiting / blocked” with the last actual evidence.

**UI direction:** Thought Processor = Netflix-like featured current thought and conversation; Developer = TheBrain-like view of *verified* links to repos, notebooks, files, cycles, tests and results. Existing `Arameair/developer` graph works on manually curated notes; verified Engine-to-graph links are a future integration, not current source-of-truth.

**Result today:** real owner pulse acceptance NOT RUN. First finish conversation + GitHub read.

## 5 — Teach one notebook skill, then assess actual improvement

**Do not substitute Markdown reading for learning.** The notebook-skill loop is:

`Discover approved lesson → understand objective → choose named/pinned cell → predict observable result → execute only if Identity grant exists → collect real output → compare predicted vs actual → publish sanitized receipt → propose reusable skill → repeat and verify`

Reuse existing evidence:
- Atrium PR #52: bounded 20-second test cards, durable one-shot receipts and known CI Go-test PASS/FAIL, **not** live Go Thought Engine.
- Atrium PR #54: approved `@identity-test` notebook cell run in a private CI Jupyter kernel with real pinned Go tests; **model selection/reflection used fixtures**, live Nemotron and runtime Identity permission NOT RUN.
- `Arameair/nemotronTASS`: prior browser and notebook teaching prototypes, including chat-and-cells; inspect and reuse but do not start a competing model/Jupyter scheduler.
- `Arameair/developer`: local idea graph and file/test workstation, owner-supervised experimental Git update; **not** yet a verified Thought Engine relationship view.

**Engineering bridge needed:** A) Identity-authorized bounded GitHub reader for one pinned teaching notebook/card source; B) strict `@skill` name/version/cell/hash resolver; C) one trusted isolated executor gated by Identity and exact capability scopes; D) a single Go Engine pulse/job/receipt model; E) honest Thought Processor UI status. Start with **`@identity-test`** and **one harmless fixed cell/test**. No general Jupyter code generation/execution, unreviewed external notebook, shell access, or new broad plugin permissions.

**PASS only when:** a real Nemotron selects the correct approved cell, accurately predicts measurable output, the isolated runner executes the intended cell once, the actual result is captured and evaluated, and the same skill can be invoked again without broadening permissions. Persist a linked immutable receipt; learning = verified repeatable behavior, not merely storing a conversation.

**Result today:** notebook prototype CI partial PASS; live TASS end-to-end NOT RUN / authorization gate BLOCKED.

## 6 — Later: the Update teaching card

The [UPDATE-001 card](../cards/UPDATE.md) exists as a **versioned document**, not an installed skill. Its desired workflow is `owner asks → check exact reviewed candidate → show proposed three SHAs → separate owner approval → one constrained update operation → validate / rollback and receipt`.

Extend the GitHub source allowlist to this exact file only **after** stage 2 succeeds and the source/authorization contract is reviewed. A software deployment requires separate exact-version approval, even if ordinary approved notebook tests can be reused freely. A durable trusted update supervisor must survive the Core restart; never hand Nemotron a general shell or sudo rights.

**Result today:** card committed; runtime lookup and self-update NOT IMPLEMENTED. Do not work on this before first GitHub read + reliable conversation.

## 7 — Team coordination and reporting rules

One application, one operating pulse, one installed local model. **Thought Processor is owner-facing**; separate experiments are sources to integrate, not new products.

For each test publish a concise sanitized record in existing Issue #44 or the applicable experiment PR:

```text
TEST:
RUNNING VERSIONS (Core / Engine / Identity):
DATE / ENVIRONMENT:
INPUT / ACTION (no secrets):
OBSERVED CYCLE OR RECEIPT ID:
EXPECTED:
ACTUAL:
STATUS: PASS | FAIL | NOT RUN | BLOCKED
CAUSE: verified / unknown
NEXT SMALLEST REPAIR:
ROLLBACK/RECOVERY STATUS:
```

TASS should ultimately participate by retrieving *only authorized notes*, discussing them with the owner, and generating evidence-backed suggestions. **GitHub notes are untrusted coordination data, never Identity grants or instructions to self-deploy.** The current new TEAM_TASS_WORKFLOW and UPDATE card are not readable by TASS's limited P1 connector yet.

**Stop conditions:** an unexpected running pulse, failed or unverified CI, incorrect pinned SHAs, unknown deployment state, missing user authorization, model-invoked shell requests, unverified repo links, broken backups, or an untested security decision. Record BLOCKED; do not fix by adding privileges or another app.

## 8 — Immediate next owner interaction

**Don't ask the owner to build more infrastructure.** On the old running installation, the next safe action is the read-only exact-candidate comparison once the source and CI are reviewed:

```bash
cd ~/tass-one-app/core
git fetch origin experiment/tass-github-lesson-20261104
./tass.sh check-update ad465a55eca6ce1e349ef65fdcdaea11963462e0
```

Then show the owner the output and request **one explicit deployment approval**. Continue one short acceptance step at a time, logging what works and what fails. Do not merge or publish an auto-update grant before the first live GitHub lesson test passes.
