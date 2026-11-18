"""Discord door v2: the only program between TASS's Core and the Communication Bridge.

Contract: core/docs/DISCORD_MAILBOX_CONTRACT_V2.md (Thought-Processor #127).

Core never holds the Bridge key or the bot token. It reads owner messages from <mailbox>/inbox/ and
writes replies to <mailbox>/replies/. The door posts a reply only after Identity has redeemed a
single-use decision bound to that exact reply (identity #46), at most once per owner message, and
never retries. Its own trusted records live in a separate state directory Core cannot write.

Derived from the workspace door (experiment/tass-grows-20261103, c30bce4). Standard library only.
"""
from __future__ import annotations

import argparse
import fcntl
import hashlib
import http.client
import json
import os
import re
import socket
import stat
import sys
import threading
import time
import urllib.error
import urllib.request
from datetime import datetime, timezone
from pathlib import Path

SCHEMA_INBOX = "tass.discord.inbox/2"
SCHEMA_REPLY = "tass.discord.reply/2"
SCHEMA_RECEIPT = "tass.discord.receipt/2"
SCHEMA_STATUS = "tass.discord.door-status/2"
SCHEMA_SOURCE = "tass.discord.source/1"
BINDING_VERSION = "tass.discord.reply/1"
DOOR_SERVICE = "discord-door"
ACTION = "discord.reply"

MAX_CHARS = 1900            # the Bridge's Discord limit, in characters (code points)
MIN_GAP = 3.0               # seconds between two posts
WINDOW, WINDOW_MAX = 600, 20  # at most 20 posts per 10 minutes
SETTLE = 0.5                # a reply file must be this old (s) before it is read
MAX_WAIT = 110.0            # a reply waiting longer than this is refused (decisions expire after 120 s)
DELIVERY_WAIT = 120.0       # no final Bridge state within this time: uncertain (never retried)
MAX_REPLY_FILE = 64 * 1024

BRIDGE_ID = re.compile(r"m_[0-9a-f]{20}")
SNOWFLAKE = re.compile(r"[1-9][0-9]{0,19}")
DECISION_ID = re.compile(r"dec_[0-9a-f]{8,64}")
CYCLE_ID = re.compile(r"[A-Za-z0-9._:-]{1,80}")
REPLY_NAME = re.compile(r"reply-(m_[0-9a-f]{20})\.json")
REPLY_KEYS = {"schema", "reply_to", "text", "cycle_id", "decision_id"}

# Go's unicode.IsSpace: the Bridge trims these from both ends before posting.
GO_SPACE = set("\t\n\v\f\r \x85\xa0\u1680\u2000\u2001\u2002\u2003\u2004\u2005\u2006\u2007\u2008\u2009"
               "\u200a\u2028\u2029\u202f\u205f\u3000")

# Bridge answers that mean "nothing was created" (docs/API.md of the Bridge).
DEFINITELY_NOT_CREATED = {400, 401, 403, 404, 409, 413, 415, 503}
BRIDGE_DELIVERY = {"queued": "queued", "sending": "sending", "delivered": "delivered",
                   "failed": "not_sent", "uncertain": "uncertain"}
FINAL = {"delivered", "not_sent", "uncertain", "answered_elsewhere"}
V1_MARKERS = ("outbox", "sent", "failed", ".door-state.json")


class DoorError(Exception):
    pass


def log(msg: str):
    print(f"[door {time.strftime('%H:%M:%S')}] {msg}", flush=True)   # never message text, never a key


def iso(t: float) -> str:
    return datetime.fromtimestamp(t, timezone.utc).strftime("%Y-%m-%dT%H:%M:%S.%fZ")


_TS = re.compile(r"(\d{4}-\d\d-\d\dT\d\d:\d\d:\d\d)(\.\d+)?(Z|[+-]\d\d:\d\d)")


def parse_ts(s) -> float | None:
    """RFC 3339 as the Bridge writes it (up to nanoseconds); None if it isn't one."""
    m = _TS.fullmatch(s) if isinstance(s, str) else None
    if not m:
        return None
    frac = (m.group(2) or ".0")[1:7].ljust(6, "0")
    tz = "+00:00" if m.group(3) == "Z" else m.group(3)
    try:
        return datetime.fromisoformat(f"{m.group(1)}.{frac}{tz}").timestamp()
    except ValueError:
        return None


def sha256_hex(b: bytes) -> str:
    return hashlib.sha256(b).hexdigest()


def binding(rec: dict, text: str) -> str:
    """The contract's eight-line binding, from the door's own record of the owner message."""
    lines = [BINDING_VERSION, DOOR_SERVICE, rec["id"], rec["source_message_id"], rec["sender_id"],
             rec["channel_id"], "discord:" + rec["channel_id"], sha256_hex(text.encode("utf-8"))]
    for v in lines:
        if not v or "\n" in v or "\r" in v:
            raise DoorError("a binding field is empty or contains a line break")
    return sha256_hex("\n".join(lines).encode("utf-8"))


def text_problem(text) -> str:
    """"" if the Bridge would post this text exactly as given; otherwise why not."""
    if not isinstance(text, str):
        return "text must be a string"
    if text == "":
        return "text is empty"
    try:
        text.encode("utf-8")
    except UnicodeEncodeError:
        return "text is not valid UTF-8"
    if text[0] in GO_SPACE or text[-1] in GO_SPACE:
        return "text starts or ends with whitespace (the Bridge would change it)"
    for ch in text:
        o = ord(ch)
        if (o < 0x20 and ch not in "\n\t") or 0x7F <= o <= 0x9F:
            return "text contains a control character (only newline and tab are allowed)"
    if len(text) > MAX_CHARS:
        return f"text is {len(text)} characters; the limit is {MAX_CHARS}"
    return ""


# ------------------------------------------------------------------ durable files

def _fsync_dir(d: Path):
    fd = os.open(d, os.O_RDONLY)
    try:
        os.fsync(fd)
    finally:
        os.close(fd)


def _tmp_write(path: Path, data: bytes, mode: int) -> Path:
    tmp = path.parent / f".tmp-{os.getpid()}-{os.urandom(6).hex()}-{path.name}"
    fd = os.open(tmp, os.O_WRONLY | os.O_CREAT | os.O_EXCL, mode)
    try:
        with os.fdopen(fd, "wb") as f:
            f.write(data)
            f.flush()
            os.fsync(f.fileno())
    except BaseException:
        tmp.unlink(missing_ok=True)
        raise
    return tmp


def _dump(obj) -> bytes:
    return (json.dumps(obj, indent=2, ensure_ascii=False) + "\n").encode("utf-8")


def write_atomic(path: Path, obj, mode: int = 0o644):
    """Temp file in the same directory, fsync, rename, fsync the directory."""
    tmp = _tmp_write(path, _dump(obj), mode)
    os.replace(tmp, path)
    _fsync_dir(path.parent)


