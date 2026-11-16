#!/usr/bin/env python3
"""Stage 1 slice 2 (WP1-D): MEASUREMENT of Identity's decision-log growth caused by Cognition reads.

A measurement, not an acceptance test: it changes nothing in Identity and asserts nothing about how
much Identity should log. It answers one question for the automatic refresh: how many lines does
Identity append to decisions.jsonl for ONE library read, ONE pulse read and ONE Thought read, and what
that means per hour at the page's cadence (library every 60 s, pulse every 10 s, visible tab only).

REAL Identity, REAL Thought Engine, REAL Core, the TEST FIXTURE model (NOT Nemotron), no Docker, a
throwaway APP_HOME. It reuses the isolated Stage 2 harness (nudge_process_native.py): its own ports,
preflight, clean environment and owner-port sentinels. It never touches the owner's machine or data.

    APP_HOME=/tmp/e2e/load BIN=/tmp/e2e/bin SEED=/tmp/e2e/seed \\
    CORE_BIN=tass-core-s2 ENGINE_BIN=thought-engine-s2 IDENTITY_BIN=tass-identity-s2 \\
    [S2_PORT_BASE=39100] python3 core/test/cognition_refresh_load_native.py
"""
import collections, json, os, sys, time, urllib.request

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import nudge_process_native as h  # noqa: E402  (runs the isolation preflight on import)
from acceptance import core_post, http, idn_post, token, wait  # noqa: E402

LOG = f"{h.APP}/data/identity/decisions.jsonl"
READS = int(os.environ.get("LOAD_READS", "20"))


def lines():
    try:
        return open(LOG, "rb").read().splitlines()
    except FileNotFoundError:
        return []


def get(path, tok, expect=None):
    """One GET as the page sends it. Returns (status, body bytes, headers, seconds)."""
    req = urllib.request.Request(h.CORE + path, headers={"X-TASS-Token": tok, **(expect or {})})
    t = time.time()
    try:
        with urllib.request.urlopen(req, timeout=30) as r:
            return r.status, r.read(), dict(r.headers), time.time() - t
    except urllib.error.HTTPError as e:
        return e.code, e.read(), dict(e.headers), time.time() - t


