#!/usr/bin/env bash
set -euo pipefail

# G-MAN build script
# Cross-compiles Go sidecar and builds Tauri app

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
TARGET="${1:-linux/amd64}"

echo "==> Building G-MAN v1.0.0 for ${TARGET}"

# Build Go sidecar
echo "==> Building Go sidecar..."
cd "$ROOT/core"
GOOS="${TARGET%/*}" GOARCH="${TARGET#*/}" go build -o "$ROOT/core/gman-server" ./cmd/gman-server/

# Build Tauri app
echo "==> Building Tauri app..."
cd "$ROOT/app"
pnpm tauri build

echo "==> Build complete"
