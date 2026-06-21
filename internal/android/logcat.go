package android

import (
	"bufio"
	"context"
	"os/exec"
	"strings"
)

// Level is a logcat priority. Ordered so higher values are more severe, which
// lets the UI filter with a single >= comparison.
type Level int

const (
	LevelVerbose Level = iota
	LevelDebug
	LevelInfo
	LevelWarn
	LevelError
	LevelFatal
	LevelUnknown
)

// ParseLevel maps a logcat priority letter to a Level.
func ParseLevel(c byte) Level {
	switch c {
	case 'V':
		return LevelVerbose
	case 'D':
		return LevelDebug
	case 'I':
		return LevelInfo
	case 'W':
		return LevelWarn
	case 'E':
		return LevelError
	case 'F':
		return LevelFatal
	default:
		return LevelUnknown
	}
}

// Letter returns the single-character priority for a Level.
func (l Level) Letter() byte {
	switch l {
	case LevelVerbose:
		return 'V'
	case LevelDebug:
		return 'D'
	case LevelInfo:
		return 'I'
	case LevelWarn:
		return 'W'
	case LevelError:
		return 'E'
	case LevelFatal:
		return 'F'
	default:
		return '?'
	}
}

// String returns the human name of a Level.
func (l Level) String() string {
	switch l {
	case LevelVerbose:
		return "VERBOSE"
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	case LevelFatal:
		return "FATAL"
	default:
		return "?"
	}
}

// LogLine is one parsed logcat record.
type LogLine struct {
	Time  string
	PID   string
	TID   string
	Level Level
	Tag   string
	Msg   string
	Raw   string // original line, for the on-disk mirror
}

// Logcat is a running logcat stream for a single device.
type Logcat struct {
	cancel context.CancelFunc
	Lines  <-chan LogLine
	Done   <-chan error
}

// StartLogcat begins streaming `adb -s <serial> logcat -v threadtime`. Parsed
// lines arrive on Lines; the stream ends (and Done receives the final error,
// possibly nil) when the context is cancelled via Stop or the process exits.
func StartLogcat(adb, serial string) (*Logcat, error) {
	if adb == "" {
		return nil, errNoAdb
	}
	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, adb, "-s", serial, "logcat", "-v", "threadtime")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return nil, err
	}
	cmd.Stderr = nil
	if err := cmd.Start(); err != nil {
		cancel()
		return nil, err
	}

	lines := make(chan LogLine, 1024)
	done := make(chan error, 1)
	go func() {
		defer close(lines)
		sc := bufio.NewScanner(stdout)
		sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for sc.Scan() {
			lines <- parseThreadtime(sc.Text())
		}
		done <- cmd.Wait()
	}()

	return &Logcat{cancel: cancel, Lines: lines, Done: done}, nil
}

// Stop terminates the logcat stream.
func (l *Logcat) Stop() {
	if l != nil && l.cancel != nil {
		l.cancel()
	}
}

// parseThreadtime parses the `threadtime` format:
//
//	MM-DD HH:MM:SS.mmm  PID  TID L TAG: message
//
// Lines that don't match (e.g. "--------- beginning of main") are returned with
// the whole text as Msg and an unknown level.
func parseThreadtime(line string) LogLine {
	ll := LogLine{Raw: line, Level: LevelUnknown, Msg: line}
	fields := strings.Fields(line)
	// Need at least: date time pid tid level tag... message
	if len(fields) < 6 || len(fields[4]) != 1 {
		return ll
	}
	ll.Time = fields[0] + " " + fields[1]
	ll.PID = fields[2]
	ll.TID = fields[3]
	ll.Level = ParseLevel(fields[4][0])

	// Tag runs from after the level up to the colon that separates it from the
	// message. Find that colon in the original line to preserve message spacing.
	rest := strings.TrimLeft(lineAfter(line, fields[4]), " ")
	if idx := strings.IndexByte(rest, ':'); idx >= 0 {
		ll.Tag = strings.TrimSpace(rest[:idx])
		ll.Msg = strings.TrimSpace(rest[idx+1:])
	} else {
		ll.Msg = rest
	}
	return ll
}

// lineAfter returns the remainder of line following the first standalone
// occurrence of the single-character level token.
func lineAfter(line, level string) string {
	// The level is a lone character preceded by whitespace; search for " L ".
	needle := " " + level + " "
	if idx := strings.Index(line, needle); idx >= 0 {
		return line[idx+len(needle):]
	}
	return line
}
