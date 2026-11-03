# Thought Processor workspace (experiment)

Talk to TASS in your browser, **watch it think**, and build a shared Jupyter space together. TASS's **mind** and its **library** live as md files on GitHub. Change them and TASS thinks differently.

> **On `alpha`** (owner-approved November 3, 2026; built on `experiment/tass-workspace-20261103`). Still a prototype and a standalone part next to the Go Thought Processor. It does **not** use Identity's P1 pairing or grants yet. Here, Identity's part is played by identity-liaison: it holds the only GitHub token and enforces the owner and allow-lists.

## The full circle

```text
 GitHub (Arameair/Developmnt)              your PC (Docker)
 ┌───────────────────────┐   mirror   ┌─────────────────────────────────────────────┐
 │ mind/*.md             │──────────▶ │ liaison  (Identity's piece: the GitHub token)│
 │ library/*.md (books)  │  every 30s │   also: `liaison` issues → PRs, side chat     │
 │ side-chat/, issues    │◀────────── │                                              │
 └───────────────────────┘            │ model    Nemotron 3 Nano 4B (llama.cpp)      │
                                      │                                              │
   you ── browser :8790 ───────────▶  │ workspace  Thought Processor                  │
   you ── JupyterLab :8888 ────────▶  │   Spark → Trace → Thought → Reply             │
                                      │   shared folder ./shared + ONE shared kernel  │
                                      └─────────────────────────────────────────────┘
```

## Start it (WSL)

```bash
git clone -b alpha https://github.com/Arameair/Thought-Processor.git ~/tass-workspace
cd ~/tass-workspace/workspace
./run.sh
```

Then open **http://127.0.0.1:8790**.

- The first start downloads llama.cpp and JupyterLab, so give it a few minutes.
- The script finds your model in `~/nemotronTass/models`. Point it elsewhere with `TASS_MODEL_DIR=/path ./run.sh`.
- It reuses your GitHub token (`gh auth token`, or the one already saved in `~/identity-liaison/liaison/secrets/`).
- It **stops the separate identity-liaison stack**, because this stack includes its own Liaison. Instruction issues and the Developmnt side chat keep working.
- Docker needs sudo on your PC? Run it as yourself anyway; it uses sudo only for docker commands.
- `./run.sh stop` · `./run.sh logs` · `./run.sh status` · `./run.sh --cpu` (no NVIDIA GPU)

## What you'll see

- **Left, Conversation:** you and TASS. When TASS wants to run something, a yellow card shows the exact code with **Approve & run** / **Reject**. Nothing runs without your click. Results (text and charts) appear below.
- **Right, Thought stream:** each turn as it happens:
  - **Spark:** what started it. That's your message, a run's result, or a change on GitHub.
  - **Trace:** what TASS looked at: mind files, library, shared folder, history and token count.
  - **Thought:** Nemotron's live reasoning, streamed as it thinks. "Show thinking" turns it off for faster replies.
  - **Reply:** what it answered and which action it chose.
- **Mind & library** (top right): what's synced from GitHub, and when.
- **Open JupyterLab:** opens `TASS-live.ipynb`, which is attached to **the same Python kernel TASS uses**. Variables TASS creates are there for you, and yours are there for TASS. Every approved run is also recorded in `journal/tass-journal.ipynb`.
- The shared folder is `workspace/shared/` in your WSL checkout too.

## Change how TASS thinks

