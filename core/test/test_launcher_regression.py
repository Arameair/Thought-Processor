#!/usr/bin/env python3
"""Offline one-app launcher regression with fake Docker/sudo/Git.

Never talks to real Docker, the owner's model, or any running installation.
"""
import os
from pathlib import Path
import subprocess
import tempfile
import textwrap
import unittest

CORE = Path(__file__).resolve().parents[1]
ROOT = CORE.parent


def call(argv, env=None, cwd=CORE):
    return subprocess.run(argv, cwd=cwd, env=env, text=True, capture_output=True, timeout=45)


class LauncherSafety(unittest.TestCase):
    def setUp(self):
        self.t = tempfile.TemporaryDirectory(prefix="one-app-offline-")
        self.addCleanup(self.t.cleanup)
        self.root = Path(self.t.name)
        self.bin = self.root / "bin"
        self.bin.mkdir()
        self.app = self.root / "folder with spaces" / "app"
        self.app.mkdir(parents=True)
        self.dockerlog = self.root / "docker.log"
        self.sha = call(["/usr/bin/git", "rev-parse", "HEAD"], cwd=ROOT).stdout.strip()
        self.assertEqual(len(self.sha), 40)
        (self.app / "app.env").write_text(
            "MODEL_MODE=external\nMODEL_NETWORK=tass-workspace_default\n"
            "MODEL_URL=http://model:8081\nCORE_PORT=18897\nIDENTITY_PORT=18896\nMODEL_GPU=off\n")
        self.env = {**os.environ, "APP_HOME": str(self.app), "HOME": str(self.root),
                    "PATH": str(self.bin) + ":" + os.environ["PATH"],
                    "TEST_LOG": str(self.dockerlog), "TEST_FORCE_SUDO": "0",
                    "TEST_FAIL_CLONE": "0", "TEST_FAIL_BUILD": "", "TEST_IMAGE_MISSING": "0"}
        self.exe("docker", r"""
            #!/usr/bin/env bash
            set -euo pipefail
            verb="${1:-}"; shift || true
            case "$verb" in
              info)
                [[ "${TEST_FORCE_SUDO:-0}" != 1 || "${SUDO_GRANTED:-0}" == 1 ]] || exit 1
                echo '{"runtimes":{}}';;
              version) echo 29.0.0;;
              network) exit 0;;
              ps) exit 0;;
              image) [[ "${TEST_IMAGE_MISSING:-0}" != 1 ]];;
              compose)
                printf 'compose %s\n' "$*" >>"$TEST_LOG"
                for k in CORE_IMAGE ENGINE_IMAGE IDENTITY_IMAGE APP_UID APP_GID APP_DATA APP_SECRETS MODEL_ARGS MODEL_URL MODEL_NETWORK CORE_PORT IDENTITY_PORT; do
                  v="${!k:-}"
                  [[ -n "$v" ]] || { printf 'MISSING %s\n' "$k" >&2; exit 95; }
                  printf '%s=%s\n' "$k" "$v" >>"$TEST_LOG"
                done;;
              build)
                printf 'build %s\n' "$*" >>"$TEST_LOG"
                [[ "$*" != *"-t tass-${TEST_FAIL_BUILD:-nonesuch}:"* ]] || exit 77;;
              *) echo "unexpected docker: $verb" >&2; exit 96;;
            esac
        """)
        self.exe("sudo", r"""
            #!/usr/bin/env bash
            set -euo pipefail
            [[ "${1:-}" == "-n" ]] || exit 91
            shift
            # Simulated sudo env_reset: only fixture flags survive.
            exec /usr/bin/env -i PATH="$PATH" HOME="$HOME" APP_HOME="$APP_HOME" \
              TEST_LOG="$TEST_LOG" TEST_FORCE_SUDO="$TEST_FORCE_SUDO" \
              TEST_IMAGE_MISSING="${TEST_IMAGE_MISSING:-0}" \
              TEST_FAIL_BUILD="${TEST_FAIL_BUILD:-}" SUDO_GRANTED=1 "$@"
        """)
        self.exe("git", r"""
            #!/usr/bin/env bash
            set -euo pipefail
            if [[ "${1:-}" == clone ]]; then
              printf 'clone %s\n' "$*" >>"$TEST_LOG"
              [[ "${TEST_FAIL_CLONE:-0}" != 1 ]] || exit 72
              dest="${@: -1}"
              mkdir -p "$dest/.git"
              exit 0
            fi
            if [[ "${1:-}" == -C && "${2:-}" == "$APP_HOME"/deploy/src/engine-* ]]; then
              if [[ "${3:-}" == checkout ]]; then exit 0; fi
              if [[ "${3:-}" == rev-parse ]]; then echo a847385000000000000000000000000000000000; exit 0; fi
            fi
            if [[ "${1:-}" == -C && "${2:-}" == "$APP_HOME"/deploy/src/identity-* ]]; then
              if [[ "${3:-}" == checkout ]]; then exit 0; fi
              if [[ "${3:-}" == rev-parse ]]; then echo 8b175c8000000000000000000000000000000000; exit 0; fi
            fi
            exec /usr/bin/git "$@"
        """)

    def exe(self, name, value):
        p = self.bin / name
        p.write_text(textwrap.dedent(value).strip() + "\n")
        p.chmod(0o755)

    def deployed(self, include=True):
        d = self.app / "deploy"
        d.mkdir(exist_ok=True)
        f = d / "current.env"
        if include:
            f.write_text(
                f"CORE_SHA={self.sha}\nENGINE_SHA=a847385000000000000000000000000000000000\n"
                "IDENTITY_SHA=8b175c8000000000000000000000000000000000\n"
                f"CORE_IMAGE=tass-core:{self.sha[:12]}\n"
                "ENGINE_IMAGE=tass-engine:a84738500000\nIDENTITY_IMAGE=tass-identity:8b175c800000\n"
                f"COMPOSE_FILE={CORE}/compose.yaml\nDEPLOYED_AT=2026-11-04T00:00:00Z\n")
        return f

    def run_script(self, subcmd, **overrides):
        env = {**self.env, **{k: str(v) for k, v in overrides.items()}}
        return call(["bash", str(CORE / "tass.sh"), subcmd], env=env)

    def test_compose_receives_every_interpolation_variable_after_sudo_reset(self):
        self.deployed()
        p = self.run_script("logs", TEST_FORCE_SUDO=1)
        self.assertEqual(p.returncode, 0, p.stdout + p.stderr)
        out = self.dockerlog.read_text()
        for key in ("CORE_IMAGE", "ENGINE_IMAGE", "IDENTITY_IMAGE", "APP_DATA", "APP_SECRETS",
                    "MODEL_ARGS", "MODEL_URL", "MODEL_NETWORK", "APP_UID", "APP_GID",
                    "CORE_PORT", "IDENTITY_PORT"):
            self.assertIn(key + "=", out)
        self.assertIn("folder with spaces/app/data", out)
        self.assertIn("MODEL_URL=http://model:8081", out)
        self.assertIn("CORE_PORT=18897", out)
        self.assertNotIn("MISSING", out)

    def test_direct_docker_path_unchanged(self):
        self.deployed()
        p = self.run_script("logs")
        self.assertEqual(p.returncode, 0, p.stdout + p.stderr)
        self.assertIn("CORE_IMAGE=tass-core:", self.dockerlog.read_text())

    def test_failed_engine_clone_does_not_launch_or_create_current(self):
        cur = self.deployed(include=False)
        p = self.run_script("start", TEST_FORCE_SUDO=1, TEST_FAIL_CLONE=1)
        self.assertNotEqual(p.returncode, 0, p.stdout + p.stderr)
        self.assertIn("engine clone failed", p.stderr)
        self.assertFalse(cur.exists())
        self.assertFalse((cur.parent / "history.log").exists())
        self.assertNotIn("compose ", self.dockerlog.read_text())
        self.assertNotIn("build ", self.dockerlog.read_text())

    def test_each_failed_image_build_keeps_old_deployment_unchanged(self):
        for target in ("core", "engine", "identity"):
            with self.subTest(target=target):
                cur = self.deployed()
                original = cur.read_bytes()
                self.dockerlog.write_text("")
                p = self.run_script("start", TEST_FORCE_SUDO=1,
                                    TEST_IMAGE_MISSING=1, TEST_FAIL_BUILD=target)
                self.assertNotEqual(p.returncode, 0, p.stdout + p.stderr)
                self.assertIn(target + " image build failed", p.stderr)
                self.assertEqual(cur.read_bytes(), original)
                self.assertFalse((cur.parent / "history.log").exists())
                out = self.dockerlog.read_text()
                self.assertNotIn("compose ", out)
                self.assertIn("-t tass-" + target + ":", out)


if __name__ == "__main__":
    unittest.main(verbosity=2)
