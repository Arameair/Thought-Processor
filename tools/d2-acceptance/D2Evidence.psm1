# D2Evidence.psm1 - pure evidence parsing, cleanup decisions and scoring for the D2 P0
# disposable acceptance procedure. PROPOSED, NOT EXECUTED.
#
# Every function here works only on values passed in (bytes, strings, hashtables).
# Nothing in this module touches the file system, registry, accounts, ACLs, processes,
# environment or network. Test-D2Package.ps1 enforces that with an AST command check.
#
# Harness binding: Arameair/identity 1718c3d36c919daca6cbc8cc8ef5e4155ed9c33f,
# internal/profile/boundary/windows_test.go (TestAcceptanceDisposableProtectedProfile),
# boundary.go, probe_eval.go, probe_windows.go, paths.go.

Set-StrictMode -Version 3.0

$script:HarnessSha = '1718c3d36c919daca6cbc8cc8ef5e4155ed9c33f'
$script:T          = 'TestAcceptanceDisposableProtectedProfile'
# t.Logf call sites at that SHA: logReport (line 60, called without t.Helper), t.Fatalf (251), t.Logf (253).
$script:LineReport = 60
$script:LineFatal  = 251
$script:LinePass   = 253
$script:EmDash     = [string][char]0x2014
$script:Utf8Strict = New-Object System.Text.UTF8Encoding($false, $true)
$script:NonPass    = @('NOT_ISOLATED', 'UNVERIFIED', 'UNSUPPORTED')

# Copied from boundary.go at the harness SHA (AdminAuthorityFiles / EntityAuthorityFiles).
$script:AdminFiles  = @('profile.json','settings.json','admin.json','admin-throttle.json','approvals.json',
  'grants.json','installations.json','installation-keys.json','installation-nonces.json','paired-clients.json',
  'paired-nonces.json','identities.json','drafts.json','github-app.json','logs/identity-events.jsonl','runtime/service.json')
$script:EntityFiles = @('entity.json','credential','decisions.jsonl')
# probe_windows.go Probe() right names, in attempt order.
$script:DirRights  = @('list','add-file','add-subdir','delete-child','delete','write-dac','take-ownership')
$script:FileRights = @('read','write','write-attributes','delete','write-dac','take-ownership')
$script:AncRights  = @('delete','delete-child','write-dac','take-ownership')
# probe_eval.go wslRootRights / wslFileRights.
$script:WslRootRights = @('list','create')
$script:WslFileRights = @('read','write')

# The four result labels (revision 2 section 1). They are reported separately and never collapsed.
$script:Labels = @('DISPOSABLE-HOST FILE/ACL DIAGNOSTIC','OPERATOR-RUN WSL DIAGNOSTIC','THREAT-MODEL-B WSL ISOLATION','INSTALLED-SYSTEM SECURITY')

function Get-D2AuthorityFiles([string]$Kind) {
  if ($Kind -ceq 'admin')  { return ,([string[]]$script:AdminFiles) }
  if ($Kind -ceq 'entity') { return ,([string[]]$script:EntityFiles) }
  throw "kind $Kind"
}

function Get-D2Labels { return ,([string[]]$script:Labels) }

# ---- run plan ---------------------------------------------------------------------------
# One row per harness run. Role = the account window that must produce it. NormalSidOf says which
# SID is passed as IDENTITY_D2_NORMAL_SID. Phase orders the run against the C3 gates.
function Get-D2Plan {
  $rows = @(
    @('S-A',  'svc',  'service','admin', 'ISOLATED',    '',                         'native','base',    'usr'),
    @('S-E',  'svc',  'service','entity','ISOLATED',    '',                         'native','base',    'usr'),
    @('N-A',  'usr',  'normal', 'admin', 'ISOLATED',    '',                         'native','base',    'usr'),
    @('N-E',  'usr',  'normal', 'entity','ISOLATED',    '',                         'native','base',    'usr'),
    @('C1a',  'usr',  'service','entity','NOT_ISOLATED','PROCESS_IS_NORMAL_USER',   'native','base',    'usr'),
    @('C1b',  'svc',  'normal', 'entity','UNVERIFIED',  'PROBE_RUNNING_AS_SERVICE', 'native','base',    'usr'),
    @('C2',   'build','normal', 'entity','NOT_ISOLATED','NORMAL_USER_IS_ADMIN',     'native','base',    'admin'),
    @('W-A',  'usr',  'wsl',    'admin', 'ISOLATED',    '',                         'wsl',   'base',    'usr'),
    @('W-E',  'usr',  'wsl',    'entity','ISOLATED',    '',                         'wsl',   'base',    'usr'),
    @('C3-S', 'svc',  'service','entity','NOT_ISOLATED','UNTRUSTED_ACCESS',         'native','c3',      'usr'),
    @('C3-N', 'usr',  'normal', 'entity','NOT_ISOLATED','NORMAL_USER_ALLOWED',      'native','c3',      'usr'),
    @('C3-W', 'usr',  'wsl',    'entity','NOT_ISOLATED','WSL_ACCESS_ALLOWED',       'wsl',   'c3',      'usr'),
    @('S-E2', 'svc',  'service','entity','ISOLATED',    '',                         'native','restored','usr'),
    @('N-E2', 'usr',  'normal', 'entity','ISOLATED',    '',                         'native','restored','usr'),
    @('W-E2', 'usr',  'wsl',    'entity','ISOLATED',    '',                         'wsl',   'restored','usr')
  )
  $out = New-Object System.Collections.Generic.List[object]
  foreach ($r in $rows) {
    $out.Add([pscustomobject]@{ Tag=$r[0]; Role=$r[1]; Mode=$r[2]; Kind=$r[3]; Want=$r[4]; Code=$r[5]; Diag=$r[6]; Phase=$r[7]; NormalSidOf=$r[8] })
  }
  return ,($out.ToArray())
}

function Get-D2PlanRow([string]$Tag) {
  $m = @((Get-D2Plan) | Where-Object { $_.Tag -ceq $Tag })
  if ($m.Count -ne 1) { throw "unknown run tag $Tag" }
  return $m[0]
}

