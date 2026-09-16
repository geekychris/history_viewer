package main

import (
	"flag"
	"log"
	"net/url"
)

func main() {
	portFlag := flag.Int("port", 0, "Port to run the server on")
	historyFileFlag := flag.String("history", "", "Path to zsh history file")
	uiFlag := flag.String("ui", "web", "UI mode: 'web' or 'native'")
	filterDirFlag := flag.String("filter-dir", "", "Deep-link: pre-filter the UI to commands run under this directory")
	flag.Parse()

	// Load configuration
	config, err := LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Override with command-line flags if provided
	if *portFlag != 0 {
		config.Port = *portFlag
	}
	if *historyFileFlag != "" {
		config.HistoryFile = *historyFileFlag
	}
	if *filterDirFlag != "" {
		config.InitialDirFilter = *filterDirFlag
	}

	log.Printf("History file: %s", config.HistoryFile)
	log.Printf("Session timeout: %v", config.SessionTimeout)
	log.Printf("Ollama URL: %s", config.OllamaURL)
	log.Printf("Ollama Model: %s", config.OllamaModel)

	if *uiFlag == "native" {
		// Start native UI
		log.Println("Starting native UI...")
		if config.InitialDirFilter != "" {
			// Native UI doesn't yet have a directory-filter widget, but the
			// server does — surface the equivalent web URL so the user can
			// open it in a browser side-by-side.
			log.Printf("Note: native UI does not yet honour --filter-dir; the web UI does. Open http://localhost:%d/?dir=%s",
				config.Port, url.QueryEscape(config.InitialDirFilter))
		}
		ui := NewNativeUI(config)
		if err := ui.Start(); err != nil {
			log.Fatalf("Native UI failed: %v", err)
		}
	} else {
		if config.InitialDirFilter != "" {
			// Log the deep-link URL so callers (Chief, scripts) know where to
			// point a browser once the server is up.
			log.Printf("Deep-link: http://localhost:%d/?dir=%s",
				config.Port, url.QueryEscape(config.InitialDirFilter))
		}
		// Start web server
		server := NewServer(config)
		if err := server.Start(); err != nil {
			log.Fatalf("Server failed: %v", err)
		}
	}
}
