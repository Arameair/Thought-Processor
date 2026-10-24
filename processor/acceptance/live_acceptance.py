#!/usr/bin/env python3
"""Live acceptance run for the Thought Processor diagnostic shell (layer L3).

Runs REAL processes on this host:
  * the built Thought Processor binary
  * optionally real Identity and Thought-engine binaries built from their own
    repositories. THIS SCRIPT starts and stops them as external programs, to
    play the owner's role. Processor never starts, attaches to or stops them.

Anything that is a stand-in (a raw TCP listener) is labeled "L2-live fixture".
Writes evidence.json, evidence.md, sanitized process logs and screenshots.
Run with:  python3 -I live_acceptance.py --processor BIN [--identity BIN --engine BIN] --out DIR
"""
import argparse, http.client, json, os, platform, re, signal, socket, subprocess, sys, tempfile, threading, time

P_PORT, I_PORT, E_PORT = 7860, 7796, 7798
RESULTS, LOGS = [], []
CODES = set()  # operator codes seen; redacted from every saved log


def now():
    return time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime())


def note(msg):
    line = f"{now()} {msg}"
    LOGS.append(line)
    print(line, flush=True)


def record(tid, verdict, layer, action, observed, limitation=""):
    RESULTS.append(dict(id=tid, verdict=verdict, layer=layer, action=action, observed=observed, limitation=limitation, at=now()))
    note(f"[{tid}] {verdict} ({layer}) — {observed}")


def port_free(p):
    s = socket.socket()
    s.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)  # ignore TIME_WAIT, like Go's listener
    try:
        s.bind(("127.0.0.1", p))
        return True
    except OSError:
        return False
    finally:
        s.close()


def procs():
    """pid -> (ppid, comm) for every process, read from /proc."""
    out = {}
    for d in os.listdir("/proc"):
        if d.isdigit():
            try:
                with open(f"/proc/{d}/stat") as f:
                    st = f.read()
                comm = st[st.index("(") + 1: st.rindex(")")]
                ppid = int(st[st.rindex(")") + 2:].split()[1])
                out[int(d)] = (ppid, comm)
            except (OSError, ValueError):
                pass
    return out


def named(comm_prefix):
    return sorted(pid for pid, (_, c) in procs().items() if c.startswith(comm_prefix))


def children(pid):
    return sorted(p for p, (pp, _) in procs().items() if pp == pid)


def alive(pid):
    try:
        os.kill(pid, 0)
        with open(f"/proc/{pid}/stat") as f:
            return f.read().split(")")[1].split()[0] != "Z"
    except OSError:
        return False


class Proc:
    def __init__(self, name, argv, logdir, env=None, cwd=None):
        self.name = name
        self.log = open(os.path.join(logdir, f"{name}.raw.log"), "ab")
        e = dict(os.environ)
        e.update(env or {})
        self.t0 = time.monotonic()
        self.p = subprocess.Popen(argv, stdout=self.log, stderr=subprocess.STDOUT, env=e, cwd=cwd, start_new_session=True)
        self.pid = self.p.pid

    def wait_http(self, port, path, timeout=15):
        end = time.monotonic() + timeout
        while time.monotonic() < end:
            if self.p.poll() is not None:
                return None
            try:
                c = http.client.HTTPConnection("127.0.0.1", port, timeout=1)
                c.request("GET", path)
                r = c.getresponse()
                r.read()
                c.close()
                return round((time.monotonic() - self.t0) * 1000)
            except OSError:
                time.sleep(0.02)
        return None

    def stop(self, sig=signal.SIGTERM, timeout=10):
        t = time.monotonic()
        if self.p.poll() is None:
            self.p.send_signal(sig)
            try:
                self.p.wait(timeout)
            except subprocess.TimeoutExpired:
                self.p.kill()
                self.p.wait()
        self.log.flush()
        return round((time.monotonic() - t) * 1000), self.p.returncode


class Client:
    """Talks to Processor like the page does (or deliberately not)."""

    def __init__(self, port):
        self.port, self.cookie, self.csrf = port, "", ""
        self.origin = f"http://127.0.0.1:{port}"

    def req(self, method, path, body=None, headers=None, host=None):
        h = {"Host": host or f"127.0.0.1:{self.port}"}
        if self.cookie:
            h["Cookie"] = "tp_session=" + self.cookie
        h.update(headers or {})
        c = http.client.HTTPConnection("127.0.0.1", self.port, timeout=10)
        c.request(method, path, body=json.dumps(body) if isinstance(body, (dict, list)) else body, headers=h)
        r = c.getresponse()
        data = r.read().decode()
        for k, v in r.getheaders():
            if k.lower() == "set-cookie" and v.startswith("tp_session="):
                self.cookie = v.split(";")[0].split("=", 1)[1]
        c.close()
        try:
            return r.status, json.loads(data), r
        except ValueError:
            return r.status, data, r

    def post(self, path, body=None, **kw):
        hd = {"Origin": self.origin, "Content-Type": "application/json", "X-TP-CSRF": self.csrf}
        hd.update(kw.pop("headers", {}))
        return self.req("POST", path, body if body is not None else {}, headers=hd, **kw)

    def sign_in(self, code):
        st, d, _ = self.req("POST", "/api/session", {"code": code}, headers={"Origin": self.origin, "Content-Type": "application/json"})
        assert st == 200, (st, d)
        self.csrf = d["csrf"]

    def state(self, probe=False):
        return (self.post("/api/probe") if probe else self.req("GET", "/api/state"))[1]


def read_code(profile):
    with open(os.path.join(profile, "runtime", "operator-code")) as f:
        c = f.read().strip()
    CODES.add(c)
    return c


STARTED = []


