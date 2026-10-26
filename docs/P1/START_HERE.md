# START HERE — P1 Feature Foundation on a clean WSL / Linux computer

**What you get:** Identity + Thought Processor + one clearly labelled **trusted local test feature** (a deterministic Python worker — *not* Fast AI Worker, *not* an AI). You establish the original Admin Identity, install the feature (it gets its own durable installation identity with zero grants), run it, approve one narrow capability, watch the feature enforce it, revoke it, and restart everything.

**Experimental branches — not Alpha.** Nothing here changes `alpha`, `stable` or `main`.

| Repo | Branch |
| --- | --- |
| `Arameair/identity` | `experiment/p1-feature-foundation-claude-20261026` |
| `Arameair/Thought-Processor` | `experiment/p1-feature-foundation-claude-20261026` |

**Time:** about 5 minutes to install and build, 3–4 minutes for the automated tests, 15 minutes for the hands-on walkthrough in [TEST_P1.md](TEST_P1.md).

## 1. Prerequisites (once)

Ubuntu 24.04 / Debian 13 or newer (WSL or not):

```bash
sudo apt update && sudo apt install -y git golang-go python3
go version        # must say go1.22 or newer
python3 --version # 3.8 or newer
```

If `go version` is older than 1.22 (e.g. Ubuntu 22.04), install Go from https://go.dev/dl/ instead and open a new terminal.

## 2. Get and build (any terminal)

```bash
mkdir -p ~/p1 && cd ~/p1
git clone --branch experiment/p1-feature-foundation-claude-20261026 https://github.com/Arameair/identity.git
git clone --branch experiment/p1-feature-foundation-claude-20261026 https://github.com/Arameair/Thought-Processor.git
(cd identity && go build -ldflags "-X main.buildBranch=experiment/p1-feature-foundation-claude-20261026" -o ~/p1/identity-p1 .)
(cd Thought-Processor/processor && go build -ldflags "-X main.buildBranch=experiment/p1-feature-foundation-claude-20261026" -o ~/p1/thought-processor-p1 .)
git -C identity log -1 --format='identity  %h %s'
git -C Thought-Processor log -1 --format='processor %h %s'
```

Both repositories are public; no token is needed. There are no third-party Go modules, so nothing else is downloaded.

## 3. Automated tests (optional but recommended, ~4 minutes)

```bash
cd ~/p1
(cd identity && go test ./...)                              # every line: ok
(cd Thought-Processor/processor && go test ./...)           # every line: ok (supervisor tests start real python3 processes)
python3 -I Thought-Processor/processor/acceptance/p1_integration.py --identity-src ~/p1/identity --work ~/p1/run-integration
```

The last command starts real Identity, Processor and feature processes on ports **7941 / 7961 / 8941** (GitHub is a labelled TEST FIXTURE) and must end with `RESULT: PASS (74 checks, 0 failed)`. Its report is `~/p1/run-integration/p1-integration-report.txt` — it contains no secrets.

Optional browser walkthrough with screenshots (needs `pip install playwright && playwright install chromium`):

```bash
python3 -I Thought-Processor/processor/acceptance/p1_ui_tour.py --identity-src ~/p1/identity --work ~/p1/run-ui --shots ~/p1/shots
```

## 4. Run it yourself (two terminals + a browser)

Use **fresh, disposable profiles** for this trial. (Recovery is deferred by the owner: if you forget the Admin passphrase, the test profile's Admin can't be recovered — just delete `~/p1/profiles` and start over.)

**Terminal 1 — Identity** (leave it open; it shows pairing codes and approval links):

```bash
~/p1/identity-p1 -data ~/p1/profiles/identity
```

**Terminal 2 — Thought Processor:**

```bash
~/p1/thought-processor-p1 -data ~/p1/profiles/processor -trust-feature-dir ~/p1/Thought-Processor/processor/testfeatures
```

It prints an `open :` link. On WSL, if no browser opens, **copy that link into your Windows browser** (WSL forwards `127.0.0.1` to Windows by default). The link is private — don't share it.

Then follow **[TEST_P1.md](TEST_P1.md)**. In short:

1. **Connect** Processor to Identity (type the code shown in terminal 1).
2. **Admin**: Sign in to GitHub → *Establish Admin Identity…* → *Open Identity approval page* → choose a passphrase (12+ characters), tick the confirmation, *Establish Admin Identity*. The passphrase is typed only in Identity's page.
3. **+ Identity**: add `Tass` (companion).
4. **+ Feature** → *Install…* → *Ask the Admin to approve* → *Open Identity approval page* → passphrase → *Approve*.
5. Open the feature → **Runtime**: *Start*, *Run standalone self-test*.
6. **Connections**: *Send synthetic request* (denied), *Ask the Admin…* (approve in Identity), send again (completed), *Revoke*, send again (denied).
7. Stop Identity (Ctrl+C in terminal 1) → **AUTHORITY UNAVAILABLE**; start it again → it reconnects; same IDs.

**No real GitHub account?** Use the labelled fixture instead of step 2's real sign-in (terminal 3, before starting Identity):

```bash
python3 -I ~/p1/identity/scripts/fixture_github_engine.py 8790 ~/p1/identity/README.md linux-amd64
IDENTITY_GITHUB_CLIENT_ID=IvFIXTURE00000001 ~/p1/identity-p1 -data ~/p1/profiles/identity-fixture \
  -test-github-web http://127.0.0.1:8790 -test-github-api http://127.0.0.1:8790
```

Everything then says **TEST FIXTURE** (the Admin is bound to fixture account `Arameair`, id 1001).

## 5. Stop and clean up

*Exit* in the page (stops Processor and every feature process it started), then Ctrl+C in terminal 1. Delete `~/p1/profiles` to start from scratch.

## Never share

Pairing codes, GitHub device codes, anything after `#code=`, `#key=` or `/approve#`, the Admin passphrase, `identity-pairing.json`, `credential.json` files. The reports and logs produced by the tools contain none of these.
