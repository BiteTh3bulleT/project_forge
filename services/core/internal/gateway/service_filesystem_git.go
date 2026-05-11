package gateway

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// --- Built-in tools ---

type readFileTool struct{ workspace string }

func (t *readFileTool) ID() string             { return "fs.read" }
func (t *readFileTool) Domain() string         { return "filesystem" }
func (t *readFileTool) Action() string         { return "read_file" }
func (t *readFileTool) RiskClass() string      { return "read_only" }
func (t *readFileTool) ExecutionLevel() string { return "L0" }
func (t *readFileTool) Executes() bool         { return false }
func (t *readFileTool) UsesNetwork() bool      { return false }
func (t *readFileTool) WriteIntent() bool      { return false }
func (t *readFileTool) Description() string    { return "Read a file from the workspace" }
func (t *readFileTool) Execute(ctx context.Context, req Request) (Result, error) {
	target, err := firstWorkspacePath(req.Paths, t.workspace)
	if err != nil {
		return Result{}, err
	}
	info, err := os.Stat(target)
	if err != nil {
		return Result{}, err
	}
	if info.IsDir() {
		return Result{}, fmt.Errorf("target %q is a directory, use fs.list", target)
	}
	if !info.Mode().IsRegular() {
		return Result{}, fmt.Errorf("target %q is not a regular file", target)
	}
	if info.Size() > 2*1024*1024 {
		return Result{}, fmt.Errorf("file too large (%d bytes)", info.Size())
	}
	f, err := os.Open(target)
	if err != nil {
		return Result{}, err
	}
	defer f.Close()
	buf := bytes.Buffer{}
	if _, err := io.Copy(&buf, f); err != nil {
		return Result{}, err
	}
	return Result{
		Data: map[string]any{
			"path":  target,
			"size":  info.Size(),
			"bytes": buf.Len(),
			"text":  buf.String(),
		},
		Message: fmt.Sprintf("read %d bytes from %s", buf.Len(), target),
	}, nil
}

type listDirTool struct{ workspace string }

func (t *listDirTool) ID() string             { return "fs.list" }
func (t *listDirTool) Domain() string         { return "filesystem" }
func (t *listDirTool) Action() string         { return "list_directory" }
func (t *listDirTool) RiskClass() string      { return "read_only" }
func (t *listDirTool) ExecutionLevel() string { return "L0" }
func (t *listDirTool) Executes() bool         { return false }
func (t *listDirTool) UsesNetwork() bool      { return false }
func (t *listDirTool) WriteIntent() bool      { return false }
func (t *listDirTool) Description() string    { return "List a directory inside the workspace" }
func (t *listDirTool) Execute(ctx context.Context, req Request) (Result, error) {
	target, err := firstWorkspacePath(req.Paths, t.workspace)
	if err != nil {
		return Result{}, err
	}
	entries, err := os.ReadDir(target)
	if err != nil {
		return Result{}, err
	}
	type entry struct {
		Name  string `json:"name"`
		IsDir bool   `json:"isDir"`
		Size  int64  `json:"size"`
	}
	out := []entry{}
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}
		out = append(out, entry{Name: e.Name(), IsDir: e.IsDir(), Size: info.Size()})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return Result{
		Data: map[string]any{
			"path":    target,
			"entries": out,
			"count":   len(out),
		},
		Message: fmt.Sprintf("listed %d entries in %s", len(out), target),
	}, nil
}

type repoInspectTool struct{ workspace string }

