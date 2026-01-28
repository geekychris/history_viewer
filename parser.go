package main

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Parser struct {
	config *Config
}

func NewParser(config *Config) *Parser {
	// Initialize custom category patterns from config
	if len(config.CustomCategoryPatterns) > 0 {
		patterns := make([]struct {
			Category string
			Pattern  string
		}, len(config.CustomCategoryPatterns))
		for i, p := range config.CustomCategoryPatterns {
			patterns[i].Category = p.Category
			patterns[i].Pattern = p.Pattern
		}
		SetCustomCategoryPatterns(patterns)
	}
	return &Parser{config: config}
}

// Parse zsh history format: : <timestamp>:<duration>;<command>
var historyLineRegex = regexp.MustCompile(`^:\s*(\d+):(\d+);(.*)$`)
// Match cd command but stop at &&, ||, ;, or |
var cdRegex = regexp.MustCompile(`^\s*cd\s+([^;&|]+)`)

func (p *Parser) ParseHistory() ([]HistoryEntry, error) {
	file, err := os.Open(p.config.HistoryFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var entries []HistoryEntry
	scanner := bufio.NewScanner(file)
	
	// Increase buffer size for long commands
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)
	
	currentDir := p.config.HomeDir
	id := 1

	for scanner.Scan() {
		line := scanner.Text()
		
		matches := historyLineRegex.FindStringSubmatch(line)
		if len(matches) == 4 {
			timestamp, _ := strconv.ParseInt(matches[1], 10, 64)
			duration, _ := strconv.Atoi(matches[2])
			command := matches[3]
			
			// Handle multi-line commands
			for scanner.Scan() {
				nextLine := scanner.Text()
				if historyLineRegex.MatchString(nextLine) {
					// This is a new command, need to "unscan" it
					// Since we can't unscan, we'll process it in the next iteration
					line = nextLine
					break
				}
				command += "\n" + nextLine
			}
			
			// Track directory changes
			if cdMatches := cdRegex.FindStringSubmatch(command); len(cdMatches) > 1 {
				newDir := strings.TrimSpace(cdMatches[1])
				currentDir = p.resolveDirectory(currentDir, newDir)
			}
			
			entry := HistoryEntry{
				ID:          id,
				Timestamp:   time.Unix(timestamp, 0),
				Duration:    duration,
				Command:     command,
				Directory:   currentDir,
				Category:    CategorizeCommand(command),
				BaseCommand: GetBaseCommand(command),
			}
			
			entries = append(entries, entry)
			id++
			
			// If we read ahead to check for multiline, process that line now
			if historyLineRegex.MatchString(line) {
				matches = historyLineRegex.FindStringSubmatch(line)
				if len(matches) == 4 {
					timestamp, _ := strconv.ParseInt(matches[1], 10, 64)
					duration, _ := strconv.Atoi(matches[2])
					command := matches[3]
					
					if cdMatches := cdRegex.FindStringSubmatch(command); len(cdMatches) > 1 {
						newDir := strings.TrimSpace(cdMatches[1])
						currentDir = p.resolveDirectory(currentDir, newDir)
					}
					
					entry := HistoryEntry{
						ID:          id,
						Timestamp:   time.Unix(timestamp, 0),
						Duration:    duration,
						Command:     command,
						Directory:   currentDir,
						Category:    CategorizeCommand(command),
						BaseCommand: GetBaseCommand(command),
					}
					
					entries = append(entries, entry)
					id++
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return entries, nil
}

func (p *Parser) resolveDirectory(currentDir, newDir string) string {
	newDir = strings.Trim(newDir, "\"'")
	
	// Handle special cases
	if newDir == "~" || newDir == "" {
		return p.config.HomeDir
	}
	
	if strings.HasPrefix(newDir, "~/") {
		return filepath.Join(p.config.HomeDir, newDir[2:])
	}
	
	if filepath.IsAbs(newDir) {
		return filepath.Clean(newDir)
	}
	
	// Handle relative paths
	if newDir == ".." {
		return filepath.Dir(currentDir)
	}
	
	if newDir == "." {
		return currentDir
	}
	
	// Resolve relative path
	return filepath.Clean(filepath.Join(currentDir, newDir))
}

func (p *Parser) GroupIntoSessions(entries []HistoryEntry) []Session {
	if len(entries) == 0 {
		return []Session{}
	}

	sessions := []Session{}
	currentSession := Session{
		ID:         1,
		StartTime:  entries[0].Timestamp,
		Commands:   []HistoryEntry{},
		Categories: make(map[CommandCategory]int),
	}
	
	dirSet := make(map[string]bool)
	consecutiveCategoryChanges := 0
	var lastCategory CommandCategory

	for i, entry := range entries {
		// Check if we should start a new session
		if i > 0 {
			shouldBreak := false
			timeSinceLastCommand := entry.Timestamp.Sub(entries[i-1].Timestamp)
			
			// Heuristic 1: Timeout-based (primary)
			if timeSinceLastCommand > p.config.SessionTimeout {
				shouldBreak = true
			}
			
			// Heuristic 2: Directory change (if enabled)
			if p.config.SessionHeuristics.DirectoryChangeBreaksSession && !shouldBreak {
				if !isRelatedDirectory(entries[i-1].Directory, entry.Directory) {
					shouldBreak = true
				}
			}
			
			// Heuristic 3: Category change threshold
			if p.config.SessionHeuristics.CategoryChangeThreshold > 0 && !shouldBreak {
				if entry.Category != lastCategory && lastCategory != "" {
					consecutiveCategoryChanges++
					if consecutiveCategoryChanges >= p.config.SessionHeuristics.CategoryChangeThreshold {
						shouldBreak = true
					}
				} else {
					consecutiveCategoryChanges = 0
				}
			}
			
			// Heuristic 4: Maximum session duration
			if p.config.SessionHeuristics.MaxSessionDuration > 0 && !shouldBreak {
				sessionDuration := entry.Timestamp.Sub(currentSession.StartTime)
				maxDuration := time.Duration(p.config.SessionHeuristics.MaxSessionDuration) * time.Minute
				if sessionDuration > maxDuration {
					shouldBreak = true
				}
			}
			
			if shouldBreak {
				// Check minimum commands per session
				if len(currentSession.Commands) >= p.config.SessionHeuristics.MinCommandsPerSession {
					// Finalize current session
					currentSession.EndTime = entries[i-1].Timestamp
					currentSession.Duration = currentSession.EndTime.Sub(currentSession.StartTime)
					currentSession.Directories = getUniqueDirectories(dirSet)
					currentSession.Description = generateSessionDescription(&currentSession)
					sessions = append(sessions, currentSession)
					
					// Start new session
					currentSession = Session{
						ID:         len(sessions) + 1,
						StartTime:  entry.Timestamp,
						Commands:   []HistoryEntry{},
						Categories: make(map[CommandCategory]int),
					}
					dirSet = make(map[string]bool)
					consecutiveCategoryChanges = 0
				}
			}
		}
		
		lastCategory = entry.Category
		entry.SessionID = currentSession.ID
		currentSession.Commands = append(currentSession.Commands, entry)
		currentSession.Categories[entry.Category]++
		dirSet[entry.Directory] = true
	}
	
	// Don't forget the last session
	if len(currentSession.Commands) >= p.config.SessionHeuristics.MinCommandsPerSession {
		currentSession.EndTime = entries[len(entries)-1].Timestamp
		currentSession.Duration = currentSession.EndTime.Sub(currentSession.StartTime)
		currentSession.Directories = getUniqueDirectories(dirSet)
		currentSession.Description = generateSessionDescription(&currentSession)
		sessions = append(sessions, currentSession)
	}

	return sessions
}

func getUniqueDirectories(dirSet map[string]bool) []string {
	dirs := []string{}
	for dir := range dirSet {
		dirs = append(dirs, dir)
	}
	return dirs
}

// isRelatedDirectory checks if two directories are in the same tree
// (one is a parent/child of the other, or they share a common parent)
func isRelatedDirectory(dir1, dir2 string) bool {
	if dir1 == dir2 {
		return true
	}
	
	// Check if one is a subdirectory of the other
	if strings.HasPrefix(dir2, dir1+string(filepath.Separator)) {
		return true
	}
	if strings.HasPrefix(dir1, dir2+string(filepath.Separator)) {
		return true
	}
	
	// Check if they share a common parent (same parent directory)
	parent1 := filepath.Dir(dir1)
	parent2 := filepath.Dir(dir2)
	if parent1 == parent2 {
		return true
	}
	
	return false
}

// truncateDirectoryPath extracts a meaningful short directory name from a full path
// Examples:
//   /Users/chris/code/project/history_viewer -> history_viewer
//   /Users/chris/code/project/src/components -> project/src/components
//   /Users/chris -> ~
func truncateDirectoryPath(fullPath, homeDir string) string {
	// Replace home directory with ~
	if fullPath == homeDir {
		return "~"
	}
	
	// Split into components
	parts := strings.Split(filepath.Clean(fullPath), string(filepath.Separator))
	if len(parts) == 0 {
		return "."
	}
	
	// Filter out empty parts and home directory components
	filtered := []string{}
	for _, p := range parts {
		if p != "" {
			filtered = append(filtered, p)
		}
	}
	parts = filtered
	
	if len(parts) == 0 {
		return "."
	}
	
	// If this starts with home directory, remove those components
	if strings.HasPrefix(fullPath, homeDir+string(filepath.Separator)) {
		homeParts := strings.Split(filepath.Clean(homeDir), string(filepath.Separator))
		homePartsFiltered := []string{}
		for _, p := range homeParts {
			if p != "" {
				homePartsFiltered = append(homePartsFiltered, p)
			}
		}
		// Remove home directory prefix from parts
		if len(parts) > len(homePartsFiltered) {
			parts = parts[len(homePartsFiltered):]
		}
	}
	
	// If we have just one component, return it
	if len(parts) == 1 {
		return parts[0]
	}
	
	// For paths like code/project, return just the leaf
	if len(parts) == 2 {
		return parts[len(parts)-1]
	}
	
	// For paths like code/project/src, return last 2 segments: project/src
	if len(parts) == 3 {
		return filepath.Join(parts[len(parts)-2:]...)
	}
	
	// For longer paths like code/project/src/components,
	// return last 2 segments: src/components
	if len(parts) == 4 {
		return filepath.Join(parts[len(parts)-2:]...)
	}
	
	// For very deep paths, return last 3 segments
	return filepath.Join(parts[len(parts)-3:]...)
}

// extractTopActivities analyzes commands to identify primary activities
// Returns a concise activity summary like "git build", "docker", "search", etc.
func extractTopActivities(session *Session) string {
	if len(session.Commands) == 0 {
		return "work"
	}
	
	// Count base commands (first word of each command)
	cmdCounts := make(map[string]int)
	for _, cmd := range session.Commands {
		base := strings.ToLower(cmd.BaseCommand)
		// Clean up base command (remove ./ prefix, etc.)
		base = strings.TrimPrefix(base, "./")
		if base != "" {
			cmdCounts[base]++
		}
	}
	
	// Find top 2 commands
	type cmdCount struct {
		cmd   string
		count int
	}
	var topCmds []cmdCount
	for cmd, count := range cmdCounts {
		topCmds = append(topCmds, cmdCount{cmd, count})
	}
	
	// Sort by count (simple bubble sort for small lists)
	for i := 0; i < len(topCmds); i++ {
		for j := i + 1; j < len(topCmds); j++ {
			if topCmds[j].count > topCmds[i].count {
				topCmds[i], topCmds[j] = topCmds[j], topCmds[i]
			}
		}
	}
	
	// Build activity string
	if len(topCmds) == 0 {
		return "work"
	}
	
	// If one command dominates (>50%), use it alone
	if topCmds[0].count > len(session.Commands)/2 {
		return topCmds[0].cmd
	}
	
	// If we have 2+ distinct activities with reasonable counts
	if len(topCmds) > 1 && topCmds[1].count >= len(session.Commands)/5 {
		return topCmds[0].cmd + " " + topCmds[1].cmd
	}
	
	// Otherwise just return the top command
	return topCmds[0].cmd
}

// findMostActiveDirectory returns the directory with the most commands
func findMostActiveDirectory(session *Session) string {
	if len(session.Directories) == 0 {
		return ""
	}
	if len(session.Directories) == 1 {
		return session.Directories[0]
	}
	
	// Count commands per directory
	dirCounts := make(map[string]int)
	for _, cmd := range session.Commands {
		dirCounts[cmd.Directory]++
	}
	
	// Find most active
	maxDir := session.Directories[0]
	maxCount := 0
	for dir, count := range dirCounts {
		if count > maxCount {
			maxCount = count
			maxDir = dir
		}
	}
	
	return maxDir
}

func generateSessionDescription(session *Session) string {
	if len(session.Commands) == 0 {
		return "Empty session"
	}
	
	// Get activities and directory
	activities := extractTopActivities(session)
	primaryDir := findMostActiveDirectory(session)
	
	// Truncate directory path
	// We need to get home dir - use a reasonable default if not available
	homeDir := os.Getenv("HOME")
	if homeDir == "" {
		homeDir = "/Users/" + os.Getenv("USER")
	}
	shortDir := truncateDirectoryPath(primaryDir, homeDir)
	
	// Get primary categories
	categoryStr := ""
	if len(session.Categories) > 0 {
		// Get top 2 categories by count
		type catCount struct {
			cat   CommandCategory
			count int
		}
		var topCats []catCount
		for cat, count := range session.Categories {
			topCats = append(topCats, catCount{cat, count})
		}
		
		// Sort by count
		for i := 0; i < len(topCats); i++ {
			for j := i + 1; j < len(topCats); j++ {
				if topCats[j].count > topCats[i].count {
					topCats[i], topCats[j] = topCats[j], topCats[i]
				}
			}
		}
		
		// Take top 1-2 categories
		if len(topCats) > 0 {
			categoryStr = " [" + GetCategoryDisplayName(topCats[0].cat)
			if len(topCats) > 1 && topCats[1].count >= len(session.Commands)/5 {
				categoryStr += ", " + GetCategoryDisplayName(topCats[1].cat)
			}
			categoryStr += "]"
		}
	}
	
	// Generate description: "shortDir: activities [categories]"
	return shortDir + ": " + activities + categoryStr
}
