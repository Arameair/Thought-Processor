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


def make_mirror(root: Path):
    files = root / "files"
    (files / "mind").mkdir(parents=True)
    (files / "library").mkdir(parents=True)
    (files / "mind/who-i-am.md").write_text("# Who I am\n\n- I am curious.\n")
    (files / "library/01-look-around.md").write_text(
        "# Look around\n\nSee what's here.\n\n```python\nprint('cell one')\n```\n\n```python\nx = 41 + 1\nprint(x)\n```\n")
    (files / "library/README.md").write_text("# Library\n")
    (root / "manifest.json").write_text(json.dumps({"repo": "Arameair/Developmnt", "synced_at": time.time(),
                                                    "last_change_at": 1.0, "last_changed": [], "files": {}}))


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
        env = dict(os.environ, TASS_MODEL_URL=f"http://127.0.0.1:{cls.model_port}", TASS_MIRROR=str(root / "mirror"),
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
        st = self.wait_for(lambda s: any(e["type"] == "reply" and "small tool" in e["text"] for e in s["timeline"]), 30)
        self.assertTrue(any(e["type"] == "spark" and e.get("kind") == "repo" for e in st["timeline"]))


if __name__ == "__main__":
    unittest.main()
