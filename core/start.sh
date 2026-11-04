#!/usr/bin/env bash
# TASS's Go core: check, start, stop.
#
#   ./start.sh check     READ-ONLY preflight: tools, ports, containers, model, Identity policy, conflicts. Changes nothing.
#   ./start.sh           start: Thought Engine (Nemotron worker) + tass-core. Stops nothing that's already running.
#   ./start.sh stop      stop tass-core and the Thought Engine started by this script (nothing else)
#   ./start.sh status    what's running, the engine's last pulse, the last GitHub sync
#   ./start.sh logs      follow the core's and the engine's logs
#
# Talk to TASS: http://127.0.0.1:8795      Rollback: ./start.sh stop, then delete ~/.tass and ~/.thought-engine/tass
# Run as yourself (no sudo). If Docker needs sudo, only docker commands use it.
set -euo pipefail
cd "$(dirname "$0")"
here="$(pwd)"

# The Thought Engine is pinned to a reviewed commit (never a moving branch head).
ENGINE_REPO="Arameair/Thought-engine"
ENGINE_COMMIT="${ENGINE_COMMIT:-4924de6f1b8504abcab4fed98c0e191057f4e277}"
TASS_HOME="${TASS_DATA:-$HOME/.tass}"
ENGINE_DATA="${ENGINE_DATA:-$HOME/.thought-engine/tass}"
ENGINE_PORT="${ENGINE_PORT:-7798}"
CORE_PORT="${CORE_PORT:-8795}"
MODEL_URL="${MODEL_URL:-http://127.0.0.1:18083}"
LOGS="$TASS_HOME/logs"

say()  { printf '%s\n' "$*"; }
ok()   { printf '  [ok]   %s\n' "$*"; }
warn() { printf '  [note] %s\n' "$*"; }
bad()  { printf '  [stop] %s\n' "$*"; }

DOCKER=(docker)
docker_ok=1
if ! command -v docker >/dev/null; then docker_ok=0
elif ! docker info >/dev/null 2>&1; then
  if sudo -n docker info >/dev/null 2>&1; then DOCKER=(sudo -n docker); else docker_ok=0; fi
fi
port_busy() { (exec 3<>"/dev/tcp/127.0.0.1/$1") 2>/dev/null; }
pid_of() { [[ -f "$LOGS/$1.pid" ]] && kill -0 "$(cat "$LOGS/$1.pid")" 2>/dev/null && cat "$LOGS/$1.pid"; }
python_liaison_running() { [[ $docker_ok == 1 ]] && "${DOCKER[@]}" ps --format '{{.Names}}' 2>/dev/null | grep -q '^tass-workspace-liaison-'; }

preflight() {   # read-only
  local blockers=0
  say "Preflight (read-only: nothing is started, stopped or changed)"
  command -v go >/dev/null && ok "Go: $(go version | awk '{print $3}')" || { bad "Go 1.22+ is needed: https://go.dev/dl/"; blockers=1; }
  command -v gh >/dev/null && gh auth status >/dev/null 2>&1 && ok "GitHub CLI signed in" || { bad "GitHub CLI: run  gh auth login"; blockers=1; }
  if curl -sf -m 3 -o /dev/null "$MODEL_URL/health"; then ok "Nemotron (llama.cpp) answers at $MODEL_URL"
  else
    warn "No model answering at $MODEL_URL."
    say  "         Start it the usual way (cd ../workspace && ./run.sh), or only the model container:"
    say  "         (cd ../workspace && docker compose -p tass-workspace up -d model). This script won't start it for you."
    blockers=1
  fi
  for p in "$ENGINE_PORT:Thought Engine:thought-engine:ENGINE_PORT" "$CORE_PORT:tass-core:tass-core:CORE_PORT"; do
    IFS=: read -r port name pidname var <<<"$p"
    if [[ -n "$(pid_of "$pidname" || true)" ]]; then ok "$name already running from this script (port $port)"
    elif port_busy "$port"; then bad "Port $port ($name) is used by another program. Use another port ($var=...) or stop that program yourself."; blockers=1
    else ok "Port $port free for $name"; fi
  done
  if python_liaison_running; then
    warn "The Python workspace's liaison is running: it also answers 'tass' issues on GitHub."
    say  "         The Go core will start with GitHub messages PAUSED so TASS doesn't answer twice. Nothing is stopped."
  fi
  if [[ -s "$TASS_HOME/grant/identity-grant.json" ]]; then ok "Identity policy cached locally (from the last sync)"
  else warn "No Identity policy yet: TASS won't act until policies/tass.json is on Arameair/identity main (PR #31, owner review)."; fi
  [[ "$ENGINE_COMMIT" =~ ^[0-9a-f]{40}$ ]] && ok "Thought Engine pinned to $ENGINE_COMMIT" || { bad "ENGINE_COMMIT isn't a full commit SHA"; blockers=1; }
  return $blockers
}

stop_one() {
  local p; p="$(pid_of "$1" || true)"
  if [[ -n "$p" ]]; then kill -TERM "$p"; for _ in $(seq 40); do kill -0 "$p" 2>/dev/null || break; sleep 0.25; done; say "Stopped $1."; fi
  rm -f "$LOGS/$1.pid"
}

