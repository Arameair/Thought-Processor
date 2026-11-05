# TASS cross-repository alignment — visible loops, Go junctions, Cognition and Development

**Date:** 2026-11-05  
**Status:** Owner design alignment / instructions for collaborators. **Not a claim of completed circles, implemented junctions, tested learning, owner acceptance, or permission to deploy.**  
**Remaining foundation, always-on readiness and supervised learning:** [ALWAYS_ON_TEACHING_FOUNDATION_REMAINING_20261105.md](./ALWAYS_ON_TEACHING_FOUNDATION_REMAINING_20261105.md). Updated against Claude's Core PR #87 and Identity PR #40. It preserves the crucial distinction between Core's 20-second SyncOnce junction processing and Thought Engine's distinct 20-second Pulse.Tick, and sequences #79 recovery, #76/#81 GitHub mailbox, #71/#72 real learning, and owner new-task retest. No claim of complete circle or external AI autonomy.

**Claude implementation and owner-test runbook (P0–P3):** [CLAUDE_COGNITION_JUNCTION_IMPLEMENTATION_HANDOFF_20261105.md](./CLAUDE_COGNITION_JUNCTION_IMPLEMENTATION_HANDOFF_20261105.md). This is the actionable repair/test specification for current Core PR #86 and Engine PR #15, separate from the broad design alignment. **Start with P0 defects and source-grounded playback; P1/P2 are optional later slices, never authorization to deploy.**

