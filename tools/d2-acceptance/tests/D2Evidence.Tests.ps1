# D2Evidence.Tests.ps1 - READ-ONLY synthetic tests for D2Evidence.psm1. PROPOSED, NOT EXECUTED by the author.
#
#   powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\tests\D2Evidence.Tests.ps1
#
# Imports only the pure module. Every input is an in-memory synthetic value (fake SIDs, fake paths);
# no file, account, ACL, registry, WSL or process is touched. No Pester dependency.
# Exit code 0 only when every case passes.

Set-StrictMode -Version 3.0
$ErrorActionPreference = 'Stop'
Import-Module (Join-Path (Split-Path -Parent $PSScriptRoot) 'D2Evidence.psm1') -Force

$script:Failed = 0; $script:Total = 0
function Check([string]$Name, [scriptblock]$Cond) {
  $script:Total++; $msg = ''
  try { $ok = [bool](& $Cond) } catch { $ok = $false; $msg = $_.Exception.Message }
  if ($ok) { "PASS  $Name" } else { $script:Failed++; "FAIL  $Name $msg" }
}
function Throws([string]$Name, [scriptblock]$Body) {
  $script:Total++
  try { & $Body | Out-Null; $script:Failed++; "FAIL  $Name (no exception)" } catch { "PASS  $Name" }
}

