#!/usr/bin/env python3
"""WP0-E async ordering (Codex audit on #110): deterministic delayed-response browser regressions.
NOT the owner's machine, NOT Nemotron.

The fixture HOLDS chosen GET responses (no timers): it keeps the request open, lets the test change the
owner's navigation intent, then releases the held response(s) in a chosen order. After every release,
the visible view, URL, focus and toast must still describe the NEWEST intent.

  R1 delayed nudge history, then Back to Saved Notes -> Notes stay (Codex finding 1)
  R2 delayed Thought detail, then Nudge history -> history stays (Codex finding 2)
  R3 rapid history -> Back -> Forward with all three reads held, released newest-first and oldest-last
  R4 cross-Thought: delayed Thought A, Close, open Thought B -> B stays; A's private Note never appears
  R5 delayed Thought detail, then Close -> the sheet stays closed, no toast
  R6 delayed ERROR (403) for an abandoned view -> no error toast over the current view
  R7 nudge-history poll (running run) is cancelled by navigation; a held poll response is ignored
  R8 delayed Thought detail on a direct link, then Back to the shelf -> shelf stays
  R9 zero non-GET requests; no page errors

Repair 2 (Codex re-audit on #110): a pending library Refresh versus a newer navigation intent.
The library GET is held; "before" is observed while it is held, "after" once it is released.
  F1  Codex's exact case: Notes -> nudge history -> Close -> held Refresh -> Back -> release.
      The history is open, focused and in the URL before AND after; no history entry is added or lost
  F2  held Refresh -> Back -> Forward -> release: the shelf (newest intent) stays
  F3  consecutive Refresh clicks, answers released oldest-first and newest-first: only the newest
      Refresh may replace the library
  F4  held Refresh is DENIED (403) after Back: fail closed (no cards, sheet closed, no Note or history
      text left in the page, truthful banner), the Back URL is kept; a later authorized Refresh opens it
  F5  Refresh denied (403) / F6 failed (503) while Notes are open: fail closed; retry recovers
  F7  the card is really gone from a NEWLY AUTHORIZED library: only then is the route reported
      unavailable and the URL replaced (no extra history entry)
  F8  a superseded Refresh that is denied while the newest is pending is not masked: nothing is shown
      until the newest answers
  F9  a superseded 200 arriving after the newest Refresh was denied can't bring the cards back
  F10 held Refresh, then open another Thought: it stays; the other Thought's private Note never appears
  F11 link followed while the FIRST library read is pending: kept (not rejected), opened on release
  F12 held Refresh, then Close: the sheet stays closed
Run at 1280x1000, 960x1000 and 390x844.
"""
import json
import sys
from pathlib import Path
from playwright.sync_api import sync_playwright

HTML_PATH = Path(sys.argv[1]) if len(sys.argv) > 1 else Path(__file__).resolve().parents[2] / "cmd/tass-core/cognition.html"
HTML = HTML_PATH.read_text()
ORIGIN = "http://127.0.0.1:8795"
NOW = "2026-11-09T18:20:00Z"
NOTE_A = "PRIVATE-A: note for the first fixture thought."
NOTE_B = "Note for the second fixture thought."


def card(sid, title):
    return {"id": "thought:" + sid, "kind": "thought", "title": title, "summary": "q", "source": "Thought Engine: persistent Thought",
            "source_id": sid, "status": "active", "verification": "OBSERVED", "recorded_at": NOW, "related_cycle_ids": [], "fields": []}


LIB = {"identity_entity_id": "ent_0123", "engine_entity_id": "eng_1", "identity_name": "Mira", "engine_commit": "c" * 40,
       "brain_revision": 7, "note": "", "thoughts": [card("thought_a", "alpha thought"), card("thought_b", "beta thought")],
       "cycles": [], "outcomes": [], "library": []}


