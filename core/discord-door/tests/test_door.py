"""Discord door v2 against a fake Communication Bridge and a fake Identity (Thought-Processor #127).

Both fakes follow the documented APIs: the Bridge's docs/API.md (Communication-Bridge 1938f45 with
#3's destination check) and identity's docs/DISCORD_REPLY_CONTRACT.md (#46, as accepted in identity
#47 at a67064380cb55d1b55f9e8454cf2f60d12c6b306: check mode refused, single consuming redeem). Every identifier is a
fixture value. Nothing here talks to Discord, a real Bridge or a real Identity.
"""
import hashlib
import json
import os
import socket
import sys
import tempfile
import threading
import time
import unittest
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path

HERE = Path(__file__).resolve()
sys.path.insert(0, str(HERE.parents[1]))
import door as D  # noqa: E402

BRIDGE_KEY = "fixture-bridge-key-" + "b" * 24
DOOR_KEY = "fixture-door-identity-key-" + "d" * 16
CHAN = "1100000000000000001"
OWNER = "2100000000000000001"
OTHER = "2100000000000000009"
OTHER_CHAN = "1100000000000000002"


def iso(t):
    return D.iso(t)


class _Server:
    def __init__(self, handler):
        self.srv = ThreadingHTTPServer(("127.0.0.1", 0), handler)
        self.srv.daemon_threads = True
        threading.Thread(target=self.srv.serve_forever, daemon=True).start()
        self.url = f"http://127.0.0.1:{self.srv.server_address[1]}"

    def close(self):
        self.srv.shutdown()
        self.srv.server_close()


def _handler(fake):
    class H(BaseHTTPRequestHandler):
        def log_message(self, *a):
            pass

        def send(self, code, body):
            b = json.dumps(body).encode()
            self.send_response(code)
            self.send_header("Content-Type", "application/json")
            self.send_header("Content-Length", str(len(b)))
            self.end_headers()
            try:
                self.wfile.write(b)
            except (BrokenPipeError, ConnectionResetError):
                pass                                       # the door gave up first (timeout plans)

        def body(self):
            n = int(self.headers.get("Content-Length") or 0)
            return json.loads(self.rfile.read(n) or b"{}")

        def do_GET(self):
            fake.get(self)

        def do_POST(self):
            fake.post(self)
    return H


class FakeBridge:
    """The Bridge's local API: status, the message feed, and POST /api/replies."""

    def __init__(self):
        self.lock = threading.Lock()
        self.discord_state, self.paused, self.mode = "connected", False, "manual"
        self.msgs, self.rev, self.seq, self.epoch = {}, 0, 0, 0
        self.idem, self.reply_posts, self.message_posts, self.plan = {}, [], 0, []
        self.server = _Server(_handler(self))

    def _bump(self, m):
        self.rev += 1
        m["rev"] = self.rev

    def _new(self, **kw):
        self.seq += 1
        m = {"schema": "communication.message.v1", "id": "m_" + os.urandom(10).hex(), "seq": self.seq,
             "received_at": iso(time.time()), "replies": [], **kw}
        self.msgs[m["id"]] = m
        self._bump(m)
        return m

    def inbound(self, text="Hello TASS", sender=OWNER, channel=CHAN, **kw):
        with self.lock:
            fields = {"source": "discord", "target": "bridge", "direction": "inbound", "kind": "message",
                      "source_message_id": str(1562034986897375235 + self.seq), "source_channel_id": channel,
                      "sender_id": sender, "sender_kind": "owner", "conversation_id": "discord:" + channel,
                      "source_time": iso(time.time()), "text": text, "delivery": "received",
                      "origin_trace": ["discord"]}
            fields.update(kw)
            return dict(self._new(**fields))

    def outbound_reply(self, parent_id, text, kind):
        """A reply made by someone else: the owner's page (manual_reply) or another API client (api_reply)."""
        with self.lock:
            p = self.msgs[parent_id]
            m = self._new(source="bridge", target="discord", direction="outbound", kind=kind,
                          sender_kind="api" if kind == "api_reply" else "owner", sender_id="x",
                          conversation_id=p["conversation_id"], reply_to_id=parent_id,
                          reply_to_source_id=p["source_message_id"], text=text, delivery="queued",
                          origin_trace=["discord", "bridge"])
            p["replies"].append(m["id"])
            self._bump(p)
            return dict(m)

    def deliver(self, mid, state, detail="", external_id=""):
        with self.lock:
            m = self.msgs[mid]
            m["delivery"], m["delivery_detail"] = state, detail
            if external_id:
                m["external_id"] = external_id
            self._bump(m)

    def outbound(self):
        with self.lock:
            return [dict(m) for m in self.msgs.values() if m["direction"] == "outbound"]

    def get(self, h):
        if h.path == "/api/status":
            with self.lock:
                return h.send(200, {"service": "communication-bridge", "discord_state": self.discord_state,
                                    "paused": self.paused, "mode": self.mode})
        if h.headers.get("Authorization") != "Bearer " + BRIDGE_KEY:
            return h.send(401, {"error": "no credential"})
        if not h.path.startswith("/api/messages?"):
            return h.send(404, {"error": "unknown"})
        q = dict(kv.split("=", 1) for kv in h.path.split("?", 1)[1].split("&"))
        after, limit = int(q.get("after", 0)), int(q.get("limit", 100))
        with self.lock:
            changed = sorted((m for m in self.msgs.values() if m["rev"] > after), key=lambda m: m["rev"])
            out = [json.loads(json.dumps(m)) for m in changed[:limit]]
            cursor = out[-1]["rev"] if out else max(after, 0)
            return h.send(200, {"messages": out, "cursor": cursor, "more": len(changed) > limit,
                                "history_epoch": self.epoch})

    def post(self, h):
        if h.headers.get("Authorization") != "Bearer " + BRIDGE_KEY:
            return h.send(401, {"error": "no credential"})
        body = h.body()
        if h.path == "/api/messages":
            with self.lock:
                self.message_posts += 1
            return h.send(500, {"error": "the door must never use /api/messages"})
        if h.path != "/api/replies":
            return h.send(404, {"error": "unknown"})
        key = h.headers.get("Idempotency-Key")
        with self.lock:
            self.reply_posts.append({"key": key, **body})
            plan = self.plan.pop(0) if self.plan else "ok"
            if key in self.idem:
                return h.send(200, {**self.idem[key], "replayed": True})
            if plan in ("503", "404", "409", "400"):
                return h.send(int(plan), {"error": "refused by the fake (plan " + plan + ")"})
            if self.paused or self.discord_state != "connected":
                return h.send(503, {"error": "Discord unavailable or paused"})
            p = self.msgs.get(body.get("reply_to_id"))
            if p is None:
                return h.send(404, {"error": "message is not in current history"})
            m = self._new(source="bridge", target="discord", direction="outbound", kind="api_reply",
                          sender_kind="api", sender_id="api", conversation_id=p["conversation_id"],
                          source_channel_id=p["conversation_id"], reply_to_id=p["id"],
                          reply_to_source_id=p["source_message_id"], text=body["text"].strip(),
                          delivery="queued", origin_trace=p["origin_trace"] + ["bridge"])
            p["replies"].append(m["id"])
            self._bump(p)
            receipt = {"receipt_id": m["id"], "delivery": "queued", "replayed": False, "message": dict(m)}
            if plan == "mismatch":
                receipt["message"] = {**m, "conversation_id": "discord:" + OTHER_CHAN}
            self.idem[key] = receipt
        if plan == "500":
            return h.send(500, {"error": "boom"})
        if plan == "drop":
            h.close_connection = True
            return
        if plan == "timeout":
            time.sleep(1.0)
        return h.send(201, receipt)


