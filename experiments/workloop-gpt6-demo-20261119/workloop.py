#!/usr/bin/env python3
"""Offline, synthetic Markdown coordination fixture. NO network, tokens or Identity calls."""
from __future__ import annotations

import argparse
from concurrent.futures import ThreadPoolExecutor
from dataclasses import dataclass
import hashlib
import importlib.util
import os
from pathlib import Path
import re
import tempfile
import threading

ROOT = Path(__file__).resolve().parent
SOURCES = ROOT / "fixtures"
REDACT = re.compile(r"(?i)(memory\.jsonl|identity-grant\.json|gh[pousr]_[A-Za-z0-9]{12,}|bearer\s+\S+|private[_ -]?memory)")


class GuardError(RuntimeError):
    """Failed authorization, transition, privacy, or optimistic-lock check."""


def file_digest(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


def run_cases(path: Path, function: str, cases: list[tuple[str, str]]) -> tuple[str, str]:
    """Run known trusted fixture source; NEVER point this loader at GitHub/model content."""
    spec = importlib.util.spec_from_file_location("fixture_candidate", path)
    assert spec and spec.loader
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    fn = getattr(module, function)
    failures = [f"{value!r}: got {fn(value)!r}, expected {expected!r}" for value, expected in cases if fn(value) != expected]
    return ("FAIL", "; ".join(failures)) if failures else ("PASS", f"{len(cases)} checked cases")


@dataclass
class Stream:
    builder: str
    reviewer: str
    state: str
    source: str | None = None
    digest: str | None = None
    smoke: str = "NOT RUN"
    audit: str = "NOT RUN"


class Workloop:
    def __init__(self, out: Path):
        self.out = out
        self.lock = threading.RLock()
        self.events: list[tuple[str, str, str, str]] = []
        self.versions: dict[str, int] = {}
        self.streams = {
            "WS-A": Stream("Claude", "Codex", "ASSIGNED"),
            "WS-B": Stream("Fixture author", "Codex", "TESTED", "status.py", file_digest(SOURCES / "status.py"), "PASS"),
        }
        # The fixture's previously completed candidate has a real smoke check.
        smoke, evidence = run_cases(SOURCES / "status.py", "public_status", [("READY", "READY")])
        if smoke != "PASS":
            raise GuardError(evidence)

    def receipt(self, task: str, actor: str, action: str, evidence: str, expected: int | None = None) -> None:
        if REDACT.search(evidence):
            raise GuardError("private or credential-like data rejected")
        if action not in {"BUILD", "TEST", "AUDIT", "REPAIR", "RETEST", "RE-AUDIT", "COORDINATE", "ACK_PROPOSED"}:
            raise GuardError("unrecognized event type")
        if task not in self.streams or actor not in {"Claude", "Codex", "GPT-6", "TASS-sim"}:
            raise GuardError("unknown stream or actor")
        if action in {"BUILD", "TEST", "REPAIR", "RETEST"} and actor != self.streams[task].builder:
            raise GuardError("builder-only event")
        if action in {"AUDIT", "RE-AUDIT"} and (actor != self.streams[task].reviewer or actor == self.streams[task].builder):
            raise GuardError("independent auditor required")
        if action == "COORDINATE" and actor != "GPT-6":
            raise GuardError("coordinator-only event")
        if action == "ACK_PROPOSED" and actor != "TASS-sim":
            raise GuardError("synthetic TASS-only event")
        key = f"{task}/{actor}"
        with self.lock:
            version = self.versions.get(key, 0)
            if expected is not None and expected != version:
                raise GuardError("stale receipt revision (CAS conflict)")
            self.versions[key] = version + 1
            self.events.append((task, actor, action, evidence))
            path = self.out / "handoffs" / task / (actor.replace(" ", "-") + ".md")
            path.parent.mkdir(parents=True, exist_ok=True)
            entries = [(a, ev, msg) for t, a, ev, msg in self.events if t == task and a == actor]
            content = f"# {task} / {actor}\n\nWriter: {actor} only. Receipt revision: {self.versions[key]}\n\n" + "".join(f"- **{act}** — {msg}\n" for _, act, msg in entries)
            with tempfile.NamedTemporaryFile(mode="w", dir=path.parent, encoding="utf-8", delete=False) as f:
                f.write(content)
                tmp = f.name
            os.replace(tmp, path)

    def build(self):
        s = self.streams["WS-A"]
        if s.state != "ASSIGNED":
            raise GuardError("build must follow assignment")
        s.source, s.digest, s.state = "normalize_v1.py", file_digest(SOURCES / "normalize_v1.py"), "BUILT"
        self.receipt("WS-A", "Claude", "BUILD", f"candidate fixture {s.source}; sha256:{s.digest}")
        s.smoke, details = run_cases(SOURCES / s.source, "normalize", [(" hello ", "hello")])
        s.state = "TESTED" if s.smoke == "PASS" else "BUILD_FAILED"
        self.receipt("WS-A", "Claude", "TEST", f"smoke {s.smoke}: {details}; candidate sha256:{s.digest}")

    def audit(self, task: str, candidate_digest: str, repeat: bool = False):
        s = self.streams[task]
        if s.state not in {"TESTED", "RETESTED"} or candidate_digest != s.digest:
            raise GuardError("audit requires exact current, tested candidate revision")
        if task == "WS-A":
            result, details = run_cases(SOURCES / str(s.source), "normalize", [(" hello  world ", "hello world"), (" Mixed Case ", "Mixed Case")])
        else:
            result, details = run_cases(SOURCES / str(s.source), "public_status", [("READY", "READY"), ("BLOCKED", "BLOCKED"), ("SUSPICIOUS", "UNKNOWN")])
        s.audit = result
        s.state = "AUDIT_PASS" if result == "PASS" else "NEEDS_REPAIR"
        action = "RE-AUDIT" if repeat else "AUDIT"
        self.receipt(task, "Codex", action, f"{result}: {details}; candidate sha256:{candidate_digest}")
        return result

    def repair(self):
        s = self.streams["WS-A"]
        if s.state != "NEEDS_REPAIR":
            raise GuardError("repair requires failed independent audit")
        s.source, s.digest, s.state = "normalize_v2.py", file_digest(SOURCES / "normalize_v2.py"), "REPAIRED"
        s.audit = "NOT RUN"  # The old audit cannot certify a new candidate.
        self.receipt("WS-A", "Claude", "REPAIR", f"new fixture {s.source}; sha256:{s.digest}; prior audit invalidated")
        s.smoke, smoke_note = run_cases(SOURCES / s.source, "normalize", [(" hello ", "hello")])
        regression, regression_note = run_cases(SOURCES / s.source, "normalize", [(" hello  world ", "hello world")])
        if s.smoke != "PASS" or regression != "PASS":
            s.state = "RETEST_FAILED"
            raise GuardError("repair retest failed")
        s.state = "RETESTED"
        self.receipt("WS-A", "Claude", "RETEST", f"PASS: smoke {smoke_note}; regression {regression_note}; sha256:{s.digest}")

    def proposed_tass_ack(self):
        # No live model, repository sync, Identity grant, or task execution.
        self.receipt("WS-A", "TASS-sim", "ACK_PROPOSED", "Read synthetic assignment WS-A; response: ready to observe, NOT AUTHORIZED TO ACT; NOT SENT")

    def index(self, title: str) -> str:
        lines = [f"# {title}", "", "Fixture only — not live TASS, no owner authorization, no GitHub action.", "", "| Stream | Builder | Auditor | State | Candidate SHA-256 | Smoke/retest | Independent audit |", "| --- | --- | --- | --- | --- | --- | --- |"]
        for key, s in sorted(self.streams.items()):
            lines.append(f"| {key} | {s.builder} | {s.reviewer} | {s.state} | {(s.digest or 'none')[:16]} | {s.smoke} | {s.audit} |")
        return "\n".join(lines) + "\n"

    def run(self):
        self.out.mkdir(parents=True, exist_ok=True)
        (self.out / "BEFORE.md").write_text(self.index("BEFORE — synthetic assignment state"), encoding="utf-8")
        self.receipt("WS-A", "GPT-6", "COORDINATE", "Synthetic task assigned to Claude; Codex reviews independent WS-B candidate")
        self.proposed_tass_ack()
        # Actual independent local threads; separate per-stream record ownership.
        with ThreadPoolExecutor(max_workers=2) as pool:
            a = pool.submit(self.build)
            b = pool.submit(lambda: self.audit("WS-B", self.streams["WS-B"].digest))
            a.result(); b.result()
        old_digest = self.streams["WS-A"].digest
        first_result = self.audit("WS-A", old_digest)
        if first_result != "FAIL" or self.streams["WS-A"].state != "NEEDS_REPAIR":
            raise GuardError("required negative audit did not fail")
        self.repair()
        self.audit("WS-A", self.streams["WS-A"].digest, repeat=True)
        if not all(s.state == "AUDIT_PASS" for s in self.streams.values()):
            raise GuardError("final stream not independently accepted")
        self.receipt("WS-A", "GPT-6", "COORDINATE", "Both synthetic audits PASS; fixture stops before owner merge/deploy/grant gate")
        (self.out / "AFTER.md").write_text(self.index("AFTER — synthetic completed cycle"), encoding="utf-8")
        lines = ["# Synthetic event evidence", "", "Not GitHub events. Source digests are SHA-256 of local fixture files, not Git commit SHAs.", ""]
        lines += [f"{i}. **{task} / {actor} / {action}** — {evidence}" for i, (task, actor, action, evidence) in enumerate(self.events, 1)]
        (self.out / "EVENTS.md").write_text("\n".join(lines) + "\n", encoding="utf-8")
        return self


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--out", type=Path, default=ROOT / "records" / "run")
    args = parser.parse_args()
    result = Workloop(args.out).run()
    print(f"DEMO PASS; before={args.out / 'BEFORE.md'} after={args.out / 'AFTER.md'} events={len(result.events)}")


if __name__ == "__main__":
    main()