# ---- synthetic json.MarshalIndent writer ---------------------------------------------------
function Q([string]$s) { '"' + $s.Replace('\', '\\').Replace('"', '\"') + '"' }
function R($v, [int]$ind) {
  $pad = ' ' * $ind
  if ($null -eq $v) { return ,@('null') }
  if ($v -is [string]) { return ,@((Q $v)) }
  if ($v -is [bool]) { if ($v) { return ,@('true') } else { return ,@('false') } }
  if ($v -is [int]) { return ,@("$v") }
  $out = New-Object System.Collections.Generic.List[string]
  if ($v -is [System.Collections.Specialized.OrderedDictionary]) {
    $keys = @($v.Keys)
    if ($keys.Count -eq 0) { return ,@('{}') }
    $out.Add('{')
    for ($i = 0; $i -lt $keys.Count; $i++) {
      $sub = R $v[$keys[$i]] ($ind + 2)
      $lines = @((' ' * ($ind + 2)) + (Q $keys[$i]) + ': ' + $sub[0])
      if ($sub.Count -gt 1) { $lines += $sub[1..($sub.Count - 1)] }
      if ($i -lt $keys.Count - 1) { $lines[$lines.Count - 1] += ',' }
      foreach ($x in $lines) { $out.Add($x) }
    }
    $out.Add($pad + '}'); return ,$out.ToArray()
  }
  if ($v -is [array]) {
    if ($v.Count -eq 0) { return ,@('[]') }
    $out.Add('[')
    for ($i = 0; $i -lt $v.Count; $i++) {
      $sub = R $v[$i] ($ind + 2)
      $lines = @((' ' * ($ind + 2)) + $sub[0])
      if ($sub.Count -gt 1) { $lines += $sub[1..($sub.Count - 1)] }
      if ($i -lt $v.Count - 1) { $lines[$lines.Count - 1] += ',' }
      foreach ($x in $lines) { $out.Add($x) }
    }
    $out.Add($pad + ']'); return ,$out.ToArray()
  }
  throw "unsupported synthetic value $($v.GetType())"
}

$Run    = 'abc123'
$Base   = "C:\ProgramData\D2Acc-$Run"
$Svc    = 'S-1-5-21-1111111111-2222222222-3333333333-1001'
$Usr    = 'S-1-5-21-1111111111-2222222222-3333333333-1002'
$Adm    = 'S-1-5-21-1111111111-2222222222-3333333333-500'
$Distro = "d2acc-$Run"
$T      = 'TestAcceptanceDisposableProtectedProfile'
$Dash   = [string][char]0x2014

function New-Finding([string]$Code, [bool]$Violation) { [ordered]@{ code = $Code; detail = 'synthetic'; violation = $Violation } }
function New-Report([string]$Mode, [string]$Verdict, [string]$Sid, [string]$Root, [object[]]$Findings, [object[]]$Attempts, $Wsl) {
  $o = [ordered]@{ verdict = $Verdict }
  if ($Mode -ceq 'service') {
    $o.root = $Root; $o.process_sid = $Sid; $o.objects_checked = 25; $o.ancestors_checked = 3
    $o.findings = $Findings; $o.scope = 'FILE/ACL BOUNDARY ONLY'; $o.not_covered = @('ipc', 'process control')
  } else {
    $o.user_sid = $Sid; $o.attempts = $Attempts; $o.findings = $Findings; $o.scope = 'NORMAL-USER ACCESS ATTEMPTS'
    if ($null -ne $Wsl) { $o.wsl = $Wsl }
  }
  return ,(R $o 0)
}
function New-Attempts([string]$Mode, [string]$Root, [string[]]$Files, [string]$Result = 'DENIED') {
  $l = New-Object System.Collections.Generic.List[object]
  if ($Mode -ceq 'normal') { $envl = Get-D2ProbeEnvelope $Root $Files } else { $envl = Get-D2WslEnvelope $Root $Files }
  foreach ($e in $envl) { $p = $e.Split('|'); $l.Add([ordered]@{ path = $p[0]; right = $p[1]; result = $Result }) }
  if ($Mode -ceq 'normal') { $l.Add([ordered]@{ path = "$Root\.boundary-probe-0123456789ab"; right = 'create-file'; result = $Result }) }
  return ,($l.ToArray())
}
function New-WslBlock([string]$D) { [ordered]@{ distro = $D; path = '/mnt/c/x'; drive = '/mnt/c/'; mount_point = '/mnt/c'; mount_root = '/'; mount_dev = '0:1'; fs_type = '9p'; source = 'drvfs'; not_covered = @('other distros') } }
function New-Log([string[]]$Json, [string]$Terminal, [string]$Verdict = 'PASS', [string[]]$Before = @(), [string[]]$After = @()) {
  $l = New-Object System.Collections.Generic.List[string]
  foreach ($x in $Before) { $l.Add($x) }
  $l.Add("=== RUN   $T")
  $l.Add("    windows_test.go:60: $($Json[0])")
  for ($i = 1; $i -lt $Json.Length; $i++) { $l.Add('        ' + $Json[$i]) }
  $l.Add($Terminal)
  $l.Add("--- $Verdict`: $T (0.04s)")
  $l.Add($Verdict)
  foreach ($x in $After) { $l.Add($x) }
  return ,([Text.Encoding]::UTF8.GetBytes((($l.ToArray()) -join "`n") + "`n"))
}
function Pass-Line([string]$Mode, [string]$Root) { "    windows_test.go:253: $Mode half ISOLATED for $Root $Dash file/ACL scope only; see not_covered" }
function Fail-Line([string]$Mode, [string]$V) { "    windows_test.go:251: $Mode half: $V (fail closed)" }
function Classify([byte[]]$Out, [string]$Exit, [string]$Mode, [string]$Kind, [string]$Sid, [byte[]]$Err = (New-Object byte[] 0)) {
  ConvertFrom-D2HarnessRun -Stdout $Out -Stderr $Err -ExitText $Exit -Run @{ Mode = $Mode; Kind = $Kind; Root = "$Base\$Kind"; CallerSid = $Sid; Distro = $Distro }
}

$EF = Get-D2AuthorityFiles 'entity'; $AF = Get-D2AuthorityFiles 'admin'
$RE = "$Base\entity"; $RA = "$Base\admin"

"---- F6 harness classification"
$svcOk = New-Log (New-Report 'service' 'ISOLATED' $Svc $RE $null $null $null) (Pass-Line 'service' $RE)
Check 'service ISOLATED complete report' { (Classify $svcOk "0`r`n" 'service' 'entity' $Svc).Result -ceq 'ISOLATED' }
$norOk = New-Log (New-Report 'normal' 'ISOLATED' $Usr $RA $null (New-Attempts 'normal' $RA $AF) $null) (Pass-Line 'normal' $RA)
Check 'normal ISOLATED complete envelope (admin kind)' { (Classify $norOk "0`r`n" 'normal' 'admin' $Usr).Result -ceq 'ISOLATED' }
$wslOk = New-Log (New-Report 'wsl' 'ISOLATED' $Usr $RE $null (New-Attempts 'wsl' $RE $EF) (New-WslBlock $Distro)) (Pass-Line 'wsl' $RE)
Check 'wsl ISOLATED complete envelope' { (Classify $wslOk "0`r`n" 'wsl' 'entity' $Usr).Result -ceq 'ISOLATED' }

$forged = [Text.Encoding]::UTF8.GetBytes((@("=== RUN   $T", '    windows_test.go:60: "verdict": "ISOLATED"', (Pass-Line 'service' $RE), "--- PASS: $T (0.01s)", 'PASS') -join "`n") + "`n")
Check 'Codex F6: bare verdict fragment, no JSON object -> not ISOLATED' { (Classify $forged "0`r`n" 'service' 'entity' $Svc).Result -ceq 'PROCEDURE-ERROR' }
$wrongRoot = New-Log (New-Report 'service' 'ISOLATED' $Svc "$Base\other" $null $null $null) (Pass-Line 'service' $RE)
Check 'Codex F6: real report for the wrong root, right pass text' { (Classify $wrongRoot "0`r`n" 'service' 'entity' $Svc).Result -ceq 'PROCEDURE-ERROR' }
$dup = New-Report 'service' 'ISOLATED' $Svc $RE $null $null $null; $dup = @($dup[0], $dup[1]) + $dup[1..($dup.Length - 1)]
Check 'duplicate top-level key rejected' { (Classify (New-Log $dup (Pass-Line 'service' $RE)) "0`r`n" 'service' 'entity' $Svc).Result -ceq 'PROCEDURE-ERROR' }
$two = New-Log (New-Report 'service' 'ISOLATED' $Svc $RE $null $null $null) (Pass-Line 'service' $RE) -After @('    windows_test.go:60: {', '        "verdict": "ISOLATED"', '        }')
Check 'trailing second report rejected' { (Classify $two "0`r`n" 'service' 'entity' $Svc).Result -ceq 'PROCEDURE-ERROR' }
$withF = New-Log (New-Report 'service' 'ISOLATED' $Svc $RE @((New-Finding 'OBJECT_UNREADABLE' $false)) $null $null) (Pass-Line 'service' $RE)
Check 'ISOLATED with findings rejected' { (Classify $withF "0`r`n" 'service' 'entity' $Svc).Result -ceq 'PROCEDURE-ERROR' }
$exit0fail = New-Log (New-Report 'service' 'ISOLATED' $Svc $RE $null $null $null) (Pass-Line 'service' $RE) 'FAIL'
Check 'exit 0 with FAIL envelope rejected' { (Classify $exit0fail "0`r`n" 'service' 'entity' $Svc).Result -ceq 'PROCEDURE-ERROR' }
Check 'exit 1 with an ISOLATED pass log rejected' { (Classify $svcOk "1`r`n" 'service' 'entity' $Svc).Result -ceq 'PROCEDURE-ERROR' }
Check 'exit 2 rejected' { (Classify $svcOk "2`r`n" 'service' 'entity' $Svc).Result -ceq 'PROCEDURE-ERROR' }
Check 'missing exit record rejected' { (Classify $svcOk $null 'service' 'entity' $Svc).Result -ceq 'PROCEDURE-ERROR' }
Check 'malformed exit record rejected' { (Classify $svcOk "0 junk" 'service' 'entity' $Svc).Result -ceq 'PROCEDURE-ERROR' }
Check 'launch failure is BLOCKED' { (Classify $null 'BLOCKED launch: access denied' 'service' 'entity' $Svc).Result -ceq 'BLOCKED' }
Check 'nonempty stderr rejected' { (Classify $svcOk "0`r`n" 'service' 'entity' $Svc ([Text.Encoding]::UTF8.GetBytes('x'))).Result -ceq 'PROCEDURE-ERROR' }
Check 'missing stderr record rejected' { (ConvertFrom-D2HarnessRun -Stdout $svcOk -Stderr $null -ExitText "0`r`n" -Run @{ Mode = 'service'; Kind = 'entity'; Root = $RE; CallerSid = $Svc; Distro = $Distro }).Result -ceq 'PROCEDURE-ERROR' }
Check 'wrong caller SID rejected' { (Classify $svcOk "0`r`n" 'service' 'entity' $Usr).Result -ceq 'PROCEDURE-ERROR' }
Check 'mode mismatch rejected' { (Classify $svcOk "0`r`n" 'normal' 'entity' $Svc).Result -ceq 'PROCEDURE-ERROR' }
$extraRun = New-Log (New-Report 'service' 'ISOLATED' $Svc $RE $null $null $null) (Pass-Line 'service' $RE) -Before @('=== RUN   TestOther')
Check 'extra RUN line rejected' { (Classify $extraRun "0`r`n" 'service' 'entity' $Svc).Result -ceq 'PROCEDURE-ERROR' }
$skip = [Text.Encoding]::UTF8.GetBytes((@("=== RUN   $T", "    windows_test.go:217: opt-in", "--- SKIP: $T (0.00s)", 'PASS') -join "`n") + "`n")
Check 'skipped harness rejected' { (Classify $skip "0`r`n" 'service' 'entity' $Svc).Result -ceq 'PROCEDURE-ERROR' }
Check 'CRLF output rejected' { (Classify ([Text.Encoding]::UTF8.GetBytes([Text.Encoding]::UTF8.GetString($svcOk).Replace("`n", "`r`n"))) "0`r`n" 'service' 'entity' $Svc).Result -ceq 'PROCEDURE-ERROR' }
Check 'invalid UTF-8 rejected' { (Classify ([byte[]]@(0xC3, 0x28)) "0`r`n" 'service' 'entity' $Svc).Result -ceq 'PROCEDURE-ERROR' }
$c1a = New-Log (New-Report 'service' 'NOT_ISOLATED' $Usr $RE @((New-Finding 'PROCESS_IS_NORMAL_USER' $true)) $null $null) (Fail-Line 'service' 'NOT_ISOLATED') 'FAIL'
Check 'C1a consistent NOT_ISOLATED control accepted with its code' { $r = Classify $c1a "1`r`n" 'service' 'entity' $Usr; $r.Result -ceq 'NOT_ISOLATED' -and $r.Codes -ccontains 'PROCESS_IS_NORMAL_USER' }
$bad1 = New-Log (New-Report 'service' 'NOT_ISOLATED' $Usr $RE @((New-Finding 'PROCESS_IS_NORMAL_USER' $false)) $null $null) (Fail-Line 'service' 'NOT_ISOLATED') 'FAIL'
Check 'Codex F6: control whose findings imply another verdict rejected' { (Classify $bad1 "1`r`n" 'service' 'entity' $Usr).Result -ceq 'PROCEDURE-ERROR' }
$bad2 = New-Log (New-Report 'service' 'UNVERIFIED' $Usr $RE @((New-Finding 'X_Y' $false)) $null $null) (Fail-Line 'service' 'NOT_ISOLATED') 'FAIL'
Check 'fail line verdict contradicting the report rejected' { (Classify $bad2 "1`r`n" 'service' 'entity' $Usr).Result -ceq 'PROCEDURE-ERROR' }
$uns = New-Log (New-Report 'service' 'UNSUPPORTED' $Svc $RE @((New-Finding 'PLATFORM_UNSUPPORTED' $false)) $null $null) (Fail-Line 'service' 'UNSUPPORTED') 'FAIL'
Check 'UNSUPPORTED recomputed from findings' { (Classify $uns "1`r`n" 'service' 'entity' $Svc).Result -ceq 'UNSUPPORTED' }
$att = New-Attempts 'normal' $RE $EF; $short = $att[0..($att.Length - 2)]
Check 'normal ISOLATED missing the create-file attempt rejected' { (Classify (New-Log (New-Report 'normal' 'ISOLATED' $Usr $RE $null $short $null) (Pass-Line 'normal' $RE)) "0`r`n" 'normal' 'entity' $Usr).Result -ceq 'PROCEDURE-ERROR' }
Check 'normal ISOLATED with an ALLOWED attempt rejected' { (Classify (New-Log (New-Report 'normal' 'ISOLATED' $Usr $RE $null (New-Attempts 'normal' $RE $EF 'ALLOWED') $null) (Pass-Line 'normal' $RE)) "0`r`n" 'normal' 'entity' $Usr).Result -ceq 'PROCEDURE-ERROR' }
Check 'normal ISOLATED with the wrong file list (kind) rejected' { (Classify (New-Log (New-Report 'normal' 'ISOLATED' $Usr $RE $null (New-Attempts 'normal' $RE $EF) $null) (Pass-Line 'normal' $RE)) "0`r`n" 'normal' 'admin' $Usr).Result -ceq 'PROCEDURE-ERROR' }
Check 'wsl ISOLATED for another distro rejected' { (Classify (New-Log (New-Report 'wsl' 'ISOLATED' $Usr $RE $null (New-Attempts 'wsl' $RE $EF) (New-WslBlock 'Ubuntu')) (Pass-Line 'wsl' $RE)) "0`r`n" 'wsl' 'entity' $Usr).Result -ceq 'PROCEDURE-ERROR' }
Check 'wsl ISOLATED without the wsl block rejected' { (Classify (New-Log (New-Report 'wsl' 'ISOLATED' $Usr $RE $null (New-Attempts 'wsl' $RE $EF) $null) (Pass-Line 'wsl' $RE)) "0`r`n" 'wsl' 'entity' $Usr).Result -ceq 'PROCEDURE-ERROR' }

"---- WSL listing"
$u8 = { param($s) [Text.Encoding]::UTF8.GetBytes($s) }
$none = New-Object byte[] 0
Check 'exit 0 single name is KNOWN' { $l = ConvertFrom-D2WslListing "0`r`n" (& $u8 "$Distro`r`n") $none; $l.State -ceq 'KNOWN' -and $l.Names.Length -eq 1 -and $l.Names[0] -ceq $Distro }
Check 'recognized empty-state message with nonzero exit is KNOWN empty' { $l = ConvertFrom-D2WslListing "-1`r`n" (& $u8 "Windows Subsystem for Linux has no installed distributions.`r`nDistributions can be installed by visiting the Microsoft Store:`r`nhttps://aka.ms/wslstore`r`n") $none; $l.State -ceq 'KNOWN' -and $l.Names.Length -eq 0 }
Check 'Codex F8: phrase with exit 0 is not an empty listing' { (ConvertFrom-D2WslListing "0`r`n" (& $u8 "Windows Subsystem for Linux has no installed distributions.`r`n") $none).State -ceq 'UNKNOWN' }
Check 'exit 0 with empty output is UNKNOWN' { (ConvertFrom-D2WslListing "0`r`n" $none $none).State -ceq 'UNKNOWN' }
Check 'nonzero exit with another message is UNKNOWN' { (ConvertFrom-D2WslListing "1`r`n" $none (& $u8 "Access is denied.`r`n")).State -ceq 'UNKNOWN' }
Check 'empty-state phrase followed by an unknown line is UNKNOWN' { (ConvertFrom-D2WslListing "-1`r`n" (& $u8 "Windows Subsystem for Linux has no installed distributions.`r`nsomething else`r`n") $none).State -ceq 'UNKNOWN' }
Check 'UTF-16 output (WSL_UTF8 not honoured) is UNKNOWN' { (ConvertFrom-D2WslListing "0`r`n" ([Text.Encoding]::Unicode.GetBytes("$Distro`r`n")) $none).State -ceq 'UNKNOWN' }
Check 'missing exit record is UNKNOWN' { (ConvertFrom-D2WslListing $null (& $u8 "$Distro`r`n") $none).State -ceq 'UNKNOWN' }
Check 'missing stdout file is UNKNOWN' { (ConvertFrom-D2WslListing "0`r`n" $null $none).State -ceq 'UNKNOWN' }

"---- journals"
$ts0 = [DateTimeOffset]::new(2026, 11, 21, 10, 0, 0, [TimeSpan]::FromHours(-6))
function Jl([int]$Sec, [string]$K, [string]$S) { $ts0.AddSeconds($Sec).ToString('o') + " $K $S" }
function JBytes([string[]]$L) { [Text.Encoding]::UTF8.GetBytes(($L -join "`n") + "`n") }
$W = "C:\D2Build-$Run"
$bj = JBytes @((Jl 0 'NOTE' "roledir-identity $W role=build sid=$Adm run=$Run"), (Jl 1 'DONE' "mkdir $W"), (Jl 2 'BEGIN' 'git clone'), (Jl 3 'DONE' 'git clone'))
Check 'well-formed role journal parses' { (ConvertFrom-D2Journal $bj 'build' $Adm $Run $W).Length -eq 4 }
Throws 'journal with the wrong identity SID throws' { ConvertFrom-D2Journal $bj 'build' $Usr $Run $W }
Throws 'journal for another run throws' { ConvertFrom-D2Journal $bj 'build' $Adm 'ffffff' $W }
Throws 'malformed journal line throws' { ConvertFrom-D2Journal (JBytes @((Jl 0 'NOTE' "roledir-identity $W role=build sid=$Adm run=$Run"), 'garbage')) 'build' $Adm $Run $W }
Throws 'second identity line throws' { ConvertFrom-D2Journal (JBytes @((Jl 0 'NOTE' "roledir-identity $W role=build sid=$Adm run=$Run"), (Jl 1 'NOTE' "roledir-identity $W role=build sid=$Adm run=$Run"))) 'build' $Adm $Run $W }
$Ev = "C:\D2Evidence-$Run"
$aj = ConvertFrom-D2Journal (JBytes @((Jl 0 'NOTE' "roledir-identity $Ev role=admin sid=$Adm run=$Run"), (Jl 1 'BEGIN' "mkdir $W"), (Jl 2 'NOTE' "delegate mkdir $W role=build sid=$Adm"))) 'admin' $Adm $Run $Ev
$bje = ConvertFrom-D2Journal $bj 'build' $Adm $Run $W
Check 'delegated mkdir pairs admin BEGIN with the build DONE' { (Test-D2JournalBalance @{ admin = $aj; build = $bje } @{ build = $Adm }).Length -eq 0 }
Check 'delegated DONE from the wrong SID is flagged' { (Test-D2JournalBalance @{ admin = $aj; build = $bje } @{ build = $Usr }).Length -gt 0 }
$sj = ConvertFrom-D2Journal (JBytes @((Jl 0 'NOTE' "roledir-identity C:\Users\d2svc-$Run\d2-$Run role=svc sid=$Svc run=$Run"), (Jl 1 'DONE' "mkdir $W"))) 'svc' $Svc $Run "C:\Users\d2svc-$Run\d2-$Run"
Check 'Codex F8: cross-role DONE without delegation is flagged' { @(Test-D2JournalBalance @{ admin = $aj; build = $bje; svc = $sj } @{ build = $Adm; svc = $Svc }).Length -gt 0 }
$oj = ConvertFrom-D2Journal (JBytes @((Jl 0 'NOTE' "roledir-identity $W role=build sid=$Adm run=$Run"), (Jl 1 'BEGIN' 'x'), (Jl 2 'BEGIN' 'x'), (Jl 3 'DONE' 'x'))) 'build' $Adm $Run $W
Check 'Codex F8: duplicate BEGIN of one operation is flagged' { @(Test-D2JournalBalance @{ build = $oj } @{ build = $Adm }).Length -gt 0 }
$open = ConvertFrom-D2Journal (JBytes @((Jl 0 'NOTE' "roledir-identity $W role=build sid=$Adm run=$Run"), (Jl 1 'BEGIN' 'go build'))) 'build' $Adm $Run $W
Check 'BEGIN without DONE is flagged' { @(Test-D2JournalBalance @{ build = $open } @{ build = $Adm }).Length -eq 1 }

"---- C3 state"
function Ce([string]$t) { [pscustomobject]@{ Kind = 'NOTE'; Text = $t; Ts = $ts0; N = 1 } }
Check 'bind, pending, weakened, verified -> RESTORED' { (Get-D2C3State @((Ce 'C3 BIND id=1 pre-sha256=A'), (Ce 'C3 PENDING-RESTORE id=1'), (Ce 'C3 weakened'), (Ce 'C3 revert verified'))) -ceq 'RESTORED' }
Check 'weakened without revert -> WEAKENED' { (Get-D2C3State @((Ce 'C3 BIND id=1 pre-sha256=A'), (Ce 'C3 PENDING-RESTORE id=1'), (Ce 'C3 weakened'))) -ceq 'WEAKENED' }
Check 'Codex F7: grant failure after write-ahead -> PENDING (restoration owed)' { (Get-D2C3State @((Ce 'C3 BIND id=1 pre-sha256=A'), (Ce 'C3 PENDING-RESTORE id=1'), (Ce 'C3 GRANT FAILED x'))) -ceq 'PENDING' }
Check 'grant failure then verified restore -> RESTORED' { (Get-D2C3State @((Ce 'C3 BIND id=1 pre-sha256=A'), (Ce 'C3 PENDING-RESTORE id=1'), (Ce 'C3 GRANT FAILED x'), (Ce 'C3 revert verified'))) -ceq 'RESTORED' }
Check 'revert failure -> FAILED' { (Get-D2C3State @((Ce 'C3 BIND id=1 pre-sha256=A'), (Ce 'C3 PENDING-RESTORE id=1'), (Ce 'C3 weakened'), (Ce 'C3 REVERT FAILED x'))) -ceq 'FAILED' }
Check 'weakened without write-ahead -> CONTRADICTORY' { (Get-D2C3State @((Ce 'C3 weakened'))) -ceq 'CONTRADICTORY' }

"---- cleanup decisions"
$dirAll = @{ Exists = 'YES'; PrestateAbsent = $true; CreateProven = $true; ChainOk = $true; MarkerOk = $true; OwnerOk = $true; NoReparseInside = $true }
Check 'directory with complete ownership evidence -> REMOVE' { (Get-D2DirCleanupDecision $dirAll).Action -ceq 'REMOVE' }
foreach ($k in 'PrestateAbsent', 'CreateProven', 'ChainOk', 'MarkerOk', 'OwnerOk', 'NoReparseInside') {
  $x = $dirAll.Clone(); $x[$k] = $false
  Check "directory without $k -> RESIDUAL" { (Get-D2DirCleanupDecision $x).Action -ceq 'RESIDUAL' }
}
Check 'directory existence unknown -> RESIDUAL' { (Get-D2DirCleanupDecision @{ Exists = 'UNKNOWN' }).Action -ceq 'RESIDUAL' }
$acAll = @{ Exists = 'YES'; JournaledSid = $Svc; ActualSid = $Svc; PrestateAbsent = $true; BeginDone = $true; PreCsvLacks = $true; DescriptionOk = $true; GroupsOk = $true; ProfileOk = $true; ProfileLoaded = $false }
Check 'account with complete evidence -> REMOVE' { (Get-D2AccountCleanupDecision $acAll).Action -ceq 'REMOVE' }
$x = $acAll.Clone(); $x.JournaledSid = $null
Check 'Codex F8: account without a journaled SID is never matched to itself -> RESIDUAL' { (Get-D2AccountCleanupDecision $x).Action -ceq 'RESIDUAL' }
$x = $acAll.Clone(); $x.ActualSid = $Usr
Check 'account SID differs from journal -> RESIDUAL' { (Get-D2AccountCleanupDecision $x).Action -ceq 'RESIDUAL' }
$x = $acAll.Clone(); $x.ProfileLoaded = $null
Check 'profile load state unknown -> RESIDUAL' { (Get-D2AccountCleanupDecision $x).Action -ceq 'RESIDUAL' }
$known = [pscustomobject]@{ State = 'KNOWN'; Names = [string[]]@($Distro) }
$wAll = @{ Listing = $known; Distro = $Distro; PreEmpty = $true; ImportBegin = $true; Registry = 'MATCH' }
Check 'identified run-owned registration -> UNREGISTER' { (Get-D2WslCleanupDecision $wAll).Action -ceq 'UNREGISTER' }
foreach ($k in 'PreEmpty', 'ImportBegin') { $x = $wAll.Clone(); $x[$k] = $false; Check "Codex F8: registration without $k -> RESIDUAL" { (Get-D2WslCleanupDecision $x).Action -ceq 'RESIDUAL' } }
$x = $wAll.Clone(); $x.Registry = 'MISMATCH'
Check 'registration not positively identified -> RESIDUAL' { (Get-D2WslCleanupDecision $x).Action -ceq 'RESIDUAL' }
$x = $wAll.Clone(); $x.Listing = [pscustomobject]@{ State = 'UNKNOWN'; Names = [string[]]@() }
Check 'Codex F8: unknown listing never authorizes unregister' { (Get-D2WslCleanupDecision $x).Action -ceq 'RESIDUAL' }
$x = $wAll.Clone(); $x.Listing = [pscustomobject]@{ State = 'KNOWN'; Names = [string[]]@() }; $x.Registry = 'ABSENT'
Check 'not listed and no registry key -> ABSENT' { (Get-D2WslCleanupDecision $x).Action -ceq 'ABSENT' }
$x.Registry = 'MATCH'
Check 'not listed but registry key present -> RESIDUAL' { (Get-D2WslCleanupDecision $x).Action -ceq 'RESIDUAL' }

"---- Go LookPath emulation"
$fs = @{ 'C:\Windows\System32\wsl.exe' = 'FILE'; 'C:\Bad\wsl.exe.com' = 'FILE'; 'C:\Cwd\wsl.exe' = 'FILE'; 'C:\DirOnly\wsl.exe' = 'DIR' }
$probe = { param($p) if ($fs.ContainsKey($p)) { $fs[$p] } else { 'ABSENT' } }
Check 'approved launcher resolves first' { (Resolve-D2GoLookPath 'wsl.exe' 'C:\Windows\System32;C:\Windows' '.COM;.EXE' 'C:\Empty' $false $probe).Path -ceq 'C:\Windows\System32\wsl.exe' }
Check 'Codex F1/F4: PATHEXT candidate in an earlier directory wins, as in Go' { (Resolve-D2GoLookPath 'wsl.exe' 'C:\Bad;C:\Windows\System32' '.COM;.EXE' 'C:\Empty' $false $probe).Path -ceq 'C:\Bad\wsl.exe.com' }
Check 'Codex F1/F4: current directory candidate is detected' { (Resolve-D2GoLookPath 'wsl.exe' 'C:\Windows\System32' '' 'C:\Cwd' $false $probe).Dot }
Check 'directory named wsl.exe is skipped' { (Resolve-D2GoLookPath 'wsl.exe' 'C:\DirOnly;C:\Windows\System32' '' 'C:\Empty' $false $probe).Path -ceq 'C:\Windows\System32\wsl.exe' }
Throws 'quoted PATH is refused' { Resolve-D2GoLookPath 'wsl.exe' '"C:\x";C:\Windows\System32' '' 'C:\Empty' $false $probe }

"---- F9 scoring"
function Ev([string]$R, [string[]]$C = @()) { [pscustomobject]@{ Result = $R; Reason = ''; Verdict = $R; Codes = [string[]]$C } }
$tW = $ts0.AddMinutes(30); $tR = $ts0.AddMinutes(40)
function Full {
  $e = @{}; $c = @{}
  foreach ($p in (Get-D2Plan)) {
    $codes = @(); if ($p.Code) { $codes = @($p.Code) }
    $e[$p.Tag] = @((Ev $p.Want $codes))
    $t = switch ($p.Phase) { 'base' { $ts0.AddMinutes(10) } 'c3' { $ts0.AddMinutes(35) } default { $ts0.AddMinutes(45) } }
    $c[$p.Tag] = @([pscustomobject]@{ Result = $p.Want; Ts = $t })
  }
  return @{ E = $e; C = $c }
}
$admFacts = @{ FixtureVerified = $true; C3State = 'RESTORED'; C3WeakenedTs = $tW; C3RestoredTs = $tR; Incomplete = @() }
$f = Full; $s = Get-D2Score $f.E $f.C $admFacts 'READY'
Check 'complete unique evidence -> both diagnostics PASS' { $s.Labels['DISPOSABLE-HOST FILE/ACL DIAGNOSTIC'] -ceq 'PASS' -and $s.Labels['OPERATOR-RUN WSL DIAGNOSTIC'] -ceq 'PASS' }
Check 'threat-model-B and installed-system labels stay UNVERIFIED; D2 OPEN' { $s.Labels['THREAT-MODEL-B WSL ISOLATION'] -ceq 'UNVERIFIED' -and $s.Labels['INSTALLED-SYSTEM SECURITY'] -ceq 'UNVERIFIED' -and $s.D2 -ceq 'OPEN' -and $s.Labels.Count -eq 4 }
$f = Full; $f.E['S-A'] = @((Ev 'ISOLATED'), (Ev 'ISOLATED'))
Check 'Codex F9: duplicated evidence for one tag -> not PASS' { (Get-D2Score $f.E $f.C $admFacts 'READY').Labels['DISPOSABLE-HOST FILE/ACL DIAGNOSTIC'] -ceq 'UNVERIFIED' }
$f = Full; $f.C['N-E'] = @([pscustomobject]@{ Result = 'ISOLATED'; Ts = $ts0.AddMinutes(10) }, [pscustomobject]@{ Result = 'ISOLATED'; Ts = $ts0.AddMinutes(11) })
Check 'Codex F9: duplicated journal claim -> not PASS' { (Get-D2Score $f.E $f.C $admFacts 'READY').Labels['DISPOSABLE-HOST FILE/ACL DIAGNOSTIC'] -ceq 'UNVERIFIED' }
$f = Full; $f.C['N-A'] = @([pscustomobject]@{ Result = 'NOT_ISOLATED'; Ts = $ts0.AddMinutes(10) })
Check 'journal claim contradicting evidence -> not PASS' { (Get-D2Score $f.E $f.C $admFacts 'READY').Labels['DISPOSABLE-HOST FILE/ACL DIAGNOSTIC'] -ceq 'UNVERIFIED' }
$f = Full; $f.E.Remove('C2'); $f.C.Remove('C2')
Check 'Codex F9: missing evidence for one tag -> not PASS' { (Get-D2Score $f.E $f.C $admFacts 'READY').Labels['DISPOSABLE-HOST FILE/ACL DIAGNOSTIC'] -ceq 'UNVERIFIED' }
$f = Full; $f.E['S-E'] = @((Ev 'NOT_ISOLATED' @('UNTRUSTED_ACCESS'))); $f.C['S-E'] = @([pscustomobject]@{ Result = 'NOT_ISOLATED'; Ts = $ts0.AddMinutes(10) })
Check 'baseline NOT_ISOLATED -> FAIL' { (Get-D2Score $f.E $f.C $admFacts 'READY').Labels['DISPOSABLE-HOST FILE/ACL DIAGNOSTIC'] -ceq 'FAIL' }
$f = Full; $f.E['C3-S'] = @((Ev 'ISOLATED')); $f.C['C3-S'] = @([pscustomobject]@{ Result = 'ISOLATED'; Ts = $ts0.AddMinutes(35) })
Check 'C3 control not detected -> FAIL' { (Get-D2Score $f.E $f.C $admFacts 'READY').Labels['DISPOSABLE-HOST FILE/ACL DIAGNOSTIC'] -ceq 'FAIL' }
$f = Full
Check 'preflight BLOCKED -> WSL BLOCKED, native still scored separately' { $x = Get-D2Score $f.E $f.C $admFacts 'BLOCKED'; $x.Labels['OPERATOR-RUN WSL DIAGNOSTIC'] -ceq 'BLOCKED' -and $x.Labels['DISPOSABLE-HOST FILE/ACL DIAGNOSTIC'] -ceq 'PASS' }
Check 'contradictory preflight -> WSL UNVERIFIED' { (Get-D2Score $f.E $f.C $admFacts 'CONTRADICTORY').Labels['OPERATOR-RUN WSL DIAGNOSTIC'] -ceq 'UNVERIFIED' }
$a2 = $admFacts.Clone(); $a2.C3State = 'FAILED'; $a2.C3RestoredTs = $null
Check 'Codex F7: failed restoration -> native BLOCKED' { (Get-D2Score $f.E $f.C $a2 'READY').Labels['DISPOSABLE-HOST FILE/ACL DIAGNOSTIC'] -ceq 'BLOCKED' }
$f = Full; $f.C['S-E2'] = @([pscustomobject]@{ Result = 'ISOLATED'; Ts = $ts0.AddMinutes(38) })
Check 'restored run before verified revert -> not PASS' { (Get-D2Score $f.E $f.C $admFacts 'READY').Labels['DISPOSABLE-HOST FILE/ACL DIAGNOSTIC'] -ceq 'UNVERIFIED' }
$f = Full; $f.E['N-E'] = @((Ev 'PROCEDURE-ERROR')); $f.C['N-E'] = @([pscustomobject]@{ Result = 'PROCEDURE-ERROR'; Ts = $ts0.AddMinutes(10) })
Check 'native not PASS -> WSL cannot PASS' { $x = Get-D2Score $f.E $f.C $admFacts 'READY'; $x.Labels['DISPOSABLE-HOST FILE/ACL DIAGNOSTIC'] -ceq 'UNVERIFIED' -and $x.Labels['OPERATOR-RUN WSL DIAGNOSTIC'] -cne 'PASS' }
$a3 = $admFacts.Clone(); $a3.Incomplete = @('journal-svc: missing')
$f = Full
Check 'incomplete collection -> not PASS' { (Get-D2Score $f.E $f.C $a3 'READY').Labels['DISPOSABLE-HOST FILE/ACL DIAGNOSTIC'] -ceq 'UNVERIFIED' }

"---- F0 parser inventory decision"
$exp = @('a.psm1', 'b.ps1')
Check 'Codex F0: empty folder is FAIL' { (Get-D2ParseVerdict $exp @() @{}).Verdict -ceq 'FAIL' }
Check 'missing expected file is FAIL' { (Get-D2ParseVerdict $exp @('a.psm1') @{ 'a.psm1' = 0 }).Verdict -ceq 'FAIL' }
Check 'unexpected extra file is FAIL' { (Get-D2ParseVerdict $exp @('a.psm1', 'b.ps1', 'c.ps1') @{ 'a.psm1' = 0; 'b.ps1' = 0; 'c.ps1' = 0 }).Verdict -ceq 'FAIL' }
Check 'parse error is FAIL' { (Get-D2ParseVerdict $exp @('a.psm1', 'b.ps1') @{ 'a.psm1' = 0; 'b.ps1' = 2 }).Verdict -ceq 'FAIL' }
Check 'empty expected list is FAIL' { (Get-D2ParseVerdict @() @() @{}).Verdict -ceq 'FAIL' }
Check 'exact set, all parsed -> PASS' { (Get-D2ParseVerdict $exp @('b.ps1', 'a.psm1') @{ 'a.psm1' = 0; 'b.ps1' = 0 }).Verdict -ceq 'PASS' }

"---- ACL models (SDDL parsing only)"
$sd = 'O:BAD:P(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)'
Check 'SDDL model: owner, protection and ACEs' { $m = ConvertFrom-D2Sddl $sd; $m.Owner -ceq 'S-1-5-32-544' -and $m.Protected -and (($m.Aces | Sort-Object) -join ';') -ceq 'S-1-5-18|0x1F01FF|3;S-1-5-32-544|0x1F01FF|3' }
Throws 'deny ACE is refused' { ConvertFrom-D2Sddl 'O:BAD:P(D;;FA;;;WD)(A;OICI;FA;;;SY)' }
Throws 'ACL model mismatch throws' { Test-D2AclModel $sd 'S-1-5-32-544' $true @('S-1-5-18|0x1F01FF|3') }
Check 'C3 grant adds exactly one allow ACE and keeps the rest' { $m = ConvertFrom-D2Sddl (Add-D2SddlAllowAce $sd $Usr '0x1200A9' 3); $m.Aces.Length -eq 3 -and $m.Aces -ccontains "$Usr|0x1200A9|3" -and $m.Protected }
Throws 'C3 grant refuses a SID that already has an ACE' { Add-D2SddlAllowAce $sd 'S-1-5-18' '0x1200A9' 3 }

"---- plan"
Check 'plan has 15 unique tags and keeps native/WSL separate' { $p = Get-D2Plan; $p.Length -eq 15 -and @($p | ForEach-Object Tag | Sort-Object -Unique).Count -eq 15 -and @($p | Where-Object { $_.Mode -ceq 'wsl' -and $_.Diag -cne 'wsl' }).Count -eq 0 }
Check 'C2 runs only in the build window, never as wsl' { $r = Get-D2PlanRow 'C2'; $r.Role -ceq 'build' -and $r.Mode -ceq 'normal' }

""
"TESTS: $($script:Total - $script:Failed)/$($script:Total) passed"
if ($script:Failed) { exit 1 } else { exit 0 }
