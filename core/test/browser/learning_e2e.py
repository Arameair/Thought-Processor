#!/usr/bin/env python3
"""Gate F (P2/P3) end to end in the browser with the REAL Go services, TEST FIXTURE model, fake GitHub.
NOT the owner's machine and NOT Nemotron: the fixture answers lesson checks from a lookup table, so
this proves the plumbing (provenance, grading, approval, persistence, retest, evidence), not learning.

  Development "Teach & test": choose lesson -> Study (MODEL_FAILED, not graded) -> Retry -> Study
  (new job ID, no 409) -> inspect answer -> grade 3/3 PASSED (not learned) -> Approve (LEARNED) ->
  Test again on NEW questions (lesson text withheld) -> restart every service -> evidence persists ->
  Test again after restart -> Cognition lesson card shows source SHA, attempts with job+cycle IDs,
  grade, approval, retests, "no skill from learning". Then #78 in the brain editor.

  APP_HOME=/tmp/e2e/learn BIN=/tmp/e2e/bin SEED=/tmp/e2e/seed CAND=x ALPHA=x OUT=/tmp/e2e/shots-learn \
  CORE_BIN=tass-core-int python3 core/test/browser/learning_e2e.py
"""
import json, os, sys, time

sys.path.insert(0, os.path.join(os.path.dirname(os.path.abspath(__file__)), ".."))
os.environ.setdefault("CAND", "x"); os.environ.setdefault("ALPHA", "x")
import lesson_retry_native as n  # noqa: E402
from acceptance import CORE, check, core_post, http, idn_post, results, token, wait  # noqa: E402
from playwright.sync_api import sync_playwright  # noqa: E402

OUT = os.environ["OUT"]; os.makedirs(OUT, exist_ok=True)
CORE_BIN = os.environ.get("CORE_BIN", "tass-core-int")
L01 = "lessons/claude/01-my-work-pulse.md"


def wait_text(pg, sel, text, secs=60):
    end = time.time() + secs
    while time.time() < end:
        try:
            if text.lower() in pg.inner_text(sel).lower():
                return True
        except Exception:
            pass
        time.sleep(0.4)
    raise TimeoutError(f"{sel} never contained {text!r}: {pg.inner_text(sel)[:300]!r}")


def lesson():
    st = http("GET", CORE + "/api/status", headers={"X-TASS-Token": token(CORE)})[1]
    return next((l for l in st["learning"]["lessons"] if l["path"] == L01), {})


def services():
    n.start("identity", [f"{n.BIN}/tass-identity-jn"], {"HOME": "/tmp", "IDENTITY_DATA": f"{n.APP}/data/identity", "IDENTITY_LISTEN": "127.0.0.1:8796",
            "IDENTITY_SERVICE_KEYS": f"{n.APP}/secrets/service-keys", "IDENTITY_CORE_URL": "http://127.0.0.1:8795/"}, "http://127.0.0.1:8796/v1/health")
    n.start("engine", [f"{n.BIN}/thought-engine-cog", "-data", f"{n.APP}/data/engine", "-model", "http://127.0.0.1:8081", "-worker", "nemotron",
            "-identity-url", "http://127.0.0.1:8796", "-identity-key-file", f"{n.APP}/secrets/engine.key", "-pulse", "20s"],
            {"PORT": "7798", "HOME": "/tmp"}, "http://127.0.0.1:7798/api/model")
    n.start_core(CORE_BIN)


def retest(pg, lines, want):
    pg.fill("#lQs", "\n".join(lines))
    pg.click("#lRetest")
    wait_text(pg, "#lMsg", "retest OK", 20)
    before = len(lesson().get("retests") or [])
    got = wait("retest graded", lambda: len(lesson().get("retests") or []) > before and lesson()["retests"][-1], 150, 2)
    pg.click("#lReload"); wait_text(pg, "#lEvidence", want, 20)
    return got


