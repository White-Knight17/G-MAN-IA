#!/bin/bash
set -euo pipefail

echo "=== G-MAN v1.0 Build ==="
echo "Building Go sidecar..."
cd "$(dirname "$0")/../core"
go build -o ../app/src-tauri/binaries/gman-core-x86_64-unknown-linux-gnu ./cmd/gman-server
echo "Building Svelte frontend..."
cd ../app
pnpm build
echo "Building Tauri bundle..."
pnpm tauri build
echo "=== Build complete ==="
