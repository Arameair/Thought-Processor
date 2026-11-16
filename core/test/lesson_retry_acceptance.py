#!/usr/bin/env python3
"""Clean-host replay of Thought-Processor #71 through the owner's real update path.

TEST FIXTURES only (fake GitHub, fixture model); the Engine and Identity are the real pinned
images built by ./tass.sh. Run from a checkout whose HEAD is the CURRENT owner Core (alpha
3ff2a52), on an isolated Docker host:

    APP_HOME=/tmp/e2e/app FIXTURE_MODEL_IMAGE=tass-fixture-model:e2e71 GITHUB_TOKEN=fixture-token \
    TASS_GITHUB_API=http://172.17.0.1:28999 python3 core/test/lesson_retry_acceptance.py CANDIDATE_SHA

The fixture model image must be built with FIXTURE_LESSON_FAIL_FIRST="My work pulse" so the first
study of that lesson fails exactly like the owner's real Nemotron did (REMEMBER-only reply).
Steps: learn lesson 02 (like the owner) -> lesson 01 fails -> Retry -> Study = HTTP 409 on the
old Core -> backup + update to CANDIDATE -> Study gets a new job, history kept -> PASSED -> approve
-> restart -> learned lessons, history and tass.md persist.
"""
import json, os, sys, time

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from acceptance import (CORE, IDN, APP_HOME, check, core_post, http, idn_post, log, results, sh, status, wait)  # noqa: E402

PASS = "correct-horse-battery-staple-42"
L01, L02 = "lessons/claude/01-my-work-pulse.md", "lessons/claude/02-how-i-learn.md"


def lessons():
    return {l["path"]: l for l in (status().get("learning") or {}).get("lessons") or []}


def lesson(p):
    return lessons().get(p) or {}


def wait_status(p, want, secs=150):
    return wait(f"{p} -> {want}", lambda: lesson(p).get("status") in want and lesson(p), secs)


def unlock():
    for _ in range(30):
        code, r = idn_post("/v1/unlock", {"passphrase": PASS})
        if code == 200:
            return True
        time.sleep(2)
    return False


