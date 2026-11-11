#!/usr/bin/env python3
"""WP1-E bounded load measurement. Fixture-only, never touches a deployed TASS.

Build the three pinned binaries separately. The runner requires their pinned source
checkout paths, a fresh /tmp APP_HOME, and the established Stage 2 isolation harness.
No model, nudge, lesson, heartbeat or autonomous action is requested.

Offline check (no service or GitHub access):
    python3 core/test/cognition_wp1e_native.py --self-test

Native run (full invocation documented in core/docs/COGNITION_STAGE1_WP1E_LOAD_EVIDENCE_20261111.md):
    APP_HOME=/tmp/tass-wp1e-UNIQUE/app BIN=/tmp/tass-wp1e-UNIQUE/bin \
    SEED=/tmp/tass-wp1e-UNIQUE/seed WP1E_CORE_SRC=/tmp/.../core-pinned \
    WP1E_ENGINE_SRC=/tmp/.../Thought-engine WP1E_IDENTITY_SRC=/tmp/.../identity \
    CORE_BIN=tass-core-wp1e ENGINE_BIN=thought-engine-wp1e \
    IDENTITY_BIN=tass-identity-wp1e S2_PORT_BASE=39100 \
    python3 core/test/cognition_wp1e_native.py

Output: APP_HOME/wp1e-results.json and one concise JSON summary on stdout. A failed
scenario is recorded in the JSON and the process exits nonzero, without claiming PASS.
"""

import argparse
import collections
import hashlib
import json
import math
import os
from pathlib import Path
import platform
import statistics
import subprocess
import sys
import tempfile
import time
from datetime import datetime, timedelta, timezone
from urllib.error import HTTPError, URLError
from urllib.request import Request, urlopen

CORE_SHA = "2ed99269a1ef8da96319d2c776ba0651b71633ca"
ENGINE_SHA = "40d4e91e3b04196fa4e4f55a92b25789b82d8675"
IDENTITY_SHA = "cfc4079b690e2f7dc57afb9f14c6832ae38fdf54"
LIMIT = 6 << 20
HALF = LIMIT // 2
SCENARIOS = (("small", 1), ("medium", 50), ("large", 100))
NOTES_PER = 50
NOTE_CHARS = 400
CANARY = "WP1E_PRIVATE_NOTE_CANARY_NEVER_IN_LIBRARY"
DECOY = "WP1E_DECOY_ENTITY_SECRET_NEVER_IN_LIBRARY"
EPOCH = datetime(2026, 11, 11, tzinfo=timezone.utc)


def utc(t):
    return t.isoformat(timespec="seconds").replace("+00:00", "Z")


def note_text(thought, note):
    prefix = f"{CANARY} thought={thought:03d} note={note:02d} "
    return (prefix + "x" * NOTE_CHARS)[:NOTE_CHARS]


def generated_thought(eid, install_id, thought, state=None):
    if state is None:
        state = "active" if thought <= 50 else "resolved"
    created = EPOCH + timedelta(minutes=thought)
    notes, history = [], [{
        "at": utc(created), "kind": "formed", "detail": "Synthetic fixture formation",
        "to": "active", "by": "engine"
    }]
    for j in range(1, NOTES_PER + 1):
        at = utc(created + timedelta(seconds=j))
        request_id = f"wp1e-request-{thought:03d}-{j:02d}"
        txt = note_text(thought, j)
        notes.append({"at": at, "text": txt, "by": "Nemotron (heartbeat)",
                      "worker_request_id": request_id, "suggested_state": "keep"})
        # The real Think() path duplicates a developed note into history detail.
        history.append({"at": at, "kind": "developed", "detail": "Nemotron: " + txt,
                        "worker_request_id": request_id, "by": "heartbeat"})
    return {"schema_version": 1, "thought_id": f"wp1e-thought-{thought:03d}",
            "entity_id": eid, "installation_id": install_id,
            "topic": f"WP1E synthetic topic {thought:03d}",
            "question": f"What can test Thought {thought:03d} establish?",
            "question_by": "fixture", "state": state,
            "formation_reason": "WP1-E isolated deterministic fixture",
            "origin_trace_ids": [], "trace_ids": [], "notes": notes,
            "created_at": utc(created), "updated_at": notes[-1]["at"],
            "last_developed_at": notes[-1]["at"], "history": history}


