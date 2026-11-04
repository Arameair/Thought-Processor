#!/usr/bin/env python3
"""Read-only-first independent acceptance checks for the Go TASS stack.

Uses the live APIs and append-only Engine records. Never downloads models,
changes Identity, starts/stops Docker, writes GitHub, or prints API keys.
The only mutating probe (owner chat) requires explicit --send.
"""
import argparse
import datetime as dt
import hashlib
from decimal import Decimal
import json
import pathlib
import sys
import time
import urllib.error
import urllib.request
import uuid

SUCCESS, FAILURE, NOT_RUN = 0, 1, 2


def check(name, passed, why):
    return (name, bool(passed), why)


def render(checks):
    failures = 0
    for name, passed, why in checks:
        print(f"{'PASS' if passed else 'FAIL'} | {name} | {why}")
        failures += not passed
    return FAILURE if failures else SUCCESS


def instant(value):
    if not isinstance(value, str) or not value:
        raise ValueError("missing timestamp")
    parsed = dt.datetime.fromisoformat(value.replace("Z", "+00:00"))
    if parsed.tzinfo is None:
        raise ValueError("timestamp has no timezone")
    return parsed.astimezone(dt.timezone.utc)


def json_lines(path):
    path = pathlib.Path(path).expanduser()
    rows = []
    with path.open("r", encoding="utf-8") as f:
        for n, line in enumerate(f, 1):
            if line.strip():
                try:
                    rows.append(json.loads(line))
                except json.JSONDecodeError as exc:
                    raise ValueError(f"{path.name} line {n}: invalid JSON") from exc
    return rows


def prediction_hash(cycle_id, action, prediction):
    """Recompute Engine cycles.go's unkeyed SHA-256 evidence checksum.

    This checks accidental corruption, not authenticity against an attacker
    with write access to the journal; no signed evidence exists yet.
    """
    val = prediction["value"]
    number = format(Decimal(str(val)), "f")
    if "." in number:
        number = number.rstrip("0").rstrip(".")
    payload = "|".join((
        cycle_id, action, prediction.get("text", ""), number,
        prediction["locked_at"],
    ))
    return hashlib.sha256(payload.encode("utf-8")).hexdigest()


def cycle_checks(rows, cycle_id):
    """A strictly ordered append-only cycle is necessary, not sufficient,
    evidence for executed action and real inference."""
    entries = [e for e in rows if e.get("cycle_id") == cycle_id]
    steps = [e.get("step") for e in entries]
    required = ["received", "spark", "proposal", "outcome", "evaluation"]
    result = [check("cycle ledger has all five stages",
                    all(x in steps for x in required),
                    f"cycle={cycle_id}, stages={steps}")]
    if not all(x in steps for x in required):
        return result
    offsets = [steps.index(x) for x in required]
    result.append(check("durable record order", offsets == sorted(offsets),
                        "received < spark < proposal < outcome < evaluation"))
    data = {x: entries[steps.index(x)].get("data", {}) for x in required}
    spark, prop, out, ev = (data[x] for x in required[1:])
    prediction = prop.get("prediction") or {}
    result.append(check("real Spark receipt linked to event",
                        spark.get("status") == "done"
                        and spark.get("event_id") == cycle_id
                        and bool(spark.get("spark_id"))
                        and bool(spark.get("model_request_ids")),
                        f"spark status={spark.get('status')}; request IDs present={bool(spark.get('model_request_ids'))}"))
    result.append(check("Trace retained", bool(spark.get("trace_id")),
                        "a low-rated Spark with no Trace does not satisfy this full-cycle acceptance"))
    result.append(check("model plan structurally readable",
                        prop.get("format") == "READ" and bool(prop.get("model_request_id"))
                        and bool(prop.get("reply"))
                        and "TEST FIXTURE" not in prop.get("reply", ""),
                        f"format={prop.get('format')}; model request ID present={bool(prop.get('model_request_id'))}"))
    try:
        checksum_ok = (prediction.get("sha256") ==
                       prediction_hash(cycle_id, prop.get("action", ""), prediction))
    except (KeyError, ValueError, TypeError):
        checksum_ok = False
    result.append(check("prediction committed with matching SHA-256",
                        prediction.get("state") == "LOCKED"
                        and prop.get("action") in
                            ("tass_sections", "tass_words", "tass_memory_lines")
                        and prediction.get("value") is not None and checksum_ok,
                        f"prediction state={prediction.get('state')}; checksum valid={checksum_ok}"))
    try:
        lock = instant(prediction.get("locked_at"))
        executed = instant(out.get("executed_at"))
        received = instant(out.get("received_at"))
        evaluated = instant(ev.get("at"))
        result.append(check("prediction locked BEFORE action and evaluation",
                            lock < executed <= received <= evaluated,
                            "locked_at < executed_at <= received_at <= evaluation.at"))
        proposal_record = instant(entries[steps.index("proposal")]["at"])
        outcome_record = instant(entries[steps.index("outcome")]["at"])
        result.append(check("append-only prediction pre-dates outcome",
                            proposal_record <= outcome_record,
                            "proposal journal timestamp <= outcome journal timestamp"))
    except (TypeError, ValueError, KeyError) as exc:
        result.append(check("timestamps complete and ordered", False, str(exc)))
    result.append(check("authorized read-only executor receipt",
                        out.get("status") == "EXECUTED"
                        and "action.read_tass" in out.get("authorized", "")
                        and out.get("value") is not None
                        and out.get("executor", "").startswith("thought-processor"),
                        f"status={out.get('status')}; executor={out.get('executor')}"))
    predicted, measured = prediction.get("value"), out.get("value")
    try:
        correct = abs(float(predicted) - float(measured)) < 1e-9
        expected_forecast = "PREDICTION_CORRECT" if correct else "PREDICTION_INCORRECT"
    except (TypeError, ValueError):
        expected_forecast = None
    result.append(check("deterministic evaluation matches measurement",
                        expected_forecast is not None
                        and ev.get("forecast") == expected_forecast
                        and ev.get("goal") == "GOAL_MET"
                        and bool(ev.get("evaluator")),
                        f"expected={expected_forecast}; forecast={ev.get('forecast')}; goal={ev.get('goal')}"))
    # A pass here does not verify the model was Nemotron. That requires real
    # owner hardware and an independent /v1/models/runtime identity check.
    return result


