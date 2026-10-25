# Toward a serious Thought Processor prototype — design notes for discussion

**Status:** discussion notes only. Nothing here is decided, and no code changed. Written 2026-10-25 by Claude, after reading the current heads of all three repos:

| Repo | Branch read | What it is today |
| --- | --- | --- |
| Arameair/identity | `alpha-candidate-20261025`, `experiment/engine-access-pairing-20261026` | GitHub device sign-in with the publisher app, feature catalog, local drafts, Processor discovery. The experiment adds pairing with Processor and an install-access answer. |
| Arameair/Thought-Processor | `alpha-candidate-20261026` | The diagnostic shell from 10-24/25, plus an Engine card (attach with a code, send observations, follow them to Thoughts). |
| Arameair/Thought-engine | `experiment/engine-processor-loop-20261026` | The v1 Spark → Trace → Thought engine, plus the Processor contract, attach codes and a labelled TEST FIXTURE worker. |

These notes build on, and try not to repeat:
- [DESIGN_IDEAS_TO_LOOK_AT_2026-10-25.md](DESIGN_IDEAS_TO_LOOK_AT_2026-10-25.md): outside ideas (TUF, PASETO, blackboard, event sourcing…)
- [INSTALL_AUTHORITY_DISCUSSION_2026-10-25.md](INSTALL_AUTHORITY_DISCUSSION_2026-10-25.md): who may install the Engine
- [THOUGHT_PROCESS_DISCUSSION_2026-10-25.md](THOUGHT_PROCESS_DISCUSSION_2026-10-25.md): how the thinking should work

This file is about the **Processor itself**: what it should be, how it should be built, and what "serious" means for the next prototype.

Tags: **[KNOW]** = seen in code or a run · **[INFER]** = follows from what was seen · **[GUESS]** = opinion or prediction.

---

## 1. Where we actually are (honest inventory)

### What works now
- **Processor ↔ Engine loop.** [KNOW] Processor attaches to a running Engine with a one-time code. It creates or chooses an identity, switches Thought Processing on, and sends observations. Each observation shows an honest state: waiting for AI, queued, processing, done or failed.
- **Restarts.** [KNOW] The attachment, the observations and the Thoughts survive a restart of everything. This was proven with real processes; the AI was a labelled TEST FIXTURE.
- **Identity.** [KNOW] It signs a person in with GitHub (the publisher app is built in) and can answer "may this account have Thought Engine?" (rule repo-visibility-v0). That answer goes to a paired Processor; this is experimental and the install is on hold.
- **Safety basics in all three.** [KNOW] Loopback only, Host/Origin checks, secrets redacted from diagnostics, and fixtures labelled on screen.

### What hurts, roughly in the order a new user hits it
1. **Too many windows and sign-ins.** [KNOW]
   - The loop needs **3 programs** (Chat-Jimmy, Engine, Processor), each in its own PowerShell window. Identity makes 4, Thought Controller 5.
   - Processor prints a **new `#code=` sign-in link every run**.
   - Identity **forgets the GitHub sign-in on every restart**.
   - The Engine needs an attach code once.
   [INFER] That is four different "prove it's you" moments, and none of them is shared.
2. **Two pairing mechanisms that do the same thing.** [KNOW]
   - Identity ↔ Processor pairing: 8-character code, hashed token, HMAC proof.
   - Engine ↔ Processor attach: 8-character code, hashed token, no proof.
   They were written separately and differ in small ways.
3. **The Engine is only half locked.** [KNOW] The Processor contract needs a key, but the Engine's own page and its v1 routes (`/api/status`, `/api/entities/...`) are open to any program on the computer. So the attachment records and scopes what Processor does; it doesn't protect the Engine.
4. **Three UIs show the same Thoughts.** [KNOW] The Engine's own page (Sparks sky, Traces, Thoughts), Thought Controller, and now Processor's Engine card. [INFER] Nobody has decided which one the person lives in.
5. **Processor's page is two products stacked.** [KNOW]
   - The 10-24 diagnostic shell: Identity card, "Engine controls" with disabled buttons, flow, metrics.
   - The 10-26 working Engine card, on top.
   [INFER] It reads long and repeats itself: the Engine appears twice, and an "Attach" button is shown disabled next to the working one.
