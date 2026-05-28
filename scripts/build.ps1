<#
.SYNOPSIS
    Build xShare for production (Go sidecar + Tauri desktop app).
.PARAMETER Platform
    Target platform: windows, linux, darwin (default: windows)
.PARAMETER Arch
    Target architecture (default: from env PROCESSOR_ARCHITECTURE)
#>
param(
    [string]$Platform = "windows",
    [string]$Arch = ""
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot

# --- Read version from package.json ---
$pkgPath = Join-Path $root "package.json"
$version = (Get-Content $pkgPath -Raw | ConvertFrom-Json).version

# --- Determine target triple ---
switch ($Platform) {
    "windows" { $Target = "x86_64-pc-windows-msvc"; $Ext = ".exe" }
    "linux"   { $Target = "x86_64-unknown-linux-gnu"; $Ext = "" }
    "darwin"  {
        if ($Arch -eq "arm64") {
            $Target = "aarch64-apple-darwin"; $Ext = ""
        } else {
            $Target = "x86_64-apple-darwin"; $Ext = ""
        }
    }
    default { Write-Error "Unknown platform: $Platform"; exit 1 }
}

Write-Host "=== 1/3 编译 Go sidecar ($TARGET) v$version ==="
Push-Location (Join-Path $root "go-engine")
$env:GOOS = $Platform
$env:GOARCH = if ($Arch) { $Arch } else { "amd64" }
$ldflags = "-X main.version=$version"
go build -ldflags $ldflags -o "../src-tauri/binaries/go-engine-$Target$Ext" .
Pop-Location

Write-Host "=== 2/3 安装前端依赖 ==="
Push-Location $root
npm install --silent

Write-Host "=== 3/3 构建 Tauri 安装包 ==="
npm run tauri build
Pop-Location

Write-Host "=== 完成 ==="
Get-ChildItem -Path (Join-Path $root "src-tauri/target/release/bundle/") -Recurse -File -ErrorAction SilentlyContinue | Select-Object -First 10
