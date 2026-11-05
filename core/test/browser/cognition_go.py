#!/usr/bin/env python3
"""Isolated Chromium Cognition interaction test. Fixtures, NOT real owner/Nemotron.
Does not start Core, change settings, or execute any operation."""
import json
from pathlib import Path
from playwright.sync_api import sync_playwright

HTML=(Path(__file__).resolve().parents[2]/"cmd/tass-core/cognition.html").read_text()
ORIGIN="http://127.0.0.1:8795"
NOW="2026-11-05T19:10:00Z"
LIB={
 "identity_entity_id":"ent_0123","engine_entity_id":"eng_1","identity_name":"Mira",
 "engine_commit":"cab101c3d1d62953d99dd929e4d1b11ad55eb750","brain_revision":7,
 "note":"","thoughts":[{"id":"thought:thought_1","kind":"thought","title":"Real persistent Thought",
   "summary":"Canonical question","source":"Thought Engine: persistent Thought","source_id":"thought_1",
   "status":"active","verification":"OBSERVED","recorded_at":NOW,"related_cycle_ids":["tp_1"],
   "fields":[{"name":"Canonical Thought ID","value":"thought_1"}]}],
 "cycles":[{"id":"cycle:tp_1","kind":"cycle","title":"Cycle tp_1",
   "summary":"linked","source":"Thought Engine: cycle aggregate","source_id":"tp_1",
   "status":"evaluated","verification":"VERIFIED-LINK","recorded_at":NOW,"fields":[]},
  {"id":"cycle:tp_unlinked","kind":"cycle","title":"Cycle tp_unlinked",
   "summary":"unlinked","source":"Thought Engine: cycle aggregate","source_id":"tp_unlinked",
   "status":"evaluated","verification":"UNLINKED","recorded_at":NOW,"fields":[]}],
 "outcomes":[{"id":"outcome:tp_1","kind":"outcome","title":"Outcome NO_ACTION","summary":"not execution",
   "source":"Thought Engine: executor-report","source_id":"tp_1","status":"NO_ACTION",
   "verification":"APP-REPORTED","recorded_at":NOW,"fields":[]}],
 "library":[{"id":"cell:cell1","kind":"cell","title":"Lesson cell","summary":"NOT EXECUTABLE",
   "source":"Core: BrainBlocks","source_id":"cell1","status":"DISCOVERED","verification":"APP-REPORTED","fields":[]}]
}
THOUGHT={"thought_id":"thought_1","topic":"Real persistent Thought","question":"Canonical question",
 "state":"active","history":[{"at":NOW,"kind":"formed","event_id":"evt_1"}],
 "traces":[{"trace_id":"trace_1","verified":True}],"related_cycle_ids":["tp_1"]}
TIMELINE={"cycle_id":"tp_1","thought_id":"thought_1","link_status":"VERIFIED-LINK",
 "source":"Thought Engine: cycles.jsonl","engine_commit":LIB["engine_commit"],"state":"evaluated",
 "steps":[{"offset":0,"at":NOW,"stage":"received","source":"cycles.jsonl","status":"OBSERVED"},
          {"offset":123,"at":NOW,"stage":"proposal","source":"cycles.jsonl","status":"PROPOSED",
           "action":"NONE","prediction_state":"LOCKED"},
          {"offset":231,"at":NOW,"stage":"outcome","source":"cycles.jsonl","status":"APP-REPORTED",
           "outcome_status":"NO_ACTION"}],
 "missing":["spark","evaluation"],"truncated":False,"partial":False}
