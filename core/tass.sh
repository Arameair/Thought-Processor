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
#   ./tass.sh restore-previous the version from before the last update AND its verified pre-update data
#                              (the current data is first saved in a new verified backup). Asks first.
#   ./tass.sh recover          finish an INTERRUPTED update/rollback/restore: put back the version and data
#                              from before it (start/update/rollback refuse until this is done)
#   ./tass.sh history          versions, verified backups and the update history. Read-only.
#   ./tass.sh backup           stop, copy data and keys into a VERIFIED backup, start again
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
# engine: experiment/stage2-nudge-run-20261108 (durable nudge runs, Stop, verified development; Stage 2 + ST2-I01 repair)
ENGINE_COMMIT_PIN="8bed0fc522905b3c2e110a0b426f4e258812a9b9"
# identity: experiment/nudge-process-stop-20261108 (single-use bound decisions for nudge process / stop; Stage 2)
IDENTITY_COMMIT_PIN="cfc4079b690e2f7dc57afb9f14c6832ae38fdf54"

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
  # Stage 2 restore fence: restore_data must be able to flush single files and folders.
  local probe; probe="$(mktemp 2>/dev/null)" && if sync "$probe" 2>/dev/null; then ok "sync can flush single files (needed by a safe restore)"
  else bad "this system's sync can't flush a single file (coreutils 8.24+ needed for a safe restore)"; blockers=1; fi; rm -f -- "${probe:-}"
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

sha256_of() { sha256sum "$1" | cut -d' ' -f1; }

# backup → prints the backup file (data and keys; 0600 in a 0700 folder). Issue #79: a backup only
# counts once it is VERIFIED: the archive is extracted to a scratch folder and every file is compared
# with the live data (the services are stopped while this runs), then its SHA-256 is recorded next
# to it. Nothing is ever pruned: older backups, including the last known-good one, are kept.
backup() {
  local name f tmp scratch
  name="$(date -u +%Y%m%dT%H%M%SZ)-${1:-manual}.tgz"; f="$DEPLOY/backups/$name"; tmp="$DEPLOY/backups/.$name.partial"
  rm -f -- "$tmp"
  (umask 077; tar -czf "$tmp" -C "$APP_HOME" data secrets 2>/dev/null) || { rm -f -- "$tmp"; say "backup failed: could not write $tmp" >&2; return 1; }
  scratch="$(mktemp -d "$DEPLOY/backups/.verify.XXXXXX")" || { rm -f -- "$tmp"; return 1; }
  if ! tar -xzf "$tmp" -C "$scratch" 2>/dev/null || ! diff -r -q "$scratch/data" "$APP_DATA" >/dev/null 2>&1 ||
     ! diff -r -q "$scratch/secrets" "$APP_SECRETS" >/dev/null 2>&1; then
    rm -rf -- "$scratch" "$tmp"; say "backup FAILED verification: the archive does not match the data" >&2; return 1
  fi
  rm -rf -- "$scratch"
  (umask 077; sha256_of "$tmp" >"$f.sha256.new") && mv -- "$tmp" "$f" && mv -- "$f.sha256.new" "$f.sha256" || { say "backup failed: could not finish $f" >&2; return 1; }
  printf '%s' "$f"
}

backup_ok() { # $1 = backup: present and unchanged since it was verified
  [[ -f "$1" && -f "$1.sha256" && "$(sha256_of "$1")" == "$(cat "$1.sha256")" ]]
}

# restore_data $1 = backup. Stage 2 (design rev 4 §4.6): the backup is unpacked into a STAGING folder
# first, a restore fence is written into the staged Core data (so every nudge draft in the restored
# journal is PRE_RESTORE and can't be processed), the staged files are synced, and only then is the
# live data moved aside (never deleted) and the staged data renamed into place. A crash before the
# rename leaves the old data live; a crash after it leaves restored data that already carries its
# fence. Same signature and callers as before.
restore_fence_write() { # $1 = staged data folder
  local core="$1/core" j fence tmp bytes hash
  mkdir -p "$core" && chmod 700 "$core"
  j="$core/nudges.jsonl"; fence="$core/restore-fence.json"; tmp="$core/.restore-fence.json.tmp"
  if [[ -f "$j" ]]; then bytes="$(stat -c %s "$j")"; hash="$(sha256_of "$j")"
  else bytes=0; hash="$(printf '' | sha256sum | cut -d' ' -f1)"; fi
  (umask 077; printf '{"schema":"tass.restore-fence/2","restored_at":"%s","backup":"%s","journal_bytes":%s,"journal_sha256":"%s"}\n' \
    "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$(basename "$2")" "$bytes" "$hash" >"$tmp") || return 1
  sync "$tmp" && mv -f -- "$tmp" "$fence" && sync "$fence" "$core"
}

