"""P0 integration test — REAL Thought Processor + REAL Identity processes, with
GitHub replaced by a labelled TEST FIXTURE (Identity's scripts/fixture_github_engine.py).

A headless browser clicks through Processor's own page like a person would;
faults are injected through the fixture. Linux developer evidence, not a
Windows run and not a real release.

  python3 -I acceptance/p0_integration.py --identity-src DIR --work DIR [--shots DIR]

DIR for --identity-src is a checkout of Arameair/identity at
experiment/feature-contract-20261026.
"""
import argparse, hashlib, http.cookiejar, json, os, re, signal, socket, subprocess, sys, time, urllib.request
from pathlib import Path

ap = argparse.ArgumentParser()
ap.add_argument("--identity-src", required=True)
ap.add_argument("--work", required=True)
ap.add_argument("--shots", default="")
args = ap.parse_args()
W = Path(args.work).resolve()
W.mkdir(parents=True, exist_ok=True, mode=0o700)
os.chmod(W, 0o700)
SHOTS = Path(args.shots).resolve() if args.shots else None
if SHOTS:
    SHOTS.mkdir(parents=True, exist_ok=True)
PROC_SRC = Path(__file__).resolve().parent.parent
def free_port():
    # OS-chosen ports: never 7790/7860/8790 or anything the owner already runs.
    with socket.socket() as s:
        s.bind(("127.0.0.1", 0))
        return s.getsockname()[1]


FX, IP, IP2, IP3, PP = (free_port() for _ in range(5))
LOC = W / "locators"
# TP_IDENTITY_DEFAULT_URL="-": never probe the default Identity port 7790 (Grok M4).
ENV = dict(os.environ, IDENTITY_LOCATOR_DIR=str(LOC), IDENTITY_GITHUB_CLIENT_ID="IvFIXTURE00000001", TP_IDENTITY_DEFAULT_URL="-")
results, procs = [], {}


def rec(ok, name, detail=""):
    results.append(("PASS" if ok else "FAIL", name, detail))
    print(("PASS " if ok else "FAIL ") + name + ("  — " + detail if detail else ""), flush=True)


def build():
    br_id = subprocess.check_output(["git", "-C", args.identity_src, "rev-parse", "--abbrev-ref", "HEAD"], text=True).strip()
    br_tp = subprocess.check_output(["git", "-C", str(PROC_SRC), "rev-parse", "--abbrev-ref", "HEAD"], text=True).strip()
    subprocess.check_call(["go", "build", "-ldflags", "-X main.buildBranch=" + br_id, "-o", str(W / "identity"), "."], cwd=args.identity_src)
    subprocess.check_call(["go", "build", "-ldflags", "-X main.buildBranch=" + br_tp, "-o", str(W / "thought-processor"), "."], cwd=PROC_SRC)
    asset = W / "fake-engine.bin"
    asset.write_bytes(b"TEST FIXTURE - not a real Thought Engine\n" * 20000)
    return asset


def start(name, cmd):
    log = os.fdopen(os.open(W / (name + ".console.txt"), os.O_WRONLY | os.O_CREAT | os.O_TRUNC, 0o600), "w")
    p = subprocess.Popen(cmd, stdout=log, stderr=subprocess.STDOUT, env=ENV)
    procs[name] = p
    return p


def stop(name, sig=signal.SIGINT):
    p = procs.pop(name, None)
    if p and p.poll() is None:
        p.send_signal(sig)
        try:
            p.wait(10)
        except subprocess.TimeoutExpired:
            p.kill()


def wait_http(url, t=15):
    end = time.time() + t
    while time.time() < end:
        try:
            urllib.request.urlopen(url, timeout=1)
            return True
        except Exception:
            time.sleep(0.2)
    return False


def start_identity(name, port, data):
    start(name, [str(W / "identity"), "-port", str(port), "-data", str(W / data),
                 "-test-github-web", f"http://127.0.0.1:{FX}", "-test-github-api", f"http://127.0.0.1:{FX}"])
    assert wait_http(f"http://127.0.0.1:{port}/api/health"), "identity did not start"


