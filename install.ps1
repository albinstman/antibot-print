<#
.SYNOPSIS
  antibot installer for Windows.

.DESCRIPTION
  Downloads the prebuilt antibot.exe from the latest release, verifies its
  SHA-256 checksum, installs it, and adds the install directory to your user PATH.

    irm https://raw.githubusercontent.com/albinstman/antibot-print/main/install.ps1 | iex

  Override defaults with environment variables before running:
    $env:ANTIBOT_BIN_DIR = "C:\tools\antibot"   # install location
    $env:ANTIBOT_REF     = "v1.2.3"                     # release tag (default: latest)
    $env:ANTIBOT_REPO    = "owner/name"                 # source repo
#>

$ErrorActionPreference = "Stop"

$repo   = if ($env:ANTIBOT_REPO)    { $env:ANTIBOT_REPO }    else { "albinstman/antibot-print" }
$ref    = if ($env:ANTIBOT_REF)     { $env:ANTIBOT_REF }     else { "latest" }
$binDir = if ($env:ANTIBOT_BIN_DIR) { $env:ANTIBOT_BIN_DIR } else { Join-Path $env:LOCALAPPDATA "antibot" }

function Info($m) { Write-Host "==> $m" -ForegroundColor Cyan }
function Ok($m)   { Write-Host "OK  $m"  -ForegroundColor Green }
function Die($m)  { Write-Host "ERR $m"  -ForegroundColor Red; exit 1 }

# --- pick the asset ----------------------------------------------------------
# Only a windows/amd64 binary is published; ARM64 Windows runs it via emulation.
$arch = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString()
if ($arch -eq "Arm64") {
  Write-Host "note: no native ARM64 build; using amd64 (runs under Windows emulation)." -ForegroundColor Yellow
}
$asset = "antibot-windows-amd64.exe"

if ($ref -eq "latest") {
  $base = "https://github.com/$repo/releases/latest/download"
} else {
  $base = "https://github.com/$repo/releases/download/$ref"
}

$tmp = Join-Path ([System.IO.Path]::GetTempPath()) ("antibot-" + [System.Guid]::NewGuid().ToString("N"))
New-Item -ItemType Directory -Path $tmp -Force | Out-Null
try {
  $exe  = Join-Path $tmp $asset
  $sums = Join-Path $tmp "SHA256SUMS"

  Info "Downloading $asset ($repo@$ref)"
  Invoke-WebRequest -Uri "$base/$asset"   -OutFile $exe  -UseBasicParsing
  Invoke-WebRequest -Uri "$base/SHA256SUMS" -OutFile $sums -UseBasicParsing
  if ((Get-Item $exe).Length -eq 0) { Die "downloaded binary is empty." }

  # --- verify checksum -------------------------------------------------------
  Info "Verifying checksum"
  $actual = (Get-FileHash -Algorithm SHA256 -Path $exe).Hash.ToLower()
  $line = Select-String -Path $sums -Pattern ("[ *]" + [regex]::Escape($asset) + "\s*$") | Select-Object -First 1
  if (-not $line) { Die "no checksum for $asset in SHA256SUMS." }
  $expected = ($line.Line -split '\s+')[0].ToLower()
  if ($actual -ne $expected) { Die "checksum mismatch (expected $expected, got $actual)." }

  # --- self-test, then install ----------------------------------------------
  Info "Verifying it runs"
  $out = "HTTP/1.1 403`r`nSet-Cookie: __cf_bm=x; path=/`r`n`r`n" | & $exe
  if ($out -notcontains "cloudflare") { Die "self-test failed (did not detect a Cloudflare response)." }

  New-Item -ItemType Directory -Path $binDir -Force | Out-Null
  $dest = Join-Path $binDir "antibot.exe"
  Copy-Item -Path $exe -Destination $dest -Force
  Ok ("Installed " + (& $dest --version) + " -> $dest")
}
finally {
  Remove-Item -Recurse -Force $tmp -ErrorAction SilentlyContinue
}

# --- ensure it's on PATH -----------------------------------------------------
$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if (-not $userPath) { $userPath = "" }
if (($userPath -split ';') -notcontains $binDir) {
  $newPath = if ($userPath) { "$userPath;$binDir" } else { $binDir }
  [Environment]::SetEnvironmentVariable("Path", $newPath, "User")
  Write-Host "Added $binDir to your user PATH. Open a new terminal to pick it up." -ForegroundColor Yellow
}

# The command was renamed antibot-print -> antibot, and so was the install dir.
# Flag the old install (binary + lingering PATH entry) so the user can clean it up.
$oldDir = Join-Path $env:LOCALAPPDATA "antibot-print"
if (Test-Path (Join-Path $oldDir "antibot-print.exe")) {
  Write-Host "Note: an old 'antibot-print' install remains at $oldDir; the command is now 'antibot'." -ForegroundColor Yellow
  Write-Host "      Remove it with: Remove-Item -Recurse '$oldDir'  (and drop that folder from your PATH)." -ForegroundColor Yellow
}

Write-Host ""
Write-Host "Try it (PowerShell):" -ForegroundColor Cyan
Write-Host '    curl.exe -isS https://example.com | antibot'
