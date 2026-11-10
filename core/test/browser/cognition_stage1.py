#!/usr/bin/env python3
"""Stage 1 slice 1 (plan PR #109 §8.2, WP1-B): truthful, metadata-only Thought cards.
NOT the owner's machine, NOT Nemotron. Go-shaped fixture responses only; zero writes.

  B1  lifecycle badge from the fixture: Active / Sleeping / Resolved; an unknown state says so
  B2  saved Note count: none, one, many, capped ("200 or more")
  B3  latest recorded development: kind, time and recorded source; "No output recorded" when none
  B4  the nudge receipt line appears ONLY with run_activity, with the recorded state's wording
  B5  no Thought tile says BUSY, WORKING or THINKING (text or accessible name), even when the Engine
      state or source is one of those words
  B6  hostile values (script tags, unknown enums) are shown as "not recorded", never as markup
  B7  the library response and the tiles carry no Note text (marker planted in the detail only)
  B8  a tile opens the same Thought's saved Notes (P0 view), by click and by keyboard (Tab + Enter)
  B9  the accessible name starts "Inspect <title>" and includes the lifecycle, count and output
  B10 a card without the new fields (older Core) is drawn exactly as before
  B11 no horizontal overflow inside a tile at 1280, 960 and 390 wide
  B12 cycle tiles are unchanged (no preview block)
  B13 zero non-GET requests; no page errors
    python3 cognition_stage1.py [path/to/cognition.html]
"""
import json
import sys
from pathlib import Path
from playwright.sync_api import sync_playwright
from cognition_scope_fixture import scoped

HTML_PATH = Path(sys.argv[1]) if len(sys.argv) > 1 else Path(__file__).resolve().parents[2] / "cmd/tass-core/cognition.html"
HTML = HTML_PATH.read_text()
ORIGIN = "http://127.0.0.1:8795"
NOW = "2026-11-10T18:20:00Z"
NOTE = "PRIVATE-NOTE-TEXT-STAGE1"
FORBIDDEN = ["BUSY", "WORKING", "THINKING"]


def card(sid, title, status, **preview):
    c = {"id": "thought:" + sid, "kind": "thought", "title": title, "summary": "question of " + sid, "source": "Thought Engine: persistent Thought",
         "source_id": sid, "status": status, "verification": "OBSERVED", "recorded_at": NOW, "related_cycle_ids": [], "fields": []}
    c.update(preview)
    return c


def out(kind, by):
    return {"kind": kind, "at": NOW, "by": by}


def run(state):
    return {"state": state, "nudge_id": "nudge_%020x" % 7, "at": NOW}


CARDS = [
    card("thought_a", "alpha active", "active", lifecycle="active", notes_count=3, latest_output=out("DEVELOPED_NOTE", "owner-nudge"),
         run_activity=run("COMPLETED_VERIFIED")),
    card("thought_b", "beta sleeping", "sleeping", lifecycle="sleeping", notes_count=1, latest_output=out("DEVELOPED_NOTE", "engine")),
    card("thought_c", "gamma resolved", "resolved", lifecycle="resolved", notes_count=200, notes_count_capped=True,
         latest_output=out("DEVELOPED_STEP_NOTE_NOT_STORED", "owner"), run_activity=run("SUBMITTED_NO_OUTCOME")),
    card("thought_d", "delta empty", "active", lifecycle="active", notes_count=0),
    card("thought_e", "epsilon odd", "BUSY", lifecycle="NOT RECORDED", notes_count=2, latest_output=out("DEVELOPED_NOTE", "NOT RECORDED"),
         run_activity=run("NEEDS_RECOVERY")),
    # Values a correct Core never sends: the page must not trust them.
    card("thought_f", "zeta hostile", "THINKING", lifecycle="WORKING", notes_count="<b>9</b>",
         latest_output=out("<script>window.__x=1</script>", "THINKING"), run_activity=run("BUSY")),
    card("thought_g", "eta stopped", "active", lifecycle="active", notes_count=1, latest_output=out("DEVELOPED_NOTE", "engine"), run_activity=run("STOPPED")),
    card("thought_h", "theta failed", "active", lifecycle="active", notes_count=1, latest_output=out("DEVELOPED_NOTE", "engine"), run_activity=run("FAILED")),
    card("thought_i", "iota uncertain", "active", lifecycle="active", notes_count=1, latest_output=out("DEVELOPED_NOTE", "engine"), run_activity=run("UNCERTAIN")),
    card("thought_old", "kappa older core", "active"),   # no preview fields at all
]
CYCLE = {"id": "cycle:tp_1", "kind": "cycle", "title": "cycle one", "summary": "recorded cycle", "source": "Thought Engine: cycle",
         "source_id": "tp_1", "status": "evaluated", "verification": "OBSERVED", "recorded_at": NOW, "fields": []}
