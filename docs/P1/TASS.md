# Developer Platform + TASS, your private companion (2026-10-29)

**Branches:**
- `Arameair/Thought-Processor` · `experiment/tass-companion-claude-20261029`. Based on PR #29 (the Developer entrance).
- `Arameair/TASS` · `experiment/tass-companion-feature-claude-20261029`.

The exact commits are in the delivery report and at the top of each draft PR.

- **Identity:** unchanged at `64940b6`. No Identity code or policy changed.
- **Not merged anywhere.** Your older TASS branches (Discord, local screen, ChatGPT) are untouched, as are `main`, `alpha`, PR #27/#28/#29 and every other trial.

## The Developer Platform

Once Processor is connected, it opens on **Developer Platform** (first item on the left). It is your home screen for the whole system:

- **System strip:** Identity (connected?), Admin (established?), Thought Processor (version, pid), and services running.
- **Next:** while you're setting up, the one thing to do now.
- **Services:** every installed service with its state and evidence chips (RUNNING, CONNECTED, READY, and MODEL for TASS), who may use it, its last self-test, and **Start / Stop / Self-test / Open →**. **Start all** and **Stop all** act on every service.
- **Ready to install:** reviewed, pinned packages you haven't installed. **Install…** still needs the Admin's approval in Identity.
- **Recent requests:** across all services. It shows outcomes only (COMPLETED, DENIED…), never what was said.

It can only do what this paired Processor session could already do. Installs and permissions still go through the Admin, so **no Identity change was needed**.

## What TASS is now

TASS is a feature you install, start, watch and stop from Thought Processor, like the worker, Thought and the Developer entrance.

| | |
|---|---|
| **Who may talk to it** | Only an identity the Admin granted `tass.chat.send` to, checked live by Identity on **every** message. History (`tass.chat.history`) and forgetting (`tass.chat.clear`) are separate permissions. |
| **Where the conversation lives** | One private file per identity (`0600`) in TASS's own data folder. Processor passes each message through and keeps only the outcome. Identity sees only a SHA-256 fingerprint of each message. Logs record who and when, never the words. |
| **Who answers** | A **local model on this computer** (Ollama or any OpenAI-compatible server on `127.0.0.1`), if you set one up. Until then TASS says plainly *"no local model yet — your message was saved, nothing was answered"*. It never makes up an answer. |
| **What it can't do** | Run programs, read files, browse, see your screen or change settings. It has **no tools**: a model's words are only words, and you decide what to do. |
| **What it refuses** | Any model address that isn't this computer (a hostname, another PC, `https` to anywhere, even `localhost`). It ignores proxy settings and never follows a redirect, so nothing can bounce a conversation to the internet. |

**TASS's pulse (its birth and its thinking):**
- **One cheap tick every 5 s.** The first three ticks write TASS's first thoughts: **I am · I am · What am I?** These are fixed text from you, labelled as seeds. After the third tick, TASS is **born**: its birth time, its installation, and its first question, "What am I?". It is born once and stays born across restarts.
- **After birth, at most once a minute.** TASS wakes the local model only when the facts about TASS changed, for example a model was added. The model thinks about "What am I?" from those true facts alone, and its **thinking trace** is saved next to the answer, labelled *the model's reasoning — not a fact*.
- **Every other tick is skipped.** A skip takes well under a millisecond: no model, no disk, no network. With no model, the tick records that it *would* think, and thinks nothing. Missed ticks are never replayed.
- **The pulse never reads anyone's conversation.**
- **Viewing it:**
  - The **PULSE** chip shows its stage and tick counts.
  - **Reading its first thoughts and reflections** needs its own permission, `tass.self.read`.

**The screen** is TASS's **Overview** tab:
- the chips **RUNNING · CONNECTED · READY · MODEL**;
- who is allowed;
- a **Talk to TASS** panel: your conversation, a message box and **Send**. Enter sends; Shift+Enter makes a new line.

## Results (Linux, automated)

| Suite | What it proves |
|---|---|
| `Arameair/TASS` `tests/test_tass_feature.py` | 39 tests: the pulse and genesis, conversations, isolation, the loopback-only guard, redirects, proxies, timeouts, forgetting, the protocol |
| `processor/acceptance/p1_tass.py --shots DIR` | 64 checks with real Identity, Processor, TASS, Thought and the worker, and a real browser (TASS chat and the Developer Platform) |
| Every earlier suite | nothing that worked before broke |

What `p1_tass.py` checks:
- **Trust:** pinned package; a tampered copy is BLOCKED; Identity refuses an impostor.
- **Install:** Admin deny, then approve.
- **Denials:** no permission, a direct call with no ticket, and another program all get DENIED.
- **No model:** an honest answer.
- **With a local test stand-in model on 127.0.0.1:**
  - the answer comes back with the model's name;
  - the model received the earlier conversation;
  - a non-local address is REFUSED;
  - a redirect is refused;
  - **every open TASS connection goes to 127.0.0.1 (OS check)**.
