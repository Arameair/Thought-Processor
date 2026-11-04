# TASS's Go core: first vertical slice

One working path, built so it keeps running without the internet:

```
you ──► tass-core (Thought-Processor) ──► Thought Engine ──► Nemotron (llama.cpp, local)
            │   ▲                              │
            │   └──── reply, request ID, ◄─────┘   records every request and a 20-second local pulse
            │         elapsed time, error
            ├── tass.md (local copy he runs on)
            └── GitHub sync, its own loop: Identity's policy · tass.md · messages · replies
```

| Repo | Its part here |
| --- | --- |
| **TASS** | `tass.md` on branch `live`: who TASS is, his memory, what he's doing now. The versioned record. |
| **Thought-Processor** | `core/` (this folder), the `tass-core` program: the local page, conversation, the local `tass.md`, GitHub sync, and asking Identity before acting. |
| **Thought-engine** | Branch `experiment/nemotron-link-20261104`: `POST /api/model/requests` to Nemotron, plus the 20-second local pulse, and records of both. |
| **identity** | `policies/tass.json`: what TASS may do and whom he obeys. Checked before every action. |
| Nemotron 3 Nano 4B | Weights on local storage, served by llama.cpp in Docker. Never in Git. |

The Python workspace and the Atrium are untouched. They are still on `alpha` and the experiment branches.

## Start it (WSL)

```bash
cd ~/tass-workspace && git fetch && git checkout experiment/go-core-20261104 && cd core
./start.sh
```

Then open **http://127.0.0.1:8795**.
- `./start.sh status` shows what's running, the engine's last pulse and the last GitHub sync.
- `./start.sh logs` follows the logs. `./start.sh stop` stops the core.

You need Go 1.22+ and `gh` signed in. The script:
1. starts the model container (the same one the workspace uses);
2. fetches the Thought Engine branch with your `gh` login and builds it;
3. builds `tass-core`, then starts both.

It also stops the Python workspace's liaison, so TASS doesn't answer GitHub twice.

**First time:** TASS does nothing until Identity's policy exists on `Arameair/identity` `main`. Merging the policy pull request is that approval.

## Which `tass.md` is authoritative

- **TASS runs on the local copy** (`~/.tass/tass.md`). He reads it for every reply and writes his `REMEMBER:` and `NOW:` lines into it, with or without the internet.
- **The GitHub copy (`Arameair/TASS@live:tass.md`) is the versioned record.** Every sync merges the two, section by section (`## Who I am`, `## Memory`, …):
  - **Changed on one side only:** that side wins. When you edit on GitHub, TASS thinks with the change on the next sync.
  - **`## Memory` changed on both sides:** the lines are combined. A line one side removed stays removed.
  - **Any other section changed on both sides:** TASS runs on the GitHub version of that section, and opens a pull request into `live` holding his version. **Merge** it to take his; **close** it to keep yours. Nothing is lost.
- The core only writes to `live`. Identity's policy refuses `main`, `alpha` and `stable`.

## The two loops

| Loop | Where | How often | Needs internet |
| --- | --- | --- | --- |
| **Pulse** | Thought Engine | every 20 s: checks Nemotron, appends to `~/.thought-engine/tass/pulse.jsonl` | no |
| **GitHub sync** | tass-core | every 20 s while it works. After a failure it waits 20, 40, 80, 160 s, then 5 min, and is back to 20 s as soon as a sync succeeds. Reads use ETags, so "no change" costs no rate limit. Every attempt goes to `~/.tass/sync.jsonl`. | yes, but nothing waits on it |

Replies to GitHub messages wait in `~/.tass/state.json` until GitHub is back.

## Identity before action

`policies/tass.json` (in `Arameair/identity`) lists whom TASS obeys and allows each action separately:
- `model.request`
- `github.read_messages`
- `github.reply`
- `tass_md.sync`
- `tass_md.conflict_pr`

How it is enforced:
- The core checks the policy **before** each action.
- The **Thought Engine checks it again before running the model**. It reads the same local copy, `~/.tass/identity-policy.json`. With no policy, it refuses the request and records the refusal.
- The last good policy is kept locally, so permissions hold offline.
- If the policy is deleted on GitHub, the local copy is deleted on the next sync, and the engine refuses from then on.

This is a policy file, not yet Identity's live `authorize` API. That API answers DENIED for everything on alpha; its grants work on the P1 experiment. Moving the check onto it is a later step.

## Evidence against the four acceptance criteria

| # | Criterion | Shown by | Result |
| --- | --- | --- | --- |
| 1 | **Local-first:** TASS keeps answering with the internet disconnected | Real processes in the sandbox: GitHub cut off; TASS answered and remembered, also after a restart while still offline. Test `TestOutageBackoffAndRecoveryKeepLocalState`. | PASS with a **TEST FIXTURE model**. Real Nemotron: **NOT RUN** (no GPU or weights here). |
| 2 | **Persistent identity:** a local `tass.md` and a versioned GitHub copy, with an authority and conflict rule | Real GitHub: the first sync took `tass.md` from `TASS@live`. Tests: merge rules, conflict pull request, no duplicate pull requests. | PASS for reads and the rules. Writing to real GitHub: **NOT RUN** here, because the sandbox's network proxy refuses API writes (HTTP 403). The same code path passes against the fake GitHub. |
| 3 | **20-second pulse:** recorded locally; GitHub sync separate, with retries and backoff | Real engine: `pulse.jsonl` every 20 s while GitHub was down. The core's backoff was seen live and checked in a test (20 s → 5 min). | PASS |
| 4 | **Evidence:** start, stop, reconnect and outage recovery, without losing local state or bypassing Identity | Real processes: stop and start offline kept memory and permissions; reconnecting resumed syncing. The engine refused with no policy, and for another identity. Tests: Dustin's offline edit and TASS's offline memory both survive; withdrawn permission stops thinking. | PASS (fixtures as noted) |

Fixtures used: a fake llama.cpp and a fake GitHub API, both labelled **TEST FIXTURE** in the code. The real-process run used:
- the real `thought-engine` and `tass-core` binaries;
- real GitHub reads, through a small forwarding proxy that was stopped to cut the connection.

**Not built yet:**
- TASS doesn't see the Thought Engine's Spark/Trace/Thought pipeline; this slice uses only its model link;
- streaming the model's thinking live;
- Discord, lessons and Developmnt (left out, as decided).

## Tests

```bash
cd core && go test -race ./...                       # tass.md merge, sync, outage/backoff, conflicts, Identity
cd vendor/thought-engine && go test ./...            # engine incl. TestModelRequest*, TestLocalPulse*
```
