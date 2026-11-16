"""Workstation + secure-open acceptance — REAL Identity + REAL Thought Processor
processes, with GitHub replaced by a labelled TEST FIXTURE served by this
script (so WAITING, DENIED, EXPIRED and VERIFIED can each be forced).

Checks: no Processor link file, single-use launch ticket, no secret in
Processor's process arguments, GitHub states propagating Identity ->
Processor -> Workstation gate, read-only repo/file views, path guards,
allowlisted test execution, fail-closed when Identity dies.

  python3 -I acceptance/p0_workstation.py --identity-src DIR --tass-src DIR --work DIR

Linux/WSL developer evidence; NOT a Windows run and NOT real GitHub. Pairing
codes and links are read from the disposable work folder and never printed.
Exit 0 only when every check PASSes.
"""
import argparse, json, os, re, signal, socket, subprocess, sys, threading, time, urllib.error, urllib.parse, urllib.request
import http.cookiejar
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path

ap = argparse.ArgumentParser()
ap.add_argument("--identity-src", required=True)
ap.add_argument("--tass-src", required=True)
ap.add_argument("--work", required=True)
args = ap.parse_args()
W = Path(args.work).resolve()
W.mkdir(parents=True, exist_ok=True, mode=0o700)
PROC_REPO = Path(__file__).resolve().parents[2]
ID_SRC, TASS_SRC = Path(args.identity_src).resolve(), Path(args.tass_src).resolve()


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
    results.append(ok)
    print(("PASS " if ok else "FAIL ") + name + (("  — " + detail) if detail else ""), flush=True)


# ---------------- GitHub TEST FIXTURE with a forced outcome ----------------
FIX = {"outcome": "pending", "n": 0}


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
            FIX["n"] += 1
            return self.send(200, {"device_code": "fx-device-%d" % FIX["n"], "user_code": "FIXT-URE%d" % FIX["n"],
                                   "verification_uri": f"http://127.0.0.1:{FX}/login/device", "expires_in": 300, "interval": 1})
        if self.path == "/login/oauth/access_token":
            o = FIX["outcome"]
            if o == "approve":
                return self.send(200, {"access_token": "ghu_FIXTUREonlyNOTrealTOKEN0000000000", "expires_in": 28800})
            return self.send(200, {"error": {"pending": "authorization_pending", "deny": "access_denied", "expire": "expired_token"}[o]})
        self.send(404, {})

    def do_GET(self):
        if self.path == "/user":
            return self.send(200, {"login": "fixture-owner", "id": 4242, "name": "Fixture Owner", "type": "User",
                                   "html_url": "https://github.com/fixture-owner"})
        if self.path.startswith("/user/installations"):
            return self.send(200, {"total_count": 0, "installations": []})
        self.send(404, {"message": "Not Found"})


threading.Thread(target=ThreadingHTTPServer(("127.0.0.1", FX), GH).serve_forever, daemon=True).start()


# ---------------- processes ----------------
def build():
    (W / "bin").mkdir(exist_ok=True)
    subprocess.run(["go", "build", "-o", str(W / "bin/identity"), "."], cwd=ID_SRC, check=True)
    subprocess.run(["go", "build", "-o", str(W / "bin/processor"), "."], cwd=PROC_REPO / "processor", check=True)


def start(name, cmd):
    log = open(W / (name + ".console.txt"), "ab")
    procs[name] = subprocess.Popen(cmd, stdout=log, stderr=subprocess.STDOUT, stdin=subprocess.DEVNULL, env=ENV, start_new_session=True)


def stop(name, sig=signal.SIGTERM):
    p = procs.get(name)
    if p and p.poll() is None:
        p.send_signal(sig)
        try:
            p.wait(10)
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


def start_identity():
    start("identity", [str(W / "bin/identity"), "-port", str(IP), "-data", str(W / "identity-profile"),
                       "-test-github-web", f"http://127.0.0.1:{FX}", "-test-github-api", f"http://127.0.0.1:{FX}"])
    assert wait_http(IDENTITY + "/api/health"), "identity did not start"


