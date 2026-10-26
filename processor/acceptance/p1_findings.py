"""Grok QA findings (Issue #22) — before/after check against REAL processes.

Each line asks "is the SECURE behaviour observed?". On the reviewed P1
baseline the M/L lines are expected to FAIL (= Grok's finding reproduced);
on the cleanup branches they must PASS. Uses free ports chosen by the OS,
private (0700) work folders, and never touches other running programs.

  python3 -I acceptance/p1_findings.py --identity-src DIR --processor-src DIR --work DIR [--label before|after]
"""
import argparse, hashlib, hmac, json, os, re, secrets, shutil, signal, socket, stat, subprocess, sys, time, urllib.error, urllib.request
from pathlib import Path

os.umask(0o077)
ap = argparse.ArgumentParser()
ap.add_argument("--identity-src", required=True)
ap.add_argument("--processor-src", required=True)
ap.add_argument("--work", required=True)
ap.add_argument("--label", default="run")
args = ap.parse_args()
W = Path(args.work).resolve()
if W.exists():
    shutil.rmtree(W)
W.mkdir(parents=True, mode=0o700)
os.chmod(W, 0o700)
os.umask(0o022)  # programs under test run with an ordinary umask (the L3 check depends on it)
PSRC = Path(args.processor_src).resolve()
NOPROXY = urllib.request.build_opener(urllib.request.ProxyHandler({}))
PASS = "correct horse battery staple"
CAP = "worker.test.synthetic"
results, procs = [], {}


def free_port():
    s = socket.socket()
    s.bind(("127.0.0.1", 0))
    p = s.getsockname()[1]
    s.close()
    return p


FX, IP, PP = free_port(), free_port(), free_port()
IDENTITY, PROCESSOR = f"http://127.0.0.1:{IP}", f"http://127.0.0.1:{PP}"
ENV = {k: v for k, v in os.environ.items() if k.lower() not in ("https_proxy", "http_proxy")}
ENV.update(IDENTITY_LOCATOR_DIR=str(W / "locators"), IDENTITY_GITHUB_CLIENT_ID="IvFIXTURE00000001",
           TP_IDENTITY_DEFAULT_URL="-", HOME=str(W / "home"))


def rec(fid, ok, name, detail=""):
    results.append((fid, "PASS" if ok else "FAIL", name, detail))
    print(("PASS" if ok else "FAIL") + f" [{fid}] {name}" + ("  — " + detail if detail else ""), flush=True)


def http(method, url, body=None, headers=None, raw=False):
    data = body if isinstance(body, (bytes, type(None))) else json.dumps(body).encode()
    r = urllib.request.Request(url, data=data if method == "POST" else None, method=method)
    for k, v in (headers or {}).items():
        r.add_header(k, v)
    try:
        resp = NOPROXY.open(r, timeout=30)
        st, b, h = resp.status, resp.read(), resp.headers
    except urllib.error.HTTPError as e:
        st, b, h = e.code, e.read(), e.headers
    if raw:
        return st, b, h
    try:
        return st, json.loads(b or b"{}")
    except Exception:
        return st, {"_raw": b[:200].decode(errors="replace")}


def start(name, cmd):
    f = os.fdopen(os.open(W / (name + ".console.txt"), os.O_WRONLY | os.O_CREAT | os.O_APPEND, 0o600), "a")
    procs[name] = subprocess.Popen(cmd, stdout=f, stderr=subprocess.STDOUT, env=ENV)


def stop(name):
    p = procs.pop(name, None)
    if p and p.poll() is None:
        p.send_signal(signal.SIGINT)
        try:
            p.wait(20)
        except subprocess.TimeoutExpired:
            p.kill()


def wait_http(u, t=20):
    end = time.time() + t
    while time.time() < end:
        try:
            NOPROXY.open(u, timeout=1)
            return True
        except Exception:
            time.sleep(0.2)
    return False


def console_code():
    m = re.findall(r"\b([A-HJ-NP-Z2-9]{4}-[A-HJ-NP-Z2-9]{4})\b", (W / "identity.console.txt").read_text())
    return m[-1]


