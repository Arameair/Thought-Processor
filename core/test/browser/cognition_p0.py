#!/usr/bin/env python3
"""P0 (#106) browser fixture — NOT the owner's machine, NOT Nemotron. Go-shaped responses only.

A1  opening a Thought shows the question and the full newest Note first (no extra click, no JSON)
A2  a COMPLETED receipt's "Read what TASS wrote" focuses the development with the SAME request ID
A3  no stale BLOCKED text; readiness is shown; a WAITING draft offers no Process button
A4  Copy as new: one POST /api/nudges, no /process, the new draft is highlighted and named
A5  reading (open, history, deep link, refresh) sends zero POSTs
A7  Note text that looks like HTML is shown literally and runs nothing
A11 half-window (960x1000) and phone (390x844): question + newest Note visible without scrolling
    sideways; screenshots in /tmp/p0-*.png
+   refresh with #thought=…&dev=… reopens the same Note; history labels keep request/nudge IDs
"""
import json
from pathlib import Path
from playwright.sync_api import sync_playwright

HTML = (Path(__file__).resolve().parents[2] / "cmd/tass-core/cognition.html").read_text()
ORIGIN = "http://127.0.0.1:8795"
NOW = "2026-11-09T18:20:00Z"
NID = "nudge_%020x" % 1
NID2 = "nudge_%020x" % 2
XSS = 'A short idea. <img src=x onerror="window.__xss=1"></script><b>not bold</b> & more'
LIB = {"identity_entity_id": "ent_0123", "engine_entity_id": "eng_1", "identity_name": "Mira", "engine_commit": "c" * 40,
       "brain_revision": 7, "note": "",
       "thoughts": [{"id": "thought:thought_1", "kind": "thought", "title": "identity test (fixture)", "summary": "q",
                     "source": "Thought Engine: persistent Thought", "source_id": "thought_1", "status": "active",
                     "verification": "OBSERVED", "recorded_at": NOW, "related_cycle_ids": [], "fields": []}],
       "cycles": [], "outcomes": [], "library": []}
THOUGHT = {"thought_id": "thought_1", "question": "What is the fixture notebook skill's purpose?", "state": "active",
           "history_total": 43, "history": [
               {"at": NOW, "kind": "formed", "event_id": "evt_1", "trace_id": "trace_1"},
               {"at": NOW, "kind": "developed", "worker_request_id": "mr_0001", "source_nudge_id": NID, "by": "owner-nudge"}],
           "developments": [
               {"request_id": "mr_0001", "at": NOW, "source_nudge_id": NID, "persistence": "NOTE_AND_STEP_CORRELATED",
                "nudge_receipt": "COMPLETED_VERIFIED", "step": {"at": NOW, "kind": "developed"},
                "note": {"at": NOW, "text": XSS, "by": "Nemotron (owner-nudge)", "suggested_state": "resolve", "suggested_state_applied": False},
                "meaning": "Saved: one Note and one developed step share this request ID. That shows it was saved, not that it answers the Thought's question."},
               {"request_id": "mr_0000", "at": "2026-11-08T10:00:00Z", "persistence": "STEP_ONLY", "step": {"at": NOW, "kind": "developed"},
                "note": None, "meaning": "A developed step is recorded, but its Note is no longer stored."}],
           "developments_total": 2, "traces": [], "related_cycle_ids": []}
state = {"copied": None}
posts = []


def nudges():
    done = {"nudge_id": NID, "status": "DRAFT", "recipient_id": "self", "recipient_kind": "local_entity", "reason_code": "owner_request",
            "requested_action": "Pay attention to the notebook", "priority": "normal", "created_at": NOW, "expires_at": NOW,
            "source_thought_id": "thought_1", "body_digest": "d1", "processing": {"state": "SEE_RUN", "reason": "x"},
            "run": {"state": "COMPLETED", "verified": True, "model_http": "responded", "request_id": "mr_0001", "step_at": NOW,
                    "can_copy": True, "engine": {"suggested_state": "resolve"}}}
    waiting = dict(done, nudge_id=NID2, run=None, body_digest="d2",
                   processing={"state": "WAITING", "reason": "Not ready: the owner grant cognition.nudge-process is not enabled on Identity's page."})
    out = [done, waiting]
    if state["copied"]:
        out.insert(0, dict(waiting, nudge_id=state["copied"], body_digest="d3"))
    return {"nudges": out, "processing": {"state": "NOT_READY", "reason": "Not ready: the owner grant cognition.nudge-process is not enabled on Identity's page. Saving drafts still works."},
            "readiness": {"identity": "UNLOCKED", "process_grant": "NOT_ENABLED", "engine": "REACHABLE"}, "note": "local"}


def route(req):
    r = req.request
    url = r.url
    if url.split("#")[0] == ORIGIN + "/cognition":
        req.fulfill(status=200, content_type="text/html", body=HTML.replace("__PAGE_TOKEN__", "t"))
        return
    if r.method == "POST":
        posts.append(url.replace(ORIGIN, ""))
        if url.endswith("/api/nudges"):
            state["copied"] = "nudge_%020x" % 3
            req.fulfill(status=201, content_type="application/json", body=json.dumps({"nudge_id": state["copied"], "status": "DRAFT"}))
            return
        req.fulfill(status=405)
        return
    if "/api/cognition/library" in url:
        req.fulfill(status=200, content_type="application/json", body=json.dumps(LIB)); return
    if "/api/cognition/thoughts/thought_1" in url:
        req.fulfill(status=200, content_type="application/json", body=json.dumps(THOUGHT)); return
    if "/api/cognition/pulse" in url:
        req.fulfill(status=503, body="n/a"); return
    if "/api/nudges?" in url:
        req.fulfill(status=200, content_type="application/json", body=json.dumps(nudges())); return
    req.fulfill(status=404)


