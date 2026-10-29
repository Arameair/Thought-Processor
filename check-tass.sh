#!/usr/bin/env bash
# check-tass.sh — finds what stops the Developer Platform + TASS from running
# on this computer, and prints the exact fix for each problem.
#
#   curl -fsSL https://raw.githubusercontent.com/Arameair/Thought-Processor/experiment/tass-companion-claude-20261029/check-tass.sh | bash
#
# It only READS: it starts, installs, clones and changes nothing. It never
# prints secrets, pairing codes or private links. Safe to paste its output.
set -uo pipefail
P1C="${P1C:-$HOME/p1c}"
TP="$P1C/tp-tass-clone"
TS="${P1_TASS_SRC:-$P1C/tass-companion}"
ID="$P1C/identity"
TP_BRANCH=experiment/tass-companion-claude-20261029
TS_BRANCH=experiment/tass-companion-feature-claude-20261029
ID_SHA=64940b629a5ef52eaa143b2402726ec81d382002
MODEL="${TASS_MODEL:-qwen3.5:4b}"
problems=0
ok()   { printf '  OK    %s\n' "$*"; }
bad()  { problems=$((problems+1)); printf '  FIX   %s\n' "$1"; shift; for l in "$@"; do printf '        %s\n' "$l"; done; }
note() { printf '  NOTE  %s\n' "$*"; }
ver_ge() { [ "$(printf '%s\n%s\n' "$2" "$1" | sort -V | head -1)" = "$2" ]; }

printf '\nTASS + Developer Platform check — %s\n\n' "$(date '+%Y-%m-%d %H:%M')"

printf '1. This computer\n'
if grep -qi microsoft /proc/version 2>/dev/null; then ok "WSL ($(. /etc/os-release 2>/dev/null; echo "${PRETTY_NAME:-Linux}"))"
elif [ "$(uname -s)" = Linux ]; then ok "Linux ($(. /etc/os-release 2>/dev/null; echo "${PRETTY_NAME:-Linux}"))"
else bad "This must run in WSL (Ubuntu) or Linux, not Windows PowerShell." "Open the Ubuntu (WSL) app and run this again there."; fi

printf '\n2. Tools\n'
if command -v git >/dev/null; then ok "git $(git --version | awk '{print $3}')"; else bad "git is missing" "sudo apt update && sudo apt install -y git"; fi
if command -v go >/dev/null; then gv="$(go env GOVERSION 2>/dev/null | sed 's/^go//')"
  if ver_ge "$gv" 1.22; then ok "go $gv"; else bad "go $gv is too old (need 1.22+)" "Install Go from https://go.dev/dl/ (linux-amd64), then open a new terminal"; fi
else bad "Go is missing (needed to build Identity and Thought Processor)" "Install Go 1.22+ from https://go.dev/dl/ (linux-amd64), then open a new terminal"; fi
if command -v python3 >/dev/null; then pv="$(python3 -c 'import sys;print("%d.%d"%sys.version_info[:2])')"
  if ver_ge "$pv" 3.8; then ok "python3 $pv"; else bad "python3 $pv is too old (need 3.8+)" "sudo apt install -y python3"; fi
else bad "python3 is missing (TASS runs on it)" "sudo apt update && sudo apt install -y python3"; fi
command -v curl >/dev/null && ok "curl" || bad "curl is missing" "sudo apt install -y curl"

printf '\n3. Code\n'
check_repo() { # dir branch label
  local d="$1" br="$2" label="$3"
  if [ ! -d "$d" ]; then return 1; fi
  local cur; cur="$(git -C "$d" branch --show-current 2>/dev/null)"
  if [ "$cur" != "$br" ]; then bad "$label is on branch '${cur:-?}', not $br" "git -C $d fetch && git -C $d checkout $br"; return 0; fi
  if [ -n "$(git -C "$d" status --porcelain 2>/dev/null)" ]; then bad "$label has local changes (the launcher refuses a changed checkout)" "git -C $d status   (then: git -C $d stash  — or ask Claude)"; fi
  if git -C "$d" fetch -q origin "$br" 2>/dev/null; then
    if [ "$(git -C "$d" rev-parse HEAD)" != "$(git -C "$d" rev-parse FETCH_HEAD)" ]; then bad "$label is not the newest version" "git -C $d pull"; else ok "$label up to date ($(git -C "$d" rev-parse --short HEAD))"; fi
  else note "$label: couldn't check GitHub for updates (offline or no access); local $(git -C "$d" rev-parse --short HEAD)"; fi
  return 0
}
if [ -d "$ID" ]; then
  [ "$(git -C "$ID" rev-parse HEAD 2>/dev/null)" = "$ID_SHA" ] && ok "Identity at $ID ($(git -C "$ID" rev-parse --short HEAD))" \
    || bad "Identity at $ID is not on the reviewed commit ${ID_SHA:0:7}" "git -C $ID fetch && git -C $ID checkout experiment/p1-cleanup-claude-20261026 && git -C $ID reset --hard $ID_SHA   (only if you have no own changes there)"
