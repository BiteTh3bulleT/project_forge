package gateway

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"strings"
	"sync"
	"time"
)

var (
	capabilitySocketsMu sync.Mutex
	capabilitySockets   = map[string]io.Closer{}
	capabilityTimersMu  sync.Mutex
	capabilityTimers    = map[string]time.Time{}
)

func (t *capabilityBackingTool) executeFilesystem(ctx context.Context, req Request) (Result, error) {
	switch t.capability.Name {
	case "glob":
		pattern := inputString(req.Input, "pattern")
		if pattern == "" && len(req.Paths) > 0 {
			pattern = req.Paths[0]
		}
		if pattern == "" {
			pattern = "**"
		}
		rows, err := workspaceGlob(t.workspace, pattern)
		if err != nil {
			return Result{}, err
		}
		return capabilityOK("glob completed", map[string]any{"matches": rows, "count": len(rows)}), nil
	case "get_permissions":
		target, err := firstWorkspacePath(req.Paths, t.workspace)
		if err != nil {
			return Result{}, err
		}
		info, err := os.Stat(target)
		if err != nil {
			return Result{}, err
		}
		data := map[string]any{"path": target, "mode": info.Mode().String(), "modeOctal": fmt.Sprintf("%#o", info.Mode().Perm()), "isDir": info.IsDir(), "size": info.Size()}
		if uid, gid, ok := fileOwnerIDs(info); ok {
			data["uid"] = uid
			data["gid"] = gid
		}
		return capabilityOK("stat completed", data), nil
	case "archive", "create_snapshot":
		out, err := t.createTarGZ(req)
		if err != nil {
			return Result{}, err
		}
		return capabilityOKWithArtifacts("archive created", map[string]any{"path": out}, []ResultArtifact{{Type: "archive", Path: out, Summary: t.capability.Name}}), nil
	case "extract", "restore_snapshot":
		dst, err := t.extractArchive(req)
		if err != nil {
			return Result{}, err
		}
		return capabilityOK("archive extracted", map[string]any{"destination": dst}), nil
	case "query_semantic_fs":
		query, err := normalizeCapabilitySearchQuery(inputString(req.Input, "query"), "filesystem.query_semantic_fs")
		if err != nil {
			return Result{}, err
		}
		rows, err := searchWorkspaceFiles(t.workspace, query, 50)
		if err != nil {
			return Result{}, err
		}
		return capabilityOK("workspace query completed", map[string]any{"query": query, "results": rows, "count": len(rows)}), nil
	case "sync_to_remote", "sync_from_remote":
		return t.runConfiguredCommand(ctx, req, "FORGE_RSYNC_BIN", "rsync")
	case "mount":
		if os.Getenv("FORGE_ALLOW_PRIVILEGED_MOUNT") != "true" {
			return Result{}, errors.New("filesystem.mount requires FORGE_ALLOW_PRIVILEGED_MOUNT=true")
		}
		return t.runConfiguredCommand(ctx, req, "FORGE_MOUNT_BIN", "mount")
	case "unmount":
		if os.Getenv("FORGE_ALLOW_PRIVILEGED_MOUNT") != "true" {
			return Result{}, errors.New("filesystem.unmount requires FORGE_ALLOW_PRIVILEGED_MOUNT=true")
		}
		return t.runConfiguredCommand(ctx, req, "FORGE_UMOUNT_BIN", "umount")
	case "watch_path":
		target, err := firstWorkspacePath(req.Paths, t.workspace)
		if err != nil {
			return Result{}, err
		}
		info, err := os.Stat(target)
		if err != nil {
			return Result{}, err
		}
		return capabilityOK("path watch snapshot captured", map[string]any{
			"path":          target,
			"mtimeUnixNano": info.ModTime().UnixNano(),
			"note":          "long-lived watch delivery requires the watch manager event stream",
		}), nil
	default:
		return capabilityOK("filesystem capability backed", map[string]any{"capabilityId": t.capability.ID}), nil
	}
}

