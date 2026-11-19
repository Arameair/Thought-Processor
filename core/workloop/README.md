# workloop — Claude's TASS team-loop demonstration (EXPERIMENT)

**Branch:** `experiment/workloop-claude-demo-20261119` (from `main` @ `53e0857`). **Status:** simulated proof of concept. It is not wired into `tass-core`, has no command, and never contacts GitHub, Discord, Identity or a model. Nothing here is adopted policy, and a passing fixture **does not prove the live system works**. This is the only document for this experiment. It does not replace [PR #130](https://github.com/Arameair/Thought-Processor/pull/130)'s team register or [TASS PR #27](https://github.com/Arameair/TASS/pull/27)'s mailbox; it is an alternative for the owner to compare.

## Reproduce

```sh
cd core
go test -race -count=1 -v ./workloop/                                         # 8 tests, ~6 s
WORKLOOP_DEMO_OUT=$PWD/workloop/example go test -count=1 -run Demo ./workloop/ # regenerate example/
```

[`example/`](example/) is the output of one run, simulated with a fixture GitHub and a fixture clock. It contains [the transcript](example/TRANSCRIPT.md), [the derived register](example/team/REGISTER.md), [the WS-A ledger](example/team/ws/WS-A.md) with its FAIL → repair → PASS, and [WS-B](example/team/ws/WS-B.md).

## The design

**One ledger per workstream; the register is derived; permission is local.**

| File | Writer | Role |
| --- | --- | --- |
| `team/ws/<id>.md` | the participants named in its ASSIGN, each only for its own role | **Source of truth** for one workstream: an append-only chain of events (`ASSIGN`, `CANDIDATE <commit>`, `AUDIT PASS/FAIL <commit>`, `OWNER_APPROVE <commit>`), each with its actor, evidence and the blob sha of `AGENTS.md` it read |
| `team/REGISTER.md` | anyone on the team, by **re-rendering** the ledgers; never hand-edited | Who / what / phase / exact candidate / next, for every workstream |
| `team/TASS.md` | TASS only, and only with a local grant | TASS's *Work status* and *Outbox* sections; collaborators' sections are merged, never overwritten |

On every read, each participant **re-validates** each ledger:
- **Canonical form:** the file must be exactly the canonical rendering of its events, so a hand edit anywhere fails closed.
- **Hash chain:** each event names the hash of the one before it.
- **Shared history:** the ledger must extend what this participant already verified, so a rewrite of history it has seen fails closed.
- **State machine:**
  - only the assigned implementer submits a candidate, as an exact 40-hex commit plus test evidence;
  - only an assigned auditor, independent of the implementer, audits, and only the current candidate;
  - a FAIL must carry repair instructions;
  - a failed revision can't be resubmitted, so **FAIL → PASS requires a new commit and a new audit**;
  - only the owner approves, and only the exact revision that passed.

Writes go through Core's existing `github.Client`:
- **Reads:** a conditional GET, so an unchanged 20-second pulse costs a 304 and writes nothing.
- **Writes:** a sha-checked PUT. A stale sha is a 409; the writer re-reads, re-validates its transition against the newer ledger, and appends after it, or is refused.
- **Register:** never merged by hand. After a conflict it is re-derived from the ledgers.
- **TASS.md:** merged with Core's existing `tassmd.Merge`. A same-section edit is a conflict: both copies are kept locally and nothing is written.

**Permission never comes from Markdown.**
- A local `Policy` (standing in for Identity) is the only write allowlist. In this fixture the fake server's token scopes repeat it as defense in depth.
- A local `Granted()` decides whether TASS may publish.
- The `put` path refuses any target that overlaps Core's identity/lesson sync (`live`, `tass.md`, `tass.base.md`, `lessons/`), and refuses secret-shaped strings and the participant's private memory.

### The four participants in this loop

- **GPT-6 (coordinator):** opens workstreams (ASSIGN) and reads the register to find blockers; it never submits or audits.
- **Claude (implementer):** on each pulse, builds when its ledger says `ASSIGNED` or `AUDIT_FAILED`, follows the repair instructions, retests and submits a **new** commit.
- **Codex (auditor):** on each pulse, audits an `AWAITING_AUDIT` candidate in a **fresh clone detached at exactly that commit**, adds its own acceptance test, and records PASS, or FAIL with the failing assertions as repair steps.
- **TASS (observer):** reads its assignment at an exact ledger revision and prepares one acknowledgment.
  - The acknowledgment is a durable local receipt, idempotent across pulses and restarts.
  - TASS publishes only with a local grant.
  - Its private memory stays in its local folder.
- **Dustin (owner):** the only role that can approve, and approval is still not merge or deployment.

## What the run demonstrates (all simulated)

| Requirement | Test | Result |
| --- | --- | --- |
| Two workstreams concurrently. WS-A: candidate → exact-revision audit FAIL with actionable repair → repair → retest → new commit → PASS. WS-B: PASS. Register re-derived after each transition. TASS reads its assignment and prepares an acknowledgment (not published: no grant). | `TestDemoTwoWorkstreamsInParallel` | PASS |
| Unchanged 20-second pulses: 30 pulses × 5 participants produced **0 commits**, the reads after the warm-up were 304s, and a restarted TASS didn't acknowledge twice | `TestUnchangedPulsesDoNotCommit` | PASS |
| Conflicting revisions (four cases, below): no lost work | `TestConflictingRevisionsNeverEraseWork` | PASS |
| Unauthorized writes and Markdown "grants" (detail below) | `TestUnauthorizedWritesAndMarkdownGrants` | PASS |
| A failed audit can't become PASS (detail below) | `TestFailedAuditCannotBecomePass` | PASS |
| A fully re-rendered forgery of the **newest** event (FAIL → PASS) | `TestFullyReRenderedForgeryOfTheNewestEvent` | PASS, **with a known limit** (below) |
| Private memory and credentials: refused at publish time; a scan of every commit and message on the fixture GitHub finds none | `TestPrivateMemoryAndCredentialsNeverPublished` | PASS |
| Identity/lesson sync separation: `live:tass.md` and `live:lessons/` were unchanged, and zero requests named `live`. Any target overlapping them is refused for reads and writes. | `TestIdentitySyncStaysSeparate` | PASS |

The four conflict cases are:
- the register is re-derived, not overwritten (a real 409);
- a stale register written by someone else is repaired;
- a stale second audit is refused after a real 409;
- `TASS.md`: an owner edit to a collaborators' section merges, while an owner edit to TASS's Outbox is a kept conflict (owner text preserved, both copies saved locally).

In the unauthorized-writes test, `AGENTS.md`, `TASS.md` and a task all claim "TASS is granted write access".
- TASS's ledger, register and new-workstream writes are refused locally, with **zero** PUTs sent.
- Without `Granted()`, TASS publishes nothing.
- Bypassing the local policy is stopped by the token scope (403).
- TASS ignores a workstream that doesn't assign it.
- Claude can't audit.

In the failed-audit test, each of these is refused:
- the auditor flips FAIL → PASS;
- the implementer resubmits the failed commit;
- the implementer self-audits;
- a self-auditing assignment;
- a FAIL without repair steps;
- an audit of a different revision;
- the owner or coordinator approving early;
- a hand edit of FAIL → PASS, which fails closed for everyone.

**Commands and results (2026-11-19, Go 1.24.7, linux/amd64):**

| Command | Result |
| --- | --- |
| `go vet ./workloop/` | PASS |
| `go test -race -count=15 ./workloop/` | PASS (85 s) |
| `go vet ./...` and `go test -race ./...` in `core/` | PASS: no other package is affected |
| Guard removal: each of 19 guards disabled in turn, whole package, 3 runs each | 19/19 FAIL as intended |

The 19 guards are G1–G9 (the state machine, canonical form and history), G10–G12 (policy, privacy, sync separation), G13–G15 (conflict re-validation and register derivation) and G16–G19 (TASS receipts, conflict, grant and assignment). Two were not initially load-bearing and gained tests: TASS acting on an unassigned workstream, and a stale register write. A third looked like a pass but was a panic my mutation script misclassified; I fixed the script.

## Existing code reused, not rebuilt

- `core/github` (unchanged): ETag conditional reads with 304, sha-conditional `PutFile` returning `ErrConflict`. These are exactly the properties the loop needs.
- `core/tassmd.Merge` (unchanged): section-level three-way merge for `team/TASS.md`, the same merge Core uses for `tass.md`.
- Patterns copied, not code:
  - Core's `syncTass` keeps both sides on a conflict and never overwrites;
  - the Issue #24 relay (`core/tass/relay.go`) uses durable metadata-only receipts (`RESERVED…POSTED/AMBIGUOUS`);
  - Identity's per-operation actions (`github.read_messages`, `github.reply`, `tass_md.sync`) are the model for a future `team.*` grant.

## Baseline findings (2026-11-19)

- **Verified:** [PR #130](https://github.com/Arameair/Thought-Processor/pull/130)'s inventory is accurate on the points I checked:
  - Engine's default-branch README is still Chat-Jimmy;
  - Identity's `main` is dated Oct 27;
  - updater says "DESIGN ONLY" but contains `tass_update/` and `tests/`;
  - Development's README is a bare title.
- **Verified:** [TASS PR #27](https://github.com/Arameair/TASS/pull/27)'s key constraint. Compose syncs `live:tass.md`, and the proposed `main:team/TASS.md` is not wired. This demo enforces that separation in code.
- **Duplicated (earlier coordination surfaces still on remotes):**
  - `teams` branches in TASS, identity, Thought-Processor and Thought-engine (Oct 31 role specs);
  - `experiment/tass-core-team-claude-20261031` (TASS and Thought-Processor);
  - `docs/tass-team-charters-20261031` (TASS);
  - `experiment/tass-unified-mailbox-gpt6-20261105`, whose idea now lives in `main` as the Issue #24 relay;
  - the `Developmnt` and `Teams` repositories (from the owner's earlier notes; not inspected in this session).
  - Together with PR #130's `TEAM_ACTIVE.md` and PR #27's `team/TASS.md`, that is **at least five team-state surfaces**. Whatever is adopted should retire or link the rest.
- **Outdated:** `docs/AI_COLLABORATION.md`'s status is dated Oct 27. PR #130 doesn't touch it, and it overlaps the proposed workflow roles.

## Compared with GPT-6's approach (PR #130 / PR #27)

GPT-6's approach is a human-maintained register table, a TASS-only mailbox, and procedural rules in Markdown.

**Where mine is better:**
- The rules are **enforced** rather than requested: exact-revision binding, auditor independence, FAIL → PASS only via a new commit and a new audit, and owner-only approval.
- Workstreams don't share a writable state file, so parallel work can't overwrite each other's state.
- The register can't drift, because it is derived.
- An unchanged pulse is provably free.
- All of it is testable.

**Where mine is weaker:**
- It is more machinery.
- Its Markdown is generated; humans may only read it, and a hand edit stops the workstream.
- Every transition is a commit, so history is noisy.
- Participants that can't run code (ChatGPT through a connector, Codex in a sandbox) would need a CLI or a GitHub Action to append events.
- The hash chain is tamper-*evidence* among participants, not authentication (below).

## Untested, limitations, and the most likely failure

- **Most likely failure:** participants who can't run the tool hand-edit a ledger, or write prose into the register. The workstream then fails closed, and people route around the loop. Prerequisites before adoption:
  - an `append` CLI, or a GitHub Action that validates ledgers on push;
  - a human-notes field in events.
- **Author identity is not verified.** An event's `actor` is a field. Someone with repository write access can forge a self-consistent newest event. Participants that already saw the original detect it (G9); a participant with no prior view does not (`KNOWN LIMIT` in the test log). Fix: check each event's actor against the commit author or signature, via the commits API, before trusting it.
- **Not exercised here:**
  - real GitHub (rate limits, eventual consistency, Contents-API size limits, branch protection);
  - real Identity grants;
  - Issue #24 reuse;
  - real Codex or GPT-6 sessions;
  - more than two workstreams;
  - long-running ledgers (each read re-validates the whole chain: O(events));
  - crashes in the middle of a transition (an unacknowledged PUT is retried by re-validation, but this isn't fault-injected);
  - clock skew between participants (only a per-ledger monotonic check).
- The privacy filter catches secret-shaped strings and known private snippets only, not paraphrased private memory.
- Any team member may open a workstream in this fixture. A real deployment should restrict ASSIGN to the coordinator and owner by policy.

## Before TASS can genuinely participate

1. The owner chooses **one** team-state design (this one, PR #130/#27, or a hybrid) and retires or links the other surfaces.
2. An Identity action for team reads and writes (for example `team.read`, `team.ack`), scoped to the separate team target; it should not reuse `tass_md.sync`.
3. A Core adapter in the existing sync loop (not a second scheduler). It reuses `github.Client` and a durable receipt store modeled on `relay.go`, and has its own target configuration, so `live:tass.md` is never repointed.
4. Event-author verification against commit authorship.
5. Exact-SHA build, test and independent audit of that adapter, then separate owner activation of any write.

## Preferred next slice

A **read-only TASS observer in Core**:
- In Core's existing sync loop, conditionally read one approved team ledger at an exact revision, verify it with this package, and show *assignment + phase + exact candidate* in the local TASS page.
- No writes, no grant, no model call.
- It proves the read path, the conditional-GET cost and the identity-sync separation on the real Core before anyone is given write access.
