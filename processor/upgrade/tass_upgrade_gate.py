#!/usr/bin/env python3
"""Read-only, fail-closed TASS upgrade eligibility gate for Thought-Processor #31.

This is NOT an upgrade tool. No files, grants, credentials, ports or services
are changed. Identity v64940b6 lacks an upgrade authorization transaction.
Run only against a disposable copy of a STOPPED P1 trial.
"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import stat
import sys

OLD = "86d79ba30ab3afb26bf015364fa0556dfa553c1e48d673b4243a827d6f8ba757"
NEW = "985027bbaebfd99bb07926285ceac218726ddca3fca0dff421bf665e2781abef"
EXPECTED_ID = "ins_2883db5a24e1e0e93cad"
GOOD_PATH = re.compile(r"[A-Za-z0-9_.-]+(?:/[A-Za-z0-9_.-]+)*\Z")
DIGEST = re.compile(r"[0-9a-f]{64}\Z")


class GateError(Exception):
    pass


def regular_bytes(path, limit):
    path = Path(path)
    if not path.is_file() or path.is_symlink() or not stat.S_ISREG(path.lstat().st_mode):
        raise GateError("MISSING_OR_UNSAFE_FILE")
    if path.stat().st_size > limit:
        raise GateError("FILE_TOO_LARGE")
    # Must not follow symlinked directories in the package.
    with path.open("rb") as f:
        content = f.read(limit + 1)
    if len(content) > limit:
        raise GateError("FILE_TOO_LARGE")
    return content


def package_digest(folder):
    folder = Path(folder)
    if folder.is_symlink() or not folder.is_dir():
        raise GateError("UNSAFE_PACKAGE_DIRECTORY")
    manifest_raw = regular_bytes(folder / "feature.json", 65536)
    try:
        manifest = json.loads(manifest_raw)
    except (ValueError, UnicodeError):
        raise GateError("INVALID_MANIFEST")
    if manifest.get("schema") != "thought.feature-manifest/1" or manifest.get("feature_id") != "tass":
        raise GateError("WRONG_FEATURE")
    files = manifest.get("files")
    if not isinstance(files, list) or not files or len(files) > 32:
        raise GateError("INVALID_FILE_LIST")
    lines = []
    seen = set()
    for entry in files:
        rel, declared = entry.get("path"), entry.get("sha256")
        if not isinstance(rel, str) or not GOOD_PATH.fullmatch(rel) or ".." in rel or rel == "feature.json":
            raise GateError("UNSAFE_PACKAGE_PATH")
        if rel in seen or not isinstance(declared, str) or not DIGEST.fullmatch(declared):
            raise GateError("INVALID_FILE_DIGEST")
        seen.add(rel)
        path = folder
        for part in rel.split("/"):
            path = path / part
            if path.is_symlink():
                raise GateError("UNSAFE_PACKAGE_LINK")
        actual = hashlib.sha256(regular_bytes(path, 10 * 1024 * 1024)).hexdigest()
        if actual != declared:
            raise GateError("FILES_CHANGED")
        lines.append((rel, actual))
    raw_sha = hashlib.sha256(manifest_raw).hexdigest()
    payload = "thought.feature-package/1\nmanifest " + raw_sha + "\n"
    payload += "".join(d + "  " + p + "\n" for p, d in sorted(lines))
    return hashlib.sha256(payload.encode()).hexdigest(), manifest, raw_sha


def check(installed, candidate, *, expected_id=EXPECTED_ID, old_pin=OLD, new_pin=NEW):
    installed, candidate = Path(installed), Path(candidate)
    try:
        record = json.loads(regular_bytes(installed / "install.json", 65536))
    except (ValueError, UnicodeError):
        raise GateError("INSTALL_RECORD_INVALID")
    if not re.fullmatch(r"ins_[0-9a-f]{20}", expected_id):
        raise GateError("INVALID_EXPECTED_ID")
    if record.get("installation_id") != expected_id or installed.name != expected_id:
        raise GateError("WRONG_INSTALLATION_ID")
    if record.get("state") != "INSTALLED" or record.get("feature_id") != "tass":
        raise GateError("NOT_INSTALLED_TASS")
    old_digest, old_manifest, old_manifest_sha = package_digest(installed / "package")
    new_digest, new_manifest, new_manifest_sha = package_digest(candidate)
    if old_digest != old_pin or record.get("artifact_sha256") != old_pin:
        raise GateError("OLD_ARTIFACT_NOT_PINNED")
    if new_digest != new_pin:
        raise GateError("NEW_ARTIFACT_NOT_PINNED")
    if record.get("manifest_sha256") != old_manifest_sha:
        raise GateError("OLD_MANIFEST_RECORD_MISMATCH")
    if record.get("identity_system_id") in (None, ""):
        raise GateError("MISSING_IDENTITY_BINDING")
    if record.get("files") != old_manifest.get("files"):
        raise GateError("OLD_FILE_RECORD_MISMATCH")
    if record.get("source_repo") != "Arameair/TASS/feature":
        raise GateError("SOURCE_NOT_TASS")
    if not (installed / "credential.json").is_file() or (installed / "credential.json").is_symlink():
        raise GateError("CREDENTIAL_MISSING_OR_UNSAFE")
    # Data is not read. The gate never inspects, copies or logs private chat.
    if not (installed / "data").is_dir() or (installed / "data").is_symlink():
        raise GateError("DATA_MISSING_OR_UNSAFE")
    old_caps = {c["id"] for c in old_manifest.get("capabilities", [])}
    new_caps = {c["id"] for c in new_manifest.get("capabilities", [])}
    if not old_caps.issubset(new_caps):
        raise GateError("EXISTING_CAPABILITY_REMOVED_REVIEW_REQUIRED")
    return {
        "gate": "BLOCKED_IDENTITY_UPGRADE_API_MISSING",
        "installation_id": expected_id,
        "old_artifact": old_digest,
        "new_artifact": new_digest,
        "new_manifest_sha256": new_manifest_sha,
        "retained_capabilities": sorted(old_caps & new_caps),
        "added_capabilities_must_be_separately_granted": sorted(new_caps - old_caps),
        "identity_approval": "NOT_REQUESTED",
        "record_or_data_modified": False,
        "safe_to_start_new_artifact": False,
        "next": "Owner-approved Identity upgrade protocol and disposable-state tests required"
    }


def main():
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--installed", required=True, help="TASS installation folder in a DISPOSABLE STOPPED profile copy")
    ap.add_argument("--candidate", required=True, help="Pinned new Arameair/TASS/feature folder")
    args = ap.parse_args()
    try:
        result = check(args.installed, args.candidate)
    except (GateError, OSError, TypeError, KeyError) as exc:
        print(json.dumps({"gate": "REFUSED", "reason": str(exc), "record_or_data_modified": False}, sort_keys=True))
        return 2
    print(json.dumps(result, sort_keys=True))
    return 3  # Always blocked until Identity can authorize same-ID upgrades.


if __name__ == "__main__":
    sys.exit(main())
