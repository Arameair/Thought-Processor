#!/usr/bin/env bash
# TASS as one Docker application: Identity + Thought Engine (20-s work pulse) + core + one local model.
#
#   ./tass.sh check            READ-ONLY preflight: Docker, GPU, model file, ports, other models, inventory
#   ./tass.sh start            start the deployed version (first time: build this checkout's pinned version)
#   ./tass.sh status           READY / WORKING / BUSY / BLOCKED / UNKNOWN, Identity, pulse, versions
#   ./tass.sh logs             follow the application's logs
#   ./tass.sh check-update     current vs candidate version and what changed. Applies nothing.
#   ./tass.sh update [REF]     build the candidate, back up, switch, validate; if validation fails, put back
#                              the previous version AND the data. Asks first (--yes to skip).
#   ./tass.sh rollback         switch back to the previous version (--restore-data BACKUP to restore data too)
#   ./tass.sh backup           copy the application's data and keys into a backup (nothing else changes)
#   ./tass.sh stop             stop this application's containers (nothing else)
#
# A fresh install has NO entity: open http://127.0.0.1:8796 and create your own. Nothing personal is
# built in or downloaded: no one else's entity, identity, memory, document or policy. After every restart
# Identity is LOCKED until you unlock it with your passphrase.
#
# Settings live in $APP_HOME/app.env (written on first run). This script never starts, stops or changes
# containers that aren't part of this application, never merges, resets or switches your branches,
# never changes group membership, and never deletes data (it moves it aside).
set -euo pipefail
cd "$(dirname "$0")"
here="$(pwd)"

# Pinned versions of the other two Go services for THIS version of the core
# (an update changes these lines, never a branch head).
ENGINE_COMMIT_PIN="21b3d502775b3ed065ab445b88f51909500770b3"
IDENTITY_COMMIT_PIN="91e1521659a31db79044a44bd33bce48178554d9"

APP_HOME="${APP_HOME:-$HOME/.tass-app}"
APP_DATA="$APP_HOME/data"
APP_SECRETS="$APP_HOME/secrets"
DEPLOY="$APP_HOME/deploy"
APP_PROJECT="${APP_PROJECT:-tass-app}"
TRACK="${TASS_TRACK:-experiment/integration-20261104}"
ISSUE50_FIX="90c8892"   # core commit that rejects unlocked / mismatched proposals before any action
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
json_field() { python3 -c 'import json,sys; d=json.load(sys.stdin); v=d
for k in sys.argv[1].split("."): v=(v or {}).get(k) if isinstance(v,dict) else None
print("" if v is None else (" ".join(map(str,v)) if isinstance(v,list) else v))' "$1"; }

# ------------------------------------------------------------------------------- settings
settings() {
  mkdir -p "$APP_HOME" && chmod 700 "$APP_HOME"
  if [[ ! -f "$APP_HOME/app.env" ]]; then
    cat >"$APP_HOME/app.env" <<'EOF'
# Settings for this TASS application. Edit, then ./tass.sh check.
#
# MODEL_MODE=bundled   start the application's own llama.cpp model from your weights (MODEL_DIR/MODEL_FILE)
# MODEL_MODE=external  use a model that is already running (MODEL_NETWORK + MODEL_URL); nothing new starts
MODEL_MODE=bundled
MODEL_DIR=
MODEL_FILE=
MODEL_GPU=auto          # auto | on | off (off = CPU, slow for large models)
MODEL_CTX=8192
MODEL_IMAGE_GPU=ghcr.io/ggml-org/llama.cpp:server-cuda
MODEL_IMAGE_CPU=ghcr.io/ggml-org/llama.cpp:server
MODEL_NETWORK=          # external only, e.g. tass-workspace_default
MODEL_URL=              # external only, e.g. http://model:8081
CORE_PORT=8795
IDENTITY_PORT=8796
PULSE_EVERY=20s
IDENTITY_UNLOCK_HOURS=12
EOF
    chmod 600 "$APP_HOME/app.env"
  fi
  if ! grep -q '^GITHUB_REPO=' "$APP_HOME/app.env"; then
    cat >>"$APP_HOME/app.env" <<'EOF'

# GitHub: your entity's document and its lessons. Empty = everything stays on this computer.
# The token comes from "gh auth token" (or GITHUB_TOKEN) at start, kept in secrets/, never in an image.
GITHUB_REPO=            # e.g. Arameair/TASS
GITHUB_BRANCH=live      # the document syncs here (never main/alpha/stable); lessons are read from here
GITHUB_DOC=tass.md
GITHUB_LESSONS=lessons
EOF
  fi
  # shellcheck disable=SC1091
  set -a; source "$APP_HOME/app.env"; set +a
  CORE_PORT="${CORE_PORT:-8795}"; IDENTITY_PORT="${IDENTITY_PORT:-8796}"
}

gpu_usable() { [[ $docker_ok == 1 ]] && "${DOCKER[@]}" info --format '{{json .Runtimes}}' 2>/dev/null | grep -q nvidia && command -v nvidia-smi >/dev/null && nvidia-smi -L >/dev/null 2>&1; }
use_gpu() { case "${MODEL_GPU:-auto}" in on) return 0 ;; off) return 1 ;; *) gpu_usable ;; esac; }

