"""Discord door: a mailbox between TASS's shared space and the Communication Bridge.

TASS's code (the Jupyter kernel) never holds a secret. It only writes and reads
plain files in shared/discord/; this small process, in its own container, is the
only thing that holds the bridge's API key and talks to the bridge:

    discord/status.json      is Discord connected? (written by the door, no secrets)
    discord/outbox/NAME.txt  a message to post in the channel         (TASS writes)
    discord/outbox/NAME.json {"text": "...", "reply_to": "m_..."}     (TASS writes)
    discord/sent/NAME.json   receipt: delivery queued/sending/delivered/failed/uncertain
    discord/failed/NAME.json refused, with the reason
    discord/inbox/*.json     what Dustin said in the channel           (door writes)

The bridge itself only accepts the owner's messages in one channel, never
double-posts (idempotency keys) and pauses itself if a send is uncertain.
The door adds a rate limit and a size limit.  Standard library only.
"""
from __future__ import annotations

import argparse
import hashlib
import json
import os
import re
import socket
import threading
import time
import urllib.error
import urllib.request
from pathlib import Path

MAX_CHARS = 1900                 # the bridge's Discord limit
MIN_GAP = 3.0                    # seconds between two posts
WINDOW, WINDOW_MAX = 600, 20     # at most 20 posts per 10 minutes
SETTLE = 1.0                     # a file must be this old (s) before it is sent: TASS may still be writing it
NAME_OK = re.compile(r"^[A-Za-z0-9][A-Za-z0-9._-]{0,80}$")


def log(msg: str):
    print(f"[door {time.strftime('%H:%M:%S')}] {msg}", flush=True)   # never message text, never the key


