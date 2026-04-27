package gateway

import (
	"path/filepath"
	"regexp"
	"strings"
)

// fallback_intent_parser.go contains deterministic natural-language fallback
// parsing for simple gateway intents when a model response omits structured
// tool calls. The parser proposes typed intents only. Gateway policy,
// capability checks, approvals, and audited execution remain authoritative.

type FallbackIntentType string

const (
	FallbackIntentUnknown          FallbackIntentType = "unknown"
	FallbackIntentMkdir            FallbackIntentType = "mkdir"
	FallbackIntentWriteFile        FallbackIntentType = "write_file"
	FallbackIntentReadFile         FallbackIntentType = "read_file"
	FallbackIntentListDirectory    FallbackIntentType = "list_directory"
	FallbackIntentRunCommand       FallbackIntentType = "run_command"
	FallbackIntentGenerateTemplate FallbackIntentType = "generate_template"
)

type FallbackIntent struct {
	Type             FallbackIntentType
	TargetPath       string
	Directory        string
	FileName         string
	Content          string
	Command          string
	Confidence       float64
	RequiresApproval bool
	Warnings         []string
	Source           string
}

var (
	reDirectoryCalled  = regexp.MustCompile(`(?i)(?:directory|folder)\s+(?:called|call|named)\s+['"]?([^\s'"]+(?:/[^\s'"]*)?)`)
	reDirectoryLabeled = regexp.MustCompile(`(?i)(?:directory|folder)\s+(?:labeled|labelled|labled|labeld)\s+['"]?([^\s'"]+(?:/[^\s'"]*)?)`)
	reDownloadsDirName = regexp.MustCompile(`(?i)(?:directory|folder)\s+in\s+(?:the\s+)?downloads(?:\s+(?:directory|folder))?\s+(?:called|call|named|labeled|labelled|labled|labeld)\s+['"]?([^\s'"]+(?:/[^\s'"]*)?)`)
	reInsidePath       = regexp.MustCompile(`(?i)\binside\s+(?:the\s+)?['"]?([^\s'"]+(?:/[^\s'"]*)?)['"]?(?:\s+directory)?`)
	reCreateDirPath    = regexp.MustCompile(`(?i)\b(?:create|make)\s+(?:an?\s+)?([a-z0-9_.\-\/~]+)\s+directory\b`)
	reFileLabeled      = regexp.MustCompile(`(?i)(?:a\s+)?file\s+labeled\s+['"]([^'"]+)['"]`)
	reTheWords         = regexp.MustCompile(`(?i)the words\s+['"]([^'"]+)['"]`)
	reMkdirOnly        = regexp.MustCompile(`(?i)\bmkdir(?:\s+-p)?\s+([^\s#]+)`)
	reCatPath          = regexp.MustCompile(`(?i)^\s*cat\s+([^\s#]+)`)
	reReadFilePath     = regexp.MustCompile(`(?i)\bread(?:\s+the)?\s+file\s+['"]?([^\s'"]+(?:/[^\s'"]*)?)`)
	reLsPath           = regexp.MustCompile(`(?i)^\s*ls(?:\s+-[^\s]+)*\s+([^\s#]+)`)
	reListDirPath      = regexp.MustCompile(`(?i)\blist(?:\s+the)?\s+(?:directory|files)(?:\s+(?:in|at|under))?\s+['"]?([^\s'"]+(?:/[^\s'"]*)?)`)
	reInDirectoryPath  = regexp.MustCompile(`(?i)\bin\s+(?:the\s+)?['"]?([a-z0-9_.\-\/~]+)['"]?\s+directory\b`)
	rePyFilePath       = regexp.MustCompile(`(?i)\b([a-z0-9_\-./~]+\.py)\b`)
	reSaysQuoted       = regexp.MustCompile(`(?i)\bsays?\s+['"]([^'"]+)['"]`)
	reSVGObject        = regexp.MustCompile(`(?i)(?:svg\s+file\s+of|svg\s+of|one\s+of)\s+(?:a\s+|an\s+)?([a-z][a-z0-9_-]*)`)
)

type fallbackPathOptions struct {
	AllowAbsolute bool
	AllowTilde    bool
	FileNameOnly  bool
}

type DeterministicWrite struct {
	Path     string
	Contents string
}

