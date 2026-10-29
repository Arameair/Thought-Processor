"""Developer entrance — acceptance test (Linux / WSL).

REAL Identity + REAL Thought Processor + the REAL Developer Utility service
(processor/developer-utility), with Thought and the P1 worker beside it.
Labelled TEST FIXTURE: GitHub (Identity's fixture). The "Admin" typing the
passphrase into Identity's own approval page is this script; a second paired
program ("actor B") tries things Processor's page never offers.

The Developer package is trusted only as one exact artifact: computed here,
required to equal the pin in run-p1.sh, then pinned in Processor
(-trust-feature-artifact) and Identity (-trust-fixture-artifact). No Identity
code or policy is changed. Processes AND listening ports are checked at the OS
level (/proc), not from Processor's own view.

  python3 -I acceptance/p1_developer_entrance.py --identity-src DIR --thought-src DIR --work DIR
"""

import argparse, hashlib, hmac, json, os, re, secrets, shutil, signal, socket, subprocess, sys, threading, time, urllib.error, urllib.request
from pathlib import Path

ap = argparse.ArgumentParser()
ap.add_argument("--identity-src", required=True)
ap.add_argument("--work", required=True)
ap.add_argument("--thought-src", required=True)
ap.add_argument("--shots", default="", help="folder for screenshots; enables the real-browser check (needs Playwright + Chromium)")
args = ap.parse_args()
W = Path(args.work).resolve()
if W.exists():
    shutil.rmtree(W)
W.mkdir(parents=True, mode=0o700)
PROC_SRC = Path(__file__).resolve().parent.parent
FEATURES = PROC_SRC / "testfeatures"


def free_port():
    with socket.socket() as s:
        s.bind(("127.0.0.1", 0))
        return s.getsockname()[1]


FX, IP, PP = free_port(), free_port(), free_port()
IDENTITY, PROCESSOR = f"http://127.0.0.1:{IP}", f"http://127.0.0.1:{PP}"
PASS = "correct horse battery staple"
CAP = "worker.test.synthetic"
FEATURE = "p1-deterministic-worker"
THOUGHT_PKG = Path(args.thought_src).resolve() / "feature"
DEV_PKG = PROC_SRC / "developer-utility"
PINS = {}
ENV = dict(os.environ, IDENTITY_LOCATOR_DIR=str(W / "locators"), IDENTITY_GITHUB_CLIENT_ID="IvFIXTURE00000001", TP_IDENTITY_DEFAULT_URL="-")
for k in ("HTTPS_PROXY", "HTTP_PROXY", "https_proxy", "http_proxy"):
    ENV.pop(k, None)
NOPROXY = urllib.request.build_opener(urllib.request.ProxyHandler({}))
results, procs, evidence = [], {}, {}
T0 = time.time()


SECRET_SHAPES = [(re.compile(r"([#&](?:t|code|key)=)[0-9A-Za-z_-]+"), r"\1<removed>"), (re.compile(r"\btk1\.[A-Za-z0-9_.-]+"), "tk1.<removed>"),
                 (re.compile(r"\b[A-HJ-NP-Z2-9]{4}-[A-HJ-NP-Z2-9]{4}\b"), "XXXX-XXXX"),
                 (re.compile(r"""(['"]?(?:secret|launch_key|passphrase|token)['"]?\s*[:=]\s*['"]?)[^'",\s}]+""", re.I), r"\1<removed>")]


def safe(detail):
    """Details go to the console and the shareable report: never let a secret through, even on failure."""
    s = str(detail)
    for rx, rep in SECRET_SHAPES:
        s = rx.sub(rep, s)
    return s[:200]


def rec(area, ok, name, detail=""):
    detail = safe(detail) if detail != "" else ""
    word = "PASS" if ok else "FAIL"
    results.append((area, word, name, detail))
    print(f"{word} [{area}] {name}" + ("  — " + str(detail) if detail != "" else ""), flush=True)
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

def start(name, cmd, env=None):
    log = os.fdopen(os.open(W / (name + ".console.txt"), os.O_WRONLY | os.O_CREAT | os.O_APPEND, 0o600), "a")
    procs[name] = subprocess.Popen(cmd, stdout=log, stderr=subprocess.STDOUT, env=env or ENV)
    return procs[name]