model_files() { # compose files and model variables for this computer
  COMPOSE_FILES=(-f "$1/compose.yaml")
  PROFILES=()
  if [[ "${MODEL_MODE:-bundled}" == external ]]; then
    COMPOSE_FILES+=(-f "$1/compose.external-model.yaml")
    MODEL_ARGS="unused"; RUN_MODEL_URL="$MODEL_URL"
  else
    PROFILES=(--profile bundled-model)
    RUN_MODEL_URL="http://model:8081"
    if [[ -n "${FIXTURE_MODEL_IMAGE:-}" ]]; then   # tests only (no real model in the test sandbox)
      MODEL_IMAGE="$FIXTURE_MODEL_IMAGE"; MODEL_ARGS="8081"
      MODEL_HEALTH_CMD="python3 -c \"import urllib.request; urllib.request.urlopen('http://127.0.0.1:8081/health')\""
    elif use_gpu; then
      COMPOSE_FILES+=(-f "$1/compose.gpu.yaml")
      MODEL_IMAGE="$MODEL_IMAGE_GPU"; MODEL_ARGS="-m /models/$MODEL_FILE --host 0.0.0.0 --port 8081 -c ${MODEL_CTX:-8192} -ngl 99"
    else
      MODEL_IMAGE="$MODEL_IMAGE_CPU"; MODEL_ARGS="-m /models/$MODEL_FILE --host 0.0.0.0 --port 8081 -c ${MODEL_CTX:-8192}"
    fi
  fi
}

load_env() { [[ -f "$1" ]] || return 1; source "$1"; }   # shellcheck disable=SC1090

compose() { # compose for the version in $1 (an env file)
  local envf="$1"; shift
  load_env "$envf" || die "no deployed version recorded ($envf)"
  model_files "$(dirname "$COMPOSE_FILE")"
  local compose_env=(
    "APP_PROJECT=$APP_PROJECT" "APP_UID=$(id -u)" "APP_GID=$(id -g)" "APP_DATA=$APP_DATA" "APP_SECRETS=$APP_SECRETS"
    "IDENTITY_IMAGE=$IDENTITY_IMAGE" "ENGINE_IMAGE=$ENGINE_IMAGE" "CORE_IMAGE=$CORE_IMAGE"
    "MODEL_IMAGE=${MODEL_IMAGE:-}" "MODEL_ARGS=$MODEL_ARGS" "MODEL_DIR=${MODEL_DIR:-}" "MODEL_URL=$RUN_MODEL_URL"
    "MODEL_HEALTH_CMD=${MODEL_HEALTH_CMD:-}" "MODEL_NETWORK=${MODEL_NETWORK:-}" "FIXTURE_DELAY=${FIXTURE_DELAY:-}"
    "CORE_PORT=$CORE_PORT" "IDENTITY_PORT=$IDENTITY_PORT" "PULSE_EVERY=${PULSE_EVERY:-20s}"
    "IDENTITY_UNLOCK_HOURS=${IDENTITY_UNLOCK_HOURS:-12}"
    "GITHUB_REPO=${GITHUB_REPO:-}" "GITHUB_BRANCH=${GITHUB_BRANCH:-live}" "GITHUB_DOC=${GITHUB_DOC:-tass.md}"
    "GITHUB_LESSONS=${GITHUB_LESSONS:-lessons}" "TASS_GITHUB_API=${TASS_GITHUB_API:-}"
  )
  if [[ "${DOCKER[0]}" == sudo ]]; then
    # sudo normally resets its environment. Put the Compose environment after sudo so `env`
    # creates it for docker instead of relying on the caller's sudoers env_keep policy.
    "${DOCKER[@]:0:${#DOCKER[@]}-1}" env "${compose_env[@]}" "${DOCKER[-1]}" \
      compose -p "$APP_PROJECT" "${COMPOSE_FILES[@]}" "${PROFILES[@]}" "$@"
  else
    env "${compose_env[@]}" "${DOCKER[@]}" compose -p "$APP_PROJECT" \
      "${COMPOSE_FILES[@]}" "${PROFILES[@]}" "$@"
  fi
}

ours() { "${DOCKER[@]}" ps -q --filter "label=com.docker.compose.project=$APP_PROJECT" 2>/dev/null; }
other_models() { # running model servers that are NOT this application's (so we never start a second)
  "${DOCKER[@]}" ps --format '{{.Names}}|{{.Image}}|{{.Label "com.docker.compose.project"}}|{{.Label "com.docker.compose.service"}}' 2>/dev/null |
    awk -F'|' -v p="$APP_PROJECT" '$3 != p && ($2 ~ /llama|vllm|ollama|nemotron/ || $4 == "model") {print $1" ("$2")"}'
}
health() { curl -sf -m 4 "http://127.0.0.1:$CORE_PORT/api/health" 2>/dev/null; }
id_state() { curl -sf -m 4 "http://127.0.0.1:$IDENTITY_PORT/v1/state" 2>/dev/null; }