func normalizeFallbackPath(raw string, opts fallbackPathOptions) (string, bool) {
	p := strings.TrimSpace(raw)
	p = strings.Trim(p, `"'`)
	p = strings.TrimRight(p, ".,;:!?")
	p = strings.ReplaceAll(p, "\\", "/")
	p = filepath.ToSlash(p)
	p = strings.TrimSpace(p)
	if p == "" || containsUnsafePathMeta(p) {
		return "", false
	}
	if opts.FileNameOnly && strings.Contains(p, "/") {
		return "", false
	}
	if strings.HasPrefix(p, "~") {
		if !opts.AllowTilde {
			return "", false
		}
		if p != "~" && !strings.HasPrefix(p, "~/") {
			return "", false
		}
		return cleanTildePath(p)
	}
	if !opts.FileNameOnly {
		if aliased := normalizeOperatorPathAlias(p); aliased != p {
			return normalizeFallbackPath(aliased, fallbackPathOptions{AllowTilde: true, FileNameOnly: opts.FileNameOnly})
		}
	}
	if filepath.IsAbs(p) || strings.HasPrefix(p, "/") {
		if !opts.AllowAbsolute {
			return "", false
		}
		clean := filepath.ToSlash(filepath.Clean(p))
		if clean == "." || hasTraversalSegment(clean) {
			return "", false
		}
		return clean, true
	}
	clean := filepath.ToSlash(filepath.Clean(p))
	clean = strings.TrimPrefix(clean, "./")
	if clean == "." || clean == "" || hasTraversalSegment(clean) {
		return "", false
	}
	return clean, true
}

func normalizeOperatorPathAlias(p string) string {
	slash := filepath.ToSlash(strings.TrimSpace(p))
	trimmed := strings.Trim(slash, "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) == 0 {
		return p
	}
	switch strings.ToLower(parts[0]) {
	case "downloads":
		if len(parts) == 1 {
			return "~/Downloads"
		}
		return filepath.ToSlash(filepath.Join(append([]string{"~", "Downloads"}, parts[1:]...)...))
	default:
		return p
	}
}

func containsUnsafePathMeta(p string) bool {
	return strings.ContainsAny(p, "\x00\r\n;&|<>`$")
}

func hasTraversalSegment(p string) bool {
	for _, part := range strings.Split(filepath.ToSlash(p), "/") {
		if part == ".." {
			return true
		}
	}
	return false
}

func cleanTildePath(p string) (string, bool) {
	if p == "~" {
		return "", false
	}
	rest := strings.TrimPrefix(p, "~/")
	rest = strings.TrimSpace(rest)
	rest = strings.Trim(rest, `"'`)
	rest = strings.TrimRight(rest, ".,;:!?")
	rest = strings.ReplaceAll(rest, "\\", "/")
	rest = filepath.ToSlash(rest)
	if rest == "" || containsUnsafePathMeta(rest) {
		return "", false
	}
	clean := filepath.ToSlash(filepath.Clean(rest))
	clean = strings.TrimPrefix(clean, "./")
	if clean == "." || clean == "" || hasTraversalSegment(clean) {
		return "", false
	}
	return "~/" + clean, true
}

func unknownFallbackIntent() FallbackIntent {
	return FallbackIntent{
		Type:       FallbackIntentUnknown,
		Confidence: 0,
		Source:     "fallback_intent_parser",
	}
}