6. **Observations are typed by hand.** [KNOW] There is no real source yet (TASS, Communication Bridge, ScreenBridge, game logs).
7. **The thinking is untested on the real model.** [KNOW] The live 8B model has never run this loop. [GUESS] Grouping by topic label is the weakest link; THOUGHT_PROCESS_DISCUSSION Q4 covers it.
8. **Docs drift.** [KNOW] `docs/STATUS.md` still says "documentation only — no runnable implementation" (last reviewed 10-23). The `alpha` branch is still the docs baseline.

---

## 2. The first question: what *is* Thought Processor?

Every later choice depends on this. Here are three honest options.

| | **A. Control room** | **B. The Thought app** | **C. Hub / supervisor** |
| --- | --- | --- | --- |
| One line | Where you **set up, watch and fix** Thought Processing; the Engine page stays the deep view | **The** place you use Thought Processing day to day: observations in, Thoughts out | **Starts and keeps alive** Engine, Chat-Jimmy (and later sources); shows health; links to each program's own page |
| You live in | Processor for status; Engine page for details | Processor only | Each program's own page |
| Thought Controller | Stays a separate dashboard, or is retired | Folded into Processor | Stays separate |
| Processor owns processes? | No | No (or optional) | **Yes** |
| Closest to today | Yes, mostly | Partly: the Engine card is a start | No: start/stop are blocked |
| Risk [GUESS] | Two UIs to keep in sync | Processor grows into a second Engine UI, with the "second database" temptation the VISION warns about | A supervisor that kills or spawns the wrong thing is worse than none |

**Suggestion to argue with** [GUESS]: **A + a small slice of C.**
- Processor is the control room.
- Processor can **start and stop only the programs it launched itself**: Engine and Chat-Jimmy, from paths you approved once. That alone removes most of the window juggling.
- The Engine's own page stays the place for Spark/Trace/Thought detail.
- Thought Controller's job is merged into the Engine page or retired, so there are two UIs, not three.

Processor would still never hold its own copy of Thoughts (VISION, "Truthfulness").

---

## 3. Process topology: how many programs, and who starts them

Today: 3–5 independent processes, each started by hand. [KNOW]

| Option | How it works | Good | Bad |
| --- | --- | --- | --- |
| **T1. Keep separate, start by hand** (today) | runbook + windows | maximum independence; nothing hidden | worst first-run experience; easy to run the wrong branch |
| **T2. Separate processes, Processor supervises** | Processor runs `thought-engine.exe` and `chat-jimmy.exe` as child processes from approved paths; restarts on crash with backoff; Stop stops only its own children | one window; still separate programs, as "features are the product" asks | Processor needs a careful runner (Windows Job Objects so children die with it, no adopting strangers); it's the code that was deliberately left out so far |
| **T3. One binary, several roles** | `thought.exe processor`, `thought.exe engine` … same file | one download; versions always match | blurs the repo boundaries; one build carries everything |
| **T4. Engine as a library inside Processor** | Processor links the Engine package and runs it in-process | simplest to run | the Engine stops being an independent feature; Thought Controller/TASS lose a stable service to talk to |

**Suggestion** [GUESS]: **T2**, with these rules:
- Children are started only from a path the owner approved, recorded with its SHA-256 (fits the install work on hold: "installed → approved path").
- Processor **never adopts or kills** a process it didn't start (the shell's rule already). If an Engine is already running, Processor attaches to it instead.
- Each child's stdout is captured into Processor's log view. This answers "the attach code is in another window": Processor can read it from its own child's output and **auto-attach** without the owner typing it, because Processor started that Engine. Discuss whether that's acceptable.
- Chat-Jimmy live stays an explicit owner switch ("Start Chat-Jimmy live — uses chatjimmy.ai"), never automatic.

Note: T2 is the "runner" that STATUS and the protected-actions boundary list as missing. If adopted, the "start/stop BLOCKED" wording in the shell becomes "start/stop **programs Processor launched**".

---

## 4. One local trust pattern for all three programs

We now have the same idea implemented twice. [KNOW] A serious prototype should have **one** pattern, written once, with the same words on every screen.

