# TEST FIXTURE standing in for api.github.com (the sandbox can't use the owner's real GitHub).
# Serves the slice the core uses: contents GET/PUT (with ETags and blob shas) and git/trees.
# Seeded from a folder: python3 fake_github.py PORT REPO BRANCH SEED_DIR
# POST /fixture/put {"path","text"} adds or changes a file (a "new lesson on GitHub");
# GET /fixture/file?path=... shows a file; GET /fixture/log shows the requests seen.
# Mailbox (#76/#81), only for REPO Arameair/TASS: issue #24 (open, label tass-core) with comments
#   GET/POST /repos/R/issues/24[/comments]   real-looking numeric IDs, second-resolution created_at
#   POST /fixture/comment {"issue","user","body"} -> {"id"}   (someone writes on GitHub)
#   POST /fixture/fail {"read": bool, "post": ""|"down"|"lost"}
#        down: POST answers 503 and creates nothing; lost: the comment IS created, the answer is 502
#   GET  /fixture/comments?issue=24          what is on the issue now
import base64, hashlib, json, os, sys, threading, time
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from urllib.parse import urlparse, parse_qs

PORT, REPO, BRANCH, SEED = int(sys.argv[1]), sys.argv[2], sys.argv[3], sys.argv[4]
files, lock, log = {}, threading.Lock(), []
for root, _, names in os.walk(SEED):
    for n in names:
        p = os.path.relpath(os.path.join(root, n), SEED).replace(os.sep, "/")
        files[p] = open(os.path.join(root, n)).read()

def sha(t): return hashlib.sha1(t.encode()).hexdigest()

issues = {24: {"number": 24, "title": "TASS mailbox", "state": "open", "labels": [{"name": "tass-core"}]}} if REPO == "Arameair/TASS" else {}
comments, fail, next_id = [], {"read": False, "post": ""}, [4100000000]

def add_comment(issue, user, body):
    next_id[0] += 7
    c = {"id": next_id[0], "body": body, "user": {"login": user},
         "issue_url": f"https://api.github.com/repos/{REPO}/issues/{issue}",
         "html_url": f"https://github.com/{REPO}/issues/{issue}#issuecomment-{next_id[0]}",
         "created_at": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
         "updated_at": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime())}
    comments.append(c)
    return c

class H(BaseHTTPRequestHandler):
    def log_message(self, *a): pass
    def send(self, code, obj=None, etag=None):
        b = json.dumps(obj).encode() if obj is not None else b""
        self.send_response(code)
        if etag: self.send_header("ETag", etag)
        self.send_header("Content-Type", "application/json"); self.end_headers(); self.wfile.write(b)
    def do_GET(self):
        u = urlparse(self.path); q = parse_qs(u.query)
        with lock:
            log.append("GET " + u.path)
            if u.path == "/fixture/log": return self.send(200, {"requests": log[-200:], "count": len(log)})
            if u.path == "/fixture/file": return self.send(200, {"text": files.get(q["path"][0], "")})
            if u.path == "/fixture/comments":
                n = int(q.get("issue", ["24"])[0])
                return self.send(200, [c for c in comments if c["issue_url"].endswith(f"/issues/{n}")])
            pre = f"/repos/{REPO}/"
            if not u.path.startswith(pre): return self.send(404, {"message": "Not Found"})
            rest = u.path[len(pre):]
            if rest.startswith("issues"):
                if fail["read"]: log.append("  -> 503 (injected)"); return self.send(503, {"message": "Service Unavailable"})
                parts = rest.split("/")
                if len(parts) >= 2 and parts[1].isdigit() and int(parts[1]) in issues:
                    n = int(parts[1])
                    if len(parts) == 2: return self.send(200, issues[n])
                    if parts[2] == "comments":
                        mine = [c for c in comments if c["issue_url"].endswith(f"/issues/{n}")]
                        if "since" in q: mine = [c for c in mine if c["updated_at"] >= q["since"][0]]
                        page, per = int(q.get("page", ["1"])[0]), int(q.get("per_page", ["30"])[0])
                        return self.send(200, mine[(page - 1) * per: page * per])
                return self.send(404, {"message": "Not Found"})
            if rest.startswith("git/trees/"):
                tree = [{"path": p, "type": "blob", "sha": sha(t)} for p, t in sorted(files.items())]
                etag = '"' + sha("".join(x["path"] + x["sha"] for x in tree)) + '"'
                if self.headers.get("If-None-Match") == etag: log.append("  -> 304"); return self.send(304)
                return self.send(200, {"tree": tree, "truncated": False}, etag)
            if rest.startswith("contents/"):
                p = rest[len("contents/"):]
                if p not in files: return self.send(404, {"message": "Not Found"})
                etag = '"' + sha(files[p]) + '"'
                if self.headers.get("If-None-Match") == etag: log.append("  -> 304"); return self.send(304)
                return self.send(200, {"content": base64.b64encode(files[p].encode()).decode(), "encoding": "base64", "sha": sha(files[p])}, etag)
            self.send(404, {"message": "Not Found"})
    def do_PUT(self):
        u = urlparse(self.path); b = json.loads(self.rfile.read(int(self.headers["Content-Length"])))
        with lock:
            log.append("PUT " + u.path)
            p = u.path[len(f"/repos/{REPO}/contents/"):]
            if b.get("branch") != BRANCH: return self.send(422, {"message": "wrong branch"})
            if p in files and b.get("sha") != sha(files[p]): return self.send(409, {"message": "sha mismatch"})
            files[p] = base64.b64decode(b["content"]).decode()
            self.send(200, {"content": {"sha": sha(files[p])}})
    def do_POST(self):
        b = json.loads(self.rfile.read(int(self.headers["Content-Length"])))
        with lock:
            log.append("POST " + self.path)
            if self.path == "/fixture/comment":
                return self.send(201, add_comment(int(b.get("issue", 24)), b["user"], b["body"]))
            if self.path == "/fixture/fail":
                fail.update(b); return self.send(200, fail)
            pre = f"/repos/{REPO}/issues/"
            if self.path.startswith(pre) and self.path.endswith("/comments"):
                n = int(self.path[len(pre):].split("/")[0])
                if n not in issues: return self.send(404, {"message": "Not Found"})
                if self.headers.get("Authorization", "") == "": return self.send(401, {"message": "Requires authentication"})
                if fail["post"] == "down": log.append("  -> 503 (injected)"); return self.send(503, {"message": "Service Unavailable"})
                c = add_comment(n, "Arameair", b["body"])
                if fail["post"] == "lost": log.append("  -> 502 after create (injected)"); return self.send(502, {"message": "Bad Gateway"})
                return self.send(201, c)
            if self.path == "/fixture/put":
                files[b["path"]] = b["text"]; log.append("FIXTURE put " + b["path"]); return self.send(200, {"ok": True})
        self.send(404, {})

ThreadingHTTPServer(("0.0.0.0", PORT), H).serve_forever()