def start_processor(allow_fixture=True):
    (W / "processor.console.txt").write_bytes(b"")
    cmd = [str(W / "bin/processor"), "-port", str(TPP), "-data", str(W / "processor-profile"), "-no-browser", "-no-link-file",
           "-identity-default-url", IDENTITY, "-workstation-repo", "processor=" + str(PROC_REPO),
           "-workstation-repo", "identity=" + str(ID_SRC), "-workstation-repo", "tass=" + str(TASS_SRC)]
    if allow_fixture:
        cmd.append("-workstation-allow-test-github")
    start("processor", cmd)
    end = time.time() + 20
    while time.time() < end:
        m = re.search(r"open\s*:\s*(http://\S+#code=([0-9a-f]+))", (W / "processor.console.txt").read_text(errors="replace"))
        if m:
            return m.group(2)
        time.sleep(0.2)
    raise SystemExit("processor did not announce its link")


class Browser:
    def __init__(self):
        self.op = urllib.request.build_opener(urllib.request.ProxyHandler({}), urllib.request.HTTPCookieProcessor(http.cookiejar.CookieJar()))
        self.csrf = ""

    def req(self, method, path, body=None, origin=True, csrf=True):
        data = json.dumps(body or {}).encode() if method == "POST" else None
        r = urllib.request.Request(PROCESSOR + path, data=data, method=method)
        if method == "POST":
            r.add_header("Content-Type", "application/json")
            if origin:
                r.add_header("Origin", PROCESSOR)
            if csrf and self.csrf:
                r.add_header("X-TP-CSRF", self.csrf)
        try:
            with self.op.open(r, timeout=30) as resp:
                return resp.status, json.loads(resp.read() or b"{}")
        except urllib.error.HTTPError as e:
            return e.code, json.loads(e.read() or b"{}")

    def sign_in(self, code):
        st, j = self.req("POST", "/api/session", {"code": code})
        self.csrf = j.get("csrf", "")
        return st

    def until(self, path, pred, t=40):
        end = time.time() + t
        while True:
            st, j = self.req("GET", path)
            try:
                if pred(j):
                    return j
            except (KeyError, TypeError, IndexError):
                pass
            if time.time() > end:
                return j
            time.sleep(0.4)


def gh_state(j):
    return (j.get("github") or {}).get("state")


def git(repo, *a):
    return subprocess.run(["git", "-C", str(repo), *a], capture_output=True, text=True).stdout.strip()


