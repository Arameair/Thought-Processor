# TASS as one Docker application (Thought-Processor #44, one-app directive)

One application you **check, start, use, update and recover** with one script. It is four containers in one Compose project, `tass-app`:

```
 your browser ──127.0.0.1:8796──► identity  (tass-identity, Go)   create YOUR entity · unlock · lock
      │                              ▲  authorizes every action (service keys)
      │                              │
      └──────127.0.0.1:8795──►┌──── network namespace of "engine" ───────────────┐
                              │ core (tass-core)  ──127.0.0.1:7798──► engine      │──► model:8081
                              │ page · tasks ·                        (Thought    │    llama.cpp, ONE per
                              │ pulse controls                         Engine,    │    computer, internal
                              │                                        20-s pulse)│    network only
                              └───────────────────────────────────────────────────┘
```

| Piece | What it is |
| --- | --- |
| `docker/identity.Dockerfile` | tass-identity from the Identity repo at `IDENTITY_COMMIT_PIN`. |
| `docker/engine.Dockerfile` | Thought Engine at `ENGINE_COMMIT_PIN`. |
| `docker/core.Dockerfile` | tass-core from this checkout. |
| `compose.yaml` | The four services. Every Go container: read-only root filesystem, all capabilities dropped, no-new-privileges, your own user ID, healthchecks. The model is on an **internal** network: no published port and no internet. |
| `compose.gpu.yaml` | Added automatically when Docker can use an NVIDIA GPU. |
| `compose.external-model.yaml` | Added when `MODEL_MODE=external`: use a model that's already running instead of starting one. |
| `tass.sh` | All commands. Settings in `~/.tass-app/app.env`. |
| `test/` | The clean-host acceptance driver and a **TEST FIXTURE** model image (never used for a real install). |

Nothing depends on the Python workspace, the liaison or the `tass-workspace` network. Everything lives in `~/.tass-app` (data, keys, backups, build checkouts), never in `~/.tass`.

## First run

```bash
cd core
./tass.sh check        # writes ~/.tass-app/app.env the first time; otherwise read-only
nano ~/.tass-app/app.env   # MODEL_DIR + MODEL_FILE (your .gguf), or MODEL_MODE=external
                           # GITHUB_REPO=owner/repo to connect GitHub (token from gh auth token)
./tass.sh check        # must end with "Ready"
./tass.sh start        # builds the pinned version, starts the 4 services, validates them
```

Then open **http://127.0.0.1:8796**, create your own entity (a name and a passphrase), and open **http://127.0.0.1:8795** to talk to it and run the work pulse.

- **A fresh install has no entity.** Nothing is downloaded or built in: no one else's entity, identity, memory, document or policy. The core says BLOCKED ("Identity NO_ENTITY") until you create one.
- **LOCKED after every restart.** Unlocking is held in memory only. After any restart or update, unlock again with your passphrase. While LOCKED, nothing reaches the model and no task can be queued.
- **Keys are made on your computer.** `tass.sh start` generates the service keys in `~/.tass-app/secrets` (0600). They're mounted read-only and are never in an image or in Git; the acceptance run searches every image layer for them.

## Commands

```bash
./tass.sh check          # read-only preflight: Docker, CPUs/memory/disk, GPU, model file, ports, other models, data
./tass.sh start
./tass.sh status         # Identity state, READY/WORKING/BUSY/BLOCKED/UNKNOWN, last pulse tick, versions
./tass.sh logs
./tass.sh check-update   # current vs candidate and what changed; applies nothing
./tass.sh update [REF]   # asks first; backup → build → switch → validate → automatic restore on failure
./tass.sh rollback       # previous version (add --restore-data <backup> to restore data too)
./tass.sh backup
./tass.sh stop           # only this application's containers
```

**Validation** (after start, update and rollback) passes only if all of these hold:
- all services are healthy;
- exactly one model container is running (bundled mode);
- the running core reports exactly the expected commit;
- the entity is the same one as before;
- a LOCKED Identity makes the core BLOCKED; with an UNLOCKED Identity, the state is no worse than before;
- no record file shrank or disappeared.

A failed update puts back the previous version **and** the data from the backup taken just before. The data that was there is moved to `~/.tass-app/deploy/failed/`, never deleted.

## One model per computer

- **Bundled** (`MODEL_MODE=bundled`): the app runs llama.cpp with your weights mounted read-only from `MODEL_DIR`. It uses the GPU when Docker can use it (`MODEL_GPU=auto`), otherwise the CPU.
  - `check` stops if any other model server is already running.
