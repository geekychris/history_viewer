.PHONY: all build build-web build-native clean deps deps-check deps-ubuntu deps-macos test help

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
