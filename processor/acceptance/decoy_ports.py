"""Decoy listeners for port-conflict tests (M4). Pretends to be an UNRELATED
Identity / Processor / GitHub fixture that some other person or older build
left running. It answers health checks plausibly and refuses everything else.

  python3 -I acceptance/decoy_ports.py 7790 7860 8790     (Ctrl+C to stop)

Never stops or touches anything else; if a port is already taken it says so
and leaves it alone (that real program is then the "decoy").
"""
import json, socket, sys, threading
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer


class H(BaseHTTPRequestHandler):
    def log_message(self, *a):
        pass

    def reply(self, code, obj):
        b = json.dumps(obj).encode()
        self.send_response(code)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(b)))
        self.end_headers()
        self.wfile.write(b)

    def do_GET(self):
        if self.path == "/api/health":
            port = self.server.server_address[1]
            app = "identity" if port != 7860 else "thought-processor"
            return self.reply(200, {"app": app, "version": "0.0.0-DECOY", "api": "identity/2-alpha", "protocol": "identity/1-alpha",
                                    "note": "DECOY — an unrelated program; tests must not use it"})
        self.reply(404, {"error": "decoy"})

    def do_POST(self):
        self.rfile.read(int(self.headers.get("Content-Length") or 0))
        self.reply(401, {"error": "decoy: not your Identity"})


servers = []
for p in [int(x) for x in sys.argv[1:]]:
    try:
        s = ThreadingHTTPServer(("127.0.0.1", p), H)
    except OSError:
        print(f"port {p} already in use by another program — left alone (it acts as the decoy)", flush=True)
        continue
    servers.append(s)
    threading.Thread(target=s.serve_forever, daemon=True).start()
    print(f"decoy listening on 127.0.0.1:{p}", flush=True)
threading.Event().wait()
