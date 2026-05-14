#!/bin/bash
echo "=== 环境检查 ==="

check_cmd() {
    if command -v "$1" &> /dev/null; then
        echo "  ✓ $1: $(command -v $1)"
    else
        echo "  ✗ $1: 未安装"
    fi
}

echo "--- 必需 ---"
check_cmd go
check_cmd node
check_cmd npm

echo "--- Tauri 构建 (可选) ---"
check_cmd rustc
check_cmd cargo

echo "--- Go 验证 ---"
go version 2>/dev/null || echo "  > 请安装 Go: https://go.dev/dl/"

echo "--- Node.js 验证 ---"
node --version 2>/dev/null || echo "  > 请安装 Node.js: https://nodejs.org"