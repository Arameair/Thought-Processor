# TASS workbench: talk in the Development Workstation, watch the Pipeline, press Update

**Branch:** `experiment/tass-workbench-claude-20261031`, from `alpha` @ `e4b45a9`. `alpha` is untouched. Nothing is merged.

| Piece | Where |
| --- | --- |
| Talk to TASS | **Development Workstation** → TASS panel. Each reply names the model that wrote it. |
| See each message go through the system | **Pipeline** (chip on the Processor page, or **Pipeline ↗** in the workstation, which opens next to it): You → Thought Processor → Identity → TASS → Model → Reply, with the model, Identity's decision, and the outcome and timing. |
| Get the newest version | Workbench menu **8 Update to the newest version** (or `secure_launch.py update`). |

It uses **your existing alpha profile** (`~/p1c/p1-tass-alpha`) and alpha's pins: Identity `a582f8e`, TASS 0.2.0 `543611f8…`, and Thought `772200b`. Your Admin, TASS, grants and conversation carry over, so you don't set anything up again. Also included: the fix for the "missing or wrong anti-forgery token" error (#33).

## Switch to it (once)

In Ubuntu (WSL):

```bash
L=~/p1c/alpha/Thought-Processor/tools/p0-launcher/secure_launch.py
python3 $L stop
cp -a ~/p1c/p1-tass-alpha ~/p1c/p1-tass-alpha.backup-20261031      # safety copy, while stopped
cd ~/p1c/alpha/Thought-Processor
git fetch origin experiment/tass-workbench-claude-20261031
git checkout -b experiment/tass-workbench-claude-20261031 --track origin/experiment/tass-workbench-claude-20261031
python3 $L start
python3 $L open
```

Or double-click `Start-Workbench.cmd`, then choose **1** (Start) and **3** (Open).

## Before you start: is your model reachable from Ubuntu?

TASS runs inside Ubuntu (WSL), so the model has to answer **there**:

```bash
curl -s http://127.0.0.1:8080/v1/models
```

- **You see JSON with an `"id"`:** good. Note that id; it's the model name TASS needs.
- **"Connection refused":** start the server in Ubuntu:
  ```bash
  llama-server -hf ibm-research/granite-vision-3.2-2b-GGUF:Q4_K_M --host 127.0.0.1 --port 8080 -c 16384
  ```
  `-c 16384` gives room for a whole project file. Without it, questions about longer files fail with "HTTP 400".
- **It works in Windows but not in Ubuntu:** your server is running on the Windows side, and WSL can't reach it at 127.0.0.1. Run it in Ubuntu instead.

## Your acceptance test (#31)

**Start → Open → Sign in → Development Workstation → Select repository → Open file → Ask TASS → real Granite response.**

1. `python3 $L start`, then `python3 $L open`. Edge opens the Development Workstation already signed in. The chips should read **Identity** and **GitHub: verified**.
2. In the TASS panel, follow its steps if any appear: allow chat, allow model. You approve each one in Identity's page.
3. At **Choose TASS's local model**: choose **llama.cpp / OpenAI-compatible**, address `http://127.0.0.1:8080`, model = the `"id"` from curl, then press **Use this model**. The chip should say **MODEL REACHABLE**.
4. On the left, pick a repository (for example **TASS**), open a folder, and click a file (for example `README.md`).
5. Leave **Show TASS the file I'm viewing** ticked. Ask a question only the file can answer, for example *"What are the folders listed in this file, and what is each one for?"*
6. **Pass:** the answer uses facts from that file and is labelled with your model (`… (openai-compatible at http://127.0.0.1:8080)`). Click **Pipeline ↗** to see the same message go through Identity → TASS → model.
7. **Refusal check:** a file outside the allowed repositories can't be opened (the workstation only lists the launcher's repositories, and paths such as `..` or `.git` are refused).

If something fails, the TASS panel now says what to do next, for example the exact llama-server command.

## Which profile?

**Your existing alpha profile is safe to use, with the backup step above.** This branch uses the same pins and the same profile format as alpha. It adds two things: a Pipeline page that only reads, and `launcher-control/update-history.jsonl` (a 0600 file holding version numbers only). It never resets or rewrites identities, grants, the model setting or the conversation.

If you'd rather try it isolated first, add `P1_LAUNCH_WORK=~/p1c/p1-tass-workbench-test` in front of each launcher command. That is a fresh profile, so you set up the Admin and TASS once more there.

## Use it

1. In the Development Workstation's TASS panel, untick **Show TASS the file I'm viewing**, type a message and press **Ask TASS**.
2. Click **Pipeline ↗**. Your message shows up as a row: the model that answered, Identity's decision, TASS, and the reply. The page refreshes every 10 s.
   - Seeing the words needs the `tass.chat.history` permission for you. **Get TASS ready for this person** asks for it.
   - Without that permission, the steps still show and the words say why they're hidden.

## Update

Press **8** in the Workbench menu (or `python3 $L update`). It:

- moves **only** the Thought-Processor checkout, forward only, on this branch;
- refuses if you changed files there, switched branch, or the branch history was rewritten;
- lists what changed, stops, checks the new version the same way Start does (exact Identity SHA, pinned TASS/Thought digests), then starts it;
- if any check or the start fails, puts back the exact previous version and starts that.

Your Identity, TASS and Thought checkouts, profile, identities, grants and conversations are never touched. A version that would trust a different TASS package than the one you have is refused, not installed. After an update, press **3** (Open). Every Open makes a fresh one-use sign-in ticket.

## Go back to alpha

```bash
python3 $L stop
cd ~/p1c/alpha/Thought-Processor && git checkout alpha
python3 $L start
```

## What was tested (cloud, Linux), on the commit that adds this section

| Check | Result |
| --- | --- |
| `go test ./...` | PASS |
| Launcher unit tests (incl. 8 Update tests on real temporary git repos) | 30/30 PASS |
| `acceptance/p0_workstation_tass.py`: real Identity + Processor + TASS 0.2.0; single-use ticket; open `processor/main.go`; TASS's answer proves the model saw the file (Ollama-shaped **and** llama.cpp-shaped servers); too-long file refused with the server's reason; files outside the repositories refused; revoked permission denied at once; one owner identity reused; Identity stopped → fail closed; no private link on disk | 33/33 PASS |
| `acceptance/p0_workstation.py`: launcher-style tickets, GitHub WAITING/DENIED/EXPIRED/VERIFIED gate, path refusals, allowlisted tests, restart, fail closed | 31/31 PASS |
| `acceptance/p1_tass.py`: install, grants, wrong person denied, revoke, Identity outage, restart keeps identity and conversation, Pipeline in a real browser | 81/81 PASS |
| Real launcher with a disposable profile: Start, Status, Open, no private link saved, real Update `e134c76` → `4b3f60c`, restart, Open again, UP_TO_DATE | PASS |
| **On your PC:** real Granite on llama.cpp, real GitHub, Windows Edge, your profile | **NOT RUN.** That's your acceptance test above. |

In all cloud runs, GitHub and the model are labelled test stand-ins. They prove the path, not the AI.
