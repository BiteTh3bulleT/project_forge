package sqlutil

import "testing"

func TestPlaceholders(t *testing.T) {
	tests := []struct {
		name string
		n    int
		want string
	}{
		{name: "zero", n: 0, want: ""},
		{name: "negative", n: -2, want: ""},
		{name: "one", n: 1, want: "?"},
		{name: "many", n: 3, want: "?,?,?"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Placeholders(tt.n); got != tt.want {
				t.Fatalf("Placeholders(%d) = %q, want %q", tt.n, got, tt.want)
			}
		})
	}
}
