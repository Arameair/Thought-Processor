# TEST FIXTURE standing in for llama.cpp's server (no real model or GPU in the test sandbox).
# It answers like llama.cpp (/health, /v1/chat/completions) and can be slow on purpose
# (FIXTURE_DELAY seconds for any request mentioning "slow") so a job outlasts several pulses.
# It counts its own requests in /tmp/requests so a test can prove nothing ran twice.
import json, os, sys, threading, time
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

DELAY = float(os.environ.get("FIXTURE_DELAY") or 45)
lock = threading.Lock()
count = 0
# Stage 2: owner nudges (Engine prompt: the Think prompt + "The owner asked you to pay attention").
# Counted on their own and the last one kept, so a test can prove ONE model request per nudge and see
# exactly what the Engine sent. Attention text containing "slow" waits FIXTURE_DELAY seconds.
nudges = 0
last_nudge = ""

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

# Engine lesson mode (Thought-engine lesson_cycle.go: "OWNER-SUPERVISED LESSON ANSWERS"): numbered
# ANSWER lines only. FIXTURE_LESSON_FAIL_FIRST="<title>" makes the first FIXTURE_LESSON_FAIL_COUNT
# (default 2 = request + the Engine's one retry) replies for that lesson REMEMBER-only, the exact
# shape the owner's real Nemotron produced on Nov 5 (Thought-Processor #71 / Thought-engine #12).
FAIL_TITLE = os.environ.get("FIXTURE_LESSON_FAIL_FIRST", "")
FAIL_COUNT = int(os.environ.get("FIXTURE_LESSON_FAIL_COUNT") or 2)
lesson_calls = {}

def lesson_answers(allm):
    import re
    title = (re.search(r"^# (.+)$", allm, re.M) or [None, "a lesson"])[1].strip()
    with lock:
        lesson_calls[title] = lesson_calls.get(title, 0) + 1
        n = lesson_calls[title]
    if FAIL_TITLE and title == FAIL_TITLE and n <= FAIL_COUNT:
        return "REMEMBER: (TEST FIXTURE) I study my lessons and keep what I learn about my work pulse."
    out = []
    for num, q in re.findall(r"^Check (\d+): (.*)$", allm, re.M):
        ql = q.lower()
        out.append(f"ANSWER {num}: " + next((v for k, v in [("attempts", "3 attempts")] + TABLE if k in ql), "I don't know"))
    return "\n".join(out)

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
            return self.reply(200, {"requests": count, "nudges": nudges})
        if self.path == "/fixture/last-nudge":
            return self.reply(200, {"prompt": last_nudge})
        self.reply(200, {"status": "ok"})

    def do_POST(self):
        global count, nudges, last_nudge
        req = json.loads(self.rfile.read(int(self.headers.get("Content-Length", "0") or 0)) or b"{}")
        msgs = req.get("messages", [])
        allm = "\n".join(str(m.get("content", "")) for m in msgs)
        last = str(msgs[-1].get("content", "")) if msgs else ""
        with lock:
            count += 1
        if "You are developing an ongoing thought" in allm and "The owner asked you to pay attention" in allm:
            with lock:
                nudges += 1
                last_nudge = allm
            if "slow" in allm.split("The owner asked you to pay attention", 1)[1]:
                time.sleep(DELAY)
            text = ("NOTE: (TEST FIXTURE) Following the owner's pointer, check whether the pattern holds "
                    f"on the next session (nudge {nudges}).\nSTATE: keep")
        elif "OWNER-SUPERVISED LESSON ANSWERS" in allm:
            text = lesson_answers(allm)
        elif "IMPORTANCE: <a whole number" in allm:
            text = "IMPORTANCE: 6\nTOPIC: local work"
        elif "HOW TO ANSWER" in allm and "Study this lesson" in last:
            text = study(allm)
        elif "HOW TO ANSWER" in allm:
            if "slow" in last:
                time.sleep(DELAY)
            if "INFORMATION YOU ASKED FOR" in allm:
                import re as _re
                got = _re.search(r"^title: (.+)$", allm, _re.M)
                text = ("REPLY: (TEST FIXTURE) My Core gave me the lesson metadata I asked for: " + (got.group(1) if got else "?") +
                        "\nACTION: NONE\nPREDICTION: NONE\nWHY: Junction data arrived.")
            elif "lesson-metadata" in last:
                text = ("REPLY: (TEST FIXTURE) I need to look that lesson up first.\nNEED lesson-metadata: lessons/claude/01-my-work-pulse.md"
                        "\nACTION: NONE\nPREDICTION: NONE\nWHY: I asked my Core for the lesson's metadata.")
            elif "section" in last:
                text = ("REPLY: (TEST FIXTURE) I think my document has 4 sections.\nACTION: tass_sections\n"
                        "PREDICTION: 4\nWHY: I can see the headings.")
            else:
                text = "REPLY: (TEST FIXTURE) Hello.\nACTION: NONE\nPREDICTION: NONE\nWHY: Just a greeting."
        else:
            text = "TEST FIXTURE"
        self.reply(200, {"choices": [{"message": {"content": text, "reasoning_content": "fixture reasoning"}}]})

ThreadingHTTPServer(("0.0.0.0", int(sys.argv[1]) if len(sys.argv) > 1 else 8081), H).serve_forever()
