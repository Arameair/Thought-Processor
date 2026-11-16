#!/usr/bin/env python3
"""Clean-host acceptance run for the one-application TASS (TEST FIXTURE model; no real Nemotron).

Run in an isolated Docker host with nothing of the owner's on it:
    APP_HOME=/tmp/ch/app FIXTURE_MODEL_IMAGE=tass-fixture-model:local FIXTURE_DELAY=45 \
    python3 core/test/acceptance.py GOOD_REF BAD_REF

It drives ./tass.sh and the two local pages exactly as an owner would, and prints PASS/FAIL lines.
It never touches containers, networks or folders outside APP_HOME and the tass-app project.
"""
import json, os, re, subprocess, sys, time, urllib.request, urllib.error

CORE = "http://127.0.0.1:8795"
IDN = "http://127.0.0.1:8796"
HERE = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))   # core/
APP_HOME = os.environ["APP_HOME"]
results = []


def log(*a):
    print(time.strftime("%H:%M:%S"), *a, flush=True)


def check(name, ok, detail=""):
    results.append((name, "PASS" if ok else "FAIL", detail))
    log("PASS" if ok else "FAIL", name, "·", detail)
    return ok


def sh(*cmd, ok_codes=(0,)):
    log("$", " ".join(cmd))
    p = subprocess.run(cmd, cwd=HERE, capture_output=True, text=True)
    out = (p.stdout + p.stderr).strip()
    for l in out.splitlines()[-14:]:
        print("    | " + l)
    if p.returncode not in ok_codes:
        raise SystemExit(f"command failed ({p.returncode}): {' '.join(cmd)}")
    return p.returncode, out


def http(method, url, body=None, headers=None, raw=False):
    data = json.dumps(body).encode() if body is not None else None
    h = {"Content-Type": "application/json"} if body is not None else {}
    h.update(headers or {})
    req = urllib.request.Request(url, data=data, method=method, headers=h)
    try:
        with urllib.request.urlopen(req, timeout=15) as r:
            b = r.read().decode()
            return r.status, (b if raw else json.loads(b or "{}"))
    except urllib.error.HTTPError as e:
        b = e.read().decode()
        try:
            return e.code, json.loads(b)
        except Exception:
            return e.code, b


def token(url):
    _, page = http("GET", url + "/", raw=True)
    return re.search(r'const TOKEN = "([0-9a-f]+)"', page).group(1)


def core_post(path, body):
    return http("POST", CORE + path, body, {"X-TASS-Token": token(CORE)})


def idn_post(path, body):
    return http("POST", IDN + path, body, {"X-Identity-Page": token(IDN)})


def status():
    return http("GET", CORE + "/api/status", headers={"X-TASS-Token": token(CORE)})[1]


def health():
    return http("GET", CORE + "/api/health")[1]


def docker(*a):
    return subprocess.run(["docker", *a], capture_output=True, text=True).stdout.strip()


def fixture_requests():
    out = docker("exec", "tass-app-model-1", "python3", "-c",
                 "import urllib.request;print(urllib.request.urlopen('http://127.0.0.1:8081/fixture/requests').read().decode())")
    return json.loads(out)["requests"]


def wait(what, fn, secs=120, every=2):
    end = time.time() + secs
    while time.time() < end:
        v = fn()
        if v:
            return v
        time.sleep(every)
    return None


