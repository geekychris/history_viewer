package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// TestParser_CwdTruthOverridesInference — the whole point of the preexec
// hook. The zsh_history is riddled with cross-terminal cd drift, but the
// hook's log has the real cwd. When both agree on (ts, cmd), truth wins
// even if inference would produce something wildly different.
func TestParser_CwdTruthOverridesInference(t *testing.T) {
	home := t.TempDir()
	histFile := filepath.Join(home, ".zsh_history")
	truthFile := filepath.Join(home, ".zsh_cwd_history")

	// Inference would drift: cd into A, then B, no time gap → both build up.
	// Ground truth says the second command actually ran in a totally different
	// dir (simulating a different terminal writing to the same history).
	dirA := filepath.Join(home, "proj-a")
	dirB := filepath.Join(home, "proj-b")
	realDir := filepath.Join(home, "real-cwd-from-other-shell")
	for _, d := range []string{dirA, dirB, realDir} {
		_ = os.MkdirAll(d, 0o755)
	}

	histLines := []string{
		fmt.Sprintf(": 1000:0;cd %s", dirA),
		fmt.Sprintf(": 1001:0;cd %s", dirB), // inference would land here
		": 1002:0;ls",                       // truth: ran in realDir
	}
	_ = os.WriteFile(histFile, []byte(joinLines(histLines)), 0o644)

	// Truth log: only the third command has a truth entry.
	truthLines := []string{
		fmt.Sprintf("1002\t%s\tls", realDir),
	}
	_ = os.WriteFile(truthFile, []byte(joinLines(truthLines)), 0o644)

	cfg := &Config{
		HistoryFile:    histFile,
		CwdHistoryFile: truthFile,
		HomeDir:        home,
		SessionTimeout: 30 * 60 * 1_000_000_000,
	}
	entries, err := NewParser(cfg).ParseHistory()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Fatalf("want 3, got %d", len(entries))
	}
	// First two entries come from inference (cd A, then cd B).
	if entries[0].Directory != dirA {
		t.Errorf("entry 0 (cd A): want %q, got %q", dirA, entries[0].Directory)
	}
	if entries[1].Directory != dirB {
		t.Errorf("entry 1 (cd B): want %q, got %q", dirB, entries[1].Directory)
	}
	// Third entry: truth wins.
	if entries[2].Directory != realDir {
		t.Errorf("entry 2 (ls, with truth): want %q, got %q — truth override failed", realDir, entries[2].Directory)
	}
}

// TestParser_MissingCwdHistoryFallsBackToInference — with no truth file,
// behavior should be identical to pre-hook parsing.
func TestParser_MissingCwdHistoryFallsBackToInference(t *testing.T) {
	home := t.TempDir()
	histFile := filepath.Join(home, ".zsh_history")
	proj := filepath.Join(home, "proj")
	_ = os.MkdirAll(proj, 0o755)
	_ = os.WriteFile(histFile, []byte(fmt.Sprintf(": 100:0;cd %s\n: 101:0;ls\n", proj)), 0o644)

	cfg := &Config{
		HistoryFile:    histFile,
		CwdHistoryFile: filepath.Join(home, "does-not-exist"),
		HomeDir:        home,
		SessionTimeout: 30 * 60 * 1_000_000_000,
	}
	entries, err := NewParser(cfg).ParseHistory()
	if err != nil {
		t.Fatal(err)
	}
	if entries[1].Directory != proj {
		t.Errorf("no truth file: want inference cwd %q, got %q", proj, entries[1].Directory)
	}
}

// TestLoadCwdHistory_HandlesEscapedNewlinesAndTabs — the hook escapes
// \n and \t in commands so each record fits on one line; the loader
// must reverse the encoding cleanly.
func TestLoadCwdHistory_HandlesEscapedNewlinesAndTabs(t *testing.T) {
	home := t.TempDir()
	truthFile := filepath.Join(home, ".zsh_cwd_history")
	// A multi-line command (`echo a<newline>b`) and one with a literal tab.
	lines := []string{
		"100\t/x\techo a\\nb",
		"101\t/y\tprintf 'x\\ty'",
	}
	_ = os.WriteFile(truthFile, []byte(joinLines(lines)), 0o644)

	cfg := &Config{CwdHistoryFile: truthFile, HomeDir: home}
	m := NewParser(cfg).loadCwdHistory()
	if got := m[cwdKey(100, "echo a\nb")]; got != "/x" {
		t.Errorf("multi-line cmd lookup: got %q want %q", got, "/x")
	}
	if got := m[cwdKey(101, "printf 'x\ty'")]; got != "/y" {
		t.Errorf("tab-in-cmd lookup: got %q want %q", got, "/y")
	}
}

// TestParser_CwdTruthDoesNotBleedIntoNextEntry — truth for entry N sets
// cwd for entry N, but if entry N+1 has no truth, inference from cd
// still runs starting from truth (so `cd sub` after a truthed entry
// takes us to truth/sub, not $HOME/sub).
func TestParser_CwdTruthDoesNotBleedIntoNextEntry(t *testing.T) {
	home := t.TempDir()
	histFile := filepath.Join(home, ".zsh_history")
	truthFile := filepath.Join(home, ".zsh_cwd_history")
	real := filepath.Join(home, "real")
	sub := filepath.Join(real, "sub")
	_ = os.MkdirAll(sub, 0o755)

	histLines := []string{
		": 1000:0;ls",         // truth says /real
		": 1001:0;cd sub",     // no truth; inheritance from /real → /real/sub
	}
	_ = os.WriteFile(histFile, []byte(joinLines(histLines)), 0o644)
	_ = os.WriteFile(truthFile, []byte(fmt.Sprintf("1000\t%s\tls\n", real)), 0o644)

	cfg := &Config{
		HistoryFile:    histFile,
		CwdHistoryFile: truthFile,
		HomeDir:        home,
		SessionTimeout: 30 * 60 * 1_000_000_000,
	}
	entries, err := NewParser(cfg).ParseHistory()
	if err != nil {
		t.Fatal(err)
	}
	if entries[0].Directory != real {
		t.Errorf("truthed entry: want %q got %q", real, entries[0].Directory)
	}
	if entries[1].Directory != sub {
		t.Errorf("post-truth cd sub: want %q got %q", sub, entries[1].Directory)
	}
}
