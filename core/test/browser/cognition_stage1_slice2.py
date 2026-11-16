#!/usr/bin/env python3
"""Stage 1 slice 2 (plan PR #109 §4, WP1-C + WP1-D): activity strip, automatic refresh, staleness.
NOT the owner's machine, NOT Nemotron. A scripted fixture stands in for Core; the page clock is
installed and driven by the test, so no outcome depends on a sleep. Responses can be HELD and released.

WP1-C  activity strip (what the Engine RECORDED, in plain words)
  S1   PAUSED: the owner's setting, nothing processed automatically; not shown as an error
  S2   RUNNING / READY: processing is on, no job running
  S3   RUNNING / WORKING and BUSY: the recorded active job; said plainly when none is recorded
  S4   RUNNING / BLOCKED: the recorded reason code in words; "not recorded" when there is none
  S5   UNKNOWN control or state: UNAVAILABLE, no activity
  S6   a failed read after a good one: UNAVAILABLE since <time>; nothing of the previous answer remains
  S7   last pulse time, queued / running counts and "as of" are shown
  S8   the tick history is collapsed under "Recent ticks" and opens on request
  S9   RUNNING with no tick for more than 3 intervals: "No pulse tick recorded since"
  S10  hostile values are text, never markup; no strip width overflow
WP1-D  automatic refresh (a GET of the library every 60 s while visible; same rules as Refresh)
  A1   a card's new facts and a new card appear with no click; no duplicate tiles after 5 reads;
       unchanged tiles are the SAME elements (no full redraw); search text and filter are kept
  A2   an open Thought survives 3 automatic reads: same view, URL, history length, sheet scroll,
       focus and rail scroll; zero extra detail reads
  A3   the open Thought's record changed: a "Show latest" line appears, the Note on screen is not
       replaced until the owner asks; one detail read when they do
  A4   a card missing from ONE complete authorized read is gone (tile and, if open, the sheet: closed,
       purged, URL replaced, visible notice). No two-read grace period
  A5   a failed (503) automatic read FAILS CLOSED: no cards, no sheet, no Note text in the page,
       the banner says so and names the next automatic attempt; recovery reopens the URL's view
  A6   persistent failure: retries at 60/120/240/300/300 s, then stop (5 retries); Refresh starts again
  A7   a refused (403) automatic read fails closed and is NOT retried automatically
  A8   Identity entity / Engine entity / link changed: nothing of the previous context remains
  A9   authorization revoked while an automatic read is in flight: (a) the read itself is refused;
       (b) another read is refused first, then the automatic read answers 200: it is discarded
  A10  out of order: a held automatic read released after a newer Refresh changes nothing;
       an automatic read never starts while a library read is pending
  A11  Back / Forward across automatic reads; an automatic read adds no history entry
  A12  hidden tab: zero requests for 10 minutes; returning re-checks authorization (P0 resume rule)
  A13  a library read that never answers is aborted after 30 s and fails closed; never two at once
  A14  manual Refresh keeps its P0 behaviour (closes and re-reads the open view) and restarts the timer
  A15  10 minutes visible: zero non-GET requests, only library and pulse are read, at most one
       library and six pulse reads per minute
  A16  pulse failures back off (20 s, 40 s … 5 min) and recover
  A17  Note text never appears in a library response, on a tile, or in a closed sheet
Every case: zero page errors.
    python3 cognition_stage1_slice2.py [path/to/cognition.html]
"""
import copy
import json
import sys
from pathlib import Path
from urllib.parse import urlparse
from playwright.sync_api import sync_playwright

HTML_PATH = Path(sys.argv[1]) if len(sys.argv) > 1 else Path(__file__).resolve().parents[2] / "cmd/tass-core/cognition.html"
HTML = HTML_PATH.read_text()
ORIGIN = "http://127.0.0.1:8795"
T0 = "2026-11-11T15:00:00Z"
NOTE = "PRIVATE-NOTE-TEXT-SLICE2"
LIBRARY, PULSE, DETAIL, NUDGES = "/api/cognition/library", "/api/cognition/pulse", "/api/cognition/thoughts/", "/api/nudges"
OWNERS = {"A": {"identity": "ent_aaaaaaaaaaaa", "engine": "eng_1", "link": "dec_a0000001", "name": "Mira"},
          "B": {"identity": "ent_bbbbbbbbbbbb", "engine": "eng_2", "link": "dec_b0000001", "name": "Nova"}}


def card(o, sid, notes=1, at=T0, kind="thought"):
    c = {"id": kind + ":" + sid, "kind": kind, "title": f"{o}-title {sid}", "summary": f"question of {sid}", "source": "Thought Engine",
         "source_id": sid, "status": "active", "verification": "OBSERVED", "recorded_at": at, "related_cycle_ids": [], "fields": []}
    if kind == "thought":
        c.update(lifecycle="active", notes_count=notes, latest_output={"kind": "DEVELOPED_NOTE", "at": at, "by": "engine"})
    return c


def pulse(control="PAUSED", state="READY", **kw):
    p = {"observed_at": T0, "control": control, "every_s": 20, "state": state, "queued": 0, "running": 0, "core_tick_every_s": 20,
         "junctions": {}, "recent": [], "note": "Recorded by the Engine; this page only reads it."}
    p.update(kw)
    return p


def tick(n, state="READY", at=T0, job=""):
    t = {"pulse_id": "pulse_%06d" % n, "at": at, "trigger": "timer", "state": state}
    if job:
        t["job_id"] = job
    return t