inventory() { # data files and their line counts (records are only ever appended)
  for f in "$APP_DATA/identity/entity.json" "$APP_DATA/identity/decisions.jsonl" "$APP_DATA/core/tass.md" \
           "$APP_DATA/core/memory.jsonl" "$APP_DATA/core/conversation.jsonl" "$APP_DATA/engine/cycles.jsonl" \
           "$APP_DATA/engine/jobs.jsonl" "$APP_DATA/engine/ticks.jsonl" "$APP_DATA/engine/model-requests.jsonl"; do
    printf '%s %s\n' "$f" "$( [[ -f "$f" ]] && lines "$f" || echo missing)"
  done
}

# ------------------------------------------------------------------------------- check (read-only)
check() {
  local blockers=0
  say "Preflight (read-only: nothing is built, started, stopped or written except a first app.env)"
  [[ $docker_ok == 1 ]] && ok "Docker $("${DOCKER[@]}" version --format '{{.Server.Version}}' 2>/dev/null) · compose $("${DOCKER[@]}" compose version --short 2>/dev/null)" \
    || { bad "Docker isn't reachable (in WSL: sudo service docker start)"; blockers=1; }
  ok "Computer: $(nproc 2>/dev/null || echo ?) CPUs · $(awk '/MemTotal/ {printf "%.0f GB", $2/1048576}' /proc/meminfo 2>/dev/null) memory · $(df -h "$HOME" 2>/dev/null | awk 'NR==2 {print $4" free"}')"
  if [[ "${MODEL_MODE:-bundled}" == external ]]; then
    [[ -n "${MODEL_NETWORK:-}" && -n "${MODEL_URL:-}" ]] || { bad "MODEL_MODE=external needs MODEL_NETWORK and MODEL_URL in $APP_HOME/app.env"; blockers=1; }
    if [[ $docker_ok == 1 && -n "${MODEL_NETWORK:-}" ]]; then
      "${DOCKER[@]}" network inspect "$MODEL_NETWORK" >/dev/null 2>&1 && ok "Using the model already running on Docker network $MODEL_NETWORK ($MODEL_URL); no second model starts" \
        || { bad "Docker network $MODEL_NETWORK doesn't exist (is that model running?)"; blockers=1; }
    fi
  elif [[ -n "${FIXTURE_MODEL_IMAGE:-}" ]]; then
    note "TEST FIXTURE model image $FIXTURE_MODEL_IMAGE (tests only: not a real model)"
  else
    if [[ -n "${MODEL_DIR:-}" && -n "${MODEL_FILE:-}" && -f "$MODEL_DIR/$MODEL_FILE" ]]; then
      ok "Model weights: $MODEL_DIR/$MODEL_FILE ($(du -h "$MODEL_DIR/$MODEL_FILE" | cut -f1)), mounted read-only, never copied into an image"
    else
      bad "No model file: set MODEL_DIR and MODEL_FILE in $APP_HOME/app.env (or MODEL_MODE=external to use one already running)"; blockers=1
    fi
    if use_gpu; then ok "GPU: $(nvidia-smi --query-gpu=name,memory.total --format=csv,noheader 2>/dev/null | head -1) (the model runs on it)"
    elif [[ "${MODEL_GPU:-auto}" == on ]]; then bad "MODEL_GPU=on but Docker can't use an NVIDIA GPU here (install the NVIDIA Container Toolkit)"; blockers=1
    else note "No usable GPU from Docker: the model runs on the CPU (slow). Set MODEL_GPU=off to silence this."; fi
  fi
  if [[ "${MODEL_MODE:-bundled}" != external && $docker_ok == 1 ]]; then
    local others; others="$(other_models)"
    if [[ -n "$others" ]]; then
      bad "Another model is already running: $others. One computer runs ONE model: set MODEL_MODE=external, MODEL_NETWORK and MODEL_URL in $APP_HOME/app.env to use it (nothing of it is stopped)."
      blockers=1
    else ok "No other model server running: the application starts exactly one"; fi
  fi
  for p in "$CORE_PORT" "$IDENTITY_PORT"; do
    if port_busy "$p" && [[ -z "$( [[ $docker_ok == 1 ]] && ours)" ]]; then bad "Port $p is used by another program (set CORE_PORT / IDENTITY_PORT in app.env)"; blockers=1
    else ok "Port 127.0.0.1:$p available (or already this application's)"; fi
  done
  if [[ -n "${GITHUB_REPO:-}" ]]; then
    case "${GITHUB_BRANCH:-live}" in main|alpha|stable) bad "GITHUB_BRANCH=${GITHUB_BRANCH}: the document may not sync straight onto it; use its own branch (live)"; blockers=1 ;; esac
    if [[ -s "$APP_SECRETS/github_token" || -n "${GITHUB_TOKEN:-}" ]] || { command -v gh >/dev/null && gh auth status >/dev/null 2>&1; }; then
      ok "GitHub: $GITHUB_REPO@${GITHUB_BRANCH:-live} (${GITHUB_DOC:-tass.md}, lessons in ${GITHUB_LESSONS:-lessons}/); token from $( [[ -s "$APP_SECRETS/github_token" ]] && echo 'secrets/github_token' || echo 'gh / GITHUB_TOKEN at start')"
    else bad "GITHUB_REPO is set but there's no token: gh auth login (or set GITHUB_TOKEN)"; blockers=1; fi
  else note "No GITHUB_REPO in app.env: the entity runs locally only (no lessons from GitHub)."; fi
  [[ "$ENGINE_COMMIT_PIN" =~ ^[0-9a-f]{7,40}$ && "$IDENTITY_COMMIT_PIN" =~ ^[0-9a-f]{7,40}$ ]] && ok "Pins: engine $ENGINE_COMMIT_PIN · identity $IDENTITY_COMMIT_PIN" || { bad "bad commit pins"; blockers=1; }
  if [[ -f "$DEPLOY/current.env" ]]; then
    ( load_env "$DEPLOY/current.env"; ok "Deployed: core $(short "$CORE_SHA") · engine $(short "$ENGINE_SHA") · identity $(short "$IDENTITY_SHA") · since $DEPLOYED_AT" )
    [[ -f "$DEPLOY/previous.env" ]] && ( load_env "$DEPLOY/previous.env"; ok "Previous (rollback target): core $(short "$CORE_SHA")" )
  else note "Nothing deployed yet: start builds core $(short "$(git -C "$REPO" rev-parse HEAD)") with the pins above."; fi
  if [[ -s "$APP_DATA/identity/entity.json" ]]; then ok "This installation's entity exists ($(json_field name <"$APP_DATA/identity/entity.json")); it is LOCKED after every start until you unlock it"
  else note "No entity yet: after start, open http://127.0.0.1:$IDENTITY_PORT and create your own."; fi
  say "  Data in $APP_DATA (line counts; records are only appended):"
  inventory | sed "s|$APP_DATA/||; s/^/         /"
  return $blockers
}

