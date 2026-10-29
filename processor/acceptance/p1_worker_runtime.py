"""P1 secure worker runtime — acceptance test (Linux / WSL).

REAL Identity + REAL Thought Processor + REAL P1 Deterministic Worker processes.
Labelled TEST FIXTURES: GitHub (Identity's scripts/fixture_github_engine.py)
and the worker itself. The "Admin" typing the passphrase into Identity's own
approval page is this script. A second, independently paired program
("actor B") is used for the outstanding-ticket revocation check.

Worker processes are counted at the OS level (every process whose environment
carries TF_INSTALLATION_ID=<this installation>), not from Processor's own view.

  python3 -I acceptance/p1_worker_runtime.py --identity-src DIR --work DIR

Exit code 0 only when every check PASSed. The report contains no secrets; the
script itself checks that.
"""
import argparse, hashlib, hmac, json, os, re, secrets, shutil, signal, socket, subprocess, sys, threading, time, urllib.error, urllib.request
from pathlib import Path

ap = argparse.ArgumentParser()
ap.add_argument("--identity-src", required=True)
ap.add_argument("--work", required=True)
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
                       "-test-github-web", f"http://127.0.0.1:{FX}", "-test-github-api", f"http://127.0.0.1:{FX}"])
    assert wait_http(IDENTITY + "/api/health"), "identity did not start"


def start_processor():
    f = W / "processor-profile" / "runtime" / "operator-link.txt"
    if f.exists():
        f.unlink()
    # No -test-feature-faults: this is the normal (non-test-mode) runtime.
    start("processor", [str(W / "thought-processor"), "-port", str(PP), "-data", str(W / "processor-profile"), "-no-browser",
                        "-trust-feature-dir", str(FEATURES)])
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

