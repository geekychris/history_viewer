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
	// Reset cwd tracking whenever we see a time gap longer than this. Rationale:
	// zsh's history file is a single stream from every terminal + session on
	// the machine, so `cd` events interleave across shells. A large gap between
	// two consecutive entries almost always means "a different terminal
	// started" — its cwd is unrelated to whatever the previous terminal was in.
	// Absent this reset, the accumulated cwd drifts wildly.
	cwdResetGap := time.Duration(p.config.SessionTimeout)
	if cwdResetGap <= 0 {
		cwdResetGap = 30 * time.Minute
	}
	var lastTS time.Time

	// Local closure so both the main path and the read-ahead branch below share
	// the same cwd-tracking logic (including the time-gap reset).
	appendEntry := func(timestamp int64, duration int, command string) {
		ts := time.Unix(timestamp, 0)
		if !lastTS.IsZero() && ts.Sub(lastTS) > cwdResetGap {
			currentDir = p.config.HomeDir
		}
		lastTS = ts
		if cdMatches := cdRegex.FindStringSubmatch(command); len(cdMatches) > 1 {
			newDir := strings.TrimSpace(cdMatches[1])
			currentDir = p.resolveDirectory(currentDir, newDir)
		}
		entries = append(entries, HistoryEntry{
			ID:          id,
			Timestamp:   ts,
			Duration:    duration,
			Command:     command,
			Directory:   currentDir,
			Category:    CategorizeCommand(command),
			BaseCommand: GetBaseCommand(command),
		})
		id++
	}

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

			appendEntry(timestamp, duration, command)

			// If we read ahead to check for multiline, process that line now
			if historyLineRegex.MatchString(line) {
				matches = historyLineRegex.FindStringSubmatch(line)
				if len(matches) == 4 {
					ts2, _ := strconv.ParseInt(matches[1], 10, 64)
					dur2, _ := strconv.Atoi(matches[2])
					appendEntry(ts2, dur2, matches[3])
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

	// Resolve relative path, then sanitise the result — see sanitizeDir.
	return sanitizeDir(filepath.Clean(filepath.Join(currentDir, newDir)), p.config.HomeDir)
}

// sanitizeDir applies three defensive fixes to a resolved cwd. All of them
// exist because zsh's history file conflates every terminal's `cd` events
// into a single stream, so replaying them serially accumulates cwd drift
// from unrelated shells:
//
//  1. Collapse 3+ consecutive identical segments to one adjacent pair.
//     Two consecutive identicals is real (rare — e.g. `foo/foo`), three+
//     is almost always chained-cd drift.
//  2. If any single segment appears 3+ times in the whole path (adjacent
//     or not), the path is bogus — reset to home. Real project paths
//     don't have the same directory name appearing three times.
//  3. If the path is deeper than 30 segments, reset to home. macOS
//     working dirs don't run 30 levels deep in practice.
//
// All heuristics; the fundamental problem can only be bounded, not solved.
func sanitizeDir(dir, home string) string {
	if dir == "" {
		return home
	}
	parts := strings.Split(dir, string(filepath.Separator))

	// (1) Drop 3rd+ adjacent identical segments.
	out := make([]string, 0, len(parts))
	repeat := 0
	for i, p := range parts {
		if i > 0 && p == parts[i-1] && p != "" {
			repeat++
			if repeat >= 2 {
				continue
			}
		} else {
			repeat = 0
		}
		out = append(out, p)
	}
	cleaned := filepath.Clean(strings.Join(out, string(filepath.Separator)))

	// (2) Any single segment repeated 3+ times anywhere → bogus.
	counts := map[string]int{}
	for _, seg := range strings.Split(cleaned, string(filepath.Separator)) {
		if seg == "" {
			continue
		}
		counts[seg]++
		if counts[seg] >= 3 {
			return home
		}
	}

	// (3) Depth cap.
	depth := 0
	for _, seg := range strings.Split(cleaned, string(filepath.Separator)) {
		if seg != "" {
			depth++
		}
	}
	if depth > 30 {
		return home
	}
	return cleaned
}

func (p *Parser) GroupIntoSessions(entries []HistoryEntry, sessionIndex *SessionIndex) []Session {
	if len(entries) == 0 {
		return []Session{}
	}

	sessions := []Session{}
	currentSession := Session{
		ID:             "", // Will be set when session is finalized
		SequenceNumber: 1,
		StartTime:      entries[0].Timestamp,
		Commands:       []HistoryEntry{},
		Categories:     make(map[CommandCategory]int),
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
					
					// Generate stable ID from first command
					firstCmd := currentSession.Commands[0]
					stableID := sessionIndex.GetOrCreate(
						currentSession.StartTime,
						currentSession.EndTime,
						firstCmd.Command,
						currentSession.Description,
					)
					currentSession.ID = stableID
					currentSession.SequenceNumber = sessionIndex.GetSequenceNumber(stableID)
					
					// Update all commands in this session with the stable ID
					// Need to update both the session copy and the original entries slice
					for j := range currentSession.Commands {
						currentSession.Commands[j].SessionID = stableID
						// Also update the original entry in the entries slice
						for k := range entries {
							if entries[k].ID == currentSession.Commands[j].ID {
								entries[k].SessionID = stableID
								break
							}
						}
					}
					
					sessions = append(sessions, currentSession)
					
					// Start new session
					currentSession = Session{
						ID:             "",
						SequenceNumber: len(sessions) + 1,
						StartTime:      entry.Timestamp,
						Commands:       []HistoryEntry{},
						Categories:     make(map[CommandCategory]int),
					}
					dirSet = make(map[string]bool)
					consecutiveCategoryChanges = 0
				}
			}
		}
		
	lastCategory = entry.Category
	// Don't set SessionID yet - will be set when session is finalized
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
		
		// Generate stable ID from first command
		firstCmd := currentSession.Commands[0]
		stableID := sessionIndex.GetOrCreate(
			currentSession.StartTime,
			currentSession.EndTime,
			firstCmd.Command,
			currentSession.Description,
		)
		currentSession.ID = stableID
		currentSession.SequenceNumber = sessionIndex.GetSequenceNumber(stableID)
		
		// Update all commands in this session with the stable ID
		// Need to update both the session copy and the original entries slice
		for j := range currentSession.Commands {
			currentSession.Commands[j].SessionID = stableID
			// Also update the original entry in the entries slice
			for k := range entries {
				if entries[k].ID == currentSession.Commands[j].ID {
					entries[k].SessionID = stableID
					break
				}
			}
		}
		
		sessions = append(sessions, currentSession)
	}

	// Reassign sequence numbers to ensure they're in chronological order
	sessionIndex.ReassignSequenceNumbers()
	for i := range sessions {
		sessions[i].SequenceNumber = sessionIndex.GetSequenceNumber(sessions[i].ID)
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