- **External** (`MODEL_MODE=external`, `MODEL_NETWORK`, `MODEL_URL`): the engine joins that model's Docker network and nothing new starts.
  - On a PC where the workspace's Nemotron already runs: `MODEL_NETWORK=tass-workspace_default` and `MODEL_URL=http://model:8081`. That container isn't stopped or changed.

## GitHub and the lessons loop

Set `GITHUB_REPO` (and `GITHUB_BRANCH`, default `live`) in `~/.tass-app/app.env`. At start, `tass.sh` saves a token from `gh auth token` (or `GITHUB_TOKEN`) as `~/.tass-app/secrets/github_token` (0600). It's mounted read-only and is never in an image. Without `GITHUB_REPO`, everything stays on this computer.

- **Document:** the entity's `tass.md` comes from the repo. The app syncs it every 20 s with ETags and a section-by-section merge. If both sides changed the same section, it opens a pull request instead of choosing. A repo without the file gets the entity's own starter, but a local starter is never merged over a real document.
- **Lessons:** turn on **Work pulse every 20 seconds** and **Learn new lessons from GitHub** on the page (8795). Then each pulse:
  1. Looks at `lessons/` in the repo. One request, usually a free 304 when nothing changed.
  2. Downloads new or changed lessons and keeps them locally.
  3. Hands **one** lesson at a time to the engine's pulse. The entity studies it **without** the expected answers.
  4. The **app grades** the answers against each step's `Answer:`. A wrong answer means another try on a later pulse, up to 3; then the lesson is STUCK until you press **Try again**.
  5. A lesson that passes waits for **Approve**. Approved, it's LEARNED: one line in `## Learned` of `tass.md`, synced to GitHub. The entity is given its learned lessons in conversation.
- **The setting sticks.** The pulse comes back on by itself after every restart, as soon as you unlock Identity.
- **Identity decides each step.** LOCKED means no GitHub sync, no lessons and no pulse.
- **Lesson format:** `lessons/README.md` in the TASS repo (Teach / Check / Answer). The app never runs code from a lesson. Older code-cell lessons show up as "needs the workspace".

## The work pulse

Every 20 s (`PULSE_EVERY`) the engine runs one tick: it observes pending work, decides, starts or continues one authorized job, observes results and reports. Each tick is a line in `~/.tass-app/data/engine/ticks.jsonl` and appears on the page.

- **States:** READY, WORKING, BUSY (model slot in use) and BLOCKED (Identity said no). UNKNOWN appears on the page when the engine doesn't answer.
- **Controls:** the pulse starts PAUSED after every start. Use **Start**, **Pause**, **Resume** and **Step once** on the page or via `POST /api/pulse`.
- **No duplicate runs.** A job is saved as RUNNING before it starts. Later ticks say CONTINUE and never start it again. A job cut off by a restart becomes INTERRUPTED and is not re-run on its own.
- **Other entities' work is refused.** A job for any entity other than this installation's is refused at submit. If one is found queued, it's marked REFUSED, so it never blocks the queue.
- **Conversation stays available.** Talking doesn't go through the pulse. A message gets a receipt, or `429` if the inbox is full (#46: never silently dropped).

## Threat model and its limits (#47)

- **What protects the two pages:**
  - They listen on 127.0.0.1 only.
  - Each page embeds a token that's new for every process start. POSTs need that token, JSON and a local Origin.
  - The token is a **CSRF and other-website defense, not authentication**. Any program running as you on this computer can read the page and get the token. Identity's passphrase is what authenticates the owner.
- **Identity owner routes** (create, unlock, lock) also require `Host: 127.0.0.1` or `localhost`. Services inside the app reach Identity as `identity:8796` and are refused there. The acceptance run proves this from inside the core container.
- **Services** authenticate to Identity with their own keys. Identity returns a decision ID, which is recorded in `decisions.jsonl`. Every decision fails closed when Identity can't be reached.
- **At rest**, the entity's credential and verifier are protected by file permissions (0600/0700) only, not encryption. Anyone with your user account or root can copy them. A copied `entity.json` without its matching `credential` stays LOCKED. A copied pair plus the passphrase is a full copy of that entity.
- **Not covered:**
  - remote access;
  - multiple users on one computer;
  - a compromised Docker daemon or host;
  - side channels in the model;
  - memory and training (separate future components).

## Results (clean-host acceptance)

