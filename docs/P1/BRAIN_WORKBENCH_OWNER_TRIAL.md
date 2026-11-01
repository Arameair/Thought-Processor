# Brain Workbench — owner WSL/Edge trial (experimental)

**Status:** Prepared for owner testing. **NOT browser-accepted or WSL-tested by author.** This branch is separate from the existing `experiment/tass-workbench-claude-20261031` installation; no merge and no profile migration.

## What changed
- The Workstation opens with a **Brain Workbench** showing **real installed feature instances**, not invented nodes.
- Node positions are local browser-session visual preferences only. Moving a node does not rewire a service.
- **Connect proposal** adds dashed, inactive planning edges only; no arbitrary live connections or grants.
- Select an installed feature for runtime state, ID, version, PID when reported, and actual **Start / Stop feature** using existing authenticated Processor routes.
- Diagnostics show Processor activity plus allowlisted test output with existing server-side scrub and an additional client-side scrub; **Copy Clean Log** copies the filtered result. Review before sharing; some unknown secret formats may not be recognized. The panel explicitly does NOT claim to show Identity's audit, service stdout, or unavailable telemetry.
- Top-level **Check Updates / Update Workbench** controls call the real protected WSL launcher's `check` and `update` commands. The update request requires signed-in Processor session, a fresh Identity-verified real GitHub sign-in for `Arameair`, CSRF protection, an explicit confirmation, and no running Workstation tests. The launcher remains responsible for clean checkout, fast-forward-only branch update, exact pinned preflight, restart and rollback. The browser may disconnect during an update; **use launcher Open to return**. Update result is read back from `launcher-control/update-history.jsonl`; no fake completion claim.
- The original projects, file reader, allowlisted tests, TASS conversation and Thoughts panel stay accessible in **Project tools**.

## IMPORTANT safety note
This first owner-specific update bridge relies on the existing authenticated Workstation session and **Identity-verified GitHub login matching Arameair**. It does **not** implement a new general-purpose Identity `apply-update` permission/grant. This is for the operator's isolated experimental WSL Workbench only; it must receive security review before any multi-user/public release.

## Owner smoke test (on your actual WSL computer)

All commands operate in the current `~/p1c/alpha/Thought-Processor` clone, preserving the existing alpha profile, and will stop on error. Close unsaved Workstation work before switching; the old launcher is running `ffc0604`.

```bash
(
set -e
cd ~/p1c/alpha/Thought-Processor
python3 tools/p0-launcher/secure_launch.py stop
test -z "$(git status --porcelain)" || { echo "STOP: Local changes present."; exit 1; }
cp -a ~/p1c/p1-tass-alpha ~/p1c/p1-tass-alpha.brain-backup-$(date +%Y%m%d-%H%M%S)
git remote set-branches --add origin experiment/brain-workstation-controls-20261031
git fetch origin
if git show-ref --verify --quiet refs/heads/experiment/brain-workstation-controls-20261031; then
  git switch experiment/brain-workstation-controls-20261031
  git merge --ff-only origin/experiment/brain-workstation-controls-20261031
else
  git switch --track -c experiment/brain-workstation-controls-20261031 origin/experiment/brain-workstation-controls-20261031
fi
echo "BRAIN WORKBENCH SHA: $(git rev-parse HEAD)"
(cd processor && go test ./...)
python3 tools/p0-launcher/secure_launch.py start
python3 tools/p0-launcher/secure_launch.py open
)
```

**Acceptance on your machine, record PASS/FAIL/NOT RUN:**

1. Workstation opens with a **Brain Workbench** as its first screen and GitHub/Identity genuine status. Project tools still open.
2. Installed instance nodes show real data; unknown fields say `Not reported`. Move one and refresh: only its visual position changes.
3. Draw a proposed connection: dashed **PROPOSED/INACTIVE**, no live grant/configuration change.
4. Select a feature; verify runtime/PID and Start/Stop (only for a real installed feature). Recheck service state after the action.
5. Run an allowlisted test from Project tools, expand Diagnostics and **Copy Clean Log**. Confirm the pasted text has actual result and no secrets. Also check an honest `No telemetry` case.
6. Click **Check Updates**: see actual current and available SHAs, or an honest `UNAVAILABLE`.
7. Click **Update Workbench**, confirm, then reopen with launcher `open` if the browser disconnects. Check **last update receipt** reads `UP_TO_DATE`, `UPDATED`, `REFUSED`, or `ROLLED_BACK` from the launcher. A successful no-change run verifies the button path; it does NOT prove a real fast-forward or rollback.
8. In Project tools, verify TASS still uses the actual Granite endpoint and selected-file context. **A real Granite answer remains the separately required #31 acceptance.**

If `go test ./...` fails, **do not start the trial**; return to the previous branch and launch it:

```bash
cd ~/p1c/alpha/Thought-Processor
git switch experiment/tass-workbench-claude-20261031
python3 tools/p0-launcher/secure_launch.py start
python3 tools/p0-launcher/secure_launch.py open
```

**Do not merge or reconfigure Identity, user profiles, pins or model settings.** No claims of independently executed browser, Windows/Edge, Go suite or update-rollback tests exist for this candidate.
