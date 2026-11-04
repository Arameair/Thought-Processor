# TASS's Go core in Docker (Thought-Processor #44)

One system you can **check, start, inspect, update and recover**. It runs the Thought Engine and tass-core as two containers, and uses the **Nemotron model container that is already running** in the workspace stack.

```
 your browser ──127.0.0.1:8795──►┌──────── network namespace of "engine" ─────────┐
                                 │  core (tass-core)  ──127.0.0.1:7798──► engine   │──► model:8081  (existing
                                 │  page, GitHub sync, Identity checks   (Thought  │     tass-workspace-model-1,
                                 │                                        Engine)  │     same GPU, no second copy)
                                 └──────────────────────────────────────────────────┘
 The engine listens only on 127.0.0.1 INSIDE that namespace: none of its routes, including the older
 unauthenticated ones, can be reached from your computer or from any other container.
```

| Piece | What it is |
| --- | --- |
| `docker/core.Dockerfile` | Multi-stage Go build. Static binary, `USER 65532` (compose runs it as your own user ID), version in a label and in `/api/health`. |
| `docker/engine.Dockerfile` | The same pattern for the Thought Engine, built from its **pinned commit** (`ENGINE_COMMIT_PIN` in `tass.sh`). |
| `compose.yaml` | Project `tass-core`, two services. Both containers: read-only root filesystem, all capabilities dropped, no-new-privileges, healthchecks run by their own binaries. The model is reached on the existing `tass-workspace_default` network. |
| `tass.sh` | Every command below. |

## Commands

```bash
cd ~/tass-workspace && git fetch && git checkout experiment/go-core-docker-20261104 && cd core
./tass.sh check          # read-only preflight and inventory: writes nothing, stops nothing
./tass.sh start          # first time: builds this checkout's version, then starts it and validates it
./tass.sh status         # READY / WORKING / BUSY / BLOCKED / UNKNOWN, versions, pulse, GitHub sync
./tass.sh logs
./tass.sh check-update   # current vs candidate (branch head), commit list, engine pin change. Applies nothing.
./tass.sh update [SHA]   # asks first. Backup → build → switch → validate → restore on failure.
./tass.sh rollback       # back to the previous version. Data untouched; add --restore-data <backup> to restore it.
./tass.sh backup
./tass.sh stop           # stops only the two Go containers
```

- **State meanings.**
  - READY: idle; the engine and model answer, and the Identity policy is valid now.
  - WORKING: a thought cycle is running.
  - BUSY: the engine's model slot is busy.
  - BLOCKED: something is missing, and `status` says what (no or expired policy, engine or model down).
  - UNKNOWN: the core isn't answering.
- **The model.** `./tass.sh` never starts or stops it. If it isn't running, `check` says so and `start` refuses.
- **The Python workspace, liaison and Atrium are untouched.** While the Python liaison runs, the Go core keeps **GitHub messages paused**, so TASS never answers twice. Cutting over needs your explicit OK.
- **The host-process core** (`./start.sh`) uses the same port. `start` asks before stopping it.

## How an update works (and fails safely)

1. `check-update` fetches the tracked branch (`TASS_TRACK`, default `experiment/go-core-docker-20261104`). It only fetches into the remote-tracking ref; **your working tree and branches are never touched**. It shows:
   - the current and candidate SHAs;
   - the commits in `core/`;
   - a diff summary;
   - whether the engine pin changes.
2. `update` asks before doing anything. Then:
   1. builds the candidate from a separate checkout (`~/.tass-deploy/src/core-<sha>`, a git worktree) and the engine at the candidate's pin;
   2. stops the Go containers;
   3. **backs up `~/.tass` and `~/.thought-engine/tass`** (`~/.tass-deploy/backups/`, mode 600);
   4. switches to the candidate and validates it:
      - both containers healthy;
      - `/api/health` reports the **candidate's exact SHA**;
      - the state is **no worse than before** (READY stays READY);
      - **no record file shrank or disappeared**.
3. **If validation fails,** the previous version comes back, together with your data from the backup. What was there is **moved** to `~/.tass-deploy/failed/<time>/`, never deleted. Then the previous version is validated again.
4. Nothing deploys itself. There are no automatic merges, branch switches or resets, and no "new commit, so redeploy".

Records:
- versions: `~/.tass-deploy/current.env` and `previous.env`, holding SHAs, image tags, image IDs, and the compose file used;
- history: `~/.tass-deploy/history.log`.

Old images and build checkouts are kept, so rollback works offline.

## Data and secrets