// ParseFallbackIntent returns a typed proposal for the gateway fallback path.
// It never executes commands or writes files; callers must route proposed work
// through gateway policy, capability checks, approvals, and audit.
func ParseFallbackIntent(user string) FallbackIntent {
	if dir, fileName, content, ok := ParseCombinedMkdirAndWrite(user); ok {
		intentType := FallbackIntentWriteFile
		source := "fallback_intent_parser"
		if content == defaultRecipeMarkdown {
			intentType = FallbackIntentGenerateTemplate
			source = "deterministic_template"
		}
		return FallbackIntent{
			Type:             intentType,
			TargetPath:       filepath.ToSlash(filepath.Join(dir, fileName)),
			Directory:        dir,
			FileName:         fileName,
			Content:          content,
			Confidence:       0.78,
			RequiresApproval: true,
			Source:           source,
		}
	}
	if path, content, ok := ParsePythonBannerScriptIntent(user); ok {
		return generatedTemplateIntent(path, content, 0.80)
	}
	if path, content, ok := ParseDownloadSorterScriptIntent(user); ok {
		return generatedTemplateIntent(path, content, 0.82)
	}
	if writes, ok := ParseSVGAssetWriteIntents(user); ok && len(writes) > 0 {
		first := writes[0]
		return generatedTemplateIntent(first.Path, first.Contents, 0.80)
	}
	if path, ok := ParseMkdirShellPath(user); ok {
		return FallbackIntent{Type: FallbackIntentMkdir, TargetPath: path, Directory: path, Confidence: 0.86, Source: "fallback_intent_parser"}
	}
	if path, ok := ParseDirectoryCalled(user); ok {
		return FallbackIntent{Type: FallbackIntentMkdir, TargetPath: path, Directory: path, Confidence: 0.72, Source: "fallback_intent_parser"}
	}
	if path, ok := parseDirectoryLabeled(user); ok {
		return FallbackIntent{Type: FallbackIntentMkdir, TargetPath: path, Directory: path, Confidence: 0.72, Source: "fallback_intent_parser"}
	}
	if path, ok := ParseReadPath(user); ok {
		return FallbackIntent{Type: FallbackIntentReadFile, TargetPath: path, Confidence: 0.84, Source: "fallback_intent_parser"}
	}
	if path, ok := ParseListPath(user); ok {
		return FallbackIntent{Type: FallbackIntentListDirectory, TargetPath: path, Directory: path, Confidence: 0.82, Source: "fallback_intent_parser"}
	}
	if cmd, ok := ParseShellCommand(user); ok {
		return FallbackIntent{
			Type:             FallbackIntentRunCommand,
			Command:          cmd,
			Confidence:       0.74,
			RequiresApproval: true,
			Warnings: []string{
				"command execution intent; gateway approval and capability policy required",
			},
			Source: "fallback_intent_parser",
		}
	}
	return unknownFallbackIntent()
}

func generatedTemplateIntent(path, content string, confidence float64) FallbackIntent {
	dir := filepath.ToSlash(filepath.Dir(path))
	fileName := filepath.Base(path)
	return FallbackIntent{
		Type:             FallbackIntentGenerateTemplate,
		TargetPath:       path,
		Directory:        dir,
		FileName:         fileName,
		Content:          content,
		Confidence:       confidence,
		RequiresApproval: true,
		Source:           "deterministic_template",
	}
}

// ParseCombinedMkdirAndWrite extracts mkdir + labeled file + quoted content from a single
// natural-language request (used when the model omits tool_calls).
//
// Supported shapes include:
//   - "directory called foo ... file labeled "a" ... the words "b""
//   - "file labeled "a" inside foo/bar/ ... the words "b"" (first "inside <path>" wins over later "inside said")
func ParseCombinedMkdirAndWrite(user string) (dirRel, fileName, content string, ok bool) {
	user = strings.TrimSpace(user)
	if user == "" {
		return "", "", "", false
	}
	fm := reFileLabeled.FindStringSubmatch(user)
	wm := reTheWords.FindStringSubmatch(user)
	if len(fm) >= 2 && len(wm) >= 2 {
		fileName = strings.TrimSpace(fm[1])
		content = wm[1]
	} else if looksLikeRecipeMarkdownIntent(user) {
		fileName = "recipe.md"
		content = defaultRecipeMarkdown
	} else {
		return "", "", "", false
	}
	var safe bool
	fileName, safe = normalizeFallbackPath(fileName, fallbackPathOptions{FileNameOnly: true})
	if !safe {
		return "", "", "", false
	}

	var dirRaw string
	if dm := reDirectoryCalled.FindStringSubmatch(user); len(dm) >= 2 {
		dirRaw = strings.TrimSpace(dm[1])
	} else if im := reInsidePath.FindStringSubmatch(user); len(im) >= 2 {
		dirRaw = strings.TrimSpace(im[1])
	} else {
		return "", "", "", false
	}
	dirRel, safe = normalizeFallbackPath(dirRaw, fallbackPathOptions{AllowTilde: true})
	if !safe {
		return "", "", "", false
	}
	return dirRel, fileName, content, true
}

