# Screenshots

To add screenshots of the History Viewer UI:

1. **Start the application:**
   ```bash
   go build -o history_viewer && ./history_viewer
   ```

2. **Open in browser:**
   ```
   http://localhost:8080
   ```

3. **Capture screenshots:**
   
   **macOS:**
   ```bash
   # Full window capture (interactive)
   screencapture -i screenshots/main-view.png
   
   # Or use Cmd+Shift+4, then press Space to select the window
   ```
   
   **Linux:**
   ```bash
   # Using gnome-screenshot
   gnome-screenshot -w -f screenshots/main-view.png
   
   # Using import (ImageMagick)
   import screenshots/main-view.png
   ```

4. **Recommended screenshots:**
   - `main-view.png` - Main interface showing sessions and timeline
   - `ai-analysis.png` - AI analysis panel with example output
   - `filters.png` - Filter controls and date pickers
   - `export.png` - Export dialog with format options
   - `native-ui.png` - Native desktop application (if available)

## Screenshot Guidelines

- Use a clean browser window without bookmarks/extensions visible
- Show actual command history (not empty state)
- Include the command timeline chart at the top
- Capture at 1920x1080 or similar 16:9 resolution
- Use the default purple gradient theme
- Show interactive features like hover states if possible
