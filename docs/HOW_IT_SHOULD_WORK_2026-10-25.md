# How Thought Processor should work — notes for the build (2026-10-25)

**Status:** notes for discussion and planning; no code changed. They pair with:
- `Arameair/identity` → `docs/HOW_IT_SHOULD_WORK_2026-10-25.md`
- `Arameair/Thought-engine` → `docs/HOW_IT_SHOULD_WORK_2026-10-25.md`

The "whole picture" section is identical in all three repos; the rest is Processor's part. For the wider design options (what Processor should be, topology, UI, sessions, build paths), see [SERIOUS_PROTOTYPE_DESIGN_NOTES_2026-10-25.md](SERIOUS_PROTOTYPE_DESIGN_NOTES_2026-10-25.md).

Tags: **[KNOW]** seen in code or a run · **[INFER]** follows from that · **[GUESS]** opinion.

## The whole picture (this section is the same in all three repos)

### What the owner has asked for (collected from his directions, 10-23 → 10-25)
1. **Identity is the first program a person opens.**
   - It's a simple, polished setup for a non-technical person (for example the owner's brother).
   - **Connect GitHub** uses the publisher's built-in GitHub App, so nobody creates their own app. A guided "use your own app" walkthrough exists only for people who choose it.
   - Identity then finds and opens **Thought Processor**.
2. **Thought Processor comes first**, before work on the thinking itself.
   - Its job is to **have authority to install the Thought Engine**, and then start or attach it.
   - Processor installs the Engine **only if the person's Identity allows it**. First rule: the verified GitHub account can see `Arameair/Thought-engine`. A real grant service comes later.
   - Installs come from a **tagged release with a checksum**.
3. **Identity approves, Processor acts.** Processor starts or attaches the selected Engine only with Identity's approval.
   - **Strict rule:** Processor must **not** read the Engine's private status before an *Identity-approved* attachment.
4. **The Thought Engine is real.**
   - No simulations or mocks in normal use.
   - **Chat-Jimmy** is the current AI worker.
   - If the worker isn't ready, observations wait and say so.
   - Test stand-ins are allowed only when they are labelled **TEST FIXTURE** on screen and off by default.
5. **Truthful everywhere.** No fabricated state, no green light without evidence. Secrets (tokens, codes, `#key=` / `#code=` links, cookies) never appear in logs, diagnostics or issues.
6. **Features are the product.** Each program runs on its own, and turning one off doesn't break the others.
7. **How work is done:**
   - Work on experiment branches; the owner alone promotes to `alpha` / `stable` / `main`.
   - Every candidate must be runnable by the owner's Grok bot from an AGENTS.md runbook on **Windows**.
   - Report every check as PASS / FAIL / NOT RUN, tagged [KNOW] / [INFER] / [GUESS].

### The journey we're building toward (target, not today)

```text
 PERSON                IDENTITY (7790)            THOUGHT PROCESSOR (7860)        THOUGHT ENGINE (7798)      CHAT-JIMMY (7791)
   │ open Identity ──────►│
   │ Connect GitHub ─────►│ device sign-in (built-in app)
   │                      │ find / open Processor ───►│
   │                      │◄──── link (automatic if Identity started it, else one code)
   │                      │                            │ "may I install Thought Engine?"
   │                      │ rule: GitHub can see repo  │
   │                      │ ── signed INSTALL grant ──►│ download tagged release THROUGH Identity,
   │                      │                            │ check SHA-256, install to profile
   │                      │                            │ "may I start/attach it for TASS?"
   │                      │ ── signed ENGINE grant ───►│ start Engine (own child process),
   │                      │                            │ hand it the grant ───────────────►│ verifies Identity's
   │                      │                            │                                   │ signature, THEN attaches
   │                      │                            │◄── private status only now ───────│
   │ "start Chat-Jimmy live" (owner's explicit choice) │ start worker ─────────────────────┼──────────►│
   │ observations (typed, then real sources) ─────────►│ ──────── send ───────────────────►│ ◄── rate / question ──►│
   │◄──────────── Thoughts, with honest states ────────│◄──────────────────────────────────│
```

### Where today's code stands against that journey (2026-10-26 heads)

