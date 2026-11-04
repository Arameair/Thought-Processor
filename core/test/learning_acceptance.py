#!/usr/bin/env python3
"""Clean-host run of the lessons loop: GitHub → pulse → study → grade → approve → Learned on GitHub.

TEST FIXTURES only: core/test/fake_github.py stands in for api.github.com and the fixture model answers
lesson checks from a lookup table. Run on an isolated Docker host:

    APP_HOME=/tmp/ch/app3 FIXTURE_MODEL_IMAGE=tass-fixture-model:local GITHUB_TOKEN=fixture-token \
    TASS_GITHUB_API=http://172.17.0.1:28999 python3 core/test/learning_acceptance.py
"""
import json, os, subprocess, sys, time, urllib.request

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from acceptance import (CORE, IDN, APP_HOME, check, core_post, docker, health, http, idn_post, log, results, sh,  # noqa: E402
                        status, wait)

GH = os.environ["TASS_GITHUB_API"].replace("172.17.0.1", "127.0.0.1")
PASS = "correct-horse-battery-staple-42"


def gh_file(p):
    return json.loads(urllib.request.urlopen(f"{GH}/fixture/file?path={p}").read())["text"]


def gh_log():
    return json.loads(urllib.request.urlopen(f"{GH}/fixture/log").read())


def gh_put(p, text):
    r = urllib.request.Request(f"{GH}/fixture/put", data=json.dumps({"path": p, "text": text}).encode(),
                               headers={"Content-Type": "application/json"}, method="POST")
    urllib.request.urlopen(r).read()


def lessons():
    return {l["path"]: l for l in (status().get("learning") or {}).get("lessons") or []}


