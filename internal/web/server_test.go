package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/carlosmuvi/acli/internal/android"
	"github.com/carlosmuvi/acli/internal/doctor"
	"github.com/carlosmuvi/acli/internal/sdk"
)

// fakeBackend implements android.Backend with canned data and records calls.
type fakeBackend struct {
	avds     []string
	devices  []android.Device
	launched []string
	killed   []string
}

func (f *fakeBackend) Name() string                       { return "fake" }
func (f *fakeBackend) ListAVDs() ([]string, error)        { return f.avds, nil }
func (f *fakeBackend) Devices() ([]android.Device, error) { return f.devices, nil }
func (f *fakeBackend) Launch(avd string, _ android.LaunchOpts) error {
	f.launched = append(f.launched, avd)
	return nil
}
func (f *fakeBackend) Kill(serial string) error {
	f.killed = append(f.killed, serial)
	return nil
}
func (f *fakeBackend) Screenshot(_, path string) error {
	return os.WriteFile(path, []byte("\x89PNG fake"), 0o644)
}

func testServer(b android.Backend) *Server {
	return &Server{
		tools:   sdk.Tools{Adb: "adb"},
		backend: b,
		report: doctor.Report{Checks: []doctor.Check{
			{Name: "adb", OK: true, Hard: true, Detail: "/usr/bin/adb"},
		}},
	}
}

func do(t *testing.T, h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var r *http.Request
	if body != "" {
		r = httptest.NewRequest(method, path, strings.NewReader(body))
	} else {
		r = httptest.NewRequest(method, path, nil)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	return rec
}

func TestStaticAssets(t *testing.T) {
	h := testServer(&fakeBackend{}).Handler()

	idx := do(t, h, "GET", "/", "")
	if idx.Code != 200 || !strings.Contains(idx.Body.String(), "<title>acli</title>") {
		t.Fatalf("index: code=%d body has title=%v", idx.Code, strings.Contains(idx.Body.String(), "<title>"))
	}

	mod := do(t, h, "GET", "/js/main.js", "")
	if mod.Code != 200 {
		t.Fatalf("main.js code=%d", mod.Code)
	}
	if ct := mod.Header().Get("Content-Type"); !strings.Contains(ct, "javascript") {
		t.Fatalf("main.js content-type=%q (must be a javascript type for ES modules)", ct)
	}
}

func TestDoctorEndpoint(t *testing.T) {
	h := testServer(&fakeBackend{}).Handler()
	rec := do(t, h, "GET", "/api/doctor", "")
	var checks []doctor.Check
	if err := json.Unmarshal(rec.Body.Bytes(), &checks); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(checks) != 1 || checks[0].Name != "adb" || !checks[0].OK {
		t.Fatalf("checks = %+v", checks)
	}
}

func TestInventoryEndpoint(t *testing.T) {
	b := &fakeBackend{
		avds: []string{"Pixel_Tablet", "medium_phone"},
		devices: []android.Device{
			{Serial: "emulator-5554", State: "device", Name: "medium_phone", IsEmu: true},
		},
	}
	rec := do(t, testServer(b).Handler(), "GET", "/api/inventory", "")
	var resp struct {
		Backend string          `json:"backend"`
		Entries []android.Entry `json:"entries"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Backend != "fake" || len(resp.Entries) != 2 {
		t.Fatalf("resp = %+v", resp)
	}
	// The running emulator must be matched onto its AVD row (the merge logic).
	var mp android.Entry
	for _, e := range resp.Entries {
		if e.Name == "medium_phone" {
			mp = e
		}
	}
	if !mp.Running || mp.Serial != "emulator-5554" {
		t.Fatalf("medium_phone not matched as running: %+v", mp)
	}
}

func TestLaunchValidationAndDispatch(t *testing.T) {
	b := &fakeBackend{}
	h := testServer(b).Handler()

	if rec := do(t, h, "GET", "/api/launch", ""); rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET launch = %d, want 405", rec.Code)
	}
	if rec := do(t, h, "POST", "/api/launch", `{}`); rec.Code != http.StatusBadRequest {
		t.Errorf("empty launch = %d, want 400", rec.Code)
	}
	if rec := do(t, h, "POST", "/api/launch", `{"avd":"medium_phone","cold":true}`); rec.Code != 200 {
		t.Errorf("valid launch = %d, want 200", rec.Code)
	}
	if len(b.launched) != 1 || b.launched[0] != "medium_phone" {
		t.Errorf("backend.Launch not called: %v", b.launched)
	}
}

func TestKillEndpoint(t *testing.T) {
	b := &fakeBackend{}
	rec := do(t, testServer(b).Handler(), "POST", "/api/kill", `{"serial":"emulator-5554"}`)
	if rec.Code != 200 || len(b.killed) != 1 || b.killed[0] != "emulator-5554" {
		t.Fatalf("kill code=%d killed=%v", rec.Code, b.killed)
	}
}

func TestScreenshotEndpoint(t *testing.T) {
	t.Chdir(t.TempDir()) // screenshots are written under ./.acli/shots
	rec := do(t, testServer(&fakeBackend{}).Handler(), "POST", "/api/screenshot",
		`{"serial":"emulator-5554","name":"medium_phone"}`)
	if rec.Code != 200 {
		t.Fatalf("screenshot code=%d body=%s", rec.Code, rec.Body.String())
	}
	var resp struct{ Path, URL string }
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !strings.HasPrefix(resp.URL, "/shots/") {
		t.Fatalf("url = %q", resp.URL)
	}
	if _, err := os.Stat(resp.Path); err != nil {
		t.Fatalf("screenshot file not written: %v", err)
	}
}
