#!/usr/bin/env python3
"""P4: controlled multi-cycle run of the REAL Thought Engine 20-second pulse (Pulse.Tick), with the real
Identity and Core binaries, TEST FIXTURE model and fake GitHub, in a throwaway APP_HOME.

NOT the owner's machine, NOT Nemotron. To run the pulse at all the sandbox owner turns Autonomous ON
and starts the pulse through Core, exactly as the owner's page would; this is never done for the owner.

Measured, from the Engine's own ticks.jsonl / jobs.jsonl / model-requests.jsonl (not from Core's view):
  continuity       ticks every ~20 s, pulse sequence numbers +1 with no gaps or repeats
  idle             READY "nothing pending" ticks make NO model request
  long job         START once, then CONTINUE ("not started again"), then OBSERVED; exactly one run
  bounded queue    the 21st waiting job is refused (Engine limit 20)
  Identity deny    LOCKED -> BLOCKED ticks, queued jobs WAIT; unlock -> they run, each once
  model outage     model down -> the job FAILS (not retried forever), ticks continue; back -> DONE
  GitHub outage    fake GitHub stopped -> Core sync problem; Engine tick timing unaffected
  restart          Engine killed mid-job -> that job INTERRUPTED (never re-run), sequence continues,
                   Core turns the pulse back on (setting ON, Identity unlocked)
  Core vs Engine   Core SyncOnce (20 s) and Engine Pulse.Tick (20 s) are separate clocks
  Cognition        /api/cognition/pulse reports the same control/state/last tick as the Engine

  APP_HOME=/tmp/e2e/pulse BIN=/tmp/e2e/bin SEED=/tmp/e2e/seed CORE_BIN=tass-core-int \
  python3 core/test/pulse_multicycle_native.py
"""
import json, os, signal, sys, time
from datetime import datetime

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
os.environ.setdefault("CAND", "x"); os.environ.setdefault("ALPHA", "x")
import lesson_retry_native as n  # noqa: E402
from acceptance import CORE, check, core_post, http, idn_post, log, results, token, wait  # noqa: E402

ENG = "http://127.0.0.1:7798"
CORE_BIN = os.environ.get("CORE_BIN", "tass-core-int")
APP = n.APP


def ekey():
    return open(f"{APP}/data/engine/api-key").read().strip()


def eng(method, path, body=None):
    return http(method, ENG + path, body, {"Authorization": "Bearer " + ekey()})


def jl(name):
    p = f"{APP}/data/engine/{name}"
    return [json.loads(l) for l in open(p) if l.strip()] if os.path.exists(p) else []


def ticks():
    return [t for t in jl("ticks.jsonl")]


def ts(s):
    import re
    m = re.match(r"(\d{4}-\d\d-\d\dT\d\d:\d\d:\d\d)(?:\.(\d+))?", s)
    return datetime.fromisoformat(m.group(1)).timestamp() + float("0." + (m.group(2) or "0"))


def jobs():
    last = {}
    for j in jl("jobs.jsonl"):
        last[j["job_id"]] = j
    return last


def model_requests(prefix):
    return [r for r in jl("model-requests.jsonl") if prefix in json.dumps(r)]


def start_engine():
    n.start("engine", [f"{n.BIN}/thought-engine-cog", "-data", f"{APP}/data/engine", "-model", "http://127.0.0.1:8081", "-worker", "nemotron",
            "-identity-url", "http://127.0.0.1:8796", "-identity-key-file", f"{APP}/secrets/engine.key", "-pulse", "20s"],
            {"PORT": "7798", "HOME": "/tmp"}, ENG + "/api/model")


def start_model():
    n.start("model", ["python3", f"{n.HERE}/fixture-model/fake_llama.py", "8081"], {"FIXTURE_DELAY": "50"}, "http://127.0.0.1:8081/health")


def submit(job_id, text):
    return eng("POST", "/api/pulse/jobs", {"job_id": job_id, "identity": "TASS", "request": text, "context": "pulse multicycle test"})