class Server:
    def __init__(self):
        self.owner = "A"
        self.scope = dict(OWNERS)          # owner -> scope (a test may change engine/link)
        self.cards = {"A": [card("A", "thought_1"), card("A", "thought_2"), card("A", "tp_1", kind="cycle")], "B": [card("B", "thought_1")]}
        self.observed = T0
        self.pulse = pulse()
        self.status = {}                   # path prefix -> HTTP status to answer instead
        self.hold, self.held = set(), []
        self.log = []                      # (method, path)
        self.bodies = []                   # every library body sent
        self.note_n = 1

    def lib(self):
        ow = self.scope[self.owner]
        cs = copy.deepcopy(self.cards[self.owner])
        return {"identity_entity_id": ow["identity"], "engine_entity_id": ow["engine"], "identity_name": ow["name"], "engine_commit": "c" * 40,
                "observed_at": self.observed, "brain_revision": 7, "note": "", "thoughts": [c for c in cs if c["kind"] == "thought"],
                "cycles": [c for c in cs if c["kind"] == "cycle"], "outcomes": [], "library": [], "junctions": [],
                "link": {"state": "LINKED", "candidates": [], "unscoped_cycles": 0, "other_entity_cycles": 0, "explanation": "Linked by the owner.",
                         "link": {"identity_entity_id": ow["identity"], "engine_entity_id": ow["engine"], "engine_display_name": ow["name"],
                                  "linked_at": T0, "identity_decision_id": ow["link"], "by": "owner"}}}

    def headers(self):
        ow = self.scope[self.owner]
        return {"X-Cognition-Identity": ow["identity"], "X-Cognition-Engine": ow["engine"], "X-Cognition-Link": ow["link"]}

    def answer(self, method, path, hdr):
        if method != "GET":
            return 405, {}, {}
        for prefix, st in self.status.items():
            if path.startswith(prefix):
                return st, {"error": "refused" if st == 403 else "unavailable"}, {}
        scope = self.headers()
        if path.startswith(LIBRARY):
            return 200, self.lib(), scope
        expect = (hdr.get("x-cognition-expect-identity"), hdr.get("x-cognition-expect-engine"), hdr.get("x-cognition-expect-link"))
        if expect[0] is not None and expect != tuple(scope.values()):
            return 412, {"error": "scope changed", "code": "SCOPE_MISMATCH"}, {}
        if path.startswith(PULSE):
            return 200, self.pulse, scope
        if path.startswith(DETAIL):
            sid = path[len(DETAIL):]
            if not any(c["source_id"] == sid and c["kind"] == "thought" for c in self.cards[self.owner]):
                return 503, {"error": "NOT RECORDED"}, {}
            devs = [{"request_id": "mr_%d" % i, "at": T0, "persistence": "NOTE_AND_STEP_CORRELATED",
                     "note": {"at": T0, "text": f"{NOTE}-{self.owner}-{sid}-{i} " + "long line of saved text. " * 40, "by": "engine", "suggested_state": "keep"},
                     "step": {"at": T0, "kind": "developed"}, "meaning": "Saved."} for i in range(self.note_n, 0, -1)]
            return 200, {"thought_id": sid, "question": "question of " + sid, "state": "active", "history_total": len(devs),
                         "history": [{"at": T0, "kind": "developed", "worker_request_id": d["request_id"]} for d in devs],
                         "developments": devs, "developments_total": len(devs), "traces": [], "related_cycle_ids": []}, scope
        if path.startswith(NUDGES):
            return 200, {"owner_entity_id": self.scope[self.owner]["identity"], "nudges": [], "processing": {"state": "NOT_READY", "reason": "Not ready."},
                         "readiness": {"identity": "UNLOCKED", "process_grant": "NOT_ENABLED", "engine": "REACHABLE"}, "note": "local"}, scope
        return 404, {}, {}

    def route(self, req):
        r = req.request
        path = urlparse(r.url).path
        if path == "/cognition" and r.method == "GET":
            req.fulfill(status=200, content_type="text/html", body=HTML.replace("__PAGE_TOKEN__", "t")); return
        if not path.startswith("/api/"):
            req.fulfill(status=404, body=""); return
        self.log.append((r.method, path))
        hdr = {k.lower(): v for k, v in r.headers.items()}
        if any(path.startswith(h) for h in self.hold):
            self.held.append((req, r.method, path, hdr)); return
        self.send(req, r.method, path, hdr)

    def send(self, req, method, path, hdr):
        status, body, headers = self.answer(method, path, hdr)
        text = json.dumps(body)
        if path.startswith(LIBRARY) and status == 200:
            self.bodies.append(text)
        try:
            req.fulfill(status=status, content_type="application/json", body=text, headers=headers)
        except Exception:
            pass                           # the page aborted the request (time limit): nothing to answer

    def release(self, pg, index=0):
        """Answer a held request NOW, with what the fixture would say now."""
        req, method, path, hdr = self.held.pop(index)
        self.send(req, method, path, hdr)
        pg.wait_for_timeout(250)

    def release_as(self, pg, status_or_none, index=0):
        """Answer a held request with what the fixture said when it was ASKED (a delayed response)."""
        req, method, path, hdr = self.held.pop(index)
        status, body, headers = status_or_none
        req.fulfill(status=status, content_type="application/json", body=json.dumps(body), headers=headers)
        pg.wait_for_timeout(250)

    def n(self, prefix, method="GET"):
        return len([1 for m, p in self.log if m == method and p.startswith(prefix)])


