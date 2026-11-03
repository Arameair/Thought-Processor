"""How TASS thinks: build the prompt from its mind, stream the model, read its action."""

from __future__ import annotations

import json
import re
import time
import urllib.error
import urllib.request
from dataclasses import dataclass

HOW_TO_ACT = """HOW TO ACT
You can end a reply with ONE action, on its own line at the very end:
  OPEN: library/<file>          read a book before deciding (you'll get its text, then continue)
  RUN: library/<file>#<n>       propose running cell n of a book
  RUN: code                     propose your own code; put it in a ```python block right after this line
Dustin approves or rejects every RUN. Code runs in the shared Python kernel with the shared folder as the
working directory. Never say something ran, or what it printed, until you see its RESULT message.
Talk like a teammate: short, concrete, and say what you want to try and why."""


@dataclass
class Action:
    kind: str          # "open" | "run"
    book: str | None = None
    cell: int | None = None
    code: str | None = None

    def label(self) -> str:
        if self.kind == "open":
            return f"OPEN {self.book}"
        return f"RUN {self.book}#{self.cell}" if self.book else "RUN own code"


ACTION_OPEN = re.compile(r"^[ \t*_`>-]*OPEN[*_ ]*:[*_ ]*`?(library/[^\s`*]+)`?", re.M | re.I)
ACTION_RUN_BOOK = re.compile(r"^[ \t*_`>-]*RUN[*_ ]*:[*_ ]*`?(library/[^\s#`*]+)#(\d+)`?", re.M | re.I)
ACTION_RUN_CODE = re.compile(r"^[ \t*_`>-]*RUN[*_ ]*:[*_ ]*`?code`?[^\n]*\n+\s*```(?:python|py)?\s*\n(.*?)\n```",
                             re.M | re.I | re.S)


def parse_action(reply: str) -> tuple[str, Action | None]:
    """Split the reply into (text to show, action). The last action in the reply wins."""
    found = []
    for m in ACTION_RUN_CODE.finditer(reply):
        found.append((m.start(), m.end(), Action("run", code=m.group(1))))
    for m in ACTION_RUN_BOOK.finditer(reply):
        found.append((m.start(), m.end(), Action("run", book=m.group(1), cell=int(m.group(2)))))
    for m in ACTION_OPEN.finditer(reply):
        found.append((m.start(), m.end(), Action("open", book=m.group(1))))
    if not found:
        return reply.strip(), None
    start, end, action = max(found, key=lambda f: f[0])
    text = (reply[:start] + reply[end:]).strip()
    return text, action


def build_system(mind_files: list[tuple[str, str]], catalog: list, listing: list[str],
                 recent_runs: list[str]) -> str:
    parts = ["You are TASS. Your MIND is the set of files below, kept in Dustin's GitHub repo. "
             "Dustin can change them at any time; always follow the latest version.\n"]
    if mind_files:
        for path, text in mind_files:
            parts.append(f"=== MIND: {path} ===\n{text.strip()}\n")
    else:
        parts.append("(Your mind folder is empty or not synced yet. Say so if it matters.)\n")
    parts.append("You and Dustin share a workspace: a folder and a live Python (Jupyter) kernel. "
                 "You are building this environment together, as your shared space.\n")
    if catalog:
        parts.append("LIBRARY (books you can open; each has numbered Python cells):")
        for b in catalog:
            parts.append(f"- {b.path} — {b.title}: {b.summary} ({len(b.cells)} cells)")
        parts.append("")
    else:
        parts.append("LIBRARY: empty.\n")
    parts.append("SHARED FOLDER NOW:\n" + ("\n".join(f"  {l}" for l in listing) if listing else "  (empty)"))
    if recent_runs:
        parts.append("\nRECENT RUNS:\n" + "\n".join(recent_runs))
    parts.append("\n" + HOW_TO_ACT)
    return "\n".join(parts)


class ModelClient:
    def __init__(self, url: str, max_tokens: int = 3072, thinking: bool = True, temperature: float = 0.6,
                 margin: int = 256):
        self.url = url.rstrip("/")
        self.max_tokens = max_tokens
        self.thinking = thinking
        self.temperature = temperature
        self.margin = margin

    def _call(self, path: str, payload=None, timeout: int = 30):
        data = None if payload is None else json.dumps(payload).encode()
        req = urllib.request.Request(self.url + path, data=data, headers={"Content-Type": "application/json"})
        with urllib.request.urlopen(req, timeout=timeout) as r:
            return json.load(r)

    def health(self) -> bool:
        try:
            self._call("/health", timeout=3)
            return True
        except Exception:  # noqa: BLE001
            return False

    def measure(self, messages: list[dict]) -> tuple[int, int]:
        try:
            ctx = self._call("/props")["default_generation_settings"]["n_ctx"]
            prompt = self._call("/apply-template", {"messages": messages})["prompt"]
            return len(self._call("/tokenize", {"content": prompt, "add_special": True})["tokens"]), int(ctx)
        except Exception:  # noqa: BLE001 — not llama.cpp: estimate
            return sum(len(m["content"]) for m in messages) // 3 + 1, 8192

    def stream(self, messages: list[dict], on_thought, on_text, thinking: bool | None = None) -> dict:
        """Stream a reply. Calls on_thought(delta) for reasoning, on_text(delta) for the answer."""
        thinking = self.thinking if thinking is None else thinking
        used, ctx = self.measure(messages)
        limit = min(self.max_tokens, ctx - used - self.margin)
        if limit < 256:
            raise RuntimeError(f"Context full: prompt {used} of {ctx} tokens")
        payload = {"model": "local", "messages": messages, "stream": True, "max_tokens": limit,
                   "temperature": self.temperature,
                   "chat_template_kwargs": {"enable_thinking": thinking}}
        req = urllib.request.Request(self.url + "/v1/chat/completions", data=json.dumps(payload).encode(),
                                     headers={"Content-Type": "application/json"})
        thought, text, finish, in_think = [], [], None, False
        started = time.monotonic()
        try:
            with urllib.request.urlopen(req, timeout=600) as r:
                for raw in r:
                    line = raw.decode("utf-8", "replace").strip()
                    if not line.startswith("data:"):
                        continue
                    data = line[5:].strip()
                    if data == "[DONE]":
                        break
                    try:
                        chunk = json.loads(data)
                    except ValueError:
                        continue
                    ch = (chunk.get("choices") or [{}])[0]
                    finish = ch.get("finish_reason") or finish
                    delta = ch.get("delta", {})
                    if delta.get("reasoning_content"):
                        thought.append(delta["reasoning_content"])
                        on_thought(delta["reasoning_content"])
                    piece = delta.get("content") or ""
                    # Some servers inline the reasoning as <think>…</think> in the content.
                    while piece:
                        if in_think:
                            end = piece.find("</think>")
                            seg, piece = (piece, "") if end < 0 else (piece[:end], piece[end + 8:])
                            in_think = end < 0
                            if seg:
                                thought.append(seg)
                                on_thought(seg)
                        else:
                            beg = piece.find("<think>")
                            seg, piece = (piece, "") if beg < 0 else (piece[:beg], piece[beg + 7:])
                            in_think = beg >= 0
                            if seg:
                                text.append(seg)
                                on_text(seg)
        except urllib.error.URLError as e:
            raise RuntimeError(f"Model not reachable at {self.url}: {e}") from None
        return {"thought": "".join(thought), "text": "".join(text), "finish": finish,
                "prompt_tokens": used, "context": ctx, "seconds": round(time.monotonic() - started, 1)}
