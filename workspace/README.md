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

## The two stations: Processor and Atrium

Open each in its own browser:
- **Processor** (Claude's station): **http://127.0.0.1:8790/processor**, or **⚙ Processor** in the Workshop header. A live map of how TASS is working (identity, mind, thinking, memory, library, Discord, shared space, journal, audit), **Lessons** with real step progress, **Ideas** (`studio/ideas/*.md`), **Rooms** TASS builds (`studio/rooms/<name>/`) and the conversation.
- **Atrium** (ChatGPT's station): **http://127.0.0.1:8791/**, a separate origin with a chat-only bridge.

Runs are always approved in the Workshop.

## Discord

TASS learns to talk in Discord with the lesson **Talk in Discord** (`lessons/claude/01-talk-in-discord.md` in Developmnt).

1. `./run.sh discord`: it explains how to make the bot, asks for the bot token (hidden), the channel ID and your user ID, saves `~/.communication-bridge/discord.json` (mode 600, outside any Git checkout) and starts everything.
2. In the Processor, **Lessons → Talk in Discord → Teach TASS this**.

How it fits together:
- **bridge**: Arameair/Communication-Bridge, unchanged, in Manual mode, so it never answers by itself. It holds the token, accepts only your messages in that one channel, never double-posts, and pauses itself if a send might have half-happened. Its own page is at http://127.0.0.1:7820. It shares `~/.communication-bridge` with any other bridge you run, so the one-bot lock stops two bridges answering the same bot.
- **discord-door**: a small mailbox carrier. TASS writes `discord/outbox/<name>.txt`; the door posts it and writes the receipt to `discord/sent/<name>.json` (`delivered`, `failed` or `uncertain`). What you say lands in `discord/inbox/`. It enforces 1900 characters, 3 seconds between posts and 20 posts per 10 minutes.
- **TASS's code never sees a secret**: the kernel only reads and writes files in `shared/discord/`.
- The bridge repo is private, so `run.sh` fetches it with your `gh` login into `vendor/` (ignored by Git).

## Skills TASS earns, and his own folder

Some lessons teach a **skill**. When TASS has really run every step (from the audit log), the lesson in the Processor shows **✓ Learned** and a button **▶ Let TASS use it**:
- off (default): each use is a card in the Workshop that you approve;
- on: TASS uses it on his own. Every use is still in the audit log and the Workshop.

| Skill | Lesson | What TASS can do |
| --- | --- | --- |
| Talk in Discord | `lessons/claude/01-talk-in-discord.md` | `SAY IN DISCORD: <words>` |
| Improve yourself in your own folder | `lessons/claude/04-your-own-folder.md` | `CHANGE FILE: tass/<path>`: a pull request on **Arameair/Developmnt** `tass/`, which you merge. That is the only folder he may change (`TASS_OWN_FOLDER`). |

In his folder TASS keeps an improvement log and writes **"Teach me" requests** (`tass/requests/`). ChatGPT, the team member, answers each with a lesson in `lessons/chatgpt/`, and TASS learns it with you.

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
