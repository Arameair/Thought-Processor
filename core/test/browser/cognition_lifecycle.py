#!/usr/bin/env python3
"""P0 Cognition lifecycle contract (#106, contract v2): deterministic browser acceptance matrix.
NOT the owner's machine, NOT Nemotron. A fixture stands in for Core with two owners, A and B, whose
Engine entities both have a Thought with the SAME ID. Responses are HELD and released in a chosen
order (no sleeps decide an outcome); the 10-second pulse timer is driven by the page clock.

Every case runs in a fresh browser context at 1280x1000, 960x1000 and 390x844. Owner A's private
markers ("OWNER-A-", "PULSE-A-") are searched for in the WHOLE document (text and markup, hidden or
not), in the clipboard and in prepared downloads, not only in what is visible.

The fixture behaves like Core after this contract: every 2xx scoped GET carries X-Cognition-* scope
headers; in "narrow" mode a request whose X-Cognition-Expect-* headers name another scope gets 412
SCOPE_MISMATCH and no data. In "attest-only" mode it ignores the expectation (a server without the
narrowing check), so the page's OWN attestation check is exercised.

Context, library, pulse
  C1   A's pulse held; Refresh accepts B; B's pulse renders; A's pulse released: never shown
  C1n  Identity silently becomes B; the pulse timer fires (narrow): page revalidates into B, nothing of A
  C1a  same, attest-only server: the page rejects B-attested data shown under A, revalidates into B
  C2   two same-context pulses released newest-first / oldest-first: the newest is shown
  C3a  an older pulse 503 after a newer success: no effect
  C3b  an older success after a newer pulse 503: no effect
  C3c  an older pulse 403 after a newer success, same epoch: fails closed (contract v2 §4 step 2)
  C4   timer pulse and Refresh pulse overlap, both release orders: the newest is shown
  C5   library 403 / latest 503 / superseded 403: pulse strip neutral, no PULSE-A anywhere
  C6   current-epoch 403 fails the page closed even when the view moved on: Thought detail (after
       navigating to history), pulse, cycle timeline, export (Copy), nudge history
  C7a  an OLD-epoch 403 released after a changed-owner library: the new context stays
  C7b  a 403 from before a same-scope accepted Refresh: fails closed (deliberate, fail closed)
  C8   three concurrent Refreshes (A 200, 503, B 200), two release orders: only the newest commits
View, private content, timers
  V2   held nudge poll across an owner change: nothing of A appears; the poll does not re-arm
  V3n  Identity silently becomes B, owner opens the same Thought ID (narrow): B's Note under B only
  V3a  same, attest-only server
  V4   Step, Step, View Logs while the timeline read is held: one read; the Logs view wins
  V4x  held timeline of cycle tp_1, owner opens a Thought, release: nothing of the timeline appears
  V7   missing / pending / unavailable route: VISIBLE shelf notice outside the sheet; sheet empty
  V9h  hidden -> visible after Identity changed: sheet and shelf hidden BEFORE revalidation, URL kept
  V9p  back/forward-cache restore (pageshow persisted): same
  V9t  hidden tab: playback timer stops
  V10  manual hash change opens a card with exactly one detail read; reload + Back follow the route
Copy and export
  X1   Copy held, Close, release: no clipboard write, no toast, no fallback text
  X2   clipboard rejects after another card opened: no old export text in the new view
  X3a  Download held, Close, release: no download prepared
  X3b  Download held, Refresh accepts B, release: no download prepared
  X3c  export attested for B while the page shows A (attest-only): no download, page revalidates
  X4   a current Copy and Download work, once, with the export text and file name
  X5   the download's object URL is revoked; nothing visible changes at revocation
Owner actions
  W1   Save completes after Close: one POST, no toast anywhere; the receipt is there on return
  W3   Stop -> Copy with navigation only: both POSTs, in order
  W4   Stop -> Copy when the owner changes in between: Stop only
  W6   a late Unlink failure after the owner changed: no stale feedback; a current failure is
       shown in the VISIBLE shelf notice
  E1   scoped GETs and owner POSTs carry the committed scope as X-Cognition-Expect-*; library none
  E2   a POST 403 (write grant missing) is write feedback only; the page stays
  E3   a POST 412 (owner changed): nothing written; the page revalidates into B
Supporting (structural, not the contract)
  T1   exactly two fetch( call sites in the page
Every case: zero page errors; read-only cases emit zero POSTs.
    python3 cognition_lifecycle.py [path/to/cognition.html]
"""
import json
import sys
from pathlib import Path
from urllib.parse import urlparse
from playwright.sync_api import sync_playwright

HTML_PATH = Path(sys.argv[1]) if len(sys.argv) > 1 else Path(__file__).resolve().parents[2] / "cmd/tass-core/cognition.html"
HTML = HTML_PATH.read_text()
ORIGIN = "http://127.0.0.1:8795"
NOW = "2026-11-10T15:00:00Z"
OWNERS = {
    "A": {"identity": "ent_aaaaaaaaaaaa", "engine": "eng_1", "link": "dec_a0000001", "name": "Mira"},
    "B": {"identity": "ent_bbbbbbbbbbbb", "engine": "eng_2", "link": "dec_b0000001", "name": "Nova"},
}
A_MARKS = ["OWNER-A-", "PULSE-A-"]
LIBRARY, PULSE, DETAIL, NUDGES = "/api/cognition/library", "/api/cognition/pulse", "/api/cognition/thoughts/", "/api/nudges"
TIMELINE, EXPORT = "/api/cognition/cycles/tp_1/timeline", "/api/cognition/cycles/tp_1/export"


def mark(o, what):
    return f"OWNER-{o}-{what}"


def card(o, sid, kind="thought"):
    return {"id": kind + ":" + sid, "kind": kind, "title": mark(o, "TITLE-" + sid), "summary": mark(o, "SUMMARY-" + sid),
            "source": "Thought Engine", "source_id": sid, "status": "active", "verification": "OBSERVED", "recorded_at": NOW,
            "related_cycle_ids": [], "fields": [{"name": "Private field", "value": mark(o, "FIELD")}]}