def thought(sid, text):
    return {"thought_id": sid, "question": "Question for " + sid, "state": "active", "history_total": 1,
            "history": [{"at": NOW, "kind": "developed", "worker_request_id": "mr_" + sid}],
            "developments": [{"request_id": "mr_" + sid, "at": NOW, "persistence": "NOTE_AND_STEP_CORRELATED",
                              "note": {"at": NOW, "text": text, "by": "Nemotron (engine)", "suggested_state": "keep", "suggested_state_applied": False},
                              "step": {"at": NOW, "kind": "developed"}, "meaning": "Saved, not that it answers."}],
            "developments_total": 1, "traces": [], "related_cycle_ids": []}


def nudges(running=False):
    run = {"state": "RUNNING", "model_http": "none"} if running else None
    return {"nudges": [{"nudge_id": "nudge_%020x" % 1, "status": "DRAFT", "recipient_id": "self", "recipient_kind": "local_entity",
                        "reason_code": "owner_request", "requested_action": "Look", "priority": "normal", "created_at": NOW,
                        "expires_at": NOW, "source_thought_id": "thought_a", "body_digest": "d",
                        "processing": {"state": "SEE_RUN" if running else "WAITING", "reason": "x"}, "run": run}],
            "processing": {"state": "NOT_READY", "reason": "Not ready: grant not enabled."},
            "readiness": {"identity": "UNLOCKED", "process_grant": "NOT_ENABLED", "engine": "REACHABLE"}, "note": "local"}


class Server:
    def __init__(self):
        self.reset()

    def reset(self):
        self.hold = set()          # path prefixes whose next responses are held
        self.held = []             # [(route, kind, status, body)]
        self.writes = []
        self.gets = []
        self.deny = set()          # path prefixes answered 403
        self.running = False
        self.fail = set()          # path prefixes answered 503
        self.lib = LIB             # what the library GET returns

    def answer(self, path):
        for d in self.deny:
            if path.startswith(d):
                return 403, json.dumps({"error": "Cognition read denied"})
        for d in self.fail:
            if path.startswith(d):
                return 503, json.dumps({"error": "unavailable"})
        if path.startswith("/api/cognition/library"):
            return 200, json.dumps(self.lib)
        if path.startswith("/api/cognition/thoughts/thought_a"):
            return 200, json.dumps(thought("thought_a", NOTE_A))
        if path.startswith("/api/cognition/thoughts/thought_b"):
            return 200, json.dumps(thought("thought_b", NOTE_B))
        if path.startswith("/api/nudges"):
            return 200, json.dumps(nudges(self.running))
        if path.startswith("/api/cognition/pulse"):
            return 503, "n/a"
        return 404, "{}"

    def route(self, req):
        r = req.request
        path = r.url.replace(ORIGIN, "").split("#")[0]
        if r.method != "GET":
            self.writes.append(r.method + " " + path)
            req.fulfill(status=405); return
        if path == "/cognition":
            req.fulfill(status=200, content_type="text/html", body=HTML.replace("__PAGE_TOKEN__", "t")); return
        self.gets.append(path)
        status, body = self.answer(path)
        for h in self.hold:
            if path.startswith(h):
                self.held.append((req, path, status, body)); return
        req.fulfill(status=status, content_type="application/json", body=body)

    def release(self, pg, order="fifo"):
        items = list(self.held) if order == "fifo" else list(reversed(self.held))
        self.held.clear()
        for req, _, status, body in items:
            req.fulfill(status=status, content_type="application/json", body=body)
            pg.wait_for_timeout(150)
        pg.wait_for_timeout(300)

    def release_one(self, pg, index):
        req, _, status, body = self.held.pop(index)
        req.fulfill(status=status, content_type="application/json", body=body)
        pg.wait_for_timeout(450)

    def wait_held(self, pg, n):
        for _ in range(100):
            if len(self.held) >= n:
                return
            pg.wait_for_timeout(50)
        raise AssertionError(f"expected {n} held request(s), have {len(self.held)}")


