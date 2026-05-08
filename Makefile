.PHONY: build-core build-ui build bundle clean test test-all test-core test-ui dev install

# Build Go sidecar into the Tauri sidecar binaries directory
build-core:
	cd core && go build -o ../app/src-tauri/binaries/gman-core-$$(uname -m) ./cmd/gman-server

# Build Svelte frontend
build-ui:
	cd app && pnpm build

# Full build
build: build-core build-ui
	cd app && pnpm tauri build

# Bundle for distribution
bundle:
	cd app && pnpm tauri build --bundles deb,appimage,rpm

# Clean build artifacts
clean:
	rm -rf app/src-tauri/binaries/
	rm -rf app/dist/
	rm -rf app/src-tauri/target/release/

# Run all tests
test-all:
	cd core && go test ./... -count=1
	cd app && pnpm test

# Run Go tests only
test-core:
	cd core && go test ./... -count=1

# Run frontend tests only
test-ui:
	cd app && pnpm test

# Dev mode
dev:
	cd app && pnpm tauri dev

# Install dependencies
install:
	cd app && pnpm install
	cargo install tauri-cli --version "^2"