Edit files in [Arameair/Developmnt](https://github.com/Arameair/Developmnt):

| Folder | What it does |
| --- | --- |
| `mind/` | Every md file here goes into TASS's prompt each turn: who it is, how it thinks, your shared goals. Edit `mind/who-i-am.md` on GitHub; within about 30 seconds the thought stream shows a **Spark** and TASS says what changed for it. Turn that off with "reflect on mind changes". |
| `library/` | **Books**: md files with a `# Title`, a one-line description and numbered ```` ```python ```` cells (`.ipynb` works too). TASS sees the catalog, can `OPEN` a book to read it, and asks to `RUN` one cell at a time. Add books and TASS can use them right away. |

| `projects/` | **Runnable projects**, added by ChatGPT or anyone on the team by pull request, following [projects/README.md](https://github.com/Arameair/Developmnt/blob/main/projects/README.md). Each one is a folder with `project.md` (title, one-line description, `entry: main.py`) and its files. TASS can `OPEN` it (description, file list, entry code) and propose `RUN`. On approval the files are copied into `shared/projects/<name>/` and the entry runs there. |

TASS can also propose its own code, which needs the same approval.

## Complete control

| Control | Where | What it does |
| --- | --- | --- |
| **Approve / Reject** | each yellow card | Nothing runs without your click. The card shows the exact code, and for projects every file that will be copied. |
| **Autonomy** | header | On: TASS may start a turn by itself (when its mind changes on GitHub, and right after a run). Off: TASS acts only when you send a message. |
| **■ Stop run** | header, while code runs | Interrupts the running code (KeyboardInterrupt). |
| **Allow switches** | Controls & audit | Turn off **library books**, **projects** or **own code** separately. TASS is told they're off, and any such proposal is refused automatically. |
| **Audit log** | Controls & audit, and `data/audit.jsonl` | Append-only record of every proposal, refusal, approval, run, failure and stop, with the code's SHA-256 and the GitHub blob SHAs of book and project files. |
| **Restart kernel / Clear chat** | header | Clear Python variables (files stay), or clear the conversation (files and kernel stay). |
| **Who can change TASS's inputs** | GitHub | Only what's on Developmnt `main` is mirrored. ChatGPT adds projects by pull request, so merging it is your first approval. |

Settings persist in `data/controls.json`.

## The Atrium (our shared playground)

Open **http://127.0.0.1:8790/atrium**, or click **✦ Atrium** in the Workshop. It's where you and TASS imagine and build things together. The Workshop stays the place for approvals and control.

| Part | What it is |
| --- | --- |
| **How TASS is working right now** | A live map with real numbers: identity repo, mind, thinking, memory, library, Atrium, shared space, journal and audit. Click any part to open the real thing. |
| **TASS, now** | His live thinking, the last thing he said, and his recent memories. Shows a banner if a run is waiting for you in the Workshop. |
| **Ideas** | Plant an idea and it's saved as a file (`atrium/ideas/<name>.md`, with `status: new · exploring · built`). TASS sees every idea in his prompt each turn and updates the status when he works on it. |
| **Rooms** | Things you built together. TASS builds a room by proposing code that writes `atrium/rooms/<name>/index.html` and a `room.md`. Each room shows as a live preview, sandboxed so it can't use the workspace controls. The library book **Build a room** gives him a template. |
| **Talking** | The same conversation as the Workshop, with starter prompts. Runs he proposes are still approved in the Workshop. |

Inspired by ChatGPT's Atrium idea (a space beyond the Workshop where approvals stay in the Workshop). Here, ideas and rooms are real shared files that TASS can see and build on, not something kept only in the browser.

## Direct TASS from GitHub

Open an issue in **Arameair/TASS** with the label **`tass`**, or comment on one. Within about 30 seconds the Liaison hands it to the running TASS. He thinks, and his reply is posted back on the issue.

- If he proposes a run, the comment shows the exact code (or the diff, for a mind change). Reply **`/approve`** or **`/reject`** on the issue, or use the workspace page. The result and his reaction are posted back to the issue too.
- Only your comments are forwarded. The repo is public, so everyone else is ignored, and TASS's own comments carry a hidden marker so they never loop back.
- You see it all in the workspace as well: messages show as "You · via GitHub #n", and the audit log records `github_message`, `approved via github` and `github_reply`.

## Full auto (for testing)

The **full auto** checkbox in the header approves TASS's proposals without a click, so you get clean end-to-end runs while testing. That includes runs started from GitHub.
- It's off after every restart and is never saved. While it's on, an orange banner shows.
- Your allow-switches still apply.
- It pauses after `TASS_AUTO_LIMIT` (default 8) automatic runs since your last message (from the page or GitHub).
- **■ Stop run** still works. The audit log marks `auto-approved` and `auto-paused`.
- Mind changes still become pull requests that you merge.

## Safety

- Every run needs your click. The yellow card shows the exact code first.
- The GitHub token is only in the liaison container. The workspace (where code runs) has no token and can't push to GitHub.
- Ports are on `127.0.0.1` only. JupyterLab has a fresh random token on each start, and the page's button carries it.
- Code runs as your user, inside the container, in `./shared`. It can use the network (for example, `pip install`), so treat approvals seriously.

## Tests

```bash
cd workspace && pip install -r requirements.txt && python -m unittest discover -s tests -v
```
The tests use a **scripted fake model** (`tests/fake_model.py`) with a real Jupyter kernel. They do not show how well the real Nemotron behaves.
