from playwright.sync_api import sync_playwright
import time
with sync_playwright() as p:
    b=p.chromium.launch(executable_path="/usr/bin/google-chrome",args=["--no-sandbox"]); pg=b.new_page()
    pg.goto("http://127.0.0.1:8797/"); pg.wait_for_timeout(2000)
    pg.click("#btnAuto"); pg.wait_for_timeout(2000)
    print("after toggle:", pg.inner_text("#btnAuto"), "|", pg.inner_text("#btnPulse"))
    t0=time.time()
    while time.time()-t0<150:
        pg.wait_for_timeout(5000)
        rows=[l for l in pg.inner_text("#lessons").splitlines() if l in ("NEW","QUEUED","PASSED","LEARNED","RETRY","STUCK","NEEDS_WORKSPACE")]
        if "NEW" not in rows and "QUEUED" not in rows and "RETRY" not in rows: break
    print(f"{int(time.time()-t0)}s statuses:", rows)
    txt=pg.inner_text("#lessons"); print("\n".join("   "+l for l in txt.splitlines() if l.startswith(("✓","✗","auto-approved","Claims")) or "right ·" in l))
    pg.click("#btnAuto"); pg.wait_for_timeout(1500); print("turned back:", pg.inner_text("#btnAuto"))
