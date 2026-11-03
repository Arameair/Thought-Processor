"""Scripted stand-in for llama.cpp (streaming, reasoning_content). FIXTURE ONLY — not real AI.

python tests/fake_model.py PORT
"""

import json
import re
import sys
import time
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer


def script(messages):
    system = messages[0]["content"]
    last = messages[-1]["content"]
    who = re.search(r"=== MIND: mind/who-i-am.md ===\n(.*?)(?:\n===|\Z)", system, re.S)
    first_line = ""
    if who:
        bullets = [l for l in who.group(1).splitlines() if l.startswith("- ")]
        first_line = bullets[0][2:] if bullets else ""
    if "improve" in last.lower() and not last.startswith("["):
        return ("Dustin wants me to think better. My how-i-think file could say: plan before acting.",
                "Here's one change I'd make to how I think.\n"
                "REMEMBER: Dustin wants me to keep improving how I think.\n"
                "CHANGE MIND: mind/who-i-am.md\nFIND: - I am a teammate, not a boss.\nACTION: after\n"
                "<<<TEXT\n- Before I act, I make a one-line plan.\nTEXT>>>")
    if "rewrite yourself" in last.lower():
        return ("Rewrite everything.", "Fresh start.\nCHANGE MIND: mind/who-i-am.md\n```markdown\n# Me\n```")
    if "what do you remember" in last.lower():
        mem = system.split("YOUR MEMORY", 1)[1].split("\n\n", 1)[0] if "YOUR MEMORY" in system else ""
        return ("Check my memory.", "From my memory:" + mem.split(":", 1)[-1])
    if "[PROJECT projects/hello-tass" in last:
        return ("ChatGPT's project only writes inside its own folder. Safe to propose.",
                "It greets us and writes status.json in its own folder.\nRUN: projects/hello-tass")
    if "sleep" in last.lower():
        return ("A slow job.", "This takes a while.\nRUN: code\n```python\nimport time\ntime.sleep(60)\n```")
    if "project" in last.lower() and not last.startswith("["):
        return ("Dustin mentioned a project. Open it first.", "Let me read it first.\nOPEN: projects/hello-tass")
    if "[BOOK library/01-look-around.md" in last:
        return ("The book has three cells. Cell 1 lists our shared folder, a safe first look.",
                "Cell 1 only reads. I'll explain why.\nRUN: library/01-look-around.md#1")
    if "[Spark] The run finished" in last:
        res = next((m["content"] for m in reversed(messages) if m["content"].startswith("[RESULT")), "")
        first = res.splitlines()[1] if len(res.splitlines()) > 1 else "(nothing)"
        return ("The result came back. Summarize it and suggest the next step.",
                f"It ran. First line of the output: `{first}`. Next we could make our home with book 02.")
    if "[Spark] Your mind files just changed" in last:
        return ("My who-i-am file changed. Compare the newest version.",
                f"My mind changed. Now my first rule says: “{first_line}”.")
    if "own code" in last.lower():
        return ("Dustin wants me to write code myself.",
                "Here's a tiny check of our kernel.\nRUN: code\n```python\nanswer = sum(range(10))\nprint('answer =', answer)\n```")
    return ("Spark: Dustin said hello. Trace: I have my mind files and a library. "
            "Thought: before building anything, look around (how-i-think says look before changing).",
            "Hi Dustin! Before we build anything, I want to see what's already in our space.\n"
            "OPEN: library/01-look-around.md")


class H(BaseHTTPRequestHandler):
    def log_message(self, *a):
        pass

    def _json(self, obj):
        b = json.dumps(obj).encode()
        self.send_response(200)
        self.send_header("Content-Length", str(len(b)))
        self.end_headers()
        self.wfile.write(b)

    def do_GET(self):
        if self.path == "/health":
            return self._json({"status": "ok"})
        if self.path == "/props":
            return self._json({"default_generation_settings": {"n_ctx": 16384}})
        self.send_response(404)
        self.end_headers()

    def do_POST(self):
        body = json.loads(self.rfile.read(int(self.headers["Content-Length"])))
        if self.path == "/apply-template":
            return self._json({"prompt": "\n".join(m["content"] for m in body["messages"])})
        if self.path == "/tokenize":
            return self._json({"tokens": list(range(len(body["content"]) // 4 + 1))})
        if self.path != "/v1/chat/completions":
            self.send_response(404)
            self.end_headers()
            return
        thought, reply = script(body["messages"])
        thinking = body.get("chat_template_kwargs", {}).get("enable_thinking", True)
        self.send_response(200)
        self.send_header("Content-Type", "text/event-stream")
        self.end_headers()

        def send(delta, finish=None):
            chunk = {"choices": [{"delta": delta, "finish_reason": finish}]}
            self.wfile.write(f"data: {json.dumps(chunk)}\n\n".encode())
            self.wfile.flush()

        if thinking:
            for word in thought.split(" "):
                send({"reasoning_content": word + " "})
                time.sleep(0.03)
        for word in re.split(r"(\s+)", reply):
            if word:
                send({"content": word})
                time.sleep(0.01)
        send({}, "stop")
        self.wfile.write(b"data: [DONE]\n\n")


if __name__ == "__main__":
    srv = ThreadingHTTPServer((sys.argv[2] if len(sys.argv) > 2 else "127.0.0.1", int(sys.argv[1])), H)
    print("fake model on", sys.argv[1], flush=True)
    srv.serve_forever()
