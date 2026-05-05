package gateway

import (
	"strings"
	"testing"
)

func TestParseCombinedMkdirAndWrite(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		user    string
		wantDir string
		wantFn  string
		wantCt  string
		ok      bool
	}{
		{
			name:    "directory called plus labeled file",
			user:    `Create a directory called scratch/foo and a file labeled "x.txt" and the words "hi"`,
			wantDir: "scratch/foo",
			wantFn:  "x.txt",
			wantCt:  "hi",
			ok:      true,
		},
		{
			name:    "inside path plus labeled file",
			user:    `create a file labeled "test.txt" inside scratch/not_another_test/  and inside said file the words "This is a test file"`,
			wantDir: "scratch/not_another_test",
			wantFn:  "test.txt",
			wantCt:  "This is a test file",
			ok:      true,
		},
		{
			name: "missing directory fragment",
			user: `create a file labeled "a.txt" and the words "b"`,
			ok:   false,
		},
		{
			name:    "directory call with markdown recipe intent",
			user:    `Create a directory call Test1. Inside the Test1 directory, create a markdown file with your favorite recipe in it.`,
			wantDir: "Test1",
			wantFn:  "recipe.md",
			wantCt:  defaultRecipeMarkdown,
			ok:      true,
		},
		{
			name:    "directory called with trailing slash and punctuation",
			user:    `Create a directory called scratch/Test2/. Inside the Test2 directory, create a markdown file with your favorite recipe in it.`,
			wantDir: "scratch/Test2",
			wantFn:  "recipe.md",
			wantCt:  defaultRecipeMarkdown,
			ok:      true,
		},
		{
			name:    "directory call with directory phrase",
			user:    `Create a directory call Test3. Inside the directory Test3, create a file labeled "recipe.md" and the words "hi"`,
			wantDir: "Test3",
			wantFn:  "recipe.md",
			wantCt:  "hi",
			ok:      true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			dir, fn, ct, ok := ParseCombinedMkdirAndWrite(tc.user)
			if ok != tc.ok {
				t.Fatalf("ok = %v, want %v", ok, tc.ok)
			}
			if !tc.ok {
				return
			}
			if dir != tc.wantDir || fn != tc.wantFn || ct != tc.wantCt {
				t.Fatalf("got dir=%q file=%q content=%q; want dir=%q file=%q content=%q", dir, fn, ct, tc.wantDir, tc.wantFn, tc.wantCt)
			}
		})
	}
}

