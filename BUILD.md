# Build and Run Instructions

## Prerequisites

### Required
- **Go**: Version 1.23 or higher (the project uses Go 1.23.3)
  - Check your version: `go version`
  - Download from: https://go.dev/dl/

### For Native UI (Optional)
If you want to use the native UI mode, you'll need:

**macOS:**
- Xcode command line tools: `xcode-select --install`

**Linux (Ubuntu/Debian):**
```bash
sudo apt-get install gcc libgl1-mesa-dev libxcursor-dev libxrandr-dev libxinerama-dev libxi-dev libxxf86vm-dev
```

Alternatively, install all at once:
```bash
sudo apt-get update && sudo apt-get install -y gcc libgl1-mesa-dev libxcursor-dev libxrandr-dev libxinerama-dev libxi-dev libxxf86vm-dev
```

**Linux (Fedora/RHEL):**
```bash
sudo dnf install gcc libXcursor-devel libXrandr-devel mesa-libGL-devel libXi-devel libXinerama-devel libXxf86vm-devel
```

### Optional (for AI features)
- **Ollama**: For AI-powered command analysis
  - Install from: https://ollama.ai/
  - Pull a model: `ollama pull llama3.3`

### Zsh History Configuration
Ensure your zsh history includes timestamps. Add to your `~/.zshrc`:

```bash
setopt EXTENDED_HISTORY
setopt INC_APPEND_HISTORY
setopt SHARE_HISTORY
HISTFILE=~/.zsh_history
HISTSIZE=50000
SAVEHIST=50000
```

After adding, reload: `source ~/.zshrc`

## Building

### 1. Clone or navigate to the project directory
```bash
cd /path/to/history_viewer
```

### 2. Download dependencies
```bash
go mod download
```

### 3. Build the application

**Standard build:**
```bash
go build -o history_viewer
```

This creates an executable named `history_viewer` in the current directory.

**Note:** On macOS, you may see a harmless linker warning:
```
ld: warning: ignoring duplicate libraries: '-lobjc'
```
This is expected and does not affect functionality. It's related to Fyne's CGo dependencies.

### Build for specific platforms (optional)
```bash
# macOS (Intel)
GOOS=darwin GOARCH=amd64 go build -o history_viewer-darwin-amd64

# macOS (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -o history_viewer-darwin-arm64

# Linux
GOOS=linux GOARCH=amd64 go build -o history_viewer-linux-amd64

# Windows
GOOS=windows GOARCH=amd64 go build -o history_viewer.exe
```

## Running

### Basic Usage

**Web UI (default):**
```bash
./history_viewer
```

The application will:
- Start a web server on `http://localhost:8080`
- Use your default zsh history file (`~/.zsh_history`)
- Use Ollama on `http://localhost:11434` with model `llama3.3`

Open your browser to: **http://localhost:8080**

**Native UI:**
```bash
./history_viewer -ui native
```

This will launch a native desktop application instead of a web server. The native UI provides:
- Cross-platform desktop application (macOS, Linux)
- All the same features as the web UI
- Native file dialogs for exporting
- No browser required

### Command Line Options

```bash
./history_viewer [options]
```

**Available flags:**
- `-ui` - UI mode: 'web' or 'native' (default: web)
- `-port` - Web server port (default: 8080, only applies to web UI)
- `-history` - Path to zsh history file (default: ~/.zsh_history)
- `-config` - Path to config file (default: config.json)

**Examples:**

```bash
# Use native UI
./history_viewer -ui native

# Use a different port (web UI)
./history_viewer -port 9090

# Use native UI with specific history file
./history_viewer -ui native -history /path/to/custom/.zsh_history

# Use a custom config file
./history_viewer -config /path/to/my-config.json
```

### Configuration File

The application automatically creates `~/.history_viewer.json` on first run with default settings:

```json
{
  "history_file": "~/.zsh_history",
  "port": 8080,
  "session_timeout_minutes": 30,
  "ollama_url": "http://localhost:11434",
  "ollama_model": "llama3.3",
  "auto_refresh_seconds": 30,
  "home_dir": "/Users/yourusername"
}
```

