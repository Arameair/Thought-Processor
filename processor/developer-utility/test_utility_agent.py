"""Isolated protocol tests. No Identity, TheBrain, or Windows environment needed.

Identity is deliberately unreachable here (an unused loopback port), so these
tests also prove the service never reports CONNECTED/READY without a signed
Identity answer. End-to-end tests with a real Identity are in
processor/acceptance/p1_developer_entrance.py."""
import hashlib
import hmac
import http.client
import json
import os
from pathlib import Path
import secrets
import socket
import subprocess
import sys
import tempfile
import time
import unittest

HERE = Path(__file__).resolve().parent
API = "thought.feature-api/1"


class AgentTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        folder = Path(self.tmp.name)
        (folder / "runtime").mkdir()
        (folder / "data").mkdir()
        self.key = secrets.token_bytes(32)
        (folder / "key").write_text(self.key.hex())
        (folder / "credential.json").write_text(json.dumps({"installation_id": "ins_test", "identity_system_id": "idn_test",
                                                            "secret": secrets.token_hex(32), "generation": 1}))
        with socket.socket() as s:  # an unused port: never the owner's real Identity (7790/7792/7794/7796)
            s.bind(("127.0.0.1", 0))
            unused = s.getsockname()[1]
        env = dict(os.environ, TF_API=API, TF_INSTALLATION_ID="ins_test", TF_LAUNCH_ID="lch_test",
                   TF_IDENTITY_SYSTEM_ID="idn_test", TF_IDENTITY_URL="http://127.0.0.1:%d" % unused,
                   TF_RUNTIME_DIR=str(folder / "runtime"), TF_DATA_DIR=str(folder / "data"),
                   TF_CREDENTIAL_FILE=str(folder / "credential.json"), TF_LAUNCH_KEY_FILE=str(folder / "key"))
        self.proc = subprocess.Popen([sys.executable, str(HERE / "utility_agent.py")], env=env,
                                     stdout=subprocess.DEVNULL, stderr=subprocess.PIPE)
        self.addCleanup(self.stop)
        endpoint = folder / "runtime" / "endpoint.json"
        for _ in range(100):
            if endpoint.exists():
                break
            if self.proc.poll() is not None:
                self.fail("adapter exited during start")
            time.sleep(0.02)
        self.assertTrue(endpoint.exists())
        self.endpoint = json.loads(endpoint.read_text())
        self.assertFalse((folder / "key").exists())
        self.assertEqual(self.endpoint["pid"], self.proc.pid)

    def stop(self):
        if self.proc.poll() is None:
            self.proc.terminate()
            try:
                self.proc.communicate(timeout=3)
            except subprocess.TimeoutExpired:
                self.proc.kill()
                self.proc.communicate(timeout=3)
        elif self.proc.stderr:
            self.proc.stderr.close()

    def call(self, method, path, signed=False, nonce=None, auth=True, headers_extra=None):
        port = int(self.endpoint["url"].rsplit(":", 1)[1])
        conn = http.client.HTTPConnection("127.0.0.1", port, timeout=2)
        body = b"{}" if method == "POST" else b""
        headers = {"Host": "127.0.0.1"}
        if signed:
            nonce = nonce or secrets.token_hex(12)
            stamp = str(int(time.time()))
            payload = (API + " request\n" + method + "\n" + path + "\n" + stamp + "\n" + nonce + "\n" + hashlib.sha256(body).hexdigest()).encode()
            mac = hmac.new(self.key, payload, hashlib.sha256).hexdigest()
            headers.update({"X-TF-Time": stamp, "X-TF-Nonce": nonce, "X-TF-Auth": mac if auth else "0" * 64})
        headers.update(headers_extra or {})
        conn.request(method, path, body=body, headers=headers)
        response = conn.getresponse()
        raw = response.read()
        status, h = response.status, dict(response.getheaders())
        conn.close()
        return status, json.loads(raw), h, raw, nonce

    def test_unsigned_status_is_denied(self):
        code, result, *_ = self.call("GET", "/v1/status")
        self.assertEqual((code, result["error"]), (401, "LAUNCHER_AUTH_REQUIRED"))

    def test_signed_status_has_proof_and_not_ready(self):
        time.sleep(0.3)  # let the first Identity check fail
        code, result, headers, raw, nonce = self.call("GET", "/v1/status", signed=True)
        self.assertEqual(code, 200)
        self.assertFalse(result["ready"])
        self.assertFalse(result["entrance"]["connected"])
        self.assertIn(result["authority"]["state"], ("CHECKING", "UNAVAILABLE"))
        self.assertEqual(headers["X-TF-Installation-ID"], "ins_test")
        proof = hmac.new(self.key, (API + " response\n" + nonce + "\n200\n").encode() + raw, hashlib.sha256).hexdigest()
        self.assertEqual(headers["X-TF-Proof"], proof)

    def test_self_test_fails_honestly_without_identity(self):
        code, result, *_ = self.call("POST", "/v1/self-test", signed=True)
        self.assertEqual(code, 200)
        self.assertEqual(result["result"], "FAIL")
        self.assertFalse([c for c in result["output"]["checks"] if "Identity" in c["check"]][0]["ok"])

    def test_replay_is_denied(self):
        nonce = secrets.token_hex(12)
        self.assertEqual(self.call("GET", "/v1/status", signed=True, nonce=nonce)[0], 200)
        self.assertEqual(self.call("GET", "/v1/status", signed=True, nonce=nonce)[0], 401)

    def test_bad_mac_denied(self):
        self.assertEqual(self.call("GET", "/v1/status", signed=True, auth=False)[0], 401)

    def test_no_privileged_action(self):
        # Anything outside the one reviewed capability does not exist.
        for cap in ("developer.utility.launch", "developer.shell", "thought.event.submit"):
            status, result, *_ = self.call("POST", "/v1/actions/" + cap, signed=True)
            self.assertEqual((status, result["error"]), (404, "UNKNOWN_CAPABILITY"), cap)

    def test_connection_check_needs_identity_ticket(self):
        for ticket in ("", "tk1.e30." + "0" * 64, "not-a-ticket"):
            status, result, *_ = self.call("POST", "/v1/actions/developer.connection.check", headers_extra={"X-TF-Ticket": ticket})
            self.assertEqual((status, result["error"]), (401, "TICKET_INVALID"), ticket)

    def test_shutdown_only_with_valid_signature(self):
        self.assertEqual(self.call("POST", "/v1/shutdown")[0], 401)
        self.assertIsNone(self.proc.poll())
        self.assertEqual(self.call("POST", "/v1/shutdown", signed=True)[0], 202)
        self.proc.wait(timeout=3)
        self.assertEqual(self.proc.returncode, 0)


if __name__ == "__main__":
    unittest.main()
