#!/usr/bin/env python3
"""Same #71 replay as lesson_retry_acceptance.py, without Docker (for sandboxes with no registry).

Runs the REAL pinned Identity and Thought Engine binaries plus the old and new Core binaries as
local processes on one data directory, with the TEST FIXTURE model and fake GitHub. An "update"
is: stop the old Core, start the new Core binary on the same data (what ./tass.sh update does
after building images). Build the binaries first (see the PR description), then:

    APP_HOME=/tmp/e2e/app BIN=/tmp/e2e/bin SEED=/tmp/e2e/seed CAND=<sha> ALPHA=<sha> \
    python3 core/test/lesson_retry_native.py
"""
import json, os, subprocess, sys, time

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from acceptance import CORE, check, core_post, http, idn_post, log, results, wait  # noqa: E402

HERE = os.path.dirname(os.path.abspath(__file__))
APP, BIN, SEED = os.environ["APP_HOME"], os.environ["BIN"], os.environ["SEED"]
CAND, ALPHA = os.environ["CAND"], os.environ["ALPHA"]
PASS = "correct-horse-battery-staple-42"
L01, L02 = "lessons/claude/01-my-work-pulse.md", "lessons/claude/02-how-i-learn.md"
procs = {}


def key():
    return os.urandom(32).hex()


def prepare():
    for d in ("data/identity", "data/core", "data/engine", "secrets", "logs"):
        os.makedirs(f"{APP}/{d}", exist_ok=True)
    ek, ck = key(), key()
    open(f"{APP}/secrets/engine.key", "w").write(ek)
    open(f"{APP}/secrets/core.key", "w").write(ck)
    open(f"{APP}/secrets/service-keys", "w").write(f"engine {ek}\ncore {ck}\n")
    open(f"{APP}/data/engine/api-key", "w").write(key())
    open(f"{APP}/secrets/github_token", "w").write("fixture-token")
    for f in ("engine.key", "core.key", "service-keys", "github_token"):
        os.chmod(f"{APP}/secrets/{f}", 0o600)
    os.chmod(f"{APP}/data/engine/api-key", 0o600)


def up(url):
    try:
        return http("GET", url)[0] in (200, 401, 403)
    except Exception:
        return False


def start(name, cmd, env, health_url):
    out = open(f"{APP}/logs/{name}.log", "a")
    procs[name] = subprocess.Popen(cmd, env={**os.environ, **env}, stdout=out, stderr=subprocess.STDOUT)
    ok = wait(name, lambda: up(health_url), 40, 1)
    log(f"started {name}: {' '.join(cmd[:1])} healthy={bool(ok)}")


def stop(name):
    p = procs.pop(name, None)
    if p:
        p.terminate()
        p.wait(10)


def start_core(binary):
    start("core", [f"{BIN}/{binary}"], {
        "HOME": "/tmp", "TASS_DATA": f"{APP}/data/core", "TASS_LISTEN": "127.0.0.1:8795",
        "TASS_ENGINE_URL": "http://127.0.0.1:7798", "TASS_ENGINE_KEY": f"{APP}/data/engine/api-key",
        "TASS_IDENTITY_URL": "http://127.0.0.1:8796", "TASS_IDENTITY_KEY_FILE": f"{APP}/secrets/core.key",
        "TASS_REPO": "Mira/home", "TASS_BRANCH": "live", "TASS_DOC": "tass.md", "TASS_LESSONS": "lessons",
        "TASS_TOKEN_FILE": f"{APP}/secrets/github_token", "TASS_GITHUB_API": "http://127.0.0.1:28999"},
        CORE + "/api/health")


def lessons():
    st = http("GET", CORE + "/api/status", headers={"X-TASS-Token": __import__("acceptance").token(CORE)})[1]
    return {l["path"]: l for l in (st.get("learning") or {}).get("lessons") or []}


def lesson(p):
    return lessons().get(p) or {}


def wait_status(p, want, secs=120):
    return wait(f"{p} -> {want}", lambda: lesson(p).get("status") in want and lesson(p), secs)


def unlock():
    return wait("unlock", lambda: idn_post("/v1/unlock", {"passphrase": PASS})[0] == 200, 30, 1)