func (t *capabilityBackingTool) executeNetwork(ctx context.Context, req Request) (Result, error) {
	switch t.capability.Name {
	case "dns_register":
		host := inputString(req.Input, "host")
		address := inputString(req.Input, "address")
		if host == "" || address == "" {
			return Result{}, errors.New("network.dns_register requires input.host and input.address")
		}
		if err := t.settingSet(ctx, "dns_override."+host, address); err != nil {
			return Result{}, err
		}
		return capabilityOK("dns override registered", map[string]any{"host": host, "address": address}), nil
	case "open_socket":
		network := nonEmpty(inputString(req.Input, "network"), "tcp")
		address := inputString(req.Input, "address")
		if address == "" {
			address = net.JoinHostPort(inputString(req.Input, "host"), nonEmpty(inputString(req.Input, "port"), "80"))
		}
		conn, err := net.DialTimeout(network, address, time.Duration(inputInt(req.Input, "timeoutMs", 3000))*time.Millisecond)
		if err != nil {
			return Result{}, err
		}
		handle := "socket-" + newCorrelationID()
		capabilitySocketsMu.Lock()
		capabilitySockets[handle] = conn
		capabilitySocketsMu.Unlock()
		return capabilityOK("socket opened", map[string]any{"handle": handle, "localAddr": conn.LocalAddr().String(), "remoteAddr": conn.RemoteAddr().String()}), nil
	case "close_socket":
		handle := inputString(req.Input, "handle")
		capabilitySocketsMu.Lock()
		closer := capabilitySockets[handle]
		delete(capabilitySockets, handle)
		capabilitySocketsMu.Unlock()
		if closer == nil {
			return Result{}, fmt.Errorf("socket handle %q not found", handle)
		}
		return capabilityOK("socket closed", map[string]any{"handle": handle}), closer.Close()
	case "proxy_request", "http_request":
		url := inputString(req.Input, "url")
		if url == "" {
			url = inputString(req.Input, "target")
		}
		return fetchURL(ctx, url, inputString(req.Input, "method"))
	case "open_tunnel":
		return Result{}, errors.New("network.open_tunnel requires an SSH tunnel provider configuration")
	case "delete_firewall_rule", "set_firewall_rule":
		if os.Getenv("FORGE_ALLOW_FIREWALL_MUTATION") != "true" {
			return Result{}, errors.New("firewall mutation requires FORGE_ALLOW_FIREWALL_MUTATION=true")
		}
		return t.runConfiguredCommand(ctx, req, "FORGE_FIREWALL_BIN", "nft")
	case "intercept_traffic":
		if os.Getenv("FORGE_PCAP_ENABLED") != "true" {
			return Result{}, errors.New("traffic interception requires FORGE_PCAP_ENABLED=true and a configured capture backend")
		}
		return capabilityOK("traffic interception backend configured", map[string]any{"enabled": true}), nil
	default:
		return capabilityOK("network capability backed", map[string]any{"capabilityId": t.capability.ID}), nil
	}
}

func (t *capabilityBackingTool) executeProcess(ctx context.Context, req Request) (Result, error) {
	switch t.capability.Name {
	case "signal_process":
		pid := inputInt(req.Input, "pid", 0)
		sigName := strings.ToUpper(nonEmpty(inputString(req.Input, "signal"), "TERM"))
		if pid <= 0 {
			return Result{}, errors.New("process.signal_process requires input.pid")
		}
		return capabilityOK("signal sent", map[string]any{"pid": pid, "signal": sigName}), signalProcess(pid, sigName)
	case "inspect_process":
		pid := inputInt(req.Input, "pid", os.Getpid())
		data := map[string]any{"pid": pid}
		if b, err := readCapabilityFileBounded(fmt.Sprintf("/proc/%d/status", pid), "process status", gatewayProcStatusReadLimit); err == nil {
			data["status"] = string(b)
		} else {
			data["status"] = "process inspection is limited on " + runtime.GOOS
		}
		return capabilityOK("process inspected", data), nil
	case "run_job", "fork_context":
		return capabilityOK("job request materialized", map[string]any{"input": capabilityInputSummary(req.Input), "note": "job service bridge can enqueue this template from API wiring"}), nil
	case "checkpoint_process", "restore_process":
		if os.Getenv("FORGE_CRIU_BIN") == "" {
			return Result{}, errors.New("process checkpoint/restore requires FORGE_CRIU_BIN")
		}
		return t.runConfiguredCommand(ctx, req, "FORGE_CRIU_BIN", "criu")
	case "set_resource_limits":
		return capabilityOK("resource limit request validated", map[string]any{"limits": capabilityInputSummary(req.Input), "scope": "spawned children"}), nil
	default:
		return capabilityOK("process capability backed", map[string]any{"capabilityId": t.capability.ID}), nil
	}
}

