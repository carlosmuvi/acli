package android

import (
	"bufio"
	"errors"
	"os/exec"
	"strings"

	"github.com/carlosmuvi/acli/internal/sdk"
)

var errNoAdb = errors.New("adb not found")

// androidCLI drives Google's agent-first `android` CLI for emulator management
// and screenshots. Device listing falls back to adb (the CLI has no equivalent).
type androidCLI struct {
	tools sdk.Tools
}

func (a *androidCLI) Name() string { return "android CLI" }

func (a *androidCLI) ListAVDs() ([]string, error) {
	out, err := exec.Command(a.tools.Android, "emulator", "list").Output()
	if err != nil {
		return nil, err
	}
	return parseAVDList(out2str(out)), nil
}

func (a *androidCLI) Devices() ([]Device, error) {
	return listDevices(a.tools.Adb)
}

func (a *androidCLI) Launch(avd string, opts LaunchOpts) error {
	// The real `android emulator start` only supports `--cold`; wipe-data and
	// headless are emulator-binary features, so delegate those to it.
	if opts.WipeData || opts.Headless {
		return emulatorLaunch(a.tools.Emulator, avd, opts)
	}
	args := []string{"emulator", "start", avd}
	if opts.ColdBoot {
		args = append(args, "--cold")
	}
	return startDetached(a.tools.Android, args...)
}

func (a *androidCLI) Kill(serial string) error {
	return exec.Command(a.tools.Android, "emulator", "stop", serial).Run()
}

func (a *androidCLI) Screenshot(serial, path string) error {
	// `android screen capture` can't target a serial, so use adb (always present)
	// for serial-accurate capture.
	return adbScreenshot(a.tools.Adb, serial, path)
}

// parseAVDList extracts AVD names from `android emulator list` output, skipping
// blank lines and any header/decoration the CLI may print.
func parseAVDList(out string) []string {
	var avds []string
	sc := bufio.NewScanner(strings.NewReader(out))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		low := strings.ToLower(line)
		if strings.HasPrefix(low, "available") || strings.HasPrefix(low, "name") || strings.HasPrefix(low, "---") {
			continue
		}
		// Some listings prefix entries with a bullet or index; take the last field.
		fields := strings.Fields(line)
		avds = append(avds, fields[len(fields)-1])
	}
	return avds
}
