# Native UI Implementation

## Overview

The Zsh History Viewer now includes a native desktop UI built with [Fyne](https://fyne.io/), a cross-platform GUI toolkit for Go. This provides an alternative to the web-based interface.

## Features

The native UI includes all features from the web UI:

### Viewing & Filtering
- **Session List**: Scrollable list of all history sessions with descriptions
- **Date Range Filters**: Filter sessions by start and end dates
- **Category Filter**: Filter by command category (VCS, Build, File Ops, etc.)
- **Keyword Search**: Search across session descriptions and commands
- **Sort Toggle**: Switch between newest-first and oldest-first ordering
- **Auto-refresh**: Manual refresh button to reload history

### Session Details
- **Session Information**: View start time, end time, duration, directory, categories
- **Command List**: See all commands in a session with their directories
- **Markdown Rendering**: Session info displays in formatted markdown

### Export Options
- **Data Export**: Export filtered sessions to JSON, CSV, Markdown, or Zsh format
- **Script Generation**: Convert session commands to executable scripts:
  - Bash (with error handling)
  - Python (with subprocess)
  - Java (with ProcessBuilder)
  - Go (with exec.Command)
- **Native File Dialogs**: Use OS-native save dialogs

### AI Analysis (via Ollama)
- **Explain Session**: Get AI explanation of what commands accomplished
- **Custom Prompts**: Ask specific questions about session commands
- **Progress Indicators**: Visual feedback during AI analysis
- **Modal Results**: AI analysis displayed in scrollable dialog

## Architecture

### Components

#### NativeUI Struct
- `app`: Fyne application instance
- `window`: Main application window
- `server`: Backend server instance (reused from web UI)
- `sessionList`: Widget displaying sessions
- `sessions`: All available sessions
- `filtered`: Currently filtered/displayed sessions
- Filter widgets (date, category, keyword)
- `detailsContainer`: Panel showing selected session details

#### Key Methods

**Initialization**
- `NewNativeUI(config)`: Creates UI with configuration
- `Start()`: Parses history and shows window
- `buildUI()`: Constructs UI layout

**UI Components**
- `createToolbar()`: Refresh, export, about buttons
- `createFiltersPanel()`: Date pickers, category selector, keyword search
- `createContentPanel()`: Split view with session list and details

**Data Operations**
- `applyFilters()`: Filter sessions based on criteria
- `refresh()`: Reload history from file
- `updateStatus()`: Update status bar with session counts

**Dialogs**
- `showSessionDetails()`: Display detailed session info
- `showExportDialog()`: Choose export format
- `showSessionExportDialog()`: Export session to script
- `showAIAnalysisDialog()`: Run AI analysis with progress
- `showAboutDialog()`: Display app information

### Backend Integration

The native UI reuses the existing backend server components:
- **Parser**: Reads and parses zsh history
- **Session Management**: Groups commands into sessions
- **Exporter**: Converts to various formats
- **Ollama Client**: AI analysis integration

Additional helper methods were added to `Server`:
- `ParseHistory()`: Public wrapper for parsing
- `GetSessions()`: Returns filtered sessions as pointers
- `ExportSessions()`: Exports to byte array
- `ConvertToScript()`: Generates executable scripts
- `AnalyzeSession()`: Performs AI analysis

## Platform Support

### macOS
- **Requirements**: Xcode command line tools
- **Build**: `go build -o history_viewer .`
- **Run**: `./history_viewer -ui native`

### Linux
- **Requirements**: 
  - Ubuntu/Debian: `gcc libgl1-mesa-dev xorg-dev`
  - Fedora/RHEL: `gcc libXcursor-devel libXrandr-devel mesa-libGL-devel libXi-devel libXinerama-devel libXxf86vm-devel`
- **Build**: `go build -o history_viewer .`
- **Run**: `./history_viewer -ui native`

### Windows
While Fyne supports Windows, the tool is designed for Unix-like systems (zsh history parsing). Windows support is not officially tested.

## Dependencies

### Go Modules
- `fyne.io/fyne/v2` v2.5.2 - Core UI framework
- Standard library packages (fmt, strings, time, sort, strconv)

### Build Dependencies
See BUILD.md for platform-specific build requirements.

## Usage Examples

### Basic Launch
```bash
./history_viewer -ui native
```

### With Custom History File
```bash
./history_viewer -ui native -history /path/to/.zsh_history
```

### With Custom Config
```bash
./history_viewer -ui native -config /path/to/config.json
```

## UI Screenshots & Behavior

### Main Window Layout
- **Top**: Toolbar with refresh, export, and about icons
- **Below Toolbar**: Filters (date range, category, keywords, sort)
- **Main Area**: Split view
  - **Left (40%)**: Session list with descriptions, timestamps, summaries
  - **Right (60%)**: Session details panel
- **Bottom**: Status bar showing session counts

### Interactions
- **Click Session**: Shows details in right panel
- **Apply Filters**: Updates session list
- **Clear Filters**: Resets all filters
- **Refresh Button**: Reloads history from file
- **Export Button**: Opens format selection dialog
- **Session Export**: Opens language selection dialog
- **AI Analysis**: Shows progress spinner, then result in modal

### Visual Design
- Native OS styling via Fyne
- Responsive layout
- Scrollable panels
- Progress indicators for long operations
- Modal dialogs for results and confirmations

## Comparison: Web UI vs Native UI

| Feature | Web UI | Native UI |
|---------|--------|-----------|
| Installation | Browser required | Standalone app |
| Chart Visualization | Interactive Chart.js | Not implemented |
| File Export | Browser download | Native file dialog |
| Markdown Rendering | marked.js | Fyne rich text |
| Auto-refresh | Every 30s (background) | Manual refresh button |
| Custom Prompts | Inline text area | Dialog with text entry |
| Platform Support | Any with browser | macOS, Linux |
| Resource Usage | Browser + Server | Single process |

## Implementation Notes

### Code Organization
- `native_ui.go`: All native UI code (546 lines)
- `main.go`: Modified to support `-ui` flag
- `server.go`: Added helper methods for native UI
- `export.go`: Added script generation methods
- `ollama.go`: Added custom prompt support

### Design Decisions
1. **Reuse Backend**: Leverages existing parser, categorization, session grouping
2. **Fyne Framework**: Cross-platform, native look and feel, active development
3. **Separate Binary**: Same binary supports both UIs via flag
4. **No Chart**: Native chart widgets are complex; list view is more practical
5. **Modal Dialogs**: AI results in dialogs to keep main window clean

### Challenges Solved
- **Session List Rendering**: Used Fyne's List widget with custom template
- **Split View**: HSplit container with adjustable divider
- **Markdown**: Fyne's RichTextFromMarkdown for formatted text
- **Categories Conversion**: Map to slice for display
- **Session ID Types**: Convert int to string for API compatibility
- **Import Aliasing**: Avoided conflict between `sort` variable and package

## Future Enhancements

Potential improvements:
- Chart visualization using a chart library
- Drag-to-select date ranges on timeline
- Keyboard shortcuts for common actions
- Session tagging and notes
- Command editing and replay
- Diff view between sessions
- Real-time auto-refresh with file watching
- Dark/light theme toggle
- Window state persistence
- Recent files menu

## Testing

### Manual Testing Checklist
- [ ] Launch native UI
- [ ] View session list
- [ ] Select session and view details
- [ ] Apply date range filter
- [ ] Apply category filter
- [ ] Apply keyword search
- [ ] Toggle sort order
- [ ] Refresh history
- [ ] Export to JSON/CSV/Markdown/Zsh
- [ ] Export session to Bash/Python/Java/Go
- [ ] Run AI explain analysis
- [ ] Run AI custom prompt
- [ ] Check about dialog

### Known Limitations
- No chart visualization (compared to web UI)
- No automatic refresh (manual button instead)
- Requires display server (X11/Wayland on Linux)
- Markdown rendering simpler than web version
