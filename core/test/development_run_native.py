#!/usr/bin/env python3
"""Stage 3A (#120): "How this Note was produced" on the REAL stack: real Identity, the real Thought Engine
candidate (with the read-only run record), the candidate Core, and the TEST FIXTURE model (NOT Nemotron).
Isolated ports, throwaway APP_HOME and owner-port sentinels from nudge_process_native.py. Never the
owner's machine or data.

R1  one real supervised nudge is processed (setup, as in the Stage 2 harness)
R2  GET /api/development/runs/<nudge>: every stage of a completed run is RECORDED / CODE-DEFINED / NOT PART
    OF THIS PATH / NOT RECORDED as expected, from the REAL Engine record: the logged prompt holds the
    attention's digest only; the full reply matches the run's reply digest; the saved Note is the Thought's
R3  reading it calls no model, adds no Engine log line and no run file, changes no Thought
R4  scoped and fail-closed: a draft never processed, an unknown or malformed ID -> 404; Identity LOCKED -> 403;
    POST -> 405; the Engine record itself needs the Engine key
R5  in a real browser: Cognition links the development to /dev?run=<nudge>; Development shows the 12 stages,
    the next action, and sends zero POSTs
    APP_HOME=/tmp/e2e/s3a BIN=/tmp/e2e/bin SEED=/tmp/e2e/seed CORE_BIN=… ENGINE_BIN=… IDENTITY_BIN=… \\
    [S2_PORT_BASE=39100] python3 core/test/development_run_native.py
"""
import json, os, sys, time, urllib.request

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import nudge_process_native as h  # noqa: E402  (runs the isolation preflight on import)
from acceptance import core_post, http, idn_post, token, wait  # noqa: E402

results = []


def check(name, ok, detail=""):
    results.append((name, "PASS" if ok else "FAIL", detail if not ok else ""))
    print(("PASS " if ok else "FAIL ") + name + ("" if ok else "  " + str(detail)[:300]), flush=True)


def raw_get(path, tok, method="GET"):
    req = urllib.request.Request(h.CORE + path, method=method, headers={"X-TASS-Token": tok})
    try:
        with urllib.request.urlopen(req, timeout=30) as r:
            return r.status, r.read().decode(), dict(r.headers)
    except urllib.error.HTTPError as e:
        return e.code, e.read().decode(), dict(e.headers)


def size(p):
    return os.path.getsize(p) if os.path.exists(p) else -1


def files(root):
    out = {}
    for d, _, fs in os.walk(root):
        for f in fs:
            p = os.path.join(d, f)
            st = os.stat(p)
            out[p] = (st.st_size, st.st_mtime_ns)
    return out


def setup():
    code, r = idn_post("/v1/entity", {"name": "Mira", "passphrase": h.n.PASS})
    check("setup: real Identity entity UNLOCKED", code == 201, str(code))
    core_post("/api/say", {"text": "Hello Mira, what are you thinking about?"})
    cands = wait("Engine entity", lambda: (h.get("/api/cognition/library")[1].get("link") or {}).get("candidates"), 90, 2)
    eid = cands[0]["engine_entity_id"]
    check("setup: owner links Cognition", core_post("/api/cognition/link", {"engine_entity_id": eid})[0] == 200)
    for i in range(3):
        http("POST", f"{h.ENG}/api/entities/{eid}/events", {"event_id": f"s3a-obs-{i}", "source": "test",
             "text": f"The owner worked on the local project again (observation {i})"}, {"Authorization": "Bearer " + h.ekey()})
    thoughts = wait("a persistent Thought", lambda: h.get("/api/cognition/library")[1].get("thoughts"), 120, 2)
    tid = thoughts[0]["source_id"]
    http("POST", f"{h.ENG}/api/entities/{eid}/grant", {"enabled": True}, {"Authorization": "Bearer " + h.ekey()})
    idn_post("/v1/grants/skill", {"skill": "cognition.nudge-draft", "passphrase": h.n.PASS})
    idn_post("/v1/grants/skill", {"skill": "cognition.nudge-process", "passphrase": h.n.PASS})
    time.sleep(3)
    code, d = h.draft(tid, "Pay attention to whether the local project keeps coming back.", "s3a-key-0001")
    code2, _ = h.process(d)
    st = wait("COMPLETED", lambda: h.run_state(d["nudge_id"]) in ("COMPLETED", "FAILED", "UNCERTAIN") and h.run_state(d["nudge_id"]), 60, 1)
    check("R1 one real supervised nudge COMPLETED", code == 201 and code2 == 202 and st == "COMPLETED", f"{code} {code2} {st}")
    code, d2 = h.draft(tid, "A second draft that is never processed.", "s3a-key-0002")
    return eid, tid, d["nudge_id"], d2["nudge_id"]


