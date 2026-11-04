#!/usr/bin/env python3
"""One pinned cell, inside a private Jupyter kernel. No HTTP server or general cell API.

This process is launched only by the fixed, disabled-by-default adapter. It
reuses the Workshop's SharedKernel implementation from an exact Git checkout;
the owner's live Jupyter kernel/session is NEVER attached.
"""
from __future__ import annotations

import argparse
import hashlib
import importlib.util
import json
import os
import sys
import tempfile
from pathlib import Path

CELL_SHA256 = "822f3b33168b95887c4b0b66fdbb783e8659850b67f8e3f1213a4113fe840d65"
NOTEBOOK_BLOB_SHA = "72d2a86db4209b18afe34b26a130ddce00676f07"
IDENTITY_SHA = "b0c7267b47f6111c5024c9bba8a4135e2357b432"
RESULT_PREFIX = "TASS_IDENTITY_TEST_RESULT="


def cell_from_fixed_notebook(path):
    raw = Path(path).read_bytes()
    blob = b"blob " + str(len(raw)).encode() + b"\x00" + raw
    if hashlib.sha1(blob).hexdigest() != NOTEBOOK_BLOB_SHA:
        raise ValueError("PINNED_NOTEBOOK_BLOB_MISMATCH")
    data = json.loads(raw)
    code_cells = [c for c in data.get("cells", []) if c.get("cell_type") == "code"]
    if len(code_cells) != 1 or code_cells[0].get("id") != "identity-go-packages":
        raise ValueError("APPROVED_CELL_NOT_FOUND_OR_AMBIGUOUS")
    code = code_cells[0].get("source")
    if isinstance(code, list):
        code = "".join(code)
    if not isinstance(code, str) or hashlib.sha256(code.encode()).hexdigest() != CELL_SHA256:
        raise ValueError("PINNED_CELL_HASH_MISMATCH")
    return code


def main():
    args = argparse.ArgumentParser()
    args.add_argument("--cell", choices=["identity-go-packages"], required=True)
    options = args.parse_args()
    notebook = Path(os.environ["TASS_LAB_NOTEBOOK_SOURCE"])
    kernel_path = Path(os.environ["TASS_LAB_CORE_KERNEL"])
    source = Path(os.environ["TASS_ATRIUM_IDENTITY_CHECKOUT"])
    code = cell_from_fixed_notebook(notebook)
    if not source.is_dir() or kernel_path.name != "kernel.py":
        raise ValueError("Missing approved test input")

    # Source is taken from the exact approved checkout. No owner network/Jupyter service.
    module_spec = importlib.util.spec_from_file_location("_atrium_shared_kernel", kernel_path)
    module = importlib.util.module_from_spec(module_spec)
    module_spec.loader.exec_module(module)

    with tempfile.TemporaryDirectory(prefix="atrium-jupyter-") as shared:
        kernel = module.SharedKernel(shared, jupyter_url=None, log=lambda _: None)
        try:
            kernel.start()  # private Jupyter kernel via existing jupyter_client path
            result = kernel.execute(code, timeout=140)
        finally:
            if kernel.client:
                kernel.client.stop_channels()
            if kernel.manager:
                kernel.manager.shutdown_kernel(now=True)
        text = result.get("text", "")
        records = [line[len(RESULT_PREFIX):] for line in text.splitlines()
                   if line.startswith(RESULT_PREFIX)]
        if len(records) != 1:
            # This is delivered only through the parent adapter's redacted,
            # bounded receipt (not dumped directly from the kernel to GitHub).
            print("ATRIUM_NOTEBOOK_DIAGNOSTIC=" + json.dumps({
                "kernel_ok": result.get("ok"),
                "kernel_error_type": str(result.get("error") or "")[:140],
                "text_excerpt": text[-650:],
                "record_lines": len(records),
            }), flush=True)
            raise RuntimeError("Cell did not produce exactly one verifiable Identity test result")
        record = json.loads(records[0])
        if (record.get("repository") != "Arameair/identity"
                or record.get("revision") != IDENTITY_SHA
                or record.get("operation") != "identity_go_packages"
                or not isinstance(record.get("exit_code"), int)
                or not isinstance(record.get("stdout_excerpt"), str)):
            raise RuntimeError("Notebook claimed an unexpected Identity test target/result")
        receipt = {
            "cell_id": options.cell,
            "notebook_blob_sha": NOTEBOOK_BLOB_SHA,
            "cell_sha256": CELL_SHA256,
            "source_sha": IDENTITY_SHA,
            "kernel_mode": kernel.mode,
            "test_exit_code": record["exit_code"],
            "test_output_sha256": hashlib.sha256(text.encode()).hexdigest(),
            "output_excerpt": record["stdout_excerpt"][-2000:],
            "kernel_ok": result.get("ok") is True,
            "kernel_error": str(result.get("error") or "")[:250],
        }
        print("ATRIUM_NOTEBOOK_RESULT=" + json.dumps(receipt, sort_keys=True), flush=True)
        return 0 if receipt["kernel_ok"] and receipt["test_exit_code"] == 0 else 1


if __name__ == "__main__":
    try:
        sys.exit(main())
    except Exception as exc:
        # No tracebacks, paths, environment dumps or private notebook contents.
        print("ATRIUM_NOTEBOOK_FAILURE=" + type(exc).__name__, file=sys.stderr)
        sys.exit(2)
