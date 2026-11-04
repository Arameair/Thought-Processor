#!/usr/bin/env bash
# TASS's Go core in Docker: one system you can check, start, inspect, update and recover.
#
#   ./tass.sh check            READ-ONLY preflight and inventory. Changes nothing.
#   ./tass.sh start            start the deployed version (first time: build this checkout's version)
#   ./tass.sh status           READY / WORKING / BUSY / BLOCKED / UNKNOWN, versions, pulse, sync
#   ./tass.sh logs             follow both services' logs
#   ./tass.sh check-update     show the current and candidate versions and what changed. Applies nothing.
#   ./tass.sh update [REF]     build the candidate, back up your data, switch, validate; if validation
#                              fails, put back the previous version AND your data. Asks first (--yes to skip).
#   ./tass.sh rollback         switch back to the previous version (data untouched; --restore-data BACKUP to restore it)
#   ./tass.sh backup           copy ~/.tass and ~/.thought-engine/tass into a backup (nothing else changes)
#   ./tass.sh stop             stop the two Go containers (nothing else)
#
# Uses the Nemotron model already running in the workspace stack. Never starts or stops the model,
# the Python workspace, the liaison or the Atrium; never merges, resets or switches your branches.
set -euo pipefail
cd "$(dirname "$0")"
here="$(pwd)"

# Pinned Thought Engine for THIS version of the core (an update changes this line, never a branch head).
ENGINE_COMMIT_PIN="4924de6f1b8504abcab4fed98c0e191057f4e277"

TRACK="${TASS_TRACK:-experiment/go-core-docker-20261104}"   # branch check-update looks at
DEPLOY="${TASS_DEPLOY:-$HOME/.tass-deploy}"                  # versions, backups, build sources, token copy
TASS_DATA="${TASS_DATA:-$HOME/.tass}"
ENGINE_DATA="${ENGINE_DATA:-$HOME/.thought-engine/tass}"
CORE_PORT="${CORE_PORT:-8795}"
MODEL_NETWORK="${MODEL_NETWORK:-tass-workspace_default}"
MODEL_URL="${MODEL_URL:-http://model:8081}"
MODEL_HOST_URL="${MODEL_HOST_URL:-http://127.0.0.1:18083}"
REPO="$(git -C "$here" rev-parse --show-toplevel)"

say()  { printf '%s\n' "$*"; }
ok()   { printf '  [ok]   %s\n' "$*"; }
note() { printf '  [note] %s\n' "$*"; }
bad()  { printf '  [stop] %s\n' "$*"; }
die()  { printf 'error: %s\n' "$*" >&2; exit 1; }

DOCKER=(docker)
docker_ok=1
if ! command -v docker >/dev/null 2>&1; then docker_ok=0
elif ! docker info >/dev/null 2>&1; then
  if sudo -n docker info >/dev/null 2>&1; then DOCKER=(sudo -n docker); else docker_ok=0; fi
fi
port_busy() { (exec 3<>"/dev/tcp/127.0.0.1/$1") 2>/dev/null; }
short() { printf '%s' "${1:0:12}"; }
lines() { [[ -f "$1" ]] && wc -l <"$1" | tr -d ' ' || echo 0; }

load_env() { # current.env / previous.env → variables
  [[ -f "$1" ]] || return 1
  # shellcheck disable=SC1090
  source "$1"
}

compose() { # compose for the version in $1 (an env file)
  local envf="$1"; shift
  load_env "$envf" || die "no deployed version recorded ($envf)"
  CORE_IMAGE="$CORE_IMAGE" ENGINE_IMAGE="$ENGINE_IMAGE" TASS_UID="$(id -u)" TASS_GID="$(id -g)" \
  TASS_DATA="$TASS_DATA" ENGINE_DATA="$ENGINE_DATA" TOKEN_FILE="$DEPLOY/secrets/github_token" \
  MODEL_NETWORK="$MODEL_NETWORK" MODEL_URL="$MODEL_URL" CORE_PORT="$CORE_PORT" \
  TASS_GITHUB_MESSAGES="${TASS_GITHUB_MESSAGES:-off}" TASS_IDENTITY="${TASS_IDENTITY:-}" \
  TASS_SYNC="${TASS_SYNC:-}" TASS_GITHUB_API="${TASS_GITHUB_API:-}" \
    "${DOCKER[@]}" compose -p tass-core -f "$COMPOSE_FILE" "$@"
}