class FakeIdentity:
    """POST /v1/decisions/redeem for discord.reply, per docs/DISCORD_REPLY_CONTRACT.md (identity #46)."""

    def __init__(self):
        self.lock = threading.Lock()
        self.decisions, self.requests, self.plan = {}, [], []
        self.granted, self.ttl, self.clock = True, 120.0, time.time
        self.server = _Server(_handler(self))

    def mint(self, bind, action="discord.reply"):
        with self.lock:
            did = "dec_" + os.urandom(8).hex()
            self.decisions[did] = {"binding": bind, "action": action, "used": False,
                                   "expires": self.clock() + self.ttl}
            return did

    def redeems(self):
        with self.lock:
            return list(self.requests)

    def get(self, h):
        h.send(404, {"error": "unknown"})

    def post(self, h):
        if h.path != "/v1/decisions/redeem":
            return h.send(404, {"error": "unknown"})
        if h.headers.get("Authorization") != "Bearer " + DOOR_KEY:
            return h.send(401, {"error": "unknown service key"})
        body = h.body()
        with self.lock:
            self.requests.append(body)
            plan = self.plan.pop(0) if self.plan else "ok"
            did = body.get("decision_id")
            out = {"allow": False, "decision_id": "dec_" + os.urandom(8).hex(), "for_decision": did,
                   "reason": "", "entity_id": "", "service": "discord-door",
                   "mode": "check" if body.get("check") else "redeem"}
            d = self.decisions.get(did)
            if body.get("check"):
                out["reason"] = "check mode is not allowed for discord.reply"
            elif d is None:
                out["reason"] = "no such bound decision"
            elif d["action"] != body.get("action"):
                out["reason"] = "the decision is for another action"
            elif d["binding"] != body.get("binding"):
                out["reason"] = "the binding does not match the decision"
            elif d["used"]:
                out["reason"] = "the decision was already used"
            elif self.clock() > d["expires"]:
                out["reason"] = "the decision expired"
            elif not self.granted:
                d["used"] = True
                out["reason"] = "discord.reply is not granted"
            else:
                d["used"] = True
                out.update(allow=True, reason="unlocked; discord.reply is granted", entity_id="ent_fixture")
        if plan == "500":
            return h.send(500, {"error": "boom"})
        if plan == "no-mode":
            out.pop("mode")
        if plan == "wrong-for":
            out["for_decision"] = "dec_" + "0" * 16
        if plan == "wrong-service":
            out["service"] = "engine"
        if plan == "mode-check":
            out["mode"] = "check"
        if plan == "allow-string":
            out["allow"] = "true"
        return h.send(200, out)


def closed_port_url():
    s = socket.socket()
    s.bind(("127.0.0.1", 0))
    port = s.getsockname()[1]
    s.close()
    return f"http://127.0.0.1:{port}"


class Rig(unittest.TestCase):
    def setUp(self):
        self.tmp = Path(tempfile.mkdtemp())
        self.box, self.state = self.tmp / "mailbox", self.tmp / "state"
        (self.tmp / "keys").mkdir()
        self.bridge_key = self.tmp / "keys" / "bridge"
        self.bridge_key.write_text(BRIDGE_KEY)
        self.id_key = self.tmp / "keys" / "identity"
        self.id_key.write_text(DOOR_KEY)
        self.fb, self.fi = FakeBridge(), FakeIdentity()
        self.addCleanup(self.fb.server.close)
        self.addCleanup(self.fi.server.close)
        self.offset = 0.0
        self.door = self.open_door()

    def tearDown(self):
        try:
            self.door.close()
        except (ValueError, OSError):
            pass

    def cfg(self, **kw):
        a = dict(mailbox=self.box, state=self.state, bridge=self.fb.server.url, key_file=self.bridge_key,
                 identity=self.fi.server.url, identity_key_file=self.id_key, channel_id=CHAN, owner_id=OWNER,
                 bridge_timeout=0.5, identity_timeout=0.5)
        a.update(kw)
        return D.Config(**a)

    def open_door(self, **kw):
        return D.Door(self.cfg(**kw), now=lambda: time.time() + self.offset)

    def restart(self, **kw):
        self.door.close()
        self.door = self.open_door(**kw)

    # --- what Core does -------------------------------------------------------------------------
    def inbox(self, mid):
        files = list((self.box / "inbox").glob(f"*-{mid}.json"))
        return json.loads(files[0].read_text()) if files else None

    def core_binding(self, msg, text):
        """Core's binding, computed from the inbox file alone (independent of the door's code)."""
        th = hashlib.sha256(text.encode("utf-8")).hexdigest()
        lines = ["tass.discord.reply/1", "discord-door", msg["id"], msg["source_message_id"], msg["sender_id"],
                 msg["channel_id"], "discord:" + msg["channel_id"], th]
        return hashlib.sha256("\n".join(lines).encode()).hexdigest()

    def write_reply(self, mid, text, decision=None, age=5.0, raw=None, name=None, **override):
        inbox = self.inbox(mid)
        if decision is None:
            decision = self.fi.mint(self.core_binding(inbox, text))
        body = {"schema": "tass.discord.reply/2", "reply_to": mid, "text": text, "cycle_id": "cy_fixture",
                "decision_id": decision, **override}
        p = self.box / "replies" / (name or f"reply-{mid}.json")
        tmp = p.parent / (".tmp-" + p.name)
        tmp.write_bytes(raw if raw is not None else json.dumps(body).encode())
        os.replace(tmp, p)
        t = time.time() - age
        os.utime(p, (t, t))
        return decision

    def heard(self, **kw):
        m = self.fb.inbound(**kw)
        self.door.tick()
        return m

    def receipt(self, mid):
        p = self.box / "receipts" / f"{mid}.json"
        return json.loads(p.read_text()) if p.exists() else None

    def rejected(self):
        return [json.loads(p.read_text()) for p in sorted((self.box / "rejected").glob("*.json"))]

    def status(self):
        return json.loads((self.box / "status.json").read_text())

    def assert_nothing_sent(self):
        self.assertEqual(self.fb.reply_posts, [])
        self.assertEqual(self.fb.message_posts, 0)

    def assert_no_check_mode(self):
        for r in self.fi.redeems():
            self.assertNotIn("check", r)
            self.assertEqual(set(r), {"decision_id", "action", "binding"})
            self.assertEqual(r["action"], "discord.reply")


