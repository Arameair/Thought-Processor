#!/usr/bin/env python3
"""WSL-only prototype supervisor for one pinned P1 development profile.

Private Processor URL and Identity pairing challenges stay in RAM. No shell
script or private-output logfile is used for startup. No Identity decisions
are made by this tool. Commands: start, stop, open, status, pairing, versions.
"""
import fcntl
import json
import os
from pathlib import Path
import re
import signal
import socket
import stat
import subprocess
import sys
import threading
import time
import urllib.error
import urllib.parse
import urllib.request

ROOT = Path(__file__).resolve().parents[2]
WORK = Path(os.environ.get('P1_LAUNCH_WORK', str(Path.home() / 'p1c/p1-tass-fresh'))).expanduser()
IDENTITY = Path(os.environ.get('P1_IDENTITY_SRC', str(ROOT.parent / 'identity'))).expanduser()
TASS = Path(os.environ.get('P1_TASS_SRC', str(Path.home() / 'p1c/tass-companion'))).expanduser()
THOUGHT = Path(os.environ.get('P1_THOUGHT_SRC', str(ROOT.parent / 'thought'))).expanduser()
RUN = ROOT / 'run-p1.sh'
RUNTIME = WORK / 'launcher-control'
SOCK = RUNTIME / 'control.sock'
LOCK = RUNTIME / 'owner.lock'
ID_PORT, TP_PORT = 7797, 7867
LAUNCH_BRANCH = 'experiment/p0-launcher-fresh-start-20261030'
IDENTITY_BRANCH = 'experiment/p1-cleanup-claude-20261026'
PREFLIGHT_TIMEOUT = 60
# Replaceable without an explicit Stop: these daemons own no running service.
REPLACEABLE = ('FAILED', 'STOPPED')
PRIVATE_LINK = re.compile(r'^\s*open\s*:\s*(http://127\.0\.0\.1:7867/\S+)', re.I)
PAIRING = re.compile(r'  \|          ([A-HJKMNP-Z2-9]{4}-[A-HJKMNP-Z2-9]{4})')
REV = re.compile(r'^[0-9a-f]{40}$')
MAX_EVENT = 4096


def safe_dir(path):
    if path.is_symlink() or (path.exists() and not path.is_dir()):
        raise RuntimeError('REFUSED_UNSAFE_WORK_DIRECTORY')
    path.mkdir(mode=0o700, parents=True, exist_ok=True)
    os.chmod(path, 0o700)


def git_rev(repo):
    out = subprocess.run(['git', '-C', str(repo), 'rev-parse', 'HEAD'], capture_output=True, text=True, timeout=5)
    return out.stdout.strip() if out.returncode == 0 and REV.fullmatch(out.stdout.strip()) else 'UNAVAILABLE'


def pinned():
    source = RUN.read_text()
    names = ('TASS_ARTIFACT', 'DEV_ARTIFACT', 'THOUGHT_ARTIFACT', 'IDENTITY_SHA')
    values = {}
    for name in names:
        m = re.search(r'^' + name + r'=([a-f0-9]{40,64})$', source, re.M)
        if not m:
            raise RuntimeError('UNVERIFIED_PIN_' + name)
        values[name] = m.group(1)
    return values


def versions(remote=False):
    refs = [('processor', ROOT, LAUNCH_BRANCH),
            ('identity', IDENTITY, IDENTITY_BRANCH),
            ('tass', TASS, 'experiment/tass-companion-feature-claude-20261029')]
    result = {'update_apply': 'DISABLED_REQUIRES_OWNER_AND_IDENTITY_REVIEW', 'revisions': {}, 'pins': pinned()}
    for name, path, branch in refs:
        current = git_rev(path)
        value = {'current_sha': current, 'reviewed_release': False, 'branch': branch}
        if remote:
            try:
                p = subprocess.run(['git', '-C', str(path), 'ls-remote', '--heads', 'origin', branch],
                                   capture_output=True, text=True, timeout=8, env={**os.environ, 'GIT_TERMINAL_PROMPT': '0'})
                match = p.stdout.split('\t')[0] if p.returncode == 0 else ''
                if not REV.fullmatch(match):
                    value['check'] = 'UNAVAILABLE'
                else:
                    value.update(remote_sha=match, check='NO_CHANGE' if match == current else 'UNVERIFIED_REMOTE_CHANGE')
            except (OSError, subprocess.TimeoutExpired):
                value['check'] = 'UNAVAILABLE'
        result['revisions'][name] = value
    return result


