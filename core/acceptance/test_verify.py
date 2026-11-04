"""TEST FIXTURES ONLY: validate the acceptance verifier against good and bad
records. Passing these tests does not prove live Nemotron or any Go deployment."""
import datetime as dt
import hashlib
import tempfile
import unittest
from pathlib import Path
from types import SimpleNamespace
from unittest.mock import patch

import verify


def stamp(seconds):
    return (dt.datetime(2026, 11, 4, tzinfo=dt.timezone.utc)
            + dt.timedelta(seconds=seconds)).isoformat().replace("+00:00", "Z")


def fake_cycle():
    cid = "tp_independent_fixture_1"
    cases = [
        ("received", {"cycle_id": cid, "request": "count sections"}),
        ("spark", {"status": "done", "event_id": cid, "spark_id": "sp1",
                   "model_request_ids": ["mr1"], "trace_id": "tr1"}),
        ("proposal", {"format": "READ", "model_request_id": cid + ":plan",
                      "reply": "There are four sections.", "action": "tass_sections",
                      "prediction": {"state": "LOCKED", "locked_at": stamp(2),
                                     "sha256": "a"*64, "value": 4}}),
        ("outcome", {"status": "EXECUTED", "executed_at": stamp(3), "received_at": stamp(4),
                     "value": 4, "authorized": "Identity grant action.read_tass allowed",
                     "executor": "thought-processor (read-only)"}),
        ("evaluation", {"forecast": "PREDICTION_CORRECT", "goal": "GOAL_MET",
                        "evaluator": "deterministic", "at": stamp(5)}),
    ]
    rows = [{"cycle_id": cid, "step": step, "at": stamp(i+1), "data": data}
            for i, (step, data) in enumerate(cases)]
    prop = rows[2]["data"]
    prop["prediction"]["text"] = "4"
    prop["prediction"]["sha256"] = verify.prediction_hash(
        cid, prop["action"], prop["prediction"])
    return cid, rows


class CycleEvidenceTests(unittest.TestCase):
    def test_complete_fixture_passes_ledger_checks(self):
        cid, rows = fake_cycle()
        self.assertTrue(all(passed for _, passed, _ in verify.cycle_checks(rows, cid)))

    def test_prediction_digest_tampering_is_rejected(self):
        cid, rows = fake_cycle()
        rows[2]["data"]["prediction"]["sha256"] = "0"*64
        self.assertFalse(all(p for _, p, _ in verify.cycle_checks(rows, cid)))

    def test_forged_evaluation_result_is_rejected(self):
        cid, rows = fake_cycle()
        rows[4]["data"]["forecast"] = "PREDICTION_INCORRECT"
        self.assertFalse(all(p for _, p, _ in verify.cycle_checks(rows, cid)))

    def test_local_document_action_is_independently_measured(self):
        cid, rows = fake_cycle()
        with tempfile.TemporaryDirectory() as d:
            path = Path(d, "tass.md")
            doc = "# TASS\\n## A\\n## B\\n## C\\n## D\\n".replace("\\\\n", "\\n")
            path.write_text(doc)
            digest = hashlib.sha256(doc.strip().encode()).hexdigest()[:12]
            rows[3]["data"]["detail"] = f"tass_sections on tass.md ({digest})"
            self.assertTrue(all(p for _, p, _ in
                verify.current_document_action_checks(rows, cid, path)))
            path.write_text("# TASS\\n## A\\n".replace("\\\\n", "\\n"))
            self.assertFalse(all(p for _, p, _ in
                verify.current_document_action_checks(rows, cid, path)))

    def test_missing_evaluation_fails_closed(self):
        cid, rows = fake_cycle()
        self.assertFalse(all(p for _, p, _ in verify.cycle_checks(rows[:-1], cid)))

    def test_prediction_after_action_is_rejected(self):
        cid, rows = fake_cycle()
        rows[2]["data"]["prediction"]["locked_at"] = stamp(6)
        self.assertFalse(all(p for _, p, _ in verify.cycle_checks(rows, cid)))

    def test_unreadable_model_reply_does_not_count_as_success(self):
        cid, rows = fake_cycle()
        rows[2]["data"]["format"] = "UNREADABLE"
        self.assertFalse(all(p for _, p, _ in verify.cycle_checks(rows, cid)))

    def test_no_trace_is_not_claimed_as_complete_cycle(self):
        cid, rows = fake_cycle()
        rows[1]["data"].pop("trace_id")
        self.assertFalse(all(p for _, p, _ in verify.cycle_checks(rows, cid)))

    def test_denial_is_not_accepted_as_executed(self):
        cid, rows = fake_cycle()
        rows[3]["data"]["status"] = "NOT_AUTHORIZED"
        self.assertFalse(all(p for _, p, _ in verify.cycle_checks(rows, cid)))

    def test_corrupt_jsonl_is_not_ignored(self):
        with tempfile.TemporaryDirectory() as d:
            p = Path(d, "journal.jsonl")
            p.write_text('{"good":true}\ninvalid\n')
            with self.assertRaises(ValueError):
                verify.json_lines(p)