else bad "Identity is missing at $ID" "git clone --branch experiment/p1-cleanup-claude-20261026 https://github.com/Arameair/identity.git $ID"; fi
check_repo "$TP" "$TP_BRANCH" "Thought Processor ($TP)" || bad "Thought Processor is missing at $TP" "git clone --branch $TP_BRANCH https://github.com/Arameair/Thought-Processor.git $TP"
if ! check_repo "$TS" "$TS_BRANCH" "TASS ($TS)"; then
  if GIT_TERMINAL_PROMPT=0 git ls-remote https://github.com/Arameair/TASS.git >/dev/null 2>&1; then
    bad "TASS is missing at $TS" "git clone --branch $TS_BRANCH https://github.com/Arameair/TASS.git $TS"
  else
    bad "TASS is missing at $TS, and git here can't sign in to GitHub (TASS is a private repository)" \
        "sudo apt install -y gh && gh auth login        (choose GitHub.com, HTTPS, log in with a web browser)" \
        "gh auth setup-git" \
        "git clone --branch $TS_BRANCH https://github.com/Arameair/TASS.git $TS"
  fi
fi

printf '\n4. Local model (Ollama)\n'
if command -v ollama >/dev/null; then
  ov="$(ollama --version 2>/dev/null | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | tail -1)"
  if [ -n "$ov" ] && ver_ge "$ov" 0.17.4; then ok "ollama $ov"; else bad "ollama ${ov:-?} is too old for $MODEL (need 0.17.4+)" "curl -fsSL https://ollama.com/install.sh | sh"; fi
  if curl -s --noproxy '*' -m 3 http://127.0.0.1:11434/api/version >/dev/null; then
    ok "Ollama is running at 127.0.0.1:11434"
    if curl -s --noproxy '*' -m 5 http://127.0.0.1:11434/api/tags | grep -q "\"$MODEL\""; then ok "model $MODEL is downloaded"
      caps="$(curl -s --noproxy '*' -m 10 -X POST http://127.0.0.1:11434/api/show -d "{\"model\":\"$MODEL\"}" | python3 -c 'import sys,json
try: print(",".join(json.load(sys.stdin).get("capabilities") or []))
except Exception: print("")' 2>/dev/null)"
      [ -n "$caps" ] && ok "$MODEL can: $caps" || note "couldn't read what $MODEL can do"
      if ollama ps 2>/dev/null | grep -q "$MODEL"; then ok "$MODEL is loaded: $(ollama ps 2>/dev/null | awk -v m="$MODEL" '$1==m' | tr -s ' ')"
      else note "$MODEL isn't loaded yet (first answer will be slow). Warm it up: ollama run $MODEL \"Say hello in five words.\""; fi
    else bad "model $MODEL isn't downloaded" "ollama pull $MODEL"; fi
  else bad "Ollama is installed but not running" "Open a second WSL terminal and run:  ollama serve   (leave it open)"; fi
else bad "Ollama is missing" "curl -fsSL https://ollama.com/install.sh | sh" "then: ollama pull $MODEL"; fi

printf '\n5. Is the workbench already running?\n'
if curl -s --noproxy '*' -m 2 http://127.0.0.1:7867/api/health | grep -q thought-processor; then
  ok "Thought Processor is already running on 127.0.0.1:7867"
  note "Use the browser tab it opened. If you closed it, open the private link from the terminal where you started run-p1.sh"
  note "(or stop it with Ctrl+C in that terminal and start again). Don't start a second one."
  running=1
else ok "not running (ports 7797/7867 will be checked by the launcher)"; running=0; fi

printf '\n6. Launcher checks (builds and starts nothing)\n'
if [ -x "$(command -v bash)" ] && [ -f "$TP/run-p1.sh" ] && [ -d "$TS" ] && [ "$running" = 0 ]; then
  out="$(cd "$TP" && P1_TASS_SRC="$TS" P1_PREFLIGHT_ONLY=1 bash ./run-p1.sh 2>&1)"; rc=$?
  printf '%s\n' "$out" | grep -E '^(ERROR|TASS feature|Thought feature|Developer Utility|PREFLIGHT|Trial folder)' | sed 's/^/        /'
  if [ $rc -eq 0 ] && printf '%s' "$out" | grep -q 'TASS feature source'; then ok "launcher preflight OK, TASS is offered"
  elif [ $rc -eq 0 ]; then bad "the launcher would start WITHOUT TASS (it didn't find the TASS checkout)" "Start it with:  cd $TP && P1_TASS_SRC=$TS bash ./run-p1.sh"
  else bad "the launcher refused (see the ERROR line above)" "Copy that ERROR line to Claude"; fi
elif [ "$running" = 1 ]; then note "skipped: the workbench is already running"
else note "skipped: fix the items above first"; fi

printf '\n'
if [ "$problems" -eq 0 ]; then
  printf 'READY. Start the workbench with:\n\n  cd %s && P1_TASS_SRC=%s bash ./run-p1.sh\n\n' "$TP" "$TS"
  printf 'Then follow docs/P1/TASS.md, section E (everything happens in the TASS box on the Developer Platform).\n\n'
else
  printf '%d thing(s) to fix — do the FIX lines from the top, then run this check again.\n' "$problems"
  printf 'Stuck? Paste this whole output to Claude (it contains no secrets).\n\n'
fi
