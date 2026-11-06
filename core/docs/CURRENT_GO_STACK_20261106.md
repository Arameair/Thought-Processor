# Current TASS Go integration — source-of-truth map

**Reviewed:** November 6, 2026. **Status:** experimental integration candidate; **not** an accepted or deployed product. This document indexes current engineering work and supersedes older README **current-status claims**, not historical results. It does not authorize code, Identity grants, deployment or deletion.

## Current candidate and fixed references

| Concern | Authoritative location for this candidate | What is actually established |
| --- | --- | --- |
| Human interface, orchestration, supervised skills, lesson records, Development UI, Cognition projection | [Thought-Processor PR #88](https://github.com/Arameair/Thought-Processor/pull/88), Core `8e040b33e7a492f042a1f4449b8c7b013c227965` | Draft integrated candidate; source/fixture tests reported; **not owner-deployed or real-Nemotron-accepted** |
| Operational thought/cycle journals and 20-second **Engine** pulse | [Thought-engine PR #15](https://github.com/Arameair/Thought-engine/pull/15), `d69d3917d1601d806d4b2ba2254693e1adb92b6b` | Go Engine; fixture-backed journal endpoint and cycle evidence; canonical Identity mapping **unresolved** |
| Canonical Identity, grant and unlock service | [Identity PR #40](https://github.com/Arameair/identity/pull/40), `7da7f7c5bbecc42d3a7e9316a2037243eda10b88` | Experimental instance Identity and narrow junction grant; general multi-entity security **not accepted** |
| Companion's descriptive identity and approved lesson sources | [TASS `live`](https://github.com/Arameair/TASS/tree/live) | TASS-specific source/mirror; **not** the authority for credentials/grants or raw private operational journals |
| Development ideas, standalone *non-executing* Brain Builder experiment | [Development PR #2](https://github.com/Arameair/Development/pull/2) | Proposals and structural graph validation; no direct Identity/Engine execution from standalone prototype |
| Cognition (no separate active Cognition service) | [Cognition #82](https://github.com/Arameair/Thought-Processor/issues/82), [Core PR #87](https://github.com/Arameair/Thought-Processor/pull/87), [Engine PR #15](https://github.com/Arameair/Thought-engine/pull/15) | Core-hosted read-only Go projection of real Engine records; first identity link/owner live run still pending |

**Critical detail:** Engine `Pulse.Tick` is a work/job coordinator; Core `SyncOnce` is a separate periodic synchronization/lessons/junction process. Two 20-second timers do **not** prove a closed autonomous thinking loop. The proposed five-card thought batches remain design, not implemented.

## How to navigate current work

- **Owner acceptance / today's order:** [#89](https://github.com/Arameair/Thought-Processor/issues/89): first audit nightly results, verify installed full SHAs, safe backup/restore, one real authorized read-only skill, Cognition evidence, then Brain graph A/B.
- **Release safety:** [#79](https://github.com/Arameair/Thought-Processor/issues/79): update, failure recovery and data restore. Candidate has **fake-Docker fault-injection evidence**, not owner real-Docker acceptance.
- **Identity routing defect:** [Thought-engine #17](https://github.com/Arameair/Thought-engine/issues/17), [Identity #30](https://github.com/Arameair/identity/issues/30); engine can still resolve entities by name. A safer Cognition link is not the underlying Engine authorization solution.
- **Conflict remediation:** [draft PR #91](https://github.com/Arameair/Thought-Processor/pull/91), [#89 register](https://github.com/Arameair/Thought-Processor/issues/89). C1–C10 are **open engineering decisions** until separately proved.
- **Portability:** [#90](https://github.com/Arameair/Thought-Processor/issues/90): neutral entities/config and synthetic second install, without connecting KYSSA.
- **Cognition evidence:** [Cognition status](COGNITION_STATUS_20261106.md); fixture-browser tests do not certify real local Nemotron use.
- **Owner session:** [PR #88 owner runbook](OWNER_SESSION_FOUNDATION_20261105.md). All command instructions apply only to the explicitly version-pinned candidate and require owner authorization before update.

## Architectural ownership

**Identity** owns identity IDs, authorization and revocation; **Thought Engine** owns actual cognitive records and cycle/prediction/outcome journal; **Core** is the only integrated TASS app/UI and handles supervised lessons, registered skills and their own receipts; **Development** proposes/tests; **Cognition** reads and explains source-backed records. Neither Markdown, GitHub comments, Browser UI, graph, model replies, nor a developer tool confers permissions. The TASS repository is entity-specific descriptive material, not an executable runtime or separate grant database.

## Old material: retain history, stop treating it as current

The independent **Thought**, **Thought Controller**, **chat-jimmy**, **legacy Team Member**, **Discord-first TASS**, **Developmnt** spelling, and standalone **developer** Granite workbench are historical/reference or future *candidate* capabilities; **not the approved current Go one-app runtime**. Do not follow their launcher/port/model directions to operate the current stack. Keep links to them only in explicitly titled historical design/test records where provenance matters. Do **not** delete repositories, issues, evidence or messages during documentation hygiene.

## Evidence labels and releases

Use **IMPLEMENTED (candidate)**, **FIXTURE PASS**, **OWNER TEST PASS**, **NOT RUN**, **KNOWN DEFECT**, and **SUPERSEDED/HISTORICAL** distinctly. Never label a feature resolved because a draft PR exists. Confirm the installed stack directly from `./tass.sh status`; a GitHub branch does **not** prove what is running. Only a separate owner decision permits merging/deploying or changing Identity grants. Any app client (including eventual mobile) needs a designed authenticated API: localhost browser tokens are not safe remote sessions.

**Documentation maintenance rule:** Link to this dated map from each active repository landing page; correct the map when a new owner-accepted pinned candidate supersedes this one. Leave older evidence reachable through Git history and archive references without resurrecting old directions.
