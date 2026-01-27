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
	return &Parser{config: config}
}

// Parse zsh history format: : <timestamp>:<duration>;<command>
var historyLineRegex = regexp.MustCompile(`^:\s*(\d+):(\d+);(.*)$`)
var cdRegex = regexp.MustCompile(`^\s*cd\s+(.+)$`)

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

func generateSessionDescription(session *Session) string {
	if len(session.Commands) == 0 {
		return "Empty session"
	}
	
	// Find the most common category
	var maxCategory CommandCategory
	maxCount := 0
	for cat, count := range session.Categories {
		if count > maxCount {
			maxCount = count
			maxCategory = cat
		}
	}
	
	// Generate a simple description
	primaryDir := ""
	if len(session.Directories) > 0 {
		primaryDir = session.Directories[0]
	}
	
	if maxCount > len(session.Commands)/2 {
		return string(maxCategory) + " in " + primaryDir
	}
	
	return "Mixed tasks in " + primaryDir
}
