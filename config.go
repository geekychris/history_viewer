package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type Config struct {
	HistoryFile       string        `json:"history_file"`
	Port              int           `json:"port"`
	SessionTimeout    time.Duration `json:"session_timeout_minutes"`
	OllamaURL         string        `json:"ollama_url"`
	OllamaModel       string        `json:"ollama_model"`
	AutoRefreshSec    int           `json:"auto_refresh_seconds"`
	HomeDir           string        `json:"home_dir"`
}

func LoadConfig() (*Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	config := &Config{
		HistoryFile:    filepath.Join(homeDir, ".zsh_history"),
		Port:           8080,
		SessionTimeout: 30 * time.Minute,
		OllamaURL:      "http://localhost:11434",
		OllamaModel:    "llama3.3",
		AutoRefreshSec: 30,
		HomeDir:        homeDir,
	}

	configPath := filepath.Join(homeDir, ".history_viewer.json")
	if data, err := os.ReadFile(configPath); err == nil {
		var fileConfig Config
		if err := json.Unmarshal(data, &fileConfig); err == nil {
			if fileConfig.HistoryFile != "" {
				config.HistoryFile = fileConfig.HistoryFile
			}
			if fileConfig.Port != 0 {
				config.Port = fileConfig.Port
			}
			if fileConfig.SessionTimeout != 0 {
				config.SessionTimeout = fileConfig.SessionTimeout * time.Minute
			}
			if fileConfig.OllamaURL != "" {
				config.OllamaURL = fileConfig.OllamaURL
			}
			if fileConfig.OllamaModel != "" {
				config.OllamaModel = fileConfig.OllamaModel
			}
			if fileConfig.AutoRefreshSec != 0 {
				config.AutoRefreshSec = fileConfig.AutoRefreshSec
			}
		}
	}

	return config, nil
}

func SaveConfig(config *Config) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configPath := filepath.Join(homeDir, ".history_viewer.json")
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}