# ------------------------------------------------------------------------------- build
worktree_for() { # $1 = core commit → a build checkout outside your working tree
  local sha="$1" dir="$DEPLOY/src/core-${1:0:12}"
  if [[ ! -d "$dir" ]]; then
    git -C "$REPO" worktree add --detach "$dir" "$sha" >/dev/null 2>&1 || die "can't check out $sha for building"
  fi
  [[ "$(git -C "$dir" rev-parse HEAD)" == "$sha" ]] || die "build checkout for $sha is not at $sha"
  printf '%s' "$dir"
}

src_for() { # $1 = name, $2 = pin, $3 = local source or "", $4 = HTTPS clone URL
  local name="$1" sha="$2" local_src="$3" url="$4" dir="$DEPLOY/src/$1-${2:0:12}"
  if [[ ! -d "$dir/.git" ]]; then
    [[ ! -e "$dir" ]] || die "$name checkout path exists but is not a Git clone: $dir"
    rm -rf -- "$dir.new"   # own previous incomplete clone under APP_HOME only
    if ! git clone -q --no-checkout "${local_src:-$url}" "$dir.new" 2>/dev/null; then
      rm -rf -- "$dir.new"   # a private repo over plain HTTPS: fall back to the signed-in GitHub CLI
      gh repo clone "${url#https://github.com/}" "$dir.new" -- -q --no-checkout || die "$name clone failed; deployment unchanged"
    fi
    git -C "$dir.new" checkout -q "$sha" || die "$name commit $sha not found"
    mv -- "$dir.new" "$dir" || die "$name checkout move failed"
  fi
  local found expected
  found="$(git -C "$dir" rev-parse --verify HEAD)" || die "$name HEAD not available"
  expected="$(git -C "$dir" rev-parse --verify "$sha^{commit}")" || die "$name pin not available"
  [[ "$found" == "$expected" ]] || die "$name checkout is not at $sha"
  printf '%s' "$dir"
}

pin_of() { git -C "$REPO" show "$1:core/tass.sh" 2>/dev/null | sed -n "s/^$2=\"\([0-9a-f]\{7,40\}\)\"$/\1/p"; }

