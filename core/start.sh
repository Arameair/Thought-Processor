#!/usr/bin/env bash
# TASS's Go core: start / stop.
#
#   ./start.sh           build, then start: Nemotron (llama.cpp in Docker) → Thought Engine → tass-core
#   ./start.sh stop      stop tass-core and the Thought Engine (the model keeps running; ./start.sh stop-all stops it too)
#   ./start.sh status    what's running, and the last pulse and sync
#   ./start.sh logs      follow the core's and the engine's logs
#
# Talk to TASS:  http://127.0.0.1:8795      Thought Engine: http://127.0.0.1:7798
# Run it as yourself (no sudo). If Docker needs sudo, only docker commands use it.
set -euo pipefail
cd "$(dirname "$0")"
here="$(pwd)"

ENGINE_REF="${ENGINE_REF:-experiment/nemotron-link-20261104}"
TASS_HOME="${TASS_DATA:-$HOME/.tass}"
ENGINE_DATA="${ENGINE_DATA:-$HOME/.thought-engine/tass}"
LOGS="$TASS_HOME/logs"
mkdir -p "$LOGS" bin
chmod 700 "$TASS_HOME"

DOCKER=(docker)
if command -v docker >/dev/null && ! docker info >/dev/null 2>&1; then
  sudo -n true 2>/dev/null || echo "Docker needs sudo on this computer; you may be asked for your password."
  DOCKER=(sudo --preserve-env=TASS_MODEL_DIR,TASS_MODEL_FILE,TASS_UID,TASS_GID docker)
fi
dc() { "${DOCKER[@]}" compose -p tass-workspace -f ../workspace/compose.yaml "$@"; }

pid_of() { [[ -f "$LOGS/$1.pid" ]] && kill -0 "$(cat "$LOGS/$1.pid")" 2>/dev/null && cat "$LOGS/$1.pid"; }
stop_one() {
  local p; p="$(pid_of "$1" || true)"
  if [[ -n "$p" ]]; then kill -TERM "$p"; for _ in $(seq 20); do kill -0 "$p" 2>/dev/null || break; sleep 0.25; done; echo "Stopped $1."; fi
  rm -f "$LOGS/$1.pid"
}

case "${1:-start}" in
  stop) stop_one tass-core; stop_one thought-engine; exit 0 ;;
  stop-all) stop_one tass-core; stop_one thought-engine; dc stop model >/dev/null 2>&1 || true; echo "Stopped the model."; exit 0 ;;
  logs) exec tail -n 40 -F "$LOGS/tass-core.log" "$LOGS/thought-engine.log" ;;
  status)
    for n in thought-engine tass-core; do printf '%-15s %s\n' "$n" "$( [[ -n "$(pid_of $n || true)" ]] && echo running || echo stopped)"; done
    echo "last engine pulse: $(tail -n 1 "$ENGINE_DATA/pulse.jsonl" 2>/dev/null || echo none)"
    echo "last GitHub sync:  $(tail -n 1 "$TASS_HOME/sync.jsonl" 2>/dev/null || echo none)"
    exit 0 ;;
  start) ;;
  *) sed -n '2,10p' "$0"; exit 2 ;;
esac

command -v go >/dev/null || { echo "Go is needed: https://go.dev/dl/ (1.22 or newer)"; exit 1; }
command -v gh >/dev/null || { echo "The GitHub CLI (gh) is needed: https://cli.github.com"; exit 1; }

# --- GitHub token (only tass-core reads it; never the engine or the model) ---------
if [[ ! -s secrets/github_token ]]; then
  mkdir -p secrets; chmod 700 secrets
  if [[ -s ../workspace/secrets/github_token ]]; then cp ../workspace/secrets/github_token secrets/github_token
  else (umask 177; gh auth token > secrets/github_token) || { echo "Run  gh auth login  first."; exit 1; }; fi
  chmod 600 secrets/github_token
fi