restore_data() { # $1 = backup; current data is MOVED aside (never deleted) first
  local b="$1" aside="$DEPLOY/failed/$(date -u +%Y%m%dT%H%M%SZ)-$$" stage="$DEPLOY/restore-stage.$(date -u +%Y%m%dT%H%M%SZ).$$" need free
  # R1: verify the backup, and that there is room for an unpacked copy beside the live data
  [[ -f "$b" ]] || die "no backup $b"
  if [[ -f "$b.sha256" ]]; then backup_ok "$b" || die "backup $b does not match its recorded SHA-256; nothing was changed"
  else note "backup $b predates verified backups (no .sha256): restoring it unverified"; fi
  need="$(tar -tvzf "$b" 2>/dev/null | awk '{s+=$3} END {printf "%d", s/1024+1024}')" || die "could not read $b; nothing was changed"
  free="$(df -Pk "$DEPLOY" | awk 'NR==2 {print $4}')"
  [[ -n "$need" && -n "$free" && "$free" -gt "$need" ]] || die "not enough free space to restore $b (need about ${need} KB, have ${free:-?} KB); nothing was changed"
  # R2: unpack into the staging folder (the live data is untouched)
  rm -rf -- "$DEPLOY"/restore-stage.*   # left by an interrupted restore (operations hold the op lock, so none is live)
  mkdir -p "$stage" && chmod 700 "$stage"
  if ! tar -xzf "$b" -C "$stage" || [[ ! -d "$stage/data" ]]; then
    rm -rf -- "$stage"; die "could not unpack $b; nothing was changed"
  fi
  mkdir -p "$stage/secrets" && chmod 700 "$stage/secrets"
  # R3: fence the restored nudge journal, then make the staged copy durable
  restore_fence_write "$stage/data" "$b" || { rm -rf -- "$stage"; die "could not write the restore fence; nothing was changed"; }
  sync -f "$stage" 2>/dev/null || sync
  # R4: move the live data aside (never deleted)
  mkdir -p "$aside" && chmod 700 "$DEPLOY/failed" "$aside"
  [[ -d "$APP_DATA" ]] && mv "$APP_DATA" "$aside/data"
  [[ -d "$APP_SECRETS" ]] && mv "$APP_SECRETS" "$aside/secrets"
  # R5: publish the staged data with renames (same file system), then sync the parent folder
  if ! mv -- "$stage/data" "$APP_DATA" || ! mv -- "$stage/secrets" "$APP_SECRETS"; then
    rm -rf -- "$APP_DATA" "$APP_SECRETS"; mv "$aside/data" "$APP_DATA" 2>/dev/null; mv "$aside/secrets" "$APP_SECRETS" 2>/dev/null
    rm -rf -- "$stage"; die "could not put the restored data in place; the data that was there is back in place"
  fi
  sync "$APP_HOME" 2>/dev/null || sync
  rm -rf -- "$stage"
  say "Restored data and keys from $b (what was there is kept in $aside; nudge drafts from before the restore are marked and can't be processed)."
}