build_version() { # $1 = core commit; writes $2 (env file) describing the built version
  local csha="$1" out="$2" esha isha src esrc isrc tmp
  esha="$(pin_of "$csha" ENGINE_COMMIT_PIN)"; isha="$(pin_of "$csha" IDENTITY_COMMIT_PIN)"
  [[ -n "$esha" && -n "$isha" ]] || die "core $csha has no engine/identity pins in core/tass.sh"
  src="$(worktree_for "$csha")" || die "cannot prepare pinned Core source"
  esrc="$(src_for engine "$esha" "${ENGINE_SOURCE:-}" "https://github.com/Arameair/Thought-engine.git")" || die "cannot prepare pinned Engine source"
  isrc="$(src_for identity "$isha" "${IDENTITY_SOURCE:-}" "https://github.com/Arameair/identity.git")" || die "cannot prepare pinned Identity source"
  esha="$(git -C "$esrc" rev-parse --verify HEAD)" || die "cannot determine Engine revision"
  isha="$(git -C "$isrc" rev-parse --verify HEAD)" || die "cannot determine Identity revision"
  local bargs=(); [[ -n "${GO_IMAGE:-}" ]] && bargs+=(--build-arg "GO_IMAGE=$GO_IMAGE"); [[ -n "${RUN_IMAGE:-}" ]] && bargs+=(--build-arg "RUN_IMAGE=$RUN_IMAGE")
  say "Building core $(short "$csha") · engine $(short "$esha") · identity $(short "$isha")…"
  "${DOCKER[@]}" build -q "${bargs[@]}" --build-arg "VERSION=$csha" -f "$src/core/docker/core.Dockerfile" -t "tass-core:$(short "$csha")" "$src/core" >/dev/null || die "core image build failed; deployment unchanged"
  "${DOCKER[@]}" build -q "${bargs[@]}" --build-arg "VERSION=$esha" -f "$src/core/docker/engine.Dockerfile" -t "tass-engine:$(short "$esha")" "$esrc" >/dev/null || die "engine image build failed; deployment unchanged"
  "${DOCKER[@]}" build -q "${bargs[@]}" --build-arg "VERSION=$isha" -f "$src/core/docker/identity.Dockerfile" -t "tass-identity:$(short "$isha")" "$isrc" >/dev/null || die "identity image build failed; deployment unchanged"
  tmp="$out.new.$$"
  cat >"$tmp" <<EOF || die "could not write candidate deployment record"
CORE_SHA=$csha
ENGINE_SHA=$esha
IDENTITY_SHA=$isha
CORE_IMAGE=tass-core:$(short "$csha")
ENGINE_IMAGE=tass-engine:$(short "$esha")
IDENTITY_IMAGE=tass-identity:$(short "$isha")
COMPOSE_FILE=$src/core/compose.yaml
DEPLOYED_AT=$(date -u +%Y-%m-%dT%H:%M:%SZ)
EOF
  mv "$tmp" "$out" || die "can't record built version in $out"
}

# ------------------------------------------------------------------------------- data and keys
newkey() { head -c 32 /dev/urandom | od -An -tx1 | tr -d ' \n'; }
prepare() { # folders and THIS installation's own keys (generated here, never shipped, never in an image)
  mkdir -p "$DEPLOY/backups" "$DEPLOY/src" "$APP_SECRETS" "$APP_DATA/identity" "$APP_DATA/core" "$APP_DATA/engine"
  chmod 700 "$APP_HOME" "$DEPLOY" "$DEPLOY/backups" "$APP_SECRETS" "$APP_DATA" "$APP_DATA/identity" "$APP_DATA/core" "$APP_DATA/engine"
  (umask 177
   [[ -s "$APP_SECRETS/engine.key" ]] || newkey >"$APP_SECRETS/engine.key"
   [[ -s "$APP_SECRETS/core.key" ]] || newkey >"$APP_SECRETS/core.key"
   [[ -s "$APP_DATA/engine/api-key" ]] || newkey >"$APP_DATA/engine/api-key"
   printf 'engine %s\ncore %s\n' "$(cat "$APP_SECRETS/engine.key")" "$(cat "$APP_SECRETS/core.key")" >"$APP_SECRETS/service-keys.new")
  mv "$APP_SECRETS/service-keys.new" "$APP_SECRETS/service-keys"
  # GitHub token for the core (only when a repository is set). Mounted read-only, never in an image.
  if [[ -n "${GITHUB_REPO:-}" && ! -s "$APP_SECRETS/github_token" ]]; then
    (umask 177
     if [[ -n "${GITHUB_TOKEN:-}" ]]; then printf '%s' "$GITHUB_TOKEN" >"$APP_SECRETS/github_token"
     else gh auth token >"$APP_SECRETS/github_token" 2>/dev/null || true; fi)
  fi
  [[ -f "$APP_SECRETS/github_token" ]] || (umask 177; : >"$APP_SECRETS/github_token")   # exists, so Docker mounts a file
  chmod 600 "$APP_SECRETS/github_token"
}

backup() { # → prints the backup file (data and keys; 0600 in a 0700 folder)
  local f="$DEPLOY/backups/$(date -u +%Y%m%dT%H%M%SZ)-${1:-manual}.tgz"
  (umask 077; tar -czf "$f" -C "$APP_HOME" data secrets 2>/dev/null) || die "backup failed: $f"
  printf '%s' "$f"
}

restore_data() { # $1 = backup; current data is MOVED aside (never deleted) first
  local b="$1" aside="$DEPLOY/failed/$(date -u +%Y%m%dT%H%M%SZ)"
  [[ -f "$b" ]] || die "no backup $b"
  mkdir -p "$aside" && chmod 700 "$DEPLOY/failed" "$aside"
  [[ -d "$APP_DATA" ]] && mv "$APP_DATA" "$aside/data"
  [[ -d "$APP_SECRETS" ]] && mv "$APP_SECRETS" "$aside/secrets"
  tar -xzf "$b" -C "$APP_HOME"
  say "Restored data and keys from $b (what was there is kept in $aside)."
}

