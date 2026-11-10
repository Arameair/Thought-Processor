#!/usr/bin/env python3
"""WP0-E Repair 3 (Codex re-audit on #110): a hidden detail panel holds no private content.
NOT the owner's machine, NOT Nemotron. Synthetic fixture; GET responses are held and released, no timers.

Every check inspects the page itself, not just visibility: #detailKind, #detailTitle, #detailIntro,
#fields, #thoughtDetail, #toast, the whole closed .sheet (text AND markup) and, where the library is
no longer authorized for the record, the whole document. Cached detail state (selection, cycle
timeline) must be empty too. Owner A's markers are unique strings.

  P1  Close: the closed sheet is empty at once (the authorized shelf cards stay)
  -- Codex's six failures: open A's private Note, Close to the shelf, then
  P2  the newest Refresh is refused (403)
  P3  the newest Refresh is unavailable (503)
  P4  a newly authorized library no longer contains the card
  P5  the authorized library changes from Owner A to Owner B
  P6a a superseded 403 settles while a newer authorized GET is held
  P6b the newer authorized GET then succeeds without the old card
  -- the same transitions while the Note is still OPEN: purged when the read settles, not on reopen
  P7a 403   P7b 503   P7c identity change (URL normalized only because the card is really gone)
  -- navigation stays correct together with the purge
  P8a Notes -> history -> Close -> held Refresh -> Back: history open before and after release
  P8b the reopened history came from a NEW authorized read; Forward to the shelf leaves an empty sheet
  P8c Close -> Back reopens the same Note through a new authorized read, focused, same URL
  P9  open A, Close, open B: nothing of A in the open B view, nor after closing it
  -- delayed responses
  P10a held Thought detail, Close, release: stays empty
  P10b held Thought detail, Refresh refused, release: stays empty
  P10c held nudge history, Close, release: stays empty
  P10d held cycle timeline (Step), Close, release: nothing drawn into the closed sheet, not cached
  P11 zero non-GET requests; no page errors
Run at 1280x1000, 960x1000 and 390x844:  python3 cognition_privacy.py [path/to/cognition.html]
"""
import json
import sys
from pathlib import Path
from playwright.sync_api import sync_playwright

HTML_PATH = Path(sys.argv[1]) if len(sys.argv) > 1 else Path(__file__).resolve().parents[2] / "cmd/tass-core/cognition.html"
HTML = HTML_PATH.read_text()
ORIGIN = "http://127.0.0.1:8795"
NOW = "2026-11-10T13:00:00Z"
NOTE_A, QUESTION_A, TITLE_A, SUMMARY_A, FIELD_A = "PRIVATE-NOTE-A", "PRIVATE-QUESTION-A", "PRIVATE-TITLE-A", "PRIVATE-SUMMARY-A", "PRIVATE-FIELD-A"
NUDGE_A, STEP_A, CYCLE_A = "PRIVATE-NUDGE-A", "PRIVATE-STEP-A", "PRIVATE-CYCLE-A"
MARKERS = [NOTE_A, QUESTION_A, TITLE_A, SUMMARY_A, FIELD_A, NUDGE_A, STEP_A, CYCLE_A, "thought_a", "OWNER-A"]
NOTE_B = "Note of the second fixture thought."
LIBRARY, DETAIL_A, NUDGES, TIMELINE = "/api/cognition/library", "/api/cognition/thoughts/thought_a", "/api/nudges", "/api/cognition/cycles/cyc_1/timeline"


def card(sid, title, summary="q", fields=()):
    return {"id": "thought:" + sid, "kind": "thought", "title": title, "summary": summary, "source": "Thought Engine: persistent Thought",
            "source_id": sid, "status": "active", "verification": "OBSERVED", "recorded_at": NOW, "related_cycle_ids": [],
            "fields": [{"name": "Private field", "value": v} for v in fields]}


