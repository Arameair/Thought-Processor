"""Drive the TASS Core page like the owner: Find lessons -> Study 'My work pulse' -> watch -> Approve.
Usage: python3 ui_core.py PHASE OUTPREFIX   PHASE = find | full
"""
import json, sys, time, urllib.request
from playwright.sync_api import sync_playwright

phase, out = sys.argv[1], sys.argv[2]
LESSON = "lessons/claude/01-my-work-pulse.md"


def lessons_text(pg):
    return pg.inner_text("#lessons")


with sync_playwright() as p:
    b = p.chromium.launch(executable_path="/usr/bin/google-chrome", args=["--no-sandbox"])
    pg = b.new_page(viewport={"width": 1300, "height": 1800})
    net = []
    pg.on("response", lambda r: net.append(f"{r.request.method} {r.url.split('8797')[1]} -> {r.status}")
          if r.request.method == "POST" else None)
    pg.goto("http://127.0.0.1:8797/")
    pg.wait_for_timeout(2500)
    print("Autonomous button:", pg.inner_text("#btnAuto"))
    print("GitHub line:", pg.inner_text("#ghinfo"))
    net.clear()
    pg.click("#findLessons")
    pg.wait_for_timeout(2500)
    print("FIND -> network:", net, "\n   message:", repr(pg.inner_text("#lessonMessage")))
    print("   lessons card:\n" + "\n".join("     " + l for l in lessons_text(pg).splitlines()))
    pg.screenshot(path=out + "-1-find.png", full_page=True)
    if phase == "find":
        sys.exit(0)
    btn = pg.locator(f'button[data-study="{LESSON}"]')
    print("Study button for My work pulse present:", btn.count())
    net.clear()
    btn.click()
    pg.wait_for_timeout(2000)
    print("STUDY -> network:", net, "\n   message:", repr(pg.inner_text("#lessonMessage")))
    t0 = time.time()
    status = ""
    while time.time() - t0 < 180:
        pg.wait_for_timeout(3000)
        txt = lessons_text(pg)
        line = [l for l in txt.splitlines() if "My work pulse" in l]
        status = line[0] if line else ""
        if "PASSED" in txt.split("My work pulse")[0][-40:] or "RETRY" in status or "STUCK" in status or "PASSED" in status:
            break
    print(f"after {int(time.time()-t0)}s lesson row:", status)
    print("   lessons card:\n" + "\n".join("     " + l for l in lessons_text(pg).splitlines()))
    pg.screenshot(path=out + "-2-studied.png", full_page=True)
    ap = pg.locator(f'button[data-approve="{LESSON}"]')
    print("Approve button present:", ap.count())
    if ap.count():
        net.clear()
        ap.click()
        pg.wait_for_timeout(2500)
        print("APPROVE -> network:", net, "\n   message:", repr(pg.inner_text("#lessonMessage")))
        print("   lessons card:\n" + "\n".join("     " + l for l in lessons_text(pg).splitlines()))
        md = pg.inner_text("#md")
        i = md.find("## Learned")
        print("   tass.md Learned section:", repr(md[i:i + 400]) if i >= 0 else "NOT FOUND")
        pg.screenshot(path=out + "-3-approved.png", full_page=True)
