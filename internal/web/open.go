package web

import (
	"os/exec"
	"runtime"
)

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
