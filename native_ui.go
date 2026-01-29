package main

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

type NativeUI struct {
	app        fyne.App
	window     fyne.Window
	server     *Server
	sessionList *widget.List
	sessions   []*Session
	filtered   []*Session
	
	// Filters
	startDate  *widget.Entry
	endDate    *widget.Entry
	categorySelect *widget.Select
	keywordEntry *widget.Entry
	sortDescending bool
	
	// Selected session
	selectedIndex int
	detailsContainer *fyne.Container
	
	// Status
	statusLabel *widget.Label
}

func NewNativeUI(config *Config) *NativeUI {
	ui := &NativeUI{
		app:        app.New(),
		server:     NewServer(config),
		selectedIndex: -1,
		sortDescending: true,
	}
	
	ui.window = ui.app.NewWindow("Zsh History Viewer")
	ui.window.Resize(fyne.NewSize(1200, 800))
	
	return ui
}

func (ui *NativeUI) Start() error {
	// Parse history
	if err := ui.server.ParseHistory(); err != nil {
		return fmt.Errorf("failed to parse history: %w", err)
	}
	
	ui.sessions = ui.server.GetSessions("", "", "", "desc")
	ui.filtered = ui.sessions
	
	// Build UI
	ui.buildUI()
	
	ui.window.ShowAndRun()
	return nil
}

func (ui *NativeUI) buildUI() {
	// Top toolbar
	toolbar := ui.createToolbar()
	
	// Main content: session list + details (must be created before filters to avoid nil pointer)
	content := ui.createContentPanel()
	
	// Status bar (must be created before filters to avoid nil pointer)
	ui.statusLabel = widget.NewLabel("")
	
	// Filters panel
	filters := ui.createFiltersPanel()
	
	ui.updateStatus()
	
	// Layout
	mainContent := container.NewBorder(
		container.NewVBox(toolbar, filters),
		ui.statusLabel,
		nil,
		nil,
		content,
	)
	
	ui.window.SetContent(mainContent)
}

func (ui *NativeUI) createToolbar() *fyne.Container {
	// Create buttons with text labels for clarity
	refreshBtn := widget.NewButton("🔄 Refresh", func() {
		ui.refresh()
	})
	
	exportBtn := widget.NewButton("💾 Export All", func() {
		ui.showExportDialog()
	})
	
	aiBtn := widget.NewButton("🤖 AI Analyze Selected", func() {
		if ui.selectedIndex >= 0 && ui.selectedIndex < len(ui.filtered) {
			ui.showAIAnalysisDialog(ui.filtered[ui.selectedIndex])
		} else {
			dialog.ShowInformation("No Selection", "Please select a session first", ui.window)
		}
	})
	aiBtn.Importance = widget.HighImportance
	
	exportSessionBtn := widget.NewButton("📄 Export Selected", func() {
		if ui.selectedIndex >= 0 && ui.selectedIndex < len(ui.filtered) {
			ui.showSessionExportDialog(ui.filtered[ui.selectedIndex])
		} else {
			dialog.ShowInformation("No Selection", "Please select a session first", ui.window)
		}
	})
	
	aboutBtn := widget.NewButton("ℹ️ About", func() {
		ui.showAboutDialog()
	})
	
	return container.NewHBox(
		refreshBtn,
		widget.NewSeparator(),
		exportBtn,
		widget.NewSeparator(),
		exportSessionBtn,
		widget.NewSeparator(),
		aiBtn,
		widget.NewSeparator(),
		aboutBtn,
	)
}

