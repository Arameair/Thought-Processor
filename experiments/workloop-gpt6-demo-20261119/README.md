# GPT-6 synthetic TASS development loop demonstration

**Scope:** isolated/offline proof of concept, 2026-11-19. No real Claude, Codex, TASS, Identity, GitHub synchronization, service deployment, permission grant, merge or PR activity is performed by this program. Participant names are **simulated roles**; Codex has not independently audited this program.

## Run

Requires only Python 3.10+ and a local writable directory. From this folder:

```sh
python3 workloop.py --out records/run
python3 -m unittest discover -s tests -v
```

Read `records/run/BEFORE.md`, `AFTER.md`, `EVENTS.md`, and `handoffs/WS-*/<actor>.md` for the generated Markdown records. The checked-in `records/BEFORE.md`, `AFTER.md` and `EVENTS.md` are a reproducible snapshot of one run.

## What the fixture exercises

- **WS-A:** Claude role builds a deliberately incomplete `normalize_v1.py`. A real smoke check passes. Codex auditor role runs a separate regression that **FAILS**; the handoff is `NEEDS_REPAIR`. Claude role installs `normalize_v2.py`, retests, then Codex role independently re-audits that exact SHA-256 revision **PASS**.
- **WS-B:** a distinct previously tested `status.py` candidate is independently audited **PASS** by the Codex role. The A builder and B auditor use two actual local worker threads and distinct actor-owned receipt files; their interleaving is not synchronized live with anyone.
- **GPT-6:** coordinator role writes task summaries, never executable authority or accepted production status.
- **TASS-sim:** reads a synthetic assignment and emits `ACK_PROPOSED / NOT SENT`; it cannot build, authorize, or publish.

## Safety limitations

These are **trusted checked-in fixture Python files**, not a sandbox for arbitrary GitHub Markdown, untrusted branches or model-generated code. The compare-and-swap guard is process-local, not durable across computers or crashes; production requires GitHub blob-SHA conditional writes, retry limits, durable event/receipt keys, authentication, Identity authorization and fail-closed delivery. Receipt privacy filtering here is illustrative and not a security boundary. No test exercises the running local TASS installation.

See [coordination design](docs/COORDINATION.md) and [test evidence](docs/DEMO_RESULTS.md). The experiment lives only in a separate branch of `Arameair/Thought-Processor`; it does not change [PR #130](https://github.com/Arameair/Thought-Processor/pull/130) or [TASS PR #27](https://github.com/Arameair/TASS/pull/27).
