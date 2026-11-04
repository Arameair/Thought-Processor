"""One notebook-cell adapter for PR #52's existing pulse controller.

No independent scheduler, model authorization or arbitrary Jupyter execution.
The existing two Go cards still use LocalTestAdapter unchanged. The new card is
routed to a fixed worker running a PRIVATE jupyter_client kernel in a sandbox.
"""
from __future__ import annotations

import hashlib
import json
import os
import resource
import signal
import subprocess
import sys
import tempfile
import time
from pathlib import Path

from pulse_cards import CATALOG, CardError, LocalTestAdapter, NOTEBOOK_SHA, TIMEOUT, redact, utc

LESSON_ID = "NOTEBOOK-IDENTITY-LAB"
SKILL = "@identity-test"
WORKER = Path(__file__).with_name("notebook_cell_worker.py")


def resolve_skill(name: str) -> dict:
    """A fixed name lookup only. NEVER a permission, run request or action grant."""
    if type(name) is not str or name != SKILL:
        raise CardError("Unknown or ambiguous @skill name", 404)
    spec = CATALOG[LESSON_ID]
    return {
        "type": "notebook_skill_candidate", "name": SKILL, "state": "CANDIDATE_UNAPPROVED",
        "card_id": LESSON_ID, "repository": spec["repository"],
        "sha": spec["sha"], "notebook_path": spec["notebook_path"],
        "notebook_blob_sha": spec["notebook_blob_sha"], "cell_id": spec["cell_id"],
        "cell_sha256": spec["cell_sha256"],
        "required_capability": spec["capability"],
        "requires_owner_approval": True, "execution_enabled_by_lookup": False,
    }


def lesson_prompt() -> str:
    """Teach TASS which one cell exists. This string is data, not authority."""
    ref = resolve_skill(SKILL)
    return (
        "TASS, your teacher has given you Lesson 01: inspect the one pinned Identity notebook.\n"
        "Only the fixed, owner-approved cell 'identity-go-packages' is available.\n"
        f"Notebook Git revision: {ref['sha']}; notebook blob: {ref['notebook_blob_sha']}.\n"
        "Its Python cell runs the exact approved Go authority, pairing and entities tests "
        "from the pinned Identity checkout; it reports exit code and actual package output.\n"
        "Choose a cell and predict its observable result. Do not execute anything directly.\n"
        'Reply with ONLY JSON: {"skill":"@identity-test","cell_id":"identity-go-packages",'
        '"expectation":"<specific prediction in your own words>"}.\n'
        "Your selection is a PROPOSAL. The trusted gate will still require a separate owner approval."
    )


def validate_model_selection(raw, *, actor="MODEL_FIXTURE"):
    """Parse bounded model selection; never converts model text into executable code."""
    if not isinstance(raw, str) or len(raw) > 1200:
        raise CardError("Model selection must be bounded JSON", 400)
    try:
        selected = json.loads(raw)
    except ValueError as exc:
        raise CardError("Model selection is not valid JSON", 400) from exc
    if not isinstance(selected, dict) or set(selected) != {"skill", "cell_id", "expectation"}:
        raise CardError("Model selection must have exactly three fields", 400)
    ref = resolve_skill(selected["skill"])
    if type(selected["cell_id"]) is not str or selected["cell_id"] != ref["cell_id"]:
        raise CardError("Selected cell is not allowlisted", 403)
    prediction = selected["expectation"]
    if not isinstance(prediction, str) or not 12 <= len(prediction.strip()) <= 320:
        raise CardError("Model must explain the expected result before requesting execution", 400)
    if actor not in {"MODEL_FIXTURE", "NEMOTRON_UNVERIFIED"}:
        raise CardError("Invalid untrusted actor label", 400)
    return {
        "actor": actor, "cell_id": ref["cell_id"], "skill": SKILL,
        "expected_result": prediction.strip(),
        "source_sha": ref["sha"], "cell_sha256": ref["cell_sha256"],
        "execution_authorized": False,
    }