# --- one TASS at a time --------------------------------------------------------------
# The Python workspace stack also answers 'tass' issues; stop its liaison and workspace so
# TASS doesn't answer twice. Its model container is reused below.
if "${DOCKER[@]}" ps --format '{{.Names}}' 2>/dev/null | grep -Eq '^tass-workspace-(liaison|workspace|atrium)-'; then
  echo "Stopping the Python workspace's liaison and workspace (the Go core answers TASS's messages now)…"
  dc stop liaison workspace atrium bridge discord-door >/dev/null 2>&1 || true
fi

# --- Nemotron: llama.cpp in Docker, weights on local storage --------------------------
model_file="${TASS_MODEL_FILE:-NVIDIA-Nemotron3-Nano-4B-Q4_K_M.gguf}"
if [[ -z "${TASS_MODEL_DIR:-}" ]]; then
  for d in "$HOME/nemotronTass/models" "$HOME/nemotronTASS/models" ../workspace/models; do
    [[ -f "$d/$model_file" ]] && { TASS_MODEL_DIR="$(cd "$d" && pwd)"; break; }
  done
fi
[[ -f "${TASS_MODEL_DIR:-/nonexistent}/$model_file" ]] || { echo "Model not found. Point at it:  TASS_MODEL_DIR=/path/to/models ./start.sh"; exit 1; }
export TASS_MODEL_DIR TASS_MODEL_FILE="$model_file" TASS_UID="$(id -u)" TASS_GID="$(id -g)"
mkdir -p ../workspace/secrets && touch ../workspace/secrets/github_token   # compose file needs it to exist
echo "Starting Nemotron (model from $TASS_MODEL_DIR)…"
dc up -d --wait model >/dev/null

# --- build ------------------------------------------------------------------------------
echo "Fetching Thought Engine ($ENGINE_REF)…"
rm -rf vendor/thought-engine.new
if gh repo clone Arameair/Thought-engine vendor/thought-engine.new -- -q --depth 1 --branch "$ENGINE_REF"; then
  rm -rf vendor/thought-engine && mv vendor/thought-engine.new vendor/thought-engine
elif [[ ! -d vendor/thought-engine ]]; then echo "Couldn't fetch Arameair/Thought-engine (check  gh auth status)."; exit 1
else echo "  Couldn't fetch it; using the copy from last time."; fi
(cd vendor/thought-engine && go build -o "$here/bin/thought-engine" .)
go build -o bin/tass-core ./cmd/tass-core
echo "Built: engine $(git -C vendor/thought-engine rev-parse --short HEAD), core $(git rev-parse --short HEAD 2>/dev/null || echo '?')"

# --- start --------------------------------------------------------------------------------
stop_one tass-core; stop_one thought-engine
mkdir -p "$ENGINE_DATA"
PORT=7798 nohup bin/thought-engine -data "$ENGINE_DATA" -model http://127.0.0.1:18083 \
  -identity-policy "$TASS_HOME/identity-policy.json" >>"$LOGS/thought-engine.log" 2>&1 &
echo $! > "$LOGS/thought-engine.pid"
for _ in $(seq 40); do curl -sf -o /dev/null http://127.0.0.1:7798/api/model && break; sleep 0.25; done
TASS_DATA="$TASS_HOME" TASS_TOKEN_FILE="$here/secrets/github_token" nohup bin/tass-core >>"$LOGS/tass-core.log" 2>&1 &
echo $! > "$LOGS/tass-core.pid"
for _ in $(seq 40); do curl -sf -o /dev/null http://127.0.0.1:8795/api/status && break; sleep 0.25; done
curl -sf -o /dev/null http://127.0.0.1:8795/api/status || { echo "tass-core didn't start:"; tail -n 20 "$LOGS/tass-core.log"; exit 1; }

echo
echo "TASS is running."
echo "  Talk to him:     http://127.0.0.1:8795   (works without internet)"
echo "  Thought Engine:  http://127.0.0.1:7798/api/model   (its 20-second pulse: $ENGINE_DATA/pulse.jsonl)"
echo "  His local files: $TASS_HOME   (tass.md, sync.jsonl, conversation.jsonl)"
echo "  Stop: ./start.sh stop     Status: ./start.sh status     Logs: ./start.sh logs"
