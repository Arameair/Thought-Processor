"""Workstation + TASS acceptance — REAL Identity (no-open-link-file build) + REAL
Thought Processor + REAL TASS 0.2.0 processes. Two labelled TEST STAND-INS:
GitHub (device flow) and the local model (Ollama's API on 127.0.0.1:11434,
recording exactly what TASS sends it). The script plays the Admin in
Identity's own approval page.

Milestone checked: open the workstation with a single-use ticket, open a real
project file, ask TASS about it, get an answer from the (stand-in) local model
that demonstrably saw the file — with Identity authorizing every step, ONE
owner person identity, no private link on disk, no secret in process args.

  python3 -I acceptance/p0_workstation_tass.py --identity-src DIR --tass-src DIR --work DIR [--shots DIR]
"""
import argparse, hashlib, json, os, re, shutil, signal, socket, subprocess, sys, threading, time, urllib.error, urllib.request
import http.cookiejar
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path

ap = argparse.ArgumentParser()
ap.add_argument("--identity-src", required=True)
ap.add_argument("--tass-src", required=True)
ap.add_argument("--work", required=True)
ap.add_argument("--shots", default="")
args = ap.parse_args()
W = Path(args.work).resolve()
if W.exists():
    shutil.rmtree(W)
W.mkdir(parents=True, mode=0o700)
PROC_REPO = Path(__file__).resolve().parents[2]
ID_SRC, TASS_SRC = Path(args.identity_src).resolve(), Path(args.tass_src).resolve()
PASS_PHRASE = "correct horse battery staple 2026"


def free_port():
    with socket.socket() as s:
        s.bind(("127.0.0.1", 0))
        return s.getsockname()[1]


IP, TPP, FX, MODEL_PORT = free_port(), free_port(), free_port(), free_port()
IDENTITY, PROCESSOR = f"http://127.0.0.1:{IP}", f"http://127.0.0.1:{TPP}"
NOPROXY = urllib.request.build_opener(urllib.request.ProxyHandler({}))
ENV = {k: v for k, v in os.environ.items() if not k.lower().endswith("_proxy")}
ENV.update(IDENTITY_LOCATOR_DIR=str(W / "locators"), IDENTITY_GITHUB_CLIENT_ID="IvFIXTURE00000001", TP_IDENTITY_DEFAULT_URL="-")
results, procs, RUN = [], {}, {"code": ""}


def rec(ok, name, detail=""):
    results.append(bool(ok))
    print(("PASS " if ok else "FAIL ") + name + (("  — " + str(detail)) if detail != "" else ""), flush=True)


class Quiet(BaseHTTPRequestHandler):
    def log_message(self, *a):
        pass

    def send(self, code, obj):
        b = json.dumps(obj).encode()
        self.send_response(code)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(b)))
        self.end_headers()
        self.wfile.write(b)


class GH(Quiet):
    def do_POST(self):
        self.rfile.read(int(self.headers.get("Content-Length", 0)))
        if self.path == "/login/device/code":
            return self.send(200, {"device_code": "fx", "user_code": "FIXT-URE1", "verification_uri": f"http://127.0.0.1:{FX}/x", "expires_in": 300, "interval": 1})
        if self.path == "/login/oauth/access_token":
            return self.send(200, {"access_token": "ghu_FIXTUREonlyNOTrealTOKEN0000000000", "expires_in": 28800})
        self.send(404, {})

    def do_GET(self):
        if self.path == "/user":
            return self.send(200, {"login": "fixture-owner", "id": 4242, "name": "Fixture Owner", "type": "User", "html_url": "https://github.com/x"})
        if self.path.startswith("/user/installations"):
            return self.send(200, {"total_count": 0, "installations": []})
        self.send(404, {})


MODEL_SEEN = []


