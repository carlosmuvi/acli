// Package android wraps the Android tooling acli drives: emulator/AVD
// management and screenshots (via a swappable Backend) plus logcat streaming
// (always via adb, since the android CLI has no logcat support).
package android

import "github.com/carlosmuvi/acli/internal/sdk"

// Device is a connected emulator or physical device.
type Device struct {
	Serial string // e.g. "emulator-5554" or a USB serial
	State  string // "device", "offline", "unauthorized", ...
	Name   string // AVD name for emulators, model for physical devices
	IsEmu  bool   // true for emulators (launch/kill/wipe apply)
}

// LaunchOpts controls how an AVD is started.
type LaunchOpts struct {
	ColdBoot bool // ignore saved snapshot
	WipeData bool // factory reset on boot
	Headless bool // no window
}

// Backend abstracts emulator/AVD management and screenshots so acli can prefer
// the agent-first `android` CLI and fall back to raw adb/emulator.
type Backend interface {
	// Name identifies the backend for the doctor/UI ("android CLI" or "adb/emulator").
	Name() string
	// ListAVDs returns the names of available (not necessarily running) AVDs.
	ListAVDs() ([]string, error)
	// Devices returns currently connected emulators and devices.
	Devices() ([]Device, error)
	// Launch starts the named AVD with the given options. It returns once the
	// process has been spawned; boot completes asynchronously.
	Launch(avd string, opts LaunchOpts) error
	// Kill stops the emulator with the given serial.
	Kill(serial string) error
	// Screenshot captures the device screen to a PNG at path.
	Screenshot(serial, path string) error
}

// Select chooses the best available backend: the android CLI when present,
// otherwise raw adb/emulator. It returns nil only if neither adb nor emulator
// nor the android CLI is usable (doctor should have caught this first).
func Select(t sdk.Tools) Backend {
	if t.Android != "" {
		return &androidCLI{tools: t}
	}
	return &adbBackend{tools: t}
}