python_liaison_running() { [[ $docker_ok == 1 ]] && "${DOCKER[@]}" ps --format '{{.Names}}' 2>/dev/null | grep -q '^tass-workspace-liaison-'; }
host_core_running() { [[ -f "$TASS_DATA/logs/tass-core.pid" ]] && kill -0 "$(cat "$TASS_DATA/logs/tass-core.pid")" 2>/dev/null; }
health() { curl -sf -m 4 "http://127.0.0.1:$CORE_PORT/api/health" 2>/dev/null; }
json_field() { python3 -c 'import json,sys; d=json.load(sys.stdin); v=d;
for k in sys.argv[1].split("."): v=(v or {}).get(k) if isinstance(v,dict) else None
print("" if v is None else v)' "$1"; }

inventory() { # data files and their line counts (records are only ever appended)
  for f in "$TASS_DATA/tass.md" "$TASS_DATA/memory.jsonl" "$TASS_DATA/conversation.jsonl" "$TASS_DATA/sync.jsonl" \
           "$TASS_DATA/grant/identity-grant.json" "$ENGINE_DATA/cycles.jsonl" "$ENGINE_DATA/pulse.jsonl" \
           "$ENGINE_DATA/model-requests.jsonl"; do
    printf '%s %s\n' "$f" "$( [[ -f "$f" ]] && lines "$f" || echo missing)"
  done
}

# ------------------------------------------------------------------------------- check (read-only)
check() {
  local blockers=0
  say "Preflight and inventory (read-only: nothing is built, started, stopped or written)"
  [[ $docker_ok == 1 ]] && ok "Docker: $("${DOCKER[@]}" version --format '{{.Server.Version}}' 2>/dev/null) · compose $("${DOCKER[@]}" compose version --short 2>/dev/null)" \
    || { bad "Docker isn't reachable (in WSL: sudo service docker start)"; blockers=1; }
  command -v git >/dev/null && ok "This checkout: $(git -C "$REPO" rev-parse --abbrev-ref HEAD) @ $(short "$(git -C "$REPO" rev-parse HEAD)")$( [[ -n "$(git -C "$REPO" status --porcelain -- core)" ]] && echo ' (core/ has local changes)')"
  command -v gh >/dev/null && gh auth status >/dev/null 2>&1 && ok "GitHub CLI signed in" || note "GitHub CLI not signed in (needed to fetch the Thought Engine and for the token): gh auth login"
  if [[ $docker_ok == 1 ]]; then
    local model; model="$("${DOCKER[@]}" ps --filter "network=$MODEL_NETWORK" --filter "name=model" --format '{{.Names}} ({{.Status}})' 2>/dev/null | head -1)"
    [[ -n "$model" ]] && ok "Existing model container on $MODEL_NETWORK: $model" || { bad "No model container on Docker network $MODEL_NETWORK. Start the workspace's model (cd ../workspace && ./run.sh). This script never starts a second one."; blockers=1; }
  fi
  curl -sf -m 3 -o /dev/null "$MODEL_HOST_URL/health" && ok "Nemotron health answers at $MODEL_HOST_URL" || note "Nemotron health not answering at $MODEL_HOST_URL (still loading?)"
  if [[ -f "$DEPLOY/current.env" ]]; then
    load_env "$DEPLOY/current.env"
    ok "Deployed version: core $(short "$CORE_SHA") · engine $(short "$ENGINE_SHA") · since $DEPLOYED_AT"
    [[ -f "$DEPLOY/previous.env" ]] && { ( load_env "$DEPLOY/previous.env"; ok "Previous (rollback target): core $(short "$CORE_SHA") · engine $(short "$ENGINE_SHA")" ); }
    if [[ $docker_ok == 1 ]]; then
      for img in "$CORE_IMAGE" "$ENGINE_IMAGE"; do
        "${DOCKER[@]}" image inspect "$img" >/dev/null 2>&1 && ok "Image $img: $("${DOCKER[@]}" image inspect -f '{{.Id}}' "$img" | cut -c1-19)" || note "Image $img not on this computer (start will rebuild it)"
      done
      "${DOCKER[@]}" ps -a --filter "label=com.docker.compose.project=tass-core" --format '  [info] container {{.Names}}: {{.Status}}' 2>/dev/null
    fi
  else
    note "Nothing deployed yet: start builds this checkout (core $(short "$(git -C "$REPO" rev-parse HEAD)"), engine $(short "$ENGINE_COMMIT_PIN"))."
  fi
  [[ "$ENGINE_COMMIT_PIN" =~ ^[0-9a-f]{40}$ ]] && ok "Thought Engine pin for this checkout: $ENGINE_COMMIT_PIN" || { bad "ENGINE_COMMIT_PIN isn't a full commit SHA"; blockers=1; }
  if port_busy "$CORE_PORT" && ! [[ $docker_ok == 1 && -n "$("${DOCKER[@]}" ps -q --filter label=com.docker.compose.project=tass-core 2>/dev/null)" ]]; then
    if host_core_running; then bad "Port $CORE_PORT is used by the host-process core (./start.sh). Start asks before stopping it."
    else bad "Port $CORE_PORT is used by another program (CORE_PORT=... to use another)"; blockers=1; fi
  else ok "Port $CORE_PORT available for the core's page (or already ours)"; fi
  python_liaison_running && note "The Python liaison is running: it also answers 'tass' issues, so the Go core keeps GitHub messages PAUSED. Nothing is stopped." \
    || ok "No Python liaison running"
  if [[ -s "$TASS_DATA/grant/identity-grant.json" ]]; then ok "Identity policy cached (PROTOTYPE file), confirmed $(json_field fetched_at <"$TASS_DATA/grant/identity-grant.json")"
  else note "No Identity policy cached: TASS stays BLOCKED until policies/tass.json is on Arameair/identity main (PR #31, owner review)."; fi
  [[ -s "$DEPLOY/secrets/github_token" ]] && ok "GitHub token present for the core (mounted read-only; never in an image)" || note "No token copy yet: start makes one from gh (0600, $DEPLOY/secrets/)"
  say "  Data (line counts; these files are only appended to):"
  inventory | sed 's/^/         /'
  if [[ $docker_ok == 1 ]] && "${DOCKER[@]}" ps -q --filter label=com.docker.compose.project=tass-core 2>/dev/null | grep -q .; then
    local h; h="$(health || true)"
    [[ -n "$h" ]] && ok "Running core says: $(json_field state <<<"$h") (version $(short "$(json_field version <<<"$h")"))" || note "Core container present but not answering"
  fi
  return $blockers
}

# ------------------------------------------------------------------------------- build
worktree_for() { # $1 = core commit → a read-only build checkout outside your working tree
  local sha="$1" dir="$DEPLOY/src/core-${1:0:12}"
  if [[ ! -d "$dir" ]]; then
    git -C "$REPO" worktree add --detach "$dir" "$sha" >/dev/null 2>&1 || die "can't check out $sha for building"
  fi
  [[ "$(git -C "$dir" rev-parse HEAD)" == "$sha" ]] || die "build checkout for $sha is not at $sha"
  printf '%s' "$dir"
}

engine_src_for() { # $1 = engine commit
  local sha="$1" dir="$DEPLOY/src/engine-${1:0:12}"
  if [[ ! -d "$dir/.git" ]]; then
    rm -rf "$dir.new"   # only our own half-finished clone
    if [[ -n "${ENGINE_SOURCE:-}" ]]; then git clone -q --no-checkout "$ENGINE_SOURCE" "$dir.new"
    else gh repo clone Arameair/Thought-engine "$dir.new" -- -q --no-checkout; fi
    git -C "$dir.new" checkout -q "$sha" || die "engine commit $sha not found"
    mv "$dir.new" "$dir"
  fi
  [[ "$(git -C "$dir" rev-parse HEAD)" == "$sha" ]] || die "engine checkout is not at $sha"
  printf '%s' "$dir"
}

pin_of() { git -C "$REPO" show "$1:core/tass.sh" 2>/dev/null | sed -n 's/^ENGINE_COMMIT_PIN="\([0-9a-f]\{40\}\)"$/\1/p'; }

build_version() { # $1 = core commit; writes $2 (env file) describing the built version
  local csha="$1" out="$2" esha src esrc
  esha="$(pin_of "$csha")"; [[ -n "$esha" ]] || die "core $csha has no ENGINE_COMMIT_PIN in core/tass.sh"
  src="$(worktree_for "$csha")"; esrc="$(engine_src_for "$esha")"
  local bargs=(); [[ -n "${GO_IMAGE:-}" ]] && bargs+=(--build-arg "GO_IMAGE=$GO_IMAGE"); [[ -n "${RUN_IMAGE:-}" ]] && bargs+=(--build-arg "RUN_IMAGE=$RUN_IMAGE")
  say "Building tass-core:$(short "$csha") and tass-engine:$(short "$esha")…"
  "${DOCKER[@]}" build -q "${bargs[@]}" --build-arg "VERSION=$csha" -f "$src/core/docker/core.Dockerfile" -t "tass-core:$(short "$csha")" "$src/core" >/dev/null
  "${DOCKER[@]}" build -q "${bargs[@]}" --build-arg "VERSION=$esha" -f "$src/core/docker/engine.Dockerfile" -t "tass-engine:$(short "$esha")" "$esrc" >/dev/null
  cat >"$out" <<EOF
CORE_SHA=$csha
ENGINE_SHA=$esha
CORE_IMAGE=tass-core:$(short "$csha")
ENGINE_IMAGE=tass-engine:$(short "$esha")
CORE_IMAGE_ID=$("${DOCKER[@]}" image inspect -f '{{.Id}}' "tass-core:$(short "$csha")")
ENGINE_IMAGE_ID=$("${DOCKER[@]}" image inspect -f '{{.Id}}' "tass-engine:$(short "$esha")")
COMPOSE_FILE=$src/core/compose.yaml
DEPLOYED_AT=$(date -u +%Y-%m-%dT%H:%M:%SZ)
EOF
}

# ------------------------------------------------------------------------------- data
prepare() {
  mkdir -p "$DEPLOY/secrets" "$DEPLOY/backups" "$DEPLOY/src" "$TASS_DATA/grant" "$ENGINE_DATA"
  chmod 700 "$DEPLOY" "$DEPLOY/secrets" "$DEPLOY/backups" "$TASS_DATA" "$TASS_DATA/grant" "$ENGINE_DATA"
  if [[ ! -s "$ENGINE_DATA/api-key" ]]; then (umask 177; head -c 32 /dev/urandom | od -An -tx1 | tr -d ' \n' >"$ENGINE_DATA/api-key"); fi
  chmod 600 "$ENGINE_DATA/api-key"
  if [[ ! -s "$DEPLOY/secrets/github_token" ]]; then
    if [[ -s "$here/secrets/github_token" ]]; then cp "$here/secrets/github_token" "$DEPLOY/secrets/github_token"
    else (umask 177; gh auth token >"$DEPLOY/secrets/github_token") || die "no GitHub token: run gh auth login"; fi
  fi
  chmod 600 "$DEPLOY/secrets/github_token"
}

backup() { # → prints the backup file
  local f="$DEPLOY/backups/$(date -u +%Y%m%dT%H%M%SZ)-${1:-manual}.tgz"
  mkdir -p "$DEPLOY/backups" && chmod 700 "$DEPLOY/backups"
  (umask 077; tar -czf "$f" -C / "${TASS_DATA#/}" "${ENGINE_DATA#/}" 2>/dev/null) || die "backup failed: $f"
  printf '%s' "$f"
}

restore_data() { # $1 = backup; current data is MOVED aside (never deleted) first
  local b="$1" aside="$DEPLOY/failed/$(date -u +%Y%m%dT%H%M%SZ)"
  [[ -f "$b" ]] || die "no backup $b"
  mkdir -p "$aside" && chmod 700 "$DEPLOY/failed" "$aside"
  [[ -d "$TASS_DATA" ]] && mv "$TASS_DATA" "$aside/tass"
  [[ -d "$ENGINE_DATA" ]] && mv "$ENGINE_DATA" "$aside/engine"
  tar -xzf "$b" -C /
  say "Restored data from $b (what was there is kept in $aside)."
}

# ------------------------------------------------------------------------------- validate
rank() { case "$1" in READY|BUSY|WORKING) echo 2 ;; BLOCKED) echo 1 ;; *) echo 0 ;; esac; }

