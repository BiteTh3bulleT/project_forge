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
	if writes, ok := gateway.ParseSVGAssetWriteIntents(lastUserContent); ok {
		return s.runDeterministicSVGWrites(ctx, corr, writes, pushStage, gwActivity, final)
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
	if writePath, script, ok := gateway.ParseDownloadSorterScriptIntent(lastUserContent); ok {
		pushStage("deterministic_download_sorter_dispatch", map[string]any{"path": writePath})
		res, err := s.gwChatExec(ctx, corr, "fs.write", []string{writePath}, map[string]any{"contents": script})
		if err != nil {
			if final.Len() > 0 {
				final.WriteString("\n\n")
			}
			final.WriteString("FORGE (deterministic download sorter): " + err.Error())
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
			final.WriteString(fmt.Sprintf("FORGE (deterministic download sorter): gateway %s — %s", res.Status, strings.TrimSpace(coalesceReason(res))))
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
		final.WriteString("FORGE (deterministic download sorter): " + formatToolResult("fs.write", res))
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
	if forcedModel == gateway.ChatModelName("web.search") {
		query, ok := gateway.ParseWebSearchQuery(lastUserContent)
		if ok {
			pushStage("deterministic_web_search_dispatch", map[string]any{"query": query})
			res, err := s.gwChatExec(ctx, corr, "web.search", nil, map[string]any{"query": query, "limit": 5})
			if err != nil {
				if final.Len() > 0 {
					final.WriteString("\n\n")
				}
				final.WriteString("FORGE (deterministic web search): " + err.Error())
				gwActivity["executionState"] = "error"
				gwActivity["failureReason"] = err.Error()
				gwActivity["toolSelected"] = "web.search"
				gwActivity["toolArgs"] = map[string]any{"query": query}
				gwActivity["syntheticToolExecution"] = true
				return true
			}
			if res.Status != gateway.StatusOK {
				if final.Len() > 0 {
					final.WriteString("\n\n")
				}
				final.WriteString(fmt.Sprintf("FORGE (deterministic web search): gateway %s — %s", res.Status, strings.TrimSpace(coalesceReason(res))))
				gwActivity["executionState"] = res.Status
				gwActivity["failureReason"] = coalesceReason(res)
				gwActivity["toolSelected"] = "web.search"
				gwActivity["toolArgs"] = map[string]any{"query": query}
				gwActivity["executionResult"] = res.Data
				gwActivity["syntheticToolExecution"] = true
				return true
			}
			if final.Len() > 0 {
				final.WriteString("\n\n")
			}
			final.WriteString("FORGE (deterministic web search): " + formatToolResult("web.search", res))
			gwActivity["toolSelected"] = "web.search"
			gwActivity["toolArgs"] = map[string]any{"query": query}
			gwActivity["executionState"] = "ok"
			gwActivity["executionResult"] = res.Data
			gwActivity["syntheticToolExecution"] = true
			return true
		}
	}
	if forcedModel == gateway.ChatModelName("net.fetch") {
		rawURL, ok := gateway.ParseURLFromText(lastUserContent)
		if ok {
			pushStage("deterministic_url_fetch_dispatch", map[string]any{"url": rawURL})
			res, err := s.gwChatExec(ctx, corr, "net.fetch", nil, map[string]any{"url": rawURL})
			if err != nil {
				if final.Len() > 0 {
					final.WriteString("\n\n")
				}
				final.WriteString("FORGE (deterministic fetch): " + err.Error())
				gwActivity["executionState"] = "error"
				gwActivity["failureReason"] = err.Error()
				gwActivity["toolSelected"] = "net.fetch"
				gwActivity["toolArgs"] = map[string]any{"url": rawURL}
				gwActivity["syntheticToolExecution"] = true
				return true
			}
			if res.Status != gateway.StatusOK {
				if final.Len() > 0 {
					final.WriteString("\n\n")
				}
				final.WriteString(fmt.Sprintf("FORGE (deterministic fetch): gateway %s — %s", res.Status, strings.TrimSpace(coalesceReason(res))))
				gwActivity["executionState"] = res.Status
				gwActivity["failureReason"] = coalesceReason(res)
				gwActivity["toolSelected"] = "net.fetch"
				gwActivity["toolArgs"] = map[string]any{"url": rawURL}
				gwActivity["executionResult"] = res.Data
				gwActivity["syntheticToolExecution"] = true
				return true
			}
			if final.Len() > 0 {
				final.WriteString("\n\n")
			}
			final.WriteString("FORGE (deterministic fetch): " + formatToolResult("net.fetch", res))
			gwActivity["toolSelected"] = "net.fetch"
			gwActivity["toolArgs"] = map[string]any{"url": rawURL}
			gwActivity["executionState"] = "ok"
			gwActivity["executionResult"] = res.Data
			gwActivity["syntheticToolExecution"] = true
			return true
		}
	}
	if forcedModel == gateway.ChatModelName("desktop.open") {
		rawURL, ok := gateway.ParseURLFromText(lastUserContent)
		if ok {
			pushStage("deterministic_browser_open_dispatch", map[string]any{"url": rawURL})
			res, err := s.gwChatExec(ctx, corr, "desktop.open", nil, map[string]any{"url": rawURL})
			if err != nil {
				if final.Len() > 0 {
					final.WriteString("\n\n")
				}
				final.WriteString("FORGE (deterministic browser): " + err.Error())
				gwActivity["executionState"] = "error"
				gwActivity["failureReason"] = err.Error()
				gwActivity["toolSelected"] = "desktop.open"
				gwActivity["toolArgs"] = map[string]any{"url": rawURL}
				gwActivity["syntheticToolExecution"] = true
				return true
			}
			if final.Len() > 0 {
				final.WriteString("\n\n")
			}
			final.WriteString("FORGE (deterministic browser): " + formatToolResult("desktop.open", res))
			gwActivity["toolSelected"] = "desktop.open"
			gwActivity["toolArgs"] = map[string]any{"url": rawURL}
			gwActivity["executionState"] = res.Status
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

func (s *Server) runDeterministicSVGWrites(
	ctx context.Context,
	corr string,
	writes []gateway.DeterministicWrite,
	pushStage func(string, map[string]any),
	gwActivity map[string]any,
	final *strings.Builder,
) bool {
	if len(writes) == 0 {
		return false
	}
	paths := make([]string, 0, len(writes))
	files := make([]map[string]any, 0, len(writes))
	for _, write := range writes {
		writePath := strings.TrimSpace(write.Path)
		if writePath == "" || !pathAllowed(s.cfg.WorkspaceDir, writePath) {
			pushStage("deterministic_svg_precheck_failed", map[string]any{"path": writePath})
			if final.Len() > 0 {
				final.WriteString("\n\n")
			}
			final.WriteString("FORGE (deterministic svg): refused path outside workspace.")
			gwActivity["executionState"] = "denied"
			gwActivity["failureReason"] = "path rejected (outside workspace or traversal)"
			gwActivity["toolSelected"] = "fs.write"
			gwActivity["toolArgs"] = map[string]any{"paths": paths}
			gwActivity["syntheticToolExecution"] = true
			return true
		}
		paths = append(paths, writePath)
		files = append(files, map[string]any{
			"contents": write.Contents,
		})
	}

	pushStage("deterministic_svg_dispatch", map[string]any{"paths": paths, "count": len(paths)})
	res, err := s.gwChatExec(ctx, corr, "fs.write", paths, map[string]any{"files": files})
	if err != nil {
		pushStage("deterministic_svg_error", map[string]any{"error": err.Error()})
		if final.Len() > 0 {
			final.WriteString("\n\n")
		}
		final.WriteString("FORGE (deterministic svg): " + err.Error())
		gwActivity["executionState"] = "error"
		gwActivity["failureReason"] = err.Error()
		gwActivity["toolSelected"] = "fs.write"
		gwActivity["toolArgs"] = map[string]any{"paths": paths}
		gwActivity["syntheticToolExecution"] = true
		return true
	}
	if res.Status != gateway.StatusOK {
		if final.Len() > 0 {
			final.WriteString("\n\n")
		}
		final.WriteString(fmt.Sprintf("FORGE (deterministic svg): gateway %s — %s", res.Status, strings.TrimSpace(coalesceReason(res))))
		gwActivity["executionState"] = res.Status
		gwActivity["failureReason"] = coalesceReason(res)
		gwActivity["toolSelected"] = "fs.write"
		gwActivity["toolArgs"] = map[string]any{"paths": paths}
		gwActivity["executionResult"] = res.Data
		gwActivity["syntheticToolExecution"] = true
		return true
	}

	pushStage("deterministic_svg_ok", map[string]any{"paths": res.Data["paths"], "count": res.Data["count"]})
	if final.Len() > 0 {
		final.WriteString("\n\n")
	}
	final.WriteString("FORGE (deterministic svg): " + formatToolResult("fs.write", res))
	gwActivity["executionState"] = "ok"
	gwActivity["toolSelected"] = "fs.write"
	gwActivity["toolArgs"] = map[string]any{"paths": paths}
	gwActivity["executionResult"] = res.Data
	gwActivity["syntheticToolExecution"] = true
	return true
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

func (s *Server) runDeterministicWrite(
	ctx context.Context,
	corr string,
	label string,
	writePath string,
	contents string,
	pushStage func(string, map[string]any),
	gwActivity map[string]any,
	final *strings.Builder,
) bool {
	stageLabel := strings.TrimSpace(label)
	if stageLabel == "" {
		stageLabel = "write"
	}
	if !pathAllowed(s.cfg.WorkspaceDir, writePath) {
		pushStage("deterministic_"+stageLabel+"_precheck_failed", map[string]any{"path": writePath})
		if final.Len() > 0 {
			final.WriteString("\n\n")
		}
		final.WriteString("FORGE (deterministic " + stageLabel + "): refused path outside workspace.")
		gwActivity["executionState"] = "denied"
		gwActivity["failureReason"] = "path rejected (outside workspace or traversal)"
		gwActivity["toolSelected"] = "fs.write"
		gwActivity["toolArgs"] = map[string]any{"path": writePath}
		gwActivity["syntheticToolExecution"] = true
		return true
	}

	pushStage("deterministic_"+stageLabel+"_dispatch", map[string]any{"path": writePath})
	res, err := s.gwChatExec(ctx, corr, "fs.write", []string{writePath}, map[string]any{"contents": contents})
	if err != nil {
		pushStage("deterministic_"+stageLabel+"_error", map[string]any{"error": err.Error()})
		if final.Len() > 0 {
			final.WriteString("\n\n")
		}
		final.WriteString("FORGE (deterministic " + stageLabel + "): " + err.Error())
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
		final.WriteString(fmt.Sprintf("FORGE (deterministic %s): gateway %s — %s", stageLabel, res.Status, strings.TrimSpace(coalesceReason(res))))
		gwActivity["executionState"] = res.Status
		gwActivity["failureReason"] = coalesceReason(res)
		gwActivity["toolSelected"] = "fs.write"
		gwActivity["toolArgs"] = map[string]any{"path": writePath}
		gwActivity["executionResult"] = res.Data
		gwActivity["syntheticToolExecution"] = true
		return true
	}

	pushStage("deterministic_"+stageLabel+"_ok", map[string]any{"path": res.Data["path"]})
	if final.Len() > 0 {
		final.WriteString("\n\n")
	}
	final.WriteString("FORGE (deterministic " + stageLabel + "): " + formatToolResult("fs.write", res))
	gwActivity["executionState"] = "ok"
	gwActivity["toolSelected"] = "fs.write"
	gwActivity["toolArgs"] = map[string]any{"path": writePath}
	gwActivity["executionResult"] = res.Data
	gwActivity["syntheticToolExecution"] = true
	return true
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
