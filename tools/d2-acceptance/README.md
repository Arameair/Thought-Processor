# D2 P0 disposable acceptance scripts (PROPOSED, NOT EXECUTED)

This folder holds the D2 P0 Identity protected-profile acceptance procedure as version-controlled
PowerShell, replacing the hand-assembled Patch F blocks in [PR #134](https://github.com/Arameair/Thought-Processor/pull/134).
It drives the existing harness `TestAcceptanceDisposableProtectedProfile` in
`Arameair/identity@1718c3d36c919daca6cbc8cc8ef5e4155ed9c33f` and does not change it.

- **Status:** proposed for independent Codex review. Nothing here has been run on any host.
- **Not authorized:** VM setup, accounts, ACL changes, WSL import, or acceptance runs. Each needs the owner's explicit authorization of an exact run.
- **D2 stays OPEN.** THREAT-MODEL-B WSL ISOLATION and INSTALLED-SYSTEM SECURITY are always **UNVERIFIED** here.
- This folder holds no production Identity code. The prose rules of revision 2 (§1 labels, §2 Option A, §3, §9, §10) and Addendum A5 still apply. The security playbook is unchanged.

## Files

| File | Role | Side effects |
| --- | --- | --- |
| `D2Evidence.psm1` | Pure parsing, cleanup decisions and scoring: harness reports (F6), WSL listings, journals, C3 state, Go `LookPath` emulation, F9 labels, parse-inventory verdict, SDDL models | None. `Test-D2Package.ps1` enforces this with an AST command allowlist |
| `D2Host.psm1` | Host steps F2–F9 (`Initialize-D2Role`, `Invoke-D2Step*`, `Invoke-D2Run`) | **Changes the disposable VM.** Importing it only compiles a P/Invoke helper |
| `Test-D2Package.ps1` | F0 replacement: exact file inventory, PowerShell parse, ASCII, AST rules, SHA-256 list | Read-only |
| `tests/D2Evidence.Tests.ps1` | Synthetic failure-path tests of the pure module (no Pester needed) | Read-only, in memory |
| `tests/static_check.py` | Author-side heuristic checks when no PowerShell is available. **Not a parser** | Read-only |

## Read-only verification (Codex)

These commands need no elevation. They run nothing from `D2Host.psm1`.

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\tools\d2-acceptance\Test-D2Package.ps1
powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\tools\d2-acceptance\tests\D2Evidence.Tests.ps1
python3 -I tools/d2-acceptance/tests/static_check.py tools/d2-acceptance
```

`Test-D2Package.ps1` prints `PACKAGE CHECK: PASS` only if exactly the four expected script files are present and each one parses. An empty or partial folder is FAIL.

## Owner inputs (before any authorization)

These go in one `-Auth` hashtable, passed to `Initialize-D2Role`. Any value containing `<` is refused.

| Key | Meaning |
| --- | --- |
| `VmName` | Authorized VM `COMPUTERNAME` (Option A; CP0 checked on the hypervisor by hand) |
| `Git`, `Go`, `Wsl` | Full paths of the approved `git.exe`, `go.exe` and `wsl.exe` |
| `ImgUrl`, `ImgSha`, `ImgName` | Official WSL image URL, its uppercase SHA-256, and its file name |

The owner also decides whether `runas` is acceptable for the svc window.

## Procedure order (each row is a separate owner-authorized step)

Work from a clone at the exact reviewed commit. Run `Test-D2Package.ps1` first, and compare its hashes with the reviewed ones. Every window starts with `Import-Module <package>\D2Host.psm1` and `Initialize-D2Role -RunId <id> -Role <role> [-ExpectSid <sid>] -Auth $auth`. The svc and usr windows import from `C:\ProgramData\D2AccBin-<id>\pkg\`, which F4 publishes. That folder is Administrators-owned, and its file digests are journaled.

| Step | Window | Command |
| --- | --- | --- |
| F2 | admin, elevated | `Invoke-D2StepAdminPrestate` (prints the ADMIN SID) |
| F3 | build: same admin, **non**-elevated, `-ExpectSid <ADMIN SID>` | `Invoke-D2StepBuild` (prints the BUILD HASH) |
| F4 | admin | `Invoke-D2StepFixture '<BUILD HASH>'` (prints the SVC/USR SIDs) |
| F5 | svc, then usr (`-ExpectSid` = own SID) | `Invoke-D2StepAccess '<BUILD HASH>'` |
| F5W | usr | `Invoke-D2StepWslPreflight` (READY or BLOCKED; either is journaled) |
| F6 | svc / usr / build | `Set-D2Hash '<BUILD HASH>'` if new window, then `Invoke-D2Run <tag>` for the base tags: S-A, S-E, C1b (svc); N-A, N-E, C1a, and W-A, W-E if READY (usr); C2 (build) |
| F7.0 | operator gate | C3 starts only if S-E and N-E (and W-E when READY) are `EXPECT … MET` |
| F7.1 | admin | `Set-D2Hash`, then `Invoke-D2StepC3Weaken` |
| F7.2 | svc / usr | `Invoke-D2Run C3-S` · `C3-N` · `C3-W` (only if READY). Always continue to F7.3 |
| F7.3 | admin | `Invoke-D2StepC3Restore`. **Mandatory**, even if F7.2 failed or was lost |
| F7.4 | svc / usr | Only after `C3 RESTORED AND VERIFIED`: `S-E2`, `N-E2`, and `W-E2` if READY |
| F8.1 | usr | `Invoke-D2StepWslCleanup`, then sign out of usr and close the svc window |
| F8a | admin | `Invoke-D2StepCollect` |
| F8b | admin | `Invoke-D2StepCleanup`. It always ends in `CLEANUP: CLEAN` or `CLEANUP: INCOMPLETE (n)` |
| F9 | admin | `Invoke-D2StepScore`. It prints all four labels separately |

**Recovery path:** if `Invoke-D2StepC3Restore` cannot verify the bound object, or the restore fails, stop all runs. Then revert the VM to CP0 or discard it. On Option A, revert or discard the VM after every run, whatever the cleanup status. No script ever deletes `$Ev` or the package folder.

## Behavior changes from Patch F that need owner and Codex acceptance

1. **C3 weakening goes through one bound handle.** It applies `SetSecurityInfo` to the entity root only, after binding the object's file ID and its pre-mutation DACL. The procedure does not assume how existing children are treated; restore verification re-checks every descendant exactly. Patch F used `icacls /grant` by path.
2. **Directories are created exclusively, with their final DACL** (`CreateDirectoryW` plus a security descriptor). This replaces create-then-`icacls`. It assumes the VM's default owner for elevated objects is Administrators. If it is not, the fixture check STOPs; nothing is repaired.
3. **The post-state comparison of `C:\` and `C:\ProgramData`** uses handle-read owner and DACL models (`pre-sddl.txt`), not `icacls` text dumps.
4. **Role SIDs, the build hash, the WSL launcher digest and the image hash** come from an Administrators-owned run record in `$Bin`. Hand-typed values are only cross-checked against it.
5. **The WSL empty-state message is an exact allowlist.** Unrecognized text means an UNKNOWN listing, which gives a BLOCKED preflight or a RESIDUAL cleanup, and the allowlist is never widened at run time.
6. **The harness output grammar is pinned to the candidate.** That means the call-site lines 60, 251 and 253 of `windows_test.go`, and exactly one RUN, report, terminal, PASS/FAIL and final line. Any other test-runner output, for example `=== NAME` lines from a different Go version, gives `PROCEDURE-ERROR`, never `ISOLATED`.
7. **The package uses no simplified `Where-Object Name -op` syntax.** `Test-D2Package.ps1` rejects any operator bound as a command parameter.