class Model(Quiet):
    """TEST STAND-IN for Ollama. Answers from what it was actually shown."""

    def do_GET(self):
        if self.path == "/v1/models":
            return self.send(200, {"data": [{"id": "granite-test"}]})
        if self.path == "/api/tags":
            return self.send(200, {"models": [{"name": "qwen3.5:4b"}]})
        if self.path == "/api/version":
            return self.send(200, {"version": "0.17.4"})
        self.send(404, {})

    def do_POST(self):
        body = json.loads(self.rfile.read(int(self.headers.get("Content-Length", 0))) or b"{}")
        if self.path == "/api/show":
            return self.send(200, {"capabilities": ["completion", "vision", "thinking"]})
        if self.path == "/v1/chat/completions":
            MODEL_SEEN.append(body)
            return self.send(200, {"choices": [{"message": {"role": "assistant", "content": "STAND-IN OPENAI: authorized local model answered"}}]})
        if self.path != "/api/chat":
            return self.send(404, {})
        MODEL_SEEN.append(body)
        sysmsgs = [m["content"] for m in body["messages"] if m["role"] == "system"]
        fileblock = [m for m in sysmsgs if "----- file text -----" in m]
        if fileblock:
            text = fileblock[0].split("----- file text -----\n", 1)[1].rsplit("\n----- end of file text -----", 1)[0]
            first = text.strip().splitlines()[0]
            funcs = sorted(set(re.findall(r"^func (\w+)", text, re.M)))
            ans = "STAND-IN MODEL saw %d lines. First line: %s. Functions: %s." % (len(text.splitlines()), first, ", ".join(funcs[:5]))
        else:
            ans = "STAND-IN MODEL: no file this time; I saw %d earlier messages." % (len(body["messages"]) - 2)
        self.send(200, {"message": {"role": "assistant", "content": ans}, "done": True})


for port, h in ((FX, GH), (MODEL_PORT, Model)):
    threading.Thread(target=ThreadingHTTPServer(("127.0.0.1", port), h).serve_forever, daemon=True).start()


def digest(d):
    m = json.loads((d / "feature.json").read_text())
    ms = hashlib.sha256((d / "feature.json").read_bytes()).hexdigest()
    b = "thought.feature-package/1\nmanifest %s\n" % ms + "".join("%s  %s\n" % (f["sha256"], f["path"]) for f in sorted(m["files"], key=lambda f: f["path"]))
    return hashlib.sha256(b.encode()).hexdigest()


TASS_PIN = digest(TASS_SRC / "feature")
LAUNCH_PIN = re.search(r"^TASS_ARTIFACT=([0-9a-f]{64})$", (PROC_REPO / "run-p1.sh").read_text(), re.M).group(1)


def start(name, cmd):
    log = open(W / (name + ".console.txt"), "ab")
    procs[name] = subprocess.Popen(cmd, stdout=log, stderr=subprocess.STDOUT, stdin=subprocess.DEVNULL, env=ENV, start_new_session=True)


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


def start_processor():
    (W / "processor.console.txt").write_bytes(b"")
    start("processor", [str(W / "bin/processor"), "-port", str(TPP), "-data", str(W / "processor-profile"), "-no-browser", "-no-link-file",
                        "-identity-default-url", IDENTITY, "-workstation-allow-test-github", "-trust-feature-dir", str(TASS_SRC / "feature"),
                        "-trust-feature-artifact", TASS_PIN, "-workstation-repo", "processor=" + str(PROC_REPO),
                        "-workstation-repo", "identity=" + str(ID_SRC), "-workstation-repo", "tass=" + str(TASS_SRC)])
    end = time.time() + 20
    while time.time() < end:
        m = re.search(r"#code=([0-9a-f]{48})", (W / "processor.console.txt").read_text(errors="replace"))
        if m:
            RUN["code"] = m.group(1)
            return
        time.sleep(0.2)
    raise SystemExit("processor did not start")


