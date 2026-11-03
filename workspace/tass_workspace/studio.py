"""The Studio (Claude's station): the shared playground where Dustin and TASS imagine and build things together.

Everything here is real files in the shared space, so both of them see the same thing:
    studio/ideas/<slug>.md           ideas, with a "status:" line (new · exploring · built)
    studio/rooms/<slug>/index.html   rooms: things they built (a page, a tool, a view)
    studio/rooms/<slug>/room.md      optional: "# Title" and a one-line description
TASS sees the ideas and rooms in his prompt every turn, and builds rooms with approved code.
Rooms are served from /files/ in a sandbox, so they can't use the workspace's controls.
"""

from __future__ import annotations

import re
import time
from pathlib import Path

STATUSES = ("new", "exploring", "built")


def _slug(text: str) -> str:
    s = re.sub(r"[^a-z0-9]+", "-", text.lower()).strip("-")
    return s[:48] or f"idea-{int(time.time())}"


def _title_and_line(text: str, fallback: str) -> tuple[str, str]:
    title = next((l.lstrip("# ").strip() for l in text.splitlines() if l.startswith("#")), fallback)
    line = next((l.strip() for l in text.splitlines()
                 if l.strip() and not l.startswith("#") and not re.match(r"(?i)\s*(status|from)\s*:", l)), "")
    return title, line


class Studio:
    def __init__(self, shared: str | Path):
        self.root = Path(shared) / "studio"
        self.ideas_dir = self.root / "ideas"
        self.rooms_dir = self.root / "rooms"

    # --- ideas ------------------------------------------------------------------------

    def ideas(self) -> list[dict]:
        out = []
        if self.ideas_dir.is_dir():
            for p in sorted(self.ideas_dir.glob("*.md"), key=lambda x: x.stat().st_mtime, reverse=True):
                text = p.read_text("utf-8", "replace")
                title, line = _title_and_line(text, p.stem)
                m = re.search(r"(?im)^\s*status\s*:\s*(\w+)", text)
                frm = re.search(r"(?im)^\s*from\s*:\s*(.+)$", text)
                body = "\n".join(l for l in text.splitlines()
                                 if not l.startswith("#") and not re.match(r"(?i)\s*(status|from)\s*:", l)).strip()
                out.append({"id": p.stem, "title": title, "summary": line[:200], "text": body[:2000],
                            "status": (m.group(1).lower() if m else "new"), "from": frm.group(1).strip() if frm else "",
                            "path": f"studio/ideas/{p.name}", "updated": p.stat().st_mtime})
        return out

    def plant(self, title: str, text: str, author: str = "Dustin") -> dict:
        title = " ".join(title.split())[:120]
        if not title:
            raise ValueError("An idea needs a title.")
        self.ideas_dir.mkdir(parents=True, exist_ok=True)
        slug = _slug(title)
        path = self.ideas_dir / f"{slug}.md"
        n = 2
        while path.exists():
            path = self.ideas_dir / f"{slug}-{n}.md"
            n += 1
        path.write_text(f"# {title}\n\nstatus: new\nfrom: {author}\n\n{text.strip()[:4000]}\n", "utf-8")
        return next(i for i in self.ideas() if i["id"] == path.stem)

    # --- rooms ------------------------------------------------------------------------

    def rooms(self) -> list[dict]:
        out = []
        if self.rooms_dir.is_dir():
            for d in sorted((x for x in self.rooms_dir.iterdir() if x.is_dir() and not x.name.startswith(".")),
                            key=lambda x: x.stat().st_mtime, reverse=True):
                about = d / "room.md"
                title, line = _title_and_line(about.read_text("utf-8", "replace"), d.name) if about.is_file() \
                    else (d.name.replace("-", " ").title(), "")
                entry = d / "index.html"
                files = [str(p.relative_to(d)) for p in d.rglob("*") if p.is_file()]
                out.append({"id": d.name, "title": title, "summary": line[:200], "has_page": entry.is_file(),
                            "url": f"/files/studio/rooms/{d.name}/" + ("index.html" if entry.is_file() else ""),
                            "files": len(files),
                            "updated": max([d.stat().st_mtime] + [(d / f).stat().st_mtime for f in files])})
        return out

    # --- what TASS sees -------------------------------------------------------------------

    def prompt_section(self, public_url: str) -> str:
        ideas, rooms = self.ideas(), self.rooms()
        lines = ["PROCESSOR (Claude's station, our thought processor and shared playground: ideas we plant and rooms "
                 f"we build, kept in studio/; Dustin sees it at {public_url}/processor):"]
        if ideas:
            lines.append("Ideas:")
            for i in ideas[:10]:
                lines.append(f"- [{i['status']}] {i['title']} ({i['path']}){': ' + i['summary'] if i['summary'] else ''}")
        else:
            lines.append("Ideas: none planted yet.")
        if rooms:
            lines.append("Rooms: " + "; ".join(f"{r['title']} (studio/rooms/{r['id']}/)" for r in rooms[:10]))
        else:
            lines.append("Rooms: none built yet.")
        lines.append("To build a room: propose code (RUN: code) that writes studio/rooms/<short-name>/index.html "
                     "(a small self-contained HTML page) and studio/rooms/<short-name>/room.md (# Title + one line). "
                     "When you start or finish an idea, update its 'status:' line to exploring or built in the same code. "
                     "The library book 'Build a room' has a template.")
        return "\n".join(lines)

    def summary(self) -> dict:
        ideas = self.ideas()
        return {"ideas": len(ideas), "rooms": len(self.rooms()),
                "by_status": {s: sum(1 for i in ideas if i["status"] == s) for s in STATUSES}}
