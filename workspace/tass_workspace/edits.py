"""Small, safe edits to md files: FIND / ACTION / TEXT.

Copied from identity-liaison (identity_liaison/agent.py), where it was hardened
against real Nemotron 3 Nano 4B replies (identity issue #25 → PR #28). A small
model rewriting a whole file tends to drop or repeat parts of it; describing
one edit and letting the program apply it keeps everything else exactly as it was.
"""

from __future__ import annotations

import difflib
import re

_LABEL = r"(?:^|\n)[ \t>*_#-]*{name}[*_ ]*[:\-][*_ ]*"
EDIT_BLOCK = re.compile(
    _LABEL.format(name="FIND") + r"(?P<find>[^\n]*)"
    + _LABEL.format(name="ACTION") + r"`?(?P<action>[A-Za-z]+)`?[^\n]*\n"
    + r"(?:[ \t>*_#-]*TEXT[*_ ]*:?[*_ ]*\n)?"
    + r"\s*(?:<<<\s*TEXT[^\n]*\n(?P<text>.*?)\n?[ \t]*(?:TEXT)?[ \t]*>{3,}"
    + r"|```[\w-]*[ \t]*\n(?P<fenced>.*?)\n\s*```"
    + r"|(?P<loose>.*?)(?=\n[ \t>*_#-]*FIND[*_ ]*[:\-]|\Z))",
    re.S | re.I,
)


class EditError(ValueError):
    pass


def _norm(line: str) -> str:
    line = line.strip().strip("`\"'")
    line = re.sub(r"^(?:[-*+]\s+|\d+[.)]\s+|>\s*|#+\s*)", "", line)
    return " ".join(line.replace("**", "").split()).lower()


def _kind(line: str) -> str:
    s = line.strip()
    if re.match(r"^(?:[-*+]|\d+[.)])\s", s):
        return "list"
    if s.startswith("|"):
        return "table"
    return "other"


def _find_line(lines: list[str], find: str) -> int | None:
    target = find.strip()
    if not target or target.upper() in ("END", "END OF CARD", "(END)"):
        return None
    for i, l in enumerate(lines):
        if l.strip() == target.strip("`"):
            return i
    nt = _norm(target)
    hits = [i for i, l in enumerate(lines) if l.strip() and _norm(l) == nt]
    if not hits and len(nt) >= 15:
        hits = [i for i, l in enumerate(lines) if nt in _norm(l)]
    if not hits:
        normed = {i: _norm(l) for i, l in enumerate(lines) if l.strip()}
        close = difflib.get_close_matches(nt, list(normed.values()), n=1, cutoff=0.85)
        hits = [i for i, v in normed.items() if close and v == close[0]]
    if len(hits) >= 1:
        return hits[0]
    raise EditError(f"The model pointed at a line that isn't in the card: “{target[:120]}”. Nothing was written.")


def parse_edits(reply: str) -> tuple[str, list[tuple[str, str, str]]]:
    m = re.search(r"SUMMARY[*_ ]*:[*_ ]*(.+)", reply, re.I)
    summary = m.group(1).strip().strip("*_ ") if m else "Updated card."
    edits = []
    for b in EDIT_BLOCK.finditer(reply):
        action = b.group("action").lower()
        if action not in ("after", "before", "replace"):
            raise EditError(f"Unknown ACTION `{action}` in the model's reply. Nothing was written.")
        text = next(g for g in (b.group("text"), b.group("fenced"), b.group("loose")) if g is not None)
        text = text.strip("\n")
        inner = re.fullmatch(r"```(?:markdown|md)?\s*\n(.*?)\n```", text.strip(), re.S)
        find = b.group("find").strip().strip("*_").strip()
        if (find[:1] == find[-1:]) and find[:1] in ("`", '"', "'") and len(find) > 1:
            find = find[1:-1]
        edits.append((find, action, inner.group(1) if inner else text))
    return summary, edits


MARKER_ONLY = re.compile(r"^\s*(?:<<<\s*\w*|\w*\s*>{3,}|```[\w-]*)\s*$")


def _clean_text(text: str) -> list[str]:
    """Edit text minus stray format markers and lines the model repeated within it."""
    out, seen = [], set()
    for l in text.strip("\n").split("\n"):
        if MARKER_ONLY.match(l):
            continue
        key = _norm(l)
        if key and key in seen and len(key) >= 20:
            continue
        seen.add(key)
        out.append(l.rstrip())
    while out and not out[0].strip():
        out.pop(0)
    while out and not out[-1].strip():
        out.pop()
    return out


def apply_edits(old: str, edits: list[tuple[str, str, str]]) -> str:
    lines = old.rstrip("\n").split("\n")
    for find, action, text in edits:
        new_lines = _clean_text(text)
        if not new_lines:
            new_lines = [""]
        if not any(l.strip() for l in new_lines) and action != "replace":
            continue
        # Don't add text the card already contains.
        have = {_norm(l) for l in lines if l.strip()}
        if action != "replace" and all(_norm(l) in have for l in new_lines if l.strip()):
            continue
        idx = _find_line(lines, find)
        if idx is None:  # FIND: END  -> append at the end
            idx, action = len(lines) - 1, "after"
        if action == "replace":
            # Don't let a replacement re-add lines that already exist elsewhere in the card.
            others = {_norm(l) for j, l in enumerate(lines) if j != idx and l.strip()}
            new_lines = [l for l in new_lines if not l.strip() or _norm(l) not in others] or [""]
            # A retyped copy of the FIND line keeps the card's exact original wording/formatting.
            anchor = lines[idx]
            new_lines = [anchor if l.strip() and _norm(l) == _norm(anchor) else l for l in new_lines]
            lines[idx:idx + 1] = new_lines
            continue
        tight = _kind(lines[idx]) == _kind(new_lines[0]) != "other"
        if action == "before":
            pos = idx
            if tight:
                lines[pos:pos] = new_lines
            else:
                pad_before = [""] if pos > 0 and lines[pos - 1].strip() else []
                lines[pos:pos] = pad_before + new_lines + [""]
            continue
        pos = idx + 1
        if not tight:
            # Insert after the whole block (paragraph/list) the FIND line belongs to,
            # unless that line is a heading.
            if not lines[idx].lstrip().startswith("#"):
                while pos < len(lines) and lines[pos].strip():
                    pos += 1
            after = [""] if pos < len(lines) and lines[pos].strip() else []
            lines[pos:pos] = [""] + new_lines + after
        else:
            lines[pos:pos] = new_lines
    out = "\n".join(lines)
    out = re.sub(r"\n{3,}", "\n\n", out) if "\n\n\n" not in old else out
    return out.strip("\n") + "\n"
