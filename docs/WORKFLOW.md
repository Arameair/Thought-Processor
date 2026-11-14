# TASS source, review, release, and installation workflow

**Current agreement:** Thought-Processor Issue #119, 2026-11-14.

## Target flow

1. Record one bounded problem and observable acceptance criteria in a primary GitHub issue.
2. Prepare a small issue branch from the approved source base. Preserve exact dependencies and unrelated work.
3. Test untrusted code only in disposable fixtures. Record exact base/head SHAs, changed paths, commands, PASS/FAIL/NOT RUN results, and limitations.
4. Obtain independent Codex review of the final executable SHA. The author does not independently certify their own candidate.
5. Integrate only after required review, CI, branch controls, and risk-appropriate owner authorization.
6. Identify compatible multi-repository releases with immutable pins and a manifest.
7. Install, roll back, change grants, or change autonomy only under separate owner authorization with verified recovery controls.

## Current transition constraints

- The accepted Stage 3 source is an unmerged stacked history; `main`, `alpha`, and `stable` do not presently encode that release.
- Do not flatten, rebase, force-push, close, or merge the stack merely to make GitHub appear tidy.
- Preserve unique test/security/documentation side branches until their content is classified and reviewed.
- Documentation-only work must not alter runtime, test configuration, workflow execution, branch protections, or release behavior without being treated as executable change.
- `INTEGRATED` and `RELEASED` are separate states. Neither implies deployment.

Historical documents that describe `alpha → stable → main` promotion remain useful context but are not the current operating agreement.
