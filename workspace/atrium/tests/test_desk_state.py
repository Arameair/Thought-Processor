"""Tests for the daily Atrium operator task ledger and read-only UI routes."""
import json
import sys
import tempfile
import threading
import unittest
from pathlib import Path
from urllib.error import HTTPError
from urllib.request import Request, build_opener, ProxyHandler
from http.server import ThreadingHTTPServer

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
from desk_state import BoardStore, InvalidBoard, StaleBoard, starter_board, validate_board
from server import create_handler

OPENER = build_opener(ProxyHandler({}))


class BoardTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.file = Path(self.tmp.name) / "state" / "desk.json"
        self.store = BoardStore(self.file)

    def test_starter_is_unverified_and_not_memory(self):
        board = self.store.read()
        self.assertGreaterEqual(len(board["tasks"]), 5)
        self.assertFalse(self.file.exists())
        self.assertTrue(all(t["result"] == "unverified" for t in board["tasks"]))

    def test_save_survives_restart_and_is_owner_only_data(self):
        initial = self.store.read()
        initial["tasks"][0]["notes"] = "Observed model ready; actual run pending."
        saved = self.store.write(initial)
        self.assertEqual(saved["revision"], 1)
        self.assertEqual(BoardStore(self.file).read()["tasks"][0]["notes"],
                         "Observed model ready; actual run pending.")
        self.assertEqual(self.file.stat().st_mode & 0o777, 0o600)

    def test_pass_requires_evidence(self):
        board = self.store.read()
        board["tasks"][0]["result"] = "reported_pass"
        with self.assertRaises(InvalidBoard):
            self.store.write(board)
        self.assertFalse(self.file.exists())
        board["tasks"][0]["evidence"] = "Owner observed a harmless test; example SHA abc123."
        self.assertEqual(self.store.write(board)["tasks"][0]["result"], "reported_pass")

    def test_rejects_duplicate_ids_and_invalid_status(self):
        board = self.store.read()
        board["tasks"].append(dict(board["tasks"][0]))
        with self.assertRaises(InvalidBoard):
            validate_board(board)
        board["tasks"].pop()
        board["tasks"][0]["status"] = "auto_approve"
        with self.assertRaises(InvalidBoard):
            validate_board(board)
        self.assertFalse(self.file.exists())

    def test_conflict_does_not_overwrite_existing(self):
        first = self.store.read()
        other = self.store.read()
        first["tasks"][0]["title"] = "First tab"
        self.store.write(first)
        other["tasks"][0]["title"] = "Stale second tab"
        with self.assertRaises(StaleBoard):
            self.store.write(other)
        self.assertEqual(self.store.read()["tasks"][0]["title"], "First tab")

    def test_daily_edition_does_not_erase_previous_work(self):
        board = self.store.read()
        board["day"] = "2026-11-03"
        board["tasks"][0]["notes"] = "Protected previous-day work"
        self.store.write(board)
        loaded = BoardStore(self.file).read()
        self.assertEqual(loaded["day"], "2026-11-03")
        self.assertEqual(loaded["tasks"][0]["notes"], "Protected previous-day work")

    def test_corrupt_file_fails_closed(self):
        self.file.parent.mkdir(parents=True)
        self.file.write_text("unreadable json", encoding="utf-8")
        with self.assertRaises(ValueError):
            self.store.read()
        self.assertEqual(self.file.read_text(), "unreadable json")


class DeskHTTPTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        import os
        import socket
        cls.tmp = tempfile.TemporaryDirectory()
        cls.old = os.environ.get("TASS_ATRIUM_DESK_STATE")
        os.environ["TASS_ATRIUM_DESK_STATE"] = str(Path(cls.tmp.name) / "desk.json")
        with socket.socket() as sock:
            sock.bind(("127.0.0.1", 0))
            port = sock.getsockname()[1]
        cls.port = port
        cls.server = ThreadingHTTPServer(("127.0.0.1", port),
            create_handler(8790, "unit-test-csrf", port))
        cls.thread = threading.Thread(target=cls.server.serve_forever, daemon=True)
        cls.thread.start()

    @classmethod
    def tearDownClass(cls):
        import os
        cls.server.shutdown()
        cls.server.server_close()
        cls.thread.join(timeout=2)
        if cls.old is None:
            os.environ.pop("TASS_ATRIUM_DESK_STATE", None)
        else:
            os.environ["TASS_ATRIUM_DESK_STATE"] = cls.old
        cls.tmp.cleanup()

    def url(self, path):
        return "http://127.0.0.1:" + str(self.port) + path

    def post(self, board, csrf="unit-test-csrf", origin=None):
        headers = {"Content-Type": "application/json"}
        if csrf is not None:
            headers["X-Playground-CSRF"] = csrf
        if origin is not None:
            headers["Origin"] = origin
        req = Request(self.url("/desk/state"),
                      data=json.dumps(board).encode(), method="POST", headers=headers)
        try:
            with OPENER.open(req, timeout=3) as response:
                return response.status, json.load(response)
        except HTTPError as e:
            return e.code, json.loads(e.read())

    def test_daily_page_and_legacy_routes_available(self):
        for path, marker in [
            ("/", b"DAILY WORKSTATION"),
            ("/today", b"DAILY WORKSTATION"),
            ("/build", b"BUILD ROOM"),
            ("/explore", b"atrium"),
            ("/assets/today.js", b"pollLive"),
            ("/assets/today.css", b"reality-grid"),
        ]:
            with self.subTest(path=path):
                with OPENER.open(self.url(path), timeout=3) as response:
                    self.assertEqual(response.status, 200)
                    self.assertIn(marker.lower(), response.read().lower())

    def test_csrf_origin_validation_and_persistent_post(self):
        with OPENER.open(self.url("/desk/state"), timeout=3) as r:
            board = json.load(r)
        self.assertEqual(self.post(board, csrf=None)[0], 403)
        self.assertEqual(self.post(board, origin="http://evil.example")[0], 403)
        status, saved = self.post(board, origin="http://127.0.0.1:" + str(self.port))
        self.assertEqual(status, 200)
        self.assertEqual(saved["revision"], board["revision"] + 1)
        self.assertEqual(self.post(board)[0], 409)
        with OPENER.open(self.url("/desk/state"), timeout=3) as r:
            self.assertEqual(json.load(r)["revision"], saved["revision"])

    def test_invalid_pass_not_saved(self):
        with OPENER.open(self.url("/desk/state"), timeout=3) as r:
            board = json.load(r)
        board["tasks"][0]["result"] = "reported_pass"
        board["tasks"][0]["evidence"] = ""
        self.assertEqual(self.post(board)[0], 400)


if __name__ == "__main__":
    unittest.main()