class Browser:
    def __init__(self):
        self.op = urllib.request.build_opener(urllib.request.ProxyHandler({}), urllib.request.HTTPCookieProcessor(http.cookiejar.CookieJar()))
        self.csrf = ""

    def req(self, method, path, body=None, csrf=True, timeout=150):
        data = json.dumps(body or {}).encode() if method == "POST" else None
        r = urllib.request.Request(PROCESSOR + path, data=data, method=method)
        if method == "POST":
            r.add_header("Content-Type", "application/json")
            r.add_header("Origin", PROCESSOR)
            if csrf:
                r.add_header("X-TP-CSRF", self.csrf)
        try:
            with self.op.open(r, timeout=timeout) as resp:
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
            except (KeyError, TypeError, IndexError, AttributeError):
                pass
            if time.time() > end:
                return j
            time.sleep(0.5)


def ticket_browser():
    st, j = Browser().req("POST", "/api/launch-ticket", {"code": RUN["code"]})
    b = Browser()
    return b, b.sign_in(j.get("ticket", "")), j.get("ticket", "")


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
    return (st, j) if st != 200 else post("/api/approval/decide", body, j["csrf"])


def approve(url):
    return admin(url, {"decision": "approve", "passphrase": PASS_PHRASE})


def secret_scan(tokens):
    hits = []
    for root in (W / "identity-profile", W / "processor-profile"):
        for p in root.rglob("*"):
            if p.is_file():
                b = p.read_bytes()
                for name, tok in tokens.items():
                    if tok and tok.encode() in b:
                        hits.append("%s in %s" % (name, p.relative_to(W)))
    return hits


def shot(name):
    if not args.shots:
        return
    from playwright.sync_api import sync_playwright
    st, j = Browser().req("POST", "/api/launch-ticket", {"code": RUN["code"]})
    with sync_playwright() as pw:
        br = pw.chromium.launch()
        pg = br.new_page(viewport={"width": 1600, "height": 1000}, color_scheme="dark")
        pg.goto(PROCESSOR + "/workstation#code=" + j["ticket"])
        pg.wait_for_selector("[data-repo='processor']", timeout=20000)
        pg.click("[data-repo='processor']")
        pg.wait_for_timeout(800)
        pg.click("[data-dir='processor']")
        pg.wait_for_timeout(800)
        pg.click("[data-file='processor/main.go']")
        pg.wait_for_timeout(800)
        pg.wait_for_selector("#ws-tass-q", state="visible", timeout=20000)
        pg.fill("#ws-tass-q", "What does this file do, and where does it start the program?")
        pg.click("#ws-tass-send")
        pg.wait_for_selector(".ws-msg.tass", timeout=60000)
        pg.wait_for_timeout(500)
        Path(args.shots).mkdir(parents=True, exist_ok=True)
        pg.screenshot(path=str(Path(args.shots) / (name + ".png")), full_page=False)
        br.close()


