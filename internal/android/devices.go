package android

import (
	"bufio"
	"os/exec"
	"strings"
)

// listDevices parses `adb devices -l` into Devices. Both backends share this
// because the android CLI has no device-list command. For emulators it resolves
// the AVD name via `adb -s <serial> emu avd name`.
func listDevices(adb string) ([]Device, error) {
	if adb == "" {
		return nil, errNoAdb
	}
	out, err := exec.Command(adb, "devices", "-l").Output()
	if err != nil {
		return nil, err
	}
	var devices []Device
	sc := bufio.NewScanner(strings.NewReader(out2str(out)))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "List of devices") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		serial, state := fields[0], fields[1]
		isEmu := strings.HasPrefix(serial, "emulator-")
		name := deviceLabel(fields)
		if isEmu && state == "device" {
			if avd := emuAVDName(adb, serial); avd != "" {
				name = avd
			}
		}
		devices = append(devices, Device{Serial: serial, State: state, Name: name, IsEmu: isEmu})
	}
	return devices, sc.Err()
}

// deviceLabel extracts a human label from the `-l` annotations (model:Pixel_7),
// falling back to the device/product fields.
func deviceLabel(fields []string) string {
	for _, f := range fields[2:] {
		if v, ok := strings.CutPrefix(f, "model:"); ok {
			return strings.ReplaceAll(v, "_", " ")
		}
	}
	for _, f := range fields[2:] {
		if v, ok := strings.CutPrefix(f, "device:"); ok {
			return v
		}
	}
	return fields[0]
}

// emuAVDName asks a running emulator for its AVD name. `adb emu avd name`
// returns the name on the first line and "OK" on the second.
func emuAVDName(adb, serial string) string {
	out, err := exec.Command(adb, "-s", serial, "emu", "avd", "name").Output()
	if err != nil {
		return ""
	}
	sc := bufio.NewScanner(strings.NewReader(out2str(out)))
	if sc.Scan() {
		return strings.TrimSpace(sc.Text())
	}
	return ""
}

func out2str(b []byte) string { return strings.ReplaceAll(string(b), "\r\n", "\n") }
