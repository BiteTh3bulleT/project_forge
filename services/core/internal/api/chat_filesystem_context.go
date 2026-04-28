package api

import (
	"context"
	"path/filepath"
	"strings"

	"forge/projectforge/services/core/internal/chat"
)

func (s *Server) latestGatewayFilesystemDir(ctx context.Context, th *chat.ThreadDetail) string {
	if th == nil {
		return ""
	}
	if snap, ok := s.latestGatewayProbeSnapshot(ctx, th); ok {
		if dir := filesystemDirFromPath(snap.Path); dir != "" {
			return dir
		}
	}
	for i := len(th.Messages) - 1; i >= 0; i-- {
		msg := th.Messages[i]
		if !strings.EqualFold(strings.TrimSpace(msg.Role), "assistant") || msg.Metadata == nil {
			continue
		}
		activity, ok := msg.Metadata["toolGatewayActivity"].(map[string]any)
		if !ok || activity == nil {
			continue
		}
		if dir := filesystemDirFromPath(asString(activity["toolArgs"])); dir != "" {
			return dir
		}
		if args, ok := activity["toolArgs"].(map[string]any); ok {
			for _, key := range []string{"path", "writePath", "file"} {
				if dir := filesystemDirFromPath(asString(args[key])); dir != "" {
					return dir
				}
			}
		}
		if result, ok := activity["executionResult"].(map[string]any); ok {
			for _, key := range []string{"path", "writePath"} {
				if dir := filesystemDirFromPath(asString(result[key])); dir != "" {
					return dir
				}
			}
			if write, ok := result["write"].(map[string]any); ok {
				if dir := filesystemDirFromPath(asString(write["path"])); dir != "" {
					return dir
				}
			}
		}
	}
	return ""
}

func filesystemDirFromPath(path string) string {
	path = filepath.ToSlash(strings.TrimSpace(path))
	path = strings.TrimRight(path, ".,;:!?")
	if path == "" {
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

func isHTMLWritePath(path string) bool {
	ext := strings.ToLower(filepath.Ext(strings.TrimSpace(path)))
	return ext == ".html" || ext == ".htm"
}