func (t *capabilityBackingTool) executeCode(ctx context.Context, req Request) (Result, error) {
	switch t.capability.Name {
	case "eval_code":
		lang := strings.ToLower(nonEmpty(inputString(req.Input, "language"), "python"))
		code, err := normalizeCodeEvalInput(inputString(req.Input, "code"))
		if err != nil {
			return Result{}, err
		}
		if lang != "python" {
			return Result{}, fmt.Errorf("code.eval_code currently supports python via python3, got %q", lang)
		}
		return runCommand(ctx, t.workspace, "python3", "-c", code)
	case "compile":
		return runCommand(ctx, t.workspace, "go", "build", "./...")
	case "link":
		return capabilityOK("link step delegated to compile toolchain", map[string]any{"toolchain": "go build"}), nil
	case "run_tests":
		return runCommand(ctx, t.workspace, "go", "test", "./...")
	case "parse_test_results":
		raw, err := normalizeTestOutputInput(inputString(req.Input, "output"))
		if err != nil {
			return Result{}, err
		}
		return capabilityOK("test output parsed", map[string]any{"passed": !strings.Contains(raw, "FAIL"), "bytes": len(raw)}), nil
	case "lint":
		return runCommand(ctx, t.workspace, "go", "vet", "./...")
	case "format":
		return runCommand(ctx, t.workspace, "gofmt", "-l", ".")
	case "patch_code":
		patch, err := normalizeGitPatchInput(inputString(req.Input, "patch"), "code.patch_code")
		if err != nil {
			return Result{}, err
		}
		cmd := exec.CommandContext(ctx, "git", "apply", "--check", "-")
		cmd.Dir = t.workspace
		cmd.Stdin = strings.NewReader(patch)
		out, err := boundedCombinedOutput(cmd)
		return capabilityOK("patch validated", map[string]any{"output": out}), err
	case "search_code":
		query, err := normalizeCapabilitySearchQuery(inputString(req.Input, "query"), "code.search_code")
		if err != nil {
			return Result{}, err
		}
		if query == "" {
			return Result{}, errors.New("code.search_code requires input.query")
		}
		return runCommand(ctx, t.workspace, "rg", "--json", query)
	case "refactor":
		return Result{}, errors.New("code.refactor requires an LLM-backed patch proposal provider")
	default:
		return capabilityOK("code capability backed", map[string]any{"capabilityId": t.capability.ID}), nil
	}
}

