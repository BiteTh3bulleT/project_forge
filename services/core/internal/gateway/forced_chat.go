package gateway

import (
	"regexp"
	"strings"
)

var (
	reMkdirShell = regexp.MustCompile(`(?i)\bmkdir(?:\s+-p)?\s+([^\s#]+)`)
	reMakeFolder = regexp.MustCompile(`(?i)(?:create|make)\s+(?:a\s+)?(?:directory|dir|folder)(?:\s+(?:at|in|under|for|:))?\s+['"]?([^'"\n]+?)['"]?(?:\s|$)`)
	reActionVerb = regexp.MustCompile(`(?i)\b(create|write|save|edit|modify|update|delete|remove|rename|move|copy|run|execute|build|test|install|commit|checkout|stash|chmod|fetch|scan|open|start|stop|restart|read|list|show)\b`)
)

var smallTalkTurns = map[string]struct{}{
	"ok":           {},
	"okay":         {},
	"k":            {},
	"cool":         {},
	"nice":         {},
	"great":        {},
	"awesome":      {},
	"good job":     {},
	"well done":    {},
	"thanks":       {},
	"thank you":    {},
	"thx":          {},
	"bet":          {},
	"ready":        {},
	"ready?":       {},
	"lets go":      {},
	"let's go":     {},
	"sounds good":  {},
	"bravo":        {},
	"bravo lad":    {},
	"good work":    {},
	"excellent":    {},
	"solid":        {},
	"perfect":      {},
	"continue":     {},
	"proceed":      {},
	"carry on":     {},
	"all good":     {},
	"looks good":   {},
	"that works":   {},
	"that ll hold": {},
	"that'll hold": {},
	"nice work":    {},
	"good stuff":   {},
	"looks better": {},
	"much better":  {},
	"great work":   {},
	"you got it":   {},
	"impressive":   {},
	"good":         {},
	"yep":          {},
	"yes":          {},
	"no":           {},
	"nah":          {},
}

// ForcedChatModelName returns a forge_* function name when the user text clearly maps to one gateway tool.
func ForcedChatModelName(user string) string {
	if wantsFilesystemMkdir(user) {
		return ChatModelName("fs.mkdir")
	}
	if wantsListDirectory(user) {
		return ChatModelName("fs.list")
	}
	if wantsReadFile(user) {
		return ChatModelName("fs.read")
	}
	if wantsWriteFile(user) {
		return ChatModelName("fs.write")
	}
	if wantsShellRun(user) {
		return ChatModelName("proc.run")
	}
	if wantsGitStatus(user) {
		return ChatModelName("git.status")
	}
	return ""
}

func wantsFilesystemMkdir(user string) bool {
	s := strings.TrimSpace(strings.ToLower(user))
	if s == "" {
		return false
	}
	if reMkdirShell.MatchString(user) || reMakeFolder.MatchString(user) {
		return true
	}
	if strings.Contains(s, "mkdir") {
		return true
	}
	if strings.Contains(s, "create a directory") || strings.Contains(s, "create directory") {
		return true
	}
	if strings.Contains(s, "make a folder") || strings.Contains(s, "make folder") {
		return true
	}
	return false
}

func wantsListDirectory(user string) bool {
	s := strings.TrimSpace(strings.ToLower(user))
	if s == "" {
		return false
	}
	if strings.HasPrefix(s, "ls ") || s == "ls" {
		return true
	}
	if strings.Contains(s, "list files") || strings.Contains(s, "list directory") || strings.Contains(s, "list the directory") || strings.Contains(s, "list the files") {
		return true
	}
	if strings.Contains(s, "what files") || strings.Contains(s, "what's in") || strings.Contains(s, "show files") {
		return true
	}
	return false
}

