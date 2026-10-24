# Identity → Engine lifecycle approval: missing contract and dependency request

**Status: BLOCKED.** Prepared 2026-10-24 on `experiment/processor-diagnostic-shell-20261024`. Human-directed, AI-assisted.
This is a **dependency request**, not an implemented interface and not permission to edit Identity, Thought-engine or a launcher. It names requirements; it deliberately does **not** invent endpoints, field names or a wire format (per [BUILD_INSTRUCTIONS Gate 0](https://github.com/Arameair/Thought-Processor/blob/be6904618a9533513746f8e9aaf353ece37498f8/prototype/BUILD_INSTRUCTIONS.md)).

## 1. What was inspected (exact revisions)

| Repository | Ref | Full SHA | What exists that is relevant |
| --- | --- | --- | --- |
| Arameair/Thought-Processor | `alpha` (base of this experiment) | `9fac9fa44cb020bb6f8aa6bcb1e9faeda6e0b47a` | Documentation only; no code |
| Arameair/Thought-Processor | `experiment/identity-approved-build-package-20261024` (spec) | `be6904618a9533513746f8e9aaf353ece37498f8` | STORY, BUILD_INSTRUCTIONS, ACCEPTANCE, START_HERE |
| Arameair/Thought-Processor | `experiment/processor-story-prototype-20261024` (earlier research) | `1c8ebc4c1d4ed0deded8e866968b5df6a7e02230` | Owner journey, brief, product expectations, test design |
| Arameair/identity | `alpha` | `8d08fec6629dc9cc35ea2266556572b86e5499d7` | Documentation only |
| Arameair/identity | `main` | `667ec872d48e45a2de761560fcd03713951c530a` | Documentation only; `docs/STATUS.md` lists grant issuance and authenticated registry as **DESIGN** |
| Arameair/identity | `experiment/first-admin-wizard-20261023` | `4686c94f7bdb84ebb8abf70e7409c87580dc461e` | Local Go app. `GET /api/health` → `{"app":"identity","version":…}` (public metadata). Owner "consent" records for a Thought **Controller** target. No decision issued to any other component |
| Arameair/identity | `experiment/identity-github-verification-20261023` | `e74c71e3f276fb1dff2c36d7dd8b624465768f25` | As above + read-only GitHub CLI check |
| Arameair/identity | `experiment/identity-controller-startup-20261023` | `4c28da23ac23aa85d9bd73f51a396c8d05caa253` | Identity **itself** approves and launches Thought-Controller from a local file record (path + SHA-256 + port). Guarded only by loopback Host + Origin. Not a decision another component can verify |
| Arameair/identity | `experiment/identity-controller-port-discovery-20261023` | `ccfb8940100310d387461195d8c4d899f0175c18` | Port scan 7778–7899 for a Controller marker; discovery only |
| Arameair/identity | `experiment/identity-local-v1-20261023`, `experiment/identity-real-manager-20261023` | `2b31e996706d8d4510964e4fa3a727b78bf1eab5`, `d060041dfc85276eea59d59071cb4b044f4bb9ab` | Identity cards / archive / GitHub CLI association; no authorization |
| Arameair/identity | `design-foundation`, `dynamic-feature-cards-notes`, `first-run-experience-brief`, `fresh-admin-bootstrap-design`, `github-owned-bootstrap-notes`, `processor-identity-connection-notes` experiments | `12bfb8d6`, `849e9de7`, `2f23e5f9`, `b95ea98d`, `8dc2384f`, `aed72fab` | Documentation only (zero `.go` files) |
| Arameair/Thought-engine | `experiment/thought-engine-v1-20261023` | `cb7abc0309b170bff85ac6f85b3eaaaf3f913018` | Engine with an Engine-local grant toggle; no service authentication |
| Arameair/Thought-engine | `experiment/identity-grant-enforcement-20261024` | `9ecd863645c24d3d7ad113426fefeeedbcbcd75e` | Ingest-only Identity check (below). No lifecycle or attach contract |

The Thought-engine repo has no `alpha`, `stable` or `main` branch at the time of inspection.

### What the Engine actually supports today (grant-enforcement experiment)

- **Start:** run the binary; `PORT` (default 7798) and `-data` select port and data root. No approval is checked at startup.
- **Stop:** SIGINT/SIGTERM. No stop endpoint, no ownership concept, no approval.
- **Read ("attach"):** `GET /api/status`, `/api/diagnostics`, `/api/entities/…` are guarded only by a loopback `Host` check. Any local process can read entity cards, grant state and the activity log. There is no caller authentication, so there is **no boundary at which an attach approval could be enforced**.
- **Ingest:** when started with `THOUGHT_IDENTITY_URL`, `THOUGHT_IDENTITY_TOKEN`, `THOUGHT_IDENTITY_VERIFY_KEY`, `THOUGHT_SOURCE_SERVICE_ID`, `THOUGHT_SOURCE_TOKEN`, the Engine authenticates one source by bearer token and asks Identity for a nonce-bound, HMAC-signed `thought.observation.ingest` decision on every intake and dequeue, failing closed. **The Identity side of this exchange is not implemented in any Identity branch.** It is an ingest contract only; it must not be reused for start/attach/stop (BUILD_INSTRUCTIONS forbids reusing an ingestion grant).
- Engine write routes accept requests with **no** `Origin` header (non-browser callers), so local programs can change Engine state without authentication.

### Conclusion

No producer exists for an Identity decision permitting Processor to start, operationally attach to, or stop a selected Engine, and no runner or Engine control boundary exists that could validate one. **Gate 2/3 integration is BLOCKED.** Processor ships with these actions disabled in the UI and refused server-side with HTTP 501 before any effect, and it contains no process runner.

## 2. The missing contract — requirements

Four **separate scopes**. Approval for one never implies another; a combined "ensure" needs explicit coverage of each branch it might take.

| Scope | Asker | Decider | **Validator / enforcer (must check the approval itself)** |
| --- | --- | --- | --- |
| Start one Engine instance | Processor (on an authenticated operator's request) | Identity | The bounded **runner** that spawns the process |
| Operationally attach to a running instance | Processor | Identity | The **Engine** (its read/control boundary) |
| Stop a session-owned instance | Processor | Identity (or a defined control policy Identity issues) | The **runner** that owns the process |
| Ingest observations | Authenticated source | Identity | The **Engine** at intake and before dispatch (exists experimentally on the Engine side only) |

Processor forwarding or displaying an approval is **not** enforcement. A frontend button state is not enforcement.

### Every lifecycle decision must bind, at minimum

- the **authenticated actor**: the human operator and the Processor instance making the request (not a display name, URL parameter or port);
- the **installation**, and the selected **entity/session** where the scope applies;
- the **intended action** (exactly one of start / attach / stop);
- the **target service** and the approved **build/manifest**: executable path and digest, product/protocol, pinned source provenance, isolated data root, working directory, allowed arguments, supported local endpoint;
- a **decision identity** usable for audit and correlation;
- **validity window / freshness**, and **integrity/authenticity** verifiable by the validator without trusting Processor.

### Behavior the contract must specify

- **Replay:** a decision is single-operation (or explicitly bounded); reuse for an unrelated or new launch is refused.
- **Concurrency / idempotency:** repeated or parallel requests for the same intended operation converge on one instance; no duplicate spawn.
- **Expiration and revocation:** how a validator learns of revocation; checked **at execution**, not when a page loaded. Loss of start/attach permission is **not** a kill instruction.
- **Timeouts and unavailability:** Identity unreachable, slow, malformed or unverifiable → fail closed, distinguishable from explicit deny (Processor already distinguishes 501 *no producer* / 503 *authority unavailable* / 403 *denied* in its design; only 501 is reachable today).
- **Key trust:** how the validator obtains and rotates the means to verify Identity's decisions. *Consideration (not a decision):* the Engine's ingest experiment uses one shared HMAC key; a shared symmetric key lets any holder mint decisions, so the owner may prefer a scheme where only Identity can sign.
- **Peer authentication of the Engine** to Processor (attach) and of Processor/operator to Identity. A port, a claimed SHA or a product string is not authentication.
- **Ownership records:** started-by-session vs. external, preserved across Processor restart with live process identity verification (bare PIDs are insufficient). Uncertain ownership → no kill, no claim.

## 3. Requests by owning project (separate assignments needed)

**Identity** (`Arameair/identity`)
1. Protected first-administrator enrollment and an authenticated operator ↔ Processor relationship.
2. A decision producer for the three lifecycle scopes above, with the bindings and behaviors listed; plus readback of the decision for audit.
3. The Identity side of the existing Engine ingest check (separate contract).
4. A public, minimal metadata endpoint that is stable across branches (today only some branches serve `GET /api/health`).

**Thought-engine** (`Arameair/Thought-engine`)
1. An authenticated read/control boundary that validates an Identity **attach** approval before serving private status, and a public minimal metadata endpoint (product, protocol, instance) that is safe before attachment.
2. Caller authentication on state-changing routes (today a request with no `Origin` is accepted from any local process).
3. A declared protocol/version so Processor can detect incompatibility.

**Runner / launcher** (owner to assign: Identity's "Start Experience" runner or a separate component)
1. One bounded runner for approved manifests only — no arbitrary commands — that validates the Identity start/stop approval itself, records ownership, and stops only verified session-owned descendants under a written cleanup policy.
2. Explicit runner lifetime (does it outlive Processor?).

**Owner decisions still open** (not implemented here): exact authority wire contract, admin recovery, starter contents/licence/update method, publisher entitlement service, GitHub outage policy, launcher packaging.

## 4. Conflicts found between documents

- The earlier [implementation brief](https://github.com/Arameair/Thought-Processor/blob/1c8ebc4c1d4ed0deded8e866968b5df6a7e02230/docs/build/CLAUDE_IMPLEMENTATION_BRIEF.md) Gate 2 says to read the Engine's `/api/status` for live read-only integration. The Identity-approved package allows only public read-only metadata before approval. Engine `/api/status` returns entity cards, grants and logs, so **this build follows the stricter rule and does not read it**; it only checks whether the Engine's TCP port accepts connections.
- The earlier brief allows an "engineering trial" to start/attach without Identity. START_HERE explicitly withdraws that; this build has no such mode.
- Identity's `controller-startup` experiment lets Identity approve and launch a component from a local file. It is a precedent for *Identity as launcher*, not a decision another component can verify, so it does not satisfy this contract.

## 5. What Processor already provides for when the contract exists

- A single `Authority` port (`processor/authority.go`) with one implementation that always reports *no producer*. Replacing it requires the agreed contract; no allow-all or developer-mode implementation exists.
- A control boundary (`POST /api/engine/{start|attach|stop|observe}`) that requires a local operator session, same-origin, JSON and CSRF, accepts **no** parameters (no paths, arguments or endpoints), records an operation ID, and refuses before any effect.
- Probe and status vocabulary that keeps *found*, *claims*, *authenticated*, *approved*, *attached* and *running* separate, with freshness.