SRV = Server()


def view(pg):
    """What the sheet shows now: notes | nudges | none, plus URL and focus."""
    return pg.evaluate("""()=>{const o=document.getElementById('overlay').classList.contains('open');
      const t=document.getElementById('thoughtDetail').innerText;
      const a=document.activeElement;
      return {focusText:a?(a.textContent||'').slice(0,60):'',tiles:document.querySelectorAll('.tile').length,
        hlen:history.length,banner:document.getElementById('banner').textContent,mode:document.getElementById('mode').textContent,
        dom:document.getElementById('thoughtDetail').textContent+'|'+document.getElementById('detailTitle').textContent,
        open:o,notes:!!document.getElementById('savedNotes')&&o,nudges:o&&t.includes('Nudge history for'),
        text:o?document.querySelector('.sheet').innerText:'',hash:location.hash,toast:document.getElementById('toast').textContent,
        focus:document.activeElement&&(document.activeElement.id||document.activeElement.tagName)}}""")


LIB_BETA_ONLY = dict(LIB, thoughts=[card("thought_b", "beta thought")])
LIB_THREE = dict(LIB, thoughts=LIB["thoughts"] + [card("thought_c", "gamma thought")])
LIBRARY = "/api/cognition/library"
HISTORY_HASH = "#thought=thought_a&view=nudges"


def click_refresh(pg):
    pg.evaluate("document.getElementById('refresh').click()")   # also reachable while the sheet is open


def history_open(v):
    return v["open"] and v["nudges"] and not v["notes"] and v["hash"] == HISTORY_HASH and v["focusText"].startswith("Nudge history for thought_a")


def closed_clean(v):
    """Fail closed: no sheet, no cards, and no Note or history text left anywhere in the sheet."""
    return (not v["open"] and v["tiles"] == 0 and NOTE_A not in v["dom"] and "Nudge history" not in v["dom"]
            and "alpha thought" not in v["dom"])


def settle(pg, selector):
    """Wait for an expected element, but let the following check (not a timeout) report a regression."""
    try:
        pg.wait_for_selector(selector, timeout=2500)
    except Exception:
        pass


def open_thought(pg, title):
    pg.get_by_role("button", name="Inspect " + title).click()