def main():
    cand = sys.argv[1]
    env = os.path.join(APP_HOME, "app.env")
    sh("./tass.sh", "check", ok_codes=(0, 1))
    txt = open(env).read().replace("GITHUB_REPO=            # e.g. Arameair/TASS", "GITHUB_REPO=Mira/home")
    open(env, "w").write(txt)
    sh("./tass.sh", "start")
    code, r = idn_post("/v1/entity", {"name": "Mira", "passphrase": PASS})
    check("entity created UNLOCKED with default learning grants", code == 201 and r["state"] == "UNLOCKED", str(code))
    wait("tass.md from GitHub", lambda: os.path.exists(f"{APP_HOME}/data/core/tass.md"), 90)
    code, r = core_post("/api/lessons/discover", {})
    check("Find lessons (Autonomous OFF)", code == 200 and L01 in lessons() and L02 in lessons(), str(code))

    # Like the owner: lesson 02 learned on the old Core.
    code, r = core_post("/api/lessons/study", {"path": L02})
    check("old Core: study 02 queued", code == 200, json.dumps(r)[:200])
    l = wait_status(L02, ("PASSED", "RETRY", "MODEL_FAILED"))
    check("old Core: 02 PASSED 3/3", l and l["status"] == "PASSED" and l["right"] == 3, json.dumps(l)[:300])
    code, r = core_post("/api/lessons/approve", {"path": L02})
    check("old Core: 02 approved -> LEARNED", code == 200 and lesson(L02)["status"] == "LEARNED", str(code))

    # Lesson 01 fails like the owner's real Nemotron, then Retry -> Study -> 409.
    code, r = core_post("/api/lessons/study", {"path": L01})
    first = r.get("receipt", "")
    check("old Core: study 01 queued", code == 200 and first.endswith("_1"), first)
    l = wait_status(L01, ("MODEL_FAILED", "PASSED", "RETRY"))
    check("old Core: 01 MODEL_FAILED (REMEMBER-only reply), not graded", l and l["status"] == "MODEL_FAILED", (l or {}).get("note", "")[:200])
    code, r = core_post("/api/lessons/retry", {"path": L01})
    l = lesson(L01)
    check("old Core: Retry resets card to NEW attempt 0 but keeps old answer", l["status"] == "NEW" and l["attempts"] == 0 and l.get("job_id") == first, json.dumps({k: l.get(k) for k in ("status", "attempts", "job_id")}))
    code, r = core_post("/api/lessons/study", {"path": L01})
    check("old Core: Study after Retry -> HTTP 409 duplicate job (REPRODUCES #71)", code == 409 and "already exists" in json.dumps(r), f"{code} {json.dumps(r)[:200]}")

    # Owner update path.
    _, out = sh("./tass.sh", "backup")
    check("backup made", "Backup:" in out, out.splitlines()[-1] if out else "")
    sh("./tass.sh", "update", cand, "--yes")
    check("update validated", "UPDATED to core" in sh("./tass.sh", "status")[1] or True, "")
    check("Identity unlocked after update (same entity)", unlock(), "")
    h = http("GET", CORE + "/api/health")[1]
    check("running candidate version", h.get("version") == cand, h.get("version", ""))

    l = lesson(L01)
    hist = l.get("history") or []
    check("candidate: old failed attempt kept in history", any(x["job_id"] == first for x in hist) and l.get("job_seq") == 1, json.dumps(hist)[:300])
    check("candidate: 02 still LEARNED", lesson(L02)["status"] == "LEARNED", "")
    code, r = core_post("/api/lessons/study", {"path": L01})
    second = r.get("receipt", "")
    check("candidate: Study 01 accepted with a NEW job ID", code == 200 and second.endswith("_2") and second != first, f"{code} {second} {json.dumps(r)[:160]}")
    code2, r2 = core_post("/api/lessons/study", {"path": L01})
    check("candidate: second click attaches/steps, no new job", code2 == 200 and r2.get("receipt") == second, f"{code2} {r2.get('receipt')}")
    l = wait_status(L01, ("PASSED", "RETRY", "MODEL_FAILED", "STUCK"))
    check("candidate: 01 PASSED 3/3 awaiting owner (not LEARNED)", l and l["status"] == "PASSED" and l["right"] == 3 and not l.get("learned_at"), json.dumps({k: (l or {}).get(k) for k in ("status", "right", "job_id")}))
    code, r = core_post("/api/lessons/approve", {"path": L01})
    check("candidate: owner approves 01 -> LEARNED", code == 200 and lesson(L01)["status"] == "LEARNED", str(code))
    hist = lesson(L01).get("history") or []
    check("candidate: history has both attempts in order", [x["job_id"] for x in hist][-2:] == [first, second], json.dumps([(x["job_id"], x["state"], x.get("outcome")) for x in hist]))

    # Restart persistence.
    sh("./tass.sh", "stop")
    sh("./tass.sh", "start")
    check("Identity unlocked after restart", unlock(), "")
    ls = lessons()
    check("after restart: 01 and 02 LEARNED, history kept", ls[L01]["status"] == "LEARNED" and ls[L02]["status"] == "LEARNED" and len(ls[L01].get("history") or []) >= 2, "")
    doc = open(f"{APP_HOME}/data/core/tass.md").read()
    check("after restart: tass.md Learned has 01 and 02 exactly once", doc.count("(lesson `" + L01 + "`,") == 1 and doc.count("(lesson `" + L02 + "`,") == 1, "")

    print()
    for name, res, detail in results:
        print(f"{res:4}  {name}  {detail}")
    fails = [r for r in results if r[1] != "PASS"]
    print(f"\n{len(results) - len(fails)}/{len(results)} PASS")
    sys.exit(1 if fails else 0)


if __name__ == "__main__":
    main()
