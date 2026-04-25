package gateway

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"forge/projectforge/services/core/internal/aios/domain"
	"forge/projectforge/services/core/internal/approvals"
	"forge/projectforge/services/core/internal/audit"
	"forge/projectforge/services/core/internal/lanes"
	"forge/projectforge/services/core/internal/permissions"
	"forge/projectforge/services/core/internal/store"
)

func TestGatewayAllRegisteredToolsSmoke(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	workspace := t.TempDir()
	dataDir := t.TempDir()

	mustWrite(t, filepath.Join(workspace, "AGENTS.md"), "ok\n")
	mustWrite(t, filepath.Join(workspace, "CLAUDE.md"), "ok\n")
	mustWrite(t, filepath.Join(workspace, "docs", "FORGE_PROJECT_BRIEFING.md"), "ok\n")
	mustWrite(t, filepath.Join(workspace, "sample.txt"), "hello\n")
	mustWrite(t, filepath.Join(workspace, "tmp", "old.txt"), "old\n")
	mustWrite(t, filepath.Join(workspace, "tmp", "delete-me.txt"), "bye\n")

	mustRun(t, workspace, "git", "init")
	mustRun(t, workspace, "git", "config", "user.email", "forge@example.local")
	mustRun(t, workspace, "git", "config", "user.name", "FORGE")
	mustRun(t, workspace, "git", "add", ".")
	mustRun(t, workspace, "git", "commit", "-m", "initial")

	st, err := store.Open(dataDir)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	laneSvc := lanes.New(st.DB)
	if err := laneSvc.EnsureDefaults(ctx, workspace); err != nil {
		t.Fatalf("ensure lanes: %v", err)
	}
	permSvc := permissions.New(st.DB)
	if err := permSvc.EnsureDefaults(ctx, workspace); err != nil {
		t.Fatalf("ensure permissions: %v", err)
	}

	gw := New(Options{
		DB:           st.DB,
		Permissions:  permSvc,
		Lanes:        laneSvc,
		Approvals:    approvals.New(st.DB),
		Audit:        audit.New(st.DB),
		WorkspaceDir: workspace,
		DataDir:      dataDir,
	})

	toolRows := gw.Tools()
	toolIDs := make([]string, 0, len(toolRows))
	for _, row := range toolRows {
		toolIDs = append(toolIDs, row.ID)
	}
	if _, err := permSvc.Save(ctx, permissions.Profile{
		ID:                   "smoke_all",
		Name:                 "Smoke All",
		Description:          "Tool smoke verification profile",
		AllowedReadPaths:     []string{workspace},
		AllowedWritePaths:    []string{workspace},
		AllowedExecutePaths:  []string{workspace},
		ForbiddenPaths:       []string{},
		AllowedTools:         toolIDs,
		ApprovalRequiredRisk: []string{},
		MaxBytesPerWrite:     32 * 1024 * 1024,
		AllowNetwork:         true,
		Editable:             true,
		Active:               true,
	}); err != nil {
		t.Fatalf("save smoke profile: %v", err)
	}

	if _, err := st.DB.ExecContext(ctx, `INSERT OR REPLACE INTO settings(key, value) VALUES('secret.test_api', 'top-secret-token')`); err != nil {
		t.Fatalf("seed secret: %v", err)
	}

	for _, tool := range toolRows {
		tool := tool
		t.Run(tool.ID, func(t *testing.T) {
			laneID := "smoke." + strings.ReplaceAll(tool.ID, ".", "_")
			if _, err := laneSvc.Save(ctx, lanes.Lane{
				ID:                laneID,
				Name:              "Smoke " + tool.ID,
				Description:       "Smoke lane for " + tool.ID,
				ActionType:        tool.Action,
				AllowedPaths:      []string{workspace},
				ForbiddenPaths:    []string{},
				WriteIntent:       tool.WriteIntent,
				RequiresApproval:  false,
				RiskClass:         tool.RiskClass,
				MaxBytes:          32 * 1024 * 1024,
				ExpectedArtifacts: []string{},
				Enabled:           true,
			}); err != nil {
				t.Fatalf("save lane: %v", err)
			}

			req := smokeRequestForTool(t, workspace, tool.ID, laneID)
			res, err := gw.Execute(ctx, req)
			if err != nil {
				t.Fatalf("execute failed: %v", err)
			}
			if res == nil {
				t.Fatalf("nil result")
			}
			if capability, ok := gw.capabilities.Resolve(tool.ID); ok {
				switch capability.Status {
				case domain.ToolCapabilityApprovalOnly:
					if res.Status != StatusNeedsApprov {
						t.Fatalf("expected needs_approval for approval-only capability %s, got %s (%s)", capability.ID, res.Status, res.DeniedReason)
					}
					if res.InvocationID <= 0 {
						t.Fatalf("missing invocation id")
					}
					return
				case domain.ToolCapabilityDeferred:
					if res.Status != StatusUnsupported {
						t.Fatalf("expected unsupported for deferred capability %s, got %s (%s)", capability.ID, res.Status, res.DeniedReason)
					}
					if res.InvocationID <= 0 {
						t.Fatalf("missing invocation id")
					}
					return
				case domain.ToolCapabilityDisabled, domain.ToolCapabilityDeprecated:
					if res.Status != StatusDisabled {
						t.Fatalf("expected disabled for disabled capability %s, got %s (%s)", capability.ID, res.Status, res.DeniedReason)
					}
					if res.InvocationID <= 0 {
						t.Fatalf("missing invocation id")
					}
					return
				}
			}
			if res.Status == StatusDenied {
				t.Fatalf("unexpected denied result: %s", res.DeniedReason)
			}
			if res.Status == StatusNeedsApprov {
				t.Fatalf("unexpected needs_approval result for smoke lane")
			}
			if res.InvocationID <= 0 {
				t.Fatalf("missing invocation id")
			}
		})
	}
}