def library(o, thoughts=None, cycles=None, link=None):
    ow = OWNERS[o]
    thoughts = thoughts if thoughts is not None else (["thought_1", "thought_2"] if o == "A" else ["thought_1"])
    cycles = cycles if cycles is not None else (["tp_1"] if o == "A" else [])
    lk = link or ow["link"]
    return {"identity_entity_id": ow["identity"], "engine_entity_id": ow["engine"], "identity_name": ow["name"], "engine_commit": "c" * 40,
            "brain_revision": 7, "note": "", "thoughts": [card(o, s) for s in thoughts], "cycles": [card(o, c, "cycle") for c in cycles],
            "outcomes": [], "library": [], "junctions": [],
            "link": {"state": "LINKED", "candidates": [], "unscoped_cycles": 0, "other_entity_cycles": 0, "explanation": "Linked by the owner.",
                     "link": {"identity_entity_id": ow["identity"], "engine_entity_id": ow["engine"], "engine_display_name": ow["name"],
                              "linked_at": NOW, "identity_decision_id": lk, "by": "owner"}}}


def scope_headers(lib):
    return {"X-Cognition-Identity": lib["identity_entity_id"], "X-Cognition-Engine": lib["engine_entity_id"],
            "X-Cognition-Link": lib["link"]["link"]["identity_decision_id"]}


def thought(o, sid):
    return {"thought_id": sid, "question": mark(o, "QUESTION"), "state": "active", "history_total": 1,
            "history": [{"at": NOW, "kind": "developed", "worker_request_id": "mr_" + o}],
            "developments": [{"request_id": "mr_" + o, "at": NOW, "persistence": "NOTE_AND_STEP_CORRELATED",
                              "note": {"at": NOW, "text": mark(o, "NOTE-" + sid), "by": "engine", "suggested_state": "keep"},
                              "step": {"at": NOW, "kind": "developed"}, "meaning": "Saved."}],
            "developments_total": 1, "traces": [], "related_cycle_ids": []}


def nudges(o, running=False, saved=None):
    run = {"state": "SUBMITTED", "model_http": "none", "can_stop": True} if running else None
    ns = [{"nudge_id": "nudge_%020x" % 1, "status": "DRAFT", "recipient_id": "self", "recipient_kind": "local_entity",
           "reason_code": "owner_request", "requested_action": mark(o, "NUDGE"), "priority": "normal", "created_at": NOW,
           "expires_at": NOW, "source_thought_id": "thought_1", "body_digest": "d",
           "processing": {"state": "SEE_RUN" if running else "WAITING", "reason": "x"}, "run": run}]
    for sid in saved or []:
        ns.append(dict(ns[0], nudge_id=sid, run=None, requested_action=mark(o, "SAVED")))
    return {"owner_entity_id": OWNERS[o]["identity"], "nudges": ns,
            "processing": {"state": "NOT_READY", "reason": "Not ready."},
            "readiness": {"identity": "UNLOCKED", "process_grant": "NOT_ENABLED", "engine": "REACHABLE"}, "note": "local"}


def timeline(o, steps=2):
    return {"source": "engine journal", "engine_commit": "c" * 40, "identity_evidence": "x", "link_status": "UNLINKED", "missing": [],
            "truncated": False, "partial": False, "steps": [{"at": NOW, "stage": mark(o, "STAGE-%d" % i), "offset": i, "status": "ok"} for i in range(steps)]}


def pulse(o, n):
    return {"control": "RUNNING", "every_s": 20, "state": "READY", "last": {"pulse_id": f"PULSE-{o}-{n}", "at": NOW, "trigger": "timer", "state": "READY"},
            "queued": 0, "running": 0, "core_tick_every_s": 20, "junctions": {}, "recent": [], "note": ""}


class Server:
    def __init__(self):
        self.reset()

    def reset(self):
        self.owner, self.locked, self.mode = "A", False, "narrow"
        self.lib = {}               # owner -> library override
        self.hold, self.held = set(), []
        self.deny, self.fail = set(), set()
        self.posts, self.gets = [], []
        self.pulse_n = 0
        self.running = False
        self.timeline_steps = 2
        self.saved = []
        self.post_status = {}       # path prefix -> (status, body)

    def library_of(self, o):
        return self.lib.get(o) or library(o)

    def answer(self, method, path, query, hdr):
        o = self.owner
        lib = self.library_of(o)
        scope = scope_headers(lib)
        expect = (hdr.get("x-cognition-expect-identity"), hdr.get("x-cognition-expect-engine"), hdr.get("x-cognition-expect-link"))
        mismatch = self.mode == "narrow" and expect[0] is not None and expect != tuple(scope.values())
        if method == "POST":
            if mismatch:
                return 412, {"error": "scope changed", "code": "SCOPE_MISMATCH"}, {}
            for prefix, (st, body) in self.post_status.items():
                if path.startswith(prefix):
                    return st, body, {}
            if path.endswith("/stop"):
                return 200, {"run": {"state": "NOT_DELIVERED"}}, {}
            if path == "/api/nudges":
                nid = "nudge_%020x" % (50 + len(self.posts))
                self.saved.append(nid)
                return 201, {"nudge_id": nid, "status": "DRAFT"}, {}
            return 200, {"ok": True}, {}
        if any(path.startswith(d) for d in self.deny) or self.locked:
            return 403, {"error": "Cognition read denied"}, {}
        if any(path.startswith(d) for d in self.fail):
            return 503, {"error": "unavailable"}, {}
        if path.startswith(LIBRARY):
            return 200, lib, scope
        if mismatch:
            return 412, {"error": "scope changed", "code": "SCOPE_MISMATCH"}, {}
        if path.startswith(PULSE):
            self.pulse_n += 1
            return 200, pulse(o, self.pulse_n), scope
        if path.startswith(DETAIL):
            sid = path[len(DETAIL):]
            if any(c["source_id"] == sid for c in lib["thoughts"]):
                return 200, thought(o, sid), scope
            return 503, {"error": "NOT RECORDED"}, {}
        if path.startswith(NUDGES):
            return 200, nudges(o, self.running, self.saved), scope
        if path.startswith(TIMELINE):
            return 200, timeline(o, self.timeline_steps), scope
        if path.startswith(EXPORT):
            return 200, "# Cognition sanitized\n" + mark(o, "EXPORT") + "\n", {**scope, "Content-Type": "text/markdown"}
        return 404, {}, {}

    def route(self, req):
        r = req.request
        u = urlparse(r.url)
        path = u.path
        if path == "/cognition" and r.method == "GET":
            req.fulfill(status=200, content_type="text/html", body=HTML.replace("__PAGE_TOKEN__", "t")); return
        if not path.startswith("/api/"):
            req.fulfill(status=404, body=""); return
        hdr = {k.lower(): v for k, v in r.headers.items()}
        if r.method == "POST":
            self.posts.append((path, {k: v for k, v in hdr.items() if k.startswith("x-cognition-expect")}))
        else:
            self.gets.append((path, {k: v for k, v in hdr.items() if k.startswith("x-cognition-expect")}))
        status, body, headers = self.answer(r.method, path, u.query, hdr)
        text = body if isinstance(body, str) else json.dumps(body)
        ctype = headers.pop("Content-Type", "application/json")
        key = ("POST " if r.method == "POST" else "") + path
        if any(key.startswith(h) for h in self.hold):
            self.held.append((req, status, text, ctype, headers, key)); return
        req.fulfill(status=status, content_type=ctype, body=text, headers=headers)

    def release_one(self, pg, index=0):
        req, status, text, ctype, headers, _ = self.held.pop(index)
        req.fulfill(status=status, content_type=ctype, body=text, headers=headers)
        pg.wait_for_timeout(450)

    def release_all(self, pg, order="fifo"):
        while self.held:
            self.release_one(pg, 0 if order == "fifo" else len(self.held) - 1)

    def wait_held(self, pg, n):
        for _ in range(120):
            if len(self.held) >= n:
                return
            pg.wait_for_timeout(50)
        raise AssertionError(f"expected {n} held request(s), have {len(self.held)}: {[h[5] for h in self.held]}")

    def count(self, prefix):
        return len([g for g in self.gets if g[0].startswith(prefix)])