LIB = {"identity_entity_id": "ent_0123", "engine_entity_id": "eng_1", "identity_name": "Mira", "engine_commit": "c" * 40, "brain_revision": 7,
       "note": "", "thoughts": CARDS, "cycles": [CYCLE], "outcomes": [], "library": []}
THOUGHT = {"thought_id": "thought_a", "question": "question of thought_a", "state": "active", "history_total": 1,
           "history": [{"at": NOW, "kind": "developed", "worker_request_id": "mr_1"}],
           "developments": [{"request_id": "mr_1", "at": NOW, "persistence": "NOTE_AND_STEP_CORRELATED",
                             "note": {"at": NOW, "text": NOTE, "by": "Nemotron (owner nudge)", "suggested_state": "keep", "suggested_state_applied": False},
                             "step": {"at": NOW, "kind": "developed"}, "meaning": "Saved."}],
           "developments_total": 1, "traces": [], "related_cycle_ids": []}
writes, library_bodies = [], []


def route(req):
    req = scoped(req)
    r = req.request
    path = r.url.replace(ORIGIN, "").split("#")[0]
    if r.method != "GET":
        writes.append(r.method + " " + path); req.fulfill(status=405); return
    if path == "/cognition":
        req.fulfill(status=200, content_type="text/html", body=HTML.replace("__PAGE_TOKEN__", "t")); return
    if path.startswith("/api/cognition/library"):
        body = json.dumps(LIB); library_bodies.append(body)
        req.fulfill(status=200, content_type="application/json", body=body); return
    if path.startswith("/api/cognition/thoughts/thought_a"):
        req.fulfill(status=200, content_type="application/json", body=json.dumps(THOUGHT)); return
    if path.startswith("/api/cognition/pulse"):
        req.fulfill(status=200, content_type="application/json", body=json.dumps(
            {"control": "PAUSED", "every_s": 20, "state": "READY", "queued": 0, "running": 0, "junctions": {}, "recent": [], "note": ""})); return
    req.fulfill(status=404, content_type="application/json", body="{}")


TILES = """()=>[...document.querySelectorAll('.tile')].map(t=>({label:t.getAttribute('aria-label'),text:t.innerText,html:t.innerHTML,
  status:(t.querySelector('.status')||{}).textContent||'',preview:!!t.querySelector('.preview'),run:(t.querySelector('.preview .run')||{}).textContent||'',
  over:t.scrollWidth-t.clientWidth,inner:[...t.querySelectorAll('*')].some(e=>e.scrollWidth>e.clientWidth+1&&getComputedStyle(e).overflowX==='visible'),
  w:Math.round(t.getBoundingClientRect().width)}))"""

