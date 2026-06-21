package android

import "testing"

func TestParseThreadtime(t *testing.T) {
	cases := []struct {
		name      string
		line      string
		wantLevel Level
		wantTag   string
		wantMsg   string
	}{
		{
			name:      "error line",
			line:      "06-20 14:42:01.123  1234  1250 E AndroidRuntime: FATAL EXCEPTION: main",
			wantLevel: LevelError,
			wantTag:   "AndroidRuntime",
			wantMsg:   "FATAL EXCEPTION: main",
		},
		{
			name:      "info with spaced tag",
			line:      "06-20 14:42:01.500  1234  1234 I ActivityManager: Start proc 1: foo",
			wantLevel: LevelInfo,
			wantTag:   "ActivityManager",
			wantMsg:   "Start proc 1: foo",
		},
		{
			name:      "non-matching banner",
			line:      "--------- beginning of main",
			wantLevel: LevelUnknown,
			wantTag:   "",
			wantMsg:   "--------- beginning of main",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := parseThreadtime(c.line)
			if got.Level != c.wantLevel {
				t.Errorf("level = %v, want %v", got.Level, c.wantLevel)
			}
			if got.Tag != c.wantTag {
				t.Errorf("tag = %q, want %q", got.Tag, c.wantTag)
			}
			if got.Msg != c.wantMsg {
				t.Errorf("msg = %q, want %q", got.Msg, c.wantMsg)
			}
			if got.Raw != c.line {
				t.Errorf("raw not preserved")
			}
		})
	}
}

func TestParseAVDList(t *testing.T) {
	out := "Pixel_7_API_34\nPixel_Tablet_API_34\n"
	got := parseAVDList(out)
	if len(got) != 2 || got[0] != "Pixel_7_API_34" || got[1] != "Pixel_Tablet_API_34" {
		t.Fatalf("parseAVDList = %v", got)
	}
}
