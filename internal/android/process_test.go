package android

import "testing"

func TestParsePS(t *testing.T) {
	// `ps -A -o PID,NAME` form (with header).
	out := "PID NAME\n1234 com.example.app\n1250 com.example.app:push\n88 surfaceflinger\n"
	m := parsePS(out)
	if m["1234"] != "com.example.app" {
		t.Errorf("pid 1234 = %q", m["1234"])
	}
	if m["1250"] != "com.example.app:push" {
		t.Errorf("pid 1250 = %q", m["1250"])
	}
	if m["88"] != "surfaceflinger" {
		t.Errorf("pid 88 = %q", m["88"])
	}

	// Default `ps` form: USER PID PPID ... NAME (PID second, name last).
	def := "USER PID PPID VSZ RSS WCHAN ADDR S NAME\nu0_a1 999 1 100 50 0 0 S com.foo.bar\n"
	dm := parsePS(def)
	if dm["999"] != "com.foo.bar" {
		t.Errorf("default-form pid 999 = %q", dm["999"])
	}
}

func TestParsePackages(t *testing.T) {
	out := "package:com.example.app\npackage:com.foo.bar\n\n"
	pkgs := parsePackages(out)
	if len(pkgs) != 2 || pkgs[0] != "com.example.app" || pkgs[1] != "com.foo.bar" {
		t.Fatalf("parsePackages = %v", pkgs)
	}
}