failures, passed = [], 0
with sync_playwright() as p:
    b = p.chromium.launch(headless=True, args=["--no-sandbox"])
    try:
        for (w, h, tag) in [(1280, 1000, "desktop"), (960, 1000, "half"), (390, 844, "phone")]:
            ctx = b.new_context(viewport={"width": w, "height": h}, timezone_id="UTC", locale="en-US")
            pg = ctx.new_page(); errs = []
            pg.on("pageerror", lambda e: errs.append(str(e)))
            pg.route("**/*", route)
            pg.goto(ORIGIN + "/cognition"); pg.wait_for_selector(".tile")

            def check(name, cond, detail=""):
                global passed
                if cond:
                    passed += 1
                else:
                    failures.append(f"[{tag}] {name}: {str(detail)[:400]}")

            tiles = {t["label"].split(".")[0].replace("Inspect ", ""): t for t in pg.evaluate(TILES)}
            T = lambda k: tiles.get(k, {"text": "", "label": "", "status": "", "preview": False, "run": "", "html": ""})

            check("B1 lifecycle badges", [T(k)["status"] for k in ("alpha active", "beta sleeping", "gamma resolved", "epsilon odd")]
                  == ["Active", "Sleeping", "Resolved", "State not recorded"], {k: T(k)["status"] for k in tiles})
            check("B2 saved Note counts", "3 saved Notes" in T("alpha active")["text"] and "1 saved Note\n" in T("beta sleeping")["text"] + "\n"
                  and "200 or more saved Notes" in T("gamma resolved")["text"] and "No saved Notes" in T("delta empty")["text"],
                  [T(k)["text"] for k in ("alpha active", "beta sleeping", "gamma resolved", "delta empty")])
            check("B3 latest recorded development", "Latest: Developed Note · " in T("alpha active")["text"] and "· supervised nudge" in T("alpha active")["text"]
                  and "· Engine" in T("beta sleeping")["text"] and "Latest: Developed step, Note not stored" in T("gamma resolved")["text"]
                  and "owner request in the Engine" in T("gamma resolved")["text"] and "No output recorded" in T("delta empty")["text"]
                  and "source not recorded" in T("epsilon odd")["text"] and "2026" in T("alpha active")["text"],
                  [T(k)["text"] for k in ("alpha active", "beta sleeping", "gamma resolved", "delta empty", "epsilon odd")])
            runs = {k: T(k)["run"].split(" · 11/")[0] for k in tiles}
            check("B4 nudge receipt line only with run_activity, recorded wording",
                  runs["alpha active"].startswith("Nudge completed · verified") and runs["gamma resolved"].startswith("Nudge submitted · no outcome recorded yet")
                  and runs["epsilon odd"].startswith("Nudge needs recovery") and runs["eta stopped"].startswith("Nudge stopped · model never asked")
                  and runs["theta failed"].startswith("Nudge failed") and runs["iota uncertain"].startswith("Nudge uncertain")
                  and runs["beta sleeping"] == "" and runs["delta empty"] == "" and runs["zeta hostile"] == "" and runs["kappa older core"] == "", runs)
            thought_tiles = [t for k, t in tiles.items() if k != "cycle one" and k != "kappa older core"]
            bad = [(t["label"][:40], f) for t in thought_tiles for f in FORBIDDEN if f in t["text"].upper() or f in t["label"].upper()]
            check("B5 no Thought tile says BUSY, WORKING or THINKING", len(thought_tiles) == 9 and not bad, bad)
            z = T("zeta hostile")
            check("B6 hostile values shown as not recorded, never as markup", z["status"] == "State not recorded" and "Saved Notes: not recorded" in z["text"]
                  and "No output recorded" in z["text"] and "<script" not in z["html"] and "<b>" not in z["html"] and pg.evaluate("window.__x") is None, z)
            check("B7 no Note text in the library response or on any tile", library_bodies and all(NOTE not in x for x in library_bodies)
                  and all(NOTE not in t["html"] and NOTE not in (t["label"] or "") for t in tiles.values()))
            a = T("alpha active")
            check("B9 accessible name: Inspect <title>, lifecycle, count, output, receipt", a["label"].startswith("Inspect alpha active. Active. 3 saved Notes. Latest: Developed Note")
                  and "Nudge completed · verified" in a["label"], a["label"])
            old = T("kappa older core")
            check("B10 a card without preview fields is drawn as before", old["label"] == "Inspect kappa older core" and old["status"] == "active" and not old["preview"], old)
            over = {k: (t["over"], t["inner"], t["w"]) for k, t in tiles.items() if t["over"] > 1 or t["inner"]}
            check("B11 no horizontal overflow inside a tile", not over, over)
            cy = T("cycle one")
            check("B12 cycle tiles unchanged", cy["label"] == "Inspect cycle one" and not cy["preview"] and cy["status"] == "evaluated", cy)

            # B8 — click, then keyboard
            pg.get_by_role("button", name="Inspect alpha active").click(); pg.wait_for_selector("text=" + NOTE)
            v = pg.evaluate("({hash:location.hash,notes:!!document.getElementById('savedNotes'),focus:document.activeElement&&document.activeElement.id})")
            check("B8 click opens the same Thought's saved Notes", v == {"hash": "#thought=thought_a", "notes": True, "focus": "savedNotes"}, v)
            pg.get_by_role("button", name="Close details").click()
            pg.evaluate("document.getElementById('search').focus()")
            for _ in range(12):
                pg.keyboard.press("Tab")
                if pg.evaluate("(document.activeElement.getAttribute('aria-label')||'').startsWith('Inspect alpha active')"):
                    break
            focused = pg.evaluate("document.activeElement.getAttribute('aria-label')||''")
            pg.keyboard.press("Enter"); pg.wait_for_selector("text=" + NOTE)
            v = pg.evaluate("({hash:location.hash,notes:!!document.getElementById('savedNotes')})")
            check("B8 keyboard (Tab + Enter) opens the same Thought", focused.startswith("Inspect alpha active") and v == {"hash": "#thought=thought_a", "notes": True}, {"focused": focused, **v})

            check("B13 zero non-GET requests", not writes, writes)
            check("B13 no page errors", not errs, errs)
            pg.get_by_role("button", name="Close details").click()
            pg.screenshot(path=f"/tmp/stage1-{tag}.png", full_page=False)
            ctx.close()
    finally:
        b.close()
for f in failures:
    print("FAIL:", f)
print(f"{passed}/{passed + len(failures)} Stage 1 card checks PASS across 1280/960/390 ({HTML_PATH})")
sys.exit(1 if failures else 0)