CARD_A = card("thought_a", TITLE_A, SUMMARY_A, [FIELD_A])
CARD_B = card("thought_b", "beta thought")
CYCLE = {"id": "cycle:cyc_1", "kind": "cycle", "title": CYCLE_A, "summary": "recorded cycle", "source": "Thought Engine: cycle",
         "source_id": "cyc_1", "status": "done", "verification": "OBSERVED", "recorded_at": NOW, "fields": []}


def library(owner, thoughts, cycles=()):
    return {"identity_entity_id": "ent_" + owner, "engine_entity_id": "eng_" + owner, "identity_name": owner, "engine_commit": "c" * 40,
            "brain_revision": 7, "note": "", "thoughts": list(thoughts), "cycles": list(cycles), "outcomes": [], "library": []}


LIB_A = library("OWNER-A", [CARD_A, CARD_B], [CYCLE])
LIB_A_WITHOUT_CARD = library("OWNER-A", [CARD_B])
LIB_B = library("OWNER-B", [card("thought_z", "owner b thought")])
# P6b and P10d need a library without Owner A's name in the banner, so the whole document can be checked.
LIB_NEUTRAL_WITHOUT_CARD = library("Mira", [CARD_B])


def thought(sid, question, text):
    return {"thought_id": sid, "question": question, "state": "active", "history_total": 1,
            "history": [{"at": NOW, "kind": "developed", "worker_request_id": "mr_" + sid}],
            "developments": [{"request_id": "mr_" + sid, "at": NOW, "persistence": "NOTE_AND_STEP_CORRELATED",
                              "note": {"at": NOW, "text": text, "by": "Nemotron (engine)", "suggested_state": "keep", "suggested_state_applied": False},
                              "step": {"at": NOW, "kind": "developed"}, "meaning": "Saved, not that it answers."}],
            "developments_total": 1, "traces": [], "related_cycle_ids": []}


NUDGE_BODY = {"nudges": [{"nudge_id": "nudge_%020x" % 1, "status": "DRAFT", "recipient_id": "self", "recipient_kind": "local_entity",
                          "reason_code": "owner_request", "requested_action": NUDGE_A, "priority": "normal", "created_at": NOW,
                          "expires_at": NOW, "source_thought_id": "thought_a", "body_digest": "d",
                          "processing": {"state": "WAITING", "reason": "x"}, "run": None}],
              "processing": {"state": "NOT_READY", "reason": "Not ready: grant not enabled."},
              "readiness": {"identity": "UNLOCKED", "process_grant": "NOT_ENABLED", "engine": "REACHABLE"}, "note": "local"}
TIMELINE_BODY = {"source": "engine journal", "engine_commit": "c" * 40, "identity_evidence": "x", "link_status": "UNLINKED",
                 "missing": [], "truncated": False, "partial": False,
                 "steps": [{"at": NOW, "stage": STEP_A, "offset": 1, "status": "ok"}]}


class Server:
    def __init__(self):
        self.writes = []
        self.reset()

    def reset(self):
        self.hold, self.held, self.deny, self.fail = set(), [], set(), set()
        self.lib = LIB_A
        self.gets = []

    def answer(self, path):
        if any(path.startswith(d) for d in self.deny):
            return 403, json.dumps({"error": "Cognition read denied"})
        if any(path.startswith(d) for d in self.fail):
            return 503, json.dumps({"error": "unavailable"})
        if path.startswith(LIBRARY):
            return 200, json.dumps(self.lib)
        if path.startswith(DETAIL_A):
            return 200, json.dumps(thought("thought_a", QUESTION_A, NOTE_A))
        if path.startswith("/api/cognition/thoughts/thought_b"):
            return 200, json.dumps(thought("thought_b", "Question B", NOTE_B))
        if path.startswith(NUDGES):
            return 200, json.dumps(NUDGE_BODY)
        if path.startswith(TIMELINE):
            return 200, json.dumps(TIMELINE_BODY)
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
        if any(path.startswith(h) for h in self.hold):
            self.held.append((req, status, body)); return
        req.fulfill(status=status, content_type="application/json", body=body)

    def release_one(self, pg, index=0):
        req, status, body = self.held.pop(index)
        req.fulfill(status=status, content_type="application/json", body=body)
        pg.wait_for_timeout(450)

    def wait_held(self, pg, n):
        for _ in range(100):
            if len(self.held) >= n:
                return
            pg.wait_for_timeout(50)
        raise AssertionError(f"expected {n} held request(s), have {len(self.held)}")

    def count(self, prefix):
        return len([g for g in self.gets if g.startswith(prefix)])


