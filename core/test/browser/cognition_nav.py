#!/usr/bin/env python3
"""P0 navigation repair (#108) browser fixture — NOT the owner's machine, NOT Nemotron.

Root cause fixed: Cognition only ever called history.replaceState and had no popstate handler, so
opening a Thought created no history entry and Back left the application.

Checked at 1280x1000, 960x1000 and 390x844:
  N1 shelf -> Thought -> Back (shelf, still /cognition) -> Forward (same Thought + Note)
  N2 refresh with a Thought selected reopens it
  N3 completed nudge receipt -> "Read what TASS wrote" -> Back (nudge history) -> Back (Thought) -> Back (shelf)
     -> Back leaves Cognition only now -> Forward x4 returns to the same focused Note
  N4 Close -> shelf; Back reopens the Thought
  N5 direct links: #thought=..., #thought=...&dev=..., #thought=...&view=nudges, #card=cycle:...
  N6 invalid / unavailable references: unknown Thought, unknown card, hostile values, Thought whose
     detail read is refused (503): shelf or honest error, nothing invented, no script runs
  N7 non-Thought card (cycle) -> Back -> shelf
  N8 navigation sends no POST, no /process, no /stop, no model call
"""
import json
from pathlib import Path
from playwright.sync_api import sync_playwright

HTML = (Path(__file__).resolve().parents[2] / "cmd/tass-core/cognition.html").read_text()
ORIGIN = "http://127.0.0.1:8795"
NOW = "2026-11-09T18:20:00Z"
NID = "nudge_%020x" % 1
NOTE = "A short idea about the fixture notebook."


def card(kind, sid, title, **kw):
    c = {"id": kind + ":" + sid, "kind": kind, "title": title, "summary": "s", "source": "fixture", "source_id": sid,
         "status": "active", "verification": "OBSERVED", "recorded_at": NOW, "related_cycle_ids": [], "fields": []}
    c.update(kw)
    return c


LIB = {"identity_entity_id": "ent_0123", "engine_entity_id": "eng_1", "identity_name": "Mira", "engine_commit": "c" * 40,
       "brain_revision": 7, "note": "",
       "thoughts": [card("thought", "thought_1", "identity test (fixture)"), card("thought", "thought_2", "unreadable thought (fixture)")],
       "cycles": [card("cycle", "tp_1", "Cycle tp_1", verification="VERIFIED-LINK")], "outcomes": [], "library": []}