SRV = Server()

INSTRUMENT = """
window.__clip=[];window.__clipMode='ok';window.__clipPending=null;
Object.defineProperty(navigator,'clipboard',{configurable:true,value:{writeText:(t)=>{window.__clip.push(t);
  if(window.__clipMode==='reject')return Promise.reject(new Error('denied'));
  if(window.__clipMode==='hold')return new Promise((ok,no)=>{window.__clipPending={ok,no};});
  return Promise.resolve();}}});
window.__objects=[];window.__revoked=[];window.__downloads=[];
const __oc=URL.createObjectURL.bind(URL),__or=URL.revokeObjectURL.bind(URL);
URL.createObjectURL=(b)=>{const u=__oc(b);window.__objects.push(u);return u;};
URL.revokeObjectURL=(u)=>{window.__revoked.push(u);__or(u);};
const __click=HTMLAnchorElement.prototype.click;
HTMLAnchorElement.prototype.click=function(){if(this.download){window.__downloads.push(this.download);return;}return __click.call(this);};
window.__setHidden=(h)=>{Object.defineProperty(document,'hidden',{configurable:true,get:()=>h});
  Object.defineProperty(document,'visibilityState',{configurable:true,get:()=>h?'hidden':'visible'});
  document.dispatchEvent(new Event('visibilitychange'));};
"""

PROBE = """(marks)=>{
  const id=x=>document.getElementById(x), ov=id('overlay'), open=!!ov&&ov.classList.contains('open');
  const notice=id('notice');
  const vis=e=>{if(!e)return false;const r=e.getBoundingClientRect(),s=getComputedStyle(e);
    return s.display!=='none'&&s.visibility!=='hidden'&&r.width>0&&r.height>0&&r.bottom>0&&r.top<innerHeight;};
  const all=document.documentElement.outerHTML+' '+document.body.textContent+' '+location.href+' '+
    (window.__clip||[]).join(' ');
  const a=document.activeElement;
  return {open,hash:location.hash,href:location.href,tiles:document.querySelectorAll('.tile').length,
    leaks:marks.filter(m=>all.includes(m)),
    sheet:open?document.querySelector('.sheet').innerText:'',
    detail:id('thoughtDetail').textContent,toast:id('toast').textContent,
    notice:notice?notice.textContent:'',noticeVisible:vis(notice),
    banner:id('banner').textContent,mode:id('mode').textContent,pulse:id('pulse').textContent,
    selected:typeof S==='object'&&S.selected?S.selected.source_id:null,timeline:typeof S==='object'&&S.timeline?'set':null,
    cursor:typeof S==='object'?S.cursor:null,focusText:a?(a.textContent||'').slice(0,40):'',hlen:history.length,
    clip:(window.__clip||[]).length,downloads:(window.__downloads||[]).slice(),objects:(window.__objects||[]).length,
    revoked:(window.__revoked||[]).length,textarea:!!document.querySelector('#thoughtDetail textarea')}}"""


def probe(pg, marks=A_MARKS):
    return pg.evaluate(PROBE, marks)


def brief(v):
    keep = ("open", "hash", "tiles", "leaks", "toast", "notice", "noticeVisible", "banner", "mode", "pulse", "selected", "timeline",
            "cursor", "clip", "downloads", "textarea")
    return {k: (v[k][:90] if isinstance(v[k], str) else v[k]) for k in keep}


def failed_closed(v, hash_kept=None):
    """The page shows nothing of the old context: no cards, no sheet, no marker anywhere, banner says so."""
    ok = not v["open"] and v["tiles"] == 0 and not v["leaks"] and v["selected"] is None and v["timeline"] is None
    ok = ok and v["mode"] in ("EVIDENCE UNAVAILABLE", "CHECKING")
    if hash_kept is not None:
        ok = ok and v["hash"] == hash_kept
    return ok


