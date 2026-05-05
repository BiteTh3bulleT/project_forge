package gateway

import (
	"regexp"
	"strings"

	"forge/projectforge/services/core/internal/aios/hyperlane"
)

var (
	reMkdirShell  = regexp.MustCompile(`(?i)\bmkdir(?:\s+-p)?\s+([^\s#]+)`)
	reMakeFolder  = regexp.MustCompile(`(?i)(?:create|make)\s+(?:a\s+)?(?:directory|dir|folder)(?:\s+(?:at|in|under|for|:))?\s+['"]?([^'"\n]+?)['"]?(?:\s|$)`)
	reActionVerb  = regexp.MustCompile(`(?i)\b(create|write|save|edit|modify|update|delete|remove|rename|move|copy|run|execute|build|test|install|commit|checkout|stash|chmod|fetch|scan|open|start|stop|restart|read|list|show)\b`)
	reStatusProbe = regexp.MustCompile(`(?i)\b(how\s+are|is\s+everything|is\s+it\s+working|any\s+updates|any\s+progress|where\s+are\s+we|what\s+is\s+the\s+status|status\s+of|how\s+is\s+it\s+going|what\s+is\s+next|did\s+it\s+work|seemed\s+to\s+work|was\s+it\s+created|was\s+that\s+created|was\s+the\s+file\s+created|did\s+that\s+work)\b`)
	reURL         = regexp.MustCompile(`https?://[^\s]+`)
)

var smallTalkTurns = map[string]struct{}{
	"ok":                     {},
	"okay":                   {},
	"k":                      {},
	"cool":                   {},
	"nice":                   {},
	"great":                  {},
	"awesome":                {},
	"good job":               {},
	"well done":              {},
	"thanks":                 {},
	"thank you":              {},
	"thx":                    {},
	"bet":                    {},
	"ready":                  {},
	"ready?":                 {},
	"lets go":                {},
	"let's go":               {},
	"sounds good":            {},
	"bravo":                  {},
	"bravo lad":              {},
	"good work":              {},
	"excellent":              {},
	"solid":                  {},
	"perfect":                {},
	"continue":               {},
	"proceed":                {},
	"carry on":               {},
	"all good":               {},
	"looks good":             {},
	"that works":             {},
	"that ll hold":           {},
	"that'll hold":           {},
	"nice work":              {},
	"how are":                {},
	"how are we":             {},
	"how are we looking":     {},
	"how are we looking now": {},
	"how's it going":         {},
	"how is it going":        {},
	"what's up":              {},
	"what is up":             {},
	"what now":               {},
	"any updates":            {},
	"any progress":           {},
	"where are we":           {},
	"where are we at":        {},
	"good stuff":             {},
	"looks better":           {},
	"much better":            {},
	"great work":             {},
	"you got it":             {},
	"impressive":             {},
	"good":                   {},
	"yep":                    {},
	"yes":                    {},
	"no":                     {},
	"nah":                    {},
}

// ForcedChatModelName returns a forge_* function name when the user text clearly maps to one gateway tool.
func ForcedChatModelName(user string) string {
	if wantsCompositeFilesystemWorkflow(user) {
		return ""
	}
	intent := ParseHyperlaneIntent(user)
	switch intent.Route {
	case hyperlane.RouteGatewayFSMkdir,
		hyperlane.RouteGatewayFSList,
		hyperlane.RouteGatewayFSRead,
		hyperlane.RouteGatewayFSWrite,
		hyperlane.RouteGatewayProcRun,
		hyperlane.RouteGatewayWebSearch,
		hyperlane.RouteGatewayNetFetch,
		hyperlane.RouteGatewayDesktopOpen,
		hyperlane.RouteGatewayGitStatus:
		return ChatModelName(intent.Route)
	}
	return ""
}

// IsCompositeFilesystemWorkflow reports whether a user turn combines folder creation
// with a follow-on file write/create step. Chat dispatch uses this to avoid
// executing partial filesystem side effects before the write action is approved.
func IsCompositeFilesystemWorkflow(user string) bool {
	return wantsCompositeFilesystemWorkflow(user)
}