def propose_skill_candidate(receipt: dict, *, selection: dict, reflection: str, repeats=None):
    """Construct a PROPOSED skill only after an evidence-backed PASS; never approve it."""
    spec = CATALOG[LESSON_ID]
    if (not isinstance(receipt, dict) or receipt.get("card_id") != LESSON_ID
            or receipt.get("sha") != NOTEBOOK_SHA
            or receipt.get("cell_id") != spec["cell_id"]
            or receipt.get("cell_sha256") != spec["cell_sha256"]
            or receipt.get("notebook_blob_sha") != spec["notebook_blob_sha"]
            or receipt.get("operation") != spec["operation"]
            or receipt.get("outcome") != "PASS" or receipt.get("executed") is not True
            or receipt.get("exit_code") != 0
            or not receipt.get("output_sha256") or not receipt.get("receipt_id")):
        raise CardError("Cannot propose a skill without an actual matching PASS receipt", 403)
    if not isinstance(selection, dict) or selection.get("cell_id") != spec["cell_id"]:
        raise CardError("Skill proposal needs a validated model selection", 403)
    if not isinstance(reflection, str) or not 12 <= len(reflection.strip()) <= 500:
        raise CardError("Skill proposal needs an attributable interpretation", 403)
    # A successful fixture demonstration is not proof of Nemotron competence.
    return {
        **resolve_skill(SKILL),
        "state": "PROPOSED_FOR_OWNER_REVIEW",
        "execution_enabled_by_lookup": False,
        "lesson_selection_actor": selection["actor"],
        "lesson_expectation": selection["expected_result"][:320],
        "reflection": reflection.strip(),
        "reflection_actor": selection["actor"],
        "verification_receipts": [receipt["receipt_id"], *(repeats or [])][:8],
        "first_receipt_digest": receipt["output_sha256"],
        "learned_by_live_nemotron": False,
        "review_note": "Notebook skill definition only. Requires repeat verification, owner review and live Identity permission.",
    }


