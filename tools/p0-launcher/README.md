# P0 Windows/WSL Workbench Launcher — isolated engineering prototype

**Status: LOCAL UNIT TESTS PASS; OWNER WINDOWS/WSL TEST NOT RUN.** This is not a release or a fully verified workstation. It is the #32 Start / Stop / Open / Versions / Check Updates READ-ONLY slice. **Apply Update is disabled.** #31 real repo/file/test workstation is a separate unfinished milestone.

## Exact reviewed inputs

- Processor branch: `experiment/p0-windows-wsl-launcher-20261029`, descending from PR #30 `72bd667fdf0ce14332f6b1e717d263f1d53e5d93`.
- TASS branch: `experiment/tass-companion-feature-claude-20261029`, source `ecace8fdffa72fc40f8a0a83465194a8c51e4053`; package digest `985027bbaebfd99bb07926285ceac218726ddca3fca0dff421bf665e2781abef`.
- Identity unchanged: `64940b629a5ef52eaa143b2402726ec81d382002`. No merge/deploy; no Identity policy or trust pin changes.
- Persistent **fresh experiment profile** only: `~/p1c/p1-tass-fresh` (existing legacy `~/p1c/p1-tass` untouched). Identity port **7797**, Processor port **7867**.
- No TASS installation, enrollment, device authorization, grants or Admin approvals automated.

## What is implemented

- `secure_launch.py`: WSL supervisor holds a private Unix control socket (`0700` directory, `0600` socket, SO_PEERCRED same-UID check) and an exclusive advisory lock. Preflight uses the existing verified `run-p1.sh` **preflight-only path**; it never runs the original runtime path, which persists private links in `*-private.log`.
- Builds the pinned Identity/Processor source, starts each as an owned child, collects their combined process output **in memory only**, recognizes the Processor private session link and fresh Identity pairing challenge, discards all other output. No session link/code/password written to stdout, log or file by this supervisor.
- `Start` reports true STARTING/FAILED/RUNNING only when the private Processor opening link and both owned listeners exist. Repeated Start does not spawn duplicates. `Stop` signals only tracked child process groups and reports `STOP_INCOMPLETE_PORT_BUSY` if ports remain occupied. No foreign PID killing.
- `Open`: on WSL, forwards private link through **PowerShell stdin**, requests Windows Edge **InPrivate**. It says `OPEN_REQUESTED`, not owner-browser PASS. `Pairing` likewise shows the latest Identity pairing challenge in a private Windows dialog, never in console output.
- `Status`: redacted process/port information; internal browser session polls genuine Processor `/api/state` for Identity-derived GitHub state when available. Does not equate github.com login with `VERIFIED`. No raw `/api/state` is exposed without a session. `VERSIONS`: local HEADs and pinned digests. `CHECK`: read-only `git ls-remote`, labels any differing upstream head **UNVERIFIED_REMOTE_CHANGE**, not an approved update.
- `Workbench.ps1` text menu and `Start-Workbench.cmd` Windows double-click entry. The CMD wrapper does **not** disable Windows execution policy.

## Local test evidence

```bash
python3 -m unittest discover -s tools/p0-launcher -p 'test_*.py' -v
python3 -m py_compile tools/p0-launcher/secure_launch.py
```

**Engineering scratch environment: 11/11 local tests PASS (ten mock/negative checks and one real owned-listener lifecycle test) and Python compilation PASS.** Git blob checksums matched the uploaded, locally-tested Python source/test bytes. These tests cover truthful GitHub status, status/pairing redaction, pairing one-time retrieval, no unsupported command, Open unavailability, foreign process safety, unsigned update refusal, unknown upstream change, actual closure of two owned local test listeners on Stop, and preserving failed preflight evidence. **Not tested:** Go build/run, Ubuntu WSL, Windows menu, Edge opening, live GitHub device flow, genuine Identity sign-in, real TASS model, actual port cleanup, restart/orphan resilience, private link visible to OS process inspectors, or independent security review.

