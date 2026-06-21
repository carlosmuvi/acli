// Package tui implements acli's interactive terminal UI: an emulator/device
// list, a live filterable logcat view, screenshots, and a doctor screen.
package tui

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/carlosmuvi/acli/internal/android"
	"github.com/carlosmuvi/acli/internal/doctor"
	"github.com/carlosmuvi/acli/internal/sdk"

	tea "github.com/charmbracelet/bubbletea"
)

type view int

const (
	viewEmulators view = iota
	viewLogcat
	viewDoctor
)

// ShotDir is where screenshots are written.
const ShotDir = ".acli/shots"

// Model is acli's root Bubble Tea model.
type Model struct {
	tools   sdk.Tools
	backend android.Backend
	report  doctor.Report

	view      view
	prevView  view // to return from the doctor overlay
	emulators emulatorsModel
	logcat    logcatModel

	status        string // transient status line (last action / error)
	width, height int
}

// New builds the root model from discovered tooling and a doctor report.
func New(tools sdk.Tools, report doctor.Report) Model {
	return Model{
		tools:     tools,
		backend:   android.Select(tools),
		report:    report,
		view:      viewEmulators,
		emulators: emulatorsModel{},
		logcat:    newLogcatModel(),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(loadAVDs(m.backend), loadDevices(m.backend), refreshTick())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.emulators.setSize(msg.Width, msg.Height)
		m.logcat.setSize(msg.Width, msg.Height)
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case avdsMsg:
		if msg.err != nil {
			m.status = errStatus("list AVDs", msg.err)
		}
		m.emulators.rebuild(msg.avds, devicesFrom(m.emulators))
		// AVDs alone don't include running state; a devices refresh follows via tick.
		return m, nil

	case devicesMsg:
		if msg.err != nil {
			m.status = errStatus("list devices", msg.err)
		}
		m.emulators.rebuild(avdsFrom(m.emulators), msg.devices)
		return m, nil

	case tickMsg:
		return m, tea.Batch(loadDevices(m.backend), refreshTick())

	case actionMsg:
		if msg.err != nil {
			m.status = errStatus(msg.verb, msg.err)
		} else {
			m.status = okStyle.Render("✓ " + msg.verb)
		}
		return m, loadDevices(m.backend)

	case screenshotMsg:
		if msg.err != nil {
			m.status = errStatus("screenshot", msg.err)
		} else {
			m.status = okStyle.Render("✓ saved " + msg.path)
		}
		return m, nil

	case logBatchMsg:
		var cmd tea.Cmd
		m.logcat, cmd = m.logcat.Update(msg)
		return m, cmd
	}

	// Forward anything else to the active view.
	if m.view == viewLogcat {
		var cmd tea.Cmd
		m.logcat, cmd = m.logcat.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Quit is global, except while typing in the logcat filter.
	if matches(msg, keys.Quit) && !(m.view == viewLogcat && m.logcat.editing) {
		m.logcat.stop()
		return m, tea.Quit
	}

	switch m.view {
	case viewDoctor:
		// Any key dismisses the doctor overlay.
		m.view = m.prevView
		return m, nil

	case viewLogcat:
		// esc returns to the emulator list (unless editing the filter).
		if matches(msg, keys.Back) && !m.logcat.editing {
			m.logcat.stop()
			m.view = viewEmulators
			return m, nil
		}
		if (matches(msg, keys.Help) || matches(msg, keys.Doctor)) && !m.logcat.editing {
			m.prevView = viewLogcat
			m.view = viewDoctor
			return m, nil
		}
		var cmd tea.Cmd
		m.logcat, cmd = m.logcat.Update(msg)
		return m, cmd

	default: // viewEmulators
		return m.handleEmulatorKey(msg)
	}
}

func (m Model) handleEmulatorKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case matches(msg, keys.Help), matches(msg, keys.Doctor):
		m.prevView = viewEmulators
		m.view = viewDoctor
		return m, nil

	case matches(msg, keys.Refresh):
		m.status = mutedStyle.Render("refreshing…")
		return m, tea.Batch(loadAVDs(m.backend), loadDevices(m.backend))

	case matches(msg, keys.Launch):
		if it, ok := m.emulators.selected(); ok {
			if it.running {
				return m.openLogcat(it)
			}
			if it.avd != "" {
				m.status = mutedStyle.Render("launching " + it.avd + "…")
				return m, launchAVD(m.backend, it.avd, android.LaunchOpts{})
			}
		}
		return m, nil

	case matches(msg, keys.ColdBoot):
		if it, ok := m.emulators.selected(); ok && it.avd != "" {
			m.status = mutedStyle.Render("cold booting " + it.avd + "…")
			return m, launchAVD(m.backend, it.avd, android.LaunchOpts{ColdBoot: true})
		}
		return m, nil

	case matches(msg, keys.Wipe):
		if it, ok := m.emulators.selected(); ok && it.avd != "" {
			m.status = mutedStyle.Render("wipe+launch " + it.avd + "…")
			return m, launchAVD(m.backend, it.avd, android.LaunchOpts{WipeData: true})
		}
		return m, nil

	case matches(msg, keys.Kill):
		if it, ok := m.emulators.selected(); ok && it.running && it.device != nil {
			m.status = mutedStyle.Render("killing " + it.device.Serial + "…")
			return m, killDevice(m.backend, it.device.Serial)
		}
		return m, nil

	case matches(msg, keys.Logs):
		if it, ok := m.emulators.selected(); ok && it.running {
			return m.openLogcat(it)
		}
		m.status = mutedStyle.Render("device not running")
		return m, nil

	case matches(msg, keys.Shot):
		if it, ok := m.emulators.selected(); ok && it.running && it.device != nil {
			path := screenshotPath(it.device.Name)
			m.status = mutedStyle.Render("capturing…")
			return m, takeScreenshot(m.backend, it.device.Serial, path)
		}
		m.status = mutedStyle.Render("device not running")
		return m, nil
	}

	m.emulators.Update(msg)
	return m, nil
}