# Exact set of log tags each role window may write (each tag = <tag>.out.txt, .err.txt, .exit.txt).
function Get-D2RoleTags([string]$Role) {
  $ops = @{
    'build' = @('git-clone','git-checkout','git-head','git-status','go-version','go-vet','go-test','go-test-skip','go-build')
    'svc'   = @('svc-list')
    'usr'   = @('usr-list','wsl-list-pre','wsl-import','wsl-default','wsl-list-post','wsl-dd','wsl-mountinfo','wsl-conf',
                'wsl-list-clean1','wsl-terminate','wsl-unregister','wsl-list-clean2')
  }
  if (-not $ops.ContainsKey($Role)) { throw "role $Role has no collected logs" }
  $tags = New-Object System.Collections.Generic.List[string]
  foreach ($t in $ops[$Role]) { $tags.Add($t) }
  foreach ($p in (Get-D2Plan)) { if ($p.Role -ceq $Role) { $tags.Add($p.Tag); $tags.Add("$($p.Tag)-whoami") } }
  return ,($tags.ToArray())
}

# ---- byte / text helpers ------------------------------------------------------------------
function ConvertFrom-D2Utf8([byte[]]$Bytes) {
  if ($null -eq $Bytes) { throw 'record missing' }
  if ($Bytes.Length -ge 3 -and $Bytes[0] -eq 0xEF -and $Bytes[1] -eq 0xBB -and $Bytes[2] -eq 0xBF) { throw 'unexpected byte-order mark' }
  if ([Array]::IndexOf($Bytes, [byte]0) -ge 0) { throw 'NUL byte (not UTF-8 text)' }
  return $script:Utf8Strict.GetString($Bytes)
}

function Split-D2Lines([string]$Text, [switch]$AllowCrLf) {
  if ($AllowCrLf) { $Text = $Text.Replace("`r`n", "`n") }
  if ($Text.Contains("`r")) { throw 'unexpected carriage return' }
  $l = $Text.Split([char]10)
  $n = $l.Length
  if ($n -gt 0 -and $l[$n - 1] -ceq '') { $n-- }
  $out = New-Object string[] $n
  [Array]::Copy($l, $out, $n)
  return ,$out
}

# Exit record written by Invoke-D2 ("<int>" via Set-Content, or "BLOCKED launch: ...", or "UNKNOWN").
function ConvertFrom-D2ExitText([string]$Text) {
  if ($null -eq $Text) { return [pscustomobject]@{ Kind='MISSING'; Code=$null; Detail='exit record missing' } }
  $m = [regex]::Match($Text, '\A(-?[0-9]{1,10})(\r\n)?\z')
  if ($m.Success) {
    $n = 0
    if ([int]::TryParse($m.Groups[1].Value, [ref]$n)) { return [pscustomobject]@{ Kind='EXIT'; Code=$n; Detail='' } }
  }
  if ($Text.StartsWith('BLOCKED launch: ', [StringComparison]::Ordinal)) { return [pscustomobject]@{ Kind='BLOCKED'; Code=$null; Detail=$Text.Trim() } }
  return [pscustomobject]@{ Kind='MALFORMED'; Code=$null; Detail='exit record is not an integer' }
}

# ---- JSON structure (json.MarshalIndent(v, "", "  ") output) ------------------------------
# Returns the top-level keys in order. Throws on a duplicate key at any depth, a key inside an
# array, or unbalanced nesting. ConvertFrom-Json alone would silently keep one duplicate.
function Get-D2JsonTopKeys([string[]]$Lines) {
  $stack = New-Object System.Collections.Generic.List[object]
  $top = New-Object System.Collections.Generic.List[string]
  $seenTop = $false
  foreach ($ln in $Lines) {
    $t = $ln.TrimStart(' ')
    if ($t.StartsWith('}') -or $t.StartsWith(']')) {
      if ($stack.Count -eq 0) { throw 'unbalanced JSON' }
      $stack.RemoveAt($stack.Count - 1)
      if ($stack.Count -eq 0) { $seenTop = $true }
      continue
    }
    if ($seenTop) { throw 'content after the report object' }
    $m = [regex]::Match($t, '\A"((?:[^"\\]|\\.)*)": ')
    if ($m.Success) {
      if ($stack.Count -eq 0) { throw 'key outside an object' }
      $cur = $stack[$stack.Count - 1]
      if ($null -eq $cur) { throw 'key inside an array' }
      if (-not $cur.Add($m.Groups[1].Value)) { throw "duplicate key $($m.Groups[1].Value)" }
      if ($stack.Count -eq 1) { $top.Add($m.Groups[1].Value) }
    }
    if ($t.EndsWith('{')) { $stack.Add((New-Object 'System.Collections.Generic.HashSet[string]' -ArgumentList ([StringComparer]::Ordinal))) }
    elseif ($t.EndsWith('[')) { $stack.Add($null) }
  }
  if ($stack.Count -ne 0 -or -not $seenTop) { throw 'unbalanced JSON' }
  return ,($top.ToArray())
}

function Get-D2Names($Obj) { return ,([string[]]@($Obj.PSObject.Properties | ForEach-Object { $_.Name })) }

function Test-D2IsInt($v) { return ($v -is [int] -or $v -is [long]) }

# verdictOf() from boundary.go, recomputed from the reported findings.
function Get-D2VerdictOf([object[]]$Findings) {
  $v = 'ISOLATED'
  foreach ($f in $Findings) {
    if ($f.code -ceq 'PLATFORM_UNSUPPORTED') { return 'UNSUPPORTED' }
    if ($f.violation) { $v = 'NOT_ISOLATED' } elseif ($v -ceq 'ISOLATED') { $v = 'UNVERIFIED' }
  }
  return $v
}

