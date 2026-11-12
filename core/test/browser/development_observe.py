#!/usr/bin/env python3
"""Stage 3A UX repair: one Development workspace with OBSERVE (what TASS actually did, read only) and DESIGN
(the existing Brain Builder, advisory). Reproduces the owner's failed walkthrough: "How this Note was produced"
opened twelve full-height audit cards followed by the unrelated Brain Builder canvas.

NOT the owner's machine, NOT Nemotron. Payloads are cognition_stage3a_run.json (complete, verified) and
cognition_stage3a_unreconciled.json (gaps), both Core's real DevelopmentRun output (checked by
TestDevelopmentRun*BrowserFixtureIsCoreOutput). Hostile text is substituted for the question and Note.

O1  Cognition → "How this Note was produced" opens Observe: the run, no Brain Builder, no design controls,
    and no /api/brain read while observing
O2  first viewport: the question, a plain summary, the saved Note, the map and the next action; at 1280×900
    the page does not scroll; at ~860 the question, outcome and map are on screen; at 390 nothing scrolls sideways
O3  the 12 original stages, IDs, revisions and legend sit in collapsed technical evidence
O4  map blocks and connections carry exactly Core's statuses, in Core's order
O5  connection truth: a connection Core did not establish is drawn and labelled as a gap, with what is missing
O6  selecting a block or a connection opens the inspector: what happened, evidence status, what establishes
    the relationship, what is missing, technical details collapsed
O7  keyboard: arrow keys move through the map, Enter selects, focus is visible
O8  direct URL, refresh, Back and Forward keep the mode; switching to Design is one history entry
O9  Design keeps the existing editor (library, canvas, inspector, Check, Save, Teach & test, Junction lab)
    behind an advisory note; plain /dev opens Design as before
O10 refused / not found / unavailable: an error in Observe, no map, nothing invented
O11 unreconciled run: no "verified by Core", no "exactly one", no foreign bytes, no Note shown as the outcome
O12 zero POSTs; only the expected GETs; hostile text stays text
    python3 development_observe.py
"""
import copy
import json
import sys
from pathlib import Path
from urllib.parse import urlparse
from playwright.sync_api import sync_playwright
from cognition_scope_fixture import scoped

ROOT = Path(__file__).resolve().parents[2]
COG = (ROOT / "cmd/tass-core/cognition.html").read_text()
DEV = (ROOT / "cmd/tass-core/dev.html").read_text()
HERE = Path(__file__).resolve().parent
RUN = json.loads((HERE / "cognition_stage3a_run.json").read_text())
UNREC = json.loads((HERE / "cognition_stage3a_unreconciled.json").read_text())
ORIGIN = "http://127.0.0.1:8796"
NOW = "2026-11-12T10:00:00Z"
NID, REQ, TID = RUN["nudge_id"], RUN["request_id"], RUN["thought_id"]
HOSTILE = '<img src=x onerror="window.__xss=1"><b>bold?</b>'
QUESTION = "Why does the pulse stall? " + HOSTILE
NOTE = "Watch the queue depth next " + HOSTILE
RUN["question"], RUN["note"] = QUESTION, NOTE
UNREC = copy.deepcopy(UNREC); UNREC["nudge_id"] = NID
SHOT = Path("/tmp/observe"); SHOT.mkdir(exist_ok=True)

LIB = {"identity_entity_id": "ent_0123", "engine_entity_id": "eng_1", "identity_name": "Mira", "engine_commit": "c" * 40,
       "brain_revision": 7, "note": "", "cycles": [], "outcomes": [], "library": [],
       "thoughts": [{"id": "thought:" + TID, "kind": "thought", "title": "observe (fixture)", "summary": "q", "source": "Thought Engine: persistent Thought",
                     "source_id": TID, "status": "active", "verification": "OBSERVED", "recorded_at": NOW, "related_cycle_ids": [], "fields": [],
                     "lifecycle": "active", "notes_count": 1, "latest_output": {"kind": "DEVELOPED_NOTE", "at": NOW, "by": "engine"}}]}