validate() { # $1 = env file of the version that should now run; $2 = inventory before; $3 = state before
  local envf="$1" before="$2" was="$3" h state ver i
  load_env "$envf"
  say "Validating core $(short "$CORE_SHA") / engine $(short "$ENGINE_SHA")…"
  for i in $(seq 60); do
    local e c
    e="$("${DOCKER[@]}" inspect -f '{{.State.Health.Status}}' tass-core-engine-1 2>/dev/null || echo none)"
    c="$("${DOCKER[@]}" inspect -f '{{.State.Health.Status}}' tass-core-core-1 2>/dev/null || echo none)"
    [[ "$e" == healthy && "$c" == healthy ]] && break
    sleep 2
  done
  [[ "$e" == healthy && "$c" == healthy ]] || { say "  FAIL: containers not healthy (engine $e, core $c)"; return 1; }
  h="$(health)" || { say "  FAIL: the core's /api/health doesn't answer"; return 1; }
  state="$(json_field state <<<"$h")"; ver="$(json_field version <<<"$h")"
  [[ "$ver" == "$CORE_SHA" ]] || { say "  FAIL: running core reports version $ver, expected $CORE_SHA"; return 1; }
  [[ "$(rank "$state")" -ge "$(rank "$was")" ]] || { say "  FAIL: state $state is worse than before ($was): $(json_field reasons <<<"$h")"; return 1; }
  while read -r f n; do   # records may grow, never shrink or disappear
    local now; now="$( [[ -f "$f" ]] && lines "$f" || echo missing)"
    if [[ "$n" != missing && ( "$now" == missing || "$now" -lt "$n" ) ]]; then say "  FAIL: $f had $n lines, now $now"; return 1; fi
  done <<<"$before"
  say "  PASS: both healthy · version $(short "$ver") · state $state (before: ${was:-none}) · no records lost"
}

