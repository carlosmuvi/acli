package android

import (
	"bufio"
	"os/exec"
	"strings"
)

// ProcessMap returns a PID→process-name map for a device via `adb shell ps`.
// Used to enrich logcat lines (which carry only a PID) with a package/process
// name so the UI can filter by `package:`/`process:`.
func ProcessMap(adb, serial string) map[string]string {
	if adb == "" {
		return nil
	}
	out, err := exec.Command(adb, "-s", serial, "shell", "ps", "-A", "-o", "PID,NAME").Output()
	if err != nil || len(out) == 0 {
		// Older devices may not support -o; fall back to a plain listing.
		out, err = exec.Command(adb, "-s", serial, "shell", "ps").Output()
		if err != nil {
			return nil
		}
	}
	return parsePS(out2str(out))
}

// ThirdPartyPackages lists user-installed (non-system) packages via
// `pm list packages -3` — the closest the device can tell us to "your apps",
// used to resolve `package:mine`.
func ThirdPartyPackages(adb, serial string) []string {
	if adb == "" {
		return nil
	}
	out, err := exec.Command(adb, "-s", serial, "shell", "pm", "list", "packages", "-3").Output()
	if err != nil {
		return nil
	}
	return parsePackages(out2str(out))
}

// parsePS parses `ps` output into PID→name, tolerating either the explicit
// `-o PID,NAME` two-column form or a full default listing (PID is the second
// column, the process name is the last).
func parsePS(out string) map[string]string {
	m := map[string]string{}
	sc := bufio.NewScanner(strings.NewReader(out))
	first := true
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) < 2 {
			continue
		}
		// Skip a header row like "PID NAME" or "USER PID ...".
		if first {
			first = false
			if !isNumeric(fields[0]) && !isNumeric(fields[1]) {
				continue
			}
		}
		var pid, name string
		if isNumeric(fields[0]) {
			pid, name = fields[0], fields[len(fields)-1] // PID,NAME form
		} else if isNumeric(fields[1]) {
			pid, name = fields[1], fields[len(fields)-1] // default ps form
		} else {
			continue
		}
		m[pid] = name
	}
	return m
}

// parsePackages turns `package:com.foo` lines into a sorted-ish slice of names.
func parsePackages(out string) []string {
	var pkgs []string
	sc := bufio.NewScanner(strings.NewReader(out))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if name, ok := strings.CutPrefix(line, "package:"); ok && name != "" {
			pkgs = append(pkgs, name)
		}
	}
	return pkgs
}

func isNumeric(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
