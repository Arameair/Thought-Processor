#!/usr/bin/env python3
"""#72 first @ skill, end to end with REAL Identity and Thought Engine binaries (TEST FIXTURE model
and fake GitHub). No Docker needed. The entity is created on the owner's current Identity
(91e1521) and then Identity is upgraded in place to the skill-grant build, like ./tass.sh update.

    APP_HOME=/tmp/e2e/app2 BIN=/tmp/e2e/bin SEED=/tmp/e2e/seed CAND=<core sha> ALPHA=unused \
    OLD_IDENTITY=tass-identity NEW_IDENTITY=tass-identity-new CORE_BIN=tass-core-c2 \
    python3 core/test/skill_native.py
"""
import json, os, sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import lesson_retry_native as n  # noqa: E402  (process helpers; reads the same env)
from acceptance import CORE, check, core_post, http, idn_post, results, wait  # noqa: E402

SKILL = "lesson-workspace.list-lessons"
OLD_ID, NEW_ID, CORE_BIN = os.environ["OLD_IDENTITY"], os.environ["NEW_IDENTITY"], os.environ["CORE_BIN"]


def start_identity(binary):
    n.start("identity", [f"{n.BIN}/{binary}"], {"HOME": "/tmp", "IDENTITY_DATA": f"{n.APP}/data/identity", "IDENTITY_LISTEN": "127.0.0.1:8796",
            "IDENTITY_SERVICE_KEYS": f"{n.APP}/secrets/service-keys", "IDENTITY_CORE_URL": "http://127.0.0.1:8795/"}, "http://127.0.0.1:8796/v1/health")


def start_engine():
    n.start("engine", [f"{n.BIN}/thought-engine", "-data", f"{n.APP}/data/engine", "-model", "http://127.0.0.1:8081", "-worker", "nemotron",
            "-identity-url", "http://127.0.0.1:8796", "-identity-key-file", f"{n.APP}/secrets/engine.key", "-pulse", "20s"],
            {"PORT": "7798", "HOME": "/tmp"}, "http://127.0.0.1:7798/api/model")


def status():
    return http("GET", CORE + "/api/status", headers={"X-TASS-Token": __import__("acceptance").token(CORE)})[1]


def receipts():
    return (status().get("skills") or {}).get("receipts") or []


def newest():
    r = receipts()
    return r[0] if r else {}


def say(text):
    before = len(receipts())
    code, r = core_post("/api/say", {"text": text})
    wait("skill receipt", lambda: len(receipts()) > before, 20, 0.5)
    return code


def main():
    n.prepare()
    n.start("fake-github", ["python3", f"{n.HERE}/fake_github.py", "28999", "Mira/home", "live", n.SEED], {}, "http://127.0.0.1:28999/fixture/log")
    n.start("model", ["python3", f"{n.HERE}/fixture-model/fake_llama.py", "8081"], {}, "http://127.0.0.1:8081/health")
    start_identity(OLD_ID)
    start_engine()
    n.start_core(CORE_BIN)
    try:
        run()
    finally:
        for p in list(n.procs):
            n.stop(p)
    print()
    for name, res, detail in results:
        print(f"{res:4}  {name}  {detail}")
    fails = [r for r in results if r[1] != "PASS"]
    print(f"\n{len(results) - len(fails)}/{len(results)} PASS")
    sys.exit(1 if fails else 0)


