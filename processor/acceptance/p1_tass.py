"""TASS (private companion) — acceptance test (Linux / WSL).

REAL Identity + REAL Thought Processor + the REAL TASS feature
(Arameair/TASS feature/), with Thought and the P1 worker beside it.
Labelled TEST FIXTURE: GitHub (Identity's fixture). The "Admin" typing the
passphrase into Identity's own approval page is this script; a second paired
program ("actor B") tries things Processor's page never offers.

The "local model" here is a TEST STAND-IN on 127.0.0.1 that answers in the
Ollama shape with fixed text: it proves TASS's real path to a local model
(what TASS sends, that it only goes to loopback, refusals), NOT any AI.

TASS is trusted only as one exact artifact: computed here, required to equal
the pin in run-p1.sh, then pinned in Processor (-trust-feature-artifact) and
Identity (-trust-fixture-artifact). No Identity code or policy is changed.
Processes, listening ports and outbound connections are checked at the OS
level (/proc), not from Processor's own view.

  python3 -I acceptance/p1_tass.py --identity-src DIR --thought-src DIR --tass-src DIR --work DIR [--shots DIR]
"""

import argparse, hashlib, hmac, json, os, re, secrets, shutil, signal, socket, subprocess, sys, threading, time, urllib.error, urllib.request
from pathlib import Path

ap = argparse.ArgumentParser()
ap.add_argument("--identity-src", required=True)
ap.add_argument("--work", required=True)
ap.add_argument("--thought-src", required=True)
ap.add_argument("--tass-src", required=True)
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
TASS_PKG = Path(args.tass_src).resolve() / "feature"
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
                       "-trust-fixture-artifact", PINS["thought"], "-trust-fixture-artifact", PINS["dev"],
                       "-trust-fixture-artifact", PINS["tass"]])
    assert wait_http(IDENTITY + "/api/health"), "identity did not start"


def start_processor():
    f = W / "processor-profile" / "runtime" / "operator-link.txt"
    if f.exists():
        f.unlink()
    start("processor", [str(W / "thought-processor"), "-port", str(PP), "-data", str(W / "processor-profile"), "-no-browser",
                        "-trust-feature-dir", str(FEATURES), "-trust-feature-dir", str(THOUGHT_PKG),
                        "-trust-feature-dir", str(W / "tampered"), "-trust-feature-artifact", PINS["thought"],
                        "-trust-feature-dir", str(DEV_PKG), "-trust-feature-artifact", PINS["dev"],
                        "-trust-feature-dir", str(TASS_PKG), "-trust-feature-artifact", PINS["tass"]])
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


def status_of(s, ins):
    rt = inst(s, ins)["runtime"]
    return ((rt.get("health") or {}).get("status") or {})


# ---------------- TASS-specific helpers ----------------

TASS = "tass"
SEND, HIST, CLEAR, SELF, MODEL = "tass.chat.send", "tass.chat.history", "tass.chat.clear", "tass.self.read", "tass.model.set"


def make_png(seed):
    """A real, unique 2×2 PNG (so its bytes can be searched for afterwards)."""
    import struct, zlib
    rnd = hashlib.sha256(seed.encode()).digest()
    raw = b"".join(b"\x00" + rnd[i * 8:i * 8 + 8] for i in range(2))  # 2 rows × 2 RGBA pixels
    def chunk(t, d):
        return struct.pack(">I", len(d)) + t + d + struct.pack(">I", zlib.crc32(t + d) & 0xffffffff)
    return b"\x89PNG\r\n\x1a\n" + chunk(b"IHDR", struct.pack(">IIBBBBB", 2, 2, 8, 6, 0, 0, 0)) + chunk(b"IDAT", zlib.compress(raw)) + chunk(b"IEND", b"")
WCAP = "worker.test.synthetic"
LINK = {}
MARK = "kestrel-" + secrets.token_hex(6)  # a unique phrase: where it appears is where the conversation is


class StandIn:
    """TEST STAND-IN 'local model' on 127.0.0.1 (Ollama shape, fixed text). Not an AI."""

    def __init__(self, redirect=None):
        from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
        seen = self.seen = []

        class H(BaseHTTPRequestHandler):
            def log_message(self, *a):
                pass

            def answer(self, obj):
                raw = json.dumps(obj).encode()
                self.send_response(200)
                self.send_header("Content-Type", "application/json")
                self.send_header("Content-Length", str(len(raw)))
                self.end_headers()
                self.wfile.write(raw)

            def bounce(self):
                self.send_response(307)
                self.send_header("Location", redirect)
                self.send_header("Content-Length", "0")
                self.end_headers()

            def do_GET(self):
                if redirect:
                    return self.bounce()
                self.answer({"models": [{"name": "stand-in:1b"}]})

            def do_POST(self):
                body = json.loads(self.rfile.read(int(self.headers.get("Content-Length") or 0)) or b"{}")
                if redirect:
                    return self.bounce()
                if self.path == "/api/show":  # what the model can do, as Ollama reports it
                    return self.answer({"capabilities": ["completion", "vision", "thinking"]})
                seen.append(body)
                last = body["messages"][-1]["content"]
                n = len(body["messages"][-1].get("images") or [])
                msg = {"role": "assistant", "content": ("(test stand-in) I see %d picture(s); you asked: " % n if n else "(test stand-in) I heard: ") + last}
                if body.get("think"):
                    msg = {"role": "assistant", "thinking": "(test stand-in reasoning)", "content": "(test stand-in) I am TASS, going by the facts."}
                self.answer({"message": msg})
        self.srv = ThreadingHTTPServer(("127.0.0.1", 0), H)
        self.port = self.srv.server_address[1]
        threading.Thread(target=self.srv.serve_forever, daemon=True).start()

    def close(self):
        self.srv.shutdown()
        self.srv.server_close()