def stop(name, sig=signal.SIGINT, t=30):
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
                       "-test-github-web", f"http://127.0.0.1:{FX}", "-test-github-api", f"http://127.0.0.1:{FX}",
                       "-trust-fixture-artifact", PINS["thought"], "-trust-fixture-artifact", PINS["dev"]])
    assert wait_http(IDENTITY + "/api/health"), "identity did not start"


def start_processor():
    f = W / "processor-profile" / "runtime" / "operator-link.txt"
    if f.exists():
        f.unlink()
    start("processor", [str(W / "thought-processor"), "-port", str(PP), "-data", str(W / "processor-profile"), "-no-browser",
                        "-trust-feature-dir", str(FEATURES), "-trust-feature-dir", str(THOUGHT_PKG),
                        "-trust-feature-dir", str(W / "tampered"), "-trust-feature-artifact", PINS["thought"],
                        "-trust-feature-dir", str(DEV_PKG), "-trust-feature-artifact", PINS["dev"]])
    assert wait_http(PROCESSOR + "/api/health"), "processor did not start"
    for _ in range(100):
        if f.exists():
            return f.read_text().strip()
        time.sleep(0.1)
    raise SystemExit("no operator link")


def console_code(name):
    m = re.findall(r"\b([A-HJ-NP-Z2-9]{4}-[A-HJ-NP-Z2-9]{4})\b", (W / (name + ".console.txt")).read_text())
    return m[-1] if m else ""


def pid_alive(pid):
    try:
        os.kill(pid, 0)
        return True
    except OSError:
        return False


def worker_pids(ins_id):
    """OS-level truth: every live process started for this installation."""
    want = ("TF_INSTALLATION_ID=" + ins_id).encode()
    out = []
    for d in os.listdir("/proc"):
        if not d.isdigit():
            continue
        try:
            env = Path("/proc", d, "environ").read_bytes()
            stat = Path("/proc", d, "stat").read_text()
        except OSError:
            continue
        if want in env.split(b"\0") and ") Z " not in stat:  # ignore zombies
            out.append(int(d))
    return sorted(out)


# ---------------- Processor's page API ----------------

class TP:
    def __init__(self, link):
        import http.cookiejar as cj
        self.op = urllib.request.build_opener(urllib.request.ProxyHandler({}), urllib.request.HTTPCookieProcessor(cj.CookieJar()))
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


def inst(s, ins_id):
    for v in s.get("installations") or []:
        if v["record"]["installation_id"] == ins_id:
            return v
    return None


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


def approve(url):
    return admin_decide(url, {"decision": "approve", "passphrase": PASS})


class Signer:
    """identity/2 signing for actor B, implemented independently of the Go code."""

    def __init__(self, cid, secret, system_id):
        self.id, self.system_id = cid, system_id
        s = secret.encode()
        self.rk = hmac.new(s, b"identity/2 request key", hashlib.sha256).digest()

    def call(self, method, path, obj=None):
        body = json.dumps(obj).encode() if obj is not None else b""
        ts, nonce = str(int(time.time())), secrets.token_hex(12)
        mac = hmac.new(self.rk, ("identity/2 request\n%s\n%s\n%s\n%s\n%s" % (method, path, ts, nonce, hashlib.sha256(body).hexdigest())).encode(), hashlib.sha256).hexdigest()
        hdr = {"Content-Type": "application/json", "X-Identity-Time": ts, "X-Identity-Nonce": nonce, "X-Identity-Auth": mac, "X-Processor-ID": self.id}
        return http(method, IDENTITY + path, body if method == "POST" else None, hdr, timeout=10)


def pair_actor_b():
    pid = "tp_runtime_harness_b"
    http("POST", IDENTITY + "/api/v1/pair/request", {"processor_id": pid, "processor_label": "runtime harness actor B"}, {"Content-Type": "application/json"})
    time.sleep(0.5)
    st, j = http("POST", IDENTITY + "/api/v1/pair", {"code": console_code("identity"), "processor_id": pid, "processor_label": "runtime harness actor B"},
                 {"Content-Type": "application/json"})
    if st != 200:
        raise SystemExit("actor B could not pair: %s" % st)
    return Signer(pid, j["secret"], j["identity_system_id"])


