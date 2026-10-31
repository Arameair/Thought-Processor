# Thought Processor Team (`TP`) — team identity

**Status:** Proposed team identity, v0.1, October 31, 2026. No one is appointed; all five roles are **unassigned**. Shared rules: [TEAM_OPERATING_MODEL.md](TEAM_OPERATING_MODEL.md). Other teams: [README.md](README.md).

## Who we are

| | |
| --- | --- |
| **Team code** | `TP` |
| **Repository** | [Arameair/Thought-Processor](https://github.com/Arameair/Thought-Processor) |
| **Owner / final approval** | Dustin (Arameair) |
| **Supervising identity** | None assigned. Dustin directs. |
| **Roles** | Project Lead · Engineer · Test Architect · UX Architect · Team Liaison (all unassigned). The proposed **Product Designer** charter (`docs/PRODUCT_DESIGNER.md` on `main`) fills our UX Architect seat. |

**Our mission:** Be the one window a person opens to set up, install, approve, use and understand everything else, and make the system's boundaries **easy to see**: what is discovered, paired, installed, authorized or running, and what to do next when something fails.

We **coordinate**. We don't decide permissions (Identity does) and we don't do the features' work (they do).

## What we own

- The human-facing app: onboarding, Feature Library, install review, feature pages, identity selection, status and safe diagnostics.
- The **feature contract** `thought.feature-api/1` and manifest `thought.feature-manifest/1` (`docs/P1/FEATURE_API_V1.md`): how features are packaged, pinned by digest, launched, watched and stopped.
- Installing, starting, watching and stopping features, **only after** Identity approves.
- Asking Identity for single-use tickets on the person's behalf.
- The launcher (`tools/p0-launcher/`): Windows/WSL start, open, status, stop.
- The Developer Workstation (read-only, selected-file discussion with TASS).
- The coordinated alpha bundle: which Identity, TASS and Thought commits go together.

## What we don't own

- Grants, approvals, identities. That is `IDN`. We show them and ask for them.
- Any feature's data or reasoning. TASS conversations stay in TASS; Thoughts stay in Thought / Engine.
- Being a mandatory data relay. Features can talk to each other directly when the receiver enforces a grant.
- Taking over another program's process. Closing Processor doesn't mean closing other things.

## Where we are right now [KNOW, from the repo on 2026-10-31]

- `alpha` @ `e4b45a9` (2026-10-30): the coordinated TASS alpha. Identity `a582f8e` + Processor workstation + TASS `ae33d33` (0.2.0) + Thought `772200b`. Separate profile at `~/p1c/p1-tass-alpha`, Identity on 7798, Processor on 7868, IBM Granite through llama.cpp on `127.0.0.1:8080`. **Owner acceptance steps (real GitHub sign-in, Windows rendering, Granite conversation) are not done** (`docs/ALPHA.md`).
- `main` @ `1b1f496`: documentation; adds the proposed Product Designer role.
- Open owner issues: [#31](https://github.com/Arameair/Thought-Processor/issues/31) TASS-connected Developer Workstation · [#32](https://github.com/Arameair/Thought-Processor/issues/32) one-click launcher + safe updates · [#33](https://github.com/Arameair/Thought-Processor/issues/33) **owner blocked at Establish Admin (anti-forgery token)**. Also [#20](https://github.com/Arameair/Thought-Processor/issues/20), [#21](https://github.com/Arameair/Thought-Processor/issues/21).
- [INFER] #33 blocks every team's real end-to-end test, because nothing can be approved without an Admin. It is the top cross-team blocker until a fix is verified on Dustin's machine.

## What each role means on this team

| Role | On the Thought Processor team |
| --- | --- |
| **Project Lead** | Keeps one usable system-wide experience as the goal. Decides what goes into a coordinated alpha and which feature commits are bundled. |
| **Engineer** | Go app, launcher, feature launcher/watchdog, workstation. Never fakes a state when a dependency is offline. |
| **Test Architect** | Real-process acceptance: fresh profile, pair, Admin, install, grant, use, revoke, restart. Labels every stand-in (fixture, stand-in GitHub, stand-in model) as such. |
| **UX Architect** | First-use flow, feature cards, approval and denial screens, recovery paths. A non-technical person must be able to finish setup. |
| **Team Liaison** | Busiest Liaison: most handoffs pass through us because we sit between Identity and every feature. |

## Working with the other teams

| Team | We give them | We need from them |
| --- | --- | --- |
| **`IDN` Identity** | Requests naming exact identity + capability + feature installation; a place to show their approval prompts and denials | Stable API, clear denial reasons, the capability limits written down (e.g. 200-character descriptions) |
| **`TASS`** | Install, launch, watch, stop; ticket requests; the Talk to TASS screen; the workstation | A pinned package (`feature/feature.json` with exact SHA-256), feature-API compliance, honest status ("no local model yet") |
| **`THT` Thought** | Same as TASS: Library listing, install, run, the Send request box | A pinned package; capabilities that stay separate for read and write |
| **`TE` Thought Engine** | The attach/observation contract tested on `alpha-candidate-20261026`; a future P1 install path from tagged releases with checksums | Move from the old attach key to `thought.feature-api/1` + Identity grants ([TE #4](https://github.com/Arameair/Thought-engine/issues/4)) |

### Our contracts (other teams build on these)

Changing any of these needs a handoff to `TASS`, `THT`, `TE` (and `IDN` where tickets are involved) **before** it lands:

- `thought.feature-api/1` and `thought.feature-manifest/1`;
- the launcher's profile layout, ports and paths;
- how a feature's package digest is pinned and trusted.

### How to reach us

Open an issue in this repo titled `[HANDOFF HO-<YOUR TEAM>-TP-<date>-<nn>] …` with the handoff card. Our Liaison answers `RECEIVED`; our Project Lead answers `ACCEPTED`, `DECLINED` or `NEEDS-INFO`.

## Open questions we know about

- **Name clash.** TASS's team charters (TASS `teams` branch, `THOUGHT_PROCESSOR.md`) use "Thought Processor" for a **status-card role** on the TASS team, not this app. Suggest renaming that role (e.g. "Status Card Analyst") so handoffs don't get misrouted. Dustin's call.
- **Where team cards are displayed.** The Liaison design puts worker cards and communication history in Thought Processor. That would be a new Processor feature area and is not designed or scheduled.

## What we will never do

- Show "installed" or "allowed" without Identity's record behind it.
- Install, re-pin or update a feature without the person approving it in Identity.
- Promote anything to `alpha`. Dustin does that.
