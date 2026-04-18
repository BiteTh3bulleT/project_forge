package gateway

import "testing"

func TestIsLikelySmallTalkTurn(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		in   string
		want bool
	}{
		{name: "exact bravo", in: "Bravo lad", want: true},
		{name: "thanks prefix", in: "Thanks for the fix", want: true},
		{name: "action request", in: "Create scratch/Test2 and write test.txt", want: false},
		{name: "question", in: "How do you operate?", want: false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := IsLikelySmallTalkTurn(tc.in)
			if got != tc.want {
				t.Fatalf("IsLikelySmallTalkTurn(%q)=%v want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestShouldAttachChatTools(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		in   string
		want bool
	}{
		{name: "small talk disabled", in: "Bravo lad", want: false},
		{name: "thanks disabled", in: "Thanks", want: false},
		{name: "python banner enabled", in: `Create scratch/Python directory. Inside the directory create a simple scrolling banner python script that says "FORGE LIVES!" in vegas lights font.`, want: true},
		{name: "mkdir write enabled", in: `create a file labeled "test.txt" inside scratch/not_another_test/ and inside said file the words "This is a test file"`, want: true},
		{name: "shell run enabled", in: "run go test ./...", want: true},
		{name: "chat question disabled", in: "How do you operate?", want: false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := ShouldAttachChatTools(tc.in)
			if got != tc.want {
				t.Fatalf("ShouldAttachChatTools(%q)=%v want %v", tc.in, got, tc.want)
			}
		})
	}
}