func (t *repoInspectTool) ID() string             { return "repo.inspect" }
func (t *repoInspectTool) Domain() string         { return "filesystem" }
func (t *repoInspectTool) Action() string         { return "inspect_repo" }
func (t *repoInspectTool) RiskClass() string      { return "read_only" }
func (t *repoInspectTool) ExecutionLevel() string { return "L0" }
func (t *repoInspectTool) Executes() bool         { return false }
func (t *repoInspectTool) UsesNetwork() bool      { return false }
func (t *repoInspectTool) WriteIntent() bool      { return false }
func (t *repoInspectTool) Description() string    { return "Return a shallow workspace inspection report" }
func (t *repoInspectTool) Execute(ctx context.Context, req Request) (Result, error) {
	target, err := workspaceDirFromRequest(req.Paths, t.workspace)
	if err != nil {
		return Result{}, err
	}
	entries, err := os.ReadDir(target)
	if err != nil {
		return Result{}, err
	}
	files := 0
	dirs := 0
	topFiles := []string{}
	topDirs := []string{}
	for _, e := range entries {
		if e.IsDir() {
			dirs++
			if len(topDirs) < 20 {
				topDirs = append(topDirs, e.Name())
			}
		} else {
			files++
			if len(topFiles) < 20 {
				topFiles = append(topFiles, e.Name())
			}
		}
	}
	return Result{
		Data: map[string]any{
			"path":     target,
			"files":    files,
			"dirs":     dirs,
			"topFiles": topFiles,
			"topDirs":  topDirs,
		},
		Message: fmt.Sprintf("inspected %s: %d files, %d dirs", target, files, dirs),
	}, nil
}

type gitStatusTool struct{ workspace string }

func (t *gitStatusTool) ID() string             { return "git.status" }
func (t *gitStatusTool) Domain() string         { return "git" }
func (t *gitStatusTool) Action() string         { return "status" }
func (t *gitStatusTool) RiskClass() string      { return "read_only" }
func (t *gitStatusTool) ExecutionLevel() string { return "L0" }
func (t *gitStatusTool) Executes() bool         { return true }
func (t *gitStatusTool) UsesNetwork() bool      { return false }
func (t *gitStatusTool) WriteIntent() bool      { return false }
func (t *gitStatusTool) Description() string    { return "Return git status --short for the workspace" }
func (t *gitStatusTool) Execute(ctx context.Context, req Request) (Result, error) {
	dir, err := workspaceDirFromRequest(req.Paths, t.workspace)
	if err != nil {
		return Result{}, err
	}
	cmd := exec.CommandContext(ctx, "git", "status", "--short", "--branch")
	cmd.Dir = dir
	out, err := boundedCombinedOutput(cmd)
	if err != nil {
		return Result{Data: map[string]any{"path": dir, "available": false, "error": err.Error(), "output": out}}, nil
	}
	return Result{
		Data: map[string]any{
			"path":      dir,
			"available": true,
			"output":    out,
		},
		Message: "git status captured",
	}, nil
}

type gitDiffTool struct{ workspace string }

func (t *gitDiffTool) ID() string             { return "git.diff" }
func (t *gitDiffTool) Domain() string         { return "git" }
func (t *gitDiffTool) Action() string         { return "diff" }
func (t *gitDiffTool) RiskClass() string      { return "read_only" }
func (t *gitDiffTool) ExecutionLevel() string { return "L0" }
func (t *gitDiffTool) Executes() bool         { return true }
func (t *gitDiffTool) UsesNetwork() bool      { return false }
func (t *gitDiffTool) WriteIntent() bool      { return false }
func (t *gitDiffTool) Description() string    { return "Return `git diff --stat` for the workspace" }
func (t *gitDiffTool) Execute(ctx context.Context, req Request) (Result, error) {
	dir, err := workspaceDirFromRequest(req.Paths, t.workspace)
	if err != nil {
		return Result{}, err
	}
	cmd := exec.CommandContext(ctx, "git", "diff", "--stat")
	cmd.Dir = dir
	out, err := boundedCombinedOutput(cmd)
	if err != nil {
		return Result{Data: map[string]any{"path": dir, "available": false, "error": err.Error(), "output": out}}, nil
	}
	return Result{
		Data: map[string]any{
			"path":      dir,
			"available": true,
			"output":    out,
		},
		Message: "git diff --stat captured",
	}, nil
}

type writeFileTool struct{ workspace string }

