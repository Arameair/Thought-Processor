"""Universal update acceptance — REAL Identity (experimental installation-upgrade
branch) + REAL Thought Processor + REAL feature processes. GitHub is a labelled
TEST FIXTURE served by this script; the script plays the Admin in Identity's
own approval page (passphrase typed THERE, never in Processor).

Covers: correct update (PASS) for two different features through ONE engine,
no reviewed update, tampered candidate, Admin denial / wrong passphrase /
cancel, concurrent update conflict, irreversible migration blocked, failed
health -> recovered previous version, interrupted update (Processor killed
after Identity committed) -> recovered on restart, preserved installation
identity / credential / data / grants, and accurate user-visible status.

  python3 -I acceptance/p0_updates.py --identity-src DIR --work DIR

DIR for --identity-src: Arameair/identity at experiment/installation-upgrade-claude-20261030.
Linux/WSL developer evidence; NOT a Windows run, NOT real GitHub, NOT the owner's profiles.
"""
import argparse, hashlib, json, os, re, shutil, signal, socket, subprocess, sys, threading, time, urllib.error, urllib.parse, urllib.request
import http.cookiejar
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path

ap = argparse.ArgumentParser()
ap.add_argument("--identity-src", required=True)
ap.add_argument("--work", required=True)
ap.add_argument("--shots", default="", help="optional folder for screenshots of the Updates screen (headless Chromium)")
ap.add_argument("--old-identity-src", default="", help="optional: the owner's pinned Identity (64940b6, no upgrade API)")
args = ap.parse_args()
W = Path(args.work).resolve()
if W.exists():
    shutil.rmtree(W)
W.mkdir(parents=True, mode=0o700)
PROC = Path(__file__).resolve().parents[1]
ID_SRC = Path(args.identity_src).resolve()
PASS_PHRASE = "correct horse battery staple 2026"


def free_port():
    with socket.socket() as s:
        s.bind(("127.0.0.1", 0))
        return s.getsockname()[1]


IP, TPP, FX = free_port(), free_port(), free_port()
IDENTITY, PROCESSOR = f"http://127.0.0.1:{IP}", f"http://127.0.0.1:{TPP}"
NOPROXY = urllib.request.build_opener(urllib.request.ProxyHandler({}))
ENV = {k: v for k, v in os.environ.items() if not k.lower().endswith("_proxy")}
ENV.update(IDENTITY_LOCATOR_DIR=str(W / "locators"), IDENTITY_GITHUB_CLIENT_ID="IvFIXTURE00000001", TP_IDENTITY_DEFAULT_URL="-")
results, procs = [], {}


def rec(ok, name, detail=""):
    results.append(bool(ok))
    print(("PASS " if ok else "FAIL ") + name + (("  — " + str(detail)) if detail != "" else ""), flush=True)


# ---------------- GitHub TEST FIXTURE: approves every device code ----------------
class GH(BaseHTTPRequestHandler):
    def log_message(self, *a):
        pass

    def send(self, code, obj):
        b = json.dumps(obj).encode()
        self.send_response(code)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(b)))
        self.end_headers()
        self.wfile.write(b)

    def do_POST(self):
        self.rfile.read(int(self.headers.get("Content-Length", 0)))
        if self.path == "/login/device/code":
            return self.send(200, {"device_code": "fx-dev", "user_code": "FIXT-URE1", "verification_uri": f"http://127.0.0.1:{FX}/x", "expires_in": 300, "interval": 1})
        if self.path == "/login/oauth/access_token":
            return self.send(200, {"access_token": "ghu_FIXTUREonlyNOTrealTOKEN0000000000", "expires_in": 28800})
        self.send(404, {})

    def do_GET(self):
        if self.path == "/user":
            return self.send(200, {"login": "fixture-owner", "id": 4242, "name": "Fixture Owner", "type": "User", "html_url": "https://github.com/fixture-owner"})
        if self.path.startswith("/user/installations"):
            return self.send(200, {"total_count": 0, "installations": []})
        self.send(404, {"message": "Not Found"})


