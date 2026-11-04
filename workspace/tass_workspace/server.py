"""Thought Processor workspace: talk to TASS, watch it think, build a shared Jupyter space together.

Each turn is shown as   Spark (what started it) → Trace (what TASS looked at)
→ Thought (the model's live reasoning) → Reply (+ an optional action).
Actions: OPEN a library book, or RUN a book cell / its own code (always after your approval).
"""

from __future__ import annotations

import difflib
import mimetypes
import hashlib
import json
import os
import shutil
import queue
import secrets
import subprocess
import sys
import threading
import time
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from importlib import resources
from pathlib import Path
from urllib.parse import urlparse

from .kernel import JOURNAL, LIVE_NOTEBOOK, SharedKernel
from .mind import Mind
from .thinker import Action, ModelClient, build_system, parse_reply

MAX_OPENS_PER_TURN = 2
PLAYGROUND = "Thought Playground.ipynb"
VIEW_TYPES = {".html", ".htm", ".md", ".txt", ".json", ".csv", ".py", ".png", ".jpg", ".jpeg", ".gif", ".svg",
              ".ipynb", ".log", ".yaml", ".yml", ".toml"}
MAX_TIMELINE = 600


def env(name: str, default: str) -> str:
    return os.environ.get(name, default)


class Workspace:
    def __init__(self):
        self.data = Path(env("TASS_DATA", "/workspace/data"))
        self.data.mkdir(parents=True, exist_ok=True)
        self.mind = Mind(env("TASS_MIRROR", "/shared/mirror"))
        self.model = ModelClient(env("TASS_MODEL_URL", "http://model:8081"),
                                 max_tokens=int(env("TASS_MAX_TOKENS", "3072")),
                                 thinking=env("TASS_THINKING", "1") == "1")
        # Owner controls. Autonomy = TASS may start turns on its own (mind changes, after a run).
        self.autonomy = env("TASS_AUTONOMY", env("TASS_AUTO_REFLECT", "1")) == "1"
        self.allow = {"library books": True, "projects": True, "own code": True,
                      "remember": True, "mind changes": True}
        self.outbox = Path(env("TASS_OUTBOX", "/shared/outbox"))
        self.inbox = Path(env("TASS_INBOX", "/shared/inbox"))
        self.public_url = env("TASS_PUBLIC_URL", "http://127.0.0.1:8790").rstrip("/")   # messages from GitHub, written by the Liaison
        self._origin: dict | None = None                        # GitHub issue the current turn answers
        self._turn_replies: list[str] = []
        self._turn_proposals: list[dict] = []
        self.memory_path = Path(env("TASS_SHARED", "/workspace/shared")) / "memory" / "tass-memory.md"
        self.jupyter_token = secrets.token_hex(16)
        self.jupyter_internal = env("TASS_JUPYTER_INTERNAL", "http://127.0.0.1:8888")
        self.jupyter_public = env("TASS_JUPYTER_PUBLIC", "http://127.0.0.1:8888")
        self.jupyter_proc = None
        self.kernel = SharedKernel(env("TASS_SHARED", "/workspace/shared"),
                                   self.jupyter_internal if env("TASS_JUPYTER", "1") == "1" else None,
                                   self.jupyter_token, log=self.log)
        self.timeline: list[dict] = self._load("timeline.json", [])
        self.messages: list[dict] = self._load("messages.json", [])
        self.proposals: dict[str, dict] = {p["id"]: p for p in self._load("proposals.json", [])}
        self.recent_runs: list[str] = self._load("recent_runs.json", [])
        # Older saved timelines could hold a proposal id (text) in "id"; only count real event numbers.
        self.next_id = max([e["id"] for e in self.timeline if isinstance(e.get("id"), int)] + [0]) + 1
        self.running: dict | None = None
        saved = self._load("controls.json", {})
        self.autonomy = saved.get("autonomy", self.autonomy)
        # Full auto: proposals are approved without a click. Never saved: off after every restart.
        self.full_auto = False
        self.auto_limit = int(env("TASS_AUTO_LIMIT", "8"))
        self.auto_streak = 0          # automatic approvals since Dustin's last message
        self._auto_queue: list[str] = []
        self.allow.update({k: v for k, v in saved.get("allow", {}).items() if k in self.allow})
        self.clients: list[queue.Queue] = []
        self.turn_lock = threading.Lock()
        self.busy = False
        self.kernel_error = None
        self.last_change_seen = self.mind.manifest().get("last_change_at")

    # --- persistence / events -----------------------------------------------------

    def _load(self, name, default):
        try:
            return json.loads((self.data / name).read_text("utf-8"))
        except (OSError, ValueError):
            return default

    def _save(self):
        for name, value in (("timeline.json", self.timeline[-MAX_TIMELINE:]), ("messages.json", self.messages),
                            ("proposals.json", list(self.proposals.values())), ("recent_runs.json", self.recent_runs)):
            tmp = self.data / f".{name}.tmp"
            tmp.write_text(json.dumps(value))
            tmp.replace(self.data / name)

    def audit(self, event: str, p: dict, **extra):
        """Append-only record of every proposal, decision and run (for security review)."""
        rec = {"at": time.strftime("%Y-%m-%dT%H:%M:%S%z"), "event": event, "id": p.get("id"),
               "label": p.get("label"), "source": p.get("source"),
               "code_sha256": hashlib.sha256((p.get("code") or "").encode()).hexdigest(),
               "files": p.get("file_shas"), **extra}
        with open(self.data / "audit.jsonl", "a", encoding="utf-8") as f:
            f.write(json.dumps(rec) + "\n")

    def read_audit(self, n: int = 100) -> list[dict]:
        try:
            lines = (self.data / "audit.jsonl").read_text("utf-8").splitlines()[-n:]
        except OSError:
            return []
        return [json.loads(l) for l in lines if l.strip()]

    def memory_text(self, lines: int = 60) -> str:
        try:
            body = [l for l in self.memory_path.read_text("utf-8").splitlines() if l.startswith("- ")]
        except OSError:
            return ""
        return "\n".join(body[-lines:])

    def remember(self, items: list[str]):
        if not items:
            return
        if not self.allow["remember"]:
            self.emit("notice", text="TASS wanted to remember something, but remembering is turned off.")
            return
        self.memory_path.parent.mkdir(parents=True, exist_ok=True)
        if not self.memory_path.exists():
            self.memory_path.write_text("# TASS memory\n\nThings TASS chose to remember. Dustin can edit or delete "
                                        "any line (JupyterLab: memory/tass-memory.md).\n\n")
        with open(self.memory_path, "a", encoding="utf-8") as f:
            for item in items:
                f.write(f"- {time.strftime('%Y-%m-%d')}: {item[:300]}\n")
        for item in items:
            self.emit("memory", text=item[:300])
            self.audit("remembered", {"label": "REMEMBER", "source": item[:300]})

    def identity(self) -> dict:
        m = self.mind.manifest()
        return {"name": "TASS", "repo": m.get("repo"), "branch": m.get("branch")}

    def save_controls(self):
        (self.data / "controls.json").write_text(json.dumps({"autonomy": self.autonomy, "allow": self.allow}))

    def log(self, msg: str):
        print(f"[{time.strftime('%H:%M:%S')}] {msg}", flush=True)

    def emit(self, type_: str, keep: bool = True, **data) -> dict:
        data.pop("id", None)   # "id" is the event's own number; a proposal's id travels as "pid"
        event = {**data, "id": self.next_id, "type": type_, "at": time.time()}
        self.next_id += 1
        if keep:
            self.timeline.append(event)
            self.timeline = self.timeline[-MAX_TIMELINE:]
        for q in list(self.clients):
            q.put(event)
        return event

    def status(self) -> dict:
        return {"busy": self.busy, "model_ok": self.model.health(), "thinking": self.model.thinking,
                "autonomy": self.autonomy, "allow": self.allow, "full_auto": self.full_auto,
                "auto_streak": self.auto_streak, "auto_limit": self.auto_limit, "identity": self.identity(),
                "memory": self.memory_text(20),
                "running": {"id": self.running["id"], "label": self.running["label"]} if self.running else None,
                "kernel": self.kernel.mode, "kernel_error": self.kernel_error,
                "jupyter": f"{self.jupyter_public}/lab/tree/{LIVE_NOTEBOOK}?token={self.jupyter_token}"
                           if self.jupyter_proc else None,
                "journal": JOURNAL, "mind": self.mind.status(), "listing": self.kernel.listing(200),
                "files_url": f"{self.public_url}/files/",
                "jupyter_tree": f"{self.jupyter_public}/lab/tree/{{path}}?token={self.jupyter_token}"
                                if self.jupyter_proc else None,
                "playground": PLAYGROUND if (self.kernel.shared / PLAYGROUND).exists() else None}

    # --- startup ----------------------------------------------------------------------

    def start_background(self):
        if env("TASS_JUPYTER", "1") == "1":
            shared = Path(env("TASS_SHARED", "/workspace/shared"))
            shared.mkdir(parents=True, exist_ok=True)
            port = urlparse(self.jupyter_internal).port or 8888
            cmd = [sys.executable, "-m", "jupyter", "lab", "--no-browser", "--ip=0.0.0.0", f"--port={port}",
                   f"--ServerApp.token={self.jupyter_token}", f"--ServerApp.root_dir={shared}",
                   "--ServerApp.allow_remote_access=True"]
            if os.geteuid() == 0:
                cmd.append("--allow-root")
            log = open(self.data / "jupyter.log", "ab")
            self.jupyter_proc = subprocess.Popen(cmd, stdout=log, stderr=subprocess.STDOUT)
        threading.Thread(target=self._start_kernel, daemon=True).start()
        threading.Thread(target=self._watch_mind, daemon=True).start()

    def _start_kernel(self):
        try:
            self.kernel.start()
            self.kernel_error = None
            self.emit("notice", text=f"Python kernel ready ({self.kernel.mode}).")
        except Exception as e:  # noqa: BLE001
            self.kernel_error = str(e)
            self.emit("error", text=f"Kernel failed to start: {e}")

    def _watch_mind(self):
        while True:
            time.sleep(3)
            try:
                self._check_outbox()
            except Exception as e:  # noqa: BLE001
                self.log(f"outbox check failed: {e}")
            try:
                self._check_inbox()
            except Exception as e:  # noqa: BLE001
                self.log(f"inbox check failed: {e}")
            m = self.mind.manifest()
            changed_at = m.get("last_change_at")
            if not changed_at or changed_at == self.last_change_seen:
                continue
            first_sync = self.last_change_seen is None
            self.last_change_seen = changed_at
            files = m.get("last_changed", [])
            self.emit("spark", kind="repo", text=("First sync from GitHub: " if first_sync else
                                                  "Changed on GitHub: ") + ", ".join(files[:8]))
            mind_changed = [f for f in files if f.startswith("mind/")]
            if mind_changed and not first_sync and self.autonomy and not self.busy:
                threading.Thread(target=self.turn, args=(
                    "mind", f"[Spark] Your mind files just changed on GitHub: {', '.join(mind_changed)}. "
                            "In one or two sentences, tell Dustin what is different for you now."),
                    daemon=True).start()

    # --- a turn ---------------------------------------------------------------------

    def _fit(self, system: str) -> tuple[list[dict], int, int]:
        history = list(self.messages)
        while True:
            msgs = [{"role": "system", "content": system}] + history
            used, ctx = self.model.measure(msgs)
            if used + 1024 + self.model.margin <= ctx or len(history) <= 2:
                return msgs, used, ctx
            history = history[2:] if len(history) > 3 and history[2]["role"] == "user" else history[1:]

    def turn(self, kind: str, text: str, origin: dict | None = None, comment_prefix: str = "",
             *, chat_only: bool = False, request_id: str | None = None, claimed: bool = False):
        # A user message never waits behind a running turn; follow-ups (results, mind changes) do, briefly.
        if not claimed and not self.turn_lock.acquire(blocking=kind != "user",
                                                       timeout=-1 if kind == "user" else 60):
            self.emit("notice", text="TASS is still thinking; try again in a moment.")
            return
        try:
            self.busy = True
            self.emit("busy", keep=False, busy=True)
            self._origin, self._turn_replies, self._turn_proposals = origin, [], []
            if kind == "user":
                self.emit("user", text=text, request_id=request_id, chat_only=chat_only)
            if kind == "github":
                self.emit("user", text=text, via=origin.get("url"), issue=f"{origin['repo']}#{origin['issue']}")
                self.messages.append({"role": "user", "content":
                                      f"[Dustin, on GitHub {origin['repo']}#{origin['issue']}] {text}"})
                self.emit("spark", kind="github", text=f"Message on GitHub {origin['repo']}#{origin['issue']}")
            else:
                self.messages.append({"role": "user", "content": text})
                self.emit("spark", kind=kind, text=text if kind != "user" else "Your message")
            opened: list[str] = []
            for _ in range(MAX_OPENS_PER_TURN + 1):
                mind_files = self.mind.mind_files()
                catalog = self.mind.catalog()
                listing = self.kernel.listing()
                projects = self.mind.projects()
                ident = self.identity()
                system = build_system(mind_files, catalog, listing, self.recent_runs[-3:], projects, self.allow,
                                      identity=f"{ident['repo']}@{ident['branch']}" if ident["repo"] else "",
                                      memory=self.memory_text())
                system += (f"\n\nLINKS: Dustin can open any file in the shared folder in his browser at "
                           f"{self.public_url}/files/<path> (for example {self.public_url}/files/notes/). "
                           "When he asks for a link, give that full address, and only for a file that is in the "
                           "SHARED FOLDER list above. If a file isn't there yet, say so; don't claim you made it.")
                if chat_only:
                    system += ("\n\nCHAT-ONLY TURN: Answer in plain text. Do not issue OPEN, RUN, "
                               "CHANGE MIND or REMEMBER instructions. Do not claim execution, "
                               "file edits, memory writes or GitHub changes. Any action emitted "
                               "will be ignored by the server.")
                msgs, used, ctx = self._fit(system)
                self.emit("trace", mind=[p for p, _ in mind_files], books=len(catalog), projects=len(projects),
                          opened=opened,
                          files=len(listing), history=len(msgs) - 1, tokens=used, context=ctx)
                self.emit("thought_start", keep=False)
                out = self.model.stream(msgs, lambda d: self.emit("thought_delta", keep=False, delta=d),
                                        lambda d: self.emit("reply_delta", keep=False, delta=d))
                if not out["text"].strip():
                    self.emit("notice", text="The thinking ran long without an answer; asking again without thinking.")
                    out2 = self.model.stream(msgs, lambda d: None,
                                             lambda d: self.emit("reply_delta", keep=False, delta=d), thinking=False)
                    out2["thought"] = out["thought"]
                    out = out2
                shown, action, memories = parse_reply(out["text"])
                self.emit("thought", text=out["thought"], seconds=out["seconds"])
                if chat_only:
                    # Full Auto must never authorize action or memory output from Atrium chat.
                    if action or memories:
                        self.emit("notice", text="Chat-only: model action/memory instructions ignored.",
                                  request_id=request_id)
                    action, memories = None, []
                    if not shown:
                        self.emit("error", text="Model provided no conversational reply; no action was taken.",
                                  request_id=request_id)
                        break
                if shown:
                    self._turn_replies.append(shown)
                self.emit("reply", text=shown, tokens=out["prompt_tokens"], context=out["context"],
                          action=action.label() if action else None, request_id=request_id)
                self.messages.append({"role": "assistant", "content": shown if chat_only else out["text"].strip()})
                self.remember(memories)
                if action and action.kind == "open" and action.project:
                    if len(opened) >= MAX_OPENS_PER_TURN:
                        break
                    try:
                        proj = self.mind.project(action.project)
                        code = self.mind.entry_code(proj)
                        opened.append(proj.path)
                        self.messages.append({"role": "user", "content":
                                              f"[PROJECT {proj.path} — you opened it]\n{proj.about.strip()[:3000]}\n\n"
                                              f"FILES: {', '.join(proj.files)}\nENTRY: {proj.entry}\n"
                                              f"```python\n{code[:4000]}\n```\n"
                                              "[Now continue: explain what it does and whether to RUN it.]"})
                        self.emit("book", path=proj.path, title=proj.title, cells=len(proj.files), kind="project")
                        continue
                    except ValueError as e:
                        self.messages.append({"role": "user", "content": f"[Could not open: {e}]"})
                        self.emit("notice", text=str(e))
                        continue
                if action and action.kind == "open":
                    if len(opened) >= MAX_OPENS_PER_TURN:
                        break
                    try:
                        book = self.mind.book(action.book)
                        opened.append(book.path)
                        self.messages.append({"role": "user", "content":
                                              f"[BOOK {book.path} — you opened it]\n{self.mind.numbered(book)}\n"
                                              "[Now continue: explain what you'd do with it, and RUN a cell if you want.]"})
                        self.emit("book", path=book.path, title=book.title, cells=len(book.cells))
                        continue
                    except ValueError as e:
                        self.messages.append({"role": "user", "content": f"[Could not open: {e}]"})
                        self.emit("notice", text=str(e))
                        continue
                if action and action.kind in ("run", "mind"):
                    self._propose(action)
                break
        except Exception as e:  # noqa: BLE001
            self.emit("error", text=str(e), request_id=request_id)
        finally:
            if origin:
                try:
                    self._reply_on_github(origin, comment_prefix)
                except Exception as e:  # noqa: BLE001
                    self.log(f"could not queue GitHub reply: {e}")
            self._origin = None
            self.busy = False
            self.emit("busy", keep=False, busy=False)
            self._save()
            self.turn_lock.release()
            self._drain_auto()

    # --- proposals ------------------------------------------------------------------

    def _refuse(self, what: str, label: str):
        msg = f"{label} not proposed: Dustin has turned off {what}."
        self.messages.append({"role": "user", "content": f"[{msg}]"})
        self.emit("notice", text=msg)
        self.audit("refused", {"label": label, "source": what})

    def _propose(self, action: Action):
        p: dict = {"id": secrets.token_hex(4), "label": action.label(), "status": "pending", "at": time.time()}
        try:
            if action.kind == "mind":
                if not self.allow["mind changes"]:
                    return self._refuse("mind changes", p["label"])
                man = self.mind.manifest()
                rec = man.get("files", {}).get(action.path, {})
                repo, branch = rec.get("repo") or man.get("repo"), rec.get("branch") or man.get("branch")
                if not repo:
                    raise ValueError("Mind isn't synced from GitHub yet, so there's no repo to propose a change to.")
                cur_path = self.mind.files / action.path
                current = cur_path.read_text("utf-8") if cur_path.is_file() else ""
                if action.edits is not None:
                    if not current:
                        raise ValueError(f"{action.path} doesn't exist yet; a new mind file needs its whole text")
                    from .edits import EditError, apply_edits
                    try:
                        action.content = apply_edits(current, action.edits)
                    except EditError as e:
                        raise ValueError(str(e)) from None
                elif current and len(action.content) < 0.6 * len(current):
                    # A small model rewriting a whole file tends to drop most of it (seen in testing).
                    raise ValueError(f"That would remove most of {action.path} ({len(action.content)} of "
                                     f"{len(current)} characters left), so it was not proposed. "
                                     "Describe one edit with FIND/ACTION/TEXT instead.")
                if current.strip() == action.content.strip():
                    raise ValueError(f"{action.path}: the change doesn't alter anything (maybe it's already there)")
                diff = "".join(difflib.unified_diff(current.splitlines(True), action.content.splitlines(True),
                                                    f"a/{action.path}", f"b/{action.path}"))
                p.update(kind="mind", path=action.path, repo=repo, branch=branch, content=action.content,
                         diff=diff, code=action.content, new_file=not current,
                         source=f"{action.path} in {repo}@{branch} → pull request (you merge)")
            elif action.project:
                if not self.allow["projects"]:
                    return self._refuse("projects", p["label"])
                proj = self.mind.project(action.project)
                if not proj.entry:
                    raise ValueError(f"{proj.path} has no entry file (.py or .ipynb) to run")
                shas = self.mind.manifest().get("files", {})
                dest = f"projects/{proj.name}"
                p.update(kind="project", project=proj.name, entry=proj.entry, files=proj.files,
                         file_shas={f: shas.get(f"{proj.path}/{f}", {}).get("sha") for f in proj.files},
                         source=f"{proj.path} → copy {len(proj.files)} file(s) into shared/{dest}/, then run {proj.entry}",
                         code=self.mind.entry_code(proj))
            elif action.book:
                if not self.allow["library books"]:
                    return self._refuse("library books", p["label"])
                book = self.mind.book(action.book)
                if not 1 <= action.cell <= len(book.cells):
                    raise ValueError(f"{book.path} has {len(book.cells)} cells; there is no cell {action.cell}")
                sha = self.mind.manifest().get("files", {}).get(book.path, {}).get("sha")
                p.update(kind="book", code=book.cells[action.cell - 1], source=f"{book.path} · cell {action.cell}",
                         file_shas={book.path: sha})
            else:
                if not self.allow["own code"]:
                    return self._refuse("own code", p["label"])
                p.update(kind="code", code=action.code, source="TASS's own code")
        except ValueError as e:
            self.messages.append({"role": "user", "content": f"[Could not propose: {e}]"})
            self.emit("notice", text=str(e))
            return
        if self._origin:
            p["origin"] = self._origin
        self._turn_proposals.append(p)
        self.proposals[p["id"]] = p
        self.audit("proposed", p)
        self.emit("proposal", proposal=p)
        if self.full_auto:
            if self.auto_streak >= self.auto_limit:
                self.emit("notice", text=f"Full auto paused: {self.auto_limit} automatic runs in a row. "
                                         "Approve this one yourself, or send a message to continue.")
                self.audit("auto-paused", p)
            else:
                self._auto_queue.append(p["id"])   # approved right after this turn finishes

    def _drain_auto(self):
        while self._auto_queue:
            pid = self._auto_queue.pop(0)
            p = self.proposals.get(pid)
            if not p or p["status"] != "pending" or not self.full_auto:
                continue
            self.auto_streak += 1
            p["auto"] = True
            self.audit("auto-approved", p, streak=self.auto_streak)
            try:
                self.decide(pid, "approve")
            except ValueError as e:
                self.emit("notice", text=f"Full auto couldn't approve {p['label']}: {e}")

    def decide(self, pid: str, decision: str, via: str = "workspace") -> dict:
        p = self.proposals.get(pid)
        if not p or p["status"] != "pending":
            raise ValueError("That proposal is not waiting for a decision.")
        p["decided_via"] = via
        if decision == "reject":
            p["status"] = "rejected"
            self.audit("rejected", p, via=via)
            self.messages.append({"role": "user", "content": f"[Dustin rejected {p['label']}.]"})
            self.emit("decision", pid=pid, status="rejected")
            self._save()
            return p
        if p.get("kind") == "mind":
            return self._send_mind(p)
        if self.running:
            raise ValueError(f"Wait: {self.running['label']} is still running (or press Stop).")
        p["status"] = "running"
        self.running = p
        self.audit("approved", p, via=via)
        self.emit("decision", pid=pid, status="running")
        threading.Thread(target=self._run, args=(p,), daemon=True).start()
        return p

    def _send_mind(self, p: dict) -> dict:
        """Owner approved a mind change: hand it to the Liaison (which has the token) as a PR request."""
        self.outbox.mkdir(parents=True, exist_ok=True)
        req = {"id": p["id"], "repo": p["repo"], "branch": p["branch"], "path": p["path"], "content": p["content"],
               "title": f"TASS: update {p['path']}",
               "summary": f"TASS proposed this change in the workspace; Dustin approved it at {time.strftime('%H:%M')}.",
               "approved_at": time.strftime("%Y-%m-%dT%H:%M:%S%z")}
        tmp = self.outbox / f".{p['id']}.tmp"
        tmp.write_text(json.dumps(req))
        tmp.replace(self.outbox / f"{p['id']}.json")
        p["status"] = "sent"
        self.audit("approved", p)
        self.emit("decision", pid=p["id"], status="sent")
        self.emit("notice", text=f"Sent to the Liaison: it will open a pull request for {p['path']} (usually within 30 s).")
        self._save()
        return p

    def _outbox_write(self, req: dict) -> str:
        rid = req.get("id") or secrets.token_hex(4)
        req["id"] = rid
        self.outbox.mkdir(parents=True, exist_ok=True)
        tmp = self.outbox / f".{rid}.tmp"
        tmp.write_text(json.dumps(req))
        tmp.replace(self.outbox / f"{rid}.json")
        return rid

    def _reply_on_github(self, origin: dict, prefix: str = ""):
        parts = [prefix] if prefix else []
        if self._turn_replies:
            parts.append("**TASS:** " + "\n\n".join(self._turn_replies))
        for p in self._turn_proposals:
            if p.get("kind") == "mind":
                shown = f"```diff\n{p['diff'][:3000]}\n```"
            else:
                files = f"Files: {', '.join(p['files'])}\n" if p.get("files") else ""
                shown = f"{files}```python\n{(p.get('code') or '')[:3000]}\n```"
            how = ("Full auto is on, so it runs now." if self.full_auto and p["status"] == "pending"
                   else f"Reply **`/approve`** to run it, or **`/reject`** (or use the workspace page). Id: `{p['id']}`")
            parts.append(f"**Proposed:** `{p['label']}` — {p.get('source', '')}\n{shown}\n{how}")
        if not parts:
            parts.append("**TASS:** (no reply text)")
        self._outbox_write({"type": "comment", "repo": origin["repo"], "issue": origin["issue"],
                            "body": "\n\n".join(parts)})
        self.audit("github_reply", {"label": f"{origin['repo']}#{origin['issue']}"})

    def _check_inbox(self):
        if not self.inbox.is_dir():
            return
        for f in sorted(self.inbox.glob("*.json"), key=lambda x: x.stat().st_mtime):
            if f.name.startswith(".") or f.name.endswith(".done.json"):
                continue
            try:
                msg = json.loads(f.read_text("utf-8"))
            except (OSError, ValueError):
                continue
            if self.busy or self.running:
                return                       # try again on the next check, in order
            origin = {"repo": msg["repo"], "issue": msg["issue"], "url": msg.get("url")}
            f.replace(f.with_name(f.name[:-5] + ".done.json"))
            text = (msg.get("text") or "").strip()
            cmd = text.split()[0].lower() if text else ""
            if cmd in ("/approve", "/reject"):
                self._github_decision(origin, cmd[1:], text.split()[1] if len(text.split()) > 1 else None)
                continue
            self.audit("github_message", {"label": f"{origin['repo']}#{origin['issue']}", "source": text[:200]})
            self.auto_streak = 0
            self.busy = True   # claim the turn now so the next check waits for this one
            threading.Thread(target=self.turn, args=("github", text), kwargs={"origin": origin}, daemon=True).start()
            return

    def _github_decision(self, origin: dict, decision: str, pid: str | None):
        waiting = [p for p in self.proposals.values() if p["status"] == "pending"
                   and (pid == p["id"] if pid else (p.get("origin") or {}).get("issue") == origin["issue"])]
        if not waiting:
            body = "Nothing from this issue is waiting for a decision." if not pid else f"No pending proposal `{pid}`."
        else:
            p = waiting[-1]
            try:
                self.decide(p["id"], "approve" if decision == "approve" else "reject", via="github")
                body = (f"✓ Approved `{p['label']}` from GitHub; running now. The result will be posted here."
                        if decision == "approve" else f"Rejected `{p['label']}`. TASS will see that.")
                self.emit("notice", text=f"{decision.title()}d from GitHub: {p['label']}")
            except ValueError as e:
                body = f"Couldn't {decision} `{p['label']}`: {e}"
        self._outbox_write({"type": "comment", "repo": origin["repo"], "issue": origin["issue"], "body": body})

    def _check_outbox(self):
        for p in self.proposals.values():
            if p.get("kind") != "mind" or p["status"] != "sent":
                continue
            res_path = self.outbox / f"{p['id']}.result.json"
            if not res_path.exists():
                continue
            try:
                res = json.loads(res_path.read_text("utf-8"))
            except (OSError, ValueError):
                continue
            if res.get("ok"):
                p["status"], p["pr_url"] = "pr opened", res["pr_url"]
                self.audit("pr_opened", p, pr_url=res["pr_url"])
                self.messages.append({"role": "user", "content":
                                      f"[Your change to {p['path']} is now a pull request: {res['pr_url']}. "
                                      "You'll think with it once Dustin merges it.]"})
            else:
                p["status"] = "pr failed"
                self.audit("pr_failed", p, error=res.get("error"))
                self.messages.append({"role": "user", "content": f"[Your change to {p['path']} could not become a "
                                      f"pull request: {res.get('error')}]"})
            self.emit("mind_pr", pid=p["id"], path=p["path"], ok=bool(res.get("ok")), url=res.get("pr_url"),
                      error=res.get("error"))
            self._save()

    def _install_project(self, p: dict) -> str:
        """Copy the mirrored project into shared/projects/<name>/ and return the code that runs its entry."""
        src = self.mind.files / "projects" / p["project"]
        dest = self.kernel.shared / "projects" / p["project"]
        dest.mkdir(parents=True, exist_ok=True)
        for rel in p["files"]:
            target = dest / rel
            target.parent.mkdir(parents=True, exist_ok=True)
            shutil.copyfile(src / rel, target)
        entry = p["entry"]
        if entry.endswith(".ipynb"):
            body = f"exec(compile({p['code']!r}, {entry!r}, 'exec'), globals())"
        else:
            body = f"runpy.run_path({entry!r}, run_name='__main__')"
        return (f"import os as _os, runpy\n_back = _os.getcwd()\n_os.chdir({str(dest)!r})\n"
                f"try:\n    {body}\nfinally:\n    _os.chdir(_back)\n")

    def _run(self, p: dict):
        try:
            code = self._install_project(p) if p.get("kind") == "project" else p["code"]
            r = self.kernel.execute(code)
        except Exception as e:  # noqa: BLE001
            r = {"ok": False, "text": "", "images": [], "error": str(e), "seconds": 0}
        finally:
            self.running = None
        p["status"] = "ran" if r["ok"] else "failed"
        self.audit(p["status"], p, seconds=r["seconds"], error=r["error"])
        text = r["text"][-3000:]
        self.emit("result", pid=p["id"], label=p["label"], ok=r["ok"], text=text, images=r["images"][:4],
                  error=r["error"], seconds=r["seconds"])
        summary = f"{p['label']}: {'ok' if r['ok'] else 'ERROR ' + str(r['error'])}"
        self.recent_runs = (self.recent_runs + [summary])[-10:]
        self.messages.append({"role": "user", "content":
                              f"[RESULT of {p['label']} — {'ok' if r['ok'] else 'error'}, {r['seconds']} s]\n"
                              f"{text[-1500:] or '(no output)'}"
                              + (f"\nERROR: {r['error']}" if r["error"] else "")
                              + ("\n(It also produced an image.)" if r["images"] else "")})
        self._save()
        if self.autonomy:
            block = (f"**Result of `{p['label']}`:** {'✓ ok' if r['ok'] else '✗ ' + str(r['error'])} ({r['seconds']} s)\n"
                     f"```\n{(text[-1500:] or '(no output)')}\n```\n") if p.get("origin") else ""
            self.turn("result", f"[Spark] The run finished: {summary}", origin=p.get("origin"), comment_prefix=block)
        else:
            self.emit("notice", text="Autonomy is off, so TASS waits for you before reacting to this result.")
            if p.get("origin"):
                self._outbox_write({"type": "comment", "repo": p["origin"]["repo"], "issue": p["origin"]["issue"],
                                    "body": f"**Result of `{p['label']}`:** {'✓ ok' if r['ok'] else '✗ ' + str(r['error'])}"
                                            f"\n```\n{(text[-1500:] or '(no output)')}\n```"})

    def stop_run(self):
        if not self.running:
            raise ValueError("Nothing is running.")
        self.kernel._interrupt()
        self.audit("stopped", self.running)
        self.emit("notice", text=f"Stop requested for {self.running['label']}.")


