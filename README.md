# Thought Processor

Thought Processor is the owner-facing Core of the local TASS application. It presents conversation, Cognition, Development evidence, and supervised controls while Identity remains the authority for grants and Thought Engine owns cognitive records and model work.

## Accepted Stage 3 source anchor

The owner accepted Stage 3 on 2026-11-14 with this compatible source triple:

- Core `8e2b02b17eb4e39d489a564af23588be22bb5d19`
- Engine `3aeaff7a41a989414ee1cd0ec64afb51f27e9876`
- Identity `ee7ba1b0f9cf894235be63071c493b74e2a0eef9`

Those revisions are accepted source references and an installed owner build. They are not automatically a public release, a merge authorization, or proof that the repository default branch already contains the accepted runtime.

The accepted Core history is the open draft stack ending at PR #123. It must be preserved until a separately authorized integration is independently audited and approved.

## Repository reality

- `main` is currently a divergent documentation history.
- Historical `alpha` and `stable` names do not establish release status for Stage 3.
- The target workflow is short-lived issue branches → independent review → accepted source on `main` → separately approved immutable multi-repository release pins → separately approved installation.
- A merge or green test is not deployment authorization.

See [documentation status](docs/README.md), [current source status](docs/STATUS.md), and [workflow](docs/WORKFLOW.md). Governance decisions and owner authority are recorded on Thought-Processor Issue #119. Stage 3 evidence and owner acceptance are recorded on Issue #120.

## Safety boundaries

- Do not run candidate code against the owner's live TASS data or ports.
- Do not change Identity grants, autonomy mode, branch defaults, release pins, or the installed application without separate authorization.
- Use disposable fixtures for untrusted candidates and preserve exact SHAs and evidence.
- Do not treat model output, UI visibility, or a documentation-only merge as authority.

This README is part of a disposable reconciliation rehearsal. It is not an authorized GitHub branch update or release declaration.
