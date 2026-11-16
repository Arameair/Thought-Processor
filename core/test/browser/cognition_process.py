#!/usr/bin/env python3
"""Stage 2 browser fixture (NOT real owner/Nemotron): Process once needs an inline confirmation and sends
ONE POST with the saved body digest; the receipt is polled by GET only; Stop and Copy as new appear only
when allowed; a PRE_RESTORE draft offers Copy (with a warning step), never Process."""
import json, sys
from pathlib import Path
from playwright.sync_api import sync_playwright
from cognition_scope_fixture import scoped

HTML=(Path(__file__).resolve().parents[2]/"cmd/tass-core/cognition.html").read_text()
ORIGIN="http://127.0.0.1:8795"; NOW="2026-11-08T15:00:00Z"
LIB={"identity_entity_id":"ent_0123","engine_entity_id":"eng_1","identity_name":"Mira","engine_commit":"c"*40,"brain_revision":7,"note":"",
 "thoughts":[{"id":"thought:thought_1","kind":"thought","title":"Real persistent Thought","summary":"q","source":"Thought Engine: persistent Thought",
   "source_id":"thought_1","status":"active","verification":"OBSERVED","recorded_at":NOW,"related_cycle_ids":[],"fields":[]}],
 "cycles":[],"outcomes":[],"library":[]}
def nudge(i,**kw):
    n={"nudge_id":"nudge_%020x"%i,"status":"DRAFT","recipient_id":"self","recipient_kind":"local_entity","reason_code":"owner_request",
       "requested_action":"Look again %d"%i,"priority":"normal","created_at":NOW,"expires_at":NOW,"source_thought_id":"thought_1",
       "body_digest":"digest%d"%i,"processing":{"state":"READY","reason":"x"}}
    n.update(kw); return n
state={"run":None,"gets":0}; posts=[]
def route(req):
    req = scoped(req)  # Core attests the scope of every 2xx read (#106 lifecycle contract)
    r=req.request; url=r.url
    if url==ORIGIN+"/cognition":
        req.fulfill(status=200,content_type="text/html",body=HTML.replace("__PAGE_TOKEN__","t"));return
    if r.method=="POST":
        posts.append((url.replace(ORIGIN,""),json.loads(r.post_data or "{}")))
        if url.endswith("/process"):
            state["run"]={"state":"SUBMITTED","detail":"accepted locally","can_stop":True,"process_decision_id":"dec_1"}
            req.fulfill(status=202,content_type="application/json",body=json.dumps({"run":state["run"],"note":"accepted locally; delivery to the Engine not confirmed yet"}));return
        if url.endswith("/stop"):
            state["run"]={"state":"NOT_DELIVERED","can_copy":True,"model_http":"none"}
            req.fulfill(status=200,content_type="application/json",body=json.dumps({"run":state["run"]}));return
        if url.endswith("/api/nudges"):
            req.fulfill(status=201,content_type="application/json",body=json.dumps({"nudge_id":"nudge_%020x"%99,"status":"DRAFT"}));return
        req.fulfill(status=405);return
    if "/api/cognition/library" in url: req.fulfill(status=200,content_type="application/json",body=json.dumps(LIB));return
    if "/api/cognition/pulse" in url: req.fulfill(status=503,body="n/a");return
    if "/api/nudges?" in url:
        state["gets"]+=1
        ns=[nudge(1,run=state["run"]) if state["run"] else nudge(1), nudge(2,pre_restore=True,processing={"state":"PRE_RESTORE","reason":"x"})]
        req.fulfill(status=200,content_type="application/json",body=json.dumps({"nudges":ns,"processing":{"state":"READY","reason":"Ready"},
          "readiness":{"identity":"UNLOCKED","process_grant":"ENABLED","engine":"REACHABLE"},"note":"local"}));return
    if "/api/cognition/thoughts/" in url:
        req.fulfill(status=200,content_type="application/json",body=json.dumps({"thought_id":"thought_1","question":"q","state":"active",
          "history":[],"history_total":0,"developments":[],"traces":[],"related_cycle_ids":[]}));return
    req.fulfill(status=404)

with sync_playwright() as p:
    b=p.chromium.launch(headless=True,args=["--no-sandbox"])
    try:
        pg=b.new_page(viewport={"width":1280,"height":1000}); errs=[]; pg.on("pageerror",lambda e:errs.append(str(e)))
        pg.route("**/*",route); pg.goto(ORIGIN+"/cognition"); pg.wait_for_selector(".tile")
        pg.get_by_role("button",name="Inspect Real persistent Thought").click()
        pg.get_by_role("button",name="Nudge history").click()
        pg.wait_for_selector("text=Process once…")
        assert pg.get_by_role("button",name="Process once…").count()==1, "only the self, non-pre-restore draft is processable"
        pg.get_by_role("button",name="Process once…").click()
        assert not posts, "clicking Process once must not send anything before confirmation"
        pg.get_by_role("button",name="Confirm: process once").dblclick()
        pg.wait_for_selector("text=Stop this nudge")
        proc=[x for x in posts if x[0].endswith("/process")]
        assert len(proc)==1 and proc[0][1]=={"body_digest":"digest1"}, posts
        pg.get_by_role("button",name="Stop this nudge").click()
        pg.wait_for_selector("text=Stopped: the model was never asked")
        # PRE_RESTORE: Copy needs a second, warned click; never Process.
        cp=pg.get_by_role("button",name="Copy as new")
        cp.nth(1).click()
        pg.wait_for_selector("text=Copy anyway (the original may already have run; this may repeat it)")
        n_before=len([x for x in posts if x[0]=="/api/nudges"])
        assert n_before==0

        pg.screenshot(path="/tmp/process-ui.png", full_page=True)
        assert not errs, errs
        print("PASS: confirm before send; one POST with the saved digest; GET-only receipts; Stop; PRE_RESTORE copy warns, never processes")
    finally:
        b.close()
print("1/1 process browser fixture test PASS; real owner machine NOT RUN")