**Configuration options:**
- `history_file` - Path to zsh history file (supports ~ expansion)
- `port` - Web server port (only used in web UI mode)
- `session_timeout_minutes` - Minutes of inactivity before starting a new session
- `ollama_url` - Ollama API endpoint for AI features
- `ollama_model` - Ollama model to use (e.g., llama3.3, llama3.2, codellama)
- `auto_refresh_seconds` - How often the UI auto-refreshes
- `home_dir` - User's home directory (auto-detected)

You can edit this file directly or use command-line flags to override settings.

## Verifying the Setup

### 1. Check if the server is running
```bash
curl http://localhost:8080/api/sessions
```

You should see JSON output with your command history sessions.

### 2. Test Ollama integration (optional)
```bash
# Verify Ollama is running
ollama list

# Test Ollama API
curl http://localhost:11434/api/tags
```

### 3. Access the UI
Open http://localhost:8080 in your browser. You should see:
- A command volume chart at the top
- Date range pickers and filters
- Session cards with your command history

## Troubleshooting

### Build Issues

**"Fyne error: Failed to load user locales" (macOS)**
This is a non-critical warning. Fix it with:
```bash
defaults write -g AppleLanguages -array en-US
```

**"ld: warning: ignoring duplicate libraries: '-lobjc'" (macOS)**
This is expected and harmless. It's from Fyne's CGo dependencies.

**Build errors on Linux**
Make sure you have all required packages:
```bash
# Ubuntu/Debian
sudo apt-get install gcc libgl1-mesa-dev libxcursor-dev libxrandr-dev libxinerama-dev libxi-dev libxxf86vm-dev

# Fedora/RHEL
sudo dnf install gcc libXcursor-devel libXrandr-devel mesa-libGL-devel libXi-devel libXinerama-devel libXxf86vm-devel
```

If you encounter specific linker errors like "cannot find -lXxf86vm", install the missing library:
```bash
sudo apt-get install libxxf86vm-dev  # for Xxf86vm
sudo apt-get install libxcursor-dev  # for Xcursor
sudo apt-get install libxrandr-dev   # for Xrandr
sudo apt-get install libxinerama-dev # for Xinerama
sudo apt-get install libxi-dev       # for Xi
```

**Build errors on macOS**
Install Xcode command line tools:
```bash
xcode-select --install
```

### Runtime Issues

**"No such file or directory" - history file not found**
- Verify your history file location: `echo $HISTFILE`
- Specify the correct path: `./history_viewer -history $HISTFILE`
- Or update `~/.history_viewer.json`

**"Port already in use"**
- Change the port: `./history_viewer -port 9090`
- Or kill the process using the port: `lsof -ti:8080 | xargs kill`

**"Ollama connection refused" - AI features not working**
- Start Ollama: `ollama serve` (if not running as a service)
- Verify it's accessible: `curl http://localhost:11434/api/tags`
- Pull a model if needed: `ollama pull llama3.3`
- Update `~/.history_viewer.json` with correct Ollama URL if using a different address

**Empty or no sessions showing**
- Ensure your zsh history has timestamps (see Prerequisites)
- Check if the history file has content: `head ~/.zsh_history`
- Verify the format includes timestamps: `: 1234567890:0;command`
- New commands only will have timestamps after enabling EXTENDED_HISTORY

**Browser shows "Cannot GET /" (Web UI)**
- Ensure you're accessing http://localhost:8080 (not just localhost:8080)
- Check server logs for any errors
- Verify the server started successfully

**Native UI won't start**
- Check if you have the required GUI dependencies (see Prerequisites)
- Try the web UI as a fallback: `./history_viewer` (without `-ui native`)
- Check for error messages in the terminal output

## Development Mode

For development with auto-rebuild on file changes:

```bash
# Install air (live reload tool)
go install github.com/cosmtrek/air@latest

# Run with auto-reload
air
```

Or manually rebuild and run:
```bash
go build -o history_viewer && ./history_viewer
```

**Quick rebuild and test:**
```bash
# Build and run in one command
go build -o history_viewer && ./history_viewer

# Or run in background
go build -o history_viewer && ./history_viewer &
```

## Installing Globally (Optional)

Move the binary to your PATH:

```bash
# macOS/Linux
sudo mv history_viewer /usr/local/bin/

# Now you can run from anywhere
history_viewer
```

Or add an alias to your `~/.zshrc`:
```bash
alias hist-viewer='/path/to/history_viewer'
```

## Stopping the Server

Press `Ctrl+C` in the terminal where the server is running.