func TestParseReadPath(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		user string
		want string
		ok   bool
	}{
		{name: "cat path", user: "cat docs/USER_MANUAL.md", want: "docs/USER_MANUAL.md", ok: true},
		{name: "read file path", user: `read file "README.md"`, want: "README.md", ok: true},
		{name: "no path", user: "read file", ok: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, ok := ParseReadPath(tc.user)
			if ok != tc.ok {
				t.Fatalf("ok = %v, want %v", ok, tc.ok)
			}
			if !tc.ok {
				return
			}
			if got != tc.want {
				t.Fatalf("path = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestParseDirectoryCalledAndMkdirPath(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		fn   func(string) (string, bool)
		user string
		want string
		ok   bool
	}{
		{name: "directory called", fn: ParseDirectoryCalled, user: `create a directory called scratch/foo`, want: "scratch/foo", ok: true},
		{name: "folder named", fn: ParseDirectoryCalled, user: `make a folder named notes`, want: "notes", ok: true},
		{name: "mkdir shell path", fn: ParseMkdirShellPath, user: `mkdir -p scratch/bar`, want: "scratch/bar", ok: true},
		{name: "directory labeled only is typed elsewhere", fn: ParseDirectoryCalled, user: `create a directory labeled notes`, ok: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, ok := tc.fn(tc.user)
			if ok != tc.ok {
				t.Fatalf("ok = %v, want %v", ok, tc.ok)
			}
			if ok && got != tc.want {
				t.Fatalf("path = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestParseListPath(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		user string
		want string
		ok   bool
	}{
		{name: "ls explicit", user: "ls scratch", want: "scratch", ok: true},
		{name: "list files in path", user: "list files in docs", want: "docs", ok: true},
		{name: "list directory default", user: "list directory", want: ".", ok: true},
		{name: "no list intent", user: "read file README.md", ok: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, ok := ParseListPath(tc.user)
			if ok != tc.ok {
				t.Fatalf("ok = %v, want %v", ok, tc.ok)
			}
			if !tc.ok {
				return
			}
			if got != tc.want {
				t.Fatalf("path = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestFallbackParserRejectsUnsafePaths(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		fn   func(string) (string, bool)
		user string
	}{
		{name: "directory traversal", fn: ParseDirectoryCalled, user: `create a directory called ../escape`},
		{name: "absolute path", fn: ParseMkdirShellPath, user: `mkdir /tmp/forge-escape`},
		{name: "shell metacharacter", fn: ParseReadPath, user: `cat docs/README.md;rm -rf .`},
		{name: "list shell metacharacter", fn: ParseListPath, user: `ls docs && pwd`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got, ok := tc.fn(tc.user); ok {
				t.Fatalf("expected unsafe path rejection, got %q", got)
			}
		})
	}
}

func TestParseShellCommand(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		user string
		want string
		ok   bool
	}{
		{name: "run prefix", user: "run go test ./...", want: "go test ./...", ok: true},
		{name: "execute prefix", user: "execute npm run build", want: "npm run build", ok: true},
		{name: "no command", user: "run", ok: false},
		{name: "non command", user: "show me git status", ok: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, ok := ParseShellCommand(tc.user)
			if ok != tc.ok {
				t.Fatalf("ok = %v, want %v", ok, tc.ok)
			}
			if !tc.ok {
				return
			}
			if got != tc.want {
				t.Fatalf("cmd = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestParseFallbackIntentTypedModel(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name             string
		user             string
		wantType         FallbackIntentType
		wantTarget       string
		wantCommand      string
		requiresApproval bool
		hasContent       bool
	}{
		{name: "mkdir", user: "mkdir scratch", wantType: FallbackIntentMkdir, wantTarget: "scratch"},
		{name: "read", user: "read file README.md", wantType: FallbackIntentReadFile, wantTarget: "README.md"},
		{name: "list", user: "list files in docs", wantType: FallbackIntentListDirectory, wantTarget: "docs"},
		{name: "labeled directory", user: `create a directory labled "scratch/labeled"`, wantType: FallbackIntentMkdir, wantTarget: "scratch/labeled"},
		{name: "write", user: `Create a directory called scratch/foo and a file labeled "x.txt" and the words "hi"`, wantType: FallbackIntentWriteFile, wantTarget: "scratch/foo/x.txt", requiresApproval: true, hasContent: true},
		{name: "command", user: "run go test ./...", wantType: FallbackIntentRunCommand, wantCommand: "go test ./...", requiresApproval: true},
		{name: "none", user: "what is the best sandwich", wantType: FallbackIntentUnknown},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := ParseFallbackIntent(tc.user)
			if got.Type != tc.wantType {
				t.Fatalf("type = %q, want %q", got.Type, tc.wantType)
			}
			if tc.wantTarget != "" && got.TargetPath != tc.wantTarget {
				t.Fatalf("target = %q, want %q", got.TargetPath, tc.wantTarget)
			}
			if tc.wantCommand != "" && got.Command != tc.wantCommand {
				t.Fatalf("command = %q, want %q", got.Command, tc.wantCommand)
			}
			if got.RequiresApproval != tc.requiresApproval {
				t.Fatalf("requiresApproval = %v, want %v", got.RequiresApproval, tc.requiresApproval)
			}
			if tc.hasContent && got.Content == "" {
				t.Fatalf("expected content")
			}
			if got.Type == FallbackIntentRunCommand && len(got.Warnings) == 0 {
				t.Fatalf("expected command warning")
			}
		})
	}
}

func TestParsePythonBannerScriptIntent(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		user     string
		wantPath string
		mustHave []string
		ok       bool
	}{
		{
			name:     "inside directory with quoted banner text",
			user:     `Now, in the ForgeTestFile directory, create a simple scrolling banner python script that says "FORGE LIVES!" in vegas lights font.`,
			wantPath: "ForgeTestFile/banner.py",
			mustHave: []string{`TEXT = "FORGE LIVES!"`, `PRIMARY_FONT = "Vegas Lights"`, `def pulse()`},
			ok:       true,
		},
		{
			name:     "explicit py filename",
			user:     `Inside scratch/Test2 create a python banner script named neon_scroll.py that says "HELLO"`,
			wantPath: "scratch/Test2/neon_scroll.py",
			mustHave: []string{`TEXT = "HELLO"`},
			ok:       true,
		},
		{
			name:     "program called with words phrasing",
			user:     `Create a directory labled Auto_Banner. Inside that directory create a python program called hello_world.py. I want it to be a scrolling flashing banner with the words "HELLO WORLD".`,
			wantPath: "Auto_Banner/hello_world.py",
			mustHave: []string{`TEXT = "HELLO WORLD"`},
			ok:       true,
		},
		{
			name:     "create directory then inside directory",
			user:     `Create scratch/Python directory. Inside the directory create a simple scrolling banner python script that says "FORGE LIVES!" in vegas lights font.`,
			wantPath: "scratch/Python/banner.py",
			mustHave: []string{`TEXT = "FORGE LIVES!"`, `PRIMARY_FONT = "Vegas Lights"`},
			ok:       true,
		},
		{
			name:     "directory labled typo no script keyword purple blue pulse",
			user:     `I would like you to create a directory labled "test_project". Inside that directory, Create a scrolling banner in python that says "Te Queiro Mucho Mi Riena" in purple and blue pulsing letters.`,
			wantPath: "test_project/banner.py",
			mustHave: []string{`TEXT = "Te Queiro Mucho Mi Riena"`, `COLOR_A = "#8a2be2"`, `COLOR_B = "#1e90ff"`},
			ok:       true,
		},
		{
			name: "non script request",
			user: "create a markdown file with notes",
			ok:   false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			path, content, ok := ParsePythonBannerScriptIntent(tc.user)
			if ok != tc.ok {
				t.Fatalf("ok = %v, want %v", ok, tc.ok)
			}
			if !tc.ok {
				return
			}
			if path != tc.wantPath {
				t.Fatalf("path = %q, want %q", path, tc.wantPath)
			}
			for _, needle := range tc.mustHave {
				if !strings.Contains(content, needle) {
					t.Fatalf("content missing %q", needle)
				}
			}
		})
	}
}

func TestGeneratedTemplateIntentIsSafeAndDeterministic(t *testing.T) {
	t.Parallel()
	user := `Create a folder in the Downloads directory labled Python_Scripts/. Inside the folder create a python script that will make anything I download get sorted into a folder in the downloads folder.`
	intentA := ParseFallbackIntent(user)
	intentB := ParseFallbackIntent(user)
	if intentA.Type != FallbackIntentGenerateTemplate {
		t.Fatalf("type = %q", intentA.Type)
	}
	if intentA.TargetPath != "~/Downloads/Python_Scripts/sort_downloads.py" {
		t.Fatalf("path = %q", intentA.TargetPath)
	}
	if !intentA.RequiresApproval {
		t.Fatalf("template write intent should require approval")
	}
	if intentA.Content == "" || intentA.Content != intentB.Content {
		t.Fatalf("template content should be deterministic and non-empty")
	}
	if strings.ContainsAny(intentA.TargetPath, "\n\r;&|<>`$") || strings.Contains(intentA.TargetPath, "..") {
		t.Fatalf("unsafe generated template path %q", intentA.TargetPath)
	}
}

func TestParseDownloadSorterScriptIntent(t *testing.T) {
	t.Parallel()
	path, content, ok := ParseDownloadSorterScriptIntent(`Create a folder in the Downloads directory labled Python_Scripts/. Inside the folder create a python script that will make anything I download get sorted into a folder in the downloads folder.`)
	if !ok {
		t.Fatalf("expected Downloads sorter script intent")
	}
	if path != "~/Downloads/Python_Scripts/sort_downloads.py" {
		t.Fatalf("path = %q", path)
	}
	for _, needle := range []string{
		`DOWNLOADS = Path.home() / "Downloads"`,
		`SCRIPT_DIR = DOWNLOADS / "Python_Scripts"`,
		`def sort_once()`,
		`def watch(interval_seconds: float)`,
	} {
		if !strings.Contains(content, needle) {
			t.Fatalf("content missing %q", needle)
		}
	}
}

func TestParseDownloadsDirectoryCalled(t *testing.T) {
	t.Parallel()
	path, ok := ParseDownloadsDirectoryCalled(`Create a directory in Downloads called RandomSVGs.`)
	if !ok {
		t.Fatalf("expected Downloads directory intent")
	}
	if path != "~/Downloads/RandomSVGs" {
		t.Fatalf("path = %q", path)
	}
}

func TestParseSVGAssetWriteIntents(t *testing.T) {
	t.Parallel()
	writes, ok := ParseSVGAssetWriteIntents(`Create a directory in Downloads called RandomSVGs. Inside that folder create an svg file of a turtle and then one of stitch.`)
	if !ok {
		t.Fatalf("expected SVG write intents")
	}
	if len(writes) != 2 {
		t.Fatalf("write count = %d, writes=%#v", len(writes), writes)
	}
	want := []string{"~/Downloads/RandomSVGs/turtle.svg", "~/Downloads/RandomSVGs/stitch.svg"}
	for i, path := range want {
		if writes[i].Path != path {
			t.Fatalf("write[%d].Path = %q, want %q", i, writes[i].Path, path)
		}
		if !strings.Contains(writes[i].Contents, "<svg") {
			t.Fatalf("write[%d] content should be SVG", i)
		}
	}
}

func TestParseVideoGameJournalWebpageIntent(t *testing.T) {
	t.Parallel()
	path, content, ok := ParseVideoGameJournalWebpageIntent(
		`In the same directory, create a test webpage. I would like it to look like it belongs to a video game journal site.`,
		`~/Downloads/PeanutButterJellyTime/flower.svg`,
	)
	if !ok {
		t.Fatalf("expected webpage intent")
	}
	if path != "~/Downloads/PeanutButterJellyTime/test-webpage.html" {
		t.Fatalf("path=%q", path)
	}
	for _, want := range []string{"<!doctype html>", "Checkpoint Journal", "video-game journal"} {
		if !strings.Contains(content, want) {
			t.Fatalf("expected content to contain %q", want)
		}
	}
}
