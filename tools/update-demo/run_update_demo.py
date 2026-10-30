#!/usr/bin/env python3
"""DISPOSABLE update demo for owner testing — NOT the owner's workstation launcher.

Runs the EXPERIMENTAL Identity (identity.installation-upgrade/1, unreviewed)
and this Processor build in their OWN throwaway profile and ports, so the
owner can watch one real feature update through the Development Workstation:

    ~/p1c/update-demo/            disposable profiles + binaries (delete to reset)
    Identity 127.0.0.1:7799       Processor 127.0.0.1:7869

It never touches ~/p1c/p1-tass-fresh, the launcher's ports 7797/7867, or any
pin in run-p1.sh. GitHub sign-in is REAL (device flow approved by the owner on
github.com). Pairing codes are shown only in THIS interactive terminal and are
never written to disk; Processor's private link stays in memory and the
browser only ever gets a single-use ticket.

Commands while running:  o = open the workstation in Edge InPrivate (fresh ticket)
                         p = show the newest Identity pairing code (this terminal only)
                         q = stop both programs (Ctrl+C works too)
"""
import json, os, re, signal, socket, subprocess, sys, threading, time, urllib.request
from pathlib import Path

HOME = Path.home()
TP_ROOT = Path(__file__).resolve().parents[2]
ID_SRC = Path(os.environ.get("UPDATE_DEMO_IDENTITY_SRC", str(HOME / "p1c/identity-upgrade")))
WORK = HOME / "p1c" / "update-demo"
ID_PORT, TP_PORT = 7799, 7869
ID_BRANCH = "experiment/installation-upgrade-claude-20261030"
ID_SHA = "4047787"  # prefix of the reviewed-for-demo experimental commit
CANDIDATE = TP_ROOT / "processor/testfeatures-updates/deterministic-worker-0.2.0"
LINK = re.compile(r"^\s*open\s*:\s*(http://127\.0\.0\.1:%d/#code=([0-9a-f]{48}))" % TP_PORT)
CODE = re.compile(r"  \|          ([A-HJKMNP-Z2-9]{4}-[A-HJKMNP-Z2-9]{4})")
state = {"code": None, "pairing": None}
procs = []


def die(msg):
    print("UPDATE DEMO: " + msg, file=sys.stderr)
    sys.exit(1)


def git(repo, *a):
    p = subprocess.run(["git", "-C", str(repo), *a], capture_output=True, text=True)
    return p.stdout.strip() if p.returncode == 0 else ""


def digest(d):
    import hashlib
    m = json.loads((d / "feature.json").read_text())
    ms = hashlib.sha256((d / "feature.json").read_bytes()).hexdigest()
    b = "thought.feature-package/1\nmanifest %s\n" % ms + "".join("%s  %s\n" % (f["sha256"], f["path"]) for f in sorted(m["files"], key=lambda f: f["path"]))
    return hashlib.sha256(b.encode()).hexdigest()


def free(port):
    with socket.socket() as s:
        return s.connect_ex(("127.0.0.1", port)) != 0


def reader(stream, kind):
    for raw in iter(stream.readline, ""):
        line = raw.rstrip("\r\n")
        if kind == "processor":
            m = LINK.match(line)
            if m:
                state["code"] = m.group(2)
        else:
            m = CODE.search(line)
            if m:
                state["pairing"] = m.group(1)
                if sys.stdout.isatty():
                    print("\n  [Identity asked for pairing — press p to see the code here]\n", flush=True)
    stream.close()


def spawn(kind, cmd, env):
    p = subprocess.Popen(cmd, env=env, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, stdin=subprocess.DEVNULL, text=True, bufsize=1, start_new_session=True)
    procs.append(p)
    threading.Thread(target=reader, args=(p.stdout, kind), daemon=True).start()
    return p


def open_workstation():
    code = state["code"]
    if not code:
        print("Processor isn't ready yet.")
        return
    base = "http://127.0.0.1:%d" % TP_PORT
    op = urllib.request.build_opener(urllib.request.ProxyHandler({}))
    req = urllib.request.Request(base + "/api/launch-ticket", json.dumps({"code": code}).encode(),
                                 headers={"Content-Type": "application/json", "Origin": base}, method="POST")
    ticket = json.loads(op.open(req, timeout=5).read()).get("ticket", "")
    url = base + "/workstation#code=" + ticket
    ps = ("$u=[Console]::In.ReadToEnd(); $edge=Join-Path ${env:ProgramFiles(x86)} 'Microsoft\\Edge\\Application\\msedge.exe'; "
          "if(-not (Test-Path $edge)){$edge=Join-Path $env:ProgramFiles 'Microsoft\\Edge\\Application\\msedge.exe'}; "
          "if(-not (Test-Path $edge)){exit 3}; Start-Process -FilePath $edge -ArgumentList @('--inprivate',$u)")
    try:
        r = subprocess.run(["powershell.exe", "-NoProfile", "-NonInteractive", "-Command", ps], input=url, text=True, capture_output=True, timeout=15)
        print("Opened Edge InPrivate on the Development Workstation (single-use ticket)." if r.returncode == 0 else "Edge could not be opened (exit %d)." % r.returncode)
    except (OSError, subprocess.TimeoutExpired):
        print("powershell.exe not available from this shell; this demo needs Windows + WSL to open Edge.")


