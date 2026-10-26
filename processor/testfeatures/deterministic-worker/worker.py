#!/usr/bin/env python3
"""P1 Deterministic Worker — TRUSTED LOCAL TEST FIXTURE.

A feature that implements the language-neutral local feature API
`thought.feature-api/1` (see docs/FEATURE_API_V1.md) with Python's standard
library only. It is NOT Fast AI Worker and NOT an AI: its one protected
capability, `worker.test.synthetic`, is a deterministic text transform
labelled SYNTHETIC TEST DATA. It makes no external network calls except to
the local Identity it was installed against.

What it enforces itself (receiver-side enforcement):
  * launcher-signed requests for status/self-test/shutdown (per-launch key)
  * for the protected capability: a ticket only Identity can sign for THIS
    installation, bound to the exact request body, unexpired, single-use
    (durable ledger) — and then a LIVE, signed Identity decision before any
    work. Identity unreachable => 503 AUTHORITY_UNAVAILABLE, never allow.

Python 3.8+ (Ubuntu 20.04 and newer). Run only by Thought Processor.
"""
import base64
import hashlib
import hmac
import json
import os
import re
import secrets
import signal
import sys
import threading
import time
import urllib.error
import urllib.request
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

API = "thought.feature-api/1"
VERSION = "0.1.0"
FEATURE_ID = "p1-deterministic-worker"
CAPABILITY = "worker.test.synthetic"
DATA_CLASS = "SYNTHETIC TEST DATA — deterministic local transform; no AI provider, no external call"
MAX_BODY = 16 * 1024
SKEW = 60


def env(name):
    v = os.environ.get(name, "")
    if not v:
        sys.stderr.write("missing %s — this program is started by Thought Processor\n" % name)
        sys.exit(2)
    return v


def log(level, event, detail="", **f):
    rec = {"at": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()), "level": level, "event": event, "detail": detail}
    rec.update(f)
    print(json.dumps(rec, sort_keys=True), flush=True)


def sha256_hex(b):
    return hashlib.sha256(b).hexdigest()


def hmac_hex(key, msg):
    return hmac.new(key, msg, hashlib.sha256).hexdigest()


