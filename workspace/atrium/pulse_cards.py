"""Atrium Issue #51: narrowly scoped local test cards and pulse coordinator.

This is NOT Identity authorization or the Go thought pulse. It runs no model text,
GitHub instructions, user-selected command/path, Docker command or arbitrary shell.
Default: DISABLED. Optional execution requires an independently sandboxed process.
"""
from __future__ import annotations

import fcntl
import hashlib
import json
import os
import re
import resource
import shutil
import signal
import subprocess
import tempfile
import threading
import time
import uuid
from datetime import datetime, timezone
from pathlib import Path

IDENTITY_SHA = "b0c7267b47f6111c5024c9bba8a4135e2357b432"
INTEGRITY_SHA = "aa800af3eee5d026521395bde3236737285e6287"
CATALOG = {
    "IDENTITY-014": {
        "id": "IDENTITY-014",
        "objective": "Run the approved Go Identity authority, pairing and entities package tests.",
        "expected": "Exit 0, with real Go test output from the pinned Identity alpha checkout.",
        "repository": "Arameair/identity",
        "sha": IDENTITY_SHA,
        "operation": "identity_go_packages",
        "argv": ["go", "test", "./internal/authority/...", "./internal/pairing/...", "./internal/entities/..."],
        "subdir": "",
        "env_key": "TASS_ATRIUM_TEST_IDENTITY_CHECKOUT",
    },
    "INTEGRITY-050": {
        "id": "INTEGRITY-050",
        "objective": "Run PR #49's independent integrity regression assertions against the Go Core.",
        "expected": "Known P0 tests FAIL on this revision; nonzero exit is FAIL, not a green expected failure.",
        "repository": "Arameair/Thought-Processor",
        "sha": INTEGRITY_SHA,
        "operation": "integrity_go_red",
        "argv": ["go", "test", "-race", "-tags=acceptance", "./tass", "-run=^TestAcceptanceRejects", "-count=1", "-v"],
        "subdir": "core",
        "env_key": "TASS_ATRIUM_TEST_INTEGRITY_CHECKOUT",
    },
}
FINAL = {"PASS", "FAIL", "ERROR", "BLOCKED"}
CRED = re.compile(
    r"(?i)(?:github_pat_[A-Za-z0-9_]+|gh[pousr]_[A-Za-z0-9_]+|"
    r"(?:(?:bearer|password|passwd|api[_-]?key|token|secret)\s*[:=]?\s*)[^\s,;]+|"
    r"(?:[?&](?:token|key|code)=)[^&\s]+)"
)
TIMEOUT = 180
PREVIEW = 1200


def utc():
    return datetime.now(timezone.utc).isoformat(timespec="milliseconds")


def redact(value):
    return CRED.sub("[REDACTED]", str(value)).replace("\x00", "")[:PREVIEW]


class CardError(ValueError):
    def __init__(self, detail, code=400):
        super().__init__(detail)
        self.code = code


def make_card(spec):
    return {
        **{k: v for k, v in spec.items() if k != "env_key"},
        "status": "PROPOSED",
        "approval": None,
        "job_id": None,
        "attempts": 0,
        "started_at": None,
        "finished_at": None,
        "receipt": None,
        "pending_receipt": None,
        "reason": None,
    }


def initial():
    return {
        "schema": 1, "paused": True, "tick_sequence": 0, "last_tick": None,
        "last_pulse_id": None, "cards": {key: make_card(v) for key, v in CATALOG.items()},
    }