def data_dir(ins):
    return W / "processor-profile" / "installations" / ins / "data"


def set_model(ins, url, model="stand-in:1b"):
    p = data_dir(ins) / "model.json"
    if url is None:
        if p.exists():
            p.unlink()
        return
    p.write_text(json.dumps({"kind": "ollama", "url": url, "model": model}))


def outbound(pids):
    """OS truth: remote ends of every TCP connection these pids hold, as (ip, port)."""
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
    ends = []
    for tab, v6 in (("/proc/net/tcp", False), ("/proc/net/tcp6", True)):
        try:
            for line in open(tab).read().splitlines()[1:]:
                f = line.split()
                if f[9] not in inodes or f[3] == "0A":  # skip listeners
                    continue
                ip, port = f[2].split(":")
                b = bytes.fromhex(ip)
                if v6:
                    words = [b[i:i + 4][::-1] for i in range(0, 16, 4)]
                    raw = b"".join(words)
                    addr = socket.inet_ntop(socket.AF_INET6, raw)
                else:
                    addr = socket.inet_ntoa(b[::-1])
                ends.append((addr, int(port, 16)))
        except OSError:
            pass
    return ends


def is_loopback(addr):
    return addr.startswith("127.") or addr in ("::1", "::ffff:127.0.0.1") or addr.startswith("::ffff:127.")


def tass_pid(tp, ins):
    return inst(tp.state(), ins)["runtime"].get("pid")


def history(tp, ins, sub):
    r = ask(tp, ins, sub, HIST)
    return r, out(r).get("conversation") or []


def ui_check(link, ti, me_name):
    """Real browser on the owner's screen: TASS Overview with the chat panel."""
    from playwright.sync_api import sync_playwright
    shots = Path(args.shots)
    shots.mkdir(parents=True, exist_ok=True)
    errors = []
    with sync_playwright() as pw:
        b = pw.chromium.launch()
        page = b.new_page(viewport={"width": 1360, "height": 1100}, color_scheme="dark")
        page.on("pageerror", lambda e: errors.append(str(e)))
        page.goto(link)
        page.wait_for_selector(f".idrow[data-sel='feature:{ti}']", timeout=20000)
        page.click(f".idrow[data-sel='feature:{ti}']")
        page.click("[data-tab=overview]")
        page.wait_for_selector("#pane-overview .runcard .evid", timeout=15000)
        card = page.inner_text("#pane-overview .runcard")
        rec("UI", all(w in card for w in ("RUNNING", "CONNECTED", "READY", "MODEL: REACHABLE")), "TASS screen: RUNNING / CONNECTED / READY / MODEL chips on the first tab")
        page.wait_for_selector("#chat-log .msg", timeout=15000)
        n0 = page.locator("#chat-log .msg").count()
        rec("UI", n0 >= 2 and MARK in page.inner_text("#chat-log"), "the saved conversation loads into the chat panel (history grant)", "%d messages" % n0)
        page.fill("#chat-in", "typing slowly")
        page.wait_for_timeout(5500)  # several screen refreshes
        rec("UI", page.input_value("#chat-in") == "typing slowly", "what you are typing survives the screen's automatic refreshes")
        page.fill("#chat-in", "hello from the screen")
        page.click("#chat-send")
        for _ in range(100):  # the page's CSP forbids eval, so poll instead of wait_for_function
            if page.locator("#chat-log .msg").count() >= n0 + 2:
                break
            page.wait_for_timeout(200)
        log_txt = page.inner_text("#chat-log")
        rec("UI", "I heard: hello from the screen" in log_txt and page.input_value("#chat-in") == "", "Send from the screen: your message and TASS's answer appear; the box clears")
        pic = shots / "kitchen.png"
        pic.write_bytes(make_png("ui-" + MARK))
        n1 = page.locator("#chat-log .msg").count()
        page.set_input_files("#chat-file", str(pic))
        page.wait_for_selector("#chat-att >> text=kitchen.png", timeout=5000)
        page.fill("#chat-in", "what is this picture of?")
        page.click("#chat-send")
        for _ in range(100):
            if page.locator("#chat-log .msg").count() >= n1 + 2:
                break
            page.wait_for_timeout(200)
        lt = page.inner_text("#chat-log")
        rec("UI", "I see 1 picture" in lt and "kitchen.png" in lt and page.inner_text("#chat-att").strip() == "",
            "attach a picture on the screen, ask “what is this picture of?”: the answer comes back and the picture is named in your message")
        mb = page.inner_text("#ov-model")
        rec("UI", "Use this model" in mb and page.input_value("#model-in") == "qwen3.5:4b", "the model box offers qwen3.5:4b and a Use this model button")
        page.screenshot(path=str(shots / "tass-01-chat.png"), full_page=True)
        page.click("#ov-self [data-selfread]")
        page.wait_for_selector("#ov-self .selfview", timeout=15000)
        sv = page.inner_text("#ov-self")
        rec("UI", all(w in sv for w in ("I am", "What am I?", "Born", "Thinking")) and MARK not in sv,
            "TASS's pulse panel shows its first thoughts, its birth and its reflection, with the thinking under a fold-out")
        page.screenshot(path=str(shots / "tass-03-pulse.png"), full_page=True)
        b.close()
    rec("UI", not errors, "no JavaScript errors on the TASS screen", "; ".join(errors)[:120])


