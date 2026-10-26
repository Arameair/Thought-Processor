# Thought Processor ↔ Identity — concrete connection handoff (2026-10-26)

**Purpose:** Both teams implement against the *same* verified connection, not two independently invented protocols. **Documentation only; no code, grant, branch promotion, or claim of tested Processor integration.**

## Exact baseline and ownership

- Identity implementation candidate: [Arameair/identity `alpha-candidate-20261026`](https://github.com/Arameair/identity/tree/alpha-candidate-20261026), version `0.4.0-alpha`, API label `identity/1-alpha`; [source API contract](https://github.com/Arameair/identity/blob/alpha-candidate-20261026/docs/API_V1.md) and [reference client](https://github.com/Arameair/identity/blob/alpha-candidate-20261026/cmd/identity-probe/main.go). The candidate documents a tested code commit `df983ce` (verify exact current SHA when implementing).
- Processor: human-facing application and *client* of Identity. Identity: canonical identities, service-pairing records, policy/decision authority. Processor must not invent canonical identities, tokens or grants. Each feature remains independent and must enforce protected authorizations at its receiving boundary.
- This is an **API contract for the current candidate**, not an assertion that Processor has already integrated it. Keep independent experimental branches and record exact compatible SHAs.

## Connection lifecycle — exact steps

1. **Locate, do not trust yet.** Identity binds **127.0.0.1:7790** by default (override possible). If the Processor knows the Identity data profile, `<profile>/runtime/service.json` provides `base_url`, PID and run ID. This is a locator, *not* proof. `GET /api/health` is public and unsigned; checking it may discover the service but must never establish READY/trust.
2. **First-time, human-approved pairing:** Processor generates and persists its unique `processor_id`, then sends `POST /api/v1/pair/request` with `{"processor_id":"tp_...","processor_label":"Thought Processor"}`. Identity replies `202 CODE_SHOWN_IN_IDENTITY` and shows the one-time code **in Identity's own console/troubleshooting page**, not in the API response. Processor asks the user to enter the code. Processor sends `POST /api/v1/pair` with `code`, same `processor_id` and label. Success returns bearer `token`, `identity_system_id`, `identity_version`. Code is one-time, expires in 5 minutes; pairing rate limits and lockouts apply.
3. **Persist/reconnect:** Save the token and `identity_system_id` in Processor's local profile with restrictive permissions (candidate specifies mode 0600; on Windows use appropriate per-user ACLs and plan OS-protected storage, currently unimplemented). Subsequent restart/reconnect uses existing credentials; **do not auto-approve a new or changed Identity installation**. `POST /api/v1/unpair` revokes the current pairing. The Identity troubleshooting page can also unpair.
4. **Authenticate *every protected API call*:** Send `Authorization: Bearer <token>` and a fresh random `X-Processor-Nonce`. Identity's response contains `X-Identity-System-ID` and `X-Identity-Proof`. Verify exact installation ID plus `hex(HMAC-SHA256(token, "identity-v1\\n" + nonce + "\\n" + exact_response_body_bytes))` before trusting the response. Compare exact raw body, not reserialized JSON. Follow reference client's `call()` implementation; unsigned/wrong service/stale response is a hard failure. The two pairing calls are exceptions. **Do not send a browser Origin header**; Identity intentionally rejects it. Prefer backend-to-backend localhost calls, not browser JavaScript.
5. **Retrieve real records:** `GET /api/v1/status` for build/lifecycle/authority states; `GET /api/v1/identities` and `GET /api/v1/identities/{id}` for actual stable identity IDs (`ent_...`); `GET /api/v1/features` or `GET /api/v1/identities/{id}/features` for eligibility and identity-specific feature state. Only report the states returned; `installed` / `running` may currently be `UNKNOWN_TO_IDENTITY`.
6. **Drive GitHub from Processor:** `GET /api/v1/github` for state; `POST /api/v1/github/connect` for device flow, display returned `user_code` / `verification_uri` to the human, poll until VERIFIED; cancel/disconnect endpoints exist. The GitHub credential stays in Identity memory only and **signs out on Identity restart** in this candidate. `eligible_to_install` for Engine is currently `repo-visibility-v0`, *install eligibility only*; not publisher-authenticity, identity run access, or source-data consent.
7. **Ask Identity for protected decisions when supported:** `POST /api/v1/authorize` body `{"identity_id":"ent_...","feature":"thought-engine","operation":"observe","source":"discord"}`. **Every decision is DENIED in this candidate**, because grant creation does not exist yet. Never silently route protected work, infer authorization from pairing/installation, or show a successful grant.
8. **Explicit exit:** Only after a user chooses **Shut down managed workspace**, `POST /api/v1/shutdown` with `{"confirm":"shut-down-identity","reason":"person chose Shut down on exit"}`. If `lifecycle.ownership=EXTERNAL`, ask separately before stopping something Processor didn't start. Keep running = send no shutdown request. Do not treat browser tab-close as intentional exit. Identity does not manage the Engine.

## Error/state mapping for a clean UI

| Situation | Processor must display / do |
|---|---|
| No locator or connection refused | **AUTHORITY UNAVAILABLE**; wait/retry discovery; block newly protected actions |
| Health succeeds but no pairing | **DISCOVERED · NEEDS PAIRING**, not Ready |
| Pair request returns 429 | Back off per `Retry-After` |
| Incorrect/expired code | Explain, offer new explicit pairing; honor lockout |
| Auth returns 401 | **UNPAIRED / TOKEN REFUSED**; require new approved pair |
| HMAC mismatch or wrong `identity_system_id` | **WRONG / UNVERIFIED SERVICE**; fail closed and flag diagnostic |
| Paired but authorization DENIED | **CONNECTED · ACTION DENIED**; no protected action |
| Eligible to install but no grants | **INSTALL ELIGIBLE · NOT AUTHORIZED TO RUN / READ SOURCES** |
| Identity stopped/restarted | Preserve local installed state, reconnect with paired credentials, re-check grants; never use cached allow |

## P0 integration scope and handoff

**Processor team owns:** its durable local `processor_id` and pairing credentials; API client/response proof verification; simple connection UI; catalog & installer UI; local verified installed-state tracking; correct status and error handling; owner test evidence.

**Identity/Claude team owns:** published API v1 and reference test evidence; secure per-installation pairing/identity persistence; feature install eligibility reply; clear API version/change log; documented corrections where the existing contract cannot support Processor. **Any new grant/feature registration endpoint requires explicit cross-team proposal**, not invention by Processor.

**Do not silently assume:** `GET /api/health` is trusted; repository visibility is runtime permission; a paired Processor is a separately authenticated per-person admin; installed/running state tracked by Identity; a release checksum alone establishes publisher authenticity; GitHub login persists across Identity restart; current `authorize` can ALLOW; feature service registration exists.

**P0 test sequence:** real Independent Identity + Processor → discover unverified → operator pairing once → authenticated `status` → list/create/select real identity → read actual scoped feature/eligibility states → restart either order and reconnect → wrong profile/token/unpair/offline fail closed → explicit exit choice and process ownership. Then separately prove real Engine artifact install and local installed-state persistence if approved release exists. Test Windows manually; fixture/smoke is not a Windows integration PASS.

## Questions to resolve at the code boundary

- How Processor discovers the *correct* Identity data profile when neither service was launched by the other (not just port 7790).
- How Windows stores paired secrets securely (0600 Unix mode is insufficient specification for Windows).
- Whether installation eligibility has a stable, generic authenticated API; existing Engine-specific download routes are temporary and must be isolated behind an adapter.
- How publisher-approved release metadata is authenticated independently of a downloaded checksum.
- When Claude extends API v1, coordinate exact request/response fixtures and test evidence before implementation on Processor.

**Review gate:** Both teams cite this document and [Identity API_V1](https://github.com/Arameair/identity/blob/alpha-candidate-20261026/docs/API_V1.md), identify contradictions in an issue, and confirm compatible commits. Owner alone approves build/promotion.
