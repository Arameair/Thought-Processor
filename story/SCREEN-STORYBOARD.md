# Screen storyboard — what Maya would actually see

[← Story home](../README.md) · [First scene](00-START-HERE.md) · [Blueprint seeds](BLUEPRINT-SEEDS.md)

These are **story panels for a designer**, not screenshots, not executable HTML, and not claims that any page already exists. **Every status, account, identity, count, receipt and version in the future product must come from an actual authoritative source or explicitly remain Unknown/Unavailable.** See [Trust and evidence](TRUST-AND-EVIDENCE.md). Each screen has one dominant task and one honest outcome.

## Screen A — Before account connection

**Visible:** Identity name; message "Let's create your own starting version"; GitHub **Not connected**; a brief explanation of the original project and what the customer receives. Primary action **Connect GitHub**. Secondary **Recover existing installation**.

**After action:** Opens GitHub authentication; returns verified username and permission summary or a clear setup failure. Never starts Processor or creates a repo without explicit consent.

**User emotion to test:** In control, not trapped.

## Screen B — This version belongs to you

**Visible:** Actual GitHub account (only once verified), proposed repository destination and visibility choice, **verified starter contents and licensing only after those open decisions have been settled**, true origin and attribution, and explicit requested access. Before this, show **Starter contents/license: Not decided** and keep creation disabled rather than guessing. Primary **Create my version**; **Choose an existing version** for returning users.

**After action:** Only after a lawful, approved distribution method exists should GitHub verify the created/selected repository and actual source lineage. On failure the flow remains in setup; never invent a successful creation or duplicate a repository on retry.

**User emotion:** Ownership.

## Screen C — One Start Experience

**Visible:** Identity status, durable installation, selected companion entity, verified repo, available feature distribution/access. Primary **Start Experience**. Expansion shows the pinned Thought Processor, selected Engine, demo source, optional worker and anticipated filesystem/process effects.

**After action:** A single consent starts a bounded session. Identity brings up the verified **Thought Processor**, not the older Thought Controller app or a random localhost service.

**User emotion:** Clarity and predictability.

## Screen D — The calm control room

**Visible:**

```text
THOUGHT PROCESSOR      First Experience     [Copy Context] [Stop Demo]
GitHub: Verified    Identity: Verified    Working entity: Compass

Selected Engine: Thought Engine
  Build: <exact SHA, verified artifact>
  Process: checking / started / attached / stopped / crashed
  Connection: disconnected / authenticated / incompatible
  Permission: no grant / granted / denied / authority unavailable
  Worker: not configured / ready / unavailable

[Ensure Engine Available] [Open original Engine UI] [Inspect logs]
```

**After action:** The launcher attaches to the correct running Engine or starts a session-owned one. The UI shows one truthful transition at a time, not a single "Everything online" label.

**User emotion:** Confidence through evidence, not green decoration.

## Screen E — May this observation be processed?

**Visible:** One selected entity, authenticated Demo Source S, Engine E, requested action `thought.observation.ingest`, exactly what permission covers and does not cover. Primary **Request approval from Identity**.

**After action:** Actual Identity grant result, ID/version and last-check timestamp. Local Engine toggles are never represented as an Identity grant. Publisher feature availability appears in a different status area.

**User emotion:** Trust.

## Screen F — Try something tiny

**Visible:** One **user-entered** test observation and **Send one**. It is identified as manually submitted test content, **not a game sensor reading**. The evidence timeline draws from authenticated service receipts, not scripted milestones:

```text
[S authenticated] -> [Identity checked] -> [Engine accepted/denied]
                                     -> [Queued] -> [Worker invoked?]
                                     -> [Spark / Trace / Thought / None]
```

**After action:** One real event receipt, worker state and optional actual result. If no worker is ready, no invented thought.

**User emotion:** "I can see the system noticing."

## Screen G — Show me it respects NO

**Visible:** Current grant and **Revoke**, then **Try again**; honest denial receipt and zero worker calls for refused new event. Alternative failure drill stops Identity: **Authority unavailable** is distinct from **Permission denied**. Other features remain running.

**After action:** No new protected processing. Data already sent before revocation cannot be magically recalled.

**User emotion:** Safety and agency.

## Screen H — Close it, reopen it, update it

**Visible:** **Stop Demo** explicitly describes what will stop (session-owned child processes only), and what remains (repo, Identity, Engine-owned history, unrelated services). Next opening restores the same installation and verifies actual GitHub/repo state. Any upstream update is separately proposed, reviewed and approved.

**User emotion:** Continuity.

## Five visual rules that matter

- **One active task per screen.** Advanced details are expandable, not missing.
- **Different status axes** for installed / running / authenticated / authorized / processing / worker ready.
- **Requested → accepted/refused → observed effect** for every action.
- **Only one expanded real service preview** at once; clear Copy Context, Logs and Metrics.
- **Never substitute a pre-written story card for a live outcome.** If a fact is unverified, show the appropriate Unknown/Unavailable/Not Tested state. Fixtures belong in isolated automated tests, not normal user-facing cards.
- **No fabricated statuses, performance numbers, timestamps, service IDs, source events, worker outputs, or successful grants.** Every rendered fact should have an inspectable source and last-check time. See [Trust and evidence](TRUST-AND-EVIDENCE.md).

## Ten-minute owner rehearsal (can be performed as a spoken walkthrough, before code)

1. Read Screen A: Is the GitHub step clear or unnecessarily intimidating?
2. Read Screen B: Is it obvious what the user owns and what remains publisher-controlled?
3. Read Screen C: Does **Start Experience** give enough consent without becoming a wall of setup?
4. Read Screens D–F: Can the owner predict what pressing each control will actually do?
5. Read G: Does the difference between revocation, outage, and model unavailability make sense?
6. Read H: What should survive closing UI, stopping the demo, or restarting the computer?

**A successful story rehearsal is not a software acceptance test.** It only verifies that the narrative and interface requirements make sense before technical blueprints.
