package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

type MetadataStore struct {
	Notes            map[string]Note            `json:"notes"`             // key: note ID
	Tags             map[string]Tag             `json:"tags"`              // key: tag ID
	SessionMetadatas map[string]SessionMetadata `json:"session_metadatas"` // key: metadata ID
	filePath         string
	mu               sync.RWMutex
}

func NewMetadataStore() (*MetadataStore, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	filePath := filepath.Join(homeDir, ".history_viewer_metadata.json")
	store := &MetadataStore{
		Notes:            make(map[string]Note),
		Tags:             make(map[string]Tag),
		SessionMetadatas: make(map[string]SessionMetadata),
		filePath:         filePath,
	}

	// Try to load existing metadata
	if err := store.load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to load metadata: %w", err)
	}

	return store, nil
}

func (m *MetadataStore) load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := os.ReadFile(m.filePath)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, m)
}

func (m *MetadataStore) save() error {
	// Note: Caller must hold the lock (write lock)
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(m.filePath, data, 0644)
}

// Note operations

func (m *MetadataStore) AddNote(targetType TargetType, targetID int, text string) (*Note, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	note := Note{
		ID:         uuid.New().String(),
		TargetType: targetType,
		TargetID:   targetID,
		Text:       text,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	m.Notes[note.ID] = note

	if err := m.save(); err != nil {
		return nil, err
	}

	return &note, nil
}

func (m *MetadataStore) UpdateNote(noteID string, text string) (*Note, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	note, exists := m.Notes[noteID]
	if !exists {
		return nil, fmt.Errorf("note not found: %s", noteID)
	}

	note.Text = text
	note.UpdatedAt = time.Now()
	m.Notes[noteID] = note

	if err := m.save(); err != nil {
		return nil, err
	}

	return &note, nil
}

func (m *MetadataStore) DeleteNote(noteID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.Notes[noteID]; !exists {
		return fmt.Errorf("note not found: %s", noteID)
	}

	delete(m.Notes, noteID)

	return m.save()
}

func (m *MetadataStore) GetNotesForTarget(targetType TargetType, targetID int) []Note {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var notes []Note
	for _, note := range m.Notes {
		if note.TargetType == targetType && note.TargetID == targetID {
			notes = append(notes, note)
		}
	}

	return notes
}

// Tag operations (now just keywords)

func (m *MetadataStore) AddTag(targetType TargetType, targetID int, keyword string) (*Tag, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if keyword == "" {
		return nil, fmt.Errorf("keyword is required")
	}

	tag := Tag{
		ID:         uuid.New().String(),
		TargetType: targetType,
		TargetID:   targetID,
		Keyword:    keyword,
	}

	m.Tags[tag.ID] = tag

	if err := m.save(); err != nil {
		return nil, err
	}

	return &tag, nil
}

func (m *MetadataStore) DeleteTag(tagID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.Tags[tagID]; !exists {
		return fmt.Errorf("tag not found: %s", tagID)
	}

	delete(m.Tags, tagID)

	return m.save()
}

func (m *MetadataStore) GetTagsForTarget(targetType TargetType, targetID int) []Tag {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var tags []Tag
	for _, tag := range m.Tags {
		if tag.TargetType == targetType && tag.TargetID == targetID {
			tags = append(tags, tag)
		}
	}

	return tags
}

// SessionMetadata operations

func (m *MetadataStore) SetSessionMetadata(targetType TargetType, targetID int, colorCode string, starRating int) (*SessionMetadata, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Validate star rating
	if starRating < 0 || starRating > 5 {
		return nil, fmt.Errorf("star rating must be between 0 and 5")
	}

	// Find existing metadata for this target
	var existingID string
	for id, meta := range m.SessionMetadatas {
		if meta.TargetType == targetType && meta.TargetID == targetID {
			existingID = id
			break
		}
	}

	var metadata SessionMetadata
	if existingID != "" {
		// Update existing
		metadata = m.SessionMetadatas[existingID]
		metadata.ColorCode = colorCode
		metadata.StarRating = starRating
	} else {
		// Create new
		metadata = SessionMetadata{
			ID:         uuid.New().String(),
			TargetType: targetType,
			TargetID:   targetID,
			ColorCode:  colorCode,
			StarRating: starRating,
		}
	}

	m.SessionMetadatas[metadata.ID] = metadata

	if err := m.save(); err != nil {
		return nil, err
	}

	return &metadata, nil
}

func (m *MetadataStore) GetSessionMetadata(targetType TargetType, targetID int) *SessionMetadata {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, meta := range m.SessionMetadatas {
		if meta.TargetType == targetType && meta.TargetID == targetID {
			return &meta
		}
	}

	return nil
}

// Merge metadata into sessions and commands

func (m *MetadataStore) MergeIntoSessions(sessions []Session) []Session {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for i := range sessions {
		// Add session notes, tags, and metadata
		sessions[i].Notes = m.GetNotesForTarget(TargetSession, sessions[i].ID)
		sessions[i].Tags = m.GetTagsForTarget(TargetSession, sessions[i].ID)
		sessions[i].Metadata = m.GetSessionMetadata(TargetSession, sessions[i].ID)

		// Add command notes and tags
		for j := range sessions[i].Commands {
			cmdID := sessions[i].Commands[j].ID
			sessions[i].Commands[j].Notes = m.GetNotesForTarget(TargetCommand, cmdID)
			sessions[i].Commands[j].Tags = m.GetTagsForTarget(TargetCommand, cmdID)
		}
	}

	return sessions
}

func (m *MetadataStore) MergeIntoCommands(commands []HistoryEntry) []HistoryEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for i := range commands {
		commands[i].Notes = m.GetNotesForTarget(TargetCommand, commands[i].ID)
		commands[i].Tags = m.GetTagsForTarget(TargetCommand, commands[i].ID)
	}

	return commands
}
