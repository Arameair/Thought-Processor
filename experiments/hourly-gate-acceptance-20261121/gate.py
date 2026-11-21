"""Read-only synthetic fixture for TASS hourly-build readiness.

Not a GitHub client, permission authority, workstream scheduler, or production gate.
The caller MUST verify approval/provenance independently; strings read from
Markdown, model output, or task descriptions NEVER constitute trusted approval.
"""
from dataclasses import dataclass
import re
from pathlib import PurePosixPath

FULL_SHA = re.compile(r"[0-9a-f]{40}\Z")
REPO = re.compile(r"[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+\Z")
TARGET_REPO = "Arameair/Thought-Processor"

# Only this synthetic fixture path is eligible in the first trial; expansion
# requires separately reviewed policy. Actual file contents must still be reviewed.
FIXTURE_ROOT = "experiments/hourly-gate-acceptance-20261121/"


@dataclass(frozen=True)
class Candidate:
    repository: str
    issue_number: int
    base_sha: str
    observed_base_sha: str
    paths: tuple[str, ...]
    # These signals MUST come from an external, authenticated authority/record.
    owner_approval_verified: bool
    assignment: str  # unclaimed | hourly-gate | other
    exact_scope_verified: bool
    audit_in_progress: bool = False
    existing_draft: bool = False
    lease_confirmed: bool = False
    action: str = "draft"  # never permit merge, deploy, or grant
    untrusted_markdown_claim: str = ""  # deliberately ignored


@dataclass(frozen=True)
class Decision:
    status: str
    reasons: tuple[str, ...]


def _is_fixture_path(path: str) -> bool:
    if not path or "\\" in path or path.startswith("/") or "//" in path:
        return False
    # Normalize to eliminate traversal / confusing segments.
    if any(p in (".", "..", "") for p in path.split("/")):
        return False
    if PurePosixPath(path).as_posix() != path:
        return False
    return path.startswith(FIXTURE_ROOT) and len(path) > len(FIXTURE_ROOT)


def evaluate(candidate: Candidate) -> Decision:
    """Fail closed. READY means *proposal eligible*, never authenticated grant.

    This pure function does not fetch GitHub, authenticate the owner, acquire
    a workstream lease, run tests, modify files or authorize unattended writes.
    """
    reasons: list[str] = []
    if not REPO.fullmatch(candidate.repository) or candidate.repository != TARGET_REPO:
        reasons.append("invalid-repository")
    if candidate.issue_number <= 0:
        reasons.append("missing-issue")
    if not FULL_SHA.fullmatch(candidate.base_sha) or not FULL_SHA.fullmatch(candidate.observed_base_sha):
        reasons.append("invalid-sha")
    elif candidate.base_sha != candidate.observed_base_sha:
        reasons.append("stale-base")
    if not candidate.owner_approval_verified:
        reasons.append("approval-unverified")
    if candidate.assignment not in ("unclaimed", "hourly-gate"):
        reasons.append("claimed-or-invalid-assignment")
    if not candidate.exact_scope_verified:
        reasons.append("scope-unverified")
    if candidate.audit_in_progress:
        reasons.append("audit-in-progress")
    if candidate.existing_draft:
        reasons.append("duplicate-draft")
    if not candidate.lease_confirmed:
        reasons.append("claim-unverified")
    if candidate.action != "draft":
        reasons.append("protected-action")
    if not candidate.paths or any(not _is_fixture_path(path) for path in candidate.paths):
        reasons.append("path-out-of-scope")
    # Never consume untrusted_markdown_claim for authorization.
    return Decision("BLOCKED" if reasons else "READY_FOR_DRAFT", tuple(reasons))