def create_exclusive(path: Path, obj, mode: int = 0o600) -> bool:
    """Atomically create path with this content; False if it already exists."""
    tmp = _tmp_write(path, _dump(obj), mode)
    try:
        os.link(tmp, path)
    except FileExistsError:
        return False
    finally:
        tmp.unlink(missing_ok=True)
    _fsync_dir(path.parent)
    return True


def read_json(path: Path):
    try:
        return json.loads(path.read_text("utf-8"))
    except (OSError, ValueError):
        return None


# ------------------------------------------------------------------ source manifest

DISCORD_EPOCH_MS = 1420070400000
MANIFEST_MAGIC = "tass.discord.source-manifest/1"
MANIFEST_OWNER_LOST = "-"          # claimed, owner unknown (recorded only by an operator baseline)
_MANIFEST_KEY = re.compile(r"[0-9a-f]{64}")


class SourceManifest:
    """The bound, append-only record of every Discord source the door has ever claimed.

    One line per claim: "claim <sha256 of channel and message ID> <owning Bridge ID>", plus operator
    baseline lines. It is written (and fsynced) before the per-source claim file, never pruned and
    never rewritten. Every source claim file must agree with it, and every claim in it must have its
    file: a missing, extra, altered or replaced claim is detected, not recreated as new. Before and
    after each append, and before authorization and delivery, the door proves the path still names
    the file it opened, at the size it wrote, with the content it wrote (full check) or header (fast).
    """

    def __init__(self, path: Path):
        self.path, self.fd, self.size = path, -1, 0
        self.hdr, self.entries, self.baselines = b"", {}, []
        self.floor: int | None = None
        self.sha = hashlib.sha256()

    @classmethod
    def create(cls, path: Path) -> "SourceManifest":
        m = cls(path)
        m.hdr = f"{MANIFEST_MAGIC} {os.urandom(16).hex()}\n".encode()
        tmp = _tmp_write(path, m.hdr, 0o600)
        try:
            os.link(tmp, path)
        except FileExistsError:
            raise DoorError("the door's source manifest appeared while it was being created") from None
        finally:
            tmp.unlink(missing_ok=True)
        _fsync_dir(path.parent)
        m._open(len(m.hdr), m.hdr)
        return m

    @classmethod
    def load(cls, path: Path) -> "SourceManifest":
        m = cls(path)
        try:
            raw = path.read_bytes()
        except OSError as e:
            raise DoorError(f"the door's source manifest can't be read ({e.__class__.__name__})") from None
        complete = raw[:raw.rfind(b"\n") + 1]           # drop a torn last line (a crash mid-append)
        lines = complete.split(b"\n")[:-1]
        if not lines or not lines[0].startswith((MANIFEST_MAGIC + " ").encode()) or len(lines[0].split()) != 2:
            raise DoorError("the door's source manifest has no valid header")
        m.hdr = lines[0] + b"\n"
        for n, line in enumerate(lines[1:], 2):
            if not m._parse(line.decode("ascii", "replace")):
                raise DoorError(f"the door's source manifest line {n} is damaged")
        if len(complete) != len(raw):
            os.truncate(path, len(complete))
        m._open(len(complete), complete)
        return m

    def _parse(self, line: str) -> bool:
        f = line.split(" ")
        if len(f) == 3 and f[0] == "claim" and _MANIFEST_KEY.fullmatch(f[1]) \
                and (BRIDGE_ID.fullmatch(f[2]) or f[2] == MANIFEST_OWNER_LOST) and f[1] not in self.entries:
            self.entries[f[1]] = f[2]
            return True
        if len(f) == 6 and f[:2] == ["baseline", "v1"] and f[4] == "discord-ms" and f[2].isdigit() and f[5].isdigit() \
                and re.fullmatch(r"[a-z-]{1,40}", f[3]):
            self.floor = max(self.floor or 0, int(f[5]))
            self.baselines.append(line)
            return True
        return False

    def _open(self, size: int, content: bytes):
        self.fd = os.open(self.path, os.O_RDWR | os.O_APPEND)
        self.size = size
        self.sha.update(content)
        self.check(full=True)

    def close(self):
        if self.fd >= 0:
            os.close(self.fd)
            self.fd = -1

    def check(self, full: bool = False):
        """Raise DoorError unless the path still names this file, unchanged."""
        try:
            ps, fs = os.lstat(self.path), os.fstat(self.fd)
        except OSError as e:
            raise DoorError(f"the door's source manifest is missing or unreadable ({e.__class__.__name__})") from None
        if not stat.S_ISREG(ps.st_mode) or (ps.st_dev, ps.st_ino) != (fs.st_dev, fs.st_ino):
            raise DoorError("the door's source manifest was replaced by a different file")
        if fs.st_size != self.size:
            raise DoorError(f"the door's source manifest changed size outside the door ({fs.st_size} bytes, "
                            f"expected {self.size})")
        if full:
            if hashlib.sha256(os.pread(self.fd, self.size, 0)).hexdigest() != self.sha.hexdigest():
                raise DoorError("the door's source manifest was changed outside the door")
        elif os.pread(self.fd, len(self.hdr), 0) != self.hdr:
            raise DoorError("the door's source manifest has a changed header")

    def append(self, line: str):
        self.check()
        b = (line + "\n").encode("ascii")
        if not self._parse(line):
            raise DoorError("refusing to append an invalid manifest line")
        n = os.write(self.fd, b)
        self.size += n
        self.sha.update(b[:n])
        if n != len(b):
            raise DoorError("a short write to the door's source manifest")
        os.fsync(self.fd)
        self.check()                                  # it landed in the linked manifest, not an unlinked one


# ------------------------------------------------------------------ HTTP

_OPENER = urllib.request.build_opener(urllib.request.ProxyHandler({}))   # local services only, never a proxy


def http_json(url: str, method: str, body=None, headers=None, timeout: float = 10.0):
    """(status, dict or None, "") or (None, None, "refused" | "unknown").

    "refused": the connection was never made, so the request cannot have had any effect.
    "unknown": anything after that (timeout, reset): the request may have had its effect.
    """
    h = {"Accept": "application/json", **(headers or {})}
    data = None
    if body is not None:
        data = json.dumps(body).encode("utf-8")
        h["Content-Type"] = "application/json"
    req = urllib.request.Request(url, data=data, method=method, headers=h)
    try:
        with _OPENER.open(req, timeout=timeout) as r:
            code, raw = r.status, r.read()
    except urllib.error.HTTPError as e:
        code = e.code
        try:
            raw = e.read()
        except (OSError, http.client.HTTPException):
            raw = b""
    except urllib.error.URLError as e:
        return None, None, "refused" if isinstance(e.reason, ConnectionRefusedError) else "unknown"
    except (OSError, http.client.HTTPException):
        return None, None, "unknown"
    try:
        d = json.loads(raw or b"{}")
    except ValueError:
        d = None
    return code, (d if isinstance(d, dict) else None), ""


# ------------------------------------------------------------------ the door