def seed(directory, eid, install_id, count, *, decoy=False):
    """Replace *only* the fixture entity's Thought files, while Engine is stopped."""
    if count not in (1, 50, 100):
        raise ValueError("refusing non-WP1-E dataset size")
    folder = Path(directory) / "entities" / eid / "thoughts"
    card = folder.parent / "card.json"
    stored = json.loads(card.read_text(encoding="utf-8"))
    if stored["entity_id"] != eid or stored["installation_id"] != install_id:
        raise RuntimeError("offline seed does not match existing synthetic entity installation")
    if list(folder.glob("*.json")):
        raise RuntimeError("fixture Thought directory must be empty (new APP_HOME per run)")
    folder.mkdir(parents=True, exist_ok=True)
    manifest = {"active": 0, "sleeping": 0, "resolved": 0, "notes": 0}
    for index in range(1, count + 1):
        state = "active" if index <= min(50, count) else "resolved"
        record = generated_thought(eid, install_id, index, state)
        if decoy:
            record["thought_id"] = f"wp1e-decoy-{index:03d}"
            record["notes"][0]["text"] = DECOY
            record["history"][1]["detail"] = "fixture " + DECOY
        assert len(record["notes"]) == NOTES_PER
        assert len(record["history"]) == NOTES_PER + 1
        manifest[state] += 1
        manifest["notes"] += len(record["notes"])
        (folder / (record["thought_id"] + ".json")).write_text(
            json.dumps(record, sort_keys=True, indent=2) + "\n", encoding="utf-8")
    assert manifest["active"] + manifest["sleeping"] <= 50
    return manifest


def self_test():
    with tempfile.TemporaryDirectory(prefix="wp1e-offline-test-") as root:
        for name, n in SCENARIOS:
            eid, iid = "ent-synthetic", "install-synthetic"
            base = Path(root) / name / "entities" / eid
            (base / "thoughts").mkdir(parents=True)
            (base / "card.json").write_text(json.dumps({"entity_id": eid,
                                                          "installation_id": iid}), encoding="utf-8")
            got = seed(Path(root) / name, eid, iid, n)
            files = sorted((base / "thoughts").glob("*.json"))
            assert len(files) == n and got["notes"] == n * NOTES_PER
            assert got["active"] == min(50, n) and got["resolved"] == max(0, n - 50)
            assert all(len(note_text(i, j)) == NOTE_CHARS for i in (1, n) for j in (1, NOTES_PER))
            for p in files:
                record = json.loads(p.read_text(encoding="utf-8"))
                assert record["thought_id"] + ".json" == p.name
                assert len(record["notes"]) == 50 and len(record["history"]) == 51
        print("PASS: deterministic offline seed 1/50/100 Thoughts, 50 Notes each, <=50 open")


def ensure_disposable():
    raw = os.environ.get("APP_HOME", "")
    if not raw or not Path(raw).is_absolute():
        raise RuntimeError("APP_HOME must be an absolute disposable /tmp directory")
    p = Path(raw)
    if any(x.is_symlink() for x in [p, *p.parents]):
        raise RuntimeError("APP_HOME cannot contain symlinks")
    resolved = p.resolve()
    if not resolved.is_relative_to("/tmp") or len(resolved.parts) < 4 or "wp1e" not in str(resolved).lower():
        raise RuntimeError("APP_HOME must be nested under /tmp in a uniquely named wp1e test directory")
    if p.exists() and any(p.iterdir()):
        raise RuntimeError("APP_HOME must be new and empty for all three scenarios")
    for label, expected in (("WP1E_CORE_SRC", CORE_SHA), ("WP1E_ENGINE_SRC", ENGINE_SHA),
                            ("WP1E_IDENTITY_SRC", IDENTITY_SHA)):
        src = os.environ.get(label, "")
        if not src or not Path(src).is_dir():
            raise RuntimeError(f"{label} is required and must be a checkout")
        got = subprocess.check_output(["git", "-C", src, "rev-parse", "HEAD"], text=True).strip()
        if got != expected:
            raise RuntimeError(f"{label}: wrong revision {got}, expected {expected}")


