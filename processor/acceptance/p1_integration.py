"""P1 integration test — REAL Identity + REAL Thought Processor + REAL feature
processes (the trusted local test feature in testfeatures/), on Linux/WSL.

What is real: every program, every HTTP call, every signature, every grant
decision, every process start/stop/crash. What is a labelled TEST FIXTURE:
GitHub (Identity's scripts/fixture_github_engine.py) and the feature itself
(a deterministic Python worker, not Fast AI Worker). The "Admin" typing the
passphrase into Identity's approval page is this script.

The script also acts as an INDEPENDENT second program ("actor B", its own
Identity pairing) to try out-of-scope, replayed, tampered and forged requests
directly against Identity and the feature — things Processor's UI never offers.

  python3 -I acceptance/p1_integration.py --identity-src DIR --work DIR [--shots DIR]

Exit code 0 only when every check PASSed.
"""
import argparse, base64, hashlib, hmac, json, os, re, secrets, shutil, signal, socket, subprocess, sys, threading, time, urllib.error, urllib.request
from datetime import datetime, timezone
from pathlib import Path

ap = argparse.ArgumentParser()
ap.add_argument("--identity-src", required=True)
ap.add_argument("--work", required=True)
ap.add_argument("--shots", default="")
ap.add_argument("--keep", action="store_true", help="leave the programs running at the end (for looking around)")
args = ap.parse_args()
W = Path(args.work).resolve()
if W.exists():
    shutil.rmtree(W)
W.mkdir(parents=True, mode=0o700)
os.chmod(W, 0o700)
SHOTS = Path(args.shots).resolve() if args.shots else None
PROC_SRC = Path(__file__).resolve().parent.parent
FEATURES = PROC_SRC / "testfeatures"
def free_port():
    # The OS picks unused loopback ports, so an Identity/Processor the owner
    # already runs (7790 / 7860 / 8790 …) is never touched or contacted.
    with socket.socket() as s:
        s.bind(("127.0.0.1", 0))
        return s.getsockname()[1]


FX, IP, PP = free_port(), free_port(), free_port()
IDENTITY = f"http://127.0.0.1:{IP}"
PROCESSOR = f"http://127.0.0.1:{PP}"
PASS = "correct horse battery staple"
CAP = "worker.test.synthetic"
LOC = W / "locators"
# TP_IDENTITY_DEFAULT_URL="-": this run's Processor never probes the default
# Identity port, so a stranger on 7790 can't interfere (Grok M4).
ENV = dict(os.environ, IDENTITY_LOCATOR_DIR=str(LOC), IDENTITY_GITHUB_CLIENT_ID="IvFIXTURE00000001", TP_IDENTITY_DEFAULT_URL="-")
ENV.pop("HTTPS_PROXY", None), ENV.pop("HTTP_PROXY", None), ENV.pop("https_proxy", None), ENV.pop("http_proxy", None)
NOPROXY = urllib.request.build_opener(urllib.request.ProxyHandler({}))
results, procs, evidence = [], {}, {}
T0 = time.time()


def rec(gate, ok, name, detail=""):
    word = "PASS" if ok else "FAIL"
    results.append((gate, word, name, detail))
    print(f"{word} [{gate}] {name}" + ("  — " + detail if detail else ""), flush=True)
    return ok


def http(method, url, body=None, headers=None, raw=False, timeout=30):
    data = body if isinstance(body, (bytes, type(None))) else json.dumps(body).encode()
    r = urllib.request.Request(url, data=data if method == "POST" else None, method=method)
    for k, v in (headers or {}).items():
        r.add_header(k, v)
    try:
        resp = NOPROXY.open(r, timeout=timeout)
        st, b, h = resp.status, resp.read(), resp.headers
    except urllib.error.HTTPError as e:
        st, b, h = e.code, e.read(), e.headers
    if raw:
        return st, b, h
    try:
        return st, json.loads(b or b"{}")
    except Exception:
        return st, {"_raw": b[:200].decode(errors="replace")}


# ---------------- processes ----------------

def build():
    ids = subprocess.check_output(["git", "-C", args.identity_src, "rev-parse", "HEAD"], text=True).strip()
    tps = subprocess.check_output(["git", "-C", str(PROC_SRC), "rev-parse", "HEAD"], text=True).strip()
    evidence["identity_commit"], evidence["processor_commit"] = ids, tps
    subprocess.check_call(["go", "build", "-o", str(W / "identity"), "."], cwd=args.identity_src)
    subprocess.check_call(["go", "build", "-o", str(W / "thought-processor"), "."], cwd=PROC_SRC)
    asset = W / "fixture-asset.bin"
    asset.write_bytes(b"TEST FIXTURE\n")
    return asset


def start(name, cmd, env=None):
    log = os.fdopen(os.open(W / (name + ".console.txt"), os.O_WRONLY | os.O_CREAT | os.O_APPEND, 0o600), "a")
    p = subprocess.Popen(cmd, stdout=log, stderr=subprocess.STDOUT, env=env or ENV)
    procs[name] = p
    return p


def stop(name, sig=signal.SIGINT, t=20):
    p = procs.pop(name, None)
    if p and p.poll() is None:
        p.send_signal(sig)
        try:
            p.wait(t)
        except subprocess.TimeoutExpired:
            p.kill()
            p.wait()
    return p


def wait_http(url, t=15):
    end = time.time() + t
    while time.time() < end:
        try:
            NOPROXY.open(url, timeout=1)
            return True
        except Exception:
            time.sleep(0.2)
    return False


def start_identity():
    start("identity", [str(W / "identity"), "-port", str(IP), "-data", str(W / "identity-profile"),
                       "-test-github-web", f"http://127.0.0.1:{FX}", "-test-github-api", f"http://127.0.0.1:{FX}"])
    assert wait_http(IDENTITY + "/api/health"), "identity did not start"


def start_processor(env=None):
    f = W / "processor-profile" / "runtime" / "operator-link.txt"
    if f.exists():
        f.unlink()
    start("processor", [str(W / "thought-processor"), "-port", str(PP), "-data", str(W / "processor-profile"), "-no-browser", "-test-feature-faults",
                        "-trust-feature-dir", str(FEATURES), "-trust-feature-dir", str(W / "trusted-but-blocked")], env=env)
    assert wait_http(PROCESSOR + "/api/health"), "processor did not start"
    for _ in range(100):
        if f.exists():
            return f.read_text().strip()
        time.sleep(0.1)
    raise SystemExit("no operator link")


def fixture(path):
    NOPROXY.open(urllib.request.Request(f"http://127.0.0.1:{FX}{path}", data=b"", method="POST"), timeout=5).read()


def console_code(name):
    m = re.findall(r"\b([A-HJ-NP-Z2-9]{4}-[A-HJ-NP-Z2-9]{4})\b", (W / (name + ".console.txt")).read_text())
    return m[-1] if m else ""


