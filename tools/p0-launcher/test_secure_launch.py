import importlib.util
import json
from pathlib import Path
import unittest
from unittest.mock import patch, MagicMock

s = importlib.util.spec_from_file_location('launcher', Path(__file__).with_name('secure_launch.py'))
launcher = importlib.util.module_from_spec(s)
s.loader.exec_module(launcher)

class TestLauncher(unittest.TestCase):
    def test_launcher_and_preflight_require_same_experiment_branch(self):
        # Prevent branch rename regressions: preflight must continue to enforce
        # the exact approved launcher branch, not accept arbitrary checkouts.
        import re
        script = launcher.RUN.read_text()
        match = re.search(r"(?m)^LAUNCH_BRANCH=([^\s#]+)$", script)
        self.assertIsNotNone(match, "missing P1 branch preflight requirement")
        self.assertEqual(match.group(1), launcher.LAUNCH_BRANCH)
        self.assertEqual(launcher.LAUNCH_BRANCH, "experiment/brain-workstation-controls-20261031")

    def test_real_status_does_not_infer_github_verified(self):
        d = launcher.redact_status('RUNNING', {}, 'http://127.0.0.1:7868/#code=PRIVATE', '', 'WAITING')
        self.assertEqual(d['github'], 'WAITING')
        self.assertNotIn('PRIVATE', json.dumps(d))
        self.assertNotIn('link', str(d).lower().replace('authenticated_open_available', ''))

    def test_no_link_or_challenge_in_status(self):
        o = launcher.Supervisor()
        o.link = 'http://127.0.0.1:7868/#code=secret'
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
            self.assertEqual(launcher.Supervisor().handle('open')['result'], 'NOT_RUNNING')
            run.assert_not_called()

    def test_open_hands_browser_a_ticket_never_the_run_code(self):
        code = 'c' * 48
        o = launcher.Supervisor()
        o.link = 'http://127.0.0.1:7868/#code=' + code
        resp = MagicMock()
        resp.__enter__.return_value.read.return_value = json.dumps({'ticket': 't' * 0 + 'a' * 48}).encode()
        opener = MagicMock()
        opener.open.return_value = resp
        with patch.object(launcher.urllib.request, 'build_opener', return_value=opener), \
             patch.object(launcher, 'open_edge', return_value='OPEN_REQUESTED') as edge:
            out = o.handle('open')
        url = edge.call_args[0][0]
        self.assertEqual(url, 'http://127.0.0.1:7868/workstation#code=' + 'a' * 48)
        self.assertNotIn(code, url)
        self.assertNotIn(code, json.dumps(out))
        sent = json.loads(opener.open.call_args[0][0].data)
        self.assertEqual(sent, {'code': code})

    def test_open_refuses_when_ticket_unavailable(self):
        o = launcher.Supervisor()
        o.link = 'http://127.0.0.1:7868/#code=' + 'c' * 48
        opener = MagicMock()
        opener.open.side_effect = OSError('refused')
        with patch.object(launcher.urllib.request, 'build_opener', return_value=opener), \
             patch.object(launcher, 'open_edge') as edge:
            self.assertEqual(o.handle('open')['result'], 'NO_TICKET')
        edge.assert_not_called()

    def test_sweep_removes_only_link_files(self):
        import tempfile
        with tempfile.TemporaryDirectory() as d:
            work = Path(d)
            for prof, name in launcher.LINK_FILES:
                (work / prof / 'runtime').mkdir(parents=True)
                (work / prof / 'runtime' / name).write_text('private')
                (work / prof / 'runtime' / 'keep.lock').write_text('')
            with patch.object(launcher, 'WORK', work):
                self.assertEqual(launcher.sweep_link_files(), 2)
                self.assertEqual(launcher.sweep_link_files(), 0)
            for prof, name in launcher.LINK_FILES:
                self.assertFalse((work / prof / 'runtime' / name).exists())
                self.assertTrue((work / prof / 'runtime' / 'keep.lock').exists())

    def test_pairing_copy_button_uses_explicit_click_and_private_stdin(self):
        # This checks the PowerShell command contract, not a real Windows UI.
        secret = 'ABCD-EFGH'
        with patch.object(launcher, 'shutil_which', return_value=True), \
             patch.object(launcher.subprocess, 'run', return_value=MagicMock(returncode=0)) as run:
            self.assertEqual(launcher.open_edge(secret, pairing=True), 'OPEN_REQUESTED')
        argv = run.call_args.args[0]
        script = argv[-1]
        self.assertIn('-Sta', argv)
        self.assertIn("Copy code", script)
        self.assertIn("$copy.Add_Click(", script)
        self.assertIn("[System.Windows.Forms.Clipboard]::SetText($field.Text)", script)
        self.assertIn("$field.ReadOnly=$true", script)
        self.assertIn("GetNewClosure()", script)
        self.assertNotIn(secret, ' '.join(argv))
        self.assertEqual(run.call_args.kwargs['input'], secret)
        self.assertEqual(run.call_args.kwargs['timeout'], 120)

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

    def test_stop_closes_real_owned_test_listeners(self):
        import socket
        import subprocess
        import sys
        import time
        ports = []
        for _ in range(2):
            with socket.socket() as sock:
                sock.bind(('127.0.0.1', 0))
                ports.append(sock.getsockname()[1])
        supervisor = launcher.Supervisor()
        for name, port in zip(('identity', 'processor'), ports):
            child = subprocess.Popen([sys.executable, '-c',
                                      'import socket,time; s=socket.socket(); s.setsockopt(socket.SOL_SOCKET,socket.SO_REUSEADDR,1); s.bind(("127.0.0.1",int(__import__("sys").argv[1]))); s.listen(); time.sleep(30)',
                                      str(port)], stdin=subprocess.DEVNULL,
                                     stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL,
                                     start_new_session=True)
            supervisor.children[name] = child
        try:
            for _ in range(100):
                if all(launcher.listener(port) for port in ports):
                    break
                time.sleep(.02)
            self.assertTrue(all(launcher.listener(port) for port in ports))
            with patch.object(launcher, 'ID_PORT', ports[0]), patch.object(launcher, 'TP_PORT', ports[1]):
                supervisor.stop()
            self.assertEqual(supervisor.state, 'STOPPED')
            self.assertTrue(all(p.poll() is not None for p in supervisor.children.values()))
            self.assertTrue(all(not launcher.listener(port) for port in ports))
        finally:
            for p in supervisor.children.values():
                if p.poll() is None:
                    p.kill()
                    p.wait(timeout=5)

    def test_start_failure_stays_observable_without_spawning(self):
        o = launcher.Supervisor()
        failure = MagicMock(returncode=1, stdout=b'')
        with patch.object(launcher.subprocess, 'run', return_value=failure), patch.object(launcher, 'listener', return_value=False):
            o.build_and_start()
            status = o.handle('status')
        self.assertEqual(status['launcher'], 'FAILED')
        self.assertEqual(status['error'], 'PREFLIGHT_REFUSED')
        self.assertFalse(o.stop_requested.is_set())
        self.assertEqual(o.children, {})

    def test_versions_refuses_to_apply(self):
        with patch.object(launcher, 'git_rev', return_value='UNAVAILABLE'), patch.object(launcher, 'pinned', return_value={'TASS_ARTIFACT': 'test'}):
            v = launcher.versions()
        self.assertTrue(v['update_apply'].startswith('OWNER_PRESSES_UPDATE'))
        self.assertTrue(all(x['current_sha'] == 'UNAVAILABLE' for x in v['revisions'].values()))

    def test_remote_head_change_is_not_approved_update(self):
        fake = MagicMock(returncode=0, stdout='a'*40+'\trefs/heads/test\n')
        with patch.object(launcher, 'git_rev', return_value='b'*40), patch.object(launcher, 'pinned', return_value={}), patch.object(launcher.subprocess, 'run', return_value=fake):
            v = launcher.versions(remote=True)
        self.assertEqual({x['check'] for x in v['revisions'].values()}, {'UNVERIFIED_REMOTE_CHANGE'})
        self.assertIn('OWNER_PRESSES_UPDATE', v['update_apply'])  # a remote change is only reported; nothing applies it but the owner's Update

    def test_anonymous_port_is_not_session(self):
        with patch.object(launcher, 'listener', return_value=True):
            result = launcher.redact_status('RUNNING', {}, None, '', 'NOT_CHECKED')
        self.assertFalse(result['authenticated_open_available'])
        self.assertEqual(result['github'], 'NOT_CHECKED')

    def test_preflight_reason_is_shown_masked(self):
        err = (b'Identity P1 source: x\n'
               b'ERROR: TASS: /home/u/p1c/t is a checkout of a different repository '
               b'(origin: https://user:ghp_SECRET@github.com/x/y); expected Arameair/TASS.\n')
        with patch.object(launcher.Path, 'home', return_value=Path('/home/u')):
            reason = launcher.preflight_reason(err)
        self.assertTrue(reason.startswith('TASS: ~/p1c/t is a checkout'))
        self.assertNotIn('ghp_SECRET', reason)
        self.assertNotIn('user:', reason)
        self.assertEqual(launcher.preflight_reason(b'no error line'), '')
        self.assertEqual(launcher.preflight_reason(None), '')

    def test_start_failure_reports_preflight_reason(self):
        o = launcher.Supervisor()
        failure = MagicMock(returncode=1, stdout=b'', stderr=b'ERROR: Dirty working tree in /x. Commit first.\n')
        with patch.object(launcher.subprocess, 'run', return_value=failure), patch.object(launcher, 'listener', return_value=False):
            o.build_and_start()
            status = o.handle('status')
        self.assertEqual(status['error'], 'PREFLIGHT_REFUSED')
        self.assertIn('Dirty working tree', status['error_detail'])

    def test_start_replaces_failed_daemon_instead_of_echoing_it(self):
        replies = iter([{'launcher': 'FAILED', 'error': 'PREFLIGHT_REFUSED'}, {'launcher': 'RUNNING'}])
        with patch.object(launcher, 'connect', side_effect=lambda cmd: next(replies)), \
             patch.object(launcher, 'retire_daemon', return_value=True) as retire, \
             patch.object(launcher, 'safe_dir'), patch.object(launcher.time, 'sleep'), \
             patch.object(launcher.subprocess, 'Popen') as spawn, patch('builtins.print'):
            code = launcher.main(['start'])
        retire.assert_called_once()
        spawn.assert_called_once()
        self.assertEqual(code, 0)

    def test_start_never_replaces_running_or_busy_daemon(self):
        for state, expected in (('RUNNING', 0), ('STARTING', 3), ('STOP_INCOMPLETE_PORT_BUSY', 3)):
            with patch.object(launcher, 'connect', return_value={'launcher': state}), \
                 patch.object(launcher, 'retire_daemon') as retire, \
                 patch.object(launcher.subprocess, 'Popen') as spawn, patch('builtins.print'):
                self.assertEqual(launcher.main(['start']), expected)
            retire.assert_not_called()
            spawn.assert_not_called()

    def test_start_reports_when_old_daemon_will_not_exit(self):
        with patch.object(launcher, 'connect', return_value={'launcher': 'FAILED'}), \
             patch.object(launcher, 'retire_daemon', return_value=False), \
             patch.object(launcher.subprocess, 'Popen') as spawn, patch('builtins.print') as out:
            self.assertEqual(launcher.main(['start']), 3)
        spawn.assert_not_called()
        self.assertIn('PREVIOUS_LAUNCHER_DID_NOT_EXIT', out.call_args[0][0])

    def test_orphans_are_found_by_owned_binary_and_stopped(self):
        import shutil, subprocess, tempfile
        sleep = shutil.which('sleep')
        with tempfile.TemporaryDirectory() as d:
            work = Path(d)
            (work / 'bin').mkdir()
            own = work / 'bin' / 'identity-p1'
            shutil.copy(sleep, own)
            foreign = work / 'other-sleep'
            shutil.copy(sleep, foreign)
            mine = subprocess.Popen([str(own), '60'])
            theirs = subprocess.Popen([str(foreign), '60'])
            try:
                with patch.object(launcher, 'WORK', work):
                    found = launcher.owned_orphans()
                    self.assertIn(mine.pid, found)
                    self.assertNotIn(theirs.pid, found)
                    self.assertTrue(launcher.stop_orphans(found, grace=3))
                mine.wait(timeout=5)
                self.assertIsNone(theirs.poll())
            finally:
                for p in (mine, theirs):
                    if p.poll() is None:
                        p.kill()
                        p.wait(timeout=5)

    def test_start_refuses_over_unsupervised_services(self):
        with patch.object(launcher, 'connect', side_effect=OSError), patch.object(launcher, 'owned_orphans', return_value=[123]), \
             patch.object(launcher, 'retire_daemon', return_value=True), \
             patch.object(launcher.subprocess, 'Popen') as spawn, patch('builtins.print') as out:
            self.assertEqual(launcher.main(['start']), 3)
        spawn.assert_not_called()
        self.assertIn('PREVIOUS_SERVICES_STILL_RUNNING', out.call_args[0][0])

