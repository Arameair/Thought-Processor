"""The shared Python kernel and the journal notebook.

When JupyterLab is running, TASS creates a Jupyter *session* for
`TASS-live.ipynb` and runs approved code in that session's kernel. Opening
`TASS-live.ipynb` in JupyterLab attaches you to the same kernel, so you see
TASS's variables and TASS sees yours. Without JupyterLab it falls back to a
private kernel in the same folder.

Every run is also appended to `journal/tass-journal.ipynb` with its outputs.
"""

from __future__ import annotations

import json
import queue
import re
import threading
import time
import urllib.request
from pathlib import Path

ANSI = re.compile(r"\x1b\[[0-9;]*[A-Za-z]")
LIVE_NOTEBOOK = "TASS-live.ipynb"
JOURNAL = "journal/tass-journal.ipynb"


def _empty_notebook(cells=None) -> dict:
    return {"cells": cells or [], "metadata": {"kernelspec": {"display_name": "Python 3", "language": "python",
                                                             "name": "python3"},
                                               "language_info": {"name": "python"}},
            "nbformat": 4, "nbformat_minor": 5}


class SharedKernel:
    def __init__(self, shared: str | Path, jupyter_url: str | None = None, jupyter_token: str = "", log=print):
        self.shared = Path(shared)
        self.jupyter_url = jupyter_url.rstrip("/") if jupyter_url else None
        self.token = jupyter_token
        self.log = log
        self.client = None
        self.manager = None
        self.mode = "not started"
        self.kernel_id = None
        self.lock = threading.Lock()
        self.run_count = 0

    # --- start ------------------------------------------------------------------

    def _jupyter(self, method: str, path: str, body=None):
        req = urllib.request.Request(f"{self.jupyter_url}{path}", method=method,
                                     data=None if body is None else json.dumps(body).encode(),
                                     headers={"Authorization": f"token {self.token}",
                                              "Content-Type": "application/json"})
        with urllib.request.urlopen(req, timeout=20) as r:
            return json.load(r)

    def start(self) -> None:
        from jupyter_client import BlockingKernelClient, KernelManager
        from jupyter_core.paths import jupyter_runtime_dir

        self.shared.mkdir(parents=True, exist_ok=True)
        live = self.shared / LIVE_NOTEBOOK
        if not live.exists():
            cell = {"cell_type": "markdown", "id": "intro", "metadata": {},
                    "source": ["# TASS live kernel\n",
                               "This notebook is attached to the **same Python kernel TASS uses**. ",
                               "Run cells here and TASS can see the results; TASS's approved runs show up here as variables and files.\n",
                               "\nTASS's runs are recorded in `journal/tass-journal.ipynb`."]}
            live.write_text(json.dumps(_empty_notebook([cell]), indent=1))
        if self.jupyter_url:
            for _ in range(60):
                try:
                    sessions = self._jupyter("GET", "/api/sessions")
                    mine = [s for s in sessions if s.get("path") == LIVE_NOTEBOOK]
                    session = mine[0] if mine else self._jupyter("POST", "/api/sessions", {
                        "path": LIVE_NOTEBOOK, "name": LIVE_NOTEBOOK, "type": "notebook",
                        "kernel": {"name": "python3"}})
                    self.kernel_id = session["kernel"]["id"]
                    cf = Path(jupyter_runtime_dir()) / f"kernel-{self.kernel_id}.json"
                    for _ in range(50):
                        if cf.exists():
                            break
                        time.sleep(0.2)
                    kc = BlockingKernelClient()
                    kc.load_connection_file(str(cf))
                    kc.start_channels()
                    kc.wait_for_ready(timeout=60)
                    self.client, self.mode = kc, "shared with JupyterLab"
                    self.log(f"kernel: attached to JupyterLab session {LIVE_NOTEBOOK} ({self.kernel_id[:8]})")
                    return
                except Exception as e:  # noqa: BLE001 — Jupyter still starting
                    last = e
                    time.sleep(1)
            self.log(f"kernel: JupyterLab not reachable ({last}); using a private kernel")
        km = KernelManager(kernel_name="python3")
        km.start_kernel(cwd=str(self.shared))
        kc = km.client()
        kc.start_channels()
        kc.wait_for_ready(timeout=60)
        self.manager, self.client, self.mode = km, kc, "private"

    def restart(self) -> None:
        with self.lock:
            if self.jupyter_url and self.kernel_id:
                self._jupyter("POST", f"/api/kernels/{self.kernel_id}/restart")
                self.client.wait_for_ready(timeout=60)
            elif self.manager:
                self.manager.restart_kernel()
                self.client.wait_for_ready(timeout=60)

    # --- run --------------------------------------------------------------------

    def execute(self, code: str, timeout: float = 120.0) -> dict:
        """Run code; return {"ok", "text", "images", "error", "seconds", "outputs"}."""
        if self.client is None:
            raise RuntimeError("Kernel not started")
        with self.lock:
            kc = self.client
            started = time.monotonic()
            msg_id = kc.execute(code, store_history=True, allow_stdin=False)
            texts, images, outputs, error = [], [], [], None
            deadline = started + timeout
            while True:
                left = deadline - time.monotonic()
                if left <= 0:
                    self._interrupt()
                    error = f"Timed out after {int(timeout)} s (interrupted)"
                    break
                try:
                    msg = kc.get_iopub_msg(timeout=min(left, 1.0))
                except queue.Empty:
                    continue
                if msg.get("parent_header", {}).get("msg_id") != msg_id:
                    continue
                t, c = msg["msg_type"], msg["content"]
                if t == "stream":
                    texts.append(c["text"])
                    outputs.append({"output_type": "stream", "name": c["name"], "text": c["text"]})
                elif t in ("execute_result", "display_data"):
                    data = c.get("data", {})
                    if "image/png" in data:
                        images.append(data["image/png"])
                    if "text/plain" in data:
                        texts.append(data["text/plain"] + "\n")
                    out = {"output_type": t, "data": data, "metadata": c.get("metadata", {})}
                    if t == "execute_result":
                        out["execution_count"] = c.get("execution_count")
                    outputs.append(out)
                elif t == "error":
                    tb = ANSI.sub("", "\n".join(c.get("traceback", [])))
                    error = f"{c.get('ename')}: {c.get('evalue')}"
                    texts.append(tb + "\n")
                    outputs.append({"output_type": "error", "ename": c.get("ename"), "evalue": c.get("evalue"),
                                    "traceback": c.get("traceback", [])})
                elif t == "status" and c.get("execution_state") == "idle":
                    break
            self.run_count += 1
            result = {"ok": error is None, "text": "".join(texts), "images": images, "error": error,
                      "seconds": round(time.monotonic() - started, 2), "outputs": outputs}
        self._journal(code, result)
        return result

    def _interrupt(self) -> None:
        try:
            if self.jupyter_url and self.kernel_id:
                self._jupyter("POST", f"/api/kernels/{self.kernel_id}/interrupt")
            elif self.manager:
                self.manager.interrupt_kernel()
        except Exception:  # noqa: BLE001
            pass

    def _journal(self, code: str, result: dict) -> None:
        path = self.shared / JOURNAL
        path.parent.mkdir(parents=True, exist_ok=True)
        try:
            nb = json.loads(path.read_text("utf-8"))
        except (OSError, ValueError):
            nb = _empty_notebook([{"cell_type": "markdown", "id": "journal", "metadata": {},
                                   "source": ["# TASS journal\n", "Every run Dustin approved, with its output. "
                                              "Written by the Thought Processor workspace."]}])
        n = len(nb["cells"])
        nb["cells"].append({"cell_type": "code", "id": f"run-{n}", "execution_count": None,
                            "metadata": {"tass": {"at": time.strftime("%Y-%m-%d %H:%M:%S"),
                                                  "seconds": result["seconds"]}},
                            "source": code.splitlines(True), "outputs": result["outputs"]})
        tmp = path.with_name(".journal.tmp")
        tmp.write_text(json.dumps(nb, indent=1))
        tmp.replace(path)

    # --- what's in the shared folder ------------------------------------------------

    def listing(self, limit: int = 40) -> list[str]:
        out = []
        for p in sorted(self.shared.rglob("*")):
            rel = p.relative_to(self.shared)
            if any(part.startswith(".") for part in rel.parts) or len(rel.parts) > 3:
                continue
            out.append(str(rel) + ("/" if p.is_dir() else ""))
            if len(out) >= limit:
                out.append("…")
                break
        return out
