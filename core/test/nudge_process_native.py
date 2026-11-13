#!/usr/bin/env python3
"""Stage 2 (design rev 4): ONE owner-approved nudge to TASS himself, end to end with the REAL Go services.

Real Identity (bound single-use process/stop decisions), real Thought Engine (durable nudge runs), real
Core, the TEST FIXTURE model (NOT Nemotron), no Docker, no GitHub, a throwaway APP_HOME. It covers:

  A  no process grant -> 403, nothing submitted, zero nudge model requests
  B  owner enables cognition.nudge-process -> Process once -> 202 -> COMPLETED, verified:
     exactly ONE nudge model request, the prompt carries the owner's attention text, ONE note and ONE
     "developed" step with source_nudge_id + the request ID; the same Process again returns the same run
  C  admission: a slow nudge holds the one active run; a second nudge is 409 busy; same-ID repeats
     return the same run; Engine killed -9 mid-call -> after restart UNCERTAIN, never re-sent
  D  Stop: an unsubmitted draft is voided (bound stop decision); Process on it is then refused
  E  Identity LOCKED -> Process refused; revoke the grant -> Process refused (403)
  F  restore fence written by tass.sh's own function -> older drafts PRE_RESTORE, never processed;
     a new draft afterwards still processes
  G  Core restart: receipts are read only; no request is ever re-sent

    APP_HOME=/tmp/e2e/s2 BIN=/tmp/e2e/bin SEED=/tmp/e2e/seed \\
    CORE_BIN=tass-core-s2 ENGINE_BIN=thought-engine-s2 IDENTITY_BIN=tass-identity-s2 \\
    [S2_PORT_BASE=39100] python3 core/test/nudge_process_native.py

ISOLATION (Codex audit PR #105 / GPT-6 handoff): this test must never reach the owner's running TASS.
  - Every service uses its own port from S2_PORT_BASE (default 39100): Core +0, Identity +1, Engine +2,
    fault proxy +3, fixture model +4. The owner's ports (8795/8797 Core, 8796 Identity, 7798 Engine,
    8081/18083 model, 7790/7860/7791) are refused outright, as is any base that overlaps them.
  - Preflight, BEFORE any request or process start: every test port must be free to bind (else abort),
    and APP_HOME must be new or a folder this test made before (marker file), never the owner's.
  - Services get a CLEAN environment (no inherited GITHUB_TOKEN, CHAT_JIMMY_URL, THOUGHT_ENGINE_*,
    IDENTITY_*, TASS_* ...) and HOME inside APP_HOME, and every address is passed explicitly.
  - Sentinels: each owner port that is FREE here is bound by a listener that counts connections; the
    run fails if any test component touched one. (A port in use, e.g. the owner's live Identity, can't
    be watched, but no test address points at it.)
"""
import glob, json, os, signal, subprocess, sys, threading, time

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
os.environ.setdefault("CAND", "x"); os.environ.setdefault("ALPHA", "x")
os.environ.setdefault("FIXTURE_DELAY", "8")
import socket  # noqa: E402

OWNER_PORTS = (8795, 8796, 8797, 7798, 7799, 8081, 18083, 7790, 7860, 7791)  # 8797: owner's second Core port (#100)
BASE = int(os.environ.get("S2_PORT_BASE", "39100"))
P_CORE, P_IDN, P_ENG, P_PROXY, P_MODEL = (BASE + i for i in range(5))
TEST_PORTS = (P_CORE, P_IDN, P_ENG, P_PROXY, P_MODEL)


def preflight():
    """Abort before anything starts or any request is made, unless the run is provably isolated."""
    bad = [p for p in TEST_PORTS if p in OWNER_PORTS or not 1024 < p < 65536]
    if bad:
        sys.exit(f"ABORT: S2_PORT_BASE={BASE} uses owner/invalid port(s) {bad}; pick another base. Nothing was started.")
    for p in TEST_PORTS:
        sock = socket.socket()
        sock.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)  # TIME_WAIT is not "in use"; a listener is
        try:
            sock.bind(("127.0.0.1", p))
        except OSError as err:
            sys.exit(f"ABORT: test port {p} is already in use ({err}); something else is running there. Nothing was started or contacted.")
        finally:
            sock.close()
    app = os.environ["APP_HOME"]
    if os.path.exists(app) and os.listdir(app) and not os.path.exists(os.path.join(app, ".s2-native-test")):
        sys.exit(f"ABORT: APP_HOME={app} exists and was not made by this test (no .s2-native-test marker); "
                 "use a new throwaway folder. Nothing was started.")
    for name in ("TASS_DATA", "IDENTITY_DATA"):
        if os.environ.get(name):
            print(f"note: ignoring inherited {name}; services get a clean environment", flush=True)


