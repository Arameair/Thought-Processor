# Thought-Processor #31 — TASS same-ID upgrade gate (experimental)

**Status: SECURITY CHECK PASS / USER WORKFLOW BLOCKED.** This branch deliberately **does not** upgrade any owner profile. The current Identity service at `64940b629a5ef52eaa143b2402726ec81d382002` has **no** Admin-approved existing-installation upgrade API. It registers a new ID for a new install, and uninstall revokes grants. A Processor-only package replacement would be an authority bypass.

## Scope and verified inputs

- Old TASS pinned artifact: `86d79ba30ab3afb26bf015364fa0556dfa553c1e48d673b4243a827d6f8ba757` (reviewed earlier P1 launcher `695b03c`).
- New TASS pinned artifact: `985027bbaebfd99bb07926285ceac218726ddca3fca0dff421bf665e2781abef` from TASS `ecace8fdffa72fc40f8a0a83465194a8c51e4053`.
- Existing installation ID: `ins_2883db5a24e1e0e93cad` (must stay unchanged).
- Existing identity: `kk`; existing chat and grants **must not be copied into issue comments**.
- Processor base: `c898c647590ea86c6b8d59d60e06b970a4f5c361`; Claude PR #30, TASS PR #7 and Identity are untouched.
- Existing WSL workbench `~/p1c/p1-tass`, Identity `7797`, Processor `7867`; **no new ports**.
- Ollama/Qwen status was owner-observed on WSL; this branch does not retest or change Ollama.

## Deliverable completed in this checkpoint: read-only eligibility proof

`processor/upgrade/tass_upgrade_gate.py` recalculates the canonical manifest+files digest on *both* packages. It checks the original install record, package manifest, file checksums, ID, Identity binding, source, capability additions, data folder and credential **existence**. It does not open credentials or chat content. It returns **BLOCKED_IDENTITY_UPGRADE_API_MISSING** even if all eligibility checks pass. Any mismatch returns **REFUSED**. A successful preflight **is not authorization to launch**.

`processor/upgrade/test_tass_upgrade_gate.py` has nine synthetic tests for correct block status, tampering, wrong ID, absent credential, wrong pin, capability removal and symlink rejection. **Commit presence is not a test PASS; actual execution evidence is still required.** No owner test has run.

## Proposed *minimal Identity* protocol — approval required before any implementation

1. **Request:** Processor asks Identity for `upgrade_installation` approval, binding **existing installation ID + Identity system ID + current artifact + proposed exact new digest, manifest, version and capability delta + caller**. Identity verifies old install state, old digest and both operator pins. Admin UI shows old→new, what stays, what is added and **Approve / Deny** with passphrase. No generic arbitrary replacement endpoint.
2. **Reserve / maintenance fence:** On approved request, Identity marks the same installation as `UPGRADING`. Tickets and protected requests fail closed while in transition. Existing grants remain stored but cannot be used to authorize new capabilities.
3. **Processor prepare:** verify staged approved files again, verify no running TASS, securely snapshot original package + record and locally held data *with confidential permissions*, and create a fsync-backed crash journal. Never emit copied chat, credentials, links or secrets in evidence.
4. **Commit / proof:** Processor swaps only this installation's package/record via transactional staging, retaining `data/`, credential, ID and old grants; reports exact installed hash back to Identity using its existing authenticated installation context. Identity atomically commits new metadata for the **same ID** and retains grants only for unchanged capability scopes. Newly introduced `tass.model.set`, `tass.self.read`, etc. require separate Admin grants. No implied Admin power for TASS.
5. **Recovery:** On denial, expiry, version mismatch, wrong Identity, replay, crash or power loss, the journal and Identity transaction resolve to a coherent **OLD** or **NEW** state. Any ambiguous split state remains **BLOCKED** until owner-reviewed recovery; never silently starts the old/new package or invents a new ID. Rollback of an approved upgrade must be separately recorded and governed by Identity, not silent file replacement.
6. **Tests before touching owner's state:** run against a disposable copy with synthetic credentials and a disposable Identity profile; prove same ID, conversation hash, grants, no unauthorized capabilities, signed status, denial, mismatch, replay, revoke, failure at each transaction phase, clean stop and successful rollback. **No copy of the live Admin profile should be used in automated tests without explicit approval and containment.**

**Owner decision requested:** authorize the narrow Identity upgrade transaction and designate the Identity code owner/security reviewer. This is a required authority change; **no authorization is inferred** from this design note. Until approved and tested there is no safe way to label Deliverable A complete.

## One Windows + WSL preflight list — safe, read-only with respect to the original

**Do not paste until all P1 programs using `~/p1c/p1-tass` have been stopped normally with Ctrl+C.** Run in a Windows PowerShell window with WSL installed. This creates an isolated code clone and a local, private disposable *copy* of the stopped Processor profile, but no listener or new port. Do **not** post the copy or its logs anywhere.

```powershell
wsl.exe -- bash -lc 'set -euo pipefail; umask 077; test -d "$HOME/p1c/p1-tass/processor-profile"; test -d "$HOME/p1c/tass-companion/feature"; test ! -e "$HOME/p1c/tp-tass-upgrade-review"; git clone --branch experiment/tass-pinned-upgrade-20261029 https://github.com/Arameair/Thought-Processor.git "$HOME/p1c/tp-tass-upgrade-review"; git -C "$HOME/p1c/tp-tass-upgrade-review" rev-parse HEAD; test ! -e "$HOME/p1c/tass-upgrade-disposable"; mkdir -m 700 "$HOME/p1c/tass-upgrade-disposable"; cp -a "$HOME/p1c/p1-tass/processor-profile" "$HOME/p1c/tass-upgrade-disposable/processor-profile"; python3 -I "$HOME/p1c/tp-tass-upgrade-review/processor/upgrade/tass_upgrade_gate.py" --installed "$HOME/p1c/tass-upgrade-disposable/processor-profile/installations/ins_2883db5a24e1e0e93cad" --candidate "$HOME/p1c/tass-companion/feature"'
```

**Expected when the input files match:** JSON with `gate: BLOCKED_IDENTITY_UPGRADE_API_MISSING`, old/new full digest, ID, `identity_approval: NOT_REQUESTED`, and `safe_to_start_new_artifact: false`, plus process exit code **3** (a deliberate stop, not PASS). Bad files produce `gate: REFUSED`, process exit code **2**. The actual owner's old profile and grants are never edited. The one-liner intentionally leaves the disposable local copy for owner-controlled inspection/deletion.

**Installation, launch, status, stop on this same-ID new artifact: NOT YET SAFE/AVAILABLE.** Once Identity's approved transaction and disposable end-to-end tests exist, this document must be replaced by **one pinned commit** and the final owner copy/paste Windows+WSL list retaining `~/p1c/p1-tass`, `7797/7867` and the same ID. Until then do not repeat Start, uninstall TASS, reset profile, change its installed manifest, or add old digests as an unconditional runtime bypass.

**Follow-on Deliverable B:** real authenticated TASS Workspace with project tree/tests is a separate gate; it must not be confused with this read-only upgrade verification.