func looksLikeRecipeMarkdownIntent(user string) bool {
	s := strings.ToLower(strings.TrimSpace(user))
	if s == "" {
		return false
	}
	hasFileIntent := strings.Contains(s, "markdown file") || strings.Contains(s, "md file") || strings.Contains(s, ".md")
	hasRecipeIntent := strings.Contains(s, "recipe")
	hasCreateIntent := strings.Contains(s, "create") || strings.Contains(s, "write")
	return hasFileIntent && hasRecipeIntent && hasCreateIntent
}

// ParseDirectoryCalled extracts a path after "directory called|named".
func ParseDirectoryCalled(user string) (dirRel string, ok bool) {
	if dir, ok := ParseDownloadsDirectoryCalled(user); ok {
		return dir, true
	}
	dm := reDirectoryCalled.FindStringSubmatch(strings.TrimSpace(user))
	if len(dm) < 2 {
		return "", false
	}
	return normalizeFallbackPath(dm[1], fallbackPathOptions{AllowTilde: true})
}

func parseDirectoryLabeled(user string) (dirRel string, ok bool) {
	if dir, ok := ParseDownloadsDirectoryCalled(user); ok {
		return dir, true
	}
	dm := reDirectoryLabeled.FindStringSubmatch(strings.TrimSpace(user))
	if len(dm) < 2 {
		return "", false
	}
	return normalizeFallbackPath(dm[1], fallbackPathOptions{AllowTilde: true})
}

func ParseDownloadsDirectoryCalled(user string) (dirRel string, ok bool) {
	dm := reDownloadsDirName.FindStringSubmatch(strings.TrimSpace(user))
	if len(dm) < 2 {
		return "", false
	}
	name, safe := normalizeFallbackPath(dm[1], fallbackPathOptions{})
	if !safe {
		return "", false
	}
	return filepath.ToSlash(filepath.Join("~", "Downloads", name)), true
}

// ParseMkdirShellPath extracts path from a mkdir shell-style fragment.
func ParseMkdirShellPath(user string) (path string, ok bool) {
	s := strings.TrimSpace(user)
	if containsUnsafePathMeta(s) {
		return "", false
	}
	m := reMkdirOnly.FindStringSubmatch(s)
	if len(m) < 2 {
		return "", false
	}
	return normalizeFallbackPath(m[1], fallbackPathOptions{AllowTilde: true})
}

// ParseReadPath extracts a file path from "cat <path>" or "read file <path>".
func ParseReadPath(user string) (path string, ok bool) {
	s := strings.TrimSpace(user)
	if s == "" {
		return "", false
	}
	if containsUnsafePathMeta(s) {
		return "", false
	}
	if m := reCatPath.FindStringSubmatch(s); len(m) >= 2 {
		if p, safe := normalizeFallbackPath(m[1], fallbackPathOptions{AllowTilde: true}); safe {
			return p, true
		}
	}
	if m := reReadFilePath.FindStringSubmatch(s); len(m) >= 2 {
		if p, safe := normalizeFallbackPath(m[1], fallbackPathOptions{AllowTilde: true}); safe {
			return p, true
		}
	}
	return "", false
}

// ParseListPath extracts an optional directory path from "ls <path>" or "list files in <path>".
// Returns "." when listing intent exists but no explicit path is found.
func ParseListPath(user string) (path string, ok bool) {
	s := strings.TrimSpace(user)
	if s == "" {
		return "", false
	}
	if containsUnsafePathMeta(s) {
		return "", false
	}
	if m := reLsPath.FindStringSubmatch(s); len(m) >= 2 {
		if p, safe := normalizeFallbackPath(m[1], fallbackPathOptions{AllowTilde: true}); safe {
			return p, true
		}
	}
	if m := reListDirPath.FindStringSubmatch(s); len(m) >= 2 {
		if p, safe := normalizeFallbackPath(m[1], fallbackPathOptions{AllowTilde: true}); safe {
			return p, true
		}
	}
	sl := strings.ToLower(s)
	if strings.HasPrefix(sl, "ls") || strings.Contains(sl, "list files") || strings.Contains(sl, "list directory") {
		return ".", true
	}
	return "", false
}

// ParseShellCommand extracts command text from "run ..." or "execute ...".
// Typed fallback command intents are always marked approval-required.
func ParseShellCommand(user string) (cmd string, ok bool) {
	s := strings.TrimSpace(user)
	if s == "" {
		return "", false
	}
	low := strings.ToLower(s)
	if strings.HasPrefix(low, "run ") {
		cmd = strings.TrimSpace(s[len("run "):])
		return cmd, cmd != ""
	}
	if strings.HasPrefix(low, "execute ") {
		cmd = strings.TrimSpace(s[len("execute "):])
		return cmd, cmd != ""
	}
	return "", false
}

