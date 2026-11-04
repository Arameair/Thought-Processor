# TEST FIXTURE standing in for llama.cpp's server (no real model or GPU in the test sandbox).
# It answers like llama.cpp (/health, /v1/chat/completions) and can be slow on purpose
# (FIXTURE_DELAY seconds for any request mentioning "slow") so a job outlasts several pulses.
# It counts its own requests in /tmp/requests so a test can prove nothing ran twice.
import json, os, sys, threading, time
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

DELAY = float(os.environ.get("FIXTURE_DELAY", "45"))
lock = threading.Lock()
count = 0

class H(BaseHTTPRequestHandler):
    def log_message(self, *a):
        pass

    def reply(self, code, obj):
        b = json.dumps(obj).encode()
        self.send_response(code)
        self.send_header("Content-Type", "application/json")
        self.end_headers()
        self.wfile.write(b)

    def do_GET(self):
        if self.path == "/fixture/requests":
            return self.reply(200, {"requests": count})
        self.reply(200, {"status": "ok"})

    def do_POST(self):
        global count
        req = json.loads(self.rfile.read(int(self.headers.get("Content-Length", "0") or 0)) or b"{}")
        msgs = req.get("messages", [])
        allm = "\n".join(str(m.get("content", "")) for m in msgs)
        last = str(msgs[-1].get("content", "")) if msgs else ""
        with lock:
            count += 1
        if "IMPORTANCE: <a whole number" in allm:
            text = "IMPORTANCE: 6\nTOPIC: local work"
        elif "HOW TO ANSWER" in allm:
            if "slow" in last:
                time.sleep(DELAY)
            if "section" in last:
                text = ("REPLY: (TEST FIXTURE) I think my document has 4 sections.\nACTION: tass_sections\n"
                        "PREDICTION: 4\nWHY: I can see the headings.")
            else:
                text = "REPLY: (TEST FIXTURE) Hello.\nACTION: NONE\nPREDICTION: NONE\nWHY: Just a greeting."
        else:
            text = "TEST FIXTURE"
        self.reply(200, {"choices": [{"message": {"content": text, "reasoning_content": "fixture reasoning"}}]})

ThreadingHTTPServer(("0.0.0.0", int(sys.argv[1]) if len(sys.argv) > 1 else 8081), H).serve_forever()
