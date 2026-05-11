package gateway

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"forge/projectforge/services/core/internal/aios/domain"
)

func capabilityHasEffect(capability domain.ToolCapability, effect domain.ToolEffect) bool {
	for _, item := range capability.Effect {
		if item == effect {
			return true
		}
	}
	return false
}

func capabilityOK(message string, data map[string]any) Result {
	return capabilityOKWithArtifacts(message, data, nil)
}

func capabilityOKWithArtifacts(message string, data map[string]any, artifacts []ResultArtifact) Result {
	if data == nil {
		data = map[string]any{}
	}
	return Result{Status: StatusOK, Message: message, Data: data, Artifacts: artifacts}
}

func capabilityInputSummary(input map[string]any) map[string]any {
	fieldCount := len(input)
	fields := []string{}
	truncatedFields := false
	truncatedFieldNames := false
	hasSensitiveFields := false
	if fieldCount > 0 {
		keys := make([]string, 0, fieldCount)
		for key := range input {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			if len(fields) >= maxCapabilityResultInputFields {
				truncatedFields = true
				break
			}
			name := strings.TrimSpace(key)
			if name == "" {
				name = "<empty>"
			}
			if len(name) > maxCapabilityResultInputFieldNameBytes {
				name = name[:maxCapabilityResultInputFieldNameBytes]
				truncatedFieldNames = true
			}
			if capabilityInputFieldLooksSensitive(key) {
				hasSensitiveFields = true
			}
			fields = append(fields, name)
		}
	}
	summary := map[string]any{
		"fieldCount":          fieldCount,
		"fields":              fields,
		"truncatedFields":     truncatedFields,
		"truncatedFieldNames": truncatedFieldNames,
		"hasSensitiveFields":  hasSensitiveFields,
	}
	for len(fields) > 0 {
		body, err := json.Marshal(summary)
		if err != nil || len(body) <= maxCapabilityResultInputSummaryBytes {
			break
		}
		fields = fields[:len(fields)-1]
		summary["fields"] = fields
		summary["truncatedFields"] = true
	}
	return summary
}

func capabilityInputFieldLooksSensitive(key string) bool {
	name := strings.ToLower(strings.TrimSpace(key))
	for _, marker := range []string{"secret", "token", "password", "passwd", "credential", "plaintext", "ciphertext", "signature", "private"} {
		if strings.Contains(name, marker) {
			return true
		}
	}
	return false
}

func inputString(input map[string]any, key string) string {
	if input == nil {
		return ""
	}
	switch v := input[key].(type) {
	case string:
		return strings.TrimSpace(v)
	case fmt.Stringer:
		return strings.TrimSpace(v.String())
	case json.Number:
		return strings.TrimSpace(v.String())
	case nil:
		return ""
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", v))
	}
}

func inputInt(input map[string]any, key string, fallback int) int {
	raw := inputString(input, key)
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return v
}

func inputBool(input map[string]any, key string) bool {
	raw := strings.ToLower(inputString(input, key))
	return raw == "1" || raw == "true" || raw == "yes"
}

func workspaceGlob(root, pattern string) ([]string, error) {
	if strings.TrimSpace(root) == "" {
		root = "."
	}
	root = filepath.Clean(root)
	pattern = strings.TrimSpace(pattern)
	if pattern == "" || pattern == "**" {
		pattern = "*"
	}
	matches := []string{}
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		rel = filepath.ToSlash(rel)
		if rel == "." {
			return nil
		}
		ok, _ := filepath.Match(filepath.ToSlash(pattern), rel)
		if !ok && strings.Contains(pattern, "**") {
			ok = simpleDoubleStarMatch(pattern, rel)
		}
		if ok {
			matches = append(matches, rel)
		}
		if len(matches) >= 500 {
			return filepath.SkipAll
		}
		return nil
	})
	sort.Strings(matches)
	return matches, err
}

func simpleDoubleStarMatch(pattern, rel string) bool {
	parts := strings.Split(pattern, "**")
	return strings.HasPrefix(rel, strings.Trim(parts[0], "/")) && strings.HasSuffix(rel, strings.Trim(parts[len(parts)-1], "/"))
}

func searchWorkspaceFiles(root, query string, limit int) ([]map[string]any, error) {
	query = strings.ToLower(strings.TrimSpace(query))
	if limit <= 0 {
		limit = 20
	}
	rows := []map[string]any{}
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 {
			return nil
		}
		if err := validateWorkspacePath(root, path); err != nil {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		score := 0
		if query == "" || strings.Contains(strings.ToLower(rel), query) {
			score += 2
		}
		if score == 0 {
			if b, err := readCapabilityFileBounded(path, "workspace search file", gatewayWorkspaceSearchReadLimit); err == nil && strings.Contains(strings.ToLower(string(b)), query) {
				score++
			}
		}
		if score > 0 {
			rows = append(rows, map[string]any{"path": rel, "score": score})
		}
		if len(rows) >= limit {
			return filepath.SkipAll
		}
		return nil
	})
	return rows, err
}