current_state() { local h; h="$(health || true)"; [[ -n "$h" ]] && json_field state <<<"$h" || echo UNKNOWN; }

# ------------------------------------------------------------------------------- commands
cmd_start() {
  check || { say; say "Not starting: fix the [stop] items (nothing was changed)."; exit 1; }
  say
  if host_core_running; then
    [[ -t 0 ]] || die "the host-process core (./start.sh) is running; stop it first with ./start.sh stop"
    read -r -p "The host-process core (./start.sh) is running on port $CORE_PORT. Stop it and run the Docker version? [y/N] " yn
    [[ "$yn" == [yY]* ]] || { say "Nothing changed."; exit 0; }
    ./start.sh stop
  fi
  export TASS_GITHUB_MESSAGES=on
  if python_liaison_running; then
    TASS_GITHUB_MESSAGES=off
    say "The Python liaison is running, so GitHub messages stay PAUSED in the Go core (cutover needs your OK)."
  fi
  prepare
  if [[ ! -f "$DEPLOY/current.env" ]] || ! (load_env "$DEPLOY/current.env"; "${DOCKER[@]}" image inspect "$CORE_IMAGE" "$ENGINE_IMAGE" >/dev/null 2>&1); then
    local sha; sha="$( [[ -f "$DEPLOY/current.env" ]] && (load_env "$DEPLOY/current.env"; echo "$CORE_SHA") || git -C "$REPO" rev-parse HEAD)"
    build_version "$sha" "$DEPLOY/current.env.new" && mv "$DEPLOY/current.env.new" "$DEPLOY/current.env"
    echo "$(date -u +%FT%TZ) start: deployed core $sha" >>"$DEPLOY/history.log"
  fi
  local before; before="$(inventory)"
  compose "$DEPLOY/current.env" up -d
  validate "$DEPLOY/current.env" "$before" "" || { say "Started, but validation failed: ./tass.sh logs"; exit 1; }
  say "TASS is running: http://127.0.0.1:$CORE_PORT"
}

