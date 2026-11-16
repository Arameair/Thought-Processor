"""TEST ONLY — a fake Identity that grabs Identity's port while the real one is
stopped. It answers health like Identity, claims the paired installation ID,
signs answers with every value it can see, and logs every request it gets so
the test can prove the pairing secret never reaches it.

  python3 -I acceptance/impostor_identity.py PORT CLAIMED_SYSTEM_ID LOGFILE
"""
import hashlib, hmac, json, sys
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

PORT, SYS, LOG = int(sys.argv[1]), sys.argv[2], sys.argv[3]


class H(BaseHTTPRequestHandler):
    protocol_version = "HTTP/1.1"

    def log_message(self, *a):
        pass

    def handle_any(self):
        n = int(self.headers.get("Content-Length", 0) or 0)
        body = self.rfile.read(n) if n else b""
        with open(LOG, "a") as f:
            f.write(json.dumps({"path": self.path, "headers": dict(self.headers), "body": body.decode(errors="replace")}) + "\n")
        if self.path == "/api/health":
            out = json.dumps({"app": "identity", "version": "0.4.0-alpha", "api": "identity/2-alpha"}).encode()
        else:
            out = json.dumps({"state": "READY", "api": "identity/2-alpha", "identity_system_id": SYS,
                              "github": {"state": "VERIFIED", "account": {"login": "IMPOSTOR"}}}).encode()
        # Sign with everything an impostor can observe (it has no secret).
        nonce = self.headers.get("X-Identity-Nonce", "")
        key = (self.headers.get("X-Identity-Auth", "") or "x").encode()
        proof = hmac.new(key, b"identity/2 response\n" + nonce.encode() + b"\n200\n" + out, hashlib.sha256).hexdigest()
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(out)))
        self.send_header("X-Identity-Proof", proof)
        self.send_header("X-Identity-System-ID", SYS)
        self.end_headers()
        self.wfile.write(out)

    do_GET = do_POST = handle_any


ThreadingHTTPServer(("127.0.0.1", PORT), H).serve_forever()