## Windows 11 + Ubuntu WSL developer/test instructions (NOT owner-release approval)

1. In an existing WSL shell, leave running trials alone. Ensure ports 7797/7867 are free before starting (never forcibly kill foreign processes). If an owner trial uses those ports, stop it through its original owner controls first.
2. Existing pinned `~/p1c/identity` and `~/p1c/tass-companion` checkouts must be present, clean and on expected reviewed branches. A separate clean checkout of the Processor experiment is required:
   ```bash
   test ! -e ~/p1c/tp-p0-launcher && git clone --branch experiment/p0-windows-wsl-launcher-20261029 https://github.com/Arameair/Thought-Processor.git ~/p1c/tp-p0-launcher
   git -C ~/p1c/identity rev-parse HEAD
   git -C ~/p1c/tass-companion rev-parse HEAD
   git -C ~/p1c/tp-p0-launcher rev-parse HEAD
   python3 ~/p1c/tp-p0-launcher/tools/p0-launcher/secure_launch.py versions
   ```
   Expect Identity `64940b6...`, TASS `ecace8f...`; Processor must be the exact reviewed launcher SHA communicated by the Test Architect.
3. Windows: open `tools\p0-launcher\Start-Workbench.cmd` from the Windows-accessible clone path (`\\wsl.localhost\Ubuntu\home\<your-wsl-user>\p1c\tp-p0-launcher\tools\p0-launcher`). Choose **Start**, **Status**, **Open**, **Check Updates (read only)**, **Stop**. Do not click or approve security actions unattended. `Open` must start Edge InPrivate; `Pairing` displays a challenge only in local UI. If Windows blocks the script by policy, do not silently bypass it.
4. Alternative WSL terminal commands (no secret output):
   ```bash
   python3 ~/p1c/tp-p0-launcher/tools/p0-launcher/secure_launch.py start
   python3 ~/p1c/tp-p0-launcher/tools/p0-launcher/secure_launch.py status
   python3 ~/p1c/tp-p0-launcher/tools/p0-launcher/secure_launch.py open
   python3 ~/p1c/tp-p0-launcher/tools/p0-launcher/secure_launch.py check
   python3 ~/p1c/tp-p0-launcher/tools/p0-launcher/secure_launch.py stop
   ```
5. After Stop, independently verify **no owned listeners** on 7797/7867; if another process uses a port, Stop must show busy and must not kill it. Test genuine Identity pairing, Admin enrollment, GitHub device WAITING→VERIFIED, real signed TASS RUNNING, and Qwen model responses separately. Every outcome must be recorded PASS/FAIL/NOT RUN, not inferred from Start's exit code.

## Blockers / threat boundaries

1. **Not yet security-reviewed:** Edge receives the private session link as a navigation argument. Windows process inspection and browser telemetry may expose it transiently even though PowerShell receives it through stdin. **Do not claim full secrecy across Windows process argv** until the host-browser bridge is audited/modified. Browser opener success is not proof of browser authentication.
2. **Daemon-crash recovery:** normal Stop supervises its actual owned children; unexpected daemon termination or Windows/WSL shutdown has not been tested for orphan process cleanup. No PID-file-based arbitrary cleanup exists.
3. **GitHub VERIFIED** is read from Processor's real authenticated state only; we have **not** fixed or verified the owner's original WAITING→VERIFIED failure. The linked Identity draft PR #22 is separate and not included.
4. **No safe Update Apply:** same-ID Admin-approved artifact upgrade and rollback remain separate owner/security decisions (#31). **Only Check Updates is available** and it never treats an upstream Git branch as a reviewed release.
5. This launcher alone does not provide repo tree, bounded tests, source edits, or a finished IDE. Actual #31 workstation UI is still outstanding.

**Never paste private codes, #code fragments, tokens, passphrases or local credential files into GitHub issues, screenshots or shared test evidence.**