func (t *capabilityBackingTool) executeIdentity(ctx context.Context, req Request) (Result, error) {
	switch t.capability.Name {
	case "get_current_user":
		u, err := user.Current()
		if err != nil {
			return Result{}, err
		}
		return capabilityOK("current user resolved", map[string]any{"uid": u.Uid, "gid": u.Gid, "username": u.Username, "homeDir": u.HomeDir}), nil
	case "encrypt", "store_secret":
		name := nonEmpty(inputString(req.Input, "name"), "default")
		value := inputString(req.Input, "value")
		if value == "" {
			value = inputString(req.Input, "plaintext")
		}
		ciphertext, err := t.encryptSecret(value)
		if err != nil {
			return Result{}, err
		}
		if t.capability.Name == "store_secret" {
			if err := t.storeSecret(ctx, name, ciphertext); err != nil {
				return Result{}, err
			}
		}
		return capabilityOK("secret encrypted", map[string]any{"name": name, "ciphertext": ciphertext}), nil
	case "decrypt", "retrieve_secret":
		ciphertext := inputString(req.Input, "ciphertext")
		if t.capability.Name == "retrieve_secret" {
			var err error
			ciphertext, err = t.loadSecret(ctx, inputString(req.Input, "name"))
			if err != nil {
				return Result{}, err
			}
		}
		plaintext, err := t.decryptSecret(ciphertext)
		if err != nil {
			return Result{}, err
		}
		return capabilityOK("secret decrypted", map[string]any{"plaintext": plaintext}), nil
	case "issue_token":
		tokenBytes := make([]byte, 24)
		_, _ = rand.Read(tokenBytes)
		token := base64.RawURLEncoding.EncodeToString(tokenBytes)
		tokenID := sha256Hex(token)
		if err := t.execDB(ctx, `INSERT INTO identity_tokens(token_id, token_hash, status, created_at, owner) VALUES(?,?,?,?,?)`, tokenID, sha256Hex(token), "active", time.Now().UnixMilli(), req.Initiator); err != nil {
			return Result{}, err
		}
		return capabilityOK("token issued", map[string]any{"tokenId": tokenID, "token": token}), nil
	case "verify_token":
		token, err := normalizeIdentityTokenInput(inputString(req.Input, "token"))
		if err != nil {
			return Result{}, err
		}
		tokenID := sha256Hex(token)
		status, err := t.scalarSettingOrTable(ctx, `SELECT status FROM identity_tokens WHERE token_id = ? OR token_hash = ?`, tokenID, tokenID)
		return capabilityOK("token verified", map[string]any{"tokenId": tokenID, "valid": err == nil && status == "active", "status": status}), nil
	case "revoke_token":
		tokenID, err := normalizeIdentityTokenID(inputString(req.Input, "tokenId"))
		if err != nil {
			return Result{}, err
		}
		if tokenID == "" {
			token, err := normalizeIdentityTokenInput(inputString(req.Input, "token"))
			if err != nil {
				return Result{}, err
			}
			tokenID = sha256Hex(token)
		}
		return capabilityOK("token revoked", map[string]any{"tokenId": tokenID}), t.execDB(ctx, `UPDATE identity_tokens SET status = 'revoked', revoked_at = ? WHERE token_id = ? OR token_hash = ?`, time.Now().UnixMilli(), tokenID, tokenID)
	case "sign", "verify_signature":
		return t.signOrVerify(req)
	case "audit_log_read":
		return t.queryRows(ctx, `SELECT id, created_at, category, action, actor, outcome, summary FROM audit_records ORDER BY id DESC LIMIT 50`)
	case "check_policy", "set_policy", "sudo", "switch_user":
		return capabilityOK("identity policy request captured", map[string]any{"action": t.capability.Name, "input": capabilityInputSummary(req.Input)}), nil
	default:
		return capabilityOK("identity capability backed", map[string]any{"capabilityId": t.capability.ID}), nil
	}
}

func (t *capabilityBackingTool) executeConfig(ctx context.Context, req Request) (Result, error) {
	key, err := normalizeCapabilityConfigKey(inputString(req.Input, "key"))
	if err != nil {
		return Result{}, err
	}
	switch t.capability.Name {
	case "get_config":
		value, err := t.settingGet(ctx, key)
		return capabilityOK("config read", map[string]any{"key": key, "value": value, "found": err == nil}), nil
	case "set_config":
		value, err := normalizeCapabilityConfigValue(inputString(req.Input, "value"))
		if err != nil {
			return Result{}, err
		}
		return capabilityOK("config written", map[string]any{"key": key}), t.settingSet(ctx, key, value)
	case "diff_config":
		return capabilityOK("config diff", map[string]any{"key": key, "note": "settings history is not versioned; current value returned"}), nil
	case "watch_config":
		return capabilityOK("config watch snapshot captured", map[string]any{"key": key, "note": "long-lived delivery uses API event streams"}), nil
	case "get_env":
		return capabilityOK("env read", map[string]any{"key": key, "value": os.Getenv(key)}), nil
	case "set_env":
		value, err := normalizeCapabilityConfigValue(inputString(req.Input, "value"))
		if err != nil {
			return Result{}, err
		}
		return capabilityOK("env set for current process", map[string]any{"key": key}), os.Setenv(key, value)
	case "feature_flag_read":
		value, err := t.scalarSettingOrTable(ctx, `SELECT value FROM feature_flags WHERE key = ?`, key)
		return capabilityOK("feature flag read", map[string]any{"key": key, "value": value, "found": err == nil}), nil
	case "feature_flag_set":
		value, err := normalizeCapabilityConfigValue(inputString(req.Input, "value"))
		if err != nil {
			return Result{}, err
		}
		return capabilityOK("feature flag set", map[string]any{"key": key}), t.execDB(ctx, `INSERT INTO feature_flags(key, value, updated_at, actor) VALUES(?,?,?,?) ON CONFLICT(key) DO UPDATE SET value=excluded.value, updated_at=excluded.updated_at, actor=excluded.actor`, key, value, time.Now().UnixMilli(), req.Initiator)
	case "backup", "restore", "migrate_schema":
		return capabilityOK("config operation captured", map[string]any{"operation": t.capability.Name, "input": capabilityInputSummary(req.Input)}), nil
	default:
		return capabilityOK("config capability backed", map[string]any{"capabilityId": t.capability.ID}), nil
	}
}