- **Isolation:** each identity has its own conversation.
- **Privacy:** the words are **not** in Processor's state, logs or activity, and **not** anywhere in Identity.
- **Lifecycle:**
  - revoke, then DENIED;
  - Identity down, then fails closed and turns NOT CONNECTED;
  - Stop, then process and port gone;
  - after restarting everything, **TASS still has the conversation**;
  - "forget everything" deletes it.

The stand-in model is a test fixture that answers with fixed text. It proves the path, not any AI.

**Owner acceptance on Windows/WSL: NOT RUN** until you do the steps below.

## Owner walkthrough (Windows + WSL)

**What this keeps safe:**
- New folders `~/p1c/tp-tass-clone`, `~/p1c/tass-companion` and `~/p1c/p1-tass`, on ports **7797/7867**.
- Your other trials (`p1-launcher`, `p1-worker-runtime`, `p1-developer-entrance`, their clones and profiles) are **never touched**.
- Fresh profiles: you'll pair once, sign in to GitHub and establish an Admin for this trial.

**Never share:** pairing codes, the GitHub device code, anything after `#code=`, `#key=` or `/approve#`, your passphrase, `credential.json`, or `*-private.log`. **To report a problem,** copy only the `ERROR:` line, red result text, or **Copy activity**.

**Use test words** (e.g. "hello TASS") until the Identity vault exists. Until then the conversation file is private to your user but not encrypted.

One command at a time, in a WSL terminal.

### 1. Check the layout (reads only)

```bash
git -C ~/p1c/identity rev-parse HEAD; ls -d ~/p1c/p1-tass ~/p1c/tp-tass-clone ~/p1c/tass-companion 2>&1 | tail -3
```

**Expected:**
- `64940b629a5ef52eaa143b2402726ec81d382002`;
- three lines ending in `No such file or directory`.

### 2. Get the code (new folders)

```bash
git clone --branch experiment/tass-companion-claude-20261029 https://github.com/Arameair/Thought-Processor.git ~/p1c/tp-tass-clone && git clone --branch experiment/tass-companion-feature-claude-20261029 https://github.com/Arameair/TASS.git ~/p1c/tass-companion && git -C ~/p1c/tp-tass-clone rev-parse HEAD && git -C ~/p1c/tass-companion rev-parse HEAD
```

TASS is private, so Git may ask you to sign in.

**Expected:** the two commit SHAs from the delivery report.

### 3. Preflight (checks only; builds and starts nothing)

```bash
cd ~/p1c/tp-tass-clone && P1_TASS_SRC=~/p1c/tass-companion P1_PREFLIGHT_ONLY=1 bash ./run-p1.sh
```

**Expected:**
- `Identity P1 source: 64940b6…`.
- `Developer Utility package: matches pinned artifact 9d33ed8bfccf…`.
- `TASS feature source: … — package matches pinned artifact 86d79ba30ab3…`.
- A Thought line. Either form is fine.
- `Trial folder: …/p1c/p1-tass (no profiles yet (fresh trial)). Ports 7797/7867 are free.`
- `PREFLIGHT OK — nothing was built or started.`

### 4. Start the trial

```bash
P1_TASS_SRC=~/p1c/tass-companion bash ./run-p1.sh
```

**Expected:** `P1 Processor ready on 127.0.0.1:7867`, and the browser opens. Leave the terminal open. After 5.1 the page lands on the **Developer Platform**.

### 5. In the browser