class TestUpdate(unittest.TestCase):
    """Update against real temporary git repositories (origin + checkout)."""

    def setUp(self):
        import subprocess, tempfile
        self.sp = subprocess
        self.tmp = Path(tempfile.mkdtemp())
        g = lambda *a, cwd=None: subprocess.run(['git', *a], cwd=cwd, check=True, capture_output=True, text=True).stdout.strip()
        self.g = g
        g('init', '--bare', '-q', '-b', launcher.LAUNCH_BRANCH, str(self.tmp / 'origin.git'))
        g('clone', '-q', str(self.tmp / 'origin.git'), str(self.tmp / 'co'))
        co = self.tmp / 'co'
        for k, v in (('user.email', 't@t'), ('user.name', 't')):
            g('config', k, v, cwd=co)
        g('checkout', '-q', '-b', launcher.LAUNCH_BRANCH, cwd=co)
        (co / 'f.txt').write_text('v1\n')
        g('add', '.', cwd=co); g('commit', '-q', '-m', 'v1', cwd=co); g('push', '-q', 'origin', launcher.LAUNCH_BRANCH, cwd=co)
        self.v1 = g('rev-parse', 'HEAD', cwd=co)
        # a second clone publishes v2 the way GitHub would
        g('clone', '-q', '-b', launcher.LAUNCH_BRANCH, str(self.tmp / 'origin.git'), str(self.tmp / 'pub'))
        pub = self.tmp / 'pub'
        for k, v in (('user.email', 't@t'), ('user.name', 't')):
            g('config', k, v, cwd=pub)
        (pub / 'f.txt').write_text('v2\n')
        g('commit', '-q', '-am', 'v2: new screen', cwd=pub); g('push', '-q', 'origin', launcher.LAUNCH_BRANCH, cwd=pub)
        self.v2 = g('rev-parse', 'HEAD', cwd=pub)
        self.co = co
        self.calls = []
        self.patches = [patch.object(launcher, 'ROOT', co), patch.object(launcher, 'RUNTIME', self.tmp / 'control'),
                        patch.object(launcher, 'WORK', self.tmp)]
        for p in self.patches:
            p.start()

    def tearDown(self):
        for p in self.patches:
            p.stop()
        import shutil
        shutil.rmtree(self.tmp, ignore_errors=True)

    def run_stub(self, starts=('RUNNING',)):
        seq = list(starts)
        def run(cmd):
            self.calls.append(cmd)
            if cmd == 'start':
                return {'launcher': seq.pop(0) if seq else 'RUNNING'}
            return {'launcher': 'STOPPED'}
        return run

    def head(self):
        return self.g('rev-parse', 'HEAD', cwd=self.co)

    def test_fast_forward_update_verifies_then_starts(self):
        r = launcher.update(run=self.run_stub(), preflight=lambda: (True, ''))
        self.assertEqual((r['result'], r['previous'], r['now_running']), ('UPDATED', self.v1, self.v2))
        self.assertEqual(self.head(), self.v2)
        self.assertEqual(self.calls, ['stop', 'start'])
        self.assertIn('v2: new screen', ' '.join(r['changes']))

    def test_up_to_date_changes_nothing(self):
        launcher.update(run=self.run_stub(), preflight=lambda: (True, ''))
        self.calls.clear()
        r = launcher.update(run=self.run_stub(), preflight=lambda: (True, ''))
        self.assertEqual(r['result'], 'UP_TO_DATE')
        self.assertEqual(self.calls, [])

    def test_failed_preflight_rolls_back_to_exact_previous_commit(self):
        r = launcher.update(run=self.run_stub(), preflight=lambda: (False, 'TASS package is not the pinned artifact'))
        self.assertEqual(r['result'], 'ROLLED_BACK')
        self.assertIn('pinned artifact', r['reason'])
        self.assertEqual(self.head(), self.v1)
        self.assertEqual(self.calls, ['stop', 'start'])

    def test_new_version_that_will_not_start_rolls_back(self):
        r = launcher.update(run=self.run_stub(starts=('FAILED', 'RUNNING')), preflight=lambda: (True, ''))
        self.assertEqual((r['result'], r['launcher']), ('ROLLED_BACK', 'RUNNING'))
        self.assertEqual(self.head(), self.v1)
        self.assertEqual(self.calls, ['stop', 'start', 'stop', 'start'])

    def test_local_changes_refuse_and_touch_nothing(self):
        (self.co / 'f.txt').write_text('my edit\n')
        r = launcher.update(run=self.run_stub(), preflight=lambda: (True, ''))
        self.assertEqual(r['result'], 'REFUSED')
        self.assertEqual((self.head(), (self.co / 'f.txt').read_text(), self.calls), (self.v1, 'my edit\n', []))

    def test_rewritten_history_is_refused(self):
        pub = self.tmp / 'pub'
        self.g('reset', '-q', '--hard', 'HEAD~1', cwd=pub)
        (pub / 'other.txt').write_text('x')
        self.g('add', '.', cwd=pub); self.g('commit', '-q', '-m', 'divergent', cwd=pub)
        self.g('push', '-q', '-f', 'origin', launcher.LAUNCH_BRANCH, cwd=pub)
        self.g('commit', '-q', '--allow-empty', '-m', 'local only', cwd=self.co)
        before = self.head()
        r = launcher.update(run=self.run_stub(), preflight=lambda: (True, ''))
        self.assertEqual(r['result'], 'REFUSED')
        self.assertIn('fast-forward', r['reason'])
        self.assertEqual((self.head(), self.calls), (before, []))

    def test_other_branch_is_refused(self):
        self.g('checkout', '-q', '-b', 'something-else', cwd=self.co)
        r = launcher.update(run=self.run_stub(), preflight=lambda: (True, ''))
        self.assertEqual((r['result'], self.calls), ('REFUSED', []))

    def test_history_is_private_and_holds_no_secrets(self):
        launcher.update(run=self.run_stub(), preflight=lambda: (True, ''))
        f = self.tmp / 'control' / launcher.UPDATE_LOG
        self.assertEqual(f.stat().st_mode & 0o777, 0o600)
        line = json.loads(f.read_text().splitlines()[-1])
        self.assertEqual((line['result'], line['previous'], line['now_running']), ('UPDATED', self.v1, self.v2))
        self.assertNotIn('code=', f.read_text())


if __name__ == '__main__':
    unittest.main()
