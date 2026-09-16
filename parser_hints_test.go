package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestInferHintCwd_GitDashC covers the strongest signal.
func TestInferHintCwd_GitDashC(t *testing.T) {
	home := t.TempDir()
	proj := filepath.Join(home, "proj")
	_ = os.MkdirAll(proj, 0o755)

	cases := []struct {
		cmd  string
		want string
	}{
		{"git -C " + proj + " status", proj},
		{"git -C \"" + proj + "\" log", proj},
		{"git -C ~/proj status", proj}, // ~ expansion
		{"cd foo && git -C " + proj + " diff", proj},
	}
	for _, c := range cases {
		if got := inferHintCwd(c.cmd, home); got != c.want {
			t.Errorf("inferHintCwd(%q): got %q want %q", c.cmd, got, c.want)
		}
	}
}

// TestInferHintCwd_AbsPathToRealFile — cwd hint derived from any absolute
// path in the command that maps to a real file/dir on disk.
func TestInferHintCwd_AbsPathToRealFile(t *testing.T) {
	home := t.TempDir()
	proj := filepath.Join(home, "code", "myproj")
	_ = os.MkdirAll(proj, 0o755)
	file := filepath.Join(proj, "main.go")
	_ = os.WriteFile(file, []byte("package main"), 0o644)

	// File path → containing dir.
	if got := inferHintCwd("vim "+file, home); got != proj {
		t.Errorf("vim <file>: got %q want %q", got, proj)
	}
	// Dir path → itself.
	if got := inferHintCwd("ls "+proj, home); got != proj {
		t.Errorf("ls <dir>: got %q want %q", got, proj)
	}
	// Quoted paths.
	if got := inferHintCwd("cat \""+file+"\"", home); got != proj {
		t.Errorf("cat \"<file>\": got %q want %q", got, proj)
	}
}

// TestInferHintCwd_IgnoresSystemPaths — /usr/bin/foo isn't a cwd signal.
func TestInferHintCwd_IgnoresSystemPaths(t *testing.T) {
	home := t.TempDir()
	// System dirs, homebrew, /tmp — none should return a hint.
	for _, cmd := range []string{
		"/usr/bin/env python",
		"cat /etc/hosts",
		"ls /Applications",
		"open /Library/Preferences",
		"/opt/homebrew/bin/brew list",
		"cat /tmp/foo",
	} {
		if got := inferHintCwd(cmd, home); got != "" {
			t.Errorf("system path %q should not produce hint, got %q", cmd, got)
		}
	}
}

// TestInferHintCwd_NonexistentPathReturnsEmpty — we require the path to
// exist (except for git -C which is stronger).
func TestInferHintCwd_NonexistentPathReturnsEmpty(t *testing.T) {
	home := t.TempDir()
	if got := inferHintCwd("vim /Users/nobody/does/not/exist.txt", home); got != "" {
		t.Errorf("nonexistent path should not produce hint, got %q", got)
	}
}

// TestParser_AbsPathCorrectsDrift — the big one. Simulate cwd drift from
// two interleaved shells (within one time window so time-gap reset doesn't
// mask it), then verify a command with a real abs-path arg pulls cwd back.
func TestParser_AbsPathCorrectsDrift(t *testing.T) {
	home := t.TempDir()
	realProj := filepath.Join(home, "code", "real-proj")
	realFile := filepath.Join(realProj, "notes.md")
	_ = os.MkdirAll(realProj, 0o755)
	_ = os.WriteFile(realFile, []byte("x"), 0o644)

	histFile := filepath.Join(home, ".zsh_history")
	// Terminal A drifts into a weird spot; terminal B then edits a file
	// with an abs path — the hint should snap cwd back.
	lines := []string{
		": 1000:0;cd " + home + "/code/other/nested/deep/place",
		": 1001:0;ls",
		": 1002:0;vim " + realFile, // abs-path hint → cwd should become realProj
		": 1003:0;grep foo *.md",
	}
	if err := os.WriteFile(histFile, []byte(joinLines(lines)), 0o644); err != nil {
		t.Fatal(err)
	}
	// Create the drifted dir too so the initial cd resolves cleanly.
	_ = os.MkdirAll(filepath.Join(home, "code", "other", "nested", "deep", "place"), 0o755)

	cfg := &Config{HistoryFile: histFile, HomeDir: home, SessionTimeout: 30 * 60 * 1_000_000_000}
	p := NewParser(cfg)
	entries, err := p.ParseHistory()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 4 {
		t.Fatalf("want 4 entries, got %d", len(entries))
	}
	// After the vim <realFile> command, cwd should be corrected to realProj.
	if entries[2].Directory != realProj {
		t.Errorf("vim entry cwd: want %q, got %q — abs-path hint didn't correct drift", realProj, entries[2].Directory)
	}
	// And the subsequent `grep` inherits that corrected cwd.
	if entries[3].Directory != realProj {
		t.Errorf("grep entry cwd: want %q, got %q — corrected cwd didn't stick", realProj, entries[3].Directory)
	}
}

// TestParser_PushdPopdStack covers the proper stack behaviour: pushd
// remembers cwd, popd restores it.
func TestParser_PushdPopdStack(t *testing.T) {
	home := t.TempDir()
	a := filepath.Join(home, "a")
	b := filepath.Join(home, "b")
	_ = os.MkdirAll(a, 0o755)
	_ = os.MkdirAll(b, 0o755)
	histFile := filepath.Join(home, ".zsh_history")
	lines := []string{
		": 1000:0;cd " + a,
		": 1001:0;pushd " + b,
		": 1002:0;ls",
		": 1003:0;popd",
		": 1004:0;ls",
	}
	_ = os.WriteFile(histFile, []byte(joinLines(lines)), 0o644)

	cfg := &Config{HistoryFile: histFile, HomeDir: home, SessionTimeout: 30 * 60 * 1_000_000_000}
	entries, err := NewParser(cfg).ParseHistory()
	if err != nil {
		t.Fatal(err)
	}
	if entries[2].Directory != b {
		t.Errorf("after pushd, cwd should be %q, got %q", b, entries[2].Directory)
	}
	if entries[4].Directory != a {
		t.Errorf("after popd, cwd should return to %q, got %q", a, entries[4].Directory)
	}
}

// TestParser_HintIgnoredWhenAlreadyInScope — no correction when the abs
// path is already under the current cwd (the drift is real work in the
// same tree).
func TestParser_HintIgnoredWhenAlreadyInScope(t *testing.T) {
	home := t.TempDir()
	proj := filepath.Join(home, "proj")
	sub := filepath.Join(proj, "sub")
	file := filepath.Join(sub, "x.go")
	_ = os.MkdirAll(sub, 0o755)
	_ = os.WriteFile(file, []byte("x"), 0o644)
	histFile := filepath.Join(home, ".zsh_history")
	lines := []string{
		": 100:0;cd " + proj,
		": 101:0;vim " + file, // abs path shares prefix with cwd → don't correct to sub
	}
	_ = os.WriteFile(histFile, []byte(joinLines(lines)), 0o644)

	cfg := &Config{HistoryFile: histFile, HomeDir: home, SessionTimeout: 30 * 60 * 1_000_000_000}
	entries, err := NewParser(cfg).ParseHistory()
	if err != nil {
		t.Fatal(err)
	}
	// cwd stays at proj — no correction needed.
	if entries[1].Directory != proj {
		t.Errorf("vim inside proj shouldn't move cwd; want %q got %q", proj, entries[1].Directory)
	}
}