func wantsCompositeFilesystemWorkflow(user string) bool {
	s := strings.TrimSpace(strings.ToLower(user))
	if s == "" {
		return false
	}
	if !wantsFilesystemMkdir(user) {
		return false
	}
	hasFollowOnCreate := wantsWriteFile(user) ||
		strings.Contains(s, "python app") ||
		strings.Contains(s, "inside that directory") ||
		strings.Contains(s, "inside the directory") ||
		strings.Contains(s, "inside that folder") ||
		strings.Contains(s, "inside the folder")
	if !hasFollowOnCreate {
		return false
	}
	return strings.Contains(s, " and ") ||
		strings.Contains(s, " then ") ||
		strings.Contains(s, " inside ")
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
	if IsVideoGameJournalWebpageIntent(user) {
		return true
	}
	if (strings.Contains(s, "create") || strings.Contains(s, "write") || strings.Contains(s, "make") || strings.Contains(s, "save")) &&
		(strings.Contains(s, "webpage") || strings.Contains(s, "web page") || strings.Contains(s, "html page")) {
		return true
	}
	if (strings.Contains(s, "create") || strings.Contains(s, "write") || strings.Contains(s, "save")) &&
		(strings.Contains(s, " file") || strings.Contains(s, " script")) &&
		(strings.Contains(s, "svg") || strings.Contains(s, "json") || strings.Contains(s, "markdown") || strings.Contains(s, "text") ||
			strings.Contains(s, "html") || strings.Contains(s, "css") || strings.Contains(s, "javascript") || strings.Contains(s, "typescript") ||
			strings.Contains(s, "go file") || strings.Contains(s, ".go") || strings.Contains(s, "python")) {
		return true
	}
	if strings.Contains(s, "create a script") || strings.Contains(s, "create script") {
		return true
	}
	if strings.Contains(s, "create a python app") || strings.Contains(s, "create python app") {
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

func wantsTerminalOpen(user string) bool {
	s := strings.TrimSpace(strings.ToLower(user))
	if s == "" {
		return false
	}
	if !(strings.Contains(s, "terminal") || strings.Contains(s, "konsole")) {
		return false
	}
	if !(strings.Contains(s, "open") || strings.Contains(s, "launch") || strings.Contains(s, "start")) {
		return false
	}
	return true
}

func wantsWebSearch(user string) bool {
	s := strings.TrimSpace(strings.ToLower(user))
	if s == "" {
		return false
	}
	if strings.Contains(s, "search the web") || strings.Contains(s, "web search") || strings.Contains(s, "look up") || strings.Contains(s, "google ") {
		return true
	}
	if strings.Contains(s, "weather") && (strings.Contains(s, " today") || strings.Contains(s, " current") || strings.Contains(s, " forecast")) {
		return strings.Contains(s, " in ") || strings.Contains(s, " for ") || strings.Contains(s, " at ")
	}
	return strings.HasPrefix(s, "search ") && !strings.Contains(s, "/")
}

func wantsURLFetch(user string) bool {
	s := strings.TrimSpace(strings.ToLower(user))
	if s == "" || !reURL.MatchString(user) {
		return false
	}
	return strings.Contains(s, "fetch") || strings.Contains(s, "read") || strings.Contains(s, "inspect") || strings.Contains(s, "summarize")
}

func wantsBrowserOpen(user string) bool {
	s := strings.TrimSpace(strings.ToLower(user))
	if s == "" {
		return false
	}
	return reURL.MatchString(user) && (strings.Contains(s, "open browser") || strings.Contains(s, "open in browser") || strings.Contains(s, "browse to") || strings.Contains(s, "open url"))
}

func ParseWebSearchQuery(user string) (string, bool) {
	s := strings.TrimSpace(user)
	if s == "" || !wantsWebSearch(s) {
		return "", false
	}
	lower := strings.ToLower(s)
	prefixes := []string{"search the web for", "search web for", "web search for", "look up", "google", "search"}
	for _, prefix := range prefixes {
		if strings.HasPrefix(lower, prefix) {
			query := strings.TrimSpace(s[len(prefix):])
			query = strings.Trim(query, " :")
			if query != "" {
				return query, true
			}
		}
	}
	return s, true
}

func ParseURLFromText(user string) (string, bool) {
	raw := strings.TrimSpace(reURL.FindString(user))
	if raw == "" {
		return "", false
	}
	raw = strings.TrimRight(raw, ".,);]")
	return raw, true
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

// IsLikelyStatusProbeTurn identifies conversational status checks that should get direct operator feedback.
func IsLikelyStatusProbeTurn(user string) bool {
	s := normalizeIntentText(user)
	if s == "" {
		return false
	}
	if IsLikelySmallTalkTurn(user) {
		return true
	}
	if reStatusProbe.MatchString(s) {
		return true
	}
	if strings.HasPrefix(s, "how are") && (strings.Contains(s, "it") || strings.Contains(s, "we") || strings.Contains(s, "we looking")) {
		return true
	}
	if strings.Contains(s, "status") && (strings.Contains(s, "core") || strings.Contains(s, "health") || strings.Contains(s, "adapter")) {
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
	if _, _, ok := ParseDownloadSorterScriptIntent(user); ok {
		return true
	}
	if IsVideoGameJournalWebpageIntent(user) {
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