def run():
    eid, tid, nid, unprocessed = setup()
    tok = token(h.CORE)
    eng = f"{h.APP}/data/engine"
    logs = [os.path.join(eng, "calls.jsonl"), os.path.join(eng, "model-requests.jsonl")]
    before_logs, before_files, base = [size(p) for p in logs], files(f"{h.APP}/data"), h.fixture()
    code_t0, t0 = h.thought_detail(eid, tid)

    code, body, hdr = raw_get("/api/development/runs/" + nid, tok)
    check("R2 GET /api/development/runs/<nudge> -> 200 with the scope attestation", code == 200 and hdr.get("X-Cognition-Identity", "").startswith("ent_"), f"{code} {body[:200]}")
    r = json.loads(body) if code == 200 else {}
    st = {s["stage"]: s for s in r.get("stages", [])}
    want = {"thought": "RECORDED", "attention": "RECORDED", "authorization": "RECORDED", "context_sent": "RECORDED", "instruction": "CODE-DEFINED",
            "model_call": "RECORDED", "model_reply": "RECORDED", "parse": "CODE-DEFINED", "saved": "RECORDED",
            "memory_retrieval": "NOT PART OF THIS PATH", "tools": "NOT PART OF THIS PATH", "model_reasoning": "NOT RECORDED"}
    got = {k: v["status"] for k, v in st.items()}
    check("R2 every stage has the expected status from the REAL Engine record", got == want, json.dumps({k: (v["status"], v["summary"][:90]) for k, v in st.items() if want.get(k) != v["status"]}))
    check("R2 header: Thought, request, verified", r.get("thought_id") == tid and (r.get("request_id") or "").startswith("mr_") and r.get("verified") is True, json.dumps({k: r.get(k) for k in ("thought_id", "request_id", "verified", "run_state")}))
    prompt = (st.get("context_sent") or {}).get("text", "")
    check("R2 the logged prompt holds the attention's digest only", "[owner attention sha256:" in prompt and "Pay attention to whether" not in prompt
          and "Reply with exactly two lines" in prompt, prompt[-300:])
    check("R2 the attention stage shows the owner's own words", (st.get("attention") or {}).get("text") == "Pay attention to whether the local project keeps coming back.", "")
    reply = (st.get("model_reply") or {}).get("text", "")
    notes = [x for x in (t0 or {}).get("notes") or [] if x.get("source_nudge_id") == nid]
    check("R2 the saved Note is the Thought's Note, and it came from the shown reply", len(notes) == 1 and (st.get("parse") or {}).get("text") == notes[0].get("text")
          and notes[0].get("text", "")[:20] in reply, f"{reply[:200]} / {json.dumps(notes)[:200]}")
    check("R2 suggested state not applied", "not applied" in json.dumps((st.get("parse") or {}).get("fields")), json.dumps((st.get("parse") or {}).get("fields")))

    for _ in range(2):
        raw_get("/api/development/runs/" + nid, tok)
    after_files = files(f"{h.APP}/data")
    eng_changed = [p for p in set(before_files) | set(after_files) if p.startswith(eng) and before_files.get(p) != after_files.get(p)]
    check("R3 reading three times: zero model requests", h.fixture()["nudges"] == base["nudges"], f"{base} -> {h.fixture()}")
    check("R3 no Engine log line added", [size(p) for p in logs] == before_logs, f"{before_logs} -> {[size(p) for p in logs]}")
    check("R3 no Engine file created or changed", not eng_changed, eng_changed[:5])
    code_t1, t1 = h.thought_detail(eid, tid)
    check("R3 the Thought is unchanged", json.dumps(t0, sort_keys=True) == json.dumps(t1, sort_keys=True), "")
    core_runs = [p for p in set(before_files) | set(after_files) if "/nudge-runs/" in p and before_files.get(p) != after_files.get(p)]
    check("R3 no Core run receipt created or changed", not core_runs, core_runs[:5])

    for name, path, want_code in [("a draft never processed", "/api/development/runs/" + unprocessed, 404),
                                  ("an unknown nudge", "/api/development/runs/nudge_00000000000000000099", 404),
                                  ("a malformed ID", "/api/development/runs/nudge_..%2f..%2fetc", 404)]:
        c, b, hd = raw_get(path, tok)
        check(f"R4 {name} -> {want_code}, no attestation", c == want_code and "X-Cognition-Identity" not in hd, f"{c} {b[:120]}")
    c, b, _ = raw_get("/api/development/runs/" + nid, tok, method="POST")
    check("R4 POST -> 405", c == 405, str(c))
    c, b, _ = raw_get("/api/development/runs/" + nid, "wrong-token")
    check("R4 without the page token -> 401", c == 401, str(c))
    c, _ = http("GET", f"{h.ENG}/api/entities/{eid}/nudges/{nid}/record")
    check("R4 the Engine record needs the Engine key", c == 401, str(c))
    c, rec = http("GET", f"{h.ENG}/api/entities/{eid}/nudges/{nid}/record", None, {"Authorization": "Bearer " + h.ekey()})
    check("R4 with the key the Engine returns this run's record", c == 200 and (rec.get("run") or {}).get("nudge_id") == nid and rec.get("call") and rec.get("model_request"), str(c))

    browser(tok, tid, nid, r.get("request_id"))

    code, _ = idn_post("/v1/lock", {})
    c, b, hd = raw_get("/api/development/runs/" + nid, tok)
    check("R4 Identity LOCKED -> 403, nothing returned", c == 403 and "X-Cognition-Identity" not in hd and "Pay attention" not in b, f"{code} {c} {b[:120]}")


