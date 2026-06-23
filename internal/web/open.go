package web

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// OpenApp tries to open url as a standalone, chromeless "app" window using a
// Chromium-family browser (Chrome, Brave, Edge, Chromium). This gives a
// floating window with no tabs/address bar that can sit beside the emulator.
// Returns false if no suitable browser was found, so the caller can fall back
// to a normal browser tab.
func OpenApp(url string, width, height int) bool {
	bin := findChromium()
	if bin == "" {
		return false
	}
	// A dedicated profile dir avoids clashing with the user's running browser
	// (Chromium locks its default profile to a single process).
	dataDir := filepath.Join(os.TempDir(), "acli-web-profile")
	args := []string{
		"--app=" + url,
		"--user-data-dir=" + dataDir,
		fmt.Sprintf("--window-size=%d,%d", width, height),
		"--no-first-run",
		"--no-default-browser-check",
	}
	return exec.Command(bin, args...).Start() == nil
}

// OpenBrowser best-effort opens url in the user's default browser. Failures are
// ignored — the URL is also printed by the caller.
func OpenBrowser(url string) {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
	case "windows":
		cmd, args = "rundll32", []string{"url.dll,FileProtocolHandler"}
	default:
		cmd = "xdg-open"
	}
	args = append(args, url)
	_ = exec.Command(cmd, args...).Start()
}

// findChromium returns the path to a Chromium-family browser binary, or "".
func findChromium() string {
	switch runtime.GOOS {
	case "darwin":
		for _, p := range []string{
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Brave Browser.app/Contents/MacOS/Brave Browser",
			"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
		} {
			if fileExists(p) {
				return p
			}
		}
	case "windows":
		for _, p := range []string{
			os.Getenv("ProgramFiles") + `\Google\Chrome\Application\chrome.exe`,
			os.Getenv("ProgramFiles(x86)") + `\Google\Chrome\Application\chrome.exe`,
			os.Getenv("ProgramFiles(x86)") + `\Microsoft\Edge\Application\msedge.exe`,
		} {
			if fileExists(p) {
				return p
			}
		}
	default: // linux and friends
		for _, n := range []string{
			"google-chrome", "google-chrome-stable", "chromium",
			"chromium-browser", "brave-browser", "microsoft-edge",
		} {
			if p, err := exec.LookPath(n); err == nil {
				return p
			}
		}
	}
	return ""
}

func fileExists(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && !fi.IsDir()
}