def pid_alive(pid):
    try:
        os.kill(pid, 0)
        return True
    except OSError:
        return False


# ---------------- Processor's own page API ----------------

class TP:
    def __init__(self, link):
        self.op = urllib.request.build_opener(urllib.request.ProxyHandler({}), urllib.request.HTTPCookieProcessor(http_cookiejar()))
        code = re.search(r"#code=([0-9a-f]+)", link).group(1)
        self.csrf = None
        self.csrf = self.post("/api/session", {"code": code})[1]["csrf"]

    def _req(self, method, path, body=None, timeout=60):
        data = json.dumps(body or {}).encode() if method == "POST" else None
        r = urllib.request.Request(PROCESSOR + path, data=data, method=method)
        if method == "POST":
            r.add_header("Content-Type", "application/json")
            r.add_header("Origin", PROCESSOR)
            if self.csrf:
                r.add_header("X-TP-CSRF", self.csrf)
        try:
            with self.op.open(r, timeout=timeout) as resp:
                return resp.status, json.loads(resp.read() or b"{}")
        except urllib.error.HTTPError as e:
            return e.code, json.loads(e.read() or b"{}")

    def state(self):
        return self._req("GET", "/api/state")[1]

    def post(self, path, body=None, timeout=60):
        return self._req("POST", path, body, timeout)

    def get(self, path):
        return self._req("GET", path)

    def until(self, pred, t=40, what=""):
        end = time.time() + t
        s = self.state()
        while time.time() < end:
            try:
                if pred(s):
                    return s
            except (KeyError, TypeError, IndexError):
                pass
            time.sleep(0.4)
            s = self.state()
        print("   (timed out waiting for " + what + ")", flush=True)
        return s


def http_cookiejar():
    import http.cookiejar as cj
    return cj.CookieJar()


def inst(s, ins_id):
    for v in s.get("installations") or []:
        if v["record"]["installation_id"] == ins_id:
            return v
    return None


def ident_inst(s, ins_id):
    for v in s.get("identity_installations") or []:
        if v["installation"]["installation_id"] == ins_id:
            return v
    return None


# ---------------- the Admin, in Identity's own page ----------------

def admin_decide(url, body):
    import http.cookiejar as cj
    op = urllib.request.build_opener(urllib.request.ProxyHandler({}), urllib.request.HTTPCookieProcessor(cj.CookieJar()))
    m = re.search(r"#a=(apr_[0-9a-f]+)&t=([0-9a-f]+)", url)

    def post(path, b, csrf=None):
        r = urllib.request.Request(IDENTITY + path, data=json.dumps(b).encode(), method="POST")
        r.add_header("Content-Type", "application/json")
        r.add_header("Origin", IDENTITY)
        if csrf:
            r.add_header("X-Identity-CSRF", csrf)
        try:
            with op.open(r, timeout=30) as resp:
                return resp.status, json.loads(resp.read())
        except urllib.error.HTTPError as e:
            return e.code, json.loads(e.read() or b"{}")
    st, j = post("/api/approval/open", {"id": m.group(1), "token": m.group(2)})
    if st != 200:
        return st, j
    return post("/api/approval/decide", body, j["csrf"])


def admin_session(url):
    """Open the approval page once; return a decide(body) function (same page session)."""
    m = re.search(r"#a=(apr_[0-9a-f]+)&t=([0-9a-f]+)", url)
    import http.cookiejar as cj
    op = urllib.request.build_opener(urllib.request.ProxyHandler({}), urllib.request.HTTPCookieProcessor(cj.CookieJar()))

    def post(path, b, csrf=None):
        r = urllib.request.Request(IDENTITY + path, data=json.dumps(b).encode(), method="POST")
        r.add_header("Content-Type", "application/json")
        r.add_header("Origin", IDENTITY)
        if csrf:
            r.add_header("X-Identity-CSRF", csrf)
        try:
            with op.open(r, timeout=30) as resp:
                return resp.status, json.loads(resp.read())
        except urllib.error.HTTPError as e:
            return e.code, json.loads(e.read() or b"{}")
    st, view = post("/api/approval/open", {"id": m.group(1), "token": m.group(2)})
    return st, view, (lambda body: post("/api/approval/decide", body, view.get("csrf")))


def approve(url, passphrase=PASS):
    return admin_decide(url, {"decision": "approve", "passphrase": passphrase})


# ---------------- actor B: an independent paired program ----------------

class Signer:
    """identity/2 signing, implemented independently of the Go code."""

    def __init__(self, kind, cid, secret, system_id):
        self.kind, self.id, self.system_id = kind, cid, system_id
        s = secret.encode()
        self.rk = hmac.new(s, b"identity/2 request key", hashlib.sha256).digest()
        self.sk = hmac.new(s, b"identity/2 response key", hashlib.sha256).digest()
        self.tk = hmac.new(s, b"identity/2 ticket key", hashlib.sha256).digest()

    def call(self, method, path, obj=None):
        body = json.dumps(obj).encode() if obj is not None else b""
        ts, nonce = str(int(time.time())), secrets.token_hex(12)
        mac = hmac.new(self.rk, ("identity/2 request\n%s\n%s\n%s\n%s\n%s" % (method, path, ts, nonce, hashlib.sha256(body).hexdigest())).encode(), hashlib.sha256).hexdigest()
        hdr = {"Content-Type": "application/json", "X-Identity-Time": ts, "X-Identity-Nonce": nonce, "X-Identity-Auth": mac,
               ("X-Processor-ID" if self.kind == "processor" else "X-Installation-ID"): self.id}
        try:
            st, raw, h = http(method, IDENTITY + path, body if method == "POST" else None, hdr, raw=True, timeout=10)
        except Exception as e:
            return 0, {"_error": "UNREACHABLE " + type(e).__name__}
        j = json.loads(raw or b"{}") if raw else {}
        if st not in (401, 503) or h.get("X-Identity-Proof"):
            want = hmac.new(self.sk, ("identity/2 response\n%s\n%d\n" % (nonce, st)).encode() + raw, hashlib.sha256).hexdigest()
            if want != h.get("X-Identity-Proof") or h.get("X-Identity-System-ID") != self.system_id:
                j["_unsigned"] = True
        return st, j


def pair_actor_b():
    pid = "tp_harness_b"
    st, _ = http("POST", IDENTITY + "/api/v1/pair/request", {"processor_id": pid, "processor_label": "P1 harness actor B"}, {"Content-Type": "application/json"})
    time.sleep(0.5)
    st, j = http("POST", IDENTITY + "/api/v1/pair", {"code": console_code("identity"), "processor_id": pid, "processor_label": "P1 harness actor B"},
                 {"Content-Type": "application/json"})
    if st != 200:
        raise SystemExit("actor B could not pair: %s %s" % (st, j))
    return Signer("processor", pid, j["secret"], j["identity_system_id"])