// ParsePythonBannerScriptIntent parses requests like:
// "in <dir>, create a scrolling banner python script that says 'FORGE LIVES!'".
// Returns a proposed write path plus deterministic script contents.
func ParsePythonBannerScriptIntent(user string) (writePath string, contents string, ok bool) {
	raw := strings.TrimSpace(user)
	if raw == "" {
		return "", "", false
	}
	lower := strings.ToLower(raw)
	if !strings.Contains(lower, "python") || (!strings.Contains(lower, "banner") && !strings.Contains(lower, "scroll")) {
		return "", "", false
	}
	if !(strings.Contains(lower, "create") || strings.Contains(lower, "write") || strings.Contains(lower, "save")) {
		return "", "", false
	}

	dir := firstNonPlaceholderDir(raw)
	if dir == "" {
		return "", "", false
	}
	var safe bool
	dir, safe = normalizeFallbackPath(dir, fallbackPathOptions{AllowTilde: true})
	if !safe {
		return "", "", false
	}

	fileName := "banner.py"
	if m := rePyFilePath.FindStringSubmatch(raw); len(m) >= 2 {
		candidate := strings.TrimSpace(m[1])
		if normalized, ok := normalizeFallbackPath(candidate, fallbackPathOptions{FileNameOnly: true}); ok {
			fileName = normalized
		} else if normalized, ok := normalizeFallbackPath(filepath.Base(filepath.ToSlash(candidate)), fallbackPathOptions{FileNameOnly: true}); ok {
			// Keep compatibility for "named path/file.py" while stripping the
			// directory; target directory is parsed independently and validated.
			fileName = normalized
		}
	}
	if !strings.HasSuffix(strings.ToLower(fileName), ".py") {
		fileName += ".py"
	}
	fileName, safe = normalizeFallbackPath(fileName, fallbackPathOptions{FileNameOnly: true})
	if !safe {
		return "", "", false
	}

	bannerText := "FORGE LIVES!"
	if m := reSaysQuoted.FindStringSubmatch(raw); len(m) >= 2 {
		if quoted := strings.TrimSpace(m[1]); quoted != "" {
			bannerText = quoted
		}
	}
	colorA := "#ffd766"
	colorB := "#ff5722"
	if strings.Contains(lower, "purple") && strings.Contains(lower, "blue") {
		colorA = "#8a2be2"
		colorB = "#1e90ff"
	}

	writePath = filepath.ToSlash(filepath.Join(dir, fileName))
	return writePath, pythonBannerScriptTemplate(bannerText, colorA, colorB), true
}

func firstNonPlaceholderDir(raw string) string {
	matchers := []*regexp.Regexp{
		reCreateDirPath,
		reMkdirOnly,
		reInsidePath,
		reDirectoryCalled,
		reDirectoryLabeled,
		reInDirectoryPath,
	}
	for _, matcher := range matchers {
		if m := matcher.FindStringSubmatch(raw); len(m) >= 2 {
			candidate := strings.Trim(strings.TrimSpace(m[1]), `"'`)
			if !isPlaceholderDirToken(candidate) {
				return candidate
			}
		}
	}
	return ""
}

// ParseDownloadSorterScriptIntent parses requests to create a Python script under
// Downloads/Python_Scripts that sorts downloaded files into Downloads subfolders.
// The generated script is deterministic and standard-library only; it is not run.
func ParseDownloadSorterScriptIntent(user string) (writePath string, contents string, ok bool) {
	raw := strings.TrimSpace(user)
	if raw == "" {
		return "", "", false
	}
	lower := strings.ToLower(raw)
	if !strings.Contains(lower, "python") || !strings.Contains(lower, "download") {
		return "", "", false
	}
	if !(strings.Contains(lower, "sort") || strings.Contains(lower, "organize") || strings.Contains(lower, "organise")) {
		return "", "", false
	}
	if !(strings.Contains(lower, "script") || strings.Contains(lower, "file")) {
		return "", "", false
	}

	fileName := "sort_downloads.py"
	if m := rePyFilePath.FindStringSubmatch(raw); len(m) >= 2 {
		candidate := filepath.Base(filepath.ToSlash(strings.TrimSpace(m[1])))
		if normalized, safe := normalizeFallbackPath(candidate, fallbackPathOptions{FileNameOnly: true}); safe && strings.HasSuffix(strings.ToLower(normalized), ".py") {
			fileName = normalized
		}
	}

	writePath = filepath.ToSlash(filepath.Join("~", "Downloads", "Python_Scripts", fileName))
	return writePath, downloadSorterScriptTemplate(), true
}

