"""Recompute the SHA-256 of every file listed in a feature package's manifest
(for developers editing the trusted test feature). Never run this on a package
you did not write: it makes the manifest match whatever is on disk.

  python3 -I processor/scripts/refresh_fixture_manifest.py processor/testfeatures/deterministic-worker
"""
import hashlib, json, sys
from pathlib import Path

d = Path(sys.argv[1])
m = json.loads((d / "feature.json").read_text())
for f in m["files"]:
    f["sha256"] = hashlib.sha256((d / f["path"]).read_bytes()).hexdigest()
    print(f["sha256"], f["path"])
(d / "feature.json").write_text(json.dumps(m, indent=2) + "\n")
