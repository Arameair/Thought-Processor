"""Synthetic, strictly read-only eligibility checks; no owner profiles or live services."""
import importlib.util
import hashlib
import json
from pathlib import Path
import tempfile
import unittest

SPEC = importlib.util.spec_from_file_location("gate", Path(__file__).with_name("tass_upgrade_gate.py"))
gate = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(gate)


def make_package(folder, caps, *, body=b"synthetic"):
    folder.mkdir(parents=True)
    (folder / "tass_feature.py").write_bytes(body)
    digest = hashlib.sha256(body).hexdigest()
    manifest = {"schema": "thought.feature-manifest/1", "feature_id": "tass",
                "files": [{"path": "tass_feature.py", "sha256": digest}],
                "capabilities": [{"id": cap} for cap in caps]}
    (folder / "feature.json").write_text(json.dumps(manifest))
    return gate.package_digest(folder)[0], manifest


class UpgradeGateTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        root = Path(self.tmp.name)
        self.installed = root / gate.EXPECTED_ID
        self.installed.mkdir()
        self.candidate = root / "candidate"
        self.old, manifest = make_package(self.installed / "package", ["tass.chat.send", "tass.chat.history"])
        self.new, _ = make_package(self.candidate, ["tass.chat.send", "tass.chat.history", "tass.model.set"], body=b"updated")
        old_manifest_sha = hashlib.sha256((self.installed / "package" / "feature.json").read_bytes()).hexdigest()
        self.record = {"installation_id": gate.EXPECTED_ID, "feature_id": "tass", "state": "INSTALLED",
                       "artifact_sha256": self.old, "manifest_sha256": old_manifest_sha,
                       "identity_system_id": "idn_example", "source_repo": "Arameair/TASS/feature",
                       "files": manifest["files"]}
        self.write_record()
        (self.installed / "credential.json").write_text("{}")
        (self.installed / "data").mkdir()
        (self.installed / "data" / "secret-chat.json").write_text("SENSITIVE_TEST_SENTINEL")

    def write_record(self):
        (self.installed / "install.json").write_text(json.dumps(self.record))

    def run_gate(self):
        return gate.check(self.installed, self.candidate, old_pin=self.old, new_pin=self.new)

    def test_valid_candidate_still_blocked_waiting_for_identity_upgrade_api(self):
        result = self.run_gate()
        self.assertEqual(result["gate"], "BLOCKED_IDENTITY_UPGRADE_API_MISSING")
        self.assertFalse(result["safe_to_start_new_artifact"])
        self.assertEqual(result["added_capabilities_must_be_separately_granted"], ["tass.model.set"])
        self.assertEqual((self.installed / "data" / "secret-chat.json").read_text(), "SENSITIVE_TEST_SENTINEL")

    def test_old_package_changed_denied(self):
        (self.installed / "package" / "tass_feature.py").write_bytes(b"tampered")
        with self.assertRaisesRegex(gate.GateError, "FILES_CHANGED"):
            self.run_gate()

    def test_new_package_changed_denied(self):
        (self.candidate / "tass_feature.py").write_bytes(b"tampered")
        with self.assertRaisesRegex(gate.GateError, "FILES_CHANGED"):
            self.run_gate()

    def test_wrong_installation_id_denied(self):
        self.record["installation_id"] = "ins_" + "0" * 20
        self.write_record()
        with self.assertRaisesRegex(gate.GateError, "WRONG_INSTALLATION_ID"):
            self.run_gate()

    def test_missing_identity_credential_denied(self):
        (self.installed / "credential.json").unlink()
        with self.assertRaisesRegex(gate.GateError, "CREDENTIAL_MISSING_OR_UNSAFE"):
            self.run_gate()

    def test_wrong_pinned_artifact_denied(self):
        with self.assertRaisesRegex(gate.GateError, "NEW_ARTIFACT_NOT_PINNED"):
            gate.check(self.installed, self.candidate, old_pin=self.old, new_pin="0" * 64)

    def test_removed_old_capability_denied(self):
        changed = self.candidate / "feature.json"
        doc = json.loads(changed.read_text())
        doc["capabilities"] = [{"id": "tass.model.set"}]
        changed.write_text(json.dumps(doc))
        new_pin = gate.package_digest(self.candidate)[0]
        with self.assertRaisesRegex(gate.GateError, "EXISTING_CAPABILITY_REMOVED_REVIEW_REQUIRED"):
            gate.check(self.installed, self.candidate, old_pin=self.old, new_pin=new_pin)

    def test_symlink_file_denied(self):
        (self.candidate / "tass_feature.py").unlink()
        (self.candidate / "tass_feature.py").symlink_to(self.installed / "package" / "tass_feature.py")
        with self.assertRaisesRegex(gate.GateError, "UNSAFE_PACKAGE_LINK"):
            self.run_gate()

    def test_invalid_state_denied(self):
        self.record["state"] = "UNINSTALLED"
        self.write_record()
        with self.assertRaisesRegex(gate.GateError, "NOT_INSTALLED_TASS"):
            self.run_gate()


if __name__ == "__main__":
    unittest.main()
