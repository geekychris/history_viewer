package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestSanitizeDir_CollapseRepeats verifies the "chained cd artefact" fix.
// The rule: keep up to 2 adjacent identical segments (real paths do have
// duplicates like github.com/foo/foo) but drop the 3rd and beyond. This
// is the pattern that shows up in cross-terminal cwd drift bugs.
func TestSanitizeDir_CollapseRepeats(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"/a/build/build/build", "/a/build/build"},          // 3rd drops
		{"/a/build/build/build/build", "/a/build/build"},    // 4th+ also drop
		{"/a/b/b", "/a/b/b"},                                // 2 kept
		{"/a/.claude/.claude/thing", "/a/.claude/.claude/thing"}, // 2 adjacent stays
		{"/a/.claude/.claude/.claude/thing", "/a/.claude/.claude/thing"},
		{"/a", "/a"},
		{"/", "/"},
	}
	for _, c := range cases {
		got := sanitizeDir(c.in, "/home/me")
		if got != c.want {
			t.Errorf("sanitizeDir(%q): got %q want %q", c.in, got, c.want)
		}
	}
}

// TestSanitizeDir_SegmentRepeatedManyTimesResets covers the "same segment
// appears 3+ times anywhere in the path" defence — cross-terminal drift
// often produces `foo/.../foo/.../foo` patterns where each `cd foo` came
// from a different shell.
func TestSanitizeDir_SegmentRepeatedManyTimesResets(t *testing.T) {
	// Adjacent case is already covered — this one is non-adjacent repeats.
	cases := []struct {
		in       string
		wantHome bool
	}{
		{"/a/proj/src/proj/build/proj", true},      // proj × 3, non-adjacent
		{"/a/b/c/d/e/f/g", false},                  // unique segments
		{"/a/build/x/build", false},                // 2 buildss — allowed
		{"/a/hitorro/src/hitorro/test/hitorro", true}, // hitorro × 3
	}
	home := "/home/me"
	for _, c := range cases {
		got := sanitizeDir(c.in, home)
		if c.wantHome && got != home {
			t.Errorf("sanitizeDir(%q): expected reset to home, got %q", c.in, got)
		}
		if !c.wantHome && got == home {
			t.Errorf("sanitizeDir(%q): unexpected reset to home", c.in)
		}
	}
}

// TestSanitizeDir_DepthCapResetsToHome covers the absurd-depth defence.
// Uses distinct segments so the collapse rule doesn't hide the depth.
func TestSanitizeDir_DepthCapResetsToHome(t *testing.T) {
	deep := ""
	for i := 0; i < 40; i++ {
		deep += "/seg" + string(rune('a'+(i%26)))
	}
	got := sanitizeDir(deep, "/home/me")
	if got != "/home/me" {
		t.Errorf("depth cap: want reset to home, got %q", got)
	}

	// Just under the cap should pass through.
	shallow := ""
	for i := 0; i < 25; i++ {
		shallow += "/seg" + string(rune('a'+(i%26)))
	}
	got = sanitizeDir(shallow, "/home/me")
	if got == "/home/me" {
		t.Errorf("depth 25: want passthrough, got reset")
	}
}