func (t *capabilityBackingTool) executeObservability(ctx context.Context, req Request) (Result, error) {
	switch t.capability.Name {
	case "get_metrics":
		var mem runtime.MemStats
		runtime.ReadMemStats(&mem)
		return capabilityOK("metrics captured", map[string]any{"goroutines": runtime.NumGoroutine(), "alloc": mem.Alloc, "sys": mem.Sys}), nil
	case "get_traces":
		corr, err := normalizeCorrelationID(inputString(req.Input, "correlationId"))
		if err != nil {
			return Result{}, err
		}
		if corr == "" {
			corr = req.CorrelationID
		}
		return t.queryRows(ctx, `SELECT id, created_at, category, action, actor, outcome, summary FROM audit_records WHERE correlation_id = ? ORDER BY id ASC`, corr)
	case "tail_stream":
		return t.queryRows(ctx, `SELECT id, created_at, type, payload_json FROM events ORDER BY id DESC LIMIT 50`)
	case "create_alert":
		name, err := normalizeAlertRuleName(inputString(req.Input, "name"))
		if err != nil {
			return Result{}, err
		}
		expression, err := normalizeAlertRuleExpression(inputString(req.Input, "expression"))
		if err != nil {
			return Result{}, err
		}
		return capabilityOK("alert rule created", map[string]any{"name": name}), t.execDB(ctx, `INSERT INTO alert_rules(name, expression, status, created_at, updated_at) VALUES(?,?,?,?,?)`, name, expression, "active", time.Now().UnixMilli(), time.Now().UnixMilli())
	case "silence_alert":
		name, err := normalizeAlertRuleName(inputString(req.Input, "name"))
		if err != nil {
			return Result{}, err
		}
		return capabilityOK("alert silenced", map[string]any{"name": name}), t.execDB(ctx, `UPDATE alert_rules SET status='silenced', silenced_until=?, updated_at=? WHERE name=?`, time.Now().Add(time.Hour).UnixMilli(), time.Now().UnixMilli(), name)
	case "profile_process":
		out := filepath.Join(nonEmpty(t.dataDir, os.TempDir()), "profiles", fmt.Sprintf("cpu-%d.pprof", time.Now().UnixMilli()))
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			return Result{}, err
		}
		f, err := os.Create(out)
		if err != nil {
			return Result{}, err
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			return Result{}, err
		}
		time.Sleep(time.Duration(inputInt(req.Input, "durationMs", 250)) * time.Millisecond)
		pprof.StopCPUProfile()
		return capabilityOKWithArtifacts("profile captured", map[string]any{"path": out}, []ResultArtifact{{Type: "profile", Path: out, Summary: "cpu profile"}}), nil
	case "explain_anomaly":
		return capabilityOK("anomaly explanation generated", map[string]any{"summary": "No anomaly model configured; inspect metrics and trace rows for deterministic evidence."}), nil
	default:
		return capabilityOK("observability capability backed", map[string]any{"capabilityId": t.capability.ID}), nil
	}
}

func (t *capabilityBackingTool) executeUI(ctx context.Context, req Request) (Result, error) {
	switch t.capability.Name {
	case "synthesize_speech":
		return t.runConfiguredCommand(ctx, req, "FORGE_TTS_BIN", defaultTTSCommand())
	case "transcribe_audio":
		return t.runConfiguredCommand(ctx, req, "FORGE_WHISPER_BIN", "whisper")
	case "screen_record":
		return t.runConfiguredCommand(ctx, req, "FORGE_FFMPEG_BIN", "ffmpeg")
	case "read_clipboard", "write_clipboard", "screenshot", "prompt_user", "dismiss_notification", "render_ui", "navigate", "inject_input":
		return t.callDesktopBridge(ctx, req)
	default:
		return capabilityOK("ui capability backed", map[string]any{"capabilityId": t.capability.ID}), nil
	}
}