preflight()
os.environ["TASS_TEST_ISOLATED_PORTS"] = "1"  # tells the shared fixed-port guard (lesson_retry_native) this run is isolated
os.makedirs(os.environ["APP_HOME"], exist_ok=True)
open(os.path.join(os.environ["APP_HOME"], ".s2-native-test"), "w").write("throwaway Stage 2 native test folder\n")

import acceptance  # noqa: E402
import lesson_retry_native as n  # noqa: E402
from acceptance import check, core_post, http, idn_post, log, results, token, wait  # noqa: E402

CORE = acceptance.CORE = f"http://127.0.0.1:{P_CORE}"   # core_post / idn_post read these at call time
IDN = acceptance.IDN = f"http://127.0.0.1:{P_IDN}"
APP, BIN = n.APP, n.BIN
os.makedirs(f"{APP}/home", exist_ok=True)


def clean_env(extra):
    """Only what the services need: no inherited tokens, URLs or data folders from the caller's shell."""
    return {"PATH": os.environ.get("PATH", "/usr/bin:/bin"), "HOME": f"{APP}/home", "LANG": "C.UTF-8",
            "TMPDIR": f"{APP}/home", **extra}


def start(name, cmd, env, health_url, cwd=None):
    out = open(f"{APP}/logs/{name}.log", "a")
    n.procs[name] = subprocess.Popen(cmd, env=clean_env(env), stdout=out, stderr=subprocess.STDOUT, cwd=cwd)
    ok = wait(name, lambda: n.up(health_url), 40, 1)
    log(f"started {name}: {cmd[0]} healthy={bool(ok)}")
    if not ok:
        raise SystemExit(f"{name} did not come up on its isolated port; see {APP}/logs/{name}.log")


SENTINEL_HITS = []


def start_sentinels():
    """Listen on every owner port that is free here; any connection from this run is a FAIL."""
    watched = []
    for p in OWNER_PORTS:
        sock = socket.socket()
        sock.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
        try:
            sock.bind(("127.0.0.1", p))
        except OSError:
            sock.close()
            continue
        sock.listen(8)

        def serve(s=sock, port=p):
            while True:
                try:
                    c, _ = s.accept()
                except OSError:
                    return
                SENTINEL_HITS.append(port)
                c.close()
        threading.Thread(target=serve, daemon=True).start()
        watched.append(p)
    return watched
CORE_BIN = os.environ.get("CORE_BIN", "tass-core-s2")
ENGINE_BIN = os.environ.get("ENGINE_BIN", "thought-engine-s2")
IDENTITY_BIN = os.environ.get("IDENTITY_BIN", "tass-identity-s2")
MODEL = f"http://127.0.0.1:{P_MODEL}"
ENG = f"http://127.0.0.1:{P_ENG}"
HERE = os.path.dirname(os.path.abspath(__file__))


def get(path):
    return http("GET", CORE + path, headers={"X-TASS-Token": token(CORE)})


def fixture():
    return http("GET", MODEL + "/fixture/requests")[1]


def ekey():
    return open(f"{APP}/data/engine/api-key").read().strip()


# Fault proxy between Core and the Engine (test only): passes every request through, except that while
# DROP_NUDGE is set it answers a nudge POST with a 502 WITHOUT forwarding it (a lost delivery).
PROXY = f"http://127.0.0.1:{P_PROXY}"
DROP_NUDGE = threading.Event()
PROXIED = []


def start_proxy():
    import http.client
    from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

    class P(BaseHTTPRequestHandler):
        def log_message(self, *a):
            pass

        def go(self):
            body = self.rfile.read(int(self.headers.get("Content-Length", "0") or 0))
            if self.command == "POST" and "/experiments/" in self.path:  # Stage 3B: every experiment write Core sent
                PROXIED.append(self.path)
            if self.command == "POST" and self.path.endswith("/nudge"):
                PROXIED.append(self.path)
                if DROP_NUDGE.is_set():
                    self.send_response(502); self.end_headers(); return
            try:
                c = http.client.HTTPConnection("127.0.0.1", P_ENG, timeout=600)
                c.request(self.command, self.path, body or None, {k: v for k, v in self.headers.items() if k.lower() not in ("host", "connection")})
                r = c.getresponse(); data = r.read()
                self.send_response(r.status)
                for k, v in r.getheaders():
                    if k.lower() not in ("transfer-encoding", "connection", "content-length"):
                        self.send_header(k, v)
                self.send_header("Content-Length", str(len(data))); self.end_headers(); self.wfile.write(data)
            except Exception:
                self.send_response(502); self.send_header("Content-Length", "0"); self.end_headers()

        do_GET = do_POST = go

    srv = ThreadingHTTPServer(("127.0.0.1", P_PROXY), P)
    threading.Thread(target=srv.serve_forever, daemon=True).start()


