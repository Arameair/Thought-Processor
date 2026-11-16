# `thought.feature-api/1` — the P1 local feature contract

**Status:** experimental (P1, 2026-10-26). Implemented by Thought Processor (launcher), Identity (authority) and the trusted test feature in `processor/testfeatures/deterministic-worker/` (Python, standard library only). Any language that can do HTTP, JSON, SHA-256 and HMAC-SHA256 can implement it.

Three parties, three jobs:

| Party | Job | Never does |
| --- | --- | --- |
| **Identity** | Mints installation identities, issues grants (only from an Admin approval), signs request tickets, answers the live decision. | Run features. |
| **Thought Processor** | Installs, launches, watches and stops features; asks Identity for tickets; shows everything. | Decide permissions. |
| **Feature** (receiver) | Does its work **only** after verifying a ticket **and** getting a live ALLOW from Identity. | Trust a port, a body field or a caller's claim. |

## 1. Package and manifest (`feature.json`, schema `thought.feature-manifest/1`)

```json
{
  "schema": "thought.feature-manifest/1",
  "feature_id": "p1-deterministic-worker",
  "name": "P1 Deterministic Worker (TEST FIXTURE)",
  "version": "0.1.0",
  "description": "…",
  "publisher": {"name": "Arameair P1 test harness", "proof": "none (claim only)"},
  "source": {"repo": "Arameair/Thought-Processor", "path": "processor/testfeatures/deterministic-worker"},
  "trust_class": "TRUSTED_LOCAL_TEST_FIXTURE",
  "api": "thought.feature-api/1",
  "runtime": {"command": ["python3", "-I", "{package}/worker.py"], "requires": ["python3 (3.8 or newer) on PATH"],
              "startup_modes": ["manual", "on_demand", "always_running"], "default_startup": "manual",
              "default_idle_seconds": 30, "recovery": "manual"},
  "capabilities": [{"id": "worker.test.synthetic", "description": "…", "data_class": "SYNTHETIC", "requires_grant": true}],
  "standalone": {"health": true, "self_test": true},
  "information_sources": [],
  "files": [{"path": "worker.py", "sha256": "…"}]
}
```

Processor refuses a package when: unknown fields; any listed file's SHA-256 differs; a file is a symlink or **crosses a symlinked folder**; a path is absolute, contains `..` or odd characters; `runtime.command` names anything other than a listed package file (`{package}/…`) or a bare `python3`/`node`; `information_sources` is not empty; the folder wasn't trusted with `-trust-feature-dir`; or `trust_class` isn't `TRUSTED_LOCAL_TEST_FIXTURE`. **The publisher name is a claim, not proof.** A checksum next to a file proves integrity, not authorship.

**Trust = exact pinned artifact (P1 cleanup, Grok M1).** `trust_class` is only what a package says about itself, so it can't grant trust. A package may be advertised as installable, installed and started only if its **artifact digest** — `sha256("thought.feature-package/1\nmanifest <manifest sha256>\n" + "<file sha256>  <path>\n"…)`, files sorted by path — is pinned:

- built in: the reviewed `p1-deterministic-worker 0.1.0` fixture (`featpkg.BuiltinFixturePins` in Processor, `authority.BuiltinFixturePins` in Identity — the same digest; a unit test fails if the fixture changes without re-pinning);
- or by the operator: Processor `-trust-feature-artifact <sha256>` **and** Identity `-trust-fixture-artifact <sha256>`.

Checked by Processor before listing as installable, before staging, before registering and before every start (the installed files are re-verified against the approved digest), and independently by Identity before it registers an installation (`403 UNTRUSTED_EXECUTION_BLOCKED: ARTIFACT_NOT_PINNED`), when it commits it, and on every ticket and live check (`ARTIFACT_NOT_PINNED`). Processor also records the interpreter's resolved path (e.g. `/usr/bin/python3.12`) at install and refuses to start if it's gone. **This is trust in reviewed content, not isolation**: a pinned feature still runs with the user's full rights.

**Artifact digest** = SHA-256 of `"thought.feature-package/1\nmanifest <sha256(feature.json)>\n"` followed by `"<sha256>  <path>\n"` for each file in path order. Identity records it; the Admin approves it.

## 2. Launch (Processor → feature)

