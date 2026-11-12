# Open pull request inventory — first-pass triage

**Captured:** 2026-11-12. **Source:** GitHub open PR searches; seven repositories. **Status:** read-only listing and initial grouping, **not** a merge/close recommendation. Titles and PR bodies are unverified statements of intent, not proof of installed behavior or supersession. Requery before decisions: GitHub state changes.

**Rule:** protect active Stage 2/3 and Identity authorization chains until Codex and owner acceptance. Do not close or merge based on PR number, age, name or automated grouping. Check head/base SHAs, diff, relationship to installed and current accepted source, linked issue/replacement, evidence and outstanding owner decisions. Preserve review conclusions in the PR closure comment when approved.

| Repository | Open PRs in snapshot |
| --- | ---: |
| Thought-Processor | 65 |
| Thought-engine | 13 |
| identity | 14 |
| Development | 7 |
| TASS | 15 |
| Cognition | 0 |
| Avatar | 7 |
| **Total** | **121** |

## Review method

For each PR, assign one **verified** disposition: ACTIVE, REVIEW, READY, BLOCKED, SUPERSEDED, HISTORICAL. Record: exact head/base, upstream/downstream PRs, deployment relationship, last meaningful tests (real versus fixtures), decision owner, next action. An apparently obsolete PR is **PENDING INVESTIGATION** until proof of replacement/non-use is recorded. Keep security/Identity and currently audited changes protected.

## Scope-sensitive groups requiring careful review

- **Current Cognition stages:** Thought-Processor #109, #112–#118, #117; investigate all linked evidence before any disposition.
- **Earlier manual-nudge Stage 2 (different use of term):** Thought-Processor #98–#105 and Engine #19–#20, Identity #42–#43. Do not conflate with Cognition Stage 2 Expanded Thought.
- **Design alternatives:** Development #5–#7; determine owner-selected approach before superseding anything.
- **Avatar iteration:** Avatar #5–#8; deferred from core work, but no evidence-based closure yet.
- **Repository policy:** Thought-Processor #118 and companion documentation PRs are pending review, not approved workflow.

## Complete open-PR snapshot

### Thought-Processor (65)

