#!/usr/bin/env bash
# Launcher-only regressions. Uses fake docker/sudo commands; never touches Docker or owner data.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
TASS="$ROOT/core/tass.sh"
SANDBOX="$(mktemp -d)"
trap 'rm -rf "$SANDBOX"' EXIT
FAKE_BIN="$SANDBOX/bin"
mkdir -p "$FAKE_BIN"

cat >"$FAKE_BIN/sudo" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
[[ "${1:-}" == -n ]] && shift
# Model sudo's default env_reset. Harness controls are explicitly retained, launcher variables are not.
exec env -i PATH="$PATH" TRACE="${TRACE:-}" TEST_COMPOSE_RC="${TEST_COMPOSE_RC:-0}" SUDO_TEST=1 "$@"
EOF

cat >"$FAKE_BIN/docker" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
cmd="${1:-}"; shift || true
case "$cmd" in
  info) [[ "${SUDO_TEST:-}" == 1 ]] ;;
  version) echo 27.0.0 ;;
  network|image) exit 0 ;;
  ps) exit 0 ;;
  compose)
    if [[ "${1:-}" == version ]]; then echo 2.29.0; exit 0; fi
    required=(CORE_IMAGE ENGINE_IMAGE IDENTITY_IMAGE APP_UID APP_GID APP_DATA APP_SECRETS MODEL_ARGS MODEL_URL MODEL_NETWORK)
    for name in "${required[@]}"; do
      [[ -n "$(printenv "$name")" ]] || { echo "missing Compose environment: $name" >&2; exit 90; }
    done
    printf '%s\n' "CORE_IMAGE=$CORE_IMAGE" "ENGINE_IMAGE=$ENGINE_IMAGE" "APP_UID=$APP_UID" \
      "APP_DATA=$APP_DATA" "APP_SECRETS=$APP_SECRETS" "MODEL_ARGS=$MODEL_ARGS" >>"$TRACE"
    if [[ " $* " == *" up "* && "${TEST_COMPOSE_RC:-0}" != 0 ]]; then exit "$TEST_COMPOSE_RC"; fi
    ;;
  build) exit 0 ;;
  *) exit 0 ;;
esac
EOF
chmod +x "$FAKE_BIN/sudo" "$FAKE_BIN/docker"

APP_HOME="$SANDBOX/app"
mkdir -p "$APP_HOME/deploy"
cat >"$APP_HOME/app.env" <<'EOF'
MODEL_MODE=external
MODEL_NETWORK=tass-workspace_default
MODEL_URL=http://model:8081
CORE_PORT=49876
IDENTITY_PORT=49877
EOF
cat >"$APP_HOME/deploy/current.env" <<EOF
CORE_SHA=1111111111111111111111111111111111111111
ENGINE_SHA=2222222222222222222222222222222222222222
IDENTITY_SHA=3333333333333333333333333333333333333333
CORE_IMAGE=tass-core:111111111111
ENGINE_IMAGE=tass-engine:222222222222
IDENTITY_IMAGE=tass-identity:333333333333
COMPOSE_FILE=$ROOT/core/compose.yaml
DEPLOYED_AT=2026-11-04T00:00:00Z
EOF

TRACE="$SANDBOX/compose.env"
export APP_HOME PATH="$FAKE_BIN:$PATH" TRACE
"$TASS" stop >/dev/null
for expected in CORE_IMAGE=tass-core:111111111111 ENGINE_IMAGE=tass-engine:222222222222 \
  APP_DATA="$APP_HOME/data" APP_SECRETS="$APP_HOME/secrets" MODEL_ARGS=unused; do
  grep -Fx "$expected" "$TRACE" >/dev/null || { echo "FAIL: Compose lost $expected through sudo"; exit 1; }
done
echo "PASS: sudo-required Docker Compose receives the complete launcher environment"

set +e
TEST_COMPOSE_RC=37 "$TASS" start >"$SANDBOX/start.out" 2>&1
rc=$?
set -e
[[ $rc == 37 ]] || { echo "FAIL: Compose startup exit 37 became $rc"; cat "$SANDBOX/start.out"; exit 1; }
echo "PASS: Docker Compose startup failure is returned to the caller (exit 37)"

HEAD_SHA="$(git -C "$ROOT" rev-parse HEAD)"
OUT="$SANDBOX/checkout-failure.env"
set +e
TASS_LIB_ONLY=1 OUT="$OUT" HEAD_SHA="$HEAD_SHA" bash -c '
  source "$0"
  pin_of() { printf a847385; }
  worktree_for() { return 23; }
  build_version "$HEAD_SHA" "$OUT"
' "$TASS" >/dev/null 2>&1
rc=$?
set -e
[[ $rc != 0 && ! -e "$OUT" ]] || { echo "FAIL: failed checkout produced a deployment record"; exit 1; }
echo "PASS: checkout failure cannot produce a deployment record"

BUILD_DOCKER="$SANDBOX/build-docker"
cat >"$BUILD_DOCKER" <<'EOF'
#!/usr/bin/env bash
count=$(cat "${BUILD_COUNT}" 2>/dev/null || echo 0)
count=$((count + 1)); printf '%s' "$count" >"$BUILD_COUNT"
[[ $count -ne 2 ]]
EOF
chmod +x "$BUILD_DOCKER"
OUT="$SANDBOX/build-failure.env"
BUILD_COUNT="$SANDBOX/build-count"
set +e
TASS_LIB_ONLY=1 OUT="$OUT" HEAD_SHA="$HEAD_SHA" BUILD_DOCKER="$BUILD_DOCKER" BUILD_COUNT="$BUILD_COUNT" bash -c '
  source "$0"
  pin_of() { printf a847385; }
  worktree_for() { printf %s "$REPO"; }
  src_for() { printf %s "$REPO"; }
  DOCKER=("$BUILD_DOCKER")
  build_version "$HEAD_SHA" "$OUT"
' "$TASS" >/dev/null 2>&1
rc=$?
set -e
[[ $rc != 0 && ! -e "$OUT" ]] || { echo "FAIL: failed image build produced a deployment record"; exit 1; }
echo "PASS: image-build failure cannot produce a deployment record"