Processor starts `runtime.command` (with `{package}` replaced) in its **own process group**, working directory = the installation's data folder, with a **minimal environment** (PATH, HOME, LANG, TMP… — never Processor's own secrets) plus:

| Variable | Meaning |
| --- | --- |
| `TF_API` | `thought.feature-api/1` |
| `TF_INSTALLATION_ID` | durable `ins_…` minted by Identity (never changes on restart) |
| `TF_FEATURE_ID` | the manifest's feature id |
| `TF_LAUNCH_ID` | `lch_…`, new for every launch |
| `TF_LAUNCH_KEY_FILE` | file (0600) holding a fresh 256-bit hex **launch key**. Read it, **delete it**. |
| `TF_DATA_DIR` | this installation's private data folder |
| `TF_RUNTIME_DIR` | where to write `endpoint.json` |
| `TF_CREDENTIAL_FILE` | this installation's Identity credential (0600): `{installation_id, secret, identity_system_id, generation}` |
| `TF_IDENTITY_URL` | `http://127.0.0.1:<port>` of the paired Identity |
| `TF_IDENTITY_SYSTEM_ID` | the Identity installation that must sign every answer |

**Endpoint discovery:** the feature binds `127.0.0.1:0` (the OS picks a free port — no collisions), then writes `TF_RUNTIME_DIR/endpoint.json` atomically:
`{"api", "installation_id", "launch_id", "pid", "url": "http://127.0.0.1:<port>", "started_at"}`.
Processor accepts the endpoint **only** after `GET /v1/status`, signed with the launch key, names the same installation and launch. A port, PID or file alone is never trusted.

## 3. Launcher-signed calls (Processor ↔ feature)

Request headers: `X-TF-Time` (unix s, ±60 s), `X-TF-Nonce` (16–64 hex, never reused), `X-TF-Auth` = hex HMAC-SHA256(launch_key, `"thought.feature-api/1 request\n" + METHOD + "\n" + PATH + "\n" + time + "\n" + nonce + "\n" + hex(sha256(body))`).
Response headers: `X-TF-Installation-ID`, `X-TF-Launch-ID`, `X-TF-Proof` = hex HMAC-SHA256(launch_key, `"thought.feature-api/1 response\n" + nonce + "\n" + status + "\n"` ‖ raw body).

| Route | Auth | Purpose |
| --- | --- | --- |
| `GET /v1/health` | none | `{"api","state":"RUNNING"}` — liveness only, no identity details |
| `GET /v1/status` | launcher | installation, launch, pid, ready, **heartbeat_seq** (liveness — not an observation), provider readiness (separate), counters, latency |
| `GET /v1/capabilities` | launcher | declared capabilities; standalone health/self-test are **nonprivileged** |
| `POST /v1/self-test` | launcher | deterministic standalone test, labelled SYNTHETIC TEST DATA, no grant, no external call |
| `GET /v1/connections` | launcher | what Identity tells **this installation** about its own grants (signed) |
| `POST /v1/shutdown` | launcher | refuse new protected work, finish in-flight work, exit |

Browser `Origin` headers and non-loopback `Host` headers are refused.

## 4. Protected requests (requester → feature), with tickets