class NotebookTestAdapter(LocalTestAdapter):
    """Reuse PR52's preflight on exact Git checkouts and its one-shot receipt flow."""

    def preflight(self, card):
        if card.get("id") != LESSON_ID:
            return super().preflight(card)
        # Both independent inputs are checked as exact clean source trees.
        note_dir = super().preflight(card)
        identity_dir = super().preflight(CATALOG["IDENTITY-014"])
        spec = CATALOG[LESSON_ID]
        notebook = Path(self.roots[LESSON_ID]) / spec["notebook_path"]
        if notebook.is_symlink() or not notebook.is_file():
            raise CardError("Approved notebook is missing or linked", 403)
        raw = notebook.read_bytes()
        if len(raw) > 200_000:
            raise CardError("Notebook file exceeds approved bound", 403)
        blob = b"blob " + str(len(raw)).encode() + b"\x00" + raw
        if hashlib.sha1(blob).hexdigest() != spec["notebook_blob_sha"]:
            raise CardError("Notebook hash mismatch; no Jupyter kernel started", 403)
        try:
            nb = json.loads(raw)
            cells = [c for c in nb["cells"] if c.get("cell_type") == "code"]
            cell = cells[0] if len(cells) == 1 else None
            code = cell.get("source") if cell else None
            if isinstance(code, list):
                code = "".join(code)
            if (not cell or cell.get("id") != spec["cell_id"]
                    or not isinstance(code, str)
                    or hashlib.sha256(code.encode()).hexdigest() != spec["cell_sha256"]):
                raise ValueError("cell ID/content mismatch")
        except (ValueError, KeyError, TypeError, IndexError) as exc:
            raise CardError("Notebook contains no uniquely approved cell", 403) from exc
        kernel_module = Path(self.roots[LESSON_ID]) / "workspace/tass_workspace/kernel.py"
        if not kernel_module.is_file() or kernel_module.is_symlink():
            raise CardError("Pinned Workshop SharedKernel not available", 403)
        if not WORKER.is_file() or WORKER.is_symlink():
            raise CardError("Reviewed notebook worker unavailable", 403)
        return {"notebook": notebook, "identity": identity_dir, "kernel": kernel_module,
                "source_dir": note_dir}

    @staticmethod
    def _kernel_limits():
        # Jupyter/IPython imports and kernel sockets need more than PR52's
        # Go-only 64-descriptor limit; keep a bounded 512 FD ceiling instead.
        resource.setrlimit(resource.RLIMIT_CPU, (TIMEOUT, TIMEOUT + 10))
        _, hard = resource.getrlimit(resource.RLIMIT_NOFILE)
        cap = 512 if hard == resource.RLIM_INFINITY else min(512, hard)
        resource.setrlimit(resource.RLIMIT_NOFILE, (cap, cap))

    def execute(self, card, job_id, cancel=None):
        if card.get("id") != LESSON_ID:
            return super().execute(card, job_id, cancel=cancel)
        spec = CATALOG[LESSON_ID]
        started = utc()
        trusted = self.preflight(card)  # verify revision+cell again before any kernel starts
        with tempfile.TemporaryDirectory(prefix="atrium-notebook-job-") as d:
            root = Path(d)
            env = {
                "PATH": os.environ.get("PATH", os.defpath),
                "HOME": d, "TMPDIR": d, "JUPYTER_CONFIG_DIR": str(root / "jupyter-config"),
                "JUPYTER_RUNTIME_DIR": str(root / "runtime"),
                "IPYTHONDIR": str(root / "ipython"),
                "GOCACHE": str(root / "go-cache"), "GOMODCACHE": str(root / "mod-cache"),
                "GOPROXY": "off", "GOSUMDB": "off", "GOENV": "off", "GOTOOLCHAIN": "local",
                "GIT_TERMINAL_PROMPT": "0", "GIT_CONFIG_NOSYSTEM": "1",
                "GIT_CONFIG_GLOBAL": "/dev/null", "PYTHONNOUSERSITE": "1",
                "TASS_LAB_NOTEBOOK_SOURCE": str(trusted["notebook"]),
                "TASS_LAB_CORE_KERNEL": str(trusted["kernel"]),
                "TASS_ATRIUM_IDENTITY_CHECKOUT": str(trusted["identity"]),
                "LANG": "C.UTF-8", "LC_ALL": "C.UTF-8",
            }
            for location in ("jupyter-config", "runtime", "ipython", "go-cache", "mod-cache"):
                (root / location).mkdir()
            with tempfile.TemporaryFile(mode="w+b") as output:
                p = subprocess.Popen(
                    [sys.executable, "-I", str(WORKER), "--cell", spec["cell_id"]],
                    cwd=d, env=env, stdin=subprocess.DEVNULL,
                    stdout=output, stderr=subprocess.STDOUT, shell=False,
                    start_new_session=True, preexec_fn=self._kernel_limits,
                )
                cancelled, timeout, oversized = False, False, False
                deadline = time.monotonic() + TIMEOUT
                while p.poll() is None:
                    if cancel is not None and cancel.is_set():
                        cancelled = True
                        break
                    if time.monotonic() >= deadline:
                        timeout = True
                        break
                    output.seek(0, os.SEEK_END)
                    if output.tell() > 2 << 20:
                        oversized = True
                        break
                    try:
                        p.wait(timeout=.2)
                    except subprocess.TimeoutExpired:
                        continue
                if timeout or cancelled or oversized:
                    try:
                        os.killpg(p.pid, signal.SIGKILL)
                    except ProcessLookupError:
                        pass
                    p.wait(timeout=10)
                output.seek(0)
                raw = output.read((2 << 20) + 1)
        digest = hashlib.sha256(raw).hexdigest()
        lines = raw.decode("utf-8", "replace").splitlines()
        records = [line[len("ATRIUM_NOTEBOOK_RESULT="):]
                   for line in lines if line.startswith("ATRIUM_NOTEBOOK_RESULT=")]
        parsed = None
        if len(records) == 1:
            try:
                parsed = json.loads(records[0])
            except ValueError:
                pass
        exact = (isinstance(parsed, dict)
                 and parsed.get("cell_id") == spec["cell_id"]
                 and parsed.get("notebook_blob_sha") == spec["notebook_blob_sha"]
                 and parsed.get("cell_sha256") == spec["cell_sha256"]
                 and parsed.get("source_sha") == CATALOG["IDENTITY-014"]["sha"]
                 and parsed.get("kernel_mode") == "private"
                 and type(parsed.get("test_exit_code")) is int)
        if not exact or cancelled or timeout or oversized:
            status = "ERROR"
        else:
            status = "PASS" if p.returncode == 0 and parsed["test_exit_code"] == 0 and parsed.get("kernel_ok") is True else "FAIL"
        excerpt = (parsed.get("output_excerpt", "") if exact else "\n".join(lines[-6:]))
        reason = ("Stopped by owner" if cancelled else
                  "Notebook timed out" if timeout else
                  "Notebook output oversized" if oversized else
                  "Notebook did not provide one matching Jupyter execution receipt" if not exact else
                  "Go package test failed in Jupyter cell" if status == "FAIL" else None)
        return {
            "receipt_id": "receipt-" + __import__("uuid").uuid4().hex,
            "job_id": job_id, "card_id": LESSON_ID, "operation": card["operation"],
            "repository": card["repository"], "sha": card["sha"],
            "argv": list(card["argv"]), "started_at": started,
            "finished_at": utc(), "exit_code": None if (timeout or cancelled) else p.returncode,
            "outcome": status, "executed": True, "environment": "ISOLATED_PRIVATE_JUPYTER_KERNEL",
            "output_sha256": digest, "output_excerpt": redact(excerpt),
            "reason": reason,
            "cell_id": spec["cell_id"], "cell_sha256": spec["cell_sha256"],
            "notebook_blob_sha": spec["notebook_blob_sha"],
        }
