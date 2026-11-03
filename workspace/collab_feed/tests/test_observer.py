import json
from pathlib import Path
import sys
import tempfile
import unittest

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
from observer import Publisher, clean_text, event_summary, process_owner_briefs


class PublisherTest(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)
        self.data = self.root / "data"
        self.data.mkdir()
        self.outbox = self.root / "outbox"
        self.state = self.root / "state"
        self.pub = Publisher(self.data, self.outbox, self.state, 19)

    def write(self, events):
        (self.data / "timeline.json").write_text(json.dumps(events))

    def queued(self):
        return [json.loads(f.read_text()) for f in self.outbox.glob("feed-*.json")]

    def test_queued_owner_brief_is_delivered_via_liaison_without_code(self):
        requests=self.root/'requests';requests.mkdir()
        source=requests/'brief-1793732700-aabbccddee.json'
        source.write_text(json.dumps({'id':'brief-1793732700-aabbccddee',
                                      'kind':'owner_task_brief',
                                      'task':'Please build a useful local page.',
                                      'code':'DANGEROUS IGNORE THIS'}))
        self.assertEqual(process_owner_briefs(requests,self.outbox,19),1)
        queued=json.loads((self.outbox/'brief-1793732700-aabbccddee.json').read_text())
        self.assertEqual(queued['repo'],'Arameair/TASS')
        self.assertEqual(queued['issue'],19)
        self.assertEqual(queued['type'],'comment')
        self.assertIn('Please build a useful local page.',queued['body'])
        self.assertNotIn('DANGEROUS',queued['body'])
        self.assertIn('no execution or test result is implied',queued['body'].lower())
        self.assertEqual(process_owner_briefs(requests,self.outbox,19),0)
        self.assertFalse(source.exists())

    def test_credential_like_owner_brief_is_not_forwarded(self):
        requests=self.root/'requests';requests.mkdir()
        source=requests/'brief-1793732700-ffeeaabbcc.json'
        source.write_text(json.dumps({'id':'brief-1793732700-ffeeaabbcc',
                                      'kind':'owner_task_brief',
                                      'task':'token=SUPER_SECRET do something'}))
        self.assertEqual(process_owner_briefs(requests,self.outbox,19),0)
        self.assertEqual(list(self.outbox.glob('*.json')),[])
        self.assertTrue((requests/(source.name+'.rejected')).exists())

    def test_initial_sync_does_not_export_history(self):
        self.write([{"id": 1, "type": "user", "text": "private past request"}])
        self.assertEqual(self.pub.pulse(), 0)
        self.assertEqual(self.queued(), [])
        self.write([{"id": 1, "type": "user", "text": "private past request"},
                    {"id": 2, "type": "result", "label": "RUN library/01-look-around.md#2",
                     "ok": True, "text": "sensitive output"}])
        self.assertEqual(self.pub.pulse(), 1)
        queued = self.queued()
        self.assertEqual(len(queued), 1)
        self.assertEqual(queued[0]["repo"], "Arameair/TASS")
        self.assertEqual(queued[0]["issue"], 19)
        self.assertIn("PASS", queued[0]["body"])
        self.assertNotIn("sensitive output", queued[0]["body"])
        self.assertNotIn("private past request", queued[0]["body"])
        self.assertEqual(self.pub.pulse(), 0)

    def test_thoughts_and_code_are_never_shared(self):
        self.write([{"id": 10, "type": "user", "text": "old"}])
        self.pub.pulse()
        self.write([{"id": 10, "type": "user", "text": "old"},
                    {"id": 11, "type": "thought", "text": "private chain"},
                    {"id": 12, "type": "thought_delta", "delta": "private inner"},
                    {"id": 13, "type": "proposal", "proposal": {
                        "label": "RUN a project", "code": "password=secrets",
                        "diff": "sensitive patch"}},
                    {"id": 14, "type": "reply", "text": "text not opted in"}])
        self.assertEqual(self.pub.pulse(), 2)
        body = self.queued()[0]["body"]
        for secret in ["private chain", "private inner", "password=secrets",
                       "sensitive patch", "text not opted in"]:
            self.assertNotIn(secret, body)

    def test_explicit_text_opt_in_still_redacts_tokens(self):
        p = Publisher(self.data, self.outbox, self.state, 19, share_text=True)
        self.write([{"id": 1, "type": "user", "text": "old"}])
        p.pulse()
        self.write([{"id": 1, "type": "user", "text": "old"},
                    {"id": 2, "type": "user",
                     "text": "Please help me, token=ABCD1234 and hello."}])
        self.assertEqual(p.pulse(), 1)
        body = self.queued()[0]["body"]
        self.assertIn("Please help me", body)
        self.assertNotIn("ABCD1234", body)

    def test_cursor_advances_only_after_writable_queue(self):
        self.write([{"id": 1, "type": "user", "text": "old"}])
        self.pub.pulse()
        self.write([{"id": 1, "type": "user"}, {"id": 2, "type": "result", "ok": False,
                     "label": "RUN library/01-look-around.md#3"}])
        self.outbox.write_text("not a directory")
        with self.assertRaises(OSError):
            self.pub.pulse()
        self.outbox.unlink()
        self.assertEqual(self.pub.pulse(), 1)
        self.assertEqual(len(self.queued()), 1)

    def test_unknown_types_do_not_appear(self):
        self.assertIsNone(event_summary({"id": 1, "type": "memory", "text": "private"}))
        self.assertIsNone(event_summary({"id": 1, "type": "thought", "text": "private"}))
        self.assertNotIn("authorization=abc", clean_text("authorization=abc"))


if __name__ == "__main__":
    unittest.main()
