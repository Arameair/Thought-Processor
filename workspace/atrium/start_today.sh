#!/usr/bin/env bash
# Atrium daily desk: standalone local UI; does not start or stop Docker services.
set -Eeuo pipefail

HERE="$(cd -- "$(dirname -- "$0")" && pwd)"
STATE_DIR="${TASS_ATRIUM_STATE_DIR:-$HOME/.local/state/tass-atrium}"
PORT="${TASS_ATRIUM_PORT:-8793}"
BACKEND_PORT="${TASS_WORKSHOP_PORT:-8790}"

if ! command -v python3 >/dev/null 2>&1; then
  echo "BLOCKED: Python 3 is required; nothing was started." >&2
  exit 1
fi
if [[ ! "$PORT" =~ ^[0-9]+$ || ! "$BACKEND_PORT" =~ ^[0-9]+$ ]] ||
  (( PORT < 1024 || PORT > 65535 || BACKEND_PORT < 1024 || BACKEND_PORT > 65535 || PORT == BACKEND_PORT )); then
  echo "BLOCKED: Invalid Atrium/Workshop port selection; nothing was started." >&2
  exit 1
fi

umask 077
mkdir -p -- "$STATE_DIR"
chmod 700 -- "$STATE_DIR"
export TASS_ATRIUM_DESK_STATE="$STATE_DIR/desk.json"
export TASS_ATRIUM_BIND_HOST=127.0.0.1

echo "ATRIUM DAILY DESK: http://127.0.0.1:$PORT/"
echo "Workshop backend expected: http://127.0.0.1:$BACKEND_PORT/"
echo "Local task notes: $TASS_ATRIUM_DESK_STATE"
echo "No Docker containers, branches, permissions, or GitHub files will be changed."
echo "Open the Atrium URL in your browser. Stop Atrium with Ctrl+C."
cd -- "$HERE"
exec python3 -u server.py --port "$PORT" --backend-port "$BACKEND_PORT"
