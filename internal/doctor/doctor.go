// Package doctor runs a preflight health check before acli starts, reporting
// which tools and resources are available and how to fix what's missing.
package doctor

import (
	"bufio"
	"os/exec"
	"strconv"
	"strings"

	"github.com/carlosmuvi/acli/internal/sdk"
)

// Check is a single preflight result. Hard checks gate startup; soft checks are
// informational and never block.
type Check struct {
	Name   string
	OK     bool
	Hard   bool   // a failing hard check aborts startup
	Detail string // current state, e.g. the resolved path or "not found"
	Fix    string // one-line remediation hint when not OK
}

// Report is the full set of checks plus a convenience flag.
type Report struct {
	Checks []Check
}

// HasHardFailure reports whether any required check failed.
func (r Report) HasHardFailure() bool {
	for _, c := range r.Checks {
		if c.Hard && !c.OK {
			return true
		}
	}
	return false
}

// Run inspects the resolved tooling and returns a Report.
func Run(t sdk.Tools) Report {
	var checks []Check

	// adb is required: logcat depends on it.
	checks = append(checks, Check{
		Name:   "adb",
		OK:     t.Adb != "",
		Hard:   true,
		Detail: detail(t.Adb),
		Fix:    "Install the Android SDK platform-tools (Android Studio, or `brew install --cask android-commandline-tools`) and set ANDROID_HOME.",
	})

	// android CLI is preferred but optional; adb/emulator is the fallback.
	checks = append(checks, Check{
		Name:   "android CLI",
		OK:     t.Android != "",
		Hard:   false,
		Detail: detailOr(t.Android, "not found — falling back to adb/emulator"),
		Fix:    "Optional: install the agent-first CLI from https://developer.android.com/tools/agents",
	})

	// emulator is needed for the fallback launcher; soft when android CLI exists.
	checks = append(checks, Check{
		Name:   "emulator",
		OK:     t.Emulator != "",
		Hard:   false,
		Detail: detailOr(t.Emulator, "not found"),
		Fix:    "Install the `emulator` package via the SDK manager to launch AVDs without the android CLI.",
	})

	checks = append(checks, Check{
		Name:   "SDK root",
		OK:     t.SDKRoot != "",
		Hard:   false,
		Detail: detailOr(t.SDKRoot, "not resolved"),
		Fix:    "Set ANDROID_HOME (or ANDROID_SDK_ROOT) to your SDK location.",
	})

	// Informational counts, best-effort.
	avds := countAVDs(t)
	checks = append(checks, Check{
		Name:   "AVDs",
		OK:     avds > 0,
		Hard:   false,
		Detail: plural(avds, "AVD"),
		Fix:    "Create one with `android emulator create` (or Android Studio's Device Manager).",
	})

	devs := countDevices(t)
	checks = append(checks, Check{
		Name:   "connected devices",
		OK:     devs > 0,
		Hard:   false,
		Detail: plural(devs, "device"),
		Fix:    "Launch an emulator or plug in a device — none are required to start acli.",
	})

	return Report{Checks: checks}
}

// countAVDs counts available AVDs via the android CLI when present, else emulator.
func countAVDs(t sdk.Tools) int {
	if t.Android != "" {
		// `android emulator list` prints one AVD per line (plus possible header noise).
		return nonEmptyLines(run(t.Android, "emulator", "list"))
	}
	if t.Emulator != "" {
		return nonEmptyLines(run(t.Emulator, "-list-avds"))
	}
	return 0
}

// countDevices counts entries from `adb devices` (excluding the header line).
func countDevices(t sdk.Tools) int {
	if t.Adb == "" {
		return 0
	}
	out := run(t.Adb, "devices")
	n := 0
	sc := bufio.NewScanner(strings.NewReader(out))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "List of devices") {
			continue
		}
		// A real entry looks like "emulator-5554\tdevice".
		if len(strings.Fields(line)) >= 2 {
			n++
		}
	}
	return n
}

func run(name string, args ...string) string {
	out, err := exec.Command(name, args...).Output()
	if err != nil {
		return ""
	}
	return string(out)
}

func nonEmptyLines(out string) int {
	n := 0
	sc := bufio.NewScanner(strings.NewReader(out))
	for sc.Scan() {
		if strings.TrimSpace(sc.Text()) != "" {
			n++
		}
	}
	return n
}

func detail(path string) string { return detailOr(path, "not found") }

func detailOr(path, missing string) string {
	if path == "" {
		return missing
	}
	return path
}

func plural(n int, unit string) string {
	s := "s"
	if n == 1 {
		s = ""
	}
	return strconv.Itoa(n) + " " + unit + s
}