func wantsReadFile(user string) bool {
	s := strings.TrimSpace(strings.ToLower(user))
	if s == "" {
		return false
	}
	if strings.HasPrefix(s, "cat ") {
		return true
	}
	if strings.Contains(s, "read file") || strings.Contains(s, "read the file") || strings.Contains(s, "show me the file") {
		return true
	}
	if strings.Contains(s, "show me ") && (strings.Contains(s, ".go") || strings.Contains(s, ".ts") || strings.Contains(s, ".json") || strings.Contains(s, ".yaml") || strings.Contains(s, ".yml") || strings.Contains(s, ".md") || strings.Contains(s, ".txt") || strings.Contains(s, ".toml")) {
		return true
	}
	return false
}

func wantsWriteFile(user string) bool {
	s := strings.TrimSpace(strings.ToLower(user))
	if s == "" {
		return false
	}
	if strings.Contains(s, "write file") || strings.Contains(s, "write a file") || strings.Contains(s, "write to file") {
		return true
	}
	if strings.Contains(s, "save to ") || strings.Contains(s, "save this to ") {
		return true
	}
	if strings.Contains(s, "create a file") || strings.Contains(s, "create file") {
		return true
	}
	if strings.Contains(s, "create a script") || strings.Contains(s, "create script") {
		return true
	}
	if strings.Contains(s, "python script") || strings.Contains(s, ".py") {
		return strings.Contains(s, "create") || strings.Contains(s, "write") || strings.Contains(s, "save")
	}
	return false
}

func wantsShellRun(user string) bool {
	s := strings.TrimSpace(strings.ToLower(user))
	if s == "" {
		return false
	}
	if strings.HasPrefix(s, "run ") || strings.HasPrefix(s, "execute ") {
		return true
	}
	if strings.Contains(s, "run the command") || strings.Contains(s, "run this command") {
		return true
	}
	if strings.Contains(s, "go build") || strings.Contains(s, "go test") || strings.Contains(s, "make ") || strings.Contains(s, "npm ") {
		return true
	}
	return false
}

func wantsGitStatus(user string) bool {
	s := strings.TrimSpace(strings.ToLower(user))
	if s == "" {
		return false
	}
	if strings.HasPrefix(s, "git status") || s == "git status" {
		return true
	}
	if strings.Contains(s, "uncommitted changes") || strings.Contains(s, "modified files") || strings.Contains(s, "what files changed") || strings.Contains(s, "what changed") {
		return true
	}
	return false
}

// IsLikelySmallTalkTurn returns true when the message looks conversational and non-operational.
func IsLikelySmallTalkTurn(user string) bool {
	s := normalizeIntentText(user)
	if s == "" {
		return false
	}
	if _, ok := smallTalkTurns[s]; ok {
		return true
	}
	if strings.HasPrefix(s, "thanks ") || strings.HasPrefix(s, "thank you ") {
		return true
	}
	return false
}

// ShouldAttachChatTools gates whether the tool catalog should be attached for this user turn.
// It is intentionally conservative: casual acknowledgements do not get tool access.
func ShouldAttachChatTools(user string) bool {
	if IsLikelySmallTalkTurn(user) {
		return false
	}
	if ForcedChatModelName(user) != "" {
		return true
	}
	if _, _, _, ok := ParseCombinedMkdirAndWrite(user); ok {
		return true
	}
	if _, _, ok := ParsePythonBannerScriptIntent(user); ok {
		return true
	}
	if _, ok := ParseShellCommand(user); ok {
		return true
	}
	if _, ok := ParseReadPath(user); ok {
		return true
	}
	if _, ok := ParseListPath(user); ok {
		return true
	}
	s := normalizeIntentText(user)
	if strings.Contains(s, "/") {
		return true
	}
	return reActionVerb.MatchString(s)
}

func normalizeIntentText(user string) string {
	s := strings.TrimSpace(strings.ToLower(user))
	if s == "" {
		return ""
	}
	replacer := strings.NewReplacer(
		"!", " ",
		"?", " ",
		".", " ",
		",", " ",
		";", " ",
		":", " ",
		"\n", " ",
		"\t", " ",
		"  ", " ",
	)
	s = replacer.Replace(s)
	s = strings.Join(strings.Fields(s), " ")
	return s
}
