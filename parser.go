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

// pushdRegex — proper directory stack push. `pushd <path>` is like `cd`
// for our purposes; the fact that it also pushes to a stack matters only
// if we later see `popd`, which we handle separately.
var pushdRegex = regexp.MustCompile(`^\s*pushd\s+([^;&|]+)`)

// popdRegex — pop the most recent pushd. Restores the previous cwd.
var popdRegex = regexp.MustCompile(`^\s*popd(\s|$)`)

// gitCRegex — `git -C <path> ...` runs the git command with cwd=<path>.
// Doesn't change the caller's cwd (git returns to it), but it's a strong
// signal that <path> is a real dir the user is working in — good hint
// when the derived cwd looks stale.
var gitCRegex = regexp.MustCompile(`\bgit\s+-C\s+([^\s;&|]+)`)

// absPathRegex — any absolute path token in the command line. Used to
// correct drifted cwd: if the command mentions /Users/me/foo/bar.go but
// the derived cwd is somewhere unrelated, cwd was almost certainly wrong.
// Matches Unix-style abs paths only (no C:\, this is macOS/Linux).
var absPathRegex = regexp.MustCompile(`(?:^|\s|["'=(])((?:/[^\s"'()`+"`"+`;&|]+)+)`)

func (p *Parser) ParseHistory() ([]HistoryEntry, error) {
	file, err := os.Open(p.config.HistoryFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Ground truth from the preexec hook (see
	// scripts/history-viewer-cwd-hook.zsh). When available for a given
	// (timestamp, command) pair, it wins over cd-chain inference.
	cwdTruth := p.loadCwdHistory()

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

	// Proper directory stack for pushd/popd. Grows/shrinks alongside
	// currentDir; empty stack + popd is a no-op (matches shell behaviour).
	var dirStack []string
	// Local closure so both the main path and the read-ahead branch below
	// share the same cwd-tracking logic (time-gap reset, cd, pushd/popd,
	// git -C hints, absolute-path correction).
	appendEntry := func(timestamp int64, duration int, command string) {
		ts := time.Unix(timestamp, 0)

		// Ground-truth override: if the preexec hook logged this exact
		// (ts, command), use that PWD verbatim. Bypass all inference
		// (time-gap reset, cd inheritance, hint correction) — we know
		// where the command actually ran.
		hasTruth := false
		if cwdTruth != nil {
			if truth, ok := cwdTruth[cwdKey(timestamp, command)]; ok {
				currentDir = truth
				dirStack = nil
				hasTruth = true
			}
		}

		if !hasTruth {
			if !lastTS.IsZero() && ts.Sub(lastTS) > cwdResetGap {
				currentDir = p.config.HomeDir
				dirStack = nil
			}
		}
		lastTS = ts

		// 1. Explicit cd/pushd/popd — always applied so currentDir
		//    tracks correctly for the NEXT entry (whether or not this
		//    one had truth).
		if m := cdRegex.FindStringSubmatch(command); len(m) > 1 {
			currentDir = p.resolveDirectory(currentDir, strings.TrimSpace(m[1]))
		}
		if m := pushdRegex.FindStringSubmatch(command); len(m) > 1 {
			dirStack = append(dirStack, currentDir)
			currentDir = p.resolveDirectory(currentDir, strings.TrimSpace(m[1]))
		}
		if popdRegex.MatchString(command) && len(dirStack) > 0 {
			currentDir = dirStack[len(dirStack)-1]
			dirStack = dirStack[:len(dirStack)-1]
		}

		// 2. Hint-based correction — skip when we have truth.
		if !hasTruth {
			if hint := inferHintCwd(command, p.config.HomeDir); hint != "" {
				if !sharesPrefix(currentDir, hint) {
					currentDir = hint
				}
			}
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

			// Handle multi-line commands — read forward until we hit
			// either EOF or another history header line.
			readAhead := false
			for scanner.Scan() {
				nextLine := scanner.Text()
				if historyLineRegex.MatchString(nextLine) {
					line = nextLine
					readAhead = true
					break
				}
				command += "\n" + nextLine
			}

			appendEntry(timestamp, duration, command)

			// If we consumed an extra header line in the look-ahead,
			// process it now. Without the readAhead guard we'd
			// double-count the last entry when EOF closes the inner
			// loop with `line` still holding the current outer text.
			if readAhead {
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

// inferHintCwd extracts a "the user is probably in this directory" hint
// from a command by looking for:
//
//  1. `git -C <path>` — path is definitely a real dir the user cares about.
//  2. Any absolute path token that refers to an existing file — the
//     directory containing that file is a strong candidate for cwd.
//
// Returns "" when no confident hint is available. We deliberately keep
// this cautious: better to leave a possibly-stale cwd than to flip it on
// weak evidence (e.g., a command referencing a path in a shell script it
// happens to run).
func inferHintCwd(command, home string) string {
	// Prefer git -C when present — the strongest signal.
	if m := gitCRegex.FindStringSubmatch(command); len(m) > 1 {
		p := strings.Trim(m[1], `"'`)
		if strings.HasPrefix(p, "~/") {
			p = filepath.Join(home, p[2:])
		}
		if filepath.IsAbs(p) {
			if info, err := os.Stat(p); err == nil && info.IsDir() {
				return filepath.Clean(p)
			}
			// Even if the dir doesn't exist right now (user deleted it),
			// git -C is a strong intent signal.
			return filepath.Clean(p)
		}
	}
	// Any absolute path in the command that resolves to a real file.
	// We use the FIRST match (typically the most operative arg) and stop.
	for _, m := range absPathRegex.FindAllStringSubmatch(command, -1) {
		if len(m) < 2 {
			continue
		}
		p := strings.Trim(m[1], `"'`)
		if isSystemPath(p, home) {
			continue
		}
		if info, err := os.Stat(p); err == nil {
			if info.IsDir() {
				return filepath.Clean(p)
			}
			return filepath.Clean(filepath.Dir(p))
		}
	}
	return ""
}

// systemPathPrefixes are directories that appear all over shell history but
// almost never indicate the user's actual cwd. Excluded UNLESS the path
// lives under HOME (in which case it's clearly a user artefact — this also
// matters because macOS $TMPDIR is under /var/folders, so test tempdirs
// don't trip the blocklist).
var systemPathPrefixes = []string{
	"/usr", "/bin", "/sbin", "/tmp", "/opt", "/etc",
	"/var", "/System", "/Library", "/Applications", "/dev",
}

func isSystemPath(p, home string) bool {
	if home != "" && (p == home || strings.HasPrefix(p, home+"/")) {
		return false
	}
	for _, sp := range systemPathPrefixes {
		if p == sp || strings.HasPrefix(p, sp+"/") {
			return true
		}
	}
	return false
}

// loadCwdHistory reads the supplemental cwd log written by the preexec
// hook (see scripts/history-viewer-cwd-hook.zsh). Line format:
//
//	<epoch>\t<pwd>\t<cmd>
//
// Commands are stored with literal newlines escaped as `\n` and literal
// tabs as `\t` so each record fits on one physical line. This function
// reverses that encoding and returns a map keyed by (ts, cmd) so the
// parser can look up ground-truth cwd for a given history entry.
//
// Any I/O error or malformed line is silently ignored — the caller
// degrades gracefully to inference.
func (p *Parser) loadCwdHistory() map[string]string {
	if p.config.CwdHistoryFile == "" {
		return nil
	}
	f, err := os.Open(p.config.CwdHistoryFile)
	if err != nil {
		return nil
	}
	defer f.Close()

	out := make(map[string]string)
	sc := bufio.NewScanner(f)
	buf := make([]byte, 0, 64*1024)
	sc.Buffer(buf, 1024*1024)
	for sc.Scan() {
		parts := strings.SplitN(sc.Text(), "\t", 3)
		if len(parts) != 3 {
			continue
		}
		ts, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			continue
		}
		cmd := strings.ReplaceAll(parts[2], `\n`, "\n")
		cmd = strings.ReplaceAll(cmd, `\t`, "\t")
		out[cwdKey(ts, cmd)] = parts[1]
	}
	return out
}

// cwdKey builds the lookup key used by loadCwdHistory. NUL separator
// avoids collisions from any command text.
func cwdKey(ts int64, cmd string) string {
	return strconv.FormatInt(ts, 10) + "\x00" + cmd
}

// sharesPrefix reports whether a is a prefix of b or vice-versa (with
// slash-boundary respect so /a/b is not a prefix of /a/bad).
func sharesPrefix(a, b string) bool {
	a = strings.TrimRight(a, "/")
	b = strings.TrimRight(b, "/")
	if a == b {
		return true
	}
	if strings.HasPrefix(a, b+"/") || strings.HasPrefix(b, a+"/") {
		return true
	}
	return false
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