def main():
    n.prepare()
    n.start("fake-github", ["python3", f"{n.HERE}/fake_github.py", "28999", "Mira/home", "live", n.SEED], {}, "http://127.0.0.1:28999/fixture/log")
    n.start("model", ["python3", f"{n.HERE}/fixture-model/fake_llama.py", "8081"], {"FIXTURE_LESSON_FAIL_FIRST": "My work pulse"}, "http://127.0.0.1:8081/health")
    services()
    posts = []
    try:
        idn_post("/v1/entity", {"name": "TASS", "passphrase": n.PASS})
        wait("tass.md", lambda: os.path.exists(f"{n.APP}/data/core/tass.md"), 60)
        core_post("/api/lessons/discover", {})
        with sync_playwright() as p:
            b = p.chromium.launch(executable_path="/usr/bin/google-chrome", args=["--no-sandbox"])
            pg = b.new_page(viewport={"width": 1500, "height": 1100})
            pg.on("request", lambda r: posts.append(r.method + " " + r.url.split("8795")[-1]) if r.method != "GET" else None)
            pg.goto(CORE + "/dev?learn=" + L01)
            wait_text(pg, "#lCard", "NEW")
            check("1 Choose: lesson from GitHub with its Git blob revision", lesson().get("sha", "")[:12] in pg.inner_text("#lCard"), pg.inner_text("#lCard")[:200].replace("\n", " "))
            check("pulse shown PAUSED, Core sync described as separate", "PAUSED" in pg.inner_text("#lPulse") and "separate" in pg.inner_text("#lPulse"), pg.inner_text("#lPulse"))
            posts.clear()
            pg.click("#lStudy")
            wait("first attempt done", lambda: lesson().get("status") in ("MODEL_FAILED", "PASSED", "RETRY"), 150, 2)
            pg.click("#lReload"); wait_text(pg, "#lCard", "MODEL_FAILED")
            first = lesson()["history"][-1]["job_id"]
            check("2 Study: real Engine job; REMEMBER-only reply is MODEL_FAILED and NOT GRADED", "NOT GRADED" in pg.inner_text("#lGrade"), pg.inner_text("#lGrade")[:160])
            pg.click("#lRetry"); wait_text(pg, "#lMsg", "retry OK")
            pg.click("#lStudy"); wait_text(pg, "#lMsg", "study OK", 20)
            second = lesson().get("job_id")
            check("Retry -> Study uses a NEW job ID (no 409, #71)", second and second != first and second.endswith("_2"), f"{first} -> {second}")
            wait("graded", lambda: lesson().get("status") in ("PASSED", "RETRY", "STUCK", "MODEL_FAILED"), 150, 2)
            pg.click("#lReload"); wait_text(pg, "#lGrade", "PASSED")
            check("3 Inspect answer: raw Engine reply shown", "ANSWER 1" in pg.inner_text("#lReply"), pg.inner_text("#lReply")[:120].replace("\n", " | "))
            check("4 Grade: 3/3 PASSED, waiting for approval, NOT learned", "3/3" in pg.inner_text("#lGrade") and "not learned yet" in pg.inner_text("#lGrade") and not lesson().get("learned_at"), "")
            check("Test again disabled before approval (PASSED is not LEARNED)", pg.is_disabled("#lRetest"), "")
            pg.screenshot(path=f"{OUT}/1-passed.png")
            pg.click("#lApprove"); wait_text(pg, "#lCard", "LEARNED")
            check("5 Approve: LEARNED with the approval time", lesson().get("status") == "LEARNED" and "Approved" in pg.inner_text("#lEvidence"), "")
            r1 = retest(pg, ["If Identity is locked when my pulse ticks, what do I report? => blocked",
                             "What color is a stop sign? => red"], "RETEST_FAILED")
            check("6 Test again (owner's new questions): graded per question, 1/2 RETEST_FAILED, still LEARNED",
                  r1 and r1["outcome"] == "RETEST_FAILED" and r1["right"] == 1 and lesson()["status"] == "LEARNED", json.dumps({k: r1.get(k) for k in ("job_id", "cycle_id", "outcome", "right", "checks")}))
            check("retest context is approved knowledge only (lesson text withheld)", "lesson text withheld" in r1["context"], r1["context"])
            pg.screenshot(path=f"{OUT}/2-retest-failed.png")

            # restart every service, then a fresh-task retest in a new session
            b.close()
            for s in ("core", "engine", "identity"):
                n.stop(s)
            services()
            n.unlock()
            b = p.chromium.launch(executable_path="/usr/bin/google-chrome", args=["--no-sandbox"])
            pg = b.new_page(viewport={"width": 1500, "height": 1100})
            pg.on("request", lambda r: posts.append(r.method + " " + r.url.split("8795")[-1]) if r.method != "GET" else None)
            pg.goto(CORE + "/dev?learn=" + L01)
            wait_text(pg, "#lCard", "LEARNED")
            ev = pg.inner_text("#lEvidence")
            check("after full restart: LEARNED, both attempts, approval and retest kept", first in ev and second in ev and "RETEST_FAILED" in ev and "Approved" in ev, "")
            r2 = retest(pg, ["If Identity is locked at a tick, which state do I show? => blocked"], "RETEST_PASSED")
            check("6 after restart: fresh question RETEST_PASSED 1/1 with its own job + cycle ID",
                  r2 and r2["outcome"] == "RETEST_PASSED" and r2["job_id"] != r1["job_id"] and r2.get("cycle_id", "").startswith(r2["job_id"]),
                  json.dumps({k: r2.get(k) for k in ("job_id", "cycle_id", "outcome")}))
            check("7 Evidence: skill receipts separate; learning granted no skill", "never grants or runs a skill" in pg.inner_text("#lEvidence"), "")
            pg.screenshot(path=f"{OUT}/3-evidence-after-restart.png", full_page=True)
            check("only lesson POSTs were made (no settings, pulse or Identity change)",
                  all(x.split(" ")[1].startswith("/api/lessons/") for x in posts), json.dumps(posts))

            # Cognition lesson card
            pg.goto(CORE + "/cognition")
            wait_text(pg, "#linkbox", "LINKED")
            if "NOT LINKED" in pg.inner_text("#linkbox"):
                pg.locator("#linkbox button", has_text="Link to this Identity").first.click()
                wait_text(pg, "#linkbox", "Engine history link: LINKED")
            pg.click('button[data-filter="lesson"]')
            wait_text(pg, "#shelves", "My work pulse")
            pg.locator(".tile", has_text="My work pulse").first.click()
            wait_text(pg, "#fields", "Fresh-task retests")
            f = pg.inner_text("#fields")
            check("Cognition lesson card: source SHA, attempts with job and cycle IDs", lesson()["sha"] in f and first in f and second in f and f"cycle {second}:cycle" in f, "")
            check("Cognition: grade, owner approval, retests, no skill from learning", "3/3 graded by Core" in f and "owner approved" in f and "RETEST_PASSED" in f and "RETEST_FAILED" in f and "never grants or runs a skill" in f, "")
            check("Cognition: the failed attempt's missing grade is not invented", "MODEL_FAILED" in f, "")
            pg.screenshot(path=f"{OUT}/4-cognition-lesson.png")

            # #78 in the editor: refuse uses->non-skill and a next cycle, before the graph changes
            pg.goto(CORE + "/dev")
            wait_text(pg, "#lib", "Add to brain")
            pg.select_option("#kind", "lesson")
            pg.locator("#lib .item", has_text="My work pulse").locator("button").click()
            pg.locator("#lib .item", has_text="How I learn").locator("button").click()
            edges = lambda: pg.locator("path.edge").count()
            pg.locator(".node", has_text="My work pulse").click()
            pg.select_option("#cRel", "uses"); pg.select_option("#cTo", label="How I learn")
            pg.locator("#insp button", has_text="Connect").first.click()
            check("#78 editor: 'uses' to a lesson refused with its reason, nothing added",
                  "must point to a skill" in pg.inner_text("#report") and edges() == 0, pg.inner_text("#report"))
            pg.select_option("#cRel", "next"); pg.select_option("#cTo", label="How I learn")
            pg.locator("#insp button", has_text="Connect").first.click()
            check("valid next edge is added", edges() == 1, pg.inner_text("#report"))
            pg.locator(".node", has_text="How I learn").click()
            pg.select_option("#cRel", "next"); pg.select_option("#cTo", label="My work pulse")
            pg.locator("#insp button", has_text="Connect").first.click()
            check("#78 editor: reverse 'next' (a circle) refused, nothing added", "circle" in pg.inner_text("#report") and edges() == 1, pg.inner_text("#report"))
            pg.locator("path.hit").first.click(force=True)
            pg.select_option("#eRel", "uses")
            check("#78 editor: changing a connection to 'uses' on a lesson refused", "must point to a skill" in pg.inner_text("#report") and pg.input_value("#eRel") == "next", pg.inner_text("#report"))
            pg.screenshot(path=f"{OUT}/5-brain-refusals.png")
            b.close()
    finally:
        for s in list(n.procs):
            n.stop(s)
    print()
    for name, res, detail in results:
        print(f"{res:4}  {name}  {detail}")
    fails = [r for r in results if r[1] != "PASS"]
    print(f"\n{len(results) - len(fails)}/{len(results)} PASS")
    sys.exit(1 if fails else 0)


if __name__ == "__main__":
    main()