def measure(name, fn, n=READS):
    before = lines()
    sizes, secs, codes = [], [], collections.Counter()
    for _ in range(n):
        code, body, _, dt = fn()
        codes[code] += 1; sizes.append(len(body)); secs.append(dt)
    time.sleep(0.5)
    new = lines()[len(before):]
    kinds = collections.Counter()
    for ln in new:
        try:
            d = json.loads(ln)
            kinds[(str(d.get("action") or d.get("kind") or "?"), str(d.get("allow", d.get("result", "?"))))] += 1
        except Exception:
            kinds[("unparsed", "?")] += 1
    row = {"name": name, "reads": n, "status": dict(codes), "lines": len(new), "lines_per_read": len(new) / n if n else 0,
           "bytes_per_read": sum(len(x) + 1 for x in new) / n if n else 0, "response_bytes": max(sizes) if sizes else 0,
           "latency_ms_median": sorted(secs)[len(secs) // 2] * 1000 if secs else 0, "kinds": {f"{a} allow={b}": c for (a, b), c in kinds.items()}}
    print(json.dumps(row), flush=True)
    return row


def main():
    h.n.prepare()
    watched = h.start_sentinels()
    h.start("model", ["python3", f"{h.HERE}/fixture-model/fake_llama.py", str(h.P_MODEL)], {"FIXTURE_DELAY": "1"}, h.MODEL + "/health")
    h.start("identity", [f"{h.BIN}/{h.IDENTITY_BIN}"], {"IDENTITY_DATA": f"{h.APP}/data/identity", "IDENTITY_LISTEN": f"127.0.0.1:{h.P_IDN}",
            "IDENTITY_SERVICE_KEYS": f"{h.APP}/secrets/service-keys", "IDENTITY_CORE_URL": h.CORE + "/"}, h.IDN + "/v1/health")
    h.start_engine(); h.start_proxy(); h.start_core()
    rows = []
    try:
        code, r = idn_post("/v1/entity", {"name": "Mira", "passphrase": h.n.PASS})
        assert code == 201, ("entity", code, r)
        core_post("/api/say", {"text": "Hello Mira, what are you thinking about?"})
        cands = wait("Engine entity", lambda: (h.get("/api/cognition/library")[1].get("link") or {}).get("candidates"), 90, 2)
        assert cands, "no Engine entity candidate"
        eid = cands[0]["engine_entity_id"]
        assert core_post("/api/cognition/link", {"engine_entity_id": eid})[0] == 200
        for i in range(3):
            http("POST", f"{h.ENG}/api/entities/{eid}/events", {"event_id": f"load-obs-{i}", "source": "test",
                 "text": f"The owner worked on the local project again (observation {i})"}, {"Authorization": "Bearer " + h.ekey()})
        thoughts = wait("a persistent Thought", lambda: h.get("/api/cognition/library")[1].get("thoughts"), 120, 2)
        assert thoughts, "no persistent Thought"
        tid = thoughts[0]["source_id"]
        time.sleep(3)

        tok = token(h.CORE)
        code, _, hdr, _ = get("/api/cognition/library", tok)
        assert code == 200, code
        expect = {"X-Cognition-Expect-Identity": hdr["X-Cognition-Identity"], "X-Cognition-Expect-Engine": hdr["X-Cognition-Engine"],
                  "X-Cognition-Expect-Link": hdr["X-Cognition-Link"]}

        # Background: what Identity logs with NO Cognition read at all (Core's own 20-second tick, the Engine).
        before = lines(); time.sleep(60); idle = len(lines()) - len(before)
        print(json.dumps({"name": "idle 60 s (no Cognition read)", "lines": idle}), flush=True)

        rows.append(measure("library read (automatic refresh; no expectation headers)", lambda: get("/api/cognition/library", tok)))
        rows.append(measure("pulse read (expectation headers, as the page sends)", lambda: get("/api/cognition/pulse", tok, expect)))
        rows.append(measure("Thought detail read (owner opens a Thought)", lambda: get("/api/cognition/thoughts/" + tid, tok, expect), 5))

        # A LOCKED Identity: what a refused read costs (the page stops automatic reads after a refusal).
        assert idn_post("/v1/lock", {})[0] in (200, 204), "lock"
        rows.append(measure("library read while Identity is LOCKED (refused)", lambda: get("/api/cognition/library", tok), 5))
        rows.append(measure("pulse read while Identity is LOCKED (refused)", lambda: get("/api/cognition/pulse", tok, expect), 5))
    finally:
        for name in list(h.n.procs):
            h.n.stop(name)

    lib, pul = rows[0], rows[1]
    per_hour_lines = 60 * lib["lines_per_read"] + 360 * pul["lines_per_read"]
    per_hour_bytes = 60 * lib["bytes_per_read"] + 360 * pul["bytes_per_read"]
    print()
    print(f"idle background: {idle} line(s) in 60 s with no Cognition read")
    for r in rows:
        print(f"{r['name']}: {r['lines_per_read']:.2f} decision line(s) per read, {r['bytes_per_read']:.0f} bytes per read, "
              f"HTTP {r['status']}, response {r['response_bytes']} bytes, median {r['latency_ms_median']:.1f} ms")
    print(f"visible tab, per hour: library 60 x {lib['lines_per_read']:.2f} + pulse 360 x {pul['lines_per_read']:.2f} = {per_hour_lines:.0f} lines "
          f"({per_hour_bytes / 1024:.0f} KiB); of which the automatic LIBRARY refresh adds {60 * lib['lines_per_read']:.0f} lines "
          f"({60 * lib['bytes_per_read'] / 1024:.0f} KiB). The pulse part exists without this slice.")
    print(f"per 24 h visible: {per_hour_lines * 24:.0f} lines ({per_hour_bytes * 24 / 1048576:.1f} MiB)")
    print(f"owner-port sentinels watched {watched}; connections seen: {sorted(set(h.SENTINEL_HITS))}")
    print("MEASUREMENT ONLY (TEST FIXTURE model; not Nemotron, not the owner's machine). Identity was not changed.")
    sys.exit(1 if h.SENTINEL_HITS else 0)


if __name__ == "__main__":
    main()
