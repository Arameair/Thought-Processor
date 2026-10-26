# Thought Processor P0 — Windows owner trial

**Status:** experimental candidate, not owner accepted. Run this **without Project Controller**. Nothing in this document grants runtime/source access or authorizes a release. The app starts no other services.

## Exact integration baseline

- Processor branch: `experiment/p0-reusable-feature-onboarding-20261026`. Use `git rev-parse HEAD` to record the precise commit actually built.
- Identity service dependency: `Arameair/identity` branch `alpha-candidate-20261026`, protocol `identity/1-alpha`. The documentation's tested source was `df983ce`, and more recent same-branch additions include generic endpoints and per-user locators; record and validate the actual Identity HEAD you use.
- Engine release: `Arameair/Thought-engine`, an **approved published tagged binary** must exist for your platform before a real installation can succeed. No new release was published by this assignment.

## 1. Build Identity

In a clean PowerShell window:

```powershell
git clone --branch alpha-candidate-20261026 https://github.com/Arameair/identity.git identity-p0
cd identity-p0
git rev-parse HEAD
.\scripts\build-windows.ps1
.\identity.exe
```

Keep this window open. Default address is `http://127.0.0.1:7790`. This is an independent service; its first pairing code appears **here**, not in Processor logs.

## 2. Build and launch Processor

In a *second* PowerShell window:

```powershell
git clone --branch experiment/p0-reusable-feature-onboarding-20261026 https://github.com/Arameair/Thought-Processor.git processor-p0
cd processor-p0
git rev-parse HEAD
cd processor
go version
go vet ./...
go test ./...
go build -o thought-processor-p0.exe .
.\thought-processor-p0.exe
```

Default local UI: `http://127.0.0.1:7860/`. The Processor console prints a one-time local sign-in link or the operator code location. **Do not paste that private code/link into GitHub issues or exported logs.** If a custom port/data directory is needed:

```powershell
.\thought-processor-p0.exe -port 7861 -data "$env:LOCALAPPDATA\Arameair\Processor\P0"
```

Processor inherits the existing standalone, loopback-only local session, origin/CSRF checks, and profile locking. The older diagnostic screen remains available as `http://127.0.0.1:7860/diagnostics`.

## 3. First user journey — expected outcomes

1. Open Processor before or after Identity. If Identity is off, Processor should say **AUTHORITY UNAVAILABLE**. No privileged actions should be offered.
2. With Identity on, press **Connect to Identity → Request code**. Read the code from the Identity window, type it in Processor and press **Verify code**. Processor must report **VERIFIED** only after the signed Identity status was checked.
3. Create an identity (e.g., a separate `Trial Companion`) and record that Identity produced a durable `ent_…` ID with **zero authorized sources**.
4. Switch identities. No previous selected identity's feature/source data should remain on screen.
5. Press **Connect GitHub** and explicitly approve GitHub's device flow. After VERIFIED, approved feature cards should appear based on Identity's eligibility response. GitHub's token stays in Identity; restarting Identity will sign GitHub out.
6. If Thought Engine is eligible and an **approved actual release** is available, click **Review installation**, inspect its exact tag, asset, size and digest plus the publisher-authenticity limitation, then approve. Processor downloads through Identity's **generic** `/api/v1/features/{id}/release*` routes, verifies SHA-256/size and records the version. It does **not** start Engine or grant data access. If the release is missing or the publisher authenticity policy is insufficient, record BLOCKED rather than claiming a pass.
7. Use **Verify installed files** and restart Processor. The installed version and selected Identity remain separate facts. Installed status on startup initially reads **INSTALLED_UNVERIFIED** until rehash; this is deliberate.
8. Stop Identity and refresh. The interface must say **AUTHORITY UNAVAILABLE**. Unpair in Identity and confirm Processor refuses the old bearer; manual re-pairing is required.
9. Use **Exit workspace**. Since P0 never launches Identity or Engine, the default **Shut down managed workspace** stops Processor only; externally launched processes remain running. Closing just the browser tab stops none of them.

**Test account caution:** Identity's generic feature catalog still requires backend work to enforce private-feature discovery/enrolled tester visibility for arbitrary accounts. Processor filters unavailable private feature cards in P0, but this frontend filtering is **not a security boundary**. Do not certify general private feature confidentiality until Identity enforces it.

## 4. Acceptance record

| Check | Result before owner trial | Observed outcome |
| --- | --- | --- |
| Independent Processor and Identity launch on Windows | NOT RUN | |
| Pair once; HMAC verify; persist and reconnect | NOT RUN | |
| Identity create/select from canonical API | NOT RUN | |
| GitHub real user device sign-in | NOT RUN | |
| Private feature discovery enforced by Identity | BLOCKED | Identity needs a server-side privacy/entitlement contract |
| Generic real Engine release download and install | BLOCKED until a real approved publisher release is available | |
| Installed artifact persisted and verifiable on restart | NOT RUN | |
| Denied/expired pairing, fake proof, wrong installation | NOT RUN on owner Windows; automated fixtures separate | |
| Invalid checksum, interrupted download, existing install preserved | NOT RUN on owner Windows; automated fixtures separate | |
| Engine runtime feature/source grant | BLOCKED, deliberately out of P0 | |
| Separate external process remains alive on Processor exit | NOT RUN | |

Record Processor/Identity full commit SHAs, OS, Go version, exact commands and exit codes, actual screenshots with secrets removed, observed error messages, fixture versus real provider, and owner verdict **KEEP / REVISE / REJECT**. Do not alter `alpha`, `stable`, or `main` without a separate owner instruction.

## 5. Code ownership and limitations

- `processor/p0identity.go` — Identity v1 paired client (HMAC, fresh nonce, strict local addresses, locator and reconnection).
- `processor/p0routes.go` — session-protected API and product-visible state.
- `processor/p0installer.go` — generic feature-ID install preview, explicit tag approval, staging/hash, atomic current receipt, previous-version preservation.
- `processor/web/workspace.*` — simple standalone workspace. Developer Mode is view-only.
- `processor/p0_test.go` — local **fixtures**, not live production services.
- Existing experimental diagnostic shell and old `/api/te/*` installer routes are retained for comparison under `/diagnostics`; **P0's normal workflow uses new generic routes only**.

**Known unresolved trust limit:** The released binary checksum originates from GitHub's release rather than an independent publisher-signed attestation. Treat it as a verified transfer through a trusted paired Identity, not independently proven publisher authorship. If the owner requires cryptographically signed publisher provenance, installation acceptance stays BLOCKED until that mechanism is approved and implemented.
