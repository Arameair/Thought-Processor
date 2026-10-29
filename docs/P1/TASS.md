# Developer Platform + TASS, your private companion (2026-10-29)

**Branches:**
- `Arameair/Thought-Processor` · `experiment/tass-companion-claude-20261029`. Based on PR #29 (the Developer entrance).
- `Arameair/TASS` · `experiment/tass-companion-feature-claude-20261029`.

The exact commits are in the delivery report and at the top of each draft PR.

- **Identity:** unchanged at `64940b6`. No Identity code or policy changed.
- **Not merged anywhere.** Your older TASS branches (Discord, local screen, ChatGPT) are untouched, as are `main`, `alpha`, PR #27/#28/#29 and every other trial.

## How "putting TASS on your computer" works

| You think of it as | What happens |
|---|---|
| **Open the development workbench** | `run-p1.sh` starts Identity and Thought Processor. The browser lands on the **Developer Platform**, which has a **TASS** box where you install it, run it, get it ready for you, choose its model and talk to it. |
| **Sign in as TASS** | You **install TASS** from the Developer Platform. You approve it as Admin in Identity's page, and **Identity gives TASS its own identity** (an installation ID that is never reused). From then on TASS *is* that identity on your computer. |
| **Let me talk to it / see it think / pick its model** | You (the person *Dustin*) get one permission for each thing, each approved by the Admin: `tass.chat.send`, `tass.chat.history`, `tass.self.read`, `tass.model.set`. |
| **The model** | **Ollama**, running in WSL on your computer, with **`qwen3.5:4b`**. That model reads pictures and text, and it can think. TASS only accepts a model at `127.0.0.1`. |

## The Developer Platform

Once Processor is connected, it opens on the **Developer Platform** (first item on the left). It is the home screen for the whole system:

- **System strip:** Identity, Admin, Processor and services.
- **Next:** one step while you're setting up.
- **Services:** every service with its evidence chips (RUNNING, CONNECTED, READY; for TASS also MODEL and PULSE), who may use it, its last self-test, and **Start / Stop / Self-test / Open →**, plus **Start all / Stop all**.
- **Ready to install:** reviewed, pinned packages. The Admin still approves each install.
- **Recent requests:** outcomes only (COMPLETED, DENIED…), never what was said.

It can only do what this paired Processor session could already do. Installs and permissions still go through the Admin.

## What TASS does

| | |
|---|---|
| **Talk** | Text, or **one picture** (PNG/JPEG/WebP, up to 4 MB) plus a question such as "what is this picture of?". **Every message is checked live by Identity.** |
| **Where things live** | One private conversation file per identity (`0600`). Pictures are stored privately next to it (`0600`), and *forget everything* deletes both. Processor and Identity keep **no copy** of your words or pictures; Identity sees only a SHA-256 fingerprint of each request. |
| **Who answers** | The local model you chose. With no model, TASS says so and invents nothing. If the model can't see pictures, TASS says that and saves nothing. |
| **Its pulse** | One cheap tick every 5 s. Its first three thoughts are **I am · I am · What am I?** (fixed seeds from you); then it is **born**, once. After that, at most once a minute and only when something about it changed, the model thinks about "What am I?" from true facts about TASS. The **thinking trace** is saved, labelled *the model's reasoning — not a fact*. |
| **What it can't do** | Run programs, read files, browse, see your screen, change settings, or reach anything outside your computer. Any model address that isn't `127.0.0.1` is refused, proxies are ignored, and redirects are refused. |

## Results (Linux, automated)

| Suite | Result |
|---|---|
| `Arameair/TASS` `tests/test_tass_feature.py` | **49** tests: pictures, model abilities, chat thinking, model choice, the pulse and genesis, conversations, isolation, the loopback guard, redirects, proxies, timeouts, forgetting, the protocol |
| `processor/acceptance/p1_tass.py --shots DIR` | **78** checks with real Identity, Processor, TASS, Thought, the worker and a real browser |
| Every earlier suite | still passing |

