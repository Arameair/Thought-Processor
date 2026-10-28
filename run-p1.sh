#!/usr/bin/env bash
# P1 cleanup launcher. No authentication or approval is automated.
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
id_pid= tp_pid= lock=
INJECT="${P1_TEST_INJECT:-}"
die() { printf 'ERROR: %s\n' "$*" >&2; exit 1; }
need() { command -v "$1" >/dev/null 2>&1 || die "Missing $1 (install it and retry)."; }
stop() {
  trap - EXIT INT TERM
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
lock="$WORK/.launcher-running"
mkdir "$lock" 2>/dev/null || die "Launcher lock exists at $lock (another run or stale lock). Check existing processes before removing it."
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
opened=0
if grep -qi microsoft /proc/version 2>/dev/null && command -v cmd.exe >/dev/null 2>&1; then
  # Avoid passing private fragment to cmd.exe shell interpretation; PowerShell Start-Process receives argv.
  if command -v powershell.exe >/dev/null 2>&1; then
    powershell.exe -NoProfile -NonInteractive -Command 'Start-Process -FilePath $args[0]' "$url" >/dev/null 2>&1 && opened=1 || :
  fi
elif command -v xdg-open >/dev/null 2>&1; then
  xdg-open "$url" >/dev/null 2>&1 && opened=1 || :
fi
if ((opened)); then
  printf 'Browser opener launched; confirm the private P1 Processor page appears.\n'
else
  printf 'Browser did not open automatically. Open the private Processor link from %s/processor-private.log locally (open : line); never share it.\n' "$WORK"
fi
printf 'Connect Identity manually in Processor, then read the pairing code in %s/identity-private.log locally. Approvals remain human-only.\n' "$WORK"
printf 'Press Ctrl+C here to stop ONLY the two child processes this launcher started. Keep profiles to resume; remove them only to reset the trial.\n'
wait "$tp_pid" || die "Processor exited unsuccessfully; inspect private local log."
tp_pid=
