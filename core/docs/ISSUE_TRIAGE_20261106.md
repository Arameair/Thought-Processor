# Go TASS cross-repository issue triage — November 6, 2026

**Scope:** A complete review of the 72 GitHub issues open at the start of this pass in Thought-Processor (including Cognition), Thought-engine, Identity, TASS and Development. Classification is based on issue bodies and recorded discussions, not a new real-owner-machine test. The live GitHub issue state remains authoritative.


**ARCHIVAL VERIFIED (2026-11-06), with independent audit corrections below: 37 historical/superseded issues were annotated individually BEFORE closing.** The five repositories now have **35 open issues**: 22 marked NOW and 13 marked LATER. No issue body, comment, commit history, code, grants or running installation was deleted/changed. Every historical closure explicitly distinguishes a superseded approach from a completed bounded report; no open security, recovery or owner-acceptance blocker was closed.

| Repository | Initial open | Historical issues closed | Still open |
| --- | ---: | ---: | ---: |
| Thought-Processor (including Cognition) | 39 | 17 | 22 |
| Thought-engine | 8 | 3 | 5 |
| Identity | 10 | 5 | 5 |
| TASS | 14 | 12 | 2 |
| Development | 1 | 0 | 1 |
| **TOTAL** | **72** | **37** | **35** |

**NOW** = currently relevant security/correctness or first real Go integrated acceptance work. **LATER** = valid product/design/UX work without blocking the first stable Go loop. **ARCHIVE** = a completed narrow report or historical approach that should be closed with a dated explanation; not proof old defects were fixed.

**Triage:** 22 NOW, 13 LATER, 37 ARCHIVE. NOW issues are not all confirmed bugs; many are acceptance or architecture tasks.

## Important preservation rules

- Issue history, evidence, comments and source commits must remain available. For each ARCHIVE issue, post a dated closure note **before** closing it. `completed` means only a bounded delivered report/reply; otherwise use `not_planned` for a superseded path.
- Preserve owner-approved boundaries: Identity alone issues grants; every service enforces authorization, default deny; trusted, specifically approved first-party code until OS sandboxing is proven; no Markdown, GitHub, model or graph can authorize itself. Do not enable full autonomy or expose current localhost APIs to mobile/LAN.
- A fixture PASS, source edit or older P0/P1 owner test is not a PASS for the current version-pinned Go stack.

## Exhaustive 72-issue classification