class CardStore:
    """One Atrium owner process at a time, with durable atomic state and recovery."""

    def __init__(self, file_path):
        self.path = Path(file_path)
        self.path.parent.mkdir(parents=True, exist_ok=True)
        self._mutex = threading.RLock()
        self._lockfile = os.open(str(self.path) + ".lock", os.O_CREAT | os.O_RDWR, 0o600)
        try:
            fcntl.flock(self._lockfile, fcntl.LOCK_EX | fcntl.LOCK_NB)
        except BlockingIOError as exc:
            os.close(self._lockfile)
            raise CardError("Test-card state is already owned by another Atrium process", 503) from exc
        try:
            with self._mutex:
                doc = self._read()
                changed = False
                for card in doc["cards"].values():
                    if card["status"] == "WORKING":
                        if card.get("pending_receipt"):
                            self._complete(card)
                        else:
                            card.update(status="BLOCKED", reason="UNKNOWN_INFLIGHT after restart; never auto-retry",
                                        finished_at=utc())
                        changed = True
                if changed or not self.path.exists():
                    self._write(doc)
        except Exception:
            self.close()
            raise

    def close(self):
        if self._lockfile is not None:
            fcntl.flock(self._lockfile, fcntl.LOCK_UN)
            os.close(self._lockfile)
            self._lockfile = None

    def _read(self):
        if not self.path.exists():
            return initial()
        raw = self.path.read_bytes()
        if len(raw) > 128_000:
            raise CardError("Test-card state too large; existing data not overwritten", 503)
        try:
            doc = json.loads(raw)
            if doc["schema"] != 1 or set(doc["cards"]) != set(CATALOG):
                raise ValueError("schema or cards do not match")
            for key, card in doc["cards"].items():
                spec = CATALOG[key]
                for field in ("id", "sha", "repository", "operation", "argv", "subdir"):
                    if card.get(field) != spec[field]:
                        raise ValueError(f"definition changed for {key}")
                if card.get("status") not in {"PROPOSED", "APPROVED", "QUEUED", "WORKING", *FINAL}:
                    raise ValueError("unknown state")
            return doc
        except (KeyError, TypeError, ValueError, json.JSONDecodeError) as exc:
            raise CardError("Test-card state invalid; preserved, manual review required", 503) from exc

    def _write(self, doc):
        fd, path = tempfile.mkstemp(prefix=".pulse-", dir=self.path.parent)
        try:
            with os.fdopen(fd, "w", encoding="utf-8") as out:
                json.dump(doc, out, sort_keys=True, separators=(",", ":"))
                out.write("\n")
                out.flush()
                os.fsync(out.fileno())
            os.chmod(path, 0o600)
            os.replace(path, self.path)
            # Directory sync ensures the atomic rename itself is durable on Linux.
            d = os.open(self.path.parent, os.O_RDONLY)
            try:
                os.fsync(d)
            finally:
                os.close(d)
        finally:
            if os.path.exists(path):
                os.unlink(path)

    def read(self):
        with self._mutex:
            if self._lockfile is None:
                raise CardError("Test-card store is closed", 503)
            return self._read()

    def mutate(self, fn):
        with self._mutex:
            if self._lockfile is None:
                raise CardError("Test-card store is closed", 503)
            doc = self._read()
            result = fn(doc)
            self._write(doc)
            return result

    @staticmethod
    def _complete(card):
        receipt = card.pop("pending_receipt")
        # A worker receipt is evidence of execution; PASS needs exit=0, not a model's opinion.
        card["receipt"] = receipt
        card["status"] = receipt["outcome"]
        card["finished_at"] = receipt["finished_at"]
        card["reason"] = receipt.get("reason")


