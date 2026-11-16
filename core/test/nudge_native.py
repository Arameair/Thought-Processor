#!/usr/bin/env python3
"""#96 manual nudge drafts end to end with the REAL Go services, no Docker.

Real pinned Identity (with the cognition.nudge.draft grant) and Thought Engine binaries, the real Core
binary, the TEST FIXTURE model (NOT Nemotron), no GitHub repository (local only), in a throwaway APP_HOME.

  conversation -> a real persistent Thought in the Engine -> owner links Cognition to the Engine entity
  -> no grant: POST /api/nudges is 403, nothing written
  -> owner enables "cognition.nudge-draft" on Identity (passphrase) -> draft saved (201)
  -> same request = same draft (200 replayed) -> restart Core -> same draft in history
  -> owner revokes the grant -> the same request is 403; history still readable
  -> ZERO model requests and zero new Engine jobs/cycles from any nudge call; a card-position draft is
     NOT_FOR_PROCESSING (P0 #106: derived state, no stale BLOCKED)

    APP_HOME=/tmp/e2e/nudge BIN=/tmp/e2e/bin SEED=/tmp/e2e/seed CORE_BIN=tass-core-int \
    python3 core/test/nudge_native.py
"""
import json, os, sys, time

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
os.environ.setdefault("CAND", "x"); os.environ.setdefault("ALPHA", "x")
import lesson_retry_native as n  # noqa: E402
from acceptance import CORE, check, core_post, http, idn_post, log, results, token, wait  # noqa: E402

APP, BIN = n.APP, n.BIN
CORE_BIN = os.environ.get("CORE_BIN", "tass-core-int")
MODEL = "http://127.0.0.1:8081"
ENG = "http://127.0.0.1:7798"


def get(path):
    return http("GET", CORE + path, headers={"X-TASS-Token": token(CORE)})


def model_requests():
    return http("GET", MODEL + "/fixture/requests")[1].get("requests", 0)


def eng_counts():
    key = open(f"{APP}/data/engine/api-key").read().strip()
    cycles = http("GET", ENG + "/api/cycles", headers={"Authorization": "Bearer " + key})[1].get("cycles") or []
    jobs = 0
    p = f"{APP}/data/engine/jobs.jsonl"
    if os.path.exists(p):
        jobs = sum(1 for _ in open(p))
    return len(cycles), jobs


def start_core():
    n.start("core", [f"{BIN}/{CORE_BIN}"], {
        "HOME": "/tmp", "TASS_DATA": f"{APP}/data/core", "TASS_LISTEN": "127.0.0.1:8795",
        "TASS_ENGINE_URL": ENG, "TASS_ENGINE_KEY": f"{APP}/data/engine/api-key",
        "TASS_IDENTITY_URL": "http://127.0.0.1:8796", "TASS_IDENTITY_KEY_FILE": f"{APP}/secrets/core.key",
        "TASS_REPO": "", "TASS_TOKEN_FILE": f"{APP}/secrets/github_token"}, CORE + "/api/health")


def main():
    n.prepare()
    n.start("model", ["python3", f"{n.HERE}/fixture-model/fake_llama.py", "8081"], {}, MODEL + "/health")
    n.start("identity", [f"{BIN}/tass-identity"], {"HOME": "/tmp", "IDENTITY_DATA": f"{APP}/data/identity", "IDENTITY_LISTEN": "127.0.0.1:8796",
            "IDENTITY_SERVICE_KEYS": f"{APP}/secrets/service-keys", "IDENTITY_CORE_URL": "http://127.0.0.1:8795/"}, "http://127.0.0.1:8796/v1/health")
    n.start("engine", [f"{BIN}/thought-engine", "-data", f"{APP}/data/engine", "-model", MODEL, "-worker", "nemotron",
            "-identity-url", "http://127.0.0.1:8796", "-identity-key-file", f"{APP}/secrets/engine.key", "-pulse", "20s"],
            {"PORT": "7798", "HOME": "/tmp"}, ENG + "/api/model")
    start_core()
    try:
        run()
    finally:
        for name in list(n.procs):
            n.stop(name)
    print()
    for name, res, detail in results:
        print(f"{res:4}  {name}  {detail}")
    fails = [r for r in results if r[1] != "PASS"]
    print(f"\n{len(results) - len(fails)}/{len(results)} PASS (TEST FIXTURE model; not Nemotron, not the owner's machine)")
    sys.exit(1 if fails else 0)