class Door:
    def __init__(self, mailbox: Path, bridge: str, key_file: Path):
        self.box, self.bridge, self.key_file = mailbox, bridge.rstrip("/"), key_file
        for d in ("outbox", "sent", "failed", "inbox"):
            (self.box / d).mkdir(parents=True, exist_ok=True)
        self.state_file = self.box / ".door-state.json"
        try:
            st = json.loads(self.state_file.read_text())
        except (OSError, ValueError):
            st = {}
        self.cursor = int(st.get("cursor", 0))
        self.epoch = st.get("epoch")
        self.tracked: dict[str, str] = dict(st.get("tracked", {}))   # bridge message id -> sent/ file name
        self.posts: list[float] = []
        self.waiting = ""
        self.offline = 0

    # --- bridge API ----------------------------------------------------------------------------
    def key(self) -> str:
        return self.key_file.read_text().strip()

    def call(self, method: str, path: str, body: dict | None = None, idem: str | None = None, auth=True):
        headers = {"Accept": "application/json"}
        data = None
        if body is not None:
            data = json.dumps(body).encode()
            headers["Content-Type"] = "application/json"
        if idem:
            headers["Idempotency-Key"] = idem
        if auth:
            headers["Authorization"] = "Bearer " + self.key()
        req = urllib.request.Request(self.bridge + path, data=data, method=method, headers=headers)
        try:
            with urllib.request.urlopen(req, timeout=30) as r:
                return r.status, json.loads(r.read() or b"{}")
        except urllib.error.HTTPError as e:
            try:
                return e.code, json.loads(e.read() or b"{}")
            except ValueError:
                return e.code, {}

    # --- status --------------------------------------------------------------------------------
    def write_status(self, st: dict | None, problem: str = ""):
        out = {"door": "running", "updated_at": time.strftime("%Y-%m-%dT%H:%M:%S%z"), "problem": problem,
               "waiting": self.waiting, "outbox": len(self.outbox_files()),
               "limits": {"max_chars": MAX_CHARS, "seconds_between_posts": MIN_GAP,
                          "posts_per_10_min": WINDOW_MAX}}
        if st:
            out.update({k: st.get(k) for k in ("discord_state", "discord_configured", "discord_detail",
                                               "paused", "mode", "last_inbound_at", "last_outbound_at")})
            out["connected"] = st.get("discord_state") == "connected" and not st.get("paused")
        else:
            out.update({"discord_state": "bridge_offline", "connected": False})
        tmp = self.box / ".status.tmp"
        tmp.write_text(json.dumps(out, indent=2))
        tmp.replace(self.box / "status.json")
        return out

    def save(self):
        self.state_file.write_text(json.dumps({"cursor": self.cursor, "epoch": self.epoch,
                                               "tracked": dict(list(self.tracked.items())[-500:])}))

    # --- inbound and delivery updates -----------------------------------------------------------
    def pull(self):
        while True:
            code, d = self.call("GET", f"/api/messages?after={self.cursor}&limit=100")
            if code != 200:
                raise RuntimeError(f"bridge /api/messages answered {code}")
            if self.epoch is not None and d.get("history_epoch") != self.epoch:
                log("bridge history was cleared; continuing from its new cursor")
            self.epoch = d.get("history_epoch")
            for m in d.get("messages", []):
                self.take(m)
            self.cursor = d.get("cursor", self.cursor)
            if not d.get("more"):
                break
        self.save()

    def take(self, m: dict):
        if m.get("direction") == "inbound" and m.get("source") == "discord":
            p = self.box / "inbox" / f"{int(m.get('seq', 0)):06d}-{m['id']}.json"
            if not p.exists():
                p.write_text(json.dumps({"id": m["id"], "from": "Dustin", "text": m.get("text", ""),
                                         "at": m.get("source_time") or m.get("received_at"),
                                         "catch_up": bool(m.get("catch_up"))}, indent=2))
                log(f"heard {m['id']}")
        name = self.tracked.get(m.get("id", ""))
        if name:
            p = self.box / "sent" / name
            try:
                r = json.loads(p.read_text())
            except (OSError, ValueError):
                return
            r.update({"delivery": m.get("delivery"), "detail": m.get("delivery_detail") or "",
                      "delivered_at": m.get("delivered_at")})
            p.write_text(json.dumps(r, indent=2))

    # --- outbound -------------------------------------------------------------------------------
    def outbox_files(self):
        return sorted((p for p in (self.box / "outbox").iterdir()
                       if p.is_file() and p.suffix in (".txt", ".json") and not p.name.startswith(".")),
                      key=lambda p: p.stat().st_mtime)

    def fail(self, p: Path, why: str, text: str = ""):
        (self.box / "failed" / (p.stem + ".json")).write_text(json.dumps(
            {"file": p.name, "error": why, "text": text[:200], "at": time.strftime("%Y-%m-%dT%H:%M:%S%z")}, indent=2))
        p.unlink(missing_ok=True)
        log(f"refused {p.name}: {why}")

    def send_next(self, connected: bool):
        self.waiting = ""
        now = time.time()
        self.posts = [t for t in self.posts if now - t < WINDOW]
        for p in self.outbox_files():
            if now - p.stat().st_mtime < SETTLE:
                continue
            if not NAME_OK.match(p.stem):
                self.fail(p, "file name: use letters, numbers, dot, dash or underscore")
                continue
            try:
                raw = p.read_text("utf-8")
            except (OSError, UnicodeDecodeError):
                self.fail(p, "not readable UTF-8 text")
                continue
            reply_to = None
            if p.suffix == ".json":
                try:
                    d = json.loads(raw)
                    text, reply_to = str(d.get("text", "")), d.get("reply_to")
                except (ValueError, AttributeError):
                    self.fail(p, 'not valid JSON: write {"text": "...", "reply_to": "m_..."}')
                    continue
            else:
                text = raw
            text = text.strip()
            if not text:
                self.fail(p, "empty message")
                continue
            if len(text) > MAX_CHARS:
                self.fail(p, f"too long: {len(text)} characters, the limit is {MAX_CHARS}", text)
                continue
            if reply_to is not None and not re.fullmatch(r"m_[0-9a-f]{6,40}", str(reply_to)):
                self.fail(p, "reply_to must be a message id like m_1a2b3c… from discord/inbox", text)
                continue
            if not connected:
                self.waiting = "Discord isn't connected; messages wait in outbox/"
                return
            if self.posts and now - self.posts[-1] < MIN_GAP or len(self.posts) >= WINDOW_MAX:
                self.waiting = "rate limit: waiting before the next post"
                return
            idem = "tass-" + hashlib.sha256(f"{p.name}\0{reply_to}\0{text}".encode()).hexdigest()[:40]
            if reply_to:
                code, d = self.call("POST", "/api/replies", {"reply_to_id": reply_to, "text": text}, idem)
            else:
                code, d = self.call("POST", "/api/messages", {"text": text, "destination": "discord"}, idem)
            if code in (200, 201):
                msg = d.get("message") or {}
                rid = d.get("receipt_id") or msg.get("id")
                (self.box / "sent" / (p.stem + ".json")).write_text(json.dumps(
                    {"file": p.name, "id": rid, "text": text, "reply_to": reply_to,
                     "delivery": d.get("delivery") or msg.get("delivery"), "detail": "",
                     "queued_at": time.strftime("%Y-%m-%dT%H:%M:%S%z")}, indent=2))
                if rid:
                    self.tracked[rid] = p.stem + ".json"
                p.unlink(missing_ok=True)
                self.posts.append(time.time())
                log(f"sent {p.name} as {rid}")
                self.save()
                return                                    # one post per tick
            if code == 503:
                self.waiting = "the bridge says Discord isn't available right now (paused or not connected)"
                return
            self.fail(p, f"bridge refused it ({code}): {d.get('error', '')}", text)

    # --- loop -----------------------------------------------------------------------------------
    def tick(self):
        try:
            code, st = self.call("GET", "/api/status", auth=False)
            st = st if code == 200 else None
        except (OSError, ValueError):
            self.write_status(None, "the Communication Bridge isn't answering")
            self.offline += 1
            return
        self.offline = 0
        problem = ""
        try:
            if not self.key_file.exists():
                problem = "waiting for the bridge to create its API key"
            else:
                self.pull()
                self.send_next(bool(st and st.get("discord_state") == "connected" and not st.get("paused")))
        except (OSError, ValueError, RuntimeError) as e:
            problem = str(e)[:200]
        self.write_status(st, problem)

    def run(self, every: float = 2.0):
        log(f"mailbox {self.box}, bridge {self.bridge}")
        while True:
            self.tick()
            if self.offline * every > 60:
                # The bridge restarted under us (we share its network): exit so Docker restarts us next to it.
                log("bridge unreachable for a minute; restarting")
                raise SystemExit(3)
            time.sleep(every)