def write_atomic(path, data):
    tmp = path + ".tmp-" + secrets.token_hex(4)
    fd = os.open(tmp, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
    with os.fdopen(fd, "wb") as f:
        f.write(data)
        f.flush()
        os.fsync(f.fileno())
    os.replace(tmp, path)


class Worker:
    def __init__(self):
        if os.environ.get("TF_API") != API:
            sys.stderr.write("TF_API must be %s\n" % API)
            sys.exit(2)
        self.installation_id = env("TF_INSTALLATION_ID")
        self.launch_id = env("TF_LAUNCH_ID")
        self.data_dir = env("TF_DATA_DIR")
        self.runtime_dir = env("TF_RUNTIME_DIR")
        self.identity_url = env("TF_IDENTITY_URL").rstrip("/")
        self.identity_system_id = env("TF_IDENTITY_SYSTEM_ID")
        if not self.identity_url.startswith("http://127.0.0.1:"):
            sys.stderr.write("Identity must be on 127.0.0.1\n")
            sys.exit(2)
        key_file = env("TF_LAUNCH_KEY_FILE")
        with open(key_file, "r") as f:
            self.launch_key = bytes.fromhex(f.read().strip())
        os.remove(key_file)  # the per-launch key exists only in this process now
        with open(env("TF_CREDENTIAL_FILE"), "r") as f:
            cred = json.load(f)
        if cred.get("installation_id") != self.installation_id or cred.get("identity_system_id") != self.identity_system_id:
            sys.stderr.write("credential does not belong to this installation\n")
            sys.exit(2)
        secret = cred["secret"].encode()
        self.req_key = hmac.new(secret, b"identity/2 request key", hashlib.sha256).digest()
        self.resp_key = hmac.new(secret, b"identity/2 response key", hashlib.sha256).digest()
        self.ticket_key = hmac.new(secret, b"identity/2 ticket key", hashlib.sha256).digest()
        self.lock = threading.Lock()
        self.launch_nonces = {}
        self.ledger_path = os.path.join(self.data_dir, "ticket-ledger.json")
        self.ledger = self._load_ledger()
        self.started = time.time()
        self.heartbeat_seq = 0
        self.in_flight = 0
        self.stopping = False
        self.counters = {"self_tests": 0, "requests": 0, "allowed": 0, "denied": 0, "unavailable": 0, "invalid": 0}
        self.latencies = []
        self.faults = self._load_faults()

    # ---- durable single-use ledger (survives restart) ----
    def _load_ledger(self):
        try:
            with open(self.ledger_path, "rb") as f:
                d = json.loads(f.read().decode())
            if not isinstance(d, dict):
                raise ValueError("ledger shape")
            return d
        except FileNotFoundError:
            return {}
        except Exception:
            # A damaged ledger is not an empty one: refuse protected work.
            log("error", "ledger.damaged", "ticket ledger unreadable; protected requests are refused until it is repaired")
            return None

    def _remember_ticket(self, tid, exp):
        """Record tid before asking Identity. False = already seen; None = can't persist."""
        with self.lock:
            if self.ledger is None:
                return None
            if tid in self.ledger:
                return False
            now = int(time.time())
            nxt = {k: v for k, v in self.ledger.items() if v > now - 600}
            nxt[tid] = exp
            try:
                write_atomic(self.ledger_path, json.dumps(nxt).encode())
            except OSError:
                return None
            self.ledger = nxt
            return True

    def _load_faults(self):
        # TEST-ONLY fault injection: enabled only if the operator/harness put
        # this file in the installation's data folder. Never on by default.
        p = os.path.join(self.data_dir, "TEST-ONLY-fault-injection.json")
        try:
            with open(p) as f:
                d = json.load(f)
            log("warn", "faults.enabled", "TEST-ONLY fault injection file present", hooks=sorted(d))
            return d
        except FileNotFoundError:
            return {}
        except Exception:
            return {}

    # ---- launcher authentication (per-launch key) ----
    def verify_launcher(self, method, path, headers, body):
        ts, nonce, mac = headers.get("X-TF-Time", ""), headers.get("X-TF-Nonce", ""), headers.get("X-TF-Auth", "")
        try:
            t = int(ts)
        except ValueError:
            return False
        if abs(time.time() - t) > SKEW or not (16 <= len(nonce) <= 64) or len(mac) != 64:
            return False
        want = hmac_hex(self.launch_key, ("%s request\n%s\n%s\n%s\n%s\n%s" % (API, method, path, ts, nonce, sha256_hex(body))).encode())
        if not hmac.compare_digest(want, mac):
            return False
        with self.lock:
            now = time.time()
            for k in [k for k, v in self.launch_nonces.items() if now - v > 3 * SKEW]:
                del self.launch_nonces[k]
            if nonce in self.launch_nonces:
                return False
            self.launch_nonces[nonce] = now
        return True

    def launcher_proof(self, nonce, status, body):
        return hmac_hex(self.launch_key, ("%s response\n%s\n%d\n" % (API, nonce, status)).encode() + body)

    # ---- Identity (identity/2 with this installation's credential) ----
    def identity_call(self, method, path, obj):
        body = json.dumps(obj).encode() if obj is not None else b""
        ts, nonce = str(int(time.time())), secrets.token_hex(12)
        mac = hmac_hex(self.req_key, ("identity/2 request\n%s\n%s\n%s\n%s\n%s" % (method, path, ts, nonce, sha256_hex(body))).encode())
        req = urllib.request.Request(self.identity_url + path, data=body if method == "POST" else None, method=method)
        req.add_header("Content-Type", "application/json")
        req.add_header("X-Installation-ID", self.installation_id)
        req.add_header("X-Identity-Time", ts)
        req.add_header("X-Identity-Nonce", nonce)
        req.add_header("X-Identity-Auth", mac)
        opener = urllib.request.build_opener(urllib.request.ProxyHandler({}))  # loopback only, never a proxy
        try:
            resp = opener.open(req, timeout=3)
            status, raw, hdr = resp.status, resp.read(1 << 20), resp.headers
        except urllib.error.HTTPError as e:
            status, raw, hdr = e.code, e.read(1 << 20), e.headers
        except Exception as e:
            return "UNAVAILABLE", None, "Identity did not answer (%s)" % type(e).__name__
        if status in (401,):
            return "UNAVAILABLE", None, "Identity refused this installation's credential (401)"
        proof = hdr.get("X-Identity-Proof", "")
        want = hmac_hex(self.resp_key, ("identity/2 response\n%s\n%d\n" % (nonce, status)).encode() + raw)
        if not hmac.compare_digest(want, proof) or hdr.get("X-Identity-System-ID", "") != self.identity_system_id:
            return "UNAVAILABLE", None, "answer not signed by the Identity this feature was installed against"
        try:
            return status, json.loads(raw.decode()), ""
        except Exception:
            return "UNAVAILABLE", None, "unreadable answer"

    # ---- ticket ----
    def verify_ticket(self, tk, capability, body):
        parts = tk.split(".")
        if len(parts) != 3 or parts[0] != "tk1":
            return None, "TICKET_MALFORMED"
        want = hmac_hex(self.ticket_key, ("identity/2 ticket\n" + parts[1]).encode())
        if not hmac.compare_digest(want, parts[2]):
            return None, "TICKET_SIGNATURE_INVALID"
        try:
            p = json.loads(base64.urlsafe_b64decode(parts[1] + "=" * (-len(parts[1]) % 4)).decode())
        except Exception:
            return None, "TICKET_MALFORMED"
        now = int(time.time())
        if p.get("v") != 1 or p.get("iss") != self.identity_system_id:
            return None, "TICKET_WRONG_ISSUER"
        if p.get("aud") != self.installation_id:
            return None, "TICKET_WRONG_RECIPIENT"
        if p.get("cap") != capability:
            return None, "TICKET_WRONG_CAPABILITY"
        if now >= int(p.get("exp", 0)) or int(p.get("iat", 0)) > now + SKEW:
            return None, "TICKET_EXPIRED"
        if p.get("dig") != sha256_hex(body):
            return None, "TICKET_BODY_MISMATCH"
        return p, ""

    @staticmethod
    def transform(text):
        return {"upper": text.upper(), "reversed": text[::-1], "length": len(text),
                "sha256": sha256_hex(text.encode("utf-8"))}

    def self_test(self):
        got = self.transform("p1 self-test")
        ok = got == {"upper": "P1 SELF-TEST", "reversed": "tset-fles 1p", "length": 12,
                     "sha256": sha256_hex(b"p1 self-test")}
        with self.lock:
            self.counters["self_tests"] += 1
        return {"result": "PASS" if ok else "FAIL", "data_class": DATA_CLASS, "input": "p1 self-test (built-in)",
                "output": got, "external_calls": 0, "grants_used": "none — standalone, nonprivileged self-test"}

    def status(self):
        with self.lock:
            lat = sorted(self.latencies[-100:])
            return {"api": API, "feature_id": FEATURE_ID, "version": VERSION, "installation_id": self.installation_id,
                    "launch_id": self.launch_id, "pid": os.getpid(), "state": "STOPPING" if self.stopping else "RUNNING",
                    "ready": not self.stopping and self.ledger is not None,
                    "provider": {"kind": "deterministic-local", "ready": "NOT_APPLICABLE — no AI provider in this fixture"},
                    "heartbeat_seq": self.heartbeat_seq, "uptime_s": int(time.time() - self.started),
                    "in_flight": self.in_flight, "counters": dict(self.counters),
                    "latency_ms_p50": lat[len(lat) // 2] if lat else None, "latency_ms_max": lat[-1] if lat else None,
                    "trust_class": "TRUSTED_LOCAL_TEST_FIXTURE", "data_class": DATA_CLASS,
                    "heartbeat_note": "heartbeat = process liveness only; it is not an observation, grant or result"}


W = None


class Handler(BaseHTTPRequestHandler):
    protocol_version = "HTTP/1.1"
    server_version = "p1-deterministic-worker"

    def log_message(self, *a):
        pass

    def send(self, status, obj, launcher_nonce=None):
        body = json.dumps(obj, sort_keys=True).encode()
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.send_header("Cache-Control", "no-store")
        self.send_header("X-TF-API", API)
        if launcher_nonce is not None:
            self.send_header("X-TF-Installation-ID", W.installation_id)
            self.send_header("X-TF-Launch-ID", W.launch_id)
            self.send_header("X-TF-Proof", W.launcher_proof(launcher_nonce, status, body))
        self.end_headers()
        self.wfile.write(body)

    def body(self):
        n = int(self.headers.get("Content-Length") or 0)
        if n > MAX_BODY:
            return None
        return self.rfile.read(n) if n else b""

    def guard(self):
        if self.headers.get("Origin"):
            self.send(403, {"error": "BROWSER_REFUSED"})
            return None
        if self.headers.get("Host", "").split(":")[0] not in ("127.0.0.1", "localhost"):
            self.send(421, {"error": "WRONG_HOST"})
            return None
        b = self.body()
        if b is None:
            self.send(413, {"error": "TOO_LARGE"})
        return b

    def launcher(self, b):
        if not W.verify_launcher(self.command, self.path, self.headers, b):
            self.send(401, {"error": "LAUNCHER_AUTH_REQUIRED"})
            return None
        return self.headers.get("X-TF-Nonce")

    def do_GET(self):
        b = self.guard()
        if b is None:
            return
        if self.path == "/v1/health":
            return self.send(200, {"api": API, "state": "RUNNING", "note": "self-reported liveness; not authentication"})
        n = self.launcher(b)
        if n is None:
            return
        if self.path == "/v1/status":
            return self.send(200, W.status(), n)
        if self.path == "/v1/capabilities":
            return self.send(200, {"api": API, "capabilities": [{"id": CAPABILITY, "data_class": DATA_CLASS, "requires_grant": True,
                                                                 "input": {"input": "text up to 1000 characters"}}],
                                   "standalone": {"health": True, "self_test": True, "privileged": False}}, n)
        if self.path == "/v1/connections":
            st, j, why = W.identity_call("GET", "/api/v1/installation/self", None)
            if st != 200:
                return self.send(200, {"source": "Identity", "state": "AUTHORITY_UNAVAILABLE", "detail": why or str(st)}, n)
            return self.send(200, {"source": "Identity (signed answer to this installation)", "state": "OK", "self": j}, n)
        self.send(404, {"error": "NOT_FOUND"}, n)

    def do_POST(self):
        b = self.guard()
        if b is None:
            return
        if self.path == "/v1/self-test":
            n = self.launcher(b)
            if n is None:
                return
            return self.send(200, W.self_test(), n)
        if self.path == "/v1/shutdown":
            n = self.launcher(b)
            if n is None:
                return
            with W.lock:
                W.stopping = True
                busy = W.in_flight
            self.send(202, {"stopping": True, "in_flight": busy, "note": "new protected requests are refused; in-flight work finishes first"}, n)
            threading.Thread(target=shutdown, daemon=True).start()
            return
        if self.path.startswith("/v1/actions/"):
            return self.action(self.path[len("/v1/actions/"):], b)
        self.send(404, {"error": "NOT_FOUND"})

    def action(self, capability, body):
        # If the launcher sends a fresh nonce, every answer below is signed
        # with this launch's key, so the launcher can tell a real answer from
        # one made up by whatever might sit on this port after a crash.
        ln = self.headers.get("X-TF-Nonce", "")
        ln = ln if re.match(r"^[0-9a-f]{16,64}$", ln) else None
        reply = lambda st, obj: self.send(st, obj, ln)
        t0 = time.time()
        with W.lock:
            if W.stopping:
                return reply(503, {"error": "STOPPING", "detail": "this feature is shutting down; nothing was done"})
            W.counters["requests"] += 1
        if capability != CAPABILITY:
            with W.lock:
                W.counters["invalid"] += 1
            return reply(404, {"error": "UNKNOWN_CAPABILITY", "capability": capability[:64]})
        p, why = W.verify_ticket(self.headers.get("X-TF-Ticket", ""), capability, body)
        if p is None:
            with W.lock:
                W.counters["invalid"] += 1
            log("warn", "action.refused", why)
            return reply(401, {"error": "TICKET_INVALID", "reason": why, "detail": "nothing was done"})
        seen = W._remember_ticket(p["tid"], p["exp"])
        if seen is None:
            return reply(503, {"error": "REPLAY_LEDGER_UNAVAILABLE", "detail": "nothing was done"})
        if seen is False:
            with W.lock:
                W.counters["invalid"] += 1
            log("warn", "action.replay", "ticket already used", ticket_id=p["tid"])
            return reply(409, {"error": "TICKET_REPLAYED", "ticket_id": p["tid"], "detail": "nothing was done"})
        try:
            req = json.loads(body.decode() or "{}")
            text, rid = req.get("input"), req.get("request_id")
            if not isinstance(text, str) or len(text) > 1000 or not isinstance(rid, str) or not (1 <= len(rid) <= 64):
                raise ValueError
        except Exception:
            return reply(400, {"error": "BAD_REQUEST", "expected": {"request_id": "1-64 chars", "input": "text ≤ 1000 chars"}})
        # LIVE decision from Identity — before any work, every time.
        st, d, why = W.identity_call("POST", "/api/v1/authz/check", {
            "ticket_id": p["tid"], "grant_id": p["gid"], "requester": p["req"], "capability": capability,
            "subject_identity_id": p["sub"], "request_sha256": p["dig"]})
        if st == "UNAVAILABLE" or (isinstance(st, int) and st >= 500):
            with W.lock:
                W.counters["unavailable"] += 1
            log("warn", "action.authority_unavailable", why or "Identity error", ticket_id=p["tid"], request_id=rid)
            return reply(503, {"error": "AUTHORITY_UNAVAILABLE", "detail": (why or "Identity could not decide") + "; nothing was done",
                                   "request_id": rid})
        if st != 200 or not d or d.get("decision") != "ALLOW":
            with W.lock:
                W.counters["denied"] += 1
            log("warn", "action.denied", (d or {}).get("reason", ""), ticket_id=p["tid"], request_id=rid, decision_id=(d or {}).get("decision_id"))
            return reply(403, {"error": "DENIED", "reason": (d or {}).get("reason"), "decision_id": (d or {}).get("decision_id"),
                                   "request_id": rid, "detail": "Identity did not allow this; nothing was done"})
        with W.lock:
            W.in_flight += 1
        try:
            if text.startswith("__slow__:") and "slow" in W.faults:
                time.sleep(min(10, float(text.split(":", 1)[1] or 0)))
            if text == "__crash__" and "crash" in W.faults:
                log("warn", "faults.crash", "TEST-ONLY crash injected mid-request")
                os._exit(3)
            out = W.transform(text)
            obs = {"observation_id": "obs_" + secrets.token_hex(8), "request_id": rid, "decision_id": d["decision_id"],
                   "grant_id": d["grant_id"], "ticket_id": p["tid"], "requester": p["req"], "subject_identity_id": p["sub"],
                   "capability": capability, "data_class": DATA_CLASS, "output": out,
                   "at": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime())}
            with W.lock:
                with open(os.path.join(W.data_dir, "observations.jsonl"), "a") as f:
                    f.write(json.dumps({k: obs[k] for k in obs if k != "output"}) + "\n")
                W.counters["allowed"] += 1
                W.latencies.append(int((time.time() - t0) * 1000))
            log("info", "action.completed", "synthetic observation produced", request_id=rid, decision_id=d["decision_id"])
            reply(200, {"result": "COMPLETED", "observation": obs})
        finally:
            with W.lock:
                W.in_flight -= 1


SERVER = None


def shutdown():
    deadline = time.time() + 15
    while time.time() < deadline:
        with W.lock:
            if W.in_flight == 0:
                break
        time.sleep(0.05)
    log("info", "feature.stopping", "clean stop; installation identity and data are kept")
    SERVER.shutdown()


def heartbeat():
    while True:
        time.sleep(1)
        with W.lock:
            W.heartbeat_seq += 1


def main():
    global W, SERVER
    W = Worker()
    SERVER = ThreadingHTTPServer(("127.0.0.1", 0), Handler)
    SERVER.daemon_threads = True
    port = SERVER.server_address[1]
    ep = {"api": API, "installation_id": W.installation_id, "launch_id": W.launch_id, "pid": os.getpid(),
          "url": "http://127.0.0.1:%d" % port, "started_at": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime())}
    write_atomic(os.path.join(W.runtime_dir, "endpoint.json"), json.dumps(ep).encode())
    threading.Thread(target=heartbeat, daemon=True).start()

    def on_term(*_):
        with W.lock:
            W.stopping = True
        threading.Thread(target=shutdown, daemon=True).start()
    signal.signal(signal.SIGTERM, on_term)
    signal.signal(signal.SIGINT, on_term)
    log("info", "feature.started", "listening on loopback; zero grants until Identity says otherwise",
        url=ep["url"], installation_id=W.installation_id, launch_id=W.launch_id)
    SERVER.serve_forever()
    log("info", "feature.stopped", "")


if __name__ == "__main__":
    main()