def listener(port):
    with socket.socket() as s:
        return s.connect_ex(('127.0.0.1', port)) == 0


def preflight_reason(stderr):
    """Last `ERROR:` line of run-p1.sh preflight, safe to show the owner.

    Preflight runs before any service exists, so it never sees a session link
    or pairing code. It does print paths and Git origin URLs, so credentials
    embedded in a URL are masked and the home folder is shortened to ~.
    """
    if not isinstance(stderr, (bytes, str)):
        return ''
    text = stderr.decode('utf-8', 'replace') if isinstance(stderr, bytes) else stderr
    lines = [ln.strip() for ln in text.splitlines() if ln.strip().startswith('ERROR:')]
    if not lines:
        return ''
    reason = lines[-1][len('ERROR:'):].strip()
    reason = re.sub(r'://[^/\s@]+@', '://***@', reason)
    home = str(Path.home())
    if home and home != '/':
        reason = reason.replace(home, '~')
    reason = ''.join(ch if ch.isprintable() else '?' for ch in reason)
    return reason[:400]


def redact_status(state, children, link, error, github='NOT_CHECKED', detail=''):
    return {'launcher': state, 'identity': 'RUNNING' if children.get('identity') and children['identity'].poll() is None else 'STOPPED',
            'processor': 'RUNNING' if children.get('processor') and children['processor'].poll() is None else 'STOPPED',
            'identity_port': 'LISTENING' if listener(ID_PORT) else 'CLOSED',
            'processor_port': 'LISTENING' if listener(TP_PORT) else 'CLOSED',
            'authenticated_open_available': bool(link), 'github': github,
            'error': error, 'error_detail': detail}


def open_edge(secret, *, pairing=False):
    if not secret or not shutil_which('powershell.exe'):
        return 'UNAVAILABLE_WINDOWS_OPENER'
    # The secret enters PowerShell through stdin, never argv or disk. Edge may
    # expose its final navigation URL in its own process command line.
    code = ("Add-Type -AssemblyName System.Windows.Forms; "
            "$v=[Console]::In.ReadToEnd(); "
            "[System.Windows.Forms.MessageBox]::Show($v,'Private Identity pairing code') | Out-Null") if pairing else (
            "$u=[Console]::In.ReadToEnd(); "
            "if(-not $u.StartsWith('http://127.0.0.1:7867/')){exit 2}; "
            "$edge=Join-Path ${env:ProgramFiles(x86)} 'Microsoft\\Edge\\Application\\msedge.exe'; "
            "if(-not (Test-Path $edge)){$edge=Join-Path $env:ProgramFiles 'Microsoft\\Edge\\Application\\msedge.exe'}; "
            "if(-not (Test-Path $edge)){exit 3}; Start-Process -FilePath $edge -ArgumentList @('--inprivate',$u)")
    try:
        p = subprocess.run(['powershell.exe', '-NoProfile', '-NonInteractive', '-Command', code],
                           input=secret, text=True, capture_output=True, timeout=12)
        return 'OPEN_REQUESTED' if p.returncode == 0 else 'OPENER_FAILED'
    except (OSError, subprocess.TimeoutExpired):
        return 'OPENER_FAILED'


def shutil_which(program):
    from shutil import which
    return which(program) is not None