| Where | What | Who sees it |
| --- | --- | --- |
| `~/.tass` (0700) | `tass.md`, `memory.jsonl`, `conversation.jsonl`, `sync.jsonl`, `conflicts/`, `state.json`, `api-key` | core (read-write) |
| `~/.tass/grant/` | Identity's (prototype) policy with its confirmation time | core writes it; the **engine reads only this folder** (read-only) |
| `~/.thought-engine/tass` (0700) | `cycles.jsonl`, `pulse.jsonl`, `model-requests.jsonl`, `calls.jsonl`, `api-key` | engine (read-write). The core gets **only the key file**, read-only. |
| `~/.tass-deploy/secrets/github_token` (0600) | GitHub token, copied from `gh auth token` | **core only**, mounted read-only at `/run/secrets` |
| `~/.tass-deploy/backups/` (0700) | backups. They include the key files, so treat them as private. | you |

Nothing secret is in an image, a build argument, the repository or a log. Containers run as **your** user ID, so every file stays yours.

## Evidence (2026-11-04)

| # | Check | Where | Result |
| --- | --- | --- | --- |
| 1 | `check` writes nothing (file listing before and after) and reports the model, liaison, ports, policy and data | sandbox, Docker 29.9.2 / compose 5.6.1 | PASS |
| 2 | Go images build (core + engine at the pin) | Sandbox: both, with **stand-in base images** (Docker Hub isn't reachable there). GitHub CI run 41847703616: the core image with the real `golang:1.24-alpine` / `alpine:3.20` bases (label = commit, `user=65532:65532`), plus compose validated. The engine image isn't built in CI because Thought-engine is private. | PASS |
| 3 | `start` with a running "liaison" and "model": nothing of theirs stopped; messages paused; exactly one model container | sandbox, **simulated** workspace containers | PASS |
| 4 | The engine reaches the existing model container by network (`model:8081`) | sandbox, **fixture model** | PASS |
| 5 | Stop, then start: every record kept (cycles 6 → 12, conversation 1 → 2), the policy still valid, READY | sandbox | PASS |
| 6 | GitHub cut: still READY, a cycle ran, failed syncs recorded; reconnects by itself | sandbox, GitHub reads through a local proxy | PASS |
| 7 | Update to a good candidate (PASS). Update to a **deliberately broken** candidate: FAIL detected, previous version **and data** restored (counts and `tass.md` checksum identical), failed data moved aside. Update with an engine pin change (PASS). Rollback twice (PASS). | sandbox, `docs/evidence/docker-update-20261104.txt` | PASS |
| 7b | The engine refuses on its own when the Identity grant file is missing (cycle `denied`, **no model request made**) | sandbox | PASS |
| 8 | Keys: status without key → 401; the engine isn't reachable from the host or from another container | sandbox | PASS |
| 9 | **Real Nemotron thought cycle, on your PC** | — | **NOT RUN**: owner test below |

## Owner acceptance (about 15 minutes)

You need the workspace model running as usual. Identity PR #31 can stay unmerged for steps 1–3: you'll see BLOCKED, which is correct.

1. `./tass.sh check`. Expected: `[ok]` lines, plus notes. Your Python stack keeps running (`docker ps`).
2. `./tass.sh start`. Expected: `PASS: both healthy … version …`. Run `docker ps` again: still **one** model container, and the workspace containers are still up.
3. `./tass.sh status`. Expected: BLOCKED with "no Identity policy yet" until #31 is merged; otherwise READY.
4. *(After reviewing and merging #31.)* Open http://127.0.0.1:8795 and ask **How many sections does your tass.md have?**

   Expected: a real Nemotron reply and a cycle card showing a Spark, a **LOCKED** prediction, EXECUTED and an evaluation. Paste the last line of `~/.thought-engine/tass/cycles.jsonl` into #44.
5. `./tass.sh stop`, then `./tass.sh start`, then `./tass.sh status`. Expected: READY, and the earlier cycle is still on the page.
6. Turn your network off, then ask TASS something. Expected: he answers; `status` shows GitHub not synced. Turn the network back on; within a few minutes, `GitHub synced True`.
7. `./tass.sh check-update`. Expected: "Up to date", or a candidate with its commit list. Apply it only if you want to: `./tass.sh update`.
8. `./tass.sh rollback`, then `./tass.sh rollback` again. Expected: each step ends with PASS, and you finish on the version you started with.

## Known limits

- Containerization keeps the existing pulse: a 20-second **model health pulse**, plus thought cycles **on demand**. **Start/Pause/Resume/Step Once, the 10 observable ticks and Correct-a-result are not built yet.** The issue allows staging them after real-model acceptance.
- Identity is still the **prototype policy file** (PR #31, owner review). It is not a live Identity grant.
- The engine's older routes are unauthenticated. Here they are reachable **only inside its own container**.
- Base images are pinned by tag, not digest. Image IDs are recorded in `current.env` at build time.
- The engine reports its git commit as "unknown" in its own log. The pinned commit is in the image label and in `current.env`.
- Real-model behaviour, the GPU and your Windows/WSL Docker were **not** exercised here.
