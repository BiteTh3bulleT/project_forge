package api

import "testing"

func TestNormalizeWakeCommand(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{in: "/wake", want: "wake_forge"},
		{in: "wake forge", want: "wake_forge"},
		{in: "forge wake", want: "wake_forge"},
		{in: "/wake pc", want: "wake_computer"},
		{in: "wake computer", want: "wake_computer"},
		{in: "hello", want: ""},
	}
	for _, tc := range tests {
		if got := normalizeWakeCommand(tc.in); got != tc.want {
			t.Fatalf("normalizeWakeCommand(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}

func TestParseMAC(t *testing.T) {
	valid := []string{
		"AA:BB:CC:DD:EE:FF",
		"aa-bb-cc-dd-ee-ff",
		"aabb.ccdd.eeff",
		"aabbccddeeff",
	}
	for _, in := range valid {
		got, err := parseMAC(in)
		if err != nil {
			t.Fatalf("parseMAC(%q) unexpected err: %v", in, err)
		}
		if len(got) != 6 {
			t.Fatalf("parseMAC(%q) len=%d want 6", in, len(got))
		}
	}
	if _, err := parseMAC("xyz"); err == nil {
		t.Fatalf("parseMAC invalid input expected error")
	}
}