func (t *writeFileTool) ID() string             { return "fs.write" }
func (t *writeFileTool) Domain() string         { return "filesystem" }
func (t *writeFileTool) Action() string         { return "write_file" }
func (t *writeFileTool) RiskClass() string      { return "safe_write" }
func (t *writeFileTool) ExecutionLevel() string { return "L1" }
func (t *writeFileTool) Executes() bool         { return false }
func (t *writeFileTool) UsesNetwork() bool      { return false }
func (t *writeFileTool) WriteIntent() bool      { return true }
func (t *writeFileTool) Description() string    { return "Write content to a file inside approved scope" }
func (t *writeFileTool) Execute(ctx context.Context, req Request) (Result, error) {
	if rawFiles, ok := req.Input["files"]; ok {
		return t.executeBatch(ctx, req, rawFiles)
	}
	target, err := firstWorkspacePath(req.Paths, t.workspace)
	if err != nil {
		return Result{}, err
	}
	contents, _ := req.Input["contents"].(string)
	if contents == "" {
		return Result{}, errors.New("fs.write requires input.contents")
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return Result{}, err
	}
	if err := validateWorkspacePath(t.workspace, target); err != nil {
		return Result{}, err
	}
	if err := os.WriteFile(target, []byte(contents), 0o644); err != nil {
		return Result{}, err
	}
	return Result{
		Data: map[string]any{
			"path":  target,
			"bytes": len(contents),
		},
		Artifacts: []ResultArtifact{{Type: "writtenFile", Path: target, Summary: fmt.Sprintf("%d bytes", len(contents))}},
		Message:   fmt.Sprintf("wrote %d bytes to %s", len(contents), target),
	}, nil
}

func (t *writeFileTool) executeBatch(ctx context.Context, req Request, rawFiles any) (Result, error) {
	files, err := writeBatchFiles(rawFiles)
	if err != nil {
		return Result{}, err
	}
	if len(files) == 0 {
		return Result{}, errors.New("fs.write batch requires input.files")
	}
	if len(req.Paths) != len(files) {
		return Result{}, fmt.Errorf("fs.write batch requires one path per file: got %d paths for %d files", len(req.Paths), len(files))
	}

	outFiles := make([]map[string]any, 0, len(files))
	artifacts := make([]ResultArtifact, 0, len(files))
	totalBytes := 0
	for i, file := range files {
		contents := file.Contents
		if contents == "" {
			return Result{}, fmt.Errorf("fs.write batch file %d requires contents", i)
		}
		target, err := firstWorkspacePath([]string{req.Paths[i]}, t.workspace)
		if err != nil {
			return Result{}, err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return Result{}, err
		}
		if err := validateWorkspacePath(t.workspace, target); err != nil {
			return Result{}, err
		}
		if err := os.WriteFile(target, []byte(contents), 0o644); err != nil {
			return Result{}, err
		}
		n := len(contents)
		totalBytes += n
		outFiles = append(outFiles, map[string]any{"path": target, "bytes": n})
		artifacts = append(artifacts, ResultArtifact{Type: "writtenFile", Path: target, Summary: fmt.Sprintf("%d bytes", n)})
	}

	paths := make([]string, 0, len(outFiles))
	for _, file := range outFiles {
		if p, ok := file["path"].(string); ok {
			paths = append(paths, p)
		}
	}
	return Result{
		Data: map[string]any{
			"files": outFiles,
			"paths": paths,
			"count": len(outFiles),
			"bytes": totalBytes,
		},
		Artifacts: artifacts,
		Message:   fmt.Sprintf("wrote %d files (%d bytes)", len(outFiles), totalBytes),
	}, nil
}

type writeBatchFile struct {
	Contents string
}

func writeBatchFiles(raw any) ([]writeBatchFile, error) {
	switch typed := raw.(type) {
	case []map[string]any:
		out := make([]writeBatchFile, 0, len(typed))
		for _, item := range typed {
			contents, _ := item["contents"].(string)
			out = append(out, writeBatchFile{Contents: contents})
		}
		return out, nil
	case []any:
		out := make([]writeBatchFile, 0, len(typed))
		for _, item := range typed {
			rec, ok := item.(map[string]any)
			if !ok {
				return nil, errors.New("fs.write batch input.files must contain objects")
			}
			contents, _ := rec["contents"].(string)
			out = append(out, writeBatchFile{Contents: contents})
		}
		return out, nil
	default:
		return nil, errors.New("fs.write batch input.files must be an array")
	}
}

type validateContextTool struct {
	workspace string
	dataDir   string
}