SRV = Server()

PROBE = """(markers)=>{
  const id=x=>document.getElementById(x), sheet=document.querySelector('.sheet');
  const parts={kind:id('detailKind').textContent,title:id('detailTitle').textContent,intro:id('detailIntro').textContent,
    fields:id('fields').innerHTML,thoughtDetail:id('thoughtDetail').innerHTML,toast:id('toast').textContent};
  const found=(hay)=>markers.filter(m=>hay.includes(m));
  const a=document.activeElement;
  return {open:id('overlay').classList.contains('open'),hash:location.hash,parts,
    sheetLeaks:found(sheet.innerHTML+' '+sheet.textContent),pageLeaks:found(document.body.innerHTML),
    cached:{selected:S.selected===null?null:(S.selected.source_id||'set'),timeline:S.timeline===null?null:'set'},
    tiles:document.querySelectorAll('.tile').length,banner:id('banner').textContent,mode:id('mode').textContent,
    focus:a&&(a.id||a.tagName),focusText:a?(a.textContent||'').slice(0,40):'',hlen:history.length,
    sheetText:sheet.innerText}}"""


def probe(pg):
    return pg.evaluate(PROBE, MARKERS)


def sheet_empty(v):
    """The closed detail panel holds nothing: every content element empty, no marker in text or markup, no cached detail."""
    p = v["parts"]
    return (not v["open"] and not v["sheetLeaks"] and p["kind"] == "" and p["title"] == "" and p["intro"] == "" and p["fields"] == ""
            and p["thoughtDetail"] == "" and not any(m in p["toast"] for m in MARKERS)
            and v["cached"] == {"selected": None, "timeline": None})


def page_clean(v):
    """Nothing of Owner A's record anywhere in the document."""
    return sheet_empty(v) and not v["pageLeaks"]


def settle(pg, selector):
    try:
        pg.wait_for_selector(selector, timeout=2500)
    except Exception:
        pass


def click_refresh(pg):
    pg.evaluate("document.getElementById('refresh').click()")


def brief(v):
    return {k: v[k] for k in ("open", "hash", "sheetLeaks", "pageLeaks", "parts", "cached", "tiles", "mode")}