class Config:
    def __init__(self, mailbox, state, bridge, key_file, identity, identity_key_file, channel_id, owner_id,
                 bridge_timeout: float = 10.0, identity_timeout: float = 5.0):
        self.mailbox, self.state = Path(mailbox), Path(state)
        self.bridge, self.identity = str(bridge).rstrip("/"), str(identity).rstrip("/")
        self.key_file, self.identity_key_file = Path(key_file), Path(identity_key_file)
        self.channel_id, self.owner_id = str(channel_id or ""), str(owner_id or "")
        self.bridge_timeout, self.identity_timeout = bridge_timeout, identity_timeout


class Door:
    def __init__(self, cfg: Config, now=time.time, accept_source_baseline: bool = False):
        self.cfg, self.now = cfg, now
        if not SNOWFLAKE.fullmatch(cfg.channel_id) or not SNOWFLAKE.fullmatch(cfg.owner_id):
            raise DoorError("DOOR_CHANNEL_ID and DOOR_OWNER_ID must be Discord IDs")
        box, st = cfg.mailbox.resolve(), cfg.state.resolve()
        if box == st or box.is_relative_to(st) or st.is_relative_to(box):
            raise DoorError("the mailbox and the door's state directory must not overlap")
        for m in V1_MARKERS:
            if (box / m).exists():
                raise DoorError(f"this mailbox was used by the v1 workspace door ({m}); give v2 its own mailbox")
        self.box, self.st = box, st
        # Evidence of an earlier run, gathered before this run creates anything (see _open_sources).
        prior = [n for n in ("door.lock", "cursor.json") if (st / n).exists()] + \
                [d for d in ("inbound", "claims", "sources") if (st / d).is_dir() and any((st / d).iterdir())]
        for d in ("inbox", "replies", "receipts", "rejected"):
            (box / d).mkdir(parents=True, exist_ok=True)
        for d in ("inbound", "claims", "sources"):
            (st / d).mkdir(parents=True, exist_ok=True)
        os.chmod(st, 0o700)
        self._lock = open(st / "door.lock", "a")
        try:
            fcntl.flock(self._lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
        except BlockingIOError:
            self._lock.close()
            raise DoorError("another door is already using this state directory") from None
        self.started = now()
        cur = read_json(st / "cursor.json") or {}
        self.cursor = int(cur.get("cursor", 0)) if isinstance(cur.get("cursor", 0), int) else 0
        self.epoch = cur.get("epoch")
        self.blocked = ""
        self.waiting = ""
        self.posts: list[float] = []
        self.refused_inbound: set[str] = set()
        self.pending: set[str] = set()
        self.ignored: set[str] = set()
        self.offline = 0
        self.integrity = ""                                # set once: the door stops hearing and answering
        self.manifest: SourceManifest | None = None
        self.baseline_report = ""
        self._src_lock = threading.RLock()
        try:
            if accept_source_baseline:
                self.baseline_report = self._accept_source_baseline(prior)
            else:
                self._open_sources(prior)
            self._recover()
        except BaseException:
            self.close()
            raise

    def close(self):
        if getattr(self, "manifest", None) is not None:
            self.manifest.close()
        try:
            fcntl.flock(self._lock, fcntl.LOCK_UN)
        finally:
            self._lock.close()

    # --- keys and calls -------------------------------------------------------------------------
    @staticmethod
    def _key(p: Path) -> str:
        try:
            k = p.read_text().strip()
        except OSError:
            return ""
        return k if len(k) >= 16 and "\n" not in k else ""

    def bridge_call(self, method, path, body=None, idem=None, auth=True):
        headers = {}
        if auth:
            k = self._key(self.cfg.key_file)
            if not k:
                raise DoorError("the Bridge API key file is missing")
            headers["Authorization"] = "Bearer " + k
        if idem:
            headers["Idempotency-Key"] = idem
        return http_json(self.cfg.bridge + path, method, body, headers, self.cfg.bridge_timeout)

    # --- immutable Discord sources ---------------------------------------------------------------
    # One Discord message (channel + immutable Discord message ID) may become at most ONE answerable
    # Bridge envelope, ever. The Bridge can hand the same Discord message over again under a new
    # Bridge ID (redelivery, a reset journal). The door keeps its own durable records in its private
    # state directory (Core can't read or write them), and each must agree with the others:
    #   sources.manifest           the bound append-only list of every claim (SourceManifest)
    #   sources/<key>.json         the per-source claim file naming the owning Bridge envelope
    #   inbound/<bridge id>.json   the door's record of each envelope (owner or duplicate_of)
    # A claim is written to the manifest first, then its file. Nothing is ever removed. A missing,
    # extra, altered or replaced record is never recreated as if its source were new: the door stops
    # (integrity error) and only an explicit operator baseline (--accept-source-baseline) resumes it.
    INTEGRITY_REMEDY = ("Keep every file. The door stopped hearing and answering Discord messages. After "
                        "investigating, stop the door and run `door.py --accept-source-baseline` with the same "
                        "state directory: it keeps all records and refuses every Discord message sent before then.")

    def _source_path(self, channel: str, source_id: str) -> Path:
        if not SNOWFLAKE.fullmatch(str(channel)) or not SNOWFLAKE.fullmatch(str(source_id)):
            raise DoorError("a Discord source needs a valid channel and message ID")
        key = "\n".join(("discord", str(channel), str(source_id))).encode("utf-8")
        return self.st / "sources" / (sha256_hex(key) + ".json")

    def _file_owner(self, path: Path) -> str | None:
        """None: no file. "": damaged. Otherwise the Bridge ID (or MANIFEST_OWNER_LOST) it names."""
        if not os.path.lexists(path):
            return None
        d = read_json(path) if path.is_file() and not path.is_symlink() else None
        if not isinstance(d, dict) or d.get("schema") != SCHEMA_SOURCE:
            return ""
        owner = d.get("bridge_id")
        return owner if isinstance(owner, str) and (BRIDGE_ID.fullmatch(owner) or owner == MANIFEST_OWNER_LOST) else ""

    def _fail(self, why: str):
        """Integrity failure: from now on nothing is heard, authorized or posted."""
        with self._src_lock:
            if not self.integrity:
                self.integrity = f"source integrity: {why}. {self.INTEGRITY_REMEDY}"
                log("STOPPED: source integrity: " + why)
        raise DoorError(self.integrity)

    def _write_source_file(self, path: Path, owner: str) -> bool:
        try:
            return create_exclusive(path, {"schema": SCHEMA_SOURCE, "bridge_id": owner, "claimed_at": iso(self.now())})
        except OSError as e:
            raise DoorError(f"the door's source index could not be written ({e.__class__.__name__}); "
                            "inbound messages are held until it can") from None

    def _below_floor(self, source_id: str) -> bool:
        m = self.manifest
        return m is not None and m.floor is not None and (int(source_id) >> 22) <= m.floor

    def _checked_owner(self, path: Path, full: bool) -> str | None:
        """The owner the manifest records for this source, proven to agree with its claim file.
        None: never claimed. Any disagreement is an integrity failure."""
        if self.integrity:
            raise DoorError(self.integrity)
        try:
            self.manifest.check(full=full)
        except DoorError as e:
            self._fail(str(e))
        owner, fo = self.manifest.entries.get(path.stem), self._file_owner(path)
        if owner is None:
            if fo is not None:
                self._fail("a source claim file exists that the manifest never recorded")
            return None
        if fo is None:
            if owner != MANIFEST_OWNER_LOST and self.record(owner) is None and self.claim(owner) is None:
                # The manifest line is durable but the file write was interrupted before anything
                # else happened for this owner: complete it for the SAME owner (never as new).
                if not self._write_source_file(path, owner):
                    self._fail("a source claim file appeared while it was being completed")
                return owner
            self._fail(f"the source claim held by {owner} is missing")
        if fo != owner:
            self._fail(f"a source claim file was altered (the manifest says {owner})")
        return owner

    def source_owner(self, channel: str, source_id: str) -> str:
        """The Bridge ID that owns this Discord message, or "" if none (or if anything disagrees)."""
        try:
            with self._src_lock:
                owner = self._checked_owner(self._source_path(channel, source_id), full=False)
        except DoorError:
            return ""
        return owner if owner and owner != MANIFEST_OWNER_LOST else ""

    def claim_source(self, channel: str, source_id: str, rid: str) -> str:
        """Claim the Discord message for envelope rid. Returns "" if rid owns it (now or already),
        otherwise why rid is a duplicate. Any doubt raises DoorError (and an integrity failure stops
        the door)."""
        path = self._source_path(channel, source_id)
        with self._src_lock:
            owner = self._checked_owner(path, full=False)
            if owner == rid:
                return ""                                  # our own claim (e.g. a crash before the record)
            if owner == MANIFEST_OWNER_LOST:
                return "duplicate of a Discord message already heard (recorded by an operator baseline)"
            if owner:
                return f"duplicate of Discord message already heard as {owner}"
            if self._below_floor(source_id):
                return "sent before the operator's source baseline, so it can't be proven new"
            try:
                self.manifest.append(f"claim {path.stem} {rid}")
            except (OSError, DoorError) as e:
                self._fail(f"the source manifest could not be appended ({e})")
            if not self._write_source_file(path, rid):
                self._fail("a source claim file appeared while it was being written")
            return ""

    def source_problem(self, rec: dict) -> str:
        """Called on receipt, before authorization and immediately before delivery: "" when this
        record provably owns its Discord message; otherwise why not. Damage raises DoorError."""
        with self._src_lock:
            owner = self._checked_owner(self._source_path(rec.get("channel_id", ""), rec.get("source_message_id", "")),
                                        full=True)
        if owner != rec.get("id"):
            return "another Bridge message already holds this Discord message (one reply per Discord message)"
        return ""

    def _dup_owner(self, channel: str, source_id: str) -> str:
        owner = self.manifest.entries.get(self._source_path(channel, source_id).stem)
        if owner:
            return owner
        return "baseline" if self._below_floor(source_id) else MANIFEST_OWNER_LOST

    # --- source records at startup ---------------------------------------------------------------
    def _manifest_path(self) -> Path:
        return self.st / "sources.manifest"

    def _open_sources(self, prior: list[str]):
        mp = self._manifest_path()
        if not mp.exists():
            if prior:
                raise DoorError("source integrity: the door's source manifest is missing but its state directory holds "
                                f"records of an earlier run ({', '.join(prior)}): this is an upgrade or state loss, "
                                "not a first start. " + self.INTEGRITY_REMEDY)
            self.manifest = SourceManifest.create(mp)
            return
        try:
            self.manifest = SourceManifest.load(mp)
            self._verify_all()
        except DoorError as e:
            msg = str(e) if str(e).startswith("source integrity") else f"source integrity: {e}. {self.INTEGRITY_REMEDY}"
            raise DoorError(msg) from None

    def _records(self) -> list[dict]:
        out = []
        for p in (self.st / "inbound").glob("m_*.json"):
            r = self.record(p.stem)
            if r is None:
                raise DoorError(f"the door's record {p.name} is damaged")
            out.append(r)
        return out

    def _source_files(self) -> dict[str, str]:
        out = {}
        for p in (self.st / "sources").iterdir():
            if p.name.startswith(".tmp-"):
                continue
            if not (p.suffix == ".json" and _MANIFEST_KEY.fullmatch(p.stem)):
                raise DoorError(f"an unexpected file {p.name} is in the door's source index")
            out[p.stem] = self._file_owner(p) or ""
        return out

    def _verify_all(self):
        """Every record, claim file and manifest line agrees; else DoorError (refuse to start)."""
        m = self.manifest
        m.check(full=True)
        files = self._source_files()
        for key, fo in files.items():
            if key not in m.entries:
                raise DoorError("a source claim file exists that the manifest never recorded")
            if fo != m.entries[key]:
                raise DoorError("a source claim file is damaged or altered" if fo else "a source claim file is damaged")
        for key, owner in m.entries.items():
            if key in files:
                continue
            if owner != MANIFEST_OWNER_LOST and self.record(owner) is None and self.claim(owner) is None:
                path = self.st / "sources" / f"{key}.json"   # interrupted claim: complete it for the same owner
                if not self._write_source_file(path, owner):
                    raise DoorError("a source claim file appeared while it was being completed")
                continue
            raise DoorError(f"the source claim held by {owner} is missing")
        for r in self._records():
            path = self._source_path(r.get("channel_id", ""), r.get("source_message_id", ""))
            owner = m.entries.get(path.stem)
            if not r.get("duplicate_of"):
                if owner != r["id"]:
                    raise DoorError(f"the record {r['id']} has no matching source claim")
            elif r["duplicate_of"] == "baseline":
                if owner == r["id"] or (owner is None and not self._below_floor(r["source_message_id"])):
                    raise DoorError(f"the duplicate record {r['id']} has no matching source claim")
            elif owner is None or owner == r["id"]:
                raise DoorError(f"the duplicate record {r['id']} has no matching source claim")

    def _accept_source_baseline(self, prior: list[str]) -> str:
        """The operator's explicit, auditable decision after an integrity refusal. Deletes nothing:
        the manifest (if any) and every claim file and record are kept; records with no claim are
        claimed (first heard wins; later ones are demoted to duplicates, as before); claims with no
        file get their file back for the SAME owner; a baseline line then refuses every Discord
        message sent at or before now. Conflicting or damaged records are still refused."""
        mp = self._manifest_path()
        try:
            self.manifest = SourceManifest.load(mp)
            self._verify_all()
            return ""                                      # intact: nothing to decide, nothing changed
        except FileNotFoundError:
            pass
        except DoorError:
            if self.manifest is None:
                if mp.exists():
                    raise                                  # a damaged manifest: the operator must look
            else:
                self.manifest.check(full=True)
        reason = "upgrade-or-state-loss" if self.manifest is None else "inconsistent-sources"
        if self.manifest is None and not mp.exists():
            if not prior:
                self.manifest = SourceManifest.create(mp)
                return "first start: a new source manifest was created"
            self.manifest = SourceManifest.create(mp)
        m = self.manifest
        files = self._source_files()
        for key, fo in sorted(files.items()):
            if not fo:
                raise DoorError(f"the source claim file {key}.json is damaged; inspect it (move it out of the state "
                                "directory only if you accept that) and run the baseline again")
            if key not in m.entries:
                m.append(f"claim {key} {fo}")
            elif m.entries[key] != fo:
                raise DoorError(f"the source claim file {key}.json names {fo} but the manifest names {m.entries[key]}; "
                                "resolve this by hand")
        for key, owner in sorted(m.entries.items()):
            if key not in files:
                self._write_source_file(self.st / "sources" / f"{key}.json", owner)
        recs = self._records()
        recs.sort(key=lambda r: (self.claim(r["id"]) is None, r.get("seq", 0), r["id"]))
        floor = max(0, int(self.now() * 1000) - DISCORD_EPOCH_MS)
        seeded = demoted = 0
        for r in recs:
            path = self._source_path(r.get("channel_id", ""), r.get("source_message_id", ""))
            floor = max(floor, int(r["source_message_id"]) >> 22)
            owner = m.entries.get(path.stem)
            if owner is None:
                if r.get("duplicate_of"):
                    owner = r["duplicate_of"] if BRIDGE_ID.fullmatch(str(r["duplicate_of"])) else MANIFEST_OWNER_LOST
                    if owner == r["id"]:
                        owner = MANIFEST_OWNER_LOST
                else:
                    owner = r["id"]
                m.append(f"claim {path.stem} {owner}")
                self._write_source_file(path, owner)
                seeded += 1
            elif not r.get("duplicate_of") and owner != r["id"]:
                self._demote(r, "a duplicate of a Discord message already heard (found by the operator baseline)")
                demoted += 1
        m.append(f"baseline v1 {int(self.now() * 1000)} {reason} discord-ms {floor}")
        self._verify_all()
        at = datetime.fromtimestamp((floor + DISCORD_EPOCH_MS) / 1000, timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
        return (f"operator source baseline recorded ({reason}): {len(m.entries)} claims kept, {seeded} added from "
                f"the door's records, {demoted} records demoted to duplicates; every Discord message sent at or "
                f"before {at} is refused from now on")

    def _demote(self, rec: dict, why: str):
        owner = self.manifest.entries.get(self._source_path(rec["channel_id"], rec["source_message_id"]).stem)
        rec = {**rec, "answerable": False, "duplicate_of": owner if owner and owner != rec["id"] else MANIFEST_OWNER_LOST, "not_answerable_reason": "; ".join(x for x in (rec.get("not_answerable_reason"), why) if x)}
        write_atomic(self.st / "inbound" / f"{rec['id']}.json", rec, 0o600)
        for p in (self.box / "inbox").glob(f"*-{rec['id']}.json"):
            p.unlink(missing_ok=True)                      # never leave a duplicate in Core's inbox
        log(f"refused inbound {rec['id']}: {why}")

    # --- records --------------------------------------------------------------------------------
    def record(self, rid: str) -> dict | None:
        r = read_json(self.st / "inbound" / f"{rid}.json")
        return r if isinstance(r, dict) and r.get("id") == rid else None

    def claim(self, rid: str) -> dict | None:
        c = read_json(self.st / "claims" / f"{rid}.json")
        return c if isinstance(c, dict) and c.get("reply_to") == rid else None

    def _receipt(self, c: dict) -> dict:
        r = {k: v for k, v in c.items() if k not in ("refinable", "queued_at")}
        r["final"] = c["state"] in FINAL and not c.get("refinable")
        return r

    def save_claim(self, c: dict):
        c["updated_at"] = iso(self.now())
        write_atomic(self.st / "claims" / f"{c['reply_to']}.json", c, 0o600)
        write_atomic(self.box / "receipts" / f"{c['reply_to']}.json", self._receipt(c))
        if c["state"] in FINAL:
            self.pending.discard(c["reply_to"])
        else:
            self.pending.add(c["reply_to"])

    def new_claim(self, rid: str, **fields) -> dict | None:
        now = iso(self.now())
        c = {"schema": SCHEMA_RECEIPT, "reply_to": rid, "cycle_id": "", "decision_id": "", "reply_file": "",
             "text_sha256": "", "binding": "", "state": "claimed", "detail": "", "bridge_message_id": "",
             "discord_message_id": "", "claimed_at": now, "updated_at": now, "refinable": False, **fields}
        if not create_exclusive(self.st / "claims" / f"{rid}.json", c):
            return None
        write_atomic(self.box / "receipts" / f"{rid}.json", self._receipt(c))
        if c["state"] not in FINAL:
            self.pending.add(rid)
        return c

    def finish(self, c: dict, state: str, detail: str, refinable: bool = False):
        c["state"], c["detail"], c["refinable"] = state, detail[:300], refinable
        self.save_claim(c)
        log(f"reply to {c['reply_to']}: {state}" + (f" ({detail[:120]})" if detail else ""))

    def _recover(self):
        for p in sorted((self.st / "claims").glob("m_*.json")):
            c = self.claim(p.stem)
            if not c:
                continue
            if c["state"] == "claimed":
                self.finish(c, "not_sent", "interrupted before the Identity redemption; nothing was posted")
            elif c["state"] == "redeemed":
                self.finish(c, "uncertain", "interrupted after the Identity redemption; the post may have been "
                                            "made; never retried", refinable=True)
            elif c["state"] not in FINAL:
                self.pending.add(c["reply_to"])

    # --- inbound and delivery updates -----------------------------------------------------------
    def pull(self, mode):
        while True:
            code, d, err = self.bridge_call("GET", f"/api/messages?after={self.cursor}&limit=100")
            if code != 200 or d is None:
                raise DoorError(f"the Bridge's message feed answered {code or err}")
            if self.epoch is not None and d.get("history_epoch") != self.epoch:
                log("the Bridge's history was cleared; continuing from its new cursor")
            self.epoch = d.get("history_epoch")
            for m in d.get("messages") or []:
                if isinstance(m, dict):
                    self.take(m, mode)
            c = d.get("cursor")
            if isinstance(c, int) and c >= 0:
                self.cursor = c
            if not d.get("more"):
                break
        write_atomic(self.st / "cursor.json", {"cursor": self.cursor, "epoch": self.epoch}, 0o600)

    def take(self, m: dict, mode):
        if m.get("direction") == "inbound" and m.get("source") == "discord":
            self.take_inbound(m, mode)
        elif m.get("direction") == "outbound" and BRIDGE_ID.fullmatch(str(m.get("reply_to_id") or "")):
            self.take_outbound(m)

    def sender_problem(self, m: dict) -> str:
        ch = self.cfg.channel_id
        checks = [
            (m.get("kind") == "message", "not a person's message"),
            (m.get("sender_kind") == "owner" and m.get("sender_id") == self.cfg.owner_id, "not the configured owner"),
            (m.get("source_channel_id") == ch, "not the configured channel"),
            (m.get("conversation_id") == "discord:" + ch, "conversation does not match the channel"),
            (SNOWFLAKE.fullmatch(str(m.get("source_message_id") or "")) is not None, "no valid Discord message ID"),
            (m.get("origin_trace") == ["discord"], "not a first-hop Discord message"),
            (isinstance(m.get("text"), str), "no text"),
        ]
        for ok, why in checks:
            if not ok:
                return why
        return ""

    def take_inbound(self, m: dict, mode):
        rid = str(m.get("id") or "")
        if not BRIDGE_ID.fullmatch(rid):
            return
        rec = self.record(rid)
        if rec is None:
            why = self.sender_problem(m)
            if why:
                if rid not in self.refused_inbound:
                    self.refused_inbound.add(rid)
                    log(f"refused inbound {rid}: {why}")
                return
            dup = self.claim_source(m["source_channel_id"], m["source_message_id"], rid)
            if dup:
                # Never answerable and never written to Core's inbox: it can't reach Identity or
                # POST /api/replies. Recorded so a reply naming it is refused, and logged once.
                rec = {"schema": SCHEMA_INBOX, "id": rid, "seq": 0, "source_message_id": m["source_message_id"],
                       "channel_id": m["source_channel_id"], "sender_id": m["sender_id"],
                       "sender_kind": m["sender_kind"], "conversation_id": m["conversation_id"],
                       "source_time": "", "received_at": str(m.get("received_at") or ""), "catch_up": bool(m.get("catch_up")),
                       "answerable": False, "duplicate_of": self._dup_owner(m["source_channel_id"], m["source_message_id"]),
                       "not_answerable_reason": dup}
                if create_exclusive(self.st / "inbound" / f"{rid}.json", rec):
                    log(f"refused inbound {rid}: {dup}")
                return
            received = parse_ts(m.get("received_at"))
            reasons = []
            if m.get("catch_up"):
                reasons.append("catch-up message (arrived while TASS was not listening)")
            if received is None:
                reasons.append("no valid receive time")
            elif received < self.started:
                reasons.append("catch-up message (received before the door started)")
            if m.get("replies"):
                reasons.append("already answered")
            if mode != "manual":
                reasons.append(f"the Bridge was in {mode!r} mode")
            seq = m.get("seq") if isinstance(m.get("seq"), int) and m.get("seq") >= 0 else 0
            rec = {"schema": SCHEMA_INBOX, "id": rid, "seq": seq,
                   "source_message_id": m["source_message_id"], "channel_id": m["source_channel_id"],
                   "sender_id": m["sender_id"], "sender_kind": m["sender_kind"],
                   "conversation_id": m["conversation_id"],
                   "source_time": str(m.get("source_time") or ""), "received_at": str(m.get("received_at") or ""),
                   "catch_up": bool(m.get("catch_up")), "answerable": not reasons,
                   "not_answerable_reason": "; ".join(reasons)}
            if not create_exclusive(self.st / "inbound" / f"{rid}.json", rec):
                rec = self.record(rid)
                if rec is None:
                    return
            else:
                log(f"heard {rid}" + ("" if rec["answerable"] else " (not answerable)"))
        if rec.get("duplicate_of"):
            return
        if self.source_problem(rec):                       # damage raises; a non-owner is never put in the inbox
            return
        inbox = self.box / "inbox" / f"{rec['seq']:06d}-{rid}.json"
        if not inbox.exists() and isinstance(m.get("text"), str):
            write_atomic(inbox, {**rec, "text": m["text"]})
        if m.get("replies") and self.claim(rid) is None:
            # Someone (the owner on the Bridge page, an echo test, another program) answered first.
            self.new_claim(rid, state="answered_elsewhere", detail="another reply was made through the Bridge")

    def take_outbound(self, m: dict):
        parent, out_id = m["reply_to_id"], str(m.get("id") or "")
        rec = self.record(parent)
        if rec is None or not BRIDGE_ID.fullmatch(out_id):
            return
        c = self.claim(parent)
        if c and c.get("bridge_message_id") == out_id:
            self.update_delivery(c, m)
            return
        text = m.get("text") if isinstance(m.get("text"), str) else ""
        if (c and c["state"] == "uncertain" and c.get("refinable") and not c.get("bridge_message_id")
                and m.get("kind") == "api_reply" and m.get("conversation_id") == rec["conversation_id"]
                and sha256_hex(text.encode("utf-8")) == c.get("text_sha256")):
            c["bridge_message_id"] = out_id          # the reply posted just before a crash
            self.update_delivery(c, m)
            return
        if m.get("kind") == "api_reply" and not self.blocked:
            self.blocked = ("another program replied to an owner message through the Bridge API; the door "
                            "stopped sending: remove the other responder, then restart the door")
            log(f"blocked: foreign API reply {out_id} to {parent}")
        if c is None:
            self.new_claim(parent, state="answered_elsewhere", detail=f"answered by a {m.get('kind')} reply")

    def update_delivery(self, c: dict, m: dict):
        if c["state"] in FINAL and not c.get("refinable"):
            return
        new = BRIDGE_DELIVERY.get(str(m.get("delivery") or ""))
        if new is None or (new == c["state"] and new not in FINAL):
            return
        if new in ("queued", "sending"):
            if c["state"] in FINAL:
                return                                # never move a recorded result back to "in progress"
            c["state"] = new
            self.save_claim(c)
            return
        detail = ""
        if new == "not_sent":
            detail = "the Bridge did not post it: " + str(m.get("delivery_detail") or "")[:200]
        elif new == "uncertain":
            detail = "the Bridge cannot tell whether it was posted; never retried: " + str(m.get("delivery_detail") or "")[:160]
        if new == "delivered":
            c["discord_message_id"] = str(m.get("external_id") or "")
        self.finish(c, new, detail)

    def check_timeouts(self):
        now = self.now()
        for rid in sorted(self.pending):
            c = self.claim(rid)
            if not c or c["state"] in FINAL:
                self.pending.discard(rid)
                continue
            t = c.get("queued_at") or c.get("claimed_at")
            started = parse_ts(t) or now
            if c["state"] in ("queued", "sending", "redeemed") and now - started > DELIVERY_WAIT:
                self.finish(c, "uncertain", f"no final delivery state from the Bridge within {int(DELIVERY_WAIT)} s; "
                                            "never retried", refinable=True)

    # --- replies --------------------------------------------------------------------------------
    def reject(self, p: Path, why: str, **ids):
        out = {"file": p.name, "reason": why, "at": iso(self.now()), **ids}
        dst = self.box / "rejected" / (p.stem + ".json")
        if dst.exists():
            dst = self.box / "rejected" / f"{p.stem}-{os.urandom(4).hex()}.json"
        write_atomic(dst, out)
        try:
            p.unlink(missing_ok=True)
            _fsync_dir(p.parent)
        except OSError:
            self.ignored.add(p.name)                       # e.g. a directory: rejected once, then left alone
        log(f"rejected {p.name}: {why}")

    def ready(self, st: dict | None, rate: bool = True) -> str:
        """"" when a reply may be sent now; otherwise why not (rate=False: ignore the rate limit)."""
        if st is None:
            return "the Communication Bridge isn't answering"
        if st.get("discord_state") != "connected":
            return "Discord is not connected"
        if st.get("paused") is not False:
            return "the Bridge is paused"
        if st.get("mode") != "manual":
            return f"the Bridge is in {st.get('mode')!r} mode; it must be in manual mode"
        if self.blocked:
            return self.blocked
        if not rate:
            return ""
        now = self.now()
        self.posts = [t for t in self.posts if now - t < WINDOW]
        if self.posts and now - self.posts[-1] < MIN_GAP or len(self.posts) >= WINDOW_MAX:
            return "rate limit: waiting before the next post"
        return ""

    def reply_files(self):
        out = []
        for p in (self.box / "replies").iterdir():
            if p.name.startswith(".") or p.name in self.ignored:
                continue
            try:
                s = os.lstat(p)
            except OSError:
                continue
            out.append((s.st_mtime, p, s))
        return sorted(out, key=lambda x: (x[0], x[1].name))

    def process_replies(self, st: dict | None):
        for mtime, p, s in self.reply_files():
            if self.now() - mtime < SETTLE:
                continue
            if self.handle(p, s, st, self.now() - mtime) in ("wait", "posted"):
                return                                     # at most one post per tick

    def handle(self, p: Path, s: os.stat_result, st: dict | None, age: float) -> str:
        import stat as _stat
        mt = REPLY_NAME.fullmatch(p.name)
        if not _stat.S_ISREG(s.st_mode):
            self.reject(p, "not a regular file")
            return "done"
        if not mt:
            self.reject(p, "the file name must be reply-<bridge message id>.json")
            return "done"
        rid = mt.group(1)
        if s.st_size > MAX_REPLY_FILE:
            self.reject(p, "the file is too large", reply_to=rid)
            return "done"
        try:
            fd = os.open(p, os.O_RDONLY | os.O_NOFOLLOW)
            with os.fdopen(fd, "rb") as f:
                d = json.loads(f.read().decode("utf-8"))
        except (OSError, UnicodeDecodeError, ValueError):
            self.reject(p, "not readable UTF-8 JSON", reply_to=rid)
            return "done"
        if not isinstance(d, dict) or set(d) != REPLY_KEYS:
            self.reject(p, "the reply must have exactly the keys " + ", ".join(sorted(REPLY_KEYS)), reply_to=rid)
            return "done"
        if d["schema"] != SCHEMA_REPLY:
            self.reject(p, f"schema must be {SCHEMA_REPLY}", reply_to=rid)
            return "done"
        if d["reply_to"] != rid:
            self.reject(p, "reply_to does not match the file name", reply_to=rid)
            return "done"
        cycle, decision = d["cycle_id"], d["decision_id"]
        if not isinstance(cycle, str) or not CYCLE_ID.fullmatch(cycle):
            self.reject(p, "cycle_id is not valid", reply_to=rid)
            return "done"
        if not isinstance(decision, str) or not DECISION_ID.fullmatch(decision):
            self.reject(p, "decision_id is not valid", reply_to=rid, cycle_id=cycle)
            return "done"
        ids = {"reply_to": rid, "cycle_id": cycle, "decision_id": decision}
        why = text_problem(d["text"])
        if why:
            self.reject(p, why, **ids)
            return "done"
        text = d["text"]
        rec = self.record(rid)
        if rec is None:
            self.reject(p, "no owner message with that id was heard by this door", **ids)
            return "done"
        if not rec.get("answerable"):
            self.reject(p, "that message must not be answered: " + rec.get("not_answerable_reason", ""), **ids)
            return "done"
        if (rec.get("channel_id") != self.cfg.channel_id or rec.get("conversation_id") != "discord:" + rec["channel_id"]
                or rec.get("sender_id") != self.cfg.owner_id):
            self.reject(p, "the recorded message does not match the configured channel and owner", **ids)
            return "done"
        if rec.get("duplicate_of"):
            self.reject(p, "another Bridge message already holds this Discord message (one reply per Discord message)", **ids)
            return "done"
        why = self.source_problem(rec)                     # damage raises DoorError: the reply file stays, unsent
        if why:
            self.reject(p, why, **ids)
            return "done"
        c = self.claim(rid)
        if c is not None:
            if c.get("decision_id") == decision and c.get("reply_file") == p.name:
                p.unlink(missing_ok=True)                  # already handled before a crash
                return "done"
            self.reject(p, "that message already has a reply (one reply per owner message)", **ids)
            return "done"
        not_ready = self.ready(st)
        if not_ready:
            if age > MAX_WAIT:
                self.reject(p, f"waited {int(age)} s without sending ({not_ready}); the decision would expire", **ids)
                return "done"
            self.waiting = not_ready
            return "wait"
        bind = binding(rec, text)
        c = self.new_claim(rid, cycle_id=cycle, decision_id=decision, reply_file=p.name,
                           text_sha256=sha256_hex(text.encode("utf-8")), binding=bind)
        if c is None:
            self.reject(p, "that message already has a reply (one reply per owner message)", **ids)
            return "done"
        p.unlink(missing_ok=True)
        _fsync_dir(p.parent)
        why = self._source_guard(rec)                      # immediately before authorization
        if why:
            self.finish(c, "not_sent", "not authorized: " + why)
            return "done"
        ok, why = self.redeem(decision, bind)
        if not ok:
            self.finish(c, "not_sent", "Identity: " + why)
            return "done"
        c["state"] = "redeemed"
        self.save_claim(c)                                 # durable before the one post
        why = self._source_guard(rec)                      # immediately before delivery
        if why:
            self.finish(c, "not_sent", "redeemed, but refused before posting; nothing was posted; never retried: " + why)
            return "done"
        self.posts.append(self.now())
        self.post(c, rec, text)
        return "posted"

    def _source_guard(self, rec: dict) -> str:
        try:
            return self.source_problem(rec)
        except DoorError as e:
            return str(e)[:240]

    def redeem(self, decision: str, bind: str) -> tuple[bool, str]:
        k = self._key(self.cfg.identity_key_file)
        if not k:
            return False, "the door's Identity key file is missing"
        code, d, err = http_json(self.cfg.identity + "/v1/decisions/redeem", "POST",
                                 {"decision_id": decision, "action": ACTION, "binding": bind},
                                 {"Authorization": "Bearer " + k}, self.cfg.identity_timeout)
        if code is None:
            return False, "Identity is not reachable"
        if code != 200 or d is None:
            return False, f"Identity answered HTTP {code}"
        if d.get("allow") is not True:
            return False, "refused: " + str(d.get("reason") or "")[:200]
        if d.get("mode") != "redeem" or d.get("for_decision") != decision or d.get("service") != DOOR_SERVICE \
                or not isinstance(d.get("entity_id"), str) or not d.get("entity_id"):
            return False, "Identity's answer is not a redemption of this decision for the door"
        return True, ""

    def post(self, c: dict, rec: dict, text: str):
        rid = c["reply_to"]
        try:
            code, d, err = self.bridge_call("POST", "/api/replies", {"reply_to_id": rid, "text": text},
                                            idem="tass-door-" + rid)
        except DoorError as e:
            self.finish(c, "not_sent", str(e))
            return
        if code is None:
            if err == "refused":
                self.finish(c, "not_sent", "the Bridge was not reachable; nothing was posted")
            else:
                # The Bridge may have created it: refinable, so the Bridge's own record of this reply
                # (same message, same text) is recognised as ours later, never as another responder.
                self.finish(c, "uncertain", "no answer from the Bridge after posting; it may have been sent; never retried",
                            refinable=True)
            return
        if code in DEFINITELY_NOT_CREATED:
            self.finish(c, "not_sent", f"the Bridge refused it (HTTP {code}): " + str((d or {}).get("error") or "")[:200])
            return
        msg = (d or {}).get("message")
        if code not in (200, 201) or not isinstance(msg, dict):
            self.finish(c, "uncertain", f"unexpected Bridge answer (HTTP {code}); it may have been sent; never retried",
                        refinable=True)
            return
        mid = str(msg.get("id") or "")
        if not (BRIDGE_ID.fullmatch(mid) and msg.get("reply_to_id") == rid and msg.get("direction") == "outbound"
                and msg.get("conversation_id") == rec["conversation_id"] and msg.get("text") == text):
            self.blocked = "the Bridge's receipt did not match the authorized reply; the door stopped sending"
            if BRIDGE_ID.fullmatch(mid):
                c["bridge_message_id"] = mid
            self.finish(c, "uncertain", "the Bridge's receipt does not match the authorized reply; never retried")
            return
        c["bridge_message_id"], c["queued_at"] = mid, iso(self.now())
        c["state"] = "queued"
        self.save_claim(c)
        log(f"reply to {rid} queued as {mid}")
        self.update_delivery(c, msg)

    # --- loop -----------------------------------------------------------------------------------
    def write_status(self, st: dict | None, problem: str):
        n = {"delivered": 0, "not_sent": 0, "uncertain": 0, "answered_elsewhere": 0, "in_progress": 0}
        for p in (self.st / "claims").glob("m_*.json"):
            c = read_json(p) or {}
            k = c.get("state")
            n[k if k in n else "in_progress"] += 1
        not_ready = self.ready(st, rate=False) if not problem else problem
        out = {"schema": SCHEMA_STATUS, "door": "running", "updated_at": iso(self.now()),
               "ready": not not_ready, "not_ready": not_ready, "problem": problem, "blocked": self.blocked,
               "waiting": self.waiting,
               "bridge": {k: (st or {}).get(k) for k in ("discord_state", "paused", "mode")} if st else
               {"discord_state": "bridge_offline"},
               "replies_waiting": sum(1 for _, p, _ in self.reply_files()),
               "refused_inbound": len(self.refused_inbound), "receipts": n,
               "limits": {"max_chars": MAX_CHARS, "seconds_between_posts": MIN_GAP,
                          "posts_per_10_min": WINDOW_MAX, "max_wait_s": MAX_WAIT}}
        write_atomic(self.box / "status.json", out)
        return out

    def tick(self):
        self.waiting = ""
        if self.integrity:
            self.write_status(None, self.integrity[:400])
            return
        code, st, _ = self.bridge_call("GET", "/api/status", auth=False)
        if code != 200 or st is None:
            self.offline += 1
            self.write_status(None, "the Communication Bridge isn't answering")
            return
        self.offline = 0
        problem = ""
        try:
            self.pull(st.get("mode"))
            self.process_replies(st)
            self.check_timeouts()
        except DoorError as e:
            problem = str(e)[:400]
        self.write_status(st, problem)

    def run(self, every: float = 1.0):
        log(f"mailbox {self.box}, bridge {self.cfg.bridge}, identity {self.cfg.identity}")
        while True:
            self.tick()
            if self.offline * every > 60:
                log("the Bridge has been unreachable for a minute; restarting")
                raise SystemExit(3)
            time.sleep(every)


def forward(listen_port: int, target_port: int):
    """Let the owner open the Bridge's own page: the Bridge only listens on 127.0.0.1 inside its container."""
    srv = socket.create_server(("0.0.0.0", listen_port), reuse_port=False)

    def pipe(a, b):
        try:
            while chunk := a.recv(65536):
                b.sendall(chunk)
        except OSError:
            pass
        finally:
            for s in (a, b):
                try:
                    s.shutdown(socket.SHUT_RDWR)
                except OSError:
                    pass

    def serve():
        while True:
            c, _ = srv.accept()
            try:
                u = socket.create_connection(("127.0.0.1", target_port), timeout=5)
                u.settimeout(None)
            except OSError:
                c.close()
                continue
            threading.Thread(target=pipe, args=(c, u), daemon=True).start()
            threading.Thread(target=pipe, args=(u, c), daemon=True).start()

    threading.Thread(target=serve, daemon=True).start()
    log(f"bridge page forwarded on port {listen_port}")


def main():
    e = os.environ.get
    ap = argparse.ArgumentParser()
    ap.add_argument("--mailbox", default=e("DOOR_MAILBOX", "/mailbox"))
    ap.add_argument("--state", default=e("DOOR_STATE", "/door-state"))
    ap.add_argument("--bridge", default=e("DOOR_BRIDGE", "http://127.0.0.1:7820"))
    ap.add_argument("--key-file", default=e("DOOR_KEY_FILE", "/bridge-data/api-key"))
    ap.add_argument("--identity", default=e("DOOR_IDENTITY", "http://identity:8796"))
    ap.add_argument("--identity-key-file", default=e("DOOR_IDENTITY_KEY_FILE", "/run/secrets/door-identity-key"))
    ap.add_argument("--channel-id", default=e("DOOR_CHANNEL_ID", ""))
    ap.add_argument("--owner-id", default=e("DOOR_OWNER_ID", ""))
    ap.add_argument("--forward", type=int, default=int(e("DOOR_FORWARD_PORT", "0")))
    ap.add_argument("--every", type=float, default=1.0)
    ap.add_argument("--accept-source-baseline", action="store_true",
                    help="operator decision after a source-integrity refusal: keep every record, refuse every "
                         "Discord message sent before now, then exit")
    a = ap.parse_args()
    try:
        door = Door(Config(a.mailbox, a.state, a.bridge, a.key_file, a.identity, a.identity_key_file,
                           a.channel_id, a.owner_id), accept_source_baseline=a.accept_source_baseline)
    except DoorError as err:
        log(f"not starting: {err}")
        sys.exit(2)
    if a.accept_source_baseline:
        log(door.baseline_report or "the door's source records are intact; nothing was changed")
        door.close()
        return
    for b in door.manifest.baselines:
        log("source manifest operator baseline: " + b)
    if a.forward:
        forward(a.forward, int(a.bridge.rsplit(":", 1)[1].split("/")[0]))
    door.run(a.every)


if __name__ == "__main__":
    main()
