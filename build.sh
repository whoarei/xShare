#!/bin/bash
set -e
PLATFORM="${1:-$(go env GOOS)}"
ARCH="${2:-$(go env GOARCH)}"

case "$PLATFORM" in
    linux)   TARGET="x86_64-unknown-linux-gnu";    EXT="" ;;
    windows) TARGET="x86_64-pc-windows-msvc";       EXT=".exe" ;;
    darwin)  TARGET="x86_64-apple-darwin";          EXT="" ;;
    *) echo "Unknown platform: $PLATFORM"; exit 1 ;;
esac

echo "=== 1/3 编译 Go sidecar ($TARGET) ==="
cd go-engine && GOOS="$PLATFORM" GOARCH="$ARCH" go build -o "../src-tauri/binaries/go-engine-$TARGET$EXT" . && cd ..

echo "=== 2/3 安装前端依赖 ==="
npm install --silent

echo "=== 3/3 构建 Tauri 安装包 ==="
npm run tauri build

echo "=== 完成 ==="
ls -lh src-tauri/target/release/bundle/*/ 2>/dev/null || true
