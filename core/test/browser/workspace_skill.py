#!/usr/bin/env python3
"""Drive the Core page like the owner for #72/#31 (TEST FIXTURE model and fake GitHub; real
Identity, Engine and Core binaries): Talk "Find lessons using @" -> Workspace receipt ->
Approve run -> real output -> view a project file -> ask TASS about it. Screenshots to OUT.

    APP_HOME=/tmp/e2e/app3 BIN=/tmp/e2e/bin SEED=/tmp/e2e/seed CAND=x ALPHA=x OUT=/tmp/e2e/shots \
    IDENTITY_BIN=tass-identity-new CORE_BIN=tass-core-c2 python3 core/test/browser/workspace_skill.py
"""
import os, sys, time

sys.path.insert(0, os.path.join(os.path.dirname(os.path.abspath(__file__)), ".."))
import lesson_retry_native as n  # noqa: E402
from acceptance import CORE, check, core_post, idn_post, results, wait  # noqa: E402
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
        wait("tass.md", lambda: os.path.exists(f"{n.APP}/data/core/tass.md"), 60)
        core_post("/api/lessons/discover", {})
        with sync_playwright() as p:
            b = p.chromium.launch(executable_path="/usr/bin/google-chrome", args=["--no-sandbox"])
            idp = b.new_page(viewport={"width": 1000, "height": 1200})
            idp.goto("http://127.0.0.1:8796/")
            idp.wait_for_selector("#skillGrants:not([hidden])", timeout=10000)
            idp.fill("#skillPass", n.PASS)
            idp.click("[data-enable-skill='lesson-workspace.list-lessons']")
            wait_text(idp, "#skillMsg", "Enabled")
            idp.screenshot(path=f"{OUT}/1-identity-skill-enabled.png", full_page=True)
            check("Identity page: owner enables the skill with passphrase", "GRANTED" in idp.inner_text("#skillList"), idp.inner_text("#skillMsg"))

            pg = b.new_page(viewport={"width": 1300, "height": 1700})
            pg.goto(CORE + "/")
            wait_text(pg, "#wsInfo", "Mira/home")
            pg.fill("#t", "Find lessons using @")
            pg.press("#t", "Enter")
            pg.wait_for_selector("[data-skill-approve]", timeout=15000)
            pg.screenshot(path=f"{OUT}/2-request-awaiting-approval.png", full_page=False)
            check("Talk → Workspace shows AWAITING_APPROVAL with an Approve run button", "AWAITING_APPROVAL" in pg.inner_text("#wsReceipts"), "")
            pg.click("[data-skill-approve]")
            wait_text(pg, "#wsReceipts", "DONE")
            pg.click("#wsReceipts details summary")
            time.sleep(1.5)
            pg.screenshot(path=f"{OUT}/3-receipt-done-output.png", full_page=False)
            txt = pg.inner_text("#wsReceipts")
            check("receipt DONE, executed once, MET, real output visible", "DONE" in txt and "yes, once" in txt and "MET" in txt and "count=3" in txt, "")
            check("Talk shows TASS's report from the real receipt", "I ran @lesson-workspace.list-lessons" in pg.inner_text("#log"), "")
            pg.select_option("#wsFile", "lessons/claude/01-my-work-pulse.md")
            pg.click("#wsView")
            pg.wait_for_selector("#wsFileText:not([hidden])", timeout=10000)
            pg.fill("#wsQ", "What does step 2 of this lesson say I do?")
            pg.click("#wsAsk button")
            wait_text(pg, "#log", "read-only file as context", 20)
            time.sleep(1.5)
            pg.locator("#wsCard").screenshot(path=f"{OUT}/4-workspace-file-and-ask.png")
            check("file view: read-only, path@branch and blob shown", "read-only" in pg.inner_text("#wsFileHead") and "Mira/home@live" in pg.inner_text("#wsFileHead"), pg.inner_text("#wsFileHead")[:160])
            check("ask-about-file reached TASS with the file as read-only context", "read-only file as context" in pg.inner_text("#log"), "")
            pg.screenshot(path=f"{OUT}/5-full-page.png", full_page=True)
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
