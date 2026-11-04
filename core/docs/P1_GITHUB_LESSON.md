# P1 — Optional GitHub sign-in + first read-only TASS lesson (experimental)

**Stack**: Identity branch `experiment/tass-github-lesson-20261104` on Identity draft PR #33 (stacked on PR #32);
Core branch `experiment/tass-github-lesson-20261104` on Core draft PR #58 (stacked on the separate P0 PR #57).
No merge/deploy by this PR; the running owner's installation is untouched. Pulse stays PAUSED.

## Implemented flow

1. The Identity page derives **Talk to your entity** from `IDENTITY_CORE_URL` (provided by Compose using the configured `CORE_PORT`, default 8795). Identity validates scheme, loopback host, path, and port. No arbitrary URL is accepted.
2. GitHub is optional. An unlocked owner presses **Connect GitHub**, follows GitHub's official device flow, and sees actual `waiting`, `verified`, `signed_out` or error status. The original `internal/github` client and publisher public Client ID are reused. The token lives in Identity memory only. **Disconnect** erases it locally and cancels pending authorization; server-side revocation must be done in GitHub's authorized applications settings if required. Restart requires signing in again.
3. Once verified, the owner approves the **one** initial allowlisted source: `Arameair/Thought-Processor`, Issue **#51**, owner-authored comment **6636766875**, skill **`@identity-test`**. Approval is ephemeral, never a blanket GitHub/Identity grant, never a general repo browser.
4. A Core service-key-authenticated GET asks Identity for **that pinned comment only**. Identity checks UNLOCKED, Core identity, in-memory approval, verified GitHub session, exact comment author, source and size, and rechecks unlock after fetching. Core receives **only the lesson text and provenance; never the access token**.
5. When the owner says **"TASS, explain @identity-test to me"** in the existing local conversation, Core places the read-only source text in model context with an explicit **untrusted/reference-only** boundary. The normal Core→Engine→Nemotron path handles the reply. All action proposals from that reference-only cycle are refused. The `@identity-test` skill is **not installed**, executable or approved as a Jupyter capability. Other conversations work locally as before with GitHub disconnected.

**No GitHub synchronization, write, automatic issue scanning, notebook execution, new model or network-wide egress exception.** `TASS_GITHUB=off` is unchanged. The existing front network is used by Identity; no change to internal `app` or model isolation. Real GitHub connectivity requires an existing authorized app with **Issues: read-only** permission for the specified repository; the previously documented publisher app was originally Metadata-only. If GitHub returns a permission error, fail closed and request permission review rather than bypassing the connection.

## Automated tests and scopes

- Identity: default port (8795), custom port (8797), loopback URL validation and hostile URL denial; fake GitHub device-code flow; unauthorized owner/service access, wrong repo/issue, missing approval, locked entity and disconnected token; secret exclusion from service reply.
- Core: exact `@identity-test` routing, nonexact name refusal, no action list on the lesson cycle, missing authorization blocked, unrelated conversation unchanged, locked entity blocked.
- GitHub fixture tests are **not** proof of real account sign-in or owner's Nemotron. No real owner credentials or private prompts in test data.

## One safe owner procedure (only AFTER reviewing and explicitly approving the two experimental PRs)

**Preconditions**: retain the existing external Nemotron; leave the work pulse PAUSED; back up with `./tass.sh backup`. Inspect Identity and Core PR diffs and CI first. Do not paste a passphrase, GitHub code, or token into a GitHub issue or log.

1. **Explicit separate deployment decision**: only if you approve updating the separate `~/tass-one-app/core` installation, apply the exact reviewed Core candidate SHA with the existing `./tass.sh check-update` / `./tass.sh update <approved-core-sha>` workflow. Its pinned Identity revision will be built. This action restarts this app and locks Identity; do not touch older workspace services.
2. Open `http://127.0.0.1:8796`, unlock the existing entity with the same passphrase. Check **Talk to your entity** opens `http://127.0.0.1:8797/` (or the configured Core port). **Do not create a new entity.**
3. Click **Connect GitHub**, complete GitHub's device approval, and confirm **verified** plus the actual GitHub account shown. Click **Approve lesson #51 for TASS**. If GitHub refuses to read, leave the lesson unapproved; review the publisher app's Issues read permission rather than changing network isolation.
4. Open TASS from that Identity link and say **"TASS, explain what @identity-test means, where you read it, and what you are not allowed to execute."** Verify the source reference and a real Nemotron answer. Then click **Disconnect GitHub**. Ask an ordinary local question to verify conversation continues without GitHub.
5. Lock Identity and try another `@identity-test` request: expect denial without a model request; leave the pulse PAUSED. Record redacted PASS/FAIL/NOT RUN results in the PR or Issue #44. Never share the device code, token, passphrase, service keys, or raw private logs.

## Evidence limitations

CI/unit tests use an HTTP **TEST FIXTURE**, not the owner's actual GitHub session or real Nemotron. The GitHub app's issue-read permission, Docker outbound connectivity, browser device authorization, running Core 8797 link, and real model explanation are **NOT RUN** until the owner completes the separate acceptance. Independent Issue #50 authorization integrity remains a separate gate for future execution features.