def request_bytes(url, headers=None):
    req = Request(url, method="GET", headers=headers or {})
    start = time.perf_counter()
    try:
        with urlopen(req, timeout=30) as r:
            return {"status": r.status, "body": r.read(), "headers": dict(r.headers),
                    "ms": (time.perf_counter() - start) * 1000, "error": None}
    except HTTPError as e:
        return {"status": e.code, "body": e.read(), "headers": dict(e.headers),
                "ms": (time.perf_counter() - start) * 1000, "error": None}
    except (URLError, TimeoutError, OSError) as e:
        return {"status": "ERROR", "body": b"", "headers": {},
                "ms": (time.perf_counter() - start) * 1000, "error": str(e)}


def log_snapshot(log):
    try:
        content = Path(log).read_bytes()
    except FileNotFoundError:
        content = b""
    return len(content.splitlines()), len(content)


def sample(label, count, fn, log):
    before = log_snapshot(log)
    calls = [fn() for _ in range(count)]
    after = log_snapshot(log)
    latencies = sorted(x["ms"] for x in calls)
    statuses = dict(collections.Counter(str(x["status"]) for x in calls))
    delta_lines, delta_bytes = after[0] - before[0], after[1] - before[1]
    return {"name": label, "requests": count, "status": statuses,
            "failure_count": sum(x["status"] != 200 for x in calls),
            "errors": [x["error"] for x in calls if x["error"]],
            "response_bytes_min": min(len(x["body"]) for x in calls),
            "response_bytes_max": max(len(x["body"]) for x in calls),
            "latency_ms_median": round(statistics.median(latencies), 3),
            "latency_ms_p95_nearest_rank": round(latencies[math.ceil(.95 * count) - 1], 3),
            "latency_ms_samples_ordered": [round(x["ms"], 3) for x in calls],
            "identity_lines_added": delta_lines, "identity_bytes_added": delta_bytes,
            "identity_lines_per_request": delta_lines / count,
            "identity_bytes_per_request": delta_bytes / count}


def process_rss(pid):
    try:
        for line in Path(f"/proc/{pid}/status").read_text().splitlines():
            if line.startswith("VmRSS:"):
                return int(line.split()[1]) * 1024
    except (OSError, ValueError, IndexError):
        return None
    return None