def main():
    prepare()
    start("fake-github", ["python3", f"{HERE}/fake_github.py", "28999", "Mira/home", "live", SEED], {}, "http://127.0.0.1:28999/fixture/log")
    start("model", ["python3", f"{HERE}/fixture-model/fake_llama.py", "8081"], {"FIXTURE_LESSON_FAIL_FIRST": "My work pulse"}, "http://127.0.0.1:8081/health")
    start("identity", [f"{BIN}/tass-identity"], {"HOME": "/tmp", "IDENTITY_DATA": f"{APP}/data/identity", "IDENTITY_LISTEN": "127.0.0.1:8796",
          "IDENTITY_SERVICE_KEYS": f"{APP}/secrets/service-keys", "IDENTITY_CORE_URL": "http://127.0.0.1:8795/"}, "http://127.0.0.1:8796/v1/health")
    start("engine", [f"{BIN}/thought-engine", "-data", f"{APP}/data/engine", "-model", "http://127.0.0.1:8081", "-worker", "nemotron",
          "-identity-url", "http://127.0.0.1:8796", "-identity-key-file", f"{APP}/secrets/engine.key", "-pulse", "20s"],
          {"PORT": "7798", "HOME": "/tmp"}, "http://127.0.0.1:7798/api/model")
    start_core("tass-core-alpha")
    try:
        run()
    finally:
        for n in list(procs):
            stop(n)
    print()
    for name, res, detail in results:
        print(f"{res:4}  {name}  {detail}")
    fails = [r for r in results if r[1] != "PASS"]
    print(f"\n{len(results) - len(fails)}/{len(results)} PASS")
    sys.exit(1 if fails else 0)