class Signer:
    def __init__(self, cid, secret, system_id):
        s = secret.encode()
        self.id, self.system_id = cid, system_id
        self.rk = hmac.new(s, b"identity/2 request key", hashlib.sha256).digest()

    def call(self, method, path, obj=None):
        body = json.dumps(obj).encode() if obj is not None else b""
        ts, nonce = str(int(time.time())), secrets.token_hex(12)
        mac = hmac.new(self.rk, ("identity/2 request\n%s\n%s\n%s\n%s\n%s" % (method, path, ts, nonce, hashlib.sha256(body).hexdigest())).encode(), hashlib.sha256).hexdigest()
        try:
            return http(method, IDENTITY + path, body if method == "POST" else None, {"Content-Type": "application/json", "X-Identity-Time": ts,
                        "X-Identity-Nonce": nonce, "X-Identity-Auth": mac, "X-Processor-ID": self.id})
        except Exception as e:
            return 0, {"_error": str(e)}


def pair(pid, label):
    http("POST", IDENTITY + "/api/v1/pair/request", {"processor_id": pid, "processor_label": label}, {"Content-Type": "application/json"})
    time.sleep(0.4)
    st, j = http("POST", IDENTITY + "/api/v1/pair", {"code": console_code(), "processor_id": pid, "processor_label": label}, {"Content-Type": "application/json"})
    assert st == 200, (st, j)
    time.sleep(3.2)  # Identity spaces pairing requests 3 s apart
    return Signer(pid, j["secret"], j["identity_system_id"])


def approval_session(url):
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
    return st, j, (lambda body: post("/api/approval/decide", body, j.get("csrf")))


def approve(url, extra=None):
    st, j, decide = approval_session(url)
    body = {"decision": "approve", "passphrase": PASS}
    body.update(extra or {})
    return decide(body)


class TP:
    def __init__(self, link):
        import http.cookiejar as cj
        self.op = urllib.request.build_opener(urllib.request.ProxyHandler({}), urllib.request.HTTPCookieProcessor(cj.CookieJar()))
        self.csrf = None
        self.csrf = self.post("/api/session", {"code": re.search(r"#code=([0-9a-f]+)", link).group(1)})[1]["csrf"]

    def post(self, path, body=None):
        r = urllib.request.Request(PROCESSOR + path, data=json.dumps(body or {}).encode(), method="POST")
        r.add_header("Content-Type", "application/json")
        r.add_header("Origin", PROCESSOR)
        if self.csrf:
            r.add_header("X-TP-CSRF", self.csrf)
        try:
            with self.op.open(r, timeout=60) as resp:
                return resp.status, json.loads(resp.read() or b"{}")
        except urllib.error.HTTPError as e:
            return e.code, json.loads(e.read() or b"{}")

    def state(self):
        with self.op.open(PROCESSOR + "/api/state", timeout=30) as r:
            return json.loads(r.read())

    def until(self, pred, t=30):
        end = time.time() + t
        while time.time() < end:
            try:
                s = self.state()
                if pred(s):
                    return s
            except Exception:
                pass
            time.sleep(0.4)
        return self.state()


