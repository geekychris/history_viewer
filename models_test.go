package main

import (
	"testing"
)

func TestCategorizeCommand_BuiltIn(t *testing.T) {
	tests := []struct {
		command  string
		expected CommandCategory
	}{
		{"git status", CategoryVCS},
		{"git commit -m 'test'", CategoryVCS},
		{"docker ps", CategoryContainers},
		{"kubectl get pods", CategoryContainers},
		{"npm install", CategoryBuild}, // npm is in build category (matches build pattern first)
		{"ls -la", CategoryNavigation},
		{"cd /tmp", CategoryNavigation},
		{"vim test.go", CategoryDevTools}, // vim matches dev-tools before editor
		{"grep -r 'foo'", CategorySearch},
		{"apt install foo", CategoryPackage},
		{"unknown command", CategoryOther},
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			result := CategorizeCommand(tt.command)
			if result != tt.expected {
				t.Errorf("CategorizeCommand(%q) = %v, want %v", tt.command, result, tt.expected)
			}
		})
	}
}

func TestCategorizeCommand_CustomPatterns(t *testing.T) {
	// Set up custom patterns
	patterns := []struct {
		Category string
		Pattern  string
	}{
		{"terraform", `^terraform\s`},
		{"my-tools", `^(\.\/)?history_viewer(\s|$)`},
		{"cloud-cli", `^(aws|gcloud|az)\s`},
	}
	SetCustomCategoryPatterns(patterns)

	tests := []struct {
		command  string
		expected CommandCategory
	}{
		{"terraform plan", "terraform"},
		{"terraform apply -auto-approve", "terraform"},
		{"history_viewer", "my-tools"},
		{"./history_viewer", "my-tools"},
		{"./history_viewer &", "my-tools"},
		{"aws s3 ls", "cloud-cli"},
		{"gcloud compute instances list", "cloud-cli"},
		{"az vm list", "cloud-cli"},
		{"git status", CategoryVCS}, // Should still match built-in
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			result := CategorizeCommand(tt.command)
			if result != tt.expected {
				t.Errorf("CategorizeCommand(%q) = %v, want %v", tt.command, result, tt.expected)
			}
		})
	}

	// Clean up
	SetCustomCategoryPatterns(nil)
}

func TestCategorizeCommand_CustomOverridesBuiltIn(t *testing.T) {
	// Custom pattern that overrides built-in categorization
	patterns := []struct {
		Category string
		Pattern  string
	}{
		{"my-git", `^git\s`}, // Override built-in git category
	}
	SetCustomCategoryPatterns(patterns)

	result := CategorizeCommand("git status")
	expected := CommandCategory("my-git")
	
	if result != expected {
		t.Errorf("CategorizeCommand('git status') = %v, want %v (custom should override built-in)", result, expected)
	}

	// Clean up
	SetCustomCategoryPatterns(nil)
}

func TestSetCustomCategoryPatterns_InvalidRegex(t *testing.T) {
	// Invalid regex pattern should be silently ignored
	patterns := []struct {
		Category string
		Pattern  string
	}{
		{"invalid", `[unclosed`}, // Invalid regex
		{"valid", `^test\s`},      // Valid regex
	}
	
	SetCustomCategoryPatterns(patterns)
	
	// Should only match the valid pattern
	result := CategorizeCommand("test foo")
	expected := CommandCategory("valid")
	
	if result != expected {
		t.Errorf("CategorizeCommand('test foo') = %v, want %v", result, expected)
	}

	// Clean up
	SetCustomCategoryPatterns(nil)
}

func TestGetBaseCommand(t *testing.T) {
	tests := []struct {
		command  string
		expected string
	}{
		{"git status", "git"},
		{"ls -la /tmp", "ls"},
		{"./history_viewer &", "./history_viewer"},
		{"", ""},
		{"single", "single"},
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			result := GetBaseCommand(tt.command)
			if result != tt.expected {
				t.Errorf("GetBaseCommand(%q) = %v, want %v", tt.command, result, tt.expected)
			}
		})
	}
}

func TestCategorizeCommand_DotSlashPrefix(t *testing.T) {
	patterns := []struct {
		Category string
		Pattern  string
	}{
		{"local-bin", `^(\.\/)?my_script(\s|$)`},
	}
	SetCustomCategoryPatterns(patterns)

	tests := []struct {
		command  string
		expected CommandCategory
	}{
		{"my_script", "local-bin"},
		{"./my_script", "local-bin"},
		{"./my_script arg1 arg2", "local-bin"},
		{"my_script arg1", "local-bin"},
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			result := CategorizeCommand(tt.command)
			if result != tt.expected {
				t.Errorf("CategorizeCommand(%q) = %v, want %v", tt.command, result, tt.expected)
			}
		})
	}

	// Clean up
	SetCustomCategoryPatterns(nil)
}

func TestCategorizeCommand_SpecialCharacters(t *testing.T) {
	// Test that underscore and other special chars work correctly
	patterns := []struct {
		Category string
		Pattern  string
	}{
		{"test", `^history_viewer(\s|$)`},
		{"test2", `^my-script(\s|$)`},
		{"test3", `^script\.sh(\s|$)`},
	}
	SetCustomCategoryPatterns(patterns)

	tests := []struct {
		command  string
		expected CommandCategory
	}{
		{"history_viewer", "test"},
		{"history_viewer arg", "test"},
		{"my-script", "test2"},
		{"script.sh", "test3"},
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			result := CategorizeCommand(tt.command)
			if result != tt.expected {
				t.Errorf("CategorizeCommand(%q) = %v, want %v", tt.command, result, tt.expected)
			}
		})
	}

	// Clean up
	SetCustomCategoryPatterns(nil)
}
