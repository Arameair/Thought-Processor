# TASS hourly readiness: isolated acceptance fixture

**Status: experimental, offline, non-authoritative. NOT integrated, not deployed.**

Purpose: test the *preparation* eligibility rules of the existing hourly scheduled task without introducing another coordinator, ledger, daemon, GitHub writer, or Identity authority. This exercise is independent of the in-flight D2 security candidate and S2-A PR #134.

## Reproduce

From this directory, run:

```sh
python3 -m unittest discover -s . -p 'test_*.py' -v
```

This package uses only Python standard library. Its sole `READY_FOR_DRAFT` case is **synthetic**; no real owner approval is asserted. The fixture produces no side effects.

## Trust boundary / what this does not prove

`owner_approval_verified`, `exact_scope_verified`, `assignment` and `lease_confirmed` are test inputs, not verified here. A real component must first fetch current canonical GitHub records, validate genuine owner authorization from a trusted independent source, confirm exact scope and paths, verify repository/branch revision and workstream ownership, establish an atomic claim/lease against concurrent workers, and fail closed on conflicts or inaccessible evidence. A Markdown statement, PR comment, agent self-report or boolean assembled from untrusted text must not establish authority. This is not a security control.

This fixture only permits files under its own experiment directory; it cannot touch D2, Identity, Core workflow, private data, main, or a current audit candidate. It never writes to GitHub, invokes Claude or Codex, runs model tests, or approves a merge. A production code change requires a new scoped owner decision, relevant S2 design acceptance, dedicated tests and independent Codex review.

## Handoff

**Claude (implementation/repair):** Compare the fixture to the actual hourly agent's readiness language and the existing Core workstream S2-A design; identify how to reuse trusted provenance and avoid a second ledger. Do not integrate this test into production until the corresponding stage and authority are accepted.

**Codex (independent tester):** Check the fail-closed cases, wrong-repo/forged ownership/stale SHA handling and whether this fixture's synthetic proof is misrepresented as runtime assurance. Audit the exact draft PR head if it is published. Owner controls merge and protected operations.

**DOC SYNC:** TEAM, WORKFLOW, SECURITY and PROJECT are reviewed without change. Neither an experiment nor passing synthetic tests establishes accepted source, installed state or granted authority.
