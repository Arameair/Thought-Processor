#!/usr/bin/env bash
# Issue #79 harness: runs the REAL tass.sh update/rollback/restore/recover code with the image build
# replaced (no Docker, no network). Fake docker/tar live in $PATH; see test_updater_recovery.py.
#   bash updater_harness.sh <tass.sh subcommand> [args...]
# Fault injection (environment):
#   TEST_KILL_AT=<phase>   kill -9 this process right after the operation records that phase
#                          (STOPPED, BACKED_UP, SWITCHED, VALIDATING, RESTORED_DATA): a power cut
set -euo pipefail
TASS_LIB_ONLY=1 source "$(dirname "$0")/../tass.sh"

cmd_check_update() { :; }          # needs git history/network; not what is under test
build_version() {                  # $1 = core commit; writes $2 exactly as the real one does
  [[ "${TEST_BUILD_FAIL:-0}" == 1 ]] && die "core image build failed; deployment unchanged"
  local tmp="$2.new.$$"
  cat >"$tmp" <<EOF
CORE_SHA=$1
ENGINE_SHA=d69d3917d1601d806d4b2ba2254693e1adb92b6b
IDENTITY_SHA=7da7f7c5bbecc42d3a7e9316a2037243eda10b88
CORE_IMAGE=tass-core:$(short "$1")
ENGINE_IMAGE=tass-engine:d69d3917d160
IDENTITY_IMAGE=tass-identity:7da7f7c5bbec
COMPOSE_FILE=$REPO/core/compose.yaml
DEPLOYED_AT=$(date -u +%Y-%m-%dT%H:%M:%SZ)
EOF
  mv "$tmp" "$2"
  echo "tass-core:$(short "$1") $1" >>"$TEST_STATE/images"
}
sleep() { :; }                     # validate's health wait loop: the fake answers at once

eval "real_$(declare -f op_set)"
op_set() {
  real_op_set "$@"
  if [[ "$1" == OP_PHASE && "${TEST_KILL_AT:-}" == "$2" ]]; then
    echo "HARNESS: power cut at $2" >&2; kill -9 $$
  fi
}

tass_main "$@"