THOUGHT = {"thought_id": "thought_1", "question": "What is the fixture notebook skill's purpose?", "state": "active",
           "history_total": 2, "history": [{"at": NOW, "kind": "developed", "worker_request_id": "mr_0001", "source_nudge_id": NID}],
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
writes = []


def route(req):
    r = req.request
    url = r.url
    path = url.replace(ORIGIN, "").split("#")[0]
    if r.method != "GET":
        writes.append(r.method + " " + path)
        req.fulfill(status=405); return
    if path == "/":
        req.fulfill(status=200, content_type="text/html", body="<!doctype html><title>Core home</title><a href='/cognition'>Cognition</a>"); return
    if path == "/cognition":
        req.fulfill(status=200, content_type="text/html", body=HTML.replace("__PAGE_TOKEN__", "t")); return
    if path.startswith("/api/cognition/library"):
        req.fulfill(status=200, content_type="application/json", body=json.dumps(LIB)); return
    if path.startswith("/api/cognition/thoughts/thought_1"):
        req.fulfill(status=200, content_type="application/json", body=json.dumps(THOUGHT)); return
    if path.startswith("/api/cognition/thoughts/"):
        req.fulfill(status=503, body="Cognition evidence NOT RECORDED or temporarily unavailable"); return
    if path.startswith("/api/cognition/cycles/"):
        req.fulfill(status=200, content_type="application/json", body=json.dumps({"cycle_id": "tp_1", "steps": [], "missing": []})); return
    if path.startswith("/api/cognition/pulse"):
        req.fulfill(status=503, body="n/a"); return
    if path.startswith("/api/nudges"):
        req.fulfill(status=200, content_type="application/json", body=json.dumps(NUDGES)); return
    req.fulfill(status=404)


def sheet_open(pg):
    return pg.evaluate("document.getElementById('overlay').classList.contains('open')")


def wait_url(pg, needle, present=True):
    pg.wait_for_function("([n,p])=>p?location.href.includes(n):!location.href.includes(n)", arg=[needle, present])


with sync_playwright() as p:
    b = p.chromium.launch(headless=True, args=["--no-sandbox"])
    done = []
    try:
        for (w, h, tag) in [(1280, 1000, "desktop"), (960, 1000, "half"), (390, 844, "phone")]:
            writes.clear()
            pg = b.new_page(viewport={"width": w, "height": h}); errs = []
            pg.on("pageerror", lambda e: errs.append(str(e)))
            pg.route("**/*", route)
            pg.goto(ORIGIN + "/")                      # a page before Cognition, to detect "Back leaves the app"
            pg.click("text=Cognition"); pg.wait_for_selector(".tile")
            # N1
            pg.get_by_role("button", name="Inspect identity test (fixture)").click()
            pg.wait_for_selector("#dev-mr_0001"); wait_url(pg, "#thought=thought_1")
            pg.go_back(); pg.wait_for_function("!document.getElementById('overlay').classList.contains('open')")
            assert pg.url == ORIGIN + "/cognition", ("N1 Back must stay on the shelf", pg.url)
            pg.go_forward(); pg.wait_for_selector("#dev-mr_0001")
            assert sheet_open(pg) and NOTE in pg.locator("#thoughtDetail").inner_text(), "N1 Forward reopens the same Thought"
            # N2
            pg.reload(); pg.wait_for_selector("#dev-mr_0001")
            assert sheet_open(pg) and "#thought=thought_1" in pg.url, "N2 refresh"
            # N3
            pg.get_by_role("button", name="Nudge history").click(); wait_url(pg, "view=nudges")
            pg.get_by_role("button", name="Read what TASS wrote").click()
            pg.wait_for_selector("#dev-mr_0001.focus"); wait_url(pg, "dev=mr_0001")
            pg.go_back(); wait_url(pg, "view=nudges")
            pg.wait_for_function("document.getElementById('thoughtDetail').textContent.includes('Nudge history for thought_1')")
            assert pg.get_by_role("button", name="Read what TASS wrote").count() == 1, "N3 Back returns to the nudge history"
            pg.go_back(); wait_url(pg, "view=nudges", False); pg.wait_for_selector("#dev-mr_0001")
            assert pg.url.endswith("#thought=thought_1"), pg.url
            pg.go_back(); pg.wait_for_function("!document.getElementById('overlay').classList.contains('open')")
            assert pg.url == ORIGIN + "/cognition", ("N3 third Back = shelf", pg.url)
            pg.go_back(); pg.wait_for_url(ORIGIN + "/")    # only now does Back leave Cognition, as a browser should
            pg.go_forward(); pg.wait_for_selector(".tile")
            for _ in range(4):
                pg.go_forward()
            pg.wait_for_selector("#dev-mr_0001.focus"); assert "dev=mr_0001" in pg.url, ("N3 Forward to the focused Note", pg.url)
            # N4
            pg.get_by_role("button", name="Close details").click()
            pg.wait_for_function("!document.getElementById('overlay').classList.contains('open')")
            assert pg.url == ORIGIN + "/cognition", pg.url
            pg.go_back(); pg.wait_for_selector("#dev-mr_0001.focus")
            assert sheet_open(pg), "N4 Back after Close reopens what was closed"
            # N5 direct links
            for frag, expect in [("#thought=thought_1", "#dev-mr_0001"), ("#thought=thought_1&dev=mr_0001", "#dev-mr_0001.focus"),
                                 ("#thought=thought_1&view=nudges", "text=Nudge history for thought_1")]:
                pg.goto(ORIGIN + "/"); pg.goto(ORIGIN + "/cognition" + frag); pg.wait_for_selector(expect)
                assert sheet_open(pg), frag
            pg.goto(ORIGIN + "/"); pg.goto(ORIGIN + "/cognition#card=cycle%3Atp_1")
            pg.wait_for_function("document.getElementById('detailTitle').textContent==='Cycle tp_1'")
            assert sheet_open(pg), "N5 card link"
            # N6 invalid / unavailable
            for frag in ["#thought=nope", "#card=cycle%3Azzz", "#thought=..%2F..%2Fsecrets&dev=%3Cimg%20src%3Dx%20onerror%3Dwindow.__x%3D1%3E",
                         "#thought=%3Cscript%3Ewindow.__x%3D1%3C%2Fscript%3E"]:
                pg.goto(ORIGIN + "/"); pg.goto(ORIGIN + "/cognition" + frag); pg.wait_for_selector(".tile")  # a fresh load of the link
                pg.wait_for_function("document.getElementById('toast').textContent.includes('not in this identity')")
                assert not sheet_open(pg) and pg.url == ORIGIN + "/cognition", (frag, pg.url)
                assert pg.evaluate("window.__x") is None
            pg.goto(ORIGIN + "/"); pg.goto(ORIGIN + "/cognition#thought=thought_2")
            pg.wait_for_function("document.getElementById('toast').textContent.includes('Thought detail HTTP 503')")
            assert "Saved Notes" not in pg.locator("#thoughtDetail").inner_text(), "N6 unavailable detail: nothing invented"
            # N7 cycle card
            pg.goto(ORIGIN + "/"); pg.goto(ORIGIN + "/cognition"); pg.wait_for_selector(".tile")
            pg.get_by_role("button", name="Inspect Cycle tp_1").click(); wait_url(pg, "card=cycle")
            pg.go_back(); pg.wait_for_function("!document.getElementById('overlay').classList.contains('open')")
            assert pg.url == ORIGIN + "/cognition", ("N7", pg.url)
            # N8
            assert not writes, ("N8 navigation must not write", writes)
            assert not errs, errs
            pg.screenshot(path=f"/tmp/nav-{tag}.png")
            pg.close()
            done.append(tag)
            print(f"PASS [{tag} {w}x{h}]: Back/Forward/refresh/Close/receipt->Note/direct+invalid links; zero writes")
    finally:
        b.close()
print(f"{len(done)}/3 P0 navigation browser fixture runs PASS (fixtures; owner machine and real Nemotron NOT RUN)")
