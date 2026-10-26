# START HERE — P1 Feature Foundation (cleanup build) on a clean WSL / Linux computer

**What you get:** Identity + Thought Processor + one clearly labelled **trusted local test feature** (a deterministic Python worker — *not* Fast AI Worker, *not* an AI). You establish the original Admin Identity, install the feature (it gets its own durable installation identity with zero grants), run it, approve one narrow capability, watch the feature enforce it, revoke it, and restart everything.

**Experimental branches — not Alpha.** Nothing here changes `alpha`, `stable`, `main` or the original P1 branches.

| Repo | Branch |
| --- | --- |
| `Arameair/identity` | `experiment/p1-cleanup-claude-20261026` |
| `Arameair/Thought-Processor` | `experiment/p1-cleanup-claude-20261026` |

What changed since the first P1 build (Grok's findings, details in [P1_CLEANUP_REPORT_20261026.md](P1_CLEANUP_REPORT_20261026.md)): only the **exact pinned** test-fixture artifact can be installed (a package's own "trusted" label counts for nothing); only a grant's own parties — or the Admin — can revoke it; Identity's approval page shows **who** is asking (from its pairing record) and warns when it isn't your main Thought Processor; grants expire after **1 hour** by default; tests never use or disturb ports 7790 / 7860 / 8790.

**Time:** about 5 minutes to install and build, 4–5 minutes for the automated tests, 15 minutes for the hands-on walkthrough in [TEST_P1.md](TEST_P1.md).

## 1. Prerequisites (once)

Ubuntu 24.04 / Debian 13 or newer (WSL or not):

```bash
sudo apt update && sudo apt install -y git golang-go python3 python3-venv
go version        # must say go1.22 or newer
python3 --version # 3.8 or newer
```

If `go version` is older than 1.22 (e.g. Ubuntu 22.04), install Go from https://go.dev/dl/ instead and open a new terminal.

## 2. Get and build (any terminal)

```bash
mkdir -p ~/p1c && cd ~/p1c
git clone --branch experiment/p1-cleanup-claude-20261026 https://github.com/Arameair/identity.git
git clone --branch experiment/p1-cleanup-claude-20261026 https://github.com/Arameair/Thought-Processor.git
(cd identity && go build -ldflags "-X main.buildBranch=experiment/p1-cleanup-claude-20261026" -o ~/p1c/identity-p1 .)
(cd Thought-Processor/processor && go build -ldflags "-X main.buildBranch=experiment/p1-cleanup-claude-20261026" -o ~/p1c/thought-processor-p1 .)
git -C identity log -1 --format='identity  %h %s'
git -C Thought-Processor log -1 --format='processor %h %s'
```

Both repositories are public; no token is needed. There are no third-party Go modules, so nothing else is downloaded.

## 3. Automated tests (optional but recommended, ~5 minutes)

```bash
cd ~/p1c
(cd identity && go test ./...)                              # every line: ok
(cd Thought-Processor/processor && go test ./...)           # every line: ok (supervisor tests start real python3 processes)
python3 -I Thought-Processor/processor/acceptance/p1_integration.py --identity-src ~/p1c/identity --work ~/p1c/run-integration
python3 -I Thought-Processor/processor/acceptance/p1_findings.py --identity-src ~/p1c/identity --processor-src ~/p1c/Thought-Processor --work ~/p1c/run-findings
```

- `p1_integration.py` starts real Identity, Processor and feature processes and must end with `RESULT: PASS (81 checks, 0 failed)`. Report: `~/p1c/run-integration/p1-integration-report.txt` (no secrets).
- `p1_findings.py` re-runs Grok's findings (M1–M3, L1, L3, L5) as attacks and must end with `secure behaviour observed: 15 of 15`.

**Ports.** The test scripts let the operating system pick free ports, and they tell their Processor not to look for an Identity on the default port. So they run fine — and touch nothing — while **your own** Identity (7790), Processor (7860) or GitHub fixture (8790) are running. Want proof? Start the decoys in another terminal first, run the tests, then Ctrl+C the decoys:

```bash
python3 -I ~/p1c/Thought-Processor/processor/acceptance/decoy_ports.py 7790 7860 8790   # skips any port already in use
```

**Browser walkthrough with screenshots (optional).** Ubuntu's Python refuses `pip install` system-wide (PEP 668), so use a private virtual environment:

```bash
python3 -m venv ~/p1c/venv
~/p1c/venv/bin/pip install playwright
~/p1c/venv/bin/playwright install --with-deps chromium     # asks for sudo once for browser libraries
~/p1c/venv/bin/python Thought-Processor/processor/acceptance/p1_ui_tour.py --identity-src ~/p1c/identity --work ~/p1c/run-ui --shots ~/p1c/shots
```