class Supervisor:
    def __init__(self):
        self.state = 'STARTING'
        self.error = ''
        self.error_detail = ''
        self.children = {}
        self.link = None
        self.pairing = None
        self.github = 'NOT_CHECKED'
        self.lock = threading.Lock()
        self.stop_requested = threading.Event()

    def on_output(self, which, stream):
        try:
            for raw in iter(stream.readline, ''):
                line = raw.strip('\r\n')[:MAX_EVENT]
                if which == 'processor':
                    m = PRIVATE_LINK.fullmatch(line)
                    if m:
                        with self.lock:
                            self.link = m.group(1)
                elif which == 'identity':
                    m = PAIRING.search(line)
                    if m:
                        with self.lock:
                            self.pairing = m.group(1)
                # Drop all other output; never persist or echo private data.
        finally:
            stream.close()

    def spawn(self, name, args, env):
        p = subprocess.Popen(args, env=env, cwd=str(ROOT), stdout=subprocess.PIPE, stderr=subprocess.STDOUT,
                             stdin=subprocess.DEVNULL, text=True, bufsize=1, start_new_session=True)
        self.children[name] = p
        threading.Thread(target=self.on_output, args=(name, p.stdout), daemon=True).start()

    def poll_github(self):
        """Read Identity-derived state via a real Processor session in RAM."""
        import http.cookiejar
        with self.lock:
            link = self.link
        if not link:
            return
        code = re.search(r'[#&]code=([0-9a-f]{48})', link)
        if not code:
            self.github = 'SESSION_LINK_UNRECOGNIZED'
            return
        jar = http.cookiejar.CookieJar()
        opener = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(jar),
                                             urllib.request.ProxyHandler({}))
        base = 'http://127.0.0.1:7867'
        req = urllib.request.Request(base + '/api/session',
                                     json.dumps({'code': code.group(1)}).encode(),
                                     headers={'Content-Type': 'application/json', 'Origin': base}, method='POST')
        try:
            opener.open(req, timeout=3).read(4096)
        except (OSError, ValueError, urllib.error.URLError):
            self.github = 'SESSION_UNAVAILABLE'
            return
        while not self.stop_requested.wait(5):
            try:
                with opener.open(base + '/api/state', timeout=3) as response:
                    result = json.loads(response.read(262144))
                if not result.get('verified'):
                    # Processor only reports GitHub once it is paired with
                    # and verified by Identity; say so instead of 'unknown'.
                    self.github = 'IDENTITY_NOT_PAIRED'
                    continue
                state = (result.get('github') or {}).get('state')
                valid = {'VERIFIED', 'WAITING', 'DENIED', 'EXPIRED', 'NOT_CONNECTED', 'SIGNED_OUT',
                         'AVAILABLE', 'ERROR', 'UNAVAILABLE'}
                self.github = state if state in valid else 'NOT_VERIFIED_OR_UNKNOWN'
            except (OSError, ValueError, urllib.error.URLError):
                self.github = 'STATE_UNAVAILABLE'

    def build_and_start(self):
        try:
            env = os.environ.copy()
            env.update(P1_LAUNCH_WORK=str(WORK), P1_TASS_SRC=str(TASS), P1_IDENTITY_SRC=str(IDENTITY),
                       P1_PREFLIGHT_ONLY='1', P1_IDENTITY_PORT=str(ID_PORT), P1_PROCESSOR_PORT=str(TP_PORT))
            # Only the PRE-FLIGHT section runs: never invoke legacy runtime
            # that writes private links/challenges to persistent logs.
            try:
                pre = subprocess.run(['bash', str(RUN)], env=env, capture_output=True, timeout=PREFLIGHT_TIMEOUT)
            except subprocess.TimeoutExpired:
                raise RuntimeError('PREFLIGHT_TIMEOUT')
            if pre.returncode != 0 or b'PREFLIGHT OK' not in pre.stdout:
                self.error_detail = preflight_reason(pre.stderr) or 'preflight gave no reason (exit %s)' % pre.returncode
                raise RuntimeError('PREFLIGHT_REFUSED')
            if not (TASS / 'feature/feature.json').is_file():
                raise RuntimeError('TASS_CHECKOUT_MISSING')
            safe_dir(WORK)
            for folder in ('bin', 'identity-profile', 'processor-profile', 'locators'):
                safe_dir(WORK / folder)
            for name, repo, sub, branch in [('identity', IDENTITY, '.', IDENTITY_BRANCH),
                                           ('processor', ROOT, 'processor', LAUNCH_BRANCH)]:
                dest = WORK / 'bin' / (name + '-p1')
                proc = subprocess.run(['go', 'build', '-ldflags', '-X main.buildBranch=' + branch,
                                      '-o', str(dest), '.'], cwd=str(repo / sub), capture_output=True, timeout=150)
                if proc.returncode != 0:
                    raise RuntimeError(name.upper() + '_BUILD_FAILED')
            pin = pinned()
            thought = THOUGHT / 'feature'
            id_flags = ['-trust-fixture-artifact', pin['DEV_ARTIFACT'], '-trust-fixture-artifact', pin['TASS_ARTIFACT']]
            tp_flags = ['-trust-feature-dir', str(ROOT / 'processor/developer-utility'),
                        '-trust-feature-artifact', pin['DEV_ARTIFACT'], '-trust-feature-dir', str(TASS / 'feature'),
                        '-trust-feature-artifact', pin['TASS_ARTIFACT']]
            if thought.is_dir() and os.environ.get('P1_THOUGHT', 'on') != 'off':
                id_flags += ['-trust-fixture-artifact', pin['THOUGHT_ARTIFACT']]
                tp_flags += ['-trust-feature-dir', str(thought), '-trust-feature-artifact', pin['THOUGHT_ARTIFACT']]
            keep = ('PATH', 'HOME', 'LANG', 'LC_ALL', 'TMPDIR', 'USER', 'WSL_DISTRO_NAME', 'WSL_INTEROP')
            runtime_env = {k: os.environ[k] for k in keep if k in os.environ}
            runtime_env.update(IDENTITY_LOCATOR_DIR=str(WORK / 'locators'),
                               TP_IDENTITY_DEFAULT_URL='http://127.0.0.1:7797')
            if listener(ID_PORT) or listener(TP_PORT):
                raise RuntimeError('PORTS_OCCUPIED_REFUSED')
            if self.stop_requested.is_set():
                return
            self.spawn('identity', [str(WORK / 'bin/identity-p1'), '-port', str(ID_PORT),
                                    '-data', str(WORK / 'identity-profile'), *id_flags], runtime_env)
            time.sleep(.5)
            if self.children['identity'].poll() is not None:
                raise RuntimeError('IDENTITY_EXITED')
            if self.stop_requested.is_set():
                return
            self.spawn('processor', [str(WORK / 'bin/processor-p1'), '-port', str(TP_PORT),
                                     '-data', str(WORK / 'processor-profile'),
                                     '-identity-default-url', 'http://127.0.0.1:7797',
                                     '-trust-feature-dir', str(ROOT / 'processor/testfeatures'), *tp_flags,
                                     '-no-browser'], runtime_env)
            for _ in range(120):
                if self.stop_requested.is_set():
                    return
                if any(p.poll() is not None for p in self.children.values()):
                    raise RuntimeError('SERVICE_EXITED_DURING_START')
                with self.lock:
                    ready = self.link is not None
                if ready and listener(ID_PORT) and listener(TP_PORT):
                    self.state = 'RUNNING'
                    threading.Thread(target=self.poll_github, daemon=True).start()
                    return
                time.sleep(.1)
            raise RuntimeError('READY_NOT_CONFIRMED')
        except (OSError, RuntimeError, subprocess.TimeoutExpired) as exc:
            self.error = str(exc) if isinstance(exc, RuntimeError) else type(exc).__name__.upper()
            self.cleanup_children()
            self.state = 'FAILED'

    def cleanup_children(self):
        for name in ('processor', 'identity'):
            proc = self.children.get(name)
            if proc and proc.poll() is None:
                # Only stop the actual Popen child we created, not a pidfile,
                # port owner, unrelated group or reused PID.
                try:
                    os.killpg(proc.pid, signal.SIGTERM)
                    proc.wait(timeout=8)
                except (OSError, subprocess.TimeoutExpired):
                    if proc.poll() is None:
                        try:
                            os.killpg(proc.pid, signal.SIGKILL)
                            proc.wait(timeout=3)
                        except (OSError, subprocess.TimeoutExpired):
                            pass
        self.link = None
        self.pairing = None
        return not listener(ID_PORT) and not listener(TP_PORT)

    def stop(self):
        self.stop_requested.set()
        clear = self.cleanup_children()
        self.state = 'STOPPED' if clear else 'STOP_INCOMPLETE_PORT_BUSY'

    def handle(self, cmd):
        with self.lock:
            if cmd == 'status':
                if self.state == 'RUNNING' and any(p.poll() is not None for p in self.children.values()):
                    self.state, self.error = 'FAILED', 'OWNED_SERVICE_EXITED'
                return redact_status(self.state, self.children, self.link, self.error, self.github, self.error_detail)
            if cmd == 'open':
                link = self.link
            elif cmd == 'pairing':
                link = self.pairing
                self.pairing = None
            else:
                link = None
        if cmd == 'open':
            return {'result': open_edge(link)}
        if cmd == 'pairing':
            return {'result': open_edge(link, pairing=True) if link else 'NO_ACTIVE_CHALLENGE'}
        if cmd == 'stop':
            self.stop()
            return redact_status(self.state, self.children, None, self.error, self.github, self.error_detail)
        return {'error': 'UNKNOWN_COMMAND'}


