"""P1 UI tour — a headless browser clicks through REAL Thought Processor and
REAL Identity pages, like the owner would, and saves screenshots.

GitHub is Identity's labelled TEST FIXTURE; the feature is the trusted local
test feature. The Admin passphrase is typed into IDENTITY's approval page (a
second browser tab), never into Processor.

  python3 -I acceptance/p1_ui_tour.py --identity-src DIR --work DIR --shots DIR
"""
import argparse, json, os, re, shutil, signal, subprocess, sys, time, urllib.request
from pathlib import Path
from playwright.sync_api import sync_playwright

ap = argparse.ArgumentParser()
ap.add_argument("--identity-src", required=True)
ap.add_argument("--work", required=True)
ap.add_argument("--shots", required=True)
args = ap.parse_args()
W = Path(args.work).resolve()
if W.exists():
    shutil.rmtree(W)
W.mkdir(parents=True)
SH = Path(args.shots).resolve()
SH.mkdir(parents=True, exist_ok=True)
PROC_SRC = Path(__file__).resolve().parent.parent
FX, IP, PP = 8951, 7951, 7971
ENV = {k: v for k, v in os.environ.items() if k.lower() not in ("https_proxy", "http_proxy")}
ENV.update(IDENTITY_LOCATOR_DIR=str(W / "locators"), IDENTITY_GITHUB_CLIENT_ID="IvFIXTURE00000001")
NOPROXY = urllib.request.build_opener(urllib.request.ProxyHandler({}))
PASS = "correct horse battery staple"
procs, results = {}, []


def rec(ok, name):
    results.append(("PASS" if ok else "FAIL", name))
    print(("PASS " if ok else "FAIL ") + name, flush=True)


def start(name, cmd):
    procs[name] = subprocess.Popen(cmd, stdout=open(W / (name + ".console.txt"), "a"), stderr=subprocess.STDOUT, env=ENV)


def wait_http(u, t=20):
    end = time.time() + t
    while time.time() < end:
        try:
            NOPROXY.open(u, timeout=1)
            return True
        except Exception:
            time.sleep(0.2)
    return False


def code():
    m = re.findall(r"\b([A-HJ-NP-Z2-9]{4}-[A-HJ-NP-Z2-9]{4})\b", (W / "identity.console.txt").read_text())
    return m[-1]