class TestBinding(Rig):
    def test_vector_matches_both_contract_documents(self):
        rec = {"id": "m_4d442e5526b608cfe449", "source_message_id": "1562034986897375235",
               "sender_id": "2100000000000000001", "channel_id": "1100000000000000001"}
        want = "f5ab28d15e23a3850274a3604fb999d433bb3980d9e4ff970bec8029d095a25c"
        self.assertEqual(D.binding(rec, "TEST FIXTURE reply: hello from TASS"), want)
        doc = (HERE.parents[2] / "docs" / "DISCORD_MAILBOX_CONTRACT_V2.md").read_text()
        self.assertIn(want, doc)
        self.assertIn("412238e0e55005100cbdeb1137fcec8122648f95aa03fe30bce39fa946ca5a6d", doc)
        # Every bound field changes it: message, Discord message, sender, channel, content.
        seen = {want}
        for k in rec:
            seen.add(D.binding({**rec, k: rec[k][:-1] + "7"}, "TEST FIXTURE reply: hello from TASS"))
        seen.add(D.binding(rec, "TEST FIXTURE reply: hello from TASS!"))
        self.assertEqual(len(seen), 6)
        with self.assertRaises(D.DoorError):
            D.binding({**rec, "sender_id": "1\n2"}, "x")


class TestReactiveReply(Rig):
    def test_one_owner_message_gets_exactly_one_authorized_reply(self):
        m = self.heard(text="Hello TASS")
        inbox = self.inbox(m["id"])
        self.assertEqual(inbox["schema"], "tass.discord.inbox/2")
        for k_inbox, k_bridge in [("id", "id"), ("source_message_id", "source_message_id"),
                                  ("channel_id", "source_channel_id"), ("sender_id", "sender_id"),
                                  ("sender_kind", "sender_kind"), ("conversation_id", "conversation_id"),
                                  ("text", "text"), ("seq", "seq")]:
            self.assertEqual(inbox[k_inbox], m[k_bridge], k_inbox)
        self.assertTrue(inbox["answerable"])
        self.assertNotIn("from", inbox)                    # no hardcoded sender name any more
        text = "Hi!\nHere is my answer: 2 + 2 = 4 🙂"
        dec = self.write_reply(m["id"], text)
        self.door.tick()
        self.assertEqual(len(self.fb.reply_posts), 1)
        post = self.fb.reply_posts[0]
        self.assertEqual((post["reply_to_id"], post["text"], post["key"]), (m["id"], text, "tass-door-" + m["id"]))
        self.assertEqual(self.fi.redeems(), [{"decision_id": dec, "action": "discord.reply",
                                              "binding": self.core_binding(inbox, text)}])
        r = self.receipt(m["id"])
        self.assertEqual((r["state"], r["final"], r["cycle_id"], r["decision_id"]), ("queued", False, "cy_fixture", dec))
        self.assertEqual(r["binding"], self.core_binding(inbox, text))
        self.assertFalse((self.box / "replies" / f"reply-{m['id']}.json").exists())
        out = self.fb.outbound()[0]
        self.assertEqual(r["bridge_message_id"], out["id"])
        self.fb.deliver(out["id"], "delivered", external_id="1562035000000000001")
        self.door.tick()
        r = self.receipt(m["id"])
        self.assertEqual((r["state"], r["final"], r["discord_message_id"]), ("delivered", True, "1562035000000000001"))
        for _ in range(3):
            self.door.tick()
        self.assertEqual(len(self.fb.reply_posts), 1)
        self.assertEqual(self.fb.message_posts, 0)
        self.assert_no_check_mode()
        self.assertTrue(self.status()["ready"])

    def test_a_second_reply_to_the_same_message_is_refused_without_asking_identity(self):
        m = self.heard()
        self.write_reply(m["id"], "first answer")
        self.door.tick()
        self.write_reply(m["id"], "second answer")
        self.door.tick()
        self.assertEqual(len(self.fb.reply_posts), 1)
        self.assertEqual(len(self.fi.redeems()), 1)
        self.assertIn("already has a reply", self.rejected()[-1]["reason"])

    def test_the_same_file_reappearing_after_a_crash_is_not_sent_again(self):
        m = self.heard()
        dec = self.write_reply(m["id"], "answer")
        self.door.tick()
        self.write_reply(m["id"], "answer", decision=dec)   # the file Core wrote, seen again
        self.restart()
        self.door.tick()
        self.assertEqual(len(self.fb.reply_posts), 1)
        self.assertEqual(len(self.fi.redeems()), 1)
        self.assertEqual(self.rejected(), [])

    def test_text_is_posted_exactly_or_refused(self):
        m = self.heard()
        bad = ["", " leading", "trailing ", "trailing newline\n", "　ideographic space", "tab\x00nul",
               "bell\x07", "del\x7f", "c1\x85", "x" * 1901]
        for i, t in enumerate(bad):
            self.write_reply(m["id"], t, decision="dec_" + f"{i:016x}")
            self.door.tick()
        raw = b'{"schema":"tass.discord.reply/2","reply_to":"%s","text":"\\ud800","cycle_id":"c","decision_id":"dec_00000000000000aa"}' % m["id"].encode()
        self.write_reply(m["id"], "", raw=raw)
        self.door.tick()
        self.write_reply(m["id"], 7, decision="dec_00000000000000ab")
        self.door.tick()
        self.assertEqual(len(self.rejected()), len(bad) + 2)
        self.assert_nothing_sent()
        self.assertEqual(self.fi.redeems(), [])
        self.assertIsNone(self.receipt(m["id"]))          # nothing claimed: Core may still answer correctly
        ok = "😀" * 1900 + ""                              # 1,900 characters (not bytes) is allowed
        self.write_reply(m["id"], ok)
        self.door.tick()
        self.assertEqual([p["text"] for p in self.fb.reply_posts], [ok])

    def test_malformed_reply_files_never_reach_identity_or_the_bridge(self):
        m = self.heard()
        mid = m["id"]
        self.write_reply(mid, "a", name="answer.json")
        self.write_reply(mid, "a", name="reply-m_00000000000000000000.json")
        self.write_reply(mid, "a", extra="x")
        self.door.tick(); self.door.tick(); self.door.tick()
        self.write_reply(mid, "a", raw=b"not json")
        self.door.tick()
        self.write_reply(mid, "a", raw=json.dumps({"schema": "tass.discord.reply/1", "reply_to": mid, "text": "a",
                                                   "cycle_id": "c", "decision_id": "dec_00000000000000aa"}).encode())
        self.door.tick()
        self.write_reply(mid, "a", decision="not-a-decision")
        self.door.tick()
        self.write_reply(mid, "a", raw=json.dumps({"schema": "tass.discord.reply/2", "reply_to": mid, "text": "a",
                                                   "decision_id": "dec_00000000000000aa"}).encode())
        self.door.tick()
        target = self.tmp / "keys" / "identity"
        link = self.box / "replies" / f"reply-{mid}.json"
        os.symlink(target, link)
        t = time.time() - 5
        os.utime(link, (t, t), follow_symlinks=False)
        self.door.tick()
        (self.box / "replies" / "reply-m_11111111111111111111.json").mkdir()
        t = time.time() - 5
        os.utime(self.box / "replies" / "reply-m_11111111111111111111.json", (t, t))
        self.door.tick()
        self.assertEqual(len(self.rejected()), 9)
        self.door.tick()                                   # the directory is rejected once, then left alone
        self.assertEqual(len(self.rejected()), 9)
        self.assertTrue(target.exists())
        self.assert_nothing_sent()
        self.assertEqual(self.fi.redeems(), [])

    def test_a_fresh_file_is_not_read_before_it_settles(self):
        m = self.heard()
        self.write_reply(m["id"], "answer", age=0.0)
        self.door.tick()
        self.assertEqual(self.fi.redeems(), [])
        p = self.box / "replies" / f"reply-{m['id']}.json"
        t = time.time() - 1
        os.utime(p, (t, t))
        self.door.tick()
        self.assertEqual(len(self.fb.reply_posts), 1)


