# TASS Core Team experiment: launcher pins only

**Branch:** `experiment/tass-core-team-claude-20261031`, from `alpha` @ `e4b45a9`. `alpha` is untouched.

This branch changes **only the launcher pins**, so Processor can start TASS 0.3.0 (the Core Team thought cycle). Processor's own code is unchanged.

| Setting | alpha | this branch |
| --- | --- | --- |
| `LAUNCH_BRANCH` (run-p1.sh, secure_launch.py) | `alpha` | `experiment/tass-core-team-claude-20261031` |
| `TASS_BRANCH` | `alpha` | `experiment/tass-core-team-claude-20261031` (Arameair/TASS @ `6c3ff01`) |
| `TASS_ARTIFACT` | `543611f8…` (0.2.0) | `f8b76df4104d9f9d0b6129c4836fbad24a7f4fa7cfa6b92c02d83865e52e86c4` (0.3.0) |
| `TASS_PREVIOUS_ARTIFACT` (run-only) | `985027bb…` (0.1.0) | `543611f8…` (0.2.0) |
| Profile folder | `~/p1c/p1-tass-alpha` | `~/p1c/p1-tass-team`: a **separate fresh profile**; the alpha profile is never touched |

Identity (`alpha` @ `a582f8e`) and Thought (`alpha` @ `772200b`) are unchanged.

How the cycle works, what leaves your computer, and the home steps: [Arameair/TASS `docs/TEAM_CYCLE.md`](https://github.com/Arameair/TASS/blob/experiment/tass-core-team-claude-20261031/docs/TEAM_CYCLE.md).

**Checks run here:** the launcher's 22 unit tests pass; `bash -n run-p1.sh` passes; the new digest was computed with the same `package_digest` function as `run-p1.sh`, which reproduces alpha's pinned `543611f8…` from alpha's TASS package. **Not run here:** the full launcher on Windows/WSL, a real Identity Admin enrollment (see #33), or a live cycle.
