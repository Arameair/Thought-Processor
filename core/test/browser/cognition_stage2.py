#!/usr/bin/env python3
"""Stage 2 (Expanded Thought, plan PR #109 §5; assignment #109 comment 6820732549): a saved Note is not
evidence that anything was done. NOT the owner's machine, NOT Nemotron. Go-shaped responses only; the
receipt codes are the ones Core's nudgeReceipt() emits (pinned by TestCognitionNudgeReceiptVocabulary).

E1  every development card carries exactly one action line, right under its Note (or missing-Note text)
E2  no nudge on the Note: "No verified action recorded for this Note."; never "was not executed"
E3  COMPLETED_VERIFIED, same request ID, IDs agree: "Verified by Core" for that nudge ONLY, and says it
    is not evidence that any tool, skill or other change ran
E4  COMPLETED_VERIFIED for a step whose Note is evicted (STEP_ONLY): verified for the step, Note not shown
E5  COMPLETED_VERIFIED but the IDs disagree, or no request ID: NOT attributed to the Note, not "Verified"
E6  COMPLETED_NOT_VERIFIED: "Unverified"
E7  NO_CORE_RECEIPT, RECOVERY_REQUIRED, UNCERTAIN: uncertain, the actual code shown, "not known"; not
    "completed", not "never ran"
E8  NOT_IN_THIS_OWNERS_JOURNAL (foreign), FAILED, NOT_DELIVERED: no verified action, code shown
E9  an unknown or hostile receipt code: uncertain, shown literally, runs nothing
E10 a receipt code without a nudge ID: no verified action (nothing to attribute)
E11 the suggested state is still "— not applied"; the Core meaning line is unchanged
E12 the only "Verified" lines are E3 and E4; no line anywhere claims non-execution
E13 opening, deep link (#thought=…&dev=…), Back/Forward and refresh send zero POSTs
E14 closing the sheet removes every Note and action line; the tiles never carry Note text
E15 1280 / 960 / 390: no sideways scrolling in the sheet; the newest card's action line is in view
    with its Note
    python3 cognition_stage2.py [path/to/cognition.html]
"""
import json
import sys
from pathlib import Path
from playwright.sync_api import sync_playwright
from cognition_scope_fixture import scoped

HTML_PATH = Path(sys.argv[1]) if len(sys.argv) > 1 else Path(__file__).resolve().parents[2] / "cmd/tass-core/cognition.html"
HTML = HTML_PATH.read_text()
ORIGIN = "http://127.0.0.1:8795"
NOW = "2026-11-12T10:00:00Z"
NOTE = "PRIVATE-STAGE2-NOTE"
HOSTILE = '<img src=x onerror="window.__xss=1">'


def nid(i):
    return "nudge_%020x" % i