def action(tp, ins, subject, text):
    st, j = tp.post(f"/api/installations/{ins}/action", {"subject": subject, "capability": CAP, "input": text})
    return j.get("result") or {"outcome": "HTTP %d" % st, "detail": safe(j.get("error", ""))}


def endpoint(ins):
    return json.loads((W / "processor-profile" / "installations" / ins / "runtime" / "endpoint.json").read_text())["url"]


# ---------------- the test ----------------

DEV = "developer-utility"
CHECK = "developer.connection.check"
WCAP = "worker.test.synthetic"


def artifact_digest(pkg):
    m = hashlib.sha256((pkg / "feature.json").read_bytes()).hexdigest()
    files = sorted(json.loads((pkg / "feature.json").read_text())["files"], key=lambda f: f["path"])
    b = "thought.feature-package/1\nmanifest %s\n" % m + "".join("%s  %s\n" % (hashlib.sha256((pkg / f["path"]).read_bytes()).hexdigest(), f["path"]) for f in files)
    return hashlib.sha256(b.encode()).hexdigest()


def listening_ports(pids):
    """OS truth: TCP ports in LISTEN state owned by any of these pids."""
    inodes = set()
    for pid in pids:
        try:
            for fd in os.listdir("/proc/%d/fd" % pid):
                try:
                    t = os.readlink("/proc/%d/fd/%s" % (pid, fd))
                except OSError:
                    continue
                if t.startswith("socket:["):
                    inodes.add(t[8:-1])
        except OSError:
            pass
    ports = []
    for tab in ("/proc/net/tcp", "/proc/net/tcp6"):
        try:
            for line in open(tab).read().splitlines()[1:]:
                f = line.split()
                if f[3] == "0A" and f[9] in inodes:
                    ports.append(int(f[1].split(":")[1], 16))
        except OSError:
            pass
    return sorted(ports)


def port_open(port):
    with socket.socket() as s:
        s.settimeout(0.5)
        return s.connect_ex(("127.0.0.1", port)) == 0


def ask(tp, ins, subject, cap, text=""):
    st, j = tp.post(f"/api/installations/{ins}/action", {"subject": subject, "capability": cap, "input": text})
    return j.get("result") or {"outcome": "HTTP %d" % st, "detail": safe(j.get("error", ""))}


def out(r):
    return ((r.get("observation") or {}).get("output")) or {}


def grant(tp, ins, subject, cap, expires=0):
    st, j = tp.post(f"/api/installations/{ins}/grants/request", {"subject": subject, "capability": cap, "expires_in_seconds": expires})
    approve(j["approval_url"])
    s = tp.until(lambda s: any(g["status"] == "ACTIVE" and g["grant"]["capability"] == cap and g["grant"]["subject_identity_id"] == subject
                               for g in inst(s, ins)["identity"]["grants"]), 20, "grant " + cap)
    g = [g for g in inst(s, ins)["identity"]["grants"] if g["status"] == "ACTIVE" and g["grant"]["capability"] == cap and g["grant"]["subject_identity_id"] == subject]
    return g[0]["grant"]["grant_id"] if g else ""


def install(tp, feature):
    st, j = tp.post(f"/api/library/{feature}/install")
    if j.get("already_installed"):
        return j["installation_id"]
    approve(j["approval_url"])
    s = tp.until(lambda s: any(v["record"]["feature_id"] == feature and v["identity"] and v["identity"]["installation"]["state"] == "INSTALLED" for v in s["installations"]), 30, "installed " + feature)
    return [v["record"]["installation_id"] for v in s["installations"] if v["record"]["feature_id"] == feature][0]


def entrance(s, ins):
    rt = inst(s, ins)["runtime"]
    return ((rt.get("health") or {}).get("status") or {}).get("entrance") or {}


LINK = {}