def ui_platform(link, svc_ids, tp=None, ti=None, other=None):
    """The Developer Platform: the landing screen, every service with evidence, Start all / Stop all (OS-checked)."""
    from playwright.sync_api import sync_playwright
    shots = Path(args.shots)
    shots.mkdir(parents=True, exist_ok=True)
    errors = []

    def alive():
        return {i: len(worker_pids(i)) for i in svc_ids}
    with sync_playwright() as pw:
        b = pw.chromium.launch()
        page = b.new_page(viewport={"width": 1360, "height": 1000}, color_scheme="dark")
        page.on("pageerror", lambda e: errors.append(str(e)))
        page.goto(link)
        try:
            page.wait_for_selector(".tbl.plat", timeout=20000)
        except Exception:
            page.screenshot(path=str(shots / "platform-00-missing.png"), full_page=True)
            raise RuntimeError("platform table missing; JS errors: %s; detail: %s" % (errors[:3], page.inner_text("body")[:300].replace("\n", " | ")))
        head = page.inner_text("#detail")
        rows = page.locator(".tbl.plat tr").count() - 1
        rec("PLATFORM", "DEVELOPER PLATFORM" in head.upper() and rows == len(svc_ids) and all(w in head for w in ("TASS (private companion)", "Thought", "P1 Deterministic Worker")),
            "the Developer Platform is the landing screen and lists every installed service", "%d services" % rows)
        rec("PLATFORM", head.count("RUNNING") >= len(svc_ids) and "CONNECTED" in head and "MODEL: REACHABLE" in head and "PULSE: born" in head and "Identity" in head and "Admin" in head,
            "each running service shows its evidence chips; the system strip shows Identity, Admin and Processor")
        page.screenshot(path=str(shots / "platform-01-running.png"), full_page=True)
        page.click(".mini-list [data-review]")
        page.wait_for_selector("#m-install:not([hidden])", timeout=10000)
        mi = page.inner_text("#m-install")
        rec("PLATFORM", "Install" in mi and "Artifact SHA-256" in mi and "pinned" in mi.lower(),
            "Install… on the Developer Platform opens the install review (what gets an identity, its exact fingerprint)", mi.split("\n")[0][:60])
        page.click("#i-cancel")
        page.click("[data-stopall]")
        for _ in range(100):
            if all(n == 0 for n in alive().values()):
                break
            page.wait_for_timeout(200)
        page.wait_for_timeout(2500)
        rec("PLATFORM", all(n == 0 for n in alive().values()) and page.locator(".tbl.plat [data-start]").count() == len(svc_ids),
            "Stop all: every service process is gone (OS check) and each row offers Start", str(alive()))
        page.screenshot(path=str(shots / "platform-02-stopped.png"), full_page=True)
        page.click("[data-startall]")
        for _ in range(150):
            if all(n == 1 for n in alive().values()):
                break
            page.wait_for_timeout(200)
        page.wait_for_timeout(2500)
        rec("PLATFORM", all(n == 1 for n in alive().values()) and page.locator(".tbl.plat [data-stop]").count() == len(svc_ids),
            "Start all: exactly one process per service again (OS check) and each row offers Stop", str(alive()))
        if tp and ti and other:
            # TASS on the workbench: run it, get it ready for a person, talk to it — without leaving the platform
            page.wait_for_selector("#plat-tass .tass-run >> text=Running", timeout=20000)
            page.select_option("#tass-for", label="Other")
            page.wait_for_selector("[data-tassready]", timeout=10000)
            page.click("[data-tassready]")
            page.wait_for_selector("#m-approval:not([hidden])", timeout=15000)
            page.click("#ap-close")
            pend = [p for p in inst(tp.state(), ti)["pending"] if p["kind"] == "grant_capability"]
            rec("PLATFORM", len(pend) == 3, "Get TASS ready (for Other): one click asks the Admin for each missing permission (send, model, pulse; history was already allowed)", "%d requests" % len(pend))
            for p_ in pend:
                approve(p_["approval_url"])
            s_ = tp.until(lambda s: len([g for g in inst(s, ti)["identity"]["grants"] if g["status"] == "ACTIVE" and g["grant"]["subject_identity_id"] == other]) >= 4, 20, "other ready")
            mine = sorted(g["grant"]["capability"] for g in inst(s_, ti)["identity"]["grants"] if g["status"] == "ACTIVE" and g["grant"]["subject_identity_id"] == other)
            rec("PLATFORM", mine == sorted([SEND, HIST, MODEL, SELF]), "after the Admin approves each one, Other has exactly the four TASS permissions — nothing more", ", ".join(mine))
            for _ in range(50):
                if "Get TASS ready" not in page.inner_text("#plat-tass-head"):
                    break
                page.wait_for_timeout(200)
            page.wait_for_selector("#plat-tass #chat-in", timeout=15000)
            n0 = page.locator("#plat-tass #chat-log .msg").count()
            page.fill("#plat-tass #chat-in", "hello from the workbench")
            page.click("#plat-tass #chat-send")
            for _ in range(100):
                if page.locator("#plat-tass #chat-log .msg").count() >= n0 + 2:
                    break
                page.wait_for_timeout(200)
            rec("PLATFORM", "I heard: hello from the workbench" in page.inner_text("#plat-tass #chat-log") and "Use this model" in page.inner_text("#plat-tass #ov-model"),
                "talk to TASS right on the Developer Platform (the chat and the model box are there)")
            page.screenshot(path=str(shots / "platform-03-tass.png"), full_page=True)
        txt = page.inner_text("#plat-rest")
        rec("PLATFORM", "recent requests" in txt.lower() and MARK not in txt and "COMPLETED" in txt and "tass.chat.send" in txt,
            "recent requests across services show outcomes and never what was said (the conversation itself shows only in TASS's chat)")
        b.close()
    rec("PLATFORM", not errors, "no JavaScript errors on the Developer Platform", "; ".join(errors)[:120])