def health_checks(rows, count=10, interval=20.0, tolerance=6.0):
    # This is a HEALTH ping. It cannot be promoted to a thought pulse result.
    last = rows[-count:]
    results = [check("health-pulse sample count", len(last) == count,
                     f"found {len(last)} of {count} required health records")]
    if len(last) < count:
        return results
    try:
        ts = [instant(x["at"]) for x in last]
    except (KeyError, ValueError, TypeError) as exc:
        return results + [check("health timestamps valid", False, str(exc))]
    intervals = [(b-a).total_seconds() for a, b in zip(ts, ts[1:])]
    results.append(check("10 consecutive 20-second health samples",
                         all(interval-tolerance <= n <= interval+tolerance for n in intervals),
                         "interval seconds=" + ",".join(f"{x:.1f}" for x in intervals)))
    results.append(check("health records preserve model signal",
                         all(isinstance(x.get("model_ok"), bool) for x in last),
                         "boolean model_ok on each record; no inference is implied"))
    return results


def thought_pulse_checks(rows):
    """PROPOSED future normalized evidence contract, NOT an implemented API.
    kind=thought_tick, pulse_id, at, mode, state, active_request_id,
    model_started_ids, action_executed_ids. kind=control with command;
    kind=process_start with at. No fake state derivation from health records.
    """
    ticks = [x for x in rows if x.get("kind") == "thought_tick"]
    auto = [x for x in ticks if x.get("mode") == "RUNNING"]
    controls = [x.get("command") for x in rows if x.get("kind") == "control"]
    ids = [x.get("pulse_id") for x in ticks]
    results = [check("10 coordinated automatic thought ticks", len(auto) >= 10,
                     f"found {len(auto)}; health pulse does not count"),
               check("unique durable pulse IDs",
                     len(ids) == len(set(ids)) and all(ids),
                     f"{len(ids)} tick IDs, {len(set(ids))} unique"),
               check("explicit Pause / Step Once / Resume receipts",
                     all(x in controls for x in ("pause", "step_once", "resume")),
                     f"controls={controls}"),
               check("Step Once emits exactly a paused step",
                     sum(x.get("mode") == "PAUSED" and x.get("reason") == "STEP_ONCE"
                         for x in ticks) == 1,
                     "one STEP_ONCE tick while PAUSED"),
               check("per-capability state not inferred from health",
                     all(x.get("state") in ("READY", "WORKING", "BUSY", "BLOCKED")
                         for x in ticks),
                     "explicit READY/WORKING/BUSY/BLOCKED")]
    if len(auto) >= 10:
        try:
            times = [instant(x["at"]) for x in auto[-10:]]
            gap = [(b-a).total_seconds() for a, b in zip(times, times[1:])]
            results.append(check("20-second coordinated tick spacing",
                                 all(14 <= v <= 26 for v in gap),
                                 "interval seconds=" + ",".join(f"{x:.1f}" for x in gap)))
        except (KeyError, ValueError, TypeError) as exc:
            results.append(check("tick timestamp format", False, str(exc)))
    started = [i for x in ticks for i in x.get("model_started_ids", [])]
    actions = [i for x in ticks for i in x.get("action_executed_ids", [])]
    active = [x.get("active_request_id") for x in ticks if x.get("active_request_id")]
    repeated = any(active.count(x) >= 3 for x in set(active))
    results.append(check("slow inference spans >=3 ticks",
                         repeated and all(started.count(x) == 1 for x in set(started)),
                         f"long-lived work={repeated}; duplicate starts={len(started)!=len(set(started))}"))
    results.append(check("no action duplicated across pulses",
                         len(actions) == len(set(actions)),
                         f"{len(actions)} actions, {len(set(actions))} unique IDs"))
    starts = [x for x in rows if x.get("kind") == "process_start"]
    results.append(check("restart evidence linked to durable pulse stream",
                         len(starts) >= 2, f"process starts recorded={len(starts)}"))
    return results


