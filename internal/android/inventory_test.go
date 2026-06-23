package android

import "testing"

func TestMerge(t *testing.T) {
	avds := []string{"Pixel_Tablet", "medium_phone"}
	devices := []Device{
		{Serial: "emulator-5554", State: "device", Name: "medium_phone", IsEmu: true},
		{Serial: "abc123", State: "device", Name: "Pixel 9", IsEmu: false},
	}
	got := Merge(avds, devices)

	// Running entries (the running emulator + the physical device) sort first.
	if !got[0].Running || !got[1].Running {
		t.Fatalf("running entries should sort first: %+v", got)
	}
	// medium_phone is matched to its emulator.
	var mp *Entry
	for i := range got {
		if got[i].Name == "medium_phone" {
			mp = &got[i]
		}
	}
	if mp == nil || !mp.Running || mp.Serial != "emulator-5554" || mp.AVD != "medium_phone" {
		t.Fatalf("medium_phone not matched to its emulator: %+v", mp)
	}
	// Pixel_Tablet stays a stopped, launchable AVD.
	var pt *Entry
	for i := range got {
		if got[i].Name == "Pixel_Tablet" {
			pt = &got[i]
		}
	}
	if pt == nil || pt.Running || pt.AVD != "Pixel_Tablet" {
		t.Fatalf("Pixel_Tablet should be a stopped AVD: %+v", pt)
	}
	if len(got) != 3 {
		t.Fatalf("want 3 entries, got %d: %+v", len(got), got)
	}
}
