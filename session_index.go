package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// SessionBoundary represents a persisted session with stable ID
type SessionBoundary struct {
	ID               string    `json:"id"`                // Stable hash-based ID
	SequenceNumber   int       `json:"sequence_number"`   // Display number (Session #5)
	StartTime        time.Time `json:"start_time"`
	EndTime          time.Time `json:"end_time"`
	FirstCommand     string    `json:"first_command"`
	FirstCommandHash string    `json:"first_command_hash"`
	Description      string    `json:"description"`
}

// SessionIndex manages stable session IDs and boundaries
type SessionIndex struct {
	mu         sync.RWMutex
	boundaries map[string]*SessionBoundary // ID -> Boundary
	filePath   string
}

// NewSessionIndex creates a new session index
func NewSessionIndex(configDir string) (*SessionIndex, error) {
	filePath := filepath.Join(configDir, "sessions.json")
	
	index := &SessionIndex{
		boundaries: make(map[string]*SessionBoundary),
		filePath:   filePath,
	}
	
	// Try to load existing index
	if err := index.Load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to load session index: %w", err)
	}
	
	return index, nil
}

// GenerateStableID creates a stable session ID from the first command
func GenerateStableID(timestamp time.Time, command string) string {
	// Create a hash from timestamp + command
	data := fmt.Sprintf("%d:%s", timestamp.Unix(), command)
	hash := sha256.Sum256([]byte(data))
	// Use first 12 chars of hex for readability
	return fmt.Sprintf("sess_%x", hash[:6])
}

// GetOrCreate returns existing session ID or creates a new one
func (si *SessionIndex) GetOrCreate(startTime time.Time, endTime time.Time, firstCommand string, description string) string {
	si.mu.Lock()
	defer si.mu.Unlock()
	
	stableID := GenerateStableID(startTime, firstCommand)
	
	if existing, found := si.boundaries[stableID]; found {
		// Update end time if it changed (session grew)
		if endTime.After(existing.EndTime) {
			existing.EndTime = endTime
			existing.Description = description
		}
		return stableID
	}
	
	// Create new boundary
	boundary := &SessionBoundary{
		ID:               stableID,
		SequenceNumber:   len(si.boundaries) + 1,
		StartTime:        startTime,
		EndTime:          endTime,
		FirstCommand:     firstCommand,
		FirstCommandHash: fmt.Sprintf("%x", sha256.Sum256([]byte(firstCommand))),
		Description:      description,
	}
	
	si.boundaries[stableID] = boundary
	return stableID
}

// GetSequenceNumber returns the display sequence number for a session ID
func (si *SessionIndex) GetSequenceNumber(sessionID string) int {
	si.mu.RLock()
	defer si.mu.RUnlock()
	
	if boundary, found := si.boundaries[sessionID]; found {
		return boundary.SequenceNumber
	}
	return 0
}

// GetByID returns a session boundary by ID
func (si *SessionIndex) GetByID(sessionID string) *SessionBoundary {
	si.mu.RLock()
	defer si.mu.RUnlock()
	
	return si.boundaries[sessionID]
}

// ReassignSequenceNumbers ensures sequence numbers are ordered by start time
func (si *SessionIndex) ReassignSequenceNumbers() {
	si.mu.Lock()
	defer si.mu.Unlock()
	
	// Convert to slice for sorting
	boundaries := make([]*SessionBoundary, 0, len(si.boundaries))
	for _, b := range si.boundaries {
		boundaries = append(boundaries, b)
	}
	
	// Sort by start time
	sort.Slice(boundaries, func(i, j int) bool {
		return boundaries[i].StartTime.Before(boundaries[j].StartTime)
	})
	
	// Reassign sequence numbers
	for i, b := range boundaries {
		b.SequenceNumber = i + 1
	}
}

// Load reads the session index from disk
func (si *SessionIndex) Load() error {
	si.mu.Lock()
	defer si.mu.Unlock()
	
	data, err := os.ReadFile(si.filePath)
	if err != nil {
		return err
	}
	
	var boundaries []*SessionBoundary
	if err := json.Unmarshal(data, &boundaries); err != nil {
		return fmt.Errorf("failed to parse session index: %w", err)
	}
	
	si.boundaries = make(map[string]*SessionBoundary)
	for _, b := range boundaries {
		si.boundaries[b.ID] = b
	}
	
	return nil
}

// Save writes the session index to disk
func (si *SessionIndex) Save() error {
	si.mu.RLock()
	defer si.mu.RUnlock()
	
	// Convert to slice
	boundaries := make([]*SessionBoundary, 0, len(si.boundaries))
	for _, b := range si.boundaries {
		boundaries = append(boundaries, b)
	}
	
	// Sort by sequence number for readability
	sort.Slice(boundaries, func(i, j int) bool {
		return boundaries[i].SequenceNumber < boundaries[j].SequenceNumber
	})
	
	data, err := json.MarshalIndent(boundaries, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal session index: %w", err)
	}
	
	// Ensure directory exists
	dir := filepath.Dir(si.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	
	if err := os.WriteFile(si.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write session index: %w", err)
	}
	
	return nil
}

// Prune removes old sessions beyond a certain age (optional cleanup)
func (si *SessionIndex) Prune(maxAge time.Duration) int {
	si.mu.Lock()
	defer si.mu.Unlock()
	
	cutoff := time.Now().Add(-maxAge)
	removed := 0
	
	for id, boundary := range si.boundaries {
		if boundary.EndTime.Before(cutoff) {
			delete(si.boundaries, id)
			removed++
		}
	}
	
	if removed > 0 {
		si.ReassignSequenceNumbers()
	}
	
	return removed
}