class LocalTestAdapter:
    """Only fixed Go test argv on exact preexisting, isolated, clean checkouts."""

    def __init__(self, *, enabled=None, sandbox_confirmed=None, roots=None, root=None):
        self.enabled = (os.getenv("TASS_ATRIUM_TEST_EXECUTION") == "1") if enabled is None else enabled
        self.sandbox_confirmed = (os.getenv("TASS_ATRIUM_TEST_SANDBOX_CONFIRMED") == "1") if sandbox_confirmed is None else sandbox_confirmed
        self.root = root if root is not None else os.getenv("TASS_ATRIUM_TEST_ROOT", "")
        self.roots = roots if roots is not None else {
            k: os.getenv(v["env_key"], "") for k, v in CATALOG.items()
        }

    def preflight(self, card):
        # Fail closed if identity authorization or process isolation is unknown.
        if not self.enabled:
            raise CardError("Execution disabled (TASS_ATRIUM_TEST_EXECUTION is not 1)", 403)
        if not self.sandbox_confirmed:
            raise CardError("Isolated runner not confirmed; no privileged execution", 403)
        if os.name != "posix" or os.geteuid() == 0:
            raise CardError("Runner must be non-root on POSIX", 403)
        key = card.get("id")
        spec = CATALOG.get(key)
        if spec is None or any(card.get(k) != spec.get(k) for k in ("repository", "sha", "operation", "argv", "subdir")):
            raise CardError("Unknown or modified allowlisted operation", 403)
        source = self.roots.get(key, "")
        if not self.root or not source or not Path(self.root).is_absolute() or not Path(source).is_absolute():
            raise CardError("An explicit isolated checkout root and source are required", 403)
        base, checkout = Path(self.root).resolve(strict=True), Path(source).resolve(strict=True)
        if base != Path(self.root) or checkout == base or not checkout.is_relative_to(base) or checkout != Path(source):
            raise CardError("Checkout must be a direct non-symlink path below the isolated test root", 403)
        if not checkout.is_dir() or not (checkout / ".git").is_dir() or (checkout / ".git").is_symlink():
            raise CardError("Checkout missing Git metadata", 403)
        clean_env = self._git_env()
        def git(*argv):
            r = subprocess.run(["git", "-c", "core.fsmonitor=false", "-c", "core.hooksPath=/dev/null",
                                "-C", str(checkout), *argv], capture_output=True,
                               text=True, timeout=8, env=clean_env, check=False)
            if r.returncode:
                raise CardError("Cannot verify checkout revision and repository", 403)
            return r.stdout.strip()
        if Path(git("rev-parse", "--show-toplevel")).resolve() != checkout:
            raise CardError("Checkout is not a separate repository root", 403)
        if git("rev-parse", "HEAD") != spec["sha"]:
            raise CardError("Checkout SHA does not match the approved card", 403)
        remote = git("remote", "get-url", "origin")
        known = {f"https://github.com/{spec['repository']}",
                 f"https://github.com/{spec['repository']}.git",
                 f"git@github.com:{spec['repository']}.git"}
        if remote not in known:
            raise CardError("Checkout origin does not match the approved repository", 403)
        if git("status", "--porcelain", "--untracked-files=all"):
            raise CardError("Checkout is not clean; never test a mutable live checkout", 403)
        cwd = checkout / spec["subdir"]
        if not cwd.is_dir() or cwd.is_symlink():
            raise CardError("Fixed package directory missing or symlinked", 403)
        if not shutil.which("go"):
            raise CardError("Go compiler not installed in isolated runner", 503)
        return cwd

    @staticmethod
    def _git_env():
        return {
            "PATH": os.defpath + ":/usr/local/bin:/usr/local/go/bin",
            "HOME": "/nonexistent",
            "GIT_CONFIG_NOSYSTEM": "1", "GIT_CONFIG_GLOBAL": "/dev/null",
            "GIT_TERMINAL_PROMPT": "0", "GIT_OPTIONAL_LOCKS": "0",
        }

    @staticmethod
    def _limits():
        resource.setrlimit(resource.RLIMIT_CPU, (TIMEOUT, TIMEOUT + 10))
        # Never set RLIMIT_FSIZE: it also breaks Go compiler output archives.
        # Bound the captured stdout file in the parent loop instead.
        resource.setrlimit(resource.RLIMIT_NOFILE, (64, 64))
        # Memory and network must be limited by the EXTERNAL sandbox/cgroup.
        # Go's race runtime needs high virtual address space; RLIMIT_AS is unsafe here.

    def execute(self, card, job_id, cancel=None):
        started = utc()
        cwd = self.preflight(card)  # Check SHA AGAIN just before launching.
        with tempfile.TemporaryDirectory(prefix="atrium-test-") as tmp:
            env = {
                "PATH": os.environ.get("PATH", os.defpath),
                "HOME": tmp, "TMPDIR": tmp, "GOCACHE": tmp + "/go-cache",
                "GOMODCACHE": tmp + "/modcache", "GOPROXY": "off",
                "GOSUMDB": "off", "GOENV": "off", "GOTOOLCHAIN": "local",
                "GIT_TERMINAL_PROMPT": "0", "GIT_CONFIG_NOSYSTEM": "1",
                "GIT_CONFIG_GLOBAL": "/dev/null", "GIT_OPTIONAL_LOCKS": "0",
                "CGO_ENABLED": "1", "LANG": "C", "LC_ALL": "C",
            }
            with tempfile.TemporaryFile(mode="w+b") as output:
                proc = subprocess.Popen(card["argv"], cwd=cwd, env=env,
                                        stdin=subprocess.DEVNULL, stdout=output,
                                        stderr=subprocess.STDOUT, shell=False,
                                        start_new_session=True, preexec_fn=self._limits)
                deadline = time.monotonic() + TIMEOUT
                timeout, cancelled, output_limited = False, False, False
                while proc.poll() is None:
                    if cancel is not None and cancel.is_set():
                        cancelled = True
                        break
                    if time.monotonic() >= deadline:
                        timeout = True
                        break
                    output.seek(0, os.SEEK_END)
                    if output.tell() > 2 << 20:
                        output_limited = True
                        break
                    try:
                        proc.wait(timeout=0.2)
                    except subprocess.TimeoutExpired:
                        continue
                if timeout or cancelled or output_limited:
                    try:
                        os.killpg(proc.pid, signal.SIGKILL)
                    except ProcessLookupError:
                        pass
                    proc.wait(timeout=10)
                output.seek(0, os.SEEK_END)
                output_limited = output_limited or output.tell() > (2 << 20)
                output.seek(0)
                digest = hashlib.sha256()
                preview = bytearray()
                while chunk := output.read(8192):
                    digest.update(chunk)
                    if len(preview) < 8192:
                        preview.extend(chunk[:8192 - len(preview)])
        raw_excerpt = preview.decode("utf-8", "replace")
        test_build_failed = ("[build failed]" in raw_excerpt or "compile: " in raw_excerpt)
        integrity_no_assertions = (card["id"] == "INTEGRITY-050" and proc.returncode != 0 and
                                   "--- FAIL: TestAcceptanceRejects" not in raw_excerpt)
        result = "ERROR" if (timeout or cancelled or output_limited or test_build_failed or integrity_no_assertions) else ("PASS" if proc.returncode == 0 else "FAIL")
        return {
            "receipt_id": "receipt-" + uuid.uuid4().hex,
            "job_id": job_id, "card_id": card["id"], "operation": card["operation"],
            "repository": card["repository"], "sha": card["sha"],
            "argv": list(card["argv"]), "started_at": started,
            "finished_at": utc(), "exit_code": None if (timeout or cancelled or output_limited) else proc.returncode,
            "outcome": result, "executed": True, "environment": "OWNER_SANDBOX",
            "output_sha256": digest.hexdigest(),
            "output_excerpt": redact(raw_excerpt),
            "reason": ("Test timed out; process group killed" if timeout else "Owner stopped test; process group killed" if cancelled else "Output exceeded 2 MiB limit" if output_limited else "Go build failed or regression tests never ran" if (test_build_failed or integrity_no_assertions) else None),
        }


