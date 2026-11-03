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


@dataclass
class Project:
    name: str
    title: str
    description: str
    entry: str | None
    files: list[str]
    about: str
    root: Path

    @property
    def path(self) -> str:
        return f"projects/{self.name}"


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
            "projects": [{"path": p.path, "title": p.title, "entry": p.entry, "files": len(p.files)}
                         for p in self.projects()],
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

    # --- projects -------------------------------------------------------------
    # A project is a folder projects/<name>/ with a project.md (title, description, `entry: <file>`)
    # and its files. Added on GitHub (e.g. by ChatGPT), mirrored by the Liaison.

    def projects(self) -> list[Project]:
        d = self.files / "projects"
        if not d.is_dir():
            return []
        out = []
        for p in sorted(d.iterdir()):
            if p.is_dir() and not p.name.startswith("."):
                try:
                    out.append(self.project(p.name))
                except ValueError:
                    continue
        return out

    def project(self, name: str) -> Project:
        name = name.strip().strip("/").removeprefix("projects/").strip("/")
        if not re.fullmatch(r"[\w.\-]+", name) or name.startswith("."):
            raise ValueError(f"Not a project name: {name}")
        root = self.files / "projects" / name
        if not root.is_dir():
            raise ValueError(f"No such project: projects/{name}")
        files = sorted(str(p.relative_to(root)) for p in root.rglob("*")
                       if p.is_file() and not any(x.startswith(".") for x in p.relative_to(root).parts))
        about = next((root / f for f in ("project.md", "README.md") if (root / f).is_file()), None)
        text = about.read_text("utf-8", "replace") if about else ""
        title = next((l.lstrip("# ").strip() for l in text.splitlines() if l.startswith("#")), name)
        desc = next((l.strip() for l in text.splitlines()
                     if l.strip() and not l.startswith(("#", "```", "|", "-", "*", ">"))
                     and not re.match(r"(?i)entry\s*:", l.strip())), "")
        m = re.search(r"(?im)^\s*[-*]?\s*`?entry`?\s*:\s*`?([\w./\-]+)`?", text)
        entry = m.group(1) if m else next((f for f in ("main.py", "run.py") if f in files), None)
        if entry is None:
            entry = next((f for f in files if f.endswith((".py", ".ipynb"))), None)
        if entry and (entry not in files or ".." in entry):
            raise ValueError(f"projects/{name}: entry {entry} is not one of its files")
        return Project(name=name, title=title, description=desc[:200], entry=entry, files=files,
                       about=text, root=root)

    def entry_code(self, project: "Project") -> str:
        """The code that will actually run, shown to the owner before approval."""
        if not project.entry:
            return ""
        raw = (project.root / project.entry).read_text("utf-8", "replace")
        if project.entry.endswith(".ipynb"):
            nb = json.loads(raw)
            return "\n\n".join("".join(c.get("source", [])) for c in nb.get("cells", [])
                               if c.get("cell_type") == "code")
        return raw

    def numbered(self, book: Book) -> str:
        """The book's text with each python cell labelled CELL n, as TASS sees it."""
        counter = iter(range(1, len(book.cells) + 1))
        return FENCE.sub(lambda m: f"CELL {next(counter)}:\n```python\n{m.group(1)}\n```", book.text)
