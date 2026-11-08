#!/usr/bin/env python3
"""Isolated Chromium test of the manual nudge workbench (#96). Fixtures, NOT real owner/Nemotron.
Checks: Manual nudge appears only on a real Thought card, Save draft posts ONE body with a stable
idempotency key (a double click can't create two), history shows DRAFT with Process BLOCKED, and the
only POST the page makes is /api/nudges."""
import json, sys
from pathlib import Path
from playwright.sync_api import sync_playwright

sys.path.insert(0, str(Path(__file__).resolve().parent))
HTML=(Path(__file__).resolve().parents[2]/"cmd/tass-core/cognition.html").read_text()
ORIGIN="http://127.0.0.1:8795"
NOW="2026-11-08T15:00:00Z"
LIB={"identity_entity_id":"ent_0123","engine_entity_id":"eng_1","identity_name":"Mira",
 "engine_commit":"cab101c3d1d62953d99dd929e4d1b11ad55eb750","brain_revision":7,"note":"",
 "thoughts":[{"id":"thought:thought_1","kind":"thought","title":"Real persistent Thought","summary":"Canonical question",
   "source":"Thought Engine: persistent Thought","source_id":"thought_1","status":"active","verification":"OBSERVED",
   "recorded_at":NOW,"related_cycle_ids":["tp_1"],"fields":[]}],
 "cycles":[{"id":"cycle:tp_1","kind":"cycle","title":"Cycle tp_1","summary":"linked","source":"Thought Engine: cycle aggregate",
   "source_id":"tp_1","status":"evaluated","verification":"VERIFIED-LINK","recorded_at":NOW,"fields":[]}],
 "outcomes":[],"library":[]}
BLOCKED={"state":"BLOCKED","reason":"No Identity action for nudge processing exists yet."}
saved=[]; posts=[]
def route(req):
    r=req.request; url=r.url
    if url==ORIGIN+"/cognition":
        req.fulfill(status=200,content_type="text/html",body=HTML.replace("__PAGE_TOKEN__","fixture-token"));return
    if r.method=="POST":
        posts.append(url)
        if url.endswith("/api/nudges"):
            body=json.loads(r.post_data)
            replay=any(n["idempotency_key"]==body["idempotency_key"] for n in saved)
            if not replay:
                saved.append(dict(body,nudge_id="nudge_fixture%d"%len(saved),status="DRAFT",recipient_kind="card_position",
                    created_at=NOW,expires_at=NOW,source_revision=NOW,source_evidence_refs=["trace:trace_1","cycle:tp_1"],
                    draft_identity_decision_id="dec_fixture",model_calls=0,processing=BLOCKED,
                    processing_cycle_id=None,result_thought_id=None))
            n=[x for x in saved if x["idempotency_key"]==body["idempotency_key"]][0]
            req.fulfill(status=200 if replay else 201,content_type="application/json",body=json.dumps(dict(n,replayed=replay)));return
        req.fulfill(status=405,body="NO ACTION");return
    if "/api/cognition/library" in url:
        req.fulfill(status=200,content_type="application/json",body=json.dumps(LIB));return
    if "/api/cognition/pulse" in url:
        req.fulfill(status=503,body="n/a");return
    if "/api/nudges?" in url:
        req.fulfill(status=200,content_type="application/json",body=json.dumps({"owner_entity_id":"ent_0123",
            "nudges":list(reversed(saved)),"invalid_lines":0,"processing":BLOCKED,"note":"local only"}));return
    req.fulfill(status=404,body="Not found")

with sync_playwright() as p:
    browser=p.chromium.launch(headless=True,args=["--no-sandbox"])
    try:
        pg=browser.new_page(viewport={"width":1280,"height":900})
        errors=[]; pg.on("pageerror",lambda e:errors.append(str(e)))
        pg.route("**/*",route)
        pg.goto(ORIGIN+"/cognition")
        pg.wait_for_selector(".tile",timeout=10000)
        # Not on cycles.
        pg.get_by_role("button",name="Inspect Cycle tp_1").click()
        assert not pg.locator("#nudge").is_visible(), "nudge must not be offered on a cycle"
        pg.get_by_role("button",name="Close details").click()
        pg.get_by_role("button",name="Inspect Real persistent Thought").click()
        pg.get_by_role("button",name="Manual nudge (draft)").click()
        proc=pg.get_by_role("button",name="Process — BLOCKED: no Identity processing grant exists yet")
        assert proc.is_disabled()
        area=pg.locator("#thoughtDetail")
        area.locator("select").nth(0).select_option("position-3")
        area.locator("select").nth(1).select_option("new_evidence")
        area.locator("textarea").fill("Compare this with the latest cycle.")
        save=pg.get_by_role("button",name="Save draft (local only, no model call)")
        save.dblclick()
        pg.wait_for_function("document.querySelector('#thoughtDetail').textContent.includes('Nudge history for thought_1')")
        text=area.inner_text()
        assert "DRAFT · nudge_fixture0" in text and "BLOCKED" in text and "position-3" in text, text
        assert len(saved)==1, saved
        assert saved[0]["source_thought_id"]=="thought_1" and saved[0]["recipient_id"]=="position-3"
        assert saved[0]["idempotency_key"].startswith("nudge-")
        assert all(u.endswith("/api/nudges") for u in posts), posts
        pg.screenshot(path="/tmp/nudge-history.png",full_page=False)
        assert not errors, errors
        print("PASS: manual nudge draft on a real Thought, one record for a double click, Process BLOCKED, only POST is /api/nudges")
    finally:
        browser.close()
print("1/1 nudge browser fixture test PASS; real owner machine NOT RUN")