class PulseCoordinator:
    """Owns state/dispatch; single daemon scheduler INSIDE Atrium, not an AI runtime."""

    def __init__(self, path, *, adapter=None, interval=20, auto=False):
        if not 1 <= int(interval) <= 3600:
            raise CardError("Pulse interval outside 1..3600 seconds")
        self.interval = int(interval)
        self.adapter = adapter or LocalTestAdapter()
        self.store = CardStore(path)
        self._stop = threading.Event()
        self._thread = None
        self._cancels = {}
        self._workers = {}
        self.next_tick_at = time.monotonic() + self.interval
        if auto:
            self.start()

    def close(self):
        self._stop.set()
        for event in list(self._cancels.values()):
            event.set()
        for worker in list(self._workers.values()):
            worker.join(timeout=3)
        if self._thread is not None:
            self._thread.join(timeout=2)
        self.store.close()

    def start(self):
        if self._thread is not None:
            return
        def loop():
            while not self._stop.wait(self.interval):
                try:
                    self.tick()
                except (CardError, OSError, ValueError):
                    # Store remains available for explicit error inspection; no retries of work.
                    pass
                self.next_tick_at = time.monotonic() + self.interval
        self._thread = threading.Thread(target=loop, name="atrium-pulse", daemon=True)
        self._thread.start()

    def view(self):
        doc = self.store.read()
        # Never expose internal executable checkout paths or environment variables.
        return {
            "interval_seconds": self.interval, "next_tick_seconds": max(0, round(self.next_tick_at - time.monotonic())),
            "execution_enabled": bool(self.adapter.enabled and self.adapter.sandbox_confirmed),
            "authority": "LOCAL_SANDBOX_PILOT_NOT_IDENTITY",
            "paused": doc["paused"], "tick_sequence": doc["tick_sequence"],
            "last_tick": doc["last_tick"], "last_pulse_id": doc["last_pulse_id"],
            "cards": [{k: v for k, v in card.items() if k != "pending_receipt"}
                      for card in doc["cards"].values()],
        }

    def action(self, action, card_id=None, ack=None):
        if action not in {"approve", "queue", "pause", "resume", "step", "stop"}:
            raise CardError("Unsupported test-card action", 403)
        if action == "step":
            self.tick(force=True)
            return self.view()
        if action in {"approve", "queue"} and card_id not in CATALOG:
            raise CardError("Unknown card", 403)
        if action == "queue" and not (self.adapter.enabled and self.adapter.sandbox_confirmed):
            raise CardError("Execution is disabled or isolation is not confirmed", 403)
        def change(doc):
            if action in {"pause", "resume", "stop"}:
                doc["paused"] = action != "resume"
                if action == "stop":
                    for card in doc["cards"].values():
                        if card["status"] == "WORKING":
                            event = self._cancels.get(card["job_id"])
                            if event is not None:
                                event.set()
                        if card["status"] == "QUEUED":
                            card.update(status="BLOCKED", reason="Stopped by owner before dispatch",
                                        finished_at=utc())
                return
            card = doc["cards"][card_id]
            if action == "approve":
                if ack != "APPROVE " + card_id:
                    raise CardError("Explicit card-specific owner acknowledgement is required", 403)
                if card["status"] != "PROPOSED":
                    raise CardError("Only PROPOSED cards can be approved", 409)
                card["approval"] = {"at": utc(), "kind": "LOCAL_UI_ATTESTATION_NOT_IDENTITY"}
                card["status"] = "APPROVED"
            elif action == "queue":
                if card["status"] in {"QUEUED", "WORKING", *FINAL}:
                    return  # idempotency: repeated queue cannot reset a job or rerun it.
                if card["status"] != "APPROVED" or not card["approval"]:
                    raise CardError("Explicit owner approval is required before queueing", 403)
                card["job_id"] = "job-" + uuid.uuid4().hex
                card["attempts"] += 1
                card["status"] = "QUEUED"
                card["reason"] = None
        self.store.mutate(change)
        return self.view()

    def tick(self, *, force=False):
        chosen = None
        def dispatch(doc):
            nonlocal chosen
            doc["tick_sequence"] += 1
            doc["last_pulse_id"] = "pulse-" + uuid.uuid4().hex
            doc["last_tick"] = utc()
            for card in doc["cards"].values():
                if card["status"] == "WORKING" and card.get("pending_receipt"):
                    self.store._complete(card)
            if doc["paused"] and not force:
                return
            if any(c["status"] == "WORKING" for c in doc["cards"].values()):
                return
            for card_id in CATALOG:
                card = doc["cards"][card_id]
                if card["status"] != "QUEUED":
                    continue
                if not card.get("approval") or not (self.adapter.enabled and self.adapter.sandbox_confirmed):
                    card.update(status="BLOCKED", reason="Execution or local approval missing; never dispatched", finished_at=utc())
                    continue
                try:
                    self.adapter.preflight(card)
                except (CardError, OSError, subprocess.SubprocessError) as exc:
                    card.update(status="BLOCKED", reason=redact(exc), finished_at=utc())
                    continue
                # Durable claim written BEFORE any process starts. This job ID is one-shot.
                card["status"] = "WORKING"
                card["started_at"] = utc()
                chosen = (card_id, card["job_id"], dict(card))
                self._cancels[card["job_id"]] = threading.Event()
                break
        self.store.mutate(dispatch)
        if chosen:
            worker = threading.Thread(target=self._execute, args=chosen, daemon=True, name="atrium-card-worker")
            self._workers[chosen[1]] = worker
            try:
                worker.start()
            except RuntimeError as exc:
                self._workers.pop(chosen[1], None)
                self._cancels.pop(chosen[1], None)
                self._failure(chosen[0], chosen[1], "Worker could not launch: " + str(exc))

    def _failure(self, card_id, job_id, message):
        if self.store._lockfile is None:
            return  # Prior instance no longer owns the file. Recovery owns reconciliation.
        finished = utc()
        def change(doc):
            card = doc["cards"][card_id]
            if card["status"] == "WORKING" and card["job_id"] == job_id:
                card["pending_receipt"] = {
                    "receipt_id": "receipt-" + uuid.uuid4().hex, "job_id": job_id,
                    "card_id": card_id, "operation": card["operation"], "repository": card["repository"],
                    "sha": card["sha"], "argv": list(card["argv"]),
                    "started_at": card["started_at"], "finished_at": finished,
                    "exit_code": None, "outcome": "ERROR", "executed": False,
                    "environment": "OWNER_SANDBOX", "output_sha256": None,
                    "output_excerpt": "", "reason": redact(message),
                }
        self.store.mutate(change)

    def _execute(self, card_id, job_id, card):
        try:
            receipt = self.adapter.execute(card, job_id, cancel=self._cancels.get(job_id))
            if (receipt.get("card_id") != card_id or receipt.get("job_id") != job_id
                    or receipt.get("sha") != card["sha"] or receipt.get("outcome") not in {"PASS", "FAIL", "ERROR"}):
                raise CardError("Adapter receipt mismatch")
            # Only allowlisted receipt fields can be persisted/displayed.
            keys = ("receipt_id", "job_id", "card_id", "operation", "repository", "sha",
                    "argv", "started_at", "finished_at", "exit_code", "outcome", "executed",
                    "environment", "output_sha256", "output_excerpt", "reason")
            receipt = {key: receipt.get(key) for key in keys}
            if self.store._lockfile is None:
                return  # Never write from an old worker after process shutdown/restart.
            receipt["output_excerpt"] = redact(receipt.get("output_excerpt") or "")
            receipt["reason"] = redact(receipt.get("reason") or "") or None
            def record(doc):
                live = doc["cards"][card_id]
                if live["status"] == "WORKING" and live["job_id"] == job_id and not live.get("pending_receipt"):
                    live["pending_receipt"] = receipt
            self.store.mutate(record)
        except Exception as exc:
            self._failure(card_id, job_id, type(exc).__name__ + ": " + str(exc))
        finally:
            self._cancels.pop(job_id, None)
            self._workers.pop(job_id, None)