def main():
    env = os.path.join(APP_HOME, "app.env")
    sh("./tass.sh", "check", ok_codes=(0, 1))       # writes app.env the first time
    txt = open(env).read().replace("GITHUB_REPO=            # e.g. Arameair/TASS", "GITHUB_REPO=Mira/home")
    open(env, "w").write(txt)
    sh("./tass.sh", "check")
    sh("./tass.sh", "start")
    tok = open(f"{APP_HOME}/secrets/github_token").read().strip()
    check("GitHub token made at start, 0600, mounted read-only", tok == "fixture-token" and oct(os.stat(f"{APP_HOME}/secrets/github_token").st_mode)[-3:] == "600", "")
    leaked = subprocess.run(f"docker save {docker('inspect', 'tass-app-core-1', '--format', '{{.Config.Image}}')} | grep -a -c fixture-token",
                            shell=True, capture_output=True, text=True).stdout.strip()
    check("the token is not in the core image", leaked in ("", "0"), leaked or "0")

    check("before an entity exists, nothing syncs with GitHub", not any("contents" in r for r in gh_log()["requests"]), f"{gh_log()['count']} requests")
    code, r = idn_post("/v1/entity", {"name": "Mira", "passphrase": PASS})
    check("owner creates the entity (UNLOCKED)", code == 201 and r["state"] == "UNLOCKED", str(code))

    doc = wait("doc", lambda: os.path.exists(f"{APP_HOME}/data/core/tass.md") and open(f"{APP_HOME}/data/core/tass.md").read(), 60)
    check("connected to GitHub: the entity's document came from the repo", bool(doc) and doc.startswith("# Mira"), (doc or "").splitlines()[0] if doc else "none")
    code, r = core_post("/api/say", {"text": "hi"})
    cy = wait("chat", lambda: [c for c in status().get("cycles") or [] if c.get("request") == "hi" and c.get("evaluation")], 90)
    check("chat with the entity (receipt + answered)", code == 202 and bool(cy), r.get("receipt", ""))

    time.sleep(25)
    check("pulse setting off: no lessons looked at, pulse PAUSED",
          not any("git/trees" in x for x in gh_log()["requests"]) and status()["pulse"]["control"] == "PAUSED", status()["pulse"]["control"])

    code, r = core_post("/api/settings", {"pulse": True, "learn": True})
    check("turn the pulse setting ON", code == 200 and r["pulse"], json.dumps(r))
    ok = wait("RUNNING", lambda: status()["pulse"]["control"] == "RUNNING", 30)
    check("the engine pulse is RUNNING (a tick every 20 s)", bool(ok), status()["pulse"]["control"])
    found = wait("lessons", lambda: (lambda ls: ls if len(ls) >= 3 else None)(lessons()), 60)
    check("a pulse found the 3 lessons on GitHub", bool(found), ", ".join(sorted(lessons())))
    t0 = time.time()
    done = wait("all passed", lambda: (lambda ls: ls if ls and all(l["status"] == "PASSED" for l in ls.values()) else None)(lessons()), 600, 5)
    ls = lessons()
    for p, l in sorted(ls.items()):
        log(f"   {l['status']:7} {p}  {l['right']}/{l['checks']} attempts {l['attempts']}  remembers: {l.get('remember', '')}")
        for x in l.get("results") or []:
            log(f"            {'✓' if x['right'] else '✗'} {x['check']} → {x['given']}")
    check("each lesson was studied on the pulse and graded by the app: all 3 PASSED", bool(done), f"in {time.time() - t0:.0f} s")
    retry = ls.get("lessons/claude/02-how-i-learn.md", {})
    check("a wrong answer made the entity try again (attempt 2 passed)", retry.get("attempts") == 2, f"attempts {retry.get('attempts')}")
    jobs = status()["pulse"]["jobs"]
    check("one lesson at a time, each job started once", len({j["job_id"] for j in jobs if j["job_id"].startswith("lesson_")}) == 4, f"{len(jobs)} jobs")
    check("nothing counts as learned before approval", "## Learned" not in gh_file("tass.md"), "")

    for p in sorted(ls):
        code, _ = core_post("/api/lessons/approve", {"path": p})
        check(f"owner approves {p.split('/')[-1]}", code == 200, str(code))
    remote = wait("synced", lambda: (lambda t: t if t.count("(lesson `lessons/") == 3 else None)(gh_file("tass.md")), 60)
    log("tass.md on GitHub now ends:\n    " + "\n    ".join((remote or gh_file("tass.md")).rstrip().splitlines()[-5:]))
    check("the 3 learned lessons are in the Learned section on GitHub", bool(remote), "")

    sh("./tass.sh", "stop")
    sh("./tass.sh", "start")
    check("after restart: LOCKED, nothing runs", health()["state"] == "BLOCKED", health()["state"])
    idn_post("/v1/unlock", {"passphrase": PASS})
    ok = wait("RUNNING again", lambda: status()["pulse"]["control"] == "RUNNING", 45)
    check("after unlock the pulse comes back ON by itself (setting kept)", bool(ok), status()["pulse"]["control"])
    ls = lessons()
    check("learned lessons are remembered, not studied again", all(l["status"] == "LEARNED" for l in ls.values()) and len(ls) == 3, "")

    gh_put("lessons/claude/04-new.md", "# A new thing\nby: Dustin\n\nAnswer a question.\n\n## Step 1: Colors\nTeach: The sky is blue.\nCheck: What color is the sky?\nAnswer: blue\n")
    new = wait("new lesson", lambda: (lambda l: l if l and l["status"] == "PASSED" else None)(lessons().get("lessons/claude/04-new.md")), 180, 5)
    check("a lesson added to GitHub later is found on a pulse and studied", bool(new), (new or {}).get("status", "not found"))
    trees = [x for x in gh_log()["requests"] if "git/trees" in x]
    check("GitHub is checked cheaply (304 Not Modified when nothing changed)", gh_log()["requests"].count("  -> 304") > 0, f"{len(trees)} tree requests")
    sh("./tass.sh", "status")

    print("\nSUMMARY")
    for n, r, d in results:
        print(f"  {r:4}  {n}")
    fails = [n for n, r, _ in results if r != "PASS"]
    print(f"\n{len(results) - len(fails)} PASS · {len(fails)} FAIL")
    sys.exit(1 if fails else 0)


if __name__ == "__main__":
    main()