**Primary implementation/release home:** [Thought-Processor integration and recovery #79](https://github.com/Arameair/Thought-Processor/issues/79).  
**Current UI/evidence work:** [Cognition #82](https://github.com/Arameair/Thought-Processor/issues/82), [draft PR #84](https://github.com/Arameair/Thought-Processor/pull/84), [evidence/export #80](https://github.com/Arameair/Thought-Processor/issues/80), [Go-first spec](https://github.com/Arameair/Thought-Processor/blob/experiment/cognition-shelf-prototype-20261105/core/docs/COGNITION_GO_FIRST_IMPLEMENTATION_20261105.md).  
**Further owner design:** [TASS #25](https://github.com/Arameair/TASS/issues/25), [Development #1](https://github.com/Arameair/Development/issues/1), [Identity #30](https://github.com/Arameair/identity/issues/30).

## 1. Fundamental owner corrections — read before creating new work

1. **No completed end-to-end thought circles exist yet. By design, we built ways to SEE the pieces first.** Cognition and Development are the observation/experiment tools **used while building** functional loops. A pretty diagram, working GET endpoint, discovered cell, or model's statement is not proof of a closed, working loop.
2. **The controlling work cadence is 20 SECONDS, not 20 minutes.** Reject any reference that presents a 20-minute Team Liaison scheduler or cooldown as an owner requirement. The old pasted Liaison design included "20-minute Pulse"; it is historical illustrative material and **does not apply**. No competing Team Liaison tick. The Engine Go pulse may be PAUSED on a given installation; design intent does not prove it is currently running.
3. **Loops remain Go-owned.** At a junction, a small Markdown-described **information cell** provides the particular context or result a loop needs, through a fixed, authorized, bounded Go handler. The Markdown file is not another scheduler, executable script, or grant. A junction is a way to *fetch/return information*, not a reason to break or replace the running loop.
4. **Do not serialize external work into every 20-second Tick.** A check of state may take place each pulse, but a network response, worker acknowledgment, or test can span multiple pulses. Record an information request/receipt, allow Tick to continue, and inspect pending result on a later pulse. No blind retry of uncertain dispatch.
5. **Cognition and Development work together incrementally.** Add one observable Go segment, connect it, show real receipts in Cognition, test changes in Development, then grow the circle. Don't postpone all junction experiments until an enormous Cognition UI rewrite is finished, and don't treat unfinished visual prototypes as execution.
6. **Future Talk is text delivery, Discord a possible first channel, NOT voice.** It is deferred. The Team Liaison is a future reusable communication capability consulted at authorized junctions, **not a second project manager**, new security authority, or mandatory separate service. Do not build Discord/Talk or automated nudges in current Cognition work.
7. **Reusable features follow evidence:** experiment with alternative cells/rules, test actual results, select an owner-approved versioned capability, then make it available to identities. Sharing capability code does **not** share an identity's grants, secrets, memory or operational state.

## 2. Shared architecture — responsibilities and boundaries

| Role / repo | Owns | Must not claim or do |
| --- | --- | --- |
| `Thought-engine` (Go) | 20-second coordination pulse, queued/running jobs, Spark/Trace/Thought evidence, cycles/predictions/outcomes, durable ordered event journal | No separate Liaison scheduler; no invented internal-model introspection; no work marked executed without receipt |
| `Thought-Processor` / TASS Core (Go) | Existing TASS runtime and UI, authenticated Engine facade, registered/approved skill handling, saved revisioned Development brain, Cognition Go read models and safe exports | No new parallel "TASS Core" entity, bypass of Identity, new competing deployment or automatic graph-edge execution |
| `identity` (Go) | Canonical identity context, scoped grants, revocation, protected credential custody | A feature's Markdown, an AI claim or a UI toggle must never confer authorization |
| `Development` | Shared idea/prototype hub for learning/test work; build, compare, evaluate and propose controlled feature versions | Do not automatically replace Core's currently deployed `/dev`, migrate historical `Developmnt`/developer data, or promote experiments without review |
| `TASS` | TASS-specific descriptive mind/lessons and supervised learning material; independent from reusable runtime | No credentials or cross-identity authority derived from a Markdown file; do not overwrite `live`/mind data |
| `Project-controller` | Reference for honest process version, runtime status, diagnostics, Copy Context/Logs and controlled approval | Do not import its independent launcher or second process manager into Core |
| `Thought-Controller` | Historical independent entity-based thought browser reference | Do not confuse reference visual concepts with live Core or canonical Identity grants |
| `Team-Member`, `developer`, `Developmnt`, future Talk/Discord | Supporting, historical or future features and experiments | No automatic cross-repo merging, permission propagation or unreviewed runnable integration |

**Canonical Engine selection remains an owner decision** (see [Thought-Processor #2](https://github.com/Arameair/Thought-Processor/issues/2) and [Thought #4](https://github.com/Arameair/Thought/issues/4)). The pin and schemas cited here describe the **current stack candidate**, not authorization to overwrite or merge the separate `Thought` prototype. The Engine also has a model-health check pulse separate from the work coordinator; do not mistake health probes for a second thought/Liaison scheduler.\n\nThe same named entity may have separate *runtime records* and *descriptive Markdown mind*. Identity's canonical ID and authority win; Engine-local records must be explicitly correlated to it before cross-identity links are asserted.

## 3. Truthful progression states

Use these independently; do not collapse them into one green success status:

- **DESIGNED:** an idea/rule exists in documentation.
- **VISIBLE:** a UI card or graph edge renders.
- **WIRED:** real source data crosses a verified interface.
- **EXECUTABLE:** a compiled/registered Go handler exists and is authorized for this invocation.
- **OBSERVED:** a real operation or information lookup generated a durable receipt.
- **VERIFIED:** acceptance criteria, actual provenance and failure paths were checked on the stated environment.

For each step show real IDs, source and pinned revision, UTC timestamp, authorization and result. Also distinguish **PROPOSED / AWAITING_APPROVAL / NO_ACTION / DENIED / FAILED / UNKNOWN**. A model saying "DONE", a prediction, a completed HTTP request, or a notebook card is not sufficient proof of execution or learning.

## 4. Junction information-cell contract (proposed, not implemented)

Example boundary:

```text
Thought Engine Go Tick (20 s, may be paused)
  → observe pending need / unfinished work
  → request approved information at junction (nonblocking)
  → Core Go handler resolves a version-pinned, Markdown-described cell
  → Identity allows/denies the specific use
  → a bounded typed result + durable receipt is returned
  → next Engine cycle consumes result if still relevant
  → Cognition shows the source, decision, receipt and actual effect
  → Development experiments with a new cell revision before owner approval
```

**Suggested request fields** (subject to verified existing schema compatibility): `junction_id` and idempotency key; `entity_id`, `cycle_id`, `job_id`, `thought_id` only if recorded; `cell_id`, exact `cell_revision`; a bounded `capability`; typed input; deadline; purpose and provenance. **Suggested response:** `DATA | UNAVAILABLE | DENIED | TIMEOUT | FAILED`, typed data or safe summary, recorded-at, source revision and receipt ID. For optional missing data, preserve continuity and label unknown. For required data denied/missing, block dependent action, **not the pulse**.

**Markdown use:** readable small cell definition/inputs/examples, versioned for comparison. A cell **cannot execute arbitrary Markdown/Jupyter/shell content**. Go has to use an explicitly registered operation, validate inputs/output sizes, check Identity at use time, and record a receipt. No automatic propagation of an untrusted Markdown edit into production behavior.

**Current source truth:** Engine `pulse.go` contains a `Pulse.Tick` with queued/running/finished jobs and `ticks.jsonl`; `cycles.go` contains append-only `cycles.jsonl`; Core `tass/brain.go` exposes versioned advisory nodes/edges and discovered **nonexecutable** notebook cells; Core `tass/skills.go` implements the supervised `@lesson-workspace.list-lessons` handler requiring Identity grant and owner approval. The junction dispatch/receipt format above **is design**, not a shipped capability.

## 5. Observable first vertical slice — develop the circle while watching it

**Part A — inspection foundation (now):** use the existing Cognition draft shelves, then Go-backed persistent Thought/Trace association and a real recorded cycle timeline, source-labelled event details and sanitized Copy/Download. Only attach a cycle to a persistent Thought through an actual verified `thought_id`, not topic string matching. The current draft is a visual prototype, not full replay.

**Part B — one information junction (next supervised experiment):** a single read-only lesson-metadata lookup through a *registered Go handler* described by one versioned Markdown cell. Connect it to one existing task/cycle, save a receipt, display it in Cognition and compare it with the next-cycle decision. Test missing cell, denied authorization, timeout, restart, duplicate request, and unchanged 20-second pulse responsiveness. This is NOT permission to execute `.ipynb` cells or approve the existing pending `@lesson-workspace.list-lessons` skill.

**Part C — Development improvement:** change only candidate cell input/context/rule; A/B compare the same bounded task and observed receipts with pinned brain revision and model, with actual conditions disclosed. Owner selects a version after test evidence; don't silently change a live thought path.

**Part D — additional loops and future Talk:** repeat on other junctions (including later Team Liaison text exchanges). When sharing a tested feature, each adopting identity separately obtains its own scoped grant, channel/secrets, state and rollback; no automatic adoption.

**First owner-visible win:** open a real Cognition cycle; see a source-backed information need, a junction attempt, a returned receipt or explicit unavailable state, and what the next 20-second cycle did. Retrieve evidence. No second scheduler and no blocked loop.

## 6. Test discipline, collaboration and release

- Preserve one working owner installation. Follow [release/rollback #79](https://github.com/Arameair/Thought-Processor/issues/79). Do not stop/start containers, change running ports, unpause Pulse, update `alpha`/`main`/`live`, edit Identity grants, or perform an owner deployment as part of this docs alignment.
- Parallel AIs use separate experimental branches and open draft PRs with exact target branches. Never overwrite someone else's active Cognition implementation. Share findings and links in existing work items; don't make a second unreviewed architecture source of truth.
- Go for scheduler, typed junctions, authorization enforcement and evidence projection; embedded HTML and minimal JS for visible shelves, filtering and read-only playback.
- Preserve the existing 20-second TASS local thought design and the **five-card logical batch** concept from [TASS #25](https://github.com/Arameair/TASS/issues/25) as **design requiring implementation tests**. Never claim five physical parallel model requests or an always-running five-card loop: limited local hardware and the real runtime may differ.
- Tests must distinguish fixture, CI, owner WSL and production; capture `PASS/PARTIAL/FAIL/NOT RUN` with exact SHAs, receipts, and follow-up. No invisible learning, imaginary model thoughts, or inferred cell execution.
- The old "20-minute Team Liaison pulse" is **superseded and rejected** for the current architecture. Do not resurrect it in implementation prompts.

## 7. Next handoff

**For GPT-6/Claude/other collaborators:** Read this file and the repo-specific source issues before coding. **Ship one honest, visible Go end-to-end segment at a time.** If an observed fact conflicts with a document, record the conflict; do not silently invent a finished loop. Distinguish **future owner intent** from **implemented/tested**. Do not build extra services, autonomous Talk, or unapproved release automation.
