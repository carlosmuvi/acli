// Package web serves a small browser dashboard for acli: an emulator/device
// list with launch/kill/screenshot actions and a live logcat stream. It reuses
// the same backend as the TUI (internal/android, internal/doctor, internal/sdk)
// and adds only an HTTP presentation layer.
package web

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/carlosmuvi/acli/internal/android"
	"github.com/carlosmuvi/acli/internal/doctor"
	"github.com/carlosmuvi/acli/internal/logmirror"
	"github.com/carlosmuvi/acli/internal/sdk"
)

//go:embed all:static
var staticFS embed.FS

// shotDir is where screenshots taken from the web UI are written.
const shotDir = ".acli/shots"

// Server holds the resolved tooling and selected backend.
type Server struct {
	tools   sdk.Tools
	backend android.Backend
	report  doctor.Report
}

// NewServer builds a Server from discovered tooling and a doctor report.
func NewServer(tools sdk.Tools, report doctor.Report) *Server {
	return &Server{tools: tools, backend: android.Select(tools), report: report}
}

// Handler returns the HTTP handler for the dashboard.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	sub, _ := fs.Sub(staticFS, "static")
	mux.Handle("/", http.FileServer(http.FS(sub)))
	mux.Handle("/shots/", http.StripPrefix("/shots/", http.FileServer(http.Dir(shotDir))))

	mux.HandleFunc("/api/doctor", s.handleDoctor)
	mux.HandleFunc("/api/inventory", s.handleInventory)
	mux.HandleFunc("/api/launch", s.handleLaunch)
	mux.HandleFunc("/api/kill", s.handleKill)
	mux.HandleFunc("/api/screenshot", s.handleScreenshot)
	mux.HandleFunc("/api/logcat", s.handleLogcat)
	mux.HandleFunc("/api/meta", s.handleMeta)
	return mux
}

func (s *Server) handleDoctor(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, s.report.Checks)
}

func (s *Server) handleInventory(w http.ResponseWriter, r *http.Request) {
	avds, errA := s.backend.ListAVDs()
	devices, errD := s.backend.Devices()
	resp := map[string]any{
		"backend": s.backend.Name(),
		"entries": android.Merge(avds, devices),
	}
	if errA != nil {
		resp["avdError"] = errA.Error()
	}
	if errD != nil {
		resp["deviceError"] = errD.Error()
	}
	writeJSON(w, resp)
}

func (s *Server) handleLaunch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		AVD  string `json:"avd"`
		Cold bool   `json:"cold"`
		Wipe bool   `json:"wipe"`
	}
	if !decode(w, r, &req) {
		return
	}
	if req.AVD == "" {
		httpErr(w, http.StatusBadRequest, "missing avd")
		return
	}
	err := s.backend.Launch(req.AVD, android.LaunchOpts{ColdBoot: req.Cold, WipeData: req.Wipe})
	writeResult(w, err)
}

func (s *Server) handleKill(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Serial string `json:"serial"`
	}
	if !decode(w, r, &req) {
		return
	}
	if req.Serial == "" {
		httpErr(w, http.StatusBadRequest, "missing serial")
		return
	}
	writeResult(w, s.backend.Kill(req.Serial))
}

func (s *Server) handleScreenshot(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Serial string `json:"serial"`
		Name   string `json:"name"`
	}
	if !decode(w, r, &req) {
		return
	}
	if err := os.MkdirAll(shotDir, 0o755); err != nil {
		httpErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	name := req.Name
	if name == "" {
		name = req.Serial
	}
	file := fmt.Sprintf("%s-%s.png", safeFile(name), time.Now().Format("150405"))
	path := filepath.Join(shotDir, file)
	if err := s.backend.Screenshot(req.Serial, path); err != nil {
		httpErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, map[string]string{"path": path, "url": "/shots/" + file})
}

// handleLogcat streams a device's logcat as Server-Sent Events and mirrors each
// raw line to disk (same as the TUI), so an agent can grep the history too.
func (s *Server) handleLogcat(w http.ResponseWriter, r *http.Request) {
	serial := r.URL.Query().Get("serial")
	if serial == "" {
		httpErr(w, http.StatusBadRequest, "missing serial")
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		httpErr(w, http.StatusInternalServerError, "streaming unsupported")
		return
	}

	lc, err := android.StartLogcat(s.tools.Adb, serial)
	if err != nil {
		httpErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer lc.Stop()

	name := r.URL.Query().Get("name")
	if name == "" {
		name = serial
	}
	var mirror *logmirror.Mirror
	if mr, err := logmirror.New(name); err == nil {
		mirror = mr
		defer mirror.Close()
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher.Flush()

	// PID→process map, refreshed periodically, so we can label each line with a
	// package/process name for `package:`/`process:` filtering.
	pidName := android.ProcessMap(s.tools.Adb, serial)
	refresh := time.NewTicker(4 * time.Second)
	defer refresh.Stop()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case <-refresh.C:
			if m := android.ProcessMap(s.tools.Adb, serial); len(m) > 0 {
				pidName = m
			}
		case line, ok := <-lc.Lines:
			if !ok {
				fmt.Fprint(w, "event: end\ndata: {}\n\n")
				flusher.Flush()
				return
			}
			if mirror != nil {
				mirror.WriteLine(line.Raw)
			}
			payload, _ := json.Marshal(map[string]any{
				"time":    clockOnly(line.Time),
				"level":   string(line.Level.Letter()),
				"prio":    int(line.Level),
				"tag":     line.Tag,
				"msg":     line.Msg,
				"pid":     line.PID,
				"process": pidName[line.PID],
			})
			fmt.Fprintf(w, "data: %s\n\n", payload)
			flusher.Flush()
		}
	}
}

// handleMeta returns values for filter autocomplete: third-party packages
// (for `package:mine`) and the current process names.
func (s *Server) handleMeta(w http.ResponseWriter, r *http.Request) {
	serial := r.URL.Query().Get("serial")
	if serial == "" {
		httpErr(w, http.StatusBadRequest, "missing serial")
		return
	}
	pkgs := android.ThirdPartyPackages(s.tools.Adb, serial)
	procs := android.ProcessMap(s.tools.Adb, serial)
	var procNames []string
	seen := map[string]bool{}
	for _, n := range procs {
		base := n
		if i := strings.IndexByte(base, ':'); i >= 0 {
			base = base[:i] // collapse "com.foo:svc" → "com.foo"
		}
		if base != "" && !seen[base] {
			seen[base] = true
			procNames = append(procNames, base)
		}
	}
	writeJSON(w, map[string]any{"mine": pkgs, "processes": procNames})
}

// --- helpers ---

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func writeResult(w http.ResponseWriter, err error) {
	if err != nil {
		httpErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, map[string]bool{"ok": true})
}

func httpErr(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	if r.Method != http.MethodPost {
		httpErr(w, http.StatusMethodNotAllowed, "POST required")
		return false
	}
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		httpErr(w, http.StatusBadRequest, "bad json: "+err.Error())
		return false
	}
	return true
}

func clockOnly(t string) string {
	if i := strings.IndexByte(t, ' '); i >= 0 {
		return t[i+1:]
	}
	return t
}

func safeFile(s string) string {
	if s == "" {
		return "device"
	}
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_', r == '.':
			return r
		default:
			return '_'
		}
	}, s)
}

// Listen starts the HTTP server on the given port and blocks. addr is returned
// via the provided callback once listening (for opening a browser).
func (s *Server) Listen(port int) error {
	addr := ":" + strconv.Itoa(port)
	return http.ListenAndServe(addr, s.Handler())
}
