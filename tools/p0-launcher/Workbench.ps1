# Windows 11 launcher front end for the isolated P0 WSL experiment.
# No private link, pairing code or token is printed or stored here.
# This only invokes the WSL supervisor's bounded commands.
$ErrorActionPreference = 'Stop'
$Script = 'exec python3 "$HOME/p1c/alpha/Thought-Processor/tools/p0-launcher/secure_launch.py" "$1"'
function Invoke-Workbench([string]$Action) {
    & wsl.exe -e bash -lc $Script -- $Action
    if ($LASTEXITCODE -ne 0) {
        Write-Host "Result: $Action did not finish successfully (exit $LASTEXITCODE)." -ForegroundColor Yellow
    }
}
if (-not (Get-Command wsl.exe -ErrorAction SilentlyContinue)) {
    Write-Host "WSL is unavailable. No service was started." -ForegroundColor Red
    exit 2
}
while ($true) {
    Write-Host ""
    Write-Host "TASS Workbench (alpha, Windows / WSL)" -ForegroundColor Cyan
    Write-Host "Profile: ~/p1c/p1-tass-alpha | Identity 7798 | Processor 7868"
    Write-Host "1  Start         2  Stop          3  Open Development Workstation (Edge InPrivate)"
    Write-Host "4  Status        5  Versions      6  Check Updates (read-only)"
    Write-Host "7  Show private pairing challenge       8  Apply Update (disabled)"
    Write-Host "0  Exit menu (does not stop running services)"
    switch ((Read-Host "Choose").Trim()) {
        '1' { Invoke-Workbench 'start' }
        '2' { Invoke-Workbench 'stop' }
        '3' { Invoke-Workbench 'open' }
        '4' { Invoke-Workbench 'status' }
        '5' { Invoke-Workbench 'versions' }
        '6' { Invoke-Workbench 'check' }
        '7' { Invoke-Workbench 'pairing' }
        '8' { Write-Host "BLOCKED: requires reviewed compatible pins and Identity approval." -ForegroundColor Yellow }
        '0' { exit 0 }
        default { Write-Host "Unknown choice." -ForegroundColor Yellow }
    }
}