# Build Thought Processor (P0) on Windows from this checkout.
# Usage (PowerShell, from the "processor" folder):  .\scripts\build-windows.ps1
$ErrorActionPreference = "Stop"
go version
go vet ./...
if ($LASTEXITCODE -ne 0) { Write-Host "go vet FAILED - stopping"; exit 1 }
go test ./...
if ($LASTEXITCODE -ne 0) { Write-Host "go test FAILED - stopping"; exit 1 }
$branch = (git rev-parse --abbrev-ref HEAD).Trim()
go build -ldflags "-X main.buildBranch=$branch" -o thought-processor.exe .
if ($LASTEXITCODE -ne 0) { exit 1 }
Write-Host ""
Write-Host "Built thought-processor.exe from $branch @ $(git rev-parse HEAD)"
Write-Host "Start it with:  .\thought-processor.exe"
