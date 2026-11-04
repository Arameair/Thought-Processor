# TEST FIXTURE standing in for api.github.com (the sandbox can't use the owner's real GitHub).
# Serves the slice the core uses: contents GET/PUT (with ETags and blob shas) and git/trees.
# Seeded from a folder: python3 fake_github.py PORT REPO BRANCH SEED_DIR
# POST /fixture/put {"path","text"} adds or changes a file (a "new lesson on GitHub");
# GET /fixture/file?path=... shows a file; GET /fixture/log shows the requests seen.
import base64, hashlib, json, os, sys, threading
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from urllib.parse import urlparse, parse_qs

PORT, REPO, BRANCH, SEED = int(sys.argv[1]), sys.argv[2], sys.argv[3], sys.argv[4]
files, lock, log = {}, threading.Lock(), []
for root, _, names in os.walk(SEED):
    for n in names:
        p = os.path.relpath(os.path.join(root, n), SEED).replace(os.sep, "/")
        files[p] = open(os.path.join(root, n)).read()

def sha(t): return hashlib.sha1(t.encode()).hexdigest()

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
            pre = f"/repos/{REPO}/"
            if not u.path.startswith(pre): return self.send(404, {"message": "Not Found"})
            rest = u.path[len(pre):]
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
            if self.path == "/fixture/put":
                files[b["path"]] = b["text"]; log.append("FIXTURE put " + b["path"]); return self.send(200, {"ok": True})
        self.send(404, {})

ThreadingHTTPServer(("0.0.0.0", PORT), H).serve_forever()
