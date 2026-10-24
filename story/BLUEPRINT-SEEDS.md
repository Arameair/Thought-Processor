# From this story to blueprints

[← Story home](../README.md) · [Screen storyboard](SCREEN-STORYBOARD.md) · [Review questions](REVIEW-QUESTIONS.md)

This is the **translation ledger**. It deliberately does not freeze the final protocols, installation technology, or visual design. **No runtime panel or demo counts as credible if it fabricates or guesses data.** See [Trust and evidence](TRUST-AND-EVIDENCE.md). The story tells us **what must happen**; the eventual blueprint tells us **how to implement and verify it**.

## Scene → blueprint inventory

| Story moment | Blueprint we will need | Unambiguous behavior to preserve |
| --- | --- | --- |
| [00 · The door opens](00-START-HERE.md) | GitHub-gated first-run/setup/recovery states | Before account verification, repair/setup allowed but full product never incorrectly shown READY |
| [01 · It becomes hers](01-IT-BECOMES-HERS.md) | **Open research:** source scope, legal redistribution, repository creation and update options; publisher access boundary | Real starter source (not just settings), attributed origin, explicit approval; do not assume a chosen creation method |
| [02 · The control room](02-THE-CONTROL-ROOM.md) | Identity→Processor Start Experience/session manifest | Exact pinned product/build, bounded launch, no Thought-Controller substitution or unapproved autostart |
| [03 · The engine wakes](03-THE-ENGINE-WAKES.md) | Processor→launcher `ensure_running`; launcher→Engine health/identity | Attach if correct instance runs; otherwise start once; accurate ownership and compatibility |
| [04 · A real permission](04-A-REAL-PERMISSION.md) | Identity grant/registration + Engine caller authentication | Issuer different from enforcer, authenticated sender, scoped grant, action-time check |
| [05 · A thought begins](05-A-THOUGHT-BEGINS.md) | Source→Engine observation, queue/worker/trace receipts | One real event, clear distinction queued/processed/persisted, no fake worker output |
| [06 · The system says no](06-WHEN-IT-GOES-WRONG.md) | Failure, revocation, crash recovery, diagnostics and safe stop | Denial vs unavailable; no new worker call on refusal; independent services survive UI close |
| [07 · Maya comes back](07-THE-NEXT-DAY.md) | Durable identities/data, reconnect, customer-owned source updates | Same repo and IDs; no silent repo duplication or forced update |

## Shared vocabulary to settle before detailed diagrams

- **Publisher entitled:** feature *delivery/hosted access* currently permitted by a trusted publisher-controlled boundary; not local reading permission.
- **GitHub verified:** current account/repository permission confirmed; not proof of local operating trust.
- **Identity authorized:** the authenticated **source S** has a scoped **action on entity X at Engine E**, and Engine can verify it now.
- **Available:** correct executable/build is present and a verified running instance responds; not necessarily permitted or model-ready.
- **Worker ready:** actual model service can answer; Engine may be connected while this is false.
- **Engine receipt:** accepted/queued/processed/persisted are distinct claims.
- **Session-owned:** process started by this approved experience; **attached:** process that was already running, not ours to kill.
- **Thought:** a persistent candidate arising from real observations/rules; *not* synonymous with an AI answer or a purchased Memory feature.

## Authority / lifecycle picture for a future schematic

```text
USER
  |
  v
GITHUB ACCOUNT + CUSTOMER-OWNED STARTER REPO   <-- verifiable Arameair origin; bootstrap method OPEN
  |
  v
IDENTITY (who, where, local permission) <----- publisher feature access evidence
  |
  | approves one Start Experience
  v
SINGLE SESSION LAUNCHER ----------> THOUGHT PROCESSOR
  |                                    |
  +--- starts/attaches Engine <--------+ ensure_running request
                 ^                     |
                 | authenticated read  +--- diagnostics/control receipts
                 |
    DEMO SOURCE -+--> ENGINE ----- check scoped grant ----> IDENTITY
                         |
                         +--- optional worker ---> actual result
                         |
                         +--- Engine-owned events, Traces, Thoughts
```

**Not a runtime claim.** The launcher is one bounded approved process-management authority, not a second publisher entitlement issuer. Thought Processor is never required in the observation data path. Identity does not execute model work.

## Five major blueprint decisions the story cannot answer by itself

1. **Bootstrap and license: UNDECIDED.** Exactly which real starter files can be redistributed, the applicable license, and whether fork, template or another reviewed method best preserves ownership, updates and origin. **Do not imply a selected mechanism or a published starter release.**
2. **Publisher control:** How feature access is issued/verified and what withdrawal actually affects: new downloads, hosted use, future updates or an expressly licensed local term. Copied code cannot be assumed retractable.
3. **Lifecycle component placement:** The *first demo* prefers a small session launcher created after Identity's explicit consent and callable by Processor. Should its process outlive Processor backend? How will later Windows/WSL/VM service control work?
4. **Trust and grant contract:** How the genuine Engine/source/Identity authenticate, how grants are signed or checked, what exact expiry/revocation and queued-work behavior is acceptable, and how secret keys are protected.
5. **Feature bundle and persistence:** Whether Processor and Engine are separately delivered/installed components or one commercial feature bundle containing independently running services; which data stays stable across updates.

**Explicit future work:** Memory, game/desktop log capture, multiple models, continuous thought waves and dynamic coding must be tested separately and cannot be represented as already present in this first prototype.

## Suggested staged prototype after blueprint approval

| Stage | First visible result | Evidence required |
| --- | --- | --- |
| A. Ownership | Verified customer account/repo with actual starter files and retained origin | GitHub-side ownership/access/contents + no secrets |
| B. Startup | Identity launches exact Processor; Processor ensures exact Engine is available | SHA/artifact digest, PID/port/instance, attach-or-start trace, no duplicate |
| C. Trust | Identity allows then denies an authenticated test source; Engine enforces | Signed/verified decision, intake and queue checks, worker call counts, fail closed on outage |
| D. Experience | One event produces real evidence and, if the worker is ready, an actual result | Source ID, event receipt, worker request ID, persisted result, truthful missing outcome |
| E. Resilience | Close/reopen, kill one service, stop demo, propose update | Process ownership, durable IDs/history, no orphan/kill of unrelated apps, rollback |

Until Stage C is real, do **not** call the integration an Identity-authorized live experience. **Do not render fixture results in the normal UI.** Fixtures, if required for technical testing, remain isolated, labeled **NOT LIVE**, and never feed production status, demonstrations or KPIs.

## One proposed first blueprint to draft

**Blueprint B-01: Processor–Engine Availability Contract**

Define `ensure_running` with a bounded approved candidate manifest; whether an existing process is session-owned or independent; what proves product, version, service identity and protocol; how a missing port, failure, crash and restart are surfaced; and what survives closing the browser or Processor backend. Build Stage B **before** an expansive multi-feature control UI.

Tie all acceptance evidence to branch/commit SHA, actual executable digest, **authoritative source of every displayed value**, timestamp, freshness, sanitized log/decision receipts, explicit pass/fail or unknown, and owner decision. If the source cannot be checked, expose that limitation rather than substituting a plausible value.

## Completion definition for this *storybook* (not software)

The owner should be able to read the chapters in order and explain, without architecture jargon:
- What do I own?
- How does it start?
- How does an Engine join?
- What am I allowing it to do?
- What happened to my observation?
- What breaks when I withdraw permission?
- What remains when I come back?

If any answer is missing or confusing, revise the story *before* writing the implementation blueprint.
