import sys
from playwright.sync_api import sync_playwright

PASS = "correct horse battery staple"
out = sys.argv[1] if len(sys.argv) > 1 else "/tmp/e2e/identity.png"
with sync_playwright() as p:
    b = p.chromium.launch(executable_path="/usr/bin/google-chrome", args=["--no-sandbox"])
    pg = b.new_page()
    log = []
    pg.on("request", lambda r: log.append(f"POST {r.url.split('8796')[1]}") if r.method == "POST" else None)
    pg.on("response", lambda r: log.append(f"-> HTTP {r.status}") if r.request.method == "POST" else None)
    pg.goto("http://127.0.0.1:8796/")
    pg.wait_for_timeout(800)
    pg.fill("#up", PASS)
    pg.click("#ub")
    pg.wait_for_timeout(1200)
    print("state:", pg.inner_text("#state"), "| grants:", pg.inner_text("#grants"))
    print("learning card visible:", pg.is_visible("#learningGrants"))
    log.clear(); pg.click("#enableLearning"); pg.wait_for_timeout(800)
    print("A) click with EMPTY field -> message:", repr((pg.inner_text("#grantMsg") + " " + pg.inner_text("#err")).strip()), "| network:", log)
    log.clear(); pg.fill("#grantPass", "wrong wrong wrong"); pg.click("#enableLearning"); pg.wait_for_timeout(1500)
    print("B) WRONG passphrase -> message:", repr((pg.inner_text("#grantMsg") + " " + pg.inner_text("#err")).strip()), "| network:", log)
    log.clear(); pg.fill("#grantPass", PASS); pg.click("#enableLearning"); pg.wait_for_timeout(1500)
    print("C) RIGHT passphrase -> message:", repr((pg.inner_text("#grantMsg") + " " + pg.inner_text("#err")).strip()), "| network:", log)
    print("   success line:", repr(pg.inner_text("#grantDone"))); print("   grants now:", pg.inner_text("#grants"), "| card still visible:", pg.is_visible("#learningGrants"))
    pg.screenshot(path=out, full_page=True)
