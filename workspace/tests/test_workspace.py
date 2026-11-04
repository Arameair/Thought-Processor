"""Tests for the Thought Processor workspace. FIXTURE model only — not evidence about real Nemotron.

Run:  cd workspace && python -m unittest discover -s tests -v
(needs requirements.txt installed for the kernel test)
"""

import json
import os
import socket
import subprocess
import sys
import tempfile
import threading
import time
import unittest
import urllib.request
import urllib.error
from pathlib import Path

HERE = Path(__file__).resolve().parent
sys.path.insert(0, str(HERE.parent))

from tass_workspace.mind import Mind  # noqa: E402
from tass_workspace.thinker import build_system, parse_action  # noqa: E402


def free_port():
    s = socket.socket()
    s.bind(("127.0.0.1", 0))
    port = s.getsockname()[1]
    s.close()
    return port


MIND_WHO = "# Who I am\n\n- I am curious.\n- I am a teammate, not a boss.\n- I never pretend something happened.\n"


def make_mirror(root: Path):
    files = root / "files"
    (files / "mind").mkdir(parents=True)
    (files / "library").mkdir(parents=True)
    (files / "mind/who-i-am.md").write_text(MIND_WHO)
    (files / "library/01-look-around.md").write_text(
        "# Look around\n\nSee what's here.\n\n```python\nprint('cell one')\n```\n\n```python\nx = 41 + 1\nprint(x)\n```\n")
    (files / "library/README.md").write_text("# Library\n")
    (files / "projects/hello-tass").mkdir(parents=True)
    (files / "projects/hello-tass/project.md").write_text(
        "# Hello TASS\n\nA first project that greets us.\n\nentry: main.py\n")
    (files / "projects/hello-tass/main.py").write_text(
        "import os\nopen('status.json','w').write('{}')\nprint('Hello from hello-tass in', os.path.basename(os.getcwd()))\n")
    (files / "projects/hello-tass/data/notes.txt").parent.mkdir()
    (files / "projects/hello-tass/data/notes.txt").write_text("hi\n")
    (root / "manifest.json").write_text(json.dumps({"repo": "Arameair/TASS", "branch": "alpha", "synced_at": time.time(),
                                                    "last_change_at": 1.0, "last_changed": [], "files": {
        "mind/who-i-am.md": {"sha": "abc", "repo": "Arameair/TASS", "branch": "alpha"}}}))


class ActionTests(unittest.TestCase):
    def test_open(self):
        text, a = parse_action("Let me look.\nOPEN: library/01-look-around.md")
        self.assertEqual((text, a.kind, a.book), ("Let me look.", "open", "library/01-look-around.md"))

    def test_run_book_cell_with_formatting(self):
        _, a = parse_action("Ok.\n**RUN:** `library/02-make-our-home.md#2`")
        self.assertEqual((a.kind, a.book, a.cell), ("run", "library/02-make-our-home.md", 2))

    def test_run_own_code(self):
        text, a = parse_action("Try this.\nRUN: code\n```python\nprint(1)\n```\nThanks")
        self.assertEqual((a.kind, a.code), ("run", "print(1)"))
        self.assertEqual(text, "Try this.\n\nThanks")

    def test_no_action_and_last_wins(self):
        self.assertIsNone(parse_action("just talking")[1])
        _, a = parse_action("OPEN: library/a.md\nthen\nRUN: library/a.md#1")
        self.assertEqual(a.kind, "run")


class MindTests(unittest.TestCase):
    def test_catalog_book_and_prompt(self):
        with tempfile.TemporaryDirectory() as d:
            make_mirror(Path(d))
            m = Mind(d)
            cat = m.catalog()
            self.assertEqual([b.path for b in cat], ["library/01-look-around.md"])  # README skipped
            self.assertEqual((cat[0].title, cat[0].summary, len(cat[0].cells)), ("Look around", "See what's here.", 2))
            self.assertIn("CELL 2:", m.numbered(cat[0]))
            with self.assertRaises(ValueError):
                m.book("../secrets/github_token")
            system = build_system(m.mind_files(), cat, ["notes/"], [])
            self.assertIn("I am curious.", system)
            self.assertIn("library/01-look-around.md — Look around", system)
            self.assertIn("RUN: library/<file>#<n>", system)