func ParseSVGAssetWriteIntents(user string) ([]DeterministicWrite, bool) {
	raw := strings.TrimSpace(user)
	if raw == "" {
		return nil, false
	}
	lower := strings.ToLower(raw)
	if !strings.Contains(lower, "svg") || !(strings.Contains(lower, "create") || strings.Contains(lower, "write") || strings.Contains(lower, "make") || strings.Contains(lower, "save")) {
		return nil, false
	}
	dir, ok := ParseDownloadsDirectoryCalled(raw)
	if !ok {
		dir, ok = ParseDirectoryCalled(raw)
	}
	if !ok {
		dir, ok = parseDirectoryLabeled(raw)
	}
	if !ok || dir == "" {
		return nil, false
	}

	matches := reSVGObject.FindAllStringSubmatch(raw, -1)
	if len(matches) == 0 {
		return nil, false
	}
	seen := map[string]bool{}
	writes := make([]DeterministicWrite, 0, len(matches))
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		subject := strings.ToLower(strings.TrimSpace(match[1]))
		name, safe := normalizeFallbackPath(subject+".svg", fallbackPathOptions{FileNameOnly: true})
		if !safe || seen[name] {
			continue
		}
		seen[name] = true
		writes = append(writes, DeterministicWrite{
			Path:     filepath.ToSlash(filepath.Join(dir, name)),
			Contents: simpleSVGTemplate(subject),
		})
	}
	if len(writes) == 0 {
		return nil, false
	}
	return writes, true
}

// IsVideoGameJournalWebpageIntent reports whether the user is asking for a
// deterministic test webpage styled as a video-game journal page.
func IsVideoGameJournalWebpageIntent(user string) bool {
	s := strings.ToLower(strings.TrimSpace(user))
	if s == "" {
		return false
	}
	hasCreate := strings.Contains(s, "create") || strings.Contains(s, "write") || strings.Contains(s, "make") || strings.Contains(s, "save")
	hasPage := strings.Contains(s, "webpage") || strings.Contains(s, "web page") || strings.Contains(s, "html page") || strings.Contains(s, ".html")
	hasStyle := strings.Contains(s, "video game") || strings.Contains(s, "game journal") || strings.Contains(s, "journal site")
	return hasCreate && hasPage && hasStyle
}

// ParseVideoGameJournalWebpageIntent returns a deterministic HTML write target
// and content for a follow-up request such as "in the same directory, create a
// test webpage..." dirHint may be either a directory path or a prior file path.
func ParseVideoGameJournalWebpageIntent(user, dirHint string) (writePath string, contents string, ok bool) {
	if !IsVideoGameJournalWebpageIntent(user) {
		return "", "", false
	}
	dir := normalizeDirectoryHint(dirHint)
	if dir == "" {
		return "", "", false
	}
	writePath = filepath.ToSlash(filepath.Join(dir, "test-webpage.html"))
	return writePath, videoGameJournalHTML(), true
}

func normalizeDirectoryHint(path string) string {
	path, ok := normalizeFallbackPath(path, fallbackPathOptions{AllowAbsolute: true, AllowTilde: true})
	if !ok {
		return ""
	}
	if strings.HasSuffix(path, "/") {
		return strings.TrimRight(path, "/")
	}
	base := filepath.Base(path)
	if strings.Contains(base, ".") {
		dir := filepath.Dir(path)
		if dir == "." {
			return ""
		}
		return filepath.ToSlash(dir)
	}
	return strings.TrimRight(path, "/")
}

func isPlaceholderDirToken(v string) bool {
	s := strings.ToLower(strings.TrimSpace(v))
	switch s {
	case "", "a", "an", "directory", "folder", "dir", "the", "this", "that", "said":
		return true
	default:
		return false
	}
}
