package android

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/carlosmuvi/acli/internal/sdk"
)

// adbBackend drives the classic `emulator` and `adb` binaries directly. Used
// when the agent-first android CLI is not installed.
type adbBackend struct {
	tools sdk.Tools
}

func (b *adbBackend) Name() string { return "adb/emulator" }

func (b *adbBackend) ListAVDs() ([]string, error) {
	if b.tools.Emulator == "" {
		return nil, errors.New("emulator binary not found")
	}
	out, err := exec.Command(b.tools.Emulator, "-list-avds").Output()
	if err != nil {
		return nil, err
	}
	var avds []string
	sc := bufio.NewScanner(strings.NewReader(out2str(out)))
	for sc.Scan() {
		if line := strings.TrimSpace(sc.Text()); line != "" {
			avds = append(avds, line)
		}
	}
	return avds, nil
}

func (b *adbBackend) Devices() ([]Device, error) {
	return listDevices(b.tools.Adb)
}

func (b *adbBackend) Launch(avd string, opts LaunchOpts) error {
	if b.tools.Emulator == "" {
		return errors.New("emulator binary not found")
	}
	args := []string{"@" + avd}
	if opts.ColdBoot {
		args = append(args, "-no-snapshot-load")
	}
	if opts.WipeData {
		args = append(args, "-wipe-data")
	}
	if opts.Headless {
		args = append(args, "-no-window")
	}
	return startDetached(b.tools.Emulator, args...)
}

func (b *adbBackend) Kill(serial string) error {
	if b.tools.Adb == "" {
		return errNoAdb
	}
	return exec.Command(b.tools.Adb, "-s", serial, "emu", "kill").Run()
}

func (b *adbBackend) Screenshot(serial, path string) error {
	if b.tools.Adb == "" {
		return errNoAdb
	}
	args := []string{}
	if serial != "" {
		args = append(args, "-s", serial)
	}
	args = append(args, "exec-out", "screencap", "-p")
	out, err := exec.Command(b.tools.Adb, args...).Output()
	if err != nil {
		return fmt.Errorf("adb screencap: %w", err)
	}
	return os.WriteFile(path, out, 0o644)
}

// startDetached spawns a long-lived process (an emulator) without waiting for
// it, so the TUI keeps running. The child is released from acli's process group
// concerns by simply not calling Wait; output is discarded.
func startDetached(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Start(); err != nil {
		return err
	}
	// Reap asynchronously so the OS doesn't keep a zombie once it exits.
	go func() { _ = cmd.Wait() }()
	return nil
}
