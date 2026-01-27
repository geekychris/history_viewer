package main

import (
	"flag"
	"log"
)

func main() {
	portFlag := flag.Int("port", 0, "Port to run the server on")
	historyFileFlag := flag.String("history", "", "Path to zsh history file")
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

	// Create and start server
	server := NewServer(config)
	log.Printf("History file: %s", config.HistoryFile)
	log.Printf("Session timeout: %v", config.SessionTimeout)
	log.Printf("Ollama URL: %s", config.OllamaURL)
	log.Printf("Ollama Model: %s", config.OllamaModel)
	
	if err := server.Start(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
