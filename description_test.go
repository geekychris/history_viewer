package main

import (
	"testing"
)

func TestTruncateDirectoryPath(t *testing.T) {
	homeDir := "/Users/chris"
	
	tests := []struct {
		name     string
		fullPath string
		want     string
	}{
		{
			name:     "home directory",
			fullPath: "/Users/chris",
			want:     "~",
		},
		{
			name:     "home subdirectory",
			fullPath: "/Users/chris/code",
			want:     "code",
		},
		{
			name:     "two levels deep",
			fullPath: "/Users/chris/code/project",
			want:     "project",
		},
		{
			name:     "three levels deep",
			fullPath: "/Users/chris/code/project/src",
			want:     "project/src",
		},
		{
			name:     "four levels deep",
			fullPath: "/Users/chris/code/project/src/components",
			want:     "src/components",
		},
		{
			name:     "very deep path",
			fullPath: "/Users/chris/code/warp_experiments/history_viewer/src/lib/utils",
			want:     "src/lib/utils",
		},
		{
			name:     "root directory",
			fullPath: "/",
			want:     ".",
		},
		{
			name:     "system path",
			fullPath: "/usr/local/bin",
			want:     "local/bin",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateDirectoryPath(tt.fullPath, homeDir)
			if got != tt.want {
				t.Errorf("truncateDirectoryPath(%q) = %q, want %q", tt.fullPath, got, tt.want)
			}
		})
	}
}

func TestExtractTopActivities(t *testing.T) {
	tests := []struct {
		name     string
		commands []HistoryEntry
		want     string
	}{
		{
			name: "dominant git",
			commands: []HistoryEntry{
				{BaseCommand: "git", Command: "git status"},
				{BaseCommand: "git", Command: "git add ."},
				{BaseCommand: "git", Command: "git commit -m 'test'"},
				{BaseCommand: "ls", Command: "ls -la"},
			},
			want: "git",
		},
		{
			name: "mixed git and npm",
			commands: []HistoryEntry{
				{BaseCommand: "git", Command: "git status"},
				{BaseCommand: "git", Command: "git add ."},
				{BaseCommand: "npm", Command: "npm install"},
				{BaseCommand: "npm", Command: "npm test"},
				{BaseCommand: "ls", Command: "ls"},
			},
			want: "git npm",
		},
		{
			name: "single dominant command",
			commands: []HistoryEntry{
				{BaseCommand: "docker", Command: "docker ps"},
				{BaseCommand: "docker", Command: "docker logs"},
				{BaseCommand: "docker", Command: "docker exec"},
				{BaseCommand: "ls", Command: "ls"},
			},
			want: "docker",
		},
		{
			name: "varied commands",
			commands: []HistoryEntry{
				{BaseCommand: "git", Command: "git status"},
				{BaseCommand: "git", Command: "git add ."},
				{BaseCommand: "npm", Command: "npm test"},
				{BaseCommand: "npm", Command: "npm install"},
				{BaseCommand: "ls", Command: "ls"},
			},
			want: "git npm",
		},
		{
			name: "local executable",
			commands: []HistoryEntry{
				{BaseCommand: "./history_viewer", Command: "./history_viewer"},
				{BaseCommand: "./history_viewer", Command: "./history_viewer -port 8080"},
			},
			want: "history_viewer",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := &Session{
				Commands: tt.commands,
			}
			got := extractTopActivities(session)
			if got != tt.want {
				t.Errorf("extractTopActivities() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFindMostActiveDirectory(t *testing.T) {
	tests := []struct {
		name     string
		session  *Session
		want     string
	}{
		{
			name: "single directory",
			session: &Session{
				Directories: []string{"/Users/chris/code/project"},
				Commands: []HistoryEntry{
					{Directory: "/Users/chris/code/project"},
					{Directory: "/Users/chris/code/project"},
				},
			},
			want: "/Users/chris/code/project",
		},
		{
			name: "multiple directories - first is most active",
			session: &Session{
				Directories: []string{"/Users/chris/code/project", "/Users/chris/code/other"},
				Commands: []HistoryEntry{
					{Directory: "/Users/chris/code/project"},
					{Directory: "/Users/chris/code/project"},
					{Directory: "/Users/chris/code/project"},
					{Directory: "/Users/chris/code/other"},
				},
			},
			want: "/Users/chris/code/project",
		},
		{
			name: "multiple directories - second is most active",
			session: &Session{
				Directories: []string{"/Users/chris/code/project", "/Users/chris/code/other"},
				Commands: []HistoryEntry{
					{Directory: "/Users/chris/code/project"},
					{Directory: "/Users/chris/code/other"},
					{Directory: "/Users/chris/code/other"},
					{Directory: "/Users/chris/code/other"},
				},
			},
			want: "/Users/chris/code/other",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findMostActiveDirectory(tt.session)
			if got != tt.want {
				t.Errorf("findMostActiveDirectory() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGenerateSessionDescription(t *testing.T) {
	tests := []struct {
		name    string
		session *Session
		homeDir string
		want    string
	}{
		{
			name: "git work in project",
			session: &Session{
				Commands: []HistoryEntry{
					{BaseCommand: "git", Command: "git status", Directory: "/Users/chris/code/project"},
					{BaseCommand: "git", Command: "git add .", Directory: "/Users/chris/code/project"},
					{BaseCommand: "git", Command: "git commit", Directory: "/Users/chris/code/project"},
				},
				Directories: []string{"/Users/chris/code/project"},
				Categories:  map[CommandCategory]int{CategoryVCS: 3},
			},
			homeDir: "/Users/chris",
			want:    "project: git [Version Control]",
		},
		{
			name: "mixed work in components",
			session: &Session{
				Commands: []HistoryEntry{
					{BaseCommand: "git", Command: "git status", Directory: "/Users/chris/code/project/src/components"},
					{BaseCommand: "git", Command: "git add .", Directory: "/Users/chris/code/project/src/components"},
					{BaseCommand: "npm", Command: "npm test", Directory: "/Users/chris/code/project/src/components"},
					{BaseCommand: "npm", Command: "npm build", Directory: "/Users/chris/code/project/src/components"},
					{BaseCommand: "ls", Command: "ls", Directory: "/Users/chris/code/project/src/components"},
				},
				Directories: []string{"/Users/chris/code/project/src/components"},
				Categories:  map[CommandCategory]int{CategoryVCS: 2, CategoryBuild: 2, CategoryFileOps: 1},
			},
			homeDir: "/Users/chris",
			want:    "src/components: git npm [Version Control, Build]",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment for test
			t.Setenv("HOME", tt.homeDir)
			
			got := generateSessionDescription(tt.session)
			if got != tt.want {
				t.Errorf("generateSessionDescription() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGenerateSessionDescriptionEmpty(t *testing.T) {
	session := &Session{
		Commands: []HistoryEntry{},
	}
	
	got := generateSessionDescription(session)
	if got != "Empty session" {
		t.Errorf("generateSessionDescription() for empty session = %q, want %q", got, "Empty session")
	}
}