| Step | Today | Gap |
| --- | --- | --- |
| Identity sign-in, built-in app | **Works** (alpha-candidate-20261025) [KNOW] | Sign-in is forgotten on every Identity restart |
| Identity finds / opens Processor | **Works**, read-only discovery [KNOW] | Doesn't start Processor; no automatic link |
| Identity ↔ Processor link | **Works** on experiment: one-time code, hashed token, HMAC-proofed answers [KNOW] | Typed code every time; not on alpha |
| "May I install?" + install through Identity | **Works** on experiment with a TEST FIXTURE GitHub [KNOW] | On hold; no real release tag yet; Contents:read not approved |
| Identity approves start/attach | **Missing** [KNOW] | No signed grant exists; the Engine can't verify Identity |
| Processor starts the Engine | **Missing** (no runner, by design) [KNOW] | Three or more windows by hand |
| Processor ↔ Engine attach | **Works**, but with an **Engine-local code**, not Identity approval [KNOW] | Breaks owner rule 3 as written; interim only |
| Engine refuses mock, waits honestly | **Works** [KNOW] | Live Chat-Jimmy never run in this loop |
| Engine's own page and v1 routes | **Open to any local program** [KNOW] | Must be locked once Identity grants exist |
| Real observation sources | **Missing**: typed only [KNOW] | — |

### Shared words (use these exact names in all three repos)
| Word | Meaning |
| --- | --- |
| **Link** | Two local programs agree to trust each other. Done once with an 8-character one-time code, or automatically when one program started the other. Result: a 256-bit key; the issuer keeps only its hash. |
| **Grant** | A short signed statement **made by Identity**: *who* (GitHub account), *what* (feature + actions), *for which identity* (e.g. TASS), *which Engine installation*, *until when*. Signed with Identity's Ed25519 key; anyone holding Identity's public key can check it **without Identity running**. |
| **Actions** | `install`, `start`, `attach`, `observe`. Each one is granted separately. |
| **Entity** | An identity being served (TASS, a business). Today the Engine creates these; see the decision about who owns them. |
| **TEST FIXTURE** | Any stand-in. Off by default and labelled on every screen it touches. |

### Proposed grant format (to argue about; the same text is in every repo)
```json
{
  "v": "idg1",
  "grant_id": "g-…",
  "issuer": "<Identity system ID>",
  "account": {"github_login": "…", "github_id": 123},
  "feature": "thought-engine",
  "actions": ["start", "attach", "observe"],
  "entity_id": "<entity, when the action concerns one>",
  "engine_installation_id": "<pinned once attached>",
  "rule": "repo-visibility-v0",
  "issued_at": "RFC3339", "expires_at": "RFC3339 (e.g. +24 h)"
}
```
- Sent as `base64url(json) + "." + base64url(ed25519 signature)`.
- The Engine **pins Identity's public key once**, during the first owner-approved setup, by showing its fingerprint to the owner.
- From then on, the Engine checks grants offline.
- Revocation in v1: short expiry; Processor refreshes grants from Identity. Revocation lists come later.

### Order of work across the three repos (suggested)
1. **Identity:** signing key + `grant` endpoint for paired Processors (G1). Keep sign-in across restarts (G2).
2. **Engine:** pin Identity's key, and accept a grant for `attach` / `observe` *instead of* the local attach code (E1). Lock the page and v1 routes (E2).
3. **Processor:** ask Identity for grants and hand them to the Engine (P1). Then become the runner that starts the Engine and Chat-Jimmy (P2).
4. **All three:** one shared Link package; one Windows AGENTS.md covering the whole journey; one end-to-end acceptance script run on Windows CI.

---

## Processor's job, in one sentence
**Processor gets permission from Identity and then acts on it:** it installs the Engine, starts and attaches it, starts the worker, and sends observations. It shows the person truthfully what is happening at every step.

## What Processor has today [KNOW]
- **`alpha-candidate-20261026`** (@ `787d828`):
  - the 10-24/25 diagnostic shell (operator sign-in, guards, flow, metrics, Copy Context)
  - Engine card: attach with the **Engine's** one-time code, create/choose identity, switch Thought Processing, send observations, honest states, Thoughts, Engine-page link
  - AGENTS.md (Windows)
- **`experiment/install-thought-engine-20261026`** (@ `9596c04`, **on hold**):
  - pair with Identity
  - ask for access
  - download the release through Identity, check SHA-256, install into the profile
  - verify / uninstall

## Gaps against the owner's way
1. **Attach isn't Identity-approved yet.** It uses the Engine's own code. [KNOW]
2. **Install and attach live on different branches**, with two different pairing implementations. [KNOW]
3. **No runner.** The person starts 3+ programs by hand. [KNOW] The owner's direction says Processor starts/attaches the selected Engine. [KNOW]
4. **The page is two generations stacked.** It also has a new sign-in link every run. [KNOW]

## What Processor must add

