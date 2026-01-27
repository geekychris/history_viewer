package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Server struct {
	Config       *Config
	config       *Config
	parser       *Parser
	ollama       *OllamaClient
	exporter     *Exporter
	metadata     *MetadataStore
	sessions     []Session
	entries      []HistoryEntry
	lastModTime  time.Time
	mu           sync.RWMutex
}

func NewServer(config *Config) *Server {
	metadata, err := NewMetadataStore()
	if err != nil {
		log.Printf("Warning: Failed to load metadata: %v", err)
	}
	
	return &Server{
		config:   config,
		parser:   NewParser(config),
		ollama:   NewOllamaClient(config.OllamaURL, config.OllamaModel),
		exporter: NewExporter(),
		metadata: metadata,
	}
}

func (s *Server) refreshData() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entries, err := s.parser.ParseHistory()
	if err != nil {
		return err
	}

	s.entries = entries
	s.sessions = s.parser.GroupIntoSessions(entries)
	s.lastModTime = time.Now()

	return nil
}

func (s *Server) Start() error {
	// Initial data load
	if err := s.refreshData(); err != nil {
		return fmt.Errorf("failed to load history: %w", err)
	}

	// Setup routes
	http.HandleFunc("/", s.handleIndex)
	http.HandleFunc("/api/sessions", s.handleSessions)
	http.HandleFunc("/api/sessions/", s.handleSessionDetail)
	http.HandleFunc("/api/commands", s.handleCommands)
	http.HandleFunc("/api/search", s.handleSearch)
	http.HandleFunc("/api/patterns", s.handlePatterns)
	http.HandleFunc("/api/stats", s.handleStats)
	http.HandleFunc("/api/volume", s.handleVolume)
	http.HandleFunc("/api/refresh", s.handleRefresh)
	http.HandleFunc("/api/export", s.handleExport)
	http.HandleFunc("/api/llm/analyze", s.handleLLMAnalyze)
	http.HandleFunc("/api/config", s.handleConfig)
	// Metadata routes
	http.HandleFunc("/api/metadata/notes", s.handleNotes)
	http.HandleFunc("/api/metadata/tags", s.handleTags)
	http.HandleFunc("/api/metadata/session", s.handleSessionMetadata)

	addr := fmt.Sprintf(":%d", s.config.Port)
	log.Printf("Starting history viewer on http://localhost%s\n", addr)
	return http.ListenAndServe(addr, s.corsMiddleware(http.DefaultServeMux))
}

func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(indexHTML))
}

