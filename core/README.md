# TASS's Go core: one thought cycle (Thought-Processor #42)

```
you ─► tass-core (Thought-Processor) ─► Thought Engine ─► Nemotron (llama.cpp, local)
          │  checks Identity first           │  observation → Spark → Trace → Thought   (its existing pipeline)
          │                                  │  plan: reply + proposed action + PREDICTION, locked before anything runs
          │◄──────── cycle (IDs, timing) ────┘
          │  runs the read-only action only if Identity allows it at that moment → receipt
          └─► engine evaluates: PREDICTION_CORRECT / INCORRECT / UNSCORABLE / NO_ACTION, and GOAL_MET / UNMET / UNKNOWN
```

| Repo | Its part |
| --- | --- |
| **TASS** | `tass.md` on branch `live`: the reviewed record of who TASS is. PR #22 brings it in line with this core. |
| **Thought-Processor** | `core/` (`tass-core`): the owner page, conversation, the local `tass.md`, the memory journal, GitHub sync, the read-only executor, and asking Identity first. |
| **Thought-engine** | Branch `experiment/nemotron-link-20261104`, pinned in `start.sh`. It holds: `-worker nemotron`, so the existing Spark/Trace/Thought pipeline runs on the local model; the cycle API; the model link; and the 20-second pulse. |
| **identity** | `policies/tass.json` (PR #31, **owner review**). A **PROTOTYPE** static policy, not a live Identity grant. |
| Nemotron 3 Nano 4B | Weights on local storage, served by llama.cpp. Never in Git. |

The Python workspace and the Atrium are untouched.

## Run it (WSL)

```bash
cd ~/tass-workspace && git fetch && git checkout experiment/go-core-20261104 && cd core
./start.sh check     # read-only: tools, ports, model, policy, conflicts. Changes nothing.
./start.sh           # starts the Thought Engine and tass-core. Stops nothing that's already running.
```

Then open **http://127.0.0.1:8795**. Other commands:
- `./start.sh status` and `./start.sh logs`;
- `./start.sh stop` stops only what this script started.

**Rollback:** `./start.sh stop`, then delete `~/.tass` and `~/.thought-engine/tass`.

- **The model must already be running.** Start it the usual way: `cd ../workspace && ./run.sh`, or only its container. The script never starts or stops the model.
- **If the Python liaison is running,** the script asks whether to stop it. The default is **no**: the Go core then starts with **GitHub messages paused**, so TASS never answers twice.
- **Nothing happens until Identity's policy is on `Arameair/identity` `main`.** That is PR #31: review it, then merge.

The owner test is in [ACCEPTANCE.md](ACCEPTANCE.md).

## Rules this slice follows

- **Conversation is not authority.** A reply is just text. The one action TASS may propose is read-only: count the sections, memory lines or words of his own `tass.md`.
  - The action runs only if Identity allows `action.read_tass` **at that moment**.
  - Its prediction is locked (time and SHA-256) **before** it runs.
  - There is no Full Auto in this core; nothing runs on its own.
- **Identity first, everywhere.** `tass-core` checks before each step. The engine checks again on arrival, and **again after waiting for the model slot**, so a queued request whose permission was withdrawn is refused.
  - A cached policy authorizes local work for at most `offline_grant_hours` (24 h) without GitHub confirming it. After that, everything fails closed.
  - GitHub actions always use a policy fetched in the same sync.
  - Deleting the policy on GitHub stops everything on the next sync.
- **Every state-changing local API needs a key.**
  - tass-core uses a fresh page token each start, plus `~/.tass/api-key` for local programs. Its status endpoint holds conversation, so it needs the key too.
  - The engine's model and cycle endpoints need `~/.thought-engine/tass/api-key`, which only tass-core is given.
  - **Open risk:** the engine's older routes (identities, grants, events, settings) are still unauthenticated, as before this slice.
- **Evidence is append-only.**
  - Engine: `cycles.jsonl`, `model-requests.jsonl`, `pulse.jsonl`, `calls.jsonl`.
  - Core: `conversation.jsonl`, `sync.jsonl`, `memory.jsonl`.
  - Unknown steps show **UNKNOWN**.
  - Evaluation is a deterministic rule. **No evaluator model is configured**, and every record says so.
- **`tass.md` is the reviewed identity.**
  - The core never writes it.
  - TASS's `REMEMBER` lines go to `~/.tass/memory.jsonl` with provenance (cycle ID, time, source), keeping the last 200. `NOW` is local state.
  - Edits made on either side merge section by section.
  - If the same section changed on both sides, TASS runs on the GitHub version. Both versions are kept in `~/.tass/conflicts/`, and a PR into `live` offers the local one.
- **The pulse is not thinking.** The engine records a 20-second local model check (`pulse.jsonl`) with or without the internet. A thought cycle runs only when something happens.
- **GitHub sync is separate.** It runs every 20 s, backing off to 40, 80 and 160 s, then 5 min, while unreachable. ETags mean an unchanged poll costs no rate limit, and nothing is written unless something changed.

## Evidence (2026-11-04, this sandbox)

| Check | How | Result |
| --- | --- | --- |
| One full cycle: event → Spark (6/10, Trace kept) → plan → locked prediction (8) → authorized read-only action (measured 7) → `PREDICTION_INCORRECT` + `GOAL_MET` | Real `thought-engine` (pinned commit) and real `tass-core` binaries; real GitHub reads (policy branch, `TASS@live`); the model was a **TEST FIXTURE** | PASS (fixture model) |
| Third related request formed a **Thought** through the engine's existing rule (3 related Traces) | same run | PASS (fixture model) |
| `REMEMBER` → memory journal with cycle ID; `tass.md` untouched | same run | PASS |
| Keys: say/status without a token → 401; engine cycle without its key → 401; key files 0600 | same run | PASS |
| GitHub cut: cycles still ran; the engine pulse kept recording; the sync backed off | real processes, proxy stopped | PASS (fixture model) |
| Both programs restarted **offline**: cycles rebuilt from `cycles.jsonl`; memory and policy kept; a new cycle ran | real processes | PASS |
| Revoked or stale grant, queued request re-checked, wrong identity, no policy | engine and core unit tests | PASS (fixtures) |
| Conflict: same section on both sides → PR into `live` + both sides in `conflicts/` | core unit test (fake GitHub) | PASS (fixture) |
| **Real Nemotron**, real GPU, real model identity | — | **NOT RUN** (no GPU or weights here). Owner test in ACCEPTANCE.md. |
| **GitHub writes** (pushing `tass.md`, conflict PR) on real GitHub | — | **NOT RUN** here (this sandbox's proxy refuses API writes). Owner test step 6. |

Screenshot of that run (fixture model): `docs/evidence/cycle-20261104-fixture.png`.

```bash
cd core && go test -race ./...                         # core: cycle flow, keys, grant expiry, sync, outage, conflicts
cd vendor/thought-engine && go test -race ./...        # engine: cycle end-to-end on its pipeline, grants, re-check, pulse
```
