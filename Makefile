.PHONY: all build build-web build-native build-app build-menu install-app install-menu clean deps deps-check deps-ubuntu deps-macos test help

# Binary name
BINARY=history_viewer

# Default target
all: deps build

# Build both web and native binaries
build: build-web build-native

# Build for web UI only (no GUI dependencies needed)
build-web:
	@echo "Building web UI binary..."
	go build -tags web -o $(BINARY) .

# Build with native UI support
# On macOS, suppress duplicate -lobjc library warnings from Fyne dependencies
build-native: deps-check
	@echo "Building with native UI support..."
	@if [ "$$(uname)" = "Darwin" ]; then \
		CGO_LDFLAGS="-Wl,-no_warn_duplicate_libraries" go build -o $(BINARY) .; \
	else \
		go build -o $(BINARY) .; \
	fi

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -f $(BINARY)
	@rm -rf dist/

# Check and install all dependencies
deps: deps-check
	@echo "Installing Go dependencies..."
	go mod download
	go mod tidy

# Check platform-specific dependencies
deps-check:
	@echo "Checking platform dependencies..."
	@if [ "$$(uname)" = "Darwin" ]; then \
		$(MAKE) deps-macos-check; \
	elif [ "$$(uname)" = "Linux" ]; then \
		$(MAKE) deps-ubuntu-check; \
	else \
		echo "Warning: Unsupported platform. Only macOS and Linux are supported."; \
	fi

# Check and install Ubuntu/Debian dependencies
deps-ubuntu-check:
	@echo "Checking Ubuntu/Linux dependencies..."
	@missing=""; \
	if [ "$(ARCH)" = "arm64" ]; then \
		suffix=":arm64"; \
	else \
		suffix=""; \
	fi; \
	for pkg in libxcursor-dev libxrandr-dev libxinerama-dev libxi-dev libgl1-mesa-dev libxxf86vm-dev; do \
		if ! dpkg -l | grep -q "$$pkg$$suffix"; then \
			missing="$$missing $$pkg$$suffix"; \
		fi; \
	done; \
	if [ -n "$$missing" ]; then \
		echo "Missing packages:$$missing"; \
		echo "Installing missing packages..."; \
		$(MAKE) deps-ubuntu ARCH=$(ARCH); \
	else \
		echo "All Ubuntu dependencies are installed."; \
	fi

# Install Ubuntu/Debian dependencies
deps-ubuntu:
	@echo "Installing Ubuntu/Linux dependencies..."
	@if [ "$(ARCH)" = "arm64" ]; then \
		suffix=":arm64"; \
		sudo dpkg --add-architecture arm64; \
	else \
		suffix=""; \
	fi; \
	sudo apt-get update && sudo apt-get install -y \
		libxcursor-dev$$suffix \
		libxrandr-dev$$suffix \
		libxinerama-dev$$suffix \
		libxi-dev$$suffix \
		libgl1-mesa-dev$$suffix \
		libxxf86vm-dev$$suffix \
		pkg-config

# Check macOS dependencies
deps-macos-check:
	@echo "Checking macOS dependencies..."
	@if ! command -v xcode-select >/dev/null 2>&1; then \
		echo "Xcode command line tools not found. Installing..."; \
		xcode-select --install; \
		echo "Please wait for Xcode command line tools installation to complete, then run 'make' again."; \
		exit 1; \
	else \
		echo "Xcode command line tools are installed."; \
	fi

# Install macOS dependencies
deps-macos:
	@echo "Installing macOS dependencies..."
	@if ! command -v xcode-select >/dev/null 2>&1; then \
		echo "Installing Xcode command line tools..."; \
		xcode-select --install; \
		echo "Please complete the installation dialog and run 'make deps-macos' again."; \
	else \
		echo "Xcode command line tools already installed."; \
	fi

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Install the binary to /usr/local/bin
install: build-native
	@echo "Installing $(BINARY) to /usr/local/bin..."
	@sudo cp $(BINARY) /usr/local/bin/
	@echo "Installation complete. Run '$(BINARY)' to start."

# Uninstall the binary from /usr/local/bin
uninstall:
	@echo "Removing $(BINARY) from /usr/local/bin..."
	@sudo rm -f /usr/local/bin/$(BINARY)
	@echo "Uninstall complete."

# Run the web UI
run-web: build-web
	./$(BINARY)

# Run the native UI
run-native: build-native
	./$(BINARY) -ui native

# ---- Wails wrapper + menu bar (adds "native-app" experience) ----

