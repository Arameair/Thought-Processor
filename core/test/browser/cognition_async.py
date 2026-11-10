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

    def answer(self, path):
        for d in self.deny:
            if path.startswith(d):
                return 403, json.dumps({"error": "Cognition read denied"})
        if path.startswith("/api/cognition/library"):
            return 200, json.dumps(LIB)
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
      return {open:o,notes:!!document.getElementById('savedNotes')&&o,nudges:o&&t.includes('Nudge history for'),
        text:o?document.querySelector('.sheet').innerText:'',hash:location.hash,toast:document.getElementById('toast').textContent,
        focus:document.activeElement&&(document.activeElement.id||document.activeElement.tagName)}}""")


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
                SRV.hold.clear(); SRV.deny.clear(); SRV.running = False
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