# ------------------------------------------------------------------------------- validate
rank() { case "$1" in READY|BUSY|WORKING) echo 2 ;; BLOCKED) echo 1 ;; *) echo 0 ;; esac; }

validate() { # $1 = env file that should now run; $2 = inventory before; $3 = state before; $4 = entity id before
  local envf="$1" before="$2" was="$3" ent="$4" h state ver i s all
  load_env "$envf"
  say "Validating core $(short "$CORE_SHA") · engine $(short "$ENGINE_SHA") · identity $(short "$IDENTITY_SHA")…"
  local svcs=(identity engine core); [[ "${MODEL_MODE:-bundled}" == external ]] || svcs+=(model)
  for i in $(seq 90); do
    all=1
    for s in "${svcs[@]}"; do
      [[ "$("${DOCKER[@]}" inspect -f '{{.State.Health.Status}}' "$APP_PROJECT-$s-1" 2>/dev/null || echo none)" == healthy ]] || all=0
    done
    [[ $all == 1 ]] && break
    sleep 2
  done
  for s in "${svcs[@]}"; do
    local hs; hs="$("${DOCKER[@]}" inspect -f '{{.State.Health.Status}}' "$APP_PROJECT-$s-1" 2>/dev/null || echo none)"
    [[ "$hs" == healthy ]] || { say "  FAIL: $s is $hs"; return 1; }
  done
  if [[ "${MODEL_MODE:-bundled}" != external ]]; then
    local n; n="$("${DOCKER[@]}" ps -q --filter "label=com.docker.compose.project=$APP_PROJECT" --filter "label=com.docker.compose.service=model" | wc -l | tr -d ' ')"
    [[ "$n" == 1 ]] || { say "  FAIL: $n model containers (expected exactly 1)"; return 1; }
  fi
  h="$(health)" || { say "  FAIL: the core's /api/health doesn't answer"; return 1; }
  state="$(json_field state <<<"$h")"; ver="$(json_field version <<<"$h")"
  # A healthy but unguarded old Core is NOT acceptance for this one-app trial.
  case "$(json_field action_mode <<<"$h")" in
    CHAT_ONLY|AUTONOMOUS_GUARDED) ;;   # both enforce the Issue #50 checks; older builds report neither
    *) say "  FAIL: Core doesn't report a guarded action mode (Issue #50 safety build required)"; return 1 ;;
  esac
  [[ "$ver" == "$CORE_SHA" ]] || { say "  FAIL: running core reports version $ver, expected $CORE_SHA"; return 1; }
  local ids now_ent ids_state; ids="$(id_state)" || { say "  FAIL: Identity doesn't answer"; return 1; }
  now_ent="$(json_field entity.entity_id <<<"$ids")"; ids_state="$(json_field state <<<"$ids")"
  if [[ -n "$ent" && "$now_ent" != "$ent" ]]; then say "  FAIL: the entity changed ($ent → ${now_ent:-none})"; return 1; fi
  if [[ "$ids_state" == UNLOCKED ]]; then
    [[ "$(rank "$state")" -ge "$(rank "$was")" ]] || { say "  FAIL: state $state is worse than before ($was): $(json_field reasons <<<"$h")"; return 1; }
  else
    [[ "$state" == BLOCKED ]] || { say "  FAIL: Identity is $ids_state but the core says $state (it must be BLOCKED)"; return 1; }
  fi
  while read -r f n; do   # records may grow, never shrink or disappear
    local cur; cur="$( [[ -f "$f" ]] && lines "$f" || echo missing)"
    if [[ "$n" != missing && ( "$cur" == missing || "$cur" -lt "$n" ) ]]; then say "  FAIL: $f had $n lines, now $cur"; return 1; fi
  done <<<"$before"
  say "  PASS: all healthy · version $(short "$ver") · Identity $ids_state${now_ent:+ ($now_ent)} · state $state · no records lost"
  case "$ids_state" in
    NO_ENTITY) say "  No entity yet: create your own at http://127.0.0.1:$IDENTITY_PORT" ;;
    UNLOCKED) ;;
    *) say "  Identity is $ids_state: unlock at http://127.0.0.1:$IDENTITY_PORT to let it work." ;;
  esac
}

current_state() { local h; h="$(health || true)"; [[ -n "$h" ]] && json_field state <<<"$h" || echo UNKNOWN; }
current_entity() { local s; s="$(id_state || true)"; [[ -n "$s" ]] && json_field entity.entity_id <<<"$s" || true; }

