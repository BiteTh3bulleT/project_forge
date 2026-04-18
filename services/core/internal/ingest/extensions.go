package ingest

import (
	"path/filepath"
	"strings"
)

var defaultExtensions = map[string]struct{}{
	".md": {}, ".txt": {}, ".json": {},
	".ts": {}, ".tsx": {}, ".js": {}, ".jsx": {},
	".go": {}, ".py": {}, ".rs": {}, ".java": {},
	".yaml": {}, ".yml": {},
	".c": {}, ".h": {}, ".cpp": {}, ".hpp": {},
	".css": {}, ".html": {}, ".sql": {},
}

func IsSupportedPath(path string, allowed map[string]struct{}) bool {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == "" {
		return false
	}
	if len(allowed) == 0 {
		_, ok := defaultExtensions[ext]
		return ok
	}
	_, ok := allowed[ext]
	return ok
}

func ParseExtensionList(csv string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, p := range strings.Split(csv, ",") {
		e := strings.TrimSpace(strings.ToLower(p))
		if e == "" {
			continue
		}
		if !strings.HasPrefix(e, ".") {
			e = "." + e
		}
		out[e] = struct{}{}
	}
	return out
}

func DefaultExtensionsCSV() string {
	list := []string{
		".md", ".txt", ".json",
		".ts", ".tsx", ".js",
		".go", ".py", ".rs", ".java", ".yaml", ".yml",
	}
	return strings.Join(list, ",")
}
