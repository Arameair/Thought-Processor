#!/usr/bin/env python3
"""#82 end to end with REAL binaries (Identity + Thought Engine + Core) and the TEST FIXTURE model /
fake GitHub. Not the owner's machine, not Nemotron.

  Cognition: NOT LINKED → owner links → real cycles; pulse strip PAUSED / 20 s; playback is GET-only.
  Junction:  TASS's reply asks NEED lesson-metadata → Core 20 s tick reads it (Identity grant) →
             next cycle receives it → Cognition shows requested/authorized/returned/consumed →
             Development opens the junction, saves an alternative cell, compares (not activated).

  APP_HOME=/tmp/e2e/app5 BIN=/tmp/e2e/bin SEED=/tmp/e2e/seed CAND=x ALPHA=x OUT=/tmp/e2e/shots-cog \
  python3 core/test/browser/cognition_junction_e2e.py
"""
import json, os, sys, time

sys.path.insert(0, os.path.join(os.path.dirname(os.path.abspath(__file__)), ".."))
import lesson_retry_native as n  # noqa: E402
from acceptance import CORE, check, core_post, idn_post, results, wait, http, token  # noqa: E402
from playwright.sync_api import sync_playwright  # noqa: E402

OUT = os.environ["OUT"]
os.makedirs(OUT, exist_ok=True)
L01 = "lessons/claude/01-my-work-pulse.md"


def wait_text(pg, sel, text, secs=20):
    end = time.time() + secs
    while time.time() < end:
        try:
            if text.lower() in pg.inner_text(sel).lower():
                return True
        except Exception:
            pass
        time.sleep(0.3)
    raise TimeoutError(f"{sel} never contained {text!r}")


def jget():
    return http("GET", CORE + "/api/junctions", headers={"X-TASS-Token": token(CORE)})[1]


def say(text):
    core_post("/api/say", {"text": text})