### Proposal: "Link" (working name)
1. **One shared package**, e.g. `Arameair/locallink` or a vendored folder: code generation, 8-character codes, TTL, lockout, a 256-bit token stored hashed by the issuer and 0600 by the holder, bearer header, and an optional HMAC proof on answers.
2. **Same UX everywhere:** "*Program A wants to link to Program B. Code: ABCD-EFGH*". It is used by Identity ↔ Processor, Engine ↔ Processor, and later Engine ↔ each source (TASS, Bridge…).
3. **When Processor started the program itself (T2),** linking is automatic. It can read the code from its own child, or pass a one-time secret via an environment variable at launch. The code-typing step only appears for programs you started yourself.
4. **Lock the Engine fully once linked.** [GUESS] The Engine's own page should need the owner too: either its own one-time `#key=` link like Identity's, or "open from Processor" with a short-lived ticket. Its v1 read routes should need a key once any link exists. Thought Controller would then link like everyone else.

### How this grows into Identity grants (three stages, no redesign)
| Stage | Who says "this identity may be processed" | Engine checks |
| --- | --- | --- |
| **S1 (now)** | the owner, by linking Processor to the Engine and switching Thought Processing on | its own link table + its per-identity switch |
| **S2** | Identity issues a **signed grant** (Ed25519, PASETO-style; see DESIGN_IDEAS A5) naming the Engine installation, identity, feature and expiry; Processor carries it to the Engine | the signature against Identity's public key, learned once when the Engine links to Identity |
| **S3** | the same, plus revocation: short expiry + refresh, or a revocation list Identity publishes | expiry / revocation |

Key point [INFER]: in S2, Identity **doesn't need to be running** for the Engine to check a grant. It only needs Identity's public key. That avoids "Identity restarted → everything is NOT_SIGNED_IN", which the pairing demo showed. [KNOW]

Related question: should Identity keep the GitHub sign-in across restarts? Options: Windows Credential Manager / DPAPI for a refresh token, or re-sign-in each time (today). With S2 grants this matters less [INFER], because a grant can outlive one sign-in.

---

## 5. A single, explicit state model for the feature

VISION lists the states in words. A serious prototype should compute **one status line** from evidence, and every screen should use the same names. Strawman, in priority order (the first that matches wins):

| # | State | Evidence (who says so) | Next step shown |
| --- | --- | --- | --- |
| 1 | NOT INSTALLED | no approved Engine path (Processor) | Install / choose the Engine |
| 2 | ENGINE STOPPED | port closed (Processor probe) | Start the Engine |
| 3 | ENGINE NOT LINKED | health OK, no key accepted (Engine) | Link |
| 4 | NO IDENTITY | Engine has none chosen | Create / choose |
| 5 | NOT PERMITTED | Thought Processing off, or (S2) no valid grant (Engine) | Turn on / ask Identity |
| 6 | NO SOURCES | no feed linked (Engine) | Connect a source, or type observations |
| 7 | WAITING FOR AI | worker not READY (Engine heartbeat) | Start Chat-Jimmy (N waiting, saved) |
| 8 | TEST FIXTURE | worker is the fixture (Engine) | *(striped, always visible)* |
| 9 | DEGRADED | failures / unreadable replies above a threshold (Engine metrics) | Look at Chat-Jimmy calls |
| 10 | RUNNING | all of the above OK | — |

[KNOW] Today's Processor computes most of these (next-step logic in `server.go`, `nextAction`), but as "next step" text rather than a named state. Naming them makes Copy Context, tests and docs line up.

---

## 6. Getting real observations in (sources / feeds)

Typing "Player entered Goldshire" proves the plumbing, not the product. [KNOW] Options for the first real source:

| Option | Path | Good | Bad |
| --- | --- | --- | --- |
| **F1. Source → Engine directly** | TASS / Bridge / ScreenBridge links to the Engine (same Link pattern) and POSTs observations; Processor *manages* the links (approve, pause, remove) and shows per-feed counts | Engine stays the one enforcer; Processor isn't in the data path | each source needs the small Link client |
| **F2. Source → Processor → Engine** | Processor relays | sources only know Processor | Processor becomes a data-plane hop and a second place data can leak or get lost |
| **F3. Engine pulls files** | Engine tails a log file the owner points at (e.g. a game log) | zero changes to other programs; great for replay | file formats vary; needs a parser per source |

