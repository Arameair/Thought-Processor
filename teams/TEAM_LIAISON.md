# Team Liaison — shared role charter

**Shared file, v0.1 — October 31, 2026.** Same text in all five repos' `teams/` folders; change all copies together.

**Status:** Proposed role. No Liaison is appointed or running. Adapted from Dustin's *Team Liaison — Detailed Design Proposal v0.1*.

## Mission

> Maintain trustworthy communication between authorized team participants. Ensure assignments are acknowledged, progress claims are evidence-based, blockers reach the correct decision-maker, and cross-team handoffs are reliably tracked without exceeding delegated authority.

The biggest risk is the Liaison turning into a second boss. It isn't one. It reports problems; the Project Lead decides.

## Who the Liaison is

Every team has its own Liaison: the TASS Liaison, the Identity Liaison, and so on. A KYSSA team has its own too. Each one works only inside its own team's access. **No Liaison controls another team.** Liaisons talk to each other only through handoffs ([TEAM_OPERATING_MODEL.md](TEAM_OPERATING_MODEL.md), section 7).

The Liaison is responsible for the **whole team's** communication: messages between its own workers, plus handoffs with other teams.

## Six jobs

| Job | What it means |
| --- | --- |
| **Communication** | Deliver authorized task messages and status requests |
| **Acknowledgment** | Make sure every new assignment gets a `WORKING` reply |
| **Progress tracking** | Keep one card per worker, showing activity and evidence |
| **Completion check** | Check `DONE` claims against GitHub or another trusted source |
| **Escalation** | Take unresolved problems to the right person (Project Lead, then Dustin) |
| **Cross-team** | Send and receive handoffs with other teams' Liaisons |

## The worker card

One **durable** card per worker, updated each pulse (not a new card every time):

| Field | Example |
| --- | --- |
| Worker / role | Engineer |
| Owning team | TP |
| Conversation target | where messages reach this worker |
| Active task | `TP-T-014` — implement approved launcher fix |
| Activity status | WORKING received at … |
| Evidence status | UNVERIFIED — commit reported, tests pending |
| Last verified evidence | repo, branch, full SHA, time |
| Last nudge | message ID, time, delivery receipt |
| Next checkpoint | time or event |
| Requested nudge | yes/no, from whom |
| Delivery state | sent · acknowledged · uncertain · blocked · unavailable |
| Next permitted action | monitor · one nudge · escalate |

**Activity and evidence are tracked separately.** A worker can say `WORKING` while its result is still UNVERIFIED.

## How the Liaison handles silence

| Situation | Liaison does |
| --- | --- |
| Acknowledged and inside its checkpoint | Leaves it alone |
| Worker asked for a later nudge | Queues it for the decision-maker |
| Checkpoint overdue, delivery confirmed | Recommends one targeted nudge |
| Sent, but delivery uncertain | Records UNKNOWN; investigates; does **not** resend |
| Worker reports a real blocker | Escalates to the Project Lead |
| The Liaison itself goes quiet | The independent pulse reports "Liaison unavailable" |

## Limits

The Liaison **may**: read authorized status, send permitted messages, ask for acknowledgments, record receipts, escalate.

The Liaison **may not**: create new scope, approve permission changes, read another team's or identity's private records, merge or promote branches, or say a test ran when it didn't.

When Identity, GitHub or a worker's channel is down, it fails safe: it records what it doesn't know and waits for the next pulse or a person.

## First thing to build

Before any automatic nudges: **a durable communication record** that survives restarts and ties every message to its task and its authorization. Without it, the system can forget it already sent a nudge and send duplicate instructions.

Smallest useful test: one worker sends `REQUEST_NUDGE`, the Liaison acknowledges it, the decision-maker approves it, one nudge is delivered, and the receipt shows up in the record.
