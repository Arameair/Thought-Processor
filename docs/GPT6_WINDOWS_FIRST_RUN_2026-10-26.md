# GPT-6 — frictionless first-run Windows owner guide

**Branch:** `experiment/gpt6-frictionless-onboarding-20261026`  
**Origin:** owner-controlled `alpha` SHA `9fac9fa44cb020bb6f8aa6bcb1e9faeda6e0b47a`. Independent GPT-6 product experience; reuse of the earlier GPT-6 P0 backend explicitly credited in [EVIDENCE](GPT6_DEVELOPMENT_EVIDENCE_2026-10-26.md).  
**Status:** experimental. The linked CI build is not an owner-accepted release or an independently signed publisher binary.

## Quick start: no Go installation required for Processor

1. Visit [GPT-6 first-run CI builds](https://github.com/Arameair/Thought-Processor/actions/workflows/p0.yml). Select the **latest green workflow run for the GPT-6 frictionless branch**. Scroll to **Artifacts** and download `GPT6-Thought-Processor-Windows`. GitHub may require sign-in to download artifacts. This is a CI test binary, **not a published release**.
2. Extract the ZIP somewhere you control. Run `thought-processor.exe` directly. On Windows, Processor opens your default browser to its local private workspace. If browser launch fails, copy the private `http://127.0.0.1:7860/#code=…` link shown in the local Processor window; do not paste it into messages or issues.
3. If Identity is already running, Processor should locate it. Otherwise start your *existing* independently installed Identity application (`identity.exe`). Processor explains why it cannot connect and lets you retry. **P0 does not download, install or launch Identity automatically.** The owner must first provide Identity separately.
4. Press **Request connection code**. Read the one-time code from **Identity's own window or troubleshooting page** and enter it into Processor. You only need this first-time human approval. Processor verifies the proof and installation ID before continuing.
5. Choose one of your real identities, or create a new Companion, Person, Business or Project. Identity owns that record; no source permissions are granted.
6. If the feature needs GitHub eligibility, choose **Connect GitHub**. Open the **github.com** link, enter the displayed temporary device code and approve GitHub yourself. The GitHub token remains inside Identity and is not stored in Processor. Identity restart requires GitHub sign-in again.
7. When eligible, Thought Engine appears as a feature card. Choose **Review installation** to inspect actual release tag, Windows binary filename, byte size, SHA-256 and publisher-provenance warning. Choose **Approve installation** only if you trust the release source. If no approved real release exists, the UI reports the actual reason; **do not call that installation complete**.
8. Processor saves an installation receipt under its own profile and can verify the on-disk bytes. Installing a binary does **not** start or register Engine or authorize any identity/source.
9. Exit Processor and launch it again. It should **reconnect to the same paired Identity** without asking for the first code. If Identity is unavailable, protected operations must remain blocked.

### If Identity is not installed

The current Identity API candidate is not bundled as an installer with this experiment. For an owner/developer machine with Go and Git installed, build it in its **own** terminal:

```powershell
git clone --branch alpha-candidate-20261026 https://github.com/Arameair/identity.git identity-trial
cd identity-trial
git rev-parse HEAD
go build -o identity.exe .
.\identity.exe
```

Compatible tested Identity SHA for this candidate's real-process smoke: `dfbb24e9999b7f9bbb252cff82702b18f9b3e4c1`, API `identity/1-alpha`. For a reproducible version use `git checkout dfbb24e9999b7f9bbb252cff82702b18f9b3e4c1` before building. Starting Identity does **not** grant data access.

### Alternative: build Processor from source

```powershell
git clone --branch experiment/gpt6-frictionless-onboarding-20261026 https://github.com/Arameair/Thought-Processor.git processor-gpt6
cd processor-gpt6
git rev-parse HEAD
cd processor
go version
go test ./... -count=1
go build -o thought-processor.exe .
.\thought-processor.exe
```

If the default port is occupied or you want an isolated test profile: `.\thought-processor.exe -port 7861 -data "$env:LOCALAPPDATA\Arameair\Processor\GPT6Trial"`. Run `.\thought-processor.exe -open=false` to prevent auto-opening a browser.

## The guided experience: illustrated text walkthrough

**Screen 1 — Connect.** The single large next action reads *Find Identity again* when offline, or *Request connection code* when a local Identity is found but unverified. An explanation distinguishes local discovery from trust. Technical port selection lives under a collapsed **Identity is running, but wasn't found?** control. First approval is code-in-Identity → type-in-Processor; it is never silently scraped.

**Screen 2 — Identity.** Existing identities appear in one selector; creating a real identity is an alternative, not a mandatory form. The screen explicitly says each identity begins with no feature/source permissions.

**Screen 3 — Features.** A contextual GitHub sign-in card is shown only as needed for eligibility. Each actual allowed feature card distinguishes **install eligibility**, **installed state**, **runtime access for the selected identity**, and **approved information sources**. There is only one high-priority install action per eligible feature. The install review dialog shows exact release metadata, size/hash and the known unsigned publisher limitation.

**Developer Mode.** A header toggle reveals Status, Logs, Metrics and Flow from Processor's own authenticated APIs. It does not unlock permissions. **Copy redacted report** copies limited structural data, not GitHub codes, local operator codes, bearer tokens or raw logs.

**Exit.** User chooses *Shut down managed workspace* (default) or *Keep other services running*. In this candidate Processor starts no Identity/Engine subprocess, so it only stops itself and leaves both external services alone.

## User acceptance matrix

Record **PASS / FAIL / BLOCKED / NOT RUN** for the precise SHA and Windows machine, not previous candidate results.

| Scenario | Current evidence classification | What the owner should see |
| --- | --- | --- |
| Windows GitHub Actions Go build and unit tests | **PASS** on green commit-specific CI; not a Windows UI trial | Binary builds and tests |
| Windows double-click + automatic browser open | **NOT RUN** by an owner | Local browser opens without copying port/URL |
| Identity-first and Processor-first | **PASS** Linux real-process integration; Windows **NOT RUN** | Both orders find the untrusted service |
| First pairing + HMAC verification | **PASS** Linux real-process integration; Windows **NOT RUN** | One-time approval then VERIFIED |
| Wrong code | **PASS** Linux real-process integration | Refused as pairing error, no silent access |
| Persist pairing and restart | **PASS** Linux real-process integration | No repeated approval after clean restart |
| Real canonical identity creation | **PASS** Linux real-process integration | Actual `ent_…`; zero authorized sources |
| Other installation / bad proof | **PASS** fixture unit test; owner **NOT RUN** | Refuse wrong Identity, no leaked credentials |
| Remote pairing revocation | **PASS** Linux real-process integration | Fresh approval required |
| Identity disappears | **PASS** Linux real-process integration | Authority unavailable; protected operations blocked |
| GitHub device authentication with a live account | **NOT RUN** | Human approval on GitHub; correct states |
| Real Thought Engine binary install | **BLOCKED** | No real published release currently found in private Engine repo |
| Generic installer / invalid checksum / truncated download | **PASS** fixture tests, real installation **NOT RUN** | No false install; previous files preserved |
| Installed-feature persistence and manual hash verification | **PASS** fixture unit tests; real binary **BLOCKED** | Saved installation remains after restart |
| Private feature name disclosure prevention | **BLOCKED** | Current Identity API embeds names in its response; UI filtering alone is insufficient |
| Publisher provenance / signed metadata | **BLOCKED** | SHA256SUMS verifies transfer integrity, not signed publisher authenticity |
| Windows ACL/DPAPI secret protection | **NOT RUN / NOT IMPLEMENTED** | Local profile file permission audit still needed |
| Developer Mode functional screen test | **NOT RUN** in browser; JS syntax checked in CI | Real logs, metrics, status, no privilege escalation |
| External process ownership on exit | **PASS** Linux real-process integration | Identity remains alive after Processor shuts down |
| Engine permissions, source ingestion and cognition | **OUT OF P0** | Not falsely marked implemented |

No screenshots of the private, real-user workflow are claimed here. The above is the directly implemented interface's walkthrough. A screenshot-based owner evaluation should avoid publishing pairing codes, user codes, profile data or tokens.

## Known limitations and recommended verdict

**Recommendation to owner: REVISE prior to promotion**, while accepting this build as a meaningful UX trial.

- Processor setup is self-contained once its unsigned CI EXE is downloaded; installing Identity itself is still a separate step.
- Private-feature confidentiality requires server-side eligibility-filtered discovery in Identity. This Processor does not claim otherwise.
- GitHub sign-in and an actual private Engine release were not verified live. No owner-permitted Engine release was published by this assignment.
- Release integrity is checked by a signed response from the paired Identity and SHA-256 download verification; **independent publisher-authenticity signing is absent**.
- Pairing token is stored in a restricted user profile, but Windows ACL hardening and DPAPI are future work; do not declare it audited.
- The Browser Auto-Open argument is a temporary local-fragment code passed to Windows for default-browser opening; as with the prior printed local link, same-user local processes may be able to inspect it. Do not share the link. The browser clears its fragment before further navigation.
- Old imported diagnostic routes remain only for comparison; the normal home uses the new guided experience. Their removal needs an explicit security/deprecation review.

The code is experimental, not Alpha. No `alpha`, `stable` or `main` branch was touched.
