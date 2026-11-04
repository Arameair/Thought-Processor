"""HTTP authorization and input-boundary checks for Atrium test cards (simulated Workshop)."""
import json
import socket
import sys
import tempfile
import threading
import unittest
from http.server import ThreadingHTTPServer
from pathlib import Path
from urllib.error import HTTPError
from urllib.request import ProxyHandler, Request, build_opener

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
from pulse_cards import LocalTestAdapter, PulseCoordinator
from server import create_handler

HTTP = build_opener(ProxyHandler({}))


class TestCardHTTP(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.pulse = PulseCoordinator(Path(self.temp.name) / "test-cards.json",
                                      adapter=LocalTestAdapter(enabled=False, sandbox_confirmed=False))
        self.addCleanup(self.pulse.close)
        with socket.socket() as sock:
            sock.bind(("127.0.0.1", 0))
            self.port = sock.getsockname()[1]
        self.csrf = "fixture-csrf-not-owner-auth"
        self.server = ThreadingHTTPServer(("127.0.0.1", self.port),
                                          create_handler(8790, self.csrf, self.port, pulse=self.pulse))
        self.thread = threading.Thread(target=self.server.serve_forever, daemon=True)
        self.thread.start()
        self.addCleanup(self.server.server_close)
        self.addCleanup(self.server.shutdown)

    def request(self, path, obj=None, headers=None):
        body = json.dumps(obj).encode() if obj is not None else None
        request = Request(f"http://127.0.0.1:{self.port}{path}", data=body,
                          headers=headers or ({"Content-Type": "application/json"} if body else {}),
                          method="POST" if body else "GET")
        try:
            with HTTP.open(request, timeout=4) as res:
                return res.status, json.loads(res.read())
        except HTTPError as exc:
            return exc.code, json.loads(exc.read())

    def signed(self):
        return {"Content-Type": "application/json", "X-Playground-CSRF": self.csrf,
                "Origin": f"http://127.0.0.1:{self.port}"}

    def test_two_allowlisted_cards_and_disabled_gate(self):
        code, state = self.request("/tests/state")
        self.assertEqual(code, 200)
        self.assertFalse(state["execution_enabled"])
        self.assertTrue(state["paused"])
        self.assertEqual({c["id"] for c in state["cards"]}, {"IDENTITY-014", "INTEGRITY-050", "NOTEBOOK-IDENTITY-LAB"})
        self.assertFalse(any(c["status"] == "WORKING" for c in state["cards"]))
        self.assertNotIn("env_key", json.dumps(state))
        self.assertNotIn(self.temp.name, json.dumps(state))

    def test_csrf_origin_host_and_payload_authorization(self):
        approve = {"action": "approve", "card_id": "IDENTITY-014", "ack": "APPROVE IDENTITY-014"}
        self.assertEqual(self.request("/tests/action", approve)[0], 403)
        self.assertEqual(self.request("/tests/action", approve, {
            "Content-Type": "application/json", "X-Playground-CSRF": self.csrf,
            "Origin": "https://evil.example"})[0], 403)
        self.assertEqual(self.request("/tests/action", {**approve, "argv": ["sh", "-c", "touch /tmp/unsafe"]},
                                      self.signed())[0], 400)
        self.assertEqual(self.request("/tests/action", {"action": "queue", "card_id": "IDENTITY-014"},
                                      self.signed())[0], 403)
        self.assertEqual(self.request("/tests/action", approve, self.signed())[0], 200)
        self.assertEqual(self.request("/tests/action", {"action": "queue", "card_id": "IDENTITY-014"},
                                      self.signed())[0], 403)
        self.assertEqual(self.request("/tests/state")[1]["cards"][0]["status"], "APPROVED")

    def test_pause_and_step_do_not_execute_without_runner(self):
        self.assertEqual(self.request("/tests/action", {"action": "resume"}, self.signed())[0], 200)
        self.assertEqual(self.request("/tests/action", {"action": "step"}, self.signed())[0], 200)
        state = self.request("/tests/state")[1]
        self.assertGreaterEqual(state["tick_sequence"], 1)
        self.assertEqual(state["cards"][0]["status"], "PROPOSED")

    def test_rejects_nonallowlisted_http_routing(self):
        for path in ("/tests/run", "/tests/execute", "/api/exec", "/tests/admin"):
            self.assertEqual(self.request(path, {"command": "sh -c id"}, self.signed())[0], 404)


if __name__ == "__main__":
    unittest.main()