cmd_status() {
  if [[ -f "$DEPLOY/current.env" ]]; then load_env "$DEPLOY/current.env"; say "Deployed: core $(short "$CORE_SHA") · engine $(short "$ENGINE_SHA") · since $DEPLOYED_AT"; fi
  [[ $docker_ok == 1 ]] && "${DOCKER[@]}" ps -a --filter label=com.docker.compose.project=tass-core --format '  {{.Names}}: {{.Status}}'
  local h; h="$(health || true)"
  if [[ -z "$h" ]]; then say "TASS: UNKNOWN (the core isn't answering on 127.0.0.1:$CORE_PORT)"
  else
    say "TASS: $(json_field state <<<"$h")  $(json_field reasons <<<"$h")"
    say "  engine reachable $(json_field engine_reachable <<<"$h") · Nemotron ok $(json_field model_ok <<<"$h") · Identity policy valid $(json_field policy_valid_now <<<"$h")"
    say "  GitHub synced $(json_field github_online <<<"$h") (last $(json_field last_sync <<<"$h")) · messages $( [[ "$(json_field github_messages <<<"$h")" == True ]] && echo on || echo PAUSED) · replies waiting $(json_field replies_waiting <<<"$h")"
  fi
  say "  last engine pulse: $(tail -n 1 "$ENGINE_DATA/pulse.jsonl" 2>/dev/null || echo none)"
  say "  last thought cycle: $(tail -n 1 "$ENGINE_DATA/cycles.jsonl" 2>/dev/null | cut -c1-160 || echo none)"
}

