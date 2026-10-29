#!/usr/bin/env bash
# P1 cleanup launcher. No authentication or approval is automated.
# Do not trace session URLs or pairing challenges even if invoked with bash -x.
set +x
set -Eeuo pipefail
umask 077
BASE_BRANCH=experiment/p1-cleanup-claude-20261026
LAUNCH_BRANCH=experiment/developer-entrance-claude-20261029
# Developer Utility (the Developer entrance): exact reviewed package digest,
# pinned in Processor (-trust-feature-artifact) and Identity
# (-trust-fixture-artifact). Any other content is refused before anything starts.
DEV_ARTIFACT=9d33ed8bfccff317aeec1914718e32c3691845a5a2dac33b57d344487001b39d
# Thought (Spark → Trace → Thought) feature: the exact reviewed package digest.
# Processor (-trust-feature-artifact) and Identity (-trust-fixture-artifact)
# both pin it; any other content stays BLOCKED in both.
THOUGHT_BRANCH=experiment/thought-feature-p1-claude-20261029
THOUGHT_ARTIFACT=8f92ee685c99851128f322c9063ff151e28d7ccf48709315d8857fc56e8079ec
THOUGHT_BASE_SHA=772200b5bc29efce6da3b0e064fb6714570da1d4
IDENTITY_SHA=64940b629a5ef52eaa143b2402726ec81d382002
PROCESSOR_BASE_SHA=db31e4785d292925249a2f9f06e3c06bd78ba5b0
ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)"
ID_SRC="${P1_IDENTITY_SRC:-$(dirname "$ROOT")/identity}"
if [[ -z "${P1_THOUGHT_SRC:-}" && -d "$(dirname "$ROOT")/Thought" ]]; then P1_THOUGHT_SRC="$(dirname "$ROOT")/Thought"; fi
THOUGHT_SRC="${P1_THOUGHT_SRC:-$(dirname "$ROOT")/thought}"
# Developer entrance trial: its OWN work folder and ports. The owner's other
# trials (~/p1c/p1-launcher 7792/7862, ~/p1c/p1-worker-runtime 7794/7864)
# are never reused, overwritten or contacted.
WORK="${P1_LAUNCH_WORK:-$HOME/p1c/p1-developer-entrance}"
ID_PORT="${P1_IDENTITY_PORT:-7796}"
TP_PORT="${P1_PROCESSOR_PORT:-7866}"
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
# A checkout may be an ordinary clone (.git is a folder) or a Git worktree
# (.git is a file pointing at the main repository). Git itself decides; the
# checks below are the same for both and are not loosened for either.
# Exact artifact digest of a feature package (same algorithm as Processor and Identity).
package_digest() {
  python3 -I - "$1" <<'PY'
import hashlib, json, os, sys
d = sys.argv[1]
def h(p):
    if os.path.islink(p) or not os.path.isfile(p):
        sys.exit("not a regular file: " + os.path.basename(p))
    with open(p, "rb") as f:
        return hashlib.sha256(f.read()).hexdigest()
m = json.load(open(os.path.join(d, "feature.json"), "rb"))
files = sorted(m["files"], key=lambda f: f["path"])
b = "thought.feature-package/1\nmanifest %s\n" % h(os.path.join(d, "feature.json"))
b += "".join("%s  %s\n" % (h(os.path.join(d, f["path"])), f["path"]) for f in files)
print(hashlib.sha256(b.encode()).hexdigest())
PY
}
checkout_ok() { # dir expected-owner/repo label
  local dir="$1" slug="$2" label="$3" top url norm
  [[ -d "$dir" && ! -L "$dir" ]] || die "$label: expected a checkout folder at $dir (not a link). Set P1_IDENTITY_SRC / P1_THOUGHT_SRC if it lives elsewhere."
  top="$(git -C "$dir" rev-parse --show-toplevel 2>/dev/null)" || die "$label: $dir is not a working Git checkout (clone or worktree), or its worktree link is broken."
  [[ "$(cd "$dir" && pwd -P)" == "$(cd "$top" && pwd -P)" ]] || die "$label: $dir is inside another checkout ($top), not a checkout itself."
  [[ "$(git -C "$dir" rev-parse --is-bare-repository)" == false ]] || die "$label: $dir is a bare repository."
  url="$(git -C "$dir" remote get-url origin 2>/dev/null)" || die "$label: $dir has no 'origin' remote; can't confirm it is $slug."
  norm="$(printf '%s' "$url" | tr '[:upper:]' '[:lower:]')"
  norm="${norm%/}"; norm="${norm%.git}"
  case "$norm" in
    "https://github.com/${slug,,}"|"git@github.com:${slug,,}"|"ssh://git@github.com/${slug,,}") ;;
    *) die "$label: $dir is a checkout of a different repository (origin: $url); expected $slug." ;;
  esac
  [[ -z "$(git -C "$dir" status --porcelain --untracked-files=normal)" ]] || die "Dirty working tree in $dir. Commit/stash/review changes first; nothing modified."
}
checkout_ok "$ID_SRC" Arameair/identity Identity
checkout_ok "$ROOT" Arameair/Thought-Processor "Thought Processor"
[[ "$(git -C "$ID_SRC" branch --show-current)" == "$BASE_BRANCH" ]] || die "Identity must be checked out on $BASE_BRANCH (no checkout performed)."
[[ "$(git -C "$ROOT" branch --show-current)" == "$LAUNCH_BRANCH" ]] || die "Processor must be checked out on $LAUNCH_BRANCH (no checkout performed)."
id_sha="$(git -C "$ID_SRC" rev-parse HEAD)"
tp_sha="$(git -C "$ROOT" rev-parse HEAD)"
[[ "$id_sha" == "$IDENTITY_SHA" ]] || die "Identity changed: expected $IDENTITY_SHA, found $id_sha. Review before proceeding."
git -C "$ROOT" merge-base --is-ancestor "$PROCESSOR_BASE_SHA" HEAD 2>/dev/null || die "Processor is not descended from the reviewed P1 cleanup commit."
printf 'Identity P1 source: %s (%s)\nProcessor P1 launcher: %s (base %s) (%s)\n' "$id_sha" "$ID_SRC" "$tp_sha" "$PROCESSOR_BASE_SHA" "$ROOT"
ID_EXTRA=() TP_EXTRA=()
DEV_PKG="$ROOT/processor/developer-utility"
[[ -f "$DEV_PKG/feature.json" && ! -L "$DEV_PKG" ]] || die "No Developer Utility package at $DEV_PKG."
dev_digest="$(package_digest "$DEV_PKG")" || die "Could not read the Developer Utility package at $DEV_PKG."
[[ "$dev_digest" == "$DEV_ARTIFACT" ]] || die "Developer Utility package content ($dev_digest) is not the reviewed pinned artifact ($DEV_ARTIFACT). Nothing started."
printf 'Developer Utility package: matches pinned artifact %s…\n' "${DEV_ARTIFACT:0:12}"
if [[ "${P1_THOUGHT:-on}" != off && -e "$THOUGHT_SRC" ]]; then
  # A folder is there, so it must be the reviewed Thought checkout; never skip it silently.
  checkout_ok "$THOUGHT_SRC" Arameair/Thought Thought
  [[ "$(git -C "$THOUGHT_SRC" branch --show-current)" == "$THOUGHT_BRANCH" ]] || die "Thought must be checked out on $THOUGHT_BRANCH (no checkout performed), or run with P1_THOUGHT=off."
  git -C "$THOUGHT_SRC" merge-base --is-ancestor "$THOUGHT_BASE_SHA" HEAD 2>/dev/null || die "Thought is not descended from the reviewed commit $THOUGHT_BASE_SHA."
  [[ -f "$THOUGHT_SRC/feature/feature.json" && ! -L "$THOUGHT_SRC/feature" ]] || die "No Thought feature package at $THOUGHT_SRC/feature."
  # Integrity before anything is built or started: the package must be exactly the pinned artifact.
  thought_digest="$(package_digest "$THOUGHT_SRC/feature")" || die "Could not read the Thought package at $THOUGHT_SRC/feature."
  [[ "$thought_digest" == "$THOUGHT_ARTIFACT" ]] || die "Thought package content ($thought_digest) is not the reviewed pinned artifact ($THOUGHT_ARTIFACT). Nothing started."
  TP_EXTRA+=(-trust-feature-dir "$THOUGHT_SRC/feature" -trust-feature-artifact "$THOUGHT_ARTIFACT")
  ID_EXTRA+=(-trust-fixture-artifact "$THOUGHT_ARTIFACT")
  printf 'Thought feature source: %s (%s) — package matches pinned artifact %s…\n' "$(git -C "$THOUGHT_SRC" rev-parse HEAD)" "$THOUGHT_SRC" "${THOUGHT_ARTIFACT:0:12}"
