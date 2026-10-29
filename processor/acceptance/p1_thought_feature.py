"""Thought feature plugged into Thought Processor — acceptance test (Linux / WSL).

REAL Identity + REAL Thought Processor + the REAL Thought feature
(Arameair/Thought, feature/) and the P1 worker side by side. Labelled TEST
FIXTURE: GitHub (Identity's scripts/fixture_github_engine.py). The "Admin"
typing the passphrase into Identity's own approval page is this script.

Thought is trusted only as one exact pinned artifact, computed here from the
Thought checkout and required to equal the pin in run-p1.sh, then passed to
Processor (-trust-feature-artifact) and Identity (-trust-fixture-artifact).

  python3 -I acceptance/p1_thought_feature.py --identity-src DIR --thought-src DIR --work DIR

Exit code 0 only when every check PASSed. The report contains no secrets.
"""
import argparse, hashlib, hmac, json, os, re, secrets, shutil, signal, socket, subprocess, sys, threading, time, urllib.error, urllib.request
from pathlib import Path

ap = argparse.ArgumentParser()
ap.add_argument("--identity-src", required=True)
ap.add_argument("--work", required=True)
ap.add_argument("--thought-src", required=True)
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
                       "-trust-fixture-artifact", PINS["thought"]])
    assert wait_http(IDENTITY + "/api/health"), "identity did not start"


def start_processor():
    f = W / "processor-profile" / "runtime" / "operator-link.txt"
    if f.exists():
        f.unlink()
    start("processor", [str(W / "thought-processor"), "-port", str(PP), "-data", str(W / "processor-profile"), "-no-browser",
                        "-trust-feature-dir", str(FEATURES), "-trust-feature-dir", str(THOUGHT_PKG),
                        "-trust-feature-dir", str(W / "tampered"), "-trust-feature-artifact", PINS["thought"]])
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

SUBMIT, READ, UPDATE = "thought.event.submit", "thought.thoughts.read", "thought.thought.update"


def artifact_digest(pkg):
    m = hashlib.sha256((pkg / "feature.json").read_bytes()).hexdigest()
    files = sorted(json.loads((pkg / "feature.json").read_text())["files"], key=lambda f: f["path"])
    b = "thought.feature-package/1\nmanifest %s\n" % m + "".join("%s  %s\n" % (hashlib.sha256((pkg / f["path"]).read_bytes()).hexdigest(), f["path"]) for f in files)
    return hashlib.sha256(b.encode()).hexdigest()


def ask(tp, ins, subject, cap, text):
    st, j = tp.post(f"/api/installations/{ins}/action", {"subject": subject, "capability": cap, "input": text})
    return j.get("result") or {"outcome": "HTTP %d" % st, "detail": safe(j.get("error", ""))}


def out(r):
    return ((r.get("observation") or {}).get("output")) or {}


def grant(tp, ins, subject, cap):
    st, j = tp.post(f"/api/installations/{ins}/grants/request", {"subject": subject, "capability": cap, "expires_in_seconds": 0})
    st2, _ = approve(j["approval_url"])
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


