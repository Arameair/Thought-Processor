"""Owner-started, tokenless Workshop -> GitHub evidence publisher.

This optional process only reads local Workshop evidence and queues bounded comments
for the existing Identity Liaison. It has no GitHub credentials or network access.
It starts at the current event high-water mark: past history is not uploaded.
"""
from __future__ import annotations

import argparse
import json
import os
from pathlib import Path
import re
import secrets
import time

ALLOWED = {"user", "reply", "book", "proposal", "decision", "result", "error"}
CODE = re.compile(r"^(?:RUN\s+)?library/[\w.\-]+\.(?:md|ipynb)#\d+$")
SECRET = re.compile(r"(?i)(?:gh[pousr]_[A-Za-z0-9_]{15,}|github_pat_[A-Za-z0-9_]{15,}|"
                    r"(?:token|password|api[_-]?key|authorization)\s*[:=]\s*[^\s,;]+|"
                    r"bearer\s+\S+|[?&](?:token|key|code)=[^\s&]+)")
MAX_ROWS = 18
ISSUE_REPO = "Arameair/TASS"


def clean_text(value: object, limit: int = 300) -> str:
    if not isinstance(value, str):
        return ""
    # Mask common secret patterns and keep GitHub markdown tables single-line.
    return SECRET.sub("[REDACTED]", value).replace("\r", " ").replace("\n", " ").replace("|", "/")[:limit]


def event_summary(event: dict, share_text: bool = False) -> str | None:
    """Make an evidence statement, not a copy of the complete event."""
    kind = event.get("type")
    if kind not in ALLOWED or not isinstance(event.get("id"), int):
        return None
    if kind in {"user", "reply"}:
        detail = clean_text(event.get("text")) if share_text else "[text withheld]"
    elif kind == "book":
        detail = clean_text(event.get("path"), 100) + " opened"
    elif kind == "proposal":
        p = event.get("proposal") or {}
        detail = "Proposed " + clean_text(p.get("label") or p.get("source") or p.get("kind"), 100)
    elif kind == "decision":
        detail = "Proposal " + clean_text(event.get("pid"), 45) + " → " + clean_text(event.get("status"), 35)
    elif kind == "result":
        label = clean_text(event.get("label"), 140)
        verdict = "PASS (executor reported ok)" if event.get("ok") is True else "FAIL (executor reported error)"
        detail = label + ": " + verdict
    else:
        detail = "[error recorded; details withheld]"
    return f"| {event['id']} | {kind} | {detail} |"


def read_events(data_dir: Path) -> list[dict]:
    try:
        raw = json.loads((data_dir / "timeline.json").read_text("utf-8"))
        return raw if isinstance(raw, list) else []
    except (OSError, ValueError):
        return []


class Publisher:
    def __init__(self, data_dir: Path, outbox: Path, state_dir: Path, issue: int,
                 share_text: bool = False):
        if issue < 1:
            raise ValueError("A positive, owner-created private TASS GitHub issue number is required")
        self.data_dir, self.outbox, self.state_dir = data_dir, outbox, state_dir
        self.issue, self.share_text = issue, share_text
        self.state_file = state_dir / "cursor.json"

    def _cursor(self) -> int | None:
        try:
            return int(json.loads(self.state_file.read_text("utf-8"))["last_id"])
        except (OSError, KeyError, ValueError, TypeError):
            return None

    def _save_cursor(self, last_id: int) -> None:
        self.state_dir.mkdir(parents=True, exist_ok=True)
        tmp = self.state_dir / ".cursor.tmp"
        tmp.write_text(json.dumps({"last_id": last_id}))
        os.replace(tmp, self.state_file)

    def pulse(self) -> int:
        events = sorted((x for x in read_events(self.data_dir)
                         if isinstance(x, dict) and isinstance(x.get("id"), int)),
                        key=lambda x: x["id"])
        if not events:
            return 0
        cursor = self._cursor()
        if cursor is None:
            # Explicit owner start is the beginning of the shared session; no bulk history dump.
            self._save_cursor(events[-1]["id"])
            return 0
        pending = [x for x in events if x["id"] > cursor]
        if not pending:
            return 0
        batch = pending[:MAX_ROWS]
        lines = [line for x in batch if (line := event_summary(x, self.share_text))]
        upper = batch[-1]["id"]
        if not lines:
            self._save_cursor(upper)
            return 0
        body = ("<!-- tass-workshop:activity:v1 -->\n"
                "**Owner-enabled TASS Workshop activity** (executor events, not model self-reports)\n\n"
                "Private collaboration feed. Internal thoughts, code, memory and run output are excluded.\n\n"
                "| Event ID | Type | Observation |\n|---:|---|---|\n"
                + "\n".join(lines)
                + "\n\nThe underlying result and audit remain local; PASS is a reported executor outcome, "
                  "not an independent filesystem assertion.")
        self.outbox.mkdir(parents=True, exist_ok=True)
        rid = f"feed-{int(time.time())}-{secrets.token_hex(4)}"
        request = {"id": rid, "type": "comment", "repo": ISSUE_REPO,
                   "issue": self.issue, "body": body}
        dest = self.outbox / f"{rid}.json"
        tmp = self.outbox / f".{rid}.tmp"
        tmp.write_text(json.dumps(request, ensure_ascii=False), encoding="utf-8")
        os.replace(tmp, dest)
        self._save_cursor(upper)
        return len(lines)


def main() -> None:
    cli = argparse.ArgumentParser(description="Opt-in evidence feed; no GitHub credentials")
    cli.add_argument("--issue", type=int, required=True)
    cli.add_argument("--data", type=Path, default=Path("/workspace/data"))
    cli.add_argument("--outbox", type=Path, default=Path("/shared/outbox"))
    cli.add_argument("--state", type=Path, default=Path("/collab/state"))
    cli.add_argument("--include-conversation-text", action="store_true",
                     help="Send bounded user/TASS messages too. Do not use for private conversations.")
    cli.add_argument("--once", action="store_true", help="One pulse for testing")
    cli.add_argument("--interval", type=float, default=5)
    args = cli.parse_args()
    if not 1 <= args.interval <= 3600:
        cli.error("--interval must be 1-3600 seconds")
    share_text = (args.include_conversation_text or os.environ.get("TASS_COLLAB_INCLUDE_TEXT") == "1")
    publisher = Publisher(args.data, args.outbox, args.state, args.issue, share_text)
    print("Owner-started private GitHub activity feed; no old conversations exported.", flush=True)
    while True:
        count = publisher.pulse()
        if count:
            print(f"Queued {count} allowlisted activity records for Identity Liaison.", flush=True)
        if args.once:
            break
        time.sleep(args.interval)


if __name__ == "__main__":
    main()
