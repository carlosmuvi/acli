package tui

import (
	"strings"
	"testing"

	"github.com/carlosmuvi/acli/internal/android"
	"github.com/carlosmuvi/acli/internal/doctor"
	"github.com/carlosmuvi/acli/internal/sdk"

	tea "github.com/charmbracelet/bubbletea"
)

// drive sends a message and returns the updated concrete Model.
func drive(m Model, msg tea.Msg) Model {
	next, _ := m.Update(msg)
	return next.(Model)
}

func TestEmulatorListRendersAVDsAndDevices(t *testing.T) {
	m := New(sdk.Tools{}, doctor.Report{})
	m = drive(m, tea.WindowSizeMsg{Width: 100, Height: 30})
	m = drive(m, avdsMsg{avds: []string{"Pixel_7_API_34", "Pixel_Tablet"}})
	m = drive(m, devicesMsg{devices: []android.Device{
		{Serial: "emulator-5554", State: "device", Name: "Pixel_7_API_34", IsEmu: true},
	}})

	view := m.View()
	if !strings.Contains(view, "Pixel_7_API_34") || !strings.Contains(view, "Pixel_Tablet") {
		t.Fatalf("emulator view missing entries:\n%s", view)
	}
	if !strings.Contains(view, "running") || !strings.Contains(view, "stopped") {
		t.Fatalf("expected running+stopped states:\n%s", view)
	}
	// Running emulator should sort first.
	if strings.Index(view, "Pixel_7_API_34") > strings.Index(view, "Pixel_Tablet") {
		t.Errorf("running emulator should be listed before stopped one")
	}
}

func TestLogcatFilteringByLevelAndText(t *testing.T) {
	lc := newLogcatModel()
	lc.setSize(100, 30)
	lc.active = true

	lines := []android.LogLine{
		{Level: android.LevelInfo, Tag: "Foo", Msg: "hello world", Raw: "i"},
		{Level: android.LevelError, Tag: "Bar", Msg: "boom crash", Raw: "e"},
		{Level: android.LevelDebug, Tag: "Baz", Msg: "noisy debug", Raw: "d"},
	}
	lc, _ = lc.Update(logBatchMsg{lines: lines})

	// No filter: all three visible.
	if got := strings.Count(lc.viewport.View(), "\n"); got < 0 {
		t.Fatal("viewport not initialized")
	}
	all := lc.viewport.View()
	for _, want := range []string{"hello world", "boom crash", "noisy debug"} {
		if !strings.Contains(all, want) {
			t.Fatalf("unfiltered view missing %q:\n%s", want, all)
		}
	}

	// Raise min level to ERROR: only the error line survives.
	lc.minLevel = android.LevelError
	lc.refreshContent()
	lvl := lc.viewport.View()
	if strings.Contains(lvl, "hello world") || strings.Contains(lvl, "noisy debug") {
		t.Errorf("level filter did not drop lower-priority lines:\n%s", lvl)
	}
	if !strings.Contains(lvl, "boom crash") {
		t.Errorf("level filter dropped the error line:\n%s", lvl)
	}

	// Text filter on a verbose level.
	lc.minLevel = android.LevelVerbose
	lc.filter.SetValue("noisy")
	lc.refreshContent()
	txt := lc.viewport.View()
	if strings.Contains(txt, "hello world") || strings.Contains(txt, "boom crash") {
		t.Errorf("text filter did not exclude non-matches:\n%s", txt)
	}
	if !strings.Contains(txt, "noisy debug") {
		t.Errorf("text filter dropped the match:\n%s", txt)
	}
}

func TestDoctorOverlayToggles(t *testing.T) {
	m := New(sdk.Tools{}, doctor.Report{Checks: []doctor.Check{
		{Name: "adb", OK: true, Detail: "/usr/bin/adb"},
	}})
	m = drive(m, tea.WindowSizeMsg{Width: 80, Height: 24})
	m = drive(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	if m.view != viewDoctor {
		t.Fatalf("expected doctor view, got %v", m.view)
	}
	if !strings.Contains(m.View(), "doctor") {
		t.Errorf("doctor view not rendered")
	}
	// Any key dismisses.
	m = drive(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if m.view != viewEmulators {
		t.Errorf("doctor overlay did not dismiss, view=%v", m.view)
	}
}
