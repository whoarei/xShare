Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$PLATFORM = if ($args.Count -gt 0) { $args[0] } else { "windows" }
$ARCH = if ($args.Count -gt 1) { $args[1] } else { $env:PROCESSOR_ARCHITECTURE.ToLower() }

$TARGET = "x86_64-pc-windows-msvc"
$EXT = ".exe"

Write-Host "=== 1/3 编译 Go sidecar ($TARGET) ==="
Set-Location go-engine
$env:GOOS = $PLATFORM
$env:GOARCH = $ARCH
go build -o "../src-tauri/binaries/go-engine-$TARGET$EXT" .
Set-Location ..

Write-Host "=== 2/3 安装前端依赖 ==="
npm install --silent

Write-Host "=== 3/3 构建 Tauri 安装包 ==="
npm run tauri build

Write-Host "=== 完成 ==="
Get-ChildItem -Path "src-tauri/target/release/bundle/" -Recurse -File -ErrorAction SilentlyContinue | Select-Object -First 10