See `docs/evidence/one-app-clean-host-20261104.txt` for the full run. "Fixture" means the sandbox run with the TEST FIXTURE model and stand-in base images (Docker Hub isn't reachable there).

| Acceptance item | Fixture (sandbox) | CI | Owner's PC (real Nemotron) |
| --- | --- | --- | --- |
| Clean install: 4 services healthy, one start command | PASS (re-run on 8208895: 37/37) | NOT RUN (CI builds only the core image) | NOT RUN |
| Exactly one model; preflight refuses a second; external mode starts none | PASS | NOT RUN | NOT RUN |
| No Python workspace, liaison or tass-workspace dependency | PASS | — | NOT RUN |
| No keys inside any image (every layer searched) | PASS | — | NOT RUN |
| Fresh install: NO_ENTITY → BLOCKED, nothing personal | PASS | — | NOT RUN |
| Can't impersonate: no service key → 401; no page token → 403; from inside the app → 403; other entity's work refused (unit test) | PASS | — | NOT RUN |
| Enrollment: create own entity, wrong passphrase refused, unlock → READY | PASS | — | NOT RUN |
| Conversation with receipt; available while a long job runs (queued for the one model slot) | PASS | — | NOT RUN |
| Work pulse: PAUSED until Start; 10 ticks ~20 s apart; 45-s job, 1 START, 2 CONTINUE, result reported on a later tick | PASS | — | NOT RUN |
| Restart: same entity, LOCKED → BLOCKED, records persisted, pulse PAUSED | PASS | — | NOT RUN |
| Good update; broken update auto-restored (version + data); manual rollback | PASS | — | NOT RUN |
| GPU overlay (`compose.gpu.yaml`) and the real llama.cpp image | NOT RUN (no GPU, no Docker Hub) | — | NOT RUN |
| Real Nemotron answers through the app | NOT RUN | — | NOT RUN: owner verifies |

Found and fixed by this run: a failed update overwrote the rollback target (`tass.sh` now keeps it).

Lessons loop (`docs/evidence/lessons-loop-clean-host-20261104.txt`, TEST FIXTURE GitHub and model): **23 PASS, 0 FAIL**:
- token handling;
- document from the repo;
- chat;
- setting off → nothing looked at;
- setting on → the pulse runs, 3 lessons found, studied one at a time, graded by the app, one retry after a wrong answer;
- approval → Learned on GitHub;
- restart → the pulse comes back on by itself and nothing is studied again;
- a lesson added later is picked up;
- 304s when nothing changed.

Against your real GitHub and Nemotron: NOT RUN.

## Owner acceptance (on your PC, about 15 minutes)

Your existing containers aren't touched. Because your workspace's Nemotron is already running, use external mode so the PC still runs ONE model.

```bash
cd ~/tass-workspace && git fetch origin && git worktree add ~/tass-one-app origin/experiment/one-app-20261104
cd ~/tass-one-app/core
./tass.sh check                        # creates ~/.tass-app/app.env; it will [stop] on "Another model is already running"
nano ~/.tass-app/app.env               # MODEL_MODE=external  MODEL_NETWORK=tass-workspace_default  MODEL_URL=http://model:8081
./tass.sh check                        # must end "Ready"
sudo -v; ./tass.sh start               # don't run the script itself with sudo: it calls "sudo -n docker" when needed
                                       # ends with "PASS … Identity NO_ENTITY"
```

In `app.env`, also set `GITHUB_REPO=Arameair/TASS` (branch `live`). Merge TASS PR #23 (the starter lessons) first, or the pulse won't find any lessons.

1. Open http://127.0.0.1:8795: the red bar says NO_ENTITY and BLOCKED.
2. Open http://127.0.0.1:8796 and create your entity: a name and a passphrase only you know. With `GITHUB_REPO=Arameair/TASS`, its document is TASS's `tass.md` from `live`, so name it TASS. The download never does this for anyone: it's your choice on your PC.
3. Back on 8795, say "hi". Expected: a real Nemotron reply and a full thought cycle.
4. In Work pulse, queue "count your sections", press **Start**, and watch at least 10 ticks. Expected: START once, then CONTINUE while it runs, then OBSERVED. Then **Pause**.
5. In **Learning**, tick both switches. Within a minute or two, the three lessons appear and get studied one by one. Read the answers, then press **Approve** on each lesson that passed. Check that `tass.md` on `live` now has a `## Learned` section.
6. Run `./tass.sh stop && ./tass.sh start`. Expected: same entity, LOCKED and BLOCKED until you unlock. After unlock, the pulse comes back on by itself, and learned lessons stay learned.
7. Run `./tass.sh status` and paste the output into #44 with PASS or FAIL for each step.

To remove the test app afterwards, run `./tass.sh stop`. Your data stays in `~/.tass-app` until you delete it yourself.