def main():
    subprocess.check_call(["go", "build", "-o", str(W / "identity"), "."], cwd=args.identity_src)
    subprocess.check_call(["go", "build", "-o", str(W / "thought-processor"), "."], cwd=PROC_SRC)
    (W / "asset").write_bytes(b"x")
    start("fixture", [sys.executable, "-I", str(Path(args.identity_src) / "scripts/fixture_github_engine.py"), str(FX), str(W / "asset"), "linux-amd64"])
    wait_http(f"http://127.0.0.1:{FX}/fixture/state")
    start("identity", [str(W / "identity"), "-port", str(IP), "-data", str(W / "id"), "-test-github-web", f"http://127.0.0.1:{FX}", "-test-github-api", f"http://127.0.0.1:{FX}"])
    wait_http(f"http://127.0.0.1:{IP}/api/health")
    start("processor", [str(W / "thought-processor"), "-port", str(PP), "-data", str(W / "tp"), "-no-browser", "-trust-feature-dir", str(PROC_SRC / "testfeatures")])
    wait_http(f"http://127.0.0.1:{PP}/api/health")
    for _ in range(50):
        f = W / "tp/runtime/operator-link.txt"
        if f.exists():
            link = f.read_text().strip()
            break
        time.sleep(0.1)
    pw = sync_playwright().start()
    br = pw.chromium.launch()
    ctx = br.new_context(viewport={"width": 1400, "height": 920}, color_scheme="dark")
    page = ctx.new_page()
    page.on("dialog", lambda d: d.accept())
    errors = []
    page.on("pageerror", lambda e: errors.append(str(e)))
    shot = lambda n: page.screenshot(path=str(SH / (n + ".png")), full_page=True)

    def admin_in_identity(click, fill):
        with ctx.expect_page() as pinfo:
            click()
        ip = pinfo.value
        ip.wait_for_selector("#ap-main:not([hidden])", timeout=10000)
        fill(ip)
        return ip

    page.goto(link)
    page.wait_for_selector("[data-connect]", timeout=20000)
    shot("ui-01-connect")
    page.click("[data-connect]")
    page.wait_for_selector("#code-in", timeout=10000)
    time.sleep(0.3)
    page.fill("#code-in", code())
    page.click("#code-form button[type=submit]")
    page.wait_for_selector("#home:not([hidden])", timeout=15000)
    rec(True, "connected to Identity by typing the code from Identity's window")
    page.click("[data-sel='admin:admin']")
    page.wait_for_selector("[data-gh]")
    shot("ui-02-admin-not-established")
    page.click("[data-gh]")
    page.wait_for_selector("[data-enroll]", timeout=40000)
    rec(True, "GitHub sign-in (TEST FIXTURE) verified from Processor's Admin page")
    page.click("[data-enroll]")
    page.wait_for_selector("#m-approval:not([hidden])")
    shot("ui-03-approval-handoff")

    def enroll(ip):
        ip.screenshot(path=str(SH / "ui-04-identity-enroll-page.png"), full_page=True)
        ip.fill("#ap-pass1", PASS)
        ip.fill("#ap-pass2", PASS)
        ip.check("#ap-ack")
        ip.click("#ap-ok")
        ip.wait_for_selector("#ap-done:not([hidden])", timeout=15000)
        ip.screenshot(path=str(SH / "ui-05-identity-enrolled.png"), full_page=True)
    ip = admin_in_identity(lambda: page.click("#ap-open"), enroll)
    ip.close()
    page.wait_for_selector("#chip-admin.ok", timeout=15000)
    page.click("#ap-close")
    page.click("[data-sel='admin:admin']")
    time.sleep(0.6)
    shot("ui-06-admin-identity")
    rec("Admin Identity" in page.inner_text("#detail") and "adm_" in page.inner_text("#detail"), "Admin Identity established in Identity's page and shown in Processor")

    page.click("#add-id-btn")
    page.fill("#new-name", "Tass")
    page.select_option("#new-kind", "companion")
    page.click("#new-id button[type=submit]")
    page.wait_for_selector("text=Tass")
    page.click("#add-feature-btn")
    page.wait_for_selector("[data-review]")
    shot("ui-07-library")
    page.click("[data-review]")
    page.wait_for_selector("#m-install:not([hidden])")
    shot("ui-08-install-review")
    page.click("#i-ok")
    page.wait_for_selector("#m-approval:not([hidden])")

    def approve(ip, name=None):
        if name:
            ip.screenshot(path=str(SH / (name + ".png")), full_page=True)
        ip.fill("#ap-pass", PASS)
        ip.click("#ap-ok")
        ip.wait_for_selector("#ap-done:not([hidden])", timeout=15000)
    ip = admin_in_identity(lambda: page.click("#ap-open"), lambda p: approve(p, "ui-09-identity-install-approval"))
    ip.close()
    page.wait_for_selector(".idrow[data-sel^='feature:ins_']", timeout=30000)
    page.click("#ap-close")
    page.click(".idrow[data-sel^='feature:ins_']")
    page.click("[data-tab=overview]")
    time.sleep(0.6)
    shot("ui-10-feature-overview-zero-grants")
    txt = page.inner_text("#detail")
    rec("FEATURE INSTALLATION IDENTITY" in txt and "0 active grants" in txt and "Identity: INSTALLED" in txt, "installed feature shown as an identity: INSTALLED, 0 grants")
    page.click("[data-tab=runtime]")
    page.click("[data-start]")
    page.wait_for_selector("[data-selftest]", timeout=15000)
    time.sleep(2.5)
    page.click("[data-selftest]")
    page.wait_for_selector("text=Self-test PASS", timeout=10000)
    shot("ui-11-runtime-running-selftest")
    rec(True, "started from the Runtime tab; standalone self-test PASS shown")
    page.click("[data-tab=connections]")
    page.click("#action-form button[type=submit]")
    page.wait_for_selector(".result.bad", timeout=15000)
    shot("ui-12-denied-before-grant")
    rec("DENIED BY IDENTITY" in page.inner_text("#fx-action-result"), "synthetic request before any grant: DENIED BY IDENTITY")
    page.select_option("#gf-exp", "0")
    page.click("#grant-form button[type=submit]")
    page.wait_for_selector("#m-approval:not([hidden])")
    ip = admin_in_identity(lambda: page.click("#ap-open"), lambda p: approve(p, "ui-13-identity-grant-approval"))
    ip.close()
    page.wait_for_selector("[data-revoke]", timeout=20000)
    page.click("#ap-close")
    page.click("#action-form button[type=submit]")
    page.wait_for_selector(".result.ok", timeout=15000)
    shot("ui-14-allowed-synthetic-observation")
    rec("COMPLETED" in page.inner_text("#fx-action-result") and "SYNTHETIC" in page.inner_text("#fx-action-result"), "after Admin approval: COMPLETED with SYNTHETIC TEST DATA")
    page.click("[data-revoke]")
    page.wait_for_selector("text=REVOKED", timeout=10000)
    page.click("#action-form button[type=submit]")
    page.wait_for_selector(".result.bad", timeout=15000)
    rec("REVOKED" in page.inner_text("#fx-action-result"), "after Revoke in the UI: next request denied")
    shot("ui-15-revoked-denied")
    page.click("[data-tab=metrics]")
    time.sleep(2.5)
    shot("ui-16-metrics")
    page.click("[data-tab=logs]")
    page.wait_for_selector("text=authz.checked", timeout=10000)
    shot("ui-17-logs-decisions")
    rec(True, "Logs tab shows Identity decisions, the feature's own view and its log")
    page.select_option("#theme", "light")
    time.sleep(0.8)
    page.click("[data-tab=overview]")
    time.sleep(0.4)
    shot("ui-18-light-theme")
    page.select_option("#theme", "dark")
    procs["identity"].send_signal(signal.SIGINT)
    procs["identity"].wait(10)
    page.wait_for_selector("#banner:not([hidden])", timeout=20000)
    page.click("[data-tab=connections]")
    page.click("#action-form button[type=submit]")
    page.wait_for_selector(".result.warn", timeout=15000)
    shot("ui-19-authority-unavailable")
    rec("AUTHORITY UNAVAILABLE" in page.inner_text("#fx-action-result"), "Identity stopped: banner + AUTHORITY UNAVAILABLE (not DENIED)")
    rec(not errors, "no JavaScript errors on the page" + ("" if not errors else ": " + "; ".join(errors[:3])))
    br.close()
    pw.stop()


try:
    main()
except Exception as e:
    import traceback
    traceback.print_exc()
    rec(False, "tour crashed: " + repr(e)[:160])
finally:
    for p in procs.values():
        if p.poll() is None:
            p.send_signal(signal.SIGINT)
            try:
                p.wait(15)
            except subprocess.TimeoutExpired:
                p.kill()
fails = [r for r in results if r[0] != "PASS"]
(SH / "ui-tour-report.txt").write_text("\n".join(f"{a} {b}" for a, b in results) + f"\n\nRESULT: {'PASS' if not fails else 'FAIL'} ({len(results)} checks, {len(fails)} failed)\n")
print("RESULT:", "PASS" if not fails else "FAIL")
sys.exit(1 if fails else 0)
