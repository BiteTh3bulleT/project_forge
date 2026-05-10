package gateway

import "testing"

func TestNormalizeChmodModeRejectsSpecialBits(t *testing.T) {
	t.Parallel()
	for _, mode := range []string{"1000", "2000", "4000", "7777", "888", "abc"} {
		if _, _, err := normalizeChmodMode(mode); err == nil {
			t.Fatalf("expected chmod mode %q to be rejected", mode)
		}
	}
}

func TestNormalizeChmodModeAllowsPermissionBits(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		raw     string
		wantRaw string
		want    uint32
	}{
		{raw: "", wantRaw: "0644", want: 0o644},
		{raw: "644", wantRaw: "644", want: 0o644},
		{raw: "0755", wantRaw: "0755", want: 0o755},
		{raw: "0000", wantRaw: "0000", want: 0},
	} {
		gotRaw, got, err := normalizeChmodMode(tc.raw)
		if err != nil {
			t.Fatalf("expected chmod mode %q to be accepted: %v", tc.raw, err)
		}
		if gotRaw != tc.wantRaw || got != tc.want {
			t.Fatalf("expected (%q, %#o), got (%q, %#o)", tc.wantRaw, tc.want, gotRaw, got)
		}
	}
}