func TestGatewayApprovedJobBypassesRepeatedApproval(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	workspace := t.TempDir()
	dataDir := t.TempDir()

	st, err := store.Open(dataDir)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	laneSvc := lanes.New(st.DB)
	if err := laneSvc.EnsureDefaults(ctx, workspace); err != nil {
		t.Fatalf("ensure lanes: %v", err)
	}
	permSvc := permissions.New(st.DB)
	if err := permSvc.EnsureDefaults(ctx, workspace); err != nil {
		t.Fatalf("ensure permissions: %v", err)
	}

	gw := New(Options{
		DB:           st.DB,
		Permissions:  permSvc,
		Lanes:        laneSvc,
		Approvals:    approvals.New(st.DB),
		Audit:        audit.New(st.DB),
		WorkspaceDir: workspace,
		DataDir:      dataDir,
	})

	now := time.Now().UnixMilli()
	jobID := "job_approved_write"
	if _, err := st.DB.ExecContext(ctx, `
INSERT INTO jobs(
  id, created_at, updated_at, queued_at,
  title, requested_action, target_adapter, initiating_source,
  execution_boundary, risk_class, status, approval_status, write_intent, metadata_json
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		jobID, now, now, now,
		"approved write", "gateway.action", "forge", "test",
		"command_execution", "safe_write", "awaiting_approval", "granted", 1, "{}",
	); err != nil {
		t.Fatalf("seed approved job: %v", err)
	}

	req := Request{
		ToolID:        "fs.write",
		LaneID:        "fs.write.bounded",
		Action:        "invoke",
		CorrelationID: "test-approved-write",
		Paths:         []string{"scratch/repeated-approval-test.txt"},
		Input:         map[string]any{"contents": "approved write"},
		JobID:         &jobID,
		Initiator:     "test",
	}
	approveGatewayRequestForTest(t, ctx, gw, req, "approved write")
	res, err := gw.Execute(ctx, req)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if res == nil {
		t.Fatalf("nil result")
	}
	if res.Status != StatusOK {
		t.Fatalf("expected status ok, got %s (%s)", res.Status, res.DeniedReason)
	}
}

func smokeRequestForTool(t *testing.T, workspace, toolID, laneID string) Request {
	t.Helper()
	req := Request{
		ToolID:         toolID,
		LaneID:         laneID,
		Action:         "smoke",
		RiskClass:      "",
		ExecutionLevel: "",
		CorrelationID:  fmt.Sprintf("smoke-%d-%s", time.Now().UnixNano(), strings.ReplaceAll(toolID, ".", "_")),
		Paths:          []string{"."},
		Input:          map[string]any{},
		Initiator:      "test",
	}
	switch toolID {
	case "fs.read":
		req.Paths = []string{"sample.txt"}
	case "fs.list", "repo.inspect", "git.status", "git.diff", "git.branch", "validate.project_context":
		req.Paths = []string{"."}
	case "fs.write":
		req.Paths = []string{"tmp/write.txt"}
		req.Input = map[string]any{"contents": "written by smoke test\n"}
	case "fs.mkdir":
		req.Paths = []string{"tmp/new-dir"}
	case "fs.rename":
		req.Paths = []string{"tmp/old.txt", "tmp/renamed.txt"}
	case "fs.copy":
		req.Paths = []string{"sample.txt", "tmp/copied.txt"}
	case "fs.chmod":
		req.Paths = []string{"sample.txt"}
		req.Input = map[string]any{"mode": 0644}
	case "fs.delete":
		req.Paths = []string{"tmp/delete-me.txt"}
	case "proc.run":
		req.Paths = []string{"."}
		req.Input = map[string]any{"command": "echo smoke", "timeoutMs": 3000}
	case "proc.terminate":
		cmd := exec.Command("bash", "-lc", "sleep 10")
		cmd.Dir = workspace
		if err := cmd.Start(); err != nil {
			t.Fatalf("start sleep process: %v", err)
		}
		t.Cleanup(func() {
			if cmd.Process != nil {
				_ = cmd.Process.Kill()
				_, _ = cmd.Process.Wait()
			}
		})
		req.Input = map[string]any{"pid": cmd.Process.Pid}
	case "git.commit":
		mustWrite(t, filepath.Join(workspace, "tmp", "commit-file.txt"), "commit me\n")
		mustRun(t, workspace, "git", "add", "tmp/commit-file.txt")
		req.Paths = []string{"."}
		req.Input = map[string]any{"message": "smoke commit"}
	case "git.checkout":
		branch := strings.TrimSpace(mustOutput(t, workspace, "git", "branch", "--show-current"))
		if branch == "" {
			branch = "main"
		}
		req.Paths = []string{"."}
		req.Input = map[string]any{"ref": branch}
	case "git.stash":
		req.Paths = []string{"."}
		req.Input = map[string]any{"mode": "list"}
	case "git.apply_patch":
		req.Paths = []string{"."}
		req.Input = map[string]any{"patch": "diff --git a/nope b/nope\n"}
	case "system.service_status":
		req.Paths = nil
		req.Input = map[string]any{"service": "ssh"}
	case "system.service_control":
		req.Paths = nil
		req.Input = map[string]any{"service": "ssh", "control": "restart"}
	case "system.logs":
		req.Paths = nil
		req.Input = map[string]any{"service": "ssh", "lines": 10}
	case "desktop.notify":
		req.Paths = nil
		req.Input = map[string]any{"title": "FORGE", "body": "smoke"}
	case "desktop.open":
		req.Paths = []string{"sample.txt"}
	case "net.interfaces":
		req.Paths = nil
	case "net.connectivity":
		req.Paths = nil
		req.Input = map[string]any{"target": "1.1.1.1:53", "timeoutMs": 3000}
	case "net.dns_lookup":
		req.Paths = nil
		req.Input = map[string]any{"host": "localhost"}
	case "net.fetch":
		req.Paths = nil
		req.Input = map[string]any{"url": "https://example.com"}
	case "secret.get":
		req.Paths = nil
		req.Input = map[string]any{"name": "test_api"}
	}
	return req
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func mustRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run %q failed: %v\n%s", strings.Join(args, " "), err, string(out))
	}
}

func mustOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run %q failed: %v\n%s", strings.Join(args, " "), err, string(out))
	}
	return string(out)
}