class ProjectTests(unittest.TestCase):
    def test_project_parse_and_actions(self):
        with tempfile.TemporaryDirectory() as d:
            make_mirror(Path(d))
            m = Mind(d)
            p = m.project("hello-tass")
            self.assertEqual((p.title, p.entry, p.files), ("Hello TASS", "main.py", ["data/notes.txt", "main.py", "project.md"]))
            self.assertIn("Hello from hello-tass", m.entry_code(p))
            for bad in ("../mind", ".hidden", "nope"):
                with self.assertRaises(ValueError):
                    m.project(bad)
            self.assertIn("projects/hello-tass — Hello TASS", build_system([], [], [], [], m.projects()))
        a = parse_action("ok\nRUN: projects/hello-tass")[1]
        self.assertEqual((a.kind, a.project), ("run", "hello-tass"))
        a = parse_action("OPEN: projects/hello-tass/")[1]
        self.assertEqual((a.kind, a.project), ("open", "hello-tass"))


class RestartTests(unittest.TestCase):
    def test_starts_with_a_saved_timeline_from_an_older_version(self):
        # Before the fix, decision/result events saved a proposal id (text) as "id", and the next start crashed.
        with tempfile.TemporaryDirectory() as d:
            root = Path(d)
            make_mirror(root / "mirror")
            (root / "data").mkdir()
            (root / "data/timeline.json").write_text(json.dumps([
                {"id": 1, "type": "user", "text": "hi"}, {"id": "a1b2c3d4", "type": "decision", "status": "running"},
                {"id": 7, "type": "reply", "text": "ok"}]))
            env = dict(os.environ, TASS_MIRROR=str(root / "mirror"), TASS_SHARED=str(root / "shared"),
                       TASS_DATA=str(root / "data"), TASS_JUPYTER="0")
            old = dict(os.environ)
            os.environ.update(env)
            try:
                from tass_workspace.server import Workspace
                ws = Workspace()
                self.assertEqual(ws.next_id, 8)
                ev = ws.emit("decision", pid="ffff0000", status="ran", id="should-not-win")
                self.assertEqual((ev["id"], ev["pid"]), (8, "ffff0000"))
            finally:
                os.environ.clear()
                os.environ.update(old)


    def test_atrium_chat_only_offline_model_reports_correlated_error(self):
        # Deterministic model failure: no reply, proposal, or fabricated success.
        with tempfile.TemporaryDirectory() as d:
            root = Path(d)
            make_mirror(root / "mirror")
            (root / "shared").mkdir()
            previous = dict(os.environ)
            os.environ.update(TASS_MIRROR=str(root / "mirror"),
                              TASS_SHARED=str(root / "shared"),
                              TASS_DATA=str(root / "data"),
                              TASS_JUPYTER="0")
            try:
                from tass_workspace.server import Workspace
                ws = Workspace()
                ws.model.measure = lambda messages: (16, 16384)

                def offline(*args, **kwargs):
                    raise ConnectionError("fixture model is offline")
                ws.model.stream = offline
                rid = "0123456789abcdef"
                ws.turn("user", "Hello", chat_only=True, request_id=rid)
                self.assertTrue(any(e["type"] == "error" and e.get("request_id") == rid
                                    and "offline" in e["text"] for e in ws.timeline))
                self.assertFalse(any(e["type"] == "reply" and e.get("request_id") == rid
                                     for e in ws.timeline))
                self.assertFalse(ws.proposals)
                self.assertTrue(any(r["event"] == "chat_failed" and r["id"] == rid
                                    for r in ws.read_audit()))
            finally:
                os.environ.clear()
                os.environ.update(previous)


