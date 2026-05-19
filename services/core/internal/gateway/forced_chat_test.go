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
		{name: "did not work phrasing", in: "That didn't seem to work", want: true},
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
		{name: "web search enabled", in: "search the web for forge ai os", want: true},
		{name: "current news enabled", in: "Was there anything exciting in the news today?", want: true},
		{name: "weather with location enabled", in: "what is the weather in Chicago today?", want: true},
		{name: "weather without location disabled", in: "what is the weather looking like today?", want: false},
		{name: "browser open enabled", in: "open browser https://example.com", want: true},
		{name: "repo exploration enabled", in: "You can explore your repo. Familiarize yourself with yourself.", want: true},
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

func TestForcedChatModelNameWebAndBrowser(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		in   string
		want string
	}{
		{name: "web search", in: "search the web for FORGE docs", want: ChatModelName("web.search")},
		{name: "current news", in: "Was there anything exciting in the news today?", want: ChatModelName("web.search")},
		{name: "weather with location", in: "what is the weather in Chicago today?", want: ChatModelName("web.search")},
		{name: "fetch url", in: "fetch https://example.com", want: ChatModelName("net.fetch")},
		{name: "open browser", in: "open browser https://example.com", want: ChatModelName("desktop.open")},
		{name: "open chrome app", in: "Open google chrome please.", want: ChatModelName("desktop.open")},
		{name: "open file explorer", in: "Can you open file explorer please", want: ChatModelName("desktop.open")},
		{name: "open file explorer typo", in: "Can you open file expolorer for me please?", want: ChatModelName("desktop.open")},
		{name: "open terminal and run", in: "Open terminal and run sudo zypper refresh", want: ChatModelName("desktop.open")},
		{name: "open terminal ssh workflow", in: "Open terminal, ssh into robert@10.150.1.9 password test-pass. Create a directory labled SSH-AI-TEST", want: ChatModelName("desktop.open")},
		{name: "repo exploration", in: "You can explore your repo. Familiarize yourself with yourself.", want: ChatModelName("repo.inspect")},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := ForcedChatModelName(tc.in); got != tc.want {
				t.Fatalf("ForcedChatModelName(%q)=%q want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestParseWebSearchQueryAndURL(t *testing.T) {
	t.Parallel()
	query, ok := ParseWebSearchQuery("search the web for model runtime adapters")
	if !ok || query != "model runtime adapters" {
		t.Fatalf("ParseWebSearchQuery got %q ok=%v", query, ok)
	}
	query, ok = ParseWebSearchQuery("Was there anything exciting in the news today?")
	if !ok || query != "latest notable news headlines today" {
		t.Fatalf("ParseWebSearchQuery current news got %q ok=%v", query, ok)
	}
	rawURL, ok := ParseURLFromText("open browser https://example.com/test.")
	if !ok || rawURL != "https://example.com/test" {
		t.Fatalf("ParseURLFromText got %q ok=%v", rawURL, ok)
	}
}

func TestForcedChatModelNameCompositeWorkflowNotForced(t *testing.T) {
	t.Parallel()
	in := `Open konsole, cd to Projects/. run mkdir and create new directory called "MyFirstTele". Inside that directory create a python app that is a clock that tells jokes on the hour.`
	if got := ForcedChatModelName(in); got != "" {
		t.Fatalf("ForcedChatModelName should not force single tool for composite workflow, got %q", got)
	}
}

func TestCompositeFilesystemWorkflowDetectsTypedFileCreate(t *testing.T) {
	t.Parallel()
	in := `Create a directory in Downloads called PeanutButterJellyTime. Inside that folder create an svg file of a flower.`
	if !IsCompositeFilesystemWorkflow(in) {
		t.Fatalf("expected composite filesystem workflow")
	}
	if got := ForcedChatModelName(in); got != "" {
		t.Fatalf("ForcedChatModelName should not force single mkdir for composite SVG workflow, got %q", got)
	}
}

func TestForcedChatModelNameSimpleMkdirStillForced(t *testing.T) {
	t.Parallel()
	in := `create a directory called "MyFirstTele"`
	if got := ForcedChatModelName(in); got != ChatModelName("fs.mkdir") {
		t.Fatalf("ForcedChatModelName should force fs.mkdir for simple mkdir, got %q", got)
	}
}