def main():
    subprocess.check_call(["go", "build", "-o", str(W / "identity"), "."], cwd=args.identity_src)
    subprocess.check_call(["go", "build", "-o", str(W / "tp"), "."], cwd=PSRC / "processor")
    # A folder the operator trusts, holding NEW code that only CLAIMS to be the fixture (Grok Q63).
    fake = W / "trusted-folder" / "fake-fixture"
    shutil.copytree(PSRC / "processor" / "testfeatures" / "deterministic-worker", fake)
    (fake / "worker.py").write_text((fake / "worker.py").read_text() + "\n# NEW CODE: not the reviewed fixture\n")
    m = json.loads((fake / "feature.json").read_text())
    m["feature_id"], m["name"] = "p1-self-labelled", "Self-labelled fixture (NOT reviewed)"
    m["files"][0]["sha256"] = hashlib.sha256((fake / "worker.py").read_bytes()).hexdigest()
    (fake / "feature.json").write_text(json.dumps(m, indent=2))
    (W / "asset").write_bytes(b"x")
    start("fixture", [sys.executable, "-I", str(Path(args.identity_src) / "scripts" / "fixture_github_engine.py"), str(FX), str(W / "asset"), "linux-amd64"])
    wait_http(f"http://127.0.0.1:{FX}/fixture/state")
    start("identity", [str(W / "identity"), "-port", str(IP), "-data", str(W / "id-profile"), "-test-github-web", f"http://127.0.0.1:{FX}", "-test-github-api", f"http://127.0.0.1:{FX}"])
    assert wait_http(IDENTITY + "/api/health")
    start("processor", [str(W / "tp"), "-port", str(PP), "-data", str(W / "tp-profile"), "-no-browser", "-test-feature-faults",
                        "-trust-feature-dir", str(PSRC / "processor" / "testfeatures"), "-trust-feature-dir", str(W / "trusted-folder")])
    if not wait_http(PROCESSOR + "/api/health"):  # the baseline has no -test-feature-faults flag
        procs.pop("processor")
        start("processor", [str(W / "tp"), "-port", str(PP), "-data", str(W / "tp-profile"), "-no-browser",
                            "-trust-feature-dir", str(PSRC / "processor" / "testfeatures"), "-trust-feature-dir", str(W / "trusted-folder")])
        assert wait_http(PROCESSOR + "/api/health")
    for _ in range(100):
        f = W / "tp-profile" / "runtime" / "operator-link.txt"
        if f.exists():
            break
        time.sleep(0.1)
    tp = TP(f.read_text().strip())
    tp.post("/api/connect/request", {"url": IDENTITY})
    time.sleep(0.4)
    tp.post("/api/connect/code", {"code": console_code()})
    s = tp.until(lambda s: s["verified"])
    pid = s["processor"]["processor_id"]
    time.sleep(3.2)
    tp.post("/api/github/connect")
    tp.until(lambda s: (s.get("github") or {}).get("state") == "VERIFIED", 40)
    st, j = tp.post("/api/admin/enroll")
    approve(j["approval_url"], {"passphrase_confirm": PASS, "acknowledge": True})
    tp.until(lambda s: (s.get("admin") or {}).get("state") == "ENROLLED")
    C = pair("tp_findings_c", "Findings program C")
    D = pair("tp_findings_d", "Findings program D (unexpected)")
    st, e = C.call("POST", "/api/v1/identities", {"name": "Tass", "kind": "companion"})
    TASS = e["id"]

    # ---- M1: self-declared trust ----
    lib = {x.get("feature_id"): x for x in tp.state()["library"]}
    fakeitem = lib.get("p1-self-labelled") or {}
    rec("M1", bool(fakeitem.get("blocked")), "self-labelled 'fixture' with new code is NOT offered as installable",
        "offered as installable" if not fakeitem.get("blocked") else fakeitem["blocked"][:70])
    st, j = tp.post("/api/library/p1-self-labelled/install")
    rec("M1", st != 200, "Processor refuses to start installing it", "%d %s" % (st, (j.get("error") or "")[:60]))
    if st == 200 and j.get("install_key"):
        tp.post("/api/install-tx/%s/cancel" % j["install_key"])
    st, j = C.call("POST", "/api/v1/installations/requests", {"feature_id": "p1-forged", "name": "Forged", "version": "0.1.0",
        "api": "thought.feature-api/1", "publisher_claim": "x", "source_kind": "local-trusted-package", "source_ref": "x",
        "trust_class": "TRUSTED_LOCAL_TEST_FIXTURE", "artifact_sha256": secrets.token_hex(32), "manifest_sha256": secrets.token_hex(32),
        "install_key": secrets.token_hex(16), "capabilities": [{"id": CAP, "description": "x", "data_class": "SYNTHETIC"}]})
    rec("M1", st == 403, "Identity itself refuses an install request for an artifact digest it has not pinned (even if a program claims fixture trust)",
        "%d %s" % (st, (j.get("message") or j.get("error") or "")[:70]))
    # the real pinned fixture still installs
    st, j = tp.post("/api/library/p1-deterministic-worker/install")
    approve(j["approval_url"])
    s = tp.until(lambda s: any(v["identity"] and v["identity"]["installation"]["state"] == "INSTALLED" for v in s["installations"]))
    INS = s["installations"][0]["record"]["installation_id"]
    rec("BASE", bool(INS), "the real reviewed fixture still installs", INS)
    tp.post(f"/api/installations/{INS}/start")
    tp.until(lambda s: s["installations"][0]["runtime"]["state"] == "RUNNING", 20)
    FURL = json.loads((W / "tp-profile" / "installations" / INS / "runtime" / "endpoint.json").read_text())["url"]

    # ---- M3: approval page clarity + default expiry ----
    st, j = D.call("POST", "/api/v1/grants/requests", {"destination_installation_id": INS, "subject_identity_id": TASS, "capability": CAP})
    st2, view, decide = approval_session(j["approval_url"])
    vs = json.dumps(view)
    rec("M3", "Findings program D (unexpected)" in vs, "approval page shows the requester's verified paired label")
    disp = view.get("display") or {}
    rec("M3", bool((disp.get("requester") or {}).get("warning")), "approval page warns that the requester is not the primary Thought Processor")
    decide({"decision": "approve", "passphrase": PASS})
    st, g = D.call("POST", "/api/v1/grants", {"approval_id": j["approval"]["id"]})
    exp = (g.get("grant") or {}).get("expires_at")
    rec("M3", bool(exp), "a grant requested without an expiry gets a finite default expiry", str(exp))
    st, j = C.call("POST", "/api/v1/grants/requests", {"destination_installation_id": INS, "subject_identity_id": TASS, "capability": CAP, "until_revoked": True})
    if st == 202:
        st2, d2 = approve(j["approval_url"])
        st3, g3 = C.call("POST", "/api/v1/grants", {"approval_id": j["approval"]["id"]})
        rec("M3", st2 == 400 and st3 != 201, "'until revoked' is refused unless the Admin deliberately confirms it on the page", "decide %d commit %d" % (st2, st3))
    else:
        rec("M3", False, "'until revoked' is refused unless the Admin deliberately confirms it on the page", "request %d (no until_revoked support)" % st)

    # ---- M2: cross-principal revocation ----
    st, j = C.call("POST", "/api/v1/grants/requests", {"destination_installation_id": INS, "subject_identity_id": TASS, "capability": CAP, "expires_in_seconds": 3600})
    approve(j["approval_url"])
    st, g = C.call("POST", "/api/v1/grants", {"approval_id": j["approval"]["id"]})
    CG = g["grant"]["grant_id"]
    st, j = D.call("POST", "/api/v1/grants/%s/revoke" % CG, {"reason": "D tries"})
    rec("M2", st == 403, "unrelated paired program D can NOT revoke program C's grant", "HTTP %d" % st)
    st, j = C.call("GET", "/api/v1/grants?installation=" + INS)
    still = [x for x in j.get("grants", []) if x["grant"]["grant_id"] == CG]
    rec("M2", bool(still) and still[0]["status"] == "ACTIVE", "C's grant is still ACTIVE after D's attempt")

    # one completed request (creates the observation file checked under L3)
    okbody = json.dumps({"request_id": "ok1", "input": "fine"}).encode()
    st, t = C.call("POST", "/api/v1/tickets", {"destination_installation_id": INS, "subject_identity_id": TASS, "capability": CAP,
                                               "request_sha256": hashlib.sha256(okbody).hexdigest()})
    st, r = http("POST", FURL + "/v1/actions/" + CAP, okbody, {"Content-Type": "application/json", "X-TF-Ticket": t.get("ticket", "")})
    rec("BASE", st == 200, "C's own valid request is allowed by the feature", str(st))

    # ---- L1: ticket spent during an outage ----
    body = json.dumps({"request_id": "l1", "input": "outage"}).encode()
    st, t = C.call("POST", "/api/v1/tickets", {"destination_installation_id": INS, "subject_identity_id": TASS, "capability": CAP,
                                               "request_sha256": hashlib.sha256(body).hexdigest()})
    stop("identity")
    st1, r1 = http("POST", FURL + "/v1/actions/" + CAP, body, {"Content-Type": "application/json", "X-TF-Ticket": t["ticket"]})
    start("identity", [str(W / "identity"), "-port", str(IP), "-data", str(W / "id-profile"), "-test-github-web", f"http://127.0.0.1:{FX}", "-test-github-api", f"http://127.0.0.1:{FX}"])
    wait_http(IDENTITY + "/api/health")
    st2, r2 = http("POST", FURL + "/v1/actions/" + CAP, body, {"Content-Type": "application/json", "X-TF-Ticket": t["ticket"]})
    rec("L1", st1 == 503 and st2 == 409 and r2.get("error") != "TICKET_REPLAYED" and "outage" in json.dumps(r2).lower(),
        "a ticket spent during an Identity outage is reported as such (still refused), not as a malicious replay", "%d %s / %d %s" % (st1, r1.get("error"), st2, r2.get("error")))

    # ---- L5: fault hook needs an explicit test mode ----
    st, j = C.call("POST", "/api/v1/grants/requests", {"destination_installation_id": INS, "subject_identity_id": TASS, "capability": CAP, "expires_in_seconds": 3600})
    approve(j["approval_url"])
    C.call("POST", "/api/v1/grants", {"approval_id": j["approval"]["id"]})
    data = W / "tp-profile" / "installations" / INS / "data"
    (data / "TEST-ONLY-fault-injection.json").write_text(json.dumps({"crash": True}))
    # restart the feature WITHOUT test mode: a dropped file must do nothing
    stop("processor")
    start("processor", [str(W / "tp"), "-port", str(PP), "-data", str(W / "tp-profile"), "-no-browser",
                        "-trust-feature-dir", str(PSRC / "processor" / "testfeatures")])
    wait_http(PROCESSOR + "/api/health")
    time.sleep(1)
    tp = TP((W / "tp-profile" / "runtime" / "operator-link.txt").read_text().strip())
    tp.until(lambda s: s["verified"], 30)
    tp.post(f"/api/installations/{INS}/start")
    tp.until(lambda s: s["installations"][0]["runtime"]["state"] == "RUNNING", 20)
    FURL = json.loads((W / "tp-profile" / "installations" / INS / "runtime" / "endpoint.json").read_text())["url"]
    body = json.dumps({"request_id": "l5", "input": "__crash__"}).encode()
    st, t = C.call("POST", "/api/v1/tickets", {"destination_installation_id": INS, "subject_identity_id": TASS, "capability": CAP,
                                               "request_sha256": hashlib.sha256(body).hexdigest()})
    try:
        st, r = http("POST", FURL + "/v1/actions/" + CAP, body, {"Content-Type": "application/json", "X-TF-Ticket": t["ticket"]})
    except Exception as e:
        st, r = 0, {"error": "connection broke: " + type(e).__name__}
    rec("L5", st == 200, "a dropped fault-injection file is ignored unless Processor runs in explicit test mode", "%d %s" % (st, r.get("error") or r.get("result")))

    # ---- L3: file permissions of feature data ----
    obs = data / "observations.jsonl"
    mode = stat.S_IMODE(os.stat(obs).st_mode) if obs.exists() else None
    old = os.umask(0o022)
    os.umask(old)
    rec("L3", mode is not None and mode & 0o077 == 0, "feature observation file is private (no group/other access)", oct(mode) if mode is not None else "missing")
    led = data / "ticket-ledger.json"
    lm = stat.S_IMODE(os.stat(led).st_mode) if led.exists() else None
    rec("L3", lm is not None and lm & 0o077 == 0, "feature ticket ledger is private", oct(lm) if lm is not None else "missing")


def finish():
    for n in list(procs):
        stop(n)
    fails = [r for r in results if r[1] != "PASS"]
    out = ["Grok findings check (%s) — identity %s / processor %s" % (args.label,
           subprocess.check_output(["git", "-C", args.identity_src, "rev-parse", "--short", "HEAD"], text=True).strip(),
           subprocess.check_output(["git", "-C", str(PSRC), "rev-parse", "--short", "HEAD"], text=True).strip()), ""]
    out += [f"{w} [{f}] {n}" + ("  — " + d if d else "") for f, w, n, d in results]
    out += ["", "secure behaviour observed: %d of %d" % (len(results) - len(fails), len(results))]
    (W / "findings-report.txt").write_text("\n".join(out) + "\n")
    print(out[-1])


try:
    main()
except Exception as e:
    import traceback
    traceback.print_exc()
    rec("HARNESS", False, "script stopped: " + repr(e)[:160])
finish()
