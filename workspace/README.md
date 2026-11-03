# Thought Processor workspace (experiment)

Talk to TASS in your browser, **watch it think**, and build a shared Jupyter space together. TASS's **mind** and its **library** live as md files on GitHub. Change them and TASS thinks differently.

> **Experimental** (`experiment/tass-workspace-20261103`, branched from `alpha`). This is a standalone prototype next to the Go Thought Processor. It does **not** use Identity's P1 pairing or grants yet. Here, Identity's part is played by identity-liaison: it holds the only GitHub token and enforces the owner and allow-lists.

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
git clone -b experiment/tass-workspace-20261103 https://github.com/Arameair/Thought-Processor.git ~/tass-workspace
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

TASS can also propose its own code, which needs the same approval.

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