def main():
    ids = subprocess.check_output(["git", "-C", args.identity_src, "rev-parse", "HEAD"], text=True).strip()
    tps = subprocess.check_output(["git", "-C", str(PROC_SRC), "rev-parse", "HEAD"], text=True).strip()
    dirty = subprocess.run(["git", "-C", str(PROC_SRC), "diff", "--quiet", "HEAD", "--", "."]).returncode != 0
    evidence.update(identity_commit=ids, processor_commit=tps + (" (+ uncommitted changes)" if dirty else ""))
    subprocess.check_call(["go", "build", "-o", str(W / "identity"), "."], cwd=args.identity_src)
    subprocess.check_call(["go", "build", "-o", str(W / "thought-processor"), "."], cwd=PROC_SRC)
    (W / "asset").write_bytes(b"TEST FIXTURE\n")
    start("fixture", [sys.executable, "-I", str(Path(args.identity_src) / "scripts" / "fixture_github_engine.py"), str(FX), str(W / "asset"), "linux-amd64"])
    assert wait_http(f"http://127.0.0.1:{FX}/fixture/state")
    NOPROXY.open(urllib.request.Request(f"http://127.0.0.1:{FX}/fixture/login?user=Arameair", data=b"", method="POST"), timeout=5).read()
    start_identity()
    tp = TP(start_processor())

    # ---- setup: pair, GitHub (fixture), Admin ----
    tp.until(lambda s: s["connection"]["state"] == "FOUND_NOT_CONNECTED", 20, "Identity found")
    tp.post("/api/connect/request", {"url": IDENTITY})
    time.sleep(0.5)
    tp.post("/api/connect/code", {"code": console_code("identity")})
    s = tp.until(lambda s: s["verified"], 20, "connected")
    rec("SETUP", s["verified"], "Processor paired with Identity")
    tp.post("/api/github/connect")
    tp.until(lambda s: (s.get("github") or {}).get("state") == "VERIFIED", 40, "GitHub verified")
    st, j = tp.post("/api/admin/enroll")
    st, j = admin_decide(j["approval_url"], {"decision": "approve", "passphrase": PASS, "passphrase_confirm": PASS, "acknowledge": True})
    s = tp.until(lambda s: s["admin"]["state"] == "ENROLLED", 15, "enrolled")
    rec("SETUP", s["admin"]["state"] == "ENROLLED", "Admin Identity established in Identity's own page")

    # ---- 1. Identity approval + idempotency while waiting ----
    st, j = tp.post(f"/api/library/{FEATURE}/install")
    url, key = j.get("approval_url", ""), j.get("install_key", "")
    rec("INSTALL", st == 200 and "/approve#a=apr_" in url, "install request goes to Identity for the Admin's approval (Processor can't approve)")
    st2, j2 = tp.post(f"/api/library/{FEATURE}/install")
    rec("INSTALL", st2 == 200 and j2.get("already_in_progress") is True and j2.get("install_key") == key,
        "a repeated install while the first waits returns THAT install (no second approval request)", "same install_key" if j2.get("install_key") == key else "a different install was created")
    st, j = approve(url)
    rec("INSTALL", st == 200 and j["approval"]["state"] == "APPROVED", "Identity approval succeeds (Admin passphrase in Identity's page)")
    s = tp.until(lambda s: len(s["installations"]) == 1 and s["installations"][0]["identity"]["installation"]["state"] == "INSTALLED" and not s["install_txs"], 30, "installed")
    v = s["installations"][0]
    INS = v["record"]["installation_id"]
    evidence["installation_id"] = INS
    ok = (len(s["installations"]) == 1 and len(s["identity_installations"]) == 1 and v["files"] == "OK"
          and v["identity"]["installation"]["artifact_sha256"] == v["record"]["artifact_sha256"] and v["record"]["trust_basis"])
    rec("INSTALL", ok, "ONE verified installation: files match the approved artifact; Identity and Processor agree on the digest", INS)
    rec("INSTALL", v["identity"]["active_grants"] == 0 and v["runtime"]["state"] == "STOPPED", "it starts with zero grants and is not running")

    # ---- 2. repeated install returns the existing installation ----
    st, j = tp.post(f"/api/library/{FEATURE}/install")
    rec("INSTALL", st == 200 and j.get("already_installed") is True and j.get("installation_id") == INS and "approval_url" not in j,
        "repeated install returns the EXISTING installation; no approval, nothing registered", j.get("installation_id") or ("new approval requested" if j.get("approval_url") else "HTTP %d" % st))
    outs = []
    th = [threading.Thread(target=lambda: outs.append(tp.post(f"/api/library/{FEATURE}/install"))) for _ in range(5)]
    [t.start() for t in th]
    [t.join() for t in th]
    s = tp.state()
    rec("INSTALL", all(o[1].get("installation_id") == INS for o in outs) and len(s["installations"]) == 1 and len(s["identity_installations"]) == 1 and not s["install_txs"],
        "5 simultaneous installs: all return the same installation; still 1 record in Processor and 1 in Identity")

    # ---- 3. unauthorized request denied before anything starts ----
    tp.post("/api/identities", {"name": "Tass", "kind": "companion"})
    TASS = [e for e in tp.state()["identities"] if e["name"] == "Tass"][0]["id"]
    r = action(tp, INS, TASS, "no grant yet")
    rec("AUTHZ", r["outcome"] == "DENIED_BY_IDENTITY" and not r["started_feature"] and worker_pids(INS) == [],
        "unauthorized capability request DENIED by Identity; no worker process started", r["detail"][:70])

    # ---- 4. Start: exactly one managed process ----
    st, j = tp.post(f"/api/installations/{INS}/start")
    s = tp.until(lambda s: inst(s, INS)["runtime"]["state"] == "RUNNING", 15, "running")
    rt = inst(s, INS)["runtime"]
    pids = worker_pids(INS)
    rec("RUN", st == 200 and j.get("already_running") is False and rt["state"] == "RUNNING" and pids == [rt["pid"]],
        "Start launches exactly ONE managed worker process (OS process scan agrees)", "pids %s" % pids)
    PID1, LAUNCH1 = rt["pid"], rt["launch_id"]

    # ---- 5. repeated Start: no new process ----
    st, j = tp.post(f"/api/installations/{INS}/start")
    outs = []
    th = [threading.Thread(target=lambda: outs.append(tp.post(f"/api/installations/{INS}/start"))) for _ in range(5)]
    [t.start() for t in th]
    [t.join() for t in th]
    rt = inst(tp.state(), INS)["runtime"]
    rec("RUN", j.get("already_running") is True and all(o[0] == 200 and o[1].get("already_running") for o in outs)
        and worker_pids(INS) == [PID1] and rt["launch_id"] == LAUNCH1,
        "repeated Start (1 + 5 simultaneous) reports already running; still the same single process and launch", "pid %d" % PID1)

    # ---- 6. standalone self-test ----
    time.sleep(1)
    st, j = tp.post(f"/api/installations/{INS}/self-test")
    t = j.get("self_test") or {}
    rec("SELFTEST", st == 200 and t.get("result") == "PASS" and t.get("launch_id") == LAUNCH1 and t.get("external_calls") == 0
        and (t.get("output") or {}).get("upper") == "P1 SELF-TEST", "standalone self-test returns a real PASS from this launch (signed by its launch key)", t.get("result"))
    kept = inst(tp.state(), INS).get("self_test") or {}
    rec("SELFTEST", kept.get("result") == "PASS" and kept.get("at") == t.get("at"), "the result is kept by Processor and shown in the page state (not lost on refresh)")
    furl = endpoint(INS)
    st, _ = http("POST", furl + "/v1/self-test", b"{}", {"Content-Type": "application/json"})
    rec("SELFTEST", st == 401, "the worker refuses a self-test call not signed with its launch key", str(st))

    # ---- 7. authorized request succeeds ----
    st, j = tp.post(f"/api/installations/{INS}/grants/request", {"subject": TASS, "capability": CAP, "expires_in_seconds": 0})
    approve(j["approval_url"])
    s = tp.until(lambda s: inst(s, INS)["identity"]["active_grants"] == 1, 20, "grant active")
    GID = [g for g in inst(s, INS)["identity"]["grants"] if g["status"] == "ACTIVE"][0]["grant"]["grant_id"]
    r = action(tp, INS, TASS, "hello runtime")
    obs = r.get("observation") or {}
    rec("AUTHZ", r["outcome"] == "COMPLETED" and obs.get("output", {}).get("upper") == "HELLO RUNTIME" and not r["started_feature"] and worker_pids(INS) == [PID1],
        "correctly authorized request succeeds (ticket + live Identity decision) on the SAME running process", r.get("decision_id", ""))
    r = action(tp, INS, TASS, "")  # empty input still needs and has authorization
    st, j = http("POST", furl + "/v1/actions/" + CAP, json.dumps({"request_id": "x", "input": "x"}).encode(), {"Content-Type": "application/json", "X-TF-Ticket": "tk1.e30." + "0" * 64})
    rec("AUTHZ", st == 401, "a forged ticket sent straight to the worker is refused", j.get("reason", ""))

    # ---- 8. revocation ----
    st, j = tp.post(f"/api/installations/{INS}/grants/{GID}/revoke")
    r = action(tp, INS, TASS, "after revoke")
    rec("REVOKE", st == 200 and r["outcome"] == "DENIED_BY_IDENTITY" and "REVOKED" in r["detail"].upper(), "after revocation the next privileged request is denied", r["detail"][:70])
    B = pair_actor_b()
    st, j = B.call("POST", "/api/v1/grants/requests", {"destination_installation_id": INS, "subject_identity_id": TASS, "capability": CAP, "expires_in_seconds": 0})
    approve(j["approval_url"])
    st, j = B.call("POST", "/api/v1/grants", {"approval_id": j["approval"]["id"]})
    BGID = (j.get("grant") or {}).get("grant_id", "")
    body = json.dumps({"request_id": "b-1", "input": "outstanding ticket"}).encode()
    st, tk = B.call("POST", "/api/v1/tickets", {"destination_installation_id": INS, "subject_identity_id": TASS, "capability": CAP, "request_sha256": hashlib.sha256(body).hexdigest()})
    st2, _ = B.call("POST", f"/api/v1/grants/{BGID}/revoke", {"reason": "acceptance: revoke with a ticket outstanding"})
    st3, j3 = http("POST", furl + "/v1/actions/" + CAP, body, {"Content-Type": "application/json", "X-TF-Ticket": tk.get("ticket", "")})
    rec("REVOKE", bool(tk.get("ticket")) and st2 == 200 and st3 == 403 and j3.get("error") == "DENIED",
        "a ticket issued BEFORE revocation is refused by the worker's live Identity check afterwards", "%s %s" % (st3, j3.get("reason", "")))

    # ---- 9. Stop ----
    st, j = tp.post(f"/api/installations/{INS}/stop")
    s = tp.until(lambda s: inst(s, INS)["runtime"]["state"] == "STOPPED", 20, "stopped")
    le = inst(s, INS)["runtime"].get("last_exit") or {}
    rec("RUN", inst(s, INS)["runtime"]["state"] == "STOPPED" and not pid_alive(PID1) and worker_pids(INS) == [] and le.get("expected") is True and le.get("code") == 0,
        "Stop terminates the owned worker cleanly (exit 0 after a stop request); nothing left running", "pid %d" % PID1)
    rt_dir = W / "processor-profile" / "installations" / INS / "runtime"
    rec("RUN", not list(rt_dir.glob("launch-key-*")) and not (rt_dir / "launch.json").exists(), "no launch key file or launch record left behind")

    # ---- 10. failures handled safely ----
    wp = W / "processor-profile" / "installations" / INS / "package" / "worker.py"
    orig = wp.read_bytes()
    os.chmod(wp, 0o600)
    wp.write_bytes(orig + b"\n# tampered\n")
    st, j = tp.post(f"/api/installations/{INS}/start")
    s = tp.state()
    rec("ERRORS", st == 409 and inst(s, INS)["runtime"]["state"] == "FAILED_TO_START" and worker_pids(INS) == [],
        "tampered installed file: Start refused, state is an error, no process started", (j.get("error") or "")[:70])
    wp.write_bytes(orig)
    tp.post(f"/api/installations/{INS}/start")
    s = tp.until(lambda s: inst(s, INS)["runtime"]["state"] == "RUNNING", 15, "running again")
    pid2 = inst(s, INS)["runtime"]["pid"]
    rec("ERRORS", worker_pids(INS) == [pid2], "after restoring the approved file, Start works again (one process)")
    os.kill(pid2, signal.SIGKILL)
    s = tp.until(lambda s: inst(s, INS)["runtime"]["state"] == "NEEDS_ATTENTION", 15, "crash noticed")
    rec("ERRORS", inst(s, INS)["runtime"]["state"] == "NEEDS_ATTENTION" and worker_pids(INS) == [],
        "worker killed from outside: shown as an error (manual recovery), not silently restarted, nothing orphaned")

    # Identity unavailable
    stop("identity")
    s = tp.until(lambda s: not s["verified"], 30, "authority unavailable")
    st, j = tp.post(f"/api/installations/{INS}/start")
    rec("ERRORS", st == 409 and "AUTHORITY_UNAVAILABLE" in j.get("error", "") and worker_pids(INS) == [],
        "Identity unavailable: Start refused (fails closed), nothing started", j.get("error", "")[:70])
    r = action(tp, INS, TASS, "no authority")
    rec("ERRORS", r["outcome"] == "AUTHORITY_UNAVAILABLE" and worker_pids(INS) == [], "Identity unavailable: privileged request refused, nothing started", r["detail"][:60])
    st, j = tp.post(f"/api/library/{FEATURE}/install")
    rec("ERRORS", st != 200, "Identity unavailable: install refused", str(st))
    start_identity()
    s = tp.until(lambda s: s["verified"] and inst(s, INS)["identity"] is not None, 40, "Identity back")
    rec("ERRORS", s["verified"], "Identity back: Processor reconnected by itself")

    # ---- 11. restart preserves identity, no duplicates ----
    tp.post(f"/api/installations/{INS}/start")
    s = tp.until(lambda s: inst(s, INS)["runtime"]["state"] == "RUNNING", 15, "running before restart")
    pid3 = inst(s, INS)["runtime"]["pid"]
    stop("processor")
    time.sleep(0.5)
    rec("RESTART", not pid_alive(pid3) and worker_pids(INS) == [], "Processor exit stops the worker it started (no orphan)")
    stop("identity")
    start_identity()
    tp = TP(start_processor())
    s = tp.until(lambda s: s["verified"] and inst(s, INS) and inst(s, INS)["identity"] is not None, 40, "back after restart")
    rec("RESTART", [v["record"]["installation_id"] for v in s["installations"]] == [INS] and len(s["identity_installations"]) == 1
        and s["identity_installations"][0]["installation"]["installation_id"] == INS,
        "after restarting both programs: the same single installation ID in Processor and Identity", INS)
    st, j = tp.post(f"/api/library/{FEATURE}/install")
    rec("RESTART", j.get("already_installed") is True and j.get("installation_id") == INS, "install after restart still returns the existing installation")
    tp.post(f"/api/installations/{INS}/start")
    s = tp.until(lambda s: inst(s, INS)["runtime"]["state"] == "RUNNING", 15, "running after restart")
    pid4 = inst(s, INS)["runtime"]["pid"]
    st, j = tp.post(f"/api/installations/{INS}/self-test")
    rec("RESTART", worker_pids(INS) == [pid4] and (j.get("self_test") or {}).get("result") == "PASS",
        "after restart: one process again and self-test PASS (same installation identity)")
    tp.post(f"/api/installations/{INS}/stop")
    tp.until(lambda s: inst(s, INS)["runtime"]["state"] == "STOPPED", 20, "final stop")

    # ---- 12. logs are useful and redacted ----
    cred = json.loads((W / "processor-profile" / "installations" / INS / "credential.json").read_text())
    secrets_seen = [cred["secret"], console_code("identity")]
    logs = [W / "processor-profile" / "logs" / "processor.jsonl", W / "processor-profile" / "installations" / INS / "logs" / "feature.log",
            W / "processor-profile" / "installations" / INS / "logs" / "requests.jsonl"]
    text = "".join(p.read_text(errors="replace") for p in logs if p.exists())
    leaks = [x[:4] + "…" for x in secrets_seen if x and x in text] + (["approval token"] if re.search(r"[#&]t=[0-9a-f]{16,}", text) else []) + (["ticket"] if "tk1." in text and "tk1.<removed>" not in text else [])
    useful = all(e in text for e in ("install.already_installed", "feature.start_ignored", "feature.self_test", "feature.stopped", "action.completed", "action.denied_by_identity"))
    rec("LOGS", not leaks and useful, "operational logs record installs, starts, self-tests, decisions and stops; no credential secret, pairing code, approval token or ticket", ", ".join(leaks) or "clean")


if __name__ == "__main__":
    try:
        main()
    except Exception as e:  # report, don't hide
        import traceback
        traceback.print_exc()
        rec("HARNESS", False, "the test script itself failed", type(e).__name__ + ": " + str(e)[:160])
    finally:
        for n in list(procs):
            stop(n)
    failed = [r for r in results if r[1] != "PASS"]
    evidence["elapsed_s"] = int(time.time() - T0)
    lines = [f"P1 secure worker runtime acceptance — {time.strftime('%Y-%m-%d %H:%M:%S %Z')}", "environment: " + " ".join(os.uname()[0:3:2]),
             "evidence: " + json.dumps(evidence, sort_keys=True), ""] + [f"{w} [{a}] {n}" + (f"  — {d}" if d != "" else "") for a, w, n, d in results]
    lines.append(f"\nRESULT: {'PASS' if not failed else 'FAIL'} ({len(results)} checks, {len(failed)} failed)")
    (W / "p1-worker-runtime-report.txt").write_text("\n".join(lines) + "\n")
    print(lines[-1].strip())
    print("report:", W / "p1-worker-runtime-report.txt")
    sys.exit(0 if not failed else 1)