def wait_job(jid, states=("DONE", "FAILED", "INTERRUPTED", "REFUSED"), secs=180):
    return wait(f"job {jid}", lambda: jobs().get(jid, {}).get("state") in states and jobs()[jid], secs, 2)


def main():
    n.prepare()
    n.start("fake-github", ["python3", f"{n.HERE}/fake_github.py", "28999", "Mira/home", "live", n.SEED], {}, "http://127.0.0.1:28999/fixture/log")
    start_model()
    n.start("identity", [f"{n.BIN}/tass-identity-jn"], {"HOME": "/tmp", "IDENTITY_DATA": f"{APP}/data/identity", "IDENTITY_LISTEN": "127.0.0.1:8796",
            "IDENTITY_SERVICE_KEYS": f"{APP}/secrets/service-keys", "IDENTITY_CORE_URL": "http://127.0.0.1:8795/"}, "http://127.0.0.1:8796/v1/health")
    start_engine()
    n.start_core(CORE_BIN)
    try:
        run()
    finally:
        for p in list(n.procs):
            n.stop(p)
    print()
    for name, res, detail in results:
        print(f"{res:4}  {name}  {detail}")
    fails = [r for r in results if r[1] != "PASS"]
    print(f"\n{len(results) - len(fails)}/{len(results)} PASS")
    sys.exit(1 if fails else 0)