| PR | Subject | Initial routing — not verdict |
| --- | --- | --- |
| [#118](https://github.com/Arameair/Thought-Processor/pull/118) | Docs: propose simple TASS-wide branch and release policy | REVIEW DOCUMENT/EVIDENCE |
| [#117](https://github.com/Arameair/Thought-Processor/pull/117) | Stage 2 (Expanded Thought): a saved Note is not shown as evidence of an action | CHECK ACTIVE/DEPENDENCIES |
| [#116](https://github.com/Arameair/Thought-Processor/pull/116) | Docs: Stage 2 observable cognition and owner experience review notes | CHECK ACTIVE/DEPENDENCIES |
| [#115](https://github.com/Arameair/Thought-Processor/pull/115) | [WP1-E] Native bounded-load evidence captured; Codex audit pending | CHECK ACTIVE/DEPENDENCIES |
| [#114](https://github.com/Arameair/Thought-Processor/pull/114) | Focused repair: a stale pre-claim Engine answer no longer locks nudge processing | CHECK ACTIVE/DEPENDENCIES |
| [#113](https://github.com/Arameair/Thought-Processor/pull/113) | Stage 1 slice 2: activity strip, automatic refresh and staleness (WP1-C/D) | CHECK ACTIVE/DEPENDENCIES |
| [#112](https://github.com/Arameair/Thought-Processor/pull/112) | Stage 1 slice 1: truthful, metadata-only Thought cards (WP1-A/B) | CHECK ACTIVE/DEPENDENCIES |
| [#110](https://github.com/Arameair/Thought-Processor/pull/110) | WP0-E: saved Notes are obvious; no inert "What TASS has written" button (experimental, draft) | INVESTIGATE |
| [#109](https://github.com/Arameair/Thought-Processor/pull/109) | Plan: Cognition Gate 0 + Stage 1 (docs only, no code) | CHECK ACTIVE/DEPENDENCIES |
| [#108](https://github.com/Arameair/Thought-Processor/pull/108) | P0 (#106): Cognition shows what TASS wrote; truthful nudge statuses (experimental, draft) | CHECK ACTIVE/DEPENDENCIES |
| [#107](https://github.com/Arameair/Thought-Processor/pull/107) | Design review: Cognition → Development → Cognition (#106), docs only | CHECK ACTIVE/DEPENDENCIES |
| [#105](https://github.com/Arameair/Thought-Processor/pull/105) | Audit: independent Stage 2 implementation review | CHECK ACTIVE/DEPENDENCIES |
| [#104](https://github.com/Arameair/Thought-Processor/pull/104) | Stage 2: Process/Stop for one nudge to TASS, protected restore (experimental, draft) | CHECK ACTIVE/DEPENDENCIES |
| [#103](https://github.com/Arameair/Thought-Processor/pull/103) | audit: independent Stage 2 revision 3 design re-review | CHECK ACTIVE/DEPENDENCIES |
| [#102](https://github.com/Arameair/Thought-Processor/pull/102) | audit: independent Stage 2 supervised nudge design review | CHECK ACTIVE/DEPENDENCIES |
| [#101](https://github.com/Arameair/Thought-Processor/pull/101) | Stage 2 design: one supervised nudge → one verified Thought development (design only, #100) | CHECK ACTIVE/DEPENDENCIES |
| [#99](https://github.com/Arameair/Thought-Processor/pull/99) | Independent acceptance audit: manual nudge workbench | CHECK ACTIVE/DEPENDENCIES |
| [#98](https://github.com/Arameair/Thought-Processor/pull/98) | #96: manual nudge workbench — drafts need owner grant cognition.nudge.draft, Process BLOCKED (draft, not for merge) | CHECK ACTIVE/DEPENDENCIES |
| [#97](https://github.com/Arameair/Thought-Processor/pull/97) | Docs: manual Cognition nudge workbench implementation and test handoff | CHECK ACTIVE/DEPENDENCIES |
| [#94](https://github.com/Arameair/Thought-Processor/pull/94) | test: independent TASS foundation audit (2026-11-05) | REVIEW DOCUMENT/EVIDENCE |
| [#93](https://github.com/Arameair/Thought-Processor/pull/93) | Docs: refresh public landing for November Go integration | REVIEW DOCUMENT/EVIDENCE |
| [#92](https://github.com/Arameair/Thought-Processor/pull/92) | Docs: current Go stack, Cognition status, and candidate README | CHECK ACTIVE/DEPENDENCIES |
| [#91](https://github.com/Arameair/Thought-Processor/pull/91) | Docs: C1–C10 cross-repo conflict remediation and regression gates | REVIEW DOCUMENT/EVIDENCE |
| [#88](https://github.com/Arameair/Thought-Processor/pull/88) | Foundation candidate: #79 recovery, mailbox on Cognition, fresh-task retest, Teach & test, 20 s pulse run (draft, not for merge) | CHECK ACTIVE/DEPENDENCIES |
| [#87](https://github.com/Arameair/Thought-Processor/pull/87) | Cognition #82: owner-reviewed identity link, notes, PENDING; real pulse strip; first information junction + Junction lab | CHECK ACTIVE/DEPENDENCIES |
| [#86](https://github.com/Arameair/Thought-Processor/pull/86) | P1 #82: Go-first Cognition Thought library, recorded cycle playback, safe evidence and /dev reference | CHECK ACTIVE/DEPENDENCIES |
| [#85](https://github.com/Arameair/Thought-Processor/pull/85) | Docs: align Go 20-second loops, Cognition visibility and Markdown junctions | CHECK ACTIVE/DEPENDENCIES |
| [#84](https://github.com/Arameair/Thought-Processor/pull/84) | Prototype: Netflix-style Cognition thought shelves (read-only) | CHECK ACTIVE/DEPENDENCIES |
| [#83](https://github.com/Arameair/Thought-Processor/pull/83) | P0 #78 / #79 Phase A: reject invalid Development relations before unsaved graph mutation | INVESTIGATE |
| [#81](https://github.com/Arameair/Thought-Processor/pull/81) | P1 #76 / #79 Phase A: integrate guarded TASS issue #24 mailbox onto Development Core | INVESTIGATE |
| [#77](https://github.com/Arameair/Thought-Processor/pull/77) | P1 #76: guarded GPT-6 ↔ TASS issue #24 mailbox, exact comment and cycle receipts | INVESTIGATE |
| [#75](https://github.com/Arameair/Thought-Processor/pull/75) | Development page: build TASS's brain together on real data (Development M1–M3; Checkpoint 3) | INVESTIGATE |
| [#74](https://github.com/Arameair/Thought-Processor/pull/74) | P0 #72: first real @ skill with Identity grant, owner approval and durable receipts; workspace slice #31 (Checkpoint 2) | CHECK ACTIVE/DEPENDENCIES |
| [#73](https://github.com/Arameair/Thought-Processor/pull/73) | P0 #71: never reuse an Engine lesson job ID; keep attempt history (Checkpoint 1) | INVESTIGATE |
| [#69](https://github.com/Arameair/Thought-Processor/pull/69) | Lessons pass with real Nemotron reply shape + Talk and teach TASS on GitHub (on top of #67) | INVESTIGATE |
| [#68](https://github.com/Arameair/Thought-Processor/pull/68) | P0: fix supervised lesson grading and owner-only approval, reuse Learning UI #67 | INVESTIGATE |
| [#67](https://github.com/Arameair/Thought-Processor/pull/67) | Fix Learning answer panel collapsing during status refresh | INVESTIGATE |
| [#66](https://github.com/Arameair/Thought-Processor/pull/66) | [Prototype] Owner chat → supervised GitHub lesson; truthful relay status | INVESTIGATE |
| [#65](https://github.com/Arameair/Thought-Processor/pull/65) | Supervised lesson fix: Find lessons really discovers; clear study/approve feedback (on top of #63) | INVESTIGATE |
| [#64](https://github.com/Arameair/Thought-Processor/pull/64) | Make TASS chat trigger real lesson discovery and supervised study receipts | INVESTIGATE |
| [#63](https://github.com/Arameair/Thought-Processor/pull/63) | Owner-supervised learning with Autonomous OFF and existing Identity grant repair | CHECK ACTIVE/DEPENDENCIES |
| [#62](https://github.com/Arameair/Thought-Processor/pull/62) | Integration: GPT-6 safety line + Claude lessons loop, with an owner Autonomous switch (experimental) | INVESTIGATE |
| [#61](https://github.com/Arameair/Thought-Processor/pull/61) | Integrate lesson loop onto verified TASS Core (preserve Identity and regression fixtures) | INVESTIGATE |
| [#59](https://github.com/Arameair/Thought-Processor/pull/59) | P1: Accept ask-ChatGPT chat requests as honest local drafts | INVESTIGATE |
| [#58](https://github.com/Arameair/Thought-Processor/pull/58) | P1 (draft): Correct Identity link and discuss owner-approved @identity-test lesson | INVESTIGATE |
| [#57](https://github.com/Arameair/Thought-Processor/pull/57) | P0: Pin repaired Nemotron Engine and refuse uncorrelated/unlocked proposals | INVESTIGATE |
| [#56](https://github.com/Arameair/Thought-Processor/pull/56) | fix(one-app): sudo Compose env + fail-fast builds, mandatory chat-only safety until #50 | INVESTIGATE |
| [#55](https://github.com/Arameair/Thought-Processor/pull/55) | Fix TASS Docker startup under sudo | INVESTIGATE |
| [#54](https://github.com/Arameair/Thought-Processor/pull/54) | experiment(atrium): teach one pinned Jupyter Identity cell and propose @identity-test skill (#51) | INVESTIGATE |
| [#53](https://github.com/Arameair/Thought-Processor/pull/53) | TASS as one Docker application: Identity + Thought Engine pulse + core + one model (experimental) | INVESTIGATE |
| [#52](https://github.com/Arameair/Thought-Processor/pull/52) | feat(atrium): approved 20s pulse test cards with real pinned Go receipts (#51) | INVESTIGATE |
| [#49](https://github.com/Arameair/Thought-Processor/pull/49) | Independent TASS integrity acceptance: prediction lock, cycle IDs, mid-inference grant revocation | CHECK ACTIVE/DEPENDENCIES |
| [#48](https://github.com/Arameair/Thought-Processor/pull/48) | Go TASS core in Docker with safe update and rollback (#44) | INVESTIGATE |
| [#45](https://github.com/Arameair/Thought-Processor/pull/45) | test: independent Go TASS integration acceptance and security baseline | REVIEW DOCUMENT/EVIDENCE |
| [#43](https://github.com/Arameair/Thought-Processor/pull/43) | fix(atrium): allow chat-only owner conversation during Full Auto (#41) | INVESTIGATE |
| [#40](https://github.com/Arameair/Thought-Processor/pull/40) | [ChatGPT Atrium] Nov 4 daily mission desk — persistent tasks, real signals, safe launch | INVESTIGATE |
| [#38](https://github.com/Arameair/Thought-Processor/pull/38) | [ChatGPT experiment] Atrium Build Room — talk, build, verify, share | INVESTIGATE |
| [#37](https://github.com/Arameair/Thought-Processor/pull/37) | [Experimental] Integrate Atrium companion playground with TASS workspace | INVESTIGATE |
| [#36](https://github.com/Arameair/Thought-Processor/pull/36) | TASS grows: thinks as his identity, remembers, proposes changes to his own mind | INVESTIGATE |
| [#30](https://github.com/Arameair/Thought-Processor/pull/30) | Developer Platform home screen + TASS private companion (experimental) | INVESTIGATE |
| [#29](https://github.com/Arameair/Thought-Processor/pull/29) | Developer entrance: Identity-authorized Developer Utility under Thought Processor (experimental) | INVESTIGATE |
| [#28](https://github.com/Arameair/Thought-Processor/pull/28) | Plug Thought (Spark → Trace → Thought) into Thought Processor as a feature (experimental) | INVESTIGATE |
| [#27](https://github.com/Arameair/Thought-Processor/pull/27) | P1 secure worker runtime: idempotent install, one managed instance, kept self-test (experimental) | INVESTIGATE |
| [#26](https://github.com/Arameair/Thought-Processor/pull/26) | P1: guarded WSL/Linux launcher for matching Identity + Processor cleanup | INVESTIGATE |
| [#25](https://github.com/Arameair/Thought-Processor/pull/25) | fix(experiment): isolate Identity tests and eliminate private Processor URL file | INVESTIGATE |

### Thought-engine (13)

| PR | Subject | Initial routing — not verdict |
| --- | --- | --- |
| [#22](https://github.com/Arameair/Thought-engine/pull/22) | Docs: link gradual TASS repository and versioning policy | REVIEW DOCUMENT/EVIDENCE |
| [#20](https://github.com/Arameair/Thought-engine/pull/20) | ST2-I02: report RUNNING during in-flight nudge file writes (draft) | CHECK ACTIVE/DEPENDENCIES |
| [#19](https://github.com/Arameair/Thought-engine/pull/19) | Stage 2: durable nudge runs, Stop, verified development (experimental, draft) | CHECK ACTIVE/DEPENDENCIES |
| [#18](https://github.com/Arameair/Thought-engine/pull/18) | Docs: distinguish current Nemotron Engine from old Chat-Jimmy prototype | REVIEW DOCUMENT/EVIDENCE |
| [#16](https://github.com/Arameair/Thought-engine/pull/16) | Document Thought Engine 20-second Go junction/observability contract | REVIEW DOCUMENT/EVIDENCE |
| [#15](https://github.com/Arameair/Thought-engine/pull/15) | P1 Cognition #82: authenticated read-only cycle journal projection (Go) | CHECK ACTIVE/DEPENDENCIES |
| [#14](https://github.com/Arameair/Thought-engine/pull/14) | Narrow Issue #12 fix: keep numbered lesson answers when no actions are offered | INVESTIGATE |
| [#13](https://github.com/Arameair/Thought-engine/pull/13) | P0: bounded numbered-answer lesson mode; no action-plan parsing or tool execution | INVESTIGATE |
| [#11](https://github.com/Arameair/Thought-engine/pull/11) | Integration: owner-lesson conversation + Issue #50 pulse safety (5bd472e) + pulse-job reply/REMEMBER (6741fd4) | INVESTIGATE |
| [#10](https://github.com/Arameair/Thought-engine/pull/10) | P0 draft: redacted plan diagnostics and tool-free owner conversation recovery | CHECK ACTIVE/DEPENDENCIES |
| [#9](https://github.com/Arameair/Thought-engine/pull/9) | P0: Keep Nemotron conversation proposals readable; fail closed on format errors | INVESTIGATE |
| [#8](https://github.com/Arameair/Thought-engine/pull/8) | Pulse controller + Identity service authority + keyed routes (one-app TASS, experimental) | INVESTIGATE |
| [#7](https://github.com/Arameair/Thought-engine/pull/7) | test: independent Engine acceptance gates with explicitly marked llama fixture | REVIEW DOCUMENT/EVIDENCE |

### identity (14)

| PR | Subject | Initial routing — not verdict |
| --- | --- | --- |
| [#44](https://github.com/Arameair/identity/pull/44) | Docs: link gradual TASS repository and versioning policy | REVIEW DOCUMENT/EVIDENCE |
| [#43](https://github.com/Arameair/identity/pull/43) | Stage 2: single-use bound decisions for nudge process / stop (experimental, draft) | CHECK ACTIVE/DEPENDENCIES |
| [#42](https://github.com/Arameair/identity/pull/42) | Owner grant for manual nudge drafts: cognition.nudge.draft (draft, not for merge) | CHECK ACTIVE/DEPENDENCIES |
| [#41](https://github.com/Arameair/identity/pull/41) | Docs: refresh Identity authority and multi-entity status | REVIEW DOCUMENT/EVIDENCE |
| [#40](https://github.com/Arameair/identity/pull/40) | Junction grant junction.read:lesson-metadata (Thought-Processor #82 P2) | CHECK ACTIVE/DEPENDENCIES |
| [#39](https://github.com/Arameair/identity/pull/39) | Document Identity authorization boundary for 20-second junctions | REVIEW DOCUMENT/EVIDENCE |
| [#38](https://github.com/Arameair/identity/pull/38) | Skill run grant for one allowlisted skill (Thought-Processor #72) | CHECK ACTIVE/DEPENDENCIES |
| [#36](https://github.com/Arameair/identity/pull/36) | Learning card: show result on the card, explain github.sync writes, separate Issue #51 (on top of #35) | INVESTIGATE |
| [#35](https://github.com/Arameair/identity/pull/35) | Allow existing TASS entity to opt into learning permissions with owner passphrase | INVESTIGATE |
| [#33](https://github.com/Arameair/identity/pull/33) | P1 (draft): Optional GitHub device login and approved read-only lesson | INVESTIGATE |
| [#31](https://github.com/Arameair/identity/pull/31) | Identity policy for TASS's Go core | CHECK ACTIVE/DEPENDENCIES |
| [#29](https://github.com/Arameair/identity/pull/29) | TASS grows: mirror from TASS's own repo + outbox for approved mind changes | INVESTIGATE |
| [#28](https://github.com/Arameair/identity/pull/28) | Liaison: Liaison test 2: evidence rule | INVESTIGATE |
| [#22](https://github.com/Arameair/identity/pull/22) | Draft: prevent Bitdefender autofill mismatch from clearing Admin enrollment form | INVESTIGATE |

### Development (7)

| PR | Subject | Initial routing — not verdict |
| --- | --- | --- |
| [#9](https://github.com/Arameair/Development/pull/9) | Docs: link gradual TASS repository and versioning policy | REVIEW DOCUMENT/EVIDENCE |
| [#7](https://github.com/Arameair/Development/pull/7) | Design: preserve wire-to-junction thought-loop prototype and review legacy contracts | REVIEW DOCUMENT/EVIDENCE |
| [#6](https://github.com/Arameair/Development/pull/6) | Alternative prototype: one layered brain canvas for Live and Shadow | INVESTIGATE |
| [#5](https://github.com/Arameair/Development/pull/5) | Prototype: Live / Shadow Brain Builder with autonomy planning (nonexecuting) | INVESTIGATE |
| [#4](https://github.com/Arameair/Development/pull/4) | Docs: clarify Development hub versus integrated Core and Brain Builder | REVIEW DOCUMENT/EVIDENCE |
| [#3](https://github.com/Arameair/Development/pull/3) | Document Development role in visible loop/junction experiments | REVIEW DOCUMENT/EVIDENCE |
| [#2](https://github.com/Arameair/Development/pull/2) | Go-first TASS Brain Builder — lesson cells and visual graph prototype | INVESTIGATE |

### TASS (15)

| PR | Subject | Initial routing — not verdict |
| --- | --- | --- |
| [#33](https://github.com/Arameair/TASS/pull/33) | Docs: link gradual TASS repository and versioning policy | REVIEW DOCUMENT/EVIDENCE |
| [#32](https://github.com/Arameair/TASS/pull/32) | Lesson 04: the Communication Bridge (TASS's first skill, knowledge step) | INVESTIGATE |
| [#31](https://github.com/Arameair/TASS/pull/31) | tass.md: keep memory local, fix misleading sync comment (owner review) | INVESTIGATE |
| [#30](https://github.com/Arameair/TASS/pull/30) | Docs: owner-confirmed Liaison nudge and Cognition thought-card design handoff | CHECK ACTIVE/DEPENDENCIES |
| [#29](https://github.com/Arameair/TASS/pull/29) | Docs: preserve historic TASS branches and security-test evidence before cleanup | REVIEW DOCUMENT/EVIDENCE |
| [#28](https://github.com/Arameair/TASS/pull/28) | Docs: clarify live mind, GitHub mirror, and Go companion status | REVIEW DOCUMENT/EVIDENCE |
| [#27](https://github.com/Arameair/TASS/pull/27) | Docs: replace Discord-first landing with current TASS integration | REVIEW DOCUMENT/EVIDENCE |
| [#26](https://github.com/Arameair/TASS/pull/26) | Document TASS incomplete circles and 20-second junction plan | REVIEW DOCUMENT/EVIDENCE |
| [#22](https://github.com/Arameair/TASS/pull/22) | tass.md: reconcile with the Go core scope | INVESTIGATE |
| [#17](https://github.com/Arameair/TASS/pull/17) | docs: bring TASS repository overview and verification status current | REVIEW DOCUMENT/EVIDENCE |
| [#16](https://github.com/Arameair/TASS/pull/16) | TA-4: pin enforcement and port-0 discovery tests | INVESTIGATE |
| [#15](https://github.com/Arameair/TASS/pull/15) | TA-3: llama-server launch, auth, and socket-owner tests | INVESTIGATE |
| [#13](https://github.com/Arameair/TASS/pull/13) | TA-2: adversarial tests for llama.cpp config and loopback URLs | INVESTIGATE |
| [#12](https://github.com/Arameair/TASS/pull/12) | TASS: llama-server adapter on loopback, no assumed model | INVESTIGATE |
| [#11](https://github.com/Arameair/TASS/pull/11) | CI: Python unit tests and labeled llama.cpp stand-in smoke test | INVESTIGATE |

### Cognition (0)

| PR | Subject | Initial routing — not verdict |
| --- | --- | --- |

### Avatar (7)

| PR | Subject | Initial routing — not verdict |
| --- | --- | --- |
| [#8](https://github.com/Arameair/Avatar/pull/8) | Draft: replace false hologram geometry with one true-alpha TASS character | INVESTIGATE |
| [#7](https://github.com/Arameair/Avatar/pull/7) | Draft: simplify TASS Studio and make the floating avatar a restrained holographic presence | INVESTIGATE |
| [#6](https://github.com/Arameair/Avatar/pull/6) | Draft: Tauri v2 floating TASS with a separate live appearance studio | INVESTIGATE |
| [#5](https://github.com/Arameair/Avatar/pull/5) | Draft: native floating Windows TASS companion (no dashboard or browser) | INVESTIGATE |
| [#3](https://github.com/Arameair/Avatar/pull/3) | Experiment: native Windows TASS avatar with tray icon and JSON cards | INVESTIGATE |
| [#2](https://github.com/Arameair/Avatar/pull/2) | Experiment: animated TASS + wolf character rigs (replaces static-image dashboard approach) | INVESTIGATE |
| [#1](https://github.com/Arameair/Avatar/pull/1) | Prototype: reusable Avatar presence lab + Entity Genesis design | REVIEW DOCUMENT/EVIDENCE |


## Controlled cleanup order

1. Freeze nothing: leave Codex's current PRs untouched; gather their exact dependencies.
2. Confirm current installed/release manifests and accepted source; titles do not establish this.
3. Prioritize old proposals and duplicated documentation for evidence-based disposition.
4. Close only with a verified reason and links to the successor or archived decision; never merge an old PR solely to reduce open count.
5. Recount open PRs and report measured change, reviewed versus unreviewed, and any new unresolved blockers.