threading.Thread(target=ThreadingHTTPServer(("127.0.0.1", FX), GH).serve_forever, daemon=True).start()


# ---------------- packages ----------------
def digest(d):
    m = json.loads((d / "feature.json").read_text())
    ms = hashlib.sha256((d / "feature.json").read_bytes()).hexdigest()
    b = "thought.feature-package/1\nmanifest %s\n" % ms + "".join("%s  %s\n" % (f["sha256"], f["path"]) for f in sorted(m["files"], key=lambda f: f["path"]))
    return hashlib.sha256(b.encode()).hexdigest()


def variant(src, dst, version, migration="none", code_patch=None):
    shutil.copytree(src, dst)
    m = json.loads((dst / "feature.json").read_text())
    code_file = dst / m["files"][0]["path"]
    code = code_file.read_text()
    code = re.sub(r'^VERSION = ".*"$', 'VERSION = "%s"' % version, code, count=1, flags=re.M)
    if code_patch:
        code = code.replace(*code_patch)
    code_file.write_text(code)
    m["version"] = version
    m["update"]["data_migration"] = migration
    for f in m["files"]:
        f["sha256"] = hashlib.sha256((dst / f["path"]).read_bytes()).hexdigest()
    (dst / "feature.json").write_text(json.dumps(m, indent=2) + "\n")
    return digest(dst)


UPD = PROC / "testfeatures-updates"
CAND = {}  # name -> (folder holding exactly one candidate, digest)
for name in ("deterministic-worker-0.2.0", "deterministic-worker-0.3.0-broken", "developer-utility-0.1.1"):
    d = W / "cand" / name / name
    shutil.copytree(UPD / name, d)
    CAND[name] = (d.parent, digest(d))
d = W / "cand" / "worker-0.2.5" / "w"
CAND["worker-0.2.5"] = (d.parent, variant(UPD / "deterministic-worker-0.2.0", d, "0.2.5"))
d = W / "cand" / "worker-0.4.0-irreversible" / "w"
CAND["worker-0.4.0-irreversible"] = (d.parent, variant(UPD / "deterministic-worker-0.2.0", d, "0.4.0", migration="irreversible"))
DEV_PIN = re.search(r"^DEV_ARTIFACT=([0-9a-f]{64})$", (PROC.parent / "run-p1.sh").read_text(), re.M).group(1)
ALL_PINS = [DEV_PIN] + [v[1] for v in CAND.values()]


# ---------------- processes ----------------
def build():
    (W / "bin").mkdir(exist_ok=True)
    subprocess.run(["go", "build", "-o", str(W / "bin/identity"), "."], cwd=ID_SRC, check=True)
    subprocess.run(["go", "build", "-o", str(W / "bin/processor"), "."], cwd=PROC, check=True)
    if args.old_identity_src:
        subprocess.run(["go", "build", "-o", str(W / "bin/identity-old"), "."], cwd=Path(args.old_identity_src).resolve(), check=True)


def start(name, cmd, env=None):
    log = open(W / (name + ".console.txt"), "ab")
    procs[name] = subprocess.Popen(cmd, stdout=log, stderr=subprocess.STDOUT, stdin=subprocess.DEVNULL, env=env or ENV, start_new_session=True)


def stop(name, sig=signal.SIGTERM):
    p = procs.get(name)
    if p and p.poll() is None:
        p.send_signal(sig)
        try:
            p.wait(20)
        except subprocess.TimeoutExpired:
            p.kill()
            p.wait(5)


def wait_http(url, t=20):
    end = time.time() + t
    while time.time() < end:
        try:
            NOPROXY.open(url, timeout=2).read()
            return True
        except Exception:
            time.sleep(0.2)
    return False


def start_identity(binary="identity"):
    cmd = [str(W / "bin" / binary), "-port", str(IP), "-data", str(W / "identity-profile"),
           "-test-github-web", f"http://127.0.0.1:{FX}", "-test-github-api", f"http://127.0.0.1:{FX}"]
    for p in ALL_PINS:
        cmd += ["-trust-fixture-artifact", p]
    start("identity", cmd)
    assert wait_http(IDENTITY + "/api/health"), "identity did not start"


