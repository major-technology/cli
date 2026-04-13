#Requires -Version 5.1
<#
.SYNOPSIS
    Installs the Major CLI on Windows.
.DESCRIPTION
    Downloads the latest Major CLI release from S3, verifies its checksum,
    extracts the binary to ~/.major/bin, and runs shell integration setup.
.EXAMPLE
    irm https://major-cli-releases.s3.us-west-1.amazonaws.com/install.ps1 | iex
#>

$ErrorActionPreference = 'Stop'

# --- Configuration ---
$Binary      = 'major'
$InstallDir  = Join-Path $env:USERPROFILE '.major\bin'
$S3BucketUrl = 'https://major-cli-releases.s3.us-west-1.amazonaws.com'

# --- Helpers ---
function Write-Step  { param([string]$Msg) Write-Host "  > $Msg" -ForegroundColor Cyan }
function Write-Ok    { param([string]$Msg) Write-Host "  + $Msg" -ForegroundColor Green }
function Write-Fail  { param([string]$Msg) Write-Host "  x $Msg" -ForegroundColor Red }

Write-Host "`nMajor CLI Installer`n" -ForegroundColor White

# --- Detect architecture ---
$Arch = switch ([System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture) {
    'X64'   { 'amd64' }
    'Arm64' { 'arm64' }
    default { Write-Fail "Architecture $_ not supported"; exit 1 }
}

$OS = 'windows'

# --- Get latest version ---
Write-Step 'Finding latest release...'
$Version = (Invoke-RestMethod -Uri "$S3BucketUrl/latest-version").Trim()
if (-not $Version) {
    Write-Fail 'Could not find latest release version'
    exit 1
}

# --- Download ---
$AssetName    = "${Binary}_${Version}_${OS}_${Arch}.zip"
$ChecksumName = "${Binary}_${Version}_checksums.txt"
$DownloadUrl  = "$S3BucketUrl/$Version/$AssetName"
$ChecksumUrl  = "$S3BucketUrl/$Version/$ChecksumName"

Write-Step "Downloading ${Binary} v${Version}..."

$TmpDir = Join-Path ([System.IO.Path]::GetTempPath()) ([System.IO.Path]::GetRandomFileName())
New-Item -ItemType Directory -Path $TmpDir -Force | Out-Null

try {
    Invoke-WebRequest -Uri $DownloadUrl  -OutFile (Join-Path $TmpDir $AssetName)    -UseBasicParsing
    Invoke-WebRequest -Uri $ChecksumUrl  -OutFile (Join-Path $TmpDir 'checksums.txt') -UseBasicParsing
} catch {
    Write-Fail "Failed to download: $_"
    Remove-Item -Recurse -Force $TmpDir
    exit 1
}

# --- Verify checksum ---
Write-Step 'Verifying checksum...'

$Checksums = Get-Content (Join-Path $TmpDir 'checksums.txt')
$Expected  = ($Checksums | Where-Object { $_ -match $AssetName }) -replace '\s+.*$', ''

if (-not $Expected) {
    Write-Fail "Could not find checksum for $AssetName"
    Remove-Item -Recurse -Force $TmpDir
    exit 1
}

$Actual = (Get-FileHash -Path (Join-Path $TmpDir $AssetName) -Algorithm SHA256).Hash.ToLower()

if ($Expected -ne $Actual) {
    Write-Fail 'Checksum verification failed!'
    Write-Host "  Expected: $Expected"
    Write-Host "  Actual:   $Actual"
    Remove-Item -Recurse -Force $TmpDir
    exit 1
}

Write-Ok 'Checksum verified'

# --- Extract and install ---
Write-Step "Installing to $InstallDir..."

Expand-Archive -Path (Join-Path $TmpDir $AssetName) -DestinationPath $TmpDir -Force

New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
Copy-Item -Path (Join-Path $TmpDir "$Binary.exe") -Destination (Join-Path $InstallDir "$Binary.exe") -Force

# Cleanup
Remove-Item -Recurse -Force $TmpDir

# --- Shell integration ---
Write-Step 'Setting up shell integration...'
& (Join-Path $InstallDir "$Binary.exe") install

# --- Verify ---
Write-Step 'Verifying installation...'
$InstalledVersion = & (Join-Path $InstallDir "$Binary.exe") --version 2>&1 | Select-Object -First 1

if ($LASTEXITCODE -ne 0) {
    Write-Fail "Installed binary failed to execute (exit $LASTEXITCODE). Output: $InstalledVersion"
    Write-Host "  Check that $InstallDir is in your PATH and the .exe is not blocked by antivirus."
    exit 1
}

if ($InstalledVersion -notmatch [regex]::Escape($Version)) {
    Write-Fail "Version mismatch. Expected $Version, got: $InstalledVersion"
    exit 1
}

Write-Ok "Successfully installed ${Binary} v${Version}"

Write-Host "`nWelcome to Major!`n" -ForegroundColor Green
Write-Host "Get started by running:`n"
Write-Host "  major user login      Log in to your Major account" -ForegroundColor White