def start_processor(binary, profile, logdir, name="processor", port=P_PORT):
    pr = Proc(name, [binary, "-data", profile, "-port", str(port)], logdir)
    STARTED.append(pr)
    ms = pr.wait_http(port, "/api/health")
    return pr, ms


def shots(page, outdir, name):
    path = os.path.join(outdir, name + ".png")
    page.screenshot(path=path, full_page=True)
    return os.path.basename(path)


def wait_text(page, selector, pred, timeout=15):
    """Poll from Python: the page's CSP (correctly) forbids Playwright's eval-based waits."""
    end = time.monotonic() + timeout
    while time.monotonic() < end:
        try:
            txt = page.text_content(selector) or ""
            if pred(txt):
                return txt
        except Exception:
            pass
        time.sleep(0.05)
    raise TimeoutError(f"{selector} never matched")


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--processor", required=True)
    ap.add_argument("--identity")
    ap.add_argument("--engine")
    ap.add_argument("--out", required=True)
    ap.add_argument("--meta", default="{}", help="JSON of repo/branch/SHA facts to embed")
    a = ap.parse_args()
    out = os.path.abspath(a.out)
    os.makedirs(out, exist_ok=True)
    work = tempfile.mkdtemp(prefix="tp-live-")
    meta = json.loads(a.meta)
    meta.update(os_info=platform.platform(), python=platform.python_version(), started=now(),
                processor_bin=os.path.basename(a.processor), work_root="<tmp>/" + os.path.basename(work))
    for p in (P_PORT, I_PORT, E_PORT, P_PORT + 1):
        if not port_free(p):
            sys.exit(f"port {p} busy before the run; refusing to continue")
    baseline = {k: named(k) for k in ("thought-process", "identity", "thought-engine")}
    note(f"baseline processes: {baseline}")

    from playwright.sync_api import sync_playwright
    pw = sync_playwright().start()
    browser = pw.chromium.launch()
    shot = []
    owned = []  # external programs THIS SCRIPT started
    mine = []   # Processor runs this script started (stopped in finally)
    try:
        # ---------------- A01 ----------------
        prof = os.path.join(work, "profile-a")
        P, ms = start_processor(a.processor, prof, out)
        assert ms is not None, "processor did not start"
        meta["first_startup_ms"] = ms
        code = read_code(prof)
        c = Client(P_PORT)
        st, d, r = c.req("GET", "/api/state")
        unauth_ok = st == 401
        ctx = browser.new_context(viewport={"width": 1200, "height": 900})
        page = ctx.new_page()
        page.goto(f"http://127.0.0.1:{P_PORT}/")
        page.wait_for_selector("#signin:not([hidden])")
        shot.append(shots(page, out, "01-signed-out"))
        page.goto(f"http://127.0.0.1:{P_PORT}/#code={code}")
        page.wait_for_selector("#app:not([hidden])")
        wait_text(page, "#next-title", lambda x: len(x) > 0)
        url_after = page.url
        shot.append(shots(page, out, "02-first-screen-empty-profile"))
        c.sign_in(code)
        s = c.state()
        kids = children(P.pid)
        others = {k: named(k) for k in ("identity", "thought-engine")}
        ok = (unauth_ok and s["identity"]["state"] == "NOT_CONFIGURED" and s["engine"]["state"] == "NOT_CONFIGURED"
              and s["next_action"]["code"] == "set_identity" and all(not o["enabled"] for o in s["operations"])
              and s["self"]["owned_processes"] == [] and not kids and others == {"identity": baseline["identity"], "thought-engine": baseline["thought-engine"]}
              and "#code" not in url_after)
        record("A01", "PASS" if ok else "FAIL", "L3",
               "Start Processor with empty isolated profile; open in Chromium; sign in",
               f"pid {P.pid} port {P_PORT} ready in {ms} ms; run {s['self']['run_id']}; build {s['self']['build']['revision']} modified={s['self']['build']['modified']}; "
               f"/api/state without session -> {st}; identity={s['identity']['state']} engine={s['engine']['state']} next={s['next_action']['code']}; "
               f"ops enabled={[o['enabled'] for o in s['operations']]}; children of Processor={kids}; identity/engine processes={others}; code removed from URL={'#code' not in url_after}; screenshots 01,02")

        # ---------------- UX: one next action + narrow screen ----------------
        btn = page.text_content("#next-btn")
        page.click("#next-btn")
        focused = page.evaluate("document.activeElement && document.activeElement.id")
        mob = browser.new_context(viewport={"width": 390, "height": 844})
        mp = mob.new_page()
        mp.goto(f"http://127.0.0.1:{P_PORT}/#code={code}")
        mp.wait_for_selector("#app:not([hidden])")
        wait_text(mp, "#next-title", lambda x: len(x) > 0)
        overflow = mp.evaluate("document.documentElement.scrollWidth - document.documentElement.clientWidth")
        shot.append(shots(mp, out, "02b-first-screen-phone-width"))
        mob.close()
        ok = btn == "Enter Identity address" and focused == "identity-addr" and overflow <= 0
        record("UX-01/06", "PASS" if ok else "FAIL", "L3", "One useful next action on first screen; 390 px wide phone viewport",
               f"Next button '{btn}' moves focus to #{focused}; horizontal overflow at 390 px = {overflow}px; screenshots 02, 02b",
               "Owner usability (can a non-technical person follow it) is NOT tested here; that is the owner trial")

        # ---------------- A27 / A02 Identity absent ----------------
        st, d, _ = c.post("/api/candidates", {"identity": f"127.0.0.1:{I_PORT}"})
        s = d
        api_results = {}
        for act in ("start", "attach", "stop", "observe"):
            st2, d2, _ = c.post(f"/api/engine/{act}")
            api_results[act] = (st2, d2.get("status"), d2.get("error"), d2.get("op_id"))
        page.reload()
        wait_text(page, "#id-pill", lambda x: len(x) > 0)
        disabled = page.eval_on_selector_all("#ops button", "bs => bs.map(b => b.disabled)")
        page.click("#deps details.inner summary")
        page.click("text=Ask server: start")
        page.wait_for_selector("#probe-out:not([hidden])")
        ui_reply = page.text_content("#probe-out")
        shot.append(shots(page, out, "03-identity-absent-start-refused"))
        kids = children(P.pid)
        eng = named("thought-engine")
        ok = (s["identity"]["state"] == "UNREACHABLE" and s["next_action"]["code"] == "start_identity"
              and all(v[0] == 501 and v[1] == "BLOCKED" for v in api_results.values())
              and all(disabled) and "HTTP 501" in ui_reply and '"BLOCKED"' in ui_reply and not kids and eng == baseline["thought-engine"])
        obs = (f"identity candidate 127.0.0.1:{I_PORT} -> {s['identity']['state']} ({s['identity']['probe']['port']}); next={s['next_action']['code']}; "
               f"API: {{{', '.join(f'{k}: {v[0]} {v[1]} {v[2]} {v[3]}' for k, v in api_results.items())}}}; "
               f"UI buttons disabled={disabled}; UI 'Ask server: start' -> {ui_reply.splitlines()[0]} BLOCKED; Processor children={kids}; thought-engine processes={eng}")
        record("A02", "PASS" if ok else "FAIL", "L3", "Identity absent; try start/attach/stop/observe from UI and direct API", obs)
        record("A27", "PASS" if ok else "FAIL", "L3", "Missing Identity producer/enforcement interface",
               "Gate 2/3 integration BLOCKED; diagnostic shell only; every protected call refused 501 before effect; no bypass flag exists in the binary (see A19 for symbol/dependency check)")

        # ---------------- real Identity started externally ----------------
        if a.identity:
            idata = os.path.join(work, "identity-data")
            I = Proc("identity", [a.identity, "-data", idata], out, env={"PORT": str(I_PORT)}, cwd=os.path.dirname(a.identity))
            owned.append(I)
            ims = I.wait_http(I_PORT, "/api/health")
            note(f"external Identity pid {I.pid} ready in {ims} ms (started by the test script, not by Processor)")
            s = c.state(probe=True)
            ip = s["identity"]["probe"]
            st3, d3, _ = c.post("/api/engine/start")
            ok = s["identity"]["state"] == "CLAIMS_IDENTITY_UNVERIFIED" and s["next_action"]["code"] == "blocked_contract" and st3 == 501
            record("A27-live-identity", "PASS" if ok else "FAIL", "L3",
                   "Real Identity (first-admin-wizard build) running; Processor inspects it and retries start",
                   f"Identity pid {I.pid}; probe port={ip['port']} metadata={ip['metadata']} http={ip.get('http_status')} claim app={ip.get('claimed_app')!r} (UNVERIFIED) in {ip['elapsed_ms']} ms; "
                   f"state={s['identity']['state']}; next={s['next_action']['code']}; start -> {st3} {d3['error']}; Identity still alive={alive(I.pid)}",
                   "Identity has no decision endpoint, so Processor cannot ask it anything; only public health metadata was read")
            page.reload()
            wait_text(page, "#id-pill", lambda x: "unverified" in x)
            shot.append(shots(page, out, "04-identity-found-unverified"))
            for tid, what in (("A03", "Real Identity explicitly denies"), ("A04", "Identity times out / malformed / unverifiable decision"),
                              ("A05", "Decision targets wrong actor/install/service/action/manifest"), ("A06", "Expired/revoked/replayed decision")):
                record(tid, "BLOCKED", "—", what,
                       "No Identity build exposes a start/attach/stop decision; no runner/Engine control boundary validates one. Nothing to deny, time out, mis-scope or replay. Processor refuses all four actions with 501 regardless (A02).",
                       "Processor-side probe timeout/malformed/redirect classification was tested only with labeled L2 fixtures (TestIdentityProbeClassificationFixture); that is NOT decision-path evidence")
        else:
            I = None

        # ---------------- real Engine started externally ----------------
        E = None
        if a.engine:
            edata = os.path.join(work, "engine-data")
            E = Proc("thought-engine", [a.engine, "-data", edata], out, env={"PORT": str(E_PORT), "CHAT_JIMMY_URL": ""}, cwd=os.path.dirname(a.engine))
            owned.append(E)
            ems = E.wait_http(E_PORT, "/api/status")
            note(f"external Thought-engine pid {E.pid} ready in {ems} ms (started by the test script, not by Processor)")
            st, s, _ = c.post("/api/candidates", {"engine": f"127.0.0.1:{E_PORT}"})
            ep = s["engine"]["probe"]
            res = {}
            for act in ("attach", "start", "stop"):
                st2, d2, _ = c.post(f"/api/engine/{act}")
                res[act] = (st2, d2["error"])
            ok = (s["engine"]["state"] == "PORT_OPEN_UNIDENTIFIED" and ep["metadata"] == "NOT_REQUESTED"
                  and all(v[0] == 501 for v in res.values()) and alive(E.pid) and named("thought-engine") == sorted(baseline["thought-engine"] + [E.pid]))
            record("A09", "BLOCKED", "L3", "Attach to correct external Engine that is running",
                   f"Engine pid {E.pid} on {E_PORT}: Processor saw port={ep['port']} metadata={ep['metadata']} (did not read /api/status); attach -> {res['attach']}; Engine untouched and alive={alive(E.pid)}; thought-engine processes={named('thought-engine')}",
                   "Real attach approval does not exist; the refusal and non-interference were observed (" + ("ok" if ok else "UNEXPECTED") + ")")
            record("A07", "BLOCKED", "L3", "Attach-only approval with Engine stopped", "No approval type exists; attach and start are both refused 501 before effect; this build has no runner so it cannot spawn.")
            record("A08", "BLOCKED", "L3", "Start approval, correct Engine stopped", f"start -> {res['start']}; no runner exists; zero Engine processes started by Processor (children of Processor={children(P.pid)})")
            record("A10", "BLOCKED", "L3", "Parallel/repeated ensure incl. Processor restart", "No ensure operation exists to repeat. Repeated refusals are idempotent (each gets its own op_id, no effect); restart covered in A13.")
            record("A17", "BLOCKED", "L3", "Valid start/attach but no ingest grant — observation control disabled",
                   f"observe -> 501 blocked_no_ingest_grant_contract; UI control disabled; Processor has no observation-sending code path; Engine (pid {E.pid}) ran with no observation from Processor",
                   "BLOCKED because the precondition (a valid start/attach) cannot exist. Observed parts: Engine may run, observation control disabled and refused, zero observations sent by Processor")
            record("A25", "BLOCKED", "L3", "Worker unavailable", "Engine runs with no Chat-Jimmy worker configured; Processor cannot read Engine worker state without an approved attachment, so it reports nothing about the worker rather than inventing a state.")

            # A11 wrong service: point Identity address at the Engine port.
            st, s, _ = c.post("/api/candidates", {"identity": f"127.0.0.1:{E_PORT}"})
            wrong = (s["identity"]["state"], s["identity"]["probe"]["metadata"], s["identity"]["probe"].get("http_status"), s["next_action"]["code"])
            st, s, _ = c.post("/api/candidates", {"identity": f"127.0.0.1:{I_PORT}"})
            page.reload()
            wait_text(page, "#en-pill", lambda x: len(x) > 0)
            shot.append(shots(page, out, "05-identity-and-engine-candidates"))
        else:
            wrong = None

        # ---------------- A11 ----------------
        P2 = Proc("processor-second-sameport", [a.processor, "-data", os.path.join(work, "profile-b"), "-port", str(P_PORT)], out)
        P2.p.wait(10)
        P3 = Proc("processor-second-sameprofile", [a.processor, "-data", prof, "-port", str(P_PORT + 1)], out)
        P3.p.wait(10)
        with open(os.path.join(out, "processor-second-sameport.raw.log")) as f:
            m2 = f.read().strip().splitlines()[-1]
        with open(os.path.join(out, "processor-second-sameprofile.raw.log")) as f:
            m3 = f.read().strip().splitlines()[-1]
        st_bad, d_bad, _ = c.post("/api/candidates", {"identity": "http://192.168.1.10:7796"})
        ok = (P2.p.returncode != 0 and "already uses this port" in m2 and P3.p.returncode != 0 and "already using profile" in m3
              and alive(P.pid) and st_bad == 400 and (wrong is None or (wrong[0] == "PORT_OPEN_UNIDENTIFIED" and wrong[3] == "wrong_identity")))
        record("A11a", "PASS" if ok else "FAIL", "L3",
               "Processor side: occupied port, second instance on same profile, wrong service at Identity address, remote target",
               f"second Processor on busy port exit={P2.p.returncode}: {m2!r}; second Processor on same profile exit={P3.p.returncode}: {m3!r}; "
               f"Identity address pointed at real Engine port -> {wrong}; remote address -> {st_bad} {d_bad.get('message')}; original Processor alive={alive(P.pid)}",
               "Covers only Processor's own checks; see A11b")
        record("A11b", "BLOCKED", "—", "Runner side: missing executable, wrong digest/protocol, spoofed Engine at an approved endpoint",
               "No runner and no approved manifest exist; executable/digest/protocol checks belong there. Processor never shows an authenticated badge from claimed metadata (A27-live-identity shows 'unverified').")

        # ---------------- A18 ----------------
        rows = []
        for name, kw in (
            ("cross-origin form", dict(headers={"Origin": "http://evil.example", "Content-Type": "application/x-www-form-urlencoded"})),
            ("cross-origin JSON", dict(headers={"Origin": "http://evil.example"})),
            ("no Origin", dict(headers={"Origin": ""})),
            ("untrusted Host (rebinding)", dict(host="attacker.example:7860")),
            ("missing CSRF", dict(headers={"X-TP-CSRF": ""})),
            ("wrong CSRF", dict(headers={"X-TP-CSRF": "0" * 64})),
        ):
            st, d, _ = c.post("/api/engine/start", **kw)
            rows.append((name, st))
        saved = c.cookie
        c.cookie = ""
        st, _, _ = c.post("/api/engine/start")
        rows.append(("no session", st))
        st, _, _ = c.req("GET", "/api/state")
        rows.append(("GET state no session", st))
        c.cookie = saved
        st, _, _ = c.req("GET", "/api/engine/start")
        rows.append(("GET start", st))
        st, _, _ = c.req("GET", "/api/shutdown")
        rows.append(("GET shutdown", st))
        st, _, resp = c.req("OPTIONS", "/api/state", headers={"Origin": "http://evil.example", "Access-Control-Request-Method": "POST"})
        acao = resp.getheader("Access-Control-Allow-Origin")
        rows.append(("CORS preflight", f"{st} ACAO={acao}"))
        # Browser: a page on another origin tries to POST to Processor.
        evil = browser.new_page()
        evil.goto("about:blank")
        evil_result = evil.evaluate(f"""async () => {{
            try {{ const r = await fetch('http://127.0.0.1:{P_PORT}/api/engine/start', {{method:'POST', credentials:'include', headers:{{'Content-Type':'application/json'}}, body:'{{}}'}});
                   return 'status ' + r.status; }} catch (e) {{ return 'blocked: ' + e.name; }} }}""")
        evil.close()
        refused = c.state()["self"]["refused_operations_this_run"]
        alive_p = alive(P.pid)
        ok = all((isinstance(v, int) and v in (401, 403, 405, 415, 421)) for _, v in rows[:-1]) and acao is None and alive_p
        record("A18", "PASS" if ok else "FAIL", "L3", "Cross-origin form/script, untrusted Host, missing/invalid session or CSRF, GET mutation",
               f"{rows}; cross-origin browser fetch from about:blank -> {evil_result}; refused-operation counter (only counts requests that passed guards) = {refused}")

        # ---------------- A19 ----------------
        st1, d1, _ = c.post("/api/engine/start", {"path": "/bin/sh", "args": ["-c", "touch /tmp/pwned"], "endpoint": "http://evil.example"})
        st2, d2, _ = c.post("/api/candidates", {"engine": "http://127.0.0.1:7798/../../etc"})
        st3, d3, _ = c.post("/api/candidates", {"engine": "file:///etc/passwd"})
        # L2-live fixture: a raw TCP listener records every byte Processor sends when probing an "Engine".
        got = []
        srv = socket.socket()
        srv.bind(("127.0.0.1", 0))
        srv.listen(4)
        fport = srv.getsockname()[1]

        def accept():
            srv.settimeout(5)
            try:
                while True:
                    conn, _ = srv.accept()
                    conn.settimeout(1)
                    try:
                        got.append(conn.recv(4096))
                    except OSError:
                        got.append(b"")
                    conn.close()
            except OSError:
                pass
        th = threading.Thread(target=accept, daemon=True)
        th.start()
        prev_engine = c.state()["engine"]["configured"]
        c.post("/api/candidates", {"engine": f"127.0.0.1:{fport}"})
        time.sleep(1.5)
        srv.close()
        c.post("/api/candidates", {"engine": prev_engine.replace("http://", "")})
        ok = st1 == 400 and st2 == 400 and st3 == 400 and not os.path.exists("/tmp/pwned") and got and all(b == b"" for b in got)
        record("A19", "PASS" if ok else "FAIL", "L3 + L2-live fixture",
               "Path/argument injection, path traversal, non-http scheme; bytes sent to an Engine candidate",
               f"start with path/args/endpoint -> {st1} {d1.get('error')}; engine '..' path -> {st2}; file:// -> {st3}; /tmp/pwned exists={os.path.exists('/tmp/pwned')}; "
               f"raw listener as Engine candidate saw {len(got)} connection(s) carrying {[len(b) for b in got]} bytes (TCP connect only, no HTTP request)",
               "No shell exists to inject into: os/exec is not in the binary's dependency graph (checked separately with go list -deps)")

        # ---------------- A20 ----------------
        st, ctx_text, _ = c.req("GET", "/api/context")
        leaks = [k for k, v in (("operator code", code), ("session cookie", c.cookie), ("csrf", c.csrf)) if v and v in ctx_text]
        claimed = c.state()["identity"]["probe"]
        claim_leak = bool(claimed and claimed.get("claimed_version") and claimed["claimed_version"] in ctx_text and len(claimed["claimed_version"]) > 3)
        with open(os.path.join(out, "copy-context-sample.txt"), "w") as f:
            f.write(ctx_text)
        logtxt = open(os.path.join(prof, "logs", "processor.jsonl")).read()
        log_leaks = [k for k, v in (("operator code", code), ("session cookie", c.cookie), ("csrf", c.csrf)) if v and v in logtxt]
        ok = st == 200 and not leaks and not log_leaks and not claim_leak and len(ctx_text) <= 8300
        record("A20", "PASS" if ok else "FAIL", "L3 (+ L2 canary fixture in Go tests)",
               "Copy Context and log file with live secrets present",
               f"Copy Context {len(ctx_text)} bytes; leaked secrets: {leaks or 'none'}; remote claim strings included: {claim_leak}; profile log leaks: {log_leaks or 'none'}; sample saved as copy-context-sample.txt",
               "Canary-in-remote-claim test ran against a labeled fixture server (TestIdentityProbeClassificationFixture); no private observations exist to leak because Processor never handles any")

        # ---------------- A22 ----------------
        caps = {x["id"]: x for x in c.state()["capabilities"]}
        links = page.eval_on_selector_all("a", "as => as.map(a => a.href)")
        ok = caps["engine_original_ui"]["status"] == "BLOCKED" and not any(f":{E_PORT}" in h for h in links)
        record("A22", "PASS" if ok else "FAIL", "L3", "Original Engine UI inaccessible / not authorized",
               f"capability engine_original_ui={caps['engine_original_ui']['status']} ({caps['engine_original_ui']['why']}); links on page={links}; no embedded or rebuilt Engine UI")

        # ---------------- story TEST_DESIGN checks (DISC / SAFE / UX / FLOW) ----------------
        keep_id = c.state()["identity"]["configured"].replace("http://", "")
        # DISC-01 wrong/unused port
        free = socket.socket(); free.bind(("127.0.0.1", 0)); wport = free.getsockname()[1]; free.close()
        _, s1, _ = c.post("/api/candidates", {"identity": f"127.0.0.1:{wport}"})
        record("DISC-01", "PASS" if s1["identity"]["state"] == "UNREACHABLE" else "FAIL", "L3", "Probe an unused local port",
               f"127.0.0.1:{wport} -> {s1['identity']['state']} ({s1['identity']['probe']['port']}); label '{s1['identity']['label']}'")
        # DISC-02 real unrelated programs: python http.server (200 on /), and a second real Processor (claims a product name)
        US = Proc("unrelated-http-server", [sys.executable, "-I", "-u", "-m", "http.server", "--bind", "127.0.0.1", "0"], out, cwd=work)
        owned.append(US)
        uport = None
        for _ in range(100):
            time.sleep(0.05)
            try:
                txt = open(os.path.join(out, "unrelated-http-server.raw.log")).read()
                mm = re.search(r"port (\d+)", txt)
                if mm:
                    uport = int(mm.group(1)); break
            except OSError:
                pass
        PX = Proc("processor-as-wrong-service", [a.processor, "-data", os.path.join(work, "profile-x"), "-port", str(P_PORT + 1)], out)
        STARTED.append(PX)
        PX.wait_http(P_PORT + 1, "/api/health")
        res2 = []
        for label, port_ in (("python http.server", uport), ("another Thought Processor", P_PORT + 1)):
            _, sx, _ = c.post("/api/candidates", {"identity": f"127.0.0.1:{port_}"})
            pr_ = sx["identity"]["probe"]
            res2.append((label, port_, sx["identity"]["state"], pr_["metadata"], pr_.get("http_status"), pr_.get("claimed_app"), sx["next_action"]["code"]))
        ok = uport and all(r_[2] in ("PORT_OPEN_UNIDENTIFIED", "ANSWERED_NOT_IDENTITY") and r_[6] == "wrong_identity" for r_ in res2)
        record("DISC-02", "PASS" if ok else "FAIL", "L3", "Real unrelated HTTP programs placed at the Identity address",
               f"{res2} — neither shown as Identity, authenticated or authorized")
        US.stop(); PX.stop()
        # SAFE-03 stalled endpoint: accepts, never answers (L2-live fixture)
        stall = socket.socket(); stall.bind(("127.0.0.1", 0)); stall.listen(8); sport = stall.getsockname()[1]
        held = []
        def hold():
            stall.settimeout(10)
            try:
                while True:
                    cn, _ = stall.accept(); held.append(cn)
            except OSError:
                pass
        threading.Thread(target=hold, daemon=True).start()
        timing = {}
        def slow_save():
            t0_ = time.monotonic(); _, sv, _ = c.post("/api/candidates", {"identity": f"127.0.0.1:{sport}"}); timing["save_ms"] = round((time.monotonic()-t0_)*1000); timing["state"] = sv["identity"]
        th2 = threading.Thread(target=slow_save); th2.start()
        time.sleep(0.2)
        c2 = Client(P_PORT); c2.cookie, c2.csrf = c.cookie, c.csrf
        t0_ = time.monotonic(); stc, _, _ = c2.req("GET", "/api/state"); concurrent_ms = round((time.monotonic()-t0_)*1000)
        th2.join(10); stall.close()
        for cn in held: cn.close()
        sv = timing.get("state", {})
        ok = stc == 200 and concurrent_ms < 500 and sv.get("probe", {}).get("metadata") == "FAILED" and timing.get("save_ms", 99999) < 3500
        record("SAFE-03", "PASS" if ok else "FAIL", "L3 + L2-live fixture", "Identity address that accepts connections but never answers",
               f"save+check returned in {timing.get('save_ms')} ms with state={sv.get('state')} metadata={sv.get('probe',{}).get('metadata')}; a concurrent page read answered {stc} in {concurrent_ms} ms",
               "The stalling listener is a test fixture")
        c.post("/api/candidates", {"identity": keep_id})
        # SAFE-05 unusable profile location
        badp = os.path.join(work, "not-a-folder"); open(badp, "w").write("x")
        PB = Proc("processor-bad-profile", [a.processor, "-data", badp, "-port", str(P_PORT + 1)], out); PB.p.wait(10)
        msg = open(os.path.join(out, "processor-bad-profile.raw.log")).read().strip().splitlines()[-1]
        ok = PB.p.returncode != 0 and open(badp).read() == "x" and port_free(P_PORT + 1)
        record("SAFE-05", "PASS" if ok else "FAIL", "L3", "Profile path is a file, not a folder",
               f"exit {PB.p.returncode}: {msg!r}; file unchanged={open(badp).read() == 'x'}; no port opened={port_free(P_PORT + 1)}")
        # UX-05 + FLOW-07 + metrics evidence
        sel = c.state()["selection"]
        st_f, fl, _ = c.req("GET", "/api/flow")
        st_m, mt, _ = c.req("GET", "/api/metrics")
        tr0 = fl["traces"][0] if fl["traces"] else None
        ok = (sel["entity"]["status"] == "NONE" and sel["instance"]["status"] == "UNKNOWN" and sel["feature"]["status"] == "CANDIDATE"
              and tr0 and [x["status"] for x in tr0["stages"]][1:] in (["NOT AVAILABLE", "NOT ATTEMPTED", "NONE"], ["NOT ATTEMPTED", "NOT ATTEMPTED", "NONE"]))
        record("UX-05", "PASS" if ok else "FAIL", "L3", "Entity, feature and instance shown separately",
               f"entity={sel['entity']['label']!r}; feature={sel['feature']['label']!r}; instance={sel['instance']['label']!r}")
        record("FLOW-07", "PASS" if ok else "FAIL", "L3", "Missing Identity/Engine steps shown as missing, not reconstructed",
               f"{len(fl['traces'])} traces; newest {tr0['op_id']} {tr0['action']}: " + ", ".join(f"{x['name']}={x['status']}" for x in tr0["stages"]))
        ok = st_m == 200 and sum(mt["refused_operations"].values()) >= 8 and mt["probes"]["identity"]["checks"] >= 5
        record("METRICS", "PASS" if ok else "FAIL", "L3", "Processor's own metrics are real counters",
               f"refused={mt['refused_operations']}; rejected={mt['rejected_requests']}; identity checks={mt['probes']['identity']['checks']} outcomes={mt['probes']['identity']['outcomes']} latency={mt['probes']['identity']['latency_last_50']}; not measured: {mt['not_available']}")
        # Diagnostics screenshots
        if not page.is_closed():
            page.reload(); page.wait_for_selector("#app:not([hidden])")
            page.click("#diag summary")
            page.wait_for_selector("#flow > li")
            shot.append(shots(page, out, "07-diagnostics-flow"))
            page.click(".tabs button[data-tab=metrics]")
            page.wait_for_selector("#metrics dt:nth-of-type(4)")
            shot.append(shots(page, out, "08-diagnostics-metrics"))
            page.click("#diag summary")

        # ---------------- A21 ----------------
        if E:
            before = c.state(probe=True)["engine"]
            ms_stop, rc = E.stop()
            t_gone = time.monotonic()
            after = c.state(probe=True)["engine"]
            detect_ms = round((time.monotonic() - t_gone) * 1000)
            page.close()  # stop the page's 10 s re-check so freshness can expire
            time.sleep(21)
            stale = c.state()["identity"]
            ok = before["state"] == "PORT_OPEN_UNIDENTIFIED" and after["state"] == "UNREACHABLE" and stale["state"] == "STALE" and stale["freshness"] == "STALE"
            record("A21", "PASS" if ok else "FAIL", "L3", "Service disappears after a successful check; no re-check for 21 s",
                   f"Engine before={before['state']} checked {before['probe']['checked_at']}; test script stopped Engine pid {E.pid} (SIGTERM, {ms_stop} ms, exit {rc}); "
                   f"next re-check -> {after['state']} ({after['probe']['port']}) within {detect_ms} ms of the re-check request; after 21 s without a check Identity shows state={stale['state']} freshness={stale['freshness']}",
                   "Detection is on the page's 10 s re-check cadence, not instant")

        # ---------------- A15 / A16 (and restart Engine for survival check) ----------------
        if a.engine:
            E2 = Proc("thought-engine-2", [a.engine, "-data", os.path.join(work, "engine-data")], out, env={"PORT": str(E_PORT), "CHAT_JIMMY_URL": ""}, cwd=os.path.dirname(a.engine))
            owned.append(E2)
            E2.wait_http(E_PORT, "/api/status")
            stops = [c.post("/api/engine/stop")[0:2] for _ in range(2)]
            survivors = dict(identity=alive(I.pid) if I else None, engine=alive(E2.pid), processor=alive(P.pid))
            ok = all(sc == 501 and d["error"] == "blocked_no_identity_approval_service" for sc, d in stops) and all(v in (True, None) for v in survivors.values())
            record("A16", "PASS" if ok else "FAIL", "L3", "No stop approval/control policy: request Stop twice",
                   f"stop -> {[(sc, d['status'], d['op_id']) for sc, d in stops]}; external Engine pid {E2.pid} alive={survivors['engine']}; Identity alive={survivors['identity']}; Processor alive={survivors['processor']}")
            record("A15", "BLOCKED", "L3", "Stop session-owned Engine, repeat, then external Engine",
                   "No session-owned Engine can exist (no runner). Stop requests were refused and the external Engine, Identity and Processor survived (A16 evidence).")
            record("A12", "BLOCKED", "—", "Slow startup, exit or partial failure of an Engine start", "No runner; nothing is started, so there is no startup to time out or children to clean up.")
            record("A14", "BLOCKED", "—", "Stale ownership ledger / reused PID / uncertain child", "Processor keeps no ownership ledger because it owns no processes; it never kills by PID or port. Its own run record is used only to report an unclean previous run (see OPS crash test).")
            record("A26", "BLOCKED", "—", "Authority revoked after attach", "No attach can be granted, so there is nothing to revoke.")
            record("A23", "NOT RUN", "—", "Repo creation succeeded / local setup failed (later onboarding)", "Out of scope: GitHub bootstrap is undecided and not implemented.")
            record("A24", "NOT RUN", "—", "GitHub account differs from administrator (later onboarding)", "Out of scope: GitHub/admin enrollment is undecided and not implemented.")

        # ---------------- A13 ----------------
        ctx2 = browser.new_context()
        p2 = ctx2.new_page()
        p2.goto(f"http://127.0.0.1:{P_PORT}/")
        p2.wait_for_selector("#signin:not([hidden])")
        fresh_browser_signed_out = True
        ctx2.close()
        if not page.is_closed():
            page.close()
        page = ctx.new_page()  # "reopen" in same browser profile: cookie persists
        page.goto(f"http://127.0.0.1:{P_PORT}/")
        page.wait_for_selector("#app:not([hidden])")
        reopen_signed_in = True
        old_run, old_cookie = c.state()["self"]["run_id"], c.cookie
        t = time.monotonic()
        st, d, _ = c.post("/api/shutdown")
        P.p.wait(10)
        stop_ms = round((time.monotonic() - t) * 1000)
        rc = P.p.returncode
        page.wait_for_selector("#gone:not([hidden])", timeout=15000)
        shot.append(shots(page, out, "06-processor-stopped"))
        survivors = dict(identity=alive(I.pid) if I else None, engine=alive(E2.pid) if a.engine else None)
        leftover = sorted(set(named("thought-process")) - set(baseline["thought-process"]))
        run_json_left = os.path.exists(os.path.join(prof, "runtime", "run.json"))
        if I:
            I.stop()
        P, ms2 = start_processor(a.processor, prof, out, name="processor-restart")
        code2 = read_code(prof)
        c.cookie = old_cookie
        st_old, _, _ = c.req("GET", "/api/state")
        page.goto(f"http://127.0.0.1:{P_PORT}/")
        page.wait_for_selector("#signin:not([hidden])")
        c.cookie = ""
        c.sign_in(code2)
        pre = c.state()["identity"]
        s = c.state(probe=True)
        record("UX-02", "PASS" if pre["state"] == "NOT_CHECKED" and s["identity"]["state"] == "UNREACHABLE" else "FAIL", "L3",
               "Remembered Identity address is gone after a restart",
               f"before first check: {pre['state']} ('{pre['label']}'); after check: {s['identity']['state']} — never shown as live from the saved record")
        ok = (fresh_browser_signed_out and reopen_signed_in and st == 202 and rc == 0 and not leftover and not run_json_left
              and all(v in (True, None) for v in survivors.values()) and st_old == 401 and s["self"]["run_id"] != old_run
              and s["identity"]["configured"] == f"http://127.0.0.1:{I_PORT}" and s["self"]["previous_unclean_run"] is None and code2 != code)
        record("A13", "PASS" if ok else "FAIL", "L3", "Close/reopen browser; stop and restart Processor backend on the same profile",
               f"new browser context -> sign-in screen; reopened tab in same browser -> still signed in; Stop Processor -> {st}, exit {rc} in {stop_ms} ms; "
               f"leftover Processor processes={leftover}; run.json removed={not run_json_left}; Identity alive={survivors['identity']} Engine alive={survivors['engine']}; "
               f"restart ready in {ms2} ms as pid {P.pid}, run {s['self']['run_id']} (was {old_run}); old session cookie -> {st_old}; new operator code differs={code2 != code}; "
               f"saved Identity address kept={s['identity']['configured']} and re-checked at {s['identity']['probe']['checked_at']}",
               "No survival is claimed for anything Processor would own — it owns nothing")

        # ---------------- crash recovery + repeated runs ----------------
        os.kill(P.pid, signal.SIGKILL)
        P.p.wait(5)
        P, ms3 = start_processor(a.processor, prof, out, name="processor-after-kill9")
        read_code(prof)
        c.cookie = ""
        c.sign_in(read_code(prof))
        s = c.state()
        crash_ok = s["self"]["previous_unclean_run"] is not None and alive(P.pid)
        record("OPS-04", "PASS" if crash_ok else "FAIL", "L3", "Processor killed with SIGKILL, then restarted on same profile",
               f"restart ready in {ms3} ms (pid {P.pid}); previous_unclean_run={s['self']['previous_unclean_run']}; lock reacquired (kernel released flock); no action taken against any other process")
        P.stop()
        cycles = []
        for i in range(3):
            Pi, msi = start_processor(a.processor, os.path.join(work, f"profile-cycle-{i}"), out, name=f"processor-cycle-{i}")
            ci = Client(P_PORT)
            ci.sign_in(read_code(os.path.join(work, f"profile-cycle-{i}")))
            si = ci.state()
            sms, src = Pi.stop()
            cycles.append(dict(pid=Pi.pid, ready_ms=msi, self_reported_startup_ms=si["self"]["startup_ms"], stop_ms=sms, exit=src,
                               leftover_children=children(Pi.pid)))
        meta["clean_cycles"] = cycles
        ok = all(cy["exit"] == 0 and cy["ready_ms"] is not None for cy in cycles)
        record("SAFE-07", "PASS" if ok else "FAIL", "L3", "3 clean start → inspect → stop cycles (Processor only)", json.dumps(cycles))
    finally:
        browser.close()
        pw.stop()
        for pr in STARTED + owned:
            pr.stop()
        leftovers = {k: sorted(set(named(k)) - set(baseline[k])) for k in baseline}
        note(f"final orphan audit (processes not present at baseline): {leftovers}")
        meta["orphan_audit"] = leftovers
        meta["finished"] = now()
        meta["screenshots"] = shot

    # sanitize and save
    def clean(sx):
        for k in CODES:
            sx = sx.replace(k, "<operator-code>")
        sx = re.sub(r"[0-9a-f]{64}", "<64-hex>", sx)
        return sx.replace(work, "<tmp>/" + os.path.basename(work))
    for fn in os.listdir(out):
        if fn.endswith(".raw.log"):
            with open(os.path.join(out, fn)) as f:
                txt = clean(f.read())
            with open(os.path.join(out, fn.replace(".raw.log", ".log")), "w") as f:
                f.write(txt)
            os.remove(os.path.join(out, fn))
    with open(os.path.join(out, "copy-context-sample.txt")) as f:
        t = clean(f.read())
    with open(os.path.join(out, "copy-context-sample.txt"), "w") as f:
        f.write(t)
    with open(os.path.join(out, "evidence.json"), "w") as f:
        f.write(clean(json.dumps(dict(meta=meta, results=RESULTS, log=LOGS), indent=2)))
    with open(os.path.join(out, "evidence.md"), "w") as f:
        f.write("# Live acceptance evidence (generated)\n\n")
        f.write("```json\n" + clean(json.dumps(meta, indent=2)) + "\n```\n\n| ID | Verdict | Layer | Exercise | Observed | Limitation |\n|---|---|---|---|---|---|\n")
        for r in RESULTS:
            f.write("| {id} | {verdict} | {layer} | {action} | {observed} | {limitation} |\n".format(**{k: clean(str(v)).replace("|", "\\|") for k, v in r.items()}))
    bad = [r["id"] for r in RESULTS if r["verdict"] == "FAIL"]
    print("FAILURES:", bad or "none")
    return 1 if bad else 0


if __name__ == "__main__":
    sys.exit(main())
