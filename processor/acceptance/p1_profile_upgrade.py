"""Profile reuse test: the owner's secure-worker profile, opened by the Thought build.

Phase A runs the build the owner accepted on WSL (Processor 8af218d, the
secure worker runtime) on fresh profiles: pair, GitHub (TEST FIXTURE), Admin,
Tass, install the P1 worker, grant it, one allowed request. Both programs stop.
Phase B starts THIS build (Thought integration) on the SAME profiles and SAME
ports, with Thought pinned, exactly as run-p1.sh would, and checks that
nothing the owner set up changed: no new pairing, same Admin, same identity,
same worker installation and grant, the worker still works; Thought installs
beside it as a separate installation.

  python3 -I acceptance/p1_profile_upgrade.py --identity-src DIR --thought-src DIR --work DIR
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
                       *(["-trust-fixture-artifact", PINS["thought"]] if PHASE["b"] else [])])
    assert wait_http(IDENTITY + "/api/health"), "identity did not start"


def start_processor():
    f = W / "processor-profile" / "runtime" / "operator-link.txt"
    if f.exists():
        f.unlink()
    start("processor", [str(W / BIN["processor"]), "-port", str(PP), "-data", str(W / "processor-profile"), "-no-browser",
                        "-trust-feature-dir", str(FEATURES),
                        *(["-trust-feature-dir", str(THOUGHT_PKG), "-trust-feature-artifact", PINS["thought"]] if PHASE["b"] else [])])
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

ACCEPTED_RUNTIME = "8af218d530b12920e0343a390830021a37704ce3"  # secure worker runtime, accepted by the owner on WSL
BIN = {}
PHASE = {"b": False}
WCAP = "worker.test.synthetic"


def artifact_digest(pkg):
    m = hashlib.sha256((pkg / "feature.json").read_bytes()).hexdigest()
    files = sorted(json.loads((pkg / "feature.json").read_text())["files"], key=lambda f: f["path"])
    b = "thought.feature-package/1\nmanifest %s\n" % m + "".join("%s  %s\n" % (hashlib.sha256((pkg / f["path"]).read_bytes()).hexdigest(), f["path"]) for f in files)
    return hashlib.sha256(b.encode()).hexdigest()


def snapshot(tp):
    s = tp.state()
    ws = [v for v in s["installations"] if v["record"]["feature_id"] == FEATURE]
    w = ws[0] if ws else None
    grants = [(g["grant"]["grant_id"], g["status"]) for g in ((w or {}).get("identity") or {}).get("grants", [])] if w else []
    return {"processor_id": s["processor"]["processor_id"], "identity_system_id": s["connection"].get("identity_system_id"),
            "admin_id": ((s.get("admin") or {}).get("admin") or {}).get("admin_id"), "identities": sorted((e["name"], e["id"]) for e in s.get("identities") or []),
            "worker_ids": [v["record"]["installation_id"] for v in ws], "grants": grants, "identity_installations": len(s["identity_installations"])}


def main():
    PINS["thought"] = artifact_digest(THOUGHT_PKG)
    ids = subprocess.check_output(["git", "-C", args.identity_src, "rev-parse", "HEAD"], text=True).strip()
    tps = subprocess.check_output(["git", "-C", str(PROC_SRC), "rev-parse", "HEAD"], text=True).strip()
    evidence.update(identity_commit=ids, phase_a_processor=ACCEPTED_RUNTIME, phase_b_processor=tps)
    old = W / "src-accepted"
    subprocess.check_call(["git", "-C", str(PROC_SRC), "worktree", "add", "-q", "--detach", str(old), ACCEPTED_RUNTIME])
    try:
        subprocess.check_call(["go", "build", "-o", str(W / "processor-accepted"), "."], cwd=old / "processor")
    finally:
        subprocess.run(["git", "-C", str(PROC_SRC), "worktree", "remove", "--force", str(old)])
    subprocess.check_call(["go", "build", "-o", str(W / "processor-new"), "."], cwd=PROC_SRC)
    subprocess.check_call(["go", "build", "-o", str(W / "identity"), "."], cwd=args.identity_src)
    (W / "asset").write_bytes(b"TEST FIXTURE\n")
    start("fixture", [sys.executable, "-I", str(Path(args.identity_src) / "scripts" / "fixture_github_engine.py"), str(FX), str(W / "asset"), "linux-amd64"])
    assert wait_http(f"http://127.0.0.1:{FX}/fixture/state")
    NOPROXY.open(urllib.request.Request(f"http://127.0.0.1:{FX}/fixture/login?user=Arameair", data=b"", method="POST"), timeout=5).read()

    # ---- Phase A: the accepted secure-worker build sets up the owner's profile ----
    BIN["processor"] = "processor-accepted"
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
    tp.until(lambda s: s["admin"]["state"] == "ENROLLED", 15, "enrolled")
    tp.post("/api/identities", {"name": "Tass", "kind": "companion"})
    TASS = [e for e in tp.state()["identities"] if e["name"] == "Tass"][0]["id"]
    st, j = tp.post(f"/api/library/{FEATURE}/install")
    approve(j["approval_url"])
    s = tp.until(lambda s: len(s["installations"]) == 1 and s["installations"][0]["identity"]["installation"]["state"] == "INSTALLED", 30, "worker installed")
    WK = s["installations"][0]["record"]["installation_id"]
    st, j = tp.post(f"/api/installations/{WK}/grants/request", {"subject": TASS, "capability": WCAP, "expires_in_seconds": 86400})
    approve(j["approval_url"])
    tp.until(lambda s: inst(s, WK)["identity"]["active_grants"] == 1, 20, "grant")
    tp.post(f"/api/installations/{WK}/start")
    tp.until(lambda s: inst(s, WK)["runtime"]["state"] == "RUNNING", 15, "running")
    r = action(tp, WK, TASS, "before upgrade")
    before = snapshot(tp)
    rec("A", r["outcome"] == "COMPLETED" and len(before["worker_ids"]) == 1 and before["grants"] and before["admin_id"],
        "accepted build (8af218d): Admin, Tass, worker installed + granted, request COMPLETED", WK)
    ledger = W / "processor-profile" / "installations" / WK / "data" / "ticket-ledger.json"
    led_before = ledger.read_bytes() if ledger.exists() else b""
    stop("processor")
    stop("identity")
    rec("A", worker_pids(WK) == [], "both programs stopped (Ctrl+C), worker stopped with them")

    # ---- Phase B: the Thought build opens the same profiles, same ports ----
    PHASE["b"] = True
    BIN["processor"] = "processor-new"
    start_identity()
    tp = TP(start_processor())
    s = tp.until(lambda s: s["verified"] and inst(s, WK) and inst(s, WK)["identity"] is not None, 40, "reconnected")
    rec("B", s["verified"] and s["connection"]["state"] == "CONNECTED", "new build reconnects to Identity by itself — no new pairing code")
    after = snapshot(tp)
    same = {k: (before[k], after[k]) for k in before if before[k] != after[k]}
    rec("B", not same, "same Processor ID, Identity, Admin, identities, worker installation ID and grants (nothing changed)", "differs: " + str(sorted(same)) if same else "")
    v = inst(s, WK)
    rec("B", v["files"] == "OK" and v["identity"]["installation"]["state"] == "INSTALLED" and v["runtime"]["state"] == "STOPPED",
        "worker files still verify against its pinned artifact; it is stopped until you press Start (manual)")
    st, j = tp.post(f"/api/library/{FEATURE}/install")
    rec("B", j.get("already_installed") is True and j.get("installation_id") == WK, "worker install still returns the SAME installation (no duplicate)")
    tp.post(f"/api/installations/{WK}/start")
    tp.until(lambda s: inst(s, WK)["runtime"]["state"] == "RUNNING", 15, "running")
    r = action(tp, WK, TASS, "after upgrade")
    rec("B", r["outcome"] == "COMPLETED" and len(worker_pids(WK)) == 1, "the existing grant still works: request COMPLETED, one worker process")
    rec("B", ledger.exists() and ledger.read_bytes().startswith(led_before[:1]) and len(ledger.read_bytes()) >= len(led_before),
        "worker's single-use ticket ledger kept (data folder untouched)")
    lib = {x.get("feature_id"): x for x in tp.state()["library"]}
    rec("B", lib.get("thought") and not lib["thought"].get("blocked") and lib.get(FEATURE, {}).get("installed_as") == [WK],
        "Library: worker shows as installed; Thought is offered (pinned) beside it")
    st, j = tp.post("/api/library/thought/install")
    approve(j["approval_url"])
    s = tp.until(lambda s: any(x["record"]["feature_id"] == "thought" and x["identity"] and x["identity"]["installation"]["state"] == "INSTALLED" for x in s["installations"]), 30, "thought installed")
    TH = [x["record"]["installation_id"] for x in s["installations"] if x["record"]["feature_id"] == "thought"][0]
    rec("B", TH != WK and len(s["identity_installations"]) == 2 and inst(s, TH)["identity"]["active_grants"] == 0 and inst(s, WK)["identity"]["active_grants"] == 1,
        "Thought installed as a SEPARATE installation with zero grants; the worker's grant is unaffected", TH)
    r = action(tp, TH, TASS, "goldshire: x")
    rec("B", r["outcome"] == "DENIED_BY_IDENTITY", "the worker's grant gives Thought nothing (separate services, separate grants)")
    tp.post(f"/api/installations/{WK}/stop")
    tp.until(lambda s: inst(s, WK)["runtime"]["state"] == "STOPPED", 20, "stopped")

    # ---- Phase A again: the accepted build can still open the profile ----
    stop("processor")
    stop("identity")
    PHASE["b"] = False
    BIN["processor"] = "processor-accepted"
    start_identity()
    tp = TP(start_processor())
    s = tp.until(lambda s: s["verified"] and inst(s, WK) and inst(s, WK)["identity"] is not None, 40, "reconnected (back)")
    th = inst(s, TH)
    rec("BACK", s["verified"] and inst(s, WK)["files"] == "OK" and th is not None and th["record"]["state"] == "INSTALLED",
        "rolling back to the accepted build works: same profile, worker intact; Thought's record is kept (shown, but not startable without its pin)")
    st, j = tp.post(f"/api/installations/{TH}/start")
    rec("BACK", st == 409 and "UNTRUSTED_EXECUTION_BLOCKED" in j.get("error", "") + json.dumps(j), "without the Thought pin the old build refuses to start Thought (fails closed)", safe(j.get("error", ""))[:70])


def action(tp, ins, subject, text):
    st, j = tp.post(f"/api/installations/{ins}/action", {"subject": subject, "capability": WCAP if ins and FEATURE in json.dumps(inst(tp.state(), ins)["record"]["feature_id"]) else "thought.event.submit", "input": text})
    return j.get("result") or {"outcome": "HTTP %d" % st, "detail": safe(j.get("error", ""))}


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
    lines = [f"Profile reuse (accepted secure-worker build → Thought build) — {time.strftime('%Y-%m-%d %H:%M:%S %Z')}",
             "evidence: " + json.dumps(evidence, sort_keys=True), ""] + [f"{w} [{a}] {n}" + (f"  — {d}" if d != "" else "") for a, w, n, d in results]
    lines.append(f"\nRESULT: {'PASS' if not failed else 'FAIL'} ({len(results)} checks, {len(failed)} failed)")
    (W / "p1-profile-upgrade-report.txt").write_text("\n".join(lines) + "\n")
    print(lines[-1].strip())
    sys.exit(0 if not failed else 1)
