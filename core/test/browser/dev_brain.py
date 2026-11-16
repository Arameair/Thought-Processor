#!/usr/bin/env python3
"""Drive the Development page (/dev) like the owner (TEST FIXTURE model and fake GitHub; real
Identity, Engine and Core binaries). Seed with real TASS lessons, including the notebook.

    APP_HOME=/tmp/e2e/app4 BIN=/tmp/e2e/bin SEED=/tmp/e2e/seed CAND=x ALPHA=x OUT=/tmp/e2e/shots-dev \
    IDENTITY_BIN=tass-identity-new CORE_BIN=tass-core-dev python3 core/test/browser/dev_brain.py
"""
import json, os, sys, time

sys.path.insert(0, os.path.join(os.path.dirname(os.path.abspath(__file__)), ".."))
import lesson_retry_native as n  # noqa: E402
from acceptance import CORE, check, core_post, idn_post, results, wait, http, token  # noqa: E402
from playwright.sync_api import sync_playwright  # noqa: E402

OUT = os.environ["OUT"]
os.makedirs(OUT, exist_ok=True)


def wait_text(pg, sel, text, secs=15):
    end = time.time() + secs
    while time.time() < end:
        try:
            if text in pg.inner_text(sel):
                return True
        except Exception:
            pass
        time.sleep(0.3)
    raise TimeoutError(f"{sel} never contained {text!r}")


def brain():
    return http("GET", CORE + "/api/brain", headers={"X-TASS-Token": token(CORE)})[1]


def add_block(pg, title):
    pg.fill("#q", title)
    pg.locator("#lib .item", has_text=title).first.locator("button").click()
    pg.fill("#q", "")


