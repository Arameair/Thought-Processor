#!/usr/bin/env python3
"""WP0-E (#106 owner report: "i hit the button but nothing happened") browser fixture.
NOT the owner's machine, NOT Nemotron.

Root cause: a Thought already opens on its saved Notes; the bottom "What TASS has written" button
re-fetched and redrew the same content, so clicking it changed nothing the owner could see.

Checked at 1280x1000, 960x1000 and 390x844:
  E1 opening the Thought shows the "Saved Notes" heading and the newest Note in the viewport, with
     focus on the heading; no "What TASS has written" button exists; no inert saved-Notes button
  E2 every visible action in the Thought sheet has an observable result, even when the owner has
     scrolled down to the action row first (the new view's heading is scrolled into view and
     focused, or the status line changes, or the page navigates)
  E3 Nudge history -> "Back to saved Notes" -> the Notes are shown again and the button disappears;
     the completed receipt's "Read what TASS wrote" still focuses the SAME request ID; Back/Forward
  E4 refresh keeps the saved Notes view
  E5 a denied / locked detail read shows an honest error, no Saved Notes heading, nothing invented
  E6 no POST / process / stop / model request from any of this
"""
import json
from pathlib import Path
from playwright.sync_api import sync_playwright

HTML = (Path(__file__).resolve().parents[2] / "cmd/tass-core/cognition.html").read_text()
ORIGIN = "http://127.0.0.1:8795"
NOW = "2026-11-09T18:20:00Z"
NID = "nudge_%020x" % 1
NOTE = "A short idea about the fixture notebook."
LIB = {"identity_entity_id": "ent_0123", "engine_entity_id": "eng_1", "identity_name": "Mira", "engine_commit": "c" * 40,
       "brain_revision": 7, "note": "",
       "thoughts": [{"id": "thought:thought_1", "kind": "thought", "title": "identity test (fixture)", "summary": "q",
                     "source": "Thought Engine: persistent Thought", "source_id": "thought_1", "status": "active",
                     "verification": "OBSERVED", "recorded_at": NOW, "related_cycle_ids": [], "fields": []}],
       "cycles": [], "outcomes": [], "library": []}
# Enough history to make the sheet scroll, so "nothing visible happened" would be reproducible.
HIST = [{"at": NOW, "kind": "reinforced", "trace_id": "trace_%02d" % i} for i in range(40)]
THOUGHT = {"thought_id": "thought_1", "question": "What is the fixture notebook skill's purpose?", "state": "active",
           "history_total": 41, "history": HIST + [{"at": NOW, "kind": "developed", "worker_request_id": "mr_0001", "source_nudge_id": NID}],
           "developments": [{"request_id": "mr_0001", "at": NOW, "source_nudge_id": NID, "persistence": "NOTE_AND_STEP_CORRELATED",
                             "nudge_receipt": "COMPLETED_VERIFIED", "step": {"at": NOW, "kind": "developed"},
                             "note": {"at": NOW, "text": NOTE, "by": "Nemotron (owner-nudge)", "suggested_state": "keep", "suggested_state_applied": False},
                             "meaning": "Saved, not that it answers the Thought's question."}],
           "developments_total": 1, "traces": [], "related_cycle_ids": []}
NUDGES = {"nudges": [{"nudge_id": NID, "status": "DRAFT", "recipient_id": "self", "recipient_kind": "local_entity",
                      "reason_code": "owner_request", "requested_action": "Look at the notebook", "priority": "normal",
                      "created_at": NOW, "expires_at": NOW, "source_thought_id": "thought_1", "body_digest": "d1",
                      "processing": {"state": "SEE_RUN", "reason": "x"},
                      "run": {"state": "COMPLETED", "verified": True, "model_http": "responded", "request_id": "mr_0001",
                              "step_at": NOW, "can_copy": True}}],
          "processing": {"state": "NOT_READY", "reason": "Not ready: grant not enabled."},
          "readiness": {"identity": "UNLOCKED", "process_grant": "NOT_ENABLED", "engine": "REACHABLE"}, "note": "local"}
mode = {"denied": False}
writes = []


def route(req):
    r = req.request
    path = r.url.replace(ORIGIN, "").split("#")[0]
    if r.method != "GET":
        writes.append(r.method + " " + path)
        req.fulfill(status=405); return
    if path == "/cognition":
        req.fulfill(status=200, content_type="text/html", body=HTML.replace("__PAGE_TOKEN__", "t")); return
    if path.startswith("/dev"):
        req.fulfill(status=200, content_type="text/html", body="<!doctype html><title>Development</title><h1>Development</h1>"); return
    if path.startswith("/api/cognition/library"):
        req.fulfill(status=200, content_type="application/json", body=json.dumps(LIB)); return
    if path.startswith("/api/cognition/thoughts/thought_1"):
        if mode["denied"]:
            req.fulfill(status=403, content_type="application/json", body=json.dumps({"error": "Cognition read denied: Identity is LOCKED"})); return
        req.fulfill(status=200, content_type="application/json", body=json.dumps(THOUGHT)); return
    if path.startswith("/api/cognition/pulse"):
        req.fulfill(status=503, body="n/a"); return
    if path.startswith("/api/nudges"):
        req.fulfill(status=200, content_type="application/json", body=json.dumps(NUDGES)); return
    req.fulfill(status=404)


def in_view(pg, sel):
    return pg.evaluate("""s=>{const e=document.querySelector(s);if(!e)return false;const r=e.getBoundingClientRect();
      return r.bottom>0&&r.top<innerHeight&&r.height>0}""", sel)


