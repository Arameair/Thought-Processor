import importlib.util
import json
from pathlib import Path
import unittest
from unittest.mock import patch, MagicMock

s = importlib.util.spec_from_file_location('launcher', Path(__file__).with_name('secure_launch.py'))
launcher = importlib.util.module_from_spec(s)
s.loader.exec_module(launcher)

class TestLauncher(unittest.TestCase):
    def test_real_status_does_not_infer_github_verified(self):
        d = launcher.redact_status('RUNNING', {}, 'http://127.0.0.1:7867/#code=PRIVATE', '', 'WAITING')
        self.assertEqual(d['github'], 'WAITING')
        self.assertNotIn('PRIVATE', json.dumps(d))
        self.assertNotIn('link', str(d).lower().replace('authenticated_open_available', ''))

    def test_no_link_or_challenge_in_status(self):
        o = launcher.Supervisor()
        o.link = 'http://127.0.0.1:7867/#code=secret'
        o.pairing = 'ABCD-EFGH'
        o.github = 'WAITING'
        with patch.object(launcher, 'listener', return_value=False):
            res = o.handle('status')
        self.assertTrue(res['authenticated_open_available'])
        self.assertNotIn('secret', json.dumps(res))
        self.assertNotIn('ABCD-EFGH', json.dumps(res))

    def test_refuse_unknown_command(self):
        self.assertEqual(launcher.Supervisor().handle('run-shell')['error'], 'UNKNOWN_COMMAND')

    def test_only_open_if_private_link_available(self):
        with patch.object(launcher, 'open_edge', return_value='UNAVAILABLE_WINDOWS_OPENER') as run:
            self.assertEqual(launcher.Supervisor().handle('open')['result'], 'UNAVAILABLE_WINDOWS_OPENER')
            run.assert_called_once_with(None)

    def test_pairing_single_use(self):
        o = launcher.Supervisor()
        o.pairing = 'ABCD-EFGH'
        with patch.object(launcher, 'open_edge', return_value='OPEN_REQUESTED') as opener:
            self.assertEqual(o.handle('pairing')['result'], 'OPEN_REQUESTED')
            opener.assert_called_once_with('ABCD-EFGH', pairing=True)
        self.assertEqual(o.handle('pairing')['result'], 'NO_ACTIVE_CHALLENGE')

    def test_stop_without_owned_process_never_kills_foreign(self):
        o = launcher.Supervisor()
        with patch.object(launcher, 'listener', return_value=True), patch.object(launcher.os, 'killpg') as kill:
            o.stop()
        kill.assert_not_called()
        self.assertEqual(o.state, 'STOP_INCOMPLETE_PORT_BUSY')

    def test_versions_refuses_to_apply(self):
        with patch.object(launcher, 'git_rev', return_value='UNAVAILABLE'), patch.object(launcher, 'pinned', return_value={'TASS_ARTIFACT': 'test'}):
            v = launcher.versions()
        self.assertTrue(v['update_apply'].startswith('DISABLED'))
        self.assertTrue(all(x['current_sha'] == 'UNAVAILABLE' for x in v['revisions'].values()))

    def test_remote_head_change_is_not_approved_update(self):
        fake = MagicMock(returncode=0, stdout='a'*40+'\trefs/heads/test\n')
        with patch.object(launcher, 'git_rev', return_value='b'*40), patch.object(launcher, 'pinned', return_value={}), patch.object(launcher.subprocess, 'run', return_value=fake):
            v = launcher.versions(remote=True)
        self.assertEqual({x['check'] for x in v['revisions'].values()}, {'UNVERIFIED_REMOTE_CHANGE'})
        self.assertIn('DISABLED', v['update_apply'])

    def test_anonymous_port_is_not_session(self):
        with patch.object(launcher, 'listener', return_value=True):
            result = launcher.redact_status('RUNNING', {}, None, '', 'NOT_CHECKED')
        self.assertFalse(result['authenticated_open_available'])
        self.assertEqual(result['github'], 'NOT_CHECKED')

if __name__ == '__main__':
    unittest.main()