| Repo | Issue | Triage | Reason and current successor |
| --- | --- | --- | --- |
| Thought-Processor | [#2](https://github.com/Arameair/Thought-Processor/issues/2) Decision required — one canonical Thought Engine for Thought Processor | **NOW** | Canonical Engine authority/data migration unresolved; C1 and Engine #17. |
| Thought-Processor | [#4](https://github.com/Arameair/Thought-Processor/issues/4) Contracts — independent log-source feature feeds and replaceable AI workers | **LATER** | Feature observation/replaceable worker contracts remain future generalized capability. |
| Thought-Processor | [#6](https://github.com/Arameair/Thought-Processor/issues/6) Design: Identity authority and feature self-registration for Thought Processor | **LATER** | Self-registration and independent feature design still relevant; not first Go loop gate. |
| Thought-Processor | [#7](https://github.com/Arameair/Thought-Processor/issues/7) Architecture review: Identity-first coordination and the Identity–Experience hypothesis (Grok / other AI critiques) | **ARCHIVE** | Oct architecture critique and alignment notes recorded; current authority analysis in #91; retain default-deny decision. |
| Thought-Processor | [#8](https://github.com/Arameair/Thought-Processor/issues/8) Plan the first Identity ↔ Engine authorization test through Thought Processor | **ARCHIVE** | Early Identity-Engine test plan superseded by Go Core #89, Engine #1/#17. |
| Thought-Processor | [#13](https://github.com/Arameair/Thought-Processor/issues/13) Design plan needed: install Thought Engine only if Identity allows (plan first, no code) | **ARCHIVE** | Oct repository-visibility eligibility plan/demo not a secure grant; use Identity #15 and Core #89. |
| Thought-Processor | [#15](https://github.com/Arameair/Thought-Processor/issues/15) Design gate: first usable Thought Processor workspace (Profiles → Catalog → Collections) | **ARCHIVE** | Earlier Profiles/Catalog UX gate replaced as active UI by Go Core /dev; owner acceptance #89. |
| Thought-Processor | [#16](https://github.com/Arameair/Thought-Processor/issues/16) Integration notes: how Thought Processor, Identity and features connect (Identity alpha-candidate-20261026) | **ARCHIVE** | Identity/1 alpha pairing notes superseded by identity/2 and Go service; Core #91. |
| Thought-Processor | [#17](https://github.com/Arameair/Thought-Processor/issues/17) P0 implementation candidate — Windows owner acceptance and release blockers | **ARCHIVE** | Historic P0 Windows build blocked by OS application-control policy; WSL first-run followed; current release #79. |
| Thought-Processor | [#18](https://github.com/Arameair/Thought-Processor/issues/18) Independent P0 test report (Linux): feature onboarding ae9e96b + Identity c8c0a54 — REVISE | **ARCHIVE** | Independent P0 Grok REVISE report delivered; historical negative findings preserved; not a blanket fix. |
| Thought-Processor | [#19](https://github.com/Arameair/Thought-Processor/issues/19) P1 plan — durable feature identities, utility onboarding and Fast AI Worker integration | **ARCHIVE** | Old P1 feature-lifecycle plan led to experimental code; product-neutral design now #90. |
| Thought-Processor | [#20](https://github.com/Arameair/Thought-Processor/issues/20) Voice-chat handoff — P1 feature identities, admin authority and connection design | **ARCHIVE** | Voice design handoff documented trusted-first-party-only, default deny and no arbitrary third-party code without OS isolation; carry forward #91. |
| Thought-Processor | [#21](https://github.com/Arameair/Thought-Processor/issues/21) Claude assignment — Build runnable P1 feature foundation from approved Alpha | **ARCHIVE** | P1 Claude experimental candidate delivered/fixture-tested, not owner accepted or promoted; current #88/#89. |
| Thought-Processor | [#22](https://github.com/Arameair/Thought-Processor/issues/22) Grok independent QA — adversarial review and reproducibility test of Claude P1 | **ARCHIVE** | Independent P1 Grok QA report delivered; corrective P1 scope #23; not production certification. |
| Thought-Processor | [#23](https://github.com/Arameair/Thought-Processor/issues/23) Claude P1 cleanup — resolve Grok QA findings before Alpha consideration | **ARCHIVE** | P1 cleanup candidate and fixture report delivered, not owner accepted; current #88/#89. |
| Thought-Processor | [#24](https://github.com/Arameair/Thought-Processor/issues/24) Portfolio follow-up — real P1 owner demo, licensing and verified evidence | **LATER** | Licensing, business boundary and verified portfolio demo deliberately deferred. |
| Thought-Processor | [#31](https://github.com/Arameair/Thought-Processor/issues/31) Owner acceptance: build TASS-connected local Developer Workstation; preserve identity through upgrade | **LATER** | Developer Workstation still a candidate capability; /dev exists but real service integration not approved. |
| Thought-Processor | [#32](https://github.com/Arameair/Thought-Processor/issues/32) P0 Owner usability: Windows/WSL one-click launcher + safe Check Updates / Update + audit sign-in, secrets and lifecycle | **NOW** | Owner launcher and update/recovery usability overlaps #79; reconcile distinct requests before closing. |
| Thought-Processor | [#33](https://github.com/Arameair/Thought-Processor/issues/33) Owner blocked at Establish Admin: missing or wrong anti-forgery token; needs verified path to real TASS chat | **ARCHIVE** | Old P1 Establish Admin token UI defect tied to replaced launcher; current token threat #47 still open. |
| Thought-Processor | [#34](https://github.com/Arameair/Thought-Processor/issues/34) [UX REVIEW — NO BUILD] TASS Thought Processor Brain / Service Map — complete proposal v1.0 | **LATER** | Brain/Service Map UX proposal partly overlaps /dev and Cognition; inspect later. |
| Thought-Processor | [#39](https://github.com/Arameair/Thought-Processor/issues/39) [DISCUSSION ONLY] Reconcile Workshop, Go Processor, Atrium and TASS mind source | **ARCHIVE** | Old Workshop/mind-source reconciliation replaced by C1-C10 audit #91; preserve launch hazards. |
| Thought-Processor | [#41](https://github.com/Arameair/Thought-Processor/issues/41) Atrium: allow owner conversation while Full Auto and Autonomy remain enabled | **ARCHIVE** | Older Python Atrium Full Auto chat-only fix fixture CI PASS (PR #43) but not owner accepted; current Go path #89. |
| Thought-Processor | [#42](https://github.com/Arameair/Thought-Processor/issues/42) Go TASS core: owner specification, Thought Engine cycle, local-first and security acceptance | **ARCHIVE** | Original Go Core spec yielded experimental code; actual live acceptance and restore #89/#79 still open. |
| Thought-Processor | [#44](https://github.com/Arameair/Thought-Processor/issues/44) Containerize verified Go TASS core and add safe Update/Rollback workflow | **NOW** | Docker Go deployment exists, but safe update/rollback acceptance unresolved; coordinate #79. |
| Thought-Processor | [#46](https://github.com/Arameair/Thought-Processor/issues/46) P1: Go /api/say returns 202 when bounded inbox silently drops an owner message | **NOW** | May acknowledge dropped inbox message with 202; reproduce and fix current code. |
| Thought-Processor | [#47](https://github.com/Arameair/Thought-Processor/issues/47) P1 security: localhost GET / leaks bearer-equivalent browser token for Go chat API | **NOW** | Browser token insufficient as authenticated owner/remote mobile session; security blocker. |
| Thought-Processor | [#50](https://github.com/Arameair/Thought-Processor/issues/50) P0 Go Core must reject unlocked or mismatched Engine proposals before executing actions | **NOW** | Locked prediction and cycle identity integrity need full execution acceptance. |
| Thought-Processor | [#51](https://github.com/Arameair/Thought-Processor/issues/51) Atrium: approved test cards and evidence receipts on 20-second pulses (GPT-6) | **ARCHIVE** | Earlier Python Atrium test cards path replaced by Go /dev and Engine loop; #70/#89. |
| Thought-Processor | [#60](https://github.com/Arameair/Thought-Processor/issues/60) P0: TASS learn from chat — one restricted notebook step and real pulse receipt | **NOW** | Restricted authorized notebook/lesson read-only execution with real receipt unproved. |
| Thought-Processor | [#70](https://github.com/Arameair/Thought-Processor/issues/70) [DESIGN HANDOFF] TASS ongoing 20-second predictive thought, five dynamic Markdown cards, GitHub teaching and bounded autonomy | **LATER** | Five-card predictive 20-second design intent; cannot call implemented before first loop. |
| Thought-Processor | [#71](https://github.com/Arameair/Thought-Processor/issues/71) P0 owner acceptance blocker: manual Study reuses prior Engine job ID after upgrade (HTTP 409) | **NOW** | Owner observed duplicate lesson job 409; retain regression until owner-history retest. |
| Thought-Processor | [#72](https://github.com/Arameair/Thought-Processor/issues/72) P0 skill invocation: TASS should resolve @skill and execute one Identity-authorized, supervised workspace operation | **NOW** | Supervised @skill operation and Identity decision must be demonstrated. |
| Thought-Processor | [#76](https://github.com/Arameair/Thought-Processor/issues/76) P1: Prove one GPT-6 ↔ Go TASS mailbox round trip via TASS issue #24 before hourly loop | **NOW** | Go mailbox pilot fixture PASS but owner real GitHub/Nemotron roundtrip pending. |
| Thought-Processor | [#78](https://github.com/Arameair/Thought-Processor/issues/78) Owner test: Development Platform 60661aa — partial; Brain relation validation defects | **NOW** | Owner Development PARTIAL and graph-validation bugs; retest repaired candidate. |
| Thought-Processor | [#79](https://github.com/Arameair/Thought-Processor/issues/79) P0: Single TASS integration branch, automatic verified backups and Restore Previous Update | **NOW** | Automatic backup and restore/rollback critical acceptance. |
| Thought-Processor | [#80](https://github.com/Arameair/Thought-Processor/issues/80) P1 Development UX: Copy reply, correlated test logs and safe evidence export | **LATER** | Sanitized copy/log/evidence UX is worthwhile post core stabilization. |
| Thought-Processor | [#82](https://github.com/Arameair/Thought-Processor/issues/82) P1 Cognition: Go-first persistent thought library, playback, evidence and Development bridge | **NOW** | Cognition integrated draft; real record playback and isolation acceptance pending. |
| Thought-Processor | [#89](https://github.com/Arameair/Thought-Processor/issues/89) Nov 6 owner start: prove one authorized agentic execution, then Brain Builder A/B outcome | **NOW** | Central real model/authorized skill/evaluation and graph A/B owner test. |
| Thought-Processor | [#90](https://github.com/Arameair/Thought-Processor/issues/90) Product portability audit: remove TASS-only runtime assumptions before second installation | **LATER** | Generic installs, no machine-specific assumptions, second synthetic identity before KYSSA. |
| Thought-engine | [#1](https://github.com/Arameair/Thought-engine/issues/1) Design gate: Engine must verify Identity grant before accepting or processing observations | **NOW** | Authenticated observation source and fail-closed receiver grants. |
| Thought-engine | [#2](https://github.com/Arameair/Thought-engine/issues/2) Build: working Engine prototype testable through Thought Processor (Attach + Send observation) | **ARCHIVE** | Old Chat-Jimmy ↔ Processor attach prototype built; current Nemotron Go stack #89 supersedes. |
| Thought-engine | [#3](https://github.com/Arameair/Thought-engine/issues/3) Design handoff — future Thought Engine foundation, decisions and first test gate | **ARCHIVE** | Future Engine/five-Spark design handoff preserved in TASS #25/TP #70; old build order no longer current. |
| Thought-engine | [#4](https://github.com/Arameair/Thought-engine/issues/4) P1: prove scoped ticket → Engine enforcement → AI Worker with identity/grant memory isolation | **NOW** | Scoped ticket → receiver → model worker and per-identity data enforcement not accepted. |
| Thought-engine | [#5](https://github.com/Arameair/Thought-engine/issues/5) [DISCUSSION ONLY] Canonical Engine choice, event semantics and first integration evidence | **ARCHIVE** | Nov 4 Engine choice/IDLE discussion superseded by TP #2, #91 and Engine #17. |
| Thought-engine | [#6](https://github.com/Arameair/Thought-engine/issues/6) P1 security hardening: legacy write endpoints unauthenticated; optional grant fails open | **NOW** | Unauthenticated legacy write routes and optional grants require hardening. |
| Thought-engine | [#12](https://github.com/Arameair/Thought-engine/issues/12) P0: Real Nemotron supervised lesson fails plan parser (REMEMBER only; no REPLY) | **NOW** | Real Nemotron parser failure had candidate repair; require preserved-owner-history regression. |
| Thought-engine | [#17](https://github.com/Arameair/Thought-engine/issues/17) P0: bind Engine execution and evidence to canonical Identity entity IDs, not display names | **NOW** | Replace display-name entity routing with canonical Identity ID and negative tests. |
| identity | [#9](https://github.com/Arameair/identity/issues/9) Identity GitHub account manager: real secure connection, read-only repos, then approved writes | **LATER** | GitHub account manager wider read/write scope beyond existing sign-in still needs consent/permissions. |
| identity | [#14](https://github.com/Arameair/identity/issues/14) Open design review: first-run Identity, organized feature storage, and background service | **ARCHIVE** | Oct P1 first-run/security design review superseded by current #15/#30 and Core #89. |
| identity | [#15](https://github.com/Arameair/identity/issues/15) Design gate: authoritative observation grant check, revocation and failure behavior | **NOW** | Source-authenticated, entity-scoped grant check and revocation at dispatch/use. |
| identity | [#19](https://github.com/Arameair/identity/issues/19) Alpha-candidate trial: setup PASS, new-user friction list | **ARCHIVE** | Oct early Identity setup friction trial and comparison delivered; current UI #37 not thereby fixed. |
| identity | [#20](https://github.com/Arameair/identity/issues/20) Alpha promotion gate — what the Identity candidate must prove | **ARCHIVE** | Old Identity alpha promotion checklist no longer applies to pinned Go candidate; Core #79/#89. |
| identity | [#21](https://github.com/Arameair/identity/issues/21) Portfolio polish follow-up — licensing, verifiable demos and repository presentation | **LATER** | Owner explicitly deferred licensing/product policy; not solved by docs cleanup. |
| identity | [#24](https://github.com/Arameair/identity/issues/24) Liaison test 1: what does the Team Liaison do? | **ARCHIVE** | Single Liaison question answered with three rules in DONE comment, not general team reliability. |
| identity | [#25](https://github.com/Arameair/identity/issues/25) Liaison test 2: evidence rule | **ARCHIVE** | Legacy Liaison evidence-rule edit reached proposed PR #28, not verified merged; not current Go Core. |
| identity | [#30](https://github.com/Arameair/identity/issues/30) [DISCUSSION ONLY] Multi-identity grants, Liaison mirror ownership and token boundaries | **LATER** | Independent KYSSA/privacy ownership design; consent and synthetic isolation later (#90). |
| identity | [#37](https://github.com/Arameair/identity/issues/37) UI cleanup: remove obsolete 'Approve lesson #51 for TASS' developer-test panel from normal Identity screen | **NOW** | Approve lesson #51 dev-test panel needs UI removal and owner verification. |
| TASS | [#2](https://github.com/Arameair/TASS/issues/2) Next prototype: natural Discord conversation with recent context | **ARCHIVE** | Discord-first natural chatbot prototype replaced by Go Core local Nemotron conversation. |
| TASS | [#3](https://github.com/Arameair/TASS/issues/3) Next pass: browser management screen for TASS; Discord remains chat | **ARCHIVE** | Discord-primary management UI/SSH approach no longer current; present Go Core has separate UI. |
| TASS | [#4](https://github.com/Arameair/TASS/issues/4) Claude handoff: standalone ScreenBridge visual capture prototype | **ARCHIVE** | Standalone ScreenBridge handoff remains historical optional-perception experiment. |
| TASS | [#5](https://github.com/Arameair/TASS/issues/5) Claude handoff: independent Communication Bridge (Discord + local chat) | **ARCHIVE** | CommunicationBridge/Discord/local chat handoff replaced by Go chat and restricted GitHub pilot. |
| TASS | [#6](https://github.com/Arameair/TASS/issues/6) Next TASS prototype: connect CommunicationBridge and ScreenBridge to existing agent | **ARCHIVE** | Browser-engine ScreenBridge/CommunicationBridge integration, B1 simulator and failure evidence preserved; local Go model path replaced it. |
| TASS | [#8](https://github.com/Arameair/TASS/issues/8) Hello from KYSSA — Jessica’s AI Chief of Staff 💜 | **ARCHIVE** | KYSSA greeting completed as correspondence only; not proof KYSSA installed or authorized. |
| TASS | [#9](https://github.com/Arameair/TASS/issues/9) Connection receipt test — Kyssa212 and TASS project collaboration | **ARCHIVE** | Old GitHub collaboration receipt lacks explicit ACK; archive UNVERIFIED, not PASS; future Identity #30. |
| TASS | [#14](https://github.com/Arameair/TASS/issues/14) Chief of Staff handoff: today’s setup for TASS and KYSSA as active project participants | **ARCHIVE** | Oct Chief-of-Staff team setup superseded by Go core-first priority; no KYSSA account action. |
| TASS | [#18](https://github.com/Arameair/TASS/issues/18) Read-only direct Workshop bridge test — TASS-DIRECT-1103-C | **ARCHIVE** | Old Workshop direct marker never independently correlated; archive NOT VERIFIED, Go mailbox #24 successor. |
| TASS | [#19](https://github.com/Arameair/TASS/issues/19) Workshop collaboration feed — owner-enabled execution evidence | **ARCHIVE** | Old Workshop collaboration feed replaced by separate Go mailbox #24; history preserved. |
| TASS | [#20](https://github.com/Arameair/TASS/issues/20) Talk with TASS — private GitHub companion conversation | **ARCHIVE** | Liaison conversation reply marker observed but not proof of Go Engine; now use #24. |
| TASS | [#21](https://github.com/Arameair/TASS/issues/21) [DISCUSSION ONLY] Nov 4 repository ownership, cleanup and integration evidence audit | **ARCHIVE** | Earlier repo/ownership inventory recorded; updated cross-repo conflict #91 and portability #90. |
| TASS | [#24](https://github.com/Arameair/TASS/issues/24) Talk with TASS (Go Core relay) | **NOW** | Restricted Go Core mailbox remains active, linked Processor #76. |
| TASS | [#25](https://github.com/Arameair/TASS/issues/25) OWNER DESIGN HANDOFF — Continuous local thinking, dynamic five-card 20-second batches, GitHub teaching and bounded autonomy | **LATER** | Five-card bounded autonomy/thought design is future, not currently executed. |
| Development | [#1](https://github.com/Arameair/Development/issues/1) Consolidate existing TASS Development Platform work without losing history | **LATER** | Design/prototype hub vs integrated /dev vs standalone Developer boundary unresolved; #89 C4. |

## What the older experiments taught us

**Discord/ChatGPT browser TASS + CommunicationBridge/ScreenBridge:** supplied early proof-of-concept messaging/perception and in some cases simulated tests, but no longer defines the runtime; local Go/Nemotron and scoped GitHub pilot are the accepted direction. Do not claim the old image/browser integration passed real local-model acceptance.

**Chat-Jimmy / Thought Controller:** the first Spark/Trace/Thought worker/visualization contracts were useful, but should not compete with the pinned Go Thought-engine as the canonical operational store. Historic evidence is retained; routing by display name remains a fresh concern (Engine #17).

**October P0/P1 feature installer, pairing, and Profiles/Catalog UI:** produced real user feedback and independent QA, but test results on old SHA/version are not proof of the modern Go Docker candidate. One prior Windows test failed because OS application-control policy blocked execution; that does not itself demonstrate a Go runtime bug.

**Python Workshop/Atrium and Identity Liaison conversations:** generated overlapping UI, scheduler and GitHub pathways. Current Go uses its distinct Core Sync and Engine Pulse plus restricted TASS issue #24 mailbox; a Liaison comment is not a verified local Go Engine cycle.

**Brain Builder/Cognition prototypes:** visual graph/edit/playback are not equivalent to permission-checked executed skills and independent outcome evaluation; verify through Core #89.

## Fix order and closure evidence

1. **Security and authority:** Processor #2/#47/#50, Engine #1/#6/#17, Identity #15 and #37.
2. **One measured loop:** Processor #46/#60/#71/#72/#78/#89, Engine #12, owner-tested real Nemotron decision/skill/grant/outcome.
3. **Recovery and observations:** Processor #32/#44/#79/#82; inspect backups and safe rollback, distinct Sync/Pulse timings.
4. **Transport:** Processor #76 and TASS #24; no broadened GitHub scopes, prove actual comment/cycle and dedup.
5. **Future product:** #90 synthetic second installation before real KYSSA; feature registry, Developer service, five-card thinking, mobile sessions and licensing later.

**NOTE:** Closing historical issues is administrative and does not delete records, merge code, grant permissions, fix old defects or change the running owner installation. Each closed issue keeps its exact original rationale and link to this register. If a historical approach becomes active again, reopen or create a narrowly scoped issue with new evidence.

## Independent review correction — November 6, 2026

**Documentation hazard fixed:** The prior `core/README.md` still included an obsolete rollback telling operators to delete local TASS data directories. The draft README now removes this dangerous legacy procedure and points only to pinned, separately approved backup/restore instructions. Do not use `start.sh` or file deletion from the 2026-11-04 prototype against the installed one-app Docker data.

**Five historical closure reasons corrected:** Thought-Processor #17, #20, #23 and #42, plus Thought-engine #2 were previously closed as `completed`, even though their full historical owner-acceptance/design conditions were not proved. The dated issue comments now explicitly mark them as `not_planned` / superseded, *not* owner-verified. Their successor security, execution and recovery gates remain open. Counts remain 37 archived / 35 open.

**Evidence scope:** All 72 issue bodies and the current GitHub states were inventoried; comments were **sampled/selected**, not every comment of every issue re-read. Documentation PRs remain draft; no live owner-machine tests were conducted in this cleanup.
