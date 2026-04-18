package api

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"forge/projectforge/services/core/internal/gateway"
)

// runChatFSDeterministicFallback runs mkdir / mkdir+write from parsed user text when the
// model omitted tool_calls. Returns true if any gateway invocation was attempted.
func (s *Server) runChatFSDeterministicFallback(
	ctx context.Context,
	corr string,
	lastUserContent string,
	forcedModel string,
	pushStage func(string, map[string]any),
	gwActivity map[string]any,
	final *strings.Builder,
) bool {
	_, _, _, combined := gateway.ParseCombinedMkdirAndWrite(lastUserContent)
	if combined {
		return s.runDeterministicMkdirThenWrite(ctx, corr, lastUserContent, pushStage, gwActivity, final)
	}
	if writePath, script, ok := gateway.ParsePythonBannerScriptIntent(lastUserContent); ok {
		pushStage("deterministic_python_banner_dispatch", map[string]any{"path": writePath})
		res, err := s.gwChatExec(ctx, corr, "fs.write", []string{writePath}, map[string]any{"contents": script})
		if err != nil {
			if final.Len() > 0 {
				final.WriteString("\n\n")
			}
			final.WriteString("FORGE (deterministic python): " + err.Error())
			gwActivity["executionState"] = "error"
			gwActivity["failureReason"] = err.Error()
			gwActivity["toolSelected"] = "fs.write"
			gwActivity["toolArgs"] = map[string]any{"path": writePath}
			gwActivity["syntheticToolExecution"] = true
			return true
		}
		if res.Status != gateway.StatusOK {
			if final.Len() > 0 {
				final.WriteString("\n\n")
			}
			final.WriteString(fmt.Sprintf("FORGE (deterministic python): gateway %s — %s", res.Status, strings.TrimSpace(coalesceReason(res))))
			gwActivity["executionState"] = res.Status
			gwActivity["failureReason"] = coalesceReason(res)
			gwActivity["toolSelected"] = "fs.write"
			gwActivity["toolArgs"] = map[string]any{"path": writePath}
			gwActivity["executionResult"] = res.Data
			gwActivity["syntheticToolExecution"] = true
			return true
		}
		if final.Len() > 0 {
			final.WriteString("\n\n")
		}
		final.WriteString("FORGE (deterministic python): " + formatToolResult("fs.write", res))
		gwActivity["executionState"] = "ok"
		gwActivity["toolSelected"] = "fs.write"
		gwActivity["toolArgs"] = map[string]any{"path": writePath}
		gwActivity["executionResult"] = res.Data
		gwActivity["syntheticToolExecution"] = true
		return true
	}
	if forcedModel == gateway.ChatModelName("fs.list") {
		listPath, ok := gateway.ParseListPath(lastUserContent)
		if ok {
			pushStage("deterministic_list_dispatch", map[string]any{"path": listPath})
			res, err := s.gwChatExec(ctx, corr, "fs.list", []string{listPath}, nil)
			if err != nil {
				if final.Len() > 0 {
					final.WriteString("\n\n")
				}
				final.WriteString("FORGE (deterministic list): " + err.Error())
				gwActivity["executionState"] = "error"
				gwActivity["failureReason"] = err.Error()
				gwActivity["toolSelected"] = "fs.list"
				gwActivity["toolArgs"] = map[string]any{"path": listPath}
				gwActivity["syntheticToolExecution"] = true
				return true
			}
			if res.Status != gateway.StatusOK {
				if final.Len() > 0 {
					final.WriteString("\n\n")
				}
				final.WriteString(fmt.Sprintf("FORGE (deterministic list): gateway %s — %s", res.Status, strings.TrimSpace(coalesceReason(res))))
				gwActivity["executionState"] = res.Status
				gwActivity["failureReason"] = coalesceReason(res)
				gwActivity["toolSelected"] = "fs.list"
				gwActivity["toolArgs"] = map[string]any{"path": listPath}
				gwActivity["executionResult"] = res.Data
				gwActivity["syntheticToolExecution"] = true
				return true
			}
			if final.Len() > 0 {
				final.WriteString("\n\n")
			}
			final.WriteString("FORGE (deterministic list): " + formatToolResult("fs.list", res))
			gwActivity["toolSelected"] = "fs.list"
			gwActivity["toolArgs"] = map[string]any{"path": listPath}
			gwActivity["executionState"] = "ok"
			gwActivity["executionResult"] = res.Data
			gwActivity["syntheticToolExecution"] = true
			return true
		}
	}
	if forcedModel == gateway.ChatModelName("fs.read") {
		readPath, ok := gateway.ParseReadPath(lastUserContent)
		if ok {
			pushStage("deterministic_read_dispatch", map[string]any{"path": readPath})
			res, err := s.gwChatExec(ctx, corr, "fs.read", []string{readPath}, nil)
			if err != nil {
				if final.Len() > 0 {
					final.WriteString("\n\n")
				}
				final.WriteString("FORGE (deterministic read): " + err.Error())
				gwActivity["executionState"] = "error"
				gwActivity["failureReason"] = err.Error()
				gwActivity["toolSelected"] = "fs.read"
				gwActivity["toolArgs"] = map[string]any{"path": readPath}
				gwActivity["syntheticToolExecution"] = true
				return true
			}
			if res.Status != gateway.StatusOK {
				if final.Len() > 0 {
					final.WriteString("\n\n")
				}
				final.WriteString(fmt.Sprintf("FORGE (deterministic read): gateway %s — %s", res.Status, strings.TrimSpace(coalesceReason(res))))
				gwActivity["executionState"] = res.Status
				gwActivity["failureReason"] = coalesceReason(res)
				gwActivity["toolSelected"] = "fs.read"
				gwActivity["toolArgs"] = map[string]any{"path": readPath}
				gwActivity["executionResult"] = res.Data
				gwActivity["syntheticToolExecution"] = true
				return true
			}
			if final.Len() > 0 {
				final.WriteString("\n\n")
			}
			final.WriteString("FORGE (deterministic read): " + formatToolResult("fs.read", res))
			gwActivity["toolSelected"] = "fs.read"
			gwActivity["toolArgs"] = map[string]any{"path": readPath}
			gwActivity["executionState"] = "ok"
			gwActivity["executionResult"] = res.Data
			gwActivity["syntheticToolExecution"] = true
			return true
		}
	}
	if forcedModel == gateway.ChatModelName("proc.run") {
		cmd, ok := gateway.ParseShellCommand(lastUserContent)
		if ok {
			pushStage("deterministic_proc_dispatch", map[string]any{"command": cmd})
			res, err := s.gwChatExec(ctx, corr, "proc.run", []string{"."}, map[string]any{"command": cmd})
			if err != nil {
				if final.Len() > 0 {
					final.WriteString("\n\n")
				}
				final.WriteString("FORGE (deterministic proc): " + err.Error())
				gwActivity["executionState"] = "error"
				gwActivity["failureReason"] = err.Error()
				gwActivity["toolSelected"] = "proc.run"
				gwActivity["toolArgs"] = map[string]any{"command": cmd}
				gwActivity["syntheticToolExecution"] = true
				return true
			}
			if res.Status != gateway.StatusOK {
				if final.Len() > 0 {
					final.WriteString("\n\n")
				}
				final.WriteString(fmt.Sprintf("FORGE (deterministic proc): gateway %s — %s", res.Status, strings.TrimSpace(coalesceReason(res))))
				gwActivity["executionState"] = res.Status
				gwActivity["failureReason"] = coalesceReason(res)
				gwActivity["toolSelected"] = "proc.run"
				gwActivity["toolArgs"] = map[string]any{"command": cmd}
				gwActivity["executionResult"] = res.Data
				gwActivity["syntheticToolExecution"] = true
				return true
			}
			if final.Len() > 0 {
				final.WriteString("\n\n")
			}
			final.WriteString("FORGE (deterministic proc): " + formatToolResult("proc.run", res))
			gwActivity["toolSelected"] = "proc.run"
			gwActivity["toolArgs"] = map[string]any{"command": cmd}
			gwActivity["executionState"] = "ok"
			gwActivity["executionResult"] = res.Data
			gwActivity["syntheticToolExecution"] = true
			return true
		}
	}
	if forcedModel == gateway.ChatModelName("git.status") {
		pushStage("deterministic_git_status_dispatch", map[string]any{"path": "."})
		res, err := s.gwChatExec(ctx, corr, "git.status", []string{"."}, nil)
		if err != nil {
			if final.Len() > 0 {
				final.WriteString("\n\n")
			}
			final.WriteString("FORGE (deterministic git): " + err.Error())
			gwActivity["executionState"] = "error"
			gwActivity["failureReason"] = err.Error()
			gwActivity["toolSelected"] = "git.status"
			gwActivity["toolArgs"] = map[string]any{"path": "."}
			gwActivity["syntheticToolExecution"] = true
			return true
		}
		if res.Status != gateway.StatusOK {
			if final.Len() > 0 {
				final.WriteString("\n\n")
			}
			final.WriteString(fmt.Sprintf("FORGE (deterministic git): gateway %s — %s", res.Status, strings.TrimSpace(coalesceReason(res))))
			gwActivity["executionState"] = res.Status
			gwActivity["failureReason"] = coalesceReason(res)
			gwActivity["toolSelected"] = "git.status"
			gwActivity["toolArgs"] = map[string]any{"path": "."}
			gwActivity["executionResult"] = res.Data
			gwActivity["syntheticToolExecution"] = true
			return true
		}
		if final.Len() > 0 {
			final.WriteString("\n\n")
		}
		final.WriteString("FORGE (deterministic git): " + formatToolResult("git.status", res))
		gwActivity["toolSelected"] = "git.status"
		gwActivity["toolArgs"] = map[string]any{"path": "."}
		gwActivity["executionState"] = "ok"
		gwActivity["executionResult"] = res.Data
		gwActivity["syntheticToolExecution"] = true
		return true
	}
	if forcedModel == gateway.ChatModelName("fs.mkdir") {
		dir, ok := gateway.ParseDirectoryCalled(lastUserContent)
		if !ok {
			dir, ok = gateway.ParseMkdirShellPath(lastUserContent)
		}
		if ok {
			return s.runDeterministicMkdirOnly(ctx, corr, dir, pushStage, gwActivity, final)
		}
	}
	return false
}