def run():
    code, r = idn_post("/v1/entity", {"name": "Mira", "passphrase": n.PASS})
    check("entity created on the owner's current Identity 91e1521", code == 201, str(code))
    ent = r["entity"]["entity_id"]
    wait("tass.md", lambda: os.path.exists(f"{n.APP}/data/core/tass.md"), 60)
    core_post("/api/lessons/discover", {})

    # Upgrade Identity in place (same data and keys).
    n.stop("identity")
    start_identity(NEW_ID)
    check("upgraded Identity: same entity, LOCKED until the owner unlocks", http("GET", "http://127.0.0.1:8796/v1/state")[1]["entity"]["entity_id"] == ent, ent)
    check("unlock with the same passphrase", bool(n.unlock()), "")
    st = http("GET", "http://127.0.0.1:8796/v1/state")[1]
    check("existing entity has NO skill grant after upgrade", not any(g.startswith("skill.") for g in st["entity"]["grants"]), ",".join(st["entity"]["grants"]))

    say("Find lessons using @")
    r = newest()
    check("before the grant: @ request is DENIED, nothing ran", r.get("state") == "DENIED" and not r.get("executed") and "Enable this skill" in r.get("error", ""), json.dumps({k: r.get(k) for k in ("state", "error")})[:220])

    code, _ = idn_post("/v1/grants/skill", {"skill": "shell.run", "passphrase": n.PASS})
    check("Identity refuses a non-allowlisted skill", code == 403, str(code))
    code, _ = idn_post("/v1/grants/skill", {"skill": SKILL, "passphrase": "wrong-passphrase-123"})
    check("Identity refuses the skill grant with a wrong passphrase", code == 403, str(code))
    code, _ = idn_post("/v1/grants/skill", {"skill": SKILL, "passphrase": n.PASS})
    check("owner enables @" + SKILL + " on Identity", code == 200, str(code))

    say("Find lessons using @")
    r = newest()
    check("Talk 'Find lessons using @' → registry selects the skill, AWAITING_APPROVAL, not run",
          r.get("skill") == SKILL and r.get("state") == "AWAITING_APPROVAL" and not r.get("executed") and "lessons" in r.get("why", ""), json.dumps({k: r.get(k) for k in ("skill", "state", "why")})[:260])
    rid = r["receipt_id"]
    code, a = core_post("/api/skills/approve", {"receipt": rid})
    a = a.get("receipt", {})
    check("owner approves → real Identity allows skill.run, ran once, DONE", code == 200 and a.get("state") == "DONE" and a.get("executed") and a.get("exit_code") == 0 and "skill.run:" + SKILL in a.get("run_decision", ""),
          json.dumps({k: a.get(k) for k in ("state", "exit_code", "run_decision")})[:260])
    check("real output lists the 3 seeded lessons", all(p in a.get("output", "") for p in ("01-my-work-pulse.md", "02-how-i-learn.md", "03-building-together.md")) and "count=3" in a.get("output", ""), a.get("output", "")[:300].replace("\n", " | "))
    check("expected vs observed: MET", a.get("expectation_met") is True, a.get("observed", ""))
    lines = len(open(f"{n.APP}/data/core/skill_receipts.jsonl").read().splitlines())
    code, again = core_post("/api/skills/approve", {"receipt": rid})
    again = again.get("receipt", {})
    check("approving the same receipt again does NOT run again", again.get("finished_at") == a.get("finished_at") and len(open(f"{n.APP}/data/core/skill_receipts.jsonl").read().splitlines()) == lines, "")
    dec = [json.loads(x) for x in open(f"{n.APP}/data/identity/decisions.jsonl") if x.strip()]
    check("Identity's own audit log has the allow decision for skill.run", any(d.get("action") == "skill.run:" + SKILL and d.get("allow") for d in dec), "")

    # Revoke, then a new request is denied at point of use.
    say("@lesson-workspace.list-lessons")
    r2 = newest()
    code, _ = idn_post("/v1/grants/skill", {"skill": SKILL, "revoke": True})
    check("owner revokes on Identity (no passphrase needed)", code == 200, str(code))
    code, d = core_post("/api/skills/approve", {"receipt": r2["receipt_id"]})
    d = d.get("receipt", {})
    check("after revoke: approve → DENIED by real Identity, executed=false, no output", d.get("state") == "DENIED" and not d.get("executed") and not d.get("output"), json.dumps({k: d.get(k) for k in ("state", "run_decision")})[:220])

    # Restart all three; receipts persist; nothing re-runs.
    for p in ("core", "engine", "identity"):
        n.stop(p)
    start_identity(NEW_ID)
    start_engine()
    n.start_core(CORE_BIN)
    n.unlock()
    rs = {x["receipt_id"]: x for x in receipts()}
    check("after restart: DONE receipt kept with the same output hash", rs.get(rid, {}).get("output_sha256") == a.get("output_sha256"), "")
    check("after restart: DENIED receipt kept", rs.get(r2["receipt_id"], {}).get("state") == "DENIED", "")
    check("after restart: nothing ran again (only r2's request + denial were logged)", len(open(f"{n.APP}/data/core/skill_receipts.jsonl").read().splitlines()) == lines + 2, "")

    # Fresh conversation: re-grant and reuse the skill from a new request.
    core_post("/api/conversation/new", {})
    idn_post("/v1/grants/skill", {"skill": SKILL, "passphrase": n.PASS})
    say("TASS, what lessons are in your workspace? use @")
    r3 = newest()
    code, e = core_post("/api/skills/approve", {"receipt": r3["receipt_id"]})
    e = e.get("receipt", {})
    check("fresh conversation: new request → approve → DONE with the same listing", e.get("state") == "DONE" and e.get("output_sha256") == a.get("output_sha256"), e.get("receipt_id", ""))

    # Workspace file view.
    import hashlib
    raw = open(f"{n.SEED}/lessons/claude/01-my-work-pulse.md", "rb").read()
    blob = hashlib.sha1(b"blob %d\x00" % len(raw) + raw).hexdigest()   # = git hash-object (real GitHub's sha)
    code, f = core_post("/api/workspace/file", {"path": "lessons/claude/01-my-work-pulse.md"})
    check("workspace: read-only lesson file view; local blob SHA = git hash-object of the file (fake GitHub reports plain sha1, so MET vs GitHub needs real GitHub)",
          code == 200 and f.get("read_only") and f.get("local_blob_sha") == blob and f.get("text", "").startswith("# My work pulse"), json.dumps({k: f.get(k) for k in ("local_blob_sha", "github_sha")}))
    code, _ = core_post("/api/workspace/file", {"path": "../secrets/github_token"})
    check("workspace: path outside the allowlist refused", code == 404, str(code))


if __name__ == "__main__":
    main()
