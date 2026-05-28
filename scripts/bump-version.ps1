<#
.SYNOPSIS
    Synchronize version from package.json to Cargo.toml and tauri.conf.json.
.DESCRIPTION
    Reads the version field from package.json (the single source of truth)
    and updates src-tauri/Cargo.toml and src-tauri/tauri.conf.json to match.
.PARAMETER Version
    Optional version string. If omitted, reads from package.json.
.EXAMPLE
    .\scripts\bump-version.ps1
    .\scripts\bump-version.ps1 -Version 1.2.0
#>
param(
    [string]$Version
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot

# --- Read version from package.json if not provided ---
$pkgPath = Join-Path $root "package.json"
if (-not (Test-Path $pkgPath)) {
    Write-Error "package.json not found at $pkgPath"
    exit 1
}

$pkg = Get-Content $pkgPath -Raw | ConvertFrom-Json
if (-not $Version) {
    $Version = $pkg.version
}

# Validate semver pattern
if ($Version -notmatch '^\d+\.\d+\.\d+(-[\w.]+)?(\+[\w.]+)?$') {
    Write-Error "Invalid semver version: $Version"
    exit 1
}

Write-Host "Version: $Version"

# --- Update package.json ---
$pkg.version = $Version
$pkg | ConvertTo-Json -Depth 10 | Set-Content $pkgPath -Encoding UTF8
Write-Host "[OK] package.json"

# --- Update Cargo.toml ---
$cargoPath = Join-Path $root "src-tauri\Cargo.toml"
if (Test-Path $cargoPath) {
    $cargo = Get-Content $cargoPath -Raw
    $cargo = $cargo -replace '(?m)^version\s*=\s*".*"', "version = `"$Version`""
    Set-Content $cargoPath -Value $cargo -Encoding UTF8
    Write-Host "[OK] src-tauri/Cargo.toml"
}

# --- Update tauri.conf.json ---
$tauriPath = Join-Path $root "src-tauri\tauri.conf.json"
if (Test-Path $tauriPath) {
    $tauri = Get-Content $tauriPath -Raw | ConvertFrom-Json
    $tauri.version = $Version
    $tauri | ConvertTo-Json -Depth 10 | Set-Content $tauriPath -Encoding UTF8
    Write-Host "[OK] src-tauri/tauri.conf.json"
}

Write-Host "`nAll version files updated to $Version"

# --- Generate CHANGELOG.md ---
$cliffInstalled = Get-Command git-cliff -ErrorAction SilentlyContinue
if ($cliffInstalled) {
    Write-Host "`nGenerating CHANGELOG.md..."
    & git-cliff --tag "v$Version" -o (Join-Path $root "CHANGELOG.md")
    Write-Host "[OK] CHANGELOG.md"
} else {
    Write-Warning "git-cliff not found. Skipping changelog generation."
    Write-Host "Install it: cargo install git-cliff"
}
