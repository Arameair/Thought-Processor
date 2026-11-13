#!/usr/bin/env python3
"""Stage 3B runtime: confirm A, then B, on one frozen snapshot, and compare. NOT the owner's machine, NOT
Nemotron. Every payload is Core's real output at one step of an experiment
(development_experiment_run.json, checked by TestExperimentRunBrowserFixtureIsCoreOutput); replies are
replaced with hostile text.

X1  ready: status, snapshot NOT CAPTURED, Run A enabled, Run B disabled with its reason; nothing sent
X2  Run A opens an in-page confirmation and sends nothing; Cancel sends nothing
X3  Confirm (double click) sends exactly ONE POST to …/arms/A with the plan digest and "RUN A <id>"
X4  while A runs: both Run buttons disabled, snapshot FROZEN with reference, capture time and digest;
    the page only READS until A is final; then A's reply is shown literally, "saved into the Thought: no",
    1 request, B enabled; no B request was sent by the page
X5  B needs its own confirmation: ONE POST to …/arms/B with "RUN B <id>"; both replies, the comparison
    sentence, "2 of at most 2", no Run button enabled, no Promote
X6  an UNCERTAIN A: no reply shown, B disabled and said to be final, nothing can be sent
X7  a refused confirmation (grant off): the reason is shown, the record is re-read, nothing is re-sent
X8  1280 / 860 / 390: no sideways scrolling; hostile text stays text; no other POSTs; no page errors
    python3 development_experiment_run.py
"""
import copy
import json
import sys
from pathlib import Path
from urllib.parse import urlparse
from playwright.sync_api import sync_playwright

ROOT = Path(__file__).resolve().parents[2]
DEV = (ROOT / "cmd/tass-core/dev.html").read_text()
FX = json.loads((Path(__file__).resolve().parent / "development_experiment_run.json").read_text())
ORIGIN = "http://127.0.0.1:8798"
RUN = FX["run"]
NID = RUN["nudge_id"]
EXP = FX["ready"]["experiment_id"]
EXP2 = FX["a_uncertain"]["experiment_id"]
HOSTILE = '<img src=x onerror="window.__xss=1"><b>bold?</b>'
STEPS = {k: copy.deepcopy(FX[k]) for k in ["ready", "a_running", "after_a", "after_b"]}
for k in ["after_a", "after_b"]:
    for a in STEPS[k]["arms"]:
        if a.get("reply"):
            a["reply"] = "NOTE: reply of " + a["arm"] + " " + HOSTILE + "\nSTATE: keep"
REPLY_A = "NOTE: reply of A " + HOSTILE + "\nSTATE: keep"
REPLY_B = "NOTE: reply of B " + HOSTILE + "\nSTATE: keep"
SHOT = Path("/tmp/experiment-run"); SHOT.mkdir(exist_ok=True)

state = {"step": "ready", "polls": 0, "refuse": None}
posts, gets = [], []


def route(req):
    r = req.request
    path = urlparse(r.url).path
    if path == "/dev" and r.method == "GET":
        req.fulfill(status=200, content_type="text/html", body=DEV.replace("__PAGE_TOKEN__", "t")); return
    if r.method == "POST" and path.startswith("/api/development/experiments/" + EXP + "/arms/"):
        arm = path.rsplit("/", 1)[1]
        posts.append((path, json.loads(r.post_data or "{}")))
        if state["refuse"]:
            code, body = state["refuse"]
            req.fulfill(status=code, content_type="application/json", body=json.dumps(body)); return
        state["step"], state["polls"] = ("a_running", 0) if arm == "A" else ("after_b", 0)
        req.fulfill(status=202, content_type="application/json", body=json.dumps(STEPS[state["step"]])); return
    if r.method != "GET":
        posts.append((path, r.post_data)); req.fulfill(status=405); return
    gets.append(path)
    if path == "/api/development/runs/" + NID:
        req.fulfill(status=200, content_type="application/json", body=json.dumps(RUN)); return
    if path == "/api/development/experiments":
        req.fulfill(status=200, content_type="application/json", body=json.dumps(FX["list_after_b"])); return
    if path == "/api/development/experiments/" + EXP:
        if state["step"] == "a_running":
            state["polls"] += 1
            if state["polls"] >= 2:  # the Engine finished A; the page learns it only by reading
                state["step"] = "after_a"
        req.fulfill(status=200, content_type="application/json", body=json.dumps(STEPS[state["step"]])); return
    if path == "/api/development/experiments/" + EXP2:
        req.fulfill(status=200, content_type="application/json", body=json.dumps(FX["a_uncertain"])); return
    if path in ("/api/brain", "/api/status"):
        req.fulfill(status=200, content_type="application/json", body="{}"); return
    req.fulfill(status=404, body="")


