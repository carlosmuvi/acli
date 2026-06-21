// Package logmirror appends streamed logcat lines to a rotating file under
// ./.acli/logs so Claude Code (in an adjacent pane) can grep the history that
// scrolls past in the TUI. It biases toward keeping history: several rotated
// generations are retained rather than discarded.
package logmirror

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const (
	// LogDir is where per-device log files live, relative to the working dir.
	LogDir = ".acli/logs"
	// maxSize triggers rotation; maxGen is how many old generations are kept.
	maxSize = 10 << 20 // 10 MB
	maxGen  = 5        // <device>.log.1 .. .5  (~60 MB/device retained)
)

// Mirror writes raw log lines for a single device to a rotating file.
type Mirror struct {
	mu   sync.Mutex
	path string
	f    *os.File
	size int64
}

// New opens (creating as needed) the mirror file for the given device name.
func New(device string) (*Mirror, error) {
	if err := os.MkdirAll(LogDir, 0o755); err != nil {
		return nil, err
	}
	path := filepath.Join(LogDir, safeName(device)+".log")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, err
	}
	m := &Mirror{path: path, f: f}
	if fi, err := f.Stat(); err == nil {
		m.size = fi.Size()
	}
	return m, nil
}

// Path returns the file being written (useful to show the user / Claude).
func (m *Mirror) Path() string { return m.path }

// WriteLine appends one raw log line, rotating first if the file is large.
func (m *Mirror) WriteLine(raw string) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.f == nil {
		return
	}
	if m.size >= maxSize {
		m.rotate()
	}
	n, err := m.f.WriteString(raw + "\n")
	if err == nil {
		m.size += int64(n)
	}
}

// Close flushes and closes the underlying file.
func (m *Mirror) Close() {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.f != nil {
		_ = m.f.Close()
		m.f = nil
	}
}

// rotate shifts <path>.N -> <path>.N+1 (dropping the oldest) and starts fresh.
// Callers must hold m.mu.
func (m *Mirror) rotate() {
	_ = m.f.Close()
	m.f = nil
	// Drop the oldest, then shift each generation up by one.
	_ = os.Remove(fmt.Sprintf("%s.%d", m.path, maxGen))
	for i := maxGen - 1; i >= 1; i-- {
		_ = os.Rename(fmt.Sprintf("%s.%d", m.path, i), fmt.Sprintf("%s.%d", m.path, i+1))
	}
	_ = os.Rename(m.path, m.path+".1")

	f, err := os.OpenFile(m.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return // writes become no-ops until next New; avoids crashing the UI
	}
	m.f = f
	m.size = 0
}

// safeName turns a device/AVD name into a filesystem-friendly token.
func safeName(s string) string {
	if s == "" {
		return "device"
	}
	repl := func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_', r == '.':
			return r
		default:
			return '_'
		}
	}
	return strings.Map(repl, s)
}