def visible_in_viewport(pg, locator):
    box = locator.bounding_box()
    vp = pg.viewport_size
    return box is not None and box["y"] >= 0 and box["y"] < vp["height"] and box["x"] >= 0 and box["x"] + min(box["width"], 50) <= vp["width"]


with sync_playwright() as p:
    b = p.chromium.launch(headless=True, args=["--no-sandbox"])
    results = []
    try:
        for (w, h, tag) in [(1280, 1000, "desktop"), (960, 1000, "half"), (390, 844, "phone")]:
            posts.clear(); state["copied"] = None
            pg = b.new_page(viewport={"width": w, "height": h}); errs = []
            pg.on("pageerror", lambda e: errs.append(str(e)))
            pg.route("**/*", route)
            pg.goto(ORIGIN + "/cognition")
            pg.wait_for_selector(".tile")
            pg.get_by_role("button", name="Inspect identity test (fixture)").click()
            # A1: no further click; the newest Note is the first development.
            pg.wait_for_selector("#dev-mr_0001")
            area = pg.locator("#thoughtDetail")
            note = pg.locator("#dev-mr_0001 .note")
            assert note.inner_text() == XSS, note.inner_text()
            text = area.inner_text()
            for want in ["What TASS has written", "What is the fixture notebook skill's purpose?", "request mr_0001", "nudge " + NID,
                         "suggested state: resolve — not applied", "not that it answers", "Note is no longer stored"]:
                assert want in text, (want, text)
            # A7: literal text, nothing executed, no element created from it.
            assert pg.evaluate("window.__xss") is None and area.locator("img").count() == 0 and area.locator("b", has_text="not bold").count() == 0
            # A11: question and newest Note are on screen, no sideways scrolling in the sheet.
            assert visible_in_viewport(pg, pg.locator("text=What is the fixture notebook skill's purpose?").first), tag
            assert visible_in_viewport(pg, note), tag
            assert pg.evaluate("(()=>{const s=document.querySelector('.sheet');return s.scrollWidth<=s.clientWidth+1})()"), tag
            pg.screenshot(path=f"/tmp/p0-{tag}-thought.png")
            # A6 (page side): Copy Evidence for a Thought carries references only, never the Note text.
            pg.evaluate("navigator.clipboard.writeText=async t=>{window.__copied=t}")
            pg.get_by_role("button", name="Copy Evidence").click()
            pg.wait_for_function("window.__copied!==undefined")
            copied = pg.evaluate("window.__copied")
            assert "thought_1" in copied and "short idea" not in copied and "mr_0001" not in copied, copied
            # Advanced (collapsed): history labels keep the IDs; "trace NOT RECORDED" is not "no development".
            assert not pg.locator("details").first.evaluate("d=>d.open")
            pg.locator("summary", has_text="Recorded details").click()
            adv = pg.locator("details").first.inner_text()
            assert "newest 2 of 43" in adv and "trace NOT RECORDED · request mr_0001 · nudge " + NID + " · by owner-nudge" in adv, adv
            # A3: nudge history: derived readiness, no stale BLOCKED, WAITING draft has no Process.
            pg.get_by_role("button", name="Nudge history").click()
            pg.wait_for_selector("text=Read what TASS wrote")
            hist = area.inner_text()
            assert "BLOCKED" not in hist and "processing grant: NOT_ENABLED" in hist and "Identity still decides each run" in hist, hist
            assert pg.get_by_role("button", name="Process once…").count() == 0
            # A2: deep link to the SAME development.
            pg.get_by_role("button", name="Read what TASS wrote").click()
            pg.wait_for_selector("#dev-mr_0001.focus")
            assert visible_in_viewport(pg, pg.locator("#dev-mr_0001")), tag
            assert "dev=mr_0001" in pg.url and "thought=thought_1" in pg.url, pg.url
            assert not posts, ("A5: reading sent POSTs", posts)
            # Refresh: the hash reopens the same Note.
            pg.reload(); pg.wait_for_selector("#dev-mr_0001.focus")
            assert not posts, posts
            # A4: Copy as new on the COMPLETED nudge: exactly one draft POST, no process, new draft highlighted.
            pg.get_by_role("button", name="Nudge history").click()
            pg.wait_for_selector("text=Read what TASS wrote")
            pg.get_by_role("button", name="Copy as new").first.click()
            pg.wait_for_selector("#nudge-" + "nudge_%020x" % 3 + ".focus")
            assert posts == ["/api/nudges"], posts
            assert "Nothing was processed" in pg.locator("#toast").inner_text()
            pg.screenshot(path=f"/tmp/p0-{tag}-copied.png")
            assert not errs, errs
            pg.close()
            results.append(tag)
            print(f"PASS [{tag} {w}x{h}]: Note first and literal; same-ID deep link; refresh reopens; derived readiness; Copy as new = 1 draft POST, highlighted; reads = 0 POST")
    finally:
        b.close()
print(f"{len(results)}/3 P0 browser fixture runs PASS (fixtures; owner machine and real Nemotron NOT RUN)")