# Expected ordered (path, right) envelope of an ISOLATED normal-mode Probe report.
function Get-D2ProbeEnvelope([string]$Root, [string[]]$Files) {
  $e = New-Object System.Collections.Generic.List[string]
  foreach ($r in $script:DirRights) { $e.Add("$Root|$r") }
  foreach ($f in $Files) { $p = $Root + '\' + $f.Replace('/', '\'); foreach ($r in $script:FileRights) { $e.Add("$p|$r") } }
  $parts = $Root.Substring(3).Split('\')
  for ($i = $parts.Length - 1; $i -ge 1; $i--) {
    $a = $Root.Substring(0, 3) + (($parts[0..($i - 1)]) -join '\')
    foreach ($r in $script:AncRights) { $e.Add("$a|$r") }
  }
  foreach ($r in $script:AncRights) { $e.Add($Root.Substring(0, 3) + "|$r") }
  return ,($e.ToArray())
}

function Get-D2WslEnvelope([string]$Root, [string[]]$Files) {
  $e = New-Object System.Collections.Generic.List[string]
  foreach ($r in $script:WslRootRights) { $e.Add("$Root|wsl $r") }
  foreach ($f in $Files) { $p = $Root + '\' + $f.Replace('/', '\'); foreach ($r in $script:WslFileRights) { $e.Add("$p|wsl $r") } }
  return ,($e.ToArray())
}

# Validates one parsed report against the run record. Returns the sorted unique finding codes.
function Test-D2Report($O, [string[]]$Keys, [hashtable]$Run, [string]$Want) {
  $mode = $Run.Mode
  switch -CaseSensitive ($mode) {
    'service' { $order = @('verdict','root','process_sid','objects_checked','ancestors_checked','findings','scope','not_covered') }
    'normal'  { $order = @('verdict','user_sid','attempts','findings','scope') }
    'wsl'     { $order = @('verdict','user_sid','attempts','findings','scope'); if ($Keys -ccontains 'wsl') { $order += 'wsl' } }
    default   { throw "mode $mode" }
  }
  if (($Keys -join ',') -cne ($order -join ',')) { throw "report keys [$($Keys -join ',')], expected [$($order -join ',')]" }
  if (-not ($O.verdict -is [string]) -or $O.verdict -cne $Want) { throw "report verdict '$($O.verdict)' contradicts terminal result $Want" }
  if (-not ($O.scope -is [string]) -or $O.scope.Length -eq 0) { throw 'report scope missing' }

  $fs = @(); if ($null -ne $O.findings) { $fs = @($O.findings) }
  $codes = New-Object System.Collections.Generic.List[string]
  foreach ($f in $fs) {
    if ($null -eq $f -or -not ($f -is [System.Management.Automation.PSCustomObject])) { throw 'finding is not an object' }
    $fk = Get-D2Names $f
    foreach ($k in $fk) { if ($k -cnotin 'code','path','sid','detail','violation') { throw "finding key $k" } }
    foreach ($k in 'code','detail','violation') { if ($fk -cnotcontains $k) { throw "finding lacks $k" } }
    if (-not ($f.code -is [string]) -or $f.code -cnotmatch '\A[A-Z_]+\z') { throw 'finding code malformed' }
    if (-not ($f.violation -is [bool])) { throw 'finding violation flag malformed' }
    if (-not ($f.detail -is [string])) { throw 'finding detail malformed' }
    $codes.Add($f.code)
  }
  $rv = Get-D2VerdictOf $fs
  if ($rv -cne $Want) { throw "findings imply $rv, report says $Want" }

  if ($mode -ceq 'service') {
    if (-not ($O.process_sid -is [string]) -or $O.process_sid -cne $Run.CallerSid) { throw "process_sid '$($O.process_sid)' is not the caller $($Run.CallerSid)" }
    if (-not ($O.root -is [string]) -or $O.root -cne $Run.Root) { throw "report root '$($O.root)' is not $($Run.Root)" }
    if (-not (Test-D2IsInt $O.objects_checked) -or -not (Test-D2IsInt $O.ancestors_checked)) { throw 'object counts malformed' }
    $nc = @(); if ($null -ne $O.not_covered) { $nc = @($O.not_covered) }
    if ($nc.Count -eq 0) { throw 'not_covered missing' }
    if ($Want -ceq 'ISOLATED') {
      if ($O.objects_checked -lt (1 + $Run.Files.Count) -or $O.ancestors_checked -lt 1) { throw 'ISOLATED report checked too few objects' }
    }
  } else {
    if (-not ($O.user_sid -is [string]) -or $O.user_sid -cne $Run.CallerSid) { throw "user_sid '$($O.user_sid)' is not the caller $($Run.CallerSid)" }
    $att = @(); if ($null -ne $O.attempts) { $att = @($O.attempts) }
    $got = New-Object System.Collections.Generic.List[string]
    $results = New-Object System.Collections.Generic.List[string]
    foreach ($a in $att) {
      if ($null -eq $a -or -not ($a -is [System.Management.Automation.PSCustomObject])) { throw 'attempt is not an object' }
      if (((Get-D2Names $a) -join ',') -cne 'path,right,result') { throw 'attempt keys' }
      foreach ($k in 'path','right','result') { if (-not ($a.$k -is [string])) { throw "attempt $k malformed" } }
      $got.Add("$($a.path)|$($a.right)"); $results.Add($a.result)
    }
    if ($Want -ceq 'ISOLATED') {
      foreach ($r in $results) { if ($r -cne 'DENIED') { throw "ISOLATED report has attempt result $r" } }
      if ($mode -ceq 'normal') {
        $exp = Get-D2ProbeEnvelope $Run.Root $Run.Files
        if ($got.Count -ne $exp.Length + 1) { throw "attempt count $($got.Count), expected $($exp.Length + 1)" }
        for ($i = 0; $i -lt $exp.Length; $i++) { if ($got[$i] -cne $exp[$i]) { throw "attempt $i is '$($got[$i])', expected '$($exp[$i])'" } }
        if ($got[$exp.Length] -cnotmatch ('\A' + [regex]::Escape($Run.Root + '\.boundary-probe-') + '[0-9a-f]{12}\|create-file\z')) { throw 'create-file attempt missing or malformed' }
      } else {
        $exp = Get-D2WslEnvelope $Run.Root $Run.Files
        if (($got.ToArray() -join "`n") -cne ($exp -join "`n")) { throw 'WSL attempt envelope differs from the expected targets and rights' }
        if (-not ($Keys -ccontains 'wsl')) { throw 'ISOLATED WSL report lacks the wsl block' }
        $w = $O.wsl
        if (((Get-D2Names $w) -join ',') -cne 'distro,path,drive,mount_point,mount_root,mount_dev,fs_type,source,not_covered') { throw 'wsl block keys' }
        if ($w.distro -cne $Run.Distro) { throw "WSL report distro '$($w.distro)' is not $($Run.Distro)" }
        if ($w.mount_root -cne '/') { throw 'WSL mount_root is not /' }
        if (@($w.not_covered).Count -eq 0) { throw 'wsl not_covered missing' }
      }
    }
    if ($mode -ceq 'normal' -and ($Keys -ccontains 'wsl')) { throw 'normal report carries a wsl block' }
  }
  return ,([string[]]@($codes | Sort-Object -Unique))
}

# ---- F6: complete harness run classification -------------------------------------------
# Returns Result = ISOLATED | NOT_ISOLATED | UNVERIFIED | UNSUPPORTED | BLOCKED | PROCEDURE-ERROR.
# ISOLATED needs exit 0, the exact RUN/report/pass/PASS envelope, one well-formed report bound to
# the run record (mode, root, caller SID, distro, authority files) and no findings.
function ConvertFrom-D2HarnessRun {
  param([byte[]]$Stdout, [byte[]]$Stderr, [string]$ExitText, [Parameter(Mandatory = $true)][hashtable]$Run)
  foreach ($k in 'Mode','Kind','Root','CallerSid','Distro') { if (-not $Run.ContainsKey($k)) { throw "run record lacks $k" } }
  $Run = $Run.Clone()
  $Run.Files = Get-D2AuthorityFiles $Run.Kind
  $mk = { param($res, $why, $v, $c) if ($null -eq $c) { $c = @() }; [pscustomobject]@{ Result=$res; Reason=$why; Verdict=$v; Codes=[string[]]@($c) } }
  $ex = ConvertFrom-D2ExitText $ExitText
  if ($ex.Kind -ceq 'BLOCKED') { return (& $mk 'BLOCKED' $ex.Detail $null @()) }
  if ($ex.Kind -cne 'EXIT') { return (& $mk 'PROCEDURE-ERROR' $ex.Detail $null @()) }
  try {
    if ($null -eq $Stderr) { throw 'stderr record missing' }
    if ($Stderr.Length -ne 0) { throw 'stderr is not empty' }
    $lines = Split-D2Lines (ConvertFrom-D2Utf8 $Stdout)
    if ($lines.Length -lt 6) { throw 'harness output is incomplete' }
    if ($lines[0] -cne "=== RUN   $script:T") { throw 'first line is not the single expected RUN line' }
    if ($lines[1] -cne "    windows_test.go:$($script:LineReport): {") { throw 'structured report does not start on line 2' }
    $json = New-Object System.Collections.Generic.List[string]
    $json.Add('{')
    $i = 2
    while ($i -lt $lines.Length -and $lines[$i].StartsWith('        ', [StringComparison]::Ordinal)) { $json.Add($lines[$i].Substring(8)); $i++ }
    if ($json[$json.Count - 1] -cne '}') { throw 'structured report is not terminated' }
    if ($lines.Length - $i -ne 3) { throw "expected 3 terminal lines after the report, found $($lines.Length - $i)" }
    $term = $lines[$i]; $tline = $lines[$i + 1]; $final = $lines[$i + 2]
    $dur = ' \([0-9]+\.[0-9]{2}s\)\z'
    if ($ex.Code -eq 0) {
      $pass = "    windows_test.go:$($script:LinePass): $($Run.Mode) half ISOLATED for $($Run.Root) $script:EmDash file/ACL scope only; see not_covered"
      if ($term -cne $pass) { throw 'exit 0 without the exact ISOLATED line for this mode and root' }
      if ($tline -cnotmatch ('\A--- PASS: ' + $script:T + $dur) -or $final -cne 'PASS') { throw 'exit 0 without the PASS envelope' }
      $want = 'ISOLATED'
    } elseif ($ex.Code -eq 1) {
      $m = [regex]::Match($term, '\A    windows_test\.go:' + $script:LineFatal + ': ' + [regex]::Escape($Run.Mode) + ' half: ([A-Z_]+) \(fail closed\)\z')
      if (-not $m.Success) { throw 'exit 1 without the fail-closed line for this mode' }
      if ($tline -cnotmatch ('\A--- FAIL: ' + $script:T + $dur) -or $final -cne 'FAIL') { throw 'exit 1 without the FAIL envelope' }
      $want = $m.Groups[1].Value
      if ($script:NonPass -cnotcontains $want) { throw "fail-closed verdict $want" }
    } else { throw "exit $($ex.Code)" }
    $jl = $json.ToArray()
    $keys = Get-D2JsonTopKeys $jl
    $obj = ConvertFrom-Json -InputObject ($jl -join "`n")
    if (-not ($obj -is [System.Management.Automation.PSCustomObject])) { throw 'report is not a JSON object' }
    $codes = Test-D2Report $obj $keys $Run $want
    return (& $mk $want '' $want $codes)
  } catch {
    return (& $mk 'PROCEDURE-ERROR' $_.Exception.Message $null @())
  }
}

# ---- WSL listing (wsl.exe -l -q with WSL_UTF8=1) ----------------------------------------
# State KNOWN (Names = exact registered names) or UNKNOWN. Unknown is never treated as empty.
$script:WslEmptyFirst = 'Windows Subsystem for Linux has no installed distributions.'
$script:WslEmptyFollow = @(
  'Distributions can be installed by visiting the Microsoft Store:',
  'https://aka.ms/wslstore',
  'You can resolve this by installing a distribution with the instructions below:',
  "Use 'wsl.exe --list --online' to list available distributions",
  "and 'wsl.exe --install <Distro>' to install."
)
function ConvertFrom-D2WslListing {
  param([string]$ExitText, [byte[]]$Stdout, [byte[]]$Stderr)
  $unk = { param($why) [pscustomobject]@{ State='UNKNOWN'; Names=[string[]]@(); Reason=$why } }
  try {
    $ex = ConvertFrom-D2ExitText $ExitText
    if ($ex.Kind -cne 'EXIT') { return (& $unk "exit record $($ex.Kind)") }
    $ol = Split-D2Lines (ConvertFrom-D2Utf8 $Stdout) -AllowCrLf
    $el = Split-D2Lines (ConvertFrom-D2Utf8 $Stderr) -AllowCrLf
    $out = @($ol | ForEach-Object { $_.Trim() } | Where-Object { $_ -ne '' })
    $err = @($el | ForEach-Object { $_.Trim() } | Where-Object { $_ -ne '' })
    if ($ex.Code -eq 0) {
      if ($err.Count -ne 0) { return (& $unk 'exit 0 with stderr output') }
      if ($out.Count -eq 0) { return (& $unk 'exit 0 with an empty listing') }
      $set = New-Object 'System.Collections.Generic.HashSet[string]' -ArgumentList ([StringComparer]::OrdinalIgnoreCase)
      foreach ($n in $out) {
        if ($n -cnotmatch '\A[A-Za-z0-9][A-Za-z0-9._-]{0,63}\z') { return (& $unk "unrecognized listing line") }
        if (-not $set.Add($n)) { return (& $unk "duplicate name $n") }
      }
      return [pscustomobject]@{ State='KNOWN'; Names=[string[]]$out; Reason='listing' }
    }
    if ($out.Count -gt 0) { $msg = $out } else { $msg = $err }
    if ($out.Count -gt 0 -and $err.Count -gt 0) { return (& $unk 'nonzero exit with output on both streams') }
    if ($msg.Count -eq 0 -or $msg[0] -cne $script:WslEmptyFirst) { return (& $unk "nonzero exit ($($ex.Code)) without the recognized empty-state message") }
    for ($i = 1; $i -lt $msg.Count; $i++) {
      if ($script:WslEmptyFollow -ccontains $msg[$i]) { continue }
      if ($msg[$i] -cmatch '\AError code: Wsl/[A-Za-z/]*WSL_E_DEFAULT_DISTRO_NOT_FOUND\z') { continue }
      return (& $unk 'unrecognized line in the empty-state message')
    }
    return [pscustomobject]@{ State='KNOWN'; Names=[string[]]@(); Reason='empty-state message' }
  } catch { return (& $unk $_.Exception.Message) }
}

# ---- journals ----------------------------------------------------------------------------
# Strict grammar: "<ISO-8601 o> <BEGIN|DONE|NOTE> <text>". Line 1 must be the role identity line
# "NOTE roledir-identity <dir> role=<role> sid=<sid> run=<runid>". Malformed input throws.
function ConvertFrom-D2Journal {
  param([byte[]]$Bytes, [string]$Role, [string]$Sid, [string]$RunId, [string]$Dir)
  $lines = Split-D2Lines (ConvertFrom-D2Utf8 $Bytes) -AllowCrLf
  if ($lines.Length -eq 0) { throw "journal-$Role is empty" }
  $out = New-Object System.Collections.Generic.List[object]
  $n = 0
  foreach ($l in $lines) {
    $n++
    $m = [regex]::Match($l, '\A([0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}\.[0-9]{7}[+-][0-9]{2}:[0-9]{2}) (BEGIN|DONE|NOTE) (\S.*)\z')
    if (-not $m.Success) { throw "journal-$Role line $n is malformed" }
    $ts = [DateTimeOffset]::ParseExact($m.Groups[1].Value, 'o', [Globalization.CultureInfo]::InvariantCulture)
    $out.Add([pscustomobject]@{ N=$n; Ts=$ts; Kind=$m.Groups[2].Value; Text=$m.Groups[3].Value; Role=$Role })
  }
  $id = "roledir-identity $Dir role=$Role sid=$Sid run=$RunId"
  if ($out[0].Kind -cne 'NOTE' -or $out[0].Text -cne $id) { throw "journal-$Role identity line is not '$id'" }
  for ($i = 1; $i -lt $out.Count; $i++) {
    if ($out[$i].Text.StartsWith('roledir-identity ', [StringComparison]::Ordinal)) { throw "journal-$Role has a second identity line" }
    if ($out[$i].Ts -lt $out[$i - 1].Ts) { throw "journal-$Role line $($out[$i].N) goes back in time" }
  }
  return ,($out.ToArray())
}

function Get-D2Notes([object[]]$Entries, [string]$Prefix) {
  return ,([string[]]@($Entries | Where-Object { $_.Kind -ceq 'NOTE' -and $_.Text.StartsWith($Prefix, [StringComparison]::Ordinal) } | ForEach-Object { $_.Text.Substring($Prefix.Length) }))
}

# Every BEGIN must be closed by exactly one DONE in the SAME journal, in order. The only cross-journal
# pairing is an explicit admin delegation "NOTE delegate <op> role=<r> sid=<s>": admin has the BEGIN,
# role r has the single DONE, and r's identity line carries sid s. Returns a list of problems.
function Test-D2JournalBalance([hashtable]$Journals, [hashtable]$Sids) {
  $problems = New-Object System.Collections.Generic.List[string]
  $deleg = @{}
  if ($Journals.ContainsKey('admin')) {
    foreach ($d in (Get-D2Notes $Journals['admin'] 'delegate ')) {
      $m = [regex]::Match($d, '\A(.+) role=(build|svc|usr) sid=(S-1-[0-9-]+)\z')
      if (-not $m.Success) { $problems.Add("malformed delegation: $d"); continue }
      if ($deleg.ContainsKey($m.Groups[1].Value)) { $problems.Add("duplicate delegation: $($m.Groups[1].Value)"); continue }
      $deleg[$m.Groups[1].Value] = @($m.Groups[2].Value, $m.Groups[3].Value)
    }
  }
  foreach ($role in @($Journals.Keys)) {
    $open = @{}
    foreach ($e in $Journals[$role]) {
      if ($e.Kind -ceq 'BEGIN') {
        if ($open.ContainsKey($e.Text)) { $problems.Add("journal-${role}: BEGIN '$($e.Text)' repeated while open"); continue }
        $open[$e.Text] = $e.N
      } elseif ($e.Kind -ceq 'DONE') {
        if ($open.ContainsKey($e.Text)) { $open.Remove($e.Text); continue }
        $d = $null; if ($deleg.ContainsKey($e.Text)) { $d = $deleg[$e.Text] }
        if ($null -ne $d -and $d[0] -ceq $role -and $Sids.ContainsKey($role) -and $Sids[$role] -ceq $d[1]) {
          $deleg[$e.Text] = @($d[0], $d[1], 'done'); continue
        }
        $problems.Add("journal-${role}: DONE '$($e.Text)' without its own BEGIN or a matching delegation")
      }
    }
    foreach ($k in @($open.Keys)) {
      if ($role -ceq 'admin' -and $deleg.ContainsKey($k)) { continue }
      $problems.Add("journal-${role}: BEGIN without DONE: $k")
    }
  }
  $adminBegins = @(); if ($Journals.ContainsKey('admin')) { $adminBegins = @($Journals['admin'] | Where-Object { $_.Kind -ceq 'BEGIN' } | ForEach-Object { $_.Text }) }
  foreach ($k in @($deleg.Keys)) {
    if (@($adminBegins | Where-Object { $_ -ceq $k }).Count -ne 1) { $problems.Add("delegated operation lacks exactly one admin BEGIN: $k") }
    if ($deleg[$k].Count -ne 3) { $problems.Add("delegated operation not completed by $($deleg[$k][0]): $k") }
  }
  return ,($problems.ToArray())
}

# C3 state from the protected admin journal. NONE | PENDING | WEAKENED | RESTORED | FAILED | CONTRADICTORY.
function Get-D2C3State([object[]]$AdminEntries) {
  $c3 = @($AdminEntries | Where-Object { $_.Kind -ceq 'NOTE' -and $_.Text.StartsWith('C3 ', [StringComparison]::Ordinal) })
  $state = 'NONE'
  foreach ($e in $c3) {
    $t = $e.Text
    if ($t.StartsWith('C3 BIND ', [StringComparison]::Ordinal)) { if ($state -cne 'NONE') { return 'CONTRADICTORY' }; $state = 'BOUND'; continue }
    if ($t.StartsWith('C3 PENDING-RESTORE ', [StringComparison]::Ordinal)) { if ($state -cne 'BOUND') { return 'CONTRADICTORY' }; $state = 'PENDING'; continue }
    if ($t -ceq 'C3 weakened') { if ($state -cne 'PENDING') { return 'CONTRADICTORY' }; $state = 'WEAKENED'; continue }
    if ($t -ceq 'C3 revert verified') { if ($state -cnotin 'PENDING','WEAKENED') { return 'CONTRADICTORY' }; $state = 'RESTORED'; continue }
    if ($t.StartsWith('C3 REVERT FAILED', [StringComparison]::Ordinal)) { return 'FAILED' }
    if ($t.StartsWith('C3 GRANT FAILED', [StringComparison]::Ordinal)) { if ($state -cne 'PENDING') { return 'CONTRADICTORY' }; continue }
    return 'CONTRADICTORY'
  }
  if ($state -ceq 'BOUND') { return 'NONE' }
  return $state
}

# ---- cleanup decisions (pure; the host module gathers the facts) -------------------------
# Each returns Action = REMOVE | UNREGISTER | ABSENT | RESIDUAL, with a reason. Anything unproven is RESIDUAL.
function Get-D2DirCleanupDecision([hashtable]$F) {
  if ($F['Exists'] -ceq 'UNKNOWN') { return [pscustomobject]@{ Action='RESIDUAL'; Reason='existence could not be determined' } }
  if ($F['Exists'] -ceq 'NO') { return [pscustomobject]@{ Action='ABSENT'; Reason='absent' } }
  if ($F['Exists'] -cne 'YES') { return [pscustomobject]@{ Action='RESIDUAL'; Reason='existence fact missing' } }
  foreach ($k in 'PrestateAbsent','CreateProven','ChainOk','MarkerOk','OwnerOk','NoReparseInside') {
    if ($F[$k] -ne $true) { return [pscustomobject]@{ Action='RESIDUAL'; Reason="ownership not proven: $k" } }
  }
  return [pscustomobject]@{ Action='REMOVE'; Reason='run-owned' }
}

function Get-D2AccountCleanupDecision([hashtable]$F) {
  if ($F['Exists'] -ceq 'NO') { return [pscustomobject]@{ Action='ABSENT'; Reason='absent' } }
  if ($F['Exists'] -cne 'YES') { return [pscustomobject]@{ Action='RESIDUAL'; Reason='existence could not be determined' } }
  if (-not $F['JournaledSid']) { return [pscustomobject]@{ Action='RESIDUAL'; Reason='no journaled SID; the account is never matched to itself' } }
  if ($F['ActualSid'] -cne $F['JournaledSid']) { return [pscustomobject]@{ Action='RESIDUAL'; Reason='SID differs from the protected journal' } }
  foreach ($k in 'PrestateAbsent','BeginDone','PreCsvLacks','DescriptionOk','GroupsOk','ProfileOk') {
    if ($F[$k] -ne $true) { return [pscustomobject]@{ Action='RESIDUAL'; Reason="ownership not proven: $k" } }
  }
  if ($F['ProfileLoaded'] -ne $false) { return [pscustomobject]@{ Action='RESIDUAL'; Reason='profile loaded or unknown; sign out and retry' } }
  return [pscustomobject]@{ Action='REMOVE'; Reason='run-owned' }
}

# Registry: MATCH (exactly one Lxss key named Distro with the run's BasePath), ABSENT, MISMATCH, UNKNOWN.
function Get-D2WslCleanupDecision([hashtable]$F) {
  $l = $F['Listing']
  if ($null -eq $l -or $l.State -cne 'KNOWN') { return [pscustomobject]@{ Action='RESIDUAL'; Reason='registration state unknown (listing not recognized)' } }
  $listed = @($l.Names) -ccontains $F['Distro']
  if (-not $listed) {
    if ($F['Registry'] -ceq 'ABSENT') { return [pscustomobject]@{ Action='ABSENT'; Reason='not registered' } }
    return [pscustomobject]@{ Action='RESIDUAL'; Reason="listing and registry disagree ($($F['Registry']))" }
  }
  if ($F['PreEmpty'] -ne $true) { return [pscustomobject]@{ Action='RESIDUAL'; Reason='no recognized empty pre-import listing' } }
  if ($F['ImportBegin'] -ne $true) { return [pscustomobject]@{ Action='RESIDUAL'; Reason='registered without a write-ahead import record' } }
  if ($F['Registry'] -cne 'MATCH') { return [pscustomobject]@{ Action='RESIDUAL'; Reason="registration not positively identified ($($F['Registry']))" } }
  return [pscustomobject]@{ Action='UNREGISTER'; Reason='run-owned registration' }
}

# ---- Go exec.LookPath emulation (Windows) -------------------------------------------------
# $Probe: scriptblock(path) -> 'FILE' | 'DIR' | 'ABSENT'. Mirrors os/exec lp_windows.go: the current
# directory first unless NoDefaultCurrentDirectoryInExePath is set, then each PATH entry; within a
# directory the exact name first (it has an extension), then name + each PATHEXT extension.
function Resolve-D2GoLookPath {
  param([string]$Name, [string]$PathVar, [string]$PathExt, [string]$Cwd, [bool]$NoCwd, [scriptblock]$Probe)
  if ($PathVar.Contains('"')) { throw 'PATH contains quotes; Go SplitList semantics not emulated' }
  $exts = New-Object System.Collections.Generic.List[string]
  if ($PathExt) { foreach ($e in $PathExt.ToLowerInvariant().Split(';')) { if ($e) { if (-not $e.StartsWith('.')) { $e = ".$e" }; $exts.Add($e) } } }
  else { foreach ($e in '.com','.exe','.bat','.cmd') { $exts.Add($e) } }
  $find = {
    param($base)
    if ((& $Probe $base) -ceq 'FILE') { return $base }
    foreach ($e in $exts) { if ((& $Probe ($base + $e)) -ceq 'FILE') { return ($base + $e) } }
    return $null
  }
  if (-not $NoCwd) {
    $c = & $find (($Cwd.TrimEnd('\')) + '\' + $Name)
    if ($c) { return [pscustomobject]@{ Path=$c; Dot=$true } }
  }
  foreach ($d in $PathVar.Split(';')) {
    if (-not $d) { continue }
    $c = & $find (($d.TrimEnd('\')) + '\' + $Name)
    if ($c) { return [pscustomobject]@{ Path=$c; Dot=$false } }
  }
  return [pscustomobject]@{ Path=$null; Dot=$false }
}

# ---- F9 scoring --------------------------------------------------------------------------
# $Evidence: tag -> list of recomputed results (one per collected triplet, from that tag's role logs).
# $Claims:   tag -> list of RESULT notes (the result each role journal recorded), with .Ts.
# $Admin:    @{ FixtureVerified=bool; C3State=<Get-D2C3State>; C3WeakenedTs; C3RestoredTs; Incomplete=[string[]] }
# $Preflight: READY | BLOCKED | MISSING | CONTRADICTORY (from exactly one usr WSL-PREFLIGHT note)
function Get-D2Score {
  param([hashtable]$Evidence, [hashtable]$Claims, [hashtable]$Admin, [string]$Preflight)
  $rows = @{}
  foreach ($p in (Get-D2Plan)) {
    $ev = @(); if ($Evidence.ContainsKey($p.Tag)) { $ev = @($Evidence[$p.Tag]) }
    $cl = @(); if ($Claims.ContainsKey($p.Tag)) { $cl = @($Claims[$p.Tag]) }
    $st = $null; $why = ''
    if ($ev.Count -eq 0) {
      if ($cl.Count -eq 0) { $st = 'NOT-REACHED' }
      elseif (@($cl | Where-Object { $_.Result -like 'BLOCKED*' }).Count -eq $cl.Count) { $st = 'BLOCKED'; $why = 'journal records BLOCKED' }
      else { $st = 'UNVERIFIED'; $why = 'journal claims a result with no collected evidence' }
    } elseif ($ev.Count -gt 1 -or $cl.Count -ne 1) {
      $st = 'UNVERIFIED'; $why = "evidence x$($ev.Count), journal claims x$($cl.Count); exactly one each is required"
    } elseif ($cl[0].Result -cne $ev[0].Result) {
      $st = 'UNVERIFIED'; $why = "journal says $($cl[0].Result), evidence recomputes to $($ev[0].Result)"
    } else {
      $r = $ev[0]
      $ts = $cl[0].Ts
      $phaseOk = $true
      if ($p.Phase -ceq 'base' -and $null -ne $Admin.C3WeakenedTs -and $ts -ge $Admin.C3WeakenedTs) { $phaseOk = $false }
      if ($p.Phase -ceq 'c3' -and ($null -eq $Admin.C3WeakenedTs -or $ts -le $Admin.C3WeakenedTs -or ($null -ne $Admin.C3RestoredTs -and $ts -ge $Admin.C3RestoredTs))) { $phaseOk = $false }
      if ($p.Phase -ceq 'restored' -and ($Admin.C3State -cne 'RESTORED' -or $null -eq $Admin.C3RestoredTs -or $ts -le $Admin.C3RestoredTs)) { $phaseOk = $false }
      if (-not $phaseOk) { $st = 'UNVERIFIED'; $why = "run is outside its $($p.Phase) phase" }
      elseif ($r.Result -ceq 'BLOCKED') { $st = 'BLOCKED'; $why = $r.Reason }
      elseif ($r.Result -ceq 'PROCEDURE-ERROR') { $st = 'PROCEDURE-ERROR'; $why = $r.Reason }
      elseif ($r.Result -ceq $p.Want -and ($p.Code -eq '' -or @($r.Codes) -ccontains $p.Code)) { $st = 'MET' }
      else { $st = 'NOT-MET'; $why = "got $($r.Result)/$(@($r.Codes) -join ',')" }
    }
    $res = $null; if ($ev.Count -eq 1) { $res = $ev[0].Result }
    $rows[$p.Tag] = [pscustomobject]@{ Tag=$p.Tag; Diag=$p.Diag; Phase=$p.Phase; Want=$p.Want; Status=$st; Result=$res; Why=$why }
  }
  $plan = Get-D2Plan
  $label = {
    param([string]$diag, [bool]$gate, [string]$gateBlock)
    $mine = @($plan | Where-Object { $_.Diag -ceq $diag } | ForEach-Object { $rows[$_.Tag] })
    foreach ($x in $mine) {
      if ($x.Status -cnotin 'MET','NOT-MET') { continue }
      if ($x.Want -ceq 'ISOLATED' -and $x.Result -ceq 'NOT_ISOLATED') { return 'FAIL' }
      if ($x.Phase -ceq 'c3' -and $x.Result -cne 'NOT_ISOLATED') { return 'FAIL' }
    }
    if ($gate -and @($mine | Where-Object { $_.Status -cne 'MET' }).Count -eq 0) { return 'PASS' }
    if ($gateBlock -or @($mine | Where-Object { $_.Status -ceq 'BLOCKED' }).Count -gt 0) { return 'BLOCKED' }
    return 'UNVERIFIED'
  }
  $c3Block = ''
  if ($Admin.C3State -cin 'FAILED','PENDING','WEAKENED') { $c3Block = "C3 restoration not verified ($($Admin.C3State)); disposable-VM recovery required" }
  $incomplete = @(); if ($Admin.ContainsKey('Incomplete') -and $null -ne $Admin.Incomplete) { $incomplete = @($Admin.Incomplete) }
  $nativeGate = ($Admin.FixtureVerified -eq $true) -and ($Admin.C3State -ceq 'RESTORED') -and ($incomplete.Count -eq 0)
  $native = & $label 'native' $nativeGate $c3Block
  if ($Preflight -cne 'READY') {
    $wsl = 'BLOCKED'; if ($Preflight -ceq 'CONTRADICTORY') { $wsl = 'UNVERIFIED' }
  } else {
    $wsl = & $label 'wsl' ($nativeGate -and $native -ceq 'PASS') $c3Block
  }
  return [pscustomobject]@{
    Labels = [ordered]@{
      'DISPOSABLE-HOST FILE/ACL DIAGNOSTIC' = $native
      'OPERATOR-RUN WSL DIAGNOSTIC'         = $wsl
      'THREAT-MODEL-B WSL ISOLATION'        = 'UNVERIFIED'
      'INSTALLED-SYSTEM SECURITY'           = 'UNVERIFIED'
    }
    Rows = @($plan | ForEach-Object { $rows[$_.Tag] })
    Preflight = $Preflight
    C3Block = $c3Block
    D2 = 'OPEN'
  }
}

# ---- ACL models (pure .NET parsing of SDDL; no object is opened) -------------------------
# ACE strings are "<sid>|0x<mask hex>|<AceFlags int>"; only plain access-allowed ACEs are accepted.
function ConvertFrom-D2Sddl([string]$Sddl) {
  $rsd = New-Object System.Security.AccessControl.RawSecurityDescriptor -ArgumentList $Sddl
  if ($null -eq $rsd.DiscretionaryAcl) { throw 'NULL DACL' }
  $prot = (($rsd.ControlFlags -band [System.Security.AccessControl.ControlFlags]::DiscretionaryAclProtected) -ne 0)
  $aces = New-Object System.Collections.Generic.List[string]
  foreach ($a in $rsd.DiscretionaryAcl) {
    if (-not ($a -is [System.Security.AccessControl.CommonAce]) -or $a.AceQualifier -ne [System.Security.AccessControl.AceQualifier]::AccessAllowed -or $a.IsCallback) { throw "unsupported ACE in $Sddl" }
    $aces.Add(('{0}|0x{1:X}|{2}' -f $a.SecurityIdentifier.Value, $a.AccessMask, [int]$a.AceFlags))
  }
  $owner = $null; if ($null -ne $rsd.Owner) { $owner = $rsd.Owner.Value }
  return [pscustomobject]@{ Owner = $owner; Protected = $prot; Aces = [string[]]$aces.ToArray() }
}

function Test-D2AclModel([string]$Sddl, [string]$Owner, [bool]$Protected, [string[]]$Aces) {
  $m = ConvertFrom-D2Sddl $Sddl
  if ($m.Owner -cne $Owner) { throw "STOP: owner $($m.Owner), expected $Owner" }
  if ($m.Protected -ne $Protected) { throw "STOP: DACL protected=$($m.Protected), expected $Protected" }
  $g = (@($m.Aces | Sort-Object) -join ' ; '); $e = (@($Aces | Sort-Object) -join ' ; ')
  if ($g -cne $e) { throw "STOP: ACL`n  got      $g`n  expected $e" }
}

function Add-D2SddlAllowAce([string]$Sddl, [string]$Sid, [string]$Mask, [int]$Flags) {
  $rsd = New-Object System.Security.AccessControl.RawSecurityDescriptor -ArgumentList $Sddl
  foreach ($a in $rsd.DiscretionaryAcl) { if ($a.SecurityIdentifier.Value -ceq $Sid) { throw "STOP: $Sid already has an ACE" } }
  $sidObj = New-Object System.Security.Principal.SecurityIdentifier -ArgumentList $Sid
  $maskInt = [Convert]::ToInt32($Mask.Substring(2), 16)
  $ace = [System.Security.AccessControl.CommonAce]::new([System.Security.AccessControl.AceFlags]$Flags, [System.Security.AccessControl.AceQualifier]::AccessAllowed, $maskInt, $sidObj, $false, $null)
  $rsd.DiscretionaryAcl.InsertAce($rsd.DiscretionaryAcl.Count, $ace)
  return $rsd.GetSddlForm([System.Security.AccessControl.AccessControlSections]'Owner, Access')
}

# ---- F0: parser inventory decision (pure) -------------------------------------------------
# $Found: relative paths actually present. $Parsed: relative path -> parser error count.
function Get-D2ParseVerdict([string[]]$Expected, [string[]]$Found, [hashtable]$Parsed) {
  $problems = New-Object System.Collections.Generic.List[string]
  if ($null -eq $Expected -or $Expected.Length -eq 0) { $problems.Add('expected file list is empty') }
  if ($null -eq $Found) { $Found = @() }
  $e = @($Expected | Sort-Object); $f = @($Found | Sort-Object)
  if ($f.Count -eq 0) { $problems.Add('no script files found') }
  foreach ($x in $e) { if ($f -cnotcontains $x) { $problems.Add("missing $x") } }
  foreach ($x in $f) { if ($e -cnotcontains $x) { $problems.Add("unexpected $x") } }
  if (@($f | Sort-Object -Unique).Count -ne $f.Count) { $problems.Add('duplicate file entries') }
  foreach ($x in $e) {
    if (-not $Parsed.ContainsKey($x)) { $problems.Add("not parsed: $x"); continue }
    if ($Parsed[$x] -isnot [int] -or $Parsed[$x] -ne 0) { $problems.Add("parse errors in ${x}: $($Parsed[$x])") }
  }
  $v = 'FAIL'; if ($problems.Count -eq 0) { $v = 'PASS' }
  return [pscustomobject]@{ Verdict=$v; Problems=[string[]]$problems.ToArray(); Count=$e.Count }
}

Export-ModuleMember -Function Get-D2AuthorityFiles, Get-D2Labels, Get-D2Plan, Get-D2PlanRow, Get-D2RoleTags,
  ConvertFrom-D2Utf8, Split-D2Lines, ConvertFrom-D2ExitText, Get-D2JsonTopKeys, Get-D2VerdictOf,
  Get-D2ProbeEnvelope, Get-D2WslEnvelope, ConvertFrom-D2HarnessRun, ConvertFrom-D2WslListing,
  ConvertFrom-D2Journal, Get-D2Notes, Test-D2JournalBalance, Get-D2C3State,
  Get-D2DirCleanupDecision, Get-D2AccountCleanupDecision, Get-D2WslCleanupDecision,
  Resolve-D2GoLookPath, Get-D2Score, Get-D2ParseVerdict, ConvertFrom-D2Sddl, Test-D2AclModel, Add-D2SddlAllowAce
