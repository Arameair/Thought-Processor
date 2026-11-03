"""Thought Processor workspace: talk to TASS, watch it think, build a shared Jupyter space together.

Each turn is shown as   Spark (what started it) → Trace (what TASS looked at)
→ Thought (the model's live reasoning) → Reply (+ an optional action).
Actions: OPEN a library book, or RUN a book cell / its own code (always after your approval).
"""

from __future__ import annotations

import json
import os
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
from .thinker import Action, ModelClient, build_system, parse_action

MAX_OPENS_PER_TURN = 2
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
        self.auto_reflect = env("TASS_AUTO_REFLECT", "1") == "1"
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
        self.next_id = max([e.get("id", 0) for e in self.timeline] + [0]) + 1
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

    def log(self, msg: str):
        print(f"[{time.strftime('%H:%M:%S')}] {msg}", flush=True)

    def emit(self, type_: str, keep: bool = True, **data) -> dict:
        event = {"id": self.next_id, "type": type_, "at": time.time(), **data}
        self.next_id += 1
        if keep:
            self.timeline.append(event)
            self.timeline = self.timeline[-MAX_TIMELINE:]
        for q in list(self.clients):
            q.put(event)
        return event

    def status(self) -> dict:
        return {"busy": self.busy, "model_ok": self.model.health(), "thinking": self.model.thinking,
                "auto_reflect": self.auto_reflect, "kernel": self.kernel.mode, "kernel_error": self.kernel_error,
                "jupyter": f"{self.jupyter_public}/lab/tree/{LIVE_NOTEBOOK}?token={self.jupyter_token}"
                           if self.jupyter_proc else None,
                "journal": JOURNAL, "mind": self.mind.status(), "listing": self.kernel.listing()}

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
            if mind_changed and not first_sync and self.auto_reflect and not self.busy:
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

    def turn(self, kind: str, text: str):
        if not self.turn_lock.acquire(blocking=False):
            self.emit("notice", text="TASS is still thinking; try again in a moment.")
            return
        try:
            self.busy = True
            self.emit("busy", keep=False, busy=True)
            if kind == "user":
                self.emit("user", text=text)
            self.messages.append({"role": "user", "content": text})
            self.emit("spark", kind=kind, text=text if kind != "user" else "Your message")
            opened: list[str] = []
            for _ in range(MAX_OPENS_PER_TURN + 1):
                mind_files = self.mind.mind_files()
                catalog = self.mind.catalog()
                listing = self.kernel.listing()
                system = build_system(mind_files, catalog, listing, self.recent_runs[-3:])
                msgs, used, ctx = self._fit(system)
                self.emit("trace", mind=[p for p, _ in mind_files], books=len(catalog), opened=opened,
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
                shown, action = parse_action(out["text"])
                self.emit("thought", text=out["thought"], seconds=out["seconds"])
                self.emit("reply", text=shown, tokens=out["prompt_tokens"], context=out["context"],
                          action=action.label() if action else None)
                self.messages.append({"role": "assistant", "content": out["text"].strip()})
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
                if action and action.kind == "run":
                    self._propose(action)
                break
        except Exception as e:  # noqa: BLE001
            self.emit("error", text=str(e))
        finally:
            self.busy = False
            self.emit("busy", keep=False, busy=False)
            self._save()
            self.turn_lock.release()

    # --- proposals ------------------------------------------------------------------

    def _propose(self, action: Action):
        code, source = action.code, "TASS's own code"
        if action.book:
            try:
                book = self.mind.book(action.book)
                if not 1 <= action.cell <= len(book.cells):
                    raise ValueError(f"{book.path} has {len(book.cells)} cells; there is no cell {action.cell}")
                code, source = book.cells[action.cell - 1], f"{book.path} · cell {action.cell}"
            except ValueError as e:
                self.messages.append({"role": "user", "content": f"[Could not propose: {e}]"})
                self.emit("notice", text=str(e))
                return
        pid = secrets.token_hex(4)
        p = {"id": pid, "label": action.label(), "source": source, "code": code, "status": "pending",
             "at": time.time()}
        self.proposals[pid] = p
        self.emit("proposal", proposal=p)

    def decide(self, pid: str, decision: str) -> dict:
        p = self.proposals.get(pid)
        if not p or p["status"] != "pending":
            raise ValueError("That proposal is not waiting for a decision.")
        if decision == "reject":
            p["status"] = "rejected"
            self.messages.append({"role": "user", "content": f"[Dustin rejected {p['label']}.]"})
            self.emit("decision", id=pid, status="rejected")
            self._save()
            return p
        p["status"] = "running"
        self.emit("decision", id=pid, status="running")
        threading.Thread(target=self._run, args=(p,), daemon=True).start()
        return p

    def _run(self, p: dict):
        try:
            r = self.kernel.execute(p["code"])
        except Exception as e:  # noqa: BLE001
            r = {"ok": False, "text": "", "images": [], "error": str(e), "seconds": 0}
        p["status"] = "ran" if r["ok"] else "failed"
        text = r["text"][-3000:]
        self.emit("result", id=p["id"], label=p["label"], ok=r["ok"], text=text, images=r["images"][:4],
                  error=r["error"], seconds=r["seconds"])
        summary = f"{p['label']}: {'ok' if r['ok'] else 'ERROR ' + str(r['error'])}"
        self.recent_runs = (self.recent_runs + [summary])[-10:]
        self.messages.append({"role": "user", "content":
                              f"[RESULT of {p['label']} — {'ok' if r['ok'] else 'error'}, {r['seconds']} s]\n"
                              f"{text[-1500:] or '(no output)'}"
                              + (f"\nERROR: {r['error']}" if r["error"] else "")
                              + ("\n(It also produced an image.)" if r["images"] else "")})
        self._save()
        self.turn("result", f"[Spark] The run finished: {summary}")


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
                    text = str(body.get("text", "")).strip()[:4000]
                    if not text:
                        return self._json(400, {"error": "empty message"})
                    if ws.busy:
                        return self._json(409, {"error": "TASS is thinking"})
                    threading.Thread(target=ws.turn, args=("user", text), daemon=True).start()
                    return self._json(202, {"ok": True})
                if path == "/api/proposal":
                    return self._json(200, ws.decide(str(body.get("id")), str(body.get("decision"))))
                if path == "/api/settings":
                    if "thinking" in body:
                        ws.model.thinking = bool(body["thinking"])
                    if "auto_reflect" in body:
                        ws.auto_reflect = bool(body["auto_reflect"])
                    return self._json(200, ws.status())
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