with sync_playwright() as p:
    browser = p.chromium.launch(headless=True, args=["--no-sandbox"])
    failures, passed, ran = [], 0, []
    OPEN = []
    try:
        for (w, h, tag) in [(1280, 1000, "desktop"), (960, 1000, "half"), (390, 844, "phone")]:

            def check(name, cond, detail=""):
                global passed
                if cond:
                    passed += 1
                else:
                    failures.append(f"[{tag}] {name}: {detail}")
                if tag == "desktop":
                    ran.append(name)

            def case(clock=False, hold=(), mode="narrow"):
                SRV.reset()
                SRV.mode = mode
                ctx = browser.new_context(viewport={"width": w, "height": h})
                OPEN.append(ctx)
                pg = ctx.new_page()
                pg.set_default_timeout(6000)
                pg.__errs = []
                pg.on("pageerror", lambda e: pg.__errs.append(str(e)))
                pg.add_init_script(INSTRUMENT)
                if clock:
                    pg.clock.install()
                pg.route("**/*", SRV.route)
                SRV.hold.update(hold)
                pg.goto(ORIGIN + "/cognition")
                return ctx, pg

            def ready(pg):
                pg.wait_for_selector(".tile")
                pg.wait_for_timeout(250)

            def done(ctx, pg, name, read_only=True):
                check(name + " / no page errors", not pg.__errs, pg.__errs)
                if read_only:
                    check(name + " / zero POSTs", not SRV.posts, SRV.posts)
                ctx.close(); OPEN.remove(ctx)

            def click_refresh(pg):
                pg.evaluate("document.getElementById('refresh').click()")
                pg.wait_for_timeout(300)

            def inspect(pg, o, sid):
                pg.get_by_role("button", name="Inspect " + mark(o, "TITLE-" + sid)).click()

            def open_note(pg, o="A", sid="thought_1"):
                inspect(pg, o, sid)
                pg.wait_for_selector("text=" + mark(o, "NOTE-" + sid))

            def settle(pg, sel, timeout=3000):
                try:
                    pg.wait_for_selector(sel, timeout=timeout)
                except Exception:
                    pass

            def sec_0():
                """C1: A's held pulse after B's context"""
                ctx, pg = case(hold={PULSE}); ready(pg); SRV.wait_held(pg, 1)
                SRV.hold.clear(); SRV.owner = "B"; click_refresh(pg); settle(pg, "text=PULSE-B-")
                SRV.release_all(pg)
                v = probe(pg)
                check("C1 held A pulse after B's library and pulse: B's pulse stays, nothing of A",
                      "PULSE-B-" in v["pulse"] and not v["leaks"] and v["tiles"] == 1, brief(v))
                done(ctx, pg, "C1")

            def sec_1():
                """C1n / C1a: silent Identity switch, pulse timer"""
                for mode, tagc in (("narrow", "C1n"), ("attest-only", "C1a")):
                    ctx, pg = case(clock=True, mode=mode); ready(pg)
                    SRV.owner = "B"
                    pg.clock.run_for(10500); pg.wait_for_timeout(900)
                    settle(pg, "text=" + mark("B", "TITLE-thought_1"))
                    v = probe(pg)
                    check(f"{tagc} Identity switched to B, pulse timer ({mode}): page revalidated into B, nothing of A",
                          not v["leaks"] and v["tiles"] == 1 and "Nova" in v["banner"], brief(v))
                    done(ctx, pg, tagc)

            def sec_2():
                """C2: two same-context pulses"""
                for order in ("lifo", "fifo"):
                    ctx, pg = case(); ready(pg)
                    SRV.hold.add(PULSE)
                    click_refresh(pg); SRV.wait_held(pg, 1); click_refresh(pg); SRV.wait_held(pg, 2)
                    SRV.hold.clear(); SRV.release_all(pg, order)
                    v = probe(pg, [])
                    check(f"C2 two pulses released {order}: the newest (PULSE-A-3) is shown", "PULSE-A-3" in v["pulse"], brief(v))
                    done(ctx, pg, "C2 " + order)

            def sec_3():
                """C3: stale failure / success / denial"""
                ctx, pg = case(); ready(pg)
                SRV.hold.add(PULSE); SRV.fail.add(PULSE); click_refresh(pg); SRV.wait_held(pg, 1)
                SRV.fail.clear(); click_refresh(pg); SRV.wait_held(pg, 2)
                SRV.hold.clear(); SRV.release_one(pg, 1); SRV.release_one(pg, 0)
                v = probe(pg, [])
                check("C3a older pulse 503 after a newer success: the newer pulse stays", "PULSE-A-" in v["pulse"] and "UNAVAILABLE" not in v["pulse"], brief(v))
                done(ctx, pg, "C3a")

                ctx, pg = case(); ready(pg)
                SRV.hold.add(PULSE); click_refresh(pg); SRV.wait_held(pg, 1)
                SRV.fail.add(PULSE); click_refresh(pg); SRV.wait_held(pg, 2)
                SRV.hold.clear(); SRV.release_one(pg, 1); SRV.release_one(pg, 0)
                v = probe(pg, [])
                check("C3b older pulse success after a newer 503: the strip stays unavailable", "UNAVAILABLE" in v["pulse"] and "PULSE-A-" not in v["pulse"], brief(v))
                done(ctx, pg, "C3b")

                ctx, pg = case(); ready(pg)
                SRV.hold.add(PULSE); SRV.deny.add(PULSE); click_refresh(pg); SRV.wait_held(pg, 1)
                SRV.deny.clear(); click_refresh(pg); SRV.wait_held(pg, 2)
                SRV.hold.clear(); SRV.release_one(pg, 1); SRV.release_one(pg, 0)
                v = probe(pg)
                check("C3c older pulse 403 after a newer success, same epoch: the page fails closed (v2 §4 step 2)",
                      failed_closed(v) and "Identity denied" in v["banner"], brief(v))
                done(ctx, pg, "C3c")

            def sec_4():
                """C4: timer pulse and Refresh pulse overlap"""
                for order in ("lifo", "fifo"):
                    ctx, pg = case(clock=True); ready(pg)
                    SRV.hold.add(PULSE)
                    pg.clock.run_for(10500); SRV.wait_held(pg, 1)
                    click_refresh(pg); SRV.wait_held(pg, 2)
                    SRV.hold.clear(); SRV.release_all(pg, order)
                    v = probe(pg, [])
                    check(f"C4 timer pulse and Refresh pulse released {order}: the newest (PULSE-A-3) is shown", "PULSE-A-3" in v["pulse"], brief(v))
                    done(ctx, pg, "C4 " + order)

            def sec_5():
                """C5: library failures neutralize the pulse"""
                for name, setup in (("403", lambda: SRV.deny.add(LIBRARY)), ("503", lambda: SRV.fail.add(LIBRARY))):
                    ctx, pg = case(); ready(pg); pg.wait_for_selector("text=PULSE-A-1")
                    setup(); click_refresh(pg); pg.wait_for_timeout(300)
                    v = probe(pg)
                    check(f"C5 library {name}: pulse strip neutral, no PULSE-A anywhere", failed_closed(v) and "PULSE-A" not in v["pulse"], brief(v))
                    done(ctx, pg, "C5 " + name)
                ctx, pg = case(); ready(pg); pg.wait_for_selector("text=PULSE-A-1")
                SRV.hold.add(LIBRARY); SRV.deny.add(LIBRARY); click_refresh(pg); SRV.wait_held(pg, 1)
                SRV.deny.clear(); click_refresh(pg); SRV.wait_held(pg, 2)
                SRV.hold.clear(); SRV.release_one(pg, 0)
                v = probe(pg)
                check("C5 superseded library 403 while the newest is pending: pulse neutral, nothing of A", failed_closed(v), brief(v))
                SRV.release_one(pg, 0)
                done(ctx, pg, "C5 superseded")

            def sec_6():
                """C6: current-epoch 403 fails closed"""
                ctx, pg = case(); ready(pg)
                SRV.hold.add(DETAIL); SRV.deny.add(DETAIL); inspect(pg, "A", "thought_1"); SRV.wait_held(pg, 1)
                SRV.hold.clear(); SRV.deny.clear()
                pg.get_by_role("button", name="Nudge history").click(); settle(pg, "text=Nudge history for thought_1")
                SRV.release_all(pg)
                v = probe(pg)
                check("C6 Thought detail 403 released after the owner moved to nudge history: fails closed, URL kept",
                      failed_closed(v, "#thought=thought_1&view=nudges") and "Identity denied" in v["banner"], brief(v))
                done(ctx, pg, "C6 detail")
                for name, act in (
                        ("pulse", lambda pg: (SRV.deny.add(PULSE), click_refresh(pg))),
                        ("timeline", lambda pg: (inspect(pg, "A", "tp_1"), SRV.deny.add(TIMELINE), pg.get_by_role("button", name="View Logs").click())),
                        ("export", lambda pg: (inspect(pg, "A", "tp_1"), SRV.deny.add(EXPORT), pg.get_by_role("button", name="Copy Evidence").click())),
                        ("nudge history", lambda pg: (open_note(pg), SRV.deny.add(NUDGES), pg.get_by_role("button", name="Nudge history").click()))):
                    ctx, pg = case(); ready(pg)
                    act(pg); pg.wait_for_timeout(700)
                    v = probe(pg)
                    check(f"C6 {name} 403 in the current context: the page fails closed", failed_closed(v) and "Identity denied" in v["banner"] and v["clip"] == 0, brief(v))
                    done(ctx, pg, "C6 " + name)

            def sec_7():
                """C7: old-epoch vs same-scope denial"""
                ctx, pg = case(); ready(pg)
                SRV.hold.add(DETAIL); SRV.deny.add(DETAIL); inspect(pg, "A", "thought_1"); SRV.wait_held(pg, 1)
                SRV.hold.clear(); SRV.deny.clear(); SRV.owner = "B"; click_refresh(pg); settle(pg, "text=" + mark("B", "NOTE-thought_1"))
                SRV.release_all(pg)
                v = probe(pg)
                check("C7a an old-epoch 403 released after a changed-owner library: B's context and Note stay",
                      v["open"] and mark("B", "NOTE-thought_1") in v["sheet"] and v["tiles"] == 1 and not v["leaks"], brief(v))
                done(ctx, pg, "C7a")
                ctx, pg = case(); ready(pg)
                SRV.hold.add(DETAIL); SRV.deny.add(DETAIL); inspect(pg, "A", "thought_1"); SRV.wait_held(pg, 1)
                SRV.hold.clear(); SRV.deny.clear(); click_refresh(pg); settle(pg, "text=" + mark("A", "NOTE-thought_1"))
                SRV.release_all(pg)
                v = probe(pg)
                check("C7b a 403 from before a same-scope accepted Refresh: fails closed (deliberate)",
                      failed_closed(v, "#thought=thought_1") and "Identity denied" in v["banner"], brief(v))
                done(ctx, pg, "C7b")

            def sec_8():
                """C8: three concurrent Refreshes"""
                for order in ("newest-first", "oldest-first"):
                    ctx, pg = case(); ready(pg)
                    SRV.hold.add(LIBRARY)
                    click_refresh(pg); SRV.wait_held(pg, 1)
                    SRV.fail.add(LIBRARY); click_refresh(pg); SRV.wait_held(pg, 2)
                    SRV.fail.clear(); SRV.owner = "B"; click_refresh(pg); SRV.wait_held(pg, 3)
                    SRV.hold.clear(); SRV.release_all(pg, "lifo" if order == "newest-first" else "fifo")
                    v = probe(pg)
                    check(f"C8 three Refreshes (A 200, 503, B 200) released {order}: only B commits", v["tiles"] == 1 and "Nova" in v["banner"] and not v["leaks"], brief(v))
                    done(ctx, pg, "C8 " + order)

            def sec_9():
                """V2: held poll across an owner change"""
                ctx, pg = case(); ready(pg)
                SRV.running = True; open_note(pg)
                pg.get_by_role("button", name="Nudge history").click(); pg.wait_for_selector("text=Nudge history for thought_1")
                SRV.hold.add(NUDGES); pg.wait_for_timeout(3300); SRV.wait_held(pg, 1)
                SRV.hold.clear(); SRV.running = False; SRV.owner = "B"; click_refresh(pg); settle(pg, "text=" + mark("B", "NUDGE"))
                SRV.release_all(pg)
                n0 = SRV.count(NUDGES); pg.wait_for_timeout(3300)
                v = probe(pg)
                check("V2 held nudge poll after an owner change: nothing of A, no further polls",
                      not v["leaks"] and SRV.count(NUDGES) == n0 and "Nova" in v["banner"], {"polls_after": SRV.count(NUDGES) - n0, **brief(v)})
                done(ctx, pg, "V2")

            def sec_10():
                """V3: silent switch, same Thought ID"""
                for mode, tagc in (("narrow", "V3n"), ("attest-only", "V3a")):
                    ctx, pg = case(mode=mode); ready(pg)
                    SRV.owner = "B"
                    inspect(pg, "A", "thought_1"); pg.wait_for_timeout(1200); settle(pg, "text=" + mark("B", "NOTE-thought_1"))
                    v = probe(pg)
                    check(f"{tagc} same Thought ID after a silent switch ({mode}): B's Note only under B's context",
                          not v["leaks"] and "Nova" in v["banner"] and (not v["open"] or mark("B", "NOTE-thought_1") in v["sheet"]), brief(v))
                    done(ctx, pg, tagc)

            def sec_11():
                """V4: timeline overlap"""
                ctx, pg = case(); ready(pg); SRV.timeline_steps = 4
                inspect(pg, "A", "tp_1"); pg.wait_for_timeout(200)
                SRV.hold.add(TIMELINE)
                pg.get_by_role("button", name="Step", exact=True).click(); pg.wait_for_timeout(150)
                pg.get_by_role("button", name="Step", exact=True).click(); pg.wait_for_timeout(150)
                pg.get_by_role("button", name="View Logs").click(); pg.wait_for_timeout(300)
                reads = SRV.count(TIMELINE)
                SRV.hold.clear(); SRV.release_all(pg, "lifo")
                v = probe(pg, [])
                check("V4 Step, Step, View Logs while held: one timeline read; the Logs view wins with a consistent cursor",
                      reads == 1 and "View logs" in v["sheet"] and v["cursor"] == 3, {"timeline_reads": reads, **brief(v)})
                done(ctx, pg, "V4")
                ctx, pg = case(); ready(pg)
                inspect(pg, "A", "tp_1"); SRV.hold.add(TIMELINE)
                pg.get_by_role("button", name="View Logs").click(); SRV.wait_held(pg, 1)
                pg.get_by_role("button", name="Close details").click(); SRV.hold.clear()
                open_note(pg, "A", "thought_2"); SRV.release_all(pg)
                v = probe(pg, [])
                check("V4x held timeline released after the owner opened a Thought: no timeline in the view or cache",
                      v["open"] and "STAGE" not in v["sheet"] and v["timeline"] is None, brief(v))
                done(ctx, pg, "V4x")

            def sec_12():
                """V7: visible shelf notice"""
                ctx, pg = case(); ready(pg)
                pg.evaluate("location.hash='thought=nope_0001'"); pg.wait_for_timeout(500)
                v = probe(pg)
                check("V7 missing record: a VISIBLE shelf notice outside the sheet; sheet empty; URL normalized",
                      v["noticeVisible"] and "not in this identity's authorized library" in v["notice"] and not v["open"] and v["detail"] == ""
                      and v["hash"] == "" and not any(m in v["notice"] + v["banner"] for m in ["OWNER-", "nope_0001"]), brief(v))
                open_note(pg, "A", "thought_2")
                v = probe(pg, [])
                check("V7 the shelf notice clears on the next navigation", v["notice"] == "" and not v["noticeVisible"], brief(v))
                done(ctx, pg, "V7 missing")
                ctx, pg = case(hold={LIBRARY})
                SRV.wait_held(pg, 1); pg.evaluate("location.hash='thought=thought_1'"); pg.wait_for_timeout(400)
                v = probe(pg)
                check("V7 link while the first library read is pending: visible 'waiting' notice, URL kept",
                      v["noticeVisible"] and "Waiting" in v["notice"] and v["hash"] == "#thought=thought_1" and not v["open"], brief(v))
                SRV.hold.clear(); SRV.release_all(pg); settle(pg, "text=" + mark("A", "NOTE-thought_1"))
                v = probe(pg, [])
                check("V7 ... then the link opens and the notice clears", v["open"] and v["notice"] == "", brief(v))
                done(ctx, pg, "V7 pending")
                ctx, pg = case(); ready(pg)
                SRV.fail.add(LIBRARY); click_refresh(pg)
                pg.evaluate("location.hash='thought=thought_1'"); pg.wait_for_timeout(400)
                v = probe(pg)
                check("V7 link while the library is unavailable: visible notice, URL kept, nothing shown",
                      v["noticeVisible"] and "can't be opened" in v["notice"] and failed_closed(v, "#thought=thought_1"), brief(v))
                done(ctx, pg, "V7 unavailable")

            def sec_13():
                """V9: resume revalidation"""
                for how, tagc in (("visible", "V9h"), ("pageshow", "V9p")):
                    ctx, pg = case(); ready(pg); open_note(pg)
                    SRV.owner = "B"; SRV.hold.add(LIBRARY)
                    if how == "visible":
                        pg.evaluate("window.__setHidden(true)"); pg.evaluate("window.__setHidden(false)")
                    else:
                        pg.evaluate("window.dispatchEvent(new PageTransitionEvent('pageshow',{persisted:true}))")
                    pg.wait_for_timeout(300)
                    before = probe(pg)
                    check(f"{tagc} resume after Identity changed: sheet and shelf hidden BEFORE revalidation, URL kept",
                          len(SRV.held) == 1 and failed_closed(before, "#thought=thought_1"), {"held": len(SRV.held), **brief(before)})
                    SRV.hold.clear(); SRV.release_all(pg); settle(pg, "text=" + mark("B", "NOTE-thought_1"))
                    v = probe(pg)
                    check(f"{tagc} ... then B's context; the same link reopens under B only",
                          not v["leaks"] and "Nova" in v["banner"] and v["open"] and mark("B", "NOTE-thought_1") in v["sheet"], brief(v))
                    done(ctx, pg, tagc)
                ctx, pg = case(); ready(pg); SRV.timeline_steps = 12
                inspect(pg, "A", "tp_1"); pg.get_by_role("button", name="Play").click(); pg.wait_for_timeout(900)
                pg.evaluate("window.__setHidden(true)"); c0 = probe(pg, [])["cursor"]; pg.wait_for_timeout(1600)
                c1 = probe(pg, [])["cursor"]
                check("V9t hidden tab: the playback timer stops", c0 == c1, {"before": c0, "after": c1})
                done(ctx, pg, "V9t")

            def sec_14():
                """V10: route convergence"""
                ctx, pg = case(); ready(pg)
                d0 = SRV.count(DETAIL)
                pg.evaluate("location.hash='thought=thought_2'"); pg.wait_for_selector("text=" + mark("A", "NOTE-thought_2"))
                pg.wait_for_timeout(400)
                n = SRV.count(DETAIL) - d0
                pg.reload(); pg.wait_for_selector("text=" + mark("A", "NOTE-thought_2"))
                pg.go_back(); pg.wait_for_timeout(500)
                v = probe(pg, [])
                check("V10 manual hash change: exactly one detail read; reload reopens; Back returns to the shelf",
                      n == 1 and not v["open"] and v["hash"] == "", {"detail_reads": n, **brief(v)})
                done(ctx, pg, "V10")

            def sec_15():
                """X: Copy and export"""
                ctx, pg = case(); ready(pg); inspect(pg, "A", "tp_1")
                SRV.hold.add(EXPORT); pg.get_by_role("button", name="Copy Evidence").click(); SRV.wait_held(pg, 1)
                pg.get_by_role("button", name="Close details").click(); SRV.hold.clear(); SRV.release_all(pg)
                v = probe(pg, [mark("A", "EXPORT")])
                check("X1 Copy held, Close, release: no clipboard write, no toast, no fallback text", v["clip"] == 0 and v["toast"] == "" and not v["leaks"], brief(v))
                done(ctx, pg, "X1")
                ctx, pg = case(); ready(pg); inspect(pg, "A", "tp_1")
                pg.evaluate("window.__clipMode='hold'"); pg.get_by_role("button", name="Copy Evidence").click(); pg.wait_for_timeout(400)
                pg.get_by_role("button", name="Close details").click(); open_note(pg, "A", "thought_2")
                pg.evaluate("window.__clipPending&&window.__clipPending.no(new Error('denied'))"); pg.wait_for_timeout(400)
                v = probe(pg, [mark("A", "EXPORT")])
                check("X2 clipboard rejects after another card opened: no old export text in the new view, no stale toast",
                      v["open"] and not v["textarea"] and mark("A", "EXPORT") not in v["sheet"] and "Clipboard" not in v["toast"], brief(v))
                done(ctx, pg, "X2")
                for name, act, mode in (("Close", lambda pg: pg.get_by_role("button", name="Close details").click(), "narrow"),
                                        ("owner change", lambda pg: (setattr(SRV, "owner", "B"), click_refresh(pg)), "narrow")):
                    ctx, pg = case(mode=mode); ready(pg); inspect(pg, "A", "tp_1")
                    SRV.hold.add(EXPORT); pg.get_by_role("button", name="Download Sanitized Markdown").click(); SRV.wait_held(pg, 1)
                    act(pg); SRV.hold.clear(); SRV.release_all(pg)
                    v = probe(pg)
                    check(f"X3 Download held, {name}, release: no download prepared, no stale toast",
                          v["downloads"] == [] and v["objects"] == 0 and "report prepared" not in v["toast"], brief(v))
                    done(ctx, pg, "X3 " + name)
                ctx, pg = case(mode="attest-only"); ready(pg); inspect(pg, "A", "tp_1")
                SRV.owner = "B"; pg.get_by_role("button", name="Download Sanitized Markdown").click(); pg.wait_for_timeout(1200)
                v = probe(pg, A_MARKS + [mark("B", "EXPORT")])
                check("X3c export attested for B while the page shows A: no download; page revalidates into B",
                      v["downloads"] == [] and v["objects"] == 0 and not v["leaks"] and "Nova" in v["banner"], brief(v))
                done(ctx, pg, "X3c")
                ctx, pg = case(); ready(pg); inspect(pg, "A", "tp_1")
                pg.get_by_role("button", name="Copy Evidence").click(); pg.wait_for_timeout(500)
                clip = pg.evaluate("window.__clip.slice()")
                pg.get_by_role("button", name="Download Sanitized Markdown").click(); pg.wait_for_timeout(500)
                v = probe(pg, [])
                check("X4 current Copy and Download work once each", clip == ["# Cognition sanitized\n" + mark("A", "EXPORT") + "\n"]
                      and v["downloads"] == ["cognition-tp_1.md"] and "report prepared" in v["toast"], {"clip": clip, **brief(v)})
                t0 = v["toast"]; pg.wait_for_timeout(1300)
                v = probe(pg, [])
                check("X5 the object URL is revoked; nothing visible changes", v["revoked"] == v["objects"] == 1 and v["toast"] == t0, brief(v))
                done(ctx, pg, "X4/X5")

            def sec_16():
                """W: owner actions"""
                ctx, pg = case(); ready(pg); open_note(pg)
                pg.get_by_role("button", name="Manual nudge (draft)").click(); pg.wait_for_selector("text=Manual nudge — draft")
                SRV.hold.add("POST /api/nudges")
                pg.get_by_role("button", name="Save draft (local only, no model call)").click(); SRV.wait_held(pg, 1)
                pg.get_by_role("button", name="Close details").click(); SRV.hold.clear(); SRV.release_all(pg)
                v = probe(pg, [])
                posted = [x[0] for x in SRV.posts]
                pg.evaluate("location.hash='thought=thought_1&view=nudges'"); settle(pg, "text=" + mark("A", "SAVED"))
                back = probe(pg, [])
                check("W1 Save completes after Close: one POST, no toast; the receipt is there on return",
                      posted == ["/api/nudges"] and v["toast"] == "" and not v["open"] and mark("A", "SAVED") in back["sheet"], {"posts": posted, **brief(v)})
                done(ctx, pg, "W1", read_only=False)
                for name, between, want in (("navigation only", lambda pg: pg.get_by_role("button", name="Close details").click(),
                                             ["/api/nudges/nudge_00000000000000000001/stop", "/api/nudges"]),
                                            ("owner change", lambda pg: (setattr(SRV, "owner", "B"), click_refresh(pg)),
                                             ["/api/nudges/nudge_00000000000000000001/stop"])):
                    ctx, pg = case(); ready(pg); SRV.running = True; open_note(pg)
                    pg.get_by_role("button", name="Nudge history").click(); pg.wait_for_selector("text=Nudge history for thought_1")
                    SRV.hold.add("POST /api/nudges/")
                    pg.get_by_role("button", name="Stop and copy as new").click(); SRV.wait_held(pg, 1)
                    between(pg); SRV.hold.clear(); SRV.release_all(pg); pg.wait_for_timeout(500)
                    posted = [x[0] for x in SRV.posts]
                    check(f"W{'3' if want[1:] else '4'} Stop -> Copy with {name}: POSTs {len(want)}", posted == want, posted)
                    done(ctx, pg, "W stop-copy " + name, read_only=False)
                ctx, pg = case(); ready(pg)
                SRV.post_status["/api/cognition/unlink"] = (409, {"error": "refused"})
                SRV.hold.add("POST /api/cognition/unlink")
                pg.get_by_role("button", name="Unlink (owner action)").click(); SRV.wait_held(pg, 1)
                SRV.hold.clear(); SRV.owner = "B"; click_refresh(pg); settle(pg, "text=" + mark("B", "TITLE-thought_1"))
                SRV.release_all(pg)
                v = probe(pg)
                check("W6 a late Unlink failure after the owner changed: no stale feedback anywhere",
                      "refused" not in v["banner"] and "refused" not in v["notice"] and "Nova" in v["banner"] and not v["leaks"], brief(v))
                pg.get_by_role("button", name="Unlink (owner action)").click(); pg.wait_for_timeout(500)
                v = probe(pg, [])
                check("W6 a current Unlink failure is shown in the VISIBLE shelf notice", v["noticeVisible"] and "refused" in v["notice"]
                      and "Nova" in v["banner"], brief(v))
                done(ctx, pg, "W6", read_only=False)

            def sec_17():
                """E: expected scope on requests"""
                ctx, pg = case(); ready(pg); open_note(pg)
                pg.get_by_role("button", name="Nudge history").click(); pg.wait_for_selector("text=Nudge history for thought_1")
                pg.get_by_role("button", name="Close details").click(); inspect(pg, "A", "tp_1")
                pg.get_by_role("button", name="View Logs").click(); pg.wait_for_timeout(300)
                pg.get_by_role("button", name="Copy Evidence").click(); pg.wait_for_timeout(300)
                pg.get_by_role("button", name="Close details").click(); open_note(pg)
                pg.get_by_role("button", name="Manual nudge (draft)").click(); pg.wait_for_selector("text=Manual nudge — draft")
                pg.get_by_role("button", name="Save draft (local only, no model call)").click(); pg.wait_for_timeout(500)
                a = OWNERS["A"]; want = {"x-cognition-expect-identity": a["identity"], "x-cognition-expect-engine": a["engine"], "x-cognition-expect-link": a["link"]}
                scoped_gets = [g for g in SRV.gets if not g[0].startswith(LIBRARY)]
                kinds = {k for k in (PULSE, DETAIL, NUDGES, TIMELINE, EXPORT) if any(g[0].startswith(k) for g in scoped_gets)}
                check("E1 every scoped GET and owner POST carries the committed scope; the library carries none",
                      len(kinds) == 5 and all(g[1] == want for g in scoped_gets) and all(p_[1] == want for p_ in SRV.posts) and SRV.posts
                      and all(g[1] == {} for g in SRV.gets if g[0].startswith(LIBRARY)),
                      {"kinds": sorted(kinds), "bad_gets": [g for g in scoped_gets if g[1] != want][:3], "posts": SRV.posts})
                done(ctx, pg, "E1", read_only=False)
                ctx, pg = case(); ready(pg); open_note(pg)
                SRV.post_status["/api/nudges"] = (403, {"error": "the owner grant cognition.nudge.draft is not enabled"})
                pg.get_by_role("button", name="Manual nudge (draft)").click(); pg.wait_for_selector("text=Manual nudge — draft")
                pg.get_by_role("button", name="Save draft (local only, no model call)").click(); pg.wait_for_timeout(500)
                v = probe(pg, [])
                check("E2 a POST 403 (write grant) is write feedback only: the page and view stay", v["open"] and v["tiles"] == 3
                      and "Draft not saved" in v["toast"] and v["mode"] == "LIVE / READ ONLY", brief(v))
                done(ctx, pg, "E2", read_only=False)
                ctx, pg = case(); ready(pg); open_note(pg)
                pg.get_by_role("button", name="Manual nudge (draft)").click(); pg.wait_for_selector("text=Manual nudge — draft")
                SRV.owner = "B"
                pg.get_by_role("button", name="Save draft (local only, no model call)").click(); pg.wait_for_timeout(1200)
                v = probe(pg)
                check("E3 a POST 412 (owner changed): nothing saved; the page revalidates into B",
                      SRV.saved == [] and not v["leaks"] and "Nova" in v["banner"], {"saved": SRV.saved, **brief(v)})
                done(ctx, pg, "E3", read_only=False)
            for sec in (sec_0, sec_1, sec_2, sec_3, sec_4, sec_5, sec_6, sec_7, sec_8, sec_9, sec_10, sec_11, sec_12, sec_13, sec_14, sec_15, sec_16, sec_17,):
                try:
                    sec()
                except Exception as e:
                    check((sec.__doc__ or sec.__name__) + " raised before all its checks ran", False, repr(e).splitlines()[0][:220])
                finally:
                    for c in list(OPEN):
                        c.close(); OPEN.remove(c)
    finally:
        browser.close()

# ---------------- T1 (supporting, once) ----------------
fetches = HTML.count("fetch(")
if fetches == 2:
    passed += 1
else:
    failures.append(f"T1 (supporting) exactly two fetch( call sites: found {fetches}")
for f in failures:
    print("FAIL:", f)
total = passed + len(failures)
print(f"{passed}/{total} lifecycle checks PASS across 1280/960/390 ({HTML_PATH})")
sys.exit(1 if failures else 0)
