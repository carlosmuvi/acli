package android

import "sort"

// Entry is a unified row combining a known AVD with its running state (if any),
// or a connected device. It's the shape both the web UI and any non-TUI caller
// consume.
type Entry struct {
	Name    string `json:"name"`             // AVD name or device label
	AVD     string `json:"avd,omitempty"`    // AVD name to launch ("" for non-AVD devices)
	Serial  string `json:"serial,omitempty"` // set when running
	Running bool   `json:"running"`
	IsEmu   bool   `json:"isEmu"`
}

// Merge combines the list of known AVDs with currently connected devices into a
// single sorted list (running first, then by name). Running emulators are
// matched to their AVD by resolved name; unmatched ones are shown without an
// AVD so they don't masquerade as launchable entries.
func Merge(avds []string, devices []Device) []Entry {
	runningByAVD := map[string]Device{}
	var nonAVD []Device
	for _, d := range devices {
		d := d
		if d.IsEmu {
			runningByAVD[d.Name] = d
		} else {
			nonAVD = append(nonAVD, d)
		}
	}

	var entries []Entry
	for _, avd := range avds {
		e := Entry{Name: avd, AVD: avd, IsEmu: true}
		if d, ok := runningByAVD[avd]; ok {
			e.Running = true
			e.Serial = d.Serial
			delete(runningByAVD, avd)
		}
		entries = append(entries, e)
	}
	for _, d := range runningByAVD {
		entries = append(entries, Entry{Name: d.Name, Serial: d.Serial, Running: true, IsEmu: true})
	}
	for _, d := range nonAVD {
		entries = append(entries, Entry{Name: d.Name, Serial: d.Serial, Running: true})
	}

	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].Running != entries[j].Running {
			return entries[i].Running
		}
		return entries[i].Name < entries[j].Name
	})
	return entries
}
