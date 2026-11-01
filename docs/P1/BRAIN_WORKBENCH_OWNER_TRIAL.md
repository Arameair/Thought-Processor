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

The first owner attempt at `61f87c4` reported a Go build failure from an unused import in `workstation_update_linux.go`; that unused import was removed in commit `de02b0e3c7f274cf83d195f2e317c6f5afe2cbac`. **Go tests and actual browser acceptance still need an owner run.** The old rollback snippet also changed directories inside an ERR trap; this one fixes that by using absolute launcher paths and testing in a disposable worktree before stopping anything.

```bash
(
set -e
ROOT="$HOME/p1c/alpha/Thought-Processor"
BRANCH="experiment/brain-workstation-controls-20261031"
EXPECTED="de02b0e3c7f274cf83d195f2e317c6f5afe2cbac"
cd "$ROOT"
PREVIOUS="$(git branch --show-current)"
test -n "$PREVIOUS"
test -z "$(git status --porcelain)" || { echo "STOP: local changes present"; exit 1; }
git remote set-branches --add origin "$BRANCH"
git fetch origin
test "$(git rev-parse "origin/$BRANCH")" = "$EXPECTED" || { echo "STOP: trial commit changed; review first"; exit 1; }
TEMP="$(mktemp -d)"
trap 'git -C "$ROOT" worktree remove --force "$TEMP/tree" 2>/dev/null || true; rmdir "$TEMP" 2>/dev/null || true' EXIT
git worktree add --detach "$TEMP/tree" "$EXPECTED"
(cd "$TEMP/tree/processor" && go test ./...)
git worktree remove --force "$TEMP/tree"
rmdir "$TEMP"
trap - EXIT

python3 "$ROOT/tools/p0-launcher/secure_launch.py" stop
cp -a "$HOME/p1c/p1-tass-alpha" "$HOME/p1c/p1-tass-alpha.brain-backup-$(date +%Y%m%d-%H%M%S)"
restore() {
  trap - ERR
  cd "$ROOT"
  python3 "$ROOT/tools/p0-launcher/secure_launch.py" stop || true
  git switch "$PREVIOUS" || true
  python3 "$ROOT/tools/p0-launcher/secure_launch.py" start || true
}
trap restore ERR
if git show-ref --verify --quiet "refs/heads/$BRANCH"; then
  git switch "$BRANCH"
  git merge --ff-only "origin/$BRANCH"
else
  git switch --track -c "$BRANCH" "origin/$BRANCH"
fi
test "$(git rev-parse HEAD)" = "$EXPECTED"
python3 "$ROOT/tools/p0-launcher/secure_launch.py" start
python3 "$ROOT/tools/p0-launcher/secure_launch.py" open
trap - ERR
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
