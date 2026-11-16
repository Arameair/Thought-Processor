#!/usr/bin/env python3
"""Stage 3B runtime on the REAL stack: real Core, real Thought Engine, real Identity, the TEST FIXTURE model
(not Nemotron), isolated ports with owner-port sentinels. Synthetic data only.

Design #107 §7 C1–C11 plus the owner's frozen-snapshot requirement:
  C1  grant off (the default) -> arm refused 403, nothing captured, zero model requests
  C2  the Thought file's sha256 is unchanged by A and by B
  C3  a consumed approval cannot be used again (direct Engine request)
  C4  a confirmed replay shows the record and makes zero requests
  C5  A's prompt is byte-for-byte the nudge prompt; B's differs by the instruction sentence only
  C6  one model slot: a nudge and a second arm are refused while an arm runs
  C7  kill -9 the Engine mid-request -> UNCERTAIN after restart, never re-sent, B refused
  C8  saving a plan makes zero model requests
  C9  the nudge grant does not authorize an experiment; revoked / LOCKED -> refused
  C10 a plan from before a restore is PRE_RESTORE and cannot run; a plan saved after can
  C11 the reply-format lines cannot be edited (400)
  S1  one snapshot: captured once before A, B uses it after the live Thought changed and after an Engine
      restart; a changed snapshot file refuses B with nothing sent
    APP_HOME=… BIN=… SEED=… CORE_BIN=… ENGINE_BIN=… IDENTITY_BIN=… ENGINE_CWD=<Engine checkout> python3 experiment_native.py
"""
import glob
import hashlib
import json
import os
import sys
import time

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
os.environ.setdefault("FIXTURE_DELAY", "8")
import nudge_process_native as h  # noqa: E402  (runs the isolation preflight on import)
from acceptance import core_post, http, idn_post, token, wait  # noqa: E402

results = []
CONTROL = "Add ONE new idea, possible explanation, or thing to watch for next. Do not repeat the evidence or earlier notes."
VARIANT = "Answer the Thought's question directly from the evidence; if it can't be answered, say exactly what is missing."


def check(name, ok, detail=""):
    results.append((name, "PASS" if ok else "FAIL", "" if ok else str(detail)[:500]))
    print(("PASS  " if ok else "FAIL  ") + name + ("" if ok else "  " + str(detail)[:500]), flush=True)


def sha_file(p):
    return hashlib.sha256(open(p, "rb").read()).hexdigest()


def main():
    h.n.prepare()
    watched = h.start_sentinels()
    h.start("model", ["python3", f"{h.HERE}/fixture-model/fake_llama.py", str(h.P_MODEL)], {"FIXTURE_DELAY": os.environ["FIXTURE_DELAY"]}, h.MODEL + "/health")
    h.start("identity", [f"{h.BIN}/{h.IDENTITY_BIN}"], {"IDENTITY_DATA": f"{h.APP}/data/identity", "IDENTITY_LISTEN": f"127.0.0.1:{h.P_IDN}",
            "IDENTITY_SERVICE_KEYS": f"{h.APP}/secrets/service-keys", "IDENTITY_CORE_URL": h.CORE + "/"}, h.IDN + "/v1/health")
    h.start_engine(); h.start_proxy(); h.start_core()
    try:
        run()
    finally:
        for name in list(h.n.procs):
            h.n.stop(name)
    check("ISOLATION: no connection to any watched owner port", not h.SENTINEL_HITS, str(sorted(set(h.SENTINEL_HITS))))
    fails = [r for r in results if r[1] == "FAIL"]
    print(f"\n{len(results) - len(fails)}/{len(results)} PASS, {len(fails)} FAIL "
          f"(sentinels {watched}; TEST FIXTURE model; not Nemotron, not the owner's machine)")
    sys.exit(1 if fails else 0)