def start_processor():
    start("processor", [str(W / "thought-processor"), "-port", str(PP), "-data", str(W / "processor-profile"), "-no-browser"])
    assert wait_http(f"http://127.0.0.1:{PP}/api/health"), "processor did not start"
    for _ in range(50):
        f = W / "processor-profile" / "runtime" / "operator-link.txt"
        if f.exists():
            return f.read_text().strip()
        time.sleep(0.1)
    raise SystemExit("no operator link")


def fixture(path):
    urllib.request.urlopen(urllib.request.Request(f"http://127.0.0.1:{FX}{path}", data=b"", method="POST"), timeout=5).read()


class TP:
    """Processor's own API, as its page uses it."""

    def __init__(self, link):
        self.base = f"http://127.0.0.1:{PP}"
        self.jar = http.cookiejar.CookieJar()
        self.op = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(self.jar))
        code = re.search(r"#code=([0-9a-f]+)", link).group(1)
        self.csrf = self._req("POST", "/api/session", {"code": code})[1]["csrf"]

    def _req(self, method, path, body=None):
        data = json.dumps(body or {}).encode() if method == "POST" else None
        r = urllib.request.Request(self.base + path, data=data, method=method)
        if method == "POST":
            r.add_header("Content-Type", "application/json")
            r.add_header("Origin", self.base)
            if getattr(self, "csrf", None):
                r.add_header("X-TP-CSRF", self.csrf)
        try:
            with self.op.open(r, timeout=30) as resp:
                return resp.status, json.loads(resp.read() or b"{}")
        except urllib.error.HTTPError as e:
            return e.code, json.loads(e.read() or b"{}")

    def state(self):
        return self._req("GET", "/api/state")[1]

    def post(self, path, body=None):
        return self._req("POST", path, body)

    def until(self, pred, t=30, what=""):
        end = time.time() + t
        s = self.state()
        while time.time() < end:
            if pred(s):
                return s
            time.sleep(0.5)
            s = self.state()
        print("  (timed out waiting for " + what + "): connection=" + json.dumps(s.get("connection", {}).get("state")))
        return s


def card(s, fid="thought-engine"):
    for c in s.get("cards") or []:
        if c["feature"] == fid:
            return c
    return None


def code_from(name):
    m = re.findall(r"\b([A-HJ-NP-Z2-9]{4}-[A-HJ-NP-Z2-9]{4})\b", (W / (name + ".console.txt")).read_text())
    return m[-1] if m else ""


def partials():
    return list((W / "processor-profile" / "features").glob("*/*.partial"))