class EndToEndTests(unittest.TestCase):
    """Real server + real (private) Jupyter kernel + scripted fake model."""

    @classmethod
    def setUpClass(cls):
        cls.tmp = tempfile.TemporaryDirectory()
        root = Path(cls.tmp.name)
        make_mirror(root / "mirror")
        cls.model_port, cls.port = free_port(), free_port()
        cls.model = subprocess.Popen([sys.executable, str(HERE / "fake_model.py"), str(cls.model_port)],
                                     stdout=subprocess.DEVNULL)
        env = dict(os.environ, TASS_AUTO_LIMIT="2", TASS_INBOX=str(root / "inbox"), TASS_OUTBOX=str(root / "outbox"), TASS_MODEL_URL=f"http://127.0.0.1:{cls.model_port}", TASS_MIRROR=str(root / "mirror"),
                   TASS_SHARED=str(root / "shared"), TASS_DATA=str(root / "data"), TASS_PORT=str(cls.port),
                   TASS_JUPYTER="0", PYTHONPATH=str(HERE.parent))
        cls.server = subprocess.Popen([sys.executable, "-m", "tass_workspace"], env=env, stdout=subprocess.DEVNULL)
        cls.root = root
        deadline = time.time() + 60
        while time.time() < deadline:
            try:
                if cls.get("/api/status")["kernel"] != "not started":
                    break
            except OSError:
                pass
            time.sleep(0.5)

    @classmethod
    def tearDownClass(cls):
        for p in (cls.server, cls.model):
            p.terminate()
            p.wait(10)
        cls.tmp.cleanup()

    def setUp(self):
        # Each test starts from an idle TASS and an empty conversation.
        self.wait_for(lambda s: not s["status"]["busy"] and not s["status"]["running"])
        time.sleep(0.5)
        self.wait_for(lambda s: not s["status"]["busy"])
        self.post("/api/reset", {})

    @classmethod
    def get(cls, path):
        with urllib.request.urlopen(f"http://127.0.0.1:{cls.port}{path}", timeout=10) as r:
            return json.load(r)

    def post(self, path, body, origin=None):
        headers = {"Content-Type": "application/json"}
        if origin:
            headers["Origin"] = origin
        req = urllib.request.Request(f"http://127.0.0.1:{self.port}{path}", data=json.dumps(body).encode(),
                                     headers=headers)
        with urllib.request.urlopen(req, timeout=10) as r:
            return json.load(r)

    def wait_for(self, pred, timeout=60):
        deadline = time.time() + timeout
        while time.time() < deadline:
            st = self.get("/api/state")
            if pred(st):
                return st
            time.sleep(0.3)
        self.fail("timed out")

    def test_conversation_open_propose_approve_result(self):
        self.post("/api/message", {"text": "Hi TASS"})
        st = self.wait_for(lambda s: s["proposals"])
        types = [e["type"] for e in st["timeline"]]
        for t in ("user", "spark", "trace", "thought", "reply", "book", "proposal"):
            self.assertIn(t, types)
        thought = next(e for e in st["timeline"] if e["type"] == "thought")
        self.assertIn("look around", thought["text"])   # reasoning was captured
        p = st["proposals"][0]
        self.assertEqual(p["source"], "library/01-look-around.md · cell 1")
        self.post("/api/proposal", {"id": p["id"], "decision": "approve"})
        st = self.wait_for(lambda s: any(e["type"] == "reply" and "Next we could" in e["text"] for e in s["timeline"]))
        result = next(e for e in st["timeline"] if e["type"] == "result")
        self.assertTrue(result["ok"])
        self.assertIn("cell one", result["text"])
        journal = json.loads((self.root / "shared/journal/tass-journal.ipynb").read_text())
        self.assertEqual(journal["cells"][-1]["outputs"][0]["text"], "cell one\n")

    def test_project_open_approve_runs_in_its_folder_and_audits(self):
        self.post("/api/message", {"text": "ChatGPT added a project, can you look?"})
        st = self.wait_for(lambda s: any(p.get("kind") == "project" for p in s["proposals"]))
        p = next(p for p in st["proposals"] if p.get("kind") == "project")
        self.assertEqual(p["files"], ["data/notes.txt", "main.py", "project.md"])
        self.assertIn("print('Hello from hello-tass", p["code"])
        self.post("/api/proposal", {"id": p["id"], "decision": "approve"})
        st = self.wait_for(lambda s: any(e["type"] == "result" and e.get("pid") == p["id"] for e in s["timeline"]))
        res = next(e for e in st["timeline"] if e["type"] == "result" and e.get("pid") == p["id"])
        self.assertTrue(res["ok"], res)
        self.assertIn("Hello from hello-tass in hello-tass", res["text"])        # ran inside its folder
        dest = self.root / "shared/projects/hello-tass"
        self.assertTrue((dest / "status.json").exists() and (dest / "data/notes.txt").exists())
        events = [(r["event"], r["label"]) for r in self.get("/api/audit")]
        for ev in ("proposed", "approved", "ran"):
            self.assertIn((ev, "RUN projects/hello-tass"), events)
        self.assertEqual(len(next(r for r in self.get("/api/audit") if r["event"] == "ran")["code_sha256"]), 64)

    def test_controls_refuse_autonomy_and_stop(self):
        # Turn projects off: TASS's RUN is refused, never proposed.
        self.post("/api/settings", {"allow": {"projects": False}})
        self.post("/api/message", {"text": "Open the project again please"})
        st = self.wait_for(lambda s: any(e["type"] == "notice" and "turned off projects" in e["text"] for e in s["timeline"]))
        self.assertFalse(st["proposals"])
        self.assertIn("refused", [r["event"] for r in self.get("/api/audit")])
        self.post("/api/settings", {"allow": {"projects": True}, "autonomy": False})
        # Autonomy off + Stop: a long run is interrupted, and TASS does not react on its own.
        self.wait_for(lambda s: not s["status"]["busy"])
        self.post("/api/message", {"text": "try a sleep job"})
        st = self.wait_for(lambda s: s["proposals"])
        pid = st["proposals"][0]["id"]
        self.post("/api/proposal", {"id": pid, "decision": "approve"})
        self.wait_for(lambda s: s["status"]["running"])
        self.post("/api/stop", {})
        st = self.wait_for(lambda s: any(e["type"] == "result" and e.get("pid") == pid for e in s["timeline"]), 30)
        res = next(e for e in st["timeline"] if e["type"] == "result" and e.get("pid") == pid)
        self.assertFalse(res["ok"])
        self.assertIn("KeyboardInterrupt", res["error"] or res["text"])
        time.sleep(1.5)
        st = self.get("/api/state")
        after = st["timeline"][[e.get("id") for e in st["timeline"]].index(next(e["id"] for e in st["timeline"] if e["type"] == "result" and e.get("pid") == pid)):]
        self.assertFalse(any(e["type"] == "spark" and e.get("kind") == "result" for e in after))
        self.post("/api/settings", {"autonomy": True})

    def test_remember_and_change_mind_become_a_pr_request(self):
        self.post("/api/message", {"text": "How could you improve how you think?"})
        st = self.wait_for(lambda s: any(p.get("kind") == "mind" for p in s["proposals"]))
        self.assertTrue(any(e["type"] == "memory" and "keep improving" in e["text"] for e in st["timeline"]))
        mem = (self.root / "shared/memory/tass-memory.md").read_text()
        self.assertIn("keep improving how I think", mem)
        reply = next(e for e in st["timeline"] if e["type"] == "reply")
        self.assertNotIn("REMEMBER", reply["text"])                      # memory lines are not shown as text
        p = next(p for p in st["proposals"] if p.get("kind") == "mind")
        self.assertEqual((p["path"], p["repo"], p["branch"]), ("mind/who-i-am.md", "Arameair/TASS", "alpha"))
        self.assertIn("+- Before I act, I make a one-line plan.", p["diff"])
        self.assertEqual([l for l in p["diff"].splitlines() if l.startswith(("+", "-")) and not l.startswith(("+++", "---"))],
                         ["+- Before I act, I make a one-line plan."])          # one line added, nothing removed
        self.assertIn("- I am a teammate, not a boss.\n- Before I act", p["content"])
        self.assertEqual(st["status"]["identity"], {"name": "TASS", "repo": "Arameair/TASS", "branch": "alpha"})
        # Memory is in the next prompt.
        self.post("/api/proposal", {"id": p["id"], "decision": "approve"})
        req = json.loads((self.root / "outbox" / f"{p['id']}.json").read_text())
        self.assertEqual((req["repo"], req["branch"], req["path"]), ("Arameair/TASS", "alpha", "mind/who-i-am.md"))
        self.assertIn("one-line plan", req["content"])
        # The Liaison answers with the PR link; the workspace reports it to Dustin and to TASS.
        (self.root / "outbox" / f"{p['id']}.result.json").write_text(
            json.dumps({"id": p["id"], "ok": True, "pr_url": "https://github.com/Arameair/TASS/pull/99"}))
        st = self.wait_for(lambda s: any(e["type"] == "mind_pr" for e in s["timeline"]), 15)
        ev = next(e for e in st["timeline"] if e["type"] == "mind_pr")
        self.assertEqual((ev["ok"], ev["url"]), (True, "https://github.com/Arameair/TASS/pull/99"))
        events = [r["event"] for r in self.get("/api/audit")]
        for e in ("remembered", "proposed", "approved", "pr_opened"):
            self.assertIn(e, events)
        self.post("/api/message", {"text": "What do you remember?"})
        st = self.wait_for(lambda s: any(e["type"] == "reply" and "From my memory" in e["text"] for e in s["timeline"]))
        self.assertIn("keep improving", [e for e in st["timeline"] if e["type"] == "reply"][-1]["text"])

    def test_whole_file_rewrite_that_drops_most_of_it_is_refused(self):
        self.post("/api/message", {"text": "Rewrite yourself from scratch"})
        st = self.wait_for(lambda s: any(e["type"] == "notice" and "remove most of" in e["text"] for e in s["timeline"]))
        self.assertFalse([p for p in st["proposals"] if p.get("kind") == "mind"])

    def test_mind_changes_can_be_turned_off(self):
        self.post("/api/settings", {"allow": {"mind changes": False}})
        try:
            self.post("/api/message", {"text": "Please improve yourself"})
            st = self.wait_for(lambda s: any(e["type"] == "notice" and "turned off mind changes" in e["text"]
                                             for e in s["timeline"]))
            self.assertFalse([p for p in st["proposals"] if p.get("kind") == "mind"])
        finally:
            self.post("/api/settings", {"allow": {"mind changes": True}})

    def test_full_auto_runs_without_clicks_and_pauses_at_the_limit(self):
        try:
            st = self.post("/api/settings", {"full_auto": True})
            self.assertTrue(st["full_auto"])
            # Hello -> OPEN book -> RUN cell 1: approved and run with no click, then TASS reacts.
            self.post("/api/message", {"text": "Hi TASS"})
            st = self.wait_for(lambda s: any(e["type"] == "reply" and "Next we could" in e["text"] for e in s["timeline"]))
            self.assertTrue(any(e["type"] == "result" and e["ok"] for e in st["timeline"]))
            self.assertIn("auto-approved", [r["event"] for r in self.get("/api/audit")])
            # A chain of runs with no message from Dustin stops at the limit (2) and waits.
            self.wait_for(lambda s: not s["status"]["busy"] and not s["status"]["running"])
            self.post("/api/message", {"text": "keep going, chain runs"})
            st = self.wait_for(lambda s: any(e["type"] == "notice" and "Full auto paused" in e["text"]
                                             for e in s["timeline"]), 60)
            self.assertEqual(sum(1 for e in st["timeline"] if e["type"] == "result" and "chained run" in e["text"]), 2)
            self.assertEqual(len(st["proposals"]), 1)                       # the third one waits for Dustin
            self.assertIn("auto-paused", [r["event"] for r in self.get("/api/audit")])
        finally:
            self.post("/api/settings", {"full_auto": False})
        # Off again: proposals wait for a click.
        self.wait_for(lambda s: not s["status"]["busy"])
        self.post("/api/reset", {})
        self.post("/api/message", {"text": "Try some of your own code please"})
        st = self.wait_for(lambda s: s["proposals"])
        time.sleep(1.5)
        self.assertEqual(self.get("/api/state")["proposals"][0]["status"], "pending")

    def test_atrium_chat_only_full_auto_and_autonomy(self):
        # The fixture deliberately emits OPEN, RUN, CHANGE MIND and REMEMBER;
        # all of them must be ignored on Atrium's chat-only request path.
        initial_audit = len(self.get("/api/audit"))
        memory = self.root / "shared/memory/tass-memory.md"
        initial_memory = memory.read_bytes() if memory.exists() else None
        outbox = self.root / "outbox"
        initial_outbox = set(outbox.iterdir()) if outbox.exists() else set()
        self.post("/api/settings", {"full_auto": True, "autonomy": True})
        request_ids = []
        try:
            for prompt in ("Hi TASS", "Try some of your own code please",
                           "How could you improve how you think?"):
                accepted = self.post("/api/message", {"text": prompt, "chat_only": True})
                self.assertTrue(accepted["ok"])
                rid = accepted["request_id"]
                request_ids.append(rid)
                self.assertRegex(rid, r"^[0-9a-f]{16}$")
                st = self.wait_for(lambda state: any(
                    e["type"] == "reply" and e.get("request_id") == rid
                    for e in state["timeline"]))
                events = [e for e in st["timeline"] if e.get("request_id") == rid]
                self.assertEqual([e["type"] for e in events if e["type"] in ("user", "reply")],
                                 ["user", "reply"])
                self.assertFalse(st["proposals"])
                self.assertTrue(st["status"]["full_auto"])
                self.assertTrue(st["status"]["autonomy"])
                self.wait_for(lambda state: not state["status"]["busy"])
            new_audit = self.get("/api/audit")[initial_audit:]
            self.assertFalse(any(r["event"] in ("proposed", "auto-approved", "remembered")
                                 for r in new_audit))
            for rid in request_ids:
                records = [r for r in new_audit if r.get("id") == rid]
                self.assertEqual([r["event"] for r in records],
                                 ["chat_accepted", "chat_replied"])
                self.assertTrue(all(r.get("source") is None for r in records))
            self.assertEqual(memory.read_bytes() if memory.exists() else None, initial_memory)
            self.assertEqual(set(outbox.iterdir()) if outbox.exists() else set(), initial_outbox)
        finally:
            self.post("/api/settings", {"full_auto": False})

    def test_atrium_busy_model_rejects_without_queuing(self):
        self.post("/api/settings", {"full_auto": True, "autonomy": True})
        try:
            first = self.post("/api/message", {"text": "Hi TASS", "chat_only": True})
            with self.assertRaises(urllib.error.HTTPError) as rejected:
                self.post("/api/message", {"text": "Second message", "chat_only": True})
            self.assertEqual(rejected.exception.code, 409)
            self.assertIn("not accepted", rejected.exception.read().decode())
            st = self.wait_for(lambda state: any(
                e["type"] == "reply" and e.get("request_id") == first["request_id"]
                for e in state["timeline"]))
            self.assertFalse(any(e["type"] == "user" and e.get("text") == "Second message"
                                 for e in st["timeline"]))
        finally:
            self.wait_for(lambda state: not state["status"]["busy"])
            self.post("/api/settings", {"full_auto": False})

    def test_full_auto_is_off_after_restart_by_design(self):
        self.assertFalse(self.get("/api/status")["full_auto"])

    def _inbox(self, name, text, issue=7):
        d = self.root / "inbox"
        d.mkdir(exist_ok=True)
        (d / f"Arameair__TASS__{issue}__{name}.json").write_text(json.dumps(
            {"repo": "Arameair/TASS", "issue": issue, "comment_id": name, "text": text,
             "url": f"https://github.com/Arameair/TASS/issues/{issue}#{name}"}))

    def _comments(self, issue=7):
        out = []
        for f in sorted((self.root / "outbox").glob("*.json"), key=lambda x: x.stat().st_mtime):
            if f.name.startswith(".") or f.name.endswith(".result.json"):
                continue
            r = json.loads(f.read_text())
            if r.get("type") == "comment" and r["issue"] == issue:
                out.append(r["body"])
        return out

    def test_direct_tass_from_github_and_approve_there(self):
        self._inbox("0", "Try some of your own code please")
        st = self.wait_for(lambda s: s["proposals"], 30)
        user = next(e for e in st["timeline"] if e["type"] == "user")
        self.assertEqual(user["issue"], "Arameair/TASS#7")
        self.wait_for(lambda s: len(self._comments()) >= 1, 15)
        first = self._comments()[0]
        self.assertIn("**TASS:**", first)
        self.assertIn("answer = sum(range(10))", first)          # the exact code is shown on GitHub
        self.assertIn("/approve", first)
        self._inbox("1001", "/approve")
        self.wait_for(lambda s: len(self._comments()) >= 3, 40)
        bodies = self._comments()
        self.assertIn("Approved `RUN own code` from GitHub", bodies[1])
        self.assertIn("**Result of `RUN own code`:** ✓ ok", bodies[2])
        self.assertIn("answer = 45", bodies[2])
        self.assertIn("**TASS:**", bodies[2])                       # his reaction to the result
        audit = self.get("/api/audit")
        self.assertTrue(any(r["event"] == "approved" and r.get("via") == "github" for r in audit))
        self.assertIn("github_message", [r["event"] for r in audit])
        self._inbox("1002", "/approve")
        self.wait_for(lambda s: len(self._comments()) >= 4, 15)
        self.assertIn("Nothing from this issue is waiting", self._comments()[3])

    def raw(self, path):
        try:
            with urllib.request.urlopen(f"http://127.0.0.1:{self.port}{path}", timeout=10) as r:
                return r.status, dict(r.headers), r.read()
        except urllib.error.HTTPError as e:
            return e.code, dict(e.headers), e.read()

    def test_shared_files_are_viewable_and_sandboxed(self):
        tools = self.root / "shared/tools"
        tools.mkdir(parents=True, exist_ok=True)
        (tools / "index.html").write_text("<h1>Notes</h1><script>fetch('/api/proposal')</script>")
        (self.root / "shared/.secret").write_text("nope")
        code, headers, body = self.raw("/files/tools/index.html")
        self.assertEqual(code, 200)
        self.assertIn(b"<h1>Notes</h1>", body)
        self.assertIn("sandbox", headers["Content-Security-Policy"])   # can't act as this app
        self.assertNotIn("allow-same-origin", headers["Content-Security-Policy"])
        code, _, body = self.raw("/files/tools/")
        self.assertEqual(code, 200)
        self.assertIn(b"index.html", body)
        for bad in ("/files/../data/messages.json", "/files/%2e%2e/data/messages.json", "/files/.secret",
                    "/files/tools/../../mirror/manifest.json"):
            self.assertEqual(self.raw(bad)[0], 404, bad)
        # A page served from /files (opaque origin "null") can't post to the controls.
        with self.assertRaises(urllib.error.HTTPError) as e:
            self.post("/api/settings", {"full_auto": True}, origin="null")
        self.assertEqual(e.exception.code, 403)
        st = self.get("/api/status")
        self.assertIn("tools/index.html", st["listing"])
        self.assertTrue(st["files_url"].endswith("/files/"))

    def test_tass_is_told_the_real_link_form(self):
        self.post("/api/message", {"text": "can you give me a link I can copy and paste?"})
        st = self.wait_for(lambda s: any(e["type"] == "reply" and "LINKCHECK" in e["text"] for e in s["timeline"]))
        self.assertIn("LINKCHECK yes", next(e["text"] for e in st["timeline"] if e["type"] == "reply"))

    def test_foreign_origin_is_refused(self):
        with self.assertRaises(urllib.error.HTTPError) as e:
            self.post("/api/message", {"text": "hi"}, origin="https://evil.example")
        self.assertEqual(e.exception.code, 403)

    def test_mind_change_sparks_reflection(self):
        self.wait_for(lambda s: not s["status"]["busy"])
        mirror = self.root / "mirror"
        (mirror / "files/mind/who-i-am.md").write_text("# Who I am\n\n- I always suggest a small tool.\n")
        m = json.loads((mirror / "manifest.json").read_text())
        m.update(last_change_at=time.time(), last_changed=["mind/who-i-am.md"])
        (mirror / "manifest.json").write_text(json.dumps(m))
        try:
            st = self.wait_for(lambda s: any(e["type"] == "reply" and "small tool" in e["text"] for e in s["timeline"]), 30)
            self.assertTrue(any(e["type"] == "spark" and e.get("kind") == "repo" for e in st["timeline"]))
        finally:
            (mirror / "files/mind/who-i-am.md").write_text(MIND_WHO)


if __name__ == "__main__":
    unittest.main()
