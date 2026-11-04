# TEST FIXTURE standing in for llama.cpp's server (no real model or GPU in the test sandbox).
# It answers like llama.cpp (/health, /v1/chat/completions) and can be slow on purpose
# (FIXTURE_DELAY seconds for any request mentioning "slow") so a job outlasts several pulses.
# It counts its own requests in /tmp/requests so a test can prove nothing ran twice.
import json, os, sys, threading, time
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

DELAY = float(os.environ.get("FIXTURE_DELAY") or 45)
lock = threading.Lock()
count = 0

# TEST FIXTURE answers for lesson checks, from a lookup table (a real model reads the lesson).
# The first attempt at "attempts" is wrong on purpose, so the retry path is exercised.
TABLE = [("seconds", "20 seconds"), ("still running", "I say CONTINUE"), ("locked", "I am BLOCKED"),
         ("which folder", "the lessons folder"), ("approve a lesson", "my owner"), ("final decisions", "my owner"),
         ("may i say", "only after I see the result"), ("new lesson reach", "by a pull request"), ("color", "blue")]
tries = {}

def study(allm):
    import re
    title = (re.search(r'Study this lesson: "([^"]+)"', allm) or [None, "a lesson"])[1]
    lines = []
    for n, q in re.findall(r"^Check (\d+): (.*)$", allm, re.M):
        ql = q.lower()
        if "attempts" in ql:
            tries[title] = tries.get(title, 0) + 1
            a = "5" if tries[title] == 1 else "3 attempts"
        else:
            a = next((v for k, v in TABLE if k in ql), "I don't know")
        lines.append(f"ANSWER {n}: {a}")
    return ("REPLY: (TEST FIXTURE) I studied it.\n" + "\n".join(lines) +
            f"\nACTION: NONE\nPREDICTION: NONE\nWHY: studying\nREMEMBER: (TEST FIXTURE) I can explain {title}")

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
        elif "HOW TO ANSWER" in allm and "Study this lesson" in last:
            text = study(allm)
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