def start_core():
    start("core", [f"{BIN}/{CORE_BIN}"], {
        "TASS_DATA": f"{APP}/data/core", "TASS_LISTEN": f"127.0.0.1:{P_CORE}",
        "TASS_ENGINE_URL": PROXY, "TASS_ENGINE_KEY": f"{APP}/data/engine/api-key",
        "TASS_IDENTITY_URL": IDN, "TASS_IDENTITY_KEY_FILE": f"{APP}/secrets/core.key",
        "TASS_REPO": "", "TASS_TOKEN_FILE": f"{APP}/secrets/github_token",
        "TASS_GITHUB_API": f"http://127.0.0.1:{P_MODEL}/no-github"}, CORE + "/api/health")


def start_engine():
    start("engine", [f"{BIN}/{ENGINE_BIN}", "-data", f"{APP}/data/engine", "-model", MODEL, "-worker", "nemotron",
          "-identity-url", IDN, "-identity-key-file", f"{APP}/secrets/engine.key", "-pulse", "20s"],
          {"PORT": str(P_ENG)}, ENG + "/api/model", cwd=os.environ.get("ENGINE_CWD") or None)


def kill9(name):
    p = n.procs.pop(name)
    p.send_signal(signal.SIGKILL)
    p.wait(10)


def nudges(tid=""):
    return get("/api/nudges" + ("?thought_id=" + tid if tid else ""))[1]


def view(nid):
    lst = nudges()
    for x in (lst.get("nudges") if isinstance(lst, dict) else None) or []:
        if x["nudge_id"] == nid:
            return x
    return {}


def run_state(nid):
    return ((view(nid).get("run")) or {}).get("state", "")


def draft(tid, action, key):
    code, d = core_post("/api/nudges", {"source_thought_id": tid, "recipient_id": "self", "reason_code": "owner_request",
                                        "requested_action": action, "idempotency_key": key})
    return code, d


def process(d):
    return core_post(f"/api/nudges/{d['nudge_id']}/process", {"body_digest": d.get("body_digest", "")})


def run_dir(nid):
    hits = glob.glob(f"{APP}/data/engine/**/nudge-runs/{nid}", recursive=True)
    return hits[0] if hits else ""


def thought_detail(eid, tid):
    code, t = http("GET", f"{ENG}/api/entities/{eid}/thoughts/{tid}", headers={"Authorization": "Bearer " + ekey()})
    return code, (t.get("thought") if isinstance(t, dict) and "thought" in t else t)


def main():
    n.prepare()
    watched = start_sentinels()
    if watched:  # positive control: a sentinel really counts a connection
        socket.create_connection(("127.0.0.1", watched[0]), timeout=2).close()
        time.sleep(0.2)
        check("ISOLATION: sentinel positive control counts a connection", SENTINEL_HITS == [watched[0]], str(SENTINEL_HITS))
        SENTINEL_HITS.clear()
    log(f"isolated ports: core {P_CORE}, identity {P_IDN}, engine {P_ENG}, proxy {P_PROXY}, model {P_MODEL}; "
        f"sentinels on free owner ports {watched}")
    start("model", ["python3", f"{HERE}/fixture-model/fake_llama.py", str(P_MODEL)], {"FIXTURE_DELAY": os.environ["FIXTURE_DELAY"]}, MODEL + "/health")
    start("identity", [f"{BIN}/{IDENTITY_BIN}"], {"IDENTITY_DATA": f"{APP}/data/identity", "IDENTITY_LISTEN": f"127.0.0.1:{P_IDN}",
          "IDENTITY_SERVICE_KEYS": f"{APP}/secrets/service-keys", "IDENTITY_CORE_URL": CORE + "/"}, IDN + "/v1/health")
    start_engine()
    start_proxy()
    start_core()
    try:
        run()
    finally:
        for name in list(n.procs):
            n.stop(name)
    check("ISOLATION: no connection to any watched owner port during the run", not SENTINEL_HITS, str(sorted(set(SENTINEL_HITS))))
    print()
    for name, res, detail in results:
        print(f"{res:4}  {name}  {detail}")
    fails = [r for r in results if r[1] not in ("PASS", "NOT RUN")]
    notrun = [r for r in results if r[1] == "NOT RUN"]
    print(f"\n{len(results) - len(fails) - len(notrun)}/{len(results)} PASS, {len(fails)} FAIL, {len(notrun)} NOT RUN (TEST FIXTURE model; not Nemotron, not the owner's machine)")
    sys.exit(1 if fails else 0)