REPORT="# Cognition — sanitized recorded cycle\nCycle: tp_1\nOutcome: NO_ACTION\nNo execution evidence.\n"
def fake_route(req,fail=False):
    url=req.request.url
    if req.request.method!="GET":
        req.fulfill(status=405,body="NO ACTION");return
    if url==ORIGIN+"/cognition":
        req.fulfill(status=200,content_type="text/html",body=HTML.replace("__PAGE_TOKEN__","fixture-token"));return
    if "/api/cognition/library" in url:
        req.fulfill(status=503 if fail else 200,content_type="application/json",
                    body=json.dumps({"error":"unavailable"} if fail else LIB));return
    if url.endswith("/api/cognition/thoughts/thought_1"):
        req.fulfill(status=200,content_type="application/json",body=json.dumps(THOUGHT));return
    if url.endswith("/api/cognition/cycles/tp_1/timeline"):
        req.fulfill(status=200,content_type="application/json",body=json.dumps(TIMELINE));return
    if "/api/cognition/cycles/tp_1/export" in url:
        req.fulfill(status=200,content_type="text/markdown",body=REPORT);return
    if "/dev?" in url:
        req.fulfill(status=200,content_type="text/html",body="<title>Development review only</title>");return
    req.fulfill(status=404,body="Not found")
with sync_playwright() as p:
    browser=p.chromium.launch(headless=True,args=["--no-sandbox"])
    try:
        pg=browser.new_page(accept_downloads=True)
        errors=[];methods=[]
        pg.on("pageerror",lambda e:errors.append(str(e)))
        pg.on("request",lambda r:methods.append(r.method))
        pg.add_init_script("Object.defineProperty(navigator,'clipboard',{configurable:true,value:{writeText:async(text)=>{window.__copied=text;}}});")
        pg.route("**/*",fake_route)
        pg.goto(ORIGIN+"/cognition")
        pg.wait_for_timeout(800)
        print("DEBUG page",pg.url,"mode",pg.locator("#mode").all_inner_texts(),
              "banner",pg.locator("#banner").all_inner_texts(),
              "errors",errors,"requests",methods,flush=True)
        pg.wait_for_selector(".tile",timeout=5000)
        assert pg.locator(".tile").count()==5
        assert "LIVE / READ ONLY" in pg.locator("#mode").inner_text()
        assert "Persistent thoughts" in pg.locator("#shelves").inner_text()
        pg.get_by_role("button",name="Thoughts",exact=True).click()
        assert pg.locator(".tile").count()==1
        pg.get_by_role("button",name="Inspect Real persistent Thought").click()
        pg.get_by_role("button",name="Inspect recorded details").click()
        assert "Canonical question" in pg.locator("#thoughtDetail").inner_text()
        pg.get_by_role("button",name="Open cycle tp_1").click()
        pg.get_by_role("button",name="Step",exact=True).click()
        pg.wait_for_function("document.querySelector('#thoughtDetail').textContent.includes('received')")
        pg.get_by_role("button",name="Play",exact=True).click()
        pg.wait_for_selector("#pause:not(.hidden)")
        pg.get_by_role("button",name="Pause",exact=True).click()
        pg.get_by_role("button",name="View Logs").click()
        pg.wait_for_function("document.querySelector('#thoughtDetail').textContent.includes('offset 231')")
        pg.get_by_role("button",name="Copy Evidence").click()
        pg.wait_for_function("window.__copied && window.__copied.includes('tp_1')")
        assert "No execution evidence" in pg.evaluate("window.__copied")
        with pg.expect_download() as d:
            pg.get_by_role("button",name="Download Sanitized Markdown").click()
        assert d.value.suggested_filename=="cognition-tp_1.md"
        pg.get_by_role("button",name="Try adjustment in Development").click()
        pg.wait_for_url("**/dev?*")
        assert "cognition_ref=cycle%3Atp_1" in pg.url and "brain_revision=7" in pg.url
        assert all(m=="GET" for m in methods),methods
        assert not errors,errors
        pg.close()
        print("PASS: real Go-shaped thought/cycle shelves, selection, playback, view log, copy/export, Development reference; zero POST")

        pg=browser.new_page()
        pg.route("**/*",lambda route:fake_route(route,True))
        pg.goto(ORIGIN+"/cognition")
        pg.wait_for_function("document.querySelector('#mode').textContent.includes('EVIDENCE UNAVAILABLE')")
        assert pg.locator(".tile").count()==0
        assert "No demo records substituted" in pg.locator("#banner").inner_text()
        pg.close()
        print("PASS: unavailable live source fails closed, no fake thoughts")
    finally:
        browser.close()
print("2/2 browser fixture tests PASS; real owner machine NOT RUN")