def browser(tok, tid, nid, req):
    try:
        from playwright.sync_api import sync_playwright
    except ImportError:
        results.append(("R5 real browser", "NOT RUN", "playwright missing")); return
    with sync_playwright() as p:
        b = p.chromium.launch(headless=True, args=["--no-sandbox"])
        pg = b.new_page(viewport={"width": 960, "height": 1000})
        sent, errs = [], []
        pg.on("request", lambda q: sent.append((q.method, q.url.replace(h.CORE, ""))))
        pg.on("pageerror", lambda e: errs.append(str(e)))
        pg.goto(h.CORE + "/cognition#thought=" + tid)
        try:
            pg.wait_for_selector("#dev-" + req + " a.how", timeout=20000)
            href = pg.locator("#dev-" + req + " a.how").get_attribute("href")
        except Exception as e:
            href = "missing: " + str(e)[:80]
        check("R5 Cognition links the development to its run", href == "/dev?run=" + nid, href)
        if href == "/dev?run=" + nid:
            pg.locator("#dev-" + req + " a.how").click()
        else:
            pg.goto(h.CORE + "/dev?run=" + nid)
        try:
            pg.wait_for_selector("#runView li.st", timeout=20000)
        except Exception:
            pass
        statuses = pg.evaluate("[...document.querySelectorAll('#runView li.st')].map(l=>l.id+':'+l.dataset.status)")
        nxt = pg.evaluate("(document.querySelector('#runNext a')||{}).textContent||''")
        check("R5 Development shows the 12 stages and the next action", len(statuses) == 12 and nxt == "Back to this Note in Cognition", statuses)
        pg.screenshot(path="/tmp/stage3a-native-half.png", full_page=False)
        pg.reload(); pg.wait_for_timeout(1500)
        check("R5 zero POSTs from the browser (Cognition, link, Development, refresh)", not [s for s in sent if s[0] != "GET"], [s for s in sent if s[0] != "GET"])
        check("R5 no page errors", not errs, errs)
        b.close()


def main():
    h.n.prepare()
    watched = h.start_sentinels()
    h.start("model", ["python3", f"{h.HERE}/fixture-model/fake_llama.py", str(h.P_MODEL)], {"FIXTURE_DELAY": "1"}, h.MODEL + "/health")
    h.start("identity", [f"{h.BIN}/{h.IDENTITY_BIN}"], {"IDENTITY_DATA": f"{h.APP}/data/identity", "IDENTITY_LISTEN": f"127.0.0.1:{h.P_IDN}",
            "IDENTITY_SERVICE_KEYS": f"{h.APP}/secrets/service-keys", "IDENTITY_CORE_URL": h.CORE + "/"}, h.IDN + "/v1/health")
    h.start_engine(); h.start_proxy(); h.start_core()
    try:
        run()
    finally:
        for name in list(h.n.procs):
            h.n.stop(name)
    check("ISOLATION: no connection to any watched owner port", not h.SENTINEL_HITS, str(sorted(set(h.SENTINEL_HITS))))
    fails = [r for r in results if r[1] == "FAIL"]
    notrun = [r for r in results if r[1] == "NOT RUN"]
    print(f"\n{len(results) - len(fails) - len(notrun)}/{len(results)} PASS, {len(fails)} FAIL, {len(notrun)} NOT RUN "
          f"(sentinels {watched}; TEST FIXTURE model; not Nemotron, not the owner's machine)")
    sys.exit(1 if fails else 0)


if __name__ == "__main__":
    main()
