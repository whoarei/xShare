#!/usr/bin/env bash
# Synchronize version from package.json to Cargo.toml and tauri.conf.json.
# Usage:
#   ./scripts/bump-version.sh          # read from package.json
#   ./scripts/bump-version.sh 1.2.0    # set explicit version
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"

# --- Read version from package.json if not provided ---
VERSION="${1:-}"
if [ -z "$VERSION" ]; then
    VERSION=$(grep -oP '"version"\s*:\s*"\K[^"]+' "$ROOT/package.json")
fi

# Validate semver
if ! echo "$VERSION" | grep -qE '^[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.]+)?(\+[a-zA-Z0-9.]+)?$'; then
    echo "Error: Invalid semver version: $VERSION" >&2
    exit 1
fi

echo "Version: $VERSION"

# --- Update package.json (use node for reliable JSON handling) ---
if command -v node &>/dev/null; then
    node -e "
const fs = require('fs');
const p = '$ROOT/package.json';
const pkg = JSON.parse(fs.readFileSync(p, 'utf8'));
pkg.version = '$VERSION';
fs.writeFileSync(p, JSON.stringify(pkg, null, 2) + '\n');
"
    echo "[OK] package.json"
else
    # Fallback: sed replacement
    sed -i "s/\"version\": \".*\"/\"version\": \"$VERSION\"/" "$ROOT/package.json"
    echo "[OK] package.json (sed fallback)"
fi

# --- Update Cargo.toml ---
CARGO="$ROOT/src-tauri/Cargo.toml"
if [ -f "$CARGO" ]; then
    sed -i "s/^version = \".*\"/version = \"$VERSION\"/" "$CARGO"
    echo "[OK] src-tauri/Cargo.toml"
fi

# --- Update tauri.conf.json ---
TAURI="$ROOT/src-tauri/tauri.conf.json"
if [ -f "$TAURI" ]; then
    if command -v node &>/dev/null; then
        node -e "
const fs = require('fs');
const p = '$TAURI';
const cfg = JSON.parse(fs.readFileSync(p, 'utf8'));
cfg.version = '$VERSION';
fs.writeFileSync(p, JSON.stringify(cfg, null, 2) + '\n');
"
    else
        sed -i "s/\"version\": \".*\"/\"version\": \"$VERSION\"/" "$TAURI"
    fi
    echo "[OK] src-tauri/tauri.conf.json"
fi

echo ""
echo "All version files updated to $VERSION"

# --- Generate CHANGELOG.md ---
if command -v git-cliff &>/dev/null; then
    echo ""
    echo "Generating CHANGELOG.md..."
    git-cliff --tag "v$VERSION" -o "$ROOT/CHANGELOG.md"
    echo "[OK] CHANGELOG.md"

    echo "Rendering CHANGELOG.html..."
    node "$ROOT/scripts/render-changelog.mjs"
else
    echo "Warning: git-cliff not found. Skipping changelog generation." >&2
    echo "Install it: cargo install git-cliff" >&2
fi
