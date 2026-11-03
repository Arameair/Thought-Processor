#!/usr/bin/env bash
# Thought Processor workspace: start / stop.
#
#   ./run.sh            start model + liaison + workspace, then open http://127.0.0.1:8790
#   ./run.sh --cpu      same, without an NVIDIA GPU
#   ./run.sh stop | logs | status
#   ./run.sh discord    set up Discord for TASS (bot token, channel, your user ID), then start
#
# Run it as yourself (no sudo). If Docker needs sudo, only docker commands use it.
set -euo pipefail
cd "$(dirname "$0")"

action=start
files=(-f compose.yaml)
for arg in "$@"; do
  case "$arg" in
    --cpu) files=(-f compose.yaml -f compose.cpu.yaml) ;;
    start|stop|logs|status|discord) action="$arg" ;;
    -h|--help) sed -n '2,9p' "$0"; exit 0 ;;
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

# --- Discord (Communication Bridge config lives outside any Git checkout) -----------
export BRIDGE_DATA_DIR="${BRIDGE_DATA_DIR:-$real_home/.communication-bridge}"
if [[ "$action" == discord ]]; then
  cfg="$BRIDGE_DATA_DIR/discord.json"
  echo "Discord setup for TASS. You need a bot from https://discord.com/developers/applications"
  echo "  Bot page: Reset Token (copy it) and turn on MESSAGE CONTENT INTENT."
  echo "  OAuth2 > URL Generator: scope 'bot', permissions View Channels, Send Messages, Read Message History;"
  echo "  open the URL and add the bot to your server."
  echo "  Discord > Settings > Advanced > Developer Mode on; right-click the channel > Copy Channel ID,"
  echo "  right-click your name > Copy User ID."
  if [[ -s "$cfg" ]]; then
    read -r -p "$cfg already exists. Replace it? [y/N] " yn
    [[ "$yn" == [yY]* ]] || { echo "Keeping it."; action=start; }
  fi
  if [[ "$action" == discord ]]; then
    read -r -s -p "Bot token (hidden): " tok; echo
    read -r -p "Channel ID: " chan
    read -r -p "Your Discord user ID: " owner
    [[ ${#tok} -ge 50 && "$chan" =~ ^[0-9]{15,22}$ && "$owner" =~ ^[0-9]{15,22}$ ]] || {
      echo "That doesn't look right (token is long; IDs are 15-22 digits). Nothing saved."; exit 1; }
    as_user mkdir -p "$BRIDGE_DATA_DIR"; chmod 700 "$BRIDGE_DATA_DIR"
    (umask 177; TOK="$tok" CHAN="$chan" OWNER="$owner" python3 -c 'import json,os; print(json.dumps({"token": os.environ["TOK"], "channel_id": os.environ["CHAN"], "owner_id": os.environ["OWNER"]}))' > "$cfg")
    chown "$real_uid:$real_gid" "$BRIDGE_DATA_DIR" "$cfg" 2>/dev/null || true
    unset tok
    echo "Saved $cfg (only you can read it; only the bridge container uses the token)."
    action=start
  fi
fi
if [[ -s "$BRIDGE_DATA_DIR/discord.json" || "$action" != start ]]; then
  export COMPOSE_PROFILES=discord        # stop/logs/status always include the Discord services
fi

# --- docker, with sudo only if needed ---------------------------------------
DOCKER=(docker)
command -v docker >/dev/null 2>&1 || { echo "Docker is not installed."; exit 1; }
docker_err="$(docker info 2>&1 >/dev/null)" || {
  if [[ "$docker_err" == *"permission denied"* && $EUID -ne 0 ]]; then
    echo "Docker needs sudo on this computer, so docker commands will run with sudo."
    keep="TASS_UID,TASS_GID,TASS_MODEL_DIR,TASS_MODEL_FILE,TASS_CTX,TASS_GPU_LAYERS,TASS_PORT,TASS_JUPYTER_PORT"
    keep+=",TASS_THINKING,TASS_AUTO_REFLECT,TASS_MAX_TOKENS,TASS_SOURCES,TASS_MIND_WRITE,TASS_DIRECT_REPOS,LIAISON_REF,LIAISON_REPOS"
    keep+=",LIAISON_CHAT_REPOS,LIAISON_OWNERS,LIAISON_BASE,WORKSPACE_BASE,WORKSPACE_PIP_FLAGS"
    keep+=",BRIDGE_DATA_DIR,BRIDGE_REF,BRIDGE_PORT,BRIDGE_GO_IMAGE,BRIDGE_RUN_IMAGE,DOOR_BASE,COMPOSE_PROFILES"
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
as_user mkdir -p secrets config shared shared/discord data mirror outbox inbox
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

# --- Communication Bridge source (private repo, so it's fetched with your gh login, not by Docker) ---
if [[ "${COMPOSE_PROFILES:-}" == *discord* && -z "${BRIDGE_SRC:-}" ]]; then
  bref="${BRIDGE_REF:-experiment/communication-bridge-v1-20261020}"
  echo "Fetching Arameair/Communication-Bridge ($bref)…"
  as_user rm -rf vendor/communication-bridge.new
  if as_user gh repo clone Arameair/Communication-Bridge vendor/communication-bridge.new -- -q --depth 1 --branch "$bref"; then
    as_user rm -rf vendor/communication-bridge && as_user mv vendor/communication-bridge.new vendor/communication-bridge
  elif [[ -d vendor/communication-bridge ]]; then
    echo "  Couldn't fetch it; using the copy from last time."
  else
    echo "  Couldn't fetch Arameair/Communication-Bridge (check  gh auth status). Starting without Discord."
    unset COMPOSE_PROFILES
  fi
fi

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
echo "  ⚙ Processor (Claude's station): http://127.0.0.1:8790/processor"
echo "  ✦ Atrium (ChatGPT's station): http://127.0.0.1:8791/   — open each in its own browser"
echo "  JupyterLab:  use the 'Open JupyterLab' button on that page (it carries the login token)"
echo "  Your shared space is also here in WSL:  $(pwd)/shared"
if [[ "${COMPOSE_PROFILES:-}" == *discord* ]]; then
  echo "  Discord: on. Bridge page: http://127.0.0.1:${BRIDGE_PORT:-7820}   TASS learns it with the lesson 'Talk in Discord'"
else
  echo "  Discord: not set up yet. To teach TASS to talk in Discord:  ./run.sh discord"
fi
echo "  Direct TASS from GitHub: open an issue labelled 'tass' in ${TASS_DIRECT_REPOS:-Arameair/TASS}"
echo "  Watch:  ./run.sh logs      Stop:  ./run.sh stop"