else
  printf 'Thought feature: not offered (no checkout at %s, or P1_THOUGHT=off).\n' "$THOUGHT_SRC"
fi
TP_EXTRA+=(-trust-feature-dir "$DEV_PKG" -trust-feature-artifact "$DEV_ARTIFACT")
ID_EXTRA+=(-trust-fixture-artifact "$DEV_ARTIFACT")
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
if [[ "${P1_PREFLIGHT_ONLY:-}" == 1 ]]; then
  # Checks only: nothing is built, started, created or changed.
  if [[ -d "$WORK/processor-profile" || -d "$WORK/identity-profile" ]]; then prof="existing profiles will be reused"; else prof="no profiles yet (fresh trial)"; fi
  printf 'Trial folder: %s (%s). Ports %s/%s are free.
PREFLIGHT OK — nothing was built or started.
' "$WORK" "$prof" "$ID_PORT" "$TP_PORT"
  exit 0
fi
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
(cd "$ROOT/processor" && go build -ldflags "-X main.buildBranch=$LAUNCH_BRANCH" -o "$WORK/bin/processor-p1" .) || die "Processor Go build failed"
printf 'Starting private P1 trial. Logs stay local in %s (DO NOT SHARE).\n' "$WORK"
# Locator isolation ensures we do not discover unrelated Identity instances.
export IDENTITY_LOCATOR_DIR="$WORK/locators"
export TP_IDENTITY_DEFAULT_URL="http://127.0.0.1:$ID_PORT"
mkdir -p "$IDENTITY_LOCATOR_DIR"
if [[ "$INJECT" == identity-start ]]; then
  (exit 42) >"$WORK/identity-private.log" 2>&1 &
else
  "$WORK/bin/identity-p1" -port "$ID_PORT" -data "$WORK/identity-profile" ${ID_EXTRA[@]+"${ID_EXTRA[@]}"} >"$WORK/identity-private.log" 2>&1 &
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
                      "\nType this code into Thought Processor. No approval is automated.\n",
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
  "$WORK/bin/processor-p1" -port "$TP_PORT" -data "$WORK/processor-profile" -identity-default-url "$TP_IDENTITY_DEFAULT_URL" -trust-feature-dir "$ROOT/processor/testfeatures" ${TP_EXTRA[@]+"${TP_EXTRA[@]}"} -no-browser >"$WORK/processor-private.log" 2>&1 &
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
