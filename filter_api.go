package main

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// filterState is the shared state for the /api/filter/* endpoints.
//
// External tools (Chief) POST to /api/filter/directory to focus the UI on
// a specific working directory. The frontend polls /api/filter/current
// every few seconds; whenever `version` increments, it applies the new
// filter in place — no reload, no scroll reset.
//
// The state is process-global rather than per-connection because there is
// at most one Wails webview per server instance, and this keeps external
// clients from having to correlate a specific browser tab.
type filterState struct {
	mu      sync.RWMutex
	dir     string
	updated time.Time
	version int64
}

var currentFilter filterState

// handleSetFilterDirectory: POST /api/filter/directory
//
//	Request:  {"dir": "/Users/me/code/my-project"}
//	Response: {"dir": "...", "version": N, "updated": "RFC3339"}
//
// Bumps the version so a frontend polling /api/filter/current will notice
// on its next tick and apply the filter.
func (s *Server) handleSetFilterDirectory(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		return // CORS preflight handled by middleware
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Dir string `json:"dir"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	// Trim + accept empty string as "clear filter".
	currentFilter.mu.Lock()
	currentFilter.dir = body.Dir
	currentFilter.updated = time.Now().UTC()
	currentFilter.version++
	v := currentFilter.version
	u := currentFilter.updated
	currentFilter.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"dir":     body.Dir,
		"version": v,
		"updated": u.Format(time.RFC3339),
	})
}

// handleGetCurrentFilter: GET /api/filter/current
//
//	Response: {"dir": "...", "version": N, "updated": "RFC3339"}
//
// The frontend calls this on load and on a 3-second poll interval. If
// `version` has advanced since the last observed value, it applies the
// new filter (typing into #directoryPathInput and firing searchDirectory).
func (s *Server) handleGetCurrentFilter(w http.ResponseWriter, r *http.Request) {
	currentFilter.mu.RLock()
	dir := currentFilter.dir
	v := currentFilter.version
	u := currentFilter.updated
	currentFilter.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	if u.IsZero() {
		json.NewEncoder(w).Encode(map[string]any{"dir": dir, "version": v})
		return
	}
	json.NewEncoder(w).Encode(map[string]any{
		"dir":     dir,
		"version": v,
		"updated": u.Format(time.RFC3339),
	})
}

// SeedFilterFromConfig lets main.go pre-populate the filter with the value
// from the --filter-dir CLI flag so a fresh page load (or the very first
// poll from a subscribing frontend) already knows about it.
func SeedFilterFromConfig(cfg *Config) {
	if cfg == nil || cfg.InitialDirFilter == "" {
		return
	}
	currentFilter.mu.Lock()
	currentFilter.dir = cfg.InitialDirFilter
	currentFilter.updated = time.Now().UTC()
	currentFilter.version = 1
	currentFilter.mu.Unlock()
}
