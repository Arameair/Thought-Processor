#!/usr/bin/env python3
"""Minimal fail-closed Thought Processor feature adapter for Developer Utility.

Only signed local lifecycle/status calls are implemented. This does NOT install
or launch Windows Brain Bar and must not be advertised as an approved feature.
Identity must independently authorize a reviewed, pinned package first.
"""
import hashlib
import hmac
import json
import os
import secrets
import signal
import sys
import threading
import time
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

API = "thought.feature-api/1"
MAX_BODY = 16 * 1024
SKEW_SECONDS = 60


def sha(data):
    return hashlib.sha256(data).hexdigest()


def mac(key, data):
    return hmac.new(key, data, hashlib.sha256).hexdigest()


def required(name):
    value = os.environ.get(name)
    if not value:
        raise RuntimeError("missing Processor launch field: " + name)
    return value


def atomic_json(path, obj):
    tmp = path + ".tmp." + secrets.token_hex(6)
    with os.fdopen(os.open(tmp, os.O_CREAT | os.O_EXCL | os.O_WRONLY, 0o600), "w") as f:
        json.dump(obj, f, sort_keys=True)
        f.flush()
        os.fsync(f.fileno())
    os.replace(tmp, path)


class Agent:
    def __init__(self):
        if required("TF_API") != API:
            raise RuntimeError("unsupported feature API")
        self.installation = required("TF_INSTALLATION_ID")
        self.launch = required("TF_LAUNCH_ID")
        self.identity = required("TF_IDENTITY_SYSTEM_ID")
        self.runtime_dir = required("TF_RUNTIME_DIR")
        required("TF_DATA_DIR")
        identity_url = required("TF_IDENTITY_URL")
        if not identity_url.startswith("http://127.0.0.1:"):
            raise RuntimeError("Identity URL must use IPv4 loopback")
        with open(required("TF_CREDENTIAL_FILE"), "r", encoding="utf8") as f:
            credential = json.load(f)
        if credential.get("installation_id") != self.installation or credential.get("identity_system_id") != self.identity:
            raise RuntimeError("installation credential does not match this launch")
        key_path = required("TF_LAUNCH_KEY_FILE")
        with open(key_path, "r", encoding="ascii") as f:
            raw = f.read().strip()
        self.key = bytes.fromhex(raw)
        if len(self.key) != 32:
            raise RuntimeError("invalid launch key length")
        os.remove(key_path)
        self.lock = threading.Lock()
        self.nonces = {}
        self.start = time.monotonic()
        self.stopping = False
        self.heartbeat = 0

    def authorized(self, method, path, headers, body):
        stamp, nonce = headers.get("X-TF-Time", ""), headers.get("X-TF-Nonce", "")
        proof = headers.get("X-TF-Auth", "")
        try:
            ts = int(stamp)
        except ValueError:
            return False
        if abs(time.time() - ts) > SKEW_SECONDS or not (16 <= len(nonce) <= 64):
            return False
        expected = mac(self.key, (API + " request\n" + method + "\n" + path + "\n" + stamp + "\n" + nonce + "\n" + sha(body)).encode())
        if not hmac.compare_digest(expected, proof):
            return False
        with self.lock:
            now = time.monotonic()
            self.nonces = {n: t for n, t in self.nonces.items() if now - t <= 180}
            if nonce in self.nonces:
                return False
            self.nonces[nonce] = now
        return True

    def response_proof(self, nonce, code, body):
        return mac(self.key, (API + " response\n" + nonce + "\n" + str(code) + "\n").encode() + body)

    def status(self):
        with self.lock:
            self.heartbeat += 1
            heartbeat = self.heartbeat
        return {"api": API, "installation_id": self.installation, "launch_id": self.launch,
                "pid": os.getpid(), "ready": False, "heartbeat_seq": heartbeat,
                "provider_ready": False, "utility_state": "NOT_INTEGRATED",
                "brain_bar_state": "NOT_LAUNCHED", "authority": "NOT_VERIFIED",
                "message": "Lifecycle adapter only. Identity policy and Windows host control are not integrated."}


class Handler(BaseHTTPRequestHandler):
    protocol_version = "HTTP/1.1"
    server_version = "developer-utility-adapter"

    def log_message(self, *_):
        pass

    def reply(self, code, data, nonce=None):
        body = json.dumps(data, sort_keys=True).encode()
        self.send_response(code)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.send_header("Cache-Control", "no-store")
        if nonce is not None:
            self.send_header("X-TF-Installation-ID", self.server.agent.installation)
            self.send_header("X-TF-Launch-ID", self.server.agent.launch)
            self.send_header("X-TF-Proof", self.server.agent.response_proof(nonce, code, body))
        self.end_headers()
        self.wfile.write(body)

    def guard(self):
        if self.headers.get("Origin"):
            self.reply(403, {"error": "BROWSER_REFUSED"})
            return None
        if self.headers.get("Host", "").split(":")[0] != "127.0.0.1":
            self.reply(421, {"error": "WRONG_HOST"})
            return None
        try:
            n = int(self.headers.get("Content-Length", "0"))
        except ValueError:
            n = -1
        if not 0 <= n <= MAX_BODY:
            self.reply(413, {"error": "BODY_INVALID"})
            return None
        return self.rfile.read(n)

    def handle_request(self):
        body = self.guard()
        if body is None:
            return
        if self.command == "GET" and self.path == "/v1/health":
            self.reply(200, {"api": API, "state": "RUNNING", "note": "liveness only; not approval or Windows bar status"})
            return
        agent = self.server.agent
        if not agent.authorized(self.command, self.path, self.headers, body):
            self.reply(401, {"error": "LAUNCHER_AUTH_REQUIRED"})
            return
        nonce = self.headers["X-TF-Nonce"]
        if self.command == "GET" and self.path == "/v1/status":
            self.reply(200, agent.status(), nonce)
        elif self.command == "GET" and self.path == "/v1/capabilities":
            self.reply(200, {"api": API, "capabilities": [], "standalone": {"health": True, "self_test": False}}, nonce)
        elif self.command == "GET" and self.path == "/v1/connections":
            self.reply(200, {"state": "AUTHORITY_NOT_CONNECTED", "grants": []}, nonce)
        elif self.command == "POST" and self.path == "/v1/shutdown":
            agent.stopping = True
            self.reply(202, {"stopping": True}, nonce)
            threading.Thread(target=self.server.shutdown, daemon=True).start()
        else:
            self.reply(403, {"error": "CAPABILITY_NOT_IMPLEMENTED"}, nonce)

    def do_GET(self):
        self.handle_request()

    def do_POST(self):
        self.handle_request()


def serve():
    os.umask(0o077)
    agent = Agent()
    server = ThreadingHTTPServer(("127.0.0.1", 0), Handler)
    server.daemon_threads = True
    server.agent = agent
    endpoint = {"api": API, "installation_id": agent.installation,
                "launch_id": agent.launch, "pid": os.getpid(),
                "url": "http://127.0.0.1:%d" % server.server_port,
                "started_at": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime())}
    atomic_json(os.path.join(agent.runtime_dir, "endpoint.json"), endpoint)
    def stop(*_):
        threading.Thread(target=server.shutdown, daemon=True).start()
    signal.signal(signal.SIGTERM, stop)
    signal.signal(signal.SIGINT, stop)
    try:
        server.serve_forever(poll_interval=0.05)
    finally:
        server.server_close()


if __name__ == "__main__":
    try:
        serve()
    except (RuntimeError, OSError, ValueError) as exc:
        print("Developer Utility adapter refused to start: " + str(exc), file=sys.stderr)
        sys.exit(2)
