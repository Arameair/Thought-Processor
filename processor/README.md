# Thought Processor — P0 (reusable feature onboarding)

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

## Experimental private runner handoff (PR #25)

When started by TheBrain's constrained WSL runner, Processor **never prints** a private `#code=` browser link to stdout or writes it to a profile file. Build the PR #25 experiment and start with `-no-browser -private-link-fd N`, where **N is an inherited anonymous pipe descriptor (3 or greater)**. The app rejects stdout/stderr and regular-file targets; it writes one line to that pipe, closes it, and keeps serving. The caller must keep the pipe in memory and securely deliver the link to the intended user; this is not an HTTP open endpoint.

Normal `-no-browser` without a private fd has **no sign-in link output** and cannot sign in automatically. Interactive non-runner mode can still open the browser directly, but the private link is **never logged or printed**. Pairing with Identity remains separate, and Identity's one-time pairing code requires a safe owner-visible UI/terminal. The [TheBrain draft PR #10](https://github.com/Arameair/The_Brain/pull/10) wires a one-time authenticated local transport and Edge InPrivate candidate; do **not** claim Windows verification until Tester runs it.