case "${1:-start}" in
  check) preflight && say "Ready to start: ./start.sh" || say "Fix the [stop] items, then run ./start.sh check again."; exit 0 ;;
  stop) stop_one tass-core; stop_one thought-engine; exit 0 ;;
  logs) mkdir -p "$LOGS"; exec tail -n 40 -F "$LOGS/tass-core.log" "$LOGS/thought-engine.log" ;;
  status)
    for n in thought-engine tass-core; do printf '%-15s %s\n' "$n" "$( [[ -n "$(pid_of $n || true)" ]] && echo running || echo stopped)"; done
    echo "last engine pulse: $(tail -n 1 "$ENGINE_DATA/pulse.jsonl" 2>/dev/null || echo none)"
    echo "last GitHub sync:  $(tail -n 1 "$TASS_HOME/sync.jsonl" 2>/dev/null || echo none)"
    exit 0 ;;
  start) ;;
  *) sed -n '2,11p' "$0"; exit 2 ;;
esac

preflight || { say; say "Not starting: fix the [stop] items above (nothing was changed)."; exit 1; }
say
mkdir -p "$LOGS" && chmod 700 "$TASS_HOME"

# --- GitHub messages: never answer twice -----------------------------------------
messages=on
if python_liaison_running; then
  messages=off
  if [[ -t 0 ]]; then
    read -r -p "The Python liaison answers 'tass' issues too. Stop it so the Go core answers them instead? [y/N] " yn
    if [[ "$yn" == [yY]* ]]; then
      (cd ../workspace && "${DOCKER[@]}" compose -p tass-workspace stop liaison) && messages=on && say "Stopped the Python liaison (restart it with: cd ../workspace && ./run.sh)."
    fi
  fi
  [[ $messages == off ]] && say "GitHub messages PAUSED in the Go core (TASS_GITHUB_MESSAGES=off). Local chat works."
fi

# --- GitHub token (only tass-core reads it; never the engine or the model) ---------
if [[ ! -s secrets/github_token ]]; then
  mkdir -p secrets && chmod 700 secrets
  (umask 177; gh auth token > secrets/github_token)
fi
chmod 600 secrets/github_token

# --- build (engine at the pinned commit) -------------------------------------------
mkdir -p bin vendor
if [[ "$(git -C vendor/thought-engine rev-parse HEAD 2>/dev/null || true)" != "$ENGINE_COMMIT" ]]; then
  say "Fetching $ENGINE_REPO at $ENGINE_COMMIT…"
  rm -rf vendor/thought-engine.new
  gh repo clone "$ENGINE_REPO" vendor/thought-engine.new -- -q --no-checkout
  git -C vendor/thought-engine.new checkout -q "$ENGINE_COMMIT"
  [[ "$(git -C vendor/thought-engine.new rev-parse HEAD)" == "$ENGINE_COMMIT" ]] || { bad "commit check failed"; exit 1; }
  rm -rf vendor/thought-engine && mv vendor/thought-engine.new vendor/thought-engine
fi
(cd vendor/thought-engine && go build -o "$here/bin/thought-engine" .)
go build -o bin/tass-core ./cmd/tass-core
say "Built: Thought Engine ${ENGINE_COMMIT:0:12}, tass-core $(git rev-parse --short=12 HEAD 2>/dev/null || echo '?')"

# --- start ----------------------------------------------------------------------------
stop_one tass-core; stop_one thought-engine
mkdir -p "$ENGINE_DATA" && chmod 700 "$ENGINE_DATA"
PORT="$ENGINE_PORT" nohup bin/thought-engine -data "$ENGINE_DATA" -model "$MODEL_URL" -worker nemotron \
  -identity-policy "$TASS_HOME/grant/identity-grant.json" >>"$LOGS/thought-engine.log" 2>&1 &
echo $! > "$LOGS/thought-engine.pid"
for _ in $(seq 40); do curl -sf -o /dev/null "http://127.0.0.1:$ENGINE_PORT/api/model" && break; sleep 0.25; done
curl -sf -o /dev/null "http://127.0.0.1:$ENGINE_PORT/api/model" || { bad "The Thought Engine didn't start:"; tail -n 20 "$LOGS/thought-engine.log"; exit 1; }
TASS_DATA="$TASS_HOME" TASS_TOKEN_FILE="$here/secrets/github_token" TASS_ENGINE_URL="http://127.0.0.1:$ENGINE_PORT" \
  TASS_ENGINE_KEY="$ENGINE_DATA/api-key" TASS_LISTEN="127.0.0.1:$CORE_PORT" TASS_GITHUB_MESSAGES="$messages" \
  nohup bin/tass-core >>"$LOGS/tass-core.log" 2>&1 &
echo $! > "$LOGS/tass-core.pid"
for _ in $(seq 40); do port_busy "$CORE_PORT" && break; sleep 0.25; done
port_busy "$CORE_PORT" || { bad "tass-core didn't start:"; tail -n 20 "$LOGS/tass-core.log"; exit 1; }

say
say "TASS is running."
say "  Talk to him:     http://127.0.0.1:$CORE_PORT   (works without internet)"
say "  His local files: $TASS_HOME   (tass.md, memory.jsonl, sync.jsonl, conversation.jsonl)"
say "  Engine records:  $ENGINE_DATA   (cycles.jsonl, model-requests.jsonl, pulse.jsonl)"
say "  Stop: ./start.sh stop     Status: ./start.sh status     Logs: ./start.sh logs"