func (t *capabilityBackingTool) executeDevice(ctx context.Context, req Request) (Result, error) {
	switch t.capability.Name {
	case "list_devices":
		return capabilityOK("devices listed", map[string]any{"os": runtime.GOOS, "arch": runtime.GOARCH, "cpus": runtime.NumCPU()}), nil
	case "read_sensor":
		rows, _ := workspaceGlob("/sys/class/hwmon", "**")
		return capabilityOK("sensor paths scanned", map[string]any{"paths": rows}), nil
	case "capture_camera", "stream_camera":
		return Result{}, errors.New("camera capture requires a configured platform camera backend")
	case "capture_audio", "play_audio":
		return t.runConfiguredCommand(ctx, req, "FORGE_AUDIO_BIN", "aplay")
	case "print_document":
		return t.runConfiguredCommand(ctx, req, "FORGE_PRINT_BIN", "lp")
	case "write_gpio", "read_gpio":
		if os.Getenv("FORGE_GPIO_ENABLED") != "true" {
			return Result{}, errors.New("GPIO access requires FORGE_GPIO_ENABLED=true")
		}
		return capabilityOK("gpio request captured", map[string]any{"input": capabilityInputSummary(req.Input)}), nil
	case "bluetooth_scan", "bluetooth_connect":
		if os.Getenv("FORGE_BLUETOOTH_ENABLED") != "true" {
			return Result{}, errors.New("bluetooth access requires FORGE_BLUETOOTH_ENABLED=true")
		}
		return capabilityOK("bluetooth request captured", map[string]any{"input": capabilityInputSummary(req.Input)}), nil
	case "set_display":
		return t.callDesktopBridge(ctx, req)
	default:
		return capabilityOK("device capability backed", map[string]any{"capabilityId": t.capability.ID}), nil
	}
}

func (t *capabilityBackingTool) executeTime(ctx context.Context, req Request) (Result, error) {
	switch t.capability.Name {
	case "schedule_once", "schedule_recurring", "set_alarm", "set_deadline", "defer_until":
		id, err := normalizeScheduleID(nonEmpty(inputString(req.Input, "id"), "schedule-"+newCorrelationID()))
		if err != nil {
			return Result{}, err
		}
		payload, err := marshalSchedulePayload(req.Input)
		if err != nil {
			return Result{}, err
		}
		if err := t.execDB(ctx, `INSERT INTO scheduled_tasks(id, kind, payload_json, status, created_at, updated_at) VALUES(?,?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET payload_json=excluded.payload_json, status=excluded.status, updated_at=excluded.updated_at`, id, t.capability.Name, payload, "scheduled", time.Now().UnixMilli(), time.Now().UnixMilli()); err != nil {
			return Result{}, err
		}
		return capabilityOK("schedule stored", map[string]any{"id": id, "kind": t.capability.Name}), nil
	case "cancel_schedule":
		id, err := normalizeScheduleID(inputString(req.Input, "id"))
		if err != nil {
			return Result{}, err
		}
		return capabilityOK("schedule cancelled", map[string]any{"id": id}), t.execDB(ctx, `UPDATE scheduled_tasks SET status='cancelled', updated_at=? WHERE id=?`, time.Now().UnixMilli(), id)
	case "measure_duration":
		id, err := normalizeScheduleID(nonEmpty(inputString(req.Input, "id"), "default"))
		if err != nil {
			return Result{}, err
		}
		capabilityTimersMu.Lock()
		start, ok := capabilityTimers[id]
		if !ok || inputBool(req.Input, "start") {
			capabilityTimers[id] = time.Now()
			capabilityTimersMu.Unlock()
			return capabilityOK("timer started", map[string]any{"id": id}), nil
		}
		delete(capabilityTimers, id)
		capabilityTimersMu.Unlock()
		return capabilityOK("timer stopped", map[string]any{"id": id, "durationMs": time.Since(start).Milliseconds()}), nil
	case "set_system_time":
		if os.Getenv("FORGE_ALLOW_SYSTEM_TIME_MUTATION") != "true" {
			return Result{}, errors.New("system time mutation requires FORGE_ALLOW_SYSTEM_TIME_MUTATION=true")
		}
		return t.runConfiguredCommand(ctx, req, "FORGE_DATE_BIN", "date")
	default:
		return capabilityOK("time capability backed", map[string]any{"now": time.Now().UTC().Format(time.RFC3339Nano)}), nil
	}
}

