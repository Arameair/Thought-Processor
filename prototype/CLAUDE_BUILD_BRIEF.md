# Claude implementation brief — The First Spark, v0 vertical slice

**Requested by owner · 2026-10-24 · IMPLEMENTATION INSTRUCTIONS, NOT IMPLEMENTATION**  
**Builder:** Claude, working as an independent developer and reviewer.  
**Source narrative:** [The First Spark](../README.md), especially [Screen Storyboard](../story/SCREEN-STORYBOARD.md), [Trust & Evidence](../story/TRUST-AND-EVIDENCE.md), and [Blueprint Seeds](../story/BLUEPRINT-SEEDS.md).  
**Expected code repository:** [Arameair/Thought-Processor](https://github.com/Arameair/Thought-Processor).  
**Build branch rule:** **Create your own fresh `experiment/...` from current `alpha`**, never build on `main`/`stable` or promote to `alpha`. This *story branch* is a read-only specification source, not the branch to which implementation is committed.  
**Candidate Engine:** [Arameair/Thought-engine `experiment/identity-grant-enforcement-20261024`](https://github.com/Arameair/Thought-engine/tree/experiment/identity-grant-enforcement-20261024). Read the [Engine integration runbook](https://github.com/Arameair/Thought-engine/blob/experiment/identity-grant-enforcement-20261024/docs/IDENTITY_AUTHORIZATION_EXPERIMENT_2026-10-24.md).  
**Identity design dependency:** [Owner GitHub bootstrap notes](https://github.com/Arameair/identity/blob/experiment/github-owned-bootstrap-notes-20261024/docs/GITHUB_OWNED_BOOTSTRAP_AND_FEATURE_ACCESS_2026-10-24.md); [Identity authorization issue #15](https://github.com/Arameair/identity/issues/15); [Identity first-experience issue #16](https://github.com/Arameair/identity/issues/16).  
**Test contract:** [Acceptance Test Matrix](ACCEPTANCE_TEST_MATRIX.md), [Expectations & Gates](EXPECTATIONS_AND_GATES.md), [Run Report Template](TEST_RUN_REPORT_TEMPLATE.md).

## Assignment — one sentence

**Build the smallest independently runnable Thought Processor that can truthfully show the selected identity and service state, safely ensure the expected Thought Engine is available, display the real Engine's health and diagnostic evidence, and prepare—but not pretend to complete—the Identity-approved observation experience from the story.**

Make it a *testable vertical slice*, not a feature gallery, imaginary demonstration, universal installer or production release.

### Success looks like

A human launches Processor's experimental build locally, opens one calm page, chooses **Ensure Engine Available**, and sees the **real** approved Engine either attach or start **once**. They can view exact service/build/instance info, what was verified versus only claimed, service logs/metrics on demand, and stop only the process owned by the test session. Missing/wrong Engine, Identity, GitHub or AI worker shows a reason, not a success animation. **Later**, once Identity implements the genuine grant decision protocol, they can see one authorized manual-source observation and its authentic receipt. Do not mark that future milestone complete now.

## Non-negotiable product boundaries

1. **GitHub is required for the normal full product first-opening experience.** A user must connect their own GitHub account, verify a repository that holds real starter software, preserve origin and credit, select a working entity, and satisfy a separate publisher feature access policy before normal product readiness. **Repository contents, license, bootstrap/fork/template mechanism and future updates are UNDECIDED**. You must NOT invent a publishable source bundle, bootstrap method, feature entitlement or approved release just to complete the UI.
2. For the **limited Processor–Engine engineering trial**, a developer can run Processor against a **real existing** installation/account and selected Engine without claiming this is a finished customer first-run. UI must label `DEVELOPMENT INTEGRATION TRIAL / FIRST-OPENING PREREQUISITES NOT VERIFIED`. Normal `Start Experience` cannot claim full product readiness until GitHub onboarding is legitimately implemented/verified.
3. **Publisher feature access is different from Identity local authorization.** The publisher can govern new private downloads/hosted access at a trusted boundary, but cannot promise remote erasure of source already delivered. A customer-editable boolean is **not** publisher evidence.
4. **Identity decides. Engine enforces. Processor connects, tests and explains.** Processor never mints Identity grants and never becomes the required production observation relay.
5. **No fabricated live data.** No seeded fake GitHub name, Engine running state, prewritten Thought, worker reply, timings, UUID, successful authorization, service status or log output. If unknown, show unknown. A typed example event is an explicitly **manual test**, not live game telemetry. Fixtures must be isolated to tests and labeled `NOT LIVE`.
6. **No worker dependency by brand.** The UI refers to `AI worker` and actual capability/readiness; do not hardcode a model, purchase APIs or quietly fall back to paid services.
7. **Do not use the old Thought-Controller app as a substitute for Thought-Processor.** The new Processor is a separate independently runnable component, even if its first form is minimal.
8. **No unauthorized cross-repo code changes.** Read Identity and Engine for compatibility. If they need alterations, document an exact handoff rather than silently modifying their branches. All new executable Processor code belongs on Claude's own new experimental branch from Alpha.

## Proposed implementation scope — stop at each gate

### Gate 0 — Reconnaissance and build contract (do this first)

- Inspect latest `Thought-Processor/alpha` and selected Engine test branch. Record exact SHAs, existing Go/tooling conventions and Engine status endpoints. The currently recorded references, **not guaranteed latest when you read**, are Processor Alpha `9fac9fa44cb020bb6f8aa6bcb1e9faeda6e0b47a` and Engine experiment `9ecd863645c24d3d7ad113426fefeeedbcbcd75e`. Re-read and pin your tested revisions.
- Verify the story's actual owner direction, especially GitHub mandatory product gate and evidence integrity. Read all [trust rules](../story/TRUST-AND-EVIDENCE.md).
- In your experiment branch, add a short `IMPLEMENTATION_NOTES.md` with **actual supported** Engine endpoints/fields, missing Identity endpoints, correct source and engine IDs, and a feature-state table. **Do not guess** capabilities or health fields.
- Run current repository tests where possible and record the baseline. Explicitly report failures that already existed.
- Propose the **minimal launcher ownership** consistent with the story: **one bounded session-scoped process manager** for the selected Processor/Engine, not an arbitrary command runner. The session must identify `started_by_session` vs `attached_external`. If integrating with Identity's future launcher isn't possible yet, create an internal adapter seam and label the limitation.

### Gate 1 — Processor runs, independent of Identity/Engine

- Build a small **Go-backed local Processor** (recommended because the existing Engine is Go) with a single calm browser workspace. A different small stack is acceptable only if it demonstrably reduces dependencies and stays easy to run on the owner's development machine.
- One documented startup command and local-only bind. Show Processor's **actual** protocol version, branch/full SHA (or `UNKNOWN` if unavailable), binary digest if verified, PID/run ID, port, uptime calculated from observed start and explicit data root.
- Identity, GitHub, Engine and worker are shown as **NOT CONNECTED / NOT CHECKED** until checked. A missing optional service does not crash Processor, and no invented demo entities are created on launch.
- Readiness endpoint for Processor itself returns only **its own** readiness and capabilities, not claims that other services are authorized. Browser UI closing must not automatically stop backend or Engine.

### Gate 2 — One real Processor ↔ Engine availability handshake

- Expose **Ensure Engine Available** as a deliberate operation via Processor backend. It calls a **bounded lifecycle adapter** with a reviewed local allowlist/manifest containing expected product, approved exact source SHA, executable absolute path, actual artifact digest, working directory, isolated data root, selected port and allowed arguments.
- **If matching Engine is already running, attach**, verify identity/protocol/build and report `ATTACHED` with `ownership=external`. Do not start another. **If installed and stopped, start exactly one** approved instance, supervise its process group, report `STARTING`, and separately verify Engine readiness.
- If no authenticated proof of service identity is available, report `REACHABLE / IDENTITY UNVERIFIED`, never `AUTHENTICATED`. Comparing a response's claimed ID/SHA to a local manifest is **not cryptographic authentication**; explicitly distinguish local executable/process provenance from service-pairing trust.
- Do not kill by port, process name or bare PID without checking ownership. If Processor is restarted, reconcile with a prior run to avoid duplicate Engine. Repeated clicks or concurrent Ensure must be idempotent.
- Keep launch operations **out of browser JS**. Backend verifies local user/approval, caller access, allowed executable, paths and arguments. Bind diagnostic endpoints loopback-only; no secrets in response, log or Copy Context.
- Actual Engine currently exposes `GET /api/status`, `GET /api/diagnostics` and entity/receipt endpoints, but **is not a secure multiuser diagnostic surface**. Limit the demo to same-OS test profile; protect Processor's own surface and record untrusted endpoint risks.

### Gate 3 — Useful, truthful operator workspace

- A small status strip: selected identity (only if truly known), verified account/repo or **not verified**, publisher availability or **unknown**, Engine process/build state, Engine connection/trust, Identity grant state, worker state.
- Single live Engine card with **Open original UI**, **Ensure/Reconnect**, **Inspect**, **Copy Context**, **Logs**, **Metrics**, **Stop Demo**. Each control must be wired and reveal *requested → accepted/refused → observed change*, or disabled with an explanation. Only one expanded preview.
- Real event/operation timeline: request ID; exact origin/source; actual response code; Engine receipt; Identity decision ID if present; worker send count/result only if actually observed.
- Copy Context must omit tokens, raw prompts, credentials and sensitive file paths/content by default. Show branch/SHA, route/port if safe, PID/run ID, elapsed operations, last-check times, errors and evidence provenance. Redact using allowlists; don't rely only on replacing strings that look like secrets.
- Distinguish feature states: `MISSING`, `STOPPED`, `STARTING`, `RUNNING`, `REACHABLE`, `AUTHENTICATED`, `COMPATIBLE`, `NO GRANT`, `AUTHORIZED`, `REVOKED`, `AUTHORITY UNAVAILABLE`, `WORKER UNAVAILABLE`, `UNKNOWN/STALE`. These are **dimensions**, not all aliases for "green".
- UI should have a simple, human-readable **Next safe action** derived from live state, not fixed canned directions.

**This Gate 3 is the first owner-visible demo candidate. Stop and request an owner test and verdict before expanding functionality.**

### Gate 4 — True Identity-authorized manual observation (blocked until dependencies exist)

- Read the Engine experiment's **draft** Identity `POST /v1/authorization/check` wire contract. It uses scope fields `entity_id`, `source_service_id`, `target_service_id`, action `thought.observation.ingest`, and nonce-bound signed decision receipt; **this is not yet a working Identity endpoint**.
- Do **not** implement a temporary `always_allow` Identity, share a production source token in the UI, or silently re-enable Engine's own local grant toggle. Never call local fixture authorization a live successful demo.
- Once the separate Identity team implements/verifies real enrolled services, protected owner grant issue/revoke, and the Engine passes genuine interoperability tests, integrate a **separate demo source service**. The source must use an authenticated identity **S**, not a user-edited JSON `source` field impersonating TASS.
- Owner grants S→E `thought.observation.ingest` for entity X via Identity. Demonstrate `ALLOW` and then live `REVOKE` with no Engine restart; wrong source/entity denied; Identity outage fails closed; queue rechecks before model work. Only a real worker result may be displayed as a Thought/Trace.
- Until then show `AUTHORIZATION INTEGRATION NOT READY` and keep the real `Send observation` control disabled. You may provide **read-only explanatory preview** clearly marked `DESIGN ONLY`; never a simulated successful transaction.

### Gate 5 — Customer first-run and publisher delivery (open research, NOT in this candidate)

- GitHub authenticates the actual user and verifies their **own** destination/source repository; publisher entitlement is issued by a publisher-controlled boundary; Identity manages local user/service/entity authority; all are different checks.
- Do not implement GitHub provisioning, license generation, fork/template/controlled copy, proprietary feature access or automatic updates until the owner selects real legal/technical terms. **Do not erase the user-owned repo requirement** from the future full product: show first-use `SETUP REQUIRED` if those prerequisites aren't truly complete.
- Do not include fake `Connected to GitHub` or invented publisher product cards in the engineering trial.

## Practical build order and outputs

1. Produce a compact **compatibility report** of actual Engine APIs and missing Identity APIs.
2. Implement Gate 1, test it, and provide a single exact launch instruction.
3. Implement Gate 2 and run attach/start/stop/error/idempotency tests against a **real Engine**.
4. Implement Gate 3, provide a working **owner trial** and evidence report.
5. List Gate 4 and 5 blockers without substituting stubs. Code for Gate 4/5 must await explicit owner approval and real dependencies.

**Required artifacts in Claude's own build branch:**
- Runnable source, documented CLI/start mode and safe test data path.
- README with **current actual supported capabilities** and visible limitations.
- Automated tests + real cross-process smoke evidence with commands, exit codes, timestamps, exact SHAs and system environment.
- `docs/TEST_REPORT.md` based on [template](TEST_RUN_REPORT_TEMPLATE.md).
- Proposed acceptance results linked to [matrix](ACCEPTANCE_TEST_MATRIX.md) with statuses `PASS / FAIL / BLOCKED / NOT RUN`. A written test is NOT a passed test.
- Copy Context example sanitized **from an actual run**, not a hand-created fake payload.
- Short human walkthrough showing exactly what button to press, what response should appear, and how to recover.
- A brief owner-facing traffic-light report: **green = verified actual pass; yellow = partial/blocked/unknown; red = failure/security issue**. Explain every item.

## Explicitly NOT part of this prototype

Do not build a complete publisher licensing platform, full GitHub starter creation, Memory, TASS/Discord integrations, screen capture, game feed, self-modifying code, paid API features, a feature marketplace, automatic autonomous observation feeds, full thought-wave redesign, all-OS service management or any permanent migration of data between `Thought` and `Thought-engine`.

Do not overwrite/delete local user data, start unknown executables, install auto-start services, send observations without owner approval, use an unapproved AI provider, merge Alpha/Stable/Main, or represent mocks as live connections.

## Cross-project coordination rule

This is a **Processor-led prototype**, not a request for Claude to rewrite Identity or Thought Engine. If a future phase requires their APIs, open/describe a precise cross-project change with required request/response, proof and failure behavior. Keep their approved branches untouched and never claim another team agreed or implemented something until verified.

## Definition of done for Claude's first delivery

**Done means Gate 1–3 have been executed and evidenced against a real selected Engine, with honest unverified/blocked statuses for GitHub, publisher and Identity.** Gate 4/5 remain clearly blocked or future work. If real Engine cannot start, deliver the smallest genuine Gate 1 result plus exact blocker and tests; do not invent "PASS" in the UI or report.

The owner decides whether the experience is intuitive and whether to keep, revise or discard the experimental candidate. No automatic promotion. A successful build isn't proof of a coherent companion, secure Identity integration or usable customer onboarding.

*AI-assisted plan prepared at the owner's request. Claude must independently verify all repository/code/environment claims before implementation.*