THOUGHT = {"thought_id": TID, "question": "q", "state": "active", "history_total": 1, "history": [{"at": NOW, "kind": "formed"}],
           "developments": [{"request_id": REQ, "at": NOW, "persistence": "NOTE_AND_STEP_CORRELATED", "step": {"at": NOW, "kind": "developed"},
                             "note": {"at": NOW, "text": "Note", "by": "Nemotron", "suggested_state": "", "suggested_state_applied": False},
                             "meaning": "Saved.", "source_nudge_id": NID, "nudge_receipt": "COMPLETED_VERIFIED"}],
           "developments_total": 1, "traces": [], "related_cycle_ids": []}
BRAIN = {"blocks": [{"id": "lesson:a", "kind": "lesson", "title": "Pulse lesson", "source": "lessons/a.md", "revision": "r1",
                     "purpose": "p", "status": "NEW", "runnable": "no"}], "relations": {},
         "graph": {"revision": 3, "actor": "owner", "saved_at": NOW, "nodes": [{"id": "n1", "block_id": "lesson:a", "pinned": "r1", "x": 40, "y": 40}],
                   "edges": [], "notes": []}, "report": {"status": "OK", "summary": "fine", "execution": "Checks never run anything.", "checks": []}, "history": []}

state = {"run": (200, RUN)}
posts, gets = [], []


def route(req):
    r = req.request
    path = urlparse(r.url).path
    if path == "/cognition" and r.method == "GET":
        req.fulfill(status=200, content_type="text/html", body=COG.replace("__PAGE_TOKEN__", "t")); return
    if path == "/dev" and r.method == "GET":
        req.fulfill(status=200, content_type="text/html", body=DEV.replace("__PAGE_TOKEN__", "t")); return
    if r.method != "GET":
        posts.append((r.method, path)); req.fulfill(status=405); return
    gets.append(path)
    if path.startswith("/api/development/runs/"):
        code, body = state["run"]
        req.fulfill(status=code, content_type="application/json" if isinstance(body, dict) else "text/plain",
                    body=json.dumps(body) if isinstance(body, dict) else body, headers={"X-Cognition-Identity": "ent_0123"}); return
    if path == "/api/brain":
        req.fulfill(status=200, content_type="application/json", body=json.dumps(BRAIN)); return
    if path == "/api/status":
        req.fulfill(status=200, content_type="application/json", body="{}"); return
    sreq = scoped(req)
    if path == "/api/cognition/library":
        sreq.fulfill(status=200, content_type="application/json", body=json.dumps(LIB)); return
    if path == "/api/cognition/thoughts/" + TID:
        sreq.fulfill(status=200, content_type="application/json", body=json.dumps(THOUGHT)); return
    req.fulfill(status=404, body="")


