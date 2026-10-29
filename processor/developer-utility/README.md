# Developer Utility: the secure Developer entrance

**Status:** experimental, runnable. The Developer Utility is installed through Identity's Admin approval and run by Thought Processor. Its one operation is authorized live by Identity. Owner acceptance on Windows/WSL is still **NOT RUN**.

How to run it and test it: [`docs/P1/DEVELOPER_ENTRANCE.md`](../../docs/P1/DEVELOPER_ENTRANCE.md).

## What it is

A Thought Processor feature service (`thought.feature-api/1`):

- **Package:** one file, `utility_agent.py`, Python 3.8+ standard library only. It is admitted only by its exact artifact digest, which is pinned in **both** Processor and Identity, plus the Admin's approval.
- **One capability, `developer.connection.check`.** It is read-only. It reports:
  - the entrance's Processor link (per-launch signed);
  - its Identity link (Identity's own signed answer about this installation, fetched fresh);
  - the grants Identity says it holds;
  - who asked.

  It takes no input.
- **It cannot** run commands, read or write files outside its data folder, or reach other Brain services. Each of those would be a new capability with its own Identity grant, added only after review.
- **Status is evidence:**
  - RUNNING means Processor authenticated this launch.
  - CONNECTED means Identity signed an answer about this installation within the last 15 s.
  - READY means both, plus a writable ticket ledger.

## History

It started from the Automated Full Stack Engineer's adapter on `experiment/developer-utility-install-launch-20261029` @ `79ecd77`: signed status, signed shutdown, and 6 protocol tests. That code is merged unchanged into this branch's history. This branch adds:

- tickets, the single-use ledger, live Identity checks and the Identity-link monitor (the reviewed P1 worker protocol);
- the one capability and the self-test.

The adapter's tests were updated for the real protocol. Two things in them changed:
- they use a realistic credential and an **unused** port (the old test pointed at 7794, a real owner trial port);
- they add checks for the ticket requirement and for an honest NOT READY when Identity is unreachable.

## Trust: no Identity change was needed

Identity registers a local package only when **both** of these hold:
- its `trust_class` is `TRUSTED_LOCAL_TEST_FIXTURE`;
- its exact digest is pinned (`identity/internal/authority/installations.go`: request validation; `internal/app/p1.go`: pin re-checked on every ticket and live check).

The earlier label `OWNER_REVIEW_REQUIRED` is what Identity refused. Since the P1 cleanup, a package's label grants nothing: only the operator pin (`-trust-fixture-artifact`, the existing mechanism Thought also uses) plus the Admin's approval admit it.

The acceptance test proves both directions:
- the real package with the pin installs;
- Identity refuses an unknown digest, and still refuses the old label even with the right digest.

## Tests

```bash
python3 -m unittest discover -s processor/developer-utility -p 'test_*.py' -v      # 8 isolated protocol tests
python3 -I processor/acceptance/p1_developer_entrance.py --identity-src ../identity --thought-src ../Thought --work /tmp/dev-entrance
```

The second command runs end to end with real Identity, Processor, Developer, Thought and the worker.