func (t *validateContextTool) ID() string             { return "validate.project_context" }
func (t *validateContextTool) Domain() string         { return "filesystem" }
func (t *validateContextTool) Action() string         { return "validate_context" }
func (t *validateContextTool) RiskClass() string      { return "read_only" }
func (t *validateContextTool) ExecutionLevel() string { return "L0" }
func (t *validateContextTool) Executes() bool         { return false }
func (t *validateContextTool) UsesNetwork() bool      { return false }
func (t *validateContextTool) WriteIntent() bool      { return false }
func (t *validateContextTool) Description() string {
	return "Check that project context artifacts exist and are non-empty"
}
func (t *validateContextTool) Execute(ctx context.Context, req Request) (Result, error) {
	want := []string{"AGENTS.md", "CLAUDE.md", "docs/FORGE_PROJECT_BRIEFING.md"}
	report := map[string]any{}
	missing := []string{}
	for _, rel := range want {
		p := filepath.Join(t.workspace, rel)
		info, err := os.Stat(p)
		switch {
		case err != nil:
			report[rel] = map[string]any{"exists": false}
			missing = append(missing, rel)
		case info.Size() == 0:
			report[rel] = map[string]any{"exists": true, "sizeBytes": 0, "empty": true}
			missing = append(missing, rel)
		default:
			report[rel] = map[string]any{"exists": true, "sizeBytes": info.Size()}
		}
	}
	status := "ok"
	if len(missing) > 0 {
		status = "incomplete"
	}
	return Result{
		Data: map[string]any{
			"status":  status,
			"report":  report,
			"missing": missing,
		},
		Message: fmt.Sprintf("project context validation: %s", status),
	}, nil
}

type mkdirTool struct{ workspace string }

func (t *mkdirTool) ID() string             { return "fs.mkdir" }
func (t *mkdirTool) Domain() string         { return "filesystem" }
func (t *mkdirTool) Action() string         { return "make_directory" }
func (t *mkdirTool) RiskClass() string      { return "low_write" }
func (t *mkdirTool) ExecutionLevel() string { return "L1" }
func (t *mkdirTool) Executes() bool         { return false }
func (t *mkdirTool) UsesNetwork() bool      { return false }
func (t *mkdirTool) WriteIntent() bool      { return true }
func (t *mkdirTool) Description() string    { return "Create a directory" }
func (t *mkdirTool) Execute(ctx context.Context, req Request) (Result, error) {
	target, err := firstWorkspacePath(req.Paths, t.workspace)
	if err != nil {
		return Result{}, err
	}
	if err := os.MkdirAll(target, 0o755); err != nil {
		return Result{}, err
	}
	if err := validateWorkspacePath(t.workspace, target); err != nil {
		return Result{}, err
	}
	return Result{Data: map[string]any{"path": target}, Message: "directory created"}, nil
}

type renameTool struct{ workspace string }

func (t *renameTool) ID() string             { return "fs.rename" }
func (t *renameTool) Domain() string         { return "filesystem" }
func (t *renameTool) Action() string         { return "rename_move" }
func (t *renameTool) RiskClass() string      { return "safe_write" }
func (t *renameTool) ExecutionLevel() string { return "L1" }
func (t *renameTool) Executes() bool         { return false }
func (t *renameTool) UsesNetwork() bool      { return false }
func (t *renameTool) WriteIntent() bool      { return true }
func (t *renameTool) Description() string    { return "Rename or move a file/directory" }
func (t *renameTool) Execute(ctx context.Context, req Request) (Result, error) {
	if len(req.Paths) < 2 {
		return Result{}, errors.New("fs.rename requires source and destination paths")
	}
	src, err := firstWorkspacePath(req.Paths[:1], t.workspace)
	if err != nil {
		return Result{}, err
	}
	dst, err := firstWorkspacePath(req.Paths[1:], t.workspace)
	if err != nil {
		return Result{}, err
	}
	if err := os.Rename(src, dst); err != nil {
		return Result{}, err
	}
	return Result{Data: map[string]any{"from": src, "to": dst}, Message: "rename completed"}, nil
}

type deleteTool struct{ workspace string }

