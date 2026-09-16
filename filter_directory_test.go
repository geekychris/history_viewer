package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestHandleSessions_DirectoryFilter exercises the new directory query
// param on /api/sessions. We build a small in-memory Server with three
// synthetic sessions in different working directories and verify prefix
// matching.
func TestHandleSessions_DirectoryFilter(t *testing.T) {
	mkSession := func(id, dir string, cat CommandCategory) Session {
		return Session{
			ID:          id,
			StartTime:   time.Now().Add(-time.Hour),
			EndTime:     time.Now(),
			Description: "session " + id,
			Categories:  map[CommandCategory]int{cat: 1},
			Commands: []HistoryEntry{
				{ID: 1, Command: "git status", Directory: dir, Category: cat, Timestamp: time.Now()},
			},
		}
	}
	s := &Server{
		config: &Config{},
		sessions: []Session{
			mkSession("in-proj", "/Users/me/proj", CategoryVCS),
			mkSession("in-subdir", "/Users/me/proj/src", CategoryVCS),
			mkSession("elsewhere", "/tmp", CategoryVCS),
		},
	}

	call := func(dir string) []map[string]any {
		req := httptest.NewRequest(http.MethodGet, "/api/sessions?directory="+dir, nil)
		w := httptest.NewRecorder()
		s.handleSessions(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("directory=%q: status %d body=%s", dir, w.Code, w.Body.String())
		}
		var out []map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
			t.Fatal(err)
		}
		return out
	}

	// Match /Users/me/proj → both proj and proj/src.
	got := call("/Users/me/proj")
	if len(got) != 2 {
		t.Fatalf("prefix match: want 2 (proj + subdir), got %d", len(got))
	}

	// Trailing-slash variant still matches (server trims trailing /).
	got = call("/Users/me/proj/")
	if len(got) != 2 {
		t.Fatalf("prefix match (trailing /): want 2, got %d", len(got))
	}

	// More specific path only matches the subdir.
	got = call("/Users/me/proj/src")
	if len(got) != 1 {
		t.Fatalf("subdir match: want 1, got %d", len(got))
	}

	// Empty directory param = no filter → all 3.
	got = call("")
	if len(got) != 3 {
		t.Fatalf("empty filter: want 3, got %d", len(got))
	}

	// No match.
	got = call("/nowhere")
	if len(got) != 0 {
		t.Fatalf("no match: want 0, got %d", len(got))
	}
}