# ------------------------------------------------------------------------------- commands
cmd_start() {
  check || { say; say "Not starting: fix the [stop] items (nothing was changed)."; exit 1; }
  # This experimental safety build must never start a previously built,
  # unguarded Core image. Replacing it requires an explicit reviewed update,
  # not an implicit upgrade or overwrite of the owner's current.env.
  local checkout_sha recorded_sha
  checkout_sha="$(git -C "$REPO" rev-parse --verify HEAD)" || die "cannot inspect checkout revision"
  if [[ -f "$DEPLOY/current.env" ]]; then
    recorded_sha="$(load_env "$DEPLOY/current.env"; printf '%s' "${CORE_SHA:-}")" || die "cannot read deployed revision"
    # Refuse only builds WITHOUT the Issue #50 fix (90c8892); an already-updated safety build starts normally.
    if [[ "$recorded_sha" != "$checkout_sha" ]] && ! git -C "$REPO" merge-base --is-ancestor "$ISSUE50_FIX" "$recorded_sha" 2>/dev/null; then
      die "saved Core $(short "$recorded_sha") predates the Issue #50 safety fix; current.env unchanged. Update explicitly: ./tass.sh update"
    fi
  fi
  say
  prepare
  if [[ ! -f "$DEPLOY/current.env" ]] || ! (load_env "$DEPLOY/current.env"; "${DOCKER[@]}" image inspect "$CORE_IMAGE" "$ENGINE_IMAGE" "$IDENTITY_IMAGE" >/dev/null 2>&1); then
    local sha; sha="$( [[ -f "$DEPLOY/current.env" ]] && (load_env "$DEPLOY/current.env"; echo "$CORE_SHA") || git -C "$REPO" rev-parse HEAD)"
    build_version "$sha" "$DEPLOY/current.env.new" || die "build aborted; current.env preserved"
    [[ -s "$DEPLOY/current.env.new" ]] || die "build produced no candidate deployment record"
    mv -- "$DEPLOY/current.env.new" "$DEPLOY/current.env" || die "could not activate built deployment record"
    echo "$(date -u +%FT%TZ) start: deployed core $sha" >>"$DEPLOY/history.log"
  fi
  local before ent; before="$(inventory)"; ent="$( [[ -s "$APP_DATA/identity/entity.json" ]] && json_field entity_id <"$APP_DATA/identity/entity.json" || true)"
  compose "$DEPLOY/current.env" up -d
  validate "$DEPLOY/current.env" "$before" "" "$ent" || { say "Started, but validation failed: ./tass.sh logs"; exit 1; }
  say "Running. Identity: http://127.0.0.1:$IDENTITY_PORT  ·  Talk and work pulse: http://127.0.0.1:$CORE_PORT"
}

cmd_status() {
  if [[ -f "$DEPLOY/current.env" ]]; then load_env "$DEPLOY/current.env"; say "Deployed: core $(short "$CORE_SHA") · engine $(short "$ENGINE_SHA") · identity $(short "$IDENTITY_SHA") · since $DEPLOYED_AT"; fi
  [[ $docker_ok == 1 ]] && "${DOCKER[@]}" ps -a --filter "label=com.docker.compose.project=$APP_PROJECT" --format '  {{.Names}}: {{.Status}}'
  local ids; ids="$(id_state || true)"
  [[ -n "$ids" ]] && say "Identity: $(json_field state <<<"$ids") $(json_field entity.name <<<"$ids") $(json_field reason <<<"$ids")" || say "Identity: UNKNOWN (not answering)"
  local h; h="$(health || true)"
  if [[ -z "$h" ]]; then say "State: UNKNOWN (the core isn't answering on 127.0.0.1:$CORE_PORT)"
  else
    say "State: $(json_field state <<<"$h")  $(json_field reasons <<<"$h")"
    say "  engine reachable $(json_field engine_reachable <<<"$h") · model ok $(json_field model_ok <<<"$h")"
    say "  action gate: $(json_field action_mode <<<"$h") (CHAT_ONLY = Autonomous off · AUTONOMOUS_GUARDED = on, Issue #50 checks enforced)"
  fi
  say "  last pulse tick: $(tail -n 1 "$APP_DATA/engine/ticks.jsonl" 2>/dev/null | cut -c1-200 || echo none)"
}