def run():
    code, r = idn_post("/v1/entity", {"name": "Mira", "passphrase": n.PASS})
    check("real Identity: entity created UNLOCKED", code == 201 and r.get("state") == "UNLOCKED", str(code))
    grants = (r.get("entity") or {}).get("grants") or []
    check("cognition.nudge.process is NOT a default grant", "cognition.nudge.process" not in grants, ",".join(grants))

    code, r = core_post("/api/say", {"text": "Hello Mira, what are you thinking about?"})
    check("conversation accepted", code in (200, 202), str(code))
    lib = wait("Engine entity candidate", lambda: (get("/api/cognition/library")[1].get("link") or {}).get("candidates"), 90, 2)
    check("Cognition lists the Engine entity", bool(lib), "")
    if not lib:
        return
    eid = lib[0]["engine_entity_id"]
    code, _ = core_post("/api/cognition/link", {"engine_entity_id": eid})
    check("owner links Cognition to the Engine entity", code == 200, str(code))
    for i in range(3):  # test setup: three related observations form one persistent Thought
        http("POST", f"{ENG}/api/entities/{eid}/events", {"event_id": f"s2-obs-{i}", "source": "test",
             "text": f"The owner worked on the local project again (observation {i})"}, {"Authorization": "Bearer " + ekey()})
    thoughts = wait("a real persistent Thought", lambda: get("/api/cognition/library")[1].get("thoughts"), 120, 2)
    check("a real persistent Thought exists", bool(thoughts), "")
    if not thoughts:
        return
    tid = thoughts[0]["source_id"]
    code, _ = http("POST", f"{ENG}/api/entities/{eid}/grant", {"enabled": True}, {"Authorization": "Bearer " + ekey()})
    check("setup: owner turns Thought Processing ON in the Engine", code == 200, str(code))
    code, r = idn_post("/v1/grants/skill", {"skill": "cognition.nudge-draft", "passphrase": n.PASS})
    check("setup: owner enables cognition.nudge-draft", code == 200, str(code))
    time.sleep(3)

    # ---- A: no process grant
    code, d1 = draft(tid, "Pay attention to whether the local project keeps coming back.", "s2-key-0001")
    check("draft saved (201)", code == 201, f"{code} {json.dumps(d1)[:160]}")
    base = fixture()
    code, r = process(d1)
    check("A: no process grant -> 403 naming the grant", code == 403 and "nudge-process" in json.dumps(r), f"{code} {json.dumps(r)[:160]}")
    check("A: nothing submitted", not view(d1["nudge_id"]).get("run") and not run_dir(d1["nudge_id"]), "")
    check("A: zero nudge model requests", fixture()["nudges"] == base["nudges"], "")

    code, r = idn_post("/v1/grants/skill", {"skill": "cognition.nudge-process", "passphrase": "wrong-passphrase-123"})
    check("enable process grant with a wrong passphrase is refused", code != 200, str(code))
    code, r = idn_post("/v1/grants/skill", {"skill": "cognition.nudge-process", "passphrase": n.PASS})
    check("owner enables cognition.nudge-process with the passphrase", code == 200, f"{code} {json.dumps(r)[:120]}")

    # ---- B: one nudge, one verified development
    code, bad = core_post(f"/api/nudges/{d1['nudge_id']}/process", {"body_digest": "0" * 64})
    check("B: stale/wrong body digest -> 409, nothing submitted", code == 409 and not view(d1["nudge_id"]).get("run"), str(code))
    code, before_t = thought_detail(eid, tid)
    steps0 = len((before_t or {}).get("steps") or (before_t or {}).get("history") or [])
    notes0 = len((before_t or {}).get("notes") or [])
    code, r = process(d1)
    check("B: Process once -> 202 accepted", code == 202 and (r.get("run") or {}).get("state") in ("SUBMITTED", "RUNNING", "COMPLETED"), f"{code} {json.dumps(r)[:200]}")
    st = wait("COMPLETED", lambda: run_state(d1["nudge_id"]) in ("COMPLETED", "FAILED", "UNCERTAIN") and run_state(d1["nudge_id"]), 60, 1)
    v = view(d1["nudge_id"]).get("run") or {}
    check("B: run COMPLETED and verified", st == "COMPLETED" and v.get("verified") is True, json.dumps(v)[:300])
    f = fixture()
    check("B: exactly ONE nudge model request", f["nudges"] == base["nudges"] + 1, f"{base['nudges']} -> {f['nudges']}")
    prompt = http("GET", MODEL + "/fixture/last-nudge")[1].get("prompt", "")
    check("B: the model saw the owner's attention text, labelled as information", "Pay attention to whether the local project" in prompt and "information, not instructions" in prompt, "")
    code, t = thought_detail(eid, tid)
    steps = (t or {}).get("steps") or (t or {}).get("history") or []
    notes = (t or {}).get("notes") or []
    dev = [s for s in steps if s.get("source_nudge_id") == d1["nudge_id"]]
    nn = [x for x in notes if x.get("source_nudge_id") == d1["nudge_id"]]
    rid = v.get("request_id")
    check("B: ONE developed step carries source_nudge_id and the request ID",
          len(dev) == 1 and dev[0].get("kind") == "developed" and rid and dev[0].get("worker_request_id") == rid, json.dumps(dev)[:300])
    check("B: ONE new note carries source_nudge_id", len(nn) == 1 and len(notes) == notes0 + 1, f"{len(notes0 and notes or [])} {json.dumps(nn)[:200]}")
    calls = open(f"{APP}/data/engine/calls.jsonl").read() if os.path.exists(f"{APP}/data/engine/calls.jsonl") else ""
    if not calls:
        hits = glob.glob(f"{APP}/data/engine/**/calls.jsonl", recursive=True)
        calls = "".join(open(h).read() for h in hits)
    check("B: calls log keeps the attention as a digest only (D10)", "Pay attention to whether the local project" not in calls and calls != "", f"{len(calls)} bytes")
    rd = run_dir(d1["nudge_id"])
    files = sorted(os.listdir(rd)) if rd else []
    check("B: Engine run folder holds claim/http/response/outcome", files == ["claim.json", "http.json", "outcome.json", "response.json"], ",".join(files))
    code, r = process(d1)
    check("B: Process again returns the SAME run (200), no new request", code == 200 and (r.get("run") or {}).get("state") == "COMPLETED"
          and fixture()["nudges"] == f["nudges"], str(code))

    # ---- P0 (#106): Cognition shows what was produced (real binaries; GET only)
    import hashlib
    tfiles = glob.glob(f"{APP}/data/engine/**/thoughts/{tid}.json", recursive=True)
    tsha = hashlib.sha256(open(tfiles[0], "rb").read()).hexdigest() if tfiles else ""
    m0, n0 = fixture()["requests"], fixture()["nudges"]
    code, tv = get(f"/api/cognition/thoughts/{tid}")
    devs = (tv.get("developments") or []) if isinstance(tv, dict) else []
    dmine = [x for x in devs if x.get("source_nudge_id") == d1["nudge_id"]]
    dv = dmine[0] if dmine else {}
    check("P0-A1: Thought detail shows the Note TASS wrote (full text), its author and time",
          code == 200 and "(TEST FIXTURE) Following the owner's pointer" in ((dv.get("note") or {}).get("text") or "")
          and (dv.get("note") or {}).get("by") and (dv.get("note") or {}).get("at"), json.dumps(dv)[:300])
    check("P0-A2: the development carries the SAME request ID as the verified receipt, and Core's receipt verified it",
          dv.get("request_id") == v.get("request_id") and dv.get("persistence") == "NOTE_AND_STEP_CORRELATED"
          and dv.get("nudge_receipt") == "COMPLETED_VERIFIED", json.dumps({k: dv.get(k) for k in ("request_id", "persistence", "nudge_receipt")}))
    check("P0: suggested state shown as not applied; meaning separates saved from useful",
          (dv.get("note") or {}).get("suggested_state_applied") is False and "not that it answers" in (dv.get("meaning") or ""), "")
    hsteps = [h for h in (tv.get("history") or []) if h.get("source_nudge_id") == d1["nudge_id"]]
    check("P0: the developed history step keeps nudge and request IDs (no more NOT RECORDED)",
          len(hsteps) == 1 and hsteps[0].get("worker_request_id") == v.get("request_id") and tv.get("history_total", 0) >= len(tv.get("history") or []), json.dumps(hsteps)[:200])
    code, lib = get("/api/cognition/library")
    note_text = (dv.get("note") or {}).get("text") or "(none)"
    check("P0-A6: the library list carries no Note text", code == 200 and note_text not in json.dumps(lib), "")
    code, hs = http("GET", f"{CORE}/api/cognition/thoughts/{tid}", headers={"X-TASS-Token": token(CORE)})
    check("P0: Thought detail needs the page token (no token -> 401)", http("GET", f"{CORE}/api/cognition/thoughts/{tid}")[0] == 401, "")
    lst = nudges(tid)
    check("P0-A3: nudge list readiness is derived (UNLOCKED, grant ENABLED, Engine REACHABLE), no stale BLOCKED",
          (lst.get("readiness") or {}).get("process_grant") == "ENABLED" and lst.get("processing", {}).get("state") == "READY"
          and "BLOCKED" not in json.dumps(lst), json.dumps(lst.get("readiness")))
    check("P0-A5: reading the Thought, library and nudge list made ZERO model requests",
          fixture()["requests"] == m0 and fixture()["nudges"] == n0, f"{m0}->{fixture()['requests']}")
    check("P0: the Engine's Thought file is byte-identical after the reads",
          bool(tsha) and hashlib.sha256(open(tfiles[0], "rb").read()).hexdigest() == tsha, tfiles[0] if tfiles else "no file")

    # ---- C: admission + crash recovery
    code, d2 = draft(tid, "Think slow about the project once more, please.", "s2-key-0002")
    code3, d3 = draft(tid, "A second nudge that has to wait its turn here.", "s2-key-0003")
    check("C: two more drafts saved", code == 201 and code3 == 201, f"{code} {code3}")
    base2 = fixture()["nudges"]
    out = {}
    def go(k, d):
        out[k] = process(d)
    th = [threading.Thread(target=go, args=(i, d2)) for i in range(3)]
    [t_.start() for t_ in th]; [t_.join() for t_ in th]
    codes = sorted(c for c, _ in out.values())
    check("C: three concurrent Process of one ID -> one 202, the rest 200 (same run)", codes == [200, 200, 202], str(codes))
    wait("slow run reaches the model", lambda: fixture()["nudges"] == base2 + 1, 20, 0.3)
    code, r = process(d3)
    check("C: a second nudge while one is active -> 409 busy, nothing submitted", code == 409 and not view(d3["nudge_id"]).get("run"), f"{code} {json.dumps(r)[:160]}")
    check("C: still exactly one model request in flight", fixture()["nudges"] == base2 + 1, "")
    kill9("engine")
    time.sleep(float(os.environ["FIXTURE_DELAY"]) + 1)  # the model finishes answering a dead connection
    start_engine()
    st = wait("UNCERTAIN after the Engine restart", lambda: run_state(d2["nudge_id"]) in ("UNCERTAIN", "RECOVERY_REQUIRED", "COMPLETED", "FAILED") and run_state(d2["nudge_id"]), 40, 1)
    v2 = view(d2["nudge_id"]).get("run") or {}
    check("C: Engine killed -9 mid-call -> UNCERTAIN (model_http attempt_uncertain)", st == "UNCERTAIN" and v2.get("model_http") == "attempt_uncertain", json.dumps(v2)[:300])
    time.sleep(3)
    check("C: never re-sent after the restart", fixture()["nudges"] == base2 + 1, f"{base2 + 1} -> {fixture()['nudges']}")
    code, t = thought_detail(eid, tid)
    steps = (t or {}).get("steps") or (t or {}).get("history") or []
    check("C: no development recorded for the uncertain run", not [s for s in steps if s.get("source_nudge_id") == d2["nudge_id"]], "")
    code, r = process(d2)
    check("C: Process on the UNCERTAIN run returns it, does not re-run", code == 200 and (r.get("run") or {}).get("state") == "UNCERTAIN", str(code))

    # ---- D: Stop. A never-submitted draft has nothing to stop (409, no Engine folder). A submitted
    # nudge the Engine never claimed (Engine down at delivery) is voided with a bound stop decision.
    code, r = core_post(f"/api/nudges/{d3['nudge_id']}/stop", {})
    check("D: Stop a never-submitted draft -> 409, no Engine folder", code == 409 and not run_dir(d3["nudge_id"]), f"{code} {json.dumps(r)[:160]}")
    code, r = core_post("/api/nudges/nudge_00000000000000000000/stop", {})
    check("D: Stop an unknown nudge ID -> refused, no Engine folder", code in (400, 404, 409) and not run_dir("nudge_00000000000000000000"), str(code))
    # A submitted nudge whose delivery is lost (the fault proxy drops the one POST): Core never re-sends;
    # Stop mints a bound stop decision, the Engine redeems it and writes the void.
    DROP_NUDGE.set()
    sent = len(PROXIED)
    code, r = process(d3)
    check("D: Process with the delivery lost -> 202 accepted locally", code == 202, f"{code} {json.dumps(r)[:160]}")
    time.sleep(3)
    s3 = run_state(d3["nudge_id"])
    check("D: lost delivery -> not terminal, not COMPLETED", s3 in ("SUBMITTED", "RECOVERY_REQUIRED"), s3)
    DROP_NUDGE.clear()
    time.sleep(2)
    check("D: Core sent exactly one POST and never re-sent", len(PROXIED) == sent + 1, f"{sent} -> {len(PROXIED)}")
    check("D: the Engine has no claim for it", not (run_dir(d3["nudge_id"]) and os.path.exists(f"{run_dir(d3['nudge_id'])}/claim.json")), "")
    code, r = core_post(f"/api/nudges/{d3['nudge_id']}/stop", {})
    check("D: Stop -> 200 NOT_DELIVERED (bound stop decision redeemed by the Engine)", code == 200 and (r.get("run") or {}).get("state") == "NOT_DELIVERED", f"{code} {json.dumps(r)[:200]}")
    rd3 = run_dir(d3["nudge_id"])
    claim = json.load(open(f"{rd3}/claim.json")) if rd3 and os.path.exists(f"{rd3}/claim.json") else {}
    check("D: Engine claim.json = void", claim.get("void") is True, json.dumps(claim)[:160])
    code, r = process(d3)
    check("D: Process after Stop returns NOT_DELIVERED, no POST, zero model requests", (r.get("run") or {}).get("state") == "NOT_DELIVERED"
          and len(PROXIED) == sent + 1 and fixture()["nudges"] == base2 + 1, f"{code} {json.dumps(r)[:160]}")
    code, r = core_post(f"/api/nudges/{d3['nudge_id']}/stop", {})
    check("D: Stop twice -> the same answer", code == 200 and (r.get("run") or {}).get("state") == "NOT_DELIVERED", str(code))
    code, d3b = draft(tid, "A second nudge that has to wait its turn here.", "s2-key-0003")
    check("D: the same draft request replays the same (stopped) draft, not a new one", code == 200 and d3b.get("nudge_id") == d3["nudge_id"], f"{code}")
    kill9("engine")
    code, r = process(d3)
    check("D: Process with the Engine down -> refused (503), nothing submitted", code == 503 and not view(d3["nudge_id"]).get("run"), f"{code} {json.dumps(r)[:120]}")
    start_engine()

    # ---- E: locked / revoked
    code, d4 = draft(tid, "Another pointer for the locked and revoked checks.", "s2-key-0004")
    idn_post("/v1/lock", {})
    code, r = process(d4)
    check("E: Identity LOCKED -> Process refused (403), nothing submitted", code == 403, f"{code} {json.dumps(r)[:160]}")
    code, r = core_post(f"/api/nudges/{d2['nudge_id']}/stop", {})
    check("E: Identity LOCKED -> Stop refused (403)", code == 403, f"{code} {json.dumps(r)[:160]}")
    idn_post("/v1/unlock", {"passphrase": n.PASS})
    check("E: (after unlock) the LOCKED attempt submitted nothing", not view(d4["nudge_id"]).get("run") and not run_dir(d4["nudge_id"]), "")
    code, r = idn_post("/v1/grants/skill", {"skill": "cognition.nudge-process", "revoke": True})
    check("E: owner revokes cognition.nudge-process", code == 200, str(code))
    lst = nudges(tid)
    check("P0-A3: after revoke the list says NOT_READY / grant NOT_ENABLED (display only)",
          (lst.get("readiness") or {}).get("process_grant") == "NOT_ENABLED" and lst.get("processing", {}).get("state") == "NOT_READY", json.dumps(lst.get("readiness")))
    code, r = process(d4)
    check("E: after revoke -> 403, nothing submitted", code == 403 and not view(d4["nudge_id"]).get("run"), f"{code} {json.dumps(r)[:160]}")
    code, r = core_post(f"/api/nudges/{d2['nudge_id']}/stop", {})
    check("E: Stop still works after revoke (D15): the UNCERTAIN run is returned unchanged", code == 200 and (r.get("run") or {}).get("state") == "UNCERTAIN", f"{code} {json.dumps(r)[:160]}")
    idn_post("/v1/grants/skill", {"skill": "cognition.nudge-process", "passphrase": n.PASS})

    # ---- X1/X19: the Engine key alone can neither process nor void (no decision -> 403, no folder)
    import hashlib
    att = "Engine key only, no Identity decision."
    code, r = http("POST", f"{ENG}/api/entities/{eid}/thoughts/{tid}/nudge",
                   {"nudge_id": d4["nudge_id"], "owner_entity_id": d4["owner_entity_id"], "attention": att,
                    "attention_sha256": hashlib.sha256(att.encode()).hexdigest(), "draft_expires_at": d4["expires_at"],
                    "decision_id": "dec_forged0000000001"}, {"Authorization": "Bearer " + ekey()})
    check("X1: Engine nudge with the key and a forged decision -> 403, no run folder", code == 403 and not run_dir(d4["nudge_id"]), f"{code} {json.dumps(r)[:120]}")
    code, r = http("POST", f"{ENG}/api/entities/{eid}/nudges/{d4['nudge_id']}/void",
                   {"owner_entity_id": d4["owner_entity_id"], "thought_id": tid, "submitted_sha256": "0" * 64,
                    "decision_id": "dec_forged0000000002"}, {"Authorization": "Bearer " + ekey()})
    check("X19: Engine void with the key and a forged decision -> 403, no run folder", code == 403 and not run_dir(d4["nudge_id"]), f"{code} {json.dumps(r)[:120]}")

    # ---- F: restore fence written by tass.sh's own function
    n.stop("core")
    p = subprocess.run(["bash", "-c", 'TASS_LIB_ONLY=1 source "$1" && restore_fence_write "$2" /x/backups/native.tgz',
                        "_", os.path.join(HERE, "..", "tass.sh"), f"{APP}/data"], capture_output=True, text=True,
                       env=clean_env({"APP_HOME": APP}))
    check("F: tass.sh restore_fence_write ran", p.returncode == 0, p.stderr[-200:])
    start_core()
    v4 = view(d4["nudge_id"])
    check("F: older draft is PRE_RESTORE", v4.get("pre_restore") is True and (v4.get("processing") or {}).get("state") == "PRE_RESTORE", json.dumps(v4.get("processing")))
    code, r = process(d4)
    check("F: PRE_RESTORE draft can't be processed (409)", code == 409 and "PRE_RESTORE" in json.dumps(r), f"{code} {json.dumps(r)[:160]}")
    check("F: processing not locked by a correct fence", not nudges().get("processing_locked"), str(nudges().get("processing_locked")))
    code, d5 = draft(tid, "A fresh pointer after the restore, to the same Thought.", "s2-key-0005")
    nb = fixture()["nudges"]
    code, r = process(d5)
    st = wait("post-restore COMPLETED", lambda: run_state(d5["nudge_id"]) in ("COMPLETED", "FAILED", "UNCERTAIN") and run_state(d5["nudge_id"]), 60, 1)
    check("F: a new draft after the restore processes once (COMPLETED)", st == "COMPLETED" and fixture()["nudges"] == nb + 1, f"{st} {nb}->{fixture()['nudges']}")

    # ---- G: Core restart reads, never re-sends
    nb = fixture()["nudges"]
    n.stop("core")
    start_core()
    time.sleep(3)
    states = {x["nudge_id"]: (x.get("run") or {}).get("state") for x in nudges().get("nudges") or []}
    check("G: after Core restart the receipts read back unchanged",
          states.get(d1["nudge_id"]) == "COMPLETED" and states.get(d2["nudge_id"]) == "UNCERTAIN" and states.get(d5["nudge_id"]) == "COMPLETED", json.dumps(states)[:300])
    check("G: Core restart sent nothing", fixture()["nudges"] == nb, "")
    decs = [json.loads(l) for l in open(f"{APP}/data/identity/decisions.jsonl")]
    pd = [x for x in decs if x.get("action") in ("cognition.nudge.process", "cognition.nudge.stop")]
    check("Identity audited bound decisions (mint + redeem, allow + deny)", any(x.get("allow") for x in pd) and any(not x.get("allow") for x in pd), f"{len(pd)} records")
    total_nudges = fixture()["nudges"]
    check("TOTAL: model requests from nudges = processed runs (3: B, C-uncertain, F)", total_nudges == base["nudges"] + 3, str(total_nudges))


if __name__ == "__main__":
    main()
