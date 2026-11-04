"""Issue #51 independent pulse acceptance: real process fixtures, no owner services."""
import json
import os
import stat
import subprocess
import sys
import tempfile
import threading
import time
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
from pulse_cards import CATALOG, CardError, CardStore, LocalTestAdapter, PulseCoordinator, redact


class ProcessFixture:
    """SCRIPTED test-only adapter. Spawns a REAL short Python child process,
    not Nemotron and not the owner's Go checkout."""
    enabled = True
    sandbox_confirmed = True

    def __init__(self, directory, *, held=False, fail=False):
        self.directory = Path(directory)
        self.held = held
        self.fail = fail
        self.started = threading.Event()
        self.release = threading.Event()
        self.calls = 0
        self.lock = threading.Lock()
        self.path = self.directory / "child-executions.txt"

    def preflight(self, card):
        if card.get("id") not in CATALOG:
            raise CardError("not in catalogue", 403)
        spec = CATALOG[card["id"]]
        for key in ("argv", "sha", "repository", "operation"):
            if spec[key] != card.get(key):
                raise CardError("modified operation", 403)
        return self.directory

    def execute(self, card, job_id, cancel=None):
        self.preflight(card)
        with self.lock:
            self.calls += 1
        self.started.set()
        if self.held:
            while not self.release.wait(0.01):
                if cancel is not None and cancel.is_set():
                    return self._receipt(card, job_id, None, "ERROR", False, "fixture stopped")
        # This child is fixed by the TEST, never supplied from a card or a model.
        script = "import pathlib,sys; pathlib.Path(sys.argv[1]).open('a').write('EXECUTED\\n'); print('real child completed')"
        result = subprocess.run([sys.executable, "-c", script, str(self.path)],
                                capture_output=True, text=True, timeout=5)
        code = 1 if self.fail else result.returncode
        receipt = self._receipt(card, job_id, code, "FAIL" if code else "PASS", True, None)
        receipt["output_excerpt"] = result.stdout
        return receipt

    @staticmethod
    def _receipt(card, job_id, code, status, executed, reason):
        from pulse_cards import utc
        return {
            "receipt_id": "fixture-receipt", "job_id": job_id, "card_id": card["id"],
            "operation": card["operation"], "repository": card["repository"],
            "sha": card["sha"], "argv": list(card["argv"]), "started_at": utc(),
            "finished_at": utc(), "exit_code": code, "outcome": status,
            "executed": executed, "environment": "CI_EXECUTABLE_FIXTURE",
            "output_sha256": "a" * 64, "output_excerpt": "real fixture process",
            "reason": reason,
        }


class PulseTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.path = Path(self.temp.name) / "cards.json"
        self.adapter = ProcessFixture(self.temp.name)
        self.p = PulseCoordinator(self.path, adapter=self.adapter, interval=20)
        self.addCleanup(self.p.close)

    def approve_queue(self, card_id="IDENTITY-014"):
        self.p.action("approve", card_id, "APPROVE " + card_id)
        self.p.action("queue", card_id)

    def wait_receipt(self, card_id="IDENTITY-014"):
        for _ in range(200):
            doc = self.p.store.read()
            if doc["cards"][card_id].get("pending_receipt"):
                self.p.tick()
                return self.p.store.read()["cards"][card_id]
            time.sleep(.01)
        self.fail("fixture never returned an execution receipt")

    def test_default_execution_disabled_even_with_owner_approval(self):
        self.p.close()
        adapter = LocalTestAdapter(enabled=False, sandbox_confirmed=False)
        blocked = PulseCoordinator(self.path, adapter=adapter)
        try:
            blocked.action("approve", "IDENTITY-014", "APPROVE IDENTITY-014")
            with self.assertRaises(CardError) as ctx:
                blocked.action("queue", "IDENTITY-014")
            self.assertEqual(ctx.exception.code, 403)
            self.assertEqual(blocked.store.read()["cards"]["IDENTITY-014"]["status"], "APPROVED")
            self.assertFalse(blocked.view()["execution_enabled"])
        finally:
            blocked.close()

    def test_explicit_approval_no_arbitrary_command_or_target(self):
        with self.assertRaises(CardError):
            self.p.action("approve", "IDENTITY-014", "APPROVE INTEGRITY-050")
        with self.assertRaises(CardError):
            self.p.action("queue", "IDENTITY-014")
        with self.assertRaises(CardError):
            self.p.action("approve", "UNKNOWN", "APPROVE UNKNOWN")
        with self.assertRaises(CardError):
            self.p.action("run_command", "IDENTITY-014", "go test ./...")
        self.assertEqual(self.adapter.calls, 0)

    def test_tick_steps_and_real_receipt_executes_once(self):
        self.approve_queue()
        self.p.tick()  # paused by default
        self.assertEqual(self.adapter.calls, 0)
        self.assertEqual(self.p.view()["cards"][0]["status"], "QUEUED")
        self.p.action("step")  # dispatch exactly one while retaining pause
        self.assertTrue(self.adapter.started.wait(3))
        card = self.wait_receipt()
        self.assertEqual(card["status"], "PASS")
        self.assertEqual(card["receipt"]["exit_code"], 0)
        self.assertTrue(card["receipt"]["executed"])
        self.assertEqual(card["receipt"]["sha"], CATALOG["IDENTITY-014"]["sha"])
        self.assertIn("real child completed", card["receipt"]["output_excerpt"])
        for _ in range(5):
            self.p.tick()
        self.assertEqual(self.adapter.calls, 1)
        self.assertEqual(self.adapter.path.read_text().count("EXECUTED"), 1)
        self.assertEqual(card["attempts"], 1)
        self.p.action("queue", "IDENTITY-014")  # terminal job is idempotent
        self.assertEqual(self.p.store.read()["cards"]["IDENTITY-014"]["attempts"], 1)

    def test_long_work_remains_working_across_four_pulses(self):
        self.adapter.held = True
        self.approve_queue()
        self.p.action("resume")
        self.p.tick()
        self.assertTrue(self.adapter.started.wait(3))
        job = self.p.store.read()["cards"]["IDENTITY-014"]["job_id"]
        ids = set()
        for _ in range(4):
            self.p.tick()  # simulated 20-second clock boundaries; no real 80-s sleep
            view = self.p.view()
            ids.add(view["last_pulse_id"])
            self.assertEqual(view["cards"][0]["status"], "WORKING")
            self.assertEqual(view["cards"][0]["job_id"], job)
        self.assertEqual(len(ids), 4)
        self.assertEqual(self.adapter.calls, 1)
        self.adapter.release.set()
        self.assertEqual(self.wait_receipt()["status"], "PASS")
        self.assertEqual(self.adapter.path.read_text().count("EXECUTED"), 1)

    def test_pause_resume_and_stop_cancels_inflight(self):
        self.adapter.held = True
        self.approve_queue()
        self.p.action("pause")
        self.p.tick()
        self.assertEqual(self.adapter.calls, 0)
        self.p.action("resume")
        self.p.tick()
        self.assertTrue(self.adapter.started.wait(3))
        self.p.action("stop")
        self.assertTrue(self.p.view()["paused"])
        self.assertEqual(self.wait_receipt()["status"], "ERROR")
        self.assertEqual(self.adapter.calls, 1)
        self.assertFalse(self.adapter.path.exists())  # stopped before child process was spawned

    def test_nonzero_exit_is_fail_not_expected_pass(self):
        self.adapter.fail = True
        self.approve_queue("INTEGRITY-050")
        self.p.action("step")
        card = self.wait_receipt("INTEGRITY-050")
        self.assertEqual(card["status"], "FAIL")
        self.assertTrue(card["receipt"]["executed"])
        self.assertEqual(card["receipt"]["exit_code"], 1)
        self.assertIn("Known P0 tests FAIL", card["expected"])

    def test_restart_after_finished_preserves_receipt_without_reexecution(self):
        self.approve_queue()
        self.p.action("step")
        before = self.wait_receipt()
        self.p.close()
        recovered = PulseCoordinator(self.path, adapter=self.adapter)
        try:
            recovered.tick(force=True)
            self.assertEqual(recovered.view()["cards"][0]["status"], "PASS")
            self.assertEqual(recovered.view()["cards"][0]["receipt"], before["receipt"])
            self.assertEqual(self.adapter.calls, 1)
        finally:
            recovered.close()

    def test_restart_unknown_inflight_blocks_without_retry(self):
        self.adapter.held = True
        self.approve_queue()
        self.p.action("step")
        self.assertTrue(self.adapter.started.wait(3))
        # An unclean process death leaves WORKING persisted, with no receipt.
        # Releasing this fixture only after recovery avoids losing the original process.
        self.p.store.close()
        recovered = PulseCoordinator(self.path, adapter=self.adapter)
        try:
            card = recovered.view()["cards"][0]
            self.assertEqual(card["status"], "BLOCKED")
            self.assertIn("UNKNOWN_INFLIGHT", card["reason"])
            recovered.tick(force=True)
            self.assertEqual(self.adapter.calls, 1)
        finally:
            recovered.close()
            self.adapter.release.set()

    def test_finished_pending_receipt_recovers_without_new_dispatch(self):
        self.approve_queue()
        def simulate_crash_after_child(doc):
            c = doc["cards"]["IDENTITY-014"]
            c["status"] = "WORKING"
            c["pending_receipt"] = self.adapter._receipt(c, c["job_id"], 0, "PASS", True, None)
        self.p.store.mutate(simulate_crash_after_child)
        self.p.store.close()
        recovered = PulseCoordinator(self.path, adapter=self.adapter)
        try:
            self.assertEqual(recovered.view()["cards"][0]["status"], "PASS")
            self.assertEqual(self.adapter.calls, 0)
        finally:
            recovered.close()

    def test_unique_pulse_ids_and_atomic_private_ledger(self):
        ids = set()
        for _ in range(10):
            self.p.tick()
            ids.add(self.p.view()["last_pulse_id"])
        self.assertEqual(len(ids), 10)
        self.assertEqual(self.p.view()["tick_sequence"], 10)
        self.assertEqual(stat.S_IMODE(self.path.stat().st_mode), 0o600)
        self.assertIn("IDENTITY-014", json.loads(self.path.read_text())["cards"])

    def test_tampered_definition_rejected_without_execution(self):
        self.approve_queue()
        def corrupt(d):
            d["cards"]["IDENTITY-014"]["argv"].append("; rm -rf /")
        self.p.store.mutate(corrupt)
        with self.assertRaises(CardError):
            self.p.tick(force=True)
        self.assertEqual(self.adapter.calls, 0)

    def test_broken_state_preserved_and_prevents_automatic_recovery(self):
        self.p.close()
        self.path.write_text("{damaged")
        with self.assertRaises(CardError):
            PulseCoordinator(self.path, adapter=self.adapter)
        self.assertEqual(self.path.read_text(), "{damaged")

    def test_one_writer_lock_prevents_two_atrium_dispatchers(self):
        with self.assertRaises(CardError):
            PulseCoordinator(self.path, adapter=self.adapter)

    def test_unsafe_outside_checkout_is_refused(self):
        adapter = LocalTestAdapter(enabled=True, sandbox_confirmed=True,
                                   root=self.temp.name,
                                   roots={"IDENTITY-014": "/etc",
                                          "INTEGRITY-050": "/etc"})
        with self.assertRaises(CardError):
            adapter.preflight(CATALOG["IDENTITY-014"])

    def test_receipt_and_log_redaction(self):
        raw = "Bearer ghp_" + "A" * 20 + " token=SHOULD_NOT_APPEAR password:stolen"
        redacted = redact(raw)
        self.assertNotIn("SHOULD_NOT_APPEAR", redacted)
        self.assertNotIn("stolen", redacted)
        self.assertNotIn("ghp_", redacted)


if __name__ == "__main__":
    unittest.main()