def request_json(base, path, key="", method="GET", body=None, timeout=5):
    url = base.rstrip("/") + path
    headers = {"Accept": "application/json"}
    if key:
        headers["X-TASS-Token" if path.startswith("/api/say") or
                (base.rstrip("/").endswith(":8795") and path == "/api/status")
                else "Authorization"] = (
                    key if base.rstrip("/").endswith(":8795") else "Bearer " + key)
    payload = None
    if body is not None:
        payload = json.dumps(body).encode("utf-8")
        headers["Content-Type"] = "application/json"
    req = urllib.request.Request(url, data=payload, headers=headers, method=method)
    try:
        with urllib.request.urlopen(req, timeout=timeout) as resp:
            raw = resp.read(4 << 20)
            return resp.status, json.loads(raw) if raw.strip() else {}
    except urllib.error.HTTPError as exc:
        # Do not print bodies: error text may contain sensitive data.
        return exc.code, {}


def key_from(path):
    p = pathlib.Path(path).expanduser()
    try:
        return p.read_text(encoding="utf-8").strip()
    except FileNotFoundError:
        return ""


def safe_call(fn):
    try:
        return fn()
    except (OSError, ValueError, KeyError, TimeoutError) as exc:
        print(f"NOT RUN | prerequisite unavailable | {type(exc).__name__}: {str(exc)[:160]}")
        return NOT_RUN


def baseline(args):
    core_key, eng_key = key_from(args.core_key), key_from(args.engine_key)
    if not core_key or not eng_key:
        print("NOT RUN | local API key file missing. Do not merge Identity permissions to satisfy a probe.")
        return NOT_RUN
    ccode, core = request_json(args.core_url, "/api/status", core_key)
    ecode, eng = request_json(args.engine_url, "/api/model")
    scode, estate = request_json(args.engine_url, "/api/status")
    mcode, models = request_json(args.model_url, "/v1/models", timeout=8)
    print(f"HTTP | core status={ccode}, engine model={ecode}, engine state={scode}, local model catalog={mcode}")
    if mcode == 200:
        print("MODEL IDs | " + ", ".join(str(x.get("id", "(unknown)"))
                for x in models.get("data", [])[:4]))
    checks = [
        check("core authenticates status", ccode == 200, "token from local file"),
        check("Engine model health endpoint", ecode == 200, "health is not inference"),
        check("Engine state readable", scode == 200, "read-only"),
        check("Identity permit reported", core.get("policy_valid_now") is True,
              "prototype policy: denied is BLOCKED, not a model failure"),
        check("local model catalog available", mcode == 200 and bool(models.get("data")),
              "inspect model ID; status alone does not establish Nemotron inference"),
    ]
    if ccode == 200:
        engs = core.get("engine", {})
        print(f"STATE | core_busy={core.get('busy')}, engine_busy={engs.get('busy')}, "
              f"model_ok={(engs.get('last_pulse') or {}).get('model_ok')}, "
              f"github_online={core.get('github_online')}; sync problem suppressed")
    return render(checks)


def verify_cycle(args):
    rows = json_lines(args.cycle_log)
    result = render(cycle_checks(rows, args.cycle_id))
    if result:
        return result
    key = key_from(args.engine_key)
    if not key:
        print("NOT RUN | live Engine correlation (no API key); journal-only checks above PASSED")
        return NOT_RUN
    code, data = request_json(args.engine_url, "/api/cycles/" + args.cycle_id, key)
    return render([check("live Engine returns same evaluated cycle",
                         code == 200 and data.get("cycle_id") == args.cycle_id
                         and data.get("state") == "evaluated",
                         f"HTTP {code}; state={data.get('state')}")])


