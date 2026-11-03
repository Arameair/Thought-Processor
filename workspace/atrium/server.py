#!/usr/bin/env python3
"""A localhost-only, read-and-chat companion for the existing TASS Workshop.

No execution approvals, settings changes, GitHub tokens, or general filesystem serving.
The existing Workshop remains the execution/approval surface.
"""
from __future__ import annotations

import argparse
import json
import os
import secrets
import socket
import sys
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from urllib.error import HTTPError, URLError
from urllib.request import ProxyHandler, Request, build_opener
from urllib.parse import urlsplit

HERE = Path(__file__).resolve().parent
OPENER = build_opener(ProxyHandler({}))
ALLOWED_ASSETS = {
    "/assets/style.css": (HERE / "assets/style.css", "text/css; charset=utf-8"),
    "/assets/app.js": (HERE / "assets/app.js", "text/javascript; charset=utf-8"),
}


def safe_event(event: dict) -> dict:
    """Allowlist event fields; strip code, image payloads, link tokens and secrets."""
    typ = event.get("type")
    keys = {
        "user": ["text", "id"],
        "reply": ["text", "id", "action"],
        "reply_delta": ["delta", "id"],
        "thought_delta": ["delta", "id"],
        "thought_start": ["id"],
        "thought": ["seconds", "id"],
        "spark": ["kind", "text", "id"],
        "trace": ["books", "projects", "files", "history", "tokens", "context", "id", "opened"],
        "book": ["title", "path", "kind", "cells", "id"],
        "proposal": ["id"],
        "decision": ["status", "pid", "id"],
        "result": ["ok", "label", "text", "error", "seconds", "id"],
        "busy": ["busy", "id"],
        "notice": ["text", "id"],
        "memory": ["id"],
        "mind_pr": ["id", "ok", "path"],
        "error": ["text", "id"],
    }
    if not isinstance(typ, str) or typ not in keys:
        return {}
    clean = {"type": typ}
    for key in keys[typ]:
        value = event.get(key)
        if isinstance(value, (str, int, float, bool)):
            clean[key] = value[:8000] if isinstance(value, str) else value
        elif key == "opened" and isinstance(value, list):
            clean[key] = [str(item)[:150] for item in value[:4]]
    if typ == "proposal":
        p = event.get("proposal") or {}
        clean["proposal"] = {key: p.get(key) for key in ("id", "kind", "source", "status", "label")
                             if isinstance(p.get(key), (str, bool, int))}
    return clean


def safe_status(status: dict) -> dict:
    if not isinstance(status, dict):
        return {}
    identity = status.get("identity") or {}
    mind = status.get("mind") or {}
    return {
        "model_ok": status.get("model_ok") is True,
        "busy": status.get("busy") is True,
        "kernel": str(status.get("kernel", "unknown"))[:90],
        "kernel_error": bool(status.get("kernel_error")),
        "identity": {key: str(identity.get(key, ""))[:120] for key in ("name", "repo", "branch")},
        "mind": {
            "synced_at": mind.get("synced_at"),
            "mind": [{"path": str(x.get("path", ""))[:150]} for x in (mind.get("mind") or [])[:30] if isinstance(x, dict)],
            "library": [{"path": str(x.get("path", ""))[:150], "title": str(x.get("title", ""))[:120]}
                        for x in (mind.get("library") or [])[:30] if isinstance(x, dict)],
            "projects": [{"path": str(x.get("path", ""))[:150], "title": str(x.get("title", ""))[:120]}
                         for x in (mind.get("projects") or [])[:30] if isinstance(x, dict)],
        },
    }


def safe_state(raw: dict) -> dict:
    timeline = raw.get("timeline") if isinstance(raw, dict) else []
    proposals = raw.get("proposals") if isinstance(raw, dict) else []
    return {
        "status": safe_status(raw.get("status", {}) if isinstance(raw, dict) else {}),
        "timeline": [e for x in (timeline or [])[-140:] if isinstance(x, dict) if (e := safe_event(x))],
        "proposals": [
            {k: p.get(k) for k in ("id", "kind", "source", "status", "label")
             if isinstance(p.get(k), (str, int, bool))}
            for p in (proposals or [])[:30] if isinstance(p, dict)
        ],
    }


