# Thought Processor source status

**Reviewed:** 2026-11-14. This status describes source reconciliation, not a release.

## Accepted owner baseline

| Component | Accepted source SHA | Evidence state |
| --- | --- | --- |
| Core | `8e2b02b17eb4e39d489a564af23588be22bb5d19` | Independently audited, installed, owner accepted Stage 3 |
| Engine | `3aeaff7a41a989414ee1cd0ec64afb51f27e9876` | Compatible accepted pin |
| Identity | `ee7ba1b0f9cf894235be63071c493b74e2a0eef9` | Compatible accepted pin |

The owner installation and its recovery controls are separate from repository branch state. The accepted Stage 3 Core is still represented by an open draft PR stack and is not equivalent to current `main`.

## Current repository lanes

- `main`: divergent documentation-only history rooted before the Go Stage 3 runtime.
- `stable`: historical documentation snapshot; not the accepted Stage 3 release lane.
- `alpha`: older runnable history; not the accepted Stage 3 source anchor.
- PRs #110 → #112 → #113 → #114 → #117 → #121 → #122 → #123: accepted Core dependency chain.
- PR #115: diverged Stage 1 bounded-load harness and evidence that must be preserved separately.
- PR #118: draft governance proposal; not active policy unless reconciled with Issue #119 and separately approved.

## Reconciliation status

The default branch contributes unique historical documentation but no default-only runtime implementation. A disposable merge rehearsal found content conflicts in the top-level README and this status file, plus semantic drift in branch/release claims. Any real integration requires a final reviewed documentation resolution, focused tests, an independent exact-SHA audit, and separate owner authorization before refs move.

No candidate described here is a public release or an independently certified integration.