**Suggestion** [GUESS]:
- **F3 first, as a replay**: one recorded file, played at real speed or faster. THOUGHT_PROCESS_DISCUSSION §5 already wants a replay to compare "smart loop vs dumb baseline".
- **Then F1** for live sources.
- In both, **"pause this feed"** is an Engine operation (per feed, per identity) that Processor calls. It never stops the source program (VISION: feed OFF ≠ source OFF).

Observation shape: the contract already carries `observation_id` and `text`. Add `source_id`, `observed_at` and an optional `sequence`, as INTERFACES.md and ARCHITECTURE.md propose, before any second source exists. [INFER] It's cheap now and painful later.

---

## 7. The AI worker as a replaceable part

[KNOW] The Engine talks to Chat-Jimmy only, one question at a time, through a copied client. The fixture proved the boundary is narrow (heartbeat + ask).

Ideas to discuss:
- **Make the boundary a named contract:** "worker protocol v0 = Chat-Jimmy's heartbeat + ask". Any program that speaks it can be a worker. Real workers must say `mode: live`; the fixture says `test-fixture` and is refused unless allowed. [KNOW] This already works; it just isn't written down as *the* worker contract.
- **A second real worker** to remove the single point of failure, e.g. a local model through Ollama behind the same protocol (a tiny adapter program). [GUESS] It's slower than chatjimmy.ai but needs no website, and gives a second opinion for evals.
- **Processor shows the worker**, but the Engine owns it. Processor shows worker health, answer times and refusals from the Engine's status. With T2, Processor can also start or stop the worker program.
- **Budgets live in the Engine** (calls per minute, per hour). Processor edits them through the contract and shows usage.

---

## 8. Processor's page: information architecture

[KNOW] The current page is one long column: next step, Engine card, selection, about, three status cards, Engine controls, diagnostics. [GUESS] For a serious prototype, three tabs:

| Tab | Contents |
| --- | --- |
| **Home** | the single status line (§5) + one next step · the live feed: observations flowing → states → Thoughts formed (compact) · "Open the Engine's page" for depth |
| **Sources & AI** | linked programs (Engine, Identity, sources, worker) with Link / Pause / Remove · worker health · budgets |
| **System** | this Processor's own facts, diagnostics (flow, logs, metrics), Copy context, versions and compatibility of every linked program |

Other UI ideas:
- **One timeline per observation** (click it): Processor sent `tp-obs-…` → Engine received → Chat-Jimmy request `…` (ms) → rated 7 → Trace → Thought. [KNOW] The IDs already line up end to end (observation id = Engine event id; Engine records Chat-Jimmy request IDs). It is mostly a view.
- **Server-Sent Events instead of polling** (1–3 s polling today). [GUESS] It's not urgent, but it makes "Processing…" feel live and lowers load.
- **Phone width.** [KNOW] It works today; keep it as a test.
- Retire the disabled "Engine controls" list once start/stop are real (T2) or clearly out of scope. Disabled buttons that explain themselves were right for the shell, but are confusing next to working ones.

---

## 9. Sessions: stop asking for a new code every run

[KNOW] Processor's operator code changes every start, by design, so a stale link can't be reused. In practice the owner opens a fresh `#code=` link after each restart.

| Option | Feel | Risk |
| --- | --- | --- |
| Keep per-run codes | safest, most annoying | — |
| **Remembered browser** | sign in once per browser; the HttpOnly cookie lasts e.g. 30 days and is stored hashed in the profile; "Sign out everywhere" button | a stolen cookie works until revoked (same as most local apps) |
| Open-from-launcher | T2 launcher / tray icon opens the browser with a fresh one-time link each time | needs the launcher (T2) |

[GUESS] Remembered browser + a revoke button is the usual trade-off for local tools. Decide it together with the Engine-page lock (§4), so all programs behave the same.

---

## 10. Versions, compatibility, packaging

- **Protocol strings already exist:** `thought-engine.processor/1-experimental`, `thought-processor.diagnostic/0-experimental`, `identity/0-alpha`. [KNOW] Processor could show a small **compatibility table** (linked program, its protocol, supported yes/no), refusing politely instead of failing oddly.
- **One download for testers:** [GUESS] a zip with Processor + Engine (+ fixture) + a `start.cmd`, built by a workflow from pinned commits, with `SHA256SUMS`. That's what was sent by hand on 10-26. It fits the release workflows that exist on experiment branches (`identity-v*`, `engine-v*`).
- **Windows CI:** [INFER] add `windows-latest` to `go test` in all three repos. Nothing has been run on Windows by an automated check yet [KNOW], and the owner's machine is Windows.