class TestInbound(Rig):
    def test_only_the_owner_in_the_configured_channel_is_heard(self):
        bad = [dict(sender=OTHER), dict(channel=OTHER_CHAN), dict(conversation_id="discord:" + OTHER_CHAN),
               dict(sender_kind="api"), dict(kind="echo_test"), dict(origin_trace=["discord", "bridge"]),
               dict(source_message_id="not-a-snowflake"), dict(text=None), dict(source_channel_id="local")]
        for kw in bad:
            self.fb.inbound(**kw)
        self.door.tick()
        self.door.tick()
        self.assertEqual(list((self.box / "inbox").iterdir()), [])
        self.assertEqual(list((self.state / "inbound").iterdir()), [])
        self.assertEqual(self.status()["refused_inbound"], len(bad))

    def test_messages_that_must_not_be_answered_are_recorded_but_refused(self):
        cases = {
            "bridge catch-up": dict(catch_up=True),
            "before the door started": dict(received_at=iso(time.time() - 3600)),
            "unreadable receive time": dict(received_at="yesterday"),
        }
        ids = {}
        for name, kw in cases.items():
            ids[name] = self.fb.inbound(**kw)["id"]
        self.door.tick()
        self.fb.mode = "echo"
        ids["bridge in echo mode"] = self.heard()["id"]
        self.fb.mode = "manual"
        for name, mid in ids.items():
            inbox = self.inbox(mid)
            self.assertFalse(inbox["answerable"], name)
            self.assertTrue(inbox["not_answerable_reason"], name)
            self.write_reply(mid, "answer")
            self.door.tick()
        self.assert_nothing_sent()
        self.assertEqual(self.fi.redeems(), [])
        self.assertEqual(len(self.rejected()), len(ids))
        self.assertTrue(all("must not be answered" in r["reason"] for r in self.rejected()))

    def test_unknown_message_is_refused(self):
        self.heard()
        p = self.box / "replies" / "reply-m_22222222222222222222.json"
        p.write_text(json.dumps({"schema": "tass.discord.reply/2", "reply_to": "m_22222222222222222222",
                                 "text": "a", "cycle_id": "c", "decision_id": "dec_00000000000000aa"}))
        t = time.time() - 5
        os.utime(p, (t, t))
        self.door.tick()
        self.assert_nothing_sent()
        self.assertIn("no owner message", self.rejected()[0]["reason"])

    def test_the_door_trusts_its_own_record_not_the_shared_inbox(self):
        m = self.heard()
        inbox_path = next((self.box / "inbox").glob(f"*-{m['id']}.json"))
        forged = {**json.loads(inbox_path.read_text()), "channel_id": OTHER_CHAN, "sender_id": OTHER}
        inbox_path.write_text(json.dumps(forged))          # something rewrote the shared inbox
        text = "answer"
        dec = self.fi.mint(self.core_binding(forged, text))
        self.write_reply(m["id"], text, decision=dec)
        self.door.tick()
        self.assert_nothing_sent()
        self.assertEqual(self.receipt(m["id"])["state"], "not_sent")
        self.assertIn("binding does not match", self.receipt(m["id"])["detail"])


class TestIdentity(Rig):
    def refused(self, mid, detail_part):
        self.door.tick()
        self.assert_nothing_sent()
        r = self.receipt(mid)
        self.assertEqual((r["state"], r["final"]), ("not_sent", True))
        self.assertIn(detail_part, r["detail"])
        self.assertTrue(r["detail"].startswith("Identity:"))
        # The message is claimed: no later reply can be sent for it.
        self.write_reply(mid, "try again")
        self.door.tick()
        self.assert_nothing_sent()
        self.assert_no_check_mode()

    def test_decision_for_other_text_is_refused(self):
        m = self.heard()
        dec = self.fi.mint(self.core_binding(self.inbox(m["id"]), "what Core authorized"))
        self.write_reply(m["id"], "something else", decision=dec)
        self.refused(m["id"], "binding does not match")

    def test_expired_decision_is_refused(self):
        m = self.heard()
        self.fi.ttl = -1
        self.write_reply(m["id"], "answer")
        self.refused(m["id"], "expired")

    def test_reused_decision_is_refused(self):
        m = self.heard()
        bind = self.core_binding(self.inbox(m["id"]), "answer")
        dec = self.fi.mint(bind)
        self.fi.decisions[dec]["used"] = True
        self.write_reply(m["id"], "answer", decision=dec)
        self.refused(m["id"], "already used")

    def test_revoked_grant_is_refused(self):
        m = self.heard()
        self.write_reply(m["id"], "answer")
        self.fi.granted = False
        self.refused(m["id"], "not granted")

    def test_unknown_decision_is_refused(self):
        m = self.heard()
        self.write_reply(m["id"], "answer", decision="dec_" + "f" * 16)
        self.refused(m["id"], "no such bound decision")

    def test_identity_down_sends_nothing(self):
        self.restart(identity=closed_port_url())
        m = self.heard()
        self.write_reply(m["id"], "answer")
        self.door.tick()
        self.assert_nothing_sent()
        self.assertEqual(self.receipt(m["id"])["detail"], "Identity: Identity is not reachable")

    def test_missing_door_key_sends_nothing(self):
        m = self.heard()
        self.write_reply(m["id"], "answer")
        self.id_key.unlink()
        self.refused(m["id"], "key file is missing")

    def test_answers_that_are_not_a_consuming_redemption_send_nothing(self):
        for plan, detail in [("500", "HTTP 500"), ("no-mode", "not a redemption"),
                             ("mode-check", "not a redemption"), ("wrong-for", "not a redemption"),
                             ("wrong-service", "not a redemption"), ("allow-string", "refused")]:
            with self.subTest(plan=plan):
                m = self.heard()
                self.write_reply(m["id"], "answer " + plan)
                self.fi.plan = [plan]
                self.door.tick()
                self.assert_nothing_sent()
                r = self.receipt(m["id"])
                self.assertEqual(r["state"], "not_sent")
                self.assertIn(detail, r["detail"])
        self.assert_no_check_mode()