In the automated tests, the "model" is a **test stand-in** on `127.0.0.1` that answers the way Ollama's API does (the same fields: `images`, `think`, `message.thinking`, and `capabilities` from `/api/show`). It proves the path end to end. **A real Ollama model has not run in the test environment** (it has no internet). Your run below is the first one with a real model.

**Owner acceptance on Windows/WSL: NOT RUN** until you do the steps below.

---

## Start here (Windows + WSL)

**Kept safe:**
- New folders `~/p1c/tp-tass-clone`, `~/p1c/tass-companion` and `~/p1c/p1-tass`, on ports **7797/7867**.
- Your other trials are **never touched**.

**Never share:** pairing codes, the GitHub device code, anything after `#code=`, `#key=` or `/approve#`, your passphrase, `credential.json`, or `*-private.log`.

**Use test words and test pictures** until the Identity vault exists. Files are private to your user but not encrypted yet.

Run one command at a time in a WSL terminal.

### A. Get the model ready (once)

1. **Install Ollama in WSL.** This downloads software from ollama.com, so it's your decision.
   ```bash
   curl -fsSL https://ollama.com/install.sh | sh
   ```
2. **Check the version.** `qwen3.5` needs **0.17.4 or newer**.
   ```bash
   ollama --version
   ```
3. **Download the model** (about 3.4 GB; reads pictures and text, and can think):
   ```bash
   ollama pull qwen3.5:4b
   ```
   If this says it *could not connect*, open a **second** WSL terminal, run `ollama serve`, leave it open, and try again.
4. **Warm it up** so TASS's first answer isn't slow. This loads the model onto your graphics card.
   ```bash
   ollama run qwen3.5:4b "Say hello in five words." && ollama ps
   ```
   **Expected:** a short hello, then a line for `qwen3.5:4b`. `100% GPU` means your RTX 2070 is doing the work. If it shows CPU, it still works, just slower.

### B. Get the code

**First time:**
```bash
git clone --branch experiment/tass-companion-claude-20261029 https://github.com/Arameair/Thought-Processor.git ~/p1c/tp-tass-clone && git clone --branch experiment/tass-companion-feature-claude-20261029 https://github.com/Arameair/TASS.git ~/p1c/tass-companion
```
**If you cloned these earlier today:**
```bash
git -C ~/p1c/tp-tass-clone pull && git -C ~/p1c/tass-companion pull
```
**Check:**
```bash
git -C ~/p1c/identity rev-parse HEAD; git -C ~/p1c/tp-tass-clone rev-parse HEAD; git -C ~/p1c/tass-companion rev-parse HEAD
```
**Expected:** `64940b629a5ef52eaa143b2402726ec81d382002`, then the two SHAs from the delivery report.

### C. Preflight (checks only; builds and starts nothing)

```bash
cd ~/p1c/tp-tass-clone && P1_TASS_SRC=~/p1c/tass-companion P1_PREFLIGHT_ONLY=1 bash ./run-p1.sh
```

**Expected:**
- `TASS feature source: … — package matches pinned artifact 985027bbaebf…`
- `PREFLIGHT OK — nothing was built or started.`

If you ran an earlier TASS trial today, it may also say *existing profiles will be reused*. That's fine.

### D. Open the workbench

```bash
P1_TASS_SRC=~/p1c/tass-companion bash ./run-p1.sh
```

**Expected:** `P1 Processor ready on 127.0.0.1:7867`, and the browser opens. Leave this terminal open; Ctrl+C here stops everything at the end.

### E. In the browser: everything on the Developer Platform

