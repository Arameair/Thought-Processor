"""Notebook lab acceptance without Nemotron or owner services."""
import json
import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
from notebook_lab import (SKILL, LESSON_ID, lesson_prompt, resolve_skill,
                          validate_model_selection, propose_skill_candidate, NotebookTestAdapter)
from pulse_cards import CATALOG, CardError, CardStore, initial, PulseCoordinator


class NotebookLabTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)

    def model_fixture(self):
        return json.dumps({
            "skill": "@identity-test", "cell_id": "identity-go-packages",
            "expectation": "I expect each approved Go package test to return exit code zero.",
        })

    def fake_pass(self):
        spec = CATALOG[LESSON_ID]
        return {
            "receipt_id": "CI-FIXTURE-RECEIPT", "job_id": "job-fixture", "card_id": LESSON_ID,
            "sha": spec["sha"], "cell_id": spec["cell_id"],
            "cell_sha256": spec["cell_sha256"],
            "notebook_blob_sha": spec["notebook_blob_sha"],
            "operation": spec["operation"],
            "outcome": "PASS", "executed": True, "exit_code": 0,
            "output_sha256": "a" * 64,
        }

    def test_skill_address_is_lookup_not_permission(self):
        r = resolve_skill(SKILL)
        self.assertEqual(r["cell_id"], "identity-go-packages")
        self.assertEqual(r["required_capability"], "test.identity.packages")
        self.assertFalse(r["execution_enabled_by_lookup"])
        self.assertEqual(r["state"], "CANDIDATE_UNAPPROVED")
        self.assertEqual(r["sha"], CATALOG[LESSON_ID]["sha"])
        for name in ("@Identity-Test", "@identity-test extra", "@unknown", "@TASS", "@ThoughtEngine", "",
                     {"skill": "@identity-test"}, None):
            with self.subTest(name=name), self.assertRaises(CardError):
                resolve_skill(name)

    def test_lesson_instructs_without_authorizing(self):
        prompt = lesson_prompt()
        self.assertIn("identity-go-packages", prompt)
        self.assertIn(CATALOG[LESSON_ID]["notebook_blob_sha"], prompt)
        self.assertIn("PROPOSAL", prompt)
        self.assertIn("separate owner approval", prompt)

    def test_model_fixture_selection_is_not_authorization(self):
        proposal = validate_model_selection(self.model_fixture())
        self.assertEqual(proposal["actor"], "MODEL_FIXTURE")
        self.assertEqual(proposal["cell_id"], "identity-go-packages")
        self.assertFalse(proposal["execution_authorized"])
        self.assertEqual(proposal["source_sha"], CATALOG[LESSON_ID]["sha"])

    def test_rejects_model_supplied_commands_unknown_cells_and_malformed(self):
        for raw in (
            '{"skill":"@identity-test","cell_id":"identity-go-packages","expectation":"good","command":"rm -rf /"}',
            '{"skill":"@identity-test","cell_id":"unsafe-shell","expectation":"This is long enough to pass."}',
            '{"skill":"@unknown","cell_id":"identity-go-packages","expectation":"This is long enough to pass."}',
            "RUN: sh -c touch /tmp/a",
            '{"skill":"@identity-test","cell_id":"identity-go-packages","expectation":"ok"}',
            '{"skill":"@identity-test","cell_id":"identity-go-packages","expectation":"This is valid"}'.replace('"identity-go-packages"', '"../identity-go-packages"'),
            '{"skill":"@identity-test","cell_id":"identity-go-packages","expectation":null}',
        ):
            with self.subTest(raw=raw[:90]), self.assertRaises(CardError):
                validate_model_selection(raw)

    def test_candidate_skill_requires_real_matching_receipt_and_interpretation(self):
        proposal = validate_model_selection(self.model_fixture())
        reflection = "The real Go packages returned exit zero; that supports only these selected packages."
        candidate = propose_skill_candidate(self.fake_pass(), selection=proposal, reflection=reflection)
        self.assertEqual(candidate["name"], "@identity-test")
        self.assertEqual(candidate["state"], "PROPOSED_FOR_OWNER_REVIEW")
        self.assertFalse(candidate["learned_by_live_nemotron"])
        self.assertFalse(candidate["execution_enabled_by_lookup"])
        self.assertEqual(candidate["first_receipt_digest"], "a" * 64)
        for field, invalid in (("sha", "0" * 40), ("cell_sha256", "b"*64),
                               ("card_id", "IDENTITY-014"), ("outcome", "FAIL"),
                               ("executed", False), ("exit_code", 1), ("receipt_id", None)):
            receipt = self.fake_pass()
            receipt[field] = invalid
            with self.subTest(field=field), self.assertRaises(CardError):
                propose_skill_candidate(receipt, selection=proposal, reflection=reflection)

    def test_existing_two_card_ledger_additive_migration_preserves_receipts(self):
        path = Path(self.temp.name) / "cards.json"
        doc = initial()
        nb = doc["cards"].pop(LESSON_ID)
        self.assertEqual(nb["status"], "PROPOSED")
        doc["cards"]["IDENTITY-014"]["status"] = "PASS"
        doc["cards"]["IDENTITY-014"]["receipt"] = {"receipt_id": "old-evidence", "outcome": "PASS"}
        doc["tick_sequence"] = 19
        path.write_text(json.dumps(doc))
        store = CardStore(path)
        try:
            now = store.read()
            self.assertEqual(now["cards"]["IDENTITY-014"]["receipt"]["receipt_id"], "old-evidence")
            self.assertEqual(now["tick_sequence"], 19)
            self.assertEqual(now["cards"][LESSON_ID]["status"], "PROPOSED")
        finally:
            store.close()
        # The new card is actually persisted; opening old data never loses the old results.
        saved = json.loads(path.read_text())
        self.assertEqual(set(saved["cards"]), set(CATALOG))
        self.assertEqual(saved["cards"]["IDENTITY-014"]["receipt"]["receipt_id"], "old-evidence")

    def test_notebook_card_no_default_execution_and_no_fallback(self):
        adapter = NotebookTestAdapter(enabled=False, sandbox_confirmed=False)
        p = PulseCoordinator(Path(self.temp.name) / "cards.json", adapter=adapter)
        try:
            p.action("approve", LESSON_ID, "APPROVE " + LESSON_ID)
            with self.assertRaises(CardError):
                p.action("queue", LESSON_ID)
            with self.assertRaises(CardError):
                adapter.preflight(CATALOG[LESSON_ID])
            self.assertEqual(p.store.read()["cards"][LESSON_ID]["attempts"], 0)
        finally:
            p.close()

    def test_trusted_notebook_must_be_inside_approved_checkout(self):
        adapter = NotebookTestAdapter(enabled=True, sandbox_confirmed=True,
                                      root=self.temp.name,
                                      roots={LESSON_ID: "/etc", "IDENTITY-014": "/etc"})
        with self.assertRaises(CardError):
            adapter.preflight(CATALOG[LESSON_ID])


if __name__ == "__main__":
    unittest.main()