with sync_playwright() as p:
    b = p.chromium.launch(headless=True, args=["--no-sandbox"])
    failures, passed = [], 0
    try:
        for (w, h, tag) in [(1280, 1000, "desktop"), (960, 1000, "half"), (390, 844, "phone")]:
            ctx = b.new_context(viewport={"width": w, "height": h})
            pg = ctx.new_page(); errs = []
            pg.on("pageerror", lambda e: errs.append(str(e)))
            pg.route("**/*", SRV.route)

            def fresh():
                SRV.reset()
                pg.goto("about:blank"); pg.goto(ORIGIN + "/cognition"); pg.wait_for_selector(".tile")

            def open_a():
                pg.get_by_role("button", name="Inspect " + TITLE_A).click(); pg.wait_for_selector("text=" + NOTE_A)

            def close():
                pg.get_by_role("button", name="Close details").click()

            def note_closed():
                """Codex's setup: an authorized private Note was open and is now closed to the shelf."""
                fresh(); open_a(); close()

            def check(name, cond, detail=""):
                global passed
                if cond:
                    passed += 1
                else:
                    failures.append(f"[{tag}] {name}: {detail}")

            # P1 — Close
            note_closed()
            v = probe(pg)
            check("P1 Close empties the sheet at once; the authorized shelf stays", sheet_empty(v) and v["tiles"] == 3 and v["hash"] == "", brief(v))

            # P2 / P3 — refused or unavailable Refresh while already on the shelf
            for name, bucket, word in (("P2 Refresh refused (403)", "deny", "Identity denied"), ("P3 Refresh unavailable (503)", "fail", "evidence unavailable")):
                note_closed()
                getattr(SRV, bucket).add(LIBRARY); click_refresh(pg); pg.wait_for_timeout(600)
                v = probe(pg)
                check(f"{name} on the shelf: nothing of the record remains in the page",
                      page_clean(v) and v["tiles"] == 0 and v["mode"] == "EVIDENCE UNAVAILABLE" and word in v["banner"] and v["hash"] == "", brief(v))

            # P4 — the card is gone from a newly authorized library
            note_closed()
            SRV.lib = LIB_NEUTRAL_WITHOUT_CARD; click_refresh(pg); pg.wait_for_timeout(600)
            v = probe(pg)
            check("P4 authorized Refresh without the card: nothing of the record remains", page_clean(v) and v["tiles"] == 1 and v["mode"] == "LIVE / READ ONLY", brief(v))

            # P5 — the authorized identity changes
            note_closed()
            SRV.lib = LIB_B; click_refresh(pg); pg.wait_for_timeout(600)
            v = probe(pg)
            check("P5 library changes from Owner A to Owner B: nothing of Owner A in Owner B's page",
                  page_clean(v) and v["tiles"] == 1 and "OWNER-B" in v["banner"], brief(v))

            # P6 — superseded 403 while a newer authorized GET is held, then the newer GET succeeds without the card
            note_closed()
            SRV.hold.add(LIBRARY); SRV.deny.add(LIBRARY); click_refresh(pg); SRV.wait_held(pg, 1)          # older: 403
            SRV.deny.clear(); SRV.lib = LIB_NEUTRAL_WITHOUT_CARD; click_refresh(pg); SRV.wait_held(pg, 2)  # newest: 200
            SRV.hold.clear(); SRV.release_one(pg, 0)
            v = probe(pg)
            check("P6a superseded 403 while the newest GET is held: nothing remains while the page says nothing is shown",
                  page_clean(v) and v["tiles"] == 0 and "refused" in v["banner"], brief(v))
            SRV.release_one(pg, 0)
            v = probe(pg)
            check("P6b the newest GET then succeeds without the card: still nothing of the record", page_clean(v) and v["tiles"] == 1, brief(v))

            # P7 — the same transitions while the Note is OPEN: purged when the read settles
            for name, bucket in (("P7a 403", "deny"), ("P7b 503", "fail")):
                fresh(); open_a()
                getattr(SRV, bucket).add(LIBRARY); click_refresh(pg); pg.wait_for_timeout(600)
                v = probe(pg)
                check(f"{name} while the Note is open: closed and purged when it settles; URL kept for a retry",
                      page_clean(v) and v["tiles"] == 0 and v["hash"] == "#thought=thought_a", brief(v))
            fresh(); open_a()
            SRV.lib = LIB_B; click_refresh(pg); pg.wait_for_timeout(600)
            v = probe(pg)
            check("P7c identity change while the Note is open: closed, purged, URL normalized because the card is gone",
                  sheet_empty(v) and v["hash"] == "" and v["tiles"] == 1 and not [m for m in v["pageLeaks"] if m != "thought_a"] and "OWNER-B" in v["banner"], brief(v))

            # P8 — navigation still works together with the purge (valid card remains)
            fresh(); open_a()
            pg.get_by_role("button", name="Nudge history").click(); pg.wait_for_selector("text=Nudge history for thought_a")
            close()
            n0, h0 = SRV.count(NUDGES), probe(pg)["hlen"]
            SRV.hold.add(LIBRARY); click_refresh(pg); SRV.wait_held(pg, 1)
            pg.go_back(); settle(pg, "text=Nudge history for thought_a")
            before = probe(pg)
            SRV.hold.clear(); SRV.release_one(pg, 0); settle(pg, "text=Nudge history for thought_a"); pg.wait_for_timeout(300)
            after = probe(pg)
            ok = lambda v: v["open"] and v["hash"] == "#thought=thought_a&view=nudges" and "Nudge history for thought_a" in v["sheetText"] and v["focusText"].startswith("Nudge history for")
            check("P8a held Refresh then Back: history open, focused, in the URL before and after release; no entry added",
                  ok(before) and ok(after) and after["hlen"] == h0 and after["tiles"] == 3, {"before": brief(before), "after": brief(after)})
            reads = SRV.count(NUDGES) - n0
            pg.go_forward(); pg.wait_for_timeout(400)
            v = probe(pg)
            check("P8b reopened history came from new authorized reads; Forward to the shelf leaves an empty sheet",
                  reads >= 2 and sheet_empty(v) and v["hash"] == "" and v["tiles"] == 3, {"reads": reads, **brief(v)})
            fresh(); open_a(); close()
            d0 = SRV.count(DETAIL_A)
            pg.go_back(); settle(pg, "text=" + NOTE_A)
            v = probe(pg)
            check("P8c Close then Back: the same Note returns through a new authorized read, focused, same URL",
                  v["open"] and NOTE_A in v["sheetText"] and v["hash"] == "#thought=thought_a" and v["focus"] == "savedNotes" and SRV.count(DETAIL_A) == d0 + 1, brief(v))

            # P9 — a different Thought
            note_closed()
            pg.get_by_role("button", name="Inspect beta thought").click(); settle(pg, "text=" + NOTE_B)
            v = probe(pg)
            close()
            c = probe(pg)
            check("P9 open B after closing A: nothing of A in the open B view, nor after closing it",
                  v["open"] and NOTE_B in v["sheetText"] and not v["sheetLeaks"] and v["hash"] == "#thought=thought_b" and sheet_empty(c), {"openB": brief(v), "closed": brief(c)})

            # P10 — delayed responses
            fresh(); SRV.hold.add(DETAIL_A)
            pg.get_by_role("button", name="Inspect " + TITLE_A).click(); SRV.wait_held(pg, 1)
            close(); SRV.hold.clear(); SRV.release_one(pg, 0)
            v = probe(pg)
            check("P10a held Thought detail, Close, release: the sheet stays empty", sheet_empty(v), brief(v))

            fresh(); SRV.hold.add(DETAIL_A)
            pg.get_by_role("button", name="Inspect " + TITLE_A).click(); SRV.wait_held(pg, 1)
            SRV.deny.add(LIBRARY); click_refresh(pg); pg.wait_for_timeout(600)
            SRV.hold.clear(); SRV.release_one(pg, 0)
            v = probe(pg)
            check("P10b held Thought detail, Refresh refused, release: nothing of the record in the page", page_clean(v) and v["tiles"] == 0, brief(v))

            fresh(); open_a(); SRV.hold.add(NUDGES)
            pg.get_by_role("button", name="Nudge history").click(); SRV.wait_held(pg, 1)
            close(); SRV.hold.clear(); SRV.release_one(pg, 0)
            v = probe(pg)
            check("P10c held nudge history, Close, release: the sheet stays empty", sheet_empty(v), brief(v))

            fresh(); pg.get_by_role("button", name="Inspect " + CYCLE_A).click()
            SRV.hold.add(TIMELINE); pg.get_by_role("button", name="Step", exact=True).click(); SRV.wait_held(pg, 1)
            close(); SRV.hold.clear(); SRV.release_one(pg, 0)
            v = probe(pg)
            check("P10d held cycle timeline, Close, release: nothing drawn into the closed sheet, nothing cached", sheet_empty(v), brief(v))

            # P11
            check("P11 zero non-GET requests", not SRV.writes, SRV.writes)
            check("P11 no page errors", not errs, errs)
            pg.screenshot(path=f"/tmp/privacy-{tag}.png")
            ctx.close()
    finally:
        b.close()
for f in failures:
    print("FAIL:", f)
total = passed + len(failures)
print(f"{passed}/{total} hidden-sheet privacy checks PASS across 1280/960/390 ({HTML_PATH})")
sys.exit(1 if failures else 0)
