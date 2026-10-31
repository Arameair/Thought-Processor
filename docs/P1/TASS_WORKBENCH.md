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

## What was tested (cloud, Linux)

| Check | Result |
| --- | --- |
| `go test ./...` (adds `TestPipelineRequestsEndpoint`, `TestSessionCSRFRecovery`) | PASS |
| Launcher unit tests (8 new Update tests on real temporary git repos: fast-forward, up to date, failed pre-flight → rollback, failed start → rollback, local changes refused, rewritten history refused, other branch refused, private history file) | 30/30 PASS |
| `processor/acceptance/p1_tass.py`: real Identity + Processor + TASS 0.2.0, with a real browser, a labelled test GitHub and a labelled test-stand-in model; includes 3 Pipeline checks | 81/81 PASS |
| Real launcher on this branch with a disposable profile: Start, Status, Open (single-use ticket), no private link saved; then a real **Update** from GitHub, restart and Open again | see the commit that adds this file |
| On your PC: your real Granite model, real GitHub, Windows Edge | **NOT RUN.** That's your acceptance run. |
