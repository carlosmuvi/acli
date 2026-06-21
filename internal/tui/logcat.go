package tui

import (
	"fmt"
	"strings"

	"github.com/carlosmuvi/acli/internal/android"
	"github.com/carlosmuvi/acli/internal/logmirror"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const logBufferCap = 5000

// logcatModel streams and displays logcat for a single device with in-memory
// level + text filtering, while mirroring every raw line to disk.
type logcatModel struct {
	adb    string
	device android.Device

	stream *android.Logcat
	mirror *logmirror.Mirror

	buffer   []android.LogLine
	viewport viewport.Model
	filter   textinput.Model

	minLevel android.Level
	editing  bool // filter input focused
	paused   bool
	active   bool // a stream is running

	width, height int
	ready         bool
}

func newLogcatModel() logcatModel {
	ti := textinput.New()
	ti.Placeholder = "filter tag/message…"
	ti.Prompt = "/ "
	ti.CharLimit = 120
	return logcatModel{
		filter:   ti,
		minLevel: android.LevelVerbose,
	}
}

// start opens a fresh mirror + logcat stream for the device.
func (m *logcatModel) start(adb string, d android.Device) tea.Cmd {
	m.stop()
	m.adb = adb
	m.device = d
	m.buffer = m.buffer[:0]
	m.paused = false

	stream, err := android.StartLogcat(adb, d.Serial)
	if err != nil {
		m.active = false
		return func() tea.Msg { return actionMsg{verb: "logcat", err: err} }
	}
	m.stream = stream
	m.active = true
	// Best-effort mirror; logging still works in the UI if the file can't open.
	if mr, err := logmirror.New(d.Name); err == nil {
		m.mirror = mr
	}
	m.refreshContent()
	return readLogs(stream.Lines)
}

// stop tears down the current stream and mirror.
func (m *logcatModel) stop() {
	if m.stream != nil {
		m.stream.Stop()
		m.stream = nil
	}
	if m.mirror != nil {
		m.mirror.Close()
		m.mirror = nil
	}
	m.active = false
}

func (m *logcatModel) setSize(w, h int) {
	m.width, m.height = w, h
	// Reserve rows: header (1), blank (1), filter (1), footer (2).
	vpHeight := h - 5
	if vpHeight < 3 {
		vpHeight = 3
	}
	if !m.ready {
		m.viewport = viewport.New(w, vpHeight)
		m.ready = true
	} else {
		m.viewport.Width = w
		m.viewport.Height = vpHeight
	}
	m.refreshContent()
}

func (m *logcatModel) Update(msg tea.Msg) (logcatModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case logBatchMsg:
		for _, l := range msg.lines {
			m.append(l)
		}
		if !m.paused && len(msg.lines) > 0 {
			m.refreshContent()
			m.viewport.GotoBottom()
		}
		if msg.closed {
			m.active = false
		} else if m.stream != nil {
			cmds = append(cmds, readLogs(m.stream.Lines))
		}
		return *m, tea.Batch(cmds...)

	case tea.KeyMsg:
		if m.editing {
			switch msg.String() {
			case "enter", "esc":
				m.editing = false
				m.filter.Blur()
				m.refreshContent()
				return *m, nil
			}
			var cmd tea.Cmd
			m.filter, cmd = m.filter.Update(msg)
			m.refreshContent()
			return *m, cmd
		}

		switch {
		case matches(msg, keys.Filter):
			m.editing = true
			m.filter.Focus()
			return *m, textinput.Blink
		case matches(msg, keys.Level):
			m.cycleLevel()
			m.refreshContent()
			return *m, nil
		case matches(msg, keys.PauseLog):
			m.paused = !m.paused
			if !m.paused {
				m.refreshContent()
				m.viewport.GotoBottom()
			}
			return *m, nil
		case matches(msg, keys.ClearLog):
			m.buffer = m.buffer[:0]
			m.refreshContent()
			return *m, nil
		}
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return *m, cmd
}

// append adds a line to the ring buffer and mirrors its raw form to disk.
func (m *logcatModel) append(l android.LogLine) {
	if m.mirror != nil {
		m.mirror.WriteLine(l.Raw)
	}
	m.buffer = append(m.buffer, l)
	if len(m.buffer) > logBufferCap {
		// Drop oldest, keep cap. Copy to avoid unbounded backing-array growth.
		n := copy(m.buffer, m.buffer[len(m.buffer)-logBufferCap:])
		m.buffer = m.buffer[:n]
	}
}

func (m *logcatModel) cycleLevel() {
	m.minLevel++
	if m.minLevel > android.LevelFatal {
		m.minLevel = android.LevelVerbose
	}
}

// refreshContent rebuilds the viewport from the buffer applying the active
// level and text filters.
func (m *logcatModel) refreshContent() {
	if !m.ready {
		return
	}
	needle := strings.ToLower(strings.TrimSpace(m.filter.Value()))
	var b strings.Builder
	for _, l := range m.buffer {
		if l.Level < m.minLevel && l.Level != android.LevelUnknown {
			continue
		}
		if needle != "" && !strings.Contains(strings.ToLower(l.Tag+" "+l.Msg), needle) {
			continue
		}
		b.WriteString(renderLine(l, m.width))
		b.WriteByte('\n')
	}
	m.viewport.SetContent(strings.TrimRight(b.String(), "\n"))
}

// renderLine formats one log record: "HH:MM:SS L Tag: message", colored.
func renderLine(l android.LogLine, width int) string {
	t := l.Time
	if i := strings.IndexByte(t, ' '); i >= 0 {
		t = t[i+1:] // drop the date, keep the clock
	}
	st := levelStyle(l.Level)
	prefix := fmt.Sprintf("%s %c ", t, l.Level.Letter())
	tag := l.Tag
	if tag != "" {
		tag += ": "
	}
	line := prefix + tag + l.Msg
	if width > 4 && lipgloss.Width(line) > width {
		line = truncate(line, width)
	}
	return st.Render(line)
}

func (m logcatModel) View() string {
	dev := m.device.Name
	if dev == "" {
		dev = m.device.Serial
	}
	status := okStyle.Render("● live")
	if m.paused {
		status = mutedStyle.Render("❚❚ paused")
	} else if !m.active {
		status = badStyle.Render("○ ended")
	}
	header := headerStyle.Render("Logcat ▸ "+dev) +
		mutedStyle.Render(fmt.Sprintf("   level≥%s   %s", m.minLevel, status))

	filterLine := mutedStyle.Render(m.filter.Prompt + dim(m.filter.Value(), "(no filter)"))
	if m.editing {
		filterLine = m.filter.View()
	}

	help := footerHelp(
		"f filter", "L level", "space pause", "x clear", "esc back", "? help",
	)

	return strings.Join([]string{
		header,
		m.viewport.View(),
		filterLine,
		help,
	}, "\n")
}

func dim(s, placeholder string) string {
	if strings.TrimSpace(s) == "" {
		return placeholder
	}
	return s
}