def main():
    PINS["thought"] = artifact_digest(THOUGHT_PKG)
    launcher = (PROC_SRC.parent / "run-p1.sh").read_text()
    pinned = re.search(r"^THOUGHT_ARTIFACT=([0-9a-f]{64})$", launcher, re.M)
    ids = subprocess.check_output(["git", "-C", args.identity_src, "rev-parse", "HEAD"], text=True).strip()
    tps = subprocess.check_output(["git", "-C", str(PROC_SRC), "rev-parse", "HEAD"], text=True).strip()
    ths = subprocess.check_output(["git", "-C", args.thought_src, "rev-parse", "HEAD"], text=True).strip()
    dirty = lambda d: subprocess.run(["git", "-C", str(d), "diff", "--quiet", "HEAD", "--", "."]).returncode != 0
    evidence.update(identity_commit=ids, processor_commit=tps + (" (+ uncommitted changes)" if dirty(PROC_SRC) else ""),
                    thought_commit=ths + (" (+ uncommitted changes)" if dirty(args.thought_src) else ""), thought_artifact=PINS["thought"])
    rec("PIN", bool(pinned) and pinned.group(1) == PINS["thought"], "the Thought package in the checkout is exactly the artifact pinned in run-p1.sh",
        PINS["thought"][:16] + "…" + ("" if pinned and pinned.group(1) == PINS["thought"] else " (run-p1.sh pins " + (pinned.group(1)[:16] if pinned else "nothing") + "…)"))
    # A tampered copy in a trusted folder must stay BLOCKED.
    bad = W / "tampered" / "thought-tampered"
    shutil.copytree(THOUGHT_PKG, bad)
    m = json.loads((bad / "feature.json").read_text())
    m["feature_id"] = "thought-tampered"
    code = (bad / "thought_feature.py").read_bytes() + b"\n# one extra line\n"
    (bad / "thought_feature.py").write_bytes(code)
    m["files"][0]["sha256"] = hashlib.sha256(code).hexdigest()
    (bad / "feature.json").write_text(json.dumps(m, indent=2))

    subprocess.check_call(["go", "build", "-o", str(W / "identity"), "."], cwd=args.identity_src)
    subprocess.check_call(["go", "build", "-o", str(W / "thought-processor"), "."], cwd=PROC_SRC)
    (W / "asset").write_bytes(b"TEST FIXTURE\n")
    start("fixture", [sys.executable, "-I", str(Path(args.identity_src) / "scripts" / "fixture_github_engine.py"), str(FX), str(W / "asset"), "linux-amd64"])
    assert wait_http(f"http://127.0.0.1:{FX}/fixture/state")
    NOPROXY.open(urllib.request.Request(f"http://127.0.0.1:{FX}/fixture/login?user=Arameair", data=b"", method="POST"), timeout=5).read()
    start_identity()
    tp = TP(start_processor())
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
    for name, kind in (("Tass", "companion"), ("Other", "person")):
        tp.post("/api/identities", {"name": name, "kind": kind})
    s = tp.state()
    TASS = [e for e in s["identities"] if e["name"] == "Tass"][0]["id"]
    OTHER = [e for e in s["identities"] if e["name"] == "Other"][0]["id"]

    # ---- library + install ----
    lib = {x.get("feature_id"): x for x in s["library"]}
    t = lib.get("thought") or {}
    rec("LIBRARY", t and not t.get("blocked") and "operator pin" in (t.get("trust_basis") or "") and len(t.get("capabilities") or []) == 3,
        "Processor's library offers Thought (pinned artifact) with its 3 capabilities", (t.get("trust_basis") or t.get("blocked") or "missing"))
    tt = lib.get("thought-tampered") or {}
    rec("LIBRARY", bool(tt.get("blocked")), "a changed copy of the Thought package in a trusted folder stays BLOCKED", (tt.get("blocked") or "offered!")[:60])
    st, j = tp.post("/api/library/thought-tampered/install")
    rec("LIBRARY", st != 200, "installing the changed copy is refused", str(st))
    TH = install(tp, "thought")
    WK = install(tp, FEATURE)
    s = tp.state()
    v = inst(s, TH)
    evidence["thought_installation"] = TH
    rec("INSTALL", v["files"] == "OK" and v["identity"]["active_grants"] == 0 and v["identity"]["installation"]["artifact_sha256"] == PINS["thought"]
        and len(s["identity_installations"]) == 2, "Thought installed through Identity approval: own installation ID, pinned artifact, zero grants; the worker installed beside it", TH)
    st, j = tp.post("/api/library/thought/install")
    rec("INSTALL", j.get("already_installed") is True and j.get("installation_id") == TH, "installing Thought again returns the same installation")

    # ---- run + self-test ----
    tp.post(f"/api/installations/{TH}/start")
    s = tp.until(lambda s: inst(s, TH)["runtime"]["state"] == "RUNNING", 15, "thought running")
    pid = inst(s, TH)["runtime"]["pid"]
    rec("RUN", worker_pids(TH) == [pid], "Thought runs as ONE process managed by Processor", "pid %d" % pid)
    st, j = tp.post(f"/api/installations/{TH}/self-test")
    t = j.get("self_test") or {}
    checks = (t.get("output") or {}).get("checks") or []
    rec("RUN", t.get("result") == "PASS" and len(checks) >= 8 and all(c["ok"] for c in checks), "Thought's standalone self-test PASS (real rules on a throwaway store)",
        "%d/%d rule checks" % (sum(c["ok"] for c in checks), len(checks)))
    st, j = tp.post(f"/api/installations/{TH}/start")
    rec("RUN", j.get("already_running") is True and worker_pids(TH) == [pid], "Start again: same single process")

    # ---- authorization per identity and per capability ----
    r = ask(tp, TH, TASS, SUBMIT, "goldshire: the player went back to Goldshire")
    rec("AUTHZ", r["outcome"] == "DENIED_BY_IDENTITY", "no grant: Tass's event is DENIED by Identity, nothing processed", r["detail"][:60])
    G_SUBMIT = grant(tp, TH, TASS, SUBMIT)
    rec("AUTHZ", bool(G_SUBMIT), "Admin granted exactly: this Processor → Thought → thought.event.submit → Tass")

    # ---- Spark → Trace → Thought through Processor ----
    r = ask(tp, TH, TASS, SUBMIT, "goldshire: a passing mention !weak")
    rec("SPARK", r["outcome"] == "COMPLETED" and out(r).get("outcome") == "faded", "weak event: Spark fades, nothing kept", out(r).get("headline", r["detail"][:60]))
    outs = []
    for i in range(3):
        r = ask(tp, TH, TASS, SUBMIT, "goldshire: the player went back, private-marker-zq7x (%d)" % (i + 1))
        outs.append(out(r).get("outcome"))
    th = out(r).get("thought") or {}
    THOUGHT_ID = th.get("thought_id", "")
    rec("SPARK", outs == ["trace_kept", "trace_kept", "thought_formed"] and th.get("evidence_count") == 3,
        "3 related events → Trace, Trace, then ONE Thought linked to all 3", " → ".join(str(o) for o in outs))
    rec("SPARK", (r.get("observation") or {}).get("subject_identity_id") == TASS,
        "the Thought belongs to the identity Identity signed into the ticket (Tass)")
    ev = json.dumps({"event_id": "evt-fixed-1", "topic": "stormwind", "summary": "first time in Stormwind"})
    r1, r2 = ask(tp, TH, TASS, SUBMIT, ev), ask(tp, TH, TASS, SUBMIT, ev)
    rec("SPARK", out(r1).get("outcome") == "trace_kept" and out(r2).get("outcome") == "duplicate_ignored", "the same event ID twice: second is ignored")
    r = ask(tp, TH, TASS, SUBMIT, "no colon here")
    rec("SPARK", r["outcome"] == "ERROR" and "BAD_INPUT" in r["detail"], "unusable input: allowed by Identity but refused by Thought with a clear reason; nothing changed", r["detail"][:70])

    # ---- read: its own grant ----
    r = ask(tp, TH, TASS, READ, "")
    rec("READ", r["outcome"] == "DENIED_BY_IDENTITY", "reading needs its OWN grant: denied with only the submit grant")
    G_READ = grant(tp, TH, TASS, READ)
    r = ask(tp, TH, TASS, READ, "")
    o = out(r)
    ths_ = o.get("thoughts") or []
    rec("READ", r["outcome"] == "COMPLETED" and len(ths_) == 1 and ths_[0]["thought_id"] == THOUGHT_ID and ths_[0]["evidence_count"] == 3
        and ths_[0]["why_it_formed"] and [e["role"] for e in ths_[0]["evidence"]] == ["ORIGIN"] * 3,
        "Tass's Thoughts read back through Processor: 1 Thought, why it formed, its 3 ORIGIN events")
    rec("READ", (o.get("building_up") or [{}])[0].get("topic") == "stormwind", "and what is building up (stormwind: 1 of 3)")

    # ---- isolation between identities ----
    r = ask(tp, TH, OTHER, READ, "")
    rec("ISOLATION", r["outcome"] == "DENIED_BY_IDENTITY", "Tass's read grant does NOT let anyone read for Other")
    grant(tp, TH, OTHER, READ)
    r = ask(tp, TH, OTHER, READ, "")
    rec("ISOLATION", r["outcome"] == "COMPLETED" and out(r).get("thoughts") == [], "with its own read grant, Other sees none of Tass's Thoughts")
    r = ask(tp, TH, OTHER, READ, THOUGHT_ID)
    rec("ISOLATION", r["outcome"] == "ERROR" and "NOT_FOUND" in r["detail"], "asking for Tass's Thought by its ID as Other: NOT_FOUND")

    # ---- update: sleep, then new evidence wakes it ----
    r = ask(tp, TH, TASS, UPDATE, "sleep " + THOUGHT_ID)
    rec("UPDATE", r["outcome"] == "DENIED_BY_IDENTITY", "changing a Thought needs its own grant too")
    grant(tp, TH, TASS, UPDATE)
    r = ask(tp, TH, TASS, UPDATE, "sleep " + THOUGHT_ID)
    rec("UPDATE", (out(r).get("thought") or {}).get("state") == "sleeping", "Thought put to sleep")
    r = ask(tp, TH, TASS, SUBMIT, "goldshire: back in Goldshire again !strong")
    rec("UPDATE", out(r).get("outcome") == "thought_woke" and (out(r).get("thought") or {}).get("thought_id") == THOUGHT_ID,
        "new related evidence woke the SAME Thought")

    # ---- revocation, forged ticket ----
    tp.post(f"/api/installations/{TH}/grants/{G_SUBMIT}/revoke")
    r = ask(tp, TH, TASS, SUBMIT, "goldshire: after revoke")
    rec("REVOKE", r["outcome"] == "DENIED_BY_IDENTITY" and "REVOKED" in r["detail"].upper(), "after revoking submit: new events denied", r["detail"][:60])
    r = ask(tp, TH, TASS, READ, "")
    rec("REVOKE", r["outcome"] == "COMPLETED" and len(out(r).get("thoughts") or []) == 1, "the separate read grant still works (need-to-know, one capability at a time)")
    furl = endpoint(TH)
    st, j = http("POST", furl + "/v1/actions/" + READ, json.dumps({"request_id": "x", "input": ""}).encode(),
                 {"Content-Type": "application/json", "X-TF-Ticket": "tk1.e30." + "0" * 64})
    rec("REVOKE", st == 401, "a forged ticket sent straight to Thought is refused", j.get("reason", ""))
    st, _ = http("POST", furl + "/v1/self-test", b"{}", {"Content-Type": "application/json"})
    rec("REVOKE", st == 401, "Thought refuses launcher calls not signed with its launch key", str(st))

    # ---- the worker still works beside it ----
    grant(tp, WK, TASS, "worker.test.synthetic")
    tp.post(f"/api/installations/{WK}/start")
    tp.until(lambda s: inst(s, WK)["runtime"]["state"] == "RUNNING", 15, "worker running")
    r = ask(tp, WK, TASS, "worker.test.synthetic", "side by side")
    rec("TOGETHER", r["outcome"] == "COMPLETED" and worker_pids(TH) == [pid] and len(worker_pids(WK)) == 1,
        "the P1 worker runs and answers beside Thought; each is one process", r["outcome"])

    # ---- Identity outage ----
    stop("identity")
    tp.until(lambda s: not s["verified"], 30, "authority unavailable")
    r = ask(tp, TH, TASS, READ, "")
    rec("ERRORS", r["outcome"] == "AUTHORITY_UNAVAILABLE", "Identity down: Thought can't be used (fails closed)")
    start_identity()
    tp.until(lambda s: s["verified"] and inst(s, TH)["identity"] is not None, 40, "Identity back")

    # ---- restart: Thoughts and identities persist ----
    stop("processor")
    time.sleep(0.5)
    rec("RESTART", worker_pids(TH) == [] and worker_pids(WK) == [], "Processor exit stops Thought and the worker (no orphans)")
    stop("identity")
    start_identity()
    tp = TP(start_processor())
    s = tp.until(lambda s: s["verified"] and inst(s, TH) and inst(s, TH)["identity"] is not None, 40, "back")
    tp.post(f"/api/installations/{TH}/start")
    tp.until(lambda s: inst(s, TH)["runtime"]["state"] == "RUNNING", 15, "thought running again")
    r = ask(tp, TH, TASS, READ, "")
    ths_ = out(r).get("thoughts") or []
    rec("RESTART", len(s["identity_installations"]) == 2 and len(ths_) == 1 and ths_[0]["thought_id"] == THOUGHT_ID and ths_[0]["evidence_count"] == 4
        and ths_[0]["state"] == "active", "after restarting everything: same installation, same single Thought (4 events), grants still in force")
    tp.post(f"/api/installations/{TH}/stop")
    tp.until(lambda s: inst(s, TH)["runtime"]["state"] == "STOPPED", 20, "stopped")
    rec("RESTART", worker_pids(TH) == [], "Stop: Thought process gone, Thoughts kept on disk")

    # ---- logs: useful, no secrets, no Thought content in Processor's logs ----
    cred = json.loads((W / "processor-profile" / "installations" / TH / "credential.json").read_text())
    plog = (W / "processor-profile" / "logs" / "processor.jsonl").read_text(errors="replace")
    reqs = (W / "processor-profile" / "installations" / TH / "logs" / "requests.jsonl").read_text(errors="replace")
    flog = (W / "processor-profile" / "installations" / TH / "logs" / "feature.log").read_text(errors="replace")
    alltext = plog + reqs + flog
    leaks = [x for x in (cred["secret"], console_code("identity")) if x and x in alltext]
    content = "private-marker-zq7x" in plog + reqs  # event text sent for Tass must not be copied into Processor's logs
    rec("LOGS", not leaks and not content and "action.completed" in flog and "thought_formed" in flog,
        "logs record decisions and outcomes, not secrets, and Processor's logs don't copy the identity's event text", "clean" if not leaks and not content else "leak")


if __name__ == "__main__":
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
    lines = [f"Thought feature in Thought Processor — acceptance {time.strftime('%Y-%m-%d %H:%M:%S %Z')}", "environment: " + " ".join(os.uname()[0:3:2]),
             "evidence: " + json.dumps(evidence, sort_keys=True), ""] + [f"{w} [{a}] {n}" + (f"  — {d}" if d != "" else "") for a, w, n, d in results]
    lines.append(f"\nRESULT: {'PASS' if not failed else 'FAIL'} ({len(results)} checks, {len(failed)} failed)")
    (W / "p1-thought-feature-report.txt").write_text("\n".join(lines) + "\n")
    print(lines[-1].strip())
    print("report:", W / "p1-thought-feature-report.txt")
    sys.exit(0 if not failed else 1)