func (s *Server) handleSessions(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Parse query parameters
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")
	category := r.URL.Query().Get("category")
	keyword := r.URL.Query().Get("keyword")
	sortOrder := r.URL.Query().Get("sort") // "asc" or "desc"
	tagKeyword := r.URL.Query().Get("tag_keyword")
	tagColor := r.URL.Query().Get("tag_color")
	tagStarsStr := r.URL.Query().Get("tag_stars")
	noteSearch := r.URL.Query().Get("note_search")

	// Filter sessions
	filteredSessions := make([]Session, 0)
	for _, session := range s.sessions {
		// Date filtering
		if startDate != "" {
			if start, err := time.Parse("2006-01-02", startDate); err == nil {
				if session.StartTime.Before(start) {
					continue
				}
			}
		}
		if endDate != "" {
			if end, err := time.Parse("2006-01-02", endDate); err == nil {
				// Add one day to include the entire end date
				end = end.Add(24 * time.Hour)
				if session.EndTime.After(end) {
					continue
				}
			}
		}
		
		// Category filtering
		if category != "" && category != "all" {
			found := false
			for cat := range session.Categories {
				if string(cat) == category {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		
		// Keyword filtering - split by whitespace and match all tokens
		if keyword != "" {
			// Split keyword into tokens
			tokens := strings.Fields(strings.ToLower(keyword))
			if len(tokens) == 0 {
				continue
			}
			
			// Check if all tokens are found in session
			allTokensFound := true
			for _, token := range tokens {
				tokenFound := false
				
				// Search in description
				if strings.Contains(strings.ToLower(session.Description), token) {
					tokenFound = true
				}
				
				// Search in commands if not found in description
				if !tokenFound {
					for _, cmd := range session.Commands {
						if strings.Contains(strings.ToLower(cmd.Command), token) {
							tokenFound = true
							break
						}
					}
				}
				
				if !tokenFound {
					allTokensFound = false
					break
				}
			}
			
	if !allTokensFound {
				continue
			}
		}
		
		filteredSessions = append(filteredSessions, session)
	}

	// Merge metadata into sessions first, so we can filter by notes and tags
	if s.metadata != nil {
		filteredSessions = s.metadata.MergeIntoSessions(filteredSessions)
	}

	// Filter by tag keywords
	if tagKeyword != "" {
		filteredByTags := make([]Session, 0)
		for _, session := range filteredSessions {
			hasMatchingTag := false

			// Check session tags
			for _, tag := range session.Tags {
				if strings.Contains(strings.ToLower(tag.Keyword), strings.ToLower(tagKeyword)) {
					hasMatchingTag = true
					break
				}
			}

			// Check command tags if no session tag matched
			if !hasMatchingTag {
				for _, cmd := range session.Commands {
					for _, tag := range cmd.Tags {
						if strings.Contains(strings.ToLower(tag.Keyword), strings.ToLower(tagKeyword)) {
							hasMatchingTag = true
							break
						}
					}
					if hasMatchingTag {
						break
					}
				}
			}

			if hasMatchingTag {
				filteredByTags = append(filteredByTags, session)
			}
		}
		filteredSessions = filteredByTags
	}

	// Filter by color (from session metadata)
	if tagColor != "" {
		filteredByColor := make([]Session, 0)
		for _, session := range filteredSessions {
			if session.Metadata != nil && strings.EqualFold(session.Metadata.ColorCode, tagColor) {
				filteredByColor = append(filteredByColor, session)
			}
		}
		filteredSessions = filteredByColor
	}

	// Filter by star rating (from session metadata)
	if tagStarsStr != "" {
		tagStars, _ := strconv.Atoi(tagStarsStr)
		if tagStars > 0 {
			filteredByStars := make([]Session, 0)
			for _, session := range filteredSessions {
				if session.Metadata != nil && session.Metadata.StarRating >= tagStars {
					filteredByStars = append(filteredByStars, session)
				}
			}
			filteredSessions = filteredByStars
		}
	}

	// Filter by notes
	if noteSearch != "" {
		filteredByNotes := make([]Session, 0)
		for _, session := range filteredSessions {
			hasMatchingNote := false

			// Check session notes
			for _, note := range session.Notes {
				if strings.Contains(strings.ToLower(note.Text), strings.ToLower(noteSearch)) {
					hasMatchingNote = true
					break
				}
			}

			// Check command notes if no session note matched
			if !hasMatchingNote {
				for _, cmd := range session.Commands {
					for _, note := range cmd.Notes {
						if strings.Contains(strings.ToLower(note.Text), strings.ToLower(noteSearch)) {
							hasMatchingNote = true
							break
						}
					}
					if hasMatchingNote {
						break
					}
				}
			}

			if hasMatchingNote {
				filteredByNotes = append(filteredByNotes, session)
			}
		}
		filteredSessions = filteredByNotes
	}

	// Sort sessions
	if sortOrder == "asc" {
		// Already in ascending order (oldest first)
	} else {
		// Default: descending order (newest first)
		for i := 0; i < len(filteredSessions)/2; i++ {
			j := len(filteredSessions) - 1 - i
			filteredSessions[i], filteredSessions[j] = filteredSessions[j], filteredSessions[i]
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(filteredSessions)
}

func (s *Server) handleSessionDetail(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Extract session ID from URL
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/sessions/"), "/")
	if len(pathParts) == 0 {
		http.Error(w, "Session ID required", http.StatusBadRequest)
		return
	}

	sessionID, err := strconv.Atoi(pathParts[0])
	if err != nil {
		http.Error(w, "Invalid session ID", http.StatusBadRequest)
		return
	}

	for _, session := range s.sessions {
		if session.ID == sessionID {
			// Merge metadata
			if s.metadata != nil {
				sessions := s.metadata.MergeIntoSessions([]Session{session})
				if len(sessions) > 0 {
					session = sessions[0]
				}
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(session)
			return
		}
	}

	http.Error(w, "Session not found", http.StatusNotFound)
}

func (s *Server) handleCommands(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries := s.entries
	if s.metadata != nil {
		entries = s.metadata.MergeIntoCommands(entries)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entries)
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "Query parameter 'q' required", http.StatusBadRequest)
		return
	}

	query = strings.ToLower(query)
	var results []HistoryEntry

	for _, entry := range s.entries {
		if strings.Contains(strings.ToLower(entry.Command), query) ||
			strings.Contains(strings.ToLower(entry.Directory), query) {
			results = append(results, entry)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func (s *Server) handlePatterns(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Build command patterns
	patterns := make(map[string]*CommandPattern)

	for _, entry := range s.entries {
		base := entry.BaseCommand
		if _, exists := patterns[base]; !exists {
			patterns[base] = &CommandPattern{
				Command:      base,
				Count:        0,
				CoOccurrence: make(map[string]int),
				Categories:   make(map[CommandCategory]int),
			}
		}

		patterns[base].Count++
		patterns[base].Categories[entry.Category]++
	}

	// Build co-occurrence within sessions
	for _, session := range s.sessions {
		commandsInSession := make(map[string]bool)
		for _, cmd := range session.Commands {
			commandsInSession[cmd.BaseCommand] = true
		}

		// For each pair of commands in the session, increment co-occurrence
		for cmd1 := range commandsInSession {
			for cmd2 := range commandsInSession {
				if cmd1 != cmd2 {
					if pattern, exists := patterns[cmd1]; exists {
						pattern.CoOccurrence[cmd2]++
					}
				}
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(patterns)
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	categoryStats := make(map[CommandCategory]int)
	for _, entry := range s.entries {
		categoryStats[entry.Category]++
	}

	stats := map[string]interface{}{
		"total_commands":  len(s.entries),
		"total_sessions":  len(s.sessions),
		"categories":      categoryStats,
		"last_updated":    s.lastModTime,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (s *Server) handleVolume(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.entries) == 0 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]interface{}{})
		return
	}

	// Group commands by day
	volume := make(map[string]int)
	for _, entry := range s.entries {
		dateKey := entry.Timestamp.Format("2006-01-02")
		volume[dateKey]++
	}

	// Convert to array sorted by date
	type VolumeData struct {
		Date  string `json:"date"`
		Count int    `json:"count"`
	}

	var volumeData []VolumeData
	for date, count := range volume {
		volumeData = append(volumeData, VolumeData{Date: date, Count: count})
	}

	// Sort by date
	sort.Slice(volumeData, func(i, j int) bool {
		return volumeData[i].Date < volumeData[j].Date
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(volumeData)
}

func (s *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	if err := s.refreshData(); err != nil {
		http.Error(w, fmt.Sprintf("Failed to refresh: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "message": "Data refreshed"})
}

func (s *Server) handleExport(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	format := r.URL.Query().Get("format")
	sessionIDStr := r.URL.Query().Get("session")

	// Get filter parameters (same as handleSessions)
	startDate := r.URL.Query().Get("startDate")
	endDate := r.URL.Query().Get("endDate")
	category := r.URL.Query().Get("category")
	keyword := r.URL.Query().Get("keyword")
	noteSearch := r.URL.Query().Get("noteSearch")
	tagKeyword := r.URL.Query().Get("tagKeyword")
	tagStarsStr := r.URL.Query().Get("tagStars")

	var sessions []Session
	if sessionIDStr != "" {
		// Export specific session
		sessionID, err := strconv.Atoi(sessionIDStr)
		if err == nil {
			for _, session := range s.sessions {
				if session.ID == sessionID {
					sessions = []Session{session}
					break
				}
			}
		}
	} else {
		// Apply filters (same logic as handleSessions)
		filteredSessions := make([]Session, 0)
		for _, session := range s.sessions {
			// Date filtering
			if startDate != "" {
				if start, err := time.Parse("2006-01-02", startDate); err == nil {
					if session.StartTime.Before(start) {
						continue
					}
				}
			}
			if endDate != "" {
				if end, err := time.Parse("2006-01-02", endDate); err == nil {
					end = end.Add(24 * time.Hour)
					if session.EndTime.After(end) {
						continue
					}
				}
			}

			// Category filtering
			if category != "" && category != "all" {
				found := false
				for cat := range session.Categories {
					if string(cat) == category {
						found = true
						break
					}
				}
				if !found {
					continue
				}
			}

			// Keyword filtering
			if keyword != "" {
				tokens := strings.Fields(strings.ToLower(keyword))
				allTokensFound := true
				for _, token := range tokens {
					tokenFound := false
					if strings.Contains(strings.ToLower(session.Description), token) {
						tokenFound = true
					}
					if !tokenFound {
						for _, cmd := range session.Commands {
							if strings.Contains(strings.ToLower(cmd.Command), token) {
								tokenFound = true
								break
							}
						}
					}
					if !tokenFound {
						allTokensFound = false
						break
					}
				}
				if !allTokensFound {
					continue
				}
			}

			filteredSessions = append(filteredSessions, session)
		}

		// Merge metadata for filtered sessions
		if s.metadata != nil {
			filteredSessions = s.metadata.MergeIntoSessions(filteredSessions)
		}

		// Filter by tag keywords
		if tagKeyword != "" {
			temp := make([]Session, 0)
			for _, session := range filteredSessions {
				hasMatchingTag := false
				for _, tag := range session.Tags {
					if strings.Contains(strings.ToLower(tag.Keyword), strings.ToLower(tagKeyword)) {
						hasMatchingTag = true
						break
					}
				}
				if !hasMatchingTag {
					for _, cmd := range session.Commands {
						for _, tag := range cmd.Tags {
							if strings.Contains(strings.ToLower(tag.Keyword), strings.ToLower(tagKeyword)) {
								hasMatchingTag = true
								break
							}
						}
						if hasMatchingTag {
							break
						}
					}
				}
				if hasMatchingTag {
					temp = append(temp, session)
				}
			}
			filteredSessions = temp
		}

		// Filter by star rating
		if tagStarsStr != "" {
			tagStars, _ := strconv.Atoi(tagStarsStr)
			if tagStars > 0 {
				temp := make([]Session, 0)
				for _, session := range filteredSessions {
					if session.Metadata != nil && session.Metadata.StarRating >= tagStars {
						temp = append(temp, session)
					}
				}
				filteredSessions = temp
			}
		}

		// Filter by notes
		if noteSearch != "" {
			temp := make([]Session, 0)
			for _, session := range filteredSessions {
				hasMatchingNote := false
				for _, note := range session.Notes {
					if strings.Contains(strings.ToLower(note.Text), strings.ToLower(noteSearch)) {
						hasMatchingNote = true
						break
					}
				}
				if !hasMatchingNote {
					for _, cmd := range session.Commands {
						for _, note := range cmd.Notes {
							if strings.Contains(strings.ToLower(note.Text), strings.ToLower(noteSearch)) {
								hasMatchingNote = true
								break
							}
						}
						if hasMatchingNote {
							break
						}
					}
				}
				if hasMatchingNote {
					temp = append(temp, session)
				}
			}
			filteredSessions = temp
		}

		sessions = filteredSessions
	}

	var content string
	var contentType string
	var filename string

	switch format {
	case "json":
		data, err := s.exporter.ExportJSON(sessions)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		content = data
		contentType = "application/json"
		filename = "history.json"

	case "csv":
		data, err := s.exporter.ExportCSV(sessions)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		content = data
		contentType = "text/csv"
		filename = "history.csv"

	case "markdown", "md":
		content = s.exporter.ExportMarkdown(sessions)
		contentType = "text/markdown"
		filename = "history.md"

	case "zsh":
		content = s.exporter.ExportSessionsToZsh(sessions)
		contentType = "text/plain"
		filename = "history.zsh"

	default:
		http.Error(w, "Invalid format. Use: json, csv, markdown, or zsh", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	w.Write([]byte(content))
}

func (s *Server) handleLLMAnalyze(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Action       string   `json:"action"`
		Commands     []string `json:"commands"`
		SessionID    int      `json:"session_id"`
		CustomPrompt string   `json:"custom_prompt"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	var result string
	var err error

	// Handle custom prompt
	if req.CustomPrompt != "" {
		result, err = s.ollama.Generate(req.CustomPrompt)
	} else if req.SessionID > 0 {
		s.mu.RLock()
		var session *Session
		for i := range s.sessions {
			if s.sessions[i].ID == req.SessionID {
				session = &s.sessions[i]
				break
			}
		}
		s.mu.RUnlock()

		if session == nil {
			http.Error(w, "Session not found", http.StatusNotFound)
			return
		}

		result, err = s.ollama.AnalyzeSession(req.Action, session)
	} else if len(req.Commands) > 0 {
		result, err = s.ollama.AnalyzeCommands(req.Action, req.Commands)
	} else {
		http.Error(w, "Either commands, session_id, or custom_prompt must be provided", http.StatusBadRequest)
		return
	}

	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": fmt.Sprintf("LLM error: %v", err),
		})
		return
	}

	response := map[string]string{
		"result": result,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		s.mu.RLock()
		defer s.mu.RUnlock()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(s.config)
		return
	}

	if r.Method == "PUT" {
		var newConfig Config
		if err := json.NewDecoder(r.Body).Decode(&newConfig); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		s.mu.Lock()
		s.config = &newConfig
		s.parser = NewParser(s.config)
		s.ollama = NewOllamaClient(s.config.OllamaURL, s.config.OllamaModel)
		s.mu.Unlock()

		if err := SaveConfig(&newConfig); err != nil {
			log.Printf("Failed to save config: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func (s *Server) handleNotes(w http.ResponseWriter, r *http.Request) {
	if s.metadata == nil {
		http.Error(w, "Metadata store not available", http.StatusInternalServerError)
		return
	}

	switch r.Method {
	case "POST":
		var req struct {
			TargetType string `json:"target_type"`
			TargetID   int    `json:"target_id"`
			Text       string `json:"text"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if req.Text == "" {
			http.Error(w, "Note text is required", http.StatusBadRequest)
			return
		}

		targetType := TargetType(req.TargetType)
		if targetType != TargetSession && targetType != TargetCommand {
			http.Error(w, "Invalid target_type. Must be 'session' or 'command'", http.StatusBadRequest)
			return
		}

		note, err := s.metadata.AddNote(targetType, req.TargetID, req.Text)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to add note: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(note)

	case "PUT":
		var req struct {
			NoteID string `json:"note_id"`
			Text   string `json:"text"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if req.NoteID == "" || req.Text == "" {
			http.Error(w, "note_id and text are required", http.StatusBadRequest)
			return
		}

		note, err := s.metadata.UpdateNote(req.NoteID, req.Text)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to update note: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(note)

	case "DELETE":
		noteID := r.URL.Query().Get("note_id")
		if noteID == "" {
			http.Error(w, "note_id query parameter is required", http.StatusBadRequest)
			return
		}

		if err := s.metadata.DeleteNote(noteID); err != nil {
			http.Error(w, fmt.Sprintf("Failed to delete note: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok", "message": "Note deleted"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleTags(w http.ResponseWriter, r *http.Request) {
	if s.metadata == nil {
		http.Error(w, "Metadata store not available", http.StatusInternalServerError)
		return
	}

	switch r.Method {
	case "POST":
		var req struct {
			TargetType string `json:"target_type"`
			TargetID   int    `json:"target_id"`
			Keyword    string `json:"keyword"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		targetType := TargetType(req.TargetType)
		if targetType != TargetSession && targetType != TargetCommand {
			http.Error(w, "Invalid target_type. Must be 'session' or 'command'", http.StatusBadRequest)
			return
		}

		if req.Keyword == "" {
			http.Error(w, "keyword is required", http.StatusBadRequest)
			return
		}

		tag, err := s.metadata.AddTag(targetType, req.TargetID, req.Keyword)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to add tag: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(tag)

	case "DELETE":
		tagID := r.URL.Query().Get("tag_id")
		if tagID == "" {
			http.Error(w, "tag_id query parameter is required", http.StatusBadRequest)
			return
		}

		if err := s.metadata.DeleteTag(tagID); err != nil {
			http.Error(w, fmt.Sprintf("Failed to delete tag: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok", "message": "Tag deleted"})

	default:
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleSessionMetadata(w http.ResponseWriter, r *http.Request) {
	if s.metadata == nil {
		http.Error(w, "Metadata store not available", http.StatusInternalServerError)
		return
	}

	switch r.Method {
	case "POST", "PUT":
		var req struct {
			TargetType string `json:"target_type"`
			TargetID   int    `json:"target_id"`
			ColorCode  string `json:"color_code"`
			StarRating int    `json:"star_rating"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		targetType := TargetType(req.TargetType)
		if targetType != TargetSession && targetType != TargetCommand {
			http.Error(w, "Invalid target_type. Must be 'session' or 'command'", http.StatusBadRequest)
			return
		}

		metadata, err := s.metadata.SetSessionMetadata(targetType, req.TargetID, req.ColorCode, req.StarRating)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to set metadata: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(metadata)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// Helper methods for native UI

func (s *Server) ParseHistory() error {
	return s.refreshData()
}

func (s *Server) GetSessions(startDate, endDate, category, keyword string) []*Session {
	s.mu.RLock()
	defer s.mu.RUnlock()

	filteredSessions := make([]*Session, 0)
	for i := range s.sessions {
		session := &s.sessions[i]
		
		// Date filtering
		if startDate != "" {
			if start, err := time.Parse("2006-01-02", startDate); err == nil {
				if session.StartTime.Before(start) {
					continue
				}
			}
		}
		if endDate != "" {
			if end, err := time.Parse("2006-01-02", endDate); err == nil {
				end = end.Add(24 * time.Hour)
				if session.EndTime.After(end) {
					continue
				}
			}
		}
		
		// Category filtering
		if category != "" && category != "all" {
			found := false
			for cat := range session.Categories {
				if string(cat) == category {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		
		// Keyword filtering
		if keyword != "" {
			found := false
			keywordLower := strings.ToLower(keyword)
			if strings.Contains(strings.ToLower(session.Description), keywordLower) {
				found = true
			}
			if !found {
				for _, cmd := range session.Commands {
					if strings.Contains(strings.ToLower(cmd.Command), keywordLower) {
						found = true
						break
					}
				}
			}
			if !found {
				continue
			}
		}
		
		filteredSessions = append(filteredSessions, session)
	}

	return filteredSessions
}

func (s *Server) ExportSessions(sessions []*Session, format string) ([]byte, error) {
	// Convert pointers to values for exporter
	valueSessions := make([]Session, len(sessions))
	for i, session := range sessions {
		valueSessions[i] = *session
	}

	switch format {
	case "json":
		return json.MarshalIndent(valueSessions, "", "  ")
	case "csv":
		return []byte(s.exporter.ToCSV(valueSessions)), nil
	case "markdown":
		return []byte(s.exporter.ToMarkdown(valueSessions)), nil
	case "zsh":
		return []byte(s.exporter.ToZshHistory(valueSessions)), nil
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}

func (s *Server) ConvertToScript(sessions []*Session, lang string) string {
	// Collect all commands from sessions
	var commands []string
	for _, session := range sessions {
		for _, cmd := range session.Commands {
			commands = append(commands, cmd.Command)
		}
	}

	switch lang {
	case "bash":
		return s.exporter.ToBashScript(commands)
	case "python":
		return s.exporter.ToPythonScript(commands)
	case "java":
		return s.exporter.ToJavaProgram(commands)
	case "go":
		return s.exporter.ToGoProgram(commands)
	default:
		return ""
	}
}

func (s *Server) AnalyzeSession(sessionID string, action string, customPrompt string) (string, error) {
	s.mu.RLock()
	var session *Session
	sessionIDInt, err := strconv.Atoi(sessionID)
	if err != nil {
		s.mu.RUnlock()
		return "", fmt.Errorf("invalid session ID: %w", err)
	}
	
	for i := range s.sessions {
		if s.sessions[i].ID == sessionIDInt {
			session = &s.sessions[i]
			break
		}
	}
	s.mu.RUnlock()

	if session == nil {
		return "", fmt.Errorf("session not found")
	}

	if customPrompt != "" {
		return s.ollama.AnalyzeSessionWithPrompt(customPrompt, session)
	}

	return s.ollama.AnalyzeSession(action, session)
}
