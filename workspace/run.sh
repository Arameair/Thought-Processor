#!/usr/bin/env bash
# Thought Processor workspace: start / stop.
#
#   ./run.sh            start model + liaison + workspace, then open http://127.0.0.1:8790
#   ./run.sh --cpu      same, without an NVIDIA GPU
#   ./run.sh stop | logs | status
#
# Run it as yourself (no sudo). If Docker needs sudo, only docker commands use it.
set -euo pipefail
cd "$(dirname "$0")"

action=start
files=(-f compose.yaml)
for arg in "$@"; do
  case "$arg" in
    --cpu) files=(-f compose.yaml -f compose.cpu.yaml) ;;
    start|stop|logs|status) action="$arg" ;;
    -h|--help) sed -n '2,8p' "$0"; exit 0 ;;
    *) echo "Unknown option: $arg (try --help)"; exit 2 ;;
  esac
done

# --- the real user (also when started with sudo) ---------------------------
if [[ $EUID -eq 0 && -n "${SUDO_USER:-}" && "${SUDO_USER}" != root ]]; then
  real_user="$SUDO_USER"; real_uid="$SUDO_UID"; real_gid="$SUDO_GID"
  real_home="$(getent passwd "$SUDO_USER" | cut -d: -f6)"
  as_user() { sudo -u "$real_user" -H "$@"; }
else
  real_user="$(id -un)"; real_uid="$(id -u)"; real_gid="$(id -g)"; real_home="$HOME"
  as_user() { "$@"; }
fi
export TASS_UID="${TASS_UID:-$real_uid}" TASS_GID="${TASS_GID:-$real_gid}"

# --- docker, with sudo only if needed ---------------------------------------
DOCKER=(docker)
command -v docker >/dev/null 2>&1 || { echo "Docker is not installed."; exit 1; }
docker_err="$(docker info 2>&1 >/dev/null)" || {
  if [[ "$docker_err" == *"permission denied"* && $EUID -ne 0 ]]; then
    echo "Docker needs sudo on this computer, so docker commands will run with sudo."
    keep="TASS_UID,TASS_GID,TASS_MODEL_DIR,TASS_MODEL_FILE,TASS_CTX,TASS_GPU_LAYERS,TASS_PORT,TASS_JUPYTER_PORT"
    keep+=",TASS_THINKING,TASS_AUTO_REFLECT,TASS_MAX_TOKENS,TASS_SOURCES,TASS_MIND_WRITE,TASS_DIRECT_REPOS,LIAISON_REF,LIAISON_REPOS"
    keep+=",LIAISON_CHAT_REPOS,LIAISON_OWNERS,LIAISON_BASE,WORKSPACE_BASE,WORKSPACE_PIP_FLAGS"
    DOCKER=(sudo "--preserve-env=$keep" docker)
    sudo true || { echo "sudo was refused."; exit 1; }
  fi
  if ! docker_err="$("${DOCKER[@]}" info 2>&1 >/dev/null)"; then
    echo "Docker isn't running."
    grep -qi microsoft /proc/version 2>/dev/null && echo "  In WSL try:  sudo service docker start" \
      || echo "  Try:  sudo systemctl start docker"
    exit 1
  fi
}
dc() { "${DOCKER[@]}" compose -p tass-workspace "${files[@]}" "$@"; }

case "$action" in
  stop) dc down; exit 0 ;;
  logs) dc logs -f --tail 50; exit 0 ;;
  status) dc ps; exit 0 ;;
esac

# --- folders and token -------------------------------------------------------
as_user mkdir -p secrets config shared data mirror outbox inbox
[[ -d secrets/github_token ]] && { rmdir secrets/github_token 2>/dev/null || { echo "secrets/github_token is a folder; remove it."; exit 1; }; }
if [[ ! -s secrets/github_token ]]; then
  for other in "$real_home/identity-liaison/liaison/secrets/github_token"; do
    [[ -s "$other" ]] && cp "$other" secrets/github_token && echo "Reused the GitHub token from $other" && break
  done
fi
if [[ ! -s secrets/github_token ]]; then
  if as_user gh auth token >/dev/null 2>&1; then
    (umask 177; as_user gh auth token > secrets/github_token)
    echo "Saved $real_user's GitHub CLI token to secrets/github_token (only the liaison container reads it)."
  else
    echo "Need a GitHub token: run  gh auth login  (as $real_user, without sudo), then this script again."
    exit 1
  fi
fi
chown -R "$real_uid:$real_gid" secrets config shared data mirror outbox inbox 2>/dev/null || true
chmod 700 secrets; chmod 600 secrets/github_token

# --- model file ----------------------------------------------------------------
model_dir="${TASS_MODEL_DIR:-./models}"
model_file="${TASS_MODEL_FILE:-NVIDIA-Nemotron3-Nano-4B-Q4_K_M.gguf}"
if [[ ! -f "$model_dir/$model_file" ]]; then
  for guess in "$real_home/nemotronTass/models" "$real_home/nemotronTASS/models" \
               "$real_home/nemotronTASS-docker/models" "$real_home/nemotronTASS-docker-liaison/models"; do
    [[ -f "$guess/$model_file" ]] && { model_dir="$guess"; break; }
  done
fi
[[ -f "$model_dir/$model_file" ]] || { echo "Model not found: $model_dir/$model_file"; echo "  Point at it:  TASS_MODEL_DIR=/path/to/models ./run.sh"; exit 1; }
export TASS_MODEL_DIR="$model_dir"
echo "Model: $TASS_MODEL_DIR/$model_file"

# --- one Liaison at a time -------------------------------------------------------
# This stack includes its own Liaison. Stop the separate identity-liaison stack so
# issues aren't answered twice and two models don't share the GPU.
if "${DOCKER[@]}" ps --format '{{.Names}}' | grep -q '^identity-liaison-'; then
  echo "Stopping the separate identity-liaison stack (this one includes the Liaison)…"
  "${DOCKER[@]}" ps --format '{{.Names}}' | grep '^identity-liaison-' | xargs "${DOCKER[@]}" stop >/dev/null
fi

echo "Building and starting… (first time: downloads llama.cpp + JupyterLab, give it a few minutes)"
if ! dc up -d --build --wait; then
  echo
  echo "Something didn't start. Last lines from the workspace:"
  "${DOCKER[@]}" logs --tail 25 tass-workspace-workspace-1 2>&1 || true
  echo "Paste this to Claude, or run  ./run.sh logs  for everything."
  exit 1
fi
if ! curl -s -m 5 -o /dev/null "http://127.0.0.1:${TASS_PORT:-8790}/api/status"; then
  echo "The page isn't answering on 127.0.0.1:${TASS_PORT:-8790} yet. Try again in a moment, or run  ./run.sh logs"
  exit 1
fi
echo
echo "Ready. Open:  http://127.0.0.1:${TASS_PORT:-8790}"
echo "  JupyterLab:  use the 'Open JupyterLab' button on that page (it carries the login token)"
echo "  Your shared space is also here in WSL:  $(pwd)/shared"
echo "  Direct TASS from GitHub: open an issue labelled 'tass' in ${TASS_DIRECT_REPOS:-Arameair/TASS}"
echo "  Watch:  ./run.sh logs      Stop:  ./run.sh stop"