def stop_all():
    for p in procs:
        if p.poll() is None:
            try:
                os.killpg(p.pid, signal.SIGTERM)
                p.wait(10)
            except (OSError, subprocess.TimeoutExpired):
                try:
                    os.killpg(p.pid, signal.SIGKILL)
                except OSError:
                    pass
    for prof, name in (("identity-profile", "open-link.txt"), ("processor-profile", "operator-link.txt")):
        try:
            (WORK / prof / "runtime" / name).unlink()
        except FileNotFoundError:
            pass
    print("Update demo stopped. Its throwaway profile stays in %s (delete that folder to reset)." % WORK)


def main():
    if not sys.stdin.isatty():
        die("run this in your own interactive WSL terminal.")
    if git(ID_SRC, "branch", "--show-current") != ID_BRANCH or not git(ID_SRC, "rev-parse", "HEAD").startswith(ID_SHA):
        die("expected %s on %s at %s… (clone it: git clone --branch %s https://github.com/Arameair/identity.git %s)" % (ID_SRC, ID_BRANCH, ID_SHA, ID_BRANCH, ID_SRC))
    if git(ID_SRC, "status", "--porcelain") or git(TP_ROOT, "status", "--porcelain"):
        die("a checkout has local changes; nothing was started.")
    if not (free(ID_PORT) and free(TP_PORT)):
        die("ports %d/%d are busy; nothing was stopped." % (ID_PORT, TP_PORT))
    pin = digest(CANDIDATE)
    for d in ("bin", "identity-profile", "processor-profile", "locators"):
        (WORK / d).mkdir(parents=True, exist_ok=True, mode=0o700)
    os.chmod(WORK, 0o700)
    print("Building the experimental Identity and this Processor (Go)…", flush=True)
    for src, out, br in ((ID_SRC, "identity-upgrade", ID_BRANCH), (TP_ROOT / "processor", "processor-demo", git(TP_ROOT, "branch", "--show-current"))):
        if subprocess.run(["go", "build", "-ldflags", "-X main.buildBranch=" + br, "-o", str(WORK / "bin" / out), "."], cwd=src).returncode != 0:
            die("Go build failed for " + str(src))
    keep = ("PATH", "HOME", "LANG", "LC_ALL", "TMPDIR", "USER", "WSL_DISTRO_NAME", "WSL_INTEROP")
    env = {k: os.environ[k] for k in keep if k in os.environ}
    env.update(IDENTITY_LOCATOR_DIR=str(WORK / "locators"), TP_IDENTITY_DEFAULT_URL="http://127.0.0.1:%d" % ID_PORT)
    spawn("identity", [str(WORK / "bin/identity-upgrade"), "-port", str(ID_PORT), "-data", str(WORK / "identity-profile"),
                       "-trust-fixture-artifact", pin], env)
    time.sleep(0.8)
    spawn("processor", [str(WORK / "bin/processor-demo"), "-port", str(TP_PORT), "-data", str(WORK / "processor-profile"),
                        "-identity-default-url", "http://127.0.0.1:%d" % ID_PORT, "-no-browser", "-no-link-file",
                        "-trust-feature-dir", str(TP_ROOT / "processor/testfeatures"), "-trust-feature-dir", str(CANDIDATE),
                        "-trust-feature-artifact", pin, "-workstation-repo", "processor=" + str(TP_ROOT),
                        "-workstation-repo", "identity=" + str(ID_SRC)], env)
    for _ in range(100):
        if state["code"] or any(p.poll() is not None for p in procs):
            break
        time.sleep(0.1)
    if not state["code"]:
        stop_all()
        die("Processor did not start.")
    for prof, name in (("identity-profile", "open-link.txt"),):
        try:
            (WORK / prof / "runtime" / name).unlink()  # Identity writes it but never reads it
        except FileNotFoundError:
            pass
    print("\nUPDATE DEMO RUNNING — experimental Identity %s… on %d, Processor on %d. Reviewed candidate: worker 0.2.0 (%s…)." % (ID_SHA, ID_PORT, TP_PORT, pin[:12]))
    open_workstation()
    print("Keys: o = open workstation again, p = show pairing code, q = quit")
    signal.signal(signal.SIGTERM, lambda *a: (stop_all(), sys.exit(0)))
    try:
        while True:
            k = sys.stdin.readline().strip().lower()
            if k == "o":
                open_workstation()
            elif k == "p":
                print("Pairing code (type it into Processor; never share it): " + (state["pairing"] or "none yet — press Connect in Processor first"))
            elif k in ("q", "quit", "exit"):
                break
            if any(p.poll() is not None for p in procs):
                print("A demo program exited.")
                break
    except KeyboardInterrupt:
        pass
    stop_all()


if __name__ == "__main__":
    main()