with sync_playwright() as p:
    b = p.chromium.launch(headless=True, args=["--no-sandbox"])
    failures = []
    passed = 0
    try:
        for (w, h, tag) in [(1280, 1000, "desktop"), (960, 1000, "half"), (390, 844, "phone")]:
            SRV.reset()
            ctx = b.new_context(viewport={"width": w, "height": h})
            pg = ctx.new_page(); errs = []
            pg.on("pageerror", lambda e: errs.append(str(e)))
            pg.route("**/*", SRV.route)

            def fresh():
                SRV.hold.clear(); SRV.deny.clear(); SRV.fail.clear(); SRV.lib = LIB; SRV.running = False
                pg.goto("about:blank"); pg.goto(ORIGIN + "/cognition"); pg.wait_for_selector(".tile")

            def check(name, cond, detail=""):
                global passed
                if cond:
                    passed += 1
                else:
                    failures.append(f"[{tag}] {name}: {detail}")

            # R1 — Codex finding 1
            fresh(); open_thought(pg, "alpha thought"); pg.wait_for_selector("#savedNotes")
            SRV.hold.add("/api/nudges")
            pg.get_by_role("button", name="Nudge history").click(); SRV.wait_held(pg, 1)
            pg.go_back(); pg.wait_for_selector("#savedNotes")
            SRV.release(pg)
            v = view(pg)
            check("R1 delayed nudge history must not overwrite Notes after Back",
                  v["notes"] and not v["nudges"] and "view=nudges" not in v["hash"] and v["focus"] == "savedNotes", v)

            # R2 — Codex finding 2
            fresh(); SRV.hold.add("/api/cognition/thoughts/thought_a")
            open_thought(pg, "alpha thought"); SRV.wait_held(pg, 1)
            SRV.hold.clear()
            pg.get_by_role("button", name="Nudge history").click(); pg.wait_for_selector("text=Nudge history for thought_a")
            SRV.release(pg)
            v = view(pg)
            check("R2 delayed Thought detail must not overwrite Nudge history",
                  v["nudges"] and not v["notes"] and "view=nudges" in v["hash"], v)

            # R3 — rapid history -> Back -> Forward, three held reads, released in both orders
            for order in ("lifo", "fifo"):
                fresh(); open_thought(pg, "alpha thought"); pg.wait_for_selector("#savedNotes")
                SRV.hold.update({"/api/nudges", "/api/cognition/thoughts/thought_a"})
                pg.get_by_role("button", name="Nudge history").click(); SRV.wait_held(pg, 1)
                pg.go_back(); SRV.wait_held(pg, 2)
                pg.go_forward(); SRV.wait_held(pg, 3)
                SRV.hold.clear(); SRV.release(pg, order)
                v = view(pg)
                check(f"R3 rapid history/back/forward ({order}) ends on the newest intent",
                      v["nudges"] and not v["notes"] and "view=nudges" in v["hash"], v)

            # R4 — cross-Thought, private Note of the abandoned Thought never shown
            fresh(); SRV.hold.add("/api/cognition/thoughts/thought_a")
            open_thought(pg, "alpha thought"); SRV.wait_held(pg, 1)
            pg.get_by_role("button", name="Close details").click()
            SRV.hold.clear()
            open_thought(pg, "beta thought"); pg.wait_for_selector("text=" + NOTE_B)
            SRV.release(pg)
            v = view(pg)
            check("R4 cross-Thought: B stays, A's private Note never appears",
                  v["notes"] and NOTE_B in v["text"] and NOTE_A not in v["text"] and "thought_b" in v["hash"], v)

            # R5 — delayed detail, then Close
            fresh(); SRV.hold.add("/api/cognition/thoughts/thought_a")
            open_thought(pg, "alpha thought"); SRV.wait_held(pg, 1)
            pg.get_by_role("button", name="Close details").click()
            toast0 = view(pg)["toast"]
            SRV.release(pg)
            v = view(pg)
            check("R5 delayed detail after Close: sheet stays closed, toast unchanged",
                  not v["open"] and v["hash"] == "" and v["toast"] == toast0, v)

            # R6 — delayed ERROR for the abandoned view
            fresh(); SRV.deny.add("/api/cognition/thoughts/thought_a"); SRV.hold.add("/api/cognition/thoughts/thought_a")
            open_thought(pg, "alpha thought"); SRV.wait_held(pg, 1)
            SRV.hold.clear()
            pg.get_by_role("button", name="Nudge history").click(); pg.wait_for_selector("text=Nudge history for thought_a")
            toast0 = view(pg)["toast"]
            SRV.release(pg)
            v = view(pg)
            check("R6 delayed error from an abandoned view: no error toast, history stays",
                  v["nudges"] and v["toast"] == toast0 and "Thought detail" not in v["toast"], v)
            SRV.deny.clear()

            # R7 — poll cancelled by navigation; held poll response ignored
            fresh(); SRV.running = True; open_thought(pg, "alpha thought"); pg.wait_for_selector("#savedNotes")
            pg.get_by_role("button", name="Nudge history").click(); pg.wait_for_selector("text=Nudge history for thought_a")
            SRV.hold.add("/api/nudges")
            n0 = len([g for g in SRV.gets if g.startswith("/api/nudges")])
            pg.wait_for_timeout(3300); SRV.wait_held(pg, 1)     # the 3-second poll fired and is held
            pg.go_back(); pg.wait_for_selector("#savedNotes")
            SRV.release(pg)
            pg.wait_for_timeout(3300)                           # a cancelled poll must not fire again
            v = view(pg)
            n1 = len([g for g in SRV.gets if g.startswith("/api/nudges")])
            check("R7 poll: held poll response ignored, no further polls after leaving the history",
                  v["notes"] and not v["nudges"] and n1 == n0 + 1, {"view": v, "polls": n1 - n0})
            SRV.running = False

            # R8 — direct link with delayed detail, then Back to the shelf
            fresh(); pg.get_by_role("button", name="Inspect beta thought").click(); pg.wait_for_selector("text=" + NOTE_B)
            pg.get_by_role("button", name="Close details").click()
            SRV.hold.add("/api/cognition/thoughts/thought_a")
            pg.goto(ORIGIN + "/cognition#thought=thought_a"); pg.wait_for_selector(".tile"); SRV.wait_held(pg, 1)
            pg.evaluate("history.back()"); pg.wait_for_timeout(400)
            SRV.hold.clear(); SRV.release(pg)
            v = view(pg)
            check("R8 direct link, delayed detail, then Back: the delayed Note does not open",
                  NOTE_A not in v["text"] and "thought_a" not in v["hash"], v)

            def to_shelf_via_history():
                """Codex's setup: Thought Notes -> nudge history -> Close (three entries after the shelf)."""
                fresh(); open_thought(pg, "alpha thought"); pg.wait_for_selector("#savedNotes")
                pg.get_by_role("button", name="Nudge history").click(); pg.wait_for_selector("text=Nudge history for thought_a")
                pg.get_by_role("button", name="Close details").click()

            # F1 — Codex's exact reproduction
            to_shelf_via_history()
            SRV.hold.add(LIBRARY); click_refresh(pg); SRV.wait_held(pg, 1)
            h0 = view(pg)["hlen"]
            pg.go_back(); pg.wait_for_timeout(500)
            before = view(pg)
            check("F1 held Refresh then Back: history open, focused and in the URL BEFORE release", history_open(before) and before["tiles"] == 2, before)
            SRV.hold.clear(); SRV.release(pg); pg.wait_for_timeout(300)
            after = view(pg)
            check("F1 held Refresh then Back: history open, focused and in the URL AFTER release", history_open(after) and after["tiles"] == 2, after)
            check("F1 no history entry added or removed", after["hlen"] == h0, {"before": h0, "after": after["hlen"]})
            pg.go_forward(); pg.wait_for_timeout(400)
            v = view(pg)
            check("F1 Forward still returns to the shelf (the entry was not replaced)", not v["open"] and v["hash"] == "", v)

            # F2 — Back then Forward while the Refresh is held
            to_shelf_via_history()
            SRV.hold.add(LIBRARY); click_refresh(pg); SRV.wait_held(pg, 1)
            pg.go_back(); pg.wait_for_timeout(400); pg.go_forward(); pg.wait_for_timeout(400)
            before = view(pg)
            SRV.hold.clear(); SRV.release(pg)
            v = view(pg)
            check("F2 held Refresh, Back, Forward: shelf stays before and after release",
                  not before["open"] and before["hash"] == "" and not v["open"] and v["hash"] == "" and v["tiles"] == 2, {"before": before, "after": v})
            pg.go_back(); pg.wait_for_timeout(500)
            check("F2 Back afterwards still opens the history", history_open(view(pg)), view(pg))

            # F3 — consecutive Refresh clicks: the newest Refresh owns the library
            for order in ("fifo", "lifo"):
                fresh(); SRV.hold.add(LIBRARY)
                SRV.lib = LIB_BETA_ONLY; click_refresh(pg); SRV.wait_held(pg, 1)     # older answer: 1 card
                SRV.lib = LIB_THREE; click_refresh(pg); SRV.wait_held(pg, 2)         # newest answer: 3 cards
                SRV.hold.clear()
                SRV.release_one(pg, 0 if order == "fifo" else 1)
                mid = view(pg)["tiles"]
                SRV.release_one(pg, 0)
                end = view(pg)["tiles"]
                check(f"F3 consecutive Refresh ({order}): only the newest answer replaces the library",
                      mid == (2 if order == "fifo" else 3) and end == 3, {"mid": mid, "end": end})

            # F4 — the held Refresh is denied after Back: fail closed, the Back URL is kept, retry recovers
            to_shelf_via_history()
            SRV.deny.add(LIBRARY); SRV.hold.add(LIBRARY); click_refresh(pg); SRV.wait_held(pg, 1)
            h0 = view(pg)["hlen"]
            pg.go_back(); pg.wait_for_timeout(500)
            SRV.hold.clear(); SRV.release(pg)
            v = view(pg)
            check("F4 denied Refresh after Back: fails closed with a truthful state",
                  closed_clean(v) and "not available" in v["banner"] and "Identity denied" in v["banner"] and v["mode"] == "EVIDENCE UNAVAILABLE"
                  and "can't be opened" in v["toast"], v)
            check("F4 denied Refresh keeps the Back destination in the URL, no entry added", v["hash"] == HISTORY_HASH and v["hlen"] == h0, v)
            SRV.deny.clear(); click_refresh(pg); pg.wait_for_timeout(700)
            check("F4 a later authorized Refresh opens the kept destination", history_open(view(pg)) and view(pg)["tiles"] == 2, view(pg))

            # F5 / F6 — Refresh denied (403) or failed (503) while Notes are open; retry recovers
            for name, bucket, word in (("F5 denied (403)", SRV.deny, "Identity denied"), ("F6 failed (503)", SRV.fail, "evidence unavailable")):
                fresh(); open_thought(pg, "alpha thought"); settle(pg, "text=" + NOTE_A)
                bucket.add(LIBRARY); click_refresh(pg); pg.wait_for_timeout(600)
                v = view(pg)
                check(f"{name} Refresh with Notes open: fails closed, private Note removed",
                      closed_clean(v) and word in v["banner"] and v["hash"] == "#thought=thought_a", v)
                bucket.clear(); click_refresh(pg); settle(pg, "text=" + NOTE_A)
                v = view(pg)
                check(f"{name} retry: an authorized Refresh restores the view", v["notes"] and v["tiles"] == 2 and v["focus"] == "savedNotes", v)

            # F7 — the card is really gone from a newly authorized library
            fresh(); open_thought(pg, "alpha thought"); settle(pg, "#savedNotes")
            pg.get_by_role("button", name="Close details").click()
            SRV.lib = LIB_BETA_ONLY; SRV.hold.add(LIBRARY); click_refresh(pg); SRV.wait_held(pg, 1)
            h0 = view(pg)["hlen"]
            pg.go_back(); settle(pg, "#savedNotes")
            before = view(pg)
            SRV.hold.clear(); SRV.release(pg)
            v = view(pg)
            check("F7 card missing from the NEW authorized library: honest unavailable route, only then",
                  before["notes"] and before["hash"] == "#thought=thought_a" and not v["open"] and v["hash"] == "" and v["tiles"] == 1
                  and "not in this identity's authorized library" in v["toast"] and NOTE_A not in v["dom"] and v["hlen"] == h0, {"before": before, "after": v})

            # F8 — a superseded DENIED Refresh is not masked while the newest is pending
            fresh(); open_thought(pg, "alpha thought"); settle(pg, "text=" + NOTE_A)
            SRV.hold.add(LIBRARY); SRV.deny.add(LIBRARY); click_refresh(pg); SRV.wait_held(pg, 1)   # older: 403
            SRV.deny.clear(); click_refresh(pg); SRV.wait_held(pg, 2)                             # newest: 200
            SRV.hold.clear(); SRV.release_one(pg, 0)
            mid = view(pg)
            SRV.release_one(pg, 0); settle(pg, "text=" + NOTE_A)
            v = view(pg)
            check("F8 superseded denial: nothing shown until the newest answers; URL kept; then restored",
                  closed_clean(mid) and mid["hash"] == "#thought=thought_a" and "refused" in mid["banner"]
                  and v["notes"] and v["tiles"] == 2 and v["hash"] == "#thought=thought_a", {"mid": mid, "after": v})

            # F9 — a superseded 200 after the newest Refresh was denied can't bring the cards back
            fresh(); open_thought(pg, "alpha thought"); settle(pg, "text=" + NOTE_A)
            SRV.hold.add(LIBRARY); click_refresh(pg); SRV.wait_held(pg, 1)                         # older: 200
            SRV.deny.add(LIBRARY); click_refresh(pg); SRV.wait_held(pg, 2)                         # newest: 403
            SRV.hold.clear(); SRV.release_one(pg, 1)
            mid = view(pg)
            SRV.release_one(pg, 0)
            v = view(pg)
            check("F9 stale 200 after the newest denial: stays failed closed",
                  closed_clean(mid) and closed_clean(v) and v["mode"] == "EVIDENCE UNAVAILABLE", {"mid": mid, "after": v})
            SRV.deny.clear()

            # F10 — held Refresh, then open another Thought: private Note isolation
            fresh(); open_thought(pg, "alpha thought"); settle(pg, "text=" + NOTE_A)
            pg.get_by_role("button", name="Close details").click()
            SRV.hold.add(LIBRARY); click_refresh(pg); SRV.wait_held(pg, 1)
            open_thought(pg, "beta thought"); settle(pg, "text=" + NOTE_B)
            h0 = view(pg)["hlen"]
            SRV.hold.clear(); SRV.release(pg); settle(pg, "text=" + NOTE_B)
            v = view(pg)
            check("F10 held Refresh then another Thought: it stays, the first Thought's Note never appears",
                  v["notes"] and NOTE_B in v["text"] and NOTE_A not in v["text"] and NOTE_A not in v["dom"]
                  and v["hash"] == "#thought=thought_b" and v["hlen"] == h0 and v["focus"] == "savedNotes", v)

            # F11 — a link followed while the FIRST library read is pending is kept, not rejected
            SRV.hold.clear(); SRV.deny.clear(); SRV.fail.clear(); SRV.lib = LIB
            pg.goto("about:blank"); SRV.hold.add(LIBRARY); pg.goto(ORIGIN + "/cognition"); SRV.wait_held(pg, 1)
            pg.evaluate("location.hash='thought=thought_a&view=nudges'"); pg.wait_for_timeout(300)
            before = view(pg)
            SRV.hold.clear(); SRV.release(pg); pg.wait_for_timeout(300)
            v = view(pg)
            check("F11 link during the first library read: kept while pending, opened on release",
                  not before["open"] and before["hash"] == HISTORY_HASH and "Waiting" in before["toast"] and history_open(v), {"before": before, "after": v})

            # F12 — held Refresh, then Close
            fresh(); open_thought(pg, "alpha thought"); settle(pg, "#savedNotes")
            SRV.hold.add(LIBRARY); click_refresh(pg); SRV.wait_held(pg, 1)
            pg.get_by_role("button", name="Close details").click()
            SRV.hold.clear(); SRV.release(pg)
            v = view(pg)
            check("F12 held Refresh then Close: the sheet stays closed", not v["open"] and v["hash"] == "" and v["tiles"] == 2, v)

            # R9
            check("R9 zero non-GET requests", not SRV.writes, SRV.writes)
            check("R9 no page errors", not errs, errs)
            pg.screenshot(path=f"/tmp/async-{tag}.png")
            ctx.close()
    finally:
        b.close()
for f in failures:
    print("FAIL:", f)
total = passed + len(failures)
print(f"{passed}/{total} delayed-response checks PASS across 1280/960/390 ({HTML_PATH})")
sys.exit(1 if failures else 0)