def run():
    code, r = idn_post("/v1/entity", {"name": "Mira", "passphrase": h.n.PASS})
    check("setup: real Identity entity UNLOCKED", code == 201, str(code))
    core_post("/api/say", {"text": "Hello Mira, what are you thinking about?"})
    cands = wait("Engine entity", lambda: (h.get("/api/cognition/library")[1].get("link") or {}).get("candidates"), 90, 2)
    eid = cands[0]["engine_entity_id"]
    check("setup: owner links Cognition", core_post("/api/cognition/link", {"engine_entity_id": eid})[0] == 200)
    ek = {"Authorization": "Bearer " + h.ekey()}
    for i in range(3):
        http("POST", f"{h.ENG}/api/entities/{eid}/events", {"event_id": f"s3b-obs-{i}", "source": "test",
             "text": f"The owner worked on the local project again (observation {i})"}, ek)
    thoughts = wait("a persistent Thought", lambda: h.get("/api/cognition/library")[1].get("thoughts"), 120, 2)
    tid = thoughts[0]["source_id"]
    http("POST", f"{h.ENG}/api/entities/{eid}/grant", {"enabled": True}, ek)
    idn_post("/v1/grants/skill", {"skill": "cognition.nudge-draft", "passphrase": h.n.PASS})
    idn_post("/v1/grants/skill", {"skill": "cognition.nudge-process", "passphrase": h.n.PASS})
    time.sleep(3)
    tok = token(h.CORE)
    T = {"X-TASS-Token": tok}

    def nudge(text, key):
        code, d = h.draft(tid, text, key)
        code2, _ = h.process(d)
        st = wait("nudge " + key, lambda: h.run_state(d["nudge_id"]) in ("COMPLETED", "FAILED", "UNCERTAIN") and h.run_state(d["nudge_id"]), 90, 1)
        return d["nudge_id"], (code, code2, st)

    ATT = "Pay attention to whether the local project keeps coming back."
    nid, res = nudge(ATT, "s3b-key-0001")
    check("setup: one real supervised nudge COMPLETED (the baseline run)", res == (201, 202, "COMPLETED"), res)

    def plan(baseline, variant, key):
        return http("POST", h.CORE + "/api/development/experiments", {"baseline_nudge_id": baseline, "variant": variant, "snapshot": "AS_OF_NOW", "idempotency_key": key}, T)

    def view(exp):
        return http("GET", h.CORE + "/api/development/experiments/" + exp, None, T)[1]

    def arm(p, a, confirm=None):
        return http("POST", h.CORE + f"/api/development/experiments/{p['experiment_id']}/arms/{a}",
                    {"body_digest": p["body_digest"], "confirm": confirm if confirm is not None else f"RUN {a} {p['experiment_id']}"}, T)

    def arm_state(exp, a):
        return next((x for x in view(exp).get("arms", []) if x["arm"] == a), {})

    def wait_arm(exp, a, timeout=60):
        return wait(f"arm {a}", lambda: arm_state(exp, a).get("state") in ("COMPLETED", "FAILED", "UNCERTAIN", "NOT_DELIVERED", "RECOVERY_REQUIRED") and arm_state(exp, a), timeout, 0.5)

    def requests():
        return h.fixture()["requests"]

    def prompts():
        return json.loads(http("GET", h.MODEL + "/fixture/thought-prompts")[1] if False else __import__("urllib.request").request.urlopen(h.MODEL + "/fixture/thought-prompts").read())["prompts"]

    edir = glob.glob(f"{h.APP}/data/engine/entities/*/")
    tfile = glob.glob(f"{h.APP}/data/engine/**/thoughts/{tid}.json", recursive=True)
    check("setup: the Thought's file is found on disk", len(tfile) == 1, str(tfile) + str(edir)[:100])
    tfile = tfile[0]
    expdir = os.path.join(os.path.dirname(os.path.dirname(tfile)), "experiments")

    # ---- C8: saving a plan
    base = requests()
    c1, p1 = plan(nid, VARIANT, "s3b-plan-0001")
    check("C8 saving a plan: 201 and zero model requests", c1 == 201 and requests() == base and p1.get("experiment_id", "").startswith("exp_"), f"{c1} {str(p1)[:200]}")
    exp = p1["experiment_id"]
    c, r = plan(nid, "Answer. NOTE: yes", "s3b-plan-bad1")
    check("C11 the reply-format lines cannot be edited (400)", c == 400, f"{c} {r}")

    # ---- C1 / C9: the grant is off by default and separate from the nudge grant
    grants = (json.loads(__import__("urllib.request").request.urlopen(h.IDN + "/v1/state").read()).get("entity") or {}).get("grants", [])
    check("C9 cognition.experiment.run is NOT a default grant, although the nudge grant is held",
          "cognition.experiment.run" not in grants and "cognition.nudge.process" in grants, ",".join(grants))
    v = view(exp)
    check("C1 the page shows A not runnable: NOT AUTHORIZED, grant named", not arm_state(exp, "A").get("runnable") and "cognition.experiment-run" in arm_state(exp, "A").get("why", "")
          and v.get("frozen_snapshot", {}).get("status") == "NOT CAPTURED", json.dumps(arm_state(exp, "A"))[:300])
    sent0 = len(h.PROXIED)
    c, r = arm(p1, "A")
    check("C1 grant off: arm A refused 403; nothing captured, sent or requested",
          c == 403 and r.get("code") == "EXPERIMENT_NOT_GRANTED" and requests() == base and len(h.PROXIED) == sent0 and not os.path.exists(os.path.join(expdir, exp)), f"{c} {r}")
    c, _ = idn_post("/v1/grants/skill", {"skill": "cognition.experiment-run", "passphrase": "wrong-passphrase-123"})
    check("enabling the experiment grant with a wrong passphrase is refused", c != 200, str(c))
    c, _ = idn_post("/v1/grants/skill", {"skill": "cognition.experiment-run", "passphrase": h.n.PASS})
    check("the owner enables cognition.experiment-run (TEST Identity only)", c == 200, str(c))
    c, r = arm(p1, "A", confirm="")
    check("an arm without the explicit confirmation is refused (409), nothing sent", c == 409 and len(h.PROXIED) == sent0 and requests() == base, f"{c} {r}")
    c, r = arm(p1, "B")
    check("B before A is refused, nothing sent", c == 409 and len(h.PROXIED) == sent0 and requests() == base, f"{c} {r}")

    # ---- A
    t_sha0 = sha_file(tfile)
    n_prompts = len(prompts())
    c, r = arm(p1, "A")
    a = wait_arm(exp, "A")
    check("A: 202 accepted, then COMPLETED with a recorded reply", c == 202 and a.get("state") == "COMPLETED" and a.get("reply", "").startswith("NOTE:"), f"{c} {json.dumps(a)[:300]}")
    check("A: exactly ONE model request", requests() == base + 1, f"{base} -> {requests()}")
    snap_file = os.path.join(expdir, exp, "snapshot.json")
    v = view(exp)
    fz = v.get("frozen_snapshot", {})
    snap_raw = open(snap_file, "rb").read() if os.path.exists(snap_file) else b""
    check("S1 one snapshot was captured before A: stored at the Engine, its sha256 is the one Core recorded and shows",
          fz.get("status") == "FROZEN" and hashlib.sha256(snap_raw.rstrip(b"\n")).hexdigest() == fz.get("sha256") == a.get("snapshot_sha256")
          and fz.get("reference") == "snap-" + fz.get("sha256", "")[:12], json.dumps(fz)[:300])
    check("S1 Core sent one capture and one arm request", [p.rsplit("/", 2)[-2] + "/" + p.rsplit("/", 1)[-1] if "/arms/" in p else p.rsplit("/", 1)[-1] for p in h.PROXIED[sent0:]] == ["snapshot", "arms/A"], h.PROXIED[sent0:])
    check("C2 the Thought file is byte-identical after A", sha_file(tfile) == t_sha0, "")
    pa = prompts()[n_prompts:]
    nudge_prompt = json.loads(__import__("urllib.request").request.urlopen(h.MODEL + "/fixture/last-nudge").read())["prompt"]
    check("C5 A sent one prompt, with the control sentence and the owner's attention", len(pa) == 1 and pa[0].count(CONTROL) == 1 and ATT in pa[0], str(pa)[:300])
    prompt_a = pa[0] if pa else ""
    check("A's receipt: request ID, single-use approval, model HTTP responded, 1 request, not saved into the Thought",
          a.get("request_id", "").startswith("mr_") and a.get("identity_decision", "").startswith("dec_") and a.get("model_http") == "responded"
          and a.get("model_requests") == 1 and a.get("saved_into_thought") == "no" and v.get("model_requests") == 1, json.dumps(a)[:300])
    check("after A: B is runnable, A is not, nothing to compare yet", arm_state(exp, "B").get("runnable") and not a.get("runnable") and not v["comparison"]["ready"], "")

    # ---- the live Thought changes, and the Engine restarts, between A and B
    nid2, res2 = nudge("Also look at how often it happens.", "s3b-key-0002")
    t_sha1 = sha_file(tfile)
    check("between the arms a real nudge changes the live Thought", res2[2] == "COMPLETED" and t_sha1 != t_sha0, res2)
    drift_prompt_marker = "check whether the pattern holds"  # the fixture's Note, now among the Thought's Notes
    h.kill9("engine"); h.start_engine(); time.sleep(2)
    base = requests(); sent1 = len(h.PROXIED); n_prompts = len(prompts())
    c, r = arm(p1, "B")
    b = wait_arm(exp, "B")
    check("B: 202 accepted, then COMPLETED, exactly ONE model request, no second capture",
          c == 202 and b.get("state") == "COMPLETED" and requests() == base + 1 and [p for p in h.PROXIED[sent1:] if p.endswith("/snapshot")] == [], f"{c} {json.dumps(b)[:300]} {h.PROXIED[sent1:]}")
    pb = prompts()[n_prompts:]
    check("S1/C5 B's prompt is A's with ONLY the instruction sentence replaced, built from the frozen snapshot (no later Note in it)",
          len(pb) == 1 and pb[0] == prompt_a.replace(CONTROL, VARIANT) and prompt_a.count(drift_prompt_marker) == pb[0].count(drift_prompt_marker), (pb[0][-400:] if pb else ""))
    check("S1 the snapshot file is unchanged and both arms name its digest", open(snap_file, "rb").read() == snap_raw and b.get("snapshot_sha256") == fz.get("sha256"), "")
    check("C2 the Thought file is byte-identical after B", sha_file(tfile) == t_sha1, "")
    v = view(exp)
    check("the comparison is ready, replies differ, 2 of at most 2 requests, both arms 'saved into the Thought: no'",
          v["comparison"]["ready"] and not v["comparison"]["same_reply"] and v["model_requests"] == 2 and b.get("reply") != a.get("reply")
          and all(x["saved_into_thought"] == "no" for x in v["arms"]) and v["status"] == "Both arms completed.", json.dumps(v["comparison"]))
    code_t, td = h.thought_detail(eid, tid)
    check("C2 no experiment reply is in the Thought", "variant reply" not in json.dumps(td), "")

    # ---- C4 / C3: replay
    base = requests(); sent2 = len(h.PROXIED)
    ca, _ = arm(p1, "A"); cb, _ = arm(p1, "B")
    check("C4 confirmed replays show the record: 200, nothing sent to the Engine, zero requests", (ca, cb) == (200, 200) and len(h.PROXIED) == sent2 and requests() == base, f"{ca} {cb}")
    sub = json.load(open(f"{h.APP}/data/core/experiment-runs/{exp}/B.submitted.json"))
    c, r = http("POST", f"{h.ENG}/api/entities/{eid}/thoughts/{tid}/experiments/{exp}/arms/B",
                {"owner_entity_id": sub["owner_entity_id"], "snapshot_sha256": sub["snapshot_sha256"], "variant": VARIANT, "variant_sha256": sub["variant_sha256"], "decision_id": sub["identity_decision"]}, ek)
    check("C3 a consumed approval sent straight to the Engine is refused (403), zero requests", c == 403 and requests() == base, f"{c} {r}")
    c, r = http("POST", f"{h.ENG}/api/entities/{eid}/thoughts/{tid}/experiments/{exp}/arms/A",
                {"owner_entity_id": sub["owner_entity_id"], "snapshot_sha256": sub["snapshot_sha256"], "variant": VARIANT, "variant_sha256": sub["variant_sha256"], "decision_id": "dec_made_up"}, ek)
    check("C3 a made-up approval is refused (403)", c == 403 and requests() == base, f"{c} {r}")
    c, _ = http("POST", f"{h.ENG}/api/entities/{eid}/thoughts/{tid}/experiments/{exp}/snapshot", {"owner_entity_id": sub["owner_entity_id"], "attention": ATT, "attention_sha256": hashlib.sha256(ATT.encode()).hexdigest()})
    check("the Engine's experiment routes need the Engine key (401)", c == 401, str(c))
    c, r = http("GET", f"{h.ENG}/api/entities/ent_nobody/experiments/{exp}", None, ek)
    check("another Engine entity cannot read the experiment (404)", c == 404, f"{c}")

    # ---- C9: revoke / LOCKED ; S1: tampered snapshot
    _, p2 = plan(nid, "Say exactly what evidence is missing.", "s3b-plan-0002")
    exp2 = p2["experiment_id"]
    idn_post("/v1/grants/skill", {"skill": "cognition.experiment-run", "revoke": True})
    base = requests(); sent3 = len(h.PROXIED)
    c, r = arm(p2, "A")
    check("C9 grant revoked: arm refused 403, nothing captured or sent; the nudge grant is still held", c == 403 and len(h.PROXIED) == sent3 and requests() == base
          and "cognition.nudge.process" in (json.loads(__import__("urllib.request").request.urlopen(h.IDN + "/v1/state").read()).get("entity") or {}).get("grants", []), f"{c} {r}")
    idn_post("/v1/grants/skill", {"skill": "cognition.experiment-run", "passphrase": h.n.PASS})
    idn_post("/v1/lock", {})
    c, r = arm(p2, "A")
    check("C9 Identity LOCKED: arm refused, nothing sent", c in (403, 423) and len(h.PROXIED) == sent3 and requests() == base, f"{c} {r}")
    c, _ = idn_post("/v1/unlock", {"passphrase": h.n.PASS})
    time.sleep(1)
    c, r = arm(p2, "A")
    a2 = wait_arm(exp2, "A")
    check("after unlock + grant: the second plan's A completes", c == 202 and a2.get("state") == "COMPLETED", f"{c} {json.dumps(a2)[:200]}")
    snap2 = os.path.join(expdir, exp2, "snapshot.json")
    good2 = open(snap2, "rb").read()
    open(snap2, "wb").write(good2.replace(b"local project", b"LOCAL PROJECT", 1))
    base = requests(); sent4 = len(h.PROXIED)
    c, r = arm(p2, "B")
    v2 = view(exp2)
    check("S1 a changed snapshot file: B refused (409), nothing sent to the model, the page says UNVERIFIED and B is not runnable",
          c == 409 and requests() == base and not [p for p in h.PROXIED[sent4:] if "/arms/" in p] and v2["frozen_snapshot"]["status"] == "UNVERIFIED"
          and not arm_state(exp2, "B").get("runnable") and arm_state(exp2, "B").get("state") == "NOT RUN", f"{c} {r} {json.dumps(v2['frozen_snapshot'])[:200]}")
    open(snap2, "wb").write(good2)

    # ---- F: the file boundary (Codex audit of f5c3b6d): each altered state refuses B with nothing sent; restored, B runs
    def refused(label, want_view, engine_side=True):
        base_f, sent_f = requests(), len(h.PROXIED)
        c_f, r_f = arm(p2, "B")
        v_f = view(exp2)
        a_f, b_f = arm_state(exp2, "A"), arm_state(exp2, "B")
        erec = http("GET", f"{h.ENG}/api/entities/{eid}/experiments/{exp2}", None, ek)[1].get("experiment", {})
        ok = (c_f == 409 and requests() == base_f and not [p for p in h.PROXIED[sent_f:] if "/arms/" in p] and not b_f.get("runnable")
              and b_f.get("state") != "COMPLETED" and not os.path.lexists(os.path.join(expdir, exp2, "B", "claim.json"))
              and not os.path.lexists(f"{h.APP}/data/core/experiment-runs/{exp2}/B.submitted.json") and sha_file(tfile) == t_now)
        if want_view == "UNVERIFIED":
            # engine_side: the Engine's own record must stop vouching for its file; a Core-side receipt
            # problem leaves the Engine's (untouched) snapshot verified, and Core alone refuses.
            ok = ok and v_f["frozen_snapshot"]["status"] == "UNVERIFIED" and bool((erec.get("snapshot") or {}).get("verified")) == (not engine_side)
        else:
            ok = ok and a_f.get("state") == "RECOVERY_REQUIRED" and not a_f.get("reply") and a2.get("reply", "x") not in json.dumps(v_f)
        check(f"F {label}: B refused (409), zero model requests, no B claim or submission, page says {want_view}, Thought unchanged", ok,
              f"{c_f} {r_f} snap={json.dumps(v_f['frozen_snapshot'])[:160]} A={a_f.get('state')} engine={json.dumps(erec.get('snapshot'))[:160]}")

    t_now = sha_file(tfile)
    os.chmod(snap2, 0o644)
    refused("Engine snapshot chmod 0644 (Codex 1)", "UNVERIFIED")
    os.chmod(snap2, 0o600)
    elsewhere = f"{h.APP}/elsewhere-snapshot.json"
    open(elsewhere, "wb").write(good2); os.chmod(elsewhere, 0o600)
    os.rename(snap2, snap2 + ".real"); os.symlink(elsewhere, snap2)
    refused("Engine snapshot replaced by a symlink to identical bytes (Codex 2)", "UNVERIFIED")
    os.remove(snap2); os.rename(snap2 + ".real", snap2)
    open(snap2, "wb").write(good2.rstrip(b"\n"))
    refused("Engine snapshot without its final newline (Codex 3)", "UNVERIFIED")
    open(snap2, "wb").write(good2); os.chmod(snap2, 0o600)
    os.chmod(os.path.join(expdir, exp2), 0o755)
    refused("Engine experiment directory chmod 0755", "UNVERIFIED")
    os.chmod(os.path.join(expdir, exp2), 0o700)
    crec = f"{h.APP}/data/core/experiment-runs/{exp2}"
    held = {f: open(os.path.join(crec, f), "rb").read() for f in os.listdir(crec)}
    os.chmod(os.path.join(crec, "A.outcome.json"), 0o644)
    refused("Core outcome receipt chmod 0644", "RECOVERY_REQUIRED")
    os.chmod(os.path.join(crec, "A.outcome.json"), 0o600)
    os.rename(os.path.join(crec, "snapshot.json"), os.path.join(crec, "snapshot.real")); os.symlink(os.path.join(crec, "snapshot.real"), os.path.join(crec, "snapshot.json"))
    refused("Core snapshot receipt replaced by a symlink", "UNVERIFIED", engine_side=False)
    os.remove(os.path.join(crec, "snapshot.json")); os.rename(os.path.join(crec, "snapshot.real"), os.path.join(crec, "snapshot.json"))
    check("F nothing was repaired or rewritten: every Core receipt and the Engine snapshot are byte-identical after the refusals",
          {f: open(os.path.join(crec, f), "rb").read() for f in os.listdir(crec)} == held and open(snap2, "rb").read() == good2, "")
    base = requests()
    c, r = arm(p2, "B")
    b2 = wait_arm(exp2, "B")
    check("F with the files as the Engine and Core wrote them, B runs: 202, COMPLETED, exactly one request, snapshot FROZEN",
          c == 202 and b2.get("state") == "COMPLETED" and requests() == base + 1 and view(exp2)["frozen_snapshot"]["status"] == "FROZEN", f"{c} {r} {json.dumps(b2)[:200]}")

    # ---- C6 + C7: a slow arm (the baseline's attention makes the fixture model wait)
    sid, sres = nudge("Take it slow and look at the local project again.", "s3b-key-slow1")
    check("setup: a second baseline whose attention makes the fixture model slow", sres[2] == "COMPLETED", sres)
    _, p3 = plan(sid, VARIANT, "s3b-plan-0003")
    _, p4 = plan(sid, "Say exactly what evidence is missing.", "s3b-plan-0004")
    exp3, exp4 = p3["experiment_id"], p4["experiment_id"]
    base = requests()
    c, r = arm(p3, "A")
    wait("arm A at the model", lambda: os.path.exists(os.path.join(expdir, exp3, "A", "http.json")), 30, 0.2)
    code_d, d5 = h.draft(tid, "A nudge that must wait.", "s3b-key-0005")
    cn, rn = h.process(d5)
    co, ro = arm(p4, "A")
    check("C6 while an arm is at the model: a nudge is refused 409 and another plan's arm is refused 409 (BUSY)", cn == 409 and co == 409 and ro.get("code") == "BUSY", f"{cn} {rn} / {co} {ro}")
    check("C6 the running arm shows RUNNING or SUBMITTED, not runnable", arm_state(exp3, "A").get("state") in ("RUNNING", "SUBMITTED") and not arm_state(exp3, "A").get("runnable"), json.dumps(arm_state(exp3, "A"))[:200])
    h.kill9("engine")
    check("C7 kill -9 the Engine after the HTTP attempt record, before the response", os.path.exists(os.path.join(expdir, exp3, "A", "http.json"))
          and not os.path.exists(os.path.join(expdir, exp3, "A", "response.json")), str(os.listdir(os.path.join(expdir, exp3, "A"))))
    h.start_engine(); time.sleep(2)
    a3 = wait_arm(exp3, "A", 40)
    check("C7 after the restart the arm is UNCERTAIN with no reply shown", a3.get("state") == "UNCERTAIN" and not a3.get("reply") and a3.get("model_http") == "attempt_uncertain", json.dumps(a3)[:300])
    time.sleep(float(os.environ["FIXTURE_DELAY"]) + 2)
    n_after = requests()
    check("C7 the model received that arm at most once", n_after - base <= 1, f"{base} -> {n_after}")
    sent5 = len(h.PROXIED)
    ca, _ = arm(p3, "A"); cb, rb = arm(p3, "B")
    check("C7 never re-sent: A again shows UNCERTAIN (200), B is refused (409), nothing sent, zero requests",
          ca == 200 and cb == 409 and len(h.PROXIED) == sent5 and requests() == n_after and arm_state(exp3, "A").get("state") == "UNCERTAIN", f"{ca} {cb} {rb}")
    v3 = view(exp3)
    check("C7 the page says the experiment ended and B is final", v3["status"].startswith("Ended: A is UNCERTAIN") and "final" in arm_state(exp3, "B").get("why", ""), v3["status"])
    wait("the slot is free again", lambda: arm_state(exp4, "A").get("runnable"), 30, 1)

    # ---- C10: a restore fence (as tass.sh restore_data writes it) makes existing plans PRE_RESTORE
    core = f"{h.APP}/data/core"
    h.n.stop("core")
    nj, xj = open(f"{core}/nudges.jsonl", "rb").read(), open(f"{core}/experiments.jsonl", "rb").read()
    fence = {"schema": "tass.restore-fence/2", "restored_at": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()), "backup": "test.tgz",
             "journal_bytes": len(nj), "journal_sha256": hashlib.sha256(nj).hexdigest(), "experiments_bytes": len(xj), "experiments_sha256": hashlib.sha256(xj).hexdigest()}
    with open(f"{core}/restore-fence.json", "w") as f:
        json.dump(fence, f)
    os.chmod(f"{core}/restore-fence.json", 0o600)
    h.start_core(); time.sleep(2)
    tok = token(h.CORE); T["X-TASS-Token"] = tok
    base = requests(); sent6 = len(h.PROXIED)
    c, r = arm(p4, "A")
    check("C10 a plan from before the restore is PRE_RESTORE: refused 409, nothing sent, not runnable on the page",
          c == 409 and "PRE_RESTORE" in json.dumps(r) and len(h.PROXIED) == sent6 and requests() == base and "PRE_RESTORE" in arm_state(exp4, "A").get("why", ""), f"{c} {r}")
    check("C10 the completed experiment from before the restore is still readable", view(exp)["status"] == "Both arms completed.", "")
    sid2, sres2 = nudge("Look at the local project once more.", "s3b-key-0006")
    _, p5 = plan(sid2, VARIANT, "s3b-plan-0005")
    c, r = arm(p5, "A")
    a5 = wait_arm(p5["experiment_id"], "A")
    check("C10 a plan saved after the restore runs", sres2[2] == "COMPLETED" and c == 202 and a5.get("state") == "COMPLETED", f"{sres2} {c} {json.dumps(a5)[:200]}")

    # ---- logs
    calls = open(f"{h.APP}/data/engine/calls.jsonl", errors="replace").read()
    check("the Engine's call log holds experiment calls with the attention's digest only", '"purpose":"experiment"' in calls.replace(" ", "") and ATT not in calls
          and "[owner attention sha256:" in calls, "")
    check("experiment plans and receipts are never in tass.md", "exp_" not in open(f"{core}/tass.md", errors="replace").read() if os.path.exists(f"{core}/tass.md") else True, "")


if __name__ == "__main__":
    main()
