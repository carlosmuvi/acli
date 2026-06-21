package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/carlosmuvi/acli/internal/android"

	tea "github.com/charmbracelet/bubbletea"
)

// emuItem is one row in the emulator list: an AVD that may or may not be
// running, or a connected (possibly physical) device.
type emuItem struct {
	name    string          // AVD name or device label
	avd     string          // AVD name to launch ("" for non-AVD devices)
	device  *android.Device // non-nil if currently connected
	running bool
}

// emulatorsModel renders the list of AVDs + running devices and handles
// launch/kill/screenshot/logcat actions.
type emulatorsModel struct {
	items  []emuItem
	cursor int

	width, height int
}

// rebuild merges the known AVD names with currently connected devices into a
// single sorted list, preserving the cursor position by name where possible.
func (m *emulatorsModel) rebuild(avds []string, devices []android.Device) {
	prevName := ""
	if m.cursor < len(m.items) {
		prevName = m.items[m.cursor].name
	}

	// Index running emulators by AVD name so we can mark AVDs as running.
	runningByAVD := map[string]android.Device{}
	var nonAVD []android.Device
	for _, d := range devices {
		d := d
		if d.IsEmu {
			runningByAVD[d.Name] = d
		} else {
			nonAVD = append(nonAVD, d)
		}
	}

	var items []emuItem
	for _, avd := range avds {
		it := emuItem{name: avd, avd: avd}
		if d, ok := runningByAVD[avd]; ok {
			dd := d
			it.device = &dd
			it.running = true
			delete(runningByAVD, avd)
		}
		items = append(items, it)
	}
	// Running emulators whose AVD name wasn't in the list (e.g. unknown).
	for _, d := range runningByAVD {
		dd := d
		items = append(items, emuItem{name: d.Name, avd: d.Name, device: &dd, running: true})
	}
	// Physical / non-AVD devices.
	for _, d := range nonAVD {
		dd := d
		items = append(items, emuItem{name: d.Name, device: &dd, running: true})
	}

	sort.SliceStable(items, func(i, j int) bool {
		if items[i].running != items[j].running {
			return items[i].running // running first
		}
		return items[i].name < items[j].name
	})

	m.items = items
	// Restore cursor by name.
	m.cursor = 0
	for i, it := range items {
		if it.name == prevName {
			m.cursor = i
			break
		}
	}
	if m.cursor >= len(items) {
		m.cursor = max(0, len(items)-1)
	}
}

func (m *emulatorsModel) setSize(w, h int) { m.width, m.height = w, h }

func (m *emulatorsModel) selected() (emuItem, bool) {
	if m.cursor < 0 || m.cursor >= len(m.items) {
		return emuItem{}, false
	}
	return m.items[m.cursor], true
}

// Update handles only cursor movement here; action keys are dispatched by the
// root model so it can issue backend commands and switch views.
func (m *emulatorsModel) Update(msg tea.Msg) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case "home", "g":
			m.cursor = 0
		case "end", "G":
			m.cursor = max(0, len(m.items)-1)
		}
	}
}

func (m emulatorsModel) View() string {
	var b strings.Builder
	b.WriteString(headerStyle.Render("Emulators & Devices") + "\n\n")

	if len(m.items) == 0 {
		b.WriteString(mutedStyle.Render("  No AVDs or devices found.\n"))
		b.WriteString(mutedStyle.Render("  Create one with `android emulator create`, or plug in a device.\n"))
	}

	for i, it := range m.items {
		cursor := "  "
		if i == m.cursor {
			cursor = headerStyle.Render("▸ ")
		}
		state := stateStoppedStyle.Render("stopped")
		serial := ""
		if it.running && it.device != nil {
			state = stateRunningStyle.Render("running")
			serial = mutedStyle.Render("  " + it.device.Serial)
		}
		name := it.name
		if i == m.cursor {
			name = headerStyle.Render(name)
		}
		b.WriteString(fmt.Sprintf("%s%-28s %s%s\n", cursor, name, state, serial))
	}

	b.WriteString("\n")
	b.WriteString(footerHelp(
		"↑/↓ move", "enter launch", "c cold", "w wipe", "K kill", "l logcat", "s shot", "r refresh", "? help",
	))
	return b.String()
}
