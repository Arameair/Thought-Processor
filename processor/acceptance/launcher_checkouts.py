"""run-p1.sh checkout regression test: ordinary clones AND Git worktrees.

Builds throwaway layouts from the given sources (local clones only; nothing
is fetched), points each clone's origin at the real GitHub URL, and runs the
COMMITTED launcher of this Processor checkout:

  * preflight (P1_PREFLIGHT_ONLY=1) must PASS for all-clones and
    all-worktrees layouts, and with Thought switched off;
  * preflight must FAIL, with a clear reason, for: dirty worktree, wrong
    branch, wrong repository, broken worktree link, a subfolder instead of a
    checkout, Thought content that differs from the pin, Thought not
    descended from the reviewed commit;
  * one full launch from the worktree layout must reach "ready" with Thought
    pinned, then stop cleanly with nothing left running.

  python3 -I acceptance/launcher_checkouts.py --identity-src DIR --thought-src DIR --work DIR

Your own checkouts are only read (cloned from); nothing in them is changed.
"""
import argparse, os, shutil, signal, socket, subprocess, sys, time
from pathlib import Path

ap = argparse.ArgumentParser()
ap.add_argument("--identity-src", required=True)
ap.add_argument("--thought-src", required=True)
ap.add_argument("--work", required=True)
args = ap.parse_args()
W = Path(args.work).resolve()
if W.exists():
    shutil.rmtree(W)
W.mkdir(parents=True, mode=0o700)
PROC_SRC = Path(__file__).resolve().parent.parent.parent  # repo root
results = []
ORIGINS = {"identity": "https://github.com/Arameair/identity.git", "Thought-Processor": "https://github.com/Arameair/Thought-Processor.git",
           "Thought": "https://github.com/Arameair/Thought.git"}
SRC = {"identity": args.identity_src, "Thought-Processor": str(PROC_SRC), "Thought": args.thought_src}


def rec(ok, name, detail=""):
    results.append(("PASS" if ok else "FAIL", name, detail))
    print(("PASS " if ok else "FAIL ") + name + ("  — " + detail if detail else ""), flush=True)
    return ok


def git(*a, cwd=None, check=True):
    return subprocess.run(["git", *a], cwd=cwd, check=check, capture_output=True, text=True,
                          env=dict(os.environ, GIT_CONFIG_NOSYSTEM="1", GIT_AUTHOR_NAME="t", GIT_AUTHOR_EMAIL="t@t", GIT_COMMITTER_NAME="t", GIT_COMMITTER_EMAIL="t@t"))


def branch_of(src):
    return git("branch", "--show-current", cwd=src).stdout.strip()


def free_port():
    with socket.socket() as s:
        s.bind(("127.0.0.1", 0))
        return s.getsockname()[1]


def clone_layout(parent):
    parent.mkdir(parents=True)
    for name, src in SRC.items():
        git("clone", "-q", "--no-hardlinks", src, str(parent / name))
        git("remote", "set-url", "origin", ORIGINS[name], cwd=parent / name)
    return parent


def worktree_layout(mains, parent):
    """mains: a clone layout whose checkouts move to a holder branch; worktrees go to parent."""
    parent.mkdir(parents=True)
    for name in SRC:
        b = branch_of(mains / name)
        git("switch", "-q", "-c", "holder-" + name.lower(), cwd=mains / name)
        git("worktree", "add", "-q", str(parent / name), b, cwd=mains / name)
    return parent


def launch(layout, preflight=True, extra=None, timeout=90):
    env = dict(os.environ, P1_LAUNCH_WORK=str(W / ("trial-" + layout.name)), P1_IDENTITY_PORT=str(free_port()), P1_PROCESSOR_PORT=str(free_port()))
    for k in ("P1_IDENTITY_SRC", "P1_THOUGHT_SRC", "P1_THOUGHT", "P1_PREFLIGHT_ONLY"):
        env.pop(k, None)
    if preflight:
        env["P1_PREFLIGHT_ONLY"] = "1"
    env.update(extra or {})
    return subprocess.run(["bash", str(layout / "Thought-Processor" / "run-p1.sh")], env=env, stdin=subprocess.DEVNULL,
                          capture_output=True, text=True, timeout=timeout)


