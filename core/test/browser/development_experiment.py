#!/usr/bin/env python3
"""Stage 3B plan slice: "Try one change…" from Observe. NOT the owner's machine, NOT Nemotron. Payloads come
from development_experiment.json, Core's real output (TestExperimentBrowserFixtureIsCoreOutput).

E1  Observe shows one "Try one change…" action beside the next action; nothing is fetched or sent until used
E2  it opens the experiment form in Observe (&exp=new): the instruction sentence pre-filled with A's, a
    character count, the reply format shown but NOT editable, AS_OF_NOW chosen and AS_OF_BASELINE disabled
    with its reason, what is held constant, what won't change, and "0 model requests" for saving
E3  the page refuses NOTE:/STATE:, more than 300 characters, two lines and "same as A" before any POST
E4  Save sends exactly ONE POST (a double click included), to /api/development/experiments, with the
    baseline nudge, the sentence, AS_OF_NOW and an idempotency key; a refused save shows Core's reason
E5  the saved plan: its ID, "0 model requests", A and B sentences, Original | A | B with the original Note
    literal, A and B NOT RUN, "saved into the Thought" yes/no/no, Run A / Run B disabled and NOT
    AUTHORIZED, no Promote control
E6  direct URL, refresh, Back and Forward keep the plan and the run; the run lists its saved plans
E7  1280 / 860 / 390: no sideways scrolling; at 1280 and 860 the sentence and Save are on screen
E8  zero other POSTs, no /api/brain read, hostile text stays text, no page errors
    python3 development_experiment.py
"""
import copy
import json
import sys
from pathlib import Path
from urllib.parse import urlparse, parse_qs
from playwright.sync_api import sync_playwright

ROOT = Path(__file__).resolve().parents[2]
DEV = (ROOT / "cmd/tass-core/dev.html").read_text()
FX = json.loads((Path(__file__).resolve().parent / "development_experiment.json").read_text())
ORIGIN = "http://127.0.0.1:8797"
RUN = FX["run"]
NID = RUN["nudge_id"]
HOSTILE = '<img src=x onerror="window.__xss=1"><b>bold?</b>'
PLAN = copy.deepcopy(FX["plan"])
PLAN["variant"] = "Answer the question directly " + HOSTILE
PLAN["original"]["text"] = "Original note " + HOSTILE
EXP = PLAN["experiment_id"]
LIST_BEFORE = copy.deepcopy(FX["list_before"])
LIST_AFTER = copy.deepcopy(FX["list_after"])
LIST_AFTER["experiments"] = [PLAN]
SHOT = Path("/tmp/experiment"); SHOT.mkdir(exist_ok=True)

state = {"saved": False, "post": None}
posts, gets = [], []


def route(req):
    r = req.request
    u = urlparse(r.url)
    path = u.path
    if path == "/dev" and r.method == "GET":
        req.fulfill(status=200, content_type="text/html", body=DEV.replace("__PAGE_TOKEN__", "t")); return
    if r.method == "POST" and path == "/api/development/experiments":
        body = json.loads(r.post_data or "{}")
        posts.append((path, body))
        if state["post"]:
            code, out = state["post"]
            req.fulfill(status=code, content_type="application/json", body=json.dumps(out)); return
        state["saved"] = True
        req.fulfill(status=201, content_type="application/json", body=json.dumps(PLAN)); return
    if r.method != "GET":
        posts.append((path, r.post_data)); req.fulfill(status=405); return
    gets.append(path)
    if path == "/api/development/runs/" + NID:
        req.fulfill(status=200, content_type="application/json", body=json.dumps(RUN)); return
    if path == "/api/development/experiments" and parse_qs(u.query).get("run") == [NID]:
        req.fulfill(status=200, content_type="application/json", body=json.dumps(LIST_AFTER if state["saved"] else LIST_BEFORE)); return
    if path == "/api/development/experiments/" + EXP:
        req.fulfill(status=200, content_type="application/json", body=json.dumps(PLAN)); return
    if path in ("/api/brain", "/api/status"):
        req.fulfill(status=200, content_type="application/json", body="{}"); return
    req.fulfill(status=404, body="")


