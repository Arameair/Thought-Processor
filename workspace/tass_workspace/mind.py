"""TASS's mind and library, read from the folder the Liaison mirrors from GitHub.

Layout (written by identity-liaison's [sync]):
    <mirror>/manifest.json
    <mirror>/files/mind/*.md        who TASS is and how it thinks
    <mirror>/files/library/*.md     "books": markdown with numbered ```python cells
    <mirror>/files/library/*.ipynb  notebooks also work as books
"""

from __future__ import annotations

import json
import re
from dataclasses import dataclass
from pathlib import Path

FENCE = re.compile(r"```(?:python|py)\s*\n(.*?)\n```", re.S)


@dataclass
class Book:
    path: str
    title: str
    summary: str
    cells: list[str]
    text: str


class Mind:
    def __init__(self, mirror: str | Path):
        self.root = Path(mirror)
        self.files = self.root / "files"

    # --- status ---------------------------------------------------------------

    def manifest(self) -> dict:
        try:
            return json.loads((self.root / "manifest.json").read_text("utf-8"))
        except (OSError, ValueError):
            return {}

    def status(self) -> dict:
        m = self.manifest()
        return {
            "repo": m.get("repo"), "branch": m.get("branch"),
            "synced_at": m.get("synced_at"), "last_change_at": m.get("last_change_at"),
            "last_changed": m.get("last_changed", []),
            "mind": [{"path": p, "updated_at": self._updated(p)} for p, _ in self.mind_files()],
            "library": [{"path": b.path, "title": b.title, "summary": b.summary, "cells": len(b.cells)}
                        for b in self.catalog()],
        }

    def _updated(self, path: str):
        return self.manifest().get("files", {}).get(path, {}).get("updated_at")

    # --- mind -----------------------------------------------------------------

    def mind_files(self) -> list[tuple[str, str]]:
        d = self.files / "mind"
        if not d.is_dir():
            return []
        return [(f"mind/{p.name}", p.read_text("utf-8", "replace")) for p in sorted(d.glob("*.md"))]

    # --- library --------------------------------------------------------------

    def catalog(self) -> list[Book]:
        d = self.files / "library"
        if not d.is_dir():
            return []
        books = []
        for p in sorted(d.iterdir()):
            if p.suffix in (".md", ".ipynb") and not p.name.startswith(".") and p.stem.lower() != "readme":
                try:
                    books.append(self.book(f"library/{p.name}"))
                except (OSError, ValueError):
                    continue
        return books

    def book(self, path: str) -> Book:
        path = path.strip().lstrip("/")
        if not re.fullmatch(r"library/[\w.\- ]+\.(md|ipynb)", path):
            raise ValueError(f"Not a library book: {path}")
        p = self.files / path
        if not p.is_file():
            raise ValueError(f"No such book: {path}")
        raw = p.read_text("utf-8", "replace")
        if p.suffix == ".ipynb":
            nb = json.loads(raw)
            cells, parts = [], []
            for c in nb.get("cells", []):
                src = "".join(c.get("source", []))
                if c.get("cell_type") == "code":
                    cells.append(src)
                    parts.append(f"```python\n{src}\n```")
                else:
                    parts.append(src)
            text = "\n\n".join(parts)
        else:
            text = raw
            cells = [m.group(1) for m in FENCE.finditer(raw)]
        title = next((l.lstrip("# ").strip() for l in text.splitlines() if l.startswith("#")), p.stem)
        summary = next((l.strip() for l in text.splitlines()
                        if l.strip() and not l.startswith(("#", "```", "|", "-", "*", ">"))), "")
        return Book(path, title, summary[:200], cells, text)

    def numbered(self, book: Book) -> str:
        """The book's text with each python cell labelled CELL n, as TASS sees it."""
        counter = iter(range(1, len(book.cells) + 1))
        return FENCE.sub(lambda m: f"CELL {next(counter)}:\n```python\n{m.group(1)}\n```", book.text)
