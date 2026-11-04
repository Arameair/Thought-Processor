# Atrium / Daily Mission — Wednesday, November 4, 2026

**EXPERIMENTAL, NOT YET OWNER-ACCEPTED.** ChatGPT's separate Atrium branch builds on draft [PR #38](https://github.com/Arameair/Thought-Processor/pull/38). Claude's independent teaching workspace is not modified. See [TASS cleanup #21](https://github.com/Arameair/TASS/issues/21) and [shared workstation agreement](https://github.com/Arameair/Developmnt/issues/16).

## What today's desk really does

- The default landing page (/, also /today) organizes six cleanup tasks. Select or add an idea, describe why it matters, record the next safe test and expected outcome, and save working notes. A reported PASS requires an evidence note. Task notes persist **locally**, NOT in GitHub or TASS memory.
- Read-only live Workshop bridge shows whether the API responds, the model's **reported** readiness, TASS's mirrored mind source and sanitized real timeline/proposals. It never claims a genuine Engine cognition loop, a valid grant or Docker health from those signals alone.
- Prepare and edit a task-specific message, send it to the real TASS through the existing restricted text bridge, and watch Workshop events. Any code execution still requires explicit approval in Workshop (port 8790). Atrium does not approve/run code and blocks new requests while Workshop reports Full Auto enabled.
- Copy a clean selected-task handoff into this ChatGPT conversation. Optional GitHub sharing requires a separately enabled queue and explicit owner consent. Queued is not posted.
- The older Build Room remains at /build. The visual Atrium remains at /explore. No code was changed in Claude's workspace.

**Not implemented:** real persistent memory, independent Thought Engine integration, automatic daily UI deployment, Identity P1 grants in Workshop, an embedded live ChatGPT, or a full Docker controller. Today's goal is proving the existing foundation and one meaningful 20–30-second cognition opportunity with valid IDLE.

## First home test — independent launch, NO Docker changes

**First inspect yesterday's running stack read-only** before changing checkouts or launchers:

~~~bash
docker ps --format 'table {{.Names}}\t{{.Image}}\t{{.Status}}\t{{.Ports}}'
~~~

Do not call the current workspace/run.sh: it has Docker rebuild and unrelated Liaison stop behavior. Do not switch the checkout from which your live stack was built.

Make a separate checkout of the new experimental branch (GitHub CLI sign-in required):

~~~bash
gh repo clone Arameair/Thought-Processor "$HOME/atrium-today" -- --branch experiment/atrium-daily-mission-20261104 --single-branch
cd ~/atrium-today/workspace/atrium
python3 -m unittest discover -s tests -v
bash ./start_today.sh
~~~

If ~/atrium-today already exists, inspect it and use another destination; do not overwrite it.

Open **http://127.0.0.1:8793/** in the Windows browser. This standalone Atrium uses Python's standard library and reads the already running Workshop at **http://127.0.0.1:8790/**. If the Workshop is down, the task desk still works but properly shows UNREACHABLE. Nothing starts/stops Docker or touches its network.

Port 8793 is intentionally separate from the older Atrium's 8791. If needed, set TASS_ATRIUM_PORT or TASS_WORKSHOP_PORT before running the script. Never kill unknown port owners. Stop this Atrium with **Ctrl+C**.

The task board is saved to ~/.local/state/tass-atrium/desk.json (directory mode 0700, file mode 0600). It is separate **operator work notes**, not TASS memory. Never write tokens, session URLs, personal secrets, or credentials there.

The optional Share Brief button may be unavailable in standalone mode. Use **Copy clean handoff** to bring ChatGPT into today's work.

## Owner acceptance sequence

1. **No disruption:** page loads on 8793, existing TASS containers still untouched, and /build and /explore remain navigable.
2. **Persistence:** add an idea, edit a task and click Save. Refresh, stop/restart standalone Atrium, and confirm task/evidence remains. Another stale tab must fail clearly rather than overwrite recent work.
3. **Truthful connection:** model offline is reported as unknown/not ready, never simulated. Identity mirror is labeled as a mirror, not permission. Engine and Docker are NOT VERIFIED by the Atrium bridge.
4. **One live collaboration:** select a task, Prepare from task, inspect exact message and click Ask the real TASS. Verify matching real Workshop conversation. Review any proposed Python ONLY in the original Workshop; no Full Auto.
5. **Clean handoff:** Copy clean handoff includes task, current result, expected and observed evidence, unknowns and next safe step. No model transcript or private notes included.

**PASS only with real owner evidence and exact running revisions.** GitHub Actions fake-model tests and UI page loads do NOT prove live Nemotron, cognition, permissions or persisted learning.

## Today’s cleanup mission

From [TASS #21](https://github.com/Arameair/TASS/issues/21): preserve Docker baseline → identify actual TASS runtime → reconcile TASS/mind vs Developmnt/mind sources → compare Thought / Thought-engine / Workshop records → observe no-work IDLE and one real event → test approval and grant boundary. No memory repo, no autonomous execution, no mass migration.

## Safety / ownership

- Atrium remains localhost-only. Host/origin and CSRF checks protect state changes but do not replace Identity authentication. Keep experimental ports private.
- Board notes are bounded and stored atomically; concurrent stale revisions are rejected.
- The current GitHub/Notebook task-sharing code does not make this a privileged executor.
- ChatGPT updates its own Atrium experiment; Claude owns Claude's independent notebook playground. Integrate proven capabilities into their owning repos only through approved PRs; no code changes, branch merge or deployment to alpha without owner acceptance.
- If anything fails, preserve the running stack, document the command and non-sensitive error, and mark BLOCKED or UNKNOWN instead of inventing success.
