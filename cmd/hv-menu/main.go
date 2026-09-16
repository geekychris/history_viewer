// cmd/hv-menu is a tiny macOS menu-bar helper for the History Viewer app.
//
// The wrapping .app bundle sets LSUIElement=true so this process runs
// menu-bar-only (no Dock icon, no app switcher). Clicking the menu bar item
// launches HistoryViewer.app (or brings it forward if already running).
package main

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/getlantern/systray"
)

// Version is stamped at build time via -ldflags.
var Version = "dev"

func main() {
	runtime.LockOSThread()
	systray.Run(onReady, onExit)
}

func onReady() {
	systray.SetTitle("⌛")
	systray.SetTooltip("Zsh History Viewer")

	mOpen := systray.AddMenuItem("Open History Viewer", "open the History Viewer window")
	mRefresh := systray.AddMenuItem("Reopen (fresh window)", "quit any existing instance and re-launch")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit menu", "exit the menu bar helper")

	go func() {
		for {
			select {
			case <-mOpen.ClickedCh:
				launchApp(false)
			case <-mRefresh.ClickedCh:
				killApp()
				launchApp(true)
			case <-mQuit.ClickedCh:
				systray.Quit()
				return
			}
		}
	}()
}

func onExit() {}

// launchApp opens the History Viewer .app via LaunchServices. `newInstance`
// forces a fresh process (macOS `open -n`); false lets the existing instance
// come to the front.
func launchApp(newInstance bool) {
	appPath := findAppBundle()
	if appPath == "" {
		log.Println("hv-menu: History Viewer.app not found in ~/Applications or /Applications")
		return
	}
	args := []string{"-a", appPath}
	if newInstance {
		args = []string{"-n", "-a", appPath}
	}
	if err := exec.Command("open", args...).Start(); err != nil {
		log.Printf("hv-menu: open %s failed: %v", appPath, err)
	}
}

// killApp best-effort terminates a running HistoryViewer process before we
// spawn a fresh one (Reopen menu item). Silently no-ops if none running.
func killApp() {
	_ = exec.Command("pkill", "-f", "History Viewer.app/Contents/MacOS/HistoryViewer").Run()
}

// findAppBundle looks for the .app in the user's Applications dir first
// (Chief installs there), then the system dir.
func findAppBundle() string {
	home, _ := os.UserHomeDir()
	for _, p := range []string{
		filepath.Join(home, "Applications", "History Viewer.app"),
		"/Applications/History Viewer.app",
	} {
		if info, err := os.Stat(p); err == nil && info.IsDir() {
			return p
		}
	}
	return ""
}