cmd_check_update() {
  local ref="${1:-origin/$TRACK}"
  [[ "$ref" == origin/* ]] && git -C "$REPO" fetch -q origin "${ref#origin/}:refs/remotes/$ref" 2>/dev/null || true
  local cand; cand="$(git -C "$REPO" rev-parse --verify -q "$ref^{commit}")" || die "can't find $ref"
  [[ -f "$DEPLOY/current.env" ]] || die "nothing deployed yet: ./tass.sh start"
  load_env "$DEPLOY/current.env"
  say "Current:   core $CORE_SHA · engine $(short "$ENGINE_SHA") · identity $(short "$IDENTITY_SHA")"
  say "Candidate: core $cand · engine $(pin_of "$cand" ENGINE_COMMIT_PIN) · identity $(pin_of "$cand" IDENTITY_COMMIT_PIN) ($ref)"
  if [[ "$cand" == "$CORE_SHA" ]]; then say "Up to date."; return 0; fi
  say; say "Changes in core/ (newest first):"
  git -C "$REPO" log --oneline "$CORE_SHA..$cand" -- core | sed 's/^/  /' | head -30
  git -C "$REPO" diff --stat "$CORE_SHA" "$cand" -- core | tail -1 | sed 's/^/  /'
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
  if [[ $yes == 0 ]]; then read -r -p "Apply this update? Data and keys are backed up first. [y/N] " yn; [[ "$yn" == [yY]* ]] || { say "Nothing changed."; return 0; }; fi
  prepare
  build_version "$cand" "$DEPLOY/candidate.env"
  local was before ent bk had_running
  had_running="$(ours)"  # preserve whether this project was actually running before update
  was="$(current_state)"; before="$(inventory)"; ent="$(current_entity)"
  compose "$DEPLOY/current.env" stop >/dev/null 2>&1 || true
  bk="$(backup "before-${cand:0:12}")"; say "Backup: $bk"
  [[ -f "$DEPLOY/previous.env" ]] && cp "$DEPLOY/previous.env" "$DEPLOY/previous.env.keep" || rm -f "$DEPLOY/previous.env.keep"
  cp "$DEPLOY/current.env" "$DEPLOY/previous.env"
  cp "$DEPLOY/candidate.env" "$DEPLOY/current.env"
  compose "$DEPLOY/current.env" up -d --force-recreate >/dev/null 2>&1 || true
  if validate "$DEPLOY/current.env" "$before" "$was" "$ent"; then
    echo "$(date -u +%FT%TZ) update: $(load_env "$DEPLOY/previous.env"; echo "$CORE_SHA") → $cand OK (backup $bk)" >>"$DEPLOY/history.log"
    say "UPDATED to core $(short "$cand"). Previous version kept for ./tass.sh rollback."
    return 0
  fi
  say "Update FAILED validation: putting back the previous version and your data."
  compose "$DEPLOY/current.env" logs --tail 20 2>/dev/null | sed 's/^/    /' || true
  compose "$DEPLOY/current.env" down >/dev/null 2>&1 || true
  cp "$DEPLOY/previous.env" "$DEPLOY/current.env"
  # the rollback target stays what it was before this failed attempt
  if [[ -f "$DEPLOY/previous.env.keep" ]]; then cp "$DEPLOY/previous.env.keep" "$DEPLOY/previous.env"; else rm -f "$DEPLOY/previous.env"; fi
  restore_data "$bk"
  if [[ -z "$had_running" ]]; then
    # Fresh first-start recovery: old images were not running before the failed
    # candidate. Preserve the old env/data, but do NOT boot an unguarded Core.
    echo "$(date -u +%FT%TZ) update: → $cand FAILED; previous record/data preserved; application remains stopped" >>"$DEPLOY/history.log"
    say "ROLLED BACK deployment record and data; the new application remains stopped (nothing was running before this attempt)."
    return 1
  fi
  compose "$DEPLOY/current.env" up -d --force-recreate >/dev/null 2>&1 || true
  if validate "$DEPLOY/current.env" "$before" "$was" "$ent"; then
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
  local before ent; before="$(inventory)"; ent="$(current_entity)"
  compose "$DEPLOY/current.env" down >/dev/null 2>&1 || true
  cp "$DEPLOY/current.env" "$DEPLOY/rolled-back.env"
  cp "$DEPLOY/previous.env" "$DEPLOY/current.env"
  cp "$DEPLOY/rolled-back.env" "$DEPLOY/previous.env"
  if [[ -n "$restore" ]]; then restore_data "$restore"; before="$(inventory)"; fi
  compose "$DEPLOY/current.env" up -d --force-recreate >/dev/null 2>&1 || true
  validate "$DEPLOY/current.env" "$before" "" "$ent" || die "rollback started but didn't validate: ./tass.sh logs"
  echo "$(date -u +%FT%TZ) rollback: now core $(load_env "$DEPLOY/current.env"; echo "$CORE_SHA")" >>"$DEPLOY/history.log"
  say "Rolled back. (./tass.sh rollback again returns to the version you just left.)"
}

if [[ "${TASS_LIB_ONLY:-0}" == 1 ]]; then return 0 2>/dev/null || exit 0; fi

settings
case "${1:-help}" in
  check) check && say "Ready: ./tass.sh start" || say "Fix the [stop] items, then ./tass.sh check again." ;;
  start) cmd_start ;;
  status) cmd_status ;;
  logs) compose "$DEPLOY/current.env" logs -f --tail 80 ;;
  check-update) shift; cmd_check_update "$@" ;;
  update) shift; cmd_update "$@" ;;
  rollback) shift; cmd_rollback "$@" ;;
  backup) mkdir -p "$DEPLOY/backups" && chmod 700 "$DEPLOY/backups"; say "Backup: $(backup manual)" ;;
  stop) compose "$DEPLOY/current.env" stop && say "Stopped this application's containers (nothing else)." ;;
  *) sed -n '2,22p' "$0" ;;
esac