class TestBridgeAndDelivery(Rig):
    def test_nothing_is_redeemed_or_sent_while_the_bridge_is_not_ready(self):
        m = self.heard()
        self.write_reply(m["id"], "answer")
        for setup, why in [(dict(paused=True), "paused"), (dict(mode="echo"), "manual mode"),
                           (dict(discord_state="reconnecting"), "not connected")]:
            for k, v in setup.items():
                setattr(self.fb, k, v)
            self.door.tick()
            self.assertEqual(self.fi.redeems(), [])
            self.assertIn(why, self.status()["not_ready"])
            self.assertFalse(self.status()["ready"])
            self.fb.paused, self.fb.mode, self.fb.discord_state = False, "manual", "connected"
        self.assertTrue((self.box / "replies" / f"reply-{m['id']}.json").exists())
        self.door.tick()
        self.assertEqual(len(self.fb.reply_posts), 1)

    def test_a_reply_that_waits_too_long_is_refused_before_identity(self):
        m = self.heard()
        self.write_reply(m["id"], "answer", age=D.MAX_WAIT + 1)
        self.fb.paused = True
        self.door.tick()
        self.assertEqual(self.fi.redeems(), [])
        self.assertIn("decision would expire", self.rejected()[0]["reason"])

    def test_definite_bridge_refusals_are_not_sent_and_never_retried(self):
        for plan in ("503", "404", "409", "400"):
            with self.subTest(plan=plan):
                self.offset += D.MIN_GAP + 0.1                # past the previous post's rate limit
                m = self.heard()
                self.write_reply(m["id"], "answer " + plan)
                self.fb.plan = [plan]
                before = len(self.fb.reply_posts)
                self.door.tick()
                self.door.tick()
                self.assertEqual(len(self.fb.reply_posts), before + 1)
                r = self.receipt(m["id"])
                self.assertEqual((r["state"], r["final"]), ("not_sent", True))
                self.assertIn("HTTP " + plan, r["detail"])

    def test_bridge_down_before_posting_is_not_sent(self):
        m = self.heard()
        self.write_reply(m["id"], "answer")
        url = self.fb.server.url
        self.door.cfg.bridge = url
        # The status call succeeds; the post goes to a closed port.
        orig = self.door.bridge_call

        def call(method, path, *a, **kw):
            if path == "/api/replies":
                self.door.cfg.bridge = closed_port_url()
                try:
                    return orig(method, path, *a, **kw)
                finally:
                    self.door.cfg.bridge = url
            return orig(method, path, *a, **kw)
        self.door.bridge_call = call
        self.door.tick()
        self.assertEqual(self.receipt(m["id"])["state"], "not_sent")
        self.assertIn("not reachable", self.receipt(m["id"])["detail"])

    def test_uncertain_posts_are_never_retried_and_are_recognised_later(self):
        for plan in ("500", "drop", "timeout"):
            with self.subTest(plan=plan):
                self.offset += D.MIN_GAP + 0.1
                m = self.heard()
                self.write_reply(m["id"], "answer " + plan)
                self.fb.plan = [plan]
                before = len(self.fb.reply_posts)
                self.door.tick()
                time.sleep(0.6 if plan == "timeout" else 0)
                r = self.receipt(m["id"])
                self.assertEqual((r["state"], r["final"]), ("uncertain", False))
                # The Bridge did create it; when it shows up, it is matched as ours (not another responder).
                out = [o for o in self.fb.outbound() if o["reply_to_id"] == m["id"]][0]
                self.fb.deliver(out["id"], "delivered", external_id="1562035000000000077")
                for _ in range(3):
                    self.door.tick()
                self.assertEqual(len(self.fb.reply_posts), before + 1)
                r = self.receipt(m["id"])
                self.assertEqual((r["state"], r["final"], r["bridge_message_id"]), ("delivered", True, out["id"]))
                self.assertEqual(self.door.blocked, "")

    def test_bridge_failure_and_destination_check_are_not_sent(self):
        m = self.heard()
        self.write_reply(m["id"], "answer")
        self.door.tick()
        out = self.fb.outbound()[0]
        self.fb.deliver(out["id"], "failed", "not sent: destination changed: queued for channel …0001, "
                                             "configured channel is now …0002")
        self.door.tick()
        r = self.receipt(m["id"])
        self.assertEqual((r["state"], r["final"]), ("not_sent", True))
        self.assertIn("not sent: destination changed", r["detail"])
        self.fb.deliver(out["id"], "delivered", external_id="1")   # a later report cannot change a final result
        self.door.tick()
        self.assertEqual(self.receipt(m["id"])["state"], "not_sent")

    def test_bridge_uncertain_is_final(self):
        m = self.heard()
        self.write_reply(m["id"], "answer")
        self.door.tick()
        self.fb.deliver(self.fb.outbound()[0]["id"], "uncertain", "timed out; not retried")
        self.door.tick()
        r = self.receipt(m["id"])
        self.assertEqual((r["state"], r["final"]), ("uncertain", True))
        self.assertEqual(len(self.fb.reply_posts), 1)

    def test_no_final_state_becomes_uncertain_then_is_corrected(self):
        m = self.heard()
        self.write_reply(m["id"], "answer")
        self.door.tick()
        self.offset = D.DELIVERY_WAIT + 5
        self.door.tick()
        r = self.receipt(m["id"])
        self.assertEqual((r["state"], r["final"]), ("uncertain", False))
        self.fb.deliver(self.fb.outbound()[0]["id"], "delivered", external_id="1562035000000000099")
        self.door.tick()
        self.assertEqual(self.receipt(m["id"])["state"], "delivered")
        self.assertEqual(len(self.fb.reply_posts), 1)

    def test_a_receipt_that_does_not_match_blocks_the_door(self):
        m = self.heard()
        self.write_reply(m["id"], "answer")
        self.fb.plan = ["mismatch"]
        self.door.tick()
        self.assertEqual(self.receipt(m["id"])["state"], "uncertain")
        self.assertIn("did not match", self.door.blocked)
        m2 = self.heard()
        self.write_reply(m2["id"], "next answer")
        self.door.tick()
        self.assertEqual(len(self.fb.reply_posts), 1)
        self.assertEqual(len(self.fi.redeems()), 1)
        self.assertIn("did not match", self.status()["blocked"])

    def test_rate_limit(self):
        a, b = self.heard(), self.heard()
        self.write_reply(a["id"], "one")
        self.write_reply(b["id"], "two")
        self.door.tick()
        self.door.tick()
        self.assertEqual(len(self.fb.reply_posts), 1)       # 3 s between posts
        self.assertIn("rate limit", self.status()["waiting"])
        self.offset += D.MIN_GAP + 0.1
        self.door.tick()
        self.assertEqual(len(self.fb.reply_posts), 2)