func (m Model) openLogcat(it emuItem) (tea.Model, tea.Cmd) {
	if it.device == nil {
		return m, nil
	}
	m.view = viewLogcat
	m.logcat.setSize(m.width, m.height)
	cmd := m.logcat.start(m.tools.Adb, *it.device)
	return m, cmd
}

func (m Model) View() string {
	switch m.view {
	case viewLogcat:
		return m.logcat.View()
	case viewDoctor:
		return m.doctorView()
	default:
		return m.emulatorView()
	}
}

func (m Model) emulatorView() string {
	top := titleStyle.Render("acli") + "  " + mutedStyle.Render("backend: "+m.backend.Name())
	body := m.emulators.View()
	status := m.status
	if status == "" {
		status = mutedStyle.Render("ready")
	}
	return strings.Join([]string{top, "", body, "", status}, "\n")
}

func (m Model) doctorView() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("doctor") + "\n\n")
	for _, c := range m.report.Checks {
		mark := okStyle.Render("✓")
		if !c.OK {
			mark = badStyle.Render("✗")
		}
		b.WriteString(fmt.Sprintf("  %s %-18s %s\n", mark, c.Name, mutedStyle.Render(c.Detail)))
		if !c.OK {
			b.WriteString("      " + mutedStyle.Render("fix: "+c.Fix) + "\n")
		}
	}
	b.WriteString("\n")
	b.WriteString(footerHelp("any key to dismiss"))
	return b.String()
}

// screenshotPath builds a timestamped PNG path for a device.
func screenshotPath(name string) string {
	if name == "" {
		name = "device"
	}
	safe := strings.Map(func(r rune) rune {
		if r == ' ' || r == '/' {
			return '_'
		}
		return r
	}, name)
	ts := time.Now().Format("150405")
	return filepath.Join(ShotDir, fmt.Sprintf("%s-%s.png", safe, ts))
}

func errStatus(verb string, err error) string {
	return badStyle.Render(fmt.Sprintf("✗ %s: %v", verb, err))
}

// avdsFrom / devicesFrom reconstruct the inputs to rebuild() from current items
// so an AVD-only or device-only refresh preserves the other dimension.
func avdsFrom(m emulatorsModel) []string {
	var avds []string
	for _, it := range m.items {
		if it.avd != "" {
			avds = append(avds, it.avd)
		}
	}
	return avds
}

func devicesFrom(m emulatorsModel) []android.Device {
	var devs []android.Device
	for _, it := range m.items {
		if it.device != nil {
			devs = append(devs, *it.device)
		}
	}
	return devs
}
