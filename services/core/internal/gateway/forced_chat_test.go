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
		{name: "status check", in: "How are we looking?", want: true},
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

func TestIsLikelyStatusProbeTurn(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		in   string
		want bool
	}{
		{name: "status of core", in: "What is the status of the core?", want: true},
		{name: "status check", in: "How are we looking?", want: true},
		{name: "plain update request", in: "Any updates on this task?", want: true},
		{name: "did it work phrasing", in: "Well that seemed to work, didn't it?", want: true},
		{name: "operational intent", in: "Run a status command", want: false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := IsLikelyStatusProbeTurn(tc.in)
			if got != tc.want {
				t.Fatalf("IsLikelyStatusProbeTurn(%q)=%v want %v", tc.in, got, tc.want)
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
		{name: "status check disabled", in: "How are we looking?", want: false},
		{name: "status probe disabled", in: "What is the status of the core?", want: false},
		{name: "did it work disabled", in: "Well that seemed to work, didn't it?", want: false},
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

func TestForcedChatModelNameCompositeWorkflowNotForced(t *testing.T) {
	t.Parallel()
	in := `Open konsole, cd to Projects/. run mkdir and create new directory called "MyFirstTele". Inside that directory create a python app that is a clock that tells jokes on the hour.`
	if got := ForcedChatModelName(in); got != "" {
		t.Fatalf("ForcedChatModelName should not force single tool for composite workflow, got %q", got)
	}
}

func TestForcedChatModelNameSimpleMkdirStillForced(t *testing.T) {
	t.Parallel()
	in := `create a directory called "MyFirstTele"`
	if got := ForcedChatModelName(in); got != ChatModelName("fs.mkdir") {
		t.Fatalf("ForcedChatModelName should force fs.mkdir for simple mkdir, got %q", got)
	}
}