def main():
    n.prepare()
    n.start("fake-github", ["python3", f"{n.HERE}/fake_github.py", "28999", "Mira/home", "live", n.SEED], {}, "http://127.0.0.1:28999/fixture/log")
    n.start("model", ["python3", f"{n.HERE}/fixture-model/fake_llama.py", "8081"], {}, "http://127.0.0.1:8081/health")
    n.start("identity", [f"{n.BIN}/{os.environ['IDENTITY_BIN']}"], {"HOME": "/tmp", "IDENTITY_DATA": f"{n.APP}/data/identity", "IDENTITY_LISTEN": "127.0.0.1:8796",
            "IDENTITY_SERVICE_KEYS": f"{n.APP}/secrets/service-keys", "IDENTITY_CORE_URL": "http://127.0.0.1:8795/"}, "http://127.0.0.1:8796/v1/health")
    n.start("engine", [f"{n.BIN}/thought-engine", "-data", f"{n.APP}/data/engine", "-model", "http://127.0.0.1:8081", "-worker", "nemotron",
            "-identity-url", "http://127.0.0.1:8796", "-identity-key-file", f"{n.APP}/secrets/engine.key", "-pulse", "20s"],
            {"PORT": "7798", "HOME": "/tmp"}, "http://127.0.0.1:7798/api/model")
    n.start_core(os.environ["CORE_BIN"])
    try:
        idn_post("/v1/entity", {"name": "TASS", "passphrase": n.PASS})
        idn_post("/v1/grants/skill", {"skill": "lesson-workspace.list-lessons", "passphrase": n.PASS})
        wait("tass.md", lambda: os.path.exists(f"{n.APP}/data/core/tass.md"), 60)
        core_post("/api/lessons/discover", {})
        with sync_playwright() as p:
            b = p.chromium.launch(executable_path="/usr/bin/google-chrome", args=["--no-sandbox"])
            pg = b.new_page(viewport={"width": 1500, "height": 950})
            pg.goto(CORE + "/")
            pg.click("text=Development →")
            wait_text(pg, "#lib", "List every lesson")
            lib = pg.inner_text("#lib")
            check("library shows real lessons, notebook cells and the skill", all(t in lib for t in ("My work pulse", "List every lesson", "@lesson-workspace.list-lessons", "CODE-CELL", "SKILL")), "")
            add_block(pg, "My work pulse")
            add_block(pg, "List every lesson")
            add_block(pg, "@lesson-workspace.list-lessons")
            check("three blocks on the canvas", pg.locator("#canvas .node").count() == 3, str(pg.locator("#canvas .node").count()))
            # Drag the first block.
            node = pg.locator("#canvas .node").nth(0)
            bb = node.bounding_box()
            pg.mouse.move(bb["x"] + 40, bb["y"] + 15); pg.mouse.down(); pg.mouse.move(bb["x"] + 120, bb["y"] + 260, steps=8); pg.mouse.up()
            # Connect "List every lesson" --uses--> skill through the keyboard-accessible inspector.
            pg.locator("#canvas .node", has_text="List every lesson").click()
            pg.select_option("#cRel", "uses")
            pg.select_option("#cTo", label="@lesson-workspace.list-lessons")
            pg.fill("#cWhy", "the notebook cell describes listing; the skill really does it, with Identity + approval")
            pg.locator("#insp").get_by_role("button", name="Connect", exact=True).click()
            # Connect lesson --supports--> cell by clicking on the canvas.
            pg.locator("#canvas .node", has_text="My work pulse").click()
            pg.select_option("#cRel", "supports")
            pg.click("text=…or click a block on the canvas")
            pg.locator("#canvas .node", has_text="List every lesson").click()
            check("two connections drawn", pg.locator("svg#links path.edge").count() == 2, str(pg.locator("svg#links path.edge").count()))
            pg.click("#check")
            wait_text(pg, "#report", "PASS")
            check("Check: PASS, says NOT RUN", "NOT RUN" in pg.inner_text("#report"), pg.inner_text("#report")[:120])
            pg.fill("#change", "first map: list lessons with the real skill")
            pg.click("#save")
            wait_text(pg, "#rev", "revision 1")
            pg.screenshot(path=f"{OUT}/1-brain-saved.png")
            g = brain()["graph"]
            check("saved revision 1 with 3 blocks, 2 connections, reasons and creator", g["revision"] == 1 and len(g["nodes"]) == 3 and len(g["edges"]) == 2 and any(e["relation"] == "uses" and "Identity" in e["reason"] and e["creator"] == "owner" for e in g["edges"]), "")
            pg.reload()
            wait_text(pg, "#rev", "revision 1")
            check("after reload: same blocks and connections", pg.locator("#canvas .node").count() == 3 and pg.locator("svg#links path.edge").count() == 2, "")
            # Select the edge, change its relation, save revision 2.
            pg.locator("svg#links text.el", has_text="supports").click()
            wait_text(pg, "#insp", "Connection")
            pg.select_option("#eRel", "next")
            pg.click("#save")
            wait_text(pg, "#rev", "revision 2")
            check("edge relation edited and saved as revision 2", any(e["relation"] == "next" for e in brain()["graph"]["edges"]), "")
            # Brain view: focus the skill.
            pg.locator("#canvas .node", has_text="@lesson-workspace").click()
            pg.check("#focusMode")
            time.sleep(0.5)
            dimmed = pg.locator("#canvas .node.dim").count()
            pg.screenshot(path=f"{OUT}/2-brain-focus-skill.png")
            check("Brain view focuses the selected block and its neighbors", dimmed == 1, f"dimmed={dimmed}")
            pg.uncheck("#focusMode")
            # Restore revision 1 as a new revision 3.
            pg.locator("#hist .ev", has_text="rev 1").locator("button").click()
            wait_text(pg, "#rev", "revision 3")
            g = brain()
            check("restore rev 1 → new revision 3; history keeps 1 and 2", g["graph"]["revision"] == 3 and [h["revision"] for h in g["history"]] == [3, 2, 1] and not any(e["relation"] == "next" for e in g["graph"]["edges"]), "")
            # Ask TASS with the saved brain.
            pg.fill("#ask", "Which block should you use first to find a lesson, and why?")
            pg.click("#askBtn")
            wait_text(pg, "#replies", "TASS:", 25)
            pg.screenshot(path=f"{OUT}/3-ask-tass.png")
            convo = open(f"{n.APP}/data/core/conversation.jsonl").read()
            check("TASS answered; the conversation log records brain revision 3", '"context":"brain revision 3"' in convo, "")
            # Skill request from the brain inspector.
            pg.locator("#canvas .node", has_text="@lesson-workspace").click()
            pg.click("text=Request a run (you approve on the Core page)")
            wait_text(pg, "#report", "AWAITING_APPROVAL")
            check("skill request from the brain creates a receipt that waits for approval", True, pg.inner_text("#report")[:100])
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