PROBE = """()=>{const q=s=>document.querySelector(s), vis=e=>!!e&&e.checkVisibility({visibilityProperty:true})&&e.getClientRects().length>0;
  const box=e=>{if(!vis(e))return null;const r=e.getBoundingClientRect();return {top:r.top,bottom:r.bottom,right:r.right}};
  const f=q('#expForm'), p=q('#expPlan');
  return {url:location.href, mode:document.body.dataset.mode, tryBtn:vis(q('#expOpen')), tryText:(q('#expOpen')||{}).textContent||'',
    map:vis(q('#runMap')), form:vis(f), plan:vis(p),
    variant:(q('#expVariant')||{}).value, count:(q('#expCount')||{}).textContent||'',
    contractTag:(q('#expContract')||{}).tagName||'', contract:(q('#expContract')||{}).textContent||'',
    snaps:[...document.querySelectorAll('#expForm input[name=expSnap]')].map(i=>({v:i.value,checked:i.checked,disabled:i.disabled,why:(i.closest('label')||{}).textContent||''})),
    formText:f?f.textContent:'', err:(q('#expErr')||{}).textContent||'', saveBox:box(q('#expSave')), varBox:box(q('#expVariant')),
    planText:p?p.textContent:'', planId:(q('#expPlan .expid')||{}).textContent||'',
    cols:[...document.querySelectorAll('#expPlan .col')].map(c=>({h:(c.querySelector('h4')||{}).textContent||'',t:c.textContent,pre:(c.querySelector('pre')||{}).textContent||'',html:(c.querySelector('pre')||{}).innerHTML||''})),
    runBtns:[...document.querySelectorAll('#expPlan button.arm')].map(b=>({t:b.textContent,disabled:b.disabled})),
    promote:[...document.querySelectorAll('button,a')].filter(b=>/promote/i.test(b.textContent)).length,
    saved:(q('#expSaved')||{}).textContent||'',
    overX:document.scrollingElement.scrollWidth-innerWidth, active:document.activeElement?document.activeElement.id:''}}"""

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

            posts.clear(); gets.clear(); state.update(saved=False, post=None)
            ctx = b.new_context(viewport={"width": w, "height": h})
            pg = ctx.new_page(); errs = []
            pg.on("pageerror", lambda e: errs.append(str(e)))
            pg.route("**/*", route)
            try:
                # E1
                pg.goto(ORIGIN + "/dev?run=" + NID); pg.wait_for_selector("#runMap .blk")
                pg.wait_for_timeout(200)
                v = pg.evaluate(PROBE)
                check("E1 Observe offers Try one change…", v["tryBtn"] and "Try one change" in v["tryText"], v["tryText"])
                check("E1 nothing about experiments is sent before it is used", not posts, posts)
                h0 = pg.evaluate("history.length")
                # E2
                pg.locator("#expOpen").click(timeout=5000)
                pg.wait_for_selector("#expForm", timeout=5000)
                pg.wait_for_timeout(200)
                v = pg.evaluate(PROBE)
                check("E2 the form opens in Observe at &exp=new", v["form"] and v["mode"] == "observe" and "exp=new" in v["url"] and not v["map"], v["url"])
                check("E2 the sentence is pre-filled with A's instruction", v["variant"] == FX["list_before"]["control"], v["variant"])
                check("E2 character count", v["count"].startswith(str(len(FX["list_before"]["control"]))) and "300" in v["count"], v["count"])
                check("E2 the reply format is shown and not editable", v["contractTag"] == "PRE" and "NOTE: <at most 40 words>" in v["contract"], v["contractTag"])
                snaps = {s["v"]: s for s in v["snaps"]}
                check("E2 AS_OF_NOW chosen; AS_OF_BASELINE disabled with its reason",
                      snaps.get("AS_OF_NOW", {}).get("checked") and snaps.get("AS_OF_BASELINE", {}).get("disabled") and "NOT AVAILABLE" in snaps.get("AS_OF_BASELINE", {}).get("why", ""), v["snaps"])
                for x in FX["list_before"]["held_constant"] + FX["list_before"]["unchanged"]:
                    check(f"E2 lists {x[:30]!r}", x in v["formText"], "")
                check("E2 saving costs 0 model requests, running 2 later", "0 model requests" in v["formText"] and "2 (A, then B)" in v["formText"], "")
                check("E2 focus is on the sentence", v["active"] == "expVariant", v["active"])
                check("E2 a GET of the form options only", not posts and "/api/development/experiments" in gets, gets)
                if tag != "phone":
                    check("E7 sentence and Save on screen", v["saveBox"] and v["saveBox"]["bottom"] <= h and v["varBox"] and v["varBox"]["top"] >= 0, (v["saveBox"], v["varBox"]))
                check("E7 no sideways scrolling (form)", v["overX"] <= 1, v["overX"])
                pg.screenshot(path=str(SHOT / f"1-form-{tag}.png"))
                # E3
                for name, text, want in [("NOTE", "Answer directly. NOTE: yes", "NOTE: or STATE:"), ("STATE", "Answer. state = keep", "NOTE: or STATE:"),
                                         ("301", "x" * 301, "300"), ("same as A", FX["list_before"]["control"], "differ"),
                                         ("two lines", "Answer.\nMore.", "one line"), ("empty", "   ", "Write")]:
                    pg.fill("#expVariant", text); pg.locator("#expSave").click(timeout=5000); pg.wait_for_timeout(150)
                    v = pg.evaluate(PROBE)
                    check(f"E3 refuses {name} before sending", want in v["err"] and not posts, (v["err"], posts))
                # E4: refused by Core → reason shown, still the form
                state["post"] = (400, {"error": "experiment refused: Core says no (fixture)"})
                pg.fill("#expVariant", "Answer the question directly " + HOSTILE)
                pg.locator("#expSave").click(timeout=5000); pg.wait_for_timeout(300)
                v = pg.evaluate(PROBE)
                check("E4 Core's refusal is shown; still the form", "Core says no" in v["err"] and v["form"] and len(posts) == 1, (v["err"], len(posts)))
                key1 = posts[0][1].get("idempotency_key")
                posts.clear(); state["post"] = None
                # E4: save (double click)
                pg.locator("#expSave").dblclick(timeout=5000)
                pg.wait_for_selector("#expPlan", timeout=5000); pg.wait_for_timeout(300)
                v = pg.evaluate(PROBE)
                check("E4 exactly one POST for a double click", len(posts) == 1, posts)
                body = posts[0][1] if posts else {}
                check("E4 the POST carries the plan", body.get("baseline_nudge_id") == NID and body.get("variant") == "Answer the question directly " + HOSTILE
                      and body.get("snapshot") == "AS_OF_NOW" and len(body.get("idempotency_key", "")) >= 8, body)
                check("E4 a retry after a refusal keeps the same idempotency key", body.get("idempotency_key") == key1, (key1, body.get("idempotency_key")))
                # E5
                check("E5 the plan replaces the form at &exp=<id>", v["plan"] and not v["form"] and ("exp=" + EXP) in v["url"], v["url"])
                check("E5 plan ID and 0 model requests", EXP in v["planId"] and "0 model requests" in v["planText"], v["planText"][:200])
                check("E5 A and B sentences", FX["list_before"]["control"] in v["planText"] and PLAN["variant"] in v["planText"], "")
                cols = v["cols"]
                check("E5 three columns: Original | A | B", [c["h"][:8] for c in cols] == ["Original", "A · unch", "B · your"], [c["h"] for c in cols])
                if len(cols) == 3:
                    check("E5 the original Note literally", cols[0]["pre"] == PLAN["original"]["text"] and "<img" not in cols[0]["html"], cols[0])
                    check("E5 A and B NOT RUN", "NOT RUN" in cols[1]["t"] and "NOT RUN" in cols[2]["t"], "")
                    check("E5 saved into the Thought: original yes, A no, B no", "Saved into the Thought: yes" in cols[0]["t"] and "Saved into the Thought: no" in cols[1]["t"]
                          and "Saved into the Thought: no" in cols[2]["t"], [c["t"][-80:] for c in cols])
                check("E5 Run A / Run B disabled", len(v["runBtns"]) == 2 and all(x["disabled"] for x in v["runBtns"]) and "Run A" in v["runBtns"][0]["t"], v["runBtns"])
                check("E5 NOT AUTHORIZED is said", "NOT AUTHORIZED" in v["planText"] and "D4" in v["planText"], "")
                check("E5 no Promote control", v["promote"] == 0, v["promote"])
                check("E8 hostile variant stays text", pg.evaluate("window.__xss") is None and pg.locator("#expPlan img").count() == 0, "")
                check("E7 no sideways scrolling (plan)", v["overX"] <= 1, v["overX"])
                pg.screenshot(path=str(SHOT / f"2-plan-{tag}.png"), full_page=True)
                # E6
                n = gets.count("/api/development/experiments/" + EXP)
                pg.reload(); pg.wait_for_selector("#expPlan"); pg.wait_for_timeout(200)
                check("E6 refresh reads the plan again", gets.count("/api/development/experiments/" + EXP) == n + 1 and EXP in pg.evaluate(PROBE)["planId"], "")
                pg.go_back(); pg.wait_for_selector("#runMap .blk"); pg.wait_for_timeout(200)
                v = pg.evaluate(PROBE)
                check("E6 Back returns to the run", v["map"] and not v["plan"] and not v["form"], v["url"])
                check("E6 the run lists its saved plan", "1 saved" in v["saved"] and EXP in v["saved"], v["saved"])
                pg.go_forward(); pg.wait_for_selector("#expPlan"); pg.wait_for_timeout(200)
                check("E6 Forward returns to the plan", EXP in pg.evaluate(PROBE)["planId"], "")
                check("E6 history: run → plan (the form was replaced, not stacked)", pg.evaluate("history.length") == h0 + 1, (h0, pg.evaluate("history.length")))
                pg.goto(ORIGIN + "/dev?run=" + NID + "&exp=" + EXP); pg.wait_for_selector("#expPlan")
                check("E6 direct URL opens the plan", EXP in pg.evaluate(PROBE)["planId"], "")
                pg.goto(ORIGIN + "/dev?run=" + NID + "&exp=exp_../../x"); pg.wait_for_timeout(500)
                v = pg.evaluate(PROBE)
                check("E6 a bad experiment ID is refused in the page", not v["plan"] and "not an experiment ID" in pg.evaluate("document.body.innerText"), "")
                pg.goto(ORIGIN + "/dev?run=" + NID); pg.wait_for_selector("#runMap .blk"); pg.wait_for_timeout(300)
                pg.screenshot(path=str(SHOT / f"0-run-with-try-and-saved-plan-{tag}.png"))
                pg.locator("#runSavedList a").first.click(timeout=5000); pg.wait_for_selector("#expPlan")
                check("E6 the saved plan link opens it", EXP in pg.evaluate(PROBE)["planId"], "")
            except Exception as e:
                failures.append(f"[{tag}] aborted: {str(e).splitlines()[0]}")
            check("E8 no other POSTs", all(p_[0] == "/api/development/experiments" for p_ in posts), posts)
            check("E8 no Brain Builder read in Observe", "/api/brain" not in gets, sorted(set(gets)))
            check("E8 no page errors", not errs, errs)
            ctx.close()
    finally:
        b.close()
for f in failures:
    print("FAIL:", f)
print(f"{passed}/{passed + len(failures)} Development experiment-plan checks PASS across 1280/860/390")
sys.exit(1 if failures else 0)