func (t *deleteTool) ID() string             { return "fs.delete" }
func (t *deleteTool) Domain() string         { return "filesystem" }
func (t *deleteTool) Action() string         { return "delete_path" }
func (t *deleteTool) RiskClass() string      { return "dangerous" }
func (t *deleteTool) ExecutionLevel() string { return "L4" }
func (t *deleteTool) Executes() bool         { return false }
func (t *deleteTool) UsesNetwork() bool      { return false }
func (t *deleteTool) WriteIntent() bool      { return true }
func (t *deleteTool) Description() string    { return "Delete a file or directory recursively" }
func (t *deleteTool) Execute(ctx context.Context, req Request) (Result, error) {
	target, err := firstWorkspacePath(req.Paths, t.workspace)
	if err != nil {
		return Result{}, err
	}
	if err := os.RemoveAll(target); err != nil {
		return Result{}, err
	}
	return Result{Data: map[string]any{"path": target}, Message: "path deleted"}, nil
}

type copyTool struct{ workspace string }

func (t *copyTool) ID() string             { return "fs.copy" }
func (t *copyTool) Domain() string         { return "filesystem" }
func (t *copyTool) Action() string         { return "copy_file" }
func (t *copyTool) RiskClass() string      { return "safe_write" }
func (t *copyTool) ExecutionLevel() string { return "L1" }
func (t *copyTool) Executes() bool         { return false }
func (t *copyTool) UsesNetwork() bool      { return false }
func (t *copyTool) WriteIntent() bool      { return true }
func (t *copyTool) Description() string    { return "Copy a file from source to destination" }
func (t *copyTool) Execute(ctx context.Context, req Request) (Result, error) {
	if len(req.Paths) < 2 {
		return Result{}, errors.New("fs.copy requires source and destination paths")
	}
	src, err := firstWorkspacePath(req.Paths[:1], t.workspace)
	if err != nil {
		return Result{}, err
	}
	dst, err := firstWorkspacePath(req.Paths[1:], t.workspace)
	if err != nil {
		return Result{}, err
	}
	in, err := os.Open(src)
	if err != nil {
		return Result{}, err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return Result{}, err
	}
	if err := validateWorkspacePath(t.workspace, dst); err != nil {
		return Result{}, err
	}
	out, err := os.Create(dst)
	if err != nil {
		return Result{}, err
	}
	defer out.Close()
	written, err := io.Copy(out, in)
	if err != nil {
		return Result{}, err
	}
	return Result{Data: map[string]any{"from": src, "to": dst, "bytes": written}, Message: "copy completed"}, nil
}

type chmodTool struct{ workspace string }

func normalizeChmodMode(raw string) (string, uint32, error) {
	modeRaw := strings.TrimSpace(raw)
	if modeRaw == "" {
		modeRaw = "0644"
	}
	v, err := strconv.ParseUint(modeRaw, 8, 32)
	if err != nil {
		return "", 0, fmt.Errorf("invalid mode %q", modeRaw)
	}
	if v > 0o777 {
		return "", 0, fmt.Errorf("mode %q sets unsupported special bits", modeRaw)
	}
	return modeRaw, uint32(v), nil
}

func (t *chmodTool) ID() string             { return "fs.chmod" }
func (t *chmodTool) Domain() string         { return "filesystem" }
func (t *chmodTool) Action() string         { return "chmod" }
func (t *chmodTool) RiskClass() string      { return "privileged" }
func (t *chmodTool) ExecutionLevel() string { return "L3" }
func (t *chmodTool) Executes() bool         { return false }
func (t *chmodTool) UsesNetwork() bool      { return false }
func (t *chmodTool) WriteIntent() bool      { return true }
func (t *chmodTool) Description() string    { return "Change file mode for a path" }
func (t *chmodTool) Execute(ctx context.Context, req Request) (Result, error) {
	target, err := firstWorkspacePath(req.Paths, t.workspace)
	if err != nil {
		return Result{}, err
	}
	modeRaw, v, err := normalizeChmodMode(inputString(req.Input, "mode"))
	if err != nil {
		return Result{}, err
	}
	if err := os.Chmod(target, os.FileMode(v)); err != nil {
		return Result{}, err
	}
	return Result{Data: map[string]any{"path": target, "mode": modeRaw}, Message: "chmod applied"}, nil
}

type gitBranchTool struct{ workspace string }