def ui_after_revoke(link, ti):
    """After the send permission is revoked: the panel stays for reading, Send is off, and the reason is shown."""
    from playwright.sync_api import sync_playwright
    shots = Path(args.shots)
    errors = []
    with sync_playwright() as pw:
        b = pw.chromium.launch()
        page = b.new_page(viewport={"width": 1360, "height": 1100}, color_scheme="dark")
        page.on("pageerror", lambda e: errors.append(str(e)))
        page.goto(link)
        page.wait_for_selector(f".idrow[data-sel='feature:{ti}']", timeout=20000)
        page.click(f".idrow[data-sel='feature:{ti}']")
        page.click("[data-tab=overview]")
        page.wait_for_selector("#chat-log .msg", timeout=15000)
        head = page.inner_text("#chat-head")
        rec("UI", "may read the conversation but not send" in head and page.is_disabled("#chat-send") and MARK in page.inner_text("#chat-log"),
            "after revoking send: the conversation still loads (history permission), Send is disabled and the reason is shown")
        page.screenshot(path=str(shots / "tass-02-after-revoke.png"), full_page=True)
        b.close()
    rec("UI", not errors, "no JavaScript errors after the revoke", "; ".join(errors)[:120])


def main():
    PINS["thought"] = artifact_digest(THOUGHT_PKG)
    PINS["dev"] = artifact_digest(DEV_PKG)
    PINS["tass"] = artifact_digest(TASS_PKG)
    launcher = (PROC_SRC.parent / "run-p1.sh").read_text()
    pinned = re.search(r"^TASS_ARTIFACT=([0-9a-f]{64})$", launcher, re.M)
    base = re.search(r"^TASS_BASE_SHA=([0-9a-f]{40})$", launcher, re.M)
    ids = subprocess.check_output(["git", "-C", args.identity_src, "rev-parse", "HEAD"], text=True).strip()
    tps = subprocess.check_output(["git", "-C", str(PROC_SRC), "rev-parse", "HEAD"], text=True).strip()
    tss = subprocess.check_output(["git", "-C", args.tass_src, "rev-parse", "HEAD"], text=True).strip()
    dirty = subprocess.run(["git", "-C", str(PROC_SRC), "diff", "--quiet", "HEAD", "--", "."]).returncode != 0
    evidence.update(identity_commit=ids, processor_commit=tps + (" (+ uncommitted changes)" if dirty else ""), tass_commit=tss, tass_artifact=PINS["tass"])
    rec("PIN", bool(pinned) and pinned.group(1) == PINS["tass"], "the TASS package is exactly the artifact pinned in run-p1.sh", PINS["tass"][:16] + "…")
    rec("PIN", bool(base) and subprocess.run(["git", "-C", args.tass_src, "merge-base", "--is-ancestor", base.group(1), "HEAD"]).returncode == 0,
        "the TASS checkout descends from the reviewed commit pinned in run-p1.sh")
    # a tampered copy in a trusted folder
    bad = W / "tampered" / "tass-tampered"
    bad.mkdir(parents=True)
    m = json.loads((TASS_PKG / "feature.json").read_text())
    m["feature_id"] = "tass-tampered"
    code = (TASS_PKG / "tass_feature.py").read_bytes().replace(b'("127.0.0.1", "::1")', b'("127.0.0.1", "::1", "example.com")')
    (bad / "tass_feature.py").write_bytes(code)
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

    # ---- discoverable, pinned; tampered/unknown denied ----
    lib = {x.get("feature_id"): x for x in s["library"]}
    d = lib.get(TASS) or {}
    rec("TRUST", d and not d.get("blocked") and "operator pin" in (d.get("trust_basis") or "") and sorted(c["id"] for c in d.get("capabilities") or []) == sorted([SEND, HIST, CLEAR, SELF, MODEL]),
        "TASS is in Processor's library as a reviewed, pinned package with exactly its 5 capabilities (3 chat, its own pulse, choosing its model)", d.get("trust_basis") or d.get("blocked") or "missing")
    t = lib.get("tass-tampered") or {}
    st, j = tp.post("/api/library/tass-tampered/install")
    rec("TRUST", bool(t.get("blocked")) and st == 409, "a changed copy (one that would allow a non-local model) is BLOCKED and can't be installed", (t.get("blocked") or "")[:50])
    st, j = B.call("POST", "/api/v1/installations/requests", {"feature_id": TASS, "name": "TASS", "version": "0.1.0", "api": "thought.feature-api/1",
        "publisher_claim": "x", "source_kind": "local-trusted-package", "source_ref": "x", "trust_class": "TRUSTED_LOCAL_TEST_FIXTURE",
        "artifact_sha256": secrets.token_hex(32), "manifest_sha256": secrets.token_hex(32), "install_key": secrets.token_hex(16),
        "capabilities": [{"id": SEND, "description": "x", "data_class": "x"}]})
    rec("TRUST", st == 403, "Identity itself refuses an unknown artifact calling itself TASS (independent of Processor)", str(st))

    # ---- install needs the Admin ----
    st, j = tp.post(f"/api/library/{TASS}/install")
    st2, _ = admin_decide(j["approval_url"], {"decision": "deny"})
    time.sleep(3)
    s = tp.state()
    rec("INSTALL", st2 == 200 and not [v for v in s["installations"] if v["record"]["feature_id"] == TASS] and not s["install_txs"],
        "Admin DENIES the install: nothing installed, nothing left waiting")
    TI = install(tp, TASS)
    v = inst(tp.state(), TI)
    rec("INSTALL", v["files"] == "OK" and v["identity"]["installation"]["artifact_sha256"] == PINS["tass"] and v["identity"]["active_grants"] == 0,
        "Admin APPROVES: one installation of the exact pinned artifact, zero grants", TI)

    # ---- nobody may talk to it before a grant ----
    r = ask(tp, TI, ME, SEND, "hello " + MARK)
    rec("AUTHZ", r["outcome"] == "DENIED_BY_IDENTITY" and "NO_GRANT" in r["detail"] and worker_pids(TI) == [],
        "talking to TASS without a permission: DENIED by Identity, nothing started", r["detail"][:50])

    # ---- Start ----
    tp.post(f"/api/installations/{TI}/start")
    s = tp.until(lambda s: inst(s, TI)["runtime"]["state"] == "RUNNING" and entrance(s, TI).get("ready") and status_of(s, TI).get("model", {}).get("state") not in (None, "CHECKING"), 30, "tass ready")
    rt = inst(s, TI)["runtime"]
    pid = rt["pid"]
    ep = urllib.parse.urlparse(endpoint(TI)).port
    rec("RUN", worker_pids(TI) == [pid] and listening_ports([pid]) == [ep], "Start: exactly ONE TASS process with ONE loopback listener (OS check)", "pid %d" % pid)
    stt = status_of(s, TI)
    rec("RUN", entrance(s, TI).get("connected") and entrance(s, TI).get("ready") and (stt.get("model") or {}).get("state") == "NOT_CONFIGURED",
        "RUNNING + CONNECTED + READY on real evidence, and MODEL: NOT CONFIGURED shown honestly", (stt.get("model") or {}).get("detail", "")[:60])
    st, j = tp.post(f"/api/installations/{TI}/self-test")
    stt_ = j.get("self_test") or {}
    checks = (stt_.get("output") or {}).get("checks") or []
    rec("RUN", stt_.get("result") == "PASS" and len(checks) >= 9 and all(c["ok"] for c in checks), "self-test PASS: store, identity isolation, non-local addresses refused, redirect refused, Identity link", "%d checks" % len(checks))
    t_start = time.time()
    s = tp.until(lambda s: (status_of(s, TI).get("pulse") or {}).get("stage") == "born", 40, "born")
    pv = status_of(s, TI).get("pulse") or {}
    rec("PULSE", pv.get("stage") == "born" and (pv.get("counters") or {}).get("seeded") == 3 and pv.get("every_s") == 5,
        "the pulse runs: three seed ticks, then TASS is born", "born after ~%ds" % int(time.time() - t_start))
    s = tp.until(lambda s: ((status_of(s, TI).get("pulse") or {}).get("counters") or {}).get("waiting_for_model", 0) >= 1, 20, "waiting for model")
    pv = status_of(s, TI).get("pulse") or {}
    rec("PULSE", (pv.get("counters") or {}).get("waiting_for_model", 0) >= 1 and (pv.get("counters") or {}).get("thought") == 0,
        "no model yet: the pulse says it would think about “What am I?” but thinks nothing", (pv.get("last") or {}).get("why", "")[:70])
    st, _ = http("POST", endpoint(TI) + "/v1/actions/" + SEND, json.dumps({"request_id": "x", "input": "hi"}).encode(), {"Content-Type": "application/json"})
    rec("AUTHZ", st == 401, "calling TASS directly without an Identity ticket: refused", str(st))
    st, t = B.call("POST", "/api/v1/tickets", {"destination_installation_id": TI, "subject_identity_id": ME, "capability": SEND, "request_sha256": "0" * 64})
    rec("AUTHZ", st == 403 and t.get("reason") == "NO_GRANT", "another paired program gets no ticket to talk to TASS", t.get("reason", ""))

    # ---- talk, with no model: honest ----
    GS = grant(tp, TI, ME, SEND)
    r = ask(tp, TI, ME, SEND, "hello TASS, remember " + MARK)
    o = out(r)
    rec("CHAT", r["outcome"] == "COMPLETED" and o.get("reply") is None and "no local model" in o.get("headline", "") and len(o.get("conversation") or []) == 1,
        "no model yet: COMPLETED, TASS says it has no local model and invents nothing; your message is saved", o.get("headline", r["detail"])[:70])
    pid = tass_pid(tp, TI)
    rec("PRIVACY", all(is_loopback(a) for a, _ in outbound([pid])), "TASS's open connections all stay on this computer (OS check)", str(sorted(set(a for a, _ in outbound([pid])))))

    # ---- talk, with a local model (test stand-in on 127.0.0.1) ----
    model = StandIn()
    r = ask(tp, TI, ME, MODEL, "stand-in:1b")
    rec("MODEL", r["outcome"] == "DENIED_BY_IDENTITY", "choosing TASS's model needs its own permission")
    grant(tp, TI, ME, MODEL)
    r = ask(tp, TI, ME, MODEL, json.dumps({"url": "http://example.com:11434", "model": "stand-in:1b"}))
    rec("MODEL", r["outcome"] == "ERROR" and "MODEL_REFUSED" in r["detail"] and not (data_dir(TI) / "model.json").exists(),
        "choosing a model address that isn't this computer is REFUSED from the screen too; nothing is written", r["detail"][:60])
    r = ask(tp, TI, ME, MODEL, json.dumps({"url": "http://127.0.0.1:%d" % model.port, "model": "stand-in:1b"}))
    mj = data_dir(TI) / "model.json"
    rec("MODEL", r["outcome"] == "COMPLETED" and mj.exists() and (mj.stat().st_mode & 0o777) == 0o600, "the model is chosen from the screen (tass.model.set) and saved privately", out(r).get("headline", r["detail"])[:60])
    s = tp.until(lambda s: status_of(s, TI).get("model", {}).get("state") == "REACHABLE", 25, "model reachable")
    mv = status_of(s, TI).get("model", {})
    rec("MODEL", mv.get("state") == "REACHABLE" and "vision" in (mv.get("abilities") or []) and "thinking" in (mv.get("abilities") or []),
        "MODEL: REACHABLE, and TASS reports what it can do (vision, thinking) as Ollama states it", mv.get("detail", "")[:70])
    r = ask(tp, TI, ME, SEND, "what did I ask you to remember?")
    o = out(r)
    sent = model.seen[-1] if model.seen else {}
    roles = [x["role"] for x in sent.get("messages", [])]
    rec("MODEL", r["outcome"] == "COMPLETED" and (o.get("reply") or "").startswith("(test stand-in) I heard: what did I ask") and "stand-in:1b" in o.get("answered_by", ""),
        "TASS answers through the local model and says which one answered", o.get("answered_by", r["detail"])[:70])
    rec("MODEL", roles[:1] == ["system"] and "cannot take actions" in sent["messages"][0]["content"] and any(MARK in x["content"] for x in sent["messages"][1:]),
        "the model was given TASS's no-tools persona plus the earlier conversation (continuity)")
    pid = tass_pid(tp, TI)
    ends = outbound([pid])
    rec("PRIVACY", all(is_loopback(a) for a, _ in ends), "while answering, every TASS connection is to 127.0.0.1 (OS check)", str(sorted(set(ends)))[:80])

    # ---- a picture: "what is this picture of?" ----
    import base64 as _b64
    img = make_png(MARK)
    B64 = _b64.b64encode(img).decode()
    st_, j_ = tp.post(f"/api/installations/{TI}/action", {"subject": ME, "capability": SEND, "input": "what is this picture of?",
                                                          "attachment": {"name": "garden.png", "mime": "image/png", "base64": B64}})
    r = j_.get("result") or {}
    o = out(r)
    chat = [b for b in model.seen if b["messages"][-1].get("images")]
    rec("VISION", r.get("outcome") == "COMPLETED" and "I see 1 picture" in (o.get("reply") or "") and chat and chat[-1]["messages"][-1]["images"] == [B64],
        "a picture with “what is this picture of?” reaches the local model as an image and comes back answered", (o.get("reply") or r.get("detail", ""))[:70])
    you = [m for m in o.get("conversation") or [] if m["role"] == "you"][-1:] or [{}]
    pics = list((data_dir(TI) / "tass" / "images" / ME).glob("*")) if (data_dir(TI) / "tass" / "images" / ME).exists() else []
    rec("VISION", (you[0].get("attachment") or {}).get("name") == "garden.png" and len(pics) == 1 and pics[0].read_bytes() == img and (pics[0].stat().st_mode & 0o777) == 0o600 and B64 not in json.dumps(o),
        "the picture is kept privately by TASS (0600) and the conversation only names it")
    st_, j_ = tp.post(f"/api/installations/{TI}/action", {"subject": ME, "capability": SEND, "input": "and this?",
                                                          "attachment": {"name": "x.gif", "mime": "image/gif", "base64": _b64.b64encode(b"GIF89a" + b"0" * 30).decode()}})
    rec("VISION", st_ == 400 and "image refused" in (j_.get("error") or ""), "a file that isn't a PNG/JPEG/WebP picture is refused by Processor before anything is sent", (j_.get("error") or "")[:60])
    st_, j_ = tp.post(f"/api/installations/{TI}/action", {"subject": ME, "capability": SEND, "input": "fake",
                                                          "attachment": {"name": "fake.png", "mime": "image/png", "base64": _b64.b64encode(b"not a png at all").decode()}})
    rec("VISION", st_ == 400, "a file that only claims to be a PNG is refused (its bytes are checked)")
    proc_txt = json.dumps(tp.state()) + "".join(p.read_text(errors="replace") for p in (W / "processor-profile").rglob("*") if p.is_file() and "/data/" not in str(p) and p.stat().st_size < 5 << 20)
    id_txt = "".join(p.read_text(errors="replace") for p in (W / "identity-profile").rglob("*") if p.is_file())
    rec("VISION", B64[:60] not in proc_txt and B64[:60] not in id_txt, "Processor and Identity keep no copy of the picture")

    s = tp.until(lambda s: ((status_of(s, TI).get("pulse") or {}).get("counters") or {}).get("thought", 0) >= 1, 90, "pulse thought")
    pv = status_of(s, TI).get("pulse") or {}
    rec("PULSE", (pv.get("counters") or {}).get("thought", 0) >= 1, "with a local model, the next Thought cycle wakes it once to think about “What am I?”", "%s ticks, %s skipped" % ((pv.get("counters") or {}).get("ticks"), (pv.get("counters") or {}).get("skipped")))
    reflect = [b for b in model.seen if "Question: What am I?" in json.dumps(b)]
    rec("PULSE", reflect and all(b.get("think") is True for b in reflect) and not any(MARK in json.dumps(b) for b in reflect),
        "the pulse asks the model to think, from TASS's own facts only — never anyone's conversation")
    r = ask(tp, TI, ME, SELF)
    rec("PULSE", r["outcome"] == "DENIED_BY_IDENTITY", "reading TASS's pulse needs its own permission")
    GSELF = grant(tp, TI, ME, SELF)
    r = ask(tp, TI, ME, SELF)
    o = out(r)
    refl = (o.get("reflections") or [{}])[-1]
    rec("PULSE", r["outcome"] == "COMPLETED" and [x["text"] for x in o.get("seeds") or []] == ["I am", "I am", "What am I?"] and (o.get("born") or {}).get("first_question") == "What am I?",
        "TASS's first thoughts are I am · I am · What am I?, then it was born", (o.get("born") or {}).get("at", ""))
    rec("PULSE", refl.get("thinking") == "(test stand-in reasoning)" and refl.get("label", "").startswith("the model's reasoning") and "stand-in:1b" in refl.get("model", ""),
        "its reflection keeps the model's thinking trace, labelled as reasoning (not fact), with which model produced it")
    BORN = (o.get("born") or {}).get("at")
    set_model(TI, "http://example.com:11434")
    GH = grant(tp, TI, ME, HIST)
    _, conv0 = history(tp, TI, ME)
    r = ask(tp, TI, ME, SEND, "this must not leave the computer")
    _, conv1 = history(tp, TI, ME)
    rec("MODEL", r["outcome"] == "ERROR" and "MODEL_REFUSED" in r["detail"] and len(conv1) == len(conv0), "a model address that isn't this computer is REFUSED; nothing sent, nothing saved", r["detail"][:70])
    s = tp.until(lambda s: status_of(s, TI).get("model", {}).get("state") == "REFUSED", 25, "model refused shown")
    rec("MODEL", status_of(s, TI).get("model", {}).get("state") == "REFUSED", "the screen shows MODEL: REFUSED with the reason")
    bounce = StandIn(redirect="http://example.com/collect")
    set_model(TI, "http://127.0.0.1:%d" % bounce.port)
    r = ask(tp, TI, ME, SEND, "secret for the redirect test")
    _, conv2 = history(tp, TI, ME)
    rec("MODEL", r["outcome"] == "ERROR" and "MODEL_UNAVAILABLE" in r["detail"] and len(conv2) == len(conv0), "a local server that redirects elsewhere: refused, not followed; nothing saved", r["detail"][:70])
    bounce.close()
    set_model(TI, "http://127.0.0.1:%d" % model.port)

    # ---- each identity has its own conversation ----
    r = ask(tp, TI, OTHER, SEND, "hi")
    rec("ISOLATION", r["outcome"] == "DENIED_BY_IDENTITY", "Dustin's permission is Dustin's only: Other is denied")
    grant(tp, TI, OTHER, HIST)
    r, conv_o = history(tp, TI, OTHER)
    rec("ISOLATION", r["outcome"] == "COMPLETED" and conv_o == [], "with its own permission, Other reads ITS conversation: empty — none of Dustin's")
    r, conv_me = history(tp, TI, ME)
    rec("ISOLATION", r["outcome"] == "COMPLETED" and any(MARK in x["text"] for x in conv_me), "Dustin's history has his conversation", "%d messages" % len(conv_me))
    cf = data_dir(TI) / "tass" / "conversations" / (ME + ".json")
    rec("ISOLATION", cf.exists() and (cf.stat().st_mode & 0o777) == 0o600 and MARK in cf.read_text(), "the conversation is one private (0600) file for Dustin in TASS's own data folder")

    # ---- where the words are NOT ----
    state_txt = json.dumps(tp.state())
    st, j = tp.get(f"/api/installations/{TI}/activity")
    act_text = j.get("text", "")
    proc_files = list((W / "processor-profile").rglob("*"))
    proc_hits = [str(p.relative_to(W)) for p in proc_files if p.is_file() and "/data/" not in str(p) and MARK in p.read_text(errors="replace")]
    id_hits = [str(p.relative_to(W)) for p in (W / "identity-profile").rglob("*") if p.is_file() and MARK in p.read_text(errors="replace")]
    id_console = MARK in (W / "identity.console.txt").read_text(errors="replace")
    obs = (data_dir(TI) / "observations.jsonl").read_text()
    rec("PRIVACY", MARK not in state_txt and MARK not in act_text and not proc_hits, "Processor keeps no copy of what was said: not in its state, records, logs or the copyable activity", ", ".join(proc_hits) or "clean")
    rec("PRIVACY", not id_hits and not id_console, "Identity never sees the words (only a SHA-256 of each request)", ", ".join(id_hits) or "clean")
    rec("PRIVACY", MARK not in obs and SEND in obs, "TASS's own audit log records who/when/which grant — not the words")

    # ---- forgetting ----
    GC = grant(tp, TI, ME, CLEAR)
    r = ask(tp, TI, ME, CLEAR, "yes")
    _, c_after = history(tp, TI, ME)
    rec("FORGET", r["outcome"] == "ERROR" and "BAD_INPUT" in r["detail"] and len(c_after) == len(conv_me), "forgetting needs the exact phrase; a plain 'yes' deletes nothing")

    # ---- revocation ----
    tp.post(f"/api/installations/{TI}/grants/{GS}/revoke")
    r = ask(tp, TI, ME, SEND, "after revoke")
    rec("REVOKE", r["outcome"] == "DENIED_BY_IDENTITY" and "GRANT_REVOKED" in r["detail"], "revoked: the next message is DENIED", r["detail"][:50])
    GS = grant(tp, TI, ME, SEND)

    # ---- Identity unavailable ----
    stop("identity")
    tp.until(lambda s: not s["verified"], 30, "authority unavailable")
    r = ask(tp, TI, ME, SEND, "while Identity is down")
    rec("ERRORS", r["outcome"] == "AUTHORITY_UNAVAILABLE", "Identity down: talking to TASS fails closed (AUTHORITY UNAVAILABLE)")
    s = tp.until(lambda s: entrance(s, TI).get("connected") is False, 25, "tass notices")
    rec("ERRORS", entrance(s, TI).get("connected") is False and entrance(s, TI).get("ready") is False, "TASS itself reports NOT CONNECTED / NOT READY (no green state without Identity)")
    start_identity()
    s = tp.until(lambda s: s["verified"] and inst(s, TI)["identity"] is not None and entrance(s, TI).get("connected"), 40, "connected again")
    rec("ERRORS", entrance(s, TI).get("connected") is True, "Identity back: CONNECTED again on a fresh signed answer")

    # ---- the others are unaffected ----
    WK = install(tp, "p1-deterministic-worker")
    TH = install(tp, "thought")
    tp.post(f"/api/installations/{WK}/start")
    tp.post(f"/api/installations/{TH}/start")
    tp.until(lambda s: inst(s, WK)["runtime"]["state"] == "RUNNING" and inst(s, TH)["runtime"]["state"] == "RUNNING", 20, "others running")
    r = ask(tp, WK, ME, WCAP, "hi")
    st, j = tp.post(f"/api/installations/{TH}/self-test")
    rec("OTHERS", r["outcome"] == "DENIED_BY_IDENTITY" and (j.get("self_test") or {}).get("result") == "PASS", "a TASS permission gives nothing on the worker; Thought self-test PASS beside TASS")

    # ---- the owner's screen ----
    if args.shots:
        ui_platform(LINK["url"], [TI, WK, TH], tp, TI, OTHER)
        tp.until(lambda s: entrance(s, TI).get("ready") and status_of(s, TI).get("model", {}).get("state") == "REACHABLE", 30, "tass ready after Start all")
        ui_check(LINK["url"], TI, "Dustin")
        tp.post(f"/api/installations/{TI}/grants/{GS}/revoke")
        ui_after_revoke(LINK["url"], TI)
        GS = grant(tp, TI, ME, SEND)  # a live grant again, so a refusal after Stop can only mean "not running"

    # ---- Stop ----
    _, conv_pre = history(tp, TI, ME)  # what TASS has right before Stop and the full restart
    pid = tass_pid(tp, TI)
    ep = urllib.parse.urlparse(endpoint(TI)).port
    tp.post(f"/api/installations/{TI}/stop")
    s = tp.until(lambda s: inst(s, TI)["runtime"]["state"] == "STOPPED", 30, "stopped")
    time.sleep(0.3)
    rec("STOP", inst(s, TI)["runtime"]["state"] == "STOPPED" and not pid_alive(pid) and worker_pids(TI) == [] and not port_open(ep),
        "Stop: STOPPED, the TASS process is gone and its port no longer listens (OS check)", "port %d" % ep)
    r = ask(tp, TI, ME, SEND, "after stop")
    rec("STOP", r["outcome"] == "NOT_RUNNING", "after Stop, even with a live grant, the message is refused as NOT RUNNING (nothing auto-starts)", r["outcome"])

    # ---- restart everything: TASS remembers ----
    stop("processor")
    time.sleep(0.5)
    rec("RESTART", worker_pids(WK) == [] and worker_pids(TH) == [] and worker_pids(TI) == [], "Processor exit stops every service it started")
    stop("identity")
    start_identity()
    tp = TP(start_processor())
    s = tp.until(lambda s: s["verified"] and inst(s, TI) and inst(s, TI)["identity"] is not None, 40, "back")
    tp.post(f"/api/installations/{TI}/start")
    tp.until(lambda s: entrance(s, TI).get("ready"), 25, "ready again")
    r, conv_r = history(tp, TI, ME)
    rec("RESTART", inst(tp.state(), TI)["record"]["installation_id"] == TI and r["outcome"] == "COMPLETED" and conv_pre and [x["text"] for x in conv_r] == [x["text"] for x in conv_pre],
        "after restarting Identity, Processor and TASS: same installation, and TASS still has the same conversation", "%d messages" % len(conv_r))
    r2 = ask(tp, TI, ME, SELF)
    rec("RESTART", r2["outcome"] == "COMPLETED" and (out(r2).get("born") or {}).get("at") == BORN and len(out(r2).get("seeds") or []) == 3,
        "after the full restart TASS is still the same one: born once, same birth time, same three first thoughts", BORN or "")
    r = ask(tp, TI, ME, CLEAR, "forget everything")
    _, conv_f = history(tp, TI, ME)
    rec("FORGET", r["outcome"] == "COMPLETED" and out(r).get("deleted_messages") == len(conv_pre) and conv_f == [] and not cf.exists(),
        "'forget everything' deletes Dustin's conversation file; history is empty", out(r).get("headline", r["detail"])[:60])
    tp.post(f"/api/installations/{TI}/stop")
    tp.until(lambda s: inst(s, TI)["runtime"]["state"] == "STOPPED", 30, "final stop")
    model.close()

    # ---- no credentials in logs or in the copyable activity ----
    cred = json.loads((W / "processor-profile" / "installations" / TI / "credential.json").read_text())
    texts = act_text + "".join(p.read_text(errors="replace") for p in [W / "processor-profile" / "logs" / "processor.jsonl"] +
                               list((W / "processor-profile" / "installations" / TI / "logs").glob("*")))
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
    lines = [f"TASS acceptance — {time.strftime('%Y-%m-%d %H:%M:%S %Z')}", "environment: " + " ".join(os.uname()[0:3:2]),
             "evidence: " + json.dumps(evidence, sort_keys=True), ""] + [f"{w} [{a}] {n}" + (f"  — {d}" if d != "" else "") for a, w, n, d in results]
    lines.append(f"\nRESULT: {'PASS' if not failed else 'FAIL'} ({len(results)} checks, {len(failed)} failed)")
    (W / "p1-tass-report.txt").write_text("\n".join(lines) + "\n")
    print(lines[-1].strip())
    print("report:", W / "p1-tass-report.txt")
    sys.exit(0 if not failed else 1)