func (ui *NativeUI) createFiltersPanel() *fyne.Container {
	// Date range
	ui.startDate = widget.NewEntry()
	ui.startDate.SetPlaceHolder("YYYY-MM-DD")
	ui.startDate.SetText("")
	ui.startDate.OnSubmitted = func(string) { ui.applyFilters() }
	
	ui.endDate = widget.NewEntry()
	ui.endDate.SetPlaceHolder("YYYY-MM-DD")
	ui.endDate.SetText("")
	ui.endDate.OnSubmitted = func(string) { ui.applyFilters() }
	
	// Category filter
	categories := []string{"All", "VCS", "Build", "File Ops", "Navigation", "Dev Tools", 
		"System Admin", "Network", "Containers", "Database", "Editor", "Search", 
		"Package Manager", "Other"}
	ui.categorySelect = widget.NewSelect(categories, func(string) {
		// Auto-apply when category changes
		ui.applyFilters()
	})
	
	// Keyword search
	ui.keywordEntry = widget.NewEntry()
	ui.keywordEntry.SetPlaceHolder("Search keywords...")
	ui.keywordEntry.OnSubmitted = func(string) { ui.applyFilters() }
	
	// Sort toggle
	var sortBtn *widget.Button
	sortBtn = widget.NewButton("Sort: Newest First", func() {
		ui.sortDescending = !ui.sortDescending
		if ui.sortDescending {
			sortBtn.SetText("Sort: Newest First")
		} else {
			sortBtn.SetText("Sort: Oldest First")
		}
		ui.applyFilters()
	})
	
	// Apply button
	applyBtn := widget.NewButton("Apply Filters", func() {
		ui.applyFilters()
	})
	applyBtn.Importance = widget.HighImportance
	
	// Clear button
	clearBtn := widget.NewButton("Clear", func() {
		ui.startDate.SetText("")
		ui.endDate.SetText("")
		ui.categorySelect.SetSelected("All")
		ui.keywordEntry.SetText("")
		ui.applyFilters()
	})
	
	// Set initial value after all widgets are created to avoid nil pointer during callback
	ui.categorySelect.SetSelected("All")
	
	// Layout
	dateRow := container.NewGridWithColumns(2,
		container.NewBorder(nil, nil, widget.NewLabel("From:"), nil, ui.startDate),
		container.NewBorder(nil, nil, widget.NewLabel("To:"), nil, ui.endDate),
	)
	
	filterRow := container.NewGridWithColumns(3,
		container.NewBorder(nil, nil, widget.NewLabel("Category:"), nil, ui.categorySelect),
		container.NewBorder(nil, nil, widget.NewLabel("Keywords:"), nil, ui.keywordEntry),
		sortBtn,
	)
	
	buttonRow := container.NewGridWithColumns(2, applyBtn, clearBtn)
	
	return container.NewVBox(dateRow, filterRow, buttonRow)
}

func (ui *NativeUI) createContentPanel() *fyne.Container {
	// Session list
	ui.sessionList = widget.NewList(
		func() int {
			return len(ui.filtered)
		},
		func() fyne.CanvasObject {
			return container.NewVBox(
				widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
				widget.NewLabel(""),
				widget.NewLabel(""),
			)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id >= len(ui.filtered) {
				return
			}
			
			session := ui.filtered[id]
			container := obj.(*fyne.Container)
			
			title := container.Objects[0].(*widget.Label)
			timestamp := container.Objects[1].(*widget.Label)
			summary := container.Objects[2].(*widget.Label)
			
			title.SetText(session.Description)
			timestamp.SetText(session.StartTime.Format("2006-01-02 15:04:05"))
			
			cmdCount := len(session.Commands)
			// Convert categories map to string
			categories := make([]string, 0, len(session.Categories))
			for cat := range session.Categories {
				categories = append(categories, string(cat))
			}
			categoryStr := strings.Join(categories, ", ")
			if categoryStr == "" {
				categoryStr = "None"
			}
			summary.SetText(fmt.Sprintf("%d commands | Categories: %s", cmdCount, categoryStr))
		},
	)
	
	ui.sessionList.OnSelected = func(id widget.ListItemID) {
		ui.selectedIndex = id
		ui.showSessionDetails(ui.filtered[id])
	}
	
	// Details panel
	ui.detailsContainer = container.NewVBox(
		widget.NewLabel("Select a session to view details"),
	)
	
	// Split view
	split := container.NewHSplit(
		container.NewBorder(
			widget.NewLabel("Sessions"),
			nil, nil, nil,
			ui.sessionList,
		),
		container.NewBorder(
			widget.NewLabel("Details"),
			nil, nil, nil,
			container.NewVScroll(ui.detailsContainer),
		),
	)
	// Set split to 60% sessions list, 40% details (wider session panel)
	split.SetOffset(0.6)
	
	return container.NewMax(split)
}

