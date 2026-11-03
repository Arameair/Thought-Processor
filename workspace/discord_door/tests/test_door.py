"""Discord door against a fake Communication Bridge (same API shape as the real one)."""
import json
import os
import sys
import tempfile
import threading
import time
import unittest
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import door as door_mod  # noqa: E402

KEY = "k" * 40


class FakeBridge:
    def __init__(self):
        self.state, self.paused, self.msgs, self.posts, self.seen_keys, self.seq = "connected", False, [], [], {}, 0
        fb = self

        class H(BaseHTTPRequestHandler):
            def log_message(self, *a):
                pass

            def send(self, code, body):
                b = json.dumps(body).encode()
                self.send_response(code)
                self.send_header("Content-Type", "application/json")
                self.end_headers()
                self.wfile.write(b)

            def authed(self):
                return self.headers.get("Authorization") == "Bearer " + KEY

            def do_GET(self):
                if self.path.startswith("/api/status"):
                    return self.send(200, {"discord_state": fb.state, "paused": fb.paused, "mode": "manual",
                                           "discord_configured": True})
                if not self.authed():
                    return self.send(401, {"error": "no"})
                after = int(self.path.split("after=")[1].split("&")[0])
                out = [m for m in fb.msgs if m["rev"] > after]
                return self.send(200, {"messages": out, "cursor": max([after] + [m["rev"] for m in out]),
                                       "more": False, "history_epoch": 0})

            def do_POST(self):
                if not self.authed():
                    return self.send(401, {"error": "no"})
                body = json.loads(self.rfile.read(int(self.headers["Content-Length"])))
                k = self.headers.get("Idempotency-Key")
                if k in fb.seen_keys:
                    return self.send(200, {**fb.seen_keys[k], "replayed": True})
                if fb.state != "connected" or fb.paused:
                    return self.send(503, {"error": "Discord unavailable"})
                if self.path == "/api/replies" and not any(m["id"] == body["reply_to_id"] for m in fb.msgs):
                    return self.send(404, {"error": "unknown message"})
                fb.seq += 1
                m = {"id": f"m_{fb.seq:020x}", "seq": fb.seq, "rev": fb.seq, "direction": "outbound",
                     "source": "api", "delivery": "queued", "text": body["text"]}
                fb.msgs.append(m)
                fb.posts.append((self.path, body))
                r = {"receipt_id": m["id"], "delivery": "queued", "message": dict(m)}
                fb.seen_keys[k] = r
                return self.send(201, r)

        self.srv = ThreadingHTTPServer(("127.0.0.1", 0), H)
        threading.Thread(target=self.srv.serve_forever, daemon=True).start()
        self.url = f"http://127.0.0.1:{self.srv.server_address[1]}"

    def owner_says(self, text):
        self.seq += 1
        self.msgs.append({"id": f"m_{self.seq:020x}", "seq": self.seq, "rev": self.seq, "direction": "inbound",
                          "source": "discord", "text": text, "source_time": "now"})

    def deliver_all(self):
        for m in self.msgs:
            if m["direction"] == "outbound" and m["delivery"] == "queued":
                self.seq += 1
                m.update(delivery="delivered", rev=self.seq)


class DoorTest(unittest.TestCase):
    def setUp(self):
        self.tmp = Path(tempfile.mkdtemp())
        (self.tmp / "api-key").write_text(KEY)
        self.fb = FakeBridge()
        self.box = self.tmp / "discord"
        self.d = door_mod.Door(self.box, self.fb.url, self.tmp / "api-key")
        door_mod.SETTLE = 0
        door_mod.MIN_GAP = 0

    def tearDown(self):
        self.fb.srv.shutdown()

    def put(self, name, content):
        (self.box / "outbox" / name).write_text(content)

    def status(self):
        return json.loads((self.box / "status.json").read_text())

    def test_say_then_delivered_receipt(self):
        self.put("hello.txt", "Hi Dustin\n")
        self.d.tick()
        self.assertEqual(self.fb.posts, [("/api/messages", {"text": "Hi Dustin", "destination": "discord"})])
        self.assertFalse((self.box / "outbox/hello.txt").exists())
        self.assertEqual(json.loads((self.box / "sent/hello.json").read_text())["delivery"], "queued")
        self.fb.deliver_all()
        self.d.tick()
        self.assertEqual(json.loads((self.box / "sent/hello.json").read_text())["delivery"], "delivered")
        self.assertTrue(self.status()["connected"])

    def test_owner_message_lands_in_inbox_once_and_reply_goes_back(self):
        self.fb.owner_says("can you hear me?")
        self.d.tick()
        self.d.tick()
        inbox = list((self.box / "inbox").glob("*.json"))
        self.assertEqual(len(inbox), 1)
        m = json.loads(inbox[0].read_text())
        self.assertEqual((m["text"], m["from"]), ("can you hear me?", "Dustin"))
        self.put("ans.json", json.dumps({"text": "yes", "reply_to": m["id"]}))
        self.d.tick()
        self.assertEqual(self.fb.posts[-1], ("/api/replies", {"reply_to_id": m["id"], "text": "yes"}))

    def test_refusals_never_reach_the_bridge(self):
        self.put("long.txt", "x" * 1901)
        self.put("empty.txt", "   ")
        self.put("bad.json", "{not json")
        self.put("badref.json", json.dumps({"text": "hi", "reply_to": "../../etc"}))
        self.put("bad name!.txt", "hi")
        for _ in range(6):
            self.d.tick()
        self.assertEqual(self.fb.posts, [])
        failed = {p.name for p in (self.box / "failed").glob("*.json")}
        self.assertEqual(failed, {"long.json", "empty.json", "bad.json", "badref.json", "bad name!.json"})
        self.assertIn("too long", json.loads((self.box / "failed/long.json").read_text())["error"])

    def test_waits_while_not_connected_then_sends(self):
        self.fb.state = "needs_attention"
        self.put("later.txt", "when you're back")
        self.d.tick()
        self.assertEqual(self.fb.posts, [])
        self.assertTrue((self.box / "outbox/later.txt").exists())
        self.assertIn("isn't connected", self.status()["waiting"])
        self.assertFalse(self.status()["connected"])
        self.fb.state = "connected"
        self.d.tick()
        self.assertEqual(len(self.fb.posts), 1)

    def test_rate_limit(self):
        door_mod.MIN_GAP = 3.0
        self.put("a.txt", "one")
        self.d.tick()
        time.sleep(0.01)
        self.put("b.txt", "two")
        self.d.tick()
        self.assertEqual(len(self.fb.posts), 1)
        self.assertIn("rate limit", self.status()["waiting"])
        self.assertTrue((self.box / "outbox/b.txt").exists())

    def test_tmp_files_are_ignored_and_bridge_offline_is_reported(self):
        self.put("half.tmp", "still writing")
        self.d.tick()
        self.assertEqual(self.fb.posts, [])
        self.fb.srv.shutdown()
        self.fb.srv.server_close()
        self.d.tick()
        self.assertEqual(self.status()["discord_state"], "bridge_offline")

    def test_status_has_no_secret(self):
        self.d.tick()
        self.assertNotIn(KEY, (self.box / "status.json").read_text())
        self.assertNotIn(KEY, (self.box / ".door-state.json").read_text())


if __name__ == "__main__":
    unittest.main()