def b_grant(B, ins_id, subject, expires=0):
    st, j = B.call("POST", "/api/v1/grants/requests", {"destination_installation_id": ins_id, "subject_identity_id": subject,
                                                      "capability": CAP, "expires_in_seconds": expires})
    if st != 202:
        return None, (st, j)
    st2, j2 = approve(j["approval_url"])
    st3, j3 = B.call("POST", "/api/v1/grants", {"approval_id": j["approval"]["id"]})
    return (j3.get("grant") or {}).get("grant_id"), (st3, j3)


def b_ticket(B, ins_id, subject, body, cap=CAP):
    return B.call("POST", "/api/v1/tickets", {"destination_installation_id": ins_id, "subject_identity_id": subject, "capability": cap,
                                              "request_sha256": hashlib.sha256(body).hexdigest()})


def feature_action(url, ticket, body, cap=CAP):
    return http("POST", url + "/v1/actions/" + cap, body, {"Content-Type": "application/json", "X-TF-Ticket": ticket or ""})


# ---------------- screenshots (optional) ----------------

PAGE = {}


def shot(name, path="/"):
    if not SHOTS or "page" not in PAGE:
        return
    try:
        page = PAGE["page"]
        page.reload()
        page.wait_for_timeout(1500)
        page.screenshot(path=str(SHOTS / (name + ".png")), full_page=True)
    except Exception as e:
        print("   (screenshot failed: %s)" % e)