func (s *Server) gwChatExec(ctx context.Context, corr, toolID string, paths []string, input map[string]any) (*gateway.Result, error) {
	lane, ok := gateway.DefaultChatLane(toolID)
	if !ok {
		return nil, fmt.Errorf("no chat lane for tool %q", toolID)
	}
	if input == nil {
		input = map[string]any{}
	}
	return s.gateway.Execute(ctx, gateway.Request{
		ToolID:        toolID,
		LaneID:        lane,
		Domain:        domainForTool(toolID),
		Action:        "invoke",
		CorrelationID: corr,
		Paths:         paths,
		Input:         input,
		Initiator:     "chat_deterministic",
		DryRun:        false,
	})
}

func (s *Server) deterministicMkdirExec(ctx context.Context, corr, dirRel string, pushStage func(string, map[string]any)) (*gateway.Result, error) {
	dirRel = strings.TrimSpace(dirRel)
	if dirRel == "" || !pathAllowed(s.cfg.WorkspaceDir, dirRel) {
		pushStage("deterministic_mkdir_precheck_failed", map[string]any{"path": dirRel})
		return nil, fmt.Errorf("mkdir path invalid or outside workspace")
	}
	pushStage("deterministic_mkdir_dispatch", map[string]any{"path": dirRel})
	return s.gwChatExec(ctx, corr, "fs.mkdir", []string{dirRel}, nil)
}

