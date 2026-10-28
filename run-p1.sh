#!/usr/bin/env bash
# P1 cleanup launcher. No authentication or approval is automated.
# Do not trace session URLs or pairing challenges even if invoked with bash -x.
set +x
set -Eeuo pipefail
umask 077
BASE_BRANCH=experiment/p1-cleanup-claude-20261026
LAUNCH_BRANCH=experiment/p1-launcher-gpt6-20261028
IDENTITY_SHA=64940b629a5ef52eaa143b2402726ec81d382002
PROCESSOR_BASE_SHA=db31e4785d292925249a2f9f06e3c06bd78ba5b0
ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)"
ID_SRC="${P1_IDENTITY_SRC:-$(dirname "$ROOT")/identity}"
WORK="${P1_LAUNCH_WORK:-$HOME/p1c/p1-launcher}"
ID_PORT="${P1_IDENTITY_PORT:-7792}"
TP_PORT="${P1_PROCESSOR_PORT:-7862}"
id_pid= tp_pid= watch_pid= lock=
INJECT="${P1_TEST_INJECT:-}"
die() { printf 'ERROR: %s\n' "$*" >&2; exit 1; }
need() { command -v "$1" >/dev/null 2>&1 || die "Missing $1 (install it and retry)."; }
stop() {
  trap - EXIT INT TERM
  if [[ -n "$watch_pid" ]]; then
    kill -TERM "$watch_pid" 2>/dev/null || :
    wait "$watch_pid" 2>/dev/null || :
  fi
  for pid in "$tp_pid" "$id_pid"; do
    if [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null; then
      kill -TERM "$pid" 2>/dev/null || :
      for ((i=0;i<30;i++)); do kill -0 "$pid" 2>/dev/null || break; sleep 0.1; done
      if kill -0 "$pid" 2>/dev/null; then
        printf 'WARNING: owned process %s did not exit; inspect it manually.\n' "$pid" >&2
      fi
      wait "$pid" 2>/dev/null || :
    fi
  done
  [[ -z "$lock" ]] || rmdir "$lock" 2>/dev/null || :
}
trap stop EXIT
trap 'exit 130' INT
trap 'exit 143' TERM
for cmd in git go python3 mkdir mktemp; do
  if [[ "$INJECT" == "missing-$cmd" ]]; then die "INJECTED: Missing $cmd (install it and retry)."; fi
  need "$cmd"
done
[[ -d "$ID_SRC/.git" && -d "$ROOT/.git" ]] || die "Both cloned repos required. Expected Identity at $ID_SRC and Processor at $ROOT. Set P1_IDENTITY_SRC if needed."
[[ "$(git -C "$ID_SRC" branch --show-current)" == "$BASE_BRANCH" ]] || die "Identity must be checked out on $BASE_BRANCH (no checkout performed)."
[[ "$(git -C "$ROOT" branch --show-current)" == "$LAUNCH_BRANCH" ]] || die "Processor must be checked out on $LAUNCH_BRANCH (no checkout performed)."
id_sha="$(git -C "$ID_SRC" rev-parse HEAD)"
tp_sha="$(git -C "$ROOT" rev-parse HEAD)"
[[ "$id_sha" == "$IDENTITY_SHA" ]] || die "Identity changed: expected $IDENTITY_SHA, found $id_sha. Review before proceeding."
git -C "$ROOT" merge-base --is-ancestor "$PROCESSOR_BASE_SHA" HEAD || die "Processor is not descended from the reviewed P1 cleanup commit."
for src in "$ID_SRC" "$ROOT"; do
  [[ -z "$(git -C "$src" status --porcelain --untracked-files=normal)" ]] || die "Dirty working tree in $src. Commit/stash/review changes first; nothing modified."
done
printf 'Identity P1 source: %s\nProcessor P1 launcher: %s (base %s)\n' "$id_sha" "$tp_sha" "$PROCESSOR_BASE_SHA"
python3 - "$ID_PORT" "$TP_PORT" <<'PY'
import socket,sys
try:
    ports=[int(p) for p in sys.argv[1:]]
    if any(not 1024 <= p <= 65535 for p in ports) or len(set(ports)) != len(ports):
        raise ValueError("ports must be distinct and between 1024 and 65535")
    socks=[]
    for port in ports:
        s=socket.socket()
        s.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
        s.bind(("127.0.0.1",port))
        socks.append(s)
    for s in socks: s.close()
except (OSError,ValueError) as e:
    sys.exit("ERROR: P1 trial ports busy/invalid: %s. Choose unused P1_IDENTITY_PORT and P1_PROCESSOR_PORT; no existing process was stopped." % e)
PY
[[ -d "$WORK" || ! -e "$WORK" ]] || die "Work path is not a directory: $WORK"
mkdir -p "$WORK"
WORK="$(cd "$WORK" && pwd -P)"
[[ ! -L "$WORK" ]] || die "Work directory must not be a symlink."
for dir in "$WORK/identity-profile" "$WORK/processor-profile"; do
  [[ ! -L "$dir" ]] || die "Refusing symlinked profile $dir"
done
lock_candidate="$WORK/.launcher-running"
mkdir "$lock_candidate" 2>/dev/null || die "Launcher lock exists at $lock_candidate (another run or stale lock). Check existing processes before removing it."
lock="$lock_candidate"
mkdir -p "$WORK/bin" "$WORK/identity-profile" "$WORK/processor-profile"
chmod 700 "$WORK" "$WORK/bin" "$WORK/identity-profile" "$WORK/processor-profile"
# Preserve reusable profiles: a service marker alone is not proof of a live process.
# The launcher lock, port guard and Identity's own profile lock arbitrate use.
case "$INJECT" in ""|missing-go|missing-python3|go-build|identity-start|processor-start) ;; *) die "Unknown P1_TEST_INJECT" ;; esac
if [[ -n "$INJECT" && "$WORK" != *"/p1c/"* ]]; then die "Injection requires disposable /p1c/ work path"; fi
if [[ "$INJECT" == go-build ]]; then die "INJECTED: Go build failure (no children started)"; fi
printf 'Building both source trees with Go...\n'
(cd "$ID_SRC" && go build -ldflags "-X main.buildBranch=$BASE_BRANCH" -o "$WORK/bin/identity-p1" .) || die "Identity Go build failed"
(cd "$ROOT/processor" && go build -ldflags "-X main.buildBranch=$BASE_BRANCH" -o "$WORK/bin/processor-p1" .) || die "Processor Go build failed"
printf 'Starting private P1 trial. Logs stay local in %s (DO NOT SHARE).\n' "$WORK"
# Locator isolation ensures we do not discover unrelated Identity instances.
export IDENTITY_LOCATOR_DIR="$WORK/locators"
export TP_IDENTITY_DEFAULT_URL="http://127.0.0.1:$ID_PORT"
mkdir -p "$IDENTITY_LOCATOR_DIR"
if [[ "$INJECT" == identity-start ]]; then
  (exit 42) >"$WORK/identity-private.log" 2>&1 &