WAILS_BIN ?= $(shell go env GOPATH)/bin/wails
APPS_DIR  ?= $(HOME)/Applications

# Build the Wails wrapper .app.
#
# We deliberately bypass `wails build` and use plain `go build` with the
# wails "desktop,production" build tags. `wails build`'s bindings-generator
# step tends to hang on this project (we don't have any Go->JS bindings —
# the app is a thin child-spawn + WebView redirect). Plain go build with
# the right tags + a hand-assembled .app bundle produces the same result
# in seconds.
build-app:
	@echo "Building History Viewer.app (native wrapper)..."
	@mkdir -p cmd/hv-app/build/bin
	@CGO_LDFLAGS="-framework UniformTypeIdentifiers -Wl,-no_warn_duplicate_libraries" \
		go build -tags "desktop,production" \
		-ldflags "-w -s" \
		-o cmd/hv-app/build/bin/HistoryViewer \
		./cmd/hv-app
	@rm -rf "cmd/hv-app/build/bin/History Viewer.app"
	@mkdir -p "cmd/hv-app/build/bin/History Viewer.app/Contents/MacOS"
	@mkdir -p "cmd/hv-app/build/bin/History Viewer.app/Contents/Resources"
	@mv cmd/hv-app/build/bin/HistoryViewer "cmd/hv-app/build/bin/History Viewer.app/Contents/MacOS/HistoryViewer"
	@cp cmd/hv-app/Info.plist "cmd/hv-app/build/bin/History Viewer.app/Contents/Info.plist"
	@echo "Built cmd/hv-app/build/bin/History Viewer.app"

# Build the menu-bar helper binary + .app bundle.
build-menu:
	@echo "Building History Viewer Menu.app (systray helper)..."
	@go build -o cmd/hv-menu/hv-menu ./cmd/hv-menu
	@rm -rf cmd/hv-menu/HistoryViewerMenu.app
	@mkdir -p "cmd/hv-menu/History Viewer Menu.app/Contents/MacOS"
	@cp cmd/hv-menu/hv-menu "cmd/hv-menu/History Viewer Menu.app/Contents/MacOS/hv-menu"
	@cp cmd/hv-menu/Info.plist "cmd/hv-menu/History Viewer Menu.app/Contents/Info.plist"
	@echo "Built cmd/hv-menu/History Viewer Menu.app"

# Install both .apps into ~/Applications. Kills any running instance first
# so LaunchServices picks up the new bundle. Requires build-native for the
# core binary the .app wrapper depends on.
install-app: build-native build-app
	@mkdir -p $(APPS_DIR)
	@rm -rf "$(APPS_DIR)/History Viewer.app"
	@cp -R "cmd/hv-app/build/bin/History Viewer.app" "$(APPS_DIR)/"
	@echo "Installed $(APPS_DIR)/History Viewer.app"
	@echo "(Depends on '$(BINARY)' being on PATH — run 'make install' to place it in /usr/local/bin.)"

install-menu: build-menu
	@mkdir -p $(APPS_DIR)
	@rm -rf "$(APPS_DIR)/History Viewer Menu.app"
	@cp -R "cmd/hv-menu/History Viewer Menu.app" "$(APPS_DIR)/"
	@echo "Installed $(APPS_DIR)/History Viewer Menu.app"

# Show help
help:
	@echo "Zsh History Viewer - Makefile targets:"
	@echo ""
	@echo "  make                 - Install dependencies and build native binary"
	@echo "  make build           - Build both web and native binaries"
	@echo "  make build-web       - Build web UI binary (no GUI dependencies)"
	@echo "  make build-native    - Build native UI binary (with GUI support)"
	@echo "  make deps            - Install all dependencies"
	@echo "  make deps-check      - Check platform-specific dependencies"
	@echo "  make deps-ubuntu     - Install Ubuntu/Debian GUI dependencies"
	@echo "  make deps-macos      - Install macOS GUI dependencies"
	@echo "  make test            - Run tests"
	@echo "  make clean           - Remove build artifacts"
	@echo "  make install         - Install binary to /usr/local/bin"
	@echo "  make uninstall       - Remove binary from /usr/local/bin"
	@echo "  make run-web         - Build and run web UI"
	@echo "  make run-native      - Build and run native UI"
	@echo "  make help            - Show this help message"
	@echo ""
	@echo "Platform-specific notes:"
	@echo "  macOS: Requires Xcode command line tools"
	@echo "  Linux: Requires X11 development libraries"
