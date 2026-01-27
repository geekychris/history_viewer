package main

import (
	"flag"
	"log"
)

func main() {
	portFlag := flag.Int("port", 0, "Port to run the server on")
	historyFileFlag := flag.String("history", "", "Path to zsh history file")
	uiFlag := flag.String("ui", "web", "UI mode: 'web' or 'native'")
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

	log.Printf("History file: %s", config.HistoryFile)
	log.Printf("Session timeout: %v", config.SessionTimeout)
	log.Printf("Ollama URL: %s", config.OllamaURL)
	log.Printf("Ollama Model: %s", config.OllamaModel)

	if *uiFlag == "native" {
		// Start native UI
		log.Println("Starting native UI...")
		ui := NewNativeUI(config)
		if err := ui.Start(); err != nil {
			log.Fatalf("Native UI failed: %v", err)
		}
	} else {
		// Start web server
		server := NewServer(config)
		if err := server.Start(); err != nil {
			log.Fatalf("Server failed: %v", err)
		}
	}
}