def dev(req, minute, persistence="NOTE_AND_STEP_CORRELATED", nudge=None, receipt=None, note=True, suggested=""):
    d = {"request_id": req, "at": "2026-11-12T%02d:%02d:00Z" % (9 - minute // 60, 59 - minute % 60), "persistence": persistence,
         "step": {"at": NOW, "kind": "developed"} if persistence != "NOTE_ONLY" else None,
         "note": {"at": NOW, "text": f"{NOTE} {req} a saved idea.", "by": "Nemotron", "suggested_state": suggested,
                  "suggested_state_applied": False} if note else None,
         "meaning": "CORE-MEANING " + req}
    if nudge:
        d["source_nudge_id"] = nudge
    if receipt is not None:
        d["nudge_receipt"] = receipt
    return d


# (dom id, development, expected kind, phrases that must appear, phrases that must not)
CASES = [
    ("mr_plain", dev("mr_plain", 0, suggested="resolve"), "none", ["No verified action recorded for this Note."], ["Verified"]),
    ("mr_verified", dev("mr_verified", 1, nudge=nid(1), receipt="COMPLETED_VERIFIED"), "verified",
     ["Verified by Core", nid(1), "produced this Note", "not evidence that any tool, skill or other change ran"], ["No verified action"]),
    ("mr_evicted", dev("mr_evicted", 2, "STEP_ONLY", nudge=nid(2), receipt="COMPLETED_VERIFIED", note=False), "verified",
     ["Verified by Core", nid(2), "Note is no longer stored", "not evidence that any tool, skill or other change ran"], ["produced this Note"]),
    ("mr_disagree", dev("mr_disagree", 3, "IDS_DISAGREE", nudge=nid(3), receipt="COMPLETED_VERIFIED"), "uncertain",
     ["No verified action recorded for this Note", "not attributed to this Note"], ["Verified by Core"]),
    ("none", dev("", 4, "NOTE_ONLY", nudge=nid(4), receipt="COMPLETED_VERIFIED"), "uncertain",
     ["No verified action recorded for this Note", "not attributed to this Note"], ["Verified by Core"]),
    ("mr_notver", dev("mr_notver", 5, nudge=nid(5), receipt="COMPLETED_NOT_VERIFIED"), "uncertain",
     ["Unverified", nid(5), "could not verify"], ["Verified by Core"]),
    ("mr_noreceipt", dev("mr_noreceipt", 6, nudge=nid(6), receipt="NO_CORE_RECEIPT"), "uncertain",
     ["No verified action recorded", "no outcome receipt", "not known"], ["Verified by Core"]),
    ("mr_recovery", dev("mr_recovery", 7, nudge=nid(7), receipt="RECOVERY_REQUIRED"), "uncertain",
     ["Uncertain", "RECOVERY_REQUIRED", "not known", "not shown as completed"], ["Verified by Core"]),
    ("mr_uncertain", dev("mr_uncertain", 8, nudge=nid(8), receipt="UNCERTAIN"), "uncertain",
     ["Uncertain", "UNCERTAIN", "not known"], ["Verified by Core"]),
    ("mr_foreign", dev("mr_foreign", 9, nudge=nid(9), receipt="NOT_IN_THIS_OWNERS_JOURNAL"), "none",
     ["No verified action recorded", "not in this owner's nudge journal"], ["Verified by Core"]),
    ("mr_failed", dev("mr_failed", 10, nudge=nid(10), receipt="FAILED"), "none", ["No verified action recorded", "FAILED"], ["Verified by Core"]),
    ("mr_notdel", dev("mr_notdel", 11, nudge=nid(11), receipt="NOT_DELIVERED"), "none", ["No verified action recorded", "NOT_DELIVERED"], ["Verified by Core"]),
    ("mr_hostile", dev("mr_hostile", 12, nudge=nid(12), receipt=HOSTILE), "uncertain", ["Uncertain", HOSTILE], ["Verified by Core"]),
    ("mr_odd", dev("mr_odd", 13, nudge=nid(13), receipt="COMPLETED"), "uncertain", ["Uncertain", "COMPLETED"], ["Verified by Core"]),
    ("mr_orphan", dev("mr_orphan", 14, receipt="COMPLETED_VERIFIED"), "none", ["No verified action recorded for this Note."], ["Verified by Core"]),
    ("mr_empty", dev("mr_empty", 15, nudge=nid(15), receipt=""), "uncertain", ["Uncertain"], ["Verified by Core"]),
]
# Claims of non-execution the evidence cannot support (plan wording rejected by the owner, 2026-11-12).
NEVER = ["No action was executed", "was not executed", "never ran", "never executed", "did not run", "nothing was done"]

LIB = {"identity_entity_id": "ent_0123", "engine_entity_id": "eng_1", "identity_name": "Mira", "engine_commit": "c" * 40,
       "brain_revision": 7, "note": "",
       "thoughts": [{"id": "thought:thought_1", "kind": "thought", "title": "stage two (fixture)", "summary": "q",
                     "source": "Thought Engine: persistent Thought", "source_id": "thought_1", "status": "active",
                     "verification": "OBSERVED", "recorded_at": NOW, "related_cycle_ids": [], "fields": [],
                     "lifecycle": "active", "notes_count": len(CASES), "latest_output": {"kind": "DEVELOPED_NOTE", "at": NOW, "by": "engine"}}],
       "cycles": [], "outcomes": [], "library": []}
THOUGHT = {"thought_id": "thought_1", "question": "What does the fixture skill do?", "state": "active", "history_total": 1,
           "history": [{"at": NOW, "kind": "formed", "event_id": "evt_1"}],
           "developments": [c[1] for c in CASES], "developments_total": len(CASES), "traces": [], "related_cycle_ids": []}
posts, reads = [], []


def route(req):
    req = scoped(req)
    r = req.request
    url = r.url
    if url.split("#")[0] == ORIGIN + "/cognition":
        req.fulfill(status=200, content_type="text/html", body=HTML.replace("__PAGE_TOKEN__", "t")); return
    if r.method != "GET":
        posts.append((r.method, url.replace(ORIGIN, ""))); req.fulfill(status=405); return
    reads.append(url.replace(ORIGIN, ""))
    if "/api/cognition/library" in url:
        req.fulfill(status=200, content_type="application/json", body=json.dumps(LIB)); return
    if "/api/cognition/thoughts/thought_1" in url:
        req.fulfill(status=200, content_type="application/json", body=json.dumps(THOUGHT)); return
    if "/api/cognition/pulse" in url:
        req.fulfill(status=503, body="n/a"); return
    req.fulfill(status=404)


PROBE = """()=>[...document.querySelectorAll('#thoughtDetail .dev')].map(d=>{
  const kids=[...d.children], a=d.querySelectorAll('.action');
  const first=a[0], noteAt=kids.findIndex(e=>e.classList.contains('note')), actAt=first?kids.indexOf(first):-1;
  return {id:d.id, actions:a.length, kind:first?first.dataset.action||'':'', text:first?first.textContent:'',
          html:first?first.innerHTML:'', adjacent:actAt===noteAt+1, meta:(d.querySelector('.meta')||{}).textContent||'',
          meaning:(d.querySelector('.meaning')||{}).textContent||'', all:d.textContent};})"""

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
                    failures.append(f"[{tag}] {name}: {str(detail)[:600]}")

            posts.clear(); reads.clear()
            ctx = b.new_context(viewport={"width": w, "height": h})
            pg = ctx.new_page(); errs = []
            pg.on("pageerror", lambda e: errs.append(str(e)))
            pg.route("**/*", route)
            pg.goto(ORIGIN + "/cognition")
            pg.wait_for_selector(".tile")
            tiles = pg.evaluate("[...document.querySelectorAll('.tile')].map(t=>t.textContent+' '+t.getAttribute('aria-label')).join(' ')")
            check("E14 tiles carry no Note text", NOTE not in tiles, tiles)
            pg.get_by_role("button", name="Inspect stage two (fixture)").click()
            pg.wait_for_selector("#dev-mr_plain")
            cards = {c["id"]: c for c in pg.evaluate(PROBE)}
            check("E1 one card per development", len(cards) == len(CASES), list(cards))
            for dom, d, kind, want, avoid in CASES:
                c = cards.get("dev-" + dom)
                if not c:
                    check(f"E1 card dev-{dom} present", False, list(cards)); continue
                check(f"E1 dev-{dom}: exactly one action line, right under the Note", c["actions"] == 1 and c["adjacent"], c)
                check(f"dev-{dom}: kind {kind}", c["kind"] == kind, c)
                for x in want:
                    check(f"dev-{dom}: says {x!r}", x in c["text"], c["text"])
                for x in avoid:
                    check(f"dev-{dom}: does not say {x!r}", x not in c["text"], c["text"])
                for x in NEVER:
                    check(f"E12 dev-{dom}: no claim of non-execution ({x!r})", x.lower() not in c["all"].lower(), c["all"])
                check(f"E11 dev-{dom}: Core's meaning line unchanged", c["meaning"] == d["meaning"], c["meaning"])
            check("E9 hostile code is literal and runs nothing", pg.evaluate("window.__xss") is None and pg.locator("#thoughtDetail img").count() == 0
                  and "<img" not in cards.get("dev-mr_hostile", {}).get("html", "<img"), cards.get("dev-mr_hostile"))
            check("E11 suggested state still 'not applied'", "suggested state: resolve — not applied" in cards["dev-mr_plain"]["meta"], cards["dev-mr_plain"]["meta"])
            verified = sorted(k for k, c in cards.items() if "Verified by Core" in c["all"])
            check("E12 only the two receipt-backed cards say 'Verified by Core'", verified == ["dev-mr_evicted", "dev-mr_verified"], verified)
            sheet_text = pg.locator("#thoughtDetail").inner_text()
            check("E2 the plan's over-claiming wording is absent", "No action was executed" not in sheet_text, "")
            check("E15 no sideways scrolling in the sheet", pg.evaluate("(()=>{const s=document.querySelector('.sheet');return s.scrollWidth<=s.clientWidth+1})()"), tag)
            first = pg.locator("#dev-mr_plain .action")
            box, nbox, vp = None, None, pg.viewport_size
            if first.count() == 1:
                first.scroll_into_view_if_needed()
                box, nbox = first.bounding_box(), pg.locator("#dev-mr_plain .note").bounding_box()
            check("E15 newest card's action line is on screen with its Note", box and nbox and 0 <= nbox["y"] and box["y"] + box["height"] <= vp["height"]
                  and box["x"] >= 0 and box["x"] + box["width"] <= vp["width"] + 1, (box, nbox, vp))
            pg.screenshot(path=f"/tmp/stage2-{tag}.png")

            # E13: deep link, Back/Forward, refresh: still zero POSTs and the action line follows the card.
            pg.goto(ORIGIN + "/cognition#thought=thought_1&dev=mr_verified")
            pg.wait_for_selector("#dev-mr_verified")
            pg.wait_for_timeout(300)
            line = pg.locator("#dev-mr_verified .action")
            check("E13 deep link opens the same card with its verified line", line.count() == 1 and "Verified by Core" in line.inner_text(), "")
            pg.get_by_role("button", name="Close details").click(); pg.wait_for_timeout(200)
            # The rendered page without its own <script> (the script source contains the phrases).
            doc = pg.evaluate("(()=>{const d=document.documentElement.cloneNode(true);d.querySelectorAll('script').forEach(x=>x.remove());return d.outerHTML})()")
            check("E14 closing removes every Note and action line", NOTE not in doc and "Verified by Core" not in doc and "No verified action" not in doc, "")
            pg.go_back(); pg.wait_for_timeout(300)
            pg.go_forward(); pg.wait_for_timeout(500)
            pg.reload(); pg.wait_for_selector(".tile"); pg.wait_for_timeout(500)
            check("E13 zero POSTs (open, deep link, close, Back/Forward, refresh)", not posts, posts)
            check("E13 reads only GET library / thought / pulse", all(x.startswith(("/api/cognition/library", "/api/cognition/thoughts/thought_1", "/api/cognition/pulse")) for x in reads), sorted(set(reads)))
            check("no page errors", not errs, errs)
            ctx.close()
    finally:
        b.close()
for f in failures:
    print("FAIL:", f)
print(f"{passed}/{passed + len(failures)} Stage 2 expanded-Thought checks PASS across 1280/960/390 ({HTML_PATH})")
sys.exit(1 if failures else 0)
