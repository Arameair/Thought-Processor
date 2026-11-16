@echo off
setlocal
REM Owner-visible launcher. Uses Windows PowerShell without disabling policy.
REM No private URLs, codes, credentials or command arguments are recorded.
powershell.exe -NoProfile -File "%~dp0Workbench.ps1"
if errorlevel 1 (
  echo Workbench could not start. Nothing has been approved or updated.
  echo Check WSL and PowerShell policy with your administrator.
  pause
)
endlocal
