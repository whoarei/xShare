#!/usr/bin/env bash
# Build xShare for production (Go sidecar + Tauri desktop app).
# Usage: ./scripts/build.sh [linux|windows|darwin] [arch]
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"

PLATFORM="${1:-$(go env GOOS)}"
ARCH="${2:-$(go env GOARCH)}"

# --- Read version from package.json ---
VERSION=$(grep -oP '"version"\s*:\s*"\K[^"]+' "$ROOT/package.json")

# --- Determine target triple ---
case "$PLATFORM" in
    linux)   TARGET="x86_64-unknown-linux-gnu";    EXT="" ;;
    windows) TARGET="x86_64-pc-windows-msvc";      EXT=".exe" ;;
    darwin)
        if [ "$ARCH" = "arm64" ]; then
            TARGET="aarch64-apple-darwin";          EXT=""
        else
            TARGET="x86_64-apple-darwin";           EXT=""
        fi
        ;;
    *) echo "Unknown platform: $PLATFORM"; exit 1 ;;
esac

echo "=== 1/3 编译 Go sidecar ($TARGET) v$VERSION ==="
cd "$ROOT/go-engine"
GOOS="$PLATFORM" GOARCH="$ARCH" go build -ldflags "-X main.version=$VERSION" -o "$ROOT/src-tauri/binaries/go-engine-$TARGET$EXT" .
cd "$ROOT"

echo "=== 2/3 安装前端依赖 ==="
npm install --silent

echo "=== 3/3 构建 Tauri 安装包 ==="
npm run tauri build

echo "=== 完成 ==="
ls -lh "$ROOT/src-tauri/target/release/bundle/"*/ 2>/dev/null || true
