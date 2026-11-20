# Synthetic demo results — 2026-11-19

**Status:** local fixture executed and tests passed; live integration NOT RUN. No actual independent Codex security audit was performed. Actor names in the fixture are deterministic *simulations*.

## Inspected repository revisions (not modified)

| Repo/source | Observed revision | Notes |
| --- | --- | --- |
| Thought-Processor `main` | `53e0857d11efe2a5eb2e7903c43cd226faa9d9c6` | Base for this separate experiment branch |
| Thought-Processor PR #130 | `2d7cf48b5956ee49cb1408a487209c2260a00b7c` | Open; docs-only workflow proposal |
| TASS PR #27 | `36fd659f8f90e2f5986ac0a8a32bbb79b36afcae` | Open; mailbox is NOT CONNECTED |
| TASS `live` tree | `df5c3cca1e9ddc0b4100f12dea7b62c031d4f676` | Tree SHA, not a commit; private identity-doc branch |
| Core GitHub client blob | `46b2a30d93776d5678cdb9aea21214ba985bb0f6` | Conditional reads/writes, issue operations |
| Core relay blob | `c0fb3738f5a7d37c3b68e0d196e2679f4eb3b1fa` | Issue #24 pilot, not team mailbox adapter |

## Actual local commands and observations

Environment: Linux container, Python 3.13.5. No model, network, or GitHub interaction during fixture execution.

```sh
python3 workloop.py --out records/run
# DEMO PASS; 10 synthetic event records written
python3 -m unittest discover -s tests -v
# Ran 10 tests; OK
```

| Check | Result | Evidence |
| --- | --- | --- |
| WS-A first smoke case | PASS | `normalize_v1.py`, one checked case |
| WS-A independent regression case | **FAIL, expected** | Double-space preservation violates acceptance; state returns to NEEDS_REPAIR |
| WS-A repair smoke + regression | PASS | `normalize_v2.py`, both checks pass |
| WS-A re-audit of new SHA-256 | PASS | Exact digest checked, old digest rejected |
| WS-B independent candidate audit | PASS | `status.py`, 3 audit cases |
| Actual two local threads write separate receipt paths | PASS | `WS-A/Claude.md`, `WS-B/Codex.md` |
| Test suite | PASS | 10/10 tests, including stale version, role, privacy keyword and unauthorized TASS checks |
| Live Core/Identity/TASS GitHub integration | NOT RUN | No grants or services touched |
| Existing repository regression/CI suites | NOT RUN | Not part of this isolated experiment |
| Actual Claude/Codex model participation | NOT RUN | Deterministic role names are simulated |
| Merge, deployment, PR, GitHub grants | NOT RUN | Explicitly excluded |

## Fixture source SHA-256 values

- Buggy candidate `fixtures/normalize_v1.py`: `6a8815f0189dceb51ed0d0702396adbffd870e35b4108059799fb8e2b228840d`
- Repaired candidate `fixtures/normalize_v2.py`: `89df20e7db340b5141f251f2a4545133a071b7eb0df2c749f7d8af31e0cb05ad`
- Separate WS-B candidate `fixtures/status.py`: `f0402d446a58f7e577303b55d8315ad337548b15a3a668bfd44b96b7d458c50b`
- Runner `workloop.py`: `b0852df1e39950aaee1f8957f7e56dff68ad5eb8ededbadd8da495f021b12ea8`
- Tests `tests/test_workloop.py`: `ed94cf4bc9e43fdc9139a79cf65509b13f57fcb1e662ca3c854c8362baaa89c9`

These SHA-256 hashes are **file-content digests**, not GitHub commit SHAs. See `records/BEFORE.md`, `AFTER.md`, `EVENTS.md` and generated per-actor handoffs for event-level evidence. A Git branch containing this fixture is *not* a passing production audit.

## Next smallest authorized implementation slice

Read-only observer only: under an explicitly approved Core issue, use existing scoped GitHub read APIs to fetch **one** approved task at an exact revision and show a sanitized `ACK_PROPOSED` locally. Test wrong author, stale revision, missing Identity grant, malformed content, no-output on no change and private-data exclusion. Do not wire GitHub publication, scheduler, auto-repair or autonomous task execution yet. Independent audit, followed by a separate owner grant decision, is required before any real outbound collaboration.