def main():
    (W / "bin").mkdir()
    subprocess.run(["go", "build", "-o", str(W / "bin/identity"), "."], cwd=ID_SRC, check=True)
    subprocess.run(["go", "build", "-o", str(W / "bin/processor"), "."], cwd=PROC_REPO / "processor", check=True)
    rec(TASS_PIN == LAUNCH_PIN, "S0 the TASS package under test is exactly the one the launcher pins", TASS_PIN[:12])
    start("identity", [str(W / "bin/identity"), "-port", str(IP), "-data", str(W / "identity-profile"), "-no-open-link-file",
                       "-trust-fixture-artifact", TASS_PIN, "-test-github-web", f"http://127.0.0.1:{FX}", "-test-github-api", f"http://127.0.0.1:{FX}"])
    assert wait_http(IDENTITY + "/api/health")
    start_processor()
    id_key = (re.search(r"#key=([0-9a-f]+)", (W / "identity.console.txt").read_text(errors="replace")) or [None, ""])[1]

    # ---- private links ----
    rec(not (W / "identity-profile/runtime/open-link.txt").exists() and not (W / "processor-profile/runtime/operator-link.txt").exists(),
        "S1 neither Identity nor Processor writes its private link to disk (-no-open-link-file / -no-link-file)")
    leaks = [n for n in ("identity", "processor") if (RUN["code"] in Path(f"/proc/{procs[n].pid}/cmdline").read_bytes().decode(errors="replace")
                                                    or (id_key and id_key in Path(f"/proc/{procs[n].pid}/cmdline").read_bytes().decode(errors="replace")))]
    rec(not leaks, "S2 no private link or key in either service's process arguments")
    b, st, tk = ticket_browser()
    rec(st == 200 and Browser().sign_in(tk) == 401, "S3 the browser gets a single-use ticket (replay refused)")
    rec(Browser().req("POST", "/api/ws/tass/ask", {"question": "x"})[0] == 401, "S4 no session -> TASS route refused (401)")
    st, j = b.req("POST", "/api/ws/tass/ask", {"question": "x"}, csrf=False)
    rec(st == 403, "S5 no anti-forgery token -> refused (403)")

    # ---- Identity: pair, GitHub, Admin ----
    b.req("POST", "/api/connect/choose", {"url": IDENTITY})
    b.req("POST", "/api/connect/request", {"url": IDENTITY})
    time.sleep(0.7)
    code = re.findall(r"\b([A-HJ-NP-Z2-9]{4}-[A-HJ-NP-Z2-9]{4})\b", (W / "identity.console.txt").read_text(errors="replace"))[-1]
    b.req("POST", "/api/connect/code", {"code": code})
    b.until("/api/state", lambda s: s["verified"], 20)
    st, j = b.req("GET", "/api/ws/tass")
    rec(st == 403 and j["gate"]["reason"] == "GITHUB_NOT_VERIFIED", "A1 TASS in the workstation is locked until GitHub is VERIFIED")
    b.req("POST", "/api/github/connect")
    s = b.until("/api/state", lambda s: s["github"]["state"] == "VERIFIED", 30)
    st, j = b.req("POST", "/api/admin/enroll")
    admin(j.get("approval_url"), {"decision": "approve", "passphrase": PASS_PHRASE, "passphrase_confirm": PASS_PHRASE, "acknowledge": True})
    s = b.until("/api/state", lambda s: s["admin"]["state"] == "ENROLLED", 15)
    rec(s["github"]["state"] == "VERIFIED" and s["admin"]["state"] == "ENROLLED", "A2 paired; GitHub VERIFIED via Identity (TEST FIXTURE); Admin established in Identity's page")

    # ---- TASS setup from the workstation ----
    t = b.req("GET", "/api/ws/tass")[1]
    rec(t["step"] == "INSTALL", "T1 workstation TASS panel: first step INSTALL", t["step"])
    st, j = b.req("POST", "/api/ws/tass/install")
    rec(st == 200 and "/approve#a=apr_" in j.get("approval_url", ""), "T1 Install asks the Admin in IDENTITY's page", j.get("version"))
    approve(j["approval_url"])
    t = b.until("/api/ws/tass", lambda t: t["step"] == "START", 40)
    rec(t["step"] == "START" and t["version"] == "0.2.0", "T1 Admin approved: TASS 0.2.0 installed (its own installation identity)", t.get("installation_id"))
    t = b.req("POST", "/api/ws/tass/start")[1]
    t = b.until("/api/ws/tass", lambda t: t["step"] == "OWNER", 30)
    rec(t["step"] == "OWNER", "T2 TASS running; next: who TASS talks to")
    st, j = b.req("POST", "/api/ws/tass/owner")
    owner = j.get("owner", {})
    st2, j2 = b.req("POST", "/api/ws/tass/owner")
    ids = [e for e in b.req("GET", "/api/state")[1]["identities"] if e["kind"] == "person"]
    rec(st == 200 and owner.get("name") == "fixture-owner (owner)" and j2["owner"]["id"] == owner["id"] and len(ids) == 1,
        "T2 ONE owner person identity, created once from the Admin's GitHub sign-in; asking again reuses it", owner.get("id"))
    t = b.until("/api/ws/tass", lambda t: t["step"] == "ALLOW_CHAT", 15)
    st, j = b.req("POST", "/api/ws/tass/ask", {"question": "hello"})
    rec(j.get("outcome") == "DENIED_BY_IDENTITY", "T3 before the Admin approves, Identity DENIES a message (nothing sent)", j.get("detail", "")[:70])
    st, j = b.req("POST", "/api/ws/tass/allow", {"what": "chat"})
    st_a, ja = approve(j["grant"]["approval_url"])
    t = b.until("/api/ws/tass", lambda t: t["chat_grant"]["status"] == "ACTIVE", 30)
    exp = t["chat_grant"].get("expires_at", "")
    rec(t["chat_grant"]["status"] == "ACTIVE" and exp, "T3 Admin approved tass.chat.send for the owner (12-hour grant)", exp)
    rec(t["step"] == "ALLOW_MODEL", "T4 no model yet -> ALLOW_MODEL", t["step"])
    st, j = b.req("POST", "/api/ws/tass/allow", {"what": "model"})
    approve(j["grant"]["approval_url"])
    t = b.until("/api/ws/tass", lambda t: t["step"] == "SET_MODEL", 30)
    st, j = b.req("POST", "/api/ws/tass/model", {"model": "qwen3.5:4b", "kind": "ollama", "url": f"http://127.0.0.1:{MODEL_PORT}"})
    t = b.until("/api/ws/tass", lambda t: t["step"] == "READY", 40)
    rec(j["result"]["outcome"] == "COMPLETED" and t["step"] == "READY" and t["model"].get("state") == "REACHABLE",
        "T4 model set to qwen3.5:4b (stand-in on an isolated loopback port): READY", json.dumps(t["model"])[:100])

    # ---- the milestone ----
    st, f = b.req("GET", "/api/ws/file?repo=processor&path=processor/main.go")
    file_text = f.get("text", "")
    rec(st == 200 and file_text == (PROC_REPO / "processor/main.go").read_text(), "M1 open a real project file: processor/main.go (exact bytes)", "%d bytes" % f.get("size", 0))
    n_before = len(MODEL_SEEN)
    st, j = b.req("POST", "/api/ws/tass/ask", {"question": "What does this file do?", "repo": "processor", "path": "processor/main.go"})
    reply = (j.get("tass") or {}).get("reply", "")
    funcs = sorted(set(re.findall(r"^func (\w+)", file_text, re.M)))
    first = file_text.strip().splitlines()[0]
    rec(j.get("outcome") == "COMPLETED" and ("First line: " + first) in reply and ("saw %d lines" % len(file_text.splitlines())) in reply and funcs[0] in reply,
        "M2 TASS answered about the file through the local model, and the answer shows the model saw it", reply[:120])
    seen = MODEL_SEEN[n_before] if len(MODEL_SEEN) > n_before else {}
    sysm = [m["content"] for m in seen.get("messages", []) if m["role"] == "system"]
    rec(any(file_text.strip() in m and "read-only" in m and "processor/main.go" in m for m in sysm) and seen.get("options", {}).get("num_ctx") == 16384,
        "M3 the model was shown the exact file text, labelled read-only, with room for it (num_ctx 16384)")
    about = (j.get("tass") or {}).get("conversation", [{}])[-2].get("about", {})
    rec(about.get("path") == "processor/main.go" and about.get("file_sha256") == hashlib.sha256(file_text.encode()).hexdigest() and "text" not in about,
        "M4 TASS's conversation keeps only the file's name, commit and fingerprint", about.get("head_sha", "")[:12])
    st, j2 = b.req("POST", "/api/ws/tass/ask", {"question": "And what should I read next?"})
    seen2 = MODEL_SEEN[-1]
    rec(j2.get("outcome") == "COMPLETED" and not any(file_text.strip()[:200] in m["content"] for m in seen2["messages"])
        and any("What does this file do?" == m["content"] for m in seen2["messages"]),
        "M5 follow-up remembers the conversation but the file text is NOT resent", (j2.get("tass") or {}).get("reply", "")[:80])
    marker = file_text.strip().splitlines()[-1]
    stored = [str(p.relative_to(W)) for p in (W / "processor-profile").rglob("*") if p.is_file() and len(marker) > 3
              and (file_text.encode()[:400] in p.read_bytes() or reply.encode() in p.read_bytes())]
    rec(not stored, "M6 neither the file text nor TASS's answer is stored by Processor (TASS keeps its own private conversation)", ", ".join(stored)[:120])

    # ---- authorization stays with Identity ----
    bad = [p for p in ("../identity/go.mod", "/etc/passwd", ".git/config") if b.req("POST", "/api/ws/tass/ask", {"question": "x", "repo": "processor", "path": p})[0] != 403]
    rec(not bad, "Z1 TASS can't be shown files outside the allowlisted repositories (traversal / absolute / .git refused)", ",".join(bad))
    tv = b.req("GET", "/api/ws/tass")[1]
    s = b.req("GET", "/api/state")[1]
    gid = [g["grant"]["grant_id"] for v in s["installations"] if v["record"]["installation_id"] == tv["installation_id"]
           for g in v["identity"]["grants"] if g["grant"]["capability"] == "tass.chat.send" and g["status"] == "ACTIVE"][0]
    b.req("POST", f"/api/installations/{tv['installation_id']}/grants/{gid}/revoke")
    st, j = b.req("POST", "/api/ws/tass/ask", {"question": "still there?", "repo": "processor", "path": "processor/main.go"})
    rec(j.get("outcome") == "DENIED_BY_IDENTITY", "Z2 after the chat permission is revoked, Identity denies the next question immediately", j.get("detail", "")[:70])
    st, j = b.req("POST", "/api/ws/tass/allow", {"what": "chat"})
    approve(j["grant"]["approval_url"])
    b.until("/api/ws/tass", lambda t: t["step"] == "READY", 30)
    shot("workstation-tass-file")
    st, j = b.req("POST", "/api/ws/tass/model", {"kind": "openai-compatible", "url": f"http://127.0.0.1:{MODEL_PORT}", "model": "granite-test"})
    rec(st == 200 and j["result"]["outcome"] == "COMPLETED", "G1 local OpenAI-compatible configuration uses the Identity-authorized model action")
    b.until("/api/ws/tass", lambda t: t["step"] == "READY", 30)
    st, j = b.req("POST", "/api/ws/tass/ask", {"question": "hello local model"})
    rec(st == 200 and "STAND-IN OPENAI" in str(j), "G2 OpenAI-compatible chat completion returns through TASS")
    st, j = b.req("POST", "/api/ws/tass/model", {"kind": "openai-compatible", "url": "http://192.0.2.1:8080", "model": "granite-test"})
    rec(j.get("result", {}).get("outcome") != "COMPLETED", "G3 TASS refuses a non-loopback model address")
    # owner reuse after Processor forgets its setting
    stop("processor")
    sp = W / "processor-profile/settings.json"
    sj = json.loads(sp.read_text())
    sj.pop("owner_subject", None)
    sp.write_text(json.dumps(sj))
    start_processor()
    b, _, _ = ticket_browser()
    b.until("/api/state", lambda s: s["verified"], 30)
    st, j = b.req("POST", "/api/ws/tass/owner")
    ids = [e for e in b.req("GET", "/api/state")[1]["identities"] if e["kind"] == "person"]
    rec(j.get("owner", {}).get("id") == owner["id"] and len(ids) == 1, "Z3 the existing person identity is reused, never a second one")
    stop("identity", signal.SIGKILL)
    st, j = b.req("POST", "/api/ws/tass/ask", {"question": "x", "repo": "processor", "path": "processor/main.go"})
    rec(st == 403 and j["gate"]["reason"] == "IDENTITY_NOT_CONNECTED", "Z4 Identity stopped -> the workstation and TASS refuse at once (fail closed)")
    hits = secret_scan({"processor run code": RUN["code"], "identity key": id_key})
    rec(not hits, "Z5 no private link/key anywhere in either profile after the whole run (incl. Identity killed)", "; ".join(hits))


try:
    main()
finally:
    for n in ("processor", "identity"):
        stop(n)
print("\n%d/%d PASS" % (sum(results), len(results)))
sys.exit(0 if results and all(results) else 1)