cmd_check_update() {
  local ref="${1:-origin/$TRACK}"
  [[ "$ref" == origin/* ]] && git -C "$REPO" fetch -q origin "${ref#origin/}:refs/remotes/$ref" 2>/dev/null || true
  local cand; cand="$(git -C "$REPO" rev-parse --verify -q "$ref^{commit}")" || die "can't find $ref"
  [[ -f "$DEPLOY/current.env" ]] || die "nothing deployed yet: ./tass.sh start"
  load_env "$DEPLOY/current.env"
  say "Current:   core $CORE_SHA · engine $ENGINE_SHA"
  say "Candidate: core $cand · engine $(pin_of "$cand") ($ref)"
  if [[ "$cand" == "$CORE_SHA" ]]; then say "Up to date."; return 0; fi
  say; say "Changes in core/ (newest first):"
  git -C "$REPO" log --oneline "$CORE_SHA..$cand" -- core | sed 's/^/  /' | head -30
  git -C "$REPO" diff --stat "$CORE_SHA" "$cand" -- core | tail -1 | sed 's/^/  /'
  [[ "$(pin_of "$cand")" != "$ENGINE_SHA" ]] && say "  Thought Engine pin changes: $(short "$ENGINE_SHA") → $(short "$(pin_of "$cand")")"
  say; say "Apply with: ./tass.sh update $cand"
}

cmd_update() {
  local ref="" yes=0
  for a in "$@"; do case "$a" in --yes) yes=1 ;; *) ref="$a" ;; esac; done
  ref="${ref:-origin/$TRACK}"
  [[ -f "$DEPLOY/current.env" ]] || die "nothing deployed yet: ./tass.sh start"
  cmd_check_update "$ref"
  local cand; cand="$(git -C "$REPO" rev-parse --verify -q "$ref^{commit}")"
  load_env "$DEPLOY/current.env"; [[ "$cand" == "$CORE_SHA" ]] && return 0
  if [[ $yes == 0 ]]; then read -r -p "Apply this update? Your data is backed up first. [y/N] " yn; [[ "$yn" == [yY]* ]] || { say "Nothing changed."; return 0; }; fi
  check >/dev/null || die "preflight has [stop] items: run ./tass.sh check"
  prepare
  build_version "$cand" "$DEPLOY/candidate.env"
  local was before bk; was="$(current_state)"; before="$(inventory)"
  compose "$DEPLOY/current.env" stop >/dev/null 2>&1 || true
  bk="$(backup "before-${cand:0:12}")"; say "Backup: $bk"
  cp "$DEPLOY/current.env" "$DEPLOY/previous.env"
  cp "$DEPLOY/candidate.env" "$DEPLOY/current.env"
  compose "$DEPLOY/current.env" up -d --force-recreate >/dev/null 2>&1 || true
  if validate "$DEPLOY/current.env" "$before" "$was"; then
    echo "$(date -u +%FT%TZ) update: $(load_env "$DEPLOY/previous.env"; echo "$CORE_SHA") → $cand OK (backup $bk)" >>"$DEPLOY/history.log"
    say "UPDATED to core $(short "$cand"). Previous version kept for ./tass.sh rollback."
    return 0
  fi
  say "Update FAILED validation: putting back the previous version and your data."
  compose "$DEPLOY/current.env" logs --tail 20 2>/dev/null | sed 's/^/    /' || true
  compose "$DEPLOY/current.env" down >/dev/null 2>&1 || true
  cp "$DEPLOY/previous.env" "$DEPLOY/current.env"
  restore_data "$bk"
  compose "$DEPLOY/current.env" up -d --force-recreate >/dev/null 2>&1 || true
  if validate "$DEPLOY/current.env" "$before" "$was"; then
    echo "$(date -u +%FT%TZ) update: → $cand FAILED, restored $(load_env "$DEPLOY/current.env"; echo "$CORE_SHA") and data ($bk)" >>"$DEPLOY/history.log"
    say "ROLLED BACK: running the previous version again, with your data as it was before the update."
    return 1
  fi
  echo "$(date -u +%FT%TZ) update: → $cand FAILED and the restore did not validate" >>"$DEPLOY/history.log"
  die "the previous version didn't validate either: ./tass.sh logs (backup: $bk)"
}

cmd_rollback() {
  local restore=""
  [[ "${1:-}" == --restore-data ]] && restore="${2:?backup file}"
  [[ -f "$DEPLOY/previous.env" ]] || die "no previous version recorded"
  local was before; was="$(current_state)"; before="$(inventory)"
  compose "$DEPLOY/current.env" down >/dev/null 2>&1 || true
  cp "$DEPLOY/current.env" "$DEPLOY/rolled-back.env"
  cp "$DEPLOY/previous.env" "$DEPLOY/current.env"
  cp "$DEPLOY/rolled-back.env" "$DEPLOY/previous.env"
  if [[ -n "$restore" ]]; then restore_data "$restore"; before="$(inventory)"; fi
  compose "$DEPLOY/current.env" up -d --force-recreate >/dev/null 2>&1 || true
  validate "$DEPLOY/current.env" "$before" "" || die "rollback started but didn't validate: ./tass.sh logs"
  echo "$(date -u +%FT%TZ) rollback: now core $(load_env "$DEPLOY/current.env"; echo "$CORE_SHA")" >>"$DEPLOY/history.log"
  say "Rolled back. (./tass.sh rollback again returns to the version you just left.)"
}

case "${1:-help}" in
  check) check && say "Ready: ./tass.sh start" || say "Fix the [stop] items, then ./tass.sh check again." ;;
  start) cmd_start ;;
  status) cmd_status ;;
  logs) compose "$DEPLOY/current.env" logs -f --tail 80 ;;
  check-update) shift; cmd_check_update "$@" ;;
  update) shift; cmd_update "$@" ;;
  rollback) shift; cmd_rollback "$@" ;;
  backup) say "Backup: $(backup manual)" ;;
  stop) compose "$DEPLOY/current.env" stop && say "Stopped the Go core containers (nothing else)." ;;
  *) sed -n '2,17p' "$0" ;;
esac