class TestCompetingResponders(Rig):
    def test_another_api_responder_blocks_the_door(self):
        a = self.heard()
        self.fb.outbound_reply(a["id"], "someone else's answer", "api_reply")
        self.door.tick()
        self.assertIn("another program", self.door.blocked)
        self.assertEqual(self.receipt(a["id"])["state"], "answered_elsewhere")
        b = self.heard()
        self.write_reply(b["id"], "answer")
        self.door.tick()
        self.assertEqual(self.fi.redeems(), [])
        self.assert_nothing_sent()
        self.assertFalse(self.status()["ready"])

    def test_the_owners_own_reply_marks_the_message_answered_without_blocking(self):
        a = self.heard()
        self.fb.outbound_reply(a["id"], "typed on the Bridge page", "manual_reply")
        self.door.tick()
        self.assertEqual(self.door.blocked, "")
        self.assertEqual(self.receipt(a["id"])["state"], "answered_elsewhere")
        self.write_reply(a["id"], "too late")
        self.door.tick()
        self.assert_nothing_sent()
        b = self.heard()
        self.write_reply(b["id"], "answer")
        self.door.tick()
        self.assertEqual(len(self.fb.reply_posts), 1)

    def test_a_second_door_cannot_use_the_same_state(self):
        with self.assertRaises(D.DoorError):
            self.open_door()

    def test_unsafe_configurations_do_not_start(self):
        self.door.close()
        for kw in [dict(channel_id=""), dict(owner_id="owner"), dict(state=self.box / "state"),
                   dict(mailbox=self.state / "box")]:
            with self.subTest(kw=kw), self.assertRaises(D.DoorError):
                self.open_door(**kw).close()
        (self.box / "outbox").mkdir()
        with self.assertRaises(D.DoorError):
            self.open_door()
        (self.box / "outbox").rmdir()
        self.door = self.open_door()


class TestDurability(Rig):
    def test_crash_before_redeeming_is_not_sent(self):
        m = self.heard()
        self.door.new_claim(m["id"], cycle_id="c", decision_id="dec_00000000000000aa",
                            reply_file=f"reply-{m['id']}.json")
        self.restart()
        r = self.receipt(m["id"])
        self.assertEqual((r["state"], r["final"]), ("not_sent", True))
        self.assertIn("nothing was posted", r["detail"])

    def test_crash_after_redeeming_is_uncertain_and_matched_to_the_bridge_record(self):
        m = self.heard()
        text = "answer"
        c = self.door.new_claim(m["id"], cycle_id="c", decision_id="dec_00000000000000aa",
                                reply_file=f"reply-{m['id']}.json",
                                text_sha256=hashlib.sha256(text.encode()).hexdigest())
        c["state"] = "redeemed"
        self.door.save_claim(c)
        self.restart()
        r = self.receipt(m["id"])
        self.assertEqual((r["state"], r["final"]), ("uncertain", False))
        out = self.fb.outbound_reply(m["id"], text, "api_reply")    # the post did happen
        self.fb.deliver(out["id"], "delivered", external_id="1562035000000000055")
        self.door.tick()
        r = self.receipt(m["id"])
        self.assertEqual((r["state"], r["bridge_message_id"]), ("delivered", out["id"]))
        self.assertEqual(self.door.blocked, "")
        self.assert_nothing_sent()

    def test_a_crash_while_posting_is_recorded_as_redeemed_first(self):
        m = self.heard()
        self.write_reply(m["id"], "answer")

        def crash(*a, **kw):
            raise SystemExit("power cut while posting")
        self.door.post = crash
        with self.assertRaises(SystemExit):
            self.door.tick()
        self.restart()
        r = self.receipt(m["id"])
        # Identity had allowed it and the post may have left: uncertain, never "not sent", never retried.
        self.assertEqual((r["state"], r["final"]), ("uncertain", False))
        self.door.tick()
        self.assertEqual(self.fb.reply_posts, [])

    def test_the_claim_itself_is_exclusive(self):
        m = self.heard()
        self.write_reply(m["id"], "first")
        self.door.tick()
        self.door.tick()                                   # the door has seen its own reply in the feed
        # Even if the earlier duplicate check were passed (a race), creating the claim refuses.
        self.door.claim = lambda rid: None
        self.offset += D.MIN_GAP + 0.1
        self.write_reply(m["id"], "second")
        self.door.tick()
        self.assertEqual(len(self.fb.reply_posts), 1)
        self.assertEqual(len(self.fi.redeems()), 1)
        self.assertIn("already has a reply", self.rejected()[-1]["reason"])

    def test_restart_does_not_resend_or_re_hear(self):
        m = self.heard()
        self.write_reply(m["id"], "answer")
        self.door.tick()
        self.restart()
        for _ in range(3):
            self.door.tick()
        self.assertEqual(len(self.fb.reply_posts), 1)
        self.assertEqual(len(list((self.box / "inbox").iterdir())), 1)

    def test_files_are_atomic_private_and_carry_no_text_or_keys(self):
        secret_inbound = "PRIVATE owner text 7f3a"
        secret_reply = "PRIVATE reply text 9c1e"
        m = self.heard(text=secret_inbound)
        self.write_reply(m["id"], secret_reply)
        self.door.tick()
        self.write_reply(m["id"], secret_reply + " again")
        self.door.tick()
        leftovers = [p for p in self.tmp.rglob(".tmp-*")]
        self.assertEqual(leftovers, [])
        for p in (self.state / "claims").iterdir():
            self.assertEqual(p.stat().st_mode & 0o777, 0o600)
        self.assertEqual(self.state.stat().st_mode & 0o777, 0o700)
        for p in list(self.box.rglob("*.json")) + list(self.state.rglob("*.json")):
            body = p.read_text()
            json.loads(body)
            self.assertNotIn(BRIDGE_KEY, body)
            self.assertNotIn(DOOR_KEY, body)
            self.assertNotIn(secret_reply, body, p)
            if p.parent.name != "inbox":
                self.assertNotIn(secret_inbound, body, p)
        status = (self.box / "status.json").read_text()
        self.assertNotIn(CHAN, status)
        self.assertNotIn(OWNER, status)

    def test_history_cleared_continues(self):
        a = self.heard()
        with self.fb.lock:
            self.fb.msgs.clear()
            self.fb.epoch += 1
        b = self.heard()
        self.assertIsNotNone(self.inbox(b["id"]))
        self.write_reply(a["id"], "answer to a message the Bridge forgot")
        self.door.tick()
        self.assertEqual(self.receipt(a["id"])["state"], "not_sent")   # Bridge: 404, nothing created


