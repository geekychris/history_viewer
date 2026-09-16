package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// resetFilterState wipes the package-global currentFilter so tests don't
// bleed into each other. Kept in-package (same-package test).
func resetFilterState() {
	currentFilter.mu.Lock()
	currentFilter.dir = ""
	currentFilter.version = 0
	currentFilter.updated = currentFilter.updated.Add(0) // no-op; zero-out below
	currentFilter.mu.Unlock()
}

// filterAPIServer builds a stripped-down Server just registered for the
// /api/filter/* routes.  Real Server pulls Ollama + parser + store; we don't
// need any of that for these endpoints. Does NOT reset filter state —
// callers that need a clean slate call resetFilterState() explicitly, and
// callers that want to test seeding call SeedFilterFromConfig before this.
func filterAPIServer(t *testing.T) http.Handler {
	t.Helper()
	s := &Server{}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/filter/directory", s.handleSetFilterDirectory)
	mux.HandleFunc("/api/filter/current", s.handleGetCurrentFilter)
	return mux
}

func TestSeedFilterFromConfig(t *testing.T) {
	resetFilterState()
	cfg := &Config{InitialDirFilter: "/Users/me/proj"}
	SeedFilterFromConfig(cfg)

	h := filterAPIServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/filter/current", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /current: status %d", w.Code)
	}
	var got map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["dir"] != "/Users/me/proj" {
		t.Fatalf("dir: want /Users/me/proj, got %v", got["dir"])
	}
	if got["version"] != float64(1) {
		t.Fatalf("version: want 1, got %v", got["version"])
	}
}

func TestSeedFilterFromConfig_EmptyIsNoop(t *testing.T) {
	resetFilterState()
	SeedFilterFromConfig(&Config{InitialDirFilter: ""})
	h := filterAPIServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/filter/current", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	var got map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	if got["dir"] != "" && got["dir"] != nil {
		t.Fatalf("empty seed should not populate dir; got %v", got["dir"])
	}
	if got["version"] != float64(0) {
		t.Fatalf("version: want 0, got %v", got["version"])
	}
}

func TestPostFilterDirectory_BumpsVersion(t *testing.T) {
	resetFilterState()
	h := filterAPIServer(t)

	// v1: POST with a dir.
	body := bytes.NewBufferString(`{"dir":"/tmp/one"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/filter/directory", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("POST v1: status %d body=%s", w.Code, w.Body.String())
	}
	var r1 map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &r1)
	if r1["dir"] != "/tmp/one" || r1["version"] != float64(1) {
		t.Fatalf("v1 response: %+v", r1)
	}

	// v2: POST a different dir, version should bump.
	body = bytes.NewBufferString(`{"dir":"/tmp/two"}`)
	req = httptest.NewRequest(http.MethodPost, "/api/filter/directory", body)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	var r2 map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &r2)
	if r2["dir"] != "/tmp/two" || r2["version"] != float64(2) {
		t.Fatalf("v2 response: %+v", r2)
	}

	// GET /current reflects the latest.
	req = httptest.NewRequest(http.MethodGet, "/api/filter/current", nil)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	var g map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &g)
	if g["dir"] != "/tmp/two" || g["version"] != float64(2) {
		t.Fatalf("after two POSTs, GET returned %+v", g)
	}
}

func TestPostFilterDirectory_EmptyDirClears(t *testing.T) {
	resetFilterState()
	h := filterAPIServer(t)
	// Seed a value first.
	_ = doPost(t, h, `{"dir":"/starting"}`)
	// Now clear.
	r := doPost(t, h, `{"dir":""}`)
	if r["dir"] != "" {
		t.Fatalf("expected empty dir after clear, got %v", r["dir"])
	}
	if r["version"] != float64(2) {
		t.Fatalf("clear should still bump version; got %v", r["version"])
	}
}

func TestPostFilterDirectory_RejectsGET(t *testing.T) {
	h := filterAPIServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/filter/directory", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET on POST-only endpoint: status %d (want 405)", w.Code)
	}
}

func TestPostFilterDirectory_InvalidJSON(t *testing.T) {
	h := filterAPIServer(t)
	req := httptest.NewRequest(http.MethodPost, "/api/filter/directory", strings.NewReader("{not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("invalid JSON: status %d (want 400)", w.Code)
	}
}

// helper — POST and unmarshal the response.
func doPost(t *testing.T, h http.Handler, body string) map[string]any {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/filter/directory", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("POST %s: status %d body=%s", body, w.Code, w.Body.String())
	}
	var m map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &m); err != nil {
		t.Fatal(err)
	}
	return m
}