1. The requester (a paired Processor, or an installed feature with its credential) asks Identity: `POST /api/v1/tickets` `{destination_installation_id, capability, subject_identity_id, request_sha256}`. Identity issues a ticket **only** under an ACTIVE grant for exactly that requester → recipient → capability → subject. Otherwise a signed `403 {decision:"DENIED", reason}` — and Processor starts nothing.
2. Ticket: `tk1.<base64url(payload)>.<hex HMAC-SHA256(ticket_key, "identity/2 ticket\n" + base64url)>`, where `ticket_key = HMAC-SHA256(installation_secret, "identity/2 ticket key")` — only Identity and the **recipient** know it. Payload: `{v:1, tid, gid, gv, iss, req, aud, cap, sub, dig, iat, exp}`; valid 60 s, single use.
3. `POST /v1/actions/<capability>` with header `X-TF-Ticket` and body `{"request_id", "input"}`. Optional `X-TF-Nonce`: the launcher sends one and the feature signs **every** answer with the launch key (as in §3), so Processor can tell a real answer from one made up by whatever sits on the port after a crash.
4. The feature, in this order: verifies the ticket signature, issuer (`TF_IDENTITY_SYSTEM_ID`), recipient (`aud` = itself), capability (= path), expiry, and `dig` = sha256(raw body) → records `tid` in a **durable** single-use ledger (refuse a replay; refuse everything if the ledger can't be written) → asks Identity **live**: `POST /api/v1/authz/check` signed with its own installation credential (`X-Installation-ID`, identity/2 scheme), body `{ticket_id, grant_id, requester, capability, subject_identity_id, request_sha256}` → verifies Identity's signed answer → only on `200 {decision:"ALLOW"}` does the work.

| Feature answer | Meaning |
| --- | --- |
| `200 {result:"COMPLETED", observation}` | allowed and done; observation carries `decision_id`, `grant_id`, `ticket_id`, `data_class` |
| `401 TICKET_INVALID` (+ reason) | malformed/forged/expired/wrong recipient/capability/body |
| `409 TICKET_REPLAYED` | ticket already used |
| `409 TICKET_SPENT_DURING_OUTAGE` | this ticket was spent on an earlier attempt that Identity couldn't answer; still refused (single use is never relaxed) — ask for a new ticket |
| `403 DENIED` (+ Identity reason, decision_id) | Identity said no: revoked, expired, unknown ticket… |
| `503 AUTHORITY_UNAVAILABLE` (+ `ticket_spent: true`) | Identity didn't answer or answer wasn't signed by the right Identity — **never treated as allow**; the ticket is used up |

No decision is cached. Revocation is seen on the very next request. The live check consumes the ticket at Identity too.

**Who may revoke a grant (Grok M2).** Revoking needs no approval, but only from a party to the grant: the requester it names or the program that asked for it (`POST /api/v1/grants/{id}/revoke`, paired), or the destination feature itself (`POST /api/v1/installation/grants/{id}/revoke`, installation credential). Anyone else gets `403 REVOKE_NOT_PERMITTED` and must ask the Admin: `POST /api/v1/grants/{id}/revoke/requests` → Admin approves in Identity's page → `POST /api/v1/grants/{id}/admin-revoke {approval_id}`. Every revocation stores `revoked_by` and `revoked_at`.

**Expiry (Grok M3).** A grant request without `expires_in_seconds` gets a **1-hour** expiry. `until_revoked: true` asks for none; the Admin must then also tick *I understand this grant does not expire* in Identity's page, or approval is refused (and an unconfirmed approval can't be committed). Grants stored before this rule keep what they had.

**Test-only faults (Grok L5).** The test feature reads `data/TEST-ONLY-fault-injection.json` (crash/delay *after* an allowed request) **only** when Processor runs with `-test-feature-faults`, which sets `TF_FIXTURE_TEST_MODE=1` in the launch environment and shows a TEST MODE banner. Otherwise the file is ignored. The feature creates its ledger and observation files `0600` (umask `077`).

## 5. Lifecycle (Processor policy, per installation)

- **manual** (default for new installs): only the person starts it.
- **on_demand**: started for an already-authorized request (ticket first — no ticket, no start), stopped after the idle window when nothing is in flight. Never stopped during active work.
- **always_running**: Processor starts it once Identity is confirmed (not after the person stopped it).
- **Recovery**: process restart ≠ request retry. `manual` (default for experimental features) → `NEEDS_ATTENTION`. `bounded` → at most 3 process restarts in 10 minutes with back-off, then `NEEDS_ATTENTION`. A request whose feature crashed is `REQUEST_UNCERTAIN` and **never resent**. A process that dies before proving itself is `FAILED_TO_START`, never auto-restarted.
- **Processor crash**: the next Processor finds `runtime/launch.json`; it stops the old process **only** if it proves itself with the old launch key; otherwise it removes the record and kills nothing.

## 6. What this contract does NOT give you

- **No OS isolation.** A feature runs with the same OS user's rights. It could read other installations' credential files, Processor's files, or reach the network directly. Grants bind cooperating processes, not hostile ones — that's why only trusted local test fixtures run in P1.
- **No publisher authenticity / release provenance.** Untrusted third-party executables stay BLOCKED.
- **Transport is loopback HTTP**, authenticated by HMAC, not encrypted; another program running as the same user (or root) can observe it.