class TestDuplicateDiscordSource(Rig):
    """Repair 2 for Thought-Processor PR #129 (audit 6976046197): one immutable Discord message
    (channel + Discord message ID) becomes at most ONE answerable envelope, whatever Bridge IDs it
    arrives under; a duplicate never reaches Identity or POST /api/replies."""

    SRC = "1562034986897300001"

    def dup_of(self, original, **kw):
        """The Bridge hands the same Discord message over again under a new Bridge ID."""
        return self.heard(source_message_id=original["source_message_id"], text=kw.pop("text", "Hello TASS"), **kw)

    def assert_refused_duplicate(self, dup, original):
        self.assertIsNone(self.inbox(dup["id"]), "a duplicate reached Core's inbox")
        with self.fb.lock:                                 # the Bridge shows the envelope again (an update)
            if dup["id"] in self.fb.msgs:
                self.fb._bump(self.fb.msgs[dup["id"]])
        self.door.tick()
        self.assertIsNone(self.inbox(dup["id"]), "a duplicate reached Core's inbox when re-read")
        rec = self.door.record(dup["id"])
        self.assertIsNotNone(rec)
        self.assertFalse(rec["answerable"])
        self.assertEqual(rec["duplicate_of"], original["id"])
        # Core tries anyway, with a valid decision bound to the duplicate envelope.
        fake_inbox = {**self.door.record(original["id"]), "id": dup["id"]}
        dec = self.fi.mint(self.core_binding(fake_inbox, "a second answer"))
        before_redeems, before_posts = len(self.fi.redeems()), len(self.fb.reply_posts)
        self.offset += D.MIN_GAP + 0.1
        self.write_reply(dup["id"], "a second answer", decision=dec)
        self.door.tick()
        self.assertEqual(len(self.fi.redeems()), before_redeems, "a duplicate reached Identity")
        self.assertEqual(len(self.fb.reply_posts), before_posts, "a duplicate reached POST /api/replies")
        self.assertFalse(self.fi.decisions[dec]["used"])
        self.assertIn("must not be answered", self.rejected()[-1]["reason"])

    def answer(self, m, text="the one answer"):
        self.write_reply(m["id"], text)
        self.door.tick()
        self.assertEqual(self.receipt(m["id"])["state"], "queued")

    def test_duplicate_after_more_than_2000_bridge_messages(self):
        orig = self.heard(source_message_id=self.SRC)
        self.answer(orig)
        for i in range(2001):
            self.fb.inbound(text=f"later {i}", source_message_id=str(1562034990000000000 + i))
        self.door.tick()
        self.assertGreater(len(list((self.state / "inbound").iterdir())), 2001)
        dup = self.dup_of(orig)
        self.assertNotEqual(dup["id"], orig["id"])
        self.assert_refused_duplicate(dup, orig)
        self.assertEqual(len(self.fb.reply_posts), 1)

    def test_duplicate_after_restart(self):
        orig = self.heard(source_message_id=self.SRC)
        self.restart()
        dup = self.dup_of(orig)
        self.assert_refused_duplicate(dup, orig)
        self.answer(orig)                                  # the original is still answerable, once
        self.assertEqual(len(self.fb.reply_posts), 1)

    def test_duplicate_after_bridge_history_clear(self):
        orig = self.heard(source_message_id=self.SRC)
        self.answer(orig)
        with self.fb.lock:
            self.fb.msgs.clear()
            self.fb.epoch += 1
        dup = self.dup_of(orig)
        self.assert_refused_duplicate(dup, orig)
        self.assertEqual(len(self.fb.reply_posts), 1)

    def test_concurrent_duplicates_with_distinct_bridge_ids(self):
        for round_ in range(15):
            src = str(1562034986897310000 + round_)
            envs = [self.fb.inbound(source_message_id=src) for _ in range(6)]
            self.assertEqual(len({e["id"] for e in envs}), 6)
            start = threading.Barrier(len(envs))
            errors = []

            def take(e):
                try:
                    start.wait()
                    self.door.take_inbound(json.loads(json.dumps(e)), "manual")
                except Exception as ex:                    # noqa: BLE001
                    errors.append(ex)
            ts = [threading.Thread(target=take, args=(e,)) for e in envs]
            for t in ts:
                t.start()
            for t in ts:
                t.join()
            self.assertEqual(errors, [])
            answerable = [e["id"] for e in envs if (self.door.record(e["id"]) or {}).get("answerable")]
            in_inbox = [e["id"] for e in envs if self.inbox(e["id"])]
            self.assertEqual(len(answerable), 1, f"round {round_}: {answerable}")
            self.assertEqual(in_inbox, answerable)
            owner = self.door.source_owner(CHAN, src)
            self.assertEqual(owner, answerable[0])

    def test_duplicate_before_and_after_authorization(self):
        # Before: the duplicate arrives while the original is still unanswered.
        orig = self.heard(source_message_id=self.SRC)
        dup1 = self.dup_of(orig)
        self.assert_refused_duplicate(dup1, orig)
        # After: the original is authorized (consuming redeem) and posted; then another duplicate.
        dec = self.write_reply(orig["id"], "the one answer")
        self.door.tick()
        self.assertTrue(self.fi.decisions[dec]["used"])
        self.fb.deliver(self.fb.outbound()[0]["id"], "delivered", external_id="1562035000000000001")
        self.door.tick()
        dup2 = self.dup_of(orig)
        self.assert_refused_duplicate(dup2, orig)
        self.assertEqual(len(self.fb.reply_posts), 1)
        self.assertEqual([r["decision_id"] for r in self.fi.redeems()], [dec])
        self.assert_no_check_mode()

    def test_single_use_redemption_is_unchanged(self):
        orig = self.heard(source_message_id=self.SRC)
        dec = self.write_reply(orig["id"], "answer")
        self.door.tick()
        # Replaying the same decision for the same message: refused before Identity (one reply).
        self.offset += D.MIN_GAP + 0.1
        self.write_reply(orig["id"], "answer", decision=dec)
        self.door.tick()
        self.assertEqual(len(self.fi.redeems()), 1)
        self.assertEqual(len(self.fb.reply_posts), 1)

    def test_revoked_wrong_destination_and_uncertain_never_get_a_second_chance_via_a_duplicate(self):
        cases = []
        # Revoked grant: the original is not sent.
        self.fi.granted = False
        a = self.heard(source_message_id="1562034986897320001")
        self.write_reply(a["id"], "answer")
        self.door.tick()
        self.assertEqual(self.receipt(a["id"])["state"], "not_sent")
        self.fi.granted = True
        cases.append(a)
        # Wrong destination: the Bridge refuses to post it.
        self.offset += D.MIN_GAP + 0.1
        b = self.heard(source_message_id="1562034986897320002")
        self.write_reply(b["id"], "answer")
        self.door.tick()
        self.fb.deliver(self.fb.outbound()[-1]["id"], "failed", "not sent: destination changed")
        self.door.tick()
        self.assertEqual(self.receipt(b["id"])["state"], "not_sent")
        cases.append(b)
        # Ambiguous delivery: never retried.
        self.offset += D.MIN_GAP + 0.1
        c = self.heard(source_message_id="1562034986897320003")
        self.write_reply(c["id"], "answer")
        self.fb.plan = ["500"]
        self.door.tick()
        self.assertEqual(self.receipt(c["id"])["state"], "uncertain")
        cases.append(c)
        posts = len(self.fb.reply_posts)
        for orig in cases:
            self.assert_refused_duplicate(self.dup_of(orig), orig)
        self.assertEqual(len(self.fb.reply_posts), posts)

    def test_no_duplicate_outbound_reply_end_to_end(self):
        orig = self.heard(source_message_id=self.SRC)
        dups = [self.dup_of(orig) for _ in range(3)]
        for m in [orig] + dups:
            if self.inbox(m["id"]):
                self.offset += D.MIN_GAP + 0.1
                self.write_reply(m["id"], "answer " + m["id"])
                self.door.tick()
        for _ in range(3):
            self.door.tick()
        self.assertEqual(len(self.fb.reply_posts), 1)
        self.assertEqual(self.fb.reply_posts[0]["reply_to_id"], orig["id"])
        self.assertEqual(len([o for o in self.fb.outbound() if o["reply_to_source_id"] == self.SRC]), 1)

    def test_reply_time_guard_refuses_a_record_that_does_not_own_its_source(self):
        orig = self.heard(source_message_id=self.SRC)
        dup = self.dup_of(orig)
        # Simulate a record that slipped past the inbound check (e.g. written by an older door).
        rec = {**self.door.record(dup["id"]), "answerable": True, "not_answerable_reason": ""}
        rec.pop("duplicate_of")
        D.write_atomic(self.state / "inbound" / f"{dup['id']}.json", rec, 0o600)
        fake_inbox = {**self.door.record(orig["id"]), "id": dup["id"]}
        self.write_reply(dup["id"], "x", decision=self.fi.mint(self.core_binding(fake_inbox, "x")))
        self.door.tick()
        self.assertEqual(self.fi.redeems(), [])
        self.assert_nothing_sent()
        self.assertIn("already holds this Discord message", self.rejected()[-1]["reason"])

    def test_upgrade_indexes_existing_records_and_demotes_duplicates(self):
        # A pre-repair state: two answerable records for one Discord message, no source index.
        a = self.heard(source_message_id=self.SRC)
        self.door.close()
        import shutil
        shutil.rmtree(self.state / "sources")
        later = self.fb.inbound(source_message_id=self.SRC)
        rec = {**self.door.record(a["id"]), "id": later["id"], "seq": later["seq"]}
        D.write_atomic(self.state / "inbound" / f"{later['id']}.json", rec, 0o600)
        D.write_atomic(self.box / "inbox" / f"{later['seq']:06d}-{later['id']}.json", {**rec, "text": "Hello TASS"})
        self.door = self.open_door()
        self.assertEqual(self.door.source_owner(CHAN, self.SRC), a["id"])
        self.assertTrue(self.door.record(a["id"])["answerable"])
        self.assertFalse(self.door.record(later["id"])["answerable"])
        self.assertIsNone(self.inbox(later["id"]))
        self.assert_refused_duplicate(later, a)

    def test_damaged_source_record_fails_closed(self):
        m = self.fb.inbound(source_message_id=self.SRC)
        self.door._source_path(CHAN, self.SRC).write_text("{not json")
        self.door.tick()
        self.assertIsNone(self.inbox(m["id"]))
        self.assertFalse(self.door.record(m["id"])["answerable"])
        self.assertIn("damaged", self.door.record(m["id"])["not_answerable_reason"])

    def test_failed_source_write_holds_the_message_and_retries_later(self):
        m = self.fb.inbound(source_message_id=self.SRC)
        real = D.create_exclusive

        def broken(path, obj, mode=0o600):
            if path.parent.name == "sources":
                raise OSError("disk full (fixture)")
            return real(path, obj, mode)
        D.create_exclusive = broken
        try:
            self.door.tick()
        finally:
            D.create_exclusive = real
        self.assertIsNone(self.inbox(m["id"]))
        self.assertIsNone(self.door.record(m["id"]))
        self.assertIn("source index could not be written", self.status()["problem"])
        self.door.tick()                                   # the feed is read again: heard normally now
        self.assertTrue(self.inbox(m["id"])["answerable"])

    def test_source_index_is_private_and_core_cannot_reach_it(self):
        m = self.heard(source_message_id=self.SRC)
        files = list((self.state / "sources").iterdir())
        self.assertEqual(len(files), 1)
        self.assertEqual(files[0].stat().st_mode & 0o777, 0o600)
        self.assertNotIn(self.SRC, files[0].name)
        self.assertNotIn(CHAN, files[0].name)
        self.assertNotIn(self.SRC, files[0].read_text())
        self.assertEqual(json.loads(files[0].read_text())["bridge_id"], m["id"])
        self.assertFalse(any("sources" in str(p) for p in self.box.rglob("*")))
        self.assertFalse((self.state / "sources").resolve().is_relative_to(self.box.resolve()))


if __name__ == "__main__":
    unittest.main()