def ui_check(link, dv):
    """Real browser on the owner's screen: Developer feature Overview."""
    from playwright.sync_api import sync_playwright
    shots = Path(args.shots)
    shots.mkdir(parents=True, exist_ok=True)
    errors = []
    with sync_playwright() as pw:
        b = pw.chromium.launch()
        page = b.new_page(viewport={"width": 1360, "height": 1000}, color_scheme="dark")
        page.on("pageerror", lambda e: errors.append(str(e)))
        page.goto(link)
        page.wait_for_selector(f".idrow[data-sel='feature:{dv}']", timeout=20000)
        page.click(f".idrow[data-sel='feature:{dv}']")
        page.click("[data-tab=overview]")
        page.wait_for_selector("#pane-overview .runcard .evid", timeout=15000)
        card = page.inner_text("#pane-overview .runcard")
        rec("UI", all(w in card for w in ("RUNNING", "CONNECTED", "READY", "Access", "allowed for Dustin")),
            "Developer screen: RUNNING / CONNECTED / READY chips and who is allowed, on the first tab")
        page.click("#pane-overview [data-quick]")
        page.wait_for_selector("#pane-overview .runcard .result.ok >> text=COMPLETED", timeout=15000)
        page.wait_for_timeout(500)
        page.screenshot(path=str(shots / "dev-01-overview-connected-check.png"), full_page=True)
        rec("UI", "Developer entrance is CONNECTED" in page.inner_text("#pane-overview .runcard"), "one click runs the connection check and shows the real answer")
        page.click("#pane-overview [data-copyact]")
        page.wait_for_selector("#pane-overview .runcard .logbox", timeout=10000)
        txt = page.inner_text("#pane-overview .runcard .logbox")
        rec("UI", "action.completed" in txt and "contains no keys" in txt and "tk1." not in txt, "Copy activity: the redacted activity text appears (and is copied where the browser allows)")
        page.screenshot(path=str(shots / "dev-02-activity.png"), full_page=True)
        b.close()
    rec("UI", not errors, "no JavaScript errors on the Developer screen", "; ".join(errors)[:120])