def run():
    code, r = idn_post("/v1/entity", {"name": "Mira", "passphrase": PASS})
    check("real Identity 91e1521: entity created UNLOCKED", code == 201 and r["state"] == "UNLOCKED", str(code))
    wait("tass.md from GitHub", lambda: os.path.exists(f"{APP}/data/core/tass.md"), 60)
    h = http("GET", CORE + "/api/health")[1]
    check("old Core running", h.get("version") == ALPHA, h.get("version", ""))
    code, r = core_post("/api/lessons/discover", {})
    check("Find lessons (Autonomous OFF)", code == 200 and L01 in lessons() and L02 in lessons(), f"{code} {json.dumps(r)[:150]}")

    code, r = core_post("/api/lessons/study", {"path": L02})
    check("old Core: study 02 queued", code == 200, json.dumps(r)[:200])
    l = wait_status(L02, ("PASSED", "RETRY", "MODEL_FAILED"))
    check("old Core + real Engine: 02 PASSED 3/3", bool(l) and l["status"] == "PASSED" and l["right"] == 3, json.dumps({k: (l or {}).get(k) for k in ("status", "right", "note")}))
    code, r = core_post("/api/lessons/approve", {"path": L02})
    check("old Core: 02 approved -> LEARNED", code == 200 and lesson(L02)["status"] == "LEARNED", f"{code} {json.dumps(r)[:150]}")

    code, r = core_post("/api/lessons/study", {"path": L01})
    first = r.get("receipt", "")
    check("old Core: study 01 queued", code == 200 and first.endswith("_1"), first)
    l = wait_status(L01, ("MODEL_FAILED", "PASSED", "RETRY"))
    check("old Core + real Engine: 01 MODEL_FAILED on REMEMBER-only reply, NOT GRADED", bool(l) and l["status"] == "MODEL_FAILED", (l or {}).get("note", "")[:220])
    core_post("/api/lessons/retry", {"path": L01})
    l = lesson(L01)
    check("old Core: Retry -> NEW attempt 0, old job still on card", l["status"] == "NEW" and l["attempts"] == 0 and l.get("job_id") == first, json.dumps({k: l.get(k) for k in ("status", "attempts", "job_id")}))
    code, r = core_post("/api/lessons/study", {"path": L01})
    check("old Core: Study -> HTTP 409 duplicate job from real Engine (REPRODUCES #71)", code == 409 and "already exists" in json.dumps(r), f"{code} {json.dumps(r)[:220]}")

    # "Update": same data, new Core binary (Engine and Identity keep running, as their pins are unchanged).
    stop("core")
    start_core("tass-core-cand")
    h = http("GET", CORE + "/api/health")[1]
    check("candidate Core running on the same data", h.get("version") == CAND, h.get("version", ""))
    l = lesson(L01)
    hist = l.get("history") or []
    check("candidate: old failed attempt migrated into history; job_seq 1", any(x["job_id"] == first for x in hist) and l.get("job_seq") == 1, json.dumps([(x["job_id"], x["source"]) for x in hist]))
    check("candidate: 02 still LEARNED", lesson(L02)["status"] == "LEARNED", "")
    code, r = core_post("/api/lessons/study", {"path": L01})
    second = r.get("receipt", "")
    check("candidate: Study 01 accepted by real Engine with NEW job ID", code == 200 and second.endswith("_2"), f"{code} {second} {json.dumps(r)[:160]}")
    code2, r2 = core_post("/api/lessons/study", {"path": L01})
    check("candidate: repeat click reuses the queued job (no new job, no 409)", code2 == 200 and r2.get("receipt") == second, f"{code2} {r2.get('receipt')}")
    l = wait_status(L01, ("PASSED", "RETRY", "MODEL_FAILED", "STUCK"))
    check("candidate: 01 PASSED 3/3, waiting for owner (not LEARNED)", bool(l) and l["status"] == "PASSED" and l["right"] == 3 and not l.get("learned_at"),
          json.dumps({k: (l or {}).get(k) for k in ("status", "right", "job_id", "note")}))
    check("candidate: graded answers shown", [x.get("given") for x in (l or {}).get("results", [])] != [], json.dumps([(x["n"], x.get("given"), x["right"]) for x in (l or {}).get("results", [])]))
    code, r = core_post("/api/lessons/approve", {"path": L01})
    check("candidate: owner approves 01 -> LEARNED", code == 200 and lesson(L01)["status"] == "LEARNED", str(code))
    hist = lesson(L01).get("history") or []
    check("candidate: history = [old failed _1, new passed _2]", [x["job_id"] for x in hist][-2:] == [first, second], json.dumps([(x["job_id"], x["state"], x.get("outcome")) for x in hist]))

    # Restart everything (Engine and Identity too), then check persistence.
    for n in ("core", "engine", "identity"):
        stop(n)
    start("identity", [f"{BIN}/tass-identity"], {"HOME": "/tmp", "IDENTITY_DATA": f"{APP}/data/identity", "IDENTITY_LISTEN": "127.0.0.1:8796",
          "IDENTITY_SERVICE_KEYS": f"{APP}/secrets/service-keys", "IDENTITY_CORE_URL": "http://127.0.0.1:8795/"}, "http://127.0.0.1:8796/v1/health")
    start("engine", [f"{BIN}/thought-engine", "-data", f"{APP}/data/engine", "-model", "http://127.0.0.1:8081", "-worker", "nemotron",
          "-identity-url", "http://127.0.0.1:8796", "-identity-key-file", f"{APP}/secrets/engine.key", "-pulse", "20s"],
          {"PORT": "7798", "HOME": "/tmp"}, "http://127.0.0.1:7798/api/model")
    start_core("tass-core-cand")
    check("after full restart: Identity unlocks (same entity)", bool(unlock()), "")
    ls = lessons()
    check("after restart: 01 and 02 LEARNED, 01 history kept", ls[L01]["status"] == "LEARNED" and ls[L02]["status"] == "LEARNED" and len(ls[L01].get("history") or []) >= 2, "")
    doc = open(f"{APP}/data/core/tass.md").read()
    check("after restart: local tass.md Learned has 01 and 02 exactly once", doc.count("(lesson `" + L01 + "`,") == 1 and doc.count("(lesson `" + L02 + "`,") == 1, "")
    import urllib.request
    gh_doc = lambda: json.loads(urllib.request.urlopen("http://127.0.0.1:28999/fixture/file?path=tass.md").read())["text"]
    got = wait("tass.md synced to GitHub", lambda: "01-my-work-pulse.md" in gh_doc() and "02-how-i-learn.md" in gh_doc(), 90)
    check("fake GitHub tass.md got both Learned lines (next 20 s sync)", bool(got), "")
    jobs = [json.loads(x)["job_id"] for x in open(f"{APP}/data/engine/jobs.jsonl") if x.strip()]
    check("real Engine job store: old _1 kept, _2 added, nothing deleted", first in jobs and second in jobs, f"{len(set(jobs))} distinct jobs")


if __name__ == "__main__":
    main()