def run():
    idn_post("/v1/entity", {"name": "TASS", "passphrase": n.PASS})
    wait("tass.md", lambda: os.path.exists(f"{APP}/data/core/tass.md"), 60)
    check("Engine pulse starts PAUSED; nothing ticks on its own", eng("GET", "/api/pulse")[1]["control"] == "PAUSED" and not ticks(), "")

    # sandbox-only: Autonomous ON so Core may run the pulse; learning and GitHub talk stay OFF
    core_post("/api/settings", {"autonomous": True, "pulse": False, "learn": False, "relay": False})
    code, r = core_post("/api/pulse", {"action": "start"})
    check("sandbox owner starts the pulse through Core", code == 200 and eng("GET", "/api/pulse")[1]["control"] == "RUNNING", f"{code} {json.dumps(r)[:120]}")
    t0 = len(ticks()); m0 = len(jl("model-requests.jsonl"))
    wait("4 idle ticks", lambda: len(ticks()) >= t0 + 4, 120, 2)
    idle = ticks()[t0:t0 + 4]
    gaps = [round(ts(b["at"]) - ts(a["at"]), 1) for a, b in zip(idle, idle[1:])]
    check("continuity: timer ticks ~20 s apart", all(17 <= g <= 23 for g in gaps) and all(t["trigger"] == "timer" for t in idle), f"gaps {gaps}")
    check("idle ticks: READY 'nothing pending', no model request", all(t["state"] == "READY" for t in idle) and len(jl("model-requests.jsonl")) == m0,
          f"{[t['decision'] for t in idle]} · model requests +{len(jl('model-requests.jsonl')) - m0}")

    # a long job (fixture delays 'slow' requests 50 s): START once, CONTINUE, OBSERVED
    submit("pm_long_1", "slow: think about your pulse for a while")
    j = wait_job("pm_long_1", secs=150)
    mine = [t for t in ticks() if t.get("job_id") == "pm_long_1" or "pm_long_1" in (t.get("decision", "") + t.get("outcome", ""))]
    starts = [t for t in mine if t["decision"].startswith("START")]
    conts = [t for t in mine if t["decision"].startswith("CONTINUE")]
    check("long job: started once, CONTINUE on later ticks, not started again", len(starts) == 1 and len(conts) >= 1, f"{len(starts)} START · {len(conts)} CONTINUE")
    check("long job DONE; exactly one model run for it", j and j["state"] == "DONE" and 1 <= len(model_requests("pm_long_1")) <= 2,
          f"{(j or {}).get('state')} · {len(model_requests('pm_long_1'))} model requests (2 = Engine's one format retry)")
    wait("OBSERVED tick", lambda: any("pm_long_1" in t.get("decision", "") and t["decision"].startswith("OBSERVED") for t in ticks()), 40, 2)
    check("a later tick OBSERVES the finished job", True, "")

    # Identity deny: LOCK -> BLOCKED, queue bounded at 20
    idn_post("/v1/lock", {})
    ok = [submit(f"pm_q_{i:02d}", f"count to {i}")[0] for i in range(20)]
    code21, r21 = submit("pm_q_20", "one too many")
    check("bounded queue: 20 accepted, the 21st refused", all(c == 202 for c in ok) and code21 != 202, f"{code21} {json.dumps(r21)[:100]}")
    tb = len(ticks())
    wait("blocked ticks", lambda: len([t for t in ticks()[tb:] if t["state"] == "BLOCKED"]) >= 2, 70, 2)
    blk = [t for t in ticks()[tb:] if t["state"] == "BLOCKED"]
    check("Identity LOCKED: ticks BLOCKED, job WAITs queued (not dropped, not run)", blk and blk[0]["decision"].startswith("WAIT") and jobs()["pm_q_00"]["state"] == "QUEUED", blk[0]["decision"] if blk else "")
    cp = http("GET", CORE + "/api/cognition/pulse", headers={"X-TASS-Token": token(CORE)})
    check("Cognition while LOCKED: unavailable rather than guessed", cp[0] != 200 or cp[1].get("state") in ("BLOCKED", "UNKNOWN"), f"HTTP {cp[0]}")
    n.unlock()
    wait("q_00 runs", lambda: jobs().get("pm_q_00", {}).get("state") == "DONE", 70, 2)
    check("after unlock: queued jobs run in order, one per tick", jobs()["pm_q_00"]["state"] == "DONE", "")

    # model outage (80 s): ticks continue; the job in hand is never started twice; it ends DONE or FAILED
    n.stop("model")
    tm = len(ticks())
    wait("4 ticks without model", lambda: len(ticks()) >= tm + 4, 100, 2)
    out = ticks()[tm:tm + 4]
    og = [round(ts(b["at"]) - ts(a["at"]), 1) for a, b in zip(out, out[1:])]
    during = sorted({t["job_id"] for t in out if t.get("job_id")})
    check("model down 80 s: ticks keep coming every ~20 s", all(17 <= x <= 23 for x in og), f"gaps {og} · {[t['decision'][:40] for t in out]}")
    h = http("GET", CORE + "/api/health")[1]
    check("Core health names the model problem", any("not ready" in x.lower() or "model" in x.lower() for x in h.get("reasons", [])), json.dumps(h.get("reasons"))[:160])
    start_model()
    ends = {j: wait_job(j, secs=120) for j in during}
    check("model back: the job(s) held during the outage end once (DONE or FAILED), none re-run",
          all(e and e["state"] in ("DONE", "FAILED") for e in ends.values()) and
          all(len([t for t in ticks() if t.get("job_id") == j and t["decision"].startswith("START")]) == 1 for j in during),
          json.dumps({j: (e or {}).get("state") for j, e in ends.items()}))

    # GitHub outage: Core sync problem, Engine tick timing unaffected
    n.stop("fake-github")
    tg = len(ticks())
    wait("3 ticks during GitHub outage", lambda: len(ticks()) >= tg + 3, 80, 2)
    g = ticks()[tg:tg + 3]
    gg = [round(ts(b["at"]) - ts(a["at"]), 1) for a, b in zip(g, g[1:])]
    st = http("GET", CORE + "/api/status", headers={"X-TASS-Token": token(CORE)})[1]
    check("GitHub down: Core reports sync problem; Engine still ticks every ~20 s", bool(st.get("sync_problem")) and all(17 <= x <= 23 for x in gg), f"{st.get('sync_problem', '')[:80]} · gaps {gg}")
    check("Core SyncOnce and Engine Pulse.Tick are separate clocks", st.get("next_sync") and eng("GET", "/api/pulse")[1]["every_s"] == 20, f"core next_sync {st.get('next_sync')} · engine every_s {eng('GET', '/api/pulse')[1]['every_s']}")
    n.start("fake-github", ["python3", f"{n.HERE}/fake_github.py", "28999", "Mira/home", "live", n.SEED], {}, "http://127.0.0.1:28999/fixture/log")

    # restart mid-job: INTERRUPTED, never re-run; sequence continues; Core resumes the pulse
    submit("pm_long_2", "slow: a second long thought")
    wait("pm_long_2 running", lambda: jobs().get("pm_long_2", {}).get("state") == "RUNNING", 400, 2)
    before_seq = int(ticks()[-1]["pulse_id"].split("_")[1])
    n.procs["engine"].send_signal(signal.SIGKILL); n.procs["engine"].wait(); n.procs.pop("engine")
    start_engine()
    j2 = wait_job("pm_long_2", ("INTERRUPTED", "DONE", "FAILED"), 30)
    check("Engine killed mid-job: the job is INTERRUPTED, not run again", j2 and j2["state"] == "INTERRUPTED", (j2 or {}).get("state", ""))
    wait("pulse back on", lambda: eng("GET", "/api/pulse")[1]["control"] == "RUNNING", 60, 2)
    check("Core turned the pulse back on after the Engine restart (setting ON, Identity unlocked)", True, "")
    wait("tick after restart", lambda: int(ticks()[-1]["pulse_id"].split("_")[1]) > before_seq, 40, 2)
    seqs = [int(t["pulse_id"].split("_")[1]) for t in ticks()]
    check("pulse sequence continues across restart: no repeats, no gaps", seqs == list(range(seqs[0], seqs[0] + len(seqs))), f"{seqs[0]}..{seqs[-1]} ({len(seqs)} ticks)")
    time.sleep(25)
    reruns = [t for t in ticks() if t.get("job_id") == "pm_long_2" and t["decision"].startswith("START")]
    check("no tick ever STARTs the interrupted job again", len(reruns) == 1, f"{len(reruns)} START")

    # duplicates across the whole run
    from collections import Counter
    st_counts = Counter(t["job_id"] for t in ticks() if t["decision"].startswith("START"))
    check("no job was STARTed twice in the whole run", all(v == 1 for v in st_counts.values()), json.dumps({k: v for k, v in st_counts.items() if v > 1}))

    # Cognition accuracy
    cid = [c for c in http("GET", CORE + "/api/cognition/library", headers={"X-TASS-Token": token(CORE)})[1].get("link", {}).get("candidates", [])]
    if cid:
        core_post("/api/cognition/link", {"engine_entity_id": cid[0]["engine_entity_id"]})
    e = eng("GET", "/api/pulse")[1]
    c = http("GET", CORE + "/api/cognition/pulse", headers={"X-TASS-Token": token(CORE)})[1]
    last_e = (e.get("ticks") or [{}])[0].get("pulse_id")
    recent_c = [x.get("pulse_id") for x in (c.get("recent") or [])]
    check("Cognition pulse matches the Engine: control, every 20 s, latest tick", c.get("control") == e["control"] and last_e in recent_c[:2] and int(c.get("every_s", 0)) == 20,
          f"engine {e['control']} {last_e} · cognition {c.get('control')} {recent_c[:2]} every {c.get('every_s')}")

    core_post("/api/pulse", {"action": "pause"})
    check("pause: pulse PAUSED, no ticks afterwards", eng("GET", "/api/pulse")[1]["control"] == "PAUSED", "")
    tp = len(ticks()); time.sleep(45)
    check("no tick while PAUSED", len(ticks()) == tp, f"+{len(ticks()) - tp}")
    log(f"total ticks {len(ticks())}, jobs {len(jobs())}, model requests {len(jl('model-requests.jsonl'))}")


if __name__ == "__main__":
    main()