PROBE = """()=>{const q=s=>document.querySelector(s), vis=e=>!!e&&e.checkVisibility({visibilityProperty:true})&&e.getClientRects().length>0;
  const col=a=>{const c=q('#expPlan .col[data-arm="'+a+'"]'); if(!c) return null; const b=c.querySelector('button.arm');
    return {state:c.dataset.state, text:c.textContent, reply:(c.querySelector('pre.reply')||{}).textContent||'', replyHTML:(c.querySelector('pre.reply')||{}).innerHTML||'',
      runDisabled:b?b.disabled:null, runText:b?b.textContent:'', why:(c.querySelector('.why')||{}).textContent||'', confirm:vis(c.querySelector('.confirm')),
      confirmText:(c.querySelector('.confirm')||{}).textContent||''}};
  const s=q('#expSnapshot');
  return {status:(q('#expStatus')||{}).textContent||'', snap:s?{status:s.dataset.status,text:s.textContent}:null, A:col('A'), B:col('B'),
    compare:(q('#expCompare')||{}).textContent||'', compareReady:(q('#expCompare')||{dataset:{}}).dataset.ready, err:(q('#expRunErr')||{}).textContent||'',
    cols:[...document.querySelectorAll('#expPlan .col h4')].map(h=>h.textContent),
    promote:[...document.querySelectorAll('button,a')].filter(b=>/promote/i.test(b.textContent)).length,
    kicker:(q('#expView .kicker')||{}).textContent||'', plan:(q('#expPlan')||{}).textContent||'',
    overX:document.scrollingElement.scrollWidth-innerWidth}}"""

