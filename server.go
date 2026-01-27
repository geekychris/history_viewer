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
	config       *Config
	parser       *Parser
	ollama       *OllamaClient
	exporter     *Exporter
	sessions     []Session
	entries      []HistoryEntry
	lastModTime  time.Time
	mu           sync.RWMutex
}

func NewServer(config *Config) *Server {
	return &Server{
		config:   config,
		parser:   NewParser(config),
		ollama:   NewOllamaClient(config.OllamaURL, config.OllamaModel),
		exporter: NewExporter(),
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
		
		// Keyword filtering
		if keyword != "" {
			found := false
			keywordLower := strings.ToLower(keyword)
			// Search in description
			if strings.Contains(strings.ToLower(session.Description), keywordLower) {
				found = true
			}
			// Search in commands
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.entries)
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

	var sessions []Session
	if sessionIDStr != "" {
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
		sessions = s.sessions
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
