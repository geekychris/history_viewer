# Installation Guide

Quick installation instructions for History Viewer.

## macOS

### Homebrew (Recommended)

```bash
brew tap geekychris/history-viewer
brew install history-viewer
```

### Pre-built Binary

**Apple Silicon (M1/M2/M3):**
```bash
curl -LO https://github.com/geekychris/history_viewer/releases/latest/download/history_viewer-darwin-arm64.tar.gz
tar -xzf history_viewer-darwin-arm64.tar.gz
sudo mv history_viewer-darwin-arm64 /usr/local/bin/history_viewer
chmod +x /usr/local/bin/history_viewer
```

**Intel:**
```bash
curl -LO https://github.com/geekychris/history_viewer/releases/latest/download/history_viewer-darwin-amd64.tar.gz
tar -xzf history_viewer-darwin-amd64.tar.gz
sudo mv history_viewer-darwin-amd64 /usr/local/bin/history_viewer
chmod +x /usr/local/bin/history_viewer
```

---

## Linux (Ubuntu/Debian)

### Install from Binary

```bash
# Download and install
curl -LO https://github.com/geekychris/history_viewer/releases/latest/download/history_viewer-linux-amd64.tar.gz
tar -xzf history_viewer-linux-amd64.tar.gz
sudo mv history_viewer-linux-amd64 /usr/local/bin/history_viewer
chmod +x /usr/local/bin/history_viewer
```

### Install Dependencies (if running with native UI)

If you want to use the native UI mode (`-ui native`), install these dependencies:

```bash
sudo apt-get update
sudo apt-get install -y gcc libgl1-mesa-dev libxcursor-dev libxrandr-dev libxinerama-dev libxi-dev libxxf86vm-dev
```

**Note:** The web UI (default mode) doesn't require these dependencies.

---

## Verify Installation

```bash
history_viewer -h
```

You should see the help message with available options.

---

## Running

**Web UI (default):**
```bash
history_viewer
```
Then open http://localhost:8080 in your browser.

**Native UI:**
```bash
history_viewer -ui native
```

**Custom options:**
```bash
# Use different port
history_viewer -port 9090

# Use specific history file
history_viewer -history ~/.zsh_history
```

---

## Updating

### macOS (Homebrew)
```bash
brew update
brew upgrade history-viewer
```

### Manual Installation
Download and install the latest version using the same commands above.

---

## Uninstalling

### macOS (Homebrew)
```bash
brew uninstall history-viewer
brew untap geekychris/history-viewer
```

### Manual Installation
```bash
sudo rm /usr/local/bin/history_viewer
```

---

## Additional Information

For more details, see:
- **BUILD.md** - Building from source
- **README.md** - Features and configuration
- **CI_CD_COMPLETE_GUIDE.md** - Complete CI/CD documentation

For releases and binaries: https://github.com/geekychris/history_viewer/releases
