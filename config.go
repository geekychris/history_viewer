package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type Config struct {
	HistoryFile          string                  `json:"history_file"`
	Port                 int                     `json:"port"`
	SessionTimeout       time.Duration           `json:"session_timeout_minutes"`
	OllamaURL            string                  `json:"ollama_url"`
	OllamaModel          string                  `json:"ollama_model"`
	AutoRefreshSec       int                     `json:"auto_refresh_seconds"`
	HomeDir              string                  `json:"home_dir"`
	SessionHeuristics    SessionHeuristics       `json:"session_heuristics"`
	CustomCategoryPatterns []CustomCategoryPattern `json:"custom_category_patterns,omitempty"`
}

// SessionHeuristics defines configurable parameters for session detection
type SessionHeuristics struct {
	// TimeoutMinutes: primary timeout - gap between commands to start new session
	TimeoutMinutes int `json:"timeout_minutes"`
	
	// DirectoryChangeBreaksSession: whether changing to a completely different directory tree starts a new session
	DirectoryChangeBreaksSession bool `json:"directory_change_breaks_session"`
	
	// CategoryChangeThreshold: if commands shift from one category to another for N consecutive commands, start new session
	// Set to 0 to disable this heuristic
	CategoryChangeThreshold int `json:"category_change_threshold"`
	
	// MinCommandsPerSession: minimum number of commands to constitute a session (prevents tiny sessions)
	MinCommandsPerSession int `json:"min_commands_per_session"`
	
	// MaxSessionDuration: absolute maximum duration for a session in minutes, even without timeout gaps
	// Set to 0 to disable this heuristic
	MaxSessionDuration int `json:"max_session_duration_minutes"`
	
	// ShortBreakMinutes: a "short break" that doesn't end a session (e.g., coffee break)
	// Only used if less than TimeoutMinutes. Commands within short break are same session.
	ShortBreakMinutes int `json:"short_break_minutes"`
}

// CustomCategoryPattern allows users to add custom command categorization rules
type CustomCategoryPattern struct {
	Category string `json:"category"`
	Pattern  string `json:"pattern"` // regex pattern
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
		SessionHeuristics: SessionHeuristics{
			TimeoutMinutes:               30,
			DirectoryChangeBreaksSession: false,
			CategoryChangeThreshold:      0, // disabled by default
			MinCommandsPerSession:        1,
			MaxSessionDuration:           0, // disabled by default
			ShortBreakMinutes:            5,
		},
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
			// Load session heuristics if provided
			if fileConfig.SessionHeuristics.TimeoutMinutes != 0 {
				config.SessionHeuristics = fileConfig.SessionHeuristics
				config.SessionTimeout = time.Duration(fileConfig.SessionHeuristics.TimeoutMinutes) * time.Minute
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
			// Load custom category patterns
			if len(fileConfig.CustomCategoryPatterns) > 0 {
				config.CustomCategoryPatterns = fileConfig.CustomCategoryPatterns
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