else
  "$WORK/bin/identity-p1" -port "$ID_PORT" -data "$WORK/identity-profile" >"$WORK/identity-private.log" 2>&1 &
fi
id_pid=$!
sleep 0.5
kill -0 "$id_pid" 2>/dev/null || die "Identity exited: inspect $WORK/identity-private.log locally (private codes/links may be present)."
# Read only new Identity pairing announcements in this launcher-owned session.
# Identity issues the code; this watcher never requests, changes or approves it.
# Never start when either input or output is redirected/noninteractive.
start_pairing_watcher() {
  [[ -t 0 && -t 1 ]] || return 0
  python3 -u - "$WORK/identity-private.log" <<'PY' &
import os
import re
import sys
import time

if not os.isatty(1):
    sys.exit(0)
# Exact code alphabet and announcement shape from pinned Identity source.
code_line = re.compile(r"  \|          ([A-HJKMNP-Z2-9]{4}-[A-HJKMNP-Z2-9]{4})")
try:
    with open(sys.argv[1], encoding="utf-8", errors="replace") as log:
        # Ignore all prior bytes, including any old challenge on startup/restart.
        log.seek(0, os.SEEK_END)
        remaining = 0
        while True:
            line = log.readline()
            if not line:
                time.sleep(0.1)
                continue
            line = line.rstrip("\r\n")
            if line.startswith("  +-- PAIRING REQUEST from "):
                remaining = 7
                continue
            if not remaining:
                continue
            match = code_line.fullmatch(line)
            if match:
                print("\nPRIVATE Identity pairing challenge (DO NOT SHARE; local terminal only):\n"
                      + match.group(1) +
                      "\nEnter this code manually in Processor. No approval is automated.\n",
                      flush=True)
                remaining = 0  # Exactly once per fresh announcement.
            else:
                remaining -= 1  # Never scan arbitrary log lines or other secrets.
except OSError:
    pass  # Identity or its log became unavailable; no secret fallback to shared output.
PY
  watch_pid=$!
}
start_pairing_watcher
if [[ "$INJECT" == processor-start ]]; then
  (exit 42) >"$WORK/processor-private.log" 2>&1 &