def connect(cmd):
    with socket.socket(socket.AF_UNIX) as sock:
        sock.settimeout(4)
        sock.connect(str(SOCK))
        sock.sendall((cmd + '\n').encode())
        return json.loads(sock.recv(16384))


def lock_free():
    try:
        with LOCK.open('a+') as lf:
            fcntl.flock(lf.fileno(), fcntl.LOCK_EX | fcntl.LOCK_NB)
            fcntl.flock(lf.fileno(), fcntl.LOCK_UN)
            return True
    except BlockingIOError:
        return False
    except OSError:
        return False


def retire_daemon(timeout=15):
    try:
        connect('stop')
    except (OSError, ValueError):
        pass
    deadline = time.monotonic() + timeout
    while time.monotonic() < deadline:
        if not SOCK.exists() and lock_free():
            return True
        time.sleep(.1)
    return False


def daemon():
    safe_dir(WORK)
    safe_dir(RUNTIME)
    with LOCK.open('a+') as lf:
        try:
            fcntl.flock(lf.fileno(), fcntl.LOCK_EX | fcntl.LOCK_NB)
        except BlockingIOError:
            return 4
        if SOCK.exists():
            try:
                connect('status')
                return 4
            except (OSError, ValueError):
                SOCK.unlink()  # stale socket owned in this private directory
        sup = Supervisor()
        with socket.socket(socket.AF_UNIX) as server:
            server.bind(str(SOCK))
            os.chmod(SOCK, stat.S_IRUSR | stat.S_IWUSR)
            server.listen(5)
            server.settimeout(.3)
            thread = threading.Thread(target=sup.build_and_start, daemon=True)
            thread.start()
            try:
                while not sup.stop_requested.is_set():
                    try:
                        conn, _ = server.accept()
                    except socket.timeout:
                        continue
                    with conn:
                        # Linux SO_PEERCRED ensures the same Unix user owns the client.
                        import struct
                        cred = conn.getsockopt(socket.SOL_SOCKET, socket.SO_PEERCRED, struct.calcsize('3i'))
                        pid, uid, gid = struct.unpack('3i', cred)
                        if uid != os.getuid():
                            continue
                        req = conn.recv(128).decode(errors='replace').strip()
                        response = sup.handle(req)
                        conn.sendall(json.dumps(response).encode())
            finally:
                sup.stop()
                SOCK.unlink(missing_ok=True)
    return 0