func (s *Server) runDeterministicMkdirOnly(
	ctx context.Context,
	corr, dirRel string,
	pushStage func(string, map[string]any),
	gwActivity map[string]any,
	final *strings.Builder,
) bool {
	res, err := s.deterministicMkdirExec(ctx, corr, dirRel, pushStage)
	if err != nil {
		if final.Len() > 0 {
			final.WriteString("\n\n")
		}
		final.WriteString("FORGE (deterministic mkdir): " + err.Error())
		gwActivity["executionState"] = "error"
		gwActivity["failureReason"] = err.Error()
		gwActivity["toolSelected"] = "fs.mkdir"
		gwActivity["toolArgs"] = map[string]any{"path": dirRel}
		gwActivity["syntheticToolExecution"] = true
		return true
	}
	if res.Status != gateway.StatusOK {
		pushStage("deterministic_mkdir_denied", map[string]any{"status": res.Status, "reason": res.DeniedReason})
		if final.Len() > 0 {
			final.WriteString("\n\n")
		}
		final.WriteString(fmt.Sprintf("FORGE (deterministic mkdir): gateway %s — %s", res.Status, strings.TrimSpace(coalesceReason(res))))
		gwActivity["executionState"] = res.Status
		gwActivity["failureReason"] = coalesceReason(res)
		gwActivity["toolSelected"] = "fs.mkdir"
		gwActivity["toolArgs"] = map[string]any{"path": dirRel}
		gwActivity["syntheticToolExecution"] = true
		return true
	}
	pushStage("deterministic_mkdir_ok", map[string]any{"path": res.Data["path"]})
	if final.Len() > 0 {
		final.WriteString("\n\n")
	}
	final.WriteString("FORGE (deterministic mkdir): " + formatToolResult("fs.mkdir", res))
	gwActivity["toolSelected"] = "fs.mkdir"
	gwActivity["toolArgs"] = map[string]any{"path": dirRel}
	gwActivity["executionState"] = "ok"
	gwActivity["executionResult"] = res.Data
	gwActivity["syntheticToolExecution"] = true
	return true
}

