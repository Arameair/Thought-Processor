# Test-D2Package.ps1 - READ-ONLY package verification for the D2 P0 acceptance scripts (replaces F0).
# PROPOSED, NOT EXECUTED by the author.
#
#   powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\Test-D2Package.ps1
#
# Parses every script file with the PowerShell parser. Only the pure D2Evidence.psm1 is imported
# (for Get-D2ParseVerdict); D2Host.psm1 and the tests are parsed, never imported or run. Then:
#   1. the set of *.ps1 / *.psm1 / *.psd1 files must equal the expected list exactly (an empty or
#      partial folder is FAIL, never PASS);
#   2. every expected file must parse with zero errors;
#   3. every file must be ASCII (Windows PowerShell 5.1 reads BOM-less files as ANSI);
#   4. no command may receive a comparison/logical operator as a parameter
#      (the Patch F "Test-Path ... -or" defect: -or bound as a parameter name);
#   5. D2Evidence.psm1 may call only an allowlisted set of side-effect-free commands, and may
#      reference no file, directory, registry, process, network or environment .NET type.
# Prints the SHA-256 of every file so a reviewer can compare it with the audited commit.
# Exit code 0 only when every check passes.

param([string]$Root = $PSScriptRoot)

Set-StrictMode -Version 3.0
$ErrorActionPreference = 'Stop'

$Expected = @('D2Evidence.psm1', 'D2Host.psm1', 'Test-D2Package.ps1', 'tests\D2Evidence.Tests.ps1')
$Problems = New-Object System.Collections.Generic.List[string]

$found = @(Get-ChildItem -LiteralPath $Root -Recurse -File -Force |
  Where-Object { $_.Extension -in '.ps1', '.psm1', '.psd1' } |
  ForEach-Object { $_.FullName.Substring($Root.TrimEnd('\').Length + 1) })

$parsed = @{}
$asts = @{}
foreach ($rel in $found) {
  $full = Join-Path $Root $rel
  $bytes = [IO.File]::ReadAllBytes($full)
  foreach ($b in $bytes) { if ($b -gt 0x7E -or ($b -lt 0x20 -and $b -notin 9, 10, 13)) { $Problems.Add("$rel contains a non-ASCII or control byte"); break } }
  $tokens = $null; $errors = $null
  $ast = [System.Management.Automation.Language.Parser]::ParseFile($full, [ref]$tokens, [ref]$errors)
  $parsed[$rel] = [int]@($errors).Count
  foreach ($e in @($errors)) { $Problems.Add("$rel line $($e.Extent.StartLineNumber): $($e.Message)") }
  $asts[$rel] = $ast
  '{0}  {1}' -f (Get-FileHash -LiteralPath $full -Algorithm SHA256).Hash, $rel
}

$ops = 'or', 'and', 'xor', 'not', 'band', 'bor', 'eq', 'ne', 'gt', 'ge', 'lt', 'le', 'like', 'notlike', 'match', 'notmatch',
  'contains', 'notcontains', 'in', 'notin', 'is', 'isnot', 'replace', 'split', 'join'
$ops = @($ops) + @($ops | ForEach-Object { "c$_"; "i$_" })
foreach ($rel in @($asts.Keys)) {
  $cmds = $asts[$rel].FindAll({ param($n) $n -is [System.Management.Automation.Language.CommandAst] }, $true)
  foreach ($c in $cmds) {
    foreach ($el in $c.CommandElements) {
      if ($el -is [System.Management.Automation.Language.CommandParameterAst] -and $ops -contains $el.ParameterName.ToLowerInvariant()) {
        $Problems.Add("$rel line $($el.Extent.StartLineNumber): operator -$($el.ParameterName) is parsed as a parameter of '$($c.GetCommandName())'; parenthesize the command")
      }
    }
  }
}

if ($asts.ContainsKey('D2Evidence.psm1')) {
  $ast = $asts['D2Evidence.psm1']
  $own = @($ast.FindAll({ param($n) $n -is [System.Management.Automation.Language.FunctionDefinitionAst] }, $true) | ForEach-Object { $_.Name })
  $allowed = @('New-Object', 'ConvertFrom-Json', 'Sort-Object', 'Where-Object', 'ForEach-Object', 'Set-StrictMode', 'Export-ModuleMember') + $own
  foreach ($c in $ast.FindAll({ param($n) $n -is [System.Management.Automation.Language.CommandAst] }, $true)) {
    $n = $c.GetCommandName()
    if ($null -eq $n -or $allowed -notcontains $n) { $Problems.Add("D2Evidence.psm1 line $($c.Extent.StartLineNumber): command '$n' is not allowlisted for the pure module") }
  }
  $deny = '(?i)(^|\.)(IO\.(File|Directory|FileInfo|DirectoryInfo|FileStream|Path|DriveInfo)|File|Directory|Registry|RegistryKey|Process|Environment|WebClient|WebRequest|HttpClient|Socket|Marshal|Activator|AppDomain|Assembly)$'
  foreach ($t in $ast.FindAll({ param($n) $n -is [System.Management.Automation.Language.TypeExpressionAst] -or $n -is [System.Management.Automation.Language.TypeConstraintAst] }, $true)) {
    if ($t.TypeName.FullName -match $deny) { $Problems.Add("D2Evidence.psm1 line $($t.Extent.StartLineNumber): type [$($t.TypeName.FullName)] is not allowed in the pure module") }
  }
  foreach ($vx in $ast.FindAll({ param($n) $n -is [System.Management.Automation.Language.VariableExpressionAst] }, $true)) {
    if ($vx.VariablePath.DriveName -in 'env', 'HKLM', 'HKCU', 'function', 'variable') { $Problems.Add("D2Evidence.psm1 line $($vx.Extent.StartLineNumber): provider variable `$$($vx.VariablePath.UserPath) is not allowed in the pure module") }
  }
}

# The pure module is imported (to reuse its inventory decision) only after it parsed cleanly and
# passed the purity rules above.
$v = [pscustomobject]@{ Count = $Expected.Count; Problems = [string[]]@('D2Evidence.psm1 not imported: parse or purity problems') }
if ($parsed.ContainsKey('D2Evidence.psm1') -and $parsed['D2Evidence.psm1'] -eq 0 -and -not @($Problems | Where-Object { $_.StartsWith('D2Evidence.psm1') }).Count) {
  Import-Module (Join-Path $Root 'D2Evidence.psm1') -Force
  $v = Get-D2ParseVerdict $Expected ([string[]]$found) $parsed
}
foreach ($p in $v.Problems) { $Problems.Add($p) }

if ($Problems.Count -eq 0) { "PACKAGE CHECK: PASS ($($v.Count)/$($v.Count) expected files present and parsed; static rules clean)"; exit 0 }
"PACKAGE CHECK: FAIL ($($Problems.Count))"
$Problems
exit 1