# ---- one operation at a time, and a record of any operation in progress (Issue #79)
OPLOCK_HELD=0
op_lock() {
  mkdir -p "$DEPLOY" && chmod 700 "$DEPLOY"
  exec 9>"$DEPLOY/.op.lock"
  flock -n 9 || die "another ./tass.sh operation is running on this installation; nothing was changed"
  OPLOCK_HELD=1
}
OPF() { printf '%s' "$DEPLOY/operation.env"; }
op_begin() { # $1 = kind; records what is needed to recover if this process dies
  local tmp; tmp="$(OPF).new"
  cp "$DEPLOY/current.env" "$DEPLOY/operation-from.env"
  { echo "OP_KIND=$1"; echo "OP_STARTED=$(date -u +%FT%TZ)"; echo "OP_PHASE=STARTING"; echo "OP_BACKUP="; echo "OP_RUNNING=${2:-}"; } >"$tmp"
  mv -- "$tmp" "$(OPF)"
}
op_set() { # $1 = key, $2 = value
  local tmp; tmp="$(OPF).new"
  grep -v "^$1=" "$(OPF)" >"$tmp"; echo "$1=$2" >>"$tmp"; mv -- "$tmp" "$(OPF)"
}
op_end() { rm -f -- "$(OPF)" "$DEPLOY/operation-from.env"; }
op_guard() { # refuse to act on top of an interrupted operation
  if [[ -f "$(OPF)" ]]; then
    ( load_env "$(OPF)"; say "An earlier $OP_KIND (started $OP_STARTED) was INTERRUPTED at phase $OP_PHASE." )
    die "run ./tass.sh recover to put back the version and data from before it (nothing was changed now)"
  fi
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
  build_version "$cand" "$DEPLOY/candidate.env"   # a build failure stops here: nothing was stopped or changed
  local was before ent bk had_running
  had_running="$(ours)"  # preserve whether this project was actually running before update
  was="$(current_state)"; before="$(inventory)"; ent="$(current_entity)"
  op_begin update "$had_running"
  compose "$DEPLOY/current.env" stop >/dev/null 2>&1 || true
  op_set OP_PHASE STOPPED
  if ! bk="$(backup "before-${cand:0:12}")"; then
    # Issue #79: a failed backup must not leave the application down or half-changed.
    say "Backup FAILED or could not be verified: the update was NOT applied."
    [[ -n "$had_running" ]] && compose "$DEPLOY/current.env" up -d >/dev/null 2>&1 || true
    op_end
    echo "$(date -u +%FT%TZ) update: → $cand REFUSED (backup failed); previous version $( [[ -n "$had_running" ]] && echo restarted || echo left stopped as it was)" >>"$DEPLOY/history.log"
    die "nothing changed; previous version $( [[ -n "$had_running" ]] && echo 'running again' || echo 'left as it was')"
  fi
  say "Backup (verified, SHA-256 recorded): $bk"
  op_set OP_BACKUP "$bk"; op_set OP_PHASE BACKED_UP
  [[ -f "$DEPLOY/previous.env" ]] && cp "$DEPLOY/previous.env" "$DEPLOY/previous.env.keep" || rm -f "$DEPLOY/previous.env.keep"
  cp "$DEPLOY/current.env" "$DEPLOY/previous.env"
  cp "$DEPLOY/candidate.env" "$DEPLOY/current.env"
  op_set OP_PHASE SWITCHED
  compose "$DEPLOY/current.env" up -d --force-recreate >/dev/null 2>&1 || true
  op_set OP_PHASE VALIDATING
  if validate "$DEPLOY/current.env" "$before" "$was" "$ent"; then
    local m; m="$DEPLOY/manifests/$(date -u +%Y%m%dT%H%M%SZ)-core-${cand:0:12}"; mkdir -p "$DEPLOY/manifests" && chmod 700 "$DEPLOY/manifests"
    cp "$DEPLOY/current.env" "$m.env"; cp "$DEPLOY/previous.env" "$m.previous.env"
    { echo "UPDATE_BACKUP=$bk"; echo "UPDATE_BACKUP_SHA256=$(cat "$bk.sha256")"; echo "UPDATE_PREVIOUS=$m.previous.env"; echo "UPDATE_CURRENT=$m.env"; } >"$DEPLOY/last-update.env.new"
    mv -- "$DEPLOY/last-update.env.new" "$DEPLOY/last-update.env"
    echo "$(date -u +%FT%TZ) update: $(load_env "$DEPLOY/previous.env"; echo "$CORE_SHA") → $cand OK (backup $bk)" >>"$DEPLOY/history.log"
    op_end
    say "UPDATED to core $(short "$cand"). Previous version kept for ./tass.sh rollback; ./tass.sh restore-previous also puts back the data from before this update."
    return 0
  fi
  say "Update FAILED validation: putting back the previous version and your data."
  compose "$DEPLOY/current.env" logs --tail 20 2>/dev/null | sed 's/^/    /' || true
  compose "$DEPLOY/current.env" down >/dev/null 2>&1 || true
  cp "$DEPLOY/previous.env" "$DEPLOY/current.env"
  # the rollback target stays what it was before this failed attempt
  if [[ -f "$DEPLOY/previous.env.keep" ]]; then cp "$DEPLOY/previous.env.keep" "$DEPLOY/previous.env"; else rm -f "$DEPLOY/previous.env"; fi
  restore_data "$bk"
  op_set OP_PHASE RESTORED_DATA
  if [[ -z "$had_running" ]]; then
    # Fresh first-start recovery: old images were not running before the failed
    # candidate. Preserve the old env/data, but do NOT boot an unguarded Core.
    echo "$(date -u +%FT%TZ) update: → $cand FAILED; previous record/data preserved; application remains stopped" >>"$DEPLOY/history.log"
    op_end
    say "ROLLED BACK deployment record and data; the new application remains stopped (nothing was running before this attempt)."
    return 1
  fi
  compose "$DEPLOY/current.env" up -d --force-recreate >/dev/null 2>&1 || true
  if validate "$DEPLOY/current.env" "$before" "$was" "$ent"; then
    echo "$(date -u +%FT%TZ) update: → $cand FAILED, restored $(load_env "$DEPLOY/current.env"; echo "$CORE_SHA") and data ($bk)" >>"$DEPLOY/history.log"
    op_end
    say "ROLLED BACK: running the previous version again, with your data as it was before the update."
    return 1
  fi
  echo "$(date -u +%FT%TZ) update: → $cand FAILED and the restore did not validate" >>"$DEPLOY/history.log"
  op_end   # previous record and data ARE back in place; the services need attention
  die "the previous version didn't validate either: ./tass.sh logs (backup: $bk)"
}