Ends with `RESULT: PASS`. Delete `~/p1c/venv` afterwards if you like; nothing else was installed.

**Cleanup after tests.** The scripts stop every process they started (also on failure). Their work folders (`~/p1c/run-*`) are private (0700) and can simply be deleted. If you ever interrupt a script with Ctrl+C twice and something stays behind, find only *its* processes with `pgrep -af ~/p1c/run-` and stop those — never anything else.

## 4. Run it yourself (two terminals + a browser)

Use **fresh, disposable profiles** for this trial. (Recovery is deferred by the owner: if you forget the Admin passphrase, the test profile's Admin can't be recovered — just delete `~/p1c/profiles` and start over.) **Don't reuse profiles from the first P1 build**: features installed there have a different artifact digest and are now refused (by design), and an older Admin record doesn't know your primary Thought Processor, so every approval page would show a warning.

**Already running Identity on 7790 or Processor on 7860?** Leave them alone and give this trial its own ports by adding `-port 7792` to the Identity command and `-port 7862` to the Processor command (then use the Processor link it prints).

**Terminal 1 — Identity** (leave it open; it shows pairing codes and approval links):

```bash
~/p1c/identity-p1 -data ~/p1c/profiles/identity
```

**Terminal 2 — Thought Processor:**

```bash
~/p1c/thought-processor-p1 -data ~/p1c/profiles/processor -trust-feature-dir ~/p1c/Thought-Processor/processor/testfeatures
```

It prints an `open :` link. On WSL, if no browser opens, **copy that link into your Windows browser** (WSL forwards `127.0.0.1` to Windows by default). The link is private — don't share it. In the Library the test feature shows **pinned artifact** (built-in pin for the reviewed fixture).

Then follow **[TEST_P1.md](TEST_P1.md)**. In short:

1. **Connect** Processor to Identity (type the code shown in terminal 1 — the line next to it names the program asking, e.g. *Thought Processor*).
2. **Admin**: Sign in to GitHub → *Establish Admin Identity…* → *Open Identity approval page* → choose a passphrase (12+ characters), tick the confirmation, *Establish Admin Identity*. This Processor becomes your **primary Thought Processor**.
3. **+ Identity**: add `Tass` (companion).
4. **+ Feature** → *Install…* → *Ask the Admin to approve* → *Open Identity approval page* → passphrase → *Approve*.
5. Open the feature → **Runtime**: *Start*, *Run standalone self-test*.
6. **Connections**: *Send synthetic request* (denied), *Ask the Admin…* (expiry defaults to 1 hour; approve in Identity), send again (completed), *Revoke*, send again (denied).
7. Stop Identity (Ctrl+C in terminal 1) → **AUTHORITY UNAVAILABLE**; start it again → it reconnects; same IDs.

**No real GitHub account?** Use the labelled fixture instead of step 2's real sign-in (terminal 3, before starting Identity). If 8790 is taken, pick any free port and use it in all three places:

```bash
python3 -I ~/p1c/identity/scripts/fixture_github_engine.py 8790 ~/p1c/identity/README.md linux-amd64
IDENTITY_GITHUB_CLIENT_ID=IvFIXTURE00000001 ~/p1c/identity-p1 -data ~/p1c/profiles/identity-fixture \
  -test-github-web http://127.0.0.1:8790 -test-github-api http://127.0.0.1:8790
```

Everything then says **TEST FIXTURE** (the Admin is bound to fixture account `Arameair`, id 1001).

### Options added in the cleanup build

| Program | Option | What it does |
| --- | --- | --- |
| Processor | `-trust-feature-artifact <sha256>` | Pins one more exact feature artifact (repeatable). The reviewed fixture is built in; you don't need this for the trial. Identity needs the same pin: |
| Identity | `-trust-fixture-artifact <sha256>` | Lets Identity register that exact artifact. Without it Identity refuses (`ARTIFACT_NOT_PINNED`), whatever Processor says. |
| Processor | `-test-feature-faults` | TEST MODE: test features obey their fault-injection file (crash/delay). Shows a banner. Only for the recovery test (TEST_P1 5.3). |
| Processor | `-identity-default-url -` | Don't look for an Identity on the default port 7790 at all (what the test scripts use). |

## 5. Stop and clean up

*Exit* in the page (stops Processor and every feature process it started), then Ctrl+C in terminal 1. Delete `~/p1c/profiles` to start from scratch, and `~/p1c/run-*` / `~/p1c/shots` to remove test output.

## Never share

Pairing codes, GitHub device codes, anything after `#code=`, `#key=` or `/approve#`, the Admin passphrase, `identity-pairing.json`, `credential.json` files. The reports and logs produced by the tools contain none of these.