PROBE = """()=>{
  const id=x=>document.getElementById(x), ov=id('overlay'), open=ov.classList.contains('open');
  const act=id('activity'), notice=id('notice'), fresh=id('fresh'), ticks=document.querySelector('#pulse details');
  const vis=e=>{if(!e)return false;const r=e.getBoundingClientRect(),s=getComputedStyle(e);return s.display!=='none'&&s.visibility!=='hidden'&&r.width>0&&r.height>0;};
  const a=document.activeElement;
  return {open,hash:location.hash,hlen:history.length,
    tiles:[...document.querySelectorAll('.tile')].map(t=>t.getAttribute('aria-label')),
    ids:[...document.querySelectorAll('.tile')].map(t=>t.dataset.id||''),
    doc:document.documentElement.outerHTML,
    sheet:open?document.querySelector('.sheet').innerText:'',detail:id('thoughtDetail').textContent,
    title:id('detailTitle').textContent,
    banner:id('banner').textContent,mode:id('mode').textContent,summary:id('summary').textContent,
    notice:notice.textContent,noticeVisible:vis(notice),
    pulse:id('pulse').innerText,pulseAll:id('pulse').textContent,pulseHTML:id('pulse').innerHTML,
    activity:act?act.textContent:'',code:act?act.dataset.activity:'',actClass:act?act.className:'',
    ticksOpen:ticks?ticks.open:null,ticksSummary:ticks?ticks.querySelector('summary').textContent:'',
    pulseOver:id('pulse').scrollWidth-id('pulse').clientWidth,
    fresh:vis(fresh),freshText:fresh?fresh.textContent:'',
    focus:a?(a.id||a.getAttribute('aria-label')||a.tagName):'',
    sheetTop:open?document.querySelector('.sheet').scrollTop:0,
    rail:(document.querySelector('.rail')||{scrollLeft:0}).scrollLeft,
    search:id('search').value,selected:typeof S==='object'&&S.selected?S.selected.source_id:null}}"""