class PulseEvidenceTests(unittest.TestCase):
    def test_ten_realistic_health_ticks_pass_only_health(self):
        rows = [{"at": stamp(i*20), "model_ok": True, "busy": False}
                for i in range(10)]
        self.assertTrue(all(p for _, p, _ in verify.health_checks(rows)))
        self.assertFalse(all(p for _, p, _ in verify.thought_pulse_checks(rows)))

    def test_duplicated_timestamp_or_gap_fails_health(self):
        rows = [{"at": stamp(i*20), "model_ok": True} for i in range(10)]
        rows[7]["at"] = rows[6]["at"]
        self.assertFalse(all(p for _, p, _ in verify.health_checks(rows)))

    def test_proposed_thought_pulse_evidence_contract(self):
        rows = [{"kind": "process_start", "at": stamp(0)},
                {"kind": "control", "command": "pause", "at": stamp(4)},
                {"kind": "control", "command": "step_once", "at": stamp(5)},
                {"kind": "thought_tick", "pulse_id": "step_1", "at": stamp(6),
                 "mode": "PAUSED", "reason": "STEP_ONCE", "state": "READY",
                 "active_request_id": None},
                {"kind": "control", "command": "resume", "at": stamp(7)},
                {"kind": "process_start", "at": stamp(8)}]
        for i in range(10):
            rows.append({"kind": "thought_tick", "pulse_id": f"tick_{i}",
                         "at": stamp(20*(i+1)), "mode": "RUNNING",
                         "state": "WORKING" if i < 3 else "READY",
                         "active_request_id": "slow_req" if i < 3 else None,
                         "model_started_ids": ["slow_req"] if i == 0 else [],
                         "action_executed_ids": ["act_1"] if i == 8 else []})
        self.assertTrue(all(p for _, p, _ in verify.thought_pulse_checks(rows)))
        rows[-1]["pulse_id"] = rows[-2]["pulse_id"]
        self.assertFalse(all(p for _, p, _ in verify.thought_pulse_checks(rows)))


class SafetyTests(unittest.TestCase):
    def test_chat_authentication_does_not_depend_on_default_port(self):
        with patch.object(verify.urllib.request, "urlopen") as http:
            resp = http.return_value.__enter__.return_value
            resp.status = 202
            resp.read.return_value = b"{}"
            code, _ = verify.request_json(
                "http://127.0.0.1:9500", "/api/say", "test-key",
                method="POST", body={"text": "fixture"})
            self.assertEqual(code, 202)
            req = http.call_args.args[0]
            self.assertEqual(req.get_header("X-tass-token"), "test-key")
            self.assertIsNone(req.get_header("Authorization"))

    def test_chat_never_sends_without_explicit_opt_in(self):
        self.assertEqual(verify.probe_chat(SimpleNamespace(send=False)), verify.NOT_RUN)


if __name__ == "__main__":
    unittest.main()