cmd_rollback() {
  local restore=""
  [[ "${1:-}" == --restore-data ]] && restore="${2:?backup file}"
  [[ -f "$DEPLOY/previous.env" ]] || die "no previous version recorded"
  local before ent guard=""; before="$(inventory)"; ent="$(current_entity)"
  op_begin rollback "$(ours)"; op_set OP_PHASE SWITCHED
  compose "$DEPLOY/current.env" down >/dev/null 2>&1 || true
  cp "$DEPLOY/current.env" "$DEPLOY/rolled-back.env"
  cp "$DEPLOY/previous.env" "$DEPLOY/current.env"
  cp "$DEPLOY/rolled-back.env" "$DEPLOY/previous.env"
  if [[ -n "$restore" ]]; then
    guard="$(backup before-rollback-restore)" || die "could not make a verified snapshot of the current data first; nothing restored"
    op_set OP_BACKUP "$guard"; say "Current data saved first (verified): $guard"
    restore_data "$restore"; before="$(inventory)"
  fi
  compose "$DEPLOY/current.env" up -d --force-recreate >/dev/null 2>&1 || true
  if ! validate "$DEPLOY/current.env" "$before" "" "$ent"; then
    # Issue #79: a bad rollback is undone: go back to the version that was running before it.
    say "Rollback FAILED validation: returning to the version that was running before the rollback."
    compose "$DEPLOY/current.env" down >/dev/null 2>&1 || true
    cp "$DEPLOY/previous.env" "$DEPLOY/current.env"
    cp "$DEPLOY/rolled-back.env" "$DEPLOY/previous.env"
    [[ -n "$guard" ]] && restore_data "$guard"
    compose "$DEPLOY/current.env" up -d --force-recreate >/dev/null 2>&1 || true
    op_end
    if validate "$DEPLOY/current.env" "$before" "" "$ent"; then
      echo "$(date -u +%FT%TZ) rollback: FAILED; stayed on core $(load_env "$DEPLOY/current.env"; echo "$CORE_SHA")" >>"$DEPLOY/history.log"
      die "rollback failed; the version from before the rollback is running again"
    fi
    echo "$(date -u +%FT%TZ) rollback: FAILED and the return did not validate" >>"$DEPLOY/history.log"
    die "rollback failed and the previous version did not validate either: ./tass.sh logs"
  fi
  op_end
  echo "$(date -u +%FT%TZ) rollback: now core $(load_env "$DEPLOY/current.env"; echo "$CORE_SHA")" >>"$DEPLOY/history.log"
  say "Rolled back. (./tass.sh rollback again returns to the version you just left.)"
}

