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
			mustHave: []string{`TEXT = "FORGE LIVES!"`, `PRIMARY_FONT = "Vegas Lights"`},
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
			name:     "create directory then inside directory",
			user:     `Create scratch/Python directory. Inside the directory create a simple scrolling banner python script that says "FORGE LIVES!" in vegas lights font.`,
			wantPath: "scratch/Python/banner.py",
			mustHave: []string{`TEXT = "FORGE LIVES!"`, `PRIMARY_FONT = "Vegas Lights"`},
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