def main():
    n.prepare()
    n.start("fake-github", ["python3", f"{n.HERE}/fake_github.py", "28999", "Mira/home", "live", n.SEED], {}, "http://127.0.0.1:28999/fixture/log")
    n.start("model", ["python3", f"{n.HERE}/fixture-model/fake_llama.py", "8081"], {}, "http://127.0.0.1:8081/health")
    n.start("identity", [f"{n.BIN}/tass-identity-jn"], {"HOME": "/tmp", "IDENTITY_DATA": f"{n.APP}/data/identity", "IDENTITY_LISTEN": "127.0.0.1:8796",
            "IDENTITY_SERVICE_KEYS": f"{n.APP}/secrets/service-keys", "IDENTITY_CORE_URL": "http://127.0.0.1:8795/"}, "http://127.0.0.1:8796/v1/health")
    n.start("engine", [f"{n.BIN}/thought-engine-cog", "-data", f"{n.APP}/data/engine", "-model", "http://127.0.0.1:8081", "-worker", "nemotron",
            "-identity-url", "http://127.0.0.1:8796", "-identity-key-file", f"{n.APP}/secrets/engine.key", "-pulse", "20s"],
            {"PORT": "7798", "HOME": "/tmp"}, "http://127.0.0.1:7798/api/model")
    n.start_core("tass-core-cog")
    posts = []
    try:
        idn_post("/v1/entity", {"name": "TASS", "passphrase": n.PASS})
        wait("tass.md", lambda: os.path.exists(f"{n.APP}/data/core/tass.md"), 60)
        core_post("/api/lessons/discover", {})
        say("hello TASS")
        wait("first cycle", lambda: "Hello" in open(f"{n.APP}/data/core/conversation.jsonl").read() if os.path.exists(f"{n.APP}/data/core/conversation.jsonl") else False, 60)
        with sync_playwright() as p:
            b = p.chromium.launch(executable_path="/usr/bin/google-chrome", args=["--no-sandbox"])
            pg = b.new_page(viewport={"width": 1400, "height": 1000})
            pg.on("request", lambda r: posts.append(r.method + " " + r.url) if r.method != "GET" else None)
            pg.goto(CORE + "/cognition")
            wait_text(pg, "#linkbox", "NOT LINKED")
            check("Cognition: Engine history NOT LINKED by default; same name is only a hint", "same name (a hint, not proof)" in pg.inner_text("#linkbox") and pg.locator(".tile", has_text="Cycle tp_").count() == 0, "")
            wait_text(pg, "#pulse", "PAUSED")
            check("pulse strip: real Engine pulse PAUSED, every 20 s; opening the page started nothing", "every 20 s" in pg.inner_text("#pulse").lower(), pg.inner_text("#pulse")[:160].replace("\n", " "))
            pg.screenshot(path=f"{OUT}/1-not-linked-pulse.png")
            posts.clear()
            pg.locator("#linkbox button", has_text="Link to this Identity").first.click()
            wait_text(pg, "#linkbox", "Engine history link: LINKED")
            wait_text(pg, "#shelves", "Cycle tp_")
            check("owner links → real recorded cycles appear", pg.locator(".tile", has_text="Cycle tp_").count() >= 1, str(pg.locator(".tile", has_text="Cycle tp_").count()))
            check("linking was the only POST", len(posts) == 1 and "/api/cognition/link" in posts[0], str(posts))
            posts.clear()
            pg.locator(".tile", has_text="Cycle tp_").first.click()
            pg.click("#logs"); wait_text(pg, "#thoughtDetail", "View logs")
            pg.click("#step"); pg.click("#play"); time.sleep(2.5); pg.click("#close")
            check("playback / view logs / step made zero non-GET requests", posts == [], str(posts))

            # Junction: grant on Identity, TASS asks, Core tick reads, next cycle consumes.
            idn_post("/v1/grants/skill", {"skill": "junction.lesson-metadata", "passphrase": n.PASS})
            say("please check the lesson-metadata of your pulse lesson")
            wait("junction requested", lambda: any(r["origin"] == "TASS reply" for r in jget()["receipts"]), 60)
            r = [x for x in jget()["receipts"] if x["origin"] == "TASS reply"][0]
            check("TASS's reply recorded a junction request with its cycle", r["status"] in ("PENDING", "DATA") and r["origin_cycle_id"].startswith("tp_"), json.dumps({k: r[k] for k in ("junction_id", "status", "origin_cycle_id")}))
            wait("Core 20 s tick reads it", lambda: [x for x in jget()["receipts"] if x["junction_id"] == r["junction_id"]][0]["status"] == "DATA", 45)
            r = [x for x in jget()["receipts"] if x["junction_id"] == r["junction_id"]][0]
            check("Core tick: Identity allowed junction.read, handler returned DATA", r["status"] == "DATA" and "dec_" in r["identity_decision"] and "title: My work pulse" in r["data"] and r["resolved_by"] == "Core 20 s sync tick", r["identity_decision"][:60])
            say("what did you learn?")
            wait("consumed", lambda: [x for x in jget()["receipts"] if x["junction_id"] == r["junction_id"]][0].get("consumed_by_cycle"), 60)
            r = [x for x in jget()["receipts"] if x["junction_id"] == r["junction_id"]][0]
            convo = open(f"{n.APP}/data/core/conversation.jsonl").read()
            check("a later real Engine cycle received it; TASS's reply used it", r["consumed_by_cycle"] in convo and "gave me the lesson metadata I asked for: My work pulse" in convo, r["consumed_by_cycle"])
            pg.goto(CORE + "/cognition")
            wait_text(pg, "#shelves", "Information junctions")
            pg.locator(".tile", has_text="Junction: lesson metadata").first.click()
            wait_text(pg, "#fields", "4 · Consumed")
            f = pg.inner_text("#fields").lower()
            check("Cognition junction card: requested → authorized → returned → consumed → effect NOT MEASURED", all(x.lower() in f for x in ("1 · Requested", "Identity dec_", "DATA at", "by cycle " + r["consumed_by_cycle"], "EFFECT NOT MEASURED")), "")
            pg.screenshot(path=f"{OUT}/2-junction-card.png")
            pg.click("#tryDev")
            wait_text(pg, "#cognitionProposal", "UNEXECUTED DEVELOPMENT PROPOSAL")
            wait_text(pg, "#jReceipts", r["junction_id"])
            posts.clear()
            pg.fill("#jProp", pg.input_value("#jProp").replace("fields: title, goal, checks, status", "fields: title, steps, sha"))
            pg.fill("#jNote", "steps instead of checks")
            pg.click("#jSave"); wait_text(pg, "#jProps", "PROPOSED")
            pg.select_option("#jLesson", L01)
            pg.locator("#jProps button", has_text="Compare").first.click()
            wait_text(pg, "#jCmp", "Different output")
            check("Development: alternative cell saved as PROPOSED and compared; effect NOT MEASURED", "not measured" in pg.inner_text("#jCmp").lower() and "steps:" in pg.inner_text("#jCmp"), "")
            check("the active cell did not change", jget()["active_cell"]["version"] == r["cell_version"], "")
            pg.screenshot(path=f"{OUT}/3-dev-junction-lab.png", full_page=False)
            b.close()
    finally:
        for p in list(n.procs):
            n.stop(p)
    for name, res, detail in results:
        print(f"{res:4}  {name}  {detail}")
    fails = [r for r in results if r[1] != "PASS"]
    print(f"\n{len(results) - len(fails)}/{len(results)} PASS")
    sys.exit(1 if fails else 0)


if __name__ == "__main__":
    main()
