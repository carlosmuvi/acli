// Package sdk locates the Android tooling acli depends on: the new agent-first
// `android` CLI, plus the classic `adb` and `emulator` binaries from the SDK.
package sdk

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// Tools holds resolved paths to the Android tooling. An empty path means the
// tool was not found.
type Tools struct {
	Android  string // `android` CLI (agent-first), preferred for emulator mgmt
	Adb      string // `adb`, required for logcat
	Emulator string // `emulator`, fallback launcher
	SDKRoot  string // resolved SDK root, if any
}

// Discover resolves the available Android tooling by inspecting environment
// variables, well-known SDK locations, and the PATH.
func Discover() Tools {
	root := discoverSDKRoot()
	return Tools{
		Android:  lookup("android"),
		Adb:      lookupSDK(root, filepath.Join("platform-tools", "adb"), "adb"),
		Emulator: lookupSDK(root, filepath.Join("emulator", "emulator"), "emulator"),
		SDKRoot:  root,
	}
}

// discoverSDKRoot resolves the SDK root from env vars, then platform defaults.
func discoverSDKRoot() string {
	for _, env := range []string{"ANDROID_HOME", "ANDROID_SDK_ROOT"} {
		if v := os.Getenv(env); v != "" && isDir(v) {
			return v
		}
	}
	home, _ := os.UserHomeDir()
	candidates := []string{}
	switch runtime.GOOS {
	case "darwin":
		candidates = append(candidates, filepath.Join(home, "Library", "Android", "sdk"))
	default: // linux and friends
		candidates = append(candidates, filepath.Join(home, "Android", "Sdk"))
	}
	for _, c := range candidates {
		if isDir(c) {
			return c
		}
	}
	return ""
}

// lookupSDK prefers the binary under the SDK root, then falls back to PATH.
func lookupSDK(root, rel, name string) string {
	if root != "" {
		p := filepath.Join(root, rel)
		if isExecFile(p) {
			return p
		}
	}
	return lookup(name)
}

// lookup finds a binary on the PATH.
func lookup(name string) string {
	p, err := exec.LookPath(name)
	if err != nil {
		return ""
	}
	return p
}

func isDir(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.IsDir()
}

func isExecFile(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && !fi.IsDir() && fi.Mode()&0o111 != 0
}
