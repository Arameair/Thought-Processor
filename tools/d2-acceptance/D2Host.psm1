# D2Host.psm1 - host operations for the D2 P0 disposable acceptance procedure.
# PROPOSED, NOT EXECUTED.
#
# Every Invoke-D2Step* function and Invoke-D2Run changes the disposable VM (directories, accounts,
# ACLs, WSL registrations, downloads, processes). Run them only inside the owner-selected Option A
# VM, in the order of README.md, after the owner authorizes that exact run. Importing this module
# compiles a small P/Invoke helper and changes nothing on the host.

Set-StrictMode -Version 3.0
$ErrorActionPreference = 'Stop'
Import-Module (Join-Path $PSScriptRoot 'D2Evidence.psm1')

if (-not ('D2Native' -as [type])) {
  Add-Type -Language CSharp -TypeDefinition @'
using System;
using System.ComponentModel;
using System.IO;
using System.Runtime.InteropServices;
using System.Security.AccessControl;
using Microsoft.Win32.SafeHandles;

public static class D2Native {
  public const uint FILE_READ_ATTRIBUTES = 0x80, READ_CONTROL = 0x20000, WRITE_DAC = 0x40000, GENERIC_READ = 0x80000000;
  const uint OPEN_EXISTING = 3, FLAG_BACKUP = 0x02000000, FLAG_OPEN_REPARSE = 0x00200000;
  const int SE_FILE_OBJECT = 1;
  const uint OWNER_SI = 1, DACL_SI = 4, PROTECTED_DACL_SI = 0x80000000, UNPROTECTED_DACL_SI = 0x20000000;

  [StructLayout(LayoutKind.Sequential)]
  struct BHFI { public uint attr, ct1, ct2, at1, at2, wt1, wt2, serial, sizeHigh, sizeLow, links, idxHigh, idxLow; }
  [StructLayout(LayoutKind.Sequential)]
  struct SA { public int nLength; public IntPtr sd; public int inherit; }

  [DllImport("kernel32.dll", CharSet = CharSet.Unicode, SetLastError = true)]
  static extern SafeFileHandle CreateFileW(string n, uint a, uint s, IntPtr sa, uint d, uint f, IntPtr t);
  [DllImport("kernel32.dll", SetLastError = true)]
  static extern bool GetFileInformationByHandle(SafeFileHandle h, out BHFI i);
  [DllImport("kernel32.dll", CharSet = CharSet.Unicode, SetLastError = true)]
  static extern uint GetFinalPathNameByHandleW(SafeFileHandle h, System.Text.StringBuilder b, uint n, uint f);
  [DllImport("kernel32.dll", CharSet = CharSet.Unicode, SetLastError = true, EntryPoint = "CreateDirectoryW")]
  static extern bool CreateDirSa(string p, ref SA sa);
  [DllImport("kernel32.dll", CharSet = CharSet.Unicode, SetLastError = true, EntryPoint = "CreateDirectoryW")]
  static extern bool CreateDirNoSa(string p, IntPtr sa);
  [DllImport("kernel32.dll", CharSet = CharSet.Unicode, SetLastError = true)]
  static extern bool RemoveDirectoryW(string p);
  [DllImport("advapi32.dll", CharSet = CharSet.Unicode, SetLastError = true)]
  static extern bool ConvertStringSecurityDescriptorToSecurityDescriptorW(string s, uint r, out IntPtr sd, IntPtr len);
  [DllImport("advapi32.dll")]
  static extern uint GetSecurityInfo(SafeFileHandle h, int t, uint i, IntPtr o, IntPtr g, IntPtr d, IntPtr s, out IntPtr sd);
  [DllImport("advapi32.dll")]
  static extern uint SetSecurityInfo(SafeFileHandle h, int t, uint i, IntPtr o, IntPtr g, byte[] d, IntPtr s);
  [DllImport("advapi32.dll")]
  static extern uint GetSecurityDescriptorLength(IntPtr sd);
  [DllImport("kernel32.dll")]
  static extern IntPtr LocalFree(IntPtr p);

  public class Info {
    public uint Attributes, Serial, Links; public ulong FileId; public long Size;
    public bool IsDir { get { return (Attributes & 0x10) != 0; } }
    public bool IsReparse { get { return (Attributes & 0x400) != 0; } }
    public string Id { get { return Serial.ToString("X8") + ":" + FileId.ToString("X16"); } }
  }

  // Opens the named object itself: a reparse point is opened, never followed.
  public static SafeFileHandle Open(string path, uint access, bool anyType) {
    uint share = anyType ? 7u : 1u;
    SafeFileHandle h = CreateFileW(path, access, share, IntPtr.Zero, OPEN_EXISTING, FLAG_OPEN_REPARSE | (anyType ? FLAG_BACKUP : 0u), IntPtr.Zero);
    if (h.IsInvalid) { int e = Marshal.GetLastWin32Error(); h.Dispose(); throw new Win32Exception(e, "open " + path); }
    return h;
  }
  public static Info GetInfo(SafeFileHandle h) {
    BHFI b;
    if (!GetFileInformationByHandle(h, out b)) throw new Win32Exception(Marshal.GetLastWin32Error());
    Info i = new Info();
    i.Attributes = b.attr; i.Serial = b.serial; i.Links = b.links;
    i.FileId = ((ulong)b.idxHigh << 32) | b.idxLow; i.Size = ((long)b.sizeHigh << 32) | b.sizeLow;
    return i;
  }
  public static string FinalPath(SafeFileHandle h) {
    System.Text.StringBuilder sb = new System.Text.StringBuilder(1024);
    uint n = GetFinalPathNameByHandleW(h, sb, 1024, 0);
    if (n == 0 || n >= 1024) throw new Win32Exception(Marshal.GetLastWin32Error());
    string s = sb.ToString();
    if (s.StartsWith(@"\\?\") && !s.StartsWith(@"\\?\UNC\", StringComparison.OrdinalIgnoreCase)) s = s.Substring(4);
    return s;
  }
  public static string Sddl(SafeFileHandle h) {
    IntPtr sd;
    uint r = GetSecurityInfo(h, SE_FILE_OBJECT, OWNER_SI | DACL_SI, IntPtr.Zero, IntPtr.Zero, IntPtr.Zero, IntPtr.Zero, out sd);
    if (r != 0) throw new Win32Exception((int)r);
    try {
      int len = (int)GetSecurityDescriptorLength(sd);
      byte[] b = new byte[len]; Marshal.Copy(sd, b, 0, len);
      return new RawSecurityDescriptor(b, 0).GetSddlForm(AccessControlSections.Owner | AccessControlSections.Access);
    } finally { LocalFree(sd); }
  }
  public static void SetDacl(SafeFileHandle h, string sddl) {
    RawSecurityDescriptor rsd = new RawSecurityDescriptor(sddl);
    if (rsd.DiscretionaryAcl == null) throw new InvalidOperationException("SDDL has no DACL");
    byte[] acl = new byte[rsd.DiscretionaryAcl.BinaryLength];
    rsd.DiscretionaryAcl.GetBinaryForm(acl, 0);
    bool prot = (rsd.ControlFlags & ControlFlags.DiscretionaryAclProtected) != 0;
    uint r = SetSecurityInfo(h, SE_FILE_OBJECT, DACL_SI | (prot ? PROTECTED_DACL_SI : UNPROTECTED_DACL_SI), IntPtr.Zero, IntPtr.Zero, acl, IntPtr.Zero);
    if (r != 0) throw new Win32Exception((int)r);
  }
  public static byte[] ReadAll(SafeFileHandle h, long max) {
    FileStream fs = new FileStream(h, FileAccess.Read, 4096, false);
    MemoryStream ms = new MemoryStream();
    byte[] buf = new byte[65536]; int n;
    while ((n = fs.Read(buf, 0, buf.Length)) > 0) { ms.Write(buf, 0, n); if (ms.Length > max) throw new IOException("over the size limit"); }
    return ms.ToArray();
  }
  // Fails if anything (file, directory or link) already exists at the path. The DACL is set at creation.
  public static void CreateDirExclusive(string p, string sddl) {
    if (string.IsNullOrEmpty(sddl)) { if (!CreateDirNoSa(p, IntPtr.Zero)) throw new Win32Exception(Marshal.GetLastWin32Error(), "mkdir " + p); return; }
    IntPtr sd;
    if (!ConvertStringSecurityDescriptorToSecurityDescriptorW(sddl, 1, out sd, IntPtr.Zero)) throw new Win32Exception(Marshal.GetLastWin32Error(), "sddl");
    try {
      SA sa = new SA(); sa.nLength = Marshal.SizeOf(typeof(SA)); sa.sd = sd; sa.inherit = 0;
      if (!CreateDirSa(p, ref sa)) throw new Win32Exception(Marshal.GetLastWin32Error(), "mkdir " + p);
    } finally { LocalFree(sd); }
  }
  // Removes only an EMPTY directory.
  public static void RemoveEmptyDir(string p) { if (!RemoveDirectoryW(p)) throw new Win32Exception(Marshal.GetLastWin32Error(), "rmdir " + p); }
}
'@
}

$script:SY = 'S-1-5-18'; $script:BA = 'S-1-5-32-544'; $script:BU = 'S-1-5-32-545'
$script:TI = 'S-1-5-80-956008885-3418522649-1831038044-1853292631-2271478464'
$script:FA = '0x1F01FF'; $script:RX = '0x1200A9'
$script:Dummy = '{"d2_acceptance_dummy":true}'
$script:SHA = '1718c3d36c919daca6cbc8cc8ef5e4155ed9c33f'
$script:Utf8 = New-Object System.Text.UTF8Encoding($false)
$script:C = $null
# Package files published by F4 into the Administrators-owned $Bin\pkg for the svc/usr windows.
$script:PkgFiles = @('D2Evidence.psm1', 'D2Host.psm1', 'Test-D2Package.ps1', 'tests\D2Evidence.Tests.ps1')

# ---- context and journal -----------------------------------------------------------------
function Get-D2Sha([byte[]]$B) { return [BitConverter]::ToString([Security.Cryptography.SHA256]::Create().ComputeHash($B)).Replace('-', '') }

function JLine([string]$K, [string]$S) {
  $c = $script:C
  if (-not $c.J -or -not (Test-Path -LiteralPath $c.J -PathType Leaf)) { throw 'STOP: journal not available' }
  $S = $S.Replace("`r", ' ').Replace("`n", ' ')
  [IO.File]::AppendAllText($c.J, "$(Get-Date -Format o) $K $S`n", $script:Utf8)
}
function JBegin([string]$S) { JLine 'BEGIN' $S }
function JDone([string]$S) { JLine 'DONE' $S }
function JNote([string]$S) { JLine 'NOTE' $S }
function Need0($R, [string]$What) { if (-not ($R -is [int] -and $R -eq 0)) { throw "STOP: $What returned $R" } }

function Assert-D2Scope([string]$P) {
  $c = $script:C
  if ($P -match '\.\.' -or $P -match '\s') { throw "STOP: unsafe path $P" }
  $roots = @($c.Ev, $c.W, $c.Base, $c.Bin, $c.RoleDir) | Where-Object { $_ }
  if (-not @($roots | Where-Object { $P -ceq $_ -or $P.StartsWith("$_\", [StringComparison]::Ordinal) })) { throw "STOP: $P is outside the disposable scope" }
}

# Every component from C:\ down exists, is not a reparse point, and opens at exactly its own name.
function Assert-D2Chain([string]$P) {
  $paths = New-Object System.Collections.Generic.List[string]
  $paths.Add('C:\')
  if ($P -cne 'C:\') {
    if ($P -cnotmatch '\AC:\\[^\\]' -or $P.EndsWith('\') -or $P -match '\.\.' -or $P -match '\s') { throw "STOP: $P is not a plain C:\ path" }
    $cur = 'C:'
    foreach ($comp in $P.Substring(3).Split('\')) {
      if (-not $comp) { throw "STOP: empty component in $P" }
      $cur = "$cur\$comp"; $paths.Add($cur)
    }
  }
  foreach ($x in $paths) {
    $h = [D2Native]::Open($x, [D2Native]::FILE_READ_ATTRIBUTES, $true)
    try {
      if ([D2Native]::GetInfo($h).IsReparse) { throw "STOP: reparse point at $x" }
      $fp = [D2Native]::FinalPath($h)
      if ($fp.TrimEnd('\') -ine $x.TrimEnd('\')) { throw "STOP: $x resolves to $fp" }
    } finally { $h.Dispose() }
  }
}

# One no-follow handle: regular file, one link, exact final path, expected owner, size limit; the
# returned bytes are read from that same handle.
function Read-D2Safe([string]$Path, [string[]]$Owners, [long]$Max = 16MB) {
  Assert-D2Chain $Path
  $h = [D2Native]::Open($Path, [D2Native]::GENERIC_READ, $false)
  try {
    $i = [D2Native]::GetInfo($h)
    if ($i.IsDir -or $i.IsReparse) { throw "not a regular file: $Path" }
    if ($i.Links -ne 1) { throw "hard-linked ($($i.Links) links): $Path" }
    if ($i.Size -gt $Max) { throw "over $Max bytes: $Path" }
    $fp = [D2Native]::FinalPath($h); if ($fp -ine $Path) { throw "path substitution: $Path opened $fp" }
    $o = (ConvertFrom-D2Sddl ([D2Native]::Sddl($h))).Owner
    if ($Owners -cnotcontains $o) { throw "owner $o of $Path is not one of $($Owners -join ',')" }
    $b = [D2Native]::ReadAll($h, $Max)
    return [pscustomobject]@{ Bytes = $b; Sha256 = (Get-D2Sha $b); Id = $i.Id; Size = $b.Length }
  } finally { $h.Dispose() }
}

# Handle-level check of one object: type, no reparse, one link for files, exact path, ACL model.
function Assert-D2Obj([string]$P, [string]$Type, [string]$Owner, [bool]$Protected, [string[]]$Aces) {
  Assert-D2Chain $P
  $h = [D2Native]::Open($P, ([D2Native]::READ_CONTROL -bor [D2Native]::FILE_READ_ATTRIBUTES), $true)
  try {
    $i = [D2Native]::GetInfo($h)
    if ($i.IsReparse) { throw "STOP: reparse point $P" }
    if (($Type -ceq 'dir') -ne $i.IsDir) { throw "STOP: $P is not a $Type" }
    if (-not $i.IsDir -and $i.Links -ne 1) { throw "STOP: $P has $($i.Links) links" }
    $fp = [D2Native]::FinalPath($h); if ($fp -ine $P) { throw "STOP: $P resolves to $fp" }
    Test-D2AclModel ([D2Native]::Sddl($h)) $Owner $Protected $Aces
    return $i
  } finally { $h.Dispose() }
}

function Write-D2Bytes([string]$P, [byte[]]$B) {
  $fs = [IO.File]::Open($P, [IO.FileMode]::CreateNew, [IO.FileAccess]::Write, [IO.FileShare]::None)
  try { $fs.Write($B, 0, $B.Length) } finally { $fs.Dispose() }
}
function New-D2File([string]$P, $Content) {
  Assert-D2Scope $P
  if (Test-Path -LiteralPath $P) { throw "STOP: $P exists" }
  if ($Content -is [string]) { $Content = $script:Utf8.GetBytes($Content) }
  JBegin "write $P"; Write-D2Bytes $P $Content; JDone "write $P"
}
function New-D2Dir([string]$P, [string]$Sddl = $null) {
  Assert-D2Scope $P; Assert-D2Chain (Split-Path -Parent $P)
  if (Test-Path -LiteralPath $P) { throw "STOP: $P exists" }
  JBegin "mkdir $P"; [D2Native]::CreateDirExclusive($P, $Sddl); JDone "mkdir $P"
}

# Creates the role directory exclusively (with its final DACL when given), then its marker and the
# journal identity line. If marker or journal cannot be written, the call removes only what it created.
function New-D2RoleDir([string]$Sddl = $null) {
  $c = $script:C; $d = $c.RoleDir
  Assert-D2Chain (Split-Path -Parent $d)
  [D2Native]::CreateDirExclusive($d, $Sddl)
  $j = "$d\journal-$($c.Role).txt"; $ok = $false
  try {
    Write-D2Bytes "$d\.d2-run" ([Text.Encoding]::ASCII.GetBytes($c.RunId))
    Write-D2Bytes $j ($script:Utf8.GetBytes("$(Get-Date -Format o) NOTE roledir-identity $d role=$($c.Role) sid=$($c.Sid) run=$($c.RunId)`n"))
    $ok = $true
  } finally {
    if (-not $ok) {
      try {
        foreach ($f in $j, "$d\.d2-run") { if (Test-Path -LiteralPath $f -PathType Leaf) { [IO.File]::Delete($f) } }
        [D2Native]::RemoveEmptyDir($d)
        Write-Host "Rolled back $d (marker/journal write failed)." -ForegroundColor Yellow
      } catch { Write-Host "RESIDUAL (record by hand): $d could not be rolled back: $($_.Exception.Message)" -ForegroundColor Red }
    }
  }
  $c.J = $j
  if ($c.Role -ceq 'build') { JDone "mkdir $d" }
  New-D2Dir $c.LogDir
}

# The run record is written by the elevated admin into the Administrators-owned, protected $Bin.
function Get-D2RunRecord {
  $c = $script:C
  Assert-D2Obj $c.Bin 'dir' $script:BA $true @("$($script:SY)|$($script:FA)|3", "$($script:BA)|$($script:FA)|3", "$($script:BU)|$($script:RX)|3") | Out-Null
  $r = Read-D2Safe "$($c.Bin)\run-record.txt" @($script:BA) 64KB
  $rec = @{}
  foreach ($l in (Split-D2Lines (ConvertFrom-D2Utf8 $r.Bytes))) {
    $m = [regex]::Match($l, '\A([a-z0-9_]+)=(\S+)\z')
    if (-not $m.Success -or $rec.ContainsKey($m.Groups[1].Value)) { throw 'STOP: malformed run record' }
    $rec[$m.Groups[1].Value] = $m.Groups[2].Value
  }
  foreach ($k in 'run','admin_sid','svc_sid','usr_sid','bin_sha256','wsl_path','wsl_sha256','distro','img_sha256') { if (-not $rec.ContainsKey($k)) { throw "STOP: run record lacks $k" } }
  if ($rec.run -cne $c.RunId -or $rec.distro -cne $c.Distro) { throw 'STOP: run record is for another run' }
  return $rec
}

# Protected admin journal: $Ev must be Administrators-owned and protected before it is read.
function Get-D2AdminFacts {
  $c = $script:C
  Assert-D2Obj $c.Ev 'dir' $script:BA $true @("$($script:SY)|$($script:FA)|3", "$($script:BA)|$($script:FA)|3") | Out-Null
  $r = Read-D2Safe "$($c.Ev)\journal-admin.txt" @($script:BA)
  $e = ConvertFrom-D2Journal $r.Bytes 'admin' $c.Sid $c.RunId $c.Ev
  $one = { param($p) $v = Get-D2Notes $e $p; if ($v.Length -ne 1) { throw "STOP: admin journal has $($v.Length) '$p' notes" }; $v[0] }
  $acct = @{}
  foreach ($v in (Get-D2Notes $e 'account-sid ')) {
    $m = [regex]::Match($v, '\A(d2(?:svc|usr)-[0-9a-f]{6}) (S-1-5-21-[0-9-]+)\z')
    if (-not $m.Success -or $acct.ContainsKey($m.Groups[1].Value)) { throw 'STOP: malformed or duplicate account-sid note' }
    $acct[$m.Groups[1].Value] = $m.Groups[2].Value
  }
  return [pscustomobject]@{ Entries = $e; AdminSid = (& $one 'admin-sid '); Acct = $acct }
}

function Initialize-D2Role {
  param([Parameter(Mandatory = $true)][string]$RunId,
        [Parameter(Mandatory = $true)][ValidateSet('admin','build','svc','usr')][string]$Role,
        [string]$ExpectSid = '', [hashtable]$Auth = @{})
  if ($RunId -cnotmatch '\A[0-9a-f]{6}\z') { throw 'STOP: RunId must be 6 lowercase hex chars' }
  $me = [Security.Principal.WindowsIdentity]::GetCurrent()
  $elev = ([Security.Principal.WindowsPrincipal]$me).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
  $inAdmins = @($me.Groups | ForEach-Object { $_.Value }) -ccontains $script:BA
  $c = [pscustomobject]@{
    RunId = $RunId; Role = $Role; Sid = $me.User.Value; Name = $me.Name; Auth = $Auth
    Ev = "C:\D2Evidence-$RunId"; W = "C:\D2Build-$RunId"; Base = "C:\ProgramData\D2Acc-$RunId"; Bin = "C:\ProgramData\D2AccBin-$RunId"
    SvcN = "d2svc-$RunId"; UsrN = "d2usr-$RunId"; Distro = "d2acc-$RunId"; Desc = "D2 P0 disposable acceptance $RunId - delete after test"
    RoleDir = $null; J = $null; LogDir = $null; Cwd = $null; Record = $null; H = $null; SvcSid = $null; UsrSid = $null; AdminSid = $null
  }
  $script:C = $c
  switch ($Role) {
    'admin' { if (-not $elev) { throw 'STOP: admin role needs an ELEVATED window' }; $c.RoleDir = $c.Ev }
    'build' { if ($elev -or $c.Sid -cne $ExpectSid) { throw 'STOP: build must be the F2 administrator in a NON-elevated window' }; $c.RoleDir = $c.W }
    default {
      $n = $c.UsrN; if ($Role -ceq 'svc') { $n = $c.SvcN }
      if ($elev -or $inAdmins -or $me.Name -notlike "*\$n" -or $c.Sid -cne $ExpectSid) { throw "STOP: this window is not the standard account $n" }
      $c.RoleDir = "$([Environment]::GetFolderPath('UserProfile'))\d2-$RunId"
    }
  }
  $c.LogDir = "$($c.RoleDir)\logs"
  $c.Cwd = $c.RoleDir
  foreach ($p in $c.Ev, $c.W, $c.Base, $c.Bin, $c.RoleDir) { if ($p -match '\s') { throw "STOP: whitespace in $p" } }
  if ($Role -ceq 'admin') {
    if (Test-Path -LiteralPath "$($c.Ev)\journal-admin.txt") {
      $f = Get-D2AdminFacts
      if ($f.AdminSid -cne $c.Sid) { throw 'STOP: this is not the F2 administrator' }
      $c.AdminSid = $f.AdminSid
      if ($f.Acct.ContainsKey($c.SvcN)) { $c.SvcSid = $f.Acct[$c.SvcN] }
      if ($f.Acct.ContainsKey($c.UsrN)) { $c.UsrSid = $f.Acct[$c.UsrN] }
    }
  } else {
    $needRecord = ($Role -cne 'build')
    if ($needRecord -or (Test-Path -LiteralPath "$($c.Bin)\run-record.txt")) {
      $rec = Get-D2RunRecord
      $want = @{ build = $rec.admin_sid; svc = $rec.svc_sid; usr = $rec.usr_sid }[$Role]
      if ($c.Sid -cne $want) { throw "STOP: caller $($c.Sid) is not the run record's $Role SID" }
      $c.Record = $rec; $c.AdminSid = $rec.admin_sid; $c.SvcSid = $rec.svc_sid; $c.UsrSid = $rec.usr_sid
    }
  }
  if (Test-Path -LiteralPath "$($c.RoleDir)\journal-$Role.txt") {
    $c.J = "$($c.RoleDir)\journal-$Role.txt"
    $owners = @($c.Sid); if ($Role -ceq 'admin') { $owners = @($script:BA) }
    $null = ConvertFrom-D2Journal (Read-D2Safe $c.J $owners).Bytes $Role $c.Sid $RunId $c.RoleDir
  }
  "D2 context: run=$RunId role=$Role sid=$($c.Sid) roledir=$($c.RoleDir)"
}

function Set-D2Hash([string]$Recorded) {
  $c = $script:C
  if ($Recorded -cnotmatch '\A[0-9A-F]{64}\z') { throw 'STOP: the recorded build hash must be 64 uppercase hex chars' }
  if ($c.Record -and $c.Record.bin_sha256 -cne $Recorded) { throw 'STOP: typed build hash differs from the run record' }
  $c.H = $Recorded
}

function Assert-D2Auth([string[]]$Need) {
  foreach ($k in $Need) {
    $v = $script:C.Auth[$k]
    if (-not ($v -is [string]) -or $v -eq '' -or $v.Contains('<')) { throw "STOP: owner input $k is not set" }
  }
}

# ---- process launch ------------------------------------------------------------------------
function Invoke-D2([string]$Exe, [string[]]$Argv, [string]$Tag, [string]$Cwd = $null) {
  $c = $script:C
  if (-not (Test-Path -LiteralPath $c.LogDir -PathType Container)) { throw "STOP: LogDir $($c.LogDir) missing" }
  if ($Tag -cnotmatch '\A[A-Za-z0-9_.-]+\z') { throw "STOP: tag $Tag" }
  foreach ($x in $Argv) { if ($x -match '[\s"]') { throw "STOP: argument with space/quote: $x" } }
  if (-not $Cwd) { $Cwd = $c.Cwd }
  Assert-D2Chain $Cwd
  $out = "$($c.LogDir)\$Tag.out.txt"; $err = "$($c.LogDir)\$Tag.err.txt"; $ex = "$($c.LogDir)\$Tag.exit.txt"
  foreach ($f in $out, $err, $ex) { if (Test-Path -LiteralPath $f) { throw "STOP: tag $Tag already used" } }
  $sp = @{ FilePath = $Exe; WorkingDirectory = $Cwd; NoNewWindow = $true; PassThru = $true; RedirectStandardOutput = $out; RedirectStandardError = $err }
  if ($Argv.Count -gt 0) { $sp.ArgumentList = $Argv }
  try { $p = Start-Process @sp } catch { Write-D2Bytes $ex ($script:Utf8.GetBytes("BLOCKED launch: $($_.Exception.Message)")); return 'BLOCKED' }
  $null = $p.Handle; $p.WaitForExit()
  $code = $p.ExitCode
  if ($null -eq $code) { Write-D2Bytes $ex ($script:Utf8.GetBytes('UNKNOWN')); return 'UNKNOWN' }
  Write-D2Bytes $ex ($script:Utf8.GetBytes("$code`r`n"))
  return [int]$code
}
function Run-D2Mut([string]$What, [string]$Exe, [string[]]$Argv, [string]$Tag, [string]$Cwd = $null) { JBegin $What; Need0 (Invoke-D2 $Exe $Argv $Tag $Cwd) $What; JDone $What }

function Read-D2OwnLog([string]$Tag) {
  $c = $script:C; $o = @($c.Sid); if ($c.Role -ceq 'admin') { $o = @($script:BA) }
  $g = { param($s) $p = "$($c.LogDir)\$Tag.$s.txt"; if (Test-Path -LiteralPath $p) { (Read-D2Safe $p $o).Bytes } else { $null } }
  $x = & $g 'exit'
  $xt = $null; if ($null -ne $x) { $xt = $script:Utf8.GetString($x) }
  return [pscustomobject]@{ Out = (& $g 'out'); Err = (& $g 'err'); Exit = $xt }
}

# ---- WSL launcher (Go-equivalent resolution) and registration ------------------------------
function Assert-D2Wsl {
  $c = $script:C
  if ($c.Role -cne 'usr' -or -not $c.Record) { throw 'STOP: WSL only in the bound usr window' }
  $auth = $c.Record.wsl_path
  $r = Read-D2Safe $auth @($script:TI, $script:SY, $script:BA) 64MB
  if ($r.Sha256 -cne $c.Record.wsl_sha256) { throw 'STOP/BLOCKED: wsl.exe digest differs from the run record' }
  $sig = Get-AuthenticodeSignature -LiteralPath $auth
  if ($sig.Status -ne 'Valid' -or $sig.SignerCertificate.Subject -notmatch 'O=Microsoft Corporation') { throw 'STOP/BLOCKED: wsl.exe signature' }
  $env:PATH = "$(Split-Path -Parent $auth);$env:SystemRoot\System32;$env:SystemRoot"
  $env:NoDefaultCurrentDirectoryInExePath = '1'
  $probe = { param($x) if (Test-Path -LiteralPath $x -PathType Leaf) { 'FILE' } elseif (Test-Path -LiteralPath $x -PathType Container) { 'DIR' } else { 'ABSENT' } }
  # Resolve as Go would, but also with the current directory considered, so nothing there can shadow it.
  $res = Resolve-D2GoLookPath 'wsl.exe' $env:PATH $env:PATHEXT $c.Cwd $false $probe
  if ($res.Dot -or $res.Path -ine $auth) { throw "STOP/BLOCKED: the harness would resolve wsl.exe to $($res.Path)" }
  $env:WSL_UTF8 = '1'
}
function Get-D2WslListing([string]$Tag) {
  $null = Invoke-D2 $script:C.Record.wsl_path @('-l', '-q') $Tag
  $l = Read-D2OwnLog $Tag
  return ConvertFrom-D2WslListing $l.Exit $l.Out $l.Err
}
function Get-D2WslRegistration {
  $c = $script:C; $k = 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Lxss'
  try {
    if (-not (Test-Path -LiteralPath $k)) { return 'ABSENT' }
    $m = @(Get-ChildItem -LiteralPath $k | Where-Object { $_.GetValue('DistributionName') -ceq $c.Distro })
    if ($m.Count -eq 0) { return 'ABSENT' }
    if ($m.Count -gt 1) { return 'MISMATCH' }
    $bp = [string]$m[0].GetValue('BasePath'); if ($bp.StartsWith('\\?\')) { $bp = $bp.Substring(4) }
    if ($bp.TrimEnd('\') -ieq "$($c.RoleDir)\distro") { return 'MATCH' }
    return 'MISMATCH'
  } catch { return 'UNKNOWN' }
}

# ---- fixture ----------------------------------------------------------------------------------
function Get-D2FixtureSpec {
  $c = $script:C; $sy = $script:SY; $ba = $script:BA; $bu = $script:BU; $fa = $script:FA; $rx = $script:RX; $s = $c.SvcSid
  if (-not $s) { throw 'STOP: service SID unknown' }
  $spec = New-Object System.Collections.Generic.List[object]
  $add = { param($p, $t, $prot, $aces, $content) $spec.Add([pscustomobject]@{ Path = $p; Type = $t; Prot = $prot; Aces = [string[]]$aces; Content = $content }) }
  & $add $c.Base 'dir' $true @("$sy|$fa|3", "$ba|$fa|3", "$bu|$rx|0") $null
  & $add "$($c.Base)\.d2-run" 'file' $false @("$sy|$fa|16", "$ba|$fa|16") $c.RunId
  foreach ($k in 'admin', 'entity') {
    $root = "$($c.Base)\$k"
    & $add $root 'dir' $true @("$sy|$fa|3", "$ba|$fa|3", "$s|$fa|3") $null
    if ($k -ceq 'admin') { foreach ($d in 'logs', 'runtime') { & $add "$root\$d" 'dir' $false @("$sy|$fa|19", "$ba|$fa|19", "$s|$fa|19") $null } }
    foreach ($f in (Get-D2AuthorityFiles $k)) { & $add ("$root\" + $f.Replace('/', '\')) 'file' $false @("$sy|$fa|16", "$ba|$fa|16", "$s|$fa|16") $script:Dummy }
  }
  $binAces = @("$sy|$fa|16", "$ba|$fa|16", "$bu|$rx|16")
  & $add $c.Bin 'dir' $true @("$sy|$fa|3", "$ba|$fa|3", "$bu|$rx|3") $null
  & $add "$($c.Bin)\.d2-run" 'file' $false $binAces $c.RunId
  & $add "$($c.Bin)\boundary.test.exe" 'file' $false $binAces '<exe>'
  & $add "$($c.Bin)\boundary.test.exe.sha256" 'file' $false $binAces $c.H
  & $add "$($c.Bin)\run-record.txt" 'file' $false $binAces '<record>'
  $pkgDir = @("$sy|$fa|19", "$ba|$fa|19", "$bu|$rx|19")
  & $add "$($c.Bin)\pkg" 'dir' $false $pkgDir $null
  & $add "$($c.Bin)\pkg\tests" 'dir' $false $pkgDir $null
  foreach ($rel in $script:PkgFiles) { & $add "$($c.Bin)\pkg\$rel" 'file' $false $binAces '<pkg>' }
  return ,($spec.ToArray())
}

# Exact inventory, then every entry by handle: type, links, owner, ACL model, exact content.
function Assert-D2Fixture {
  $c = $script:C
  if (-not $c.H) { throw 'STOP: Set-D2Hash first' }
  $spec = Get-D2FixtureSpec
  # Roots first (chain, type, owner, ACL), so the recursive enumeration below only walks verified,
  # Administrators-owned, protected trees.
  foreach ($e in @($spec | Where-Object { $_.Path -cin $c.Base, $c.Bin, "$($c.Base)\admin", "$($c.Base)\entity" })) { $null = Assert-D2Obj $e.Path $e.Type $script:BA $e.Prot $e.Aces }
  foreach ($top in $c.Base, $c.Bin) {
    $want = @($spec | Where-Object { $_.Path.StartsWith("$top\", [StringComparison]::Ordinal) } | ForEach-Object { $_.Path } | Sort-Object)
    $got = @(Get-ChildItem -LiteralPath $top -Recurse -Force | ForEach-Object { $_.FullName } | Sort-Object)
    if (($got -join '|') -cne ($want -join '|')) { throw "STOP: inventory of $top`n  got      $($got -join ', ')`n  expected $($want -join ', ')" }
  }
  foreach ($e in $spec) {
    $null = Assert-D2Obj $e.Path $e.Type $script:BA $e.Prot $e.Aces
    if ($e.Type -ceq 'file') {
      $r = Read-D2Safe $e.Path @($script:BA) 256MB
      switch -CaseSensitive ($e.Content) {
        '<exe>'    { if ($r.Sha256 -cne $c.H) { throw "STOP: binary digest $($r.Sha256)" } }
        '<pkg>'    { $rel = $e.Path.Substring("$($c.Bin)\pkg\".Length)
                     if ((Get-D2Notes (Get-D2AdminFacts).Entries "pkg $rel sha256=$($r.Sha256)").Length -ne 1) { throw "STOP: published package file $rel differs from its journaled digest" } }
        '<record>' { $ref = Read-D2Safe "$($c.Ev)\run-record.txt" @($script:BA) 64KB
                     if ($r.Sha256 -cne $ref.Sha256) { throw 'STOP: published run record differs from the admin copy' } }
        default    { if ($script:Utf8.GetString($r.Bytes) -cne $e.Content) { throw "STOP: unexpected content in $($e.Path)" } }
      }
    }
  }
}

# ---- F2 ---------------------------------------------------------------------------------------
function Invoke-D2StepAdminPrestate {
  $c = $script:C
  if ($c.Role -cne 'admin') { throw 'STOP: admin window only' }
  Assert-D2Auth @('VmName')
  $cs = Get-CimInstance Win32_ComputerSystem
  if ($cs.Model -notmatch 'Virtual') { throw "STOP: model '$($cs.Model)' is not a VM" }
  if ($env:COMPUTERNAME -cne $c.Auth.VmName) { throw 'STOP: not the authorized VM' }
  foreach ($p in $c.Ev, $c.W, $c.Base, $c.Bin) { if (Test-Path -LiteralPath $p) { throw "STOP: $p exists" } }
  if (@(Get-LocalUser | Where-Object { $_.Name -match '\Ad2(svc|usr)-' }).Count) { throw 'STOP: leftover d2 accounts' }
  if (@(Get-ChildItem -LiteralPath 'C:\ProgramData' -Force -Filter 'D2Acc*').Count -or @(Get-ChildItem -LiteralPath 'C:\' -Force -Filter 'D2*').Count) { throw 'STOP: leftover D2 directories' }
  New-D2RoleDir "O:BAD:P(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)"
  $null = Assert-D2Obj $c.Ev 'dir' $script:BA $true @("$($script:SY)|$($script:FA)|3", "$($script:BA)|$($script:FA)|3")
  foreach ($p in $c.Ev, $c.W, $c.Base, $c.Bin) { JNote "prestate-absent $p" }
  foreach ($n in $c.SvcN, $c.UsrN) { JNote "prestate-absent account $n" }
  JNote "admin-sid $($c.Sid)"
  New-D2File "$($c.Ev)\vm-identity.txt" "model=$($cs.Model) name=$env:COMPUTERNAME serial=$((Get-CimInstance Win32_BIOS).SerialNumber) run=$($c.RunId)"
  New-D2File "$($c.Ev)\pre-users.csv" ((Get-LocalUser | Select-Object Name, @{n='SID';e={$_.SID.Value}}, Enabled, Description | ConvertTo-Csv -NoTypeInformation) -join "`n")
  New-D2File "$($c.Ev)\pre-groups.csv" ((Get-D2GroupRows | ConvertTo-Csv -NoTypeInformation) -join "`n")
  New-D2File "$($c.Ev)\pre-profiles.csv" ((Get-CimInstance Win32_UserProfile | Select-Object SID, LocalPath | ConvertTo-Csv -NoTypeInformation) -join "`n")
  New-D2File "$($c.Ev)\pre-sddl.txt" (Get-D2HostSddl)
  JBegin "mkdir $($c.W)"
  JNote "delegate mkdir $($c.W) role=build sid=$($c.Sid)"
  "ADMIN SID (use as -ExpectSid in the build window): $($c.Sid)"
}
function Get-D2GroupRows {
  foreach ($g in Get-LocalGroup) { Get-LocalGroupMember -SID $g.SID | ForEach-Object { [pscustomobject]@{ Group = $g.SID.Value; Member = $_.SID.Value } } }
}
function Get-D2HostSddl {
  $lines = foreach ($p in 'C:\', 'C:\ProgramData') {
    $h = [D2Native]::Open($p, ([D2Native]::READ_CONTROL -bor [D2Native]::FILE_READ_ATTRIBUTES), $true)
    try { "$p $([D2Native]::Sddl($h))" } finally { $h.Dispose() }
  }
  return ($lines -join "`n")
}

# ---- F3 ---------------------------------------------------------------------------------------
function Invoke-D2StepBuild {
  $c = $script:C
  if ($c.Role -cne 'build') { throw 'STOP: build window only' }
  Assert-D2Auth @('Git', 'Go')
  New-D2RoleDir "D:P(A;OICI;FA;;;$($c.Sid))(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)"
  $null = Assert-D2Obj $c.W 'dir' $c.Sid $true @("$($c.Sid)|$($script:FA)|3", "$($script:SY)|$($script:FA)|3", "$($script:BA)|$($script:FA)|3")
  foreach ($t in $c.Auth.Git, $c.Auth.Go) {
    $r = Read-D2Safe $t @($script:TI, $script:SY, $script:BA) 256MB
    JNote "tool $t sha256=$($r.Sha256) sig=$((Get-AuthenticodeSignature -LiteralPath $t).Status)"
  }
  Get-ChildItem Env: | Where-Object { $_.Name -like 'IDENTITY_D2_*' } | ForEach-Object { Remove-Item -LiteralPath "Env:$($_.Name)" }
  if (@(Get-ChildItem Env: | Where-Object { $_.Name -like 'IDENTITY_D2_*' }).Count) { throw 'STOP: acceptance variables still set' }
  Remove-Item -LiteralPath Env:GOROOT -ErrorAction SilentlyContinue
  $env:GOENV = 'off'; $env:GOCACHE = "$($c.W)\gocache"; $env:GOPATH = "$($c.W)\gopath"; $env:GOMODCACHE = "$($c.W)\gopath\pkg\mod"; $env:GOTOOLCHAIN = 'local'; $env:GOFLAGS = '-mod=readonly'
  $src = "$($c.W)\identity"; $art = "$($c.W)\boundary.test.exe"
  Assert-D2Scope $src; Assert-D2Scope $art
  Run-D2Mut 'git clone' $c.Auth.Git @('clone', '--branch', 'experiment/d2-protected-profile-boundary-20261120', 'https://github.com/Arameair/identity.git', $src) 'git-clone' $c.W
  Run-D2Mut "git checkout $($script:SHA)" $c.Auth.Git @('-C', $src, '-c', 'advice.detachedHead=false', 'checkout', '--detach', $script:SHA) 'git-checkout' $c.W
  Need0 (Invoke-D2 $c.Auth.Git @('-C', $src, 'rev-parse', 'HEAD') 'git-head' $c.W) 'rev-parse'
  if ($script:Utf8.GetString((Read-D2OwnLog 'git-head').Out).Trim() -cne $script:SHA) { throw 'STOP: wrong head' }
  Need0 (Invoke-D2 $c.Auth.Git @('-C', $src, 'status', '--porcelain') 'git-status' $c.W) 'status'
  if ($script:Utf8.GetString((Read-D2OwnLog 'git-status').Out).Trim()) { throw 'STOP: dirty checkout' }
  Need0 (Invoke-D2 $c.Auth.Go @('version') 'go-version' $src) 'go version'
  Need0 (Invoke-D2 $c.Auth.Go @('vet', './internal/profile/...') 'go-vet' $src) 'go vet'
  Need0 (Invoke-D2 $c.Auth.Go @('test', '-count=1', './internal/profile/boundary') 'go-test' $src) 'go test'
  Need0 (Invoke-D2 $c.Auth.Go @('test', '-count=1', '-v', '-run', '^TestAcceptanceDisposableProtectedProfile$', './internal/profile/boundary') 'go-test-skip' $src) 'go test (skip check)'
  if ($script:Utf8.GetString((Read-D2OwnLog 'go-test-skip').Out) -cnotmatch '(?m)^\s*--- SKIP: TestAcceptanceDisposableProtectedProfile ') { throw 'STOP: acceptance harness did not SKIP in the unit run' }
  Run-D2Mut "go test -c -> $art" $c.Auth.Go @('test', '-c', '-o', $art, './internal/profile/boundary') 'go-build' $src
  $h = (Read-D2Safe $art @($c.Sid) 256MB).Sha256
  New-D2File "$art.sha256" $h
  "BUILD HASH (write it down; typed into Set-D2Hash in F4, F5 and C2): $h"
}

# ---- F4 ---------------------------------------------------------------------------------------
function Assert-D2Account([string]$N, [string]$Sid) {
  $c = $script:C
  $u = Get-LocalUser -Name $N
  if ($u.SID.Value -cne $Sid -or $u.Description -cne $c.Desc) { throw "STOP: $N is not this run's account" }
  foreach ($g in Get-LocalGroup) {
    $m = @(Get-LocalGroupMember -SID $g.SID | ForEach-Object { $_.SID.Value })
    if (($m -ccontains $Sid) -ne ($g.SID.Value -ceq $script:BU)) { throw "STOP: $N group membership differs from Users-only ($($g.Name))" }
  }
}
function New-D2Account([string]$N) {
  $c = $script:C; $f = Get-D2AdminFacts
  if ($N -cnotin $c.SvcN, $c.UsrN) { throw "STOP: $N is not a run account" }
  if ((Get-D2Notes $f.Entries "prestate-absent account $N").Length -ne 1) { throw "STOP: no pre-state for $N" }
  if (Get-LocalUser -Name $N -ErrorAction SilentlyContinue) { throw "STOP: $N exists" }
  $pw = Read-Host -AsSecureString "New password for $N"
  JBegin "account $N"; New-LocalUser -Name $N -Password $pw -Description $c.Desc -AccountExpires (Get-Date).AddHours(24) | Out-Null; JDone "account $N"
  JNote "account-sid $N $((Get-LocalUser -Name $N).SID.Value)"
  JBegin "group-users $N"; Add-LocalGroupMember -SID $script:BU -Member $N; JDone "group-users $N"
}
function Invoke-D2StepFixture([string]$BuildHash) {
  $c = $script:C
  if ($c.Role -cne 'admin') { throw 'STOP: admin window only' }
  Assert-D2Auth @('Wsl', 'ImgSha')
  Set-D2Hash $BuildHash
  $f = Get-D2AdminFacts
  New-D2Account $c.SvcN; New-D2Account $c.UsrN
  $f = Get-D2AdminFacts; $c.SvcSid = $f.Acct[$c.SvcN]; $c.UsrSid = $f.Acct[$c.UsrN]; $c.AdminSid = $f.AdminSid
  Assert-D2Account $c.SvcN $c.SvcSid; Assert-D2Account $c.UsrN $c.UsrSid
  # The build artifact is read once through a verified handle; that same byte array is hashed and published.
  $art = Read-D2Safe "$($c.W)\boundary.test.exe" @($c.AdminSid) 256MB
  $artSha = Read-D2Safe "$($c.W)\boundary.test.exe.sha256" @($c.AdminSid) 1KB
  if ($script:Utf8.GetString($artSha.Bytes) -cne $c.H -or $art.Sha256 -cne $c.H) { throw 'STOP: build artifact or its .sha256 differs from the recorded build hash' }
  $wsl = Read-D2Safe $c.Auth.Wsl @($script:TI, $script:SY, $script:BA) 64MB
  $sig = Get-AuthenticodeSignature -LiteralPath $c.Auth.Wsl
  if ($sig.Status -ne 'Valid' -or $sig.SignerCertificate.Subject -notmatch 'O=Microsoft Corporation') { throw 'STOP: wsl.exe signature' }
  if ($c.Auth.ImgSha -cnotmatch '\A[0-9A-F]{64}\z') { throw 'STOP: image SHA-256 must be 64 uppercase hex chars' }
  $record = (@("run=$($c.RunId)", "admin_sid=$($c.AdminSid)", "svc_sid=$($c.SvcSid)", "usr_sid=$($c.UsrSid)", "bin_sha256=$($c.H)",
    "wsl_path=$($c.Auth.Wsl)", "wsl_sha256=$($wsl.Sha256)", "distro=$($c.Distro)", "img_sha256=$($c.Auth.ImgSha)") -join "`n") + "`n"
  New-D2File "$($c.Ev)\run-record.txt" $record
  $sa = "O:BAD:P(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)"
  New-D2Dir $c.Base "$sa(A;;0x1200a9;;;BU)"; New-D2File "$($c.Base)\.d2-run" $c.RunId
  New-D2Dir $c.Bin "$sa(A;OICI;0x1200a9;;;BU)"; New-D2File "$($c.Bin)\.d2-run" $c.RunId
  foreach ($k in 'admin', 'entity') {
    New-D2Dir "$($c.Base)\$k" "$sa(A;OICI;FA;;;$($c.SvcSid))"
    if ($k -ceq 'admin') { New-D2Dir "$($c.Base)\admin\logs"; New-D2Dir "$($c.Base)\admin\runtime" }
    foreach ($x in (Get-D2AuthorityFiles $k)) { New-D2File ("$($c.Base)\$k\" + $x.Replace('/', '\')) $script:Dummy }
  }
  New-D2File "$($c.Bin)\boundary.test.exe" $art.Bytes
  New-D2File "$($c.Bin)\boundary.test.exe.sha256" $c.H
  New-D2File "$($c.Bin)\run-record.txt" $record
  JNote "run-record sha256=$(Get-D2Sha ($script:Utf8.GetBytes($record)))"
  # Publish this exact package (as imported by this window) for the svc/usr windows; they import it
  # from here because they can read but not modify it. Compare these digests with Test-D2Package output.
  New-D2Dir "$($c.Bin)\pkg"; New-D2Dir "$($c.Bin)\pkg\tests"
  foreach ($rel in $script:PkgFiles) {
    $bytes = [IO.File]::ReadAllBytes((Join-Path $PSScriptRoot $rel))
    New-D2File "$($c.Bin)\pkg\$rel" $bytes
    JNote "pkg $rel sha256=$(Get-D2Sha $bytes)"
  }
  Assert-D2Fixture
  JNote 'F4 fixture verified'
  "SVC SID: $($c.SvcSid)`nUSR SID: $($c.UsrSid)"
}

# ---- F5 ---------------------------------------------------------------------------------------
function Invoke-D2StepAccess([string]$BuildHash) {
  $c = $script:C
  if ($c.Role -cnotin 'svc', 'usr') { throw 'STOP: svc or usr window only' }
  Set-D2Hash $BuildHash
  New-D2RoleDir
  $c.Cwd = $c.Bin
  if ((Read-D2Safe "$($c.Bin)\boundary.test.exe" @($script:BA) 256MB).Sha256 -cne $c.H) { throw 'STOP: binary digest' }
  $r = Invoke-D2 "$($c.Bin)\boundary.test.exe" @('-test.list', '^TestAcceptanceDisposableProtectedProfile$') "$($c.Role)-list"
  if (-not ($r -is [int] -and $r -eq 0) -or $script:Utf8.GetString((Read-D2OwnLog "$($c.Role)-list").Out).Trim() -cne 'TestAcceptanceDisposableProtectedProfile') { throw "STOP/BLOCKED: test binary not runnable as $($c.Role) ($r)" }
  JNote "$($c.Role) access proven"
}

function Invoke-D2StepWslPreflight {
  $c = $script:C
  if ($c.Role -cne 'usr') { throw 'STOP: usr window only' }
  Assert-D2Auth @('ImgUrl', 'ImgName')
  try {
    $c.Cwd = $c.Bin
    Assert-D2Wsl; JNote "wsl=$($c.Record.wsl_path) sha256=$($c.Record.wsl_sha256)"
    $pre = Get-D2WslListing 'wsl-list-pre'
    if ($pre.State -cne 'KNOWN' -or $pre.Names.Length -ne 0) { throw "pre-import distro state is not a recognized empty listing ($($pre.Reason))" }
    if ((Get-D2WslRegistration) -cne 'ABSENT') { throw 'registry already holds the run distro' }
    JNote 'wsl-pre-empty'
    if ($c.Auth.ImgName -cnotmatch '\A[A-Za-z0-9._-]+\z') { throw 'image file name' }
    $img = "$($c.RoleDir)\$($c.Auth.ImgName)"; Assert-D2Scope $img
    JBegin "download $img"; Invoke-WebRequest -Uri $c.Auth.ImgUrl -OutFile $img -UseBasicParsing; JDone "download $img"
    if ((Get-FileHash -LiteralPath $img -Algorithm SHA256).Hash -cne $c.Record.img_sha256) { throw 'image hash differs from the run record' }
    Run-D2Mut "wsl import $($c.Distro)" $c.Record.wsl_path @('--import', $c.Distro, "$($c.RoleDir)\distro", $img, '--version', '2') 'wsl-import'
    Run-D2Mut "wsl set-default $($c.Distro)" $c.Record.wsl_path @('--set-default', $c.Distro) 'wsl-default'
    $post = Get-D2WslListing 'wsl-list-post'
    if ($post.State -cne 'KNOWN' -or $post.Names.Length -ne 1 -or $post.Names[0] -cne $c.Distro) { throw "expected exactly $($c.Distro) after import ($($post.Reason))" }
    if ((Get-D2WslRegistration) -cne 'MATCH') { throw 'registration not positively identified after import' }
    $w = $c.Record.wsl_path
    Need0 (Invoke-D2 $w @('-d', $c.Distro, '-e', 'dd', '--version') 'wsl-dd') 'dd --version'
    Need0 (Invoke-D2 $w @('-d', $c.Distro, '-e', 'cat', '/proc/self/mountinfo') 'wsl-mountinfo') 'mountinfo'
    $rc = Invoke-D2 $w @('-d', $c.Distro, '-e', 'cat', '/etc/wsl.conf') 'wsl-conf'
    if ($script:Utf8.GetString((Read-D2OwnLog 'wsl-dd').Out) -notmatch 'GNU coreutils') { throw 'not GNU dd' }
    if (-not ($script:Utf8.GetString((Read-D2OwnLog 'wsl-mountinfo').Out) -match '(?m) /mnt/c .* - (9p|drvfs) ')) { throw 'no 9p/drvfs /mnt/c mount' }
    if ($rc -is [int] -and $rc -eq 0 -and $script:Utf8.GetString((Read-D2OwnLog 'wsl-conf').Out) -match '(?im)^\s*root\s*=') { throw 'custom automount root' }
    JNote 'WSL-PREFLIGHT READY'
  } catch {
    JNote "WSL-PREFLIGHT BLOCKED $($_.Exception.Message)"
    Write-Host 'WSL diagnostic BLOCKED; continue with native runs. F8 WSL cleanup still runs.' -ForegroundColor Yellow
  }
}

# ---- F6 ---------------------------------------------------------------------------------------
function Invoke-D2Run([string]$Tag) {
  $c = $script:C; $p = Get-D2PlanRow $Tag
  if ($p.Role -cne $c.Role) { throw "STOP: $Tag runs in the $($p.Role) window" }
  if (-not $c.Record -or -not $c.H) { throw 'STOP: run record and Set-D2Hash are required' }
  $c.Cwd = $c.Bin
  if ($p.Mode -ceq 'wsl') {
    $pf = Get-D2Notes (ConvertFrom-D2Journal (Read-D2Safe $c.J @($c.Sid)).Bytes $c.Role $c.Sid $c.RunId $c.RoleDir) 'WSL-PREFLIGHT '
    if ($pf.Length -ne 1 -or $pf[0] -cne 'READY') { throw 'STOP: WSL runs need exactly one WSL-PREFLIGHT note, and it must be READY' }
    Assert-D2Wsl
  }
  $exe = "$($c.Bin)\boundary.test.exe"
  if ((Read-D2Safe $exe @($script:BA) 256MB).Sha256 -cne $c.H) { throw 'STOP: binary changed' }
  $root = "$($c.Base)\$($p.Kind)"
  $normal = $c.Record.usr_sid; if ($p.NormalSidOf -ceq 'admin') { $normal = $c.Record.admin_sid }
  Need0 (Invoke-D2 "$env:SystemRoot\System32\whoami.exe" @('/all') "$Tag-whoami") 'whoami'
  $clear = { Get-ChildItem Env: | Where-Object { $_.Name -like 'IDENTITY_D2_*' } | ForEach-Object { Remove-Item -LiteralPath "Env:$($_.Name)" } }
  & $clear
  try {
    $env:IDENTITY_D2_DISPOSABLE = 'I-UNDERSTAND-THIS-IS-A-DISPOSABLE-TEST-PROFILE'
    $env:IDENTITY_D2_ACCEPTANCE = $p.Mode; $env:IDENTITY_D2_KIND = $p.Kind; $env:IDENTITY_D2_ROOT = $root
    $env:IDENTITY_D2_SERVICE_SID = $c.Record.svc_sid; $env:IDENTITY_D2_NORMAL_SID = $normal
    $null = Invoke-D2 $exe @('-test.run', '^TestAcceptanceDisposableProtectedProfile$', '-test.v', '-test.count=1') $Tag
  } finally { & $clear }
  $l = Read-D2OwnLog $Tag
  $res = ConvertFrom-D2HarnessRun -Stdout $l.Out -Stderr $l.Err -ExitText $l.Exit -Run @{ Mode = $p.Mode; Kind = $p.Kind; Root = $root; CallerSid = $c.Sid; Distro = $c.Distro }
  JNote "RESULT $Tag => $($res.Result) codes=$($res.Codes -join ',')"
  if ($res.Reason) { JNote "RESULT-REASON $Tag $($res.Reason)" }
  $met = ($res.Result -ceq $p.Want) -and ($p.Code -eq '' -or $res.Codes -ccontains $p.Code)
  JNote ("EXPECT $Tag want=$($p.Want)/$($p.Code) got=$($res.Result)/$($res.Codes -join ',') => " + $(if ($met) { 'MET' } else { 'NOT-MET' }))
  return $res
}

# ---- F7 (C3) ------------------------------------------------------------------------------------
# Shared restoration through an already-verified handle. Never returns normally on failure.
function Restore-D2C3Handle($H, [string]$PreSddl, [string]$BoundId, [string]$Target) {
  $i = [D2Native]::GetInfo($H)
  if ($i.Id -cne $BoundId -or $i.IsReparse -or -not $i.IsDir) { throw "object identity changed ($($i.Id), bound $BoundId)" }
  if ([D2Native]::FinalPath($H) -ine $Target) { throw 'final path changed' }
  [D2Native]::SetDacl($H, $PreSddl)
  $pre = ConvertFrom-D2Sddl $PreSddl; $now = ConvertFrom-D2Sddl ([D2Native]::Sddl($H))
  if ($now.Owner -cne $pre.Owner -or $now.Protected -ne $pre.Protected -or ($now.Aces -join ';') -cne ($pre.Aces -join ';')) { throw 'restored DACL differs from the bound pre-mutation DACL' }
}
function Invoke-D2StepC3Weaken {
  $c = $script:C
  if ($c.Role -cne 'admin') { throw 'STOP: admin window only' }
  if (-not $c.H) { throw 'STOP: Set-D2Hash first' }
  if ((Get-D2C3State (Get-D2AdminFacts).Entries) -cne 'NONE') { throw 'STOP: C3 already started in this run' }
  Assert-D2Fixture
  $t = "$($c.Base)\entity"
  Assert-D2Chain $t
  $h = [D2Native]::Open($t, ([D2Native]::READ_CONTROL -bor [D2Native]::WRITE_DAC -bor [D2Native]::FILE_READ_ATTRIBUTES), $true)
  try {
    $i = [D2Native]::GetInfo($h)
    if ($i.IsReparse -or -not $i.IsDir -or [D2Native]::FinalPath($h) -ine $t) { throw 'STOP: entity root is not the verified directory' }
    $pre = [D2Native]::Sddl($h)
    Test-D2AclModel $pre $script:BA $true @("$($script:SY)|$($script:FA)|3", "$($script:BA)|$($script:FA)|3", "$($c.SvcSid)|$($script:FA)|3")
    New-D2File "$($c.Ev)\c3-pre.sddl" $pre
    JNote "C3 BIND id=$($i.Id) pre-sha256=$(Get-D2Sha ($script:Utf8.GetBytes($pre)))"
    JNote "C3 PENDING-RESTORE id=$($i.Id)"            # write-ahead: restoration is owed from here on
    try {
      JBegin 'C3 grant'
      [D2Native]::SetDacl($h, (Add-D2SddlAllowAce $pre $c.UsrSid $script:RX 3))
      $now = ConvertFrom-D2Sddl ([D2Native]::Sddl($h))
      if (@($now.Aces | Where-Object { $_ -ceq "$($c.UsrSid)|$($script:RX)|3" }).Count -ne 1) { throw 'grant not observed' }
      JDone 'C3 grant'
      JNote 'C3 weakened'
    } catch {
      $why = $_.Exception.Message
      JNote "C3 GRANT FAILED $why"
      try { Restore-D2C3Handle $h $pre $i.Id $t; Assert-D2Fixture; JNote 'C3 revert verified' }
      catch { JNote "C3 REVERT FAILED $($_.Exception.Message)"; Write-Host 'C3 restore FAILED: stop all runs; use the disposable-VM recovery path.' -ForegroundColor Red }
      throw "STOP: C3 grant failed: $why"
    }
  } finally { $h.Dispose() }
}
function Invoke-D2StepC3Restore {
  $c = $script:C
  if ($c.Role -cne 'admin') { throw 'STOP: admin window only' }
  try {
    if (-not $c.H) { throw 'Set-D2Hash first' }
    $f = Get-D2AdminFacts
    $st = Get-D2C3State $f.Entries
    if ($st -cnotin 'PENDING', 'WEAKENED') { throw "C3 state is $st" }
    $bid = @(Get-D2Notes $f.Entries 'C3 PENDING-RESTORE id=')
    if ($bid.Count -ne 1) { throw 'no unique bound object id' }
    $pre = $script:Utf8.GetString((Read-D2Safe "$($c.Ev)\c3-pre.sddl" @($script:BA) 64KB).Bytes)
    if ((Get-D2Notes $f.Entries "C3 BIND id=$($bid[0]) pre-sha256=$(Get-D2Sha ($script:Utf8.GetBytes($pre)))").Length -ne 1) { throw 'saved pre-mutation DACL does not match the bind record' }
    $t = "$($c.Base)\entity"
    Assert-D2Chain $t
    $h = [D2Native]::Open($t, ([D2Native]::READ_CONTROL -bor [D2Native]::WRITE_DAC -bor [D2Native]::FILE_READ_ATTRIBUTES), $true)
    try { JBegin 'C3 revert'; Restore-D2C3Handle $h $pre $bid[0] $t; JDone 'C3 revert' } finally { $h.Dispose() }
    Assert-D2Fixture
    JNote 'C3 revert verified'
    'C3 RESTORED AND VERIFIED: the restored runs (S-E2, N-E2, W-E2 if READY) may now run.'
  } catch {
    try { JNote "C3 REVERT FAILED $($_.Exception.Message)" } catch { Write-Host 'journal write also failed' -ForegroundColor Red }
    Write-Host 'C3 restore FAILED: block all further runs; use the disposable-VM recovery path (revert to CP0 / discard).' -ForegroundColor Red
    throw
  }
}

# ---- F8 step 1 (usr): WSL cleanup ------------------------------------------------------------
function Invoke-D2StepWslCleanup {
  $c = $script:C
  if ($c.Role -cne 'usr') { throw 'STOP: usr window only' }
  $c.Cwd = $c.Bin
  try { Assert-D2Wsl } catch { JNote "RESIDUAL WSL launcher could not be verified for cleanup: $($_.Exception.Message); registration of $($c.Distro) unresolved"; throw }
  $e = ConvertFrom-D2Journal (Read-D2Safe $c.J @($c.Sid)).Bytes $c.Role $c.Sid $c.RunId $c.RoleDir
  $preEmpty = $false
  try {
    $pl = Read-D2OwnLog 'wsl-list-pre'
    $pre = ConvertFrom-D2WslListing $pl.Exit $pl.Out $pl.Err
    $preEmpty = ($pre.State -ceq 'KNOWN' -and $pre.Names.Length -eq 0 -and (Get-D2Notes $e 'wsl-pre-empty').Length -eq 1)
  } catch { $preEmpty = $false }
  $imported = @($e | Where-Object { $_.Kind -ceq 'BEGIN' -and $_.Text -ceq "wsl import $($c.Distro)" }).Count -eq 1
  $unk = [pscustomobject]@{ State = 'UNKNOWN'; Names = [string[]]@(); Reason = 'listing failed' }
  $l1 = $unk; try { $l1 = Get-D2WslListing 'wsl-list-clean1' } catch { JNote "wsl-list-clean1 failed: $($_.Exception.Message)" }
  $d = Get-D2WslCleanupDecision @{ Listing = $l1; Distro = $c.Distro; PreEmpty = $preEmpty; ImportBegin = $imported; Registry = (Get-D2WslRegistration) }
  JNote "wsl-cleanup-decision $($d.Action) $($d.Reason)"
  if ($d.Action -ceq 'UNREGISTER') {
    try { Run-D2Mut "wsl terminate $($c.Distro)" $c.Record.wsl_path @('--terminate', $c.Distro) 'wsl-terminate' } catch { JNote "terminate-failed $($_.Exception.Message)" }
    try { Run-D2Mut "wsl unregister $($c.Distro)" $c.Record.wsl_path @('--unregister', $c.Distro) 'wsl-unregister' } catch { JNote "RESIDUAL unregister of $($c.Distro) failed: $($_.Exception.Message)" }
  } elseif ($d.Action -ceq 'RESIDUAL') { JNote "RESIDUAL WSL $($c.Distro): $($d.Reason)" }
  $l2 = $unk; try { $l2 = Get-D2WslListing 'wsl-list-clean2' } catch { JNote "wsl-list-clean2 failed: $($_.Exception.Message)" }
  if ($l2.State -cne 'KNOWN') { JNote "RESIDUAL WSL listing after cleanup not recognized ($($l2.Reason)); registration unresolved" }
  foreach ($n in $l2.Names) { JNote "RESIDUAL distro still registered (left alone unless it is $($c.Distro)): $n" }
  $reg = Get-D2WslRegistration
  if ($reg -cne 'ABSENT') { JNote "RESIDUAL registry state of $($c.Distro) after cleanup: $reg" }
  'WSL cleanup recorded. Sign out of this account; never run --unregister or --shutdown in any other account.'
}

# ---- F8a (admin): evidence collection ----------------------------------------------------------
function Copy-D2Evidence([string]$Role, [string]$Src, [string]$Sid) {
  $c = $script:C; $o = @($Sid)
  try {
    $dst = "$($c.Ev)\collected-$Role"; New-D2Dir $dst; New-D2Dir "$dst\logs"
    $m = Read-D2Safe "$Src\.d2-run" $o 6
    if ([Text.Encoding]::ASCII.GetString($m.Bytes) -cne $c.RunId) { throw 'marker mismatch' }
    $j = Read-D2Safe "$Src\journal-$Role.txt" $o 4MB
    $null = ConvertFrom-D2Journal $j.Bytes $Role $Sid $c.RunId $Src
    New-D2File "$dst\journal-$Role.txt" $j.Bytes; JNote "collected $Role journal-$Role.txt sha256=$($j.Sha256)"
    if ($Role -ceq 'build') {
      $s = Read-D2Safe "$Src\boundary.test.exe.sha256" $o 1KB
      New-D2File "$dst\boundary.test.exe.sha256" $s.Bytes; JNote "collected $Role boundary.test.exe.sha256 sha256=$($s.Sha256)"
    }
    $null = Assert-D2Chain "$Src\logs"
    $allowed = Get-D2RoleTags $Role; $total = 0; $seen = @{}
    foreach ($full in [IO.Directory]::GetFileSystemEntries("$Src\logs")) {
      $n = [IO.Path]::GetFileName($full)
      $mm = [regex]::Match($n, '\A([A-Za-z0-9_.-]+)\.(out|err|exit)\.txt\z')
      if (-not $mm.Success -or $allowed -cnotcontains $mm.Groups[1].Value) { JNote "INCOMPLETE $Role unexpected log entry $n (not copied)"; continue }
      $r = Read-D2Safe $full $o 16MB
      $total += $r.Size; if ($total -gt 64MB) { throw 'aggregate log size over 64 MB' }
      New-D2File "$dst\logs\$n" $r.Bytes; JNote "collected $Role logs\$n sha256=$($r.Sha256)"
      $seen[$mm.Groups[1].Value] = 1 + $(if ($seen.ContainsKey($mm.Groups[1].Value)) { $seen[$mm.Groups[1].Value] } else { 0 })
    }
    foreach ($k in @($seen.Keys)) { if ($seen[$k] -ne 3) { JNote "INCOMPLETE $Role log triplet $k has $($seen[$k]) of 3 files" } }
    JNote "collected $Role"
  } catch { JNote "INCOMPLETE evidence $Role from ${Src}: $($_.Exception.Message)" }
}
function Invoke-D2StepCollect {
  $c = $script:C
  if ($c.Role -cne 'admin') { throw 'STOP: admin window only' }
  $f = Get-D2AdminFacts
  Copy-D2Evidence 'build' $c.W $f.AdminSid
  foreach ($pair in @(@('svc', $c.SvcN), @('usr', $c.UsrN))) {
    $k = $pair[0]; $n = $pair[1]
    try {
      if (-not $f.Acct.ContainsKey($n)) { throw "no journaled SID for $n" }
      $sid = $f.Acct[$n]
      $pr = @(Get-CimInstance Win32_UserProfile | Where-Object { $_.SID -ceq $sid })
      if ($pr.Count -ne 1 -or $pr[0].LocalPath -cnotmatch ('\AC:\\Users\\' + [regex]::Escape($n) + '(\.[A-Za-z0-9_-]+)?\z')) { throw "profile for $n not found at an expected path" }
      Copy-D2Evidence $k "$($pr[0].LocalPath)\d2-$($c.RunId)" $sid
    } catch { JNote "INCOMPLETE evidence ${k}: $($_.Exception.Message)" }
  }
  'Evidence collection recorded; run Invoke-D2StepCleanup next.'
}

# ---- F8b (admin): cleanup, with a final status on every path -------------------------------------
function Invoke-D2StepCleanup {
  $c = $script:C
  if ($c.Role -cne 'admin') { throw 'STOP: admin window only' }
  $problems = New-Object System.Collections.Generic.List[string]
  $done = @{ objects = $false; post = $false; journals = $false }
  try {
    $f = Get-D2AdminFacts; $e = $f.Entries
    $one = { param($kind, $text) @($e | Where-Object { $_.Kind -ceq $kind -and $_.Text -ceq $text }).Count -eq 1 }
    $buildJ = $null
    try { $buildJ = ConvertFrom-D2Journal (Read-D2Safe "$($c.Ev)\collected-build\journal-build.txt" @($script:BA)).Bytes 'build' $f.AdminSid $c.RunId $c.W } catch { $problems.Add("INCOMPLETE build journal unusable: $($_.Exception.Message)") }
    foreach ($spec in @(@($c.Base, $script:BA, 'admin'), @($c.Bin, $script:BA, 'admin'), @($c.W, $f.AdminSid, 'build'))) {
      $p = $spec[0]; $own = $spec[1]
      try {
        $fx = @{ Exists = 'UNKNOWN' }
        if (Test-Path -LiteralPath $p) { $fx.Exists = 'YES' } else { $fx.Exists = 'NO' }
        $fx.PrestateAbsent = ((Get-D2Notes $e "prestate-absent $p").Length -eq 1)
        if ($spec[2] -ceq 'admin') { $fx.CreateProven = (& $one 'BEGIN' "mkdir $p") -and (& $one 'DONE' "mkdir $p") }
        else {
          $fx.CreateProven = (& $one 'BEGIN' "mkdir $p") -and ((Get-D2Notes $e "delegate mkdir $p role=build sid=$($f.AdminSid)").Length -eq 1) -and
            ($null -ne $buildJ) -and (@($buildJ | Where-Object { $_.Kind -ceq 'DONE' -and $_.Text -ceq "mkdir $p" }).Count -eq 1)
        }
        if ($fx.Exists -ceq 'YES') {
          try { Assert-D2Chain $p; $fx.ChainOk = $true } catch { $fx.ChainOk = $false }
          try { $fx.MarkerOk = ([Text.Encoding]::ASCII.GetString((Read-D2Safe "$p\.d2-run" @($own) 6).Bytes) -ceq $c.RunId) } catch { $fx.MarkerOk = $false }
          try { $h = [D2Native]::Open($p, ([D2Native]::READ_CONTROL -bor [D2Native]::FILE_READ_ATTRIBUTES), $true); try { $fx.OwnerOk = ((ConvertFrom-D2Sddl ([D2Native]::Sddl($h))).Owner -ceq $own) -and [D2Native]::GetInfo($h).IsDir } finally { $h.Dispose() } } catch { $fx.OwnerOk = $false }
          try { $fx.NoReparseInside = (@(Get-ChildItem -LiteralPath $p -Recurse -Force -Attributes ReparsePoint).Count -eq 0) } catch { $fx.NoReparseInside = $false }
        }
        $d = Get-D2DirCleanupDecision $fx
        if ($d.Action -ceq 'REMOVE') {
          JBegin "remove $p"; Remove-Item -LiteralPath $p -Recurse -Force; JDone "remove $p"
          if (Test-Path -LiteralPath $p) { throw 'still present after removal' }
        } elseif ($d.Action -ceq 'RESIDUAL') { JNote "RESIDUAL ${p}: $($d.Reason)" } else { JNote "absent $p" }
      } catch { $problems.Add("RESIDUAL ${p}: $($_.Exception.Message)"); try { JNote "RESIDUAL ${p}: $($_.Exception.Message)" } catch { } }
    }
    $preUsers = $null
    try { $preUsers = @(ConvertFrom-Csv -InputObject (ConvertFrom-D2Utf8 (Read-D2Safe "$($c.Ev)\pre-users.csv" @($script:BA)).Bytes)) } catch { $problems.Add("INCOMPLETE pre-users.csv unusable: $($_.Exception.Message)") }
    foreach ($n in $c.SvcN, $c.UsrN) {
      try {
        $fx = @{ Exists = 'UNKNOWN' }
        $u = $null; $prof = @()
        $all = @(Get-LocalUser | Where-Object { $_.Name -ceq $n })
        if ($all.Count -eq 1) { $u = $all[0]; $fx.Exists = 'YES' } elseif ($all.Count -eq 0) { $fx.Exists = 'NO' }
        if ($fx.Exists -ceq 'YES') {
          $fx.ActualSid = $u.SID.Value
          $fx.JournaledSid = $null; if ($f.Acct.ContainsKey($n)) { $fx.JournaledSid = $f.Acct[$n] }
          $fx.PrestateAbsent = ((Get-D2Notes $e "prestate-absent account $n").Length -eq 1)
          $fx.BeginDone = (& $one 'BEGIN' "account $n") -and (& $one 'DONE' "account $n")
          $fx.PreCsvLacks = ($null -ne $preUsers) -and (@($preUsers | Where-Object { $_.Name -ceq $n }).Count -eq 0)
          $fx.DescriptionOk = ($u.Description -ceq $c.Desc)
          try { if ($fx.JournaledSid) { Assert-D2Account $n $fx.JournaledSid }; $fx.GroupsOk = [bool]$fx.JournaledSid } catch { $fx.GroupsOk = $false }
          $prof = @(Get-CimInstance Win32_UserProfile | Where-Object { $_.SID -ceq $fx.ActualSid })
          $fx.ProfileOk = ($prof.Count -le 1) -and (@($prof | Where-Object { $_.LocalPath -cnotmatch ('\AC:\\Users\\' + [regex]::Escape($n) + '(\.[A-Za-z0-9_-]+)?\z') }).Count -eq 0)
          $fx.ProfileLoaded = (@($prof | Where-Object { $_.Loaded }).Count -ne 0)
        }
        $d = Get-D2AccountCleanupDecision $fx
        if ($d.Action -ceq 'REMOVE') {
          foreach ($pp in $prof) { JBegin "remove profile $n"; $pp | Remove-CimInstance; JDone "remove profile $n" }
          JBegin "remove account $n"; Remove-LocalUser -SID $fx.JournaledSid; JDone "remove account $n"
        } elseif ($d.Action -ceq 'RESIDUAL') { JNote "RESIDUAL account ${n}: $($d.Reason)" } else { JNote "absent account $n" }
      } catch { $problems.Add("RESIDUAL account ${n}: $($_.Exception.Message)"); try { JNote "RESIDUAL account ${n}: $($_.Exception.Message)" } catch { } }
    }
    $done.objects = $true
  } catch { $problems.Add("INCOMPLETE cleanup aborted: $($_.Exception.Message)") }
  try {
    $cmp = {
      param($name, [string]$now)
      $pre = ConvertFrom-D2Utf8 (Read-D2Safe "$($c.Ev)\pre-$name" @($script:BA)).Bytes
      New-D2File "$($c.Ev)\post-$name" $now
      if ($pre -cne $now) { $problems.Add("RESIDUAL $name differs from pre-state (do NOT repair)") }
    }
    & $cmp 'users.csv' ((Get-LocalUser | Select-Object Name, @{n='SID';e={$_.SID.Value}}, Enabled, Description | ConvertTo-Csv -NoTypeInformation) -join "`n")
    & $cmp 'groups.csv' ((Get-D2GroupRows | ConvertTo-Csv -NoTypeInformation) -join "`n")
    & $cmp 'profiles.csv' ((Get-CimInstance Win32_UserProfile | Select-Object SID, LocalPath | ConvertTo-Csv -NoTypeInformation) -join "`n")
    & $cmp 'sddl.txt' (Get-D2HostSddl)
    $done.post = $true
  } catch { $problems.Add("INCOMPLETE post-state not established: $($_.Exception.Message)") }
  try {
    $f = Get-D2AdminFacts
    $js = @{ admin = $f.Entries }; $sids = @{ build = $f.AdminSid }
    foreach ($pair in @(@('build', $c.W, $f.AdminSid), @('svc', $null, $null), @('usr', $null, $null))) {
      $r = $pair[0]
      try {
        $raw = (Read-D2Safe "$($c.Ev)\collected-$r\journal-$r.txt" @($script:BA)).Bytes
        $sid = $pair[2]; $dir = $pair[1]
        if ($r -cne 'build') {
          $n = $c.UsrN; if ($r -ceq 'svc') { $n = $c.SvcN }
          $sid = $f.Acct[$n]; $sids[$r] = $sid
          $first = (Split-D2Lines (ConvertFrom-D2Utf8 $raw) -AllowCrLf)[0]
          $mm = [regex]::Match($first, ' NOTE roledir-identity (C:\\Users\\' + [regex]::Escape($n) + '(?:\.[A-Za-z0-9_-]+)?\\d2-' + $c.RunId + ') role=')
          if (-not $mm.Success) { throw 'identity line does not name the account profile' }
          $dir = $mm.Groups[1].Value
        }
        $js[$r] = ConvertFrom-D2Journal $raw $r $sid $c.RunId $dir
      } catch { $problems.Add("INCOMPLETE journal-$r not collected or unusable (its residuals are unknown): $($_.Exception.Message)") }
    }
    foreach ($x in (Test-D2JournalBalance $js $sids)) { $problems.Add("INCOMPLETE $x") }
    foreach ($r in @($js.Keys)) { foreach ($en in $js[$r]) { if ($en.Kind -ceq 'NOTE' -and $en.Text -cmatch '\A(RESIDUAL|INCOMPLETE|C3 REVERT FAILED)') { $problems.Add("journal-${r}: $($en.Text)") } } }
    $st = Get-D2C3State $f.Entries
    if ($st -cnotin 'NONE', 'RESTORED') { $problems.Add("C3 state ${st}: restoration not verified; discard the VM") }
    $done.journals = $true
  } catch { $problems.Add("INCOMPLETE journal reconciliation failed: $($_.Exception.Message)") }
  finally {
    foreach ($k in @($done.Keys)) { if (-not $done[$k]) { $problems.Add("INCOMPLETE section '$k' did not complete") } }
    if ($problems.Count -eq 0) { 'CLEANUP: CLEAN (all role journals collected and balanced; pre/post state identical)' }
    else { "CLEANUP: INCOMPLETE ($($problems.Count))"; $problems }
    'On Option A, revert to CP0 or discard the VM regardless of this status. $Ev and the package folder are never deleted by script.'
  }
}

# ---- F9 (admin): scoring from collected evidence -----------------------------------------------
function Invoke-D2StepScore {
  $c = $script:C
  if ($c.Role -cne 'admin') { throw 'STOP: admin window only' }
  $f = Get-D2AdminFacts; $e = $f.Entries
  $collected = @{}
  foreach ($v in (Get-D2Notes $e 'collected ')) { $m = [regex]::Match($v, '\A(build|svc|usr) (\S+) sha256=([0-9A-F]{64})\z'); if ($m.Success) { $collected["$($m.Groups[1].Value)\$($m.Groups[2].Value)"] = $m.Groups[3].Value } }
  $sidOf = @{ build = $f.AdminSid; svc = $f.Acct[$c.SvcN]; usr = $f.Acct[$c.UsrN] }
  $incomplete = New-Object System.Collections.Generic.List[string]
  foreach ($v in (Get-D2Notes $e 'INCOMPLETE ')) { $incomplete.Add($v) }
  $journals = @{}
  foreach ($r in 'build', 'svc', 'usr') {
    try {
      $raw = Read-D2Safe "$($c.Ev)\collected-$r\journal-$r.txt" @($script:BA)
      if ($collected["$r\journal-$r.txt"] -cne $raw.Sha256) { throw 'collected copy differs from the collection record' }
      $first = (Split-D2Lines (ConvertFrom-D2Utf8 $raw.Bytes) -AllowCrLf)[0]
      $dir = [regex]::Match($first, ' NOTE roledir-identity (\S+) role=').Groups[1].Value
      $journals[$r] = ConvertFrom-D2Journal $raw.Bytes $r $sidOf[$r] $c.RunId $dir
    } catch { $incomplete.Add("journal-${r}: $($_.Exception.Message)") }
  }
  $evidence = @{}; $claims = @{}
  foreach ($p in (Get-D2Plan)) {
    $r = $p.Role
    if ($journals.ContainsKey($r)) {
      $cl = New-Object System.Collections.Generic.List[object]
      foreach ($en in $journals[$r]) {
        $m = [regex]::Match($en.Text, '\ARESULT ' + [regex]::Escape($p.Tag) + ' => (\S+) codes=(\S*)\z')
        if ($en.Kind -ceq 'NOTE' -and $m.Success) { $cl.Add([pscustomobject]@{ Result = $m.Groups[1].Value; Ts = $en.Ts }) }
      }
      $claims[$p.Tag] = $cl.ToArray()
    }
    $base = "$($c.Ev)\collected-$r\logs\$($p.Tag)"
    $have = @('out', 'err', 'exit' | Where-Object { Test-Path -LiteralPath "$base.$_.txt" })
    if ($have.Count -eq 0) { continue }
    try {
      if ($have.Count -ne 3) { throw 'incomplete triplet' }
      $b = @{}
      foreach ($s in 'out', 'err', 'exit') {
        $x = Read-D2Safe "$base.$s.txt" @($script:BA)
        if ($collected["$r\logs\$($p.Tag).$s.txt"] -cne $x.Sha256) { throw "collected $s differs from the collection record" }
        $b[$s] = $x.Bytes
      }
      $res = ConvertFrom-D2HarnessRun -Stdout $b.out -Stderr $b.err -ExitText ($script:Utf8.GetString($b.exit)) -Run @{ Mode = $p.Mode; Kind = $p.Kind; Root = "$($c.Base)\$($p.Kind)"; CallerSid = $sidOf[$r]; Distro = $c.Distro }
    } catch { $res = [pscustomobject]@{ Result = 'PROCEDURE-ERROR'; Reason = $_.Exception.Message; Verdict = $null; Codes = [string[]]@() } }
    $evidence[$p.Tag] = @($res)
  }
  $ts = { param($t) $x = @($e | Where-Object { $_.Kind -ceq 'NOTE' -and $_.Text -ceq $t }); if ($x.Count -eq 1) { $x[0].Ts } else { $null } }
  $pf = 'MISSING'
  if ($journals.ContainsKey('usr')) {
    $n = Get-D2Notes $journals['usr'] 'WSL-PREFLIGHT '
    if ($n.Length -gt 1) { $pf = 'CONTRADICTORY' } elseif ($n.Length -eq 1) { if ($n[0] -ceq 'READY') { $pf = 'READY' } else { $pf = 'BLOCKED' } }
  }
  $admin = @{ FixtureVerified = ((Get-D2Notes $e 'F4 fixture verified').Length -eq 1); C3State = (Get-D2C3State $e)
              C3WeakenedTs = (& $ts 'C3 weakened'); C3RestoredTs = (& $ts 'C3 revert verified'); Incomplete = $incomplete.ToArray() }
  $s = Get-D2Score -Evidence $evidence -Claims $claims -Admin $admin -Preflight $pf
  $s.Rows | Format-Table Tag, Diag, Phase, Want, Status, Result, Why -AutoSize | Out-String -Width 220
  foreach ($k in $s.Labels.Keys) { "$k : $($s.Labels[$k])"; JNote "SCORE $k = $($s.Labels[$k])" }
  if ($s.C3Block) { $s.C3Block }
  if ($incomplete.Count) { 'Evidence problems:'; $incomplete }
  'D2 remains OPEN. A diagnostic PASS is never "D2 accepted", "WSL isolated" or "secure".'
}

Export-ModuleMember -Function Initialize-D2Role, Set-D2Hash, Invoke-D2StepAdminPrestate, Invoke-D2StepBuild, Invoke-D2StepFixture,
  Invoke-D2StepAccess, Invoke-D2StepWslPreflight, Invoke-D2Run, Invoke-D2StepC3Weaken, Invoke-D2StepC3Restore,
  Invoke-D2StepWslCleanup, Invoke-D2StepCollect, Invoke-D2StepCleanup, Invoke-D2StepScore
