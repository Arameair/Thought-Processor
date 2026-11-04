# TASS Atrium — Companion Playground

This is a separate browser experience, served at **http://127.0.0.1:8791/** by an optional, Docker Compose-managed container in the Thought-Processor workspace.

It is the actual TASS Atrium playground: living constellation, idea garden, experience lab and a read-and-chat area. It connects to the **same** existing TASS Workshop service through a restricted bridge to `http://workspace:8790` inside Docker.

## Separate Pulse Test Cards pilot (Issue #51)

Atrium Today's desk now includes two **fixed, owner-approved Go test cards** with evidence receipts and a configurable 20-second coordination pulse. This is separate from the Workshop code-approval workflow and from the existing task notes. **Execution is disabled by default** in the normal Atrium Docker image, and the local card acknowledgement is NOT Identity authentication. A separately isolated test runner is required to enable fixed tests. The normal Atrium Docker container has no Go toolchain/source checkouts and must not gain privileged access merely to run these cards.

Runbook, exact targets, security boundary, fixture/real CI evidence and acceptance procedure: [PULSE_CARDS.md](PULSE_CARDS.md).

**Notebook laboratory experiment:** [NOTEBOOK_LAB.md](NOTEBOOK_LAB.md) documents
the third, hash-pinned Identity notebook lesson card, Jupyter/Discord source
inventory, CI evidence, model-fixture limitations and unapproved @identity-test
lookup. The actual Nemotron teaching loop is NOT RUN; no live Jupyter or Docker
deployment is enabled by this change.

## What it can do
- Show the local model's reachability, available books and projects, and the genuine Workshop conversation timeline.
- Send one user-written message through `/api/message` and display actual replies from the Workshop's `/api/events`.
- Show when a run requires review, with a link to the original Workshop at **http://127.0.0.1:8790/**.
- Run browser-only idea and visual experiments without pretending they came from the model.

## What it cannot do
- Approve a Workshop proposal, run notebook code, restart a kernel, change Workshop settings, access arbitrary files or open host browser windows on TASS's behalf. The separate test-card pilot never exposes arbitrary commands.
- Grant Identity permissions, verify owner login, or independently prove that an execution was correct.
- Expose the model externally; ports bind to `127.0.0.1` on the host. **Loopback-only is not authentication.** The existing Workshop's authentication and authorization gaps remain open.

## Start and stop

The normal `workspace/run.sh` starts the model, Liaison, Workshop, and Atrium in the same Compose project. It will build the Atrium service and expose only localhost port 8791. Navigate to `http://127.0.0.1:8791/` or use the **✦ Atrium** link in the Workshop header.

```bash
cd ~/tass-workspace/workspace
./run.sh
# On WSL, after health checks:
wslview http://127.0.0.1:8791/
# Status and logs:
./run.sh status
./run.sh logs
# Stop entire stack intentionally:
./run.sh stop
```

**Do not run `./run.sh` while troubleshooting another running stack without reviewing its existing behavior:** it can stop a separate identity-liaison Docker stack as part of its normal start procedure.

## Diagnose a connection before testing features

Atrium has a **Check TASS** control. It checks the existing Workshop `/api/status` and reports separately whether Atrium, the Workshop and the model are reachable. The output avoids tokens and internal container addresses.

A CLI equivalent, which **does not start or stop any containers**, is:

```bash
curl --noproxy '*' -sS --max-time 5 http://127.0.0.1:8791/bridge/diagnostics
echo
curl --noproxy '*' -sS --max-time 5 http://127.0.0.1:8790/api/status
echo
```

If port 8791 cannot be reached, check which process owns it with `ss -ltnp '( sport = :8791 )'` and verify the Atrium container is running. If the Workshop is unreachable, resolve that existing service before trying chat. Do not run the full stack start script merely to troubleshoot Atrium, since it may stop another Liaison.

Atrium also has **Try a real task**. It drafts a request for TASS to inspect `projects/hello-tass`, explain its effect, and propose a run. **Sending is still a separate owner action**. Any execution needs review in the original Workshop. A model response is not evidence of a completed run; use its actual result card and audit evidence. The **Copy report** button copies non-secret diagnostics for troubleshooting.

## Test evidence

- From the Workshop, click **✦ Atrium**: it should open a new tab at port 8791.
- The Atrium status should say **LIVE TASS CONNECTED** only if the model is reachable.
- Send a distinctive, harmless prompt in Atrium. Observe the *real* reply and the same conversation appearing at port 8790.
- Ask TASS to propose a harmless notebook action. Atrium must NOT offer approval/execution; the original Workshop must display the review card.
- Stop the model: Atrium should switch to its offline playground mode rather than invent a reply.

Tests: `python3 -m unittest discover -s workspace/atrium/tests -v` from the repository root. These cover a simulated backend, **not** the owner's local model.

## Teaching TASS

The approved, reusable procedure is: “When Dustin asks to see our Atrium, tell him to open the fixed local URL `http://127.0.0.1:8791/` or click **✦ Atrium** in the Workshop. Explain that it shares your active conversation, and that code approvals remain in the Workshop. Do not claim you launched the Windows browser unless there is an actual trusted host-side launcher receipt.”

The separate GitHub communication handoff can show TASS these instructions, but the real TASS model will only persist them as a known capability after they are added to its authorized mind/library catalog. **This PR alone does not automatically teach the model or enable host-browser control.**

## Editing

All source files are in this folder. Changes require review, rebuilding/restarting the Atrium container and owner verification. Do **not** serve a model-writable HTML/JS directory as trusted same-origin content on the Workshop, which exposes execution controls.