else
  "$WORK/bin/processor-p1" -port "$TP_PORT" -data "$WORK/processor-profile" -identity-default-url "$TP_IDENTITY_DEFAULT_URL" -trust-feature-dir "$ROOT/processor/testfeatures" -no-browser >"$WORK/processor-private.log" 2>&1 &
fi
tp_pid=$!
# Private URL is parsed and handed directly to the opener, never printed in shared output.
url=
for ((i=0;i<150;i++)); do
  kill -0 "$id_pid" 2>/dev/null || die "Identity exited; inspect private local log (do not share)."
  kill -0 "$tp_pid" 2>/dev/null || die "Processor exited; ports/profiles or startup failed; inspect private local log (do not share)."
  if [[ -s "$WORK/processor-private.log" ]]; then
    url="$(python3 - "$WORK/processor-private.log" "$TP_PORT" <<'PY'
import re,sys
s=open(sys.argv[1],encoding="utf-8",errors="replace").read()
m=re.search(r"(?m)^\s*open\s*:\s*(http://127\.0\.0\.1:"+re.escape(sys.argv[2])+r"/[^\s]+)",s)
print(m.group(1) if m else "")
PY
)"
  fi
  [[ -z "$url" ]] || break
  sleep 0.2
done
[[ -n "$url" ]] || die "Processor did not announce its private ready link; inspect local private log."
python3 - "$url" <<'PY'
import sys,urllib.request,urllib.error
try:
    with urllib.request.urlopen(sys.argv[1],timeout=4) as r:
        assert r.status == 200, r.status
except Exception as exc:
    sys.exit("ERROR: Processor did not answer readiness probe (%s); do not assume it is ready" % type(exc).__name__)
PY
printf 'P1 Processor ready on 127.0.0.1:%s. Identity trial port %s.\n' "$TP_PORT" "$ID_PORT"
# Never emit a private session URL into redirected/shared (noninteractive) output.
if [[ -t 0 && -t 1 ]]; then
  printf 'PRIVATE P1 Processor link (DO NOT SHARE; trusted local terminal only):\n%s\n' "$url"
else
  printf 'Private link withheld from noninteractive output; use the documented local-only log fallback in your own terminal.\n'
fi
opened=0
if grep -qiE 'microsoft|wsl' /proc/version /proc/sys/kernel/osrelease 2>/dev/null; then
  # Independent, optional WSL openers; never require cmd.exe or a new dependency.
  # The private fragment can briefly be visible in local process arguments.
  if command -v wslview >/dev/null 2>&1; then
    wslview "$url" >/dev/null 2>&1 && opened=1 || :
  fi
  if (( ! opened )) && command -v explorer.exe >/dev/null 2>&1; then
    explorer.exe "$url" >/dev/null 2>&1 && opened=1 || :
  fi
  if (( ! opened )) && command -v powershell.exe >/dev/null 2>&1; then
    powershell.exe -NoProfile -NonInteractive -Command 'Start-Process -FilePath $args[0]' "$url" >/dev/null 2>&1 && opened=1 || :
  fi
elif [[ -n "${DISPLAY:-}${WAYLAND_DISPLAY:-}" ]] && command -v xdg-open >/dev/null 2>&1; then
  # Preserve native Linux behavior; do not print the URL to noninteractive output.
  xdg-open "$url" >/dev/null 2>&1 && opened=1 || :
fi
if ((opened)); then
  printf 'Browser opener returned success; confirm the private P1 Processor page actually appears.\n'
else
  printf 'No browser opener succeeded. Use the private URL displayed above (interactive terminal) or the local-only documented log fallback. DO NOT SHARE.\n'
fi
if [[ -t 0 && -t 1 ]]; then
  printf 'Connect Identity manually in Processor; request a fresh challenge when prompted. It will appear here (DO NOT SHARE). If needed, inspect %s/identity-private.log locally. No automatic pairing or approval.\n' "$WORK"
else
  printf 'Identity pairing codes are withheld from noninteractive output. Use the private Identity screen or your own trusted terminal; never share the private log.\n'
fi
printf 'Press Ctrl+C here to stop ONLY the two child processes this launcher started. Keep profiles to resume; remove them only to reset the trial.\n'
wait "$tp_pid" || die "Processor exited unsuccessfully; inspect private local log."
tp_pid=
