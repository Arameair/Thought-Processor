# Brain Development Workbench — owner-test candidate

**Status:** Experimental implementation, NOT tested on Dustin's Windows 11 / Ubuntu WSL / Edge setup. No merge, release, profile migration, grant change or launcher replacement.

**Branch:** \`experiment/brain-workbench-controls-gpt6-20261031\` (branched from tested Workbench candidate \`ffc0604aba2ab536ae0896be1db6694d8e22bc06\`).

## Goal

The Development Workstation should **open to a brain-shaped map of real installed feature instances** surrounding Thought Processor. The user can move nodes, propose connections, inspect operational state, start/stop an approved installed feature, run its self-test, inspect tests/thoughts/files/TASS, and copy **clean diagnostic evidence** to share for troubleshooting.

This extends the existing working Workstation UI; it is not a separate demo or a rewrite. Launcher menu **8 Update** remains unchanged; the browser Updates button explains that updates happen through the launcher rather than impersonating a direct privileged action.

## What changed

- \`/workstation\` now lands on **Brain Map**, with five status metrics (installed/running/grants/Identity/tests).
- Real nodes are discovered from authenticated \`/api/state\` installation records, not a hard-coded sample service list. Processor and Identity are always shown as central authorities, not fabricated installations.
- Drag a node to change **only local visual positions**. Click **Connect** and two nodes, or drag from a node's connector dot to another node, to store an **inactive visual proposal**. No graph action changes live configuration or grants. Unverified connections never become solid.
- Solid lines: Processor has a live, launch-verified health/ready view of that installed feature. Optional dotted Identity lines: the installation has at least one active Identity grant **but that is not proof of live communication**. Dashed violet lines: purely local inactive proposals.
- Click a real installed instance to inspect actual version, installation ID, feature, runtime, PID, health, file verification, grants and last exit. Start, Stop, Self-test and Copy Activity call existing Processor routes; buttons stay disabled without suitable Identity-confirmed installation state. Stop asks for confirmation.
- **Files & TASS**, **Tests**, **Thoughts** tabs retain the original actual repository browser, real-file viewer, TASS local-model chat, allowlisted tests, test results, and previously implemented Thought reads.
- Always-accessible diagnostic area: choose Processor or an installed feature, refresh, and **Copy Clean Log** or **Copy Support Report**. The support report contains source-backed software version, service status, Identity/GitHub state, test-run count, and the selected sanitized event log (never raw file text/test stdout). Processor events are from a **new gated endpoint** \`GET /api/ws/diagnostics\` using \`Log.Recent(120)\` (which records redacted text); per-feature activity uses existing \`/api/installations/{id}/activity\`. A clipboard failure selects the text for manual Ctrl+C. **Never copy a raw feature.log or arbitrary test stdout as a "clean" log.** User should review project names and details before sharing.
- All assets are embedded locally; no CDN or external runtime libraries.

## What it does NOT do

- It does not automatically configure or activate feature wiring, issue Identity grants, or install anything merely because a line was drawn.
- It does not declare a service connected without live evidence.
- It does not implement automatic model selection, commercial updates, or the future five-Spark learning cycle.
- It does not call an external AI service just to draw the map.
- It does not run arbitrary shell commands.
- It does not alter the existing secure launcher, its Update menu 8, or currently running user's checkout.

## Verification status

- [KNOW] Source-level: \`brain-workbench.js\` parsed as valid JavaScript in an isolated V8 syntax check. All its queried IDs are present in the new HTML.
- [KNOW] The existing source declares POST \`start\`, \`stop\`, and \`self-test\` routes, plus the clean per-feature \`activity\` route.
- [KNOW] Existing Go \`web/*\` embedding includes the two newly created local assets.
- **NOT RUN**: \`go test ./...\` on the resulting candidate; new Go test \`TestBrainWorkbenchAssetsEmbedded\` and log scrub test; live browser layout, drag/wire connections, or clipboard; actual Identity grants and runtime actions; WSL/Windows owner acceptance.
- Code review required before any alpha promotion. Do not claim this is a production-ready update.

## Local owner-test plan — no merge

First stop the currently running launcher and confirm CLOSED status, then make a new dated backup of the alpha profile while stopped. Inspect uncommitted changes before switching. Fetch the new experiment branch, switch, run Go tests from the \`processor/\` Go module and launch with the existing launcher. This branch is distinct from the current Workbench update lane: **menu 8 on the old branch does not install this experimental UI**.

Suggested independent checks:

1. With a fresh signed login and GitHub VERIFIED, Brain Map opens first; on denial/outage, existing Workstation gate still locks it.
2. Real installed instances only; runtime changes reflected in graph and stats, unknown fields say "Not reported". Moving a node and adding/removing proposal does not alter service state, grants, profile or config.
3. On a real authorized feature, Start / Stop / Self-test show actual results and no unapproved side effects. Revoked/wrong-identity/offline requests fail closed.
4. Select feature → Diagnostics → Refresh → Copy Clean Log and Copy Support Report. Confirm copied content matches displayed **redacted** events and contains no private links, tickets, approval tokens or credentials. Verify no telemetry and unavailable-source states.
5. Files & TASS still opens a real project file and a local Granite reply; Tests still run only allowlisted commands; Thoughts retains its prior permissions and reads.
6. Updates button does not mutate the checkout; existing launcher's **menu 6 check** and **menu 8 update** are independently tested on their original intended branch.
7. Browser layout/controls at standard desktop width and narrow screen; no console errors.

**Owner acceptance remains separate from source creation and cloud/fixture tests.**