# Restore Previous Update (Issue #79): the exact version that ran before the last update AND the
# verified data snapshot taken right before it. The current data is first saved in a new verified
# snapshot, so this is itself undoable. Anything recorded after that update is set aside, never deleted.
cmd_restore_previous() {
  local yes=0; [[ "${1:-}" == --yes ]] && yes=1
  [[ -f "$DEPLOY/last-update.env" ]] || die "no recorded update to restore (updates made by older versions of this script don't record one)"
  local UPDATE_BACKUP UPDATE_BACKUP_SHA256 UPDATE_PREVIOUS UPDATE_CURRENT
  load_env "$DEPLOY/last-update.env"
  [[ -f "$UPDATE_PREVIOUS" ]] || die "the previous version's manifest $UPDATE_PREVIOUS is missing; nothing was changed"
  backup_ok "$UPDATE_BACKUP" || die "the pre-update backup $UPDATE_BACKUP is missing or changed; nothing was changed"
  local to from; to="$(load_env "$UPDATE_PREVIOUS"; echo "$CORE_SHA")"; from="$(load_env "$DEPLOY/current.env"; echo "$CORE_SHA")"
  say "Restore previous update: core $(short "$from") → $(short "$to"), data from $UPDATE_BACKUP."
  say "Anything recorded since that update is moved aside (kept, not deleted) and first saved in a new verified backup."
  if [[ $yes == 0 ]]; then read -r -p "Restore? [y/N] " yn; [[ "$yn" == [yY]* ]] || { say "Nothing changed."; return 0; }; fi
  local ent had_running guard; ent="$(current_entity)"; had_running="$(ours)"
  op_begin restore-previous "$had_running"
  compose "$DEPLOY/current.env" stop >/dev/null 2>&1 || true
  op_set OP_PHASE STOPPED
  if ! guard="$(backup "before-restore-from-${from:0:12}")"; then
    [[ -n "$had_running" ]] && compose "$DEPLOY/current.env" up -d >/dev/null 2>&1 || true
    op_end; die "could not make a verified snapshot of the current data; nothing was changed"
  fi
  say "Current data saved first (verified): $guard"
  op_set OP_BACKUP "$guard"; op_set OP_PHASE BACKED_UP
  cp "$DEPLOY/current.env" "$DEPLOY/rolled-back.env"
  cp "$UPDATE_PREVIOUS" "$DEPLOY/current.env"; cp "$DEPLOY/rolled-back.env" "$DEPLOY/previous.env"
  op_set OP_PHASE SWITCHED
  restore_data "$UPDATE_BACKUP"
  op_set OP_PHASE RESTORED_DATA
  local before; before="$(inventory)"
  compose "$DEPLOY/current.env" up -d --force-recreate >/dev/null 2>&1 || true
  if validate "$DEPLOY/current.env" "$before" "" "$ent"; then
    rm -f "$DEPLOY/last-update.env"
    { echo "UNDO_BACKUP=$guard"; echo "UNDO_ENV=$DEPLOY/rolled-back.env"; } >"$DEPLOY/last-restore.env"
    echo "$(date -u +%FT%TZ) restore-previous: $from → $to with data $UPDATE_BACKUP (current data saved in $guard)" >>"$DEPLOY/history.log"
    op_end
    say "RESTORED core $(short "$to") with its data from before the update. Undo: ./tass.sh rollback --restore-data $guard"
    return 0
  fi
  say "Restore FAILED validation: putting back the version and data that were running before the restore."
  compose "$DEPLOY/current.env" down >/dev/null 2>&1 || true
  cp "$DEPLOY/rolled-back.env" "$DEPLOY/current.env"
  restore_data "$guard"
  compose "$DEPLOY/current.env" up -d --force-recreate >/dev/null 2>&1 || true
  op_end
  echo "$(date -u +%FT%TZ) restore-previous: FAILED; back on $from with its data ($guard)" >>"$DEPLOY/history.log"
  validate "$DEPLOY/current.env" "$(inventory)" "" "$ent" || die "the restore failed and the version before it did not validate either: ./tass.sh logs"
  die "restore failed; the version and data from before the restore are running again"
}

