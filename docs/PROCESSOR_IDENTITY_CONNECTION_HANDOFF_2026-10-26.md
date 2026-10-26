# Thought Processor ↔ Identity — connection handoff (2026-10-26)

**Purpose:** both teams build against the *same* verified connection, rather than two protocols invented separately. **This is documentation only.** It authorizes no code, grant or branch promotion, and it does not claim that Processor has integrated anything.

> **Corrected 2026-10-26 by the Identity team.** Each claim was checked against the running Identity candidate. What changed:
> 1. Updated the baseline commit.
> 2. Added the per-user locator, so Processor can find the paired Identity without knowing its profile folder.
> 3. Fixed the proof formula: it showed `\\n` where the message actually contains a newline, `\n`.
> 4. Listed which answers are unsigned **by design**. A 401 means "token refused", not "wrong service".
> 5. Split "not running" from "authority unavailable".
> 6. The generic feature install routes now exist; the Engine-only routes are behind an adapter.
> 7. Answered the code-boundary questions that Identity owns (see the end of this document).
>
> Copies of this file in `Arameair/identity` and `Arameair/Thought-Processor` are kept identical.

## Exact baseline and ownership

- **Identity implementation candidate:** [Arameair/identity `alpha-candidate-20261026`](https://github.com/Arameair/identity/tree/alpha-candidate-20261026).
  - Version `0.4.0-alpha`, API label `identity/1-alpha`.
  - Code commit at the time of this correction: **`dfbb24e`**. Check the exact SHA again when you implement, and see the [API changelog](https://github.com/Arameair/identity/blob/alpha-candidate-20261026/docs/API_CHANGELOG.md).
  - [Source API contract](https://github.com/Arameair/identity/blob/alpha-candidate-20261026/docs/API_V1.md) and [reference client](https://github.com/Arameair/identity/blob/alpha-candidate-20261026/cmd/identity-probe/main.go).
  - The [smoke self-test](https://github.com/Arameair/identity/blob/alpha-candidate-20261026/cmd/identity-probe/smoke.go) passes on Linux. **The Windows run is still pending.**
- **Roles:**
  - **Processor** is the human-facing application and a *client* of Identity.
  - **Identity** is the authority for canonical identities, service-pairing records and policy decisions.
  - Processor must not invent canonical identities, tokens or grants.
  - Each feature stays independent and must enforce protected authorizations where requests arrive at it.
- This is an **API contract for the current candidate**. It does not assert that Processor has integrated it. Keep the experimental branches separate and record exact compatible SHAs.

## Connection lifecycle — exact steps

1. **Locate, but don't trust yet.**
   - Identity binds **127.0.0.1:7790** by default; this can be overridden.
   - Every running Identity writes a **per-user locator**: `%LOCALAPPDATA%\Arameair\Identity\instances\<identity_system_id>.json`. On Linux and macOS it goes in `<user config dir>/arameair-identity/instances/`. It contains `base_url`, `port`, `pid`, `run_id`, `identity_system_id` and `profile`. Once paired, Processor picks **the file whose `identity_system_id` matches the one it saved**, which works whatever port or profile Identity is using.
   - `<profile>/runtime/service.json` holds the same data except the ID. Use it if Processor knows the profile folder.
   - Both locators are removed on a clean stop. A crash leaves a stale file, which you can detect because nothing answers at that address.
   - Locators and `GET /api/health` (public, unsigned) **only discover** Identity. They never establish READY or trust.
2. **First-time pairing, approved by a person.**
   - Processor generates its unique `processor_id` and saves it.
   - Processor sends `POST /api/v1/pair/request` with `{"processor_id":"tp_...","processor_label":"Thought Processor"}`.
   - Identity replies `202 CODE_SHOWN_IN_IDENTITY` and shows the one-time code **in its own console window and troubleshooting page**. The code is not in the API response.
   - Processor asks the person to type the code in, then sends `POST /api/v1/pair` with `code`, the same `processor_id`, and the label.
   - Success returns a bearer `token`, `identity_system_id` and `identity_version`.
   - The code works once and expires after 5 minutes. Requests less than 3 s apart get `429` with `Retry-After`. After 5 wrong codes, pairing locks for 1 minute and the current code is burned.
3. **Save and reconnect.**
   - Save the token and `identity_system_id` in Processor's local profile, readable only by the current user.
     - On Windows, a file under `%LOCALAPPDATA%` or `%USERPROFILE%` already has per-user ACLs.
     - **Recommended, not yet built:** protect the token with DPAPI (`CryptProtectData`, current-user scope).
     - Identity itself stores only SHA-256 hashes of tokens.
   - After a restart, reconnect with the saved credentials. **Never auto-approve a new or changed Identity installation**: a different `identity_system_id` means pairing again, with the person's approval.
   - `POST /api/v1/unpair` revokes the current pairing. The answer is still signed, and the token stops working right after. The Identity troubleshooting page can also unpair.
4. **Authenticate every protected API call.**
   - Send `Authorization: Bearer <token>` and a fresh random `X-Processor-Nonce`.
   - Identity's response carries `X-Identity-System-ID` and `X-Identity-Proof`.
   - Before trusting the response, verify both of these:
     - the installation ID is exactly the saved one;
     - `X-Identity-Proof` equals the formula below, compared in constant time, over the **exact raw body bytes**, not re-serialized JSON.
   - Follow the reference client's `call()`.
   - **Do not send a browser `Origin` header**; Identity rejects it on purpose. Use backend-to-backend localhost calls, not browser JavaScript.

   ```text
   X-Identity-Proof = hex( HMAC-SHA256( key = token,
                                        msg = "identity-v1" + LF + nonce + LF + raw_body ) )
   ```

   **Signed:** every answer that Identity's `/api/v1/*` handlers produce, including refusals such as 400, 404 and 409.

   **Unsigned by design** (do *not* call these "wrong service"):
   - the two pairing calls and `GET /api/health`;
   - **401** token refused;
   - **403** browser `Origin` present;
   - **421** wrong `Host` header;
   - **429** pairing too fast;
   - transport errors.

   Any *other* unsigned answer, a bad proof, or a mismatched installation ID is a hard failure. The nonce stops an old answer from being replayed.
5. **Read real records.**
   - `GET /api/v1/status` gives build, lifecycle and authority states.
   - `GET /api/v1/identities` and `GET /api/v1/identities/{id}` give the actual stable identity IDs (`ent_...`).
   - `GET /api/v1/features` or `GET /api/v1/identities/{id}/features` give eligibility and the per-identity feature state.
   - Report only the states that come back. `installed` and `running` are currently `UNKNOWN_TO_IDENTITY`; Processor supplies them from its own records.
6. **Drive GitHub sign-in from Processor.**
   - `GET /api/v1/github` returns the state.
   - `POST /api/v1/github/connect` starts the device flow. Show the returned `user_code` and `verification_uri` to the person, then poll until `VERIFIED`. There are also cancel and disconnect calls.
   - The GitHub credential stays in Identity's memory only, so in this candidate it **signs out whenever Identity restarts**.
   - For the Engine, `eligible_to_install` currently uses `repo-visibility-v0` and means *install eligibility only*. It is not publisher authenticity, not permission to run for an identity, and not consent to read source data.
7. **Install a feature through the generic routes.**
   - `GET /api/v1/features/{id}/eligibility` returns the decision plus `install_adapter`. It is `"NONE"` when Identity has no install rule for that feature.
   - `GET /api/v1/features/{id}/release?platform=windows-amd64` returns `tag`, `asset`, `size` and `sha256`. It returns `403 not_allowed` when the account isn't eligible, and `404 NO_INSTALL_ADAPTER` for features with no rule.
   - `GET /api/v1/features/{id}/release/download?platform=…&tag=…` streams the file. Identity's GitHub token never leaves Identity. A 409 means a newer release appeared since you asked.
   - Thought Engine is the only adapter today.
   - The older `…/thought-engine/access|release|download` paths remain for the earlier experiment only.
   - **The SHA-256 only proves the download isn't damaged. It does not prove who published the file.**
8. **Ask Identity for protected decisions when supported.**
   - `POST /api/v1/authorize` with body `{"identity_id":"ent_...","feature":"thought-engine","operation":"observe","source":"discord"}`.
   - **Every decision in this candidate is DENIED**, with `cache: none` and `valid_for_seconds: 0`, because grant creation doesn't exist yet.
   - Never route protected work silently, never infer authorization from pairing or installation, and never show a successful grant.
9. **Explicit exit.**
   - Only after the person chooses **Shut down managed workspace**: `POST /api/v1/shutdown` with `{"confirm":"shut-down-identity","reason":"person chose Shut down on exit"}`.
   - Identity answers `202`. Then verify that the PID exits and that both locators are removed.
   - If `lifecycle.ownership=EXTERNAL`, ask separately before stopping something Processor didn't start.
   - **Keep running** means sending no shutdown request.
   - Closing a browser tab is not an intentional exit.
   - Identity does not manage the Engine.

## Error/state mapping for a clean UI

| Situation | Processor must display / do |
|---|---|
| Never verified, and no locator or connection refused | **IDENTITY NOT RUNNING**. Keep retrying discovery. Block protected actions. |
| Was verified, now no answer or a timeout | **AUTHORITY UNAVAILABLE**. Block new protected actions and protected queued work. Use no cached allow. Retry. |
| Health answers but there is no pairing | **DISCOVERED · NEEDS PAIRING**, not Ready |
| Pair request returns 429 | Back off for the time in `Retry-After` |
| Wrong or expired code (401 on `pair`); 429 lockout | Explain it and offer a new explicit pairing. Honour the lockout. |
| A paired call returns 401 | **UNPAIRED / TOKEN REFUSED**. Requires a new approved pairing. Could also mean a different profile is now on that port: check the locator for your `identity_system_id`. |
| 403 from a paired call | A browser `Origin` was sent. This is a Processor bug: call from the backend. |
| HMAC mismatch, missing proof on a signed route, or wrong `identity_system_id` | **WRONG / UNVERIFIED SERVICE**. Fail closed and flag it in diagnostics. |
| `api` major label ≠ `identity/1` | **INCOMPATIBLE**, with nothing protected |
| Paired but authorization DENIED | **CONNECTED · ACTION DENIED**, with no protected action |
| Eligible to install but no grants | **INSTALL ELIGIBLE · NOT AUTHORIZED TO RUN / READ SOURCES** |
| Identity stopped or restarted | Keep the local installed state. Reconnect with the paired credentials (find the new port through the locator). Check grants again. Never use a cached allow. |

## P0 integration scope and handoff

**The Processor team owns:**
- its durable local `processor_id` and pairing credentials;
- the API client, including verifying the proof on each answer;
- a simple connection UI;
- the catalog and installer UI;
- tracking the verified installed state locally;
- correct status and error handling;
- owner test evidence.

**The Identity/Claude team owns:**
- the published API v1 and the reference test evidence;
- secure pairing and identity records for each installation;
- the feature install-eligibility answers and install adapters;
- the [API changelog](https://github.com/Arameair/identity/blob/alpha-candidate-20261026/docs/API_CHANGELOG.md);
- documented corrections wherever the contract can't support Processor.

**Any new endpoint for grants or feature registration needs an explicit cross-team proposal.** Processor must not invent one.

**Do not silently assume any of these:**
- `GET /api/health` or a locator file can be trusted;
- repository visibility is permission to run;
- a paired Processor is a separately authenticated per-person admin;
- Identity tracks installed or running state;
- a release checksum alone proves who published it;
- the GitHub login survives an Identity restart;
- the current `authorize` can ever answer ALLOW;
- feature service registration exists.

**P0 test sequence:**
1. Run a real, independent Identity and a real Processor.
2. Discover Identity (unverified at first).
3. The operator pairs once.
4. Get an authenticated `status`.
5. List, create and select a real identity.
6. Read the actual per-identity feature and eligibility states.
7. Restart in either order and reconnect, including after a port change through the locator.
8. Check that wrong profile, wrong token, unpairing and Identity being offline all fail closed.
9. Make the explicit exit choice and check process ownership.

After that, and separately, prove a real Engine install through `features/{id}/release` and that the installed state survives restarts, but only if an approved release exists. Test on Windows by hand: a fixture or smoke run is not a Windows integration PASS. The Identity side can be checked first on its own with `identity-probe smoke`, then with the steps in Identity's `AGENTS.md`.

## Questions at the code boundary — status

| Question | Status (Identity team, 2026-10-26) |
|---|---|
| How does Processor find the *correct* Identity data profile when neither program launched the other (not just port 7790)? | **Answered in code** (`be33293`): the per-user locator `…\Arameair\Identity\instances\<identity_system_id>.json`. Match the saved ID. If several are running before pairing, ask the person to choose; `identity-probe pair` lists them. |
| How does Windows store paired secrets securely? (Unix mode 0600 is not a Windows specification.) | **Answered for now:** keep the token under the user's own profile folder, where per-user ACLs already apply. **Recommended next step, owned by Processor:** DPAPI current-user protection. Identity stores only token hashes. Other programs running as the same user remain an accepted limit for this alpha. |
| Does install eligibility have a stable, generic, authenticated API, with the Engine-specific routes behind an adapter? | **Answered in code** (`be33293`): `GET /api/v1/features/{id}/eligibility\|release\|release/download`. Thought Engine is the only adapter; the old Engine paths are kept only for the earlier experiment. |
| How is publisher-approved release metadata authenticated, separately from the downloaded checksum? | **OPEN, not built.** Proposal: the publisher signs a release manifest (for example Ed25519 or minisign), with the public key built into Identity. Identity verifies it before reporting `sha256`. This needs an owner decision and a cross-team proposal. |
| When Claude extends API v1, how do teams coordinate exact request/response fixtures and test evidence before Processor implements? | **Process adopted:** every change goes into the [API changelog](https://github.com/Arameair/identity/blob/alpha-candidate-20261026/docs/API_CHANGELOG.md) with its commit. `docs/API_V1.md` has real captured responses. `identity-probe` and `go test` cover each route. Breaking changes would bump the label to `identity/2`. |

**Review gate:** both teams cite this document and [Identity API_V1](https://github.com/Arameair/identity/blob/alpha-candidate-20261026/docs/API_V1.md), raise contradictions in an issue, and confirm compatible commits. Only the owner approves a build or promotion.