PROBE = """()=>{const q=s=>document.querySelector(s), vis=e=>!!e&&e.checkVisibility({visibilityProperty:true})&&e.getClientRects().length>0;
  const box=e=>{if(!vis(e))return null;const r=e.getBoundingClientRect();return {top:r.top,bottom:r.bottom,left:r.left,right:r.right}};
  const v=q('#runView'), ins=q('#runInsp'), te=q('#techEvidence');
  return {mode:document.body.dataset.mode||'', observePressed:(q('#modeObserve')||{getAttribute:()=>null}).getAttribute('aria-pressed'),
    runVisible:vis(v), shell:vis(q('.shell')), bottom:vis(q('.bottom')), llab:vis(q('#llab')), jlab:vis(q('#jlab')),
    designCtl:['#save','#check','#undo','#llabBtn','#jlabBtn','#change','#rev'].filter(s=>vis(q(s))),
    designNote:vis(q('#designNote')), title:(q('#runTitle')||{}).textContent||'',
    question:(q('#runQuestion .val')||{}).textContent||'', questionHTML:(q('#runQuestion .val')||{}).innerHTML||'',
    story:(q('#runStory')||{}).textContent||'', outcome:(q('#runOutcome')||{}).textContent||'',
    outcomeNote:(q('#runOutcome pre')||{}).textContent||'',
    blocks:[...document.querySelectorAll('#runMap .blk')].map(b=>({id:b.dataset.block,status:b.dataset.status,text:b.textContent,pressed:b.getAttribute('aria-pressed')})),
    links:[...document.querySelectorAll('#runMap .lnk')].map(b=>({from:b.dataset.from,to:b.dataset.to,status:b.dataset.status,text:b.textContent,cls:b.className,pressed:b.getAttribute('aria-pressed')})),
    extra:[...document.querySelectorAll('#runExtra .blk')].map(b=>({id:b.dataset.block,status:b.dataset.status})),
    insp:ins?ins.textContent:'', inspHead:(q('#runInsp h3')||{}).textContent||'', inspDetails:ins?[...ins.querySelectorAll('details')].map(d=>d.open):[],
    techOpen:te?te.open:null, techStages:te?te.querySelectorAll('li.st').length:0, techStagesVisible:te?[...te.querySelectorAll('li.st')].filter(vis).length:0,
    techText:te?te.textContent:'', next:(q('#runNext a')||{}).textContent||'', nextHref:q('#runNext a')?q('#runNext a').getAttribute('href'):'',
    err:(q('#runView .err')||{}).textContent||'',
    boxes:{title:box(q('#runTitle')),question:box(q('#runQuestion')),story:box(q('#runStory')),outcome:box(q('#runOutcome')),map:box(q('#runMap')),next:box(q('#runNext'))},
    scrollH:document.scrollingElement.scrollHeight, innerH:innerHeight, overX:document.scrollingElement.scrollWidth-innerWidth,
    body:document.body.innerText}}"""

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
                    failures.append(f"[{tag}] {name}: {str(detail)[:600]}")

            posts.clear(); gets.clear(); state["run"] = (200, RUN)
            ctx = b.new_context(viewport={"width": w, "height": h})
            pg = ctx.new_page(); errs = []
            pg.on("pageerror", lambda e: errs.append(str(e)))
            pg.route("**/*", route)

            try:
                # O1: from the Note in Cognition straight into Observe.
                pg.goto(ORIGIN + "/cognition"); pg.wait_for_selector(".tile")
                pg.get_by_role("button", name="Inspect observe (fixture)").click()
                pg.wait_for_selector("#dev-" + REQ + " a.how")
                h0 = pg.evaluate("history.length")
                pg.locator("#dev-" + REQ + " a.how").click(timeout=5000)
                try:
                    pg.wait_for_selector("#runMap .blk", timeout=4000)
                except Exception:
                    pass
                pg.wait_for_timeout(300)
                v = pg.evaluate(PROBE)
                check("O1 lands on /dev?run=<nudge>", pg.url == ORIGIN + "/dev?run=" + NID, pg.url)
                check("O1 opens in Observe", v["mode"] == "observe" and v["observePressed"] == "true", (v["mode"], v["observePressed"]))
                check("O1 the run is shown", v["runVisible"] and v["title"] == "How this Note was produced", v["title"])
                check("O1 no Brain Builder canvas, bottom panels or labs below Observe", not v["shell"] and not v["bottom"] and not v["llab"] and not v["jlab"],
                      (v["shell"], v["bottom"], v["llab"], v["jlab"]))
                check("O1 no design controls in Observe", not v["designCtl"], v["designCtl"])
                check("O1 Observe does not read the Brain Builder", "/api/brain" not in gets, sorted(set(gets)))

                # O2: the first viewport.
                check("O2 the question, literally", v["question"] == QUESTION and "<img" not in v["questionHTML"], v["question"])
                check("O2 plain summary carries Core's result", RUN["result"] in v["story"] and "7 of 7" in v["story"] and "6 of 6" in v["story"], v["story"])
                check("O2 the saved Note, literally", v["outcomeNote"] == NOTE, v["outcome"])
                check("O2 next action back to the same Note", v["next"] == "Back to this Note in Cognition" and v["nextHref"] == "/cognition#thought=" + TID + "&dev=" + REQ,
                      (v["next"], v["nextHref"]))
                bx = v["boxes"]
                if tag == "desktop":
                    check("O2 1280×900: no full-page scrolling", v["scrollH"] <= v["innerH"] + 1, (v["scrollH"], v["innerH"]))
                if tag in ("desktop", "half"):
                    for k in ["title", "question", "story", "outcome", "map", "next"]:
                        check(f"O2 {k} on screen without scrolling", bx[k] and bx[k]["top"] >= 0 and bx[k]["bottom"] <= h + 1 and bx[k]["right"] <= w + 1, bx[k])
                else:
                    for k in ["title", "question"]:
                        check(f"O2 phone: {k} on screen", bx[k] and bx[k]["top"] >= 0 and bx[k]["bottom"] <= h + 1, bx[k])
                check("O2 no sideways scrolling", v["overX"] <= 1, v["overX"])
                pg.screenshot(path=str(SHOT / f"1-observe-open-{tag}.png"))

                # O3: technical evidence collapsed, complete.
                check("O3 technical evidence collapsed", v["techOpen"] is False and v["techStagesVisible"] == 0, (v["techOpen"], v["techStagesVisible"]))
                check("O3 holds the 12 stages", v["techStages"] == 12, v["techStages"])
                for want in [NID, REQ, RUN["source_revision"], RUN["engine_commit"][:12], "NOT PART OF THIS PATH", "CODE-DEFINED"]:
                    check(f"O3 technical evidence keeps {want!r}", want in v["techText"], "")

                # O4: Core's statuses, Core's order.
                check("O4 blocks = Core's", [(x["id"], x["status"]) for x in v["blocks"]] == [(x["id"], x["status"]) for x in RUN["blocks"]], v["blocks"])
                check("O4 links = Core's", [(x["from"], x["to"], x["status"]) for x in v["links"]] == [(x["from"], x["to"], x["status"]) for x in RUN["links"]], v["links"])
                check("O4 the paths not taken are listed", [(x["id"], x["status"]) for x in v["extra"]] == [(x["id"], x["status"]) for x in RUN["extra"]], v["extra"])
                for x in v["blocks"]:
                    title = next(bb["title"] for bb in RUN["blocks"] if bb["id"] == x["id"])
                    check(f"O4 block {x['id']} names itself and its status", title in x["text"] and x["status"] in x["text"], x["text"])

                # O6: select a block.
                pg.locator('#runMap .blk[data-block="model_reply"]').click(timeout=5000)
                v = pg.evaluate(PROBE)
                mr = next(x for x in RUN["blocks"] if x["id"] == "model_reply")
                check("O6 the block is selected", [x["id"] for x in v["blocks"] if x["pressed"] == "true"] == ["model_reply"], v["blocks"])
                check("O6 inspector: what happened", v["inspHead"].startswith("Model reply") and mr["summary"] in v["insp"], v["insp"][:300])
                check("O6 inspector: evidence status and its meaning", "RECORDED" in v["insp"] and "Read from a record of this run" in v["insp"], v["insp"][:300])
                check("O6 inspector: technical details collapsed", v["inspDetails"] and not any(v["inspDetails"]), v["inspDetails"])
                if tag != "phone":
                    check("O6 selecting does not make the page scroll", v["scrollH"] <= v["innerH"] + 1 or tag == "half", (v["scrollH"], v["innerH"]))
                pg.screenshot(path=str(SHOT / f"2-selected-block-{tag}.png"))
                pg.locator("#runInsp details summary").first.click()
                check("O6 technical details open on request and stay literal", pg.locator("#runInsp img").count() == 0, "")

                # O6: select a connection.
                pg.locator('#runMap .lnk[data-from="model_reply"]').click(timeout=5000)
                v = pg.evaluate(PROBE)
                ln = next(x for x in RUN["links"] if x["from"] == "model_reply")
                check("O6 the connection is selected", [x["from"] for x in v["links"] if x["pressed"] == "true"] == ["model_reply"] and not [x for x in v["blocks"] if x["pressed"] == "true"], v["links"])
                check("O6 connection inspector: what establishes it", "Established by" in v["insp"] and ln["basis"] in v["insp"] and "Model reply" in v["inspHead"] and "Saved Note" in v["inspHead"], v["insp"][:400])
                check("O6 connected: nothing listed as missing", "Missing" not in v["insp"], v["insp"][:400])
                pg.screenshot(path=str(SHOT / f"3-selected-connection-{tag}.png"))

                # O7: keyboard.
                pg.locator('#runMap .blk[data-block="thought"]').focus()
                pg.keyboard.press("ArrowDown")
                f = pg.evaluate("(()=>{const e=document.activeElement;const s=getComputedStyle(e);return {cls:e.className,from:e.dataset.from||'',block:e.dataset.block||'',ow:s.outlineWidth,os:s.outlineStyle}})()")
                check("O7 ArrowDown moves to the next map item", f["from"] == "thought", f)
                pg.keyboard.press("ArrowDown")
                f = pg.evaluate("(()=>{const e=document.activeElement;const s=getComputedStyle(e);return {block:e.dataset.block||'',ow:s.outlineWidth,os:s.outlineStyle}})()")
                check("O7 then the next block", f["block"] == "attention", f)
                check("O7 focus is visible", f["os"] != "none" and f["ow"] not in ("0px", ""), f)
                pg.keyboard.press("Enter")
                v = pg.evaluate(PROBE)
                check("O7 Enter selects", [x["id"] for x in v["blocks"] if x["pressed"] == "true"] == ["attention"] and v["inspHead"].startswith("Your attention"), v["inspHead"])
                pg.keyboard.press("ArrowUp"); pg.keyboard.press("ArrowUp"); pg.keyboard.press("ArrowUp")
                check("O7 ArrowUp stops at the first item", pg.evaluate("document.activeElement.dataset.block") == "thought", "")

                # O3 opened.
                pg.locator("#techEvidence > summary").click(timeout=5000)
                v = pg.evaluate(PROBE)
                check("O3 technical evidence opens to the 12 stages", v["techOpen"] and v["techStagesVisible"] == 12, v["techStagesVisible"])
                check("O3 hostile text stays text", pg.evaluate("window.__xss") is None and pg.locator("#runView img").count() == 0
                      and pg.locator("#runView b", has_text="bold?").count() == 0, "")
                pg.screenshot(path=str(SHOT / f"4-technical-evidence-{tag}.png"), full_page=True)

                # O8 + O9: Design, Back, Forward, refresh.
                n_run = gets.count("/api/development/runs/" + NID)
                pg.locator("#modeDesign").click(timeout=5000)
                pg.wait_for_selector(".shell", state="visible")
                pg.wait_for_timeout(300)
                v = pg.evaluate(PROBE)
                check("O8 Design is a URL", "mode=design" in pg.url and "run=" + NID in pg.url, pg.url)
                check("O9 Design shows the existing editor", v["mode"] == "design" and v["shell"] and v["bottom"] and not v["runVisible"], (v["mode"], v["shell"], v["runVisible"]))
                check("O9 Design keeps its controls", set(v["designCtl"]) >= {"#save", "#check", "#llabBtn", "#jlabBtn"}, v["designCtl"])
                check("O9 Design is advisory and says so", v["designNote"] and "advisory" in v["body"].lower(), "")
                check("O9 Design loads the Brain Builder once, on demand", gets.count("/api/brain") == 1, gets.count("/api/brain"))
                check("O9 the existing canvas shows the saved block", pg.locator("#canvas .node").count() == 1 and "revision 3" in pg.locator("#rev").text_content(), "")
                pg.screenshot(path=str(SHOT / f"5-design-mode-{tag}.png"))
                pg.go_back(); pg.wait_for_timeout(300)
                v = pg.evaluate(PROBE)
                check("O8 Back returns to Observe with the run", v["mode"] == "observe" and v["runVisible"] and len(v["blocks"]) == 7 and not v["shell"], (v["mode"], len(v["blocks"])))
                pg.go_forward(); pg.wait_for_timeout(300)
                v = pg.evaluate(PROBE)
                check("O8 Forward returns to Design", v["mode"] == "design" and v["shell"], v["mode"])
                pg.reload(); pg.wait_for_selector(".shell", state="visible"); pg.wait_for_timeout(200)
                check("O8 refresh keeps Design", pg.evaluate(PROBE)["mode"] == "design", "")
                pg.locator("#modeObserve").click(); pg.wait_for_selector("#runMap .blk")
                check("O8 Observe from Design is one more entry, same run", pg.url == ORIGIN + "/dev?run=" + NID, pg.url)
                pg.reload(); pg.wait_for_selector("#runMap .blk")
                check("O8 refresh keeps Observe and reads the run again", pg.evaluate(PROBE)["mode"] == "observe" and gets.count("/api/development/runs/" + NID) > n_run, "")
                pg.goto(ORIGIN + "/dev?run=" + NID); pg.wait_for_selector("#runMap .blk")
                check("O8 direct URL opens Observe", pg.evaluate(PROBE)["mode"] == "observe", "")

                # O5 + O11: the run with gaps.
                state["run"] = (200, UNREC)
                pg.reload(); pg.wait_for_selector("#runMap .blk")
                v = pg.evaluate(PROBE)
                check("O4 gap run: blocks = Core's", [(x["id"], x["status"]) for x in v["blocks"]] == [(x["id"], x["status"]) for x in UNREC["blocks"]], v["blocks"])
                check("O4 gap run: links = Core's", [(x["from"], x["to"], x["status"]) for x in v["links"]] == [(x["from"], x["to"], x["status"]) for x in UNREC["links"]], v["links"])
                for x in v["links"]:
                    if x["status"] == "CONNECTED":
                        check(f"O5 {x['from']}→{x['to']} drawn connected", "connected" in x["text"].lower() and "gap" not in x["cls"], x)
                    else:
                        check(f"O5 {x['from']}→{x['to']} drawn as a gap", "gap" in x["cls"] and x["status"].lower() in x["text"].lower() and "connected" != x["text"].strip().lower(), x)
                pg.locator('#runMap .lnk[data-from="model_request"]').click(timeout=5000)
                v = pg.evaluate(PROBE)
                ln = next(x for x in UNREC["links"] if x["from"] == "model_request")
                check("O5 a gap says what would establish it and what is missing",
                      "Would be established by" in v["insp"] and "Established by" not in v["insp"].replace("Would be established by", "") and ln["basis"] in v["insp"]
                      and ln["missing"] in v["insp"] and "NOT ESTABLISHED" in v["insp"], v["insp"][:500])
                pg.screenshot(path=str(SHOT / f"6-gap-connection-{tag}.png"))
                check("O11 no 'verified by Core', no 'exactly one'", "verified by Core" not in v["body"] and "exactly one" not in v["body"], "")
                check("O11 Core's result line in the summary", UNREC["result"] in v["story"] and "6 of 6" not in v["story"], v["story"])
                check("O11 no Note shown as the outcome", v["outcomeNote"] == "" and "2 Notes" in v["outcome"], v["outcome"])
                check("O11 no foreign bytes", "FOREIGN" not in pg.content(), "")
                check("O11 no sideways scrolling", v["overX"] <= 1, v["overX"])
                state["run"] = (200, RUN)

                # O10: refusals.
                for name, resp, want in [("403", (403, "denied"), "Identity did not allow"), ("404", (404, "none"), "There is no processed nudge"),
                                         ("503", (503, "down"), "not available right now")]:
                    state["run"] = (200, RUN); pg.goto(ORIGIN + "/dev?run=" + NID); pg.wait_for_selector("#runMap .blk")
                    state["run"] = resp; pg.reload(); pg.wait_for_selector("#runView .err")
                    v = pg.evaluate(PROBE)
                    check(f"O10 {name}: error in Observe, no map, nothing invented", v["mode"] == "observe" and want in v["err"] and not v["blocks"] and not v["links"]
                          and NOTE not in v["body"] and not v["shell"], v["err"])
                state["run"] = (200, RUN)

                # O9: plain /dev is Design, as before.
                n0 = len(gets)
                pg.goto(ORIGIN + "/dev"); pg.wait_for_selector(".shell", state="visible"); pg.wait_for_timeout(300)
                v = pg.evaluate(PROBE)
                late = gets[n0:]
                check("O9 plain /dev opens Design with no run", v["mode"] == "design" and not v["runVisible"] and v["shell"] and "/api/brain" in late
                      and not [g for g in late if g.startswith("/api/development/")], (v["mode"], late))

                check("O12 zero POSTs", not posts, posts)
                allowed = ("/api/development/runs/", "/api/brain", "/api/status", "/api/cognition/library", "/api/cognition/thoughts/", "/api/cognition/pulse", "/api/nudges")
                check("O12 only expected GETs", all(g.startswith(allowed) for g in gets), sorted(set(gets)))
                check("O12 no page errors", not errs, errs)
            except Exception as e:
                failures.append(f"[{tag}] aborted: {str(e).splitlines()[0]}")
            ctx.close()
    finally:
        b.close()
for f in failures:
    print("FAIL:", f)
print(f"{passed}/{passed + len(failures)} Development Observe/Design checks PASS across 1280/860/390")
sys.exit(1 if failures else 0)