def main():
    asset = build()
    # A trusted folder that holds packages that must stay BLOCKED / INVALID.
    bad = W / "trusted-but-blocked"
    for name, mut in [("third-party", "trust"), ("symlinked", "symlink"), ("traversal", "traversal")]:
        d = bad / name
        shutil.copytree(FEATURES / "deterministic-worker", d)
        m = json.loads((d / "feature.json").read_text())
        m["feature_id"] = "p1-blocked-" + name
        if mut == "trust":
            m["trust_class"] = "THIRD_PARTY_RELEASE"
        if mut == "symlink":
            (d / "worker.py").unlink()
            os.symlink("/etc/hostname", d / "worker.py")
        if mut == "traversal":
            m["files"] = [{"path": "../../etc/passwd", "sha256": "0" * 64}]
        (d / "feature.json").write_text(json.dumps(m, indent=2))

    start("fixture", [sys.executable, "-I", str(Path(args.identity_src) / "scripts" / "fixture_github_engine.py"), str(FX), str(asset), "linux-amd64"])
    assert wait_http(f"http://127.0.0.1:{FX}/fixture/state")
    fixture("/fixture/login?user=Arameair")
    start_identity()
    link = start_processor()
    tp = TP(link)
    if SHOTS:
        from playwright.sync_api import sync_playwright
        pw = sync_playwright().start()
        PAGE["pw"] = pw
        PAGE["page"] = pw.chromium.launch().new_page(viewport={"width": 1360, "height": 900}, color_scheme="dark")
        PAGE["page"].goto(link)

    # ---------- connect (P0 flow, still required) ----------
    s = tp.until(lambda s: s["connection"]["state"] == "FOUND_NOT_CONNECTED", 20, "Identity found")
    tp.post("/api/connect/request", {"url": IDENTITY})
    time.sleep(0.5)
    tp.post("/api/connect/code", {"code": console_code("identity")})
    s = tp.until(lambda s: s["verified"], 20, "connected")
    rec("G1", s["verified"], "Processor paired with Identity (identity/2 signed requests + signed responses)")
    system_id = s["connection"]["identity_system_id"]
    evidence["identity_system_id"] = system_id

    # ---------- GATE 1: GitHub-verified Admin bootstrap ----------
    s = tp.until(lambda s: s.get("admin") is not None, 10, "admin view")
    rec("G1", s["admin"]["state"] == "NOT_ENROLLED", "fresh profile: no Admin Identity yet", s["admin"]["state"])
    st, j = tp.post("/api/library/p1-deterministic-worker/install")
    rec("G1", st in (409, 503) and "Admin" in json.dumps(j), "elevated action refused before an Admin exists", "%d %s" % (st, j.get("error", "")[:80]))
    st, j = tp.post("/api/admin/enroll")
    rec("G1", st == 409, "Admin enrollment refused until GitHub is verified", "%d" % st)
    tp.post("/api/github/connect")
    s = tp.until(lambda s: (s.get("github") or {}).get("state") == "VERIFIED", 40, "GitHub verified")
    rec("G1", (s.get("github") or {}).get("state") == "VERIFIED", "GitHub sign-in verified through Identity (TEST FIXTURE GitHub, account Arameair id 1001)")
    shot("p1-01-github-verified-admin-not-enrolled")
    st, j = tp.post("/api/admin/enroll")
    url = j.get("approval_url", "")
    rec("G1", st == 200 and "/approve#a=apr_" in url, "enrollment request returns an approval link to IDENTITY's page (Processor can't approve)")
    if SHOTS:
        p2 = PAGE["pw"].chromium.launch().new_page(viewport={"width": 900, "height": 900}, color_scheme="dark")
        p2.goto(url)
        p2.wait_for_timeout(1200)
        p2.screenshot(path=str(SHOTS / "p1-02-identity-enroll-page.png"), full_page=True)
        # the link was used by the screenshot browser; ask again for the scripted Admin
        st, j = tp.post("/api/admin/enroll")
        url = j.get("approval_url", "")
    st, j = admin_decide(url, {"decision": "approve", "passphrase": PASS, "passphrase_confirm": PASS})
    rec("G1", st == 400, "enrollment without the explicit confirmation tick is refused", str(st))
    st, j = tp.post("/api/admin/enroll")
    st, j = admin_decide(j["approval_url"], {"decision": "approve", "passphrase": PASS, "passphrase_confirm": PASS, "acknowledge": True})
    rec("G1", st == 200, "Admin explicitly established in Identity's page (passphrase never sent to Processor)", str(st))
    s = tp.until(lambda s: s["admin"]["state"] == "ENROLLED", 15, "enrolled")
    adm = s["admin"]["admin"]
    evidence["admin"] = adm
    rec("G1", s["admin"]["state"] == "ENROLLED" and adm["github_binding"]["id"] == 1001, "exactly one durable Admin Identity bound to GitHub numeric id 1001",
        adm["admin_id"])
    shot("p1-03-admin-identity")
    # Account switch cannot take over.
    B = pair_actor_b()
    rec("G1", True, "independent program 'actor B' paired separately (used for negative tests)")
    B.call("POST", "/api/v1/github/disconnect", {})
    fixture("/fixture/login?user=intruder")
    B.call("POST", "/api/v1/github/connect", {})
    end = time.time() + 40
    while time.time() < end:
        st, g = B.call("GET", "/api/v1/github")
        if g.get("state") == "VERIFIED":
            break
        time.sleep(1)
    st, a2 = B.call("GET", "/api/v1/admin")
    rec("G1", g.get("state") == "VERIFIED" and g["account"]["id"] == 2002, "a different GitHub account (intruder, id 2002) signed in")
    rec("G1", a2["admin"]["admin_id"] == adm["admin_id"] and a2["current_github"].get("is_admin_binding") is False,
        "account switch did NOT change or transfer the Admin Identity")
    st, j = B.call("POST", "/api/v1/admin/enrollment", {})
    rec("G1", st == 409 and j.get("error") == "ALREADY_ENROLLED", "second enrollment refused (no takeover by signing in)", str(st))
    st, j = B.call("POST", "/api/v1/grants", {"approved": True})
    rec("G1", st == 403 and j.get("error") == "ADMIN_APPROVAL_REQUIRED", "a paired program claiming approved:true is refused (pairing is not Admin authority)")

    # ---------- GATE 2: manifest-driven install of the trusted test feature ----------
    s = tp.state()
    lib = {x.get("feature_id") or x["dir"]: x for x in s["library"]}
    w = lib.get("p1-deterministic-worker")
    rec("G2", w is not None and not w.get("blocked"), "library shows the trusted local test feature from the trusted folder",
        w and w["trust_class"])
    blocked = [x for x in s["library"] if x.get("blocked")]
    rec("G2", len(blocked) == 3 and all("UNTRUSTED" in x["blocked"] or x["blocked"] == "INVALID_PACKAGE" for x in blocked),
        "third-party, symlinked and path-traversal packages are BLOCKED/INVALID", "; ".join((x.get("feature_id") or x["dir"]) + "=" + x["blocked"][:40] for x in blocked))
    st, j = tp.post("/api/library/p1-blocked-third-party/install")
    rec("G2", st == 409 and "UNTRUSTED_EXECUTION_BLOCKED" in j.get("error", ""), "installing a non-fixture package is refused before anything happens")
    shot("p1-04-library")
    st, j = tp.post("/api/library/p1-deterministic-worker/install")
    url = j.get("approval_url", "")
    rec("G2", st == 200 and url, "install request created; waiting for the Admin in Identity", j.get("approval_id", ""))
    st, j2 = approve(url, "not the admin passphrase")
    rec("G2", st == 401, "wrong Admin passphrase refused", str(st))
    st, j = tp.post("/api/install-tx/%s/cancel" % j["install_key"])  # that link is spent; start again
    st, j = tp.post("/api/library/p1-deterministic-worker/install")
    st, j2 = approve(j["approval_url"])
    rec("G2", st == 200 and j2["approval"]["state"] == "APPROVED", "Admin approved the exact install (bound to its SHA-256 action digest)")
    s = tp.until(lambda s: len(s["installations"]) == 1 and s["installations"][0]["identity"]["installation"]["state"] == "INSTALLED", 30, "installed")
    v = s["installations"][0]
    INS1 = v["record"]["installation_id"]
    evidence["installation_1"] = INS1
    rec("G2", len(s["installations"]) == 1 and len(s["identity_installations"]) == 1, "exactly ONE installation identity registered", INS1)
    rec("G2", v["identity"]["active_grants"] == 0 and v["identity"]["information_sources"] == [] and v["identity"]["credential_registered"],
        "it starts with ZERO grants and ZERO information sources")
    rec("G2", v["runtime"]["state"] == "STOPPED" and v["record"]["policy"]["startup_mode"] == "manual",
        "installed but not started; startup mode defaults to MANUAL")
    rec("G2", v["files"] == "OK" and v["record"]["trust_class"] == "TRUSTED_LOCAL_TEST_FIXTURE" and v["identity"]["installation"]["publisher_proof"].startswith("NONE"),
        "files verified; trust class and 'publisher not verified' shown honestly")
    rec("G2", not list((W / "processor-profile" / "installations" / ".tx").iterdir()), "no leftover install transaction")
    shot("p1-05-installed-zero-grants")

    # ---------- GATE 3: standalone run, health, self-test, manual vs on-demand ----------
    s = tp.state()
    ent = lambda name: [e for e in s["identities"] if e["name"] == name]
    st, j = tp.post("/api/identities", {"name": "Tass", "kind": "companion"})
    st, j = tp.post("/api/identities", {"name": "Other", "kind": "person"})
    s = tp.state()
    TASS, OTHER = ent("Tass")[0]["id"], ent("Other")[0]["id"]
    st, j = tp.post(f"/api/installations/{INS1}/action", {"subject": TASS, "capability": CAP, "input": "hello"})
    r = j["result"]
    rec("G3", r["outcome"] == "DENIED_BY_IDENTITY" and not r["started_feature"] and tp.state()["installations"][0]["runtime"]["state"] == "STOPPED",
        "no grant → Identity denies BEFORE anything starts", r["detail"][:70])
    st, j = tp.post(f"/api/installations/{INS1}/start")
    s = tp.until(lambda s: inst(s, INS1)["runtime"]["state"] == "RUNNING", 15, "running")
    rt = inst(s, INS1)["runtime"]
    rec("G3", rt["state"] == "RUNNING" and rt["launch_id"].startswith("lch_"), "manual start: RUNNING, endpoint authenticated with its launch key", rt.get("url", ""))
    pid1 = rt["pid"]
    time.sleep(2.5)
    s = tp.state()
    h1 = inst(s, INS1)["runtime"]["health"]
    time.sleep(2.5)
    h2 = inst(tp.state(), INS1)["runtime"]["health"]
    rec("G3", h1["ready"] and h2["heartbeat_seq"] > h1["heartbeat_seq"], "live heartbeat through the real local API (separate from any observation)",
        "seq %s → %s" % (h1["heartbeat_seq"], h2["heartbeat_seq"]))
    st, j = tp.post(f"/api/installations/{INS1}/self-test")
    stt = j.get("self_test", {})
    rec("G3", stt.get("result") == "PASS" and "SYNTHETIC" in stt.get("data_class", "") and stt.get("external_calls") == 0,
        "standalone self-test PASS, labelled SYNTHETIC TEST DATA, no grant used")
    ep = json.loads((W / "processor-profile" / "installations" / INS1 / "runtime" / "endpoint.json").read_text())
    FURL = ep["url"]
    st, j = http("GET", FURL + "/v1/status")
    rec("G3", st == 401, "feature status refuses callers without the launch key", str(st))
    st, j = http("GET", FURL + "/v1/health")
    rec("G3", st == 200 and "installation_id" not in j, "public health says only 'running' (no identity details)")
    shot("p1-06-running-health")
    tp.post(f"/api/installations/{INS1}/stop")
    s = tp.until(lambda s: inst(s, INS1)["runtime"]["state"] == "STOPPED", 20, "stopped")
    rec("G3", inst(s, INS1)["runtime"]["state"] == "STOPPED" and not pid_alive(pid1), "manual stop: process gone, identity kept", "pid %d" % pid1)

    # ---------- GATE 4: Admin-approved narrow grant, receiver enforcement ----------
    stop("fixture")  # internet/GitHub OFF from here: local Admin approval must still work
    rec("G4", True, "GitHub fixture stopped: from here GitHub is unreachable (offline Admin approval test)")
    st, j = tp.post(f"/api/installations/{INS1}/grants/request", {"subject": TASS, "capability": CAP, "expires_in_seconds": 0})
    gurl = j.get("approval_url", "")
    rec("G4", st == 200 and gurl, "connection request: Processor → " + INS1 + " / " + CAP + " / Tass")
    if SHOTS:
        p2 = PAGE["pw"].chromium.launch().new_page(viewport={"width": 900, "height": 900}, color_scheme="dark")
        p2.goto(gurl)
        p2.wait_for_timeout(1200)
        p2.screenshot(path=str(SHOTS / "p1-07-identity-grant-approval.png"), full_page=True)
        st, j = tp.post(f"/api/installations/{INS1}/grants/request", {"subject": TASS, "capability": CAP, "expires_in_seconds": 0})
        gurl = j["approval_url"]
    st, view, decide = admin_session(gurl)
    rq = (view.get("display") or {}).get("requester") or {}
    st, j = decide({"decision": "approve", "passphrase": PASS})
    rec("G4", st == 200, "Admin approved the grant OFFLINE (GitHub unreachable) with the local passphrase")
    rec("G4", rq.get("verified") is True and rq.get("is_primary") is True and not rq.get("warning") and bool(rq.get("label")),
        "approval page names the requester from Identity's pairing record (primary Processor → no warning)", rq.get("label", ""))
    s = tp.until(lambda s: ident_inst(s, INS1)["active_grants"] >= 1, 20, "grant active")
    g = [x for x in ident_inst(s, INS1)["grants"] if x["status"] == "ACTIVE"]
    GID = g[0]["grant"]["grant_id"] if g else ""
    rec("G4", len(g) == 1 and g[0]["grant"]["requester"] == "processor:" + s["processor"]["processor_id"] and g[0]["grant"]["source_scope"] == "none",
        "exactly one narrow grant: this Processor → this installation → one capability → Tass; sources none", GID)
    exp = g[0]["grant"].get("expires_at") if g else None
    left = (datetime.fromisoformat(exp.replace("Z", "+00:00")) - datetime.now(timezone.utc)).total_seconds() if exp else 0
    rec("G4", 3300 < left <= 3600, "no expiry chosen → Identity's finite default (1 hour), not 'until revoked'", str(exp))
    # 'Never expires' needs a separate, deliberate Admin confirmation.
    st, j = tp.post(f"/api/installations/{INS1}/grants/request", {"subject": OTHER, "capability": CAP, "until_revoked": True})
    sto, view, decide = admin_session(j.get("approval_url", ""))
    st1, _ = decide({"decision": "approve", "passphrase": PASS})
    st2, j2 = decide({"decision": "deny"})
    rec("G4", ((view.get("display") or {}).get("expiry") or {}).get("until_revoked") is True and st1 == 400 and st2 == 200
        and j2["approval"]["state"] == "DENIED", "'until revoked' grant is refused unless the Admin ticks the extra confirmation (then denied here)", "%d/%d" % (st1, st2))
    # on-demand: set on_demand with short idle, then act while stopped
    tp.post(f"/api/installations/{INS1}/policy", {"startup_mode": "on_demand", "idle_seconds": 6, "recovery": "manual"})
    st, j = tp.post(f"/api/installations/{INS1}/action", {"subject": TASS, "capability": CAP, "input": "hello p1"})
    r = j["result"]
    obs = r.get("observation") or {}
    rec("G4", r["outcome"] == "COMPLETED" and obs.get("output", {}).get("upper") == "HELLO P1" and "SYNTHETIC" in obs.get("data_class", ""),
        "ALLOW: one synthetic action completed (ticket verified by the feature + live Identity decision)", r.get("decision_id", ""))
    rec("G3", r["started_feature"], "ON-DEMAND: the stopped feature was started for an already-authorized request")
    evidence["first_allowed_request"] = {k: r[k] for k in ("request_id", "decision_id", "ticket_id", "outcome", "latency_ms")}
    shot("p1-08-allowed-action")
    s = tp.until(lambda s: inst(s, INS1)["runtime"]["state"] == "STOPPED", 20, "idle stop")
    rec("G3", inst(s, INS1)["runtime"]["state"] == "STOPPED", "ON-DEMAND idle window ended → clean stop (identity and data kept)")
    for (subj, cap, why) in [(OTHER, CAP, "wrong subject identity"), (TASS, "worker.test.other", "wrong capability")]:
        st, j = tp.post(f"/api/installations/{INS1}/action", {"subject": subj, "capability": cap, "input": "x"})
        rec("G4", j["result"]["outcome"] == "DENIED_BY_IDENTITY", "DENY " + why, j["result"]["detail"][:60])

    # Actor B: its own grant, then direct attacks on the RECEIVING feature.
    tp.post(f"/api/installations/{INS1}/policy", {"startup_mode": "manual", "idle_seconds": 30, "recovery": "manual"})
    tp.post(f"/api/installations/{INS1}/start")
    s = tp.until(lambda s: inst(s, INS1)["runtime"]["state"] == "RUNNING", 15, "running")
    FURL = json.loads((W / "processor-profile" / "installations" / INS1 / "runtime" / "endpoint.json").read_text())["url"]
    body = json.dumps({"request_id": "b-1", "input": "from actor B"}).encode()
    st, t = b_ticket(B, INS1, TASS, body)
    rec("G4", st == 403 and t.get("reason") == "NO_GRANT", "actor B (different requester) gets NO ticket for Processor's grant", t.get("reason", ""))
    st, j = feature_action(FURL, "tk1.e30." + "0" * 64, body)
    rec("G4", st == 401, "forged ticket refused by the feature", j.get("reason", ""))
    BGID, info = b_grant(B, INS1, TASS)
    rec("G4", bool(BGID), "actor B received its own separately approved grant", BGID or str(info))
    st, t = b_ticket(B, INS1, TASS, body)
    tk = t.get("ticket", "")
    tampered = json.dumps({"request_id": "b-1", "input": "TAMPERED"}).encode()
    st, j = feature_action(FURL, tk, tampered)
    rec("G4", st == 401 and j.get("reason") == "TICKET_BODY_MISMATCH", "tampered body refused (ticket bound to the exact request)")
    st, t = b_ticket(B, INS1, TASS, body)
    tk = t["ticket"]
    st, j = feature_action(FURL, tk, body, "worker.test.other")
    rec("G4", st in (401, 404), "ticket used for a different capability refused", "%d %s" % (st, j.get("error")))
    st, t = b_ticket(B, INS1, TASS, body)
    tk = t["ticket"]
    st, j = feature_action(FURL, tk, body)
    rec("G4", st == 200 and j.get("result") == "COMPLETED", "actor B's valid request ALLOWED by the feature after a live Identity check",
        (j.get("observation") or {}).get("decision_id", ""))
    st, j = feature_action(FURL, tk, body)
    rec("G4", st == 409 and j.get("error") == "TICKET_REPLAYED", "replayed ticket refused by the feature's durable ledger")
    # Concurrent replay: one ticket sent 12 times at once → exactly one ALLOW.
    st, t = b_ticket(B, INS1, TASS, body)
    tk, outs = t.get("ticket", ""), []
    ths = [threading.Thread(target=lambda: outs.append(feature_action(FURL, tk, body)[0])) for _ in range(12)]
    [x.start() for x in ths]
    [x.join() for x in ths]
    rec("G4", st == 200 and sorted(outs).count(200) == 1 and len(outs) == 12, "12 simultaneous sends of one ticket → exactly one allowed", ",".join(map(str, sorted(outs))))
    # Forged-with-key EXPIRED ticket (we read the installation credential as the same OS user: this also shows the same-user limit)
    cred = json.loads((W / "processor-profile" / "installations" / INS1 / "credential.json").read_text())
    tkey = hmac.new(cred["secret"].encode(), b"identity/2 ticket key", hashlib.sha256).digest()
    now = int(time.time())
    pay = {"v": 1, "tid": "tkt_" + secrets.token_hex(12), "gid": BGID, "gv": 1, "iss": system_id, "req": "processor:tp_harness_b", "aud": INS1,
           "cap": CAP, "sub": TASS, "dig": hashlib.sha256(body).hexdigest(), "iat": now - 200, "exp": now - 100}
    b64 = base64.urlsafe_b64encode(json.dumps(pay).encode()).decode().rstrip("=")
    st, j = feature_action(FURL, "tk1." + b64 + "." + hmac.new(tkey, ("identity/2 ticket\n" + b64).encode(), hashlib.sha256).hexdigest(), body)
    rec("G4", st == 401 and j.get("reason") == "TICKET_EXPIRED", "expired ticket refused")
    pay["exp"], pay["tid"] = now + 50, "tkt_" + secrets.token_hex(12)
    b64 = base64.urlsafe_b64encode(json.dumps(pay).encode()).decode().rstrip("=")
    st, j = feature_action(FURL, "tk1." + b64 + "." + hmac.new(tkey, ("identity/2 ticket\n" + b64).encode(), hashlib.sha256).hexdigest(), body)
    rec("G4", st == 403 and j.get("reason") == "TICKET_UNKNOWN", "a ticket Identity never issued (minted with a stolen key) is DENIED by the live check",
        j.get("reason", ""))
    # Revoke while a ticket is outstanding: the receiver denies.
    st, t = b_ticket(B, INS1, TASS, body)
    tk = t["ticket"]
    st, j = B.call("POST", "/api/v1/grants/%s/revoke" % BGID, {"reason": "test"})
    st, j = feature_action(FURL, tk, body)
    rec("G4", st == 403 and j.get("error") == "DENIED", "after REVOCATION the receiving feature denies (no restart of anything)", j.get("reason", ""))
    # Grok M2: an unrelated paired program can't revoke someone else's grant.
    st, j = B.call("POST", "/api/v1/grants/%s/revoke" % GID, {"reason": "B tries"})
    rec("G4", st == 403 and j.get("error") == "REVOKE_NOT_PERMITTED", "actor B can NOT revoke Processor's grant (only its parties or the Admin)", str(st))
    # Processor's grant: revoke through the UI path.
    st, j = tp.post(f"/api/installations/{INS1}/grants/{GID}/revoke")
    st, j = tp.post(f"/api/installations/{INS1}/action", {"subject": TASS, "capability": CAP, "input": "after revoke"})
    rec("G4", j["result"]["outcome"] == "DENIED_BY_IDENTITY" and "REVOKED" in j["result"]["detail"], "after revoke, Processor's next request is denied",
        j["result"]["detail"][:60])
    # Identity unavailable.
    BGID2, _ = b_grant(B, INS1, TASS)
    st, t = b_ticket(B, INS1, TASS, body)
    tk = t["ticket"]
    G4ok = BGID2 and st == 200
    stop("identity")
    st, j = feature_action(FURL, tk, body)
    rec("G4", G4ok and st == 503 and j.get("error") == "AUTHORITY_UNAVAILABLE", "Identity STOPPED → feature answers AUTHORITY_UNAVAILABLE, does no work")
    st, j = tp.post(f"/api/installations/{INS1}/action", {"subject": TASS, "capability": CAP, "input": "no identity"})
    rec("G4", j["result"]["outcome"] == "AUTHORITY_UNAVAILABLE", "Processor: AUTHORITY UNAVAILABLE, distinct from DENIED")
    st, j = tp.post(f"/api/installations/{INS1}/self-test")
    rec("G3", (j.get("self_test") or {}).get("result") == "PASS", "standalone self-test still works while Identity is down (nonprivileged)")
    shot("p1-09-identity-unavailable")

    # ---------- GATE 5: restarts, persistence, recovery ----------
    start_identity()
    s = tp.until(lambda s: s["verified"] and ident_inst(s, INS1) is not None, 30, "reconnected")
    rec("G5", s["verified"] and s["connection"]["identity_system_id"] == system_id, "Identity restarted: same installation system ID, Processor reconnected")
    iv = ident_inst(s, INS1)
    statuses = sorted(x["status"] for x in iv["grants"])
    rec("G5", iv["installation"]["installation_id"] == INS1 and len(s["identity_installations"]) == 1, "installation ID stable, no duplicate registration")
    rec("G5", statuses == ["ACTIVE", "REVOKED", "REVOKED"] and s["admin"]["admin"]["admin_id"] == adm["admin_id"],
        "grant states persisted exactly (1 active, 2 revoked); same Admin", ",".join(statuses))
    st, t = b_ticket(B, INS1, TASS, body)
    st2, j = feature_action(FURL, t.get("ticket", ""), body)
    rec("G5", st == 200 and st2 == 200, "after Identity restart the active grant works again (fresh live decision)")
    # Expiring grant (5 s) for actor B.
    tp.post(f"/api/installations/{INS1}/grants/request", {"subject": OTHER, "capability": CAP, "expires_in_seconds": 5})
    s = tp.state()
    op = [p for p in inst(s, INS1)["pending"] if p["kind"] == "grant_capability"][-1]
    approve(op["approval_url"])
    s = tp.until(lambda s: any(x["grant"]["subject_identity_id"] == OTHER and x["status"] == "ACTIVE" for x in ident_inst(s, INS1)["grants"]), 20, "short grant")
    st, j = tp.post(f"/api/installations/{INS1}/action", {"subject": OTHER, "capability": CAP, "input": "before expiry"})
    ok1 = j["result"]["outcome"] == "COMPLETED"
    time.sleep(6)
    st, j = tp.post(f"/api/installations/{INS1}/action", {"subject": OTHER, "capability": CAP, "input": "after expiry"})
    rec("G4", ok1 and j["result"]["outcome"] == "DENIED_BY_IDENTITY" and "EXPIRED" in j["result"]["detail"], "EXPIRING grant: allowed, then denied after expiry",
        j["result"]["detail"][:50])

    # Processor needs a live grant again (it revoked its own earlier).
    tp.post(f"/api/installations/{INS1}/grants/request", {"subject": TASS, "capability": CAP, "expires_in_seconds": 0})
    op = [p for p in inst(tp.state(), INS1)["pending"] if p["kind"] == "grant_capability"][-1]
    approve(op["approval_url"])
    tp.until(lambda s: any(x["grant"]["subject_identity_id"] == TASS and x["grant"]["requester"].startswith("processor:tp_") and x["grant"]["requester"] != "processor:tp_harness_b"
                           and x["status"] == "ACTIVE" for x in ident_inst(s, INS1)["grants"]), 20, "processor grant again")

    # Crash with manual recovery: request UNCERTAIN, no restart, no resend.
    (W / "processor-profile" / "installations" / INS1 / "data" / "TEST-ONLY-fault-injection.json").write_text(json.dumps({"crash": True, "slow": True}))
    tp.post(f"/api/installations/{INS1}/stop")
    tp.until(lambda s: inst(s, INS1)["runtime"]["state"] == "STOPPED", 20, "stopped")
    tp.post(f"/api/installations/{INS1}/start")
    tp.until(lambda s: inst(s, INS1)["runtime"]["state"] == "RUNNING", 15, "running")
    st, j = tp.post(f"/api/installations/{INS1}/action", {"subject": TASS, "capability": CAP, "input": "__crash__"})
    rec("G5", j["result"]["outcome"] == "REQUEST_UNCERTAIN", "feature crashed mid-request → REQUEST_UNCERTAIN, NOT resent", j["result"]["detail"][:60])
    time.sleep(4)
    s = tp.state()
    rt = inst(s, INS1)["runtime"]
    rec("G5", rt["state"] == "NEEDS_ATTENTION" and not rt["recent_restarts"], "experimental default = manual recovery: NEEDS_ATTENTION, no automatic restart",
        rt.get("note", "")[:60])
    shot("p1-10-needs-attention")
    # Bounded recovery: restarts the process, never the request; stops after 3.
    tp.post(f"/api/installations/{INS1}/policy", {"startup_mode": "manual", "idle_seconds": 30, "recovery": "bounded"})
    tp.post(f"/api/installations/{INS1}/start")
    tp.until(lambda s: inst(s, INS1)["runtime"]["state"] == "RUNNING", 15, "running")
    lids = []
    for i in range(4):
        s = tp.until(lambda s: inst(s, INS1)["runtime"]["state"] == "RUNNING", 20, "running again")
        lids.append(inst(s, INS1)["runtime"]["launch_id"])
        st, j = tp.post(f"/api/installations/{INS1}/action", {"subject": TASS, "capability": CAP, "input": "__crash__"})
        time.sleep(0.5)
    s = tp.until(lambda s: inst(s, INS1)["runtime"]["state"] == "NEEDS_ATTENTION", 30, "crash loop stop")
    rt = inst(s, INS1)["runtime"]
    rec("G5", len(rt["recent_restarts"]) == 3 and rt["state"] == "NEEDS_ATTENTION" and len(set(lids)) == 4,
        "bounded recovery: 3 automatic process restarts (new launch each), then crash-loop stop → NEEDS_ATTENTION", "launches " + ",".join(lids))
    os.remove(W / "processor-profile" / "installations" / INS1 / "data" / "TEST-ONLY-fault-injection.json")
    tp.post(f"/api/installations/{INS1}/policy", {"startup_mode": "manual", "idle_seconds": 30, "recovery": "manual"})
    tp.post(f"/api/installations/{INS1}/stop")

    # No idle stop during an active request.
    (W / "processor-profile" / "installations" / INS1 / "data" / "TEST-ONLY-fault-injection.json").write_text(json.dumps({"slow": True}))
    tp.post(f"/api/installations/{INS1}/policy", {"startup_mode": "on_demand", "idle_seconds": 5, "recovery": "manual"})
    t0 = time.time()
    st, j = tp.post(f"/api/installations/{INS1}/action", {"subject": TASS, "capability": CAP, "input": "__slow__:8"})
    took = time.time() - t0
    rec("G3", j["result"]["outcome"] == "COMPLETED" and took >= 8, "an 8 s request outlived the 5 s idle window: NOT stopped during active work",
        "%.1f s" % took)
    s = tp.until(lambda s: inst(s, INS1)["runtime"]["state"] == "STOPPED", 20, "idle stop after work")
    rec("G3", inst(s, INS1)["runtime"]["state"] == "STOPPED", "…and stopped after the idle window once the work finished")
    os.remove(W / "processor-profile" / "installations" / INS1 / "data" / "TEST-ONLY-fault-injection.json")

    # Always-running.
    tp.post(f"/api/installations/{INS1}/policy", {"startup_mode": "always_running", "idle_seconds": 30, "recovery": "manual"})
    s = tp.until(lambda s: inst(s, INS1)["runtime"]["state"] == "RUNNING", 20, "always running")
    rec("G3", inst(s, INS1)["runtime"]["state"] == "RUNNING", "ALWAYS-RUNNING: started by Processor without a request")

    # Processor crash (SIGKILL) while the feature runs → orphan cleanup on restart.
    orphan_pid = inst(s, INS1)["runtime"]["pid"]
    stop("processor", signal.SIGKILL)
    alive_after_kill = pid_alive(orphan_pid)
    link = start_processor()
    tp = TP(link)
    s = tp.until(lambda s: s["verified"], 30, "processor back")
    time.sleep(1.5)
    log = (W / "processor-profile" / "logs" / "processor.jsonl").read_text() if (W / "processor-profile" / "logs" / "processor.jsonl").exists() else ""
    gone = not pid_alive(orphan_pid)
    rec("G5", alive_after_kill and gone and "ORPHAN_STOPPED" in log, "Processor killed with the feature running → on restart the orphan proved itself with its old launch key and was stopped",
        "pid %d" % orphan_pid)
    s = tp.until(lambda s: inst(s, INS1)["runtime"]["state"] == "RUNNING", 20, "always running again")
    rec("G5", inst(s, INS1)["runtime"]["state"] == "RUNNING" and len(s["installations"]) == 1,
        "after Processor restart: same installation, always-running feature started again, no duplicate")
    tp.post(f"/api/installations/{INS1}/policy", {"startup_mode": "manual", "idle_seconds": 30, "recovery": "manual"})
    tp.post(f"/api/installations/{INS1}/stop")

    # Second copy + crash in the middle of the install transaction.
    stop("processor")
    link = start_processor(env=dict(ENV, TP_TEST_CRASH_POINT="after-register"))
    tp = TP(link)
    tp.until(lambda s: s["verified"], 30, "processor back")
    st, j = tp.post("/api/library/p1-deterministic-worker/install")
    approve(j["approval_url"])
    p = procs["processor"]
    try:
        p.wait(30)
    except subprocess.TimeoutExpired:
        pass
    crashed = p.poll() == 42
    procs.pop("processor", None)
    st, l = B.call("GET", "/api/v1/installations")
    pend = [x for x in l["installations"] if x["installation"]["state"] == "PENDING_CONFIRM"]
    rec("G5", crashed and len(pend) == 1, "TEST-ONLY crash right after Identity registered copy #2 (Processor exit 42; Identity shows 1 unconfirmed)",
        pend[0]["installation"]["installation_id"] if pend else "")
    link = start_processor()
    tp = TP(link)
    s = tp.until(lambda s: len(s["installations"]) == 2 and all(v["identity"] and v["identity"]["installation"]["state"] == "INSTALLED" for v in s["installations"]), 40, "copy 2 resumed")
    ids = [v["record"]["installation_id"] for v in s["installations"]]
    INS2 = ids[1] if len(ids) > 1 else ""
    st, l = B.call("GET", "/api/v1/installations")
    states = sorted(x["installation"]["state"] for x in l["installations"])
    rec("G5", pend and INS2 == pend[0]["installation"]["installation_id"] and states == ["INSTALLED", "INSTALLED"],
        "install resumed after the crash with the SAME installation ID; no orphan or duplicate registration", INS2)
    rec("G5", INS2 != INS1 and inst(s, INS2)["identity"]["active_grants"] == 0,
        "second copy of the same package = its own durable installation identity, zero grants, separate data folder")

    # A ticket for copy #1 sent to copy #2 is refused (wrong recipient).
    tp.post(f"/api/installations/{INS2}/start")
    tp.until(lambda s: inst(s, INS2)["runtime"]["state"] == "RUNNING", 15, "copy 2 running")
    FURL2 = json.loads((W / "processor-profile" / "installations" / INS2 / "runtime" / "endpoint.json").read_text())["url"]
    st, t = b_ticket(B, INS1, TASS, body)
    st2, j = feature_action(FURL2, t.get("ticket", ""), body)
    rec("G4", st == 200 and st2 == 401 and j.get("reason") in ("TICKET_SIGNATURE_INVALID", "TICKET_WRONG_RECIPIENT"),
        "ticket for copy #1 refused by copy #2 (signed with copy #1's key: wrong recipient)", j.get("reason", ""))
    tp.post(f"/api/installations/{INS2}/stop")
    shot("p1-11-two-installations")
    # Grok M2: Processor can't revoke actor B's grant itself, but the Admin can (emergency path).
    st, j = tp.post(f"/api/installations/{INS1}/grants/{BGID2}/revoke")
    rec("G4", st == 403 and "REVOKE_NOT_PERMITTED" in json.dumps(j), "Processor can NOT directly revoke actor B's grant", str(st))
    st, j = tp.post(f"/api/installations/{INS1}/grants/{BGID2}/admin-revoke")
    approve(j.get("approval_url", ""))
    s = tp.until(lambda s: any(x["grant"]["grant_id"] == BGID2 and x["status"] == "REVOKED" for x in ident_inst(s, INS1)["grants"]), 20, "admin revoke")
    gg = [x["grant"] for x in ident_inst(s, INS1)["grants"] if x["grant"]["grant_id"] == BGID2]
    st, t = b_ticket(B, INS1, TASS, body)
    rec("G4", gg and (gg[0].get("revoked_by") or "").startswith("admin:" + adm["admin_id"]) and st == 403 and t.get("reason") == "GRANT_REVOKED",
        "Admin emergency revocation (approved in Identity) ends B's grant; revoked_by records the Admin", gg[0].get("revoked_by", "") if gg else "")

    # Uninstall copy #2 (elevated): tombstone, grants revoked, credential gone.
    st, j = tp.post(f"/api/installations/{INS2}/uninstall", {"keep_data": False})
    approve(j["approval_url"])
    s = tp.until(lambda s: inst(s, INS2)["record"]["state"] == "UNINSTALLED", 30, "uninstalled")
    iv2 = ident_inst(s, INS2)
    rec("G5", inst(s, INS2)["record"]["state"] == "UNINSTALLED" and iv2 and iv2["installation"]["state"] == "UNINSTALLED"
        and not (W / "processor-profile" / "installations" / INS2 / "credential.json").exists(),
        "uninstall (Admin-approved): Identity tombstone kept, credential and files removed")
    # Restart everything: final persistence check.
    stop("processor")
    stop("identity")
    start_identity()
    link = start_processor()
    tp = TP(link)
    s = tp.until(lambda s: s["verified"] and len(s["identity_installations"]) == 2, 30, "final")
    st_ids = sorted((x["installation"]["installation_id"], x["installation"]["state"]) for x in s["identity_installations"])
    rec("G5", st_ids == sorted([(INS1, "INSTALLED"), (INS2, "UNINSTALLED")]) and s["admin"]["admin"]["admin_id"] == adm["admin_id"],
        "after restarting both programs: same 2 installation IDs (1 installed, 1 tombstone), same Admin", str(st_ids))
    shot("p1-12-final")
    evidence["installation_2"] = INS2
    evidence["elapsed_s"] = int(time.time() - T0)


