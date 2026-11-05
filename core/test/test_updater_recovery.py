#!/usr/bin/env python3
"""Issue #79: update manager backup / failure / interruption / restore, with failure injection.

Runs the REAL tass.sh code paths (update, rollback, restore-previous, recover, history, start, status)
through updater_harness.sh, which replaces only the image build. Docker is a fake that records
which version is "up"; /api/health and /v1/state are served by a tiny in-test HTTP server that
reports the version of whatever is up. Everything lives in a temporary folder: it never touches
Docker, the owner's install, a model or the network.

    python3 core/test/test_updater_recovery.py -v
"""
import hashlib
import http.server
import json
import os
from pathlib import Path
import socket
import subprocess
import tempfile
import textwrap
import threading
import unittest

CORE = Path(__file__).resolve().parents[1]
ROOT = CORE.parent
HARNESS = CORE / "test" / "updater_harness.sh"


def git(*a):
    return subprocess.run(["git", *a], cwd=ROOT, text=True, capture_output=True, check=True).stdout.strip()


def free_port():
    s = socket.socket(); s.bind(("127.0.0.1", 0)); p = s.getsockname()[1]; s.close(); return p


def tree(path):
    """{relative file: sha256} for every file under path (the owner's data, byte for byte)."""
    out = {}
    for p in sorted(Path(path).rglob("*")):
        if p.is_file():
            out[str(p.relative_to(path))] = hashlib.sha256(p.read_bytes()).hexdigest()
    return out


class Fake:
    """State shared by the fake docker and the fake health endpoints."""

    def __init__(self, root):
        self.state = root / "state"; self.state.mkdir()
        (self.state / "images").write_text("")
        self.bad = set()          # core versions that come up unhealthy
        self.servers = []

    def up_image(self):
        f = self.state / "up"
        return f.read_text().strip() if f.exists() else ""

    def version(self):
        img = self.up_image()
        for line in (self.state / "images").read_text().splitlines():
            tag, sha = line.split()
            if tag == img:
                return sha
        return ""

    def serve(self, port):
        fake = self

        class H(http.server.BaseHTTPRequestHandler):
            def log_message(self, *a):
                pass

            def do_GET(self):
                if not fake.up_image():
                    self.send_response(503); self.end_headers(); return
                if self.path == "/api/health":
                    body = {"state": "BLOCKED", "version": fake.version(), "action_mode": "CHAT_ONLY", "reasons": []}
                elif self.path == "/v1/state":
                    body = {"state": "LOCKED", "entity": {"entity_id": "ent-owner-1"}}
                else:
                    self.send_response(404); self.end_headers(); return
                b = json.dumps(body).encode()
                self.send_response(200); self.send_header("Content-Type", "application/json")
                self.send_header("Content-Length", str(len(b))); self.end_headers(); self.wfile.write(b)

        srv = http.server.ThreadingHTTPServer(("127.0.0.1", port), H)
        threading.Thread(target=srv.serve_forever, daemon=True).start()
        self.servers.append(srv)

    def close(self):
        for s in self.servers:
            s.shutdown(); s.server_close()