def to_actions(pg):
    pg.evaluate("document.querySelector('.sheet .actions').scrollIntoView({block:'end'})")


def state(pg):
    return pg.evaluate("[document.getElementById('thoughtDetail').innerText,document.getElementById('toast').textContent,location.href]")


with sync_playwright() as p:
    b = p.chromium.launch(headless=True, args=["--no-sandbox"])
    done = []
    try:
        for (w, h, tag) in [(1280, 1000, "desktop"), (960, 1000, "half"), (390, 844, "phone")]:
            writes.clear(); mode["denied"] = False
            ctx = b.new_context(viewport={"width": w, "height": h})
            pg = ctx.new_page(); errs = []
            pg.on("pageerror", lambda e: errs.append(str(e)))
            pg.route("**/*", route)
            pg.goto(ORIGIN + "/cognition"); pg.wait_for_selector(".tile")
            # E1
            pg.get_by_role("button", name="Inspect identity test (fixture)").click()
            pg.wait_for_selector("#savedNotes")
            assert in_view(pg, "#savedNotes") and in_view(pg, "#dev-mr_0001 .note"), f"E1 [{tag}] heading and Note on screen"
            assert pg.evaluate("document.activeElement&&document.activeElement.id") == "savedNotes", "E1 focus on Saved Notes"
            assert "1 saved development, newest first" in pg.locator("#thoughtDetail").inner_text()
            assert pg.get_by_role("button", name="What TASS has written").count() == 0, "E1 the redundant button is gone"
            assert not pg.locator("#expandEngine").is_visible(), "E1 no saved-Notes button while the Notes are shown"
            pg.screenshot(path=f"/tmp/wp0e-{tag}-open.png")
            # E2: owner scrolls to the action row, then clicks each action
            for name, expect_sel in [("Nudge history", "text=Nudge history for thought_1"), ("Manual nudge (draft)", "text=Manual nudge — draft")]:
                to_actions(pg); before = state(pg)
                pg.get_by_role("button", name=name).click(); pg.wait_for_selector(expect_sel)
                head = pg.locator("#thoughtDetail h3").first
                assert state(pg) != before, f"E2 {name}: something changed"
                assert head.evaluate("e=>{const r=e.getBoundingClientRect();return r.bottom>0&&r.top<innerHeight}"), f"E2 [{tag}] {name}: new view's heading on screen"
                assert head.evaluate("e=>document.activeElement===e"), f"E2 {name}: focus moved to the new view"
                # E3: from a nudge view, "Back to saved Notes" really goes back to them
                assert pg.locator("#expandEngine").is_visible() and pg.locator("#expandEngine").inner_text() == "Back to saved Notes"
                to_actions(pg)
                pg.get_by_role("button", name="Back to saved Notes").click(); pg.wait_for_selector("#savedNotes")
                assert in_view(pg, "#savedNotes") and pg.evaluate("document.activeElement.id") == "savedNotes", f"E3 [{tag}] Notes back on screen"
                assert not pg.locator("#expandEngine").is_visible(), "E3 button hidden again"
            to_actions(pg); before = state(pg)
            pg.evaluate("navigator.clipboard.writeText=async t=>{window.__copied=t}")
            pg.get_by_role("button", name="Copy Evidence").click(); pg.wait_for_function("window.__copied!==undefined")
            assert state(pg)[1] != before[1] and in_view(pg, "#toast"), "E2 Copy Evidence: visible confirmation"
            assert NOTE not in pg.evaluate("window.__copied"), "Copy Evidence still carries no Note text"
            # E3: receipt -> same Note, Back/Forward
            pg.get_by_role("button", name="Nudge history").click(); pg.wait_for_selector("text=Read what TASS wrote")
            pg.get_by_role("button", name="Read what TASS wrote").click(); pg.wait_for_selector("#dev-mr_0001.focus")
            assert "dev=mr_0001" in pg.url and in_view(pg, "#dev-mr_0001")
            pg.go_back(); pg.wait_for_selector("text=Nudge history for thought_1")
            pg.go_forward(); pg.wait_for_selector("#dev-mr_0001.focus")
            # E4
            pg.reload(); pg.wait_for_selector("#dev-mr_0001.focus")
            assert in_view(pg, "#dev-mr_0001 .note")
            # E2: Development link navigates (a visible result)
            to_actions(pg)
            pg.get_by_role("button", name="Open Development (general workspace, not this run)").click()
            pg.wait_for_url("**/dev**"); pg.go_back(); pg.wait_for_selector(".tile")
            # E5: denied detail read
            mode["denied"] = True
            pg.goto(ORIGIN + "/cognition"); pg.reload(); pg.wait_for_selector(".tile")
            pg.get_by_role("button", name="Inspect identity test (fixture)").click()
            pg.wait_for_function("document.getElementById('toast').textContent.includes('Thought detail HTTP 403; Identity denied')")
            assert pg.locator("#savedNotes").count() == 0 and NOTE not in pg.locator(".sheet").inner_text(), "E5 nothing invented"
            # E6
            assert not writes, ("E6 no writes", writes)
            assert not errs, errs
            ctx.close()
            done.append(tag)
            print(f"PASS [{tag} {w}x{h}]: Notes on screen + focused; no inert button; every action visible; receipt->same Note; Back/Forward/refresh; denied honest; zero writes")
    finally:
        b.close()
print(f"{len(done)}/3 WP0-E saved-Notes affordance browser fixture runs PASS (fixtures; owner machine and real Nemotron NOT RUN)")