def last_line(p):
    lines = (p.stdout + p.stderr).strip().splitlines()
    return lines[-1] if lines else ""


def main():
    want_tp = branch_of(PROC_SRC)
    if git("status", "--porcelain", "--untracked-files=no", "--", "run-p1.sh", cwd=PROC_SRC).stdout.strip():
        raise SystemExit("commit run-p1.sh first: this test runs the committed launcher")
    clones = clone_layout(W / "clones")
    p = launch(clones)
    rec(p.returncode == 0 and "PREFLIGHT OK" in p.stdout and "package matches pinned artifact" in p.stdout,
        "ordinary clones: preflight passes (repo, branch, commit, clean tree, Thought pin)", last_line(p))

    mains = clone_layout(W / "mains")
    wts = worktree_layout(mains, W / "worktrees")
    kinds = ["file" if (wts / n / ".git").is_file() else "dir" for n in SRC]
    rec(kinds == ["file", "file", "file"], "worktree layout really uses worktrees (.git is a file in all three)", ",".join(kinds))
    p = launch(wts)
    rec(p.returncode == 0 and "PREFLIGHT OK" in p.stdout and "package matches pinned artifact" in p.stdout,
        "Git worktrees: preflight passes with the same checks", last_line(p))
    p = launch(wts, extra={"P1_THOUGHT": "off"})
    rec(p.returncode == 0 and "Thought feature: not offered" in p.stdout, "P1_THOUGHT=off: worker-only trial still passes", last_line(p))

    # ---- negatives (each undone afterwards) ----
    junk = wts / "Thought" / "stray.txt"
    junk.write_text("x")
    p = launch(wts)
    rec(p.returncode != 0 and "Dirty working tree" in p.stderr, "dirty worktree is refused", last_line(p))
    junk.unlink()

    git("switch", "-q", "-c", "some-other-branch", cwd=wts / "Thought")
    p = launch(wts)
    rec(p.returncode != 0 and "must be checked out on" in p.stderr, "wrong branch in a worktree is refused", last_line(p))
    git("switch", "-q", SRC_BRANCH["Thought"], cwd=wts / "Thought")

    git("remote", "set-url", "origin", "https://github.com/Someone/identity.git", cwd=mains / "identity")
    p = launch(wts)
    rec(p.returncode != 0 and "different repository" in p.stderr, "a checkout of a different repository is refused (worktree inherits origin)", last_line(p))
    git("remote", "set-url", "origin", ORIGINS["identity"], cwd=mains / "identity")

    p = launch(wts, extra={"P1_IDENTITY_SRC": str(wts / "identity" / "internal")})
    rec(p.returncode != 0 and "inside another checkout" in p.stderr, "a subfolder of a checkout is refused", last_line(p))

    broken = W / "broken" / "Thought"
    broken.mkdir(parents=True)
    (broken / ".git").write_text("gitdir: /nonexistent/worktrees/Thought\n")
    p = launch(wts, extra={"P1_THOUGHT_SRC": str(broken)})
    rec(p.returncode != 0 and "not a working Git checkout" in p.stderr, "a broken worktree link is refused (not silently skipped)", last_line(p))

    f = wts / "Thought" / "feature" / "thought_feature.py"
    orig = f.read_bytes()
    f.write_bytes(orig + b"\n# changed\n")
    git("commit", "-q", "-am", "test: change the package", cwd=wts / "Thought")
    p = launch(wts)
    rec(p.returncode != 0 and "not the reviewed pinned artifact" in p.stderr, "committed but changed Thought package is refused before anything is built", last_line(p))
    git("reset", "-q", "--hard", "HEAD~1", cwd=wts / "Thought")

    base = git("rev-parse", "HEAD", cwd=wts / "Thought").stdout.strip()
    # Put an unrelated commit on the expected branch name in a fresh worktree to test ancestry.
    unrelated = W / "unrelated"
    git("clone", "-q", "--no-hardlinks", args.thought_src, str(unrelated / "Thought"))
    git("remote", "set-url", "origin", ORIGINS["Thought"], cwd=unrelated / "Thought")
    git("checkout", "-q", "--orphan", "x", cwd=unrelated / "Thought")
    git("commit", "-q", "-m", "unrelated history with the same files", cwd=unrelated / "Thought")
    git("branch", "-q", "-D", SRC_BRANCH["Thought"], cwd=unrelated / "Thought")
    git("branch", "-q", "-m", SRC_BRANCH["Thought"], cwd=unrelated / "Thought")
    p = launch(wts, extra={"P1_THOUGHT_SRC": str(unrelated / "Thought")})
    rec(p.returncode != 0 and "not descended from the reviewed commit" in p.stderr, "Thought with unrelated history is refused even if the files match", last_line(p))
    rec(git("rev-parse", "HEAD", cwd=wts / "Thought").stdout.strip() == base and branch_of(wts / "Thought") == SRC_BRANCH["Thought"], "(test layout restored)")

    p = launch(wts)
    rec(p.returncode == 0 and "PREFLIGHT OK" in p.stdout, "after undoing every change, the worktree layout passes again", last_line(p))

    # ---- one full launch from worktrees ----
    env = dict(os.environ, P1_LAUNCH_WORK=str(W / "trial-full"), P1_IDENTITY_PORT=str(free_port()), P1_PROCESSOR_PORT=str(free_port()))
    for k in ("P1_IDENTITY_SRC", "P1_THOUGHT_SRC", "P1_THOUGHT", "P1_PREFLIGHT_ONLY"):
        env.pop(k, None)
    out = open(W / "full-launch.txt", "w")
    proc = subprocess.Popen(["bash", str(wts / "Thought-Processor" / "run-p1.sh")], env=env, stdin=subprocess.DEVNULL, stdout=out, stderr=subprocess.STDOUT,
                            start_new_session=True)
    ready = False
    end = time.time() + 120
    while time.time() < end and proc.poll() is None:
        if "P1 Processor ready" in (W / "full-launch.txt").read_text():
            ready = True
            break
        time.sleep(0.5)
    text = (W / "full-launch.txt").read_text()
    idlog = (W / "trial-full" / "identity-private.log").read_text(errors="replace") if (W / "trial-full" / "identity-private.log").exists() else ""
    rec(ready and "package matches pinned artifact" in text and "operator-pinned feature artifact" in idlog,
        "full launch from worktrees: both programs ready, Thought pinned in Processor AND Identity")
    os.killpg(proc.pid, signal.SIGINT)
    try:
        proc.wait(30)
    except subprocess.TimeoutExpired:
        os.killpg(proc.pid, signal.SIGKILL)
    time.sleep(0.5)
    left = subprocess.run(["pgrep", "-f", str(W / "trial-full" / "bin")], capture_output=True, text=True).stdout.split()
    rec(not left, "Ctrl+C stops everything the launcher started", "left: " + ",".join(left) if left else "")
    rec(not (W / "trial-full" / ".launcher-running").exists(), "launcher lock released")
    secrets_in_output = [w for w in ("#code=", "#key=", "/approve#") if w in text]
    rec(not secrets_in_output, "non-interactive output contains no private link or approval link", ",".join(secrets_in_output))


SRC_BRANCH = {}

if __name__ == "__main__":
    for n, s in SRC.items():
        SRC_BRANCH[n] = branch_of(s)
    try:
        main()
    except Exception as e:
        import traceback
        traceback.print_exc()
        rec(False, "the test script itself failed", type(e).__name__ + ": " + str(e)[:200])
    failed = [r for r in results if r[0] != "PASS"]
    tp = git("rev-parse", "HEAD", cwd=PROC_SRC).stdout.strip()
    lines = ["run-p1.sh checkout regression — %s" % time.strftime("%Y-%m-%d %H:%M:%S %Z"), "processor commit: " + tp, ""] + \
            ["%s %s%s" % (w, n, ("  — " + d) if d else "") for w, n, d in results]
    lines.append("\nRESULT: %s (%d checks, %d failed)" % ("PASS" if not failed else "FAIL", len(results), len(failed)))
    (W / "launcher-checkouts-report.txt").write_text("\n".join(lines) + "\n")
    print(lines[-1].strip())
    sys.exit(0 if not failed else 1)