| # | Do | Expected (PASS only if you see it) |
|---|---|---|
| 5.1 | **Connect**, then type the code from the terminal | *Identity: connected* |
| 5.2 | **Sign in to GitHub** → **Establish Admin Identity…** → in Identity's page set a passphrase, tick, establish | *Admin: established* |
| 5.3 | **+ Identity**: `Dustin`, kind *person* | Dustin on the left |
| 5.4 | **Add a feature…** | card **TASS (private companion)**, badge **pinned artifact**, **Install…** |
| 5.5 | **Install…** → **Install** → approval page → passphrase → **Approve** | *TASS (private companion)* under Installed features |
| 5.6 | Open it. **Overview** → **Start** | **Running**; chips **RUNNING · CONNECTED · READY · MODEL: NOT CONFIGURED**; *Access: not allowed for anyone yet*; under **Talk to TASS**: *nobody may talk to it yet* |
| 5.7 | **Connections** → *Request a connection*: Dustin, `tass.chat.send` → **Ask the Admin…** → approve. Do the same for `tass.chat.history` | both **ACTIVE** in the grants table |
| 5.8 | **Overview** → **Talk to TASS**: type `hello TASS` → **Send** | your message appears, marked *saved · not answered (no local model)*, with *TASS has no local model yet — your message was saved, nothing was answered* |
| 5.9 | Second WSL terminal: `pgrep -af 'python[0-9.]* -I .*/p1-tass/processor-profile/installations/ins_[^ ]*/package/[t]ass_feature\.py$'` | exactly **one** line |
| 5.10 | **Connections**: **Revoke** Dustin's `tass.chat.send`. Then in **Try the protected capability**: Dustin, `tass.chat.send`, input `are you there?`, **Send request** | red **DENIED BY IDENTITY**: *The permission was revoked…*. On **Overview**, Talk to TASS says *Dustin may read the conversation but not send*, and **Send** is greyed out |
| 5.11 | **Overview** → **Stop** | **Stopped**; 5.9 prints nothing |
| 5.12 | **Start** again, then **Load conversation** | `hello TASS` is still there (TASS remembers across restarts); `are you there?` is not |
| 5.12a | About 15 s after **Start**: look at the chips | **PULSE: born**. Hover it: ticks, skipped, *waiting for a model* |
| 5.12b | **Connections**: ask the Admin for `tass.self.read` for Dustin and approve it. Then **Overview** → **Read TASS's pulse** | *TASS was born …* and its first thoughts **I am · I am · What am I?**, each marked *seed — written by Dustin*. With no model: *No reflections yet*. With a model (step 6): a reflection, with **Thinking** under a fold-out |
| 5.13 | Optional, for more services: **Developer Platform** → **Ready to install** → install **Thought** and **Developer Utility** (Admin approves each) | they appear under **Services** |
| 5.14 | **Developer Platform** → **Stop all** | every row says **Stopped** with a **Start** button; 5.9 prints nothing |
| 5.15 | **Start all** | every row **Running** with its chips (TASS: RUNNING · CONNECTED · READY · MODEL); 5.9 prints one line |
| 5.16 | Look at **Recent requests** | your earlier COMPLETED and DENIED requests, with no message text anywhere |

### 6. Optional: give TASS a local model

**Your decision:** which model, and installing it. Nothing here does that for you.

The suggested default is **`qwen3:1.7b`**. It is about 1.2 GB, runs on a processor alone in about 4 GB of RAM, and has a thinking mode, so the pulse gets real reasoning traces. On your RTX 2070 it runs entirely on the graphics card. Step up to `qwen3.5:4b` if its answers feel too thin. Run the model **inside WSL**, so `127.0.0.1` is the same computer TASS sees.

1. Install Ollama in WSL (from ollama.com), then `ollama pull qwen3:1.7b`.
2. Find TASS's data folder: it is the **Data folder** row on TASS's Overview.
3. Tell TASS about the model. Replace `DATA` with that folder:
   ```bash
   printf '{"kind":"ollama","url":"http://127.0.0.1:11434","model":"qwen3:1.7b"}\n' > DATA/model.json
   ```
4. Within about 10 s the chip becomes **MODEL: REACHABLE**. Send a message, and TASS answers. The answer is labelled with the model's name.
5. **Privacy check.** While it's answering, run this in the second terminal:
   ```bash
   ss -tnp | grep "pid=$(pgrep -f 'python[0-9.]* -I .*/p1-tass/processor-profile/installations/ins_[^ ]*/package/[t]ass_feature\.py$'),"
   ```
   Every address shown is `127.0.0.1`.

A model answer can take a while on a CPU. TASS gives up after 25 s and saves nothing. Then try a smaller model.

### 7. Finish

Press **Ctrl+C** in the trial terminal. The prompt returns, and 5.9 prints nothing.

### Report

Paste back:
- the SHAs from steps 1–2;
- the step 3 output;
- **PASS / FAIL / NOT RUN** for 5.1–5.16 (including 5.12a/b) (and step 6 if you did it);
- any `ERROR:` or red text.

## Rollback (keeps every other installation)

With the trial stopped:

```bash
rm -rf ~/p1c/p1-tass ~/p1c/tp-tass-clone ~/p1c/tass-companion
```

## Known limits (honest)

- **Not encrypted yet.** The conversation file is `0600` (private to your user), but any program running as you could read it. The Identity vault (Option C) is what fixes this. Use test words until then.
- **Same-user isolation only** (as for every P1 service). Loopback HTTP is signed but not encrypted.
- **No model chosen yet.** TASS works honestly without one; the model is your decision.
- **Memory is the conversation only** (the last 200 messages; the model sees the last 12). Connecting it to Thought's Spark → Trace → Thought is the next step, and it needs a Processor-routed feature-to-feature permission.
- **Windows-side models:** a model running in Windows (not WSL) is reachable at `127.0.0.1` only with WSL's mirrored networking. Keep the model in WSL.