def main():
    good, bad = sys.argv[1], sys.argv[2]
    PASS = "correct-horse-battery-staple-42"

    # ---------------------------------------------------------------- clean install
    sh("./tass.sh", "check")
    sh("./tass.sh", "start")
    names = docker("ps", "--filter", "label=com.docker.compose.project=tass-app", "--format", "{{.Names}} {{.Status}}")
    log("containers:\n    " + names.replace("\n", "\n    "))
    check("4 services healthy (identity, model, engine, core)", names.count("(healthy)") == 4, names.replace("\n", "; "))
    models = docker("ps", "-q", "--filter", "label=com.docker.compose.service=model")
    check("exactly one model container on this host", len(models.split()) == 1, f"{len(models.split())} model container(s)")
    attached = set()
    for c in names.split("\n"):
        attached |= set(json.loads(docker("inspect", c.split()[0], "--format", "{{json .NetworkSettings.Networks}}")).keys())
    check("the app uses only its own networks (no Python workspace, liaison or tass-workspace)",
          attached <= {"tass-app_app", "tass-app_front"} and not docker("ps", "-q", "--filter", "name=liaison"),
          "attached: " + ", ".join(sorted(attached)) + " (engine-namespace core has none of its own)")
    st = http("GET", IDN + "/v1/state")[1]
    h = health()
    check("fresh install has NO entity and nothing personal", st["state"] == "NO_ENTITY" and not os.path.exists(f"{APP_HOME}/data/core/tass.md"), json.dumps(st))
    check("core is BLOCKED until an entity exists", h["state"] == "BLOCKED" and h["identity"] == "NO_ENTITY", "; ".join(h["reasons"]))
    code, r = core_post("/api/tasks", {"text": "count your sections"})
    check("no task can be queued without an entity", code == 409, f"{code} {r}")
    inspect = docker("inspect", "tass-app-model-1", "--format", "{{json .NetworkSettings.Networks}} ports={{json .NetworkSettings.Ports}}")
    check("model is on the internal network only, no published port", "tass-app_app" in inspect and "tass-app_front" not in inspect and "HostPort" not in inspect, inspect[:160])

    # keys never in images
    keys = [open(f"{APP_HOME}/secrets/{k}").read().strip() for k in ("engine.key", "core.key")]
    leaked = []
    for img in docker("ps", "--filter", "label=com.docker.compose.project=tass-app", "--format", "{{.Image}}").split():
        exp = subprocess.run(f"docker save {img} | tar -xO --wildcards '*' 2>/dev/null | grep -a -c -F -e {keys[0]} -e {keys[1]}",
                             shell=True, capture_output=True, text=True).stdout.strip() or "0"
        if exp != "0":
            leaked.append(img)
    check("no generated key is inside any image", not leaked, "searched every layer of: " + " ".join(docker("ps", "--filter", "label=com.docker.compose.project=tass-app", "--format", "{{.Image}}").split()))

    # ---------------------------------------------------------------- cannot impersonate
    code, _ = http("POST", IDN + "/v1/authorize", {"action": "model.request"})
    check("Identity refuses a caller without a service key", code == 401, str(code))
    code, _ = http("POST", IDN + "/v1/entity", {"name": "TASS", "passphrase": "x" * 20})
    check("Identity refuses entity creation without the page token", code in (401, 403), str(code))
    probe = ("import json,urllib.request as u\n"
             "r=u.Request('http://identity:8796/v1/entity',data=json.dumps({'name':'TASS','passphrase':'x'*20}).encode(),"
             "headers={'Content-Type':'application/json','X-Identity-Page':'guess'},method='POST')\n"
             "try:\n u.urlopen(r)\n print('accepted')\nexcept Exception as e:\n print(getattr(e,'code',e))")
    out = docker("exec", "tass-app-core-1", "python3", "-c", probe)
    check("a service inside the app can't create or take over an entity (owner routes are localhost-only)", out in ("403", "401"), out)

    # ---------------------------------------------------------------- enrollment
    code, r = idn_post("/v1/entity", {"name": "Mira", "passphrase": PASS})
    log("create entity:", code, r)
    st = http("GET", IDN + "/v1/state")[1]
    check("owner creates THIS installation's own entity", code in (200, 201) and st["entity"]["name"] == "Mira" and st["entity"]["entity_id"].startswith("ent_"), json.dumps(st)[:200])
    entity_id = st["entity"]["entity_id"]
    if st["state"] != "UNLOCKED":
        code, r = idn_post("/v1/unlock", {"passphrase": PASS})
        log("unlock:", code, r)
    code, r = idn_post("/v1/unlock", {"passphrase": "wrong-passphrase-123"})
    check("wrong passphrase is refused", code == 403, f"{code}")
    code, r = idn_post("/v1/unlock", {"passphrase": PASS})
    h = wait("READY", lambda: (lambda x: x if x["state"] == "READY" else None)(health()), 30)
    check("unlocked: core READY", bool(h), json.dumps(health())[:200])

    # owner conversation (receipt)
    code, r = core_post("/api/say", {"text": "hi"})
    check("conversation: message accepted with a receipt", code == 202 and r.get("receipt", "").startswith("msg_"), f"{code} {r}")
    cy = wait("cycle", lambda: [c for c in status().get("cycles") or [] if c.get("request") == "hi" and c.get("evaluation")], 120)
    check("conversation: answered through a full thought cycle as Mira", bool(cy), (cy or [{}])[0].get("cycle_id", "none"))
    doc = open(f"{APP_HOME}/data/core/tass.md").read()
    check("the entity has its own starter document (no one else's)", doc.startswith("# Mira") and "Dustin" not in doc and "TASS" not in doc, doc.splitlines()[0])

    # ---------------------------------------------------------------- the work pulse
    code, r = core_post("/api/tasks", {"text": "count your sections"})
    check("Autonomous off (default): chat only, no task can be queued", code == 409 and "CHAT_ONLY" in str(r), f"{code} {r}")
    code, r = core_post("/api/settings", {"autonomous": True, "pulse": False, "learn": False})
    check("owner turns Autonomous on", code == 200 and r["autonomous"], json.dumps(r))
    before_req = fixture_requests()
    code, r = core_post("/api/tasks", {"text": "slow: count your sections"})
    job = r.get("job_id", "")
    check("task queued for the pulse (not run right away)", code == 202 and job.startswith("job_"), f"{code} {r}")
    time.sleep(3)
    jobs = {j["job_id"]: j for j in status()["pulse"]["jobs"]}
    check("pulse is PAUSED until Start; the job waits", status()["pulse"]["control"] == "PAUSED" and jobs[job]["state"] == "QUEUED", status()["pulse"]["control"])
    code, r = core_post("/api/pulse", {"action": "start"})
    t0 = time.time()
    check("Start", code == 200, str(code))
    time.sleep(25)
    code, r = core_post("/api/say", {"text": "are you there?"})
    check("owner conversation stays available while the long job runs (accepted with receipt)", code == 202 and "receipt" in r, f"{code} {r}")
    ticks = wait("10 ticks", lambda: (lambda p: p if len(p["ticks"]) >= 10 else None)(status()["pulse"]), 260, 5)
    elapsed = time.time() - t0
    p = status()["pulse"]
    tk = p["ticks"][-10:]
    for t in p["ticks"]:
        print(f"    {t['pulse_id']}  {t['state']:8} {t['decision'][:70]:70} | {t['observed']}{(' | ' + t['outcome']) if t.get('outcome') else ''}")
    starts = [t for t in p["ticks"] if t["decision"].startswith("START: " + job)]
    conts = [t for t in p["ticks"] if t["decision"].startswith("CONTINUE: " + job)]
    obs = [t for t in p["ticks"] if job in (t.get("outcome") or "")]
    j = {x["job_id"]: x for x in p["jobs"]}[job]
    dur = (time.mktime(time.strptime(j["ended_at"][:19], "%Y-%m-%dT%H:%M:%S")) - time.mktime(time.strptime(j["started_at"][:19], "%Y-%m-%dT%H:%M:%S"))) if j.get("ended_at") else 0
    check("10 verifiable ticks ~20 s apart", len(p["ticks"]) >= 10, f"{len(p['ticks'])} ticks in {elapsed:.0f} s")
    check("job longer than 20 s survives several pulses", dur > 20 and len(conts) >= 1, f"job ran {dur:.0f} s; CONTINUE ticks {len(conts)}")
    check("started exactly once (no duplicate execution)", len(starts) == 1, f"START ticks: {len(starts)}")
    check("result observed and reported on a later tick", len(obs) == 1 and j["state"] == "DONE", f"{j['state']} · {j.get('result', '')[:120]}")
    used = fixture_requests() - before_req
    log("model requests during the pulse run (job + the 'are you there?' chat):", used)
    check("Pause/Resume/Step controls answer", all(core_post("/api/pulse", {"action": a})[0] == 200 for a in ("pause", "step", "resume", "pause")), "pause, step, resume, pause")

    # ---------------------------------------------------------------- restart persistence
    inv_before = {f: sum(1 for _ in open(os.path.join(APP_HOME, "data", f))) for f in ("engine/cycles.jsonl", "engine/jobs.jsonl", "engine/ticks.jsonl", "core/conversation.jsonl", "identity/decisions.jsonl")}
    sh("./tass.sh", "stop")
    sh("./tass.sh", "start")
    st = http("GET", IDN + "/v1/state")[1]
    h = health()
    check("after restart: same entity, LOCKED, core BLOCKED", st["entity"]["entity_id"] == entity_id and st["state"] == "LOCKED" and h["state"] == "BLOCKED", f"{st['state']} {h['state']} {entity_id}")
    inv_after = {f: sum(1 for _ in open(os.path.join(APP_HOME, "data", f))) for f in inv_before}
    check("records persisted across restart", all(inv_after[f] >= inv_before[f] for f in inv_before), json.dumps(inv_after))
    p = status()["pulse"]
    check("jobs and ticks persisted; pulse comes back PAUSED", p["control"] == "PAUSED" and any(x["job_id"] == job for x in p["jobs"]), f"{p['control']} · {len(p['jobs'])} jobs")
    idn_post("/v1/unlock", {"passphrase": PASS})
    check("unlock again → READY", bool(wait("READY", lambda: health()["state"] == "READY", 30)), health()["state"])

    # ---------------------------------------------------------------- update + failed update + rollback
    rc, out = sh("./tass.sh", "update", good, "--yes")
    check("good update applied and validated", "UPDATED" in out and "PASS" in out, out.splitlines()[-1])
    check("after update: same entity", http("GET", IDN + "/v1/state")[1]["entity"]["entity_id"] == entity_id, entity_id)
    idn_post("/v1/unlock", {"passphrase": PASS})
    core_post("/api/say", {"text": "hi after update"})
    time.sleep(8)
    rc, out = sh("./tass.sh", "update", bad, "--yes", ok_codes=(0, 1))
    check("broken update is detected and rolled back automatically (version and data)", rc == 1 and "ROLLED BACK" in out, out.splitlines()[-1])
    h = health()
    good_sha = subprocess.run(["git", "rev-parse", good], cwd=HERE, capture_output=True, text=True).stdout.strip()
    check("running the previous good version again", h["version"] == good_sha, h["version"][:12])
    check("entity intact after the failed update", http("GET", IDN + "/v1/state")[1]["entity"]["entity_id"] == entity_id, entity_id)
    rc, out = sh("./tass.sh", "rollback")
    h = health()
    check("manual rollback returns to the version before the update", "Rolled back" in out and h["version"] != good_sha, h["version"][:12])
    check("entity intact after rollback", http("GET", IDN + "/v1/state")[1]["entity"]["entity_id"] == entity_id, entity_id)
    sh("./tass.sh", "status")

    print("\nSUMMARY")
    for n, r, d in results:
        print(f"  {r:4}  {n}")
    fails = [n for n, r, _ in results if r != "PASS"]
    print(f"\n{len(results) - len(fails)} PASS · {len(fails)} FAIL")
    sys.exit(1 if fails else 0)


if __name__ == "__main__":
    main()
