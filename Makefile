.PHONY: build-core build-ui build test clean

# Build Go sidecar
build-core:
	cd core && go build -o gman-server ./cmd/gman-server/

# Build Tauri + Svelte frontend
build-ui:
	cd app && pnpm tauri build

# Full build
build: build-core build-ui

# Run Go tests
test:
	cd core && go test ./... -v -cover

# Run Rust tests
test-rust:
	cd app/src-tauri && cargo test

# Clean build artifacts
clean:
	rm -f core/gman-server
	cd app && pnpm tauri clean 2>/dev/null || true
