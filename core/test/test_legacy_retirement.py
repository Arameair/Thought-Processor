#!/usr/bin/env python3
"""Negative tests for retired legacy Processor entrypoints.

All commands use temporary homes and profiles. They must refuse before opening
a listener, creating state, or emitting a private launch URL.
"""
from __future__ import annotations

import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import unittest


ROOT = Path(__file__).resolve().parents[2]
ESCAPE_HATCH = "allow-retired-legacy-processor"


class LegacyRetirementTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.build_dir = tempfile.TemporaryDirectory(prefix="tass-retired-processor-")
        suffix = ".exe" if os.name == "nt" else ""
        cls.binary = Path(cls.build_dir.name) / f"thought-processor{suffix}"
        subprocess.run(
            ["go", "build", "-o", str(cls.binary), "."],
            cwd=ROOT / "processor",
            check=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True,
        )

    @classmethod
    def tearDownClass(cls):
        cls.build_dir.cleanup()

    def isolated_env(self, home: Path) -> dict[str, str]:
        env = os.environ.copy()
        env["HOME"] = str(home)
        env["USERPROFILE"] = str(home)
        env["THOUGHT_PROCESSOR_DATA"] = str(home / "processor-profile")
        env["P1_LAUNCH_WORK"] = str(home / "p1-launch")
        return env

    def assert_no_private_link(self, output: str):
        self.assertNotIn("#code=", output)
        self.assertNotIn("operator-link.txt", output)
        self.assertNotRegex(output, r"http://127\.0\.0\.1:\d+/")

    def test_default_binary_refuses_without_state_or_private_link(self):
        with tempfile.TemporaryDirectory(prefix="tass-retirement-home-") as td:
            home = Path(td)
            result = subprocess.run(
                [str(self.binary), "-port", "0"],
                env=self.isolated_env(home),
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
                text=True,
                timeout=10,
            )
            output = result.stdout + result.stderr
            self.assertNotEqual(result.returncode, 0)
            self.assertIn("RETIRED", output)
            self.assertFalse((home / "processor-profile").exists())
            self.assert_no_private_link(output)

    def test_escape_hatch_is_explicit_and_has_mandatory_warning(self):
        # CI must never enable the escape hatch, even for a version probe. Its
        # source contract is checked statically; runtime tests exercise only
        # the default-deny path.
        source = (ROOT / "processor/main.go").read_text(encoding="utf-8")
        self.assertIn(f'flag.Bool("{ESCAPE_HATCH}"', source)
        self.assertIn("SECURITY WARNING", source)
        self.assertIn("DEVELOPMENT ONLY", source)

    @unittest.skipUnless(os.name == "posix" and shutil.which("bash"), "POSIX bash is required for shell launcher execution")
    def test_shell_launchers_refuse_before_creating_work(self):
        with tempfile.TemporaryDirectory(prefix="tass-retirement-home-") as td:
            home = Path(td)
            env = self.isolated_env(home)
            for script in ("run-p1.sh", "check-tass.sh"):
                with self.subTest(script=script):
                    result = subprocess.run(
                        ["bash", str(ROOT / script)],
                        cwd=ROOT,
                        env=env,
                        stdout=subprocess.PIPE,
                        stderr=subprocess.PIPE,
                        text=True,
                        timeout=10,
                    )
                    output = result.stdout + result.stderr
                    self.assertNotEqual(result.returncode, 0)
                    self.assertIn("RETIRED", output)
                    self.assert_no_private_link(output)
            self.assertFalse((home / "p1-launch").exists())

    def test_python_launcher_refuses_before_creating_work(self):
        with tempfile.TemporaryDirectory(prefix="tass-retirement-home-") as td:
            home = Path(td)
            result = subprocess.run(
                [sys.executable, str(ROOT / "tools/p0-launcher/secure_launch.py"), "start"],
                cwd=ROOT,
                env=self.isolated_env(home),
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
                text=True,
                timeout=10,
            )
            output = result.stdout + result.stderr
            self.assertNotEqual(result.returncode, 0)
            payload = json.loads(result.stdout)
            self.assertEqual(payload["launcher"], "RETIRED")
            self.assertEqual(payload["result"], "LEGACY_PROCESSOR_NOT_STARTED")
            self.assertFalse((home / "p1-launch").exists())
            self.assert_no_private_link(output)

    def test_supported_launchers_never_enable_escape_hatch(self):
        launchers = (
            ROOT / "run-p1.sh",
            ROOT / "check-tass.sh",
            ROOT / "tools/p0-launcher/secure_launch.py",
            ROOT / "tools/p0-launcher/Workbench.ps1",
            ROOT / "core/tass.sh",
            ROOT / "core/start.sh",
        )
        for launcher in launchers:
            with self.subTest(launcher=launcher.relative_to(ROOT)):
                self.assertNotIn(ESCAPE_HATCH, launcher.read_text(encoding="utf-8"))


if __name__ == "__main__":
    unittest.main(verbosity=2)