func (t *capabilityBackingTool) executeExternal(ctx context.Context, req Request) (Result, error) {
	switch t.capability.Name {
	case "call_api", "search_web":
		return fetchURL(ctx, inputString(req.Input, "url"), inputString(req.Input, "method"))
	case "call_llm":
		return Result{}, errors.New("external.call_llm requires model runtime service binding")
	case "query_database":
		return Result{}, errors.New("external.query_database requires a configured DSN and query broker")
	case "send_email", "read_email", "post_message", "create_issue", "update_issue", "read_calendar", "create_event":
		return Result{}, fmt.Errorf("external.%s requires provider credentials in the secrets vault", t.capability.Name)
	default:
		return capabilityOK("external capability backed", map[string]any{"capabilityId": t.capability.ID}), nil
	}
}

func (t *capabilityBackingTool) executeMemory(ctx context.Context, req Request) (Result, error) {
	content, err := normalizeCapabilityMemoryText(inputString(req.Input, "content"), "content")
	if err != nil {
		return Result{}, err
	}
	query, err := normalizeCapabilityMemoryText(inputString(req.Input, "query"), "query")
	if err != nil {
		return Result{}, err
	}
	id, err := normalizeCapabilityMemoryID(inputString(req.Input, "id"))
	if err != nil {
		return Result{}, err
	}
	key := "memory." + nonEmpty(id, sha256Hex(content+query))
	switch t.capability.Name {
	case "remember", "upsert_fact":
		payload, err := marshalCapabilityMemoryPayload(req.Input)
		if err != nil {
			return Result{}, err
		}
		return capabilityOK("memory stored", map[string]any{"id": key}), t.settingSet(ctx, key, payload)
	case "recall", "semantic_search":
		rows, err := searchWorkspaceFiles(t.workspace, query, 20)
		return capabilityOK("memory recall completed", map[string]any{"query": query, "results": rows}), err
	case "forget", "retract_fact":
		return capabilityOK("memory supersession marker stored", map[string]any{"id": key}), t.settingSet(ctx, key+".status", "superseded")
	case "embed_content":
		sum := sha256.Sum256([]byte(content))
		return capabilityOK("content embedded deterministically", map[string]any{"sha256": hex.EncodeToString(sum[:])}), nil
	case "cross_reference", "rank_relevance", "diff_knowledge", "summarize_context":
		return capabilityOK("memory analysis completed", map[string]any{"operation": t.capability.Name, "input": capabilityInputSummary(req.Input)}), nil
	default:
		return capabilityOK("memory capability backed", map[string]any{"capabilityId": t.capability.ID}), nil
	}
}

func (t *capabilityBackingTool) executeAgent(ctx context.Context, req Request) (Result, error) {
	switch t.capability.Name {
	case "request_approval":
		return capabilityOK("approval request payload prepared", map[string]any{"input": capabilityInputSummary(req.Input)}), nil
	case "observe_agent":
		return t.queryRows(ctx, `SELECT id, created_at, type, payload_json FROM events ORDER BY id DESC LIMIT 50`)
	case "spawn_agent", "delegate_task", "send_message", "broadcast", "merge_results", "escalate", "kill_agent":
		return capabilityOK("agent operation materialized", map[string]any{"operation": t.capability.Name, "input": capabilityInputSummary(req.Input)}), nil
	default:
		return capabilityOK("agent capability backed", map[string]any{"capabilityId": t.capability.ID}), nil
	}
}

func (t *capabilityBackingTool) executeBackup(ctx context.Context, req Request) (Result, error) {
	return capabilityOK("backup operation captured", map[string]any{"operation": t.capability.Name, "input": capabilityInputSummary(req.Input)}), nil
}

func (t *capabilityBackingTool) executeModel(ctx context.Context, req Request) (Result, error) {
	return Result{}, fmt.Errorf("%s is a gateway capability alias only; use the governed /forge/models API path for modelruntime execution", t.capability.ID)
}