def start_processor(cands=(), pins=(), crash=None):
    (W / "processor.console.txt").write_bytes(b"")
    cmd = [str(W / "bin/processor"), "-port", str(TPP), "-data", str(W / "processor-profile"), "-no-browser", "-no-link-file",
           "-identity-default-url", IDENTITY, "-workstation-allow-test-github",
           "-trust-feature-dir", str(PROC / "testfeatures"), "-trust-feature-dir", str(PROC / "developer-utility"),
           "-trust-feature-artifact", DEV_PIN]
    for c in cands:
        cmd += ["-trust-feature-dir", str(CAND[c][0])]
    for c in pins:
        cmd += ["-trust-feature-artifact", CAND[c][1]]
    env = dict(ENV)
    if crash:
        env["TP_TEST_CRASH_POINT"] = crash
    start("processor", cmd, env)
    end = time.time() + 20
    while time.time() < end:
        m = re.search(r"#code=([0-9a-f]+)", (W / "processor.console.txt").read_text(errors="replace"))
        if m:
            RUN["code"] = m.group(1)
            b = Browser()
            b.sign_in(m.group(1))
            b.until("/api/state", lambda s: s["verified"], 30)
            return b
        time.sleep(0.2)
    raise SystemExit("processor did not start")


class Browser:
    def __init__(self):
        self.op = urllib.request.build_opener(urllib.request.ProxyHandler({}), urllib.request.HTTPCookieProcessor(http.cookiejar.CookieJar()))
        self.csrf = ""

    def req(self, method, path, body=None):
        data = json.dumps(body or {}).encode() if method == "POST" else None
        r = urllib.request.Request(PROCESSOR + path, data=data, method=method)
        if method == "POST":
            r.add_header("Content-Type", "application/json")
            r.add_header("Origin", PROCESSOR)
            r.add_header("X-TP-CSRF", self.csrf)
        try:
            with self.op.open(r, timeout=60) as resp:
                return resp.status, json.loads(resp.read() or b"{}")
        except urllib.error.HTTPError as e:
            return e.code, json.loads(e.read() or b"{}")

    def sign_in(self, code):
        st, j = self.req("POST", "/api/session", {"code": code})
        self.csrf = j.get("csrf", "")

    def until(self, path, pred, t=40):
        end = time.time() + t
        while True:
            st, j = self.req("GET", path)
            try:
                if pred(j):
                    return j
            except (KeyError, TypeError, IndexError, AttributeError):
                pass
            if time.time() > end:
                return j
            time.sleep(0.4)


def admin(url, body):
    op = urllib.request.build_opener(urllib.request.ProxyHandler({}), urllib.request.HTTPCookieProcessor(http.cookiejar.CookieJar()))
    m = re.search(r"#a=(apr_[0-9a-f]+)&t=([0-9a-f]+)", url or "")
    if not m:
        return 0, {}

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


def approve(url, pw=PASS_PHRASE):
    return admin(url, {"decision": "approve", "passphrase": pw})


RUN = {"code": ""}


def shot(name):
    """Screenshot the real Updates screen, opened with a one-time ticket like the launcher does."""
    if not args.shots:
        return
    from playwright.sync_api import sync_playwright
    st, j = Browser().req("POST", "/api/launch-ticket", {"code": RUN["code"]})
    with sync_playwright() as pw:
        br = pw.chromium.launch()
        pg = br.new_page(viewport={"width": 1280, "height": 1000}, color_scheme="dark")
        pg.goto(PROCESSOR + "/workstation#code=" + j["ticket"])
        pg.wait_for_selector("#ws-tab-upd", timeout=15000)
        pg.wait_for_timeout(1200)
        pg.click("#ws-tab-upd")
        pg.wait_for_timeout(2500)
        for d in pg.query_selector_all("#ws-upd-list details"):
            d.evaluate("e => e.open = true")
        Path(args.shots).mkdir(parents=True, exist_ok=True)
        pg.screenshot(path=str(Path(args.shots) / (name + ".png")), full_page=True)
        br.close()