// TestParser_TimeGapResetsCwd creates a synthetic zsh history where two
// separate "terminals" interleave. Without the time-gap reset the second
// terminal's `cd build` would compound onto the first's cwd. With the
// reset it starts fresh from HomeDir.
func TestParser_TimeGapResetsCwd(t *testing.T) {
	home := t.TempDir()
	histFile := filepath.Join(home, ".zsh_history")

	// Two "shell sessions" separated by 2h. Both do `cd proj-x && cd build`.
	// Without reset, the second session would land in
	//   /home/proj-a/build/proj-b/build   (wrong)
	// With reset, it should land in
	//   /home/proj-b/build                (right)
	//
	// Timestamps are UNIX seconds. Session A around t=1000; session B around
	// t=1000 + 3h (12000 apart).
	//
	// zsh EXTENDED_HISTORY format:  ": <timestamp>:<duration>;<command>"
	lines := []string{
		": 1000:0;cd " + home + "/proj-a",
		": 1001:0;cd build",
		": 1002:0;ls",
		// > 30m gap → cwd reset
		": 11000:0;cd " + home + "/proj-b",
		": 11001:0;cd build",
		": 11002:0;ls",
	}
	if err := os.WriteFile(histFile, []byte(joinLines(lines)), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create the target dirs so any Clean/resolve stays sensible.
	_ = os.MkdirAll(filepath.Join(home, "proj-a", "build"), 0o755)
	_ = os.MkdirAll(filepath.Join(home, "proj-b", "build"), 0o755)

	cfg := &Config{
		HistoryFile: histFile,
		HomeDir:     home,
		// SessionTimeout in the Config type is time.Duration.  Use 30min.
		SessionTimeout: 30 * 60 * 1_000_000_000,
	}
	p := NewParser(cfg)
	entries, err := p.ParseHistory()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 6 {
		t.Fatalf("want 6 entries, got %d", len(entries))
	}

	wantB := filepath.Join(home, "proj-b", "build")
	if entries[5].Directory != wantB {
		t.Errorf("session B final entry: want cwd %q, got %q — cwd reset failed and drift accumulated", wantB, entries[5].Directory)
	}
	wantA := filepath.Join(home, "proj-a", "build")
	if entries[2].Directory != wantA {
		t.Errorf("session A: want cwd %q, got %q", wantA, entries[2].Directory)
	}
}

// TestParser_ChainedCDDoesNotCompoundWildly documents that when a shell
// does `cd build` twice within a single window (with no time gap), the
// second one lands in build/build — that's actually correct if the shell
// really did those commands. It's the CROSS-terminal interleaving that
// creates the bogus 3+-deep chains, which sanitizeDir catches at the
// resolve step.
func TestParser_ChainedCDDoesNotCompoundWildly(t *testing.T) {
	home := t.TempDir()
	histFile := filepath.Join(home, ".zsh_history")
	// One shell, five interleaved `cd build` (from confused parsing across
	// terminals) all within a minute. sanitizeDir should collapse.
	lines := []string{
		": 100:0;cd " + home + "/proj",
		": 101:0;cd build",
		": 102:0;cd build",
		": 103:0;cd build",
		": 104:0;cd build",
		": 105:0;ls",
	}
	_ = os.WriteFile(histFile, []byte(joinLines(lines)), 0o644)
	_ = os.MkdirAll(filepath.Join(home, "proj", "build"), 0o755)
	cfg := &Config{HistoryFile: histFile, HomeDir: home, SessionTimeout: 30 * 60 * 1_000_000_000}
	p := NewParser(cfg)
	entries, err := p.ParseHistory()
	if err != nil {
		t.Fatal(err)
	}
	// Last entry's cwd shouldn't be .../build/build/build/build/build (5 deep) —
	// sanitizeDir collapses runs of 3+.  Two `build`s is the maximum we permit,
	// so the result is proj/build/build.
	got := entries[len(entries)-1].Directory
	// Count consecutive "build" segments.
	repeats := 0
	max := 0
	for _, seg := range filepath.SplitList(got) {
		_ = seg
	}
	// Simpler: check the depth after "proj".
	prefix := filepath.Join(home, "proj")
	suffix := got[len(prefix):]
	for _, r := range suffix {
		if r == filepath.Separator {
			repeats++
			if repeats > max {
				max = repeats
			}
		}
	}
	// We should NOT see 4+ separators after "proj" (which would mean
	// /build/build/build/build).  2 is the max sanitizeDir allows.
	if got != filepath.Join(home, "proj", "build", "build") && got != filepath.Join(home, "proj", "build") {
		t.Errorf("expected proj/build or proj/build/build after collapse, got %q", got)
	}
}

func joinLines(lines []string) string {
	out := ""
	for _, l := range lines {
		out += l + "\n"
	}
	return out
}