def binary_hash(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()


def run():
    ensure_disposable()   # BEFORE importing the shared harness (its import runs preflight)
    sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
    import nudge_process_native as h  # noqa: E402 (reuses audited isolation)
    from acceptance import http, idn_post, token

    reads = int(os.environ.get("WP1E_READS", "30"))
    warmup = int(os.environ.get("WP1E_WARMUP", "3"))
    if not 30 <= reads <= 100 or not 1 <= warmup <= 10:
        raise RuntimeError("require 30..100 measured reads and 1..10 warmups")
    h.n.prepare()
    watched = h.start_sentinels()
    if watched:  # positive-control the owner-port isolation sentinel
        h.socket.create_connection(("127.0.0.1", watched[0]), timeout=2).close()
        time.sleep(.2)
        if h.SENTINEL_HITS != [watched[0]]:
            raise RuntimeError("owner-port sentinel positive control did not register")
        h.SENTINEL_HITS.clear()
    result = {"baseline": {"core": CORE_SHA, "engine": ENGINE_SHA, "identity": IDENTITY_SHA},
              "environment": {"platform": platform.platform(), "python": platform.python_version(),
                              "port_base": h.BASE, "model": "fixture-model (no requests expected)",
                              "samples_per_endpoint": reads, "warmups_per_endpoint": warmup,
                              "binary_sha256": {p: binary_hash(str(Path(h.BIN) / name)) for p, name in
                                                (("core", h.CORE_BIN), ("engine", h.ENGINE_BIN),
                                                 ("identity", h.IDENTITY_BIN))}},
              "limit_bytes": LIMIT, "proposal_trigger_bytes_strict_gt": HALF,
              "scenarios": [], "isolation": {}, "hidden_tab_audit": "NOT RUN: no live browser used"}
    out_path = Path(h.APP) / "wp1e-results.json"
    pass_run = False
    try:
        h.start("model", ["python3", f"{h.HERE}/fixture-model/fake_llama.py", str(h.P_MODEL)],
                {"FIXTURE_DELAY": "1"}, h.MODEL + "/health")
        h.start("identity", [f"{h.BIN}/{h.IDENTITY_BIN}"],
                {"IDENTITY_DATA": f"{h.APP}/data/identity", "IDENTITY_LISTEN": f"127.0.0.1:{h.P_IDN}",
                 "IDENTITY_SERVICE_KEYS": f"{h.APP}/secrets/service-keys",
                 "IDENTITY_CORE_URL": h.CORE + "/"}, h.IDN + "/v1/health")
        h.start_engine()
        h.start_proxy()
        h.start_core()
        code, owner = idn_post("/v1/entity", {"name": "WP1E-owner", "passphrase": h.n.PASS})
        if code != 201 or owner.get("state") != "UNLOCKED":
            raise RuntimeError(f"synthetic Identity creation refused ({code}): {owner}")
        hdr = {"Authorization": "Bearer " + h.ekey()}
        code, main = http("POST", h.ENG + "/api/identities",
                          {"display_name": "WP1E-owner", "entity_type": "ai_companion"}, hdr)
        if code != 201:
            raise RuntimeError(f"synthetic Engine entity creation refused ({code}): {main}")
        code, other = http("POST", h.ENG + "/api/identities",
                           {"display_name": "WP1E-decoy", "entity_type": "ai_companion"}, hdr)
        if code != 201:
            raise RuntimeError(f"synthetic decoy creation refused ({code}): {other}")
        eid, iid = main["card"]["entity_id"], main["card"]["installation_id"]
        other_eid, other_iid = other["card"]["entity_id"], other["card"]["installation_id"]
        code, linked = http("POST", h.CORE + "/api/cognition/link",
                             {"engine_entity_id": eid}, {"X-TASS-Token": token(h.CORE)})
        if code != 200:
            raise RuntimeError(f"synthetic owner Engine link refused ({code}): {linked}")
        # Engine must be OFF for every offline fixture-file write. Per-scenario engine
        # state is reloaded on restart; no direct Engine state mutation is introduced.
        h.n.stop("engine")
        seed(Path(h.APP) / "data/engine", other_eid, other_iid, 1, decoy=True)
        tok = token(h.CORE)
        calls_before = h.fixture().get("requests", 0)
        log = Path(h.APP) / "data/identity/decisions.jsonl"
        for name, count in SCENARIOS:
            data_dir = Path(h.APP) / "data/engine"
            # Remove only the previous fixture Thought files while Engine is stopped.
            if name != "small":
                folder = data_dir / "entities" / eid / "thoughts"
                for p in folder.glob("wp1e-thought-*.json"):
                    p.unlink()
                if list(folder.glob("*.json")):
                    raise RuntimeError("unexpected non-WP1E Thought file; aborting")
            manifest = seed(data_dir, eid, iid, count)
            h.start_engine()
            engine_pid = h.n.procs["engine"].pid
            # Confirm the real Engine reloaded every seeded Thought and Note.
            upstream = request_bytes(h.ENG + f"/api/entities/{eid}", hdr)
            if upstream["status"] != 200:
                raise RuntimeError(f"Engine raw entity read failed in {name}: {upstream['status']}")
            payload = json.loads(upstream["body"])
            loaded = payload.get("thoughts") or []
            if len(loaded) != count or any(len(t.get("notes") or []) != NOTES_PER for t in loaded):
                raise RuntimeError(f"Engine failed to load exact synthetic {name} count")
            if collections.Counter(t["state"] for t in loaded) != collections.Counter(
                    {k: v for k, v in manifest.items() if k in ("active", "sleeping", "resolved") and v}):
                raise RuntimeError("Engine state distribution mismatches synthetic fixture")
            current = request_bytes(h.CORE + "/api/cognition/library", {"X-TASS-Token": tok})
            if current["status"] == 200:
                cards = json.loads(current["body"]).get("thoughts", [])
                if len(cards) != count or any(c.get("notes_count") != NOTES_PER for c in cards):
                    raise RuntimeError(f"Core library lost a seeded Thought or Note count in {name}")
            for _ in range(warmup):
                request_bytes(h.CORE + "/api/cognition/library", {"X-TASS-Token": tok})
            lib = sample("library", reads, lambda: request_bytes(
                h.CORE + "/api/cognition/library", {"X-TASS-Token": tok}), log)
            header = {"X-TASS-Token": tok}
            if current["status"] == 200:
                for key, value in (("X-Cognition-Expect-Identity", "X-Cognition-Identity"),
                                   ("X-Cognition-Expect-Engine", "X-Cognition-Engine"),
                                   ("X-Cognition-Expect-Link", "X-Cognition-Link")):
                    header[key] = current["headers"][value]
            pulse = sample("pulse", reads, lambda: request_bytes(
                h.CORE + "/api/cognition/pulse", header), log)
            # Privacy check on raw library JSON (not the upstream Engine payload,
            # where full Notes are legitimately present). No text from the decoy.
            post_library = request_bytes(h.CORE + "/api/cognition/library", {"X-TASS-Token": tok})
            library_body = post_library["body"]
            private_absent = CANARY.encode() not in library_body and DECOY.encode() not in library_body
            decoy_absent = b"wp1e-decoy-001" not in library_body
            all_get_only = not h.PROXIED  # proxy only tracks nudge POST; no others sent by this runner
            status = {"scenario": name, "thoughts": count, "notes": count * NOTES_PER,
                      "state_distribution": manifest, "engine_http": upstream["status"],
                      "engine_bytes": len(upstream["body"]),
                      "engine_limit_pct": round(len(upstream["body"]) / LIMIT * 100, 4),
                      "over_half_limit": len(upstream["body"]) > HALF,
                      "engine_over_limit": len(upstream["body"]) > LIMIT,
                      "library_initial_http": current["status"],
                      "library_initial_bytes": len(current["body"]),
                      "library": lib, "pulse": pulse,
                      "private_note_absent_from_cards": private_absent,
                      "decoy_thought_absent_from_cards": decoy_absent,
                      "no_nudge_post": all_get_only,
                      "rss_bytes": {"core": process_rss(h.n.procs["core"].pid),
                                    "engine": process_rss(engine_pid),
                                    "identity": process_rss(h.n.procs["identity"].pid)}}
            status["result"] = "PASS" if (current["status"] == 200 and lib["failure_count"] == 0
                                                    and pulse["failure_count"] == 0 and private_absent
                                                    and decoy_absent and all_get_only) else "FAIL"
            result["scenarios"].append(status)
            print(json.dumps({k: status[k] for k in ("scenario", "engine_bytes", "engine_limit_pct",
                                                    "library_initial_bytes", "result")}), flush=True)
            h.n.stop("engine")
        calls_after = h.fixture().get("requests", 0)
        result["isolation"] = {"fixture_model_calls_before": calls_before,
                               "fixture_model_calls_after": calls_after,
                               "no_model_execution_during_load": calls_after == calls_before,
                               "owner_port_sentinel_hits": sorted(set(h.SENTINEL_HITS)),
                               "no_owner_port_access": not h.SENTINEL_HITS}
        pass_run = (len(result["scenarios"]) == 3 and
                    all(s["result"] == "PASS" for s in result["scenarios"]) and
                    result["isolation"]["no_model_execution_during_load"] and
                    result["isolation"]["no_owner_port_access"])
    except Exception as exc:
        result["execution_error"] = str(exc)
        raise
    finally:
        for proc in list(h.n.procs):
            h.n.stop(proc)
        result["completed"] = pass_run
        out_path.write_text(json.dumps(result, indent=2, sort_keys=True) + "\n", encoding="utf-8")
        print(f"WP1-E raw measurements written to: {out_path}", flush=True)
    return 0 if pass_run else 1


def main():
    parser = argparse.ArgumentParser(description="Isolated Stage 1 WP1-E native load verification")
    parser.add_argument("--self-test", action="store_true", help="offline dataset schema/determinism test")
    args = parser.parse_args()
    if args.self_test:
        self_test()
        return 0
    return run()


if __name__ == "__main__":
    try:
        sys.exit(main())
    except Exception as exc:
        print(f"WP1-E BLOCKED/FAIL: {exc}", file=sys.stderr)
        sys.exit(1)