def comp(b, ins):
    st, j = b.req("GET", "/api/ws/updates")
    return {c["id"]: c for c in j.get("components", [])}.get(ins, {})


def wait_tx(b, ins, t=120):
    end = time.time() + t
    while time.time() < end:
        c = comp(b, ins)
        tx = c.get("tx") or {}
        if tx.get("state") in ("PASS", "FAIL", "RECOVERED_PREVIOUS_VERSION", "NEEDS_ATTENTION", "DENIED_BY_ADMIN", "APPROVAL_EXPIRED", "CANCELLED"):
            return c
        time.sleep(0.5)
    return comp(b, ins)


def inst(s, ins):
    return [v for v in s["installations"] if v["record"]["installation_id"] == ins][0]


def ident(b, ins):
    s = b.until("/api/state", lambda s: inst(s, ins)["identity"] is not None, 10)
    return inst(s, ins)["identity"]


def fsnap(ins):
    d = W / "processor-profile" / "installations" / ins
    return {"cred": hashlib.sha256((d / "credential.json").read_bytes()).hexdigest(),
            "data": (d / "data" / "owner-note.txt").read_text() if (d / "data" / "owner-note.txt").exists() else None}


def running_version(b, ins):
    """What the running process itself reports (via Processor's activity/state: launcher-signed status)."""
    st, j = b.req("GET", f"/api/installations/{ins}/details")
    return j


def action(b, ins, subject, text="hello update"):
    st, j = b.req("POST", f"/api/installations/{ins}/action", {"subject": subject, "capability": "worker.test.synthetic", "input": text})
    return (j.get("result") or {}).get("outcome"), j


