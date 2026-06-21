package tui

import (
	"os"
	"path/filepath"
	"time"

	"github.com/carlosmuvi/acli/internal/android"

	tea "github.com/charmbracelet/bubbletea"
)

// --- messages ---

type devicesMsg struct {
	devices []android.Device
	err     error
}

type avdsMsg struct {
	avds []string
	err  error
}

type actionMsg struct {
	verb string // "launch", "kill", ...
	err  error
}

type screenshotMsg struct {
	path string
	err  error
}

type logBatchMsg struct {
	lines  []android.LogLine
	closed bool
}

type tickMsg time.Time

// --- commands ---

func loadDevices(be android.Backend) tea.Cmd {
	return func() tea.Msg {
		d, err := be.Devices()
		return devicesMsg{devices: d, err: err}
	}
}

func loadAVDs(be android.Backend) tea.Cmd {
	return func() tea.Msg {
		a, err := be.ListAVDs()
		return avdsMsg{avds: a, err: err}
	}
}

func launchAVD(be android.Backend, avd string, opts android.LaunchOpts) tea.Cmd {
	return func() tea.Msg {
		return actionMsg{verb: "launch " + avd, err: be.Launch(avd, opts)}
	}
}

func killDevice(be android.Backend, serial string) tea.Cmd {
	return func() tea.Msg {
		return actionMsg{verb: "kill " + serial, err: be.Kill(serial)}
	}
}

func takeScreenshot(be android.Backend, serial, path string) tea.Cmd {
	return func() tea.Msg {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return screenshotMsg{path: path, err: err}
		}
		return screenshotMsg{path: path, err: be.Screenshot(serial, path)}
	}
}

// refreshTick re-polls the device list periodically so newly booted emulators
// appear without manual refresh.
func refreshTick() tea.Cmd {
	return tea.Tick(3*time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}

// readLogs blocks until at least one line is available, then drains whatever
// else is buffered (up to a cap) so high-volume logcat is delivered in batches
// rather than one Update per line.
func readLogs(ch <-chan android.LogLine) tea.Cmd {
	return func() tea.Msg {
		line, ok := <-ch
		if !ok {
			return logBatchMsg{closed: true}
		}
		batch := []android.LogLine{line}
		for len(batch) < 500 {
			select {
			case l, ok := <-ch:
				if !ok {
					return logBatchMsg{lines: batch, closed: true}
				}
				batch = append(batch, l)
			default:
				return logBatchMsg{lines: batch}
			}
		}
		return logBatchMsg{lines: batch}
	}
}