---

## 11. Testing a "serious" prototype

What already exists [KNOW]:
- unit and real-HTTP tests in all three repos
- a real-process loop test in the Engine
- a 26-check browser demo (`Thought-engine/scripts/demo_processor_loop.py`)

Ideas:
- **Make the 26-check demo the acceptance suite.** Run it in CI on Linux and Windows. Every change to any of the three repos must keep it green.
- **Scenario tests by state (§5):** one test per state that drives the system into it and checks the status line and next step. This doubles as documentation.
- **Replay evals for the thinking** (from THOUGHT_PROCESS_DISCUSSION Q11): the same recorded source, smart loop vs dumb baseline, with the owner rating Thoughts (useful / obvious / wrong / made-up). [GUESS] This is the only way to know the Engine is worth it.
- **A live-run checklist** for the owner (AGENTS.md style), with results recorded in `docs/` per run. Fixture results never count as live.

---

## 12. Three build paths, side by side

| | **P1. Harden what works** | **P2. Control room + launcher** | **P3. Full feature** |
| --- | --- | --- | --- |
| Scope | Tidy the page (§8 tabs), named states (§5), lock the Engine page (§4), remembered sessions (§9), Windows CI | P1 + T2 launcher (start/stop own children, auto-link, logs captured) + shared Link package | P2 + S2 signed Identity grants + first real source (F3 replay, then F1) + second worker |
| Windows to open | 3 | **1** | 1 |
| New risk | low | runner code (process supervision) | grants, revocation, source parsing |
| Size [GUESS] | ~2–3 days | ~1 week | ~3 weeks |
| Answers | "is it pleasant?" | "can a new person run it?" | "is it the product?" |

[GUESS] **P2 is the "serious prototype"**: it's the first one a new person (or the Grok bot) can run without a runbook, and it keeps every boundary we've argued for. P3 is the alpha after that.

---

## 13. Decision sheet for the discussion

| # | Question | Options | Suggested starting point [GUESS] |
| --- | --- | --- | --- |
| 1 | What is Processor? (§2) | A control room · B Thought app · C hub/supervisor | A + a slice of C |
| 2 | Who starts programs? (§3) | T1 by hand · T2 Processor supervises · T3 one binary · T4 in-process | T2 |
| 3 | Auto-link programs Processor started? (§3, §4) | yes · no, always type the code | yes, only for its own children |
| 4 | One shared Link package for Identity/Engine/sources? (§4) | yes · keep separate | yes |
| 5 | Lock the Engine's own page and v1 reads? (§4) | now · after P2 · never | with P2 |
| 6 | Path to Identity grants (§4) | S1 only · S2 signed grants · S3 + revocation | S1 now, S2 in P3 |
| 7 | Keep GitHub sign-in across Identity restarts? (§4) | no (today) · DPAPI/Credential Manager | decide with #6 |
| 8 | Which UIs survive? (§2) | Engine page · Thought Controller · Processor | Processor + Engine page; retire or merge Thought Controller |
| 9 | First real source (§6) | F3 replay file · F1 TASS/Bridge · F2 relay | F3 replay, then F1 |
| 10 | Sessions (§9) | per-run code · remembered browser · launcher-opened | remembered browser + revoke |
| 11 | Second worker (§7) | none yet · local Ollama adapter · other | after P2 |
| 12 | Canonical engine (D-01): Thought vs Thought-engine | — | Thought-engine (it runs the loop today); Thought as the rules/test reference |
| 13 | Build path (§12) | P1 · P2 · P3 | P2 |
| 14 | Update `STATUS.md` / `alpha` to reflect reality | now · when promoted | now (docs only) |

## 14. Questions only the owner can answer

- When a friend or your brother uses this, which program do you expect them to open first, and which window should they **never** need to see?
- Is it OK for Processor to start programs on your computer (T2), as long as it only starts what you approved and only stops what it started?
- Should a Thought ever reach you on its own (a notification or a Discord message through Communication Bridge), or only when you look?
- What's the first real source you'd actually want thinking about: TASS conversations, a game log, ScreenBridge text, or Discord?

— Claude