### P1. Ask Identity, hand the grant to the Engine
- **Bring the install branch's Identity link into the alpha candidate**, using the shared Link package (Identity notes G6).
- **Before attaching:** ask Identity for `attach` (+ `observe` for the chosen entity). Send the grant to the Engine's new `POST /api/v1/attach {grant}` (Engine notes E1).
  - **No grant → no attach.** Show Identity's reason and next step word for word.
  - **No private Engine status before that** (owner's strict rule).
- **Refresh grants** before they expire (e.g. at 80% of their lifetime). If Identity is closed or signed out:
  - say so: "Engine keeps working until HH:MM"
  - after expiry, show **NOT PERMITTED — open Identity and sign in**.
- **Install:** keep the on-hold flow as is (it already asks Identity first and checks SHA-256). [GUESS] Later it can use an `install` grant through the same code path.

### P2. Become the runner, carefully
The owner's flow is "Identity approves, Processor starts/attaches". [KNOW]
- **Start only:**
  - the Engine that Processor **installed** (path + SHA-256 recorded in `install.json`)
  - a Chat-Jimmy path the owner approved once
- **Never adopt or kill** a process Processor didn't start. If an Engine is already running at the address, attach to it instead.
- **Start the Engine with the grant** (`-grant-file` / env var, Engine notes E4), so it attaches with no typing.
- On Windows, put children in a **Job Object** so they stop when Processor stops. Restart on crash with backoff: at most N times, then show FAILED with the last log lines.
- **Chat-Jimmy live is always an explicit owner button:** "Start Chat-Jimmy live — uses chatjimmy.ai". It's never automatic. `-mock` is never used.
- Show each child's console output in Processor (redacted). This replaces "look in the other window".
- **Tests:**
  - Start/Stop affect only children
  - an unknown or modified Engine file (SHA mismatch) is refused
  - a crash loop stops after N tries
  - stopping Processor stops its children, and only those

### P3. One status line, one next step
Name the states (see SERIOUS_PROTOTYPE_DESIGN_NOTES §5):

`NOT SIGNED IN (Identity)` → `NOT ALLOWED` → `NOT INSTALLED` → `ENGINE STOPPED` → `NOT ATTACHED` → `NO IDENTITY` → `NOT PERMITTED` → `WAITING FOR AI` → `TEST FIXTURE` → `DEGRADED` → `RUNNING`

Every screen, Copy Context and test uses these names.

### P4. Tidy the page
- Three tabs: **Home** (status line, next step, live observations and Thoughts) · **Programs** (Identity, Engine, Chat-Jimmy: link, start/stop, versions) · **System** (diagnostics).
- Remove the disabled "Engine controls" list once P2 exists.
- **Remember the browser:** sign in once, with a revocable cookie, instead of a new `#code=` link every run (owner decision).

### P5. One Windows journey, one acceptance script
- A single AGENTS.md covering Identity → Processor → install → start → Chat-Jimmy → Goldshire ×3 → restart.
- Turn the 26-check demo (`Thought-engine/scripts/demo_processor_loop.py`) into the acceptance suite for the whole journey. Run it on **Windows CI** (`windows-latest`).
- Fixture runs are reported as TEST FIXTURE, never as live.

## What Processor must NOT do
- Decide permission itself. Identity decides, the Engine enforces, Processor carries the grant and shows the result.
- Keep its own copy of Thoughts or observations (it only shows the Engine's).
- Read Engine private status before an Identity-approved attachment.
- Start anything the owner didn't approve, or stop anything it didn't start.
- Log or show grants, keys, codes, `#key=` / `#code=` links, or observation text in Copy Context.

## Acceptance checks for "Processor done for this stage"
1. Fresh Windows machine, following AGENTS.md:
   1. Identity → GitHub → Processor opens and links (no code if Identity started it)
   2. Install Engine (ALLOWED, SHA matches)
   3. Start Engine (one click; attaches with a grant)
   4. Start Chat-Jimmy live (owner click)
   5. TASS → Goldshire ×3 → one Thought
2. Brother's account without access → NOT ALLOWED with the reason. Nothing is downloaded or started.
3. Close Identity → the Engine keeps working until the grant expires, then NOT PERMITTED with a clear next step.
4. Stop Processor → only its children stop. Start again → everything comes back, same Thought.
5. No secret in any log, console, Copy Context or screenshot.

## Questions for the owner
- P2: may Processor start programs on your computer, as long as it only starts what you approved and only stops what it started?
- P4: remember the browser (sign in once), or keep a new link every run?
- Should Processor become *the* place to see Thoughts (folding in Thought Controller), or should the Engine page / Thought Controller stay that place? (SERIOUS_PROTOTYPE_DESIGN_NOTES §2)
- Order of work: P1 (grants) before P2 (runner), or the runner first because it removes the most friction?

— Claude
