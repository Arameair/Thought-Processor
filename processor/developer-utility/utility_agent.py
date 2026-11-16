#!/usr/bin/env python3
"""Developer Utility — the secure Developer entrance, as a Thought Processor feature.

Started, watched and stopped by Thought Processor; installed only after
Identity's Admin approves this exact pinned package; every protected
operation is authorized by Identity, live, for one identity at a time.

What the entrance can do (and nothing else):
  developer.connection.check   read-only: report this entrance's real state —
                               the Processor link (per-launch key), the Identity
                               link (Identity's own signed answer about this
                               installation, fetched fresh), the grants Identity
                               says it holds, and who asked. Needs its own grant.
It has no shell, no command execution, no file access outside its own data
folder, and no route to other Brain services: each of those would be its own
capability with its own Identity grant, added only after review.

Status is evidence, not a claim:
  RUNNING    Processor authenticated this launch with its per-launch key
  CONNECTED  Identity answered, signed, about THIS installation within the last 15 s
  READY      both of the above, the single-use ticket ledger is writable, not stopping

Started from the team's adapter (experiment/developer-utility-install-launch-20261029
@ 79ecd77, signed status/shutdown); the ticket / ledger / live-check / shutdown
code is the reviewed P1 Deterministic Worker's, reused as is.

Python 3.8+, standard library only. Run only by Thought Processor.
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
FEATURE_ID = "developer-utility"
CAP_CHECK = "developer.connection.check"
CAPABILITIES = {CAP_CHECK: "read-only report of this Developer entrance: Processor link, Identity link (signed, fresh), grants held, who asked. Input: empty."}
DATA_CLASS = "DEVELOPER ENTRANCE STATUS — read-only; no secrets, no other services, no external calls"
MAX_BODY = 16 * 1024
SKEW = 60
AUTHORITY_EVERY = 5
AUTHORITY_FRESH = 15


class EngineError(Exception):
    def __init__(self, code, detail):
        super().__init__(detail)
        self.code, self.detail = code, detail


def write_atomic(path, data):
    tmp = path + ".tmp-" + secrets.token_hex(4)
    fd = os.open(tmp, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
    try:
        with os.fdopen(fd, "wb") as f:
            f.write(data)
            f.flush()
            os.fsync(f.fileno())
        os.replace(tmp, path)
    except BaseException:
        try:
            os.remove(tmp)
        except OSError:
            pass
        raise


def summarize_self(view):
    """Only what a developer needs; never secrets (the self view holds none)."""
    ins = (view or {}).get("installation") or {}
    grants = []
    for g in (view or {}).get("grants") or []:
        gr = g.get("grant") or {}
        grants.append({"capability": gr.get("capability"), "for_identity": gr.get("subject_identity_id"), "requester": gr.get("requester"),
                       "status": g.get("status"), "expires_at": gr.get("expires_at")})
    return {"installation_state": ins.get("state"), "feature_id": ins.get("feature_id"), "artifact_sha256": ins.get("artifact_sha256"),
            "administered_by": ins.get("administered_by"), "active_grants": (view or {}).get("active_grants"), "grants": grants}


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


class Feature:
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
        self.authority = {"state": "CHECKING", "checked_at": None, "detail": "first check pending"}
        self.started = time.time()
        self.heartbeat_seq = 0
        self.in_flight = 0
        self.stopping = False
        self.counters = {"self_tests": 0, "requests": 0, "allowed": 0, "denied": 0, "unavailable": 0, "invalid": 0,
                         "authority_checks": 0, "authority_failures": 0}
        self.latencies = []

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
            log("error", "ledger.damaged", "ticket ledger unreadable; protected requests are refused until it is repaired")
            return None

    @staticmethod
    def _entry(v):
        return v if isinstance(v, dict) else {"exp": int(v), "state": "used"}

    def _remember_ticket(self, tid, exp):
        with self.lock:
            if self.ledger is None:
                return None, None
            if tid in self.ledger:
                return False, self._entry(self.ledger[tid]).get("state")
            now = int(time.time())
            nxt = {k: v for k, v in self.ledger.items() if self._entry(v)["exp"] > now - 600}
            nxt[tid] = {"exp": exp, "state": "checking"}
            try:
                write_atomic(self.ledger_path, json.dumps(nxt).encode())
            except OSError:
                return None, None
            self.ledger = nxt
            return True, None

    def _mark_ticket(self, tid, state):
        with self.lock:
            if self.ledger is None or tid not in self.ledger:
                return
            nxt = dict(self.ledger)
            e = self._entry(nxt[tid])
            e["state"] = state
            nxt[tid] = e
            try:
                write_atomic(self.ledger_path, json.dumps(nxt).encode())
                self.ledger = nxt
            except OSError:
                pass

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

    # ---- Identity link: Identity's own signed answer about THIS installation ----
    def check_authority(self):
        st, j, why = self.identity_call("GET", "/api/v1/installation/self", None)
        now = time.time()
        with self.lock:
            self.counters["authority_checks"] += 1
            if st == 200 and j and ((j.get("installation") or {}).get("installation_id") == self.installation_id):
                state = (j.get("installation") or {}).get("state")
                self.authority = {"state": "CONNECTED" if state == "INSTALLED" else "NOT_INSTALLED", "checked_at": now,
                                  "detail": "Identity %s signed an answer about this installation (state %s)" % (self.identity_system_id, state),
                                  "self": summarize_self(j)}
            else:
                self.counters["authority_failures"] += 1
                self.authority = {"state": "UNAVAILABLE", "checked_at": now, "detail": why or ("Identity answered HTTP %s" % st)}
            return dict(self.authority)

    def authority_view(self):
        with self.lock:
            a = dict(self.authority)
        fresh = a.get("checked_at") and time.time() - a["checked_at"] <= AUTHORITY_FRESH
        if a.get("state") == "CONNECTED" and not fresh:
            a["state"], a["detail"] = "STALE", "no signed answer from Identity in the last %d s" % AUTHORITY_FRESH
        if a.get("checked_at"):
            a["checked_at"] = time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime(a["checked_at"]))
        return a

    def self_test(self):
        """Standalone, nonprivileged: proves the two links work right now.
        No grant used; reports only facts it can verify."""
        a = self.check_authority()
        with self.lock:
            self.counters["self_tests"] += 1
            ledger_ok = self.ledger is not None
        checks = [{"check": "Processor link: this request was signed with this launch's key", "ok": True},
                  {"check": "Identity link: Identity signed a fresh answer about this installation", "ok": a.get("state") == "CONNECTED"},
                  {"check": "single-use ticket ledger is readable", "ok": ledger_ok}]
        ok = all(c["ok"] for c in checks)
        return {"result": "PASS" if ok else "FAIL", "data_class": DATA_CLASS, "input": "none (built-in checks)", "external_calls": 0,
                "grants_used": "none — standalone, nonprivileged self-test",
                "output": {"checks": checks, "identity_link": a.get("detail")}}

    def status(self):
        a = self.authority_view()
        with self.lock:
            lat = sorted(self.latencies[-100:])
            ready = not self.stopping and self.ledger is not None and a.get("state") == "CONNECTED"
            return {"api": API, "feature_id": FEATURE_ID, "version": VERSION, "installation_id": self.installation_id,
                    "launch_id": self.launch_id, "pid": os.getpid(), "state": "STOPPING" if self.stopping else "RUNNING",
                    "ready": ready, "entrance": {"running": not self.stopping, "connected": a.get("state") == "CONNECTED", "ready": ready},
                    "authority": {"state": a.get("state"), "checked_at": a.get("checked_at"), "detail": a.get("detail")},
                    "provider": {"kind": "none", "ready": "NOT_APPLICABLE — no AI provider"},
                    "heartbeat_seq": self.heartbeat_seq, "uptime_s": int(time.time() - self.started),
                    "in_flight": self.in_flight, "counters": dict(self.counters),
                    "latency_ms_p50": lat[len(lat) // 2] if lat else None, "latency_ms_max": lat[-1] if lat else None,
                    "trust_class": "TRUSTED_LOCAL_TEST_FIXTURE", "data_class": DATA_CLASS,
                    "heartbeat_note": "heartbeat = process liveness only; it is not an observation, grant or result"}

    def run_capability(self, capability, sub, text, rid, ticket):
        if capability != CAP_CHECK:
            raise EngineError("UNKNOWN_CAPABILITY", "unknown capability")
        if text.strip():
            raise EngineError("BAD_INPUT", "developer.connection.check takes no input; leave the Input box empty")
        a = self.check_authority()  # fresh, signed, at the moment of the request
        return {"entrance": "developer", "headline": "Developer entrance is %s" % ("CONNECTED" if a.get("state") == "CONNECTED" else a.get("state")),
                "service": {"feature_id": FEATURE_ID, "version": VERSION, "installation_id": self.installation_id, "launch_id": self.launch_id,
                            "pid": os.getpid(), "uptime_s": int(time.time() - self.started)},
                "processor_link": {"state": "AUTHENTICATED", "how": "every Processor call to this service is HMAC-signed with this launch's own key; answers are signed back"},
                "identity_link": {"state": a.get("state"), "identity_system_id": self.identity_system_id, "detail": a.get("detail"),
                                  "this_installation": (a.get("self") or {})},
                "this_request": {"requested_by": ticket.get("req"), "for_identity": sub, "capability": capability,
                                 "allowed_by": "Identity, live, just now (ticket + authz check)"},
                "can": sorted(CAPABILITIES), "cannot": ["run commands", "read or write files outside its own data folder",
                                                         "reach other Brain services (each needs its own capability and grant)"]}


W = None


class Handler(BaseHTTPRequestHandler):
    protocol_version = "HTTP/1.1"
    server_version = "developer-utility"

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
            return self.send(200, {"api": API, "capabilities": [{"id": c, "data_class": DATA_CLASS, "requires_grant": True, "input": d}
                                                                for c, d in CAPABILITIES.items()],
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
        ln = self.headers.get("X-TF-Nonce", "")
        ln = ln if re.match(r"^[0-9a-f]{16,64}$", ln) else None
        reply = lambda st, obj: self.send(st, obj, ln)
        t0 = time.time()
        with W.lock:
            if W.stopping:
                return reply(503, {"error": "STOPPING", "detail": "this feature is shutting down; nothing was done"})
            W.counters["requests"] += 1
        if capability not in CAPABILITIES:
            with W.lock:
                W.counters["invalid"] += 1
            return reply(404, {"error": "UNKNOWN_CAPABILITY", "capability": capability[:64]})
        p, why = W.verify_ticket(self.headers.get("X-TF-Ticket", ""), capability, body)
        if p is None:
            with W.lock:
                W.counters["invalid"] += 1
            log("warn", "action.refused", why, capability=capability)
            return reply(401, {"error": "TICKET_INVALID", "reason": why, "detail": "nothing was done"})
        seen, prev = W._remember_ticket(p["tid"], p["exp"])
        if seen is None:
            return reply(503, {"error": "REPLAY_LEDGER_UNAVAILABLE", "detail": "nothing was done"})
        if seen is False:
            with W.lock:
                W.counters["invalid"] += 1
            if prev == "spent_authority_unavailable":
                log("info", "action.spent_ticket", "ticket was spent during an Identity outage; not executed", ticket_id=p["tid"])
                return reply(409, {"error": "TICKET_SPENT_DURING_OUTAGE", "ticket_id": p["tid"],
                                   "detail": "this ticket was used once while Identity was unavailable, so the request was NOT carried out. "
                                             "Tickets are single-use: ask Identity for a new ticket. This is not resent automatically."})
            log("warn", "action.replay", "ticket already used", ticket_id=p["tid"], previous=prev or "used")
            return reply(409, {"error": "TICKET_REPLAYED", "ticket_id": p["tid"], "detail": "this ticket was already used; nothing was done"})
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
            W._mark_ticket(p["tid"], "spent_authority_unavailable")
            log("warn", "action.authority_unavailable", why or "Identity error", ticket_id=p["tid"], request_id=rid)
            return reply(503, {"error": "AUTHORITY_UNAVAILABLE", "detail": (why or "Identity could not decide") +
                               "; nothing was done. This ticket is now spent — ask for a new one when Identity is back.",
                               "request_id": rid, "ticket_spent": True})
        if st != 200 or not d or d.get("decision") != "ALLOW":
            W._mark_ticket(p["tid"], "denied")
            with W.lock:
                W.counters["denied"] += 1
            log("warn", "action.denied", (d or {}).get("reason", ""), ticket_id=p["tid"], request_id=rid, decision_id=(d or {}).get("decision_id"))
            return reply(403, {"error": "DENIED", "reason": (d or {}).get("reason"), "decision_id": (d or {}).get("decision_id"),
                               "request_id": rid, "detail": "Identity did not allow this; nothing was done"})
        W._mark_ticket(p["tid"], "executed")
        with W.lock:
            W.in_flight += 1
        try:
            # For whom: ONLY the identity Identity signed into the ticket.
            try:
                out = W.run_capability(capability, p["sub"], text, rid, p)
            except EngineError as e:
                log("warn", "action.refused_by_service", e.code, request_id=rid, decision_id=d["decision_id"], capability=capability)
                return reply(422, {"error": e.code, "detail": e.detail, "request_id": rid, "decision_id": d["decision_id"],
                                   "note": "Identity allowed the request, but the Developer entrance could not carry it out; nothing changed"})
            obs = {"observation_id": "obs_" + secrets.token_hex(8), "request_id": rid, "decision_id": d["decision_id"],
                   "grant_id": d["grant_id"], "ticket_id": p["tid"], "requester": p["req"], "subject_identity_id": p["sub"],
                   "capability": capability, "data_class": DATA_CLASS, "output": out,
                   "at": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime())}
            with W.lock:
                with os.fdopen(os.open(os.path.join(W.data_dir, "observations.jsonl"), os.O_WRONLY | os.O_CREAT | os.O_APPEND, 0o600), "a") as f:
                    f.write(json.dumps({k: obs[k] for k in obs if k != "output"}) + "\n")
                W.counters["allowed"] += 1
                W.latencies.append(int((time.time() - t0) * 1000))
            log("info", "action.completed", capability + " " + str(out.get("outcome", "ok")), request_id=rid, decision_id=d["decision_id"])
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
    log("info", "feature.stopping", "clean stop; installation identity is kept")
    SERVER.shutdown()


def heartbeat():
    while True:
        time.sleep(1)
        with W.lock:
            W.heartbeat_seq += 1


def authority_monitor():
    while True:
        W.check_authority()
        time.sleep(AUTHORITY_EVERY)


def main():
    global W, SERVER
    os.umask(0o077)  # everything this feature creates is private to its user
    W = Feature()
    SERVER = ThreadingHTTPServer(("127.0.0.1", 0), Handler)
    SERVER.daemon_threads = True
    port = SERVER.server_address[1]
    ep = {"api": API, "installation_id": W.installation_id, "launch_id": W.launch_id, "pid": os.getpid(),
          "url": "http://127.0.0.1:%d" % port, "started_at": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime())}
    write_atomic(os.path.join(W.runtime_dir, "endpoint.json"), json.dumps(ep).encode())
    threading.Thread(target=heartbeat, daemon=True).start()
    threading.Thread(target=authority_monitor, daemon=True).start()

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
