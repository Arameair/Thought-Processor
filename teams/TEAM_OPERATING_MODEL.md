# Team Operating Model — shared by all five teams

**Shared file, v0.1 — October 31, 2026.** This exact file is in the `teams/` folder of the `teams` branch in TASS, Identity, Thought-Processor, Thought-engine and Thought. If you change it, change all five copies together.

**Status:** Proposal. Nothing here appoints anyone, starts an agent, schedules a pulse or grants a permission. Dustin (Arameair) approves every change to `alpha`, `stable` and `main`.

**Sources:** Dustin's *How an AI Team Should Work* and *Team Liaison — Detailed Design Proposal v0.1* (both October 31, 2026), plus each repository's README, ARCHITECTURE and WORKFLOW files.

---

## 1. The idea in one paragraph

Every team has the same five roles and speaks the same small protocol. Each worker knows how to report its state and ask for attention. The Liaison knows how to route that information. The decision-maker (Dustin, or a supervising identity he authorizes, such as TASS or KYSSA) decides what it means and what to do. Teams work together through **handoffs** between Liaisons, never by one team ordering another team's workers around.

## 2. The five roles in every team

| Role | Owns | Reports to the Liaison | Asks for help when |
| --- | --- | --- | --- |
| **Project Lead** | Assignments, priorities, scope inside what Dustin approved, acceptance criteria | Assignments, decisions, deadlines, dependencies | Assignments conflict, or a decision needs Dustin |
| **Engineer** | The code and the commit it lives in | Accepted tasks, branch + full SHA, test output | Specs, access or verification are missing |
| **Test Architect** | Proving it works (or doesn't) on a named commit | PASS / FAIL / NOT RUN per check, open defects | The build, test environment or candidate SHA is missing |
| **UX Architect** | What the person sees and understands | Designs, usability findings, acceptance findings | Product behavior or a user decision is unclear |
| **Team Liaison** | Communication: who was told what, who acknowledged, what evidence exists, what is stuck | Overall communication state, unresolved handoffs, nudge recommendations | Supervisory intervention is needed |

The Project Lead owns **tasks**. The Liaison owns **communication tracking**. Neither one can grant a permission just because a worker asks.

**Full role specifications:** [TEAM_PROJECT_LEAD.md](TEAM_PROJECT_LEAD.md) · [TEAM_LIAISON.md](TEAM_LIAISON.md) · [TEAM_ENGINEER.md](TEAM_ENGINEER.md) · [TEAM_UX_ARCHITECT.md](TEAM_UX_ARCHITECT.md) · [TEAM_TEST_ARCHITECT.md](TEAM_TEST_ARCHITECT.md).

**Who checks which evidence** (Project Lead spec, section 10):

| Role | Confirms |
| --- | --- |
| Team Liaison | Messages were received, and the artifacts referenced actually exist |
| Test Architect | The evidence supports technical correctness and security |
| Project Lead | Every required deliverable, verification gate and acceptance decision is accounted for, using existing verified records rather than retesting |

**Reassignment** (Project Lead spec, section 8): bounded. The Project Lead may reassign within approved scope and existing grants, recording the reason and keeping all evidence. After **two** reassignments of the same task it must escalate to the owner. Testing is **never** reassigned to get a different result. A stalled Project Lead can be escalated by the Liaison straight to the identity or owner.

**Task lifecycle** (Project Lead spec, section 9): Proposed → Approved → Authorized → Assigned → Acknowledged (`WORKING`) → Executing → Reported complete (`DONE`) → Verified → Accepted → Closed. Other states: BLOCKED, PAUSED, FAILED VERIFICATION, REJECTED, CANCELLED, AWAITING APPROVAL, AWAITING EVIDENCE. A task never jumps from `DONE` to Accepted without the gates in between.

**Two things from the Test Architect spec that apply to everyone:** a confirmed **P0 (critical security)** finding blocks the affected delivery, and no worker can waive that gate. An urgent security escalation is never held back by the normal nudge limits or by waiting for the next pulse.

**Engineer specialty (from TEAM_ENGINEER.md):** for each task, the Project Lead picks the Engineer's specialty (software, systems integration, hardware, diagnostics, data, automation, or custom). It is **locked** once the Engineer sends `WORKING`. If the task needs a different specialty, the Engineer reports `BLOCKED`, the assignment is closed or cancelled, and a new one is created and authorized.

## 3. The three-message protocol

Every assignment, inside a team or between teams, goes through three stages.

| Stage | Who sends | Format | What it proves |
| --- | --- | --- | --- |
| **A. Dispatch** | Project Lead or Liaison | Task ID, immediate action, permissions that apply, acceptance criteria, how to report | Only that the message was handed to a channel |
| **B. Working** | The receiving worker | `WORKING: <task ID> — <immediate action>` | The worker acknowledged it. **Not** that it is succeeding. |
| **C. Result** | The receiving worker | `DONE: <result> — evidence: <repo, branch, full SHA, test output or link> — limits: <what's left>` or `BLOCKED: <verified obstacle> — evidence: <…> — needs: <next action and who>` | `DONE` is a **claim**. The Liaison checks the evidence before marking it VERIFIED. It does not mean independent review passed, Dustin accepted it, or it was released. The Project Lead formally closes the assignment. |

Three things are never the same thing:

- **delivered ≠ acknowledged** — a sent message is not a `WORKING`.
- **acknowledged ≠ done** — a `WORKING` is not progress.
- **claimed ≠ verified** — a `DONE` without evidence stays UNVERIFIED.

## 4. IDs

| Thing | Format | Example |
| --- | --- | --- |
| Team code | see [README.md](README.md) | `IDN` |
| Task | `<TEAM>-T-<number>` | `TP-T-014` |
| Message | `<TEAM>-M-<YYYYMMDD>-<number>` | `TASS-M-20261031-003` |
| Cross-team handoff | `HO-<FROM>-<TO>-<YYYYMMDD>-<number>` | `HO-TP-IDN-20261031-01` |
| Alert | `<TEAM>-A-<YYYYMMDD>-<number>` | `TE-A-20261031-02` |

A retried message keeps its original ID. A receiver that sees an ID it already handled answers with the earlier result instead of starting the work again. That is what stops a resend from creating a duplicate task.

## 5. Asking for attention: the alert record

A worker never relies on a chat sentence alone. It sends a small record:

| Field | Example |
| --- | --- |
| Alert ID | `TP-A-20261031-01` |
| From | Engineer |
| To | Team Liaison |
| Type | `REQUEST_NUDGE` · `NEEDS_ATTENTION` · `BLOCKED` |
| Target | Test Architect |
| Task | `TP-T-014` |
| Reason | Required test acknowledgment not received |
| Evidence | Link to the task and latest handoff |
| Requested action | Ask Test Architect for runner availability |
| Priority | Normal |

The three types mean:

- **`REQUEST_NUDGE`** — "check on me (or on X) after the next checkpoint." The Liaison queues it; the decision-maker or an approved policy decides whether to send it.
- **`NEEDS_ATTENTION`** — "someone else has gone quiet on a handoff I need." The Liaison checks the evidence and recommends **one** targeted nudge, not a wake-everyone broadcast.
- **`BLOCKED`** — "I can't continue without X." The Liaison takes it to the Project Lead, who decides.

**A request for a future nudge must be written down outside the chat** (a durable record: a file, an issue comment, a queue). An idle chat has no running process to remember it.

## 6. Ground Truth First — mandatory for every team

No one on any team may claim:

- a worker is executing because a message was delivered;
- a worker finished because it said `DONE`, without evidence;
- a test passed when no real test result exists;
- a GitHub change exists without naming the repository, branch and full commit SHA;
- someone approved something without an authorization record (for permissions, that record lives in Identity).

Label claims with **[KNOW]** (seen directly), **[INFER]** (reasoned from evidence) or **[GUESS]**.

| Kind of state | Values |
| --- | --- |
| **Evidence** (is the claim proven?) | **VERIFIED** (passed a named check) · **UNVERIFIED** (claimed, not checked) · **FAILED** (the check ran and failed) · **CONFLICT** (evidence contradicts the claim) |
| **Test result** (per check) | **PASS** · **FAIL** · **NOT RUN** · **BLOCKED** (couldn't run, obstacle documented) |
| **Delivery / communication** | sent · acknowledged · **UNKNOWN** (uncertain, don't resend) · **BLOCKED** · unavailable |

Activity and evidence are tracked separately: a worker can be `WORKING` while its result is UNVERIFIED. Every significant evidence record names its source, when it was observed, and how it was verified. A previous PASS is not proof for the current version. When evidence is missing, say it is missing. A test fixture is not real AI. A reachable port is not authorization.

This matches the rule already in every repo: **discovered ≠ paired ≠ installed ≠ authorized ≠ running.**

## 7. Working with another team: handoffs

Suppose the Thought Processor team needs the Identity team to add a capability.

1. **TP Project Lead** decides the request and its approved scope.
2. **TP Liaison** writes a handoff card (section 8) addressed to the **IDN Liaison**, with a new handoff ID.
3. If the handoff shares protected material or asks for access, it needs an **authorization record** (Identity's grants, or Dustin's written approval). Plain repo docs and public issues need none.
4. **IDN Liaison** answers `RECEIVED: <handoff ID>` and puts it in front of IDN's Project Lead.
5. **IDN Project Lead** answers `ACCEPTED`, `DECLINED` or `NEEDS-INFO`, through IDN's own authority. IDN's workers are **never** treated as assigned just because TP asked.
6. Once accepted, the work runs on IDN's own task IDs with the usual `WORKING` / `DONE` / `BLOCKED` messages.
7. **TP Liaison** records the outcome and links the evidence (IDN's branch + SHA + test results).

If two teams disagree about who owns something, neither team decides. Both Liaisons record a **CONFLICT** and the Project Leads take it to Dustin. Each team file lists its known open conflicts.

### Where handoffs live today

[KNOW] The **Team Member** transport that would carry messages between AI conversations automatically is not built or verified. Until it is:

- [INFER] A handoff is a **GitHub issue in the receiving team's repository**, titled `[HANDOFF HO-…] <short request>`, with the handoff card as the body. `RECEIVED`, `ACCEPTED`, `WORKING`, `DONE` and `BLOCKED` are comments on that issue.
- Cross-repository coordination stays on [Developmnt issue #1](https://github.com/Arameair/Developmnt/issues/1).
- A GitHub post does **not** wake another agent or prove it read anything. Only an observed reply counts as an acknowledgment.

## 8. Handoff card

```text
HANDOFF:      HO-<FROM>-<TO>-<YYYYMMDD>-<nn>
FROM:         <team> Liaison, for <team> Project Lead
TO:           <team> Liaison
REQUEST:      <one sentence>
WHY:          <what it unblocks>
SCOPE:        <what is in / what is out>
ACCEPTANCE:   <how both sides will know it is done>
EVIDENCE NEEDED BACK: <repo, branch, full SHA, test results, screenshots…>
AUTHORIZATION: <none needed | Identity grant ref | Dustin approval link>
NEEDED BY:    <date or "not set">
CONTACT IF BLOCKED: <role>
```

## 9. The pulse (proposal, not running)

**Latest direction:** the Project Lead spec v1.0 (`TEAM_PROJECT_LEAD.md`) sets a **local 30-second heartbeat** as the system pulse. The Engineer, UX Architect and Test Architect specs (all v1.0, written earlier) still say a provisional **20 minutes**. Those files are Dustin's text and aren't edited here. **Until Dustin confirms, treat the 30-second heartbeat as the current direction.**

What the 30-second heartbeat means (from the Project Lead spec):

- It is a **scheduling opportunity**, not a deadline. Nobody has to finish anything in 30 seconds.
- Each tick advances the **rotating thought**: the five specialist perspectives add to one shared thought record (sections 5–6 of the Project Lead spec), along with a **Prediction Model** and an **Outcome Analyst**. Neither of those gets any execution authority.
- A tick must **not**: duplicate a protected action already running, replace current evidence with stale information, reopen finished tasks, create an assignment just because time passed, or interrupt legitimate work. If a model is still busy, the tick records that instead of starting another identical run.
- Thinking, execution and communication are three different activities. They can overlap but must not be confused.

[INFER] Proposed reconciliation, **for Dustin to confirm:**

| Activity | Cadence | Notes |
| --- | --- | --- |
| Heartbeat / thinking rotation | every 30 s, local | Advances the shared thought; mostly no-ops when nothing changed |
| Communication check | on events, plus a slow sweep | Reads receipts, overdue checkpoints, requested nudges. Sends nothing by itself. |
| Worker nudges | rate-limited by time, not per tick | The old "at most 5 per pulse" was written for a 20-minute pulse. At 30 seconds that would allow up to 600 an hour, so the cap should be per time window (for example, 5 per 20 minutes). |
| Security-critical escalation | immediately | Never held by rate limits or ticks (Test Architect spec, section 11) |

Each cycle: **Wake → Predict → Collect → Interpret → Deliver → Remember.** If the team is on track, nothing is sent. An uncertain delivery is recorded as UNKNOWN and investigated, never blindly resent. The pulse must stay observable even if the Liaison is down, because the Liaison can't be the only thing that notices its own failure. Task and thought state persist across ticks and restarts.

## 10. What every team shares, no matter what

- **Owner approval.** Experiments never promote themselves. Only Dustin moves `alpha`, `stable` or `main`.
- **Branching.** New work starts on `experiment/<topic>-<YYYYMMDD>` from the latest approved base, with the base SHA recorded.
- **Evidence with every technical handoff.** Exact commit, PASS / FAIL / NOT RUN, environment.
- **Identity is the only permission authority.** No team grants access by itself, and GitHub access never means access to private conversations or Admin credentials.
- **Privacy.** Each identity's data stays with that identity unless a grant says otherwise.