func (ui *NativeUI) showSessionDetails(session *Session) {
	// Clear existing details
	ui.detailsContainer.Objects = nil
	
	// Session info
	// Convert categories map to string
	categories := make([]string, 0, len(session.Categories))
	for cat := range session.Categories {
		categories = append(categories, string(cat))
	}
	categoryStr := strings.Join(categories, ", ")
	if categoryStr == "" {
		categoryStr = "None"
	}
	
	// Get primary directory (most common directory)
	directory := "N/A"
	if len(session.Directories) > 0 {
		directory = session.Directories[0]
	}
	
	info := widget.NewRichTextFromMarkdown(fmt.Sprintf(`
**Description:** %s
**Start Time:** %s
**End Time:** %s
**Duration:** %s
**Directory:** %s
**Categories:** %s
**Command Count:** %d
`, 
		session.Description,
		session.StartTime.Format("2006-01-02 15:04:05"),
		session.EndTime.Format("2006-01-02 15:04:05"),
		session.Duration.Round(time.Second).String(),
		directory,
		categoryStr,
		len(session.Commands),
	))
	
	ui.detailsContainer.Add(info)
	
	// Commands list
	ui.detailsContainer.Add(widget.NewSeparator())
	ui.detailsContainer.Add(widget.NewLabelWithStyle("Commands:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
	
	for i, entry := range session.Commands {
		cmdText := fmt.Sprintf("%d. [%s] %s", i+1, entry.Directory, entry.Command)
		cmdLabel := widget.NewLabel(cmdText)
		cmdLabel.Wrapping = fyne.TextWrapWord
		ui.detailsContainer.Add(cmdLabel)
	}
	
	ui.detailsContainer.Refresh()
}

func (ui *NativeUI) applyFilters() {
	startDate := ui.startDate.Text
	endDate := ui.endDate.Text
	category := ui.categorySelect.Selected
	if category == "All" {
		category = ""
	}
	keyword := ui.keywordEntry.Text
	
	ui.filtered = ui.server.GetSessions(startDate, endDate, category, keyword)
	
	// Apply sort
	sortOrder := "desc"
	if !ui.sortDescending {
		sortOrder = "asc"
	}
	
	if sortOrder == "desc" {
		sort.Slice(ui.filtered, func(i, j int) bool {
			return ui.filtered[i].StartTime.After(ui.filtered[j].StartTime)
		})
	} else {
		sort.Slice(ui.filtered, func(i, j int) bool {
			return ui.filtered[i].StartTime.Before(ui.filtered[j].StartTime)
		})
	}
	
	ui.sessionList.Refresh()
	ui.selectedIndex = -1
	ui.detailsContainer.Objects = nil
	ui.detailsContainer.Add(widget.NewLabel("Select a session to view details"))
	ui.detailsContainer.Refresh()
	ui.updateStatus()
}

func (ui *NativeUI) refresh() {
	ui.statusLabel.SetText("Refreshing...")
	
	if err := ui.server.ParseHistory(); err != nil {
		dialog.ShowError(fmt.Errorf("failed to refresh: %w", err), ui.window)
		ui.statusLabel.SetText("Refresh failed")
		return
	}
	
	ui.sessions = ui.server.GetSessions("", "", "", "desc")
	ui.applyFilters()
	ui.statusLabel.SetText("Refreshed successfully")
	
	// Clear status after 3 seconds
	time.AfterFunc(3*time.Second, func() {
		ui.updateStatus()
	})
}

func (ui *NativeUI) updateStatus() {
	total := len(ui.sessions)
	filtered := len(ui.filtered)
	
	if filtered == total {
		ui.statusLabel.SetText(fmt.Sprintf("Total sessions: %d", total))
	} else {
		ui.statusLabel.SetText(fmt.Sprintf("Showing %d of %d sessions", filtered, total))
	}
}

func (ui *NativeUI) showExportDialog() {
	// Export options
	formatSelect := widget.NewSelect([]string{"JSON", "CSV", "Markdown", "Zsh"}, nil)
	formatSelect.SetSelected("JSON")
	
	content := container.NewVBox(
		widget.NewLabel("Select export format:"),
		formatSelect,
	)
	
	dialog.ShowCustomConfirm("Export History", "Export", "Cancel", content, func(ok bool) {
		if !ok {
			return
		}
		
		format := strings.ToLower(formatSelect.Selected)
		
		// Get filtered data
		data, err := ui.server.ExportSessions(ui.filtered, format)
		if err != nil {
			dialog.ShowError(fmt.Errorf("export failed: %w", err), ui.window)
			return
		}
		
		// Save file dialog
		saveDialog := dialog.NewFileSave(func(writer fyne.URIWriteCloser, err error) {
			if err != nil {
				dialog.ShowError(err, ui.window)
				return
			}
			if writer == nil {
				return
			}
			
			_, err = writer.Write(data)
			writer.Close()
			
			if err != nil {
				dialog.ShowError(err, ui.window)
			} else {
				dialog.ShowInformation("Success", "Export completed successfully", ui.window)
			}
		}, ui.window)
		
		saveDialog.SetFileName(fmt.Sprintf("history_export.%s", format))
		saveDialog.Show()
		
	}, ui.window)
}

func (ui *NativeUI) showSessionExportDialog(session *Session) {
	// Export to script formats
	langSelect := widget.NewSelect([]string{"Bash", "Python", "Java", "Go"}, nil)
	langSelect.SetSelected("Bash")
	
	content := container.NewVBox(
		widget.NewLabel("Export session to:"),
		langSelect,
	)
	
	dialog.ShowCustomConfirm("Export Session", "Export", "Cancel", content, func(ok bool) {
		if !ok {
			return
		}
		
		lang := strings.ToLower(langSelect.Selected)
		script := ui.server.ConvertToScript([]*Session{session}, lang)
		
		// Save file dialog
		saveDialog := dialog.NewFileSave(func(writer fyne.URIWriteCloser, err error) {
			if err != nil {
				dialog.ShowError(err, ui.window)
				return
			}
			if writer == nil {
				return
			}
			
			_, err = writer.Write([]byte(script))
			writer.Close()
			
			if err != nil {
				dialog.ShowError(err, ui.window)
			} else {
				dialog.ShowInformation("Success", "Script exported successfully", ui.window)
			}
		}, ui.window)
		
		ext := map[string]string{"bash": "sh", "python": "py", "java": "java", "go": "go"}
		saveDialog.SetFileName(fmt.Sprintf("session_%d.%s", session.SequenceNumber, ext[lang]))
		saveDialog.Show()
		
	}, ui.window)
}

func (ui *NativeUI) showAIAnalysisDialog(session *Session) {
	// Analysis type
	actionSelect := widget.NewSelect([]string{"Explain", "Custom Prompt"}, nil)
	actionSelect.SetSelected("Explain")
	
	// Custom prompt entry - make it taller
	promptEntry := widget.NewMultiLineEntry()
	promptEntry.SetPlaceHolder("Enter your custom prompt...")
	promptEntry.SetMinRowsVisible(8)
	promptEntry.Hide()
	
	actionSelect.OnChanged = func(selected string) {
		if selected == "Custom Prompt" {
			promptEntry.Show()
		} else {
			promptEntry.Hide()
		}
	}
	
	content := container.NewVBox(
		widget.NewLabel("Select analysis type:"),
		actionSelect,
		container.NewVScroll(promptEntry),
	)
	
	analysisDialog := dialog.NewCustomConfirm("AI Analysis", "Analyze", "Cancel", content, func(ok bool) {
		if !ok {
			return
		}
		
		// Show progress dialog
		progress := dialog.NewProgressInfinite("Analyzing", "Running AI analysis...", ui.window)
		progress.Show()
		
		go func() {
			var result string
			var err error
			
			sessionID := session.ID
			if actionSelect.Selected == "Explain" {
				result, err = ui.server.AnalyzeSession(sessionID, "explain", "")
			} else {
				result, err = ui.server.AnalyzeSession(sessionID, "", promptEntry.Text)
			}
			
			progress.Hide()
			
			if err != nil {
				dialog.ShowError(fmt.Errorf("analysis failed: %w", err), ui.window)
				return
			}
			
			// Show result in a selectable/copyable text area
			resultEntry := widget.NewMultiLineEntry()
			resultEntry.SetText(result)
			resultEntry.Wrapping = fyne.TextWrapWord
			resultScroll := container.NewVScroll(resultEntry)
			resultScroll.SetMinSize(fyne.NewSize(800, 500))
			
			// Add copy button
			copyBtn := widget.NewButton("📋 Copy to Clipboard", func() {
				ui.window.Clipboard().SetContent(result)
				dialog.ShowInformation("Copied", "Result copied to clipboard", ui.window)
			})
			copyBtn.Importance = widget.HighImportance
			
			resultContent := container.NewBorder(nil, copyBtn, nil, nil, resultScroll)
			
			resultDialog := dialog.NewCustom("AI Analysis Result", "Close", resultContent, ui.window)
			resultDialog.Resize(fyne.NewSize(900, 600))
			resultDialog.Show()
		}()
		
	}, ui.window)
	analysisDialog.Resize(fyne.NewSize(600, 400))
	analysisDialog.Show()
}

func (ui *NativeUI) showAboutDialog() {
	about := fmt.Sprintf(`Zsh History Viewer
Version: 1.0.0
Go Version: %s

A powerful tool for analyzing zsh command history with AI integration.

Features:
• Session grouping and categorization
• Interactive filtering and search
• AI-powered analysis via Ollama
• Export to multiple formats
• Native cross-platform UI

History File: %s
Ollama URL: %s
Model: %s`,
		"1.23",
		ui.server.Config.HistoryFile,
		ui.server.Config.OllamaURL,
		ui.server.Config.OllamaModel,
	)
	
	dialog.ShowInformation("About", about, ui.window)
}