def create_handler(backend_port: int, csrf: str, server_port: int):
    backend = os.environ.get("TASS_ATRIUM_BACKEND_URL", f"http://127.0.0.1:{backend_port}").rstrip("/")
    valid_hosts = {f"127.0.0.1:{server_port}", f"localhost:{server_port}"}

    class Handler(BaseHTTPRequestHandler):
        server_version = "TASSAtrium/0.1"
        protocol_version = "HTTP/1.1"

        def log_message(self, fmt: str, *args):
            # No message content or credentials in console logs.
            if args and isinstance(args[0], str):
                print(f"{self.address_string()} {args[0][:95]}", file=sys.stderr)

        def _secure_headers(self, content_type: str, count: int, status: int = 200):
            self.send_response(status)
            self.send_header("Content-Type", content_type)
            self.send_header("Content-Length", str(count))
            self.send_header("Cache-Control", "no-store")
            self.send_header("X-Content-Type-Options", "nosniff")
            self.send_header("Referrer-Policy", "no-referrer")
            self.send_header("X-Frame-Options", "DENY")
            self.send_header("Content-Security-Policy", "default-src 'none'; script-src 'self'; style-src 'self'; connect-src 'self'; img-src 'self' data:; base-uri 'none'; form-action 'self'; frame-ancestors 'none'")
            self.end_headers()

        def _send(self, status: int, data: bytes, ctype="application/json; charset=utf-8"):
            self._secure_headers(ctype, len(data), status)
            try:
                self.wfile.write(data)
            except (BrokenPipeError, ConnectionResetError):
                pass

        def _json(self, code: int, obj: dict):
            self._send(code, json.dumps(obj, ensure_ascii=False).encode("utf-8"))

        def _valid_host(self) -> bool:
            return self.headers.get("Host", "") in valid_hosts

        def _upstream(self, path: str, *, method: str = "GET", data: bytes | None = None, timeout: int = 8):
            req = Request(backend + path, data=data, method=method,
                          headers={"Content-Type": "application/json", "Accept": "application/json"})
            return OPENER.open(req, timeout=timeout)

        def do_GET(self):
            if not self._valid_host():
                return self._json(403, {"error": "invalid host"})
            path = urlsplit(self.path).path
            if path == "/":
                raw = (HERE / "index.html").read_text("utf-8")
                data = raw.replace("__PLAYGROUND_CSRF__", csrf).encode("utf-8")
                return self._send(200, data, "text/html; charset=utf-8")
            if path in ALLOWED_ASSETS:
                p, ctype = ALLOWED_ASSETS[path]
                return self._send(200, p.read_bytes(), ctype)
            if path == "/bridge/health":
                return self._json(200, {"playground": "ready", "backend": "unverified"})
            if path == "/bridge/diagnostics":
                # Read-only, on-demand check. Never returns tokens or internal addresses.
                try:
                    with self._upstream("/api/status", timeout=3) as response:
                        status = json.loads(response.read(60_000))
                    if not isinstance(status, dict):
                        raise ValueError("unexpected status")
                    available = status.get("model_ok") is True
                    result = {
                        "atrium": "ready", "workshop": "reachable",
                        "model": "ready" if available else "not ready",
                        "identity": "repo mirrored" if (status.get("identity") or {}).get("repo") else "not confirmed",
                        "next_step": ("Send a message to TASS, then review any proposed code in the Workshop."
                                      if available else
                                      "The Workshop responds but the model is unavailable. Check the existing Workshop status/logs."),
                    }
                    return self._json(200, result)
                except (OSError, ValueError, HTTPError, URLError):
                    return self._json(200, {
                        "atrium": "ready", "workshop": "unreachable", "model": "unknown",
                        "identity": "not confirmed",
                        "next_step": "Check that the Workshop opens at http://127.0.0.1:8790/ and inspect its existing service logs."
                    })
            if path == "/bridge/state":
                try:
                    with self._upstream("/api/state") as response:
                        body = response.read(1_500_000)
                    return self._json(200, safe_state(json.loads(body)))
                except (OSError, ValueError, HTTPError, URLError) as exc:
                    return self._json(503, {"error": "TASS is offline or unreachable", "kind": type(exc).__name__})
            if path == "/bridge/events":
                return self._events()
            return self._json(404, {"error": "not found"})

        def _events(self):
            try:
                upstream = self._upstream("/api/events", timeout=40)
            except (OSError, HTTPError, URLError):
                return self._json(503, {"error": "TASS events unavailable"})
            self.send_response(200)
            self.send_header("Content-Type", "text/event-stream; charset=utf-8")
            self.send_header("Cache-Control", "no-store")
            self.send_header("Connection", "keep-alive")
            self.send_header("X-Content-Type-Options", "nosniff")
            self.end_headers()
            try:
                with upstream:
                    self.wfile.write(b": connected\n\n")
                    self.wfile.flush()
                    for line in upstream:
                        if line.startswith(b"data: "):
                            try:
                                event = safe_event(json.loads(line[6:]))
                            except (ValueError, UnicodeError):
                                continue
                            if event:
                                data = json.dumps(event, ensure_ascii=False).encode("utf-8")
                                self.wfile.write(b"data: " + data + b"\n\n")
                                self.wfile.flush()
                        elif line.startswith(b":"):
                            self.wfile.write(b": keepalive\n\n")
                            self.wfile.flush()
            except (BrokenPipeError, ConnectionResetError, OSError, TimeoutError):
                pass

        def do_POST(self):
            if not self._valid_host():
                return self._json(403, {"error": "invalid host"})
            if urlsplit(self.path).path != "/bridge/message":
                return self._json(404, {"error": "not found"})
            origin = self.headers.get("Origin")
            if origin and origin not in (f"http://127.0.0.1:{server_port}", f"http://localhost:{server_port}"):
                return self._json(403, {"error": "forbidden origin"})
            if self.headers.get("X-Playground-CSRF") != csrf:
                return self._json(403, {"error": "missing or invalid authorization"})
            if self.headers.get("Content-Type", "").split(";", 1)[0] != "application/json":
                return self._json(415, {"error": "JSON required"})
            try:
                n = int(self.headers.get("Content-Length", "0"))
                if not 0 < n <= 8192:
                    return self._json(413, {"error": "message too large"})
                obj = json.loads(self.rfile.read(n))
                msg = obj.get("text")
                if not isinstance(msg, str) or not msg.strip() or len(msg) > 4000:
                    return self._json(400, {"error": "text required (max 4000 characters)"})
                data = json.dumps({"text": msg.strip()}).encode("utf-8")
                with self._upstream("/api/message", method="POST", data=data) as upstream:
                    status = upstream.status
                    response = json.loads(upstream.read(16_384))
                return self._json(status, {"accepted": status == 202, "result": response.get("ok", False)})
            except HTTPError as exc:
                try:
                    description = json.loads(exc.read(4000)).get("error", "backend rejected message")
                except (OSError, ValueError):
                    description = "backend rejected message"
                return self._json(exc.code, {"error": str(description)[:150]})
            except (URLError, OSError, ValueError):
                return self._json(503, {"error": "TASS is unavailable"})

    return Handler


def main():
    cli = argparse.ArgumentParser(description="TASS Atrium — localhost-only companion playground")
    cli.add_argument("--port", type=int, default=8791)
    cli.add_argument("--backend-port", type=int, default=8790)
    args = cli.parse_args()
    if not (1024 <= args.port <= 65535 and 1024 <= args.backend_port <= 65535):
        cli.error("ports must be between 1024 and 65535")
    if args.port == args.backend_port:
        cli.error("playground port must be different from the Workshop port")
    handler = create_handler(args.backend_port, secrets.token_urlsafe(32), args.port)
    try:
        httpd = ThreadingHTTPServer((os.environ.get("TASS_ATRIUM_BIND_HOST", "127.0.0.1"), args.port), handler)
    except OSError as exc:
        cli.error(f"Cannot listen on port {args.port}: {exc}")
    httpd.daemon_threads = True
    print(f"TASS Atrium ready at http://127.0.0.1:{args.port}/", flush=True)
    print("Existing TASS Workshop is unchanged. Stop with Ctrl+C.", flush=True)
    try:
        httpd.serve_forever(poll_interval=0.25)
    except KeyboardInterrupt:
        pass
    finally:
        httpd.server_close()


if __name__ == "__main__":
    main()