def main():
    asset = build()
    sha = hashlib.sha256(asset.read_bytes()).hexdigest()
    start("fixture", [sys.executable, "-I", str(Path(args.identity_src) / "scripts" / "fixture_github_engine.py"), str(FX), str(asset), "linux-amd64"])
    assert wait_http(f"http://127.0.0.1:{FX}/fixture/state")

    pw = page = None
    if SHOTS:
        from playwright.sync_api import sync_playwright
        pw = sync_playwright().start()
        page = pw.chromium.launch().new_page(viewport={"width": 1280, "height": 860})

    def shot(n):
        if page:
            page.wait_for_timeout(2300)
            page.screenshot(path=str(SHOTS / (n + ".png")), full_page=True)

    # 1. Processor first: no Identity yet.
    link = start_processor()
    tp = TP(link)
    s = tp.until(lambda s: s["connection"]["state"] == "IDENTITY_NOT_FOUND", 10, "not found")
    rec(s["connection"]["state"] == "IDENTITY_NOT_FOUND", "Processor alone shows 'Start Identity'", s["connection"]["state"])
    if page:
        page.goto(link)
        shot("p01-start-identity")

    start_identity("identity", IP, "identity-profile")
    s = tp.until(lambda s: s["connection"]["state"] == "FOUND_NOT_CONNECTED", 15, "found")
    cands = s["connection"].get("candidates") or []
    rec(s["connection"]["state"] == "FOUND_NOT_CONNECTED" and any(c.get("identity_system_id") for c in cands),
        "Identity started later is found via its locator", f"{len(cands)} candidate(s)")
    shot("p02-identity-found")

    # 2. Pairing through the page.
    if page:
        page.click("button[data-connect]")
        page.wait_for_selector("#code-in")
        shot("p03-type-code")
        page.fill("#code-in", "ZZZZ-ZZZZ")
        page.click("#code-form button[type=submit]")
        page.wait_for_timeout(800)
        wrong = page.inner_text("#c-error")
        rec("wrong or expired" in wrong, "wrong code refused with an explanation", wrong[:80])
        page.fill("#code-in", code_from("identity"))
        page.click("#code-form button[type=submit]")
    else:
        tp.post("/api/connect/request", {"url": cands[0]["base_url"]})
        tp.post("/api/connect/code", {"code": code_from("identity")})
    s = tp.until(lambda s: s["verified"], 15, "verified")
    rec(s["verified"] and s["connection"]["state"] == "CONNECTED", "paired: signed status from Identity", s["connection"].get("identity_system_id", ""))
    pairing = json.loads((W / "processor-profile" / "identity-pairing.json").read_text())
    rec("sealed_token" in pairing and "token" not in pairing, "pairing token stored sealed, not as a plain field", pairing.get("token_protection"))

    # 3. Identities; private feature hidden before GitHub sign-in.
    if page:
        page.fill("#new-name", "Tass")
        page.select_option("#new-kind", "companion")
        page.click("#new-id button")
        page.wait_for_timeout(1200)
    else:
        tp.post("/api/identities", {"name": "Tass", "kind": "companion"})
    s = tp.until(lambda s: s.get("selected"), 10, "selected identity")
    rec(bool(s.get("selected")) and s["selected"]["name"] == "Tass" and s["selected"]["authorized_sources"] == [],
        "identity created in Identity, selected, starts with no access", s.get("selected", {}).get("id", ""))
    rec(card(s) is None and "Sign in to GitHub" in (s.get("catalog_note") or ""), "private Thought Engine hidden before sign-in", s.get("catalog_note", "")[:60])
    shot("p04-identity-no-features-yet")

    # 4. GitHub sign-in via Identity (fixture approves after two polls).
    if page:
        page.click("#catalog-note button")
        page.wait_for_selector("#m-github:not([hidden])", timeout=8000)
        shot("p05-github-code")
    else:
        tp.post("/api/github/connect")
    s = tp.until(lambda s: (s.get("github") or {}).get("state") == "VERIFIED" and card(s), 45, "github verified")
    c = card(s)
    rec(bool(c) and c["eligible_to_install"]["state"] == "ALLOWED" and c["action"] == "install" and c["visibility"] == "private",
        "after sign-in Thought Engine appears, eligible to install", (c or {}).get("action", ""))
    rec(bool(c) and c.get("enabled_for_identity") == "NOT_GRANTED" and c.get("authorized_sources") == [],
        "eligible to install is not permission to run or read sources", (c or {}).get("enabled_for_identity", ""))
    shot("p06-engine-available")

    # 5. Unapproved source refused before anything downloads.
    fixture("/fixture/mode?uploader=someone-else")
    tp.post("/api/features/thought-engine/plan")
    s = tp.state()
    p = (card(s) or {}).get("install", {}).get("problem") or {}
    rec(p.get("code") == "UNAPPROVED_SOURCE" and not partials(), "release uploaded by a person is refused", p.get("message", "")[:70])
    fixture("/fixture/mode?uploader=github-actions[bot]")

    # 6. No release published.
    fixture("/fixture/mode?release=no")
    tp.post("/api/features/thought-engine/plan")
    p = (card(tp.state()) or {}).get("install", {}).get("problem") or {}
    rec(p.get("code") == "NO_RELEASE", "no published release reported honestly", p.get("message", "")[:70])
    fixture("/fixture/mode?release=yes")

    # 7/8. Tampered and interrupted downloads install nothing.
    for mode, want in (("tamper", "CHECKSUM_MISMATCH"), ("cut", "DOWNLOAD_INTERRUPTED")):
        fixture("/fixture/mode?asset=" + mode)
        tp.post("/api/features/thought-engine/plan")
        plan = card(tp.state())["install"].get("plan")
        tp.post("/api/features/thought-engine/approve", {"plan_id": plan["plan_id"]})
        s = tp.until(lambda s: not card(s)["install"]["busy"] and card(s)["install"].get("problem"), 20, mode)
        iv = card(s)["install"]
        rec(iv["problem"]["code"] == want and not iv.get("installed") and not partials(),
            f"{mode} download: {want}, nothing installed, no leftover file", iv["problem"]["message"][:70])
        if mode == "tamper":
            shot("p07-checksum-mismatch")
    fixture("/fixture/mode?asset=good")

    # 9. Approval required, then a verified install.
    if page:
        page.wait_for_timeout(2200)
        page.click("button[data-plan='thought-engine']")
        page.wait_for_selector("#m-approve:not([hidden])", timeout=8000)
        shot("p08-approve-install")
        page.click("#a-ok")
    else:
        tp.post("/api/features/thought-engine/plan")
        tp.post("/api/features/thought-engine/approve", {"plan_id": card(tp.state())["install"]["plan"]["plan_id"]})
    s = tp.until(lambda s: (card(s) or {}).get("install", {}).get("phase") == "INSTALLED" and not card(s)["install"]["busy"], 30, "installed")
    iv = card(s)["install"]
    inst = iv.get("installed") or {}
    f = W / "processor-profile" / "features" / "thought-engine" / inst.get("tag", "x") / inst.get("asset", "x")
    on_disk = f.exists() and hashlib.sha256(f.read_bytes()).hexdigest() == sha
    rec(iv["verify"] == "OK" and inst.get("sha256") == sha and on_disk, "approved install: file on disk matches approved SHA-256", inst.get("tag", ""))
    rec(inst.get("uploader") == "github-actions[bot]" and inst.get("repo") == "Arameair/Thought-engine", "install record keeps where it came from", inst.get("repo", ""))
    shot("p09-installed")

    # 10. Restart Processor: reconnects without a code; install recognised.
    tp.post("/api/exit", {"choice": "keep"})
    procs["processor"].wait(10)
    rec(procs["processor"].returncode == 0 and wait_http(f"http://127.0.0.1:{IP}/api/health", 2), "Exit → Keep running: Processor stops, Identity stays up")
    procs.pop("processor")
    link = start_processor()
    tp = TP(link)
    s = tp.until(lambda s: s["verified"] and card(s), 20, "reconnect")
    c = card(s) or {}
    rec(s["verified"] and c.get("install", {}).get("verify") == "OK", "Processor restart: reconnected with no code, install re-verified", c.get("install", {}).get("phase", ""))
    if page:
        page.goto("about:blank")  # a new run's link differs only after '#': force a real load
        page.goto(link)

    # 11. Identity down → fail closed; installed state kept.
    stop("identity")
    s = tp.until(lambda s: s["connection"]["state"] == "AUTHORITY_UNAVAILABLE", 20, "unavailable")
    st, r = tp.post("/api/features/thought-engine/plan")
    st2, _ = tp.post("/api/identities", {"name": "X", "kind": "person"})
    c = card(s) or {}
    rec(s["connection"]["state"] == "AUTHORITY_UNAVAILABLE" and st == 503 and st2 == 503 and c.get("install", {}).get("installed"),
        "Identity down: protected actions blocked (503), installed feature still shown", f"plan {st}, create {st2}")
    shot("p10-identity-unavailable")

    # 11b. An impostor grabs Identity's port: never trusted, never gets the secret.
    paired = json.loads((W / "processor-profile" / "identity-pairing.json").read_text())
    imp_log = W / "impostor-requests.jsonl"
    start("impostor", [sys.executable, "-I", str(PROC_SRC / "acceptance" / "impostor_identity.py"), str(IP), paired["identity_system_id"], str(imp_log)])
    wait_http(f"http://127.0.0.1:{IP}/api/health")
    s = tp.until(lambda s: s["connection"]["state"] == "WRONG_SERVICE", 20, "impostor detected")
    st, _ = tp.post("/api/features/thought-engine/plan")
    import base64
    secret = base64.b64decode(paired["sealed_token"]).decode() if paired.get("token_protection", "").startswith("file") else None
    seen = imp_log.read_text() if imp_log.exists() else ""
    rec(s["connection"]["state"] == "WRONG_SERVICE" and not s["verified"] and st != 200 and "IMPOSTOR" not in json.dumps(s),
        "impostor on Identity's port is not trusted; actions refused", s["connection"]["state"])
    rec(secret is not None and len(seen) > 0 and secret not in seen, "the pairing secret never reached the impostor", f"{seen.count(chr(10))} requests inspected")
    stop("impostor")

    # 12. A different Identity installation on the same port = not trusted.
    start_identity("other", IP, "other-identity-profile")
    s = tp.until(lambda s: s["connection"]["state"] in ("CONNECTION_REMOVED", "WRONG_SERVICE"), 20, "wrong")
    rec(s["connection"]["state"] in ("CONNECTION_REMOVED", "WRONG_SERVICE"), "other Identity on the same port is not trusted", s["connection"]["state"])

    # 13. Our Identity comes back on a NEW port → found by installation ID.
    start_identity("identity", IP2, "identity-profile")
    s = tp.until(lambda s: s["verified"], 25, "relocate")
    rec(s["verified"] and s["connection"].get("base_url", "").endswith(str(IP2)), "paired Identity found again on a new port (locator)", s["connection"].get("base_url", ""))
    stop("other")
    s = tp.until(lambda s: s["verified"], 10, "settle")
    c = card(s)
    rec(c is not None and c.get("install", {}).get("installed") and not c.get("from_identity"),
        "after Identity restart (GitHub signed out) the private feature is hidden; install still shown as installed", (c or {}).get("note", "")[:60])

    # 14. Another GitHub account that may not see the feature.
    fixture("/fixture/login?user=brother-account")
    tp.post("/api/github/connect")
    s = tp.until(lambda s: (s.get("github") or {}).get("state") == "VERIFIED", 45, "brother verified")
    c = card(s)
    rec((s.get("github") or {}).get("account", {}).get("login") == "brother-account" and (c is None or not c.get("from_identity")),
        "account without access never sees the private feature from Identity", "")

    # 15. Installed file changed on disk → detected.
    inst_file = next((W / "processor-profile" / "features" / "thought-engine").glob("*/thought-engine-*"))
    inst_file.write_bytes(b"changed")
    tp.post("/api/features/thought-engine/verify")
    iv = card(tp.state())["install"]
    rec(iv.get("verify") == "FILES_CHANGED", "changed installed file detected", iv.get("verify", ""))

    # 16. Developer Mode is display only.
    tp.post("/api/devmode", {"on": True})
    s = tp.state()
    rec("dev" in s and card(s) and card(s)["action"] == "none", "Developer Mode shows details and grants nothing", "")
    shot("p11-developer-mode")
    tp.post("/api/devmode", {"on": False})

    # 17. Exit → Shut down, also stopping the (externally started) Identity.
    if page:
        page.click("#exit-btn")
        shot("p12-exit-dialog")
    st, r = tp.post("/api/exit", {"choice": "shutdown", "stop_identity": True})
    procs["processor"].wait(15)
    time.sleep(1)
    rec(r.get("identity", "").startswith("stopped") and procs["identity"].poll() is not None and procs["processor"].poll() == 0,
        "Exit → Shut down + stop Identity: both stopped, verified", r.get("identity", ""))
    leftovers = list(LOC.glob("idn_*.json"))
    ours = [x for x in leftovers if "identity-profile" in x.read_text() and "other" not in x.read_text()]
    rec(not ours, "Identity's locator removed on clean stop", "")

    if pw:
        pw.stop()


try:
    main()
except Exception as e:  # a crash is a failure, never a silent pass
    import traceback
    traceback.print_exc()
    rec(False, "test script crashed", repr(e)[:200])
finally:
    for n in list(procs):
        stop(n)
    fails = [r for r in results if r[0] == "FAIL"]
    report = W / "p0-integration-report.txt"
    lines = [f"P0 INTEGRATION REPORT {time.strftime('%Y-%m-%dT%H:%M:%S%z')}",
             "Real Thought Processor + real Identity processes; GitHub = TEST FIXTURE; Linux developer machine.", ""]
    lines += [f"{r[0]}  {r[1]}" + (f"  — {r[2]}" if r[2] else "") for r in results]
    lines += ["", f"RESULT: {'PASS' if not fails else 'FAIL'} ({len(results) - len(fails)}/{len(results)})"]
    report.write_text("\n".join(lines) + "\n")
    print("\n".join(lines[-1:]))
    sys.exit(1 if fails else 0)
