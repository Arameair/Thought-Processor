# AGENTS.md: run the Thought loop on Windows (Chat-Jimmy → Thought Engine → Thought Processor)

This runbook is for people and AI operators (for example a Grok bot). Follow it exactly, and report every step as **PASS / FAIL / NOT RUN** with what you saw.

**Goal (Thought-engine#2):**
1. Run three programs.
2. In Processor, type **"Player entered Goldshire"** three times.
3. See **one Thought with a question**.
4. Restart everything and see that the Thought is still there.

| # | Program | Repo · branch | Port | Data |
| --- | --- | --- | --- | --- |
| 1 | Chat-Jimmy (the AI worker) | `Arameair/chat-jimmy` · `experiment/fast-clean-worker-20261023` | 7791 | `%USERPROFILE%\.chat-jimmy` |
| 2 | Thought Engine | `Arameair/Thought-engine` · `experiment/engine-processor-loop-20261026` | 7798 | `%USERPROFILE%\.thought-engine\experiment_engine-processor-loop-20261026` |
| 3 | Thought Processor | `Arameair/Thought-Processor` · `alpha-candidate-20261026` | 7860 | `%USERPROFILE%\.thought-processor\default` |

Identity isn't needed for this loop.

## Rules: read these first

1. **Never paste** into chat, issues or reports:
   - the Engine's **attach code**
   - Processor's `#code=` link or operator code
   - the contents of `engine-link.json` or `attachments.json`

   Type the attach code only into Processor's page.
2. Each program runs in **its own** PowerShell window. Stop programs with **Ctrl+C** in their own window. Don't kill processes you didn't start.
3. Don't edit any branch. Report problems; don't "fix" code unless asked.
4. **Running Chat-Jimmy live is the owner's decision.** It drives chatjimmy.ai in a background Chrome or Edge, and Chat-Jimmy's README notes the site's terms. If the owner hasn't said yes, stop after Step 1's checks and report.
5. Never report a step as passed unless you saw the expected result.

## Step 0: prerequisites

```powershell
git --version
go version        # go1.22 or newer
```
Chat-Jimmy also needs **Google Chrome or Microsoft Edge** installed.

## Step 1: Chat-Jimmy, live (window 1)

```powershell
cd $HOME
git clone --branch experiment/fast-clean-worker-20261023 https://github.com/Arameair/chat-jimmy.git
cd chat-jimmy
go build -mod=vendor -o chat-jimmy.exe .
.\chat-jimmy.exe -serve
```
Notes:
- If `git clone` asks you to sign in, the repo is private: sign in as the owner.
- Do **not** add `-mock`. The Engine refuses mock workers, and observations would just wait.
- If the chat page never becomes ready in the background, stop it with Ctrl+C and run `.\chat-jimmy.exe -serve -show`. This uses a visible browser window, so you can see a challenge or limit page.

**Check from another window:**
```powershell
Invoke-RestMethod http://127.0.0.1:7791/api/heartbeat
```
**Expected:** `ok = True`, `service = chat-jimmy`, `mode = live`, `state = ready`. It can take a few seconds after start.

Chat-Jimmy also writes `%USERPROFILE%\.chat-jimmy\worker.json`. That file is how the Engine finds it.

## Step 2: Thought Engine (window 2)

```powershell
cd $HOME
git clone --branch experiment/engine-processor-loop-20261026 https://github.com/Arameair/Thought-engine.git
cd Thought-engine
go build -ldflags "-X main.buildBranch=experiment/engine-processor-loop-20261026" -o thought-engine.exe .
.\thought-engine.exe
```
**Expected console lines:**
- `open    : http://127.0.0.1:7798/`
- `attach  : no program is attached yet. Attach code XXXX-XXXX (single use, until HH:MM) …` This line is private (rule 1).
- A `start` line with branch `experiment/engine-processor-loop-20261026` and `0 waiting event(s)`.

**Check:**
```powershell
Invoke-RestMethod http://127.0.0.1:7798/api/health
```
**Expected:** `app = thought-engine`, `protocol = thought-engine.processor/1-experimental`.

Open `http://127.0.0.1:7798/`. The top bar should read **LIVE · AI WORKER: Chat-Jimmy READY**.
- If it says **NOT CONNECTED** / **NOT RUNNING**, Step 1 isn't ready. The Engine still works, and observations will **wait**.
- If the code expired (10 minutes), press **Make an attach code** in the Engine page's **Programs attached** panel.

## Step 3: Thought Processor (window 3)

```powershell
cd $HOME
git clone --branch alpha-candidate-20261026 https://github.com/Arameair/Thought-Processor.git
cd Thought-Processor\processor
go build -o thought-processor.exe .
.\thought-processor.exe
```
**Expected:** `Thought Processor 0.1.0-experiment (<sha>)`, plus a line `open    : http://127.0.0.1:7860/#code=…`. That link is private: open it locally.

## Step 4: the loop, in Processor's page

1. **Next step** says *Tell Processor where the Thought Engine runs*. In the **Thought Engine** card, type `127.0.0.1:7798` → **Save**.
   **Expected:** the Thought Engine card shows **Running — says it is Thought Engine**, and the top card shows **Running · not attached**.
2. In the top card, type the attach code from window 2 → **Attach**.
   **Expected:** pill **Attached**, with the Engine version and branch. The AI worker shows **READY**, or the honest reason it isn't.
3. **Create on the Engine** (name `TASS`).
   **Expected:** the identity is chosen, and the button reads **Thought Processing: OFF — switch on**. Press it.
   **Expected:** **ON**.
4. Type `Player entered Goldshire` → **Send observation**. Do this **three times**.
   **Expected:** each observation goes **Queued / Processing → Done** ("rated N/10, topic …").
   If Chat-Jimmy isn't ready, each one shows **Waiting for AI**, with the reason, and stays saved. That is correct, not a failure.
5. **Expected:** **one Thought** under *Thoughts*: a question written by **Chat-Jimmy**, about 3 traces.
   With a live 8B model the topic names may differ slightly between observations. If they don't match, the observations become separate Traces and no Thought forms yet. Report exactly what each observation says ("topic …"), and add more Goldshire observations if needed.
6. **Open the Engine's own page** (button in the top card).
   **Expected:** the **Programs attached** panel lists Thought Processor, and *Chat-Jimmy calls* shows each question and reply.

## Step 5: restart check

1. Press **Ctrl+C** in windows 3, 2 and 1.
2. Start them again with the same commands: `.\chat-jimmy.exe -serve`, `.\thought-engine.exe`, `.\thought-processor.exe`.
3. Open Processor's **new** `#code=` link.

**Expected:**
- The Engine prints **no** attach code, because Processor is still attached.
- Processor shows **Attached** without a new code.
- The **same Thought** and the three **Done** observations are still there.

## Step 6: report

- In Processor: **Diagnostics → Copy context**.
- In the Engine page: **Copy Diagnostics**.

Paste both with your PASS/FAIL/NOT RUN list. Both leave out codes, keys, observation text and questions. Glance over them anyway.

## Without Chat-Jimmy: the TEST FIXTURE rehearsal (optional, labelled)

Use this only to check the wiring when Chat-Jimmy can't run. It is **not AI**.

Window 1:
```powershell
cd $HOME\Thought-engine
go run ./cmd/chat-jimmy-test-fixture -port 7795
```
Window 2 (instead of Step 2's start command):
```powershell
.\thought-engine.exe -test-fixture-worker -chat-jimmy http://127.0.0.1:7795
```
What to expect:
- Everything shows **TEST FIXTURE**: the Engine's striped top bar, Processor's striped banner, and every question starting `[TEST FIXTURE]`.
- The Engine keeps this data in its own folder (`…-test-fixture`), so it never mixes with live Thoughts.
- Report results from this mode as **TEST FIXTURE**, never as live.

## Troubleshooting

| You see | Meaning / fix |
| --- | --- |
| Processor: **Thought Engine is not running** | Window 2 is closed or still building. Start it; the card recovers by itself. |
| Processor: **No contract** / Engine `/api/health` 404 | Wrong Engine branch (e.g. the older v1). Use `experiment/engine-processor-loop-20261026`. |
| **wrong attach code** | Typo, or a newer code replaced it. After 5 wrong tries the code is void: make a new one on the Engine page. |
| **no attach code is waiting** | The code expired or was used. Press **Make an attach code** on the Engine page. |
| **The Engine no longer accepts this attachment** | It was detached on the Engine page, or the Engine runs with a different data folder. Attach again. |
| Observations stay **Waiting for AI** | Chat-Jimmy isn't live and ready. Check `Invoke-RestMethod http://127.0.0.1:7791/api/heartbeat`. **MOCK MODE — NOT USED** means it was started with `-mock`. |
| Observation **Failed** | Chat-Jimmy's reply didn't fit the format twice. Nothing was made up. Use **Retry** on the Engine page and report the reply under *Chat-Jimmy calls*. |
| `cannot listen` / `profile in use` | Another copy is running on that port or profile. Use it, or close it first. |

## What this does NOT do (don't try)

- Processor does not start or stop programs (no runner).
- Identity grants, pairing and installing the Engine are separate work.
- No releases, tags or merges.