def main(argv):
    if len(argv) != 1 or argv[0] not in ('start', 'stop', 'status', 'open', 'pairing', 'versions', 'check', '_daemon'):
        print('Usage: secure_launch.py start|stop|status|open|pairing|versions|check')
        return 2
    cmd = argv[0]
    if cmd == '_daemon':
        return daemon()
    if cmd in ('versions', 'check'):
        print(json.dumps(versions(remote=(cmd == 'check')), sort_keys=True))
        return 0
    if cmd == 'start':
        try:
            existing = connect('status')
        except (OSError, ValueError):
            existing = None
        if existing is not None:
            if existing.get('launcher') not in REPLACEABLE:
                # RUNNING, STARTING or STOP_INCOMPLETE_PORT_BUSY: never start
                # a second copy over a daemon that may still own services.
                print(json.dumps(existing, sort_keys=True))
                return 0 if existing.get('launcher') == 'RUNNING' else 3
            # A FAILED/STOPPED daemon still holds the result (and environment)
            # of an earlier attempt. Retire it through its own Stop, which
            # only signals children it started, then start fresh so this
            # Start re-runs preflight with the caller's current settings.
            if not retire_daemon():
                print(json.dumps({'launcher': 'FAILED', 'error': 'PREVIOUS_LAUNCHER_DID_NOT_EXIT',
                                  'previous': existing}, sort_keys=True))
                return 3
        safe_dir(WORK)
        safe_dir(RUNTIME)
        subprocess.Popen([sys.executable, str(Path(__file__).resolve()), '_daemon'], cwd=str(ROOT),
                         stdin=subprocess.DEVNULL, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL,
                         start_new_session=True, close_fds=True)
        for _ in range(150):
            time.sleep(.2)
            try:
                response = connect('status')
            except (OSError, ValueError):
                continue
            if response.get('launcher') != 'STARTING':
                print(json.dumps(response, sort_keys=True))
                return 0 if response.get('launcher') == 'RUNNING' else 3
        print(json.dumps({'launcher': 'STARTING', 'status': 'NOT_YET_VERIFIED'}))
        return 3
    try:
        response = connect(cmd)
    except (OSError, ValueError):
        response = {'launcher': 'STOPPED_OR_UNREACHABLE', 'result': 'NO_OWNED_SESSION'}
    print(json.dumps(response, sort_keys=True))
    return 0 if response.get('launcher') not in ('STOP_INCOMPLETE_PORT_BUSY',) else 3


if __name__ == '__main__':
    sys.exit(main(sys.argv[1:]))
