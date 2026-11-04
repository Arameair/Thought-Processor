"""Local operator task board for Atrium. Not TASS memory or authorization.

The board contains user-authored task notes and owner-reported test results.
It cannot execute code, change grants, or write to GitHub.
"""
from __future__ import annotations

import json
import os
import re
import tempfile
import threading
from pathlib import Path

DAY = "2026-11-04"
MAX_TASKS = 40
STATUSES = {"planned", "working", "blocked", "review"}
RESULTS = {"unverified", "reported_pass", "failed", "blocked"}

SEED_TASKS = [
    {
        "id": "baseline",
        "title": "Protect yesterday's Docker baseline",
        "why": "Find the exact running configuration before changing or rebuilding anything.",
        "next_step": "Read-only inventory: container names, ports, repo branches and SHAs; inspect the launcher before using it.",
        "expected": "A sanitized list of running components and their source versions, with no service stopped.",
    },
    {
        "id": "runtime",
        "title": "Confirm which process is TASS",
        "why": "Notebook, Go feature, and Workshop prototypes should not masquerade as one runtime.",
        "next_step": "Inspect live Workshop status, identify the real model endpoint and record one reply.",
        "expected": "One attributable real model response and the exact TASS process/revision; fixture status separately labeled.",
    },
    {
        "id": "mind",
        "title": "Reconcile the TASS mind source",
        "why": "Processor's alpha Compose points at Developmnt/mind while TASS/alpha declares TASS/mind.",
        "next_step": "Read current mirror source, review draft Processor PR #36 and Identity PR #29; do not cut over yet.",
        "expected": "Documented current source, proposed canonical source, safe rollback, and outstanding approval.",
    },
    {
        "id": "engine",
        "title": "Select the canonical Thought Engine",
        "why": "Thought, Thought-engine, and Workshop traces currently overlap in concept.",
        "next_step": "Compare actual running event producers and identify which engine owns authoritative cognition.",
        "expected": "A selected candidate or an explicit BLOCKED decision with evidence, not a cosmetic thought stream.",
    },
    {
        "id": "idle",
        "title": "Prove a meaningful 20-30 second cycle",
        "why": "TASS must be able to stay IDLE instead of manufacturing thoughts or unnecessary actions.",
        "next_step": "Observe several no-work ticks, then send one harmless approved observation and trace its outcome.",
        "expected": "IDLE without fabricated events, a real attributable cognition event when work exists, then IDLE again.",
    },
    {
        "id": "authority",
        "title": "Check owner approval and authorization",
        "why": "Workshop's experimental Liaison is not equivalent to enforced Identity P1 grants.",
        "next_step": "Review one harmless proposal in Workshop; confirm nothing executes before owner approval.",
        "expected": "An audited accept/deny outcome; do not enable autonomous execution or handle secrets.",
    },
]


class InvalidBoard(ValueError):
    pass


class StaleBoard(ValueError):
    pass


def _string(v, name, maximum, *, allow_empty=False):
    if not isinstance(v, str) or len(v) > maximum or (not allow_empty and not v.strip()):
        raise InvalidBoard("invalid " + name)
    if "\x00" in v:
        raise InvalidBoard("invalid " + name)
    return v.strip()


def starter_board():
    return {
        "version": 1,
        "day": DAY,
        "revision": 0,
        "selected": "baseline",
        "tasks": [dict(t, status="planned", result="unverified", notes="", evidence="") for t in SEED_TASKS],
    }


def validate_board(value):
    if not isinstance(value, dict) or value.get("version") != 1 or value.get("day") != DAY:
        raise InvalidBoard("invalid board format or mission date")
    revision = value.get("revision")
    if not isinstance(revision, int) or isinstance(revision, bool) or revision < 0:
        raise InvalidBoard("invalid revision")
    tasks = value.get("tasks")
    if not isinstance(tasks, list) or not 1 <= len(tasks) <= MAX_TASKS:
        raise InvalidBoard("invalid task count")
    clean_tasks, ids = [], set()
    for task in tasks:
        if not isinstance(task, dict):
            raise InvalidBoard("invalid task")
        task_id = _string(task.get("id"), "id", 64)
        if not re.fullmatch(r"[a-z0-9][a-z0-9-]{0,63}", task_id) or task_id in ids:
            raise InvalidBoard("duplicate or invalid task id")
        ids.add(task_id)
        entry = {"id": task_id}
        for field, maximum, allow_empty in (
            ("title", 120, False), ("why", 500, True),
            ("next_step", 750, True), ("expected", 750, True),
            ("notes", 1500, True), ("evidence", 1200, True),
        ):
            entry[field] = _string(task.get(field, ""), field, maximum, allow_empty=allow_empty)
        if task.get("status") not in STATUSES or task.get("result") not in RESULTS:
            raise InvalidBoard("invalid status or result")
        entry["status"] = task["status"]
        entry["result"] = task["result"]
        if entry["result"] == "reported_pass" and not entry["evidence"]:
            raise InvalidBoard("a reported PASS requires an evidence note")
        clean_tasks.append(entry)
    selected = value.get("selected")
    if selected not in ids:
        raise InvalidBoard("selected task not found")
    return {"version": 1, "day": DAY, "revision": revision,
            "selected": selected, "tasks": clean_tasks}


class BoardStore:
    """Atomic file-backed notes, with optimistic revision control for two tabs."""

    def __init__(self, path):
        self.path = Path(path)
        self.lock = threading.RLock()

    def read(self):
        with self.lock:
            if not self.path.exists():
                return starter_board()
            raw = self.path.read_text(encoding="utf-8")
            if len(raw) > 100_000:
                raise InvalidBoard("stored board too large")
            return validate_board(json.loads(raw))

    def write(self, proposed):
        with self.lock:
            new = validate_board(proposed)
            current = self.read()
            if new["revision"] != current["revision"]:
                raise StaleBoard("board changed in another tab; reload before saving")
            new["revision"] += 1
            self.path.parent.mkdir(parents=True, exist_ok=True)
            fd, tmp = tempfile.mkstemp(prefix=".atrium-", dir=self.path.parent)
            try:
                with os.fdopen(fd, "w", encoding="utf-8") as out:
                    json.dump(new, out, ensure_ascii=False, indent=2)
                    out.write("\n")
                    out.flush()
                    os.fsync(out.fileno())
                os.chmod(tmp, 0o600)
                os.replace(tmp, self.path)
            finally:
                if os.path.exists(tmp):
                    os.unlink(tmp)
            return new
