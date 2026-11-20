"""Offline tests for the synthetic TASS workloop, not production integration tests."""
from pathlib import Path
import sys
import tempfile
import unittest

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
from workloop import GuardError, SOURCES, Workloop, run_cases


class WorkloopTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.w = Workloop(Path(self.temp.name))

    def test_bounded_demo_and_before_after(self):
        self.w.run()
        self.assertTrue(all(s.state == "AUDIT_PASS" for s in self.w.streams.values()))
        self.assertIn("ASSIGNED", (Path(self.temp.name) / "BEFORE.md").read_text())
        self.assertIn("AUDIT_PASS", (Path(self.temp.name) / "AFTER.md").read_text())
        self.assertIn("FAIL:", (Path(self.temp.name) / "EVENTS.md").read_text())
        self.assertIn("RE-AUDIT", (Path(self.temp.name) / "EVENTS.md").read_text())

    def test_smoke_passes_but_independent_regression_fails(self):
        self.w.build()
        self.assertEqual("PASS", self.w.streams["WS-A"].smoke)
        self.assertEqual("FAIL", self.w.audit("WS-A", self.w.streams["WS-A"].digest))
        self.assertEqual("NEEDS_REPAIR", self.w.streams["WS-A"].state)

    def test_repair_invalidates_old_audit_revision(self):
        self.w.build()
        old = self.w.streams["WS-A"].digest
        self.w.audit("WS-A", old)
        self.w.repair()
        self.assertEqual("NOT RUN", self.w.streams["WS-A"].audit)
        with self.assertRaisesRegex(GuardError, "exact current"):
            self.w.audit("WS-A", old, repeat=True)
        self.assertEqual("PASS", self.w.audit("WS-A", self.w.streams["WS-A"].digest, repeat=True))

    def test_auditor_cannot_write_builder_record(self):
        with self.assertRaisesRegex(GuardError, "builder-only"):
            self.w.receipt("WS-A", "Codex", "BUILD", "try")
        with self.assertRaisesRegex(GuardError, "independent auditor"):
            self.w.receipt("WS-A", "Claude", "AUDIT", "try")

    def test_cas_rejects_stale_writer(self):
        self.w.receipt("WS-A", "GPT-6", "COORDINATE", "first", expected=0)
        with self.assertRaisesRegex(GuardError, "CAS conflict"):
            self.w.receipt("WS-A", "GPT-6", "COORDINATE", "stale", expected=0)
        self.assertEqual(1, self.w.versions["WS-A/GPT-6"])

    def test_private_content_is_rejected(self):
        for secret in ["memory.jsonl", "identity-grant.json", "Bearer abc123", "private memory"]:
            with self.subTest(secret=secret), self.assertRaisesRegex(GuardError, "private"):
                self.w.receipt("WS-A", "GPT-6", "COORDINATE", secret)

    def test_two_distinct_workstream_records(self):
        self.w.run()
        claude = Path(self.temp.name) / "handoffs/WS-A/Claude.md"
        codex = Path(self.temp.name) / "handoffs/WS-B/Codex.md"
        self.assertNotEqual(claude, codex)
        self.assertIn("REPAIR", claude.read_text())
        self.assertIn("PASS", codex.read_text())
        self.assertNotIn("REPAIR", codex.read_text())

    def test_tass_has_no_executable_authority(self):
        self.w.proposed_tass_ack()
        self.assertEqual("ASSIGNED", self.w.streams["WS-A"].state)
        with self.assertRaises(GuardError):
            self.w.receipt("WS-A", "TASS-sim", "BUILD", "execute")
        self.assertIn("NOT AUTHORIZED", (Path(self.temp.name) / "handoffs/WS-A/TASS-sim.md").read_text())

    def test_invalid_candidate_cannot_be_audited(self):
        with self.assertRaises(GuardError):
            self.w.audit("WS-A", "deadbeef")

    def test_fixtures_are_real_executable_checks(self):
        result, _ = run_cases(SOURCES / "normalize_v1.py", "normalize", [(" a  b ", "a b")])
        self.assertEqual("FAIL", result)
        result, _ = run_cases(SOURCES / "normalize_v2.py", "normalize", [(" a  b ", "a b")])
        self.assertEqual("PASS", result)


if __name__ == "__main__":
    unittest.main()