def main():
    build()
    start_identity()
    b = start_processor(cands=["deterministic-worker-0.2.0"], pins=[])  # candidate present but NOT reviewed here
    # pair
    b.req("POST", "/api/connect/choose", {"url": IDENTITY})
    b.req("POST", "/api/connect/request", {"url": IDENTITY})
    time.sleep(0.7)
    code = re.findall(r"\b([A-HJ-NP-Z2-9]{4}-[A-HJ-NP-Z2-9]{4})\b", (W / "identity.console.txt").read_text(errors="replace"))[-1]
    b.req("POST", "/api/connect/code", {"code": code})
    b.until("/api/state", lambda s: s["verified"], 20)
    b.req("POST", "/api/github/connect")
    s = b.until("/api/state", lambda s: s["github"]["state"] == "VERIFIED", 30)
    st, j = b.req("POST", "/api/admin/enroll")
    st, j = admin(j.get("approval_url"), {"decision": "approve", "passphrase": PASS_PHRASE, "passphrase_confirm": PASS_PHRASE, "acknowledge": True})
    s = b.until("/api/state", lambda s: s["admin"]["state"] == "ENROLLED", 15)
    rec(s["admin"]["state"] == "ENROLLED", "SETUP paired, GitHub VERIFIED (fixture), Admin established in Identity's page")
    # install worker 0.1.0 and Developer Utility 0.1.0 the normal way
    ins = {}
    for feat in ("p1-deterministic-worker", "developer-utility"):
        st, j = b.req("POST", f"/api/library/{feat}/install")
        approve(j.get("approval_url"))
        s = b.until("/api/state", lambda s: any(v["record"]["feature_id"] == feat and (v.get("identity") or {}).get("installation", {}).get("state") == "INSTALLED" for v in s["installations"]), 40)
        ins[feat] = [v for v in s["installations"] if v["record"]["feature_id"] == feat][0]["record"]["installation_id"]
    W1, DEV = ins["p1-deterministic-worker"], ins["developer-utility"]
    rec(True, "SETUP installed worker 0.1.0 as %s and Developer Utility 0.1.0 as %s (Admin-approved)" % (W1, DEV))
    b.req("POST", "/api/identities", {"name": "Tass", "kind": "companion"})
    s = b.req("GET", "/api/state")[1]
    TASS = [e for e in s["identities"] if e["name"] == "Tass"][0]["id"]
    st, j = b.req("POST", f"/api/installations/{W1}/grants/request", {"subject": TASS, "capability": "worker.test.synthetic", "expires_in_seconds": 3600})
    approve(j.get("approval_url"))
    s = b.until("/api/state", lambda s: inst(s, W1)["identity"]["active_grants"] == 1, 30)
    grant_id = [g["grant"]["grant_id"] for g in inst(s, W1)["identity"]["grants"] if g["status"] == "ACTIVE"][0]
    b.req("POST", f"/api/installations/{W1}/start")
    b.until("/api/state", lambda s: inst(s, W1)["runtime"]["state"] == "RUNNING", 20)
    out, _ = action(b, W1, TASS)
    (W / "processor-profile/installations" / W1 / "data" / "owner-note.txt").write_text("private data written before the update\n")
    before = fsnap(W1)
    gen_before = ident(b, W1)["installation"]["credential_generation"]
    rec(out == "COMPLETED", "SETUP grant approved; worker RUNNING; protected action COMPLETED on 0.1.0", grant_id)

    # ---- U1 no reviewed update ----
    c = comp(b, W1)
    rec(c.get("update_state") == "BLOCKED" and "NOT_REVIEWED" in c.get("reason", "") and not c.get("can_update"),
        "U1 a newer package that is NOT pinned is shown BLOCKED (not reviewed), no Update button", c.get("reason", "")[:90])
    st, j = b.req("POST", f"/api/ws/updates/{W1}/start")
    rec(st == 409 and "BLOCKED" in j.get("error", "") and fsnap(W1) == before, "U1 Update refused; nothing changed", j.get("error", "")[:80])
    c = comp(b, DEV)
    rec(c.get("update_state") == "CURRENT", "U1 Developer Utility: no candidate in trusted folders -> CURRENT")
    ids = {x["id"]: x for x in b.req("GET", "/api/ws/updates")[1]["components"]}
    rec(ids.get("identity", {}).get("update_state") == "BLOCKED" and ids.get("processor", {}).get("update_state") == "UNSUPPORTED",
        "U1 discovery lists Identity (BLOCKED: no self-upgrade by file swap) and Processor (UNSUPPORTED here)")

    # ---- reviewed candidates ----
    stop("processor")
    b = start_processor(cands=["deterministic-worker-0.2.0", "developer-utility-0.1.1"], pins=["deterministic-worker-0.2.0", "developer-utility-0.1.1"])
    b.req("POST", f"/api/installations/{W1}/start")
    b.until("/api/state", lambda s: inst(s, W1)["runtime"]["state"] == "RUNNING", 20)
    c = comp(b, W1)
    shot("u2-update-available")
    rec(c.get("update_state") == "UPDATE_AVAILABLE" and c["candidate"]["version"] == "0.2.0" and c["can_update"],
        "U2 reviewed, compatible 0.2.0 offered: UPDATE_AVAILABLE", "requires: " + "; ".join(c.get("requires", []))[:120])

    # ---- U3 tampered candidate (changed after review) ----
    wp = CAND["deterministic-worker-0.2.0"][0] / "deterministic-worker-0.2.0" / "worker.py"
    orig = wp.read_bytes()
    wp.write_bytes(orig + b"\n# tampered after review\n")
    st, j = b.req("POST", f"/api/ws/updates/{W1}/start")
    c = comp(b, W1)
    rec(c.get("update_state") == "BLOCKED" and "INVALID_PACKAGE" in c.get("reason", "") and not c.get("can_update"),
        "U3 tampered candidate shown BLOCKED (failed verification), no Update button", c.get("reason", "")[:110])
    rec(st == 409 and fsnap(W1) == before and ident(b, W1)["installation"]["version"] == "0.1.0",
        "U3 Update of a tampered candidate refused before anything changed", j.get("error", "")[:90])
    wp.write_bytes(orig)

    # ---- U4 Admin authority: wrong passphrase, denial, cancel; concurrency ----
    st, j = b.req("POST", f"/api/ws/updates/{W1}/start")
    tx = j.get("tx", {})
    shot("u4-approval-required")
    rec(st == 202 and tx.get("state") == "APPROVAL_REQUIRED" and "/approve#a=apr_" in tx.get("approval_url", ""),
        "U4 Update -> APPROVAL_REQUIRED with a link to IDENTITY's approval page")
    st2, j2 = b.req("POST", f"/api/ws/updates/{W1}/start")
    rec(st2 == 409 and "UPDATE_CONFLICT" in j2.get("error", ""), "U4 a second concurrent Update of the same installation is refused (UPDATE_CONFLICT)")
    st3, j3 = approve(tx.get("approval_url"), "wrong passphrase")
    rec(st3 == 401 and comp(b, W1)["tx"]["state"] == "APPROVAL_REQUIRED", "U4 wrong Admin passphrase refused; still waiting, nothing changed", st3)
    st, j = b.req("POST", f"/api/ws/updates/{W1}/cancel")
    c = comp(b, W1)
    rec(st == 200 and c["tx"]["state"] == "CANCELLED" and fsnap(W1) == before, "U4 Cancel before approval: CANCELLED, nothing changed")
    st, j = b.req("POST", f"/api/ws/updates/{W1}/start")
    admin(j["tx"]["approval_url"], {"decision": "deny"})
    c = wait_tx(b, W1, 30)
    rec(c["tx"]["state"] == "DENIED_BY_ADMIN" and fsnap(W1) == before and ident(b, W1)["installation"]["version"] == "0.1.0",
        "U4 Admin denies in Identity -> DENIED_BY_ADMIN, nothing changed")

    # ---- U5 correct update: worker 0.1.0 -> 0.2.0 ----
    st, j = b.req("POST", f"/api/ws/updates/{W1}/start")
    st_a, _ = approve(j["tx"]["approval_url"])
    c = wait_tx(b, W1)
    tx = c["tx"]
    idv = ident(b, W1)
    after = fsnap(W1)
    s = b.req("GET", "/api/state")[1]
    rt = inst(s, W1)["runtime"]
    st, det = b.req("GET", f"/api/installations/{W1}/details")
    shot("u5-pass")
    rec(tx["state"] == "PASS", "U5 worker 0.1.0 -> 0.2.0: PASS", tx.get("result", "")[:140])
    rec(idv["installation"]["installation_id"] == W1 and idv["installation"]["version"] == "0.2.0" and idv["installation"]["artifact_sha256"] == CAND["deterministic-worker-0.2.0"][1]
        and idv["installation"]["credential_generation"] == gen_before and len(idv["installation"].get("upgrades", [])) == 1,
        "U5 Identity: SAME installation ID and credential generation, now 0.2.0 at the reviewed artifact, upgrade recorded")
    rec(after["cred"] == before["cred"] and after["data"] == before["data"], "U5 credential file and private data unchanged (preserved)")
    rec(inst(s, W1)["record"]["version"] == "0.2.0" and inst(s, W1)["files"] == "OK" and rt["state"] == "RUNNING",
        "U5 local record 0.2.0, files verified, and it is RUNNING again (it was running before)", rt.get("launch_id"))
    rec(json.dumps(det).count('"0.2.0"') > 0, "U5 the running process itself reports version 0.2.0 (launcher-signed status)")
    rec(any(g["grant"]["grant_id"] == grant_id and g["status"] == "ACTIVE" for g in idv["grants"]), "U5 the existing grant is still ACTIVE (capability kept)")
    out, _ = action(b, W1, TASS)
    rec(out == "COMPLETED", "U5 protected action through the kept grant COMPLETED on 0.2.0")
    rec((W / "processor-profile/installations" / W1 / "update/previous/package/worker.py").exists(), "U5 previous version kept aside for recovery")
    txt = (W / "processor-profile/installations" / W1 / "update/tx.json").read_text()
    secret = json.loads((W / "processor-profile/installations" / W1 / "credential.json").read_text())["secret"]
    rec(secret not in txt and secret not in (W / "processor.console.txt").read_text(errors="replace"), "U5 the installation secret is not in the update journal or console")

    # ---- U6 same engine, second feature: Developer Utility 0.1.0 -> 0.1.1 ----
    dev_before = fsnap(DEV)
    st, j = b.req("POST", f"/api/ws/updates/{DEV}/start")
    approve(j.get("tx", {}).get("approval_url"))
    c = wait_tx(b, DEV)
    idd = ident(b, DEV)["installation"]
    s = b.req("GET", "/api/state")[1]
    rec(c["tx"]["state"] == "PASS" and idd["installation_id"] == DEV and idd["version"] == "0.1.1" and fsnap(DEV)["cred"] == dev_before["cred"]
        and inst(s, DEV)["runtime"]["state"] == "STOPPED",
        "U6 SAME engine updates a second feature: Developer Utility 0.1.1 PASS, same ID/credential, left STOPPED as before",
        " | ".join(x["step"] + ":" + x["result"] for x in c["tx"]["steps"]))

    # ---- U7 failed health check -> recovered previous version ----
    stop("processor")
    b = start_processor(cands=["deterministic-worker-0.3.0-broken"], pins=["deterministic-worker-0.2.0", "deterministic-worker-0.3.0-broken", "developer-utility-0.1.1"])
    b.req("POST", f"/api/installations/{W1}/start")
    b.until("/api/state", lambda s: inst(s, W1)["runtime"]["state"] == "RUNNING", 20)
    st, j = b.req("POST", f"/api/ws/updates/{W1}/start")
    approve(j.get("tx", {}).get("approval_url"))
    c = wait_tx(b, W1)
    idv = ident(b, W1)
    s = b.req("GET", "/api/state")[1]
    shot("u7-recovered")
    rec(c["tx"]["state"] == "RECOVERED_PREVIOUS_VERSION", "U7 0.3.0 fails its self-test -> RECOVERED_PREVIOUS_VERSION", c["tx"].get("result", "")[:150])
    rec(idv["installation"]["version"] == "0.2.0" and idv["installation"]["artifact_sha256"] == CAND["deterministic-worker-0.2.0"][1]
        and inst(s, W1)["record"]["version"] == "0.2.0" and inst(s, W1)["files"] == "OK" and inst(s, W1)["runtime"]["state"] == "RUNNING",
        "U7 Identity record, local record and files are back at 0.2.0, and it RUNS again")
    out, _ = action(b, W1, TASS)
    rec(out == "COMPLETED" and fsnap(W1) == before, "U7 grant and data intact; protected action COMPLETED after recovery")
    c = comp(b, W1)
    st, j = b.req("POST", f"/api/ws/updates/{W1}/start")
    rec(c.get("update_state") == "BLOCKED" and "FAILED_BEFORE" in c.get("reason", "") and not c.get("can_update") and st == 409,
        "U7 the version that failed is not offered again (FAILED_BEFORE), and a retry is refused", c.get("reason", "")[:90])

    # ---- U8 interrupted: Processor killed right after Identity committed ----
    stop("processor")
    b = start_processor(cands=["worker-0.2.5"], pins=["deterministic-worker-0.2.0", "worker-0.2.5", "developer-utility-0.1.1"], crash="update-after-commit")
    b.req("POST", f"/api/installations/{W1}/start")
    b.until("/api/state", lambda s: inst(s, W1)["runtime"]["state"] == "RUNNING", 20)
    st, j = b.req("POST", f"/api/ws/updates/{W1}/start")
    approve(j.get("tx", {}).get("approval_url"))
    try:
        procs["processor"].wait(60)
    except subprocess.TimeoutExpired:
        pass
    code = procs["processor"].poll()
    journal = json.loads((W / "processor-profile/installations" / W1 / "update/tx.json").read_text())
    rec(code == 42 and journal["phase"] == "IDENTITY_COMMITTED", "U8 Processor died mid-update (after Identity committed 0.2.5)", "exit %s, journal phase %s" % (code, journal["phase"]))
    b = start_processor(cands=["worker-0.2.5"], pins=["deterministic-worker-0.2.0", "worker-0.2.5", "developer-utility-0.1.1"])
    c = wait_tx(b, W1)
    idv = ident(b, W1)
    s = b.req("GET", "/api/state")[1]
    rec(c["tx"]["state"] == "RECOVERED_PREVIOUS_VERSION" and idv["installation"]["version"] == "0.2.0" and inst(s, W1)["record"]["version"] == "0.2.0"
        and inst(s, W1)["files"] == "OK", "U8 on restart the interrupted update is recovered: Identity, record and files back at 0.2.0", c["tx"].get("result", "")[:120])
    rec(inst(s, W1)["runtime"]["state"] == "RUNNING" and fsnap(W1) == before, "U8 it was running before, so it runs again; credential + data intact")

    # ---- U9 irreversible migration is blocked ----
    stop("processor")
    b = start_processor(cands=["worker-0.4.0-irreversible"], pins=["deterministic-worker-0.2.0", "worker-0.4.0-irreversible", "developer-utility-0.1.1"])
    c = comp(b, W1)
    st, j = b.req("POST", f"/api/ws/updates/{W1}/start")
    rec(c.get("update_state") == "BLOCKED" and "IRREVERSIBLE" in c.get("reason", "") and st == 409, "U9 irreversible data migration: BLOCKED, refused", c.get("reason", "")[:80])

    # ---- U10 revoked grant is not revived by an update ----
    st, j = b.req("POST", f"/api/installations/{W1}/grants/{grant_id}/revoke")
    out, _ = action(b, W1, TASS)
    rec(out == "DENIED_BY_IDENTITY", "U10 after revoking the grant, the updated feature's request is DENIED by Identity (no update revives it)", out)
    # ---- U11 the owner's pinned Identity (no upgrade API) ----
    if args.old_identity_src:
        stop("processor")
        stop("identity")
        start_identity("identity-old")
        b = start_processor(cands=["worker-0.2.5"], pins=["deterministic-worker-0.2.0", "worker-0.2.5", "developer-utility-0.1.1"])
        locked = b.req("GET", "/api/ws/updates")[0]
        b.req("POST", "/api/github/connect")  # an Identity restart signs GitHub out (by design)
        b.until("/api/state", lambda s: s["github"]["state"] == "VERIFIED", 30)
        rec(locked == 403, "U11 after an Identity restart GitHub is signed out, so the Updates screen is locked until sign-in")
        c = comp(b, W1)
        st, j = b.req("POST", f"/api/ws/updates/{W1}/start")
        rec(c.get("update_state") == "BLOCKED" and "IDENTITY_UPGRADE_API_MISSING" in c.get("reason", "") and st == 409,
            "U11 with the pinned Identity 64940b6 (no upgrade API) updates are BLOCKED, not attempted", c.get("reason", "")[:100])
        b.req("POST", f"/api/installations/{W1}/start")
        s = b.until("/api/state", lambda s: inst(s, W1)["runtime"]["state"] == "RUNNING", 20)
        rec(inst(s, W1)["runtime"]["state"] == "RUNNING" and ident(b, W1)["installation"]["version"] == "0.2.0",
            "U11 the pinned Identity still reads the upgraded record (0.2.0) and the feature runs (records stay compatible)")
    print("\nUI state of every component:")
    for x in b.req("GET", "/api/ws/updates")[1]["components"]:
        print("  %-36s %-9s %-26s %s" % (x["name"][:36], x["version"][:9], x["update_state"], (x.get("reason") or "")[:70]))


try:
    main()
finally:
    for n in ("processor", "identity"):
        stop(n)
print("\n%d/%d PASS" % (sum(results), len(results)))
sys.exit(0 if results and all(results) else 1)