failures, passed = [], 0
with sync_playwright() as p:
    browser = p.chromium.launch(headless=True, args=["--no-sandbox"])
    try:
        for (w, h, tag) in [(1280, 1000, "desktop"), (960, 1000, "half"), (390, 844, "phone")]:

            def check(name, cond, detail=""):
                global passed
                if cond:
                    passed += 1
                else:
                    failures.append(f"[{tag}] {name}: {str(detail)[:700]}")

            def case(frag="", srv=None, hold=()):
                srv = srv or Server()
                srv.hold.update(hold)
                ctx = browser.new_context(viewport={"width": w, "height": h}, timezone_id="UTC", locale="en-US")
                pg = ctx.new_page()
                pg.set_default_timeout(6000)
                pg.errs = []
                pg.on("pageerror", lambda e: pg.errs.append(str(e)))
                pg.add_init_script("""window.__setHidden=(h)=>{Object.defineProperty(document,'hidden',{configurable:true,get:()=>h});
                  Object.defineProperty(document,'visibilityState',{configurable:true,get:()=>h?'hidden':'visible'});
                  document.dispatchEvent(new Event('visibilitychange'));};""")
                pg.clock.install(time="2026-11-11T15:00:00Z")
                pg.route("**/*", srv.route)
                pg.goto(ORIGIN + "/cognition" + frag)
                pg.clock.pause_at("2026-11-11T15:00:01Z")      # from here the page clock moves only when the test moves it
                return srv, ctx, pg

            def ready(pg):
                pg.wait_for_selector(".tile"); pg.wait_for_timeout(200)

            def run(pg, ms, settle=200):
                """Advance the page clock; then give held/instant answers real time to be applied."""
                pg.clock.run_for(ms); pg.clock.run_for(5); pg.wait_for_timeout(settle)

            def minutes(pg, n, step=10000):
                for _ in range(int(n * 60000 / step)):
                    pg.clock.run_for(step); pg.clock.run_for(5); pg.wait_for_timeout(40)
                pg.wait_for_timeout(200)

            def probe(pg):
                return pg.evaluate(PROBE)

            def brief(v):
                return {k: (x[:160] if isinstance(x, str) else x) for k, x in v.items() if k not in ("doc", "pulseHTML")}

            def done(srv, ctx, pg, name):
                check(name + " / no page errors", not pg.errs, pg.errs)
                check(name + " / zero non-GET requests", not [x for x in srv.log if x[0] != "GET"], [x for x in srv.log if x[0] != "GET"])
                ctx.close()

            def open_thought(pg, o="A", sid="thought_1"):
                pg.get_by_role("button", name=f"Inspect {o}-title {sid}").click()
                pg.wait_for_selector(f"text={NOTE}-{o}-{sid}-1")

            def closed_clean(v):
                return (not v["open"]) and v["tiles"] == [] and NOTE not in v["doc"] and v["selected"] is None and v["title"] == ""

            def strip(p_):
                srv = Server(); srv.pulse = p_
                srv, ctx, pg = case(srv=srv); ready(pg)
                try:
                    pg.wait_for_selector("#activity", timeout=3000)
                except Exception:
                    pass
                return srv, ctx, pg, probe(pg)

            def attempt(name, fn):
                """One case. A crash (for example a control this page does not have) is a failed check, not a stopped run."""
                before = len(browser.contexts)
                try:
                    fn()
                except Exception as e:
                    check(name + " / case completed", False, repr(e).replace("\n", " "))
                    for c in browser.contexts[before:]:
                        c.close()
                else:
                    check(name + " / case completed", True)

            def case_s1():
                srv, ctx, pg, v = strip(pulse("PAUSED", "WORKING", last=tick(61, "WORKING", job="job_old"), recent=[tick(61, "WORKING", job="job_old")]))
                check("S1 PAUSED: the owner's setting, nothing processed automatically", v["code"] == "PAUSED" and "Paused" in v["activity"]
                      and "owner's setting" in v["activity"] and "Nothing is processed automatically" in v["activity"], brief(v))
                check("S1 PAUSED is not shown as an error or as work", "warn" not in v["actClass"] and "bad" not in v["actClass"]
                      and "Working" not in v["activity"] and "UNAVAILABLE" not in v["pulse"], brief(v))
                done(srv, ctx, pg, "S1")

            attempt("S1", case_s1)

            def case_s2():
                srv, ctx, pg, v = strip(pulse("RUNNING", "READY", last=tick(62), recent=[tick(62), tick(61)], queued=2, running=0, observed_at="2026-11-11T15:00:05Z"))
                check("S2 RUNNING/READY: processing on, no job running", v["code"] == "READY" and "Ready" in v["activity"] and "No job is running" in v["activity"], brief(v))
                check("S7 last pulse time, counts and 'as of' are shown", "pulse_000062" in v["pulse"] and "3:00:00 PM" in v["pulse"] and "2 / 0" in v["pulse"]
                      and "3:00:05 PM" in v["pulse"] and "every 20 s" in v["pulse"], brief(v))
                check("S8 tick history collapsed under 'Recent ticks'", v["ticksOpen"] is False and "Recent ticks (2 recorded)" in v["ticksSummary"]
                      and "pulse_000061" not in v["pulse"] and "pulse_000061" in v["pulseAll"], brief(v))
                pg.locator("#pulse details summary").click(); pg.wait_for_timeout(100)
                v2 = probe(pg)
                check("S8 the tick history opens on request", v2["ticksOpen"] is True and "pulse_000061" in v2["pulse"], brief(v2))
                check("S10 no strip width overflow", v["pulseOver"] <= 1 and v2["pulseOver"] <= 1, (v["pulseOver"], v2["pulseOver"]))
                done(srv, ctx, pg, "S2")

            attempt("S2 S7 S8 S10", case_s2)

            def case_s3():
                for state in ("WORKING", "BUSY"):
                    srv, ctx, pg, v = strip(pulse("RUNNING", state, active_job="job_7", running=1, last=tick(63, state, job="job_7"), recent=[tick(63, state, job="job_7")]))
                    check(f"S3 RUNNING/{state}: the recorded active job is shown", v["code"] == state and "job_7" in v["activity"], brief(v))
                    done(srv, ctx, pg, "S3 " + state)
                    srv, ctx, pg, v = strip(pulse("RUNNING", state))
                    check(f"S3 RUNNING/{state} without a recorded job says so, invents none", v["code"] == state and "ctive job is recorded" in v["activity"]
                          and "job_" not in v["activity"], brief(v))
                    done(srv, ctx, pg, "S3 no job " + state)

            attempt("S3", case_s3)

            def case_s4():
                srv, ctx, pg, v = strip(pulse("RUNNING", "BLOCKED", blocked_reason="AUTHORIZATION_DENIED", queued=1,
                                              last=tick(64, "BLOCKED", job="job_9"), recent=[tick(64, "BLOCKED", job="job_9")]))
                check("S4 BLOCKED: the recorded reason in words, the waiting job, nothing ran", v["code"] == "BLOCKED" and "authorization was denied" in v["activity"]
                      and "job_9" in v["activity"] and "nothing ran" in v["activity"], brief(v))
                done(srv, ctx, pg, "S4")
                for reason in ("NOT RECORDED", None, "<img src=x onerror=window.__x=1>"):
                    kw = {} if reason is None else {"blocked_reason": reason}
                    srv, ctx, pg, v = strip(pulse("RUNNING", "BLOCKED", **kw))
                    check(f"S4 BLOCKED with reason {reason!r}: 'not recorded', nothing invented", v["code"] == "BLOCKED" and "reason is not recorded" in v["activity"]
                          and "denied" not in v["activity"] and "<img" not in v["pulseHTML"] and pg.evaluate("window.__x") is None, brief(v))
                    done(srv, ctx, pg, "S4 none")

            attempt("S4", case_s4)

            def case_s5():
                for control, state in (("UNKNOWN", "READY"), ("RUNNING", "UNKNOWN"), ("RUNNING", "NOT RECORDED"), ("<b>x</b>", "WORKING")):
                    srv, ctx, pg, v = strip(pulse(control, state, active_job="job_7"))
                    check(f"S5 {control}/{state}: UNAVAILABLE, no activity claimed", v["code"] == "UNAVAILABLE" and "UNAVAILABLE" in v["activity"]
                          and "Working" not in v["activity"] and "Ready" not in v["activity"] and "job_7" not in v["activity"] and "<b>x</b>" not in v["pulseHTML"], brief(v))
                    done(srv, ctx, pg, "S5")

            attempt("S5", case_s5)

            def case_s6():
                srv, ctx, pg, v = strip(pulse("RUNNING", "WORKING", active_job="job_7", running=1, last=tick(65, "WORKING", job="job_7")))
                check("S6 (before) the working job is shown", "job_7" in v["activity"], brief(v))
                srv.status[PULSE] = 503
                run(pg, 10500)
                v = probe(pg)
                check("S6 failed pulse read: UNAVAILABLE since <time>, nothing of the previous answer", v["code"] == "UNAVAILABLE" and "UNAVAILABLE since 3:00:1" in v["activity"]
                      and "job_7" not in v["pulseAll"] and "pulse_000065" not in v["pulseAll"] and "Working" not in v["pulseAll"]
                      and "Nothing was started" in v["activity"] and len(v["tiles"]) == 3, brief(v))
                done(srv, ctx, pg, "S6")

            attempt("S6", case_s6)

            def case_s9():
                srv, ctx, pg, v = strip(pulse("RUNNING", "READY", last=tick(66, at="2026-11-11T14:58:00Z"), observed_at="2026-11-11T15:00:00Z"))
                check("S9 RUNNING, last tick older than 3 intervals: said plainly", "No pulse tick recorded since" in v["pulse"] and "2:58:00 PM" in v["pulse"], brief(v))
                done(srv, ctx, pg, "S9")
                srv, ctx, pg, v = strip(pulse("RUNNING", "READY", last=tick(66, at="2026-11-11T14:59:30Z"), observed_at="2026-11-11T15:00:00Z"))
                check("S9 a recent tick: no staleness line", "No pulse tick recorded since" not in v["pulse"], brief(v))
                done(srv, ctx, pg, "S9 fresh")

            attempt("S9", case_s9)

            def case_a1():
                srv, ctx, pg = case(); ready(pg)
                pg.evaluate("document.querySelectorAll('.tile').forEach((t,i)=>t.__mark='m'+i)")
                v0 = probe(pg)
                srv.cards["A"][0] = card("A", "thought_1", notes=2, at="2026-11-11T15:00:30Z")
                srv.cards["A"].insert(0, card("A", "thought_new"))
                srv.observed = "2026-11-11T15:01:00Z"
                n0 = srv.n(LIBRARY)
                run(pg, 61000)
                v = probe(pg)
                marks = pg.evaluate("[...document.querySelectorAll('.tile')].map(t=>t.__mark||'')")
                check("A1 new facts and a new card appear with no click", srv.n(LIBRARY) == n0 + 1 and any("thought_1. Active. 2 saved Notes" in t for t in v["tiles"])
                      and any("thought_new" in t for t in v["tiles"]) and len(v["tiles"]) == 4, {"reads": srv.n(LIBRARY) - n0, **brief(v)})
                check("A1 the new card is first on its shelf; existing tiles are the same elements", v["ids"][0] == "thought:thought_new" and marks == ["", "m0", "m1", "m2"], marks)
                check("A1 'as of' follows the newest authorized read", "as of" in v["summary"] and "3:01:00 PM" in v["summary"] and "3:00:00 PM" in v0["summary"], (v0["summary"], v["summary"]))
                pg.fill("#search", "thought_2"); pg.wait_for_timeout(100)
                for _ in range(4):
                    run(pg, 60500)
                v = probe(pg)
                check("A1 search text kept across automatic reads; one matching tile", v["search"] == "thought_2" and len(v["tiles"]) == 1 and srv.n(LIBRARY) == n0 + 5, brief(v))
                pg.fill("#search", ""); pg.wait_for_timeout(100)
                v = probe(pg)
                check("A1 no duplicate tiles after 5 automatic reads", len(v["ids"]) == 4 and len(set(v["ids"])) == 4, v["ids"])
                done(srv, ctx, pg, "A1")

            attempt("A1", case_a1)

            def case_a2():
                srv, ctx, pg = case(); ready(pg)
                srv.note_n = 6
                open_thought(pg)
                pg.evaluate("document.querySelector('.sheet').scrollTop=180;document.querySelector('.rail').scrollLeft=40;document.getElementById('copy').focus({preventScroll:true})")
                v0 = probe(pg); d0, n0 = srv.n(DETAIL), srv.n(LIBRARY)
                srv.cards["A"][1] = card("A", "thought_2", notes=9, at="2026-11-11T15:02:00Z")   # ANOTHER card changes
                for _ in range(3):
                    run(pg, 60500)
                v = probe(pg)
                keep = ("open", "hash", "hlen", "sheetTop", "focus", "rail", "selected", "detail")
                check("A2 open Thought survives 3 automatic reads: view, URL, history, sheet scroll, focus, rail scroll", srv.n(LIBRARY) == n0 + 3 and v0["sheetTop"] > 0
                      and all(v[k] == v0[k] for k in keep) and v["open"] and v["hash"] == "#thought=thought_1" and v["focus"] == "copy",
                      {"reads": srv.n(LIBRARY) - n0, "before": {k: v0[k] for k in keep if k != "detail"}, "after": {k: v[k] for k in keep if k != "detail"}})
                check("A2 zero extra detail reads; no 'Show latest' for an unchanged record; the other card did update", srv.n(DETAIL) == d0 and not v["fresh"]
                      and any("thought_2. Active. 9 saved Notes" in t for t in v["tiles"]), {"detail_reads": srv.n(DETAIL) - d0, "fresh": v["fresh"]})
                done(srv, ctx, pg, "A2")

            attempt("A2", case_a2)

            def case_a3():
                srv, ctx, pg = case(); ready(pg); open_thought(pg)
                d0 = srv.n(DETAIL)
                srv.note_n = 2
                srv.cards["A"][0] = card("A", "thought_1", notes=2, at="2026-11-11T15:00:40Z")
                run(pg, 61000)
                v = probe(pg)
                check("A3 the open record changed: 'Show latest' offered, the Note on screen not replaced, no detail read", v["open"] and v["fresh"] and "Show latest" in v["freshText"]
                      and NOTE + "-A-thought_1-2" not in v["detail"] and NOTE + "-A-thought_1-1" in v["detail"] and srv.n(DETAIL) == d0, brief(v))
                pg.get_by_role("button", name="Show latest").click(); pg.wait_for_selector(f"text={NOTE}-A-thought_1-2")
                v = probe(pg)
                check("A3 'Show latest': one authorized detail read, the line goes away", srv.n(DETAIL) == d0 + 1 and not v["fresh"] and v["hash"] == "#thought=thought_1", brief(v))
                pg.get_by_role("button", name="Close details").click(); pg.wait_for_timeout(100)
                v = probe(pg)
                check("A3 the closed sheet holds no 'Show latest' line and no Note", not v["fresh"] and NOTE not in v["doc"], brief(v))
                done(srv, ctx, pg, "A3")

            attempt("A3", case_a3)

            def case_a4():
                srv, ctx, pg = case(); ready(pg)
                srv.cards["A"] = [c for c in srv.cards["A"] if c["source_id"] != "thought_2"]
                run(pg, 61000)
                v = probe(pg)
                check("A4 a card missing from ONE complete authorized read has no tile", v["ids"] == ["thought:thought_1", "cycle:tp_1"], v["ids"])
                open_thought(pg); h0 = probe(pg)["hlen"]
                srv.cards["A"] = [c for c in srv.cards["A"] if c["source_id"] != "thought_1"]
                run(pg, 61000)
                v = probe(pg)
                check("A4 the OPEN card went missing: sheet closed and purged, URL replaced, visible notice, no Note left", not v["open"] and v["hash"] == "" and v["hlen"] == h0
                      and NOTE not in v["doc"] and v["selected"] is None and v["title"] == "" and v["noticeVisible"] and "not in this identity's authorized library" in v["notice"]
                      and v["ids"] == ["cycle:tp_1"], brief(v))
                done(srv, ctx, pg, "A4")

            attempt("A4", case_a4)

            def case_a5():
                srv, ctx, pg = case(); ready(pg); open_thought(pg)
                srv.status[LIBRARY] = 503
                n0 = srv.n(LIBRARY)
                run(pg, 61000)
                v = probe(pg)
                check("A5 failed automatic read FAILS CLOSED: no cards, no sheet, no Note text anywhere", closed_clean(v) and v["mode"] == "EVIDENCE UNAVAILABLE"
                      and "not available" in v["banner"] and v["hash"] == "#thought=thought_1", brief(v))
                check("A5 the banner names the next automatic attempt; the strip shows no activity", "Trying again automatically at 3:02:0" in v["banner"] and "attempt 1 of 5" in v["banner"]
                      and "NOT SHOWN" in v["pulse"], brief(v))
                del srv.status[LIBRARY]
                run(pg, 61000, settle=600)
                v = probe(pg)
                check("A5 recovery: one retry after 60 s, the URL's view is read again and shown", srv.n(LIBRARY) == n0 + 2 and v["open"] and v["hash"] == "#thought=thought_1"
                      and NOTE + "-A-thought_1-1" in v["detail"] and len(v["tiles"]) == 3 and "Trying again" not in v["banner"], {"reads": srv.n(LIBRARY) - n0, **brief(v)})
                done(srv, ctx, pg, "A5")

            attempt("A5", case_a5)

            def case_a6():
                srv, ctx, pg = case(); ready(pg)
                srv.status[LIBRARY] = 503
                n0 = srv.n(LIBRARY); seen = []
                for minute in range(1, 46):
                    run(pg, 60000, settle=60)
                    seen.append(srv.n(LIBRARY) - n0)
                at = [i + 1 for i in range(len(seen)) if seen[i] != (seen[i - 1] if i else 0)]
                v = probe(pg)
                check("A6 persistent failure: first failure, then 5 retries at +60/120/240/300/300 s, then none", at == [1, 2, 4, 8, 13, 18] and seen[-1] == 6, {"minutes": at, "total": seen[-1]})
                check("A6 the banner says automatic retries stopped; nothing is shown", closed_clean(v) and "Automatic retries stopped after 5 failed attempts" in v["banner"], brief(v))
                del srv.status[LIBRARY]
                pg.evaluate("document.getElementById('refresh').click()"); pg.wait_for_selector(".tile"); pg.wait_for_timeout(200)
                n1 = srv.n(LIBRARY); run(pg, 61000)
                check("A6 Refresh recovers and automatic refresh starts again", len(probe(pg)["tiles"]) == 3 and srv.n(LIBRARY) == n1 + 1, srv.n(LIBRARY) - n1)
                done(srv, ctx, pg, "A6")

            attempt("A6", case_a6)

            def case_a7():
                srv, ctx, pg = case(); ready(pg); open_thought(pg)
                srv.status[LIBRARY] = 403
                n0 = srv.n(LIBRARY)
                run(pg, 61000)
                v = probe(pg)
                check("A7 refused automatic read fails closed", closed_clean(v) and "Identity denied" in v["banner"] and v["mode"] == "EVIDENCE UNAVAILABLE", brief(v))
                minutes(pg, 30, step=60000)
                v = probe(pg)
                check("A7 a refusal is not retried automatically (30 min: zero library reads, zero pulse reads)", srv.n(LIBRARY) == n0 + 1 and "Automatic refresh is off" in v["banner"]
                      and closed_clean(v), {"reads": srv.n(LIBRARY) - n0, **brief(v)})
                p0 = srv.n(PULSE)
                minutes(pg, 1)
                check("A7 no pulse read without an authorized context", srv.n(PULSE) == p0, srv.n(PULSE) - p0)
                done(srv, ctx, pg, "A7")

            attempt("A7", case_a7)

            def case_a8():
                for what in ("identity", "engine", "link"):
                    srv, ctx, pg = case(); ready(pg); open_thought(pg)
                    if what == "identity":
                        srv.owner = "B"
                    elif what == "engine":
                        srv.scope["A"] = dict(OWNERS["A"], engine="eng_9"); srv.cards["A"] = [card("A", "thought_7")]
                    else:
                        srv.scope["A"] = dict(OWNERS["A"], link="dec_a0000002"); srv.cards["A"] = [card("A", "thought_7")]
                    srv.status[PULSE] = 503            # only the automatic LIBRARY read may discover the change
                    srv.status[DETAIL] = 503
                    run(pg, 61000, settle=500)
                    v = probe(pg)
                    gone = NOTE + "-A-thought_1" not in v["doc"] and "A-title thought_1" not in v["doc"] and "A-title thought_2" not in v["doc"]
                    want = ["thought:thought_1"] if what == "identity" else ["thought:thought_7"]
                    check(f"A8 {what} changed, found by the automatic read: nothing of the previous context remains", gone and v["ids"] == want
                          and ("Nova" in v["banner"]) == (what == "identity") and "job_" not in v["pulseAll"], brief(v))
                    done(srv, ctx, pg, "A8 " + what)

            attempt("A8", case_a8)

            def case_a9a():
                srv, ctx, pg = case(); ready(pg); open_thought(pg)
                srv.hold.add(LIBRARY); n0 = srv.n(LIBRARY)
                run(pg, 61000)
                check("A9a (setup) the automatic read is in flight", len(srv.held) == 1 and srv.n(LIBRARY) == n0 + 1, len(srv.held))
                v = probe(pg)
                check("A9a while it is in flight the open view is untouched", v["open"] and NOTE + "-A-thought_1-1" in v["detail"], brief(v))
                srv.status[LIBRARY] = 403; srv.hold.clear(); srv.release(pg)
                v = probe(pg)
                check("A9a revoked during the in-flight automatic read: protected content removed", closed_clean(v) and "Identity denied" in v["banner"], brief(v))
                done(srv, ctx, pg, "A9a")

            attempt("A9a", case_a9a)

            def case_a9b():
                srv, ctx, pg = case(); ready(pg)
                srv.hold.add(LIBRARY); n0 = srv.n(LIBRARY)
                run(pg, 61000)
                granted = srv.answer("GET", LIBRARY, {})           # what Core said BEFORE the revocation
                srv.status[DETAIL] = 403
                pg.get_by_role("button", name="Inspect A-title thought_1").click(); pg.wait_for_timeout(400)
                v = probe(pg)
                check("A9b a read is refused while the automatic read is in flight: fails closed", closed_clean(v) and "Identity denied" in v["banner"], brief(v))
                srv.hold.clear(); srv.release_as(pg, granted)
                v = probe(pg)
                check("A9b the delayed automatic 200 from before the refusal restores nothing", closed_clean(v) and "Identity denied" in v["banner"]
                      and "Automatic refresh is off" in v["banner"], brief(v))
                minutes(pg, 10, step=60000)
                check("A9b and no automatic read follows a refusal", srv.n(LIBRARY) == n0 + 1 and closed_clean(probe(pg)), srv.n(LIBRARY) - n0)
                done(srv, ctx, pg, "A9b")

            attempt("A9b", case_a9b)

            def case_a10():
                srv, ctx, pg = case(); ready(pg)
                srv.hold.add(LIBRARY); n0 = srv.n(LIBRARY)
                run(pg, 61000)
                old = srv.answer("GET", LIBRARY, {})               # the automatic read's (older) answer
                srv.hold.clear()
                srv.cards["A"] = [card("A", "thought_3")]
                pg.evaluate("document.getElementById('refresh').click()"); pg.wait_for_selector("[aria-label^='Inspect A-title thought_3']"); pg.wait_for_timeout(200)
                srv.release_as(pg, old)
                v = probe(pg)
                check("A10 a held automatic read released after a newer Refresh changes nothing", v["ids"] == ["thought:thought_3"], v["ids"])
                srv.hold.add(LIBRARY)
                pg.evaluate("document.getElementById('refresh').click()"); pg.wait_for_timeout(200)
                n1 = srv.n(LIBRARY)
                minutes(pg, 5, step=30000)
                check("A10 no automatic read starts while a library read is pending (5 min)", srv.n(LIBRARY) == n1 and len(srv.held) == 1, {"new": srv.n(LIBRARY) - n1, "held": len(srv.held)})
                srv.hold.clear(); srv.release(pg)
                n2 = srv.n(LIBRARY); run(pg, 61000)
                check("A10 once it answers, the 60 s schedule continues", srv.n(LIBRARY) == n2 + 1 and probe(pg)["ids"] == ["thought:thought_3"], srv.n(LIBRARY) - n2)
                done(srv, ctx, pg, "A10")

            attempt("A10", case_a10)

            def case_a11():
                srv, ctx, pg = case(); ready(pg)
                h0 = probe(pg)["hlen"]
                open_thought(pg)
                run(pg, 61000)
                h1 = probe(pg)["hlen"]
                pg.go_back(); pg.wait_for_timeout(300)
                v = probe(pg)
                check("A11 Back after an automatic read: the shelf, nothing left in the sheet", not v["open"] and v["hash"] == "" and NOTE not in v["doc"] and len(v["tiles"]) == 3, brief(v))
                run(pg, 61000)
                pg.go_forward(); pg.wait_for_selector(f"text={NOTE}-A-thought_1-1")
                v = probe(pg)
                check("A11 Forward after another automatic read: the Thought again; history length unchanged by automatic reads", v["open"] and v["hash"] == "#thought=thought_1"
                      and h1 == h0 + 1 and v["hlen"] == h1, {"h0": h0, "h1": h1, **brief(v)})
                srv.hold.add(LIBRARY); run(pg, 61000)
                pg.go_back(); pg.wait_for_timeout(300); pg.go_forward(); pg.wait_for_timeout(500)
                v = probe(pg)
                check("A11 Back/Forward while an automatic read is in flight still opens the card", len(srv.held) == 1 and v["open"] and v["hash"] == "#thought=thought_1"
                      and NOTE + "-A-thought_1-1" in v["detail"], brief(v))
                srv.hold.clear(); srv.release(pg)
                v = probe(pg)
                check("A11 and the view is still there when that read answers", v["open"] and NOTE + "-A-thought_1-1" in v["detail"] and v["hlen"] == h1, brief(v))
                done(srv, ctx, pg, "A11")

            attempt("A11", case_a11)

            def case_a12():
                srv, ctx, pg = case(); ready(pg)
                pg.evaluate("window.__setHidden(true)"); pg.wait_for_timeout(100)
                n0 = len(srv.log)
                minutes(pg, 10, step=30000)
                check("A12 hidden tab: zero requests in 10 minutes", len(srv.log) == n0, srv.log[n0:])
                l0 = srv.n(LIBRARY)
                pg.evaluate("window.__setHidden(false)"); pg.wait_for_selector(".tile"); pg.wait_for_timeout(200)
                check("A12 returning re-checks authorization with one library read (P0 resume rule)", srv.n(LIBRARY) == l0 + 1 and len(probe(pg)["tiles"]) == 3, srv.n(LIBRARY) - l0)
                l1 = srv.n(LIBRARY); run(pg, 61000)
                check("A12 and the 60 s schedule starts again", srv.n(LIBRARY) == l1 + 1, srv.n(LIBRARY) - l1)
                done(srv, ctx, pg, "A12")

            attempt("A12", case_a12)

            def case_a13():
                srv, ctx, pg = case(); ready(pg); open_thought(pg)
                srv.hold.add(LIBRARY); n0 = srv.n(LIBRARY)
                run(pg, 61000)
                run(pg, 25000)
                v = probe(pg)
                check("A13 a slow automatic read: one request, the view untouched while it is within its limit", srv.n(LIBRARY) == n0 + 1 and v["open"], {"reads": srv.n(LIBRARY) - n0})
                run(pg, 6000, settle=400)
                v = probe(pg)
                check("A13 no answer within 30 s: aborted, fails closed, says why", closed_clean(v) and "no answer within 30 s" in v["banner"] and "Trying again automatically" in v["banner"]
                      and srv.n(LIBRARY) == n0 + 1, brief(v))
                done(srv, ctx, pg, "A13")

            attempt("A13", case_a13)

            def case_a14():
                srv, ctx, pg = case(); ready(pg); open_thought(pg)
                run(pg, 40000)
                d0, n0 = srv.n(DETAIL), srv.n(LIBRARY)
                pg.evaluate("document.getElementById('refresh').click()"); pg.wait_for_timeout(500)
                v = probe(pg)
                check("A14 manual Refresh: one library read, the open view is read again (P0 behaviour)", srv.n(LIBRARY) == n0 + 1 and srv.n(DETAIL) == d0 + 1 and v["open"]
                      and v["hash"] == "#thought=thought_1", {"lib": srv.n(LIBRARY) - n0, "detail": srv.n(DETAIL) - d0})
                run(pg, 30000)
                check("A14 the automatic timer restarts from the manual Refresh (nothing at the old 60 s mark)", srv.n(LIBRARY) == n0 + 1, srv.n(LIBRARY) - n0)
                run(pg, 31000)
                check("A14 and fires 60 s after it", srv.n(LIBRARY) == n0 + 2, srv.n(LIBRARY) - n0)
                done(srv, ctx, pg, "A14")

            attempt("A14", case_a14)

            def case_a15():
                srv, ctx, pg = case(); srv.pulse = pulse("RUNNING", "READY", last=tick(70)); ready(pg)
                n0 = len(srv.log); l0, p0 = srv.n(LIBRARY), srv.n(PULSE)
                per_minute = []
                for _ in range(10):
                    a, b = srv.n(LIBRARY), srv.n(PULSE)
                    minutes(pg, 1)
                    per_minute.append((srv.n(LIBRARY) - a, srv.n(PULSE) - b))
                new = srv.log[n0:]
                check("A15 10 minutes visible: only GETs, only library and pulse", all(m == "GET" and (pth.startswith(LIBRARY) or pth.startswith(PULSE)) for m, pth in new), sorted(set(new)))
                check("A15 at most 1 library and 6 pulse reads per minute; it does keep reading", all(a <= 1 and b <= 6 for a, b in per_minute)
                      and srv.n(LIBRARY) - l0 >= 9 and srv.n(PULSE) - p0 >= 55, per_minute)
                done(srv, ctx, pg, "A15")

            attempt("A15", case_a15)

            def case_a16():
                srv, ctx, pg = case(); ready(pg)
                srv.status[PULSE] = 503
                p0 = srv.n(PULSE); seen = []
                for _ in range(90):
                    run(pg, 10000, settle=40); seen.append(srv.n(PULSE) - p0)
                at = [(i + 1) * 10 for i in range(len(seen)) if seen[i] != (seen[i - 1] if i else 0)]
                v = probe(pg)
                check("A16 pulse failures back off: 10, +20, +40, +80, +160, +300 s", at == [10, 30, 70, 150, 310, 610], at)
                check("A16 the strip stays UNAVAILABLE with the first failure's time; the cards stay (the library is authorized)", "UNAVAILABLE since 3:00:1" in v["activity"]
                      and len(v["tiles"]) == 3, brief(v))
                del srv.status[PULSE]
                for _ in range(31):
                    run(pg, 10000, settle=40)
                v = probe(pg)
                p1 = srv.n(PULSE); run(pg, 10500)
                check("A16 recovery: the strip is current again and the 10 s cadence resumes", v["code"] == "PAUSED" and srv.n(PULSE) == p1 + 1, brief(v))
                done(srv, ctx, pg, "A16")

            attempt("A16", case_a16)

            def case_a17():
                srv, ctx, pg = case(); ready(pg); open_thought(pg)
                run(pg, 61000)
                pg.get_by_role("button", name="Close details").click(); pg.wait_for_timeout(100)
                run(pg, 61000)
                v = probe(pg)
                check("A17 Note text is in no library response, on no tile, and not in the closed sheet", len(srv.bodies) >= 3 and all(NOTE not in b for b in srv.bodies)
                      and NOTE not in v["doc"], {"bodies": len(srv.bodies)})
                done(srv, ctx, pg, "A17")

            attempt("A17", case_a17)

    finally:
        browser.close()
for f in failures:
    print("FAIL:", f)
print(f"{passed}/{passed + len(failures)} Stage 1 slice 2 checks PASS across 1280/960/390 ({HTML_PATH})")
sys.exit(1 if failures else 0)