def main():
    build()
    start_identity()
    code = start_processor()
    pp = procs["processor"].pid

    # ---- secrets ----
    rec(not (W / "processor-profile/runtime/operator-link.txt").exists(), "S1 Processor -no-link-file: no operator-link.txt on disk while running")
    cmdline = Path(f"/proc/{pp}/cmdline").read_bytes().decode(errors="replace")
    rec(code not in cmdline, "S2 Processor process arguments carry no session code")
    rec((W / "identity-profile/runtime/open-link.txt").exists(),
        "S3 (known, Identity unchanged) Identity still writes runtime/open-link.txt — the launcher deletes it", "file present before launcher sweep")
    st, j = Browser().req("POST", "/api/launch-ticket", {"code": "0" * 48})
    rec(st == 401, "S4 launch ticket refused without the run code", str(st))
    st, j = Browser().req("POST", "/api/launch-ticket", {"code": code}, origin=False)
    rec(st == 403, "S5 launch ticket refused without same-origin header", str(st))
    st, j = Browser().req("POST", "/api/launch-ticket", {"code": code})
    ticket = j.get("ticket", "")
    rec(st == 200 and len(ticket) == 48 and ticket != code, "S6 launcher trades the run code for a ticket")
    b = Browser()
    rec(b.sign_in(ticket) == 200, "S7 a browser opens a session with the ticket")
    rec(Browser().sign_in(ticket) == 401, "S8 the same ticket is refused the second time (single use)")
    rec(Browser().req("GET", "/api/ws/state")[0] == 401, "S9 workstation API refuses a browser with no session")

    # ---- workstation locked before pairing ----
    st, j = b.req("GET", "/api/ws/tree?repo=tass&path=")
    rec(st == 403 and j["gate"]["reason"] == "IDENTITY_NOT_CONNECTED", "W0 workstation locked before Identity pairing", j.get("error", ""))

    # ---- pair (the code is read from Identity's console in the disposable work folder) ----
    b.req("POST", "/api/connect/choose", {"url": IDENTITY})
    b.req("POST", "/api/connect/request", {"url": IDENTITY})
    time.sleep(0.7)
    codes = re.findall(r"\b([A-HJ-NP-Z2-9]{4}-[A-HJ-NP-Z2-9]{4})\b", (W / "identity.console.txt").read_text(errors="replace"))
    b.req("POST", "/api/connect/code", {"code": codes[-1] if codes else ""})
    s = b.until("/api/state", lambda s: s["verified"], 20)
    rec(bool(s.get("verified")), "P1 Processor paired with Identity (signed status)")
    s = b.until("/api/state", lambda s: gh_state(s) == "SIGNED_OUT", 15)
    rec(gh_state(s) == "SIGNED_OUT", "G0 GitHub starts SIGNED_OUT (Identity keeps no token across restarts)", str(gh_state(s)))
    st, j = b.req("GET", "/api/ws/tree?repo=tass&path=")
    rec(st == 403 and j["gate"]["reason"] == "GITHUB_NOT_VERIFIED", "W1 workstation locked while GitHub is not VERIFIED", j["gate"].get("github_state", ""))

    # ---- GitHub states, each forced at the fixture, read back through Processor ----
    def sign_in_github(outcome, want):
        FIX["outcome"] = "pending"
        st, j = b.req("POST", "/api/github/connect")
        s = b.until("/api/state", lambda s: gh_state(s) == "WAITING", 15)
        waiting = gh_state(s) == "WAITING"
        g = b.req("GET", "/api/ws/state")[1]["gate"]
        FIX["outcome"] = outcome
        s = b.until("/api/state", lambda s: gh_state(s) == want, 30)
        return waiting, g, s

    waiting, g, s = sign_in_github("deny", "DENIED")
    rec(waiting and g["github_state"] == "WAITING" and not g["ok"], "G1 WAITING reaches Processor and keeps the workstation locked")
    rec(gh_state(s) == "DENIED", "G2 GitHub access_denied -> Identity DENIED -> Processor DENIED", str(gh_state(s)))
    waiting, g, s = sign_in_github("expire", "EXPIRED")
    rec(gh_state(s) == "EXPIRED", "G3 GitHub expired_token -> Identity EXPIRED -> Processor EXPIRED", str(gh_state(s)))
    rec(b.req("GET", "/api/ws/tree?repo=tass&path=")[0] == 403, "G4 workstation stays locked after DENIED/EXPIRED")
    waiting, g, s = sign_in_github("approve", "VERIFIED")
    acct = (s.get("github") or {}).get("account") or {}
    rec(gh_state(s) == "VERIFIED" and acct.get("login") == "fixture-owner",
        "G5 approved device code -> Identity VERIFIED with account -> Processor VERIFIED", "%s %s mode=%s" % (gh_state(s), acct.get("login"), (s.get("github") or {}).get("mode")))

    # ---- workstation unlocked ----
    st, ws = b.req("GET", "/api/ws/state")
    gate = ws.get("gate", {})
    rec(st == 200 and gate.get("ok") and gate.get("github_login") == "fixture-owner", "W2 workstation opens on Identity-signed VERIFIED", json.dumps(gate))
    heads = {r["name"]: r["head_sha"] for r in ws.get("repos", [])}
    rec(heads == {"processor": git(PROC_REPO, "rev-parse", "HEAD"), "identity": git(ID_SRC, "rev-parse", "HEAD"), "tass": git(TASS_SRC, "rev-parse", "HEAD")},
        "W3 repository info is the real Git HEAD of each checkout", " ".join("%s=%s" % (k, v[:7]) for k, v in heads.items()))
    st, t = b.req("GET", "/api/ws/tree?repo=tass&path=")
    names = {e["name"] for e in t.get("entries", [])}
    rec(st == 200 and {"feature", "tests", "README.md"} <= names and ".git" not in names, "W4 file tree lists the real TASS checkout (no .git)", ", ".join(sorted(names)))
    st, f = b.req("GET", "/api/ws/file?repo=tass&path=README.md")
    rec(st == 200 and f.get("text") == (TASS_SRC / "README.md").read_text(), "W5 file view returns the exact bytes on disk", "%s bytes" % f.get("size"))
    bad = [p for p in ("../identity/go.mod", "/etc/passwd", ".git/config", "feature/../../x") if b.req("GET", "/api/ws/file?repo=tass&path=" + urllib.parse.quote(p))[0] != 403]
    rec(not bad, "W6 traversal, absolute and .git paths are refused", ",".join(bad))
    rec(b.req("POST", "/api/ws/test", {"id": "tass-unit"}, csrf=False)[0] == 403, "W7 test run refused without anti-forgery token")
    rec(b.req("POST", "/api/ws/test", {"id": "rm -rf /"})[0] == 404, "W8 only allowlisted test ids run")
    for tid in ("identity-go-github", "tass-unit"):
        st, j = b.req("POST", "/api/ws/test", {"id": tid})
        ws = b.until("/api/ws/state", lambda w: w["runs"] and w["runs"][0]["test"] == tid and w["runs"][0]["state"] != "RUNNING", 240)
        run = ws["runs"][0]
        last = [ln for ln in run["output"].strip().splitlines() if ln.strip()][-1:] or [""]
        rec(st == 202 and run["state"] == "PASS" and run["exit_code"] == 0 and run["requested_by"] == "fixture-owner",
            "W9 allowlisted test %s really ran: %s" % (tid, run["state"]), "exit %s @ %s, last line: %s" % (run["exit_code"], run["head_sha"][:7], last[0][:80]))

    # ---- fixture GitHub is not accepted unless explicitly allowed ----
    stop("processor")
    code2 = start_processor(allow_fixture=False)
    b2 = Browser()
    b2.sign_in(code2)
    j = b2.until("/api/ws/state", lambda w: w["gate"].get("github_state") == "VERIFIED", 20)
    rec(j["gate"]["reason"] == "GITHUB_TEST_FIXTURE" and not j["gate"]["ok"], "W10 a TEST FIXTURE GitHub never unlocks a normal workstation", j["gate"].get("reason", ""))
    rec(not (W / "processor-profile/runtime/operator-link.txt").exists(), "S10 still no Processor link file after restart")

    # ---- Identity dies: fail closed ----
    stop("processor")
    start_processor(allow_fixture=True)
    b3 = Browser()
    b3.sign_in(re.search(r"#code=([0-9a-f]+)", (W / "processor.console.txt").read_text()).group(1))
    b3.until("/api/ws/state", lambda w: w["gate"]["ok"], 20)
    stop("identity", signal.SIGKILL)
    st, j = b3.req("GET", "/api/ws/file?repo=tass&path=README.md")
    rec(st == 403 and j["gate"]["reason"] == "IDENTITY_NOT_CONNECTED", "W11 Identity killed -> workstation refuses immediately (fail closed)", str(st))
    rec((W / "identity-profile/runtime/open-link.txt").exists(),
        "S11 (known) a killed Identity leaves open-link.txt behind — why the launcher sweeps on start/stop")


try:
    main()
finally:
    for n in ("processor", "identity"):
        stop(n)
print("\n%d/%d PASS" % (sum(results), len(results)))
sys.exit(0 if results and all(results) else 1)