func (t *gitBranchTool) ID() string             { return "git.branch" }
func (t *gitBranchTool) Domain() string         { return "git" }
func (t *gitBranchTool) Action() string         { return "branch" }
func (t *gitBranchTool) RiskClass() string      { return "read_only" }
func (t *gitBranchTool) ExecutionLevel() string { return "L0" }
func (t *gitBranchTool) Executes() bool         { return true }
func (t *gitBranchTool) UsesNetwork() bool      { return false }
func (t *gitBranchTool) WriteIntent() bool      { return false }
func (t *gitBranchTool) Description() string    { return "List git branches" }
func (t *gitBranchTool) Execute(ctx context.Context, req Request) (Result, error) {
	dir, err := workspaceDirFromRequest(req.Paths, t.workspace)
	if err != nil {
		return Result{}, err
	}
	out, err := runCmd(ctx, dir, "git", "branch", "--all", "--verbose")
	return Result{Data: map[string]any{"path": dir, "output": out, "ok": err == nil}, Message: "git branch captured"}, nil
}

type gitCommitTool struct{ workspace string }

const maxGitMessageInputBytes = 16 << 10

var errGitMessageTooLarge = errors.New("git message input too large")

func normalizeGitMessageInput(raw, toolID string) (string, error) {
	message := strings.TrimSpace(raw)
	if len(message) > maxGitMessageInputBytes {
		return "", fmt.Errorf("%w: %d > %d bytes", errGitMessageTooLarge, len(message), maxGitMessageInputBytes)
	}
	return message, nil
}

func (t *gitCommitTool) ID() string             { return "git.commit" }
func (t *gitCommitTool) Domain() string         { return "git" }
func (t *gitCommitTool) Action() string         { return "commit" }
func (t *gitCommitTool) RiskClass() string      { return "dangerous" }
func (t *gitCommitTool) ExecutionLevel() string { return "L4" }
func (t *gitCommitTool) Executes() bool         { return true }
func (t *gitCommitTool) UsesNetwork() bool      { return false }
func (t *gitCommitTool) WriteIntent() bool      { return true }
func (t *gitCommitTool) Description() string    { return "Create git commit with provided message" }
func (t *gitCommitTool) Execute(ctx context.Context, req Request) (Result, error) {
	dir, err := workspaceDirFromRequest(req.Paths, t.workspace)
	if err != nil {
		return Result{}, err
	}
	message, err := normalizeGitMessageInput(inputString(req.Input, "message"), t.ID())
	if err != nil {
		return Result{}, err
	}
	if message == "" {
		message = "FORGE gateway commit"
	}
	out, err := runCmd(ctx, dir, "git", "commit", "-m", message)
	return Result{Data: map[string]any{"path": dir, "message": message, "output": out, "ok": err == nil}, Message: "git commit executed"}, nil
}

type gitCheckoutTool struct{ workspace string }

var gitCheckoutRefPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._/@+^~-]*$`)

func normalizeGitCheckoutRef(raw string) (string, error) {
	ref := strings.TrimSpace(raw)
	if ref == "" {
		return "", errors.New("git checkout ref required")
	}
	if len(ref) > 256 {
		return "", errors.New("git checkout ref too long")
	}
	if strings.HasPrefix(ref, "-") {
		return "", errors.New("git checkout ref must not start with '-'")
	}
	if strings.Contains(ref, "..") ||
		strings.Contains(ref, "@{") ||
		strings.Contains(ref, `\`) ||
		strings.HasSuffix(ref, ".lock") ||
		!gitCheckoutRefPattern.MatchString(ref) {
		return "", errors.New("git checkout ref contains unsafe characters")
	}
	return ref, nil
}

func (t *gitCheckoutTool) ID() string             { return "git.checkout" }
func (t *gitCheckoutTool) Domain() string         { return "git" }
func (t *gitCheckoutTool) Action() string         { return "checkout" }
func (t *gitCheckoutTool) RiskClass() string      { return "dangerous" }
func (t *gitCheckoutTool) ExecutionLevel() string { return "L4" }
func (t *gitCheckoutTool) Executes() bool         { return true }
func (t *gitCheckoutTool) UsesNetwork() bool      { return false }
func (t *gitCheckoutTool) WriteIntent() bool      { return true }
func (t *gitCheckoutTool) Description() string    { return "Git checkout branch/ref" }
func (t *gitCheckoutTool) Execute(ctx context.Context, req Request) (Result, error) {
	dir, err := workspaceDirFromRequest(req.Paths, t.workspace)
	if err != nil {
		return Result{}, err
	}
	ref, err := normalizeGitCheckoutRef(inputString(req.Input, "ref"))
	if err != nil {
		return Result{}, errors.New("git.checkout requires input.ref")
	}
	out, err := runCmd(ctx, dir, "git", "checkout", ref)
	return Result{Data: map[string]any{"path": dir, "ref": ref, "output": out, "ok": err == nil}, Message: "git checkout executed"}, nil
}

type gitStashTool struct{ workspace string }

func normalizeGitStashMode(raw string) (string, error) {
	mode := strings.TrimSpace(strings.ToLower(raw))
	if mode == "" {
		return "push", nil
	}
	switch mode {
	case "push", "pop", "list":
		return mode, nil
	default:
		return "", errors.New("git.stash supports mode=push|pop|list")
	}
}

func (t *gitStashTool) ID() string             { return "git.stash" }
func (t *gitStashTool) Domain() string         { return "git" }
func (t *gitStashTool) Action() string         { return "stash" }
func (t *gitStashTool) RiskClass() string      { return "safe_write" }
func (t *gitStashTool) ExecutionLevel() string { return "L1" }
func (t *gitStashTool) Executes() bool         { return true }
func (t *gitStashTool) UsesNetwork() bool      { return false }
func (t *gitStashTool) WriteIntent() bool      { return true }
func (t *gitStashTool) Description() string    { return "Git stash push/pop/list" }
func (t *gitStashTool) Execute(ctx context.Context, req Request) (Result, error) {
	dir, err := workspaceDirFromRequest(req.Paths, t.workspace)
	if err != nil {
		return Result{}, err
	}
	mode, err := normalizeGitStashMode(inputString(req.Input, "mode"))
	if err != nil {
		return Result{}, err
	}
	args := []string{"stash", mode}
	if mode == "push" {
		msg, err := normalizeGitMessageInput(inputString(req.Input, "message"), t.ID())
		if err != nil {
			return Result{}, err
		}
		if msg != "" {
			args = append(args, "-m", msg)
		}
	}
	out, err := runCmd(ctx, dir, append([]string{"git"}, args...)...)
	return Result{Data: map[string]any{"path": dir, "mode": mode, "output": out, "ok": err == nil}, Message: "git stash executed"}, nil
}

type gitApplyPatchTool struct{ workspace string }

const maxGitPatchInputBytes = 2 << 20

var errGitPatchTooLarge = errors.New("git patch input too large")

func normalizeGitPatchInput(raw, toolID string) (string, error) {
	patch := strings.TrimSpace(raw)
	if patch == "" {
		return "", fmt.Errorf("%s requires input.patch", toolID)
	}
	if len(patch) > maxGitPatchInputBytes {
		return "", fmt.Errorf("%w: %d > %d bytes", errGitPatchTooLarge, len(patch), maxGitPatchInputBytes)
	}
	return patch, nil
}

func (t *gitApplyPatchTool) ID() string             { return "git.apply_patch" }
func (t *gitApplyPatchTool) Domain() string         { return "git" }
func (t *gitApplyPatchTool) Action() string         { return "apply_patch" }
func (t *gitApplyPatchTool) RiskClass() string      { return "dangerous" }
func (t *gitApplyPatchTool) ExecutionLevel() string { return "L4" }
func (t *gitApplyPatchTool) Executes() bool         { return true }
func (t *gitApplyPatchTool) UsesNetwork() bool      { return false }
func (t *gitApplyPatchTool) WriteIntent() bool      { return true }
func (t *gitApplyPatchTool) Description() string    { return "Apply git patch from input.patch" }
func (t *gitApplyPatchTool) Execute(ctx context.Context, req Request) (Result, error) {
	dir, err := workspaceDirFromRequest(req.Paths, t.workspace)
	if err != nil {
		return Result{}, err
	}
	patch, err := normalizeGitPatchInput(inputString(req.Input, "patch"), t.ID())
	if err != nil {
		return Result{}, err
	}
	cmd := exec.CommandContext(ctx, "git", "apply", "-")
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader(patch)
	out, err := boundedCombinedOutput(cmd)
	return Result{Data: map[string]any{"path": dir, "output": out, "ok": err == nil}, Message: "git apply executed"}, nil
}