# recover: finish an interrupted update/rollback/restore by putting back the version and data from
# before it (the recorded verified backup), then validating.
cmd_recover() {
  [[ -f "$(OPF)" ]] || { say "No interrupted operation recorded. Nothing to do."; return 0; }
  local OP_KIND OP_STARTED OP_PHASE OP_BACKUP OP_RUNNING; load_env "$(OPF)"
  say "Recovering an interrupted $OP_KIND (started $OP_STARTED, phase $OP_PHASE)."
  local ent; ent="$( [[ -s "$APP_DATA/identity/entity.json" ]] && json_field entity_id <"$APP_DATA/identity/entity.json" || true)"
  compose "$DEPLOY/current.env" down >/dev/null 2>&1 || true
  case "$OP_PHASE" in
    STARTING|STOPPED|BACKED_UP) ;;   # the deployment record was not switched yet; data untouched since
    *) [[ -f "$DEPLOY/operation-from.env" ]] || die "the record of the version before $OP_KIND is missing; nothing changed"
       cp "$DEPLOY/operation-from.env" "$DEPLOY/current.env"
       [[ -n "$OP_BACKUP" ]] && restore_data "$OP_BACKUP" ;;
  esac
  if [[ -n "$OP_RUNNING" ]]; then
    compose "$DEPLOY/current.env" up -d --force-recreate >/dev/null 2>&1 || true
    validate "$DEPLOY/current.env" "$(inventory)" "" "$ent" || die "recovered record and data, but the services did not validate: ./tass.sh logs"
  fi
  op_end
  echo "$(date -u +%FT%TZ) recover: interrupted $OP_KIND at $OP_PHASE → back on core $(load_env "$DEPLOY/current.env"; echo "$CORE_SHA")" >>"$DEPLOY/history.log"
  say "RECOVERED: core $(load_env "$DEPLOY/current.env"; short "$CORE_SHA") with the data from before the interrupted $OP_KIND."
}

cmd_history() {
  [[ -f "$DEPLOY/current.env" ]] && say "Current:  $(load_env "$DEPLOY/current.env"; echo "core $CORE_SHA · engine $ENGINE_SHA · identity $IDENTITY_SHA")"
  [[ -f "$DEPLOY/previous.env" ]] && say "Previous: $(load_env "$DEPLOY/previous.env"; echo "core $CORE_SHA · engine $ENGINE_SHA · identity $IDENTITY_SHA")"
  [[ -f "$DEPLOY/last-update.env" ]] && say "Restore previous update available: $(load_env "$DEPLOY/last-update.env"; echo "$UPDATE_BACKUP")"
  [[ -f "$(OPF)" ]] && say "INTERRUPTED operation recorded: run ./tass.sh recover"
  say "Backups (newest first):"
  local b
  for b in $(ls -1t "$DEPLOY"/backups/*.tgz 2>/dev/null | head -20); do
    if [[ -f "$b.sha256" ]]; then backup_ok "$b" && say "  verified  $b" || say "  CHANGED   $b (does not match its SHA-256)"
    else say "  unverified (older) $b"; fi
  done
  say "History (newest last):"; tail -n 20 "$DEPLOY/history.log" 2>/dev/null | sed 's/^/  /' || true
}

tass_main() {
settings
case "${1:-help}" in
  check) check && say "Ready: ./tass.sh start" || say "Fix the [stop] items, then ./tass.sh check again." ;;
  start) op_lock; op_guard; cmd_start ;;
  status) cmd_status; [[ -f "$(OPF)" ]] && say "  INTERRUPTED operation recorded ($(load_env "$(OPF)"; echo "$OP_KIND at $OP_PHASE")): ./tass.sh recover" || true ;;
  logs) compose "$DEPLOY/current.env" logs -f --tail 80 ;;
  check-update) shift; cmd_check_update "$@" ;;
  update) shift; op_lock; op_guard; cmd_update "$@" ;;
  rollback) shift; op_lock; op_guard; cmd_rollback "$@" ;;
  restore-previous) shift; op_lock; op_guard; cmd_restore_previous "$@" ;;
  recover) op_lock; cmd_recover ;;
  history) cmd_history ;;
  backup) op_lock; mkdir -p "$DEPLOY/backups" && chmod 700 "$DEPLOY/backups"
          # a consistent snapshot needs the services stopped; they are started again right after
          r="$(ours)"; [[ -n "$r" ]] && compose "$DEPLOY/current.env" stop >/dev/null 2>&1
          if b="$(backup manual)"; then say "Backup (verified): $b"; else say "Backup FAILED (nothing changed)"; fi
          [[ -n "$r" ]] && compose "$DEPLOY/current.env" up -d >/dev/null 2>&1; [[ -n "${b:-}" ]] ;;
  stop) op_lock; compose "$DEPLOY/current.env" stop && say "Stopped this application's containers (nothing else)." ;;
  *) sed -n '2,27p' "${BASH_SOURCE[0]}" ;;
esac
}

if [[ "${TASS_LIB_ONLY:-0}" == 1 ]]; then return 0 2>/dev/null || exit 0; fi
tass_main "$@"