def finish(code):
    if not args.keep:
        for n in list(procs):
            stop(n)
    if "pw" in PAGE:
        PAGE["pw"].stop()
    fails = [r for r in results if r[1] != "PASS"]
    lines = ["P1 integration report — REAL Identity, Processor and feature processes; GitHub and the feature are labelled TEST FIXTURES",
             "identity commit:  " + evidence.get("identity_commit", "?"), "processor commit: " + evidence.get("processor_commit", "?"),
             "platform: " + " ".join(os.uname()), "python: " + sys.version.split()[0], "elapsed: %ds" % (time.time() - T0), ""]
    for g, w, n, d in results:
        lines.append(f"{w} [{g}] {n}" + ("  — " + d if d else ""))
    lines += ["", "evidence: " + json.dumps(evidence, sort_keys=True, default=str), "",
              "RESULT: %s (%d checks, %d failed)" % ("PASS" if not fails and code == 0 else "FAIL", len(results), len(fails))]
    (W / "p1-integration-report.txt").write_text("\n".join(lines) + "\n")
    print("\n".join(lines[-3:]))
    print("report:", W / "p1-integration-report.txt")
    sys.exit(0 if not fails and code == 0 else 1)


if __name__ == "__main__":
    try:
        main()
        finish(0)
    except SystemExit:
        raise
    except BaseException as e:
        import traceback
        traceback.print_exc()
        rec("HARNESS", False, "harness crashed (counts as FAIL): " + repr(e)[:200])
        finish(1)