class UpdaterRecovery(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        # three real commits of this repository stand in for three versions
        cls.v = [git("rev-parse", f"HEAD~{n}") for n in (2, 1, 0)]

    def setUp(self):
        self.t = tempfile.TemporaryDirectory(prefix="tass-updater-79-")
        self.addCleanup(self.t.cleanup)
        self.root = Path(self.t.name)
        self.bin = self.root / "bin"; self.bin.mkdir()
        self.app = self.root / "app"; self.app.mkdir()
        self.fake = Fake(self.root); self.addCleanup(self.fake.close)
        cp, ip = free_port(), free_port()
        self.fake.serve(cp); self.fake.serve(ip)
        (self.app / "app.env").write_text(
            "MODEL_MODE=external\nMODEL_NETWORK=tass-test_default\nMODEL_URL=http://model:8081\n"
            f"CORE_PORT={cp}\nIDENTITY_PORT={ip}\nMODEL_GPU=off\n")
        self.exe("docker", r"""
            #!/usr/bin/env bash
            set -euo pipefail
            verb="${1:-}"; shift || true
            S="$TEST_STATE"
            case "$verb" in
              info) echo '{"runtimes":{}}' ;;
              version) echo 29.0.0 ;;
              network|image) exit 0 ;;
              ps) [[ -f "$S/up" ]] && echo c0ffee ; exit 0 ;;
              inspect)
                if [[ -f "$S/up" ]] && ! grep -qxF "$(cat "$S/up")" "$S/bad" 2>/dev/null; then echo healthy; else echo unhealthy; fi ;;
              compose)
                echo "compose $* CORE_IMAGE=$CORE_IMAGE" >>"$S/log"
                case " $* " in
                  *" up "*) echo "$CORE_IMAGE" >"$S/up"
                            # a candidate that damages data when it starts (records shrink)
                            if grep -qxF "$CORE_IMAGE" "$S/corrupts" 2>/dev/null; then : >"$APP_DATA/core/memory.jsonl"; fi ;;
                  *" stop "*|*" down "*) rm -f "$S/up" ;;
                esac ;;
              *) echo "unexpected docker $verb" >&2; exit 96 ;;
            esac
        """)
        self.exe("tar", r"""
            #!/usr/bin/env bash
            # Real tar; TEST_TAR_TRUNCATE=1 cuts every archive it writes in half (bad disk / full disk).
            /usr/bin/tar "$@"; rc=$?
            if [[ $rc == 0 && "${TEST_TAR_TRUNCATE:-0}" == 1 && "$1" == -czf ]]; then
              s=$(stat -c %s "$2"); truncate -s $((s / 2)) "$2"
            fi
            exit $rc
        """)
        self.env = {**os.environ, "APP_HOME": str(self.app), "HOME": str(self.root),
                    "PATH": f"{self.bin}:{os.environ['PATH']}", "TEST_STATE": str(self.fake.state)}
        self.data = self.app / "data"
        self.seed_owner_data()
        # deploy version 0 (as start would) and bring it up
        self.h("start-fake")

    # ------------------------------------------------------------------ helpers
    def exe(self, name, body):
        p = self.bin / name; p.write_text(textwrap.dedent(body).strip() + "\n"); p.chmod(0o755)

    def seed_owner_data(self):
        files = {
            "identity/entity.json": '{"entity_id":"ent-owner-1"}\n',
            "identity/decisions.jsonl": "".join(f'{{"n":{i}}}\n' for i in range(5)),
            "identity/credentials.json": '{"sealed":"…owner-credential…"}\n',
            "core/tass.md": "# TASS\nowner notes\n",
            "core/memory.jsonl": "".join(f'{{"note":{i}}}\n' for i in range(7)),
            "core/conversation.jsonl": '{"m":"hi"}\n',
            "core/lessons.json": '{"00":{"status":"APPROVED"}}\n',
            "core/approvals.jsonl": '{"lesson":"00","by":"owner"}\n',
            "core/junctions.jsonl": '{"junction":"lesson-metadata"}\n',
            "engine/cycles.jsonl": "".join(f'{{"cycle":{i}}}\n' for i in range(9)),
            "engine/jobs.jsonl": '{"job":"j1"}\n',
            "engine/api-key": "seeded-engine-api-key\n",   # a real install already has its keys
        }
        for rel, txt in files.items():
            p = self.data / rel; p.parent.mkdir(parents=True, exist_ok=True); p.write_text(txt)
        (self.app / "secrets").mkdir()
        for k in ("github_token", "engine.key", "core.key"):
            (self.app / "secrets" / k).write_text(f"not-a-real-{k}\n")

    def h(self, *args, ok=None, **env):
        if args[0] == "start-fake":
            d = self.app / "deploy"; d.mkdir(exist_ok=True)
            v = self.v[0]
            (d / "current.env").write_text(
                f"CORE_SHA={v}\nENGINE_SHA=d69d3917d1601d806d4b2ba2254693e1adb92b6b\n"
                "IDENTITY_SHA=7da7f7c5bbecc42d3a7e9316a2037243eda10b88\n"
                f"CORE_IMAGE=tass-core:{v[:12]}\nENGINE_IMAGE=tass-engine:d69d3917d160\n"
                f"IDENTITY_IMAGE=tass-identity:7da7f7c5bbec\nCOMPOSE_FILE={CORE}/compose.yaml\n"
                "DEPLOYED_AT=2026-11-05T00:00:00Z\n")
            with open(self.fake.state / "images", "a") as f:
                f.write(f"tass-core:{v[:12]} {v}\n")
            (self.fake.state / "up").write_text(f"tass-core:{v[:12]}\n")
            return None
        p = subprocess.run(["bash", str(HARNESS), *args], env={**self.env, **env}, text=True,
                           capture_output=True, timeout=120, input="")
        out = p.stdout + p.stderr
        if ok is True:
            self.assertEqual(p.returncode, 0, out)
        elif ok is False:
            self.assertNotEqual(p.returncode, 0, out)
        return p.returncode, out

    def running(self):
        return self.fake.version()

    def current_env_sha(self):
        for line in (self.app / "deploy" / "current.env").read_text().splitlines():
            if line.startswith("CORE_SHA="):
                return line.split("=", 1)[1]

    def mark_bad(self, sha):
        with open(self.fake.state / "bad", "a") as f:
            f.write(f"tass-core:{sha[:12]}\n")

    def no_op_marker(self):
        self.assertFalse((self.app / "deploy" / "operation.env").exists(), "operation marker left behind")

    def backups(self):
        return sorted((self.app / "deploy" / "backups").glob("*.tgz"))

    # ------------------------------------------------------------------ tests
    def test_01_successful_update_makes_verified_backup_and_restore_point(self):
        before = tree(self.data)
        rc, out = self.h("update", self.v[1], "--yes", ok=True)
        self.assertIn("Backup (verified, SHA-256 recorded)", out)
        self.assertEqual(self.running(), self.v[1])
        self.assertEqual(tree(self.data), before, "an update must not change owner data")
        [b] = self.backups()
        self.assertEqual(hashlib.sha256(b.read_bytes()).hexdigest(), Path(str(b) + ".sha256").read_text().strip())
        self.assertTrue((self.app / "deploy" / "last-update.env").exists())
        self.assertFalse(list((self.app / "deploy" / "backups").glob(".*")), "no partial/scratch files left")
        self.assertEqual(oct(b.stat().st_mode & 0o777), "0o600")
        self.no_op_marker()
        rc, hist = self.h("history", ok=True)
        self.assertIn("verified", hist); self.assertIn("Restore previous update available", hist)

    def test_02_unhealthy_candidate_restores_previous_version_and_data(self):
        before = tree(self.data)
        self.mark_bad(self.v[1])
        rc, out = self.h("update", self.v[1], "--yes", ok=False)
        self.assertIn("Update FAILED validation", out)
        self.assertIn("ROLLED BACK: running the previous version again", out)
        self.assertEqual(self.running(), self.v[0])
        self.assertEqual(self.current_env_sha(), self.v[0])
        self.assertEqual(tree(self.data), before)
        self.no_op_marker()

    def test_03_candidate_that_damages_data_is_undone_and_damage_kept_aside(self):
        before = tree(self.data)
        (self.fake.state / "corrupts").write_text(f"tass-core:{self.v[1][:12]}\n")
        rc, out = self.h("update", self.v[1], "--yes", ok=False)
        self.assertIn("memory.jsonl had 7 lines, now 0", out)
        self.assertEqual(self.running(), self.v[0])
        self.assertEqual(tree(self.data), before, "owner data must be byte-identical after the restore")
        aside = list((self.app / "deploy" / "failed").glob("*/data/core/memory.jsonl"))
        self.assertEqual(len(aside), 1, "the damaged data is moved aside, never deleted")
        self.no_op_marker()

    def test_04_backup_that_fails_verification_refuses_update_and_restarts_old_version(self):
        before = tree(self.data)
        rc, out = self.h("update", self.v[1], "--yes", ok=False, TEST_TAR_TRUNCATE="1")
        self.assertIn("backup FAILED verification", out)
        self.assertIn("the update was NOT applied", out)
        self.assertEqual(self.running(), self.v[0], "old version must be running again")
        self.assertEqual(self.current_env_sha(), self.v[0])
        self.assertEqual(tree(self.data), before)
        self.assertEqual(self.backups(), [], "a failed backup is not kept as if it were good")
        self.assertFalse(list((self.app / "deploy" / "backups").glob(".*")))
        self.assertFalse((self.app / "deploy" / "last-update.env").exists())
        self.no_op_marker()

    def test_05_build_failure_changes_nothing(self):
        before = tree(self.data)
        self.h("update", self.v[1], "--yes", ok=False, TEST_BUILD_FAIL="1")
        self.assertEqual(self.running(), self.v[0])
        self.assertEqual(self.current_env_sha(), self.v[0])
        self.assertEqual(tree(self.data), before)
        self.assertEqual(self.backups(), [])
        self.no_op_marker()

    def interrupted(self, phase, corrupt=False):
        before = tree(self.data)
        if corrupt:
            (self.fake.state / "corrupts").write_text(f"tass-core:{self.v[1][:12]}\n")
        rc, out = self.h("update", self.v[1], "--yes", TEST_KILL_AT=phase)
        self.assertEqual(rc, -9 if rc < 0 else 137, out)
        self.assertTrue((self.app / "deploy" / "operation.env").exists())
        # every mutating command refuses to act on top of it
        for cmd in (("update", self.v[2], "--yes"), ("rollback",), ("start",), ("restore-previous", "--yes")):
            rc2, out2 = self.h(*cmd, ok=False)
            self.assertIn("INTERRUPTED", out2, cmd)
        rc, st = self.h("status")
        self.assertIn("INTERRUPTED operation recorded", st)
        rc, out = self.h("recover", ok=True)
        self.assertIn("RECOVERED", out)
        self.assertEqual(self.current_env_sha(), self.v[0])
        self.assertEqual(self.running(), self.v[0])
        self.assertEqual(tree(self.data), before)
        self.no_op_marker()
        self.h("recover", ok=True)  # idempotent: nothing left to do

    def test_06_power_cut_after_stop_before_backup(self):
        self.interrupted("STOPPED")

    def test_07_power_cut_after_switch(self):
        self.interrupted("SWITCHED")

    def test_08_power_cut_while_validating_a_candidate_that_damaged_data(self):
        self.interrupted("VALIDATING", corrupt=True)

    def test_09_restore_previous_update_puts_back_version_and_pre_update_data(self):
        before = tree(self.data)
        self.h("update", self.v[1], "--yes", ok=True)
        # the new version writes new records after the update
        with open(self.data / "core" / "memory.jsonl", "a") as f:
            f.write('{"note":"written after the update"}\n')
        after_update = tree(self.data)
        rc, out = self.h("restore-previous", "--yes", ok=True)
        self.assertIn("Current data saved first (verified)", out)
        self.assertEqual(self.running(), self.v[0])
        self.assertEqual(self.current_env_sha(), self.v[0])
        self.assertEqual(tree(self.data), before)
        # what was recorded after the update is kept twice: moved aside and in a verified backup
        aside = list((self.app / "deploy" / "failed").glob("*/data/core/memory.jsonl"))
        self.assertEqual(len(aside), 1)
        self.assertIn("written after the update", aside[0].read_text())
        guard = [b for b in self.backups() if "before-restore-from" in b.name]
        self.assertEqual(len(guard), 1)
        self.no_op_marker()
        # and the restore itself is undoable
        rc, out = self.h("rollback", "--restore-data", str(guard[0]), ok=True)
        self.assertEqual(self.running(), self.v[1])
        self.assertEqual(tree(self.data), after_update)

    def test_10_restore_previous_refuses_a_changed_backup(self):
        self.h("update", self.v[1], "--yes", ok=True)
        before = tree(self.data)
        [b] = self.backups()
        with open(b, "ab") as f:
            f.write(b"tampered")
        rc, out = self.h("restore-previous", "--yes", ok=False)
        self.assertIn("missing or changed", out)
        self.assertEqual(self.running(), self.v[1])
        self.assertEqual(tree(self.data), before)
        rc, hist = self.h("history", ok=True)
        self.assertIn("CHANGED", hist)
        self.no_op_marker()

    def test_11_restore_previous_that_fails_validation_returns_to_where_it_was(self):
        self.h("update", self.v[1], "--yes", ok=True)
        with open(self.data / "core" / "memory.jsonl", "a") as f:
            f.write('{"note":"post-update"}\n')
        before = tree(self.data)
        self.mark_bad(self.v[0])
        rc, out = self.h("restore-previous", "--yes", ok=False)
        self.assertIn("Restore FAILED validation", out)
        self.assertEqual(self.running(), self.v[1])
        self.assertEqual(tree(self.data), before)
        self.no_op_marker()

    def test_12_rollback_to_a_broken_version_is_undone(self):
        self.h("update", self.v[1], "--yes", ok=True)
        before = tree(self.data)
        self.mark_bad(self.v[0])
        rc, out = self.h("rollback", ok=False)
        self.assertIn("Rollback FAILED validation", out)
        self.assertIn("running again", out)
        self.assertEqual(self.running(), self.v[1])
        self.assertEqual(self.current_env_sha(), self.v[1])
        self.assertEqual(tree(self.data), before)
        self.no_op_marker()

    def test_13_only_one_operation_at_a_time(self):
        (self.app / "deploy").mkdir(exist_ok=True)
        lock = open(self.app / "deploy" / ".op.lock", "w")
        self.addCleanup(lock.close)
        import fcntl
        fcntl.flock(lock, fcntl.LOCK_EX)
        rc, out = self.h("update", self.v[1], "--yes", ok=False)
        self.assertIn("another ./tass.sh operation is running", out)
        self.assertEqual(self.running(), self.v[0])
        self.assertEqual(self.backups(), [])

    def test_14_manual_backup_is_verified_and_app_comes_back(self):
        before = tree(self.data)
        rc, out = self.h("backup", ok=True)
        self.assertIn("Backup (verified)", out)
        self.assertEqual(self.running(), self.v[0])
        self.assertEqual(tree(self.data), before)
        rc, out = self.h("backup", ok=False, TEST_TAR_TRUNCATE="1")
        self.assertIn("Backup FAILED", out)
        self.assertEqual(self.running(), self.v[0])


if __name__ == "__main__":
    unittest.main()
