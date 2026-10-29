# Developer Utility — isolated lifecycle adapter

**STATUS: INTEGRATION BLOCKED.** This is an experimental checkpoint, not a runnable Identity-approved Developer Utility or the Windows Brain Bar.

## Source and boundaries

- This work starts from Thought Processor launcher commit `45a6af30cc411166300608003aebba9375730262`. The existing P1 branch, profiles and PR #27 are not modified.
- Reference Brain Bar candidate: `Arameair/The_Brain`, branch `experiment/brain-utility-alpha-claude-20261027`, commit `4b324f6b1344c965b964ce76d10900289c8dd1c1`. Its Windows Forms `install.ps1` / `brain-bar.ps1` remain separate; this adapter does **not** start them.
- Protocol follows existing P1 `docs/P1/FEATURE_API_V1.md` and the deterministic-worker signing scheme. No proprietary TheBrain application source is copied.

## Implemented

`utility_agent.py` provides a Python standard-library loopback HTTP feature endpoint with:
- signed `GET /v1/status`, `GET /v1/capabilities`, `GET /v1/connections`, and `POST /v1/shutdown`;
- process-bound launch key, matching installation-credential fields and atomic endpoint discovery;
- replay and invalid-MAC denial, without any privileged actions.

**Status is intentionally unambiguous:** `ready=false`, `utility_state=NOT_INTEGRATED`, `brain_bar_state=NOT_LAUNCHED`, `authority=NOT_VERIFIED`. An adapter PID is never represented as a real Windows Brain Bar process.

### Local tests (isolated, synthetic credentials)

```bash
python3 -m unittest discover -s processor/developer-utility -p 'test_*.py' -v
```

The six adapter tests ran in the engineering scratch environment with PASS. They do **not** establish an Identity-approved installation, WSL/Windows handoff, real browser UI or owner acceptance. No secret values are required for these tests.

## Required decisions / missing functionality

1. **Identity policy:** Existing P1 independently refuses non-fixture registration. This package declares `OWNER_REVIEW_REQUIRED` and ships with no trusted artifact pin. Do not re-label the package as a fixture, pin only the Processor side, or bypass Admin approval. Project Lead/Dustin must approve an exact owner-utility trust policy before real installation can be called a PASS.
2. **Host lifecycle:** Need an explicitly allowlisted and source-pinned Windows launcher plus proof of ownership, readiness and stop of the **specific** Brain Bar process. Must not take over or kill an existing bar or TheBrain desktop session.
3. **Windows/WSL guide and tests:** After both gates, provide an isolated-clone install → Identity Admin approve → launch → RUNNING → stop → STOPPED walkthrough, denial/revocation tests, and exact SHA. Do not use the existing `~/p1c` working trial.

**NOT RUN:** Identity install/authorization, real Windows Brain Bar start/stop, owner WSL test and P1 regression tests.

No Project Controller dependency. No additional features, source promotions, PR or merges.
