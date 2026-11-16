# One-app Docker startup hardening — GPT-6 independent slice, 2026-11-04

**Branch:** `experiment/one-app-startup-hardening-gpt6-20261104`, based on Claude's unmerged PR #53 `experiment/one-app-20261104` SHA `84583b23ddeced339d98a73af374b41f73414842`.
**Scope:** Fix the reported sudo Compose interpolation and build errors without touching the owner's local WSL, Docker containers, running Nemotron, source modifications, Identity grants, or existing saved data. No PR merge.

## Exact source diagnosis

1. `core/tass.sh:compose()` originally prepended temporary environment assignments to `"${DOCKER[@]}" compose ...`. When Docker requires `sudo -n docker`, typical `sudo env_reset` drops these variables. Compose then reports `CORE_IMAGE`, `ENGINE_IMAGE`, `APP_DATA`, `APP_SECRETS`, `MODEL_ARGS` and other required fields missing.
2. `core/tass.sh:src_for()` used an `eval` of `gh repo clone Arameair/Thought-engine -- -q --no-checkout` with a dynamically appended destination, which is not a reliable/predictable Git checkout. The owner had already reproduced a nonexistent Engine clone directory.
3. `cmd_start()` called `build_version ... && mv ...` in an error-tolerant shell condition, allowing failures from inside `build_version` to be skipped. The resulting pointer might be missing or stale. A previous valid pointer must stay unchanged on failure.
4. Separately, Issue #50 demonstrates **two P0 Core integrity defects**: a model action can execute on unlocked prediction and mismatched Engine cycle ID. No task/pulse/autonomous action is safe merely because three containers are healthy. An ordinary chat greeting may still induce a model-proposed action if unrestricted Core is unlocked.

## Patch and intended behavior

- Compose vars now appear **after sudo** via `sudo -n env KEY=value ... docker compose ...`, or `env KEY=value ... docker compose ...` without sudo. Only explicit interpolation values, not the caller's full environment, are forwarded. The user-selected ports 8797/8796 and external model network/URL are carried through the same path.
- Direct `git clone -q --no-checkout` is used for the two fixed upstream URLs or optional `ENGINE_SOURCE`/`IDENTITY_SOURCE` paths. The *source* directories are not altered: local sources are cloned into a dedicated build cache inside `APP_HOME/deploy/src`. Every checkout and image build fails explicitly, and only three successful images yield a candidate deployment record. A failed candidate never overwrites `current.env`.
- `cmd_start` refuses to start a saved Core revision that does not match the safety-reviewed checkout HEAD. It does not auto-update, delete or replace that saved record. An explicit reviewed `update` is the existing separate activation path.
- In the **newly compiled one-app Core** (`IdentitySvc` configured), `ChatOnly` is enforced with no runtime opt-out in this slice. Nemotron gets no actionable tool list. Even if Engine returns an action anyway, Core records `NOT_AUTHORIZED` without executing. `SubmitTask` rejects and `PulseControl` rejects Start/Resume/Step; emergency Pause is allowed. The health JSON includes `action_mode: CHAT_ONLY`; `tass.sh validate` requires it, and `tass.sh status` prints it.
- A failed explicit update when **no prior tass-app containers were running** restores the old record/data without launching an old, unguarded Core image. Existing old `tass-workspace-*` containers are not managed by the `tass-app` project.
- This is a **temporary enforcement boundary**, not a correction to the missing prediction-lock/cycle-ID validations. The full strict independent Issue #50 tests still need to pass before enabling actual actions or the 20-second work pulse. PR #53's clean-host 37 PASS fixture evidence remains historical evidence for its old pulse-capable build, **not** a passing pulse result for this chat-only branch.

## Regression coverage / current truth

`core/test/test_launcher_regression.py` launches the actual `core/tass.sh` in an isolated temp `APP_HOME` with fake `docker`, `sudo`, `git clone` and a fixture environment reset. It checks:

- `CORE_IMAGE`, `ENGINE_IMAGE`, `APP_DATA`, `APP_SECRETS`, ports and external Nemotron URL propagate through sudo or direct Docker;
- clone failures cause immediate nonzero exit, **no Compose launch and no deployment record**;
- failure of each of three Docker image builds causes immediate nonzero exit, **old `current.env` byte-for-byte unchanged**, no Compose launch and no deployment history event;
- saved prior Core SHA is refused before keys/data are created or containers started;
- a failed first-start update restores old record and data and leaves old Core **stopped**.

`core/tass/chat_only_test.go` uses the existing Engine/GitHub fixtures. A deliberately disobedient Engine returns an action; Core in chat-only mode must reject it while preserving the text reply and refusing all pulse actions and task queue submission.

`.github/workflows/one-app-launcher-regression.yml` checks `bash -n` and executes the isolated no-Docker Python tests. Existing `core-tests` workflow executes the Go unit/race tests.

**Evidence categories:** source inspection and structural assertions PASS; mocked execution NOT RUN until GitHub Actions jobs finish; owner WSL/Docker/Nemotron/chat/entity/persistence NOT RUN; autonomous pulse BLOCKED by design. No model fixture may be labeled real Nemotron.

## Owner-machine acceptance staging

The owner's checkout `~/tass-one-app/core` is **not** assumed clean. The local `tass.sh` has uncommitted changes, and the owner previously provided a `current.env` with three valid built image references. This session does **not** replace that file or override the old deployment pointer.

**First command (read-only):**

```bash
cd ~/tass-one-app/core && sed -n '/^compose() {/,/^}/p' tass.sh
```

Paste the output before any source change or start. We will compare the **actual local** Compose function to the tested patch; then use a separate worktree to preserve local edits. The subsequent owner-controlled explicit activation must produce a health response with `action_mode=CHAT_ONLY` before unlock/chat. The owner should not run `start`, `step` or other pulse controls on an unguarded old Core.

After the owner verifies **three new healthy `tass-app` containers**, original Nemotron unchanged, new Identity enrollment, one **real** model conversation, and records across restart, resume the unmerged Notebook Laboratory PR #54. Its `@identity-test` is presently a candidate lookup only, **not integrated into this Go app** and not authority for notebook execution. First lesson can be a supervised **text-only** prediction from real Nemotron; run the Jupyter cell only after a separately reviewed Identity-backed isolated runner is connected.
