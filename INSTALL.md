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

## Install Ollama (Optional - for AI Features)

History Viewer can use Ollama for AI-powered command analysis. This is optional but highly recommended.

### macOS

```bash
# Download and install from the official website
curl -fsSL https://ollama.com/install.sh | sh

# Or install via Homebrew
brew install ollama

# Start Ollama service
brew services start ollama

# Or run manually
ollama serve
```

### Linux (Ubuntu/Debian)

```bash
# Install Ollama
curl -fsSL https://ollama.com/install.sh | sh

# Ollama will start automatically as a service
# Check status:
systemctl status ollama

# Or start manually if needed:
ollama serve
```

### Pull a Model

After installing Ollama, download a language model:

```bash
# Pull the default model (llama3.3 - recommended)
ollama pull llama3.3

# Or pull a smaller/faster model
ollama pull llama3.2

# Verify it's working
ollama list
```

### Configure History Viewer

History Viewer will automatically connect to Ollama at `http://localhost:11434` with the `llama3.3` model.

To use a different model or URL, edit `~/.history_viewer.json`:

```json
{
  "ollama_url": "http://localhost:11434",
  "ollama_model": "llama3.3"
}
```

**Note:** History Viewer works fine without Ollama - you just won't have AI analysis features.

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
