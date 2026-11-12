#!/usr/bin/env python3
"""Stage 3A (#120; design #107 §3.2): Cognition → Development, "How this Note was produced". NOT the owner's
machine, NOT Nemotron. The run payload is cognition_stage3a_run.json, Core's real DevelopmentRun output
(checked by TestDevelopmentRunBrowserFixtureIsCoreOutput); a hostile prompt/reply is substituted for display.

H1  in the expanded Thought, a development with THIS owner's nudge links to /dev?run=<nudge>; one with no
    nudge or another owner's nudge has no link
H2  following it opens Development on "How this Note was produced": the Thought, nudge, request, source
    revision, Engine commit and result; the 12 stages in Core's order, each with Core's status and summary
H3  every status has a one-sentence meaning (legend); the "UNEXECUTED DEVELOPMENT PROPOSAL" banner is not shown
H4  prompt, reply, Note and attention are shown literally: hostile markup is text and runs nothing
H5  one obvious next action: "Back to this Note in Cognition", which opens the same Note
H6  direct URL, refresh, Back to Cognition and Forward to Development work; history has no extra entries
H7  zero POSTs in all of it; Development reads only /api/development/runs/<id> (plus its usual GETs);
    Cognition never reads a run
H8  refused (403), not found (404), unavailable (503), an answer for another nudge, or a bad ID: an error
    line, no stage, nothing left from an earlier view
H9  1280 / 960 / 390: no sideways scrolling in the run view; the title and next action are on screen
    python3 cognition_stage3a.py
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
RUN = json.loads((Path(__file__).resolve().parent / "cognition_stage3a_run.json").read_text())
ORIGIN = "http://127.0.0.1:8795"
NOW = "2026-11-12T10:00:00Z"
NID, OTHER = RUN["nudge_id"], "nudge_%020x" % 0xbeef
REQ, TID = RUN["request_id"], RUN["thought_id"]
HOSTILE = '<img src=x onerror="window.__xss=1"><b>bold?</b>'
PROMPT = "You are developing an ongoing thought for Mira.\nThought: \"q\"\n" + HOSTILE + "\n[owner attention sha256:abc]"
REPLY = "NOTE: a saved idea " + HOSTILE + "\nSTATE: keep"
for s in RUN["stages"]:
    if s["stage"] == "context_sent":
        s["text"] = PROMPT
    if s["stage"] == "model_reply":
        s["text"] = REPLY

LIB = {"identity_entity_id": "ent_0123", "engine_entity_id": "eng_1", "identity_name": "Mira", "engine_commit": "c" * 40,
       "brain_revision": 7, "note": "", "cycles": [], "outcomes": [], "library": [],
       "thoughts": [{"id": "thought:" + TID, "kind": "thought", "title": "stage three (fixture)", "summary": "q", "source": "Thought Engine: persistent Thought",
                     "source_id": TID, "status": "active", "verification": "OBSERVED", "recorded_at": NOW, "related_cycle_ids": [], "fields": [],
                     "lifecycle": "active", "notes_count": 3, "latest_output": {"kind": "DEVELOPED_NOTE", "at": NOW, "by": "engine"}}]}


def dev(req, nudge=None, receipt=None, minute=0):
    d = {"request_id": req, "at": "2026-11-12T09:%02d:00Z" % (59 - minute), "persistence": "NOTE_AND_STEP_CORRELATED",
         "step": {"at": NOW, "kind": "developed"}, "note": {"at": NOW, "text": "Note " + req, "by": "Nemotron", "suggested_state": "", "suggested_state_applied": False},
         "meaning": "Saved: one Note and one developed step share this request ID. That shows it was saved, not that it answers the Thought's question."}
    if nudge:
        d["source_nudge_id"], d["nudge_receipt"] = nudge, receipt
    return d


THOUGHT = {"thought_id": TID, "question": "q", "state": "active", "history_total": 1, "history": [{"at": NOW, "kind": "formed"}],
           "developments": [dev(REQ, NID, "COMPLETED_VERIFIED", 0), dev("mr_plain", None, None, 1), dev("mr_foreign", OTHER, "NOT_IN_THIS_OWNERS_JOURNAL", 2)],
           "developments_total": 3, "traces": [], "related_cycle_ids": []}
BRAIN = {"blocks": [], "relations": {}, "graph": {"revision": 0, "nodes": [], "edges": [], "notes": []}, "report": {"problems": []}, "history": []}

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
    if path == "/api/cognition/pulse":
        req.fulfill(status=503, body="n/a"); return
    req.fulfill(status=404, body="")


RUNPROBE = """()=>{const v=document.getElementById('runView');if(!v||v.hidden)return null;
  const a=document.querySelector('#runNext a');
  return {title:(v.querySelector('h2')||{}).textContent||'',ids:(v.querySelector('.ids')||{}).textContent||'',
    stages:[...v.querySelectorAll('li.st')].map(li=>({id:li.id,status:li.dataset.status,chip:li.querySelector('.status').textContent,
      sum:li.querySelector('.sum').textContent,text:(li.querySelector('pre')||{}).textContent||'',html:(li.querySelector('pre')||{}).innerHTML||''})),
    legend:(v.querySelector('details.legend')||{}).textContent||'',next:a?a.textContent:'',href:a?a.getAttribute('href'):'',
    err:(v.querySelector('.err')||{}).textContent||'',over:v.scrollWidth-v.clientWidth,
    proposal:!document.getElementById('cognitionProposal').hidden,text:v.textContent}}"""

failures, passed = [], 0
with sync_playwright() as p:
    b = p.chromium.launch(headless=True, args=["--no-sandbox"])
    try:
        for (w, h, tag) in [(1280, 1000, "desktop"), (960, 1000, "half"), (390, 844, "phone")]:
            def check(name, cond, detail=""):
                global passed
                if cond:
                    passed += 1
                else:
                    failures.append(f"[{tag}] {name}: {str(detail)[:700]}")

            posts.clear(); gets.clear(); state["run"] = (200, RUN)
            ctx = b.new_context(viewport={"width": w, "height": h})
            pg = ctx.new_page(); errs = []
            pg.on("pageerror", lambda e: errs.append(str(e)))
            pg.route("**/*", route)

            # H1
            pg.goto(ORIGIN + "/cognition")
            pg.wait_for_selector(".tile")
            pg.get_by_role("button", name="Inspect stage three (fixture)").click()
            pg.wait_for_selector("#dev-" + REQ)
            links = pg.evaluate("[...document.querySelectorAll('#thoughtDetail .dev')].map(d=>[d.id,(d.querySelector('a.how')||{}).getAttribute?d.querySelector('a.how').getAttribute('href'):null])")
            links = dict(links)
            check("H1 this owner's processed nudge links to its run", links.get("dev-" + REQ) == "/dev?run=" + NID, links)
            check("H1 no link without a nudge, or for another owner's nudge", links.get("dev-mr_plain") is None and links.get("dev-mr_foreign") is None, links)
            check("H7 Cognition never reads a run", not [g for g in gets if g.startswith("/api/development/")], gets)
            h0 = pg.evaluate("history.length")

            # H2–H5
            pg.locator("#dev-" + REQ + " a.how").click()
            pg.wait_for_selector("#runView li.st")
            v = pg.evaluate(RUNPROBE)
            check("H2 lands on /dev?run=<nudge>", pg.url == ORIGIN + "/dev?run=" + NID, pg.url)
            check("H2 title", v and v["title"] == "How this Note was produced", v)
            for want in [TID, NID, REQ, RUN["source_revision"], (RUN["engine_commit"] or "")[:12], "COMPLETED", "verified by Core"]:
                check(f"H2 header shows {want!r}", v and want in v["ids"], v and v["ids"])
            got = [(s["id"], s["status"], s["chip"], s["sum"]) for s in (v or {}).get("stages", [])]
            want = [("stage-" + s["stage"], s["status"], s["status"], s["summary"]) for s in RUN["stages"]]
            check("H2 the 12 stages, in Core's order, with Core's status and summary", got == want, got)
            for k in ["RECORDED", "PARTIAL", "CODE-DEFINED", "NOT RECORDED", "NOT PART OF THIS PATH"]:
                check(f"H3 legend explains {k}", v and k in v["legend"], v and v["legend"])
            check("H3 the old proposal banner is not shown", v and not v["proposal"], v)
            stages = {s["id"]: s for s in v["stages"]}
            check("H4 prompt shown literally", stages["stage-context_sent"]["text"] == PROMPT and "<img" not in stages["stage-context_sent"]["html"], stages["stage-context_sent"])
            check("H4 reply shown literally", stages["stage-model_reply"]["text"] == REPLY, stages["stage-model_reply"])
            check("H4 hostile text runs nothing and makes no element", pg.evaluate("window.__xss") is None and pg.locator("#runView img").count() == 0
                  and pg.locator("#runView b", has_text="bold?").count() == 0, "")
            check("H4 reasoning is not invented", stages["stage-model_reasoning"]["text"] == "" and "thinking off" in stages["stage-model_reasoning"]["sum"], stages["stage-model_reasoning"])
            check("H5 one next action back to the same Note", v["next"] == "Back to this Note in Cognition" and v["href"] == "/cognition#thought=" + TID + "&dev=" + REQ, v)
            check("H5 says nothing here can run anything", "nothing on this page can run anything" in v["text"], "")
            check("H9 no sideways scrolling in the run view", v["over"] <= 1, v["over"])
            tbox, nbox = pg.locator("#runTitle").bounding_box(), pg.locator("#runNext").bounding_box()
            check("H9 title and next action on screen", tbox and nbox and tbox["y"] >= 0 and nbox["y"] + 20 <= h and nbox["x"] + 40 <= w, (tbox, nbox))
            pg.screenshot(path=f"/tmp/stage3a-{tag}.png")

            # H6: refresh, Back, Forward, direct URL
            n_run = gets.count("/api/development/runs/" + NID)
            pg.reload(); pg.wait_for_selector("#runView li.st")
            check("H6 refresh reads the run again and shows it", gets.count("/api/development/runs/" + NID) == n_run + 1 and len(pg.evaluate(RUNPROBE)["stages"]) == 12, "")
            pg.go_back(); pg.wait_for_selector("#dev-" + REQ)
            check("H6 Back returns to the Note in Cognition", "/cognition" in pg.url and pg.locator("#dev-" + REQ).count() == 1, pg.url)
            pg.go_forward(); pg.wait_for_selector("#runView li.st")
            check("H6 Forward returns to the run", pg.url == ORIGIN + "/dev?run=" + NID and len(pg.evaluate(RUNPROBE)["stages"]) == 12, pg.url)
            check("H6 history: one entry for Development", pg.evaluate("history.length") == h0 + 1, (h0, pg.evaluate("history.length")))
            # H5: the next action
            pg.get_by_role("link", name="Back to this Note in Cognition").click()
            pg.wait_for_selector("#dev-" + REQ + ".focus")
            check("H5 the next action opens the same Note, focused", pg.locator("#dev-" + REQ + ".focus").count() == 1 and "dev=" + REQ in pg.url, pg.url)

            # H8: refusals and bad answers, each from a direct URL, after a good view in the same tab
            other = copy.deepcopy(RUN); other["nudge_id"] = OTHER
            for name, resp, want in [("403", (403, "Identity denied Cognition read"), "Identity did not allow"),
                                     ("404", (404, "No processed nudge"), "There is no processed nudge"),
                                     ("503", (503, "unavailable"), "not available right now"),
                                     ("answer for another nudge", (200, other), "could not be read")]:
                state["run"] = (200, RUN)
                pg.goto(ORIGIN + "/dev?run=" + NID); pg.wait_for_selector("#runView li.st")
                state["run"] = resp
                pg.reload(); pg.wait_for_selector("#runView .err")
                v = pg.evaluate(RUNPROBE)
                check(f"H8 {name}: error line, no stage, no earlier text", v and want in v["err"] and not v["stages"] and REPLY not in v["text"]
                      and "NOTE:" not in pg.evaluate("document.body.innerText"), v)
            state["run"] = (200, RUN)
            n = len([g for g in gets if g.startswith("/api/development/")])
            pg.goto(ORIGIN + "/dev?run=nudge_../../etc"); pg.wait_for_selector("#runView .err")
            v = pg.evaluate(RUNPROBE)
            check("H8 a bad ID is refused in the page, with no request", "not a nudge ID" in v["err"] and len([g for g in gets if g.startswith("/api/development/")]) == n and not v["stages"], v)
            pg.goto(ORIGIN + "/dev"); pg.wait_for_timeout(400)
            check("H8 plain /dev has no run view", pg.evaluate(RUNPROBE) is None, "")

            check("H7 zero POSTs (Cognition, link, Development, refresh, Back/Forward, errors)", not posts, posts)
            allowed = ("/api/development/runs/", "/api/brain", "/api/status", "/api/cognition/library", "/api/cognition/thoughts/", "/api/cognition/pulse", "/api/nudges")
            check("H7 only expected GETs", all(g.startswith(allowed) for g in gets), sorted(set(gets)))
            check("no page errors", not errs, errs)
            ctx.close()
    finally:
        b.close()
for f in failures:
    print("FAIL:", f)
print(f"{passed}/{passed + len(failures)} Stage 3A provenance checks PASS across 1280/960/390")
sys.exit(1 if failures else 0)
