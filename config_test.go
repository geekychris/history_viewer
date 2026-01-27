package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_CustomCategoryPatterns(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".history_viewer.json")
	
	configData := map[string]interface{}{
		"history_file": "/tmp/test_history",
		"port":         9999,
		"custom_category_patterns": []map[string]string{
			{"category": "terraform", "pattern": `^terraform\s`},
			{"category": "my-tools", "pattern": `^(\.\/)?history_viewer(\s|$)`},
		},
	}
	
	data, err := json.MarshalIndent(configData, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}
	
	err = os.WriteFile(configPath, data, 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}
	
	// Override home directory for testing
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)
	
	// Load config
	config, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	
	// Verify custom patterns were loaded
	if len(config.CustomCategoryPatterns) != 2 {
		t.Errorf("Expected 2 custom patterns, got %d", len(config.CustomCategoryPatterns))
	}
	
	if config.CustomCategoryPatterns[0].Category != "terraform" {
		t.Errorf("Expected first pattern category to be 'terraform', got %q", config.CustomCategoryPatterns[0].Category)
	}
	
	if config.CustomCategoryPatterns[0].Pattern != `^terraform\s` {
		t.Errorf("Expected first pattern to be '^terraform\\s', got %q", config.CustomCategoryPatterns[0].Pattern)
	}
}

func TestLoadConfig_EmptyCustomPatterns(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".history_viewer.json")
	
	configData := map[string]interface{}{
		"history_file": "/tmp/test_history",
		"port":         9999,
	}
	
	data, err := json.MarshalIndent(configData, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}
	
	err = os.WriteFile(configPath, data, 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}
	
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)
	
	config, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	
	// Should have no custom patterns
	if len(config.CustomCategoryPatterns) != 0 {
		t.Errorf("Expected 0 custom patterns, got %d", len(config.CustomCategoryPatterns))
	}
}

func TestSaveConfig_CustomCategoryPatterns(t *testing.T) {
	tmpDir := t.TempDir()
	
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)
	
	config := &Config{
		HistoryFile: "/tmp/test_history",
		Port:        9999,
		CustomCategoryPatterns: []CustomCategoryPattern{
			{Category: "test", Pattern: `^test\s`},
		},
	}
	
	err := SaveConfig(config)
	if err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}
	
	// Read back the config
	configPath := filepath.Join(tmpDir, ".history_viewer.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read saved config: %v", err)
	}
	
	var savedConfig Config
	err = json.Unmarshal(data, &savedConfig)
	if err != nil {
		t.Fatalf("Failed to unmarshal saved config: %v", err)
	}
	
	if len(savedConfig.CustomCategoryPatterns) != 1 {
		t.Errorf("Expected 1 custom pattern in saved config, got %d", len(savedConfig.CustomCategoryPatterns))
	}
	
	if savedConfig.CustomCategoryPatterns[0].Category != "test" {
		t.Errorf("Expected saved category to be 'test', got %q", savedConfig.CustomCategoryPatterns[0].Category)
	}
}