| # | Do | Expected |
|---|---|---|
| 1 | **Connect**, then type the code shown in the terminal | *Identity: connected*, then the **Developer Platform** |
| 2 | **Sign in to GitHub** → **Establish Admin Identity…** → in Identity's page set a passphrase, tick, establish | *Admin: established*. Skip this if your earlier TASS trial already did it |
| 3 | **+ Identity** (left): `Dustin`, kind *person* | Dustin on the left |
| 4 | **"Sign in as TASS":** in the **TASS** box on the platform → **Install TASS…** → **Install** → approval page → passphrase → **Approve** | the TASS box shows **Stopped** and a **Run TASS** button. TASS now has its own identity |
| 5 | **Run TASS** | **Running**. Within about 15 s the chips read **RUNNING · CONNECTED · READY · MODEL: NOT CONFIGURED · PULSE: born** |
| 6 | *For Dustin* → **Get TASS ready for this person** → approve each request (**Approve…** → passphrase), four in all | **For Dustin: ✓ ready**. The **Talk to TASS** chat and **TASS's model** appear in the same box |
| 7 | **TASS's model** (it already says `qwen3.5:4b`) → **Use this model** | within about 10 s: **MODEL: REACHABLE · qwen3.5:4b · thinking · vision** |
| 8 | **Talk to TASS**: `hello TASS, what are you?` → **Send** | TASS answers, labelled with the model's name |
| 9 | **Attach picture** → choose a test photo → `what is this picture of?` → **Send** | your message shows 📎 *the file name*, and TASS describes the picture |
| 10 | Wait about a minute → **Read TASS's pulse** (further down the box) | **I am · I am · What am I?** (seeds), *Born …*, and a reflection with **Thinking** under a fold-out |
| 11 | Optional: tick *show thinking in chat* → **Use this model** → ask something | answers get a **Thinking** fold-out (slower) |
| 12 | **Privacy check:** in a second terminal, while TASS answers: `ss -tnp \| grep "pid=$(pgrep -f 'python[0-9.]* -I .*/p1-tass/processor-profile/installations/ins_[^ ]*/package/[t]ass_feature\.py$'),"` | every address is `127.0.0.1` |
| 13 | **Stop TASS**, then **Run TASS**, then **Load conversation** | it's the same TASS: same birth time, same conversation |

### F. Finish

Press **Ctrl+C** in the workbench terminal. Leave `ollama serve` running or stop it; your choice.

### If something goes wrong

| You see | Do this |
|---|---|
| **MODEL: UNREACHABLE** | Ollama isn't running: start `ollama serve` in a second WSL terminal |
| **MODEL: MODEL MISSING** | `ollama pull qwen3.5:4b` |
| *The local model took too long* (MODEL_TIMEOUT) | The first answer loads the model. Run the warm-up in A.4 and send again. Nothing was saved |
| *The chosen model can't see pictures* | You picked a text-only model. Use `qwen3.5:4b` |
| *ollama: qwen3.5 requires a newer version* | Rerun the install command in A.1 |
| An `ERROR:` line in the terminal | Copy just that line to me |

### Report

Paste back:
- the three SHAs;
- the C output;
- **PASS / FAIL** for E1–E13;
- any red text.

## Rollback (keeps every other installation)

With the workbench stopped:

```bash
rm -rf ~/p1c/p1-tass ~/p1c/tp-tass-clone ~/p1c/tass-companion
```

`ollama rm qwen3.5:4b` removes the model.

## Known limits (honest)

- **Not encrypted yet.** Files are `0600`, but any program running as you could read them. The Identity vault (Option C) fixes this.
- **Same-user isolation only.** Loopback HTTP is signed but not encrypted.
- **Ollama is outside the protected point.** It is a separate program any local program can use, and it can download models. Making it a pinned, managed service is the next security step.
- **Answers must finish within about 25 s.** On a GPU that's plenty. On a slow CPU, use a smaller model. Pulse thinking runs in the background and may take up to 2 minutes.
- **Memory is the conversation** (last 200 messages; the model sees the last 12). The model sees each picture only with the message it came with.
- **Windows-side models** are reachable from WSL only with mirrored networking. Keep the model in WSL.