func (s *Server) runDeterministicMkdirThenWrite(
	ctx context.Context,
	corr, user string,
	pushStage func(string, map[string]any),
	gwActivity map[string]any,
	final *strings.Builder,
) bool {
	dirRel, fname, content, ok := gateway.ParseCombinedMkdirAndWrite(user)
	if !ok {
		return false
	}
	// For combined mkdir+write intents, issue a single fs.write action.
	// fs.write already creates parent directories, so this avoids split approval
	// loops where mkdir is approved but write never runs.
	writePath := filepath.ToSlash(filepath.Join(strings.TrimSpace(dirRel), strings.TrimSpace(fname)))
	if !pathAllowed(s.cfg.WorkspaceDir, writePath) {
		pushStage("deterministic_write_precheck_failed", map[string]any{"path": writePath})
		if final.Len() > 0 {
			final.WriteString("\n\n")
		}
		final.WriteString("FORGE (deterministic write): refused path outside workspace.")
		gwActivity["toolSelected"] = "fs.write"
		gwActivity["toolArgs"] = map[string]any{"directory": dirRel, "file": fname, "writePath": writePath}
		gwActivity["executionState"] = "error"
		gwActivity["failureReason"] = "write path outside workspace after mkdir"
		gwActivity["executionResult"] = map[string]any{"writePath": writePath}
		gwActivity["syntheticToolExecution"] = true
		return true
	}
	pushStage("deterministic_write_dispatch", map[string]any{"path": writePath})
	resWr, err := s.gwChatExec(ctx, corr, "fs.write", []string{writePath}, map[string]any{"contents": content})
	if err != nil {
		pushStage("deterministic_write_error", map[string]any{"error": err.Error()})
		if final.Len() > 0 {
			final.WriteString("\n\n")
		}
		final.WriteString("FORGE (deterministic write): " + err.Error())
		gwActivity["toolSelected"] = "fs.write"
		gwActivity["toolArgs"] = map[string]any{"directory": dirRel, "file": fname, "writePath": writePath}
		gwActivity["executionState"] = "error"
		gwActivity["failureReason"] = err.Error()
		gwActivity["executionResult"] = map[string]any{"writePath": writePath}
		gwActivity["syntheticToolExecution"] = true
		return true
	}
	if resWr.Status != gateway.StatusOK {
		if final.Len() > 0 {
			final.WriteString("\n\n")
		}
		final.WriteString(fmt.Sprintf("FORGE (deterministic write): gateway %s — %s", resWr.Status, strings.TrimSpace(coalesceReason(resWr))))
		gwActivity["toolSelected"] = "fs.write"
		gwActivity["toolArgs"] = map[string]any{"directory": dirRel, "file": fname, "writePath": writePath}
		gwActivity["executionState"] = resWr.Status
		gwActivity["failureReason"] = coalesceReason(resWr)
		gwActivity["executionResult"] = map[string]any{"write": resWr.Data}
		gwActivity["syntheticToolExecution"] = true
		return true
	}
	pushStage("deterministic_write_ok", map[string]any{"path": resWr.Data["path"]})
	if final.Len() > 0 {
		final.WriteString("\n\n")
	}
	final.WriteString("FORGE (deterministic write): " + formatToolResult("fs.write", resWr))
	gwActivity["toolSelected"] = "fs.write"
	gwActivity["toolArgs"] = map[string]any{"directory": dirRel, "file": fname, "writePath": writePath}
	gwActivity["executionState"] = "ok"
	gwActivity["executionResult"] = map[string]any{"write": resWr.Data}
	gwActivity["syntheticToolExecution"] = true
	return true
}

func coalesceReason(res *gateway.Result) string {
	if res == nil {
		return ""
	}
	if strings.TrimSpace(res.DeniedReason) != "" {
		return res.DeniedReason
	}
	return res.Message
}