def inspect_security(args):
    code, _ = request_json(args.core_url, "/api/status")
    ecode, _ = request_json(args.engine_url, "/api/cycles")
    public, _ = request_json(args.engine_url, "/api/model")
    print("NO WRITE attempted: Engine legacy POST routes are source-audited, not probed against a running service.")
    return render([
        check("core status denies missing key", code == 401, f"HTTP {code}"),
        check("Engine cycle log denies missing key", ecode == 401, f"HTTP {ecode}"),
        check("Engine public model status is observable", public == 200,
              f"HTTP {public}; note request IDs/metadata exposure"),
    ])


def probe_chat(args):
    if not args.send:
        print("NOT RUN | chat sends a real local request; opt in with --send")
        return NOT_RUN
    key = key_from(args.core_key)
    if not key:
        print("NOT RUN | missing local core API key")
        return NOT_RUN
    stcode, st = request_json(args.core_url, "/api/status", key)
    if stcode != 200:
        print(f"FAIL | cannot inspect current core: HTTP {stcode}")
        return FAILURE
    if args.require_busy and st.get("busy") is not True:
        print("NOT RUN | --require-busy needs an already in-flight Go core cycle")
        return NOT_RUN
    marker = "acceptance-" + uuid.uuid4().hex[:10]
    message = f"{marker} Please reply with one short sentence. Do not propose an action."
    code, _ = request_json(args.core_url, "/api/say", key, method="POST", body={"text": message})
    if code != 202:
        print(f"FAIL | chat ingress HTTP {code}; no cycle accepted")
        return FAILURE
    print(f"ACCEPTED? | HTTP 202 for {marker}; not proof of inference or delivery")
    cutoff = time.monotonic() + args.timeout
    while time.monotonic() < cutoff:
        _, now = request_json(args.core_url, "/api/status", key)
        matches = [x for x in now.get("cycles", []) if x.get("request") == message]
        if matches:
            c = matches[0]
            cid = c.get("cycle_id")
            prop = c.get("proposal") or {}
            if c.get("state") == "evaluated" and prop:
                good = (prop.get("format") == "READ"
                        and bool(prop.get("reply"))
                        and bool(prop.get("model_request_id"))
                        and "TEST FIXTURE" not in prop.get("reply", ""))
                if not good:
                    print(f"FAIL | cycle {cid} evaluated without a readable, non-fixture model reply")
                    return FAILURE
                print(f"PASS | Go transport: correlated completed cycle {cid}, readable reply")
                print("NOT RUN | model/GPU identity and Go Full Auto/Autonomy (separate evidence required)")
                return SUCCESS
            if c.get("state") in ("denied", "failed"):
                print(f"FAIL | cycle {cid}: {c.get('state')}; {str(c.get('problem'))[:100]}")
                return FAILURE
        time.sleep(2)
    print("FAIL | HTTP 202 did not yield a correlated completed cycle within deadline")
    return FAILURE


def main():
    p = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    p.add_argument("--core-url", default="http://127.0.0.1:8795")
    p.add_argument("--engine-url", default="http://127.0.0.1:7798")
    p.add_argument("--model-url", default="http://127.0.0.1:18083")
    p.add_argument("--core-key", default="~/.tass/api-key")
    p.add_argument("--engine-key", default="~/.thought-engine/tass/api-key")
    p.add_argument("--cycle-log", default="~/.thought-engine/tass/cycles.jsonl")
    p.add_argument("--health-log", default="~/.thought-engine/tass/pulse.jsonl")
    s = p.add_subparsers(dest="command", required=True)
    s.add_parser("baseline", help="read-only service, policy and model catalogue check")
    cp = s.add_parser("cycle", help="verify one real, recorded cycle against live Engine")
    cp.add_argument("cycle_id")
    hp = s.add_parser("health-pulses", help="verify health pulse timing ONLY, not thoughts")
    hp.add_argument("--count", type=int, default=10)
    tp = s.add_parser("thought-pulses", help="future coordinated pulse evidence contract")
    tp.add_argument("log", help="normalized thought-pulse JSONL (NOT current health pulse)")
    s.add_parser("security", help="credential tests using GET only; no writes")
    ch = s.add_parser("chat", help="send one harmless owner message, opt-in only")
    ch.add_argument("--send", action="store_true")
    ch.add_argument("--require-busy", action="store_true")
    ch.add_argument("--timeout", type=int, default=120)
    args = p.parse_args()
    if args.command == "baseline":
        return safe_call(lambda: baseline(args))
    if args.command == "cycle":
        return safe_call(lambda: verify_cycle(args))
    if args.command == "health-pulses":
        return safe_call(lambda: render(health_checks(json_lines(args.health_log), args.count)))
    if args.command == "thought-pulses":
        return safe_call(lambda: render(thought_pulse_checks(json_lines(args.log))))
    if args.command == "security":
        return safe_call(lambda: inspect_security(args))
    if args.command == "chat":
        return safe_call(lambda: probe_chat(args))
    return NOT_RUN


if __name__ == "__main__":
    sys.exit(main())