failures, passed = [], 0
with sync_playwright() as p:
    b = p.chromium.launch(headless=True, args=["--no-sandbox"])
    try:
        for (w, h, tag) in [(1280, 900, "desktop"), (860, 900, "half"), (390, 844, "phone")]:
            def check(name, cond, detail=""):
                global passed
                if cond:
                    passed += 1
                else:
                    failures.append(f"[{tag}] {name}: {str(detail)[:500]}")

            posts.clear(); gets.clear(); state.update(step="ready", polls=0, refuse=None)
            ctx = b.new_context(viewport={"width": w, "height": h})
            pg = ctx.new_page(); errs = []
            pg.on("pageerror", lambda e: errs.append(str(e)))
            pg.route("**/*", route)
            try:
                # X1
                pg.goto(ORIGIN + "/dev?run=" + NID + "&exp=" + EXP); pg.wait_for_selector("#expPlan .col[data-arm=A]")
                v = pg.evaluate(PROBE)
                check("X1 status: planned, 0 requests, nothing saved", "Planned. Nothing has run." in v["status"] and "0 of at most 2" in v["status"] and "nothing" in v["status"], v["status"])
                check("X1 snapshot NOT CAPTURED, and says when it will be", v["snap"]["status"] == "NOT CAPTURED" and "when you confirm A" in v["snap"]["text"], v["snap"])
                check("X1 Original | A | B", [c[:8] for c in v["cols"]] == ["Original", "A · unch", "B · your"], v["cols"])
                check("X1 Run A enabled", v["A"]["runDisabled"] is False and "Run A" in v["A"]["runText"], v["A"])
                check("X1 Run B disabled with its reason", v["B"]["runDisabled"] is True and "Run A first" in v["B"]["why"], v["B"])
                check("X1 nothing sent", not posts, posts)
                check("X8 no sideways scrolling (ready)", v["overX"] <= 1, v["overX"])
                pg.screenshot(path=str(SHOT / f"1-ready-{tag}.png"), full_page=True)
                # X2
                pg.locator("#expPlan button.arm[data-arm=A]").click(timeout=5000)
                v = pg.evaluate(PROBE)
                check("X2 confirmation shown; nothing sent", v["A"]["confirm"] and "ONE request" in v["A"]["confirmText"] and "never retried" in v["A"]["confirmText"]
                      and "Nothing is saved into the Thought" in v["A"]["confirmText"] and not posts, (v["A"]["confirmText"], posts))
                pg.screenshot(path=str(SHOT / f"2-confirm-A-{tag}.png"), full_page=True)
                pg.locator("#expPlan .confirm button", has_text="Cancel").click(timeout=5000)
                check("X2 Cancel sends nothing", not posts and not pg.evaluate(PROBE)["A"]["confirm"], posts)
                # X3
                pg.locator("#expPlan button.arm[data-arm=A]").click(timeout=5000)
                pg.locator("#expPlan button[data-confirm=A]").dblclick(timeout=5000)
                pg.wait_for_selector('#expPlan .col[data-arm=A][data-state=RUNNING]', timeout=5000)
                check("X3 exactly one POST for a double click", len(posts) == 1, posts)
                path0, body0 = posts[0]
                check("X3 it is arm A of this plan, with the digest and the phrase", path0.endswith("/" + EXP + "/arms/A") and body0 == {"body_digest": FX["ready"]["body_digest"], "confirm": "RUN A " + EXP}, posts[0])
                # X4
                v = pg.evaluate(PROBE)
                check("X4 while A runs both Run buttons are disabled", v["A"]["runDisabled"] and v["B"]["runDisabled"] and "A is RUNNING" in v["status"], (v["A"]["runDisabled"], v["B"]["runDisabled"]))
                fz = FX["a_running"]["frozen_snapshot"]
                check("X4 snapshot FROZEN: reference, capture time, digest", v["snap"]["status"] == "FROZEN" and fz["reference"] in v["snap"]["text"] and fz["sha256"] in v["snap"]["text"]
                      and fz["captured_at"] in v["snap"]["text"] and "not from the Thought as it is later" in v["snap"]["text"], v["snap"])
                check("X4 no 'as the Thought is now' wording after freezing", "as the thought is now" not in v["plan"].lower(), "")
                pg.screenshot(path=str(SHOT / f"3-A-running-{tag}.png"), full_page=True)
                n_get = gets.count("/api/development/experiments/" + EXP)
                pg.wait_for_selector('#expPlan .col[data-arm=A][data-state=COMPLETED]', timeout=9000)
                v = pg.evaluate(PROBE)
                check("X4 the page only read until A was final", len(posts) == 1 and gets.count("/api/development/experiments/" + EXP) > n_get, (len(posts), n_get))
                check("X4 A's reply literally", v["A"]["reply"] == REPLY_A and "<img" not in v["A"]["replyHTML"], v["A"]["reply"])
                check("X4 A: saved into the Thought: no; 1 request; approval shown", "Saved into the Thought: no" in v["A"]["text"] and "single use" in v["A"]["text"]
                      and "1 of at most 2" in v["status"] and "A completed. B has not run." in v["status"], v["status"])
                check("X4 B enabled now; nothing to compare yet", v["B"]["runDisabled"] is False and v["compareReady"] == "false" and "Nothing to compare yet" in v["compare"], v["compare"])
                check("X4 A can't be run again", v["A"]["runDisabled"] is True, "")
                pg.screenshot(path=str(SHOT / f"4-after-A-{tag}.png"), full_page=True)
                pg.wait_for_timeout(1800)
                check("X4 B was not sent by the page", len(posts) == 1, posts)
                # X5
                pg.locator("#expPlan button.arm[data-arm=B]").click(timeout=5000)
                v = pg.evaluate(PROBE)
                check("X5 B's own confirmation names the same snapshot", v["B"]["confirm"] and "same frozen snapshot as A" in v["B"]["confirmText"] and len(posts) == 1, v["B"]["confirmText"])
                pg.locator("#expPlan button[data-confirm=B]").click(timeout=5000)
                pg.wait_for_selector('#expPlan .col[data-arm=B][data-state=COMPLETED]', timeout=5000)
                v = pg.evaluate(PROBE)
                check("X5 exactly one POST for B", len(posts) == 2 and posts[1][0].endswith("/arms/B") and posts[1][1].get("confirm") == "RUN B " + EXP, posts)
                check("X5 both replies literally", v["A"]["reply"] == REPLY_A and v["B"]["reply"] == REPLY_B and "<img" not in v["B"]["replyHTML"], (v["A"]["reply"], v["B"]["reply"]))
                check("X5 the comparison is Core's sentence", v["compareReady"] == "true" and FX["after_b"]["comparison"]["text"] in v["compare"], v["compare"])
                check("X5 2 of at most 2; both completed", "2 of at most 2" in v["status"] and "Both arms completed." in v["status"], v["status"])
                check("X5 no Run button enabled; no Promote", v["A"]["runDisabled"] and v["B"]["runDisabled"] and v["promote"] == 0, v["promote"])
                check("X5 B: saved into the Thought: no", "Saved into the Thought: no" in v["B"]["text"], "")
                check("X8 hostile text stays text", pg.evaluate("window.__xss") is None and pg.locator("#expPlan img").count() == 0, "")
                check("X8 no sideways scrolling (results)", v["overX"] <= 1, v["overX"])
                pg.screenshot(path=str(SHOT / f"5-after-B-{tag}.png"), full_page=True)
                pg.wait_for_timeout(1800)
                n_after = len(posts)
                pg.reload(); pg.wait_for_selector('#expPlan .col[data-arm=B][data-state=COMPLETED]')
                check("X5 refresh shows the same record and sends nothing", len(posts) == n_after and pg.evaluate(PROBE)["B"]["reply"] == REPLY_B, "")
                # X6
                pg.goto(ORIGIN + "/dev?run=" + NID + "&exp=" + EXP2); pg.wait_for_selector("#expPlan .col[data-arm=A]")
                v = pg.evaluate(PROBE)
                check("X6 UNCERTAIN A: no reply, said plainly", v["A"]["state"] == "UNCERTAIN" and v["A"]["reply"] == "" and "may have received it" in v["A"]["text"]
                      and "Ended: A is UNCERTAIN" in v["status"], v["A"])
                check("X6 B disabled and final; A not re-runnable", v["B"]["runDisabled"] and "final" in v["B"]["why"] and v["A"]["runDisabled"], v["B"])
                pg.screenshot(path=str(SHOT / f"6-A-uncertain-{tag}.png"), full_page=True)
                # X7
                state.update(step="ready", polls=0, refuse=(403, {"error": "Identity has not granted experiment runs: enable \"cognition.experiment-run\" on Identity's page", "code": "EXPERIMENT_NOT_GRANTED"}))
                pg.goto(ORIGIN + "/dev?run=" + NID + "&exp=" + EXP); pg.wait_for_selector("#expPlan .col[data-arm=A]")
                n0 = len(posts)
                pg.locator("#expPlan button.arm[data-arm=A]").click(timeout=5000)
                pg.locator("#expPlan button[data-confirm=A]").click(timeout=5000)
                pg.wait_for_function("document.querySelector('#expRunErr') && document.querySelector('#expRunErr').textContent.length>0", timeout=5000)
                v = pg.evaluate(PROBE)
                check("X7 the refusal is shown and nothing is re-sent", "was not started" in v["err"] and "cognition.experiment-run" in v["err"] and len(posts) == n0 + 1
                      and v["A"]["state"] == "NOT RUN", (v["err"], len(posts) - n0))
                pg.wait_for_timeout(1800)
                check("X7 still exactly one attempt", len(posts) == n0 + 1, len(posts) - n0)
            except Exception as e:
                failures.append(f"[{tag}] aborted: {str(e).splitlines()[0]}")
            check("X8 only arm POSTs were sent", all("/arms/" in p_[0] for p_ in posts), posts)
            check("X8 no Brain Builder read", "/api/brain" not in gets, "")
            check("X8 no page errors", not errs, errs)
            ctx.close()
    finally:
        b.close()
for f in failures:
    print("FAIL:", f)
print(f"{passed}/{passed + len(failures)} Development experiment-runtime checks PASS across 1280/860/390")
sys.exit(1 if failures else 0)