# --- HTTP ---------------------------------------------------------------------------

def make_handler(ws: Workspace):
    page = resources.files(__package__).joinpath("static/index.html").read_bytes()

    class H(BaseHTTPRequestHandler):
        protocol_version = "HTTP/1.1"

        def log_message(self, *a):
            pass

        def _json(self, code: int, obj):
            b = json.dumps(obj).encode()
            self.send_response(code)
            self.send_header("Content-Type", "application/json")
            self.send_header("Content-Length", str(len(b)))
            self.send_header("Cache-Control", "no-store")
            self.end_headers()
            self.wfile.write(b)

        def _file(self, raw: str):
            """Serve the shared folder read-only. Pages run in a sandbox (no access to this app's controls)."""
            from urllib.parse import unquote
            rel = unquote(raw).lstrip("/")
            root = ws.kernel.shared.resolve()
            target = (root / rel).resolve()
            if (root not in target.parents and target != root) or any(part.startswith(".") for part in Path(rel).parts):
                return self._json(404, {"error": "not found"})
            if target.is_dir():
                items = sorted(target.iterdir(), key=lambda x: (not x.is_dir(), x.name.lower()))
                from html import escape
                rows = "".join(
                    f'<li><a href="{escape(p.name)}{"/" if p.is_dir() else ""}">{escape(p.name)}{"/" if p.is_dir() else ""}</a></li>'
                    for p in items if not p.name.startswith("."))
                body = (f"<!doctype html><meta charset=utf-8><title>shared/{escape(rel)}</title>"
                        f"<style>body{{font:15px system-ui;background:#0f1720;color:#e6edf3;padding:20px}}a{{color:#5fb3f9}}</style>"
                        f"<h3>shared/{escape(rel)}</h3><ul>{rows or '<li>(empty)</li>'}</ul>").encode()
                if not raw.endswith("/") and rel:
                    self.send_response(301)
                    self.send_header("Location", f"/files/{rel}/")
                    self.send_header("Content-Length", "0")
                    self.end_headers()
                    return
                ctype = "text/html; charset=utf-8"
            elif target.is_file() and target.suffix.lower() in VIEW_TYPES:
                body = target.read_bytes()[:20 * 1024 * 1024]
                if target.suffix.lower() in (".html", ".htm", ".svg"):
                    ctype = mimetypes.guess_type(target.name)[0] or "text/html"
                elif target.suffix.lower() in (".png", ".jpg", ".jpeg", ".gif"):
                    ctype = mimetypes.guess_type(target.name)[0]
                else:
                    ctype = "text/plain; charset=utf-8"   # md, py, json, ipynb… shown as text
            else:
                return self._json(404, {"error": "not found or not a viewable type"})
            self.send_response(200)
            self.send_header("Content-Type", ctype)
            self.send_header("Content-Length", str(len(body)))
            self.send_header("Cache-Control", "no-store")
            self.send_header("X-Content-Type-Options", "nosniff")
            # Opaque origin: scripts in TASS-made pages can't call /api/* as this app.
            self.send_header("Content-Security-Policy", "sandbox allow-scripts allow-popups allow-forms")
            self.end_headers()
            self.wfile.write(body)

        def do_GET(self):
            path = urlparse(self.path).path
            if path == "/":
                self.send_response(200)
                self.send_header("Content-Type", "text/html; charset=utf-8")
                self.send_header("Content-Length", str(len(page)))
                self.end_headers()
                self.wfile.write(page)
            elif path == "/api/state":
                self._json(200, {"timeline": ws.timeline[-300:], "status": ws.status(),
                                 "proposals": [p for p in ws.proposals.values() if p["status"] == "pending"]})
            elif path == "/files" or path.startswith("/files/"):
                self._file(urlparse(self.path).path[len("/files"):])
            elif path == "/api/audit":
                self._json(200, ws.read_audit())
            elif path == "/api/status":
                self._json(200, ws.status())
            elif path == "/api/events":
                self.send_response(200)
                self.send_header("Content-Type", "text/event-stream")
                self.send_header("Cache-Control", "no-store")
                self.send_header("Connection", "keep-alive")
                self.end_headers()
                q: queue.Queue = queue.Queue()
                ws.clients.append(q)
                try:
                    while True:
                        try:
                            ev = q.get(timeout=15)
                            self.wfile.write(f"data: {json.dumps(ev)}\n\n".encode())
                        except queue.Empty:
                            self.wfile.write(b": ping\n\n")
                        self.wfile.flush()
                except (BrokenPipeError, ConnectionResetError, OSError):
                    pass
                finally:
                    ws.clients.remove(q)
            else:
                self._json(404, {"error": "not found"})

        def do_POST(self):
            # Only this page may post: JSON body and a local (or missing) Origin.
            origin = self.headers.get("Origin")
            if origin and urlparse(origin).hostname not in ("127.0.0.1", "localhost"):
                return self._json(403, {"error": "forbidden origin"})
            if not (self.headers.get("Content-Type") or "").startswith("application/json"):
                return self._json(415, {"error": "JSON only"})
            try:
                n = int(self.headers.get("Content-Length") or 0)
                body = json.loads(self.rfile.read(n) or b"{}") if 0 <= n <= 65536 else {}
            except ValueError:
                return self._json(400, {"error": "bad JSON"})
            path = urlparse(self.path).path
            try:
                if path == "/api/message":
                    text = body.get("text")
                    if not isinstance(text, str) or not text.strip() or len(text) > 4000:
                        return self._json(400, {"error": "text required (max 4000 characters)"})
                    # Reserve the inference lock before acknowledging. This also closes
                    # the check-then-start race across simultaneous HTTP requests.
                    if ws.busy or not ws.turn_lock.acquire(blocking=False):
                        return self._json(409, {"error": "TASS is thinking; message was not accepted"})
                    request_id = secrets.token_hex(8)
                    try:
                        ws.busy = True
                        ws.auto_streak = 0  # Owner spoke; modes remain unchanged.
                        threading.Thread(target=ws.turn, args=("user", text.strip()),
                                         kwargs={"chat_only": body.get("chat_only") is True,
                                                 "request_id": request_id, "claimed": True},
                                         daemon=True).start()
                    except Exception:
                        ws.busy = False
                        ws.turn_lock.release()
                        raise
                    return self._json(202, {"ok": True, "request_id": request_id, "status": "accepted"})
                if path == "/api/proposal":
                    return self._json(200, ws.decide(str(body.get("id")), str(body.get("decision"))))
                if path == "/api/settings":
                    if "thinking" in body:
                        ws.model.thinking = bool(body["thinking"])
                    if "full_auto" in body:
                        ws.full_auto = bool(body["full_auto"])
                        ws.auto_streak = 0
                        ws.audit("full_auto_on" if ws.full_auto else "full_auto_off", {"label": "settings"})
                        ws.emit("notice", text="FULL AUTO is ON: TASS's proposals run without a click "
                                               f"(pauses after {ws.auto_limit} in a row; Stop still works)."
                                if ws.full_auto else "Full auto is off: every run needs your click again.")
                        if ws.full_auto:
                            for p in ws.proposals.values():
                                if p["status"] == "pending":
                                    ws._auto_queue.append(p["id"])
                            threading.Thread(target=ws._drain_auto, daemon=True).start()
                    if "autonomy" in body:
                        ws.autonomy = bool(body["autonomy"])
                        ws.emit("notice", text=f"Autonomy {'on' if ws.autonomy else 'off'}.")
                    for k, v in (body.get("allow") or {}).items():
                        if k in ws.allow:
                            ws.allow[k] = bool(v)
                            ws.emit("notice", text=f"{k}: {'allowed' if v else 'turned off'}.")
                    ws.save_controls()
                    return self._json(200, ws.status())
                if path == "/api/stop":
                    ws.stop_run()
                    return self._json(200, {"ok": True})
                if path == "/api/kernel/restart":
                    ws.kernel.restart()
                    ws.emit("notice", text="Python kernel restarted (variables cleared; files kept).")
                    return self._json(200, {"ok": True})
                if path == "/api/reset":
                    if ws.busy:
                        return self._json(409, {"error": "TASS is thinking"})
                    ws.messages.clear()
                    ws.timeline.clear()
                    ws.proposals.clear()
                    ws._save()
                    ws.emit("notice", text="Conversation cleared. The shared folder and kernel were kept.")
                    return self._json(200, {"ok": True})
            except ValueError as e:
                return self._json(400, {"error": str(e)})
            return self._json(404, {"error": "not found"})

    return H


def main():
    ws = Workspace()
    ws.start_background()
    port = int(env("TASS_PORT", "8790"))
    srv = ThreadingHTTPServer((env("TASS_HOST", "0.0.0.0"), port), make_handler(ws))
    srv.daemon_threads = True
    ws.log(f"Thought Processor workspace on port {port}; model {ws.model.url}; mirror {ws.mind.root}")
    try:
        srv.serve_forever()
    except KeyboardInterrupt:
        pass
    finally:
        if ws.jupyter_proc:
            ws.jupyter_proc.terminate()


if __name__ == "__main__":
    main()