def main():
    PINS["thought"] = artifact_digest(THOUGHT_PKG)
    PINS["dev"] = artifact_digest(DEV_PKG)
    launcher = (PROC_SRC.parent / "run-p1.sh").read_text()
    pinned = re.search(r"^DEV_ARTIFACT=([0-9a-f]{64})$", launcher, re.M)
    ids = subprocess.check_output(["git", "-C", args.identity_src, "rev-parse", "HEAD"], text=True).strip()
    tps = subprocess.check_output(["git", "-C", str(PROC_SRC), "rev-parse", "HEAD"], text=True).strip()
    dirty = subprocess.run(["git", "-C", str(PROC_SRC), "diff", "--quiet", "HEAD", "--", "."]).returncode != 0
    evidence.update(identity_commit=ids, processor_commit=tps + (" (+ uncommitted changes)" if dirty else ""), developer_artifact=PINS["dev"])
    rec("PIN", bool(pinned) and pinned.group(1) == PINS["dev"], "the Developer package in this checkout is exactly the artifact pinned in run-p1.sh", PINS["dev"][:16] + "…")
    # a tampered copy in a trusted folder
    bad = W / "tampered" / "developer-utility-tampered"
    bad.mkdir(parents=True)
    m = json.loads((DEV_PKG / "feature.json").read_text())
    m["feature_id"] = "developer-utility-tampered"
    code = (DEV_PKG / "utility_agent.py").read_bytes().replace(b'"cannot": [', b'"cannot": ["(tampered)", ')
    (bad / "utility_agent.py").write_bytes(code)
    m["files"][0]["sha256"] = hashlib.sha256(code).hexdigest()
    (bad / "feature.json").write_text(json.dumps(m, indent=2))

    subprocess.check_call(["go", "build", "-o", str(W / "identity"), "."], cwd=args.identity_src)
    subprocess.check_call(["go", "build", "-o", str(W / "thought-processor"), "."], cwd=PROC_SRC)
    (W / "asset").write_bytes(b"TEST FIXTURE\n")
    start("fixture", [sys.executable, "-I", str(Path(args.identity_src) / "scripts" / "fixture_github_engine.py"), str(FX), str(W / "asset"), "linux-amd64"])
    assert wait_http(f"http://127.0.0.1:{FX}/fixture/state")
    NOPROXY.open(urllib.request.Request(f"http://127.0.0.1:{FX}/fixture/login?user=Arameair", data=b"", method="POST"), timeout=5).read()
    start_identity()
    LINK["url"] = start_processor()
    tp = TP(LINK["url"])
    tp.until(lambda s: s["connection"]["state"] == "FOUND_NOT_CONNECTED", 20, "Identity found")
    tp.post("/api/connect/request", {"url": IDENTITY})
    time.sleep(0.5)
    tp.post("/api/connect/code", {"code": console_code("identity")})
    tp.until(lambda s: s["verified"], 20, "connected")
    tp.post("/api/github/connect")
    tp.until(lambda s: (s.get("github") or {}).get("state") == "VERIFIED", 40, "GitHub verified")
    st, j = tp.post("/api/admin/enroll")
    admin_decide(j["approval_url"], {"decision": "approve", "passphrase": PASS, "passphrase_confirm": PASS, "acknowledge": True})
    s = tp.until(lambda s: s["admin"]["state"] == "ENROLLED", 15, "enrolled")
    rec("SETUP", s["admin"]["state"] == "ENROLLED", "paired, GitHub (fixture) verified, Admin established in Identity's page")
    tp.post("/api/identities", {"name": "Dustin", "kind": "person"})
    tp.post("/api/identities", {"name": "Other", "kind": "person"})
    s = tp.state()
    ME = [e for e in s["identities"] if e["name"] == "Dustin"][0]["id"]
    OTHER = [e for e in s["identities"] if e["name"] == "Other"][0]["id"]
    B = pair_actor_b()

    # ---- discoverable, pinned; unknown/tampered denied ----
    lib = {x.get("feature_id"): x for x in s["library"]}
    d = lib.get(DEV) or {}
    rec("TRUST", d and not d.get("blocked") and "operator pin" in (d.get("trust_basis") or "") and [c["id"] for c in d.get("capabilities") or []] == [CHECK],
        "Developer Utility is discoverable in Processor's library as a reviewed, pinned package with ONE capability", d.get("trust_basis") or d.get("blocked") or "missing")
    t = lib.get("developer-utility-tampered") or {}
    st, j = tp.post("/api/library/developer-utility-tampered/install")
    rec("TRUST", bool(t.get("blocked")) and st == 409, "a changed copy in a trusted folder is BLOCKED and can't be installed", (t.get("blocked") or "")[:50])
    st, j = B.call("POST", "/api/v1/installations/requests", {"feature_id": DEV, "name": "Developer Utility", "version": "0.1.0", "api": "thought.feature-api/1",
        "publisher_claim": "x", "source_kind": "local-trusted-package", "source_ref": "x", "trust_class": "TRUSTED_LOCAL_TEST_FIXTURE",
        "artifact_sha256": secrets.token_hex(32), "manifest_sha256": secrets.token_hex(32), "install_key": secrets.token_hex(16),
        "capabilities": [{"id": CHECK, "description": "x", "data_class": "x"}]})
    rec("TRUST", st == 403, "Identity itself refuses to register an unknown artifact claiming the Developer name (independent of Processor)", str(st))
    st, j = B.call("POST", "/api/v1/installations/requests", {"feature_id": DEV, "name": "Developer Utility", "version": "0.1.0", "api": "thought.feature-api/1",
        "publisher_claim": "x", "source_kind": "local-trusted-package", "source_ref": "x", "trust_class": "OWNER_REVIEW_REQUIRED",
        "artifact_sha256": PINS["dev"], "manifest_sha256": secrets.token_hex(32), "install_key": secrets.token_hex(16),
        "capabilities": [{"id": CHECK, "description": "x", "data_class": "x"}]})
    rec("TRUST", st == 403, "…and the team's original label (OWNER_REVIEW_REQUIRED) is still refused by Identity: no policy was loosened", str(st))

    # ---- install needs the Admin ----
    st, j = tp.post(f"/api/library/{DEV}/install")
    st2, j2 = admin_decide(j["approval_url"], {"decision": "approve", "passphrase": "not the passphrase"})
    s = tp.state()
    rec("INSTALL", st2 == 401 and not [v for v in s["installations"] if v["record"]["feature_id"] == DEV], "wrong Admin passphrase: refused, nothing installed", str(st2))
    tp.post("/api/install-tx/%s/cancel" % j["install_key"])
    st, j = tp.post(f"/api/library/{DEV}/install")
    st2, j2 = admin_decide(j["approval_url"], {"decision": "deny"})
    time.sleep(3)
    s = tp.state()
    rec("INSTALL", st2 == 200 and not [v for v in s["installations"] if v["record"]["feature_id"] == DEV] and not s["install_txs"],
        "Admin DENIES the install: nothing installed, nothing left waiting")
    DV = install(tp, DEV)
    s = tp.state()
    v = inst(s, DV)
    rec("INSTALL", v["files"] == "OK" and v["identity"]["installation"]["artifact_sha256"] == PINS["dev"] and v["identity"]["active_grants"] == 0,
        "Admin APPROVES: one installation of the exact pinned artifact, zero grants", DV)
    outs = []
    th = [threading.Thread(target=lambda: outs.append(tp.post(f"/api/library/{DEV}/install"))) for _ in range(5)]
    [x.start() for x in th]
    [x.join() for x in th]
    s = tp.state()
    rec("INSTALL", all(o[1].get("installation_id") == DV for o in outs) and len([x for x in s["identity_installations"] if x["installation"]["feature_id"] == DEV]) == 1,
        "repeated install (5 at once) returns the same installation; still one in Identity")

    # ---- connection denied before any grant ----
    r = ask(tp, DV, ME, CHECK)
    rec("AUTHZ", r["outcome"] == "DENIED_BY_IDENTITY" and "NO_GRANT" in r["detail"] and worker_pids(DV) == [],
        "unauthorized Developer connection check: DENIED by Identity, nothing started", r["detail"][:50])

    # ---- Start: one process, one listener, CONNECTED/READY on real evidence ----
    tp.post(f"/api/installations/{DV}/start")
    s = tp.until(lambda s: inst(s, DV)["runtime"]["state"] == "RUNNING" and entrance(s, DV).get("ready"), 25, "dev ready")
    rt = inst(s, DV)["runtime"]
    pid = rt["pid"]
    ports = listening_ports([pid])
    ep = urllib.parse.urlparse(endpoint(DV)).port
    rec("RUN", worker_pids(DV) == [pid] and ports == [ep], "Start: exactly ONE Developer process with ONE loopback listener (OS check)", "pid %d port %s" % (pid, ports))
    e = entrance(s, DV)
    a = (rt["health"]["status"] or {}).get("authority") or {}
    rec("RUN", e.get("running") and e.get("connected") and e.get("ready") and a.get("state") == "CONNECTED",
        "RUNNING + CONNECTED + READY, from the service's launch-signed status and Identity's signed answer", (a.get("detail") or "")[:70])
    for _ in range(3):
        tp.post(f"/api/installations/{DV}/start")
    rec("RUN", worker_pids(DV) == [pid] and listening_ports([pid]) == [ep], "Start pressed again 3×: still one process, one listener")
    st, j = tp.post(f"/api/installations/{DV}/self-test")
    stt = j.get("self_test") or {}
    rec("RUN", stt.get("result") == "PASS" and all(c["ok"] for c in (stt.get("output") or {}).get("checks") or [{"ok": False}]),
        "self-test PASS: Processor link and a fresh signed Identity answer both verified")
    st, _ = http("POST", endpoint(DV) + "/v1/actions/" + CHECK, json.dumps({"request_id": "x", "input": ""}).encode(), {"Content-Type": "application/json"})
    rec("AUTHZ", st == 401, "calling the running service directly without an Identity ticket: refused", str(st))
    st, t = B.call("POST", "/api/v1/tickets", {"destination_installation_id": DV, "subject_identity_id": ME, "capability": CHECK, "request_sha256": "0" * 64})
    rec("AUTHZ", st == 403 and t.get("reason") == "NO_GRANT", "another paired program gets no ticket for the Developer entrance", t.get("reason", ""))

    # ---- approved connection: one bounded operation ----
    G = grant(tp, DV, ME, CHECK)
    r = ask(tp, DV, ME, CHECK)
    o = out(r)
    il = o.get("identity_link") or {}
    held = [(g["capability"], g["status"]) for g in (il.get("this_installation") or {}).get("grants") or []]
    rec("CONNECT", r["outcome"] == "COMPLETED" and il.get("state") == "CONNECTED" and (CHECK, "ACTIVE") in held
        and (o.get("this_request") or {}).get("for_identity") == ME and o.get("can") == [CHECK],
        "approved: connection check COMPLETED — Identity link CONNECTED (signed, fresh), shows the one grant it holds and who asked", o.get("headline", r["detail"][:60]))
    rec("CONNECT", "run commands" in " ".join(o.get("cannot") or []), "the report states what the entrance can NOT do (no commands, files or other services)")
    r = ask(tp, DV, OTHER, CHECK)
    rec("CONNECT", r["outcome"] == "DENIED_BY_IDENTITY", "the grant is for Dustin only: Other is still denied")
    r = ask(tp, DV, ME, CHECK, "rm -rf /")
    rec("CONNECT", r["outcome"] == "ERROR" and "BAD_INPUT" in r["detail"], "input is refused (the operation takes none): no way to smuggle a command", r["detail"][:60])
    st, j = tp.get(f"/api/installations/{DV}/activity")
    act_text = j.get("text", "")
    rec("UI", st == 200 and "action.completed" in act_text and "action.denied_by_identity" in act_text and "feature.running" in act_text,
        "copyable activity log for the Developer entrance has its real starts, decisions and requests")

    # ---- revocation, expiry ----
    tp.post(f"/api/installations/{DV}/grants/{G}/revoke")
    r = ask(tp, DV, ME, CHECK)
    rec("REVOKE", r["outcome"] == "DENIED_BY_IDENTITY" and "GRANT_REVOKED" in r["detail"], "revoked: the next connection check is DENIED", r["detail"][:50])
    grant(tp, DV, ME, CHECK, expires=5)
    r1 = ask(tp, DV, ME, CHECK)
    time.sleep(6.5)
    r2 = ask(tp, DV, ME, CHECK)
    rec("REVOKE", r1["outcome"] == "COMPLETED" and r2["outcome"] == "DENIED_BY_IDENTITY" and "EXPIRED" in r2["detail"],
        "expiring grant: allowed, then DENIED after it expires", r2["detail"][:50])

    # ---- Identity unavailable ----
    stop("identity")
    tp.until(lambda s: not s["verified"], 30, "authority unavailable")
    r = ask(tp, DV, ME, CHECK)
    rec("ERRORS", r["outcome"] == "AUTHORITY_UNAVAILABLE", "Identity down: connection check fails closed (AUTHORITY UNAVAILABLE)")
    s = tp.until(lambda s: entrance(s, DV).get("connected") is False, 25, "service notices")
    st_ = ((inst(s, DV)["runtime"].get("health") or {}).get("status") or {})
    rec("ERRORS", entrance(s, DV).get("connected") is False and entrance(s, DV).get("ready") is False and (st_.get("authority") or {}).get("state") in ("UNAVAILABLE", "STALE"),
        "the service itself reports NOT CONNECTED / NOT READY (no green state without Identity)", (st_.get("authority") or {}).get("state", ""))
    start_identity()
    s = tp.until(lambda s: s["verified"] and inst(s, DV)["identity"] is not None and entrance(s, DV).get("connected"), 40, "connected again")
    rec("ERRORS", entrance(s, DV).get("connected") is True, "Identity back: CONNECTED again on a fresh signed answer")

    # ---- Thought and the worker are unaffected ----
    WK = install(tp, "p1-deterministic-worker")
    TH = install(tp, "thought")
    tp.post(f"/api/installations/{WK}/start")
    tp.post(f"/api/installations/{TH}/start")
    s = tp.until(lambda s: inst(s, WK)["runtime"]["state"] == "RUNNING" and inst(s, TH)["runtime"]["state"] == "RUNNING", 20, "others running")
    r = ask(tp, WK, ME, WCAP, "hi")
    rec("OTHERS", r["outcome"] == "DENIED_BY_IDENTITY", "a Developer grant gives nothing on the worker (each service has its own grants)")
    grant(tp, WK, ME, WCAP)
    r = ask(tp, WK, ME, WCAP, "hi")
    st, j = tp.post(f"/api/installations/{TH}/self-test")
    rec("OTHERS", r["outcome"] == "COMPLETED" and (j.get("self_test") or {}).get("result") == "PASS" and len(worker_pids(WK)) == 1 and len(worker_pids(TH)) == 1,
        "worker request COMPLETED and Thought self-test PASS beside the Developer entrance; one process each")

    # A live grant from here on: the screen shows it, and a refusal after Stop can only mean "not running".
    grant(tp, DV, ME, CHECK)

    # ---- the owner's screen (real browser) ----
    if args.shots:
        ui_check(LINK["url"], DV)

    # ---- Stop: process and listener gone ----
    pid = inst(tp.state(), DV)["runtime"]["pid"]
    ep = urllib.parse.urlparse(endpoint(DV)).port
    tp.post(f"/api/installations/{DV}/stop")
    s = tp.until(lambda s: inst(s, DV)["runtime"]["state"] == "STOPPED", 20, "stopped")
    time.sleep(0.3)
    rec("STOP", inst(s, DV)["runtime"]["state"] == "STOPPED" and not pid_alive(pid) and worker_pids(DV) == [] and not port_open(ep),
        "Stop: STOPPED, the Developer process is gone and its port no longer listens (OS check)", "port %d" % ep)
    rec("STOP", len(worker_pids(WK)) == 1 and len(worker_pids(TH)) == 1, "stopping the Developer entrance left the worker and Thought running")
    r = ask(tp, DV, ME, CHECK)
    rec("STOP", r["outcome"] == "NOT_RUNNING", "after Stop, even with a live grant, the request is refused as NOT RUNNING (nothing auto-starts)", r["outcome"])

    # ---- restart keeps the installation ----
    stop("processor")
    time.sleep(0.5)
    rec("RESTART", worker_pids(WK) == [] and worker_pids(TH) == [] and worker_pids(DV) == [], "Processor exit stops every service it started")
    stop("identity")
    start_identity()
    tp = TP(start_processor())
    s = tp.until(lambda s: s["verified"] and inst(s, DV) and inst(s, DV)["identity"] is not None, 40, "back")
    tp.post(f"/api/installations/{DV}/start")
    s = tp.until(lambda s: entrance(s, DV).get("ready"), 25, "ready again")
    rec("RESTART", inst(s, DV)["record"]["installation_id"] == DV and len([x for x in s["identity_installations"] if x["installation"]["feature_id"] == DEV]) == 1
        and entrance(s, DV).get("connected"), "after restarting everything: same Developer installation, CONNECTED and READY again", DV)
    tp.post(f"/api/installations/{DV}/stop")
    tp.until(lambda s: inst(s, DV)["runtime"]["state"] == "STOPPED", 20, "final stop")

    # ---- no credentials in logs or in the copyable activity ----
    cred = json.loads((W / "processor-profile" / "installations" / DV / "credential.json").read_text())
    texts = act_text + "".join(p.read_text(errors="replace") for p in [W / "processor-profile" / "logs" / "processor.jsonl"] +
                               list((W / "processor-profile" / "installations" / DV / "logs").glob("*")))
    leaks = [n for n, x in (("credential secret", cred["secret"]), ("pairing code", console_code("identity"))) if x and x in texts]
    leaks += ["approval token"] if re.search(r"[#&]t=[0-9a-f]{16,}", texts) else []
    leaks += ["ticket"] if re.search(r"tk1\.[A-Za-z0-9_-]{8,}", texts) else []
    rec("LOGS", not leaks, "no credential secret, pairing code, approval token or ticket in logs or the copied activity", ", ".join(leaks) or "clean")


if __name__ == "__main__":
    import urllib.parse
    try:
        main()
    except Exception as e:
        import traceback
        traceback.print_exc()
        rec("HARNESS", False, "the test script itself failed", type(e).__name__ + ": " + str(e)[:160])
    finally:
        for n in list(procs):
            stop(n)
    failed = [r for r in results if r[1] != "PASS"]
    evidence["elapsed_s"] = int(time.time() - T0)
    lines = [f"Developer entrance acceptance — {time.strftime('%Y-%m-%d %H:%M:%S %Z')}", "environment: " + " ".join(os.uname()[0:3:2]),
             "evidence: " + json.dumps(evidence, sort_keys=True), ""] + [f"{w} [{a}] {n}" + (f"  — {d}" if d != "" else "") for a, w, n, d in results]
    lines.append(f"\nRESULT: {'PASS' if not failed else 'FAIL'} ({len(results)} checks, {len(failed)} failed)")
    (W / "p1-developer-entrance-report.txt").write_text("\n".join(lines) + "\n")
    print(lines[-1].strip())
    print("report:", W / "p1-developer-entrance-report.txt")
    sys.exit(0 if not failed else 1)