def forward(listen_port: int, target_port: int):
    """Let Dustin open the bridge's own page: the bridge only listens on 127.0.0.1 inside its container."""
    srv = socket.create_server(("0.0.0.0", listen_port), reuse_port=False)

    def pipe(a, b):
        try:
            while chunk := a.recv(65536):
                b.sendall(chunk)
        except OSError:
            pass
        finally:
            for s in (a, b):
                try:
                    s.shutdown(socket.SHUT_RDWR)
                except OSError:
                    pass

    def serve():
        while True:
            c, _ = srv.accept()
            try:
                u = socket.create_connection(("127.0.0.1", target_port), timeout=5)
                u.settimeout(None)
            except OSError:
                c.close()
                continue
            threading.Thread(target=pipe, args=(c, u), daemon=True).start()
            threading.Thread(target=pipe, args=(u, c), daemon=True).start()

    threading.Thread(target=serve, daemon=True).start()
    log(f"bridge page forwarded on port {listen_port}")


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--mailbox", default=os.environ.get("DOOR_MAILBOX", "/mailbox"))
    ap.add_argument("--bridge", default=os.environ.get("DOOR_BRIDGE", "http://127.0.0.1:7820"))
    ap.add_argument("--key-file", default=os.environ.get("DOOR_KEY_FILE", "/bridge-data/api-key"))
    ap.add_argument("--forward", type=int, default=int(os.environ.get("DOOR_FORWARD_PORT", "0")))
    ap.add_argument("--every", type=float, default=2.0)
    a = ap.parse_args()
    if a.forward:
        forward(a.forward, int(a.bridge.rsplit(":", 1)[1].split("/")[0]))
    Door(Path(a.mailbox), a.bridge, Path(a.key_file)).run(a.every)


if __name__ == "__main__":
    main()
