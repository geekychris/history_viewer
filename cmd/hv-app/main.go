// cmd/hv-app is a Wails v2 wrapper around the geekychris/history_viewer
// web UI. It spawns the existing `history_viewer --ui web` as a child
// process on a chosen loopback port, waits for the server to come up,
// then runs a Wails window whose embedded index.html redirects to that
// loopback URL.
//
// Rationale: gives users a real native app window (menu bar, dock, cmd-tab,
// resizable frame) around the same UI without refactoring history_viewer's
// existing main package into a library.
//
// Flags (forwarded to the child):
//   --filter-dir <path>   deep-link filter (see history_viewer --filter-dir)
//   --history <path>      alternate zsh history file
//   --port <n>            pin the child's port (default: auto-pick a free one)
//   --bin <path>          override the child binary (default: PATH lookup +
//                         well-known Homebrew / ~/.local/bin locations)
package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var assets embed.FS

// Version is stamped at build time via -ldflags.
var Version = "dev"

// App owns the spawned child process so we can tear it down cleanly on quit.
type App struct {
	mu    sync.Mutex
	ctx   context.Context
	child *exec.Cmd
	port  int
}

func (a *App) shutdown(_ context.Context)      { a.killChild() }
func (a *App) beforeClose(_ context.Context) bool { a.killChild(); return false }

func (a *App) killChild() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.child != nil && a.child.Process != nil {
		_ = a.child.Process.Kill()
		_ = a.child.Wait()
		a.child = nil
	}
}

func main() {
	filterDir := flag.String("filter-dir", "", "Deep-link filter: only show commands run under this directory")
	historyFile := flag.String("history", "", "Path to zsh history file (default: ~/.zsh_history)")
	port := flag.Int("port", 0, "Loopback port for the embedded web server (0 = auto-pick)")
	binOverride := flag.String("bin", "", "Path to the history_viewer binary (default: PATH lookup)")
	flag.Parse()

	bin, err := findChildBinary(*binOverride)
	if err != nil {
		log.Fatalf("hv-app: %v", err)
	}
	log.Printf("hv-app: using child binary %s", bin)

	chosenPort := *port
	if chosenPort == 0 {
		chosenPort, err = pickFreePort()
		if err != nil {
			log.Fatalf("hv-app: pick port: %v", err)
		}
	}

	app := &App{port: chosenPort}
	if err := app.spawnChild(bin, chosenPort, *filterDir, *historyFile); err != nil {
		log.Fatalf("hv-app: spawn child: %v", err)
	}
	// Wait for the web server to accept connections (up to ~5s).
	if err := waitForPort(chosenPort, 5*time.Second); err != nil {
		app.killChild()
		log.Fatalf("hv-app: child never came up on :%d: %v", chosenPort, err)
	}

	// Build the target URL, including ?dir=… for the frontend deep-link.
	// The redirect stub in frontend/dist/index.html reads window.HV_URL,
	// which we inject via a runtime call once the window is ready.
	target := fmt.Sprintf("http://127.0.0.1:%d/", chosenPort)
	if *filterDir != "" {
		target += "?dir=" + urlEscape(*filterDir)
	}

	err = wails.Run(&options.App{
		Title:            "History Viewer",
		Width:            1200,
		Height:           780,
		MinWidth:         800,
		MinHeight:        480,
		BackgroundColour: &options.RGBA{R: 32, G: 32, B: 42, A: 1},
		OnStartup: func(ctx context.Context) {
			app.mu.Lock()
			app.ctx = ctx
			app.mu.Unlock()
			log.Printf("hv-app: target %s", target)
		},
		OnDomReady: func(ctx context.Context) {
			// Belt-and-suspenders: the embedded stub also does location.replace,
			// but if for any reason it didn't fire, force it here.
			runtime.WindowExecJS(ctx, fmt.Sprintf("window.location.replace(%q);", target))
		},
		OnBeforeClose: app.beforeClose,
		OnShutdown:    app.shutdown,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		Mac: &mac.Options{
			About: &mac.AboutInfo{
				Title:   "History Viewer",
				Message: "Wails wrapper for geekychris/history_viewer.\nVersion " + Version,
			},
		},
	})
	if err != nil {
		app.killChild()
		log.Fatalf("hv-app: wails.Run: %v", err)
	}
}

// spawnChild launches history_viewer as a detached child. Stderr/stdout are
// inherited so users can see logs when running from a terminal; the .app
// bundle discards them via Info.plist redirection.
func (a *App) spawnChild(bin string, port int, filterDir, historyFile string) error {
	args := []string{"--ui", "web", "--port", fmt.Sprintf("%d", port)}
	if filterDir != "" {
		args = append(args, "--filter-dir", filterDir)
	}
	if historyFile != "" {
		args = append(args, "--history", historyFile)
	}
	cmd := exec.Command(bin, args...)
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return err
	}
	a.mu.Lock()
	a.child = cmd
	a.mu.Unlock()
	return nil
}

// findChildBinary tries, in order: --bin flag, $PATH, common Homebrew and
// user-local install paths. Returns absolute path or an error.
func findChildBinary(override string) (string, error) {
	if override != "" {
		if _, err := os.Stat(override); err != nil {
			return "", fmt.Errorf("--bin %s: %w", override, err)
		}
		return override, nil
	}
	if p, err := exec.LookPath("history_viewer"); err == nil {
		return p, nil
	}
	home, _ := os.UserHomeDir()
	for _, p := range []string{
		filepath.Join(home, ".local", "bin", "history_viewer"),
		"/opt/homebrew/bin/history_viewer",
		"/usr/local/bin/history_viewer",
	} {
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			return p, nil
		}
	}
	return "", fmt.Errorf("history_viewer binary not found — install with `brew install geekychris/history-viewer/history-viewer` or from source")
}

// pickFreePort listens on 127.0.0.1:0, reads the port, closes. Not
// race-free but good enough for this launch path.
func pickFreePort() (int, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port, nil
}

// waitForPort polls a TCP dial until it succeeds or the timeout elapses.
func waitForPort(port int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 200*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return nil
		}
		time.Sleep(150 * time.Millisecond)
	}
	return fmt.Errorf("timeout")
}

// urlEscape is a tiny helper so we don't drag in net/url just for one call.
func urlEscape(s string) string {
	out := make([]byte, 0, len(s)+8)
	for i := 0; i < len(s); i++ {
		c := s[i]
		// Percent-encode anything that isn't unreserved-URL-safe.
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') || c == '-' || c == '_' || c == '.' || c == '~' {
			out = append(out, c)
		} else {
			const hex = "0123456789ABCDEF"
			out = append(out, '%', hex[c>>4], hex[c&0xf])
		}
	}
	return string(out)
}
