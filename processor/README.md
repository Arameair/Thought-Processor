# Thought Processor — P1 (feature foundation) on top of P0

> **P1:** run it with [../docs/P1/START_HERE.md](../docs/P1/START_HERE.md). New packages: `internal/featpkg` (verify/stage local feature packages), `internal/supervisor` (launch, endpoint discovery, launch-key auth, startup modes, idle stop, controlled recovery, orphan cleanup), `internal/app/p1.go` (install transaction, approvals, grants, tickets-first actions), `testfeatures/deterministic-worker` (trusted local test feature, Python), `acceptance/p1_integration.py`, `acceptance/p1_ui_tour.py`. Extra flag: `-trust-feature-dir <folder>` (repeatable). Contract: [../docs/P1/FEATURE_API_V1.md](../docs/P1/FEATURE_API_V1.md).

## P0 notes (still valid)

The person's interface to the system. It connects to **Identity**, shows the person's real identities, lists the features their account is approved for, and installs them with a **reusable installer** that is driven by each feature's contract (`thought.feature/1`). Thought Engine is the first example.

- **Windows run and test:** [../AGENTS.md](../AGENTS.md)
- **What works, what doesn't, evidence, recommendation:** [../docs/experiments/P0_FEATURE_ONBOARDING_2026-10-26.md](../docs/experiments/P0_FEATURE_ONBOARDING_2026-10-26.md)
- **Needs Identity** at `Arameair/identity` · `experiment/feature-contract-20261026`

```powershell
.\scripts\build-windows.ps1     # vet, test, build thought-processor.exe
.\thought-processor.exe         # opens your browser; Ctrl+C or Exit in the page to stop
```

| Flag | Default |
| --- | --- |
| `-port` | 7860 (127.0.0.1 only) |
| `-data` | `%USERPROFILE%\.thought-processor\p0` |
| `-no-browser` | off; when set, the private link is only printed |

## Code layout

- `internal/idclient`: the Identity v1 client.
  - Checks the signature on every answer.
  - Pairing, with the token sealed by DPAPI on Windows.
  - Finds Identity through its locator files.
- `internal/install`: the reusable installer.
  - Builds a plan, waits for the person's approval, then downloads.
  - Verifies the file and checks it comes from the approved source.
  - Keeps install records.
- `internal/app`: the connection state, the HTTP server and the page (`web/`).
- `internal/store`: the profile folder, lock and secrets.
- `acceptance/p0_integration.py`: an integration test that runs real processes and a browser, with GitHub replaced by a labelled test fixture.

Run the tests with `go test ./...`.