def run():
    code, r = idn_post("/v1/entity", {"name": "Mira", "passphrase": n.PASS})
    check("real Identity: entity created UNLOCKED", code == 201 and r.get("state") == "UNLOCKED", str(code))
    grants = (r.get("entity") or {}).get("grants") or []
    check("cognition.nudge.draft is NOT a default grant", "cognition.nudge.draft" not in grants and "thought.process" in grants, ",".join(grants))

    # A real conversation turn makes the Engine record a cycle and a persistent Thought.
    code, r = core_post("/api/say", {"text": "Hello Mira, what are you thinking about?"})
    check("conversation accepted", code in (200, 202), f"{code} {json.dumps(r)[:120]}")
    lib = wait("Engine entity candidate", lambda: (get("/api/cognition/library")[1].get("link") or {}).get("candidates"), 90, 2)
    check("Cognition lists the Engine entity for owner review", bool(lib), json.dumps(lib)[:160])
    code, r = core_post("/api/cognition/link", {"engine_entity_id": lib[0]["engine_entity_id"]})
    check("owner links Cognition to the Engine entity", code == 200, f"{code} {json.dumps(r)[:120]}")
    # Test setup (not a nudge call): three related observations, the way the Engine forms a persistent
    # Thought (Thought-engine: three related traces -> one Thought, with the fixture model's TOPIC).
    key = open(f"{APP}/data/engine/api-key").read().strip()
    eid = lib[0]["engine_entity_id"]
    for i in range(3):
        code, r = http("POST", f"{ENG}/api/entities/{eid}/events", {"event_id": f"native-obs-{i}", "source": "test",
                       "text": f"The owner worked on the local project again (observation {i})"}, {"Authorization": "Bearer " + key})
        check(f"setup: observation {i} accepted by the real Engine", code in (200, 202), f"{code} {json.dumps(r)[:100]}")
    thoughts = wait("a real persistent Thought", lambda: get("/api/cognition/library")[1].get("thoughts"), 120, 2)
    check("a real persistent Thought is listed (no fixture card)", bool(thoughts), json.dumps(thoughts or [])[:200])
    if not thoughts:
        return
    tid = thoughts[0]["source_id"]
    body = {"source_thought_id": tid, "recipient_id": "position-1", "reason_code": "owner_request",
            "requested_action": "Look at this Thought again tomorrow.", "idempotency_key": "native-key-0001"}
    time.sleep(3)  # let the conversation's own cycle finish
    base_model, (base_cycles, base_jobs) = model_requests(), eng_counts()

    code, r = core_post("/api/nudges", body)
    check("no grant: draft refused 403 and names the grant", code == 403 and "cognition.nudge-draft" in json.dumps(r), f"{code} {json.dumps(r)[:160]}")
    check("no grant: nothing written", not os.path.exists(f"{APP}/data/core/nudges.jsonl"), "")

    code, r = idn_post("/v1/grants/skill", {"skill": "cognition.nudge-draft", "passphrase": "wrong-passphrase-123"})
    check("enable with a wrong passphrase is refused", code != 200, str(code))
    code, r = idn_post("/v1/grants/skill", {"skill": "cognition.nudge-draft", "passphrase": n.PASS})
    check("owner enables cognition.nudge-draft with the passphrase", code == 200, f"{code} {json.dumps(r)[:120]}")

    code, d = core_post("/api/nudges", body)
    check("draft saved 201, DRAFT, processing NOT_FOR_PROCESSING (card position; derived, no stale BLOCKED), model_calls 0",
          code == 201 and d.get("status") == "DRAFT" and (d.get("processing") or {}).get("state") == "NOT_FOR_PROCESSING" and d.get("model_calls") == 0,
          f"{code} {json.dumps(d)[:200]}")
    nid = d.get("nudge_id")
    code, d2 = core_post("/api/nudges", body)
    check("same request = same draft (200 replayed)", code == 200 and d2.get("replayed") and d2.get("nudge_id") == nid, f"{code}")
    code, d3 = core_post("/api/nudges", dict(body, recipient_id="discord", idempotency_key="native-key-0002"))
    check("external recipient refused 400", code == 400, str(code))

    n.stop("core")
    start_core()
    code, h = get("/api/nudges?thought_id=" + tid)
    ids_ = [x["nudge_id"] for x in h.get("nudges") or []]
    check("after Core restart: the same single draft is in history", code == 200 and ids_ == [nid], f"{code} {ids_}")
    mode = oct(os.stat(f"{APP}/data/core/nudges.jsonl").st_mode & 0o777)
    check("journal is 0600 and local", mode == "0o600", mode)

    code, r = idn_post("/v1/grants/skill", {"skill": "cognition.nudge-draft", "revoke": True})
    check("owner revokes the grant", code == 200, str(code))
    code, r = core_post("/api/nudges", body)
    check("after revoke: the SAME request is 403", code == 403 and not r.get("nudge_id"), f"{code} {json.dumps(r)[:120]}")
    code, h = get("/api/nudges")
    check("after revoke: history still readable, one draft", code == 200 and len(h.get("nudges") or []) == 1, str(code))
    lines = sum(1 for _ in open(f"{APP}/data/core/nudges.jsonl"))
    check("exactly one record in the journal", lines == 1, str(lines))

    model, (cycles, jobs) = model_requests(), eng_counts()
    check("ZERO model requests from all nudge calls", model == base_model, f"{base_model} -> {model}")
    check("no Engine cycle or job from any nudge call", (cycles, jobs) == (base_cycles, base_jobs),
          f"cycles {base_cycles}->{cycles}, jobs {base_jobs}->{jobs}")
    decs = [json.loads(l) for l in open(f"{APP}/data/identity/decisions.jsonl")]
    nd = [x for x in decs if x.get("action") == "cognition.nudge.draft"]
    check("Identity audited every draft decision (allow and deny)", any(x.get("allow") for x in nd) and any(not x.get("allow") for x in nd), f"{len(nd)} decisions")


if __name__ == "__main__":
    main()
